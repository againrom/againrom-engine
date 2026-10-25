package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// PassArm is one reason a cell is closed to a ground mover, as the host's
// world derivation names it.
type PassArm uint8

const (
	PassBorder PassArm = 1 << iota
	PassMountain
	PassWater
	PassFlag
	PassObject
	PassStructure
)

// passArmWord is the word the status line gives one arm.
func (e *MapEditor) passArmWord(arm PassArm) string {
	switch arm {
	case PassBorder:
		return e.word("editor.pass.border")
	case PassMountain:
		return e.word("editor.pass.mountain")
	case PassWater:
		return e.word("editor.pass.water")
	case PassFlag:
		return e.word("editor.pass.flag")
	case PassObject:
		return e.word("editor.pass.object")
	}
	return e.word("editor.pass.structure")
}

// The passability view's two fills, straight alpha: an open cell is washed
// white, a closed one red, both light enough for the art to read through. The
// border view dims the map's border cells and draws a line where the playable
// cells begin.
var (
	passOpenFill   = color.RGBA{0xff, 0xff, 0xff, 0x40}
	passClosedFill = color.RGBA{0xe0, 0x20, 0x20, 0x60}
	borderFill     = color.RGBA{0x00, 0x00, 0x00, 0x70}
	borderEdge     = color.RGBA{0xff, 0x70, 0x30, 0xff}
)

// passabilityShown reports whether the map carries the passability fills:
// the View menu's switch is on or the passability brush owns the canvas.
func (e *MapEditor) passabilityShown() bool {
	if e.doc == nil || len(e.doc.Passability) != e.doc.Width*e.doc.Height || len(e.doc.Passability) == 0 {
		return false
	}
	return e.view.passability || e.brushActive() && e.brush.mode == brushPassability
}

// borderShown reports whether the map carries the border view: the View
// menu's switch is on, which it is until the user turns it off. The border
// cells are those the host's world closes by its border arm.
func (e *MapEditor) borderShown() bool {
	if e.doc == nil || len(e.doc.Passability) != e.doc.Width*e.doc.Height || len(e.doc.Passability) == 0 {
		return false
	}
	return !e.view.hideBorder
}

// inBorder reports whether the host's world closes cell by its border arm.
func (e *MapEditor) inBorder(cell image.Point) bool {
	return e.passArmAt(cell)&PassBorder != 0
}

// borderQuads is a dimming fill for every drawn border cell, while the
// passability view, which already fills those cells, is off.
func (e *MapEditor) borderQuads() []passQuad {
	if !e.borderShown() || e.passabilityShown() {
		return nil
	}
	v := e.doc.Viewer
	var out []passQuad
	v.forEachDrawnTile(func(col, row int) {
		if e.inBorder(image.Pt(col, row)) {
			out = append(out, passQuad{verts: v.cellSpace().vertices(v.cam, col, row), fill: borderFill})
		}
	})
	return out
}

// borderLines is the line between the border and the playable cells: each
// drawn edge of a playable cell whose neighbour is a border cell.
func (e *MapEditor) borderLines() []editorLine {
	if !e.borderShown() {
		return nil
	}
	v := e.doc.Viewer
	var out []editorLine
	v.forEachDrawnTile(func(col, row int) {
		cell := image.Pt(col, row)
		if col < 0 || row < 0 || col >= e.doc.Width || row >= e.doc.Height || e.inBorder(cell) {
			return
		}
		verts := v.cellSpace().vertices(v.cam, col, row)
		for k, edge := range [4][2]int{{0, 1}, {1, 3}, {3, 2}, {2, 0}} {
			if !e.inBorder(cell.Add(cellSides[k])) {
				continue
			}
			a, b := verts[edge[0]], verts[edge[1]]
			out = append(out, editorLine{a.DstX, a.DstY, b.DstX, b.DstY, borderEdge})
		}
	})
	return out
}

// BorderCell reports whether the border view dims cell, for a headless
// driver.
func (e *MapEditor) BorderCell(cell image.Point) bool {
	return e.borderShown() && !e.passabilityShown() && e.inBorder(cell)
}

