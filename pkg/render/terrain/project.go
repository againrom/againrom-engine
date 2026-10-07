package terrain

import (
	"image"
	"image/color"
	"math"
)

// The projection: a map's altitude grid turned into the vertex mesh a
// height-displaced terrain render is drawn on, the canvas that mesh spans,
// and the compositor that draws a map onto it.
//
// The mesh formula is a decoded game fact (research TERR-GEOM-031): one named
// instruction each for the r*32 term, for the SUBTRACTION of the altitude, for
// the altitude being read SIGNED (a MOVSX — and decidable only there, since not
// one of the 880 704 shipped height bytes reaches 0x80), and for the identity of
// the grid it is read from. Displacement is vertical only: no altitude-derived
// term reaches the destination X (TERR-GEOM-035).
//
// Two things here are the project's own choice, not transcription:
//
//   - The far-edge ring. A cell owns four corners, so the last cell row and
//     column need vertices outside a W*H grid. The engine reads them at a raw
//     flat offset into that unpadded grid — the last column lands on the next
//     row's column 0, the last row past the allocation, and the allocator does
//     not zero it (TERR-EDGE-024..026). We clamp the index into the grid
//     instead: the safe, intent-matching treatment the research names for a
//     port, never a claim about what the engine draws there.
//   - The canvas itself. The engine has a clip rect, a scroll origin and an
//     over-scan margin; it defines no map-sized image and has no scale. The
//     [minV, maxV) canvas, its origin and the integer scale are ours.
//
// See docs/0012-height-displaced-terrain/spec.md.

// Projection is one map's height-displaced vertex mesh: the destination row
// every mesh vertex projects to, and the extremes those rows reach.
//
// It is a value, computed by Project before anything is allocated, and it holds
// no image — the compositor asks it for a vertex and for the canvas it must
// allocate. Width and Height are the map's size in cells; the mesh is one vertex
// wider and taller than that, (Width+1) x (Height+1) in all, because a cell owns
// its four corners and the last cell's far corners lie outside the grid.
//
// MinV and MaxV are the least and greatest vertex row over the whole mesh. The
// native canvas is [0, Width*CellSize) x [MinV, MaxV): MaxV is a bottom edge and
// not a drawn row, and MinV is the vertical origin the render translates by and
// reports.
type Projection struct {
	// Width and Height are the map's size in cells, as passed to Project.
	Width  int
	Height int

	// MinV and MaxV are the canvas extremes over all (Width+1) x (Height+1)
	// vertices: the canvas is the half-open [MinV, MaxV).
	MinV int
	MaxV int

	// MinH and MaxH are the least and greatest ALTITUDE over the grid, read
	// signed — a different quantity from MinV/MaxV, which are destination
	// rows. They bound how far a cell's corners can leave the band its row
	// index nominally occupies, which is what RowRange is made of.
	//
	// They are the extremes over the grid and over the mesh alike: the
	// far-edge ring reads clamped indices, so every altitude it can see is
	// one the grid already holds.
	MinH int
	MaxH int

	// altitudes is the caller's grid, borrowed and never written; it is read
	// signed at the point of use. Unexported because it is the one field a
	// caller has no business replacing between the pass that measured the
	// canvas and the pass that draws into it.
	altitudes []uint8
}

// Project computes the vertex mesh of one altitude grid and the canvas it
// spans.
//
// altitudes is the map's Width*Height altitude grid in row-major order — the
// bytes alm decodes and CompositeLit already takes — read as SIGNED 8-bit values
// at the point of use: 0x80 is -128, not 128.
//
// The extremes are found in one pass over the whole mesh, before any image
// exists, because the canvas the compositor must allocate is exactly those
// extremes.
//
// Project validates nothing: a non-positive dimension and a grid whose
// length is not Width*Height are the compositor's rejections, made before it
// gets here. It is nonetheless total — every argument set yields a
// Projection whose accessors answer rather than panic.
func Project(altitudes []uint8, width, height int) Projection {
	p := Projection{Width: width, Height: height, altitudes: altitudes}

	// One pass over all (Width+1) x (Height+1) vertices. The c == Width column
	// is the c == Width-1 column's own clamp and so can move no extreme; it is
	// still walked, because the mesh is (Width+1) wide and a loop that stopped
	// early would be a second, unstated rule about where the mesh ends.
	minV := p.Vertex(0, 0)
	maxV := minV
	minH := p.Altitude(0, 0)
	maxH := minH
	for r := 0; r <= height; r++ {
		for c := 0; c <= width; c++ {
			v := p.Vertex(c, r)
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
			h := p.Altitude(c, r)
			if h < minH {
				minH = h
			}
			if h > maxH {
				maxH = h
			}
		}
	}
	p.MinV, p.MaxV = minV, maxV
	p.MinH, p.MaxH = minH, maxH
	return p
}

