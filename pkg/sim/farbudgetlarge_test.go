package sim

import "testing"

// The far search of a mover larger than one cell spends StaticScanAhead + D
// generations whoever owns it, the n x n arm of the budget: it has no D>>2 term
// and no flat override (MOVE-TERM-003, MOVE-PARAM-006, MOVE-SPEED-011). A one-cell
// mover of another owner spends max(5, D>>2) + D and a one-cell mover of the
// participant a flat thousand. The arms agree while D is under 24, so the fixture
// puts the goal 40 cells from the mover, where they spend 45 and 50 generations,
// and puts a wall between them that the wave needs 46 to 49 generations to round.
//
//	. . . . . . . # . . . . . . .    A column of blocked cells at x = 30 in a
//	. . . . . . . . . . . . . . .    64 x 64 world, open only on rows 7 to
//	. . . . . . . # . . . . . . .    gapEnd. A mover at (10,32) and a goal at
//	. . @ . . . . # . . . . G . .    (50,32) stand 40 cells apart with the wall
//	. . . . . . . # . . . . . . .    between them, and the only way through is
//	                                  the gap, 22 to 25 rows north.
//
// The wave of the arm the claims give a large mover stops one to four
// generations short of the goal and settles on a cell a few cells from it.

var bigWallBounds = Bounds{Width: 64, Height: 64}

const (
	bigWallColumn                    int32 = 30
	bigWallGapStart                  int32 = 7
	bigWallStartX, bigWallStartY     int32 = 10, 32
	bigWallGoalX, bigWallGoalY       int32 = 50, 32
	bigWallHorizon                         = 140
	bigWallSettleReach               int64 = 14
	bigWallLargeArm, bigWallSmallArm int64 = 45, 50
)

// bigWallGrid is the wall fixture's static plane: the column blocked except on
// rows bigWallGapStart to gapEnd.
func bigWallGrid(gapEnd int32) []byte {
	g := make([]byte, bigWallBounds.Width*bigWallBounds.Height)
	for y := int32(0); y < bigWallBounds.Height; y++ {
		if y < bigWallGapStart || y > gapEnd {
			g[y*bigWallBounds.Width+bigWallColumn] = blockGround
		}
	}
	return g
}

// bigWallSteps is the fewest steps a mover whose footprint is side x side takes
// from the start to the goal over grid: eight neighbours, a step needing every
// cell of the footprint at the new anchor open, no corner rule. It is the number
// of generations a wave of the far search runs before the goal carries a label,
// found here without the search so that the fixture states its own window.
func bigWallSteps(grid []byte, side int32) int64 {
	open := func(x, y int32) bool {
		for dy := int32(0); dy < side; dy++ {
			for dx := int32(0); dx < side; dx++ {
				cx, cy := x+dx, y+dy
				if cx < 0 || cy < 0 || cx >= bigWallBounds.Width || cy >= bigWallBounds.Height ||
					grid[cy*bigWallBounds.Width+cx] != 0 {
					return false
				}
			}
		}
		return true
	}
	dist := map[cell]int64{{x: bigWallStartX, y: bigWallStartY}: 0}
	frontier := []cell{{x: bigWallStartX, y: bigWallStartY}}
	for len(frontier) > 0 {
		var next []cell
		for _, c := range frontier {
			for dx := int32(-1); dx <= 1; dx++ {
				for dy := int32(-1); dy <= 1; dy++ {
					n := cell{x: c.x + dx, y: c.y + dy}
					if _, seen := dist[n]; seen || !open(n.x, n.y) {
						continue
					}
					dist[n] = dist[c] + 1
					next = append(next, n)
				}
			}
		}
		frontier = next
	}
	if d, ok := dist[cell{x: bigWallGoalX, y: bigWallGoalY}]; ok {
		return d
	}
	return -1
}

// bigWallWalk gives one mover a move across the wall and returns the tick it
// arrived at the goal on (0 when it never did), the last cell of the route it held
// after the tick of the order, and where it ended.
func bigWallWalk(t *testing.T, grid []byte, size uint8, owner uint32) (arrived uint64, routeEnd [2]int32, end Entity) {
	t.Helper()
	w := mustWorldGrid(t, 1, bigWallBounds, ModeCanonical, grid,
		[]Entity{{ID: 1, X: bigWallStartX, Y: bigWallStartY, Owner: owner, TokenSize: size}})
	Step(w, []Command{MoveTo(1, CellPoint{X: bigWallGoalX, Y: bigWallGoalY})})
	if route := w.Route(1); len(route) > 0 {
		routeEnd = route[len(route)-1]
	}
	for i := 0; i < bigWallHorizon; i++ {
		e := w.Entities()[0]
		if e.X == bigWallGoalX && e.Y == bigWallGoalY && arrived == 0 {
			arrived = w.tick
		}
		Step(w, nil)
	}
	return arrived, routeEnd, w.Entities()[0]
}

