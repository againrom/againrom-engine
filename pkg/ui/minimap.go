package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/terrain"
)

// DIV-1281, AI-MINIMAP-062

var minimapBox = image.Pt(200, 200)

// The minimap's own authored colours (spec D-4).
//
// minimapFill and minimapBorder echo AuthoredReadoutLayout's own palette
// (readout.go) rather than inventing a third one, so every authored box in
// this window reads as one family.
//
// minimapLocalColor and minimapOtherColor are deliberately far apart in
// hue — green against red — so the one dot that means "you" is legible even
// as a single pixel, which is the smallest a mark can be drawn at (plan
// R-3's sampled mode, over a large map).
var (
	minimapFill   = color.RGBA{R: 0x10, G: 0x12, B: 0x18, A: 0xff}
	minimapBorder = color.RGBA{R: 0x8a, G: 0x74, B: 0x46, A: 0xff}

	minimapLocalColor = color.RGBA{R: 0x40, G: 0xff, B: 0x40, A: 0xff}
	minimapOtherColor = color.RGBA{R: 0xff, G: 0x40, B: 0x40, A: 0xff}

	minimapUnseenColor = color.RGBA{A: 0xff}

	// minimapViewColor outlines what the camera can see (0140, owner: "a
	// green square showing how much of the camera we see"). It is a PALER
	// green than minimapLocalColor above rather than a different hue, because
	// the owner asked for green and the two are told apart by SHAPE — the
	// view is a one-pixel outline, a unit is a filled block — which stays
	// true at every scale this file computes, including the sampled one where
	// a unit is a single pixel.
	minimapViewColor = color.RGBA{R: 0xa8, G: 0xff, B: 0xa8, A: 0xff}
)

type minimapMark struct {
	Cell  image.Point
	Local bool
}

func composeMinimap(colours []color.RGBA, cols, rows int, fog func(col, row int) uint8, marks []minimapMark, view image.Rectangle, box image.Point) *image.RGBA {
	return composeMinimapLayer(colours, cols, rows, fog, marks, view, box, false)
}

func composeMinimapLayer(colours []color.RGBA, cols, rows int, fog func(col, row int) uint8, marks []minimapMark, view image.Rectangle, box image.Point, crystal bool) *image.RGBA {
	if cols <= 0 || rows <= 0 || box.X <= 0 || box.Y <= 0 {
		return image.NewRGBA(image.Rectangle{})
	}

	num, den := minimapScale(cols, rows, box)
	cellColor := func(col, row int) color.RGBA {
		if crystal && fog != nil && fog(col, row) == FogUnseen {
			return color.RGBA{}
		}
		return minimapCellColor(colours, cols, col, row, fog)
	}

	size := minimapPixelExtent(cols, rows, num, den)
	img := image.NewRGBA(image.Rectangle{Max: size})
	if num >= den {
		// Per-cell pixel bounds, not a uniform block: minimapCellPixel(cols, ..)
		// lands exactly on the box's own extent on the constraining axis (R-3,
		// owner), and every cell's own [start, end) width is at most one pixel
		// away from every other's, which is the same guarantee a uniform block
		// gave when num was a multiple of den and simply continues to hold when
		// it is not.
		for row := 0; row < rows; row++ {
			y0, y1 := minimapCellPixel(row, num, den), minimapCellPixel(row+1, num, den)
			for col := 0; col < cols; col++ {
				x0, x1 := minimapCellPixel(col, num, den), minimapCellPixel(col+1, num, den)
				fillBlock(img, x0, y0, x1-x0, y1-y0, cellColor(col, row))
			}
		}
	} else {
		// Sample each output pixel through the same rational scale the geometry
		// and click path use. An integer source-cell step can leave both axes
		// short of the available box when either map dimension exceeds it.
		for y := 0; y < size.Y; y++ {
			row := min(rows-1, y*den/num)
			for x := 0; x < size.X; x++ {
				col := min(cols-1, x*den/num)
				img.SetRGBA(x, y, cellColor(col, row))
			}
		}
	}

	b := img.Bounds()
	outAt := func(col, row int) (int, int) {
		return min(b.Max.X-1, minimapCellPixel(col, num, den)), min(b.Max.Y-1, minimapCellPixel(row, num, den))
	}
	outEnd := func(col, row int) (int, int) {
		if num >= den {
			return minimapCellPixel(col, num, den), minimapCellPixel(row, num, den)
		}
		return min(b.Max.X, ceilDiv(col*num, den)), min(b.Max.Y, ceilDiv(row*num, den))
	}
	for _, m := range marks {
		r := minimapMarkRect(m.Cell, cols, rows, num, den, b)
		if r.Empty() {
			continue
		}
		c := minimapOtherColor
		if m.Local {
			c = minimapLocalColor
		}
		fillBlock(img, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), c)
	}

	drawMinimapView(img, view, outAt, outEnd)
	return img
}

