package mapload

import (
	"fmt"

	"againrom/pkg/formats/alm"
	"againrom/pkg/sim"
)

// OriginalTerrainPlanes constructs the raw, fixed-stride original terrain,
// not the reduced native passability grid. TERR-PASS-049/050 fix the fills,
// ordered ingest and border stores. This is the constructor baseline only;
// saved overlays and later object callbacks belong to the caller.
func OriginalTerrainPlanes(m *alm.Map, costs [11]byte) (*sim.SavedCellPlanes, error) {
	if m == nil || m.Width < 8 || m.Width > 256 || m.Height < 8 || m.Height > 256 {
		return nil, fmt.Errorf("original terrain requires dimensions in 8..256")
	}
	n := m.Width * m.Height
	if len(m.Tiles) != n || len(m.Altitudes) != n || len(m.Overlay) != n || costs[0] != 255 {
		return nil, fmt.Errorf("original terrain has incomplete planes or invalid reject cost")
	}
	p := &sim.SavedCellPlanes{Costs: costs}
	for i := range p.Cost {
		p.Cost[i], p.CostKnown[i] = 1, 1
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			i, cell := y*m.Width+x, y*256+x
			word := m.Tiles[i]
			index := word & 0x3ff
			// These reachable pairs were never initialized by ROM1. Rejecting
			// the producer is explicit; assigning our native fallback is not.
			if index&0x300 != 0x200 && index&15 < 14 && (index>>6)&15 >= 13 {
				return nil, fmt.Errorf("original terrain cell %04x uses an uninitialized terrain pair", cell)
			}
			class, cost := classifyCosts(word, costs)
			p.Cost[cell], p.Height[cell] = cost, m.Altitudes[i]
			if word&0x2000 != 0 || class == classMountain {
				p.Static[cell] = 1
			}
			if index&0x300 == 0x200 {
				p.Static[cell] = 1
			}
			if m.Overlay[i] != 0 {
				p.Static[cell] = 5
			}
		}
	}
	stamp := func(row, col int) { p.Static[row*256+col] = 0x1f }
	if m.Width == m.Height {
		for i := 0; i < m.Width; i++ {
			for j := 0; j < 8; j++ {
				stamp(i, j)
				stamp(i, m.Width-1-j)
				stamp(j, i)
				stamp(m.Width-1-j, i)
			}
		}
	} else {
		for i := 0; i < 256; i++ {
			for j := 0; j < 8; j++ {
				stamp(i, j)
				stamp(i, 255-j)
				stamp(j, i)
				stamp(255-j, i)
			}
		}
		for y := 0; y < m.Height; y++ {
			for j := 0; j < 8; j++ {
				stamp(y, m.Width-1-j)
			}
		}
		for x := 0; x < m.Width; x++ {
			for j := 0; j < 8; j++ {
				stamp(m.Height-1-j, x)
			}
		}
	}
	p.Dynamic = p.Static
	return p, nil
}
