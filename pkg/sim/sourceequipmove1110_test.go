package sim

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"testing"
)

func sourceEquipmentWeapon(code uint16, weight int16, hit, damage, defence byte, hands int32) ItemInstance {
	return ItemInstance{Code: code, WeightPresent: true, Weight: weight,
		SourceEquipment: SourceEquipment{Class: SourceWeapon, DefinitionRow: 1, OwnKind: 4,
			Attack: [24]byte{0: hit, 14: damage}, Defence: [22]byte{defence},
			Definition: SourceWeaponDefinition{Present: true, AttackType: 2, Hands: hands, Charge: 13, Relax: 17, Suitable: 1}}}
}

func TestSourceEquipment1110ReplacementTraceAndLateFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		newItem := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 2)
		newItem.Effects = []ItemEffect{{Kind: 12, Operand: 2}, {Kind: 7, Operand: 3}}
		w := sourceMutationWorld(t, newItem)
		old := sourceEquipmentWeapon(0x0101, 5, 11, 3, 4, 1)
		old.Effects = []ItemEffect{{Kind: 12, Operand: 4}}
		old.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 1, Range: 7, ManaCost: 3}
		shield := ItemInstance{Code: 0x0201, WeightPresent: true, Weight: 3,
			SourceEquipment: SourceEquipment{Class: SourceShield, DefinitionRow: 1, Defence: [22]byte{7}}, Effects: []ItemEffect{{Kind: 0}}}
		w.equipment[0][0], w.equipment[0][1] = old, shield
		e := &w.entities[0]
		e.ActorLoad.OwnWeight, e.ActorLoad.Accumulator, e.Load, e.HumanMovement.Load = 8, 2, 9, 9
		e.Reach, e.AttackCharge, e.AttackRelax = 4, 9, 6
		e.ActorLoad.Source.EquipmentRuntimePresent = true
		e.ActorLoad.Source.Modifier[18], e.ActorLoad.Source.Modifier[32], e.ActorLoad.Source.Modifier[42] = 11, 3, 15
		e.Defence = 15
		type event struct {
			hit                                    uint16
			defence, damage, active, reach, charge byte
			hp                                     uint16
			acc                                    int32
		}
		var trace []event
		w.BindSourceDerive(func(s SourceActor, acc int32, _ Rules) (SourceActor, error) {
			trace = append(trace, event{binary.LittleEndian.Uint16(s.Modifier[18:]), s.Modifier[42], s.Modifier[32], s.Attack[16], s.Reach, s.AttackCharge, s.Stats[8], acc})
			if fail && s.Stats[8] == 53 {
				return s, fmt.Errorf("late last-effect failure")
			}
			return s, nil
		})
		before := w.Hash()
		ok := w.EquipSourceCarried(1, 0)
		if fail {
			if ok || w.Hash() != before {
				t.Fatal("late failure committed displacement/effect prefix")
			}
			continue
		}
		// The first call is read-only numeric admission. All six later calls
		// are original reached derives, not a batch's reconstructed loadout.
		want := []event{{7, 15, 3, 0, 4, 9, 50, 0}, {65532, 11, 0, 0, 4, 9, 50, 0},
			{65532, 4, 0, 0, 1, 8, 50, 0}, {1, 10, 7, 2, 1, 8, 50, 3},
			{3, 10, 7, 2, 4, 13, 50, 3}, {3, 10, 7, 2, 4, 13, 53, 3}}
		if !ok || !reflect.DeepEqual(trace[1:], want) {
			t.Fatalf("ordered callback inputs: %+v want %+v", trace, want)
		}
		if w.entities[0].Load != 6 || w.entities[0].ActorLoad.Accumulator != 8 || len(w.carried[0]) != 2 ||
			w.carried[0][0].Code != old.Code || w.carried[0][1].Code != shield.Code || w.carried[0][0].SourceEquipment.Spell.Present {
			t.Fatal("same-slot return/opposite-hand insertion or late Spell teardown")
		}
	}
}

func TestSourceEquipment1110PartialAndDuplicateBoundary(t *testing.T) {
	item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	other := item.Clone()
	other.Code, other.SourceEquipment.DefinitionRow = 0x0702, 2
	w := sourceMutationWorld(t, item)
	w.carried[0] = []ItemStack{StackItem(other, 1), StackItem(item, 2), StackItem(other, 1)}
	w.equipment[0][6] = other
	w.entities[0].ActorLoad.Accumulator = 123 // saved accumulator is not a population sum
	if !w.EquipSourceCarried(1, 1) {
		t.Fatal("partial equip")
	}
	if len(w.carried[0]) != 3 || w.carried[0][0].Count != 2 || w.carried[0][1].Count != 1 || w.carried[0][2].Count != 1 ||
		w.entities[0].ActorLoad.InsertIndex != 1 || w.entities[0].ActorLoad.Accumulator != 123 {
		t.Fatal("partial take merged original duplicate boundaries or lost insertion cursor", w.carried[0])
	}
}

