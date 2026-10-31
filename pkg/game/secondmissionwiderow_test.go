package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondWideRowMission is a mission 10 holding one placed Human of the
// player whose Humans row is 256, as the knights of missions 73 and 86 are
// (R2-SESSION-297).
func secondWideRowMission() secondMissionContent {
	p := humansParams(40, 0, 52)
	p[16], p[18], p[24] = 9, 0, 7301
	return secondMissionContent{side: 128, humans: map[int]dbEntry{256: {name: "Knight", params: p}},
		units: []alm.Unit{{X: 0x3080, Y: 0x3080, ClassID: 9, ClassSubID: 1, Flags: 0x10, ServerID: 7301, Owner: 1, UnitID: 7}}}
}

func secondWideRowKnight(t *testing.T, w *sim.World) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.MapUnitID == 7 {
			return e
		}
	}
	t.Fatal("the World holds no placed knight")
	return sim.Entity{}
}

// secondWideRowSave is the save page's mission file of f, whose app is app,
// holding the placed knight.
func secondWideRowSave(t *testing.T, f *FrontEnd, app *ui.App) []byte {
	t.Helper()
	if knight := secondWideRowKnight(t, f.live.world); !knight.Humanoid || knight.TypeID != 9 {
		t.Fatalf("the placed knight is humanoid %v type %d, want a Human of type 9", knight.Humanoid, knight.TypeID)
	}
	return secondPageSave(t, f, app)
}

// secondPageSave is the save page's mission file of f, whose app is app.
func secondPageSave(t *testing.T, f *FrontEnd, app *ui.App) []byte {
	t.Helper()
	dir := t.TempDir()
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(100, 0) })
	prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
		OnMap: true, Directory: dir, Name: "page", Label: "page", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "page.sav"))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// checkSecondWideRowRecord: the file's one knight record holds row 256 in
// +0xc and the row's type 9 in +0xe.
func checkSecondWideRowRecord(t *testing.T, name string, raw []byte) {
	t.Helper()
	doc := checkSecondMissionFile(t, raw, 10)
	value := func(r sav.DocumentRecordData, name string) uint32 {
		for _, v := range r.Values {
			if v.Name == name {
				return v.Value
			}
		}
		return 0
	}
	found := 0
	for _, r := range doc.Objects {
		if r.Class == "Human" && uint16(value(r, "T08")) == 7 {
			found++
			if value(r, "T0C") != 256 || value(r, "T0E") != 9 {
				t.Fatalf("%s: the knight's record holds row %d type %d, want 256 and 9", name, value(r, "T0C"), value(r, "T0E"))
			}
		}
	}
	if found != 1 {
		t.Fatalf("%s: %d knight records, want 1", name, found)
	}
}

// A second-game mission whose placed Human is Humans row 256 SAVEs a Bsg&
// file whose Human record holds the whole row in +0xc and the row's type in
// +0xe (R2-ENGINE-739). Its LOAD has the World hash of a LOAD of the
// engine's own document, and a LOAD of its records alone, as of an
// original's file, SAVEs the row again. Fails while a row above 255
// refuses the mission's actor construction.
func TestSecondMissionSavesAPlacedHumanAboveRow255(t *testing.T) {
	c := secondWideRowMission()
	f, app, screen, _ := secondBagFrontOf(t, secondBagSample(), c)
	secondBagEnter(t, f, app, screen)
	raw := secondWideRowSave(t, f, app)
	checkSecondWideRowRecord(t, "the mission SAVE", raw)
	cold := secondMissionLoadBytesOf(t, c, raw)
	if cold.live == nil {
		t.Fatal("the file does not LOAD")
	}
	own, err := sav.EngineDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	engine := secondMissionLoadBytesOf(t, c, own)
	if cold.live.world.Hash() != engine.live.world.Hash() {
		t.Fatalf("the World hash is %x, the engine document's LOAD %x", cold.live.world.Hash(), engine.live.world.Hash())
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0001.sav"), withoutEngineDocument(t, raw), 0o600); err != nil {
		t.Fatal(err)
	}
	bare, bareApp := secondMissionLoadOf(t, c, dir, "game0001.sav")
	if bare.live == nil || bare.liveMission != 10 {
		t.Fatal("the file's records alone do not LOAD mission 10")
	}
	checkSecondWideRowRecord(t, "the records' resave", secondWideRowSave(t, bare, bareApp))
}
