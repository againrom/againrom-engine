package sim

import (
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
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, e := range w.entities {
		if e.EscortOrder == escortOrderNone && !e.EscortTurnPending {
			continue
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(e.ID))
		b = append(b, e.EscortOrder, boolFlag(e.EscortTurnPending))
	}
	return endWrapper(b, ws, escortFormVersion)
}

func (w *World) unmarshalEscortResidues(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed escort continuation") }
	env, ok := readWrapper(data, escortFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+escortResidueRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+escortResidueRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
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
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
