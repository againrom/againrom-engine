package terrain_test

import (
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// The effect layer's three selections (spec AC-6, AC-7, AC-10). Every one of
// them is a pure integer function, so nothing here opens an entry, builds a
// texture or reads a clock.

func effSheet(n, phases, rotation int, flip bool) *terrain.EffectSheet {
	frames := make([]*terrain.EffectFrame, n)
	for i := range frames {
		frames[i] = &terrain.EffectFrame{Width: 2, Height: 2, Pixels: make([]color.RGBA, 4)}
	}
	return &terrain.EffectSheet{Frames: frames, Phases: phases, RotationPhases: rotation, Flip: flip}
}

// TestARotatingSheetIndexesPhasesTimesFacingPlusPhase is AC-6 over all sixteen
// facings: a sheet carrying the halving bit stores nine facings and the draw
// mirrors 9..15 onto 7..1, which is what makes the bit a halving of the sheet
// rather than a flag on the art.
func TestARotatingSheetIndexesPhasesTimesFacingPlusPhase(t *testing.T) {
	t.Parallel()

	// firebolt's own shape: 4 phases over 9 stored facings is 36 frames.
	s := effSheet(36, 4, 16, true)
	for facing := range 16 {
		for phase := range 4 {
			frame, mirror, ok := terrain.SelectEffectFrame(s, facing, phase)
			if !ok {
				t.Fatalf("facing %d phase %d selected nothing from a 36-frame sheet", facing, phase)
			}
			wantFacing, wantMirror := facing, false
			if facing > 8 {
				wantFacing, wantMirror = 16-facing, true
			}
			if want := 4*wantFacing + phase; frame != want || mirror != wantMirror {
				t.Errorf("facing %d phase %d selected frame %d mirror %v, want %d %v",
					facing, phase, frame, mirror, want, wantMirror)
			}
		}
	}
}

func TestASheetOfOneRotationPhaseIsIndexedByPhaseAlone(t *testing.T) {
	t.Parallel()

	s := effSheet(8, 7, 1, false)
	for facing := range 16 {
		frame, mirror, ok := terrain.SelectEffectFrame(s, facing, 5)
		if !ok || frame != 5 || mirror {
			t.Errorf("facing %d selected frame %d mirror %v ok %v, want frame 5 unmirrored",
				facing, frame, mirror, ok)
		}
	}
}

// TestAnUnflippedRotatingSheetTakesTheFacingAsGiven is the halving bit's own
// boundary: without it there is no fold and no mirror.
func TestAnUnflippedRotatingSheetTakesTheFacingAsGiven(t *testing.T) {
	t.Parallel()

	s := effSheet(32, 2, 16, false)
	frame, mirror, ok := terrain.SelectEffectFrame(s, 13, 1)
	if !ok || frame != 2*13+1 || mirror {
		t.Errorf("facing 13 selected frame %d mirror %v ok %v, want %d unmirrored",
			frame, mirror, ok, 2*13+1)
	}
}

func TestEveryRefusedSelectionIsReported(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name          string
		s             *terrain.EffectSheet
		facing, phase int
	}{
		{"a nil sheet", nil, 0, 0},
		{"a sheet with no frames", effSheet(0, 4, 1, false), 0, 0},
		{"a sheet with no phases", effSheet(4, 0, 1, false), 0, 0},
		{"an index past the frames", effSheet(4, 4, 1, false), 0, 9},
		{"a negative phase", effSheet(4, 4, 1, false), 0, -1},
		{"a facing block past the frames", effSheet(8, 4, 16, false), 7, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if frame, mirror, ok := terrain.SelectEffectFrame(tc.s, tc.facing, tc.phase); ok {
				t.Errorf("selected frame %d mirror %v, want a refusal", frame, mirror)
			}
		})
	}
}

// TestThePhaseClockAdvancesOneFrameEveryTwoTicks is AC-7: a projectile is not on
// the unit clock, which advances one frame per tick.
func TestThePhaseClockAdvancesOneFrameEveryTwoTicks(t *testing.T) {
	t.Parallel()

	want := []int{0, 1, 1, 2, 2, 3, 3, 0, 0, 1}
	for age, w := range want {
		got, ok := terrain.EffectPhase(terrain.EffectClockHalf, age, 4)
		if !ok || got != w {
			t.Errorf("at age %d the phase is %d (ok %v), want %d", age, got, ok, w)
		}
	}
}

func TestTheTwoOverrideClocksRunWithNoModulus(t *testing.T) {
	t.Parallel()

	for age := range 21 {
		got, ok := terrain.EffectPhase(terrain.EffectClockDirect, age, 21)
		if !ok || got != age {
			t.Errorf("the direct clock at age %d is %d (ok %v), want %d", age, got, ok, age)
		}
	}
	for age := range 5 {
		got, ok := terrain.EffectPhase(terrain.EffectClockRaw, age, 9)
		if !ok || got != age+1 {
			t.Errorf("the raw clock at age %d is %d (ok %v), want %d", age, got, ok, age+1)
		}
	}
}

// TestASheetWithNoPhasesHasNoPhase is the registry's own default for a row that
// states none, reaching the clock.
func TestASheetWithNoPhasesHasNoPhase(t *testing.T) {
	t.Parallel()

	for _, phases := range []int{0, -1} {
		if _, ok := terrain.EffectPhase(terrain.EffectClockHalf, 3, phases); ok {
			t.Errorf("a sheet of %d phases answered a phase", phases)
		}
	}
}

