package sim

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func sourceMutationWorld(t *testing.T, item ItemInstance) *World {
	t.Helper()
	w := bookWorld(t, Entity{ID: 1, HP: 50, MaxHP: 100}, item)
	s := SourceActor{Class: 2, Fighter: true, Stats: [14]uint16{10, 20, 10, 10, 18, 0, 0, 101, 50, 100, 100, 0, 0, 50}}
	if err := w.RestoreActorLoad(1, ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}); err != nil {
		t.Fatal(err)
	}
	w.BindSourceDerive(func(s SourceActor, acc int32, _ Rules) (SourceActor, error) { return s, nil })
	return w
}

func TestSourceMutation1110WholePotionRollsBackLateFault(t *testing.T) {
	for _, timed := range []bool{false, true} {
		first := ItemEffect{Kind: 2, Operand: 1}
		if timed {
			first = ItemEffect{Kind: 16, Mode: 1, Operand: 3 | 9<<16}
		}
		w := sourceMutationWorld(t, testPotion(first, ItemEffect{Kind: 3, Operand: 1}))
		w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
			if s.Stats[2] == 11 {
				return s, fmt.Errorf("bounded candidate failure")
			}
			return s, nil
		})
		before := w.Hash()
		if w.UseCarriedPotion(1, 0) || before != w.Hash() || len(w.attached) != 0 {
			t.Fatal("late effect fault committed prefix", timed)
		}
	}
}

func TestSourceMutation1110PotionLoadCallbackFaultAndTimedReplaceAreAtomic(t *testing.T) {
	item := testPotion(ItemEffect{Kind: 2, Operand: 1})
	item.WeightPresent, item.Weight = true, 400
	w := sourceMutationWorld(t, item)
	w.BindSourceDerive(func(s SourceActor, acc int32, _ Rules) (SourceActor, error) {
		if acc != 0 {
			return s, fmt.Errorf("candidate load failure")
		}
		return s, nil
	})
	before := w.Hash()
	if w.UseCarriedPotion(1, 0) || w.Hash() != before {
		t.Fatal("load callback failure consumed/applied potion")
	}
	w = sourceMutationWorld(t, testPotion(ItemEffect{Kind: 16, Mode: 1, Operand: 5 | 9<<16}))
	if !w.RestorePotionEffect(1, ActiveEffect{Kind: EffectAbsorption, Mode: EffectDuration, Magnitude: 3, Remaining: 7}) {
		t.Fatal("seed effect")
	}
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
		if binary.LittleEndian.Uint16(s.Modifier[44:]) == 5 {
			return s, fmt.Errorf("replacement failure")
		}
		return s, nil
	})
	before = w.Hash()
	if w.UseCarriedPotion(1, 0) || w.Hash() != before || w.attached[0].Remaining != 7 || w.attached[0].Magnitude != 3 {
		t.Fatal("replacement failure retired old effect")
	}
}

func TestSourceMutation1110ScrollRefundFailureRetainsDetachedOwner(t *testing.T) {
	for _, reason := range []string{"capacity", "rule", "candidate"} {
		w := sourceMutationWorld(t, PlainItem(0xe01))
		w.carried[0] = nil
		w.scrollCasts = []ScrollCast{{Caster: 1, Item: ItemInstance{Code: 0xe01, Weight: 400, WeightPresent: true}}}
		switch reason {
		case "capacity":
			w.entities[0].Capacity = 0
			w.entities[0].HumanMovement.Capacity = 0
		case "rule":
			w.BindSourceDerive(nil)
		case "candidate":
			w.BindSourceDerive(func(s SourceActor, acc int32, _ Rules) (SourceActor, error) {
				if acc != 0 {
					return s, fmt.Errorf("refund arithmetic failure")
				}
				return s, nil
			})
		}
		before := w.Hash()
		if w.cancelScroll(0) || w.Hash() != before || len(w.scrollCasts) != 1 || len(w.carried[0]) != 0 {
			t.Fatal("failed refund dropped detached item", reason)
		}
	}
}

func TestSourceMutation1110TransferSecondCallbackAndPickupRefuseAllOwners(t *testing.T) {
	item := ItemInstance{Code: 0xe01, Weight: 400, WeightPresent: true}
	w := mustStockedWorld(t, 1, []Entity{{ID: 1, X: 4, Y: 4, HP: 50, MaxHP: 100, Owner: 1}, {ID: 2, X: 4, Y: 4, HP: 50, MaxHP: 100, Owner: 2}}, []Stock{{ID: 1, ItemInstances: []ItemInstance{item}}})
	for id := EntityID(1); id <= 2; id++ {
		s := SourceActor{Class: 2, Fighter: true, TypeID: uint16(id), Stats: [14]uint16{10, 20, 10, 10, 18, 0, 0, 101, 50, 100, 100, 0, 0, 50}}
		if err := w.RestoreActorLoad(id, ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true, Source: s}, Capacity: 101, Speed: 18, Movement: HumanMovement{Present: true, RawSpeed: 18, NativeSpeed: 18, Capacity: 101}}); err != nil {
			t.Fatal(err)
		}
	}
	w.BindSourceDerive(func(s SourceActor, acc int32, _ Rules) (SourceActor, error) {
		if s.TypeID == 2 && acc != 0 {
			return s, fmt.Errorf("recipient failure")
		}
		return s, nil
	})
	before := w.Hash()
	if err := w.MoveCarried(1, 2, item.Code, 1); err == nil || w.Hash() != before {
		t.Fatal("second callback committed first actor", err)
	}
	w.pourSack(4, 4, 31, []ItemInstance{item})
	before = w.Hash()
	if err := w.TakeSack(2, 4, 4); err == nil || w.Hash() != before || w.Purse(2) != 0 {
		t.Fatal("pickup callback spent sack/gold", err)
	}
}
