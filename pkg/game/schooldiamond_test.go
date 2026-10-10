package game

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

func diamondSchool(t *testing.T) (*FrontEnd, *townScreen) {
	t.Helper()
	f := shellFrontEnd()
	schoolArt, err := LoadTownSchoolArt(ROM1TownDescription(), townSchoolSource())
	if err != nil {
		t.Fatal(err)
	}
	f.TownSchoolArt = resolved(schoolArt, nil)
	s := f.townUI
	s.Back()
	s.Choose(2) // the production school door
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlCell, Index: 1}, false)
	return f, s
}

func paintDiamondSchool(t *testing.T, s *townScreen) *image.RGBA {
	t.Helper()
	pix, err := ui.ComposeTownScreen(s, "")
	if err != nil {
		t.Fatal(err)
	}
	return pix
}

func TestSchoolDiamondTrainPaintOrbitAndCompletion(t *testing.T) {
	f, s := diamondSchool(t)
	if schoolDiamondState(s) != (diamondState{}) {
		t.Fatal("entry or skill selection armed the diamond")
	}
	before := paintDiamondSchool(t, s)
	s.townSurfaceButton(0)
	if schoolDiamondState(s) != (diamondState{step: 1}) {
		t.Fatalf("first Train should arm without publishing a frame: %+v", *s.schoolDiamond())
	}
	if f.Town.Gold() != 800 || f.Carried[0].Hero.Skill[2] != 1 {
		t.Fatal("diamond wiring changed the admitted training transaction")
	}
	party, gold := f.Carried[0], f.Town.Gold()
	// This sequence is transcribed from TOWN-381, not calculated with the
	// production transition or inferred from frames that happened to render.
	want := []int{1, 2, 3, 4, 5, 6, 7, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	for paint, frame := range want {
		pix := paintDiamondSchool(t, s)
		if s.schoolDiamond().Frame != frame || s.schoolDiamond().Shown() != (paint < 15) {
			t.Fatalf("paint %d: frame=%d active=%v, want %d/%v", paint+1,
				s.schoolDiamond().Frame, s.schoolDiamond().Shown(), frame, paint < 15)
		}
		wantPixel := before.RGBAAt(240, 100)
		if paint < 15 {
			wantPixel = color.RGBA{R: uint8(frame + 1), G: 23, A: 255}
		}
		if got := pix.RGBAAt(240, 100); got != wantPixel {
			t.Fatalf("paint %d: pixel=%v, want %v", paint+1, got, wantPixel)
		}
	}
	if schoolDiamondState(s) != (diamondState{ready: true}) {
		t.Fatalf("completion must cache hidden frame zero: %+v", *s.schoolDiamond())
	}
	paintDiamondSchool(t, s)
	if schoolDiamondState(s) != (diamondState{ready: true}) ||
		f.Town.Gold() != gold || !reflect.DeepEqual(party, f.Carried[0]) {
		t.Fatal("idle paint advanced state or animation changed the training model")
	}
}

func TestSchoolDiamondSnapshotsAndRawCompositionDoNotAdvance(t *testing.T) {
	_, s := diamondSchool(t)
	s.townSurfaceButton(0)
	paintDiamondSchool(t, s)
	want := schoolDiamondState(s)
	for i := 0; i < 20; i++ {
		view := s.TownSurface()
		ui.TownSurfaceControlAt(view, image.Pt(550, 95))
		ui.ComposeTownSurface(view)
	}
	if schoolDiamondState(s) != want {
		t.Fatalf("a snapshot, hit test, or pure view composition advanced state: %+v != %+v", schoolDiamondState(s), want)
	}
}

func TestSchoolDiamondTrainRetriggersEveryReachablePhaseWithoutReset(t *testing.T) {
	// Paint counts choose reachable ascending, peak and descending states.
	// The next-phase oracle is explicit and independent of advance().
	for _, tc := range []struct{ paints, phase, next int }{
		{0, 0, 1}, {1, 1, 2}, {2, 2, 3}, {3, 3, 4}, {4, 4, 5},
		{5, 5, 6}, {6, 6, 7}, {7, 7, 8}, {8, 8, 8},
		{9, 7, 8}, {10, 6, 7}, {11, 5, 6}, {12, 4, 5},
		{13, 3, 4}, {14, 2, 3}, {15, 1, 2}, {16, 0, 1},
	} {
		f, s := diamondSchool(t)
		f.Town.gold = 100000
		s.townSurfaceButton(0)
		for i := 0; i < tc.paints; i++ {
			paintDiamondSchool(t, s)
		}
		before := schoolDiamondState(s)
		s.townSurfaceButton(0)
		if before.frame != tc.phase || s.schoolDiamond().Frame != tc.phase ||
			s.schoolDiamond().Stepped != before.ready || s.schoolDiamond().Step != 1 {
			t.Fatalf("retrigger at paint %d reset phase/current: before=%+v after=%+v", tc.paints, before, *s.schoolDiamond())
		}
		paintDiamondSchool(t, s)
		if s.schoolDiamond().Frame != tc.next {
			t.Fatalf("retrigger at phase %d: next=%d, want %d", tc.phase, s.schoolDiamond().Frame, tc.next)
		}
	}
}

func TestSchoolDiamondLocalRefusalPreservesIdleAndActiveState(t *testing.T) {
	for _, active := range []bool{false, true} {
		f, s := diamondSchool(t)
		if active {
			s.townSurfaceButton(0)
			for i := 0; i < 10; i++ {
				paintDiamondSchool(t, s)
			}
		}
		f.Town.gold = 0
		want := schoolDiamondState(s)
		s.townSurfaceButton(0)
		if schoolDiamondState(s) != want {
			t.Fatalf("unaffordable Train changed active=%v animation: %+v != %+v", active, schoolDiamondState(s), want)
		}
		s.clearSchoolSelection()
		s.townSurfaceButton(0)
		if schoolDiamondState(s) != want {
			t.Fatal("Train without a selected skill changed the animation")
		}
	}
}

func TestSchoolDiamondPickerLeaveReentryAndNewGameBoundaries(t *testing.T) {
	f, s := diamondSchool(t)
	f.Carried = append(f.Carried, f.Carried[0])
	s.townSurfaceButton(0)
	for i := 0; i < 10; i++ {
		paintDiamondSchool(t, s)
	}
	want := schoolDiamondState(s)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlNext}, false)
	s.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlMode}, false)
	if schoolDiamondState(s) != want || s.shopMemberIndex() != 1 {
		t.Fatal("picker or statistics toggle reset/rearmed the animation")
	}
	paintDiamondSchool(t, s)
	if s.schoolDiamond().Frame != 5 || s.schoolDiamond().Step != -1 {
		t.Fatalf("paint following picker must continue descent: %+v", *s.schoolDiamond())
	}
	s.townSurfaceButton(1)             // Exit
	_, _ = ui.ComposeTownScreen(s, "") // fixture square has no shipped picture
	s.Choose(0)                        // tavern
	want = schoolDiamondState(s)
	paintDiamondSchool(t, s)
	if schoolDiamondState(s) != want {
		t.Fatal("painting another room advanced hidden school state")
	}
	s.Back()
	s.Choose(2)
	if schoolDiamondState(s) != (diamondState{}) {
		t.Fatalf("school reentry did not clear state: %+v", *s.schoolDiamond())
	}
	setSchoolDiamondState(s, want)
	s.resetForNewGame()
	if schoolDiamondState(s) != (diamondState{}) {
		t.Fatal("new game retained the diamond")
	}
}
