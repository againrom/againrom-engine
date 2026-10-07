package sim

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

func TestItemEqualityKeepsOrderedEffectsDuplicatesAndPotionOwnership(t *testing.T) {
	effect := ItemEffect{Kind: 15, Mode: 1, Operand: 0x00200007}
	other := ItemEffect{Kind: 21, Operand: 9}
	plain := ItemInstance{Code: 0x0311, Kind: 1}
	if plain.HasEnchantment() || !(ItemInstance{Code: plain.Code, Effects: []ItemEffect{effect}}).HasEnchantment() {
		t.Fatal("the read-only enchantment predicate disagrees with the ordered effect list")
	}
	if !ItemEqual(plain, ItemInstance{Code: plain.Code, Kind: 2}) {
		t.Fatal("two equal-code effect-free items are stackable regardless of concrete class")
	}
	if ItemEqual(ItemInstance{Code: plain.Code, Kind: 1, Effects: []ItemEffect{effect}}, plain) {
		t.Fatal("exactly one stackable item compared equal")
	}
	ordered := ItemInstance{Code: plain.Code, Kind: 1, Effects: []ItemEffect{effect, other, effect}, Price: 10}
	if !ItemEqual(ordered, ItemInstance{Code: plain.Code, Kind: 1,
		Effects: []ItemEffect{effect, other, effect}, Price: 999}) {
		t.Fatal("non-Potion equality read transaction price")
	}
	if ItemEqual(ordered, ItemInstance{Code: plain.Code, Kind: 1, Effects: []ItemEffect{effect, effect, other}}) {
		t.Fatal("effect order was ignored")
	}
	if ItemEqual(ordered, ItemInstance{Code: plain.Code, Kind: 1, Effects: []ItemEffect{effect, other}}) {
		t.Fatal("a duplicate effect was ignored")
	}

	potion := ItemInstance{Code: 0x0e06, Kind: 3, Effects: []ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00064}}, Price: 125}
	if !ItemEqual(potion, potion.Clone()) {
		t.Fatal("an identical Potion instance did not compare equal")
	}
	for _, changed := range []ItemInstance{
		{Code: potion.Code, Kind: 3, Effects: []ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00065}}, Price: potion.Price},
		{Code: potion.Code, Kind: 3, Effects: potion.Clone().Effects, Price: potion.Price + 1},
		{Code: potion.Code, Effects: potion.Clone().Effects, Price: potion.Price},
	} {
		if ItemEqual(potion, changed) {
			t.Errorf("Potion owner-retention merged changed instance %+v", changed)
		}
	}
}

