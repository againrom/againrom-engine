package sim

// What a tick costs, counted rather than timed.
//
// Two of this story's acceptance criteria are performance criteria, and one half
// of that pair is machine-independent: how many cells a near search may label,
// and how many searches of each kind a whole run may run. Those are counted
// here. The milliseconds are the harness's business, in grouporder_test.go, and
// nothing in this file reads a clock.
//
// The labelled cells are read off the scratch's own touch list — the list a
// reset walks — so they are the work a search actually did rather than a count
// re-derived from the answer it gave.
//
// The searches are counted by asking the contract's own staleness test, subGoal,
// of every unit before every tick: that is the predicate the tick itself
// consults, asked at the moment the tick consults it. What it cannot see is a
// tick calling a search for some reason of its own — a second near search per
// unit, say. That is a property of the tick having exactly one call site for
// each kind, which this file states and the source shows; what it measures is
// everything else.

import "testing"

// ------------------------------------------------- the labelled cells

// cntSide is the large map AC-4's first half is measured on: big enough that an
// unbounded search over it labels thousands of cells, so "the window bounds the
// near search" is a difference of two orders of magnitude rather than a rounding.
const cntSide int32 = 256

// cntTouched is how many cells a search labelled and how far the furthest of
// them lies from a given centre.
func cntTouched(w *World, s *routeScratch, centre cell) (int, int32) {
	far := int32(0)
	for _, i := range s.touched {
		x, y := int32(i)%w.bounds.Width, int32(i)/w.bounds.Width
		dx, dy := x-centre.x, y-centre.y
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dy > dx {
			dx = dy
		}
		if dx > far {
			far = dx
		}
	}
	return len(s.touched), far
}

// TestANearSearchOnALargeMapLabelsAtMost289CellsAndNoneOutsideItsWindow is
// AC-4's first half, in both modes and against the far search on the same world
// at the same moment. The unit is MID-ORDER — a tick has been advanced, so it
// holds a route the far search gave it — and the near search measured is the one
// the next tick would run, with the sub-goal the tick would aim at.
func TestANearSearchOnALargeMapLabelsAtMost289CellsAndNoneOutsideItsWindow(t *testing.T) {
	b := Bounds{Width: cntSide, Height: cntSide}

	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, b, mode, nil, []Entity{{ID: 1, X: 4, Y: 4}})
		Step(w, []Command{{Entity: 1, X: cntSide - 5, Y: cntSide - 5}})

		e := w.Entities()[0]
		if !e.HasTarget || len(w.routes[0]) == 0 {
			t.Fatalf("mode %d: after one tick the unit is %+v with %d route cell(s); this measures a "+
				"unit mid-order", mode, e, len(w.routes[0]))
		}
		here := cell{e.X, e.Y}
		sub, serves := w.subGoal(newRouteScratch(w), 0)
		if !serves {
			t.Fatalf("mode %d: the stored route does not serve, so the next tick runs a far search", mode)
		}

		// The near search the next tick would run, with its own scratch so the
		// touch list holds this search and nothing else.
		s := newRouteScratch(w)
		if _, ok := w.searchRoute(s, 0, unitRelation, dynamicWindow, scaledBudget, exactGoal, sub.x, sub.y); !ok {
			t.Fatalf("mode %d: the near search found no route to its own sub-goal (%d,%d)", mode, sub.x, sub.y)
		}
		near, reach := cntTouched(w, s, here)
		if near > 289 || reach > 8 {
			t.Errorf("mode %d: the near search labelled %d cell(s), the furthest %d away; the window "+
				"admits 289 and reaches 8", mode, near, reach)
		}

		// The far search on the same world, from the same cell, at the same
		// moment: the order's real destination, no window, no bound.
		s = newRouteScratch(w)
		if _, ok := w.searchRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, e.TargetX, e.TargetY); !ok {
			t.Fatalf("mode %d: the far search found no route across open ground", mode)
		}
		far, farReach := cntTouched(w, s, here)
		if far <= near || farReach <= 8 {
			t.Errorf("mode %d: the far search labelled %d cell(s) reaching %d, against the near search's "+
				"%d reaching %d — the far search is the unbounded one",
				mode, far, farReach, near, reach)
		}
		t.Logf("mode %d on %dx%d: near %d cells (reach %d), far %d cells (reach %d)",
			mode, cntSide, cntSide, near, reach, far, farReach)
	}
}

// ------------------------------------------------- what a refused search labels

// cntSealedGrid is an open map with ONE cell walled in by blocking terrain: the
// eight cells around (gx, gy) block ground and (gx, gy) itself does not.
//
// The goal is therefore OPEN and unreachable, which is the only shape that
// measures anything here. A goal that itself blocked ground would be refused
// before any sweep and would label nothing, and the harness's own sealed shape
// is sealed by UNITS, which the far search cannot see at all — it routes to that
// goal and the movers walk. Terrain is what a terrain-only search can be stopped
// by.
func cntSealedGrid(gx, gy int32) []byte {
	g := make([]byte, cntSide*cntSide)
	for dy := int32(-1); dy <= 1; dy++ {
		for dx := int32(-1); dx <= 1; dx++ {
			if dx != 0 || dy != 0 {
				g[(gy+dy)*cntSide+(gx+dx)] = blockGround
			}
		}
	}
	return g
}

