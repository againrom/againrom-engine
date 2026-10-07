package ui

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

func nativeEntitySquare(col, row int) []image.Rectangle {
	cx, cy := 32*col+16, 32*row+16
	return []image.Rectangle{image.Rect(cx-5, cy-5, cx+5, cy+5)}
}

// wantEntityScreenRects is the square oracle: every cell's square placed
// by exactly the transform Draw gives a terrain tile — top-left
// cam.WorldToScreen(square.Min), size the native size scaled by the exported
// cam.Zoom field (the same field the tile path reads). It applies no culling, so
// callers keep their entities inside the view or assert the culling themselves.
func wantEntityScreenRects(cam *camera.Camera, cells []image.Point) []screenRect {
	var want []screenRect
	for _, c := range cells {
		for _, sq := range nativeEntitySquare(c.X, c.Y) {
			sx, sy := cam.WorldToScreen(float64(sq.Min.X), float64(sq.Min.Y))
			want = append(want, screenRect{
				X: sx,
				Y: sy,
				W: float64(sq.Dx()) * cam.Zoom,
				H: float64(sq.Dy()) * cam.Zoom,
			})
		}
	}
	return want
}

// artless is how this file spells 0020's cell-only push through the setter
// that replaced it: one MapEntity per cell, every Art nil, in cell order —
// exactly the value pkg/game's push hands over at the T4 boundary.
func artless(cells []image.Point) []MapEntity {
	ents := make([]MapEntity, len(cells))
	for i, c := range cells {
		ents[i] = MapEntity{Cell: c}
	}
	return ents
}

// entitySquares is the square pass's rectangles as this file reads them: the
// rects of the pass carrying the entity marker colour, nil when no such pass
// was built. Reading through overlayPasses rather than a private accessor is
// deliberate — the pass slice is the draw contract, so every transform
// assertion below holds for exactly what Draw walks.
func entitySquares(v *Viewer) []screenRect {
	for _, p := range v.overlayPasses() {
		if p.Color == terrain.EntityMarkerColor {
			return p.Rects
		}
	}
	return nil
}

// unitArt hand-assembles one drawable unit class: the class canvas and
// ground-touching pixel beside a one-frame sheet of the given size, every
// frame pixel opaque. Only the geometry and the frame's pointer identity
// reach anything under test here.
func unitArt(canvasW, canvasH, cx, cy, frameW, frameH int) *terrain.UnitClass {
	f := &terrain.StaticFrame{Width: frameW, Height: frameH, Pixels: make([]terrain.StaticPixel, frameW*frameH)}
	for i := range f.Pixels {
		f.Pixels[i] = terrain.StaticPixel{Index: 5, Opaque: true}
	}
	return &terrain.UnitClass{Width: canvasW, Height: canvasH, CenterX: cx, CenterY: cy,
		Frames: []*terrain.StaticFrame{f}}
}

// withFrame is how this file spells a resolved entity at the T4 boundary: the
// class beside its own Frames[0], unmirrored — exactly the value pkg/game's
// interim push hands over (0024 T4; T5 puts real selection behind it).
func withFrame(cell image.Point, art *terrain.UnitClass) MapEntity {
	return MapEntity{Cell: cell, Art: art, Frame: art.Frames[0]}
}

// The two drawable art values the sprite fixtures use, and the anchor each
// one owes by the spec's formula (every /2 truncating):
//
//	artA: canvas 64x64, centre (32,60), frame 10x6
//	      anchorX = (32-32) + 5 = 5      anchorY = (60-32) + 3 = 31
//	artB: canvas 31x31, centre (15,29), frame  4x4
//	      anchorX = (15-15) + 2 = 2      anchorY = (29-15) + 2 = 16
//
// The canvases and centres are deliberately unequal to the frames, so an
// anchor read off the frame alone, or the canvas alone, moves every literal
// below. They are the object bundle's own worked examples (statics_test.go),
// so the placement table there cross-checks the arithmetic here.
func entityArtA() *terrain.UnitClass { return unitArt(64, 64, 32, 60, 10, 6) }
func entityArtB() *terrain.UnitClass { return unitArt(31, 31, 15, 29, 4, 4) }

