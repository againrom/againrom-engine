package sim

import "testing"

// poisonTrace attaches one Poison Cloud record to a unit and returns the
// target's health and the record's remainder after the attach and after each
// of n ticks.
func poisonTrace(t *testing.T, rule SpellRule, repeat bool, n int) (hp []int32, remaining []uint16) {
	t.Helper()
	w := spWorld(t, 1, []SpellRule{rule}, spEnt(1, 0, 0), spEnt(2, 1, 0))
	if !w.attachEffect(2, 1, rule, EffectHealth, -2, 128, EffectContinuous) {
		t.Fatal("first Poison Cloud did not attach")
	}
	if repeat && !w.attachEffect(2, 1, rule, EffectHealth, -2, 64, EffectContinuous) {
		t.Fatal("renewing Poison Cloud did not attach")
	}
	sample := func() {
		e, _ := w.Entity(2)
		hp = append(hp, e.HP)
		got := w.ActiveEffects()
		if len(got) != 1 {
			t.Fatalf("want one record, have %+v", got)
		}
		remaining = append(remaining, got[0].Remaining)
	}
	sample()
	for i := 0; i < n; i++ {
		Step(w, nil)
		sample()
	}
	return hp, remaining
}

func TestPoisonCloudFirstAttachHealthChange(t *testing.T) {
	const ticks = 20
	// Ticks 1..ticks: the remainder starts at 128, a multiple of 8, so the
	// first every-eighth-step tick comes on the first Step and the next at
	// remainder 120, nine Steps later.
	for _, c := range []struct {
		name    string
		second  bool
		repeat  bool
		atCast  int32
		wantHP9 int32
	}{
		{"second game first attach", true, false, 100, 96},
		{"first game first attach", false, false, 98, 94},
		{"second game renewal", true, true, 100, 96},
	} {
		rule := SpellRule{ID: 8, School: 2, Area: true, Distribution: 3, EffectKind: EffectHealth, EffectMode: EffectContinuous, EffectMagnitude: -2, EffectDuration: 128}
		if c.second {
			rule = secondRow(SpellRule{ID: 6, School: 2, Area: true, Distribution: 3, SpellDuration: 5, EffectKind: EffectHealth, EffectMode: EffectContinuous, EffectMagnitude: -2, EffectDuration: 8})
		}
		hp, rem := poisonTrace(t, rule, c.repeat, ticks)
		if hp[0] != c.atCast {
			t.Errorf("%s: health at attach %d, want %d", c.name, hp[0], c.atCast)
		}
		for k := 1; k <= ticks; k++ {
			// Every eighth remainder value (128, 120, ...) drops health by 2.
			drops := int32((k-1)/8 + 1)
			want := c.atCast - 2*drops
			if hp[k] != want {
				t.Errorf("%s: tick %d health %d, want %d", c.name, k, hp[k], want)
			}
			if rem[k] != 128-uint16(k) {
				t.Errorf("%s: tick %d remainder %d, want %d", c.name, k, rem[k], 128-uint16(k))
			}
		}
		if hp[9] != c.wantHP9 {
			t.Errorf("%s: health after nine ticks %d, want %d", c.name, hp[9], c.wantHP9)
		}
	}
}
