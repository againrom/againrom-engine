package ui

import (
	"image"
	"image/color"
	"image/draw"
	"slices"
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
	Edit         *editorTextView
	Kind         popupRowKind
	Label, Value string
	Ink          color.RGBA // the value's ink; zero is the popup's paper
	Focus        bool       // the keyboard edits this row
	Caret        int        // the caret's pixel offset in a focused edit
	// Buttons are the captions of the row's own buttons, at its right end;
	// a pick row without them carries the popup's PickLabel.
	Buttons []string
	// Lines is a multi-line row's height in rows, zero for six; First is
	// the first text line it shows while it is not typed in.
	Lines, First int
	// Hint is an empty edit row's grey text: the value it stands for.
	Hint string
	// Icons are pictures drawn after the value, left to right, each square
	// at the row's height.
	Icons []*image.RGBA
}

// popupGrid is a grid of picture cells as data: one widget for every grid
// of pictures a popup shows. A cell is drawn pressed while it is the chosen
// one and carries a tick while it is marked. Rows is how many rows show and
// Top the first row shown; the columns follow the popup's width.
type popupGrid struct {
	Cells []popupGridCell
	Side  int // a cell's side in pixels; zero is gridCellSide
	Rows  int
	Top   int
}

// popupGridCell is one cell of a grid.
type popupGridCell struct {
	Icon    *image.RGBA
	Pressed bool
	Ticked  bool
}

// gridCellSide is a grid cell's side when the grid names none.
const gridCellSide = 44

func (g *popupGrid) side() int {
	if g.Side > 0 {
		return g.Side
	}
	return gridCellSide
}

// height is the grid's height in the popup, its gap below included.
func (g *popupGrid) height() int {
	if g == nil {
		return 0
	}
	return max(1, g.Rows)*g.side() + popupPad
}

// columns is how many cells a row of a grid width wide holds, its scroll bar
// aside.
func (g *popupGrid) columns(width int) int {
	return max(1, (width-widgetSpriteSize-2)/g.side())
}

// totalRows is how many rows the grid's cells fill at cols columns.
func (g *popupGrid) totalRows(cols int) int {
	return (len(g.Cells) + cols - 1) / cols
}

// popupSpec is a popup as data. At is its top-left corner in window pixels,
// Width its width; the height follows the content and Bounds clamps both.
// Top is the first row shown and MaxRows how many rows show at most; List,
// when set, is a list box over that model below the rows.
type popupSpec struct {
	Card     *PanelSubject
	CardFont *text.Font
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
	PickArmed bool
	// LabelWidth is the label column; zero is two fifths of the width.
	LabelWidth int
	// Size is the outer size the user dragged the popup to; zero keeps the
	// size the content gives. The popup then spans that size, no smaller than
	// popupMinSize: a list shows as many rows as fit, a row-scrolled popup
	// shows as many rows, a multi-line field takes the free height and the
	// buttons stay on the bottom edge.
	Size image.Point
	// MinWidth caps the least width a sized popup is held to: a popup
	// whose own width was narrower than popupMinWidth keeps that width as
	// its least.
	MinWidth int
	// Grid, when set, stands between the rows and the list.
	Grid *popupGrid
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
	RowBar             image.Rectangle
	Rect, Title, Close image.Rectangle
	Rows               []popupRowRect
	List               listBox
	HasList            bool
	Aside              image.Rectangle // the column ListAside keeps
	Notes              []image.Rectangle
	Buttons            []image.Rectangle
	Pitch              int
	// ListRows and RowsFit are the list rows and the scrolled rows the
	// layout shows: the spec's own, or the ones a sized popup fits.
	ListRows, RowsFit int
	// Grid is the grid's box, GridBar its scroll bar, GridCells the shown
	// cells' boxes from cell GridFirst on, GridCols its columns and GridTop
	// its first shown row.
	Grid, GridBar                image.Rectangle
	GridCells                    []image.Rectangle
	GridFirst, GridCols, GridTop int
}

const (
	popupPad      = 6
	popupRowExtra = 8
	popupButtonW  = 96
)

// multilineRows is a multi-line row's height in rows.
func multilineRows(row popupRow) int {
	if row.Lines > 0 {
		return row.Lines
	}
	return 6
}

