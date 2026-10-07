package sim

// The wave, read against the contract rather than against itself.
//
// Every route below is worked out by hand from the procedure — the seed, the
// generations in frontier order, the stop test, the backwards walk and its
// asymmetric accept — on grids small enough that the whole label plane fits in
// a comment. Nothing here is a route captured from a run of the search.
//
// The searches are called directly rather than by advancing a world: a tick
// neither reads a route nor takes one yet, so a test that went through Step
// would be measuring code that does not exist.

import (
	"fmt"
	"testing"
)

func routeWorld(t *testing.T, b Bounds, grid []byte, ents []Entity) *World {
	t.Helper()
	return mustWorldGrid(t, 1, b, ModeCanonical, grid, ents)
}

func fmtRoute(r []cell) string {
	out := ""
	for _, c := range r {
		out += fmt.Sprintf("(%d,%d) ", c.x, c.y)
	}
	if out == "" {
		return "<empty>"
	}
	return out
}

// checkRoute compares a route cell by cell. A route is a sequence, so a set
// comparison or a cost comparison would both pass a different walk of the same
// length.
func checkRoute(t *testing.T, what string, got []cell, ok bool, want []cell) {
	t.Helper()
	if !ok {
		t.Fatalf("%s: no route, want %s", what, fmtRoute(want))
	}
	if len(got) != len(want) {
		t.Fatalf("%s: route is %s, want %s", what, fmtRoute(got), fmtRoute(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: route is %s, want %s", what, fmtRoute(got), fmtRoute(want))
		}
	}
}

// checkLabel reads one cell's label back off the plane. The plane is left
// holding the last search's labels, which is what lets a test say WHY a route
// came out the way it did and not merely that it did.
func checkLabel(t *testing.T, w *World, s *routeScratch, start cell, x, y int32, want uint64) {
	t.Helper()
	got, ok := w.label(s, start, x, y)
	if !ok {
		t.Errorf("(%d,%d) is unlabelled, want %d", x, y, want)
		return
	}
	if got != want {
		t.Errorf("(%d,%d) is labelled %d, want %d", x, y, got, want)
	}
}

// openGrid is a passable extent of the given size, so a fixture that means "no
// obstacles" says so once.
func openGrid(b Bounds) []byte { return make([]byte, b.Width*b.Height) }

// ------------------------------------------------- the label read where it stands

// TestTwoFrontierCellsReachingOneNeighbour is the relaxation rule's own shape.
//
// A 3x3 with nothing on it, one unit at (2,1) ordered to (0,1). Generation 1
// labels (1,0) and (1,2) at 3 and (1,1) at 2, all three neighbouring (0,1); the
// frontier holds them in that order. Generation 2 relaxes them in it: (1,0)
// offers (0,1) a diagonal 6 and writes it, (1,1) offers an orthogonal 4 and
// lowers it, and (1,2)'s diagonal 24 is then refused against the 16 that (1,1)
// wrote where the contract says to read it. So the target is 16, and the whole
// plane comes out
//
//	20 16 20
//	12  8 12
//	 8  0  8   (column x, row y; the start's 0 at (2,1))
//
// The backwards walk from (0,1) at 16 takes (1,1) — the diagonal 24 offered by
// (1,0) loses to (1,1)'s orthogonal 16 — and from (1,1) it reaches the start.
// Every number is the uniform default plane's 8 and 12 where the flat rule gave
// 2 and 3: a constant factor, and the same walk.
func TestTwoFrontierCellsReachingOneNeighbour(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	w := routeWorld(t, b, openGrid(b), []Entity{{ID: 1, X: 2, Y: 1}})
	s := newRouteScratch(w)

	got, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 0, 1)
	checkRoute(t, "(2,1) to (0,1) on an open 3x3", got, ok, []cell{{1, 1}, {0, 1}})

	start := cell{2, 1}
	for _, tc := range []struct {
		x, y  int32
		label uint64
	}{
		{0, 0, 20}, {1, 0, 12}, {2, 0, 8},
		{0, 1, 16}, {1, 1, 8}, {2, 1, 0},
		{0, 2, 20}, {1, 2, 12}, {2, 2, 8},
	} {
		checkLabel(t, w, s, start, tc.x, tc.y, tc.label)
	}
}

