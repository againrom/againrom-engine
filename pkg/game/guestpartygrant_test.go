package game

import (
	"math/bits"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/render/terrain"
)

const grantedPartyMission = 25

// grantedPartyFront is a first-game front end whose mod declares mission 25
// after mission 20 with the party given, over a table that holds three
// spells and a mod's character who is no mage.
func grantedPartyFront(t *testing.T, party string) *FrontEnd {
	t.Helper()
	f := winRouteFirstFront(t, mod.MissionData{})
	f.Archives.Containers = missionArchive(t, 10, 20, 30)
	f.Tiles = &terrain.Tileset{}
	f.Table.Mods.Set = mod.Set{Base: "rom1-en", Mods: []mod.SetEntry{{ID: "flow", Version: "1", Digest: "synthetic"}}}
	humans := f.Table.Humans.(dbCollection)
	humans[27] = dbEntry{name: "PC_Sword", params: humansParams(30, 0, 2)}
	humans[27].params[0x18], humans[27].params[0x10] = 27, 22
	f.Table.Mods.Catalog = append(f.Table.Mods.Catalog, mod.Character{Key: "sword", Row: "PC_Sword"})
	spell := make([]int32, 19)
	spell[1], spell[2], spell[4], spell[6] = 3, 1, 1, 7
	spell[16], spell[17] = 15, 18
	f.Table.Spells = dbCollection{{}, {name: "Fire Arrow", params: spell}, {name: "Fire Ball", params: spell}, {name: "Fire Wall", params: spell}}
	src := "[[mission]]\nkey = \"side\"\nnumber = 25\nmap = \"maps/side.alm\"\nafter = 20\nparty = " + party + "\n"
	d, err := mod.ParseMissions("flow", mod.MissionsFile, []byte(src), func(string) (string, bool) { return "", false }, "en")
	if err != nil {
		t.Fatal(err)
	}
	d.Missions[0].MapBytes = synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "side"})
	if _, err := f.SetModMissions(d); err != nil {
		t.Fatal(err)
	}
	return f
}

// grantedPartyStart starts the mission by its number, as a start that names
// the mission does after the hero is made.
func grantedPartyStart(t *testing.T, f *FrontEnd) {
	t.Helper()
	open, err := f.prepareNewGameWith(grantedPartyMission, 0, f.Carried)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	if f.liveMission != grantedPartyMission {
		t.Fatalf("plays %d", f.liveMission)
	}
}

func grantedPartyIDs(party []mapload.PartyMember) []string {
	var ids []string
	for _, p := range party {
		ids = append(ids, p.ID)
	}
	return ids
}