func TestANegativeAgeNeverLeavesTheSheet(t *testing.T) {
	t.Parallel()

	for age := -8; age < 0; age++ {
		got, ok := terrain.EffectPhase(terrain.EffectClockHalf, age, 5)
		if !ok || got < 0 || got >= 5 {
			t.Errorf("at age %d the phase is %d (ok %v), want one inside [0,5)", age, got, ok)
		}
	}
}

// ---------------------------------------------------------------- the facing

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
			if got := terrain.EffectFacing(tc.dx, tc.dy); got != tc.want {
				t.Errorf("EffectFacing(%d, %d) = %d, want %d", tc.dx, tc.dy, got, tc.want)
			}
		})
	}
}

// TestTheFacingSplitsTheSixteenSectorsEvenly is the reason a sign table was not
// used: a bolt crosses an arbitrary delta, and five cells east by one north is
// an ordinary cast that a sign table would draw 45 degrees off.
func TestTheFacingSplitsTheSixteenSectorsEvenly(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		dx, dy, want int
	}{
		// Five east and one north is 11.3 degrees off east: the sector next to
		// east, and not north-east.
		{5, -1, 11},
		{5, 1, 13},
		{-5, -1, 5},
		// Two east and five north is 21.8 degrees off north, which is the same
		// sector north-north-east covers.
		{2, -5, 9},
		// A zero delta has no direction and answers south rather than refusing.
		{0, 0, 0},
	} {
		if got := terrain.EffectFacing(tc.dx, tc.dy); got != tc.want {
			t.Errorf("EffectFacing(%d, %d) = %d, want %d", tc.dx, tc.dy, got, tc.want)
		}
	}

	// Every delta lands inside the wheel, at any magnitude and either sign.
	for dx := -20; dx <= 20; dx++ {
		for dy := -20; dy <= 20; dy++ {
			if got := terrain.EffectFacing(dx, dy); got < 0 || got > 15 {
				t.Fatalf("EffectFacing(%d, %d) = %d, outside [0,16)", dx, dy, got)
			}
		}
	}
}

// TestTheFacingIsMirrorSymmetricAboutTheNorthSouthAxis is what makes the halving
// bit's fold correct: reflecting a delta east-west reflects its facing about the
// same axis, so the nine stored facings cover the sixteen.
func TestTheFacingIsMirrorSymmetricAboutTheNorthSouthAxis(t *testing.T) {
	t.Parallel()

	for dx := 1; dx <= 12; dx++ {
		for dy := -12; dy <= 12; dy++ {
			right := terrain.EffectFacing(dx, dy)
			left := terrain.EffectFacing(-dx, dy)
			if want := (16 - right) & 0xf; left != want {
				t.Fatalf("(%d,%d) faces %d and (%d,%d) faces %d, want %d",
					dx, dy, right, -dx, dy, left, want)
			}
		}
	}
}

// ---------------------------------------------------------------- the frame

// TestAFrameOfNoAreaStillYieldsADrawableImage is EffectFrame.RGBA's own rule:
// the engine refuses an image with no pixels, and a caller that had to test for
// that would test for it at every draw.
func TestAFrameOfNoAreaStillYieldsADrawableImage(t *testing.T) {
	t.Parallel()

	for _, f := range []*terrain.EffectFrame{nil, {}, {Width: 4, Height: 0}} {
		if got := f.RGBA().Bounds(); got.Dx() < 1 || got.Dy() < 1 {
			t.Errorf("a frame of no area yielded a %v image", got)
		}
	}
}

// TestAFramesPixelsReachTheImageInRowMajorOrder is the one thing the drawing
// tier reads off a frame, checked at both ends of the grid.
func TestAFramesPixelsReachTheImageInRowMajorOrder(t *testing.T) {
	t.Parallel()

	f := &terrain.EffectFrame{Width: 2, Height: 2, Pixels: []color.RGBA{
		{R: 1, A: 1}, {G: 2, A: 2}, {B: 3, A: 3}, {R: 4, G: 4, B: 4, A: 4},
	}}
	pic := f.RGBA()
	if got := pic.RGBAAt(0, 0); got != (color.RGBA{R: 1, A: 1}) {
		t.Errorf("pixel (0,0) is %+v", got)
	}
	if got := pic.RGBAAt(1, 0); got != (color.RGBA{G: 2, A: 2}) {
		t.Errorf("pixel (1,0) is %+v", got)
	}
	if got := pic.RGBAAt(1, 1); got != (color.RGBA{R: 4, G: 4, B: 4, A: 4}) {
		t.Errorf("pixel (1,1) is %+v", got)
	}
}

// TestASheetAnswersNilForAnIndexItDoesNotHold is the bundle's own two lookups,
// both of which are total on a nil receiver.
func TestASheetAnswersNilForAnIndexItDoesNotHold(t *testing.T) {
	t.Parallel()

	s := effSheet(3, 3, 1, false)
	if s.Frame(0) == nil || s.Frame(2) == nil {
		t.Error("a sheet refused an index it holds")
	}
	for _, i := range []int{-1, 3, 99} {
		if s.Frame(i) != nil {
			t.Errorf("a 3-frame sheet answered index %d", i)
		}
	}
	if (*terrain.EffectSheet)(nil).Frame(0) != nil {
		t.Error("a nil sheet answered a frame")
	}
	if (*terrain.EffectSet)(nil).Sheet(10) != nil {
		t.Error("a nil set answered a sheet")
	}
	set := &terrain.EffectSet{Sheets: map[int]*terrain.EffectSheet{10: s}}
	if set.Sheet(10) != s || set.Sheet(11) != nil {
		t.Error("the set is not keyed by picture id")
	}
}
