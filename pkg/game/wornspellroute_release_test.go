package game

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestReleaseTavernHireWearsTheBuiltStaffSpell: a first-game tavern hire is a
// Human construction with template equip, so every reachable template whose
// Weapon casts wears it with the Spell built from its first castSpell Effect
// (ITEM-212, ITEM-203). The row's own worn Weapon, before the hire, holds
// none: the control.
func TestReleaseTavernHireWearsTheBuiltStaffSpell(t *testing.T) {
	f := releaseFront(t)
	if f.Archives.Game() != base.GameROM1 {
		t.Skip("the second game keeps the row's Spell")
	}
	seen := map[string]bool{}
	staffs := 0
	for _, mission := range f.Campaign.Value().Main {
		chapter := f.Campaign.Value().Chapters[mission]
		if len(chapter.Mercenaries) == 0 {
			continue
		}
		town := NewTown(f.Campaign.Value())
		for _, prior := range f.Campaign.Value().Main {
			if prior >= mission {
				break
			}
			town.Won(prior)
		}
		if town.Chapter() != mission {
			continue
		}
		f.Town = town
		s := f.TownScreen().(*townScreen)
		s.room = roomTavern
		for _, typ := range chapter.Mercenaries {
			template := fmt.Sprintf("NPC%02d_%d", typ, mercenaryLevel(mission))
			if typ <= 2 || seen[template] {
				continue
			}
			seen[template] = true
			row := data.FindHumanByName(f.Table.Humans, template)
			if row == data.NotFound {
				t.Fatalf("template %s is absent", template)
			}
			raw, _, _, err := mapload.HumanRowEquipment(f.Table.Humans.EntryStrings(row), f.Table)
			if err != nil {
				t.Fatal(err)
			}
			id, _, casts := raw[0].CastSpell()
			if !casts {
				continue
			}
			if raw[0].SourceEquipment.Spell.Present {
				t.Fatalf("control: template %s's row Weapon already holds a Spell", template)
			}
			members, ok := s.buildMercenarySquad(typ, 1)
			if !ok || len(members) != 1 {
				t.Fatalf("template %s is unavailable", template)
			}
			if got := members[0].WornItems[0].SourceEquipment.Spell; !got.Present || got.ID != uint8(id) {
				t.Fatalf("hired %s wears Spell %+v, want the built Spell %d", template, got, uint8(id))
			}
			staffs++
		}
	}
	if staffs == 0 {
		t.Fatal("no reachable tavern template wears a castSpell Weapon")
	}
	t.Logf("%d hired templates wear a castSpell Weapon with its built Spell", staffs)
}

// TestReleaseWornGapStaffTextAndCardFollowTheWeapon: on the owner's SAV,
// written by the original, Ferion wears a castSpell staff with no Spell. Its
// information text is the one the same staff gives with its Spell and on a
// shelf, and its range line states the staff's range (ITEM-211). His card,
// a mage's, states the damage of a Spell filled from the staff's spell id
// and power and no attack row (HERO-172).
func TestReleaseWornGapStaffTextAndCardFollowTheWeapon(t *testing.T) {
	f, mage := wornGapLoad(t)
	worn, _ := f.live.world.EquippedItems(mage)
	staff := worn[0]
	spell, _, _ := staff.CastSpell()
	table, words := f.Table, f.live.view.Words()
	f.live.switchInventorySubject(uint32(mage))
	got := f.live.invSubject.SlotInfo[0]
	built := mapload.SourceEquippedItem(staff, table)
	if !built.SourceEquipment.Spell.Present {
		t.Fatal("control: the install builds no Spell for the staff")
	}
	shelf := itemInstanceInfoLines(staff, table, words)
	withSpell := itemInstanceInfoLinesWithWeaponDamage(built, table, f.live.itemWeaponDamage, words)
	if len(got) == 0 || !slices.Equal(got, shelf) || !slices.Equal(got, withSpell) {
		t.Fatalf("worn without Spell %q, on a shelf %q, worn with Spell %q: want one text", got, shelf, withSpell)
	}
	weapon, err := data.WeaponFromCode(data.ItemCode(staff.Code), table.Shapes, table.Materials, table.Weapons)
	if err != nil {
		t.Fatal(err)
	}
	rangeLine := words.Hover[spellLabelRange] + ": " + strconv.Itoa(int(uint8(weapon.Range)))
	if !slices.Contains(got, rangeLine) {
		t.Fatalf("staff text %q lacks the Weapon's range line %q", got, rangeLine)
	}
	rules := mapload.SpellRules(table)
	rule := rules[uint8(spell)-1]
	if spellRange := words.Hover[spellLabelRange] + ": " + strconv.Itoa(int(rule.MaxRange)); rule.MaxRange != uint8(weapon.Range) && slices.Contains(got, spellRange) {
		t.Fatalf("staff text %q states the spell's range %q", got, spellRange)
	}
	drawn := false
	for _, d := range f.live.entityDraws() {
		if d.ID != uint32(mage) {
			continue
		}
		drawn = true
		if !d.Char.Known || !d.Char.Mage || !d.Combat.WeaponSpellKnown {
			t.Fatalf("Ferion's card: char %+v combat %+v, want a known mage", d.Char, d.Combat)
		}
		var cast sim.ItemEffect
		for _, effect := range staff.Effects {
			if effect.Kind == 41 {
				cast = effect
			}
		}
		c := sim.SpellCharacteristicsAtPower(f.live.world.Rules(), rule, int32(uint8(cast.Operand>>16)))
		if d.Combat.WeaponSpellDamageKnown != (c.DamageMax != 0) ||
			c.DamageMax != 0 && (d.Combat.SpellDamageBase != c.DamageMin || d.Combat.SpellDamageSpread != c.DamageMax-c.DamageMin) {
			t.Fatalf("Ferion's card combat %+v, want the filled Spell's damage %d-%d", d.Combat, c.DamageMin, c.DamageMax)
		}
		t.Logf("Ferion %d: staff 0x%04x range %d, spell %d range %d; card damage %d-%d (shown %t); text %q",
			mage, staff.Code, uint8(weapon.Range), uint8(spell), rule.MaxRange, c.DamageMin, c.DamageMax, d.Combat.WeaponSpellDamageKnown, got)
	}
	if !drawn {
		t.Fatal("Ferion has no map draw")
	}
}
