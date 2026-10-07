package terrain_test

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
	"testing"
	"time"

	terrain "againrom/pkg/render/terrain"
)

// projCell is the destination rows one map row advances the mesh by, and the
// only place a cell's size enters the vertex formula. Guarded against
// terrain.CellSize below.
const projCell = 32

func projRecipeAltitude(alt []uint8, w, h, c, r int) int {
	ci, ri := c, r
	if ci > w-1 {
		ci = w - 1
	}
	if ri > h-1 {
		ri = h - 1
	}
	return int(int8(alt[ri*w+ci]))
}

func projRecipeVertex(alt []uint8, w, h, c, r int) int {
	return r*projCell - projRecipeAltitude(alt, w, h, c, r)
}

func projRecipeCanvas(alt []uint8, w, h int) (minV, maxV int) {
	minV = projRecipeVertex(alt, w, h, 0, 0)
	maxV = minV
	for r := 0; r <= h; r++ {
		for c := 0; c <= w; c++ {
			v := projRecipeVertex(alt, w, h, c, r)
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
	}
	return minV, maxV
}

// projFixture is one synthetic altitude grid. Nothing is stored with it: every
// expectation is computed from the recipe above, or listed as literals beside
// the fixture that carries them.
type projFixture struct {
	name string
	w, h int
	alt  []uint8
}

// projUniform is a grid whose every altitude is the same byte.
func projUniform(w, h int, a uint8) []uint8 {
	alt := make([]uint8, w*h)
	for i := range alt {
		alt[i] = a
	}
	return alt
}

// projSweepGrid walks the whole byte range: 37 is coprime with 256, so a 16x16
// grid holds every value from 0x00 to 0xff exactly once — 0x80 and 0x7f with
// them, and every altitude difference the format can produce somewhere in it.
func projSweepGrid() []uint8 {
	alt := make([]uint8, 16*16)
	for i := range alt {
		alt[i] = uint8((i*37 + 11) & 0xff)
	}
	return alt
}

// projFixtureFarEdge is AC-3's grid: it holds 0x80 and 0x7f, and it is built so
// that both far-edge reads are decidable. Signed, it is
//
//	row 0:   0   127    64
//	row 1:   1  -128   127
//
// At vertex (3,0) the engine's own raw flat read would land on index 3 — the
// next row's column 0, altitude 1 — where the clamp gives column 2's 64; the two
// differ, so a lost column clamp is a wrong value here, not a coincidence. At
// every vertex of row 2 the raw read runs off the end of the grid entirely.
var projFixtureFarEdge = projFixture{
	name: "signed extremes on the far edge",
	w:    3, h: 2,
	alt: []uint8{0x00, 0x7f, 0x40, 0x01, 0x80, 0x7f},
}

// projFarEdgeVertices is that fixture's whole mesh, hand-computed from
// V(c,r) = r*32 - h(c,r), rows r = 0..2 and columns c = 0..3:
//
//	r=0:  -0,      -127,      -64,   (c=3 clamps to c=2)  -64
//	r=1:  32-1=31, 32+128=160, 32-127=-95,                -95
//	r=2:  64-1=63, 64+128=192, 64-127=-63,                -63
//
// The least is -127 at (1,0) and the greatest 192 at (1,2) — the greatest on the
// r = Height row, which exists only because the mesh is (Height+1) tall, and
// strictly above the 160 of the row before it.
var projFarEdgeVertices = [][]int{
	{0, -127, -64, -64},
	{31, 160, -95, -95},
	{63, 192, -63, -63},
}

const (
	projFarEdgeMinV = -127
	projFarEdgeMaxV = 192
)

// projFixtureColumn is the one-column grid whose altitudes are 0, 5, 0. Its
// vertices are 0, 27, 64 and 96, so its canvas is [0, 96) — 96 rows, exactly the
// flat render's 3*32, over a mesh no flat render produces.
var projFixtureColumn = projFixture{
	name: "one column, relief that leaves the canvas height alone",
	w:    1, h: 3,
	alt: []uint8{0x00, 0x05, 0x00},
}

var projColumnVertices = [][]int{{0, 0}, {27, 27}, {64, 64}, {96, 96}}

// projFixtureSingle is the smallest map there is: one cell, four vertices, all
// of them the same altitude read through the clamp.
var projFixtureSingle = projFixture{
	name: "single cell",
	w:    1, h: 1,
	alt: []uint8{0x05},
}

var projSingleVertices = [][]int{{-5, -5}, {27, 27}}

// projFixtures is every grid the sweeps run over.
var projFixtures = []projFixture{
	projFixtureFarEdge,
	projFixtureColumn,
	projFixtureSingle,
	{"uniform zero", 4, 3, projUniform(4, 3, 0x00)},
	{"uniform 0x7f", 4, 3, projUniform(4, 3, 0x7f)},
	{"uniform 0x80", 4, 3, projUniform(4, 3, 0x80)},
	{"every byte value", 16, 16, projSweepGrid()},
	{"one row", 5, 1, []uint8{0x00, 0x7f, 0x80, 0x40, 0xff}},
	{"tall and thin", 2, 9, projSweepGrid()[:18]},
}

// projCheckMesh compares a Projection's whole mesh against a literal table.
func projCheckMesh(t *testing.T, name string, p terrain.Projection, want [][]int) {
	t.Helper()
	for r := range want {
		for c := range want[r] {
			if got := p.Vertex(c, r); got != want[r][c] {
				t.Errorf("%s: Vertex(%d,%d) = %d, want %d", name, c, r, got, want[r][c])
			}
		}
	}
}

// TestProjectVertices covers SC-1 / AC-3: the vertex formula and its out-of-grid
// clamp, the signed altitude read and the direction it displaces in, the canvas
// bounds and the origin, and both output dimensions at several scales.
func TestProjectVertices(t *testing.T) {
	if terrain.CellSize != projCell {
		t.Fatalf("CellSize = %d, want %d; the vertex formula's r*%d term is stated over %d-pixel cells",
			terrain.CellSize, projCell, projCell, projCell)
	}

	t.Run("hand-computed meshes", func(t *testing.T) {
		// Literal expectations, so the recipe transcription cannot drift with
		// the implementation it checks.
		f := projFixtureFarEdge
		p := terrain.Project(f.alt, f.w, f.h)
		projCheckMesh(t, f.name, p, projFarEdgeVertices)
		if p.MinV != projFarEdgeMinV || p.MaxV != projFarEdgeMaxV {
			t.Errorf("%s: canvas [%d,%d), want [%d,%d)", f.name, p.MinV, p.MaxV, projFarEdgeMinV, projFarEdgeMaxV)
		}
		if got, want := p.CanvasHeight(), projFarEdgeMaxV-projFarEdgeMinV; got != want {
			t.Errorf("%s: CanvasHeight = %d, want %d", f.name, got, want)
		}

		f = projFixtureColumn
		p = terrain.Project(f.alt, f.w, f.h)
		projCheckMesh(t, f.name, p, projColumnVertices)
		// The case the canvas cannot decide: 96 rows either way, and only the
		// mesh above says whether the middle vertex rose or fell.
		if got := p.CanvasHeight(); got != 96 {
			t.Errorf("%s: CanvasHeight = %d, want 96 (the flat render's 3*32)", f.name, got)
		}
		if p.MinV != 0 || p.MaxV != 96 {
			t.Errorf("%s: canvas [%d,%d), want [0,96)", f.name, p.MinV, p.MaxV)
		}

		f = projFixtureSingle
		p = terrain.Project(f.alt, f.w, f.h)
		projCheckMesh(t, f.name, p, projSingleVertices)
		if p.MinV != -5 || p.MaxV != 27 {
			t.Errorf("%s: canvas [%d,%d), want [-5,27)", f.name, p.MinV, p.MaxV)
		}
	})

	t.Run("the vertex formula over the whole mesh", func(t *testing.T) {
		// Every vertex of every fixture, c in [0,W] and r in [0,H] — the mesh is
		// one wider and one taller than the cell grid, and both of those extra
		// lines are read here.
		for _, f := range projFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			if p.Width != f.w || p.Height != f.h {
				t.Fatalf("%s: Projection is %dx%d cells, want %dx%d", f.name, p.Width, p.Height, f.w, f.h)
			}
			for r := 0; r <= f.h; r++ {
				for c := 0; c <= f.w; c++ {
					if got, want := p.Altitude(c, r), projRecipeAltitude(f.alt, f.w, f.h, c, r); got != want {
						t.Fatalf("%s: Altitude(%d,%d) = %d, want %d (recipe)", f.name, c, r, got, want)
					}
					if got, want := p.Vertex(c, r), projRecipeVertex(f.alt, f.w, f.h, c, r); got != want {
						t.Fatalf("%s: Vertex(%d,%d) = %d, want %d (recipe)", f.name, c, r, got, want)
					}
				}
			}
		}
	})

	t.Run("the far edge is clamped into the grid", func(t *testing.T) {
		// The two identities the clamp produces, over every fixture.
		for _, f := range projFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			// Column Width has no altitude of its own: it repeats column
			// Width-1, vertex for vertex. That identity is also why omitting
			// that column from a walk of the mesh can move no extreme — the
			// justification the canvas subtest leans on, asserted here rather
			// than assumed.
			for r := 0; r <= f.h; r++ {
				if got, want := p.Vertex(f.w, r), p.Vertex(f.w-1, r); got != want {
					t.Errorf("%s: Vertex(%d,%d) = %d, want %d (column %d clamps to %d)",
						f.name, f.w, r, got, want, f.w, f.w-1)
				}
			}
			// Row Height likewise repeats row Height-1's altitudes, so its
			// vertices sit exactly one cell below that row's — which is why the
			// canvas is never shorter than one cell, and why the greatest vertex
			// can live on a row the cell grid does not have.
			for c := 0; c <= f.w; c++ {
				if got, want := p.Vertex(c, f.h), p.Vertex(c, f.h-1)+projCell; got != want {
					t.Errorf("%s: Vertex(%d,%d) = %d, want %d (row %d clamps to %d, one cell lower)",
						f.name, c, f.h, got, want, f.h, f.h-1)
				}
			}
		}

		f := projFixtureFarEdge
		p := terrain.Project(f.alt, f.w, f.h)

		// The clamp is not the engine's own read. At (Width,0) a raw flat
		// offset into the unpadded grid lands on the next row's column 0, an
		// index that exists and holds a different altitude; ours is column
		// Width-1's. (TERR-EDGE-024..026: the engine defines no value here, and
		// clamping is our choice, so this asserts the choice, not the engine.)
		raw := int(int8(f.alt[f.w]))
		clamped := int(int8(f.alt[f.w-1]))
		if raw == clamped {
			t.Fatalf("fixture is blind: the raw far-edge read and the clamped one are both %d", raw)
		}
		if got := p.Altitude(f.w, 0); got != clamped {
			t.Errorf("Altitude(%d,0) = %d, want %d; %d is the next row's column 0, the read we do NOT make",
				f.w, got, clamped, raw)
		}
	})

	t.Run("altitudes are read signed", func(t *testing.T) {
		// 0x80 is -128, not 128. This is decidable only from the game's own
		// sign-extending load: no shipped height byte reaches 0x80, so no
		// corpus could tell the two apart.
		alt := []uint8{0x00, 0x7f, 0x80, 0xff}
		want := []int{0, 127, -128, -1}
		p := terrain.Project(alt, len(alt), 1)
		for c, w := range want {
			if got := p.Altitude(c, 0); got != w {
				t.Errorf("Altitude(%d,0) = %d, want %d (byte 0x%02x read signed)", c, got, w, alt[c])
			}
			if got, wantV := p.Vertex(c, 0), -w; got != wantV {
				t.Errorf("Vertex(%d,0) = %d, want %d (V = 0*32 - h, byte 0x%02x)", c, got, wantV, alt[c])
			}
		}
	})

	t.Run("a larger altitude moves the vertex up", func(t *testing.T) {
		// The altitude is subtracted: one unit of altitude is one destination
		// row, and a positive altitude moves the vertex toward row 0. On a grid
		// of one row, altitudes 0, +1 and -1 must land on 0, -1 and +1.
		p := terrain.Project([]uint8{0x00, 0x01, 0xff}, 3, 1)
		for c, want := range []int{0, -1, 1} {
			if got := p.Vertex(c, 0); got != want {
				t.Errorf("Vertex(%d,0) = %d, want %d", c, got, want)
			}
		}
		// Stated as an ordering too, so a flipped sign fails by name and not
		// only by arithmetic.
		if !(p.Vertex(1, 0) < p.Vertex(0, 0) && p.Vertex(0, 0) < p.Vertex(2, 0)) {
			t.Errorf("vertices at altitudes +1, 0, -1 are %d, %d, %d; want strictly increasing (a larger altitude is a smaller V)",
				p.Vertex(1, 0), p.Vertex(0, 0), p.Vertex(2, 0))
		}
	})

	t.Run("canvas bounds and origin", func(t *testing.T) {
		// MinV is the origin the render translates by and reports; MaxV is a
		// bottom edge, not a drawn row.
		//
		// The extremes are the mesh's, over all (W+1)*(H+1) vertices. The r = H
		// row can and does hold the greatest of them (the far-edge fixture), so
		// a mesh one row short is caught here. The c = W column cannot: it is
		// column W-1's own clamp, vertex for vertex, so no walk that includes or
		// omits it can move an extreme — that column is pinned in the mesh sweep
		// above instead, where it is observable.
		for _, f := range projFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			wantMin, wantMax := projRecipeCanvas(f.alt, f.w, f.h)
			if p.MinV != wantMin {
				t.Errorf("%s: MinV = %d, want %d (recipe)", f.name, p.MinV, wantMin)
			}
			if p.MaxV != wantMax {
				t.Errorf("%s: MaxV = %d, want %d (recipe)", f.name, p.MaxV, wantMax)
			}
			if got, want := p.CanvasHeight(), wantMax-wantMin; got != want {
				t.Errorf("%s: CanvasHeight = %d, want %d", f.name, got, want)
			}
			if p.MinV > p.MaxV {
				t.Errorf("%s: canvas [%d,%d) is empty or inverted", f.name, p.MinV, p.MaxV)
			}
			// Rows H-1 and H read the same altitudes and differ by one cell, so
			// no grid can produce a canvas shorter than a cell.
			if got := p.CanvasHeight(); got < projCell {
				t.Errorf("%s: CanvasHeight = %d, want at least %d", f.name, got, projCell)
			}
			// Every vertex is inside the canvas, and both bounds are attained.
			hitMin, hitMax := false, false
			for r := 0; r <= f.h; r++ {
				for c := 0; c <= f.w; c++ {
					v := p.Vertex(c, r)
					if v < p.MinV || v > p.MaxV {
						t.Fatalf("%s: Vertex(%d,%d) = %d is outside the canvas [%d,%d]",
							f.name, c, r, v, p.MinV, p.MaxV)
					}
					hitMin = hitMin || v == p.MinV
					hitMax = hitMax || v == p.MaxV
				}
			}
			if !hitMin || !hitMax {
				t.Errorf("%s: canvas [%d,%d) is wider than the mesh (min attained %v, max attained %v)",
					f.name, p.MinV, p.MaxV, hitMin, hitMax)
			}
		}
	})

	t.Run("a uniform grid keeps the flat canvas", func(t *testing.T) {
		// Every altitude equal shifts the whole mesh and stretches nothing: the
		// canvas is H*32 rows tall, as flat, and the origin is the negated
		// altitude.
		const w, h = 4, 3
		for _, a := range []uint8{0x00, 0x01, 0x7f, 0x80, 0xff} {
			p := terrain.Project(projUniform(w, h, a), w, h)
			if got, want := p.CanvasHeight(), h*projCell; got != want {
				t.Errorf("uniform 0x%02x: CanvasHeight = %d, want %d", a, got, want)
			}
			if got, want := p.MinV, -int(int8(a)); got != want {
				t.Errorf("uniform 0x%02x: MinV = %d, want %d", a, got, want)
			}
			if got, want := p.MaxV, h*projCell-int(int8(a)); got != want {
				t.Errorf("uniform 0x%02x: MaxV = %d, want %d", a, got, want)
			}
		}
	})

	t.Run("output dimensions", func(t *testing.T) {
		for _, f := range projFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			minV, maxV := projRecipeCanvas(f.alt, f.w, f.h)
			for _, scale := range []int{1, 2, 3, 7} {
				wantW := int64(f.w) * projCell * int64(scale)
				wantH := int64(maxV-minV) * int64(scale)
				gotW, gotH := p.OutputSize(scale)
				if gotW != wantW || gotH != wantH {
					t.Errorf("%s: OutputSize(%d) = %dx%d, want %dx%d",
						f.name, scale, gotW, gotH, wantW, wantH)
				}
			}
		}
	})

	t.Run("total on arguments a compositor rejects", func(t *testing.T) {
		// Project validates nothing — the rejections are the compositor's —
		// but it must still answer rather than panic, whatever it is handed, so
		// that a rejection can be reported instead of crashed on. Reaching the end
		// of this subtest is the assertion.
		cases := []struct {
			name string
			alt  []uint8
			w, h int
		}{
			{"nil grid", nil, 0, 0},
			{"nil grid, positive dimensions", nil, 4, 4},
			{"grid shorter than W*H", []uint8{0x01}, 4, 4},
			{"grid longer than W*H", projUniform(8, 8, 0x03), 2, 2},
			{"negative dimensions", []uint8{0x01}, -3, -3},
			{"zero width", []uint8{0x01}, 0, 1},
		}
		for _, c := range cases {
			p := terrain.Project(c.alt, c.w, c.h)
			for r := -2; r <= 6; r++ {
				for col := -2; col <= 6; col++ {
					_ = p.Vertex(col, r)
					_ = p.Altitude(col, r)
				}
			}
			_ = p.CanvasHeight()
			_, _ = p.OutputSize(1)
			if p.MinV > p.MaxV {
				t.Errorf("%s: canvas [%d,%d) is inverted", c.name, p.MinV, p.MaxV)
			}
		}
		var zero terrain.Projection
		if got := zero.Vertex(3, 3); got != 3*projCell {
			t.Errorf("zero Projection: Vertex(3,3) = %d, want %d (no altitudes to read)", got, 3*projCell)
		}
	})
}

