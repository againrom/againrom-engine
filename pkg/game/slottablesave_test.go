package game

import (
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// A second-game mission holds its cell slots from its start and moves them
// with each crossing; a mission SAVE and a cold LOAD keep that table, and the
// loaded World continues as the saved one does.
func TestSlotTableCrossesSaveAndColdLoad(t *testing.T) {
	f, app, screen, dir := secondBagFrontOf(t, nil, victimKindMission())
	secondBagEnter(t, f, app, screen)
	w := f.live.world
	hero := f.live.mission.ids[0]
	if !w.HoldsSlotTable() {
		t.Fatal("the mission start holds no slot table")
	}
	_, start, _, _ := w.SavedActorMotions()
	h, _ := w.Entity(hero)
	sim.Step(w, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: h.X + 4, Y: h.Y + 2})})
	for range 150 {
		sim.Step(w, nil)
	}
	if moved, _ := w.Entity(hero); moved.X == h.X && moved.Y == h.Y {
		t.Fatal("the hero did not cross a cell")
	}
	_, held, _, _ := w.SavedActorMotions()
	if slices.EqualFunc(start, held, func(a, b sim.SavedActorCell) bool { return a.Cell == b.Cell }) {
		t.Fatal("the crossing left the slot table where the start put it")
	}

	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := secondSaveFiles(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("mission SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	cold, _ := secondMissionLoadOf(t, victimKindMission(), dir, entries[0].Name)
	if cold.live == nil {
		t.Fatal("the cold LOAD opened no mission")
	}
	loaded := cold.live.world
	if !loaded.HoldsSlotTable() {
		t.Fatal("the loaded World holds no slot table")
	}
	_, back, _, _ := loaded.SavedActorMotions()
	if !slices.EqualFunc(held, back, func(a, b sim.SavedActorCell) bool { return a.Cell == b.Cell && a.Ground == b.Ground && a.Air == b.Air }) {
		t.Fatalf("the slot table changed across SAVE and LOAD:\n saved  %+v\n loaded %+v", held, back)
	}
	for _, x := range []*sim.World{w, loaded} {
		sim.Step(x, []sim.Command{sim.MoveTo(hero, sim.CellPoint{X: h.X, Y: h.Y})})
		for range 150 {
			sim.Step(x, nil)
		}
	}
	_, a, _, _ := w.SavedActorMotions()
	_, b, _, _ := loaded.SavedActorMotions()
	if !slices.EqualFunc(a, b, func(x, y sim.SavedActorCell) bool { return x.Cell == y.Cell && x.Ground == y.Ground && x.Air == y.Air }) {
		t.Fatalf("the loaded slot table crossed differently:\n saved  %+v\n loaded %+v", a, b)
	}
}
