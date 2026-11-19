package ui

import (
	"image"
	"strings"

	"againrom/pkg/render/text"
)

// naturalWidth is the width spec s asks for at its natural size in
// the fonts title and body. It first gives s its label column: as wide as
// the widest label beside a control, its (?) mark and a card's grip
// included, and at least least. The width is the widest of the title with
// its close and help marks, the button row, and each row at its natural
// size: the label column, the value and the row's own buttons; a line
// across the row with its mark and buttons; a heading; the pictures a row
// shows after its value. Text values, multiline values and notes count up to
// the reading measure. A plain list asks for its widest row and scrollbar; a
// list of columns for its columns' need; a grid for its head column, its
// columns and scrollbar. A grid with LabelHeads takes its head column from
// the label column. It is the one measure of a window's width.
func naturalWidth(s *popupSpec, title, body *text.Font, least int) int {
	measure := func(t string) int { return 8 * len([]rune(t)) }
	if body != nil {
		measure = body.Advance
	}
	pitch := fontHeight(body, 15) + popupRowExtra
	s.LabelWidth = least
	for _, r := range s.Rows {
		switch {
		case r.placed():
		case r.Kind == popupCard && r.Value != "" && !r.Struck:
			s.LabelWidth = max(s.LabelWidth, cardGrip+measure(r.Label)+8)
		case rowHasControl(r):
			s.LabelWidth = max(s.LabelWidth, measure(r.Label)+8+helpMarkW(r, pitch))
		}
	}
	if g := s.Grid; g != nil && g.LabelHeads {
		for _, h := range g.RowHeads {
			s.LabelWidth = max(s.LabelWidth, measure(h)+8)
		}
		g.HeadW = s.LabelWidth
	}
	w := specWindow(*s)
	most := readingMeasure(measure)
	width := max(160, buttonRowWidth(w, body))
	if title != nil {
		titleH := popupTitleHeight(title)
		marks := titleH
		if s.TitleHelp != "" {
			marks += titleH + 4
		}
		width = max(width, 2*popupPad+title.Advance(s.Title)+marks)
	}
	for _, r := range s.Rows {
		if r.placed() {
			continue
		}
		buttons := 0
		for _, bw := range rowButtonWidths(w, r, body, pitch) {
			buttons += bw + 4
		}
		need := 0
		switch {
		case r.Kind == popupHeading:
			heading := measure
			if title != nil {
				heading = title.Advance
			}
			need = 4 + foldIndent(r) + heading(r.Label)
		case r.Kind == popupMultiline:
			longest := 0
			for _, line := range strings.Split(r.Value, "\n") {
				longest = max(longest, min(measure(line), most))
			}
			need = s.LabelWidth + 4 + max(popupMinField, longest+8)
		case r.Kind == popupCard && (r.Value == "" || r.Struck), r.Kind != popupCard && !rowHasControl(r):
			grip := 0
			if r.Kind == popupCard && !r.Struck {
				grip = cardGrip
			}
			need = grip + measure(r.Label) + 8 + helpMarkW(r, pitch) + buttons
		default:
			field := measure(r.Value) + 2
			switch r.Kind {
			case popupText, popupLink:
				field = min(measure(r.Value), most) + 2*(editTextX+2)
			case popupChoice, popupCard:
				field += editTextX + pitch
			case popupEdit, popupPick:
				field += editTextX
			}
			need = s.LabelWidth + 4 + max(popupMinField, field) + buttons + (pitch+2)*len(r.Icons)
		}
		width = max(width, 2*popupPad+need)
	}
	for _, n := range s.Notes {
		width = max(width, 2*(popupPad+popupTextInset)+min(measure(n), most))
	}
	if g := s.Grid; g != nil {
		cols := g.Cols
		if cols <= 0 {
			cols = min(len(g.Cells), gridLeastCols)
		}
		need := 2*popupPad + g.HeadW + cols*g.side() + widgetSpriteSize
		if g.Cols <= 0 {
			need += 2
		}
		width = max(width, need)
	}
	if len(s.ListColumns) > 0 {
		list := uiList{Picker: s.List, Columns: s.ListColumns, Aside: s.ListAside}
		need := widgetSpriteSize + list.Aside + 2*listTextX
		for _, n := range listColumnNeeds(list, listHeadFont(title, body), body) {
			need += n
		}
		width = max(width, 2*popupPad+need)
	} else if s.List != nil {
		pitch := newListBox(image.Rectangle{}, 1, body).Pitch
		for _, row := range s.List.Rows() {
			advance := measure
			if row.Heading && title != nil {
				advance = title.Advance
			}
			lead := listRowLead(row, pitch, listTextX)
			width = max(width, 2*popupPad+widgetSpriteSize+s.ListAside+2*listTextX+lead+advance(row.Text))
		}
	}
	return width
}

// cardGrip is how far right of its column a card's number stands: the grip's
// strokes come first.
const cardGrip = 18

// readingMeasure is questionMeasure average characters in measure's font.
func readingMeasure(measure func(string) int) int {
	return questionMeasure * max(1, measure(averageLetters)/len(averageLetters))
}
