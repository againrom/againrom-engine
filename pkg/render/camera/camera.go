// Package camera is the map viewer's camera model: where the view sits over the
// world, how it pans and zooms, and which tiles it covers.
//
// The model is the project's own design, not a reproduction of any
// documented original-engine camera. It is deliberately free of the
// windowing engine and of every formats package so the arithmetic is
// unit-testable without opening a window.
package camera

import "math"

// CellSize is the native edge of one terrain cell in world pixels. It matches
// the 32x32 sub-cell the terrain tileset draws (pkg/render/terrain).
const CellSize = 32

// Zoom limits. Zoom is screen pixels per world pixel, so 1.0 draws a cell at its
// native 32x32 and the limits bound how far the view may scale in each direction.
const (
	ZoomMin = 0.125
	ZoomMax = 8.0
)

// Camera views a Cols x Rows tile world through a ViewW x ViewH pixel window.
//
// X and Y are the world-pixel coordinate of the view's top-left corner. They are
// world pixels, not screen pixels, so a pan of n moves the same distance across
// the map at any zoom.
//
// The world's HEIGHT is not derived from the tile grid: it is a value the
// camera is told, and Cols/Rows stay the tile grid alone — what
// VisibleTiles clips against and what a caller indexes a map with. The two
// coincided while every terrain was flat; a height-displaced terrain's world
// is as tall as its vertices span, which is neither Rows*CellSize nor
// anything this package could compute, since it may depend on no projection
// (the import DAG).
type Camera struct {
	Cols, Rows   int     // world size in tiles
	ViewW, ViewH int     // window size in screen pixels
	X, Y         float64 // world-pixel coordinate of the view's top-left corner
	Zoom         float64 // screen pixels per world pixel
	minimumZoom  float64 // optional per-camera floor; zero keeps ZoomMin

	// worldH is the world's height in world pixels, read by WorldH and
	// written by SetWorldHeight. It is unexported so that every write goes
	// through a mutator that re-clamps, which is the invariant the exported
	// position fields carry a Clamp call to restore instead.
	worldH float64

	// clampRect is an optional drawable interior inside the complete world.
	// Tile indexing and culling still use Cols/Rows and WorldW/WorldH; only the
	// position invariant uses this rectangle. That lets a renderer keep an
	// engine-owned black margin in the map data without treating that margin as
	// useful camera room.
	clampRect [4]float64 // minX, minY, maxX, maxY
	clampSet  bool

	// verticalClamp optionally refines clampRect's Y span for the horizontal
	// world interval the view actually covers. Height-displaced terrain has a
	// piecewise edge: a hill outside the visible columns must not crop the
	// columns the player is looking at. The function is installed by the tier
	// that owns that projection; camera remains geometry-agnostic.
	verticalClamp func(viewMinX, viewMaxX float64) (minY, maxY float64)
}

// New returns a camera over a Cols x Rows world at native zoom, positioned at the
// world origin and clamped into range. The world's height starts at the tile
// grid's own Rows*CellSize, which is what a flat terrain spans; a caller drawing
// a displaced one calls SetWorldHeight.
func New(cols, rows, viewW, viewH int) *Camera {
	// The initialiser is the expression WorldH used to evaluate, verbatim: no
	// flat construction may drift, whatever it was handed (0013 SC-1). The
	// normalisation SetWorldHeight applies is a mutator's guard on an
	// arbitrary caller value, and deliberately not a second rule here.
	c := &Camera{Cols: cols, Rows: rows, ViewW: viewW, ViewH: viewH, Zoom: 1,
		worldH: float64(rows) * CellSize}
	c.Clamp()
	return c
}

// WorldW and WorldH are the world extent in world pixels.
func (c *Camera) WorldW() float64 { return float64(c.Cols) * CellSize }

// WorldH returns the world's height in world pixels: New's Rows*CellSize until
// SetWorldHeight replaces it.
func (c *Camera) WorldH() float64 { return c.worldH }

