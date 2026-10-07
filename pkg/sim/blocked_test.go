package sim

// This file is what one unit standing in another's way does to it. The obstacle
// in every case below is a unit that is going nowhere — not a contender —
// because that is the half of the occupancy relation a rule assembled out of the
// units that are MOVING gets wrong, and no contest between two movers can see
// it: both of those carry a target.
//
// What that obstacle is worth changed when a route began to be searched per
// tick. One occupied cell no longer stops anybody: it is a cell the search will
// not label, so a way round it is found where one exists and the mover only
// holds where none does. The cases here were written against the old outcome and
// have been rewritten to the current one rather than dropped — a unit routing
// round an obstacle is the same fact about enterability, observed one step later.
//
// occEntity and nearerCell come from occupancy_test.go — the reader that goes
// through the exported entity list, and the test's own straight-line arithmetic
// for staging fixtures. Both belong to the package rather than to that file, and
// a second copy here would only be somewhere for the two to drift apart.

import "testing"

// blkBounds is roomy enough that nothing here is stopped by an edge, and roomy
// enough for every detour these cases produce. Bounds ARE a movement constraint
// now, so a case that ran into one would be staging two things at once.
var blkBounds = Bounds{Width: 24, Height: 24}

// blkParked is the far corner an obstacle is moved to for the control run of a
// case. Every case checks that it holds nothing and is nobody's desired cell, so
// parking an obstacle there changes exactly one thing about the world: the cell
// the obstacle stands on.
var blkParked = [2]int32{23, 23}

// blkHolds compares an entity against the fields it must still be carrying, one
// field at a time and every field there is. A negative invariant over an
// entity's own canonical state is not a claim about its position: a step that
// moved nobody but quietly cleared a blocked entity's target would pass a
// position comparison, and a caller reading HasTarget could then no longer tell
// an obstacle from an arrival. A struct or byte-form equality would catch that
// too, and would report "not equal" without naming the field that moved.
func blkHolds(t *testing.T, tick int, what string, got, want Entity) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("tick %d: %s ID is %d, want %d", tick, what, got.ID, want.ID)
	}
	if got.X != want.X {
		t.Errorf("tick %d: %s X is %d, want %d", tick, what, got.X, want.X)
	}
	if got.Y != want.Y {
		t.Errorf("tick %d: %s Y is %d, want %d", tick, what, got.Y, want.Y)
	}
	if got.HasTarget != want.HasTarget {
		t.Errorf("tick %d: %s HasTarget is %v, want %v", tick, what, got.HasTarget, want.HasTarget)
	}
	if got.TargetX != want.TargetX {
		t.Errorf("tick %d: %s TargetX is %d, want %d", tick, what, got.TargetX, want.TargetX)
	}
	if got.TargetY != want.TargetY {
		t.Errorf("tick %d: %s TargetY is %d, want %d", tick, what, got.TargetY, want.TargetY)
	}
	if got.Class != want.Class {
		t.Errorf("tick %d: %s Class is %d, want %d", tick, what, got.Class, want.Class)
	}
}

// blkUnfinished is the other half of every unchanged-field comparison here: the
// entity must still hold a target naming a cell it is not standing on. Without
// it "unmoved" is satisfied by an entity that had nothing left to do, and the
// case would pass over a rule that never blocks anything.
func blkUnfinished(t *testing.T, tick int, what string, got Entity) {
	t.Helper()
	if !got.HasTarget || (got.TargetX == got.X && got.TargetY == got.Y) {
		t.Fatalf("tick %d: %s has nothing left to do — target %v (%d,%d) while standing at (%d,%d)",
			tick, what, got.HasTarget, got.TargetX, got.TargetY, got.X, got.Y)
	}
}

// blkFree fails the case unless the cell (x, y) is empty in ents. A case that
// says a cell is free and is wrong about it proves something other than what it
// claims to.
func blkFree(t *testing.T, what string, ents []Entity, x, y int32) {
	t.Helper()
	for _, e := range ents {
		if e.X == x && e.Y == y {
			t.Fatalf("fixture: %s (%d,%d) is held by entity %d", what, x, y, e.ID)
		}
	}
}

