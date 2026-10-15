package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// difficultyWords reads a SAV's head level and campaign record word.
func difficultyWords(t *testing.T, raw []byte) (head, word uint32) {
	t.Helper()
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	word, ok, err := sf.CampaignDifficulty()
	if err != nil || !ok {
		t.Fatal("SAV has no campaign record word", ok, err)
	}
	return sf.Head.Difficulty, word
}

func withDifficultyWords(t *testing.T, raw []byte, head, word uint32) []byte {
	t.Helper()
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	sf.SetDifficulty(head)
	if err := sf.SetCampaignScalar(1, word); err != nil {
		t.Fatal(err)
	}
	return sf.Marshal()
}

func TestDifficultyRecordRule(t *testing.T) {
	for _, d := range []mapload.Difficulty{mapload.DifficultyEasy, mapload.DifficultyNormal, mapload.DifficultyHard} {
		var doc sav.DocumentData
		writeDifficulty(&doc, firstCampaignRules{}, d)
		if doc.Head.Difficulty != uint32(d) || doc.Campaign.Scalars[1] != uint32(d)-1 {
			t.Fatal("first game words", d, doc.Head.Difficulty, doc.Campaign.Scalars[1])
		}
		doc.Campaign.Scalars[1] = 7
		writeDifficulty(&doc, secondCampaignRules{}, d)
		if doc.Head.Difficulty != uint32(d) || doc.Campaign.Scalars[1] != 7 {
			t.Fatal("second game changed its record word", d, doc.Campaign.Scalars[1])
		}
	}
}

func TestDifficultyRecordTownSaveAndLoad(t *testing.T) {
	source, newFront := spellbookCitySource(t)
	// A town file with head 3 and record word 1 loads at Hard, its head.
	mixed := withDifficultyWords(t, source, 3, 1)
	f := newFront()
	if _, town, err := f.RestoreOriginal(mixed); err != nil || !town {
		t.Fatal("mixed town LOAD", town, err)
	}
	if f.Difficulty != mapload.DifficultyHard {
		t.Fatalf("head 3 word 1 town LOADed at %d, want Hard", f.Difficulty)
	}
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "unchanged")
	if err != nil {
		t.Fatal(err)
	}
	if head, word := difficultyWords(t, raw); head != 3 || word != 2 {
		t.Fatalf("unchanged town SAVE wrote head %d word %d, want 3 and 2", head, word)
	}
	for _, level := range []mapload.Difficulty{mapload.DifficultyHard, mapload.DifficultyEasy} {
		f.Difficulty = level
		s, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		if head, word := difficultyWords(t, raw); head != uint32(level) || word != uint32(level)-1 {
			t.Fatalf("town SAVE at %d wrote head %d word %d", level, head, word)
		}
		cold := newFront()
		if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
			t.Fatal("cold town LOAD", town, err)
		}
		if cold.Difficulty != level {
			t.Fatalf("town SAVE at %d LOADed cold at %d", level, cold.Difficulty)
		}
	}
}

func TestDifficultyRecordMissionSaveAndLoad(t *testing.T) {
	front := func() *FrontEnd { return itemMutationFront(t, 1) }
	doc, err := sav.DecodeDocumentData(crossingCellLiteral(t, front(), true))
	if err != nil {
		t.Fatal(err)
	}
	doc.Head.Difficulty, doc.Campaign.Scalars[1] = 3, 1
	mixed, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	load := func(raw []byte) *FrontEnd {
		t.Helper()
		f := front()
		open, town, err := f.RestoreOriginal(raw)
		if err != nil || town {
			t.Fatal("mission LOAD", town, err)
		}
		if err := f.App("difficulty record").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		return f
	}
	f := load(mixed)
	if f.Difficulty != mapload.DifficultyHard {
		t.Fatalf("head 3 word 1 mission LOADed at %d, want Hard", f.Difficulty)
	}
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	if head, word := difficultyWords(t, raw); head != 3 || word != 2 {
		t.Fatalf("unchanged mission SAVE wrote head %d word %d, want 3 and 2", head, word)
	}
	for _, level := range []mapload.Difficulty{mapload.DifficultyHard, mapload.DifficultyEasy} {
		f.Difficulty = level
		s, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(s, label)
		if err != nil {
			t.Fatal(err)
		}
		if head, word := difficultyWords(t, raw); head != uint32(level) || word != uint32(level)-1 {
			t.Fatalf("mission SAVE at %d wrote head %d word %d", level, head, word)
		}
		if cold := load(raw); cold.Difficulty != level {
			t.Fatalf("mission SAVE at %d LOADed cold at %d", level, cold.Difficulty)
		}
	}
}