// minimapMarkRect is the pixel rectangle, inside a composed picture of bounds
// b, that a mark on cell fills: the cell's own block when a cell spans at
// least one pixel, otherwise the one sampled pixel naming it. A cell outside
// the grid or the picture has an empty rectangle.
func minimapMarkRect(cell image.Point, cols, rows, num, den int, b image.Rectangle) image.Rectangle {
	if cell.X < 0 || cell.Y < 0 || cell.X >= cols || cell.Y >= rows {
		return image.Rectangle{}
	}
	x, y := min(b.Max.X-1, minimapCellPixel(cell.X, num, den)), min(b.Max.Y-1, minimapCellPixel(cell.Y, num, den))
	if x < b.Min.X || y < b.Min.Y || x >= b.Max.X || y >= b.Max.Y {
		return image.Rectangle{}
	}
	if num >= den {
		return image.Rect(x, y, minimapCellPixel(cell.X+1, num, den), minimapCellPixel(cell.Y+1, num, den)).Intersect(b)
	}
	return image.Rect(x, y, x+1, y+1)
}

// drawMinimapView outlines the cell range the camera can currently see (0140,
// owner). view is in CELLS, Min inclusive and Max exclusive — image.Rectangle's
// own convention — and outAt/outEnd map a cell to its output pixel and to the
// exclusive bound of one.
//
// IT IS AN OUTLINE AND NOT A FILL, which is the whole of why it can be drawn
// LAST, over the marks: a filled rectangle at any alpha would either hide the
// terrain it stands over or need blending this file has no other reason to
// carry, where four one-pixel edges hide four lines of it and nothing else.
// Drawn last is deliberate — the box says where the player is looking, and a
// unit dot standing on its edge must not be able to erase the corner of it.
//
// IT IS NEVER THINNER THAN ONE PIXEL AND NEVER LARGER THAN THE PICTURE. A view
// covering fewer cells than one output pixel represents still yields a 1x1
// mark rather than nothing, because a rectangle that vanishes at some zooms
// reads as a bug and not as a scale. A view covering the whole map yields edges
// on the picture's own border, which fillBlock clips exactly as it clips
// everything else.
//
// The lower and upper edges round in opposite directions when the picture is
// smaller than the source grid. One visible cell still produces a one-pixel
// outline, including at the far edge of a short axis.
//
// AN EMPTY OR INVERTED RANGE DRAWS NOTHING. camera.VisibleTiles clips to the
// world and can return an empty range; a caller handing one in is saying the
// camera sees no cell, and no cell is what gets outlined.
func drawMinimapView(img *image.RGBA, view image.Rectangle, outAt, outEnd func(col, row int) (int, int)) {
	if view.Empty() {
		return
	}
	x0, y0 := outAt(view.Min.X, view.Min.Y)
	x1, y1 := outEnd(view.Max.X, view.Max.Y)
	if x1 <= x0 {
		x1 = x0 + 1
	}
	if y1 <= y0 {
		y1 = y0 + 1
	}
	w, h := x1-x0, y1-y0

	c := minimapViewColor
	fillBlock(img, x0, y0, w, 1, c)   // top
	fillBlock(img, x0, y1-1, w, 1, c) // bottom
	fillBlock(img, x0, y0, 1, h, c)   // left
	fillBlock(img, x1-1, y0, 1, h, c) // right
}

