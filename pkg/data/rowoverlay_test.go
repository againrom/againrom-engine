package data

import (
	"slices"
	"testing"
)

type fakeMasks struct {
	names  []string
	params [][]int32
	raw    [][]byte
}

func (f fakeMasks) Len() int                    { return len(f.names) }
func (f fakeMasks) EntryName(i int) string      { return f.names[i] }
func (f fakeMasks) EntryParams(i int) []int32   { return f.params[i] }
func (f fakeMasks) EntryStrings(i int) []string { return nil }
func (f fakeMasks) EntryRaw(i int) []byte       { return f.raw[i] }

func TestRowOverlayExtendsAndReplaces(t *testing.T) {
	base := fakeMasks{
		names:  []string{"", "Mail", "Cap"},
		params: [][]int32{nil, {1, 2}, {3, 4}},
		raw:    [][]byte{nil, {9, 9}, {8, 8}},
	}
	o := NewRowOverlay(base, map[int]OverlayRow{
		2: {Name: "Cap", Params: []int32{7, 7}},
		5: {Name: "Added", Params: []int32{5}, Raw: []byte{1, 2}},
		0: {Name: "ignored"},
	})
	if o.Len() != 6 {
		t.Fatalf("len %d", o.Len())
	}
	if o.EntryName(0) != "" || o.EntryName(1) != "Mail" || o.EntryName(2) != "Cap" || o.EntryName(3) != "" || o.EntryName(5) != "Added" || o.EntryName(9) != "" {
		t.Fatal("names")
	}
	if !slices.Equal(o.EntryParams(1), []int32{1, 2}) || !slices.Equal(o.EntryParams(2), []int32{7, 7}) || o.EntryParams(4) != nil || !slices.Equal(o.EntryParams(5), []int32{5}) {
		t.Fatal("params")
	}
	if !slices.Equal(o.EntryRaw(1), []byte{9, 9}) || !slices.Equal(o.EntryRaw(2), []byte{8, 8}) || !slices.Equal(o.EntryRaw(5), []byte{1, 2}) || o.EntryRaw(4) != nil {
		t.Fatal("raw")
	}
	if base.params[2][0] != 3 {
		t.Fatal("the base was changed")
	}
	if o.EntryStrings(1) != nil || o.EntryStrings(5) != nil {
		t.Fatal("strings")
	}
}

func TestRowOverlayOverANilBase(t *testing.T) {
	o := NewRowOverlay(nil, map[int]OverlayRow{3: {Name: "x", Params: []int32{1}}})
	if o.Len() != 4 || o.EntryName(3) != "x" || o.EntryName(1) != "" || o.EntryRaw(3) != nil {
		t.Fatal("nil base")
	}
}
