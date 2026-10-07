package ui

// The window's half of the shadow pass, as drawn.
//
// The fixture carries one of each caster — an object, a structure and a unit
// — spaced apart on one flat grid so their shadows and bodies cannot overlap,
// viewed through an 8-cell window opened at the world origin so the camera's
// POSITION is the identity (X=0, Y=0, Zoom=1) and every recorded screen
// coordinate IS a world one, even though the grid itself is wider than the
// window (shadowGrid's own comment: the extra width is what gives the cull
// test room to pan). The object's class carries a Frames slice (unlike
// planeStatics' in structures_test.go, which sets only the single Frame
// field a plain body draw needs) because terrain.ObjectShadowPlace anchors
// against Frames[0] and refuses a class that has none.
//
// No game data is read and no window opens. Ebitengine's draw-side calls
// (NewImage, NewImageFromImage, DrawImage) do run before the game starts, so
// the recorder pattern statics_test.go established — a plain Go struct
// standing in for the draw target — is what makes the pass's arrangement,
// blend and alpha observable at all.

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/terrain"
)

func shadowObjectFrame() *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: 10, Height: 6, Pixels: make([]terrain.StaticPixel, 60)}
	for i := range f.Pixels {
		f.Pixels[i] = terrain.StaticPixel{Index: 1, Opaque: true}
	}
	return f
}

// shadowObjectSet is one drawable class — canvas 64x64, centre (32,60), so
// anchor (5,31), the object bundle's own worked example (statics_test.go,
// entity_overlay_test.go) — carrying BOTH Frame and Frames, unlike every
// other StaticClass fixture in this package.
func shadowObjectSet() *terrain.StaticSet {
	f := shadowObjectFrame()
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{
		Width: 64, Height: 64, CenterX: 32, CenterY: 60,
		Frame: f, Frames: []*terrain.StaticFrame{f},
	}
	return set
}

func shadowStructureSet() *terrain.StructureSet {
	set := new(terrain.StructureSet)
	set.Classes[1] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{structureFrame()},
	}
	return set
}

// shadowGrid is 20x20, flat (no altitude layer, so Mode() is ModeFlat and a
// world pixel is a cell times CellSize): one object at (2,2) and one
// structure record at (4,4), left apart from each other and from the unit
// cell shadowUnitCell names so no two shadows or bodies can coincide.
//
// The grid is wider than the WINDOW shadowViewer opens (8 cells, 256px) on
// purpose — TestShadowDrawsCullsAPlacementTheViewDoesNotReach needs slack to
// pan the camera off the fixture and back; an 8x8 grid matched exactly to an
// 8-cell window has none; Camera.Clamp pins X and Y to 0 the moment the world
// is no wider than the view.
func shadowGrid() terrain.Grid {
	const w, h = 20, 20
	overlay := make([]uint8, w*h)
	overlay[2*w+2] = 1
	return terrain.Grid{
		Width:   w,
		Height:  h,
		Tiles:   make([]uint16, w*h),
		Overlay: overlay,
		Structures: []terrain.StructureRecord{
			{X: 4 << 8, Y: 4 << 8, Key: 1},
		},
	}
}

// shadowViewer builds shadowGrid's viewer through an 8-cell window positioned
// at the world origin, so the camera's position is the identity (X=0, Y=0,
// Zoom=1) and every recorded translate IS a world coordinate, exactly as the
// planeViewer/identityViewer precedent — even though the grid itself is
// wider than the window (shadowGrid's own comment).
func shadowViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewerWithStatics("m", shadowGrid(), &terrain.Tileset{}, shadowObjectSet(), true, false,
		terrain.AnimGateTiles, shadowStructureSet(), true)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	layoutViewport(v, 8*terrain.CellSize, 8*terrain.CellSize)
	cam := v.Camera()
	if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
		t.Fatalf("camera is (%v,%v) at zoom %v, want the identity", cam.X, cam.Y, cam.Zoom)
	}
	if v.Mode() != ModeFlat {
		t.Fatalf("Mode() = %v, want flat; the fixture carries no altitude layer", v.Mode())
	}
	return v
}

// shadowUnitCell and shadowUnitBody are the fixture's third caster: a unit at
// (6,1) drawn from entityArtA() (canvas 64x64, centre (32,60), frame 10x6,
// anchor (5,31) — entity_overlay_test.go), well clear of the object at (2,2)
// and the structure at (4,4). Flat, its body stands at
// (6*32+16-5, 1*32+16-31) = (203, 17).
var shadowUnitCell = image.Pt(6, 1)

// shadowRecorder is a draw target that records everything drawArt or
// drawShadows submits: the texture, the GeoM as six elements (statics_test.go's
// own shape), the blend and the colour scale's own alpha — the two fields
// staticRecorder does not carry, and the ones a shadow draw is identified and
// checked by.
type shadowRecorder struct {
	imgs  []*ebiten.Image
	geo   [][6]float64
	blend []ebiten.Blend
	alpha []float32
}

