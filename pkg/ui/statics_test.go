package ui

// The windowed viewer's half of the static-object layer's construction
// (AC-3, AC-7, SC-8's viewer half).
//
// Every fixture here is hand-built: a three-class bundle of synthetic frames
// over mode_test.go's 3x4 cliff grid, whose MinV is -95 and whose four-corner
// means are worked out in the table below. No game data is read, no install is
// touched and no window opens — the whole of this file runs under -check
// conditions, which is the point of it.

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

// staticsFrame builds one synthetic drawable frame: every pixel opaque at the
// given palette index, so the frame is a legal blit source even though nothing
// in this file blits. Only its SIZE reaches an anchor, and the size is the point
// — a frame that does not fill its class's canvas is the ordinary case.
func staticsFrame(w, h int, index uint8) *terrain.StaticFrame {
	f := &terrain.StaticFrame{Width: w, Height: h, Pixels: make([]terrain.StaticPixel, w*h)}
	for i := range f.Pixels {
		f.Pixels[i] = terrain.StaticPixel{Index: index, Opaque: true}
	}
	return f
}

// The bundle: two drawable classes, one the loader resolved but could not draw,
// and — by omission — a byte naming no class at all. The canvases and centres
// are deliberately odd and unequal to the frames, so a halving that took the
// canvas alone, or a frame taken for its canvas, moves the tables below.
//
//	byte 1: canvas 64x64, centre (32,60), frame 10x6
//	        anchorX = 32 - 64/2 + 10/2 =  5    anchorY = 60 - 64/2 + 6/2 = 31
//	byte 2: canvas 31x31, centre (15,29), frame  4x4
//	        anchorX = 15 - 31/2 +  4/2 =  2    anchorY = 29 - 31/2 + 4/2 = 16
//	byte 3: resolved, Frame nil — the NoFrame skip
//	byte 9: absent          — the NoClass skip
func staticsBundle() *terrain.StaticSet {
	set := new(terrain.StaticSet)
	set.Classes[1] = &terrain.StaticClass{Width: 64, Height: 64, CenterX: 32, CenterY: 60, Frame: staticsFrame(10, 6, 7)}
	set.Classes[2] = &terrain.StaticClass{Width: 31, Height: 31, CenterX: 15, CenterY: 29, Frame: staticsFrame(4, 4, 9)}
	set.Classes[3] = &terrain.StaticClass{Width: 8, Height: 8, CenterX: 4, CenterY: 7}
	return set
}

// staticsOverlay is the 3x4 object grid laid over the cliff fixture: three cells
// that place, one that names an artless class, one that names no class, and
// seven that are *no object*.
func staticsOverlay() []uint8 {
	return []uint8{
		1, 0, 2,
		0, 1, 0,
		0, 0, 0,
		3, 0, 9,
	}
}

// staticsGrid is cliffGrid with that overlay: the same altitudes mode_test.go
// pins, so MinV is -95 and the four-corner means the displaced table uses are
//
//	AnchorHeight(0,0) = (0+0+0+127)/4     = 31
//	AnchorHeight(2,0) = (0+0+127+127)/4   = 63
//	AnchorHeight(1,1) = (127+127+127+127)/4 = 127
//
// each read off the fixture's own altitude table rather than by calling the
// projection.
func staticsGrid() terrain.Grid {
	g := cliffGrid()
	g.Overlay = staticsOverlay()
	return g
}

// staticsWant is one expected placement, transcribed from the spec's "Sprite
// anchor" formula by hand.
type staticsWant struct {
	cell, topLeft, anchor, frame image.Point
}

// The flat geometry: lift and originY both zero, so
// destX = col*32 + 16 - anchorX and destY = row*32 + 16 - anchorY.
var staticsFlatWant = []staticsWant{
	{image.Pt(0, 0), image.Pt(11, -15), image.Pt(5, 31), image.Pt(10, 6)},
	{image.Pt(2, 0), image.Pt(78, 0), image.Pt(2, 16), image.Pt(4, 4)},
	{image.Pt(1, 1), image.Pt(43, 17), image.Pt(5, 31), image.Pt(10, 6)},
}

// The displaced geometry: lift is the cell's own four-corner mean and originY is
// the projection's MinV of -95, both SUBTRACTED, so
// destY = row*32 + 16 - anchorY - lift + 95 and destX is untouched.
//
//	(0,0):  0 + 16 - 31 -  31 + 95 =  49
//	(2,0):  0 + 16 - 16 -  63 + 95 =  32
//	(1,1): 32 + 16 - 31 - 127 + 95 = -15
var staticsDisplacedWant = []staticsWant{
	{image.Pt(0, 0), image.Pt(11, 49), image.Pt(5, 31), image.Pt(10, 6)},
	{image.Pt(2, 0), image.Pt(78, 32), image.Pt(2, 16), image.Pt(4, 4)},
	{image.Pt(1, 1), image.Pt(43, -15), image.Pt(5, 31), image.Pt(10, 6)},
}

