package databin_test

import (
	"math"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/databin"
)

// The nine doubles come back at their own slots, in order, and slot j is the
// j'th eight bytes of the record — the whole of the decode.
func TestTheNineDoublesComeBackAtTheirOwnSlots(t *testing.T) {
	want := []float64{0.2, 1.5, 250, 0.75, 1, -3.25, 0, 4.5, 12.5}
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes: {{Name: "Common", Doubles: want}},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	got := f.Collection(databin.Shapes).EntryDoubles(0)
	if len(got) != len(want) {
		t.Fatalf("EntryDoubles gave %d value(s), want %d", len(got), len(want))
	}
	for j := range want {
		if got[j] != want[j] {
			t.Errorf("slot %d = %v, want %v", j, got[j], want[j])
		}
	}
}

// A short row leaves the rest of the record zero rather than shifting anything:
// the writer places what it is given and the reader still reads nine.
func TestAShortDoubleRowLeavesTheRestZero(t *testing.T) {
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinMaterials: {{Name: "Iron", Doubles: []float64{7, 8}}},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	got := f.Collection(databin.Materials).EntryDoubles(0)
	if len(got) != 9 || got[0] != 7 || got[1] != 8 {
		t.Fatalf("EntryDoubles = %v, want [7 8 0 0 0 0 0 0 0]", got)
	}
	for j := 2; j < 9; j++ {
		if got[j] != 0 {
			t.Errorf("slot %d = %v, want 0", j, got[j])
		}
	}
}

// An entry of a collection whose grammar writes no double record has none —
// nil, not nine zeroes, so a caller cannot mistake "no record" for "a record of
// zeroes".
func TestACollectionWithNoDoubleRecordHasNone(t *testing.T) {
	f, err := databin.Parse(synth.DataBinUnitsTable(
		[]synth.DataBinRow{{Name: "Goblin", Params: []int32{1, 2, 3}}}, nil))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := f.Collection(databin.Units).EntryDoubles(1); got != nil {
		t.Fatalf("EntryDoubles = %v, want nil for a row that carries no such record", got)
	}
}

// The bytes are handed over as a COPY: writing what came back cannot reach the
// parsed file.
func TestEntryDoublesIsACopy(t *testing.T) {
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes: {{Name: "Common", Doubles: []float64{0.2}}},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := f.Collection(databin.Shapes)
	c.EntryDoubles(0)[0] = math.Inf(1)
	if got := c.EntryDoubles(0)[0]; got != 0.2 {
		t.Fatalf("slot 0 = %v after a caller wrote its copy, want 0.2", got)
	}
}
