package sim

import "testing"

// occBounds is roomy enough that nothing in the contested cases below is stopped
// by an edge. Bounds ARE a movement constraint now — a unit never leaves them —
// so a fixture that ran into one would be staging two things at once.
var occBounds = Bounds{Width: 32, Height: 16}

// occCorridor is one cell high, and that is what makes the convoy cases below
// measure the resolution order rather than the search. On open ground a unit
// refused a cell walks round it and every ordering of the units produces
// movement; in a corridor there is nowhere to go round to, so a follower moves
// only if the cell ahead was vacated EARLIER IN THIS SAME TICK.
var occCorridor = Bounds{Width: 32, Height: 1}

// occEntity reads one entity out of a world by id, through the exported reader so
// that a case never has to know where in the slice an id sits.
func occEntity(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("the world holds no entity %d", id)
	return Entity{}
}

// nearerCell is the test's OWN arithmetic — one step nearer on each axis — and
// it is not a claim about what a step does: a route is searched, and the cell it
// begins with is the search's answer and not this. It is here to STAGE fixtures,
// so that a case wanting an obstacle on the straight line between a unit and its
// order can put one there and check that it did.
func nearerCell(e Entity, tx, ty int32) (int32, int32) {
	x, y := e.X, e.Y
	switch {
	case tx > x:
		x++
	case tx < x:
		x--
	}
	switch {
	case ty > y:
		y++
	case ty < y:
		y--
	}
	return x, y
}

// contest is one case of the contested cell: two units, each one step from a
// cell BOTH are ordered onto, and one uninvolved unit whose id sits BETWEEN
// theirs. The order names that cell itself rather than something past it, so the
// contest is decided by which unit is resolved first and by nothing about how
// either one would have continued.
type contest struct {
	name         string
	lo, idle, hi Entity
	at           [2]int32
}

// contests varies the approach geometry, because the rule must not care which
// axes carry the step.
var contests = []contest{
	{
		name: "converging along x",
		lo:   Entity{ID: 1, X: 2, Y: 3},
		idle: Entity{ID: 3, X: 12, Y: 9, ActorState: actorStateGuard, Reach: 1, PostX: 12, PostY: 9},
		hi:   Entity{ID: 5, X: 4, Y: 3},
		at:   [2]int32{3, 3},
	},
	{
		name: "converging along y",
		lo:   Entity{ID: 2, X: 5, Y: 1},
		idle: Entity{ID: 4, X: 1, Y: 12, ActorState: actorStateGuard, Reach: 1, PostX: 1, PostY: 12},
		hi:   Entity{ID: 6, X: 5, Y: 3},
		at:   [2]int32{5, 2},
	},
	{
		name: "converging on the diagonal",
		lo:   Entity{ID: 1, X: 2, Y: 2},
		idle: Entity{ID: 4, X: 0, Y: 14, ActorState: actorStateGuard, Reach: 1, PostX: 0, PostY: 14},
		hi:   Entity{ID: 7, X: 4, Y: 4},
		at:   [2]int32{3, 3},
	},
}

// chebyshev is the test's own distance, used only to check that a fixture put
// its two contenders one step from the contested cell.
func chebyshev(x0, y0, x1, y1 int32) int32 {
	dx, dy := x1-x0, y1-y0
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dy > dx {
		return dy
	}
	return dx
}

// TestStepGivesAContestedCellToTheLowerID names the winner, which is the half of
// the rule an outcome like "no two units share a cell" cannot see: that one holds
// under a descending resolution order too. The contenders are never adjacent in
// id and an uninvolved unit stands between them in the slice, so a rule comparing
// a unit only against its predecessor there answers this correctly for the wrong
// reason — and a three-way contest wrongly.
//
// The loser's outcome is now a consequence of the search rather than of a refused
// step, and it is stated as one: the cell it was ordered onto is the cell the
// winner is standing on, a cell another unit holds is not enterable, and a target
// that is not enterable is never labelled — so the loser's search returns no
// route and it keeps its cell and its whole order. Reverse the resolution order
// and the two swap places.
func TestStepGivesAContestedCellToTheLowerID(t *testing.T) {
	for _, tc := range contests {
		t.Run(tc.name, func(t *testing.T) {
			cx, cy := tc.at[0], tc.at[1]
			loCmd := Command{Entity: tc.lo.ID, X: cx, Y: cy}
			hiCmd := Command{Entity: tc.hi.ID, X: cx, Y: cy}

			// The fixture, on the test's own arithmetic: both contenders are one
			// step from the cell, that cell starts empty, and the ids are shaped
			// the way this case needs. Without this the case could pass having
			// staged no contest.
			for _, e := range []Entity{tc.lo, tc.hi} {
				if d := chebyshev(e.X, e.Y, cx, cy); d != 1 {
					t.Fatalf("fixture: unit %d is %d cell(s) from the contested cell (%d,%d), want 1",
						e.ID, d, cx, cy)
				}
			}
			for _, e := range []Entity{tc.lo, tc.idle, tc.hi} {
				if e.X == cx && e.Y == cy {
					t.Fatalf("fixture: unit %d already stands on the contested cell (%d,%d)", e.ID, cx, cy)
				}
			}
			if tc.lo.ID+1 == tc.hi.ID || tc.idle.ID <= tc.lo.ID || tc.idle.ID >= tc.hi.ID {
				t.Fatalf("fixture: ids %d and %d must not be adjacent and %d must fall between them",
					tc.lo.ID, tc.hi.ID, tc.idle.ID)
			}

			w := mustWorld(t, 1, occBounds, []Entity{tc.lo, tc.idle, tc.hi})
			Step(w, []Command{loCmd, hiCmd})

			lo := occEntity(t, w, tc.lo.ID)
			if lo.X != cx || lo.Y != cy {
				t.Errorf("the lower id is at (%d,%d), want the contested cell (%d,%d)", lo.X, lo.Y, cx, cy)
			}
			if lo.HasTarget || lo.TargetX != 0 || lo.TargetY != 0 {
				t.Errorf("the winner arrived holding target %v (%d,%d); an arrival clears with no residue",
					lo.HasTarget, lo.TargetX, lo.TargetY)
			}

			hi := occEntity(t, w, tc.hi.ID)
			if hi.X != tc.hi.X || hi.Y != tc.hi.Y {
				t.Errorf("the higher id moved to (%d,%d), from (%d,%d)", hi.X, hi.Y, tc.hi.X, tc.hi.Y)
			}
			// The refusal happened and left work undone. A unit that had arrived,
			// or whose target had been cleared, would satisfy "unmoved" for a
			// reason that has nothing to do with the contest.
			if !hi.HasTarget || hi.TargetX != hiCmd.X || hi.TargetY != hiCmd.Y {
				t.Errorf("the loser holds target %v (%d,%d), want the order it was given (%d,%d)",
					hi.HasTarget, hi.TargetX, hi.TargetY, hiCmd.X, hiCmd.Y)
			}
			if before, after := tc.idle, occEntity(t, w, tc.idle.ID); after != before {
				t.Errorf("the uninvolved unit is %+v, want %+v", after, before)
			}
		})
	}
}

