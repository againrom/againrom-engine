package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func collectKeyMission(t *testing.T, dx int32) (*FrontEnd, *ui.App, sim.Entity, int32, int32) {
	t.Helper()
	f, app, screen, _ := secondBagFront(t, nil)
	secondBagEnter(t, f, app, screen)
	w := f.live.world
	hero := w.EntityView()[0]
	for _, e := range w.EntityView() {
		if e.Owner == sim.SelfSlot && sim.InPersistBand(e.TypeID) {
			hero = e
			break
		}
	}
	if hero.ScanRange < 2 {
		t.Fatalf("fixture scan range %d, need at least 2", hero.ScanRange)
	}
	x, y := hero.X+dx, hero.Y
	w.SetPurse(sim.SelfSlot, 100)
	f.live.pending = append(f.live.pending, sim.DropGold(sim.SelfSlot, 17, sim.CellPoint{X: x, Y: y}))
	f.live.tick()
	if sacks := w.Sacks(); len(sacks) != 1 || sacks[0].X != x || sacks[0].Y != y {
		t.Fatalf("fixture drop at (%d,%d): %+v", x, y, sacks)
	}
	if err := app.HeadlessSelectEntity(uint32(hero.ID)); err != nil {
		t.Fatal(err)
	}
	return f, app, hero, x, y
}

func TestSecondGamePickupKeySeeksASackAwayFromTheHero(t *testing.T) {
	for _, dx := range []int32{0, 2} {
		f, app, hero, x, y := collectKeyMission(t, dx)
		w := f.live.world
		before := w.Purse(sim.SelfSlot)
		if err := app.HeadlessKey("p"); err != nil {
			t.Fatal(err)
		}
		for n := 0; n < 120 && len(w.Sacks()) != 0; n++ {
			f.live.tick()
		}
		after, _ := w.Entity(hero.ID)
		if len(w.Sacks()) != 0 || w.Purse(sim.SelfSlot) != before+17 || after.X != x || after.Y != y {
			t.Fatalf("P left %d sacks, purse %d -> %d, hero (%d,%d), want take at (%d,%d)", len(w.Sacks()), before, w.Purse(sim.SelfSlot), after.X, after.Y, x, y)
		}
	}
}

func TestSecondGameCollectWalkFinishesAfterOrdinarySAVLoad(t *testing.T) {
	f, app, hero, x, y := collectKeyMission(t, 2)
	if err := app.HeadlessKey("p"); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	before, _ := f.live.world.Entity(hero.ID)
	if before.PendingOrder.Kind != sim.PendingPickup || before.PendingOrder.X != x || before.PendingOrder.Y != y || len(f.live.world.Sacks()) != 1 {
		t.Fatalf("checkpoint has no pending take at (%d,%d): %+v", x, y, before)
	}
	raw := secondPageSave(t, f, app)
	checkSecondMissionFile(t, raw, 10)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cold, _ := secondMissionLoadOf(t, secondMissionContent{side: 128}, dir, "game0001.sav")
	if cold.live == nil {
		t.Fatal("SAV did not restore a mission")
	}
	restored, ok := cold.live.world.Entity(hero.ID)
	if !ok || restored.PendingOrder.Kind != sim.PendingPickup || restored.PendingOrder.X != x || restored.PendingOrder.Y != y || len(cold.live.world.Sacks()) != 1 || cold.live.world.Purse(sim.SelfSlot) != 83 {
		t.Fatalf("SAV lost current pickup destination, sack or purse: %+v, %v", restored, ok)
	}
	for _, mw := range []*mapWorld{f.live, cold.live} {
		mw.stopped = false
		for n := 0; n < 120; n++ {
			mw.tick()
		}
		e, _ := mw.world.Entity(hero.ID)
		if len(mw.world.Sacks()) != 0 || mw.world.Purse(sim.SelfSlot) != 100 || e.X != x || e.Y != y || e.PendingOrder.Kind != sim.PendingNone {
			t.Fatalf("continued take at (%d,%d): %+v, sacks %v, purse %d", x, y, e, mw.world.Sacks(), mw.world.Purse(sim.SelfSlot))
		}
	}
}

