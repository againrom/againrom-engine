package ui

import (
	"image"
	"math"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/terrain"
)

// grid builds a synthetic W*H grid of tile words. No game data is involved.
func grid(w, h int, words ...uint16) terrain.Grid {
	tiles := make([]uint16, w*h)
	copy(tiles, words)
	return terrain.Grid{Width: w, Height: h, Tiles: tiles}
}

// A malformed map must be rejected before any window opens, not panic in the
// draw path.
func TestNewViewerRejectsMalformedInput(t *testing.T) {
	tests := []struct {
		name string
		g    terrain.Grid
		set  *terrain.Tileset
		want string
	}{
		{"zero width", terrain.Grid{Width: 0, Height: 4, Tiles: nil}, &terrain.Tileset{}, "non-positive size"},
		{"zero height", terrain.Grid{Width: 4, Height: 0, Tiles: nil}, &terrain.Tileset{}, "non-positive size"},
		{"negative size", terrain.Grid{Width: -2, Height: 4, Tiles: nil}, &terrain.Tileset{}, "non-positive size"},
		{"short tile slice", terrain.Grid{Width: 4, Height: 4, Tiles: make([]uint16, 5)}, &terrain.Tileset{}, "want 16"},
		{"long tile slice", terrain.Grid{Width: 2, Height: 2, Tiles: make([]uint16, 99)}, &terrain.Tileset{}, "want 4"},
		{"nil tileset", grid(2, 2), nil, "nil tileset"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := NewViewer("t", tc.g, tc.set)
			if err == nil {
				t.Fatalf("NewViewer succeeded, want error containing %q", tc.want)
			}
			if v != nil {
				t.Fatalf("NewViewer returned a viewer alongside an error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
}

// A well-formed map yields a camera sized to the world.
func TestNewViewerSetsUpCamera(t *testing.T) {
	v, err := NewViewer("Map", grid(64, 48), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	cam := v.Camera()
	if cam.Cols != 64 || cam.Rows != 48 {
		t.Fatalf("camera world = %dx%d, want 64x48", cam.Cols, cam.Rows)
	}
	if cam.Zoom != 1 {
		t.Fatalf("camera zoom = %v, want native 1", cam.Zoom)
	}
}

// Layout holds the logical height at 768 and expands the logical width to a
// wider window's aspect. The camera receives that frame less the fixed strip.
func TestLayoutExpandsTheMissionFrameAndSizesTheViewport(t *testing.T) {
	v, err := NewViewer("Map", grid(300, 300), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.hudHidden[hudPanelPack], v.hudHidden[hudPanelBook] = true, true

	for _, tc := range []struct {
		winW, winH        int
		frameW, viewportW int
	}{
		{1280, 720, 1366, 1206},
		{640, 480, 1024, 864},
		{1600, 1200, 1024, 864},
		{3440, 1440, 1835, 1675},
	} {
		w, h := v.Layout(tc.winW, tc.winH)
		if w != tc.winW || h != tc.winH {
			t.Fatalf("Layout(%d,%d) returned (%d,%d), want the window size back", tc.winW, tc.winH, w, h)
		}
		if got := v.FrameSize(); got != image.Pt(tc.frameW, MissionFrameH) {
			t.Fatalf("at %dx%d FrameSize = %v, want (%d,%d)", tc.winW, tc.winH, got, tc.frameW, MissionFrameH)
		}
		if got := v.ViewportSize(); got != image.Pt(tc.viewportW, MissionFrameH) {
			t.Fatalf("at %dx%d ViewportSize = %v, want (%d,%d)", tc.winW, tc.winH, got, tc.viewportW, MissionFrameH)
		}
		if got, want := v.place, frame.Fit(tc.frameW, MissionFrameH, tc.winW, tc.winH); got != want {
			t.Fatalf("at a %dx%d window the placement is %+v, want expanded mission frame fitted into it, %+v",
				tc.winW, tc.winH, got, want)
		}
	}

	// A zero-size layout (minimised window) must not corrupt the camera or the
	// placement.
	before := v.place
	v.Layout(0, 0)
	view := v.ViewportSize()
	if v.Camera().ViewW != view.X || v.Camera().ViewH != view.Y {
		t.Fatalf("zero layout changed the camera to %dx%d", v.Camera().ViewW, v.Camera().ViewH)
	}
	if v.place != before {
		t.Fatalf("zero layout changed the placement to %+v, want %+v", v.place, before)
	}
}

// The visible range the draw path iterates must stay inside the grid for any
// camera position, so Draw cannot index a tile word out of range.
func TestVisibleRangeStaysInsideGrid(t *testing.T) {
	v, err := NewViewer("Map", grid(40, 25), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	cam := v.Camera()
	layoutViewport(v, 800, 600)

	for _, pan := range [][2]float64{{-99999, -99999}, {99999, 99999}, {0, 0}, {512, -300}} {
		cam.Pan(pan[0], pan[1])
		r := cam.VisibleTiles()
		if r.Col0 < 0 || r.Row0 < 0 || r.Col1 > 40 || r.Row1 > 25 {
			t.Fatalf("pan %v: visible %+v escapes the 40x25 grid", pan, r)
		}
		// Every index the draw loop would form must be inside the tile slice.
		for row := r.Row0; row < r.Row1; row++ {
			for col := r.Col0; col < r.Col1; col++ {
				if i := row*v.grid.Width + col; i < 0 || i >= len(v.grid.Tiles) {
					t.Fatalf("draw index %d out of range for %d words", i, len(v.grid.Tiles))
				}
			}
		}
	}
}

// SubCellBounds pins the draw path's expected sub-cell size to the terrain
// tier's cell size, so the two cannot drift apart silently.
func TestSubCellBoundsMatchesTerrainCell(t *testing.T) {
	b := SubCellBounds()
	if b.Dx() != terrain.CellSize || b.Dy() != terrain.CellSize {
		t.Fatalf("SubCellBounds = %v, want %dx%d", b, terrain.CellSize, terrain.CellSize)
	}
}

func TestViewerAnimation(t *testing.T) {
	v, err := NewViewer("m", grid(4, 4), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}

	on, idx := v.Animation()
	if !on {
		t.Fatal("animation is off by default; the game animates water by default")
	}
	if idx != terrain.DefaultSpeedIndex {
		t.Fatalf("default speed index = %d, want %d (the index map load pushes)", idx, terrain.DefaultSpeedIndex)
	}
	if got := v.AnimationCounter(); got != 0 {
		t.Fatalf("initial counter = %d, want 0", got)
	}

	// The first advance only establishes the baseline, so a slow startup cannot
	// fire a burst.
	base := time.Unix(0, 0)
	v.advanceAnimation(base)
	if got := v.AnimationCounter(); got != 0 {
		t.Fatalf("counter = %d after the baseline call, want 0", got)
	}

	// 62 ms per tick at the default index: 620 ms is 10 ticks.
	v.advanceAnimation(base.Add(620 * time.Millisecond))
	if got := v.AnimationCounter(); got != 10 {
		t.Fatalf("counter = %d after 620 ms at 62 ms/tick, want 10", got)
	}

	// Turning animation off stops the counter but does not rewind it.
	v.SetAnimated(false)
	if on, _ := v.Animation(); on {
		t.Fatal("SetAnimated(false) did not take effect")
	}
	v.advanceAnimation(base.Add(5 * time.Second))
	if got := v.AnimationCounter(); got != 10 {
		t.Fatalf("counter = %d while animation is off, want it held at 10", got)
	}

	// A speed change re-rates the ticker; index 8 is 32 tps -> 31 ms.
	v.SetAnimated(true)
	v.SetSpeedIndex(8)
	if _, idx := v.Animation(); idx != 8 {
		t.Fatalf("speed index = %d, want 8", idx)
	}
	v.advanceAnimation(base.Add(5*time.Second + 310*time.Millisecond))
	if got := v.AnimationCounter(); got != 20 {
		t.Fatalf("counter = %d after 310 ms at 31 ms/tick, want 20", got)
	}

	// An out-of-range index clamps rather than erroring.
	v.SetSpeedIndex(99)
	if _, idx := v.Animation(); idx != terrain.SpeedIndexMax {
		t.Fatalf("speed index = %d after SetSpeedIndex(99), want %d (clamped)", idx, terrain.SpeedIndexMax)
	}
}

func TestCellImageUsesAnimatedResolveOnlyWhenAnimating(t *testing.T) {
	v, err := NewViewer("m", grid(4, 4), &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}

	// A water word: group 8 (bits 6..) with blend column 1.
	const water = uint16(8<<6) | uint16(1<<4)
	const col, row = 2, 3

	v.SetAnimated(false)
	if got, want := v.resolveCell(water, col, row), terrain.Resolve(water); got != want {
		t.Fatalf("animation off: %+v, want the static %+v", got, want)
	}

	v.SetAnimated(true)
	want := terrain.ResolveAnimated(water, col, row, v.AnimationCounter())
	if got := v.resolveCell(water, col, row); got != want {
		t.Fatalf("animation on: %+v, want the animated %+v", got, want)
	}
	if terrain.Resolve(water) == want {
		t.Fatal("the test position does not distinguish the two paths; pick another cell")
	}
}

// ---------------------------------------------------------------------------
// The displaced draw path (AC-4, AC-6, AC-6a, SC-4, SC-5, SC-6, SC-9).
//
// Every test below is windowless. An ebiten.Vertex is a struct literal, so the
// whole of the geometry is reachable without a graphics context; only the
// DrawTriangles call needs one, and it is not under test.
// ---------------------------------------------------------------------------

// SC-4, AC-4: the four vertices of one tile, pinned as literal screen pixels.
//
// The camera state is the one NewViewer leaves: the cliff fixture's world is
// 96x223 in a 1024x768 view, so both axes centre — X at (96-1024)/2 = -464 and
// Y at (223-768)/2 = -272.5 — and the zoom is native. Tile (0,1) straddles the
// cliff, so its four corners are pairwise distinct in Y: a transposed corner or
// a sign error cannot land on the right answer here, which a re-derivation of
// WorldCorner could not tell you.
func TestTileVerticesAreLiteralScreenPixels(t *testing.T) {
	v := newViewer(t, cliffGrid())
	layoutViewport(v, 1024, 768)
	if v.Mode() != ModeDisplaced {
		t.Fatalf("Mode() = %v, want displaced", v.Mode())
	}
	cam := v.Camera()
	if cam.X != -464 || cam.Y != -272.5 || cam.Zoom != 1 {
		t.Fatalf("camera is (%v,%v) at zoom %v, not the state this test's literals assume",
			cam.X, cam.Y, cam.Zoom)
	}

	got := tileVertices(cam, v.proj, 0, 1)

	// TL, TR, BL, BR — the order quadIndices assumes.
	want := [4][2]float32{
		{464, 399.5}, // mesh (0,1): world (0,127)
		{496, 272.5}, // mesh (1,1): world (32,0), the raised corner
		{464, 431.5}, // mesh (0,2): world (0,159)
		{496, 304.5}, // mesh (1,2): world (32,32)
	}
	names := [4]string{"TL", "TR", "BL", "BR"}
	for i := range want {
		if got[i].DstX != want[i][0] || got[i].DstY != want[i][1] {
			t.Errorf("%s = (%v,%v), want (%v,%v)",
				names[i], got[i].DstX, got[i].DstY, want[i][0], want[i][1])
		}
	}
}

func TestTileVerticesCorners(t *testing.T) {
	// Guards on the FIXTURES, checked once over the whole sweep: four coincident
	// corners would let a transposition pass, and a sweep of nothing but
	// axis-aligned rectangles would never reach a displaced quad at all.
	distinct, skewed := 0, 0

	for _, f := range displacedFixtures() {
		t.Run(f.name, func(t *testing.T) {
			v := newViewer(t, f.grid)
			cam := v.Camera()
			w, h := f.grid.Width, f.grid.Height

			for _, tile := range [][2]int{
				{0, 0}, {w / 2, h / 2}, {w - 1, h / 2}, {w / 2, h - 1}, {w - 1, h - 1},
			} {
				for _, state := range cameraStates() {
					state(v)
					tx, ty := tile[0], tile[1]
					verts := tileVertices(cam, v.proj, tx, ty)

					corners := [4][2]int{{tx, ty}, {tx + 1, ty}, {tx, ty + 1}, {tx + 1, ty + 1}}
					names := [4]string{"TL", "TR", "BL", "BR"}
					for i, c := range corners {
						wx, wy := v.proj.WorldCorner(c[0], c[1])
						sx, sy := cam.WorldToScreen(float64(wx), float64(wy))
						if verts[i].DstX != float32(sx) || verts[i].DstY != float32(sy) {
							t.Fatalf("tile (%d,%d) %s = (%v,%v), want mesh vertex (%d,%d) at (%v,%v)",
								tx, ty, names[i], verts[i].DstX, verts[i].DstY, c[0], c[1], float32(sx), float32(sy))
						}
					}

					// The source span is the sub-cell's own bounds, and the
					// colour is 1 across the board: the ZERO value has
					// ColorA == 0, which ebiten documents as fully transparent
					// and which would draw the whole terrain invisible.
					b := SubCellBounds()
					wantSrc := [4][2]float32{
						{float32(b.Min.X), float32(b.Min.Y)},
						{float32(b.Max.X), float32(b.Min.Y)},
						{float32(b.Min.X), float32(b.Max.Y)},
						{float32(b.Max.X), float32(b.Max.Y)},
					}
					for i := range verts {
						if verts[i].SrcX != wantSrc[i][0] || verts[i].SrcY != wantSrc[i][1] {
							t.Fatalf("tile (%d,%d) %s src = (%v,%v), want (%v,%v)",
								tx, ty, names[i], verts[i].SrcX, verts[i].SrcY, wantSrc[i][0], wantSrc[i][1])
						}
						if verts[i].ColorR != 1 || verts[i].ColorG != 1 || verts[i].ColorB != 1 || verts[i].ColorA != 1 {
							t.Fatalf("tile (%d,%d) %s colour = (%v,%v,%v,%v), want 1,1,1,1",
								tx, ty, names[i], verts[i].ColorR, verts[i].ColorG, verts[i].ColorB, verts[i].ColorA)
						}
					}

					seen := map[[2]float32]bool{}
					for i := range verts {
						seen[[2]float32{verts[i].DstX, verts[i].DstY}] = true
					}
					if len(seen) == 4 {
						distinct++
					}
					if verts[0].DstY != verts[1].DstY || verts[2].DstY != verts[3].DstY {
						skewed++
					}
				}
			}
		})
	}

	if distinct == 0 {
		t.Error("no tested tile had four distinct corners, so a transposed corner would have passed")
	}
	if skewed == 0 {
		t.Error("every tested quad had both horizontal edges level, i.e. was an axis-aligned " +
			"rectangle flat mode could already draw; the fixtures do not reach the case " +
			"this story exists for")
	}
}

// SC-4: the index list itself. {0,1,3, 0,3,2} and {0,1,2, 1,3,2} put the
// identical four corners on screen and differ only in the interior of a
// non-parallelogram tile, so the list is pinned twice — as the literal, and as
// the diagonal it makes the two triangles share.
func TestQuadIndicesSplitOnTheTopLeftBottomRightDiagonal(t *testing.T) {
	if want := [6]uint16{0, 1, 3, 0, 3, 2}; quadIndices != want {
		t.Fatalf("quadIndices = %v, want %v", quadIndices, want)
	}

	a := map[uint16]bool{quadIndices[0]: true, quadIndices[1]: true, quadIndices[2]: true}
	b := map[uint16]bool{quadIndices[3]: true, quadIndices[4]: true, quadIndices[5]: true}
	if len(a) != 3 || len(b) != 3 {
		t.Fatalf("a triangle repeats a corner: %v", quadIndices)
	}

	var shared []uint16
	for i := uint16(0); i < 4; i++ {
		if a[i] && b[i] {
			shared = append(shared, i)
		}
		if !a[i] && !b[i] {
			t.Errorf("corner %d appears in neither triangle", i)
		}
	}
	// 0 is TL and 3 is BR. The alternative shares 1-2, which is TR-BL: the
	// other diagonal, and the wrong one.
	if len(shared) != 2 || shared[0] != 0 || shared[1] != 3 {
		t.Errorf("the triangles share corners %v, want the TL-BR diagonal [0 3]", shared)
	}
}

func TestDisplacedSetCoversEveryIntersectingQuad(t *testing.T) {
	for _, f := range displacedFixtures() {
		t.Run(f.name, func(t *testing.T) {
			v := newViewer(t, f.grid)
			cam := v.Camera()

			for si, state := range cameraStates() {
				state(v)
				drawn := displacedSet(v)

				vx0, vy0 := cam.ScreenToWorld(0, 0)
				vx1, vy1 := cam.ScreenToWorld(float64(cam.ViewW), float64(cam.ViewH))

				for ty := 0; ty < f.grid.Height; ty++ {
					for tx := 0; tx < f.grid.Width; tx++ {
						x0, y0, x1, y1 := quadBox(v.proj, tx, ty)
						if !(x0 < vx1 && vx0 < x1 && y0 < vy1 && vy0 < y1) {
							continue
						}
						if !drawn[[2]int{tx, ty}] {
							t.Fatalf("state %d: tile (%d,%d) spans world [%v,%v)x[%v,%v) and meets "+
								"the view [%v,%v)x[%v,%v), but is not drawn",
								si, tx, ty, x0, x1, y0, y1, vx0, vx1, vy0, vy1)
						}
					}
				}
			}
		})
	}
}

// AC-6a, SC-5: over-covering is permitted but bounded, so "draw the whole map"
// is not an answer; and the columns are the unmodified flat ones, exactly.
//
// The rows and columns are read off the tiles the loop ACTUALLY visits, not
// recomputed from RowRange — recomputing would test the projection a second time
// and say nothing about the loop, which is what this task changed.
func TestDisplacedSetStaysInsideTheTwoSidedBound(t *testing.T) {
	narrowed, exercised := 0, 0

	for _, f := range displacedFixtures() {
		t.Run(f.name, func(t *testing.T) {
			v := newViewer(t, f.grid)
			cam := v.Camera()
			p := v.proj
			h := f.grid.Height

			up := max(0, p.MaxH)
			down := max(0, -p.MinH)

			for si, state := range cameraStates() {
				state(v)

				rows, contiguous := displacedRows(v)
				if !contiguous {
					t.Fatalf("state %d: the visited rows %v are not a contiguous range", si, rows)
				}
				r0, r1 := rows[0], rows[1]
				if r0 < 0 || r1 > h || r0 > r1 {
					t.Fatalf("state %d: rows [%d,%d) escape a %d-row map", si, r0, r1, h)
				}

				// The bound, in NATIVE V: the world window shifted back by MinV,
				// not the world window itself.
				_, top := cam.ScreenToWorld(0, 0)
				_, bottom := cam.ScreenToWorld(float64(cam.ViewW), float64(cam.ViewH))
				a := top + float64(p.MinV)
				b := bottom + float64(p.MinV)
				lo := clampRowIndex(rowsFloor(a)-ceilCells(down)-1, h)
				hi := clampRowIndex(rowsCeil(b)+ceilCells(up)+1, h)
				if r0 < lo || r1 > hi {
					t.Fatalf("state %d: rows [%d,%d) leave the two-sided bound [%d,%d)", si, r0, r1, lo, hi)
				}
				if hi-lo < h {
					narrowed++
					if r1 > r0 {
						exercised++
					}
				}

				// AC-6a: the column bounds are the flat range's, untouched.
				want := cam.VisibleTiles()
				cols := displacedColumns(v)
				if len(cols) > 0 && (cols[0] != want.Col0 || cols[len(cols)-1] != want.Col1-1) {
					t.Fatalf("state %d: columns %v, want [%d,%d)", si, cols, want.Col0, want.Col1)
				}
			}
		})
	}

	if narrowed == 0 {
		t.Error("no camera state produced a bound narrower than the whole map, so returning " +
			"every row would have passed")
	}
	if exercised == 0 {
		t.Error("every state with a narrow bound drew nothing at all, so the bound was never " +
			"actually tested against a drawn row")
	}
}

func TestDisplacedLoopVisitsRowMajor(t *testing.T) {
	v := newViewer(t, bigCliffGrid())
	cam := v.Camera()
	layoutViewport(v, 200, 160)
	cam.SetZoom(0.5)
	cam.Pan(0, 40)

	order := displacedOrder(v)
	if len(order) == 0 {
		t.Fatal("nothing was drawn; the fixture does not exercise the loop")
	}

	cols := cam.VisibleTiles()
	_, top := cam.ScreenToWorld(0, 0)
	_, bottom := cam.ScreenToWorld(float64(cam.ViewW), float64(cam.ViewH))
	r0, r1 := v.proj.RowRange(top, bottom)

	var want [][2]int
	for ty := r0; ty < r1; ty++ {
		for tx := cols.Col0; tx < cols.Col1; tx++ {
			want = append(want, [2]int{tx, ty})
		}
	}
	if len(order) != len(want) {
		t.Fatalf("drew %d tiles, want %d", len(order), len(want))
	}
	if len(want) < 4 {
		t.Fatalf("only %d tiles drawn; the fixture cannot tell row-major from column-major", len(want))
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("visit %d = %v, want %v — the loop is not row-major over [%d,%d) x [%d,%d)",
				i, order[i], want[i], cols.Col0, cols.Col1, r0, r1)
		}
	}
}

func TestInvertedQuadIsStillDrawn(t *testing.T) {
	v := newViewer(t, pushedGrid()) // the whole first row pushed to -128
	cam := v.Camera()

	verts := tileVertices(cam, v.proj, 0, 0)
	if verts[0].DstY <= verts[2].DstY {
		t.Fatalf("TL is at %v and BL at %v: the fixture's quad is not inverted",
			verts[0].DstY, verts[2].DstY)
	}
	if verts[1].DstY <= verts[3].DstY {
		t.Fatalf("TR is at %v and BR at %v: the fixture's right edge is not inverted",
			verts[1].DstY, verts[3].DstY)
	}

	drawn := displacedSet(v)
	for tx := 0; tx < pushedW; tx++ {
		if !drawn[[2]int{tx, 0}] {
			t.Errorf("inverted tile (%d,0) was not drawn", tx)
		}
	}
}

func TestResolvedCellIsTheSameInBothModes(t *testing.T) {
	// A water word: group 8 (bits 6..) with blend column 1, so ResolveAnimated
	// and Resolve genuinely differ.
	const water = uint16(8<<6) | uint16(1<<4)

	g := cliffGrid()
	for i := range g.Tiles {
		g.Tiles[i] = water
	}
	flat := grid(cliffW, cliffH)
	copy(flat.Tiles, g.Tiles)

	dv := newViewer(t, g)
	fv := newViewer(t, flat)
	if dv.Mode() != ModeDisplaced || fv.Mode() != ModeFlat {
		t.Fatalf("modes are %v and %v, want displaced and flat", dv.Mode(), fv.Mode())
	}

	base := time.Unix(0, 0)
	moved := false
	for _, ms := range []int{0, 620, 1240, 3100} {
		at := base.Add(time.Duration(ms) * time.Millisecond)
		dv.advanceAnimation(at)
		fv.advanceAnimation(at)
		if dv.AnimationCounter() != fv.AnimationCounter() {
			t.Fatalf("the two viewers' tick counters diverged: %d vs %d",
				dv.AnimationCounter(), fv.AnimationCounter())
		}

		for ty := 0; ty < cliffH; ty++ {
			for tx := 0; tx < cliffW; tx++ {
				if dv.tileWord(tx, ty) != fv.tileWord(tx, ty) {
					t.Fatalf("tile (%d,%d): the two modes read different words", tx, ty)
				}
				d := dv.resolveCell(dv.tileWord(tx, ty), tx, ty)
				f := fv.resolveCell(fv.tileWord(tx, ty), tx, ty)
				if d != f {
					t.Fatalf("tile (%d,%d) at tick %d: displaced resolves %+v, flat %+v",
						tx, ty, dv.AnimationCounter(), d, f)
				}
				if d != terrain.Resolve(water) {
					moved = true
				}
			}
		}
	}
	if !moved {
		t.Error("the phase never left the static mapping, so the water substitution was not exercised")
	}
}

func TestDisplacedGeometryDoesNotMutateTheAltitudes(t *testing.T) {
	alts := append([]uint8(nil), bigCliffAlts...)
	before := append([]uint8(nil), alts...)

	g := grid(bigCliffW, bigCliffH)
	g.Altitudes = alts
	v := newViewer(t, g)

	for _, state := range cameraStates() {
		state(v)
		v.forEachDisplacedTile(func(tx, ty int) {
			_ = tileVertices(v.Camera(), v.proj, tx, ty)
		})
	}

	for i := range before {
		if alts[i] != before[i] {
			t.Fatalf("altitude %d changed from %d to %d", i, before[i], alts[i])
		}
	}
}

// --- fixtures and helpers for the displaced draw path ---

const (
	// bigCliff is a 4x10 grid whose first five rows are pushed to -128 and whose
	// last five are raised 127, so MinV is 33 rather than 0 and the mesh is far
	// from monotonic: exactly where an omitted MinV term stops hiding.
	bigCliffW, bigCliffH = 4, 10

	ditchW, ditchH = 1, 8
)

var (
	bigCliffAlts = func() []uint8 {
		a := make([]uint8, bigCliffW*bigCliffH)
		for r := 0; r < bigCliffH; r++ {
			for c := 0; c < bigCliffW; c++ {
				if r < 5 {
					a[r*bigCliffW+c] = 0x80
				} else {
					a[r*bigCliffW+c] = 127
				}
			}
		}
		return a
	}()

	ditchAlts = func() []uint8 {
		a := make([]uint8, ditchW*ditchH)
		for r := 0; r < 4; r++ {
			a[r] = 0x80
		}
		return a
	}()
)

func bigCliffGrid() terrain.Grid { return altGrid(bigCliffW, bigCliffH, bigCliffAlts...) }
func ditchGrid() terrain.Grid    { return altGrid(ditchW, ditchH, ditchAlts...) }

type displacedFixture struct {
	name string
	grid terrain.Grid
}

func displacedFixtures() []displacedFixture {
	return []displacedFixture{
		{"flat but valid", altGrid(6, 6, make([]uint8, 36)...)},
		{"raised cliff", cliffGrid()},
		{"pushed row", pushedGrid()},
		{"short world", shortGrid()},
		{"push then raise", bigCliffGrid()},
		{"the 1x8 ditch", ditchGrid()},
	}
}

// cameraStates returns the camera positions each geometry test is swept over:
// hard against every edge and at both ends of the zoom range, because
// RowRange's anchoring is invisible to a containment test taken near the middle
// of a map.
func cameraStates() []func(*Viewer) {
	type state struct {
		w, h   int
		zoom   float64
		dx, dy float64
	}
	states := []state{
		{1024, 768, 1, 0, 0},
		{1024, 768, 1, 0, -1e6},
		{1024, 768, 1, 0, 1e6},
		{320, 240, 1, 0, -1e6},
		{320, 240, 1, 0, 1e6},
		{320, 240, 1, 1e6, 40},
		{200, 100, 4, 0, -1e6},
		{200, 100, 4, 0, 1e6},
		{200, 100, 4, 48, 60},
		{800, 600, 0.125, 0, -1e6},
		{800, 600, 0.125, 0, 1e6},
		{96, 64, 2, 0, 96},
	}
	out := make([]func(*Viewer), 0, len(states))
	for _, s := range states {
		out = append(out, func(v *Viewer) {
			cam := v.Camera()
			layoutViewport(v, s.w, s.h)
			cam.SetZoom(s.zoom)
			cam.X, cam.Y = 0, 0
			cam.Clamp()
			cam.Pan(s.dx, s.dy)
		})
	}
	return out
}

// quadBox is the axis-aligned box of tile (tx,ty)'s four displaced world
// corners. AC-6 takes it half-open on both axes, as the flat range already is.
func quadBox(p *terrain.Projection, tx, ty int) (x0, y0, x1, y1 float64) {
	first := true
	for _, d := range [4][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
		wx, wy := p.WorldCorner(tx+d[0], ty+d[1])
		fx, fy := float64(wx), float64(wy)
		if first {
			x0, y0, x1, y1 = fx, fy, fx, fy
			first = false
			continue
		}
		x0, y0 = math.Min(x0, fx), math.Min(y0, fy)
		x1, y1 = math.Max(x1, fx), math.Max(y1, fy)
	}
	return x0, y0, x1, y1
}

func displacedOrder(v *Viewer) [][2]int {
	var out [][2]int
	v.forEachDisplacedTile(func(tx, ty int) { out = append(out, [2]int{tx, ty}) })
	return out
}

func displacedSet(v *Viewer) map[[2]int]bool {
	out := map[[2]int]bool{}
	v.forEachDisplacedTile(func(tx, ty int) { out[[2]int{tx, ty}] = true })
	return out
}

// displacedRows is the half-open row range the loop actually visited, [r0,r1),
// and whether the rows it touched form a contiguous run. An empty visit yields
// [0,0), contiguous.
func displacedRows(v *Viewer) (rows [2]int, contiguous bool) {
	seen := map[int]bool{}
	lo, hi := 0, 0
	first := true
	v.forEachDisplacedTile(func(tx, ty int) {
		seen[ty] = true
		if first || ty < lo {
			lo = ty
		}
		if first || ty >= hi {
			hi = ty + 1
		}
		first = false
	})
	if first {
		return [2]int{0, 0}, true
	}
	return [2]int{lo, hi}, len(seen) == hi-lo
}

// displacedColumns is the distinct column indices the loop visits, in the order
// it first reaches them.
func displacedColumns(v *Viewer) []int {
	seen := map[int]bool{}
	var out []int
	v.forEachDisplacedTile(func(tx, ty int) {
		if !seen[tx] {
			seen[tx] = true
			out = append(out, tx)
		}
	})
	return out
}

// rowsFloor and rowsCeil quantise a native V coordinate to cell rows the way the
// FLAT range does, which is what the spec's two-sided bound is written against.
func rowsFloor(v float64) int { return int(math.Floor(v / terrain.CellSize)) }
func rowsCeil(v float64) int  { return int(math.Ceil(v / terrain.CellSize)) }

// ceilCells is ceil(n/CellSize) for a non-negative altitude magnitude.
func ceilCells(n int) int { return (n + terrain.CellSize - 1) / terrain.CellSize }

// clampRowIndex holds a row index in [0, height], the inclusive upper bound of a
// half-open range.
func clampRowIndex(r, height int) int { return min(max(r, 0), height) }

// TestThePressPointOutlivesTheDragAnchor — 0030 SC-1: the press point is
// written where the gesture begins and is still readable after several ticks
// of movement, which is exactly what the drag anchor is not.
//
// The two are asserted TOGETHER on the same gesture. dragX/dragY is overwritten
// by every held tick, so a build that stored the press point in it would pass a
// test that only read pressX/pressY on the anchor tick; the second assertion is
// what says the two fields are still different fields three ticks in.
func TestThePressPointOutlivesTheDragAnchor(t *testing.T) {
	const pressAtX, pressAtY = 400, 300

	v := stepViewer(t)
	v.commandMode = true

	v.step(Input{PrimaryDown: true, CursorX: pressAtX, CursorY: pressAtY}, dragFrozen)
	for _, p := range [][2]int{{410, 305}, {430, 280}, {370, 340}} {
		v.step(Input{PrimaryDown: true, CursorX: p[0], CursorY: p[1]}, dragFrozen)
	}

	if v.pressX != pressAtX || v.pressY != pressAtY {
		t.Errorf("the press point reads (%d,%d) after three ticks of movement, want (%d,%d)",
			v.pressX, v.pressY, pressAtX, pressAtY)
	}
	if v.dragX == pressAtX && v.dragY == pressAtY {
		t.Errorf("the drag anchor is still at the press point (%d,%d), so this gesture cannot tell "+
			"the two apart", v.dragX, v.dragY)
	}

	// A release, then a fresh press: the point moves with the new gesture rather
	// than surviving it, so nothing here outlives the press it belongs to.
	v.step(Input{CursorX: 370, CursorY: 340}, dragFrozen)
	v.step(Input{PrimaryDown: true, CursorX: 200, CursorY: 100}, dragFrozen)
	if v.pressX != 200 || v.pressY != 100 {
		t.Errorf("after a release and a fresh press the press point reads (%d,%d), want (200,100)",
			v.pressX, v.pressY)
	}
}
