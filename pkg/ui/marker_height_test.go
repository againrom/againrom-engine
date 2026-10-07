package ui

// The marker's own height, in pkg/ui's world space (SC-5, SC-6, SC-11).
//
// Every fixture reuses mode_test.go's cliffGrid — the same 3x4 raised-cliff
// altitude grid TestTileVerticesAreLiteralScreenPixels et al. already pin,
// MinV == -95 — so the AnchorHeight/dy numbers below are the story's own
// documented formula evaluated by an independent terrain.Project call, never
// by re-deriving through overlayScreenRects itself. No game data is read and
// no window opens.

import (
	"bytes"
	"image"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// cliffProjection is the independent oracle: terrain.Project called directly
// over a COPY of the cliffGrid fixture's own altitude bytes, never over
// v.proj. Its AnchorHeight/MinV are what SC-5's "-AnchorHeight(col,row) -
// proj.MinV" names, and are the ground truth every displaced assertion below
// is checked against.
func cliffProjection() terrain.Projection {
	alts := append([]uint8(nil), cliffAlts...)
	return terrain.Project(alts, cliffW, cliffH)
}

func wantDisplacedRects(cam *camera.Camera, proj terrain.Projection, cells []image.Point, native func(col, row int) []image.Rectangle) []screenRect {
	var want []screenRect
	for _, c := range cells {
		dy := -proj.AnchorHeight(c.X, c.Y) - proj.MinV
		for _, arm := range native(c.X, c.Y) {
			arm = arm.Add(image.Pt(0, dy))
			sx, sy := cam.WorldToScreen(float64(arm.Min.X), float64(arm.Min.Y))
			want = append(want, screenRect{
				X: sx, Y: sy,
				W: float64(arm.Dx()) * cam.Zoom,
				H: float64(arm.Dy()) * cam.Zoom,
			})
		}
	}
	return want
}

// rectsOverlap reports whether two screen rects share any pixel, both taken
// half-open as the culling test elsewhere in this package already treats them.
func rectsOverlap(a, b screenRect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

// identityViewer builds a viewer over g whose camera is the identity
// transform: zoom 1, X=Y=0. Matching the view exactly to the world on both
// axes forces Clamp to centre each axis at zero (world <= view centres at
// (world-view)/2), so screen coordinates equal world coordinates and the
// hand-derived arm numbers in the comments below can be read off directly.
func identityViewer(t *testing.T, g terrain.Grid, viewW, viewH int) *Viewer {
	t.Helper()
	v := newViewer(t, g)
	layoutViewport(v, viewW, viewH)
	cam := v.Camera()
	if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
		t.Fatalf("camera is (%v,%v) at zoom %v, want the identity (0,0) at 1 — pick another view size", cam.X, cam.Y, cam.Zoom)
	}
	return v
}

func TestDisplacedMarkerSharesOffsetBetweenObjectAndUnit(t *testing.T) {
	// cliffCanvasH/cliffW*CellSize matches the world on both axes exactly, so
	// the camera sits at the identity and every screen coordinate below is a
	// world coordinate.
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	cam := v.Camera()

	// The non-negotiable guard: the lift is applied only where v.Mode() itself
	// reports displaced, read at the same point the assertions below rely on
	// it having been applied.
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced — the fixture must select displaced for this test to say anything", v.Mode())
	}

	proj := cliffProjection()
	const col, row = 0, 1
	dy := -proj.AnchorHeight(col, row) - proj.MinV
	if dy != 32 {
		t.Fatalf("fixture drift: -AnchorHeight(%d,%d)-MinV = %d, want 32 (hand-computed: "+
			"AnchorHeight=63, MinV=-95)", col, row, dy)
	}

	cells := []image.Point{{X: col, Y: row}}
	v.SetObjects(true, cells)
	v.SetUnits(true, cells)

	gotObj, gotUnit := v.objectScreenRects(), v.unitScreenRects()
	assertScreenRects(t, gotObj, wantDisplacedRects(cam, proj, cells, nativeArms))
	assertScreenRects(t, gotUnit, wantDisplacedRects(cam, proj, cells, nativeUnitArms))

	// SC-5 stated explicitly, not only implied by the two exact matches above:
	// recover each kind's own applied offset from the FLAT (unshifted) oracle
	// already established elsewhere in this package, and require the two
	// offsets to be the identical value the AnchorHeight/MinV formula gives —
	// there is one anchor rule for both kinds, not two.
	flatObj, flatUnit := wantScreenRects(cam, cells), wantUnitScreenRects(cam, cells)
	if len(gotObj) != len(flatObj) || len(gotUnit) != len(flatUnit) {
		t.Fatalf("rectangle count changed under displacement: objects %d->%d, units %d->%d (P-4: count must be unchanged)",
			len(flatObj), len(gotObj), len(flatUnit), len(gotUnit))
	}
	for i := range gotObj {
		if gotObj[i].X != flatObj[i].X || gotObj[i].W != flatObj[i].W || gotObj[i].H != flatObj[i].H {
			t.Fatalf("object arm %d: displaced %+v, flat %+v — only Y may move (P-4)", i, gotObj[i], flatObj[i])
		}
		if offset := gotObj[i].Y - flatObj[i].Y; offset != float64(dy) {
			t.Fatalf("object arm %d: Y offset = %v, want the AnchorHeight/MinV offset %d", i, offset, dy)
		}
	}
	for i := range gotUnit {
		if gotUnit[i].X != flatUnit[i].X || gotUnit[i].W != flatUnit[i].W || gotUnit[i].H != flatUnit[i].H {
			t.Fatalf("unit arm %d: displaced %+v, flat %+v — only Y may move (P-4)", i, gotUnit[i], flatUnit[i])
		}
		if offset := gotUnit[i].Y - flatUnit[i].Y; offset != float64(dy) {
			t.Fatalf("unit arm %d: Y offset = %v, want the AnchorHeight/MinV offset %d", i, offset, dy)
		}
	}
	if gotObj[0].Y-flatObj[0].Y != gotUnit[0].Y-flatUnit[0].Y {
		t.Fatalf("SC-5: object offset %v != unit offset %v for the same anchor cell (%d,%d) — one anchor rule for both kinds",
			gotObj[0].Y-flatObj[0].Y, gotUnit[0].Y-flatUnit[0].Y, col, row)
	}
}

