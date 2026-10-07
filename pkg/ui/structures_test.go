package ui

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

// structureLiftGrid is an 8x8 map with one altitude of 40 and every other 0,
// carrying one type-4 record: a 3x3 class anchored at (2,3).
//
// THE 40 SITS AT (3,4), which is the centre cell of that rectangle. That single
// value is what separates the two candidate height accessors:
//
//	Altitude, the mesh CORNER accessor the contract names:
//	  the four samples are the vertices (3,4) (4,4) (3,5) (4,5)
//	  = (40 + 0 + 0 + 0) / 4 = 10
//
//	AnchorHeight, the per-CELL four-corner mean the object layer takes:
//	  the four samples are AnchorHeight(3,4)=10, (4,4)=0, (3,5)=0, (4,5)=0
//	  = (10 + 0 + 0 + 0) / 4 = 2
//
// Both are worked out here from the altitude table by hand, so the assertion
// compares the viewer against arithmetic rather than against a second call.
//
// Every altitude is non-negative and row 0 is flat, so the projection's MinV is
// 0 and the origin term drops out of the expectation without being ignored by
// it — TestStructureOriginIsTheProjectionsOwn pins the origin separately.
const (
	structLiftW, structLiftH     = 8, 8
	structLiftCol, structLiftRow = 2, 3
	structLiftPeak               = 40
	structLiftCorner             = structLiftPeak / 4 // 10, the contract's answer
	structLiftPerCell            = 2                  // what AnchorHeight would give
)

func structureLiftGrid() terrain.Grid {
	alt := make([]uint8, structLiftW*structLiftH)
	alt[4*structLiftW+3] = structLiftPeak
	return terrain.Grid{
		Width:     structLiftW,
		Height:    structLiftH,
		Tiles:     make([]uint16, structLiftW*structLiftH),
		Altitudes: alt,
		Structures: []terrain.StructureRecord{
			{X: structLiftCol << 8, Y: structLiftRow << 8, Key: 1},
		},
	}
}

// structureFrame is one drawable tile-sized frame. Only its size reaches the
// geometry, and a structure frame IS a cell.
func structureFrame() *terrain.StaticFrame {
	return &terrain.StaticFrame{
		Width:  terrain.CellSize,
		Height: terrain.CellSize,
		Pixels: make([]terrain.StaticPixel, terrain.CellSize*terrain.CellSize),
	}
}

// structureBundle is one 3x3 class of FullHeight 3 — no overhang, so each cell
// of the rectangle is one frame and a hand-computed destY needs no strip step.
func structureBundle() *terrain.StructureSet {
	set := new(terrain.StructureSet)
	frames := make([]*terrain.StaticFrame, 9)
	for i := range frames {
		frames[i] = structureFrame()
	}
	set.Classes[1] = &terrain.StructureClass{TileWidth: 3, TileHeight: 3, FullHeight: 3, Frames: frames}
	return set
}

func newStructureViewer(t *testing.T, g terrain.Grid, set *terrain.StructureSet, art bool) *Viewer {
	t.Helper()
	v, err := NewViewerWithStatics("m", g, &terrain.Tileset{}, nil, false, false, terrain.AnimGateTiles, set, art)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	return v
}

func TestStructureLiftTakesCornerHeightsAndNotCellHeights(t *testing.T) {
	v := newStructureViewer(t, structureLiftGrid(), structureBundle(), true)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced; the fixture's altitude layer is valid", v.Mode())
	}
	if v.proj.MinV != 0 {
		t.Fatalf("MinV = %d, want 0; the expectations below drop the origin term", v.proj.MinV)
	}
	if len(v.structuresDisplaced) == 0 {
		t.Fatal("the displaced list is empty")
	}

	// destY = row*CellSize - lift - originY, and this class has no overhang, so
	// the back row's entries sit at exactly row*CellSize - lift.
	for _, p := range v.structuresDisplaced {
		lift := p.Cell.Y*terrain.CellSize - p.TopLeft.Y
		if lift == structLiftPerCell {
			t.Fatalf("the lift is %d, the per-CELL four-corner mean; the contract's sample is over the "+
				"terrain's CORNER heights at the rectangle's centre and is %d (DD-6)",
				lift, structLiftCorner)
		}
		if lift != structLiftCorner {
			t.Fatalf("cell %v is lifted by %d, want %d", p.Cell, lift, structLiftCorner)
		}
	}
}