// SetWorldHeight sets the world's height in world pixels and re-clamps, like
// every other mutator. It is how a displaced terrain's extent reaches the
// camera without the camera knowing what a projection is.
//
// The new height can be SHORTER than the flat Rows*CellSize as well as taller —
// a grid whose altitudes span less than a cell per row produces exactly that —
// so this can move an axis from clamping to centring, and does so at once rather
// than at the next pan.
//
// A negative or non-finite height is taken as zero, which is the world a
// zero-row grid already produces: an axis smaller than any view, so it centres.
// That is what keeps the clamp total instead of letting a NaN reach the position.
func (c *Camera) SetWorldHeight(h float64) {
	if math.IsNaN(h) || math.IsInf(h, 0) || h < 0 {
		h = 0
	}
	c.worldH = h
	c.Clamp()
}

// SetClampBounds confines camera motion to the drawable world rectangle
// [minX,maxX] x [minY,maxY]. The complete world extents remain unchanged for
// tile lookup, minimap projection and culling.
//
// When one drawable axis is smaller than the current view, it is centred in
// the view. This is the only case in which the camera deliberately exposes
// space outside the drawable rectangle: no translation could fill that axis.
// Invalid or inverted bounds restore the complete-world clamp.
func (c *Camera) SetClampBounds(minX, minY, maxX, maxY float64) {
	values := [...]float64{minX, minY, maxX, maxY}
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			c.ClearClampBounds()
			return
		}
	}
	if maxX < minX || maxY < minY {
		c.ClearClampBounds()
		return
	}
	c.clampRect = values
	c.clampSet = true
	c.verticalClamp = nil
	c.Clamp()
}

// SetAdaptiveClampBounds is SetClampBounds with a vertical span resolved from
// the horizontal world interval currently visible. The supplied rectangle is
// both the horizontal bound and the fallback when vertical returns a non-finite
// or inverted span.
//
// Clamp resolves X first, then calls vertical with [X,X+viewWidth]. A pan or
// zoom therefore reaches the correct local top and bottom in the same mutation;
// it never waits for a viewer frame to replace a stale global bound.
func (c *Camera) SetAdaptiveClampBounds(minX, minY, maxX, maxY float64,
	vertical func(viewMinX, viewMaxX float64) (minY, maxY float64),
) {
	values := [...]float64{minX, minY, maxX, maxY}
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			c.ClearClampBounds()
			return
		}
	}
	if maxX < minX || maxY < minY || vertical == nil {
		c.SetClampBounds(minX, minY, maxX, maxY)
		return
	}
	c.clampRect = values
	c.clampSet = true
	c.verticalClamp = vertical
	c.Clamp()
}

// ClearClampBounds restores clamping against the complete world.
func (c *Camera) ClearClampBounds() {
	c.clampRect = [4]float64{}
	c.clampSet = false
	c.verticalClamp = nil
	c.Clamp()
}

// viewWorld returns the view's extent measured in world pixels. A view of 800
// screen px at zoom 2 covers only 400 world px.
func (c *Camera) viewWorld() (w, h float64) {
	z := c.zoom()
	return float64(c.ViewW) / z, float64(c.ViewH) / z
}

// zoom returns the zoom clamped into its legal range, treating a zero or
// non-finite value as native. Reading through this keeps a caller that assigned
// Zoom directly from producing a division by zero or a NaN position.
func (c *Camera) zoom() float64 {
	z := c.Zoom
	if math.IsNaN(z) || z <= 0 {
		z = 1
	}
	return math.Min(math.Max(z, c.zoomFloor()), ZoomMax)
}

func (c *Camera) zoomFloor() float64 {
	if c.minimumZoom > 0 {
		return c.minimumZoom
	}
	return ZoomMin
}

// SetMinimumZoom changes this camera's floor only. The map inspector can fit
// large authored maps into its narrow canvas without changing game/mapview
// defaults. Invalid values restore ZoomMin; the maximum stays ZoomMax.
func (c *Camera) SetMinimumZoom(z float64) {
	if math.IsNaN(z) || math.IsInf(z, 0) || z <= 0 || z > ZoomMax {
		z = 0
	}
	c.minimumZoom = z
	c.Clamp()
}

// SetZoom clamps to the camera's floor (ZoomMin by default) and ZoomMax.
func (c *Camera) SetZoom(z float64) {
	c.Zoom = math.Min(math.Max(z, c.zoomFloor()), ZoomMax)
	c.Clamp()
}

