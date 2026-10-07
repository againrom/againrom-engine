package sim

// The wall with one gap, and the target off the map.
//
// Both are runs rather than searches: what is measured is where a unit STANDS
// after each tick, never which cells a search picked out. A route is searched
// afresh every tick from wherever the unit now is, so a walk is the contract's
// answer repeated, and pinning the first answer would say nothing about the
// tenth.

import "testing"

// checkP1 is the invariant, checked over EVERY unit after EVERY tick: in bounds,
// on a cell whose blocks-ground bit is clear, and alone on it. Checked at the end
// of a run instead, it would pass a walk that crossed the wall and came back.
func checkP1(t *testing.T, w *World, b Bounds, grid []byte, tick int) {
	t.Helper()
	ents := w.Entities()
	for i, e := range ents {
		if e.X < 0 || e.Y < 0 || e.X >= b.Width || e.Y >= b.Height {
			t.Fatalf("tick %d: entity %d is at (%d,%d), outside %dx%d", tick, e.ID, e.X, e.Y, b.Width, b.Height)
		}
		if grid[e.Y*b.Width+e.X]&blockGround != 0 {
			t.Fatalf("tick %d: entity %d stands on the blocked cell (%d,%d)", tick, e.ID, e.X, e.Y)
		}
		for j := i + 1; j < len(ents); j++ {
			if ents[j].X == e.X && ents[j].Y == e.Y {
				t.Fatalf("tick %d: entities %d and %d share (%d,%d)", tick, e.ID, ents[j].ID, e.X, e.Y)
			}
		}
	}
}

// The wall fixture: a nine by nine with column 4 blocking ground from top to
// bottom but for one cell, and a unit ordered from one side to the other.
//
//	@ . . . # . . . T        The gap is at (4,4), the crossing unit starts at
//	. . . . # . . . .        (0,0) and is ordered to (8,0). A second unit is
//	. . . . # . . . .        parked at (2,4) with no order of its own, so the
//	. . . . # . . . .        invariant's "alone on it" clause has something to
//	. . x . _ . . . .        be true of and the corridor is not empty.
//	. . . . # . . . .
//	. . . . # . . . .        The gap sits on the crossing's own row, so the
//	. . . . # . . . .        detour through it costs no more hops than the
//	. . . . # . . . .        straight line between the two sides: this fixture
//	                         is about the wall and the gap, and not about how
//	                         much room a search is given to leave a line.
var wallBounds = Bounds{Width: 9, Height: 9}

func wallGrid() []byte {
	g := make([]byte, 81)
	for y := int32(0); y < 9; y++ {
		if y != 4 {
			g[y*9+4] = blockGround
		}
	}
	return g
}

func wallUnits() []Entity {
	return []Entity{{ID: 1, X: 0, Y: 0}, {ID: 2, X: 2, Y: 4}}
}

const wallTargetX, wallTargetY = 8, 0

// TestTheFixtureFitsBothModesBounds is SC-3's condition on the fixture itself.
// A wall that defeats one mode is a fixture defect, so what could defeat one is
// asserted before the walk rather than diagnosed after it.
//
// What is asserted is the DETOUR'S OWN HOP COUNT and not an allowance. An
// allowance is a property of whatever budget rule the tree carries this week, so
// a fixture asserting one has to be re-derived every time that rule moves, and
// says nothing about the fixture either way. The hop count is the fixture's.
func TestTheFixtureFitsBothModesBounds(t *testing.T) {
	w := mustWorldGrid(t, 1, wallBounds, ModeCanonical, wallGrid(), wallUnits())
	s := newRouteScratch(w)

	// Ten hops through the gap against a Chebyshev distance of eight, the two
	// extra being the corner rule making the search enter and leave the gap
	// orthogonally — the same two the detour fixture below pays. The wall costs
	// this crossing nothing else, because the gap lies on the row a straight run
	// would take anyway.
	route, ok := w.optimisedRoute(s, 0, terrainRelation, noWindow, settleOrdered, wallTargetX, wallTargetY)
	if !ok {
		t.Fatal("no route through the gap at all: the fixture is a wall without a gap")
	}
	if len(route) != 10 {
		t.Errorf("the detour through the gap is %d hops, want 10: %s", len(route), fmtRoute(route))
	}
}

