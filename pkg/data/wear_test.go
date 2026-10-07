package data_test

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
)

// wearRow builds a Shields or Armors row: a name and, for an armour, the
// Slot column (param 4) — with every other cell the format's own empty
// one, out to armorAbsorptionColumn (param 10), the widest column any
// resolver in this file reads off an Armors row since 0136. slot is only
// meaningful when the row is later parsed as Armors; a Shields row never
// has any of it read.
//
// IT IS 11 CELLS WIDE, NOT 5, SINCE 0136 (plan's own risk, "a fixture
// widening reaches another package's test file"): ArmorFromCode refuses a
// row this short of the absorption column outright, so a fixture built for
// one of THAT function's own accepting cases must reach at least as far.
// wearArmorRow is this same row with the two raw columns 0136 adds stated
// explicitly, for a test that reads Defence or Absorption.
func wearRow(name string, slot int32) synth.DataBinRow {
	return wearArmorRow(name, slot, -1, -1)
}

// wearArmorRow is wearRow widened with the two raw columns 0136 adds:
// defence (param 9) and absorption (param 10), UNSCALED — the value this
// contract reads before either factor is applied.
func wearArmorRow(name string, slot, defence, absorption int32) synth.DataBinRow {
	p := []int32{-1, -1, -1, -1, slot, -1, -1, -1, -1, defence, absorption}
	return synth.DataBinRow{Name: name, Params: p}
}

func shortArmorRow(name string, slot int32) synth.DataBinRow {
	return synth.DataBinRow{Name: name, Params: []int32{-1, -1, -1, -1, slot}}
}

// wearFactorRow builds a shape or material row carrying ONE factor at BOTH
// of the two factor-ladder slots 0136 reads — 6 (defence) and 7
// (absorption) — the same value at each. AC-1's own fixture needs exactly
// this: a shape and a material whose two ladder slots agree, so a difference
// between the resolved Defence and Absorption can only be the rounding rule
// and never a difference in the scale itself.
func wearFactorRow(name string, factor float64) synth.DataBinRow {
	d := make([]float64, 9)
	d[6], d[7] = factor, factor
	return synth.DataBinRow{Name: name, Doubles: d}
}

// wearTables parses one synthetic definition table and hands back the four
// collections ResolveShield and ResolveArmor read. It is wear_test.go's own
// helper and not weapon_test.go's tables — that file is an existing test and
// this story edits none of it — but it builds the identical shape of
// fixture (plan D-8: every collection is built in the test, from names this
// test invents, and none of them is a shipped one).
func wearTables(t *testing.T, shapes, materials, shields, armors []synth.DataBinRow) (data.ScaleTable, data.ScaleTable, data.Collection, data.Collection) {
	t.Helper()
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes:    shapes,
			synth.DataBinMaterials: materials,
			synth.DataBinShields:   shields,
			synth.DataBinArmors:    armors,
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return f.Collection(databin.Shapes), f.Collection(databin.Materials),
		f.Collection(databin.Shields), f.Collection(databin.Armors)
}

// wearScaleRow is scaleRow's own shape (weapon_test.go), rebuilt here because
// this file may not reach into that one: a nine-double record at every
// slot zero. ResolveShield and ResolveArmor read slots 6 and 7, so a row
// built with THIS helper resolves to zero Defence and Absorption. A test
// which reads either field builds a wearFactorRow instead, so the factor
// under test is never accidentally zero.
func wearScaleRow(name string) synth.DataBinRow {
	return synth.DataBinRow{Name: name, Doubles: make([]float64, 9)}
}

