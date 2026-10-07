package terrain_test

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// TestSkyLightArmsAtFirstAndLastMinute - AC-1: SkyLight at the first and last
// minute of all seven arms, against hand-derived literals. The two rows for
// "day" and "cycle off" are the same five values by hand, independently
// derived, which is what AC-3's own test then checks as a property rather
// than assumes here.
func TestSkyLightArmsAtFirstAndLastMinute(t *testing.T) {
	tests := []struct {
		minute uint64
		cycle  bool
		want   terrain.Light
		what   string
	}{
		{0, false, terrain.Light{Ambient: 14, Range: 32, SkyTint: [3]uint8{0, 0, 0}, ShroudObject: 4, ShroudUnit: 2}, "cycle off, minute 0"},
		{1439, false, terrain.Light{Ambient: 14, Range: 32, SkyTint: [3]uint8{0, 0, 0}, ShroudObject: 4, ShroudUnit: 2}, "cycle off, minute 1439"},

		{0, true, terrain.Light{Ambient: 14, Range: 32, SkyTint: [3]uint8{0, 0, 0}, ShroudObject: 4, ShroudUnit: 2}, "day, first minute"},
		{719, true, terrain.Light{Ambient: 14, Range: 32, SkyTint: [3]uint8{0, 0, 0}, ShroudObject: 4, ShroudUnit: 2}, "day, last minute"},

		{720, true, terrain.Light{Ambient: 14, Range: 32, SkyTint: [3]uint8{0, 0, 0}, ShroudObject: 6, ShroudUnit: 3}, "dusk 1, first minute"},
		{839, true, terrain.Light{Ambient: 23, Range: 21, SkyTint: [3]uint8{23, 0, 7}, ShroudObject: 6, ShroudUnit: 3}, "dusk 1, last minute"},

		{840, true, terrain.Light{Ambient: 24, Range: 20, SkyTint: [3]uint8{24, 0, 8}, ShroudObject: 6, ShroudUnit: 3}, "dusk 2, first minute"},
		{959, true, terrain.Light{Ambient: 31, Range: 9, SkyTint: [3]uint8{1, 11, 47}, ShroudObject: 6, ShroudUnit: 3}, "dusk 2, last minute"},

		{960, true, terrain.Light{Ambient: 32, Range: 8, SkyTint: [3]uint8{0, 12, 48}, ShroudObject: 8, ShroudUnit: 4}, "night, first minute"},
		{1199, true, terrain.Light{Ambient: 32, Range: 8, SkyTint: [3]uint8{0, 12, 48}, ShroudObject: 8, ShroudUnit: 4}, "night, last minute"},

		{1200, true, terrain.Light{Ambient: 32, Range: 8, SkyTint: [3]uint8{0, 12, 48}, ShroudObject: 6, ShroudUnit: 3}, "dawn 1, first minute"},
		{1319, true, terrain.Light{Ambient: 29, Range: 19, SkyTint: [3]uint8{23, 12, 1}, ShroudObject: 6, ShroudUnit: 3}, "dawn 1, last minute"},

		{1320, true, terrain.Light{Ambient: 28, Range: 20, SkyTint: [3]uint8{24, 12, 0}, ShroudObject: 6, ShroudUnit: 3}, "dawn 2, first minute"},
		{1439, true, terrain.Light{Ambient: 15, Range: 31, SkyTint: [3]uint8{1, 1, 0}, ShroudObject: 6, ShroudUnit: 3}, "dawn 2, last minute"},
	}
	for _, tc := range tests {
		got := terrain.SkyLight(tc.minute, tc.cycle)
		if got.Theta != 0 {
			t.Errorf("%s: SkyLight(%d, %v).Theta = %v, want the zero angle (FR-1)",
				tc.what, tc.minute, tc.cycle, got.Theta)
		}
		if got != tc.want {
			t.Errorf("%s: SkyLight(%d, %v) = %+v, want %+v", tc.what, tc.minute, tc.cycle, got, tc.want)
		}
	}
}

