package camera

import (
	"math"
	"math/rand"
	"testing"
)

// AC-1: panning past an edge clamps to [0, worldPx-viewPx] on each axis.
func TestPanClampsToWorldEdges(t *testing.T) {
	// 100x100 tiles = 3200x3200 world px, in an 800x600 window at zoom 1.
	c := New(100, 100, 800, 600)
	maxX, maxY := 3200.0-800, 3200.0-600

	tests := []struct {
		name   string
		dx, dy float64
		wantX  float64
		wantY  float64
	}{
		{"past the left/top edge", -5000, -5000, 0, 0},
		{"past the right/bottom edge", 99999, 99999, maxX, maxY},
		{"inside the world", 100, 50, 100, 50},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := *c
			c.Pan(tc.dx, tc.dy)
			if c.X != tc.wantX || c.Y != tc.wantY {
				t.Fatalf("Pan(%v,%v) = (%v,%v), want (%v,%v)", tc.dx, tc.dy, c.X, c.Y, tc.wantX, tc.wantY)
			}
		})
	}
}

// AC-2: an axis whose world is smaller than the view is centered.
func TestSmallerAxisIsCentered(t *testing.T) {
	// 10x10 tiles = 320x320 world px, viewed in an 800x600 window.
	c := New(10, 10, 800, 600)

	// Centering puts the offset at half the (negative) slack on both axes.
	wantX := (320.0 - 800) / 2
	wantY := (320.0 - 600) / 2
	if c.X != wantX || c.Y != wantY {
		t.Fatalf("centered = (%v,%v), want (%v,%v)", c.X, c.Y, wantX, wantY)
	}

	// Panning must not move a centered axis.
	c.Pan(500, -500)
	if c.X != wantX || c.Y != wantY {
		t.Fatalf("after pan = (%v,%v), want it to stay centered (%v,%v)", c.X, c.Y, wantX, wantY)
	}
}

// AC-3: zooming about a cursor keeps the world point under the cursor, and the
// scale stays inside the limits.
func TestZoomAboutCursorPreservesPoint(t *testing.T) {
	const eps = 1e-9

	// Start well away from every edge so the clamp cannot interfere.
	c := New(200, 200, 800, 600)
	c.X, c.Y = 2000, 2000

	for _, factor := range []float64{2, 0.5, 1.25, 0.8} {
		sx, sy := 300.0, 220.0
		beforeX, beforeY := c.ScreenToWorld(sx, sy)

		c.ZoomAbout(sx, sy, factor)

		afterX, afterY := c.ScreenToWorld(sx, sy)
		if math.Abs(afterX-beforeX) > eps || math.Abs(afterY-beforeY) > eps {
			t.Fatalf("factor %v: world point moved (%v,%v) -> (%v,%v)", factor, beforeX, beforeY, afterX, afterY)
		}
		if c.Zoom < ZoomMin || c.Zoom > ZoomMax {
			t.Fatalf("factor %v: zoom %v outside [%v,%v]", factor, c.Zoom, ZoomMin, ZoomMax)
		}
	}
}

// AC-3 (limits): zoom saturates at the bounds instead of running away.
func TestZoomClampsToLimits(t *testing.T) {
	c := New(200, 200, 800, 600)

	for i := 0; i < 64; i++ {
		c.ZoomAbout(400, 300, 2)
	}
	if c.Zoom != ZoomMax {
		t.Fatalf("zoom in: got %v, want ZoomMax %v", c.Zoom, ZoomMax)
	}

	for i := 0; i < 64; i++ {
		c.ZoomAbout(400, 300, 0.5)
	}
	if c.Zoom != ZoomMin {
		t.Fatalf("zoom out: got %v, want ZoomMin %v", c.Zoom, ZoomMin)
	}
}