// TestTheWalkBackAcceptsAnEqualStraightAndRefusesAnEqualDiagonal names the tie
// the fixture above contains, since a route pinned cell by cell says which cell
// was taken and not which rule took it.
//
// At (1,1) the walk scans in the contract's order and the running best goes
// 8 at (0,0) — 6 at (0,1) — 5 at (1,0) — 5 at (1,2) — 5 refused at (2,0) — 2 at
// (2,1). The fourth is an EQUAL ORTHOGONAL and it displaces; the fifth is an
// EQUAL DIAGONAL and it does not. Both are read off the labels here rather than
// asserted about the code.
//
// What this fixture does NOT do is decide the route by that asymmetry: the start
// wins the cell outright either way. The two mutants that turn the accepts
// around are run by the entries that own them.
//
// The numbers are the GROUND arm's over a world naming no cost plane, so every
// cell costs the uniform default: 8 orthogonally and 12 diagonally, which is
// four times the flat 2 and 3 a non-ground mover still pays. A constant factor
// on every label, so every comparison here lands where it landed before.
func TestTheWalkBackAcceptsAnEqualStraightAndRefusesAnEqualDiagonal(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	w := routeWorld(t, b, openGrid(b), []Entity{{ID: 1, X: 2, Y: 1}})
	s := newRouteScratch(w)

	if _, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 0, 1); !ok {
		t.Fatal("no route")
	}
	start := cell{2, 1}

	// At (1,1): the straight candidates (1,0) and (1,2) both cost label+8 = 20,
	// and the diagonal (2,0) costs label+12 = 20 as well.
	for _, tc := range []struct {
		what             string
		x, y             int32
		label, candidate uint64
	}{
		{"the straight taken first", 1, 0, 12, 20},
		{"the equal straight that displaces it", 1, 2, 12, 20},
		{"the equal diagonal that does not", 2, 0, 8, 20},
	} {
		l, ok := w.label(s, start, tc.x, tc.y)
		if !ok {
			t.Fatalf("%s: (%d,%d) is unlabelled", tc.what, tc.x, tc.y)
		}
		if l != tc.label {
			t.Errorf("%s: (%d,%d) is labelled %d, want %d", tc.what, tc.x, tc.y, l, tc.label)
		}
		dx, dy := tc.x-1, tc.y-1
		// Charged at the cell the step ENTERS, which for this backwards walk is
		// the one being left — (1,1), not the candidate.
		if c := l + stepCost(DomainGround, w.costAt(cell{1, 1}), dx, dy); c != tc.candidate {
			t.Errorf("%s: (%d,%d) offers %d into (1,1), want %d", tc.what, tc.x, tc.y, c, tc.candidate)
		}
	}
}

// ------------------------------------------------- enterability

// TestAnotherUnitIsNotEnterable is the same grid, the same order and the same
// start, with one unit parked on (1,1). It is the cheapest possible statement
// that occupancy is part of enterability rather than a separate later test: the
// route changes to (1,0) then (0,1), because (1,1) is never labelled at all.
//
// The parked unit carries the LOWER id, so it is not the resolution order that
// makes the difference here.
func TestAnotherUnitIsNotEnterable(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	w := routeWorld(t, b, openGrid(b), []Entity{
		{ID: 1, X: 1, Y: 1},
		{ID: 2, X: 2, Y: 1},
	})
	s := newRouteScratch(w)

	// Index 1 is the unit at (2,1): the slice is kept in ascending id.
	got, ok := w.canonicalRoute(s, 1, unitRelation, noWindow, flatBudget, exactGoal, 0, 1)
	checkRoute(t, "(2,1) to (0,1) past a unit on (1,1)", got, ok, []cell{{1, 0}, {0, 1}})

	if _, labelled := w.label(s, cell{2, 1}, 1, 1); labelled {
		t.Error("(1,1) is labelled, but a cell another unit stands on is not enterable")
	}
	// And a unit is not its own obstacle: its own cell is where the search
	// begins, and it is enterable to itself.
	if !w.enterable(s, 1, 2, 1) {
		t.Error("a unit's own cell is not enterable to itself")
	}
	if w.enterable(s, 0, 2, 1) {
		t.Error("a cell another unit stands on is enterable to a second unit")
	}
}

