package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The two cells the cases below use.
const (
	poHeroID           = 7
	poHeroX, poHeroY   = 3, 3
	poSackX, poSackY   = 8, 3
	poEmptyX, poEmptyY = 20, 20
)

// poWorld is one hero at (poHeroX, poHeroY) and one sack at (poSackX, poSackY),
// with the sack carrying one code so the transfer is observable at the
// container.
func poWorld(t *testing.T) (*sim.World, *mapWorld) {
	t.Helper()
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: poHeroID, X: poHeroX, Y: poHeroY}}, nil, sim.Relations{},
		[]sim.Sack{{X: poSackX, Y: poSackY, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	return w, grabWorld(t, w, poHeroID, missionSource{})
}

// poRun ticks until the hero stands on the sack's cell or the bound is spent,
// and returns the tick it arrived on. The bound is generous and is not the
// subject of any assertion: what is measured is the world after it.
func poRun(mw *mapWorld, w *sim.World) int {
	for i := 1; i <= 400; i++ {
		mw.tick()
		if e, ok := mw.entity(poHeroID); ok && e.X == poSackX && e.Y == poSackY {
			return i
		}
	}
	return -1
}

// TestThePickupOrderWalksTheOrderedUnitToTheSackAndTakesItOnArrival is
// `ITEM-PICK-016`'s own arm, end to end.
//
// That claim reads order-space opcode `0x21` whole: it looks a sack up at an
// ARBITRARY cell, refuses when none stands there, then writes `actor+0x50 = 2`
// and `ord+0x0a = (row<<8)|col`. `AI-STATE-011` arm 2 compares `ord+0x0a`
// against the actor's own cell every tick -- not there walks, there sets
// `ord+0x08 = 7` -- and the take runs on the following tick. The owner's own
// account of the behaviour is quoted in that row.
//
// THE ASSERTIONS ARE ORDERED AS THE DEFECT WAS REPORTED: the unit must move,
// and then the sack must be taken. Round 3 failed both.
func TestThePickupOrderWalksTheOrderedUnitToTheSackAndTakesItOnArrival(t *testing.T) {
	w, mw := poWorld(t)

	mw.grab(poHeroID, poSackX, poSackY, true)

	// NOTHING HAS MOVED YET. The order is a standing order, not an immediate
	// transfer: the sack is still on the ground and the container is empty
	// until the walk finishes.
	if got := len(w.Sacks()); got != 1 {
		t.Fatalf("after the order and before any tick, Sacks() = %d, want 1 still standing", got)
	}
	if codes, _ := w.Carried(poHeroID); len(codes) != 0 {
		t.Fatalf("after the order and before any tick, Carried = %v, want none", codes)
	}

	arrived := poRun(mw, w)
	if arrived < 0 {
		e, _ := mw.entity(poHeroID)
		t.Fatalf("the ordered unit never reached (%d,%d); it is at (%d,%d) -- the pick-up order issued no walk",
			poSackX, poSackY, e.X, e.Y)
	}

	if got := len(w.Sacks()); got != 0 {
		t.Errorf("Sacks() = %d after arrival, want 0 -- the sack must be taken on arrival", got)
	}
	codes, ok := w.Carried(poHeroID)
	if !ok || len(codes) != 1 || codes[0] != eqSwordCode {
		t.Errorf("Carried(%d) = %v,%v after arrival, want the sack's one code", poHeroID, codes, ok)
	}
	if mw.pickup.set {
		t.Error("the standing order is still armed after it completed")
	}
}

// TestThePickupOrderRefusesACellWithNoSackOnIt is the original's own refusal.
// `ITEM-PICK-016` reads the arm looking the named cell up in the world sack
// registry and refusing with the engine's "Sack not found at " string when it
// answers nothing. So no walk is started and nothing is armed: the unit is not
// sent across the map for a sack that is not there.
func TestThePickupOrderRefusesACellWithNoSackOnIt(t *testing.T) {
	w, mw := poWorld(t)

	mw.grab(poHeroID, poEmptyX, poEmptyY, true)

	if mw.pickup.set {
		t.Error("a pick-up order naming a cell with no sack armed a standing order")
	}
	if got := len(mw.pending); got != 0 {
		t.Errorf("pending = %d command(s), want 0 -- a refused pick-up starts no walk", got)
	}
	e, _ := mw.entity(poHeroID)
	if e.X != poHeroX || e.Y != poHeroY {
		t.Errorf("the unit is at (%d,%d), want (%d,%d) -- it must not have been sent anywhere", e.X, e.Y, poHeroX, poHeroY)
	}
	_ = w
}

// TestALaterOrderCancelsAStandingPickup is the driver-side half of what the
// original gets for free. `0x21` writes `actor+0x50 = 2` and every other order
// arm writes that same field (`AI-CMD-032`, `AI-STATE-011`), so a second order
// overwrites the pick-up rather than queuing behind it. Here the state is on
// the driver, so the overwrite is performed by cancelPickup from queueGroup.
//
// THE UNIT IS WALKED TO THE SACK'S CELL ANYWAY, by a plain move order to the
// same cell, so the cancel is measured against a unit that DOES arrive. Without
// the cancel it would arrive and take the sack, which is the wrong behaviour
// this asserts against.
func TestALaterOrderCancelsAStandingPickup(t *testing.T) {
	w, mw := poWorld(t)

	mw.grab(poHeroID, poSackX, poSackY, true)
	if !mw.pickup.set {
		t.Fatal("setup: the standing order did not arm")
	}
	mw.enqueue(poHeroID, poSackX, poSackY)
	if mw.pickup.set {
		t.Fatal("a later order left the standing pick-up armed")
	}

	if arrived := poRun(mw, w); arrived < 0 {
		t.Fatal("setup: the unit never reached the sack's cell under the plain move order")
	}
	if got := len(w.Sacks()); got != 1 {
		t.Errorf("Sacks() = %d, want the sack still standing -- a cancelled pick-up must not fire on arrival", got)
	}
	if codes, _ := w.Carried(poHeroID); len(codes) != 0 {
		t.Errorf("Carried(%d) = %v, want none", poHeroID, codes)
	}
}

// TestAStandingPickupDisarmsWhenTheSackGoesAway is settlePickup's second exit.
// The original refuses the same case one tick later rather than at issue: its
// arm looks the cell up in the registry, so a sack taken by someone else
// between the order and the arrival leaves nothing to find.
//
// THE SACK IS REMOVED BY THE ORDINARY PRIMITIVE, TakeSack itself, called for a
// second entity that starts on the sack's cell -- not by reaching into the
// world's registry, so the state the latch then reads is the state a real
// competing pick-up would have left.
func TestAStandingPickupDisarmsWhenTheSackGoesAway(t *testing.T) {
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: poHeroID, X: poHeroX, Y: poHeroY}, {ID: poHeroID + 1, X: poSackX, Y: poSackY}},
		nil, sim.Relations{},
		[]sim.Sack{{X: poSackX, Y: poSackY, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	mw := grabWorld(t, w, poHeroID, missionSource{})

	mw.grab(poHeroID, poSackX, poSackY, true)
	if !mw.pickup.set {
		t.Fatal("setup: the standing order did not arm")
	}
	if err := w.TakeSack(poHeroID+1, poSackX, poSackY); err != nil {
		t.Fatalf("setup: the competing take failed: %v", err)
	}

	mw.tick()

	if mw.pickup.set {
		t.Error("the standing order survived a tick with no sack left at its cell")
	}
	if codes, _ := w.Carried(poHeroID); len(codes) != 0 {
		t.Errorf("Carried(%d) = %v, want none -- the ordered unit never reached the cell", poHeroID, codes)
	}
}

// TestThePickupOrderTakesForTheORDEREDUnitAndNotTheInventorySubject is
// `DIV-292`'s own close, and it is the case a fixture whose subject and whose
// ordered unit are the same entity cannot see.
//
// `AI-CURSOR-242` gates the `pickup` cursor on exactly one SELECTED object, so
// the cursor names one unit; `AI-CLICK-050` gives the click under it opcode
// `0x21`, which `ITEM-PICK-016` reads as acting for the ORDERED actor. Through
// round 3 MapGrab took no entity at all and the far side answered the "which
// character" question once, for the inventory window's subject. Here the two
// are deliberately different entities.
func TestThePickupOrderTakesForTheORDEREDUnitAndNotTheInventorySubject(t *testing.T) {
	const subjectID, orderedID = 7, 8
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{{ID: subjectID, X: poHeroX, Y: poHeroY}, {ID: orderedID, X: poHeroX + 1, Y: poHeroY}},
		nil, sim.Relations{},
		[]sim.Sack{{X: poSackX, Y: poSackY, Items: []uint16{eqSwordCode}}})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	m := worldFixtureMap()
	ms := &Mission{Number: 1, Map: m, World: w,
		Party: []mapload.PartyMember{{PlayerCharacter: true, StartingHero: true}, {PlayerCharacter: true}},
		Start: mapload.Start{IDs: []sim.EntityID{subjectID, orderedID}}}
	mw := openMission(ms, nil, nil, worldFixtureViewer(t, m), missionSource{}, nil, nil)

	if !mw.invSubjectSet || mw.invSubject.ID != subjectID {
		t.Fatalf("setup: inventory subject = %v,%v, want %d — this case needs the two to differ",
			mw.invSubject.ID, mw.invSubjectSet, subjectID)
	}

	mw.grab(orderedID, poSackX, poSackY, true)
	arrived := -1
	for i := 1; i <= 400; i++ {
		mw.tick()
		if e, ok := mw.entity(orderedID); ok && e.X == poSackX && e.Y == poSackY {
			arrived = i
			break
		}
	}
	if arrived < 0 {
		t.Fatal("setup: the ordered unit never reached the sack's cell")
	}

	codes, ok := w.Carried(orderedID)
	if !ok || len(codes) != 1 || codes[0] != eqSwordCode {
		t.Errorf("Carried(%d) = %v,%v, want the sack's one code — the take must act for the ORDERED unit",
			orderedID, codes, ok)
	}
	if got, _ := w.Carried(subjectID); len(got) != 0 {
		t.Errorf("Carried(%d) = %v, want none — the inventory window's subject did not take this sack",
			subjectID, got)
	}
}