// DIV-1281
func minimapScale(cols, rows int, box image.Point) (num, den int) {
	if cols <= 0 || rows <= 0 || box.X <= 0 || box.Y <= 0 {
		return 0, 0
	}
	num, den = box.X, cols
	if int64(box.Y)*int64(cols) < int64(box.X)*int64(rows) {
		num, den = box.Y, rows
	}
	return num, den
}

func minimapPixelExtent(cols, rows, num, den int) image.Point {
	return image.Pt(max(1, minimapCellPixel(cols, num, den)), max(1, minimapCellPixel(rows, num, den)))
}

// minimapCellPixel is minimapScale's own boundary rule (den > 0): the output
// pixel where cell index at begins. The same multiply-then-floor-divide rule
// keeps the limiting axis exact at both ends without accumulated rounding.
func minimapCellPixel(at, num, den int) int {
	if den <= 0 {
		return 0
	}
	return at * num / den
}

func minimapCellColor(colours []color.RGBA, cols, col, row int, fog func(col, row int) uint8) color.RGBA {
	var base color.RGBA
	if i := row*cols + col; i >= 0 && i < len(colours) {
		base = colours[i]
	}
	switch fog(col, row) {
	case FogVisible:
		return base
	case FogExplored:
		return color.RGBA{R: base.R / 2, G: base.G / 2, B: base.B / 2, A: base.A}
	default: // FogUnseen, and any value none of the plane's writers can produce
		return minimapUnseenColor
	}
}

// fillBlock paints a w x h solid rectangle at (x,y), clipped to img's own
// bounds — the terrain fill and the mark fill in composeMinimap share it
// rather than each writing its own nested loop.
func fillBlock(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	b := img.Bounds()
	for dy := 0; dy < h; dy++ {
		py := y + dy
		if py < b.Min.Y || py >= b.Max.Y {
			continue
		}
		for dx := 0; dx < w; dx++ {
			px := x + dx
			if px < b.Min.X || px >= b.Max.X {
				continue
			}
			img.SetRGBA(px, py, c)
		}
	}
}

// ceilDiv gives the exclusive pixel boundary of a downscaled cell range.
func ceilDiv(a, b int) int {
	if b <= 0 {
		return a
	}
	return (a + b - 1) / b
}

// minimapTerrainColours is the cached terrain-colour build: one colour per
// map cell, row-major, sampled ONCE from the tileset through the same
// resolveCell -> Slot -> SubCell chain cornerScales already uses
// (viewer.go), and kept for the viewer's whole life. NOTHING INVALIDATES IT,
// because nothing can: the grid and the tileset arrive together at
// construction (NewViewerWithStatics) and neither has a setter that replaces
// either afterwards, so there is no event this cache could be stale across.
//
// THE SAMPLE IS ONE PIXEL, the sub-cell's own centre, not an average of the
// whole 32x32 block: cornerScales' own four-corner reads make the same
// trade for shading (viewer.go), and a minimap pixel is smaller than the
// source cell under any scale this file computes, so a second colour
// averaged from the whole block would cost a 1024-pixel walk per cell for a
// difference nothing at minimap scale could show.
//
// A CELL WHOSE SUB-CELL IS NIL — an off-corpus tile word, or (every test in
// this package that builds a viewer over &terrain.Tileset{}) a tileset with
// nothing loaded at all — FALLS BACK TO terrain.PlaceholderColor, the same
// magenta the terrain draw path itself fills an unresolved cell with
// (pkg/render/terrain/composite.go). Reusing it rather than inventing a
// second "we don't know" colour is deliberate: the minimap's placeholder
// and the ground's placeholder name the same condition and should look
// like they do.
func (v *Viewer) minimapTerrainColours() []color.RGBA {
	if v.minimapColours != nil {
		return v.minimapColours
	}
	cols, rows := v.grid.Width, v.grid.Height
	out := make([]color.RGBA, cols*rows)
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			ref := v.resolveCell(v.tileWord(col, row), col, row)
			img := v.set.Slot(ref.Slot).SubCell(ref.Sub)
			c := terrain.PlaceholderColor
			if img != nil {
				b := img.Bounds()
				c = paletteRGBA(img, b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)
			}
			out[row*cols+col] = c
		}
	}
	v.minimapColours = out
	return out
}