// Altitude returns h(c,r): the altitude at mesh vertex (c,r), read signed,
// with an index outside the grid clamped into it:
//
//	h(c,r) = altitude[ min(r,Height-1)*Width + min(c,Width-1) ]
//
// That clamp is what gives the far-edge ring — column Width and row Height — any
// value at all; it is our defined choice for a region the engine defines no
// value for (see the file comment).
//
// A negative coordinate is outside the contract's domain (c is in [0,Width], r
// in [0,Height]) and is pinned to 0, which is what keeps the method total rather
// than an index panic.
func (p Projection) Altitude(c, r int) int {
	c = min(c, p.Width-1)
	r = min(r, p.Height-1)
	c = max(c, 0)
	r = max(r, 0)
	i := r*p.Width + c
	if i < 0 || i >= len(p.altitudes) {
		// Unreachable for a Projection over a grid the compositor validated: the
		// clamps put the index inside a Width*Height grid. It is the same
		// belt-and-braces bound paletteColor carries, so that a hand-built or zero
		// Projection answers instead of panicking.
		return 0
	}
	return int(int8(p.altitudes[i]))
}

// AnchorHeight returns the signed native-pixel height a marker anchored in
// cell (col,row) is lifted by (0015 spec, Height lookup contract): the mean of
// that cell's four corner altitudes, summed and divided by 4 with Go's native
// truncating-toward-zero integer division —
//
//	h00 = Altitude(col,row)     h10 = Altitude(col+1,row)
//	h01 = Altitude(col,row+1)   h11 = Altitude(col+1,row+1)
//	AnchorHeight(col,row) = (h00 + h10 + h01 + h11) / 4
//
// col and row are clamped into the grid's own cell range — [0,Width-1] and
// [0,Height-1] — using the SAME min-then-max order Altitude itself uses,
// BEFORE col+1/row+1 are formed — never after. Altitude clamps each index
// it is given, but only once it reaches it: forming col+1 from an unclamped
// col at math.MaxInt overflows to a negative number first, which Altitude's
// own clamp then reads as column 0 — the wrong edge, silently. Clamping
// col/row here first keeps col+1/row+1 ordinary ints (at most Width/Height)
// that can never overflow, so an off-grid anchor returns the nearest edge
// cell's own mean instead.
//
// It is total over any col/row, a pure function of the borrowed altitude
// slice, and never mutates it. On a flat cell — four corners equal to h
// — it returns exactly h.
func (p Projection) AnchorHeight(col, row int) int {
	col = min(col, p.Width-1)
	row = min(row, p.Height-1)
	col = max(col, 0)
	row = max(row, 0)

	h00 := p.Altitude(col, row)
	h10 := p.Altitude(col+1, row)
	h01 := p.Altitude(col, row+1)
	h11 := p.Altitude(col+1, row+1)
	return (h00 + h10 + h01 + h11) / 4
}

// Vertex returns V(c,r), the destination row mesh vertex (c,r) projects to
// before the canvas is translated:
//
//	V(c,r) = r*CellSize - h(c,r)
//
// The altitude is SUBTRACTED, so a larger altitude yields a smaller V and moves
// the pixel UP the image. There is no x term: displacement is vertical only, and
// a cell's destination columns are always its own.
func (p Projection) Vertex(c, r int) int {
	return r*CellSize - p.Altitude(c, r)
}

func (p Projection) CanvasHeight() int {
	return p.MaxV - p.MinV
}

func (p Projection) OutputSize(scale int) (w, h int64) {
	return int64(p.Width) * int64(CellSize) * int64(scale),
		int64(p.CanvasHeight()) * int64(scale)
}

// --- world space: the camera world of the windowed viewer (0013) ---
//
// The interactive viewer draws this same mesh onto a camera world rather
// than onto an image, so it needs the canvas translated to a top-left origin
// and it needs to know which cell rows a view window can reach.

