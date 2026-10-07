package ui

import (
	"image"
	"image/color"
	"testing"
)

// cardWithNarrowTop is a 160x242 card whose writing surface is only 60 pixels
// wide over its first 30 rows, like the shipped frame's ornament at the name
// row, and full width below.
func cardWithNarrowTop() *image.RGBA {
	bg := image.NewRGBA(image.Rect(0, 0, compactPanelW, compactPanelH))
	surface, frame := color.RGBA{50, 90, 50, 255}, color.RGBA{120, 60, 20, 255}
	for y := 0; y < compactPanelH; y++ {
		for x := 0; x < compactPanelW; x++ {
			c := surface
			if y < 30 && (x < 50 || x >= 110) {
				c = frame
			}
			bg.SetRGBA(x, y, c)
		}
	}
	return bg
}

// A name wider than the card's top row wraps at a space onto a second row
// instead of losing its last letters (tester items 69 and 93).
func TestCardNameWrapsAtASpaceWhenItsRowIsTooNarrow(t *testing.T) {
	f := panelWideFont()
	l := CompactPanelLayout(cardWithNarrowTop())
	s := panelWideSubjectFixture()
	for _, c := range []struct{ name, first, second string }{
		{"Aaaaaa Bbbbbb", "Aaaaaa", "Bbbbbb"},
		{"Ann", "Ann", ""},
	} {
		s.Name, s.Char.Name = c.name, c.name
		lines := layoutLines(l, f, panelItems(l, s))
		name := lines[0]
		if name.field != PanelFieldName || name.value != c.first || name.wrap != c.second {
			t.Fatalf("name %q laid out as %q + %q, want %q + %q", c.name, name.value, name.wrap, c.first, c.second)
		}
		if c.second == "" {
			continue
		}
		left, right := characterCardRowExtent(l, l.Background, name.at.Y, f.Height())
		left2, right2 := characterCardRowExtent(l, l.Background, name.wrapAt.Y, f.Height())
		w1, _ := f.Measure(name.value)
		w2, _ := f.Measure(name.wrap)
		if name.at.X < left || name.at.X+w1 > right || name.wrapAt.X < left2 || name.wrapAt.X+w2 > right2 {
			t.Fatalf("wrapped rows leave their writing surface: first %d..%d in %d..%d, second %d..%d in %d..%d",
				name.at.X, name.at.X+w1, left, right, name.wrapAt.X, name.wrapAt.X+w2, left2, right2)
		}
		if name.wrapAt.Y <= name.at.Y {
			t.Fatalf("second row at y=%d is not below the first at y=%d", name.wrapAt.Y, name.at.Y)
		}
	}
}

// A three-digit resistance keeps a space between its caption and its number
// (tester item 100): the right cell slides left rather than touching.
func TestCardResistanceKeepsASpaceBeforeAThreeDigitValue(t *testing.T) {
	f := panelWideFont()
	bg := image.NewRGBA(image.Rect(0, 0, compactPanelW, compactPanelH))
	for y := 0; y < compactPanelH; y++ {
		for x := 0; x < compactPanelW; x++ {
			bg.SetRGBA(x, y, color.RGBA{50, 90, 50, 255})
		}
	}
	l := CompactPanelLayout(bg)
	for _, fire := range []int{5, 100, 99999} {
		s := panelWideSubjectFixture()
		s.Char.Protection[0] = fire
		s.Char.Skills[1] = 3
		var row *panelLine
		lines := layoutLines(l, f, panelItems(l, s))
		for i := range lines {
			if lines[i].right && lines[i].rightLabel == "FIRE" {
				row = &lines[i]
			}
		}
		if row == nil {
			t.Fatalf("fire %d: no FIRE row", fire)
		}
		if gap := row.rightValueX - f.Advance(row.rightLabel); gap < f.Advance(" ") {
			t.Fatalf("fire %d: %d pixels between caption and number, want at least one space (%d)", fire, gap, f.Advance(" "))
		}
		if row.value != "3" || row.label != "BLADE" {
			t.Fatalf("fire %d: the left cell became %q %q", fire, row.label, row.value)
		}
	}
}