// convoyOrders sends every unit of the line one cell further along the corridor
// than where it currently stands, so the line is asked to move as a line and any
// stretching in it is the rule's doing and not the orders'.
//
// The order is one cell ahead rather than the corridor's far end, and that is
// forced: a corridor is blocked by whatever is standing in it, so a unit ordered
// PAST the unit in front of it has no route at all and would hold whatever the
// resolution order was. One cell ahead is the shortest order that still asks the
// question this file exists to ask.
func convoyOrders(w *World) []Command {
	var cmds []Command
	for _, e := range w.Entities() {
		cmds = append(cmds, Command{Entity: e.ID, X: e.X + 1, Y: e.Y})
	}
	return cmds
}

// TestStepAdvancesALowIDLedConvoyAsOneBody is the vacated cell: each follower's
// only way forward is the cell the unit ahead of it has already left inside this
// same tick, which only an incremental resolution in ascending id makes free. A
// frozen snapshot of the tick's start positions advances the leader alone, and so
// does a descending loop.
//
// The corridor is what closes the alternative. On open ground a follower refused
// the cell ahead simply routes around the unit in front, so it moves either way
// and the case would witness nothing.
func TestStepAdvancesALowIDLedConvoyAsOneBody(t *testing.T) {
	start := []Entity{{ID: 0, X: 2, Y: 0}, {ID: 1, X: 1, Y: 0}, {ID: 2, X: 0, Y: 0}}
	w := mustWorld(t, 1, occCorridor, start)

	for tick := 1; tick <= 5; tick++ {
		Step(w, convoyOrders(w))
		for _, e := range start {
			want := e.X + int32(tick)
			got := occEntity(t, w, e.ID)
			if got.X != want || got.Y != e.Y {
				t.Fatalf("tick %d: unit %d is at (%d,%d), want (%d,%d) — the whole line advances every tick",
					tick, e.ID, got.X, got.Y, want, e.Y)
			}
		}
	}
}

// highLeaderCells is the contract's stretch-then-flow example written out by
// hand, one row per tick, holding the x of unit 0, unit 1 and unit 2 in that
// order — the row index IS the unit id, since the ids run 0..2. The first three
// rows are the worked example; the last two continue it by the same reasoning.
// Nothing here is recomputed from the step's own rule.
var highLeaderCells = [][3]int32{
	{0, 1, 3},
	{0, 2, 4},
	{1, 3, 5},
	{2, 4, 6},
	{3, 5, 7},
}

// TestStepStretchesThenFlowsWhenTheLeaderCarriesTheHighestID witnesses a
// disclosed limitation instead of asserting it. The same three units as the
// convoy above, re-ordered so the leader carries the highest id, reproduce the
// contract's second worked example tick for tick: the line stretches while each
// follower waits for the cell ahead to be vacated, then flows again once one-cell
// gaps have opened. Under a descending order there is no stretch — all three
// advance from the first tick — so this pins the order as much as the convoy does.
//
// A held follower here has NO ROUTE at all rather than a refused step: the
// corridor's only neighbour toward the order is the cell the unit ahead holds,
// so the frontier empties in the first generation.
func TestStepStretchesThenFlowsWhenTheLeaderCarriesTheHighestID(t *testing.T) {
	start := []Entity{{ID: 0, X: 0, Y: 0}, {ID: 1, X: 1, Y: 0}, {ID: 2, X: 2, Y: 0}}
	w := mustWorld(t, 1, occCorridor, start)

	blocked := 0
	for tick, want := range highLeaderCells {
		before := [3]int32{
			occEntity(t, w, 0).X,
			occEntity(t, w, 1).X,
			occEntity(t, w, 2).X,
		}
		Step(w, convoyOrders(w))

		for id, wantX := range want {
			got := occEntity(t, w, EntityID(id))
			if got.X != wantX || got.Y != 0 {
				t.Fatalf("tick %d: unit %d is at (%d,%d), want (%d,0)", tick+1, id, got.X, got.Y, wantX)
			}
			if got.X == before[id] {
				blocked++
			}
		}
	}
	// None of these units can arrive inside the run, so a unit that did not move
	// was held. Had the count come out zero, the table above would be describing a
	// line that never contended.
	if blocked == 0 {
		t.Error("no unit was held anywhere in the run: the fixture staged no contention")
	}
}
