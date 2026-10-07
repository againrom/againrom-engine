package sim

import "testing"

func TestAppliedPotionRestoreValidatesAndExpiresWithoutReapplying(t *testing.T) {
	w, err := NewWorld(1, Bounds{Width: 8, Height: 8}, ModeCanonical, nil, []Entity{{ID: 1, X: 2, Y: 2, HP: 10, MaxHP: 10, HealthRegeneration: 125}})
	if err != nil {
		t.Fatal(err)
	}
	effect := ActiveEffect{Target: 900, Kind: EffectHealthRegeneration, Mode: EffectDuration, Magnitude: 100, Remaining: 3}
	for _, bad := range []ActiveEffect{{}, {Kind: EffectHealth, Mode: EffectDuration, Magnitude: 100, Remaining: 3}} {
		before := w.Hash()
		if w.RestoreAppliedPotionEffect(1, bad) || w.Hash() != before {
			t.Fatal("malformed potion mutated the world")
		}
	}
	before := w.Hash()
	if w.RestoreAppliedPotionEffect(2, effect) || w.Hash() != before {
		t.Fatal("missing potion owner mutated the world")
	}
	if !w.RestoreAppliedPotionEffect(1, effect) {
		t.Fatal("valid applied potion refused")
	}
	before = w.Hash()
	if w.RestoreAppliedPotionEffect(1, effect) || w.Hash() != before {
		t.Fatal("repeated restore duplicated the potion")
	}
	for tick := 0; tick <= 4; tick++ {
		e, _ := w.Entity(1)
		want := int32(125)
		if tick >= 3 {
			want = 25
		}
		if e.HealthRegeneration != want {
			t.Fatal("restored potion applied or removed twice", tick, e.HealthRegeneration, want)
		}
		if tick < 3 && (len(w.ActiveEffects()) != 1 || w.ActiveEffects()[0].Target != 1 || w.ActiveEffects()[0].Remaining != uint16(3-tick)) || tick >= 3 && len(w.ActiveEffects()) != 0 {
			t.Fatal("restored timer differs", tick, w.ActiveEffects())
		}
		Step(w, nil)
	}
}
