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
	// Icon, when set, draws the closed box as an icon button, sunk while
	// the list is open: a toolbar's drop-down.
	Icon *editorIcon
	// Edit, when set, draws the closed box as that text field: a search
	// field whose open list holds its hits, its text in EditFont.
	Edit               *popupRow
	EditFont           *text.Font
	Press              *buttonPress
	Button, ButtonRows bool
	listLayout         *listBox
	listFont           *text.Font
	allRows            bool
	ListOnly           bool
}

// button is the arrow button at the closed box's right end.
func (d dropdown) button() image.Rectangle {
	w := min(d.Rect.Dx()/2, d.Rect.Dy())
	return image.Rect(d.Rect.Max.X-w, d.Rect.Min.Y, d.Rect.Max.X, d.Rect.Max.Y)
}

func (d dropdown) listBox(f *text.Font) listBox {
	if d.listLayout != nil {
		return *d.listLayout
	}
	rows := d.Rows
	if d.List != nil {
		rows = max(1, min(rows, d.List.Len()))
	}
	r := image.Rect(d.Rect.Min.X, d.Rect.Max.Y+1, d.Rect.Max.X-widgetSpriteSize, d.Rect.Max.Y+1)
	return newListBox(r, max(1, rows), f)
}

func (d dropdown) listFrame(f *text.Font) image.Rectangle {
	l := d.listBox(f)
	return l.Rect.Union(l.Bar()).Inset(-dropdownRim)
}

// Bounds is everything the drop-down covers: the box, and the open list with
// its bar.
func (d dropdown) Bounds(f *text.Font) image.Rectangle {
	if !d.Open {
		return d.Rect
	}
	return d.Rect.Union(d.listFrame(f))
}

// Covers reports whether p lies on the painted box or the open list
// with its rim and bar. A list wider than the box leaves the strip beside the
// box free, which Bounds' rectangle does not.
func (d dropdown) Covers(f *text.Font, p image.Point) bool {
	if !d.ListOnly && p.In(d.Rect) {
		return true
	}
	if !d.Open {
		return false
	}
	return p.In(d.listFrame(f))
}

// drawDropdown is the one drop-down builder: the closed box, an edit field's
// well with the text and an arrow push button, with Icon an icon button or
// with Edit a text field, then, while open, a fill and the shared list box over the drop-down's list
// model.
func drawDropdown(dst *image.RGBA, f *text.Font, frames []*image.RGBA, d dropdown, rowText listRowText, pointer image.Point, pointerOK bool) {
	recordWidget(widgetDropdown, d.Rect, d)
	if dst == nil || d.Rect.Dx() < 8 || d.Rect.Dy() < 4 {
		return
	}
	d.Press.cover(d.Rect)
	switch {
	case d.ListOnly:
	case d.Button:
		drawPushButton(dst, f, pushButton{Rect: d.Rect, Label: d.Text + " v", Literal: true, LabelRaise: labelRaise(f), Hover: pointerOK && pointer.In(d.Rect), Disabled: d.Disabled, Press: d.Press})
	case d.Icon != nil:
		drawIconButton(dst, iconButton{Rect: d.Rect, Icon: d.Icon, On: d.Open, Hover: pointerOK && pointer.In(d.Rect), Disabled: d.Disabled, Press: d.Press})
	case d.Edit != nil:
		drawEditRow(dst, d.EditFont, *d.Edit, d.Rect, popupInk{Ground: d.Fill, Edge: d.Ink, Hint: popupHintInk}, d.Ink)
	default:
		drawDropdownBox(dst, f, d, pointer, pointerOK)
	}
	if !d.Open || d.List == nil {
		return
	}
	l := d.listBox(f)
	frame := d.listFrame(f)
	d.Press.cover(frame)
	drawFrame(dst, panelFrame(frame, d.Fill, d.Ink))
	if d.listFont != nil {
		f = d.listFont
	}
	drawListBox(dst, f, frames, l, d.List, rowText, pointer, pointerOK)
	if d.ButtonRows && d.Press != nil {
		top, count := d.List.Visible()
		for i := range count {
			if d.List.Rows()[top+i].Choosable {
				drawPushButton(dst, f, pushButton{Rect: l.Row(i), FeedbackOnly: true, Press: d.Press})
			}
		}
	}
}

// drawDropdownBox is the closed box: an edit field's well with the text and
// an arrow push button.
func drawDropdownBox(dst *image.RGBA, f *text.Font, d dropdown, pointer image.Point, pointerOK bool) {
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
		Hover: pointerOK && pointer.In(arrow), Disabled: d.Disabled, Press: d.Press})
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
	if d.Open && d.List != nil {
		// The open list is painted over the box, so it is hit first.
		l := d.listBox(f)
		if p.In(l.Bar()) {
			return dropdownBar, 0
		}
		if row, ok := l.RowAt(p); ok {
			if top, n := d.List.Visible(); row < n {
				return dropdownRow, top + row
			}
		}
	}
	if p.In(d.Rect) {
		return dropdownToggle, 0
	}
	return dropdownMiss, 0
}

// labelRaise centres a push button's label of font f on the button: the
// shared painter's default raise suits its own dialogue buttons only.
func labelRaise(f *text.Font) int { return max(1, fontHeight(f, 15)/2) }