func TestAUnitCrossesTheWallThroughItsGapUnderBothModes(t *testing.T) {
	grid := wallGrid()
	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, wallBounds, mode, wallGrid(), wallUnits())

		arrived := 0
		for tick := 1; tick <= 24 && arrived == 0; tick++ {
			if tick == 1 {
				Step(w, []Command{{Entity: 1, X: wallTargetX, Y: wallTargetY}})
			} else {
				Step(w, nil)
			}
			checkP1(t, w, wallBounds, grid, tick)

			e := w.Entities()[0]
			if e.X == wallTargetX && e.Y == wallTargetY {
				if e.HasTarget || e.TargetX != 0 || e.TargetY != 0 || e.Stall != 0 {
					t.Errorf("mode %d: the unit arrived as %+v, want its target cleared with no residue", mode, e)
				}
				arrived = tick
			}
		}

		if arrived == 0 {
			t.Fatalf("mode %d: the unit is at %+v after 24 ticks and has not crossed", mode, w.Entities()[0])
		}
		// A tick moves a unit at most one cell on each axis, so no crossing can
		// beat the Chebyshev distance between the two sides.
		if arrived < 8 {
			t.Errorf("mode %d: the unit arrived at tick %d, and 8 is the fewest a crossing can take", mode, arrived)
		}
		// The parked unit is not a mover: it holds no order, so nothing about it
		// changed while the other walked past it.
		if got := w.Entities()[1]; got != (Entity{ID: 2, X: 2, Y: 4, ActorState: actorStateGuard, Reach: 1, PostX: 2, PostY: 4}) {
			t.Errorf("mode %d: the parked unit is %+v, want it where it started", mode, got)
		}
	}
}

// TestATargetOffTheMapIsWalkedTowardOrEndedByMode is AC-2's second case, and it
// is renamed because its old name — ...EndsTheOrderOnTheFirstTickUnderBothModes
// — asserts what is no longer true of one of the two modes (0045 verification
// cites the old name; this is the same test).
//
// An order off the map is not refused when it is given — a target may name
// any cell — and nothing off the map is open to either relation, so
// neither search ever labels it. What the two modes then do differs: the
// canonical wave SETTLES for the labelled cell nearest what was asked for
// and the mover walks to the edge, while the optimised search, whose plane
// is built outward from the target and so holds nothing at all when the
// target is refused, ends the order in that tick as before.
func TestATargetOffTheMapIsWalkedTowardOrEndedByMode(t *testing.T) {
	b := Bounds{Width: 5, Height: 5}
	grid := openGrid(b)

	for _, tc := range []struct {
		target cell
		// settles is where the canonical mover comes to rest; the optimised one
		// never leaves (2, 2).
		settles cell
	}{
		{cell{7, 2}, cell{4, 2}},
		{cell{-3, 1}, cell{0, 2}},
	} {
		for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
			w := mustWorldGrid(t, 1, b, mode, openGrid(b), []Entity{{ID: 1, X: 2, Y: 2}})

			rest := cell{2, 2}
			if mode == ModeCanonical {
				rest = tc.settles
			}
			arrivedBy := int(rest.chebyshevTo(cell{2, 2})) + 1

			for tick := 1; tick <= stallLimit; tick++ {
				if tick == 1 {
					Step(w, []Command{{Entity: 1, X: tc.target.x, Y: tc.target.y}})
				} else {
					Step(w, nil)
				}
				checkP1(t, w, b, grid, tick)
				if tick < arrivedBy {
					continue
				}

				// THE FACING IS DROPPED FROM THIS COMPARISON and asserted
				// nowhere here. Every other field is checked whole, which is what
				// makes this a no-residue case; the facing is not residue, it
				// depends on which cell each MODE settled from, and that choice is
				// this case's subject. It is asserted against the step that
				// produced it in arrival_test.go, over all eight directions.
				settled := w.Entities()[0]
				settled.Facing = 0
				settled.DesiredFacing = 0
				if got := (Entity{ID: 1, X: rest.x, Y: rest.y, ActorState: actorStateGuard, Reach: 1,
					PostX: 2, PostY: 2}); settled != got {
					t.Fatalf("target (%d,%d) mode %d tick %d: the unit is %+v, want %+v — settled, with "+
						"no residue and no stall",
						tc.target.x, tc.target.y, mode, tick, settled, got)
				}
				if len(w.routes[0]) != 0 {
					t.Errorf("target (%d,%d) mode %d tick %d: the unit holds a route with no target",
						tc.target.x, tc.target.y, mode, tick)
				}
			}
		}
	}
}

// ------------------------------------------------- the gap no rectangle contains