// ---------------------------------------------------------------------------
// The raster contract, transcribed a second time.
//
// Everything below is the spec's "Projection and raster contract" written out
// again as a rasteriser, from the spec's own words. Its edges come from the step
// recipe of step_test.go (stepRecipeRow / stepRecipeForward / stepRecipeMirrored)
// and NOT from the package's EdgeForward / EdgeMirrored, so which walk each edge
// takes is decided here independently of the code under test. The pieces it does
// borrow from the package — Resolve, IsImpassable, DirtSubCell, LevelGrid,
// InterpSpan, ShadeRGBA — are pre-existing behaviour this story does not change,
// each pinned by its own suite.
// ---------------------------------------------------------------------------

// projTransparent is what an untouched pixel is: image.NewRGBA zero-fills, and
// every drawn pixel is opaque, so alpha alone separates covered from uncovered.
var projTransparent = color.RGBA{}

func projRecipeFlat(alt []uint8, w, h, col, row int) bool {
	a := projRecipeAltitude(alt, w, h, col, row)
	return a == projRecipeAltitude(alt, w, h, col+1, row) &&
		a == projRecipeAltitude(alt, w, h, col, row+1) &&
		a == projRecipeAltitude(alt, w, h, col+1, row+1)
}

func projRecipeEdge(yN, yF, i int, forward bool) int {
	if yN == yF {
		return yN // d = 0 leaves the edge constant.
	}
	d, dir := yF-yN, 1
	if d < 0 {
		d, dir = -d, -1
	}
	row := stepRecipeRow(d)
	if forward {
		return yN + dir*stepRecipeForward(row, d, i)
	}
	return yN + dir*stepRecipeMirrored(row, d, i)
}

func projRecipeSpan(alt []uint8, w, h, col, row, i int) (top, s int) {
	yTL := projRecipeVertex(alt, w, h, col, row)
	yTR := projRecipeVertex(alt, w, h, col+1, row)
	yBL := projRecipeVertex(alt, w, h, col, row+1)
	yBR := projRecipeVertex(alt, w, h, col+1, row+1)
	top = projRecipeEdge(yTL, yTR, i, yTL < yTR)
	bot := projRecipeEdge(yBL, yBR, i, yBL >= yBR)
	return top, bot - top
}

func projRecipeSrcRow(j, s int) int {
	return (j * ((projCell << 16) / s)) >> 16
}

// projRecipeLevels is a cell's four corner levels out of the level grid, with the
// far-edge +1 corner clamped in — the read CompositeLit documents.
func projRecipeLevels(levels []uint8, w, h, col, row int) (l00, l10, l01, l11 uint8) {
	at := func(c, r int) uint8 {
		if c > w-1 {
			c = w - 1
		}
		if r > h-1 {
			r = h - 1
		}
		return levels[r*w+c]
	}
	return at(col, row), at(col+1, row), at(col, row+1), at(col+1, row+1)
}

// projRecipePixel is one drawn pixel as the spec composes it: the source
// pixel at (sx,sy), the dirt overlay read at the SAME source coordinates,
// and the level interpolated across the cell's 32 columns and down spanY
// rows. A nil source is the placeholder, which keeps its cell's geometry and
// stays unlit.
func projRecipePixel(src, dirt *image.Paletted, sx, sy, lx, ly, spanY int, l00, l10, l01, l11 uint8, tint [3]uint8) color.RGBA {
	if src == nil {
		return terrain.PlaceholderColor
	}
	sb := src.Bounds()
	c := pixColor(src, sb.Min.X+sx, sb.Min.Y+sy)
	if dirt != nil {
		db := dirt.Bounds()
		c = overlayExpect(c, dirt.ColorIndexAt(db.Min.X+sx, db.Min.Y+sy))
	}
	return terrain.ShadeRGBA(c, tint, terrain.InterpSpan(l00, l10, l01, l11, lx, ly, projCell, spanY))
}

// projOracle is one map rasterised by the recipe: the image, the placeholder
// count, and the canvas the two bound.
type projOracle struct {
	img          *image.RGBA
	placeholders int
	minV, maxV   int
}

func projRecipeRender(t *testing.T, ts *terrain.Tileset, g terrain.Grid, alt []uint8, lt terrain.Light, scale int) projOracle {
	t.Helper()
	w, h := g.Width, g.Height
	minV, maxV := projRecipeCanvas(alt, w, h)
	out := projOracle{
		img:  image.NewRGBA(image.Rect(0, 0, w*projCell*scale, (maxV-minV)*scale)),
		minV: minV, maxV: maxV,
	}
	levels := terrain.LevelGrid(alt, w, h, lt)

	// put writes one NATIVE pixel, replicated scale x scale, translated so minV
	// is output row 0.
	put := func(nx, ny int, c color.RGBA) {
		if nx < 0 || nx >= w*projCell || ny < minV || ny >= maxV {
			t.Fatalf("the recipe drew native pixel (%d,%d), outside its own canvas [0,%d) x [%d,%d) (P-2)",
				nx, ny, w*projCell, minV, maxV)
		}
		ox, oy := nx*scale, (ny-minV)*scale
		for sy := 0; sy < scale; sy++ {
			for sx := 0; sx < scale; sx++ {
				out.img.SetRGBA(ox+sx, oy+sy, c)
			}
		}
	}

	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			word := g.Tiles[row*w+col]
			ref := terrain.Resolve(word)
			src := ts.Slot(ref.Slot).SubCell(ref.Sub)
			if src == nil {
				out.placeholders++
			}
			var dirt *image.Paletted
			if src != nil && terrain.IsImpassable(word) && !ref.Water {
				dirt = ts.Dirt.SubCell(terrain.DirtSubCell(col, row))
			}
			l00, l10, l01, l11 := projRecipeLevels(levels, w, h, col, row)
			tint := lt.SkyTint

			if projRecipeFlat(alt, w, h, col, row) {
				yTL := projRecipeVertex(alt, w, h, col, row)
				for y := 0; y < projCell; y++ {
					for x := 0; x < projCell; x++ {
						put(col*projCell+x, yTL+y,
							projRecipePixel(src, dirt, x, y, x, y, projCell, l00, l10, l01, l11, tint))
					}
				}
				continue
			}
			for i := 0; i < projCell; i++ {
				top, s := projRecipeSpan(alt, w, h, col, row, i)
				if s <= 0 {
					continue
				}
				for j := 0; j < s; j++ {
					put(col*projCell+i, top+j,
						projRecipePixel(src, dirt, i, projRecipeSrcRow(j, s), i, j, s, l00, l10, l01, l11, tint))
				}
			}
		}
	}
	return out
}

// projCompare asserts the package's render is the recipe's, pixel for pixel,
// with its dimensions, its placeholder count and its origin.
func projCompare(t *testing.T, what string, got *terrain.Render, want projOracle) {
	t.Helper()
	wb, gb := want.img.Bounds(), got.Image.Bounds()
	if gb != wb {
		t.Fatalf("%s: image bounds %v, want %v", what, gb, wb)
	}
	if got.OriginY != want.minV {
		t.Errorf("%s: OriginY = %d, want %d (the canvas's own MinV)", what, got.OriginY, want.minV)
	}
	if got.Placeholders != want.placeholders {
		t.Errorf("%s: Placeholders = %d, want %d", what, got.Placeholders, want.placeholders)
	}
	for y := gb.Min.Y; y < gb.Max.Y; y++ {
		for x := gb.Min.X; x < gb.Max.X; x++ {
			if g, w := got.Image.RGBAAt(x, y), want.img.RGBAAt(x, y); g != w {
				t.Fatalf("%s: pixel (%d,%d) = %v, want %v (recipe)", what, x, y, g, w)
			}
		}
	}
}

// --- synthetic fixtures ---

// pixelStrip builds a tile strip whose fill varies with BOTH axes inside a
// sub-cell, which solidStrip and halfStrip cannot express: a span's source row is
// only observable when the source rows differ from one another.
func pixelStrip(cells int, fill func(k, x, y int) byte) []byte {
	const size = 32
	h := size * cells
	idx := make([]byte, size*h)
	for k := 0; k < cells; k++ {
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				idx[(k*size+y)*size+x] = fill(k, x, y)
			}
		}
	}
	return buildBMP8(size, h, rampPalette(), idx)
}

// projLandIndex codes a land pixel by its sub-cell and its position inside it.
// Within one column x the 32 values are consecutive and so all distinct, and
// within one row y the 32 values are 8 apart and so all distinct: any single
// pixel read at the wrong source row, or from the wrong source column, decodes
// to a different colour.
func projLandIndex(k, x, y int) byte { return byte((k*16 + x*8 + y) & 0xff) }

// projSource is the tileset the projected fixtures draw from: a position-coded
// land strip, a solid water strip, and a dirt strip whose transparent key falls
// on EVEN source rows — so the impassable composite is itself sensitive to which
// source row a span samples.
func projSource() mapSource {
	return mapSource{
		terrain.TilePath(1, 0): pixelStrip(14, projLandIndex),
		terrain.TilePath(3, 0): solidStrip(8, func(k int) byte { return fillOf(waterBase, k) }),
		terrain.DirtPath: pixelStrip(4, func(k, x, y int) byte {
			if y%2 == 0 {
				return 0 // the transparent key: the terrain shows through
			}
			return byte(200 + k + x%8)
		}),
	}
}

// projLand, projWater, projImpassable and projAbsent are the four words a
// fixture cell can carry, one per compositor path.
func projLand(sub int) uint16       { return tileWord(0, 0, sub%14) }
func projWater(sub int) uint16      { return tileWord(8, 0, sub%8) }
func projImpassable(sub int) uint16 { return projLand(sub) | terrain.ImpassableBit }
func projAbsent() uint16            { return tileWord(1, 0, 0) } // slot 4, absent from projSource

// projTiles cycles a grid's cells through all four paths, so every fixture
// exercises the tile, water, dirt-composite and placeholder branches on whatever
// geometry its altitudes produce.
func projTiles(n int) []uint16 {
	out := make([]uint16, n)
	for i := range out {
		switch i % 4 {
		case 0:
			out[i] = projLand(i)
		case 1:
			out[i] = projWater(i)
		case 2:
			out[i] = projImpassable(i + 5)
		default:
			out[i] = projAbsent()
		}
	}
	return out
}

