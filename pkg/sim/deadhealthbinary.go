package sim

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"
)

const originalDeadHealthFormVersion byte = 123
const originalDeadHealthRecordLen = 8

func (w *World) appendOriginalDeadHealth(b []byte) []byte {
	var rows []OriginalDeadActor
	for _, r := range w.originalDead {
		if !r.Source.MaxHPPresent {
			continue
		}
		row := r.OriginalDeadActor
		if i := indexOfEntity(w.entities, r.ID); i >= 0 {
			row.Source.MaxHP = w.entities[i].MaxHP
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return b
	}
	slices.SortFunc(rows, func(a, b OriginalDeadActor) int { return cmp.Compare(a.ID, b.ID) })
	ws := beginWrapper(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, row := range rows {
		b = binary.LittleEndian.AppendUint32(b, uint32(row.ID))
		b = binary.LittleEndian.AppendUint32(b, uint32(row.Source.MaxHP))
	}
	return endWrapper(b, ws, originalDeadHealthFormVersion)
}

func (w *World) unmarshalOriginalDeadHealth(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed original dead maximum health section") }
	env, ok := readWrapper(data, originalDeadHealthFormVersion)
	span := uint64(len(env.section()))
	if !ok || span < 4+originalDeadHealthRecordLen {
		return fail()
	}
	start := env.start
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+originalDeadHealthRecordLen*count {
		return fail()
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	if count > uint64(len(next.originalDead)) {
		return fail()
	}
	indices := make(map[EntityID]int, len(next.originalDead))
	for i, row := range next.originalDead {
		indices[row.ID] = i
	}
	var last EntityID
	for n := 0; n < int(count); n++ {
		at := start + 4 + originalDeadHealthRecordLen*n
		id := EntityID(binary.LittleEndian.Uint32(data[at:]))
		maxHP := int32(binary.LittleEndian.Uint32(data[at+4:]))
		i, ok := indices[id]
		if !ok || n > 0 && id <= last {
			return fail()
		}
		if ei := indexOfEntity(next.entities, id); ei >= 0 && next.entities[ei].MaxHP != maxHP {
			return fail()
		}
		next.originalDead[i].Source.MaxHP, next.originalDead[i].Source.MaxHPPresent = maxHP, true
		last = id
	}
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
