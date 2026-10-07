package sim

// The order that could not be given, and the one this revision still refuses.
//
// The owner ordered a unit across a river where a land route exists. Up, across
// and down as three orders walks; the far bank clicked once moved nobody. It was
// never about water: the far search's budget was a function of the straight
// line, so the whole allowance for LEAVING that line was a quarter of the
// distance ALONG it, and a route that goes round rather than through needs about
// twice the crossing leg in rings the straight line does not use.
//
// This file is that order, in the tree. The fixture asserts its own claims
// first — the crossing really is sealed, the way round really is longer in rings
// than the old allowance — because a green arrival over a fixture that happened
// to fit would be a formality rather than evidence. Then the OLD budget is
// driven from the test, so the refusal it used to give is witnessed here and not
// recalled from a document.
//
// And the residual is witnessed beside the fix. A thousand generations is a
// real bound: a route longer than that is still refused in canonical mode
// where the optimised search, which counts no generations at all, still
// finds it.
//
// The channel fixture itself is chanBounds/chanGrid in budget_test.go, where
// AC-14's recorded run needs it too. Nothing here reads a clock.

import "testing"

// TestTheChannelFixtureAssertsItsOwnClaim is SC-12's condition on the fixture:
// what makes the arrival below evidence rather than a fixture that happens to
// fit. Three things are asserted, and each of them could be false.
func TestTheChannelFixtureAssertsItsOwnClaim(t *testing.T) {
	grid := chanGrid()

	// One: the crossing really is sealed. Every cell of column 20 above row 30
	// blocks ground, so no route crosses the channel except round its end.
	for y := int32(0); y < chanOpenFrom; y++ {
		if grid[y*chanBounds.Width+chanWallX]&blockGround == 0 {
			t.Fatalf("(%d,%d) is open, so the channel has a second crossing and this fixture bounds nothing",
				chanWallX, y)
		}
	}
	for y := chanOpenFrom; y < chanBounds.Height; y++ {
		if grid[y*chanBounds.Width+chanWallX]&blockGround != 0 {
			t.Fatalf("(%d,%d) is blocked, so there is no way round at all and this fixture is sealed",
				chanWallX, y)
		}
	}

	// Two: the way round is longer in rings than the old allowance. The straight
	// line is 39 cells; the only land route is 62 hops, so it leaves that line
	// for 23 rings where max(5, 39>>2) allowed 9.
	w := mustWorldGrid(t, 1, chanBounds, ModeCanonical, grid, []Entity{{ID: 1, X: 0, Y: 0}})
	route, ok := w.optimisedRoute(newRouteScratch(w), 0, terrainRelation, noWindow, settleOrdered, chanTargetX, chanTargetY)
	if !ok {
		t.Fatal("no land route round the channel at all: the fixture is sealed, not long")
	}
	const straight = int64(chanTargetX) // (0,0) to (39,0), Chebyshev
	detour := int64(len(route)) - straight
	oldAllowance := int64(5)
	if q := straight >> 2; q > oldAllowance {
		oldAllowance = q
	}
	if detour <= oldAllowance {
		t.Fatalf("the way round is %d hops against a straight line of %d, so it leaves that line for %d "+
			"rings where the old allowance was %d — this fixture does not discriminate",
			len(route), straight, detour, oldAllowance)
	}
	t.Logf("the way round is %d hops: %d rings off a straight line of %d, where the old allowance was %d",
		len(route), detour, straight, oldAllowance)

	// Three: the target is not somewhere a mover could not stand anyway, which
	// would refuse before any sweep and measure nothing about a budget.
	if !w.terrainOpen(DomainGround, chanTargetX, chanTargetY) {
		t.Fatalf("the target (%d,%d) blocks ground", chanTargetX, chanTargetY)
	}
}