// projSeamAlt is a 2x2 grid whose shared vertex row between cell (0,0) and cell
// (0,1) carries exactly d steps: h(0,·) = d/2 and h(1,·) = d/2 - d, which stays
// inside a signed byte for every d in 1..255. Both cell rows therefore straddle
// the same step count, and the seam between them is whatever class the two walks
// fall into at d.
func projSeamAlt(d int) []uint8 {
	h0 := d / 2
	h1 := h0 - d
	return []uint8{uint8(int8(h0)), uint8(int8(h1)), uint8(int8(h0)), uint8(int8(h1))}
}

// projCollapseAlt is a 2x2 grid, signed
//
//	row 0:  -2   -2
//	row 1: -16   64
//
// whose cell (0,0) has a constant top edge at row 2 and a bottom edge walking
// from 48 down to -32, so its 32 spans run 45, 43, ... 2, 0, -3, ... -33: 18
// drawn columns, one column of EXACTLY zero height, and 13 inverted ones. Cell
// (1,0) is inverted at every column and so is drawn nowhere at all.
var projCollapseAlt = []uint8{0xfe, 0xfe, 0xf0, 0x40}

// projOverlapAlt is a 2x3 grid, signed rows [0,0], [0,0], [80,60]: the last cell
// row is lifted 80 rows and so is drawn ON TOP of the first, which row-major
// order painted long before. Its cells are sloped (their two altitudes differ).
var projOverlapAlt = []uint8{0, 0, 0, 0, 80, 60}

// projOverlapFlatAlt is the same overlap with a flat last row, so the rectangle
// blitter's ownership is exercised too.
var projOverlapFlatAlt = []uint8{0, 0, 0, 0, 80, 80}

// projOneCornerAlt is a 2x2 grid differing in ONE corner, signed rows [0,1],
// [0,0]. Cell (0,0) is sloped and visibly not a rectangle (its top edge steps
// from 0 to -1 halfway across). Cell (1,0) is the interesting one: its altitudes
// are 1,1,0,0 — unequal, so it is sloped — while its projected corners
// (-1,-1,32,32) are an axis-aligned rectangle of height 33. A selector that
// tested the projected rows instead of the altitudes would draw it 32 rows tall
// and unresampled.
var projOneCornerAlt = []uint8{0x00, 0x01, 0x00, 0x00}

// projFlatStepAlt is a 2x3 grid, signed rows [0,0], [0,0], [40,40]: cell row 0
// and cell row 2 are both flat, at DIFFERENT projected tops (0 and 24), so where
// a flat cell lands is observable rather than cancelled out by a uniform shift.
var projFlatStepAlt = []uint8{0, 0, 0, 0, 40, 40}

// projShadeAlt is a 5x5 grid whose altitude is constant across each row and
// steps 0, 0, 20, 40, 60 down them. Cell row 0 is therefore flat (four equal
// altitudes) and cell rows 1..3 are sloped with a constant 12-row span — short
// enough that the level ramp over S and the level ramp over 32 rows part company
// at almost every row. The level grid varies down the rows because the height
// field does, which is what makes the VERTICAL domain observable at all.
var projShadeAlt = func() []uint8 {
	rows := []uint8{0, 0, 20, 40, 60}
	alt := make([]uint8, 5*5)
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			alt[y*5+x] = rows[y]
		}
	}
	return alt
}()

// projRampAlt varies the altitude in both axes, so no two neighbouring cells
// share a step count and the level grid ramps as well.
func projRampAlt(w, h int) []uint8 {
	alt := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			alt[y*w+x] = uint8((x*13 + y*29) & 0xff)
		}
	}
	return alt
}

// projExtremeAlt chequers 0x7f (127) against 0x80 (-128): every horizontal and
// vertical neighbour differs by 255, the widest step two signed bytes can make.
func projExtremeAlt(w, h int) []uint8 {
	alt := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x+y)%2 == 0 {
				alt[y*w+x] = 0x7f
			} else {
				alt[y*w+x] = 0x80
			}
		}
	}
	return alt
}

// projMap is one whole fixture: a grid of altitudes, a grid of tile words, and
// the scales it is composed at.
type projMap struct {
	name   string
	w, h   int
	alt    []uint8
	tiles  []uint16
	scales []int
}

func (f projMap) grid() terrain.Grid {
	return terrain.Grid{Width: f.w, Height: f.h, Tiles: f.tiles}
}

// projMaps is every fixture the whole-contract sweep runs over.
func projMaps() []projMap {
	return []projMap{
		{"uniform zero", 3, 3, projUniform(3, 3, 0x00), projTiles(9), []int{1, 2, 3}},
		{"uniform 0x80", 3, 3, projUniform(3, 3, 0x80), projTiles(9), []int{1, 2}},
		{"one corner by one", 2, 2, projOneCornerAlt, projTiles(4), []int{1, 2, 3}},
		{"flat cells at two tops", 2, 3, projFlatStepAlt, projTiles(6), []int{1, 2}},
		{"relief in both axes", 4, 4, projRampAlt(4, 4), projTiles(16), []int{1, 2}},
		{"row-constant ramp, flat top and bottom rows", 5, 5, projShadeAlt, projTiles(25), []int{1, 2}},
		{"collapsing columns", 2, 2, projCollapseAlt, projTiles(4), []int{1, 2}},
		{"overlapping sloped quads", 2, 3, projOverlapAlt, projTiles(6), []int{1, 2}},
		{"overlapping flat quads", 2, 3, projOverlapFlatAlt, projTiles(6), []int{1}},
		{"seam, mirrored ahead (d=63)", 2, 2, projSeamAlt(63), projTiles(4), []int{1}},
		{"seam, mirrored behind (d=191)", 2, 2, projSeamAlt(191), projTiles(4), []int{1}},
		{"seam, interior disagreement (d=240)", 2, 2, projSeamAlt(240), projTiles(4), []int{1}},
		{"extreme deltas", 4, 4, projExtremeAlt(4, 4), projTiles(16), []int{1, 2}},
		{"every byte value", 16, 16, projSweepGrid(), projTiles(256), []int{1}},
	}
}

// projRender composes a fixture and fails the test on an unexpected error.
func projRender(t *testing.T, ts *terrain.Tileset, g terrain.Grid, alt []uint8, lt terrain.Light, scale int) *terrain.Render {
	t.Helper()
	r, err := terrain.CompositeProjectedLit(ts, g, alt, lt, scale)
	if err != nil {
		t.Fatalf("CompositeProjectedLit: %v", err)
	}
	return r
}

// projAt reads one NATIVE pixel of a render at scale 1, translated by the origin
// the render reports. Written in native coordinates because that is the language
// the contract's geometry is stated in.
func projAt(r *terrain.Render, nx, ny int) color.RGBA {
	return r.Image.RGBAAt(nx, ny-r.OriginY)
}

// TestProjectedCompositeMatchesRecipe is the whole-contract sweep: every fixture
// at every scale, compared with the recipe's own rasterisation pixel for pixel,
// plus its dimensions, its placeholder count and its reported origin. The
// focused tests that follow each name one clause of the contract; this one is
// the statement that nothing else drifted.
func TestProjectedCompositeMatchesRecipe(t *testing.T) {
	ts := terrain.LoadTileset(projSource())

	// The sweep is only worth anything if its fixtures reach every branch.
	sawFlat, sawSloped, sawDropped, sawPlaceholder := false, false, false, false

	for _, f := range projMaps() {
		g := f.grid()
		for row := 0; row < f.h; row++ {
			for col := 0; col < f.w; col++ {
				if projRecipeFlat(f.alt, f.w, f.h, col, row) {
					sawFlat = true
					continue
				}
				sawSloped = true
				for i := 0; i < projCell; i++ {
					if _, s := projRecipeSpan(f.alt, f.w, f.h, col, row, i); s <= 0 {
						sawDropped = true
					}
				}
			}
		}
		for _, w := range f.tiles {
			ref := terrain.Resolve(w)
			if ts.Slot(ref.Slot).SubCell(ref.Sub) == nil {
				sawPlaceholder = true
			}
		}

		for _, scale := range f.scales {
			got := projRender(t, ts, g, f.alt, terrain.DefaultDaytime, scale)
			want := projRecipeRender(t, ts, g, f.alt, terrain.DefaultDaytime, scale)
			projCompare(t, fmt.Sprintf("%s at scale %d", f.name, scale), got, want)
		}
	}

	if !sawFlat || !sawSloped || !sawDropped || !sawPlaceholder {
		t.Fatalf("the fixture set is not exercising the contract: flat=%v sloped=%v dropped column=%v placeholder=%v",
			sawFlat, sawSloped, sawDropped, sawPlaceholder)
	}
}

// projCellExpect is what the recipe says one cell draws: its source sub-cell, the
// dirt sub-cell it composites with (nil unless the cell is impassable non-water),
// its four corner levels and the sky tint.
type projCellExpect struct {
	src, dirt          *image.Paletted
	l00, l10, l01, l11 uint8
	tint               [3]uint8
}

func projExpectFor(ts *terrain.Tileset, g terrain.Grid, alt []uint8, lt terrain.Light, col, row int) projCellExpect {
	word := g.Tiles[row*g.Width+col]
	ref := terrain.Resolve(word)

	e := projCellExpect{tint: lt.SkyTint}
	e.src = ts.Slot(ref.Slot).SubCell(ref.Sub)
	if e.src != nil && terrain.IsImpassable(word) && !ref.Water {
		e.dirt = ts.Dirt.SubCell(terrain.DirtSubCell(col, row))
	}
	levels := terrain.LevelGrid(alt, g.Width, g.Height, lt)
	e.l00, e.l10, e.l01, e.l11 = projRecipeLevels(levels, g.Width, g.Height, col, row)
	return e
}

// spanPixel is row j of column i of a span s rows tall; rectPixel is pixel (x,y)
// of a flat cell's rectangle.
func (e projCellExpect) spanPixel(i, j, s int) color.RGBA {
	return projRecipePixel(e.src, e.dirt, i, projRecipeSrcRow(j, s), i, j, s, e.l00, e.l10, e.l01, e.l11, e.tint)
}

func (e projCellExpect) rectPixel(x, y int) color.RGBA {
	return projRecipePixel(e.src, e.dirt, x, y, x, y, projCell, e.l00, e.l10, e.l01, e.l11, e.tint)
}

// TestProjectedSelector covers SC-4 / AC-4: which blitter a corner set
// takes, and where its output lands.
func TestProjectedSelector(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime

	t.Run("four equal altitudes are a rectangle at the projected top", func(t *testing.T) {
		// projFlatStepAlt holds two flat cell rows at DIFFERENT tops — 0 and 24 —
		// so "at its projected top" is observable: a raster that ignored the
		// altitude would put the last row at native 64, off the canvas entirely.
		const w, h = 2, 3
		alt := projFlatStepAlt
		g := terrain.Grid{Width: w, Height: h, Tiles: []uint16{
			projLand(1), projLand(2),
			projLand(3), projLand(4),
			projLand(5), projLand(6),
		}}
		if !projRecipeFlat(alt, w, h, 1, 0) || !projRecipeFlat(alt, w, h, 1, 2) {
			t.Fatal("fixture: cell rows 0 and 2 must both be flat")
		}
		top := projRecipeVertex(alt, w, h, 1, 2)
		if top != 24 || top == 2*projCell {
			t.Fatalf("fixture: cell (1,2) projects to native top %d, want 24 (and not the un-projected %d)", top, 2*projCell)
		}

		r := projRender(t, ts, g, alt, lt, 1)
		e := projExpectFor(ts, g, alt, lt, 1, 2)
		for y := 0; y < projCell; y++ {
			for x := 0; x < projCell; x++ {
				if got, want := projAt(r, projCell+x, top+y), e.rectPixel(x, y); got != want {
					t.Fatalf("cell (1,2) pixel (%d,%d) at native row %d = %v, want %v", x, y, top+y, got, want)
				}
			}
		}
		// The rectangle begins exactly at its top: the row above still belongs to
		// the cell painted before it, which stands 24 rows higher.
		above := projExpectFor(ts, g, alt, lt, 1, 0)
		if got, want := projAt(r, projCell, top-1), above.rectPixel(0, top-1); got != want {
			t.Fatalf("native row %d = %v, want cell (1,0)'s own row %d (%v)", top-1, got, top-1, want)
		}
	})

	t.Run("one differing corner is per-column spans", func(t *testing.T) {
		const w, h = 2, 2
		alt := projOneCornerAlt
		g := terrain.Grid{Width: w, Height: h, Tiles: []uint16{
			projLand(1), projLand(2),
			projLand(3), projLand(4),
		}}
		r := projRender(t, ts, g, alt, lt, 1)
		e := projExpectFor(ts, g, alt, lt, 0, 0)

		// Cell (0,0)'s top edge steps from native 0 to -1 partway across, so the
		// canvas's own first row is drawn by SOME of its columns and by none of
		// the others. A rectangle would make that row all one thing.
		drawn, blank := 0, 0
		for i := 0; i < projCell; i++ {
			top, s := projRecipeSpan(alt, w, h, 0, 0, i)
			if top == -1 {
				drawn++
				if got, want := projAt(r, i, -1), e.spanPixel(i, 0, s); got != want {
					t.Fatalf("column %d at native row -1 = %v, want %v", i, got, want)
				}
				continue
			}
			blank++
			if got := projAt(r, i, -1); got != projTransparent {
				t.Fatalf("column %d starts at native row %d, so row -1 must be untouched; got %v", i, top, got)
			}
		}
		if drawn == 0 || blank == 0 {
			t.Fatalf("fixture is blind: %d columns reach native row -1 and %d do not", drawn, blank)
		}
	})

	t.Run("the selector reads altitudes, not the projected shape", func(t *testing.T) {
		// Cell (1,0) of the same fixture has altitudes 1,1,0,0 — unequal, so it is
		// sloped — while its projected corners (-1,-1,32,32) form a perfectly
		// axis-aligned rectangle, 33 rows tall. It must be drawn as 33 resampled
		// span rows, not as a 32-row copy: a selector testing the projected shape
		// instead of the altitudes gets this cell wrong and nothing else.
		const w, h = 2, 2
		alt := projOneCornerAlt
		g := terrain.Grid{Width: w, Height: h, Tiles: []uint16{
			projLand(1), projLand(2),
			projLand(3), projLand(4),
		}}
		if projRecipeFlat(alt, w, h, 1, 0) {
			t.Fatal("fixture: cell (1,0) must be sloped")
		}
		top, s := projRecipeSpan(alt, w, h, 1, 0, 0)
		if top != -1 || s != projCell+1 {
			t.Fatalf("fixture: cell (1,0) column 0 is %d rows at native %d, want %d at -1", s, top, projCell+1)
		}
		if projRecipeSrcRow(s-1, s) == s-1 {
			t.Fatal("fixture is blind: the span's last row is not resampled")
		}

		r := projRender(t, ts, g, alt, lt, 1)
		e := projExpectFor(ts, g, alt, lt, 1, 0)
		for i := 0; i < projCell; i++ {
			ti, si := projRecipeSpan(alt, w, h, 1, 0, i)
			for j := 0; j < si; j++ {
				if got, want := projAt(r, projCell+i, ti+j), e.spanPixel(i, j, si); got != want {
					t.Fatalf("cell (1,0) column %d row %d (native %d) = %v, want %v", i, j, ti+j, got, want)
				}
			}
		}
	})

	t.Run("the rectangle is the span rule at S = 32", func(t *testing.T) {
		if got := (projCell << 16) / projCell; got != 0x10000 {
			t.Fatalf("srcStep at S = 32 is 0x%x, want 0x10000", got)
		}
		for j := 0; j < projCell; j++ {
			if got := projRecipeSrcRow(j, projCell); got != j {
				t.Fatalf("srcRow(%d) at S = 32 = %d, want %d", j, got, j)
			}
		}
		for y := 0; y < projCell; y++ {
			for x := 0; x < projCell; x++ {
				span := terrain.InterpSpan(10, 40, 70, 95, x, y, projCell, projCell)
				row := terrain.InterpRow(10, 40, 70, 95, x, y, projCell)
				if span != row {
					t.Fatalf("level at (%d,%d): InterpSpan over 32 rows = %d, InterpRow = %d", x, y, span, row)
				}
			}
		}
	})
}

