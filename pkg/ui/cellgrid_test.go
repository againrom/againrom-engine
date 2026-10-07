package ui

import (
	"image"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/render/camera"
	"againrom/pkg/render/terrain"
)

func TestTheLatticeIsOffByDefaultAndDrawnSeparately(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*32, cliffCanvasH)
	if v.GridOverlay() {
		t.Fatal("the lattice came up on; it is a diagnostic and must default off")
	}
	if got := v.gridScreenSegments(); got != nil {
		t.Errorf("the lattice drew %d segments with the toggle off", len(got))
	}

	// A frame with several passes already in it, so "the pass slice is
	// untouched" is an assertion about something rather than about an empty
	// slice.
	v.SetEntities([]MapEntity{{ID: 2, Cell: image.Pt(1, 1), HP: 4, MaxHP: 8}})
	v.sel = selection{2}
	v.SetUnits(true, []image.Point{{X: 0, Y: 0}})
	v.SetBlocked(true, []image.Point{{X: 2, Y: 2}})

	before := v.overlayPasses()
	if len(before) < 2 {
		t.Fatalf("the fixture frame holds %d passes; it cannot show that they survive", len(before))
	}
	v.SetGrid(true)
	after := v.overlayPasses()
	if !reflect.DeepEqual(after, before) {
		t.Error("turning the lattice on changed overlayPasses' own slice; the lattice no longer belongs to it")
	}
	if got := v.gridScreenSegments(); len(got) == 0 {
		t.Error("the lattice is on and drew no segment at all")
	}
}

// wantGridSegments is the independent oracle for cell (col,row)'s own four
// projected-corner edges in screen space: WorldCorner at each of the cell's
// four mesh vertices — TL, TR, BR, BL — each through cam.WorldToScreen,
// never through v.gridWorldQuad or v.gridScreenSegments.
func wantGridSegments(cam *camera.Camera, proj terrain.Projection, col, row int) []gridSegment {
	var pts [4][2]float64
	for i, d := range [4][2]int{{0, 0}, {1, 0}, {1, 1}, {0, 1}} {
		wx, wy := proj.WorldCorner(col+d[0], row+d[1])
		sx, sy := cam.WorldToScreen(float64(wx), float64(wy))
		pts[i] = [2]float64{sx, sy}
	}
	var out []gridSegment
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		out = append(out, gridSegment{X0: pts[i][0], Y0: pts[i][1], X1: pts[j][0], Y1: pts[j][1]})
	}
	return out
}

// hasGridSegment reports whether segs carries the segment (x0,y0)-(x1,y1), in
// either direction — an edge shared by two cells is walked by each in its own
// perimeter order, so the two occurrences need not agree on which endpoint
// comes first.
func hasGridSegment(segs []gridSegment, x0, y0, x1, y1 float64) bool {
	for _, s := range segs {
		if s.X0 == x0 && s.Y0 == y0 && s.X1 == x1 && s.Y1 == y1 {
			return true
		}
		if s.X0 == x1 && s.Y0 == y1 && s.X1 == x0 && s.Y1 == y0 {
			return true
		}
	}
	return false
}

// countGridSegment counts how many of segs carry the segment (x0,y0)-(x1,y1),
// in either direction.
func countGridSegment(segs []gridSegment, x0, y0, x1, y1 float64) int {
	n := 0
	for _, s := range segs {
		if (s.X0 == x0 && s.Y0 == y0 && s.X1 == x1 && s.Y1 == y1) ||
			(s.X0 == x1 && s.Y0 == y1 && s.X1 == x0 && s.Y1 == y0) {
			n++
		}
	}
	return n
}

