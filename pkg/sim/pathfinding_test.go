package sim

import "testing"

// generationBudget is the budget under the shipped values, for the tests that
// state the shipped arithmetic.
func generationBudget(start, target cell, rule budgetRule, goalOpen bool) int64 {
	return ShippedPathFinding().generationBudget(start, target, rule, goalOpen)
}

func TestAWorldWithNoPathFindingRunsTheShippedValues(t *testing.T) {
	w := pursuitLine(t, 8)
	if got, want := w.pathFinding(), (PathFinding{StaticScanAhead: 5, DynamicScanAhead: 3, StaticRefreshRate: 16, DynamicRefreshRate: 32, SpeedMultiplier: 8, DynamicByStaticLookup: 3, StaticIsntNeeded: 5}); got != want {
		t.Fatalf("path finding %+v, want %+v", got, want)
	}
	custom := ShippedPathFinding()
	custom.StaticIsntNeeded = 9
	w.SetBuildRules(BuildRules{Path: &custom})
	if got := w.pathFinding(); got != custom {
		t.Fatalf("path finding %+v, want the set %+v", got, custom)
	}
}

func TestRouteEndGoalFollowsThePathFindingValues(t *testing.T) {
	route := make([]cell, 6)
	for i := range route {
		route[i] = cell{x: int32(10 + i), y: 0}
	}
	for _, c := range []struct {
		name    string
		edit    func(*PathFinding)
		from    cell
		index   int
		atEnd   bool
		comment string
	}{
		{"shipped, head near", func(*PathFinding) {}, cell{x: 8, y: 0}, lookAhead, false, "six cells exceed StaticIsntNeeded 5; head 2 off is within 3"},
		{"shipped, head far", func(*PathFinding) {}, cell{x: 4, y: 0}, 0, false, "head 6 off exceeds DynamicByStaticLookup 3"},
		{"longer static allowance", func(p *PathFinding) { p.StaticIsntNeeded = 6 }, cell{x: 4, y: 0}, 5, true, "six cells are within StaticIsntNeeded 6"},
		{"shorter head lookup", func(p *PathFinding) { p.DynamicByStaticLookup = 1 }, cell{x: 8, y: 0}, 0, false, "head 2 off exceeds DynamicByStaticLookup 1"},
		{"longer head lookup", func(p *PathFinding) { p.DynamicByStaticLookup = 9 }, cell{x: 4, y: 0}, lookAhead, false, "head 6 off is within 9"},
	} {
		p := ShippedPathFinding()
		c.edit(&p)
		if index, atEnd := routeEndGoal(p, route, c.from); index != c.index || atEnd != c.atEnd {
			t.Errorf("%s: goal (%d, %t), want (%d, %t): %s", c.name, index, atEnd, c.index, c.atEnd, c.comment)
		}
	}
}

func TestGenerationBudgetFollowsTheScanAheadValues(t *testing.T) {
	start, target := cell{x: 0, y: 0}, cell{x: 2, y: 0}
	p := ShippedPathFinding()
	if got := p.generationBudget(start, target, scaledBudget, false); got != 5 {
		t.Fatalf("shipped near budget %d, want 3+2", got)
	}
	if got := p.generationBudget(start, target, staticBudget, false); got != 7 {
		t.Fatalf("shipped far budget %d, want 5+2", got)
	}
	p.DynamicScanAhead, p.StaticScanAhead = 10, 20
	if got := p.generationBudget(start, target, scaledBudget, false); got != 12 {
		t.Errorf("near budget %d, want 10+2", got)
	}
	if got := p.generationBudget(start, target, staticBudget, false); got != 22 {
		t.Errorf("far budget %d, want 20+2", got)
	}
	if got := p.generationBudget(start, target, largeBudget, false); got != 22 {
		t.Errorf("large budget %d, want 20+2", got)
	}
}

// pursuitCounts runs the pursuit of the first-pursuit test over w and returns
// the static count each full search or rebuild left.
func pursuitCounts(t *testing.T, w *World) []uint16 {
	t.Helper()
	var counts []uint16
	cmds := []Command{cbOrder(1, 2)}
	last := PursuitSearch{}
	for k := 0; k < 12; k++ {
		Step(w, cmds)
		cmds = nil
		p := cbAt(t, w, 1).Pursuit
		if p.Held && (!last.Held || p.Passes < last.Passes) {
			counts = append(counts, p.Count)
		}
		last = p
	}
	return counts
}