func TestSourceEquipment1110FirstSpellAndPrepareBeforeEviction(t *testing.T) {
	for _, mode := range []string{"first", "no match", "missing rule"} {
		t.Run(mode, func(t *testing.T) {
			item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
			item.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 3, Range: 9, ManaCost: 5}
			if mode != "no match" {
				item.Effects = []ItemEffect{{Kind: 41, Operand: 1 | 7<<16}, {Kind: 41, Operand: 2 | 9<<16}}
			}
			w := sourceMutationWorld(t, item)
			w.entities[0].ActorLoad.Source.EquipmentRuntimePresent = true
			w.entities[0].Reach = 1
			w.equipment[0][0] = sourceEquipmentWeapon(0x0101, 3, 1, 1, 1, 1)
			w.spells = []SpellRule{{ID: 1, MaxRange: 7, ManaCost: 3, Defensive: true}}
			if mode == "missing rule" {
				w.spells = nil
			}
			before := w.Hash()
			ok := w.EquipSourceCarried(1, 0)
			if mode == "missing rule" {
				if ok || w.Hash() != before {
					t.Fatal("missing first Spell evicted old weapon")
				}
				return
			}
			want := SourceItemSpell{Present: true, ID: 1, Range: 7, ManaCost: 3, Defensive: 1}
			if mode == "no match" {
				want = item.SourceEquipment.Spell
			}
			if !ok || w.equipment[0][0].SourceEquipment.Spell != want {
				t.Fatal("first/no-match owned Spell", w.equipment[0][0])
			}
		})
	}
}

func TestSourceEquipment1110DeathFreshContainerAndRefusal(t *testing.T) {
	for _, mode := range []string{"ordinary", "suppressed", "undroppable", "late fault"} {
		t.Run(mode, func(t *testing.T) {
			w := sourceMutationWorld(t, ItemInstance{Code: 0xe01, WeightPresent: true, Weight: 3})
			e := &w.entities[0]
			e.ActorLoad.Accumulator, e.ActorLoad.InsertIndex, e.ActorLoad.OwnWeight = 777, 0, 7
			e.ActorLoad.Source.EquipmentRuntimePresent = true
			e.HP, e.Reach = -10, 4
			weapon := sourceEquipmentWeapon(0x0101, 5, 11, 3, 4, 1)
			weapon.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 1, Range: 7, ManaCost: 3}
			armor := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
			armor.Effects = []ItemEffect{{Kind: 12, Operand: 2}}
			w.equipment[0][0], w.equipment[0][6] = weapon, armor
			if mode == "suppressed" {
				e.SuppressCorpseLoot = true
			}
			if mode == "undroppable" {
				w.equipment[0][0].SourceEquipment.Definition.Suitable = 0
			}
			if mode == "late fault" {
				w.BindSourceDerive(func(s SourceActor, acc int32, _ Rules) (SourceActor, error) {
					if binary.LittleEndian.Uint16(s.Modifier[18:]) == 65523 {
						return s, fmt.Errorf("armor removal failure")
					}
					return s, nil
				})
			}
			before := w.Hash()
			ok := w.dropTerminalLoot(0)
			if mode == "late fault" {
				if ok || w.Hash() != before {
					t.Fatal("failed death changed holdings/sack/RNG")
				}
				return
			}
			if !ok || len(w.carried[0]) != 0 || w.entities[0].ActorLoad.Accumulator != 0 || w.entities[0].ActorLoad.InsertIndex != 10000 {
				t.Fatal("not a fresh empty container")
			}
			if mode == "suppressed" {
				if len(w.sacks) != 0 {
					t.Fatal("suppressed source contents dropped")
				}
				return
			}
			wantCount := 3
			if mode == "undroppable" {
				wantCount = 2
				if w.equipment[0][0].Empty() {
					t.Fatal("parameter15 zero weapon removed")
				}
			}
			if len(w.sacks) != 1 || len(w.sacks[0].ItemInstances) != wantCount {
				t.Fatal("death loot count", w.sacks)
			}
			for _, item := range w.sacks[0].ItemInstances {
				if item.SourceEquipment.Spell.Present {
					t.Fatal("removed Spell survived on loot")
				}
			}
		})
	}
}
