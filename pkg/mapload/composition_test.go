package mapload_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
)

func composedRegistry(t *testing.T) *data.NPCDefs {
	t.Helper()
	b := synth.Reg(0x11, []synth.RegNode{
		{Name: "npc21", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,Me,Start"},
			{Name: "DataBinID", Kind: 0x02, Int: 26},
		}},
		{Name: "npc22", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,Mage,!MySex,Start"},
			{Name: "DataBinID", Kind: 0x02, Int: 26},
		}},
		{Name: "npc23", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,!Mage,!MySex,Start"},
			{Name: "DataBinID", Kind: 0x02, Int: 26},
		}},
		{Name: "npc24", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,!MyClass,MySex,Start"},
			{Name: "DataBinID", Kind: 0x02, Int: 26},
		}},
	})
	r, err := reg.Parse(b)
	if err != nil {
		t.Fatalf("reg.Parse: %v", err)
	}
	return data.LoadNPCDefs(r)
}

func composedHumanRow(typeID, face, gender int32) []int32 {
	return humanRow(map[int]int32{
		slotHumanBody: 20, slotHumanReaction: 20, slotHumanHealth: 40,
		slotHumanType: typeID, 17: face, 18: gender,
	})
}

// The mutation that reads the composition sentinel as a direct row selects one
// fixed person in every case and fails the table below; a generic-class fallback
// leaves entity.Class at 4 and fails independently.
func TestEveryComposedNPCUsesTheEnteringHeroesArchetypeAcrossWorldRosterAndLoadout(t *testing.T) {
	weapons := swordAndBow()
	table := &mapload.Table{
		Humans: defCollection{
			{},
			{name: "PC_Danath", params: composedHumanRow(3, 5, 0), strings: []string{"Sword"}},
			{name: "PC_Naira", params: composedHumanRow(14, 1, 1), strings: []string{"Bow"}},
			{name: "PC_Fergard", params: composedHumanRow(24, 3, 0)},
			{name: "PC_Reniesta", params: composedHumanRow(24, 1, 1)},
		},
		NPC: composedRegistry(t), Shapes: identityScale(), Materials: identityScale(), Weapons: weapons,
	}
	for _, tc := range []struct {
		name       string
		npc        uint16
		playerMage bool
		dir        data.FigureDir
		row        string
		class      int32
		figure     data.FigureDir
		face       int
		weapon     string
	}{
		{"npc21 copies male fighter", 21, false, data.FigureDirManFighter, "PC_Danath", 3, data.FigureDirManFighter, 5, "Sword"},
		{"npc21 copies female mage", 21, true, data.FigureDirWomanMage, "PC_Reniesta", 24, data.FigureDirWomanMage, 1, ""},
		{"npc22 makes opposite-sex mage", 22, false, data.FigureDirManFighter, "PC_Reniesta", 24, data.FigureDirWomanMage, 1, ""},
		{"npc22 flips female to male", 22, false, data.FigureDirWomanFighter, "PC_Fergard", 24, data.FigureDirManMage, 3, ""},
		{"npc23 makes opposite-sex fighter", 23, false, data.FigureDirManFighter, "PC_Naira", 14, data.FigureDirWomanFighter, 1, "Bow"},
		{"npc23 flips female to male", 23, false, data.FigureDirWomanFighter, "PC_Danath", 3, data.FigureDirManFighter, 5, "Sword"},
		{"npc24 changes male fighter class", 24, false, data.FigureDirManFighter, "PC_Fergard", 24, data.FigureDirManMage, 3, ""},
		{"npc24 changes female mage class", 24, true, data.FigureDirWomanMage, "PC_Naira", 14, data.FigureDirWomanFighter, 1, "Bow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{
				X: 0x0c80, Y: 0x0c80, ClassID: 4, ClassSubID: tc.npc, Flags: 1,
			}}}
			party := []mapload.PartyMember{{ID: "hero", StartingHero: true, PlayerCharacter: true,
				Mage: tc.playerMage, FigureDir: string(tc.dir)}}
			w, st, err := mapload.StartMission(m, table, mapload.DifficultyNormal, party)
			if err != nil {
				t.Fatalf("StartMission: %v", err)
			}
			if got := w.Entities()[0].Class; got != tc.class {
				t.Errorf("placed class = %d, want composed row class %d", got, tc.class)
			}
			p, ok := st.Roster[0]
			if !ok {
				t.Fatal("composed placement has no roster template")
			}
			if p.Name != tc.row || p.Class != tc.class || p.FigureDir != string(tc.figure) || p.FigureFace != tc.face {
				t.Errorf("roster identity = name %q class %d figure %q/%d, want %q %d %q/%d",
					p.Name, p.Class, p.FigureDir, p.FigureFace, tc.row, tc.class, tc.figure, tc.face)
			}
			if tc.weapon != "" {
				want, err := data.ResolveWeapon(tc.weapon, table.Shapes, table.Materials, weapons)
				if err != nil {
					t.Fatalf("ResolveWeapon: %v", err)
				}
				if p.Weapon == nil || p.Worn[0] != uint16(want.Code) {
					t.Errorf("roster weapon = %+v, slot 1 %#04x; want %s %#04x",
						p.Weapon, p.Worn[0], tc.weapon, uint16(want.Code))
				}
			}
		})
	}
}

func TestCampaignMissionSelectsTheTieredComposedRow(t *testing.T) {
	table := &mapload.Table{
		Humans: defCollection{
			{},
			{name: "PC_Naira", params: withServerID(composedHumanRow(14, 1, 1), 27)},
			{name: "PC_Naira_2", params: withServerID(composedHumanRow(14, 7, 1), 31)},
		},
		NPC: composedRegistry(t),
	}
	m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{{
		X: 0x0c80, Y: 0x0c80, ClassID: 4, ClassSubID: 23, Flags: 1,
	}}}
	party := []mapload.PartyMember{{ID: "hero", StartingHero: true, PlayerCharacter: true,
		FigureDir: string(data.FigureDirManFighter)}}
	_, st, err := mapload.StartCampaignMission(m, table, mapload.DifficultyNormal, party, 70)
	if err != nil {
		t.Fatalf("StartCampaignMission: %v", err)
	}
	got := st.Roster[0]
	if got.Name != "PC_Naira_2" || got.FigureFace != 7 {
		t.Fatalf("mission 70 npc23 = %q face %d; want PC_Naira_2 face 7", got.Name, got.FigureFace)
	}
}

func TestTownCampaignMemberUsesTheSameExactRowConstructor(t *testing.T) {
	weapons := swordAndBow()
	table := &mapload.Table{
		Humans: defCollection{
			{},
			{name: "PC_Reniesta", params: withServerID(composedHumanRow(24, 1, 1), 29), strings: []string{"Sword"}},
		},
		NPC: composedRegistry(t), Shapes: identityScale(), Materials: identityScale(), Weapons: weapons,
	}
	party := []mapload.PartyMember{{ID: "hero", StartingHero: true, PlayerCharacter: true,
		FigureDir: string(data.FigureDirManFighter)}}
	got, ok := mapload.CampaignNPCMember(table, 22, 30, party)
	if !ok {
		t.Fatal("CampaignNPCMember did not resolve npc22")
	}
	if got.ID != "npc:22" || got.Name != "PC_Reniesta" || got.CompanionNPC != 22 ||
		got.StartingHero || !got.PlayerCharacter || got.WornItems[0].Code == 0 || got.Weapon == nil {
		t.Fatalf("town member lost exact row payload: %+v", got)
	}
}