// AC-4: the visible range covers exactly the tiles intersecting the view, clipped
// to the map.
func TestVisibleTiles(t *testing.T) {
	tests := []struct {
		name         string
		cols, rows   int
		viewW, viewH int
		x, y, zoom   float64
		want         TileRange
	}{
		{
			name: "aligned origin covers ceil(view/cell) tiles",
			cols: 100, rows: 100, viewW: 640, viewH: 320,
			x: 0, y: 0, zoom: 1,
			want: TileRange{Col0: 0, Row0: 0, Col1: 20, Row1: 10},
		},
		{
			name: "unaligned offset includes the partially covered tiles",
			cols: 100, rows: 100, viewW: 64, viewH: 64,
			x: 33, y: 31, zoom: 1,
			want: TileRange{Col0: 1, Row0: 0, Col1: 4, Row1: 3},
		},
		{
			name: "clipped at the far edge, never past the map",
			cols: 10, rows: 10, viewW: 640, viewH: 640,
			x: 200, y: 200, zoom: 1,
			want: TileRange{Col0: 0, Row0: 0, Col1: 10, Row1: 10},
		},
		{
			name: "zoomed in covers fewer tiles",
			cols: 100, rows: 100, viewW: 640, viewH: 640,
			x: 0, y: 0, zoom: 4,
			want: TileRange{Col0: 0, Row0: 0, Col1: 5, Row1: 5},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := New(tc.cols, tc.rows, tc.viewW, tc.viewH)
			c.Zoom = tc.zoom
			c.X, c.Y = tc.x, tc.y
			c.Clamp()

			got := c.VisibleTiles()
			if got != tc.want {
				t.Fatalf("VisibleTiles() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestRandomizedPanZoomKeepsInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(20051))

	for trial := 0; trial < 200; trial++ {
		cols := 1 + rng.Intn(300)
		rows := 1 + rng.Intn(300)
		viewW := 1 + rng.Intn(1600)
		viewH := 1 + rng.Intn(1200)

		c := New(cols, rows, viewW, viewH)

		for step := 0; step < 40; step++ {
			switch rng.Intn(3) {
			case 0:
				c.Pan((rng.Float64()-0.5)*20000, (rng.Float64()-0.5)*20000)
			case 1:
				c.ZoomAbout(rng.Float64()*float64(viewW), rng.Float64()*float64(viewH), 0.25+rng.Float64()*4)
			case 2:
				c.SetZoom(rng.Float64() * 16)
			}

			vw, vh := c.viewWorld()
			assertAxis(t, "X", c.X, c.WorldW(), vw)
			assertAxis(t, "Y", c.Y, c.WorldH(), vh)

			if c.Zoom < ZoomMin || c.Zoom > ZoomMax {
				t.Fatalf("zoom %v outside [%v,%v]", c.Zoom, ZoomMin, ZoomMax)
			}

			r := c.VisibleTiles()
			if r.Col0 < 0 || r.Row0 < 0 || r.Col1 > cols || r.Row1 > rows || r.Col0 > r.Col1 || r.Row0 > r.Row1 {
				t.Fatalf("VisibleTiles %+v escapes a %dx%d grid", r, cols, rows)
			}
		}
	}
}

func assertAxis(t *testing.T, name string, pos, world, view float64) {
	t.Helper()
	const eps = 1e-9

	if math.IsNaN(pos) || math.IsInf(pos, 0) {
		t.Fatalf("%s is not finite: %v", name, pos)
	}
	if world <= view {
		if want := (world - view) / 2; math.Abs(pos-want) > eps {
			t.Fatalf("%s = %v, want centered %v", name, pos, want)
		}
		return
	}
	if pos < -eps || pos > world-view+eps {
		t.Fatalf("%s = %v outside [0, %v]", name, pos, world-view)
	}
}

// A degenerate or hostile camera must not panic or produce a non-finite state.
func TestDegenerateInputsAreSafe(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Camera)
	}{
		{"zero-size view", func(c *Camera) { c.ViewW, c.ViewH = 0, 0; c.Clamp() }},
		{"NaN pan", func(c *Camera) { c.Pan(math.NaN(), math.NaN()) }},
		{"NaN position", func(c *Camera) { c.X, c.Y = math.NaN(), math.NaN(); c.Clamp() }},
		{"zero zoom", func(c *Camera) { c.Zoom = 0; c.Clamp() }},
		{"negative zoom", func(c *Camera) { c.Zoom = -4; c.Clamp() }},
		{"NaN zoom factor", func(c *Camera) { c.ZoomAbout(10, 10, math.NaN()) }},
		{"zero zoom factor", func(c *Camera) { c.ZoomAbout(10, 10, 0) }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := New(64, 64, 800, 600)
			tc.mutate(c)

			if math.IsNaN(c.X) || math.IsNaN(c.Y) || math.IsInf(c.X, 0) || math.IsInf(c.Y, 0) {
				t.Fatalf("position went non-finite: (%v,%v)", c.X, c.Y)
			}
			if c.Zoom < ZoomMin || c.Zoom > ZoomMax {
				t.Fatalf("zoom %v outside [%v,%v]", c.Zoom, ZoomMin, ZoomMax)
			}
			r := c.VisibleTiles()
			if r.Col0 < 0 || r.Row0 < 0 || r.Col1 > 64 || r.Row1 > 64 {
				t.Fatalf("VisibleTiles %+v escapes the grid", r)
			}
		})
	}
}

// A world smaller than one tile still behaves.
func TestSingleTileWorld(t *testing.T) {
	c := New(1, 1, 800, 600)
	r := c.VisibleTiles()
	if (r != TileRange{Col0: 0, Row0: 0, Col1: 1, Row1: 1}) {
		t.Fatalf("VisibleTiles = %+v, want the single tile", r)
	}
}

// ScreenToWorld and WorldToScreen are inverses.
func TestScreenWorldRoundTrip(t *testing.T) {
	const eps = 1e-9

	c := New(200, 200, 800, 600)
	c.SetZoom(2.5)
	c.Pan(1234, 567)

	for _, p := range [][2]float64{{0, 0}, {400, 300}, {799, 599}, {-20, 40}} {
		wx, wy := c.ScreenToWorld(p[0], p[1])
		sx, sy := c.WorldToScreen(wx, wy)
		if math.Abs(sx-p[0]) > eps || math.Abs(sy-p[1]) > eps {
			t.Fatalf("round trip (%v,%v) -> (%v,%v)", p[0], p[1], sx, sy)
		}
	}
}

// SC-1: New initialises the world height to the tile grid's own Rows*CellSize,
// so a flat construction is what it always was.
func TestNewWorldHeightIsTheTileGrid(t *testing.T) {
	for _, tc := range []struct{ cols, rows, viewW, viewH int }{
		{100, 100, 800, 600},
		{10, 10, 800, 600},
		{1, 1, 800, 600},
		{64, 48, 320, 320},
		{256, 256, 1, 1},
	} {
		c := New(tc.cols, tc.rows, tc.viewW, tc.viewH)
		if got, want := c.WorldH(), float64(tc.rows)*CellSize; got != want {
			t.Errorf("New(%d,%d,%d,%d).WorldH() = %v, want %v",
				tc.cols, tc.rows, tc.viewW, tc.viewH, got, want)
		}
		if got, want := c.WorldW(), float64(tc.cols)*CellSize; got != want {
			t.Errorf("New(%d,%d,%d,%d).WorldW() = %v, want %v",
				tc.cols, tc.rows, tc.viewW, tc.viewH, got, want)
		}
	}
}

// AC-3, AC-5: SetWorldHeight replaces the extent and re-clamps at once, over a
// world taller and shorter than the flat one, and the X axis and the tile grid
// are untouched by any of it.
func TestSetWorldHeight(t *testing.T) {
	// 10x20 tiles is a 320x640 flat world in an 800x600 window: X is already
	// centred (320 < 800) and Y already clamps (640 > 600), so both branches of
	// clampAxis are in play before anything moves.
	const flatH = 20 * CellSize // 640
	wantX := (320.0 - 800) / 2

	base := func(t *testing.T) *Camera {
		t.Helper()
		c := New(10, 20, 800, 600)
		c.Pan(0, 10000) // hard against the bottom: Y = 640 - 600 = 40
		if c.Y != flatH-600 {
			t.Fatalf("setup: Y = %v, want %v", c.Y, flatH-600)
		}
		return c
	}

	t.Run("a taller world extends the bound", func(t *testing.T) {
		c := base(t)
		c.SetWorldHeight(2000)
		if c.WorldH() != 2000 {
			t.Fatalf("WorldH() = %v, want 2000", c.WorldH())
		}
		// Y was in bounds and stays put: a taller world cannot pull it in.
		if c.Y != 40 {
			t.Errorf("Y = %v, want it left at 40", c.Y)
		}
		c.Pan(0, 10000)
		if want := 2000.0 - 600; c.Y != want {
			t.Errorf("after panning to the bottom Y = %v, want %v", c.Y, want)
		}
	})

	t.Run("a shorter world re-clamps at once, not at the next pan", func(t *testing.T) {
		// 620 is still taller than the 600 view, so the axis keeps clamping —
		// but its bound is now 20, and Y is sitting at 40.
		c := base(t)
		c.SetWorldHeight(620)
		if c.Y != 20 {
			t.Errorf("Y = %v, want 20 (pulled in by the new bound without a pan)", c.Y)
		}
	})

	t.Run("a world shorter than the view centres", func(t *testing.T) {
		// The path a real flat map never reaches: MaxV-MinV is not bounded below
		// by Rows*CellSize, so a displaced world can be shorter than the view.
		c := base(t)
		c.SetWorldHeight(400)
		want := (400.0 - 600) / 2
		if c.Y != want {
			t.Errorf("Y = %v, want centered %v", c.Y, want)
		}
		c.Pan(0, 5000)
		if c.Y != want {
			t.Errorf("after pan Y = %v, want it to stay centered %v", c.Y, want)
		}
	})

	t.Run("X and the tile grid are untouched", func(t *testing.T) {
		c := base(t)
		for _, h := range []float64{2000, 620, 400, 0} {
			c.SetWorldHeight(h)
			if c.X != wantX {
				t.Errorf("SetWorldHeight(%v): X = %v, want it still centered at %v", h, c.X, wantX)
			}
			if c.WorldW() != 320 {
				t.Errorf("SetWorldHeight(%v): WorldW() = %v, want 320", h, c.WorldW())
			}
			if c.Cols != 10 || c.Rows != 20 {
				t.Errorf("SetWorldHeight(%v): tile grid = %dx%d, want 10x20", h, c.Cols, c.Rows)
			}
		}
	})

	t.Run("VisibleTiles still clips to the tile grid", func(t *testing.T) {
		// Rows is the tile grid, not the world height. A world ten times taller
		// than the grid may not produce a row index the grid does not have — the
		// range is what a caller indexes cells with.
		c := base(t)
		c.SetWorldHeight(20000)
		c.Pan(0, 100000)
		r := c.VisibleTiles()
		if r.Row0 < 0 || r.Row1 > 20 || r.Col0 < 0 || r.Col1 > 10 {
			t.Errorf("VisibleTiles = %+v, escapes the 10x20 tile grid", r)
		}
	})

	t.Run("a hostile height leaves the camera usable", func(t *testing.T) {
		// Every other mutator guards its arguments; this one does too, so a
		// non-finite extent cannot reach the position through the clamp.
		for _, h := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -1, -1e30} {
			c := base(t)
			c.SetWorldHeight(h)
			if c.WorldH() != 0 {
				t.Errorf("SetWorldHeight(%v): WorldH() = %v, want 0", h, c.WorldH())
			}
			if math.IsNaN(c.X) || math.IsNaN(c.Y) || math.IsInf(c.X, 0) || math.IsInf(c.Y, 0) {
				t.Errorf("SetWorldHeight(%v): position went non-finite: (%v,%v)", h, c.X, c.Y)
			}
			r := c.VisibleTiles()
			if r.Row0 < 0 || r.Row1 > 20 {
				t.Errorf("SetWorldHeight(%v): VisibleTiles = %+v, escapes the grid", h, r)
			}
		}
	})
}

