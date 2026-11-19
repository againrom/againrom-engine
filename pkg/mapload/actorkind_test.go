package mapload_test

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// kindTable is a second-game table: a peasant (server id 2108), the mage
// hero template (10230, schools 2 and 3 equal and largest), a person whose
// server id cell is empty, and a creature whose parameter 55 is 10030.
func kindTable(kinds bool) *mapload.Table {
	mageHero := withServerID(composedHumanRow(23, 20, 1), 10230)
	mageHero[5] = 70
	mageHero[slotHumanSkillBase+1], mageHero[slotHumanSkillBase+2] = 5, 7
	mageHero[slotHumanSkillBase+3], mageHero[slotHumanSkillBase+4] = 7, 3
	creature := unitDefRow(26, 0, 30)
	for len(creature) < 56 {
		creature = append(creature, -1)
	}
	creature[55] = 10030
	edition := base.GameROM2.Edition()
	return &mapload.Table{
		Edition:  &edition,
		UnitKeys: mapload.ServerUnitKeys,
		Units:    defCollection{{}, {name: "Goblin", params: creature}},
		Humans: defCollection{
			{},
			{name: "2U_PeasantMale", params: withServerID(composedHumanRow(1, 46, 0), 2108)},
			{name: "Mage_Hero", params: mageHero},
			{name: "Man_Unarmed", params: composedHumanRow(1, 46, 0)},
		},
		Shapes: identityScale(), Materials: identityScale(), Weapons: swordAndBow(),
		ActorKindWords: kinds,
	}
}

// A second-game row gives its actor the kind word its constructor stores
// and, for a `_Hero` or `Start_` template, the first of its largest schools
// as the main skill (R2-ENGINE-319, R2-ENGINE-525); the first game gives
// neither.
func TestSecondGameRowsGiveTheirKindWordAndMainSkill(t *testing.T) {
	for _, tc := range []struct {
		name  string
		human bool
		row   int
		kind  uint16
		main  int32
	}{
		{"a person below 10000", true, 1, 2108, 0},
		{"the hero template", true, 2, 23, 2},
		{"an empty server id", true, 3, 0, 0},
		{"a creature keeps parameter 55", false, 1, 10030, 0},
	} {
		kind, main := mapload.RowActorKind(tc.human, tc.row, kindTable(true))
		if kind != tc.kind || main != tc.main {
			t.Errorf("%s: kind %d main %d, want %d and %d", tc.name, kind, main, tc.kind, tc.main)
		}
		if kind, main := mapload.RowActorKind(tc.human, tc.row, kindTable(false)); kind != 0 || main != 0 {
			t.Errorf("%s in the first game: kind %d main %d, want none", tc.name, kind, main)
		}
	}
}

// A map placement's actor carries them through the one actor constructor.
func TestSecondGamePlacementsCarryTheirKindWordAndMainSkill(t *testing.T) {
	for _, tc := range []struct {
		name string
		unit alm.Unit
		kind uint16
		main int32
	}{
		{"the hero template", alm.Unit{ClassID: 23, ClassSubID: 1, ServerID: 10230, Flags: 0x10}, 23, 2},
		{"a peasant", alm.Unit{ClassID: 1, ClassSubID: 1, ServerID: 2108, Flags: 0x10}, 2108, 0},
		{"a creature", alm.Unit{ClassID: 26, ServerID: 10030}, 10030, 0},
	} {
		tc.unit.X, tc.unit.Y = 0x0c80, 0x0c80
		m := &alm.Map{Width: 40, Height: 40, Units: []alm.Unit{tc.unit}}
		world, _, err := mapload.FromALMRoster(m, kindTable(true), mapload.DifficultyNormal)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		e, _ := world.Entity(sim.EntityID(0))
		if e.KindWord != tc.kind || e.MainSkill != tc.main {
			t.Errorf("%s: actor kind %d main %d, want %d and %d", tc.name, e.KindWord, e.MainSkill, tc.kind, tc.main)
		}
	}
}

// A party member's actor holds his identity as the kind word and his own
// main skill; a hero an older SAVE carried without one holds the first of
// his largest schools.
func TestSecondGamePartyActorsCarryTheirIdentityAndMainSkill(t *testing.T) {
	hero := mapload.PartyMember{StartingHero: true, PlayerCharacter: true}
	hero.Hero.Skill = [6]int32{0, 20, 20, 0, 0, 10}
	if got := mapload.PartyMainSkill(hero); got != 1 {
		t.Errorf("an older hero's main skill %d, want 1", got)
	}
	hero.MainSkill = 2
	if got := mapload.PartyMainSkill(hero); got != 2 {
		t.Errorf("a hero's own main skill %d, want 2", got)
	}
	hire := mapload.PartyMember{Temporary: true, CampaignHire: 2, Class: 1}
	hire.Hero.Skill = [6]int32{0, 20, 0, 0, 0, 0}
	if got := mapload.PartyMainSkill(hire); got != 0 {
		t.Errorf("a hire's main skill %d, want 0", got)
	}
	hero.Class = 1
	unit := mapload.PartyMember{Temporary: true, CampaignHire: 4, CampaignUnit: true, DefinitionRow: 1, Class: 26}
	w, st, err := mapload.StartMission(&alm.Map{Width: 40, Height: 40}, kindTable(true), mapload.DifficultyNormal,
		[]mapload.PartyMember{hero, hire, unit})
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range [][2]int32{{21, 2}, {2, 0}, {4, 0}} {
		e, _ := w.Entity(st.IDs[i])
		if int32(e.KindWord) != want[0] || e.MainSkill != want[1] {
			t.Errorf("member %d: actor kind %d main %d, want %d and %d", i, e.KindWord, e.MainSkill, want[0], want[1])
		}
	}
}

// A Summon's creature is built by CheatActor through the same constructor.
func TestSecondGameSummonedCreatureCarriesItsKindWord(t *testing.T) {
	e, _, _, err := mapload.CheatActor("Goblin", false, kindTable(true), mapload.DifficultyNormal)
	if err != nil || e.KindWord != 10030 || e.MainSkill != 0 {
		t.Errorf("the summoned creature holds kind %d main %d (%v), want 10030 and 0", e.KindWord, e.MainSkill, err)
	}
}