func TestProjectedSpanSampling(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime
	const w, h = 2, 2
	alt := projCollapseAlt

	for _, tc := range []struct {
		name string
		word uint16
	}{
		{"plain", projLand(3)},
		{"impassable, composited over dirt", projImpassable(3)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := terrain.Grid{Width: w, Height: h, Tiles: []uint16{
				tc.word, projLand(1),
				projLand(9), projWater(2),
			}}
			r := projRender(t, ts, g, alt, lt, 1)
			e := projExpectFor(ts, g, alt, lt, 0, 0)
			if tc.word&terrain.ImpassableBit != 0 && e.dirt == nil {
				t.Fatal("fixture: the impassable cell must carry a dirt sub-cell")
			}

			resampled, drawnColumns := 0, 0
			for i := 0; i < projCell; i++ {
				top, s := projRecipeSpan(alt, w, h, 0, 0, i)
				if s <= 0 {
					continue
				}
				drawnColumns++
				if s != projCell {
					resampled++
				}
				for j := 0; j < s; j++ {
					// Source column i, source row srcRow(j) — nothing else.
					if got, want := projAt(r, i, top+j), e.spanPixel(i, j, s); got != want {
						t.Fatalf("column %d row %d (native %d, S = %d, source row %d) = %v, want %v",
							i, j, top+j, s, projRecipeSrcRow(j, s), got, want)
					}
				}
			}
			if drawnColumns == 0 || resampled == 0 {
				t.Fatalf("fixture is blind: %d drawn columns, %d of them resampled", drawnColumns, resampled)
			}

			// Top inclusive, bottom exclusive. At column 0 the cell below starts
			// exactly where this one stops (its step count is not one of the two
			// walks' disagreements), so the row past the span is ITS first row and
			// not this cell's last.
			top, s := projRecipeSpan(alt, w, h, 0, 0, 0)
			below := projExpectFor(ts, g, alt, lt, 0, 1)
			topB, sB := projRecipeSpan(alt, w, h, 0, 1, 0)
			if topB != top+s {
				t.Fatalf("fixture: cell (0,1) starts at native %d, want %d (the row past cell (0,0))", topB, top+s)
			}
			last := e.spanPixel(0, s-1, s)
			first := below.spanPixel(0, 0, sB)
			if last == first {
				t.Fatal("fixture is blind: the two cells draw the same colour at the seam")
			}
			if got := projAt(r, 0, top+s-1); got != last {
				t.Fatalf("native row %d (the span's last row) = %v, want %v", top+s-1, got, last)
			}
			if got := projAt(r, 0, top+s); got != first {
				t.Fatalf("native row %d (one past the span) = %v, want the cell below's first row %v", top+s, got, first)
			}
		})
	}

	t.Run("the source row never leaves the sub-cell", func(t *testing.T) {
		for s := 1; s <= 1024; s++ {
			if got := projRecipeSrcRow(0, s); got != 0 {
				t.Fatalf("S = %d: srcRow(0) = %d, want 0", s, got)
			}
			prev := 0
			for j := 0; j < s; j++ {
				v := projRecipeSrcRow(j, s)
				if v < 0 || v > projCell-1 {
					t.Fatalf("S = %d: srcRow(%d) = %d, outside [0,%d]", s, j, v, projCell-1)
				}
				if v < prev {
					t.Fatalf("S = %d: srcRow(%d) = %d went backwards from %d", s, j, v, prev)
				}
				prev = v
			}
		}
		// And over every span every fixture actually produces.
		for _, f := range projMaps() {
			for row := 0; row < f.h; row++ {
				for col := 0; col < f.w; col++ {
					for i := 0; i < projCell; i++ {
						_, s := projRecipeSpan(f.alt, f.w, f.h, col, row, i)
						if s <= 0 {
							continue
						}
						if v := projRecipeSrcRow(s-1, s); v > projCell-1 {
							t.Fatalf("%s: cell (%d,%d) column %d has S = %d, srcRow(S-1) = %d", f.name, col, row, i, s, v)
						}
					}
				}
			}
		}
	})
}

// projCoveredRows is the set of NATIVE rows the contract draws at
// destination column i of cell column col: the union over every cell row of
// what that cell covers there.
func projCoveredRows(alt []uint8, w, h, col, i int) map[int]bool {
	cover := map[int]bool{}
	for row := 0; row < h; row++ {
		if projRecipeFlat(alt, w, h, col, row) {
			yTL := projRecipeVertex(alt, w, h, col, row)
			for y := 0; y < projCell; y++ {
				cover[yTL+y] = true
			}
			continue
		}
		top, s := projRecipeSpan(alt, w, h, col, row, i)
		for j := 0; j < s; j++ {
			cover[top+j] = true
		}
	}
	return cover
}

func projAssertCoverage(t *testing.T, what string, r *terrain.Render, alt []uint8, w, h, col, i int) {
	t.Helper()
	minV, maxV := projRecipeCanvas(alt, w, h)
	cover := projCoveredRows(alt, w, h, col, i)
	for ny := minV; ny < maxV; ny++ {
		got := projAt(r, col*projCell+i, ny)
		switch {
		case cover[ny] && got == projTransparent:
			t.Fatalf("%s: cell column %d, column %d, native row %d is covered but was left transparent", what, col, i, ny)
		case !cover[ny] && got != projTransparent:
			t.Fatalf("%s: cell column %d, column %d, native row %d is covered by no drawn column but holds %v", what, col, i, ny, got)
		}
	}
}

func TestProjectedCollapsedColumn(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime
	const w, h = 2, 2
	alt := projCollapseAlt

	// What the fixture is for, asserted rather than assumed: cell (0,0) must hold
	// drawn columns, a column of EXACTLY zero height, and inverted ones — the
	// zero-height column being the one that separates "S <= 0" from "S < 0", and
	// the one on which a span blitter that did not drop it would divide by zero.
	drawn, zero, inverted := 0, 0, 0
	for i := 0; i < projCell; i++ {
		switch _, s := projRecipeSpan(alt, w, h, 0, 0, i); {
		case s > 0:
			drawn++
		case s == 0:
			zero++
		default:
			inverted++
		}
	}
	if drawn == 0 || zero == 0 || inverted == 0 {
		t.Fatalf("fixture is blind: %d drawn columns, %d of zero height, %d inverted", drawn, zero, inverted)
	}

	for _, tc := range []struct {
		name string
		word uint16
	}{
		{"a tile cell", projLand(3)},
		{"a placeholder cell", projAbsent()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := terrain.Grid{Width: w, Height: h, Tiles: []uint16{
				tc.word, tc.word,
				projLand(9), projLand(5),
			}}
			r := projRender(t, ts, g, alt, lt, 1)
			e := projExpectFor(ts, g, alt, lt, 0, 0)

			// Cell (0,0): the collapsed columns contribute nothing at all, the
			// drawn ones contribute exactly their span, and no pixel that no
			// column covers is painted.
			for i := 0; i < projCell; i++ {
				projAssertCoverage(t, tc.name, r, alt, w, h, 0, i)

				top, s := projRecipeSpan(alt, w, h, 0, 0, i)
				for j := 0; j < s; j++ {
					if got, want := projAt(r, i, top+j), e.spanPixel(i, j, s); got != want {
						t.Fatalf("column %d row %d (native %d) = %v, want %v", i, j, top+j, got, want)
					}
				}
			}

			// Named neighbours: the last drawn column, the column of exactly zero
			// height and the first inverted one sit side by side, and only the
			// first of them marks the image.
			lastDrawn, firstZero, firstInverted := -1, -1, -1
			for i := 0; i < projCell; i++ {
				_, s := projRecipeSpan(alt, w, h, 0, 0, i)
				if s > 0 {
					lastDrawn = i
				}
				if s == 0 && firstZero < 0 {
					firstZero = i
				}
				if s < 0 && firstInverted < 0 {
					firstInverted = i
				}
			}
			if lastDrawn+1 != firstZero || firstZero+1 != firstInverted {
				t.Fatalf("fixture: expected adjacent columns, got last drawn %d, zero %d, inverted %d", lastDrawn, firstZero, firstInverted)
			}
			topD, sD := projRecipeSpan(alt, w, h, 0, 0, lastDrawn)
			if got, want := projAt(r, lastDrawn, topD), e.spanPixel(lastDrawn, 0, sD); got != want {
				t.Fatalf("column %d (S = %d) top row = %v, want %v", lastDrawn, sD, got, want)
			}

			// How much of the image each of the three columns marks. The cell
			// below draws in all of them; only the drawn one carries anything of
			// cell (0,0) as well.
			minV, maxV := projRecipeCanvas(alt, w, h)
			painted := func(i int) int {
				n := 0
				for ny := minV; ny < maxV; ny++ {
					if projAt(r, i, ny) != projTransparent {
						n++
					}
				}
				return n
			}
			for _, i := range []int{firstZero, firstInverted} {
				_, below := projRecipeSpan(alt, w, h, 0, 1, i)
				if got := painted(i); got != below {
					t.Fatalf("column %d has a non-positive span, so it must carry the cell below's %d rows and nothing else; %d rows are painted",
						i, below, got)
				}
			}
			if got, other := painted(lastDrawn), painted(firstZero); got <= other {
				t.Fatalf("the drawn column %d paints %d rows and the dropped column %d paints %d; the dropped column's neighbour must be unaffected",
					lastDrawn, got, firstZero, other)
			}

			// Cell (1,0) inverts at every column, so a whole cell is drawn
			// nowhere: at cell column 1 only the cell below leaves a mark.
			for i := 0; i < projCell; i++ {
				if _, s := projRecipeSpan(alt, w, h, 1, 0, i); s > 0 {
					t.Fatalf("fixture: cell (1,0) column %d has span %d, want it inverted everywhere", i, s)
				}
				projAssertCoverage(t, tc.name, r, alt, w, h, 1, i)
			}

			// The placeholder is still counted for both cells, geometry or no
			// geometry — including the one that drew nothing at all.
			wantPlaceholders := 0
			if tc.word == projAbsent() {
				wantPlaceholders = 2
			}
			if r.Placeholders != wantPlaceholders {
				t.Fatalf("Placeholders = %d, want %d", r.Placeholders, wantPlaceholders)
			}
		})
	}
}

// TestProjectedOwnership covers SC-8 / AC-8: where two cells' quads overlap
// the later cell in row-major order owns every shared pixel, and a pixel no
// drawn column covers stays transparent.
func TestProjectedOwnership(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime

	for _, tc := range []struct {
		name string
		alt  []uint8
	}{
		{"flat quads", projOverlapFlatAlt},
		{"sloped quads", projOverlapAlt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const w, h = 2, 3
			alt := tc.alt
			g := terrain.Grid{Width: w, Height: h, Tiles: []uint16{
				projLand(1), projLand(2),
				projLand(3), projLand(4),
				projLand(5), projLand(6),
			}}
			r := projRender(t, ts, g, alt, lt, 1)
			expect := make([]projCellExpect, w*h)
			for row := 0; row < h; row++ {
				for col := 0; col < w; col++ {
					expect[row*w+col] = projExpectFor(ts, g, alt, lt, col, row)
				}
			}

			shared := 0
			for col := 0; col < w; col++ {
				for i := 0; i < projCell; i++ {
					// Walk the cells in painting order, keeping the LAST colour
					// claimed for each native row: that is the ownership rule.
					owner := map[int]color.RGBA{}
					claims := map[int]int{}
					for row := 0; row < h; row++ {
						e := expect[row*w+col]
						if projRecipeFlat(alt, w, h, col, row) {
							yTL := projRecipeVertex(alt, w, h, col, row)
							for y := 0; y < projCell; y++ {
								owner[yTL+y] = e.rectPixel(i, y)
								claims[yTL+y]++
							}
							continue
						}
						top, s := projRecipeSpan(alt, w, h, col, row, i)
						for j := 0; j < s; j++ {
							owner[top+j] = e.spanPixel(i, j, s)
							claims[top+j]++
						}
					}
					for ny, want := range owner {
						if claims[ny] > 1 {
							shared++
						}
						if got := projAt(r, col*projCell+i, ny); got != want {
							t.Fatalf("cell column %d, column %d, native row %d (claimed by %d cells) = %v, want the last claimant's %v",
								col, i, ny, claims[ny], got, want)
						}
					}
					projAssertCoverage(t, tc.name, r, alt, w, h, col, i)
				}
			}
			if shared == 0 {
				t.Fatal("fixture is blind: no pixel is covered by two cells")
			}
		})
	}
}

