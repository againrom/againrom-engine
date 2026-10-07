package ui

// The view a mission opens with: where it sits, how much it spans, and that
// it happens exactly once (AC-5, AC-6, AC-7, AC-8, AC-9, AC-12).
//
// Every fixture is a synthetic grid. No game data is read and no window opens —
// Layout is called directly, which is what the engine does to a running viewer
// and what App.syncViewerLayout does to a freshly opened one.
//
// NO ASSERTION HERE NAMES THE COUNT AS A LITERAL. Every case below reads it
// from AuthoredStartColumns, so replacing it stays the one edit it is meant
// to be. That rule was written when the number was AUTHORED and a test
// carrying it would have punished overruling it; it survives 0091, which
// decoded the number, for the opposite reason and a better one. A test that
// reads a figure off the constant it is exercising pins nothing about that
// figure, so exactly ONE test states where the number comes from — from
// the decoded viewport rectangle and the native cell size — and every
// other reads the constant (0091 AC-13).

import (
	"image"
	"math"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/menu"
	"againrom/pkg/render/terrain"
)

// bigGrid is a map comfortably larger than any view used here at the authored
// extent, so the clamp never enters an assertion that is about the centring.
func bigGrid() terrain.Grid { return grid(200, 200) }

// bigAltGrid is bigGrid carrying a valid altitude layer with a raised block
// around raisedCell, so the viewer runs displaced and that cell's lift is
// nonzero. It is large for the same reason bigGrid is: a small displaced
// fixture makes the clamp, not the centring, decide the answer.
const raisedBy = 100

var raisedCell = image.Pt(100, 90)

func bigAltGrid() terrain.Grid {
	g := bigGrid()
	alts := make([]uint8, g.Width*g.Height)
	for r := raisedCell.Y; r <= raisedCell.Y+1; r++ {
		for c := raisedCell.X; c <= raisedCell.X+1; c++ {
			alts[r*g.Width+c] = raisedBy
		}
	}
	g.Altitudes = alts
	return g
}

// centredOn reports the screen point the cell's drawn centre lands on.
func centredOn(v *Viewer, cell image.Point) (float64, float64) {
	wx, wy := v.cellWorldCentre(cell)
	return v.cam.WorldToScreen(wx, wy)
}

func assertCentred(t *testing.T, v *Viewer, cell image.Point) {
	t.Helper()
	sx, sy := centredOn(v, cell)
	wantX, wantY := float64(v.cam.ViewW)/2, float64(v.cam.ViewH)/2
	if math.Abs(sx-wantX) > 1e-9 || math.Abs(sy-wantY) > 1e-9 {
		t.Errorf("cell %v draws at (%v,%v), want the view centre (%v,%v)", cell, sx, sy, wantX, wantY)
	}
}

// The opening zoom remains the 1024-arm's authored 27 columns. A wider window
// keeps that zoom and therefore reveals more columns instead of magnifying the
// map back to the old span.
func TestStartViewKeepsBaseZoomAndWideWindowsRevealMore(t *testing.T) {
	cols := AuthoredStartColumns()
	cell := image.Pt(100, 90)

	for _, tc := range []struct {
		name              string
		winW, winH        int
		frameW, viewportW int
	}{
		{"the default 4:3 window", 1024, 768, 1024, 864},
		{"a 1920x1080 window", 1920, 1080, 1366, 1206},
		{"an ultrawide window", 3440, 1440, 1835, 1675},
		{"a narrow tall window", 600, 1000, 1024, 864},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newViewer(t, bigGrid())
			v.SetStartView(cell)
			v.Layout(tc.winW, tc.winH)

			assertCentred(t, v, cell)
			if got := v.FrameSize(); got != image.Pt(tc.frameW, MissionFrameH) {
				t.Fatalf("FrameSize = %v, want (%d,%d)", got, tc.frameW, MissionFrameH)
			}
			if got := v.ViewportSize(); got != image.Pt(tc.viewportW, MissionFrameH) {
				t.Fatalf("ViewportSize = %v, want (%d,%d)", got, tc.viewportW, MissionFrameH)
			}
			if v.cam.Zoom != 1 {
				t.Fatalf("opening zoom = %v, want base 1024-arm zoom 1", v.cam.Zoom)
			}
			spanW := float64(v.cam.ViewW) / v.cam.Zoom / camera.CellSize
			wantSpan := float64(tc.viewportW) / camera.CellSize
			if math.Abs(spanW-wantSpan) > 1e-9 {
				t.Errorf("the view spans %v columns, want %v at unchanged zoom", spanW, wantSpan)
			}
			if tc.viewportW == MissionViewportSize().X && math.Abs(spanW-float64(cols)) > 1e-9 {
				t.Errorf("the base view spans %v columns, want the authored %d", spanW, cols)
			}
		})
	}
}

