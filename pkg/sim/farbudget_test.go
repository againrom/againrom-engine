package sim

import "testing"

// The far search's budget of a one-cell unit belongs to its owner: a unit a
// human participant owns takes the flat thousand and every other owner's unit
// the computed max(5, D>>2) + D. Both fixtures below are one wall across a small
// world, so the difference between the two owners is the whole of what differs.
//
//	. . . . . # . . .        A column of blocked cells at x = 18 from y = 2 to
//	. . . . . # . . .        y = 45. A mover at (20,20) and a goal at (17,20)
//	. . G . . # @ . .        stand three cells apart with the wall between them,
//	. . . . . # . . .        and the only way round is past the ends of the
//	. . . . . # . . .        column: more than twenty hops each way.
//
// D is 3 and max(5, 3>>2) + 3 is 8, so the wave of a non-participant's search
// stops long before it can label the goal, and the search settles on a labelled
// cell beside the wall.

// farWallGrid is the wall fixture's static plane.
func farWallGrid() []byte {
	g := make([]byte, engBounds.Width*engBounds.Height)
	for y := int32(2); y <= 45; y++ {
		g[y*engBounds.Width+18] = blockGround
	}
	return g
}

const farWallStartX, farWallStartY int32 = 20, 20
const farWallGoalX, farWallGoalY int32 = 17, 20

// farWallWalk gives one mover a move across the wall and returns the tick it
// arrived at the goal on (0 when it never did), the west-most column it stood
// in, and where it ended.
func farWallWalk(t *testing.T, owner uint32) (arrived uint64, west int32, end Entity) {
	t.Helper()
	w := mustWorldGrid(t, 1, engBounds, ModeCanonical, farWallGrid(),
		[]Entity{{ID: 1, X: farWallStartX, Y: farWallStartY, Owner: owner}})
	west = farWallStartX
	Step(w, []Command{MoveTo(1, CellPoint{X: farWallGoalX, Y: farWallGoalY})})
	for i := 0; i < 160; i++ {
		e := w.Entities()[0]
		west = min(west, e.X)
		if e.X == farWallGoalX && e.Y == farWallGoalY && arrived == 0 {
			arrived = w.tick
		}
		Step(w, nil)
	}
	end = w.Entities()[0]
	west = min(west, end.X)
	return arrived, west, end
}

// TestAMoversFarSearchSpendsTheBudgetOfItsOwner drives the same order over the
// same terrain from the same cell for three movers. A participant's unit and a
// unit that names no roster slot walk the detour to the goal; another owner's
// unit settles beside the wall and never crosses it (MOVE-TERM-003, MOVE-ALT-018,
// MOVE-ALT-019, MOVE-ALT-021).
func TestAMoversFarSearchSpendsTheBudgetOfItsOwner(t *testing.T) {
	for _, tc := range []struct {
		name   string
		owner  uint32
		detour bool
	}{
		{"a human participant's unit", SelfSlot, true},
		{"a unit that names no roster slot", 0, true},
		{"another owner's unit", 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			arrived, west, end := farWallWalk(t, tc.owner)
			if tc.detour {
				if arrived == 0 {
					t.Fatalf("the mover ended at (%d,%d) and never reached the goal (%d,%d)", end.X, end.Y, farWallGoalX, farWallGoalY)
				}
				if arrived < 30 {
					t.Fatalf("the mover arrived at tick %d, so the way round is no longer than the computed budget and the fixture discriminates nothing", arrived)
				}
				return
			}
			if arrived != 0 || west < 18 {
				t.Fatalf("the mover reached (%d,%d) at tick %d and stood as far west as column %d: its far search walked the detour a budget of 8 generations cannot label",
					farWallGoalX, farWallGoalY, arrived, west)
			}
			if end.HasTarget || end.X < 19 || end.X > 20 || end.Y < 19 || end.Y > 21 {
				t.Errorf("the mover ended at (%d,%d) holding a target %t, want it settled on a cell beside the wall", end.X, end.Y, end.HasTarget)
			}
		})
	}
}

