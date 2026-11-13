package mod

import (
	"strings"
	"testing"
)

const memberMission = "[[mission]]\nkey = \"side\"\nnumber = 25\nmap = \"a.alm\"\nafter = 20\n"

// The hero form and the two member keys parse, write back and parse again
// to the same members; a party that omits them reads as it did.
func TestPartyMemberHeroSkillsAndSpells(t *testing.T) {
	src := memberMission + "party = [{ hero = true, skills = 100, spells = \"all\" }, { character = \"npc22\", level = 2, spells = \"all\" }, { character = \"npc25\", skills = 100 }, { character = \"npc23\" }]\n"
	d, err := ParseMissions("m", MissionsFile, []byte(src), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	want := []GuestMember{
		{Hero: true, Skills: GuestSkills, Spells: GuestSpellsAll},
		{Character: "npc22", Level: 2, Spells: GuestSpellsAll},
		{Character: "npc25", Skills: GuestSkills},
		{Character: "npc23"},
	}
	same := func(got []GuestMember) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("party is %+v", got)
		}
		for i, g := range got {
			w := want[i]
			if g.Hero != w.Hero || g.Character != w.Character || g.Level != w.Level || g.Skills != w.Skills || g.Spells != w.Spells {
				t.Fatalf("member %d is %+v, want %+v", i+1, g, w)
			}
		}
	}
	same(d.Missions[0].Party)
	if id := d.Missions[0].Party[0].ID(); id != "hero" {
		t.Fatalf("the hero member is named %q", id)
	}
	written, err := WriteMissions(nil, nil, d.Missions)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{`{ hero = true, skills = 100, spells = "all" }`, `{ character = "npc22", level = 2, spells = "all" }`, `{ character = "npc23" }`} {
		if !strings.Contains(string(written), text) {
			t.Fatalf("the written file lacks %s:\n%s", text, written)
		}
	}
	again, err := ParseMissions("m", MissionsFile, written, nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	same(again.Missions[0].Party)
	same(CloneMission(again.Missions[0]).Party)
}

// A value the keys do not take, a hero that names a character and a second
// hero are refused with the file and the line.
func TestPartyMemberKeysAreChecked(t *testing.T) {
	for _, c := range []struct{ name, party, want string }{
		{"skills value", `[{ character = "npc22", skills = 50 }]`, `data/missions.toml:6: party member 1: skills must be 100`},
		{"skills kind", `[{ character = "npc22", skills = "100" }]`, `data/missions.toml:6: party member 1: skills must be 100`},
		{"spells value", `[{ character = "npc22", spells = "fire" }]`, `data/missions.toml:6: party member 1: spells must be "all"`},
		{"spells kind", `[{ character = "npc22", spells = true }]`, `data/missions.toml:6: party member 1: spells must be "all"`},
		{"hero false", `[{ hero = false }]`, `data/missions.toml:6: party member 1: hero must be true`},
		{"hero with a character", `[{ hero = true, character = "npc22" }]`, `data/missions.toml:6: party member 1 is the hero and takes no character`},
		{"hero with a level", `[{ hero = true, level = 2 }]`, `data/missions.toml:6: party member 1 is the hero and takes no level`},
		{"two heroes", `[{ hero = true }, { hero = true }]`, `data/missions.toml:6: party names the hero twice`},
		{"unknown key", `[{ character = "npc22", mana = 1 }]`, `unknown key "mana" (character, hero, level, weapon, shield, armour, skills, spells)`},
	} {
		_, err := ParseMissions("m", MissionsFile, []byte(memberMission+"party = "+c.party+"\n"), nil, "en")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %s", c.name, err, c.want)
		}
	}
}

// The hero enters a chain at its first link: a later link that names him
// after a link without him has no hero to take.
func TestPartyHeroEntersAChainAtItsFirstLink(t *testing.T) {
	first := "[[mission]]\nkey = \"first\"\nnumber = 25\nmap = \"a.alm\"\nafter = 20\nparty = [{ character = \"npc22\" }]\n"
	second := "[[mission]]\nkey = \"second\"\nnumber = 26\nmap = \"b.alm\"\nafter = 25\nparty = [{ hero = true }]\n"
	d, err := ParseMissions("m", MissionsFile, []byte(first+second), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = MergeMissions(d)
	if err == nil || !strings.Contains(err.Error(), "data/missions.toml:12") || !strings.Contains(err.Error(), "the hero") {
		t.Fatalf("a hero in a later link alone: %v", err)
	}
	both := strings.Replace(first, `{ character = "npc22" }`, `{ hero = true }, { character = "npc22" }`, 1) + second
	if d, err = ParseMissions("m", MissionsFile, []byte(both), nil, "en"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = MergeMissions(d); err != nil {
		t.Fatalf("a hero in both links: %v", err)
	}
}