// TestProjectedSeam covers SC-3: a cell's bottom edge and the cell below's
// top edge are one vertex pair walked once forward and once mirrored, so at
// the step counts where the two walks disagree the seam is off by a row —
// repainted where the mirrored walk runs ahead, undrawn where it runs
// behind. Both are drawn as they fall.
//
// The classification is the contract's, twice over: recomputed here from the step
// recipe, and checked against stepWalkDisagreement, the literal table step_test.go
// pins against that same recipe.
func TestProjectedSeam(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime
	const w, h = 2, 2

	for _, d := range []int{20, 63, 127, 191, 240, 255} {
		t.Run(fmt.Sprintf("d=%d", d), func(t *testing.T) {
			alt := projSeamAlt(d)
			g := terrain.Grid{Width: w, Height: h, Tiles: []uint16{
				projLand(2), projLand(1),
				projLand(9), projLand(5),
			}}
			// The shared line is vertices (0,1) and (1,1); its step count must be
			// the d this subtest is about.
			a := projRecipeVertex(alt, w, h, 0, 1)
			b := projRecipeVertex(alt, w, h, 1, 1)
			if b-a != d {
				t.Fatalf("fixture: the shared edge runs %d steps, want %d", b-a, d)
			}

			wantDelta := map[int]int{}
			if cls, ok := stepWalkDisagreement[d]; ok {
				for _, c := range cls.cols {
					wantDelta[c] = cls.delta
				}
			}

			r := projRender(t, ts, g, alt, lt, 1)
			upper := projExpectFor(ts, g, alt, lt, 0, 0)
			lower := projExpectFor(ts, g, alt, lt, 0, 1)
			stepRow := stepRecipeRow(d)

			overdraw, gaps, exact := 0, 0, 0
			for i := 0; i < projCell; i++ {
				top0, s0 := projRecipeSpan(alt, w, h, 0, 0, i)
				top1, s1 := projRecipeSpan(alt, w, h, 0, 1, i)
				if s0 <= 0 || s1 <= 0 {
					t.Fatalf("fixture: column %d has spans %d and %d, want both drawn", i, s0, s1)
				}
				bot0 := top0 + s0

				// The seam offset IS the two walks' difference at (d,i).
				delta := stepRecipeMirrored(stepRow, d, i) - stepRecipeForward(stepRow, d, i)
				if delta != wantDelta[i] {
					t.Fatalf("column %d: the recipe's walks differ by %d, but the pinned classification says %d", i, delta, wantDelta[i])
				}
				if bot0-top1 != delta {
					t.Fatalf("column %d: the upper cell ends at %d and the lower starts at %d, a seam offset of %d, want %d",
						i, bot0, top1, bot0-top1, delta)
				}

				switch delta {
				case +1:
					// Overdraw: the upper cell's last row is the lower cell's
					// first, and the lower one — painted after it — owns it.
					overdraw++
					was := upper.spanPixel(i, s0-1, s0)
					now := lower.spanPixel(i, 0, s1)
					if was == now {
						t.Fatalf("fixture is blind at column %d: both cells draw %v at the shared row", i, was)
					}
					if got := projAt(r, i, top1); got != now {
						t.Fatalf("column %d: the shared row %d = %v, want the LOWER cell's %v (it is painted second)", i, top1, got, now)
					}
					if got := projAt(r, i, top1-1); got != upper.spanPixel(i, s0-2, s0) {
						t.Fatalf("column %d: native row %d = %v, want the upper cell's own row above the seam", i, top1-1, got)
					}
				case -1:
					// A hole: the row between the two cells is drawn by neither,
					// and shows as a transparent pixel with terrain above and
					// below it.
					gaps++
					if got := projAt(r, i, bot0); got != projTransparent {
						t.Fatalf("column %d: native row %d lies between the two cells and is drawn by neither, so it must be transparent; got %v",
							i, bot0, got)
					}
					if got := projAt(r, i, bot0-1); got != upper.spanPixel(i, s0-1, s0) {
						t.Fatalf("column %d: native row %d = %v, want the upper cell's last row", i, bot0-1, got)
					}
					if got := projAt(r, i, top1); got != lower.spanPixel(i, 0, s1) {
						t.Fatalf("column %d: native row %d = %v, want the lower cell's first row", i, top1, got)
					}
				default:
					exact++
					if got := projAt(r, i, bot0); got != lower.spanPixel(i, 0, s1) {
						t.Fatalf("column %d: the two cells tile exactly, so native row %d must be the lower cell's first row; got %v", i, bot0, got)
					}
				}

				// Whatever the class, no OTHER row is left out: the covered set
				// and the painted set agree at this column.
				projAssertCoverage(t, fmt.Sprintf("d=%d", d), r, alt, w, h, 0, i)
			}

			// The class this step count belongs to, counted at composite level.
			wantOver, wantGap := 0, 0
			for _, delta := range wantDelta {
				if delta > 0 {
					wantOver++
				} else {
					wantGap++
				}
			}
			if overdraw != wantOver || gaps != wantGap || exact != projCell-wantOver-wantGap {
				t.Fatalf("d=%d: %d repainted seams, %d holes, %d exact; want %d, %d, %d",
					d, overdraw, gaps, exact, wantOver, wantGap, projCell-wantOver-wantGap)
			}
		})
	}
}

// TestProjectedShadingDomain covers SC-6 / AC-6: the level ramp runs over
// the drawn span on a sloped cell and over 32 rows on a flat one.
func TestProjectedShadingDomain(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime
	const w, h = 5, 5
	alt := projShadeAlt
	tiles := make([]uint16, w*h)
	for i := range tiles {
		tiles[i] = projLand(i)
	}
	g := terrain.Grid{Width: w, Height: h, Tiles: tiles}
	r := projRender(t, ts, g, alt, lt, 1)

	t.Run("a sloped cell ramps over the span", func(t *testing.T) {
		// Cell (2,1) is sloped with a 12-row span, and its top and bottom corner
		// levels differ — without both, the vertical domain is unobservable.
		const col, row = 2, 1
		if projRecipeFlat(alt, w, h, col, row) {
			t.Fatalf("fixture: cell (%d,%d) must be sloped", col, row)
		}
		e := projExpectFor(ts, g, alt, lt, col, row)
		if e.l00 == e.l01 {
			t.Fatalf("fixture is blind: cell (%d,%d)'s top and bottom levels are both %d", col, row, e.l00)
		}

		differed := 0
		for i := 0; i < projCell; i++ {
			top, s := projRecipeSpan(alt, w, h, col, row, i)
			if s <= 0 {
				continue
			}
			if s == projCell {
				t.Fatalf("fixture is blind: column %d spans exactly %d rows", i, projCell)
			}
			for j := 0; j < s; j++ {
				overSpan := terrain.InterpSpan(e.l00, e.l10, e.l01, e.l11, i, j, projCell, s)
				over32 := terrain.InterpSpan(e.l00, e.l10, e.l01, e.l11, i, j, projCell, projCell)
				if overSpan != over32 {
					differed++
				}
				want := projRecipePixel(e.src, e.dirt, i, projRecipeSrcRow(j, s), i, j, s, e.l00, e.l10, e.l01, e.l11, e.tint)
				if got := projAt(r, col*projCell+i, top+j); got != want {
					t.Fatalf("column %d row %d (S = %d): pixel = %v, want %v (level %d over the span, not %d over 32 rows)",
						i, j, s, got, want, overSpan, over32)
				}
			}
		}
		if differed == 0 {
			t.Fatal("fixture is blind: no drawn pixel's level distinguishes the span from 32 rows")
		}
	})

	t.Run("a flat cell ramps over 32 rows", func(t *testing.T) {
		// Cell row 0 is flat (four equal altitudes) and still sits on a level
		// gradient, its top and bottom vertices carrying different levels.
		const col, row = 2, 0
		if !projRecipeFlat(alt, w, h, col, row) {
			t.Fatalf("fixture: cell (%d,%d) must be flat", col, row)
		}
		e := projExpectFor(ts, g, alt, lt, col, row)
		if e.l00 == e.l01 {
			t.Fatalf("fixture is blind: cell (%d,%d)'s top and bottom levels are both %d", col, row, e.l00)
		}
		top := projRecipeVertex(alt, w, h, col, row)
		sb := e.src.Bounds()
		for y := 0; y < projCell; y++ {
			for x := 0; x < projCell; x++ {
				want := terrain.ShadeRGBA(
					pixColor(e.src, sb.Min.X+x, sb.Min.Y+y),
					e.tint,
					terrain.InterpRow(e.l00, e.l10, e.l01, e.l11, x, y, projCell))
				if got := projAt(r, col*projCell+x, top+y); got != want {
					t.Fatalf("flat cell (%d,%d) pixel (%d,%d) = %v, want %v (InterpRow over 32 rows)", col, row, x, y, got, want)
				}
			}
		}
	})
}

func TestProjectedPixelsInBounds(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime

	for _, f := range projMaps() {
		minV, maxV := projRecipeCanvas(f.alt, f.w, f.h)

		for row := 0; row < f.h; row++ {
			for col := 0; col < f.w; col++ {
				if projRecipeFlat(f.alt, f.w, f.h, col, row) {
					yTL := projRecipeVertex(f.alt, f.w, f.h, col, row)
					if yTL < minV || yTL+projCell-1 >= maxV {
						t.Fatalf("%s: flat cell (%d,%d) covers native [%d,%d), outside the canvas [%d,%d)",
							f.name, col, row, yTL, yTL+projCell, minV, maxV)
					}
					continue
				}
				for i := 0; i < projCell; i++ {
					top, s := projRecipeSpan(f.alt, f.w, f.h, col, row, i)
					if s <= 0 {
						continue
					}
					if top < minV || top+s-1 >= maxV {
						t.Fatalf("%s: cell (%d,%d) column %d covers native [%d,%d), outside the canvas [%d,%d)",
							f.name, col, row, i, top, top+s, minV, maxV)
					}
				}
			}
		}

		for _, scale := range f.scales {
			r := projRender(t, ts, f.grid(), f.alt, lt, scale)
			wantW, wantH := f.w*projCell*scale, (maxV-minV)*scale
			b := r.Image.Bounds()
			if b != image.Rect(0, 0, wantW, wantH) {
				t.Fatalf("%s at scale %d: bounds %v, want %dx%d", f.name, scale, b, wantW, wantH)
			}
			// Replication: a pixel and the scale x scale block it heads are one
			// colour, painted or transparent.
			for y := 0; y < wantH; y++ {
				for x := 0; x < wantW; x++ {
					if got, want := r.Image.RGBAAt(x, y), r.Image.RGBAAt(x-x%scale, y-y%scale); got != want {
						t.Fatalf("%s at scale %d: pixel (%d,%d) = %v breaks its %dx%d block, whose head is %v",
							f.name, scale, x, y, got, scale, scale, want)
					}
				}
			}
		}
	}
}

func TestProjectedExtremeDeltas(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("CompositeProjectedLit panicked: %v", p)
		}
	}()

	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime

	fixtures := []projMap{
		{"chequered 0x7f against 0x80", 4, 4, projExtremeAlt(4, 4), projTiles(16), []int{1, 2}},
		{"-128 beside 127, both axes", 2, 2, []uint8{0x80, 0x7f, 0x7f, 0x80}, projTiles(4), []int{1, 3}},
		{"every byte value", 16, 16, projSweepGrid(), projTiles(256), []int{1}},
	}

	widest := 0
	for _, f := range fixtures {
		// The step count of every cell edge in the fixture, which is where a
		// "128 or more" delta actually shows up.
		for row := 0; row <= f.h; row++ {
			for col := 0; col < f.w; col++ {
				d := projRecipeVertex(f.alt, f.w, f.h, col+1, row) - projRecipeVertex(f.alt, f.w, f.h, col, row)
				if d < 0 {
					d = -d
				}
				if d > widest {
					widest = d
				}
			}
		}
		for _, scale := range f.scales {
			got := projRender(t, ts, f.grid(), f.alt, lt, scale)
			want := projRecipeRender(t, ts, f.grid(), f.alt, lt, scale)
			projCompare(t, fmt.Sprintf("%s at scale %d", f.name, scale), got, want)
		}
	}
	if widest != 255 {
		t.Fatalf("the widest edge in these fixtures is %d steps, want the full 255 (127 beside -128)", widest)
	}
}

// projEntry is one compositor behind one signature, so a single argument set can
// be put to all four and their answers compared. heights says whether the entry
// point takes a height grid and validates its length — Composite alone does not,
// and is therefore the one entry point for which a bad height grid is not an
// error at all.
type projEntry struct {
	name    string
	heights bool
	compose func(ts *terrain.Tileset, g terrain.Grid, alt []uint8, lt terrain.Light, scale int) (*terrain.Render, error)
}

// projProjectedEntries are the two entry points of this story's geometry:
// the unshaded sibling and the shaded one. What SC-12 asserts, it asserts of
// both — they are meant to share one validation, and a sweep over one of
// them alone could not tell that from two copies that happen to agree today.
func projProjectedEntries() []projEntry {
	return []projEntry{
		{name: "CompositeProjected", heights: true,
			compose: func(ts *terrain.Tileset, g terrain.Grid, alt []uint8, _ terrain.Light, scale int) (*terrain.Render, error) {
				return terrain.CompositeProjected(ts, g, alt, scale)
			}},
		{name: "CompositeProjectedLit", heights: true,
			compose: func(ts *terrain.Tileset, g terrain.Grid, alt []uint8, lt terrain.Light, scale int) (*terrain.Render, error) {
				return terrain.CompositeProjectedLit(ts, g, alt, lt, scale)
			}},
	}
}

// projAllEntries adds the two flat compositors, whose rejections the
// projected pair must match word for word: one rejection set, four ways in.
func projAllEntries() []projEntry {
	return append(projProjectedEntries(),
		projEntry{name: "CompositeLit", heights: true,
			compose: func(ts *terrain.Tileset, g terrain.Grid, alt []uint8, lt terrain.Light, scale int) (*terrain.Render, error) {
				return terrain.CompositeLit(ts, g, alt, lt, scale)
			}},
		projEntry{name: "Composite", heights: false,
			compose: func(ts *terrain.Tileset, g terrain.Grid, _ []uint8, _ terrain.Light, scale int) (*terrain.Render, error) {
				return terrain.Composite(ts, g, scale)
			}},
	)
}