func (r *shadowRecorder) DrawImage(img *ebiten.Image, op *ebiten.DrawImageOptions) {
	r.imgs = append(r.imgs, img)
	r.geo = append(r.geo, [6]float64{
		op.GeoM.Element(0, 0), op.GeoM.Element(0, 1), op.GeoM.Element(0, 2),
		op.GeoM.Element(1, 0), op.GeoM.Element(1, 1), op.GeoM.Element(1, 2),
	})
	r.blend = append(r.blend, op.Blend)
	r.alpha = append(r.alpha, op.ColorScale.A())
}

// TestShadowDrawsOrderCountAndGating — AC-8: the pass submits one draw per
// drawable placement, all of them before the first content-plane draw, and
// showStaticArt/showStructureArt gate the object's and the structure's own
// shadow exactly as they gate their bodies while the unit's is never gated.
//
// A shadow draw is told apart from a content draw by shadowBlend and by
// ColorScale.A() < 1 — a content draw leaves both at DrawImageOptions' own
// zero value (the default blend, alpha scale 1) — which is what lets this
// test read the ORDER off a single recorder shared by the whole of drawArt
// rather than by comparing two separate passes' own output.
func TestShadowDrawsOrderCountAndGating(t *testing.T) {
	isShadow := func(rec *shadowRecorder, i int) bool { return rec.blend[i] == shadowBlend }

	t.Run("both switches on", func(t *testing.T) {
		v := shadowViewer(t)
		v.SetEntities([]MapEntity{withFrame(shadowUnitCell, entityArtA())})

		var rec shadowRecorder
		v.drawArt(&rec)

		if len(rec.geo) != 6 {
			t.Fatalf("drawArt made %d draw calls, want 6 (3 shadows + 3 bodies)\ngeo: %v", len(rec.geo), rec.geo)
		}
		// Structure shadow first; row1 unit shadow/body, row2 object
		// shadow/body, row4 structure body (ANIM-CELL-085).
		for i, want := range []bool{true, true, false, true, false, false} {
			if got := isShadow(&rec, i); got != want {
				t.Errorf("draw %d shadow=%v want=%v", i, got, want)
			}
		}
	})

	t.Run("both switches off, the unit's shadow alone", func(t *testing.T) {
		v := shadowViewer(t)
		v.showStaticArt, v.showStructureArt = false, false
		v.SetEntities([]MapEntity{withFrame(shadowUnitCell, entityArtA())})

		var rec shadowRecorder
		v.drawArt(&rec)

		// One shadow (the unit's, never gated) and one body (the unit's own;
		// planeSprites never gates an entity either).
		if len(rec.geo) != 2 {
			t.Fatalf("drawArt made %d draw calls with both switches off, want 2 (the unit's shadow and body)\n"+
				"geo: %v", len(rec.geo), rec.geo)
		}
		if !isShadow(&rec, 0) {
			t.Errorf("draw 0 is not a shadow draw — the unit's shadow must still lead, ungated")
		}
		if isShadow(&rec, 1) {
			t.Errorf("draw 1 carries shadowBlend — want the unit's own body")
		}
	})

	t.Run("object switch off alone", func(t *testing.T) {
		v := shadowViewer(t)
		v.showStaticArt = false

		draws := v.shadowDraws()
		if len(draws) != 1 {
			t.Fatalf("shadowDraws() = %d entries with the object switch off, want 1 (the structure's)", len(draws))
		}
	})

	t.Run("structure switch off alone", func(t *testing.T) {
		v := shadowViewer(t)
		v.showStructureArt = false

		draws := v.shadowDraws()
		if len(draws) != 1 {
			t.Fatalf("shadowDraws() = %d entries with the structure switch off, want 1 (the object's)", len(draws))
		}
	})
}

func TestShadowDrawsBlendAndAlphaPerKind(t *testing.T) {
	v := shadowViewer(t)
	v.SetEntities([]MapEntity{withFrame(shadowUnitCell, entityArtA())})

	sun := v.Sun()
	wantObject := float32(sun.ShroudObject) / 16
	if sun.ShroudUnit == sun.ShroudObject {
		t.Fatalf("fixture does not discriminate: ShroudObject and ShroudUnit both read %d", sun.ShroudObject)
	}

	var rec shadowRecorder
	v.drawShadows(&rec)

	if len(rec.geo) != 3 {
		t.Fatalf("drawShadows made %d draw calls, want 3", len(rec.geo))
	}
	kinds := []string{"object", "structure", "unit"}
	want := []float32{wantObject, wantObject, wantObject}
	for i, kind := range kinds {
		if rec.blend[i] != shadowBlend {
			t.Errorf("%s: blend %+v, want shadowBlend %+v (FR-17)", kind, rec.blend[i], shadowBlend)
		}
		if rec.alpha[i] != want[i] {
			t.Errorf("%s: ColorScale alpha %v, want %v (FR-14)", kind, rec.alpha[i], want[i])
		}
	}
}