// TestEntityLayerRoutesSpriteOrSquareInSliceOrder — SC-5 (AC-6): the layer
// builder walks the entities once in slice order — ascending id, the
// world's own — routes UnitPlace-ok to the sprite list and everything else
// to the square list, and drops an off-map entity from BOTH lists on the one
// bounds test, art or no art. Nothing is added, dropped or reordered beyond
// that: three resolved in, three placements out; two unresolved in, two
// cells out.
//
// The placements are asserted against hand-transcribed spec-formula literals,
// displaced AND flat, so the lift terms — AnchorHeight and MinV displaced, 0
// and 0 flat, handed to UnitPlace unnegated — are pinned here and not merely
// exercised. The lifts are read off the cliff fixture's own altitude table:
// AnchorHeight(0,1) = (0+127+0+127)/4 = 63, AnchorHeight(2,0) =
// (0+0+127+127)/4 = 63, AnchorHeight(1,1) = 127, MinV = -95.
//
//	destX = col*32 + 16 - anchorX, destY = row*32 + 16 - anchorY - lift - originY
//	(0,1) artA displaced: destX = 16-5 = 11,  destY = 48-31-63+95  = 49; flat 17
//	(2,0) artB displaced: destX = 80-2 = 78,  destY = 16-16-63+95  = 32; flat  0
//	(1,1) artA displaced: destX = 48-5 = 43,  destY = 48-31-127+95 = -15; flat 17
func TestEntityLayerRoutesSpriteOrSquareInSliceOrder(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every literal in this file is stated over 32-pixel cells", terrain.CellSize)
	}

	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced — the fixture must select displaced for the lift half to say anything", v.Mode())
	}

	artA, artB := entityArtA(), entityArtB()
	noFrame := &terrain.UnitClass{Width: 8, Height: 8, CenterX: 4, CenterY: 7}
	v.SetEntities([]MapEntity{
		withFrame(image.Pt(0, 1), artA),      // id 0: resolved
		{Cell: image.Pt(1, 2)},               // id 1: nil art — an id naming no class
		withFrame(image.Pt(2, 0), artB),      // id 2: resolved
		withFrame(image.Pt(-1, 0), artA),     // id 3: off-map WITH art — neither list
		{Cell: image.Pt(0, 3), Art: noFrame}, // id 4: a resolved class with no frame
		{Cell: image.Pt(3, 4)},               // id 5: off-map artless — neither list
		withFrame(image.Pt(1, 1), artA),      // id 6: resolved, sharing artA's frame
	})

	type wantPlace struct {
		cell, topLeft, anchor image.Point
		frame                 *terrain.StaticFrame
	}
	check := func(t *testing.T, mode string, sprites []terrain.StaticPlacement, squares []MapEntity, want []wantPlace) {
		t.Helper()
		if len(sprites) != len(want) {
			t.Fatalf("%s: %d sprite placements, want %d — one per resolved entity, nothing else (AC-6)",
				mode, len(sprites), len(want))
		}
		for i, w := range want {
			g := sprites[i]
			if g.Cell != w.cell || g.TopLeft != w.topLeft || g.Anchor != w.anchor || g.Frame != w.frame {
				t.Errorf("%s sprite %d: cell %v topLeft %v anchor %v frame %p, want cell %v topLeft %v anchor %v frame %p",
					mode, i, g.Cell, g.TopLeft, g.Anchor, g.Frame, w.cell, w.topLeft, w.anchor, w.frame)
			}
		}
		wantSquares := []image.Point{{X: 1, Y: 2}, {X: 0, Y: 3}}
		if len(squares) != len(wantSquares) {
			t.Fatalf("%s: %d square entities %v, want %d — one per unresolved in-map entity, in slice order",
				mode, len(squares), cellsOf(squares), len(wantSquares))
		}
		for i, w := range wantSquares {
			if squares[i].Cell != w {
				t.Errorf("%s square %d = %v, want %v (ascending id, the slice's own order)",
					mode, i, squares[i].Cell, w)
			}
		}
	}

	sprites, squares, _ := v.entityLayer()
	check(t, "displaced", sprites, squares, []wantPlace{
		{image.Pt(0, 1), image.Pt(11, 49), image.Pt(5, 31), artA.Frames[0]},
		{image.Pt(2, 0), image.Pt(78, 32), image.Pt(2, 16), artB.Frames[0]},
		{image.Pt(1, 1), image.Pt(43, -15), image.Pt(5, 31), artA.Frames[0]},
	})

	// The flat half of the lift rule: 0 and 0, over the SAME still-valid
	// altitude grid — the guard is Mode(), not the projection's existence.
	v.SetFlat(true)
	if v.Mode() != ModeFlat {
		t.Fatalf("SetFlat(true): Mode() = %v, want flat", v.Mode())
	}
	sprites, squares, _ = v.entityLayer()
	check(t, "flat", sprites, squares, []wantPlace{
		{image.Pt(0, 1), image.Pt(11, 17), image.Pt(5, 31), artA.Frames[0]},
		{image.Pt(2, 0), image.Pt(78, 0), image.Pt(2, 16), artB.Frames[0]},
		{image.Pt(1, 1), image.Pt(43, 17), image.Pt(5, 31), artA.Frames[0]},
	})
}

func TestTheContentBandCarriesSpritesAndThePassSliceTheSquares(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	cam := v.Camera()
	artA, artB := entityArtA(), entityArtB()
	v.SetEntities([]MapEntity{
		withFrame(image.Pt(0, 1), artA), // sprite at (11,49)
		{Cell: image.Pt(1, 2)},          // square
		withFrame(image.Pt(2, 0), artB), // sprite at (78,32)
		{Cell: image.Pt(0, 3)},          // square
		withFrame(image.Pt(1, 1), artA), // sprite at (43,-15): wholly above the window, culled
	})

	passes := v.overlayPasses()
	if len(passes) != 1 {
		t.Fatalf("got %d passes with no diagnostics on, want 1 — the squares alone (0068 DD-5)", len(passes))
	}

	// Hand-placed at the identity camera: the placement table of the routing
	// test, minus the culled third entry — its rect [43,-15)+6 ends at -9,
	// wholly above a view starting at 0. This viewer holds no bundle, so the
	// content band is the entity sprites alone — in ROW order since 0068, so the
	// cell-(2,0) sprite precedes the cell-(0,1) one although its id is higher.
	sprites := v.planeSprites()
	wantSprites := []staticScreenRect{
		{screenRect: screenRect{X: 78, Y: 32, W: 4, H: 4}, order: artOrder{phase: 4, cell: image.Pt(2, 0), rank: 3, index: 1, part: 1}, Frame: artB.Frames[0], Inspection: InspectionSubject{Kind: InspectionUnit, ID: 0}},
		{screenRect: screenRect{X: 11, Y: 49, W: 10, H: 6}, order: artOrder{phase: 4, cell: image.Pt(0, 1), rank: 3, part: 1}, Frame: artA.Frames[0], Inspection: InspectionSubject{Kind: InspectionUnit, ID: 0}},
	}
	if len(sprites) != len(wantSprites) {
		t.Fatalf("the content band holds %d rects, want %d (the (43,-15) placement is culled on its exact rect)\ngot: %+v",
			len(sprites), len(wantSprites), sprites)
	}
	for i, w := range wantSprites {
		if sprites[i] != w {
			t.Errorf("sprite %d = %+v, want %+v", i, sprites[i], w)
		}
	}

	squares := passes[0]
	if squares.Color != terrain.EntityMarkerColor {
		t.Fatalf("the first pass is %+v, want the square pass in the entity marker colour %+v — "+
			"squares keep glyph and colour (AC-6, AC-7)", squares.Color, terrain.EntityMarkerColor)
	}
	assertScreenRects(t, squares.Rects,
		wantDisplacedRects(cam, cliffProjection(), []image.Point{{X: 1, Y: 2}, {X: 0, Y: 3}}, nativeEntitySquare))
}

