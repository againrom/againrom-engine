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
	// popupCard heads a card: a grip with the card's number (its label), a
	// drop-down (its value) and a small cross (its button); the card holds
	// the Span rows after it. A drag from the grip moves the card.
	popupCard
)

// popupRow is one row of a popup.
type popupRow struct {
	Column       int
	Edit         *editorTextView
	Kind         popupRowKind
	Label, Value string
	Ink          color.RGBA // the value's ink; zero is the popup's paper
	Focus        bool       // the keyboard edits this row
	Caret        int        // the caret's pixel offset in a focused edit
	// Buttons are the captions of the row's own buttons, at its right end;
	// a pick row without them carries the popup's PickLabel. Disabled marks
	// the buttons drawn greyed that take no press.
	Buttons  []string
	Disabled []bool
	// Lines is a multi-line row's height in rows, zero for six; First is
	// the first text line it shows while it is not typed in.
	Lines, First int
	// Hint is an empty edit row's grey text: the value it stands for.
	Hint string
	// Icons are pictures drawn after the value, left to right, each square
	// at the row's height.
	Icons []*image.RGBA
	// Span is how many rows after a card row its card holds. Struck marks a
	// removed card: a strip with its label and buttons, no grip or value.
	Span   int
	Struck bool
}

// popupRowDrag is a card held by its grip: Row is its card row and At the
// pointer.
type popupRowDrag struct {
	Row int
	At  image.Point
}

// popupGrid is a grid of picture cells as data: one widget for every grid
// of pictures a popup shows. A cell is drawn pressed while it is the chosen
// one and carries a tick while it is marked. Rows is how many rows show and
// Top the first row shown; the columns follow the popup's width unless Cols
// fixes them.
type popupGrid struct {
	Cells []popupGridCell
	Side  int // a cell's side in pixels; zero is gridCellSide
	Rows  int
	Top   int
	Cols  int
	// RowHeads and ColHeads head a matrix: RowHeads[r] stands left of grid
	// row r in a column HeadW pixels wide, ColHeads[c] above column c in a
	// row one cell high. With HeadH set the column heads stand upright,
	// reading from bottom to top, in a row HeadH pixels high, and the hover
	// line over one shows it whole.
	RowHeads, ColHeads []string
	HeadW, HeadH       int
}

// popupGridCell is one cell of a grid. Label is drawn on a cell without a
// picture; an Inert cell is drawn flat and takes no press. Tip is the hover
// line over the cell, empty for none.
type popupGridCell struct {
	Icon    *image.RGBA
	Label   string
	Tip     string
	Pressed bool
	Ticked  bool
	Inert   bool
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
	return max(1, g.Rows)*g.side() + g.headHeight() + popupPad
}

// headHeight is the column heads' row height: HeadH for upright heads, one
// cell for flat ones, 0 when the columns carry none.
func (g *popupGrid) headHeight() int {
	switch {
	case len(g.ColHeads) == 0:
		return 0
	case g.HeadH > 0:
		return g.HeadH
	}
	return g.side()
}

// columns is how many cells a row of a grid width wide holds, its scroll bar
// aside.
func (g *popupGrid) columns(width int) int {
	if g.Cols > 0 {
		return g.Cols
	}
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
	ColumnTops  []int
	ButtonFocus *int
	Columns     int
	Tabs        []string
	ActiveTab   int
	Card        *PanelSubject
	CardFont    *text.Font
	CardAside   int
	Title       string
	At          image.Point
	Width       int
	Bounds      image.Rectangle
	// Slack lets a dragged popup hang past the bounds, keeping its title row
	// and popupKeepWide pixels of its width inside.
	Slack    bool
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
	// Drag is the card a row drag holds, nil for none.
	Drag *popupRowDrag
}

// popupRowRect is the geometry of one shown row.
type popupRowRect struct {
	Index                int
	Rect, Label, Control image.Rectangle
	Button               image.Rectangle   // the first row button; empty otherwise
	Buttons              []image.Rectangle // every row button, left to right
	Off                  []bool            // the row's Disabled buttons
	// Grip is a card row's drag handle and Card the card's whole box.
	Grip, Card image.Rectangle
}