// blkMovesTo is the control every case here carries: the same unit under the
// same order, with the obstacle parked out of its way, reaches the cell a
// straight walk would take. It is what separates "the obstacle changed this
// unit's route" from "the route was never going that way" — and, where a case
// ends in a standstill, "the rule held this unit" from "the rule moves nobody at
// all", which satisfies every unchanged-field comparison in this file and reads
// exactly like a pass.
func blkMovesTo(t *testing.T, what string, ents []Entity, cmd Command, x, y int32) {
	t.Helper()
	w := mustWorld(t, 1, blkBounds, ents)
	Step(w, []Command{cmd})
	if got := occEntity(t, w, cmd.Entity); got.X != x || got.Y != y {
		t.Errorf("%s is at (%d,%d), want (%d,%d) — the step it was refused is one the rule would otherwise make",
			what, got.X, got.Y, x, y)
	}
}

// block is one case of the target-less obstacle: a mover ordered past a cell a
// still entity holds, with an uninvolved unit whose id sits BETWEEN theirs. The
// still entity is given no target in the fixture and no command anywhere in the
// case, which is the whole point of it.
type block struct {
	name               string
	mover, idle, still Entity
	to                 [2]int32
}

// blocks puts the obstacle on either side of the mover in id, so the entity that
// has to be seen is once behind the mover in the slice and once ahead of it — a
// predicate that scanned only the entities already resolved, or only the ones
// still to come, answers exactly one of these two correctly.
var blocks = []block{
	{
		name:  "stopped along x, the obstacle carrying the higher id",
		mover: Entity{ID: 2, X: 5, Y: 9},
		idle:  Entity{ID: 4, X: 20, Y: 2},
		still: Entity{ID: 6, X: 6, Y: 9},
		to:    [2]int32{11, 9},
	},
	{
		name:  "stopped along y, the obstacle carrying the lower id",
		mover: Entity{ID: 7, X: 3, Y: 8},
		idle:  Entity{ID: 5, X: 22, Y: 21},
		still: Entity{ID: 1, X: 3, Y: 7},
		to:    [2]int32{3, 1},
	},
}

// blkArrival is how long a case gives a mover to reach its order. It is a bound
// and not a prediction: a detour is longer than the straight walk it replaces,
// and how much longer is the search's answer, not this file's. What the bound
// has to do is fail a mover that never gets there.
const blkArrival = 24

