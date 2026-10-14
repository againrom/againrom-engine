package sim

import (
	"reflect"
	"testing"
)

// wornSpellWorld is a party actor without source operands that carries one
// saved spell weapon and equips it on the native saved-object route.
func wornSpellWorld(t *testing.T, item ItemInstance) *World {
	t.Helper()
	return wornSpellWorldOf(t, item, SelfSlot)
}

func wornSpellWorldOf(t *testing.T, item ItemInstance, owner uint32) *World {
	t.Helper()
	w := mustWorld(t, 1, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 20, MaxHP: 20, Owner: owner}, {ID: 8, X: 2, Y: 1, HP: 20, MaxHP: 20, Owner: owner}})
	w.spells = []SpellRule{{ID: 14, MaxRange: 7, ManaCost: 80}, {ID: 6, MaxRange: 6, ManaCost: 5, Defensive: true}}
	w.carried[0] = []ItemStack{StackItem(item, 1)}
	w.recomputeLoad(0)
	bindOperationsPack(t, w, 0)
	w.equip(0, 0, 1, 0)
	if w.equipment[0][0].ObjectID == 0 || len(w.carried[0]) != 0 {
		t.Fatalf("the native route did not wear the saved weapon: %+v", w.equipment[0][0])
	}
	return w
}

func staffEffects(spell, power uint32) []ItemEffect {
	return []ItemEffect{{Kind: 12, Operand: 2}, {Kind: 41, Operand: spell | power<<16}}
}

// TestNativeEquipBuildsTheWornWeaponSpell: a party actor's native equip of a
// classed staff gives the saved row a new Spell from the rule
// (ITEM-SPELLMOVE-132) and changes no actor value beyond the equip; another
// player's equip, the control, builds none.
func TestNativeEquipBuildsTheWornWeaponSpell(t *testing.T) {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	item.Effects = staffEffects(14, 20)
	control := wornSpellWorldOf(t, item, 3)
	if id := control.equipment[0][0].ObjectID; control.savedObjects.item(id).Spell != 0 || control.equipment[0][0].SourceEquipment.Spell.Present {
		t.Fatal("control: another player's native equip built a Spell")
	}
	w := wornSpellWorld(t, item)
	id := w.equipment[0][0].ObjectID
	row := w.savedObjects.item(id)
	want := SourceItemSpell{Present: true, ID: 14, Range: 7, ManaCost: 80}
	if row.Spell == 0 || w.savedObjects.spell(row.Spell).Value != want || row.Value.SourceEquipment.Spell != want {
		t.Fatalf("the worn staff holds Spell %+v", row.Value.SourceEquipment.Spell)
	}
	if !StackStateEqual(row.Value, StackItem(w.equipment[0][0], 1)) {
		t.Fatal("the worn view differs from its saved row")
	}
	first := row.Spell
	if built := w.BuildWornWeaponSpells(nil); built != 0 || w.savedObjects.item(id).Spell != first {
		t.Fatal("a held Spell was built again")
	}
	cold := reloadOperations(t, w)
	if cold.savedObjects.item(id).Spell != first || cold.equipment[0][0].SourceEquipment.Spell != want {
		t.Fatal("native SAVE/LOAD lost the Spell")
	}
}

// TestBuildWornWeaponSpellsClosesALoadedGap: a party actor whose worn staff
// holds no Spell, as a loaded record can, gains it from the build without an
// actor value changing.
func TestBuildWornWeaponSpellsClosesALoadedGap(t *testing.T) {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	item.Effects = staffEffects(14, 20)
	w := wornSpellWorldOf(t, item, 3)
	w.entities[0].Owner = SelfSlot
	entities := append([]Entity(nil), w.entities...)
	if built := w.BuildWornWeaponSpells(nil); built != 1 {
		t.Fatalf("built %d Spells, want 1", built)
	}
	if !w.equipment[0][0].SourceEquipment.Spell.Present || !reflect.DeepEqual(entities, w.entities) {
		t.Fatal("the build left the gap or changed an actor value")
	}
}

// TestHandOverBuildsTheWornWeaponSpell: a hand-over to the party closes the
// worn staff's gap; a hand-over to another player, the control, does not.
func TestHandOverBuildsTheWornWeaponSpell(t *testing.T) {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	item.Effects = staffEffects(14, 20)
	control := wornSpellWorldOf(t, item, 3)
	control.handOver(0, 2)
	if control.equipment[0][0].SourceEquipment.Spell.Present {
		t.Fatal("control: a hand-over to another player built a Spell")
	}
	w := wornSpellWorldOf(t, item, 3)
	w.handOver(0, SelfSlot)
	if w.entities[0].Owner != SelfSlot || !w.equipment[0][0].SourceEquipment.Spell.Present {
		t.Fatalf("the hand-over left the worn staff with Spell %+v", w.equipment[0][0].SourceEquipment.Spell)
	}
}

// TestBuildWornWeaponSpellsConstructsAPlainStaff: a plain worn item takes the
// constructor's operands, its definition row in its token, and the Spell.
func TestBuildWornWeaponSpellsConstructsAPlainStaff(t *testing.T) {
	plain := ItemInstance{Code: 0x0102, Kind: 2, WeightPresent: true, Weight: 2, Effects: staffEffects(6, 3)}
	w := wornSpellWorld(t, plain)
	id := w.equipment[0][0].ObjectID
	constructed := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	construct := func(item ItemInstance) ItemInstance {
		item.SourceEquipment = constructed.SourceEquipment
		return item
	}
	if built := w.BuildWornWeaponSpells(construct); built != 1 {
		t.Fatalf("built %d Spells, want 1", built)
	}
	row := w.savedObjects.item(id)
	want := constructed.SourceEquipment
	want.Spell = SourceItemSpell{Present: true, ID: 6, Range: 6, Defensive: 1, ManaCost: 5}
	if row.Value.SourceEquipment != want || row.Token.T0C != want.DefinitionRow || w.equipment[0][0].SourceEquipment != want {
		t.Fatalf("the plain staff became %+v", row.Value.SourceEquipment)
	}
	reloadOperations(t, w)
}

// TestBuildWornWeaponSpellsLeavesUnbuildableWeapons: no castSpell, spell id
// zero, an id the table lacks, a plain item without a constructor, and
// another player's actor keep their state.
func TestBuildWornWeaponSpellsLeavesUnbuildableWeapons(t *testing.T) {
	for _, tc := range []struct {
		name    string
		effects []ItemEffect
		plain   bool
		owner   uint32
	}{
		{"no castSpell", []ItemEffect{{Kind: 12, Operand: 2}}, false, SelfSlot},
		{"spell zero", staffEffects(0, 9), false, SelfSlot},
		{"unknown spell", staffEffects(15, 9), false, SelfSlot},
		{"plain without constructor", staffEffects(14, 9), true, SelfSlot},
		{"another player's actor", staffEffects(14, 9), false, 3},
	} {
		item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
		if tc.plain {
			item = ItemInstance{Code: 0x0102, Kind: 2, WeightPresent: true, Weight: 2}
		}
		item.Effects = tc.effects
		w := wornSpellWorldOf(t, item, tc.owner)
		before := mustMarshal(t, w)
		if built := w.BuildWornWeaponSpells(nil); built != 0 || !reflect.DeepEqual(before, mustMarshal(t, w)) {
			t.Fatalf("%s: built %d Spells or changed the World", tc.name, built)
		}
	}
}
