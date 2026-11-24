package sim

import "testing"

// bodyHealthAt steps a downed unit through ticks sub-ticks and returns its health and stage.
func bodyHealthAt(t *testing.T, second bool, hp int32, ticks uint64) (int32, DecayStage) {
	t.Helper()
	e := Entity{ID: 1, X: 3, Y: 3, HP: hp, MaxHP: 50, HealthRegenPeriod: 25, Decay: DecayFallen}
	e.SourceBinding.RuntimeID = 5
	var s *Script
	if second {
		s = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{e}, s, Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for w.Tick() < ticks {
		Step(w, nil)
	}
	got := w.Entities()[0]
	return got.HP, got.Decay
}

// R2-ENGINE-1207 and R2-ENGINE-1765: a second-game body above -10 loses 1
// health per 64 sub-ticks to the pool pass alone; from -10 down it is in the
// dead state, which the pool pass skips, and corpse ageing takes 1 per 32.
func TestSecondGameBodyLosesHealthAtTheClaimsSum(t *testing.T) {
	t.Parallel()
	want := func(ticks uint64) int32 {
		// The pool pass takes its point first on every 64th sub-tick while the
		// body is above -10; the walk then takes one per 32 from -10 down.
		hp := int32(-1)
		for k := uint64(0); k < ticks; k++ {
			if k%64 == 12 && hp > -10 {
				hp--
			}
			if hp <= -10 && k%32 == 12 {
				hp--
			}
		}
		return hp
	}
	for _, ticks := range []uint64{12, 13, 45, 77, 109, 256, 524, 525, 556, 640, 1200} {
		hp, stage := bodyHealthAt(t, true, -1, ticks)
		if hp != want(ticks) {
			t.Errorf("after %d sub-ticks health %d, want %d", ticks, hp, want(ticks))
		}
		if stage != decayStageFor(hp) {
			t.Errorf("after %d sub-ticks stage %d, health %d names %d", ticks, stage, hp, decayStageFor(hp))
		}
	}
	if hp, _ := bodyHealthAt(t, true, 0, 640); hp != 0 {
		t.Errorf("health 0 became %d, want no loss", hp)
	}
}

// An undying unit below zero is on no ladder and loses nothing.
func TestSecondGameUndyingUnitBelowZeroLosesNothing(t *testing.T) {
	t.Parallel()
	e := Entity{ID: 1, X: 3, Y: 3, HP: -5, MaxHP: 50, HealthRegenPeriod: 25, Undying: true}
	e.SourceBinding.RuntimeID = 5
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{e}, secondGameScript(t, nil, nil, nil), Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for w.Tick() < 640 {
		Step(w, nil)
	}
	if hp := w.Entities()[0].HP; hp != -5 {
		t.Errorf("undying unit at -5 became %d, want -5", hp)
	}
}

// The first game's body loses health at the decay walk's rate alone.
func TestFirstGameBodyLosesNoPoolPassHealth(t *testing.T) {
	t.Parallel()
	hp, _ := bodyHealthAt(t, false, -1, 640)
	if hp != -14 {
		t.Errorf("first-game health %d after 640 sub-ticks, want -14 as before", hp)
	}
}
