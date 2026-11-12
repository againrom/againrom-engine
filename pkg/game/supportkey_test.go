package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

// supportOrderOf answers id's member order and its Group's programme byte.
func supportOrderOf(w *sim.World, id sim.EntityID) (state uint32, programme uint8) {
	groups, orders, _ := w.SavedGroups()
	for _, o := range orders {
		if o.Entity == id {
			state = o.State
		}
	}
	for _, g := range groups {
		for _, m := range g.Members {
			if m.Bound && m.Entity == id {
				programme = g.AI[0x20]
			}
		}
	}
	return state, programme
}

func TestSecondGameLKeyOrderSurvivesOrdinarySAV(t *testing.T) {
	f, app, hero, _, _ := collectKeyMission(t, 2)
	if err := app.HeadlessKey("l"); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	before, _ := f.live.world.Entity(hero.ID)
	if state, programme := supportOrderOf(f.live.world, hero.ID); before.ActorState != 0x17 || state != 0x17 || programme != 0x13 {
		t.Fatalf("L left state %#x, order %#x, programme %#x", before.ActorState, state, programme)
	}
	raw := secondPageSave(t, f, app)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold, _ := secondMissionLoadOf(t, secondMissionContent{side: 128}, dir, "game0001.sav")
	if cold.live == nil {
		t.Fatal("SAV did not restore a mission")
	}
	cold.live.stopped = false
	for n := 0; n < 20; n++ {
		cold.live.tick()
	}
	after, ok := cold.live.world.Entity(hero.ID)
	state, programme := supportOrderOf(cold.live.world, hero.ID)
	if !ok || after.ActorState != 0x17 || state != 0x17 || programme != 0x13 || after.X != before.X || after.Y != before.Y {
		t.Fatalf("loaded L order: state %#x, order %#x, programme %#x, at (%d,%d) want (%d,%d)", after.ActorState, state, programme, after.X, after.Y, before.X, before.Y)
	}
}
