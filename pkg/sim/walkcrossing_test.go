package sim

import "testing"

// stepX is the column a mover's step is drawn toward (StepCell).
func stepX(e Entity) int32 {
	x, _ := e.StepCell()
	return x
}

// walkFormGrowth is the walk wrapper's share of a form: nothing without a
// walk record, else its version byte, tag and span word plus one 15-byte record
// per unit whose walk state is not zero.
func walkFormGrowth(w *World) int {
	n := 0
	for _, e := range w.entities {
		if e.Walk != (WalkRefresh{}) {
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return 1 + 4 + 4 + walkRefreshRecordLen*n
}

// A walker moved to another cell mid-route ends its walk with its target: the
// counters go with it, and the byte form and the action continuation load the
// state both before and after it stands a while.
func TestARelocatedWalkerDropsItsWalkStateWithItsTarget(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		w := walkStepWorld(t, false, walkStepUnit(1, 5, 5, 16))
		if second {
			script, err := NewROM2Script(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			w.script = script
			w.initializeROM2Script()
		}
		Step(w, []Command{MoveTo(1, CellPoint{X: 25, Y: 5})})
		for k := 0; k < 30; k++ {
			Step(w, nil)
		}
		if e := w.entities[0]; !e.HasTarget || e.Walk == (WalkRefresh{}) {
			t.Fatalf("the fixture holds no walk mid-route: target %v walk %+v", e.HasTarget, e.Walk)
		}
		if !w.relocateActor(0, 30, 30, nil) {
			t.Fatal("the relocation was refused")
		}
		for tick := 0; tick <= 5; tick++ {
			if tick > 0 {
				Step(w, nil)
			}
			e := w.entities[0]
			if e.HasTarget || e.Stall != 0 || e.Walk != (WalkRefresh{}) {
				t.Fatalf("tick %d: the relocated walker holds target %v stall %d walk %+v", tick, e.HasTarget, e.Stall, e.Walk)
			}
			var cold World
			if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
				t.Fatalf("tick %d: the byte form does not load: %v", tick, err)
			}
			// A decoded second-game World hashes apart from the live one
			// it was written from, so only the first game's digest compares.
			if !second && cold.Hash() != w.Hash() {
				t.Fatalf("tick %d: the loaded form hashes otherwise", tick)
			}
			if err := cold.RestoreActions(actionCopy(t, w.Actions()), nil); err != nil {
				t.Fatalf("tick %d: the action continuation does not restore: %v", tick, err)
			}
		}
	})
}

// A unit counting passes behind a held listed step keeps its walk state
// through the byte form, the digest and the action continuation, and the
// copies walk on as the original does. Zeroing the state changes the digest.
func TestTheWalkStateSurvivesTheByteFormAndTheActionContinuation(t *testing.T) {
	walkStepGames(t, func(t *testing.T, second bool) {
		a := walkStepUnit(1, 5, 5, 16)
		a.Facing, a.RotationSpeed = 0, 8
		w := walkStepWorld(t, false, a, walkStepUnit(2, 20, 20, 16))
		if second {
			script, err := NewROM2Script(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			w.script = script
			w.initializeROM2Script()
		}
		Step(w, []Command{MoveTo(1, CellPoint{X: 10, Y: 5})})
		w.entities[1].X, w.entities[1].Y = 6, 5
		for k := 0; k < 24; k++ {
			Step(w, nil)
		}
		walk := w.entities[0].Walk
		if !walk.Listed || walk.Age == 0 || walk.Searches == 0 || walk.NextX != 6 || walk.NextY != 5 {
			t.Fatalf("the fixture holds no counted listed step: %+v", walk)
		}
		var cold, lost World
		if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
			t.Fatal(err)
		}
		if err := lost.UnmarshalBinary(mustMarshal(t, w)); err != nil {
			t.Fatal(err)
		}
		if second != (cold.rom2 != nil) {
			t.Fatal("the byte form lost the game")
		}
		// Both copies are decoded from one form, so the digests compare copies
		// with each other.
		lost.entities[0].Walk = WalkRefresh{}
		if cold.entities[0].Walk != walk || cold.Hash() == lost.Hash() {
			t.Fatalf("the byte form carried walk %+v, want %+v", cold.entities[0].Walk, walk)
		}
		if err := lost.RestoreActions(actionCopy(t, w.Actions()), nil); err != nil {
			t.Fatal(err)
		}
		if lost.entities[0].Walk != walk || lost.Hash() != cold.Hash() {
			t.Fatalf("the action continuation carried walk %+v, want %+v", lost.entities[0].Walk, walk)
		}
		for tick := 1; tick <= 40; tick++ {
			Step(w, nil)
			Step(&cold, nil)
			Step(&lost, nil)
			e, c := w.entities[0], cold.entities[0]
			if lost.Hash() != cold.Hash() || c.X != e.X || c.Y != e.Y || c.Facing != e.Facing || c.Walk != e.Walk {
				t.Fatalf("tick %d: a restored copy walked otherwise: %+v, want %+v", tick, c.Walk, e.Walk)
			}
		}
	})
}
