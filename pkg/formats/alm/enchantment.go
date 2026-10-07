package alm

import (
	"encoding/binary"
)

const enchantmentHeadSize = 26
const enchantmentElementSize = 6

func (m *Map) decodeEnchantments() error {
	if !m.Present(9) {
		m.Enchantments = nil
		return nil
	}
	b := m.TileMarkers.Body
	off := 0
	// Count comes from the file. Bound the capacity by the smallest possible
	// number of records the already in-bounds body could hold before allocating;
	// a malformed raw section may advertise billions beside only a few bytes.
	out := make([]Enchantment, 0, enchantmentCapacity(m.TileMarkers.Count, len(b)))
	for i := uint32(0); i < m.TileMarkers.Count; i++ {
		if len(b)-off < enchantmentHeadSize {
			// Type 9 was historically raw-preserved, and editor inputs may
			// carry a body outside the newly decoded item grammar. Preserve
			// that acceptance surface. A non-zero item link still fails in
			// mapload because there is no decoded record for it to resolve.
			m.Enchantments = nil
			return nil
		}
		h := b[off : off+enchantmentHeadSize]
		n := binary.LittleEndian.Uint32(h[22:26])
		span := uint64(n) * enchantmentElementSize
		if span > uint64(len(b)-off-enchantmentHeadSize) {
			m.Enchantments = nil
			return nil
		}
		r := Enchantment{
			Tag:      binary.LittleEndian.Uint32(h[0:4]),
			X:        binary.LittleEndian.Uint32(h[4:8]),
			Y:        binary.LittleEndian.Uint32(h[8:12]),
			A:        binary.LittleEndian.Uint16(h[12:14]),
			B:        binary.LittleEndian.Uint16(h[14:16]),
			C:        binary.LittleEndian.Uint16(h[16:18]),
			SpellRaw: binary.LittleEndian.Uint32(h[18:22]),
			Elements: make([]EnchantmentElement, n),
		}
		off += enchantmentHeadSize
		for k := range r.Elements {
			e := b[off+k*enchantmentElementSize : off+(k+1)*enchantmentElementSize]
			r.Elements[k] = EnchantmentElement{
				Kind: binary.LittleEndian.Uint16(e[0:2]),
				Low:  binary.LittleEndian.Uint16(e[2:4]),
				High: binary.LittleEndian.Uint16(e[4:6]),
			}
		}
		off += int(span)
		out = append(out, r)
	}
	if off != len(b) {
		m.Enchantments = nil
		return nil
	}
	m.Enchantments = out
	return nil
}

func enchantmentCapacity(count uint32, bodyLen int) int {
	capacity := bodyLen / enchantmentHeadSize
	if uint64(count) < uint64(capacity) {
		return int(count)
	}
	return capacity
}