// The detour fixture, and SC-4's discriminator: a 24 x 41 map with row 20
// blocking ground from edge to edge but for one cell at (20,20), and a unit at
// (0,0) ordered to (0,40).
//
//	@ . . . . . . .          The order's own rectangle — the smallest one
//	. . . . . . . .          containing (0,0) and (0,40) — is one column wide,
//	# # # # # # # # ... _    and grown by eight on each side it reaches x = 8.
//	. . . . . . . .          The gap is at x = 20, so no route to this target
//	T . . . . . . .          fits inside it and the walk is not a near thing:
//	                         it misses by twelve columns.
//
// The detour itself is cheap in hops: the diagonal run down to the gap and back
// out is forty steps, the Chebyshev distance between the two cells, and the
// optimised search takes forty-two because the corner rule makes it enter and
// leave the gap orthogonally. So what this fixture discriminates is the
// RECTANGLE and nothing else — it is not near any budget's edge, and it is not
// the fixture the budget stop is witnessed on.
var detourBounds = Bounds{Width: 24, Height: 41}

const detourGapX, detourGapY int32 = 20, 20
const detourTargetX, detourTargetY int32 = 0, 40

func detourGrid() []byte {
	g := make([]byte, detourBounds.Width*detourBounds.Height)
	for x := int32(0); x < detourBounds.Width; x++ {
		if x != detourGapX {
			g[detourGapY*detourBounds.Width+x] = blockGround
		}
	}
	return g
}

// TestTheDetourLeavesAnyStartTargetRectangleGrownByEight is the fixture's own
// claim, asserted before the walk: every route to this target leaves that
// rectangle, because every route passes through the one gap and the gap is
// outside it. A fixture whose gap happened to sit inside would make the walk
// below pass under a bounded search too, and prove nothing.
func TestTheDetourLeavesAnyStartTargetRectangleGrownByEight(t *testing.T) {
	const margin int32 = 8
	minX, maxX := int32(0), detourTargetX
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if lo, hi := minX-margin, maxX+margin; detourGapX >= lo && detourGapX <= hi {
		t.Fatalf("the gap at x=%d lies inside x in [%d,%d], so this fixture bounds nothing",
			detourGapX, lo, hi)
	}

	// And the wall really is a wall: row 20 is blocked at every column but the
	// gap, so no route reaches row 21 any other way.
	g := detourGrid()
	for x := int32(0); x < detourBounds.Width; x++ {
		blocked := g[detourGapY*detourBounds.Width+x]&blockGround != 0
		if want := x != detourGapX; blocked != want {
			t.Fatalf("(%d,%d) blocked = %v, want %v", x, detourGapY, blocked, want)
		}
	}

	// And the detour's own length, in place of the allowance this used to name:
	// forty-two hops for a crossing whose two cells are forty apart. A fixture
	// whose detour were long would measure whatever bound the wave carries
	// rather than the rectangle's removal, and this one is two hops over the
	// straight line.
	w := mustWorldGrid(t, 1, detourBounds, ModeCanonical, detourGrid(), []Entity{{ID: 1, X: 0, Y: 0}})
	route, ok := w.optimisedRoute(newRouteScratch(w), 0, terrainRelation, noWindow, settleOrdered, detourTargetX, detourTargetY)
	if !ok {
		t.Fatal("no route through the gap at all: the fixture is a wall without a gap")
	}
	if len(route) != 42 {
		t.Errorf("the detour is %d hops, want 42", len(route))
	}
}

func TestAUnitWalksToAGoalNoRectangleReaches(t *testing.T) {
	grid := detourGrid()
	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, detourBounds, mode, detourGrid(), []Entity{{ID: 1, X: 0, Y: 0}})

		arrived := 0
		for tick := 1; tick <= 64 && arrived == 0; tick++ {
			if tick == 1 {
				Step(w, []Command{{Entity: 1, X: detourTargetX, Y: detourTargetY}})
			} else {
				Step(w, nil)
			}
			checkP1(t, w, detourBounds, grid, tick)

			e := w.Entities()[0]
			if e.X == detourTargetX && e.Y == detourTargetY {
				if e.HasTarget || e.Stall != 0 {
					t.Errorf("mode %d: the unit arrived as %+v, want its target cleared with no residue", mode, e)
				}
				arrived = tick
			}
			if !e.HasTarget && arrived == 0 {
				t.Fatalf("mode %d: the order was given up at tick %d with the unit at (%d,%d)",
					mode, tick, e.X, e.Y)
			}
		}
		if arrived == 0 {
			t.Fatalf("mode %d: the unit is at %+v after 64 ticks and has not crossed", mode, w.Entities()[0])
		}
		// Forty is the Chebyshev distance, so nothing can arrive sooner, and the
		// walk really did go the long way round: it passed through the gap's own
		// column, twenty columns from both ends.
		if arrived < 40 {
			t.Errorf("mode %d: the unit arrived at tick %d, and 40 is the fewest a crossing can take", mode, arrived)
		}
	}
}
