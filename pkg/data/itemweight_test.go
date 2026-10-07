package data_test

// The weight column, the ladder slot it is scaled by, and the three-way
// dispatch that answers for any item code.
//
// Every expected weight here is worked out in the comment beside it from the
// claim's own arithmetic — column x shape x material, rounded with a half and
// truncated toward zero — and written as a literal. A test that multiplied the
// same three cells it fed in would pass against any column index.

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/databin"
)

// The runtime columns and ladder slots this file writes into its fixtures.
// They are spelled here rather than imported because a test asserting that
// production reads column 3 must not read the constant that says 3.
const (
	fixtureWeightColumn   = 3 // Armors/Shields/Weapons runtime column 3
	fixtureWeightSlot     = 3 // Shapes/Materials ladder slot 3
	fixtureArmorSlotCell  = 4 // the Armors row's own equipment slot
	fixtureOtherColumn    = 2 // a neighbouring column, filled to catch an off-by-one
	fixtureOtherLadder    = 2 // a neighbouring ladder slot, same purpose
	fixtureArmorColumnMax = 10
)

// wtScaleRow builds a Shapes or Materials row carrying a weight factor at slot
// 3 and a different value at slot 2, so a build reading the neighbouring slot
// produces a different number rather than the same one.
func wtScaleRow(name string, weight float64) synth.DataBinRow {
	d := make([]float64, 9)
	d[fixtureOtherLadder] = weight * 100
	d[fixtureWeightSlot] = weight
	return synth.DataBinRow{Name: name, Doubles: d}
}

// wtWeaponRow builds a Weapons row whose weight column is weight and whose
// neighbouring column is a different number.
func wtWeaponRow(name string, weight int32) synth.DataBinRow {
	p := make([]int32, 15)
	for i := range p {
		p[i] = -1
	}
	p[fixtureOtherColumn] = weight * 100
	p[fixtureWeightColumn] = weight
	return synth.DataBinRow{Name: name, Params: p}
}

// wtShieldRow is wtWeaponRow for a Shields row: the same column, the same
// neighbour.
func wtShieldRow(name string, weight int32) synth.DataBinRow {
	return wtWeaponRow(name, weight)
}

// wtArmorRow is wtWeaponRow for an Armors row, which additionally has to carry
// its own equipment slot cell or the fill refuses it.
func wtArmorRow(name string, weight, slot int32) synth.DataBinRow {
	p := make([]int32, fixtureArmorColumnMax+1)
	p[fixtureOtherColumn] = weight * 100
	p[fixtureWeightColumn] = weight
	p[fixtureArmorSlotCell] = slot
	return synth.DataBinRow{Name: name, Params: p}
}

// wtTables parses one synthetic definition table carrying all five collections
// the dispatch can reach.
//
// Armors, Shields and Weapons are one-based collections: the writer allocates
// entry 0 and never writes it, so the single row given to each lands at index
// 1 and is what row 1 of an item code names.
func wtTables(t *testing.T) (shapes, materials data.ScaleTable, armors, shields, weapons data.Collection) {
	t.Helper()
	f, err := databin.Parse(synth.DataBin{
		Rows: [synth.DataBinCollections][]synth.DataBinRow{
			synth.DataBinShapes: {
				wtScaleRow("Plain", 1),
				wtScaleRow("Great", 3),
			},
			synth.DataBinMaterials: {
				wtScaleRow("Iron", 1),
				wtScaleRow("Wood", 0.5),
				wtScaleRow("Mithril", 0.25),
			},
			synth.DataBinArmors: {
				wtArmorRow("Mail", 40, 4),
			},
			synth.DataBinShields: {
				wtShieldRow("Buckler", 7),
			},
			synth.DataBinWeapons: {
				wtWeaponRow("Sword", 23),
			},
		},
	}.Bytes())
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return f.Collection(databin.Shapes), f.Collection(databin.Materials),
		f.Collection(databin.Armors), f.Collection(databin.Shields), f.Collection(databin.Weapons)
}

func TestAnItemCodeWeighsItsColumnScaledByShapeAndMaterial(t *testing.T) {
	shapes, materials, armors, shields, weapons := wtTables(t)

	const (
		weaponClass = 1
		shieldClass = 2
		mailSlot    = 4
		carriedB    = 0 // field B naming no equipment slot at all
	)
	for _, tc := range []struct {
		name                        string
		material, class, shape, row int
		want                        int32
		known                       bool
	}{
		{"a great wooden sword", 1, weaponClass, 1, 1, 35, true},
		{"a plain mithril sword", 2, weaponClass, 0, 1, 6, true},
		{"a plain iron buckler", 0, shieldClass, 0, 1, 7, true},
		{"a great iron mail", 0, mailSlot, 1, 1, 120, true},
		{"a code naming no equipment class", 0, carriedB, 0, 1, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := data.ComposeItemCode(tc.material, tc.class, tc.shape, tc.row)
			got, ok, err := data.ItemCodeWeight(c, shapes, materials, armors, shields, weapons)
			if err != nil {
				t.Fatalf("ItemCodeWeight(0x%04x): %v", uint16(c), err)
			}
			if ok != tc.known {
				t.Fatalf("ItemCodeWeight(0x%04x) reports known=%v, want %v", uint16(c), ok, tc.known)
			}
			if got != tc.want {
				t.Errorf("ItemCodeWeight(0x%04x) is %d, want %d", uint16(c), got, tc.want)
			}
		})
	}
}

