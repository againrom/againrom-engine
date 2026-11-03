package ui

import (
	"image"
	"testing"
)

// Owner item 5 aligns the utility group against the supported window edge.
func TestEditorBarWrapsOnlyInANarrowWindow(t *testing.T) {
	e, _ := popupEditor(t)
	_ = e.Open("popups")
	// The installed fonts advance about 7 pixels a character.
	e.font = panelFont()
	for k := range e.font.Glyphs {
		e.font.Glyphs[k].Advance = 7
	}
	for _, size := range []image.Point{{1920, 1080}, {2560, 1440}} {
		e.Layout(size.X, size.Y)
		rows := map[int]bool{}
		row := e.barEntries()[0].rect.Max.Y + 3
		for _, b := range e.barEntries() {
			if b.rect.Min.X < 0 || b.rect.Max.X > size.X || b.rect.Max.Y > e.barHeight() {
				t.Errorf("%v: bar entry %s at %v lies outside the bar (0,0)-(%d,%d)", size, b.key, b.rect, size.X, e.barHeight())
			}
			rows[b.rect.Min.Y] = true
		}
		if tb, area := e.toolbarRect(), e.mapArea(); tb.Min.Y != e.barHeight() || area.Min.Y != tb.Max.Y || e.statusRect().Dy() != row {
			t.Errorf("%v: map area %v under a bar of %d, the toolbar %v and a status strip %v", size, area, e.barHeight(), tb, e.statusRect())
		}
		if len(rows) != 1 || e.barHeight() != row {
			t.Errorf("%v: the bar has %d rows and height %d at the default size", size, len(rows), e.barHeight())
		}
		t.Logf("%v: %d bar rows, bar height %d, map area %v", size, len(rows), e.barHeight(), e.mapArea())
	}
}