// WorldCorner is mesh vertex (c,r) in the camera's world — the native
// canvas translated so its top, MinV, is world Y zero:
//
//	worldCorner(c,r) = ( c*CellSize , V(c,r) - MinV )
//
// It is the same translation compositeProjected applies as offsetY = -MinV*scale
// at scale 1; naming it is what keeps a caller from inventing a second one.
// World x carries no altitude term, because the projection has none
// (TERR-GEOM-035).
//
// For every vertex of a grid this Projection was built over, the result lies in
// [0, Width*CellSize] x [0, CanvasHeight()] with both ends of each attained,
// since MinV and MaxV are the extremes of exactly this mesh.
func (p Projection) WorldCorner(c, r int) (x, y int) {
	return c * CellSize, p.Vertex(c, r) - p.MinV
}

// CellColumnBounds returns the top and bottom rows of one cell at native world
// column x. The rows are in the same translated camera world as WorldCorner.
//
// This is ROM1's ground-picker edge model (`TERR-GEOM-036` part (d)), not the
// terrain rasterizer's table walk and not AnchorHeight's placement surface. For
// each edge it evaluates
//
//	y0 + ((y1-y0)*(x&31))/32
//
// with integer division. Go integer division truncates toward zero, including
// when the edge delta is negative. The low five bits are used exactly: native
// column 32 is column 0 of the next cell, while column 31 remains one pixel
// short of the right corner.
//
// col and row name the cell independently of x. The caller derives them from
// the same native world point before calling this method. Bounds are returned
// in top/bottom corner order without sorting. A crossed pair therefore remains
// inverted for the containment test rather than becoming a pickable span.
func (p Projection) CellColumnBounds(col, row, x int) (top, bottom int) {
	local := x & (CellSize - 1)
	_, topLeft := p.WorldCorner(col, row)
	_, topRight := p.WorldCorner(col+1, row)
	_, bottomLeft := p.WorldCorner(col, row+1)
	_, bottomRight := p.WorldCorner(col+1, row+1)

	top = topLeft + (topRight-topLeft)*local/CellSize
	bottom = bottomLeft + (bottomRight-bottomLeft)*local/CellSize
	return top, bottom
}

// RowRange is the half-open range of cell rows [r0, r1) a world-Y window can
// reach, clipped to [0, Height].
//
// top and bottom are world Y — the coordinate WorldCorner answers in, and
// the one a camera holds. They are converted back to native V by adding
// MinV, which is the step no pad applied to a camera's own row band can
// make: that band is a world coordinate while the altitudes that would pad
// it are native, and the two are offset by MinV, up to eight cell rows
// apart.
//
// Cell row r's four corners are mesh vertices on rows r and r+1, so every one of
// them lies within
//
//	[ r*CellSize - MaxH , (r+1)*CellSize - MinH ]
//
// and the rows whose corners can meet a native window [a,b] are therefore
// contained in
//
//	r0 = floor((a + MinH)/CellSize) - 1     r1 = ceil((b + MaxH)/CellSize)
//
// SIGN, in the one direction it can be read wrong: a POSITIVE altitude is
// SUBTRACTED from V, so a raised cell is drawn ABOVE the band its row index
// names and is reached by looking at LARGER row indices. MaxH is what widens r1;
// MinH, being negative on a pushed grid, is what widens r0.
//
// The range is a superset and never exact: it is bounded by the whole grid's
// altitude extremes, not by the altitudes of the rows it names. It is a superset
// by at most one row past the inequality above, which is what keeps it inside
// the two-sided bound 0013 requires — "draw the whole map" is not an answer.
//
// Both ends are clipped into the grid, and a window that is inverted, non-finite
// or wholly outside the world yields an empty range rather than an index a
// caller could not use.
func (p Projection) RowRange(top, bottom float64) (r0, r1 int) {
	if p.Height <= 0 {
		return 0, 0
	}

	a := top + float64(p.MinV)
	b := bottom + float64(p.MinV)

	r0 = clampRow(rowFloor(a+float64(p.MinH))-1, p.Height)
	r1 = clampRow(rowCeil(b+float64(p.MaxH)), p.Height)
	if r1 < r0 {
		r1 = r0
	}
	return r0, r1
}

// rowFloor and rowCeil are floor(v/CellSize) and ceil(v/CellSize) over a native
// V coordinate, guarding the two values a camera can hand a viewer that an int
// conversion cannot take: a NaN, which converts to an implementation-defined
// int, and a magnitude past int32, which a degenerate zoom produces. Both
// saturate rather than wrap, and RowRange clips into the grid immediately after,
// so a saturated bound becomes an empty or a full range and never an index.
func rowFloor(v float64) int { return rowQuant(math.Floor(v / CellSize)) }
func rowCeil(v float64) int  { return rowQuant(math.Ceil(v / CellSize)) }

