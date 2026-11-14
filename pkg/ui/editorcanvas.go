package ui

import (
	"image"
	"image/color"
	"image/draw"
	"slices"
	"strings"

	"againrom/pkg/render/text"
)

// popupCanvas is a drawing as data: boxes of text and the lines between
// them, on a sheet that may be larger than the popup shows. Sheet is the
// sheet's extent and At the sheet's point at the view's top-left corner;
// the layout holds At inside the sheet. Height is the view's height in a
// popup at its own size, zero for the whole sheet; a sized popup gives the
// view its free height. A box that is not Inert takes a press. Legend is
// words stated over the view, each after a swatch of its ink; they stand
// still while the sheet moves, and head is the height a layout gave them.
type popupCanvas struct {
	Sheet, At image.Point
	Height    int
	Boxes     []canvasBox
	Lines     []canvasLine
	Legend    []canvasMark
	head      int
}

// canvasBox is one box of a canvas, Area in sheet pixels: its text lines
// top down, and the marks drawn after its first line, each a short word in
// its own ink. An inert box is a flat frame of Fill and Edge; a box that
// takes a press is a push button. Band, when set, is a strip across the
// box's top in that colour, over the frame or the button alike. Tip reads
// whole under the box.
type canvasBox struct {
	Area       image.Rectangle
	Lines      []string
	Marks      []canvasMark
	Inert      bool
	Fill, Edge color.RGBA
	Band       color.RGBA
	Tip        string
}

// canvasMark is a short word drawn in its own ink.
type canvasMark struct {
	Text string
	Ink  color.RGBA
}

// canvasLine is a line through Points in sheet pixels, each segment level
// or upright, with an arrow head at its last point when Arrow is set. A
// line set Over is drawn over those that are not. Label is written in the
// line's ink on its first segment, from the first point on.
type canvasLine struct {
	Points []image.Point
	Ink    color.RGBA
	Arrow  bool
	Over   bool
	Label  string
}

const (
	// canvasLeastHeight is the least height a sized popup keeps its view.
	canvasLeastHeight = 48
	// canvasBoxPad is the room between a box's edge and its text.
	canvasBoxPad = 4
	// canvasBand is where a box's band ends: it fills rows 1 to 3 inside the
	// box's outer pixel, above the text's pad.
	canvasBand = 4
	// canvasArrow is an arrow head's length; it is half as wide again.
	canvasArrow = 5
)

// height is the canvas's height in the popup: its legend, its view and the
// gap below.
func (c *popupCanvas) height() int {
	if c == nil {
		return 0
	}
	return c.head + c.Height + popupPad
}

// least is the height a sized popup always keeps for the canvas.
func (c *popupCanvas) least() int {
	if c == nil {
		return 0
	}
	return c.head + canvasLeastHeight + popupPad
}

// canvasSwatch is the width of a legend word's swatch and the room after it.
const canvasSwatch = 12

// legendPlaces is where each legend word stands, its swatch first, relative
// to the legend's corner, in lines a text line and a gap high that wrap at
// width; and the height the lines take.
func (c *popupCanvas) legendPlaces(body *text.Font, width int) ([]image.Rectangle, int) {
	if c == nil || len(c.Legend) == 0 {
		return nil, 0
	}
	line := fontHeight(body, 15) + 4
	out, x, y := make([]image.Rectangle, len(c.Legend)), 0, 0
	for i, m := range c.Legend {
		w := 2*canvasSwatch + 7*len(m.Text)
		if body != nil {
			w = 2*canvasSwatch + body.Advance(m.Text)
		}
		if x > 0 && x+w > width {
			x, y = 0, y+line
		}
		out[i] = image.Rect(x, y, x+w, y+line)
		x += w + canvasSwatch
	}
	return out, y + line
}

// legendWords is the legend's words in order.
func (c *popupCanvas) legendWords() []string {
	if c == nil {
		return nil
	}
	out := make([]string, len(c.Legend))
	for i, m := range c.Legend {
		out[i] = m.Text
	}
	return out
}