// collectRowMission is mission 10 with two heroes and four sacks on the first
// hero's row: three two cells apart, then one past his range from the third.
func collectRowMission(t *testing.T) (f *FrontEnd, app *ui.App, hero, other sim.Entity, cells []sim.CellPoint) {
	t.Helper()
	f, app, screen, _ := secondBagFront(t, nil)
	secondBagEnter(t, f, app, screen)
	w := f.live.world
	var heroes []sim.Entity
	for _, e := range w.EntityView() {
		if e.Owner == sim.SelfSlot && sim.InPersistBand(e.TypeID) {
			heroes = append(heroes, e)
		}
	}
	if len(heroes) < 2 || heroes[0].ScanRange < 2 {
		t.Fatalf("fixture heroes %+v, need two and a scan range of at least 2", heroes)
	}
	hero, other = heroes[0], heroes[1]
	r := int32(hero.ScanRange)
	dir := int32(1)
	if hero.X+7+r >= 128 {
		dir = -1
	}
	for _, dx := range []int32{2, 4, 6, 7 + r} {
		cells = append(cells, sim.CellPoint{X: hero.X + dir*dx, Y: hero.Y})
	}
	// A gold drop lands within two cells of the player's first actor, the
	// hero: he walks onto each cell, drops, and walks back.
	w.SetPurse(sim.SelfSlot, 100)
	for k, at := range append(cells, sim.CellPoint{X: hero.X, Y: hero.Y}) {
		walkCollectFixture(t, f.live, hero.ID, at)
		if k < len(cells) {
			f.live.pending = append(f.live.pending, sim.DropGold(sim.SelfSlot, uint32(10+k), at))
			f.live.tick()
		}
	}
	if len(w.Sacks()) != 4 {
		t.Fatalf("fixture drops at %v: %+v", cells, w.Sacks())
	}
	hero, _ = w.Entity(hero.ID)
	other, _ = w.Entity(other.ID)
	return f, app, hero, other, cells
}

// walkCollectFixture walks id onto at and lets it settle there.
func walkCollectFixture(t *testing.T, mw *mapWorld, id sim.EntityID, at sim.CellPoint) {
	t.Helper()
	mw.enqueue(uint32(id), int(at.X), int(at.Y))
	tickUntil(mw, 400, func() bool {
		e, _ := mw.world.Entity(id)
		return e.X == at.X && e.Y == at.Y && !e.HasTarget && mw.world.SackPickupArrived(id)
	})
	if e, _ := mw.world.Entity(id); e.X != at.X || e.Y != at.Y {
		t.Fatalf("fixture walk to %+v ended at (%d,%d)", at, e.X, e.Y)
	}
}

// pressCollect dispatches one collect press over members as the key does.
func pressCollect(mw *mapWorld, members ...sim.Entity) {
	for k, e := range members {
		mw.grab(uint32(e.ID), k, 0, ui.GrabCollect)
	}
}

// tickUntil ticks mw until done holds or n ticks have run.
func tickUntil(mw *mapWorld, n int, done func() bool) {
	for k := 0; k < n && !done(); k++ {
		mw.tick()
	}
}

func TestSecondGameCollectKeyOrdersOneHeroWhoCollectsUntilTheSearchFails(t *testing.T) {
	f, _, hero, other, cells := collectRowMission(t)
	w := f.live.world
	pressCollect(f.live, hero, other)
	if q := f.live.pending; len(q) != 1 || q[0].Entity != hero.ID || q[0].Kind != sim.KindPickUp || q[0].X != cells[0].X {
		t.Fatalf("P queued %+v, want one collect pickup for hero %d at %+v", q, hero.ID, cells[0])
	}
	before, _ := w.Entity(other.ID)
	tickUntil(f.live, 600, func() bool {
		e, _ := w.Entity(hero.ID)
		return len(w.Sacks()) == 1 && len(f.live.pending) == 0 && e.PendingOrder.Kind == sim.PendingNone && !e.HasTarget
	})
	for k := 0; k < 4; k++ {
		f.live.tick()
	}
	e, _ := w.Entity(hero.ID)
	sacks := w.Sacks()
	if len(sacks) != 1 || sacks[0].X != cells[3].X || w.Purse(sim.SelfSlot) != 100-46+33 || e.X != cells[2].X || e.Y != cells[2].Y || e.HasTarget || e.PendingOrder.Kind != sim.PendingNone {
		t.Fatalf("collect left sacks %v purse %d, hero %+v; want the three near sacks taken and the hero standing at %+v", sacks, w.Purse(sim.SelfSlot), e, cells[2])
	}
	after, _ := w.Entity(other.ID)
	if after.X != before.X || after.Y != before.Y || after.HasTarget || after.PendingOrder != before.PendingOrder || after.ActorState != before.ActorState {
		t.Fatalf("the second selected hero changed: %+v -> %+v", before, after)
	}
}