func rowQuant(q float64) int {
	switch {
	case math.IsNaN(q):
		return 0
	case q > math.MaxInt32:
		return math.MaxInt32
	case q < math.MinInt32:
		return math.MinInt32
	}
	return int(q)
}

// clampRow holds a row index in [0, height]. Height is the inclusive upper bound
// because the range is half-open: r1 == Height is the row past the last one.
func clampRow(r, height int) int {
	return min(max(r, 0), height)
}

// --- one cell's geometry on the projected canvas ---

// cellQuad is one cell's four projected corners, in the order the spec names
// them: yTL and yTR carry the cell's top edge, yBL and yBR its bottom. They are
// native destination rows — before the canvas translation and before scale.
type cellQuad struct {
	yTL, yTR, yBL, yBR int
}

// quad returns cell (col,row)'s projected corners: the mesh vertices (col,row),
// (col+1,row), (col,row+1) and (col+1,row+1). The +1 vertices of the last cell
// row and column are the clamped far-edge ring (see Altitude).
func (p Projection) quad(col, row int) cellQuad {
	return cellQuad{
		yTL: p.Vertex(col, row),
		yTR: p.Vertex(col+1, row),
		yBL: p.Vertex(col, row+1),
		yBR: p.Vertex(col+1, row+1),
	}
}

// flatCell reports whether cell (col,row) takes the flat path: exactly when
// its four corner ALTITUDES are equal — not by any threshold on their
// spread, and not by the shape of the projected quad.
func (p Projection) flatCell(col, row int) bool {
	a := p.Altitude(col, row)
	return a == p.Altitude(col+1, row) &&
		a == p.Altitude(col, row+1) &&
		a == p.Altitude(col+1, row+1)
}

func (q cellQuad) span(i int) (top, s int) {
	if q.yTL < q.yTR {
		top = EdgeForward(q.yTL, q.yTR, i)
	} else {
		top = EdgeMirrored(q.yTL, q.yTR, i)
	}

	var bot int
	if q.yBL >= q.yBR {
		bot = EdgeForward(q.yBL, q.yBR, i)
	} else {
		bot = EdgeMirrored(q.yBL, q.yBR, i)
	}
	return top, bot - top
}

const srcFracBits = 16

// --- the height-displaced compositor ---

// CompositeProjected draws a whole map's terrain onto the height-displaced
// geometry of spec 0012, unshaded. It is the projected sibling of Composite,
// and the unshaded sibling of CompositeProjectedLit: the geometry below and
// the light are independent axes, so all four combinations exist and each is
// reachable on its own.
//
// It takes a height grid where Composite takes none, and validates its length
// like the shaded pair does. That is not shading creeping in: the altitudes are
// what the GEOMETRY is made of here, and a projected render without them has no
// mesh to draw on. Nothing else about the pixels differs from Composite's — no
// level grid is built and no attenuation is applied, so a terrain pixel is the
// palette colour, or the dirt overlay's where an impassable cell shows it, and
// nothing more.
//
// On a map whose altitudes are all equal it is byte-identical to Composite
// — same pixels, same dimensions, same placeholder count — at every
// scale, with OriginY the negated altitude (AC-1).
func CompositeProjected(ts *Tileset, g Grid, heights []uint8, scale int) (*Render, error) {
	return compositeProjected(ts, g, heights, Light{}, scale, withoutShading)
}

// CompositeProjectedLit draws a whole map's terrain, relief-shaded, onto the
// height-displaced geometry of spec 0012. Tile-word resolution, the water
// phase, the dirt composite, the placeholder fill and its count, the level
// grid and the shading transform are CompositeLit's, unchanged; what changes
// is where a cell's pixels land.
//
// Every mesh vertex is projected by V(c,r) = r*32 - h(c,r) (Project). The
// canvas is the half-open [MinV, MaxV) those vertices span, translated so
// MinV becomes output row 0 and reported as Render.OriginY; the width is the
// flat render's own W*32*scale. A cell whose four corner altitudes are equal
// is drawn as the axis-aligned 32x32 rectangle at its own projected top
// edge; every other cell is drawn as 32 independent vertical spans, each
// resampled from the sub-cell's 32 source rows and each free to be empty.
// Cells are painted in row-major order and a later cell owns every pixel two
// cells cover; a pixel no drawn column covers is left transparent,
// image.NewRGBA's zero fill being exactly that.
//
// Arguments are validated and rejected atomically, in CompositeLit's order and
// with its messages (validateComposite): a nil tileset, a non-positive
// dimension, a tile or height grid whose length is not Width*Height, a scale
// below 1, or an output over the pixel cap yields a nil render and an error.
// The budget is measured against the PROJECTED canvas, so a map whose altitudes
// spread widely can be refused at a scale the flat raster accepts (R-1); the
// error recommends a smaller scale, and the flat entry points remain.
func CompositeProjectedLit(ts *Tileset, g Grid, heights []uint8, lt Light, scale int) (*Render, error) {
	return compositeProjected(ts, g, heights, lt, scale, withShading)
}

