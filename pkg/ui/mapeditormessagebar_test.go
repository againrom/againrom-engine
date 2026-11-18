package ui

import "testing"

// A text taller than its field shows the scroll bar beside it, and a click on
// the bar's upper end scrolls the text a line while the caret stands at its
// end; typing returns the field to the caret.
func TestMessageWindowTextBarScrollsTheText(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e := messageWindowEditor(t, lang, longMessageCases(lang)["one of 60 lines"])
			s, _, _ := e.objectSpec()
			at := e.fieldRow(0)
			l := layoutPopup(s, e.font, e.bodyFont())
			rr := rowRectOf(t, l, at)
			if rr.TextBar.Empty() || !rr.TextBar.In(l.Rect) || rr.TextBar.Min.X < rr.Control.Max.X {
				t.Fatalf("the text field %v has no bar beside it in the window %v: %v", rr.Control, l.Rect, rr.TextBar)
			}
			// The window opens with the caret at the end of the text, the
			// last line shown.
			want := 60 - (rr.Control.Dy()-8)/fontHeight(e.bodyFont(), 15) - 1
			modesClick(e, center(vScrollBar{Rect: rr.TextBar}.top()))
			s, _, _ = e.messageSpec()
			if s.Rows[at].First != want || !s.Rows[at].Scrolled {
				t.Errorf("a click on the bar's upper end leaves the first line at %d (scrolled %v), want %d", s.Rows[at].First, s.Rows[at].Scrolled, want)
			}
			e.TextInput("x")
			s, _, _ = e.messageSpec()
			if s.Rows[at].Scrolled {
				t.Error("typing leaves the field scrolled away from the caret")
			}
		})
	}
}