// texts is each box as one line of text: its lines that hold any, then its
// marks.
func (c *popupCanvas) texts() []string {
	if c == nil {
		return nil
	}
	out := make([]string, len(c.Boxes))
	for i, b := range c.Boxes {
		parts := slices.DeleteFunc(slices.Clone(b.Lines), func(s string) bool { return s == "" })
		for _, m := range b.Marks {
			parts = append(parts, m.Text)
		}
		out[i] = strings.Join(parts, " ")
	}
	return out
}

// labels is the label of each line that carries one, in the lines' order.
func (c *popupCanvas) labels() []string {
	if c == nil {
		return nil
	}
	var out []string
	for _, l := range c.Lines {
		if l.Label != "" {
			out = append(out, l.Label)
		}
	}
	return out
}

// fitCanvas is popup s's canvas with the height its view takes, the rest
// of the popup rest pixels high and its legend head pixels over the view:
// the free height of a sized popup; else the sheet's height, up to the
// spec's own Height or, when it names none, to the room below the popup's
// corner.
func fitCanvas(s popupSpec, rest, head, sizedH int, sized bool) *popupCanvas {
	c := s.Canvas
	if c == nil {
		return nil
	}
	fit := *c
	fit.head = head
	free := func(h int) int { return max(canvasLeastHeight, h-rest-head-popupPad) }
	room := gridRoom(s)
	switch {
	case sized:
		fit.Height = free(sizedH)
	case c.Height > 0:
		fit.Height = min(c.Height, c.Sheet.Y)
	case !room.Empty():
		fit.Height = min(c.Sheet.Y, free(room.Max.Y-max(s.At.Y, room.Min.Y)))
	default:
		fit.Height = c.Sheet.Y
	}
	return &fit
}

// placeCanvas lays canvas c at y inside popup rectangle r: its legend, its
// view under the legend, the bar of a sheet taller than the view, the sheet
// point shown and each box's place in the window, a control row under the
// box's number. It answers the y below the canvas.
func (l *popupLayout) placeCanvas(c *popupCanvas, r image.Rectangle, y int, legend []image.Rectangle) int {
	corner := image.Pt(r.Min.X+popupPad, y)
	l.CanvasLegend = make([]image.Rectangle, len(legend))
	for i, at := range legend {
		l.CanvasLegend[i] = at.Add(corner)
	}
	view := image.Rect(r.Min.X+popupPad, y+c.head, r.Max.X-popupPad, y+c.head+c.Height)
	if c.Sheet.Y > view.Dy() {
		l.CanvasBar = image.Rect(view.Max.X-widgetSpriteSize, view.Min.Y, view.Max.X, view.Max.Y)
		view.Max.X -= widgetSpriteSize + 2
	}
	l.Canvas = view
	l.CanvasAt = image.Pt(max(0, min(c.At.X, c.Sheet.X-view.Dx())), max(0, min(c.At.Y, c.Sheet.Y-view.Dy())))
	shift := view.Min.Sub(l.CanvasAt)
	l.CanvasBoxes = make(controlLayout, len(c.Boxes))
	l.CanvasInert = make([]bool, len(c.Boxes))
	for i, b := range c.Boxes {
		l.CanvasBoxes[i], l.CanvasInert[i] = controlRow{id: i, rect: b.Area.Add(shift)}, b.Inert
	}
	return y + c.height()
}

// drawCanvasLegend paints the canvas's legend: each word's swatch in its
// ink, then the word in the popup's label ink.
func drawCanvasLegend(dst *image.RGBA, body *text.Font, c *popupCanvas, l popupLayout, ink popupInk) {
	for i, at := range l.CanvasLegend {
		if i >= len(c.Legend) {
			return
		}
		m := c.Legend[i]
		swatch := image.Rect(at.Min.X, at.Min.Y+3, at.Min.X+canvasSwatch, at.Max.Y-3).Intersect(l.Rect)
		draw.Draw(dst, swatch, &image.Uniform{C: m.Ink}, image.Point{}, draw.Src)
		if body != nil {
			body.Draw(dst.SubImage(at.Intersect(l.Rect)).(*image.RGBA), m.Text, at.Min.X+2*canvasSwatch-canvasSwatch/2, at.Min.Y+2, ink.Label)
		}
	}
}

