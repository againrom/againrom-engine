package ui

import (
	"image"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

// TestStaticScreenRectsCarryTheMirrorBitThroughTheCull — SC-6: the mirror
// bit rides staticScreenRect off its own placement exactly as the frame
// pointer does, so the cull cannot desynchronise the frame-mirror pair.
//
// The fixture is the object layer's own cull table with bits planted on
// placements 0, 3 and 5.
func TestStaticScreenRectsCarryTheMirrorBitThroughTheCull(t *testing.T) {
	places := cullPlacements()
	places[0].Mirror = true
	places[3].Mirror = true
	places[5].Mirror = true

	// The fixture must discriminate: pairing the survivors with the bits of
	// the first three placements has to disagree with the survivors' own.
	byIndex := []bool{places[0].Mirror, places[1].Mirror, places[2].Mirror}
	correct := []bool{places[2].Mirror, places[3].Mirror, places[5].Mirror}
	if reflect.DeepEqual(byIndex, correct) {
		t.Fatalf("fixture does not discriminate: a parallel slice paired by index would carry %v too", correct)
	}

	got := staticScreenRects(places, cullCamera(t, 1))
	want := []staticScreenRect{
		{screenRect: screenRect{X: 100, Y: 50, W: 10, H: 6}, Frame: places[2].Frame, Mirror: false},
		{screenRect: screenRect{X: 120, Y: 60, W: 10, H: 6}, Frame: places[3].Frame, Mirror: true},
		{screenRect: screenRect{X: 60, Y: 20, W: 40, H: 380}, Frame: places[5].Frame, Mirror: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d survivors, want %d\ngot: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("survivor %d = %+v, want %+v — the bit must be its own placement's, culled entries included",
				i, got[i], want[i])
		}
	}
}

// TestMirroredAndPlainPlacementsAreIdenticalRectangles — SC-6 (AC-6): one
// frame placed mirrored and plain yields IDENTICAL rectangles at the drawn
// frame's own size — placement, anchor and ground point never read the bit
// — and the displaced placement differs from the flat one by exactly
// -(lift + originY) in Y and by zero in X. Driven through the placement path
// and the screen-rect path, against hand literals.
func TestMirroredAndPlainPlacementsAreIdenticalRectangles(t *testing.T) {
	artA := entityArtA() // canvas 64x64, centre (32,60), frame 10x6 -> anchor (5,31)
	f := artA.Frames[0]

	// The literals, at cell (0,1): destX = 16 - 5 = 11;
	// destY flat = 48 - 31 = 17; destY displaced = 48 - 31 - 63 + 95 = 49.
	const lift, originY = 63, -95

	plain, ok1 := terrain.UnitPlace(0, 1, artA, f, false, lift, originY)
	mirrored, ok2 := terrain.UnitPlace(0, 1, artA, f, true, lift, originY)
	if !ok1 || !ok2 {
		t.Fatalf("UnitPlace refused a drawable class (plain %v, mirrored %v)", ok1, ok2)
	}

	// Identical rectangles, at the frame's 10x6 and not the canvas's 64x64.
	if want := image.Rect(11, 49, 21, 55); plain.Rect() != want || mirrored.Rect() != want {
		t.Fatalf("displaced rects: plain %v, mirrored %v, want both %v — the drawn frame's own size, "+
			"identical at either bit", plain.Rect(), mirrored.Rect(), want)
	}
	if want := image.Pt(16, 80); plain.Ground() != want || mirrored.Ground() != want {
		t.Fatalf("ground points: plain %v, mirrored %v, want both %v", plain.Ground(), mirrored.Ground(), want)
	}
	// The bit is the WHOLE difference: clearing it makes the two placements
	// one value, so nothing else can have read it.
	cleared := mirrored
	cleared.Mirror = false
	if !mirrored.Mirror || cleared != plain {
		t.Fatalf("mirrored placement %+v differs from plain %+v beyond the bit", mirrored, plain)
	}

	// The displaced-minus-flat clause, as literals: (0, 32), and 32 IS
	// -(63 + (-95)).
	flat, _ := terrain.UnitPlace(0, 1, artA, f, true, 0, 0)
	if want := image.Rect(11, 17, 21, 23); flat.Rect() != want {
		t.Fatalf("flat rect %v, want %v", flat.Rect(), want)
	}
	if dx, dy := mirrored.TopLeft.X-flat.TopLeft.X, mirrored.TopLeft.Y-flat.TopLeft.Y; dx != 0 || dy != 32 {
		t.Fatalf("displaced minus flat = (%d,%d), want (0,32) = (0, -(lift+originY)) at lift 63, originY -95", dx, dy)
	}

	// The screen-rect path: both placements arrive as one rectangle differing
	// only in the bit. The camera is at the origin at zoom 1, so the screen
	// literals are the world ones.
	cam := camera.New(100, 80, 400, 300)
	if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
		t.Fatalf("camera at (%v,%v) zoom %v, want the origin at native zoom", cam.X, cam.Y, cam.Zoom)
	}
	rects := staticScreenRects([]terrain.StaticPlacement{plain, mirrored}, cam)
	wantRects := []staticScreenRect{
		{screenRect: screenRect{X: 11, Y: 49, W: 10, H: 6}, Frame: f, Mirror: false},
		{screenRect: screenRect{X: 11, Y: 49, W: 10, H: 6}, Frame: f, Mirror: true},
	}
	if len(rects) != 2 || rects[0] != wantRects[0] || rects[1] != wantRects[1] {
		t.Fatalf("screen rects %+v, want %+v — one rectangle, two bits", rects, wantRects)
	}

	// And through the live viewer path — the same cell over the cliff
	// fixture, whose own AnchorHeight(0,1)/MinV are the 63/-95 handed above —
	// the bit passes from MapEntity to placement with the rectangle untouched.
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	v.SetEntities([]MapEntity{
		{Cell: image.Pt(0, 1), Art: artA, Frame: f},
		{Cell: image.Pt(0, 1), Art: artA, Frame: f, Mirror: true},
	})
	sprites, _, _ := v.entityLayer()
	if len(sprites) != 2 {
		t.Fatalf("the layer built %d placements, want 2", len(sprites))
	}
	if sprites[0].TopLeft != image.Pt(11, 49) || sprites[0].Mirror ||
		sprites[1].TopLeft != image.Pt(11, 49) || !sprites[1].Mirror {
		t.Fatalf("viewer placements %+v — want both at (11,49), the second mirrored", sprites)
	}
}