// AC-5: cursor-anchored zoom preserves the world point under the cursor over a
// displaced world too — the property is the camera's arithmetic, and the extent
// only decides where the clamp takes over.
func TestZoomAboutCursorOverDisplacedWorld(t *testing.T) {
	const eps = 1e-9

	// The flat world of a 200x200 grid is 6400 tall. Both of these are away from
	// it by the whole 255-unit altitude spread, in each direction, and both stay
	// comfortably taller than the view at every zoom below.
	for _, worldH := range []float64{6400 - 255, 6400 + 255} {
		c := New(200, 200, 800, 600)
		c.SetWorldHeight(worldH)
		c.X, c.Y = 2000, 2000 // well away from every edge, so the clamp cannot interfere
		c.Clamp()

		for _, factor := range []float64{2, 0.5, 1.25, 0.8} {
			sx, sy := 300.0, 220.0
			beforeX, beforeY := c.ScreenToWorld(sx, sy)

			c.ZoomAbout(sx, sy, factor)

			afterX, afterY := c.ScreenToWorld(sx, sy)
			if math.Abs(afterX-beforeX) > eps || math.Abs(afterY-beforeY) > eps {
				t.Fatalf("world %v, factor %v: world point moved (%v,%v) -> (%v,%v)",
					worldH, factor, beforeX, beforeY, afterX, afterY)
			}
			if c.WorldH() != worldH {
				t.Fatalf("world %v: WorldH() = %v after a zoom", worldH, c.WorldH())
			}
		}
	}

	t.Run("an axis smaller than the view stays centred under zoom", func(t *testing.T) {
		// The clamp winning on purpose, on the axis a short displaced world
		// produces: Y is centred rather than anchored, and X still anchors.
		c := New(200, 200, 800, 600)
		c.SetWorldHeight(300)
		c.X = 2000
		c.Clamp()

		for _, factor := range []float64{2, 0.5, 1.25, 0.8} {
			sx, sy := 300.0, 220.0
			beforeX, _ := c.ScreenToWorld(sx, sy)

			c.ZoomAbout(sx, sy, factor)

			afterX, _ := c.ScreenToWorld(sx, sy)
			if math.Abs(afterX-beforeX) > eps {
				t.Errorf("factor %v: X world point moved %v -> %v", factor, beforeX, afterX)
			}
			vw, vh := c.viewWorld()
			assertAxis(t, "X", c.X, c.WorldW(), vw)
			assertAxis(t, "Y", c.Y, c.WorldH(), vh)
		}
	})
}

