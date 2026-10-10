package ui

// The windowed viewer's half of the static-object layer's DRAWING (AC-6,
// AC-7, SC-7 and SC-8's window half).
//
// Two fixtures, for two different questions. The cull-and-transform is exercised
// over hand-built placements at hand-chosen world rectangles, because the
// property under test is about RECTANGLES AND A CAMERA and a placement built
// from a class registry cannot be put where a case needs it. The draw pass, the
// texture cache and the third glyph are exercised over statics_test.go's own
// three-class bundle on the 3x4 cliff grid, because there the question is what
// the VIEWER does with the list it built.
//
// No game data is read and no window opens. Ebitengine's draw-side calls
// (NewImage, NewImageFromImage, DrawImage) do run before the game starts;
// reading pixels back does not, so what a texture CONTAINS is asserted one tier
// down, where the blit returns a plain image, and here only its identity, its
// size and the transform it was submitted at.

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// syntheticStaticFrame builds one synthetic frame of the given size. Only its SIZE and its
// pointer identity reach anything under test here: the size is the world
// rectangle's own extent, and the identity is the texture cache's key.
func syntheticStaticFrame(w, h int) *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: w, Height: h, Pixels: make([]terrain.StaticPixel, w*h)}
	for i := range f.Pixels {
		f.Pixels[i] = terrain.StaticPixel{Index: uint8(i % 251), Opaque: true}
	}
	return f
}

// cullFrames are the three frames the cull fixture places. Two placements share
// cullFrameA, so the survivor list must carry that one pointer twice — nothing
// here merges two cells of a class into one entry.
var (
	cullFrameA = syntheticStaticFrame(10, 6)
	cullFrameB = syntheticStaticFrame(20, 20)
	cullFrameC = syntheticStaticFrame(40, 380)
)

// cullPlacement builds one placement at a chosen world top-left, with the anchor
// that makes its Ground() land on its own cell's flat marker point — so every
// fixture entry is a placement the builder could actually have produced, and the
// "ground cell" a cull must NOT use is a real one.
func cullPlacement(col, row, x, y int, f *terrain.StaticFrame) terrain.StaticPlacement {
	const half = terrain.CellSize / 2
	return terrain.StaticPlacement{
		Cell:    image.Pt(col, row),
		TopLeft: image.Pt(x, y),
		Anchor:  image.Pt(col*terrain.CellSize+half-x, row*terrain.CellSize+half-y),
		Frame:   f,
	}
}

// cullPlacements is the fixture: six placements over a 100x80-cell world, viewed
// through a 400x300 window whose camera sits at world (600, 500).
//
//	i  cell     world rect                what it is there for
//	0  (1,1)    [50,70)   x [40,60)       meets the view ONLY BEFORE the transform
//	1  (21,9)   [700,710) x [300,306)     wholly above the view
//	2  (21,17)  [700,710) x [550,556)     plainly inside
//	3  (22,17)  [720,730) x [560,566)     inside, sharing frame A with 2
//	4  (31,17)  [1000,1010) x [550,556)   starts exactly ON the right edge
//	5  (20,27)  [660,700) x [520,900)     the TALL one: ground cell below the view
//	6  (18,17)  [590,600) x [550,556)     ends exactly ON the left edge
//
// Entries 4 and 6 are the two edges of the half-open cull, and they are opposite
// halves of it: 6 ends exactly where the view begins and must be dropped, 4
// begins exactly where the view ends and must be dropped too. A cull written
// with either comparison one notch loose keeps exactly one of them.
//
// Entry 0 is the discriminating case for "the cull runs on the transformed
// rectangle": read as raw world coordinates its rect lies inside
// [0,400) x [0,300), so an implementation that culled before applying the camera
// keeps it — at every zoom, and with nothing else in the fixture noticing.
func cullPlacements() []terrain.StaticPlacement {
	return []terrain.StaticPlacement{
		cullPlacement(1, 1, 50, 40, cullFrameB),
		cullPlacement(21, 9, 700, 300, cullFrameA),
		cullPlacement(21, 17, 700, 550, cullFrameA),
		cullPlacement(22, 17, 720, 560, cullFrameA),
		cullPlacement(31, 17, 1000, 550, cullFrameA),
		cullPlacement(20, 27, 660, 520, cullFrameC),
		cullPlacement(18, 17, 590, 550, cullFrameA),
	}
}

