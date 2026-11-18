package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// The fold moves nothing in the first game: its Units table repeats no type
// and face, so no row folds. The second game's repeated rows all fold.
func TestReleaseDiaryFoldCountsRowsOfTheInstall(t *testing.T) {
	f := releaseFront(t)
	units := f.Table.Units
	want := 0
	for row := 1; row < units.Len(); row++ {
		if units.EntryName(row) == "" {
			continue
		}
		if typeID, face, ok := data.UnitKeyPair(units.EntryParams(row)); ok {
			if first := data.FindUnit(units, typeID, face); first != data.NotFound && first < row {
				want++
			}
		}
	}
	got := len(mapload.DiaryFold(f.Table))
	t.Logf("rows folded: %d of %d rows", got, units.Len())
	if got != want {
		t.Fatalf("fold holds %d rows, the table repeats %d", got, want)
	}
	if f.Table.ClassPlacementFields() && got != 0 {
		t.Fatalf("the first game folds %d rows, want 0", got)
	}
}