// TestSkyLightCycleOffIsConstant - AC-2: with cycle false, SkyLight is the
// cycle-off arm at the nine named minutes -- including two past a day and one
// far past a uint64 day count -- and at every one of the 1440 minutes of a
// day.
func TestSkyLightCycleOffIsConstant(t *testing.T) {
	want := terrain.Light{Ambient: 14, Range: 32, SkyTint: [3]uint8{0, 0, 0}, ShroudObject: 4, ShroudUnit: 2}

	for _, m := range []uint64{0, 1, 359, 360, 719, 720, 1439, 1440, 1_000_000} {
		if got := terrain.SkyLight(m, false); got != want {
			t.Errorf("SkyLight(%d, false) = %+v, want the cycle-off arm %+v", m, got, want)
		}
	}
	for m := uint64(0); m < terrain.MinutesPerDay; m++ {
		if got := terrain.SkyLight(m, false); got != want {
			t.Fatalf("SkyLight(%d, false) = %+v, want %+v", m, got, want)
		}
	}
}

func TestSkyLightDayEqualsCycleOffEqualsDefaultDaytime(t *testing.T) {
	// Every minute of the day band, not only its endpoints -- the arm carries
	// no ramp, so it must be exactly this constant throughout.
	for m := uint64(0); m < 720; m++ {
		day := terrain.SkyLight(m, true)
		off := terrain.SkyLight(m, false)
		if day.SkyTint != off.SkyTint || day.Ambient != off.Ambient || day.Range != off.Range {
			t.Fatalf("minute %d: day arm %+v and cycle-off arm %+v disagree on the five band values",
				m, day, off)
		}
	}

	off := terrain.SkyLight(0, false)
	if off.SkyTint != terrain.DefaultDaytime.SkyTint || off.Ambient != terrain.DefaultDaytime.Ambient ||
		off.Range != terrain.DefaultDaytime.Range {
		t.Errorf("the cycle-off arm %+v disagrees with DefaultDaytime %+v", off, terrain.DefaultDaytime)
	}
}

// TestSkyLightIsContinuousOverADay - AC-4: over all 1440 minutes with the
// cycle on, the largest step in any of R, G, B, ambient and range between one
// minute and the next is 1; every one of the five stays within [0,48]; and the
// two joins night-to-dawn1 and day-to-dusk1 are exact.
func TestSkyLightIsContinuousOverADay(t *testing.T) {
	step := func(a, b uint8) int {
		d := int(a) - int(b)
		if d < 0 {
			d = -d
		}
		return d
	}

	prev := terrain.SkyLight(terrain.MinutesPerDay-1, true) // wraps the sweep across midnight-of-the-array too
	for m := uint64(0); m < terrain.MinutesPerDay; m++ {
		cur := terrain.SkyLight(m, true)

		for _, v := range []uint8{cur.SkyTint[0], cur.SkyTint[1], cur.SkyTint[2], cur.Ambient, cur.Range} {
			if v > 48 {
				t.Fatalf("minute %d: value %d exceeds the [0,48] bound (P-2)", m, v)
			}
		}

		for i, s := range [5]int{
			step(cur.SkyTint[0], prev.SkyTint[0]),
			step(cur.SkyTint[1], prev.SkyTint[1]),
			step(cur.SkyTint[2], prev.SkyTint[2]),
			step(cur.Ambient, prev.Ambient),
			step(cur.Range, prev.Range),
		} {
			if s > 1 {
				names := [5]string{"R", "G", "B", "ambient", "range"}
				t.Fatalf("minute %d: %s stepped by %d from the previous minute (%+v -> %+v)",
					m, names[i], s, prev, cur)
			}
		}
		prev = cur
	}

	// The join is exact on the five band values AC-4 measures; the shroud pair
	// is not one of them and the table names it different on both sides of
	// each join (day/cycle-off carry 4/2, dusk1 carries 6/3; night carries
	// 8/4, dawn1 carries 6/3), so it is deliberately excluded here.
	sameFive := func(a, b terrain.Light) bool {
		return a.SkyTint == b.SkyTint && a.Ambient == b.Ambient && a.Range == b.Range
	}
	if night, dawn1 := terrain.SkyLight(1199, true), terrain.SkyLight(1200, true); !sameFive(night, dawn1) {
		t.Errorf("the night -> dawn1 join is not exact: %+v vs %+v", night, dawn1)
	}
	if day, dusk1 := terrain.SkyLight(719, true), terrain.SkyLight(720, true); !sameFive(day, dusk1) {
		t.Errorf("the day -> dusk1 join is not exact: %+v vs %+v", day, dusk1)
	}
}

