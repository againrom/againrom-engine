package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A second-game person is drawn by the face byte of his Humans row: the face
// column with the gender in bit 7, whatever the record's +0x0a word and flag
// bit 2 hold (R2-ASSET-172, R2-ENGINE-681). The constructor binds that byte,
// which SAVE stores, and the LOAD relink of the stored type and byte gives the
// same body (R2-ENGINE-682). Only a person whose NPC word is 21..23 is drawn
// with the hero class, gender + 2 for mana above 0x21, while his World type id
// stays the row's (R2-ENGINE-680).
func TestASecondGamePersonDrawsHisRowsFaceByte(t *testing.T) {
	mageHero := withServerID(composedHumanRow(23, 20, 1), 10230)
	mageHero[5] = 70
	table := &mapload.Table{
		UnitKeys: mapload.ServerUnitKeys,
		Units:    defCollection{{}},
		Humans: defCollection{
			{},
			{name: "2U_PeasantMale", params: withServerID(composedHumanRow(1, 46, 0), 2108)},
			{name: "2U_PeasantFemale", params: withServerID(composedHumanRow(1, 13, 1), 2109)},
			{name: "2A_PeasantGuard", params: withServerID(composedHumanRow(14, 16, 1), 2111)},
			{name: "2F_BrigandLeader2", params: withServerID(composedHumanRow(9, 52, 0), 2107)},
			{name: "2M_Brigand2", params: withServerID(composedHumanRow(24, 15, 1), 2106)},
			{name: "Mage_Hero", params: mageHero},
		},
		Shapes: identityScale(), Materials: identityScale(), Weapons: swordAndBow(),
	}
	for _, tc := range []struct {
		name     string
		unit     alm.Unit
		dir      data.FigureDir
		face     int
		faceByte uint8
		hero     int32
	}{
		{"a peasant man", alm.Unit{ClassID: 1, ClassSubID: 1, ServerID: 2108, Flags: 0x10}, data.FigureDirManFighter, 46, 46, 0},
		{"a peasant woman", alm.Unit{ClassID: 1, ClassSubID: 1, ServerID: 2109, Flags: 0x10}, data.FigureDirWomanFighter, 13, 0x8d, 0},
		{"the sex flag and the +0x0a word are no face", alm.Unit{ClassID: 1, ClassSubID: 5, ServerID: 2108, Flags: 0x14}, data.FigureDirManFighter, 46, 46, 0},
		{"an archer woman", alm.Unit{ClassID: 14, ClassSubID: 1, ServerID: 2111, Flags: 0x10}, data.FigureDirWomanFighter, 16, 0x90, 0},
		{"a two-handed axeman", alm.Unit{ClassID: 9, ClassSubID: 1, ServerID: 2107, Flags: 0x10}, data.FigureDirManFighter, 52, 52, 0},
		{"a staff mage woman", alm.Unit{ClassID: 24, ClassSubID: 1, ServerID: 2106, Flags: 0x10}, data.FigureDirWomanMage, 15, 0x8f, 0},
		{"the hero mage", alm.Unit{ClassID: 23, ClassSubID: 1, ServerID: 10230, Flags: 0x10}, data.FigureDirWomanMage, 20, 0x94, 0x24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.unit.X, tc.unit.Y = 0x0c80, 0x0c80
			dir, face, ok := mapload.PlacedFigure(tc.unit, table)
			if !ok || dir != tc.dir || face != tc.face {
				t.Errorf("PlacedFigure = %q/%d present %t, want %q/%d", dir, face, ok, tc.dir, tc.face)
			}
			hero, isHero := mapload.PlacedClientHero(tc.unit, table)
			if isHero != (tc.hero != 0) || hero != tc.hero {
				t.Errorf("PlacedClientHero = %#x %t, want %#x", hero, isHero, tc.hero)
			}
			m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{tc.unit}}
			world, roster, err := mapload.FromALMRoster(m, table, mapload.DifficultyNormal)
			if err != nil {
				t.Fatal(err)
			}
			person, ok := roster[sim.EntityID(0)]
			if !ok || person.FigureDir != string(tc.dir) || person.FigureFace != tc.face {
				t.Fatalf("roster figure = %q/%d present %t, want %q/%d", person.FigureDir, person.FigureFace, ok, tc.dir, tc.face)
			}
			entity, _ := world.Entity(0)
			if want := int32(tc.unit.ClassID); entity.TypeID != want {
				t.Errorf("World type id %d, want the row's %d", entity.TypeID, want)
			}
			human := data.HumanState{TypeID: uint16(entity.TypeID), Fighter: !tc.dir.Mage()}
			bound, _, err := mapload.ConstructActorBasis(entity, person, &tc.unit, table, 1, 1, &human, 0)
			if err != nil {
				t.Fatalf("constructor basis: %v", err)
			}
			if bound.SourceBinding.Face != tc.faceByte {
				t.Errorf("bound face byte %#x, want %#x", bound.SourceBinding.Face, tc.faceByte)
			}
			loaded, err := mapload.SourceActorPerson(bound, person.Name, table)
			if err != nil {
				t.Fatalf("LOAD relink: %v", err)
			}
			if loaded.FigureDir != string(tc.dir) || loaded.FigureFace != tc.face {
				t.Errorf("loaded figure = %q/%d, want %q/%d", loaded.FigureDir, loaded.FigureFace, tc.dir, tc.face)
			}
		})
	}
}

// The NPC word of a second-game Humans row is its server id, or (id/10)%1000
// for an id above 10000 (R2-ENGINE-525, R2-ASSET-172).
func TestPersonNPCWord(t *testing.T) {
	for id, want := range map[int32]uint16{2112: 2112, 694: 694, 10230: 23, 10220: 22, 10000: 10000, 10010: 1} {
		if got := mapload.PersonNPCWord(id); got != want {
			t.Errorf("PersonNPCWord(%d) = %d, want %d", id, got, want)
		}
	}
}

// The first game's placement keys never give a client hero: its hero NPCs carry
// the hero type in the World itself.
func TestTheFirstGameHasNoClientHero(t *testing.T) {
	table := &mapload.Table{
		Units:  defCollection{{}},
		Humans: defCollection{{}, {name: "Person", params: withServerID(composedHumanRow(23, 20, 1), 10230)}},
		NPC:    placedPersonNPCs(t, 10230, 10230),
	}
	for _, u := range []alm.Unit{{ClassID: 23, DefID: 10230}, {ClassID: 1, ClassSubID: 60, Flags: 1}, {ClassID: 23}} {
		if typeID, ok := mapload.PlacedClientHero(u, table); ok {
			t.Errorf("%+v: client hero %#x", u, typeID)
		}
	}
}
