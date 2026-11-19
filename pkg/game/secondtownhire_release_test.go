package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func innHireSeats(f *FrontEnd) map[int]bool {
	seats, hires := f.Town.second.innSeats()
	out := map[int]bool{}
	for _, s := range seats[:hires] {
		out[s.npc] = s.high
	}
	return out
}

// The owner's Kaarg file keeps Iglez named and hires 1 and 2 through a resave.
func TestReleaseSecondTownFileKeepsTheHiredCompanion(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(secondCompanionCorpus, "rom2-owner-kaarg-0051", "game0016.sav"))
	if err != nil {
		t.Skipf("no owner file: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game0016.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f, app := secondTownCold(t, dir, "game0016.sav")
	want := companionOf(t, "the LOAD", f, 22)
	if len(f.Table.NPCNames) < 22 || want.Name == "" || want.Name != f.Table.NPCNames[21] || want.ID != "npc:22" ||
		want.Class != 5 || want.Hero.Body != 41 || want.Hero.Skill[1] != 25 {
		t.Fatalf("the LOAD lists the companion as %+v", want)
	}
	hires := map[int]bool{1: true, 2: true}
	if got := innHireSeats(f); !reflect.DeepEqual(got, hires) {
		t.Fatalf("the inn shows hire seats %v, want %v", got, hires)
	}
	out := t.TempDir()
	secondTownNamedSave(t, f, app, out, "Kaarg hired")
	cold, _ := secondTownCold(t, out, secondSaveFile(t, out, "Kaarg hired"))
	if got := companionOf(t, "the cold LOAD", cold, 22); !reflect.DeepEqual(got, want) {
		t.Fatalf("the cold LOAD of the resave lists\n%+v\nwant\n%+v", got, want)
	}
	if got := innHireSeats(cold); !reflect.DeepEqual(got, hires) {
		t.Fatalf("after the resave the inn shows hire seats %v, want %v", got, hires)
	}
}
