package sim

// The optimised route search: a minimum-cost admissible route, taken by a
// Dijkstra outward from the TARGET and read back by a greedy walk forward from
// the start.
//
// It is ours by choice and it reconstructs nothing. Where the canonical wave
// stops at the first generation that puts any label on the destination — so the
// route it returns is the cheapest of at most that many steps and not the
// cheapest there is — this one is exact over everything it may look at, refuses
// to cut a corner, and answers "no route" where the canonical wave would have
// walked something.
//
// Like the wave it is a plain function in this package's own directory, in no
// sub-package and behind no interface, for the reason route.go states: a
// sub-package would satisfy the import DAG and escape the source scan that holds
// a search to integer arithmetic. It takes the same per-tick scratch, so the
// label plane a tick allocates is the one both searches use.

import "container/heap"

// The start/target rectangle this search used to be confined to — the smallest
// one containing both, grown by a margin of eight and clipped to the bounds — is
// GONE, and removing it is what this story exists for. As a bound it answered
// "no route" for a goal reachable only by a detour bulging past the margin,
// which on a map with a lake is not a corner case; and a search that must find
// such a detour is exactly the one that may not be bounded in space. What bounds
// a search here now is the window it is handed, which the far search does not
// carry and the near one does.

// cornerClear reports whether a step by (dx, dy) out of (x, y) is one this mode
// allows to be taken at all: an orthogonal step always, a diagonal one only when
// both of the cells it passes between are open under r.
//
// The two cells are (x+dx, y) and (x, y+dy), which is the same pair read from
// either end of the step. That symmetry is what lets the whole search run
// backwards from the target over the same graph the route runs forwards over.
//
// It asks the SAME relation the cell test asks. A corner read over occupancy
// while the cell was read over terrain would be a third relation nobody
// declared, and a route admissible under neither.
func (w *World) cornerClear(s *routeScratch, r relation, self int, x, y, dx, dy int32) bool {
	if dx == 0 || dy == 0 {
		return true
	}
	return w.open(s, r, self, x+dx, y) && w.open(s, r, self, x, y+dy)
}

// costItem is one heap entry: a cell and the cost it was pushed with.
type costItem struct {
	cost uint64
	c    cell
}

// costHeap orders by cost, then by y, then by x.
//
// The two coordinate terms are INERT — Dijkstra's distances do not depend on
// which of two equal-cost cells is popped first, because either pop relaxes the
// same neighbours to the same values. They are here so that the pop order is a
// total order of the data rather than an artefact of how the heap happens to
// sift, and a mutant that drops them is expected to survive.
type costHeap []costItem

func (h costHeap) Len() int      { return len(h) }
func (h costHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h costHeap) Less(i, j int) bool {
	if h[i].cost != h[j].cost {
		return h[i].cost < h[j].cost
	}
	if h[i].c.y != h[j].c.y {
		return h[i].c.y < h[j].c.y
	}
	return h[i].c.x < h[j].c.x
}

func (h *costHeap) Push(x any) { *h = append(*h, x.(costItem)) }

func (h *costHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}

