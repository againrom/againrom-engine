package sim

import "testing"

func TestSecondGamePlayerOrderScenario(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1042, R2-ENGINE-1076, R2-ENGINE-1105/1106/1107.
	w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 34, 41), asFoe(asFoeB, 45, 40))
	Step(w, []Command{GroupMoveTo(asHero, CellPoint{X: 33, Y: 40}, 1)})
	arrived := false
	for n := 0; n < 240; n++ {
		e := w.entities[0]
		atGoal := e.X == 33 && e.Y == 40 && e.Transit == 0
		if !atGoal && e.HasAttackTarget {
			t.Fatalf("walk acquired %d before arrival", e.AttackTarget)
		}
		if atGoal && e.HasAttackTarget {
			if o := w.savedOrder(asHero); o.State != 12 || o.Raw[8] != 6 {
				t.Fatalf("arrival order/activity=%d/%d, want 12/6", o.State, o.Raw[8])
			}
			arrived = true
			break
		}
		Step(w, nil)
	}
	if !arrived {
		t.Fatal("move did not reach adjacent idle attack")
	}
	Step(w, []Command{Attack(asHero, asFoeB), MoveTo(asFoeB, CellPoint{X: 55, Y: 40})})
	for range 90 {
		Step(w, nil)
		id, _, ok := w.entities[0].RequestedAttackTarget()
		if !ok || id != asFoeB {
			t.Fatalf("automatic choice replaced explicit moving target with %v/%d", ok, id)
		}
	}
	if w.entities[0].X <= 33 {
		t.Fatal("explicit attack did not pursue the moving target")
	}
	Step(w, []Command{MoveTo(asHero, CellPoint{X: 32, Y: 45})})
	if e := w.entities[0]; !e.HasTarget || e.TargetX != 32 || e.TargetY != 45 {
		t.Fatalf("replacement click not attached immediately: %+v", e)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	finished := false
	for n := 0; n < 400; n++ {
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("replacement continuation diverged at step %d", n)
		}
		if e := w.entities[0]; e.X == 32 && e.Y == 45 && e.Transit == 0 && !e.HasTarget && e.ActorState == actorStateAcquire {
			finished = true
			break
		}
	}
	if !finished {
		e := w.entities[0]
		t.Fatalf("replacement did not finish: cell=(%d,%d) destination=%v state=%d", e.X, e.Y, e.HasTarget, e.ActorState)
	}
}

func TestSecondGameComputerChoiceUsesTheSharedMachine(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-1111: assignment keeps least score, then the first full tie.
	self := asFighter(asHero, asSide, 30, 40)
	self.Facing = 64
	w := asWorld(t, true, self, asFighter(asFoeA, SelfSlot, 30, 39), asFighter(asFoeB, SelfSlot, 31, 40))
	w.commandGroup([]int{0}, orderStandGround, cell{})
	for range 7 {
		Step(w, nil)
	}
	if e := w.entities[0]; !e.HasAttackTarget || e.AttackTarget != asFoeB || e.AcquirePursuit {
		t.Fatalf("computer target=%v/%d acquisition=%v, want assigned %d", e.HasAttackTarget, e.AttackTarget, e.AcquirePursuit, asFoeB)
	}
}