// projSameRender asserts two renders are the same picture — bounds, placeholder
// count and every pixel — and that each reports the origin its geometry gives it:
// the projection's own MinV, and 0 for a flat canvas that begins at native row 0.
func projSameRender(t *testing.T, what string, proj, flat *terrain.Render, wantOrigin int) {
	t.Helper()
	if flat.OriginY != 0 {
		t.Errorf("%s: the flat render reports OriginY %d, want 0", what, flat.OriginY)
	}
	if proj.OriginY != wantOrigin {
		t.Errorf("%s: the projected render reports OriginY %d, want %d", what, proj.OriginY, wantOrigin)
	}
	pb, fb := proj.Image.Bounds(), flat.Image.Bounds()
	if pb != fb {
		t.Fatalf("%s: projected bounds %v, flat %v", what, pb, fb)
	}
	if proj.Placeholders != flat.Placeholders {
		t.Errorf("%s: projected placeholders %d, flat %d", what, proj.Placeholders, flat.Placeholders)
	}
	if proj.Placeholders == 0 {
		t.Fatalf("%s: neither render drew a placeholder, so the counts agree on nothing", what)
	}
	for y := pb.Min.Y; y < pb.Max.Y; y++ {
		for x := pb.Min.X; x < pb.Max.X; x++ {
			if p, f := proj.Image.RGBAAt(x, y), flat.Image.RGBAAt(x, y); p != f {
				t.Fatalf("%s: pixel (%d,%d) projected %v, flat %v", what, x, y, p, f)
			}
		}
	}
}

// projDiffers reports whether two renders differ at all: in size, in placeholder
// count, or in one pixel.
func projDiffers(a, b *terrain.Render) bool {
	ab, bb := a.Image.Bounds(), b.Image.Bounds()
	if ab != bb || a.Placeholders != b.Placeholders {
		return true
	}
	for y := ab.Min.Y; y < ab.Max.Y; y++ {
		for x := ab.Min.X; x < ab.Max.X; x++ {
			if a.Image.RGBAAt(x, y) != b.Image.RGBAAt(x, y) {
				return true
			}
		}
	}
	return false
}

func TestProjectedEqualsFlat(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime

	// 3x3 so the perturbed control has an interior cell to move, and projTiles
	// puts land, water, an impassable dirt composite and an absent slot in it.
	const w, h = 3, 3
	tiles := projTiles(w * h)
	g := terrain.Grid{Width: w, Height: h, Tiles: tiles}

	// 0x00, a positive, both signed extremes and -1: the values AC-1 names, plus
	// the two bytes that separate a signed altitude read from an unsigned one.
	for _, a := range []uint8{0x00, 0x01, 0x7f, 0x80, 0xff} {
		for _, scale := range []int{1, 2, 3} {
			t.Run(fmt.Sprintf("altitude 0x%02x at scale %d", a, scale), func(t *testing.T) {
				alt := projUniform(w, h, a)
				wantOrigin := -int(int8(a))

				// The canvas the contract gives this grid, computed from the
				// recipe rather than assumed: H*32 rows starting at -h.
				minV, maxV := projRecipeCanvas(alt, w, h)
				if minV != wantOrigin || maxV-minV != h*projCell {
					t.Fatalf("the recipe puts this uniform grid's canvas at [%d,%d), %d rows; the equality is stated over [%d, %d), %d rows",
						minV, maxV, maxV-minV, wantOrigin, wantOrigin+h*projCell, h*projCell)
				}

				flat, err := terrain.Composite(ts, g, scale)
				if err != nil {
					t.Fatalf("Composite: %v", err)
				}
				proj, err := terrain.CompositeProjected(ts, g, alt, scale)
				if err != nil {
					t.Fatalf("CompositeProjected: %v", err)
				}
				projSameRender(t, "unshaded", proj, flat, wantOrigin)

				flatLit, err := terrain.CompositeLit(ts, g, alt, lt, scale)
				if err != nil {
					t.Fatalf("CompositeLit: %v", err)
				}
				projLit, err := terrain.CompositeProjectedLit(ts, g, alt, lt, scale)
				if err != nil {
					t.Fatalf("CompositeProjectedLit: %v", err)
				}
				projSameRender(t, "shaded", projLit, flatLit, wantOrigin)

				want := image.Rect(0, 0, w*projCell*scale, h*projCell*scale)
				if got := proj.Image.Bounds(); got != want {
					t.Errorf("unshaded bounds %v, want %v (W*32*scale by (maxV-minV)*scale)", got, want)
				}

				// The control: one interior altitude moved by one step, at scale
				// 1 (the perturbation is a native-row difference; replicating it
				// scale x scale adds nothing to the question).
				if scale != 1 {
					return
				}
				bent := projUniform(w, h, a)
				bent[1*w+1] = a ^ 0x01 // exactly +/-1 signed, at whichever end
				bMinV, bMaxV := projRecipeCanvas(bent, w, h)
				if bMinV != minV || bMaxV != maxV {
					t.Fatalf("the perturbed grid moved the canvas to [%d,%d) from [%d,%d); the control is meant to differ in PIXELS alone",
						bMinV, bMaxV, minV, maxV)
				}
				bentProj, err := terrain.CompositeProjected(ts, g, bent, scale)
				if err != nil {
					t.Fatalf("CompositeProjected on the perturbed grid: %v", err)
				}
				if !projDiffers(bentProj, flat) {
					t.Fatalf("moving altitude (1,1) from 0x%02x to 0x%02x changed no pixel: the equality above holds for a reason other than the projection",
						a, bent[1*w+1])
				}
				if bentProj.OriginY != wantOrigin {
					t.Errorf("the perturbed grid reports OriginY %d, want %d", bentProj.OriginY, wantOrigin)
				}
			})
		}
	}
}

// TestProjectedUnshadedMatchesRecipe sweeps the UNSHADED entry point over every
// fixture of the whole-contract sweep, against the same transcription of the
// raster contract (projRecipeRender) composed at the IDENTITY LIGHT: every vertex
// level 64, and ShadeRGBA at 64 with a zero tint returns its argument, so the
// recipe at that light IS the unshaded composition, pixel for pixel. Both
// premises are asserted below rather than assumed, and neither is new — it is the
// pin CompositeLit already carries against Composite in lit_test.go.
//
// This is the second state of the shared core, swept over the same fixtures as
// the first (TestProjectedCompositeMatchesRecipe). Between them, a level grid
// consulted on the wrong path is wrong on one of the two whichever way round it
// is put — which is why the core's one parameter is worth having only if both of
// its values are pinned here.
func TestProjectedUnshadedMatchesRecipe(t *testing.T) {
	ts := terrain.LoadTileset(projSource())

	// Premise 1: the identity light levels every vertex at 64 whatever the
	// heights, so it removes the relief without removing the shading step.
	for i, l := range terrain.LevelGrid(projRampAlt(4, 4), 4, 4, identityLight) {
		if l != 64 {
			t.Fatalf("identityLight gives vertex %d level %d, want the identity row 64", i, l)
		}
	}
	// Premise 2: the transform at that row with no tint is the identity.
	for _, c := range []color.RGBA{{A: 0xff}, {R: 1, G: 2, B: 3, A: 0xff}, {R: 0x7f, G: 0x80, B: 0xff, A: 0xff}} {
		if got := terrain.ShadeRGBA(c, [3]uint8{}, 64); got != c {
			t.Fatalf("ShadeRGBA(%v, no tint, 64) = %v, want the argument back", c, got)
		}
	}

	for _, f := range projMaps() {
		g := f.grid()
		for _, scale := range f.scales {
			got, err := terrain.CompositeProjected(ts, g, f.alt, scale)
			if err != nil {
				t.Fatalf("CompositeProjected(%s, scale %d): %v", f.name, scale, err)
			}
			want := projRecipeRender(t, ts, g, f.alt, identityLight, scale)
			projCompare(t, fmt.Sprintf("%s at scale %d, unshaded", f.name, scale), got, want)
		}
	}

	// And unshaded is a difference and not a word: under a real light the two
	// entry points must part company on a fixture that has relief to shade.
	t.Run("the shaded sibling differs under a real light", func(t *testing.T) {
		const w, h = 4, 4
		alt := projRampAlt(w, h)
		g := terrain.Grid{Width: w, Height: h, Tiles: projTiles(w * h)}
		plain, err := terrain.CompositeProjected(ts, g, alt, 1)
		if err != nil {
			t.Fatalf("CompositeProjected: %v", err)
		}
		shaded, err := terrain.CompositeProjectedLit(ts, g, alt, terrain.DefaultDaytime, 1)
		if err != nil {
			t.Fatalf("CompositeProjectedLit: %v", err)
		}
		if !projDiffers(plain, shaded) {
			t.Fatal("the shaded and unshaded projected renders are identical; the level grid reached no pixel")
		}
	})
}

func TestProjectedRejectsBadArguments(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime
	tiles := func() []uint16 { return []uint16{projLand(0), projLand(1), projLand(2), projLand(3)} }
	// Signed 0, 32, -16, 64: a canvas of 112 native rows where the flat raster
	// would measure 64, so the budget case below is decided by the projection.
	alt := func() []uint8 { return []uint8{0x00, 0x20, 0xf0, 0x40} }

	// budget marks the one case whose message names the canvas the entry point
	// measured, and heightsOnly the cases only an entry point that TAKES a height
	// grid can call bad. Both are read by the message-identity sweep below, which
	// is the reason they are recorded rather than left to the reader.
	cases := []struct {
		name        string
		ts          *terrain.Tileset
		w, h        int
		tiles       []uint16
		alt         []uint8
		scale       int
		wantMsg     string
		heightsOnly bool
		budget      bool
	}{
		{name: "nil tileset", ts: nil, w: 2, h: 2, tiles: tiles(), alt: alt(), scale: 1, wantMsg: "nil tileset"},
		{name: "zero width", ts: ts, w: 0, h: 2, scale: 1, wantMsg: "want positive dimensions"},
		{name: "zero height", ts: ts, w: 2, h: 0, scale: 1, wantMsg: "want positive dimensions"},
		{name: "negative height", ts: ts, w: 2, h: -1, tiles: tiles(), alt: alt(), scale: 1, wantMsg: "want positive dimensions"},
		{name: "nil tile grid", ts: ts, w: 2, h: 2, alt: alt(), scale: 1, wantMsg: "terrain: grid holds 0 cells"},
		{name: "short tile grid", ts: ts, w: 2, h: 2, tiles: tiles()[:3], alt: alt(), scale: 1, wantMsg: "terrain: grid holds 3 cells"},
		{name: "long tile grid", ts: ts, w: 2, h: 2, tiles: append(tiles(), 0), alt: alt(), scale: 1, wantMsg: "terrain: grid holds 5 cells"},
		{name: "nil height grid", ts: ts, w: 2, h: 2, tiles: tiles(), scale: 1, wantMsg: "height grid holds 0 cells", heightsOnly: true},
		{name: "short height grid", ts: ts, w: 2, h: 2, tiles: tiles(), alt: alt()[:3], scale: 1, wantMsg: "height grid holds 3 cells", heightsOnly: true},
		{name: "long height grid", ts: ts, w: 2, h: 2, tiles: tiles(), alt: append(alt(), 0), scale: 1, wantMsg: "height grid holds 5 cells", heightsOnly: true},
		{name: "zero scale", ts: ts, w: 2, h: 2, tiles: tiles(), alt: alt(), scale: 0, wantMsg: "want at least 1"},
		{name: "negative scale", ts: ts, w: 2, h: 2, tiles: tiles(), alt: alt(), scale: -2, wantMsg: "want at least 1"},
		{name: "over the pixel cap", ts: ts, w: 2, h: 2, tiles: tiles(), alt: alt(), scale: 8192, wantMsg: "pixel cap", budget: true},
		// A grid that is both the wrong length and far over the budget is
		// refused for its length: every argument check stands ahead of the
		// budget, and the budget is the only one of them that needs the canvas
		// measured at all (TestProjectedRejectsBeforeProjecting).
		{name: "the wrong length wins over the budget", ts: ts, w: 2, h: 2, tiles: tiles()[:3], alt: alt(), scale: 8192, wantMsg: "terrain: grid holds 3 cells"},
	}

	for _, e := range projProjectedEntries() {
		t.Run(e.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					g := terrain.Grid{Width: tc.w, Height: tc.h, Tiles: tc.tiles}
					beforeTiles := append([]uint16(nil), tc.tiles...)
					beforeAlt := append([]uint8(nil), tc.alt...)

					r, err := e.compose(tc.ts, g, tc.alt, lt, tc.scale)
					if err == nil {
						t.Fatalf("expected an error, got a render")
					}
					if r != nil {
						t.Fatalf("expected a nil render on error, got %v", r.Image.Bounds())
					}
					if !strings.Contains(err.Error(), tc.wantMsg) {
						t.Fatalf("error %q does not name the rejection it should (%q)", err, tc.wantMsg)
					}
					if len(tc.tiles) != len(beforeTiles) {
						t.Fatalf("the tile slice changed length")
					}
					for i := range beforeTiles {
						if tc.tiles[i] != beforeTiles[i] {
							t.Fatalf("tile %d was modified: %d, was %d", i, tc.tiles[i], beforeTiles[i])
						}
					}
					for i := range beforeAlt {
						if tc.alt[i] != beforeAlt[i] {
							t.Fatalf("altitude %d was modified: %d, was %d", i, tc.alt[i], beforeAlt[i])
						}
					}
				})
			}

			t.Run("the budget is measured against the projected canvas", func(t *testing.T) {
				// R-1: the projection can make an invocation too large that the
				// flat raster fits. The refusal names the projected size, not
				// H*32*scale.
				const scale = 8192
				a := alt()
				minV, maxV := projRecipeCanvas(a, 2, 2)
				if maxV-minV == 2*projCell {
					t.Fatalf("fixture is blind: the projected canvas is the flat one, %d rows", maxV-minV)
				}
				_, err := e.compose(ts, terrain.Grid{Width: 2, Height: 2, Tiles: tiles()}, a, lt, scale)
				if err == nil {
					t.Fatal("expected an over-budget error")
				}
				want := fmt.Sprintf("%dx%d px", 2*projCell*scale, (maxV-minV)*scale)
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q does not report the projected size %q", err, want)
				}
				if !strings.Contains(err.Error(), "smaller scale") {
					t.Fatalf("error %q does not recommend a smaller scale", err)
				}
			})
		})
	}

	// The scale check stands AHEAD of the budget check, and on a NEGATIVE scale
	// that ordering is not a matter of presentation but a different verdict. Both
	// output dimensions are negative at a negative scale, so their PRODUCT is
	// positive and can sail past the cap: a budget check reached first answers
	// "over the 268435456-pixel cap; try a smaller scale" — advice that is wrong
	// and cannot be followed, since a smaller scale here is more negative still —
	// where the shipped order answers "scale -8192, want at least 1".
	//
	// Nothing else in this suite decides that order. At the "negative scale" case
	// above the product is a few hundred thousand pixels, inside the cap, so the
	// budget check passes and the scale check fires with the right message
	// whichever of the two stands first.
	t.Run("a negative scale is refused before the budget it would otherwise clear", func(t *testing.T) {
		const scale = -8192
		const pixelCap = 1 << 28 // maxRenderPixels, unexported; the number the message names
		a := alt()

		// The premise, asserted: with the two checks in the other order this
		// argument set reaches the budget branch and is refused BY it — on the
		// projected canvas and on the flat one alike, so all four entry points
		// have something to get wrong.
		minV, maxV := projRecipeCanvas(a, 2, 2)
		for _, rows := range []int{maxV - minV, 2 * projCell} {
			wpx, hpx := int64(2*projCell*scale), int64(rows*scale)
			if wpx >= 0 || hpx >= 0 {
				t.Fatalf("the case is not what it claims: %dx%d px is not a pair of negatives", wpx, hpx)
			}
			if wpx*hpx <= pixelCap {
				t.Fatalf("the case is vacuous: %dx%d px multiplies to %d, inside the %d-pixel cap", wpx, hpx, wpx*hpx, pixelCap)
			}
		}

		const want = "terrain: scale -8192, want at least 1"
		for _, e := range projAllEntries() {
			r, err := e.compose(ts, terrain.Grid{Width: 2, Height: 2, Tiles: tiles()}, a, lt, scale)
			if err == nil {
				t.Fatalf("%s: expected an error, got a render", e.name)
			}
			if r != nil {
				t.Fatalf("%s: expected a nil render on error, got %v", e.name, r.Image.Bounds())
			}
			if err.Error() != want {
				t.Fatalf("%s: error is %q, want exactly %q — the scale check must stand ahead of the budget check", e.name, err, want)
			}
		}
	})

	// One rejection set, four ways in: an argument set every entry point
	// refuses must be refused in the same words, or the set has quietly become
	// two sets that agree on which inputs are bad and not on why. The budget
	// case is excluded because its message names the canvas the entry point
	// measured, and the two geometries measure different canvases by design
	// (R-1).
	t.Run("every entry point that refuses an argument set says the same words", func(t *testing.T) {
		for _, tc := range cases {
			if tc.budget {
				continue
			}
			t.Run(tc.name, func(t *testing.T) {
				g := terrain.Grid{Width: tc.w, Height: tc.h, Tiles: tc.tiles}
				first, from := "", ""
				for _, e := range projAllEntries() {
					if tc.heightsOnly && !e.heights {
						// Composite takes no height grid, so this set is not bad
						// for it at all. It is excluded from the comparison by
						// being held to the opposite claim — that it composes —
						// rather than by being skipped quietly.
						r, err := e.compose(tc.ts, g, tc.alt, lt, tc.scale)
						if err != nil {
							t.Fatalf("%s: %v; it takes no height grid, so this argument set is good for it", e.name, err)
						}
						if r == nil {
							t.Fatalf("%s: nil render and no error", e.name)
						}
						continue
					}
					_, err := e.compose(tc.ts, g, tc.alt, lt, tc.scale)
					if err == nil {
						t.Fatalf("%s: expected an error, got a render", e.name)
					}
					if from == "" {
						first, from = err.Error(), e.name
						continue
					}
					if err.Error() != first {
						t.Fatalf("%s says %q where %s says %q", e.name, err, from, first)
					}
				}
				if !strings.Contains(first, tc.wantMsg) {
					t.Fatalf("the shared message %q does not name the rejection it should (%q)", first, tc.wantMsg)
				}
			})
		}
	})
}

