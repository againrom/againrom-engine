package sim

import "testing"

// restoredPools steps one unit through four mana passes and returns its pools.
func restoredPools(t *testing.T, second, undying bool, hp int32) (int32, int32) {
	t.Helper()
	e := Entity{
		ID: 1, X: 3, Y: 3, HP: hp, MaxHP: 50, HealthRegenPeriod: 25,
		MaxMana: 40, ManaRegenPeriod: 25, Undying: undying,
	}
	e.SourceBinding.RuntimeID = 5
	var s *Script
	if second {
		s = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, Terrain{}, []Entity{e}, s, Relations{}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	for w.Tick() < healthRegenCycle {
		Step(w, nil)
	}
	got := w.Entities()[0]
	return got.HP, got.Mana
}

// R2-ENGINE-1207: nothing is restored at zero health or below.
func TestSecondGameRestoresNothingAtZeroHealthOrBelow(t *testing.T) {
	t.Parallel()
	for _, undying := range []bool{false, true} {
		for _, hp := range []int32{0, -1, -5} {
			gotHP, gotMana := restoredPools(t, true, undying, hp)
			if gotHP > hp || gotMana != 0 {
				t.Errorf("undying %v at health %d: health %d, mana %d after 64 sub-ticks, want no gain", undying, hp, gotHP, gotMana)
			}
		}
	}
}

// A second-game unit above zero health gains both pools, undying or not.
func TestSecondGameRestoresAboveZeroHealth(t *testing.T) {
	t.Parallel()
	for _, undying := range []bool{false, true} {
		gotHP, gotMana := restoredPools(t, true, undying, 1)
		if gotHP != 5 || gotMana != 6 {
			t.Errorf("undying %v at health 1: health %d, mana %d, want 5 and 6", undying, gotHP, gotMana)
		}
	}
}

// The first game restores an undying unit at any health.
func TestFirstGameRestorationGateByHealth(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		undying bool
		hp      int32
		gains   bool
	}{
		{false, 1, true}, {false, 0, false}, {false, -5, false},
		{true, 1, true}, {true, 0, true}, {true, -5, true},
	} {
		gotHP, gotMana := restoredPools(t, false, c.undying, c.hp)
		if gained := gotHP > c.hp && gotMana > 0; gained != c.gains {
			t.Errorf("undying %v at health %d: health %d, mana %d, want gain %v", c.undying, c.hp, gotHP, gotMana, c.gains)
		}
		if !c.gains && (gotHP > c.hp || gotMana != 0) {
			t.Errorf("undying %v at health %d: health %d, mana %d, want no gain", c.undying, c.hp, gotHP, gotMana)
		}
	}
}
