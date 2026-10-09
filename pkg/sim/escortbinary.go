package sim

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const escortFormVersion byte = 116
const escortResidueRecordLen = 6

func escortResidueFault(e Entity) error {
	if e.EscortOrder > escortOrderIdle ||
		(e.EscortOrder != escortOrderNone || e.EscortTurnPending) && (!e.HasEscortTarget || !escortState(e.ActorState)) {
		return fmt.Errorf("sim: invalid escort continuation for actor %d", e.ID)
	}
	return nil
}

func (w *World) appendEscortResidues(b []byte) []byte {
	count := 0
	for _, e := range w.entities {
		if e.EscortOrder != escortOrderNone || e.EscortTurnPending {
			count++
		}
	}
	if count == 0 {
		return b
	}
	base, start := b[0], len(b)
	b[0] = escortFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if e.EscortOrder == escortOrderNone && !e.EscortTurnPending {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = append(b, e.EscortOrder, boolFlag(e.EscortTurnPending))
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'E', 'S', 'C', '1')
}

func (w *World) unmarshalEscortResidues(data []byte) error {
	fail := func() error { return fmt.Errorf("malformed escort continuation") }
	if len(data) < headerLen+13+escortResidueRecordLen || !bytes.Equal(data[len(data)-4:], []byte("ESC1")) {
		return fail()
	}
	version := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if version >= escortFormVersion || span < 4+escortResidueRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+escortResidueRecordLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = version
	next := *w
	if err := next.UnmarshalBinary(base); err != nil {
		return err
	}
	if count > uint64(len(next.entities)) {
		return fail()
	}
	last := -1
	for n := 0; n < int(count); n++ {
		at := start + 4 + escortResidueRecordLen*n
		i := indexOfEntity(next.entities, EntityID(binary.LittleEndian.Uint32(data[at:])))
		if i <= last || data[at+5] > 1 || data[at+4] == escortOrderNone && data[at+5] == 0 {
			return fail()
		}
		e := &next.entities[i]
		e.EscortOrder, e.EscortTurnPending = data[at+4], data[at+5] == 1
		if err := escortResidueFault(*e); err != nil {
			return fail()
		}
		last = i
	}
	canonical, err := next.MarshalBinary()
	if err != nil || !bytes.Equal(canonical, data) {
		return fail()
	}
	*w = next
	return nil
}
