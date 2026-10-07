package data

import (
	"slices"
	"testing"
)

type fakeRows struct {
	names   []string
	params  [][]int32
	strings [][]string
}

func (f fakeRows) Len() int                    { return len(f.names) }
func (f fakeRows) EntryName(i int) string      { return f.names[i] }
func (f fakeRows) EntryParams(i int) []int32   { return f.params[i] }
func (f fakeRows) EntryStrings(i int) []string { return f.strings[i] }

func TestEditedRowsChangeTheNamedRowsOnly(t *testing.T) {
	base := fakeRows{
		names:   []string{"", "A_1", "B_1", "B_2"},
		params:  [][]int32{nil, {1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
		strings: [][]string{nil, {"bow", "", "mail"}, {"axe"}, nil},
	}
	e := NewEditedRows(base, map[int]RowEdit{
		2: {Params: map[int]int32{1: 50, 9: 99, -1: 7}, Cells: map[int]string{0: "", 3: "x"}},
		3: {Cells: map[int]string{0: "club"}},
		9: {Params: map[int]int32{0: 1}},
		0: {Params: map[int]int32{0: 1}},
	})
	if e.Len() != 4 {
		t.Fatalf("len %d", e.Len())
	}
	for i, name := range base.names {
		if e.EntryName(i) != name {
			t.Fatalf("name %d", i)
		}
	}
	if !slices.Equal(e.EntryParams(1), base.params[1]) || !slices.Equal(e.EntryStrings(1), base.strings[1]) {
		t.Fatal("an unedited row changed")
	}
	if !slices.Equal(e.EntryParams(2), []int32{4, 50, 6}) {
		t.Fatalf("params %v: the slot past the row or below zero must be ignored", e.EntryParams(2))
	}
	if !slices.Equal(e.EntryStrings(2), []string{"", "", "", "x"}) {
		t.Fatalf("strings %v", e.EntryStrings(2))
	}
	if !slices.Equal(e.EntryParams(3), base.params[3]) || !slices.Equal(e.EntryStrings(3), []string{"club"}) {
		t.Fatalf("row 3 %v %v", e.EntryParams(3), e.EntryStrings(3))
	}
	if base.params[2][1] != 5 || len(base.strings[2]) != 1 || base.strings[2][0] != "axe" {
		t.Fatal("the base was changed")
	}
}

func TestEditedRowsOverNothing(t *testing.T) {
	e := NewEditedRows(nil, map[int]RowEdit{1: {Params: map[int]int32{0: 1}}})
	if e.Len() != 0 || e.EntryName(1) != "" || e.EntryParams(1) != nil || e.EntryStrings(1) != nil {
		t.Fatal("rows over a nil base")
	}
}
