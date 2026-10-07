package sim

// The optimised search, read against the contract rather than against itself.
//
// The search is called directly. A tick does not read the mode yet, so a test
// that went through Step would be measuring a dispatch that does not exist.

import "testing"

func optimisedWorld(t *testing.T, b Bounds, grid []byte, ents []Entity) *World {
	t.Helper()
	return mustWorldGrid(t, 1, b, ModeOptimised, grid, ents)
}

// ------------------------------------------------- the corner

// TestTheRouteMayNotCutACorner: a 3x3 with (1,0) blocking ground, a unit at
// (0,0) ordered to (1,1).
//
//	. # .        The diagonal straight to the target passes between (1,0) and
//	. . .        (0,1). (1,0) is blocked, so that step is not admissible however
//	. . .        cheap it is, and the route goes round: (0,1) then (1,1), two
//	             orthogonal steps at 4 against the diagonal's 3.
//
// The canonical wave takes the diagonal on this same grid, which is what makes
// the corner rule a difference between the two searches rather than a rule one
// of them states and neither exercises.
func TestTheRouteMayNotCutACorner(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := openGrid(b)
	grid[0*3+1] = blockGround
	w := optimisedWorld(t, b, grid, []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	got, ok := w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, 1, 1)
	checkRoute(t, "(0,0) to (1,1) round a corner it may not cut", got, ok, []cell{{0, 1}, {1, 1}})

	if w.cornerClear(s, unitRelation, 0, 0, 0, 1, 1) {
		t.Error("the diagonal out of (0,0) is corner-clear, but (1,0) blocks ground")
	}
	got, ok = w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 1, 1)
	checkRoute(t, "the wave on the same grid cuts it", got, ok, []cell{{1, 1}})
}

// ------------------------------------------------- the region, and its removal

// TestATargetReachableOnlyByADetourRightAcrossTheMapIsRoutedTo: a 25x3 with row
// 1 walled across but for one gap, and a unit at (0,0) ordered to (0,2) — one
// cell away as the crow flies, on the far side of the wall.
//
// This is the fixture the deleted region refused. The rectangle containing
// (0,0) and (0,2) grown by 8 and clipped was x in [0,8], and every row-1 cell in
// it is blocked, so with the gap at x=20 the answer was "no route" for an order
// a walk of forty-two steps satisfies. Unbounded, the search walks it: east
// along row 0 to (20,0), down through the gap, and west along row 2.
//
// The route is forced cell for cell and needs no tie-break. Row 1 is blocked but
// for the gap, so no diagonal step out of row 0 or row 2 is admissible anywhere;
// and the gap itself may only be entered and left orthogonally, since either
// diagonal passes beside a blocked row-1 cell. Forty-two straight steps at 84.
func TestATargetReachableOnlyByADetourRightAcrossTheMapIsRoutedTo(t *testing.T) {
	b := Bounds{Width: 25, Height: 3}
	const gap int32 = 20
	g := openGrid(b)
	for x := int32(0); x < b.Width; x++ {
		if x != gap {
			g[1*b.Width+x] = blockGround
		}
	}
	w := optimisedWorld(t, b, g, []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	want := make([]cell, 0, 42)
	for x := int32(1); x <= gap; x++ {
		want = append(want, cell{x, 0})
	}
	want = append(want, cell{gap, 1}, cell{gap, 2})
	for x := gap - 1; x >= 0; x-- {
		want = append(want, cell{x, 2})
	}
	got, ok := w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, 0, 2)
	checkRoute(t, "the wall whose only gap is right across the map", got, ok, want)

	// And the same search WITH a window is the bound this story does keep: the
	// gap is 20 cells away, the window reaches 8, so the near arm answers no
	// route on the same world and the same order.
	if route, ok := w.optimisedRoute(s, 0, unitRelation, dynamicWindow, exactGoal, 0, 2); ok {
		t.Errorf("a windowed search reached a gap 20 cells outside its window: %s", fmtRoute(route))
	}
}

// TestTheWindowIsAChebyshevSquareOnTheMover is the bound that replaced the
// rectangle, read one cell at a time. It is centred on the MOVER and not on the
// pair, so what it admits does not depend on where the order pointed; and the
// arithmetic is int64 because a target may name any cell, so the difference of
// two coordinates need not fit in one.
func TestTheWindowIsAChebyshevSquareOnTheMover(t *testing.T) {
	win := window{centre: cell{10, 10}, half: dynamicWindow}
	for _, tc := range []struct {
		what string
		x, y int32
		want bool
	}{
		{"the centre itself", 10, 10, true},
		{"eight east", 18, 10, true},
		{"nine east", 19, 10, false},
		{"eight west and eight north, the corner", 2, 2, true},
		{"one past that corner on one axis", 2, 1, false},
		{"eight north but nine west", 1, 2, false},
		{"a cell at the positive extreme", 2147483647, 10, false},
		{"a cell at the negative extreme", -2147483648, 10, false},
	} {
		if got := win.holds(tc.x, tc.y); got != tc.want {
			t.Errorf("%s: (%d,%d) held = %v, want %v", tc.what, tc.x, tc.y, got, tc.want)
		}
	}

	// noWindow is not a very large square: it is no square. Every cell a
	// coordinate can name is inside it, the extremes included.
	none := window{centre: cell{0, 0}, half: noWindow}
	for _, c := range []cell{{0, 0}, {2147483647, -2147483648}, {-2147483648, 2147483647}} {
		if !none.holds(c.x, c.y) {
			t.Errorf("the far search's window refused (%d,%d)", c.x, c.y)
		}
	}
}

