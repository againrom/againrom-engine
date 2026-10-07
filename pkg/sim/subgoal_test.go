package sim

// The sub-goal, the consumption, and the three tests that send a tick back to
// the far search.
//
// The routes here are written out by hand and put on the world through the
// field, because that is the only way to hold a tick to a route it did not
// choose itself: a stored route the far search would have produced anyway can be
// replaced without anything showing. Every fixture below therefore stores a
// route the far search WOULD NOT have returned, so "the stored route was used"
// and "the stored route was thrown away" are two different worlds afterwards.

import "testing"

// sgFar counts the far searches the next Step will run, by asking the contract's
// own test — the one the tick asks — of every unit before the tick.
//
// It is the tick's decision and not a second opinion about it: subGoal is the
// production predicate, asked at the moment the tick asks it. What it cannot see
// is a tick that runs a far search for some OTHER reason, which is why the
// fixtures below also compare the route a tick left behind against the route it
// was given.
func sgFar(w *World) int {
	n := 0
	for i := range w.entities {
		e := &w.entities[i]
		if !e.HasTarget || (e.X == e.TargetX && e.Y == e.TargetY) {
			continue
		}
		if _, serves := w.subGoal(newRouteScratch(w), i); !serves {
			n++
		}
	}
	return n
}

// sgWorld is a sixteen by sixteen of open ground with the given units, and one
// hand-written route put on the first of them.
func sgWorld(t *testing.T, ents []Entity, route []cell) *World {
	t.Helper()
	w := mustWorld(t, 1, Bounds{Width: 16, Height: 16}, ents)
	w.routes[0] = route
	return w
}

// ------------------------------------------------- the sub-goal

// TestTheSubGoalIsTheCellAtIndexThreeUntilFewerRemain reads the contract's own
// example off a hand-written route: six cells, and the cell at index 3 is the
// one a near search aims at. Then the same route consumed down to one cell, so
// the other arm — the target itself, once four or fewer remain — is read at every
// length it can be read at rather than at one.
func TestTheSubGoalIsTheCellAtIndexThreeUntilFewerRemain(t *testing.T) {
	route := []cell{{4, 7}, {5, 6}, {5, 5}, {5, 4}, {5, 3}, {5, 2}}
	for _, tc := range []struct {
		n    int
		want cell
	}{
		{6, cell{5, 4}}, // index 3, four cells along
		{5, cell{5, 4}},
		{4, cell{5, 4}}, // index 3 is the last cell now
		{3, cell{5, 5}},
		{2, cell{5, 6}},
		{1, cell{4, 7}},
	} {
		if got := subGoalIndex(tc.n); got != len(route[:tc.n])-1 && got != lookAhead {
			t.Errorf("a route of %d cell(s) has sub-goal index %d, which is neither its last nor %d",
				tc.n, got, lookAhead)
		}
		if got := route[:tc.n][subGoalIndex(tc.n)]; got != tc.want {
			t.Errorf("a route of %d cell(s) aims at (%d,%d), want (%d,%d)",
				tc.n, got.x, got.y, tc.want.x, tc.want.y)
		}
	}
}

