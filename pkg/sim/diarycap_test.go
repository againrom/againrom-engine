package sim

import "testing"

// diaryKillRun kills n creatures of Units row 70 in w, the hero credited,
// and returns the entity of a further creature of that row.
func diaryKillRun(t *testing.T, second bool, n int) (*World, EntityID) {
	t.Helper()
	units := map[uint16]DiaryUnit{}
	ents := []Entity{diaryHero(1)}
	for i := range n {
		id := EntityID(2 + i)
		units[uint16(id)] = DiaryUnit{Row: 70, Face: 2}
		ents = append(ents, diaryBody(id, 66, 1))
	}
	witness := EntityID(100)
	units[uint16(witness)] = DiaryUnit{Row: 70, Face: 2}
	ents = append(ents, diaryBody(witness, 66, 1))
	w := diaryWorld(t, units, ents...)
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	for i := range n {
		diaryKill(w, EntityID(2+i))
	}
	return w, witness
}

// The second game's count caps at 14 and its grade is half the count, 7 at
// the cap (R2-ENGINE-1269, R2-ENGINE-1272); the first game's cap stays 17.
func TestSecondGameKillCountCapsAtFourteenAndGradesSeven(t *testing.T) {
	for _, tc := range []struct {
		second       bool
		kills        int
		count, level uint32
		wantRem      uint16
	}{
		{true, 15, 14, 7, 1024 - 15},
		{true, 14, 14, 7, 1024 - 14},
		{true, 13, 13, 6, 1024 - 13},
		{false, 15, 15, 7, 1024 - 15},
		{false, 20, 17, 7, 1024 - 20},
	} {
		w, witness := diaryKillRun(t, tc.second, tc.kills)
		d, ok := w.playerDiary()
		if !ok || len(d.Entries) != 1 || d.Entries[0].Count != tc.count || d.Entries[0].Remaining != tc.wantRem {
			t.Fatalf("second=%v kills %d: diary %+v, want count %d remaining %d", tc.second, tc.kills, d, tc.count, tc.wantRem)
		}
		if got := w.knowledgeLevelOf(witness); got != int(tc.level) {
			t.Fatalf("second=%v kills %d: grade %d, want %d", tc.second, tc.kills, got, tc.level)
		}
	}
}

// A second-game decode folds a look's counts to the second game's cap.
func TestSecondGameFoldCapsAtFourteen(t *testing.T) {
	w := diaryWorld(t, nil, diaryHero(1))
	w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	w.SetDiaryFold(map[uint16]uint16{70: 68})
	got := w.FoldDiaryLooks([]SavedDiary{{Owner: SavedDiaryOwner{Player: true}, Length: 119, Entries: []SavedDiaryEntry{
		{Index: 68, Count: 9, Remaining: 1015}, {Index: 70, Count: 9, Remaining: 1015},
	}}})
	if e := got[0].Entries; len(e) != 1 || e[0].Index != 68 || e[0].Count != 14 {
		t.Fatalf("folded entries %+v, want one entry at row 68 with count 14", e)
	}
}

// Only the second game draws a card at the full level under the show-map
// cheat (R2-ENGINE-1269).
func TestShowMapFullCardIsTheSecondGameProfile(t *testing.T) {
	first, _ := diaryKillRun(t, false, 0)
	second, _ := diaryKillRun(t, true, 0)
	if first.ShowMapFullCard() || !second.ShowMapFullCard() {
		t.Fatalf("show-map full card: first %v second %v", first.ShowMapFullCard(), second.ShowMapFullCard())
	}
}
