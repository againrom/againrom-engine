package sim

// This file is the malformed start: two units already standing on one cell. Such a
// world is not hypothetical and that is why this file exists — a placed unit's cell
// is its record position with the sub-cell fraction dropped, so two placements
// inside one cell collapse onto it, and neither construction nor decoding inspects
// a position. What advancement does with such a world is therefore contract rather
// than luck.
//
// Nothing here separates, repairs or refuses the pair. Two things are asserted
// instead. The bound: after a tick, no cell holds more than the greater of one and
// the count it already held, so advancement cannot make a malformed state worse and
// is not required to undo it. And the order naming the cell a co-located unit is
// already standing on is cleared — which is the one thing that tells the shipped
// rule apart from one that tests the cell a unit stands on and excludes only
// itself. That variant is behaviourally identical on every well-formed world there
// is, so nowhere but here can it be seen at all.
//
// blkBounds, blkParked, blkHolds, blkUnfinished, blkFree and blkMovesTo come from
// blocked_test.go and occDesired from occupancy_test.go. The control this file
// needs is the one that file already defines — the obstacle parked in a corner the
// case has checked to be empty — and a second convention for it here would only be
// somewhere for the two to drift apart.

import (
	"fmt"
	"testing"
)

// malShared is the cell the pair loads onto. It is nowhere near blkParked, which
// every case checks, because the control parks the pair there.
var malShared = [2]int32{5, 5}

// malTicks holds each run past the tick the self-order is given on, so that a
// cleared order is shown to stay cleared and a standstill to be stable rather than
// merely late.
const malTicks = 3

// malCase is one malformed fixture: two units on malShared, a third adjacent to it
// and ordered onto it, and the ids of the co-located units that are ordered to the
// cell they are already standing on. The third unit's id falls between the pair's
// and no two ids are adjacent, so a rule comparing a unit only against its
// neighbour in the slice cannot answer any of this by accident.
type malCase struct {
	name        string
	pair        [2]Entity
	third       Entity
	selfOrdered []EntityID
}

// malCases carry the self-order on each of the two co-located units in turn and
// then on both at once. The two are not interchangeable: a rule that excluded self
// by index strands whichever of them carries the order, so which one it is decides
// nothing, and stating that by measurement is cheaper than arguing it. The third
// unit approaches from a different side in each case, once orthogonally on each
// axis and once diagonally, so the shared cell is a desired cell of every shape.
var malCases = []malCase{
	{
		name:        "the self-order on the lower of the two co-located ids",
		pair:        [2]Entity{{ID: 3, X: 5, Y: 5}, {ID: 9, X: 5, Y: 5}},
		third:       Entity{ID: 6, X: 5, Y: 4},
		selfOrdered: []EntityID{3},
	},
	{
		name:        "the self-order on the higher, and the third unit diagonal",
		pair:        [2]Entity{{ID: 3, X: 5, Y: 5}, {ID: 9, X: 5, Y: 5}},
		third:       Entity{ID: 6, X: 4, Y: 4},
		selfOrdered: []EntityID{9},
	},
	{
		name:        "both co-located units ordered onto the cell they stand on",
		pair:        [2]Entity{{ID: 3, X: 5, Y: 5}, {ID: 9, X: 5, Y: 5}},
		third:       Entity{ID: 6, X: 6, Y: 5},
		selfOrdered: []EntityID{3, 9},
	},
}

// malCell is one cell and the number of units standing on it.
type malCell struct {
	x, y int32
	n    int
}

// malCounts is the occupant count of every cell any unit in ents stands on, built
// by the test's own scan. The cells stay in the order they were first seen, so a
// failure names the same one first every time it is run: a map would count just as
// well and hand them back in a different order per process, which is the one thing
// this package's whole subject forbids.
func malCounts(ents []Entity) []malCell {
	var out []malCell
	for _, e := range ents {
		found := false
		for i := range out {
			if out[i].x == e.X && out[i].y == e.Y {
				out[i].n++
				found = true
				break
			}
		}
		if !found {
			out = append(out, malCell{x: e.X, y: e.Y, n: 1})
		}
	}
	return out
}

// malCountAt is the count of one cell in a malCounts result, and zero for a cell
// nobody stands on.
func malCountAt(cells []malCell, x, y int32) int {
	for _, c := range cells {
		if c.x == x && c.y == y {
			return c.n
		}
	}
	return 0
}

// malBounded is the bound itself, over every cell either state touches rather than
// over the shared cell alone. A rule that let a unit onto a cell it should have
// been refused raises the count of THAT cell, and the shared cell knows nothing
// about it; checking only where the pair stands would report a world getting worse
// somewhere else as unchanged.
func malBounded(t *testing.T, tick int, before, after []Entity) {
	t.Helper()
	was, now := malCounts(before), malCounts(after)
	touched := append(append([]malCell{}, was...), now...)
	for _, c := range touched {
		n0, n1 := malCountAt(was, c.x, c.y), malCountAt(now, c.x, c.y)
		limit := n0
		if limit < 1 {
			limit = 1
		}
		if n1 > limit {
			t.Errorf("tick %d: cell (%d,%d) holds %d units and held %d before — a tick may not raise a count above the greater of one and the count already there",
				tick, c.x, c.y, n1, n0)
		}
	}
}

