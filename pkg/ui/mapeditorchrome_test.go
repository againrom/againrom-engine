package ui

import (
	"image"
	"testing"
	"time"
)

// chromeClick presses and releases the primary button at p.
func chromeClick(e *MapEditor, p image.Point) {
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Time{})
}

func center(r image.Rectangle) image.Point {
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// barPoint is the centre of the bar entry key.
func barPoint(t *testing.T, e *MapEditor, key string) image.Point {
	t.Helper()
	for _, b := range e.barEntries() {
		if b.key == key {
			return center(b.rect)
		}
	}
	t.Fatalf("no bar entry %q", key)
	return image.Point{}
}

// menuPoint is the centre of row n of the open menu.
// menuKeyRow is the row of menu whose action is key.
func menuKeyRow(t *testing.T, e *MapEditor, menu, key string) int {
	t.Helper()
	n, ok := e.MenuRow(menu, key)
	if !ok {
		t.Fatalf("the %s menu has no %s row", menu, key)
	}
	return n
}

func menuPoint(t *testing.T, e *MapEditor, n int) image.Point {
	t.Helper()
	b, ok := e.openMenuEntry()
	if !ok {
		t.Fatal("no menu open")
	}
	d := e.menuDropdown(b, true)
	return center(d.listBox(e.font).Row(n))
}

func listLayout(t *testing.T, e *MapEditor) popupLayout {
	t.Helper()
	s, ok := e.listSpec()
	if !ok {
		t.Fatal("no list popup")
	}
	return layoutPopup(s, e.font, e.bodyFont())
}

// listRowPoint is the centre of list row n, which must be visible.
func listRowPoint(t *testing.T, e *MapEditor, n int) image.Point {
	t.Helper()
	l := listLayout(t, e)
	top, count := e.model.Visible()
	if n < top || n >= top+count {
		t.Fatalf("list row %d outside the visible %d..%d", n, top, top+count)
	}
	return center(l.List.Row(n - top))
}

// treeMemberPoint is the centre of member n of category key's tree
// branch, the tree scrolled to show it.
func treeMemberPoint(t *testing.T, e *MapEditor, key string, n int) image.Point {
	t.Helper()
	row := e.TreeMemberRow(key, n)
	r, ok := e.TreeRowRect(row)
	if row < 0 || !ok {
		t.Fatalf("member %d of the %s branch is not in the tree", n, key)
	}
	return center(r)
}

func objectLayout(t *testing.T, e *MapEditor) (popupSpec, popupLayout) {
	t.Helper()
	s, _, ok := e.objectSpec()
	if !ok {
		t.Fatal("no object popup")
	}
	return s, layoutPopup(s, e.font, e.bodyFont())
}

// objectRowRect is the shown row n of the object popup.
func objectRowRect(t *testing.T, e *MapEditor, n int) popupRowRect {
	t.Helper()
	_, l := objectLayout(t, e)
	for _, r := range l.Rows {
		if r.Index == n {
			return r
		}
	}
	t.Fatalf("object row %d not shown", n)
	return popupRowRect{}
}

func objectButtonPoint(t *testing.T, e *MapEditor, n int) image.Point {
	t.Helper()
	_, l := objectLayout(t, e)
	if n >= len(l.Buttons) {
		t.Fatalf("object popup has %d buttons", len(l.Buttons))
	}
	return center(l.Buttons[n])
}

func promptLayout(t *testing.T, e *MapEditor) (popupSpec, popupLayout) {
	t.Helper()
	s, ok := e.promptSpec()
	if !ok {
		t.Fatal("no prompt popup")
	}
	return s, layoutPopup(s, e.font, e.bodyFont())
}

// promptText is every row and note of the prompt popup.
func promptText(t *testing.T, e *MapEditor) []string {
	t.Helper()
	s, _ := promptLayout(t, e)
	var out []string
	for _, r := range s.Rows {
		out = append(out, r.Value)
	}
	return append(out, s.Notes...)
}

// mapPoint is a window point over the map's frame position p.
func mapPoint(e *MapEditor, p image.Point) image.Point {
	ox, oy := e.doc.Viewer.place.Origin()
	return image.Pt(p.X+int(ox), p.Y+int(oy))
}
