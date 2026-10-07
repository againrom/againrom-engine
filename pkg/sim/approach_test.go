package sim

// How far a unit gets toward a cell it cannot reach, and by what rule it picks
// the cell it stops on.
//
// Every world here is built out of bounds, a grid and positions; nothing reads a
// game install.

import "testing"

// apSide is the map these fixtures stand on, and apGoal the cell they order
// their mover onto — a cell that blocks ground, so no wave can ever label it.
const apSide int32 = 21

var apGoal = cell{10, 10}

// apGrid is the discriminating fixture, and every blocked cell in it is there to
// rule out one reading of the rule.
//
// The goal itself blocks, so the search must settle or answer nothing. Six of
// the goal's eight neighbours block too, leaving exactly TWO cells in the first
// ring around it: (9,10) on the mover's side and (11,10) on the far side, which
// is reachable only by climbing over the blocked cluster and coming back down.
// So the two candidates that decide the story sit in ONE ring, at one distance
// from the cell that was asked for, and nothing about where the click landed can
// tell them apart.
func apGrid() []byte {
	g := make([]byte, apSide*apSide)
	for _, c := range []cell{
		apGoal,
		{9, 9}, {10, 9}, {11, 9},
		{9, 11}, {10, 11}, {11, 11},
	} {
		g[c.y*apSide+c.x] = blockGround
	}
	return g
}

func apWorld(t *testing.T) *World {
	t.Helper()
	return mustWorldGrid(t, 1, Bounds{Width: apSide, Height: apSide}, ModeCanonical,
		apGrid(), []Entity{{ID: 1, X: 5, Y: 10}})
}

// TestASettlingSearchTakesTheCheapestCellOfTheNearestRing is AC-2, and it is the
// criterion that tells the decoded rule apart from the two readings a reader
// would otherwise reach for.
//
// The rule has two halves measured from two different places: the RINGS expand
// outward from the cell that was ordered, and INSIDE a ring the cell taken is
// the one with the smallest label — the cheapest to reach from the mover. This
// fixture makes each half decide against a rival that the other half cannot see.
//
//   - Against "the cheapest labelled cell there is": (8,10) is labelled, it is
//     strictly cheaper than the cell taken, and it loses because it sits one ring
//     further out. Every cell between the mover and the goal is cheaper still.
//   - Against "the first labelled cell the scan meets": (11,10) is labelled, it
//     is in the SAME ring, and the ring's own walk reaches it BEFORE the cell
//     taken — the four sides are probed (y+r), (y-r), (x+r), (x-r) for each i in
//     turn, so the far side of the ring is offered ahead of the near side. It
//     loses on cost alone.
//
// The rival the fixture deliberately cannot discriminate is "the cell nearest
// the one clicked": inside one ring every candidate is the same distance from
// the click, so that reading is silent here and is answered by the ring order
// instead, which the walled and off-map fixtures elsewhere witness.
func TestASettlingSearchTakesTheCheapestCellOfTheNearestRing(t *testing.T) {
	w := apWorld(t)
	s := newRouteScratch(w)

	route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, settleOrdered, apGoal.x, apGoal.y)
	if !ok || len(route) == 0 {
		t.Fatalf("the search settled for nothing: ok=%v route=%s", ok, fmtRoute(route))
	}
	got := route[len(route)-1]
	if want := (cell{9, 10}); got != want {
		t.Fatalf("the search settled on %v, want %v", got, want)
	}

	// The plane still holds this search's labels, which is what the two rivals
	// are read off. A candidate with no label was never a candidate at all.
	start := cell{5, 10}
	labelOf := func(c cell) uint64 {
		t.Helper()
		l, ok := w.label(s, start, c.x, c.y)
		if !ok {
			t.Fatalf("%v carries no label, so this fixture cannot discriminate anything", c)
		}
		return l
	}

	taken := labelOf(got)
	if cheaper := labelOf(cell{8, 10}); cheaper >= taken {
		t.Errorf("(8,10) costs %d and the cell taken costs %d — the fixture needs the outer ring to hold "+
			"the cheaper cell, or it does not rule out a plain minimum", cheaper, taken)
	}
	if earlier := labelOf(cell{11, 10}); earlier <= taken {
		t.Errorf("(11,10) costs %d and the cell taken costs %d — the fixture needs the ring's earlier "+
			"cell to be the dearer one, or it does not rule out taking the first one scanned",
			earlier, taken)
	}
}