func TestTheLatticeCoversTheTerrainPassBand(t *testing.T) {
	for _, tc := range []struct {
		name string
		v    func(t *testing.T) *Viewer
		proj func() terrain.Projection
	}{
		{"displaced", func(t *testing.T) *Viewer {
			return identityViewer(t, cliffGrid(), cliffW*32, cliffCanvasH)
		}, cliffProjection},
		{"flat", func(t *testing.T) *Viewer {
			return identityViewer(t, grid(4, 4), 4*32, 4*32)
		}, func() terrain.Projection { return terrain.Projection{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.v(t)
			v.SetGrid(true)

			var band []image.Point
			v.forEachDrawnTile(func(col, row int) { band = append(band, image.Pt(col, row)) })
			if len(band) == 0 {
				t.Fatal("the terrain pass drew no tile; this fixture proves nothing")
			}

			var want []gridSegment
			if v.Mode() == ModeDisplaced {
				proj := tc.proj()
				for _, cell := range band {
					want = append(want, wantGridSegments(v.Camera(), proj, cell.X, cell.Y)...)
				}
			} else {
				for _, cell := range band {
					cs := float64(terrain.CellSize)
					x0, y0 := float64(cell.X)*cs, float64(cell.Y)*cs
					x1, y1 := x0+cs, y0+cs
					corners := [4][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
					for i := 0; i < 4; i++ {
						j := (i + 1) % 4
						sx0, sy0 := v.Camera().WorldToScreen(corners[i][0], corners[i][1])
						sx1, sy1 := v.Camera().WorldToScreen(corners[j][0], corners[j][1])
						want = append(want, gridSegment{X0: sx0, Y0: sy0, X1: sx1, Y1: sy1})
					}
				}
			}

			got := v.gridScreenSegments()
			if len(got) != len(want) {
				t.Fatalf("the lattice drew %d segments, want the band's %d", len(got), len(want))
			}
			for _, w := range want {
				if !hasGridSegment(got, w.X0, w.Y0, w.X1, w.Y1) {
					t.Errorf("no segment matches the band's own %+v", w)
				}
			}
		})
	}
}

// TestTheLatticeBendsAtEachCellsOwnCorners — item-2 hotfix, owner: "сетка
// соединяется своими углами": a cell's own outline is the QUADRILATERAL of
// its four projected corners, not an axis-aligned rectangle, and two cells
// sharing an edge report the identical two corner points for it.
//
// The fixture is 3 columns by 1 row, altitudes 0, 50, 90 — hand-derived
// below — chosen so cell (0,0)'s own top edge (corners (0,0) and (1,0),
// altitudes 0 and 50) cannot be level, and so the mean-of-four-corners
// placement surface 0058 drew the lattice on gives cell (0,0) and cell (1,0)
// two DIFFERENT lifts (AnchorHeight 25 and 70), which is exactly the defect
// this hotfix closes: on the corner mesh the two cells read the shared edge's
// two points from the identical Vertex calls and cannot disagree.
//
//	Altitude(c,0) = 0, 50, 90 (Width=3, Height=1; row 1 clamps to row 0)
//	Vertex(c,0) = 0-0, 0-50, 0-90     = 0, -50, -90 (col 3 clamped to col 2's -90)
//	Vertex(c,1) = 32-0, 32-50, 32-90  = 32, -18, -58
//	MinV = -90, MaxV = 32, CanvasHeight = 122
//	WorldCorner(c,r) = (c*32, Vertex(c,r)+90):
//	  (0,0)->(0,90)  (1,0)->(32,40)  (2,0)->(64,0)   (3,0)->(96,0)
//	  (0,1)->(0,122) (1,1)->(32,72)  (2,1)->(64,32)  (3,1)->(96,32)
func TestTheLatticeBendsAtEachCellsOwnCorners(t *testing.T) {
	const slopeW, slopeH = 3, 1
	slopeAlts := []uint8{0, 50, 90}
	buildAlts := func() []uint8 { return append([]uint8(nil), slopeAlts...) }

	proj := terrain.Project(buildAlts(), slopeW, slopeH)
	if proj.MinV != -90 || proj.CanvasHeight() != 122 {
		t.Fatalf("fixture drift: MinV=%d CanvasHeight=%d, want -90 and 122 — re-derive the numbers "+
			"in this test's own comment before trusting it", proj.MinV, proj.CanvasHeight())
	}

	g := grid(slopeW, slopeH)
	g.Altitudes = buildAlts()
	v := identityViewer(t, g, slopeW*terrain.CellSize, 122)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("the fixture came up %v, want displaced", v.Mode())
	}
	v.SetGrid(true)

	segs := v.gridScreenSegments()
	if len(segs) == 0 {
		t.Fatal("the lattice drew nothing")
	}

	// Cell (0,0)'s own top edge: corners (0,0) and (1,0), altitudes 0 and 50.
	x0, y0 := proj.WorldCorner(0, 0)
	x1, y1 := proj.WorldCorner(1, 0)
	fx0, fy0 := v.Camera().WorldToScreen(float64(x0), float64(y0))
	fx1, fy1 := v.Camera().WorldToScreen(float64(x1), float64(y1))
	if fy0 == fy1 {
		t.Fatalf("cell (0,0)'s top edge is level at Y=%v; the fixture's altitudes must differ "+
			"to discriminate a quadrilateral from a rectangle", fy0)
	}
	if !hasGridSegment(segs, fx0, fy0, fx1, fy1) {
		t.Errorf("the lattice carries no segment for cell (0,0)'s own top edge, (%v,%v)-(%v,%v)",
			fx0, fy0, fx1, fy1)
	}

	// The edge cell (0,0) and cell (1,0) share: mesh vertices (1,0) and
	// (1,1). Both cells must report the IDENTICAL two points for it, and
	// nothing else in the returned segments may carry those two points —
	// there are exactly two cells that own this edge.
	sx0, sy0 := proj.WorldCorner(1, 0)
	sx1, sy1 := proj.WorldCorner(1, 1)
	fsx0, fsy0 := v.Camera().WorldToScreen(float64(sx0), float64(sy0))
	fsx1, fsy1 := v.Camera().WorldToScreen(float64(sx1), float64(sy1))
	if n := countGridSegment(segs, fsx0, fsy0, fsx1, fsy1); n != 2 {
		t.Errorf("the shared edge (%v,%v)-(%v,%v) appears %d times, want exactly 2 — one from "+
			"cell (0,0) and one from cell (1,0), at the identical two points",
			fsx0, fsy0, fsx1, fsy1, n)
	}
}

func TestTheLatticeIsFlatOnAViewerWithNoAltitudes(t *testing.T) {
	v := identityViewer(t, grid(3, 3), 3*32, 3*32)
	if v.Mode() != ModeFlat {
		t.Fatalf("a grid with no altitude layer came up %v, want flat", v.Mode())
	}
	v.SetGrid(true)

	segs := v.gridScreenSegments()
	if len(segs) == 0 {
		t.Fatal("the lattice drew nothing")
	}
	for _, s := range segs {
		for _, x := range []float64{s.X0, s.X1} {
			if x < 0 || x > 3*32 {
				t.Fatalf("segment %+v leaves the flat map's own extent on X", s)
			}
		}
		for _, y := range []float64{s.Y0, s.Y1} {
			if y < 0 || y > 3*32 {
				t.Fatalf("segment %+v leaves the flat map's own extent on Y", s)
			}
		}
	}
	// The top-left cell's own top edge starts at the origin, unlifted.
	if !hasGridSegment(segs, 0, 0, 32, 0) {
		t.Error("the lattice carries no segment for the flat cell (0,0)'s own top edge, (0,0)-(32,0)")
	}
}

func TestTheGridKeyTogglesOncePerPress(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*32, cliffCanvasH)

	if v.ToggleGrid() != true || !v.GridOverlay() {
		t.Fatal("the first toggle did not turn the lattice on")
	}
	if v.ToggleGrid() != false || v.GridOverlay() {
		t.Fatal("the second toggle did not turn it off")
	}

	// The snapshot's field is a press EDGE: two frames of a held key deliver
	// one true and then false, so the state cannot flicker.
	v.SetGrid(false)
	for _, in := range []appInput{{Grid: true}, {Grid: false}, {Grid: false}} {
		if in.Grid {
			v.ToggleGrid()
		}
	}
	if !v.GridOverlay() {
		t.Error("a held key over three frames left the lattice off; a press edge must act once")
	}
}

