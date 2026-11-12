package game

import (
	"os"
	"path/filepath"
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

func TestSecondGameCollectEmptyOrdersKeepTheirQueuePosition(t *testing.T) {
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
	if len(queue) != 3 || queue[0].Kind != sim.KindGroupStance || queue[2].Kind != sim.KindGroupStance || queue[0].Group == queue[2].Group {
		t.Fatalf("distinct collect presses merged across Move: %+v", queue)
	}
	f.live.tick()
	e, _ := f.live.world.Entity(hero.ID)
	if e.HasTarget || e.ActorState != 0xc {
		t.Fatalf("the last empty collect lost to Move: %+v", e)
	}
}