func TestProjectedRejectsBeforeProjecting(t *testing.T) {
	ts := terrain.LoadTileset(projSource())
	lt := terrain.DefaultDaytime

	// 1<<25 cells square. W*H alone is past any allocatable slice, which is
	// exactly why the grid-length check has to refuse the call — and why walking
	// the mesh first is the difference between microseconds and a fortnight.
	const huge = 1 << 25
	const ceiling = 10 * time.Second

	for _, e := range projProjectedEntries() {
		for _, tc := range []struct {
			name    string
			ts      *terrain.Tileset
			tiles   []uint16
			wantMsg string
		}{
			{"nil tileset", nil, nil, "terrain: nil tileset"},
			{"nil tile grid", ts, nil, "terrain: grid holds 0 cells"},
			{"short tile grid", ts, []uint16{projLand(0)}, "terrain: grid holds 1 cells"},
		} {
			t.Run(e.name+"/"+tc.name, func(t *testing.T) {
				g := terrain.Grid{Width: huge, Height: huge, Tiles: tc.tiles}

				start := time.Now()
				r, err := e.compose(tc.ts, g, nil, lt, 1)
				elapsed := time.Since(start)

				if err == nil {
					t.Fatalf("expected an error, got a render")
				}
				if r != nil {
					t.Fatalf("expected a nil render on error, got %v", r.Image.Bounds())
				}
				if !strings.Contains(err.Error(), tc.wantMsg) {
					t.Fatalf("error %q does not name the rejection it should (%q)", err, tc.wantMsg)
				}
				if elapsed > ceiling {
					t.Fatalf("the rejection took %v, past the %v ceiling: the mesh is being walked before the arguments are checked", elapsed, ceiling)
				}

				// The same argument set on both flat entry points, which have no
				// mesh to walk and never had: same rejection, same words. This is
				// the check that the parameter's shape changed and its outcomes
				// did not.
				if _, flatErr := terrain.CompositeLit(tc.ts, g, nil, lt, 1); flatErr == nil || flatErr.Error() != err.Error() {
					t.Fatalf("CompositeLit says %v, %s says %v", flatErr, e.name, err)
				}
				if _, flatErr := terrain.Composite(tc.ts, g, 1); flatErr == nil || flatErr.Error() != err.Error() {
					t.Fatalf("Composite says %v, %s says %v", flatErr, e.name, err)
				}
			})
		}
	}
}

// projRecipeWorldCorner is 0013's worldCorner(c,r), from that spec's words: the
// native canvas translated so its top, MinV, is world Y zero. There is no
// altitude term on x.
func projRecipeWorldCorner(alt []uint8, w, h, c, r int) (x, y int) {
	minV, _ := projRecipeCanvas(alt, w, h)
	return c * projCell, projRecipeVertex(alt, w, h, c, r) - minV
}

func projRecipeAltExtremes(alt []uint8) (minH, maxH int) {
	for i, b := range alt {
		v := int(int8(b))
		if i == 0 || v < minH {
			minH = v
		}
		if i == 0 || v > maxH {
			maxH = v
		}
	}
	return minH, maxH
}

// projCeilCells is ceil(n/32) for a non-negative n, the spec's own pad term.
func projCeilCells(n int) int { return (n + projCell - 1) / projCell }

// projWorldSpans is one fixture's world geometry, precomputed once because the
// window sweep walks it hundreds of times and none of it moves: per cell row,
// the world-Y bounding box of EVERY quad in that row — the least and greatest
// world corner over mesh rows row and row+1 across all columns, the far-edge
// column included. RowRange knows nothing of columns, so the row it must reach
// is the one where ANY column's quad meets the window.
type projWorldSpans struct {
	lo, hi     []int // per cell row, in world Y
	minV       int
	minH, maxH int
	worldH     int
}

func projWorldSpansOf(f projFixture) projWorldSpans {
	minV, maxV := projRecipeCanvas(f.alt, f.w, f.h)
	minH, maxH := projRecipeAltExtremes(f.alt)
	s := projWorldSpans{
		lo: make([]int, f.h), hi: make([]int, f.h),
		minV: minV, minH: minH, maxH: maxH, worldH: maxV - minV,
	}
	for row := 0; row < f.h; row++ {
		lo, hi, first := 0, 0, true
		for r := row; r <= row+1; r++ {
			for c := 0; c <= f.w; c++ {
				_, y := projRecipeWorldCorner(f.alt, f.w, f.h, c, r)
				if first || y < lo {
					lo = y
				}
				if first || y > hi {
					hi = y
				}
				first = false
			}
		}
		s.lo[row], s.hi[row] = lo, hi
	}
	return s
}

// projRaiseOnly is the blind family the plan warns about: every altitude at or
// above zero and row 0 flat, so MinV is 0 and the translation RowRange must
// apply is the identity. Nothing here can tell a correct implementation from one
// that never applies it.
func projRaiseOnly(w, h int) []uint8 {
	alt := make([]uint8, w*h)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			alt[r*w+c] = uint8(r * 16)
		}
	}
	return alt
}

// projColumnSlope rises across the COLUMNS instead, which puts MinV negative:
// the third sign, after the raise-only zero and the push-only positive.
func projColumnSlope(w, h int) []uint8 {
	alt := make([]uint8, w*h)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			alt[r*w+c] = uint8(c * 16)
		}
	}
	return alt
}

// projPushOnly is the family that discriminates: every altitude strictly
// negative, so every vertex is pushed DOWN and MinV is strictly positive.
func projPushOnly(w, h int) []uint8 {
	alt := make([]uint8, w*h)
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			alt[r*w+c] = uint8(int8(-8 - r*8 - c))
		}
	}
	return alt
}

// projAsymmetric raises and pushes inside one grid, with no symmetry between the
// two extremes: 13 is coprime with 97, so the values walk -48..48 without
// repeating inside a row.
func projAsymmetric(w, h int) []uint8 {
	alt := make([]uint8, w*h)
	for i := range alt {
		alt[i] = uint8(int8(-48 + (i*13)%97))
	}
	return alt
}

var projPushWitness = projFixture{
	name: "half the rows pushed -128 (DD-2's counter-example)",
	w:    1, h: 8,
	alt: []uint8{0x80, 0x80, 0x80, 0x80, 0x00, 0x00, 0x00, 0x00},
}

// projShortSpan is AC-3's grid whose vertices span LESS than Height*32: two rows
// at altitudes 0 and 20 reach 44 world rows, not 64.
var projShortSpan = projFixture{
	name: "vertices span less than Height*CellSize",
	w:    1, h: 2,
	alt: []uint8{0x00, 0x14},
}

// projWorldFixtures is every grid the world-space sweeps run over: the flat,
// single-axis-slope, asymmetric, negative-altitude and short-span families 0013
// T1 names, the two discriminating push grids, and the 0012 fixtures, which
// already carry the signed extremes and the far-edge cases.
var projWorldFixtures = []projFixture{
	{"flat: every altitude zero", 3, 8, projUniform(3, 8, 0x00)},
	{"single-axis slope down the rows (raise-only, MinV == 0)", 3, 8, projRaiseOnly(3, 8)},
	{"single-axis slope across the columns (MinV < 0)", 8, 3, projColumnSlope(8, 3)},
	{"push-only: every altitude negative (MinV > 0)", 3, 8, projPushOnly(3, 8)},
	projPushWitness,
	{"asymmetric: raised and pushed in one grid", 5, 6, projAsymmetric(5, 6)},
	projShortSpan,
	{"uniform -128", 3, 4, projUniform(3, 4, 0x80)},
	{"uniform +127", 3, 4, projUniform(3, 4, 0x7f)},
	{"every byte value", 16, 16, projSweepGrid()},
	projFixtureFarEdge,
	projFixtureColumn,
	projFixtureSingle,
}

