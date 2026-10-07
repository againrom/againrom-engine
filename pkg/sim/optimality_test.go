package sim

// AC-9: the cost a mode actually walks, against the minimum over all admissible
// routes.
//
// It enumerates over the WHOLE map, because the search it is a reference for is
// no longer bounded in space: the start/target rectangle both used to be
// confined to is gone.
//
// The enumeration is exhaustive by cost: it deepens a bound one unit at a time
// from a floor no route can beat, and the first bound that reaches the target is
// the minimum. Inside a bound it walks every admissible simple route, pruning
// only where the cheapest conceivable remainder already exceeds it — which
// removes no optimal route. A minimum-cost route never repeats a cell, since
// every step costs something, so simple routes are all there is to enumerate.
//
// The grids are small enough to exhaust. That is the entry's own limit, not a
// convenience: a grid this cannot finish would turn the reference into a second
// heuristic, and then nothing here would be a check.

import "testing"

// The enumeration's own constants. They repeat values the searches also use,
// deliberately: a reference that imported them would agree with the code under
// test about a cost model neither had proved.
const (
	optBlocksGround byte = 1
	optStraight     int  = 2
	optDiagonal     int  = 3
)

// optMap is the enumeration's own view of a world: extents, blocking bits, and
// nothing else. There are no other units in any fixture here, so enterability is
// bounds and the ground bit.
type optMap struct {
	w, h  int32
	cells []byte
}

func (m optMap) enterable(x, y int32) bool {
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		return false
	}
	return m.cells[y*m.w+x]&optBlocksGround == 0
}

func optCost(dx, dy int32) int {
	if dx != 0 && dy != 0 {
		return optDiagonal
	}
	return optStraight
}

func optChebyshev(a, b cell) int32 {
	dx, dy := a.x-b.x, a.y-b.y
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

// optMinimum is the least total cost of an admissible route from s to t, and
// whether one exists. Admissibility is the extents, the blocking bit and the
// corner rule; no rectangle takes part.
func optMinimum(m optMap, s, t cell) (int, bool) {
	seen := map[cell]bool{s: true}
	// A simple route visits at most every cell once, so no admissible route can
	// cost more than the dearest step times the cell count.
	ceiling := optDiagonal * int(m.w) * int(m.h)

	var reach func(cur cell, cost, bound int) bool
	reach = func(cur cell, cost, bound int) bool {
		// Every remaining step costs at least a straight one, and at least the
		// Chebyshev distance of them are left, so this prunes no cheaper route.
		// It runs BEFORE the arrival test, or a route dearer than the bound
		// would be accepted at it and every minimum would come out low.
		if cost+optStraight*int(optChebyshev(cur, t)) > bound {
			return false
		}
		if cur == t {
			return true
		}
		for dx := int32(-1); dx <= 1; dx++ {
			for dy := int32(-1); dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				n := cell{cur.x + dx, cur.y + dy}
				if seen[n] || !m.enterable(n.x, n.y) {
					continue
				}
				if dx != 0 && dy != 0 &&
					(!m.enterable(cur.x+dx, cur.y) || !m.enterable(cur.x, cur.y+dy)) {
					continue
				}
				seen[n] = true
				found := reach(n, cost+optCost(dx, dy), bound)
				delete(seen, n)
				if found {
					return true
				}
			}
		}
		return false
	}

	for bound := optStraight * int(optChebyshev(s, t)); bound <= ceiling; bound++ {
		if reach(s, 0, bound) {
			return bound, true
		}
	}
	return 0, false
}

// optWalked advances a world of the given mode to arrival and returns the total
// cost of the cells it actually stood on, tick by tick. It is the walk that is
// measured, not a route: a route is searched afresh each tick, so the sequence
// of cells a unit occupies is the only thing a mode can be held to.
func optWalked(t *testing.T, m optMap, mode Mode, from, to cell) (int, int, bool) {
	t.Helper()
	b := Bounds{Width: m.w, Height: m.h}
	w := mustWorldGrid(t, 1, b, mode, m.cells, []Entity{{ID: 1, X: from.x, Y: from.y}})

	cost, prev := 0, from
	for tick := 1; tick <= 3*int(m.w)*int(m.h); tick++ {
		if tick == 1 {
			Step(w, []Command{{Entity: 1, X: to.x, Y: to.y}})
		} else {
			Step(w, nil)
		}
		e := w.Entities()[0]
		at := cell{e.X, e.Y}
		if at != prev {
			cost += optCost(at.x-prev.x, at.y-prev.y)
			prev = at
		}
		if at == to {
			return cost, tick, true
		}
		if !e.HasTarget {
			return cost, tick, false // gave up
		}
	}
	return cost, 0, false
}

// optGrid builds a map of the given extents with the named cells blocking
// ground.
func optGrid(w, h int32, blocked ...cell) optMap {
	m := optMap{w: w, h: h, cells: make([]byte, w*h)}
	for _, c := range blocked {
		m.cells[c.y*w+c.x] = optBlocksGround
	}
	return m
}

