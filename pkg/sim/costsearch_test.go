package sim

import "testing"

// The cost plane REACHING the search: which mover is charged the ground,
// which cell of a step is charged, and what cost can and cannot do to a
// route.

// csWorld is a world over b whose block plane is grid and whose cost plane is
// cost, with one entity of the given domain at start.
func csWorld(t *testing.T, b Bounds, mode Mode, grid, cost []byte, dom Domain, x, y int32) *World {
	t.Helper()
	w, err := NewTerrainWorld(1, b, mode, Terrain{Block: grid, Cost: cost},
		[]Entity{{ID: 1, X: x, Y: y, Domain: dom}}, nil)
	if err != nil {
		t.Fatalf("NewTerrainWorld: %v", err)
	}
	return w
}

// csCorridors is the fixture the two arms are separated on: a 7x3 with the
// middle row blocked between the ends, so the only two ways from (0,1) to (6,1)
// are the north row and the south row and they are EXACTLY the same length in
// cells.
//
//	. . . . . . .   row 0, the north corridor
//	. # # # # # .   row 1, the start and target with a wall between them
//	. . . . . . .   row 2, the south corridor
//
// Equal length is the whole design. A cheaper route that is longer in cells is
// never reached at all (see TestACheaperRouteLongerInCellsIsNotTaken), so a
// fixture whose corridors differed in length would be measuring the wave's stop
// rule and not the cost arm.
var csBounds = Bounds{Width: 7, Height: 3}

func csGrid() []byte {
	g := openGrid(csBounds)
	for x := int32(1); x <= 5; x++ {
		g[1*7+x] = blockGround | blockAir
	}
	return g
}

// csCost is a cost plane charging north and south differently: cheap is 6 and
// dear is 16, both values the shipped corpus carries.
func csCost(cheapRow int32) []byte {
	c := make([]byte, 7*3)
	for i := range c {
		c[i] = 16
	}
	for x := int32(0); x < 7; x++ {
		c[cheapRow*7+x] = 6
	}
	return c
}

func csRoute(t *testing.T, w *World) []cell {
	t.Helper()
	s := newRouteScratch(w)
	got, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 6, 1)
	if !ok {
		t.Fatalf("no route")
	}
	return got
}

// csRow is which corridor a route went through: the row every cell of it that is
// not an endpoint lies on.
func csRow(t *testing.T, r []cell) int32 {
	t.Helper()
	for _, c := range r {
		if c.x > 0 && c.x < 6 {
			return c.y
		}
	}
	t.Fatalf("the route %s touches neither corridor", fmtRoute(r))
	return -1
}

// TestAGroundMoverTakesTheCheaperOfTwoEqualCorridors — AC-2.
//
// The claim is measured by MIRRORING the plane rather than by predicting a
// tie-break: the same world, the same order, two cost planes that differ only in
// which corridor is cheap, and two different routes. A search that read no cost
// byte would answer the same way twice, and the tie-break — whatever it is —
// cannot be what chose, because it did not move.
func TestAGroundMoverTakesTheCheaperOfTwoEqualCorridors(t *testing.T) {
	t.Parallel()

	north := csRow(t, csRoute(t, csWorld(t, csBounds, ModeCanonical, csGrid(), csCost(0), DomainGround, 0, 1)))
	south := csRow(t, csRoute(t, csWorld(t, csBounds, ModeCanonical, csGrid(), csCost(2), DomainGround, 0, 1)))

	if north != 0 {
		t.Errorf("with the north corridor cheap the mover went through row %d, want row 0", north)
	}
	if south != 2 {
		t.Errorf("with the south corridor cheap the mover went through row %d, want row 2", south)
	}
}

// TestANonGroundMoverReadsNoCostByte — AC-4.
//
// The same two mirrored planes under a mover of each of the other two domains:
// the route does not move, and it is the route the same mover takes when every
// cost byte is equal. Both halves are needed — "did not move" alone would be
// satisfied by a mover that read the plane and happened to tie.
func TestANonGroundMoverReadsNoCostByte(t *testing.T) {
	t.Parallel()

	for _, dom := range []Domain{DomainGhost, DomainAir} {
		// The block plane's air bit closes the wall to these movers too, so all
		// three domains face the same two corridors.
		uniform := csRow(t, csRoute(t, csWorld(t, csBounds, ModeCanonical, csGrid(), nil, dom, 0, 1)))
		north := csRow(t, csRoute(t, csWorld(t, csBounds, ModeCanonical, csGrid(), csCost(0), dom, 0, 1)))
		south := csRow(t, csRoute(t, csWorld(t, csBounds, ModeCanonical, csGrid(), csCost(2), dom, 0, 1)))

		if north != uniform || south != uniform {
			t.Errorf("domain %d went through rows %d and %d over the two mirrored planes, "+
				"and row %d over a uniform one — it is reading the cost plane",
				uint8(dom), north, south, uniform)
		}
	}
}

