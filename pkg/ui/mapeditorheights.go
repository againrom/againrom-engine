package ui

import (
	"image"
	"strconv"
)

// HeightLabelZero is the stored height the Layers menu's height numbers show as
// 0: the height a new map starts at (TERR-246). A number is the stored height
// minus this, so a lower point shows a negative number.
const HeightLabelZero = NewMapDefaultAltitude

// heightLabelWidest is the widest number a stored height 0..127 shows.
const heightLabelWidest = "-64"

func (e *MapEditor) heightLabelRows() []mapLabel {
	var out []mapLabel
	if e.view.heights && e.doc != nil && !e.catalog {
		f := e.bodyFont()
		v, g := e.doc.Viewer, e.doc.Viewer.grid
		if len(g.Altitudes) == g.Width*g.Height && g.Width >= 2 {
			h := fontHeight(f, 15) + 2
			wide := 6 + len(heightLabelWidest)*8
			if f != nil {
				wide = f.Advance(heightLabelWidest) + 6
			}
			x0, _ := v.cellScreenCorner(0, 0)
			x1, _ := v.cellScreenCorner(1, 0)
			cell := x1 - x0
			if cell > 0 {
				k := 1
				for cell*float64(k) < float64(max(wide, h)) && k < 1<<12 {
					k *= 2
				}
				ox, oy := e.mapOrigin()
				v.forEachDrawnTile(func(c, r int) {
					if c%k != 0 || r%k != 0 {
						return
					}
					x, y := e.corner(image.Pt(c, r))
					out = append(out, mapLabel{
						source: labelHeight, index: r*g.Width + c,
						anchor: image.Pt(int(float64(x)+ox), int(float64(y)+oy)), offset: image.Pt(1, 1),
						text:    strconv.Itoa(int(g.Altitudes[r*g.Width+c]) - HeightLabelZero),
						padding: image.Pt(3, 1), ink: editorPaper, popup: labelPopupOmit, edge: labelInside,
						collides: labelTrigger | labelGroup | labelHeight,
					})
				})
			}
		}
	}
	if g := e.brush.set; g != nil && e.brush.down && e.doc != nil {
		ox, oy := e.mapOrigin()
		out = append(out, mapLabel{
			source: labelHeightGesture,
			anchor: image.Pt(int(float64(g.pointer.X)+ox), int(float64(g.pointer.Y)+oy)), offset: image.Pt(14, 14),
			text: strconv.Itoa(g.target - HeightLabelZero), padding: image.Pt(3, 1), ink: editorPaper, edge: labelUnbounded,
		})
	}
	return out
}

// HeightLabels are the height numbers the window draws: each one's window
// rectangle and text, for a test to read.
func (e *MapEditor) HeightLabels() (rects []image.Rectangle, texts []string) {
	for _, l := range e.mapLabelsOf(labelHeight) {
		rects, texts = append(rects, l.rect), append(texts, l.text)
	}
	return rects, texts
}

// keyPassView turns the passability view on or off.
func (e *MapEditor) keyPassView(string) error {
	e.view.passability = !e.view.passability
	return nil
}

// keyHeightLabels turns the height numbers on or off and says so on the
// status line, which also tells a map maker zoomed out why none show.
func (e *MapEditor) keyHeightLabels(string) error {
	e.view.heights = !e.view.heights
	note := e.word("editor.heights.off")
	if e.view.heights {
		note = e.word("editor.heights.on")
	}
	e.noteSelection(note, nil)
	return nil
}