// TestAStartOnABlockedCellIsRoutedOffIt: a search begins from the entity's own
// cell whether or not that cell is enterable. The unit stands on the one blocked
// cell of a 3x3 and is ordered to (0,0); generation 1 labels every other cell,
// (0,0) among them at a diagonal 3, and the walk back reaches the start in one
// step.
func TestAStartOnABlockedCellIsRoutedOffIt(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := openGrid(b)
	grid[1*3+1] = blockGround
	w := routeWorld(t, b, grid, []Entity{{ID: 1, X: 1, Y: 1}})
	s := newRouteScratch(w)

	if w.enterable(s, 0, 1, 1) {
		t.Fatal("the fixture's start cell is enterable; it is meant to be blocked")
	}
	got, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 0, 0)
	checkRoute(t, "off a blocked start", got, ok, []cell{{0, 0}})
}

// TestAStartOutsideTheBoundsIsRoutedOntoTheMap: the start may lie off the grid
// entirely, where the plane has no slot for it. Its label of 0 is answered
// without one, and the walk back ends by comparing coordinates rather than by
// finding a zero.
//
// From (-1,-1) the only enterable neighbour is (0,0) at a diagonal 3; from there
// (1,1) is a diagonal 6, and the walk comes back through (0,0) to a start that
// no plane slot describes.
func TestAStartOutsideTheBoundsIsRoutedOntoTheMap(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	w := routeWorld(t, b, openGrid(b), []Entity{{ID: 1, X: -1, Y: -1}})
	s := newRouteScratch(w)

	got, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 1, 1)
	checkRoute(t, "onto the map from (-1,-1)", got, ok, []cell{{0, 0}, {1, 1}})

	start := cell{-1, -1}
	checkLabel(t, w, s, start, -1, -1, 0)
	checkLabel(t, w, s, start, 0, 0, 12)
	checkLabel(t, w, s, start, 1, 1, 24)
	if _, ok := w.label(s, start, -1, 0); ok {
		t.Error("(-1,0) is labelled: an out-of-bounds cell that is not the start has no label")
	}
}

// ------------------------------------------------- the three stop conditions

// TestADetourInsideTheBudgetIsFound is the budget's positive half, and the one
// fixture here whose route needs more generations than the Chebyshev distance
// alone would buy.
//
//	. # .        the unit starts at (0,0), the target is (2,0), and the two
//	. # .        blocked cells force it down and round: D is 2, the walk is 4
//	. . .        steps, and the budget max(5, D>>2) + D = 7 covers it.
//
// The wave reaches the target in generation 4 with a label of 10, and the walk
// back reads (2,1) — (1,2) — (0,1) — the start.
func TestADetourInsideTheBudgetIsFound(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := []byte{
		0, blockGround, 0,
		0, blockGround, 0,
		0, 0, 0,
	}
	w := routeWorld(t, b, grid, []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	got, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 2, 0)
	checkRoute(t, "round two blocked cells", got, ok, []cell{{0, 1}, {1, 2}, {2, 1}, {2, 0}})

	start := cell{0, 0}
	checkLabel(t, w, s, start, 0, 1, 8)
	checkLabel(t, w, s, start, 1, 2, 20)
	checkLabel(t, w, s, start, 2, 1, 32)
	checkLabel(t, w, s, start, 2, 0, 40)
}

// The serpentine corridor: the fixture the budget stop is witnessed on, built so
// that NEITHER budget this package has carried can reach its target.
//
//	. . . . . . . .        Open rows at every even y, walls between them, and
//	# # # # # # # .        one gap per wall alternating between the two ends.
//	. . . . . . . .        The only route is the whole snake, so its length in
//	. # # # # # # #        hops is about rows x (width - 1) and not a function
//	. . . . . . . .        of how far apart the two cells look.
//
// That last clause is the point. This one needs more than a THOUSAND hops
// between two cells 58 apart, which the flat budget does not reach either,
// and the corridor is what makes the two independent.
//
// It stays reachable in optimised mode, which counts no generations at all, so
// the refusal is the budget's and never the terrain's.
const corridorWidth int32 = 40
const corridorRows int32 = 30

var corridorBounds = Bounds{Width: corridorWidth, Height: corridorRows*2 - 1}

// corridorTarget is the far end of the last open row. Row k is entered at x=0
// and left at x=width-1 when k is even and the other way about when it is odd,
// so the last of an even number of rows ends where it began, at x=0.
var corridorTarget = cell{x: 0, y: (corridorRows - 1) * 2}

