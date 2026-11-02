package mapload_test

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

// Every choice the map tier makes by the starting hero, for the four heroes
// marked in a party behind a companion of the other class and sex and
// unmarked at its head: the composed npc21..24 person a standalone map
// places, the tiered row a campaign mission places, the member a town grant
// builds, the row a companion of the hero is bound to, and the placements an
// ordinal scan lists, with every npcnames line and with only lines 21 and 23.
func TestEveryChoiceByTheStartingHeroReadsTheSameHero(t *testing.T) {
	humans := defCollection{
		{},
		{name: "PC_Danath", params: withServerID(composedHumanRow(3, 5, 0), 26), strings: []string{"Sword"}},
		{name: "PC_Naira", params: withServerID(composedHumanRow(14, 1, 1), 27), strings: []string{"Bow"}},
		{name: "PC_Fergard", params: withServerID(composedHumanRow(24, 3, 0), 28)},
		{name: "PC_Reniesta", params: withServerID(composedHumanRow(24, 1, 1), 29)},
	}
	names := make([]string, 24)
	for i := range names {
		names[i] = fmt.Sprintf("line %d", i+1)
	}
	sparse := slices.Clone(names)
	sparse[21], sparse[23] = "", ""
	table := func(lines []string) *mapload.Table {
		return &mapload.Table{Humans: humans, NPC: composedRegistry(t), NPCNames: lines,
			Shapes: identityScale(), Materials: identityScale(), Weapons: swordAndBow()}
	}
	full, few := table(names), table(sparse)
	var got []string
	for _, hero := range []struct{ mage, female bool }{{false, false}, {false, true}, {true, false}, {true, true}} {
		dir := data.FigureDirFor(hero.mage, hero.female)
		other := data.FigureDirFor(!hero.mage, !hero.female)
		lead := mapload.PartyMember{ID: "hero", StartingHero: true, PlayerCharacter: true, Mage: hero.mage, FigureDir: string(dir)}
		guest := mapload.PartyMember{ID: "npc:9", PlayerCharacter: true, Mage: !hero.mage, FigureDir: string(other)}
		unmarked := lead
		unmarked.StartingHero = false
		for order, party := range map[string][]mapload.PartyMember{"marked": {guest, lead}, "first": {unmarked, guest}} {
			for npc := uint16(21); npc <= 24; npc++ {
				m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{X: 0x0c80, Y: 0x0c80, ClassID: 4, ClassSubID: npc, Flags: 1, Owner: 1}}}
				line := fmt.Sprintf("%s %s npc%d:", dir, order, npc)
				for _, mission := range []int{0, 30} {
					if _, st, err := mapload.StartCampaignMission(m, full, mapload.DifficultyNormal, party, mission); err != nil {
						line += fmt.Sprintf(" start%d=%v", mission, err)
					} else {
						line += fmt.Sprintf(" start%d=%s", mission, st.Roster[0].Name)
					}
				}
				member, ok := mapload.CampaignNPCMember(full, int32(npc), 30, party)
				line += fmt.Sprintf(" member=%s/%v", member.Name, ok)
				line += fmt.Sprintf(" row=%d", mapload.PartyHumanRow(mapload.PartyMember{CompanionNPC: int(npc)}, mapload.StartingHero(party), full))
				line += fmt.Sprintf(" placed=%v sparse=%v", mapload.PlacedHeroes(m, full, party, 30), mapload.PlacedHeroes(m, few, party, 30))
				got = append(got, line)
			}
		}
	}
	slices.Sort(got)
	want := []string{
		"ffighter first npc21: start0=PC_Naira start30=PC_Naira member=PC_Naira/true row=2 placed=[{0 0 {true false 1}}] sparse=[]",
		"ffighter first npc22: start0=PC_Fergard start30=PC_Fergard member=PC_Fergard/true row=3 placed=[{0 0 {false false 3}}] sparse=[{0 0 {false false 3}}]",
		"ffighter first npc23: start0=PC_Danath start30=PC_Danath member=PC_Danath/true row=1 placed=[{0 0 {false false 5}}] sparse=[{0 0 {false false 5}}]",
		"ffighter first npc24: start0=PC_Reniesta start30=PC_Reniesta member=PC_Reniesta/true row=4 placed=[{0 0 {true false 1}}] sparse=[]",
		"ffighter marked npc21: start0=PC_Naira start30=PC_Naira member=PC_Naira/true row=2 placed=[{0 0 {true false 1}}] sparse=[]",
		"ffighter marked npc22: start0=PC_Fergard start30=PC_Fergard member=PC_Fergard/true row=3 placed=[{0 0 {false false 3}}] sparse=[{0 0 {false false 3}}]",
		"ffighter marked npc23: start0=PC_Danath start30=PC_Danath member=PC_Danath/true row=1 placed=[{0 0 {false false 5}}] sparse=[{0 0 {false false 5}}]",
		"ffighter marked npc24: start0=PC_Reniesta start30=PC_Reniesta member=PC_Reniesta/true row=4 placed=[{0 0 {true false 1}}] sparse=[]",
		"fmage first npc21: start0=PC_Reniesta start30=PC_Reniesta member=PC_Reniesta/true row=4 placed=[{0 0 {true false 1}}] sparse=[]",
		"fmage first npc22: start0=PC_Fergard start30=PC_Fergard member=PC_Fergard/true row=3 placed=[{0 0 {false false 3}}] sparse=[{0 0 {false false 3}}]",
		"fmage first npc23: start0=PC_Danath start30=PC_Danath member=PC_Danath/true row=1 placed=[{0 0 {false false 5}}] sparse=[{0 0 {false false 5}}]",
		"fmage first npc24: start0=PC_Naira start30=PC_Naira member=PC_Naira/true row=2 placed=[{0 0 {true false 1}}] sparse=[]",
		"fmage marked npc21: start0=PC_Reniesta start30=PC_Reniesta member=PC_Reniesta/true row=4 placed=[{0 0 {true false 1}}] sparse=[]",
		"fmage marked npc22: start0=PC_Fergard start30=PC_Fergard member=PC_Fergard/true row=3 placed=[{0 0 {false false 3}}] sparse=[{0 0 {false false 3}}]",
		"fmage marked npc23: start0=PC_Danath start30=PC_Danath member=PC_Danath/true row=1 placed=[{0 0 {false false 5}}] sparse=[{0 0 {false false 5}}]",
		"fmage marked npc24: start0=PC_Naira start30=PC_Naira member=PC_Naira/true row=2 placed=[{0 0 {true false 1}}] sparse=[]",
		"mfighter first npc21: start0=PC_Danath start30=PC_Danath member=PC_Danath/true row=1 placed=[{0 0 {false false 5}}] sparse=[{0 0 {false false 5}}]",
		"mfighter first npc22: start0=PC_Reniesta start30=PC_Reniesta member=PC_Reniesta/true row=4 placed=[{0 0 {true false 1}}] sparse=[]",
		"mfighter first npc23: start0=PC_Naira start30=PC_Naira member=PC_Naira/true row=2 placed=[{0 0 {true false 1}}] sparse=[]",
		"mfighter first npc24: start0=PC_Fergard start30=PC_Fergard member=PC_Fergard/true row=3 placed=[{0 0 {false false 3}}] sparse=[{0 0 {false false 3}}]",
		"mfighter marked npc21: start0=PC_Danath start30=PC_Danath member=PC_Danath/true row=1 placed=[{0 0 {false false 5}}] sparse=[{0 0 {false false 5}}]",
		"mfighter marked npc22: start0=PC_Reniesta start30=PC_Reniesta member=PC_Reniesta/true row=4 placed=[{0 0 {true false 1}}] sparse=[]",
		"mfighter marked npc23: start0=PC_Naira start30=PC_Naira member=PC_Naira/true row=2 placed=[{0 0 {true false 1}}] sparse=[]",
		"mfighter marked npc24: start0=PC_Fergard start30=PC_Fergard member=PC_Fergard/true row=3 placed=[{0 0 {false false 3}}] sparse=[{0 0 {false false 3}}]",
		"mmage first npc21: start0=PC_Fergard start30=PC_Fergard member=PC_Fergard/true row=3 placed=[{0 0 {false false 3}}] sparse=[{0 0 {false false 3}}]",
		"mmage first npc22: start0=PC_Reniesta start30=PC_Reniesta member=PC_Reniesta/true row=4 placed=[{0 0 {true false 1}}] sparse=[]",
		"mmage first npc23: start0=PC_Naira start30=PC_Naira member=PC_Naira/true row=2 placed=[{0 0 {true false 1}}] sparse=[]",
		"mmage first npc24: start0=PC_Danath start30=PC_Danath member=PC_Danath/true row=1 placed=[{0 0 {false false 5}}] sparse=[{0 0 {false false 5}}]",
		"mmage marked npc21: start0=PC_Fergard start30=PC_Fergard member=PC_Fergard/true row=3 placed=[{0 0 {false false 3}}] sparse=[{0 0 {false false 3}}]",
		"mmage marked npc22: start0=PC_Reniesta start30=PC_Reniesta member=PC_Reniesta/true row=4 placed=[{0 0 {true false 1}}] sparse=[]",
		"mmage marked npc23: start0=PC_Naira start30=PC_Naira member=PC_Naira/true row=2 placed=[{0 0 {true false 1}}] sparse=[]",
		"mmage marked npc24: start0=PC_Danath start30=PC_Danath member=PC_Danath/true row=1 placed=[{0 0 {false false 5}}] sparse=[{0 0 {false false 5}}]",
	}
	if !slices.Equal(got, want) {
		for _, l := range got {
			t.Log(l)
		}
		t.Fatalf("%d choices differ from the recorded %d", len(got), len(want))
	}
}
