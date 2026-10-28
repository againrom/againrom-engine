package mapedit_test

import (
	"image"
	"testing"

	"againrom/pkg/mapedit"
)

// A flat set-height stroke gives every cell its brush covers along all its
// centres the one height, in the brush's own shape and radius, with no
// blend, and leaves every other cell; one Undo restores the plane. A radius
// of 0 covers the centres alone, where a blending set would widen to 1.
func TestFlatSetHeightLevelsTheWholePath(t *testing.T) {
	cases := []struct {
		shape  mapedit.BrushShape
		radius int
		cover  image.Rectangle
	}{
		{mapedit.BrushSquare, 1, image.Rect(9, 11, 16, 14)},
		{mapedit.BrushSquare, 0, image.Rect(10, 12, 15, 13)},
	}
	for _, c := range cases {
		ed, _ := mapedit.New(brushMap(0x14, 20))
		path := []mapedit.Cell{{X: 10, Y: 12}, {X: 11, Y: 12}, {X: 12, Y: 12}, {X: 13, Y: 12}, {X: 14, Y: 12}}
		b := mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 40, Radius: c.radius, Shape: c.shape, Falloff: mapedit.FalloffCosine, Set: true, Flat: true}
		changed, err := ed.Paint(b, path, nil)
		if err != nil || !changed {
			t.Fatalf("Paint = %v, %v", changed, err)
		}
		_, alts, _ := planes(t, ed.Bytes())
		for i, h := range alts {
			want := byte(20)
			if image.Pt(i%brushSide, i/brushSide).In(c.cover) {
				want = 40
			}
			if h != want {
				t.Fatalf("radius %d: cell (%d,%d) is %d, want %d", c.radius, i%brushSide, i/brushSide, h, want)
			}
		}
		if got := len(b.Footprint(path, brushSide, brushSide)); got != c.cover.Dx()*c.cover.Dy() {
			t.Errorf("radius %d: the footprint has %d cells, the stroke wrote %d", c.radius, got, c.cover.Dx()*c.cover.Dy())
		}
		if !ed.Undo() || ed.CanUndo() {
			t.Fatal("one Undo does not restore the heights")
		}
	}
}
