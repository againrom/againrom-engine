package game

import (
	"testing"

	"againrom/pkg/sim"
)

const headlessReachLimit = 400

// liveWalk walks a player's actor toward (x, y) on the right-click move order
// and ordinary live ticks until it arrives or the order is spent.
func liveWalk(t *testing.T, mw *mapWorld, id sim.EntityID, x, y int32) {
	t.Helper()
	mw.enqueue(uint32(id), int(x), int(y))
	for n := 0; ; n++ {
		mw.tick()
		e, ok := mw.entity(id)
		if !ok || sackPickupArrived(mw.world, e) && (e.X == x && e.Y == y || !e.HasTarget) {
			return
		}
		if n >= headlessReachLimit {
			t.Fatalf("actor %d still walking at (%d,%d) toward (%d,%d)", id, e.X, e.Y, x, y)
		}
	}
}

// livePlaceAndWalk is HeadlessPlace near (x, y), then liveWalk.
func livePlaceAndWalk(t *testing.T, mw *mapWorld, id sim.EntityID, x, y int32) {
	t.Helper()
	if err := mw.world.HeadlessPlace(id, x, y); err != nil {
		t.Fatal(err)
	}
	if e, _ := mw.entity(id); e.X != x || e.Y != y {
		liveWalk(t, mw, id, x, y)
	}
	waitHeadlessCrossing(t, mw.world, id, func() { mw.tick() })
}

// finishLoadedCycle advances ordinary live ticks until the actor's attack cycle
// is at ready. A move order waits behind a loaded cycle (DIV-1563), so a test
// that relocates an actor mid-fight and walks it calls this first, and the order
// that follows is taken on the tick it is given.
func finishLoadedCycle(t *testing.T, mw *mapWorld, id sim.EntityID) {
	t.Helper()
	for n := 0; ; n++ {
		e, ok := mw.entity(id)
		if !ok || e.AttackPhase == sim.AttackReady {
			return
		}
		if n >= 64 {
			t.Fatalf("actor %d attack cycle still at phase %d after %d ticks", id, e.AttackPhase, n)
		}
		mw.tick()
	}
}

// worldPlaceAndWalk is livePlaceAndWalk over bare Steps.
func worldPlaceAndWalk(t *testing.T, w *sim.World, id sim.EntityID, x, y int32) {
	t.Helper()
	if err := w.HeadlessPlace(id, x, y); err != nil {
		t.Fatal(err)
	}
	worldWalk(t, w, id, x, y)
}

// worldWalk is liveWalk over bare Steps.
func worldWalk(t *testing.T, w *sim.World, id sim.EntityID, x, y int32) {
	t.Helper()
	if e, _ := w.Entity(id); e.X == x && e.Y == y {
		waitHeadlessCrossing(t, w, id, func() { sim.Step(w, nil) })
		return
	}
	order := []sim.Command{sim.MoveTo(id, sim.CellPoint{X: x, Y: y})}
	for n := 0; ; n++ {
		sim.Step(w, order)
		order = nil
		e, ok := w.Entity(id)
		if !ok || sackPickupArrived(w, e) && (e.X == x && e.Y == y || !e.HasTarget) {
			return
		}
		if n >= headlessReachLimit {
			t.Fatalf("actor %d still walking at (%d,%d) toward (%d,%d)", id, e.X, e.Y, x, y)
		}
	}
}

// liveTakeAt brings the actor onto (x, y) with livePlaceAndWalk and takes the
// sack underfoot on the pickup key's world route.
func liveTakeAt(t *testing.T, mw *mapWorld, id sim.EntityID, x, y int32) sim.Sack {
	t.Helper()
	if e, _ := mw.entity(id); e.X != x || e.Y != y {
		livePlaceAndWalk(t, mw, id, x, y)
	}
	waitHeadlessCrossing(t, mw.world, id, func() { mw.tick() })
	primary, ok := mw.primaryPartyID()
	return takeUnderfoot(t, mw.world, id, x, y, primary, ok)
}

// worldTakeAt is liveTakeAt over bare Steps on a mission without a map.
func worldTakeAt(t *testing.T, ms *Mission, id sim.EntityID, x, y int32) sim.Sack {
	t.Helper()
	if e, _ := ms.World.Entity(id); e.X != x || e.Y != y {
		worldPlaceAndWalk(t, ms.World, id, x, y)
	}
	waitHeadlessCrossing(t, ms.World, id, func() { sim.Step(ms.World, nil) })
	if e, _ := ms.World.Entity(id); e.X != x || e.Y != y {
		t.Fatalf("actor %d stopped at (%d,%d), short of the sack at (%d,%d)", id, e.X, e.Y, x, y)
	}
	sack, ok := ms.PickUpUnderfoot(id)
	if !ok {
		t.Fatalf("actor %d took no sack at (%d,%d)", id, x, y)
	}
	return sack
}

func waitHeadlessCrossing(t *testing.T, w *sim.World, id sim.EntityID, step func()) {
	t.Helper()
	for n := 0; ; n++ {
		e, ok := w.Entity(id)
		if !ok || sackPickupArrived(w, e) {
			return
		}
		if n >= headlessReachLimit {
			x, y, fine := w.ActorFinePosition(id)
			t.Fatalf("actor %d did not finish its crossing: transit=%d fine=(%d,%d), present=%t", id, e.Transit, x, y, fine)
		}
		step()
	}
}

func takeUnderfoot(t *testing.T, w *sim.World, id sim.EntityID, x, y int32, primary sim.EntityID, primaryOK bool) sim.Sack {
	t.Helper()
	if e, _ := w.Entity(id); e.X != x || e.Y != y {
		t.Fatalf("actor %d stopped at (%d,%d), short of the sack at (%d,%d)", id, e.X, e.Y, x, y)
	}
	sack, ok := takeSackUnderfoot(w, id, primary, primaryOK)
	if !ok {
		t.Fatalf("actor %d took no sack at (%d,%d)", id, x, y)
	}
	return sack
}

// headlessDamage is the script's health write through sim.HeadlessDamage.
func headlessDamage(t *testing.T, w *sim.World, id sim.EntityID, amount int32) {
	t.Helper()
	if err := w.HeadlessDamage(id, amount); err != nil {
		t.Fatal(err)
	}
}

// headlessFell is the script's health write to -1, a damage of the current
// health plus one; a body already dead is left alone.
func headlessFell(t *testing.T, w *sim.World, id sim.EntityID) {
	t.Helper()
	if e, ok := w.Entity(id); ok && !e.Dead() {
		headlessDamage(t, w, id, e.HP+1)
	}
}