// malOn is the ids standing on one cell, ascending because that is the order a
// world hands its entities out in. The package's own predicate answers whether a
// cell is held; this file has to know by whom, and how many.
func malOn(ents []Entity, x, y int32) []EntityID {
	var out []EntityID
	for _, e := range ents {
		if e.X == x && e.Y == y {
			out = append(out, e.ID)
		}
	}
	return out
}

// malWant is one unit and what it must be holding, with a name for the failure to
// carry.
type malWant struct {
	what string
	e    Entity
}

// TestStepAdvancesACoLocatedPairWithoutRaisingAnyCellsCount is the malformed world
// advanced rather than repaired. The pair stays where it loaded, the third unit is
// refused the cell the two are standing on and keeps the order that named it, and
// the co-located unit ordered onto the cell it already occupies has that order
// cleared with no residue behind it — the zero-distance case decided before the
// occupancy test rather than inside it, which is a distinction no well-formed world
// can show.
func TestStepAdvancesACoLocatedPairWithoutRaisingAnyCellsCount(t *testing.T) {
	for _, tc := range malCases {
		t.Run(tc.name, func(t *testing.T) {
			ents := []Entity{tc.pair[0], tc.pair[1], tc.third}
			toThird := Command{Entity: tc.third.ID, X: malShared[0], Y: malShared[1]}
			cmds := make([]Command, 0, len(tc.selfOrdered)+1)
			for _, id := range tc.selfOrdered {
				cmds = append(cmds, Command{Entity: id, X: malShared[0], Y: malShared[1]})
			}
			cmds = append(cmds, toThird)

			// The fixture, on the test's own arithmetic. The pair is co-located on
			// the shared cell, the third unit is beside that cell and not on it and
			// its desired cell IS that cell, every self-order is carried by one of
			// the two co-located units rather than by the third, the ids are
			// non-adjacent with the third's between them, and the corner the
			// control parks the pair in is empty and is not the shared cell.
			for _, e := range tc.pair {
				if e.X != malShared[0] || e.Y != malShared[1] {
					t.Fatalf("fixture: unit %d stands at (%d,%d), the shared cell is (%d,%d) — both of the pair must load onto it",
						e.ID, e.X, e.Y, malShared[0], malShared[1])
				}
				if e.HasTarget {
					t.Fatalf("fixture: unit %d carries a target (%d,%d); every order here is given as a command",
						e.ID, e.TargetX, e.TargetY)
				}
			}
			if cx, cy := nearerCell(tc.third, toThird.X, toThird.Y); cx != malShared[0] || cy != malShared[1] {
				t.Fatalf("fixture: the third unit's desired cell is (%d,%d), the shared cell is (%d,%d)",
					cx, cy, malShared[0], malShared[1])
			}
			if tc.third.X == malShared[0] && tc.third.Y == malShared[1] {
				t.Fatalf("fixture: the third unit stands on the shared cell — it must be beside it")
			}
			for _, id := range tc.selfOrdered {
				if id != tc.pair[0].ID && id != tc.pair[1].ID {
					t.Fatalf("fixture: a self-order must be carried by one of the co-located units; %d is not one of them", id)
				}
			}
			lo, hi := tc.pair[0].ID, tc.pair[1].ID
			if lo > hi {
				lo, hi = hi, lo
			}
			if lo+1 == hi || tc.third.ID <= lo || tc.third.ID >= hi {
				t.Fatalf("fixture: ids %d and %d must not be adjacent and %d must fall between them",
					lo, hi, tc.third.ID)
			}
			blkFree(t, "the control's parking cell", ents, blkParked[0], blkParked[1])
			if blkParked[0] == malShared[0] && blkParked[1] == malShared[1] {
				t.Fatalf("fixture: the control parks the pair on the shared cell (%d,%d)", malShared[0], malShared[1])
			}

			// What every unit must be holding, from the first tick to the last. A
			// self-order is cleared on the tick it is given and leaves no residue in
			// the coordinates, and a world zeroes the target of a unit that holds
			// none — so for both of the pair the fixture entity is the whole of what
			// must be there afterwards, self-order or no self-order. That comparison
			// is the one this file exists for: a rule that tested the cell a unit
			// stands on and excluded only itself would find the co-occupant, refuse
			// the zero-distance move and strand the order, leaving HasTarget set and
			// the coordinates naming the cell the unit is standing on.
			wants := make([]malWant, 0, 3)
			for _, e := range tc.pair {
				what := "given no order"
				for _, id := range tc.selfOrdered {
					if id == e.ID {
						what = "ordered onto the cell it stands on"
					}
				}
				wants = append(wants, malWant{fmt.Sprintf("co-located unit %d, %s:", e.ID, what), e})
			}
			held := tc.third
			held.TargetX, held.TargetY, held.HasTarget = toThird.X, toThird.Y, true
			wants = append(wants, malWant{fmt.Sprintf("the third unit %d:", tc.third.ID), held})

			w := mustWorld(t, 1, blkBounds, ents)
			states := make([][]Entity, 0, malTicks)
			digests := make([]uint64, 0, malTicks)
			for tick := 1; tick <= malTicks; tick++ {
				before := w.Entities()
				if tick == 1 {
					Step(w, cmds)
				} else {
					Step(w, nil)
				}
				after := w.Entities()

				malBounded(t, tick, before, after)
				for _, want := range wants {
					blkHolds(t, tick, want.what, occEntity(t, w, want.e.ID), want.e)
				}
				// The third unit was refused and still has somewhere to be, which is
				// what makes its unchanged fields a block rather than an arrival.
				blkUnfinished(t, tick, "the third unit", occEntity(t, w, tc.third.ID))
				if on := malOn(after, malShared[0], malShared[1]); len(on) != 2 || on[0] != lo || on[1] != hi {
					t.Errorf("tick %d: the shared cell (%d,%d) holds units %v, want exactly the two that loaded onto it, %d and %d",
						tick, malShared[0], malShared[1], on, lo, hi)
				}
				states, digests = append(states, after), append(digests, w.Hash())
			}

			// The third unit's stillness is a refusal and not a rule that moves
			// nobody: with the pair parked in the corner checked above, the same unit
			// under the same order reaches the shared cell. The pair is parked
			// target-less, which changes nothing about how it obstructs — a
			// self-order is cleared on the first tick and never moved anybody — and
			// it stays co-located there, so the one thing this control changes about
			// the world is the cell the two are standing on.
			parked := []Entity{tc.third}
			for _, e := range tc.pair {
				p := e
				p.X, p.Y = blkParked[0], blkParked[1]
				parked = append(parked, p)
			}
			blkMovesTo(t, "with the co-located pair parked the third unit",
				parked, toThird, malShared[0], malShared[1])

			// The same world and the same commands again: field for field and digest
			// for digest, because a malformed start is exactly where a rule reaching
			// for something that is not in the world would show up.
			again := mustWorld(t, 1, blkBounds, ents)
			for tick := 1; tick <= malTicks; tick++ {
				if tick == 1 {
					Step(again, cmds)
				} else {
					Step(again, nil)
				}
				for _, e := range states[tick-1] {
					blkHolds(t, tick, fmt.Sprintf("the re-run's unit %d:", e.ID), occEntity(t, again, e.ID), e)
				}
				if got := again.Hash(); got != digests[tick-1] {
					t.Errorf("tick %d: the re-run's digest is %#016x, want %#016x", tick, got, digests[tick-1])
				}
			}
		})
	}
}

