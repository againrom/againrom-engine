package sim

import (
	"encoding/binary"
	"fmt"
)

const bookSelectionFormVersion byte = 108

func (w *World) appendBookSelections(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.AdmittedBookSpell != 0 {
			count++
		}
	}
	if count == 0 {
		return b
	}
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if e.AdmittedBookSpell == 0 {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = binary.LittleEndian.AppendUint16(b, e.AdmittedBookSpell)
	}
	return endWrapper(b, ws, bookSelectionFormVersion)
}

func (w *World) unmarshalBookSelections(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed admitted book selection") }
	env, ok := readWrapper(data, bookSelectionFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 10 {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+6*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	var prior EntityID
	for n := uint64(0); n < count; n++ {
		at := start + 4 + 6*int(n)
		id := EntityID(binary.LittleEndian.Uint32(data[at:]))
		index := indexOfEntity(next.entities, id)
		spell := binary.LittleEndian.Uint16(data[at+4:])
		if index < 0 || n > 0 && id <= prior || spell == 0 || spell > 28 {
			return fail()
		}
		prior = id
		next.entities[index].AdmittedBookSpell = spell
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