func corridorGrid() []byte {
	g := make([]byte, corridorBounds.Width*corridorBounds.Height)
	for row := int32(1); row < corridorRows; row++ {
		gapX := int32(0)
		if row%2 == 1 {
			gapX = corridorWidth - 1
		}
		y := row*2 - 1
		for x := int32(0); x < corridorWidth; x++ {
			if x != gapX {
				g[y*corridorBounds.Width+x] = blockGround
			}
		}
	}
	return g
}

// TestATargetOutsideTheBudgetIsNoRoute is the budget's negative half: a target a
// unit could physically walk to, and does not.
//
// The fixture asserts its own claim before it asserts the refusal — the hop
// count is read off the optimised search, which has no generation budget, so the
// route's real length is measured rather than hand-counted from the drawing.
func TestATargetOutsideTheBudgetIsNoRoute(t *testing.T) {
	w := routeWorld(t, corridorBounds, corridorGrid(), []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	// What the corridor is worth, measured: the only route is longer than the
	// flat budget, and the two cells are 58 apart, so the scaled budget is
	// nowhere near it either.
	free, ok := w.optimisedRoute(s, 0, terrainRelation, noWindow, settleOrdered, corridorTarget.x, corridorTarget.y)
	if !ok {
		t.Fatal("no route at all through the corridor: the fixture is sealed, not long")
	}
	if len(free) <= 1000 {
		t.Fatalf("the corridor's only route is %d hops, and this fixture is claimed over 1000", len(free))
	}
	t.Logf("the corridor's only route is %d hops between two cells %d apart", len(free), corridorTarget.y)

	if route, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, corridorTarget.x, corridorTarget.y); ok {
		t.Fatalf("a route was found: %s", fmtRoute(route))
	}
	// The stop that fired is the budget's and not the sealed map's: the wave ran
	// its generations, labelled its way along the snake, and still held frontier
	// cells to expand when it stopped.
	if len(s.touched) == 0 {
		t.Error("no cell was labelled: this fixture measures the unenterable-target refusal and not the budget")
	}
	if len(s.frontier) == 0 {
		t.Error("the frontier is empty, so this fixture measures the sealed-map stop and not the budget")
	}
}

// TestASealedTargetIsNoRoute is the second stop condition: the frontier runs out
// with generations still to spend, because the far side of the map cannot be
// entered at all.
func TestASealedTargetIsNoRoute(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := []byte{
		0, blockGround, 0,
		0, blockGround, 0,
		0, blockGround, 0,
	}
	w := routeWorld(t, b, grid, []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	if route, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 2, 0); ok {
		t.Fatalf("a route was found through a sealed column: %s", fmtRoute(route))
	}
	if got := generationBudget(cell{0, 0}, cell{2, 0}, flatBudget, true); got != flatGenerations {
		t.Fatalf("the budget for this fixture is %d, want %d — the stop under test is the frontier's, "+
			"and it must fire with generations still to spend", got, flatGenerations)
	}
}

// TestATargetThatIsNotEnterableIsNoRoute: nothing ever labels a blocked cell, so
// an order onto one has no route by the same rule that keeps a unit off it.
func TestATargetThatIsNotEnterableIsNoRoute(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := openGrid(b)
	grid[1*3+1] = blockGround
	w := routeWorld(t, b, grid, []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	if route, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 1, 1); ok {
		t.Fatalf("a route onto a blocked cell: %s", fmtRoute(route))
	}
	if route, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 5, 5); ok {
		t.Fatalf("a route onto a cell off the map: %s", fmtRoute(route))
	}
}

