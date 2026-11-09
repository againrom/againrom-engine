package sim

import "testing"

func TestComputerLookAroundUsesTheSharedIdleTurn(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-504 and R2-ENGINE-844: activity 11 consumes the hit alarm.
	w := orderFallbackScene(t, true, "computer support")
	if o := w.savedOrder(asHero); o.Raw[0x54] != 0 || o.Raw[8] != 0xb {
		t.Fatalf("look-around activity/alarm=%d/%d", o.Raw[8], o.Raw[0x54])
	}
	if e := w.entities[0]; e.Facing == 64 || e.HasTarget || e.HasAttackTarget || e.X != 30 || e.Y != 40 {
		t.Fatalf("computer did not look around on its cell: %+v", e)
	}
}

func TestLookAroundKeepsHeadingUntilTheTurnEnds(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-504: an unfinished heading turns without drawing a new one.
	w := asWorld(t, true, asFighter(asHero, asSide, 30, 40))
	w.entities[0].RotationSpeed = 1
	w.entities[0].requestFacing(128)
	w.entities[0].EscortTurnPending = true
	before := w.RandomState()
	desired := w.entities[0].DesiredFacing
	w.stepIdleTurn(0)
	if w.RandomState() != before || w.entities[0].DesiredFacing != desired || !w.entities[0].EscortTurnPending {
		t.Fatal("unfinished look-around consumed an alarm or drew another heading")
	}
}

func TestLookAroundRespectsPrimaryAndPlayerReplacement(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-542: prepared activity waits for primary progress zero.
	w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40))
	w.commandDefend([]int{0}, asHero)
	o := w.savedOrder(asHero)
	o.Raw[8], o.Raw[9], o.Raw[0x54] = 0xb, 1, 1
	w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 3
	before := w.RandomState()
	if w.stepEscortOrder(nil, 0) || w.RandomState() != before || o.Raw[0x54] == 0 {
		t.Fatal("look-around ran ahead of primary progress")
	}
	w.entities[0].clearAttack()
	o.Raw[9] = 0
	Step(w, []Command{MoveTo(asHero, CellPoint{X: 34, Y: 40})})
	if e := w.entities[0]; !e.HasTarget || e.TargetX != 34 || e.TargetY != 40 {
		t.Fatal("look-around replaced a player move")
	}
}
