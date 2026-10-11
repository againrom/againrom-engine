package sim

import "testing"

// TestSpellPopupFiguresAreTheCastsAtItsPower holds every power-formed popup
// figure to the cast power, on both sides of a skill and Mind sum of 30. The
// original's record level reads 100 below 30 (TEXT-097); the owner's popup
// states the cast instead (DIV-1882). Loss control: at that level the pair
// and the range differ from the cast's.
func TestSpellPopupFiguresAreTheCastsAtItsPower(t *testing.T) {
	rule := SpellRule{ID: 1, School: 1, MaxRange: 6, DamageMin: 4, DamageMax: 8}
	for _, tc := range []struct{ sum, cast int32 }{
		{0, 0}, {15, 0}, {29, 0}, {30, 0}, {31, 1}, {80, 50},
		{130, 100}, {131, 101}, {285, 255}, {286, 255}, {400, 255},
	} {
		e := Entity{Mind: 30}
		e.Skill[1] = tc.sum - 30
		c := SpellCharacteristicsFor(Rules{}, e, rule)
		lo, hi := SpellDamagePair(Rules{}, rule, tc.cast)
		if c.Power != tc.cast || c.DamageMin != lo || c.DamageMax != hi ||
			c.Range != spellRecordRange(rule, tc.cast) || c.Duration != spellRecordDuration(rule, tc.cast) {
			t.Errorf("skill+mind %d: %+v, want power %d damage %d-%d", tc.sum, c, tc.cast, lo, hi)
		}
	}
	lo, hi := SpellDamagePair(Rules{}, rule, 0)
	if flo, fhi := SpellDamagePair(Rules{}, rule, 100); flo == lo || fhi == hi || spellRecordRange(rule, 100) == spellRecordRange(rule, 0) {
		t.Errorf("loss control: level 100 states %d-%d, the cast %d-%d", flo, fhi, lo, hi)
	}
}

// TestSpellRecordValuesAtThePowerOfTheActor checks range and duration
// against the claim's own formulas (TEXT-096). The power ramps are the
// package's own tables, which the cast path shares.
func TestSpellRecordValuesAtThePowerOfTheActor(t *testing.T) {
	for power := int32(0); power <= spellPowerMax; power++ {
		for _, tc := range []struct {
			rule SpellRule
			want SpellCharacteristics
		}{
			{SpellRule{ID: 1, MaxRange: 7, DamageMin: 4, DamageMax: 9},
				SpellCharacteristics{Range: int64(7 + power/30)}},
			// A zero range stays zero, Teleport adds power/3 to its own base.
			{SpellRule{ID: 4, MaxRange: 0}, SpellCharacteristics{}},
			{SpellRule{ID: 26, MaxRange: 9}, SpellCharacteristics{Range: int64(9 + power/3)}},
			// A row with damage columns keeps its own range.
			{SpellRule{ID: 6, MaxRange: 6, DamageMin: 10, DamageMax: 20, Restorative: true},
				SpellCharacteristics{Range: int64(6 + power/30)}},
			{SpellRule{ID: 20, SpellDuration: 10},
				SpellCharacteristics{Duration: segmentedTicks(power, 10, durationSlow)}},
			{SpellRule{ID: 15, SpellDuration: 1},
				SpellCharacteristics{Duration: segmentedTicks(power, 3, durationFast)}},
			// The area word has no trailing removal tick.
			{SpellRule{ID: 7, Area: true, AreaDuration: 5},
				SpellCharacteristics{Duration: uint16(5<<4 + (int(power)<<4)/10)}},
			{SpellRule{ID: 9, Area: true}, SpellCharacteristics{}},
		} {
			e := Entity{Mind: 30 + power}
			got := SpellCharacteristicsFor(Rules{}, e, tc.rule)
			if got.Range != tc.want.Range || got.Duration != tc.want.Duration {
				t.Fatalf("spell %d power %d: range %d duration %d, want %d and %d",
					tc.rule.ID, power, got.Range, got.Duration, tc.want.Range, tc.want.Duration)
			}
		}
	}
}
