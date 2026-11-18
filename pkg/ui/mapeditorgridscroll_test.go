package ui

import (
	"image"
	"testing"
	"time"
)

// bagGrid opens the sack palette at 1920x1080 and returns its layout; the
// grid has more rows than the window shows.
func bagGrid(t *testing.T) (*MapEditor, popupLayout) {
	t.Helper()
	e := newPlacementLayoutEditor(t, "en")
	_, l := paletteGrid(t, e, "sacks")
	if l.GridBar.Empty() || l.GridTop != 0 {
		t.Fatalf("the sack grid shows bar %v at top %d, want a bar at the top", l.GridBar, l.GridTop)
	}
	return e, l
}

func bagLayout(t *testing.T, e *MapEditor) popupLayout {
	t.Helper()
	return layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
}

func TestBagGridWheelOverTheGridMovesTheTop(t *testing.T) {
	e, l := bagGrid(t)
	at := center(l.Grid)
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -3}, time.Unix(1, 0))
	if got := *e.placeTop(); got <= 0 {
		t.Fatalf("the wheel over the grid left the top at %d", got)
	}
}

func TestBagGridTrackClickMovesOnePage(t *testing.T) {
	e, l := bagGrid(t)
	at := image.Pt(l.GridBar.Min.X+4, vScrollBar{Rect: l.GridBar, Count: 2}.Thumb().Max.Y+4)
	chromeClick(e, at)
	if got := *e.placeTop(); got != l.GridRows {
		t.Fatalf("a click on the track below the thumb moved the top to %d, want one page of %d rows", got, l.GridRows)
	}
}

func TestBagGridThumbDragMovesTheTop(t *testing.T) {
	e, l := bagGrid(t)
	s := mustListSpec(t, e)
	most := s.Grid.totalRows(l.GridCols) - l.GridRows
	bar := vScrollBar{Rect: l.GridBar, Pos: 0, Count: most + 1}
	grab := center(bar.Thumb())
	e.Pointer(Input{CursorX: grab.X, CursorY: grab.Y, PrimaryDown: true}, time.Unix(1, 0))
	to := image.Pt(grab.X, grab.Y+(l.GridBar.Dy()-3*l.GridBar.Dx())/2)
	e.Pointer(Input{CursorX: to.X, CursorY: to.Y, PrimaryDown: true}, time.Unix(2, 0))
	e.Pointer(Input{CursorX: to.X, CursorY: to.Y}, time.Unix(3, 0))
	got := *e.placeTop()
	if got < most/3 || got > most {
		t.Fatalf("dragging the thumb half the track moved the top to %d of %d", got, most)
	}
	moved := vScrollBar{Rect: bagLayout(t, e).GridBar, Pos: got, Count: most + 1}.Thumb()
	if d := moved.Min.Y - bar.Thumb().Min.Y; d < l.GridBar.Dy()/4 {
		t.Fatalf("the thumb moved %d pixels", d)
	}
}

func TestBagGridWheelOverTheWindowRowsMovesTheTop(t *testing.T) {
	e, l := bagGrid(t)
	if len(l.Rows) == 0 {
		t.Fatal("the sack window has no row beside its grid")
	}
	at := center(l.Rows[0].Rect)
	if at.In(l.Grid) || at.In(l.GridBar) {
		t.Fatalf("row %v lies on the grid", l.Rows[0].Rect)
	}
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -3}, time.Unix(1, 0))
	if got := *e.placeTop(); got <= 0 {
		t.Fatalf("the wheel over the window's row left the top at %d", got)
	}
}