// minimapMarks is one mark per unit the minimap shows. A fallen unit keeps its
// owner's mark only while Heal can still raise it (DIV-1455); a body past that
// floor is not marked, so a local one does not stay green under fog until the
// world removes it.
func (v *Viewer) minimapMarks() []minimapMark {
	var out []minimapMark
	for _, e := range v.entities {
		if e.Cell.X < 0 || e.Cell.Y < 0 || e.Cell.X >= v.grid.Width || e.Cell.Y >= v.grid.Height {
			continue
		}
		if e.Life != LifeAlive && !e.Restorable {
			continue
		}
		if !v.fogGateEntity(e.Owner, e.Cell.X, e.Cell.Y) {
			continue
		}
		out = append(out, minimapMark{Cell: e.Cell, Local: e.Owner == v.localOwner})
	}
	return out
}

// authoredMinimapCorner is where the minimap docks: TOP RIGHT, the one
// corner of the window's four that neither the readout (top left,
// readout.go's AuthoredReadoutLayout: "THE CORNER IS THE OTHER ONE") nor the
// unit panel (bottom left, panel.go's own default) already occupies. Bottom
// right is free too; nothing but this project's own choice picks between the
// two, and top right is taken because it is the readout's own diagonal
// opposite, leaving the two informational boxes (readout, this one) on one
// diagonal and the interactive one (the unit panel) alone on the other. That
// choice happens to agree with the decoded right column (`SESS-VIEW-028`)
// being on the same side, which is a coincidence and not a reading of any
// claim: nothing published names a corner for id 5.
//
// THE MARGIN IS (0, 0), NOT (12, 12), SINCE ROUND 2. The mission's own right
// column runs flush to the frame's top and right edges (hud.go's
// `rightColumnBox`, this file's header); a corner offset here put the
// minimap 12 pixels inside both where `SHOP-FIGURE-041`'s id 5 has nothing
// between its own rect and either edge. `panelOrigin` with `PanelTopRight`
// and a zero margin lands the box's right edge on `area.X` and its top edge
// on 0, which is what flush means for this corner.
//
// It is a package-level value and not rebuilt per frame, mirroring
// AuthoredReadoutLayout's own field values without that function's reason
// to hand out a fresh copy each call: nothing here is a row slice another
// viewer could mutate through, so there is nothing a shared value could
// leak.
var authoredMinimapCorner = PanelLayout{Corner: PanelTopRight, Margin: image.Pt(0, 0)}