// Every entry of ONE structure carries ONE lift, as the window builds it:
// the whole building stands on one height, so the displaced list is the flat
// list translated vertically and by nothing else.
func TestStructureDisplacedListIsTheFlatListTranslated(t *testing.T) {
	v := newStructureViewer(t, structureLiftGrid(), structureBundle(), true)
	if len(v.structuresFlat) != len(v.structuresDisplaced) {
		t.Fatalf("%d flat entries against %d displaced", len(v.structuresFlat), len(v.structuresDisplaced))
	}
	for i := range v.structuresFlat {
		a, b := v.structuresFlat[i], v.structuresDisplaced[i]
		if a.Cell != b.Cell || a.GridIndex != b.GridIndex || a.TopLeft.X != b.TopLeft.X {
			t.Fatalf("entry %d differs beyond a vertical translation: %+v against %+v", i, a, b)
		}
		if d := a.TopLeft.Y - b.TopLeft.Y; d != structLiftCorner {
			t.Fatalf("entry %d is translated by %d, want %d", i, d, structLiftCorner)
		}
	}
}

// The vertical origin the structure list is built at is the projection's own
// MinV — the same value the object layer takes and the same one the raster
// reports as Render.OriginY, which is what makes one entry land on the same
// world point in the window as in the raster.
func TestStructureOriginIsTheProjectionsOwn(t *testing.T) {
	g := structureLiftGrid()
	// The altitude is SUBTRACTED to make a vertex row, so a high altitude in row
	// 0 pulls MinV below zero and the origin term becomes non-zero — which is
	// what makes an implementation that dropped it observable.
	g.Altitudes = append([]uint8(nil), g.Altitudes...)
	g.Altitudes[0] = 96

	v := newStructureViewer(t, g, structureBundle(), true)
	if v.proj.MinV == 0 {
		t.Fatal("MinV is 0; the fixture must make the origin observable")
	}
	for _, p := range v.structuresDisplaced {
		// The peak is untouched by the edit, so the lift is still the corner one.
		want := p.Cell.Y*terrain.CellSize - structLiftCorner - v.proj.MinV
		if p.TopLeft.Y != want {
			t.Fatalf("cell %v has destY %d, want %d (row*%d - lift %d - originY %d)",
				p.Cell, p.TopLeft.Y, want, terrain.CellSize, structLiftCorner, v.proj.MinV)
		}
	}
}

// AC-9's viewer half. A viewer given NO structure bundle holds no entry and
// reports a zero census in either geometry, whatever the map's records say — so
// a caller that passes none holds the viewer that shipped before this story.
func TestStructureViewerWithoutABundle(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  *terrain.StructureSet
	}{
		{"no bundle", nil},
		{"a bundle no key resolves in", new(terrain.StructureSet)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newStructureViewer(t, structureLiftGrid(), tc.set, true)
			if len(v.structuresFlat) != 0 || len(v.structuresDisplaced) != 0 {
				t.Errorf("%d flat and %d displaced entries, want none",
					len(v.structuresFlat), len(v.structuresDisplaced))
			}
			// The per-placement census still SPEAKS — a key naming no loaded
			// class is a skip it must be able to report — but nothing is drawn
			// and no cell is touched, which is what "the pre-story picture" is.
			entries, counts := v.Structures()
			if entries != 0 || counts.Placements.Drawn != 0 || counts.Cells != (terrain.StructureCellCounts{}) {
				t.Errorf("Structures() = %d, %+v, want nothing drawn and no cell touched", entries, counts)
			}
			v.SetFlat(true)
			if entries, _ := v.Structures(); entries != 0 {
				t.Errorf("flat mode reports %d entries, want none", entries)
			}
		})
	}
}

// --- the two passes, as the window draws them (AC-7) ---

// planeGrid is an 8x8 map with NO altitude layer — so the viewer is flat and a
// world coordinate is a cell times CellSize — carrying one object cell and two
// structure records: a non-flat class at (2,2) and a Flat one at (5,5).
//
// The flat one stands on a LATER row than everything else, so "the early pass
// draws first" and "the merge draws by row" cannot both be satisfied by one
// order: a merge alone would put it last.
const planeW, planeH = 8, 8