// TestStepPartsACoLocatedPairOnlyWhereMovementPartsIt is the bound's other side.
// Advancement is not required to repair a malformed world, and it does not: what
// parts this pair is an order, and the count on the shared cell falls to one only
// because a unit standing there was sent somewhere else. The unit sent away carries
// the higher id, so what let it leave cannot be a priority it won — it was refused
// nothing, because the cell it wanted was empty.
func TestStepPartsACoLocatedPairOnlyWhereMovementPartsIt(t *testing.T) {
	stays := Entity{ID: 3, X: malShared[0], Y: malShared[1]}
	leaves := Entity{ID: 9, X: malShared[0], Y: malShared[1]}
	away := Command{Entity: leaves.ID, X: malShared[0] + 3, Y: malShared[1] - 3}
	ents := []Entity{stays, leaves}

	if cx, cy := nearerCell(leaves, away.X, away.Y); cx == leaves.X && cy == leaves.Y {
		t.Fatalf("fixture: the order (%d,%d) names the cell the unit stands on — it must send it away",
			away.X, away.Y)
	}
	blkFree(t, "the cell it is sent to", ents, away.X, away.Y)

	w := mustWorld(t, 1, blkBounds, ents)
	for tick := 1; tick <= malTicks; tick++ {
		before := w.Entities()
		if tick == 1 {
			Step(w, []Command{away})
		} else {
			Step(w, nil)
		}
		malBounded(t, tick, before, w.Entities())
		blkHolds(t, tick, "the unit given no order:", occEntity(t, w, stays.ID), stays)
	}

	if got := occEntity(t, w, leaves.ID); got.X != away.X || got.Y != away.Y || got.HasTarget {
		t.Errorf("the unit ordered away is at (%d,%d) holding %v (%d,%d), want (%d,%d) with the target cleared — a co-located unit whose desired cell is free walks off like any other",
			got.X, got.Y, got.HasTarget, got.TargetX, got.TargetY, away.X, away.Y)
	}
	if on := malOn(w.Entities(), malShared[0], malShared[1]); len(on) != 1 || on[0] != stays.ID {
		t.Errorf("the shared cell (%d,%d) holds units %v, want only %d — the pair parts where movement parts it, and nowhere else",
			malShared[0], malShared[1], on, stays.ID)
	}
}