// minimapPresent is the picture to draw this frame and where its top-left
// corner goes, or false for a frame that draws no minimap at all —
// readoutPresent's own shape (readout.go), minus the font gate: the
// minimap needs no glyph, so a viewer holding no font still shows one.
//
// THE BORDER IS PAINTED HERE, ONE PIXEL BEYOND composeMinimap's OWN
// BOUNDS, and deliberately NOT inside that function: composeMinimap's
// pixels are exactly what the fog and scale tests assert on, cell by cell,
// and a border painted over its outermost row and column would make pixel
// (0,0) sometimes a fog answer and sometimes a border colour depending on
// which test asked. Framing it out here keeps that function's contract to
// "cells and marks, nothing else" while still giving the box the same
// bordered look the readout and the panel have, reusing fillPanelFrame
// (panel.go) rather than re-deriving it.
func (v *Viewer) minimapPresent() (*image.RGBA, image.Point, bool) {
	g, ok := v.minimapGeometry()
	if !ok {
		return nil, image.Point{}, false
	}
	colours, fog, marks, view := v.minimapWindowed(g)
	content := composeMinimapLayer(colours, g.Cols, g.Rows, fog, marks,
		view, g.extent, v.dialogFrame != nil && v.dialogFrame.Minimap != nil)
	cb := content.Bounds()
	if cb.Empty() {
		return nil, image.Point{}, false
	}

	box := g.Box.Size()
	seam := 0
	if v.dialogFrame != nil && v.dialogFrame.Minimap != nil && v.dialogFrame.MinimapSeam != nil {
		seam = v.dialogFrame.MinimapSeam.Bounds().Dx()
	}
	pic := image.NewRGBA(image.Rect(0, 0, box.X+seam, box.Y))
	if v.dialogFrame != nil && v.dialogFrame.Minimap != nil {
		if seam > 0 {
			copyNativeKeyed(pic, v.dialogFrame.MinimapSeam, image.Point{}, pic.Bounds())
		}
		copyNativeKeyed(pic, v.dialogFrame.Minimap, image.Pt(seam, 0), pic.Bounds())
	} else {
		fillPanelFrame(pic, box, minimapFill, minimapBorder)
	}
	// The content's offset INSIDE the box, which is g.Content taken back to the
	// box's own origin: one pixel down from the border, and horizontally
	// centred in whatever the frame's width came out as.
	off := g.Content.Min.Sub(g.Box.Min).Add(image.Pt(seam, 0))
	for y := 0; y < cb.Dy(); y++ {
		for x := 0; x < cb.Dx(); x++ {
			c := content.RGBAAt(cb.Min.X+x, cb.Min.Y+y)
			if c.A != 0 {
				pic.SetRGBA(off.X+x, off.Y+y, c)
			}
		}
	}
	return pic, g.Box.Min.Sub(image.Pt(seam, 0)), true
}

// minimapWindow is the cells the minimap draws: the walkable area, without the
// engine's blocked outer margin. A map with no margin draws whole.
func (v *Viewer) minimapWindow() image.Rectangle {
	if r, ok := v.playableCellRect(); ok && !r.Empty() {
		return r
	}
	return image.Rect(0, 0, v.grid.Width, v.grid.Height)
}

// minimapWindowed is what the composer takes for g's window: terrain colours,
// fog, marks and the camera outline, all moved to the window's own origin.
func (v *Viewer) minimapWindowed(g minimapGeom) ([]color.RGBA, func(col, row int) uint8, []minimapMark, image.Rectangle) {
	colours := v.minimapTerrainColours()
	win := image.Rectangle{Min: g.Origin, Max: g.Origin.Add(image.Pt(g.Cols, g.Rows))}
	if win.Min == (image.Point{}) && g.Cols == v.grid.Width && g.Rows == v.grid.Height {
		return colours, v.fogAt, v.minimapMarks(), v.minimapViewCells()
	}
	cut := make([]color.RGBA, 0, g.Cols*g.Rows)
	for row := win.Min.Y; row < win.Max.Y; row++ {
		cut = append(cut, colours[row*v.grid.Width+win.Min.X:row*v.grid.Width+win.Max.X]...)
	}
	fog := func(col, row int) uint8 { return v.fogAt(col+win.Min.X, row+win.Min.Y) }
	var marks []minimapMark
	for _, m := range v.minimapMarks() {
		if m.Cell.In(win) {
			marks = append(marks, minimapMark{Cell: m.Cell.Sub(win.Min), Local: m.Local})
		}
	}
	return cut, fog, marks, v.minimapViewCells().Intersect(win).Sub(win.Min)
}

