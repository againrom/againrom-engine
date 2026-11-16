package ui

import "testing"

// The detailed view holds the card in its own block at the end of the
// identity column, under the identity's blocks, with the card's inputs on
// it, and the figure under the equipment: three columns, no column of the
// card's section, at 1920x1080 in both languages.
func TestEditorDetailedViewHoldsTheCardUnderIdentity(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e := cornerEditor(t, lang)
			e.props.plain = false
			e.Layout(1920, 1080)
			e.Select(viewsUnit, false)
			s, _, _ := e.objectSpec()
			if s.Columns != 3 || !cardHeld(specWindow(s)) {
				t.Fatalf("columns %d, card held %v", s.Columns, cardHeld(specWindow(s)))
			}
			heading, card, figure := -1, -1, -1
			for i, row := range s.Rows {
				switch {
				case row.Kind == popupHeading && row.Label == e.word("editor.person."+cardSection):
					heading = i
				case row.Hold == holdCard:
					card = i
				case row.Hold == holdFigure:
					figure = i
				}
				if !row.placed() && row.Column > 2 {
					t.Errorf("row %q stands in column %d", row.Label, row.Column)
				}
			}
			if heading < 0 || card != heading+1 || figure < 0 {
				t.Fatalf("card heading %d, card %d, figure %d", heading, card, figure)
			}
			if s.Rows[card].Column != 0 || s.Rows[figure].Column != personColumn("equipment", 3) {
				t.Errorf("card in column %d, figure in column %d", s.Rows[card].Column, s.Rows[figure].Column)
			}
			// Under the card stand only the card's own fields the card does
			// not draw: no other block.
			for i := card + 1; i < len(s.Rows); i++ {
				if !s.Rows[i].placed() && s.Rows[i].Column == 0 && s.Rows[i].Kind == popupHeading {
					t.Errorf("heading %q stands under the card", s.Rows[i].Label)
				}
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			if l.Card.Empty() || !l.Card.In(l.Rect) || !l.Rect.In(s.Bounds) {
				t.Fatalf("card %v, window %v, bounds %v", l.Card, l.Rect, s.Bounds)
			}
			inputs := 0
			for _, rr := range l.Rows {
				row := s.Rows[rr.Index]
				switch {
				case row.Hold == holdCard && rr.Rect.Min != l.Card.Min:
					t.Errorf("the card stands at %v, its row at %v", l.Card.Min, rr.Rect.Min)
				case row.Hold == holdFigure && rr.Rect.Dy() < e.unitFigureSize().Y:
					t.Errorf("the figure's row %v is lower than the figure %v", rr.Rect, e.unitFigureSize())
				case row.placed() && !rr.Control.Empty():
					inputs++
					if !rr.Control.Overlaps(l.Card) {
						t.Errorf("card input %v stands off the card %v", rr.Control, l.Card)
					}
				case row.Kind == popupHeading:
					if full, cut := rowFull(e.font, e.bodyFont(), row, rr); cut {
						t.Errorf("heading %q is cut", full)
					}
				}
			}
			if inputs == 0 {
				t.Error("the card holds no input")
			}
		})
	}
}