// TestEntitySpriteCullKeepsACrownOnlySprite — SC-5 (AC-6's cull clause): a
// sprite whose ground cell sits below the view while its crown reaches into
// it survives the cull, because the cull runs on the EXACT placed world
// rectangle — the object layer's own — and not on the ground cell or the
// camera's tile band; a wholly off-view sprite in the same slice is dropped,
// and a square entity beside them is untouched.
//
// The tall art: canvas 64x64, centre (32,60), frame 20x200, so anchor
// (10,128) and, flat at cell (5,12), top-left (166,272) — 28 crown pixels
// inside a 300-tall view, the ground point at screen y 400, and cell row 12
// outside the camera's tile band. Both discriminators are asserted, not
// assumed, exactly as the object layer's own cull fixture asserts them.
func TestEntitySpriteCullKeepsACrownOnlySprite(t *testing.T) {
	v := overlayViewer(t, 100, 80, 400, 300)
	cam := v.Camera()
	if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
		t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
	}

	tall := unitArt(64, 64, 32, 60, 20, 200)
	artA := entityArtA()
	v.SetEntities([]MapEntity{
		withFrame(image.Pt(5, 12), tall),  // crown-only: kept
		withFrame(image.Pt(50, 40), artA), // nowhere near the view: culled
		{Cell: image.Pt(2, 2)},            // a square, unaffected by either
	})

	// The discriminators. The ground point is the flat cell centre — the
	// placement's own Ground() — and it must lie below the view, and the cell
	// row outside the tile band, or a cull by either would keep the sprite too
	// and this test would prove nothing.
	sprites, _, _ := v.entityLayer()
	if len(sprites) != 2 {
		t.Fatalf("the layer built %d sprite placements, want 2 before the cull", len(sprites))
	}
	ground := sprites[0].Ground()
	if want := image.Pt(5*terrain.CellSize+16, 12*terrain.CellSize+16); ground != want {
		t.Fatalf("fixture drift: Ground() = %v, want %v", ground, want)
	}
	_, groundY := cam.WorldToScreen(float64(ground.X), float64(ground.Y))
	if groundY < float64(cam.ViewH) {
		t.Fatalf("fixture does not discriminate: the ground point is at screen y %v, inside a %d-tall view — "+
			"a cull by ground cell would keep this sprite", groundY, cam.ViewH)
	}
	if band := cam.VisibleTiles(); 12 >= band.Row0 && 12 < band.Row1 {
		t.Fatalf("fixture does not discriminate: cell row 12 is inside the camera's tile band %+v — "+
			"a cull by tile band would keep this sprite", band)
	}

	passes := v.overlayPasses()
	if len(passes) != 1 {
		t.Fatalf("got %d passes, want the square pass alone", len(passes))
	}
	band := v.planeSprites()
	want := staticScreenRect{screenRect: screenRect{X: 166, Y: 272, W: 20, H: 200}, Frame: tall.Frames[0], Inspection: InspectionSubject{Kind: InspectionUnit, ID: 0}}
	want.order = artOrder{phase: 4, cell: image.Pt(5, 12), rank: 3, part: 1}
	if len(band) != 1 || band[0] != want {
		t.Fatalf("the content band is %+v, want exactly the crown-only sprite %+v — its rect meets the view "+
			"even though its ground cell does not (AC-6)", band, want)
	}
	if passes[0].Color != terrain.EntityMarkerColor {
		t.Fatalf("the first pass's colour is %+v, want the entity square's %+v", passes[0].Color, terrain.EntityMarkerColor)
	}
	assertScreenRects(t, passes[0].Rects, wantEntityScreenRects(cam, []image.Point{{X: 2, Y: 2}}))
}

