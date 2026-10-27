package game

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

// secondOriginalSample is what a LOAD of a second-game town shows: the
// hero's measured fields and book (R2-SESSION-236, R2-SESSION-237), the
// money, and the campaign's place and bag.
type secondOriginalSample struct {
	Name                         string
	Body, Reaction, Mind, Spirit int32
	Skill                        [data.SkillSlots]int32
	Experience                   int32
	HP, MaxHP, Mana, MaxMana     int32
	Worn                         []uint16
	Book                         uint32
	Gold                         int
	State                        SecondCampaignState
}

func secondOriginalSampleOf(t *testing.T, f *FrontEnd) secondOriginalSample {
	t.Helper()
	party := f.NextParty()
	if len(party) != 1 || party[0].Saved == nil {
		t.Fatalf("party of %d, want the hero with his saved pools", len(party))
	}
	m := party[0]
	h := m.Hero
	s := secondOriginalSample{Name: m.Name, Body: h.Body, Reaction: h.Reaction, Mind: h.Mind, Spirit: h.Spirit,
		Skill: h.Skill, HP: m.Saved.HP, MaxHP: m.Saved.MaxHP, Mana: m.Saved.Mana, MaxMana: m.Saved.MaxMana,
		Book: m.KnownSpells, Gold: f.Town.Gold()}
	if m.Carry != nil {
		for _, v := range m.Carry.SkillXP {
			s.Experience += v
		}
	}
	for _, it := range m.WornItems {
		if it.Code != 0 {
			s.Worn = append(s.Worn, it.Code)
		}
	}
	state, ok := f.Town.SecondState()
	if !ok {
		t.Fatal("the town has no second campaign")
	}
	s.State = state
	return s
}

// secondFixtureSample is secondFixtureHero's town as a LOAD must show it:
// experience is the five experiences' sum, without the main skill
// (R2-ENGINE-320, DIV-4154); the town opens at its square (DIV-4153).
func secondFixtureSample() secondOriginalSample {
	return secondOriginalSample{Name: "Gildarius", Body: 19, Reaction: 15, Mind: 43, Spirit: 25,
		Skill: [data.SkillSlots]int32{0, 0, 0, 0, 20, 10}, Experience: 7320,
		HP: 29, MaxHP: 29, Mana: 75, MaxMana: 75, Worn: []uint16{0x812d, 0xf72d},
		Book: 1<<24 | 1<<27, Gold: 44,
		State: SecondCampaignState{Kind: placeTown, ID: 1, Room: int(secondTownSquare),
			Available: [][2]int{{placeTown, 1}, {placeMission, 10}}, Bag: []uint16{0x812d}}}
}

// LOAD of a synthetic Bsg& town from the Load page opens the town with the
// hero, his book, the money and the bag; our SAVE writes our document, its
// cold LOAD shows the same, and the party leaves for mission 10.
func TestReleaseSecondOriginalTownLoadRoundTrip(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "game0007.sav"), secondFixtureTown("Synthetic town", secondFixtureHero(), 44), 0o644); err != nil {
		t.Fatal(err)
	}
	f, app := secondTownCold(t, out, "game0007.sav")
	want := secondFixtureSample()
	if got := secondOriginalSampleOf(t, f); !reflect.DeepEqual(got, want) {
		t.Fatalf("LOAD shows\n%+v\nwant\n%+v", got, want)
	}
	raw := secondTownNamedSave(t, f, app, out, "Our town")
	if p, ok := sav.ProfileOf(raw); !ok || p != sav.FirstGame {
		t.Fatal("our SAVE did not write the engine's document")
	}
	secondTownShape(t, raw)
	cold, capp := secondTownCold(t, out, secondSaveFile(t, out, "Our town"))
	if got := secondOriginalSampleOf(t, cold); !reflect.DeepEqual(got, want) {
		t.Fatalf("cold LOAD of our SAVE shows\n%+v\nwant\n%+v", got, want)
	}
	for _, target := range []string{"GATES", "mission 10"} {
		if err := capp.HeadlessActivate(target); err != nil {
			t.Fatal(target, err)
		}
	}
	if capp.Screen() != ui.ScreenMap || cold.liveMission != 10 {
		t.Fatal("the loaded party did not leave for mission 10", capp.Screen(), cold.liveMission)
	}
	for range 20 {
		if err := capp.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, capp, cold.live)
}

// One attribute changed in the file is the loaded hero's.
func TestReleaseSecondOriginalTownLoadFollowsTheFile(t *testing.T) {
	out := t.TempDir()
	hero := secondFixtureHero()
	hero.stats[sav.StatMind]++
	if err := os.WriteFile(filepath.Join(out, "game0008.sav"), secondFixtureTown("Changed town", hero, 44), 0o644); err != nil {
		t.Fatal(err)
	}
	f, _ := secondTownCold(t, out, "game0008.sav")
	want := secondFixtureSample()
	want.Mind++
	if got := secondOriginalSampleOf(t, f); !reflect.DeepEqual(got, want) {
		t.Fatalf("LOAD shows\n%+v\nwant\n%+v", got, want)
	}
}

// A mission-like Bsg& file with a Token after its Sacks is listed; its LOAD
// refuses before adoption and the front end stays as it was (DIV-4172).
func TestReleaseSecondOriginalMissionLoadIsRefused(t *testing.T) {
	out := t.TempDir()
	hero := secondMissionFixtureHero()
	raw := labelledContainer(sav.SecondGame.Magic, "Synthetic mission", secondSavedWorld(10, 44, &hero, true, 1),
		append(secondMissionStore(4, 0, -1), secondSavedTail(0, [2]uint32{10, placeMission})...))
	if err := os.WriteFile(filepath.Join(out, "game0009.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := secondGameFront(t)
	_, list, _ := f.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	listed := false
	for _, row := range list() {
		listed = listed || row.Name == localOriginalSaveToken("game0009.sav") && row.Label == "Synthetic mission"
	}
	if !listed {
		t.Fatal("a mission-like Bsg& file is not listed")
	}
	if _, _, err := f.RestoreOriginal(raw); err == nil || !strings.Contains(err.Error(), "after its Sacks") {
		t.Fatalf("a mission-like Bsg& file LOAD: %v, want the extra Token refusal", err)
	}
	if f.live != nil || f.liveMission != 0 {
		t.Fatal("the refused LOAD changed the front end")
	}
}
