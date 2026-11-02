package game

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

// heroMissionMaps are the maps the hero-mission fixture names, one per
// mission, which the test writes beside the fixture's copy.
var heroMissionMaps = map[string][]byte{
	"fighter":     synth.ALM(synth.ALMOptions{Width: 4, Height: 4, Name: "fighter"}),
	"mage_male":   synth.ALM(synth.ALMOptions{Width: 4, Height: 6, Name: "mage_male"}),
	"mage_female": synth.ALM(synth.ALMOptions{Width: 6, Height: 4, Name: "mage_female"}),
}

// heroMissionMod is the hero-mission fixture with its maps, its missions
// file edited by edit, read through the mod loader on the first game.
func heroMissionMod(t *testing.T, edit func(string) string) (mod.MissionData, error) {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, "hero-mission")
	if err := os.CopyFS(root, os.DirFS(filepath.Join("testdata", "hero-mission"))); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "maps"), 0o755); err != nil {
		t.Fatal(err)
	}
	for key, b := range heroMissionMaps {
		if err := os.WriteFile(filepath.Join(root, "maps", key+".alm"), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(root, filepath.FromSlash(mod.MissionsFile))
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(edit(string(src))), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := mod.Resolve(dir, []string{"hero-mission"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := modrt.Load(entries, "rom1-en", nil, modrt.Options{})
	if err != nil {
		return mod.MissionData{}, err
	}
	return res.Missions, nil
}

// heroParty is a party led by a starting hero of kind k.
func heroParty(k herokind.Kind) []mapload.PartyMember {
	dir := map[herokind.Kind]data.FigureDir{herokind.MaleFighter: data.FigureDirManFighter, herokind.FemaleFighter: data.FigureDirWomanFighter,
		herokind.MaleMage: data.FigureDirManMage, herokind.FemaleMage: data.FigureDirWomanMage}[k]
	return []mapload.PartyMember{{ID: "hero", StartingHero: true, PlayerCharacter: true, Mage: k.Mage(), FigureDir: string(dir)}}
}

// One win of mission 20 opens the hero-mission fixture's map of the starting
// hero: the fighter's for either fighter, the male mage's and the female
// mage's. A SAV inside each branch and the town SAV after its win load the
// same branch.
func TestModMissionAfterAWinIsTheStartingHerosMap(t *testing.T) {
	missions, err := heroMissionMod(t, func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	f := guestFixtureFront(t)
	if _, err := f.SetModMissions(missions); err != nil {
		t.Fatal(err)
	}
	d := f.Table.Mods.Missions
	if bytes.Equal(heroMissionMaps["mage_male"], heroMissionMaps["mage_female"]) || bytes.Equal(heroMissionMaps["fighter"], heroMissionMaps["mage_male"]) {
		t.Fatal("the fixture maps are not three maps")
	}
	for k, key := range map[herokind.Kind]string{herokind.MaleFighter: "fighter", herokind.FemaleFighter: "fighter",
		herokind.MaleMage: "mage_male", herokind.FemaleMage: "mage_female"} {
		party := heroParty(k)
		if got := mapload.StartingHeroKind(mapload.HeroHold{Party: party}); got != k {
			t.Fatalf("party of a %v reads a %v", k, got)
		}
		m, ok := d.AfterWin(20, mapload.StartingHeroKind(mapload.HeroHold{Party: party}))
		if !ok || m.Key != key {
			t.Fatalf("a %v: %+v %v, want %q", k, m, ok, key)
		}
		served := func(n int) []byte {
			address, _ := MissionMap(n)
			b, err := f.Archives.Containers.ReadFile(address)
			if err != nil {
				t.Fatal(err)
			}
			return b
		}
		if !bytes.Equal(served(m.Number), heroMissionMaps[key]) {
			t.Fatalf("a %v: mission %d serves another map", k, m.Number)
		}
		member := mapload.PartyMember{ID: "npc:22", Name: "Guest", CompanionNPC: 22, StartingHero: true, PlayerCharacter: true}
		joined := member
		joined.StartingHero = false
		for name, g := range map[string]*guestMission{
			"inside the mission":  {mission: m, start: member, stash: []byte("town SAV of " + key), chain: d.Chain(m)},
			"in town after a win": {mission: m, start: member, member: &joined},
		} {
			raw, err := json.Marshal(currentSessionData{Guest: g.record()})
			if err != nil {
				t.Fatal(err)
			}
			var s currentSessionData
			if err := json.Unmarshal(raw, &s); err != nil {
				t.Fatal(err)
			}
			back, err := restoreGuest(&s, mapload.ModContext{Missions: d})
			if err != nil {
				t.Fatalf("a %v %s: %v", k, name, err)
			}
			if back.mission.Key != key || back.mission.Number != m.Number || !bytes.Equal(served(back.mission.Number), heroMissionMaps[key]) {
				t.Fatalf("a %v %s: LOAD took %q (mission %d)", k, name, back.mission.Key, back.mission.Number)
			}
			if g.stash != nil && (!back.playing(m.Number) || string(back.stash) != string(g.stash)) {
				t.Fatalf("a %v %s: the SAV does not play its branch with its stash", k, name)
			}
			if g.stash == nil {
				if got, ok := back.joining(30, 22); !ok || got.CompanionNPC != 22 {
					t.Fatalf("a %v %s: the guest of the branch does not join", k, name)
				}
			}
		}
	}
}

// The waiting party's starting hero, which a SAV of a chain carries, chooses
// the later link after LOAD as it did before.
func TestModMissionLaterLinkIsChosenByTheSavedHero(t *testing.T) {
	g, d := chainGuest()
	for _, k := range []herokind.Kind{herokind.MaleFighter, herokind.FemaleMage} {
		p := heroParty(k)[0]
		g.hero = &guestHero{Mage: p.Mage, FigureDir: p.FigureDir}
		raw, err := json.Marshal(currentSessionData{Guest: g.record()})
		if err != nil {
			t.Fatal(err)
		}
		var s currentSessionData
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		back, err := restoreGuest(&s, mapload.ModContext{Missions: d})
		if err != nil {
			t.Fatal(err)
		}
		if got := mapload.StartingHeroKind(mapload.HeroHold{Party: back.heroParty(heroParty(herokind.MaleFighter))}); got != k {
			t.Fatalf("the saved %v chooses as a %v", k, got)
		}
	}
}

// Missions after one win whose heroes overlap are refused through the mod
// loader with both named; the fixture's own three load.
func TestModMissionsWhoseHeroesOverlapAreRefusedByTheLoader(t *testing.T) {
	_, err := heroMissionMod(t, func(s string) string {
		return strings.Replace(s, "class   = \"mage\"\nsex     = \"female\"", "sex     = \"female\"", 1)
	})
	want := "a mission already plays after mission 20 for a female fighter at line"
	if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "data/missions.toml") {
		t.Fatalf("%v, want %q", err, want)
	}
}