func planeGrid() terrain.Grid {
	overlay := make([]uint8, planeW*planeH)
	overlay[0] = 1 // one object at cell (0,0)
	return terrain.Grid{
		Width:   planeW,
		Height:  planeH,
		Tiles:   make([]uint16, planeW*planeH),
		Overlay: overlay,
		Structures: []terrain.StructureRecord{
			{X: 2 << 8, Y: 2 << 8, Key: 1},
			{X: 5 << 8, Y: 5 << 8, Key: 2},
		},
	}
}

// planeStatics is one object class whose canvas, centre and frame put its sprite
// exactly on its own cell: anchor = (16 - 16 + 16) = 16 in both axes, so
// destX = col*32 + 16 - 16.
func planeStatics() *terrain.StaticSet {
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{
		Width: terrain.CellSize, Height: terrain.CellSize,
		CenterX: terrain.CellSize / 2, CenterY: terrain.CellSize / 2,
		Frame: structureFrame(),
	}
	return set
}

// planeStructures is two 1x1 classes of FullHeight 1 — one frame per placement,
// standing on its own cell — one of them Flat.
func planeStructures() *terrain.StructureSet {
	set := new(terrain.StructureSet)
	set.Classes[1] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{structureFrame()},
	}
	set.Classes[2] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1, Flat: true,
		Frames: []*terrain.StaticFrame{structureFrame()},
	}
	return set
}

// planeViewer is that map through a window matched exactly to the world, so the
// camera is the identity and every recorded translate IS a world coordinate.
func planeViewer(t *testing.T, structures *terrain.StructureSet) *Viewer {
	t.Helper()
	v, err := NewViewerWithStatics("m", planeGrid(), &terrain.Tileset{},
		planeStatics(), true, false, terrain.AnimGateTiles, structures, true)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	layoutViewport(v, planeW*terrain.CellSize, planeH*terrain.CellSize)
	cam := v.Camera()
	if cam.X != 0 || cam.Y != 0 || cam.Zoom != 1 {
		t.Fatalf("camera is (%v,%v) at zoom %v, want the identity", cam.X, cam.Y, cam.Zoom)
	}
	if v.Mode() != ModeFlat {
		t.Fatalf("Mode() = %v, want flat; the fixture carries no altitude layer", v.Mode())
	}
	return v
}

// planeTranslates is the sequence of top-left translates one art pass submitted.
func planeTranslates(rec *staticRecorder) [][2]float64 {
	out := make([][2]float64, 0, len(rec.geo))
	for _, g := range rec.geo {
		out = append(out, [2]float64{g[2], g[5]})
	}
	return out
}

// planeStructureShadowShift is the X shift a structure shadow carries for
// EITHER of planeStructures()' two classes, at the viewer's seeded sun
// (terrain.DefaultDaytime) and this fixture's own numbers, now the SUM of
// two terms rather than one:
//
// Computed here through the same two functions shadow.go itself calls,
// rather than a hand-typed float64 literal: the sum is not an integer, and
// a literal carrying fewer digits than float64's own arithmetic would drift
// from the runtime value by a fraction of a pixel, which the exact-equality
// assertions below cannot absorb.
var planeStructureShadowShift = float64(terrain.StructureShadowShift(terrain.DefaultTheta, 1, 0, 0)) +
	terrain.ShadowSlope(terrain.DefaultTheta)*float64(terrain.CellSize)

// AC-7, as the window draws it. The FLAT structure precedes every static object
// and every other structure entry, whatever its row; the rest is the merge by
// rectangle row; and no entry is drawn twice.
func TestDrawArtPutsStructureShadowsBeforeFlatAndCellBodies(t *testing.T) {
	v := planeViewer(t, planeStructures())

	var rec staticRecorder
	v.drawArt(&rec)

	// ANIM-CELL-085: structure shadows, flat decoration, then main cell bodies.
	want := [][2]float64{
		{2*terrain.CellSize + planeStructureShadowShift, 2 * terrain.CellSize},
		{5*terrain.CellSize + planeStructureShadowShift, 5 * terrain.CellSize},
		{5 * terrain.CellSize, 5 * terrain.CellSize},
		{0, 0},
		{2 * terrain.CellSize, 2 * terrain.CellSize},
	}
	got := planeTranslates(&rec)
	if len(got) != len(want) {
		t.Fatalf("the art pass made %d draw calls, want %d\ngot %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("draw order = %v, want %v — the flat pass draws before the object layer, "+
				"and the rest merges by rectangle row", got, want)
		}
	}
}

