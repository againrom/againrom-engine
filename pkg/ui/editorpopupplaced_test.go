package ui

import (
	"image"
	"strings"
	"testing"
)

// placedCardSpec is a fitted popup at 1920x1080 with two rows over a
// character card drawn twice its size under them, and two rows placed on
// the card: a field and a line of text with its own tip.
func placedCardSpec() popupSpec {
	subject := cardColumnsSubjects()["fighter"]
	return popupSpec{Title: "Unit #0", At: image.Pt(20, 40), Width: 600, Bounds: image.Rect(0, 0, 1920, 1080), Fit: true,
		Card: &subject, CardScale: 2, CardBelow: true,
		Rows: []popupRow{
			{Kind: popupEdit, Label: "Group", Value: "2"},
			{Kind: popupChoice, Label: "Player", Value: "Self"},
			{Kind: popupEdit, Value: "43", Place: image.Rect(40, 40, 90, 60)},
			{Kind: popupText, Label: "DMG", Value: "6-10", Place: image.Rect(100, 40, 140, 60), Tip: "DMG 6-10: fixed"},
		},
		Buttons: []string{"Apply"}}
}

// A card scaled twice stands under the rows across their width; a row
// placed on it stands at its Place from the card's corner, takes no line
// among the rows and shows its tip whole; the card's pixels are the
// character panel's, each drawn as a 2x2 block.
func TestPopupPlacesRowsOnAScaledCardUnderTheRows(t *testing.T) {
	f := panelFont()
	s := placedCardSpec()
	l := layoutPopup(s, f, f)
	if got := l.Card.Size(); got != image.Pt(2*compactPanelW, 2*compactPanelH) {
		t.Fatalf("card %v, want twice the panel", got)
	}
	if popupRowCount(s) != 2 {
		t.Fatalf("%d row lines, want the two rows that are not placed", popupRowCount(s))
	}
	bottom := 0
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if !row.placed() {
			bottom = max(bottom, rr.Rect.Max.Y)
			continue
		}
		want := row.Place.Add(l.Card.Min)
		if rr.Rect != want {
			t.Errorf("row %d at %v, want %v", rr.Index, rr.Rect, want)
		}
		if field := row.Kind == popupEdit; field != (rr.Control == want) || !field && !rr.Control.Empty() {
			t.Errorf("row %d control %v", rr.Index, rr.Control)
		}
	}
	if len(l.Rows) != 4 || l.Card.Min.Y != bottom+popupPad || l.CardRest.Max.X >= l.Card.Min.X || l.CardRest.Dy() != l.Card.Dy() {
		t.Fatalf("rows %d end at %d; card %v, its room %v", len(l.Rows), bottom, l.Card, l.CardRest)
	}
	if len(l.Buttons) != 1 || l.Buttons[0].Min.Y < l.Card.Max.Y {
		t.Fatalf("the button %v stands over the card %v", l.Buttons[0], l.Card)
	}
	text := l.Rows[3]
	tip, ok := popupTipAt(s.Bounds, f, f, s, l, text.Rect.Min.Add(image.Pt(2, 2)), true)
	if !ok || strings.Join(tip.Lines, " ") != "DMG 6-10: fixed" || tip.Rect.Dx() < popupTipW {
		t.Fatalf("tip %+v %v", tip, ok)
	}

	dst := image.NewRGBA(s.Bounds)
	drawPopup(dst, f, f, nil, s, l, editorPopupInk, image.Point{}, false)
	pic := RenderCharacterPanel(CompactPanelLayout(nil), f, *s.Card)
	field := l.Rows[2].Control
	for y := 0; y < compactPanelH; y++ {
		for x := 0; x < compactPanelW; x++ {
			c := pic.RGBAAt(x, y)
			at := l.Card.Min.Add(image.Pt(2*x, 2*y))
			if c.A != 255 || image.Rect(at.X, at.Y, at.X+2, at.Y+2).Overlaps(field) {
				continue
			}
			for _, d := range []image.Point{{0, 0}, {1, 0}, {0, 1}, {1, 1}} {
				if got := dst.RGBAAt(at.X+d.X, at.Y+d.Y); got != c {
					t.Fatalf("card pixel %d,%d drawn %v at %v, want %v", x, y, got, at.Add(d), c)
				}
			}
		}
	}

	// At its own size the card keeps today's place beside the rows.
	own := placedCardSpec()
	own.CardScale, own.CardBelow, own.Rows = 0, false, own.Rows[:2]
	one := own
	one.CardScale = 1
	if a, b := layoutPopup(own, f, f), layoutPopup(one, f, f); a.Card != b.Card || a.Rect != b.Rect || a.Card.Size() != image.Pt(compactPanelW, compactPanelH) {
		t.Fatalf("scale 0 card %v rect %v, scale 1 card %v rect %v", a.Card, a.Rect, b.Card, b.Rect)
	}
}