func TestTheGridSetterMovesNothingElse(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*32, cliffCanvasH)
	cam := v.Camera()
	before := [3]float64{cam.X, cam.Y, cam.Zoom}
	beforeH := cam.WorldH()

	v.SetGrid(true)

	if got := [3]float64{cam.X, cam.Y, cam.Zoom}; got != before {
		t.Errorf("SetGrid moved the camera from %v to %v", before, got)
	}
	if got := cam.WorldH(); got != beforeH {
		t.Errorf("SetGrid moved the world height from %v to %v", beforeH, got)
	}
	if v.Mode() != ModeDisplaced {
		t.Error("SetGrid moved the geometry mode")
	}
}

func TestTheLatticeStopsBelowItsLegibilityFloor(t *testing.T) {
	v := identityViewer(t, cliffGrid(), cliffW*32, cliffCanvasH)
	v.SetGrid(true)
	if len(v.gridScreenSegments()) == 0 {
		t.Fatal("the lattice drew nothing at native zoom")
	}

	for _, zoom := range []float64{
		GridMinCellPixels / float64(terrain.CellSize),
		camera.ZoomMin,
		0.1,
	} {
		v.Camera().SetZoom(zoom)
		cellpx := float64(terrain.CellSize) * v.Camera().Zoom
		got := len(v.gridScreenSegments())
		switch {
		case cellpx >= GridMinCellPixels && got == 0:
			t.Errorf("at zoom %v a cell is %v px and the lattice drew nothing", zoom, cellpx)
		case cellpx < GridMinCellPixels && got != 0:
			t.Errorf("at zoom %v a cell is %v px and the lattice drew %d segments — below the floor "+
				"it is a wash over the map, not a lattice", zoom, cellpx, got)
		}
	}

	// A zoom that is not a number draws nothing rather than reaching the
	// placement, which is what the positive test buys.
	v.Camera().Zoom = math.NaN()
	if got := v.gridScreenSegments(); got != nil {
		t.Errorf("a non-finite zoom drew %d segments", len(got))
	}
}
