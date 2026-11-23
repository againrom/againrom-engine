package game

import (
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const (
	killGradeRow      = 70
	killGradeOtherRow = 72
	killGradeBigRow   = 90
	killGradeZeroRow  = 91
)

// killGradeServerID is the serverID the test table gives a Units row.
func killGradeServerID(row int) int32 {
	switch row {
	case killGradeBigRow:
		return 0xa05
	case killGradeZeroRow:
		return 0
	}
	return int32(600 + row)
}

// killGradeTable is the second game's definition table over a Units table of
// 100 rows, each with its own serverID.
func killGradeTable() *mapload.Table {
	t := secondDollTable()
	units := make(dbCollection, 100)
	for row := 1; row < len(units); row++ {
		units[row] = dbEntry{name: "creature", params: make([]int32, 56)}
		units[row].params[55] = killGradeServerID(row)
	}
	// One row is a victim of class 0x21..0x3f, which goes to +0xa4c.
	units[killGradeOtherRow].params[29] = 0x30
	t.Units = units
	return t
}

// killGradeTownSave is a second-game town SAVE whose town carries diary.
func killGradeTownSave(t *testing.T, diary []sim.SavedDiary) []byte {
	t.Helper()
	f, _, _ := secondCampaignFixtureFiles(t, false)
	f.Table = killGradeTable()
	hero := mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}
	hero.Hero.Skill[2] = 20
	f.Carried = []mapload.PartyMember{hero}
	c := secondShopSample()
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}, {1, 10}}, secondTownShop
	c.bank[768] = 10
	*c.heroBag() = []sim.ItemInstance{secondDollAxe, secondDollStaff}
	f.Town.second, f.Town.gold = c, 500
	f.Town.knowledge = diary
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// killGradeBlockOf is the kill-grade block of the local Player record of raw.
func killGradeBlockOf(t *testing.T, raw []byte) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if p, _ := savedStructureValue(r, "Participant"); r.Class == "Player" && p == 0 {
			return documentRawBytes(t, r, sav.KillGradeBlock)
		}
	}
	t.Fatal("no local Player record")
	return nil
}

func killGradeColdDiary(t *testing.T, raw []byte) []sim.SavedDiaryEntry {
	t.Helper()
	cold, _, _ := secondCampaignFixtureFiles(t, false)
	cold.Table = killGradeTable()
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	d := cold.Town.knowledgeDiaries()
	if len(d) == 0 {
		return nil
	}
	return d[0].Entries
}

func killGradeDiary70(count uint32) []sim.SavedDiary {
	return []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 100,
		Entries: []sim.SavedDiaryEntry{{Index: killGradeRow, Count: count, Remaining: uint16(1024 - count)}}}}
}

// A second-game SAVE writes the local Player's counts at the serverID byte of
// the Player's 2,560-byte block, and a cold LOAD reads the count back
// (R2-SESSION-417, R2-ENGINE-1272).
func TestSecondSaveWritesTheKillGradeBlockAtTheServerIDByte(t *testing.T) {
	raw := killGradeTownSave(t, killGradeDiary70(14))
	block := killGradeBlockOf(t, raw)
	if len(block) != 2560 {
		t.Fatalf("block length %d", len(block))
	}
	for i, b := range block {
		want := byte(0)
		if i == int(killGradeServerID(killGradeRow)) {
			want = 14
		}
		if b != want {
			t.Fatalf("block byte %d = %d, want %d", i, b, want)
		}
	}
	for name, file := range map[string][]byte{"the town file": raw, "the records alone": withoutEngineDocument(t, raw)} {
		got := killGradeColdDiary(t, file)
		if len(got) != 1 || got[0].Index != killGradeRow || got[0].Count != 14 {
			t.Fatalf("%s: cold LOAD diary %+v, want row %d count 14", name, got, killGradeRow)
		}
	}
}

