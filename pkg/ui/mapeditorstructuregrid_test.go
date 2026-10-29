package ui

import (
	"image"
	"testing"
)

func TestMapEditorStructureGridDrawsLargeCells(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	e.Layout(640, 480)
	e.listAt = image.Pt(8, e.barHeight()+8)

	chromeClick(e, barPoint(t, e, "units"))
	units, ok := e.PlaceGridCellRect(0)
	if !ok {
		t.Fatal("the Units grid shows no cell")
	}
	if units.Dx() != gridCellSide-2 || units.Dy() != gridCellSide-2 {
		t.Fatalf("a Units cell is %v, want the common %d square", units.Size(), gridCellSide-2)
	}

	chromeClick(e, barPoint(t, e, "structures"))
	cell, ok := e.PlaceGridCellRect(0)
	if !ok {
		t.Fatal("the Structures grid shows no cell")
	}
	box := func(r image.Rectangle) int { return r.Dx() + 2 - 8 }
	if box(cell) != 2*box(units) || cell.Dx() != cell.Dy() {
		t.Fatalf("a Structures cell is %v: its picture box is %d, want twice the Units %d", cell.Size(), box(cell), box(units))
	}
	l := layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
	if len(l.GridCells) == 0 || l.GridCols < 1 {
		t.Fatal("no grid cells shown")
	}
	pop, ok := e.PopupRect()
	if !ok || !pop.In(image.Rect(0, 0, 640, 480)) {
		t.Fatalf("the popup %v leaves the 640x480 window", pop)
	}
	if !l.Grid.In(pop) {
		t.Fatalf("the grid %v leaves the popup %v", l.Grid, pop)
	}
	for i, r := range l.GridCells {
		if !r.In(l.Grid) {
			t.Fatalf("cell %d %v leaves the grid %v", i, r, l.Grid)
		}
	}

	chromeClick(e, center(cell))
	run, ok := e.addRunning()
	if !ok || run.kind != "Structure" || !run.repeat || e.pick != pickAddCell {
		t.Fatalf("a press on the first cell's centre opened no placing: %+v", run)
	}
	if len(calls.lines) != 0 {
		t.Fatalf("the press called the host: %v", calls.lines)
	}
	if edge := cell.Max.Sub(image.Pt(1, 1)); popupHitCell(t, e, edge) != 0 {
		t.Fatal("the last pixel of cell 0 does not hit cell 0")
	}
	if popupHitCell(t, e, cell.Max) == 0 {
		t.Fatal("a pixel past cell 0 hits cell 0")
	}
}

func mustListSpec(t *testing.T, e *MapEditor) popupSpec {
	t.Helper()
	s, ok := e.listSpec()
	if !ok {
		t.Fatal("no list popup")
	}
	return s
}

func popupHitCell(t *testing.T, e *MapEditor, p image.Point) int {
	t.Helper()
	l := layoutPopup(mustListSpec(t, e), e.font, e.bodyFont())
	if part, n := l.hit(p); part == popupGridPart {
		return n
	}
	return -1
}