// The two values compositeProjected's needShading parameter takes, so an entry
// point names the one axis it stands on rather than passing a bare boolean —
// withHeights and withoutHeights being the same idiom one tier down.
const (
	withShading    = true
	withoutShading = false
)

// cellShading is the level input of one cell's blit: the light's sky tint and the
// cell's four corner relief levels, l00 top-left, l10 top-right, l01 bottom-left,
// l11 bottom-right. A NIL *cellShading is the unshaded path — not a zero level,
// which would be the brightest row there is, but no level at all: the pixel is
// written as composed, and no level is computed for it.
//
// It exists so that "whether a level grid is consulted" is one nil test
// carried down to the pixel, rather than a second copy of the cell loop and
// of the span blitter differing in one call.
type cellShading struct {
	tint               [3]uint8
	l00, l10, l01, l11 uint8
}

// compositeProjected is the one core both projected entry points draw with,
// and the only place the height-displaced cell loop is written. The single
// thing they differ in is its last parameter: shaded builds a level grid and
// consults it per cell and per pixel, unshaded builds none and consults
// none. Everything else — the mesh, the canvas, the selector, both
// blitters, the paint order, the placeholder fill and its count, and the
// whole rejection set — is one implementation serving both, so a change to
// any of it cannot reach one entry point and miss the other.
func compositeProjected(ts *Tileset, g Grid, heights []uint8, lt Light, scale int, needShading bool) (*Render, error) {
	var p Projection
	wpx, hpx, err := validateComposite(ts, g, heights, withHeights, scale, func() int {
		p = Project(heights, g.Width, g.Height)
		return p.CanvasHeight()
	})
	if err != nil {
		return nil, err
	}

	// The level grid and the per-cell shading input are the shaded path's alone.
	// sh stays nil on the unshaded one, all the way down to the pixel, and levels
	// stays a nil slice that is never read: LevelGrid is not merely ignored, it is
	// not called.
	var levels []uint8
	var cellSh cellShading
	var sh *cellShading
	if needShading {
		levels = LevelGrid(heights, g.Width, g.Height, lt)
		cellSh.tint = lt.SkyTint
		sh = &cellSh
	}

	img := image.NewRGBA(image.Rect(0, 0, int(wpx), int(hpx)))
	out := &Render{Image: img, OriginY: p.MinV}

	// The canvas translation, in output pixels: native row MinV is output row 0,
	// so native row y lands at offsetY + y*scale. Every blit below takes its
	// destination origin already translated, exactly as the flat compositors do.
	offsetY := -p.MinV * scale
	cellpx := CellSize * scale

	for row := 0; row < g.Height; row++ {
		for col := 0; col < g.Width; col++ {
			word := g.Tiles[row*g.Width+col]
			ref := Resolve(word)

			q := p.quad(col, row)
			flat := p.flatCell(col, row)
			originX := col * cellpx

			// Slot and SubCell are both bounds-checked and nil-safe, so an
			// off-corpus word lands here as a nil sub-cell, not a panic.
			src := ts.Slot(ref.Slot).SubCell(ref.Sub)
			if src == nil {
				// A placeholder takes the selector and the geometry of the cell it
				// replaces, and stays unlit: it is a diagnostic fill, not terrain, and a
				// rectangle on a sloped cell would be a mark where its cell is not. It
				// is therefore the one cell both entry points draw identically.
				if flat {
					fillCell(img, originX, offsetY+q.yTL*scale, cellpx, PlaceholderColor)
				} else {
					fillCellSpans(img, originX, offsetY, scale, q, PlaceholderColor)
				}
				out.Placeholders++
				continue
			}

			var dirt *image.Paletted
			if IsImpassable(word) && !ref.Water {
				dirt = ts.Dirt.SubCell(DirtSubCell(col, row))
			}

			// One struct refilled per cell rather than one per cell allocated; it is
			// read within this iteration and never retained. The far-edge +1 corner
			// is clamped into the grid by CornerLevels, the same read CompositeLit
			// makes.
			if sh != nil {
				cl := CornerLevels(levels, g.Width, g.Height, col, row)
				sh.l00, sh.l10, sh.l01, sh.l11 = cl[0], cl[1], cl[2], cl[3]
			}

			if flat {
				// The rectangle blitters are the flat compositors' own, called rather
				// than copied: a flat cell is the same 32x32 rectangle wherever the
				// geometry puts its origin, which is the whole reason both take a
				// destination origin.
				originY := offsetY + q.yTL*scale
				if sh == nil {
					drawCell(img, originX, originY, scale, src, dirt)
				} else {
					drawCellLit(img, originX, originY, scale, src, dirt, sh.tint, sh.l00, sh.l10, sh.l01, sh.l11)
				}
			} else {
				drawCellSpans(img, originX, offsetY, scale, q, src, dirt, sh)
			}
		}
	}
	return out, nil
}

