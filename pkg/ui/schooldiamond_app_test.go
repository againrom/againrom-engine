package ui

import (
	"testing"
	"time"
)

type fakeSchoolAnimator struct {
	fakeTavernInspectionTown
	paints int
	closed bool
}

func (s *fakeSchoolAnimator) AtTownSurface() bool { return !s.closed }
func (s *fakeSchoolAnimator) AdvanceTownSurfaceAnimation() {
	s.paints++
	s.view.SchoolDiamondFrame = s.paints
}

func TestSchoolAnimationAdvancesInAppPaintButNotUpdate(t *testing.T) {
	s := &fakeSchoolAnimator{}
	s.view = TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: -1}
	a := newTestApp(t, appRows(1), nil)
	a.SetTown(s)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the school")
	}
	for i := 0; i < 30; i++ {
		a.step(appInput{}, time.Unix(100, int64(i)))
	}
	if s.paints != 0 {
		t.Fatal("client update ticks advanced the paint-driven animation")
	}
	for i := 1; i <= 3; i++ {
		if _, err := a.composeTownRoom(); err != nil {
			t.Fatal(err)
		}
		if s.paints != i {
			t.Fatalf("App paint %d called animator %d times", i, s.paints)
		}
	}
	if _, err := ComposeTownScreen(s, ""); err != nil {
		t.Fatal(err)
	}
	if s.paints != 4 {
		t.Fatal("headless school compositor missed the same paint seam")
	}
	s.closed = true
	_, _ = a.composeTownRoom()
	if s.paints != 4 {
		t.Fatal("composing a closed school advanced its animator")
	}
}