// The owner keeps the mission's logical height fixed at 768. Window height
// changes presentation scale, not the 24 logical map rows at opening zoom.
func TestStartViewRowsStayAtTheFixedLogicalHeight(t *testing.T) {
	rowsSpanned := func(winW, winH int) float64 {
		v := newViewer(t, bigGrid())
		v.SetStartView(image.Pt(100, 90))
		v.Layout(winW, winH)
		return float64(v.cam.ViewH) / v.cam.Zoom / camera.CellSize
	}

	for _, win := range [][2]int{{1024, 768}, {1920, 1080}, {3440, 1440}, {600, 1000}} {
		if got := rowsSpanned(win[0], win[1]); got != 24 {
			t.Errorf("%dx%d opens on %v logical rows, want 24", win[0], win[1], got)
		}
	}
}

// AC-8: arming before any view size exists is the mission path's own order —
// the map screen is opened before the run loop starts. The start view must take
// effect at the FIRST positive size, and be scaled for that size and not for the
// placeholder the camera was built with.
func TestStartViewSurvivesAnArmingWithNoViewSizeYet(t *testing.T) {
	v := newViewer(t, bigGrid())
	cell := image.Pt(120, 40)

	v.SetStartView(cell)
	v.Layout(0, 0) // what App.syncViewerLayout does before the window is known

	if !v.startArmed {
		t.Fatalf("a non-positive layout consumed the arming")
	}

	v.Layout(1600, 900)
	assertCentred(t, v, cell)

	if v.cam.Zoom != 1 || v.FrameSize() != image.Pt(1366, 768) {
		t.Errorf("first positive layout opened at frame %v zoom %v, want (1366,768) at base zoom 1",
			v.FrameSize(), v.cam.Zoom)
	}
}

func TestStartViewIsAppliedOnceAndThenTheViewIsThePlayers(t *testing.T) {
	v := newViewer(t, bigGrid())
	v.SetStartView(image.Pt(100, 90))
	v.Layout(1024, 768)

	v.cam.Pan(300, 200)
	pannedX, pannedY, pannedZoom := v.cam.X, v.cam.Y, v.cam.Zoom

	for i := 0; i < 20; i++ {
		v.Layout(1024, 768)
	}
	if v.cam.X != pannedX || v.cam.Y != pannedY || v.cam.Zoom != pannedZoom {
		t.Errorf("twenty layouts moved the view from (%v,%v)@%v to (%v,%v)@%v",
			pannedX, pannedY, pannedZoom, v.cam.X, v.cam.Y, v.cam.Zoom)
	}

	// A resize re-clamps, as it always did, but must not re-establish the
	// start view: the zoom is what would give it away.
	v.Layout(1920, 1080)
	if v.cam.Zoom != pannedZoom {
		t.Errorf("a resize rescaled the view to %v, want the player's %v", v.cam.Zoom, pannedZoom)
	}
}

func TestAViewerNeverArmedIsUnmoved(t *testing.T) {
	v := newViewer(t, bigGrid())
	v.Layout(1024, 768)

	if v.cam.X != 0 || v.cam.Y != 0 {
		t.Errorf("an unarmed viewer sits at (%v,%v), want the world origin", v.cam.X, v.cam.Y)
	}
	if v.cam.Zoom != 1 {
		t.Errorf("an unarmed viewer opened at zoom %v, want native 1", v.cam.Zoom)
	}

	for i := 0; i < 10; i++ {
		v.Layout(1024, 768)
	}
	if v.cam.X != 0 || v.cam.Y != 0 || v.cam.Zoom != 1 {
		t.Errorf("ten layouts moved an unarmed viewer to (%v,%v)@%v", v.cam.X, v.cam.Y, v.cam.Zoom)
	}
}

