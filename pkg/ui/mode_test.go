package ui

import (
	"testing"

	"againrom/pkg/render/terrain"
)

// altGrid builds a synthetic W*H tile grid carrying an altitude layer. The
// altitude bytes are passed exactly as the decode delivers them — 0x80 is -128,
// not 128 — and are NOT padded, so a short slice stays short and is the
// wrong-length case.
func altGrid(w, h int, alts ...uint8) terrain.Grid {
	g := grid(w, h)
	g.Altitudes = append([]uint8(nil), alts...)
	return g
}

// The fixtures. Each canvas height below is MaxV - MinV worked out by hand from
// V(c,r) = r*32 - h(c,r) over the (W+1) x (H+1) mesh, with the far-edge ring
// reading the clamped index.
const (
	// cliffW x cliffH, a 127 raise over a 2x2 interior block.
	//   r=0: V = 0            r=1: V = 32, -95, -95, -95
	//   r=2: V = 64, -63x3    r=3: V = 96    r=4: V = 128
	// MinV = -95, MaxV = 128.
	cliffW, cliffH = 3, 4
	cliffCanvasH   = 223
	cliffFlatH     = cliffH * terrain.CellSize // 128

	// pushedW x pushedH, the whole first row pushed to -128. Row 0's TOP corners
	// land BELOW its bottom ones, so every quad in that row is inverted.
	//   r=0: V = 128    r=1: V = 32    r=2: V = 64
	// MinV = 32, MaxV = 128, against a flat 64.
	pushedW, pushedH = 2, 2
	pushedCanvasH    = 96

	shortW, shortH = 1, 2
	shortCanvasH   = 44
	shortFlatH     = shortH * terrain.CellSize // 64
)

var (
	cliffAlts = []uint8{
		0, 0, 0,
		0, 127, 127,
		0, 127, 127,
		0, 0, 0,
	}
	pushedAlts = []uint8{
		0x80, 0x80,
		0, 0,
	}
	shortAlts = []uint8{0, 20}
)

func cliffGrid() terrain.Grid  { return altGrid(cliffW, cliffH, cliffAlts...) }
func pushedGrid() terrain.Grid { return altGrid(pushedW, pushedH, pushedAlts...) }
func shortGrid() terrain.Grid  { return altGrid(shortW, shortH, shortAlts...) }

func newViewer(t *testing.T, g terrain.Grid) *Viewer {
	t.Helper()
	v, err := NewViewer("m", g, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.hudHidden[hudPanelPack], v.hudHidden[hudPanelBook] = true, true
	return v
}

func TestModeFlatInputs(t *testing.T) {
	for _, tc := range []struct {
		name  string
		g     terrain.Grid
		flatH float64
	}{
		{"no altitude layer at all", grid(4, 6), 6 * terrain.CellSize},
		{"an empty but non-nil layer", altGrid(4, 6), 6 * terrain.CellSize},
		{"one entry short", altGrid(2, 2, 1, 2, 3), 2 * terrain.CellSize},
		{"one entry long", altGrid(2, 2, 1, 2, 3, 4, 5), 2 * terrain.CellSize},
		{"a whole extra row", altGrid(2, 2, 1, 2, 3, 4, 5, 6), 2 * terrain.CellSize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newViewer(t, tc.g)
			if got := v.Mode(); got != ModeFlat {
				t.Errorf("Mode() = %v, want flat", got)
			}
			if v.proj != nil {
				t.Errorf("a projection was built over an unusable altitude layer")
			}
			cam := v.Camera()
			if got, want := cam.WorldH(), tc.flatH; got != want {
				t.Errorf("WorldH() = %v, want the flat %v", got, want)
			}
			if got, want := cam.WorldW(), float64(tc.g.Width)*terrain.CellSize; got != want {
				t.Errorf("WorldW() = %v, want %v", got, want)
			}
		})
	}
}