func TestSkyLightTintNeverDarkens(t *testing.T) {
	var zero [256][terrain.LevelCount]uint8
	for ch := 0; ch < 256; ch++ {
		for level := 0; level < terrain.LevelCount; level++ {
			zero[ch][level] = terrain.ShadeChannel(uint8(ch), 0, level)
		}
	}

	for m := uint64(0); m < terrain.MinutesPerDay; m++ {
		tint := terrain.SkyLight(m, true).SkyTint
		for ch := 0; ch < 256; ch++ {
			base := &zero[ch]
			for level := 0; level < terrain.LevelCount; level++ {
				for c := 0; c < 3; c++ {
					if lit := terrain.ShadeChannel(uint8(ch), tint[c], level); lit < base[level] {
						t.Fatalf("minute %d tint[%d]=%d: ShadeChannel(%d, %d, %d) = %d, less than "+
							"the zero-tint %d", m, c, tint[c], ch, tint[c], level, lit, base[level])
					}
				}
			}
		}
	}
}

func TestSkyLightGroundLevelAtNoonAndMidnight(t *testing.T) {
	const w, h = 3, 3
	flat := make([]uint8, w*h) // every vertex at height 0: both deltas are zero everywhere

	noon := terrain.LevelGrid(flat, w, h, terrain.SunAt(360, true))
	for i, v := range noon {
		if v != 46 {
			t.Fatalf("noon: vertex[%d] = %d, want 46", i, v)
		}
	}
	midnight := terrain.LevelGrid(flat, w, h, terrain.SunAt(1080, true))
	for i, v := range midnight {
		if v != 64 {
			t.Fatalf("midnight: vertex[%d] = %d, want 64", i, v)
		}
	}

	if got := terrain.ShadeScale(46); got != 1.5625 {
		t.Errorf("ShadeScale(46) = %v, want 1.5625", got)
	}
	if got := terrain.ShadeScale(64); got != 1.0 {
		t.Errorf("ShadeScale(64) = %v, want 1.0", got)
	}
}

// TestSkyLightSpriteRow - AC-7: SpriteRow(SunAt(minute, true)) is 3 at noon
// and 8 at midnight, and over a full day takes exactly the six values 3..8 --
// no fewer (the schedule must actually reach both ends) and no more (it must
// never leave them).
func TestSkyLightSpriteRow(t *testing.T) {
	if got := terrain.SpriteRow(terrain.SunAt(360, true)); got != 3 {
		t.Errorf("SpriteRow at noon (minute 360) = %d, want 3", got)
	}
	if got := terrain.SpriteRow(terrain.SunAt(1080, true)); got != 8 {
		t.Errorf("SpriteRow at midnight (minute 1080) = %d, want 8", got)
	}

	seen := map[int]bool{}
	for m := uint64(0); m < terrain.MinutesPerDay; m++ {
		seen[terrain.SpriteRow(terrain.SunAt(m, true))] = true
	}
	want := map[int]bool{3: true, 4: true, 5: true, 6: true, 7: true, 8: true}
	for r := range want {
		if !seen[r] {
			t.Errorf("row %d never occurred over a day", r)
		}
	}
	for r := range seen {
		if !want[r] {
			t.Errorf("row %d occurred; want only 3..8", r)
		}
	}
}
