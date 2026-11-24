package game

import (
	"image"
	"os"
	"strconv"
	"testing"

	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// openMissionForWalker opens mission n through the App and steps past its
// opening dialogue, as the other release drives do.
func openMissionForWalker(t *testing.T, n int) (*ui.App, *mapWorld) {
	t.Helper()
	app, live, err := tryOpenMissionForWalker(t, n)
	if err != nil {
		return nil, nil
	}
	return app, live
}

func tryOpenMissionForWalker(t *testing.T, n int) (*ui.App, *mapWorld, error) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	app := f.App("death step")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(n, party)); err != nil {
		return nil, nil, err
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	return app, live, nil
}

func drawOf(live *mapWorld, id sim.EntityID) (ui.MapEntity, bool) {
	for _, d := range live.entityDraws() {
		if d.ID == uint32(id) {
			return d, true
		}
	}
	return ui.MapEntity{}, false
}

// drawnOffset is the displacement in map pixels from the entity's cell
// centre that the viewer draws at the end of a tick: the fine position when
// one is carried, else the crossing's owed share of the step taken.
func drawnOffset(d ui.MapEntity) image.Point {
	if d.FinePosition {
		return image.Pt((int(d.FineX)-128)*terrain.CellSize/256, (int(d.FineY)-128)*terrain.CellSize/256)
	}
	if d.Step == (image.Point{}) {
		return image.Point{}
	}
	span := max(d.TransitSpan, 1)
	owed := min(max(d.Transit, 0), span-1)
	return image.Pt(-d.Step.X*terrain.CellSize*owed/span, -d.Step.Y*terrain.CellSize*owed/span)
}

func absPoint(p image.Point) int {
	return max(p.X, -p.X) + max(p.Y, -p.Y)
}

// walkerNotInParty ticks until a hostile or neutral actor with a crossing of
// at least minOwed ticks still to pay shows, and returns its id.
func walkerNotInParty(t *testing.T, live *mapWorld, minOwed uint16) (sim.EntityID, bool) {
	t.Helper()
	party := map[sim.EntityID]bool{}
	for _, id := range live.mission.ids {
		party[id] = true
	}
	for range 3000 {
		live.tick()
		for _, e := range live.world.Entities() {
			if party[e.ID] || !e.Alive() || e.OffMap || e.MaxHP <= 0 || e.Owner == sim.SelfSlot {
				continue
			}
			if e.Transit >= minOwed && e.Stride.Present {
				return e.ID, true
			}
		}
	}
	return 0, false
}

// TestReleaseUnitKilledMidStepFinishesTheStepBeforeItsBodyLies drives a real
// mission's walking actor, kills it between two cells and follows its drawn
// state tick by tick. The body lies on its logical cell, draws no health bar,
// slides from where it was last drawn alive by no more than one tick of its
// step per tick, and stands at the cell's centre before the death frames end.
func TestReleaseUnitKilledMidStepFinishesTheStepBeforeItsBodyLies(t *testing.T) {
	missions := []int{41, 20, 2, 3, 4, 5, 6, 7, 8, 9, 10, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 42, 43, 44}
	if v := os.Getenv("AGAINROM_DEATHSTEP_MISSION"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatal(err)
		}
		missions = []int{n}
	}
	ran := false
	for _, n := range missions {
		if ran {
			break
		}
		_, live := openMissionForWalker(t, n)
		if live == nil {
			continue
		}
		id, ok := walkerNotInParty(t, live, 3)
		if !ok {
			t.Logf("mission %d: no walking non-party actor with a long crossing in 3000 ticks", n)
			continue
		}
		ran = true
		// Let the crossing run two ticks, then fell it.
		live.tick()
		live.tick()
		before, _ := live.entity(id)
		if before.Transit == 0 || !before.Alive() {
			t.Fatalf("mission %d actor %d is not mid-step before the blow: %+v", n, id, before)
		}
		cellBefore := image.Pt(int(before.X), int(before.Y))
		alive, ok := drawOf(live, id)
		if !ok {
			t.Fatalf("no draw for %d before the blow", id)
		}
		drawnAt := func(d ui.MapEntity) image.Point { return d.Cell.Mul(terrain.CellSize).Add(drawnOffset(d)) }
		// One tick of the step, plus a pixel of rounding at each end.
		lastAt, perTick := drawnAt(alive), terrain.CellSize/max(alive.TransitSpan, 1)+2
		if err := live.world.HeadlessKill(id); err != nil {
			t.Fatal(err)
		}
		owedAtBlow := int(before.Transit)
		t.Logf("mission %d actor %d blow: cell %v transit %d/%d", n, id, cellBefore, before.Transit, before.TransitTotal)
		t.Logf("tick | cell | life | category | step | transit/span | offset | bar")
		reached, last := false, image.Point{}
		for k := 0; k <= owedAtBlow+3; k++ {
			live.tick()
			e, _ := live.entity(id)
			d, ok := drawOf(live, id)
			if !ok {
				t.Fatalf("no draw for %d", id)
			}
			off := drawnOffset(d)
			bar := d.Life != ui.LifeDead
			t.Logf("%d | %v | life %d | cat %d | step %v | %d/%d | %v | bar %v | decay %d", k, d.Cell, d.Life, d.DrawCategory, d.Step, d.Transit, d.TransitSpan, off, bar, e.Decay)
			if d.Life != ui.LifeDead {
				t.Fatalf("tick %d: the felled unit is drawn %d, not dead, so it would show a health bar", k, d.Life)
			}
			if d.Cell != cellBefore {
				t.Fatalf("tick %d: the body left its logical cell: %v, want %v", k, d.Cell, cellBefore)
			}
			if at := drawnAt(d).Sub(lastAt); max(at.X, -at.X) > perTick || max(at.Y, -at.Y) > perTick {
				t.Fatalf("tick %d: the body jumped %v in one tick, more than one tick of its step (%d pixels)", k, at, perTick)
			}
			lastAt = drawnAt(d)
			if k == 0 && off == (image.Point{}) {
				t.Fatalf("the body drew at its cell centre on the first tick after the blow; it was %d ticks from the end of its step", owedAtBlow)
			}
			if k > 0 && absPoint(off) > absPoint(last) {
				t.Fatalf("tick %d: the body moved away from its cell's centre: %v after %v", k, off, last)
			}
			if k >= owedAtBlow && off != (image.Point{}) {
				t.Fatalf("tick %d: the body is still %v short of the cell, %d ticks after a step with %d owed", k, off, k, owedAtBlow)
			}
			last = off
			if off == (image.Point{}) {
				reached = true
			}
		}
		if !reached {
			t.Fatalf("mission %d actor %d: the body never reached its cell's centre", n, id)
		}
		// A unit felled standing dies where it stands.
		standing, found := sim.EntityID(0), false
		for _, e := range live.world.Entities() {
			if e.ID != id && e.Alive() && !e.OffMap && e.MaxHP > 0 && e.Transit == 0 {
				standing, found = e.ID, true
				break
			}
		}
		if !found {
			t.Fatalf("mission %d holds no standing actor for the control", n)
		}
		if err := live.world.HeadlessKill(standing); err != nil {
			t.Fatal(err)
		}
		for k := range 4 {
			live.tick()
			d, ok := drawOf(live, standing)
			if !ok || d.Life != ui.LifeDead || d.Step != (image.Point{}) || drawnOffset(d) != (image.Point{}) {
				t.Fatalf("standing control tick %d: %+v", k, d)
			}
		}
	}
	if !ran {
		t.Skip("no mission in the install holds a long-crossing walker within the bounded drive")
	}
}