// staticsCounts is the census both builds owe: three placements, one artless
// class, one unnamed byte. Byte 0 is counted nowhere.
var staticsCounts = terrain.StaticCounts{Placed: 3, NoClass: 1, NoFrame: 1}

// newStaticsViewer builds a viewer over staticsGrid with the given bundle and
// switches. It fails the test rather than returning an error, and — like every
// other constructor call in this package — opens no window.
func newStaticsViewer(t *testing.T, set *terrain.StaticSet, art, markers bool) *Viewer {
	t.Helper()
	v, err := NewViewerWithStatics("m", staticsGrid(), &terrain.Tileset{}, set, art, markers, terrain.AnimGateTiles, nil, false)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	return v
}

// checkStaticsList compares one built list against its hand-computed table,
// entry by entry and in order.
func checkStaticsList(t *testing.T, what string, got []terrain.StaticPlacement, want []staticsWant) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s list holds %d placements, want %d", what, len(got), len(want))
	}
	for i, w := range want {
		g := got[i]
		if g.Cell != w.cell {
			t.Errorf("%s[%d]: cell %v, want %v (row-major order)", what, i, g.Cell, w.cell)
		}
		if g.TopLeft != w.topLeft {
			t.Errorf("%s[%d] cell %v: top-left %v, want %v", what, i, w.cell, g.TopLeft, w.topLeft)
		}
		if g.Anchor != w.anchor {
			t.Errorf("%s[%d] cell %v: anchor %v, want %v", what, i, w.cell, g.Anchor, w.anchor)
		}
		if g.Frame == nil {
			t.Fatalf("%s[%d] cell %v: nil frame in a built list", what, i, w.cell)
		}
		if got, want := image.Pt(g.Frame.Width, g.Frame.Height), w.frame; got != want {
			t.Errorf("%s[%d] cell %v: frame %v, want %v", what, i, w.cell, got, want)
		}
	}
}

func TestNewViewerBuildsNoStaticLayer(t *testing.T) {
	v := newViewer(t, staticsGrid())

	if v.staticsFlat != nil || v.staticsDisplaced != nil {
		t.Errorf("NewViewer built %d flat and %d displaced placements, want none at all",
			len(v.staticsFlat), len(v.staticsDisplaced))
	}
	if places, counts := v.Statics(); places != 0 || counts != (terrain.StaticCounts{}) {
		t.Errorf("Statics() = (%d, %+v), want (0, a zero census)", places, counts)
	}
	if v.showStaticArt || v.showStaticMarkers {
		t.Errorf("NewViewer left art=%v markers=%v, want both off", v.showStaticArt, v.showStaticMarkers)
	}
}

func TestNewViewerDelegatesToNewViewerWithStatics(t *testing.T) {
	set := &terrain.Tileset{}

	plain, err := NewViewer("m", staticsGrid(), set)
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	delegated, err := NewViewerWithStatics("m", staticsGrid(), set, nil, false, false, terrain.AnimGateTiles, nil, false)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	if !reflect.DeepEqual(plain, delegated) {
		t.Errorf("NewViewer and NewViewerWithStatics(nil, false, false) built different viewers")
	}
}

// SC-8's second clause (AC-3, AC-7): with a bundle the viewer builds BOTH
// geometries at construction, each to the spec's own formula, and reports
// the census once.
//
// The tables are hand-computed above from the class fields, the frame sizes, the
// fixture's altitude means and MinV — never by calling StaticPlacements a second
// time — so this fails if the viewer feeds the builder the wrong lift, the wrong
// origin or the wrong sign, and not merely if it stops calling it.
func TestNewViewerWithStaticsBuildsBothGeometries(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)

	checkStaticsList(t, "flat", v.staticsFlat, staticsFlatWant)
	checkStaticsList(t, "displaced", v.staticsDisplaced, staticsDisplacedWant)

	if v.staticCounts != staticsCounts {
		t.Errorf("census %+v, want %+v", v.staticCounts, staticsCounts)
	}
	places, counts := v.Statics()
	if places != len(staticsDisplacedWant) || counts != staticsCounts {
		t.Errorf("Statics() = (%d, %+v), want (%d, %+v)", places, counts, len(staticsDisplacedWant), staticsCounts)
	}

	// The two lists differ by exactly -(lift + originY) in Y and by zero in X:
	// the displacement is vertical only, and it is the SAME three cells in the
	// same order, not a second selection of them.
	for i := range v.staticsFlat {
		flat, disp := v.staticsFlat[i], v.staticsDisplaced[i]
		if flat.Cell != disp.Cell {
			t.Fatalf("entry %d: flat cell %v, displaced cell %v", i, flat.Cell, disp.Cell)
		}
		if flat.TopLeft.X != disp.TopLeft.X {
			t.Errorf("cell %v: X moved with the geometry, %d -> %d", flat.Cell, flat.TopLeft.X, disp.TopLeft.X)
		}
		if flat.Frame != disp.Frame {
			t.Errorf("cell %v: the two geometries carry different frame pointers", flat.Cell)
		}
	}

	// Two cells hold the same placement byte, so they carry ONE frame pointer
	// between them — the identity the window's texture cache is keyed on.
	if v.staticsFlat[0].Frame != v.staticsFlat[2].Frame {
		t.Errorf("two cells of one class carry different frame pointers; the frame is copied, not shared")
	}
}

