package sim

// Whether a stored route is state or a cache, measured rather than argued.
//
// The fork is this. If recomputing a route from a later cell of that same route
// gives the tail back, then a stored route is a derivation: it need not be
// hashed, the byte form need not carry it, and a world reloaded mid-order walks
// exactly what it was walking. If it does not, then a world saved mid-order and
// reloaded walks a DIFFERENT route from one that was never interrupted, and the
// route has to be in the hashed state.
//
// The argument for the cache is that the far search is unit-blind and terrain
// cannot change while a world is advanced. Its hole is that the mover advances,
// so the start moves — and the wave stops in the first generation that labels
// the goal, so which route it returns depends on where it started.
//
// So this file computes a route, then recomputes it from every one of its own
// cells and compares each with the stored tail. It reports what it counted and
// asserts the property it is measuring: a tail that is not recoverable. A run
// where every recomputation gave the tail back would be a finding about the
// searches, not a pass — it would say the route need not be hashed after all.
//
// TWO CLASSES of disagreement are counted, and only one of them is required.
// The weaker is an equal-cost route through different cells, which the
// first-generation stop produces from any start. The stronger is a
// recomputation REFUSED outright, and that class exists because the budget
// SHRANK as a mover closed on its goal: a detour that fitted from the cell a
// route was computed at could stop fitting from a cell along it. A flat budget
// removes that class by construction — the goal is still terrain-open, the tail
// proves the component is connected, and no hop count on this corpus approaches
// a thousand — so the refusal count is REPORTED here and not required. Requiring
// it would be asserting the rule this revision deletes.
//
// The corpus is built from this package's own seeded generator, reads no install
// and no clock, and every world holds exactly one entity, so the relation is
// terrain either way.

import "testing"

// rfkShape is one slice of the corpus: how big the maps are, how much of them
// blocks ground, and how many worlds to draw.
type rfkShape struct {
	side    int32
	blocked uint64 // per cent
	worlds  int
}

// rfkCorpus varies size and density rather than repeating one shape. Density is
// what the fork turns on: a route through open ground is a straight line whose
// tails are straight lines, and it is a detour that a shrinking budget can stop
// fitting.
var rfkCorpus = []rfkShape{
	{16, 0, 1800},
	{16, 10, 1800},
	{16, 25, 1800},
	{16, 35, 1800},
	{32, 10, 1050},
	{32, 25, 1050},
	{32, 35, 1050},
	{64, 15, 390},
	{64, 30, 390},
	{64, 38, 390},
}

// rfkWorld draws one world of the given shape: a grid, and one entity somewhere
// on it.
func rfkWorld(t *testing.T, r *rng, sh rfkShape) *World {
	t.Helper()
	grid := make([]byte, sh.side*sh.side)
	for i := range grid {
		if r.next()%100 < sh.blocked {
			grid[i] = blockGround
		}
	}
	return mustWorldGrid(t, r.next(), Bounds{Width: sh.side, Height: sh.side}, ModeCanonical, grid,
		[]Entity{{ID: 1, X: int32(r.next() % uint64(sh.side)), Y: int32(r.next() % uint64(sh.side))}})
}

// rfkTally is what one slice of the corpus came to.
type rfkTally struct {
	worlds, routes, pairs, differ, refused int
}

func (a *rfkTally) add(b rfkTally) {
	a.worlds += b.worlds
	a.routes += b.routes
	a.pairs += b.pairs
	a.differ += b.differ
	a.refused += b.refused
}

// rfkEqual compares a recomputed route with the tail it is supposed to be, cell
// for cell. A length comparison or a cost comparison would both pass a different
// walk of the same length, which is exactly the disagreement most of these are.
func rfkEqual(a, b []cell) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// rfkMeasure runs one slice: for each world, a far search from the entity's own
// cell to a drawn target, and then the same search from every cell of the route
// it produced, compared with the stored tail from that cell on.
func rfkMeasure(t *testing.T, r *rng, sh rfkShape) rfkTally {
	t.Helper()
	out := rfkTally{worlds: sh.worlds}

	for k := 0; k < sh.worlds; k++ {
		w := rfkWorld(t, r, sh)
		s := newRouteScratch(w)
		tx, ty := int32(r.next()%uint64(sh.side)), int32(r.next()%uint64(sh.side))

		route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, tx, ty)
		if !ok || len(route) == 0 {
			continue
		}
		out.routes++

		// The stored route belongs to the world as it was; the entity is walked
		// along it exactly as a tick would walk it, and each recomputation is the
		// same call the tick would make from there.
		for i := range route {
			w.entities[0].X, w.entities[0].Y = route[i].x, route[i].y
			s.occupy(w)
			out.pairs++

			again, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, tx, ty)
			if !ok {
				out.refused++
				out.differ++
				continue
			}
			if !rfkEqual(again, route[i+1:]) {
				out.differ++
			}
		}
	}
	return out
}

// TestAStoredRouteIsNotRecoverableFromItsOwnLaterCells is SC-3: the measurement
// that settled the fork, run in the tree rather than quoted in a document.
//
// What it asserts is the property, not a number. The pair count is asserted so
// that a corpus quietly shrinking to nothing cannot pass, and the disagreements
// are asserted to be non-zero because a stored route being unrecoverable is the
// whole reason it is hashed state. The refusals are logged beside them as their
// own class. Every figure is logged, because the RATE is the interesting part
// and no assertion here pins one.
//
// The corpus, its shapes and its seed are exactly SC-3's. Under the shrinking
// budget this same corpus gave 107208 pairs, 70 differing tails and 57 refusals
// — the figures that budget was worth, kept here so the two runs are one
// measurement at two revisions rather than two measurements.
func TestAStoredRouteIsNotRecoverableFromItsOwnLaterCells(t *testing.T) {
	r := &rng{state: 0xf0a5e}
	var total rfkTally

	for _, sh := range rfkCorpus {
		got := rfkMeasure(t, r, sh)
		t.Logf("%2dx%-2d %2d%% blocked: %5d worlds, %5d routes, %6d pairs, %4d tails differ, %3d refused",
			sh.side, sh.side, sh.blocked, got.worlds, got.routes, got.pairs, got.differ, got.refused)
		total.add(got)
	}
	t.Logf("total: %d worlds, %d routes, %d pairs, %d tails differ, %d of those refused outright",
		total.worlds, total.routes, total.pairs, total.differ, total.refused)

	if total.pairs < 100000 {
		t.Errorf("the corpus produced %d (route, cell) pairs, and this measurement is claimed over 100000",
			total.pairs)
	}
	if total.differ == 0 {
		t.Error("every recomputed route gave its own tail back. That is a FINDING and not a pass: it " +
			"says a stored route is recoverable from the cells it stands on, and a route that is " +
			"recoverable need not be hashed, versioned or carried by the byte form at all")
	}
	t.Logf("refusals are reported and not required: %d of the %d differing tails were refused "+
		"outright. That class is the shrinking budget's, and a flat budget takes it to zero "+
		"without touching the property above", total.refused, total.differ)
}
