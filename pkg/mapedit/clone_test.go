package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/internal/synth"
)

func TestClonePreservesAndIsolatesUndoRedoBranch(t *testing.T) {
	b := synth.ALM(synth.ALMOptions{Width: 32, Height: 32, Units: []synth.ALMUnit{{X: 512, Y: 768}}})
	e := newEditor(t, b)
	if err := e.MoveUnit(0, 1024, 1536); err != nil {
		t.Fatal(err)
	}
	first := e.Bytes()
	if err := e.MoveUnit(0, 2048, 2304); err != nil {
		t.Fatal(err)
	}
	second := e.Bytes()
	e.Undo()
	c := e.Clone()
	if !c.CanUndo() || !c.CanRedo() || !bytes.Equal(first, c.Bytes()) {
		t.Fatal("clone lost history position")
	}
	if err := c.MoveUnit(0, 2560, 2816); err != nil {
		t.Fatal(err)
	}
	if !e.Redo() || !bytes.Equal(second, e.Bytes()) || c.CanRedo() {
		t.Fatal("clone branch overwrote original redo")
	}
	c.Undo()
	c.Undo()
	if !bytes.Equal(b, c.Bytes()) || !bytes.Equal(second, e.Bytes()) {
		t.Fatal("clone shared mutable bytes/history")
	}
}