// TestSettlingIsTheCanonicalWavesAloneAndTheOptimisedSearchStillRefuses is AC-4.
//
// The two modes answer this differently and the reason is structural rather than
// a choice about behaviour: the wave floods outward from the MOVER, so a failed
// one leaves a plane saying what the mover can reach and at what cost — which is
// exactly what a substitute is chosen off. The optimised search floods outward
// from the TARGET, so when the target is refused it floods nothing at all, and
// when it is merely unreachable the cells it did cost are the ones that can
// reach the target rather than the ones the mover can. There is nothing there to
// settle for.
func TestSettlingIsTheCanonicalWavesAloneAndTheOptimisedSearchStillRefuses(t *testing.T) {
	for _, mode := range []Mode{ModeCanonical, ModeOptimised} {
		w := mustWorldGrid(t, 1, Bounds{Width: apSide, Height: apSide}, mode,
			apGrid(), []Entity{{ID: 1, X: 5, Y: 10}})
		Step(w, []Command{{Entity: 1, X: apGoal.x, Y: apGoal.y}})

		e := w.Entities()[0]
		moved := e.X != 5 || e.Y != 10
		if mode == ModeCanonical && !moved {
			t.Errorf("mode %d: the unit is still at (5,10); a settling search walks it toward the goal", mode)
		}
		if mode == ModeOptimised {
			if moved {
				t.Errorf("mode %d: the unit walked to (%d,%d)", mode, e.X, e.Y)
			}
			if e.HasTarget {
				t.Errorf("mode %d: the unit still holds an order", mode)
			}
		}
	}
}

// TestASettledOrderPointsAtTheCellItSettledFor is AC-5: the sweep runs on the
// tick the order is given and on no later one, because the order itself is
// re-pointed at the cell the search settled for.
//
// It is asserted as a property of the STORED ROUTE rather than as a count of
// searches, and that is the stronger reading of the same fact: the three
// staleness tests replace a route exactly when its last cell is not the current
// target, so a route whose last cell IS the target is a route that serves, and a
// route that serves is a tick with no far search in it.
func TestASettledOrderPointsAtTheCellItSettledFor(t *testing.T) {
	w := apWorld(t)
	Step(w, []Command{{Entity: 1, X: apGoal.x, Y: apGoal.y}})

	e := w.Entities()[0]
	if !e.HasTarget {
		t.Fatalf("the order is over already; this fixture is meant to walk")
	}
	if e.TargetX == apGoal.x && e.TargetY == apGoal.y {
		t.Fatalf("the order still points at the sealed cell (%d,%d), so its route can never serve",
			apGoal.x, apGoal.y)
	}
	route := w.routes[0]
	if len(route) == 0 {
		t.Fatalf("the unit holds a target and no route")
	}
	if last := route[len(route)-1]; last.x != e.TargetX || last.y != e.TargetY {
		t.Fatalf("the route ends at %v and the order points at (%d,%d) — the two disagree, so the next "+
			"tick sweeps the whole map again", last, e.TargetX, e.TargetY)
	}

	// And it walks the whole way there without ever sweeping again, ending
	// parked on the cell it settled for with nothing left over.
	rest := cell{e.TargetX, e.TargetY}
	for tick := 2; tick <= stallLimit; tick++ {
		Step(w, nil)
	}
	if got := w.Entities()[0]; got.X != rest.x || got.Y != rest.y || got.HasTarget || got.Stall != 0 {
		t.Fatalf("the unit is %+v, want it parked on %v with no order and no stall", got, rest)
	}
}

