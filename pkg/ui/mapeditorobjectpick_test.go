package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
)

// Move and Delete of the terrain host carry a Scenery record's byte on its
// object plane, so the next document shows the object where the edit put it.
func (p *terrainHost) Move(k string, i int, c image.Point) (*InspectionDocument, error) {
	if k == "Scenery" {
		b := p.objects[i]
		p.objects[i], p.objects[c.Y*terrainSide+c.X] = 0, b
	}
	return p.modesFull.Move(k, i, c)
}

func (p *terrainHost) Delete(k string, i int) (*InspectionDocument, error) {
	if k == "Scenery" {
		p.objects[i] = 0
	}
	return p.modesFull.Delete(k, i)
}

// withObjects reopens the terrain editor's map with object bytes, keyed by
// cell, each a Scenery record of the document.
func withObjects(t *testing.T, e *MapEditor, h *terrainHost, objects map[image.Point]uint8) {
	t.Helper()
	h.objects = make([]uint8, terrainSide*terrainSide)
	for c, b := range objects {
		h.objects[c.Y*terrainSide+c.X] = b
	}
	if err := e.Open("scenario/10.alm"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
}

// sceneryIndex is the record a Scenery on cell is in the open document, or -1.
func sceneryIndex(e *MapEditor, cell image.Point) int {
	return slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == "Scenery" && r.Cell == cell })
}

// A map object under the pointer, a Scenery record, drags and deletes as a
// structure does: a click selects it and marks its cell, a drag moves it to
// the cell under the pointer as one host Move and leaves it selected there,
// and Delete removes it as one host Delete. A unit on the object's cell keeps
// the press, a drag from bare ground still draws a selection box, the box
// takes no object, a drag inside the box from an object still moves the box,
// and Ctrl+C on a selected object copies nothing.
func TestMapObjectDragsAndDeletesAsAStructure(t *testing.T) {
	e, h, _ := terrainEditor(t)
	withObjects(t, e, h, map[image.Point]uint8{{14, 14}: 5, {10, 10}: 6, {12, 12}: 7})
	e.ShowCell(image.Pt(14, 14))
	tree := image.Pt(14, 14)
	index := tree.Y*terrainSide + tree.X

	boxDrag(e, tree, tree)
	if len(h.calls.lines) != 0 || e.selected != sceneryIndex(e, tree) || e.selected < 0 {
		t.Fatalf("a click on the object made %q and selected %d", h.calls.lines, e.selected)
	}
	marks := e.markerLines()
	for _, l := range e.cellBoxLines(image.Rect(14, 14, 15, 15), editorGold) {
		if !slices.Contains(marks, l) {
			t.Fatal("the selected object's cell is not outlined as a selected structure's")
		}
	}
	_ = e.Key("copy")
	if len(h.calls.lines) != 0 || e.selection.clip != nil {
		t.Fatalf("Ctrl+C on a selected object copied: %q", h.calls.lines)
	}

	boxDrag(e, tree, image.Pt(17, 15))
	if got := lastCall(h.calls); got != fmt.Sprintf("Move Scenery %d %v", index, image.Pt(17, 15)) {
		t.Fatalf("a drag from the object called %q", got)
	}
	if e.selected < 0 || e.doc.Records[e.selected].Kind != "Scenery" || e.doc.Records[e.selected].Cell != image.Pt(17, 15) {
		t.Fatalf("after the move record %d is selected", e.selected)
	}
	if err := e.Key("delete"); err != nil {
		t.Fatal(err)
	}
	if got := lastCall(h.calls); got != fmt.Sprintf("Delete Scenery %d", 15*terrainSide+17) {
		t.Fatalf("Delete on the selected object called %q (%q)", got, e.message)
	}
	if sceneryIndex(e, image.Pt(17, 15)) >= 0 || e.selected >= 0 {
		t.Fatalf("the deleted object stays, selected %d", e.selected)
	}

	// The unit standing on an object keeps the press.
	calls := len(h.calls.lines)
	boxDrag(e, image.Pt(10, 10), image.Pt(11, 10))
	if got := lastCall(h.calls); len(h.calls.lines) != calls+1 || !strings.HasPrefix(got, "Move Unit 0 ") {
		t.Fatalf("a drag from the unit on an object called %q", h.calls.lines[calls:])
	}

	// Bare ground still draws a selection box, which takes no object, and a
	// drag inside it from an object still moves the box.
	calls = len(h.calls.lines)
	boxDrag(e, image.Pt(11, 11), image.Pt(13, 13))
	if len(h.calls.lines) != calls || e.selection.box != image.Rect(11, 11, 14, 14) || len(e.selection.refs) != 0 {
		t.Fatalf("a press on bare ground made %q, the box %v holding %v", h.calls.lines[calls:], e.selection.box, e.selection.refs)
	}
	boxDrag(e, image.Pt(12, 12), image.Pt(13, 12))
	if got := lastCall(h.calls); got != fmt.Sprintf("MoveArea %v %v %v", []RecordRef(nil), image.Rect(11, 11, 14, 14), image.Pt(1, 0)) {
		t.Fatalf("a drag inside the box from its object called %q", got)
	}
}