// TestProjectWorldCorner covers 0013 AC-4 / SC-4 (the projection's half): every
// mesh vertex in world space, the far-edge clamp through it, and the bounds the
// translation puts the world in.
func TestProjectWorldCorner(t *testing.T) {
	t.Run("every vertex equals the recipe", func(t *testing.T) {
		// c in [0,W] and r in [0,H]: interior, right-edge, bottom-edge and corner
		// vertices are every one of them in this walk, and the named four are
		// pinned again below where a reader can see them.
		for _, f := range projWorldFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			for r := 0; r <= f.h; r++ {
				for c := 0; c <= f.w; c++ {
					gotX, gotY := p.WorldCorner(c, r)
					wantX, wantY := projRecipeWorldCorner(f.alt, f.w, f.h, c, r)
					if gotX != wantX || gotY != wantY {
						t.Fatalf("%s: WorldCorner(%d,%d) = (%d,%d), want (%d,%d) (recipe)",
							f.name, c, r, gotX, gotY, wantX, wantY)
					}
				}
			}
		}
	})

	t.Run("the four named corners of a cell", func(t *testing.T) {
		// AC-4's own list, on a grid where all four differ: interior, right-edge,
		// bottom-edge and the far corner, each read through its own vertices.
		// The values are projFarEdgeVertices minus MinV = -127.
		f := projFixtureFarEdge // 3x2
		p := terrain.Project(f.alt, f.w, f.h)
		for _, tc := range []struct {
			what         string
			c, r         int
			wantX, wantY int
		}{
			{"interior top-left", 1, 0, 32, 0},
			{"right-edge (c == Width)", 3, 0, 96, 63},
			{"bottom-edge (r == Height)", 1, 2, 32, 319},
			{"far corner (c == Width, r == Height)", 3, 2, 96, 64},
		} {
			gotX, gotY := p.WorldCorner(tc.c, tc.r)
			if gotX != tc.wantX || gotY != tc.wantY {
				t.Errorf("%s: WorldCorner(%d,%d) = (%d,%d), want (%d,%d)",
					tc.what, tc.c, tc.r, gotX, gotY, tc.wantX, tc.wantY)
			}
		}
	})

	t.Run("the far edge clamps, in world space too", func(t *testing.T) {
		// The same two identities the native mesh carries: column Width repeats
		// column Width-1 vertex for vertex, and row Height sits exactly one cell
		// below row Height-1. The translation is uniform, so it cannot repair a
		// lost clamp and cannot manufacture one.
		for _, f := range projWorldFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			for r := 0; r <= f.h; r++ {
				_, got := p.WorldCorner(f.w, r)
				_, want := p.WorldCorner(f.w-1, r)
				if got != want {
					t.Errorf("%s: WorldCorner(%d,%d).y = %d, want %d (column %d clamps to %d)",
						f.name, f.w, r, got, want, f.w, f.w-1)
				}
			}
			for c := 0; c <= f.w; c++ {
				_, got := p.WorldCorner(c, f.h)
				_, prev := p.WorldCorner(c, f.h-1)
				if want := prev + projCell; got != want {
					t.Errorf("%s: WorldCorner(%d,%d).y = %d, want %d (row %d clamps to %d, one cell lower)",
						f.name, c, f.h, got, want, f.h, f.h-1)
				}
			}
		}
	})

	t.Run("world x carries no altitude term", func(t *testing.T) {
		// TERR-GEOM-035: no altitude-derived term reaches the destination x, so a
		// column's world x is the same on every row of every grid.
		for _, f := range projWorldFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			for r := 0; r <= f.h; r++ {
				for c := 0; c <= f.w; c++ {
					if x, _ := p.WorldCorner(c, r); x != c*projCell {
						t.Fatalf("%s: WorldCorner(%d,%d).x = %d, want %d", f.name, c, r, x, c*projCell)
					}
				}
			}
		}
	})

	t.Run("the world is [0, CanvasHeight] and both ends are reached", func(t *testing.T) {
		// The translation is by MinV exactly, so world Y zero is the mesh's own
		// top: a world that started below zero or ended short of CanvasHeight
		// would be a second, unstated origin.
		for _, f := range projWorldFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			top, bottom := false, false
			for r := 0; r <= f.h; r++ {
				for c := 0; c <= f.w; c++ {
					_, y := p.WorldCorner(c, r)
					if y < 0 || y > p.CanvasHeight() {
						t.Fatalf("%s: WorldCorner(%d,%d).y = %d is outside [0,%d]", f.name, c, r, y, p.CanvasHeight())
					}
					top = top || y == 0
					bottom = bottom || y == p.CanvasHeight()
				}
			}
			if !top || !bottom {
				t.Errorf("%s: world height %d is wider than the mesh (top reached %v, bottom reached %v)",
					f.name, p.CanvasHeight(), top, bottom)
			}
		}
	})

	t.Run("the short-span fixture really is short", func(t *testing.T) {
		// Guarding the fixture, not the code: if this grid ever stopped spanning
		// less than Height*32, AC-3's short-world case would quietly leave the
		// suite while every assertion still passed.
		f := projShortSpan
		p := terrain.Project(f.alt, f.w, f.h)
		if got, flat := p.CanvasHeight(), f.h*projCell; got != 44 || got >= flat {
			t.Fatalf("%s: CanvasHeight = %d, want 44 and short of the flat %d", f.name, got, flat)
		}
	})
}

func TestProjectAltitudeExtremes(t *testing.T) {
	for _, f := range projWorldFixtures {
		p := terrain.Project(f.alt, f.w, f.h)
		wantMin, wantMax := projRecipeAltExtremes(f.alt)
		if p.MinH != wantMin || p.MaxH != wantMax {
			t.Errorf("%s: altitude extremes [%d,%d], want [%d,%d] (recipe, over the grid)",
				f.name, p.MinH, p.MaxH, wantMin, wantMax)
		}
	}

	t.Run("signed, not unsigned", func(t *testing.T) {
		// 0x80 is -128 and 0x7f is 127, so a grid holding both spans 255 and not
		// 1. Read unsigned the extremes would be [127,128].
		p := terrain.Project([]uint8{0x7f, 0x80}, 2, 1)
		if p.MinH != -128 || p.MaxH != 127 {
			t.Errorf("extremes of {0x7f,0x80} = [%d,%d], want [-128,127]", p.MinH, p.MaxH)
		}
	})

	t.Run("total on a grid no compositor would accept", func(t *testing.T) {
		// Project validates nothing; the extremes must answer rather than panic or
		// read past the slice.
		for _, c := range []struct {
			alt  []uint8
			w, h int
		}{{nil, 0, 0}, {nil, 4, 4}, {[]uint8{0x01}, 4, 4}, {[]uint8{0x01}, -3, -3}} {
			p := terrain.Project(c.alt, c.w, c.h)
			if p.MinH > p.MaxH {
				t.Errorf("extremes [%d,%d] are inverted", p.MinH, p.MaxH)
			}
		}
	})
}

// projAssertRowRange checks one window against both halves of RowRange's
// contract, and is the whole of SC-5's first two clauses.
//
// A cell row counts as meeting the window when its world-Y span and the window
// OVERLAP, both treated half-open: [lo,hi) against [top,bottom). That is the
// convention the shipped flat camera already draws by — VisibleTiles takes
// Row1 = ceil((Y+viewH)/32), so a flat band beginning exactly at the window's
// bottom edge is not visible — and 0013's AC-6 says "meets" without deciding the
// zero-width touch. Deciding it the other way would demand a row whose quad
// contributes no pixel, and would part company with flat mode on the same map.
func projAssertRowRange(t *testing.T, f projFixture, s projWorldSpans, p terrain.Projection, top, bottom float64) {
	t.Helper()

	r0, r1 := p.RowRange(top, bottom)

	if r0 < 0 || r1 > f.h || r0 > r1 {
		t.Fatalf("%s: RowRange(%v,%v) = [%d,%d), outside [0,%d] or inverted", f.name, top, bottom, r0, r1, f.h)
	}
	if bottom < top {
		// An inverted window covers nothing, so neither property below is
		// defined over it: there is no set of rows it meets, and no flat range
		// to bound the answer against. Staying in the grid is the whole of the
		// contract here, and it was just checked.
		return
	}

	// CONTAINMENT (AC-6).
	for row := 0; row < f.h; row++ {
		if !(float64(s.lo[row]) < bottom && float64(s.hi[row]) > top) {
			continue
		}
		if row < r0 || row >= r1 {
			t.Fatalf("%s: RowRange(%v,%v) = [%d,%d) drops row %d, whose quads span world [%d,%d)",
				f.name, top, bottom, r0, r1, row, s.lo[row], s.hi[row])
		}
	}

	// THE TWO-SIDED BOUND (0013 AC-6a). Measured in native V, which is the world
	// window shifted back by MinV — the spec says so in those words.
	a, b := top+float64(s.minV), bottom+float64(s.minV)
	flat0 := int(math.Floor(a / projCell))
	flat1 := int(math.Ceil(b / projCell))
	up, down := max(0, s.maxH), max(0, -s.minH)
	lo := min(max(flat0-projCeilCells(down)-1, 0), f.h)
	hi := min(max(flat1+projCeilCells(up)+1, 0), f.h)
	if r0 < lo || r1 > hi {
		t.Fatalf("%s: RowRange(%v,%v) = [%d,%d), outside the two-sided bound [%d,%d)",
			f.name, top, bottom, r0, r1, lo, hi)
	}
}

// projRowWindows is the window sweep one fixture is walked over: hard against
// the top edge and hard against the bottom edge of its world, which is where an
// anchoring error is visible and where a window taken near the middle of a map
// hides it; plus a slide across the whole world at several heights, at offsets
// stepping by a stride coprime with the cell so nothing stays cell-aligned, and
// at fractional positions, since a camera's world coordinates are float64 and
// almost never land on an integer.
func projRowWindows(worldH int) [][2]float64 {
	var out [][2]float64
	add := func(top, height float64) { out = append(out, [2]float64{top, top + height}) }

	for _, vh := range []float64{1, 7, 32, 33, 100, 320} {
		add(0, vh)                    // hard against the top
		add(float64(worldH)-vh, vh)   // hard against the bottom
		add(-vh/2, vh)                // straddling the top
		add(float64(worldH)-vh/2, vh) // straddling the bottom
		for off := -64; off <= worldH+64; off += 7 {
			add(float64(off), vh)
			add(float64(off)+0.5, vh)
			add(float64(off)+0.25, vh)
		}
	}
	// Windows a degenerate camera can produce: empty, inverted, and one that
	// covers everything with room to spare.
	out = append(out,
		[2]float64{0, 0},
		[2]float64{100, 100},
		[2]float64{200, -200},
		[2]float64{float64(-worldH), float64(2 * worldH)},
	)
	return out
}

func TestProjectRowRange(t *testing.T) {
	t.Run("the fixtures span all three signs of MinV", func(t *testing.T) {
		// Guarding the witness set, not the code. If every fixture drifted to
		// MinV == 0 the sweep below would still pass against an implementation
		// that never translates, which is the defect this test exists for.
		seen := map[int]bool{}
		for _, f := range projWorldFixtures {
			minV, _ := projRecipeCanvas(f.alt, f.w, f.h)
			switch {
			case minV < 0:
				seen[-1] = true
			case minV > 0:
				seen[1] = true
			default:
				seen[0] = true
			}
		}
		if !seen[-1] || !seen[0] || !seen[1] {
			t.Fatalf("fixtures reach MinV signs %v, want all of -1, 0 and +1", seen)
		}
	})

	t.Run("DD-2's counter-example, hand-computed", func(t *testing.T) {
		// The 1x8 grid pushed -128 for its first four rows. Its world corners
		// are, mesh row by mesh row: 0, 32, 64, 96, then 0, 32, 64, 96 again, and
		// 128 for the far-edge row. So at the very top of the world, rows 0, 3
		// and 4 all have a quad inside the first 32 world rows — row 3's quad
		// spans the whole world height, because its bottom edge is the cliff.
		f := projPushWitness
		p := terrain.Project(f.alt, f.w, f.h)
		if p.MinV != 128 || p.CanvasHeight() != 128 {
			t.Fatalf("witness: canvas [%d,%d), want [128,256)", p.MinV, p.MaxV)
		}

		r0, r1 := p.RowRange(0, 32)
		for _, row := range []int{0, 3, 4} {
			if row < r0 || row >= r1 {
				t.Errorf("RowRange(0,32) = [%d,%d) drops row %d, which is drawn in the top 32 world rows",
					r0, r1, row)
			}
		}
		if r0 != 0 || r1 != 5 {
			t.Errorf("RowRange(0,32) = [%d,%d), want [0,5) (DD-2: floor((128-128)/32)-1 clipped, ceil((160+0)/32))",
				r0, r1)
		}
	})

	t.Run("containment and the bound over every fixture and window", func(t *testing.T) {
		for _, f := range projWorldFixtures {
			s := projWorldSpansOf(f)
			p := terrain.Project(f.alt, f.w, f.h)
			for _, w := range projRowWindows(s.worldH) {
				projAssertRowRange(t, f, s, p, w[0], w[1])
			}
		}
	})

	t.Run("the whole map is not an answer", func(t *testing.T) {
		// The bound half, shown discriminating rather than left to be believed:
		// on a grid holding every byte value, [0,Height) violates it for a
		// one-cell window, so an implementation that shrugged and returned
		// everything would fail the sweep above.
		f := projFixture{"every byte value", 16, 16, projSweepGrid()}
		s := projWorldSpansOf(f)
		a, b := 0+float64(s.minV), 32+float64(s.minV)
		lo := min(max(int(math.Floor(a/projCell))-projCeilCells(max(0, -s.minH))-1, 0), f.h)
		hi := min(max(int(math.Ceil(b/projCell))+projCeilCells(max(0, s.maxH))+1, 0), f.h)
		if lo <= 0 && hi >= f.h {
			t.Fatalf("the bound admits the whole map [0,%d): [%d,%d)", f.h, lo, hi)
		}
	})

	t.Run("clipped into the grid, never past it", func(t *testing.T) {
		// Windows far outside the world on either side, and the ones a degenerate
		// camera produces, yield an empty range at the edge they ran off — not a
		// negative row, not a row past Height, and not an int conversion of a
		// NaN.
		for _, f := range projWorldFixtures {
			p := terrain.Project(f.alt, f.w, f.h)
			for _, w := range [][2]float64{
				{-1e6, -1e6 + 32},
				{1e6, 1e6 + 32},
				{math.Inf(-1), math.Inf(1)},
				{math.Inf(1), math.Inf(-1)},
				{math.NaN(), math.NaN()},
			} {
				r0, r1 := p.RowRange(w[0], w[1])
				if r0 < 0 || r1 > f.h || r0 > r1 {
					t.Fatalf("%s: RowRange(%v,%v) = [%d,%d), outside [0,%d] or inverted",
						f.name, w[0], w[1], r0, r1, f.h)
				}
			}
		}
	})

	t.Run("total on a grid no compositor would accept", func(t *testing.T) {
		for _, c := range []struct {
			alt  []uint8
			w, h int
		}{{nil, 0, 0}, {nil, 4, 4}, {[]uint8{0x01}, 4, 4}, {[]uint8{0x01}, -3, -3}, {[]uint8{0x01}, 0, 1}} {
			p := terrain.Project(c.alt, c.w, c.h)
			r0, r1 := p.RowRange(-1000, 1000)
			if r0 > r1 {
				t.Errorf("RowRange over a %dx%d grid = [%d,%d), inverted", c.w, c.h, r0, r1)
			}
		}
	})

	t.Run("the borrowed grid is not written", func(t *testing.T) {
		for _, f := range projWorldFixtures {
			before := append([]uint8(nil), f.alt...)
			p := terrain.Project(f.alt, f.w, f.h)
			for r := 0; r <= f.h; r++ {
				for c := 0; c <= f.w; c++ {
					_, _ = p.WorldCorner(c, r)
				}
			}
			for _, w := range projRowWindows(p.CanvasHeight()) {
				_, _ = p.RowRange(w[0], w[1])
			}
			for i := range before {
				if f.alt[i] != before[i] {
					t.Fatalf("%s: altitude[%d] changed 0x%02x -> 0x%02x", f.name, i, before[i], f.alt[i])
				}
			}
		}
	})
}
