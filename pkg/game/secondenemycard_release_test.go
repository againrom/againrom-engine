package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// secondCardRows groups the live creatures of the mission by the Units row
// their placement resolves to, in placement order, with one type per row.
func secondCardRows(f *FrontEnd, live *mapWorld) (map[int][]sim.EntityID, map[int]int32) {
	ids, types := map[int][]sim.EntityID{}, map[int]int32{}
	seen := map[sim.EntityID]bool{}
	for _, u := range live.mission.state.Map.Units {
		r := mapload.Resolve(u, f.Table)
		if !r.Found() || r.Index < 64 || u.UnitID == 0 {
			continue
		}
		for _, e := range live.world.Entities() {
			if e.MapUnitID == u.UnitID && !e.Humanoid && e.Owner != sim.SelfSlot && !seen[e.ID] {
				seen[e.ID] = true
				ids[r.Index] = append(ids[r.Index], e.ID)
				types[r.Index] = e.TypeID
			}
		}
	}
	return ids, types
}

func secondCardSetCount(w *sim.World, row int, count uint32) {
	var entries []sim.SavedDiaryEntry
	if count != 0 {
		entries = []sim.SavedDiaryEntry{{Index: row, Count: count, Remaining: uint16(1024 - count)}}
	}
	w.SetSavedDiaries([]sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: w.DiaryRows(), Entries: entries}})
}

// The second game's creature cards follow the Diary count of their Units row
// for every creature type, a creature above type 80 included, and a SAVE and a
// cold LOAD keep the count.
func TestReleaseSecondEnemyCardKnowledgeFollowsKillsAndSurvivesLoad(t *testing.T) {
	f, app := secondMissionSettledAt(t, 20)
	live := f.live
	w := live.world
	ids, types := secondCardRows(f, live)
	high, low := -1, -1
	for row, list := range ids {
		if len(list) < 3 {
			continue
		}
		if types[row] > 80 && (high < 0 || row < high) {
			high = row
		}
		if types[row] <= 80 && (low < 0 || row < low) {
			low = row
		}
	}
	if high < 0 {
		t.Fatalf("no row of three creatures above type 80: %v", types)
	}
	rowsOf := []int{high}
	if low >= 0 {
		rowsOf = append(rowsOf, low)
	}
	for _, row := range rowsOf {
		for count := uint32(0); count <= 17; count++ {
			secondCardSetCount(w, row, count)
			for _, id := range ids[row] {
				if l := w.KnowledgeLevel(mustEntity(t, live, id)); l != enemyCardWant(count) {
					t.Fatalf("row %d type %d count %d: entity %d level %d, want %d", row, types[row], count, id, l, enemyCardWant(count))
				}
			}
			shown := 0
			for _, id := range ids[row] {
				if s, got, ok := enemyCardHover(app, live, id); ok && got == id {
					shown++
					if s.DetailLevel != enemyCardWant(count) {
						t.Fatalf("row %d count %d: card of entity %d level %d, want %d", row, count, id, s.DetailLevel, enemyCardWant(count))
					}
				}
			}
			if shown == 0 {
				t.Fatalf("row %d: no instance could be hovered", row)
			}
		}
	}

	secondCardSetCount(w, high, 0)
	enemyCardKill(t, live, ids[high][0])
	if got := enemyCardCount(w, high); got != 1 {
		t.Fatalf("one kill gave count %d", got)
	}
	if l := w.KnowledgeLevel(mustEntity(t, live, ids[high][1])); l != 0 {
		t.Fatalf("one kill: level %d, want 0", l)
	}
	enemyCardKill(t, live, ids[high][1])
	for _, id := range ids[high][2:] {
		if l := w.KnowledgeLevel(mustEntity(t, live, id)); l != 1 {
			t.Fatalf("two kills: entity %d level %d, want 1", id, l)
		}
	}

	out := t.TempDir()
	secondMissionNamedSave(t, f, app, out, "card")
	cold, _ := secondMissionColdAt(t, out, secondSaveFile(t, out, "card"), 20)
	if got := enemyCardCount(cold.live.world, high); got != 2 {
		t.Fatalf("cold LOAD count %d, want 2", got)
	}
	for _, id := range ids[high][2:] {
		if l := cold.live.world.KnowledgeLevel(mustEntity(t, cold.live, id)); l != 1 {
			t.Fatalf("cold LOAD: entity %d level %d, want 1", id, l)
		}
	}
}

// The Diary a second-game town carries between missions is written by a town
// SAVE and restored by a cold LOAD, so the next mission opens with it.
func TestReleaseSecondEnemyCardKnowledgeSurvivesTownSave(t *testing.T) {
	f, app, _ := secondIglezJoins(t)
	want := []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 242,
		Entries: []sim.SavedDiaryEntry{{Index: 200, Count: 5, Remaining: 1019}}}}
	f.Town.knowledge = want
	out := t.TempDir()
	secondTownNamedSave(t, f, app, out, "Card")
	cold, _ := secondTownCold(t, out, secondSaveFile(t, out, "Card"))
	got := cold.Town.knowledgeDiaries()
	if len(got) != 1 || got[0].Length != 242 || len(got[0].Entries) != 1 || got[0].Entries[0] != want[0].Entries[0] {
		t.Fatalf("cold LOAD town Diary %+v, want %+v", got, want)
	}
}

// secondCardLooks groups the live creatures of the mission by the type and
// face of their Units row: the rows of one look and the creatures of each.
func secondCardLooks(f *FrontEnd, live *mapWorld) map[[2]int32]map[int][]sim.EntityID {
	ids, _ := secondCardRows(f, live)
	looks := map[[2]int32]map[int][]sim.EntityID{}
	for row, list := range ids {
		typeID, face, ok := data.UnitKeyPair(f.Table.Units.EntryParams(row))
		if !ok {
			continue
		}
		key := [2]int32{typeID, face}
		if looks[key] == nil {
			looks[key] = map[int][]sim.EntityID{}
		}
		looks[key][row] = list
	}
	return looks
}