// cullCamera is the fixture's camera at one zoom: a 100x80-cell world (3200x2560
// world px, larger than the view on both axes, so Clamp holds the offset rather
// than centring an axis) seen through 400x300 at world (600, 500).
func cullCamera(t *testing.T, zoom float64) *camera.Camera {
	t.Helper()
	cam := camera.New(100, 80, 400, 300)
	cam.SetZoom(zoom)
	cam.Pan(600, 500)
	if cam.Zoom != zoom || cam.X != 600 || cam.Y != 500 {
		t.Fatalf("camera is (%v,%v) at zoom %v, want (600,500) at zoom %v — the pan or the zoom was clamped away",
			cam.X, cam.Y, cam.Zoom, zoom)
	}
	return cam
}

// survivor is one expected entry of the cull's output: which fixture placement it
// is, and the screen rectangle it must land at, hand-computed from
// screen = (world - camera) * Zoom with the size scaled by Zoom. Nothing here is
// obtained by calling the camera, so a transform that agreed with a wrong camera
// would still fail.
type survivor struct {
	index      int
	X, Y, W, H float64
}

func TestStaticScreenRectsCullAndTransform(t *testing.T) {
	places := cullPlacements()

	for _, p := range []struct {
		name      string
		zoom      float64
		survivors []survivor
	}{
		{"native zoom", 1, []survivor{
			{2, 100, 50, 10, 6},
			{3, 120, 60, 10, 6},
			{5, 60, 20, 40, 380},
		}},
		{"zoomed in", 2, []survivor{
			{2, 200, 100, 20, 12},
			{3, 240, 120, 20, 12},
			{5, 120, 40, 80, 760},
		}},
		// Zoomed out, the view covers 800x600 world px, so entry 4 — which starts
		// exactly on the right edge at native zoom — comes into view. The survivor
		// set is not the same at every zoom, which is what makes the cull's
		// dependence on the zoom observable at all.
		{"zoomed out", 0.5, []survivor{
			{2, 50, 25, 5, 3},
			{3, 60, 30, 5, 3},
			{4, 200, 25, 5, 3},
			{5, 30, 10, 20, 190},
		}},
	} {
		t.Run(p.name, func(t *testing.T) {
			cam := cullCamera(t, p.zoom)
			got := staticScreenRects(places, cam)

			if len(got) != len(p.survivors) {
				t.Fatalf("got %d survivors, want %d\ngot: %+v", len(got), len(p.survivors), got)
			}
			for i, w := range p.survivors {
				want := staticScreenRect{
					screenRect: screenRect{X: w.X, Y: w.Y, W: w.W, H: w.H},
					Frame:      places[w.index].Frame,
				}
				if got[i] != want {
					t.Errorf("survivor %d (fixture placement %d, cell %v): %+v, want %+v",
						i, w.index, places[w.index].Cell, got[i], want)
				}
			}

			// The two cases the fixture exists for, named rather than left implied
			// by the counts above.
			if !hasFrame(got, cullFrameC) {
				t.Errorf("FR-10: the tall placement was culled. Its ground cell (20,27) is below the view and "+
					"outside cam.VisibleTiles() (%+v), but its sprite reaches into the view — culling by ground "+
					"cell or by the camera's tile band drops exactly this one", cam.VisibleTiles())
			}
			if hasFrame(got, cullFrameB) {
				t.Errorf("the placement at world [50,70)x[40,60) survived. Read as RAW world coordinates it lies " +
					"inside the 400x300 view, but the camera is at (600,500) and it is nowhere near it — the cull " +
					"must run on the rectangle the transform produced, not on the one it was given")
			}

			// The half-open boundary, asserted where the survivor table alone
			// would only say "absent": entry 6 ends exactly at screen x 0 and
			// entry 4 begins exactly at screen x ViewW at native zoom, and
			// neither shares a pixel with the view.
			edge := staticScreenRects(places[6:7], cam)
			if len(edge) != 0 {
				t.Errorf("the placement ending exactly ON the view's left edge survived as %+v — the cull is "+
					"half-open, so a rect whose right edge is the view's left edge covers no pixel of it", edge)
			}
		})
	}
}

