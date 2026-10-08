package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

func nativeArms(col, row int) []image.Rectangle {
	cx, cy := 32*col+16, 32*row+16
	return []image.Rectangle{
		image.Rect(cx-6, cy-1, cx+7, cy+2),
		image.Rect(cx-1, cy-6, cx+2, cy+7),
	}
}

// wantScreenRects is SC-6's independent oracle: every arm of every cell placed by
// exactly the transform Draw gives a terrain tile — top-left
// cam.WorldToScreen(arm.Min), size the arm's native size scaled by the exported
// cam.Zoom field (the same field the tile path reads). It applies no culling, so
// callers keep their objects inside the view or assert the culling themselves.
func wantScreenRects(cam *camera.Camera, cells []image.Point) []screenRect {
	var want []screenRect
	for _, c := range cells {
		for _, arm := range nativeArms(c.X, c.Y) {
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

// assertScreenRects compares rect lists with exact float64 equality. Both sides
// evaluate the same camera expressions on the same operands, so the result is
// bit-exact; a tolerance would only let a wrong transform through.
func assertScreenRects(t *testing.T, got, want []screenRect) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d rects, want %d\n got: %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rect %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func rectInView(cam *camera.Camera, r screenRect) bool {
	return r.X < float64(cam.ViewW) && r.Y < float64(cam.ViewH) && r.X+r.W > 0 && r.Y+r.H > 0
}

// overlayViewer builds a synthetic viewer of the given world and view size. No
// window is opened and Draw is never called; the overlay transform is pure.
func overlayViewer(t *testing.T, cols, rows, viewW, viewH int) *Viewer {
	t.Helper()
	v, err := NewViewer("t", grid(cols, rows), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, viewW, viewH)
	return v
}

// TestObjectScreenRects — 0008 SC-6 (AC-4 for the viewer): the object
// overlay places each native marker arm through the same camera transform a
// terrain tile gets, at any pan/zoom; it emits two arms per in-map object,
// horizontal first, in the order the cells were given; it drops a rect that
// lies wholly outside the view and keeps a straddling one whole; it yields
// nothing while the overlay is off or the cell list is empty; and an off-map
// anchor contributes nothing while its in-map neighbours in the same list
// still do.
func TestObjectScreenRects(t *testing.T) {
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

			v.SetObjects(true, cells)

			want := wantScreenRects(cam, cells)
			for i, r := range want {
				if !rectInView(cam, r) {
					t.Fatalf("expected rect %d %+v is off-view here, so the comparison would test the cull "+
						"instead of the transform; pick another cell", i, r)
				}
			}
			assertScreenRects(t, v.objectScreenRects(), want)
		})
	}

	t.Run("two arms per object, horizontal first, in list order", func(t *testing.T) {
		v := overlayViewer(t, 100, 80, 400, 300)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}

		if got, want := nativeArms(2, 1),
			[]image.Rectangle{image.Rect(74, 47, 87, 50), image.Rect(79, 42, 82, 55)}; got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("nativeArms(2,1) = %v, want %v — the test's own FR-6 transcription is wrong", got, want)
		}

		order := []image.Point{{X: 9, Y: 6}, {X: 2, Y: 1}, {X: 5, Y: 4}}
		v.SetObjects(true, order)

		got := v.objectScreenRects()
		if len(got) != 2*len(order) {
			t.Fatalf("got %d rects for %d objects, want %d (two arms each)", len(got), len(order), 2*len(order))
		}
		// At the origin at native zoom the camera is the identity, so the screen
		// rects are the native arms themselves: 13x3 then 3x13 about the centre.
		for i, c := range order {
			cx, cy := float64(32*c.X+16), float64(32*c.Y+16)
			if want := (screenRect{X: cx - 6, Y: cy - 1, W: 13, H: 3}); got[2*i] != want {
				t.Fatalf("object %d (cell %v) arm 0 = %+v, want the horizontal %+v", i, c, got[2*i], want)
			}
			if want := (screenRect{X: cx - 1, Y: cy - 6, W: 3, H: 13}); got[2*i+1] != want {
				t.Fatalf("object %d (cell %v) arm 1 = %+v, want the vertical %+v", i, c, got[2*i+1], want)
			}
		}
	})

	t.Run("wholly outside is culled, straddling is kept whole", func(t *testing.T) {
		// A 78x60 view over a 100x80-cell world at native zoom covers world
		// [0,78) x [0,60). Cell (0,0)'s cross is wholly inside; cell (2,1)'s
		// horizontal arm [74,87) x [47,50) straddles the right edge while its
		// vertical arm [79,82) x [42,55) starts past it; cell (50,40) is far away.
		v := overlayViewer(t, 100, 80, 78, 60)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}
		v.SetObjects(true, []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 50, Y: 40}})

		// Three of the six arms survive: culling everything or nothing fails on
		// the count alone, and the straddling arm keeps its full 13 px width even
		// though only 4 px of it are on screen (the framebuffer does the clip).
		assertScreenRects(t, v.objectScreenRects(), []screenRect{
			{X: 10, Y: 15, W: 13, H: 3},
			{X: 15, Y: 10, W: 3, H: 13},
			{X: 74, Y: 47, W: 13, H: 3},
		})

		// Pan right so the same arm now hangs off the *left* edge — still kept
		// whole, at a negative X — while cell (0,0) leaves the view entirely.
		cam.Pan(85, 0)
		if cam.X != 85 || cam.Y != 0 {
			t.Fatalf("camera at (%v,%v), want (85,0)", cam.X, cam.Y)
		}
		assertScreenRects(t, v.objectScreenRects(), []screenRect{{X: -11, Y: 47, W: 13, H: 3}})
	})

	t.Run("off state and toggle", func(t *testing.T) {
		v := overlayViewer(t, 100, 80, 400, 300)
		on := []image.Point{{X: 5, Y: 4}, {X: 6, Y: 5}}

		v.SetObjects(false, on)
		if got := v.objectScreenRects(); got != nil {
			t.Fatalf("overlay off yielded %d rects (%+v), want nil", len(got), got)
		}
		v.SetObjects(true, nil)
		if got := v.objectScreenRects(); got != nil {
			t.Fatalf("overlay on with a nil cell list yielded %d rects (%+v), want nil", len(got), got)
		}
		v.SetObjects(true, []image.Point{})
		if got := v.objectScreenRects(); got != nil {
			t.Fatalf("overlay on with an empty cell list yielded %d rects (%+v), want nil", len(got), got)
		}

		// The toggle really toggles: cells given after an off state draw again.
		v.SetObjects(true, on)
		assertScreenRects(t, v.objectScreenRects(), wantScreenRects(v.Camera(), on))

		v.SetObjects(false, on)
		if got := v.objectScreenRects(); got != nil {
			t.Fatalf("turning the overlay back off yielded %d rects (%+v), want nil", len(got), got)
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
		v.SetObjects(true, []image.Point{
			{X: -1, Y: -1},           // negative anchor
			inMap[0],                 // the map's first cell
			{X: 10, Y: 5},            // col == cols, one past the right edge
			inMap[1],                 // interior
			{X: 5, Y: 10},            // row == rows, one past the bottom edge
			{X: 1 << 20, Y: 1 << 20}, // far off the map: no geometry, no panic
			inMap[2],                 // the map's last cell
		})

		assertScreenRects(t, v.objectScreenRects(), wantScreenRects(cam, inMap))
	})
}

