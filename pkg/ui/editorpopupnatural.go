package ui

import (
	"strings"

	"againrom/pkg/render/text"
)

// naturalWidth is the width spec s asks for at its natural size in
// the fonts title and body. It first gives s its label column: as wide as
// the widest label beside a control, its (?) mark and a card's grip
// included, and at least least. The width is the widest of the title with
// its close and help marks, the button row, and each row at its natural
// size: the label column, the value and the row's own buttons; a line
// across the row with its mark and buttons; a heading. A multi-line text's
// longest line and a note count up to the reading measure (questionMeasure):
// past it they wrap.
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
	w := specWindow(*s)
	most := readingMeasure(measure)
	width := max(160, buttonRowWidth(w, body))
	if title != nil {
		titleH := fontHeight(title, 15) + popupRowExtra
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
			case popupChoice, popupCard:
				field += editTextX + pitch
			case popupEdit, popupPick:
				field += editTextX
			}
			need = s.LabelWidth + 4 + max(popupMinField, field) + buttons
		}
		width = max(width, 2*popupPad+need)
	}
	for _, n := range s.Notes {
		width = max(width, 2*(popupPad+popupTextInset)+min(measure(n), most))
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
