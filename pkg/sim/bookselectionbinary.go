package sim

import (
	"bytes"
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
	base, start := b[0], len(b)
	b[0] = bookSelectionFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if e.AdmittedBookSpell == 0 {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = binary.LittleEndian.AppendUint16(b, e.AdmittedBookSpell)
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'B', 'S', 'L', '1')
}

func (w *World) unmarshalBookSelections(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed admitted book selection") }
	if len(data) < headerLen+19 || !bytes.Equal(data[len(data)-4:], []byte("BSL1")) {
		return fail()
	}
	baseVersion := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if baseVersion >= bookSelectionFormVersion || span < 10 || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || count > 65535 || span != 4+6*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = baseVersion
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
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
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