func TestRandomizedDisplacedWorldKeepsInvariants(t *testing.T) {
	rng := rand.New(rand.NewSource(13007))

	for trial := 0; trial < 200; trial++ {
		cols := 1 + rng.Intn(300)
		rows := 1 + rng.Intn(300)
		viewW := 1 + rng.Intn(1600)
		viewH := 1 + rng.Intn(1200)

		c := New(cols, rows, viewW, viewH)
		if got, want := c.WorldH(), float64(rows)*CellSize; got != want {
			t.Fatalf("New(%d,%d,...): WorldH() = %v, want the flat %v", cols, rows, got, want)
		}

		for step := 0; step < 40; step++ {
			switch rng.Intn(4) {
			case 0:
				c.Pan((rng.Float64()-0.5)*20000, (rng.Float64()-0.5)*20000)
			case 1:
				c.ZoomAbout(rng.Float64()*float64(viewW), rng.Float64()*float64(viewH), 0.25+rng.Float64()*4)
			case 2:
				c.SetZoom(rng.Float64() * 16)
			case 3:
				// Drawn from both sides of the flat height, because MaxV-MinV is
				// bounded below by one cell and not by Rows*CellSize.
				c.SetWorldHeight(rng.Float64() * float64(rows) * CellSize * 2)
			}

			vw, vh := c.viewWorld()
			assertAxis(t, "X", c.X, c.WorldW(), vw)
			assertAxis(t, "Y", c.Y, c.WorldH(), vh)

			if c.Zoom < ZoomMin || c.Zoom > ZoomMax {
				t.Fatalf("zoom %v outside [%v,%v]", c.Zoom, ZoomMin, ZoomMax)
			}

			// The width is still the tile grid's, and the tile grid is still what the
			// range is clipped to: no world height reaches either.
			if got, want := c.WorldW(), float64(cols)*CellSize; got != want {
				t.Fatalf("WorldW() = %v, want %v", got, want)
			}
			r := c.VisibleTiles()
			if r.Col0 < 0 || r.Row0 < 0 || r.Col1 > cols || r.Row1 > rows || r.Col0 > r.Col1 || r.Row0 > r.Row1 {
				t.Fatalf("VisibleTiles %+v escapes a %dx%d grid", r, cols, rows)
			}
		}
	}
}

func TestAdaptiveClampUsesOnlyTheHorizontalIntervalNowInView(t *testing.T) {
	c := New(40, 40, 200, 100)
	c.SetAdaptiveClampBounds(100, 50, 900, 950,
		func(viewMinX, _ float64) (float64, float64) {
			if viewMinX < 500 {
				return 100, 600
			}
			return 200, 900
		})

	c.Pan(-1e9, 1e9)
	if c.X != 100 || c.Y != 500 {
		t.Fatalf("left interval = (%v,%v), want (100,500)", c.X, c.Y)
	}

	// One diagonal mutation must adopt the new interval's deeper bottom at
	// once. A resolver refreshed only by a later viewer frame would stop at the
	// old 500 here and reproduce the cropped map edge.
	c.Pan(1e9, 1e9)
	if c.X != 700 || c.Y != 800 {
		t.Fatalf("right interval = (%v,%v), want (700,800)", c.X, c.Y)
	}
}

