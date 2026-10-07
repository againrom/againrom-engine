package game

import (
	"fmt"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The far search of a mover larger than one cell spends StaticScanAhead + D
// generations whoever owns it, D being the Chebyshev distance from its cell to
// its goal, where a one-cell unit of the participant spends a flat thousand
// (MOVE-TERM-003, MOVE-PARAM-006, MOVE-SPEED-011). The party's hired Catapult and
// Ballista are Units rows with a footprint of two cells owned by the participant.
//
// A wave of B generations labels only cells within B steps. The route read back
// from its labels can be a cell or two longer than B when a cheaper way round a
// costly cell takes more steps, so every bound below allows releaseLargeSlack
// cells more. The goal lies across an obstacle that takes 34 more steps to walk
// round than the budget allows, so the two rules give routes of very different
// lengths.
const (
	releaseHiredMission                  = 121
	releaseHiredGoalX, releaseHiredGoalY = 21, 14
	releaseHiredHorizon                  = 40
)

// releaseSiegeChapter is the first main chapter whose tavern lists both siege
// mercenaries after earlier chapters have unlocked them, each with a stocked squad.
func releaseSiegeChapter(t *testing.T, c Campaign) int {
	t.Helper()
	for _, target := range c.Main {
		if !c.offers(target) {
			continue
		}
		unlocked := make(map[int]bool)
		for mission, ch := range c.Chapters {
			if mission >= target {
				continue
			}
			for _, typ := range ch.EnableMercenary {
				unlocked[typ] = true
			}
		}
		listed := make(map[int]bool)
		for _, typ := range c.Chapters[target].Mercenaries {
			listed[typ] = true
		}
		both := true
		for typ := 1; typ <= 2; typ++ {
			both = both && listed[typ] && unlocked[typ] && typ <= len(c.MercenaryCount) && c.MercenaryCount[typ-1] > 0
		}
		if both {
			return target
		}
	}
	t.Fatal("campaign has no chapter whose tavern lists both siege mercenaries")
	return 0
}

// releaseHireSiege hires the Catapult and the Ballista through the App's tavern:
// the town is loaded from a save, the tavern door is chosen, and each squad is
// selected and hired with the tavern's own controls. It leaves the party in
// f.Carried.
func releaseHireSiege(t *testing.T, f *FrontEnd) {
	t.Helper()
	target := releaseSiegeChapter(t, f.Campaign.Value())
	town := NewTown(f.Campaign.Value())
	for mission := range f.Campaign.Value().Chapters {
		if mission < target {
			town.Won(mission)
		}
	}
	if got := town.Chapter(); got != target {
		t.Fatalf("prepared chapter = %d, want %d", got, target)
	}
	town.gold = 1_000_000
	f.Town = town
	f.Carried = f.NextParty()
	f.arriveInTown()
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Unix(100, 0), payload); err != nil {
		t.Fatal(err)
	}
	app := f.App("hired siege tavern")
	t.Cleanup(app.StopAudio)
	app.Layout(640, 480)
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
	for _, target := range []string{"load game", "@first", "TAVERN"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if f.Town.Gold() < 1_000_000 {
		t.Fatalf("the loaded town holds %d gold, want the purse the fixture wrote", f.Town.Gold())
	}
	for typ := 1; typ <= 2; typ++ {
		if err := app.HeadlessActivate(fmt.Sprintf("Mercenary %d", typ)); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessActivate(f.Words.TavernHire); err != nil {
			t.Fatal(err)
		}
		if !f.Town.MercenaryHired(typ) {
			t.Fatalf("mercenary %d is not hired after the tavern's Hire control", typ)
		}
	}
	names := map[uint8]string{1: "Catapult", 2: "Ballista"}
	hired := 0
	for _, m := range f.Carried {
		if m.MercenaryType != 0 && m.Name == names[m.MercenaryType] {
			hired++
		}
	}
	if hired != 2 {
		t.Fatalf("the party holds %d hired siege members, want a Catapult and a Ballista", hired)
	}
}

// releaseSiegeTypeID is the TypeID of the installed Units row named name.
func releaseSiegeTypeID(t *testing.T, f *FrontEnd, name string) int32 {
	t.Helper()
	for i := 1; i < f.Table.Units.Len(); i++ {
		if f.Table.Units.EntryName(i) != name {
			continue
		}
		d, err := data.NewUnitDef(name, f.Table.Units.EntryParams(i))
		if err != nil {
			t.Fatal(err)
		}
		return d.TypeID
	}
	t.Fatalf("no Units row named %s", name)
	return 0
}

// releaseGoalPixel is a window pixel that the ground click resolves to the cell
// (gx, gy), found around the centre of the view once the camera is centred on that
// cell. The oracle is the production drop-cell resolution.
func releaseGoalPixel(t *testing.T, app *ui.App, gx, gy int32) (int, int) {
	t.Helper()
	var xs, ys, n int
	for y := 100; y < 520; y += 2 {
		for x := 200; x < 830; x += 2 {
			cx, cy, err := app.HeadlessDropCell(x, y)
			if err != nil {
				t.Fatal(err)
			}
			if int32(cx) == gx && int32(cy) == gy {
				xs, ys, n = xs+x, ys+y, n+1
			}
		}
	}
	if n == 0 {
		t.Fatalf("no window pixel resolves to cell (%d,%d) with the camera centred on it", gx, gy)
	}
	return xs / n, ys / n
}

// releaseHiredOrder opens the mission with the party in f.Carried and orders the
// unit that pick returns to the goal cell through the App's own input: the unit is
// selected, and the goal cell is clicked. It returns the live world and the unit as
// it stood when the order was given.
func releaseHiredOrder(t *testing.T, f *FrontEnd, pick func(*mapWorld) (sim.Entity, bool), goal [2]int32) (*mapWorld, sim.Entity) {
	t.Helper()
	app := f.App("hired far search")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpenerWith(releaseHiredMission, f.Carried)); err != nil {
		t.Fatal(err)
	}
	live := f.live
	for k := 0; k < 16 && live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	mover, ok := pick(live)
	if !ok || mover.OffMap || !mover.Alive() {
		t.Fatalf("mission %d holds no such unit of the party", releaseHiredMission)
	}
	inspectionCentre(live, int(mover.X), int(mover.Y))
	if err := app.HeadlessSelectEntity(uint32(mover.ID)); err != nil {
		t.Fatal(err)
	}
	inspectionCentre(live, int(goal[0]), int(goal[1]))
	x, y := releaseGoalPixel(t, app, goal[0], goal[1])
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	if len(live.pending) != 1 || live.pending[0].Kind != sim.KindGroupMoveTo || live.pending[0].Entity != mover.ID ||
		live.pending[0].X != goal[0] || live.pending[0].Y != goal[1] {
		t.Fatalf("the click queued %+v, want unit %d's move to (%d,%d)", live.pending, mover.ID, goal[0], goal[1])
	}
	return live, mover
}