// AC-5: on relief the view must arrive where the cell is DRAWN, carrying the
// same lift every other mark on that cell carries — not where a flat lattice
// would put it. The discriminator is that the two answers differ.
func TestStartViewOnDisplacedGroundFollowsTheDrawnCell(t *testing.T) {
	v := newViewer(t, bigAltGrid())
	if v.Mode() != ModeDisplaced {
		t.Fatalf("the fixture is %v, want displaced", v.Mode())
	}

	// The raised cell, whose lift is nonzero by construction.
	cell := raisedCell
	if lift := -v.proj.AnchorHeight(cell.X, cell.Y) - v.proj.MinV; lift == 0 {
		t.Fatalf("the fixture cell carries no lift; the test would not discriminate")
	}

	v.SetStartView(cell)
	v.Layout(1024, 768)
	assertCentred(t, v, cell)

	// The flat answer for the same cell is a different world point, so the
	// assertion above is not satisfied by both.
	flatY := float64(cell.Y*terrain.CellSize + terrain.CellSize/2)
	if _, drawnY := v.cellWorldCentre(cell); drawnY == flatY {
		t.Fatalf("the drawn centre equals the flat one; the test would not discriminate")
	}
}

func TestStartViewObeysTheExistingBounds(t *testing.T) {
	t.Run("a corner cell keeps the view inside the world", func(t *testing.T) {
		v := newViewer(t, bigGrid())
		v.SetStartView(image.Pt(0, 0))
		v.Layout(1024, 768)

		if v.cam.X != 0 || v.cam.Y != 0 {
			t.Errorf("the view sits at (%v,%v), want the world's own corner", v.cam.X, v.cam.Y)
		}
		r := v.cam.VisibleTiles()
		if r.Col0 < 0 || r.Row0 < 0 || r.Col1 > 200 || r.Row1 > 200 {
			t.Errorf("VisibleTiles %+v escapes the grid", r)
		}
	})

	t.Run("a world smaller than the view is centred on the world", func(t *testing.T) {
		// 4x4 tiles = 128x128 world px; at the authored extent the view is far
		// wider than that on both axes.
		v := newViewer(t, grid(4, 4))
		v.SetStartView(image.Pt(1, 1))
		v.Layout(1024, 768)

		vw := float64(v.cam.ViewW) / v.cam.Zoom
		vh := float64(v.cam.ViewH) / v.cam.Zoom
		if want := (v.cam.WorldW() - vw) / 2; math.Abs(v.cam.X-want) > 1e-9 {
			t.Errorf("X = %v, want the world centred at %v", v.cam.X, want)
		}
		if want := (v.cam.WorldH() - vh) / 2; math.Abs(v.cam.Y-want) > 1e-9 {
			t.Errorf("Y = %v, want the world centred at %v", v.cam.Y, want)
		}
	})

	t.Run("a wider frame does not change the base opening zoom", func(t *testing.T) {
		v := newViewer(t, bigGrid())
		v.SetStartView(image.Pt(100, 90))
		v.Layout(3440, 1440)

		if v.cam.Zoom < camera.ZoomMin || v.cam.Zoom > camera.ZoomMax {
			t.Fatalf("zoom %v escaped [%v,%v]", v.cam.Zoom, camera.ZoomMin, camera.ZoomMax)
		}
		if v.cam.Zoom != 1 {
			t.Errorf("zoom = %v, want base zoom 1", v.cam.Zoom)
		}
		assertCentred(t, v, image.Pt(100, 90))
	})
}

// AC-2: two cells give two views, each on its own cell — the arming is read, not
// a constant.
func TestStartViewFollowsTheCellItWasGiven(t *testing.T) {
	open := func(cell image.Point) *Viewer {
		v := newViewer(t, bigGrid())
		v.SetStartView(cell)
		v.Layout(1024, 768)
		return v
	}

	a, b := open(image.Pt(30, 30)), open(image.Pt(150, 160))
	if a.cam.X == b.cam.X && a.cam.Y == b.cam.Y {
		t.Fatalf("two different start cells produced the same view at (%v,%v)", a.cam.X, a.cam.Y)
	}
	assertCentred(t, a, image.Pt(30, 30))
	assertCentred(t, b, image.Pt(150, 160))
}

func TestAuthoredStartColumnsIsWhatTheApplicationUses(t *testing.T) {
	cols := AuthoredStartColumns()
	if cols <= 0 {
		t.Fatalf("AuthoredStartColumns() = %d, want a positive column count", cols)
	}

	v := newViewer(t, bigGrid())
	v.SetStartView(image.Pt(100, 100))
	v.Layout(DefaultWindowW, DefaultWindowH)

	if want := float64(MissionViewportSize().X) / float64(cols*camera.CellSize); v.cam.Zoom != want {
		t.Errorf("zoom = %v, want %v derived from AuthoredStartColumns()", v.cam.Zoom, want)
	}
}

