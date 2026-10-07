package game

import (
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The world's own step-cost read reaching the viewer.
//
// It is an internal test because what it is about is the DRIVER's installation
// of the question — the one statement that decides whether the box has anything
// to ask, and when.

// stepCostWorld holds two movers of different speed a step apart, and one that
// carries no speed at all. Three distinct answers, so an installation that
// answered from the wrong entity could not pass.
func stepCostWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil,
		[]sim.Entity{
			{ID: 0, X: 3, Y: 4, Speed: 16, HP: 40, MaxHP: 40},
			{ID: 1, X: 8, Y: 9, Speed: 40, HP: 40, MaxHP: 40},
			{ID: 2, X: 12, Y: 12, HP: 40, MaxHP: 40},
		})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	return w
}

func stepCostDriver(t *testing.T) (*mapWorld, *ui.Viewer) {
	t.Helper()
	v := worldFixtureViewer(t, worldFixtureMap())
	return newMapWorld(stepCostWorld(t), nil, nil, v), v
}

func TestTheViewerAnswersWithTheWorldsOwnStepFigures(t *testing.T) {
	mw, v := stepCostDriver(t)

	for _, tc := range []struct {
		what     string
		id       uint32
		col, row int
	}{
		{"a step onto the neighbouring cell", 0, 4, 4},
		{"a diagonal step", 0, 4, 5},
		{"a faster mover's own step", 1, 9, 9},
		{"a cell nowhere near the mover", 1, 20, 3},
	} {
		t.Run(tc.what, func(t *testing.T) {
			got, ok := v.StepCostAt(tc.id, tc.col, tc.row)
			if !ok {
				t.Fatal("the viewer answered nothing for a rated live mover")
			}
			rate, transit, adjacent, wantOK := mw.world.StepRate(
				sim.EntityID(tc.id), int32(tc.col), int32(tc.row))
			if !wantOK {
				t.Fatal("the world refused what the viewer answered")
			}
			want := ui.StepCost{Rate: int(rate), Transit: int(transit), Adjacent: adjacent}
			if got != want {
				t.Errorf("the viewer answers %+v and the world %+v", got, want)
			}
		})
	}
}

func TestTheDriverAddsNoRefusalOfItsOwn(t *testing.T) {
	mw, v := stepCostDriver(t)

	for _, tc := range []struct {
		what     string
		id       uint32
		col, row int
	}{
		{"an id no world holds", 99, 4, 4},
		{"a mover carrying no speed", 2, 13, 12},
		{"a destination equal to the source", 0, 3, 4},
	} {
		t.Run(tc.what, func(t *testing.T) {
			_, _, _, worldOK := mw.world.StepRate(sim.EntityID(tc.id), int32(tc.col), int32(tc.row))
			got, ok := v.StepCostAt(tc.id, tc.col, tc.row)
			if worldOK {
				t.Fatalf("the world answered this case — the fixture asserts nothing")
			}
			if ok {
				t.Errorf("the viewer answered %+v where the world refused", got)
			}
			if got != (ui.StepCost{}) {
				t.Errorf("a refused answer still carried %+v", got)
			}
		})
	}
}

func TestTheQuestionIsInstalledBeforeAnyTickRuns(t *testing.T) {
	mw, v := stepCostDriver(t)

	if got := mw.world.Tick(); got != 0 {
		t.Fatalf("the fixture's world has already advanced to tick %d", got)
	}
	if _, ok := v.StepCostAt(0, 4, 4); !ok {
		t.Error("a freshly opened world answers no step cost until it has ticked")
	}
}

// TestAViewerNobodyInstalledOnAnswersNothing is the other end of the same rule —
// the zero value is an absence and not a zero figure, which is what leaves every
// front-end written before this story unchanged.
func TestAViewerNobodyInstalledOnAnswersNothing(t *testing.T) {
	v := worldFixtureViewer(t, worldFixtureMap())

	got, ok := v.StepCostAt(0, 4, 4)
	if ok {
		t.Errorf("a viewer holding no question answered %+v", got)
	}
	if got != (ui.StepCost{}) {
		t.Errorf("a viewer holding no question carried %+v", got)
	}
}

func TestTheQuestionFollowsTheDriversCurrentWorld(t *testing.T) {
	mw, v := stepCostDriver(t)

	before, ok := v.StepCostAt(1, 9, 9)
	if !ok {
		t.Fatal("the viewer answered nothing for a rated live mover")
	}

	// The same entity id at the same cells, in a world that gives it a different
	// speed. Nothing but the driver's world pointer moves.
	slower, err := sim.NewWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, nil,
		[]sim.Entity{{ID: 1, X: 8, Y: 9, Speed: 8, HP: 40, MaxHP: 40}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw.world = slower

	after, ok := v.StepCostAt(1, 9, 9)
	if !ok {
		t.Fatal("the viewer answered nothing after the driver's world was replaced")
	}
	if after == before {
		t.Errorf("the answer did not follow the driver's world: %+v either way", before)
	}
}
