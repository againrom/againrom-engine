package ui

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"testing"
	"time"
)

// ghostHost previews every placement with a footprint: an Add's new record
// takes two cells square, a paste places a unit two cells square on the cell
// and refuses a 3x2 structure four cells to the east. refuse refuses every
// placement and move.
type ghostHost struct {
	*modesFull
	refuse bool
	base   *InspectionDocument
}

func (h *ghostHost) PreviewPlacement(r InspectionNewRecord) (*InspectionDocument, error) {
	d := *h.base
	d.Records = slices.Clone(h.base.Records)
	square := image.Rectangle{Min: r.Cell, Max: r.Cell.Add(image.Pt(2, 2))}
	if r.Clip != nil {
		d.Records = append(d.Records, InspectionRecord{Kind: "Unit", Index: 2, Cell: r.Cell, Spatial: true, Size: image.Pt(1, 1), Footprint: square})
		east := r.Cell.Add(image.Pt(4, 0))
		d.Refused = []SelectionRefusal{{Ref: RecordRef{Kind: "Structure", Index: 0}, Footprint: image.Rectangle{Min: east, Max: east.Add(image.Pt(3, 2))}}}
		return &d, nil
	}
	d.Records = append(d.Records, InspectionRecord{Kind: r.Kind, Index: 2, Cell: r.Cell, Spatial: true, Size: image.Pt(1, 1), Footprint: square})
	if h.refuse {
		return &d, fmt.Errorf("footprint overlaps another object")
	}
	return &d, nil
}

func (h *ghostHost) CheckObjectCell(string, int, image.Point) error {
	if h.refuse {
		return fmt.Errorf("refused")
	}
	return nil
}

func (h *ghostHost) CheckMove(refs []RecordRef, _ image.Rectangle, _ image.Point) []RecordRef {
	if h.refuse {
		return refs
	}
	return nil
}

type ghostClip struct{}

func (ghostClip) Len() int           { return 1 }
func (ghostClip) Patch() image.Point { return image.Point{} }

func newGhostEditor(t *testing.T) (*MapEditor, *ghostHost) {
	t.Helper()
	e, _ := newModesEditor(t, "full")
	h := &ghostHost{modesFull: e.doc.Edits.(*modesFull), base: e.doc}
	h.self, e.doc.Edits = h, h
	// The hall is 2x2 by its art and takes 3x2 cells in the world.
	e.doc.Records[2].Footprint = image.Rect(30, 30, 33, 32)
	e.Fit()
	return e, h
}

// ghostLines is how many more lines in c the map draws with the placement
// armed than without it.
func ghostLines(e *MapEditor, c color.RGBA) int {
	count := func() int {
		n := 0
		for _, l := range e.markerLines() {
			if l.color == c {
				n++
			}
		}
		return n
	}
	armed := count()
	pick, drag := e.pick, e.recordDrag
	e.pick, e.recordDrag = pickNone, nil
	bare := count()
	e.pick, e.recordDrag = pick, drag
	return armed - bare
}

func hoverGhost(e *MapEditor, cell image.Point) {
	p := e.CellPoint(cell)
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Time{})
	e.hoverCell, e.hoverOK = cell, true
}

func TestEditorAddGhostMarksTheWholeFootprint(t *testing.T) {
	e, h := newGhostEditor(t)
	e.openAdd("Unit")
	hoverGhost(e, image.Pt(25, 20))
	f, _ := e.placement()
	if len(f.Cells) != 1 || f.Cells[0] != (placementCells{Rect: image.Rect(25, 20, 27, 22)}) {
		t.Fatalf("Add ghost cells %+v", f.Cells)
	}
	if n := ghostLines(e, editorGold); n != 4*4 {
		t.Fatalf("Add ghost draws %d gold lines, want four cells", n)
	}
	h.refuse, e.placementKey = true, ""
	f, _ = e.placement()
	if !f.Refused || len(f.Sprites) != 0 || len(f.Cells) != 1 || !f.Cells[0].Refused {
		t.Fatalf("refused Add ghost %+v", f)
	}
	if n := ghostLines(e, editorWarning); n != 4*4 {
		t.Fatalf("refused Add ghost draws %d warning lines, want four cells", n)
	}
	// The footprints switch off leaves the anchor cell.
	e.view.hideFootprints = true
	if n := ghostLines(e, editorWarning); n != 4 {
		t.Fatalf("Add ghost with footprints off draws %d warning lines, want one cell", n)
	}
}

func TestEditorMovePickGhostMarksTheWholeFootprint(t *testing.T) {
	e, h := newGhostEditor(t)
	e.selected = 2
	e.armPick(pickMove)
	hoverGhost(e, image.Pt(10, 20))
	if n := ghostLines(e, editorGold); n != 6*4 {
		t.Fatalf("move pick of the hall draws %d gold lines, want its six cells", n)
	}
	h.refuse = true
	if n := ghostLines(e, editorWarning); n != 6*4 {
		t.Fatalf("refused move pick of the hall draws %d warning lines, want its six cells", n)
	}
	// Control: a one-cell unit marks one cell.
	h.refuse = false
	e.selected = 0
	if n := ghostLines(e, editorGold); n != 4 {
		t.Fatalf("move pick of a one-cell unit draws %d gold lines, want one cell", n)
	}
}

func TestEditorPasteGhostMarksEachRecordAndClipsTheMapEdge(t *testing.T) {
	e, _ := newGhostEditor(t)
	e.selection.clip = ghostClip{}
	e.armPick(pickPaste)
	hoverGhost(e, image.Pt(20, 20))
	if n := ghostLines(e, editorGold); n != 4*4 {
		t.Fatalf("paste ghost draws %d gold lines, want the unit's four cells", n)
	}
	if n := ghostLines(e, editorWarning); n != 6*4 {
		t.Fatalf("paste ghost draws %d warning lines, want the refused structure's six cells", n)
	}
	// At 45,20 the refused footprint starts at column 49, the map's last.
	hoverGhost(e, image.Pt(45, 20))
	if n := ghostLines(e, editorWarning); n != 2*4 {
		t.Fatalf("paste ghost at the edge draws %d warning lines, want two cells", n)
	}
}

func TestEditorRecordDragGhostColoursTheRefusedFootprints(t *testing.T) {
	e, h := newGhostEditor(t)
	e.selected = 2
	e.recordDrag = &editorRecordDrag{kind: "Structure", record: 2, index: 0, cell: image.Pt(35, 30), origin: image.Pt(30, 30), moved: true}
	if n := ghostLines(e, editorWarning); n != 0 {
		t.Fatalf("an accepted drag draws %d warning lines", n)
	}
	h.refuse, e.dragCheckKey = true, ""
	if n := ghostLines(e, editorWarning); n != 6*4 {
		t.Fatalf("a refused drag of the hall draws %d warning lines, want its six cells", n)
	}
	// A group drag asks CheckMove for the whole selection.
	e.setSelection([]RecordRef{{Kind: "Unit", Index: 0}, {Kind: "Unit", Index: 1}})
	e.recordDrag = &editorRecordDrag{kind: "Unit", record: 0, index: 0, cell: image.Pt(12, 10), origin: image.Pt(10, 10), moved: true, group: true}
	f, _ := e.placement()
	if len(f.RefusedRecords) != 2 {
		t.Fatalf("group drag refused %v", f.RefusedRecords)
	}
	if n := ghostLines(e, editorWarning); n != 2*4 {
		t.Fatalf("a refused group drag draws %d warning lines, want two cells", n)
	}
}
