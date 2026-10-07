package game

import (
	"testing"

	"againrom/pkg/ui"
)

// The original client's own scroll-origin clamp never lets either axis below
// 8 cells (SESS-VIEW-030). This engine's own displaced-mode camera clamp can
// position the live camera one row higher on a jagged top edge; a value
// under 8 written to a SAV shows an impassable edge in the original client
// for one frame on LOAD before its own clamp corrects it (owner report). The
// floor leaves an exact (0, 0) view alone: that pair is this build's own "no
// camera captured" value (originalViewOrigin), which many synthetic test
// documents that never call Layout carry, not a position the original's own
// floor was ever meant to correct.
func TestApplicationCurrentRawFloorsViewOriginToTheOriginalsOwnBound(t *testing.T) {
	view := ui.SaveApplicationState{ViewX: 12, ViewY: 13, Zoom: 1, PeriodUS: 31000}
	app := &SnapshotApplicationState{Version: 1, View: view, Baseline: view,
		Original: OriginalStateData{Wimpy: 7, Formation: 31, Speed: 4, Pressed: -1, ViewX: 12, ViewY: 13}}
	app.View.ViewX, app.View.ViewY = 7, 3
	got, err := applicationCurrentRaw(app)
	if err != nil {
		t.Fatal(err)
	}
	if got.ViewX != 8 || got.ViewY != 8 {
		t.Fatalf("ViewX=%d ViewY=%d, want both floored to 8", got.ViewX, got.ViewY)
	}
	app.View.ViewX, app.View.ViewY = 40, 105
	got, err = applicationCurrentRaw(app)
	if err != nil {
		t.Fatal(err)
	}
	if got.ViewX != 40 || got.ViewY != 105 {
		t.Fatalf("ViewX=%d ViewY=%d, want an above-floor view left unchanged", got.ViewX, got.ViewY)
	}
	app.View.ViewX, app.View.ViewY = 0, 0
	got, err = applicationCurrentRaw(app)
	if err != nil {
		t.Fatal(err)
	}
	if got.ViewX != 0 || got.ViewY != 0 {
		t.Fatalf("ViewX=%d ViewY=%d, want a never-laid-out zero/zero view left unfloored", got.ViewX, got.ViewY)
	}
	app.View.ViewX, app.View.ViewY = 0, 3
	got, err = applicationCurrentRaw(app)
	if err != nil {
		t.Fatal(err)
	}
	if got.ViewX != 8 || got.ViewY != 8 {
		t.Fatalf("ViewX=%d ViewY=%d, want a one-axis-zero pair still floored (only the exact (0,0) pair is exempt)", got.ViewX, got.ViewY)
	}
}