// TestSpriteGeoMReflectsInsideItsOwnRectangle — SC-6: the mirrored
// transform is Scale(-zoom, zoom) then Translate(s.X + s.W, s.Y), the plain
// one Scale(zoom, zoom) then Translate(s.X, s.Y), each pinned as matrix
// literals; and applying either to the frame's two vertical edges lands on
// the SAME rectangle's two edges — swapped under the mirror, in place
// without it. Y is untouched by the bit in every element.
func TestSpriteGeoMReflectsInsideItsOwnRectangle(t *testing.T) {
	elements := func(g ebiten.GeoM) [6]float64 {
		return [6]float64{
			g.Element(0, 0), g.Element(0, 1), g.Element(0, 2),
			g.Element(1, 0), g.Element(1, 1), g.Element(1, 2),
		}
	}

	// The routing fixture's rect at the identity camera: (11,49) 10x6.
	plain := staticScreenRect{screenRect: screenRect{X: 11, Y: 49, W: 10, H: 6}}
	mirrored := plain
	mirrored.Mirror = true

	if got, want := elements(spriteGeoM(plain, 1)), [6]float64{1, 0, 11, 0, 1, 49}; got != want {
		t.Fatalf("plain GeoM = %v, want %v (scale(zoom) then translate to the rect's top-left)", got, want)
	}
	if got, want := elements(spriteGeoM(mirrored, 1)), [6]float64{-1, 0, 21, 0, 1, 49}; got != want {
		t.Fatalf("mirrored GeoM = %v, want %v (scale(-zoom, zoom) then translate by s.X + s.W)", got, want)
	}

	// The same rect as the cull emits it at zoom 2 — (22,98) 20x12 for a 10x6
	// frame — because s.W is already the SCALED width the translate must use.
	zoomed := staticScreenRect{screenRect: screenRect{X: 22, Y: 98, W: 20, H: 12}, Mirror: true}
	if got, want := elements(spriteGeoM(zoomed, 2)), [6]float64{-2, 0, 42, 0, 2, 98}; got != want {
		t.Fatalf("mirrored GeoM at zoom 2 = %v, want %v", got, want)
	}

	// The frame's edges land on the rectangle's edges: 0 and 10 (the frame's
	// own width) map to 11 and 21 both ways round — the mirror swaps them and
	// moves nothing else, which is "reflected within their own rectangle".
	for _, c := range []struct {
		name         string
		s            staticScreenRect
		leftX, right float64
	}{
		{"plain", plain, 11, 21},
		{"mirrored", mirrored, 21, 11},
	} {
		g := spriteGeoM(c.s, 1)
		x0, y0 := g.Apply(0, 0)
		x1, y1 := g.Apply(10, 0)
		if x0 != c.leftX || x1 != c.right || y0 != 49 || y1 != 49 {
			t.Errorf("%s: edges map to (%v,%v) and (%v,%v), want x %v and %v at y 49",
				c.name, x0, y0, x1, y1, c.leftX, c.right)
		}
	}
}

