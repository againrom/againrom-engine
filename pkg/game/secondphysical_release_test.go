package game

import (
	"bytes"
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func loadSecondPhysicalApp1104(t *testing.T, f *FrontEnd, path string) *ui.App {
	t.Helper()
	app := f.App("1104-second-physical")
	store, original := SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: filepath.Dir(path)}
	save, list, load := f.SaveSeams(store, original, nil)
	wantToken := filepath.Base(path)
	var rows []ui.SaveEntry
	loadedToken := ""
	app.SetSaveSeams(save, func() []ui.SaveEntry {
		rows = list()
		return rows
	}, func(token string) (ui.MapOpener, bool, error) {
		if token != wantToken {
			t.Fatalf("App LOAD token %q, want%q", token, wantToken)
		}
		loadedToken = token
		return load(token)
	})
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	selected := -1
	for i, row := range rows {
		if row.Name == wantToken {
			if selected >= 0 {
				t.Fatal("duplicate exact LOAD token", wantToken)
			}
			selected = i
		}
	}
	if selected < 0 {
		t.Fatal("missing load entry", path)
	}
	shown := app.HeadlessRows()
	if len(shown) != len(rows) {
		t.Fatal("App LOAD changed the production row population")
	}
	for i, row := range rows {
		if shown[i].Text != row.Label || !shown[i].Choosable {
			t.Fatal("App LOAD changed a production row", i, row)
		}
	}
	for i := 0; i < selected; i++ {
		if err := app.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessKey("enter"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("App LOAD", app.Screen(), err)
	}
	if loadedToken != wantToken {
		t.Fatalf("App did not LOAD exact token %q: got%q", wantToken, loadedToken)
	}
	return app
}

func TestSecondPhysicalProcess1104(t *testing.T) {
	f := releaseFront(t)
	path := os.Getenv("AGAINROM_1104_INPUT")
	if path == "" {
		return
	}
	loadSecondPhysicalApp1104(t, f, path)
	h, ok := trainingPartyMember(t, f, "hero").OriginalHumanState()
	if !ok || h.Attack.SecondBase != 20 || h.Attack.SecondSpread != 7 || h.Modifier.Attack.SecondBase != 20 || h.Modifier.Attack.SecondSpread != 7 {
		t.Fatal("fresh App LOAD pair")
	}
	// This opens the real mission and verifies current SAV continuation.
	trainedMission1099(t, f)
	for i, p := range f.live.mission.party {
		if p.ID == "hero" {
			e, _ := f.live.entity(f.live.mission.ids[i])
			if e.SecondBase != 20 || e.SecondSpread != 7 {
				t.Fatal("mission pair")
			}
		}
	}
}

func TestReleaseSecondPhysicalSyntheticCityAppRoute(t *testing.T) {
	f := releaseFront(t)
	sourcePath, source := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	synthetic := secondPhysicalCity1104(t, source, [2]byte{91, 3}, [2]byte{20, 7})
	input := filepath.Join(t.TempDir(), "synthetic.sav")
	if err := WriteConvertedSave(input, synthetic, os.Getenv("AGAINROM_ASSETS")); err != nil {
		t.Fatal(err)
	}
	app := loadSecondPhysicalApp1104(t, f, input)
	h, ok := trainingPartyMember(t, f, "hero").OriginalHumanState()
	if !ok || h.Attack.SecondBase != 91 || h.Attack.SecondSpread != 3 {
		t.Fatal("synthetic current pair not loaded")
	}
	slot := 0
	for i := 1; i <= 5; i++ {
		if h.Base.Skill[i] == 0 {
			slot = i
			break
		}
	}
	if slot == 0 {
		t.Fatal("no source zero-base school slot")
	}
	gold := f.Town.Gold()
	s := f.TownScreen().(*townScreen)
	s.room, s.schoolCell = roomSchool, slot-1
	for i, p := range f.Carried {
		if p.ID == "hero" {
			s.shopMember = i
		}
	}
	if err := app.HeadlessActivate(f.Words.SchoolTrain); err != nil {
		t.Fatal(err)
	}
	h, ok = trainingPartyMember(t, f, "hero").OriginalHumanState()
	if !ok || f.Town.Gold() != gold-200 || h.Attack.SecondBase != 20 || h.Attack.SecondSpread != 7 {
		t.Fatal("App Train refused or failed", app.HeadlessMessage())
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatal("ordinary SAVE not SAV", entries, err, app.HeadlessMessage())
	}
	path := filepath.Join(store.Dir, entries[0].Name)
	raw, err := ReadSaveFile(path)
	if err != nil || !bytes.HasPrefix(raw, []byte(sav.Magic)) {
		t.Fatal("not Asg&", err)
	}
	checkSecondPhysicalWire1104(t, raw)
	for _, in := range []string{path} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSecondPhysicalProcess1104$", "-test.v")
		cmd.Env = append(os.Environ(), "AGAINROM_1104_INPUT="+in)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fresh App LOAD %s: %v\n%s", filepath.Ext(in), err, out)
		}
	}
	if again, err := os.ReadFile(sourcePath); err != nil || !bytes.Equal(again, source) {
		t.Fatal("lawful source changed")
	}
	t.Logf("SYNTHETIC scratch city from lawful SHA=%x; current91/3 modifier20/7 -> Train20/7; gold%d->%d; App LOAD/Train/ordinary SAVE Asg& bytes%d SHA=%x; fresh SAV App LOAD -> mission30 -> current continuation", sha256.Sum256(source), gold, gold-200, len(raw), sha256.Sum256(raw))
}

func checkSecondPhysicalWire1104(t *testing.T, raw []byte) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range p.Roster() {
		if !c.Hero {
			continue
		}
		h, err := p.Human(c.Identity)
		if err != nil {
			t.Fatal(err)
		}
		if h.Fields.Attack[17] != 20 || h.Fields.Attack[18] != 7 || h.Fields.Modifier[35] != 20 || h.Fields.Modifier[36] != 7 {
			t.Fatal("SAV wire lost pair")
		}
		return
	}
	t.Fatal("no hero on wire")
}
