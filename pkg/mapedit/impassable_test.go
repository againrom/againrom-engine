package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/pkg/mapedit"
)

// A Block stroke sets bit 13 of the tile word of every disc cell and an
// Unblock stroke clears it; neither changes any other bit, cell or byte, and
// each is one undo step. The control is the cell outside the disc: it keeps
// the flag it had.
func TestImpassableStrokeChangesOnlyBitThirteenOfTheDisc(t *testing.T) {
	in := brushMap(0x211, 40) // water
	outside := 3*brushSide + 3
	tf := walkFrame(t, in).byType(t, 1)
	in[tf.payloadOff+2*outside+1] |= 0x20 // the flag on a cell no stroke reaches
	ed, err := mapedit.New(in)
	if err != nil {
		t.Fatal(err)
	}
	inDisc := func(i, cx, cy, r int) bool {
		x, y := i%brushSide, i/brushSide
		return (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r
	}
	if changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 1, Radius: 2}, []mapedit.Cell{{X: 12, Y: 12}, {X: 14, Y: 12}}, nil); err != nil || !changed {
		t.Fatalf("Block = %v, %v", changed, err)
	}
	blocked := ed.Bytes()
	if len(blocked) != len(in) {
		t.Fatal("the stroke changed the file's length")
	}
	for i := range in {
		if d := in[i] ^ blocked[i]; d != 0 && (d != 0x20 || i < tf.payloadOff || i >= tf.payloadOff+tf.payloadSize || (i-tf.payloadOff)%2 != 1) {
			t.Fatalf("byte %#x changed by %#x", i, d)
		}
	}
	tiles, _, _ := planes(t, blocked)
	for i, w := range tiles {
		want := uint16(0x211)
		if inDisc(i, 12, 12, 2) || inDisc(i, 14, 12, 2) || i == outside {
			want |= 0x2000
		}
		if w != want {
			t.Fatalf("cell %d,%d = %#x, want %#x", i%brushSide, i/brushSide, w, want)
		}
	}
	if changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 0, Radius: 1}, []mapedit.Cell{{X: 12, Y: 12}}, nil); err != nil || !changed {
		t.Fatalf("Unblock = %v, %v", changed, err)
	}
	tiles, _, _ = planes(t, ed.Bytes())
	for i, w := range tiles {
		flagged := (inDisc(i, 12, 12, 2) || inDisc(i, 14, 12, 2) || i == outside) && !inDisc(i, 12, 12, 1)
		if w&^0x2000 != 0x211 || (w&0x2000 != 0) != flagged {
			t.Fatalf("after Unblock cell %d,%d = %#x", i%brushSide, i/brushSide, w)
		}
	}
	if !ed.Undo() || !bytes.Equal(ed.Bytes(), blocked) || !ed.Undo() || !bytes.Equal(ed.Bytes(), in) || ed.CanUndo() {
		t.Fatal("two Undos do not restore the two strokes one by one")
	}
	if changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 0, Radius: 2}, []mapedit.Cell{{X: 20, Y: 20}}, nil); err != nil || changed || ed.CanUndo() {
		t.Fatalf("an Unblock over clear cells: changed=%v err=%v", changed, err)
	}
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 2}, []mapedit.Cell{{X: 1, Y: 1}}, nil); err == nil || !bytes.Equal(ed.Bytes(), in) {
		t.Fatal("a flag value other than 0 and 1 was accepted")
	}
}
