package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// A second-game save written before one look was one count holds counts on
// the look's later Units rows. Its cold LOAD folds them onto the look's first
// row, so recording, reading and writing see that row alone.
func TestReleaseSecondGameEnemyCardOldSaveCountsFoldOnLoad(t *testing.T) {
	f, app := secondMissionSettledAt(t, 20)
	w := f.live.world
	units := f.Table.Units
	typeID, face, _ := data.UnitKeyPair(units.EntryParams(179))
	if t2, f2, _ := data.UnitKeyPair(units.EntryParams(180)); t2 != typeID || f2 != face {
		t.Fatalf("rows 179 and 180 are not one look: %d/%d and %d/%d", typeID, face, t2, f2)
	}
	first := data.FindUnit(units, typeID, face)
	if first <= 0 || first >= 179 {
		t.Fatalf("first row of type %d face %d is %d, want an earlier row", typeID, face, first)
	}
	w.SetSavedDiaries([]sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: w.DiaryRows(),
		Entries: []sim.SavedDiaryEntry{{Index: 179, Count: 3, Remaining: 1021}, {Index: 180, Count: 2, Remaining: 1022}}}})
	out := t.TempDir()
	secondMissionNamedSave(t, f, app, out, "old")
	cold, _ := secondMissionColdAt(t, out, secondSaveFile(t, out, "old"), 20)
	cw := cold.live.world
	if got := enemyCardCount(cw, first); got != 5 {
		t.Fatalf("cold LOAD counts %d on row %d, want 5", got, first)
	}
	if a, b := enemyCardCount(cw, 179), enemyCardCount(cw, 180); a != 0 || b != 0 {
		t.Fatalf("cold LOAD left counts %d and %d on rows 179 and 180", a, b)
	}
}
