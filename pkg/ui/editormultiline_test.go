package ui

import (
	"image"
	"testing"
)

func TestPopupMultilineSharesLayoutAndHitTest(t *testing.T) {
	s := popupSpec{Width: 660, Bounds: image.Rect(0, 0, 1024, 768), Rows: []popupRow{{Kind: popupMultiline, Label: "Text", Value: "One\nTwo"}, {Kind: popupChoice, Label: "Speaker"}}, Buttons: []string{"Apply", "Cancel"}}
	l := layoutPopup(s, panelFont(), panelFont())
	if l.Rows[0].Rect.Dy() != 6*l.Pitch || l.Rows[0].Control.Max.Y >= l.Rows[1].Rect.Min.Y || l.Buttons[0].Min.Y <= l.Rows[1].Rect.Max.Y {
		t.Fatal(l)
	}
	for _, r := range l.Rows {
		part, row := l.hit(center(r.Control))
		if part != popupRowControl || row != r.Index {
			t.Fatal(part, row)
		}
	}
	s.Rows[0].Kind = popupEdit
	control := layoutPopup(s, panelFont(), panelFont())
	if l.Rect.Dy()-control.Rect.Dy() != 5*l.Pitch {
		t.Fatal("multiline loss control")
	}
}
