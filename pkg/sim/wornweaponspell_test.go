package sim

import "testing"

// wornSpellWorld is a party actor without source operands that carries one
// saved spell weapon and equips it on the native saved-object route.
func wornSpellWorld(t *testing.T, item ItemInstance) *World {
	t.Helper()
	return wornSpellWorldOf(t, item, SelfSlot)
}

func wornSpellWorldOf(t *testing.T, item ItemInstance, owner uint32) *World {
	t.Helper()
	return wornSpellWorldWith(t, item, owner, nil)
}

// wornSpellWorldWith is wornSpellWorldOf with weights declared before the
// equip, so a plain item finds its constructor.
func wornSpellWorldWith(t *testing.T, item ItemInstance, owner uint32, weights []ItemWeight) *World {
	t.Helper()
	w := mustWorld(t, 1, Bounds{8, 8}, []Entity{{ID: 7, X: 1, Y: 1, HP: 20, MaxHP: 20, Owner: owner}, {ID: 8, X: 2, Y: 1, HP: 20, MaxHP: 20, Owner: owner}})
	if err := w.DeclareItemWeights(weights); err != nil {
		t.Fatal(err)
	}
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
// (ITEM-SPELLMOVE-132) and changes no actor value beyond the equip; a staff
// without a castSpell Effect, the control, builds none.
func TestNativeEquipBuildsTheWornWeaponSpell(t *testing.T) {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	item.Effects = []ItemEffect{{Kind: 12, Operand: 2}}
	control := wornSpellWorld(t, item)
	if id := control.equipment[0][0].ObjectID; control.savedObjects.item(id).Spell != 0 || control.equipment[0][0].SourceEquipment.Spell.Present {
		t.Fatal("control: a staff without a castSpell Effect built a Spell")
	}
	item.Effects = staffEffects(14, 20)
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
	if w.equipWornSpell(0); w.savedObjects.item(id).Spell != first {
		t.Fatal("a held Spell was built again")
	}
	cold := reloadOperations(t, w)
	if cold.savedObjects.item(id).Spell != first || cold.equipment[0][0].SourceEquipment.Spell != want {
		t.Fatal("native SAVE/LOAD lost the Spell")
	}
}

// wornGapWorld is wornSpellWorldOf with the worn staff's Spell deleted, the
// state a loaded record without one holds.
func wornGapWorld(t *testing.T, item ItemInstance, owner uint32) *World {
	t.Helper()
	w := wornSpellWorldOf(t, item, owner)
	worn := w.equipment[0][0].Clone()
	if !w.replaceObjectSpell(&worn, SourceItemSpell{}) || w.equipment[0][0].SourceEquipment.Spell.Present {
		t.Fatal("the worn staff keeps its Spell")
	}
	return w
}

// TestALoadedGapStaysThroughSaveAndLoad: a party actor whose worn classed
// staff holds no Spell, as a loaded record can, keeps the gap through SAVE and
// LOAD and does not cast (ITEM-187, MAGIC-379); a re-equip builds it
// (TestASpelllessSpellWeaponWornAgainCasts). The same staff holding its
// Spell, the control, casts.
func TestALoadedGapStaysThroughSaveAndLoad(t *testing.T) {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	item.Effects = staffEffects(14, 20)
	if control := wornSpellWorld(t, item); !control.wornWeaponHoldsSpell(control.entities[0]) {
		t.Fatal("control: the staff holding its Spell does not cast")
	}
	w := wornGapWorld(t, item, SelfSlot)
	cold := reloadOperations(t, w)
	if cold.equipment[0][0].SourceEquipment.Spell.Present || cold.savedObjects.item(cold.equipment[0][0].ObjectID).Spell != 0 {
		t.Fatal("SAVE and LOAD built the worn staff's Spell")
	}
	if _, cast := cold.weaponSpell(cold.entities[0]); cast {
		t.Fatal("the loaded gap casts")
	}
}

// TestHandOverBuildsTheWornWeaponSpell: a hand-over to the party closes the
// worn staff's gap; a hand-over to another player, the control, does not.
func TestHandOverBuildsTheWornWeaponSpell(t *testing.T) {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	item.Effects = staffEffects(14, 20)
	control := wornGapWorld(t, item, 3)
	control.handOver(0, 2)
	if control.equipment[0][0].SourceEquipment.Spell.Present {
		t.Fatal("control: a hand-over to another player built a Spell")
	}
	w := wornGapWorld(t, item, 3)
	w.handOver(0, SelfSlot)
	if w.entities[0].Owner != SelfSlot || !w.equipment[0][0].SourceEquipment.Spell.Present {
		t.Fatalf("the hand-over left the worn staff with Spell %+v", w.equipment[0][0].SourceEquipment.Spell)
	}
}

// TestEquipConstructsAPlainStaff: an equip of a plain staff gives it the
// constructor's operands, its definition row in its token, and the Spell;
// without a constructor, the control, it stays plain and holds none.
func TestEquipConstructsAPlainStaff(t *testing.T) {
	plain := ItemInstance{Code: 0x0102, Kind: 2, WeightPresent: true, Weight: 2, Effects: staffEffects(6, 3)}
	if control := wornSpellWorld(t, plain); control.equipment[0][0].SourceEquipment.Class != 0 || control.equipment[0][0].SourceEquipment.Spell.Present {
		t.Fatal("control: a plain staff without a constructor was built")
	}
	constructed := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	w := wornSpellWorldWith(t, plain, SelfSlot, []ItemWeight{{Code: 0x0102, Weight: 2, Constructor: constructed.SourceEquipment}})
	id := w.equipment[0][0].ObjectID
	row := w.savedObjects.item(id)
	want := constructed.SourceEquipment
	want.Spell = SourceItemSpell{Present: true, ID: 6, Range: 6, Defensive: 1, ManaCost: 5}
	if row.Value.SourceEquipment != want || row.Token.T0C != want.DefinitionRow || w.equipment[0][0].SourceEquipment != want {
		t.Fatalf("the plain staff became %+v", row.Value.SourceEquipment)
	}
	reloadOperations(t, w)
}

// TestEquipLeavesUnbuildableWeapons: no castSpell, spell id zero and an id
// the table lacks build no Spell on equip.
func TestEquipLeavesUnbuildableWeapons(t *testing.T) {
	for _, tc := range []struct {
		name    string
		effects []ItemEffect
	}{
		{"no castSpell", []ItemEffect{{Kind: 12, Operand: 2}}},
		{"spell zero", staffEffects(0, 9)},
		{"unknown spell", staffEffects(15, 9)},
	} {
		item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
		item.Effects = tc.effects
		w := wornSpellWorld(t, item)
		if w.equipment[0][0].SourceEquipment.Spell.Present || w.savedObjects.item(w.equipment[0][0].ObjectID).Spell != 0 {
			t.Fatalf("%s: the equip built a Spell", tc.name)
		}
	}
}
