package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// idSigns is mission 10's sign (map id 42, R2-ASSET-152) and two more
// Pointers, so that the file holds three structures.
func idSigns() []signpostPlacement {
	return append(firstMissionSignposts(),
		signpostPlacement{id: 43, kind: 0x46, col: 30, row: 30, noRecord: true},
		signpostPlacement{id: 11, kind: 0x48, col: 60, row: 70, noRecord: true})
}

// checkStructureIDs: every structure record of raw holds +4 = 0x6000 plus
// its map id at +8 (R2-ENGINE-857), and the file holds n structures.
func checkStructureIDs(t *testing.T, name string, raw []byte, n int) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	buildings, present, err := f.Buildings()
	if err != nil || !present {
		t.Fatalf("%s: Buildings present %v: %v", name, present, err)
	}
	if len(buildings) != n {
		t.Fatalf("%s: %d structures, want %d", name, len(buildings), n)
	}
	for _, b := range buildings {
		if b.AuthoredID == 0 || b.RuntimeID != 0x6000+b.AuthoredID {
			t.Fatalf("%s: the %s with map id %d holds +4 = %d, want %d", name, b.Class, b.AuthoredID, b.RuntimeID, 0x6000+b.AuthoredID)
		}
		if b.AuthoredID == 42 && b.RuntimeID != 24618 {
			t.Fatalf("%s: the sign holds +4 = %d, want 24618", name, b.RuntimeID)
		}
	}
}

// A second-game mission started from mission 10's map writes every
// structure with +4 = 0x6000 plus its map id (the sign, id 42, holds 24618):
// the start autosave and a named save. The file's LOAD, and the LOAD of its
// records alone, then SAVE the same values. Fails while the records hold the
// allocator's small ids.
func TestSecondMissionStartWritesEachStructureWithItsMapIDPlus0x6000(t *testing.T) {
	signs := idSigns()
	c := signpostMission(10, signs)
	f, app, screen, _ := secondBagFrontOf(t, nil, c)
	f.Town.second.available = []partyPlace{{kind: placeMission, id: 10}}
	dir := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	destinationTownCampaign{screen}.leave()
	if err := app.OpenMission(f.MissionOpener(10)); err != nil || f.live == nil {
		t.Fatalf("%v %s", err, app.HeadlessMessage())
	}
	prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
		OnMap: true, Directory: dir, Name: "page", Label: "page", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal(err)
	}
	var page []byte
	for _, name := range []string{"game0000.sav", "page.sav"} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		checkStructureIDs(t, name, raw, len(signs))
		page = raw
	}
	for _, load := range []struct {
		name string
		raw  []byte
	}{{"the LOAD", page}, {"the records' LOAD", withoutEngineDocument(t, page)}} {
		cold, coldApp := signpostLoad(t, c, load.raw)
		checkStructureIDs(t, load.name+"'s SAVE", secondPageSave(t, cold, coldApp), len(signs))
	}
}

// A LOAD of a file whose structures hold the small ids 2..17 at +4, as this
// engine wrote them before, plays on with every structure, and its SAVE
// writes 0x6000 plus each map id. Fails while the SAVE keeps the loaded ids.
func TestSecondMissionLoadOfSmallStructureIDsWritesTheMapIDPlus0x6000(t *testing.T) {
	signs := idSigns()
	c := signpostMission(10, signs)
	var planted []secondSavedPointer
	for i, s := range signs {
		p := secondSavedPointer{col: uint8(s.col), row: uint8(s.row), kind: uint8(s.kind), id: uint32(s.id), key: 0x0b000001 + uint32(i),
			word: s.word(), runtime: uint32(17 - i)}
		planted = append(planted, p)
	}
	hero := secondMissionFixtureHero()
	body := secondSavedWorldOf(10, 44, &hero, true, 0, planted)
	raw := labelledContainer(sav.SecondGame.Magic, "mission", body,
		append(secondMissionStore(4, 0, -1), secondSavedTail(0, [2]uint32{10, placeMission})...))
	f0, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	if b, _, err := f0.Buildings(); err != nil || len(b) != len(signs) || b[0].RuntimeID != 17 {
		t.Fatalf("the planted file holds %+v: %v", b, err)
	}
	cold, coldApp := signpostLoad(t, c, raw)
	checkSignpostWorld(t, "the planted LOAD", cold.live.world, signs)
	checkStructureIDs(t, "the planted file's SAVE", secondPageSave(t, cold, coldApp), len(signs))
}
