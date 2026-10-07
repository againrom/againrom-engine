package ui

import (
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

func nativeUnitArms(col, row int) []image.Rectangle {
	cx, cy := 32*col+16, 32*row+16
	return []image.Rectangle{
		image.Rect(cx-4, cy, cx+5, cy+1),
		image.Rect(cx, cy-4, cx+1, cy+5),
	}
}

// wantUnitScreenRects is SC-6's independent oracle for the unit overlay:
// every arm of every cell placed by exactly the transform Draw gives a
// terrain tile — top-left cam.WorldToScreen(arm.Min), size the arm's
// native size scaled by the exported cam.Zoom field (the same field the tile
// path reads). It applies no culling, so callers keep their units inside the
// view or assert the culling themselves.
func wantUnitScreenRects(cam *camera.Camera, cells []image.Point) []screenRect {
	var want []screenRect
	for _, c := range cells {
		for _, arm := range nativeUnitArms(c.X, c.Y) {
			sx, sy := cam.WorldToScreen(float64(arm.Min.X), float64(arm.Min.Y))
			want = append(want, screenRect{
				X: sx,
				Y: sy,
				W: float64(arm.Dx()) * cam.Zoom,
				H: float64(arm.Dy()) * cam.Zoom,
			})
		}
	}
	return want
}

// TestUnitScreenRects — 0009 SC-6 (AC-4 for the viewer): the unit overlay
// places each native marker arm through the same camera transform a terrain
// tile gets, at any pan/zoom; it emits two arms per in-map unit, horizontal
// first, in the order the cells were given; it drops a rect that lies wholly
// outside the view and keeps a straddling one whole; it yields nothing while
// the overlay is off or the cell list is empty; an off-map anchor
// contributes nothing while its in-map neighbours in the same list still do;
// and the unit toggle is independent of the object toggle in all four
// combinations.
func TestUnitScreenRects(t *testing.T) {
	// Cells whose crosses stay inside the view at every position below, so a
	// mismatch there can only be the transform, never the cull.
	cells := []image.Point{{X: 5, Y: 4}, {X: 9, Y: 6}, {X: 6, Y: 5}}

	positions := []struct {
		name       string
		zoom       float64
		panX, panY float64
	}{
		{"origin at native zoom", 1, 0, 0},
		{"panned at native zoom", 1, 137, 91},
		{"panned and zoomed in", 2, 137, 91},
		{"panned and zoomed out", 0.5, 137, 91},
		{"panned at a wheel-step zoom", 1.2, 137, 91},
	}

	for _, p := range positions {
		t.Run(p.name, func(t *testing.T) {
			// 100x80 cells is 3200x2560 world px against a 400x300 view, so both
			// axes are larger than the view and Clamp holds the offset in
			// [0, world-view] rather than centring the axis.
			v := overlayViewer(t, 100, 80, 400, 300)
			cam := v.Camera()
			cam.SetZoom(p.zoom)
			cam.Pan(p.panX, p.panY)

			// Assert where the camera actually ended up: every mutator re-clamps,
			// so the position must be read back, not assumed.
			if cam.Zoom != p.zoom {
				t.Fatalf("camera zoom = %v, want %v", cam.Zoom, p.zoom)
			}
			if cam.X != p.panX || cam.Y != p.panY {
				t.Fatalf("camera at (%v,%v), want (%v,%v) — the clamp moved it; pick another position",
					cam.X, cam.Y, p.panX, p.panY)
			}

			v.SetUnits(true, cells)

			want := wantUnitScreenRects(cam, cells)
			for i, r := range want {
				if !rectInView(cam, r) {
					t.Fatalf("expected rect %d %+v is off-view here, so the comparison would test the cull "+
						"instead of the transform; pick another cell", i, r)
				}
			}
			assertScreenRects(t, v.unitScreenRects(), want)
		})
	}

	t.Run("two arms per unit, horizontal first, in list order", func(t *testing.T) {
		v := overlayViewer(t, 100, 80, 400, 300)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}

		if got, want := nativeUnitArms(2, 1),
			[]image.Rectangle{image.Rect(76, 48, 85, 49), image.Rect(80, 44, 81, 53)}; got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("nativeUnitArms(2,1) = %v, want %v — the test's own FR-6 transcription is wrong", got, want)
		}

		order := []image.Point{{X: 9, Y: 6}, {X: 2, Y: 1}, {X: 5, Y: 4}}
		v.SetUnits(true, order)

		got := v.unitScreenRects()
		if len(got) != 2*len(order) {
			t.Fatalf("got %d rects for %d units, want %d (FR-6: at most 2 arms each, both in view here)",
				len(got), len(order), 2*len(order))
		}
		// At the origin at native zoom the camera is the identity, so the screen
		// rects are the native arms themselves: 9x1 then 1x9 about the centre.
		for i, c := range order {
			cx, cy := float64(32*c.X+16), float64(32*c.Y+16)
			if want := (screenRect{X: cx - 4, Y: cy, W: 9, H: 1}); got[2*i] != want {
				t.Fatalf("unit %d (cell %v) arm 0 = %+v, want the FR-6 horizontal %+v", i, c, got[2*i], want)
			}
			if want := (screenRect{X: cx, Y: cy - 4, W: 1, H: 9}); got[2*i+1] != want {
				t.Fatalf("unit %d (cell %v) arm 1 = %+v, want the FR-6 vertical %+v", i, c, got[2*i+1], want)
			}
		}
	})

	t.Run("wholly outside is culled, straddling is kept whole", func(t *testing.T) {
		// A 78x60 view over a 100x80-cell world at native zoom covers world
		// [0,78) x [0,60). Cell (0,0)'s cross is wholly inside; cell (2,1)'s
		// horizontal arm [76,85) x [48,49) straddles the right edge while its
		// vertical arm [80,81) x [44,53) starts past it; cell (50,40) is far away.
		v := overlayViewer(t, 100, 80, 78, 60)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}
		v.SetUnits(true, []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 50, Y: 40}})

		assertScreenRects(t, v.unitScreenRects(), []screenRect{
			{X: 12, Y: 16, W: 9, H: 1},
			{X: 16, Y: 12, W: 1, H: 9},
			{X: 76, Y: 48, W: 9, H: 1},
		})

		// Pan right so the same arm now hangs off the *left* edge — still kept
		// whole, at a negative X — while cell (0,0) leaves the view entirely.
		cam.Pan(84, 0)
		if cam.X != 84 || cam.Y != 0 {
			t.Fatalf("camera at (%v,%v), want (84,0)", cam.X, cam.Y)
		}
		assertScreenRects(t, v.unitScreenRects(), []screenRect{{X: -8, Y: 48, W: 9, H: 1}})
	})

	t.Run("off state and toggle", func(t *testing.T) {
		v := overlayViewer(t, 100, 80, 400, 300)
		on := []image.Point{{X: 5, Y: 4}, {X: 6, Y: 5}}

		v.SetUnits(false, on)
		if got := v.unitScreenRects(); got != nil {
			t.Fatalf("AC-4: unit overlay off yielded %d rects (%+v), want nil", len(got), got)
		}
		v.SetUnits(true, nil)
		if got := v.unitScreenRects(); got != nil {
			t.Fatalf("unit overlay on with a nil cell list yielded %d rects (%+v), want nil", len(got), got)
		}
		v.SetUnits(true, []image.Point{})
		if got := v.unitScreenRects(); got != nil {
			t.Fatalf("unit overlay on with an empty cell list yielded %d rects (%+v), want nil", len(got), got)
		}

		// The toggle really toggles: cells given after an off state draw again.
		v.SetUnits(true, on)
		assertScreenRects(t, v.unitScreenRects(), wantUnitScreenRects(v.Camera(), on))

		v.SetUnits(false, on)
		if got := v.unitScreenRects(); got != nil {
			t.Fatalf("AC-4: turning the unit overlay back off yielded %d rects (%+v), want nil", len(got), got)
		}
	})

	t.Run("off-map anchors contribute nothing", func(t *testing.T) {
		// A 10x10 world is exactly 320x320 world px; matching the view keeps the
		// camera at the origin (Clamp centres an axis only when the world is
		// *smaller* than the view) and every in-map cross visible.
		v := overlayViewer(t, 10, 10, 320, 320)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}

		inMap := []image.Point{{X: 0, Y: 0}, {X: 3, Y: 4}, {X: 9, Y: 9}}
		v.SetUnits(true, []image.Point{
			{X: -1, Y: -1},           // negative anchor
			inMap[0],                 // the map's first cell
			{X: 10, Y: 5},            // col == cols, one past the right edge
			inMap[1],                 // interior
			{X: 5, Y: 10},            // row == rows, one past the bottom edge
			{X: 1 << 20, Y: 1 << 20}, // far off the map: no geometry, no wrap, no panic
			inMap[2],                 // the map's last cell
		})

		assertScreenRects(t, v.unitScreenRects(), wantUnitScreenRects(cam, inMap))
	})

	t.Run("the two overlay toggles are independent", func(t *testing.T) {
		objects := []image.Point{{X: 4, Y: 3}, {X: 7, Y: 6}}
		units := []image.Point{{X: 7, Y: 6}, {X: 2, Y: 1}} // (7,6) holds both

		combos := []struct {
			name                   string
			showObjects, showUnits bool
		}{
			{"neither overlay", false, false},
			{"objects on, units off", true, false},
			{"objects off, units on", false, true},
			{"both overlays on", true, true},
		}

		for _, c := range combos {
			t.Run(c.name, func(t *testing.T) {
				v := overlayViewer(t, 100, 80, 400, 300)
				cam := v.Camera()
				if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
					t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
				}
				v.SetObjects(c.showObjects, objects)
				v.SetUnits(c.showUnits, units)

				gotObjects, gotUnits := v.objectScreenRects(), v.unitScreenRects()

				if c.showObjects {
					assertScreenRects(t, gotObjects, wantScreenRects(cam, objects))
				} else if gotObjects != nil {
					t.Fatalf("FR-4/AC-4: objects off alongside units=%v yielded %d object rects (%+v), want nil",
						c.showUnits, len(gotObjects), gotObjects)
				}

				if c.showUnits {
					assertScreenRects(t, gotUnits, wantUnitScreenRects(cam, units))
				} else if gotUnits != nil {
					t.Fatalf("FR-4/AC-4: units off alongside objects=%v yielded %d unit rects (%+v), want nil",
						c.showObjects, len(gotUnits), gotUnits)
				}
			})
		}
	})
}

