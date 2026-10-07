package sim

// A body standing on the cell a near search was aimed at.
//
// The far search reads terrain alone, so it routes straight through whatever
// units happen to stand on the way and hands the near search a waypoint four
// cells along that line. When a body holds that waypoint the near search cannot
// label it however long it runs — so what it comes back with is the whole
// question, and these are the two answers it has: a substitute beside the
// waypoint, or nothing at all.

import "testing"

// The waypoint fixture. A twenty-four square, the mover on row 10 at column 2,
// sent to column 14 of the same row; the body stands at column 6, which is the
// FOURTH cell of the straight line from the mover and therefore exactly the cell
// the near search is aimed at on the first tick.
const (
	wpSide  int32 = 24
	wpRow   int32 = 10
	wpFromX int32 = 2
	wpBodyX int32 = 6
	wpToX   int32 = 14
)

// wpGrid is that map. corridor closes every row but the mover's, which turns the
// open field into a one-row channel and leaves the body the only thing between
// the mover and its order.
func wpGrid(corridor bool) []byte {
	g := make([]byte, wpSide*wpSide)
	if !corridor {
		return g
	}
	for y := int32(0); y < wpSide; y++ {
		if y == wpRow {
			continue
		}
		for x := int32(0); x < wpSide; x++ {
			g[y*wpSide+x] = blockGround
		}
	}
	return g
}

// wpWorld is the fixture's world: the mover, the body, and the grid asked for.
func wpWorld(t *testing.T, corridor bool) *World {
	t.Helper()
	return mustWorldGrid(t, 1, Bounds{Width: wpSide, Height: wpSide}, ModeCanonical, wpGrid(corridor),
		[]Entity{{ID: 1, X: wpFromX, Y: wpRow}, {ID: 2, X: wpBodyX, Y: wpRow}})
}

// wpDrive runs the order to a standstill and reports where the mover ended,
// whether it ever stood on the body's cell, and whether it still holds an order.
func wpDrive(t *testing.T, w *World, ticks int) (last Entity, trod bool) {
	t.Helper()
	Step(w, []Command{{Entity: 1, X: wpToX, Y: wpRow}})
	for k := 0; ; k++ {
		e := w.Entities()[0]
		if e.X == wpBodyX && e.Y == wpRow {
			trod = true
		}
		if body := w.Entities()[1]; body.X != wpBodyX || body.Y != wpRow {
			t.Fatalf("tick %d: the body walked to (%d,%d) — it takes no order and must not move",
				k, body.X, body.Y)
		}
		if !e.HasTarget || k >= ticks {
			return e, trod
		}
		Step(w, nil)
	}
}

// TestAMoverWalksPastABodyStandingOnItsWaypoint is AC-1.
//
// The whole line from the mover to its order is open terrain, so the stored
// route runs straight through the body and the near search is aimed at the cell
// the body holds. Under an all-or-nothing near search that is sixteen refusals
// and a dropped order twelve cells short; under a settling one it is a step
// aside, and the mover arrives.
//
// The corridor case is the control, and it is what says the arrival is the
// BYPASS and not the settling: the same body, the same order, one row of map,
// and the mover gets no further than the cell before it.
func TestAMoverWalksPastABodyStandingOnItsWaypoint(t *testing.T) {
	t.Run("the way round is open", func(t *testing.T) {
		w := wpWorld(t, false)

		// The fixture's own premise: the far search really does aim the near one
		// at the body's cell. Without this the test could pass for a route that
		// never met the body at all.
		s := newRouteScratch(w)
		route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, settleOrdered, wpToX, wpRow)
		if !ok || len(route) < lookAhead+1 {
			t.Fatalf("the far route is %s, want a straight line long enough to have a fourth cell",
				fmtRoute(route))
		}
		if sub := route[subGoalIndex(len(route))]; sub != (cell{x: wpBodyX, y: wpRow}) {
			t.Fatalf("the near search is aimed at %v, and this fixture needs the body's cell (%d,%d)",
				sub, wpBodyX, wpRow)
		}

		last, trod := wpDrive(t, w, 256)
		if last.X != wpToX || last.Y != wpRow {
			t.Errorf("the mover ended at (%d,%d), want (%d,%d) — the way past the body is one cell aside",
				last.X, last.Y, wpToX, wpRow)
		}
		if trod {
			t.Errorf("the mover stood on (%d,%d), which a body holds", wpBodyX, wpRow)
		}
		if last.HasTarget || last.Stall != 0 {
			t.Errorf("the arrived mover carries target %v stall %d, want no residue",
				last.HasTarget, last.Stall)
		}
	})

	t.Run("the way round is sealed", func(t *testing.T) {
		w := wpWorld(t, true)
		last, trod := wpDrive(t, w, 256)
		if last.X != wpBodyX-1 || last.Y != wpRow {
			t.Errorf("the mover ended at (%d,%d), want (%d,%d) — a one-row channel with a body in it is "+
				"walked up to and no further", last.X, last.Y, wpBodyX-1, wpRow)
		}
		if trod {
			t.Errorf("the mover stood on (%d,%d), which a body holds", wpBodyX, wpRow)
		}
		if last.HasTarget || last.Stall != 0 || last.TargetX != 0 || last.TargetY != 0 {
			t.Errorf("the given-up mover carries target (%d,%d) has=%v stall=%d, want no residue",
				last.TargetX, last.TargetY, last.HasTarget, last.Stall)
		}
	})
}

