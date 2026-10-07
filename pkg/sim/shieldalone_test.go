package sim

import (
	"reflect"
	"testing"
)

func sourceShieldItem() ItemInstance {
	return ItemInstance{Code: 0x0201, WeightPresent: true, Weight: 3,
		SourceEquipment: SourceEquipment{Class: SourceShield, DefinitionRow: 1, Defence: [22]byte{7}}}
}

// sourceShieldWorld is an actor loaded from an original save whose pack holds
// one shield and whose slot 1 holds worn.
func sourceShieldWorld(t *testing.T, worn ItemInstance) *World {
	t.Helper()
	w := sourceMutationWorld(t, sourceShieldItem())
	w.equipment[0][0] = worn
	e := &w.entities[0]
	e.ActorLoad.Source.EquipmentRuntimePresent = true
	e.ActorLoad.OwnWeight, e.ActorLoad.Accumulator = int16(worn.Weight), 3
	e.Reach = 4
	return w
}

// A shield put on by an actor loaded from an original save stands alone. A
// weapon that fills both hands is taken off into the pack first, before the
// shield is stored; a weapon that leaves a hand free stays worn.
func TestSourceShieldStandsAloneAndTakesAWeaponFillingBothHandsOff(t *testing.T) {
	shield := sourceShieldItem()
	oneHanded := sourceEquipmentWeapon(0x0101, 5, 11, 3, 4, 1)
	twoHanded := sourceEquipmentWeapon(0x0102, 5, 11, 3, 4, 2)
	for _, tc := range []struct {
		name     string
		worn     ItemInstance
		wantSlot uint16
		wantPack []uint16
		want     []SourceEquipmentEvent
	}{
		{name: "bare", want: []SourceEquipmentEvent{
			receiptEvent(SourceEquipmentTake, 1, SourceEquipmentPack, 0, false),
			receiptEvent(SourceEquipmentPut, 1, SourceEquipmentWorn, 1, false),
		}},
		{name: "one-handed", worn: oneHanded, wantSlot: oneHanded.Code, want: []SourceEquipmentEvent{
			receiptEvent(SourceEquipmentTake, 1, SourceEquipmentPack, 0, false),
			receiptEvent(SourceEquipmentPut, 1, SourceEquipmentWorn, 1, false),
		}},
		{name: "two-handed", worn: twoHanded, wantPack: []uint16{twoHanded.Code}, want: []SourceEquipmentEvent{
			receiptEvent(SourceEquipmentTake, 1, SourceEquipmentPack, 0, false),
			receiptEvent(SourceEquipmentTake, 2, SourceEquipmentWorn, 0, false),
			{Kind: SourceEquipmentSpellWrite, Ticket: 2},
			receiptEvent(SourceEquipmentPut, 2, SourceEquipmentPack, 0, false),
			receiptEvent(SourceEquipmentPut, 1, SourceEquipmentWorn, 1, false),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := sourceShieldWorld(t, tc.worn)
			var r SourceEquipmentReceipt
			if !w.EquipSourceCarried(1, 0, SourceEquipmentOperation{Receipt: &r}) {
				t.Fatal("the shield was refused")
			}
			if got := w.equipment[0][0].Code; got != tc.wantSlot {
				t.Errorf("slot 1 = %#x, want %#x", got, tc.wantSlot)
			}
			if got := w.equipment[0][1].Code; got != shield.Code {
				t.Errorf("slot 2 = %#x, want the shield %#x", got, shield.Code)
			}
			var pack []uint16
			for _, stack := range w.carried[0] {
				pack = append(pack, stack.Code)
			}
			if !reflect.DeepEqual(pack, tc.wantPack) {
				t.Errorf("pack = %#x, want %#x", pack, tc.wantPack)
			}
			if !reflect.DeepEqual(r.Events, tc.want) {
				t.Errorf("receipt = %+v, want %+v", r.Events, tc.want)
			}
		})
	}
}

// Taking a weapon off, into the pack or out to the caller, leaves a worn
// shield where it is.
func TestSourceUnequippingWeaponLeavesTheShieldWorn(t *testing.T) {
	shield := sourceShieldItem()
	weapon := sourceEquipmentWeapon(0x0101, 5, 11, 3, 4, 1)
	for _, toPack := range []bool{true, false} {
		w := sourceShieldWorld(t, weapon)
		w.carried[0] = nil
		w.equipment[0][1] = shield
		removed, ok := w.UnequipSource(1, 1, toPack)
		if !ok || removed.Code != weapon.Code {
			t.Fatalf("toPack=%v: UnequipSource = %#x, %v", toPack, removed.Code, ok)
		}
		if !w.equipment[0][0].Empty() || w.equipment[0][1].Code != shield.Code {
			t.Errorf("toPack=%v: worn = %#x/%#x, want no weapon and the shield", toPack, w.equipment[0][0].Code, w.equipment[0][1].Code)
		}
		if want := map[bool]int{true: 1, false: 0}[toPack]; len(w.carried[0]) != want {
			t.Errorf("toPack=%v: pack holds %d stacks, want %d", toPack, len(w.carried[0]), want)
		}
	}
}
