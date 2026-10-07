package sim

import (
	"encoding/binary"
	"fmt"
)

func (w *World) appendSavedWorldEffects(b []byte) []byte {
	start := len(b)
	if s := w.savedWorldEffects; s != nil {
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s.Areas)))
		for _, a := range s.Areas {
			b = binary.LittleEndian.AppendUint32(b, a.ID)
			b = binary.LittleEndian.AppendUint32(b, uint32(a.Root))
			b = binary.LittleEndian.AppendUint32(b, a.Identity)
			b = binary.LittleEndian.AppendUint16(b, a.Key)
			b = append(b, a.Layer, a.Mode)
			b = binary.LittleEndian.AppendUint16(b, a.Spell)
			b = binary.LittleEndian.AppendUint32(b, uint32(len(a.Cells)))
			for _, key := range a.Cells {
				b = binary.LittleEndian.AppendUint16(b, key)
			}
		}
		b = binary.LittleEndian.AppendUint32(b, uint32(len(s.Projectiles)))
		for _, p := range s.Projectiles {
			b = binary.LittleEndian.AppendUint16(b, p.ID)
			b = binary.LittleEndian.AppendUint16(b, p.Phases)
			b = binary.LittleEndian.AppendUint32(b, uint32(p.Target))
			var flags byte
			if p.HasTarget {
				flags |= 1
			}
			if p.Retired {
				flags |= 2
			}
			if p.TargetDetached {
				flags |= 4
			}
			b = append(b, flags)
		}
	}
	return binary.LittleEndian.AppendUint32(b, uint32(len(b)-start))
}

func splitSavedWorldEffects(data []byte) ([]byte, *SavedWorldEffects, error) {
	fail := func() ([]byte, *SavedWorldEffects, error) {
		return nil, nil, fmt.Errorf("sim: invalid world-effect continuation footer")
	}
	if len(data) < headerLen+4 {
		return fail()
	}
	end := len(data) - 4
	span := uint64(binary.LittleEndian.Uint32(data[end:]))
	if span == 0 {
		return data[:end], nil, nil
	}
	if span < 8 || span > maxCarriedResumeBytes || span > uint64(end-headerLen) {
		return fail()
	}
	start := end - int(span)
	r := savedObjectReader{data: data[start:end]}
	n := r.u32()
	if n > 0x7fff || uint64(n)*22+4 > uint64(len(r.data)) {
		return fail()
	}
	s := &SavedWorldEffects{}
	for range n {
		a := SavedAreaDriver{ID: r.u32(), Root: int32(r.u32()), Identity: r.u32(), Key: r.u16(), Layer: r.u8(), Mode: r.u8(), Spell: r.u16()}
		count := r.u32()
		if r.err != nil || count > 65536 || uint64(count)*2 > uint64(len(r.data)) {
			return fail()
		}
		for range count {
			a.Cells = append(a.Cells, r.u16())
		}
		s.Areas = append(s.Areas, a)
	}
	n = r.u32()
	if r.err != nil || n > 65535 || uint64(n)*9 != uint64(len(r.data)) {
		return fail()
	}
	for range n {
		p := SavedProjectileDriver{ID: r.u16(), Phases: r.u16(), Target: EntityID(r.u32())}
		flags := r.u8()
		if flags&^byte(7) != 0 {
			return fail()
		}
		p.HasTarget, p.Retired, p.TargetDetached = flags&1 != 0, flags&2 != 0, flags&4 != 0
		s.Projectiles = append(s.Projectiles, p)
	}
	if r.err != nil || len(r.data) != 0 {
		return fail()
	}
	return data[:start], s, nil
}
