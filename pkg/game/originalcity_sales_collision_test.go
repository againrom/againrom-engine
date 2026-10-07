package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// cityCollisionBinding builds a minimal originalCityBinding holding two
// container stacks that share Class+DefinitionRow but differ in a retained
// runtime operand — DIV-762's "same weapon row, two materials" case.
// b.inventory mirrors what sav.CityProvenance.Inventory() actually decodes
// (no equip-time reached columns), while b.baseline carries the two real
// operand sets the character's own holdings decode supplies, in container
// order.
func cityCollisionBinding(t *testing.T, a, b sim.ItemInstance) originalCityBinding {
	t.Helper()
	piece := func(i sim.ItemInstance) sav.Piece {
		return sav.Piece{Class: "Weapon", Code: i.Code, Row: i.SourceEquipment.DefinitionRow,
			Stack: 1, Kind: i.Kind, Price: i.Price, Weight: i.Weight}
	}
	return originalCityBinding{
		partyID:            "collision",
		partyImportVersion: 5,
		baseline: mapload.PartyMember{ID: "collision",
			Carry: &mapload.Carry{ItemInstances: []sim.ItemInstance{a.Clone(), b.Clone()},
				Items: []uint16{a.Code, b.Code}}},
		inventory: []sav.CityInventoryItem{
			{Piece: piece(a), Weight: a.Weight, Exclusive: true},
			{Piece: piece(b), Weight: b.Weight, Exclusive: true},
		},
	}
}

func cityCollisionWeapon(code uint16, row, own uint8, spell bool) sim.ItemInstance {
	s := sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: row, OwnKind: own}
	if spell {
		s.Spell = sim.SourceItemSpell{Present: true, ID: 7, Range: 3, ManaCost: 4}
	}
	return sim.ItemInstance{Code: code, Kind: 1, Price: 100, Weight: 10,
		WeightPresent: true, SourceEquipment: s}
}

func TestCitySaleRemaindersAdmitsDistinctStacksSharingClassAndRow(t *testing.T) {
	cases := []struct {
		name string
		a, b sim.ItemInstance
	}{
		{"same row, different own-kind",
			cityCollisionWeapon(0x8114, 20, 4, false), cityCollisionWeapon(0x8134, 20, 6, false)},
		{"same row, one bound weapon spell",
			cityCollisionWeapon(0x8114, 20, 4, false), cityCollisionWeapon(0x8134, 20, 4, true)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := cityCollisionBinding(t, c.a, c.b)
			counts, err := citySaleRemainders(b, nil)
			if err != nil {
				t.Fatalf("distinct same-row stacks refused: %v", err)
			}
			if len(counts) != 2 || counts[0] != 1 || counts[1] != 1 {
				t.Fatalf("unexpected remainders %v", counts)
			}
			blend0, blend1 := b.inventoryInstance(0), b.inventoryInstance(1)
			if blend0.SourceEquipment.OwnKind != c.a.SourceEquipment.OwnKind ||
				blend1.SourceEquipment.OwnKind != c.b.SourceEquipment.OwnKind {
				t.Fatalf("own-kind blended across stacks: got (%d,%d) want (%d,%d)",
					blend0.SourceEquipment.OwnKind, blend1.SourceEquipment.OwnKind,
					c.a.SourceEquipment.OwnKind, c.b.SourceEquipment.OwnKind)
			}
			if blend0.SourceEquipment.Spell != c.a.SourceEquipment.Spell ||
				blend1.SourceEquipment.Spell != c.b.SourceEquipment.Spell {
				t.Fatalf("spell blended across stacks: got (%+v,%+v) want (%+v,%+v)",
					blend0.SourceEquipment.Spell, blend1.SourceEquipment.Spell,
					c.a.SourceEquipment.Spell, c.b.SourceEquipment.Spell)
			}
			// The exact symptom prepareOriginalCitySale exhibited: with the
			// container admitted, each source position must resolve to a
			// distinct display group so a sale on either item can find its
			// own, single, source position.
			if citySaleGroups(blend0, blend1) {
				t.Fatal("distinct stacks grouped as one sale target")
			}
		})
	}
}

// TestCitySaleRemaindersStillRefusesAGenuineMismatch is the negative
// control: the positional patch must not weaken citySaleRemainders' DeepEqual
// guard into admitting a container whose baseline genuinely disagrees with
// it. EffectsUnsupported also stays sourced from the city document's own
// per-item state, never blended from the baseline.
func TestCitySaleRemaindersStillRefusesAGenuineMismatch(t *testing.T) {
	t.Run("baseline code disagrees with its container position", func(t *testing.T) {
		a := cityCollisionWeapon(0x8114, 20, 4, false)
		b := cityCollisionWeapon(0x8134, 20, 6, false)
		binding := cityCollisionBinding(t, a, b)
		corrupt := binding.baseline.Carry.ItemInstances[1].Clone()
		corrupt.Code = 0x9999
		binding.baseline.Carry.ItemInstances[1] = corrupt
		if _, err := citySaleRemainders(binding, nil); err == nil {
			t.Fatal("genuine city/baseline Code mismatch was accepted")
		}
	})
	t.Run("city document's own unsupported effect state is never blended away", func(t *testing.T) {
		a := cityCollisionWeapon(0x8114, 20, 4, false)
		b := cityCollisionWeapon(0x8134, 20, 4, false)
		binding := cityCollisionBinding(t, a, b)
		binding.inventory[1].Piece.UnsupportedEffectStates = []uint8{1}
		if _, err := citySaleRemainders(binding, nil); err == nil {
			t.Fatal("baseline blend hid the city document's own unsupported effect state")
		}
	})
}