// canvasBar is the scroll bar of a canvas whose sheet is taller than its
// view: one position for each row pitch of the sheet below the view.
func (l popupLayout) canvasBar(c *popupCanvas) vScrollBar {
	step := max(1, l.Pitch)
	hidden := max(0, c.Sheet.Y-l.Canvas.Dy())
	return vScrollBar{Rect: l.CanvasBar, Pos: (l.CanvasAt.Y + step - 1) / step, Count: (hidden+step-1)/step + 1}
}

// canvasHit is the canvas part under p, a point of the view: the box that
// takes a press there, else the canvas's ground.
func (l popupLayout) canvasHit(p image.Point) (popupPart, int) {
	for i := len(l.CanvasBoxes) - 1; i >= 0; i-- {
		if l.CanvasBoxes.holds(i, p) && !l.CanvasInert[i] {
			return popupCanvasBox, i
		}
	}
	return popupCanvasGround, 0
}

// canvasOver is the canvas box a press at the pointer would take, -1 for
// none: the box drawn lit.
func (l popupLayout) canvasOver(pointer image.Point, pointerOK bool) int {
	if !pointerOK {
		return -1
	}
	if part, i := l.hit(pointer); part == popupCanvasBox {
		return i
	}
	return -1
}

// canvasShown is the part of canvas box i that shows in the view, empty for
// a box moved out of it.
func (l popupLayout) canvasShown(i int) image.Rectangle {
	return l.CanvasBoxes[i].rect.Intersect(l.Canvas.Inset(1))
}

// drawCanvasSheet paints a canvas's view below its buttons: its ground, the
// lines, then each inert box that shows, a flat frame. Everything is cut at
// the view's edge.
func drawCanvasSheet(dst *image.RGBA, c *popupCanvas, l popupLayout, ink popupInk) {
	drawFrame(dst, panelFrame(l.Canvas, ink.TitleGround, ink.Edge))
	clip := dst.SubImage(l.Canvas.Inset(1)).(*image.RGBA)
	shift := l.Canvas.Min.Sub(l.CanvasAt)
	for _, over := range []bool{false, true} {
		for _, line := range c.Lines {
			if line.Over == over {
				drawCanvasLine(clip, line, shift)
			}
		}
	}
	for i, b := range c.Boxes {
		if !b.Inert || l.canvasShown(i).Empty() {
			continue
		}
		edge := b.Edge
		if edge == (color.RGBA{}) {
			edge = ink.Edge
		}
		drawFrame(clip, panelFrame(l.CanvasBoxes[i].rect, b.Fill, edge))
	}
}

// canvasButtons are the push buttons of canvas c that show: each box that
// takes a press, the one under the pointer lit. The one painter pushes them,
// cut at the view's edge.
func (l popupLayout) canvasButtons(c *popupCanvas, over int, held *buttonPress) []pushButton {
	var out []pushButton
	for i, b := range c.Boxes {
		if !b.Inert && !l.canvasShown(i).Empty() {
			out = append(out, pushButton{Press: held, Rect: l.CanvasBoxes[i].rect, Hover: i == over})
		}
	}
	return out
}