// ---------------------------------------------------------- the entity picture as it stands

// entityPinGrid is an 8x8 grid whose relief rises three native pixels a column,
// so the lift DIFFERS from cell to cell: a uniform relief would let a glyph
// carrying the wrong cell's lift pass unnoticed.
func entityPinGrid() terrain.Grid {
	g := grid(8, 8)
	alts := make([]uint8, 8*8)
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			alts[row*8+col] = uint8(3 * col)
		}
	}
	g.Altitudes = alts
	return g
}

// entityPinProjection is the independent oracle's projection: terrain.Project
// over a COPY of that grid's own altitude bytes, never the viewer's v.proj.
func entityPinProjection() terrain.Projection {
	g := entityPinGrid()
	return terrain.Project(append([]uint8(nil), g.Altitudes...), g.Width, g.Height)
}

// entityPinEntities are four units on cells of their own: two that resolve to a
// sprite, two that keep the square, three carrying a filled bar and one вЂ” the
// downed unit вЂ” carrying an empty track. Ids are sparse and out of cell order,
// so nothing answering by slice position agrees at any of them.
func entityPinEntities() []MapEntity {
	a, b := entityArtA(), entityArtB()
	return []MapEntity{
		{ID: 9, Cell: image.Pt(1, 2), Art: a, Frame: a.Frames[0], Life: LifeAlive, HP: 100, MaxHP: 100},
		{ID: 4, Cell: image.Pt(3, 4), Art: b, Frame: b.Frames[0], Life: LifeAlive, HP: 40, MaxHP: 100},
		{ID: 7, Cell: image.Pt(5, 1), Life: LifeAlive, HP: 70, MaxHP: 100},
		{ID: 2, Cell: image.Pt(6, 6), Life: LifeDowned, HP: 0, MaxHP: 100},
	}
}

