package mapedit_test

import (
	"testing"

	"againrom/pkg/mapedit"
)

// window is the 5x5 block of the height plane centred on (12,12), rows top to
// bottom.
func window(alts []byte) [5][5]byte {
	var out [5][5]byte
	for y := range out {
		for x := range out[y] {
			out[y][x] = alts[(10+y)*brushSide+10+x]
		}
	}
	return out
}

// A set-height stroke blends each cell toward the target by
// h + (target - h) * cos(d * 3.1415926 / 2R), truncated, in the ring the
// distance falls in; the centre takes the target. Expected values are worked
// by hand: on a plane of 20 with target 60 and R 2, a cell at distance 1 is
// 20 + 40*cos(pi/4) = 48.28 -> 48, at sqrt 2 it is 20 + 40*0.4440 = 37.76 ->
// 37, at 2 the cosine is nil -> 20.
func TestSetHeightBlendsByTheRingRule(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 20))
	changed, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 60, Radius: 2, Set: true}, []mapedit.Cell{{X: 12, Y: 12}}, nil)
	if err != nil || !changed {
		t.Fatalf("Paint = %v, %v", changed, err)
	}
	_, alts, _ := planes(t, ed.Bytes())
	want := [5][5]byte{
		{20, 20, 20, 20, 20},
		{20, 37, 48, 37, 20},
		{20, 48, 60, 48, 20},
		{20, 37, 48, 37, 20},
		{20, 20, 20, 20, 20},
	}
	if got := window(alts); got != want {
		t.Fatalf("heights %v, want %v", got, want)
	}
	if !ed.Undo() || ed.CanUndo() {
		t.Fatal("one Undo does not restore the heights")
	}
}

// The step limit applies after the blend, to the cells with |dx| < |dy| only:
// on a plane of 0 with target 127 and R 2 the blend gives 89 beside the centre
// and above and below it and 56 on the diagonals; the cell above the centre
// rises to 127-31 = 96 and the one above that to 96-31 = 65, the centre keeps
// 127, and below it the step 89 to 0 is outside the disc.
func TestSetHeightBoundsTheRowStep(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 0))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 127, Radius: 2, Set: true}, []mapedit.Cell{{X: 12, Y: 12}}, nil); err != nil {
		t.Fatal(err)
	}
	_, alts, _ := planes(t, ed.Bytes())
	want := [5][5]byte{
		{0, 0, 65, 0, 0},
		{0, 56, 96, 56, 0},
		{0, 89, 127, 89, 0},
		{0, 56, 89, 56, 0},
		{0, 0, 0, 0, 0},
	}
	if got := window(alts); got != want {
		t.Fatalf("heights %v, want %v", got, want)
	}
}

// A target outside 0..127 clamps, and radius 0 acts as radius 1.
func TestSetHeightClampsAndHasRadiusOne(t *testing.T) {
	ed, _ := mapedit.New(brushMap(0x14, 20))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 300, Radius: 0, Set: true}, []mapedit.Cell{{X: 12, Y: 12}}, nil); err != nil {
		t.Fatal(err)
	}
	_, alts, _ := planes(t, ed.Bytes())
	if alts[12*brushSide+12] != 127 {
		t.Fatalf("centre %d", alts[12*brushSide+12])
	}
	// At radius 1 a neighbour is at the rim: cos(pi/2) is nil, so it keeps its height.
	if alts[12*brushSide+13] != 20 {
		t.Fatalf("neighbour %d, want 20", alts[12*brushSide+13])
	}
	ed, _ = mapedit.New(brushMap(0x14, 20))
	if _, err := ed.Paint(mapedit.Brush{Plane: mapedit.PlaneHeight, Value: -5, Set: true}, []mapedit.Cell{{X: 12, Y: 12}}, nil); err != nil {
		t.Fatal(err)
	}
	_, alts, _ = planes(t, ed.Bytes())
	if alts[12*brushSide+12] != 0 {
		t.Fatalf("centre %d, want 0", alts[12*brushSide+12])
	}
}