// SC-8, the "no texture" clause: construction with a bundle reaches no GPU.
// Nothing in the layer's build path allocates an ebiten image, so a -check
// run — which has no graphics context at all — gets this far and stops.
//
// The two caches are the viewer's only GPU state, and both are lazily filled by
// the draw path; a texture built here would be built before any window exists.
func TestNewViewerWithStaticsBuildsNoTexture(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)

	if len(v.cache) != 0 {
		t.Errorf("construction uploaded %d terrain images, want 0", len(v.cache))
	}
	if v.placeholder != nil {
		t.Errorf("construction built the placeholder image, want it lazy")
	}

	// Reading the layer must not build one either: -check calls exactly this.
	if _, _ = v.Statics(); len(v.cache) != 0 || v.placeholder != nil {
		t.Errorf("Statics() built a GPU image")
	}
}

// SC-8's third clause: the art switch gates the DRAW and never the BUILD.
func TestStaticSwitchesDoNotGateTheBuild(t *testing.T) {
	reference := newStaticsViewer(t, staticsBundle(), true, true)

	for _, c := range []struct{ art, markers bool }{
		{true, true},
		{true, false},
		{false, true},
		{false, false},
	} {
		v := newStaticsViewer(t, staticsBundle(), c.art, c.markers)

		if v.showStaticArt != c.art || v.showStaticMarkers != c.markers {
			t.Errorf("art=%v markers=%v: stored art=%v markers=%v",
				c.art, c.markers, v.showStaticArt, v.showStaticMarkers)
		}
		if !reflect.DeepEqual(v.staticsFlat, reference.staticsFlat) {
			t.Errorf("art=%v markers=%v: the flat list differs from the both-on one", c.art, c.markers)
		}
		if !reflect.DeepEqual(v.staticsDisplaced, reference.staticsDisplaced) {
			t.Errorf("art=%v markers=%v: the displaced list differs from the both-on one", c.art, c.markers)
		}
		if v.staticCounts != staticsCounts {
			t.Errorf("art=%v markers=%v: census %+v, want %+v", c.art, c.markers, v.staticCounts, staticsCounts)
		}
	}
}

func TestSetFlatSelectsAListAndRebuildsNeither(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)

	flatAddr := reflect.ValueOf(v.staticsFlat).Pointer()
	dispAddr := reflect.ValueOf(v.staticsDisplaced).Pointer()
	if flatAddr == dispAddr {
		t.Fatalf("the two geometries share one backing array; they are not two lists")
	}

	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v over the cliff fixture, want displaced", v.Mode())
	}
	if got := reflect.ValueOf(v.staticPlacements()).Pointer(); got != dispAddr {
		t.Errorf("displaced mode places from the flat list")
	}

	v.SetFlat(true)
	if v.Mode() != ModeFlat {
		t.Fatalf("Mode() = %v after SetFlat(true), want flat", v.Mode())
	}
	if got := reflect.ValueOf(v.staticPlacements()).Pointer(); got != flatAddr {
		t.Errorf("flat mode does not place from the flat list")
	}
	if reflect.ValueOf(v.staticsFlat).Pointer() != flatAddr ||
		reflect.ValueOf(v.staticsDisplaced).Pointer() != dispAddr {
		t.Errorf("SetFlat rebuilt a list; both were built at construction")
	}
	checkStaticsList(t, "flat after SetFlat(true)", v.staticsFlat, staticsFlatWant)
	checkStaticsList(t, "displaced after SetFlat(true)", v.staticsDisplaced, staticsDisplacedWant)

	// And back: the selection follows Mode() live, in both directions.
	v.SetFlat(false)
	if got := reflect.ValueOf(v.staticPlacements()).Pointer(); got != dispAddr {
		t.Errorf("SetFlat(false) does not return to the displaced list")
	}
}

