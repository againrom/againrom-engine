package data_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
)

// FoldWear's own tests: AC-5, AC-8 (SC-3, SC-7, SC-9). Every fixture is
// built in test code, from names this file invents (wearTables,
// wearArmorRow — wear_test.go's own helpers, this file's own package).

// TestFoldWearSumsArmourAloneAndZeroesProtectionAndResistance is AC-5
// (SC-3): a worn set holding two armour pieces and a weapon's own code folds
// to the two pieces' defence sum and absorption sum, every protection and
// resistance term at zero, and every other EquipMod field untouched — the
// weapon's own code contributes nothing, because ArmorFromCode refuses it
// exactly as the equip gate would.
func TestFoldWearSumsArmourAloneAndZeroesProtectionAndResistance(t *testing.T) {
	shapes, materials, _, armors := wearTables(t, nil, nil, nil, []synth.DataBinRow{
		wearArmorRow("Boots", 4, 3, 1),
		wearArmorRow("Cap", 7, 5, 2),
	})
	boots, err := data.ResolveArmor("Boots", shapes, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor(Boots): %v", err)
	}
	helm, err := data.ResolveArmor("Cap", shapes, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor(Cap): %v", err)
	}

	var e data.Equipment
	e.SetCode(4, boots.Code)
	e.SetCode(7, helm.Code)
	// A weapon's own code (field B = 1), in a slot neither piece uses: it
	// names no armour at all, so it must not move either sum.
	e.SetCode(1, data.ItemCode(1<<8|1))

	mod := data.FoldWear(e, shapes, materials, nil, armors)

	wantDefence := boots.Defence + helm.Defence
	wantAbsorption := boots.Absorption + helm.Absorption
	if mod.Defence != wantDefence || mod.Absorption != wantAbsorption {
		t.Errorf("Defence=%d Absorption=%d, want %d/%d — a weapon's code must not contribute",
			mod.Defence, mod.Absorption, wantDefence, wantAbsorption)
	}
	if mod.Protection != ([5]int32{}) || mod.Resistance != ([5]int32{}) {
		t.Errorf("Protection=%v Resistance=%v, want both the zero array (FR-6a)",
			mod.Protection, mod.Resistance)
	}
	if mod.ToHit != 0 || mod.DamageBase != 0 || mod.DamageSpread != 0 {
		t.Errorf("ToHit/DamageBase/DamageSpread = %d/%d/%d, want all zero — "+
			"the fold touches Defence and Absorption alone (FR-6)",
			mod.ToHit, mod.DamageBase, mod.DamageSpread)
	}
}

// TestFoldWearDefenceAndAbsorptionSumsLandOnTheirOwnField is SC-7: a worn
// set whose defence sum and absorption sum are DIFFERENT numbers puts each
// on its own EquipMod field — so a swap of the two assignments inside
// FoldWear would fail this test where a fixture with equal sums could not.
func TestFoldWearDefenceAndAbsorptionSumsLandOnTheirOwnField(t *testing.T) {
	shapes, materials, _, armors := wearTables(t, nil, nil, nil, []synth.DataBinRow{
		wearArmorRow("Boots", 4, 3, 1),
		wearArmorRow("Cap", 7, 5, 2),
	})
	boots, err := data.ResolveArmor("Boots", shapes, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor(Boots): %v", err)
	}
	helm, err := data.ResolveArmor("Cap", shapes, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor(Cap): %v", err)
	}

	var e data.Equipment
	e.SetCode(4, boots.Code)
	e.SetCode(7, helm.Code)

	mod := data.FoldWear(e, shapes, materials, nil, armors)

	wantDefence := boots.Defence + helm.Defence
	wantAbsorption := boots.Absorption + helm.Absorption
	if wantDefence == wantAbsorption {
		t.Fatal("fixture defence and absorption sums coincide; SC-7 needs them to differ " +
			"so a swapped assignment would be caught")
	}
	if mod.Defence != wantDefence {
		t.Errorf("Defence = %d, want %d", mod.Defence, wantDefence)
	}
	if mod.Absorption != wantAbsorption {
		t.Errorf("Absorption = %d, want %d", mod.Absorption, wantAbsorption)
	}
}

// TestFoldWearContributionDoesNotDependOnWhichSlotCarriesTheCode is AC-8
// (SC-9): the same code, resolved once as though worn in slot 4 and once
// in slot 12, contributes the identical pair either way (FR-7a) — FoldWear
// reads a code's own fields through ArmorFromCode, never the array index
// the code was read from.
func TestFoldWearContributionDoesNotDependOnWhichSlotCarriesTheCode(t *testing.T) {
	shapes, materials, _, armors := wearTables(t, nil, nil, nil,
		[]synth.DataBinRow{wearArmorRow("Cap", 7, 5, 2)})
	piece, err := data.ResolveArmor("Cap", shapes, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor(Cap): %v", err)
	}

	var inFour data.Equipment
	inFour.SetCode(4, piece.Code)
	var inTwelve data.Equipment
	inTwelve.SetCode(12, piece.Code)

	gotFour := data.FoldWear(inFour, shapes, materials, nil, armors)
	gotTwelve := data.FoldWear(inTwelve, shapes, materials, nil, armors)

	if gotFour != gotTwelve {
		t.Errorf("FoldWear differs by slot: slot4=%+v slot12=%+v, want identical (FR-7a)",
			gotFour, gotTwelve)
	}
	if gotFour.Defence != piece.Defence || gotFour.Absorption != piece.Absorption {
		t.Errorf("FoldWear = %+v, want Defence=%d Absorption=%d",
			gotFour, piece.Defence, piece.Absorption)
	}
}

func TestFoldWearAddsShieldAndArmourProtectionTogether(t *testing.T) {
	shapes, materials, shields, armors := wearTables(t, nil, nil,
		[]synth.DataBinRow{wearArmorRow("Ward", 0, 7, 3)},
		[]synth.DataBinRow{wearArmorRow("Cap", 7, 5, 2)})
	shield, err := data.ResolveShield("Ward", shapes, materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	armour, err := data.ResolveArmor("Cap", shapes, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	var e data.Equipment
	e.SetCode(2, shield.Code)
	e.SetCode(7, armour.Code)
	got := data.FoldWear(e, shapes, materials, shields, armors)
	if got.Defence != shield.Defence+armour.Defence || got.Absorption != shield.Absorption+armour.Absorption {
		t.Fatalf("FoldWear = %+v, want shield+armour Defence/Absorption %d/%d", got,
			shield.Defence+armour.Defence, shield.Absorption+armour.Absorption)
	}
}