// Clamp confines the view to its drawable bounds, or to the complete world
// when no narrower bounds were supplied. On an axis larger than the view, the
// offset is held between the two drawable edges; on an axis smaller than the
// view, the drawable span is centred instead.
//
// Every mutator ends with Clamp, so no sequence of pans and zooms can leave
// the camera outside these bounds.
func (c *Camera) Clamp() {
	c.Zoom = c.zoom()
	vw, vh := c.viewWorld()
	minX, minY, maxX, maxY := 0.0, 0.0, c.WorldW(), c.WorldH()
	if c.clampSet {
		minX, minY, maxX, maxY = c.clampRect[0], c.clampRect[1], c.clampRect[2], c.clampRect[3]
	}
	c.X = clampAxisBounds(c.X, minX, maxX, vw)
	if c.verticalClamp != nil {
		resolvedMin, resolvedMax := c.verticalClamp(c.X, c.X+vw)
		if !math.IsNaN(resolvedMin) && !math.IsInf(resolvedMin, 0) &&
			!math.IsNaN(resolvedMax) && !math.IsInf(resolvedMax, 0) &&
			resolvedMax >= resolvedMin {
			minY, maxY = resolvedMin, resolvedMax
		}
	}
	c.Y = clampAxisBounds(c.Y, minY, maxY, vh)
}

// clampAxis holds pos within [0, world-view], or centers the axis when the world
// is not wide enough to fill the view.
func clampAxis(pos, world, view float64) float64 {
	return clampAxisBounds(pos, 0, world, view)
}

// clampAxisBounds is clampAxis over an arbitrary half-open drawable span.
func clampAxisBounds(pos, lo, hi, view float64) float64 {
	if math.IsNaN(pos) {
		pos = lo
	}
	if hi-lo <= view {
		return (lo + hi - view) / 2
	}
	return math.Min(math.Max(pos, lo), hi-view)
}

// Pan moves the view by a world-pixel delta and re-clamps. The delta is in world
// pixels so edge-scroll speed does not change with zoom.
func (c *Camera) Pan(dx, dy float64) {
	if math.IsNaN(dx) {
		dx = 0
	}
	if math.IsNaN(dy) {
		dy = 0
	}
	c.X += dx
	c.Y += dy
	c.Clamp()
}

// CenterOn puts the world point (wx, wy) at the centre of the view and
// re-clamps. It is the counterpart of Pan: a pan says how far to move, this says
// where to arrive.
//
// The point arrives in WORLD pixels and this package converts nothing on the way
// in — a caller holding a cell, a tile or an entity resolves it to a world point
// first, in the tier that knows what a cell means. What is subtracted here is
// the view's own extent in world pixels, which is the view size divided by the
// zoom and therefore neither the view size nor a constant.
//
// IT ENDS IN Clamp LIKE EVERY OTHER MUTATOR, and that is the whole of what
// makes an arbitrary point safe: a point near a world edge yields a position
// the clamp pulls back to the edge, so the centring is honoured where the
// world allows it and the bound wins where it does not. On an axis whose
// world is smaller than the view, the clamp centres the WORLD rather than
// the point, and the point is then off-centre by exactly the world's own
// offset — which is the axis's defined behaviour and not a failure of this
// method.
//
// A non-finite coordinate reaches clampAxis, which answers zero for a NaN;
// nothing here compares against one first, for the reason ScreenToCell states
// its range test positively.
func (c *Camera) CenterOn(wx, wy float64) {
	vw, vh := c.viewWorld()
	c.X = wx - vw/2
	c.Y = wy - vh/2
	c.Clamp()
}

// ZoomAbout changes the zoom by factor while keeping the world point currently
// under the screen point (sx, sy) under it afterwards, then re-clamps.
//
// Clamping the position afterwards can pull the anchor off the cursor when the
// view has hit a world edge; that is the clamp winning on purpose, and the
// preservation property is asserted away from the edges (AC-3).
func (c *Camera) ZoomAbout(sx, sy, factor float64) {
	if factor <= 0 || math.IsNaN(factor) {
		return
	}
	wx, wy := c.ScreenToWorld(sx, sy)

	c.Zoom = math.Min(math.Max(c.zoom()*factor, c.zoomFloor()), ZoomMax)

	// Re-anchor: solve X so that (wx, wy) lands back on (sx, sy) at the new zoom.
	z := c.zoom()
	c.X = wx - sx/z
	c.Y = wy - sy/z
	c.Clamp()
}