func TestPathCellCentreIsTheCameraOverTheWorldCentre(t *testing.T) {
	v := newViewer(t, cliffGrid())
	layoutViewport(v, 320, 240)
	v.cam.Pan(17, 11)

	for _, cell := range []image.Point{{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 2, Y: 3}} {
		gotX, gotY := v.pathCellCentre(cell)
		wantX, wantY := v.cam.WorldToScreen(v.cellWorldCentre(cell))
		if gotX != wantX || gotY != wantY {
			t.Errorf("pathCellCentre%v = (%v,%v), want (%v,%v)", cell, gotX, gotY, wantX, wantY)
		}
	}
}

// TestTheStartColumnsAreTheDecodedViewportsOwnSpan is 0091 AC-13, and it is the
// ONE test in this file that names the figure rather than reading it.
//
// It derives the count instead of restating it: the map view's rectangle is the
// screen less the side panel, and the span it stores is that width in native
// cells. Every term comes from somewhere other than the constant under test —
// the screen from the frame the mission composes at, the cell from the camera's
// own native size, and the panel from the decoded rect — so a test that agreed
// with any value the constant took would have to be written differently from
// this one.
//
// THE PANEL'S WIDTH IS THE ONE NUMBER STATED HERE, because it is the term this
// story adds and there is nowhere else in the tree that holds it: the view is
// constructed with the rect (0, 0, screenW - 0xa0, screenH), and 0xa0 is 160.
func TestTheStartColumnsAreTheDecodedViewportsOwnSpan(t *testing.T) {
	const panelW = 0xa0

	if camera.CellSize != 32 {
		t.Fatalf("CellSize = %d; the viewport's span is that width divided by 32", camera.CellSize)
	}
	if want := (MissionFrameW - panelW) / camera.CellSize; AuthoredStartColumns() != want {
		t.Errorf("AuthoredStartColumns() = %d, want %d — the screen less the %d-pixel side panel, "+
			"in native cells", AuthoredStartColumns(), want, panelW)
	}

	// The town family's frame is still 640x480 and is no longer what the
	// mission's span is derived from (1026 B5). Both are stated so a build that
	// moved one of them fails here rather than silently changing the other.
	if menu.FrameW != 640 || menu.FrameH != 480 {
		t.Fatalf("the town family's frame is %dx%d, want 640x480", menu.FrameW, menu.FrameH)
	}
	if MissionFrameW != 1024 || MissionFrameH != 768 {
		t.Fatalf("the mission frame is %dx%d, want the owner's 1024x768", MissionFrameW, MissionFrameH)
	}

	// The ROW span, which the column span alone does not pin: the same rect's
	// height is the screen's own, undiminished by the strip, so at native cells
	// the viewport is that height divided by 32. SESS-VIEW-028 gives 24 at
	// 1024x768.
	if want := MissionFrameH / camera.CellSize; MissionViewportSize().Y/camera.CellSize != want {
		t.Errorf("the viewport spans %d rows, want %d", MissionViewportSize().Y/camera.CellSize, want)
	}
	if got := MissionViewportSize(); got.X != MissionFrameW-panelW || got.Y != MissionFrameH {
		t.Errorf("MissionViewportSize() = %v, want the frame less the %d-pixel strip on the width alone",
			got, panelW)
	}
	if got := MissionPanelRect(); got.Dx() != panelW || got.Dy() != MissionFrameH ||
		got.Max.X != MissionFrameW || got.Min.Y != 0 {
		t.Errorf("MissionPanelRect() = %v, want the frame's own right strip, %d by %d",
			got, panelW, MissionFrameH)
	}

	// And the two OTHER resolutions the same expression yields, so a derivation
	// that happened to give the right answer by some other arithmetic fails
	// here. They are not what this build opens on — nothing selects a
	// resolution — but they are what the same rect gives, and stating them is
	// what makes the divisor and the strip separable from the single answer
	// above.
	for _, tc := range []struct {
		screenW, want int
	}{
		{640, 15},
		{800, 20},
		{1024, 27},
	} {
		if got := (tc.screenW - panelW) / camera.CellSize; got != tc.want {
			t.Errorf("a %d-pixel screen gives %d columns, want %d", tc.screenW, got, tc.want)
		}
	}
}