// TestResolveShieldComposesTheCodeAndCutsTheTrailingWord: a shape word, a
// material with neither Leather nor Wood in its name (so FR-2b never fires),
// and a residue ending " Shield" at a positive offset — the ordinary case,
// cut before the Shields lookup. Row and Code are checked against the raw
// indices this test worked out by hand, the same discipline weapon_test.go's
// own code test uses.
func TestResolveShieldComposesTheCodeAndCutsTheTrailingWord(t *testing.T) {
	shapes, materials, shields, _ := wearTables(t,
		[]synth.DataBinRow{wearScaleRow("Ornate")},
		[]synth.DataBinRow{wearScaleRow("Bronze")},
		[]synth.DataBinRow{wearRow("Round", 0)},
		nil)

	got, err := data.ResolveShield("Ornate Bronze Round Shield", shapes, materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	if got.Row != 1 {
		t.Errorf("Row = %d, want 1 — \"Round\" is the fixture's first written Shields row", got.Row)
	}
	const shieldClass = 2
	want := data.ItemCode(uint16(0)<<12 | uint16(shieldClass)<<8 | uint16(0)<<5 | uint16(1))
	if got.Code != want {
		t.Errorf("Code = 0x%04x, want 0x%04x (material=0 shape=0 class=%d row=1)",
			uint16(got.Code), uint16(want), shieldClass)
	}
}

func TestResolveShieldTheShieldCutRequiresAPositiveOffset(t *testing.T) {
	t.Run("a positive offset is cut", func(t *testing.T) {
		_, _, shields, _ := wearTables(t, nil, nil,
			[]synth.DataBinRow{wearRow("Kite", 0)}, nil)

		got, err := data.ResolveShield("Kite Shield", nil, nil, shields)
		if err != nil {
			t.Fatalf("ResolveShield: %v", err)
		}
		if got.Row != 1 {
			t.Errorf("Row = %d, want 1", got.Row)
		}
	})

	t.Run("no Shield word at all leaves the residue alone", func(t *testing.T) {
		_, _, shields, _ := wearTables(t, nil, nil,
			[]synth.DataBinRow{wearRow("Targe", 0)}, nil)

		got, err := data.ResolveShield("Targe", nil, nil, shields)
		if err != nil {
			t.Fatalf("ResolveShield: %v", err)
		}
		if got.Row != 1 {
			t.Errorf("Row = %d, want 1", got.Row)
		}
	})

	t.Run("offset zero is left alone, not cut", func(t *testing.T) {
		_, materials, shields, _ := wearTables(t, nil,
			[]synth.DataBinRow{wearScaleRow("Buckler")},
			[]synth.DataBinRow{wearRow(" Shield", 0)}, nil)

		got, err := data.ResolveShield("Buckler  Shield", nil, materials, shields)
		if err != nil {
			t.Fatalf("ResolveShield: %v — the offset-zero \" Shield\" must survive uncut", err)
		}
		if got.Row != 1 {
			t.Errorf("Row = %d, want 1 — the row named %q", got.Row, " Shield")
		}
	})
}

// TestResolveArmorAppliesTheImpliedShapeWord is FR-2b through ResolveArmor:
// a material name containing "Leather" reattaches "Soft ", and the residue
// must then name an Armors row under THAT full name — the row "Jerkin"
// alone would not do, because the implied word is part of what findByName
// matches, not a hint layered on afterward.
func TestResolveArmorAppliesTheImpliedShapeWord(t *testing.T) {
	_, materials, _, armors := wearTables(t, nil,
		[]synth.DataBinRow{wearScaleRow("Aged Leather")},
		nil,
		[]synth.DataBinRow{wearRow("Soft Jerkin", 5)})

	got, err := data.ResolveArmor("Aged Leather Jerkin", nil, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	if got.Row != 1 {
		t.Errorf("Row = %d, want 1", got.Row)
	}
	if got.Slot != 5 {
		t.Errorf("Slot = %d, want 5", got.Slot)
	}
}

func TestResolveWeaponNeverAppliesTheImpliedShapeWord(t *testing.T) {
	materials := []synth.DataBinRow{{Name: "Pale Wood", Doubles: make([]float64, 9)}}
	weapons := []synth.DataBinRow{{Name: "Axe", Params: []int32{-1, -1, -1, -1, -1, 1, 4, 6, 0, 0, -1, -1, 8, 4, -1}}}

	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinMaterials: materials,
			synth.DataBinWeapons:   weapons,
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	m, w := f.Collection(databin.Materials), f.Collection(databin.Weapons)

	got, err := data.ResolveWeapon("Pale Wood Axe", nil, m, w)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v — a weapon must never take FR-2b's implied shape word", err)
	}
	if got.Row != 1 {
		t.Errorf("Row = %d, want 1", got.Row)
	}
}

func TestResolveShieldRefusesAResidueNamingNoRow(t *testing.T) {
	_, _, shields, _ := wearTables(t, nil, nil,
		[]synth.DataBinRow{wearRow("Round", 0)}, nil)

	if _, err := data.ResolveShield("Kite Shield", nil, nil, shields); err == nil {
		t.Fatal("ResolveShield accepted a name no row answers")
	}
}

func TestResolveArmorRefusesAResidueNamingNoRow(t *testing.T) {
	_, _, _, armors := wearTables(t, nil, nil, nil,
		[]synth.DataBinRow{wearRow("Jerkin", 5)})

	if _, err := data.ResolveArmor("Cuirass", nil, nil, armors); err == nil {
		t.Fatal("ResolveArmor accepted a name no row answers")
	}
}

// TestResolveArmorRefusesARowTooShortForSlot: a row that carries fewer than
// five cells has no param 4 to read, and this is refused rather than
// indexed past — the same discipline ResolveWeapon already applies to its
// own longer reach.
func TestResolveArmorRefusesARowTooShortForSlot(t *testing.T) {
	_, _, _, armors := wearTables(t, nil, nil, nil,
		[]synth.DataBinRow{{Name: "Stub", Params: []int32{1, 2, 3}}})

	if _, err := data.ResolveArmor("Stub", nil, nil, armors); err == nil {
		t.Fatal("ResolveArmor accepted a row shorter than the slot cell it reads")
	}
}

func TestResolveArmorNeverRefusesOnSlotAlone(t *testing.T) {
	_, _, _, armors := wearTables(t, nil, nil, nil, []synth.DataBinRow{
		wearRow("Broken Cap", 0),
		wearRow("Ethereal Cap", 13),
		wearRow("Cap", 7),
	})

	for _, tc := range []struct {
		name      string
		wantSlot  int32
		wantClass int
	}{
		{"Broken Cap", 0, 0},
		{"Ethereal Cap", 13, 0}, // 13 fits four bits; must still be forced to 0
		{"Cap", 7, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := data.ResolveArmor(tc.name, nil, nil, armors)
			if err != nil {
				t.Fatalf("ResolveArmor(%q): %v — a Slot out of range must not be refused", tc.name, err)
			}
			if got.Slot != tc.wantSlot {
				t.Errorf("Slot = %d, want %d", got.Slot, tc.wantSlot)
			}
			if got.Code.B() != tc.wantClass {
				t.Errorf("Code.B() = %d, want %d", got.Code.B(), tc.wantClass)
			}
		})
	}
}

