package data

import "testing"

// TestDamageKindResistanceNarrowsEachSignedContributorModulo256 is the single
// data-to-simulation boundary. Zero, both byte edges, one value past the high
// edge and signed values all keep their low byte; no clamp to 0..100 occurs.
func TestDamageKindResistanceNarrowsEachSignedContributorModulo256(t *testing.T) {
	for _, tc := range []struct {
		in   [5]int32
		want [5]uint8
	}{
		{[5]int32{0, 1, 255, 256, -1}, [5]uint8{0, 1, 255, 0, 255}},
		{[5]int32{-2147483648, 2147483647, -257, -256, 511}, [5]uint8{0, 255, 255, 0, 255}},
	} {
		if got := DamageKindResistance(tc.in); got != tc.want {
			t.Errorf("DamageKindResistance(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
