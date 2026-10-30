package ui

import (
	"image"
	"testing"
	"time"
)

// barArrowClick presses and releases the arrow button of the bar drop-down key.
func barArrowClick(t *testing.T, e *MapEditor, key string) {
	t.Helper()
	for _, b := range e.barEntries() {
		if b.key != key {
			continue
		}
		arrow := e.menuDropdown(b, false).button()
		pt := image.Pt((arrow.Min.X+arrow.Max.X)/2, (arrow.Min.Y+arrow.Max.Y)/2)
		e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y, PrimaryDown: true}, time.Unix(1, 0))
		e.Pointer(Input{CursorX: pt.X, CursorY: pt.Y}, time.Unix(2, 0))
		return
	}
	t.Fatalf("no %s entry", key)
}

// With any bar menu open, a press on another bar drop-down's arrow closes the
// open menu and opens that one; a press on the open menu's own arrow closes it.
// The open list is wider than its box, so it must not swallow the arrows of the
// entries beside it.
func TestMapEditorBarMenusOpenFromAnyOpenMenu(t *testing.T) {
	menus := []string{"map", "view", "brushes", "tools"}
	for _, first := range menus {
		for _, second := range menus {
			if first == second {
				continue
			}
			e, _ := passEditor(t)
			barArrowClick(t, e, first)
			if e.menu != first {
				t.Fatalf("%s: arrow opened menu %q", first, e.menu)
			}
			barArrowClick(t, e, second)
			if e.menu != second {
				t.Errorf("%s open, press on %s arrow: menu %q, want %q", first, second, e.menu, second)
			}
			barArrowClick(t, e, second)
			if e.menu != "" {
				t.Errorf("%s then %s, press on its own arrow: menu %q, want none", first, second, e.menu)
			}
		}
	}
}