func TestEntityPassesPrecedeTheThreeDiagnostics(t *testing.T) {
	if want := (color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff}); terrain.EntityMarkerColor != want {
		t.Fatalf("EntityMarkerColor = %+v, want the opaque magenta %+v", terrain.EntityMarkerColor, want)
	}
	for _, other := range []color.RGBA{terrain.MarkerColor, terrain.UnitMarkerColor, terrain.StaticMarkerColor} {
		if terrain.EntityMarkerColor == other {
			t.Fatalf("EntityMarkerColor equals the diagnostic colour %+v, so pass order cannot be observed by colour", other)
		}
	}

	v := staticsIdentityViewer(t, true, true)
	cam := v.Camera()
	objects := []image.Point{{X: 0, Y: 0}}
	units := []image.Point{{X: 1, Y: 1}}
	artA := entityArtA()
	v.SetObjects(true, objects)
	v.SetUnits(true, units)
	v.SetEntities([]MapEntity{
		{Cell: image.Pt(1, 1)},          // a square, on the unit cross's own cell
		withFrame(image.Pt(0, 1), artA), // a sprite
	})

	passes := v.overlayPasses()
	if len(passes) != 4 {
		t.Fatalf("got %d passes with every layer on, want 4 (squares, objects, units, statics)", len(passes))
	}
	if passes[0].Color != terrain.EntityMarkerColor || passes[1].Color != terrain.MarkerColor ||
		passes[2].Color != terrain.UnitMarkerColor || passes[3].Color != terrain.StaticMarkerColor {
		t.Fatalf("pass colours are [0]=%+v [1]=%+v [2]=%+v [3]=%+v, want squares %+v, objects %+v, units %+v, statics %+v",
			passes[0].Color, passes[1].Color, passes[2].Color, passes[3].Color,
			terrain.EntityMarkerColor, terrain.MarkerColor, terrain.UnitMarkerColor, terrain.StaticMarkerColor)
	}

	// The sprite is the routing fixture's (0,1)/artA placement — staticsGrid
	// carries the cliff altitudes, so the same hand-derived (11,49) holds. It is
	// in the content band now, which this viewer's bundles contribute nothing to.
	want := staticScreenRect{screenRect: screenRect{X: 11, Y: 49, W: 10, H: 6}, Frame: artA.Frames[0], Inspection: InspectionSubject{Kind: InspectionUnit, ID: 0}}
	want.order = artOrder{phase: 4, cell: image.Pt(0, 1), rank: 3, part: 1}
	band := v.planeSprites()
	found := 0
	for _, s := range band {
		if s == want {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("the content band is %+v, want it to carry the entity sprite %+v exactly once", band, want)
	}

	proj := cliffProjection()
	assertScreenRects(t, passes[0].Rects, wantDisplacedRects(cam, proj, []image.Point{{X: 1, Y: 1}}, nativeEntitySquare))
	assertScreenRects(t, passes[1].Rects, wantDisplacedRects(cam, proj, objects, nativeArms))
	assertScreenRects(t, passes[2].Rects, wantDisplacedRects(cam, proj, units, nativeUnitArms))
	assertScreenRects(t, passes[3].Rects, wantDisplacedRects(cam, proj, staticMarkedCells, nativeStaticArms))

	// The fixture must actually exercise the covering: if no entity square met
	// a cross, the order would be unobservable on screen.
	covered := false
	for _, e := range passes[0].Rects {
		for _, u := range passes[2].Rects {
			if rectsOverlap(e, u) {
				covered = true
			}
		}
	}
	if !covered {
		t.Fatalf("fixture does not discriminate: no entity square %+v meets a unit cross %+v, so nothing here "+
			"depends on which pass runs first", passes[0].Rects, passes[2].Rects)
	}
}

// TestEntitySquaresPlaceThroughTheFamilyTransform — 0020 SC-7 (AC-6),
// restated over SetEntities with every entity art-less: the square pass
// places each cell's native square through the same camera transform a
// terrain tile gets, at any pan and zoom; it emits exactly one rectangle per
// in-map cell, in the order the entities were given; an off-map entity
// contributes nothing while its in-map neighbours in the same list still do;
// and, with the whole map inside the view so nothing is culled, the number
// of rectangles equals the number of entities the viewer was handed.
func TestEntitySquaresPlaceThroughTheFamilyTransform(t *testing.T) {
	if terrain.CellSize != 32 {
		t.Fatalf("CellSize = %d, want 32; every literal in this file is stated over 32-pixel cells", terrain.CellSize)
	}

	// Cells whose squares stay inside the view at every position below, so a
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

			v.SetEntities(artless(cells))

			want := wantEntityScreenRects(cam, cells)
			for i, r := range want {
				if !rectInView(cam, r) {
					t.Fatalf("expected rect %d %+v is off-view here, so the comparison would test the cull "+
						"instead of the transform; pick another cell", i, r)
				}
			}
			// The square pass transforms the native geometry through the *same*
			// transform the diagnostic crosses take, with no independent pixel
			// snapping and no second cell-to-screen conversion.
			assertScreenRects(t, entitySquares(v), want)
		})
	}

	t.Run("one square per entity, in list order", func(t *testing.T) {
		v := overlayViewer(t, 100, 80, 400, 300)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}

		if got, want := nativeEntitySquare(2, 1), image.Rect(75, 43, 85, 53); len(got) != 1 || got[0] != want {
			t.Fatalf("nativeEntitySquare(2,1) = %v, want [%v] — the test's own FR-6 transcription is wrong", got, want)
		}

		order := []image.Point{{X: 9, Y: 6}, {X: 2, Y: 1}, {X: 5, Y: 4}}
		v.SetEntities(artless(order))

		got := entitySquares(v)
		if len(got) != len(order) {
			t.Fatalf("got %d rects for %d entities, want %d (one filled square each, all in view here)",
				len(got), len(order), len(order))
		}
		// At the origin at native zoom the camera is the identity, so the screen
		// rects are the native squares themselves: 10x10 about the cell centre.
		for i, c := range order {
			cx, cy := float64(32*c.X+16), float64(32*c.Y+16)
			if want := (screenRect{X: cx - 5, Y: cy - 5, W: 10, H: 10}); got[i] != want {
				t.Fatalf("entity %d (cell %v) = %+v, want the FR-6 square %+v", i, c, got[i], want)
			}
		}
	})

	t.Run("off-map entities contribute nothing", func(t *testing.T) {
		// A 10x10 world is exactly 320x320 world px; matching the view keeps the
		// camera at the origin (Clamp centres an axis only when the world is
		// *smaller* than the view) and every in-map square visible.
		v := overlayViewer(t, 10, 10, 320, 320)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}

		// "An off-map entity draws neither, as before" — a world permits an
		// entity to walk off the grid, so this is the ordinary case rather than
		// an error, and its in-map neighbours in the same list must be
		// unaffected.
		inMap := []image.Point{{X: 0, Y: 0}, {X: 3, Y: 4}, {X: 9, Y: 9}}
		v.SetEntities(artless([]image.Point{
			{X: -1, Y: -1},           // walked off the top-left
			inMap[0],                 // the map's first cell
			{X: 10, Y: 5},            // col == cols, one past the right edge
			inMap[1],                 // interior
			{X: 5, Y: 10},            // row == rows, one past the bottom edge
			{X: 1 << 20, Y: 1 << 20}, // far off the map: no geometry, no wrap, no panic
			inMap[2],                 // the map's last cell
		}))

		assertScreenRects(t, entitySquares(v), wantEntityScreenRects(cam, inMap))
	})

	t.Run("one rect per entity before the cull, and the count is reported", func(t *testing.T) {
		// With the whole 10x10 map inside the view nothing is culled, so the
		// rectangle count IS the square count before the cull, and it equals the
		// number of entities handed over — which is the number the world holds
		// (the caller's own invariant, witnessed against a world one tier up).
		v := overlayViewer(t, 10, 10, 320, 320)
		for _, n := range []int{1, 2, 5, 17} {
			cells := make([]image.Point, n)
			for i := range cells {
				cells[i] = image.Pt(i%10, (i*3)%10)
			}
			v.SetEntities(artless(cells))
			if got := v.EntityMarkers(); got != n {
				t.Fatalf("EntityMarkers() = %d after %d entities were set, want %d", got, n, n)
			}
			if got := entitySquares(v); len(got) != n {
				t.Fatalf("got %d rects for %d in-map art-less entities with the whole map in view, want %d "+
					"(exactly one square per entity before the cull)", len(got), n, n)
			}
		}
	})
}

