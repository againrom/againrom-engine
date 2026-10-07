package ui

import (
	"fmt"
	"image"
	"image/color"
)

// Map-picker layout, in frame pixels of the 640x480 virtual frame.
//
// The list is drawn with the engine's debug font, whose glyph cell is a fixed
// 6x16 — this screen's text is deliberately placeholder text, and rendering it
// with the game's own fonts is a later story. The line pitch is that cell height,
// so these constants and the font agree by construction rather than by taste.
//
// A stock install lists 38 maps and only 25 rows fit, which is why the model owns
// a scroll window. The draw path and the hit test share these constants AND share
// Visible(), so a row that is drawn is a row that can be clicked.
const (
	pickerLeft     = 8   // left margin of every line
	pickerHeaderY  = 8   // the header line
	pickerTop      = 40  // top of the first list row
	pickerLine     = 16  // line pitch == the debug font's cell height
	pickerVisible  = 25  // rows on screen; the list occupies y in [40, 440)
	pickerMessageY = 452 // where a failed load reports itself
	pickerCols     = 104 // 8 + 104*6 = 632 <= 640, so a row cannot run off the frame

	// pickerWheelRows is how far one wheel notch moves the selection. One row
	// per notch feels broken on a 38-row list in a 25-row window; a whole page
	// loses the reader's place. Three is the usual compromise and is stated here
	// rather than at the dispatch, so the number and the window it moves through
	// sit together.
	pickerWheelRows = 3
)

// PickerRow is one line of the map list: the text to show, and whether it can be
// chosen.
//
// A row that cannot be chosen is still listed. A map whose metadata will not
// decode has to stay visible — dropping it would hide part of an install rather
// than report it — and so does a map that turned out not to load when it was
// picked.
type PickerRow struct {
	Text      string
	Choosable bool

	// Width, Height and Description are this row's own decoded map size and
	// prose, carried from game.MapEntry unchanged, zero/empty for every row
	// a non-map list builds and for a map row whose metadata did not decode.
	// The hover hint below reads them; drawList's own text does not.
	Width, Height int
	Description   string

	// Word70 and Word74 fill the last two columns.
	Word70, Word74 uint32
}

// Column edges in frame pixels from pickerLeft (TEXT-083); the hover
// thresholds are the same edges.
const (
	pickerColumnSize = 300
	pickerColumnWord = 390
	pickerColumnLast = 420
)

// HasColumns reports whether the row shows the metadata columns.
func (r PickerRow) HasColumns() bool { return r.Width > 0 && r.Height > 0 }

// mapListSizeBorder is what the map list takes off each header dimension
// before printing it (MENU-067).
const mapListSizeBorder = 16

// ColumnTexts are the size and the two record words as drawn. The size is the
// map header's width and height each minus 16, as the original's row text
// prints them.
func (r PickerRow) ColumnTexts() [3]string {
	return [3]string{fmt.Sprintf("%dx%d", r.Width-mapListSizeBorder, r.Height-mapListSizeBorder), fmt.Sprintf("%d", r.Word70), fmt.Sprintf("%d", r.Word74)}
}

// InList reports whether p lies in the list's window, filled or not.
func (pk *Picker) InList(p image.Point) bool {
	return p.X >= pickerLeft && p.Y >= pickerTop && p.Y < pickerTop+pk.visibleRows()*pickerLine
}

// MapListColumnAt names the column x selects: 0 description, 1 size, 2 and 3
// the record words. Only x decides.
func MapListColumnAt(x int) int {
	switch rel := x - pickerLeft; {
	case rel < pickerColumnSize:
		return 0
	case rel < pickerColumnWord:
		return 1
	case rel < pickerColumnLast:
		return 2
	}
	return 3
}