// wpRingSide is the ring fixture's map, and wpRingGoal the cell it is built
// around: every cell within Chebyshev 4 of the goal is closed, so rings 1 to 4
// hold nothing a wave can label and ring 5 is the first that does.
const (
	wpRingSide  int32 = 24
	wpRingGoalX int32 = 12
	wpRingGoalY int32 = 12
	wpRingWall  int32 = 4
	wpRingFromX int32 = 5
)

// TestTheNearSearchLooksFurtherThanTheFarOneForASubstitute is AC-2.
//
// The two settling rules differ in their ring bound and in nothing else, so a
// fixture that puts the only labelled neighbourhood BETWEEN the two bounds is
// answered by one and refused by the other. Everything else about the two calls
// — relation, window, budget, mover, goal — is held identical, so the bound is
// the only thing that can explain the difference.
func TestTheNearSearchLooksFurtherThanTheFarOneForASubstitute(t *testing.T) {
	g := make([]byte, wpRingSide*wpRingSide)
	for y := wpRingGoalY - wpRingWall; y <= wpRingGoalY+wpRingWall; y++ {
		for x := wpRingGoalX - wpRingWall; x <= wpRingGoalX+wpRingWall; x++ {
			g[y*wpRingSide+x] = blockGround
		}
	}
	w := mustWorldGrid(t, 1, Bounds{Width: wpRingSide, Height: wpRingSide}, ModeCanonical, g,
		[]Entity{{ID: 1, X: wpRingFromX, Y: wpRingGoalY}})
	s := newRouteScratch(w)

	start := cell{x: wpRingFromX, y: wpRingGoalY}
	goal := cell{x: wpRingGoalX, y: wpRingGoalY}
	// The fixture's premise, asserted rather than assumed: the first ring holding
	// anything is outside the far bound and inside the near one. A bound of L
	// probes rings 1..L-1, so ring 5 needs L > 5 on one side and L <= 5 on the
	// other.
	if near := settleRings(settleStep, start, goal); near != stepRings {
		t.Fatalf("the near bound is %d, want %d", near, stepRings)
	}
	if far := settleRings(settleOrdered, start, goal); far > int64(wpRingWall)+1 {
		t.Fatalf("the far bound is %d, and this fixture needs it at or below %d", far, wpRingWall+1)
	}

	near, nok := w.canonicalRoute(s, 0, terrainRelation, dynamicWindow, scaledBudget, settleStep, goal.x, goal.y)
	if !nok || len(near) == 0 {
		t.Fatalf("the near rule found nothing: ok=%v route=%s", nok, fmtRoute(near))
	}
	if d := near[len(near)-1].chebyshevTo(goal); d != int64(wpRingWall)+1 {
		t.Errorf("the near rule settled %d rings from the goal, want %d", d, wpRingWall+1)
	}
	if far, fok := w.canonicalRoute(s, 0, terrainRelation, dynamicWindow, scaledBudget, settleOrdered, goal.x, goal.y); fok {
		t.Errorf("the far rule settled on %s, and its bound does not reach the first labelled ring",
			fmtRoute(far))
	}
}
