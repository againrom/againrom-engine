package sim

import (
	"reflect"
	"testing"
)

func TestOriginalHoldings1108CacheOwnershipAndAtomicity(t *testing.T) {
	cast := ItemInstance{Code: 0x0101, Kind: 2, Price: -981, Effects: []ItemEffect{{Kind: 41, Operand: 327681}, {Kind: 6, Operand: 5}, {Kind: 42, Operand: 2}}}
	for _, source := range []WeaponSpellSource{WeaponSpellNone, WeaponSpellItem, WeaponSpellInnate, WeaponSpellLegacy} {
		t.Run(string(rune('0'+source)), func(t *testing.T) {
			e := Entity{ID: 1, X: 1, Y: 1, HP: 7, MaxHP: 31, Mana: 5, MaxMana: 23, WeaponSpellSource: source}
			var worn [EquipSlots]ItemInstance
			if source != WeaponSpellNone {
				e.WeaponSpell = 1
				e.WeaponSpellLevel = 5
			}
			if source == WeaponSpellItem {
				worn[0] = cast
			}
			w := mustStockedWorld(t, 1, []Entity{e, {ID: 2, X: 2, Y: 1, HP: 10, MaxHP: 10}}, []Stock{{ID: 1, EquippedItems: worn}})
			before := w.Hash()
			if err := w.ImportOriginalActorStock([]OriginalActorStock{{ID: 1}, {ID: 99}}); err == nil || w.Hash() != before {
				t.Fatalf("late refusal %v", err)
			}
			if err := w.ImportOriginalActorStock([]OriginalActorStock{{ID: 1}}); err != nil {
				t.Fatal(err)
			}
			got := w.Entities()[0]
			if source == WeaponSpellItem {
				e.WeaponSpell = 0
				e.WeaponSpellLevel = 0
				e.WeaponSpellSource = WeaponSpellNone
			}
			if got.WeaponSpell != e.WeaponSpell || got.WeaponSpellLevel != e.WeaponSpellLevel || got.WeaponSpellSource != e.WeaponSpellSource {
				t.Fatalf("empty stock/cache: %+v", got)
			}
			worn[0] = cast
			before = w.Hash()
			err := w.ImportOriginalActorStock([]OriginalActorStock{{ID: 1, Equipped: worn}})
			if source == WeaponSpellInnate || source == WeaponSpellLegacy {
				if err == nil || w.Hash() != before {
					t.Fatal("independent spell conflict accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got = w.Entities()[0]
			if got.WeaponSpellSource != WeaponSpellItem || got.WeaponSpell != 1 || got.WeaponSpellLevel != 5 || got.HP != 7 || got.Mana != 5 || got.KnownSpells != 0 {
				t.Fatalf("cache/no replay: %+v", got)
			}
			b, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var fresh World
			if err := fresh.UnmarshalBinary(b); err != nil || fresh.Hash() != w.Hash() {
				t.Fatalf("native: %v", err)
			}
		})
	}
}

func TestOriginalHoldings1108LifecycleRetainsCanonicalInstances(t *testing.T) {
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 1, Y: 1, HP: 100, MaxHP: 100}, {ID: 2, X: 2, Y: 1, HP: 100, MaxHP: 100}}, nil)
	first := ItemInstance{Code: 0x0701, Kind: 1, Price: -50, Effects: []ItemEffect{{Kind: 12, Operand: 3}, {Kind: 12, Operand: 2}, {Kind: 12, Operand: 3}}}
	second := first.Clone()
	second.Price = 60
	third := first.Clone()
	third.Kind = 2
	stock := OriginalActorStock{ID: 1, Carried: []ItemStack{StackItem(first, 3), StackItem(second, 1), StackItem(third, 1)}}
	if err := w.ImportOriginalActorStock([]OriginalActorStock{stock}); err != nil {
		t.Fatal(err)
	}
	stock.Carried[0].Effects[0].Operand = 999
	if err := w.MoveCarried(1, 2, first.Code, 2); err != nil {
		t.Fatal(err)
	}
	Step(w, []Command{{Kind: KindEquip, Entity: 2, X: 0, Y: 7}})
	worn, _ := w.EquippedItems(2)
	if !reflect.DeepEqual(worn[6], first) {
		t.Fatalf("equip: %+v", worn)
	}
	Step(w, []Command{{Kind: KindUnequip, Entity: 2, X: 7}})
	carried, _ := w.CarriedStacks(2)
	if len(carried) != 1 || !StackStateEqual(carried[0], StackItem(first, 2)) {
		t.Fatalf("unequip: %+v", carried)
	}
	Step(w, []Command{{Kind: KindDropCarried, Entity: 2, X: 1, Y: 1, Spell: 0}})
	if len(w.Sacks()) != 1 {
		t.Fatal("drop made no sack")
	}
	sack := w.Sacks()[0]
	if err := w.TakeSack(1, sack.X, sack.Y); err != nil {
		t.Fatal(err)
	}
	carried, _ = w.CarriedStacks(1)
	if len(carried) != 3 || carried[0].Count != 2 || carried[1].Price != 60 || carried[2].Kind != 2 {
		t.Fatalf("transfer/drop/pickup lost identity: %+v", carried)
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var fresh World
	if err := fresh.UnmarshalBinary(b); err != nil || fresh.Hash() != w.Hash() {
		t.Fatalf("native: %v", err)
	}
}
