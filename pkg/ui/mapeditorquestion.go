package ui

import (
	"fmt"
	"image"

	"againrom/pkg/render/text"
)

// questionMeasure is the most a question's note line asks for, in average
// characters of the body font: past it the note wraps instead of widening
// the window.
const questionMeasure = 60

// averageLetters are the letters an average character is measured over.
const averageLetters = "abcdefghijklmnopqrstuvwxyz"

// popupTextInset is how far right of its box a popup's body line starts; a
// note wraps that far inside its box on both sides, so no line is clipped.
const popupTextInset = 2

// question is a question window as data, which questionSpec lays out. The
// title names the window. Help says what the choice does; it stands behind
// the title's (?) with the keys line after it, and a question without help
// has no mark. The body is rows, a list (listRows of its rows shown) or a
// grid whose picture rows are high pixels high. Notes are state lines in one
// ink: label ink when state is set, else the warning ink of lines that warn
// or refuse. Buttons run the action first and Cancel last.
type question struct {
	title, help string
	rows        []popupRow
	list        *Picker
	listRows    int
	grid        *popupGrid
	high        int
	notes       []string
	state       bool
	buttons     []string
	disabled    []bool
}

// questionSpec is question q in the middle of the editor window, at its
// natural width or the width the user dragged it to.
func (e *MapEditor) questionSpec(q question) popupSpec {
	body := e.bodyFont()
	s := q.spec(e.word("editor.question.keys"), e.font, body, nil, e.window, func(w int) int { return e.popupWide(popPrompt, w) })
	s.Bounds = e.popupBounds()
	if s.List != nil {
		list := s.List
		s.ListText = func(row, width int) string {
			rows := list.Rows()
			if row < 0 || row >= len(rows) {
				return ""
			}
			return editorClip(body, rows[row].Text, width)
		}
	}
	s.At = e.centreCorner(layoutPopup(s, e.font, body).Rect.Size(), openPlace{})
	return s
}

// spec is q as a popup in an editor window of size window: keys, filled
// with the first button, ends the help; the width is its widest part, notes
// counted up to the reading measure, as wide gives it and at most the
// window less 16. encode is the font's bytes of a string; nil keeps it.
func (q question) spec(keys string, title, body *text.Font, encode func(string) string, window image.Point, wide func(int) int) popupSpec {
	if encode == nil {
		encode = func(s string) string { return s }
	}
	measure := func(t string) int {
		if body == nil {
			return 8 * len([]rune(t))
		}
		return body.Advance(encode(t))
	}
	s := popupSpec{Title: q.title, Rows: q.rows, Grid: q.grid, Buttons: q.buttons, Disabled: q.disabled, NoteInk: editorWarning, LabelWidth: 1}
	if q.state {
		s.NoteInk = editorPaper
	}
	if q.help != "" {
		s.TitleHelp = q.help
		if len(q.buttons) > 0 {
			s.TitleHelp += " " + fmt.Sprintf(keys, q.buttons[0])
		}
	}
	if q.list != nil {
		s.List, s.ListRows = q.list, q.listRows
	}
	sizeGrid(s.Grid, q.high, body, encode, window)
	enc := encodedSpec(s, encode)
	w := 160
	if title != nil {
		titleH := layoutPopup(enc, title, body).Title.Dy()
		marks := titleH
		if s.TitleHelp != "" {
			marks += titleH + 4
		}
		w = max(w, 2*popupPad+title.Advance(enc.Title)+marks)
	}
	w = max(w, buttonRowWidth(specWindow(enc), body))
	pitch := fontHeight(body, 15) + popupRowExtra
	for _, r := range s.Rows {
		if r.Label != "" {
			s.LabelWidth = max(s.LabelWidth, measure(r.Label)+8+helpMarkW(r, pitch))
		}
	}
	for _, r := range s.Rows {
		w = max(w, 2*popupPad+s.LabelWidth+max(popupMinField, measure(r.Value)+2*popupPad+widgetSpriteSize))
	}
	if s.List != nil {
		for _, r := range s.List.Rows() {
			w = max(w, 4*popupPad+measure(r.Text)+widgetSpriteSize+8)
		}
	}
	if g := s.Grid; g != nil {
		w = max(w, 2*popupPad+g.HeadW+g.Cols*g.side())
	}
	most := questionMeasure * max(1, measure(averageLetters)/len(averageLetters))
	for _, n := range q.notes {
		w = max(w, 2*(popupPad+popupTextInset)+min(measure(n), most))
	}
	if wide != nil {
		w = wide(w)
	}
	s.Width = min(w, window.X-16)
	for _, n := range q.notes {
		s.Notes = append(s.Notes, editorRule.plain(lineBox{Measure: measure, Width: s.Width - 2*(popupPad+popupTextInset)}, n)...)
	}
	return s
}

// sizeGrid gives a question's grid g of columns its sizes in the body font
// in an editor window of size window: the head column as wide as its
// widest head, the columns narrowed to fit the window down to 24, a column
// head two lines high when a name needs them, and a row high (at most a
// column's width) where a cell shows a picture or the name standing for
// it, else a line. A grid without columns keeps its sizes.
func sizeGrid(g *popupGrid, high int, body *text.Font, encode func(string) string, window image.Point) {
	if g == nil || g.Cols <= 0 {
		return
	}
	line := fontHeight(body, 15) + popupRowExtra
	g.HeadW, g.HeadH = 8, line
	for _, h := range g.RowHeads {
		if body != nil {
			g.HeadW = max(g.HeadW, body.Advance(encode(h))+8)
		}
	}
	g.Side = max(24, min(g.side(), (window.X-16-2*popupPad-g.HeadW)/g.Cols))
	for _, h := range g.ColHeads {
		if len(headLines(body, encode(h), g.side()-4)) > 1 {
			g.HeadH = 2 * line
		}
	}
	if high <= 0 {
		high = g.side()
	}
	g.RowHigh = nil
	for row := range g.Rows {
		h := line + 4
		for col := range g.Cols {
			if n := row*g.Cols + col; n < len(g.Cells) && (g.Cells[n].Icon != nil || g.Cells[n].Label != "") {
				h = min(g.side(), high)
			}
		}
		g.RowHigh = append(g.RowHigh, h)
	}
}