// Picker is the map list's selection model: which row is selected, which rows are
// on screen, and which row a frame position falls on.
//
// It is pure — no engine, no assets, no clock — so the whole of its behaviour is
// decidable in a test.
type Picker struct {
	rows []PickerRow
	sel  int // index of the selected row
	top  int // index of the first visible row

	// window is how many rows this list may occupy, in rows.
	//
	// A zero window is pickerVisible, so a Picker built by a composite literal
	// behaves exactly as one built before this field existed.
	window int
}

// NewPicker builds a picker over rows. The first row is selected; an empty list
// is legal and selects nothing.
func NewPicker(rows []PickerRow) *Picker {
	return &Picker{rows: rows}
}

// SetWindow limits the list to n rows on screen, and reports the picker so a
// caller can build and size one in a single statement. An n outside 1..
// pickerVisible restores the full window rather than being refused: this is a
// layout hint, and a wrong one should not leave a screen with no rows on it.
func (p *Picker) SetWindow(n int) *Picker {
	if p == nil {
		return nil
	}
	if n < 1 || n > pickerVisible {
		n = 0
	}
	p.window = n
	p.scrollToSelection()
	return p
}

// visibleRows is this list's window in rows.
func (p *Picker) visibleRows() int {
	if p.window <= 0 || p.window > pickerVisible {
		return pickerVisible
	}
	return p.window
}

// Len reports how many rows the list holds.
func (p *Picker) Len() int { return len(p.rows) }

// Rows returns the list. The slice is the picker's own; callers read it.
func (p *Picker) Rows() []PickerRow { return p.rows }

// Selection reports the selected row index, or -1 when the list is empty.
func (p *Picker) Selection() int {
	if len(p.rows) == 0 {
		return -1
	}
	return p.sel
}

// Move shifts the selection by d rows, clamping at both ends, and scrolls the
// window by the least amount that keeps the selection visible.
//
// Clamping rather than wrapping is deliberate: with 38 rows and 25 on screen, a
// wrap would jump the view a whole page on a keypress that looks like a nudge.
func (p *Picker) Move(d int) {
	if len(p.rows) == 0 {
		return
	}
	p.sel = clampIndex(p.sel+d, len(p.rows))
	p.scrollToSelection()
}

// Select moves the selection to an absolute row, scrolling it into view, and
// reports whether the index was in range.
//
// This is what a click needs: the hit test names a row, and the selection has to
// go there. Expressing it as a relative Move of the difference would put
// arithmetic on top of a clamped operation, which is where an off-by-one hides.
func (p *Picker) Select(i int) bool {
	if i < 0 || i >= len(p.rows) {
		return false
	}
	p.sel = i
	p.scrollToSelection()
	return true
}

// scrollToSelection moves the window by the least amount that keeps the selected
// row on screen.
func (p *Picker) scrollToSelection() {
	if p.sel < p.top {
		p.top = p.sel
	}
	if n := p.visibleRows(); p.sel >= p.top+n {
		p.top = p.sel - n + 1
	}
	if max := len(p.rows) - p.visibleRows(); p.top > max {
		p.top = max
	}
	if p.top < 0 {
		p.top = 0
	}
}

// Visible reports the window the draw path must render: the first row index and
// how many rows follow it.
//
// The hit test is derived from the same value, so the set of rows a user can see
// and the set they can click are the same set by construction — not by two
// pieces of code agreeing about a constant.
func (p *Picker) Visible() (top, n int) {
	n = p.visibleRows()
	if rest := len(p.rows) - p.top; rest < n {
		n = rest
	}
	if n < 0 {
		n = 0
	}
	return p.top, n
}

// RowAt reports which row the frame position p falls on.
//
// Only the y coordinate decides. Any x inside the frame selects the row, which
// is both friendlier than a text-width hit box and immune to the debug font's
// per-glyph horizontal offset. A position above the list, below the last visible
// row, or past the end of a short list falls on no row.
func (pk *Picker) RowAt(p image.Point) (int, bool) {
	if p.Y < pickerTop {
		return 0, false
	}
	k := (p.Y - pickerTop) / pickerLine
	top, n := pk.Visible()
	if k < 0 || k >= n {
		return 0, false
	}
	return top + k, true
}

