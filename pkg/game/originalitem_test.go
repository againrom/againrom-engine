package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestOriginalThreeCountPotionKeepsItsEffectThroughPartyImport(t *testing.T) {
	piece := sav.Piece{
		Class: "Item", Code: 0x0e06, Row: 6, Stack: 3, Kind: 3, Price: 250,
		Effects:                 []sav.ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00064}},
		UnsupportedEffectStates: []uint8{12},
	}
	report := &RestoredParty{}
	worn, carried := restoredItemLoadout(sav.Character{Items: []sav.Piece{piece}}, report)
	if !reflect.DeepEqual(worn, [sim.EquipSlots]sim.ItemInstance{}) || len(carried) != 3 {
		t.Fatalf("restored loadout worn=%+v carried=%+v", worn, carried)
	}
	want := sim.ItemInstance{Code: 0x0e06, Kind: 3, Price: 250, WeightPresent: true,
		Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00064}}}
	for i := range carried {
		if !reflect.DeepEqual(carried[i], want) {
			t.Fatalf("carried[%d] = %+v, want %+v", i, carried[i], want)
		}
	}
	carried[0].Effects[0].Operand = 0
	if carried[1].Effects[0].Operand != 0x03c00064 || carried[2].Effects[0].Operand != 0x03c00064 {
		t.Fatal("split original-save stack shares its effect slice")
	}
	if report.Carried != 3 || report.UnsupportedItemEffects != 1 {
		t.Fatalf("restore report = %+v, want three carried units and one unsupported effect", report)
	}
}

// DIV-594
func TestOriginalPartyLastEffectDivertingChangesEquipAndStackEquality(t *testing.T) {
	// DROP: a Weapon whose only Effect reference diverts at state 0 (the
	// Effect_DirectDamage shape SAV-EQUIPEFFECT-553 names) restores to its own
	// slot with an empty Effects list and SourceEquipment.EffectsUnsupported
	// set, which sourceEquipmentSlot then refuses for source equip.
	weapon := sav.Piece{Class: "Weapon", Code: 0x0103, Stack: 1, Kind: 2, Price: -981,
		UnsupportedEffectStates: []uint8{0}}
	report := &RestoredParty{}
	worn, _ := restoredItemLoadout(sav.Character{Worn: []sav.Piece{weapon}}, report)
	if worn[0].Code != 0x0103 || worn[0].HasEnchantment() {
		t.Fatalf("weapon not restored to its own slot with an empty Effects list: %+v", worn[0])
	}
	if !worn[0].SourceEquipment.EffectsUnsupported {
		t.Fatalf("SourceEquipment.EffectsUnsupported not set for a Weapon whose only Effect reference diverted: %+v", worn[0].SourceEquipment)
	}
	if report.UnsupportedItemEffects != 1 {
		t.Fatalf("UnsupportedItemEffects = %d, want 1", report.UnsupportedItemEffects)
	}

	// STACK-MERGE: two same-code, same-price plain items, one whose only
	// Effect reference diverted (so it restores with an empty Effects list,
	// the same as a never-enchanted item) and one that never carried an
	// Effect at all, are indistinguishable to sim.ItemEqual today.
	wasEnchanted := sav.Piece{Class: "Item", Code: 0x0e07, Stack: 1, Kind: 1, Price: 10,
		UnsupportedEffectStates: []uint8{0}}
	neverEnchanted := sav.Piece{Class: "Item", Code: 0x0e07, Stack: 1, Kind: 1, Price: 10}
	report2 := &RestoredParty{}
	_, carried := restoredItemLoadout(sav.Character{Items: []sav.Piece{wasEnchanted, neverEnchanted}}, report2)
	if len(carried) != 2 || carried[0].HasEnchantment() || carried[1].HasEnchantment() {
		t.Fatalf("carried = %+v, want two plain (unenchanted) items", carried)
	}
	if !sim.ItemEqual(carried[0], carried[1]) {
		t.Fatalf("carried items %+v and %+v were not found stack-equal, contradicting the debt this test pins", carried[0], carried[1])
	}
}