func TestDisplacedMarkersOverlapResolvedByDrawOrder(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced", v.Mode())
	}

	objCell := image.Point{X: 0, Y: 0}
	unitCell := image.Point{X: 0, Y: 1}
	v.SetObjects(true, []image.Point{objCell})
	v.SetUnits(true, []image.Point{unitCell})

	// Guard the fixture itself: the flat crosses must NOT overlap (so any
	// overlap found below is created by displacement, not already present),
	// and the displaced ones MUST, or the test exercises nothing.
	cam := v.Camera()
	flatObj, flatUnit := wantScreenRects(cam, []image.Point{objCell}), wantUnitScreenRects(cam, []image.Point{unitCell})
	for _, fo := range flatObj {
		for _, fu := range flatUnit {
			if rectsOverlap(fo, fu) {
				t.Fatalf("fixture does not discriminate: flat object %+v already overlaps flat unit %+v", fo, fu)
			}
		}
	}

	gotObj, gotUnit := v.objectScreenRects(), v.unitScreenRects()
	overlap := false
	for _, go_ := range gotObj {
		for _, gu := range gotUnit {
			if rectsOverlap(go_, gu) {
				overlap = true
			}
		}
	}
	if !overlap {
		t.Fatalf("fixture does not discriminate: displaced object %+v and unit %+v never overlap; "+
			"pick cells whose AnchorHeight difference creates one", gotObj, gotUnit)
	}

	passes := v.overlayPasses()
	if len(passes) != 2 {
		t.Fatalf("got %d overlay passes, want 2 (objects then units)", len(passes))
	}
	if passes[0].Color != terrain.MarkerColor || passes[1].Color != terrain.UnitMarkerColor {
		t.Fatalf("pass colours = [0]=%+v [1]=%+v, want objects (%+v) then units (%+v) — "+
			"a newly-created overlap must not reorder the passes",
			passes[0].Color, passes[1].Color, terrain.MarkerColor, terrain.UnitMarkerColor)
	}
}

