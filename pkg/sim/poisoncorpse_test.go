package sim

import "testing"

// Poison follows the original on a fallen unit: its pulses test no health, so
// they keep running on the body below zero until the body is torn down, where
// every attachment of the body expires. A living unit under the same poison is
// unchanged.
func TestPoisonKeepsTickingOnAFallenBodyUntilItsTeardown(t *testing.T) {
	rule := SpellRule{ID: 8, School: 2, Area: true, Distribution: 3, EffectKind: EffectHealth, EffectMode: EffectContinuous, EffectMagnitude: -2, EffectDuration: 9000}
	weak := spEnt(2, 1, 0)
	weak.HP = 1
	w := spWorld(t, 1, []SpellRule{rule}, spEnt(1, 0, 0), weak, spEnt(3, 2, 0))
	for _, id := range []EntityID{2, 3} {
		if !w.attachEffect(id, 1, rule, EffectHealth, -2, 9000, EffectContinuous) {
			t.Fatalf("Poison Cloud did not attach to %d", id)
		}
	}
	var fallenPulses int
	var pulseBelowZero, torn bool
	for tick := 1; tick <= 3000 && !torn; tick++ {
		before, _ := w.Entity(2)
		rep := StepReported(w, nil)
		body, ok := w.Entity(2)
		held := false
		for _, a := range w.ActiveEffects() {
			if a.Target == 2 {
				held = true
			}
		}
		if before.Dead() && !held {
			torn = true
			if ok && body.OrdinaryTargetable() {
				t.Errorf("tick %d: the poison left a body at %d health that is not torn down", tick, body.HP)
			}
			break
		}
		if !ok {
			torn = true
			break
		}
		if before.Dead() {
			for _, d := range rep.Damages {
				if d.Target == 2 {
					fallenPulses++
				}
			}
			if body.HP < 0 {
				pulseBelowZero = true
			}
		}
	}
	if !torn {
		t.Fatal("the fallen body was never torn down")
	}
	if fallenPulses == 0 || !pulseBelowZero {
		t.Errorf("no poison pulse reached the fallen body (pulses %d)", fallenPulses)
	}
	for _, a := range w.ActiveEffects() {
		if a.Target == 2 {
			t.Errorf("the torn-down body still holds poison %+v", a)
		}
	}
	if living, _ := w.Entity(3); living.HP >= 100 || len(w.ActiveEffects()) == 0 {
		t.Errorf("the living unit holds %d health and %d effects, want it poisoned", living.HP, len(w.ActiveEffects()))
	}
}