// Two equal-code weapons that differ only in Weapon bytes 22 and 23 (Item
// +0x68 and +0x69) are one item to the player and merge, enchanted or not. A
// change in any other retained operand, and in the price or weight, keeps
// them apart (DIV-762, DIV-1473).
func TestItemEqualityReadsNoWeaponBytes22And23(t *testing.T) {
	spelled := SourceEquipment{Class: SourceWeapon, DefinitionRow: 14, OwnKind: 4,
		Definition: SourceWeaponDefinition{Present: true, AttackType: 3, Hands: 2, Charge: 8, Relax: 4, Suitable: 2},
		Spell:      SourceItemSpell{Present: true, ID: 14, Range: 8, Defensive: 1, ManaCost: 80}}
	for i := range spelled.Attack {
		spelled.Attack[i] = byte(10 + i)
	}
	for i := range spelled.Defence {
		spelled.Defence[i] = byte(60 + i)
	}
	for _, effects := range [][]ItemEffect{nil, {{Kind: 41, Operand: 0x1e000e}}} {
		held := ItemInstance{Code: 0x916e, Kind: 2, Price: 300, Weight: 9, WeightPresent: true, Effects: effects, SourceEquipment: spelled}
		merges := func(mutate func(*ItemInstance)) bool {
			other := held.Clone()
			mutate(&other)
			return ItemEqual(held, other) && CanMergeItemValues(held, other) && ItemEqual(other, held) && CanMergeItemValues(other, held)
		}
		for _, byteAt := range []int{22, 23} {
			if !merges(func(i *ItemInstance) { i.SourceEquipment.Attack[byteAt] ^= 0xff }) {
				t.Errorf("effects %v: a change in Weapon byte %d kept two items apart", effects, byteAt)
			}
		}
		if !merges(func(i *ItemInstance) { i.SourceEquipment.Attack[22], i.SourceEquipment.Attack[23] = 0, 0 }) {
			t.Errorf("effects %v: two zeroed bytes kept two items apart", effects)
		}
		apart := map[string]func(*ItemInstance){
			"class":         func(i *ItemInstance) { i.SourceEquipment.Class = SourceArmor },
			"row":           func(i *ItemInstance) { i.SourceEquipment.DefinitionRow++ },
			"own kind":      func(i *ItemInstance) { i.SourceEquipment.OwnKind++ },
			"definition":    func(i *ItemInstance) { i.SourceEquipment.Definition.Charge++ },
			"spell":         func(i *ItemInstance) { i.SourceEquipment.Spell.ManaCost++ },
			"spell present": func(i *ItemInstance) { i.SourceEquipment.Spell = SourceItemSpell{} },
			"unsupported":   func(i *ItemInstance) { i.SourceEquipment.EffectsUnsupported = true },
			"weight":        func(i *ItemInstance) { i.Weight++ },
			"no weight":     func(i *ItemInstance) { i.WeightPresent = false },
			"price":         func(i *ItemInstance) { i.Price++ },
			"code":          func(i *ItemInstance) { i.Code++ },
		}
		for i := range spelled.Attack[:22] {
			apart[fmt.Sprintf("weapon byte %d", i)] = func(it *ItemInstance) { it.SourceEquipment.Attack[i]++ }
		}
		for i := range spelled.Defence {
			apart[fmt.Sprintf("defence byte %d", i)] = func(it *ItemInstance) { it.SourceEquipment.Defence[i]++ }
		}
		for name, mutate := range apart {
			if merges(mutate) {
				t.Errorf("effects %v: two items differing in %s merged", effects, name)
			}
		}
	}
}

func TestItemInstancesDeepCopyAcrossConstructionReadersAndSplitTransfer(t *testing.T) {
	item := ItemInstance{Code: 0x0e06, Kind: 3, Price: 321,
		Effects: []ItemEffect{{Kind: 8, Mode: 1, Operand: 0x03c00064}}}
	w := mustStockedWorld(t, 7,
		[]Entity{{ID: 1, X: 2, Y: 2}, {ID: 2, X: 3, Y: 2}},
		[]Stock{{ID: 1, ItemInstances: []ItemInstance{item, item}}})
	item.Effects[0].Operand = 0

	first, _ := w.CarriedItems(1)
	if len(first) != 2 || first[0].Effects[0].Operand != 0x03c00064 {
		t.Fatalf("constructor retained an input alias: %+v", first)
	}
	first[0].Effects[0].Operand = 1
	again, _ := w.CarriedItems(1)
	if again[0].Effects[0].Operand != 0x03c00064 {
		t.Fatalf("CarriedItems exposed world state: %+v", again)
	}

	if err := w.MoveCarried(1, 2, item.Code, 1); err != nil {
		t.Fatalf("MoveCarried: %v", err)
	}
	left, _ := w.CarriedStacks(1)
	moved, _ := w.CarriedItems(2)
	if len(left) != 1 || left[0].Count != 1 || len(moved) != 1 ||
		!reflect.DeepEqual(left[0].Instance(), moved[0]) {
		t.Fatalf("split transfer lost the instance: left=%+v moved=%+v", left, moved)
	}
	moved[0].Effects[0].Operand = 2
	leftAgain, _ := w.CarriedItems(1)
	movedAgain, _ := w.CarriedItems(2)
	if leftAgain[0].Effects[0].Operand != 0x03c00064 || movedAgain[0].Effects[0].Operand != 0x03c00064 {
		t.Fatal("split transfer left the source and destination aliasing")
	}
}