// TestTwoWorldsSettleAlike is AC-6. The settle is chosen off a scan order and a
// strict comparison, so two worlds alike in everything take the same cell at the
// same tick — and, being over the same fields every other rule writes, they stay
// alike in their digests too.
func TestTwoWorldsSettleAlike(t *testing.T) {
	a, b := apWorld(t), apWorld(t)
	for tick := 1; tick <= stallLimit; tick++ {
		var cmds []Command
		if tick == 1 {
			cmds = []Command{{Entity: 1, X: apGoal.x, Y: apGoal.y}}
		}
		Step(a, cmds)
		Step(b, cmds)
		if a.Hash() != b.Hash() {
			t.Fatalf("tick %d: the two worlds' digests differ (%#x, %#x)", tick, a.Hash(), b.Hash())
		}
	}
}

// TestTheFlatBudgetIsAnOverrideWithAGate is AC-7. The flat thousand is not the
// far search's whole rule: it applies only where the goal is open, and a far
// search that must settle falls through to the computed form the far arm carries
// — the one whose slack is five, not the near arm's three.
//
// The four arms are asked directly, because what is being pinned is an
// arithmetic rule and the world it would otherwise be read through can only show
// its consequences.
func TestTheFlatBudgetIsAnOverrideWithAGate(t *testing.T) {
	start, target := cell{0, 0}, cell{40, 0}
	for _, tc := range []struct {
		what     string
		rule     budgetRule
		goalOpen bool
		want     int64
	}{
		{"the far search over an open goal takes the override", flatBudget, true, flatGenerations},
		{"the far search over a closed goal falls through to max(5, D>>2) + D", flatBudget, false, 50},
		{"the near search never takes the override", scaledBudget, true, 50},
		{"and its own slack is three, not five", scaledBudget, false, 50},
	} {
		if got := generationBudget(start, target, tc.rule, tc.goalOpen); got != tc.want {
			t.Errorf("%s: %d, want %d", tc.what, got, tc.want)
		}
	}
	// D>>2 is 10 above, so the two slacks are hidden by the quarter-distance
	// term there. Close in, where the slack is what binds, they part.
	near := cell{4, 0}
	if got := generationBudget(start, near, flatBudget, false); got != 9 {
		t.Errorf("the far arm's slack close in gives %d, want 5 + 4", got)
	}
	if got := generationBudget(start, near, scaledBudget, false); got != 7 {
		t.Errorf("the near arm's slack close in gives %d, want 3 + 4", got)
	}
}

// nrSide is the map the near-rule fixture stands on, and nrWall the column of
// BODIES that closes the mover's direct line to its own sub-goal.
//
// Bodies and not terrain, and that is the whole construction: the far search
// reads terrain alone, so it routes straight through this wall and hands the
// near search a sub-goal four cells away on the far side of it. The near search
// reads occupancy, so it must go round — and the way round is ten steps, inside
// a window that permits eight in every direction.
const nrSide int32 = 24

var (
	nrStart = cell{10, 10}
	nrGoal  = cell{14, 10}
)

func nrWorld(t *testing.T) *World {
	t.Helper()
	ents := []Entity{{ID: 1, X: nrStart.x, Y: nrStart.y}}
	for y := int32(6); y <= 14; y++ {
		ents = append(ents, Entity{ID: EntityID(2 + y), X: 12, Y: y})
	}
	return mustWorldGrid(t, 1, Bounds{Width: nrSide, Height: nrSide}, ModeCanonical, nil, ents)
}