// TestMirroredDrawSharesTheTextureAndCacheKey — SC-6: one frame drawn
// plain at one cell and mirrored at another goes to the GPU ONCE — the
// texture cache holds a single entry under the frame's own pointer, so the
// mirror widened no key, built no second cache and uploaded nothing of its
// own; a second draw grows nothing.
func TestMirroredDrawSharesTheTextureAndCacheKey(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	artA := entityArtA()
	v.SetEntities([]MapEntity{
		{Cell: image.Pt(0, 1), Art: artA, Frame: artA.Frames[0]},               // plain at (11,49)
		{Cell: image.Pt(2, 0), Art: artA, Frame: artA.Frames[0], Mirror: true}, // mirrored at (75,17)
	})

	// Both survive the cull, one mirrored — or the draw below would not
	// exercise both sides of the branch.
	// Row order decides the band since 0068, so the cell-(2,0) mirrored entry
	// precedes the cell-(0,1) plain one although its id is higher. What this case
	// needs is that both survive the cull and that exactly one is mirrored.
	band := v.planeSprites()
	if len(band) != 2 || !band[0].Mirror || band[1].Mirror {
		t.Fatalf("fixture does not discriminate: want a content band holding a mirrored then a plain entry, got %+v", band)
	}

	screen := ebiten.NewImage(cliffW*terrain.CellSize, cliffCanvasH)
	v.Draw(screen)
	if len(v.staticImages) != 1 {
		t.Fatalf("after drawing the frame plain and mirrored the cache holds %d textures, want 1 — "+
			"the mirror is a GeoM, never a second upload", len(v.staticImages))
	}
	// 0044's key is (frame, row); the mirror is still no part of it.
	if v.staticImages[v.spriteKey(artA.Frames[0])] == nil {
		t.Fatalf("the texture is not keyed under the frame's own pointer identity")
	}
	v.Draw(screen)
	if len(v.staticImages) != 1 {
		t.Fatalf("a second draw grew the cache to %d, want 1", len(v.staticImages))
	}
}

// TestEveryInMapEntityDrawsExactlyOneItem — SC-8: over a mixed snapshot
// — resolved art plain and mirrored, a resolved class with no art, an id
// that named no class — every in-map entity lands in exactly one list,
// sprite or square, and both off-map entities land in neither; the pass
// slice then carries exactly one drawn item per in-map entity, with the
// whole map in view so the cull removes nothing.
func TestEveryInMapEntityDrawsExactlyOneItem(t *testing.T) {
	v := overlayViewer(t, 10, 10, 320, 320)
	artA, artB := entityArtA(), entityArtB()
	noFrame := &terrain.UnitClass{Width: 8, Height: 8, CenterX: 4, CenterY: 7}
	v.SetEntities([]MapEntity{
		{Cell: image.Pt(1, 1), Art: artA, Frame: artA.Frames[0]},               // sprite
		{Cell: image.Pt(2, 2), Art: artB, Frame: artB.Frames[0], Mirror: true}, // sprite, mirrored
		{Cell: image.Pt(3, 3)},                                    // no class: square
		{Cell: image.Pt(4, 4), Art: noFrame},                      // no art: square
		{Cell: image.Pt(-1, 2), Art: artA, Frame: artA.Frames[0]}, // off-map with art: nothing
		{Cell: image.Pt(10, 10)},                                  // off-map artless: nothing
	})

	sprites, squares, _ := v.entityLayer()
	if len(sprites)+len(squares) != 4 {
		t.Fatalf("%d sprites + %d squares over 4 in-map entities, want exactly one item each",
			len(sprites), len(squares))
	}
	wantCells := []image.Point{{X: 1, Y: 1}, {X: 2, Y: 2}}
	for i, w := range wantCells {
		if sprites[i].Cell != w {
			t.Errorf("sprite %d at %v, want %v", i, sprites[i].Cell, w)
		}
	}
	if sprites[0].Mirror || !sprites[1].Mirror {
		t.Errorf("sprite mirrors [%v %v], want [false true] — the bit rides its own entity", sprites[0].Mirror, sprites[1].Mirror)
	}
	if want := []image.Point{{X: 3, Y: 3}, {X: 4, Y: 4}}; !reflect.DeepEqual(cellsOf(squares), want) {
		t.Errorf("squares %v, want %v — one square per artless in-map entity, none for off-map",
			cellsOf(squares), want)
	}

	// The drawn items: one sprite rect or one square rect per in-map entity,
	// nothing culled at this view, nothing for the two off-map entities.
	passes := v.overlayPasses()
	if len(passes) != 1 {
		t.Fatalf("got %d passes, want the square pass alone — the sprites are the content band's (0068 DD-5)", len(passes))
	}
	if got := len(v.planeSprites()) + len(passes[0].Rects); got != 4 {
		t.Fatalf("the frame draws %d items over 4 in-map entities, want 4 (P-2: exactly one each)", got)
	}
	if passes[0].Color != terrain.EntityMarkerColor {
		t.Fatalf("the square pass colour is %+v, want the unchanged %+v", passes[0].Color, terrain.EntityMarkerColor)
	}
}

