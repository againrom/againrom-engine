package ui

import (
	"image"
	"testing"
	"time"
)

// edgeDrag presses the front popup at the point pick picks on its rectangle,
// moves the pointer by d and lets go; it returns the rectangle before and after.
func edgeDrag(t *testing.T, e *MapEditor, pick func(r image.Rectangle) image.Point, d image.Point) (image.Rectangle, image.Rectangle) {
	t.Helper()
	before, ok := e.PopupRect()
	if !ok {
		t.Fatal("no popup")
	}
	p := pick(before)
	q := p.Add(d)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y, PrimaryDown: true}, time.Time{})
	e.Pointer(Input{CursorX: q.X, CursorY: q.Y}, time.Time{})
	after, _ := e.PopupRect()
	return before, after
}

var (
	rightEdge   = func(r image.Rectangle) image.Point { return image.Pt(r.Max.X-2, (r.Min.Y+r.Max.Y)/2) }
	bottomEdge  = func(r image.Rectangle) image.Point { return image.Pt((r.Min.X+r.Max.X)/2, r.Max.Y-2) }
	leftEdge    = func(r image.Rectangle) image.Point { return image.Pt(r.Min.X+1, (r.Min.Y+r.Max.Y)/2) }
	topEdge     = func(r image.Rectangle) image.Point { return image.Pt((r.Min.X+r.Max.X)/2, r.Min.Y+1) }
	bottomRight = func(r image.Rectangle) image.Point { return image.Pt(r.Max.X-2, r.Max.Y-2) }
	topLeft     = func(r image.Rectangle) image.Point { return image.Pt(r.Min.X+1, r.Min.Y+1) }
)

// resizeChecks runs the border drags on the brush list and reports the first
// result that is not a resize.
func resizeChecks(t *testing.T) string {
	t.Helper()
	e, _ := brushEditor(t)
	_ = e.Key("terrain")
	e.ShowCell(image.Pt(30, 30))
	listRows := func() int {
		s, _ := e.listSpec()
		return layoutPopup(s, e.font, e.bodyFont()).ListRows
	}
	rows0 := listRows()
	if b, a := edgeDrag(t, e, rightEdge, image.Pt(70, 0)); a.Dx() != b.Dx()+70 || a.Min != b.Min || a.Dy() != b.Dy() {
		return "right edge: " + b.String() + " -> " + a.String()
	}
	if b, a := edgeDrag(t, e, bottomEdge, image.Pt(0, 90)); a.Dy() != b.Dy()+90 || a.Min != b.Min || a.Dx() != b.Dx() {
		return "bottom edge: " + b.String() + " -> " + a.String()
	}
	if rows1 := listRows(); rows1 <= rows0 {
		return "a taller list shows no more rows"
	}
	if b, a := edgeDrag(t, e, bottomRight, image.Pt(-30, -50)); a.Dx() != b.Dx()-30 || a.Dy() != b.Dy()-50 || a.Min != b.Min {
		return "corner: " + b.String() + " -> " + a.String()
	}
	if b, a := edgeDrag(t, e, leftEdge, image.Pt(25, 0)); a.Dx() != b.Dx()-25 || a.Max.X != b.Max.X {
		return "left edge: " + b.String() + " -> " + a.String()
	}
	if b, a := edgeDrag(t, e, topLeft, image.Pt(-10, 20)); a.Max != b.Max || a.Dx() != b.Dx()+10 || a.Dy() != b.Dy()-20 {
		return "top-left corner: " + b.String() + " -> " + a.String()
	}
	if b, a := edgeDrag(t, e, topEdge, image.Pt(0, -15)); a.Max != b.Max || a.Dy() != b.Dy()+15 {
		return "top edge: " + b.String() + " -> " + a.String()
	}
	return ""
}

// A drag on any border or corner of a window resizes it, the opposite border
// stays, and a taller list shows more rows.
func TestEditorWindowResizesByItsBorders(t *testing.T) {
	if msg := resizeChecks(t); msg != "" {
		t.Fatal(msg)
	}
}

// Loss control: with the builder's resizing off, the same drags change
// nothing.
func TestEditorWindowResizeLossControl(t *testing.T) {
	popupResizeEnabled = false
	defer func() { popupResizeEnabled = true }()
	if msg := resizeChecks(t); msg == "" {
		t.Fatal("the drags resized a window whose resizing is off")
	}
}

// The window never shrinks below the size where its controls work: every row
// button stays inside it, the list keeps three rows, and the size holds while
// the window stays open and ends with it.
func TestEditorWindowResizeMinimumAndMemory(t *testing.T) {
	e, _ := fillEditor(t)
	_, small := edgeDrag(t, e, bottomRight, image.Pt(-5000, -5000))
	if small.Dx() < 240 || small.Dx() >= 400 {
		t.Fatalf("least width %d", small.Dx())
	}
	s, _ := e.listSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	if l.ListRows < popupMinRows {
		t.Fatalf("list keeps %d rows", l.ListRows)
	}
	for _, r := range l.Rows {
		for _, b := range r.Buttons {
			if !b.In(small) || b.Dx() < 40 {
				t.Fatalf("row %d button %v outside the least window %v", r.Index, b, small)
			}
		}
		if r.Control.Dx() < 0 || r.Label.Dx() < 40 {
			t.Fatalf("row %d label %v control %v", r.Index, r.Label, r.Control)
		}
	}
	// A second try to shrink does not move the window or shrink it further.
	_, again := edgeDrag(t, e, bottomRight, image.Pt(-50, -50))
	if again != small {
		t.Fatalf("below the least size the window went from %v to %v", small, again)
	}
	_ = e.Key("height") // the same window with another brush keeps its size
	if r, _ := e.PopupRect(); r.Dx() > small.Dx() {
		t.Fatalf("a new brush gave the window width %d, past the dragged %d", r.Dx(), small.Dx())
	}
	_ = e.Key("escape")
	_ = e.Key("terrain")
	if r, _ := e.PopupRect(); r.Dx() <= small.Dx() {
		t.Fatalf("a reopened window has width %d, not its natural one", r.Dx())
	}
}