// entityPinViewer holds those four over that grid, with two of them selected вЂ”
// one sprite and one square вЂ” so the mark pass has both kinds in it.
func entityPinViewer(t *testing.T, flat bool) *Viewer {
	t.Helper()
	v, err := NewViewer("pin", entityPinGrid(), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	layoutViewport(v, 600, 500)
	v.SetFlat(flat)
	if displaced := v.Mode() == ModeDisplaced; displaced == flat {
		t.Fatalf("flat=%v left the viewer in mode %v", flat, v.Mode())
	}
	v.SetEntities(entityPinEntities())
	v.sel = selection{9, 7}
	return v
}

// entityPinLift is the vertical shift the shared transform owes one cell: the
// story's own documented -AnchorHeight(col,row)-MinV displaced, zero flat.
func entityPinLift(proj terrain.Projection, flat bool, cell image.Point) int {
	if flat {
		return 0
	}
	return -proj.AnchorHeight(cell.X, cell.Y) - proj.MinV
}

// entityPinPlace is the oracle transform: already-built world-space arms shifted
// by their cell's lift, then WorldToScreen and the camera's zoom вЂ” the two steps
// every placement oracle in this package applies, with the shift ahead of them.
// It culls nothing, so the caller keeps its cells inside the view.
func entityPinPlace(cam *camera.Camera, lift int, arms ...image.Rectangle) []screenRect {
	var out []screenRect
	for _, arm := range arms {
		if arm.Empty() {
			continue
		}
		arm = arm.Add(image.Pt(0, lift))
		sx, sy := cam.WorldToScreen(float64(arm.Min.X), float64(arm.Min.Y))
		out = append(out, screenRect{X: sx, Y: sy, W: float64(arm.Dx()) * cam.Zoom, H: float64(arm.Dy()) * cam.Zoom})
	}
	return out
}

// TestEveryEntityGlyphStandsOnItsOwnCellsMarkerGeometry вЂ” 0047 SC-8
// (AC-9): with the entities standing where a freshly opened map puts them,
// the sprite's ground point is the marker path's own anchor for its cell,
// and the square, the selection mark and the two bar arms are the render
// tier's own glyphs for their cells, each carrying that cell's lift and
// nothing else.
//
// Flat and displaced, at two camera positions, because a wrong lift or a wrong
// origin is invisible at one of the two and a wrong cell mapping is hardest to
// see at the identity camera.
func TestEveryEntityGlyphStandsOnItsOwnCellsMarkerGeometry(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; the geometry below is stated over 32-pixel cells", terrain.CellSize)
	}

	proj := entityPinProjection()
	ents := entityPinEntities()

	// Non-vacuity: displaced must actually displace, and unevenly.
	lifts := map[int]bool{}
	for _, e := range ents {
		lifts[entityPinLift(proj, false, e.Cell)] = true
	}
	if len(lifts) < 2 {
		t.Fatalf("the four fixture cells lift by %d distinct value(s); displaced mode is not distinguishable "+
			"from flat over them", len(lifts))
	}

	for _, mode := range []struct {
		name string
		flat bool
	}{{"displaced", false}, {"flat", true}} {
		for _, pos := range []struct {
			name       string
			zoom       float64
			panX, panY float64
		}{{"origin at native zoom", 1, 0, 0}, {"panned and zoomed in", 2, 20, 20}} {
			t.Run(mode.name+", "+pos.name, func(t *testing.T) {
				v := entityPinViewer(t, mode.flat)
				cam := v.Camera()
				cam.SetZoom(pos.zoom)
				cam.Pan(pos.panX, pos.panY)
				// Where the clamp actually left it is read back rather than
				// assumed — this world is smaller than the view on an axis, so
				// it centres — and both readings must be off the origin, or
				// screen pixels would be world pixels and the transform would
				// be witnessing nothing.
				if cam.Zoom != pos.zoom {
					t.Fatalf("camera zoom = %v, want %v", cam.Zoom, pos.zoom)
				}
				if cam.X == 0 && cam.Y == 0 {
					t.Fatalf("camera sits at the origin at zoom %v; pick another position", pos.zoom)
				}

				w, h := v.grid.Width, v.grid.Height
				lift := func(c image.Point) int { return entityPinLift(proj, mode.flat, c) }

				// The sprites: one placement per resolving entity, in slice
				// order, each standing on the marker path's anchor for its cell.
				sprites, squares, _ := v.entityLayer()
				if len(sprites) != 2 || len(squares) != 2 {
					t.Fatalf("the layer split %d sprite(s) and %d square(s), want 2 and 2", len(sprites), len(squares))
				}
				for i, p := range sprites {
					wx, wy := terrain.MarkerAnchor(p.Cell.X, p.Cell.Y, terrain.CellSize, 0, lift(p.Cell))
					if got := p.Ground(); got != (image.Point{X: wx, Y: wy}) {
						t.Errorf("sprite %d on %v grounds at %v, want the marker path's anchor (%d,%d)",
							i, p.Cell, got, wx, wy)
					}
				}

				// The squares, the marks, and each health bar with its filled
				// interior.
				var wantSquares, wantMarks, wantGrounds, wantFills []screenRect
				for _, e := range ents {
					if e.Art == nil {
						wantSquares = append(wantSquares,
							entityPinPlace(cam, lift(e.Cell),
								terrain.EntityMarkerRects(e.Cell.X, e.Cell.Y, w, h, terrain.CellSize)...)...)
					}
					if e.ID == 9 || e.ID == 7 {
						wantMarks = append(wantMarks,
							entityPinPlace(cam, lift(e.Cell),
								terrain.SelectionMarkerRects(e.Cell.X, e.Cell.Y, w, h, terrain.CellSize)...)...)
					}
					bar, ok := terrain.StatusBarRect(terrain.HealthBar, e.Cell.X, e.Cell.Y, w, h, e.TokenSize, e.Art)
					fill, pool := terrain.StatusBarFill(terrain.HealthBar, bar.Dx(), e.HP, e.MaxHP)
					if !ok || !pool {
						continue
					}
					wantGrounds = append(wantGrounds, entityPinPlace(cam, lift(e.Cell), bar)...)
					wantFills = append(wantFills, entityPinPlace(cam, lift(e.Cell),
						image.Rect(bar.Min.X+4, bar.Min.Y, bar.Min.X+4+fill, bar.Max.Y))...)
				}

				// The counts, so an oracle that quietly built nothing cannot
				// agree with a viewer that quietly drew nothing — and every
				// oracle rect inside the view, so a mismatch below can only be
				// the transform and never the cull.
				if len(wantSquares) != 2 || len(wantMarks) == 0 || len(wantGrounds) != 4 || len(wantFills) != 3 {
					t.Fatalf("the oracle built %d square, %d mark, %d ground and %d fill rect(s); "+
						"want 2, some, 4 and 3", len(wantSquares), len(wantMarks), len(wantGrounds), len(wantFills))
				}
				for _, group := range [][]screenRect{wantSquares, wantMarks, wantGrounds, wantFills} {
					for _, r := range group {
						if !rectInView(cam, r) {
							t.Fatalf("the oracle put %+v outside the view; pick another camera position", r)
						}
					}
				}

				assertScreenRects(t, entitySquares(v), wantSquares)
				assertScreenRects(t, v.selectionScreenRects(), wantMarks)
				grounds, fills := v.healthBarScreenRects()
				assertScreenRects(t, grounds, wantGrounds)
				assertScreenRects(t, fills, wantFills)
			})
		}
	}
}