// hasFrame reports whether any survivor carries the given frame pointer.
func hasFrame(got []staticScreenRect, f *terrain.StaticFrame) bool {
	for _, s := range got {
		if s.Frame == f {
			return true
		}
	}
	return false
}

func TestStaticScreenRectsRejectedCullsWouldDropTheTallSprite(t *testing.T) {
	cam := cullCamera(t, 1)
	tall := cullPlacements()[5]

	// The ground point the sprite's own geometry puts this cell at.
	ground := tall.Ground()
	if want := image.Pt(20*terrain.CellSize+16, 27*terrain.CellSize+16); ground != want {
		t.Fatalf("fixture drift: Ground() = %v, want %v", ground, want)
	}

	_, groundScreenY := cam.WorldToScreen(float64(ground.X), float64(ground.Y))
	if groundScreenY < float64(cam.ViewH) {
		t.Fatalf("fixture does not discriminate: the ground point is at screen y %v, inside a %d-tall view — "+
			"a cull by ground cell would keep this placement and the test would prove nothing",
			groundScreenY, cam.ViewH)
	}

	if band := cam.VisibleTiles(); tall.Cell.Y >= band.Row0 && tall.Cell.Y < band.Row1 {
		t.Fatalf("fixture does not discriminate: cell row %d is inside the camera's tile band %+v — "+
			"a cull by tile band would keep this placement", tall.Cell.Y, band)
	}

	if got := staticScreenRects([]terrain.StaticPlacement{tall}, cam); len(got) != 1 {
		t.Fatalf("FR-10: the tall placement yielded %d rects, want 1 — its sprite reaches into the view", len(got))
	}
}

// staticRecorder is a draw target that records what the sprite pass submitted.
//
// It is why drawPlane takes an interface: an *ebiten.Image's pixels cannot be
// read back before the game starts, so a concrete target would let this file
// assert only that drawing did not panic — and a pass that drew the wrong
// texture at the wrong place would look exactly the same.
type staticRecorder struct {
	imgs []*ebiten.Image
	geo  [][6]float64
	filt []ebiten.Filter
	red  []float32
}

func (r *staticRecorder) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	r.imgs = append(r.imgs, img)
	r.geo = append(r.geo, [6]float64{
		op.GeoM.Element(0, 0), op.GeoM.Element(0, 1), op.GeoM.Element(0, 2),
		op.GeoM.Element(1, 0), op.GeoM.Element(1, 1), op.GeoM.Element(1, 2),
	})
	r.filt = append(r.filt, op.Filter)
	r.red = append(r.red, op.ColorScale.R())
}

// staticsIdentityViewer is statics_test.go's bundle on its cliff grid, viewed
// through a window matched exactly to the world on both axes — so Clamp centres
// each axis at zero and every screen coordinate below IS a world coordinate.
func staticsIdentityViewer(t *testing.T, art, markers bool) *Viewer {
	t.Helper()
	v := newStaticsViewer(t, staticsBundle(), art, markers)
	layoutViewport(v, cliffW*terrain.CellSize, cliffCanvasH)
	cam := v.Camera()
	if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
		t.Fatalf("camera is (%v,%v) at zoom %v, want the identity (0,0) at 1", cam.X, cam.Y, cam.Zoom)
	}
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v over the cliff fixture, want displaced", v.Mode())
	}
	return v
}

