package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const rom2ScriptFormVersion byte = 106
const rom2ScenarioWireLen = 4 * 1024
const rom2GroupRecordLen = 9
const maxROM2Groups = 65535

func (w *World) rom2ScriptStateFault() error {
	dialect := w.script.Dialect()
	if dialect != ScriptROM1 && dialect != ScriptROM2 {
		return fmt.Errorf("sim: invalid script dialect %d", dialect)
	}
	if (w.rom2 != nil) != (dialect == ScriptROM2) {
		return fmt.Errorf("sim: ROM2 script dialect and state disagree")
	}
	if w.rom2 == nil {
		return nil
	}
	if len(w.rom2.Groups) > maxROM2Groups {
		return fmt.Errorf("sim: ROM2 group activity count exceeds %d", maxROM2Groups)
	}
	for i, row := range w.rom2.Groups {
		if i > 0 {
			prior := w.rom2.Groups[i-1]
			if row.Owner < prior.Owner || row.Owner == prior.Owner && row.Group <= prior.Group {
				return fmt.Errorf("sim: ROM2 group activity is not in unique owner/group order")
			}
		}
	}
	return nil
}

func (w *World) appendROM2ScriptState(b []byte) []byte {
	if w.rom2 == nil {
		return b
	}
	base, start := b[0], len(b)
	b[0] = rom2ScriptFormVersion
	for _, v := range w.rom2.Scenario {
		b = binary.LittleEndian.AppendUint32(b, uint32(v))
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(w.rom2.Groups)))
	for _, row := range w.rom2.Groups {
		b = binary.LittleEndian.AppendUint32(b, row.Owner)
		b = binary.LittleEndian.AppendUint32(b, row.Group)
		flag := byte(0)
		if row.Forced {
			flag = 1
		}
		b = append(b, flag)
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'R', '2', 'S', '1')
}

func (w *World) unmarshalROM2ScriptState(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed ROM2 script state section") }
	if len(data) < headerLen+9+rom2ScenarioWireLen+4 || !bytes.Equal(data[len(data)-4:], []byte("R2S1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= rom2ScriptFormVersion || span < rom2ScenarioWireLen+4 || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start+rom2ScenarioWireLen:]))
	if count > maxROM2Groups || span != rom2ScenarioWireLen+4+rom2GroupRecordLen*count {
		return fail()
	}
	state := &rom2ScriptState{}
	for i := range state.Scenario {
		state.Scenario[i] = int32(binary.LittleEndian.Uint32(data[start+4*i:]))
	}
	for n := uint64(0); n < count; n++ {
		o := start + rom2ScenarioWireLen + 4 + rom2GroupRecordLen*int(n)
		flag := data[o+8]
		row := rom2GroupActivity{Owner: binary.LittleEndian.Uint32(data[o:]), Group: binary.LittleEndian.Uint32(data[o+4:]), Forced: flag != 0}
		if flag > 1 {
			return fail()
		}
		if n > 0 {
			prior := state.Groups[n-1]
			if row.Owner < prior.Owner || row.Owner == prior.Owner && row.Group <= prior.Group {
				return fail()
			}
		}
		state.Groups = append(state.Groups, row)
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	var checks []ScriptCheck
	var instants []ScriptInstant
	var triggers []ScriptTrigger
	if next.script != nil {
		checks, instants, triggers = next.script.checks, next.script.instants, next.script.triggers
	}
	script, err := newScript(ScriptROM2, checks, instants, triggers)
	if err != nil {
		return err
	}
	next.script, next.rom2 = script, state
	// The byte form carries no arm: a second-game world's rows take theirs
	// from the second game's table again.
	next.spells = append([]SpellRule(nil), next.spells...)
	AssignSecondGameArms(next.spells)
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