func TestTheThreeRefusalsAreToldApartByWhatWasLabelled(t *testing.T) {
	sealed := Bounds{Width: 3, Height: 3}
	sealedGrid := openGrid(sealed)
	for y := int32(0); y < 3; y++ {
		sealedGrid[y*3+1] = blockGround
	}

	blocked := openGrid(sealed)
	blocked[1*3+1] = blockGround

	for _, tc := range []struct {
		what             string
		b                Bounds
		grid             []byte
		tx, ty           int32
		wantTouched      bool
		wantFrontierLeft bool
	}{
		{"a blocked destination", sealed, blocked, 1, 1, false, false},
		{"a destination off the map", sealed, blocked, 5, 5, false, false},
		{"a sealed map", sealed, sealedGrid, 2, 0, true, false},
		// Driven from the corridor, so that this row keeps measuring the budget
		// stop under a budget that is no longer a function of the straight line.
		{"a budget spent", corridorBounds, corridorGrid(), corridorTarget.x, corridorTarget.y, true, true},
	} {
		w := routeWorld(t, tc.b, tc.grid, []Entity{{ID: 1, X: 0, Y: 0}})
		s := newRouteScratch(w)
		if route, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, tc.tx, tc.ty); ok {
			t.Fatalf("%s: a route was found: %s", tc.what, fmtRoute(route))
		}
		if got := len(s.touched) > 0; got != tc.wantTouched {
			t.Errorf("%s: %d cell(s) were labelled, want any=%v", tc.what, len(s.touched), tc.wantTouched)
		}
		if got := len(s.frontier) > 0; got != tc.wantFrontierLeft {
			t.Errorf("%s: %d frontier cell(s) were left, want any=%v", tc.what, len(s.frontier), tc.wantFrontierLeft)
		}
	}
}

// TestAStartOnItsOwnTargetIsStillTheEmptyRoute is the case the early refusal has
// to leave alone: a search from a cell to itself answers the empty route with ok
// true, whether or not that cell is one the unit could enter. Nothing calls it
// that way — a unit already on its target neither searches nor steps — but the
// refusal is written to excuse it rather than to happen to miss it.
func TestAStartOnItsOwnTargetIsStillTheEmptyRoute(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := openGrid(b)
	grid[1*3+1] = blockGround
	w := routeWorld(t, b, grid, []Entity{{ID: 1, X: 1, Y: 1}})
	s := newRouteScratch(w)

	if w.enterable(s, 0, 1, 1) {
		t.Fatal("the fixture's start is enterable; it is meant to be blocked")
	}
	route, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 1, 1)
	if !ok || len(route) != 0 {
		t.Errorf("a search from a blocked cell to itself is %s (ok=%v), want the empty route", fmtRoute(route), ok)
	}
}

// ------------------------------------------------- the scratch

// TestTheScratchCarriesNothingBetweenSearches: the plane is reset by its own
// touch list, so a second search sees no label the first left. The two searches
// here are ones whose answers are pinned separately above, so a leak shows as
// the wrong route rather than as a wrong number nobody reads.
func TestTheScratchCarriesNothingBetweenSearches(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	detour := routeWorld(t, b, []byte{
		0, blockGround, 0,
		0, blockGround, 0,
		0, 0, 0,
	}, []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(detour)

	got, ok := detour.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 2, 0)
	checkRoute(t, "the first search", got, ok, []cell{{0, 1}, {1, 2}, {2, 1}, {2, 0}})

	// A scratch answers about the entities of ONE world, so pointing it at
	// another re-counts them. The label plane is what is under test here and it
	// is not touched by that: what the second search must not see is a label the
	// first left.
	open := routeWorld(t, b, openGrid(b), []Entity{{ID: 1, X: 2, Y: 1}})
	s.occupy(open)
	got, ok = open.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 0, 1)
	checkRoute(t, "the second search on the same scratch", got, ok, []cell{{1, 1}, {0, 1}})

	// Run the first again, on the scratch the second left: the answer is the
	// one it gave when the scratch was fresh.
	s.occupy(detour)
	got, ok = detour.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 2, 0)
	checkRoute(t, "the first search again", got, ok, []cell{{0, 1}, {1, 2}, {2, 1}, {2, 0}})
}