func TestInstanceSurvivesEquipDropPickupAndDeathPour(t *testing.T) {
	carried := ItemInstance{Code: 0x0311, Kind: 1, Price: 812,
		Effects: []ItemEffect{{Kind: 15, Operand: 7}, {Kind: 21, Operand: 4}}}
	w := mustStockedWorld(t, 11, []Entity{{ID: 1, X: 4, Y: 4}, {ID: 2, X: 4, Y: 4}},
		[]Stock{{ID: 1, ItemInstances: []ItemInstance{carried}}})
	Step(w, []Command{{Kind: KindEquip, Entity: 1, X: 0, Y: 3}})
	worn, _ := w.EquippedItems(1)
	if !reflect.DeepEqual(worn[2], carried) {
		t.Fatalf("equip reconstructed the item: %+v", worn[2])
	}
	Step(w, []Command{{Kind: KindDropWorn, Entity: 1, X: 4, Y: 4, Spell: 3}})
	sacks := w.Sacks()
	if len(sacks) != 1 || len(sacks[0].ItemInstances) != 1 || !reflect.DeepEqual(sacks[0].ItemInstances[0], carried) {
		t.Fatalf("ground drop reconstructed the item: %+v", sacks)
	}
	if err := w.TakeSack(2, 4, 4); err != nil {
		t.Fatalf("TakeSack: %v", err)
	}
	picked, _ := w.CarriedItems(2)
	if len(picked) != 1 || !reflect.DeepEqual(picked[0], carried) {
		t.Fatalf("pickup reconstructed the item: %+v", picked)
	}
	Step(w, []Command{{Kind: KindTerminalKill, Entity: 2}})
	sacks = w.Sacks()
	if len(sacks) != 1 || len(sacks[0].ItemInstances) != 1 || !reflect.DeepEqual(sacks[0].ItemInstances[0], carried) {
		t.Fatalf("death pour reconstructed the item: %+v", sacks)
	}
}

func TestItemStateRoundTripsAndEveryIdentityFieldReachesTheHash(t *testing.T) {
	baseEffects := []ItemEffect{{Kind: 15, Mode: 1, Operand: 0x00100007}, {Kind: 21, Operand: 4}}
	build := func(effects []ItemEffect, price int32, slot int) *World {
		item := ItemInstance{Code: 0x0311, Kind: 1, Effects: effects, Price: price}
		stock := Stock{ID: 1, ItemInstances: []ItemInstance{item}}
		if slot > 0 {
			stock.ItemInstances = nil
			stock.EquippedItems[slot-1] = item
		}
		return mustStockedWorld(t, 19, []Entity{{ID: 1, X: 1, Y: 1}}, []Stock{stock})
	}
	base := build(baseEffects, 40, 0)
	form, err := base.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	again, _ := back.MarshalBinary()
	if !bytes.Equal(form, again) || back.Hash() != base.Hash() {
		t.Fatal("item state did not round-trip byte-identically")
	}

	cases := []struct {
		name    string
		effects []ItemEffect
		price   int32
		slot    int
	}{
		{"kind", []ItemEffect{{Kind: 16, Mode: 1, Operand: 0x00100007}, {Kind: 21, Operand: 4}}, 40, 0},
		{"mode", []ItemEffect{{Kind: 15, Mode: 2, Operand: 0x00100007}, {Kind: 21, Operand: 4}}, 40, 0},
		{"operand", []ItemEffect{{Kind: 15, Mode: 1, Operand: 0x00100008}, {Kind: 21, Operand: 4}}, 40, 0},
		{"order", []ItemEffect{{Kind: 21, Operand: 4}, {Kind: 15, Mode: 1, Operand: 0x00100007}}, 40, 0},
		{"price", baseEffects, 41, 0},
		{"container", baseEffects, 40, 3},
		{"equipped slot", baseEffects, 40, 4},
	}
	seen := map[uint64]string{base.Hash(): "base"}
	for _, tc := range cases {
		h := build(tc.effects, tc.price, tc.slot).Hash()
		if previous, exists := seen[h]; exists {
			t.Errorf("%s hashes like %s at %#016x", tc.name, previous, h)
		}
		seen[h] = tc.name
	}
}
