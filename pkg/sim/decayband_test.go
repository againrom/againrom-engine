package sim

import "testing"

// decayBandHealth steps one body felled at hp for ticks sub-ticks, in the first
// or the second game, and returns its health. The second game's pool-pass loss
// is its own claimed rule and is switched off so the decay walk is measured alone.
func decayBandHealth(t *testing.T, second bool, hp int32, ticks uint64) int32 {
	t.Helper()
	stage := DecayFallen
	if hp <= decayBonesHP {
		stage = DecayBones
	}
	e := Entity{ID: 1, X: 3, Y: 3, HP: hp, MaxHP: 50, HealthRegenPeriod: 25, Decay: stage}
	e.SourceBinding.RuntimeID = 5
	var s *Script
	if second {
		s = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{e}, s, Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.game.regenBodyLoss = false
	for w.Tick() < ticks {
		Step(w, nil)
	}
	return w.Entities()[0].HP
}

// A first-game body above -10 loses one point per 64 sub-ticks (HERO-DECAY-069);
// from -10 down the dead list takes one per 32 in both games (HERO-DEATH-026,
// R2-ENGINE-1765). The second game's corpse ageing never reaches a body above
// -10, whose only loss is the pool pass's. The numbers are literals.
func TestBodyDecayRateByBand(t *testing.T) {
	t.Parallel()
	if got := decayBandHealth(t, true, -1, 1000); got != -1 {
		t.Errorf("second game: health %d after 1000 sub-ticks from -1, want -1 from the walk alone", got)
	}
	for _, second := range []bool{false, true} {
		if second {
			// A second-game body is set at -10 or below to start the walk.
			if got := decayBandHealth(t, true, -10, 300); got != -19 {
				t.Errorf("second=%v: health %d after 300 sub-ticks from -10, want -19", second, got)
			}
			if got := decayBandHealth(t, true, -10, 301); got != -20 {
				t.Errorf("second=%v: health %d after 301 sub-ticks from -10, want -20", second, got)
			}
			continue
		}
		// Falls at ticks 12, 76, ... 460 are eight: -9 after 524 sub-ticks, and
		// the ninth at tick 524 carries it to -10 or below by 525.
		if got := decayBandHealth(t, second, -1, 524); got != -9 {
			t.Errorf("second=%v: health %d after 524 sub-ticks from -1, want -9", second, got)
		}
		if got := decayBandHealth(t, second, -1, 525); got > -10 {
			t.Errorf("second=%v: health %d after 525 sub-ticks from -1, want -10 or below", second, got)
		}
		// From -10 the rate stays one per 32: ten falls at ticks 12, 44, ... 300.
		if got := decayBandHealth(t, second, -10, 300); got != -19 {
			t.Errorf("second=%v: health %d after 300 sub-ticks from -10, want -19", second, got)
		}
		if got := decayBandHealth(t, second, -10, 301); got != -20 {
			t.Errorf("second=%v: health %d after 301 sub-ticks from -10, want -20", second, got)
		}
	}
}
