package mapload_test

import (
	"testing"

	"againrom/pkg/mapload"
)

func TestMissionStartScattersCompanionsAndRetainsUnplacedActors(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	w, start := mustStart(t, m, party(5))
	spread := false
	for _, c := range start.Cells[1:] {
		if c.X < 16 || c.X > 18 || c.Y < 19 || c.Y > 21 {
			spread = true
		}
	}
	if !spread {
		t.Fatalf("companions still form a fixed adjacent ring: %v", start.Cells)
	}
	if start.Cells[0] != start.Drop || w.Entities()[start.IDs[0]].OffMap {
		t.Fatal("the free drop cell must receive the hero")
	}
	w, start = mustStart(t, startMap(t, 17, 17, mapload.Cell{X: 8, Y: 8}), party(3))
	for i, id := range start.IDs {
		if got := w.Entities()[id].OffMap; got != (i != 0) {
			t.Errorf("member %d off-map=%v; refused companions must not stack on the hero", i, got)
		}
	}
}

func TestMissionStartRetriesABlockedHeroDrop(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	m.Tiles[20*40+17] |= 0x2000
	w, start := mustStart(t, m, party(1))
	if start.Cells[0] == start.Drop || w.Entities()[start.IDs[0]].OffMap {
		t.Fatalf("blocked hero drop was not retried: %+v", start)
	}
}