func TestBlockedOverlay(t *testing.T) {
	cells := []image.Point{{X: 5, Y: 4}, {X: 6, Y: 4}, {X: 5, Y: 5}}

	v := overlayViewer(t, 20, 20, 640, 480)
	if on, n := v.BlockedOverlay(); on || n != 0 {
		t.Fatalf("a fresh viewer reports the tint as %v/%d, want off and empty", on, n)
	}
	if rects := v.blockedScreenRects(); rects != nil {
		t.Errorf("the tint placed %d rects while off", len(rects))
	}

	v.SetBlocked(true, cells)
	if on, n := v.BlockedOverlay(); !on || n != len(cells) {
		t.Errorf("after SetBlocked the tint reports %v/%d, want on and %d", on, n, len(cells))
	}

	got := v.blockedScreenRects()
	if len(got) != len(cells) {
		t.Fatalf("placed %d rects for %d cells", len(got), len(cells))
	}
	// One rect per cell, in the order the cells were given, and each is exactly
	// the cell's own footprint through placeArm — the same arithmetic a marker
	// on that cell takes, which is what stops the two disagreeing about where a
	// cell is.
	for i, cell := range cells {
		arm, ok := terrain.CellFootprint(cell.X, cell.Y, v.grid.Width, v.grid.Height, terrain.CellSize)
		if !ok {
			t.Fatalf("cell %v yielded no rect", cell)
		}
		want, in := v.placeArm(cell, arm)
		if !in {
			t.Fatalf("cell %v was culled", cell)
		}
		if got[i] != want {
			t.Errorf("rect %d = %+v, want %+v", i, got[i], want)
		}
	}

	// An off-map cell contributes nothing while its in-map neighbours still do.
	v.SetBlocked(true, append(append([]image.Point(nil), cells...), image.Pt(100, 100)))
	if n := len(v.blockedScreenRects()); n != len(cells) {
		t.Errorf("an off-map cell placed a rect: %d for %d in-map cells", n, len(cells))
	}

	// Switched off again, with the cells still held: nothing is placed.
	v.SetBlocked(false, cells)
	if rects := v.blockedScreenRects(); rects != nil {
		t.Errorf("the tint placed %d rects after being switched off", len(rects))
	}
}