// TestStepCostIsTheContractsFormula — 0076 AC-5, over both arms and all eight
// deltas.
//
// The GROUND arm reads the byte it is handed; every other domain pays the flat
// 2 and 3 and the byte it is handed is IGNORED, which is what the deliberately
// absurd cost bytes on the non-ground rows are there to show — a flat arm that
// quietly read the plane would come out at 255 rather than 2.
//
// The three ground rows that are not the shipped range are the contract's own
// degenerate cases: at 1 the diagonal and the orthogonal arm are equal because
// the half truncates away, at 0 both are free, and at 255 the diagonal is 382 —
// which is why the label plane is wider than a byte.
func TestStepCostIsTheContractsFormula(t *testing.T) {
	for _, tc := range []struct {
		dom   Domain
		cost  uint8
		wantS uint64
		wantD uint64
	}{
		{DomainGround, 8, 8, 12},
		{DomainGround, 6, 6, 9},
		{DomainGround, 16, 16, 24},
		{DomainGround, 1, 1, 1},
		{DomainGround, 0, 0, 0},
		{DomainGround, 255, 255, 382},
		{DomainGhost, 8, 2, 3},
		{DomainGhost, 255, 2, 3},
		{DomainAir, 0, 2, 3},
		{DomainAir, 16, 2, 3},
	} {
		for _, d := range [][2]int32{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			if got := stepCost(tc.dom, tc.cost, d[0], d[1]); got != tc.wantS {
				t.Errorf("domain %d, cost %d, straight (%d,%d) = %d, want %d",
					uint8(tc.dom), tc.cost, d[0], d[1], got, tc.wantS)
			}
		}
		for _, d := range [][2]int32{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}} {
			if got := stepCost(tc.dom, tc.cost, d[0], d[1]); got != tc.wantD {
				t.Errorf("domain %d, cost %d, diagonal (%d,%d) = %d, want %d",
					uint8(tc.dom), tc.cost, d[0], d[1], got, tc.wantD)
			}
		}
	}
}

// TestTheScaledBudgetIsChebyshev covers the scalar's floor, the shift past it,
// and a target far enough away that the difference of two coordinates does not
// fit in the width they are stored in.
//
// The floor is nearSlack now that the scaled rule is the near search's alone.
// The far search's 5 went with the arm it fed.
func TestTheScaledBudgetIsChebyshev(t *testing.T) {
	for _, tc := range []struct {
		start, target cell
		want          int64
	}{
		{cell{0, 0}, cell{0, 0}, 3},     // D 0: the scalar's floor alone
		{cell{4, 1}, cell{0, 0}, 7},     // D 4: 4>>2 is 1, still under 3
		{cell{0, 0}, cell{19, 3}, 23},   // D 19: 19>>2 is 4, past the floor
		{cell{0, 0}, cell{20, 0}, 25},   // D 20: 20>>2 is 5
		{cell{0, 0}, cell{0, 400}, 500}, // D 400: 100 + 400
		{cell{-2147483648, 0}, cell{2147483647, 0}, 4294967295 + 4294967295>>2},
	} {
		if got := generationBudget(tc.start, tc.target, scaledBudget, true); got != tc.want {
			t.Errorf("scaled budget from (%d,%d) to (%d,%d) is %d, want %d",
				tc.start.x, tc.start.y, tc.target.x, tc.target.y, got, tc.want)
		}
	}
}

// ------------------------------------------------- the window

// winSide is the map the two runs below are measured on, and winMid the cell the
// mover stands on: far enough from every edge that a window of 8 is clipped
// nowhere, so 17 x 17 = 289 is a count and not an upper bound that the bounds
// happen to cut down.
const winSide int32 = 64
const winMid int32 = 32

// winTouched is what a search labelled: how many cells, and how far the furthest
// of them lies from the mover. It is read off the scratch's own touch list —
// the list the reset walks — so it is the work the search actually did rather
// than a count re-derived from the answer.
func winTouched(t *testing.T, w *World, s *routeScratch, centre cell) (int, int32) {
	t.Helper()
	far := int32(0)
	for _, i := range s.touched {
		x, y := int32(i)%w.bounds.Width, int32(i)/w.bounds.Width
		dx, dy := x-centre.x, y-centre.y
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dy > dx {
			dx = dy
		}
		if dx > far {
			far = dx
		}
	}
	return len(s.touched), far
}