// TestStepRoutesRoundATargetlessUnit is the obstacle that wants nothing: a unit
// standing on the cell a straight walk would take, with no target of its own. An
// occupancy relation built out of the units that are moving does not see it,
// walks the mover through it, and passes every case where both sides of a
// contest carry an order — so this is the one shape that tells the two rules
// apart.
//
// The mover no longer stops at it. The cell the obstacle holds is one the search
// will not label, so the route leaves the straight line, and the case asserts
// three things it can assert exactly: the mover never stands on the obstacle's
// cell, it moves on the very tick it was refused the straight line, and it
// arrives. The obstacle and the uninvolved unit are compared field by field,
// because an obstacle that silently cleared its own target would be
// indistinguishable from one that had arrived.
func TestStepRoutesRoundATargetlessUnit(t *testing.T) {
	for _, tc := range blocks {
		t.Run(tc.name, func(t *testing.T) {
			cmd := Command{Entity: tc.mover.ID, X: tc.to[0], Y: tc.to[1]}
			ents := []Entity{tc.mover, tc.idle, tc.still}
			cx, cy := nearerCell(tc.mover, cmd.X, cmd.Y)

			// The fixture, on the test's own arithmetic. Four things have to hold
			// or the case would pass for a reason that is not the rule: the still
			// unit stands exactly on the straight line's next cell and nothing
			// else does, it carries no target, the order names a cell beyond that
			// one so the mover cannot simply arrive, and the corner the control
			// parks the obstacle in is empty.
			if cx != tc.still.X || cy != tc.still.Y {
				t.Fatalf("fixture: the straight line's next cell is (%d,%d), the obstacle stands at (%d,%d)",
					cx, cy, tc.still.X, tc.still.Y)
			}
			blkFree(t, "the straight line's next cell", []Entity{tc.mover, tc.idle}, cx, cy)
			if tc.still.HasTarget {
				t.Fatalf("fixture: the obstacle carries a target (%d,%d) and may move",
					tc.still.TargetX, tc.still.TargetY)
			}
			if cx == cmd.X && cy == cmd.Y {
				t.Fatalf("fixture: the order (%d,%d) IS the next cell, so the mover would arrive on this tick",
					cmd.X, cmd.Y)
			}
			blkFree(t, "the control's parking cell", ents, blkParked[0], blkParked[1])
			if blkParked[0] == cx && blkParked[1] == cy {
				t.Fatalf("fixture: the control parks the obstacle on the next cell (%d,%d)", cx, cy)
			}
			lo, hi := tc.mover.ID, tc.still.ID
			if lo > hi {
				lo, hi = hi, lo
			}
			if lo+1 == hi || tc.idle.ID <= lo || tc.idle.ID >= hi {
				t.Fatalf("fixture: ids %d and %d must not be adjacent and %d must fall between them",
					lo, hi, tc.idle.ID)
			}

			w := mustWorld(t, 1, blkBounds, ents)
			arrived := 0
			for tick := 1; tick <= blkArrival; tick++ {
				before := occEntity(t, w, tc.mover.ID)
				if tick == 1 {
					Step(w, []Command{cmd})
				} else {
					Step(w, nil)
				}
				got := occEntity(t, w, tc.mover.ID)

				if got.X == tc.still.X && got.Y == tc.still.Y {
					t.Fatalf("tick %d: the mover is standing on the obstacle's cell (%d,%d)",
						tick, tc.still.X, tc.still.Y)
				}
				if tick == 1 && got.X == before.X && got.Y == before.Y {
					t.Fatalf("tick 1: the mover did not move; one occupied cell is a cell to route round, "+
						"not a reason to stand still at (%d,%d)", before.X, before.Y)
				}
				blkHolds(t, tick, "the obstacle's", occEntity(t, w, tc.still.ID), tc.still)
				blkHolds(t, tick, "the uninvolved unit's", occEntity(t, w, tc.idle.ID), tc.idle)

				if got.X == cmd.X && got.Y == cmd.Y {
					if got.HasTarget || got.TargetX != 0 || got.TargetY != 0 {
						t.Errorf("tick %d: the mover arrived holding target %v (%d,%d)",
							tick, got.HasTarget, got.TargetX, got.TargetY)
					}
					arrived = tick
					break
				}
				blkUnfinished(t, tick, "the mover", got)
			}
			if arrived == 0 {
				t.Errorf("the mover did not reach (%d,%d) in %d tick(s)", cmd.X, cmd.Y, blkArrival)
			}

			// The control: with the obstacle parked, the straight line's next
			// cell is the one the mover takes — so the detour above was the
			// obstacle's doing and not the route's own shape.
			parked := tc.still
			parked.X, parked.Y = blkParked[0], blkParked[1]
			blkMovesTo(t, "with the obstacle parked the mover",
				[]Entity{tc.mover, tc.idle, parked}, cmd, cx, cy)
		})
	}
}

// TestStepHoldsAPairEachOrderedOntoTheOthersCell witnesses the deadlock the
// contract discloses rather than asserting it, and it survives the search
// unchanged — for a reason worth stating, because it is no longer the old one.
//
// Each of the two is ordered onto the cell the other is STANDING on. A cell
// another unit holds is not enterable, so it is never labelled, so a search
// toward it returns no route however much open ground surrounds the pair. There
// is no way round a destination. So neither ever leaves: the absence of a swap
// and of a yield, both by contract, and the absence of a route because the route
// is to an occupied cell and not because searching was omitted.
//
// The two ids are not adjacent and an uninvolved unit stands between them in the
// slice, so a rule that compared a unit only against its neighbour there would
// let this pair trade cells.
func TestStepHoldsAPairEachOrderedOntoTheOthersCell(t *testing.T) {
	a := Entity{ID: 3, X: 8, Y: 12}
	b := Entity{ID: 9, X: 9, Y: 12}
	idle := Entity{ID: 6, X: 1, Y: 20}
	ents := []Entity{a, idle, b}
	toA := Command{Entity: a.ID, X: b.X, Y: b.Y}
	toB := Command{Entity: b.ID, X: a.X, Y: a.Y}

	if x, y := nearerCell(a, toA.X, toA.Y); x != b.X || y != b.Y {
		t.Fatalf("fixture: the lower id's desired cell is (%d,%d), the higher id stands at (%d,%d)",
			x, y, b.X, b.Y)
	}
	if x, y := nearerCell(b, toB.X, toB.Y); x != a.X || y != a.Y {
		t.Fatalf("fixture: the higher id's desired cell is (%d,%d), the lower id stands at (%d,%d)",
			x, y, a.X, a.Y)
	}
	if a.ID+1 == b.ID || idle.ID <= a.ID || idle.ID >= b.ID {
		t.Fatalf("fixture: ids %d and %d must not be adjacent and %d must fall between them",
			a.ID, b.ID, idle.ID)
	}
	blkFree(t, "the control's parking cell", ents, blkParked[0], blkParked[1])

	w := mustWorld(t, 1, blkBounds, ents)
	heldA, heldB := a, b
	heldA.TargetX, heldA.TargetY, heldA.HasTarget = toA.X, toA.Y, true
	heldB.TargetX, heldB.TargetY, heldB.HasTarget = toB.X, toB.Y, true

	cmds := []Command{toA, toB}
	for tick := 1; tick <= 4; tick++ {
		Step(w, cmds)
		cmds = nil

		gotA, gotB := occEntity(t, w, a.ID), occEntity(t, w, b.ID)
		blkHolds(t, tick, "the lower id of the pair:", gotA, heldA)
		blkUnfinished(t, tick, "the lower id of the pair", gotA)
		blkHolds(t, tick, "the higher id of the pair:", gotB, heldB)
		blkUnfinished(t, tick, "the higher id of the pair", gotB)
		blkHolds(t, tick, "the uninvolved entity's", occEntity(t, w, idle.ID), idle)
	}

	// Each half of the pair, with the other parked away: the cell it was refused
	// is one it does reach when nobody is standing on it.
	parkedA, parkedB := a, b
	parkedA.X, parkedA.Y = blkParked[0], blkParked[1]
	parkedB.X, parkedB.Y = blkParked[0], blkParked[1]
	blkMovesTo(t, "with its counterpart parked the lower id",
		[]Entity{a, idle, parkedB}, toA, b.X, b.Y)
	blkMovesTo(t, "with its counterpart parked the higher id",
		[]Entity{parkedA, idle, b}, toB, a.X, a.Y)
}

