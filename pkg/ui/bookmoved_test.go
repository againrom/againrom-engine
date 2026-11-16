package ui

import (
	"image"
	"image/color"
	"testing"
)

// A moved spell's cell shows its own icon over the installed picture; an
// unmoved cell keeps the installed picture, and a moved spell with no icon is
// covered.
func TestAMovedBookCellDrawsItsOwnIcon(t *testing.T) {
	green := color.RGBA{0, 200, 0, 255}
	hidden := color.RGBA{9, 9, 9, 255}
	red := color.RGBA{200, 0, 0, 255}
	art := &BottomHUDArt{Book: solidPic(480, 85, green), UnknownSpell: solidPic(36, 36, hidden)}
	bar, cols, ok := bookBarRect(image.Pt(640, 480))
	if !ok {
		t.Fatal("no bar")
	}
	entries := []SpellEntry{{ID: 3, Moved: true, Icon: solidPic(36, 36, red)}, {}, {ID: 1, Moved: true}, {ID: 4}}
	cells := bookCellRects(bar, cols)
	img := composeOriginalSpellBar(art, entries, 0, bar, 0)
	at := func(slot int) color.RGBA {
		c := cells[slot].Sub(bar.Min).Min
		return img.RGBAAt(c.X+1, c.Y+1)
	}
	for slot, want := range []color.RGBA{red, hidden, hidden, green} {
		if got := at(slot); got != want {
			t.Errorf("cell %d drew %v, want %v", slot, got, want)
		}
	}
}