// cntBefore is what this same measurement came to on the tree BEFORE the far
// search's budget went flat, in the order the cases run below.
//
// They are literals from that run and are not recomputed here. The two columns
// are one measurement at two revisions: same map, same start, same goals, same
// count off the scratch's own touch list, and the only difference between the
// runs is the budget rule — which is the thing under test.
var cntBefore = [3]int{306, 65527, 63001}

func TestWhatBoundsARefusedFarSearchIsNowTheMap(t *testing.T) {
	b := Bounds{Width: cntSide, Height: cntSide}
	start := cell{4, 4}

	for k, tc := range []struct {
		what        string
		gx, gy      int32
		sealed      bool
		wantRefusal bool
	}{
		{"a sealed goal, near", 12, 12, true, true},
		{"a sealed goal, far", 250, 250, true, true},
		{"a search that succeeds", 250, 250, false, false},
	} {
		grid := cntSealedGrid(tc.gx, tc.gy)
		if !tc.sealed {
			grid = make([]byte, cntSide*cntSide)
		}
		w := mustWorldGrid(t, 1, b, ModeCanonical, grid, []Entity{{ID: 1, X: start.x, Y: start.y}})
		s := newRouteScratch(w)

		route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, exactGoal, tc.gx, tc.gy)
		if ok == tc.wantRefusal {
			t.Fatalf("%s: the search answered ok=%v, and this case is written for the opposite", tc.what, ok)
		}
		got, _ := cntTouched(w, s, start)
		t.Logf("%-24s D=%3d labelled %6d cells, where the scaled budget labelled %6d (%d hops)",
			tc.what, maxAbs(tc.gx-start.x, tc.gy-start.y), got, cntBefore[k], len(route))

		switch tc.what {
		case "a search that succeeds":
			if got != cntBefore[k] {
				t.Errorf("%s: labelled %d cells where the scaled budget labelled %d — a search that "+
					"succeeded under the old budget is meant to label EXACTLY what it labelled",
					tc.what, got, cntBefore[k])
			}
		case "a sealed goal, near":
			// The whole cost of the change, in one number: a near refusal used
			// to be bounded by a budget that shrank with the distance.
			if got <= cntBefore[k] {
				t.Errorf("%s: labelled %d cells against the scaled budget's %d — the flat budget is "+
					"meant to sweep the map here", tc.what, got, cntBefore[k])
			}
		}
		// However refused, a search cannot label more cells than the map holds.
		if max := int(cntSide) * int(cntSide); got > max {
			t.Errorf("%s: labelled %d cells on a map of %d", tc.what, got, max)
		}
	}
}