// optimisedRoute is the minimum-cost admissible route from the entity at index
// self to (tx, ty) over the relation r, inside a window of half-width half on
// the mover's own cell, or ok=false when there is none.
//
// A route is admissible when every cell after the start is open under r, every
// cell after the start lies inside that window, and every diagonal step has both
// of the cells it passes between open under r. Among the minimum-cost ones the
// route taken is the one whose sequence of cells after the start is smallest
// compared cell by cell as (y, x).
//
// With half = noWindow the second clause is vacuous and the search is over the
// whole map: no rectangle, no region, no bound in space at all.
//
// It runs BACKWARDS. Costs are computed outward from the target with a binary
// heap over the cells the window permits, which is sound because a step's cost
// depends only on its diagonality and the corner condition is symmetric in the
// two cells the diagonal passes between — so the reverse graph is the forward
// one. The plane then holds, for every reached cell, the cost of the cheapest
// admissible route from THAT cell to the target.
//
// The start is not in that graph. It may be a cell no route may stand on and it
// may be off the map entirely, so it has no distance of its own until one is
// taken as the minimum over its legal first steps — which is exactly what routes
// a unit off a cell it may not stand on.
//
// The route is then read forward, and greedily: at each cell take, among the
// neighbours whose step keeps the running total at the minimum, the smallest
// (y, x). Sequences are compared from the start, so the first cell that differs
// decides the whole comparison and a greedy choice at each step is the exact
// answer rather than an approximation of it. No route is enumerated and no set
// of routes is sorted.
//
// The plane is left holding this search's costs, and the next search's reset is
// what clears them — the same arrangement the wave uses, and the reason a reset
// costs what was written rather than what the map could hold.
func (w *World) optimisedRoute(s *routeScratch, self int, r relation, half int32, settle settleRule, tx, ty int32) ([]cell, bool) {
	start := cell{x: w.entities[self].X, y: w.entities[self].Y}
	target := cell{x: tx, y: ty}
	win := window{centre: start, half: half}

	s.reset()
	if start == target {
		// The empty route, with ok true. Nothing calls it this way — an entity
		// standing on its target neither searches nor steps — but the consistent
		// answer beats an error, and it is the answer the wave gives too.
		return nil, true
	}
	// Every route ends at the target, so a target outside the window or one no
	// route may stand on is no route before any search is run. A world with no
	// in-bounds cell at all needs no test of its own: nothing is open there.
	if !win.holds(target.x, target.y) || !w.open(s, r, self, target.x, target.y) {
		return nil, false
	}
	// A target the mover may not come to REST on is the same answer, and this is
	// the one place that question is asked here. It is asked only of the call
	// that would settle if this search settled -- the FAR one -- because the near
	// search may not refuse a cell a flyer is merely passing over.
	//
	// This search takes no substitute: its plane holds costs TO the target and
	// says nothing about what the mover can reach, so the distinctness it
	// delivers is a REFUSAL and the order ends in the tick that gets it. Without
	// this term the stored route would be discarded and rebuilt every tick by the
	// staleness test that reads the same question, and this search has no
	// generation budget to bound that with.
	if settle == settleOrdered && !w.restFree(s, self, target.x, target.y) {
		return nil, false
	}

	// The mover's domain, read once, exactly as the canonical search reads it.
	dom := w.entities[self].Domain

	ti, _ := w.cellIndex(target.x, target.y) // open implies in bounds
	s.plane[ti] = 1                          // cost 0, held as cost+1
	s.touched = append(s.touched, ti)

	h := &costHeap{{cost: 0, c: target}}
	for h.Len() > 0 {
		it := heap.Pop(h).(costItem)
		i, _ := w.cellIndex(it.c.x, it.c.y)
		if s.plane[i]-1 != it.cost {
			// A stale entry: this cell was lowered after it was pushed. Lazy
			// deletion keeps the heap free of any index to maintain.
			continue
		}
		for dx := int32(-1); dx <= 1; dx++ {
			for dy := int32(-1); dy <= 1; dy++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx, ny := it.c.x+dx, it.c.y+dy
				if !win.holds(nx, ny) || !w.open(s, r, self, nx, ny) {
					continue
				}
				if !w.cornerClear(s, r, self, it.c.x, it.c.y, dx, dy) {
					continue
				}
				ni, _ := w.cellIndex(nx, ny)
				// THIS FLOOD RUNS BACKWARD, from the target outward, so its
				// labels are costs TO the target and the step being priced here
				// goes from the neighbour INTO it.c. The entered cell is
				// therefore it.c and not the neighbour — the opposite loop
				// variable from the canonical wave's, which is why the cell is
				// named at the site rather than inferred from the deltas.
				held := it.cost + stepCost(dom, w.costAt(it.c), dx, dy) + 1
				if s.plane[ni] != 0 && held >= s.plane[ni] {
					continue
				}
				if s.plane[ni] == 0 {
					s.touched = append(s.touched, ni)
				}
				s.plane[ni] = held
				heap.Push(h, costItem{cost: held - 1, c: cell{x: nx, y: ny}})
			}
		}
	}

	total, found := uint64(0), false
	w.eachLegalStep(s, win, r, self, start, func(_ cell, cost uint64) {
		if !found || cost < total {
			total, found = cost, true
		}
	})
	if !found {
		return nil, false
	}

	var route []cell
	for cur := start; cur != target; {
		var next cell
		got := false
		w.eachLegalStep(s, win, r, self, cur, func(n cell, cost uint64) {
			if cost != total {
				return
			}
			if !got || n.y < next.y || (n.y == next.y && n.x < next.x) {
				next, got = n, true
			}
		})
		if !got {
			// Unreachable: the running total is a cost some step realises, by
			// the way it was obtained.
			return nil, false
		}
		route = append(route, next)
		ni, _ := w.cellIndex(next.x, next.y)
		total = s.plane[ni] - 1
		cur = next
	}
	return route, true
}

// eachLegalStep calls yield for each neighbour of from that a route may step to
// — inside the window, open under r, its corner clear, and holding a cost — with
// the total cost of taking that step and then the cheapest route on from there.
//
// The window is tested here as well as where the costs were written, so the walk
// forward refuses exactly the cells the flood refused: a route may not leave
// what could not be labelled, and that is said where a route's cells are chosen
// rather than left to follow from which cells happen to carry a cost.
//
// It is one function because the start's own distance and every later choice ask
// the same question of the same graph. Two copies of this loop is how the first
// step comes to obey a rule the rest do not, or the reverse.
func (w *World) eachLegalStep(s *routeScratch, win window, r relation, self int, from cell, yield func(cell, uint64)) {
	dom := w.entities[self].Domain
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := from.x+dx, from.y+dy
			if !win.holds(nx, ny) || !w.open(s, r, self, nx, ny) {
				continue
			}
			if !w.cornerClear(s, r, self, from.x, from.y, dx, dy) {
				continue
			}
			ni, _ := w.cellIndex(nx, ny)
			if s.plane[ni] == 0 {
				continue
			}
			// The walk over these labels runs FORWARD, from the mover toward
			// the target, so the cell each step enters is the neighbour and the
			// charge is its own — the mirror of the flood above, and the same
			// rule: whichever cell the step's direction of travel enters.
			yield(cell{x: nx, y: ny}, stepCost(dom, w.searchCostAt(ni, cell{x: nx, y: ny}), dx, dy)+s.plane[ni]-1)
		}
	}
}