// SC-1, AC-1: a cursor names the cell holding camXY + screen/zoom, over several
// positions and zooms.
//
// The wanted cell is hand-computed from the camera's post-clamp position, the
// zoom and CellSize — the independent oracle. The containment assertion beside it
// states AC-1's relation itself: the cell the pick returns is the one that HOLDS
// the world point the camera's own inverse gives, at both ends of the half-open
// span.
func TestScreenToCellNamesTheCellUnderTheCursor(t *testing.T) {
	tests := []struct {
		name         string
		cols, rows   int
		viewW, viewH int
		x, y, zoom   float64
		sx, sy       float64
		col, row     int
	}{
		// A 100x100 grid is a 3200x3200 world: bigger than the view on both
		// axes at every zoom below, so the position is the one that was set.
		{"the origin resolves to the first cell", 100, 100, 800, 600, 0, 0, 1, 0, 0, 0, 0},
		{"just short of the first cell's far edge", 100, 100, 800, 600, 0, 0, 1, 31.5, 31.5, 0, 0},
		{"exactly on it, which opens the next cell", 100, 100, 800, 600, 0, 0, 1, 32, 32, 1, 1},
		{"native zoom, mid-view", 100, 100, 800, 600, 0, 0, 1, 100, 200, 3, 6},
		{"panned, at the view's own origin", 100, 100, 800, 600, 1000, 500, 1, 0, 0, 31, 15},
		{"panned, a cell boundary crossed on screen", 100, 100, 800, 600, 1000, 500, 1, 48, 12, 32, 16},
		{"zoomed in, so a screen pixel is half a world pixel", 100, 100, 800, 600, 1000, 500, 2, 200, 100, 34, 17},
		{"zoomed in, near the view's origin", 100, 100, 800, 600, 1000, 500, 2, 1, 3, 31, 15},
		{"zoomed out, so a screen pixel is two world pixels", 100, 100, 800, 600, 1000, 500, 0.5, 400, 300, 56, 34},
		{"zoomed out, near the view's origin", 100, 100, 800, 600, 1000, 500, 0.5, 2, 2, 31, 15},
		{"a fractional position and zoom", 100, 100, 800, 600, 17.75, 9.25, 2.5, 25, 25, 0, 0},
		{"a fractional position, a cell boundary crossed", 100, 100, 800, 600, 17.75, 9.25, 2.5, 60, 60, 1, 1},
		// A 10x8 grid is a 320x256 world inside an 800x600 window, so both axes
		// are centred and the world origin sits at screen (240, 172): the pick
		// runs over NEGATIVE world coordinates on the way in.
		{"a centred axis, mid-grid", 10, 8, 800, 600, 0, 0, 1, 400, 300, 5, 4},
		{"a centred axis, exactly on the world origin", 10, 8, 800, 600, 0, 0, 1, 240, 172, 0, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := New(tc.cols, tc.rows, tc.viewW, tc.viewH)
			c.SetZoom(tc.zoom)
			c.X, c.Y = tc.x, tc.y
			c.Clamp()

			col, row, inside := c.ScreenToCell(tc.sx, tc.sy)
			if !inside {
				t.Fatalf("ScreenToCell(%v,%v) = outside, want the cell (%d,%d)", tc.sx, tc.sy, tc.col, tc.row)
			}
			if col != tc.col || row != tc.row {
				t.Fatalf("ScreenToCell(%v,%v) = (%d,%d), want (%d,%d)", tc.sx, tc.sy, col, row, tc.col, tc.row)
			}

			// AC-1: the cell returned is the one holding camXY + screen/zoom.
			wx, wy := c.ScreenToWorld(tc.sx, tc.sy)
			if wx < float64(col*CellSize) || wx >= float64((col+1)*CellSize) {
				t.Errorf("world x %v is not inside cell column %d = [%d,%d)", wx, col, col*CellSize, (col+1)*CellSize)
			}
			if wy < float64(row*CellSize) || wy >= float64((row+1)*CellSize) {
				t.Errorf("world y %v is not inside cell row %d = [%d,%d)", wy, row, row*CellSize, (row+1)*CellSize)
			}
		})
	}
}

// SC-1, AC-1: a cell beyond each of the four edges comes back marked outside
// rather than clamped.
//
// Every outside case reads the BOOL ALONE. Nothing promises col and row a value
// once inside is false, and a test that pinned one would be pinning the very
// thing the contract refuses to say — so the shape of this loop is part of the
// criterion (SC-1).
func TestScreenToCellReportsOutsideNeverClamped(t *testing.T) {
	// 10x8 tiles is a 320x256 world in an 800x600 window: both axes centre, so
	// the whole grid sits on screen with a margin on all four sides and each of
	// the four edges is reachable by a cursor. At native zoom the inverse is
	// wx = sx-240, wy = sy-172.
	c := New(10, 8, 800, 600)
	if c.X != -240 || c.Y != -172 {
		t.Fatalf("setup: camera at (%v,%v), want the centred (-240,-172)", c.X, c.Y)
	}

	outside := []struct {
		name   string
		sx, sy float64
	}{
		{"past the left edge", 100, 300},
		{"past the top edge", 400, 50},
		{"past the right edge", 700, 300},
		{"past the bottom edge", 400, 560},
		{"past two edges at once", 100, 50},
		// Half a world pixel outside. A truncation towards zero would name
		// cell 0 here and call it inside; math.Floor names -1.
		{"just past the left edge, where a truncation would fall back into cell 0", 239.5, 300},
		{"just past the top edge, likewise", 400, 171.5},
		// The far extent is half-open: Cols*CellSize is the first world pixel
		// the grid does not have.
		{"exactly on the right extent", 560, 300},
		{"exactly on the bottom extent", 400, 428},
	}
	for _, tc := range outside {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, inside := c.ScreenToCell(tc.sx, tc.sy); inside {
				t.Errorf("ScreenToCell(%v,%v) reported inside; want outside", tc.sx, tc.sy)
			}
		})
	}

	// The other side of each of those boundaries, so "outside" is discriminating
	// rather than a constant.
	in := []struct {
		name     string
		sx, sy   float64
		col, row int
	}{
		{"the first cell opens exactly at the world origin", 240, 172, 0, 0},
		{"the last cell reaches to just short of the extent", 559.5, 427.5, 9, 7},
		{"the middle of the grid", 400, 300, 5, 4},
	}
	for _, tc := range in {
		t.Run(tc.name, func(t *testing.T) {
			col, row, inside := c.ScreenToCell(tc.sx, tc.sy)
			if !inside || col != tc.col || row != tc.row {
				t.Errorf("ScreenToCell(%v,%v) = (%d,%d,%v), want (%d,%d,true)",
					tc.sx, tc.sy, col, row, inside, tc.col, tc.row)
			}
		})
	}

	t.Run("a non-finite cursor is outside, not a converted int", func(t *testing.T) {
		for _, p := range [][2]float64{
			{math.NaN(), 300}, {400, math.NaN()}, {math.NaN(), math.NaN()},
			{math.Inf(1), 300}, {math.Inf(-1), 300}, {400, math.Inf(1)}, {400, math.Inf(-1)},
		} {
			if _, _, inside := c.ScreenToCell(p[0], p[1]); inside {
				t.Errorf("ScreenToCell(%v,%v) reported inside; want outside", p[0], p[1])
			}
		}
	})
}

