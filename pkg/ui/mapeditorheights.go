package ui

import (
	"image"
	"image/draw"
	"strconv"
)

// HeightLabelZero is the stored height the View menu's height numbers show as
// 0: the height a new map starts at (TERR-246). A number is the stored height
// minus this, so a lower point shows a negative number.
const HeightLabelZero = NewMapDefaultAltitude

// heightLabelWidest is the widest number a stored height 0..127 shows.
const heightLabelWidest = "-64"

// heightLabel is one height number on the map: where it stands in the
// window and its text.
type heightLabel struct {
	rect image.Rectangle
	text string
}

// heightLabels are the View menu's height numbers. A stored height belongs to
// a mesh vertex: the height at index x+W*y lifts vertex (x, y), the top-left
// corner of cell (x, y) (TERR-GEOM-031), so each number stands just inside
// that corner. They are drawn only while a cell is at least as wide on screen
// as the widest number's box and as tall as the font's line, and a number is
// left out where a popup, a trigger label, a group badge or a number already
// placed covers it. None while the switch is off.
func (e *MapEditor) heightLabels() []heightLabel {
	if !e.view.heights || e.doc == nil || e.catalog {
		return nil
	}
	f := e.bodyFont()
	v := e.doc.Viewer
	g := v.grid
	if len(g.Altitudes) != g.Width*g.Height || g.Width < 2 {
		return nil
	}
	h := fontHeight(f, 15) + 2
	wide := 6 + len(heightLabelWidest)*8
	if f != nil {
		wide = f.Advance(heightLabelWidest) + 6
	}
	x0, _ := v.cellScreenCorner(0, 0)
	x1, _ := v.cellScreenCorner(1, 0)
	if cell := x1 - x0; !(cell >= float64(wide) && cell >= float64(h)) {
		return nil
	}
	var covers []image.Rectangle
	covers = append(covers, e.popupRects()...)
	for _, l := range e.areaLabels() {
		covers = append(covers, l.rect)
	}
	for _, b := range e.groupBadges() {
		covers = append(covers, b.rect)
	}
	area := e.mapArea()
	ox, oy := v.place.Origin()
	var out []heightLabel
	v.forEachDrawnTile(func(c, r int) {
		text := strconv.Itoa(int(g.Altitudes[r*g.Width+c]) - HeightLabelZero)
		w := 6
		if f != nil {
			w += f.Advance(text)
		}
		x, y := e.corner(image.Pt(c, r))
		at := image.Pt(int(float64(x)+ox)+1, int(float64(y)+oy)+1)
		rect := image.Rectangle{Min: at, Max: at.Add(image.Pt(w, h))}
		if !rect.In(area) {
			return
		}
		for _, p := range covers {
			if p.Overlaps(rect) {
				return
			}
		}
		for _, l := range out {
			if l.rect.Overlaps(rect) {
				return
			}
		}
		out = append(out, heightLabel{rect: rect, text: text})
	})
	return out
}

// drawHeightLabels paints the height numbers on the chrome picture, under
// the trigger labels and the popups.
func (e *MapEditor) drawHeightLabels(pic *image.RGBA) {
	f := e.bodyFont()
	for _, l := range e.heightLabels() {
		draw.Draw(pic, l.rect, image.NewUniform(editorInk), image.Point{}, draw.Src)
		if f != nil {
			f.Draw(pic.SubImage(l.rect).(*image.RGBA), l.text, l.rect.Min.X+3, l.rect.Min.Y+1, editorPaper)
		}
	}
}

// HeightLabels are the height numbers the window draws: each one's window
// rectangle and text, for a test to read.
func (e *MapEditor) HeightLabels() (rects []image.Rectangle, texts []string) {
	for _, l := range e.heightLabels() {
		rects, texts = append(rects, l.rect), append(texts, l.text)
	}
	return rects, texts
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
