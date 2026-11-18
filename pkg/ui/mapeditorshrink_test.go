package ui

import (
	"image"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// shrinkWindows opens one editor window of each place and kind that can be
// dragged below the editor's bottom edge.
func shrinkWindows() []struct {
	name string
	open func(t *testing.T) *MapEditor
} {
	return []struct {
		name string
		open func(t *testing.T) *MapEditor
	}{
		{"brush list", func(t *testing.T) *MapEditor {
			e, _ := brushEditor(t)
			_ = e.Key("terrain")
			e.ShowCell(image.Pt(30, 30))
			return e
		}},
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
		{"trigger window", func(t *testing.T) *MapEditor {
			e, _ := diplomacyEditor(t, "en", 0)
			return e
		}},
		{"unit palette", func(t *testing.T) *MapEditor {
			e, _ := newModesEditor(t, "full")
			host := &cellCountHost{modesFull: e.doc.Edits.(*modesFull), units: 66}
			host.self, e.doc.Edits = host, host
			_ = e.Key("unit")
			return e
		}},
		{"unit object", func(t *testing.T) *MapEditor {
			e, _ := popupEditor(t)
			_ = e.Open("popups")
			e.Select(0, false)
			return e
		}},
	}
}

// frontPlace is the place of the front editor window.
func frontPlace(e *MapEditor) editorPopupID {
	for _, id := range []editorPopupID{popPrompt, popObject, popList} {
		if _, ok := e.movableSpec(id); ok {
			return id
		}
	}
	return popNone
}

// A window dragged below the editor's bottom edge is shortened by its top
// border: the bottom edge stays, the top follows the pointer down to the
// least height, and every row stays in reach of the scroll.
func TestEditorWindowPastTheBottomShortensByItsTopBorder(t *testing.T) {
	for _, c := range shrinkWindows() {
		t.Run(c.name, func(t *testing.T) {
			e := c.open(t)
			e.Layout(640, 480)
			if _, ok := e.PopupRect(); !ok {
				t.Fatal("no window")
			}
			start, _ := e.PopupRect()
			dragTitle(t, e, image.Pt(0, e.popupBounds().Max.Y+40-start.Max.Y))
			before, _ := e.PopupRect()
			if bounds := e.popupBounds(); before.Max.Y <= bounds.Max.Y {
				t.Fatalf("window %v does not reach past the bottom of %v", before, bounds)
			}
			b, a := edgeDrag(t, e, topEdge, image.Pt(0, 120))
			if a.Max.Y != b.Max.Y || a.Min.X != b.Min.X || a.Dx() != b.Dx() {
				t.Fatalf("shortened window %v -> %v moved its bottom or sides", b, a)
			}
			least := a
			if a.Dy() != b.Dy()-120 {
				_, least = edgeDrag(t, e, topEdge, image.Pt(0, 5000))
				if a.Dy() != least.Dy() || a.Dy() >= b.Dy() {
					t.Fatalf("window %v -> %v: neither 120 px shorter nor at its least height %d", b, a, least.Dy())
				}
			}
			// The shorter window is moved up into the editor to read it.
			if r, _ := e.PopupRect(); r.Max.Y > e.popupBounds().Max.Y {
				dragTitle(t, e, image.Pt(0, e.popupBounds().Max.Y-r.Max.Y))
			}
			s, ok := e.movableSpec(frontPlace(e))
			if !ok {
				t.Fatal("window closed")
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			w := l.window(s)
			if !l.RowBar.Empty() {
				at := center(l.RowsViewport)
				for range popupRowCount(w) + 2 {
					e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -1}, time.Unix(1, 0))
				}
				s, _ = e.movableSpec(frontPlace(e))
				l = layoutPopup(s, e.font, e.bodyFont())
				last := false
				for _, rr := range l.Rows {
					last = last || rr.Index == len(s.Rows)-1
				}
				if !last {
					t.Fatalf("the wheel does not reach the last of %d rows in %v", len(s.Rows), l.Rect)
				}
			}
			if s.Grid != nil {
				if l.GridBar.Empty() {
					t.Fatal("the shortened palette needs a scrollable grid")
				}
				at := center(l.Grid)
				for range len(s.Grid.Cells) {
					e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -1}, time.Unix(1, 0))
				}
				if _, ok := e.FindControl(ControlQuery{Role: ControlPlaceGridCell, N: len(s.Grid.Cells) - 1}); !ok {
					t.Fatal("the wheel does not reach the last palette choice")
				}
			}
		})
	}
}

// A press on the title row below the top band still moves the window, and
// the pointer shows the up-and-down shape over the top band.
func TestEditorWindowTitleMovesBelowTheTopBand(t *testing.T) {
	var set []ebiten.CursorShapeType
	was := setEditorCursorShape
	setEditorCursorShape = func(s ebiten.CursorShapeType) { set = append(set, s) }
	t.Cleanup(func() { setEditorCursorShape = was })
	e, _ := brushEditor(t)
	_ = e.Key("terrain")
	e.ShowCell(image.Pt(30, 30))
	r, _ := e.PopupRect()
	p := image.Pt((r.Min.X+r.Max.X)/2, r.Min.Y+popupTopBand-1)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Unix(1, 0))
	e.windowCursor()
	if e.cursorShape != ebiten.CursorShapeNSResize {
		t.Fatalf("cursor %d over the top band, want the up-and-down shape", e.cursorShape)
	}
	before, after := dragTitle(t, e, image.Pt(30, 20))
	if after.Min != before.Min.Add(image.Pt(30, 20)) {
		t.Fatalf("title %v -> %v, not moved by (30,20)", before, after)
	}
	if now, _ := e.PopupRect(); now.Dy() != r.Dy() {
		t.Fatalf("moving the title resized the window %v -> %v", r, now)
	}
}

// The detailed unit view opens as wide as its columns need, never as wide as
// the editor.
func TestEditorDetailedUnitWindowIsAsWideAsItsContent(t *testing.T) {
	for _, size := range []image.Point{{1920, 1080}, {1280, 1024}, {640, 480}} {
		e := wideEditor(t, "en", 0, "technical", nil)
		e.Layout(size.X, size.Y)
		e.Select(viewsUnit, false)
		s, _, ok := e.objectSpec()
		if !ok || e.PlainView() {
			t.Fatalf("%v: no detailed unit window", size)
		}
		l := layoutPopup(s, e.font, e.bodyFont())
		if l.Rect.Dx() > size.X {
			t.Errorf("%v: window %v is wider than the editor", size, l.Rect)
		}
		if size.X < 1280 {
			continue
		}
		bar := widgetSpriteSize + 2
		column := e.unitColumnWidth(s)
		if want := s.Columns * column; l.Rect.Dx() < want || l.Rect.Dx() > want+2*popupPad+bar || l.Rect.Dx() >= size.X-100 {
			t.Errorf("%v: window %v is not as wide as its %d columns of %d need", size, l.Rect, s.Columns, column)
		}
		for _, rr := range l.Rows {
			if !rr.Rect.In(l.Rect) {
				t.Errorf("%v: row %d at %v leaves the window %v", size, rr.Index, rr.Rect, l.Rect)
			}
		}
	}
}
