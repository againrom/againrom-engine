package terrain

import (
	"math"
	"testing"
)

// The tolerance, and what it is not. See the file comment.
const sunEps = 1e-15

// The model's constants, transcribed a SECOND time from the published reading so
// that the cross-model check below is a comparison of two independent
// transcriptions rather than of the package with itself.
const (
	wantHigh      = 0.78539815
	wantLow       = -0.78539815
	wantDayStep   = 0.0021816615277777777
	wantNightStep = 0.0065449845833333332
)

func closeTo(got, want float64) bool { return math.Abs(got-want) <= sunEps }

// TestSunAngleEndpoints - AC-1: the angle at each band's first and last minute.
//
// The computed values are written to full precision, and this comment used to
// add "so a mutation of either step constant in its last decimal digit still
// fails". IT DOES NOT, and the correction is measured rather than reasoned. A
// last-digit change moves no double, so nothing here or anywhere can see it; and
// the smallest change that DOES move one -- 0.0021816615277777787, some two ulp
// away -- leaves this test, the transcription check and the bounds check all
// green and fails TestSunConstantsAreTheImagesBytes alone. Full precision is
// worth writing, but the bit patterns are what hold the constants down.
func TestSunAngleEndpoints(t *testing.T) {
	tests := []struct {
		minute  uint64
		want    float64
		literal bool
		what    string
	}{
		{0, -0.78539815, true, "day band, first minute -- 06:00"},
		{359, -0.0021816615277778055, false, "day band, one minute before noon"},
		{360, -2.7999999999999999e-17, false, "day band, noon"},
		{719, 0.78321648847222214, false, "day band, last minute"},
		{720, 0.78539815, true, "dusk band, first minute"},
		{959, 0.78539815, true, "dusk band, last minute"},
		{960, 0.78539815, true, "night band, first minute"},
		{1199, -0.7788531654166666, false, "night band, last minute"},
		{1200, -0.78539815, true, "dawn band, first minute"},
		{1439, -0.78539815, true, "dawn band, last minute"},
	}
	for _, tc := range tests {
		got := SunAngle(tc.minute, true)
		if tc.literal {
			if got != tc.want {
				t.Errorf("SunAngle(%d) = %.17g, want exactly %.17g (%s)", tc.minute, got, tc.want, tc.what)
			}
			continue
		}
		if !closeTo(got, tc.want) {
			t.Errorf("SunAngle(%d) = %.17g, want %.17g (%s)", tc.minute, got, tc.want, tc.what)
		}
	}
}

// TestSunBandCensus - AC-2: the four bands take 720, 240, 240 and 240 of a day's
// minutes, 960 minutes carry a computed angle and 480 a literal one, and no
// minute is unassigned.
//
// The band is decided here from the published HOUR RANGES, transcribed
// independently, and never by asking the package which band it took.
func TestSunBandCensus(t *testing.T) {
	const (
		day = iota
		dusk
		night
		dawn
	)
	census := [4]int{}
	for m := uint64(0); m < MinutesPerDay; m++ {
		hour := ((m + 360) / 60) % 24
		switch {
		case hour >= 6 && hour <= 17:
			census[day]++
		case hour >= 18 && hour <= 21:
			census[dusk]++
		case hour >= 2 && hour <= 5:
			census[dawn]++
		case hour == 22 || hour == 23 || hour == 0 || hour == 1:
			census[night]++
		default:
			t.Fatalf("minute %d fell in no band (hour %d)", m, hour)
		}
	}
	if census != [4]int{720, 240, 240, 240} {
		t.Errorf("band census = %v, want [720 240 240 240] (day, dusk, night, dawn)", census)
	}
	if computed := census[day] + census[night]; computed != 960 {
		t.Errorf("computed-angle minutes = %d, want 960", computed)
	}
	if literal := census[dusk] + census[dawn]; literal != 480 {
		t.Errorf("literal-angle minutes = %d, want 480", literal)
	}

	// The same day counted a second way, on the ANGLE rather than on the hour:
	// 241 minutes carry exactly +0.78539815 and 241 exactly -0.78539815. Those
	// are 240 + 1 each and not 240, because both computed arms START on a
	// literal -- the day band at minute 0 and the night band at minute 960. A
	// band boundary off by one hour moves both counts.
	nHigh, nLow := 0, 0
	for m := uint64(0); m < MinutesPerDay; m++ {
		switch SunAngle(m, true) {
		case wantHigh:
			nHigh++
		case wantLow:
			nLow++
		}
	}
	if nHigh != 241 || nLow != 241 {
		t.Errorf("minutes at exactly +0.78539815 / -0.78539815 = %d / %d, want 241 / 241", nHigh, nLow)
	}
}