// overlayQuads is the cell fills the map carries: the passability view's, or
// else the border view's.
func (e *MapEditor) overlayQuads() []passQuad {
	if q := e.passabilityQuads(); len(q) > 0 {
		return q
	}
	return e.borderQuads()
}

// passArmAt is the arms closing cell, zero for an open cell.
func (e *MapEditor) passArmAt(cell image.Point) PassArm {
	if cell.X < 0 || cell.Y < 0 || cell.X >= e.doc.Width || cell.Y >= e.doc.Height {
		return 0
	}
	return e.doc.Passability[cell.Y*e.doc.Width+cell.X]
}

// hoverPassability is the status line's word for the hovered cell while the
// fills are shown: open, or closed and by which arms.
func (e *MapEditor) hoverPassability() (string, bool) {
	if !e.hoverOK || !e.passabilityShown() {
		return "", false
	}
	arms := e.passArmAt(e.hoverCell)
	if arms == 0 {
		return e.word("editor.pass.open"), true
	}
	var names []string
	for arm := PassBorder; arm <= PassStructure; arm <<= 1 {
		if arms&arm != 0 {
			names = append(names, e.passArmWord(arm))
		}
	}
	return fmt.Sprintf(e.word("editor.pass.closed"), strings.Join(names, ", ")), true
}

// hoverHeight is the status line's height of the hovered cell while the
// height brush owns the canvas.
func (e *MapEditor) hoverHeight() (string, bool) {
	if g := e.brush.set; g != nil && e.brush.down {
		return fmt.Sprintf(e.word("editor.height.target"), g.base-HeightLabelZero, g.target-HeightLabelZero), true
	}
	if !e.hoverOK || !e.brushActive() || e.brush.mode != brushHeight {
		return "", false
	}
	h, ok := e.cellHeightAt(e.hoverCell)
	if !ok {
		return "", false
	}
	return fmt.Sprintf(e.word("editor.height.cell"), h-HeightLabelZero), true
}

// cellHeightAt is the stored height of map cell c, if the map holds one.
func (e *MapEditor) cellHeightAt(c image.Point) (int, bool) {
	alt := e.doc.Viewer.grid.Altitudes
	i := c.Y*e.doc.Width + c.X
	if i < 0 || i >= len(alt) {
		return 0, false
	}
	return int(alt[i]), true
}

// cellHeight is the stored height of map cell c, 0 when the map holds none.
func (e *MapEditor) cellHeight(c image.Point) int {
	h, _ := e.cellHeightAt(c)
	return h
}

// passQuad is one cell's fill: its four screen corners in vertices order and
// its colour.
type passQuad struct {
	verts [4]ebiten.Vertex
	fill  color.RGBA
}

// passabilityQuads is one fill for every map cell the terrain pass draws in
// this frame, the band the grid walks.
func (e *MapEditor) passabilityQuads() []passQuad {
	if !e.passabilityShown() {
		return nil
	}
	v := e.doc.Viewer
	var out []passQuad
	v.forEachDrawnTile(func(col, row int) {
		if col < 0 || row < 0 || col >= e.doc.Width || row >= e.doc.Height {
			return
		}
		q := passQuad{verts: v.cellSpace().vertices(v.cam, col, row), fill: passOpenFill}
		if e.passArmAt(image.Pt(col, row)) != 0 {
			q.fill = passClosedFill
		}
		out = append(out, q)
	})
	return out
}

// drawPassability fills the cells on the GPU canvas, over the map's art and
// under every editor line. The fills are drawn at every zoom: at the
// smallest a cell is a few pixels, and its colour still reads where the
// grid's outline would not, so the cost is held down by drawing them in a
// few batches instead of one call per cell.
func (e *MapEditor) drawPassability(dst *ebiten.Image) {
	quads := e.overlayQuads()
	if len(quads) == 0 {
		return
	}
	if e.passInk == nil {
		e.passInk = ebiten.NewImage(1, 1)
		e.passInk.Fill(color.White)
	}
	op := &ebiten.DrawTrianglesOptions{}
	verts, indices := passBatches(quads)
	for i := range verts {
		dst.DrawTriangles(verts[i], indices[i], e.passInk, op)
	}
}

