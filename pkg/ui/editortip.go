package ui

import (
	"image"

	"againrom/pkg/render/text"
)

// popupTip is the editor's one tip element: the whole text it shows, that
// text wrapped to the box, the picture under the text (nil for none), the
// element it explains, the box, and whether it is a (?) mark's help, which
// shows at once; every other tip shows after editorTipDelay.
type popupTip struct {
	Text    string
	Lines   []string
	Picture image.Image
	Anchor  image.Rectangle
	Rect    image.Rectangle
	Help    bool
	// Top is the first line shown and Rows how many lines show, 0 for all:
	// a text taller than the editor shows a window of its lines and scrolls.
	Top, Rows int
	// Style is how the tip paints and Hover marks a game hover box, which
	// records as one; the painter takes both from the caller that paints.
	Style *hoverStyle
	Hover bool
}

// shown is the lines the box holds, from Top.
func (t popupTip) shown() []string {
	if t.Rows <= 0 || t.Rows >= len(t.Lines) {
		return t.Lines
	}
	top := max(0, min(t.Top, len(t.Lines)-t.Rows))
	return t.Lines[top : top+t.Rows]
}

// bar is the scroll bar at the box's right edge; its position is the first line.
func (t popupTip) bar() vScrollBar {
	r := t.Rect.Inset(2)
	return vScrollBar{Rect: image.Rect(r.Max.X-widgetSpriteSize, r.Min.Y, r.Max.X, r.Max.Y), Pos: t.Top, Count: len(t.Lines) - t.Rows + 1}
}

// scrolled is t with its first line moved by delta lines, kept inside the text.
func (t popupTip) scrolled(top, delta int) popupTip {
	if t.Rows > 0 {
		t.Top = max(0, min(top+delta, len(t.Lines)-t.Rows))
	}
	return t
}

// tipPad is the room between a tip's frame and its text or picture.
const tipPad = 4

// tipShape is how a tip's box takes its size and place: the width its text
// wraps in (bounds' width for 0), whether the box narrows to its widest
// line, whether a box past the right edge moves back to it, and whether a
// box past the left or top edge moves back and is then cut to bounds.
type tipShape struct {
	Width           int
	Measure         func(string) int // for a description preview without a font
	Fit, Pull, Clip bool
	// Scroll lets a text taller than bounds show as many lines as bounds
	// hold, the rest reached by scrolling.
	Scroll bool
}

// layoutTip lays out the one tip of text and picture for the element at
// anchor, shaped by how: the picture stands under the text, the box under
// the anchor at its left edge, or over it where bounds end first.
func layoutTip(bounds image.Rectangle, body *text.Font, anchor image.Rectangle, full string, picture image.Image, how tipShape) popupTip {
	width := how.Width
	if width <= 0 {
		width = bounds.Dx()
	}
	measure := advanceOf(body)
	if how.Measure != nil {
		measure = how.Measure
	}
	lines := editorRule.plain(lineBox{Measure: measure, Width: width - 2*tipPad}, full)
	if how.Fit {
		width = 0
		for _, line := range lines {
			width = max(width, measure(line)+2*tipPad)
		}
	}
	shape := editorTipShape(body)
	shown := len(lines)
	rows := 0
	if how.Scroll && tipSize(shown, width, body, &shape, nil).Y > bounds.Dy() && picture == nil {
		// The bar stands at the right edge, so the text wraps narrower.
		lines = editorRule.plain(lineBox{Measure: measure, Width: width - 2*tipPad - widgetSpriteSize}, full)
		rows = max(1, (bounds.Dy()-2*tipPad)/fontHeight(body, 15))
		rows = min(rows, len(lines))
		shown = rows
	}
	size := tipSize(shown, width, body, &shape, picture)
	h := size.Y
	box := image.Rect(anchor.Min.X, anchor.Max.Y, anchor.Min.X+size.X, anchor.Max.Y+h)
	place := tipPlace{Box: box, PullRight: how.Pull, FlipBy: h + anchor.Dy(), PushTopLeft: how.Clip, Clip: how.Clip}
	return popupTip{Text: full, Lines: lines, Picture: picture, Anchor: anchor, Rect: placeTip(place, bounds), Rows: rows}
}

// hoverTip is the tip of a grid cell, a toolbar button or a choice's row:
// as wide as its widest line, kept inside bounds.
func hoverTip(bounds image.Rectangle, body *text.Font, anchor image.Rectangle, full string, picture image.Image) popupTip {
	return layoutTip(bounds, body, anchor, full, picture, tipShape{Fit: true, Pull: true, Clip: true})
}

func valueTip(bounds image.Rectangle, body *text.Font, at image.Rectangle, full string) popupTip {
	return layoutTip(bounds, body, at, full, nil, tipShape{Width: min(popupTipW, bounds.Dx()), Measure: editorMeasure(body), Pull: true, Clip: true, Scroll: true})
}

// helpTip is the tip of a (?) mark at r: popupHelpW wide, the help wrapped
// to as many body lines as it takes, kept inside bounds and scrolled where
// the lines are more than bounds hold; it shows at once.
func helpTip(bounds image.Rectangle, body *text.Font, r image.Rectangle, help string) popupTip {
	tip := layoutTip(bounds, body, r, help, nil, tipShape{Width: min(popupHelpW, bounds.Dx()), Pull: true, Clip: true, Scroll: true})
	tip.Help = true
	return tip
}

// key names the tip for the tip clock: its text and the element it
// explains, so the clock restarts on another element.
func (t popupTip) key() string {
	if t.Text == "" {
		return ""
	}
	return t.Text + "\x00" + t.Anchor.String()
}

// drawPopupTip paints tip t in the editor's window inks: the title ground
// framed in the edge ink, the lines in the value ink, the picture under them.
func drawPopupTip(dst *image.RGBA, body *text.Font, t popupTip, ink popupInk) {
	if dst == nil || body == nil || t.Rect.Empty() {
		return
	}
	t.Style = editorTipStyle(body, ink)
	paintTip(dst, body, t, nil)
}