func TestSunAgreesWithAnIndependentTranscription(t *testing.T) {
	for m := uint64(0); m < MinutesPerDay; m++ {
		hour := ((m + 360) / 60) % 24
		var want float64
		switch {
		case hour >= 6 && hour <= 17:
			want = wantLow + float64(m%720)*wantDayStep
		case hour >= 18 && hour <= 21:
			want = wantHigh
		case hour >= 2 && hour <= 5:
			want = wantLow
		default:
			want = wantHigh - float64(m%240)*wantNightStep
		}
		if got := SunAngle(m, true); !closeTo(got, want) {
			t.Fatalf("SunAngle(%d) = %.17g, want %.17g", m, got, want)
		}
	}
}

// TestSunConstantsAreTheImagesBytes - AC-4: the two limits are the truncated-pi
// quarter and not pi/4, the day step is the negation of the double the image
// holds, 720 steps make a truncated pi/2 rather than pi/2, and the night step is
// three times the day step.
//
// The bit patterns are the strongest assertion in this file: they check the
// constants as the image stores them, not as a decimal a reader might mistype
// into something that rounds the same way.
func TestSunConstantsAreTheImagesBytes(t *testing.T) {
	// THE PACKAGE'S OWN CONSTANTS, not this file's copies of them. Pinning the
	// copies would check the transcription against itself; these four lines are
	// the only place sun.go's four doubles are asserted directly, and they are
	// what a mutation of any digit has to get past.
	for _, tc := range []struct {
		name string
		got  uint64
		want uint64
	}{
		{"sunHigh", math.Float64bits(sunHigh), 0x3fe921fb4d12d84a},
		{"sunLow", math.Float64bits(sunLow), 0xbfe921fb4d12d84a},
		{"the negated day step", math.Float64bits(-daySunStep), 0xbf61df469d353918},
		{"the night step", math.Float64bits(nightSunStep), 0x3f7acee9ebcfd5a4},
	} {
		if tc.got != tc.want {
			t.Errorf("%s is %#016x, want %#016x", tc.name, tc.got, tc.want)
		}
	}

	if got := math.Float64bits(SunAngle(720, true)); got != 0x3fe921fb4d12d84a {
		t.Errorf("the dusk literal is %#016x, want 0x3fe921fb4d12d84a", got)
	}
	if got := math.Float64bits(SunAngle(1200, true)); got != 0xbfe921fb4d12d84a {
		t.Errorf("the dawn literal is %#016x, want 0xbfe921fb4d12d84a", got)
	}
	if got := math.Float64bits(math.Pi / 4); got != 0x3fe921fb54442d18 {
		t.Fatalf("derivation: math.Pi/4 is %#016x, want 0x3fe921fb54442d18", got)
	}
	if SunAngle(720, true) == math.Pi/4 {
		t.Error("the sun limit equals math.Pi/4; it is a TRUNCATED pi quarter and must not")
	}
	if got := math.Float64bits(-wantDayStep); got != 0xbf61df469d353918 {
		t.Errorf("the negated day step is %#016x, want 0xbf61df469d353918", got)
	}

	// 720 day steps span a truncated pi/2, which is 2.7e-8 short of pi/2 -- ten
	// million times the tolerance, so this discriminates.
	span := 720 * wantDayStep
	if !closeTo(span, 1.5707963) {
		t.Errorf("720 day steps = %.17g, want 1.5707963", span)
	}
	if closeTo(span, math.Pi/2) {
		t.Error("720 day steps equal pi/2; the span is a TRUNCATED pi/2 and must not")
	}
	// The night arc is the same span in a third of the minutes.
	if !closeTo(240*wantNightStep, 1.5707963) {
		t.Errorf("240 night steps = %.17g, want 1.5707963", 240*wantNightStep)
	}
	if !closeTo(wantNightStep, 3*wantDayStep) {
		t.Errorf("the night step %.17g is not three times the day step %.17g", wantNightStep, wantDayStep)
	}
}

func TestSunAngleBounds(t *testing.T) {
	for m := uint64(0); m < MinutesPerDay; m++ {
		th := SunAngle(m, true)
		if th < -0.78539815 || th > 0.78539815 {
			t.Fatalf("SunAngle(%d) = %.17g, outside [-0.78539815, +0.78539815]", m, th)
		}
	}
	for m := uint64(1); m < 720; m++ {
		if SunAngle(m, true) <= SunAngle(m-1, true) {
			t.Fatalf("the day arm did not rise at minute %d", m)
		}
	}
	for m := uint64(961); m < 1200; m++ {
		if SunAngle(m, true) >= SunAngle(m-1, true) {
			t.Fatalf("the night arm did not fall at minute %d", m)
		}
	}
	if got := 32 / math.Cos(0.78539815); !closeTo(got, 45.254833389639757) {
		t.Errorf("stepH at the limit = %.17g, want 45.254833389639757", got)
	}
	if got := math.Tan(0.78539815); got > 1.0 {
		t.Errorf("tan at the limit = %.17g, want at most 1.0", got)
	}
}

