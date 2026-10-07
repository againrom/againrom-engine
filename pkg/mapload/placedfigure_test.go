package mapload_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// placedPersonNPCs is a scenario registry with two person records: npc60 carries
// the Hero flag and npc61 does not.
func placedPersonNPCs(t *testing.T, heroServerID, plainServerID int32) *data.NPCDefs {
	t.Helper()
	r, err := reg.Parse(synth.Reg(0x11, []synth.RegNode{
		{Name: "npc60", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Hero,Human"},
			{Name: "DataBinID", Kind: 0x02, Int: heroServerID}}},
		{Name: "npc61", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Flags", Kind: 0x00, Str: "Human"},
			{Name: "DataBinID", Kind: 0x02, Int: plainServerID}}},
	}))
	if err != nil {
		t.Fatalf("reg.Parse: %v", err)
	}
	return data.LoadNPCDefs(r)
}

// A placed person who is not a Hero-mode placement keeps his table type id and is
// drawn by the second arm, whose sex is bit 7 of the face byte and whose face is
// the byte's low seven bits (UNIT-PICT-035, ALM-CLS-054, PARTY-M20-031). The
// spawner replaces the constructor's face byte with the placement's secondary key
// and puts bit 2 of the placement's flag word in bit 7, on the type-key arm always
// and on the definition-id arm when the secondary key is not zero
// (ALM-FLAGPATH-109). Every other person keeps the constructor's byte, and the
// original saves put the row's gender column in its bit 7.
//
// The rows disagree with every placement below on purpose: a woman row is placed
// as a man and a man row as a woman, so a figure read off the row's own columns
// cannot pass.
func TestARosterTemplateDrawsTheFaceByteItsSpawnerWrites(t *testing.T) {
	const (
		manFighter   = 101
		womanFighter = 102
		womanMage    = 103
		manMage      = 104
		heroWoman    = 105
		plainWoman   = 106
		sexFlag      = 4
		npcFlag      = 1
	)
	table := &mapload.Table{
		Units: defCollection{{}},
		Humans: defCollection{
			{},
			{name: "ManFighter", params: withServerID(composedHumanRow(3, 8, 0), manFighter)},
			{name: "WomanFighter", params: withServerID(composedHumanRow(4, 8, 1), womanFighter)},
			{name: "WomanMage", params: withServerID(composedHumanRow(0x18, 5, 1), womanMage)},
			{name: "ManMage", params: withServerID(composedHumanRow(0x17, 6, 0), manMage)},
			{name: "HeroWoman", params: withServerID(composedHumanRow(4, 7, 1), heroWoman)},
			{name: "PlainWomanMage", params: withServerID(composedHumanRow(0x18, 4, 1), plainWoman)},
		},
		NPC:    placedPersonNPCs(t, heroWoman, plainWoman),
		Shapes: identityScale(), Materials: identityScale(), Weapons: swordAndBow(),
	}
	for _, tc := range []struct {
		name string
		unit alm.Unit
		dir  data.FigureDir
		face int
		mage bool
	}{
		{"type-key placement of a man row with secondary 5",
			alm.Unit{ClassID: 3, ClassSubID: 5}, data.FigureDirManFighter, 5, false},
		{"type-key placement of a woman row with the sex flag clear is a man",
			alm.Unit{ClassID: 4, ClassSubID: 5}, data.FigureDirManFighter, 5, false},
		{"type-key placement of a man row with the sex flag set is a woman",
			alm.Unit{ClassID: 3, ClassSubID: 5, Flags: sexFlag}, data.FigureDirWomanFighter, 5, false},
		{"definition-id placement with no secondary key keeps the row's woman",
			alm.Unit{ClassID: 4, DefID: womanFighter}, data.FigureDirWomanFighter, 8, false},
		{"the sex flag alone changes nothing on a definition-id placement",
			alm.Unit{ClassID: 3, DefID: manFighter, Flags: sexFlag}, data.FigureDirManFighter, 8, false},
		{"definition-id placement of a woman row with secondary 3 is a man of face 3",
			alm.Unit{ClassID: 4, ClassSubID: 3, DefID: womanFighter}, data.FigureDirManFighter, 3, false},
		{"definition-id placement of a woman mage with the sex flag set keeps her a woman of face 2",
			alm.Unit{ClassID: 0x18, ClassSubID: 2, DefID: womanMage, Flags: sexFlag}, data.FigureDirWomanMage, 2, true},
		{"the secondary key's own bit 7 is replaced by the flag",
			alm.Unit{ClassID: 0x17, ClassSubID: 0x82, DefID: manMage}, data.FigureDirManMage, 2, true},
		{"a face number of zero names no sheet and keeps the row",
			alm.Unit{ClassID: 3, ClassSubID: 0x100, DefID: manFighter, Flags: sexFlag}, data.FigureDirManFighter, 8, false},
		// The npc arm's secondary key is a record subscript and its spawner
		// writes no face byte, so neither the subscript nor the sex flag is a face.
		{"a Hero-mode npc placement is drawn from its row",
			alm.Unit{ClassID: 1, ClassSubID: 60, Flags: npcFlag | sexFlag}, data.FigureDirWomanFighter, 7, false},
		{"a zero-mode npc placement is drawn from its row",
			alm.Unit{ClassID: 1, ClassSubID: 61, Flags: npcFlag}, data.FigureDirWomanMage, 4, true},
		{"the sex flag alone changes nothing on a zero-mode npc placement",
			alm.Unit{ClassID: 1, ClassSubID: 61, Flags: npcFlag | sexFlag}, data.FigureDirWomanMage, 4, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.unit.X, tc.unit.Y = 0x0c80, 0x0c80
			m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{tc.unit}}
			dir, face, ok := mapload.PlacedFigure(tc.unit, table)
			if !ok || dir != tc.dir || face != tc.face {
				t.Errorf("PlacedFigure = %q face %d present %t, want %q face %d", dir, face, ok, tc.dir, tc.face)
			}
			_, roster, err := mapload.FromALMRoster(m, table, mapload.DifficultyNormal)
			if err != nil {
				t.Fatalf("FromALMRoster: %v", err)
			}
			p, ok := roster[sim.EntityID(0)]
			if !ok {
				t.Fatal("the placement has no roster template")
			}
			if p.FigureDir != string(tc.dir) || p.FigureFace != tc.face || p.Mage != tc.mage {
				t.Errorf("roster figure = %q face %d mage %t, want %q face %d mage %t",
					p.FigureDir, p.FigureFace, p.Mage, tc.dir, tc.face, tc.mage)
			}
		})
	}
}

