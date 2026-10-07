package ui

import (
	"image"
	"slices"

	"againrom/pkg/render/terrain"
)

// The owner requests a short visible fire before the dead tree. This display
// interval is authored; it does not delay the world's fire history or damage.
const sceneryBurnTicks = 24

// SetScorchedCellsAt seeds a cold scene without replaying old fires, then delays
// newly scorched trees while their installed flame overlay animates. It follows
// the simulation tick, so pause and game speed also govern this presentation.
func (v *Viewer) SetScorchedCellsAt(cells []uint16, tick uint64) {
	if !v.scorchInitialized || tick < v.scorchTick {
		v.scorchInitialized = true
		v.scorchTick = tick
		v.scorchHistory = slices.Clone(cells)
		v.burningScenery = nil
		v.SetScorchedCells(cells)
		return
	}
	if slices.Equal(cells, v.scorchHistory) && len(v.burningScenery) == 0 {
		v.scorchTick = tick
		return
	}
	old := make(map[uint16]bool, len(v.scorchHistory))
	for _, cell := range v.scorchHistory {
		old[cell] = true
	}
	burning := make(map[uint16]uint64)
	ready := make([]uint16, 0, len(cells))
	for _, cell := range cells {
		born, active := v.burningScenery[cell]
		if !old[cell] && v.destructibleScenery(cell) {
			born, active = tick, true
		}
		if active && tick-born < sceneryBurnTicks {
			burning[cell] = born
		} else {
			ready = append(ready, cell)
		}
	}
	v.scorchTick, v.scorchHistory, v.burningScenery = tick, slices.Clone(cells), burning
	v.SetScorchedCells(ready)
}

func (v *Viewer) destructibleScenery(cell uint16) bool {
	x, y := int(cell&255), int(cell>>8)
	if v.staticSet == nil || x >= v.grid.Width || y >= v.grid.Height {
		return false
	}
	i := y*v.grid.Width + x
	if i >= len(v.grid.Overlay) || i >= len(v.baseTileWords) || v.baseTileWords[i]&0x2000 != 0 {
		return false
	}
	c := v.staticSet.Classes[v.grid.Overlay[i]]
	return c != nil && c.Dead != nil
}

// BurningScenery is ordered by the fire history, not by a presentation map.
func (v *Viewer) BurningScenery() []image.Point {
	var out []image.Point
	for _, cell := range v.scorchHistory {
		if _, ok := v.burningScenery[cell]; ok {
			out = append(out, image.Pt(int(cell&255), int(cell>>8)))
		}
	}
	return out
}

// cellTexture is the ordinary GPU upload's source and cache identity. Bit13
// selects the keyed dirt overlay on land only (TERR-DIRT-017); water keeps its
// animated surface even when a burned object stands on the cell.
func (v *Viewer) cellTexture(word uint16, col, row int) (*image.Paletted, *image.Paletted, cacheKey) {
	ref := v.resolveCell(word, col, row)
	src := v.set.Slot(ref.Slot).SubCell(ref.Sub)
	key := cacheKey{slot: ref.Slot, sub: ref.Sub, tint: v.lightTint()}
	if src == nil {
		key.tint = [3]uint8{}
		return nil, nil, key
	}
	var dirt *image.Paletted
	if terrain.IsImpassable(word) && !ref.Water {
		sub := terrain.DirtSubCell(col, row)
		dirt = v.set.Dirt.SubCell(sub)
		if dirt != nil {
			key.dirt = uint8(sub + 1)
		}
	}
	return src, dirt, key
}

// SetScorchedCells projects the world's fire history without changing the map
// or simulation. Both terrain geometries read these same tile words and lists.
func (v *Viewer) SetScorchedCells(cells []uint16) {
	if slices.Equal(cells, v.scorchedCells) {
		return
	}
	v.scorchedCells = slices.Clone(cells)
	v.grid.Tiles = slices.Clone(v.baseTileWords)
	for _, key := range cells {
		x, y := int(key&255), int(key>>8)
		if x >= v.grid.Width || y >= v.grid.Height {
			continue
		}
		at := y*v.grid.Width + x
		v.grid.Tiles[at] |= 0x2000
	}
	v.staticsFlat, v.staticCounts, v.animFlat = terrain.StaticPlacements(v.grid, v.staticSet, nil, 0, v.staticAnimGate)
	if v.proj != nil {
		v.staticsDisplaced, _, v.animDisplaced = terrain.StaticPlacements(v.grid, v.staticSet, v.proj.AnchorHeight, v.proj.MinV, v.staticAnimGate)
	}
	v.planeOrder = terrain.PlaneOrder(v.structuresFlat, v.staticsFlat)
	v.ambientStatics = ambientStaticCells(v.grid, v.staticSet)
}

// ScorchedScenery reports the tile and sprite selection used by the viewer.
func (v *Viewer) ScorchedScenery() (ground, objects int) {
	for _, key := range v.scorchedCells {
		x, y := int(key&255), int(key>>8)
		if x < v.grid.Width && y < v.grid.Height && v.tileWord(x, y)&0x2000 != 0 && !terrain.Resolve(v.tileWord(x, y)).Water {
			ground++
		}
	}
	for _, p := range v.staticsFlat {
		code := v.grid.Overlay[p.Cell.Y*v.grid.Width+p.Cell.X]
		if c := v.staticSet.Classes[code]; c != nil && c.Dead != nil && p.Class == c.Dead {
			objects++
		}
	}
	return ground, objects
}