func TestDisplacedCullRunsAfterOffset(t *testing.T) {
	t.Run("out of view before the offset, in view after", func(t *testing.T) {
		// Cell (0,0): AnchorHeight 31, dy=+64. Flat cross centre (16,16); native
		// arms span world y [10,23) -- entirely above a view starting at y=60.
		// Shifted, the cross centres on y=80 (spans [74,87) and [79,82)),
		// squarely inside a [60,100) window.
		v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, 40)
		cam := v.Camera()
		cam.Pan(0, 60)
		if cam.Y != 60 {
			t.Fatalf("camera Y = %v, want 60 (the pan was clamped away; pick another window)", cam.Y)
		}

		cell := image.Point{X: 0, Y: 0}
		flat := wantScreenRects(cam, []image.Point{cell})
		for _, r := range flat {
			if rectInView(cam, r) {
				t.Fatalf("fixture does not discriminate: the UNSHIFTED arm %+v already meets the view", r)
			}
		}

		v.SetObjects(true, []image.Point{cell})
		got := v.objectScreenRects()
		if len(got) != 2 {
			t.Fatalf("SC-11/P-5: got %d rects, want 2 -- the arm meets the view only after its height "+
				"offset, and culling before the offset would drop it (under-coverage)", len(got))
		}
		for _, r := range got {
			if !rectInView(cam, r) {
				t.Fatalf("returned rect %+v does not actually meet the view; fixture arithmetic is wrong", r)
			}
		}
	})

	t.Run("in view before the offset, out of view after", func(t *testing.T) {
		// Cell (1,1): AnchorHeight 127, dy=-32. Flat cross centre (48,48); native
		// arms span world y [42,55) -- inside a [40,60) window. Shifted, the
		// cross centres on y=16 (spans [10,23) and [15,18)), entirely above it.
		v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, 20)
		cam := v.Camera()
		cam.Pan(0, 40)
		if cam.Y != 40 {
			t.Fatalf("camera Y = %v, want 40 (the pan was clamped away; pick another window)", cam.Y)
		}

		cell := image.Point{X: 1, Y: 1}
		flat := wantScreenRects(cam, []image.Point{cell})
		sawInView := false
		for _, r := range flat {
			if rectInView(cam, r) {
				sawInView = true
			}
		}
		if !sawInView {
			t.Fatalf("fixture does not discriminate: no UNSHIFTED arm %+v meets the view", flat)
		}

		v.SetObjects(true, []image.Point{cell})
		if got := v.objectScreenRects(); len(got) != 0 {
			t.Fatalf("SC-11: got %d rects, want 0 -- the arm met the view only BEFORE its height offset, "+
				"and the offset moves it entirely out; culling before the offset would keep it wrongly (%+v)",
				len(got), got)
		}
	})
}

func TestSetFlatSuppressesMarkerHeightOffset(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("setup: Mode() = %v, want displaced", v.Mode())
	}

	v.SetFlat(true)
	if v.Mode() != ModeFlat {
		t.Fatalf("SetFlat(true): Mode() = %v, want flat", v.Mode())
	}
	if v.proj == nil {
		t.Fatal("SetFlat(true): v.proj is nil -- the projection must still exist; only the mode changed")
	}

	cam := v.Camera()
	cells := []image.Point{{X: 0, Y: 1}} // the sloped cell used above: AnchorHeight 63, would shift by 32 if not suppressed
	v.SetObjects(true, cells)
	v.SetUnits(true, cells)

	assertScreenRects(t, v.objectScreenRects(), wantScreenRects(cam, cells))
	assertScreenRects(t, v.unitScreenRects(), wantUnitScreenRects(cam, cells))
}

func TestOverlayScreenRectsNeverMutatesTheBorrowedAltitudeSlice(t *testing.T) {
	alts := append([]uint8(nil), cliffAlts...)
	original := append([]uint8(nil), alts...)

	g := grid(cliffW, cliffH)
	g.Altitudes = alts

	v := newViewer(t, g)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("setup: Mode() = %v, want displaced", v.Mode())
	}

	cells := []image.Point{{X: 0, Y: 0}, {X: 1, Y: 1}, {X: cliffW - 1, Y: cliffH - 1}}
	v.SetObjects(true, cells)
	v.SetUnits(true, cells)

	_ = v.objectScreenRects()
	_ = v.unitScreenRects()

	if !bytes.Equal(alts, original) {
		t.Fatalf("overlayScreenRects mutated the borrowed altitude slice: got %v, want %v (P-6)", alts, original)
	}
	if !bytes.Equal(v.grid.Altitudes, original) {
		t.Fatalf("overlayScreenRects mutated v.grid.Altitudes: got %v, want %v (P-6)", v.grid.Altitudes, original)
	}
}