// A window cannot be dragged past the editor's bounds.
func TestEditorWindowResizeStaysInTheWindow(t *testing.T) {
	e, _ := brushEditor(t)
	_ = e.Key("terrain")
	_, big := edgeDrag(t, e, bottomRight, image.Pt(5000, 5000))
	if b := e.popupBounds(); !big.In(b) {
		t.Fatalf("window %v left the bounds %v", big, b)
	}
	if big.Max.X != e.popupBounds().Max.X || big.Max.Y != e.popupBounds().Max.Y {
		t.Fatalf("window %v does not reach the corner of %v", big, e.popupBounds())
	}
}

// followChecks drags the object popup wider, then shorter, and reports the
// first way its content does not follow: fields stretch with the width, the
// rows that fit follow the height, and a row-scrolled popup shows its scroll
// bar once rows no longer fit.
func followChecks(t *testing.T) string {
	t.Helper()
	e, _ := popupEditor(t)
	_ = e.Open("popups")
	e.Select(0, false)
	_, l0 := objectLayout(t, e)
	var edit0 popupRowRect
	for _, r := range l0.Rows {
		if r.Control.Dx() > 0 && !r.Control.Empty() {
			edit0 = r
		}
	}
	if edit0.Control.Empty() {
		return "the unit popup has no field"
	}
	if b, a := edgeDrag(t, e, leftEdge, image.Pt(-100, 0)); a.Dx() != b.Dx()+100 || a.Max.X != b.Max.X {
		return "left edge did not widen: " + b.String() + " -> " + a.String()
	}
	_, l1 := objectLayout(t, e)
	var edit1 popupRowRect
	for _, r := range l1.Rows {
		if r.Index == edit0.Index {
			edit1 = r
		}
	}
	if edit1.Control.Dx() < edit0.Control.Dx()+50 {
		return "the field did not stretch with the width"
	}
	fit0 := l1.RowsFit
	_, a := edgeDrag(t, e, bottomEdge, image.Pt(0, -150))
	_, l2 := objectLayout(t, e)
	if l2.RowsFit >= fit0 || a.Dy() >= l1.Rect.Dy() {
		return "fewer rows did not fit a shorter popup"
	}
	if l2.RowBar.Empty() {
		return "no scroll bar once rows no longer fit"
	}
	return ""
}

// The object popup lays its content out from its size.
func TestEditorObjectPopupContentFollowsSize(t *testing.T) {
	if msg := followChecks(t); msg != "" {
		t.Fatal(msg)
	}
}

// Loss control: with resizing off the same drags leave the content alone.
func TestEditorObjectPopupContentLossControl(t *testing.T) {
	popupResizeEnabled = false
	defer func() { popupResizeEnabled = true }()
	if msg := followChecks(t); msg == "" {
		t.Fatal("the content followed a size that resizing off could not change")
	}
}

// The map panel, the notice and the group popup resize by their borders and
// stop at the window's bounds.
func TestEditorPanelNoticeAndGroupResizeByBorders(t *testing.T) {
	for _, c := range []struct {
		name string
		open func(t *testing.T) *MapEditor
	}{
		{"map panel", func(t *testing.T) *MapEditor {
			f, rows := endless()
			e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
			return e
		}},
		{"notice", func(t *testing.T) *MapEditor {
			f, rows := endless()
			e, _ := mapPanelEditor(t, &mapPanelEdits{facts: f, rows: rows}, nil)
			_ = e.Key("testplay")
			return e
		}},
		{"group popup", func(t *testing.T) *MapEditor {
			e, _ := popupEditor(t)
			_ = e.Open("popups")
			openGroup(t, e)
			return e
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			e := c.open(t)
			before, _ := e.PopupRect()
			room := e.popupBounds().Max.Sub(before.Max)
			d := image.Pt(min(50, room.X), min(30, room.Y))
			_, after := edgeDrag(t, e, bottomRight, d)
			if d == (image.Point{}) || after.Min != before.Min || after.Dx() != before.Dx()+d.X || after.Dy() != before.Dy()+d.Y {
				t.Fatalf("corner drag by %v: %v -> %v", d, before, after)
			}
			_, far := edgeDrag(t, e, bottomRight, image.Pt(5000, 5000))
			if !far.In(e.popupBounds()) {
				t.Fatalf("window %v left the bounds %v", far, e.popupBounds())
			}
			_, least := edgeDrag(t, e, bottomRight, image.Pt(-5000, -5000))
			if least.Dx() < 160 || least.Dy() < 60 || least.Dx() > before.Dx() {
				t.Fatalf("least size %v from %v", least, before)
			}
		})
	}
}

// A size belongs to the window it was dragged on: the window that replaces
// it in the same place opens at its own natural size.
func TestReplacedWindowOpensAtItsNaturalSize(t *testing.T) {
	fresh, _ := fillEditor(t)
	_ = fresh.Key("catalog")
	natural, ok := fresh.PopupRect()
	if !ok {
		t.Fatal("no catalog")
	}
	e, _ := fillEditor(t)
	if _, after := edgeDrag(t, e, bottomRight, image.Pt(70, 60)); after.Dx() <= 400 {
		t.Fatalf("the brush list did not widen: %v", after)
	}
	_ = e.Key("catalog")
	if r, _ := e.PopupRect(); r != natural {
		t.Fatalf("the catalog replacing a sized brush list opens at %v, not its natural %v", r, natural)
	}
}