// TestALargeMoversFarSearchSpendsTheGenerationsOfTheNxNArm drives the same order
// over the same wall for movers of another owner and of the participant. A mover
// larger than one cell has 45 generations whoever owns it, one short of the way
// round, so its search settles on a cell a few cells short of the goal and its
// order ends there. A one-cell mover has 50 generations when another owner holds
// it and a flat thousand when the participant does, and walks to the goal.
func TestALargeMoversFarSearchSpendsTheGenerationsOfTheNxNArm(t *testing.T) {
	for _, tc := range []struct {
		name   string
		size   uint8
		owner  uint32
		gapEnd int32
		walks  bool
	}{
		{"a two-cell creature of another owner", 2, 2, 10, false},
		{"a three-cell creature of another owner", 3, 2, 11, false},
		{"a two-cell unit of the participant", 2, SelfSlot, 10, false},
		{"a three-cell unit of the participant", 3, SelfSlot, 11, false},
		{"a two-cell unit that names no roster slot", 2, 0, 10, false},
		{"a one-cell creature of another owner", 1, 2, 10, true},
		{"a one-cell unit of the participant", 1, SelfSlot, 10, true},
		{"a one-cell unit that names no roster slot", 1, 0, 10, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grid := bigWallGrid(tc.gapEnd)
			if tc.size > 1 {
				if steps := bigWallSteps(grid, int32(tc.size)); steps <= bigWallLargeArm || steps > bigWallSmallArm {
					t.Fatalf("the wave needs %d generations to label the goal, outside (%d, %d]: the fixture no longer separates the two arms",
						steps, bigWallLargeArm, bigWallSmallArm)
				}
			}
			arrived, routeEnd, end := bigWallWalk(t, grid, tc.size, tc.owner)
			if tc.walks {
				if arrived == 0 || routeEnd != [2]int32{bigWallGoalX, bigWallGoalY} {
					t.Fatalf("the mover ended at (%d,%d), its route ended at %v and it never reached the goal (%d,%d)",
						end.X, end.Y, routeEnd, bigWallGoalX, bigWallGoalY)
				}
				return
			}
			if arrived != 0 || routeEnd == [2]int32{bigWallGoalX, bigWallGoalY} {
				t.Fatalf("the mover reached the goal at tick %d and its route ended at %v: its far search labelled a goal that %d generations cannot reach",
					arrived, routeEnd, bigWallLargeArm)
			}
			gap := (cell{x: end.X, y: end.Y}).chebyshevTo(cell{x: bigWallGoalX, y: bigWallGoalY})
			if end.HasTarget || gap < 1 || gap > bigWallSettleReach {
				t.Errorf("the mover ended at (%d,%d) holding a target %t, %d cells from the goal, want it settled on a cell within %d of the goal with no order left",
					end.X, end.Y, end.HasTarget, gap, bigWallSettleReach)
			}
		})
	}
}

// TestTheFarBudgetOfALargeMover pins the n x n arm's number: StaticScanAhead + D
// whatever the goal is, D being the Chebyshev distance from the mover's cell to
// it. The one-cell form of another owner, max(5, D>>2) + D, is the same number
// while D is under 24 and exceeds it by (D>>2) - 5 from there (MOVE-TERM-003,
// MOVE-PARAM-006).
func TestTheFarBudgetOfALargeMover(t *testing.T) {
	for _, goal := range []cell{{0, 0}, {3, 0}, {23, 0}, {24, 0}, {12, 30}, {39, 0}, {40, 0}, {-100, 7}, {900, 0}} {
		start := cell{0, 0}
		d := start.chebyshevTo(goal)
		want := 5 + d
		for _, open := range []bool{true, false} {
			if got := generationBudget(start, goal, largeBudget, open); got != want {
				t.Errorf("the n x n arm's budget to (%d,%d), goal open %t, is %d, want %d", goal.x, goal.y, open, got, want)
			}
		}
		parted := int64(0)
		if d >= 24 {
			parted = d>>2 - 5
		}
		if got := generationBudget(start, goal, staticBudget, true); got != want+parted {
			t.Errorf("a one-cell creature's budget to (%d,%d) is %d, want the n x n arm's %d and %d more", goal.x, goal.y, got, want, parted)
		}
	}
}

// TestTheFarBudgetIsChosenByTheOwnerAndTheFootprint pins the one chooser every far
// search of an entity reads. A footprint of two cells or more takes the n x n arm
// whoever owns the mover: the participant's slot, an entity naming no slot and any
// other roster slot alike. Below it the participant's slot and an entity naming no
// slot keep the flat rule, any other roster slot takes the one-cell form, and a
// footprint of zero is one cell.
func TestTheFarBudgetIsChosenByTheOwnerAndTheFootprint(t *testing.T) {
	sizes := []uint8{0, 1, 2, 3}
	owners := []uint32{SelfSlot, 0, 2, 7}
	var ents []Entity
	for k, owner := range owners {
		for j, size := range sizes {
			ents = append(ents, Entity{ID: EntityID(k*len(sizes) + j + 1), X: int32(k), Y: int32(j), Owner: owner, TokenSize: size})
		}
	}
	w := mustWorld(t, 1, engBounds, ents)
	for i, e := range w.entities {
		want := staticBudget
		switch {
		case e.TokenSize >= 2:
			want = largeBudget
		case e.Owner == SelfSlot || e.Owner == 0:
			want = flatBudget
		}
		if got := w.farBudgetFor(i); got != want {
			t.Errorf("entity %d (owner %d, footprint %d): far budget rule %d, want %d", e.ID, e.Owner, e.TokenSize, got, want)
		}
	}
}

// TestTheNearSearchNeedsNoRuleForItsNxNArm holds the reason the near search has no
// large-mover rule of its own. Its goal is inside a window of eight cells, so D is
// at most 8, and there the scaled rule is DynamicScanAhead + D, the number the
// claims give a large mover's near search (MOVE-TERM-003, MOVE-PARAM-006).
func TestTheNearSearchNeedsNoRuleForItsNxNArm(t *testing.T) {
	for d := int64(0); d <= int64(dynamicWindow); d++ {
		if got := generationBudget(cell{0, 0}, cell{int32(d), 0}, scaledBudget, true); got != 3+d {
			t.Errorf("the near search's budget at D %d is %d, want %d", d, got, 3+d)
		}
	}
}