func TestAWindowedSearchLabelsNoCellOutsideItAndNoMoreThan289(t *testing.T) {
	b := Bounds{Width: winSide, Height: winSide}
	mover := Entity{ID: 1, X: winMid, Y: winMid}
	centre := cell{winMid, winMid}

	// The optimised search over open ground, its target four cells east — the
	// furthest a sub-goal ever lies.
	open := optimisedWorld(t, b, openGrid(b), []Entity{mover})
	s := newRouteScratch(open)
	if _, ok := open.optimisedRoute(s, 0, unitRelation, dynamicWindow, exactGoal, winMid+4, winMid); !ok {
		t.Fatal("the optimised near search found no route across four cells of open ground")
	}
	got, far := winTouched(t, open, s, centre)
	if got != 289 || far != 8 {
		t.Errorf("the optimised near search labelled %d cell(s), the furthest %d away; want 289 and 8",
			got, far)
	}
	if _, ok := open.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, winMid+4, winMid); !ok {
		t.Fatal("the optimised far search found no route across four cells of open ground")
	}
	unbounded, farthest := winTouched(t, open, s, centre)
	if unbounded <= 289 || farthest <= 8 {
		t.Errorf("the optimised far search labelled %d cell(s), the furthest %d away — it is meant to be unbounded",
			unbounded, farthest)
	}

	// The wave against a target sealed inside its own window, at each of the two
	// distances a sub-goal can lie at.
	for _, tc := range []struct {
		what     string
		at       int32
		wantEdge int32
	}{
		{"four cells off, where the budget stops the flood first", 4, 7},
		{"at the window's edge, where the window stops it", 8, 8},
	} {
		tx, ty := winMid+tc.at, winMid
		grid := openGrid(b)
		for dx := int32(-1); dx <= 1; dx++ {
			for dy := int32(-1); dy <= 1; dy++ {
				if dx != 0 || dy != 0 {
					grid[(ty+dy)*winSide+(tx+dx)] = blockGround
				}
			}
		}
		sealed := routeWorld(t, b, grid, []Entity{mover})
		s := newRouteScratch(sealed)

		if route, ok := sealed.canonicalRoute(s, 0, unitRelation, dynamicWindow, scaledBudget, exactGoal, tx, ty); ok {
			t.Fatalf("%s: a route to a sealed target: %s", tc.what, fmtRoute(route))
		}
		got, far := winTouched(t, sealed, s, centre)
		if got > 289 || far > 8 {
			t.Errorf("%s: the wave's near search labelled %d cell(s), the furthest %d away; "+
				"the window admits 289 and reaches 8", tc.what, got, far)
		}
		if far != tc.wantEdge {
			t.Errorf("%s: the flood reached %d cell(s) out, want %d", tc.what, far, tc.wantEdge)
		}
		if got < 100 {
			t.Errorf("%s: the wave's near search labelled %d cell(s), which is too few to be measuring a flood",
				tc.what, got)
		}

		if route, ok := sealed.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, tx, ty); ok {
			t.Fatalf("%s: a route to a sealed target from the far arm: %s", tc.what, fmtRoute(route))
		}
		unbounded, farthest := winTouched(t, sealed, s, centre)
		if unbounded <= 289 || farthest <= 8 {
			t.Errorf("%s: the wave's far search labelled %d cell(s), the furthest %d away — "+
				"it is meant to be unbounded", tc.what, unbounded, farthest)
		}
	}
}

// TestTheNearSearchesBudgetIsStillTheScaledOne: two rules, not one formula with
// two slacks. The near search keeps the shape it had — a floor of 3 over the
// quarter of the distance — at each of the distances a sub-goal can lie at, and
// the far search's budget is the same thousand at every one of them.
//
// The last row is the one worth reading: at D 20 the two used to AGREE at 25,
// because the shift had overtaken both floors. They no longer can, which is what
// makes the far search's bound independent of how far the order pointed.
func TestTheNearSearchesBudgetIsStillTheScaledOne(t *testing.T) {
	for _, tc := range []struct {
		start, target cell
		wantNear      int64
	}{
		{cell{0, 0}, cell{0, 0}, 3},   // D 0: the slack alone
		{cell{5, 5}, cell{9, 5}, 7},   // D 4: the sub-goal's own distance
		{cell{0, 0}, cell{8, 3}, 11},  // D 8: the window's edge
		{cell{0, 0}, cell{20, 0}, 25}, // D 20: 20>>2 is 5, past the floor
	} {
		if got := generationBudget(tc.start, tc.target, scaledBudget, true); got != tc.wantNear {
			t.Errorf("the near budget from (%d,%d) to (%d,%d) is %d, want %d",
				tc.start.x, tc.start.y, tc.target.x, tc.target.y, got, tc.wantNear)
		}
		if got := generationBudget(tc.start, tc.target, flatBudget, true); got != flatGenerations {
			t.Errorf("the far budget from (%d,%d) to (%d,%d) is %d, want the flat %d",
				tc.start.x, tc.start.y, tc.target.x, tc.target.y, got, flatGenerations)
		}
	}
}
