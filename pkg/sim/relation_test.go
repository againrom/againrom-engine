package sim

// The two relations, driven directly.
//
// A route is produced by one of two searches, and what tells them apart in kind
// is the relation they read: terrain alone, or terrain and occupancy as it
// stands. Both searches take that relation as a parameter, so the pair below is
// four combinations and not two — and each of the four is worked out by hand on
// a 3x3 whose whole label plane fits in a comment.
//
// The searches are called directly. A tick asks for the unit-aware relation and
// for no other, so a test that went through Step could not reach the terrain arm
// at all.

import "testing"

// ------------------------------------------------- the predicate

// TestTheTwoRelationsDifferExactlyOnOccupancy is the offer both searches make,
// asked one cell at a time. The two relations are nested — the unit-aware one is
// the terrain one plus a count — so the only cells they can disagree about are
// the ones a unit stands on, and this table says so cell by cell.
//
//	. . .        (1,1) blocks ground; unit 1 stands on (2,0) and unit 2, the
//	. # .        asker throughout, on (0,0).
//	. . .
func TestTheTwoRelationsDifferExactlyOnOccupancy(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	grid := openGrid(b)
	grid[1*3+1] = blockGround
	w := routeWorld(t, b, grid, []Entity{
		{ID: 1, X: 2, Y: 0},
		{ID: 2, X: 0, Y: 0},
	})
	s := newRouteScratch(w)

	const asker = 1 // the unit at (0,0): the slice is kept in ascending id
	for _, tc := range []struct {
		what          string
		x, y          int32
		terrain, unit bool
	}{
		{"an empty passable cell", 1, 2, true, true},
		{"a cell blocking ground", 1, 1, false, false},
		{"a cell off the map", 5, 5, false, false},
		{"a cell to the north of the map", 0, -1, false, false},
		{"a cell another unit stands on", 2, 0, true, false},
		{"the asker's own cell", 0, 0, true, true},
	} {
		if got := w.terrainOpen(DomainGround, tc.x, tc.y); got != tc.terrain {
			t.Errorf("%s: terrainOpen(%d,%d) = %v, want %v", tc.what, tc.x, tc.y, got, tc.terrain)
		}
		if got := w.enterable(s, asker, tc.x, tc.y); got != tc.unit {
			t.Errorf("%s: enterable(%d,%d) = %v, want %v", tc.what, tc.x, tc.y, got, tc.unit)
		}
		// And the one dispatch both searches go through answers each of them.
		if got := w.open(s, terrainRelation, asker, tc.x, tc.y); got != tc.terrain {
			t.Errorf("%s: open over terrain = %v, want %v", tc.what, got, tc.terrain)
		}
		if got := w.open(s, unitRelation, asker, tc.x, tc.y); got != tc.unit {
			t.Errorf("%s: open over units = %v, want %v", tc.what, got, tc.unit)
		}
	}
}

// ------------------------------------------------- the wave over either relation

// TestTheWaveRunsOverTheRelationItIsGiven is the 3x3 of route_test.go with a
// unit parked on (1,1), searched twice.
//
// Over the unit-aware relation (1,1) is never labelled and the route goes round
// it — that is TestAnotherUnitIsNotEnterable's pinned answer. Over the terrain
// relation the parked unit is not there to be seen, so the plane is the empty
// map's and the route is the straight one TestTwoFrontierCellsReachingOne-
// Neighbour pins on a world with no unit at (1,1) at all.
//
//	5 4 5        the terrain arm's plane: (1,1) at 2, the target (0,1) at 4
//	3 2 3
//	2 0 2        (column x, row y; the start's 0 at (2,1))
func TestTheWaveRunsOverTheRelationItIsGiven(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	w := routeWorld(t, b, openGrid(b), []Entity{
		{ID: 1, X: 1, Y: 1},
		{ID: 2, X: 2, Y: 1},
	})
	s := newRouteScratch(w)
	start := cell{2, 1}

	got, ok := w.canonicalRoute(s, 1, unitRelation, noWindow, flatBudget, exactGoal, 0, 1)
	checkRoute(t, "the wave over terrain and units", got, ok, []cell{{1, 0}, {0, 1}})
	if _, labelled := w.label(s, start, 1, 1); labelled {
		t.Error("(1,1) is labelled under the unit-aware relation, but a unit stands on it")
	}

	got, ok = w.canonicalRoute(s, 1, terrainRelation, noWindow, flatBudget, exactGoal, 0, 1)
	checkRoute(t, "the wave over terrain alone", got, ok, []cell{{1, 1}, {0, 1}})
	checkLabel(t, w, s, start, 1, 1, 8)
	checkLabel(t, w, s, start, 0, 1, 16)
}

