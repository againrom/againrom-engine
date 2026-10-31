package game

import (
	"bytes"
	"image"
	"testing"
	"time"

	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// objectDragEditor opens a new 48 x 48 map holding the object bytes objects,
// keyed by cell, in an editor window over the host, centred on cell (24, 24).
func objectDragEditor(t *testing.T, objects map[image.Point]uint8) (*ui.MapEditor, *mapInspectionEdits) {
	t.Helper()
	x := &MapInspector{tiles: &terrain.Tileset{}}
	doc, err := x.Open("new/48x48")
	if err != nil {
		t.Fatal(err)
	}
	s := doc.Edits.(*mapInspectionEdits)
	if _, err := s.RandomHeights(5); err != nil {
		t.Fatal(err)
	}
	doc, err = s.change(func(m *mapedit.Editor) (bool, error) {
		for c, b := range objects {
			if err := m.SetCell(mapedit.GridRaw, mapedit.GridOverlay, c.X, c.Y, uint16(b)); err != nil {
				return false, err
			}
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	e := ui.NewMapEditor(nil, nil, func(string) (*ui.InspectionDocument, error) { return doc, nil })
	if err := e.Open("new/48x48"); err != nil {
		t.Fatal(err)
	}
	e.Layout(1280, 800)
	e.ShowCell(image.Pt(24, 24))
	return e, s
}

// pointerDrag presses on cell a, moves to cell b and releases there.
func pointerDrag(e *ui.MapEditor, a, b image.Point) {
	now := time.Unix(1, 0)
	pa, pb := e.CellPoint(a), e.CellPoint(b)
	e.Pointer(ui.Input{CursorX: pa.X, CursorY: pa.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: pb.X, CursorY: pb.Y, PrimaryDown: true}, now)
	e.Pointer(ui.Input{CursorX: pb.X, CursorY: pb.Y}, now)
}

// objectPlane is the host map's object bytes.
func objectPlane(t *testing.T, s *mapInspectionEdits) []uint8 {
	t.Helper()
	m, err := s.model.Map()
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Clone(m.Overlay)
}

// One map object with no record on its cell drags and deletes as a structure
// does, through the editor's own pointer and Delete key: a drag moves its byte
// to the cell under the pointer as one edit, replacing the object there and
// leaving every other byte of the object plane as it stood, and leaves it
// selected there; Delete clears it as one edit. One undo gives back the map's
// bytes each time.
func TestEditorDragsAndDeletesOneMapObject(t *testing.T) {
	const w = 48
	for _, code := range []uint8{1, 7, 12, 255} {
		tree, stone := image.Pt(21, 22), image.Pt(25, 24)
		e, s := objectDragEditor(t, map[image.Point]uint8{tree: code, stone: 3})
		before, plane := s.model.Bytes(), objectPlane(t, s)
		pointerDrag(e, tree, stone)
		got := objectPlane(t, s)
		want := bytes.Clone(plane)
		want[tree.Y*w+tree.X], want[stone.Y*w+stone.X] = 0, code
		if !bytes.Equal(got, want) {
			t.Fatalf("object %d: the drag left the object plane %v at the two cells, want %v", code,
				[]uint8{got[tree.Y*w+tree.X], got[stone.Y*w+stone.X]}, []uint8{0, code})
		}
		if len(s.model.Bytes()) != len(before) {
			t.Fatalf("object %d: the drag resized the map", code)
		}
		if i := e.Selected(); i < 0 || e.Document().Records[i].Kind != "Scenery" || e.Document().Records[i].Cell != stone {
			t.Fatalf("object %d: after the drag record %d is selected", code, i)
		}
		if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
			t.Fatalf("object %d: one undo after the drag did not give back the map (%v)", code, err)
		}
		// A click selects the object back on its cell; Delete clears it.
		pointerDrag(e, tree, tree)
		if err := e.Key("delete"); err != nil {
			t.Fatal(err)
		}
		want = bytes.Clone(plane)
		want[tree.Y*w+tree.X] = 0
		if got := objectPlane(t, s); !bytes.Equal(got, want) {
			t.Fatalf("object %d: Delete left %d on its cell (%q)", code, got[tree.Y*w+tree.X], e.Message())
		}
		if err := e.Key("undo"); err != nil || !bytes.Equal(s.model.Bytes(), before) {
			t.Fatalf("object %d: one undo after Delete did not give back the map (%v)", code, err)
		}
	}
}

// The host's map object move refuses a target off the map and changes
// nothing, a drag check of that target refuses it too, and a delete of a cell
// with no object is no edit.
func TestHostObjectMoveRefusesOffTheMap(t *testing.T) {
	_, s := objectDragEditor(t, map[image.Point]uint8{{0, 3}: 4})
	before := s.model.Bytes()
	for _, cell := range []image.Point{{-1, 3}, {48, 3}, {0, -1}, {0, 48}} {
		if _, err := s.Move("Scenery", 3*48, cell); err == nil {
			t.Fatalf("a move onto %v was taken", cell)
		}
		if s.CheckObjectCell("Scenery", 3*48, cell) == nil {
			t.Fatalf("the drag check took %v", cell)
		}
	}
	if err := s.CheckObjectCell("Scenery", 3*48, image.Pt(5, 5)); err != nil {
		t.Fatalf("the drag check refused a cell on the map: %v", err)
	}
	if d, err := s.Delete("Scenery", 3*48+2); err != nil || d != nil {
		t.Fatalf("a delete of a bare cell made %v (%v)", d, err)
	}
	if _, err := s.Move("Scenery", 48*48, image.Pt(1, 1)); err == nil {
		t.Fatal("a Scenery index off the map was moved")
	}
	if !bytes.Equal(s.model.Bytes(), before) {
		t.Fatal("a refused move, a drag check or an empty delete changed the map")
	}
}