// TestOneOrderCarriesAUnitAcrossTheChannelInBothModes is AC-12's first half and
// the defect the owner reported, gone: ONE order, and the unit walks.
func TestOneOrderCarriesAUnitAcrossTheChannelInBothModes(t *testing.T) {
	grid := chanGrid()

	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, chanBounds, mode, chanGrid(), []Entity{{ID: 1, X: 0, Y: 0}})

		arrived := 0
		for tick := 1; tick <= 128 && arrived == 0; tick++ {
			if tick == 1 {
				Step(w, []Command{{Entity: 1, X: chanTargetX, Y: chanTargetY}})
			} else {
				Step(w, nil)
			}
			checkP1(t, w, chanBounds, grid, tick)

			e := w.Entities()[0]
			if !e.HasTarget && !(e.X == chanTargetX && e.Y == chanTargetY) {
				t.Fatalf("mode %d: the order was given up at tick %d with the unit at (%d,%d) — "+
					"this is the defect, and it is what the flat budget removes", mode, tick, e.X, e.Y)
			}
			if e.X == chanTargetX && e.Y == chanTargetY {
				if e.HasTarget || e.Stall != 0 {
					t.Errorf("mode %d: the unit arrived as %+v, want its target cleared with no residue", mode, e)
				}
				arrived = tick
			}
		}
		if arrived == 0 {
			t.Fatalf("mode %d: the unit is at %+v after 128 ticks and has not crossed", mode, w.Entities()[0])
		}
		// A tick moves a unit one cell on each axis at most, and every route must
		// touch the far end of the wall: 30 from (0,0) to (20,30), then 30 from
		// there to (39,0). Sixty is the floor.
		//
		// It is not 62. That is the length of the OPTIMISED search's own route,
		// which pays two hops to the corner rule; the wave is free of that rule
		// and walks the floor exactly, so a bound taken off one mode's route
		// would fail the other for being faster than a route it never took.
		const floor = 60
		if arrived < floor {
			t.Errorf("mode %d: the unit arrived at tick %d, and %d is the fewest the way round can take",
				mode, arrived, floor)
		}
		t.Logf("mode %d: one order, arrived at tick %d", mode, arrived)
	}
}

// TestUnderTheOldAllowanceTheChannelRefusesInCanonicalAndArrivesInOptimised is
// SC-12's discriminator: the defect itself, driven in the tree rather than
// described.
//
// The old far budget is the scaled rule. For this fixture the two slacks give
// the same number — max(5, 39>>2) and max(3, 39>>2) are both 9 + 39 — so the
// rule still in the tree for the near search reproduces the allowance the far
// search used to run under, exactly, and no deleted constant has to be revived
// to say what the old behaviour was.
//
// The two modes disagreeing here is the whole reason this story exists: the
// optimised search has never had a generation count, so it has been finding
// these routes all along.
func TestUnderTheOldAllowanceTheChannelRefusesInCanonicalAndArrivesInOptimised(t *testing.T) {
	w := mustWorldGrid(t, 1, chanBounds, ModeCanonical, chanGrid(), []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	if got := generationBudget(cell{0, 0}, cell{chanTargetX, chanTargetY}, scaledBudget, true); got != 48 {
		t.Fatalf("the old allowance for this crossing is %d, want 48", got)
	}
	if route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, scaledBudget, exactGoal, chanTargetX, chanTargetY); ok {
		t.Errorf("under the old allowance the wave found a route: %s", fmtRoute(route))
	}
	if _, ok := w.optimisedRoute(s, 0, terrainRelation, noWindow, settleOrdered, chanTargetX, chanTargetY); !ok {
		t.Error("the optimised search found no route, so the two modes never disagreed here")
	}

	// And under the rule that landed, the same call on the same world answers.
	route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, chanTargetX, chanTargetY)
	if !ok {
		t.Fatal("under the flat budget the wave still found no route")
	}
	if last := route[len(route)-1]; last.x != chanTargetX || last.y != chanTargetY {
		t.Errorf("the route ends at (%d,%d), want the target", last.x, last.y)
	}
}

func TestARouteOverAThousandRingsIsStillRefusedInCanonical(t *testing.T) {
	w := routeWorld(t, corridorBounds, corridorGrid(), []Entity{{ID: 1, X: 0, Y: 0}})
	s := newRouteScratch(w)

	free, ok := w.optimisedRoute(s, 0, terrainRelation, noWindow, settleOrdered, corridorTarget.x, corridorTarget.y)
	if !ok {
		t.Fatal("the optimised search found no route through the corridor")
	}
	if int64(len(free)) <= flatGenerations {
		t.Fatalf("the corridor's route is %d hops, and this witness needs one over %d",
			len(free), flatGenerations)
	}
	if route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, corridorTarget.x, corridorTarget.y); ok {
		t.Errorf("the wave found a route %d hops long under a budget of %d: %s",
			len(route), flatGenerations, fmtRoute(route))
	}
	t.Logf("the residual: %d hops, found in optimised mode and refused in canonical under %d generations",
		len(free), flatGenerations)
}