// Choose reports the selected row when it can be chosen.
//
// A row that is listed but unusable — unreadable metadata, or a map that failed
// to load when it was picked — yields nothing rather than an error, so the caller
// has one thing to check instead of two.
func (p *Picker) Choose() (int, bool) {
	if len(p.rows) == 0 {
		return 0, false
	}
	if !p.rows[p.sel].Choosable {
		return 0, false
	}
	return p.sel, true
}

// SetUnusable marks a row as no longer choosable, leaving it listed.
//
// This is what a failed load calls. The row stays on screen, visibly marked, so
// the user sees which map failed rather than watching it disappear.
func (p *Picker) SetUnusable(i int) {
	if i < 0 || i >= len(p.rows) {
		return
	}
	p.rows[i].Choosable = false
}

// RowText renders row i as the draw path shows it: a selection marker, then the
// row's text, clipped to the frame width.
//
// The marker is two columns wide whether or not the row is selected, so the text
// of every row starts at the same x and the list does not jitter as the selection
// moves.
func (p *Picker) RowText(i int) string {
	if i < 0 || i >= len(p.rows) {
		return ""
	}
	marker := "  "
	if i == p.sel {
		marker = "> "
	}
	return marker + clipRunes(p.rows[i].Text, pickerCols-len(marker))
}

// clipRunes truncates s to at most n runes, counting runes rather than bytes so a
// multi-byte character is never cut in half.
//
// A CUT IS MARKED WITH AN ELLIPSIS and was silent before (1032 return 1).
// These lines are drawn with ebitenutil's ASCII debug font, so the mark is
// three full stops rather than the one ellipsis rune. The three fit INSIDE
// n: a marked line is exactly as wide as an unmarked one, so nothing this
// bounds can run off the frame. Below four columns there is no room for the
// mark and the text is cut silently, which is the old behaviour for a width
// nothing readable fits in anyway.
func clipRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= len(clipMark) {
		return string(r[:n])
	}
	return string(r[:n-len(clipMark)]) + clipMark
}

// clipMark is what clipRunes leaves where it cut.
const clipMark = "..."

func clampIndex(i, n int) int {
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

// The map list's header, in two fixed parts with the list's extent between them.
// Like every string on this screen they are placeholder debug text: rendering the
// game's own fonts is a later story and no font asset is read here.
const (
	// SELECT A MISSION OR MAP names both kinds of row this screen lists now:
	// the whole distinction between a mission row and a map row is expressible
	// in the row's own text, so the title's only job is to say that both are on
	// offer, not to gain a second field or a second key.
	pickerTitle = "SELECT A MISSION OR MAP"
	pickerHelp  = "up/down + enter, or click a row   esc: back"
)

// HeaderText is the map list's one-line header: what the screen is, how much of
// the list is on it, and how to drive it.
//
// The range comes from Visible() -- the same value the draw path iterates and
// RowAt derives from -- so the extent a user reads and the rows they can reach
// cannot drift apart. It is 1-based because a human reads it; nothing indexes
// with it.
//
// Without it a 38-row list drawn 25 rows at a time looks exactly like a 25-row
// list: nothing on screen says otherwise, and a user who does not think to
// scroll concludes the install is short. That is what was reported, so the count
// is not decoration.
func (p *Picker) HeaderText() string {
	extent := "no maps"
	if top, n := p.Visible(); len(p.rows) > 0 {
		extent = fmt.Sprintf("maps %d-%d of %d", top+1, top+n, len(p.rows))
	}
	return clipRunes(pickerTitle+"   "+extent+"   "+pickerHelp, pickerCols)
}

// pickerBackground is the map list's fill. It is the project's own choice, not a
// reproduction of anything the original draws.
var pickerBackground = color.RGBA{R: 0x0d, G: 0x0d, B: 0x14, A: 0xff}
