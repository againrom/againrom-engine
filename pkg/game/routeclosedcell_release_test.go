package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// routeClosedCellWitness opens mission 101 and orders the party's second member
// across the map. The mission's own script lands its Wall of Earth cells a few
// ticks later, across the route the order has already stored. It returns the
// front end stopped on the first tick that holds both a wall and the route.
func routeClosedCellWitness(t *testing.T) (*FrontEnd, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	party := []mapload.PartyMember{
		{ID: "hero", PlayerCharacter: true, StartingHero: true, Hero: hero,
			Profile: data.Profile{HealthColumn: true, ManaColumn: true},
			Saved:   &mapload.Saved{Cell: mapload.Cell{X: 53, Y: 54}, HP: 1000, MaxHP: 1000, Mana: 1000, MaxMana: 1000}},
		{ID: "companion", Profile: data.Profile{HealthColumn: true}, Hero: data.Hero{Body: 60, Reaction: 60},
			Saved: &mapload.Saved{Cell: mapload.Cell{X: 54, Y: 55}, HP: 1000, MaxHP: 1000}},
	}
	app := f.App("route across a closed cell")
	app.Layout(1024, 768)
	t.Cleanup(app.StopAudio)
	if err := app.OpenMission(f.MissionOpenerWith(101, party)); err != nil {
		t.Fatal(err)
	}
	mover := f.live.mission.ids[1]
	sim.Step(f.live.world, []sim.Command{sim.MoveTo(mover, sim.CellPoint{X: 53, Y: 70})})
	for n := 0; n < 60; n++ {
		f.live.tick()
		if len(routeClosedCells(f.live.world, mover)) != 0 {
			return f, mover
		}
	}
	t.Fatalf("no wall landed across the stored route: route %v", f.live.world.Route(mover))
	return nil, 0
}

// routeClosedCells are the cells of the mover's stored route that a standing
// Wall of Earth covers.
func routeClosedCells(w *sim.World, id sim.EntityID) [][2]int32 {
	var walls [][2]int32
	for _, e := range w.CellEffects() {
		if e.Spell == 19 {
			walls = append(walls, e.Cells...)
		}
	}
	var out [][2]int32
	for _, c := range w.Route(id) {
		if slices.Contains(walls, c) {
			out = append(out, c)
		}
	}
	return out
}

// TestReleaseWallAcrossStoredRouteKeepsItAndMoverNeverEntersTheWall: the
// landing leaves the stored route as it was, and no tick puts the mover in a
// closed cell. The mission's walls seal the way, so the mover's order may end
// short; the test does not assert arrival.
func TestReleaseWallAcrossStoredRouteKeepsItAndMoverNeverEntersTheWall(t *testing.T) {
	f, mover := routeClosedCellWitness(t)
	w := f.live.world
	if len(w.Route(mover)) == 0 {
		t.Fatal("the landing discarded the stored route")
	}
	for n := 0; n < 300; n++ {
		f.live.tick()
		e, _ := f.live.entity(mover)
		for _, c := range w.CellEffects() {
			if c.Spell == 19 && slices.Contains(c.Cells, [2]int32{e.X, e.Y}) {
				t.Fatalf("tick %d: the mover stands in the closed cell (%d,%d)", n, e.X, e.Y)
			}
		}
	}
}

// TestReleaseRouteAcrossClosedCellSurvivesSaveAndColdLoad: a SAVE taken while
// the stored route crosses a closed cell loads cold into the same route, and
// the loaded mover then walks as the live one does. The native continuation
// above is the separate proof of the walk itself.
func TestReleaseRouteAcrossClosedCellSurvivesSaveAndColdLoad(t *testing.T) {
	f, mover := routeClosedCellWitness(t)
	want := f.live.world.Route(mover)
	cold := loadAreaContinuation(t, saveCorpseMission(t, f, t.TempDir()))
	if got := cold.live.world.Route(mover); !slices.Equal(got, want) {
		t.Fatalf("cold LOAD route %v, saved %v", got, want)
	}
	if len(routeClosedCells(cold.live.world, mover)) == 0 {
		t.Fatal("the loaded route crosses no closed cell")
	}
	for n := 0; n < 300; n++ {
		f.live.tick()
		cold.live.tick()
		a, _ := f.live.entity(mover)
		b, _ := cold.live.entity(mover)
		if a.X != b.X || a.Y != b.Y || a.HasTarget != b.HasTarget {
			t.Fatalf("tick %d: live mover (%d,%d) target %v, loaded (%d,%d) target %v",
				n, a.X, a.Y, a.HasTarget, b.X, b.Y, b.HasTarget)
		}
		if !slices.Equal(f.live.world.Route(mover), cold.live.world.Route(mover)) {
			t.Fatalf("tick %d: live route %v, loaded route %v", n, f.live.world.Route(mover), cold.live.world.Route(mover))
		}
	}
}