func TestBlockedPassIsFirstAndOptional(t *testing.T) {
	v := overlayViewer(t, 20, 20, 640, 480)
	v.SetObjects(true, []image.Point{{X: 5, Y: 4}})
	v.SetUnits(true, []image.Point{{X: 6, Y: 4}})

	before := v.overlayPasses()
	if len(before) != 2 {
		t.Fatalf("the two marker passes came out as %d passes", len(before))
	}

	v.SetBlocked(true, []image.Point{{X: 5, Y: 4}, {X: 9, Y: 9}})
	after := v.overlayPasses()
	if len(after) != len(before)+1 {
		t.Fatalf("with the tint on there are %d passes, want %d", len(after), len(before)+1)
	}
	if after[0].Color != terrain.BlockedCellColor {
		t.Errorf("the first pass is %+v, want the tint's colour — a full-cell fill drawn later "+
			"would tint the markers instead of the ground", after[0].Color)
	}
	for i := range before {
		if after[i+1].Color != before[i].Color {
			t.Errorf("pass %d changed colour: the tint was not prepended", i)
		}
	}

	// Off again: the slice is element-for-element what it was.
	v.SetBlocked(false, nil)
	restored := v.overlayPasses()
	if len(restored) != len(before) {
		t.Fatalf("with the tint off there are %d passes, want the pre-story %d", len(restored), len(before))
	}
	for i := range before {
		if restored[i].Color != before[i].Color || len(restored[i].Rects) != len(before[i].Rects) {
			t.Errorf("pass %d is not what it was before the tint was ever set", i)
		}
	}
}
