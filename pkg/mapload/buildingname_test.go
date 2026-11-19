package mapload

import (
	"testing"

	"againrom/pkg/formats/databin"
)

func TestBuildingNameSaysWhenNoRowNamesTheKind(t *testing.T) {
	tab := &Table{Buildings: &databin.Collection{Entries: []databin.Entry{{}, {Name: "Hut"}, {}}}}
	if row, name, ok := BuildingName(tab, 1); !ok || row != 1 || name != "Hut" {
		t.Fatalf("kind 1: %d %q %v", row, name, ok)
	}
	for _, kind := range []uint32{0, 2, 3, 0x101} {
		if _, name, ok := BuildingName(tab, kind); kind != 0x101 && (ok || name != "") || kind == 0x101 && !ok {
			t.Fatalf("kind %#x: %q %v", kind, name, ok)
		}
	}
	if _, _, ok := BuildingName(nil, 1); ok {
		t.Fatal("a nil table names a row")
	}
}
