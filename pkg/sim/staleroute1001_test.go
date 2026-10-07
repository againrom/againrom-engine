package sim

import "testing"

// srRoundTrip marshals w and unmarshals the result into a fresh World,
// failing the test with the decoder's own message when the byte form the
// game would have saved does not load.
func srRoundTrip(t *testing.T, w *World) {
	t.Helper()
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary refused the byte form the game would have saved: %v", err)
	}
}

// TestAMoveOrderOnACastingActorRoundTrips is R3-C1's single-writer case: the
// plain KindMoveTo arm at step.go. The caster is walking (a stored route
// stands) when a book cast begins, and a second move order then lands on it
// while the cast still owns its actor.
func TestAMoveOrderOnACastingActorRoundTrips(t *testing.T) {
	caster := spMage(1, 0, 0, 60, 50, 40, 1<<1)
	caster.Speed = 12
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 42, []SpellRule{ogArrow()}, caster, victim)

	// A walk far enough to leave a stored route standing.
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 0, Y: 8}})
	if len(w.routes[0]) == 0 {
		t.Fatalf("the fixture needs a stored route before the cast begins, got none")
	}

	// An automatic cast begins while the walk is still in progress. Admission writes
	// only Facing and the pending cast record; it does not touch the target
	// or the route (beginBookSpell, spell.go).
	if !w.beginBookSpellOnce(0, 2, 1) {
		t.Fatal("automatic cast refused")
	}
	if len(w.bookCasts) != 1 {
		t.Fatalf("the fixture admitted no cast: %d pending", len(w.bookCasts))
	}
	if len(w.routes[0]) == 0 {
		t.Fatalf("the walk finished before the cast began; the fixture needs a route still standing " +
			"when the second order lands")
	}
	oldTarget := [2]int32{spAt(t, w, 1).TargetX, spAt(t, w, 1).TargetY}

	// R3-C1's own trigger: a move order lands on the casting actor and aims
	// it somewhere else.
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 0}})
	e := spAt(t, w, 1)
	if !e.HasTarget || e.TargetX != 5 || e.TargetY != 0 {
		t.Fatalf("the move order was not taken: held=%v (%d,%d)", e.HasTarget, e.TargetX, e.TargetY)
	}
	if e.TargetX == oldTarget[0] && e.TargetY == oldTarget[1] {
		t.Fatalf("the fixture's second order named the same target as the first, want a different one")
	}
	if len(w.bookCasts) != 1 {
		t.Fatalf("the move order cancelled the pending cast: %d pending", len(w.bookCasts))
	}

	srRoundTrip(t, w)
}

// TestAGroupMoveOrderOnACastingMemberRoundTrips is R3-C1's group-writer case:
// issueGroupDestination at group.go, reached through KindGroupMoveTo. spec.md
// states that a group order addresses every living member, casting ones
// included, so the same mismatch is reachable through the group arm.
func TestAGroupMoveOrderOnACastingMemberRoundTrips(t *testing.T) {
	caster := spMage(1, 0, 0, 60, 50, 40, 1<<1)
	caster.Speed = 12
	victim := spEnt(2, 1, 0)
	w := spWorld(t, 42, []SpellRule{ogArrow()}, caster, victim)

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 0, Y: 8}})
	if len(w.routes[0]) == 0 {
		t.Fatalf("the fixture needs a stored route before the cast begins, got none")
	}

	if !w.beginBookSpellOnce(0, 2, 1) {
		t.Fatal("automatic cast refused")
	}
	if len(w.bookCasts) != 1 {
		t.Fatalf("the fixture admitted no cast: %d pending", len(w.bookCasts))
	}
	if len(w.routes[0]) == 0 {
		t.Fatalf("the walk finished before the cast began; the fixture needs a route still standing " +
			"when the group order lands")
	}
	oldTarget := [2]int32{spAt(t, w, 1).TargetX, spAt(t, w, 1).TargetY}

	// A group order of one, naming the casting member: issueGroupDestination
	// is the same writer a real multi-member selection reaches.
	Step(w, []Command{{Kind: KindGroupMoveTo, Entity: 1, X: 5, Y: 0, Group: 1}})
	e := spAt(t, w, 1)
	if !e.HasTarget || e.TargetX != 5 || e.TargetY != 0 {
		t.Fatalf("the group order was not taken: held=%v (%d,%d)", e.HasTarget, e.TargetX, e.TargetY)
	}
	if e.TargetX == oldTarget[0] && e.TargetY == oldTarget[1] {
		t.Fatalf("the fixture's second order named the same target as the first, want a different one")
	}
	if len(w.bookCasts) != 1 {
		t.Fatalf("the group order cancelled the pending cast: %d pending", len(w.bookCasts))
	}

	srRoundTrip(t, w)
}

