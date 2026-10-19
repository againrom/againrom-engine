package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestMissionStartBuildsTheWornSpellOnlyForAConstructedMember: a member this
// mission constructs wears his castSpell Weapon with the Spell the template
// equip builds; a carried member keeps his Weapon as held, a missing Spell
// included, and a held Spell stays the one he held (ITEM-212, ITEM-203).
// The second game's table builds for him as well.
func TestMissionStartBuildsTheWornSpellOnlyForAConstructedMember(t *testing.T) {
	table := &mapload.Table{Spells: defCollection{
		{}, // 0: reserved
		{name: "Fire Arrow", params: spellRow(3, 1, 1, 7, 4, 8, 0)},
	}}
	staff := sim.ItemInstance{Code: 0x0101, Kind: 2, WeightPresent: true, Weight: 3,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 1},
		Effects:         []sim.ItemEffect{{Kind: 41, Operand: 1 | 10<<16}}}
	held := staff.Clone()
	held.SourceEquipment.Spell = sim.SourceItemSpell{Present: true, ID: 1, Range: 9, ManaCost: 3}
	worn := func(item sim.ItemInstance) (codes [sim.EquipSlots]uint16, items [sim.EquipSlots]sim.ItemInstance) {
		codes[0], items[0] = item.Code, item
		return
	}
	carried := func(item sim.ItemInstance) mapload.PartyMember {
		codes, items := worn(item)
		return mapload.PartyMember{Class: 100, Hero: data.NewCampaignHero(data.SkillBlade),
			Carry: &mapload.Carry{Equipped: codes, EquippedItems: items}}
	}
	constructedCodes, constructedItems := worn(staff)
	party := []mapload.PartyMember{carried(staff), carried(held),
		{Class: 101, Hero: data.NewCampaignHero(data.SkillBlade), Worn: constructedCodes, WornItems: constructedItems}}
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20}, mapload.Cell{X: 19, Y: 20}, mapload.Cell{X: 21, Y: 20})
	w, st, err := mapload.StartMission(m, table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	spell := func(i int) sim.SourceItemSpell {
		items, ok := w.EquippedItems(st.IDs[i])
		if !ok {
			t.Fatalf("member %d holds no equipment", i)
		}
		return items[0].SourceEquipment.Spell
	}
	if got := spell(0); got.Present {
		t.Errorf("carried member without a Spell starts with %+v, want none", got)
	}
	if got := spell(1); got != held.SourceEquipment.Spell {
		t.Errorf("carried member's held Spell became %+v, want %+v", got, held.SourceEquipment.Spell)
	}
	if got := spell(2); !got.Present || got.ID != 1 {
		t.Errorf("constructed member starts with Spell %+v, want the built Spell 1", got)
	}
	if party[2].WornItems[0].SourceEquipment.Spell.Present {
		t.Fatal("the mission start changed the party it was handed")
	}
	// The second game's table builds the carried member's missing Spell too.
	table.CarriedWornSpellBuild = true
	w, st, err = mapload.StartMission(m, table, mapload.DifficultyNormal, party)
	if err != nil {
		t.Fatalf("StartMission: %v", err)
	}
	if got := spell(0); !got.Present || got.ID != 1 {
		t.Errorf("second-game carried member starts with Spell %+v, want the built Spell 1", got)
	}
}