// SC-1, AC-1: the round trip, asserted both ways — a screen point taken to world
// and back is itself, a world point taken to screen and back is itself — and the
// pick agrees with it: a cell's own centre, put on screen, resolves to that cell.
func TestScreenToCellRoundTripsBothWays(t *testing.T) {
	const eps = 1e-9

	c := New(200, 150, 800, 600)
	c.SetZoom(2.5)
	c.Pan(1234, 567)

	for _, p := range [][2]float64{{0, 0}, {400, 300}, {799, 599}, {-20, 40}} {
		wx, wy := c.ScreenToWorld(p[0], p[1])
		sx, sy := c.WorldToScreen(wx, wy)
		if math.Abs(sx-p[0]) > eps || math.Abs(sy-p[1]) > eps {
			t.Errorf("screen->world->screen (%v,%v) -> (%v,%v)", p[0], p[1], sx, sy)
		}
	}

	for _, w := range [][2]float64{{0, 0}, {160, 224}, {6399, 4799}, {1234.5, 567.25}} {
		sx, sy := c.WorldToScreen(w[0], w[1])
		wx, wy := c.ScreenToWorld(sx, sy)
		if math.Abs(wx-w[0]) > eps || math.Abs(wy-w[1]) > eps {
			t.Errorf("world->screen->world (%v,%v) -> (%v,%v)", w[0], w[1], wx, wy)
		}
	}

	// The pick over that same trip: a cell's centre put on screen names that
	// cell, the last cell of a 200x150 grid included.
	for _, cell := range [][2]int{{0, 0}, {5, 7}, {100, 42}, {199, 149}} {
		wx := float64(cell[0]*CellSize) + CellSize/2
		wy := float64(cell[1]*CellSize) + CellSize/2
		sx, sy := c.WorldToScreen(wx, wy)

		col, row, inside := c.ScreenToCell(sx, sy)
		if !inside || col != cell[0] || row != cell[1] {
			t.Errorf("cell (%d,%d) centre -> screen (%v,%v) -> (%d,%d,%v), want the same cell",
				cell[0], cell[1], sx, sy, col, row, inside)
		}
	}
}

// SC-1, C-1: the pick is the FLAT ground cell. The only height this package holds
// is the world height the camera was told, and no term of it enters the answer —
// so the cell is the same over a world taller and shorter than the flat one.
func TestScreenToCellIgnoresTheWorldHeight(t *testing.T) {
	c := New(100, 100, 800, 600)
	c.X, c.Y = 1000, 500
	c.Clamp()

	col, row, inside := c.ScreenToCell(300, 220)
	if !inside || col != 40 || row != 22 {
		t.Fatalf("flat: ScreenToCell(300,220) = (%d,%d,%v), want (40,22,true)", col, row, inside)
	}

	// The flat world of a 100-row grid is 3200 tall, and 255 is the whole
	// altitude spread a displaced one can add or lose. Each height below leaves
	// Y = 500 in bounds (h-600 >= 500), so SetWorldHeight's re-clamp does not
	// move the camera and any change in the answer would be the height entering
	// the pick.
	for _, h := range []float64{1100, 3200 - 255, 3200 + 255, 100000} {
		c.SetWorldHeight(h)
		if c.Y != 500 {
			t.Fatalf("SetWorldHeight(%v) moved the camera to Y = %v; the case no longer isolates the height", h, c.Y)
		}
		gotCol, gotRow, gotInside := c.ScreenToCell(300, 220)
		if gotCol != col || gotRow != row || gotInside != inside {
			t.Errorf("SetWorldHeight(%v): ScreenToCell(300,220) = (%d,%d,%v), want the flat (%d,%d,%v)",
				h, gotCol, gotRow, gotInside, col, row, inside)
		}
	}
}

// The 0030 SC-2 fixture: 10x8 tiles is a 320x256 world in an 800x600 window, so
// both axes centre and the world origin sits at screen (240,172). At native zoom
// the inverse is wx = sx-240, wy = sy-172, and cell (col,row) covers the screen
// band [240+32col, 240+32col+32) x [172+32row, 172+32row+32) — so the whole grid
// is on screen with a margin on all four sides, and every case below is a real
// cursor position inside the window.
func rangeFixture(t *testing.T) *Camera {
	t.Helper()
	c := New(10, 8, 800, 600)
	if c.X != -240 || c.Y != -172 || c.Zoom != 1 {
		t.Fatalf("setup: camera at (%v,%v) zoom %v, want the centred (-240,-172) at 1", c.X, c.Y, c.Zoom)
	}
	return c
}

