package game

// The drawing half of map presence: a unit the mission script has taken off
// the map is not in the draw list, and so is neither painted nor available
// to be clicked — the window tier resolves a pick against the entities in
// that list and against nothing else.

import (
	"testing"

	"againrom/pkg/sim"
)

// prDrawWorld is two entities of a class the fixture bundle draws, one of which
// the caller may take off the map.
func prDrawWorld(t *testing.T, offMap sim.EntityID) *sim.World {
	t.Helper()
	w, err := sim.NewWorld(1, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, nil,
		[]sim.Entity{
			{ID: 1, X: 2, Y: 2, Class: 3, HP: 10, MaxHP: 10, Reach: 1},
			{ID: 2, X: 5, Y: 5, Class: 3, HP: 10, MaxHP: 10, Reach: 1},
		})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	if offMap != 0 {
		if err := prTakeOffMap(w, offMap); err != nil {
			t.Fatalf("taking %d off the map: %v", offMap, err)
		}
	}
	return w
}

// prTakeOffMap reaches the simulation's own arm the only way a package outside
// it can: through a mission script whose one trigger fires unconditionally on
// the first pass. That is also the RIGHT way for this test to reach it — the
// state under test is one a script writes and nothing else in the tree does, so
// a fixture that set a field directly would be testing a state the game cannot
// actually be in.
func prTakeOffMap(w *sim.World, id sim.EntityID) error {
	s, err := sim.NewScript(
		[]sim.ScriptCheck{{Op: sim.ScriptCheckConstant, Register: 0, Args: [10]int32{1}}},
		[]sim.ScriptInstant{{Op: sim.ScriptInstantTakeOffMap, Unit: id, HasUnit: true}},
		[]sim.ScriptTrigger{{
			Pairs:    [3]sim.ScriptPair{{Left: 0, Right: 0, Cmp: sim.ScriptCmpEQ, Used: true}},
			Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone},
			Once:     true,
		}})
	if err != nil {
		return err
	}
	scripted, err := sim.NewScriptedWorld(1, w.Bounds(), sim.ModeCanonical, nil, w.Entities(), s)
	if err != nil {
		return err
	}
	// One full script cycle, which is what a pass costs.
	for range 32 {
		sim.Step(scripted, nil)
	}
	*w = *scripted
	return nil
}

func TestAnOffMapUnitIsNotDrawn(t *testing.T) {
	t.Parallel()

	set := worldFixtureUnitSet()

	both := seamDraws(prDrawWorld(t, 0), set)
	if len(both) != 2 {
		t.Fatalf("the control draws %d entities, want 2 — this case would pass for the wrong reason", len(both))
	}

	one := seamDraws(prDrawWorld(t, 2), set)
	if len(one) != 1 {
		t.Fatalf("with one unit off the map the derivation draws %d entities, want 1", len(one))
	}
	if one[0].ID != 1 {
		t.Errorf("the drawn entity is %d, want the one still on the map, 1", one[0].ID)
	}
	// AND THE SURVIVING ENTRY IS THE WHOLE ENTRY, not a shifted one. The
	// compaction runs after the loop, over a slice every write in that loop
	// addressed by the entity's own index, so a filter that got the indexing
	// wrong would leave a zero-value MapEntity here rather than a real one.
	if one[0].Cell != both[0].Cell {
		t.Errorf("the surviving entry is at %v, want the cell it has when both are drawn, %v",
			one[0].Cell, both[0].Cell)
	}
	if one[0].Art != both[0].Art {
		t.Error("the surviving entry lost its art; the compaction shifted a field")
	}
}