// TestEntitySquaresCullOnThePlacedRect — 0020 SC-7 (AC-6), restated over
// art-less SetEntities: a square whose PLACED rectangle does not meet the
// view is dropped and a straddling one is kept whole; the cull is applied to
// the rect after the camera transform, not to the native one before it; and
// it is decided against the view the viewer has NOW, never one remembered
// from an earlier call.
func TestEntitySquaresCullOnThePlacedRect(t *testing.T) {
	t.Run("wholly outside is culled, straddling is kept whole", func(t *testing.T) {
		// A 78x60 view over a 100x80-cell world at native zoom covers world
		// [0,78) x [0,60). Cell (0,0)'s square [11,21) x [11,21) is wholly inside;
		// cell (2,1)'s [75,85) x [43,53) straddles the right edge; cell (50,40) is
		// far away.
		v := overlayViewer(t, 100, 80, 78, 60)
		cam := v.Camera()
		if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
			t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
		}
		v.SetEntities(artless([]image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 50, Y: 40}}))

		// Two of the three survive: culling everything or nothing fails on the
		// count alone, and the straddling square keeps its full 10 px width even
		// though only 3 px of it are on screen — the framebuffer does that clip,
		// the transform never snaps or trims.
		assertScreenRects(t, entitySquares(v), []screenRect{
			{X: 11, Y: 11, W: 10, H: 10},
			{X: 75, Y: 43, W: 10, H: 10},
		})
	})

	t.Run("the cull sees the placed rect, not the native one", func(t *testing.T) {
		// Panned to world x 84, the two questions "does the NATIVE square meet
		// [0,78) x [0,60)" and "does the PLACED one" have opposite answers on two
		// different cells at once, so an implementation that culls before the
		// camera transform fails in both directions here rather than one.
		//
		//   cell (0,0): native [11,21) is inside the view box; placed at X=-73,
		//               its right edge -63, entirely off the left of the screen.
		//   cell (3,0): native [107,117) is past the view box's right edge;
		//               placed at X=23, squarely on screen.
		//   cell (2,1): native and placed both meet the view — the control.
		v := overlayViewer(t, 100, 80, 78, 60)
		cam := v.Camera()
		cam.Pan(84, 0)
		if cam.X != 84 || cam.Y != 0 {
			t.Fatalf("camera at (%v,%v), want (84,0) — the clamp moved it; pick another position", cam.X, cam.Y)
		}

		cells := []image.Point{{X: 0, Y: 0}, {X: 2, Y: 1}, {X: 3, Y: 0}}

		// Guard the fixture: the two discriminating cells must really disagree
		// with themselves before and after the transform, or this test passes for
		// an implementation that culls at either point.
		viewBox := screenRect{X: 0, Y: 0, W: float64(cam.ViewW), H: float64(cam.ViewH)}
		native := func(c image.Point) screenRect {
			sq := nativeEntitySquare(c.X, c.Y)[0]
			return screenRect{X: float64(sq.Min.X), Y: float64(sq.Min.Y), W: float64(sq.Dx()), H: float64(sq.Dy())}
		}
		if !rectsOverlap(native(cells[0]), viewBox) {
			t.Fatalf("fixture does not discriminate: cell %v's NATIVE square already misses the view box", cells[0])
		}
		if rectsOverlap(native(cells[2]), viewBox) {
			t.Fatalf("fixture does not discriminate: cell %v's NATIVE square already meets the view box", cells[2])
		}

		v.SetEntities(artless(cells))
		assertScreenRects(t, entitySquares(v), []screenRect{
			{X: -9, Y: 43, W: 10, H: 10},
			{X: 23, Y: 11, W: 10, H: 10},
		})
	})

	t.Run("the cull reads the view it has now", func(t *testing.T) {
		// The same entities, the same camera position, three different view sizes
		// in a row: an implementation that decided the cull against a view size it
		// captured earlier — at construction, or on the previous call — returns
		// the first answer three times.
		v := overlayViewer(t, 100, 80, 400, 300)
		cells := []image.Point{{X: 0, Y: 0}, {X: 5, Y: 4}}
		v.SetEntities(artless(cells))

		assertScreenRects(t, entitySquares(v), wantEntityScreenRects(v.Camera(), cells))

		// An 8x8 view reaches world [0,8) x [0,8): neither square, the nearest
		// starting at (11,11), meets it.
		layoutViewport(v, 8, 8)
		if cam := v.Camera(); cam.X != 0 || cam.Y != 0 {
			t.Fatalf("camera at (%v,%v) after the shrink, want the origin — the fixture assumes no re-anchoring",
				cam.X, cam.Y)
		}
		if got := entitySquares(v); len(got) != 0 {
			t.Fatalf("got %d rects in an 8x8 view (%+v), want none — every square starts past (8,8)", len(got), got)
		}

		layoutViewport(v, 400, 300)
		assertScreenRects(t, entitySquares(v), wantEntityScreenRects(v.Camera(), cells))
	})
}