// minimapViewCells is the cell range the camera can currently see, as an
// image.Rectangle with Min inclusive and Max exclusive (0140).
//
// Flat terrain uses camera.VisibleTiles directly. Displaced terrain resolves
// the projected quads intersecting the live world window: VisibleTiles' row
// arithmetic is intentionally the flat lattice and would otherwise make the
// green outline stop above terrain the player can actually see.
func (v *Viewer) minimapViewCells() image.Rectangle {
	r := v.cam.VisibleTiles()
	// Include the four lower terrain rows admitted by the draw walk.
	cells, bounded := v.renderCellRect()
	if bounded {
		r.Col0 = max(r.Col0, cells.Min.X)
		r.Col1 = min(r.Col1, cells.Max.X)
	}
	if v.Mode() != ModeDisplaced || r.Col0 >= r.Col1 {
		if bounded {
			r.Row0 = max(r.Row0, cells.Min.Y)
			r.Row1 = min(r.Row1, cells.Max.Y)
		}
		return image.Rect(r.Col0, r.Row0, r.Col1, r.Row1)
	}

	_, top := v.cam.ScreenToWorld(0, 0)
	_, bottom := v.cam.ScreenToWorld(float64(v.cam.ViewW), float64(v.cam.ViewH))
	r0, r1 := v.proj.RowRange(top, bottom)
	if bounded {
		r0 = max(r0, cells.Min.Y)
		r1 = min(r1, cells.Max.Y)
	}
	minRow, maxRow := r1, r0
	for row := r0; row < r1; row++ {
		visible := false
		for col := r.Col0; col < r.Col1; col++ {
			_, y00 := v.proj.WorldCorner(col, row)
			_, y10 := v.proj.WorldCorner(col+1, row)
			_, y01 := v.proj.WorldCorner(col, row+1)
			_, y11 := v.proj.WorldCorner(col+1, row+1)
			cellMin := min(min(y00, y10), min(y01, y11))
			cellMax := max(max(y00, y10), max(y01, y11))
			if float64(cellMax) > top && float64(cellMin) < bottom {
				visible = true
				break
			}
		}
		if visible {
			minRow = min(minRow, row)
			maxRow = max(maxRow, row+1)
		}
	}
	return image.Rect(r.Col0, minRow, r.Col1, maxRow)
}

// minimapGeom is where the minimap is, in window pixels, and under which scale
// rule (0140). Box is the whole framed box — what a click is tested against —
// and Content is the terrain area inside it, which is where a click means a
// cell. The two differ by the border and by however much padding the frame took
// to reach the unit panel's width.
type minimapGeom struct {
	Box        image.Rectangle
	Content    image.Rectangle
	Cols, Rows int
	// Origin is the grid cell at the window's top-left: the minimap draws only
	// the cells of minimapWindow, and Cols and Rows are that window's size.
	Origin image.Point
	// Num/Den is the common rational scale for drawing, marks, the camera
	// outline and click mapping, including when one cell is smaller than a pixel.
	Num, Den int

	// extent is the box handed to composeMinimap: the largest terrain area, in
	// pixels, this geometry allows. It is carried so the composer and the hit
	// test cannot be given different ones.
	extent image.Point
}

