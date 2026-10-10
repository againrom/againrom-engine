package game

import "testing"

func TestTheFacingWheelIsTheSheetsOwnOrderingDoubled(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		dx, dy int
		want   int
	}{
		{"south", 0, 1, 0},
		{"south-west", -1, 1, 2},
		{"west", -1, 0, 4},
		{"north-west", -1, -1, 6},
		{"north", 0, -1, 8},
		{"north-east", 1, -1, 10},
		{"east", 1, 0, 12},
		{"south-east", 1, 1, 14},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EffectFacing(tc.dx, tc.dy); got != tc.want {
				t.Errorf("EffectFacing(%d, %d) = %d, want %d", tc.dx, tc.dy, got, tc.want)
			}
		})
	}
}

// TestTheFacingIsMirrorSymmetricAboutTheNorthSouthAxis is what makes the halving
// bit's fold correct: reflecting a delta east-west reflects its facing about the
// same axis, so the nine stored facings cover the sixteen.
func TestTheFacingIsMirrorSymmetricAboutTheNorthSouthAxis(t *testing.T) {
	t.Parallel()

	for dx := 1; dx <= 12; dx++ {
		for dy := -12; dy <= 12; dy++ {
			right := EffectFacing(dx, dy)
			left := EffectFacing(-dx, dy)
			if want := (16 - right) & 0xf; left != want {
				t.Fatalf("(%d,%d) faces %d and (%d,%d) faces %d, want %d",
					dx, dy, right, -dx, dy, left, want)
			}
		}
	}
}
