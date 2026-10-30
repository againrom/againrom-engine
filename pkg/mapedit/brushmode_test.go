package mapedit_test

import (
	"bytes"
	"testing"

	"againrom/pkg/mapedit"
)

// unevenMap is brushMap with heights 30..42 that differ from cell to cell
// by less than the row step bound.
func unevenMap(t *testing.T) []byte {
	t.Helper()
	data := brushMap(0x14, 0)
	af := walkFrame(t, data).byType(t, 2)
	for i := 0; i < brushSide*brushSide; i++ {
		x, y := i%brushSide, i/brushSide
		data[af.payloadOff+i] = byte(30 + (7*x+3*y)%13)
	}
	return data
}

// modeDrag is a drag whose path passes cell (10,10) twice and covers most
// cells of its footprint from several centres.
var modeDrag = []mapedit.Cell{{X: 8, Y: 10}, {X: 9, Y: 10}, {X: 10, Y: 10}, {X: 11, Y: 10}, {X: 10, Y: 10}, {X: 9, Y: 10}, {X: 9, Y: 11}}

// paintModeStroke paints b along modeDrag on the uneven map as one edit and
// answers the heights before and after and the stroke's footprint; one Undo
// restores the heights and leaves nothing more to undo.
func paintModeStroke(t *testing.T, b mapedit.Brush) (before, after []byte, in map[int]bool) {
	t.Helper()
	ed, err := mapedit.New(unevenMap(t))
	if err != nil {
		t.Fatal(err)
	}
	_, alts, _ := planes(t, ed.Bytes())
	before = append([]byte(nil), alts...)
	if changed, err := ed.Paint(b, modeDrag, nil); err != nil || !changed {
		t.Fatalf("Paint = %v, %v", changed, err)
	}
	_, alts, _ = planes(t, ed.Bytes())
	after = append([]byte(nil), alts...)
	in = map[int]bool{}
	for _, c := range b.Footprint(modeDrag, brushSide, brushSide) {
		in[c.Y*brushSide+c.X] = true
	}
	if !ed.Undo() || ed.CanUndo() {
		t.Fatal("the stroke is not one edit")
	}
	if _, alts, _ = planes(t, ed.Bytes()); !bytes.Equal(alts, before) {
		t.Fatal("one Undo does not restore the heights")
	}
	return before, after, in
}

// A blending set stroke writes around its first centre alone, and its
// footprint is that centre's circle, so the marker and the report name the
// cells the stroke can write.
func TestBlendingSetFootprintIsTheFirstCentresCircle(t *testing.T) {
	b := mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 60, Radius: 2, Set: true}
	before, after, in := paintModeStroke(t, b)
	first := modeDrag[0]
	for i := range after {
		x, y := i%brushSide, i/brushSide
		circle := (x-first.X)*(x-first.X)+(y-first.Y)*(y-first.Y) <= 4
		if in[i] != circle {
			t.Fatalf("cell (%d,%d): footprint %v, first centre's circle %v", x, y, in[i], circle)
		}
		if after[i] != before[i] && !in[i] {
			t.Fatalf("cell (%d,%d) changed %d -> %d outside the footprint", x, y, before[i], after[i])
		}
	}
	if c := first.Y*brushSide + first.X; after[c] != 60 {
		t.Fatalf("first centre is %d, want 60", after[c])
	}
}

// A Step stroke (the add without fall-off, once per cell) raises or lowers
// every cell its footprint covers by exactly the strength, the cells the
// drag passes twice included, and leaves every other cell.
func TestStepStrokeAddsTheStrengthOncePerCell(t *testing.T) {
	for _, change := range []int{5, -5} {
		b := mapedit.Brush{Plane: mapedit.PlaneHeight, Value: change, Radius: 2, Falloff: mapedit.FalloffNone, Once: true}
		before, after, in := paintModeStroke(t, b)
		for i := range after {
			want := int(before[i])
			if in[i] {
				want += change
			}
			if int(after[i]) != want {
				t.Fatalf("change %d: cell (%d,%d) is %d, was %d, want %d", change, i%brushSide, i/brushSide, after[i], before[i], want)
			}
		}
	}
}

// A Level stroke (the flat set at the pressed cell's height plus the change)
// ends every cell its footprint covers at that one height, the cells above
// it brought down and those below it brought up, and leaves every other
// cell.
func TestLevelStrokeLevelsThePathAtThePressedCellPlusTheStrength(t *testing.T) {
	for _, change := range []int{3, -3} {
		b := mapedit.Brush{Plane: mapedit.PlaneHeight, Value: change, Radius: 2, Set: true, Flat: true, Relative: true}
		before, after, in := paintModeStroke(t, b)
		target := int(before[modeDrag[0].Y*brushSide+modeDrag[0].X]) + change
		above, below := 0, 0
		for i := range after {
			want := int(before[i])
			if in[i] {
				want = target
				above += map[bool]int{true: 1}[int(before[i]) > target]
				below += map[bool]int{true: 1}[int(before[i]) < target]
			}
			if int(after[i]) != want {
				t.Fatalf("change %d: cell (%d,%d) is %d, was %d, want %d", change, i%brushSide, i/brushSide, after[i], before[i], want)
			}
		}
		if above == 0 || below == 0 {
			t.Fatalf("change %d: the footprint holds %d cells above the target and %d below; the witness needs both", change, above, below)
		}
	}
}

// A Smooth stroke is the cosine add at every centre: the same drag differs
// from Step and from Level, and at strength 8 it equals an independent
// computation of the fall-off add.
func TestSmoothStrokeIsTheCosineAdd(t *testing.T) {
	smooth := mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 3, Radius: 2, Falloff: mapedit.FalloffCosine}
	_, s, _ := paintModeStroke(t, smooth)
	_, step, _ := paintModeStroke(t, mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 3, Radius: 2, Once: true})
	_, level, _ := paintModeStroke(t, mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 3, Radius: 2, Set: true, Flat: true, Relative: true})
	if bytes.Equal(s, step) || bytes.Equal(s, level) {
		t.Fatal("the Smooth stroke equals the Step or the Level stroke")
	}
	smooth.Value = 8
	before, got, _ := paintModeStroke(t, smooth)
	want := append([]byte(nil), before...)
	wave := [9]int{64, 63, 59, 53, 45, 36, 24, 12, 0}
	const r = 2
	for _, c := range modeDrag {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				d2 := dx*dx + dy*dy
				if d2 > r*r {
					continue
				}
				k := 0
				for (k+1)*(k+1) <= 64*d2/((r+1)*(r+1)) && k < 8 {
					k++
				}
				i := (c.Y+dy)*brushSide + c.X + dx
				want[i] = byte(min(max(int(want[i])+8*wave[k]/64, 0), 127))
			}
		}
	}
	if !bytes.Equal(got, want) {
		t.Fatal("the Smooth stroke at strength 8 differs from the cosine add")
	}
}