func TestSecondGameCollectKeyOrdersTheFirstMemberWhoseSearchFindsASack(t *testing.T) {
	f, _, hero, other, cells := collectRowMission(t)
	w := f.live.world
	// An id the World does not hold is a member the admission refuses.
	pressCollect(f.live, sim.Entity{ID: 0xfffff}, hero, other)
	if q := f.live.pending; len(q) != 1 || q[0].Entity != hero.ID || q[0].Kind != sim.KindPickUp {
		t.Fatalf("P queued %+v, want the first admitted member %d", q, hero.ID)
	}
	f.live.pending = f.live.pending[:0]
	// The hero stands on the third sack, the only one left, out of the
	// other's reach: the other, first in the press, is passed over.
	keep, ro := cells[2], int32(other.ScanRange)
	if d := keep.X - other.X; d <= ro && -d <= ro && keep.Y-other.Y <= ro && other.Y-keep.Y <= ro {
		t.Fatalf("fixture: sack %+v is in the other hero %+v's reach", keep, other)
	}
	walkCollectFixture(t, f.live, hero.ID, keep)
	for _, c := range []sim.CellPoint{cells[0], cells[1], cells[3]} {
		if err := w.TakeSack(other.ID, c.X, c.Y); err != nil {
			t.Fatal(err)
		}
	}
	if e, _ := w.Entity(other.ID); e.X != other.X || e.Y != other.Y {
		t.Fatalf("fixture: the other hero moved to (%d,%d)", e.X, e.Y)
	}
	pressCollect(f.live, other, hero)
	if q := f.live.pending; len(q) != 1 || q[0].Entity != hero.ID || q[0].X != keep.X || q[0].Y != keep.Y {
		t.Fatalf("P queued %+v, want the hero %d's pickup at %+v", q, hero.ID, keep)
	}
	f.live.pending = f.live.pending[:0]
	// Nobody's search finds a sack: nobody's order changes.
	if err := w.TakeSack(other.ID, keep.X, keep.Y); err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	pressCollect(f.live, other, hero)
	if q := f.live.pending; len(q) != 0 || w.Hash() != before {
		t.Fatalf("P with no sack in reach queued %+v", q)
	}
}

func TestSecondGameCollectStopsAtAMoveOrder(t *testing.T) {
	f, _, hero, other, cells := collectRowMission(t)
	w := f.live.world
	pressCollect(f.live, hero, other)
	tickUntil(f.live, 300, func() bool {
		e, _ := w.Entity(hero.ID)
		return len(w.Sacks()) == 3 && e.PendingOrder.Collect && e.PendingOrder.X == cells[1].X
	})
	if e, _ := w.Entity(hero.ID); !e.PendingOrder.Collect || e.PendingOrder.X != cells[1].X {
		t.Fatalf("no second collect walk after the first take: %+v", e.PendingOrder)
	}
	f.live.enqueue(uint32(hero.ID), int(hero.X), int(hero.Y+2))
	for k := 0; k < 300; k++ {
		f.live.tick()
	}
	e, _ := w.Entity(hero.ID)
	if len(w.Sacks()) != 3 || e.X != hero.X || e.Y != hero.Y+2 || e.PendingOrder.Kind != sim.PendingNone {
		t.Fatalf("move after the first take: sacks %v hero %+v, want sacks from %+v kept", w.Sacks(), e, cells[1])
	}
}

