package sim

import (
	"bytes"
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
	base, start := b[0], len(b)
	b[0] = originalDeadHealthFormVersion
	b = binary.LittleEndian.AppendUint32(b, uint32(len(rows)))
	for _, row := range rows {
		b = binary.LittleEndian.AppendUint32(b, uint32(row.ID))
		b = binary.LittleEndian.AppendUint32(b, uint32(row.Source.MaxHP))
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
	return append(b, base, 'D', 'M', 'H', '1')
}

func (w *World) unmarshalOriginalDeadHealth(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed original dead maximum health section") }
	if len(data) < headerLen+13+originalDeadHealthRecordLen || !bytes.Equal(data[len(data)-4:], []byte("DMH1")) {
		return fail()
	}
	version := data[len(data)-5]
	span := uint64(binary.LittleEndian.Uint32(data[len(data)-9:]))
	if version >= originalDeadHealthFormVersion || span < 4+originalDeadHealthRecordLen || span > uint64(len(data)-headerLen-9) {
		return fail()
	}
	start := len(data) - 9 - int(span)
	count := uint64(binary.LittleEndian.Uint32(data[start:]))
	if count == 0 || span != 4+originalDeadHealthRecordLen*count {
		return fail()
	}
	base := bytes.Clone(data[:start])
	base[0] = version
	next := *w
	if err := next.unmarshalForm(base, proof); err != nil {
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
