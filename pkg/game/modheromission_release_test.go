package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
	"againrom/pkg/modrt"
)

// TestReleaseModMissionAfterAWinIsTheStartingHerosMap proves, on the lawful
// install, that mission 20's win opens the hero-mission fixture's mission of
// the starting hero the generator made: each of its three maps is the
// install's mission 10 map with its own first altitude byte, so the built map
// names the branch. A mission SAVE and a cold LOAD inside each branch build
// the same map again.
func TestReleaseModMissionAfterAWinIsTheStartingHerosMap(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := guestMods(t, guestModID)
	lang := modrt.LanguageFor(BaseID(InspectInstall(os.Getenv("AGAINROM_ASSETS"))))
	raw, err := os.ReadFile(assembledMap(filepath.Join(dir, guestModID), lang, 10))
	if err != nil {
		t.Fatal(err)
	}
	m, err := alm.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	plane := m.Altitudes
	at := bytes.Index(raw, plane)
	if at < 0 || bytes.Index(raw[at+1:], plane) >= 0 {
		t.Fatal("the altitude plane is not found once in the map")
	}
	root := filepath.Join(dir, "hero-mission")
	if err := os.CopyFS(root, os.DirFS(filepath.Join("testdata", "hero-mission"))); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "maps"), 0o755); err != nil {
		t.Fatal(err)
	}
	flips := map[string]byte{"fighter": 1, "mage_male": 2, "mage_female": 3}
	for key, flip := range flips {
		b := bytes.Clone(raw)
		b[at] ^= flip
		if err := os.WriteFile(filepath.Join(root, "maps", key+".alm"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, h := range []struct {
		choices []int
		key     string
		number  int
	}{
		{[]int{0, 0, 0}, "fighter", 25}, {[]int{1, 0, 0}, "fighter", 25},
		{[]int{0, 1, 0}, "mage_male", 27}, {[]int{1, 1, 0}, "mage_female", 28},
	} {
		built := func(f *FrontEnd) {
			t.Helper()
			s := f.live.mission.state
			if s.Address != "scenario/"+strconv.Itoa(h.number)+".alm" || s.Map.Altitudes[0] != plane[0]^flips[h.key] {
				t.Fatalf("hero %v: the mission was built from %s with altitude %d", h.choices, s.Address, s.Map.Altitudes[0])
			}
		}
		f := guestFront(t, dir, "hero-mission")
		app, next := guestRoute(t, f, h.choices)
		if next != h.number || f.Town.guest == nil || f.Town.guest.mission.Key != h.key {
			t.Fatalf("hero %v: mission 20's win answered %d, want %d", h.choices, next, h.number)
		}
		if err := app.OpenMission(f.MissionOpener(next)); err != nil {
			t.Fatal(err)
		}
		built(f)
		saved := guestMissionSave(t, f)
		g := guestFront(t, dir, "hero-mission")
		open, town, err := g.RestoreOriginal(saved)
		if err != nil || town {
			t.Fatalf("hero %v: cold LOAD of the mission: town %v %v", h.choices, town, err)
		}
		if err := g.App("hero mission load").OpenMission(open); err != nil {
			t.Fatal(err)
		}
		if !g.Town.guest.playing(h.number) || g.Town.guest.mission.Key != h.key || !bytes.Equal(g.Town.guest.stash, f.Town.guest.stash) {
			t.Fatalf("hero %v: the cold LOAD plays %q", h.choices, g.Town.guest.mission.Key)
		}
		built(g)
	}
}