// drawCellSpans blits one 32x32 sub-cell as the 32 per-column vertical spans
// of a sloped cell, relief-shaded where sh is non-nil. It is the sloped
// counterpart of drawCell and drawCellLit, which stay the blitters for a
// flat cell — one function covering both of them, because the shading is a
// per-pixel multiply at the end of a pixel's composition and nothing about
// the geometry above it changes when there is none.
//
// originX is the destination pixel of the cell's first column and offsetY the
// output row native row 0 falls on — the canvas translation, -MinV*scale — so
// native row y lands at offsetY + y*scale. Both are output pixels, as
// drawCellLit's origin is.
//
// Column i is drawn top .. srcRow < 32 holds for every positive S by that
// division's own definition, so no clamp is written on the source row; and
// each edge is bounded by the vertex pair it walks between, which the canvas
// extremes bound in turn, so none is written on the destination row either.
//
// A span of S <= 0 is not drawn at all — no pixel, no clamp, no fill
// colour — and that test is also what keeps srcStep's division defined.
//
// The dirt overlay is read at the terrain pixel's OWN (i, srcRow), so a
// decoded overlay cannot shear against the tile it composites with; the
// level, where there is one, is InterpSpan across the cell's 32 columns and
// down the column's own S rows, so the ramp lives inside the pixels actually
// drawn.
func drawCellSpans(dst *image.RGBA, originX, offsetY, scale int, q cellQuad, src, dirt *image.Paletted, sh *cellShading) {
	sb := src.Bounds()
	for i := 0; i < CellSize; i++ {
		top, s := q.span(i)
		if s <= 0 {
			continue
		}
		srcStep := (CellSize << srcFracBits) / s
		x := originX + i*scale
		for j := 0; j < s; j++ {
			srcRow := (j * srcStep) >> srcFracBits

			c := paletteColor(src, sb.Min.X+i, sb.Min.Y+srcRow)
			if dirt != nil {
				db := dirt.Bounds()
				c = overlayPixel(c, dirt, db.Min.X+i, db.Min.Y+srcRow)
			}
			if sh != nil {
				c = ShadeRGBA(c, sh.tint, InterpSpan(sh.l00, sh.l10, sh.l01, sh.l11, i, j, CellSize, s))
			}

			y := offsetY + (top+j)*scale
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dst.SetRGBA(x+sx, y+sy, c)
				}
			}
		}
	}
}

// fillCellSpans paints a sloped cell's 32 spans a single colour: the
// placeholder path, which keeps its cell's geometry and stays unlit. It is
// the sloped counterpart of fillCell, and takes the same translated origin
// drawCellSpansLit does.
//
// The dropped column is dropped here too: a span of S <= 0 leaves the column
// untouched, so a placeholder never marks a pixel the tile it replaces would
// not have covered.
func fillCellSpans(dst *image.RGBA, originX, offsetY, scale int, q cellQuad, c color.RGBA) {
	for i := 0; i < CellSize; i++ {
		top, s := q.span(i)
		if s <= 0 {
			continue
		}
		x := originX + i*scale
		for j := 0; j < s; j++ {
			y := offsetY + (top+j)*scale
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dst.SetRGBA(x+sx, y+sy, c)
				}
			}
		}
	}
}