// TestEntitySquareCarriesItsCellsLiftDisplacedAndNoneFlat — 0020 SC-7
// (AC-6), restated over art-less SetEntities: an entity square carries its
// own cell's terrain height lift in displaced mode and none in flat, by
// exactly the lift the diagnostic markers take — the same
// -AnchorHeight(cell)-MinV, applied before the camera transform.
//
// Both halves are stated against oracles built from an independent
// terrain.Project over a copy of the fixture's altitude bytes, never from
// v.proj, so the assertion cannot share a mistake with the transform under
// test.
func TestEntitySquareCarriesItsCellsLiftDisplacedAndNoneFlat(t *testing.T) {
	// cliffCanvasH / cliffW*CellSize matches the world on both axes exactly, so
	// the camera sits at the identity and every screen coordinate is a world one.
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	cam := v.Camera()
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced — the fixture must select displaced for this test to say anything", v.Mode())
	}

	proj := cliffProjection()
	const col, row = 0, 1
	dy := -proj.AnchorHeight(col, row) - proj.MinV
	if dy != 32 {
		t.Fatalf("fixture drift: -AnchorHeight(%d,%d)-MinV = %d, want 32 (hand-computed: AnchorHeight=63, MinV=-95)",
			col, row, dy)
	}

	cells := []image.Point{{X: col, Y: row}}
	v.SetEntities(artless(cells))

	// The displaced placement, against the independently projected oracle.
	flat := wantEntityScreenRects(cam, cells)
	got := entitySquares(v)
	assertScreenRects(t, got, wantDisplacedRects(cam, proj, cells, nativeEntitySquare))

	// And the lift stated explicitly rather than only implied by the match: only
	// Y moves, and it moves by exactly the AnchorHeight/MinV offset the two
	// shipped crosses take for the same cell.
	if len(got) != len(flat) {
		t.Fatalf("rectangle count changed under displacement: %d -> %d", len(flat), len(got))
	}
	for i := range got {
		if got[i].X != flat[i].X || got[i].W != flat[i].W || got[i].H != flat[i].H {
			t.Fatalf("square %d: displaced %+v, flat %+v — only Y may move", i, got[i], flat[i])
		}
		if offset := got[i].Y - flat[i].Y; offset != float64(dy) {
			t.Fatalf("square %d: Y offset = %v, want the AnchorHeight/MinV offset %d", i, offset, dy)
		}
	}

	// The flat half: over the SAME valid, sloped altitude grid, flat mode
	// carries no lift at all. v.proj stays non-nil throughout, which is why the
	// guard inside the shared transform must be Mode() and not v.proj != nil.
	v.SetFlat(true)
	if v.Mode() != ModeFlat {
		t.Fatalf("SetFlat(true): Mode() = %v, want flat", v.Mode())
	}
	if v.proj == nil {
		t.Fatal("SetFlat(true): v.proj is nil — the projection must still exist; only the mode changed")
	}
	assertScreenRects(t, entitySquares(v), wantEntityScreenRects(v.Camera(), cells))
}