// TestTheTickHandsTheNearSearchTheScaledRule is AC-8, and it exists because
// nothing else in this tree asserts it.
//
// The two call sites in Step are the only choosers of a budget rule, and every
// criterion that discriminates the two rules drives canonicalRoute directly —
// so the near call site could be moved to the flat budget and the whole tree
// would stay green, with a mover silently allowed a thousand generations where
// the contract gives it seven. This drives it THROUGH Step, which is the only
// place the choice is made.
//
// The fixture is built so the two rules disagree about one tick. The sub-goal is
// four cells off, so the scaled rule allows 3 + 4 = 7 generations; the way round
// the bodies is ten steps, and it lies well inside the window, so the window
// cannot be what refuses it. Under the flat rule the near search reaches the
// sub-goal along that detour; under the scaled rule it does not reach it at all
// and settles for a substitute, which is a different first step.
//
// The discriminator used to be that the scaled rule made the mover hold its cell
// entirely. It no longer does — a near search that cannot label its waypoint
// settles rather than refusing — so what separates the two rules here is WHICH
// cell is stepped to, and both are computed from the same world rather than
// written down.
func TestTheTickHandsTheNearSearchTheScaledRule(t *testing.T) {
	w := nrWorld(t)
	s := newRouteScratch(w)

	// The far search reads terrain alone, so it routes straight through the
	// bodies and aims the near search at the sub-goal on their far side. Taken
	// before the tick, so no step has consumed any of it.
	route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, settleOrdered, nrGoal.x, nrGoal.y)
	if !ok || len(route) != 4 {
		t.Fatalf("the far route is %s, want the four cells of the straight line", fmtRoute(route))
	}
	sub := route[subGoalIndex(len(route))]
	if sub != nrGoal {
		t.Fatalf("the sub-goal is %v, want %v", sub, nrGoal)
	}
	if d := nrStart.chebyshevTo(nrGoal); d != 4 {
		t.Fatalf("the sub-goal is %d cells off, and this fixture needs four", d)
	}
	if b := generationBudget(nrStart, nrGoal, scaledBudget, true); b != 7 {
		t.Fatalf("the scaled rule allows %d generations here, and this fixture needs seven", b)
	}

	// The way round is inside the window, so what refuses it is the budget.
	win := window{centre: nrStart, half: dynamicWindow}
	for _, c := range []cell{{11, 6}, {12, 5}, {13, 6}, {14, 7}} {
		if !win.holds(c.x, c.y) {
			t.Fatalf("%v is outside the near window, so this fixture cannot tell a budget from a bound", c)
		}
	}

	// The two rules' own answers, from this world, under the rule the near call
	// site carries. Asserting they differ is what makes the tick's step evidence.
	scaled, sok := w.canonicalRoute(s, 0, unitRelation, dynamicWindow, scaledBudget, settleStep, sub.x, sub.y)
	flat, fok := w.canonicalRoute(s, 0, unitRelation, dynamicWindow, flatBudget, settleStep, sub.x, sub.y)
	if !sok || len(scaled) == 0 || !fok || len(flat) == 0 {
		t.Fatalf("one of the two rules found nothing: scaled ok=%v %s, flat ok=%v %s",
			sok, fmtRoute(scaled), fok, fmtRoute(flat))
	}
	if flat[len(flat)-1] != nrGoal {
		t.Fatalf("the flat rule ended at %v, want the sub-goal — this fixture needs the detour to be "+
			"inside the flat budget and outside the scaled one", flat[len(flat)-1])
	}
	if scaled[len(scaled)-1] == nrGoal {
		t.Fatalf("the scaled rule reached the sub-goal, so this fixture holds no disagreement")
	}
	if scaled[0] == flat[0] {
		t.Fatalf("both rules step to %v first, so the tick's step cannot tell them apart", scaled[0])
	}

	Step(w, []Command{{Entity: 1, X: nrGoal.x, Y: nrGoal.y}})
	got := w.Entities()[0]
	if got.X != scaled[0].x || got.Y != scaled[0].y {
		t.Errorf("the mover stepped to (%d,%d), want %v — the scaled rule's step, not the flat rule's %v",
			got.X, got.Y, scaled[0], flat[0])
	}
	if !got.HasTarget {
		t.Errorf("the mover lost its order; a settled near search is an advance and keeps everything")
	}
}

