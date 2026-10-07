package sim

import (
	"encoding/binary"
	"fmt"
)

const maxSavedMotionBytes = 64 << 20

// Form82 adds an independently bounded suffix after complete form81. A zero
// span is absent. Present payload: three u32 counts; variable motions; 64-byte
// typed cells; four-byte block deltas. No decoder allocation trusts a count
// without first proving its minimum byte span.
func (w *World) appendSavedMotions(b []byte) []byte {
	start := len(b)
	if s := w.savedMotion; s != nil {
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s.Motions)))
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s.Cells)))
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s.Blocks)))
		for _, m := range s.Motions {
			b = binary.LittleEndian.AppendUint32(b, uint32(m.Entity))
			b = binary.LittleEndian.AppendUint16(b, m.Position.Cell)
			b = binary.LittleEndian.AppendUint16(b, m.Position.PackedCell)
			b = append(b, m.Position.FineX, m.Position.FineY)
			b = binary.LittleEndian.AppendUint16(b, m.Position.Residue)
			b = binary.LittleEndian.AppendUint32(b, m.Position.TerrainKey)
			b = append(b, m.Mover[:]...)
			var flags byte
			if m.Current {
				flags |= 1
			}
			if m.Active {
				flags |= 2
			}
			b = append(b, flags)
			b = binary.LittleEndian.AppendUint16(b, uint16(len(m.Issue)))
			b = binary.LittleEndian.AppendUint32(b, uint32(len(m.StaticRoute)))
			b = binary.LittleEndian.AppendUint32(b, uint32(len(m.DynamicRoute)))
			b = binary.LittleEndian.AppendUint32(b, m.ActorAction)
			b = append(b, m.Issue...)
			for _, route := range [][]uint16{m.StaticRoute, m.DynamicRoute} {
				for _, cell := range route {
					b = binary.LittleEndian.AppendUint16(b, cell)
				}
			}
		}
		for _, c := range s.Cells {
			b = binary.LittleEndian.AppendUint16(b, c.Cell)
			b = append(b, c.Payload[:]...)
			for _, slot := range []SavedActorSlot{c.Ground, c.Air} {
				b = binary.LittleEndian.AppendUint32(b, uint32(slot.Entity))
				flag := byte(0)
				if slot.Bound {
					flag = 1
				}
				b = append(b, flag)
			}
		}
		for _, c := range s.Blocks {
			b = binary.LittleEndian.AppendUint16(b, c.Cell)
			b = append(b, c.Dyn, c.Static)
		}
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitSavedMotions(data []byte) ([]byte, *savedActorMotionState, error) {
	if len(data) < headerLen+4 {
		return nil, nil, fmt.Errorf("truncated saved motion footer")
	}
	end := len(data) - 4
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span > maxSavedMotionBytes || span > uint64(end-headerLen) {
		return nil, nil, fmt.Errorf("invalid saved motion span")
	}
	if span == 0 {
		return data[:end], nil, nil
	}
	start := end - int(span)
	p := data[start:end]
	if len(p) < 12 {
		return nil, nil, fmt.Errorf("truncated saved motion counts")
	}
	nm, nc, nb := uint64(binary.LittleEndian.Uint32(p)), uint64(binary.LittleEndian.Uint32(p[4:])), uint64(binary.LittleEndian.Uint32(p[8:]))
	if nm > 1<<20 || nc > 65536 || nb > 65536 || 12+nm*211+nc*64+nb*4 > span {
		return nil, nil, fmt.Errorf("saved motion counts exceed payload")
	}
	p = p[12:]
	s := &savedActorMotionState{Motions: make([]SavedActorMotion, int(nm)), Cells: make([]SavedActorCell, int(nc)), Blocks: make([]SavedActorBlock, int(nb))}
	for i := range s.Motions {
		if len(p) < 211 {
			return nil, nil, fmt.Errorf("truncated saved motion record")
		}
		m := &s.Motions[i]
		m.Entity = EntityID(binary.LittleEndian.Uint32(p))
		m.Position = SavedActorPosition{Cell: binary.LittleEndian.Uint16(p[4:]), PackedCell: binary.LittleEndian.Uint16(p[6:]), FineX: p[8], FineY: p[9], Residue: binary.LittleEndian.Uint16(p[10:]), TerrainKey: binary.LittleEndian.Uint32(p[12:])}
		copy(m.Mover[:], p[16:196])
		if p[196] > 3 {
			return nil, nil, fmt.Errorf("invalid saved motion flags")
		}
		m.Current, m.Active = p[196]&1 != 0, p[196]&2 != 0
		ni, ns, nd := int(binary.LittleEndian.Uint16(p[197:])), uint64(binary.LittleEndian.Uint32(p[199:])), uint64(binary.LittleEndian.Uint32(p[203:]))
		m.ActorAction = binary.LittleEndian.Uint32(p[207:])
		p = p[211:]
		if ni > 256 || ns > 65536 || nd > 65536 || uint64(ni)+2*(ns+nd) > uint64(len(p)) {
			return nil, nil, fmt.Errorf("saved motion lists exceed record")
		}
		m.Issue = string(p[:ni])
		p = p[ni:]
		for _, r := range []struct {
			n   uint64
			dst *[]uint16
		}{{ns, &m.StaticRoute}, {nd, &m.DynamicRoute}} {
			*r.dst = make([]uint16, int(r.n))
			for k := range *r.dst {
				(*r.dst)[k] = binary.LittleEndian.Uint16(p)
				p = p[2:]
			}
		}
	}
	if uint64(len(p)) != nc*64+nb*4 {
		return nil, nil, fmt.Errorf("saved motion cell/block span mismatch")
	}
	for i := range s.Cells {
		c := &s.Cells[i]
		c.Cell = binary.LittleEndian.Uint16(p)
		copy(c.Payload[:], p[2:54])
		for layer, slot := range []*SavedActorSlot{&c.Ground, &c.Air} {
			at := 54 + 5*layer
			if p[at+4] > 1 {
				return nil, nil, fmt.Errorf("invalid saved actor slot flag")
			}
			*slot = SavedActorSlot{Key: binary.LittleEndian.Uint32(c.Payload[4+4*layer:]), Entity: EntityID(binary.LittleEndian.Uint32(p[at:])), Bound: p[at+4] != 0}
		}
		p = p[64:]
	}
	for i := range s.Blocks {
		s.Blocks[i] = SavedActorBlock{Cell: binary.LittleEndian.Uint16(p), Dyn: p[2], Static: p[3]}
		p = p[4:]
	}
	return data[:start], s, nil
}