func TestViewerHasNoStaticBundleSetter(t *testing.T) {
	bundle := reflect.TypeOf((*terrain.StaticSet)(nil))
	vt := reflect.TypeOf((*Viewer)(nil))

	for i := 0; i < vt.NumMethod(); i++ {
		m := vt.Method(i)
		for j := 0; j < m.Type.NumIn(); j++ {
			if m.Type.In(j) == bundle {
				t.Errorf("(*Viewer).%s takes a *terrain.StaticSet; the bundle is a construction parameter", m.Name)
			}
		}
	}
}

func TestStaticListsFollowTheProjectionGuard(t *testing.T) {
	g := grid(3, 4)
	g.Overlay = staticsOverlay() // altitudes deliberately absent

	v, err := NewViewerWithStatics("m", g, &terrain.Tileset{}, staticsBundle(), true, true, terrain.AnimGateTiles, nil, false)
	if err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	if v.proj != nil {
		t.Fatalf("a grid with no altitude layer was projected")
	}
	if v.staticsDisplaced != nil {
		t.Errorf("%d displaced placements built with no projection", len(v.staticsDisplaced))
	}
	checkStaticsList(t, "flat", v.staticsFlat, staticsFlatWant)

	if places, counts := v.Statics(); places != len(staticsFlatWant) || counts != staticsCounts {
		t.Errorf("Statics() = (%d, %+v), want (%d, %+v)", places, counts, len(staticsFlatWant), staticsCounts)
	}
	if got := reflect.ValueOf(v.staticPlacements()).Pointer(); got != reflect.ValueOf(v.staticsFlat).Pointer() {
		t.Errorf("a flat-mode viewer does not place from the flat list")
	}
}

// The builder validates the object layer's own length, and a viewer over a
// grid whose overlay does not match it is not an error — the tile grid's
// length is what is fatal here. So a bundle plus a malformed overlay yields
// a viewer with no placements rather than a failed construction.
func TestStaticListsIgnoreAMismatchedOverlay(t *testing.T) {
	for _, c := range []struct {
		name    string
		overlay []uint8
	}{
		{"one entry short", staticsOverlay()[:11]},
		{"one entry long", append(staticsOverlay(), 1)},
		{"absent", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := staticsGrid()
			g.Overlay = c.overlay

			v, err := NewViewerWithStatics("m", g, &terrain.Tileset{}, staticsBundle(), true, true, terrain.AnimGateTiles, nil, false)
			if err != nil {
				t.Fatalf("NewViewerWithStatics: %v", err)
			}
			if len(v.staticsFlat) != 0 || len(v.staticsDisplaced) != 0 {
				t.Errorf("built %d flat and %d displaced placements over a %d-byte overlay, want none",
					len(v.staticsFlat), len(v.staticsDisplaced), len(c.overlay))
			}
			if places, counts := v.Statics(); places != 0 || counts != (terrain.StaticCounts{}) {
				t.Errorf("Statics() = (%d, %+v), want (0, a zero census)", places, counts)
			}
		})
	}
}

// The validation order is the one NewViewer always had, and the bundle does
// not move it: a malformed grid or a nil tileset is rejected before anything
// is built, and no viewer comes back beside the error.
func TestNewViewerWithStaticsValidatesBeforeBuilding(t *testing.T) {
	for _, c := range []struct {
		name string
		g    terrain.Grid
		set  *terrain.Tileset
		want string
	}{
		{"short tile slice", terrain.Grid{Width: 3, Height: 4, Tiles: make([]uint16, 5), Overlay: staticsOverlay()}, &terrain.Tileset{}, "want 12"},
		{"non-positive size", terrain.Grid{Width: 0, Height: 4, Overlay: staticsOverlay()}, &terrain.Tileset{}, "non-positive size"},
		{"nil tileset", staticsGrid(), nil, "nil tileset"},
	} {
		t.Run(c.name, func(t *testing.T) {
			v, err := NewViewerWithStatics("m", c.g, c.set, staticsBundle(), true, true, terrain.AnimGateTiles, nil, false)
			if err == nil {
				t.Fatalf("NewViewerWithStatics succeeded, want an error mentioning %q", c.want)
			}
			if v != nil {
				t.Errorf("a viewer came back beside the error")
			}
		})
	}
}

func TestStaticBuildLeavesTheBundleAndOverlayUntouched(t *testing.T) {
	set := staticsBundle()
	before := staticsBundle()
	g := staticsGrid()
	overlayBefore := append([]uint8(nil), g.Overlay...)

	if _, err := NewViewerWithStatics("m", g, &terrain.Tileset{}, set, true, true, terrain.AnimGateTiles, nil, false); err != nil {
		t.Fatalf("NewViewerWithStatics: %v", err)
	}
	if !reflect.DeepEqual(set, before) {
		t.Errorf("construction mutated the object bundle")
	}
	if !reflect.DeepEqual(g.Overlay, overlayBefore) {
		t.Errorf("construction mutated the map's object layer")
	}
}