// TestOverlayPassOrder — 0009 SC-10: the frame's overlay passes come back
// in draw order, objects first and units second, each carrying its own
// colour and its own rect list. Draw does nothing but iterate this slice, so
// this pins the terrain -> objects -> units order without opening a window.
//
// It is the story's only automated guard against the two passes being swapped.
// The unit cross is a strict subset of the object cross at every scale, so a swap
// would not degrade a coincident unit marker, it would erase it completely —
// indistinguishable from "no unit on this cell", which is the one thing the
// overlay exists to show.
func TestOverlayPassOrder(t *testing.T) {
	if terrain.MarkerColor == terrain.UnitMarkerColor {
		t.Fatal("MarkerColor and UnitMarkerColor are equal, so pass order cannot be observed by colour")
	}
	if want := (color.RGBA{R: 0x00, G: 0xE5, B: 0xFF, A: 0xff}); terrain.UnitMarkerColor != want {
		t.Fatalf("FR-6: UnitMarkerColor = %+v, want the opaque cyan %+v", terrain.UnitMarkerColor, want)
	}

	objects := []image.Point{{X: 4, Y: 3}, {X: 7, Y: 6}}
	units := []image.Point{{X: 7, Y: 6}, {X: 2, Y: 1}} // (7,6) holds both

	newViewer := func(t *testing.T) (*Viewer, *camera.Camera) {
		t.Helper()
		v := overlayViewer(t, 100, 80, 400, 300)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}
		return v, cam
	}

	t.Run("both overlays: objects then units", func(t *testing.T) {
		v, cam := newViewer(t)
		v.SetObjects(true, objects)
		v.SetUnits(true, units)

		passes := v.overlayPasses()
		if len(passes) != 2 {
			t.Fatalf("SC-10: got %d overlay passes with both overlays on, want 2 (objects then units)", len(passes))
		}
		if passes[0].Color != terrain.MarkerColor || passes[1].Color != terrain.UnitMarkerColor {
			t.Fatalf("FR-4/SC-10: pass colours are [0]=%+v [1]=%+v, want [0]=MarkerColor %+v then "+
				"[1]=UnitMarkerColor %+v. Draw iterates this slice, so the units pass MUST come second: "+
				"the unit cross is a strict subset of the object cross, so drawing units first makes every "+
				"coincident unit marker completely invisible — indistinguishable from \"no unit here\".",
				passes[0].Color, passes[1].Color, terrain.MarkerColor, terrain.UnitMarkerColor)
		}
		assertScreenRects(t, passes[0].Rects, wantScreenRects(cam, objects))
		assertScreenRects(t, passes[1].Rects, wantUnitScreenRects(cam, units))
	})

	t.Run("objects only: one pass", func(t *testing.T) {
		v, cam := newViewer(t)
		v.SetObjects(true, objects)
		v.SetUnits(false, units)

		passes := v.overlayPasses()
		if len(passes) != 1 {
			t.Fatalf("SC-10: got %d overlay passes with only the object overlay on, want 1", len(passes))
		}
		if passes[0].Color != terrain.MarkerColor {
			t.Fatalf("FR-4: the sole pass carries %+v, want the object MarkerColor %+v",
				passes[0].Color, terrain.MarkerColor)
		}
		assertScreenRects(t, passes[0].Rects, wantScreenRects(cam, objects))
	})

	t.Run("units only: one pass", func(t *testing.T) {
		v, cam := newViewer(t)
		v.SetObjects(false, objects)
		v.SetUnits(true, units)

		passes := v.overlayPasses()
		if len(passes) != 1 {
			t.Fatalf("SC-10: got %d overlay passes with only the unit overlay on, want 1", len(passes))
		}
		if passes[0].Color != terrain.UnitMarkerColor {
			t.Fatalf("FR-4: the sole pass carries %+v, want the UnitMarkerColor %+v",
				passes[0].Color, terrain.UnitMarkerColor)
		}
		assertScreenRects(t, passes[0].Rects, wantUnitScreenRects(cam, units))
	})

	t.Run("neither overlay: no passes", func(t *testing.T) {
		v, _ := newViewer(t)
		v.SetObjects(false, objects)
		v.SetUnits(false, units)

		if passes := v.overlayPasses(); len(passes) != 0 {
			t.Fatalf("AC-4/SC-10: got %d overlay passes with both overlays off (%+v), want none — "+
				"with the overlay off Draw must do exactly what it did without this story",
				len(passes), passes)
		}
	})

	t.Run("an enabled overlay with no rects contributes no pass", func(t *testing.T) {
		// An empty pass would still occupy an index, so the units pass would move
		// to [1] and a reader of overlayPasses could not tell order from presence.
		v, cam := newViewer(t)
		v.SetObjects(true, nil)
		v.SetUnits(true, units)

		passes := v.overlayPasses()
		if len(passes) != 1 {
			t.Fatalf("SC-10: got %d passes with the object overlay on but empty, want 1 (the units pass only)",
				len(passes))
		}
		if passes[0].Color != terrain.UnitMarkerColor {
			t.Fatalf("SC-10: the sole pass carries %+v, want UnitMarkerColor %+v — an empty object pass "+
				"must be omitted, not emitted", passes[0].Color, terrain.UnitMarkerColor)
		}
		assertScreenRects(t, passes[0].Rects, wantUnitScreenRects(cam, units))
	})
}