// ScreenToWorld converts a screen-pixel point in the view to a world-pixel point.
func (c *Camera) ScreenToWorld(sx, sy float64) (wx, wy float64) {
	z := c.zoom()
	return c.X + sx/z, c.Y + sy/z
}

// WorldToScreen converts a world-pixel point to a screen-pixel point in the view.
// It is the inverse of ScreenToWorld on the view's image.
func (c *Camera) WorldToScreen(wx, wy float64) (sx, sy float64) {
	z := c.zoom()
	return (wx - c.X) * z, (wy - c.Y) * z
}

// ScreenToCell resolves a screen-pixel point in the view to the tile under
// it: ScreenToWorld, each axis floored by CellSize, then tested against the
// tile grid. inside reports whether the grid has that cell.
//
// It answers about the FLAT ground, in a height-displaced world exactly as in a
// flat one: no vertical origin is subtracted and no height is inverted, because
// the projection's origin already cancels a typical cell's own height, so a
// shift would be wrong by the whole relief on a uniformly raised map. The error
// that leaves is owned and disclosed: exact on flat ground and on uniform
// relief, off by at most the altitude spread in rows otherwise (0028 C-1).
//
// SINCE DIV-044 IT ANSWERS THE FLAT LATTICE ONLY. pkg/ui's displaced ground
// picker searches the projected corner mesh, and its entity picker searches
// the entity's placed rectangle. This method remains the no-altitude fallback
// and the shared flat-lattice primitive used by camera tests and diagnostics.
// It contains no projection and therefore cannot answer either displaced
// question by itself.
//
// A point beyond the grid is REPORTED, never moved inside — there is no clamp on
// this path, which is what separates it from VisibleTiles, whose whole job is to
// clip. When inside is false, col and row are zero and mean nothing: they are
// neither the cell resolved nor that cell clamped into range, so the bool is
// read first or the answer is not read at all (0028 AC-1, AC-10).
//
// The floor is math.Floor and the range test runs in float64, both before any
// conversion to int. So a world point just below zero cannot truncate back into
// cell 0, and a non-finite one fails the test instead of reaching a conversion
// whose result the language leaves to the implementation. The test is stated
// positively for the same reason: every comparison against a NaN is false, so a
// NaN lands outside rather than slipping past a negated one.
func (c *Camera) ScreenToCell(sx, sy float64) (col, row int, inside bool) {
	wx, wy := c.ScreenToWorld(sx, sy)
	fc := math.Floor(wx / CellSize)
	fr := math.Floor(wy / CellSize)
	if fc >= 0 && fc < float64(c.Cols) && fr >= 0 && fr < float64(c.Rows) {
		return int(fc), int(fr), true
	}
	return 0, 0, false
}