// TestViewerGivenNoEntitiesIsTheOneThatShipped — SC-5's identity clauses
// (AC-7, 0020 AC-10): a viewer never given entities produces no entity pass
// at all and leaves the pass slice exactly as it was before either story; a
// viewer pushed ART-LESS entities produces exactly the 0020 slice — the
// square pass first, no sprite pass anywhere, every diagnostic untouched —
// which is the screen pkg/game's T4 boundary hands over, byte for byte; and
// entities given once do not outlive being replaced by fewer, by none, or by
// a list that lands entirely off the map.
func TestViewerGivenNoEntitiesIsTheOneThatShipped(t *testing.T) {
	t.Run("never given entities: no pass, and the shipped slice", func(t *testing.T) {
		v := staticsIdentityViewer(t, true, true)
		cam := v.Camera()
		objects := []image.Point{{X: 0, Y: 0}}
		units := []image.Point{{X: 1, Y: 1}}
		v.SetObjects(true, objects)
		v.SetUnits(true, units)

		if got := v.EntityMarkers(); got != 0 {
			t.Fatalf("EntityMarkers() = %d on a viewer never given entities, want 0", got)
		}
		if got := entitySquares(v); got != nil {
			t.Fatalf("the square pass holds %+v on a viewer never given entities, want none", got)
		}

		passes := v.overlayPasses()
		if len(passes) != 3 {
			t.Fatalf("got %d overlay passes with no entities, want the 3 that shipped (objects, units, statics)",
				len(passes))
		}
		if passes[0].Color != terrain.MarkerColor || passes[1].Color != terrain.UnitMarkerColor ||
			passes[2].Color != terrain.StaticMarkerColor {
			t.Fatalf("pass colours are [0]=%+v [1]=%+v [2]=%+v, want the shipped objects, units, statics — "+
				"a viewer given no entities must draw exactly what it drew before this story",
				passes[0].Color, passes[1].Color, passes[2].Color)
		}
		// The content band is the art alone: with no entity there is no entity
		// sprite in it, which is the statement the vanished sprite pass used to
		// carry and which the entity layer is now the one place to make.
		if sprites, _, _ := v.entityLayer(); len(sprites) != 0 {
			t.Fatalf("the entity layer built %d sprites on a viewer never given entities, want none", len(sprites))
		}
		proj := cliffProjection()
		assertScreenRects(t, passes[0].Rects, wantDisplacedRects(cam, proj, objects, nativeArms))
		assertScreenRects(t, passes[1].Rects, wantDisplacedRects(cam, proj, units, nativeUnitArms))
		assertScreenRects(t, passes[2].Rects, wantDisplacedRects(cam, proj, staticMarkedCells, nativeStaticArms))
	})

	t.Run("an art-less push yields the 0020 slice", func(t *testing.T) {
		v := staticsIdentityViewer(t, true, true)
		cam := v.Camera()
		objects := []image.Point{{X: 0, Y: 0}}
		units := []image.Point{{X: 1, Y: 1}}
		entities := []image.Point{{X: 1, Y: 1}, {X: 0, Y: 0}} // both coincident with a cross
		v.SetObjects(true, objects)
		v.SetUnits(true, units)
		v.SetEntities(artless(entities))

		passes := v.overlayPasses()
		if len(passes) != 4 {
			t.Fatalf("got %d overlay passes with every entity art-less, want the 4 of 0020 "+
				"(squares, objects, units, statics) — no sprite pass may exist without art", len(passes))
		}
		bare := staticsIdentityViewer(t, true, true)
		if got, want := v.planeSprites(), bare.planeSprites(); !reflect.DeepEqual(got, want) {
			t.Fatalf("the content band is %+v on an art-less push and %+v with no entity at all; an art-less "+
				"entity contributes nothing to it and the screen at this boundary is the 0020 one byte for byte",
				got, want)
		}
		if passes[0].Color != terrain.EntityMarkerColor || passes[1].Color != terrain.MarkerColor ||
			passes[2].Color != terrain.UnitMarkerColor || passes[3].Color != terrain.StaticMarkerColor {
			t.Fatalf("pass colours are [0]=%+v [1]=%+v [2]=%+v [3]=%+v, want squares %+v, objects %+v, units %+v, statics %+v — "+
				"the square pass MUST still come first: its glyph is a filled square and a superset of every "+
				"cross on the same cell, so drawn later it would erase a coincident diagnostic",
				passes[0].Color, passes[1].Color, passes[2].Color, passes[3].Color,
				terrain.EntityMarkerColor, terrain.MarkerColor, terrain.UnitMarkerColor, terrain.StaticMarkerColor)
		}

		proj := cliffProjection()
		assertScreenRects(t, passes[0].Rects, wantDisplacedRects(cam, proj, entities, nativeEntitySquare))
		assertScreenRects(t, passes[1].Rects, wantDisplacedRects(cam, proj, objects, nativeArms))
		assertScreenRects(t, passes[2].Rects, wantDisplacedRects(cam, proj, units, nativeUnitArms))
		assertScreenRects(t, passes[3].Rects, wantDisplacedRects(cam, proj, staticMarkedCells, nativeStaticArms))

		// The fixture must actually exercise the covering the order exists for.
		covered := false
		for _, e := range passes[0].Rects {
			for _, u := range passes[2].Rects {
				if rectsOverlap(e, u) {
					covered = true
				}
			}
		}
		if !covered {
			t.Fatalf("fixture does not discriminate: no entity square %+v meets a unit cross %+v, so nothing here "+
				"depends on which pass runs first", passes[0].Rects, passes[2].Rects)
		}
	})

	t.Run("no diagnostics and no entities: no passes at all", func(t *testing.T) {
		v := overlayViewer(t, 100, 80, 400, 300)
		if passes := v.overlayPasses(); len(passes) != 0 {
			t.Fatalf("got %d overlay passes on a bare viewer (%+v), want none", len(passes), passes)
		}
	})

	t.Run("an empty or off-map list produces no pass", func(t *testing.T) {
		v := overlayViewer(t, 10, 10, 320, 320)
		for _, tc := range []struct {
			name  string
			cells []image.Point
		}{
			{"nil", nil},
			{"empty but non-nil", []image.Point{}},
			{"every cell off the map", []image.Point{{X: -1, Y: 0}, {X: 10, Y: 10}}},
		} {
			v.SetEntities(artless(tc.cells))
			if got := entitySquares(v); got != nil {
				t.Errorf("%s: the square pass holds %+v, want none", tc.name, got)
			}
			if passes := v.overlayPasses(); len(passes) != 0 {
				t.Errorf("%s: got %d passes, want none — a pass with nothing to draw is omitted entirely", tc.name, len(passes))
			}
		}
	})

	t.Run("a shrunken set leaves no stale rects", func(t *testing.T) {
		v := overlayViewer(t, 10, 10, 320, 320)
		cam := v.Camera()

		three := []image.Point{{X: 1, Y: 1}, {X: 4, Y: 5}, {X: 8, Y: 2}}
		v.SetEntities(artless(three))
		assertScreenRects(t, entitySquares(v), wantEntityScreenRects(cam, three))

		one := []image.Point{{X: 4, Y: 5}}
		v.SetEntities(artless(one))
		if got := v.EntityMarkers(); got != 1 {
			t.Fatalf("EntityMarkers() = %d after the set shrank to 1, want 1", got)
		}
		assertScreenRects(t, entitySquares(v), wantEntityScreenRects(cam, one))

		v.SetEntities(nil)
		if got := v.EntityMarkers(); got != 0 {
			t.Fatalf("EntityMarkers() = %d after the set was cleared, want 0", got)
		}
		if got := entitySquares(v); got != nil {
			t.Fatalf("the square pass holds %+v after the set was cleared, want none — nothing survives a frame", got)
		}
		if passes := v.overlayPasses(); len(passes) != 0 {
			t.Fatalf("got %d passes after the set was cleared, want none", len(passes))
		}
	})
}

