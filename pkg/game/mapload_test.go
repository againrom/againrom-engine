package game_test

// Tests for the one place map bytes become a running viewer. Fixtures are
// synthetic map streams built in test code; no game install is read and no
// window is opened.

import (
	"image"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// markedMap is a synthetic map carrying two placed structures and three placed
// units, with anchors that are deliberately NOT on cell boundaries: 0x0280 is
// 2.5 cells and 0x01c0 is 1.75. A derivation that added a centre offset, a
// border inset or a rounding rule would answer a different cell for these,
// where it would agree with the plain shift on a boundary-aligned anchor.
func markedMap() []byte {
	return synth.ALM(synth.ALMOptions{
		Width: 6, Height: 5,
		Objects: []synth.ALMObject{{X: 0x0280, Y: 0x01c0}, {X: 0x0000, Y: 0x0400}},
		Units:   []synth.ALMUnit{{X: 0x0500, Y: 0x0080}, {X: 0x0180, Y: 0x0300}, {X: 0x02ff, Y: 0x0000}},
	})
}

// TestMarkerCells pins the one conversion from a placement record to the
// cell its marker is drawn in: the record's own stored anchor, shifted, and
// nothing else. Nothing about a class, a sprite or a frame may reach it —
// that independence is what lets a misplaced sprite disagree with its marker
// visibly instead of moving it along.
func TestMarkerCells(t *testing.T) {
	m, err := alm.Open(markedMap())
	if err != nil {
		t.Fatalf("alm.Open: %v", err)
	}

	objects, units := game.MarkerCells(m)

	wantObjects := []image.Point{{X: 2, Y: 1}, {X: 0, Y: 4}}
	wantUnits := []image.Point{{X: 5, Y: 0}, {X: 1, Y: 3}, {X: 2, Y: 0}}
	if len(objects) != len(wantObjects) {
		t.Fatalf("objects = %v, want %v", objects, wantObjects)
	}
	if len(units) != len(wantUnits) {
		t.Fatalf("units = %v, want %v", units, wantUnits)
	}
	for i, want := range wantObjects {
		if objects[i] != want {
			t.Errorf("object cell %d = %v, want %v", i, objects[i], want)
		}
	}
	for i, want := range wantUnits {
		if units[i] != want {
			t.Errorf("unit cell %d = %v, want %v", i, units[i], want)
		}
	}
}

// TestLoadMapViewerMarkers is the wiring the game was missing: the shared load
// path, and not either caller, is what turns the overlays on (AC-16).
func TestLoadMapViewerMarkers(t *testing.T) {
	tiles := &terrain.Tileset{}

	load := func(t *testing.T, markers game.Markers) *ui.Viewer {
		t.Helper()
		mv, err := game.LoadMapViewer(tiles, markedMap(), "x", markers, game.StaticLayer{}, game.StructureLayer{})
		if err != nil {
			t.Fatalf("LoadMapViewer: %v", err)
		}
		return mv.Viewer
	}

	check := func(t *testing.T, label string, gotOn bool, gotCells int, wantOn bool, wantCells int) {
		t.Helper()
		if gotOn != wantOn || gotCells != wantCells {
			t.Errorf("%s overlay = (on %v, %d cells), want (on %v, %d cells)",
				label, gotOn, gotCells, wantOn, wantCells)
		}
	}

	t.Run("all three requested, both record overlays reach the viewer with a cell per record", func(t *testing.T) {
		v := load(t, game.Markers{Objects: true, Units: true, Statics: true})
		on, cells := v.ObjectOverlay()
		check(t, "object", on, cells, true, 2)
		on, cells = v.UnitOverlay()
		check(t, "unit", on, cells, true, 3)
	})

	t.Run("the three are independent", func(t *testing.T) {
		// The developer viewer offers each without the others, so the shared path
		// must not couple them. The third field is exercised here for what it must
		// NOT do: it takes no cells from this package and must leave the two
		// record-derived overlays alone.
		v := load(t, game.Markers{Objects: true})
		on, cells := v.ObjectOverlay()
		check(t, "object", on, cells, true, 2)
		on, cells = v.UnitOverlay()
		check(t, "unit", on, cells, false, 0)

		v = load(t, game.Markers{Units: true})
		on, cells = v.ObjectOverlay()
		check(t, "object", on, cells, false, 0)
		on, cells = v.UnitOverlay()
		check(t, "unit", on, cells, true, 3)

		v = load(t, game.Markers{Statics: true})
		on, cells = v.ObjectOverlay()
		check(t, "object", on, cells, false, 0)
		on, cells = v.UnitOverlay()
		check(t, "unit", on, cells, false, 0)
	})

	t.Run("neither requested leaves the viewer as it was", func(t *testing.T) {
		v := load(t, game.Markers{})
		on, cells := v.ObjectOverlay()
		check(t, "object", on, cells, false, 0)
		on, cells = v.UnitOverlay()
		check(t, "unit", on, cells, false, 0)
	})
}

// staticsMap is a synthetic map whose type-3 grid exercises all three answers
// the placement builder can give, on a 4x3 lattice: two cells that resolve to a
// drawable frame, one byte naming no loaded class, and one class whose sheet the
// fixture archive does not hold.
//
// The two skip kinds are both present and are counted apart, which is what makes
// the census below discriminate: a load that dropped the map's object grid
// answers zeroes for all three, and one that passed a nil bundle answers zeroes
// too — but a load that passed the bundle over a map with nothing placed answers
// zero PLACEMENTS with a non-zero skip, and that is a different fact.
func staticsMap() []byte {
	return synth.ALM(synth.ALMOptions{
		Width: 4, Height: 3, Name: "Statics",
		Overlay: []uint8{
			0, staticGood, 0, staticNoClass,
			0, 0, 0, 0,
			staticFirst, 0, staticNoSheet, 0,
		},
	})
}

// TestLoadMapViewerStatics is the load path's half of the static-object
// layer: the map's own object grid and the caller's bundle both reaching the
// one viewer this function builds.
//
// The two switches are read back through ui.StaticOverlay and asserted APART.
// They are the one place this hand-off can go wrong silently: art and cross
// build the same lists, report the same Statics() count, and differ only in what
// a window paints, so passing them the wrong way round is invisible to every
// other assertion in this file and to every count either front-end prints. What
// they then draw is pinned inside pkg/ui, where the draw path is reachable.
func TestLoadMapViewerStatics(t *testing.T) {
	tiles := &terrain.Tileset{}

	load := func(t *testing.T, data []byte, layer game.StaticLayer) *ui.Viewer {
		t.Helper()
		mv, err := game.LoadMapViewer(tiles, data, "x", game.Markers{}, layer, game.StructureLayer{})
		if err != nil {
			t.Fatalf("LoadMapViewer: %v", err)
		}
		return mv.Viewer
	}

	t.Run("a zero-value layer gives back the pre-story viewer", func(t *testing.T) {
		// The map places objects and the viewer still holds none: without a bundle
		// there is nothing to resolve them against, so the layer is absent rather
		// than empty and the census is silent about a grid it never read.
		v := load(t, staticsMap(), game.StaticLayer{})
		places, counts := v.Statics()
		if places != 0 {
			t.Errorf("%d placements with no bundle, want none", places)
		}
		if want := (terrain.StaticCounts{}); counts != want {
			t.Errorf("counts = %+v with no bundle, want %+v", counts, want)
		}
	})

	t.Run("the bundle and the map's own object grid both reach the viewer", func(t *testing.T) {
		// The grid arrives on terrain.Grid.Overlay beside the tiles and the
		// altitudes. Dropping that field leaves this census all zeroes while every
		// other assertion in this file still passes.
		v := load(t, staticsMap(), game.StaticLayer{Set: loadFixtureStatics(t), Art: true})
		places, counts := v.Statics()
		want := terrain.StaticCounts{Placed: 2, NoClass: 1, NoFrame: 1}
		if counts != want {
			t.Errorf("counts = %+v, want %+v", counts, want)
		}
		if places != want.Placed {
			t.Errorf("%d placements, want %d", places, want.Placed)
		}
	})

	t.Run("both geometries are built, so the flat diagnostic selects rather than rebuilds", func(t *testing.T) {
		// The default synthetic map's altitude layer is valid, so this viewer
		// starts displaced; SetFlat moves it to the other list, which must have
		// been built at construction too.
		v := load(t, staticsMap(), game.StaticLayer{Set: loadFixtureStatics(t), Art: true})
		if got := v.Mode(); got != ui.ModeDisplaced {
			t.Fatalf("Mode() = %v, want displaced", got)
		}
		displaced, _ := v.Statics()
		v.SetFlat(true)
		flat, _ := v.Statics()
		if displaced != 2 || flat != 2 {
			t.Errorf("placements = %d displaced, %d flat; want 2 in each geometry", displaced, flat)
		}
	})

	t.Run("the art switch gates the draw and never the build", func(t *testing.T) {
		// A caller that wants the layer's cross alone still supplies the bundle
		// and leaves the art off; the lists it marks from are the same lists.
		set := loadFixtureStatics(t)
		onPlaces, onCounts := load(t, staticsMap(), game.StaticLayer{Set: set, Art: true}).Statics()
		offPlaces, offCounts := load(t, staticsMap(), game.StaticLayer{Set: set, Art: false}).Statics()
		if onPlaces != offPlaces || onCounts != offCounts {
			t.Errorf("art off = (%d, %+v), art on = (%d, %+v); the switch moved the build",
				offPlaces, offCounts, onPlaces, onCounts)
		}
	})

	t.Run("a map that places nothing still reports what its bytes named", func(t *testing.T) {
		// Zero placements from a bundle that WAS supplied, told apart from zero
		// placements because none was: the skip is counted, which a nil bundle
		// could not produce.
		data := synth.ALM(synth.ALMOptions{
			Width: 2, Height: 1,
			Overlay: []uint8{0, staticNoClass},
		})
		places, counts := load(t, data, game.StaticLayer{Set: loadFixtureStatics(t), Art: true}).Statics()
		if places != 0 {
			t.Errorf("%d placements, want none — neither cell resolves to a drawable frame", places)
		}
		if want := (terrain.StaticCounts{NoClass: 1}); counts != want {
			t.Errorf("counts = %+v, want %+v", counts, want)
		}
	})

	t.Run("the art switch and the cross switch arrive apart and the right way round", func(t *testing.T) {
		// Markers.Statics is the cross and StaticLayer.Art is the sprites, and
		// the two travel side by side into one constructor call. Swapping them
		// changes neither placement list, neither census and neither front-end's
		// count — it changes only what a window paints — so all four
		// combinations are asserted here rather than left to a manual pass.
		set := loadFixtureStatics(t)
		for _, tc := range []struct{ art, cross bool }{
			{false, false}, {true, false}, {false, true}, {true, true},
		} {
			mv, err := game.LoadMapViewer(tiles, staticsMap(), "x",
				game.Markers{Statics: tc.cross}, game.StaticLayer{Set: set, Art: tc.art}, game.StructureLayer{})
			if err != nil {
				t.Fatalf("LoadMapViewer: %v", err)
			}
			art, cross := mv.Viewer.StaticOverlay()
			if art != tc.art || cross != tc.cross {
				t.Errorf("art=%v cross=%v reached the viewer as art=%v cross=%v",
					tc.art, tc.cross, art, cross)
			}
		}
	})

	t.Run("the cross is reachable without the art, and the art without a map that places", func(t *testing.T) {
		// Either switch on its own must load and construct cleanly: the marker
		// field takes no cells from this package, so it cannot depend on the
		// bundle being present, and a bundle over an empty grid is not an error.
		for _, tc := range []struct {
			label   string
			markers game.Markers
			layer   game.StaticLayer
		}{
			{"the cross with no bundle at all", game.Markers{Statics: true}, game.StaticLayer{}},
			{"the cross with a bundle and no art", game.Markers{Statics: true},
				game.StaticLayer{Set: loadFixtureStatics(t)}},
			{"the art with no cross", game.Markers{},
				game.StaticLayer{Set: loadFixtureStatics(t), Art: true}},
		} {
			t.Run(tc.label, func(t *testing.T) {
				mv, err := game.LoadMapViewer(tiles, staticsMap(), "x", tc.markers, tc.layer, game.StructureLayer{})
				if err != nil {
					t.Fatalf("LoadMapViewer: %v", err)
				}
				want := 0
				if tc.layer.Set != nil {
					want = 2
				}
				if places, _ := mv.Viewer.Statics(); places != want {
					t.Errorf("%d placements, want %d", places, want)
				}
			})
		}
	})
}

func TestLoadMapViewer(t *testing.T) {
	tiles := &terrain.Tileset{}

	t.Run("a recorded name titles the viewer", func(t *testing.T) {
		data := synth.ALM(synth.ALMOptions{Width: 6, Height: 4, Name: "Crossroads"})

		mv, err := game.LoadMapViewer(tiles, data, "fallback.alm", game.Markers{}, game.StaticLayer{}, game.StructureLayer{})
		if err != nil {
			t.Fatalf("LoadMapViewer: %v", err)
		}
		if mv.Title != "Crossroads" {
			t.Errorf("Title = %q, want the map's own recorded name", mv.Title)
		}
		if mv.Viewer == nil {
			t.Errorf("Viewer is nil")
		}
		if mv.Map == nil {
			t.Fatalf("Map is nil")
		}
		if mv.Map.Width != 6 || mv.Map.Height != 4 {
			t.Errorf("Map size = %dx%d, want 6x4", mv.Map.Width, mv.Map.Height)
		}
	})

	t.Run("an empty recorded name falls back to the supplied title", func(t *testing.T) {
		// Most campaign maps record no name, so this is the ordinary case rather
		// than the exceptional one.
		data := synth.ALM(synth.ALMOptions{Width: 3, Height: 3, Name: ""})

		mv, err := game.LoadMapViewer(tiles, data, "10.alm", game.Markers{}, game.StaticLayer{}, game.StructureLayer{})
		if err != nil {
			t.Fatalf("LoadMapViewer: %v", err)
		}
		if mv.Title != "10.alm" {
			t.Errorf("Title = %q, want the fallback %q", mv.Title, "10.alm")
		}
	})

	t.Run("the camera is sized to the map", func(t *testing.T) {
		data := synth.ALM(synth.ALMOptions{Width: 12, Height: 7, Name: "Sized"})

		mv, err := game.LoadMapViewer(tiles, data, "x", game.Markers{}, game.StaticLayer{}, game.StructureLayer{})
		if err != nil {
			t.Fatalf("LoadMapViewer: %v", err)
		}
		cam := mv.Viewer.Camera()
		if cam.Cols != 12 || cam.Rows != 7 {
			t.Errorf("camera world = %dx%d tiles, want 12x7", cam.Cols, cam.Rows)
		}
	})

	t.Run("the map's altitudes reach the viewer and displace its world", func(t *testing.T) {
		// This is the path the whole of 0013 exists for: the front-end's picker
		// calls loadMap, which calls LoadMapViewer, which calls ui.NewViewer —
		// and touches neither overlay setter on the way. So the displaced extent
		// has to arrive by construction or not at all (0013 AC-9).
		//
		// A 4x3 map with its middle row raised 127. Over the 5x4 mesh,
		// V(c,r) = r*32 - h: r=0 gives 0, r=1 gives -95, r=2 gives 64, and the
		// far-edge r=3 gives 96. MinV = -95, MaxV = 96, so the world is 191 tall
		// against a flat 3*32 = 96 — the two cannot be confused.
		const w, h = 4, 3
		alts := make([]uint8, w*h)
		for c := 0; c < w; c++ {
			alts[1*w+c] = 127
		}
		data := synth.ALM(synth.ALMOptions{Width: w, Height: h, Name: "Relief", Altitudes: alts})

		mv, err := game.LoadMapViewer(tiles, data, "x", game.Markers{}, game.StaticLayer{}, game.StructureLayer{})
		if err != nil {
			t.Fatalf("LoadMapViewer: %v", err)
		}
		if got := mv.Viewer.Mode(); got != ui.ModeDisplaced {
			t.Errorf("Mode() = %v, want displaced — the altitudes did not reach the viewer", got)
		}
		cam := mv.Viewer.Camera()
		if got := cam.WorldH(); got != 191 {
			t.Errorf("WorldH() = %v, want 191 (the flat world would be %v)", got, float64(h*terrain.CellSize))
		}
		if got := cam.WorldW(); got != w*terrain.CellSize {
			t.Errorf("WorldW() = %v, want %v — no altitude reaches a column", got, float64(w*terrain.CellSize))
		}
	})

	t.Run("a map with no relief still displaces, at the flat height", func(t *testing.T) {
		// The default synthetic map's altitude layer is all zeros: valid, so the
		// mode is displaced, and MaxV-MinV is numerically Height*32. Mode never
		// reads an altitude VALUE.
		data := synth.ALM(synth.ALMOptions{Width: 6, Height: 4})

		mv, err := game.LoadMapViewer(tiles, data, "x", game.Markers{}, game.StaticLayer{}, game.StructureLayer{})
		if err != nil {
			t.Fatalf("LoadMapViewer: %v", err)
		}
		if got := mv.Viewer.Mode(); got != ui.ModeDisplaced {
			t.Errorf("Mode() = %v, want displaced over an all-zero altitude grid", got)
		}
		if got := mv.Viewer.Camera().WorldH(); got != 4*terrain.CellSize {
			t.Errorf("WorldH() = %v, want %v", got, float64(4*terrain.CellSize))
		}
	})

	t.Run("a malformed map errors before any viewer exists", func(t *testing.T) {
		for _, tc := range []struct {
			label string
			data  []byte
		}{
			{"empty", nil},
			{"garbage", []byte("not a map at all, not even close")},
			{"truncated", synth.ALM(synth.ALMOptions{Width: 4, Height: 4})[:40]},
			// Well framed, metadata reads, grid inconsistent: this is exactly the
			// map that lists cleanly in the picker and only fails when chosen.
			{"broken grid", synth.ALM(synth.ALMOptions{
				Width: 5, Height: 5, Name: "Listed But Broken",
				Type1Payload: make([]byte, 2*24),
			})},
		} {
			t.Run(tc.label, func(t *testing.T) {
				mv, err := game.LoadMapViewer(tiles, tc.data, "x", game.Markers{}, game.StaticLayer{}, game.StructureLayer{})
				if err == nil {
					t.Fatalf("LoadMapViewer accepted a malformed map")
				}
				if mv != nil {
					t.Errorf("LoadMapViewer returned a non-nil *MapView alongside an error")
				}
			})
		}
	})

	t.Run("errors come back unwrapped so each caller labels them", func(t *testing.T) {
		// This function takes bytes and has no path to name. Wrapping here would
		// force one caller's phrasing on the other; instead the standalone viewer
		// keeps printing `decode <path>: <err>` and the front-end says something
		// suited to a picker row.
		_, err := game.LoadMapViewer(tiles, []byte("garbage"), "some/path.alm", game.Markers{}, game.StaticLayer{}, game.StructureLayer{})
		if err == nil {
			t.Fatalf("expected an error")
		}
		if strings.Contains(err.Error(), "some/path.alm") {
			t.Errorf("error %q names the caller's path; it should be unwrapped", err)
		}
	})

	t.Run("a nil tileset is rejected rather than deferred to the draw path", func(t *testing.T) {
		data := synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "N"})
		if _, err := game.LoadMapViewer(nil, data, "x", game.Markers{}, game.StaticLayer{}, game.StructureLayer{}); err == nil {
			t.Errorf("LoadMapViewer accepted a nil tileset")
		}
	})
}