// TestSunAngleCycleOff - AC-5: with the switch off the angle is the fixed
// literal at every minute of the day, and it is the value light.go already held.
func TestSunAngleCycleOff(t *testing.T) {
	for m := uint64(0); m < MinutesPerDay; m++ {
		if got := SunAngle(m, false); got != 0.78539815 {
			t.Fatalf("SunAngle(%d, off) = %.17g, want exactly 0.78539815", m, got)
		}
	}
	if DefaultTheta != 0.78539815 {
		t.Errorf("DefaultTheta = %.17g, want 0.78539815", DefaultTheta)
	}
}

func TestSunAtComposesSkyLightAndSunAngle(t *testing.T) {
	for _, m := range []uint64{0, 359, 719, 720, 960, 1199, 1200, 1439} {
		off := SunAt(m, false)
		if off != DefaultDaytime {
			t.Errorf("SunAt(%d, off) = %+v, want DefaultDaytime %+v", m, off, DefaultDaytime)
		}

		on := SunAt(m, true)
		want := SkyLight(m, true)
		want.Theta = SunAngle(m, true)
		if on != want {
			t.Errorf("SunAt(%d, on) = %+v, want SkyLight's own %+v with SunAngle's Theta", m, on, want)
		}
		if on.Theta != SunAngle(m, true) {
			t.Errorf("SunAt(%d, on).Theta does not carry SunAngle's answer", m)
		}
	}
}

// TestRelightDue - AC-7: 72 unforced relights per in-game day, on the sub-tick
// that begins minutes 0, 20, 40 ... and on no other sub-tick.
func TestRelightDue(t *testing.T) {
	fired := 0
	var first []uint64
	for s := uint64(0); s < MinutesPerDay*SubTicksPerMinute; s++ {
		if !RelightDue(s) {
			continue
		}
		fired++
		if len(first) < 4 {
			first = append(first, s)
		}
		if s%SubTicksPerMinute != 0 {
			t.Fatalf("a relight fired at sub-tick %d, part-way through a minute", s)
		}
		if ClockMinute(s)%RelightPeriodMinutes != 0 {
			t.Fatalf("a relight fired at minute %d, which is not a multiple of 20", ClockMinute(s))
		}
	}
	if fired != 72 {
		t.Errorf("unforced relights per in-game day = %d, want 72", fired)
	}
	want := []uint64{0, 320, 640, 960}
	for i := range want {
		if i >= len(first) || first[i] != want[i] {
			t.Fatalf("the first four relight sub-ticks are %v, want %v", first, want)
		}
	}
	// Neither half of the gate alone is the gate: a sub-tick one past a relight
	// is in the right minute and fires nothing.
	if RelightDue(1) || RelightDue(321) {
		t.Error("a relight fired on a sub-tick that does not begin a full tick")
	}
	if RelightDue(16) || RelightDue(304) {
		t.Error("a relight fired on a full tick whose minute is not a multiple of 20")
	}
}

func TestClockMinute(t *testing.T) {
	for _, tc := range []struct{ sub, want uint64 }{{0, 0}, {15, 0}, {16, 1}, {959, 59}, {960, 60}} {
		if got := ClockMinute(tc.sub); got != tc.want {
			t.Errorf("ClockMinute(%d) = %d, want %d", tc.sub, got, tc.want)
		}
	}
	if hour := ((ClockMinute(0) + ClockOriginMinute) / 60) % 24; hour != 6 {
		t.Errorf("a run opens at hour %d, want 6", hour)
	}
}

func TestSunAngleIsTotal(t *testing.T) {
	for _, m := range []uint64{0, 1439, 1440, 1441, 1_000_000, math.MaxUint64 - 1, math.MaxUint64} {
		SunAngle(m, true)
		SunAngle(m, false)
	}
	for _, m := range []uint64{0, 137, 719, 720, 960, 1439} {
		for _, k := range []uint64{1, 2, 7919} {
			if got, want := SunAngle(m+k*MinutesPerDay, true), SunAngle(m, true); got != want {
				t.Errorf("SunAngle(%d + %d days) = %.17g, want %.17g", m, k, got, want)
			}
		}
	}
}
