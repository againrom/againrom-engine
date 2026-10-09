package ui

import (
	"image"
	"image/color"
	"time"

	"againrom/pkg/render/backdrop"
)

// The edit field's requested colours, selection level and caret blink
// (MENU-126).
var (
	editShade = color.RGBA{8, 8, 8, 255}
	editLight = color.RGBA{94, 115, 101, 255}
	editCaret = color.RGBA{255, 255, 255, 255}
)

const (
	editSelectionLevel = 12
	editTextX          = 4
	editBlink          = 500 * time.Millisecond
)

// editField is one edit control's look: rectangle, the text's pixel origin
// offset (the engine's scroll of a long text), the selection and caret as
// pixel offsets from the text origin, the text height, focus and the blink
// phase. Text itself is drawn by the caller's label routine.
type editField struct {
	Rect           image.Rectangle
	SelFrom, SelTo int
	Caret          int
	TextH          int
	Focus, Phase   bool
}

// TextOrigin is where the text starts: (L+4, T+trunc((B-T)/2)) centred on
// the text height.
func (e editField) TextOrigin() image.Point {
	l, t, b := e.Rect.Min.X, e.Rect.Min.Y, e.Rect.Max.Y-1
	return image.Pt(l+editTextX, t+(b-t)/2-e.TextH/2)
}

// drawEditField paints the four bevel lines, the level-12 selection remap
// and, after label draws the text, the two-pixel white caret while focused
// in the visible phase. It fills nothing: the parent's background shows.
func drawEditField(dst *image.RGBA, e editField, label func(at image.Point)) {
	recordWidget(widgetEdit, e.Rect, e)
	if dst == nil || e.Rect.Dx() < 3 || e.Rect.Dy() < 3 {
		return
	}
	l, t, r, b := e.Rect.Min.X, e.Rect.Min.Y, e.Rect.Max.X-1, e.Rect.Max.Y-1
	set := func(x, y int, c color.RGBA) {
		if image.Pt(x, y).In(dst.Bounds()) {
			dst.SetRGBA(x, y, c)
		}
	}
	for x := l + 1; x <= r-1; x++ {
		set(x, t, editShade)
		set(x, b, editLight)
	}
	for y := t + 1; y <= b-1; y++ {
		set(l, y, editShade)
		set(r, y, editLight)
	}
	inner := image.Rect(l+1, t+1, r, b).Intersect(dst.Bounds())
	if e.SelFrom != e.SelTo {
		x0, x1 := min(e.SelFrom, e.SelTo), max(e.SelFrom, e.SelTo)
		sel := image.Rect(l+editTextX+x0, t+2, l+editTextX+x1, b-2).Intersect(inner)
		if lookup, err := backdrop.NewLevel(backdrop.RGB565, backdrop.Full, editSelectionLevel); err == nil {
			lookup.Apply(dst, sel, dst.Bounds(), 1)
		}
	}
	if label != nil {
		label(e.TextOrigin())
	}
	if e.Focus && e.Phase {
		x := l + editTextX + e.Caret
		for y := t + 2; y < b-2; y++ {
			for dx := 0; dx < 2; dx++ {
				if image.Pt(x+dx, y).In(inner) {
					dst.SetRGBA(x+dx, y, editCaret)
				}
			}
		}
	}
}

// caretBlink is the edit caret's phase: visible on reset, and toggled when
// more than 500 ms have passed since the last toggle.
type caretBlink struct {
	last time.Time
	off  bool
}

// tick advances the phase to now.
func (c *caretBlink) tick(now time.Time) {
	if c.last.IsZero() || now.Before(c.last) {
		c.last = now
		return
	}
	if now.Sub(c.last) > editBlink {
		c.off, c.last = !c.off, now
	}
}

// reset shows the caret and restarts the interval.
func (c *caretBlink) reset(now time.Time) { c.off, c.last = false, now }

// on reports the visible phase.
func (c caretBlink) on() bool { return !c.off }