// TestScreenToCellRangeAtOnePointIsTheCellScreenToCellNames — 0030 SC-2: a
// rectangle whose two corners coincide covers exactly one cell, and it is
// the cell ScreenToCell names for that very point.
//
// THE EXPECTATION COMES OUT OF ScreenToCell, deliberately: what this criterion
// pins is that the band and the point answer to ONE transform, so a literal
// written here would pin the arithmetic twice and say nothing about the two
// agreeing. The half-open +1 is what makes the degenerate case one cell rather
// than none, and it is asserted on every row.
func TestScreenToCellRangeAtOnePointIsTheCellScreenToCellNames(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cols, rows int
		x, y, zoom float64
		sx, sy     float64
	}{
		{"the origin, native zoom", 100, 100, 0, 0, 1, 0, 0},
		{"just short of the first cell's far edge", 100, 100, 0, 0, 1, 31.5, 31.5},
		{"exactly on it, which opens the next cell", 100, 100, 0, 0, 1, 32, 32},
		{"panned, at the view's own origin", 100, 100, 1000, 500, 1, 0, 0},
		{"panned, a cell boundary crossed on screen", 100, 100, 1000, 500, 1, 48, 12},
		{"zoomed in, so a screen pixel is half a world pixel", 100, 100, 1000, 500, 2, 200, 100},
		{"zoomed out, so a screen pixel is two world pixels", 100, 100, 1000, 500, 0.5, 400, 300},
		{"a fractional position and zoom", 100, 100, 17.75, 9.25, 2.5, 25, 25},
		{"a centred axis, over negative world coordinates", 10, 8, 0, 0, 1, 400, 300},
		{"a centred axis, exactly on the world origin", 10, 8, 0, 0, 1, 240, 172},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := New(tc.cols, tc.rows, 800, 600)
			c.SetZoom(tc.zoom)
			c.X, c.Y = tc.x, tc.y
			c.Clamp()

			col, row, inside := c.ScreenToCell(tc.sx, tc.sy)
			if !inside {
				t.Fatalf("fixture: ScreenToCell(%v,%v) is outside, so there is no cell to agree with",
					tc.sx, tc.sy)
			}

			got, ok := c.ScreenToCellRange(tc.sx, tc.sy, tc.sx, tc.sy)
			if !ok {
				t.Fatalf("ScreenToCellRange over the point (%v,%v) found nothing, want the cell (%d,%d)",
					tc.sx, tc.sy, col, row)
			}
			if want := (TileRange{Col0: col, Row0: row, Col1: col + 1, Row1: row + 1}); got != want {
				t.Errorf("ScreenToCellRange = %+v, want %+v — the one cell ScreenToCell names", got, want)
			}
		})
	}
}

// TestScreenToCellRangeCoversTheCellsTheRectangleMeets — 0030 SC-2: the
// degenerate, clipped and wholly-outside rectangles, each its own case.
//
// Every expected range is written out from the fixture's own geometry above —
// the band a cell occupies on screen — and never read back off the method.
func TestScreenToCellRangeCoversTheCellsTheRectangleMeets(t *testing.T) {
	c := rangeFixture(t)

	for _, tc := range []struct {
		name           string
		x0, y0, x1, y1 float64
		want           TileRange
		ok             bool
	}{
		// Cells are read off the fixture: screen 250 -> world 10 -> col 0;
		// 350 -> 110 -> col 3; 180 -> 8 -> row 0; 300 -> 128 -> row 4.
		{"a rectangle over several cells", 250, 180, 350, 300, TileRange{0, 0, 4, 5}, true},
		{"zero width: a purely vertical drag still catches its column",
			250, 180, 250, 300, TileRange{0, 0, 1, 5}, true},
		{"zero height: a purely horizontal drag still catches its row",
			250, 180, 350, 180, TileRange{0, 0, 4, 1}, true},
		{"both points coincide", 250, 180, 250, 180, TileRange{0, 0, 1, 1}, true},

		// Straddling: screen 100 -> world -140 -> col -5, clipped to 0.
		{"straddling the left edge is clipped and still found",
			100, 300, 300, 300, TileRange{0, 4, 2, 5}, true},
		{"straddling every edge covers the whole grid",
			10, 10, 790, 590, TileRange{0, 0, 10, 8}, true},

		// Wholly outside, one case per edge. Nothing is promised about the range
		// then, so the want is the zero value the contract returns.
		{"wholly past the left edge", 0, 300, 100, 300, TileRange{}, false},
		{"wholly past the top edge", 400, 0, 400, 100, TileRange{}, false},
		{"wholly past the right edge", 700, 300, 790, 300, TileRange{}, false},
		{"wholly past the bottom edge", 400, 560, 400, 590, TileRange{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := c.ScreenToCellRange(tc.x0, tc.y0, tc.x1, tc.y1)
			if ok != tc.ok {
				t.Fatalf("ScreenToCellRange(%v,%v,%v,%v) ok = %v, want %v",
					tc.x0, tc.y0, tc.x1, tc.y1, ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("ScreenToCellRange(%v,%v,%v,%v) = %+v, want %+v",
					tc.x0, tc.y0, tc.x1, tc.y1, got, tc.want)
			}
		})
	}
}

// TestScreenToCellRangeIsOrientationIndependent — 0030 SC-2: a rectangle
// dragged in any of the four directions yields the range its corner-swapped
// twin yields, so no caller has to order the points it hands over.
func TestScreenToCellRangeIsOrientationIndependent(t *testing.T) {
	c := rangeFixture(t)

	const ax, ay, bx, by = 250.0, 180.0, 350.0, 300.0
	want, ok := c.ScreenToCellRange(ax, ay, bx, by)
	if !ok {
		t.Fatalf("the reference rectangle found no cells")
	}
	if want.Empty() {
		t.Fatalf("the reference rectangle came back empty: %+v", want)
	}

	for _, tc := range []struct {
		name           string
		x0, y0, x1, y1 float64
	}{
		{"swapped on x", bx, ay, ax, by},
		{"swapped on y", ax, by, bx, ay},
		{"swapped on both", bx, by, ax, ay},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := c.ScreenToCellRange(tc.x0, tc.y0, tc.x1, tc.y1)
			if !ok || got != want {
				t.Errorf("ScreenToCellRange(%v,%v,%v,%v) = %+v ok=%v, want %+v ok=true",
					tc.x0, tc.y0, tc.x1, tc.y1, got, ok, want)
			}
		})
	}
}

func TestScreenToCellRangeRefusesNonFiniteCorners(t *testing.T) {
	c := rangeFixture(t)
	nan := math.NaN()

	for _, tc := range []struct {
		name           string
		x0, y0, x1, y1 float64
	}{
		{"a NaN x", nan, 180, 350, 300},
		{"a NaN y", 250, nan, 350, 300},
		{"both corners NaN", nan, nan, nan, nan},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, ok := c.ScreenToCellRange(tc.x0, tc.y0, tc.x1, tc.y1); ok {
				t.Errorf("ScreenToCellRange = %+v ok=true, want nothing found", got)
			}
		})
	}

	t.Run("an infinite corner clips to the grid", func(t *testing.T) {
		got, ok := c.ScreenToCellRange(math.Inf(-1), math.Inf(-1), math.Inf(+1), math.Inf(+1))
		if want := (TileRange{0, 0, 10, 8}); !ok || got != want {
			t.Errorf("ScreenToCellRange over the infinite rectangle = %+v ok=%v, want %+v ok=true",
				got, ok, want)
		}
	})
}