func TestAPursuitReadsStaticIsntNeededFromThePathFinding(t *testing.T) {
	shipped := pursuitCounts(t, pursuitLine(t, 8))
	if len(shipped) != 3 || shipped[0] != 8 || shipped[1] != 5 || shipped[2] != 1 {
		t.Fatalf("shipped counts %v, want 8, 5, 1", shipped)
	}
	w := pursuitLine(t, 8)
	p := ShippedPathFinding()
	p.StaticIsntNeeded = 20
	w.SetBuildRules(BuildRules{Path: &p})
	got := pursuitCounts(t, w)
	if len(got) < 2 || got[0] != 8 || got[1] != 1 {
		t.Fatalf("counts with StaticIsntNeeded 20: %v, want 8 then the rebuild's 1", got)
	}
}

// walkOver walks a unit along an open strip under p (nil for none set) and
// returns the number of far searches it starts: the drops of its near search
// count.
func walkOver(t *testing.T, p *PathFinding) (far int) {
	t.Helper()
	w, err := NewWorld(1, Bounds{Width: 80, Height: 21}, ModeCanonical, nil, []Entity{cbEnt(1, 0, 10)})
	if err != nil {
		t.Fatal(err)
	}
	if p != nil {
		w.SetBuildRules(BuildRules{Path: p})
	}
	cmds := []Command{MoveTo(1, CellPoint{X: 70, Y: 10})}
	last := WalkRefresh{}
	for k := 0; k < 900; k++ {
		Step(w, cmds)
		cmds = nil
		walk := cbAt(t, w, 1).Walk
		if walk.Searches < last.Searches {
			far++
		}
		last = walk
	}
	return far
}

func TestAWalkReadsStaticRefreshRateFromThePathFinding(t *testing.T) {
	shipped := walkOver(t, nil)
	p := ShippedPathFinding()
	p.StaticRefreshRate = 2
	if quick := walkOver(t, &p); quick <= shipped {
		t.Errorf("far searches %d with StaticRefreshRate 2, %d shipped", quick, shipped)
	}
}

// listedWalkSearches steps a unit whose listed near step is ten passes old
// and returns its near search count after the pass.
func listedWalkSearches(t *testing.T, rate int32) uint8 {
	t.Helper()
	w, err := NewWorld(1, Bounds{Width: 80, Height: 21}, ModeCanonical, nil, []Entity{cbEnt(1, 0, 10)})
	if err != nil {
		t.Fatal(err)
	}
	p := ShippedPathFinding()
	p.DynamicRefreshRate = rate
	w.SetBuildRules(BuildRules{Path: &p})
	Step(w, []Command{MoveTo(1, CellPoint{X: 70, Y: 10})})
	e := cbAt(t, w, 1)
	w.entities[0].Walk = WalkRefresh{Age: 10, Searches: 1, Listed: true, NextX: e.X + 1, NextY: 10}
	Step(w, nil)
	return cbAt(t, w, 1).Walk.Searches
}

func TestAWalkReadsDynamicRefreshRateFromThePathFinding(t *testing.T) {
	if got := listedWalkSearches(t, 32); got != 1 {
		t.Errorf("searches %d at DynamicRefreshRate 32, want the listed step kept (1)", got)
	}
	if got := listedWalkSearches(t, 5); got != 2 {
		t.Errorf("searches %d at DynamicRefreshRate 5, want a new near search (2)", got)
	}
}

func TestAStaticIsntNeededBelowTheLookAheadKeepsTheRouteIndexInsideTheRoute(t *testing.T) {
	for _, v := range []int32{0, 1, 2, -1} {
		p := ShippedPathFinding()
		p.StaticIsntNeeded = v
		walkOver(t, &p)
		w := pursuitLine(t, 8)
		w.SetBuildRules(BuildRules{Path: &p})
		pursuitCounts(t, w)
		for n := 0; n <= 4; n++ {
			route := make([]cell, n)
			at, end := routeEndGoal(p, route, cell{})
			if n == 0 && (!end || at != -1) {
				t.Errorf("StaticIsntNeeded %d, empty route: %d, %v", v, at, end)
			}
			if n > 0 && !end && (at < 0 || at >= n) {
				t.Errorf("StaticIsntNeeded %d, %d cells: index %d outside the route", v, n, at)
			}
		}
	}
}
