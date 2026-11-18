package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// Creatures of one look share one count and one level, whichever of the
// look's Units rows their placements name: two kills of two rows make every
// instance of the look level 1.
func TestReleaseSecondEnemyCardCountsOneLookAlike(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("one look")
	app.Layout(1024, 768)
	const mission = 31
	if err := app.OpenMission(f.DirectNewGame(mission)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		_ = app.HeadlessKey("enter")
	}
	w := live.world

	// The look with the most Units rows among the placed creatures of one
	// owner, so the creatures kept do not fight each other.
	var look [2]int32
	var rows []int
	kept := map[int][]sim.EntityID{}
	for key, byRow := range secondCardLooks(f, live) {
		if key[0] < 64 {
			continue
		}
		owned := map[uint8]map[int][]sim.EntityID{}
		for row, list := range byRow {
			for _, id := range list {
				o := uint8(mustEntity(t, live, id).Owner)
				if owned[o] == nil {
					owned[o] = map[int][]sim.EntityID{}
				}
				owned[o][row] = append(owned[o][row], id)
			}
		}
		for _, part := range owned {
			var list []int
			for row := range part {
				list = append(list, row)
			}
			slices.Sort(list)
			if len(list) > len(rows) || len(list) == len(rows) && len(list) > 0 && list[0] < rows[0] {
				look, rows, kept = key, list, part
			}
		}
	}
	if len(rows) < 3 {
		t.Fatalf("mission %d holds no look of three Units rows under one owner: %v %v", mission, look, rows)
	}
	first := data.FindUnit(f.Table.Units, look[0], look[1])

	secondCardSetCount(w, first, 0)
	enemyCardKill(t, live, kept[rows[1]][0])
	enemyCardKill(t, live, kept[rows[2]][0])
	if got := enemyCardCount(w, first); got != 2 {
		t.Fatalf("look %v: first row %d counts %d after two kills of rows %d and %d, want 2", look, first, got, rows[1], rows[2])
	}
	check := func(w *sim.World, live *mapWorld, when string) {
		t.Helper()
		for _, row := range rows {
			for _, id := range kept[row] {
				if l := w.KnowledgeLevel(mustEntity(t, live, id)); l != 1 {
					t.Fatalf("look %v %s: row %d entity %d level %d, want 1", look, when, row, id, l)
				}
			}
		}
	}
	check(w, live, "after two kills")
}