// Rows with no byte stay out of the block: a serverID of 0 and one above 0x9ff.
func TestSecondSaveLeavesRowsWithoutAServerIDByteOut(t *testing.T) {
	diary := killGradeDiary70(4)
	diary[0].Entries = append(diary[0].Entries,
		sim.SavedDiaryEntry{Index: killGradeBigRow, Count: 6, Remaining: 1018},
		sim.SavedDiaryEntry{Index: killGradeZeroRow, Count: 8, Remaining: 1016})
	block := killGradeBlockOf(t, killGradeTownSave(t, diary))
	var set []int
	for i, b := range block {
		if b != 0 {
			set = append(set, i)
		}
	}
	if !slices.Equal(set, []int{int(killGradeServerID(killGradeRow))}) || block[0] != 0 {
		t.Fatalf("nonzero block bytes at %v", set)
	}
}

// An original-shaped Player block with a nonzero byte loads as that count on
// the byte's Units row; byte 0 and a byte with no row are no count.
func TestOriginalPlayerBlockLoadsAsCounts(t *testing.T) {
	raw := withoutEngineDocument(t, killGradeTownSave(t, nil))
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if p, _ := savedStructureValue(r, "Participant"); r.Class != "Player" || p != 0 {
			continue
		}
		for j := range r.Raw {
			if r.Raw[j].Name == sav.KillGradeBlock {
				r.Raw[j].Bytes[0] = 9
				r.Raw[j].Bytes[killGradeServerID(killGradeOtherRow)] = 5
				r.Raw[j].Bytes[0x9ff] = 3
			}
		}
	}
	patched, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	got := killGradeColdDiary(t, patched)
	if len(got) != 1 || got[0].Index != killGradeOtherRow || got[0].Count != 5 || got[0].Remaining != 1019 {
		t.Fatalf("cold LOAD diary %+v, want row %d count 5", got, killGradeOtherRow)
	}
}

// A mission LOAD of an original second-game file fills the local Player's
// Diary from the block through the table's rows.
func TestOriginalMissionLoadFillsTheDiaryFromTheBlock(t *testing.T) {
	block := make([]byte, 2560)
	block[killGradeServerID(killGradeRow)] = 14
	block[killGradeServerID(killGradeOtherRow)] = 2
	var w sim.World
	ms := &Mission{World: &w}
	if err := applyOriginalDiaries(ms, &sav.Record{Raw: map[string][]byte{sav.KillGradeBlock: block}}, nil, nil, killGradeTable()); err != nil {
		t.Fatal(err)
	}
	d := w.SavedDiaries()
	if len(d) != 1 || !d[0].Owner.Player || len(d[0].Entries) != 2 ||
		d[0].Entries[0].Index != killGradeRow || d[0].Entries[0].Count != 14 ||
		d[0].Entries[1].Index != killGradeOtherRow || d[0].Entries[1].Count != 2 {
		t.Fatalf("Diaries %+v", d)
	}
}

// killGradeLocalKills is the local Player's F48 of raw.
func killGradeLocalKills(t *testing.T, raw []byte) uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if p, _ := savedStructureValue(r, "Participant"); r.Class == "Player" && p == 0 {
			v, err := savedStructureValue(r, "F48")
			if err != nil {
				t.Fatal(err)
			}
			return v
		}
	}
	t.Fatal("no local Player record")
	return 0
}

// The Player's kill count has no cap, the grade byte stops at 14, and a victim
// of class 0x21..0x3f counts into +0xa4c, not the kill count (R2-ENGINE-772).
func TestSecondSaveKillCountPastTheGradeCap(t *testing.T) {
	diary := []sim.SavedDiary{{Owner: sim.SavedDiaryOwner{Player: true}, Length: 100,
		Entries: []sim.SavedDiaryEntry{
			{Index: killGradeRow, Count: 14, Remaining: 1024 - 30},
			{Index: killGradeOtherRow, Count: 5, Remaining: 1024 - 5},
		}}}
	raw := killGradeTownSave(t, diary)
	if got := killGradeLocalKills(t, raw); got != 30 {
		t.Fatalf("F48 = %d, want 30", got)
	}
	block := killGradeBlockOf(t, raw)
	if g := block[killGradeServerID(killGradeRow)]; g != 14 {
		t.Fatalf("grade %d, want 14", g)
	}
	if g := block[killGradeServerID(killGradeOtherRow)]; g != 5 {
		t.Fatalf("band victim grade %d, want 5", g)
	}
}
