package sim

import (
	"encoding/binary"
	"fmt"
)

// scriptSlotFormVersion carries the second game's source action slot of each
// compiled instant, which the script section's instant record does not hold.
// A form without it reads every slot as 0, which no Pointer names.
const scriptSlotFormVersion byte = 129

// appendScriptSlots writes one int32 slot per instant after the instant
// count; a World whose instants all have slot 0 writes no form.
func (w *World) appendScriptSlots(b []byte) []byte {
	if w.script == nil || !hasScriptSlots(w.script.instants) {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(w.script.instants)))
	for _, in := range w.script.instants {
		b = binary.LittleEndian.AppendUint32(b, uint32(in.Slot))
	}
	return endWrapper(b, ws, scriptSlotFormVersion)
}

func hasScriptSlots(instants []ScriptInstant) bool {
	for _, in := range instants {
		if in.Slot != 0 {
			return true
		}
	}
	return false
}

func (w *World) unmarshalScriptSlots(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed script slot section") }
	env, ok := readWrapper(data, scriptSlotFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) < 4 {
		return fail()
	}
	n := uint64(binary.LittleEndian.Uint32(s))
	if uint64(len(s)) != 4+4*n {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if next.script == nil || next.script.Dialect() != ScriptROM2 || uint64(len(next.script.instants)) != n {
		return fail()
	}
	instants := append([]ScriptInstant(nil), next.script.instants...)
	for i := range instants {
		slot := int32(binary.LittleEndian.Uint32(s[4+4*i:]))
		if slot < 0 {
			return fail()
		}
		instants[i].Slot = slot
	}
	script, err := newScript(ScriptROM2, next.script.checks, instants, next.script.triggers)
	if err != nil {
		return err
	}
	next.script = script
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