// popupLayout is a built popup: every rectangle the painter fills and the
// hit test reads.
type popupLayout struct {
	Tabs               []image.Rectangle
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
	// GridRowHeads and GridColHeads are the shown headings' boxes, from
	// row GridTop and column 0 on; GridInert marks the shown cells that
	// take no press.
	GridRowHeads, GridColHeads []image.Rectangle
	GridInert                  []bool
	// Drop is the card, counted among the spec's card rows, that a held card
	// lands before, the card count for after the last, -1 for no drag;
	// Marker is the bar drawn at that gap while the drop moves the card.
	Drop   int
	Marker image.Rectangle
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
		w += max(compactPanelW, s.CardAside) + popupPad
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
	if s.Columns > 1 && s.MaxRows > 0 {
		minH = fixed + popupMinRows*pitch
		return 0, min(popupRowCount(s)-s.Top, max(popupMinRows, (h-fixed)/pitch)), 0, minH
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
	if len(s.Tabs) > 0 {
		titleH += pitch
	}
	w := max(160, s.Width)
	if s.MaxRows > 0 && popupRowCount(s) > s.MaxRows {
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
	shown := popupRowCount(s) - s.Top
	if len(s.ColumnTops) > 0 {
		shown = 0
		for i := range s.Rows {
			column, line := popupRowPosition(s, i)
			shown = max(shown, line-popupColumnTop(s, column)+1)
		}
	}
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
	if !s.Bounds.Empty() && s.Slack {
		r = slackInside(r, s.Bounds, titleH)
	} else if !s.Bounds.Empty() {
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
	if len(s.Tabs) > 0 {
		l.Title.Max.Y -= pitch
		for i := range s.Tabs {
			left := r.Min.X + popupPad + i*(w-2*popupPad)/len(s.Tabs)
			right := r.Min.X + popupPad + (i+1)*(w-2*popupPad)/len(s.Tabs)
			l.Tabs = append(l.Tabs, image.Rect(left, l.Title.Max.Y, right, l.Title.Max.Y+pitch))
		}
	}
	l.Close = image.Rect(r.Max.X-l.Title.Dy(), r.Min.Y+2, r.Max.X-2, l.Title.Max.Y-2)
	contentRight := r.Max.X - popupPad
	if s.Card != nil {
		contentRight -= max(compactPanelW, s.CardAside) + popupPad
	}
	contentY := r.Min.Y + titleH + popupPad
	if s.MaxRows > 0 && popupRowCount(s) > s.MaxRows {
		l.RowBar = image.Rect(contentRight-widgetSpriteSize, contentY, contentRight, contentY+shown*pitch)
		contentRight -= widgetSpriteSize + 2
	}
	labelW := s.LabelWidth
	if labelW <= 0 {
		labelW = (contentRight - r.Min.X - popupPad) * 2 / 5
	}
	y := contentY
	for _, index := range popupShownRows(s, shown) {
		row := s.Rows[index]
		left, right := r.Min.X+popupPad, contentRight
		if s.Columns > 1 {
			column, line := popupRowPosition(s, index)
			columnW := (contentRight - r.Min.X - popupPad) / s.Columns
			left, right = left+column*columnW, left+(column+1)*columnW-popupPad
			y = contentY + (line-popupColumnTop(s, column))*pitch
		}
		rowH := pitch
		if row.Kind == popupMultiline {
			rowH = multilineRows(row) * pitch
		}
		rr := popupRowRect{Index: index, Rect: image.Rect(left, y, right, y+rowH)}
		rowLabelW := labelW
		if s.Columns > 1 {
			rowLabelW = rr.Rect.Dx() * 2 / 5
		}
		rr.Label = image.Rect(rr.Rect.Min.X, y, rr.Rect.Min.X+rowLabelW, y+pitch)
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
		case popupCard:
			if row.Struck {
				rr.Label, rr.Control = rr.Rect, image.Rectangle{}
			} else {
				rr.Grip = rr.Label
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
				if row.Kind == popupCard && !row.Struck {
					w = pitch - 2 // the cross: a small square
				}
				rr.Buttons[k] = image.Rect(x-w, y+1, x, y+pitch-1)
				x -= w + 4
			}
			rr.Button, rr.Off = rr.Buttons[0], row.Disabled
			if rr.Control.Empty() {
				rr.Label.Max.X = min(rr.Label.Max.X, x)
			} else {
				rr.Control.Max.X = x
			}
		}
		l.Rows = append(l.Rows, rr)
		y += rowH
	}
	if s.Columns > 1 {
		y = contentY + shown*pitch
	}
	l.Drop = -1
	layoutCards(s, &l)
	if g := s.Grid; g != nil {
		side, shown, top := g.side(), max(1, g.Rows), y
		l.GridCols = g.columns(r.Dx() - 2*popupPad - g.HeadW)
		total := g.totalRows(l.GridCols)
		l.GridTop = max(0, min(g.Top, total-shown))
		x := r.Min.X + popupPad + g.HeadW
		for c := 0; c < len(g.ColHeads) && c < l.GridCols; c++ {
			l.GridColHeads = append(l.GridColHeads, image.Rect(x+c*side, y, x+(c+1)*side, y+g.headHeight()))
		}
		y += g.headHeight()
		for k := 0; k < shown && l.GridTop+k < len(g.RowHeads); k++ {
			l.GridRowHeads = append(l.GridRowHeads, image.Rect(r.Min.X+popupPad, y+k*side, x, y+(k+1)*side))
		}
		l.Grid = image.Rect(x, y, x+l.GridCols*side, y+shown*side)
		if total > shown {
			l.GridBar = image.Rect(r.Max.X-popupPad-widgetSpriteSize, y, r.Max.X-popupPad, l.Grid.Max.Y)
		}
		l.GridFirst = l.GridTop * l.GridCols
		for i := l.GridFirst; i < len(g.Cells) && i < l.GridFirst+shown*l.GridCols; i++ {
			at := image.Pt(l.Grid.Min.X+(i-l.GridFirst)%l.GridCols*side, y+(i-l.GridFirst)/l.GridCols*side)
			l.GridCells = append(l.GridCells, image.Rectangle{Min: at, Max: at.Add(image.Pt(side, side))}.Inset(1))
			l.GridInert = append(l.GridInert, g.Cells[i].Inert)
		}
		y = top + g.height()
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

// layoutCards gives each shown card row its card's box over the rows it
// holds and, while a card is held, the card it lands before and the bar
// that marks that gap.
func layoutCards(s popupSpec, l *popupLayout) {
	ordinal := map[int]int{}
	held, cards := -1, 0
	for i, row := range s.Rows {
		if row.Kind != popupCard {
			continue
		}
		ordinal[i] = cards
		if s.Drag != nil && s.Drag.Row == i {
			held = cards
		}
		cards++
	}
	var shown []int
	for i, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if row.Kind != popupCard {
			continue
		}
		card := rr.Rect
		for k := 1; k <= row.Span && i+k < len(l.Rows) && l.Rows[i+k].Index == rr.Index+k; k++ {
			card = card.Union(l.Rows[i+k].Rect)
		}
		l.Rows[i].Card = card
		shown = append(shown, i)
	}
	if held < 0 || len(shown) == 0 {
		return
	}
	at := s.Drag.At.Y
	last := l.Rows[shown[len(shown)-1]]
	y := last.Card.Max.Y
	l.Drop = ordinal[last.Index] + 1
	for _, i := range shown {
		if c := l.Rows[i].Card; at < (c.Min.Y+c.Max.Y)/2 {
			l.Drop, y = ordinal[l.Rows[i].Index], c.Min.Y
			break
		}
	}
	if l.Drop != held && l.Drop != held+1 {
		l.Marker = image.Rect(last.Rect.Min.X, y-2, last.Rect.Max.X, y+1)
	}
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
			if p.In(b) && !(k < len(r.Off) && r.Off[k]) {
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
	popupRowGrip                     // a card's grip: a drag moves the card
	popupTab
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
	for i, tab := range l.Tabs {
		if p.In(tab) {
			return popupTab, i
		}
	}
	if row, _, ok := l.rowButtonAt(p); ok {
		return popupRowButton, row
	}
	for _, r := range l.Rows {
		switch {
		case p.In(r.Grip):
			return popupRowGrip, r.Index
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
		if p.In(c) && !l.GridInert[i] {
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

// popupEdgeBand is how many pixels inside the border count as the border;
// popupCorner is the side of the bottom-right square that resizes both ways.
const (
	popupEdgeBand = 8
	popupCorner   = 16
	// popupTopBand is the top border's band; the title row below it drags.
	popupTopBand  = 4
	popupKeepWide = 48
)

// slackInside moves r the least that keeps its title row inside bounds and at
// least popupKeepWide pixels of its width inside them.
func slackInside(r, bounds image.Rectangle, titleH int) image.Rectangle {
	keep := min(popupKeepWide, r.Dx())
	dx, dy := 0, 0
	if r.Max.X < bounds.Min.X+keep {
		dx = bounds.Min.X + keep - r.Max.X
	} else if r.Min.X > bounds.Max.X-keep {
		dx = bounds.Max.X - keep - r.Min.X
	}
	if r.Min.Y < bounds.Min.Y {
		dy = bounds.Min.Y - r.Min.Y
	} else if r.Min.Y+titleH > bounds.Max.Y {
		dy = bounds.Max.Y - titleH - r.Min.Y
	}
	return r.Add(image.Pt(dx, dy))
}

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
	case p.Y < l.Rect.Min.Y+popupTopBand:
		e |= edgeTop
	case p.Y >= l.Rect.Max.Y-popupEdgeBand:
		e |= edgeBottom
	}
	if p.X >= l.Rect.Max.X-popupCorner && p.Y >= l.Rect.Max.Y-popupCorner {
		return edgeRight | edgeBottom
	}
	// A row, list or button that reaches into the band keeps its press.
	if part, _ := l.hit(p); part != popupBody && part != popupTitle {
		return 0
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
		drawVScrollBar(dst, frames, vScrollBar{Rect: l.RowBar, Pos: s.Top, Count: popupRowCount(s) - s.MaxRows + 1}.withPointer(pointer, pointerOK))
	}
	if s.Card != nil {
		font := s.CardFont
		if font == nil {
			font = body
		}
		start := markCapture()
		if pic := RenderCharacterPanel(CompactPanelLayout(nil), font, *s.Card); pic != nil {
			at := image.Pt(l.Rect.Max.X-popupPad-compactPanelW, l.Title.Max.Y+popupPad)
			if len(l.Tabs) > 0 {
				at.Y = l.Tabs[0].Max.Y + popupPad
			}
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
	for i, tab := range l.Tabs {
		drawPushButton(dst, body, pushButton{Rect: tab, Label: s.Tabs[i], Literal: true, LabelRaise: labelRaise(body), Focus: s.ActiveTab == i, Hover: pointerOK && pointer.In(tab)})
	}
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
			band := rr.Rect.Intersect(l.Rect)
			drawFrame(dst, panelFrame(band, ink.TitleGround, ink.Edge))
			if title != nil {
				title.Draw(dst.SubImage(band).(*image.RGBA), row.Label, rr.Rect.Min.X+4, rr.Rect.Min.Y+popupRowExtra/2, ink.Title)
			}
			continue
		}
		label := rr.Label
		if row.Kind == popupCard {
			// The card's ground over the rows it holds, lighter while it is
			// held; a struck card is a strip in the hint ink.
			ground, edge := ink.TitleGround, ink.Edge
			if row.Struck {
				edge = popupHintInk
			} else if s.Drag != nil && s.Drag.Row == rr.Index {
				ground = ink.Ground
			}
			drawFrame(dst, panelFrame(rr.Card.Intersect(l.Rect), ground, edge))
			if row.Struck {
				bodyText(label, editorClip(body, row.Label, label.Dx()-2), popupHintInk)
				for k, b := range rr.Buttons {
					drawPushButton(dst, body, pushButton{Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Hover: pointerOK && pointer.In(b)})
				}
				continue
			}
			// The grip: three strokes before the card's number.
			mid := (label.Min.Y + label.Max.Y) / 2
			for k := -1; k <= 1; k++ {
				stroke := image.Rect(label.Min.X+4, mid+4*k-1, label.Min.X+14, mid+4*k+1)
				draw.Draw(dst, stroke.Intersect(l.Rect), &image.Uniform{C: ink.Edge}, image.Point{}, draw.Src)
			}
			label.Min.X += 18
		}
		bodyText(label, editorClip(body, row.Label, label.Dx()-2), ink.Label)
		if row.Edit != nil {
			drawEditorInput(dst, body, row, rr.Control, ink, value)
			drawRowIcons(dst, body, l, rr, row)
			for k, b := range rr.Buttons {
				drawPushButton(dst, body, pushButton{Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Focus: row.Kind == popupPick && s.PickArmed, Disabled: k < len(row.Disabled) && row.Disabled[k], Hover: pointerOK && pointer.In(b)})
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
				// A value longer than the field ends in an ellipsis at the
				// room rowFull measures, and the hover line shows it whole.
				room := rr.Control.Dx() - 2 - editTextX
				if row.Value == "" && row.Hint != "" {
					body.Draw(dst.SubImage(rr.Control.Inset(2)).(*image.RGBA), editorClip(body, row.Hint, room), at.X, at.Y, popupHintInk)
					return
				}
				body.Draw(dst.SubImage(rr.Control.Inset(2)).(*image.RGBA), editorClip(body, row.Value, room), at.X, at.Y, value)
			})
		case popupSwitch:
			// A long value ends in an ellipsis inside its button.
			drawPushButton(dst, body, pushButton{Rect: rr.Control, Label: editorClip(body, row.Value, rr.Control.Dx()-12), Literal: true, LabelRaise: labelRaise(body), Focus: row.Focus, Hover: pointerOK && pointer.In(rr.Control)})
		case popupChoice, popupCard:
			drawDropdown(dst, body, frames, dropdown{Rect: rr.Control, Text: row.Value, Focus: row.Focus, Fill: ink.Ground, Ink: value}, nil, pointer, pointerOK)
		default:
			bodyText(rr.Control, editorClip(body, row.Value, rr.Control.Dx()-2), value)
		}
		drawRowIcons(dst, body, l, rr, row)
		for k, b := range rr.Buttons {
			drawPushButton(dst, body, pushButton{Rect: b, Label: rowButtons(s, row)[k], Literal: true, LabelRaise: labelRaise(body), Focus: row.Kind == popupPick && s.PickArmed, Disabled: k < len(row.Disabled) && row.Disabled[k], Hover: pointerOK && pointer.In(b)})
		}
	}
	if !l.Marker.Empty() {
		draw.Draw(dst, l.Marker.Intersect(l.Rect), &image.Uniform{C: ink.Title}, image.Point{}, draw.Src)
	}
	if g := s.Grid; g != nil {
		drawPopupGrid(dst, body, frames, g, l, ink, pointer, pointerOK)
	}
	if l.HasList && s.ListText != nil {
		box := l.List
		box.HeadingFace = title
		drawListBox(dst, body, frames, box, s.List, s.ListText, pointer, pointerOK)
	}
	for i, r := range l.Notes {
		bodyText(r, s.Notes[i], s.NoteInk)
	}
	for i, r := range l.Buttons {
		disabled := i < len(s.Disabled) && s.Disabled[i]
		drawPushButton(dst, body, pushButton{Rect: r, Label: s.Buttons[i], Literal: true, LabelRaise: labelRaise(body), Disabled: disabled, Focus: s.ButtonFocus != nil && *s.ButtonFocus == i, Hover: pointerOK && pointer.In(r)})
	}
	if tip, ok := popupTipAt(dst.Bounds(), title, body, s, l, pointer, pointerOK); ok {
		drawFrame(dst, panelFrame(tip.Rect, ink.TitleGround, ink.Edge))
		clip := dst.SubImage(tip.Rect.Inset(2)).(*image.RGBA)
		for i, line := range tip.Lines {
			body.Draw(clip, line, tip.Rect.Min.X+4, tip.Rect.Min.Y+4+i*fontHeight(body, 15), ink.Value)
		}
	}
}

// popupTip is the hover line: the whole text of a row the painter cuts,
// wrapped to the row's width, in a box under the row.
type popupTip struct {
	Rect  image.Rectangle
	Lines []string
}

// rowFull is shown row rr's whole text, label then value, and whether the
// painter cuts any part of it: the label to its column, a value to its
// control, by the same widths the painter clips to.
func rowFull(title, body *text.Font, row popupRow, rr popupRowRect) (string, bool) {
	if body == nil || row.Edit != nil || row.Kind == popupMultiline {
		return "", false
	}
	full := strings.TrimSpace(row.Label + " " + row.Value)
	if row.Kind == popupHeading {
		f := title
		if f == nil {
			f = body
		}
		return full, f.Advance(row.Label) > rr.Rect.Dx()-4
	}
	label := rr.Label
	if row.Kind == popupCard && !row.Struck {
		label.Min.X += 18
	}
	cut := row.Label != "" && body.Advance(row.Label) > label.Dx()-2
	room := -1
	switch c := rr.Control; {
	case c.Empty() || row.Value == "":
	case row.Kind == popupSwitch:
		room = c.Dx() - 12
	case row.Kind == popupChoice || row.Kind == popupCard:
		room = c.Dx() - min(c.Dx()/2, c.Dy()) - 2 - editTextX
	case row.Kind == popupEdit || row.Kind == popupPick:
		room = c.Dx() - 2 - editTextX
	default:
		room = c.Dx() - 2
	}
	if room >= 0 && body.Advance(row.Value) > room {
		cut = true
	}
	return full, cut
}

// popupTipAt is the hover line of the shown row under p, false when the
// pointer is over no row or the row shows its text whole. The box sits under
// the row, or over it where bounds end first. Over the grid it is the cell's
// Tip or the whole upright column head, in a box as wide as its text.
func popupTipAt(bounds image.Rectangle, title, body *text.Font, s popupSpec, l popupLayout, p image.Point, pointerOK bool) (popupTip, bool) {
	if !pointerOK || body == nil {
		return popupTip{}, false
	}
	for _, rr := range l.Rows {
		if !p.In(rr.Rect) {
			continue
		}
		full, cut := rowFull(title, body, s.Rows[rr.Index], rr)
		if !cut {
			return popupTip{}, false
		}
		lines := editorWrap(body, full, rr.Rect.Dx()-8)
		h := len(lines)*fontHeight(body, 15) + 8
		box := image.Rect(rr.Rect.Min.X, rr.Rect.Max.Y, rr.Rect.Max.X, rr.Rect.Max.Y+h)
		if box.Max.Y > bounds.Max.Y {
			box = box.Sub(image.Pt(0, h+rr.Rect.Dy()))
		}
		return popupTip{Rect: box, Lines: lines}, true
	}
	g := s.Grid
	if g == nil {
		return popupTip{}, false
	}
	at, full := image.Rectangle{}, ""
	for c, r := range l.GridColHeads {
		if g.HeadH > 0 && p.In(r) {
			at, full = r, g.ColHeads[c]
		}
	}
	for i, r := range l.GridCells {
		if p.In(r) {
			at, full = r, g.Cells[l.GridFirst+i].Tip
		}
	}
	if full == "" {
		return popupTip{}, false
	}
	lines, w := editorWrap(body, full, bounds.Dx()-8), 0
	for _, line := range lines {
		w = max(w, body.Advance(line))
	}
	h := len(lines)*fontHeight(body, 15) + 8
	box := image.Rect(at.Min.X, at.Max.Y, at.Min.X+w+8, at.Max.Y+h)
	if box.Max.X > bounds.Max.X {
		box = box.Sub(image.Pt(box.Max.X-bounds.Max.X, 0))
	}
	if box.Max.Y > bounds.Max.Y {
		box = box.Sub(image.Pt(0, h+at.Dy()))
	}
	return popupTip{Rect: box, Lines: lines}, true
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
	head := func(r image.Rectangle, s string, centre bool) {
		if body == nil || s == "" {
			return
		}
		s = editorClip(body, s, r.Dx()-4)
		x := r.Min.X + 2
		if centre {
			x = r.Min.X + (r.Dx()-body.Advance(s))/2
		}
		body.Draw(dst.SubImage(r).(*image.RGBA), s, x, r.Min.Y+(r.Dy()-fontHeight(body, 15))/2, ink.Label)
	}
	for k, r := range l.GridRowHeads {
		head(r, g.RowHeads[l.GridTop+k], false)
	}
	for c, r := range l.GridColHeads {
		if g.HeadH > 0 && body != nil {
			drawTextUp(dst, body, editorClip(body, g.ColHeads[c], r.Dy()-4), r, ink.Label)
			continue
		}
		head(r, g.ColHeads[c], true)
	}
	for i, c := range l.GridCells {
		cell := g.Cells[l.GridFirst+i]
		if cell.Inert {
			drawFrame(dst, panelFrame(c, ink.TitleGround, ink.Edge))
			continue
		}
		label, raise := "", 0
		if cell.Icon == nil && cell.Label != "" {
			label, raise = cell.Label, labelRaise(body)
		}
		drawPushButton(dst, body, pushButton{Rect: c, Label: label, Literal: true, LabelRaise: raise, Pressed: cell.Pressed, Inside: cell.Pressed, Hover: pointerOK && pointer.In(c)})
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

// drawTextUp draws s with the body font turned a quarter turn
// counter-clockwise inside r: it reads from bottom to top, starts 2 pixels
// above r's bottom and is centred across r. It is the one way the editor
// draws upright text. The line is drawn into its own picture in its own
// capture window, so no smoothed-text overlay redraws its glyphs unturned.
func drawTextUp(dst *image.RGBA, body *text.Font, s string, r image.Rectangle, ink color.RGBA) {
	if body == nil || s == "" {
		return
	}
	w, h := body.Advance(s), fontHeight(body, 15)
	if w <= 0 {
		return
	}
	line := image.NewRGBA(image.Rect(0, 0, w, h))
	text.Record(func() { body.Draw(line, s, 0, 0, ink) })
	x0, y0, clip := r.Min.X+(r.Dx()-h)/2, r.Max.Y-2-w, r.Intersect(dst.Bounds())
	for y := range h {
		for x := range w {
			c, at := line.RGBAAt(x, y), image.Pt(x0+y, y0+w-1-x)
			if c.A != 0 && at.In(clip) {
				dst.SetRGBA(at.X, at.Y, c)
			}
		}
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
// four diagonal strokes, two pixels thick, inside the popupCorner square.
func drawPopupGrip(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	for k := 1; k <= 4; k++ {
		for i := 0; i < 4*k-2; i++ {
			for d := 0; d < 2; d++ {
				p := image.Pt(r.Max.X-3-4*k+i+d+3, r.Max.Y-3-i)
				if p.In(dst.Bounds()) && p.X < r.Max.X-1 {
					dst.SetRGBA(p.X, p.Y, c)
				}
			}
		}
	}
}
