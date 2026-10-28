package ui

import (
	"fmt"
	"image"
	"slices"
	"testing"
	"time"
)

func (p *terrainHost) MoveArea(refs []RecordRef, box image.Rectangle, offset image.Point) (SelectionEdit, error) {
	d, err := p.next("MoveArea %v %v %v", refs, box, offset)
	return SelectionEdit{Doc: d, Selected: refs, Note: "area moved"}, err
}

// A drag that starts inside the selection box, on bare ground or on a record
// the group holds, moves the group with the objects under the box as one
// host edit, and the box follows; the box cannot leave the map. A click
// inside the box and a press outside it keep their old meaning.
func TestBoxDragMovesTheBoxWithItsRecords(t *testing.T) {
	e, h, _ := terrainEditor(t)
	e.ShowCell(image.Pt(10, 10))
	boxDrag(e, image.Pt(8, 8), image.Pt(12, 11))
	if e.selection.box != image.Rect(8, 8, 13, 12) || !slices.Equal(e.selection.refs, []RecordRef{{"Unit", 0}}) {
		t.Fatalf("the box is %v holding %v", e.selection.box, e.selection.refs)
	}
	before := e.marqueeLines()
	now := time.Unix(1, 0)
	pa, pb := e.CellPoint(image.Pt(9, 9)), e.CellPoint(image.Pt(15, 13))
	e.Pointer(Input{CursorX: pa.X, CursorY: pa.Y, PrimaryDown: true}, now)
	e.Pointer(Input{CursorX: pb.X, CursorY: pb.Y, PrimaryDown: true}, now)
	if during := e.marqueeLines(); len(during) != len(before) || during[0] == before[0] {
		t.Error("the box outline does not follow the drag")
	}
	e.Pointer(Input{CursorX: pb.X, CursorY: pb.Y}, now)
	want := fmt.Sprintf("MoveArea %v %v %v", []RecordRef{{"Unit", 0}}, image.Rect(8, 8, 13, 12), image.Pt(6, 4))
	if got := lastCall(h.calls); got != want {
		t.Fatalf("a drag inside the box called %q, want %q", got, want)
	}
	if e.selection.box != image.Rect(14, 12, 19, 16) {
		t.Fatalf("the box after the move is %v", e.selection.box)
	}
	// A drag from the group's record inside the box moves the box too.
	e, h, _ = terrainEditor(t)
	e.ShowCell(image.Pt(10, 10))
	boxDrag(e, image.Pt(8, 8), image.Pt(12, 11))
	boxDrag(e, image.Pt(10, 10), image.Pt(11, 10))
	if got := lastCall(h.calls); got != fmt.Sprintf("MoveArea %v %v %v", []RecordRef{{"Unit", 0}}, image.Rect(8, 8, 13, 12), image.Pt(1, 0)) {
		t.Fatalf("a drag from the box's record called %q", got)
	}
	// The offset stops where the box meets the map's edge.
	e, h, _ = terrainEditor(t)
	e.ShowCell(image.Pt(44, 44))
	boxDrag(e, image.Pt(40, 40), image.Pt(44, 44))
	boxDrag(e, image.Pt(41, 41), image.Pt(48, 47))
	if got := lastCall(h.calls); got != fmt.Sprintf("MoveArea %v %v %v", []RecordRef(nil), image.Rect(40, 40, 45, 45), image.Pt(5, 5)) {
		t.Fatalf("a drag past the edge called %q", got)
	}
	// Controls: a click inside the box moves nothing; a press outside it
	// draws a new box.
	e, h, _ = terrainEditor(t)
	e.ShowCell(image.Pt(10, 10))
	boxDrag(e, image.Pt(8, 8), image.Pt(12, 11))
	boxDrag(e, image.Pt(9, 9), image.Pt(9, 9))
	boxDrag(e, image.Pt(14, 14), image.Pt(16, 16))
	for _, c := range h.calls.lines {
		if len(c) >= 8 && c[:8] == "MoveArea" {
			t.Fatalf("a click or an outside drag moved the area: %q", h.calls.lines)
		}
	}
	if e.selection.box != image.Rect(14, 14, 17, 17) {
		t.Fatalf("a press outside the box left the box %v", e.selection.box)
	}
}