func TestDrawArtWithoutStructuresIsTheObjectLayerAlone(t *testing.T) {
	var withNone, withEmpty staticRecorder
	planeViewer(t, nil).drawArt(&withNone)
	planeViewer(t, new(terrain.StructureSet)).drawArt(&withEmpty)

	want := [][2]float64{{0, 0}}
	for _, tc := range []struct {
		name string
		rec  *staticRecorder
	}{{"no bundle", &withNone}, {"a bundle no key resolves in", &withEmpty}} {
		got := planeTranslates(tc.rec)
		if len(got) != len(want) || got[0] != want[0] {
			t.Errorf("%s: draw order = %v, want %v — only the object layer draws", tc.name, got, want)
		}
	}
}

// Each art switch gates its own plane and nothing else, so a viewer asked for one
// draws that one in the merged order and the other not at all.
//
// SINCE 0101, showStructureArt also gates the two structure shadows (0101
// spec: shadowDraws honours the switch exactly as planeSprites does) — 2
// bodies (the early pass's flat one, the merge's non-flat one) plus 2 shadows
// — while showStaticArt gates nothing extra here, because planeStatics'
// object class carries no Frames slice for ObjectShadowPlace to anchor
// against and so contributes no shadow whichever switch is set.
func TestDrawArtGatesEachPlaneOnItsOwnSwitch(t *testing.T) {
	v := planeViewer(t, planeStructures())

	v.showStaticArt = false
	var structsOnly staticRecorder
	v.drawArt(&structsOnly)
	if got, want := len(structsOnly.geo), 4; got != want {
		t.Errorf("%d draw calls with the object switch off, want the %d structure entries and shadows", got, want)
	}

	v.showStaticArt, v.showStructureArt = true, false
	var objectsOnly staticRecorder
	v.drawArt(&objectsOnly)
	if got, want := len(objectsOnly.geo), 1; got != want {
		t.Errorf("%d draw calls with the structure switch off, want the %d object placement", got, want)
	}

	v.showStaticArt = false
	var neither staticRecorder
	v.drawArt(&neither)
	if len(neither.geo) != 0 {
		t.Errorf("%d draw calls with both switches off, want none", len(neither.geo))
	}
}

func TestViewerPlacesFromTheBuildersOwnList(t *testing.T) {
	g := structureLiftGrid()
	set := structureBundle()
	v := newStructureViewer(t, g, set, true)

	flat, flatCounts, flatAnim := terrain.StructurePlacements(g, set, nil, 0)
	if !reflect.DeepEqual(v.structuresFlat, flat) {
		t.Error("the viewer's flat list is not the builder's own")
	}
	if !reflect.DeepEqual(v.structureAnimFlat, flatAnim) {
		t.Error("the viewer's flat animated subset is not the builder's own")
	}
	if v.structureCounts != flatCounts {
		t.Errorf("the viewer's census is %+v, the builder's %+v", v.structureCounts, flatCounts)
	}

	displaced, _, displacedAnim := terrain.StructurePlacements(g, set, v.proj.Altitude, v.proj.MinV)
	if !reflect.DeepEqual(v.structuresDisplaced, displaced) {
		t.Error("the viewer's displaced list is not the builder's own at the projection's own two terms")
	}
	if !reflect.DeepEqual(v.structureAnimDisplaced, displacedAnim) {
		t.Error("the viewer's displaced animated subset is not the builder's own")
	}

	// And the merged order is the render tier's own, over those same two lists.
	if !reflect.DeepEqual(v.planeOrder, terrain.PlaneOrder(flat, v.staticsFlat)) {
		t.Error("the viewer's plane order is not the render tier's own merge")
	}
}

