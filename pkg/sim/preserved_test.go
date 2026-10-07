package sim

// Three contended runs, and what they are for now.
//
// This file was a PRESERVATION witness: two changes made the search cheaper
// without being meant to make it different, and the digests below were recorded
// from the tree as it stood before either of them, so the current code had to
// reproduce numbers it had never produced. That claim is over, and it was ended
// deliberately rather than eroded. The two-tier arrangement changes what these
// very runs do:
//
//   - a group ordered onto a cell another unit is standing on WALKS there and
//     crowds, where every member used to stand still from the first tick — the
//     far search reads terrain and cannot see the unit on the cell;
//   - an order onto a cell no route may enter is now answered by a far search
//     that fails, and its outcome is settled in the tick that finds it rather
//     than sixteen stalls later.
//
// So the digest tables are gone rather than re-recorded against the tree they
// were supposed to be measuring — a pin captured from the code under test
// witnesses nothing, and keeping the shape while dropping the meaning is worse
// than dropping both. What the file keeps is the runs themselves and the claim
// that made them worth pinning: they contend, they hold, and an order in them
// runs out. The digests come back below, against the contract as it settles.
//
// The scenarios are chosen for what they contend over rather than for size: units
// crossing each other's routes around a wall, an order onto a cell no route may
// enter, and a whole group ordered onto a cell another unit is standing on.

import "testing"

// prsScenario is one seeded run: the world it starts from, the orders it opens
// with, and how many ticks it advances.
type prsScenario struct {
	name   string
	bounds Bounds
	mode   Mode
	grid   []byte
	ents   []Entity
	cmds   []Command
	ticks  int
}

// prsCrossGrid is a five-cell wall standing in the middle of a sixteen by
// sixteen, short enough that a detour round it fits the wave's budget.
func prsCrossGrid() []byte {
	g := make([]byte, 16*16)
	for y := int32(4); y <= 8; y++ {
		g[y*16+8] = blockGround
	}
	return g
}

// prsCross is the contended run: four units ordered as a group to one cell on the
// far side of the wall, one unit walking a clear line along the bottom, and one
// ordered onto a cell of the wall itself — which no route may enter, so it holds
// for fifteen ticks and gives up on the sixteenth.
func prsCross(mode Mode) prsScenario {
	return prsScenario{
		name:   "four across a wall, one along the floor, one onto the wall",
		bounds: Bounds{Width: 16, Height: 16},
		mode:   mode,
		grid:   prsCrossGrid(),
		ents: []Entity{
			{ID: 1, X: 2, Y: 6}, {ID: 3, X: 2, Y: 7},
			{ID: 5, X: 3, Y: 6}, {ID: 7, X: 3, Y: 7},
			{ID: 9, X: 1, Y: 15}, {ID: 11, X: 6, Y: 1},
		},
		cmds: []Command{
			{Entity: 1, X: 13, Y: 6}, {Entity: 3, X: 13, Y: 6},
			{Entity: 5, X: 13, Y: 6}, {Entity: 7, X: 13, Y: 6},
			{Entity: 9, X: 14, Y: 15}, {Entity: 11, X: 8, Y: 6},
		},
		ticks: 20,
	}
}

// prsHeld is the group ordered onto a cell somebody is standing on: five movers,
// one unit parked on the destination with no order of its own, one unit walking
// an unrelated line so the run is not a standstill, and twenty ticks — four more
// than it takes every member of the group to give up.
func prsHeld() prsScenario {
	return prsScenario{
		name:   "five ordered onto one held cell",
		bounds: Bounds{Width: 12, Height: 12},
		mode:   ModeCanonical,
		ents: []Entity{
			{ID: 1, X: 6, Y: 6},
			{ID: 3, X: 1, Y: 1}, {ID: 5, X: 2, Y: 1}, {ID: 7, X: 3, Y: 1},
			{ID: 9, X: 1, Y: 2}, {ID: 11, X: 2, Y: 2},
			{ID: 13, X: 1, Y: 10},
		},
		cmds: []Command{
			{Entity: 3, X: 6, Y: 6}, {Entity: 5, X: 6, Y: 6}, {Entity: 7, X: 6, Y: 6},
			{Entity: 9, X: 6, Y: 6}, {Entity: 11, X: 6, Y: 6},
			{Entity: 13, X: 10, Y: 10},
		},
		ticks: 20,
	}
}

// prsAll is the set.
func prsAll() []prsScenario {
	optimised := prsCross(ModeOptimised)
	optimised.name += ", optimised"
	return []prsScenario{prsCross(ModeCanonical), optimised, prsHeld()}
}

// TestTheseRunsContendStallAndGiveUp is the condition any pin over these runs
// rests on, and it is worth keeping on its own: a table witnesses nothing unless
// the run that produced it exercised the paths in question, and a run of six
// units standing still would match its pin perfectly.
//
// A give-up is counted across the three runs rather than demanded of each. Which
// run ends an order and how is exactly what this story is changing, and a
// condition that pinned that would be asserting the contract twice — once here
// and once where it belongs.
func TestTheseRunsContendStallAndGiveUp(t *testing.T) {
	gaveUpAnywhere := 0
	for _, sc := range prsAll() {
		w, err := NewWorld(1, sc.bounds, sc.mode, sc.grid, sc.ents)
		if err != nil {
			t.Fatalf("%s: NewWorld: %v", sc.name, err)
		}
		moved, stalled, gaveUp := 0, 0, 0
		before := w.Entities()
		for tick := 1; tick <= sc.ticks; tick++ {
			if tick == 1 {
				Step(w, sc.cmds)
			} else {
				Step(w, nil)
			}
			after := w.Entities()
			for i := range after {
				switch {
				case after[i].X != before[i].X || after[i].Y != before[i].Y:
					moved++
				case after[i].Stall > before[i].Stall:
					stalled++
				case before[i].Stall > 0 && after[i].Stall == 0 && !after[i].HasTarget:
					gaveUp++
				}
			}
			before = after
		}
		if moved == 0 || stalled == 0 {
			t.Errorf("%s: %d advances, %d holds — a pin over a run that skips a path witnesses nothing there",
				sc.name, moved, stalled)
		}
		gaveUpAnywhere += gaveUp
	}
	if gaveUpAnywhere == 0 {
		t.Error("no order ran out in any of the three runs, so none of them reaches the give-up at all")
	}
}