// drawCanvasWords paints each box's band over its frame or button, then the
// canvas's text: each line's label on the ground, over the line, then the
// text of each box that shows: its lines, and its marks after the first line.
func drawCanvasWords(dst *image.RGBA, body *text.Font, c *popupCanvas, l popupLayout, ink popupInk, over int) {
	for i, b := range c.Boxes {
		if r := l.CanvasBoxes[i].rect; b.Band != (color.RGBA{}) {
			band := image.Rect(r.Min.X+1, r.Min.Y+1, r.Max.X-1, r.Min.Y+canvasBand).Intersect(l.Canvas.Inset(1))
			draw.Draw(dst, band, &image.Uniform{C: b.Band}, image.Point{}, draw.Src)
		}
	}
	if body == nil {
		return
	}
	line := fontHeight(body, 15)
	view, shift := dst.SubImage(l.Canvas.Inset(1)).(*image.RGBA), l.Canvas.Min.Sub(l.CanvasAt)
	// Every label's ground is cleared before any label is written.
	for _, written := range []bool{false, true} {
		for _, ln := range c.Lines {
			if ln.Label == "" || len(ln.Points) == 0 {
				continue
			}
			p := ln.Points[0].Add(shift).Add(image.Pt(canvasBoxPad, -line/2))
			if written {
				body.Draw(view, ln.Label, p.X, p.Y, ln.Ink)
				continue
			}
			ground := image.Rectangle{Min: p, Max: p.Add(image.Pt(body.Advance(ln.Label), line))}
			draw.Draw(view, ground, &image.Uniform{C: ink.TitleGround}, image.Point{}, draw.Src)
		}
	}
	for i, b := range c.Boxes {
		r := l.CanvasBoxes[i].rect
		if l.canvasShown(i).Empty() {
			continue
		}
		value := ink.Label
		if i == over {
			value = ink.Value
		}
		text := dst.SubImage(r.Inset(2).Intersect(l.Canvas.Inset(1))).(*image.RGBA)
		x, y := r.Min.X+canvasBoxPad, r.Min.Y+canvasBoxPad
		for k, s := range b.Lines {
			body.Draw(text, s, x, y+k*line, value)
			if k > 0 {
				continue
			}
			at := x + body.Advance(s)
			for _, m := range b.Marks {
				at += body.Advance(" ")
				body.Draw(text, m.Text, at, y, m.Ink)
				at += body.Advance(m.Text)
			}
		}
	}
}

// drawCanvasLine paints line moved by shift, cut at dst's bounds.
func drawCanvasLine(dst *image.RGBA, line canvasLine, shift image.Point) {
	dot := func(x, y int) { dst.SetRGBA(x, y, line.Ink) }
	for i := 1; i < len(line.Points); i++ {
		a, b := line.Points[i-1].Add(shift), line.Points[i].Add(shift)
		box := image.Rectangle{Min: a, Max: b}.Canon()
		box.Max = box.Max.Add(image.Pt(1, 1))
		if box = box.Intersect(dst.Bounds()); box.Empty() {
			continue
		}
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				dot(x, y)
			}
		}
	}
	if n := len(line.Points); line.Arrow && n > 1 {
		tip, from := line.Points[n-1].Add(shift), line.Points[n-2].Add(shift)
		back := image.Pt(sign(from.X-tip.X), sign(from.Y-tip.Y))
		for k := 1; k <= canvasArrow; k++ {
			for w := -(k + 1) / 2; w <= (k+1)/2; w++ {
				dot(tip.X+back.X*k+back.Y*w, tip.Y+back.Y*k+back.X*w)
			}
		}
	}
}

// canvasTipAt is the tip of the canvas box under p: the box's Tip, whole,
// under the box.
func canvasTipAt(bounds image.Rectangle, body *text.Font, c *popupCanvas, l popupLayout, p image.Point) (popupTip, bool) {
	if c == nil || body == nil {
		return popupTip{}, false
	}
	if part, _ := l.hit(p); part != popupCanvasBox && part != popupCanvasGround {
		return popupTip{}, false
	}
	for i := len(l.CanvasBoxes) - 1; i >= 0; i-- {
		r := l.CanvasBoxes[i].rect
		if !l.CanvasBoxes.holds(i, p) || c.Boxes[i].Tip == "" {
			continue
		}
		lines := editorTextLines(body, c.Boxes[i].Tip, min(max(r.Dx(), popupTipW), bounds.Dx())-8)
		w := 0
		for _, line := range lines {
			w = max(w, body.Advance(line))
		}
		h := len(lines)*fontHeight(body, 15) + 8
		box := image.Rect(r.Min.X, r.Max.Y, r.Min.X+w+8, r.Max.Y+h)
		return popupTip{Rect: placeTip(tipPlace{Box: box, PullRight: true, FlipBy: h + r.Dy()}, bounds), Lines: lines}, true
	}
	return popupTip{}, false
}