// 0068 AC-6, AC-7, as the window draws it. A unit takes its place in the ONE
// back-to-front order the structure and object planes already share: after every
// drawable on an earlier row, before every drawable on a later one, and after
// every FLAT structure whatever the rows.
//
// This is the defect the owner saw: units were painted after the whole art band,
// so one standing behind a building was drawn in front of it. The two entities
// are handed over in DESCENDING row so the ordering cannot come from the list
// they arrive in, and the flat structure stands on row 5 — later than both the
// object and the non-flat structure — so "the early pass is first" and "the merge
// is by row" cannot both hold unless the prefix is really held apart.
//
// The literals are the object fixture's own geometry and artA's: an entity at
// cell (c,r) drawn from a 64x64 canvas centred (32,60) with a 10x6 frame has
// anchor (5,31), so flat it stands at (32c+16-5, 32r+16-31).
//
// planeUnitShadowShift is the unit shadow's own X displacement from its
// body, now (same shape as planeStructureShadowShift above) the SUM of
// ShadowPivotShift's translation and the shear's own row-0 offset at the
// drawn frame's pivot — no longer the single translated term 0101 gave
// (UnitShadowShift(DefaultTheta) = 18, a whole-cell-independent constant).
// artA's own drawn frame is 10x6 with anchor (5,31)
// (entity_overlay_test.go), so frameH=6, anchorY=31, and both terms come
// from terrain's own functions rather than a hand-typed literal, for the
// same exact-equality reason planeStructureShadowShift does above.
func TestDrawArtPutsAUnitInTheRowOrderOfTheArtPlanes(t *testing.T) {
	planeUnitShadowShift := float64(-terrain.ShadowPivotShift(terrain.DefaultTheta, 6, 31)) +
		terrain.ShadowSlope(terrain.DefaultTheta)*6

	// Ordinary unit shadows and bodies run together in each main-sweep cell.
	// Keep AC6's actual cross-row body proof against the row2 structure.

	v := planeViewer(t, planeStructures())
	artA := entityArtA()
	v.SetEntities([]MapEntity{
		withFrame(image.Pt(1, 7), artA), // the later row, handed over FIRST
		withFrame(image.Pt(1, 1), artA),
	})

	var rec staticRecorder
	v.drawArt(&rec)

	want := [][2]float64{
		{2*terrain.CellSize + planeStructureShadowShift, 2 * terrain.CellSize}, // structure row-2 shadow
		{5*terrain.CellSize + planeStructureShadowShift, 5 * terrain.CellSize}, // structure row-5 (flat) shadow
		{5 * terrain.CellSize, 5 * terrain.CellSize},                           // flat body before every ordinary body
		{0, 0},                                 // the object on row 0
		{43 + planeUnitShadowShift, 1*32 - 15}, // row-1 ordinary shadow
		{43, 1*32 - 15},                        // the unit on row 1: BEHIND the structure on row 2
		{2 * terrain.CellSize, 2 * terrain.CellSize}, // the structure on row 2
		{43 + planeUnitShadowShift, 7*32 - 15},       // row-7 ordinary shadow
		{43, 7*32 - 15},                              // the unit on row 7: in front of everything
	}
	got := planeTranslates(&rec)
	if len(got) != len(want) {
		t.Fatalf("the art pass made %d draw calls, want %d\ngot %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("draw order = %v, want %v — a unit is drawn after everything on an earlier row, "+
				"before everything on a later one, and after every flat structure", got, want)
		}
	}
}

// 0068 AC-9's window half, and the other side of the switch fence: NEITHER art
// switch reaches a unit. A viewer with both planes turned off still draws its
// units, because a unit is not the map's art.
//
// SINCE 0101, that includes its shadow: shadowDraws never gates
// entityLayer's sprites on either switch (0101 spec: "Entities are never
// gated"), so the unit's shadow draws first, ahead of every content-plane
// draw.
func TestTheArtSwitchesDoNotGateAUnit(t *testing.T) {
	// See TestDrawArtPutsAUnitInTheRowOrderOfTheArtPlanes's own comment for
	// what the two summed terms are.
	planeUnitShadowShift := float64(-terrain.ShadowPivotShift(terrain.DefaultTheta, 6, 31)) +
		terrain.ShadowSlope(terrain.DefaultTheta)*6

	v := planeViewer(t, planeStructures())
	v.showStaticArt, v.showStructureArt = false, false
	v.SetEntities([]MapEntity{withFrame(image.Pt(1, 1), entityArtA())})

	var rec staticRecorder
	v.drawArt(&rec)

	want := [][2]float64{
		{43 + planeUnitShadowShift, 1*32 - 15}, // the unit's own shadow, ungated
		{43, 1*32 - 15},                        // the unit's own body
	}
	got := planeTranslates(&rec)
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("with both art switches off the band drew %v, want the unit's shadow then its body at %v", got, want)
	}
}
