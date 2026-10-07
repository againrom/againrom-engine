package sim

import "testing"

// heldMoveFirstStep returns the ticks from recovery reaching zero to the first
// position change of an archer whose move to dest was written behind its loaded
// cycle, and the world at the tick before that change.
func heldMoveFirstStep(t *testing.T, dest CellPoint) (uint64, *World) {
	t.Helper()
	w := lcPlayerWorld(t)
	Step(w, []Command{MoveTo(lcArcherID, dest)})
	var recovered uint64
	for range 200 {
		Step(w, nil)
		e := w.entities[0]
		if e.AttackPhase == AttackBoundaryOne && recovered == 0 {
			recovered = w.tick
		}
		if recovered != 0 && (e.Transit != 0 || e.X != 20 || e.Y != 20) {
			return w.tick - recovered, w
		}
	}
	t.Fatal("the held move never started")
	return 0, nil
}

// A held move's first walk call is 2 ticks after recovery reaches zero and
// steps in that call only when the facing already equals the direction to the
// first node; otherwise it turns and steps at the next call (MOVE-090). The
// archer faces its victim, to the east.
func TestAHeldMoveFirstStepFollowsFacingAtRecoveryPlusTwoOrThree(t *testing.T) {
	cases := []struct {
		name string
		dest CellPoint
		want uint64
	}{
		{"matching facing east", CellPoint{X: 22, Y: 20}, 2},
		{"turn west", CellPoint{X: 14, Y: 20}, 3},
		{"turn south", CellPoint{X: 20, Y: 26}, 3},
		{"turn north", CellPoint{X: 20, Y: 14}, 3},
	}
	for _, c := range cases {
		if got, _ := heldMoveFirstStep(t, c.dest); got != c.want {
			t.Errorf("%s: first position change %d ticks after recovery reached zero, want %d", c.name, got, c.want)
		}
	}
}

// The held move crosses a SAVE and cold LOAD on the last boundary tick and
// steps on the same tick as the live world.
func TestAHeldMoveFirstStepSurvivesSaveAndColdLoad(t *testing.T) {
	w := lcPlayerWorld(t)
	Step(w, []Command{MoveTo(lcArcherID, CellPoint{X: 22, Y: 20})})
	for range 200 {
		Step(w, nil)
		if w.entities[0].AttackPhase == AttackBoundaryTwo {
			break
		}
	}
	if w.entities[0].AttackPhase != AttackBoundaryTwo {
		t.Fatal("the cycle never reached its last boundary tick")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var loaded World
	if err := loaded.UnmarshalBinary(form); err != nil {
		t.Fatalf("decode: %v", err)
	}
	Step(w, nil)
	Step(&loaded, nil)
	if w.entities[0].X != 21 {
		t.Fatalf("live archer at x %d, want the step to 21 on this tick", w.entities[0].X)
	}
	if w.Hash() != loaded.Hash() {
		t.Fatalf("decoded world diverged: live %#x decoded %#x", w.Hash(), loaded.Hash())
	}
}

// A held move whose first node lies behind the facing first changes position 3
// ticks after recovery reaches zero: the first walk call at 2 turns, the step
// follows at the next call (MOVE-090, AI-375).
func TestAHeldMoveFirstStepFollowsRecoveryByThreeTicks(t *testing.T) {
	if got, _ := heldMoveFirstStep(t, CellPoint{X: 14, Y: 20}); got != 3 {
		t.Fatalf("first position change %d ticks after recovery reached zero, want 3", got)
	}
}