// TestAStoredRouteThatServesIsAskedForNoFarSearch is the discriminating case,
// and the fixture is built so that using the stored route and replacing it are
// visibly different: the unit at (0,0) is ordered to (5,0), which a far search
// answers with the straight line, and the route it is GIVEN loops south. If a
// far search ran, the loop would be gone.
//
// It also carries the second half of the consumption rule. The near search aims
// at the sub-goal (3,2) and steps to (1,1), which is on no cell of the route, so
// the mover keeps every cell of it and makes its way back.
func TestAStoredRouteThatServesIsAskedForNoFarSearch(t *testing.T) {
	loop := []cell{{0, 1}, {1, 2}, {2, 2}, {3, 2}, {4, 1}, {5, 0}}
	w := sgWorld(t, []Entity{{ID: 1, X: 0, Y: 0, TargetX: 5, TargetY: 0, HasTarget: true}}, loop)

	sub, serves := w.subGoal(newRouteScratch(w), 0)
	if !serves || sub != (cell{3, 2}) {
		t.Fatalf("the sub-goal is (%d,%d) (serves=%v), want (3,2)", sub.x, sub.y, serves)
	}
	if got := sgFar(w); got != 0 {
		t.Errorf("%d far search(es) are due, want none — the stored route serves", got)
	}

	Step(w, nil)

	if got := w.Entities()[0]; got.X != 1 || got.Y != 1 {
		t.Fatalf("the mover is at (%d,%d), want (1,1) — the near search's own first cell", got.X, got.Y)
	}
	checkRoute(t, "a route whose mover stepped off it", w.routes[0], true, loop)
}

// TestAMoverThatLandsOnItsRoutesThirdCellDropsThatCellAndEveryCellBefore is the
// consumption rule's other half, read on a route small enough to check by eye. A
// mover reaches cell 2 in one step when its route doubles back beside it, so the
// landing is put there directly rather than contrived out of a grid.
func TestAMoverThatLandsOnItsRoutesThirdCellDropsThatCellAndEveryCellBefore(t *testing.T) {
	route := []cell{{0, 1}, {1, 2}, {1, 1}, {2, 1}, {3, 1}, {4, 1}}
	for _, tc := range []struct {
		what string
		at   cell
		want []cell
	}{
		{"the head", cell{0, 1}, route[1:]},
		{"cell 2", cell{1, 1}, route[3:]},
		{"cell 3, the last that may be consumed", cell{2, 1}, route[4:]},
		{"cell 4, one past it", cell{3, 1}, route},
		{"a cell of no route", cell{9, 9}, route},
	} {
		w := sgWorld(t, []Entity{{ID: 1, X: 0, Y: 0, TargetX: 4, TargetY: 1, HasTarget: true}},
			append([]cell(nil), route...))
		w.entities[0].X, w.entities[0].Y = tc.at.x, tc.at.y
		w.consume(0)
		checkRoute(t, "landing on "+tc.what, w.routes[0], true, tc.want)
	}
}

// ------------------------------------------------- the three staleness tests

// TestEachStalenessTestSendsTheTickBackToTheFarSearch is AC-8's three cases,
// each its own fixture and each with the far searches counted. The fourth row is
// the counterfactual: the same world with a route that serves runs none.
func TestEachStalenessTestSendsTheTickBackToTheFarSearch(t *testing.T) {
	target := cell{5, 0}
	serving := []cell{{0, 1}, {1, 2}, {2, 2}, {3, 2}, {4, 1}, {5, 0}}
	// A route whose cells are all far from the mover: its sub-goal is (14,0),
	// fourteen cells off, where the window reaches eight.
	distant := []cell{{11, 0}, {12, 0}, {13, 0}, {14, 0}, {15, 0}}

	for _, tc := range []struct {
		what   string
		route  []cell
		target cell
		want   int
	}{
		{"no route at all", nil, target, 1},
		{"a route whose last cell is not the target", []cell{{0, 1}, {1, 1}, {2, 1}}, target, 1},
		{"a route whose sub-goal is outside the window", distant, cell{15, 0}, 1},
		{"a route that serves", serving, target, 0},
	} {
		w := sgWorld(t, []Entity{{
			ID: 1, X: 0, Y: 0, TargetX: tc.target.x, TargetY: tc.target.y, HasTarget: true,
		}}, tc.route)

		if got := sgFar(w); got != tc.want {
			t.Errorf("%s: %d far search(es) due, want %d", tc.what, got, tc.want)
		}
		Step(w, nil)
		// A far search replaces the stored route with one ending at the target,
		// so the three stale cases end the tick holding a route the fixture did
		// not give them.
		got := w.routes[0]
		if len(got) == 0 {
			t.Fatalf("%s: the mover holds no route after the tick", tc.what)
		}
		if last := got[len(got)-1]; last != tc.target {
			t.Errorf("%s: the stored route ends at (%d,%d), want the target (%d,%d)",
				tc.what, last.x, last.y, tc.target.x, tc.target.y)
		}
	}
}