func TestNilBundlePassSliceIsTheArtlessDerivationDriven(t *testing.T) {
	// The seam's snapshot at tick i, as cells: a walker crossing the map row
	// by column, and a stander. artless() spells the nil-bundle push over
	// them: Cell alone, Art, Frame and Mirror all zero.
	snapshot := func(tick int) []image.Point {
		return []image.Point{{X: tick % cliffW, Y: 1}, {X: 2, Y: 3}}
	}
	// The artless derivation: a FRESH viewer handed only these cells. Nothing
	// it returns can depend on a tick, because it never saw one.
	fresh := func(cells []image.Point) []overlayPass {
		v := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
		v.SetEntities(artless(cells))
		return v.overlayPasses()
	}

	driven := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	screen := ebiten.NewImage(cliffW*terrain.CellSize, cliffCanvasH)

	// Tick 0: the map has opened, nothing has run.
	driven.SetEntities(artless(snapshot(0)))
	tick0 := driven.overlayPasses()
	if len(tick0) == 0 {
		t.Fatalf("the tick-0 screen holds no pass at all; an empty comparison would prove nothing")
	}
	if !reflect.DeepEqual(tick0, fresh(snapshot(0))) {
		t.Fatalf("tick 0: the nil-bundle pass slice differs from the artless derivation\ngot:  %+v\nwant: %+v",
			tick0, fresh(snapshot(0)))
	}
	if band := driven.planeSprites(); len(band) != 0 {
		t.Fatalf("tick 0: the content band carries %+v on a nil-bundle screen, want none", band)
	}
	driven.Draw(screen)

	// k driven ticks: one push and one draw per tick, as the map screen runs.
	const k = 5
	for i := 1; i <= k; i++ {
		driven.SetEntities(artless(snapshot(i)))
		driven.Draw(screen)
	}
	if snapshot(k)[0] == snapshot(0)[0] {
		t.Fatalf("fixture does not discriminate: the walker stands where it started, so no state could have moved anything")
	}

	got := driven.overlayPasses()
	want := fresh(snapshot(k))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("after %d driven ticks the nil-bundle pass slice differs from the artless derivation — "+
			"render-side state reached an artless screen\ngot:  %+v\nwant: %+v", k, got, want)
	}
	if band := driven.planeSprites(); len(band) != 0 {
		t.Fatalf("the content band carries %+v after %d driven ticks, want none", band, k)
	}
	if len(got) == 0 || len(got[0].Rects) == 0 {
		t.Fatalf("the driven screen drew nothing; an empty comparison would prove nothing")
	}
	if driven.staticImages != nil {
		t.Fatalf("%d sprite textures exist after %d artless draws, want none — with no bundle no texture is ever built",
			len(driven.staticImages), k)
	}

	// The control: the identical comparison fails the moment one entity
	// resolves, so the equalities above could have failed.
	control := identityViewer(t, cliffGrid(), cliffW*terrain.CellSize, cliffCanvasH)
	control.SetEntities([]MapEntity{withFrame(image.Pt(0, 1), entityArtA()), {Cell: image.Pt(2, 3)}})
	if reflect.DeepEqual(control.overlayPasses(), fresh(snapshot(0))) {
		t.Fatalf("the probe does not discriminate: a resolved entity produced the artless pass slice, " +
			"so this comparison could never have seen a leak")
	}
}
