package ui

import (
	"testing"
	"time"
)

type fakeSchoolAnimator struct {
	fakeTavernInspectionTown
	paints              int
	closed              bool
	hover, paintedHover int
}

func (s *fakeSchoolAnimator) AtTownSurface() bool { return !s.closed }
func (s *fakeSchoolAnimator) AdvanceTownSurfaceAnimation() {
	s.paints++
	s.paintedHover = s.hover
}

func (s *fakeSchoolAnimator) TownSurfaceHover(cell int) { s.hover = cell }

func TestSchoolPaintDeliversHoverBeforeAnimation(t *testing.T) {
	s := &fakeSchoolAnimator{}
	s.view = TownSurfaceView{Kind: TownSurfaceSchool, SchoolClass: 0, Cells: make([]TownSurfaceCell, 5)}
	for i := range s.view.Cells {
		s.view.Cells[i].Enabled = true
	}
	a := newTestApp(t, appRows(1), nil)
	a.SetTown(s)
	if !a.flow.showTown("") {
		t.Fatal("showTown refused the school")
	}
	r := townSurfaceCellRectFor(s.view, 2)
	a.townCursor, a.hasTownCursor = r.Min.Add(r.Size().Div(2)), true
	for _, tc := range []struct {
		cursor, enabled bool
		want            int
	}{{true, true, 2}, {false, true, -1}, {true, false, -1}} {
		a.hasTownCursor, s.view.Cells[2].Enabled = tc.cursor, tc.enabled
		if _, err := a.composeTownRoom(); err != nil {
			t.Fatal(err)
		}
		if s.paintedHover != tc.want {
			t.Fatalf("cursor=%v enabled=%v: paint received hover %d, want %d", tc.cursor, tc.enabled, s.paintedHover, tc.want)
		}
	}
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