func fontHeight(f *text.Font, fallback int) int {
	if f != nil && f.Height() > 0 {
		return f.Height()
	}
	return fallback
}

// popupMinField is the narrowest a row's field is kept in a sized popup.
const popupMinField = 60

// popupMinRows is the fewest list or scrolled rows a sized popup keeps.
const popupMinRows = 3

// popupMinWidth is the narrowest a sized popup gets: the width where its
// buttons and each row's own buttons still fit beside a label and a field.
func popupMinWidth(s popupSpec, body *text.Font) int {
	w := 240
	if n := len(s.Buttons); n > 0 {
		bw := popupButtonW
		if s.ButtonW > 0 {
			bw = s.ButtonW
		}
		w = max(w, n*(bw+popupPad)+popupPad)
	}
	for _, row := range s.Rows {
		caps := rowButtons(s, row)
		if len(caps) == 0 {
			continue
		}
		need := 0
		for _, c := range caps {
			bw := popupButtonW
			if len(row.Buttons) > 0 && body != nil {
				bw = max(48, body.Advance(c)+16)
			}
			need += bw + 4
		}
		// A label, a field of at least popupMinField and the buttons.
		if s.LabelWidth > 0 {
			w = max(w, s.LabelWidth+4+popupMinField+need+2*popupPad)
		} else {
			w = max(w, (4+popupMinField+need)*5/3+2*popupPad)
		}
	}
	if s.Card != nil {
		w += compactPanelW + popupPad
	}
	return w
}

// popupFit is a sized popup's flexible part for outer height h: the list rows,
// the scrolled rows and the extra lines of the first multi-line field, and the
// least height that keeps them usable. s.Rows is not modified.
func popupFit(s popupSpec, body *text.Font, pitch, titleH, h int) (listRows, rowsFit, extraLines, minH int) {
	fixed := titleH + popupPad + len(s.Notes)*pitch + popupPad + s.Grid.height()
	if len(s.Buttons) > 0 {
		fixed += pitch + 4 + popupPad
	}
	top := max(0, min(s.Top, len(s.Rows)))
	rowH := func(r popupRow) int {
		if r.Kind == popupMultiline {
			return multilineRows(r) * pitch
		}
		return pitch
	}
	flexRows := s.MaxRows > 0
	if !flexRows {
		for _, r := range s.Rows[top:] {
			fixed += rowH(r)
		}
	}
	lp, listBase := 0, 0
	if s.List != nil {
		listBase = newListBox(image.Rect(0, 0, 10, 0), 0, body).Rect.Dy()
		lp = newListBox(image.Rect(0, 0, 10, 0), 1, body).Rect.Dy() - listBase
		fixed += listBase + popupPad
	}
	// The least height holds the flexible part at its least rows.
	least := fixed
	if s.List != nil {
		least += popupMinRows * lp
	}
	if flexRows {
		for i, r := range s.Rows[top:] {
			if i == popupMinRows {
				break
			}
			least += rowH(r)
		}
	}
	minH = least
	h = max(h, minH)
	avail := h - fixed
	switch {
	case s.List != nil:
		listRows = max(popupMinRows, avail/lp)
	case flexRows:
		used := 0
		for _, r := range s.Rows[top:] {
			if used+rowH(r) > avail && rowsFit >= popupMinRows {
				break
			}
			used += rowH(r)
			rowsFit++
		}
		rowsFit = min(rowsFit, len(s.Rows)-top)
	default:
		extraLines = max(0, avail/pitch)
	}
	return listRows, rowsFit, extraLines, minH
}

