package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondStaffFront is the second-game generator fixture with the weapons the
// campaign part names and a Spells table holding the five school spells.
func secondStaffFront(t *testing.T) (*FrontEnd, map[string]uint32) {
	t.Helper()
	f := generatorSecondFront(t)
	g := secondGeneratorCampaigns["rom2"]
	weapons := dbCollection{{}}
	for _, name := range append([]string{g.FemaleStaff, g.MaleStaff}, g.FighterWeapons...) {
		weapons = append(weapons, dbEntry{name: name, params: chargenWeaponParams(data.SkillBlade)})
	}
	spells := dbCollection{{}}
	ids := map[string]uint32{}
	for i, token := range g.StaffSpells {
		spells = append(spells, dbEntry{name: spellDisplayName(token), params: make([]int32, 19)})
		ids[token] = uint32(i + 1)
	}
	f.Table.Weapons = weapons
	f.Table.Spells = spells
	return f, ids
}

func spellDisplayName(token string) string {
	out := []byte(token)
	for i, c := range out {
		if c == '_' {
			out[i] = ' '
		}
	}
	return string(out)
}

// A mage made in the second game's generator wears a staff whose worn item
// casts the chosen school's spell at the part's level, for each school, both
// sexes, and after the school was changed on the detail page.
func TestSecondGeneratorMageStaffCastsTheChosenSchool(t *testing.T) {
	f, ids := secondStaffFront(t)
	g := secondGeneratorCampaigns["rom2"]
	check := func(label string, res ui.ChargenResult, school int) {
		t.Helper()
		party := f.ChargenParty(res)
		if len(party) != 1 {
			t.Fatalf("%s: party of %d", label, len(party))
		}
		m := party[0]
		token := g.StaffSpells[school]
		if m.Weapon == nil || m.Weapon.SpellName != token || int(m.Weapon.SpellPower) != g.StaffSpellLevel {
			t.Fatalf("%s %v: weapon %+v, want %s at %d", label, res.Choices, m.Weapon, token, g.StaffSpellLevel)
		}
		item := m.WornItems[0]
		want := sim.ItemEffect{Kind: 41, Operand: ids[token] | uint32(g.StaffSpellLevel)<<16}
		if item.Code == 0 || len(item.Effects) != 1 || item.Effects[0] != want {
			t.Fatalf("%s: worn staff %04x effects %+v, want %+v", label, item.Code, item.Effects, want)
		}
		if m.Worn[0] != item.Code || uint16(m.Weapon.Code) != item.Code {
			t.Fatalf("%s: worn code %04x, item %04x, weapon %04x", label, m.Worn[0], item.Code, m.Weapon.Code)
		}
	}
	for sex := 0; sex < 2; sex++ {
		for school := 0; school < data.SkillSlots-1; school++ {
			check("direct", ui.ChargenResult{Name: "M", Choices: []int{sex, 1, school}, Stats: []int{25, 25, 25, 25}, Difficulty: 2}, school)
		}
	}
	// The party is a function of the result alone, so a school changed on the
	// detail page and then confirmed is the school the staff casts: walk the
	// schools in a changing order on one front.
	for sex := 0; sex < 2; sex++ {
		for _, school := range []int{3, 1, 4, 0, 2, 2, 0} {
			check("changed", ui.ChargenResult{Name: "M", Choices: []int{sex, 1, school}, Stats: []int{25, 25, 25, 25}, Difficulty: 2}, school)
		}
	}
	// The town start keeps that staff on the committed hero.
	res := ui.ChargenResult{Name: "M", Choices: []int{1, 1, 4}, Stats: []int{25, 25, 25, 25}, Difficulty: 2}
	if _, err := f.NewGameBegin(10)(res); err != nil {
		t.Fatal(err)
	}
	token := g.StaffSpells[4]
	want := sim.ItemEffect{Kind: 41, Operand: ids[token] | uint32(g.StaffSpellLevel)<<16}
	if len(f.Carried) != 1 || len(f.Carried[0].WornItems[0].Effects) != 1 || f.Carried[0].WornItems[0].Effects[0] != want {
		t.Fatalf("committed hero %+v, want a staff casting %s", f.Carried, token)
	}
}