// zigzagGrid is SC-7's purpose-built grid, and it is built to be worse rather
// than found to be worse: a random sweep does not produce one, because the
// canonical wave is normally at or under the admissible minimum — it is allowed
// to cut corners, and a corner cut is not an admissible route.
//
//	. . . . . . . . . . . . .      row 0, the corridor: open end to end
//	. # # # # # # # # # # # .      row 1, walled but for the two connectors
//	@ # . # . # . # . # . # T      rows 2 and 3, a checkerboard chain in which
//	. # . # . # . # . # . # .      every step east must be diagonal
//
// The wave stops in the first generation that labels the target, so it sees only
// the twelve-step chain and walks it: twelve diagonals at 36. That chain is not
// admissible at all — every one of its steps passes between two blocked cells —
// so the least admissible route is the corridor, sixteen straight steps at 32.
// The crossover sits between six and twelve chain steps; twelve is past it.
func zigzagGrid() optMap {
	m := optMap{w: 13, h: 4, cells: make([]byte, 13*4)}
	block := func(x, y int32) { m.cells[y*13+x] = optBlocksGround }
	for x := int32(0); x < 13; x++ {
		if x != 0 && x != 12 {
			block(x, 1)
		}
		if x%2 == 1 {
			block(x, 2)
		} else {
			block(x, 3)
		}
	}
	return m
}

// TestTheOptimisedWalkCostsTheAdmissibleMinimum is AC-9's first half, over every
// grid; the second half needs a grid of its own and has one below.
func TestTheOptimisedWalkCostsTheAdmissibleMinimum(t *testing.T) {
	for _, tc := range []struct {
		what     string
		m        optMap
		from, to cell
	}{
		{"open ground", optGrid(3, 3), cell{0, 0}, cell{2, 2}},
		{"a corner it may not cut", optGrid(3, 3, cell{1, 0}), cell{0, 0}, cell{1, 1}},
		{"one cell in the way", optGrid(4, 4, cell{1, 1}), cell{0, 0}, cell{2, 2}},
		{"a wall with both ends open", optGrid(5, 5, cell{2, 1}, cell{2, 2}, cell{2, 3}), cell{0, 2}, cell{4, 2}},
		{"a diagonal chain", optGrid(5, 5, cell{1, 1}, cell{2, 2}, cell{3, 3}), cell{0, 4}, cell{4, 0}},
		{"the zigzag racing a corridor", zigzagGrid(), cell{0, 2}, cell{12, 2}},
	} {
		want, ok := optMinimum(tc.m, tc.from, tc.to)
		if !ok {
			t.Errorf("%s: the enumeration found no admissible route, so the grid measures nothing", tc.what)
			continue
		}
		got, ticks, arrived := optWalked(t, tc.m, ModeOptimised, tc.from, tc.to)
		if !arrived {
			t.Errorf("%s: the optimised walk did not arrive (%d tick(s), cost %d)", tc.what, ticks, got)
			continue
		}
		if got != want {
			t.Errorf("%s: the optimised walk cost %d, the admissible minimum is %d", tc.what, got, want)
		}
	}
}

// TestTheCanonicalWalkCostsMoreThanTheMinimumOnItsOwnGrid is AC-9's second half.
//
// The two numbers are pinned as well as compared, because "strictly greater"
// alone would also be satisfied by a grid where both had drifted.
func TestTheCanonicalWalkCostsMoreThanTheMinimumOnItsOwnGrid(t *testing.T) {
	m := zigzagGrid()
	from, to := cell{0, 2}, cell{12, 2}

	best, ok := optMinimum(m, from, to)
	if !ok {
		t.Fatal("the enumeration found no admissible route on the zigzag grid")
	}
	if best != 32 {
		t.Errorf("the admissible minimum is %d, want 32 — sixteen straight steps along the corridor", best)
	}

	canonical, ticks, arrived := optWalked(t, m, ModeCanonical, from, to)
	if !arrived {
		t.Fatalf("the canonical walk did not arrive (%d tick(s), cost %d)", ticks, canonical)
	}
	if canonical != 36 || ticks != 12 {
		t.Errorf("the canonical walk cost %d over %d tick(s), want 36 over 12 — twelve diagonals",
			canonical, ticks)
	}
	if canonical <= best {
		t.Errorf("the canonical walk cost %d and the minimum is %d, so this grid does not separate them",
			canonical, best)
	}

	// And the other mode walks the corridor on the same grid, which is what
	// makes the gap a property of the two searches rather than of the fixture.
	optimised, oticks, oarrived := optWalked(t, m, ModeOptimised, from, to)
	if !oarrived || optimised != best || oticks != 16 {
		t.Errorf("the optimised walk cost %d over %d tick(s) (arrived=%v), want %d over 16",
			optimised, oticks, oarrived, best)
	}
}
