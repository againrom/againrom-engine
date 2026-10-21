package game

import (
	"image"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// borderDrag presses the front popup at pick(rect), moves by d, lets go, and
// returns the popup rectangle before and after.
func borderDrag(t *testing.T, e *ui.MapEditor, pick func(r image.Rectangle) image.Point, d image.Point) (image.Rectangle, image.Rectangle) {
	t.Helper()
	before, ok := e.PopupRect()
	if !ok {
		t.Fatal("no popup")
	}
	p := pick(before)
	q := p.Add(d)
	e.Pointer(ui.Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(ui.Input{CursorX: q.X, CursorY: q.Y}, time.Time{})
	after, _ := e.PopupRect()
	return before, after
}

// Every editor window resizes by a border or a corner: a drag by d changes
// the window by d where there is room, a drag far inward stops at a size
// that keeps the window's controls (never below 160 by 80 pixels), and the
// size holds on the next frame. A press in the body resizes nothing.
func TestEditorPopupsResizeByTheirBorders(t *testing.T) {
	for _, c := range popupCases() {
		t.Run(c.name, func(t *testing.T) {
			e := c.open(t)
			r0, _ := e.PopupRect()
			// Grow toward the side that has room: the corner that holds the
			// opposite one still.
			var grow image.Point
			x := func(r image.Rectangle) int { return r.Max.X - 2 }
			y := func(r image.Rectangle) int { return r.Max.Y - 2 }
			switch {
			case r0.Max.X+60 <= 1280:
				grow.X = 60
			case r0.Min.X-60 >= 0:
				grow.X, x = -60, func(r image.Rectangle) int { return r.Min.X + 1 }
			}
			switch {
			case r0.Max.Y+40 <= 700:
				grow.Y = 40
			case r0.Min.Y-40 >= 60:
				grow.Y, y = -40, func(r image.Rectangle) int { return r.Min.Y + 1 }
			}
			if grow == (image.Point{}) {
				t.Skipf("no room to grow %v", r0)
			}
			corner := func(r image.Rectangle) image.Point { return image.Pt(x(r), y(r)) }
			before, after := borderDrag(t, e, corner, grow)
			want := image.Pt(before.Dx()+absInt(grow.X), before.Dy()+absInt(grow.Y))
			if after.Size() != want {
				t.Fatalf("corner drag by %v: %v -> %v, want size %v", grow, before, after, want)
			}
			if grow.X > 0 && after.Min.X != before.Min.X || grow.X < 0 && after.Max.X != before.Max.X ||
				grow.Y > 0 && after.Min.Y != before.Min.Y || grow.Y < 0 && after.Max.Y != before.Max.Y {
				t.Fatalf("the opposite border moved: %v -> %v", before, after)
			}
			held, _ := e.PopupRect()
			if held != after {
				t.Fatalf("the size did not hold: %v then %v", after, held)
			}
			_, tiny := borderDrag(t, e, corner, image.Pt(-grow.X*80, -grow.Y*80))
			if tiny.Dx() < 160 || tiny.Dy() < 80 || tiny.Dx() > after.Dx() || tiny.Dy() > after.Dy() {
				t.Fatalf("least size %v from natural %v", tiny, before)
			}
			// A press below the title bar and inside the borders resizes nothing.
			mid := func(r image.Rectangle) image.Point { return image.Pt(r.Min.X+8, r.Min.Y+r.Dy()/2) }
			if b, a := borderDrag(t, e, mid, image.Pt(0, 0)); b != a {
				t.Fatalf("a press in the body resized %v to %v", b, a)
			}
		})
	}
}