func TestCenterOnPutsTheWorldPointAtTheViewCentre(t *testing.T) {
	// 100x100 tiles = 3200x3200 world px in an 800x600 view.
	for _, tc := range []struct {
		name   string
		zoom   float64
		wx, wy float64
	}{
		{"native zoom, mid-world", 1, 1600, 1600},
		{"zoomed in", 2, 1600, 1600},
		{"zoomed out", 0.5, 1600, 1600},
		{"off the lattice", 1, 1607.5, 1591.25},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := New(100, 100, 800, 600)
			c.SetZoom(tc.zoom)
			c.CenterOn(tc.wx, tc.wy)

			sx, sy := c.WorldToScreen(tc.wx, tc.wy)
			wantX, wantY := float64(c.ViewW)/2, float64(c.ViewH)/2
			if math.Abs(sx-wantX) > 1e-9 || math.Abs(sy-wantY) > 1e-9 {
				t.Errorf("the centred point draws at (%v,%v), want the view centre (%v,%v)",
					sx, sy, wantX, wantY)
			}
		})
	}
}

func TestCenterOnNearAnEdgeStaysInsideTheWorld(t *testing.T) {
	for _, tc := range []struct {
		name         string
		wx, wy       float64
		wantX, wantY float64
	}{
		{"the origin corner", 0, 0, 0, 0},
		{"the far corner", 3200, 3200, 3200 - 800, 3200 - 600},
		{"beyond the far corner", 1e9, 1e9, 3200 - 800, 3200 - 600},
		{"beyond the origin", -1e9, -1e9, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := New(100, 100, 800, 600)
			c.CenterOn(tc.wx, tc.wy)
			if c.X != tc.wantX || c.Y != tc.wantY {
				t.Errorf("CenterOn(%v,%v) = (%v,%v), want (%v,%v)",
					tc.wx, tc.wy, c.X, c.Y, tc.wantX, tc.wantY)
			}
		})
	}
}

func TestCenterOnKeepsTheClampInvariant(t *testing.T) {
	rng := rand.New(rand.NewSource(88088))

	for trial := 0; trial < 300; trial++ {
		cols := 1 + rng.Intn(300)
		rows := 1 + rng.Intn(300)
		viewW := 1 + rng.Intn(1600)
		viewH := 1 + rng.Intn(1200)

		c := New(cols, rows, viewW, viewH)
		c.SetZoom(ZoomMin + rng.Float64()*(ZoomMax-ZoomMin))
		if trial%7 == 0 {
			// A displaced world's height is not the tile grid's.
			c.SetWorldHeight(rng.Float64() * 40000)
		}

		wx := (rng.Float64() - 0.5) * 1e6
		wy := (rng.Float64() - 0.5) * 1e6
		switch trial % 11 {
		case 3:
			wx = math.NaN()
		case 5:
			wy = math.Inf(+1)
		case 7:
			wx, wy = math.Inf(-1), math.NaN()
		}
		c.CenterOn(wx, wy)

		vw, vh := c.viewWorld()
		assertAxis(t, "X", c.X, c.WorldW(), vw)
		assertAxis(t, "Y", c.Y, c.WorldH(), vh)

		r := c.VisibleTiles()
		if r.Col0 < 0 || r.Row0 < 0 || r.Col1 > cols || r.Row1 > rows || r.Col0 > r.Col1 || r.Row0 > r.Row1 {
			t.Fatalf("VisibleTiles %+v escapes a %dx%d grid after CenterOn", r, cols, rows)
		}
	}
}

func TestCenterOnLeavesASmallAxisCentredOnTheWorld(t *testing.T) {
	// 10x4 tiles = 320x128 world px, taller view than world on Y.
	c := New(10, 4, 200, 600)
	c.CenterOn(160, 64)

	if want := (128.0 - 600.0) / 2; c.Y != want {
		t.Errorf("Y = %v, want the world centred at %v", c.Y, want)
	}
	if sx, _ := c.WorldToScreen(160, 64); math.Abs(sx-100) > 1e-9 {
		t.Errorf("the wide-enough axis was not centred: point draws at x=%v, want 100", sx)
	}
}

func TestClampBoundsPreferDrawableInteriorUntilZoomMakesSlackUnavoidable(t *testing.T) {
	c := New(64, 48, 1206, 768)
	c.SetClampBounds(256, 256, 1792, 1280)

	c.Pan(-1e9, -1e9)
	if c.X != 256 || c.Y != 256 {
		t.Fatalf("near edge = (%v,%v), want drawable top-left (256,256)", c.X, c.Y)
	}
	c.Pan(1e9, 1e9)
	if c.X != 586 || c.Y != 512 {
		t.Fatalf("far edge = (%v,%v), want drawable bottom-right origin (586,512)", c.X, c.Y)
	}

	// At half zoom the view spans 2412x1536 world pixels, more than the
	// 1536x1024 drawable interior. No translation can fill either axis, so the
	// remaining slack is centred rather than hidden on an arbitrary side.
	c.SetZoom(0.5)
	if c.X != -182 || c.Y != 0 {
		t.Fatalf("zoomed-out origin = (%v,%v), want centred unavoidable slack (-182,0)", c.X, c.Y)
	}
	c.Pan(999, -999)
	if c.X != -182 || c.Y != 0 {
		t.Fatalf("pan moved a too-small drawable span to (%v,%v)", c.X, c.Y)
	}
}
