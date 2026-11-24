package sim

import "testing"

// serverKeyWorld holds a hero, kills of victims 2 and 3, and a witness 4, all
// of one row, type and face. The game is the second one when second is set.
func serverKeyWorld(t *testing.T, second bool, row uint16, typeID int32, unit DiaryUnit, humanoid bool) *World {
	t.Helper()
	unit.Row = row
	units := map[uint16]DiaryUnit{2: unit, 3: unit, 4: unit}
	ents := []Entity{diaryHero(1)}
	for id := EntityID(2); id <= 4; id++ {
		b := diaryBody(id, typeID, 1)
		b.Humanoid = humanoid
		ents = append(ents, b)
	}
	w := diaryWorld(t, units, ents...)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	diaryKill(w, 2)
	diaryKill(w, 3)
	return w
}

func diaryCountOf(w *World) uint32 {
	d, _ := w.playerDiary()
	if len(d.Entries) != 1 {
		return 0
	}
	return d.Entries[0].Count
}

// The second game keys its Diary by server id alone: a kind outside the first
// game's type window and a humanoid victim of a high row both count and grade
// (R2-ENGINE-772, R2-ENGINE-1710), and a row with no server id in 1..0x9ff
// neither counts nor grades (R2-ENGINE-1712). The first game's windows stay.
func TestSecondGameDiaryHasNoTypeWindowAndNoHumanoidRowLimit(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		row                     uint16
		typeID                  int32
		unit                    DiaryUnit
		humanoid                bool
		secondCount, firstCount uint32
		secondLevel, firstLevel int
	}{
		{"kind outside the window", 26, 26, DiaryUnit{Face: 1}, false, 2, 2, 1, 0},
		{"humanoid of a high row", 100, 90, DiaryUnit{Face: 2}, true, 2, 0, 1, 0},
		{"no server id", 100, 66, DiaryUnit{Face: 2, NoServerKey: true}, false, 0, 2, 0, 1},
	} {
		s := serverKeyWorld(t, true, tc.row, tc.typeID, tc.unit, tc.humanoid)
		if c, l := diaryCountOf(s), s.knowledgeLevelOf(4); c != tc.secondCount || l != tc.secondLevel {
			t.Fatalf("%s, second game: count %d level %d, want %d and %d", tc.name, c, l, tc.secondCount, tc.secondLevel)
		}
		f := serverKeyWorld(t, false, tc.row, tc.typeID, tc.unit, tc.humanoid)
		if c, l := diaryCountOf(f), f.knowledgeLevelOf(4); c != tc.firstCount || l != tc.firstLevel {
			t.Fatalf("%s, first game: count %d level %d, want %d and %d", tc.name, c, l, tc.firstCount, tc.firstLevel)
		}
	}
}