// blkDiagonal is the straight line's other half: a mover whose two axes both
// carry a step, so that the cell a straight walk would take is diagonal from the
// one it stands on.
type blkDiagonal struct {
	name         string
	mover, still Entity
	to           [2]int32
}

// TestStepRoutesRoundAHeldDiagonal is the same fact as the cases above with the
// obstacle on a diagonal rather than an axis, and it is the one that would show
// a search reaching for whichever axis happened to be free. Both orthogonal
// neighbours toward the order are empty here, so a rule that slid along one of
// them would look exactly like a route — the case therefore says which cells the
// mover may stand on rather than that it moved: never the obstacle's, and,
// within a tick of leaving, one that is strictly nearer the order than where it
// started.
func TestStepRoutesRoundAHeldDiagonal(t *testing.T) {
	mover := Entity{ID: 2, X: 4, Y: 4}
	still := Entity{ID: 8, X: 5, Y: 5}
	idle := Entity{ID: 5, X: 20, Y: 1}
	ents := []Entity{mover, idle, still}
	cmd := Command{Entity: mover.ID, X: 10, Y: 10}
	cx, cy := nearerCell(mover, cmd.X, cmd.Y)

	if cx == mover.X || cy == mover.Y {
		t.Fatalf("fixture: the next cell (%d,%d) is not diagonal from (%d,%d) — this case needs a step on both axes",
			cx, cy, mover.X, mover.Y)
	}
	if cx != still.X || cy != still.Y {
		t.Fatalf("fixture: the straight line's next cell is (%d,%d), the obstacle stands at (%d,%d)",
			cx, cy, still.X, still.Y)
	}
	if still.HasTarget {
		t.Fatalf("fixture: the obstacle carries a target (%d,%d) and may move", still.TargetX, still.TargetY)
	}
	if cx == cmd.X && cy == cmd.Y {
		t.Fatalf("fixture: the order (%d,%d) IS the next cell, so the mover would arrive on this tick",
			cmd.X, cmd.Y)
	}
	// The two cells a sliding rule would settle for, and the corner the control
	// parks the obstacle in.
	blkFree(t, "the orthogonal x neighbour", ents, cx, mover.Y)
	blkFree(t, "the orthogonal y neighbour", ents, mover.X, cy)
	blkFree(t, "the control's parking cell", ents, blkParked[0], blkParked[1])

	w := mustWorld(t, 1, blkBounds, ents)
	arrived := 0
	for tick := 1; tick <= blkArrival; tick++ {
		before := occEntity(t, w, mover.ID)
		if tick == 1 {
			Step(w, []Command{cmd})
		} else {
			Step(w, nil)
		}
		got := occEntity(t, w, mover.ID)
		if got.X == still.X && got.Y == still.Y {
			t.Fatalf("tick %d: the mover is standing on the obstacle's cell (%d,%d)", tick, still.X, still.Y)
		}
		if got.X == before.X && got.Y == before.Y {
			t.Fatalf("tick %d: the mover stood still at (%d,%d); one occupied cell is a cell to route "+
				"round, and both of its neighbours toward the order are free", tick, before.X, before.Y)
		}
		blkHolds(t, tick, "the obstacle's", occEntity(t, w, still.ID), still)
		if got.X == cmd.X && got.Y == cmd.Y {
			arrived = tick
			break
		}
	}
	if arrived == 0 {
		t.Errorf("the mover did not reach (%d,%d) in %d tick(s)", cmd.X, cmd.Y, blkArrival)
	}

	parked := still
	parked.X, parked.Y = blkParked[0], blkParked[1]
	blkMovesTo(t, "with the obstacle parked the mover",
		[]Entity{mover, idle, parked}, cmd, cx, cy)
}