// loadCollectSAV writes raw as a mission file and loads it into a new front end.
func loadCollectSAV(t *testing.T, raw []byte) (*FrontEnd, *ui.App) {
	t.Helper()
	checkSecondMissionFile(t, raw, 10)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := secondMissionLoadOf(t, secondMissionContent{side: 128}, dir, "game0001.sav")
	if f.live == nil {
		t.Fatal("SAV did not restore a mission")
	}
	f.live.stopped = false
	return f, app
}

// Both runs start from one loaded file: a LOAD adds a runtime-id form. After
// a take its orphan list differs after LOAD as for a plain pickup, so that
// case compares actors, sacks and purse instead of the hash.
func TestSecondGameCollectContinuesAfterSAVLoad(t *testing.T) {
	f, app, hero, other, cells := collectRowMission(t)
	start := secondPageSave(t, f, app)
	for _, afterTake := range []bool{false, true} {
		var runs [2]*mapWorld
		for k := range runs {
			g, gapp := loadCollectSAV(t, start)
			w := g.live.world
			pressCollect(g.live, hero, other)
			first := w.Tick() + 3
			tickUntil(g.live, 300, func() bool {
				e, _ := w.Entity(hero.ID)
				if !afterTake {
					return w.Tick() >= first
				}
				return len(w.Sacks()) == 3 && e.PendingOrder.Collect && e.PendingOrder.X == cells[1].X
			})
			if e, _ := w.Entity(hero.ID); !e.PendingOrder.Collect || len(g.live.pending) != 0 {
				t.Fatalf("checkpoint is not mid-collect: %+v queue %+v", e.PendingOrder, g.live.pending)
			}
			runs[k] = g.live
			if k == 1 {
				saved := w.Hash()
				raw := secondPageSave(t, g, gapp)
				if w.Hash() != saved || runs[0].world.Hash() != saved {
					t.Fatal("SAVE changed the World, or the two runs differ before it")
				}
				cold, _ := loadCollectSAV(t, raw)
				if e, _ := cold.live.world.Entity(hero.ID); !e.PendingOrder.Collect || !afterTake && cold.live.world.Hash() != saved {
					t.Fatalf("LOAD: hero %+v, hash equal %v", e.PendingOrder, cold.live.world.Hash() == saved)
				}
				runs[k] = cold.live
			}
		}
		for k := 0; k < 600; k++ {
			runs[0].tick()
			runs[1].tick()
			a, b := runs[0].world, runs[1].world
			if !afterTake && a.Hash() != b.Hash() || !reflect.DeepEqual(a.EntityView(), b.EntityView()) || !reflect.DeepEqual(a.Sacks(), b.Sacks()) || a.Purse(sim.SelfSlot) != b.Purse(sim.SelfSlot) {
				t.Fatalf("after take %v, tick %d after LOAD: the loaded run left the unsaved one", afterTake, k)
			}
		}
		e, _ := runs[1].world.Entity(hero.ID)
		if sacks := runs[1].world.Sacks(); len(sacks) != 1 || e.X != cells[2].X || e.PendingOrder.Kind != sim.PendingNone {
			t.Fatalf("loaded collect: sacks %v hero %+v", sacks, e)
		}
	}
}

func TestSecondGameCollectWithAnEmptySearchKeepsTheMemberOrder(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, nil)
	secondBagEnter(t, f, app, screen)
	hero := f.live.world.EntityView()[0]
	if err := app.HeadlessSelectEntity(uint32(hero.ID)); err != nil {
		t.Fatal(err)
	}
	f.live.grab(uint32(hero.ID), 0, 0, ui.GrabCollect)
	f.live.enqueue(uint32(hero.ID), int(hero.X+2), int(hero.Y))
	f.live.grab(uint32(hero.ID), 0, 0, ui.GrabCollect)
	queue := f.live.pending
	if len(queue) != 1 || queue[0].Kind == sim.KindPickUp || queue[0].Kind == sim.KindGroupStance {
		t.Fatalf("empty collect presses queued %+v, want only the Move", queue)
	}
	f.live.tick()
	if e, _ := f.live.world.Entity(hero.ID); !e.HasTarget {
		t.Fatalf("an empty collect replaced the Move: %+v", e)
	}
}