func TestZeroModePlacedPersonsUseTheConstructorFaceByte(t *testing.T) {
	const plainServerID, heroServerID = 101, 102
	for _, tc := range []struct {
		name         string
		typeID       int32
		face, gender int32
		unit         alm.Unit
		dir          data.FigureDir
		wantFace     int
		wantByte     uint8
	}{
		{"empty face and gender", 3, -1, -1, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirWomanFighter, 1, 0x81},
		{"empty face with male gender", 3, -1, 0, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirManFighter, 1, 1},
		{"empty gender defaults female", 3, 7, -1, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirWomanFighter, 7, 0x87},
		{"odd gender uses its low bit", 3, 7, 3, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirWomanFighter, 7, 0x87},
		{"even gender uses its low bit", 3, 7, 2, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirManFighter, 7, 7},
		{"negative odd gender uses its low bit", 3, 7, -3, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirWomanFighter, 7, 0x87},
		{"zero constructor face has a safe sheet", 3, 0, 1, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirWomanFighter, 1, 0x81},
		{"gender is ORed with the streamed face byte", 3, 0x85, 0, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirWomanFighter, 5, 0x85},
		{"face wraps to the streamed byte", 3, 0x103, 0, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirManFighter, 3, 3},
		{"mage keeps the constructor defaults", 0x18, -1, -1, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 1}, data.FigureDirWomanMage, 1, 0x81},
		{"npc placement sex flag has no store", 3, 7, 0, alm.Unit{ClassID: 1, ClassSubID: 61, Flags: 5}, data.FigureDirManFighter, 7, 7},
		{"type key overrides the missing gender", 3, 7, -1, alm.Unit{ClassID: 3, ClassSubID: 5}, data.FigureDirManFighter, 5, 5},
		{"definition id with no secondary keeps the constructor", 3, 7, -1, alm.Unit{ClassID: 3, DefID: plainServerID}, data.FigureDirWomanFighter, 7, 0x87},
		{"stored zero falls back to the constructor", 3, 7, -1, alm.Unit{ClassID: 3, ClassSubID: 0x100, DefID: plainServerID, Flags: 4}, data.FigureDirWomanFighter, 7, 0x87},
		{"hero mode keeps its separate row arm", 3, 7, 3, alm.Unit{ClassID: 1, ClassSubID: 60, Flags: 1}, data.FigureDirManFighter, 7, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := &mapload.Table{
				Units: defCollection{{}},
				Humans: defCollection{
					{},
					{name: "PlainPerson", params: withServerID(composedHumanRow(tc.typeID, tc.face, tc.gender), plainServerID)},
					{name: "HeroPerson", params: withServerID(composedHumanRow(4, 7, 3), heroServerID)},
					{}, {},
					{name: "HeroBinding", params: withServerID(composedHumanRow(0x21, 7, 0), 103)},
				},
				NPC:    placedPersonNPCs(t, heroServerID, plainServerID),
				Shapes: identityScale(), Materials: identityScale(), Weapons: swordAndBow(),
			}
			tc.unit.X, tc.unit.Y = 0x0c80, 0x0c80
			dir, face, ok := mapload.PlacedFigure(tc.unit, table)
			if !ok || dir != tc.dir || face != tc.wantFace {
				t.Errorf("PlacedFigure = %q/%d present %t; want %q/%d", dir, face, ok, tc.dir, tc.wantFace)
			}
			m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{tc.unit}}
			world, roster, err := mapload.FromALMRoster(m, table, mapload.DifficultyNormal)
			if err != nil {
				t.Fatal(err)
			}
			person, ok := roster[sim.EntityID(0)]
			if !ok {
				t.Fatal("placement has no roster template")
			}
			if person.FigureDir != string(tc.dir) || person.FigureFace != tc.wantFace {
				t.Errorf("roster figure = %q/%d; want %q/%d", person.FigureDir, person.FigureFace, tc.dir, tc.wantFace)
			}
			entity, ok := world.Entity(0)
			if !ok {
				t.Fatal("placement has no entity")
			}
			human := data.HumanState{TypeID: uint16(entity.TypeID), Fighter: !tc.dir.Mage()}
			bound, _, err := mapload.ConstructActorBasis(entity, person, &tc.unit, table, 1, 1, &human, 0)
			if err != nil {
				t.Fatalf("constructor basis: %v", err)
			}
			if bound.SourceBinding.Face != tc.wantByte {
				t.Errorf("SourceBinding.Face = %#x; want %#x", bound.SourceBinding.Face, tc.wantByte)
			}
		})
	}
}
