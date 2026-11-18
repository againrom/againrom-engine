package sim

import "testing"

// A unit that poison has killed takes no further poison damage and shows no
// damage number; a living unit under the same poison is unchanged.
func TestPoisonStopsOnTheBodyItKilled(t *testing.T) {
	rule := SpellRule{ID: 8, School: 2, Area: true, Distribution: 3, EffectKind: EffectHealth, EffectMode: EffectContinuous, EffectMagnitude: -2, EffectDuration: 128}
	weak := spEnt(2, 1, 0)
	weak.HP = 1
	w := spWorld(t, 1, []SpellRule{rule}, spEnt(1, 0, 0), weak, spEnt(3, 2, 0))
	for _, id := range []EntityID{2, 3} {
		if !w.attachEffect(id, 1, rule, EffectHealth, -2, 128, EffectContinuous) {
			t.Fatalf("Poison Cloud did not attach to %d", id)
		}
	}
	died := false
	var livingDrops, bodyEventsAfter int
	for tick := 1; tick <= 40; tick++ {
		before3, _ := w.Entity(3)
		rep := StepReported(w, nil)
		body, _ := w.Entity(2)
		living, _ := w.Entity(3)
		if living.HP < before3.HP {
			livingDrops++
		}
		for _, d := range rep.Damages {
			if d.Target == 2 && died {
				bodyEventsAfter++
			}
		}
		if body.Dead() {
			died = true
		}
	}
	if !died {
		t.Fatal("the poisoned unit never died")
	}
	for _, a := range w.ActiveEffects() {
		if a.Target == 2 {
			t.Errorf("the dead unit still holds poison %+v", a)
		}
	}
	if bodyEventsAfter != 0 {
		t.Errorf("%d damage numbers after death", bodyEventsAfter)
	}
	if livingDrops < 4 {
		t.Errorf("living unit lost health on %d ticks, want at least 4", livingDrops)
	}
}