// A mission started by its number plays the party its party array declares:
// the hero and the two companions it names, each with what his row asks, and
// SAVE and LOAD keep that party.
func TestMissionStartedByNumberPlaysItsDeclaredParty(t *testing.T) {
	const party = `[{ hero = true, skills = 100 }, { character = "npc22", skills = 100, spells = "all" }, { character = "sword", spells = "all" }]`
	f := grantedPartyFront(t, party)
	hero := f.Carried[0]
	plain := grantedPartyFront(t, `[{ hero = true }, { character = "npc22" }, { character = "sword" }]`)
	grantedPartyStart(t, plain)
	grantedPartyStart(t, f)
	check := func(f *FrontEnd, stage string) {
		t.Helper()
		got := grantedPartyIDs(f.liveParty)
		if len(got) != 3 || got[0] != hero.ID || got[1] != "npc:22" || got[2] != "char:sword" {
			t.Fatalf("%s: the party is %v", stage, got)
		}
		limit := mapload.TableRules(f.Table).SkillCap()
		for i, p := range f.liveParty {
			own := plain.liveParty[i]
			e, ok := f.live.world.Entity(f.live.mission.ids[i])
			if !ok {
				t.Fatalf("%s: member %d has no entity", stage, i)
			}
			for slot := 1; slot < data.SkillSlots; slot++ {
				want := own.Hero.Skill[slot]
				if i < 2 {
					want = limit
				}
				if p.Hero.Skill[slot] != want || i < 2 && e.Skill[slot] < limit {
					t.Fatalf("%s: member %d skill %d is %d (entity %d), want %d", stage, i, slot, p.Hero.Skill[slot], e.Skill[slot], want)
				}
			}
			switch i {
			case 0:
				if p.KnownSpells != own.KnownSpells {
					t.Fatalf("%s: the hero's spells changed without the key", stage)
				}
			case 1:
				if !p.Mage || bits.OnesCount32(p.KnownSpells) != 3 || e.KnownSpells != p.KnownSpells {
					t.Fatalf("%s: the mage knows %#x (entity %#x), want the table's three spells", stage, p.KnownSpells, e.KnownSpells)
				}
				if bits.OnesCount32(own.KnownSpells) >= 3 {
					t.Fatalf("%s: the mage knows every spell without the key", stage)
				}
			case 2:
				if p.Mage || p.KnownSpells != own.KnownSpells || p.Book != own.Book {
					t.Fatalf("%s: the member who cannot cast took spells: %#x", stage, p.KnownSpells)
				}
			}
		}
	}
	check(f, "start")
	if limit := mapload.TableRules(f.Table).SkillCap(); limit != 100 || hero.Hero.Skill[1] >= limit {
		t.Fatalf("the fixture's cap is %d and the hero's skill %d", limit, hero.Hero.Skill[1])
	}
	saved := guestMissionSave(t, f)
	g := grantedPartyFront(t, party)
	open, town, err := g.RestoreOriginal(saved)
	if err != nil || town {
		t.Fatalf("LOAD: %t %v", town, err)
	}
	if _, _, _, _, _, _, _, _, _, _, err = open(); err != nil {
		t.Fatal(err)
	}
	check(g, "after LOAD")
	winMissionFlow(t, g, 0)
	if len(g.Carried) != 1 || g.Carried[0].ID != hero.ID || g.Carried[0].Hero.Skill != hero.Hero.Skill || g.Town.guest != nil {
		t.Fatalf("after the win the town holds %v", grantedPartyIDs(g.Carried))
	}
}

// A mission reached by the win before it still opens with its party, and a
// party that leaves the hero out plays without him.
func TestDeclaredPartyWithoutTheHero(t *testing.T) {
	f := grantedPartyFront(t, `[{ character = "npc22" }, { character = "sword" }]`)
	grantedPartyStart(t, f)
	if got := grantedPartyIDs(f.liveParty); len(got) != 2 || got[0] != "npc:22" || got[1] != "char:sword" {
		t.Fatalf("the party is %v", got)
	}
	if !f.liveParty[0].StartingHero || f.liveParty[1].StartingHero {
		t.Fatal("the first member does not lead")
	}
}

// A mission started by its number plays for the heroes its class and sex
// name: a hero of another kind is refused in a line and nothing starts.
func TestMissionStartedByNumberRefusesAHeroItDoesNotPlayFor(t *testing.T) {
	words := map[bool]string{false: herokind.Fighter, true: herokind.Mage}
	for _, own := range []bool{true, false} {
		probe := grantedPartyFront(t, `[{ hero = true }]`)
		hero := mapload.StartingHeroKind(mapload.HeroHold{Party: probe.Carried})
		f := grantedPartyFront(t, "[{ hero = true }]\nclass = \""+words[hero.Mage() == own]+"\"")
		open, err := f.prepareNewGameWith(grantedPartyMission, 0, f.Carried)
		if own {
			if err != nil {
				t.Fatalf("a mission for the hero's own class is refused: %v", err)
			}
			if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil || f.liveMission != grantedPartyMission {
				t.Fatalf("plays %d: %v", f.liveMission, err)
			}
			continue
		}
		want := "mission 25 plays for a " + words[!hero.Mage()] + ", and this hero is a " + hero.String()
		if err == nil || !strings.Contains(err.Error(), want) || f.liveMission == grantedPartyMission {
			t.Fatalf("a mission for the other class: %v, want %q", err, want)
		}
	}
}
