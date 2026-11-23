package sim

import (
	"encoding/binary"
	"fmt"
	"slices"
)

// wideRegistersFormVersion carries non-default registers past 100 (R2-SESSION-019).
const wideRegistersFormVersion byte = 142

const wideRegisterRecordLen = 8

func (w *World) appendWideRegisters(b []byte) []byte {
	if w.script.Extended() || len(w.scriptVars) == 0 {
		return b
	}
	ws := beginWrapper(b)
	le := binary.LittleEndian
	for i, v := range w.scriptVars {
		r := int32(scriptRegisters + i)
		if v == w.defaultWideRegister(r) {
			continue
		}
		b = le.AppendUint32(b, uint32(r))
		b = le.AppendUint32(b, uint32(v))
	}
	if len(b) == ws.at {
		return b
	}
	return endWrapper(b, ws, wideRegistersFormVersion)
}

func (w *World) unmarshalWideRegisters(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed wide-register section") }
	env, ok := readWrapper(data, wideRegistersFormVersion)
	if !ok {
		return fail()
	}
	s := env.section()
	if len(s) == 0 || len(s)%wideRegisterRecordLen != 0 {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if next.script.Extended() || len(next.scriptVars) == 0 {
		return fail()
	}
	vars := slices.Clone(next.scriptVars)
	le := binary.LittleEndian
	prior := int64(-1)
	for at := 0; at < len(s); at += wideRegisterRecordLen {
		reg := le.Uint32(s[at:])
		val := int32(le.Uint32(s[at+4:]))
		if int64(reg) <= prior || reg < scriptRegisters || int(reg)-scriptRegisters >= len(vars) {
			return fail()
		}
		if val == next.defaultWideRegister(int32(reg)) {
			return fail()
		}
		prior = int64(reg)
		vars[int(reg)-scriptRegisters] = val
	}
	next.scriptVars = vars
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
