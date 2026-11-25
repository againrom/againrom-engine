package ui

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"
)

// A document that carries cell values fills each cell by how many move kinds
// its value closes and names the value in the hover, a value no row names
// raw; its brush rows replace block and unblock, which a document without
// rows keeps.
func TestPassValuesFillAndNameTheMoveKinds(t *testing.T) {
	e := newMapEditorFixture(t)
	e.brush.mode = brushPassability
	if got := e.brushRows(); !slices.Equal(got, []string{e.word("editor.brush.block"), e.word("editor.brush.unblock")}) {
		t.Fatalf("rows of a document without brush rows %q", got)
	}
	n := e.doc.Width * e.doc.Height
	e.doc.Passability, e.doc.PassValues = make([]PassArm, n), make([]uint8, n)
	e.doc.PassArms = []PassArmRow{{Arm: 1, Word: "editor.pass.border", Border: true}, {Arm: 2, Word: "editor.mark.type3"}}
	e.doc.PassClosed = "editor.pass.marked"
	e.doc.PassValueRows = []PassValueRow{{Value: 0, Word: "editor.pass.value.0"}, {Value: 1, Kinds: 1, Word: "editor.pass.value.1"},
		{Value: 5, Kinds: 3, Word: "editor.pass.value.5"}, {Value: 0x1f, Kinds: 7, Word: "editor.pass.value.31"}}
	e.doc.PassBrush = []PassBrushRow{{Value: 1, Word: "editor.brush.secondclose"}, {Value: 0, Word: "editor.brush.secondopen"}}
	if got := e.brushRows(); !slices.Equal(got, []string{e.word("editor.brush.secondclose"), e.word("editor.brush.secondopen")}) {
		t.Fatalf("host rows %q", got)
	}
	cells := map[image.Point]uint8{{X: 10, Y: 10}: 1, {X: 11, Y: 10}: 5, {X: 12, Y: 10}: 0x1f, {X: 13, Y: 10}: 9}
	for c, v := range cells {
		e.doc.PassValues[c.Y*e.doc.Width+c.X] = v
		e.doc.Passability[c.Y*e.doc.Width+c.X] = 2
	}
	e.view.passability = true
	fills := map[color.RGBA]bool{}
	for c := range cells {
		fills[e.passCellAt(c).fill] = true
	}
	if fill := e.passCellAt(image.Pt(20, 20)).fill; fill != passOpenFill || len(fills) != 4 {
		t.Fatalf("open fill %v, %d distinct fills for the four values", fill, len(fills))
	}
	e.hoverOK = true
	for c, want := range map[image.Point]string{{X: 11, Y: 10}: e.word("editor.pass.value.5"), {X: 13, Y: 10}: "9", {X: 20, Y: 20}: e.word("editor.pass.value.0")} {
		e.hoverCell = c
		if s, ok := e.hoverPassability(); !ok || !strings.Contains(s, want) {
			t.Fatalf("hover of %v = %q, want %q", c, s, want)
		}
	}
}
