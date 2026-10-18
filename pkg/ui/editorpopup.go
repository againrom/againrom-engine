package ui

import (
	"image"
	"image/color"
	"strings"

	"againrom/pkg/render/text"
)

// The editor's popup window: one builder takes a popup description as data
// (title, rows of typed controls, an optional list, notes and buttons) and
// produces one layout that both the painter and the hit test read, so a
// control is drawn where it is pressed.

// popupRowKind is the control a popup row carries.
type popupRowKind uint8

const (
	popupText    popupRowKind = iota // a label and a read-only value
	popupHeading                     // a section heading across the row
	popupEdit                        // a label and an edit field
	popupChoice                      // a label and a drop-down
	popupPick                        // a label, an edit field and a pick-on-map button
	popupLink                        // a label and a value that opens what it names
	popupSwitch                      // a label and a button that switches its value
	popupMultiline
)

// popupRow is one row of a popup.
type popupRow struct {
	Kind         popupRowKind
	Label, Value string
	Ink          color.RGBA // the value's ink; zero is the popup's paper
	Focus        bool       // the keyboard edits this row
	Caret        int        // the caret's pixel offset in a focused edit
	// Buttons are the captions of the row's own buttons, at its right end;
	// a pick row without them carries the popup's PickLabel.
	Buttons []string
}

// popupSpec is a popup as data. At is its top-left corner in window pixels,
// Width its width; the height follows the content and Bounds clamps both.
// Top is the first row shown and MaxRows how many rows show at most; List,
// when set, is a list box over that model below the rows.
type popupSpec struct {
	Title    string
	At       image.Point
	Width    int
	Bounds   image.Rectangle
	Rows     []popupRow
	Top      int
	MaxRows  int
	List     *Picker
	ListRows int
	ListText listRowText
	// ListAside is a column right of the list kept free for a picture.
	ListAside int
	Notes     []string
	NoteInk   color.RGBA
	Buttons   []string
	Disabled  []bool
	// ButtonW is the button row's button width; zero is the default.
	ButtonW int
	// PickLabel is the pick-on-map button's caption.
	PickLabel string
	// LabelWidth is the label column; zero is two fifths of the width.
	LabelWidth int
}

// popupRowRect is the geometry of one shown row.
type popupRowRect struct {
	Index                int
	Rect, Label, Control image.Rectangle
	Button               image.Rectangle   // the first row button; empty otherwise
	Buttons              []image.Rectangle // every row button, left to right
}

// popupLayout is a built popup: every rectangle the painter fills and the
// hit test reads.
type popupLayout struct {
	Rect, Title, Close image.Rectangle
	Rows               []popupRowRect
	List               listBox
	HasList            bool
	Aside              image.Rectangle // the column ListAside keeps
	Notes              []image.Rectangle
	Buttons            []image.Rectangle
	Pitch              int
}

const (
	popupPad      = 6
	popupRowExtra = 8
	popupButtonW  = 96
)

func fontHeight(f *text.Font, fallback int) int {
	if f != nil && f.Height() > 0 {
		return f.Height()
	}
	return fallback
}