// minimapGeometry is the whole of where the minimap lands and how it maps to
// cells, computed once and read by BOTH the picture and the hit test (0140).
//
// THE TERRAIN IS CENTRED IN THE FRAME. Its common rational scale fills the
// limiting axis, even for a map larger than the 128-pixel content square. The
// other axis keeps its proportional margin. When installed crystal art adds a
// left seam, the terrain is centred across the complete art, not only its
// 160-pixel right piece.
//
// THE ONE THING THAT CAN STILL SHRINK IT is the room the frame actually has
// — `v.frameW`/`v.frameH`, always `MissionFrameW`/`MissionFrameH` for
// every Viewer this package constructs, so at the shipped size neither
// `maxW` nor `maxH` below ever binds and the box is the fixed 158 square
// this comment already names. The bound stays in the arithmetic rather than
// being replaced by the constant outright because nothing here assumes the
// frame size can never change again.
//
// AN EMPTY GRID OR NO ROOM AT ALL ANSWERS false, and every
// caller — the picture, the hit test and the click capture — is off together.
func (v *Viewer) minimapGeometry() (minimapGeom, bool) {
	win := v.minimapWindow()
	cols, rows := win.Dx(), win.Dy()
	if cols <= 0 || rows <= 0 {
		return minimapGeom{}, false
	}
	if v.dialogFrame != nil && v.dialogFrame.Minimap != nil && v.frameW >= 320 && v.frameH >= 316 {
		box, ok := rightColumnBox(image.Pt(v.frameW, v.frameH), 0, hudMinimapReserve)
		if !ok {
			return minimapGeom{}, false
		}
		extent := image.Pt(128, 128)
		num, den := minimapScale(cols, rows, extent)
		contentSize := minimapPixelExtent(cols, rows, num, den)
		cw, ch := contentSize.X, contentSize.Y
		// CENTRED ON WHICHEVER AXIS DID NOT JUST REACH THE FULL 128 (owner): the
		// tighter axis's own cw/ch now equals extent exactly
		// (minimapCellPixel(cols, num, den) lands on num, and num is the box
		// extent on the constraining axis by minimapScale's own contract), so its
		// own margin term is 0 and this centring is now a no-op on that axis
		// rather than the fixed dead margin the owner reported — the OTHER axis,
		// still smaller than 128 by the map's own aspect, keeps exactly the margin
		// its own proportion earns, not stretched to fill.
		seam := 0
		if v.dialogFrame.MinimapSeam != nil {
			seam = v.dialogFrame.MinimapSeam.Bounds().Dx()
		}
		at := box.Min.Add(image.Pt(16+(128-cw)/2-seam/2, 18+(128-ch)/2))
		return minimapGeom{Box: box, Content: image.Rectangle{Min: at, Max: at.Add(contentSize)}, Cols: cols, Rows: rows, Origin: win.Min, Num: num, Den: den, extent: extent}, true
	}

	m := authoredMinimapCorner.Margin
	area := image.Pt(v.frameW, v.frameH)

	// THE VIEW IS THE FIRST BOUND, and it is here rather than left to
	// panelOrigin because this box is now a CLICK SURFACE and not only a
	// picture. panelOrigin's own doc says a box too large for the area "lands
	// where the corner puts it and runs off the far edges", which for something
	// drawn is a cosmetic overflow — but a minimap wider than the window hangs
	// off the LEFT edge and swallows presses across the whole map, and a player
	// would find his clicks doing nothing with no box visibly under them.
	// An area with no room for a minimap at all has none.
	maxW, maxH := area.X-2*m.X, area.Y-2*m.Y

	// AND NEVER MORE THAN HALF THE FRAME ON EITHER AXIS. Fitting inside the
	// frame is not the same as being a corner ornament: this bound exists for
	// a frame size this package never ships (the shipped 1024x768 leaves
	// 512x384 here, well clear of the 158-pixel square `hudMinimapReserve`
	// actually bounds the box to below), and it is kept so a future frame
	// size cannot silently give the minimap most of the screen.
	if h := area.X / 2; h < maxW {
		maxW = h
	}
	if h := area.Y / 2; h < maxH {
		maxH = h
	}
	if maxW <= 2 || maxH <= 2 {
		return minimapGeom{}, false
	}

	// ONE SIDE, used for both axes — which is the whole of "always square".
	// hudMinimapReserve (158), NOT sidebarWidth (160): see this function's
	// own header for why the two parted ways at round 2.
	box := hudMinimapReserve
	if box > maxW {
		box = maxW
	}
	if box > maxH {
		box = maxH
	}
	// THE UNIT PANEL IS NOT CONSULTED AT ALL, which is the coupling the owner
	// complained about ("I can see it grow after I pick a hero") staying
	// gone: the column reserves this box's own square and hangs everything
	// else below it (hud.go's hudMinimapReserve), so the bound is the
	// reservation and the reservation is a constant — nothing a selection
	// does can reach it. What gives instead, in a frame too short for the
	// whole column, is the doll and the worn box — they refuse rather than
	// climb over the control panel, and hud.go's hudFloorY is where that is
	// decided.
	if box <= 2 {
		return minimapGeom{}, false
	}

	// The terrain's own square, inside the border.
	extent := image.Pt(box-2, box-2)
	num, den := minimapScale(cols, rows, extent)
	contentSize := minimapPixelExtent(cols, rows, num, den)
	cw, ch := contentSize.X, contentSize.Y
	if cw <= 0 || ch <= 0 {
		return minimapGeom{}, false
	}

	// THE FRAME IS THE SQUARE, never the picture's own extent: a map drawn
	// smaller than the square leaves a margin rather than shrinking the box. As
	// above, the constraining axis's own cw or ch now equals the square's own
	// extent exactly, so its own centring term is 0 and only the OTHER,
	// proportionally smaller axis still carries a margin (owner).
	at := panelOrigin(authoredMinimapCorner, area, image.Pt(box, box))
	contentAt := image.Pt(at.X+(box-cw)/2, at.Y+(box-ch)/2)
	return minimapGeom{
		Box:     image.Rectangle{Min: at, Max: at.Add(image.Pt(box, box))},
		Content: image.Rectangle{Min: contentAt, Max: contentAt.Add(image.Pt(cw, ch))},
		Cols:    cols, Rows: rows, Origin: win.Min,
		Num: num, Den: den,
		extent: extent,
	}, true
}