// ScreenToCellRange resolves a screen-pixel RECTANGLE — given as two opposite
// corners, in either order — to the tiles it meets, clipped to the tile grid.
// ok reports whether it meets any.
//
// IT IS NO LONGER THE BOX-SELECTION PATH, and saying so is the point of this
// paragraph. On relief the two are apart by exactly that lift, so a box
// drawn around a unit on a mountain selected a different one about two cells
// away. A unit is now picked by the rectangle it is DRAWN on (pkg/ui), and
// no production caller reaches this method; the tests that build a band of
// cells from a screen rectangle still do.
//
// What it is, and remains exactly right about, is the FLAT LATTICE — the same
// question VisibleTiles and ScreenToCell answer, over a rectangle. It carries no
// height term at all, so there is nothing here to fall out of step with a
// projection.
//
// It is the band generalisation of ScreenToCell and shares that method's whole
// answer: each corner through ScreenToWorld, each axis floored by CellSize, the
// flat ground in a displaced world exactly as in a flat one, and the same
// disclosed relief error — now on EVERY edge of the rectangle rather than on one
// point.
//
// THE MAX END IS RAISED BY ONE, which is what makes the range half-open over
// cells the rectangle MEETS rather than cells its corners land on. A rectangle
// of zero width or height — a horizontal drag, a vertical one, or a press and a
// release at one point — is therefore one cell wide on that axis and never
// empty, which is the whole reason the +1 is here and not at a caller.
//
// NORMALISATION HAPPENS BEFORE THE CLIP, so orientation-independence is one
// expression's property: a rectangle dragged right-to-left or bottom-to-top
// yields exactly the range its corner-swapped twin yields, and no caller has to
// order the points it hands over.
//
// A rectangle wholly outside the grid comes back ok == false with a zero range,
// on ScreenToCell's own rule: nothing is promised about the range then, and it
// is neither the range resolved nor that range clamped into view. One that
// straddles an edge is CLIPPED and reported found — a drag half off the map
// selects what it caught, which is a normal outcome and not a refusal.
//
// A non-finite corner yields no range at all rather than reaching a conversion
// whose result the language leaves to the implementation. The finiteness test
// runs on the floored values, before any clamp, for the reason ScreenToCell
// states its range test positively: every comparison against a NaN is false, so
// a NaN must be answered rather than compared past.
func (c *Camera) ScreenToCellRange(x0, y0, x1, y1 float64) (TileRange, bool) {
	wx0, wy0 := c.ScreenToWorld(x0, y0)
	wx1, wy1 := c.ScreenToWorld(x1, y1)

	fc0, fc1 := math.Floor(wx0/CellSize), math.Floor(wx1/CellSize)
	fr0, fr1 := math.Floor(wy0/CellSize), math.Floor(wy1/CellSize)
	if math.IsNaN(fc0) || math.IsNaN(fc1) || math.IsNaN(fr0) || math.IsNaN(fr1) {
		return TileRange{}, false
	}
	if fc1 < fc0 {
		fc0, fc1 = fc1, fc0
	}
	if fr1 < fr0 {
		fr0, fr1 = fr1, fr0
	}

	r := TileRange{
		Col0: clipCell(fc0, c.Cols),
		Row0: clipCell(fr0, c.Rows),
		Col1: clipCell(fc1+1, c.Cols),
		Row1: clipCell(fr1+1, c.Rows),
	}
	if r.Empty() {
		return TileRange{}, false
	}
	return r, true
}

// clipCell holds a floored cell index in [0, n] in float64 — so an infinite or
// enormous coordinate meets the bound before it meets an int conversion — and
// only then narrows.
func clipCell(f float64, n int) int {
	return int(math.Min(math.Max(f, 0), float64(n)))
}

// TileRange is the half-open range of tiles [Col0, Col1) x [Row0, Row1) that a
// view covers, already clipped to the world. An empty range has Col0 == Col1.
type TileRange struct {
	Col0, Row0 int
	Col1, Row1 int
}

// Empty reports whether the range covers no tiles.
func (r TileRange) Empty() bool { return r.Col0 >= r.Col1 || r.Row0 >= r.Row1 }

// VisibleTiles returns exactly the tiles intersecting the view, clipped to
// the world (AC-4). Clipping only shrinks the range, so a caller iterating
// it can never index a cell outside the grid.
func (c *Camera) VisibleTiles() TileRange {
	vw, vh := c.viewWorld()
	return TileRange{
		Col0: clampInt(int(math.Floor(c.X/CellSize)), 0, c.Cols),
		Row0: clampInt(int(math.Floor(c.Y/CellSize)), 0, c.Rows),
		Col1: clampInt(ceilDiv(c.X+vw), 0, c.Cols),
		Row1: clampInt(ceilDiv(c.Y+vh), 0, c.Rows),
	}
}

// ceilDiv returns ceil(v/CellSize) guarding the non-finite case, which would
// otherwise convert to an implementation-defined int.
func ceilDiv(v float64) int {
	if math.IsNaN(v) {
		return 0
	}
	q := math.Ceil(v / CellSize)
	if q > math.MaxInt32 {
		return math.MaxInt32
	}
	if q < math.MinInt32 {
		return math.MinInt32
	}
	return int(q)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