func TestDrawStaticsSubmitsOneTexturePerVisiblePlacement(t *testing.T) {
	v := staticsIdentityViewer(t, true, false)

	var rec staticRecorder
	v.drawPlane(&rec)

	// At row0 the column2 sprite precedes column0 (ANIM-WALKORDER-088).
	want := [][6]float64{
		{1, 0, 78, 0, 1, 32},
		{1, 0, 11, 0, 1, 49},
	}
	if len(rec.geo) != len(want) {
		t.Fatalf("the sprite pass made %d draw calls, want %d (the third placement is above the window)\ngot: %+v",
			len(rec.geo), len(want), rec.geo)
	}
	for i := range want {
		if rec.geo[i] != want[i] {
			t.Errorf("draw %d: GeoM = %v, want %v (scale(Zoom) then translate to the culled rect's own top-left)",
				i, rec.geo[i], want[i])
		}
		if rec.filt[i] != ebiten.FilterNearest {
			t.Errorf("draw %d: Filter = %v, want FilterNearest — object art is not smoothed", i, rec.filt[i])
		}
	}

	// Textures follow the same cell order and retain each frame's own size.
	for i, size := range []image.Point{{X: 4, Y: 4}, {X: 10, Y: 6}} {
		b := rec.imgs[i].Bounds()
		if got := image.Pt(b.Dx(), b.Dy()); got != size {
			t.Errorf("draw %d: texture is %v, want %v", i, got, size)
		}
	}
}

// TestDrawStaticsScalesByTheCameraZoom pins the scale term of the GeoM against a
// zoom the identity fixture does not exercise: at zoom 2 the window covers half
// the world, so only the first placement survives and it is submitted at twice
// its native size, twice its world offset.
func TestDrawStaticsScalesByTheCameraZoom(t *testing.T) {
	v := staticsIdentityViewer(t, true, false)
	cam := v.Camera()
	cam.SetZoom(2)
	if cam.Zoom != 2 || cam.X != 0 || cam.Y != 0 {
		t.Fatalf("camera is (%v,%v) at zoom %v, want the origin at zoom 2", cam.X, cam.Y, cam.Zoom)
	}

	var rec staticRecorder
	v.drawPlane(&rec)

	want := [][6]float64{{2, 0, 22, 0, 2, 98}}
	if len(rec.geo) != len(want) {
		t.Fatalf("the sprite pass made %d draw calls at zoom 2, want %d\ngot: %+v", len(rec.geo), len(want), rec.geo)
	}
	if rec.geo[0] != want[0] {
		t.Errorf("GeoM = %v, want %v — the sprite must be scaled by the camera's zoom, not drawn at native size",
			rec.geo[0], want[0])
	}
}

func TestStaticTexturesAreLazyAndKeyedByFrameIdentity(t *testing.T) {
	v := staticsIdentityViewer(t, true, true)

	if _, _ = v.Statics(); v.staticImages != nil {
		t.Fatalf("Statics() built the texture cache")
	}
	if rects := staticScreenRects(v.staticPlacements(), v.Camera()); len(rects) == 0 || v.staticImages != nil {
		t.Fatalf("the cull built the texture cache (or culled everything: %d rects)", len(rects))
	}
	if passes := v.overlayPasses(); len(passes) == 0 || v.staticImages != nil {
		t.Fatalf("the marker pass built the texture cache (or drew nothing: %d passes)", len(passes))
	}

	var first staticRecorder
	v.drawPlane(&first)
	if len(v.staticImages) != 2 {
		t.Fatalf("after one displaced draw the cache holds %d textures, want 2 (a 10x6 frame and a 4x4 one)",
			len(v.staticImages))
	}

	var again staticRecorder
	v.drawPlane(&again)
	if len(v.staticImages) != 2 {
		t.Errorf("a second draw grew the cache to %d, want 2 — a frame is uploaded once", len(v.staticImages))
	}
	for i := range first.imgs {
		if first.imgs[i] != again.imgs[i] {
			t.Errorf("draw %d submitted a different texture the second time; the cache is not hit", i)
		}
	}

	// The other geometry draws the SAME class from a different cell. SetFlat also
	// shrinks the world to the tile grid's own 128 px, which the 223-tall window
	// then centres, so all three flat placements — (11,-15), (78,0) and (43,17) —
	// clear the top edge and the 10x6 frame arrives from cell (1,1) as well.
	v.SetFlat(true)
	var flat staticRecorder
	v.drawPlane(&flat)
	if len(v.staticImages) != 2 {
		t.Fatalf("the flat geometry grew the cache to %d, want 2 — the key is the frame, not the cell",
			len(v.staticImages))
	}
	if len(flat.imgs) != 3 {
		t.Fatalf("the flat draw made %d calls, want 3", len(flat.imgs))
	}
	if flat.imgs[2] != first.imgs[1] {
		t.Errorf("cell (1,1) got a different texture from cell (0,0) for the one frame they share; " +
			"the cache is keyed by something other than frame identity")
	}
	if flat.imgs[0] != first.imgs[0] || flat.imgs[1] != first.imgs[1] {
		t.Errorf("the flat geometry re-uploaded a frame the displaced one had already cached")
	}
}