// TestTheOptimisedSearchRunsOverTheRelationItIsGiven is the same world and the
// same order under the other mode, where the difference is wider because the
// corner rule asks the same relation the cell test asks.
//
// Over TERRAIN the costs to (0,1) are
//
//	8 12 20     so the start (2,1) is 16 through (1,1), and the greedy walk
//	0  8 16     forward takes (1,1) then the target.
//	8 12 20
//
// Over TERRAIN AND UNITS the parked unit closes (1,1), and with it every
// diagonal that passes beside it: (0,1)->(1,0), (0,1)->(1,2), (1,0)->(2,1) and
// (1,2)->(2,1) are all refused at the corner. What is left is the ring —
//
//	8 16 24     the start is 32, reached by two equal-cost first steps, (2,0)
//	0  . 32     and (2,2). Compared as (y, x) the first orders before the
//	8 16 24     second, so the route runs north about.
func TestTheOptimisedSearchRunsOverTheRelationItIsGiven(t *testing.T) {
	b := Bounds{Width: 3, Height: 3}
	w := optimisedWorld(t, b, openGrid(b), []Entity{
		{ID: 1, X: 1, Y: 1},
		{ID: 2, X: 2, Y: 1},
	})
	s := newRouteScratch(w)
	start := cell{2, 1}

	got, ok := w.optimisedRoute(s, 1, terrainRelation, noWindow, settleOrdered, 0, 1)
	checkRoute(t, "the optimised search over terrain alone", got, ok, []cell{{1, 1}, {0, 1}})
	checkLabel(t, w, s, start, 1, 1, 8)
	checkLabel(t, w, s, start, 2, 1, 16)

	got, ok = w.optimisedRoute(s, 1, unitRelation, noWindow, exactGoal, 0, 1)
	checkRoute(t, "the optimised search over terrain and units", got, ok,
		[]cell{{2, 0}, {1, 0}, {0, 0}, {0, 1}})
	checkLabel(t, w, s, start, 2, 0, 24)
	checkLabel(t, w, s, start, 2, 1, 32)
	if _, labelled := w.label(s, start, 1, 1); labelled {
		t.Error("(1,1) holds a cost under the unit-aware relation, but a unit stands on it")
	}
	// The corner rule reads the relation it was given, so the same diagonal is
	// allowed under one and refused under the other.
	if !w.cornerClear(s, terrainRelation, 1, 0, 1, 1, -1) {
		t.Error("(0,1)->(1,0) is refused over terrain, where nothing blocks either corner cell")
	}
	if w.cornerClear(s, unitRelation, 1, 0, 1, 1, -1) {
		t.Error("(0,1)->(1,0) is allowed over units, but (1,1) is one of the cells it passes between")
	}
}

// ------------------------------------------------- the arms against each other

// relCorpus is the seeded corpus below: worlds carrying exactly ONE entity, over
// grids of varied size and blocked fraction. One entity is the whole point — the
// unit-aware relation subtracts the asker's own presence, so with nobody else on
// the map the two relations are the same predicate cell for cell, and any route
// they disagree about is a search reading something other than what it was
// handed.
func relCorpus(r *rng) (Bounds, []byte, Entity) {
	b := Bounds{Width: 4 + int32(r.next()%13), Height: 4 + int32(r.next()%13)}
	blocked := 5 + r.next()%30 // per cent
	grid := make([]byte, b.Width*b.Height)
	for i := range grid {
		if r.next()%100 < blocked {
			grid[i] = blockGround
		}
	}
	return b, grid, Entity{
		ID: 1,
		X:  int32(r.next() % uint64(b.Width)),
		Y:  int32(r.next() % uint64(b.Height)),
	}
}

// TestWithNoOtherUnitOnTheMapTheTwoArmsAgreeCellForCell is the other half of the
// claim: the unit-aware arm still reproduces what this package already pins.
// Every route pinned elsewhere in this package is asked for over that arm, so
// those cases carry it for the fixtures they name; this one carries it over a
// corpus, in both modes, and compares the two arms cell by cell rather than by
// length or by cost.
//
// It asserts the corpus produced routes and refusals both, so an agreement that
// came from two searches failing everywhere would not read as a pass.
func TestWithNoOtherUnitOnTheMapTheTwoArmsAgreeCellForCell(t *testing.T) {
	const worlds = 600
	r := &rng{state: 0x5e1ec7}

	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		routed, refused := 0, 0
		for k := 0; k < worlds; k++ {
			b, grid, e := relCorpus(r)
			w := mustWorldGrid(t, 1, b, mode, grid, []Entity{e})
			s := newRouteScratch(w)
			tx, ty := int32(r.next()%uint64(b.Width)), int32(r.next()%uint64(b.Height))

			terrain, tok := w.searchRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, tx, ty)
			units, uok := w.searchRoute(s, 0, unitRelation, noWindow, flatBudget, exactGoal, tx, ty)
			if tok != uok {
				t.Fatalf("mode %d, world %d: (%d,%d) to (%d,%d) is ok=%v over terrain and ok=%v over units",
					mode, k, e.X, e.Y, tx, ty, tok, uok)
			}
			if !tok {
				refused++
				continue
			}
			routed++
			checkRoute(t, "the unit-aware arm on a map holding one unit", units, uok, terrain)
		}
		if routed == 0 || refused == 0 {
			t.Errorf("mode %d: %d routed and %d refused — the corpus has to hold both", mode, routed, refused)
		}
	}
}