func TestResolveArmorRoundsDefenceUpAndAbsorptionDown(t *testing.T) {
	_, materials, _, armors := wearTables(t, nil,
		[]synth.DataBinRow{wearFactorRow("Heavy", 1.5)},
		nil,
		[]synth.DataBinRow{wearArmorRow("Cap", 7, 3, 3)})

	got, err := data.ResolveArmor("Heavy Cap", nil, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	// product = 3 * 1.5 = 4.5: defence = trunc(4.5+0.5) = 5, absorption =
	// trunc(4.5) = 4.
	if got.Defence != 5 {
		t.Errorf("Defence = %d, want 5 (trunc(3*1.5+0.5))", got.Defence)
	}
	if got.Absorption != 4 {
		t.Errorf("Absorption = %d, want 4 (trunc(3*1.5), no half added)", got.Absorption)
	}
	if got.Defence != got.Absorption+1 {
		t.Errorf("Defence = %d, Absorption = %d, want Defence exactly one above Absorption",
			got.Defence, got.Absorption)
	}
}

func TestResolveShieldRoundsDefenceAndTruncatesAbsorption(t *testing.T) {
	_, materials, shields, _ := wearTables(t, nil,
		[]synth.DataBinRow{wearFactorRow("Heavy", 1.5)},
		[]synth.DataBinRow{wearArmorRow("Ward", 0, 3, 3)}, nil)
	got, err := data.ResolveShield("Heavy Ward", nil, materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	if got.Defence != 5 || got.Absorption != 4 {
		t.Fatalf("shield Defence/Absorption = %d/%d, want 5/4", got.Defence, got.Absorption)
	}
	byCode, err := data.ShieldFromCode(got.Code, nil, materials, shields)
	if err != nil {
		t.Fatalf("ShieldFromCode: %v", err)
	}
	if byCode.Defence != got.Defence || byCode.Absorption != got.Absorption {
		t.Fatalf("ShieldFromCode protection = %d/%d, want %d/%d", byCode.Defence, byCode.Absorption,
			got.Defence, got.Absorption)
	}
}

// TestResolveArmorAnIntegerProductRoundsDefenceAndAbsorptionEqual is AC-2:
// a scaled product that lands exactly on an integer leaves nothing for the
// added half to move past — defence and absorption come out equal.
func TestResolveArmorAnIntegerProductRoundsDefenceAndAbsorptionEqual(t *testing.T) {
	_, materials, _, armors := wearTables(t, nil,
		[]synth.DataBinRow{wearFactorRow("Heavy", 0.5)},
		nil,
		[]synth.DataBinRow{wearArmorRow("Cap", 7, 4, 4)})

	got, err := data.ResolveArmor("Heavy Cap", nil, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	// product = 4 * 0.5 = 2.0 exactly: defence = trunc(2.5) = 2, absorption
	// = trunc(2.0) = 2.
	if got.Defence != 2 || got.Absorption != 2 {
		t.Errorf("Defence=%d Absorption=%d, want both 2 (an exact integer product)", got.Defence, got.Absorption)
	}
}

// TestArmorFromCodeAgreesWithResolveArmorFieldForField is AC-3: a code
// composed for an armour row, resolved from the code alone, answers the same
// piece the row's own name resolves to — every field but Name, which a
// bare code carries no literal for (WeaponFromCode's own doc states the
// identical exception for a weapon).
func TestArmorFromCodeAgreesWithResolveArmorFieldForField(t *testing.T) {
	shapes, materials, _, armors := wearTables(t,
		[]synth.DataBinRow{wearFactorRow("Ornate", 1.2)},
		[]synth.DataBinRow{wearFactorRow("Bronze", 0.8)},
		nil,
		[]synth.DataBinRow{wearArmorRow("Cap", 7, 5, 2)})

	byName, err := data.ResolveArmor("Ornate Bronze Cap", shapes, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	if byName.Name == "" {
		t.Fatal("ResolveArmor answered an empty Name, which would make this comparison vacuous")
	}

	byCode, err := data.ArmorFromCode(byName.Code, shapes, materials, armors)
	if err != nil {
		t.Fatalf("ArmorFromCode: %v", err)
	}
	byCode.Name = byName.Name // Name apart — everything else compared as is.
	if byCode != byName {
		t.Errorf("ArmorFromCode(byName.Code) = %+v, want %+v (Name apart)", byCode, byName)
	}
}

func TestArmorFromCodeRefusesEveryFR5Case(t *testing.T) {
	_, _, _, armors := wearTables(t, nil, nil, nil, []synth.DataBinRow{
		shortArmorRow("Stub", 7),
		wearArmorRow("ZeroSlot", 0, 1, 1),
		wearArmorRow("Slot13", 13, 1, 1),
	})

	const (
		weaponClass = 1
		shieldClass = 2
		armourClass = 7 // any B outside {weaponClass, shieldClass} and inside 1..12
	)
	code := func(class, row int) data.ItemCode {
		return data.ItemCode(uint16(class)<<8 | uint16(row))
	}

	for _, tc := range []struct {
		name string
		code data.ItemCode
	}{
		{"a weapon's code", code(weaponClass, 1)},
		{"a shield's code", code(shieldClass, 1)},
		{"a code naming no written Armors row", code(armourClass, 0)},
		{"a row too short for the absorption column", code(armourClass, 1)},
		{"a row stating Slot 0", code(armourClass, 2)},
		{"a row stating Slot 13", code(armourClass, 3)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := data.ArmorFromCode(tc.code, nil, nil, armors)
			if err == nil {
				t.Fatalf("ArmorFromCode(0x%04x) accepted, want refused — got %+v", uint16(tc.code), got)
			}
			if got != (data.Armor{}) {
				t.Errorf("ArmorFromCode(0x%04x) returned %+v on refusal, want the zero value",
					uint16(tc.code), got)
			}
		})
	}
}