// layoutPopup builds the popup's geometry: title bar with its close button,
// the shown rows, the list box, the note lines and the button row.
func layoutPopup(s popupSpec, title, body *text.Font) popupLayout {
	pitch := fontHeight(body, 15) + popupRowExtra
	titleH := fontHeight(title, 15) + popupRowExtra
	w := max(160, s.Width)
	if !s.Bounds.Empty() {
		w = min(w, s.Bounds.Dx())
	}
	shown := len(s.Rows) - s.Top
	if s.MaxRows > 0 {
		shown = min(shown, s.MaxRows)
	}
	shown = max(0, shown)
	h := titleH + popupPad + shown*pitch
	for i := 0; i < shown; i++ {
		if s.Rows[s.Top+i].Kind == popupMultiline {
			h += 5 * pitch
		}
	}
	var list listBox
	if s.List != nil {
		list = newListBox(image.Rect(0, 0, w-2*popupPad-widgetSpriteSize, 0), max(1, s.ListRows), body)
		h += list.Rect.Dy() + popupPad
	}
	h += len(s.Notes)*pitch + popupPad
	if len(s.Buttons) > 0 {
		h += pitch + 4 + popupPad
	}
	r := image.Rectangle{Min: s.At, Max: s.At.Add(image.Pt(w, h))}
	if !s.Bounds.Empty() {
		if r.Max.X > s.Bounds.Max.X {
			r = r.Add(image.Pt(s.Bounds.Max.X-r.Max.X, 0))
		}
		if r.Max.Y > s.Bounds.Max.Y {
			r = r.Add(image.Pt(0, s.Bounds.Max.Y-r.Max.Y))
		}
		if r.Min.X < s.Bounds.Min.X {
			r = r.Add(image.Pt(s.Bounds.Min.X-r.Min.X, 0))
		}
		if r.Min.Y < s.Bounds.Min.Y {
			r = r.Add(image.Pt(0, s.Bounds.Min.Y-r.Min.Y))
		}
	}
	l := popupLayout{Rect: r, Pitch: pitch}
	l.Title = image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+titleH)
	l.Close = image.Rect(r.Max.X-titleH, r.Min.Y+2, r.Max.X-2, r.Min.Y+titleH-2)
	labelW := s.LabelWidth
	if labelW <= 0 {
		labelW = (w - 2*popupPad) * 2 / 5
	}
	y := l.Title.Max.Y + popupPad
	for i := 0; i < shown; i++ {
		row := s.Rows[s.Top+i]
		rowH := pitch
		if row.Kind == popupMultiline {
			rowH = 6 * pitch
		}
		rr := popupRowRect{Index: s.Top + i, Rect: image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad, y+rowH)}
		rr.Label = image.Rect(rr.Rect.Min.X, y, rr.Rect.Min.X+labelW, y+pitch)
		rr.Control = image.Rect(rr.Label.Max.X+4, y+1, rr.Rect.Max.X, y+pitch-1)
		if row.Kind == popupMultiline {
			rr.Control.Max.Y = y + rowH - 1
		}
		switch row.Kind {
		case popupHeading:
			rr.Label, rr.Control = rr.Rect, image.Rectangle{}
		case popupText, popupLink:
			if row.Value == "" { // a line of text runs across the row
				rr.Label, rr.Control = rr.Rect, image.Rectangle{}
			}
		}
		if caps := rowButtons(s, row); len(caps) > 0 {
			x := rr.Rect.Max.X
			rr.Buttons = make([]image.Rectangle, len(caps))
			for k := len(caps) - 1; k >= 0; k-- {
				w := popupButtonW
				if len(row.Buttons) > 0 && body != nil {
					w = max(48, body.Advance(caps[k])+16)
				}
				rr.Buttons[k] = image.Rect(x-w, y+1, x, y+pitch-1)
				x -= w + 4
			}
			rr.Button = rr.Buttons[0]
			if rr.Control.Empty() {
				rr.Label.Max.X = min(rr.Label.Max.X, x)
			} else {
				rr.Control.Max.X = x
			}
		}
		l.Rows = append(l.Rows, rr)
		y += rowH
	}
	if s.List != nil {
		l.List = newListBox(image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad-widgetSpriteSize-s.ListAside, y), max(1, s.ListRows), body)
		l.HasList = true
		if s.ListAside > 0 {
			l.Aside = image.Rect(r.Max.X-popupPad-s.ListAside+popupPad, y, r.Max.X-popupPad, l.List.Rect.Max.Y)
		}
		y = l.List.Rect.Max.Y + popupPad
	}
	for range s.Notes {
		l.Notes = append(l.Notes, image.Rect(r.Min.X+popupPad, y, r.Max.X-popupPad, y+pitch))
		y += pitch
	}
	if n := len(s.Buttons); n > 0 {
		y += popupPad
		x := r.Max.X - popupPad
		bw := popupButtonW
		if s.ButtonW > 0 {
			bw = s.ButtonW
		}
		l.Buttons = make([]image.Rectangle, n)
		for i := n - 1; i >= 0; i-- {
			l.Buttons[i] = image.Rect(x-bw, y, x, y+pitch+4)
			x -= bw + popupPad
		}
	}
	return l
}

// rowButtons are the captions of row's buttons.
func rowButtons(s popupSpec, row popupRow) []string {
	if len(row.Buttons) > 0 {
		return row.Buttons
	}
	if row.Kind == popupPick {
		return []string{s.PickLabel}
	}
	return nil
}

// rowButtonAt is the row and the button of that row under p.
func (l popupLayout) rowButtonAt(p image.Point) (row, button int, ok bool) {
	for _, r := range l.Rows {
		for k, b := range r.Buttons {
			if p.In(b) {
				return r.Index, k, true
			}
		}
	}
	return 0, 0, false
}

// popupPart is what a press on a popup lands on.
type popupPart uint8

const (
	popupMiss       popupPart = iota // outside the popup
	popupBody                        // the popup's ground: consumed, no action
	popupTitle                       // the title bar: a drag moves the popup
	popupClose                       // the close button
	popupRowLine                     // a row's label or value
	popupRowControl                  // a row's control
	popupRowButton                   // a row's own button
	popupListRow                     // a list row
	popupListBar                     // the list's scroll bar
	popupButton                      // a button of the button row
)

// hit reports the part under p and its row, list or button index.
func (l popupLayout) hit(p image.Point) (popupPart, int) {
	if !p.In(l.Rect) {
		return popupMiss, 0
	}
	if p.In(l.Close) {
		return popupClose, 0
	}
	if p.In(l.Title) {
		return popupTitle, 0
	}
	if row, _, ok := l.rowButtonAt(p); ok {
		return popupRowButton, row
	}
	for _, r := range l.Rows {
		switch {
		case p.In(r.Control):
			return popupRowControl, r.Index
		case p.In(r.Rect):
			return popupRowLine, r.Index
		}
	}
	if l.HasList {
		if p.In(l.List.Bar()) {
			return popupListBar, 0
		}
		if row, ok := l.List.RowAt(p); ok {
			return popupListRow, row
		}
	}
	for i, b := range l.Buttons {
		if p.In(b) {
			return popupButton, i
		}
	}
	return popupBody, 0
}

