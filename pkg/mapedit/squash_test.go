package mapedit

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/alm"
)

func TestSquashFoldsTheEditsSinceAMarkIntoOneUndo(t *testing.T) {
	e, err := NewMap(alm.NewMapSpec{Width: 32, Height: 32, Tile: 0x11, Altitude: 64})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Paint(Brush{Plane: PlaneObject, Value: 3}, []Cell{{1, 1}}, nil); err != nil {
		t.Fatal(err)
	}
	first := e.Bytes()
	mark := e.Mark()
	for _, c := range []Cell{{5, 5}, {9, 9}} {
		if _, err := e.Paint(Brush{Plane: PlaneObject, Value: 7}, []Cell{c}, nil); err != nil {
			t.Fatal(err)
		}
	}
	after := e.Bytes()
	e.Squash(mark)
	if !bytes.Equal(e.Bytes(), after) || !e.CanUndo() || e.CanRedo() {
		t.Fatal("Squash changed the bytes or the history's ends")
	}
	if !e.Undo() || !bytes.Equal(e.Bytes(), first) {
		t.Fatal("one Undo did not revert both folded edits")
	}
	if !e.Redo() || !bytes.Equal(e.Bytes(), after) {
		t.Fatal("one Redo did not re-apply both folded edits")
	}
	if !e.Undo() || !e.Undo() || e.CanUndo() {
		t.Fatal("the edit before the mark is not its own entry")
	}
	one := e.Mark()
	e.Redo()
	e.Squash(one) // a single edit since the mark stays as it is
	if !e.CanUndo() || !e.CanRedo() {
		t.Fatal("Squash of one edit changed the history")
	}
}