func TestModeDisplacedIsChosenByValidityNotByRelief(t *testing.T) {
	for _, tc := range []struct {
		name    string
		g       terrain.Grid
		canvasH float64
	}{
		{"an all-zero valid layer", altGrid(5, 3, make([]uint8, 15)...), 3 * terrain.CellSize},
		{"a raised cliff", cliffGrid(), cliffCanvasH},
		{"a pushed row", pushedGrid(), pushedCanvasH},
		{"a world shorter than the flat one", shortGrid(), shortCanvasH},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newViewer(t, tc.g)
			if got := v.Mode(); got != ModeDisplaced {
				t.Fatalf("Mode() = %v, want displaced", got)
			}
			cam := v.Camera()
			if got := cam.WorldH(); got != tc.canvasH {
				t.Errorf("WorldH() = %v, want the projected %v", got, tc.canvasH)
			}
			if got, want := cam.WorldW(), float64(tc.g.Width)*terrain.CellSize; got != want {
				t.Errorf("WorldW() = %v, want %v — no altitude term reaches a column", got, want)
			}
			// The tile grid is untouched by any of it: it is what the visible
			// range is clipped to and what the draw loop indexes cells with.
			if cam.Cols != tc.g.Width || cam.Rows != tc.g.Height {
				t.Errorf("tile grid = %dx%d, want %dx%d", cam.Cols, cam.Rows, tc.g.Width, tc.g.Height)
			}
		})
	}
}

// SC-3, AC-9: the extent is synced by CONSTRUCTION, not by an overlay toggle.
// The game front-end reaches NewViewer and never touches either setter, so a
// viewer built with no setter call at all must already hold the displaced world.
// Dropping syncWorld from NewViewer fails here and nowhere else.
func TestNewViewerSyncsTheWorldWithNoOverlayCall(t *testing.T) {
	v := newViewer(t, cliffGrid())

	if got := v.Camera().WorldH(); got != cliffCanvasH {
		t.Fatalf("WorldH() = %v, want %v with no setter ever called", got, cliffCanvasH)
	}
	if cliffCanvasH == cliffFlatH {
		t.Fatal("the fixture does not discriminate: its canvas equals the flat height")
	}

	// The consequence, in world pixels: the bottom of the map is reachable. A
	// camera left at the flat height clips everything past it.
	if cliffCanvasH-cliffFlatH != 95 {
		t.Fatalf("fixture drift: displaced world exceeds the flat one by %d, want 95",
			cliffCanvasH-cliffFlatH)
	}
}

func TestModeFollowsOverlaysEnabledAfterConstruction(t *testing.T) {
	type toggle struct {
		name string
		set  func(v *Viewer, on bool)
	}
	objects := toggle{"objects", func(v *Viewer, on bool) { v.SetObjects(on, nil) }}
	units := toggle{"units", func(v *Viewer, on bool) { v.SetUnits(on, nil) }}

	t.Run("valid altitude grid: every combination, toggled in either order, stays displaced", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			order []toggle
		}{
			{"neither enabled", nil},
			{"objects alone", []toggle{objects}},
			{"units alone", []toggle{units}},
			{"both, objects then units", []toggle{objects, units}},
			{"both, units then objects", []toggle{units, objects}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				v := newViewer(t, cliffGrid())
				cam := v.Camera()
				check := func(step string) {
					t.Helper()
					if got := v.Mode(); got != ModeDisplaced {
						t.Fatalf("%s: Mode() = %v, want displaced (an overlay must no longer force flat)", step, got)
					}
					if got := cam.WorldH(); got != cliffCanvasH {
						t.Fatalf("%s: WorldH() = %v, want the unchanged %v", step, got, cliffCanvasH)
					}
					if !v.Lit() {
						t.Fatalf("%s: Lit() = false, want true — Lit() ignores the overlay flags too (0014 FR-4)", step)
					}
				}
				check("setup")

				for _, tg := range tc.order {
					tg.set(v, true)
					check(tg.name + " on")
				}
				// Off again: still displaced throughout, not only once every
				// overlay has cleared — there is no intermediate flat state
				// left to observe (contrast the pre-0015 version of this test,
				// which asserted flat until the LAST overlay went off).
				for _, tg := range tc.order {
					tg.set(v, false)
					check(tg.name + " off")
				}
			})
		}
	})

	// AC-4: an invalid altitude grid still reports flat regardless of the
	// overlays — AnchorHeight is total, but there is no relief for it to
	// place a marker on when the grid itself was never usable.
	t.Run("invalid altitude grid: flat regardless of the overlays", func(t *testing.T) {
		v := newViewer(t, grid(cliffW, cliffH)) // same size, no altitude layer at all
		cam := v.Camera()

		for _, tc := range []struct {
			name               string
			objectsOn, unitsOn bool
		}{
			{"neither", false, false},
			{"objects", true, false},
			{"units", false, true},
			{"both", true, true},
		} {
			v.SetObjects(tc.objectsOn, nil)
			v.SetUnits(tc.unitsOn, nil)
			if got := v.Mode(); got != ModeFlat {
				t.Fatalf("%s: Mode() = %v, want flat over an invalid altitude grid", tc.name, got)
			}
			if got := cam.WorldH(); got != float64(cliffFlatH) {
				t.Fatalf("%s: WorldH() = %v, want the flat %v", tc.name, got, float64(cliffFlatH))
			}
			if v.Lit() {
				t.Fatalf("%s: Lit() = true, want false — no altitude grid means no levels", tc.name)
			}
		}
	})
}