// layoutPopup builds the popup's geometry: title bar with its close button,
// the shown rows, the list box, the note lines and the button row. A spec
// with a Size takes that size (no smaller than its content allows) and
// fits its flexible rows to it.
func layoutPopup(s popupSpec, title, body *text.Font) popupLayout {
	pitch := fontHeight(body, 15) + popupRowExtra
	titleH := fontHeight(title, 15) + popupRowExtra
	w := max(160, s.Width)
	if s.MaxRows > 0 && len(s.Rows) > s.MaxRows {
		w += widgetSpriteSize + 2
	}
	sized := popupResizeEnabled && s.Size.X > 0 && s.Size.Y > 0
	sizedH := 0
	if sized {
		least := popupMinWidth(s, body)
		if s.MinWidth > 0 {
			least = min(least, s.MinWidth)
		}
		w = max(s.Size.X, least)
		if !s.Bounds.Empty() {
			w = min(w, s.Bounds.Dx())
		}
		lr, rf, extra, minH := popupFit(s, body, pitch, titleH, s.Size.Y)
		sizedH = max(s.Size.Y, minH)
		if !s.Bounds.Empty() {
			sizedH = min(sizedH, max(minH, s.Bounds.Dy()))
		}
		lr, rf, extra, _ = popupFit(s, body, pitch, titleH, sizedH)
		if s.List != nil {
			s.ListRows = lr
		}
		if s.MaxRows > 0 {
			s.MaxRows = rf
		}
		if extra > 0 {
			s.Rows = slices.Clone(s.Rows)
			for i := max(0, s.Top); i < len(s.Rows); i++ {
				if s.Rows[i].Kind == popupMultiline {
					s.Rows[i].Lines = multilineRows(s.Rows[i]) + extra
					break
				}
			}
		}
	}
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
		if row := s.Rows[s.Top+i]; row.Kind == popupMultiline {
			h += (multilineRows(row) - 1) * pitch
		}
	}
	h += s.Grid.height()
	var list listBox
	if s.List != nil {
		list = newListBox(image.Rect(0, 0, w-2*popupPad-widgetSpriteSize, 0), max(1, s.ListRows), body)
		h += list.Rect.Dy() + popupPad
	}
	h += len(s.Notes)*pitch + popupPad
	if len(s.Buttons) > 0 {
		h += pitch + 4 + popupPad
	}
	if sized {
		h = max(h, sizedH)
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
	l := popupLayout{Rect: r, Pitch: pitch, ListRows: s.ListRows, RowsFit: s.MaxRows}
	l.Title = image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+titleH)
	l.Close = image.Rect(r.Max.X-titleH, r.Min.Y+2, r.Max.X-2, r.Min.Y+titleH-2)
	contentRight := r.Max.X - popupPad
	if s.Card != nil {
		contentRight -= compactPanelW + popupPad
	}
	if s.MaxRows > 0 && len(s.Rows) > s.MaxRows {
		l.RowBar = image.Rect(contentRight-widgetSpriteSize, l.Title.Max.Y+popupPad, contentRight, l.Title.Max.Y+popupPad+shown*pitch)
		contentRight -= widgetSpriteSize + 2
	}
	labelW := s.LabelWidth
	if labelW <= 0 {
		labelW = (contentRight - r.Min.X - popupPad) * 2 / 5
	}
	y := l.Title.Max.Y + popupPad
	for i := 0; i < shown; i++ {
		row := s.Rows[s.Top+i]
		rowH := pitch
		if row.Kind == popupMultiline {
			rowH = multilineRows(row) * pitch
		}
		rr := popupRowRect{Index: s.Top + i, Rect: image.Rect(r.Min.X+popupPad, y, contentRight, y+rowH)}
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
	if g := s.Grid; g != nil {
		side, shown := g.side(), max(1, g.Rows)
		l.GridCols = g.columns(r.Dx() - 2*popupPad)
		total := g.totalRows(l.GridCols)
		l.GridTop = max(0, min(g.Top, total-shown))
		l.Grid = image.Rect(r.Min.X+popupPad, y, r.Min.X+popupPad+l.GridCols*side, y+shown*side)
		if total > shown {
			l.GridBar = image.Rect(r.Max.X-popupPad-widgetSpriteSize, y, r.Max.X-popupPad, l.Grid.Max.Y)
		}
		l.GridFirst = l.GridTop * l.GridCols
		for i := l.GridFirst; i < len(g.Cells) && i < l.GridFirst+shown*l.GridCols; i++ {
			at := image.Pt(l.Grid.Min.X+(i-l.GridFirst)%l.GridCols*side, y+(i-l.GridFirst)/l.GridCols*side)
			l.GridCells = append(l.GridCells, image.Rectangle{Min: at, Max: at.Add(image.Pt(side, side))}.Inset(1))
		}
		y += g.height()
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
		if sized {
			y = r.Max.Y - popupPad - (pitch + 4)
		}
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
	popupGridPart                    // a grid cell; the index is the cell's
	popupGridBar                     // the grid's scroll bar
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
	if p.In(l.GridBar) {
		return popupGridBar, 0
	}
	for i, c := range l.GridCells {
		if p.In(c) {
			return popupGridPart, l.GridFirst + i
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

// popupEdge is the border of a popup a press lands on, as bits: a corner is
// two of them. A drag from a border resizes the popup.
type popupEdge uint8

const (
	edgeLeft popupEdge = 1 << iota
	edgeRight
	edgeTop
	edgeBottom
)

// popupEdgeBand is how many pixels inside the border count as the border.
const popupEdgeBand = 4

// edgeAt is the border under p, none for a press inside the popup's body or
// on its close button.
func (l popupLayout) edgeAt(p image.Point) popupEdge {
	if !p.In(l.Rect) || p.In(l.Close) {
		return 0
	}
	var e popupEdge
	switch {
	case p.X < l.Rect.Min.X+popupEdgeBand:
		e |= edgeLeft
	case p.X >= l.Rect.Max.X-popupEdgeBand:
		e |= edgeRight
	}
	switch {
	case p.Y < l.Rect.Min.Y+popupEdgeBand:
		e |= edgeTop
	case p.Y >= l.Rect.Max.Y-popupEdgeBand:
		e |= edgeBottom
	}
	return e
}

// popupInk is the editor popup's palette.
// popupHintInk is the grey of an empty field's hint.
var popupHintInk = color.RGBA{0x8a, 0x80, 0x6c, 255}

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
	if !l.RowBar.Empty() {
		drawVScrollBar(dst, frames, vScrollBar{Rect: l.RowBar, Pos: s.Top, Count: len(s.Rows) - s.MaxRows + 1}.withPointer(pointer, pointerOK))
	}
	if s.Card != nil {
		font := s.CardFont
		if font == nil {
			font = body
		}
		start := markCapture()
		if pic := RenderCharacterPanel(CompactPanelLayout(nil), font, *s.Card); pic != nil {
			at := image.Pt(l.Rect.Max.X-popupPad-compactPanelW, l.Title.Max.Y+popupPad)
			draw.Draw(dst, pic.Bounds().Add(at), pic, pic.Bounds().Min, draw.Over)
			shiftCapture(start, at)
		}
	}
	drawFrame(dst, panelFrame(l.Title, ink.TitleGround, ink.Edge))
	drawPopupGrip(dst, l.Rect, ink.Edge)
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
		if row.Edit != nil {
			drawEditorInput(dst, body, row, rr.Control, ink, value)
			drawRowIcons(dst, body, l, rr, row)
			for k, b := range rr.Buttons {
				drawPushButton(dst, body, pushButton{Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Focus: row.Kind == popupPick && s.PickArmed, Hover: pointerOK && pointer.In(b)})
			}
			continue
		}
		switch row.Kind {
		case popupMultiline:
			drawFrame(dst, panelFrame(rr.Control, ink.Ground, ink.Edge))
			if body != nil {
				var lines []string
				for _, line := range strings.Split(row.Value, "\n") {
					lines = append(lines, editorWrap(body, line, rr.Control.Dx()-8)...)
				}
				visible := max(1, (rr.Control.Dy()-8)/fontHeight(body, 15))
				start := max(0, min(row.First, len(lines)-visible))
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
				if body == nil {
					return
				}
				if row.Value == "" && row.Hint != "" {
					body.Draw(dst.SubImage(rr.Control.Inset(2)).(*image.RGBA), row.Hint, at.X, at.Y, popupHintInk)
					return
				}
				body.Draw(dst.SubImage(rr.Control.Inset(2)).(*image.RGBA), row.Value, at.X, at.Y, value)
			})
		case popupSwitch:
			// A long value ends in an ellipsis inside its button.
			drawPushButton(dst, body, pushButton{Rect: rr.Control, Label: editorClip(body, row.Value, rr.Control.Dx()-12), Literal: true, LabelRaise: labelRaise(body), Focus: row.Focus, Hover: pointerOK && pointer.In(rr.Control)})
		case popupChoice:
			drawDropdown(dst, body, frames, dropdown{Rect: rr.Control, Text: row.Value, Focus: row.Focus, Fill: ink.Ground, Ink: value}, nil, pointer, pointerOK)
		default:
			bodyText(rr.Control, row.Value, value)
		}
		drawRowIcons(dst, body, l, rr, row)
		for k, b := range rr.Buttons {
			drawPushButton(dst, body, pushButton{Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Focus: row.Kind == popupPick && s.PickArmed, Hover: pointerOK && pointer.In(b)})
		}
	}
	if g := s.Grid; g != nil {
		drawPopupGrid(dst, body, frames, g, l, ink, pointer, pointerOK)
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

// drawRowIcons draws a row's pictures after its value, each square at the
// control's height, as many as the control holds.
func drawRowIcons(dst *image.RGBA, body *text.Font, l popupLayout, rr popupRowRect, row popupRow) {
	if len(row.Icons) == 0 || rr.Control.Empty() {
		return
	}
	x := rr.Control.Min.X + 2
	if body != nil && row.Value != "" {
		x += body.Advance(row.Value) + popupPad
	}
	side := rr.Control.Dy()
	clip := dst.SubImage(rr.Control.Intersect(l.Rect)).(*image.RGBA)
	for _, icon := range row.Icons {
		if x+side > rr.Control.Max.X {
			return
		}
		if icon != nil && !icon.Bounds().Empty() {
			editorBlit(clip, icon, addPreviewRect(icon.Bounds().Size(), image.Rect(x, rr.Control.Min.Y, x+side, rr.Control.Max.Y)), false)
		}
		x += side + 2
	}
}

// drawPopupGrid paints a grid's shown cells, each a push button holding its
// picture, sunk and framed while chosen and ticked while marked, and the
// grid's bar.
func drawPopupGrid(dst *image.RGBA, body *text.Font, frames []*image.RGBA, g *popupGrid, l popupLayout, ink popupInk, pointer image.Point, pointerOK bool) {
	for i, c := range l.GridCells {
		cell := g.Cells[l.GridFirst+i]
		drawPushButton(dst, body, pushButton{Rect: c, Literal: true, Pressed: cell.Pressed, Inside: cell.Pressed, Hover: pointerOK && pointer.In(c)})
		if cell.Pressed {
			drawFrame(dst, panelFrame(c, ink.TitleGround, ink.Edge))
		}
		if icon := cell.Icon; icon != nil && !icon.Bounds().Empty() {
			box := c.Inset(4)
			if cell.Pressed {
				box = box.Add(image.Pt(1, 1))
			}
			editorBlit(dst.SubImage(c.Inset(2)).(*image.RGBA), icon, addPreviewRect(icon.Bounds().Size(), box), false)
		}
		if cell.Ticked {
			drawGridTick(dst, c, ink.Edge)
		}
	}
	if !l.GridBar.Empty() {
		bar := vScrollBar{Rect: l.GridBar, Pos: l.GridTop, Count: g.totalRows(l.GridCols) - max(1, g.Rows) + 1}
		drawVScrollBar(dst, frames, bar.withPointer(pointer, pointerOK))
	}
}

// drawGridTick draws a tick in the top-right corner of cell c.
func drawGridTick(dst *image.RGBA, c image.Rectangle, ink color.RGBA) {
	x, y := c.Max.X-13, c.Min.Y+6
	for i := 0; i < 10; i++ {
		py := y - i/2 + 2
		if i < 3 {
			py = y + i
		}
		for d := 0; d < 2; d++ {
			if p := image.Pt(x+i, py+d); p.In(dst.Bounds()) {
				dst.SetRGBA(p.X, p.Y, ink)
			}
		}
	}
}

// drawPopupGrip marks the bottom-right corner where a drag resizes the popup:
// three short diagonal strokes.
func drawPopupGrip(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	for k := 1; k <= 3; k++ {
		for i := 0; i < 2*k; i++ {
			p := image.Pt(r.Max.X-3-2*k+i+1, r.Max.Y-3-i)
			if p.In(dst.Bounds()) {
				dst.SetRGBA(p.X, p.Y, c)
			}
		}
	}
}