// TestWithdrawalWhoseFleeCellLiesBehindAWallSettlesBesideIt is the ordinary-play
// shape of a ranged creature that a hostile stands two cells from: the decoded
// flee cell is three cells away, behind a wall whose only way round is long. The
// creature's far search cannot label that cell within max(5, D>>2) + D
// generations and settles on a cell beside the wall, so the creature stays at
// its wall and keeps shooting; before, it spent the flat thousand and ran the
// whole way round.
func TestWithdrawalWhoseFleeCellLiesBehindAWallSettlesBesideIt(t *testing.T) {
	archer := withdrawalArcher(1, 2, 20, 20)
	intruder := withdrawalFighter(2, SelfSlot, 22, 20, 100)
	w, err := NewRelatedWorld(1, engBounds, ModeCanonical, Terrain{Block: farWallGrid()},
		[]Entity{archer, intruder}, nil, engRel(t, [3]uint32{2, SelfSlot, 1}, [3]uint32{SelfSlot, 2, 2}))
	if err != nil {
		t.Fatalf("NewRelatedWorld: %v", err)
	}
	engRun(w, 1)
	west := int32(20)
	for i := 0; i < 160; i++ {
		Step(w, nil)
		west = min(west, w.entities[0].X)
	}
	got := w.entities[0]
	if west < 18 {
		t.Fatalf("the archer stood as far west as column %d: it ran the detour round a wall its far search cannot label a flee cell behind", west)
	}
	if got.X < 19 || got.X > 20 || got.Y < 19 || got.Y > 21 {
		t.Errorf("the archer ended at (%d,%d), want it beside the wall it started at", got.X, got.Y)
	}
	if blows := blowsLanded(w, 100); blows < 1 {
		t.Errorf("the archer landed %d blow(s) in 160 ticks from beside its wall, want at least 1", blows)
	}
}

// TestTheFarBudgetRuleOfEachOwner pins the three numbers a search may spend.
// D is the Chebyshev distance from the mover's cell to the goal, the larger of
// the two axis differences. A participant's unit spends a thousand while the
// goal is open, every other owner's unit max(5, D>>2) + D whatever the goal is,
// and a participant's unit whose goal is closed spends that same computed form
// (MOVE-TERM-003).
func TestTheFarBudgetRuleOfEachOwner(t *testing.T) {
	for _, tc := range []struct {
		goal cell
		want int64
	}{
		{cell{0, 0}, 5},
		{cell{3, 0}, 8},
		{cell{12, 30}, 37},
		{cell{39, 0}, 48},
		{cell{-100, 7}, 125},
		{cell{900, 0}, 1125},
	} {
		start := cell{0, 0}
		if got := generationBudget(start, tc.goal, staticBudget, true); got != tc.want {
			t.Errorf("another owner's budget to (%d,%d) is %d, want %d", tc.goal.x, tc.goal.y, got, tc.want)
		}
		if got := generationBudget(start, tc.goal, staticBudget, false); got != tc.want {
			t.Errorf("another owner's budget to the closed goal (%d,%d) is %d, want %d", tc.goal.x, tc.goal.y, got, tc.want)
		}
		if got := generationBudget(start, tc.goal, flatBudget, false); got != tc.want {
			t.Errorf("a participant's budget to the closed goal (%d,%d) is %d, want %d", tc.goal.x, tc.goal.y, got, tc.want)
		}
		if got := generationBudget(start, tc.goal, flatBudget, true); got != flatGenerations {
			t.Errorf("a participant's budget to the open goal (%d,%d) is %d, want %d", tc.goal.x, tc.goal.y, got, flatGenerations)
		}
	}
}

// TestTheFarBudgetIsChosenByTheOwnerOfTheMover pins the one chooser every far
// search of an entity reads: the participant's slot and an entity naming no slot
// spend the flat budget and every other roster slot the computed one, for a
// one-cell entity. A larger one is chosen by its footprint alone.
func TestTheFarBudgetIsChosenByTheOwnerOfTheMover(t *testing.T) {
	w := mustWorldGrid(t, 1, engBounds, ModeCanonical, farWallGrid(), []Entity{
		{ID: 1, X: 20, Y: 20, Owner: SelfSlot},
		{ID: 2, X: 21, Y: 20, Owner: 0},
		{ID: 3, X: 22, Y: 20, Owner: 2},
		{ID: 4, X: 23, Y: 20, Owner: 7},
	})
	for i, want := range []budgetRule{flatBudget, flatBudget, staticBudget, staticBudget} {
		if got := w.farBudgetFor(i); got != want {
			t.Errorf("entity %d (owner %d): far budget rule %d, want %d", w.entities[i].ID, w.entities[i].Owner, got, want)
		}
	}
}