// TestCostChangesLabelsAndNotReach — AC-2a, and the precise form of what cost
// cannot do.
//
// The wave advances ONE RING PER GENERATION whatever the ground costs — an
// unlabelled cell beside a labelled one always improves, since it holds no label
// at all — and it stops in the first generation that labels the goal. So the SET
// of cells a search labels is a property of the map and the order, and the cost
// plane moves only the numbers written into it. A cheaper route the wave never
// reached is therefore not found, however cheap it is.
//
// The set is read off the scratch's own touch list, which is the only record of
// what a search wrote, and it is compared between a uniform plane and a plane
// with 6s and 16s scattered over it.
func TestCostChangesLabelsAndNotReach(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 9, Height: 7}
	grid := openGrid(b)
	// A wall with one gap, so the reachable set is an interesting shape rather
	// than a square and a difference would have somewhere to show.
	for y := int32(0); y < 7; y++ {
		if y != 5 {
			grid[y*9+4] = blockGround
		}
	}
	varied := make([]byte, 9*7)
	for i := range varied {
		if i%3 == 0 {
			varied[i] = 6
		} else {
			varied[i] = 16
		}
	}

	touched := func(cost []byte) (map[int]uint64, int) {
		w := csWorld(t, b, ModeCanonical, grid, cost, DomainGround, 0, 3)
		s := newRouteScratch(w)
		if _, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 8, 3); !ok {
			t.Fatal("no route")
		}
		out := make(map[int]uint64, len(s.touched))
		for _, i := range s.touched {
			out[i] = s.plane[i]
		}
		return out, len(s.touched)
	}

	uniform, nU := touched(nil)
	mixed, nM := touched(varied)

	if nU != nM {
		t.Errorf("the two searches touched %d and %d cell(s)", nU, nM)
	}
	differing := 0
	for i, lu := range uniform {
		lm, ok := mixed[i]
		if !ok {
			t.Errorf("cell %d was labelled over the uniform plane and not over the mixed one", i)
			continue
		}
		if lu != lm {
			differing++
		}
	}
	for i := range mixed {
		if _, ok := uniform[i]; !ok {
			t.Errorf("cell %d was labelled over the mixed plane and not over the uniform one", i)
		}
	}
	// And the labels DID move, so the equality above is a fact about reach and
	// not about a cost plane that reached nobody.
	if differing == 0 {
		t.Errorf("the two planes produced identical labels on all %d cell(s) — "+
			"the cost plane is not being read", nU)
	}
}

// TestEachStepIsChargedAtTheCellItEnters — AC-6, in BOTH modes.
//
// One row of five cells with five different costs, so every label is a distinct
// number and the two readings of "which cell does a step pay for" give different
// planes rather than the same plane by coincidence.
//
//	cost   1  2  3  4  5     at x = 0..4
//
// The canonical wave floods FORWARD from (0,0), charging the cell entered: its
// labels are the running sum of the costs to the RIGHT of the start. The
// optimised flood runs BACKWARD from (4,0) and charges the same cell — which
// from its own direction of travel is the one being expanded from — so its
// labels are the running sum taken the other way. Charging the wrong cell in
// either shifts that mode's whole plane by one column.
func TestEachStepIsChargedAtTheCellItEnters(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 5, Height: 1}
	cost := []byte{1, 2, 3, 4, 5}

	t.Run("the canonical wave, flooding forward", func(t *testing.T) {
		w := csWorld(t, b, ModeCanonical, openGrid(b), cost, DomainGround, 0, 0)
		s := newRouteScratch(w)
		if _, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 4, 0); !ok {
			t.Fatal("no route")
		}
		start := cell{0, 0}
		// 0, then +2, +3, +4, +5 — the cost of each cell as it is entered.
		for x, want := range []uint64{0, 2, 5, 9, 14} {
			checkLabel(t, w, s, start, int32(x), 0, want)
		}
	})

	t.Run("the optimised flood, running backward", func(t *testing.T) {
		w := csWorld(t, b, ModeOptimised, openGrid(b), cost, DomainGround, 0, 0)
		s := newRouteScratch(w)
		if _, ok := w.optimisedRoute(s, 0, unitRelation, noWindow, exactGoal, 4, 0); !ok {
			t.Fatal("no route")
		}
		start := cell{0, 0}
		// The cost from each cell TO the target, charging the entered cell
		// again: 5 from (3,0), then +4, +3, +2 walking left.
		for x, want := range []uint64{14, 12, 9, 5, 0} {
			checkLabel(t, w, s, start, int32(x), 0, want)
		}
	})
}

func TestAZeroCostPlaneEndsTheWalkInsteadOfCyclingIt(t *testing.T) {
	t.Parallel()

	b := Bounds{Width: 5, Height: 1}
	w := csWorld(t, b, ModeCanonical, openGrid(b), make([]byte, 5), DomainGround, 0, 0)
	s := newRouteScratch(w)

	// The wave itself terminates either way: at a step cost of zero no label
	// ever improves after the first, so no cell is ever re-expanded. It is the
	// walk back out of those labels that has nothing to descend.
	got, ok := w.canonicalRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, 4, 0)
	if ok {
		t.Errorf("a route came back over an all-zero cost plane: %s — every label ties, so "+
			"no walk out of them can be the cheapest anything", fmtRoute(got))
	}
	// And the mover is left where a search that finds nothing leaves it, rather
	// than half-advanced along a walk that was cut off.
	if e := w.Entities()[0]; e.X != 0 || e.Y != 0 {
		t.Errorf("the mover stands at (%d,%d), want its start", e.X, e.Y)
	}
}
