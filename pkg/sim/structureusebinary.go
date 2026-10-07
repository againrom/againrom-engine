package sim

import (
	"encoding/binary"
	"fmt"
)

type structureUseMetadata struct {
	ID     StructureID
	Kind   uint16
	Amount int32
}

func (w *World) appendStructureUses(b []byte) []byte {
	count := 0
	for _, s := range w.structures {
		if s.Kind != 0 {
			count++
		}
	}
	if count == 0 && len(w.structureUses) == 0 {
		return binary.LittleEndian.AppendUint32(b, 0)
	}
	start := len(b)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for _, s := range w.structures {
		if s.Kind != 0 {
			b = binary.LittleEndian.AppendUint32(b, uint32(s.ID))
			b = binary.LittleEndian.AppendUint16(b, s.Kind)
			b = binary.LittleEndian.AppendUint32(b, uint32(s.UseAmount))
		}
	}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(w.structureUses)))
	for _, u := range w.structureUses {
		b = binary.LittleEndian.AppendUint32(b, uint32(u.Entity))
		b = binary.LittleEndian.AppendUint32(b, uint32(u.Structure))
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitStructureUses(data []byte) ([]byte, []structureUseMetadata, []StructureUse, error) {
	fail := func() ([]byte, []structureUseMetadata, []StructureUse, error) {
		return nil, nil, nil, fmt.Errorf("sim: invalid structure-use footer")
	}
	if len(data) < headerLen+4 {
		return fail()
	}
	end := len(data) - 4
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span > uint64(end-headerLen) {
		return fail()
	}
	if span == 0 {
		return data[:end], nil, nil, nil
	}
	if span < 8 {
		return fail()
	}
	start := end - int(span)
	at := start
	n := uint64(binary.LittleEndian.Uint32(data[at:]))
	at += 4
	if n > uint64((end-at-4)/10) {
		return fail()
	}
	var metadata []structureUseMetadata
	for k := uint64(0); k < n; k++ {
		r := structureUseMetadata{StructureID(binary.LittleEndian.Uint32(data[at:])), binary.LittleEndian.Uint16(data[at+4:]), int32(binary.LittleEndian.Uint32(data[at+6:]))}
		at += 10
		if r.Kind == 0 || r.Amount < 0 || (len(metadata) > 0 && metadata[len(metadata)-1].ID >= r.ID) {
			return fail()
		}
		metadata = append(metadata, r)
	}
	count := uint64(binary.LittleEndian.Uint32(data[at:]))
	at += 4
	if count != uint64((end-at)/8) || (end-at)%8 != 0 {
		return fail()
	}
	if n == 0 && count == 0 {
		return fail()
	}
	var uses []StructureUse
	for k := uint64(0); k < count; k++ {
		u := StructureUse{EntityID(binary.LittleEndian.Uint32(data[at:])), StructureID(binary.LittleEndian.Uint32(data[at+4:]))}
		at += 8
		if len(uses) > 0 && uses[len(uses)-1].Entity >= u.Entity {
			return fail()
		}
		uses = append(uses, u)
	}
	return data[:start], metadata, uses, nil
}

func (w *World) structureUseFault() error {
	for _, s := range w.structures {
		if s.UseAmount < 0 || s.Kind == 0 && s.UseAmount != 0 {
			return fmt.Errorf("sim: invalid structure-use metadata %d", s.ID)
		}
	}
	for k, u := range w.structureUses {
		i, si := indexOfEntity(w.entities, u.Entity), indexOfStructure(w.structures, u.Structure)
		if i < 0 || si < 0 || !w.entities[i].Alive() || w.entities[i].OffMap || !w.structures[si].Usable() || k > 0 && w.structureUses[k-1].Entity >= u.Entity {
			return fmt.Errorf("sim: invalid structure-use actor/target %d/%d", u.Entity, u.Structure)
		}
	}
	return nil
}