// apTieGrid leaves exactly TWO cells open in the goal's first ring, placed so
// the mover at (5,5) reaches both at the SAME cost — four diagonal steps and one
// straight, either side of the diagonal between them — and so that the ring's
// walk visits each of them exactly ONCE.
//
// That last part is what the fixture turns on, and it is easy to get wrong: the
// ring's four sides overlap at its corners, so a candidate on the `x-r` side is
// probed again at the far end of the walk and a tie against it is decided the
// same way whichever accept is used. `(10,9)` and `(9,10)` sit mid-side, are
// each offered once, and are offered in that order.
func apTieGrid() []byte {
	g := make([]byte, apSide*apSide)
	for _, c := range []cell{
		apGoal,
		{9, 9}, {11, 9}, {11, 10}, {9, 11}, {10, 11}, {11, 11},
	} {
		g[c.y*apSide+c.x] = blockGround
	}
	return g
}

// TestASettlingSearchTakesTheEarlierOfTwoEqualCells is AC-2's tie-break half,
// and it needs a fixture of its own because a tie is what the rest of them do
// not contain.
//
// Two candidates, one ring, equal labels: the ring's own walk decides, and the
// accept being STRICTLY less is what makes "the earlier one" mean anything. The
// walk takes the sides `y+r`, `y-r`, `x+r`, `x-r` for each i from -r, so the
// cell above the goal is offered before the cell to its left, and it is the one
// taken. Relax the accept to less-or-equal and the later of the two wins
// instead — a different cell, on a route a digest covers.
func TestASettlingSearchTakesTheEarlierOfTwoEqualCells(t *testing.T) {
	w := mustWorldGrid(t, 1, Bounds{Width: apSide, Height: apSide}, ModeCanonical,
		apTieGrid(), []Entity{{ID: 1, X: 5, Y: 5}})
	s := newRouteScratch(w)

	route, ok := w.canonicalRoute(s, 0, terrainRelation, noWindow, flatBudget, settleOrdered, apGoal.x, apGoal.y)
	if !ok || len(route) == 0 {
		t.Fatalf("the search settled for nothing: ok=%v route=%s", ok, fmtRoute(route))
	}

	// The tie is the fixture's whole point, so it is asserted rather than
	// assumed: without it this test would pin a cell and measure nothing.
	start := cell{5, 5}
	early, okEarly := w.label(s, start, 10, 9)
	late, okLate := w.label(s, start, 9, 10)
	if !okEarly || !okLate {
		t.Fatalf("one of the two candidates carries no label (%v, %v)", okEarly, okLate)
	}
	if early != late {
		t.Fatalf("the two candidates cost %d and %d, so this fixture holds no tie", early, late)
	}

	if got, want := route[len(route)-1], (cell{10, 9}); got != want {
		t.Fatalf("the search settled on %v, want %v — the earlier of the two in the ring's own walk",
			got, want)
	}
}

// TestRouteIsTheStoredRouteCopied is AC-9. The query hands back what the tick
// already computed, converted and copied — never a second search, and never a
// slice that still reaches into a world's hashed state.
func TestRouteIsTheStoredRouteCopied(t *testing.T) {
	w := apWorld(t)

	if got := w.Route(1); got != nil {
		t.Errorf("a unit that has taken no order answers %v, want nothing", got)
	}
	if got := w.Route(404); got != nil {
		t.Errorf("an entity this world does not hold answers %v, want nothing", got)
	}

	Step(w, []Command{{Entity: 1, X: apGoal.x, Y: apGoal.y}})

	got := w.Route(1)
	stored := w.routes[0]
	if len(got) != len(stored) {
		t.Fatalf("the query answers %d cells and the world holds %d", len(got), len(stored))
	}
	for i, c := range stored {
		if got[i] != [2]int32{c.x, c.y} {
			t.Fatalf("cell %d is %v, want %v", i, got[i], c)
		}
	}

	// The copy is what keeps a query out of the digest. A write through the
	// answer must reach nothing.
	before := w.Hash()
	got[0] = [2]int32{-1, -1}
	if w.Hash() != before {
		t.Errorf("writing through the answer changed the world's digest")
	}
	if again := w.Route(1); again[0] == ([2]int32{-1, -1}) {
		t.Errorf("the query hands back the same backing array twice")
	}
}
