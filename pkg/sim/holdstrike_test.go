package sim

import "testing"

// holdCmd is the player's Hold Position order for the one actor id 1.
func holdCmd() Command { return GroupStance(1, OrderStandGround, 2) }

func holdStrikeCastWorld(t *testing.T, cellCast bool) *World {
	t.Helper()
	w := manualCastWorld(t)
	w.entities[0].AlwaysHits, w.entities[0].DamageBase = true, 5
	w.entities[0].Reach = 1
	w.entities[1].HP, w.entities[1].MaxHP = 200, 200
	w.orderAttack(0, 2)
	w.entities[0].AttackPhase, w.entities[0].AttackCountdown = AttackCharging, 6
	cmd := Cast(1, 2, 1)
	if cellCast {
		cmd = CastAt(1, 26, CellPoint{X: 6, Y: 3})
	}
	Step(w, []Command{cmd})
	if w.entities[0].PendingOrder.Kind < PendingActorCast {
		t.Fatal("fixture: cast was not queued behind the loaded strike")
	}
	return w
}

// A queued manual cast is a pending row the Hold setter replaces with 0, while
// the loaded strike keeps its victim and lands (AI-351, AI-354, AI-356).
func TestHoldStrikeReplacesQueuedCast(t *testing.T) {
	for _, cellCast := range []bool{false, true} {
		w := holdStrikeCastWorld(t, cellCast)
		control := worldRoundTripForTest(t, w)
		before := w.entities[0]
		Step(w, []Command{holdCmd()})
		e := w.entities[0]
		if e.PendingOrder.Kind != PendingNone || !e.HasAttackTarget || e.AttackTarget != 2 || e.AttackPhase != before.AttackPhase || e.AttackCountdown != before.AttackCountdown-1 {
			t.Fatalf("Hold did not replace the queued cast and keep the strike: pending=%+v phase=%d", e.PendingOrder, e.AttackPhase)
		}
		back := worldRoundTripForTest(t, w)
		for range 150 {
			Step(w, nil)
			Step(back, nil)
			Step(control, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("cold Hold continuation differs")
			}
			if len(w.bookCasts) != 0 || w.entities[0].Mana != 1000 {
				t.Fatal("queued cast ran after Hold")
			}
		}
		if len(control.bookCasts) == 0 && control.entities[0].Mana == 1000 {
			t.Fatal("control: the queued cast never ran without Hold")
		}
		if w.entities[1].HP > 195 || w.entities[0].X != 3 || w.entities[0].Y != 3 {
			t.Fatalf("the loaded blow was lost or the actor moved: hp=%d", w.entities[1].HP)
		}
	}
}

func TestHoldStrikeReplacesQueuedPickupAndRelease(t *testing.T) {
	for _, kind := range []uint8{PendingPickup, PendingRelease} {
		w := lcPlayerWorld(t)
		w.sacks = []Sack{{X: 20, Y: 20, Gold: 500}}
		if kind == PendingPickup {
			Step(w, []Command{PickUp(1, CellPoint{X: 20, Y: 20})})
		} else {
			w.releaseAttack(0)
		}
		if w.entities[0].PendingOrder.Kind != kind {
			t.Fatalf("fixture: pending kind %d, want %d", w.entities[0].PendingOrder.Kind, kind)
		}
		Step(w, []Command{GroupStance(1, OrderStandGround, SelfSlot)})
		e := w.entities[0]
		if e.PendingOrder.Kind != PendingNone || !e.HasAttackTarget || e.AttackPhase == AttackReady {
			t.Fatalf("Hold did not replace pending kind %d and keep the strike: pending=%+v", kind, e.PendingOrder)
		}
		back := worldRoundTripForTest(t, w)
		for range 200 {
			Step(w, nil)
			Step(back, nil)
			if w.Hash() != back.Hash() {
				t.Fatal("cold Hold continuation differs")
			}
		}
		if kind == PendingPickup && (len(w.sacks) != 1 || w.entities[0].PendingOrder.Kind != PendingNone) {
			t.Fatal("pickup ran after Hold")
		}
		if w.entities[1].HP > 195 {
			t.Fatalf("the loaded blow was lost: hp=%d", w.entities[1].HP)
		}
	}
}

// Negative control: Hold with no pending row changes only what it did before.
func TestHoldStrikeWithoutPendingRowKeepsStrike(t *testing.T) {
	w := lcPlayerWorld(t)
	before := w.entities[0]
	Step(w, []Command{GroupStance(1, OrderStandGround, SelfSlot)})
	e := w.entities[0]
	if e.PendingOrder.Kind != PendingNone || !e.HasAttackTarget || e.AttackPhase != before.AttackPhase {
		t.Fatalf("Hold disturbed a plain loaded strike: pending=%+v phase=%d", e.PendingOrder, e.AttackPhase)
	}
}

// Hold, then a second order: the second order queues behind the loaded strike
// as it would without Hold, and a later Retreat replaces it.
func TestHoldStrikeThenSecondOrderAndRetreat(t *testing.T) {
	w := holdStrikeCastWorld(t, false)
	Step(w, []Command{holdCmd()})
	Step(w, []Command{CastAt(1, 26, CellPoint{X: 6, Y: 3})})
	e := w.entities[0]
	if e.PendingOrder.Kind != PendingCellCast || !e.HasAttackTarget || e.AttackTarget != 2 || len(w.bookCasts) != 0 {
		t.Fatalf("second order after Hold did not queue behind the strike: pending=%+v phase=%d", e.PendingOrder, e.AttackPhase)
	}
	back := worldRoundTripForTest(t, w)
	ran := false
	for range 150 {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("cold continuation differs")
		}
		ran = ran || len(w.bookCasts) != 0
	}
	if !ran || w.entities[0].X != 6 || w.entities[0].Y != 3 {
		t.Fatalf("second order did not run after the strike: pending=%+v", w.entities[0].PendingOrder)
	}

	w = holdStrikeCastWorld(t, true)
	Step(w, []Command{holdCmd()})
	Step(w, []Command{GroupRetreat(1, 2, 7)})
	e = w.entities[0]
	if e.PendingOrder.Kind != PendingNone || e.ActorState != actorStateRetreat || !e.Retreat.Known || !e.HasAttackTarget {
		t.Fatalf("Retreat after Hold: pending=%+v phase=%d", e.PendingOrder, e.AttackPhase)
	}
	for range 150 {
		Step(w, nil)
		if len(w.bookCasts) != 0 {
			t.Fatal("queued cast ran after Hold and Retreat")
		}
	}
}
