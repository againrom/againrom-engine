package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestCityQuickSpellsRealIDsToSignedCells1167(t *testing.T) {
	for _, pair := range [][2]uint32{{1, 0}, {2, 1}, {3, 2}, {4, 3}, {5, 4}, {23, 5}, {24, 6}, {16, 7}, {15, 8}, {14, 9}, {13, 10}, {12, 11}, {6, 12}, {7, 13}, {8, 14}, {9, 15}, {10, 16}, {25, 17}, {26, 18}, {22, 19}, {21, 20}, {20, 21}, {19, 22}, {18, 23}} {
		got, err := quickSpellsToOriginalIndices([4]uint32{0, pair[0]})
		if want := (sav.CityShortcuts{-1, int32(pair[1]), -1, -1}); err != nil || got != want {
			t.Fatalf("real ID%d => %v, want %v: %v", pair[0], got, want, err)
		}
	}
	for _, id := range []uint32{11, 17, 27, 28, 65535} {
		got, err := quickSpellsToOriginalIndices([4]uint32{id})
		if err != nil || got != (sav.CityShortcuts{-1, -1, -1, -1}) {
			t.Fatal("custom spell has no ordinary book cell", got, err)
		}
	}
	for _, slots := range [][4]uint32{{65536}, {1, 1}} {
		if _, err := quickSpellsToOriginalIndices(slots); err == nil {
			t.Fatalf("malformed bindings %v accepted", slots)
		}
	}
}
