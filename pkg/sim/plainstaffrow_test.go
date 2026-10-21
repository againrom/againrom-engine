package sim

import "testing"

// plainRowStaffWorld: a source actor's pack holds a staff, its row plain or
// constructed.
func plainRowStaffWorld(t *testing.T, constructed bool) *World {
	t.Helper()
	plain := ItemInstance{Code: 0x0102, Kind: 2, Effects: staffEffects(6, 3)}
	built := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	stored := plain
	if constructed {
		stored = constructItemWith([]ItemWeight{{Code: 0x0102, Weight: 2, Constructor: built.SourceEquipment}}, plain)
	}
	w := sourceMutationWorld(t, stored)
	if err := w.DeclareItemWeights([]ItemWeight{{Code: 0x0102, Weight: 2, Constructor: built.SourceEquipment}}); err != nil {
		t.Fatal(err)
	}
	w.spells = []SpellRule{{ID: 6, MaxRange: 6, ManaCost: 5, Defensive: true}}
	w.entities[0].ActorLoad.Source.EquipmentRuntimePresent = true
	w.recomputeLoad(0)
	bindOperationsPack(t, w, 0)
	return w
}

// A plain row is worn as the constructed control is (ITEM-SPELLMOVE-132).
func TestSourceEquipBindsAPlainStaffRow(t *testing.T) {
	var worn [2]ItemInstance
	for k, constructed := range []bool{true, false} {
		w := plainRowStaffWorld(t, constructed)
		if w.entities[0].ActorLoad.Source.Class == 0 {
			t.Fatal("the actor is not on the source route")
		}
		if !w.EquipSourceCarried(1, 0) {
			t.Fatalf("constructed row %v: the equip was refused", constructed)
		}
		got := w.equipment[0][0]
		if got.Code != 0x0102 || !got.SourceEquipment.Spell.Present || got.SourceEquipment.Spell.ID != 6 || len(w.carried[0]) != 0 {
			t.Fatalf("constructed row %v: the staff was not worn: %+v", constructed, got)
		}
		row := w.savedObjects.item(got.ObjectID)
		if row == nil || row.Token.T0C != got.SourceEquipment.DefinitionRow || !StackStateEqual(row.Value, StackItem(got, 1)) {
			t.Fatalf("constructed row %v: the registry row differs from the worn staff", constructed)
		}
		cold := reloadOperations(t, w)
		if cold.equipment[0][0].SourceEquipment != got.SourceEquipment {
			t.Fatalf("constructed row %v: the native continuation lost the worn staff", constructed)
		}
		got.ObjectID = 0
		worn[k] = got
	}
	if !StackStateEqual(StackItem(worn[0], 1), StackItem(worn[1], 1)) {
		t.Fatalf("a plain row wears %+v, a constructed row %+v", worn[1], worn[0])
	}
}