// passBatchQuads is the most cells one batch holds: four vertices each, so
// every index of the batch fits the 16-bit index type.
const passBatchQuads = 1 << 14

// passBatches is the cells' fills as triangle batches: each cell's four
// corners carry its colour and its two triangles split it as quadIndices does.
func passBatches(quads []passQuad) ([][]ebiten.Vertex, [][]uint16) {
	var verts [][]ebiten.Vertex
	var indices [][]uint16
	for start := 0; start < len(quads); start += passBatchQuads {
		part := quads[start:min(len(quads), start+passBatchQuads)]
		vs := make([]ebiten.Vertex, 0, 4*len(part))
		is := make([]uint16, 0, 6*len(part))
		for _, q := range part {
			base := uint16(len(vs))
			for _, v := range q.verts {
				v.SrcX, v.SrcY = 0.5, 0.5
				v.ColorR = float32(q.fill.R) / 255
				v.ColorG = float32(q.fill.G) / 255
				v.ColorB = float32(q.fill.B) / 255
				v.ColorA = float32(q.fill.A) / 255
				vs = append(vs, v)
			}
			for _, k := range quadIndices {
				is = append(is, base+k)
			}
		}
		verts, indices = append(verts, vs), append(indices, is)
	}
	return verts, indices
}

// paintPassability is drawPassability on the CPU frame: each cell's two
// triangles, quadIndices' split, blended over canvas at pixel centres.
func (e *MapEditor) paintPassability(canvas *image.RGBA) {
	for _, q := range e.overlayQuads() {
		var p [4][2]float64
		for i, v := range q.verts {
			p[i] = [2]float64{float64(v.DstX), float64(v.DstY)}
		}
		lo := image.Pt(int(math.Floor(min(p[0][0], p[1][0], p[2][0], p[3][0]))), int(math.Floor(min(p[0][1], p[1][1], p[2][1], p[3][1]))))
		hi := image.Pt(int(math.Ceil(max(p[0][0], p[1][0], p[2][0], p[3][0]))), int(math.Ceil(max(p[0][1], p[1][1], p[2][1], p[3][1]))))
		box := image.Rectangle{Min: lo, Max: hi}.Intersect(canvas.Bounds())
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				c := [2]float64{float64(x) + 0.5, float64(y) + 0.5}
				if inTriangle(c, p[quadIndices[0]], p[quadIndices[1]], p[quadIndices[2]]) || inTriangle(c, p[quadIndices[3]], p[quadIndices[4]], p[quadIndices[5]]) {
					canvas.SetRGBA(x, y, blendOver(canvas.RGBAAt(x, y), q.fill))
				}
			}
		}
	}
}

// inTriangle reports whether c lies inside or on triangle abc, of either
// winding.
func inTriangle(c, a, b, d [2]float64) bool {
	side := func(p, q [2]float64) float64 { return (q[0]-p[0])*(c[1]-p[1]) - (q[1]-p[1])*(c[0]-p[0]) }
	s1, s2, s3 := side(a, b), side(b, d), side(d, a)
	return (s1 >= 0 && s2 >= 0 && s3 >= 0) || (s1 <= 0 && s2 <= 0 && s3 <= 0)
}

// blendOver is the straight-alpha fill c over the opaque pixel under.
func blendOver(under, c color.RGBA) color.RGBA {
	a := uint32(c.A)
	mix := func(u, f uint8) uint8 { return uint8((uint32(f)*a + uint32(u)*(255-a)) / 255) }
	return color.RGBA{mix(under.R, c.R), mix(under.G, c.G), mix(under.B, c.B), 255}
}

// StatusLine is the text of the status strip, for a headless driver.
func (e *MapEditor) StatusLine() string { return e.statusText() }

// MenuRow is the row of the bar menu menu ("map", "view", "brushes") whose
// action is key, for a headless driver that presses it.
func (e *MapEditor) MenuRow(menu, key string) (int, bool) {
	for i, it := range e.menuItems(menu) {
		if it.key == key {
			return i, true
		}
	}
	return 0, false
}
