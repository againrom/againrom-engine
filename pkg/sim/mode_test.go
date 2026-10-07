package sim

// The mode where a tick meets it.
//
// The two searches have their own files and their own cases. What is measured
// here is that a world's mode reaches the tick at all, that it reaches it
// through route selection alone, and that two worlds alike in every other byte
// walk differently because of it.

import "testing"

// The AC-5 fixture: a 3x3 with (1,0) and (0,1) both blocking ground, and one
// unit at (0,0) ordered to (1,1).
//
//	. # .        The only way to the target in one step is the diagonal, and it
//	# . .        passes between the two blocked cells. The canonical wave takes
//	. . .        it — a diagonal is relaxed with no test on the two cells it
//	             passes between — and the optimised search refuses it and finds
//	             nothing else, because leaving (0,0) at all means cutting one
//	             corner or the other.
var ac5Bounds = Bounds{Width: 3, Height: 3}

func ac5Grid() []byte {
	g := make([]byte, 9)
	g[0*3+1] = blockGround
	g[1*3+0] = blockGround
	return g
}

var ac5Unit = Entity{ID: 1, X: 0, Y: 0}

// TestOneWorldDrivenByBothModes is AC-5. The two worlds are built from one
// fixture and differ in exactly one byte of the canonical form — the mode — so
// what separates their outcomes cannot be the grid, the seed, the bounds or the
// unit.
func TestOneWorldDrivenByBothModes(t *testing.T) {
	canonical := mustWorldGrid(t, 1, ac5Bounds, ModeCanonical, ac5Grid(), []Entity{ac5Unit})
	optimised := mustWorldGrid(t, 1, ac5Bounds, ModeOptimised, ac5Grid(), []Entity{ac5Unit})

	a, err := canonical.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	b, err := optimised.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	if len(a) != len(b) {
		t.Fatalf("the two forms are %d and %d bytes", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] && i != 29 {
			t.Fatalf("the two worlds differ at offset %d as well as at the mode byte", i)
		}
	}
	if a[29] != byte(ModeCanonical) || b[29] != byte(ModeOptimised) {
		t.Fatalf("the mode bytes are %d and %d", a[29], b[29])
	}

	order := []Command{{Entity: ac5Unit.ID, X: 1, Y: 1}}
	Step(canonical, order)
	Step(optimised, order)

	// Canonical cuts the corner, arrives, and has its target cleared by the
	// arrival rule — facing SOUTH-EAST, the diagonal step it cut the corner
	// with. The optimised unit below never steps at all, so its facing is the
	// north it was built with, and the two together are what makes the pair of
	// literals here say which unit moved.
	if got, want := canonical.Entities()[0], (Entity{ID: 1, X: 1, Y: 1, Facing: facingOfDir(3), DesiredFacing: facingOfDir(3), ActorState: actorStateGuard, Reach: 1}); got != want {
		t.Errorf("under the canonical mode the unit is %+v, want %+v", got, want)
	}
	// Optimised finds no admissible route over the TERRAIN, and terrain does not
	// change while a world is advanced — so the order is over in the tick that
	// found it unservable, with no residue and no stall.
	want := Entity{ID: 1, X: 0, Y: 0, ActorState: actorStateGuard, Reach: 1}
	if got := optimised.Entities()[0]; got != want {
		t.Errorf("under the optimised mode the unit is %+v, want %+v", got, want)
	}

	if canonical.Hash() == optimised.Hash() {
		t.Error("the two worlds hash equal after a tick that moved one of them and not the other")
	}
}

// TestTheModeIsReadFromTheWorldAndNowhereElse: a world decoded from bytes routes
// by the mode those bytes carry. The mode is not a parameter of Step and not a
// package variable, so the only way a decoded world can route by the right
// search is to have read it out of its own state.
func TestTheModeIsReadFromTheWorldAndNowhereElse(t *testing.T) {
	source := mustWorldGrid(t, 1, ac5Bounds, ModeOptimised, ac5Grid(), []Entity{ac5Unit})
	form, err := source.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}

	// Decoded into a world that was built canonical: whatever the receiver was,
	// it is the bytes that decide.
	w := mustWorldGrid(t, 99, Bounds{Width: 1, Height: 1}, ModeCanonical, nil, nil)
	if err := w.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	Step(w, []Command{{Entity: ac5Unit.ID, X: 1, Y: 1}})

	// The canonical wave cuts this corner and arrives; the optimised search
	// refuses it and ends the order. Either way the unit ends the tick holding
	// nothing, so what separates the two is WHERE it is standing.
	want := Entity{ID: 1, X: 0, Y: 0, ActorState: actorStateGuard, Reach: 1}
	if got := w.Entities()[0]; got != want {
		t.Errorf("the decoded world's unit is %+v, want %+v — it routed by the wrong search", got, want)
	}
}

func TestTheResolutionOrderIsAscendingIdUnderBothModes(t *testing.T) {
	b := Bounds{Width: 5, Height: 5}
	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, b, mode, openGrid(b), []Entity{
			{ID: 1, X: 0, Y: 0},
			{ID: 2, X: 2, Y: 2},
		})
		Step(w, []Command{
			{Entity: 1, X: 1, Y: 1},
			{Entity: 2, X: 1, Y: 1},
		})

		ents := w.Entities()
		// Id 1 stepped one cell south-east from (0,0) to (1,1) and arrived; id 2
		// found the cell taken and never moved, so it keeps the north it was
		// built with.
		if got, want := ents[0], (Entity{ID: 1, X: 1, Y: 1, Facing: facingOfDir(3), DesiredFacing: facingOfDir(3), ActorState: actorStateGuard, Reach: 1}); got != want {
			t.Errorf("mode %d: the lower id is %+v, want %+v", mode, got, want)
		}
		want := Entity{ID: 2, X: 2, Y: 2, TargetX: 1, TargetY: 1, HasTarget: true, Stall: 1, ActorState: actorStateGuard, Reach: 1,
			PostX: 2, PostY: 2}
		if got := ents[1]; got != want {
			t.Errorf("mode %d: the higher id is %+v, want %+v", mode, got, want)
		}
	}
}