// TestShadowDrawsCarrySlopeAndPivotForAllThreeCasters — AC-8's first
// clause: every one of the three casters' entries carries the slope
// terrain.ShadowSlope(theta) returns and the pivot terrain.ShadowPivotRow
// gives for the frame it draws, and neither is ever zero.
func TestShadowDrawsCarrySlopeAndPivotForAllThreeCasters(t *testing.T) {
	v := shadowViewer(t)
	v.SetEntities([]MapEntity{withFrame(shadowUnitCell, entityArtA())})

	draws := v.shadowDraws()
	if len(draws) != 3 {
		t.Fatalf("shadowDraws() = %d entries, want 3", len(draws))
	}

	slope := terrain.ShadowSlope(v.Sun().Theta)
	if slope == 0 {
		t.Fatalf("fixture does not discriminate: ShadowSlope(theta) is 0")
	}

	for i, name := range []string{"object", "structure", "unit"} {
		d := draws[i]
		if d.Slope != slope {
			t.Errorf("%s: Slope = %v, want ShadowSlope(theta) = %v (FR-14)", name, d.Slope, slope)
		}
		if want := terrain.ShadowPivotRow(d.Frame); d.PivotRow != want {
			t.Errorf("%s: PivotRow = %d, want %d — the drawn frame's own height (FR-4, FR-14)", name, d.PivotRow, want)
		}
	}
}

func TestDrawShadowsMirrorLeansTheSameWayAsUnmirrored(t *testing.T) {
	frame := &terrain.StaticFrame{Width: 10, Height: 6}
	base := shadowDraw{
		screenRect: screenRect{X: 100, Y: 50, W: 10, H: 6},
		Frame:      frame,
		Slope:      0.5,
	}
	mirrored := base
	mirrored.Mirror = true

	lean := func(d shadowDraw) float64 {
		g := shadowGeoM(d, 1)
		topX, topY := g.Apply(0, 0)
		botX, botY := g.Apply(0, float64(frame.Height-1))
		if topY >= botY {
			t.Fatalf("top row Y = %v, bottom row Y = %v, want the top strictly above the bottom", topY, botY)
		}
		return topX - botX
	}

	u, m := lean(base), lean(mirrored)
	if u == 0 || m == 0 {
		t.Fatalf("fixture does not discriminate: lean(unmirrored) = %v, lean(mirrored) = %v", u, m)
	}
	if (u > 0) != (m > 0) {
		t.Errorf("unmirrored top-minus-bottom X = %v, mirrored = %v — opposite signs mean the mirrored "+
			"shadow leans the other way; the mirror must compose INSIDE shadowGeoM's local matrix, not by "+
			"scaling the whole matrix by -1 around it (FR-15)", u, m)
	}
}

func TestShadowDrawsCullsTheStructureOnItsWidenedRectangleNotThePlainOne(t *testing.T) {
	v := shadowViewer(t)
	v.showStaticArt = false // isolate the structure arm; no entities are set, so no unit shadow either

	theta := v.Sun().Theta
	slope := terrain.ShadowSlope(theta)
	structShift := terrain.StructureShadowShift(theta, 1, 0, 0)      // shadowStructureSet: FullHeight 1, gridRow 0, ShadowY 0
	plainMaxX := 4*terrain.CellSize + structShift + terrain.CellSize // shadowGrid's structure at cell (4,4)

	pivotRow := terrain.CellSize // terrain.ShadowPivotRow(structureFrame()) == structureFrame().Height == CellSize
	top := terrain.ShadowRowOffset(slope, pivotRow, 0)
	bottom := terrain.ShadowRowOffset(slope, pivotRow, pivotRow-1)
	hi := top
	if bottom > hi {
		hi = bottom
	}
	if hi <= 0 {
		t.Fatalf("fixture does not discriminate: the widened rectangle does not grow past the plain one (hi=%d)", hi)
	}

	v.Camera().Pan(float64(plainMaxX), 0)
	if got := len(v.shadowDraws()); got != 1 {
		t.Fatalf("shadowDraws() = %d entries at the plain rectangle's own right edge, want 1 — the WIDENED "+
			"rectangle must still reach the view here even though the plain one no longer would (FR-16)", got)
	}

	v.Camera().Pan(float64(hi), 0)
	if got := len(v.shadowDraws()); got != 0 {
		t.Fatalf("shadowDraws() = %d entries past the widened rectangle's own right edge, want 0", got)
	}
}

