package sim

import "testing"

// TestSpellRecordPowerFollowsCastPower holds the popup record's level to the
// cast power up to 255. Only a sum below 30 keeps the byte fold of the
// spellbook, which reads 100 (TEXT-097).
func TestSpellRecordPowerFollowsCastPower(t *testing.T) {
	for _, tc := range []struct{ sum, record, cast int32 }{
		{0, 100, 0}, {29, 100, 0}, {30, 0, 0}, {31, 1, 1}, {80, 50, 50},
		{130, 100, 100}, {131, 101, 101}, {285, 255, 255},
		{286, 255, 255}, {287, 255, 255}, {316, 255, 255}, {400, 255, 255},
	} {
		e := Entity{Mind: 30}
		e.Skill[1] = tc.sum - 30
		rule := SpellRule{ID: 1, School: 1, MaxRange: 6, DamageMin: 4, DamageMax: 8}
		c := SpellCharacteristicsFor(Rules{}, e, rule)
		if c.RecordPower != tc.record || c.Power != tc.cast {
			t.Errorf("skill+mind %d: record power %d, cast power %d, want %d and %d",
				tc.sum, c.RecordPower, c.Power, tc.record, tc.cast)
		}
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
