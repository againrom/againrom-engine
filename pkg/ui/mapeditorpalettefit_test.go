package ui

import (
	"fmt"
	"image"
	"testing"
	"time"
)

// cellCountHost offers as many unit classes, structure kinds and tree kinds
// as the shipped first-game data does: 33, 66 and 75.
type cellCountHost struct {
	*modesFull
	units, structures, trees int
}

func (h *cellCountHost) NewRecordStep(r InspectionNewRecord, key string) InspectionNewStep {
	n := 0
	switch {
	case r.Kind == "Unit" && key == "class":
		n = h.units
	case r.Kind == "Structure" && key == "kind":
		n = h.structures
	default:
		return h.modesFull.NewRecordStep(r, key)
	}
	s := InspectionNewStep{Value: 1}
	for v := 1; v <= n; v++ {
		s.Choices = append(s.Choices, InspectionChoice{Value: int64(v), Label: fmt.Sprintf("%d: Kind %d", v, v)})
	}
	return s
}

func (h *cellCountHost) BrushObjects() []BrushObject {
	out := []BrushObject{{Code: 0, Name: "No object"}}
	for c := 1; c <= h.trees; c++ {
		out = append(out, BrushObject{Code: c, Name: fmt.Sprintf("Tree %d", c)})
	}
	return out
}

func paletteEditor(t *testing.T, w, h int) *MapEditor {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	host := &cellCountHost{modesFull: e.doc.Edits.(*modesFull), units: 33, structures: 66, trees: 75}
	host.self, e.doc.Edits = host, host
	setRightPanel(e, sidePanels[panelRight].width) // both panels open, as at first start
	e.Layout(w, h)
	e.places[popList].placed = false // the palette opens at its default place
	return e
}

// paletteGrid opens bar entry key and lays its window out.
func paletteGrid(t *testing.T, e *MapEditor, key string) (popupSpec, popupLayout) {
	t.Helper()
	chromeClick(e, barPoint(t, e, key))
	s := mustListSpec(t, e)
	if s.Grid == nil {
		t.Fatalf("%s shows no grid", key)
	}
	return s, layoutPopup(s, e.font, e.bodyFont())
}

// A palette opens at the size that shows every cell when the room the map
// gives holds them: wider first, then taller. Structures' large cells need
// more rows than the room holds at 1280x800, so that window takes the room
// down to the map's bottom edge and scrolls the rest.
func TestMapEditorPalettesShowEveryCellTheRoomHolds(t *testing.T) {
	e := paletteEditor(t, 1280, 800)
	room := e.mapArea()
	s, l := paletteGrid(t, e, "units")
	if len(l.GridCells) != len(s.Grid.Cells) || !l.GridBar.Empty() {
		t.Fatalf("Units shows %d of %d cells, scroll bar %v", len(l.GridCells), len(s.Grid.Cells), l.GridBar)
	}
	if !l.Rect.In(e.popupBounds()) || l.Rect.Dx() > room.Dx() {
		t.Fatalf("Units window %v leaves the room %v", l.Rect, room)
	}
	if l.GridCols < len(s.Grid.Cells) && room.Max.X-l.Rect.Max.X >= s.Grid.side() {
		t.Fatalf("Units window %v grew taller (%d columns) while the room %v had width", l.Rect, l.GridCols, room)
	}

	s, l = paletteGrid(t, e, "structures")
	if len(l.GridCells) == len(s.Grid.Cells) {
		t.Fatalf("Structures shows all %d cells at 1280x800; this test expects the room to fall short", len(l.GridCells))
	}
	if l.GridBar.Empty() || room.Max.Y-l.Rect.Max.Y >= s.Grid.side() {
		t.Fatalf("Structures window %v does not take the room %v with a scroll bar (%v)", l.Rect, room, l.GridBar)
	}
}

// Loss control: at 800x600 the Units grid still shows every cell, the
// Structures grid cannot; its scroll bar remains and the wheel over the
// grid scrolls to the last row.
func TestMapEditorPaletteScrollsWhereTheRoomFallsShort(t *testing.T) {
	e := paletteEditor(t, 800, 600)
	s, l := paletteGrid(t, e, "units")
	if len(l.GridCells) != len(s.Grid.Cells) || !l.GridBar.Empty() {
		t.Fatalf("Units at 800x600 shows %d of %d cells, scroll bar %v", len(l.GridCells), len(s.Grid.Cells), l.GridBar)
	}
	s, l = paletteGrid(t, e, "structures")
	if len(l.GridCells) >= len(s.Grid.Cells) || l.GridBar.Empty() {
		t.Fatalf("Structures at 800x600 shows %d of %d cells, scroll bar %v", len(l.GridCells), len(s.Grid.Cells), l.GridBar)
	}
	p := center(l.Grid)
	for range 40 {
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: -1}, time.Unix(1, 0))
	}
	l = layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
	if l.GridFirst == 0 || l.GridFirst+len(l.GridCells) != len(s.Grid.Cells) {
		t.Fatalf("after the wheel the grid shows cells %d..%d of %d", l.GridFirst, l.GridFirst+len(l.GridCells), len(s.Grid.Cells))
	}
}

// A size the border drag set is kept: the palette scrolls in it, and
// leaving for another tab and coming back shows it at that size again.
func TestMapEditorPaletteKeepsItsDraggedSize(t *testing.T) {
	e := paletteEditor(t, 1280, 800)
	_, l := paletteGrid(t, e, "units")
	e.drag, e.dragEdge = popList, edgeBottom
	e.resizePopup(image.Pt(0, -l.Rect.Dy()/2))
	e.drag, e.dragEdge = popNone, 0
	want, dragged := e.places[popList].size, layoutPopup(mustListSpec(t, e), e.font, e.bodyFont()).Rect.Size()
	if want == (image.Point{}) || dragged.Y >= l.Rect.Dy() {
		t.Fatalf("the drag set size %v, window %v from %v", want, dragged, l.Rect.Size())
	}
	check := func(when string) {
		t.Helper()
		l := layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
		if e.places[popList].size != want || l.Rect.Size() != dragged || l.GridBar.Empty() {
			t.Fatalf("%s: window %v, size %v, want %v at size %v with a scroll bar (%v)", when, l.Rect.Size(), e.places[popList].size, dragged, want, l.GridBar)
		}
	}
	check("after the drag")
	paletteGrid(t, e, "structures")
	paletteGrid(t, e, "units")
	check("back on Units")
}

// A window that still shows a list keeps its list rows: the Trees brush
// window's list is as long as the list window gives it, and its grid takes
// at least its least rows below.
func TestMapEditorBrushWindowKeepsItsListRows(t *testing.T) {
	e := paletteEditor(t, 1280, 800)
	s, l := paletteGrid(t, e, "trees")
	if s.List == nil {
		t.Fatal("the Trees brush window shows no list")
	}
	pitch := fontHeight(e.bodyFont(), 15) + listPitchExtra
	avail := e.mapArea().Dy() - 4*(fontHeight(e.bodyFont(), 15)+popupRowExtra) - 40 - e.forestGridHeight()
	if want := max(3, min(18, avail/pitch)); s.ListRows != want || l.ListRows != want {
		t.Fatalf("the list shows %d rows (spec %d), want %d", l.ListRows, s.ListRows, want)
	}
	if l.GridRows < gridLeastRows {
		t.Fatalf("the forest grid shows %d rows, want at least %d", l.GridRows, gridLeastRows)
	}
}
