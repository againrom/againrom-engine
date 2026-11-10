package sim

import "testing"

// primaryCellScene gives a standing hero an imported primary byte, orders it
// to walk and steps once.
func primaryCellScene(t *testing.T, second bool, primary uint8) *World {
	t.Helper()
	w := asWorld(t, second, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 45, 40))
	w.ensureSavedOrder(0).Raw[9] = primary
	Step(w, []Command{MoveTo(asHero, CellPoint{X: 34, Y: 40})})
	return w
}

func TestAnImportedPrimaryFiveToTwoFiftyFourClearsAtTheFirstCall(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-553 (High, all 255 cells): +9 of 5..254 takes the clear arm,
	// +9 = 0, and that call dispatches no prepared action.
	for _, primary := range []uint8{5, 7, 254} {
		w := primaryCellScene(t, true, primary)
		if o := w.savedOrder(asHero); o.Raw[9] != 0 || w.ActorOrderProgress(asHero) != 0 {
			t.Fatalf("primary %d after the first call=%d, want 0", primary, o.Raw[9])
		}
		if e := w.entities[0]; e.X != 30 || e.Transit != 0 {
			t.Fatalf("primary %d: the clearing call stepped to %d/%d", primary, e.X, e.Transit)
		}
		for range 120 {
			Step(w, nil)
		}
		if e := w.entities[0]; e.X != 34 {
			t.Fatalf("primary %d: the walk after the clear ended at x=%d, want 34", primary, e.X)
		}
	}
}

func TestAnImportedHoldByteWithoutItsEffectClears(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-553 (High): the hold arm clears +9 when the effect is gone.
	w := primaryCellScene(t, true, 4)
	if o := w.savedOrder(asHero); o.Raw[9] != 0 {
		t.Fatalf("hold byte without the hold=%d, want 0", o.Raw[9])
	}
}

// primaryStrikeScene starts a hero's blow on an adjacent foe and, once the
// cycle is charging, gives the hero an imported primary byte.
func primaryStrikeScene(t *testing.T, primary uint8) *World {
	t.Helper()
	w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 31, 40))
	Step(w, []Command{Attack(asHero, asFoeA)})
	Step(w, nil)
	Step(w, nil)
	w.ensureSavedOrder(0).Raw[9] = primary
	return w
}

func TestAnImportedPrimaryTwoFiftyFiveKeepsItsByteAndLetsTheActorWalk(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-553 holds every executor at 255 and R2-ENGINE-554 leaves the
	// exit unbuilt (DIV-4799): the byte is kept and the unit walks as it did.
	w := primaryCellScene(t, true, 255)
	for range 120 {
		Step(w, nil)
	}
	if e, o := w.entities[0], w.savedOrder(asHero); o.Raw[9] != 255 || e.X != 34 || e.Transit != 0 {
		t.Fatalf("kept primary=%d x=%d transit=%d, want 255 arrived at 34", o.Raw[9], e.X, e.Transit)
	}
}

func TestAnImportedPrimaryTwoFiftyFiveAnswersAnAttackOrder(t *testing.T) {
	t.Parallel()
	hp := func(primary uint8) int32 {
		w := asWorld(t, true, asFighter(asHero, SelfSlot, 30, 40), asFoe(asFoeA, 31, 40))
		w.ensureSavedOrder(0).Raw[9] = primary
		Step(w, []Command{Attack(asHero, asFoeA)})
		for range 40 {
			Step(w, nil)
		}
		return w.entities[1].HP
	}
	if open, kept := hp(0), hp(255); kept != open || kept >= 1000 {
		t.Fatalf("foe health %d with 255, %d with 0, want the same blows below 1000", kept, open)
	}
}

func TestAnImportedPrimaryTwoFiftyFiveFinishesAStrikeInProgress(t *testing.T) {
	t.Parallel()
	w := primaryStrikeScene(t, 255)
	if w.entities[0].AttackPhase == AttackReady {
		t.Fatal("the scene has no strike in progress")
	}
	before := w.entities[1].HP
	for range 30 {
		Step(w, nil)
	}
	if after := w.entities[1].HP; after >= before {
		t.Fatalf("foe health %d -> %d, want the charging strike to land", before, after)
	}
}

func TestAnImportedPrimaryTwoFiftyFiveHoldsOnlyAPendingOrder(t *testing.T) {
	t.Parallel()
	// The keep cell is the admission's answer, not a gate's: it holds the
	// pending order that main held through its progress byte and no other.
	w := primaryCellScene(t, true, 255)
	a := w.admission(0)
	if !a.waits(executePending) || a.waits(executeMove) || a.waits(executeStrike) || a.waits(executeCast) {
		t.Fatalf("keep cell waits: pending %v move %v strike %v cast %v",
			a.waits(executePending), a.waits(executeMove), a.waits(executeStrike), a.waits(executeCast))
	}
}

func TestSecondGameImportedPrimaryScenario(t *testing.T) {
	t.Parallel()
	// R2-ENGINE-553: a cleared byte admits a queued order on the next call;
	// the continuation survives a synthetic binary round trip.
	w := primaryCellScene(t, true, 9)
	cold := worldRoundTripForTest(t, w)
	for range 40 {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("cleared primary diverged after synthetic LOAD")
		}
	}
	kept := primaryCellScene(t, true, 255)
	coldKept := worldRoundTripForTest(t, kept)
	for range 40 {
		Step(kept, nil)
		Step(coldKept, nil)
		if kept.Hash() != coldKept.Hash() || coldKept.savedOrder(asHero).Raw[9] != 255 {
			t.Fatal("kept primary diverged after synthetic LOAD")
		}
	}
	if e := kept.entities[0]; e.X == 30 {
		t.Fatal("kept primary never walked")
	}
}