func TestSetEntitiesResyncsNothing(t *testing.T) {
	const planted = 12345.0
	cells := []image.Point{{X: 5, Y: 4}, {X: 6, Y: 5}}

	for _, tc := range []struct {
		name string
		v    func(t *testing.T) *Viewer
	}{
		{"flat viewer", func(t *testing.T) *Viewer { return overlayViewer(t, 100, 80, 400, 300) }},
		{"displaced viewer", func(t *testing.T) *Viewer {
			return identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.v(t)
			cam := v.Camera()
			mode := v.Mode()

			cam.SetWorldHeight(planted)
			if cam.WorldH() != planted {
				t.Fatalf("setup: WorldH() = %v after planting %v", cam.WorldH(), planted)
			}
			x, y := cam.X, cam.Y

			v.SetEntities(artless(cells))

			if got := cam.WorldH(); got != planted {
				t.Fatalf("SetEntities re-synced the camera's world: WorldH() = %v, want the planted %v untouched (DD-5)",
					got, planted)
			}
			if cam.X != x || cam.Y != y {
				t.Fatalf("SetEntities moved the camera to (%v,%v), want it left at (%v,%v)", cam.X, cam.Y, x, y)
			}
			if v.Mode() != mode {
				t.Fatalf("SetEntities moved the mode to %v, want the unchanged %v", v.Mode(), mode)
			}

			// The control: a setter that DOES sync overwrites the same planted
			// value, so the check above is a real observation of the call's
			// absence and not of a probe that could never fire.
			v.SetUnits(true, cells)
			if got := cam.WorldH(); got == planted {
				t.Fatalf("the probe does not discriminate: SetUnits left WorldH() at the planted %v too, "+
					"so this test could not have seen a resync in SetEntities", got)
			}
		})
	}
}

func TestDrawPaintsEntitySpritesLazilyAndSkipsZeroArea(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	artA := entityArtA()
	zero := unitArt(8, 8, 4, 7, 0, 0) // Frame non-nil, 0x0: resolves, draws nothing
	v.SetEntities([]MapEntity{
		withFrame(image.Pt(0, 1), artA),
		withFrame(image.Pt(0, 3), zero),
	})

	// The zero-area frame PLACES and survives the cull like any other: anchor
	// (0,3) over an 8x8 canvas and 0x0 frame is (0,3), so at lift 0 and MinV
	// -95 its rect stands at (16,204) with no extent — inside the 96x223 view.
	band := v.planeSprites()
	if len(band) != 2 {
		t.Fatalf("got %+v, want a content band holding both placements — a zero-area frame places and culls "+
			"like any other (DD-6)", band)
	}
	if want := (staticScreenRect{screenRect: screenRect{X: 16, Y: 204, W: 0, H: 0}, order: artOrder{phase: 4, cell: image.Pt(0, 3), rank: 3, index: 1, part: 1}, Frame: zero.Frames[0], Inspection: InspectionSubject{Kind: InspectionUnit, ID: 0}}); band[1] != want {
		t.Fatalf("the zero-area sprite is %+v, want %+v", band[1], want)
	}
	if v.staticImages != nil {
		t.Fatalf("building the passes built %d textures; textures are the draw's alone", len(v.staticImages))
	}

	screen := ebiten.NewImage(cliffW*terrain.CellSize, cliffCanvasH)
	v.Draw(screen)

	if len(v.staticImages) != 1 {
		t.Fatalf("after one draw the texture cache holds %d entries, want 1 — artA's frame, and NOTHING for "+
			"the zero-area one (ebiten refuses an empty image)", len(v.staticImages))
	}
	// The key gained the ramp row in 0044; the frame's pointer identity is
	// still what selects an entry, and v.spriteKey is the viewer's own spelling
	// of the state it is in.
	if v.staticImages[v.spriteKey(artA.Frames[0])] == nil {
		t.Fatalf("the drawn frame has no texture under its own pointer identity")
	}
	if _, built := v.staticImages[v.spriteKey(zero.Frames[0])]; built {
		t.Fatalf("the zero-area frame got a texture; it must paint nothing and build none")
	}

	// A second draw builds nothing new: the cache is keyed on frame identity.
	v.Draw(screen)
	if len(v.staticImages) != 1 {
		t.Fatalf("a second draw grew the cache to %d, want 1 — a frame is uploaded once", len(v.staticImages))
	}

	// And the art-less viewer: no sprite, no texture, anywhere.
	bare := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	bare.SetEntities(artless([]image.Point{{X: 0, Y: 1}, {X: 1, Y: 2}}))
	bare.Draw(screen)
	if bare.staticImages != nil {
		t.Fatalf("an art-less viewer's Draw built %d sprite textures, want none — with no bundle anywhere the "+
			"screen is the pre-story one", len(bare.staticImages))
	}
}

// THE DEPTH TIE CROSSES THE SEAM (hotfix). The render tier's own tests fix
// what DepthOrder does with a tie; this fixes that the tie an entity sprite
// carries is the LIFE STATE the simulation sent, and not something this
// package decides. Without it the ordering rule would be right and reach the
// wrong entities.
//
// It asserts the VALUES, one per state, rather than "the dead one differs":
// LifeDead is TieCorpse and both other states are TieUnit — a downed unit is a
// body in the way that has dropped nothing, and stays with the living, which is
// terrain.DepthTie's own stated choice.
//
// TO CONFIRM IT WITNESSES THE ASSIGNMENT, delete the `p.DepthTie = ...` pair
// from entityLayer (overlay.go): every sprite comes back at terrain.TieGround,
// the zero value, and all three rows redden.
func TestEntityLayerCarriesTheLifeStateAsADepthTie(t *testing.T) {
	v := overlayViewer(t, 100, 80, 400, 300)
	art := entityArtA()

	alive := withFrame(image.Pt(2, 2), art)
	downed := withFrame(image.Pt(3, 2), art)
	downed.Life = LifeDowned
	dead := withFrame(image.Pt(4, 2), art)
	dead.Life = LifeDead
	v.SetEntities([]MapEntity{alive, downed, dead})

	sprites, _, _ := v.entityLayer()
	if len(sprites) != 3 {
		t.Fatalf("the layer built %d sprite placements, want 3", len(sprites))
	}
	for i, want := range []terrain.DepthTie{terrain.TieUnit, terrain.TieUnit, terrain.TieCorpse} {
		if got := sprites[i].DepthTie; got != want {
			t.Errorf("sprite %d (life %d) carries DepthTie %d, want %d",
				i, []uint8{LifeAlive, LifeDowned, LifeDead}[i], got, want)
		}
	}
}
