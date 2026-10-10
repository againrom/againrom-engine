package game

import (
	"againrom/pkg/formats/alm"
	"againrom/pkg/ui"
)

// missionMusicAreas is the area array the mission's music reads: the head
// record first, only when its first theme names a list entry, then the area
// records in file order (R2-ENGINE-336). The array is built from this map
// alone (DIV-2861). A map without type-12 data has none.
func missionMusicAreas(m *alm.Map) []ui.MusicArea {
	if m == nil {
		return nil
	}
	head, areas, ok := m.MusicAreas()
	if !ok {
		return nil
	}
	out := make([]ui.MusicArea, 0, len(areas)+1)
	if head.Themes[0] >= 0 {
		out = append(out, ui.MusicArea(head))
	}
	for _, a := range areas {
		out = append(out, ui.MusicArea(a))
	}
	return out
}

// musicHero is the primary hero's position for the music-area select: the
// centre of its cell in 1/256 tile, the world tick, and whether the hero is
// in the world (DIV-2861).
func (mw *mapWorld) musicHero() (x, y int32, tick uint64, ok bool) {
	if mw == nil || mw.world == nil {
		return 0, 0, 0, false
	}
	tick = mw.world.Tick()
	id, found := mw.primaryPartyID()
	if !found {
		return 0, 0, tick, false
	}
	e, found := mw.world.Entity(id)
	if !found {
		return 0, 0, tick, false
	}
	return e.X*256 + 128, e.Y*256 + 128, tick, true
}
