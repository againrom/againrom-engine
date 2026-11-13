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
// view its free height. A box that is not Inert takes a press.
type popupCanvas struct {
	Sheet, At image.Point
	Height    int
	Boxes     []canvasBox
	Lines     []canvasLine
}

// canvasBox is one box of a canvas, Area in sheet pixels: its text lines
// top down, and the marks drawn after its first line, each a short word in
// its own ink. An inert box is a flat frame of Fill and Edge; a box that
// takes a press is a push button. Tip reads whole under the box.
type canvasBox struct {
	Area       image.Rectangle
	Lines      []string
	Marks      []canvasMark
	Inert      bool
	Fill, Edge color.RGBA
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
	// canvasArrow is an arrow head's length; it is half as wide again.
	canvasArrow = 5
)

// height is the view's height in the popup, its gap below included.
func (c *popupCanvas) height() int {
	if c == nil {
		return 0
	}
	return c.Height + popupPad
}

// least is the height a sized popup always keeps for the canvas.
func (c *popupCanvas) least() int {
	if c == nil {
		return 0
	}
	return canvasLeastHeight + popupPad
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
// of the popup rest pixels high: the free height of a sized popup; else the
// sheet's height, up to the spec's own Height or, when it names none, to
// the room below the popup's corner.
func fitCanvas(s popupSpec, rest, sizedH int, sized bool) *popupCanvas {
	c := s.Canvas
	if c == nil {
		return nil
	}
	fit := *c
	free := func(h int) int { return max(canvasLeastHeight, h-rest-popupPad) }
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

// placeCanvas lays canvas c at y inside popup rectangle r: its view, the
// bar of a sheet taller than the view, the sheet point shown and each box's
// place in the window, a control row under the box's number. It answers the
// y below the canvas.
func (l *popupLayout) placeCanvas(c *popupCanvas, r image.Rectangle, y int) int {
	view := image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad, y+c.Height)
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

// drawCanvasWords paints the canvas's text: each line's label on the ground,
// over the line, then the text of each box that shows, over its frame or
// button: its lines, and its marks after the first line.
func drawCanvasWords(dst *image.RGBA, body *text.Font, c *popupCanvas, l popupLayout, ink popupInk, over int) {
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