func TestStaticArtSwitchGatesTheDrawAndNotTheBuild(t *testing.T) {
	off := staticsIdentityViewer(t, false, true)
	on := staticsIdentityViewer(t, true, true)

	var rec staticRecorder
	off.drawPlane(&rec)
	if len(rec.imgs) != 0 {
		t.Errorf("the art switch is off and the pass still made %d draw calls", len(rec.imgs))
	}
	if off.staticImages != nil {
		t.Errorf("the art switch is off and %d textures were built", len(off.staticImages))
	}

	if len(off.staticPlacements()) != len(on.staticPlacements()) {
		t.Fatalf("art off holds %d placements, art on holds %d — the switch must gate the draw, never the build",
			len(off.staticPlacements()), len(on.staticPlacements()))
	}
	assertScreenRects(t, off.staticMarkerScreenRects(), on.staticMarkerScreenRects())
}

func nativeStaticArms(col, row int) []image.Rectangle {
	cx, cy := 32*col+16, 32*row+16
	return []image.Rectangle{
		image.Rect(cx-3, cy, cx+4, cy+1),
		image.Rect(cx, cy-3, cx+1, cy+4),
	}
}

// staticMarkedCells is the cells the third glyph must mark: exactly the cells of
// the built placement list, in its order — (0,0), (2,0), (1,1) over the
// staticsOverlay fixture, whose remaining nine cells are *no object*, an artless
// class and a byte naming no class.
var staticMarkedCells = []image.Point{{X: 0, Y: 0}, {X: 2, Y: 0}, {X: 1, Y: 1}}

func TestOverlayPassesAppendsTheStaticGlyphLast(t *testing.T) {
	if want := (color.RGBA{R: 0xff, G: 0x20, B: 0x40, A: 0xff}); terrain.StaticMarkerColor != want {
		t.Fatalf("FR-6: StaticMarkerColor = %+v, want the opaque red %+v", terrain.StaticMarkerColor, want)
	}
	if terrain.StaticMarkerColor == terrain.MarkerColor || terrain.StaticMarkerColor == terrain.UnitMarkerColor {
		t.Fatalf("FR-6: the static marker colour must differ from the object and unit ones, " +
			"or pass order cannot be observed by colour")
	}

	v := staticsIdentityViewer(t, true, true)
	cam := v.Camera()
	objects := []image.Point{{X: 0, Y: 0}}
	units := []image.Point{{X: 1, Y: 1}}
	v.SetObjects(true, objects)
	v.SetUnits(true, units)

	passes := v.overlayPasses()
	if len(passes) != 3 {
		t.Fatalf("got %d overlay passes with all three on, want 3 (objects, units, statics)", len(passes))
	}
	if passes[0].Color != terrain.MarkerColor || passes[1].Color != terrain.UnitMarkerColor ||
		passes[2].Color != terrain.StaticMarkerColor {
		t.Fatalf("pass colours are [0]=%+v [1]=%+v [2]=%+v, want objects %+v, units %+v, statics %+v. "+
			"Draw iterates this slice, so the static pass MUST come last: its glyph is a strict subset of both "+
			"shipped crosses, so drawn earlier it is the one that disappears under a coincident marker",
			passes[0].Color, passes[1].Color, passes[2].Color,
			terrain.MarkerColor, terrain.UnitMarkerColor, terrain.StaticMarkerColor)
	}

	proj := cliffProjection()
	assertScreenRects(t, passes[0].Rects, wantDisplacedRects(cam, proj, objects, nativeArms))
	assertScreenRects(t, passes[1].Rects, wantDisplacedRects(cam, proj, units, nativeUnitArms))
	assertScreenRects(t, passes[2].Rects, wantDisplacedRects(cam, proj, staticMarkedCells, nativeStaticArms))
}

