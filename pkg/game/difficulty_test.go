package game

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/gob"
	"hash/crc32"
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
	// A frozen, authored envelope actually written with the old gob schema,
	// not a modern Snapshot explicitly carrying the zero sentinel.
	b, err := base64.StdEncoding.DecodeString(preMercenaryStateSaveFixtureBase64)
	if err != nil {
		t.Fatal(err)
	}
	s, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}
	if s.Difficulty != 0 {
		t.Fatal("old schema somehow supplied difficulty")
	}
	s.Mission, s.World, s.Gold = 0, nil, 123 // exercise the town restore boundary
	f := &FrontEnd{CampaignSession: CampaignSession{Difficulty: mapload.DifficultyHard, Town: NewTown(Campaign{})}}
	if _, town, err := f.Restore(s); err != nil || !town {
		t.Fatalf("old schema restore: %v %v", town, err)
	}
	if f.Difficulty != mapload.DifficultyNormal || f.Town.Gold() != 123 {
		t.Fatalf("old schema became %+v", f)
	}
	oldTown := f.Town
	for _, difficulty := range []mapload.Difficulty{-1, 4, 2147483647} {
		s.Difficulty = difficulty
		if _, err := EncodeSave(s, "invalid"); err == nil {
			t.Fatal("encoder accepted invalid difficulty")
		}
		if _, _, err := f.Restore(s); err == nil || !strings.Contains(err.Error(), "difficulty") {
			t.Fatalf("invalid native restore=%v", err)
		}
		if f.Town != oldTown || f.Difficulty != mapload.DifficultyNormal {
			t.Fatal("invalid restore changed live session")
		}
	}
}

func TestDifficultyInvalidGobIsRejectedAfterBoundedDecode(t *testing.T) {
	var body bytes.Buffer
	if err := gob.NewEncoder(&body).Encode(Snapshot{Difficulty: 99}); err != nil {
		t.Fatal(err)
	}
	b := append([]byte(saveMagic), saveVersion, 0, 0)
	b = binary.LittleEndian.AppendUint32(b, crc32.ChecksumIEEE(body.Bytes()))
	b = binary.LittleEndian.AppendUint32(b, uint32(body.Len()))
	b = append(b, body.Bytes()...)
	if _, _, err := DecodeSave(b); err == nil || !strings.Contains(err.Error(), "difficulty") {
		t.Fatalf("corrupt nonzero difficulty: %v", err)
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
	for _, level := range []uint32{1, 2, 3} {
		f := &FrontEnd{CampaignSession: CampaignSession{Difficulty: mapload.DifficultyHard, Town: NewTown(Campaign{})}}
		if _, town, err := f.RestoreOriginal(originalDifficultyFixture(t, 0, level)); err != nil || !town {
			t.Fatalf("original %d: town=%v err=%v", level, town, err)
		}
		if f.Difficulty != mapload.Difficulty(level) {
			t.Fatalf("original %d became %d", level, f.Difficulty)
		}
		s, _, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		b, err := EncodeSave(s, "difficulty")
		if err != nil {
			t.Fatal(err)
		}
		decoded, _, err := DecodeSave(b)
		if err != nil {
			t.Fatal(err)
		}
		g := &FrontEnd{}
		if _, _, err := g.Restore(decoded); err != nil {
			t.Fatal(err)
		}
		if g.Difficulty != mapload.Difficulty(level) {
			t.Fatal("native import fallback lost difficulty")
		}
	}
	f := &FrontEnd{CampaignSession: CampaignSession{Difficulty: mapload.DifficultyEasy, Town: NewTown(Campaign{})}}
	town := f.Town
	for _, level := range []uint32{4, 0xffffffff} {
		if _, _, err := f.RestoreOriginal(originalDifficultyFixture(t, 0, level)); err == nil {
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
