package ui

import "testing"

// MENU-076: volume = trunc(-max * ((position - max) / max)^2).
func TestSoundSliderUsesTheSquareLaw(t *testing.T) {
	for _, tc := range []struct{ position, volume int }{{5000, 0}, {2500, -1250}, {0, -5000}, {4000, -200}} {
		if got := soundSliderVolume(tc.position, soundSliderRange); got != tc.volume {
			t.Errorf("position %d: volume %d, want %d", tc.position, got, tc.volume)
		}
	}
	// The original's default volume sits at slider position 3130 by the inverse.
	if back := soundSliderPositionOf(-700, soundSliderRange); back != 3130 {
		t.Errorf("inverse of -700 = %d, want 3130", back)
	}
	// Loss control: a linear law would put the midpoint at half amplitude.
	if mid := soundSliderPercent(2500); mid >= 40 || mid <= 10 {
		t.Errorf("midpoint selects %d%%, the square law gives about 24%%", mid)
	}
	if soundSliderPercent(5000) != 100 || soundSliderPercent(0) != 0 {
		t.Error("slider ends are not full and silent")
	}
}

func TestSoundPercentRoundTripsThroughTheSlider(t *testing.T) {
	for percent := 0; percent <= 100; percent++ {
		if got := soundSliderPercent(soundPercentSlider(percent)); got != percent {
			t.Errorf("%d%% reads back as %d%%", percent, got)
		}
	}
	last := -1
	for position := 0; position <= soundSliderRange; position += 50 {
		if p := soundSliderPercent(position); p < last {
			t.Fatalf("percentage fell from %d to %d at position %d", last, p, position)
		} else {
			last = p
		}
	}
}