func TestStaticMarkerPassFollowsItsOwnSwitchAndTheBuiltList(t *testing.T) {
	t.Run("cross on, art off", func(t *testing.T) {
		v := staticsIdentityViewer(t, false, true)
		passes := v.overlayPasses()
		if len(passes) != 1 || passes[0].Color != terrain.StaticMarkerColor {
			t.Fatalf("got %d passes, want exactly the static one — either flag must be usable without the other", len(passes))
		}
		assertScreenRects(t, passes[0].Rects, wantDisplacedRects(v.Camera(), cliffProjection(), staticMarkedCells, nativeStaticArms))
	})

	t.Run("art on, cross off", func(t *testing.T) {
		v := staticsIdentityViewer(t, true, false)
		if passes := v.overlayPasses(); len(passes) != 0 {
			t.Fatalf("got %d overlay passes with the cross off, want none — the art draws no marker", len(passes))
		}
		if rects := v.staticMarkerScreenRects(); rects != nil {
			t.Fatalf("the cross is off and %d marker rects came back", len(rects))
		}
	})

	t.Run("no bundle", func(t *testing.T) {
		v, err := NewViewerWithStatics("m", staticsGrid(), &terrain.Tileset{}, nil, true, true, terrain.AnimGateTiles, nil, false)
		if err != nil {
			t.Fatalf("NewViewerWithStatics: %v", err)
		}
		layoutViewport(v, cliffW*terrain.CellSize, cliffCanvasH)
		if passes := v.overlayPasses(); len(passes) != 0 {
			t.Fatalf("got %d overlay passes with no bundle, want none: no cell resolves, so none is marked", len(passes))
		}
	})

	t.Run("the marked cells are the placements', not the grid's", func(t *testing.T) {
		v := staticsIdentityViewer(t, true, true)
		got := staticMarkerCells(v.staticPlacements())
		if len(got) != len(staticMarkedCells) {
			t.Fatalf("marking %d cells over a 12-cell object grid, want %d — a cross is owed to a cell that "+
				"resolves to a drawable frame and to no other", len(got), len(staticMarkedCells))
		}
		for i, want := range staticMarkedCells {
			if got[i] != want {
				t.Errorf("marked cell %d = %v, want %v (placement order)", i, got[i], want)
			}
		}
	})
}

// TestViewerDrawRunsTheSpriteAndMarkerPasses is the integration end of SC-8's
// window half: Draw itself reaches the sprite pass, over a real *ebiten.Image.
//
// What it can assert is bounded by the engine. An *ebiten.Image's pixels cannot
// be read back before the game starts, so the ORDER Draw composes its three
// stages in — terrain, then sprites, then markers — is not observable from here;
// what is observable is that the sprite pass ran at all, which the textures Draw
// left behind say, and that the whole path completes without a graphics context.
func TestViewerDrawRunsTheSpriteAndMarkerPasses(t *testing.T) {
	v := staticsIdentityViewer(t, true, true)
	v.SetObjects(true, []image.Point{{X: 0, Y: 0}})
	v.SetUnits(true, []image.Point{{X: 1, Y: 1}})

	screen := ebiten.NewImage(cliffW*terrain.CellSize, cliffCanvasH)
	v.Draw(screen)

	if len(v.staticImages) != 2 {
		t.Fatalf("after Draw the texture cache holds %d frames, want 2 — Draw did not reach the sprite pass",
			len(v.staticImages))
	}

	// And with the art off, Draw builds none: a viewer drawing the cross alone
	// still never touches the GPU for a sprite.
	crossOnly := staticsIdentityViewer(t, false, true)
	crossOnly.Draw(screen)
	if crossOnly.staticImages != nil {
		t.Errorf("Draw built %d sprite textures with the art switch off", len(crossOnly.staticImages))
	}
}