// releaseHiredRoute ticks the world until the unit holds a route and returns the
// route and the unit as it stands then.
func releaseHiredRoute(t *testing.T, live *mapWorld, id sim.EntityID) ([][2]int32, sim.Entity) {
	t.Helper()
	for n := 0; n < 8; n++ {
		live.tick()
		if route := live.world.Route(id); len(route) > 0 {
			e, _ := live.entity(id)
			return route, e
		}
	}
	t.Fatalf("unit %d held no route eight ticks after its order", id)
	return nil, sim.Entity{}
}

// TestReleaseTheHiredSiegeEnginesFarSearchSpendsTheNxNArm hires the party's Catapult
// and Ballista in the tavern of the chapter that stocks both, enters mission 121,
// which that chapter's school offers, and orders each unit by selecting it and
// clicking a cell of the map across an obstacle. The hero, a one-cell unit of the
// participant, takes the order first: his route walks the whole detour onto the cell,
// longer than any budget of StaticScanAhead + D, so the cell separates the two
// rules. Each siege engine then holds a route no longer than its budget plus
// releaseLargeSlack that stops short of the cell, and its order stands on the cell
// its search settled on; before, it held the whole detour onto the cell.
func TestReleaseTheHiredSiegeEnginesFarSearchSpendsTheNxNArm(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	releaseHireSiege(t, f)
	goal := [2]int32{releaseHiredGoalX, releaseHiredGoalY}

	t.Run("hero", func(t *testing.T) {
		live, hero := releaseHiredOrder(t, f, func(l *mapWorld) (sim.Entity, bool) { return l.entity(l.mission.ids[0]) }, goal)
		route, _ := releaseHiredRoute(t, live, hero.ID)
		d := releaseFarDistance([2]int32{hero.X, hero.Y}, goal)
		t.Logf("hero %d at (%d,%d), %d cells from (%d,%d): a %d-cell route ending at %v, World hash %016x",
			hero.ID, hero.X, hero.Y, d, goal[0], goal[1], len(route), route[len(route)-1], live.world.Hash())
		if route[len(route)-1] != goal || len(route) <= int(releaseLargeBudget(d))+releaseLargeSlack {
			t.Fatalf("the hero's route is %d cells ending at %v, want the whole detour onto (%d,%d), longer than the %d generations of a unit larger than one cell: the drive does not discriminate",
				len(route), route[len(route)-1], goal[0], goal[1], releaseLargeBudget(d))
		}
	})

	for _, name := range []string{"Catapult", "Ballista"} {
		t.Run(name, func(t *testing.T) {
			typeID := releaseSiegeTypeID(t, f, name)
			live, unit := releaseHiredOrder(t, f, func(l *mapWorld) (sim.Entity, bool) {
				for _, e := range l.world.Entities() {
					if e.Owner == sim.SelfSlot && e.TypeID == typeID && e.TokenSize > 1 {
						return e, true
					}
				}
				return sim.Entity{}, false
			}, goal)
			if unit.TokenSize != 2 {
				t.Fatalf("the hired %s has a footprint of %d cells, want 2", name, unit.TokenSize)
			}
			route, held := releaseHiredRoute(t, live, unit.ID)
			end := route[len(route)-1]
			d := releaseFarDistance([2]int32{unit.X, unit.Y}, goal)
			t.Logf("%s %d at (%d,%d), %d cells from (%d,%d): a %d-cell route ending at %v, a budget of %d, World hash %016x",
				name, unit.ID, unit.X, unit.Y, d, goal[0], goal[1], len(route), end, releaseLargeBudget(d), live.world.Hash())
			if end == goal || len(route) > int(releaseLargeBudget(d))+releaseLargeSlack {
				t.Errorf("the hired %s holds a %d-cell route ending at %v, want one that stops short of (%d,%d) and is no longer than the %d generations of a unit larger than one cell",
					name, len(route), end, goal[0], goal[1], releaseLargeBudget(d))
			}
			if held.TargetX == goal[0] && held.TargetY == goal[1] {
				t.Errorf("the hired %s's order still names (%d,%d): its search did not settle on a nearer cell", name, goal[0], goal[1])
			}
			target := [2]int32{held.TargetX, held.TargetY}
			for n := 0; n < releaseHiredHorizon; n++ {
				live.tick()
				e, _ := live.entity(unit.ID)
				if e.X == goal[0] && e.Y == goal[1] {
					t.Fatalf("the hired %s stood on (%d,%d) at tick %d", name, goal[0], goal[1], live.world.Tick())
				}
				if e.HasTarget && [2]int32{e.TargetX, e.TargetY} != target {
					t.Fatalf("the hired %s's order moved from %v to (%d,%d) at tick %d", name, target, e.TargetX, e.TargetY, live.world.Tick())
				}
			}
		})
	}
}
