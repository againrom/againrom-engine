package terrain

import (
	"image/color"
	"testing"
)

func TestStatusBarSourceRows(t *testing.T) {
	for _, tc := range []struct {
		kind    StatusBarKind
		value   int
		r, g, b bool
	}{
		{HealthBar, 24, true, false, false}, {HealthBar, 25, true, true, false},
		{HealthBar, 49, true, true, false}, {HealthBar, 50, false, true, false},
		{ManaBar, 0, false, false, true}, {ManaBar, 101, false, false, true},
	} {
		for y, intensity := range []uint8{128, 255, 192, 128} {
			want := color.RGBA{A: 255}
			if tc.r {
				want.R = intensity
			}
			if tc.g {
				want.G = intensity
			}
			if tc.b {
				want.B = intensity
			}
			if got := statusBarRow(tc.kind, tc.value, 101, y); got != want {
				t.Errorf("kind=%d value=%d row=%d got=%v want=%v", tc.kind, tc.value, y, got, want)
			}
		}
	}
	if statusBarGrey != ([4]uint8{64, 128, 96, 64}) {
		t.Errorf("grey source rows = %v", statusBarGrey)
	}
}
