package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
)

// secondFileHeroMainSkill is the main skill the hero's record of a
// second-game file holds.
func secondFileHeroMainSkill(t *testing.T, raw []byte) (int32, [6]uint16) {
	t.Helper()
	sf, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	chars, err := sf.Party()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range chars {
		if c.Hero {
			return int32(c.MainSkill), c.SkillLevels
		}
	}
	t.Fatal("the file holds no hero")
	return 0, [6]uint16{}
}

// secondLoadedHeroMainSkill is the loaded hero's main skill: his actor's in
// a mission, his party member's in a town.
func secondLoadedHeroMainSkill(t *testing.T, f *FrontEnd) int32 {
	t.Helper()
	if mw := f.live; mw != nil {
		e, ok := mw.world.Entity(mw.mission.ids[0])
		if !ok || !mw.mission.party[0].StartingHero {
			t.Fatal("the mission holds no hero")
		}
		if e.MainSkill != mw.mission.party[0].MainSkill {
			t.Errorf("the hero's actor holds main skill %d, his party member %d", e.MainSkill, mw.mission.party[0].MainSkill)
		}
		return e.MainSkill
	}
	return nativeCityHeroOf(f.NextParty()).MainSkill
}

// The owner's second-game file game0005, read-only from the save corpus:
// its LOAD gives the hero the main skill his record holds, our SAVE writes
// it, and a cold LOAD of that SAVE gives it again (R2-ENGINE-320).
func TestReleaseSecondOriginalFileKeepsTheHerosMainSkill(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(secondCompanionCorpus, "rom2-gog-original", "game0005.sav"))
	if err != nil {
		t.Skipf("no corpus file: %v", err)
	}
	want, levels := secondFileHeroMainSkill(t, raw)
	t.Logf("game0005: main skill %d, skill levels %v", want, levels)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0005.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := secondGameFront(t)
	app := f.App("second-main-skill")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	_, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	groundAppLoad(t, app, list, "game0005.sav")
	if got := secondLoadedHeroMainSkill(t, f); got != want {
		t.Fatalf("the loaded hero's main skill %d, want the file's %d", got, want)
	}
	s, label, err := f.Snapshot(f.live != nil)
	if err != nil {
		t.Fatal(err)
	}
	ours, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := secondFileHeroMainSkill(t, ours); got != want {
		t.Fatalf("our SAVE's hero holds main skill %d, want %d", got, want)
	}
	again := t.TempDir()
	if err := os.WriteFile(filepath.Join(again, "game0006.sav"), ours, 0o600); err != nil {
		t.Fatal(err)
	}
	cold := secondGameFront(t)
	capp := cold.App("second-main-skill-cold")
	capp.Layout(1024, 768)
	cold.ConfigureSaveSeams(capp, SaveStore{Dir: again}, OriginalStore{}, nil)
	_, clist, _ := cold.SaveSeams(SaveStore{Dir: again}, OriginalStore{}, nil)
	groundAppLoad(t, capp, clist, "game0006.sav")
	if got := secondLoadedHeroMainSkill(t, cold); got != want {
		t.Fatalf("the cold LOAD's hero holds main skill %d, want %d", got, want)
	}
}
