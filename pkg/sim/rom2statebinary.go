package sim

import (
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
	ws := beginWrapper(b)
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
	return endWrapper(b, ws, rom2ScriptFormVersion)
}

func (w *World) unmarshalROM2ScriptState(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed ROM2 script state section") }
	env, ok := readWrapper(data, rom2ScriptFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < rom2ScenarioWireLen+4 {
		return fail()
	}
	start := env.start
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
	next := *w
	// The byte form carries no arm: a second-game world's rows take theirs
	// from the second game's table again, before the base form checks its
	// retained areas against them.
	next.rowArms = AssignSecondGameArms
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	next.rowArms = nil
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
	next.spells = append([]SpellRule(nil), next.spells...)
	AssignSecondGameArms(next.spells)
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
