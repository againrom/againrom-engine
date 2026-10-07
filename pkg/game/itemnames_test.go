package game_test

// Tests for the shipped item-name table's address and read (0151,
// ITEM-DISPNAME-036). ReadItemNames is ReadBodyList's contract one pair of
// files over; the parse itself is pkg/formats/itemname's own test.
//
// Every payload here is invented text, written out by hand — never the
// shipped table.

import (
	"io/fs"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/game"
)

// mapSource is a fixed set of named entries — selSource's one-entry contract
// widened to as many as a test needs, for ReadItemNames' own two files.
type mapSource map[string][]byte

func (s mapSource) ReadFile(name string) ([]byte, error) {
	if b, ok := s[name]; ok {
		return b, nil
	}
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func TestItemNameAddressesAreUnderMainsTextTree(t *testing.T) {
	if got, want := game.ItemNameKeyAddress, "main/text/itemname.bin"; got != want {
		t.Errorf("ItemNameKeyAddress = %q, want %q", got, want)
	}
	if got, want := game.ItemNameTextAddress, "main/text/itemname.txt"; got != want {
		t.Errorf("ItemNameTextAddress = %q, want %q", got, want)
	}
}

func TestReadItemNamesPairsBothFiles(t *testing.T) {
	src := mapSource{
		game.ItemNameKeyAddress:  {0x01, 0x01},
		game.ItemNameTextAddress: []byte("Sword\r\n"),
	}
	got, ok := game.ReadItemNames(src)
	if !ok {
		t.Fatal("both files present: refused, want it read")
	}
	if s, ok := got.NameFor(data.ItemCode(0x0101)); !ok || s != "Sword" {
		t.Errorf("NameFor(0x0101) = (%q, %v), want (%q, true)", s, ok, "Sword")
	}
}

func TestReadItemNamesIsSilentWithNoSourceAtAll(t *testing.T) {
	if got, ok := game.ReadItemNames(nil); ok || got != nil {
		t.Fatalf("ReadItemNames(nil) = %#v, %v; want nil, false", got, ok)
	}
}

func TestReadItemNamesIsSilentWhenEitherFileIsMissing(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  mapSource
	}{
		{"neither file present", mapSource{}},
		{"only the key file", mapSource{game.ItemNameKeyAddress: {0x01, 0x01}}},
		{"only the text file", mapSource{game.ItemNameTextAddress: []byte("Sword\r\n")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := game.ReadItemNames(tc.src)
			if ok || got != nil {
				t.Fatalf("ReadItemNames = %#v, %v; want nil, false", got, ok)
			}
		})
	}
}