// TestAMoveOrderMidTransitRoundTrips is the older invariant break the review
// names beside R3-C1: at origin/master, a re-order landing on a mover that
// owes transit ticks for its current crossing hits the same skip (step.go's
// walk loop stands down while e.Transit > 0) and left the same mismatch. This
// story's fix is unconditional on why the walk is not advancing the actor
// this tick, so the mid-transit case is fixed by the same write.
func TestAMoveOrderMidTransitRoundTrips(t *testing.T) {
	a := cbEnt(1, 0, 0)
	a.Speed = 1
	w := cbWorld(t, 3, a, cbEnt(2, 8, 8))

	// A walk far enough to take on a crossing (a slow mover: Speed 1 owes
	// several ticks per cell) while leaving cells still ahead of it on the
	// route.
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 0, Y: 8}})
	if w.entities[0].Transit == 0 {
		t.Fatalf("the fixture needs a mover mid-crossing, got Transit 0")
	}
	if len(w.routes[0]) == 0 {
		t.Fatalf("the fixture needs a stored route, got none")
	}
	oldTarget := [2]int32{cbAt(t, w, 1).TargetX, cbAt(t, w, 1).TargetY}

	// A re-order lands while the crossing is still owed.
	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 5, Y: 0}})
	e := cbAt(t, w, 1)
	if !e.HasTarget || e.TargetX != 5 || e.TargetY != 0 {
		t.Fatalf("the move order was not taken: held=%v (%d,%d)", e.HasTarget, e.TargetX, e.TargetY)
	}
	if e.TargetX == oldTarget[0] && e.TargetY == oldTarget[1] {
		t.Fatalf("the fixture's second order named the same target as the first, want a different one")
	}

	srRoundTrip(t, w)
}

// TestAGroupMoveOrderMidTransitRoundTrips is the mid-transit case at the
// group writer.
func TestAGroupMoveOrderMidTransitRoundTrips(t *testing.T) {
	a := cbEnt(1, 0, 0)
	a.Speed = 1
	w := cbWorld(t, 3, a, cbEnt(2, 8, 8))

	Step(w, []Command{{Kind: KindMoveTo, Entity: 1, X: 0, Y: 8}})
	if w.entities[0].Transit == 0 {
		t.Fatalf("the fixture needs a mover mid-crossing, got Transit 0")
	}
	if len(w.routes[0]) == 0 {
		t.Fatalf("the fixture needs a stored route, got none")
	}
	oldTarget := [2]int32{cbAt(t, w, 1).TargetX, cbAt(t, w, 1).TargetY}

	Step(w, []Command{{Kind: KindGroupMoveTo, Entity: 1, X: 5, Y: 0, Group: 1}})
	e := cbAt(t, w, 1)
	if !e.HasTarget || e.TargetX != 5 || e.TargetY != 0 {
		t.Fatalf("the group order was not taken: held=%v (%d,%d)", e.HasTarget, e.TargetX, e.TargetY)
	}
	if e.TargetX == oldTarget[0] && e.TargetY == oldTarget[1] {
		t.Fatalf("the fixture's second order named the same target as the first, want a different one")
	}

	srRoundTrip(t, w)
}
