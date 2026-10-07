package mapload

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
)

type sourceConstructorScale1110 string

func (sourceConstructorScale1110) Len() int               { return 1 }
func (s sourceConstructorScale1110) EntryName(int) string { return string(s) }
func (sourceConstructorScale1110) EntryDoubles(int) []float64 {
	return []float64{1, 1, 1, 1, 1, 1, 1, 1, 1}
}

func sourceConstructorTable1110() *Table {
	return &Table{Shapes: sourceConstructorScale1110("Common"), Materials: sourceConstructorScale1110("Iron"),
		Weapons: ghostResistanceCollection{{}, {name: "Blade", params: []int32{0, 0, 0, 2, 0, 1, 7, 16, 11, 6, 0, 4, 13, 17, 1, 1}}},
		Shields: ghostResistanceCollection{{}, {name: "Shield", params: []int32{0, 0, 0, 2, 0, 0, 0, 0, 0, 7, 3}}},
		Armors:  ghostResistanceCollection{{}, {name: "Coat", params: []int32{0, 0, 0, 2, 7, 0, 0, 0, 0, 7, 3}}}}
}

func TestSourceConstructor1110LiteralOperandsAndMissingTable(t *testing.T) {
	table := sourceConstructorTable1110()
	for _, code := range []uint16{0x0101, 0x0201, 0x0701} {
		item := SourceConstructedItem(sim.PlainItem(code), table)
		want := sim.SourceEquipment{DefinitionRow: 1, Defence: [22]byte{7, 0, 3}}
		switch code {
		case 0x0101:
			want = sim.SourceEquipment{Class: 1, DefinitionRow: 1, OwnKind: 4, Attack: [24]byte{11, 14: 7, 15: 9}, Defence: [22]byte{6}, Definition: sim.SourceWeaponDefinition{Present: true, AttackType: 1, Hands: 1, Charge: 13, Relax: 17, Suitable: 1}}
		case 0x0201:
			want.Class = 3
		case 0x0701:
			want.Class = 2
			want.OwnKind = 7
		}
		if item.SourceEquipment != want || !item.WeightPresent || item.Weight != 2 {
			t.Fatalf("code%x: %+v", code, item)
		}
		// Saved objects, including BaseItem with a weapon-like appearance,
		// never acquire the constructor of that appearance.
		saved := sim.ItemInstance{Code: code, WeightPresent: true, Weight: 0}
		if got := SourceConstructedItem(saved, table); !reflect.DeepEqual(got, saved) {
			t.Fatal("saved BaseItem reinterpreted")
		}
		item.SourceEquipment.Defence[0] = 99
		if got := SourceConstructedItem(item, table); !reflect.DeepEqual(got, item) {
			t.Fatal("saved concrete block refilled")
		}
	}
	table.Shapes = nil
	if got := SourceConstructedItem(sim.PlainItem(0x0101), table); got.SourceEquipment.Class != 0 {
		t.Fatal("partial table invented identity factors")
	}
}

func TestSourceConstructor1110GeneratedEquipmentFreshWorld(t *testing.T) {
	s := sourceLiteral1110()
	s.EquipmentRuntimePresent = true
	s.Reach = 1
	s.AttackCharge = 8
	s.AttackRelax = 4
	for _, code := range []uint16{0x0101, 0x0701} {
		w := sourceWorld1110(t, s, []sim.ItemInstance{sim.PlainItem(code)})
		DeclareCodeWeights(w, sourceConstructorTable1110(), []uint16{code})
		fresh := sourceNative1110(t, w)
		for _, current := range []*sim.World{w, fresh} {
			if !current.EquipSourceCarried(1, 0) {
				t.Fatal("generated equipment refused")
			}
			e := current.Entities()[0]
			if e.Load != 204 || e.Capacity != 301 || e.HP != 24 || e.Speed != 15 {
				t.Fatal("literal constructor action sheet", e)
			}
			worn, _ := current.EquippedItems(1)
			slot := 7
			if code == 0x0101 {
				slot = 1
				if e.ToHit != 50 || e.DamageBase != 11 || e.Reach != 4 {
					t.Fatal("weapon constructor operands", e)
				}
			}
			if !worn[slot-1].WeightPresent || worn[slot-1].SourceEquipment.Class == 0 {
				t.Fatal("constructed instance not retained")
			}
			next := sourceNative1110(t, current)
			if _, ok := next.UnequipSource(1, slot, true); !ok {
				t.Fatal("fresh next removal")
			}
			if _, ok := current.UnequipSource(1, slot, true); !ok || current.Hash() != next.Hash() {
				t.Fatal("fresh continuation diverged")
			}
		}
		if w.Hash() != fresh.Hash() {
			t.Fatal("constructor cache was not canonical")
		}
	}
}

func TestSourceConstructor1110ShieldAttachFreshNative(t *testing.T) {
	s := sourceLiteral1110()
	s.EquipmentRuntimePresent = true
	s.Reach = 1
	s.AttackCharge = 8
	s.AttackRelax = 4
	w := sourceWorld1110(t, s, []sim.ItemInstance{sim.PlainItem(0x0101), sim.PlainItem(0x0201)})
	DeclareCodeWeights(w, sourceConstructorTable1110(), []uint16{0x0101, 0x0201})
	if !w.EquipSourceCarried(1, 0) {
		t.Fatal("weapon")
	}
	for _, current := range []*sim.World{w, sourceNative1110(t, w)} {
		if !current.EquipSourceCarried(1, 0) {
			t.Fatal("onehand shield refused")
		}
		e := current.Entities()[0]
		if e.Defence != 21 || e.Absorption != 3 || e.Load != 205 || e.ActorLoad.OwnWeight != 4 || e.ActorLoad.Accumulator != 402 {
			t.Fatal("shield direct stores/load", e)
		}
		next := sourceNative1110(t, current)
		if _, ok := next.UnequipSource(1, 2, true); !ok {
			t.Fatal("shield removal")
		}
		e = next.Entities()[0]
		if e.Defence != 14 || e.Absorption != 0 || e.Load != 204 {
			t.Fatal("shield removal literals", e)
		}
	}
}
