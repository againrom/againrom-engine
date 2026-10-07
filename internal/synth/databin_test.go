package synth_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/databin"
)

// The two collections a placement can resolve against, and the width a units row
// has to reach before a definition can be built from it. They are spelled here
// rather than imported so this test says what it expects instead of asking the
// code under test.
const (
	unitsID  = databin.ID(synth.DataBinUnits)
	humansID = databin.ID(synth.DataBinHumans)
	rowWidth = 38
)

func row(name string, slots map[int]int32) synth.DataBinRow {
	p := make([]int32, rowWidth)
	for i := range p {
		p[i] = -1
	}
	for s, v := range slots {
		p[s] = v
	}
	return synth.DataBinRow{Name: name, Params: p}
}

// TestAWrittenTableParses is the whole point of the writer: a stream it produces
// is one the parser consumes exactly, with nothing left over.
func TestAWrittenTableParses(t *testing.T) {
	b := synth.DataBinUnitsTable(
		[]synth.DataBinRow{row("first", map[int]int32{4: 40}), row("second", map[int]int32{4: 99})},
		[]synth.DataBinRow{row("a-human", map[int]int32{0x10: 7})},
	)

	f, err := databin.Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.Consumed != len(b) {
		t.Errorf("the walk consumed %d of %d byte(s)", f.Consumed, len(b))
	}
}

// TestAOneBasedCollectionPutsTheFirstWrittenRowAtIndexOne pins the counting rule
// the writer has to get right: the count word is one larger than the rows, and
// entry 0 is present and empty.
func TestAOneBasedCollectionPutsTheFirstWrittenRowAtIndexOne(t *testing.T) {
	b := synth.DataBinUnitsTable([]synth.DataBinRow{row("first", nil), row("second", nil)}, nil)
	f, err := databin.Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	c := f.Collection(unitsID)
	if got := len(c.Entries); got != 3 {
		t.Fatalf("Units holds %d entries, want 3 — two written plus the reserved one", got)
	}
	if got := string(c.Entries[0].Name); got != "" {
		t.Errorf("entry 0 is named %q, want the reserved empty entry", got)
	}
	for i, want := range []string{"first", "second"} {
		if got := string(c.Entries[i+1].Name); got != want {
			t.Errorf("entry %d is named %q, want %q", i+1, got, want)
		}
	}
	if !c.OneBased {
		t.Errorf("Units did not read back as one-based")
	}
}

// TestEveryWrittenCellReadsBackAsWritten covers the cell the format uses for an
// empty column: -1 is 32 bits of ones on the wire and must not come back as
// anything else.
func TestEveryWrittenCellReadsBackAsWritten(t *testing.T) {
	want := row("cells", map[int]int32{0: 30, 4: 99, 32: 3, 37: -1})
	b := synth.DataBinUnitsTable([]synth.DataBinRow{want}, nil)
	f, err := databin.Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	got := f.Collection(unitsID).EntryParams(1)
	if len(got) != len(want.Params) {
		t.Fatalf("the row read back with %d cells, want %d", len(got), len(want.Params))
	}
	for i := range got {
		if got[i] != want.Params[i] {
			t.Errorf("cell %d read back as %d, want %d", i, got[i], want.Params[i])
		}
	}
}

// TestAnEmptyTableStillParses is the fixture every "no rows of my own" caller
// gets, and the one a stream with eleven zero counts has to survive.
func TestAnEmptyTableStillParses(t *testing.T) {
	b := synth.DataBinUnitsTable(nil, nil)
	f, err := databin.Parse(b)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.Consumed != len(b) {
		t.Errorf("the walk consumed %d of %d byte(s)", f.Consumed, len(b))
	}
	if got := len(f.Collection(humansID).Entries); got != 0 {
		t.Errorf("an empty one-based collection holds %d entries, want none", got)
	}
}

// TestTitlesReachEveryGroup checks the one array that is counted rather than a
// fixed run — the shape this format invites a writer to get wrong.
func TestTitlesReachEveryGroup(t *testing.T) {
	d := synth.DataBin{Titles: []string{"one", "two"}}
	f, err := databin.Parse(d.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, id := range []databin.ID{unitsID, humansID, databin.ID(synth.DataBinSpells)} {
		got := f.Collection(id).Titles
		if len(got) != 2 || got[0] != "one" || got[1] != "two" {
			t.Errorf("%s read back titles %q, want [one two]", id, got)
		}
	}
}