// blkMirrors hold one orthogonal neighbour toward the target and leave the
// desired diagonal free — once on the x axis and once on the y, because a rule
// that tests one extra cell tests one axis' neighbour first and a single case
// would let the other half through.
var blkMirrors = []blkDiagonal{
	{
		name:  "the orthogonal x neighbour held",
		mover: Entity{ID: 11, X: 12, Y: 6},
		still: Entity{ID: 4, X: 11, Y: 6},
		to:    [2]int32{6, 0},
	},
	{
		name:  "the orthogonal y neighbour held",
		mover: Entity{ID: 6, X: 18, Y: 18},
		still: Entity{ID: 13, X: 18, Y: 17},
		to:    [2]int32{22, 14},
	},
}

// TestStepStepsDiagonallyWhenOnlyAnOrthogonalNeighbourIsHeld is the corner cut,
// and in the canonical mode it is not an oversight but the rule: a diagonal is
// taken with NO test on the two cells it passes between. With the diagonal cell
// itself free the mover steps onto it and squeezes past an occupied cell close
// enough to touch.
//
// A search that refused to cut a corner would refuse this move, would satisfy
// every held case in this file, and nothing else in the package would notice. It
// is the other mode's rule, and this is where the two part company.
func TestStepStepsDiagonallyWhenOnlyAnOrthogonalNeighbourIsHeld(t *testing.T) {
	for _, tc := range blkMirrors {
		t.Run(tc.name, func(t *testing.T) {
			ents := []Entity{tc.mover, tc.still}
			cmd := Command{Entity: tc.mover.ID, X: tc.to[0], Y: tc.to[1]}
			cx, cy := nearerCell(tc.mover, cmd.X, cmd.Y)

			if cx == tc.mover.X || cy == tc.mover.Y {
				t.Fatalf("fixture: the desired cell (%d,%d) is not diagonal from (%d,%d)",
					cx, cy, tc.mover.X, tc.mover.Y)
			}
			blkFree(t, "the desired cell", ents, cx, cy)
			onX := tc.still.X == cx && tc.still.Y == tc.mover.Y
			onY := tc.still.X == tc.mover.X && tc.still.Y == cy
			if onX == onY {
				t.Fatalf("fixture: the obstacle at (%d,%d) must stand on exactly one orthogonal neighbour of (%d,%d) toward the target",
					tc.still.X, tc.still.Y, tc.mover.X, tc.mover.Y)
			}
			if tc.still.HasTarget {
				t.Fatalf("fixture: the obstacle carries a target (%d,%d) and may move",
					tc.still.TargetX, tc.still.TargetY)
			}

			w := mustWorld(t, 1, blkBounds, ents)
			Step(w, []Command{cmd})

			got := occEntity(t, w, tc.mover.ID)
			if got.X != cx || got.Y != cy {
				t.Errorf("the mover is at (%d,%d), want the desired cell (%d,%d) — only the one desired cell may stop it",
					got.X, got.Y, cx, cy)
			}
			if !got.HasTarget || got.TargetX != cmd.X || got.TargetY != cmd.Y {
				t.Errorf("the mover holds target %v (%d,%d), want the order it was given (%d,%d)",
					got.HasTarget, got.TargetX, got.TargetY, cmd.X, cmd.Y)
			}
			blkHolds(t, 1, "the obstacle's", occEntity(t, w, tc.still.ID), tc.still)
		})
	}
}