// TestAReTargetedMoverRunsOneFarSearchAndKeepsNothingOfTheOldRoute is the
// staleness test a player provokes: an order given to a unit already walking
// somewhere else.
func TestAReTargetedMoverRunsOneFarSearchAndKeepsNothingOfTheOldRoute(t *testing.T) {
	w := sgWorld(t, []Entity{{ID: 1, X: 0, Y: 0, TargetX: 5, TargetY: 0, HasTarget: true}},
		[]cell{{0, 1}, {1, 2}, {2, 2}, {3, 2}, {4, 1}, {5, 0}})
	if got := sgFar(w); got != 0 {
		t.Fatalf("%d far search(es) due before the re-order, want none", got)
	}

	Step(w, []Command{{Entity: 1, X: 0, Y: 5}})

	route := w.routes[0]
	if len(route) == 0 {
		t.Fatal("the re-targeted mover holds no route")
	}
	if last := route[len(route)-1]; last != (cell{0, 5}) {
		t.Errorf("the stored route ends at (%d,%d), want the new target (0,5)", last.x, last.y)
	}
	for _, c := range route {
		if c == (cell{5, 0}) {
			t.Errorf("the stored route still holds the old target's cell: %s", fmtRoute(route))
		}
	}
}

func TestAMoverThatFailedToMoveRunsNoFarSearchOnTheTickAfter(t *testing.T) {
	ents := []Entity{{ID: 1, X: 5, Y: 5, TargetX: 9, TargetY: 5, HasTarget: true}}
	id := EntityID(2)
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			if dx != 0 || dy != 0 {
				ents = append(ents, Entity{ID: id, X: 5 + dx, Y: 5 + dy})
				id++
			}
		}
	}
	route := []cell{{6, 5}, {7, 5}, {8, 5}, {9, 5}}
	w := sgWorld(t, ents, append([]cell(nil), route...))

	for tick := 1; tick <= 3; tick++ {
		if got := sgFar(w); got != 0 {
			t.Errorf("tick %d: %d far search(es) due, want none — nothing about the route changed", tick, got)
		}
		Step(w, nil)

		e := w.Entities()[0]
		if e.X != 5 || e.Y != 5 {
			t.Fatalf("tick %d: the boxed-in mover reached (%d,%d)", tick, e.X, e.Y)
		}
		if e.Stall != uint8(tick) {
			t.Errorf("tick %d: the stall count is %d, want %d", tick, e.Stall, tick)
		}
		checkRoute(t, "the route of a mover that could not move", w.routes[0], true, route)
	}
}

func TestAMoverThatBeginsOnItsTargetLosesItsRouteWithItsTarget(t *testing.T) {
	w := sgWorld(t, []Entity{{ID: 1, X: 4, Y: 4, TargetX: 4, TargetY: 4, HasTarget: true, Stall: 5}},
		[]cell{{4, 3}, {4, 4}})

	Step(w, nil)

	if got, want := w.Entities()[0], (Entity{ID: 1, X: 4, Y: 4, ActorState: actorStateGuard, Reach: 1, PostX: 4, PostY: 4}); got != want {
		t.Errorf("the unit is %+v, want %+v — cleared with no residue", got, want)
	}
	if len(w.routes[0]) != 0 {
		t.Errorf("the unit holds the route %s with no target", fmtRoute(w.routes[0]))
	}
	// And the world it leaves is one its own byte form accepts, which is the
	// mechanical reason the two are cleared together.
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Errorf("the world left behind does not decode: %v", err)
	}
}