// TestATargetOffTheMapIsNoRoute: nothing may stand off the map, so nothing
// routes there. With the region gone this is the whole of what refuses such an
// order in this mode — the target is tested against the relation before any
// sweep, exactly as the wave tests it.
func TestATargetOffTheMapIsNoRoute(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	w := optimisedWorld(t, b, openGrid(b), []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	if route, ok := w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, 5, 5); ok {
		t.Errorf("a route off the map: %s", fmtRoute(route))
	}
	if route, ok := w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, -1, 0); ok {
		t.Errorf("a route onto a negative cell: %s", fmtRoute(route))
	}
}

// ------------------------------------------------- the start

// TestAStartOnACellItMayNotStandOnIsRoutedOffIt: the start is the one cell no
// rule of admissibility touches. It has no cost of its own until one is taken as
// the minimum over its LEGAL first steps, so a unit standing on a blocked cell
// is routed off it — and the first step obeys the corner rule like every other.
//
//	. . .        Start (1,1), blocked, ordered to (2,2). The diagonal straight
//	. # #        there passes between (2,1) and (1,2), and (2,1) is blocked, so
//	. . .        it is refused: the route is (1,2) then (2,2), cost 4, against
//	             the diagonal's 3.
func TestAStartOnACellItMayNotStandOnIsRoutedOffIt(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := openGrid(b)
	grid[1*3+1] = blockGround
	grid[1*3+2] = blockGround
	w := optimisedWorld(t, b, grid, []Entity{{ID: 1, X: 1, Y: 1}})
	s := newRouteScratch(w)

	if w.enterable(s, 0, 1, 1) {
		t.Fatal("the fixture's start cell is enterable; it is meant to be blocked")
	}
	got, ok := w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, 2, 2)
	checkRoute(t, "off a blocked start", got, ok, []cell{{1, 2}, {2, 2}})
}

// ------------------------------------------------- the tie-break

// TestTwoEqualCostRoutesAreSeparatedByTheOrderTheContractNames: a 4x4 with (1,1)
// blocking ground, a unit at (0,0) ordered to (2,2).
//
//	. . . .        Two diagonals would cost 6 and both pass through (1,1). Every
//	. # . .        four-step route is two easts and two norths in some order, and
//	. . . .        four of the six pass through (1,1); the two that do not cost
//	. . . .        8 each:
//
//	(1,0) (2,0) (2,1) (2,2)   and   (0,1) (0,2) (1,2) (2,2)
//
// Compared cell by cell as (y, x), the first cells are (0,1) and (1,0), so the
// first sequence orders before the second and it is the one taken. The two costs
// are read off the plane here rather than asserted, so the case witnesses a TIE
// broken rather than a cheaper route found.
func TestTwoEqualCostRoutesAreSeparatedByTheOrderTheContractNames(t *testing.T) {
	b := Bounds{Width: 4, Height: 4}
	grid := openGrid(b)
	grid[1*4+1] = blockGround
	w := optimisedWorld(t, b, grid, []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	got, ok := w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, 2, 2)
	checkRoute(t, "the smaller of two equal-cost routes", got, ok,
		[]cell{{1, 0}, {2, 0}, {2, 1}, {2, 2}})

	// The plane holds each cell's cost to the target. Both first steps are
	// orthogonal, so over the uniform default plane both routes come to 8 + 24.
	start := cell{0, 0}
	checkLabel(t, w, s, start, 1, 0, 24)
	checkLabel(t, w, s, start, 0, 1, 24)
}

// ------------------------------------------------- the scratch

// TestTheOptimisedSearchLeavesTheScratchReusable: both searches write the same
// plane and clear it by the same touch list, so a tick may run either of them
// after either of them. The two answers here are ones pinned separately above.
func TestTheOptimisedSearchLeavesTheScratchReusable(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := openGrid(b)
	grid[0*3+1] = blockGround
	w := optimisedWorld(t, b, grid, []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	got, ok := w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, 1, 1)
	checkRoute(t, "the optimised search first", got, ok, []cell{{0, 1}, {1, 1}})
	got, ok = w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 1, 1)
	checkRoute(t, "the wave on the scratch it left", got, ok, []cell{{1, 1}})
	got, ok = w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, 1, 1)
	checkRoute(t, "the optimised search again", got, ok, []cell{{0, 1}, {1, 1}})
}
