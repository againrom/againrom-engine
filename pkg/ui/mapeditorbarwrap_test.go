package ui

import (
	"image"
	"testing"
)

// The top bar wraps the entries that do not fit onto a further row only in
// a window narrower than the bar, and the map starts below its last row; at
// the default 1280x800 the bar is one row. Fails when a bar entry lies past
// the window's right edge, where neither its label nor its press reaches.
func TestEditorBarWrapsOnlyInANarrowWindow(t *testing.T) {
	e, _ := popupEditor(t)
	_ = e.Open("popups")
	// The installed fonts advance about 7 pixels a character, which makes
	// the English bar about 850 pixels wide.
	e.font = panelFont()
	for k := range e.font.Glyphs {
		e.font.Glyphs[k].Advance = 7
	}
	for _, size := range []image.Point{{640, 480}, {800, 600}, {1280, 800}} {
		e.Layout(size.X, size.Y)
		rows := map[int]bool{}
		row := e.barEntries()[0].rect.Max.Y + 3
		for _, b := range e.barEntries() {
			if b.rect.Min.X < 0 || b.rect.Max.X > size.X || b.rect.Max.Y > e.barHeight() {
				t.Errorf("%v: bar entry %s at %v lies outside the bar (0,0)-(%d,%d)", size, b.key, b.rect, size.X, e.barHeight())
			}
			rows[b.rect.Min.Y] = true
		}
		if area := e.mapArea(); area.Min.Y != e.barHeight() || e.statusRect().Dy() != row {
			t.Errorf("%v: map area %v under a bar of %d and a status strip %v", size, area, e.barHeight(), e.statusRect())
		}
		if size.X == 1280 && (len(rows) != 1 || e.barHeight() != row) {
			t.Errorf("%v: the bar has %d rows and height %d at the default size", size, len(rows), e.barHeight())
		}
		t.Logf("%v: %d bar rows, bar height %d, map area %v", size, len(rows), e.barHeight(), e.mapArea())
	}
}
