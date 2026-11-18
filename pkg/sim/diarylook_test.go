package sim

import "testing"

// Two Units rows of one look are one count: a kill of either row raises the
// first row's count, and a body of the other row reads it.
func TestDiaryCountsOneLookUnderItsFirstRow(t *testing.T) {
	units := map[uint16]DiaryUnit{
		2: {Row: 70, Face: 2, First: 68},
		3: {Row: 72, Face: 2, First: 68},
		4: {Row: 68, Face: 2},
	}
	w := diaryWorld(t, units, diaryHero(1), diaryBody(2, 66, 1), diaryBody(3, 66, 1), diaryBody(4, 66, 1))
	diaryKill(w, 2)
	diaryKill(w, 3)
	d, ok := w.playerDiary()
	if !ok || len(d.Entries) != 1 || d.Entries[0].Index != 68 || d.Entries[0].Count != 2 {
		t.Fatalf("Player diary after two kills = %+v, want one entry at row 68 with count 2", d)
	}
	for id := EntityID(2); id <= 4; id++ {
		if l := w.knowledgeLevelOf(id); l != 1 {
			t.Fatalf("entity %d level %d, want 1", id, l)
		}
	}
}

// A Diary written before one look was one count holds counts on the look's
// later rows; a decode folds them onto the first row, the counts add to the
// cap and the spent remainders add.
func TestDiaryDecodeFoldsRepeatedRowsOntoTheFirstRow(t *testing.T) {
	w := diaryWorld(t, nil, diaryHero(1))
	w.SetDiaryFold(map[uint16]uint16{70: 68, 72: 68})
	w.savedDiaries = []SavedDiary{{Owner: SavedDiaryOwner{Player: true}, Length: 119, Entries: []SavedDiaryEntry{
		{Index: 68, Count: 1, Remaining: 1023},
		{Index: 70, Count: 3, Remaining: 1021},
		{Index: 72, Count: 16, Remaining: 1010},
		{Index: 90, Count: 4, Remaining: 1020},
	}}}
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	cold.SetDiaryFold(map[uint16]uint16{70: 68, 72: 68})
	if err := cold.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	d, ok := cold.playerDiary()
	want := []SavedDiaryEntry{{Index: 68, Count: DiaryCountCap, Remaining: 1024 - 1 - 3 - 14}, {Index: 90, Count: 4, Remaining: 1020}}
	if !ok || len(d.Entries) != len(want) {
		t.Fatalf("decoded Diary = %+v, want entries %+v", d, want)
	}
	for i := range want {
		if d.Entries[i] != want[i] {
			t.Fatalf("entry %d = %+v, want %+v", i, d.Entries[i], want[i])
		}
	}
	if again := cold.FoldDiaryLooks(cold.SavedDiaries()); len(again[0].Entries) != len(want) || again[0].Entries[0] != want[0] {
		t.Fatalf("a second fold changed the Diary: %+v", again)
	}
}
