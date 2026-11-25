package sim

import "testing"

// decayBodyAt steps a body of class typeID from health hp through ticks
// sub-ticks and returns its health and stage.
func decayBodyAt(t *testing.T, second bool, typeID, hp int32, ticks uint64) (int32, DecayStage) {
	t.Helper()
	e := Entity{ID: 1, X: 3, Y: 3, HP: hp, MaxHP: 50, TypeID: typeID, Decay: DecayFallen}
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

// The second game's ladder tests are strict: kind 2 below -10, 3 below -20,
// 4 below -40 (R2-ENGINE-1825). The first game's begin at the threshold.
// One sub-tick reads the stage before the walk's first point.
func TestSecondGameCorpseStagesBeginBelowTheirThresholds(t *testing.T) {
	for _, c := range []struct {
		hp            int32
		first, second DecayStage
	}{
		{-10, 2, 1}, {-11, 2, 2}, {-20, 3, 2}, {-21, 3, 3}, {-40, 4, 3}, {-41, 4, 4},
	} {
		if _, got := decayBodyAt(t, false, 10, c.hp, 1); got != c.first {
			t.Errorf("first game health %d: stage %d, want %d", c.hp, got, c.first)
		}
		if _, got := decayBodyAt(t, true, 10, c.hp, 1); got != c.second {
			t.Errorf("second game health %d: stage %d, want %d", c.hp, got, c.second)
		}
	}
}

// A second-game body of class word 0x21..0x24 loses health only at -60 and
// above, so the walk stops it at -61; class 0x25 walks on (R2-ENGINE-1825).
func TestSecondGameHeroBodyStopsAtMinusSixtyOne(t *testing.T) {
	for _, c := range []struct {
		typeID, want int32
	}{
		{0x21, -61}, {0x22, -61}, {0x23, -61}, {0x24, -61}, {0x25, -69},
	} {
		if hp, stage := decayBodyAt(t, true, c.typeID, -59, 32*10); hp != c.want || stage != decayLast {
			t.Errorf("class %#x: health %d stage %d, want %d and %d", c.typeID, hp, stage, c.want, decayLast)
		}
	}
}