// maxAbs is the Chebyshev distance of a coordinate difference, for the log line
// above.
func maxAbs(dx, dy int32) int32 {
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

// ------------------------------------------------- the searches over a run

type cntStale int

const (
	cntServes cntStale = iota
	cntNoRoute
	cntWrongTarget
	cntDeparted
)

// cntWhy classifies one unit's stored route against the three staleness tests,
// so that a far search can be counted BY REASON and a departure is a measured
// number rather than an assumption.
//
// It is cross-checked against subGoal on every unit of every tick below: the
// classification is the test's, the verdict is the contract's, and a
// disagreement between them fails the run.
func cntWhy(w *World, i int) cntStale {
	e := &w.entities[i]
	route := w.routes[i]
	if len(route) == 0 {
		return cntNoRoute
	}
	if last := route[len(route)-1]; last.x != e.TargetX || last.y != e.TargetY {
		return cntWrongTarget
	}
	win := window{centre: cell{x: e.X, y: e.Y}, half: dynamicWindow}
	if sub := route[subGoalIndex(len(route))]; !win.holds(sub.x, sub.y) {
		return cntDeparted
	}
	return cntServes
}

// TestOverAWholeRunTheFarSearchesAreTheOrdersPlusTheDepartures is AC-4's second
// half: eight units on a 64x64, each ordered once, advanced until every order is
// over. The far searches are counted by reason and the near ones by unit-tick.
//
// What the count has to come to is one far search per unit ordered, plus one per
// departure — a sub-goal pushed out of its window — and nothing else. In
// particular a unit that failed to move buys none, which is the whole point of
// the staleness tests being tests on the route rather than a period.
func TestOverAWholeRunTheFarSearchesAreTheOrdersPlusTheDepartures(t *testing.T) {
	const side int32 = 64
	b := Bounds{Width: side, Height: side}

	// A few blobs, so the routes are detours and a mover can be pushed off one.
	grid := make([]byte, side*side)
	for _, blob := range []struct{ x, y, w, h int32 }{
		{20, 10, 6, 12}, {36, 30, 10, 5}, {14, 40, 5, 9}, {44, 12, 4, 4},
	} {
		for y := blob.y; y < blob.y+blob.h; y++ {
			for x := blob.x; x < blob.x+blob.w; x++ {
				grid[y*side+x] = blockGround
			}
		}
	}

	// One unit per column, seven columns apart, each ordered straight down its
	// own: eight orders that cross four of the blobs and no crowd at either end,
	// so every one of them arrives and the count is a whole run's.
	ents := make([]Entity, 0, 8)
	cmds := make([]Command, 0, 8)
	for k := 0; k < 8; k++ {
		id := EntityID(1 + k)
		x := 8 + int32(k)*7
		ents = append(ents, Entity{ID: id, X: x, Y: 3})
		cmds = append(cmds, Command{Entity: id, X: x, Y: side - 4})
	}
	w := mustWorldGrid(t, 1, b, ModeCanonical, grid, ents)

	byReason := map[cntStale]int{}
	unitTicks, farFailed := 0, 0
	for tick := 1; tick <= 120; tick++ {
		if tick == 1 {
			// Commands apply before the move, so the state a tick's searches are
			// decided against is the one the orders leave. The same assignment
			// Step's first phase makes is made here, and Step then makes it again
			// over identical values.
			for _, c := range cmds {
				e := &w.entities[indexOfEntity(w.entities, c.Entity)]
				e.TargetX, e.TargetY, e.HasTarget = c.X, c.Y, true
			}
		}
		before := w.Entities()
		ranFar := make([]bool, len(before))
		active := 0
		for i := range w.entities {
			e := &w.entities[i]
			if !e.HasTarget {
				continue
			}
			active++
			if e.X == e.TargetX && e.Y == e.TargetY {
				continue // it begins the tick on its target: no search of either kind
			}
			unitTicks++

			why := cntWhy(w, i)
			ranFar[i] = why != cntServes
			_, serves := w.subGoal(newRouteScratch(w), i)
			if serves != (why == cntServes) {
				t.Fatalf("tick %d: unit %d classified %v, and the contract's own test says serves=%v",
					tick, e.ID, why, serves)
			}
			if why != cntServes {
				byReason[why]++
			}
		}
		if active == 0 && tick > 1 {
			break
		}
		if tick == 1 {
			Step(w, cmds)
		} else {
			Step(w, nil)
		}

		// A unit whose FAR search found nothing loses its order without moving,
		// and it is the one shape of unit-tick that runs no near search at all. A
		// give-up looks the same from outside and is not the same thing, so the
		// two are told apart by whether a far search ran on that tick at all.
		for i, e := range w.Entities() {
			if ranFar[i] && !e.HasTarget && e.X == before[i].X && e.Y == before[i].Y {
				farFailed++
			}
		}
	}
	// One near search per unit-tick that reached it. The tick has exactly ONE
	// call site for a near search, so no unit-tick can run two; what this count
	// says is that none runs zero either, except where the far search ended the
	// order first.
	near := unitTicks - farFailed

	far := byReason[cntNoRoute] + byReason[cntWrongTarget] + byReason[cntDeparted]
	t.Logf("8 units on %dx%d: %d far searches — %d first orders, %d re-targeted, %d departures — "+
		"and %d near searches over %d unit-ticks (%d of them ended by a far failure)",
		side, side, far, byReason[cntNoRoute], byReason[cntWrongTarget], byReason[cntDeparted],
		near, unitTicks, farFailed)

	if byReason[cntNoRoute] != len(cmds) {
		t.Errorf("%d far search(es) ran because a unit held no route, and %d units were ordered",
			byReason[cntNoRoute], len(cmds))
	}
	if byReason[cntWrongTarget] != 0 {
		t.Errorf("%d far search(es) ran because a route ended elsewhere, and nothing was re-ordered",
			byReason[cntWrongTarget])
	}
	if want := len(cmds) + byReason[cntDeparted]; far != want {
		t.Errorf("%d far searches over the run, want %d — one per unit ordered plus one per departure",
			far, want)
	}
	if near != unitTicks-farFailed || near <= 0 {
		t.Errorf("%d near searches over %d unit-ticks, of which %d ended in a far failure",
			near, unitTicks, farFailed)
	}
	// The run has to have finished, and finished by ARRIVING: counts taken over a
	// prefix of a run, or over one whose movers gave up along the way, are not
	// counts of what an order costs.
	for i, e := range w.Entities() {
		if e.HasTarget {
			t.Errorf("unit %d still holds an order after the run, so the count is not a whole run's", e.ID)
		}
		if e.X != cmds[i].X || e.Y != cmds[i].Y {
			t.Errorf("unit %d ended at (%d,%d) and was ordered to (%d,%d)", e.ID, e.X, e.Y, cmds[i].X, cmds[i].Y)
		}
	}
}
