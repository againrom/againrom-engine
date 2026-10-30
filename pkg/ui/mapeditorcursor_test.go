package ui

import (
	"image"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// The pointer over a border of an editor window shows the system resize
// cursor: sideways, up and down, or diagonal at a corner; away from the
// borders, and while a menu is open, the default.
func TestMapEditorWindowCursorFollowsResizableBorders(t *testing.T) {
	var set []ebiten.CursorShapeType
	was := setEditorCursorShape
	setEditorCursorShape = func(s ebiten.CursorShapeType) { set = append(set, s) }
	t.Cleanup(func() { setEditorCursorShape = was })

	e, _ := newModesEditor(t, "full")
	e.listAt = image.Pt(300, 200)
	e.showRecords(1)
	r, ok := e.PopupRect()
	if !ok {
		t.Fatal("no popup")
	}
	at := func(p image.Point) ebiten.CursorShapeType {
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
		e.windowCursor()
		return e.cursorShape
	}
	mid := image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
	for _, c := range []struct {
		name string
		p    image.Point
		want ebiten.CursorShapeType
	}{
		{"inside", mid, ebiten.CursorShapeDefault},
		{"bottom right corner", image.Pt(r.Max.X-3, r.Max.Y-3), ebiten.CursorShapeNWSEResize},
		{"right side", image.Pt(r.Max.X-3, r.Min.Y+80), ebiten.CursorShapeEWResize},
		{"bottom side", image.Pt(r.Min.X+60, r.Max.Y-3), ebiten.CursorShapeNSResize},
		{"left side", image.Pt(r.Min.X+2, r.Min.Y+80), ebiten.CursorShapeEWResize},
		{"outside", image.Pt(r.Max.X+40, r.Max.Y+40), ebiten.CursorShapeDefault},
	} {
		if got := at(c.p); got != c.want {
			t.Errorf("%s at %v: cursor %d, want %d", c.name, c.p, got, c.want)
		}
	}
	if len(set) != 5 {
		t.Errorf("the window cursor was set %d times %v, want once per change (5)", len(set), set)
	}
	if got := at(image.Pt(r.Max.X-3, r.Max.Y-3)); got != ebiten.CursorShapeNWSEResize {
		t.Fatalf("corner again: %d", got)
	}
	e.menu = "view"
	if got := at(image.Pt(r.Max.X-3, r.Max.Y-3)); got != ebiten.CursorShapeDefault {
		t.Errorf("with a menu open the border does not resize: cursor %d", got)
	}
}