// TestTheWeightIsTheSameNumberTheNamedResolveAnswers is the cross-check that
// keeps the code path and the name path from drifting: the same row resolved
// by NAME must carry the same weight the code path answers, on all three
// classes. Neither number is computed here.
func TestTheWeightIsTheSameNumberTheNamedResolveAnswers(t *testing.T) {
	shapes, materials, armors, shields, weapons := wtTables(t)

	w, err := data.ResolveWeapon("Great Wood Sword", shapes, materials, weapons)
	if err != nil {
		t.Fatalf("ResolveWeapon: %v", err)
	}
	byCode, _, err := data.ItemCodeWeight(w.Code, shapes, materials, armors, shields, weapons)
	if err != nil {
		t.Fatalf("ItemCodeWeight: %v", err)
	}
	if w.Weight != byCode {
		t.Errorf("the weapon resolves to weight %d by name and %d by code", w.Weight, byCode)
	}

	s, err := data.ResolveShield("Plain Iron Buckler Shield", shapes, materials, shields)
	if err != nil {
		t.Fatalf("ResolveShield: %v", err)
	}
	byCode, _, err = data.ItemCodeWeight(s.Code, shapes, materials, armors, shields, weapons)
	if err != nil {
		t.Fatalf("ItemCodeWeight: %v", err)
	}
	if s.Weight != byCode {
		t.Errorf("the shield resolves to weight %d by name and %d by code", s.Weight, byCode)
	}

	a, err := data.ResolveArmor("Great Iron Mail", shapes, materials, armors)
	if err != nil {
		t.Fatalf("ResolveArmor: %v", err)
	}
	byCode, _, err = data.ItemCodeWeight(a.Code, shapes, materials, armors, shields, weapons)
	if err != nil {
		t.Fatalf("ItemCodeWeight: %v", err)
	}
	if a.Weight != byCode {
		t.Errorf("the armour resolves to weight %d by name and %d by code", a.Weight, byCode)
	}
}

// TestATableTheCallerDoesNotHaveAnswersFalse is the partial-install arm. A
// caller resolving a mission's codes over a table missing one collection must
// get "no weight" for that class and correct answers for the other two, rather
// than an error or a fault.
//
// The nil ScaleTable case is stated separately because both scale tables are
// read by every class: a caller without them resolves nothing at all.
func TestATableTheCallerDoesNotHaveAnswersFalse(t *testing.T) {
	shapes, materials, armors, shields, weapons := wtTables(t)
	sword := data.ComposeItemCode(1, 1, 1, 1)
	buckler := data.ComposeItemCode(0, 2, 0, 1)

	// No Shields collection: the shield answers false, the weapon still answers.
	if got, ok, err := data.ItemCodeWeight(buckler, shapes, materials, armors, nil, weapons); err != nil || ok || got != 0 {
		t.Errorf("with no Shields, the shield answers (%d, %v, %v), want (0, false, nil)", got, ok, err)
	}
	if _, ok, err := data.ItemCodeWeight(sword, shapes, materials, armors, nil, weapons); err != nil || !ok {
		t.Errorf("with no Shields, the weapon answers (%v, %v), want (true, nil)", ok, err)
	}

	// No scale tables: nothing resolves.
	for _, c := range []data.ItemCode{sword, buckler} {
		if got, ok, err := data.ItemCodeWeight(c, nil, materials, armors, shields, weapons); err != nil || ok || got != 0 {
			t.Errorf("with no Shapes, code 0x%04x answers (%d, %v, %v), want (0, false, nil)", uint16(c), got, ok, err)
		}
		if got, ok, err := data.ItemCodeWeight(c, shapes, nil, armors, shields, weapons); err != nil || ok || got != 0 {
			t.Errorf("with no Materials, code 0x%04x answers (%d, %v, %v), want (0, false, nil)", uint16(c), got, ok, err)
		}
	}
}

func TestACodeNamingAnUnwrittenRowIsAnError(t *testing.T) {
	shapes, materials, armors, shields, weapons := wtTables(t)

	past := data.ComposeItemCode(0, 2, 0, 30) // shield class, row 30, past the fixture
	if _, ok, err := data.ItemCodeWeight(past, shapes, materials, armors, shields, weapons); err == nil || ok {
		t.Errorf("a shield code past the collection answers (%v, %v), want an error", ok, err)
	}
	reserved := data.ComposeItemCode(0, 2, 0, 0) // the collection's own reserved zeroth entry
	if _, ok, err := data.ItemCodeWeight(reserved, shapes, materials, armors, shields, weapons); err == nil || ok {
		t.Errorf("a shield code on the reserved row answers (%v, %v), want an error", ok, err)
	}
}
