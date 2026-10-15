package game

import (
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestDifficultyNativeCompatibilityAndAtomicRefusal(t *testing.T) {
	for _, v := range []int64{-2147483648, -1, 4, 2147483647, 4294967297} {
		if _, err := campaignDifficulty(v); err == nil {
			t.Fatalf("accepted difficulty %d", v)
		}
	}
	f := &FrontEnd{CampaignSession: CampaignSession{Difficulty: mapload.DifficultyNormal, Town: NewTown(Campaign{})}}
	oldTown := f.Town
	for _, difficulty := range []uint32{4, 2147483647, 0xffffffff} {
		if _, _, err := f.RestoreOriginal(originalDifficultyFixture(t, 0, difficulty)); err == nil || !strings.Contains(err.Error(), "difficulty") {
			t.Fatalf("invalid SAV restore=%v", err)
		}
		if f.Town != oldTown || f.Difficulty != mapload.DifficultyNormal {
			t.Fatal("invalid restore changed live session")
		}
	}
}

func originalDifficultyFixture(t *testing.T, mission, difficulty uint32) []byte {
	t.Helper()
	sf, err := sav.Open(savedFile(mission, nil))
	if err != nil {
		t.Fatal(err)
	}
	sf.SetDifficulty(difficulty)
	return sf.Marshal()
}

func TestDifficultyOriginalTownImportAndNativePersistence(t *testing.T) {
	source, newFront := spellbookCitySource(t)
	for _, level := range []uint32{1, 2, 3} {
		f := newFront()
		f.Difficulty = mapload.DifficultyHard
		if _, town, err := f.RestoreOriginal(withDifficultyWords(t, source, level, level-1)); err != nil || !town {
			t.Fatalf("original %d: town=%v err=%v", level, town, err)
		}
		if f.Difficulty != mapload.Difficulty(level) {
			t.Fatalf("original %d became %d", level, f.Difficulty)
		}
		s, _, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		b, err := f.ExportCurrentSave(s, "difficulty")
		if err != nil {
			t.Fatal(err)
		}
		g := newFront()
		if _, _, err := g.RestoreOriginal(b); err != nil {
			t.Fatal(err)
		}
		if g.Difficulty != mapload.Difficulty(level) {
			t.Fatal("native import fallback lost difficulty")
		}
	}
	f := &FrontEnd{CampaignSession: CampaignSession{Difficulty: mapload.DifficultyEasy, Town: NewTown(Campaign{})}}
	town := f.Town
	for _, level := range []uint32{4, 0xffffffff} {
		if _, _, err := f.RestoreOriginal(withDifficultyWords(t, source, level, level-1)); err == nil {
			t.Fatal("invalid original difficulty accepted")
		}
		if f.Town != town || f.Difficulty != mapload.DifficultyEasy {
			t.Fatal("invalid original difficulty mutated session")
		}
	}
}

func TestDifficultyOriginalCityWritesTheChangedLevel(t *testing.T) {
	f, old := originalCityRouteFixture(t)
	for _, level := range []mapload.Difficulty{mapload.DifficultyHard, mapload.DifficultyHard, mapload.DifficultyEasy} {
		f.Difficulty = level
		snapshot, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil || doc.Head.Difficulty != uint32(level) {
			t.Fatal("current difficulty not written", doc.Head.Difficulty, level, err)
		}
	}
	if len(old.updates) != 0 {
		t.Fatal("difficulty SAVE called the retained writer")
	}
}

func TestDifficultyNewGameDraftSurvivesFailureAndCommitsAfterOpen(t *testing.T) {
	f := missionFrontEnd(t)
	f.Difficulty = mapload.DifficultyHard
	f.Town = NewTown(Campaign{})
	f.Town.gold = 123
	f.Carried = saveParty()
	oldTown := f.Town
	res := ui.ChargenResult{Name: "Test", Difficulty: 1, Stats: []int{25, 25, 25, 25}}
	if _, _, _, _, _, _, _, _, _, _, err := f.NewGameOpener(20, res)(); err == nil {
		t.Fatal("missing map opened")
	}
	if f.Difficulty != mapload.DifficultyHard || f.Town != oldTown || len(f.Carried) == 0 {
		t.Fatal("failed new game reset the old campaign")
	}
	open, err := f.prepareNewGame(10, res)
	if err != nil {
		t.Fatal(err)
	}
	if f.Difficulty != mapload.DifficultyHard || f.Town != oldTown {
		t.Fatal("prepared draft committed early")
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	if f.Difficulty != mapload.DifficultyEasy || f.Town == oldTown || len(f.Carried) != 0 || f.live == nil {
		t.Fatal("new game did not commit selected difficulty and fresh session")
	}
}
