package game

import (
	"image"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// topRowHidden: every playable cell of the first playable row blocks and nothing stands in it.
func topRowHidden(plane []byte, w, h int, cells []image.Point) bool {
	if w <= 0 || len(plane) < w*h {
		return true
	}
	first := -1
	for y := 0; y < h && first < 0; y++ {
		for x := 0; x < w; x++ {
			if plane[y*w+x]&2 == 0 {
				first = y
				break
			}
		}
	}
	if first < 0 {
		return true
	}
	for x := 0; x < w; x++ {
		if c := plane[first*w+x]; c&2 == 0 && c&1 == 0 {
			return false
		}
	}
	for _, c := range cells {
		if c.Y == first {
			return false
		}
	}
	return true
}

func (mw *mapWorld) decideTopRow(draws []ui.MapEntity) {
	if mw.topRowDecided {
		return
	}
	if mw.mission == nil || mw.mission.state == nil || mw.mission.state.Map == nil {
		return
	}
	mw.topRowDecided = true
	m := mw.mission.state.Map
	var cells []image.Point
	for _, d := range draws {
		cells = append(cells, d.Cell)
	}
	for _, o := range m.Objects {
		cells = append(cells, image.Pt(int(o.X>>8), int(o.Y>>8)))
	}
	for _, s := range mw.world.Sacks() {
		cells = append(cells, image.Pt(int(s.X), int(s.Y)))
	}
	mw.view.SetTopRowShown(!topRowHidden(mapload.Passability(m), m.Width, m.Height, cells))
}
