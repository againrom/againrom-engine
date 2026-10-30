package ui

import (
	"image"
	"image/color"

	"againrom/pkg/render/text"
)

// dropdown is one drop-down choice: a closed box showing the chosen row's
// text with an arrow button, and, while Open, the choices below it in the
// shared list box over the same list model. The model's selection is the
// highlighted choice; Fill is painted behind the open list so it covers what
// lies under it.
type dropdown struct {
	Rect     image.Rectangle
	Text     string
	Open     bool
	List     *Picker
	Rows     int // rows the open list shows; at most the model's window
	Focus    bool
	Disabled bool
	Fill     color.RGBA
	Ink      color.RGBA
	// ListWidth widens the open list past the box; zero is the box's width.
	ListWidth int
}

// button is the arrow button at the closed box's right end.
func (d dropdown) button() image.Rectangle {
	w := min(d.Rect.Dx()/2, d.Rect.Dy())
	return image.Rect(d.Rect.Max.X-w, d.Rect.Min.Y, d.Rect.Max.X, d.Rect.Max.Y)
}

// listBox is the open list's geometry: below the box, as wide as it less the
// scroll bar, as many rows as the list holds up to Rows.
func (d dropdown) listBox(f *text.Font) listBox {
	rows := d.Rows
	if d.List != nil {
		rows = max(1, min(rows, d.List.Len()))
	}
	r := image.Rect(d.Rect.Min.X, d.Rect.Max.Y+1, d.Rect.Min.X+max(d.Rect.Dx(), d.ListWidth)-widgetSpriteSize, d.Rect.Max.Y+1)
	return newListBox(r, max(1, rows), f)
}

// Bounds is everything the drop-down covers: the box, and the open list with
// its bar.
func (d dropdown) Bounds(f *text.Font) image.Rectangle {
	if !d.Open {
		return d.Rect
	}
	l := d.listBox(f)
	return d.Rect.Union(l.Rect.Inset(-1)).Union(l.Bar())
}

// Covers reports whether p lies on the drop-down: the box, or the open list
// with its rim and bar. A list wider than the box leaves the strip beside the
// box free, which Bounds' rectangle does not.
func (d dropdown) Covers(f *text.Font, p image.Point) bool {
	if p.In(d.Rect) {
		return true
	}
	if !d.Open {
		return false
	}
	l := d.listBox(f)
	return p.In(l.Rect.Inset(-1)) || p.In(l.Bar())
}

// drawDropdown is the one drop-down builder: the closed box as an edit
// field's well with the text and an arrow push button, then, while open, a
// fill and the shared list box over the drop-down's list model.
func drawDropdown(dst *image.RGBA, f *text.Font, frames []*image.RGBA, d dropdown, rowText listRowText, pointer image.Point, pointerOK bool) {
	recordWidget(widgetDropdown, d.Rect, d)
	if dst == nil || d.Rect.Dx() < 8 || d.Rect.Dy() < 4 {
		return
	}
	arrow := d.button()
	box := image.Rect(d.Rect.Min.X, d.Rect.Min.Y, arrow.Min.X, d.Rect.Max.Y)
	th := 0
	if f != nil {
		th = f.Height()
	}
	drawEditField(dst, editField{Rect: box, TextH: th}, func(at image.Point) {
		if f != nil && d.Text != "" {
			// A choice wider than the box ends in "..." rather than a cut glyph.
			clip := box.Inset(2)
			f.Draw(dst.SubImage(clip).(*image.RGBA), editorClip(f, d.Text, clip.Max.X-at.X), at.X, at.Y, d.Ink)
		}
	})
	label := "v"
	if d.Open {
		label = "^"
	}
	drawPushButton(dst, f, pushButton{Rect: arrow, Label: label, Literal: true, LabelRaise: labelRaise(f), Focus: d.Focus,
		Hover: pointerOK && pointer.In(arrow), Disabled: d.Disabled})
	if !d.Open || d.List == nil {
		return
	}
	l := d.listBox(f)
	drawFrame(dst, panelFrame(l.Rect.Union(l.Bar()).Inset(-2), d.Fill, d.Ink))
	drawListBox(dst, f, frames, l, d.List, rowText, pointer, pointerOK)
}

// dropdownStep moves the highlighted choice by delta, wrapping at both ends:
// the keyboard path of a closed drop-down.
func dropdownStep(m *Picker, delta int) bool {
	if m == nil || m.Len() == 0 {
		return false
	}
	n := m.Len()
	m.Select(((m.Selection()+delta)%n + n) % n)
	return true
}

// dropdownHit is what a press at p on the drop-down does: open or close it,
// or choose the open list's row.
type dropdownHit uint8

const (
	dropdownMiss dropdownHit = iota
	dropdownToggle
	dropdownRow
	dropdownBar
)

// hit reports the part under p and, for a row, the list index.
func (d dropdown) hit(f *text.Font, p image.Point) (dropdownHit, int) {
	if p.In(d.Rect) {
		return dropdownToggle, 0
	}
	if !d.Open || d.List == nil {
		return dropdownMiss, 0
	}
	l := d.listBox(f)
	if p.In(l.Bar()) {
		return dropdownBar, 0
	}
	if row, ok := l.RowAt(p); ok {
		top, n := d.List.Visible()
		if row < n {
			return dropdownRow, top + row
		}
	}
	return dropdownMiss, 0
}

// labelRaise centres a push button's label of font f on the button: the
// shared painter's default raise suits its own dialogue buttons only.
func labelRaise(f *text.Font) int { return max(1, fontHeight(f, 15)/2) }