// popupInk is the editor popup's palette.
type popupInk struct {
	Ground, Edge, TitleGround, Title, Label, Value color.RGBA
}

// drawPopup is the one editor popup painter: the frame, the title bar and its
// close button, each shown row's label and control, the list box, the notes
// and the buttons, each through the widget kit's builder.
func drawPopup(dst *image.RGBA, title, body *text.Font, frames []*image.RGBA, s popupSpec, l popupLayout, ink popupInk, pointer image.Point, pointerOK bool) {
	if dst == nil {
		return
	}
	drawFrame(dst, panelFrame(l.Rect, ink.Ground, ink.Edge))
	drawFrame(dst, panelFrame(l.Title, ink.TitleGround, ink.Edge))
	if title != nil {
		clip := dst.SubImage(image.Rect(l.Title.Min.X+popupPad, l.Title.Min.Y, l.Close.Min.X-2, l.Title.Max.Y)).(*image.RGBA)
		title.Draw(clip, s.Title, l.Title.Min.X+popupPad, l.Title.Min.Y+popupRowExtra/2, ink.Title)
	}
	drawPushButton(dst, body, pushButton{Rect: l.Close, Label: "x", Literal: true, LabelRaise: labelRaise(body), Hover: pointerOK && pointer.In(l.Close)})
	bodyText := func(r image.Rectangle, s string, c color.RGBA) {
		if body == nil || s == "" {
			return
		}
		body.Draw(dst.SubImage(r.Intersect(l.Rect)).(*image.RGBA), s, r.Min.X+2, r.Min.Y+popupRowExtra/2, c)
	}
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		value := row.Ink
		if value == (color.RGBA{}) {
			value = ink.Value
		}
		if row.Kind == popupHeading {
			if title != nil {
				title.Draw(dst.SubImage(rr.Rect).(*image.RGBA), row.Label, rr.Rect.Min.X+2, rr.Rect.Min.Y+popupRowExtra/2, ink.Title)
			}
			continue
		}
		bodyText(rr.Label, row.Label, ink.Label)
		switch row.Kind {
		case popupMultiline:
			drawFrame(dst, panelFrame(rr.Control, ink.Ground, ink.Edge))
			if body != nil {
				var lines []string
				for _, line := range strings.Split(row.Value, "\n") {
					lines = append(lines, editorWrap(body, line, rr.Control.Dx()-8)...)
				}
				visible := max(1, (rr.Control.Dy()-8)/fontHeight(body, 15))
				start := 0
				if row.Focus {
					start = max(0, len(lines)-visible)
				}
				clip := dst.SubImage(rr.Control.Inset(3)).(*image.RGBA)
				y := rr.Control.Min.Y + 4
				for _, line := range lines[start:min(len(lines), start+visible)] {
					body.Draw(clip, line, rr.Control.Min.X+4, y, value)
					y += fontHeight(body, 15)
				}
				if row.Focus {
					x := min(rr.Control.Max.X-4, rr.Control.Min.X+4+body.Advance(lines[len(lines)-1]))
					for yy := y - fontHeight(body, 15); yy < y; yy++ {
						if image.Pt(x, yy).In(clip.Bounds()) {
							clip.SetRGBA(x, yy, value)
						}
					}
				}
			}
		case popupEdit, popupPick:
			drawEditField(dst, editField{Rect: rr.Control, TextH: fontHeight(body, 15), Focus: row.Focus, Phase: true, Caret: row.Caret}, func(at image.Point) {
				if body != nil {
					body.Draw(dst.SubImage(rr.Control.Inset(2)).(*image.RGBA), row.Value, at.X, at.Y, value)
				}
			})
		case popupSwitch:
			// A long value ends in an ellipsis inside its button.
			drawPushButton(dst, body, pushButton{Rect: rr.Control, Label: editorClip(body, row.Value, rr.Control.Dx()-12), Literal: true, LabelRaise: labelRaise(body), Focus: row.Focus, Hover: pointerOK && pointer.In(rr.Control)})
		case popupChoice:
			drawDropdown(dst, body, frames, dropdown{Rect: rr.Control, Text: row.Value, Focus: row.Focus, Fill: ink.Ground, Ink: value}, nil, pointer, pointerOK)
		default:
			bodyText(rr.Control, row.Value, value)
		}
		for k, b := range rr.Buttons {
			drawPushButton(dst, body, pushButton{Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Hover: pointerOK && pointer.In(b)})
		}
	}
	if l.HasList && s.ListText != nil {
		drawListBox(dst, body, frames, l.List, s.List, s.ListText, pointer, pointerOK)
	}
	for i, r := range l.Notes {
		bodyText(r, s.Notes[i], s.NoteInk)
	}
	for i, r := range l.Buttons {
		disabled := i < len(s.Disabled) && s.Disabled[i]
		drawPushButton(dst, body, pushButton{Rect: r, Label: s.Buttons[i], Literal: true, LabelRaise: labelRaise(body), Disabled: disabled, Hover: pointerOK && pointer.In(r)})
	}
}