// TestShadowDrawsSkipsVariableSizeAndFramelessStructures — AC-8: "a
// VariableSize structure and a frameless placement submit none". The three
// entries are seeded directly onto v.structuresFlat, bypassing
// terrain.StructurePlacements — which never emits a VariableSize entry at all
// (structures.go's own builder loop) — so this exercises shadowDraws' OWN
// reliance on terrain.StructureShadowPlace's ok return rather than a property
// already proven at the builder.
func TestShadowDrawsSkipsVariableSizeAndFramelessStructures(t *testing.T) {
	v := shadowViewer(t)
	v.showStaticArt = false // isolate the structure list

	drawable := &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{structureFrame()}}
	variableSize := &terrain.StructureClass{TileWidth: 1, TileHeight: 1, FullHeight: 1, VariableSize: true,
		Frames: []*terrain.StaticFrame{structureFrame()}}

	// All three well within the 256x256 window shadowViewer opens at the
	// world origin (shadowGrid's own world is wider, but the camera has not
	// moved off it).
	v.structuresFlat = []terrain.StructurePlacement{
		{Cell: image.Pt(0, 0), TopLeft: image.Pt(0, 100), GridIndex: 0, Class: drawable, Frame: drawable.Frames[0]},
		{Cell: image.Pt(1, 0), TopLeft: image.Pt(60, 100), GridIndex: 0, Class: variableSize, Frame: variableSize.Frames[0]},
		{Cell: image.Pt(2, 0), TopLeft: image.Pt(120, 100), GridIndex: 0, Class: drawable, Frame: nil},
	}
	v.structureAnimFlat = nil

	draws := v.shadowDraws()
	if len(draws) != 1 {
		t.Fatalf("shadowDraws() = %d entries, want 1 (the VariableSize and the frameless entries submit none)\n%+v",
			len(draws), draws)
	}
	if draws[0].Frame != drawable.Frames[0] {
		t.Errorf("the surviving entry's frame is not the drawable placement's own")
	}
}

func TestShadowDrawsCullsAPlacementTheViewDoesNotReach(t *testing.T) {
	v := shadowViewer(t)
	v.SetEntities([]MapEntity{withFrame(shadowUnitCell, entityArtA())})

	if got := len(v.shadowDraws()); got != 3 {
		t.Fatalf("shadowDraws() = %d entries at the identity camera, want 3", got)
	}

	v.Camera().Pan(100000, 100000)
	if got := len(v.shadowDraws()); got != 0 {
		t.Fatalf("shadowDraws() = %d entries panned far off the fixture, want 0", got)
	}

	v.Camera().Pan(-100000, -100000)
	if got := len(v.shadowDraws()); got != 3 {
		t.Fatalf("shadowDraws() = %d entries panned back to the origin, want 3 — the cull must re-run every call",
			got)
	}
}

func TestShadowMaskCacheHoldsOneEntryPerFrameAcrossTwoHours(t *testing.T) {
	v := shadowViewer(t)
	v.SetEntities([]MapEntity{withFrame(shadowUnitCell, entityArtA())})

	if v.shadowMasks != nil {
		t.Fatalf("construction built %d shadow masks, want none — the cache is lazy", len(v.shadowMasks))
	}

	var first shadowRecorder
	v.drawShadows(&first)
	if len(v.shadowMasks) != 3 {
		t.Fatalf("after one draw the mask cache holds %d entries, want 3 (one per caster's own frame)",
			len(v.shadowMasks))
	}

	// A different hour: a Light the fixture's own seed disagrees with in both
	// Theta (which moves the object's shear and every shift) and the two
	// shroud indices (which move every alpha) — so this is a genuinely
	// different sun and not a relight that happens to land on the same one.
	v.sun = terrain.Light{Theta: v.sun.Theta + 0.2, ShroudObject: 6, ShroudUnit: 3}

	var second shadowRecorder
	v.drawShadows(&second)
	if len(v.shadowMasks) != 3 {
		t.Errorf("after the second hour's draw the mask cache holds %d entries, want 3 still — the mask depends "+
			"on the frame alone", len(v.shadowMasks))
	}
	if len(second.imgs) != 3 || len(first.imgs) != 3 {
		t.Fatalf("got %d and %d draws in the two hours, want 3 and 3", len(first.imgs), len(second.imgs))
	}
	for i := range first.imgs {
		if first.imgs[i] != second.imgs[i] {
			t.Errorf("draw %d submitted a different mask texture the second hour; the cache was not hit", i)
		}
	}
	// And the two hours are actually observable at the draw: the alpha the
	// second hour's object/structure draws carry differs from the first's.
	if second.alpha[0] == first.alpha[0] {
		t.Fatalf("the second hour's alpha equals the first's; the fixture does not discriminate")
	}
}