func TestSetFlatSelectsFlatWhileStayingLit(t *testing.T) {
	v := newViewer(t, cliffGrid())
	cam := v.Camera()
	if v.Mode() != ModeDisplaced || !v.Lit() || cam.WorldH() != cliffCanvasH {
		t.Fatalf("setup: Mode()=%v Lit()=%v WorldH()=%v, want displaced, lit, %v",
			v.Mode(), v.Lit(), cam.WorldH(), cliffCanvasH)
	}

	v.SetFlat(true)
	if got := v.Mode(); got != ModeFlat {
		t.Fatalf("SetFlat(true): Mode() = %v, want flat", got)
	}
	if !v.Lit() {
		t.Fatal("SetFlat(true): Lit() = false, want true — flat must not suppress lighting")
	}
	if got := cam.WorldH(); got != float64(cliffFlatH) {
		t.Fatalf("SetFlat(true): WorldH() = %v, want the flat %v", got, float64(cliffFlatH))
	}

	v.SetFlat(false)
	if got := v.Mode(); got != ModeDisplaced {
		t.Fatalf("SetFlat(false): Mode() = %v, want displaced", got)
	}
	if !v.Lit() {
		t.Fatal("SetFlat(false): Lit() = false, want true")
	}
	if got := cam.WorldH(); got != cliffCanvasH {
		t.Fatalf("SetFlat(false): WorldH() = %v, want the displaced %v", got, cliffCanvasH)
	}
}

func TestShortDisplacedWorldCentresRatherThanClamps(t *testing.T) {
	v := newViewer(t, shortGrid())
	cam := v.Camera()

	if got := cam.WorldH(); got != shortCanvasH {
		t.Fatalf("WorldH() = %v, want the projected %v (flat would be %v)",
			got, float64(shortCanvasH), float64(shortFlatH))
	}
	if want := (float64(shortCanvasH) - float64(cam.ViewH)) / 2; cam.Y != want {
		t.Errorf("Y = %v, want the centred %v", cam.Y, want)
	}
	// Panning cannot move a centred axis, so a viewer that had kept the flat
	// height would sit at a different Y for good.
	cam.Pan(0, 10000)
	if want := (float64(shortCanvasH) - float64(cam.ViewH)) / 2; cam.Y != want {
		t.Errorf("after a pan Y = %v, want it still centred at %v", cam.Y, want)
	}
}

func TestConstructionLeavesTheAltitudeSliceUntouched(t *testing.T) {
	alts := append([]uint8(nil), cliffAlts...)
	before := append([]uint8(nil), alts...)

	g := grid(cliffW, cliffH)
	g.Altitudes = alts

	v := newViewer(t, g)
	v.SetObjects(true, nil)
	v.SetUnits(true, nil)
	v.SetObjects(false, nil)
	v.SetUnits(false, nil)

	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v after the round trip, want displaced", v.Mode())
	}
	for i := range before {
		if alts[i] != before[i] {
			t.Fatalf("altitude %d changed from %d to %d", i, before[i], alts[i])
		}
	}
}

func TestFlatTileScreenIsTheUndisplacedLattice(t *testing.T) {
	v := newViewer(t, grid(cliffW, cliffH)) // same size, no altitude layer at all
	if v.Mode() != ModeFlat {
		t.Fatalf("Mode() = %v, want flat", v.Mode())
	}
	cam := v.Camera()

	for row := 0; row < cliffH; row++ {
		for col := 0; col < cliffW; col++ {
			gotX, gotY := v.flatTileScreen(col, row)
			wantX, wantY := cam.WorldToScreen(float64(col*terrain.CellSize), float64(row*terrain.CellSize))
			if gotX != wantX || gotY != wantY {
				t.Errorf("tile (%d,%d) = (%v,%v), want (%v,%v)", col, row, gotX, gotY, wantX, wantY)
			}
		}
	}
}

// hideBottomPanels switches the pack and book off so a test that measures the
// bare map surface is not reserved a bottom strip.
func hideBottomPanels(v *Viewer) {
	v.hudHidden[hudPanelPack], v.hudHidden[hudPanelBook] = true, true
}