func (v *Viewer) minimapCaptures(x, y int) bool {
	g, ok := v.minimapGeometry()
	if !ok {
		return false
	}
	if r, ok := rightColumnBox(image.Pt(v.frameW, v.frameH), 0, hudMinimapReserve); ok {
		if v.dialogFrame != nil && v.dialogFrame.Minimap != nil && v.dialogFrame.MinimapSeam != nil {
			r.Min.X -= v.dialogFrame.MinimapSeam.Bounds().Dx()
		}
		return image.Pt(x, y).In(r)
	}
	return image.Pt(x, y).In(g.Box)
}

// minimapCellAt is the cell the window pixel (x, y) names on the minimap, or
// false when that pixel names none (0140).
//
// IT IS minimapScale's OWN ARITHMETIC RUN BACKWARDS, through the geometry both
// it and the picture were built from, so "the cell under the cursor" and "the
// cell drawn there" are one answer.
//
// A PIXEL INSIDE THE BOX BUT OUTSIDE THE TERRAIN ANSWERS false — the border and
// the centring padding. minimapCaptures still swallows such a press; this
// function is what decides whether it MEANS anything, and the two questions are
// deliberately separate.
//
// THE ANSWER IS CLAMPED INTO THE GRID. Integer division of a pixel at the
// content's own last row by the rational scale can land exactly on cols or
// rows. The answer is pulled back into the grid.
func (v *Viewer) minimapCellAt(x, y int) (image.Point, bool) {
	g, ok := v.minimapGeometry()
	if !ok {
		return image.Point{}, false
	}
	p := image.Pt(x, y)
	if !p.In(g.Content) {
		return image.Point{}, false
	}
	dx, dy := x-g.Content.Min.X, y-g.Content.Min.Y

	// Reverse the same scale that sampled or expanded the content pixel.
	col, row := dx*g.Den/g.Num, dy*g.Den/g.Num
	if col >= g.Cols {
		col = g.Cols - 1
	}
	if row >= g.Rows {
		row = g.Rows - 1
	}
	if col < 0 || row < 0 {
		return image.Point{}, false
	}
	return image.Pt(col, row).Add(g.Origin), true
}

// CenterOnMinimapPixel centres the view on the cell the window pixel (x, y)
// names on the minimap, and reports whether it did (0140, owner: "clicking any
// point of the map moves it — that is the centring").
//
// THE CLAMP IS THE CAMERA'S OWN.
func (v *Viewer) CenterOnMinimapPixel(x, y int) bool {
	cell, ok := v.minimapCellAt(x, y)
	if !ok {
		return false
	}
	v.cam.CenterOn(v.cellWorldCentre(cell))
	return true
}
