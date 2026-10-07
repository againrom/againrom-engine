package mapload

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/sim"
)

func TestSourceEquipment1110ActualArmorAndWeaponActionsFreshNative(t *testing.T) {
	for _, kind := range []string{"armor", "melee", "ranged"} {
		t.Run(kind, func(t *testing.T) {
			s := sourceLiteral1110()
			s.EquipmentRuntimePresent, s.Reach, s.AttackCharge, s.AttackRelax = true, 1, 8, 4
			item := sim.ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2,
				SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 7, Defence: [22]byte{7}}}
			slot := 7
			if kind != "armor" {
				slot, item.Code = 1, 0x0101
				item.SourceEquipment = sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 37, OwnKind: 4,
					Attack: [24]byte{0: 11, 14: 7, 15: 9}, Defence: [22]byte{6},
					Definition: sim.SourceWeaponDefinition{Present: true, AttackType: 1, Hands: 2, Charge: 13, Relax: 17}}
				if kind == "ranged" {
					item.SourceEquipment.Definition.AttackType = 11
				}
			}
			w := sourceWorld1110(t, s, []sim.ItemInstance{item})
			fresh := sourceNative1110(t, w)
			for _, current := range []*sim.World{w, fresh} {
				sim.Step(current, []sim.Command{{Kind: sim.KindEquip, Entity: 1, X: 0, Y: int32(slot)}})
				e := current.Entities()[0]
				worn, _ := current.EquippedItems(1)
				pack, _ := current.CarriedStacks(1)
				if worn[slot-1].Code != item.Code || len(pack) != 0 || e.Load != 204 || e.Capacity != 301 || e.Speed != 15 || e.HP != 24 || e.MaxHP != 24 {
					t.Fatalf("literal equip sheet: %+v", e)
				}
				switch kind {
				case "armor":
					if e.Defence != 15 || e.ToHit != 39 || e.XPSlot != 1 {
						t.Fatal("armor direct+load order", e)
					}
				case "melee":
					if e.Defence != 14 || e.ToHit != 50 || e.DamageBase != 11 || e.DamageSpread != 12 || e.XPSlot != 1 || e.Reach != 4 || e.AttackCharge != 13 || e.AttackRelax != 17 {
						t.Fatal("melee own-kind/definition split", e)
					}
				case "ranged":
					if e.Defence != 8 || e.ToHit != 8 || e.DamageBase != 2 || e.DamageSpread != 3 || e.XPSlot != 0 || e.SecondaryDamage != (sim.SecondaryDamage{Base: 7, Spread: 9, Selector: 0}) {
						t.Fatal("ranged General assignment", e)
					}
				}
				back := sourceNative1110(t, current)
				for _, resumed := range []*sim.World{current, back} {
					sim.Step(resumed, []sim.Command{{Kind: sim.KindUnequip, Entity: 1, X: int32(slot)}})
					a := resumed.Entities()[0]
					if a.Load != 203 || a.ActorLoad.Accumulator != 406 || a.ActorLoad.OwnWeight != 0 || a.Defence != 8 {
						t.Fatal("remove bookkeeping/direct fields", a)
					}
					if kind == "ranged" && (a.ToHit != -3 || binary.LittleEndian.Uint16(a.ActorLoad.Source.Modifier[18:]) != 65532) {
						t.Fatal("ranged removal is not the old modifier", a)
					}
					if kind == "melee" && (a.ToHit != 6 || a.XPSlot != 0 || a.Reach != 1 || a.AttackCharge != 8 || a.AttackRelax != 4) {
						t.Fatal("melee removal", a)
					}
				}
				if back.Hash() != current.Hash() {
					t.Fatal("next removal differs after fresh native")
				}
			}
			if w.Hash() != fresh.Hash() {
				t.Fatal("initial fresh/native actions diverge")
			}
		})
	}
}

func TestSourceEquipment1110ForwardEffectsNotBatchedOrReversed(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		effects := []sim.ItemEffect{{Kind: 2, Operand: 10}, {Kind: 6, Operand: 10}}
		wantHP := int32(34)
		if reverse {
			effects[0], effects[1] = effects[1], effects[0]
			wantHP = 24
		}
		item := sim.ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, Effects: effects,
			SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 7}}
		w := sourceWorld1110(t, sourceLiteral1110(), []sim.ItemInstance{item})
		if !w.EquipSourceCarried(1, 0) {
			t.Fatal("ordered effects refused")
		}
		e := w.Entities()[0]
		if e.HP != wantHP || e.MaxHP != 46 || e.Capacity != 401 {
			t.Fatalf("order=%t hp=%d max=%d cap=%d", reverse, e.HP, e.MaxHP, e.Capacity)
		}
		_ = sourceNative1110(t, w)
	}
}
