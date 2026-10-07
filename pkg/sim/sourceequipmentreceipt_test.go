package sim

import (
	"fmt"
	"reflect"
	"testing"
)

func receiptEvent(kind SourceEquipmentEventKind, ticket uint32, place SourceEquipmentPlaceKind, index int, split bool) SourceEquipmentEvent {
	return SourceEquipmentEvent{Kind: kind, Ticket: ticket, Place: SourceEquipmentPlace{place, index}, Split: split}
}

func TestSourceEquipmentReceiptExactPartialAndEqualOccurrences(t *testing.T) {
	item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	other := item.Clone()
	other.Code, other.SourceEquipment.DefinitionRow = 0x0702, 2
	w := sourceMutationWorld(t, item)
	w.carried[0] = []ItemStack{StackItem(other, 1), StackItem(item, 2), StackItem(other, 1)}
	w.equipment[0][6] = other
	w.entities[0].ActorLoad.Accumulator = 123
	var r SourceEquipmentReceipt
	ok := w.EquipSourceCarried(1, 1, SourceEquipmentOperation{Receipt: &r})
	want := []SourceEquipmentEvent{
		receiptEvent(SourceEquipmentTake, 1, SourceEquipmentPack, 1, true),
		receiptEvent(SourceEquipmentTake, 2, SourceEquipmentWorn, 6, false),
		receiptEvent(SourceEquipmentPut, 1, SourceEquipmentWorn, 6, false),
		receiptEvent(SourceEquipmentPut, 2, SourceEquipmentPack, 1, false),
	}
	if !ok || !reflect.DeepEqual(r.Events, want) || r.Reservation != 0 {
		t.Fatal("partial receipt lost source/destination identity", r)
	}
	if len(w.carried[0]) != 4 || w.carried[0][0].Code != other.Code || w.carried[0][1].Code != other.Code || w.carried[0][2].Code != item.Code || w.carried[0][3].Code != other.Code || w.entities[0].ActorLoad.Accumulator != 123 {
		t.Fatal("equal input occurrences folded or cursor changed", w.carried[0])
	}
	for _, row := range w.carried[0] {
		if row.ObjectID != 0 || row.Count != 1 {
			t.Fatal("receipt borrowed native identity or changed remaining counts")
		}
	}
	if w.equipment[0][6].ObjectID != 0 {
		t.Fatal("receipt stamped a worn native identity")
	}
}

func TestSourceEquipmentReceiptDisplacementExternalAndAtomicFailure(t *testing.T) {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 2)
	old := sourceEquipmentWeapon(0x0101, 5, 11, 3, 4, 1)
	shield := ItemInstance{Code: 0x0201, WeightPresent: true, Weight: 3, SourceEquipment: SourceEquipment{Class: SourceShield, DefinitionRow: 1}}
	w := sourceMutationWorld(t, item)
	w.equipment[0][0], w.equipment[0][1] = old, shield
	w.entities[0].ActorLoad.Source.EquipmentRuntimePresent = true
	w.entities[0].ActorLoad.OwnWeight, w.entities[0].ActorLoad.Accumulator = 8, 2
	w.entities[0].Reach = 4
	var r SourceEquipmentReceipt
	ok := w.EquipSourceCarried(1, 0, SourceEquipmentOperation{Receipt: &r})
	want := []SourceEquipmentEvent{
		receiptEvent(SourceEquipmentTake, 1, SourceEquipmentPack, 0, false),
		receiptEvent(SourceEquipmentTake, 2, SourceEquipmentWorn, 0, false),
		{Kind: SourceEquipmentSpellWrite, Ticket: 2},
		receiptEvent(SourceEquipmentTake, 3, SourceEquipmentWorn, 1, false),
		receiptEvent(SourceEquipmentPut, 3, SourceEquipmentPack, 0, false),
		receiptEvent(SourceEquipmentPut, 1, SourceEquipmentWorn, 0, false),
		receiptEvent(SourceEquipmentPut, 2, SourceEquipmentPack, 0, false),
	}
	if !ok || !reflect.DeepEqual(r.Events, want) || len(w.carried[0]) != 2 || w.carried[0][0].Code != old.Code || w.carried[0][1].Code != shield.Code {
		t.Fatal("ordered displacement receipt differs", r)
	}
	removed, ok := w.UnequipSource(1, 1, false, SourceEquipmentOperation{Receipt: &r})
	if !ok || removed.ObjectID != 0 || !reflect.DeepEqual(r.Events, []SourceEquipmentEvent{receiptEvent(SourceEquipmentTake, 1, SourceEquipmentWorn, 0, false), {Kind: SourceEquipmentSpellWrite, Ticket: 1}, receiptEvent(SourceEquipmentPut, 1, SourceEquipmentExternal, 0, false)}) {
		t.Fatal("external return lost its exact occurrence", r)
	}
	ok = w.EquipSourceItem(1, removed, SourceEquipmentOperation{Receipt: &r})
	if !ok || !reflect.DeepEqual(r.Events, []SourceEquipmentEvent{receiptEvent(SourceEquipmentTake, 1, SourceEquipmentExternal, 0, false), receiptEvent(SourceEquipmentPut, 1, SourceEquipmentWorn, 0, false)}) {
		t.Fatal("external input lost its ticket", r)
	}
	w.BindSourceDerive(func(s SourceActor, _ int32, _ Rules) (SourceActor, error) {
		if s.Attack[16] == 0 {
			return s, fmt.Errorf("late source removal")
		}
		return s, nil
	})
	hash := w.Hash()
	beforeReceipt := r
	beforeReceipt.Events = append([]SourceEquipmentEvent(nil), r.Events...)
	_, ok = w.UnequipSource(1, 1, true, SourceEquipmentOperation{Receipt: &r})
	if ok || !reflect.DeepEqual(r, beforeReceipt) || w.Hash() != hash {
		t.Fatal("failed receipt published a partial graph operation")
	}
}

func TestSourceEquipmentReceiptPreservesRepeatedBoundRoots(t *testing.T) {
	item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	w := sourceMutationWorld(t, item)
	bindOperationsPack1115(t, w, 0)
	row := w.savedObjects.item(10).Value.Clone()
	w.carried[0] = append(w.carried[0], row.Clone())
	w.savedObjects.Containers[0].Items = append(w.savedObjects.Containers[0].Items, 10)
	w.entities[0].ActorLoad.Accumulator += int32(row.Weight)
	w.syncSavedPack(0)
	var r SourceEquipmentReceipt
	ok := w.EquipSourceCarried(1, 1, SourceEquipmentOperation{Receipt: &r})
	if !ok || len(r.Events) != 2 || r.Events[0].Place.Index != 1 || len(w.carried[0]) != 1 || w.carried[0][0].ObjectID != 10 || w.equipment[0][6].ObjectID != 10 {
		t.Fatal("receipt lost a repeated native root", r)
	}
	_, ok = w.UnequipSource(1, 7, true, SourceEquipmentOperation{Receipt: &r})
	if !ok || len(r.Events) != 2 || len(w.carried[0]) != 2 || w.carried[0][0].ObjectID != 10 || w.carried[0][1].ObjectID != 10 || w.carried[0][0].Count != 1 || len(w.savedObjects.Locations(10)) != 2 {
		t.Fatal("receipt folded repeated same-node roots", r)
	}
}

func TestSourceEquipmentExplicitAliasesMatchNativeLoadTiming(t *testing.T) {
	item := ItemInstance{Code: 0x701, WeightPresent: true, Weight: 100, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	makeWorld := func(bound bool) (*World, *[]int32) {
		w := sourceMutationWorld(t, item)
		w.carried[0][0].Count = 2
		if bound {
			bindOperationsPack1115(t, w, 0)
		}
		w.carried[0] = append(w.carried[0], w.carried[0][0].Clone())
		w.entities[0].ActorLoad.Accumulator, w.entities[0].ActorLoad.OwnWeight = 750, 3
		w.entities[0].Load = w.entities[0].ActorLoad.CurrentLoad()
		w.entities[0].HumanMovement.Load = w.entities[0].Load
		w.syncSavedPack(0)
		var calls []int32
		w.BindSourceDerive(func(s SourceActor, accumulator int32, _ Rules) (SourceActor, error) {
			calls = append(calls, accumulator)
			return s, nil
		})
		return w, &calls
	}
	bound, boundCalls := makeWorld(true)
	scratch, scratchCalls := makeWorld(false)
	if ok := bound.EquipSourceCarried(1, 1, SourceEquipmentOperation{Receipt: &SourceEquipmentReceipt{}}); !ok {
		t.Fatal("native alias partial equip failed")
	}
	if ok := scratch.EquipSourceCarried(1, 1, SourceEquipmentOperation{PackAliases: []int{0}}); !ok {
		t.Fatal("explicit scratch alias partial equip failed")
	}
	if bound.entities[0].ActorLoad.Accumulator != 550 || !reflect.DeepEqual(bound.entities[0].CurrentActorLoad(), scratch.entities[0].CurrentActorLoad()) || !reflect.DeepEqual(*boundCalls, *scratchCalls) {
		t.Fatal("explicit aliases changed source load or derive timing", bound.entities[0].CurrentActorLoad(), scratch.entities[0].CurrentActorLoad(), *boundCalls, *scratchCalls)
	}
	for _, w := range []*World{bound, scratch} {
		if len(w.carried[0]) != 2 || w.carried[0][0].Count != 1 || w.carried[0][1].Count != 1 {
			t.Fatal("partial quantity did not reach both explicit aliases")
		}
	}
	for _, aliases := range [][]int{{-1}, {2}, {1}, {0, 0}} {
		w, _ := makeWorld(false)
		hash := w.Hash()
		var r SourceEquipmentReceipt
		if ok := w.EquipSourceCarried(1, 1, SourceEquipmentOperation{PackAliases: aliases, Receipt: &r}); ok || len(r.Events) != 0 || w.Hash() != hash {
			t.Fatal("invalid alias positions committed a partial operation", aliases)
		}
	}
	w, _ := makeWorld(false)
	w.carried[0][0].Price++
	hash := w.Hash()
	if ok := w.EquipSourceCarried(1, 1, SourceEquipmentOperation{PackAliases: []int{0}}); ok || w.Hash() != hash {
		t.Fatal("conflicting explicit alias values were silently selected")
	}
}

func TestCloneSplitItemValueUsesOneSpellConstructor(t *testing.T) {
	item := ItemInstance{Code: 0x101, Effects: []ItemEffect{{Kind: 8, Operand: 17}}, SourceEquipment: SourceEquipment{Class: SourceWeapon, Spell: SourceItemSpell{Present: true, ID: 7, Range: 99, ManaCost: 33}}}
	copy := CloneSplitItemValue(item, nil)
	if copy.ObjectID != 0 || copy.SourceEquipment.Spell != (SourceItemSpell{Present: true, ID: 7}) {
		t.Fatal("split retained source Spell scalars without a rule")
	}
	copy.Effects[0].Operand++
	if item.Effects[0].Operand != 17 {
		t.Fatal("split aliases source value slices")
	}
	copy = CloneSplitItemValue(item, []SpellRule{{ID: 7, MaxRange: 11, ManaCost: 13, Defensive: true}})
	if copy.SourceEquipment.Spell != (SourceItemSpell{Present: true, ID: 7, Range: 11, ManaCost: 13, Defensive: 1}) {
		t.Fatal("split did not use the declared Spell constructor")
	}
}

func TestSourceEquipmentSameItemReplacementUsesCurrentNode(t *testing.T) {
	for _, receipt := range []bool{false, true} {
		item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
		item.Effects = []ItemEffect{{Kind: 41, Operand: 1 | 7<<16}}
		item.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 3, Range: 9, ManaCost: 5}
		w := sourceMutationWorld(t, item)
		w.entities[0].ActorLoad.Source.EquipmentRuntimePresent = true
		w.entities[0].ActorLoad.OwnWeight = item.Weight
		w.entities[0].Reach = item.SourceEquipment.OwnKind
		w.spells = []SpellRule{{ID: 1, MaxRange: 7, ManaCost: 3, Defensive: true}}
		bindOperationsPack1115(t, w, 0)
		w.equipment[0][0] = w.carried[0][0].Instance()
		if err := w.savedObjects.AddItemRoot(10, SavedObjectOwner{Kind: SavedOwnerActorWorn, Entity: 1, Slot: 1}); err != nil {
			t.Fatal(err)
		}
		var r SourceEquipmentReceipt
		var ok bool
		if receipt {
			ok = w.EquipSourceCarried(1, 0, SourceEquipmentOperation{Receipt: &r})
		} else {
			ok = w.EquipSourceCarried(1, 0)
		}
		if !ok || len(w.savedObjects.Locations(10)) != 2 || w.savedObjects.item(10).Spell != 0 || w.carried[0][0].SourceEquipment.Spell.Present || w.equipment[0][0].SourceEquipment.Spell.Present {
			t.Fatal("same-node eviction chose the stale prepared incoming value", receipt)
		}
		if receipt {
			want := []SourceEquipmentEvent{
				receiptEvent(SourceEquipmentTake, 1, SourceEquipmentPack, 0, false),
				{Kind: SourceEquipmentSpellWrite, Ticket: 1, Spell: SourceItemSpell{Present: true, ID: 1, Range: 7, ManaCost: 3, Defensive: 1}},
				receiptEvent(SourceEquipmentTake, 2, SourceEquipmentWorn, 0, false),
				{Kind: SourceEquipmentSpellWrite, Ticket: 2},
				receiptEvent(SourceEquipmentPut, 1, SourceEquipmentWorn, 0, false),
				receiptEvent(SourceEquipmentPut, 2, SourceEquipmentPack, 0, false),
			}
			if !reflect.DeepEqual(r.Events, want) {
				t.Fatal("receipt lost prepare-before-evict writes", r)
			}
		}
		reloadOperations1115(t, w)
	}
}

func TestSourceEquipmentOperationRejectsInvalidContextWithoutOutput(t *testing.T) {
	for _, mode := range []string{"two contexts", "topology with aliases", "carried handle", "external aliases", "external zero handle", "unequip aliases", "invalid carried index", "invalid external item", "empty worn slot"} {
		t.Run(mode, func(t *testing.T) {
			item := ItemInstance{Code: 0x701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
			w := sourceMutationWorld(t, item)
			r := SourceEquipmentReceipt{Events: []SourceEquipmentEvent{{Ticket: 81, Count: 7}}, Reservation: 82}
			prior := r
			prior.Events = append([]SourceEquipmentEvent(nil), r.Events...)
			op := SourceEquipmentOperation{Receipt: &r}
			hash := w.Hash()
			var ok bool
			switch mode {
			case "two contexts":
				ok = w.EquipSourceCarried(1, 0, op, op)
			case "topology with aliases":
				op.Topology, op.PackAliases = &SourceEquipmentTopology{Pack: []uint64{1}}, []int{0}
				ok = w.EquipSourceCarried(1, 0, op)
			case "carried handle":
				op.SessionHandle = 1
				ok = w.EquipSourceCarried(1, 0, op)
			case "external aliases":
				op.PackAliases = []int{0}
				ok = w.EquipSourceItem(1, item, op)
			case "external zero handle":
				op.SessionHandle = 1
				ok = w.EquipSourceItem(1, item, op)
			case "unequip aliases":
				op.PackAliases = []int{0}
				_, ok = w.UnequipSource(1, 7, true, op)
			case "invalid carried index":
				ok = w.EquipSourceCarried(1, 4, op)
			case "invalid external item":
				ok = w.EquipSourceItem(1, ItemInstance{}, op)
			case "empty worn slot":
				_, ok = w.UnequipSource(1, 7, true, op)
			}
			if ok || w.Hash() != hash || !reflect.DeepEqual(r, prior) {
				t.Fatal("failed operation changed World or receipt", r)
			}
		})
	}
}

func TestSourceEquipmentOperationEmptyContextPreservesLegacyBehavior(t *testing.T) {
	item := ItemInstance{Code: 0x701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	left, right := sourceMutationWorld(t, item), sourceMutationWorld(t, item)
	if !left.EquipSourceCarried(1, 0) || !right.EquipSourceCarried(1, 0, SourceEquipmentOperation{}) || left.Hash() != right.Hash() {
		t.Fatal("empty context changed carried equip")
	}
	a, aOK := left.UnequipSource(1, 7, true)
	b, bOK := right.UnequipSource(1, 7, true, SourceEquipmentOperation{})
	if !aOK || !bOK || !reflect.DeepEqual(a, b) || left.Hash() != right.Hash() {
		t.Fatal("empty context changed unequip")
	}
	if !left.EquipSourceItem(1, item) || !right.EquipSourceItem(1, item, SourceEquipmentOperation{}) || left.Hash() != right.Hash() {
		t.Fatal("empty context changed external equip")
	}
}

func TestSourceEquipmentOperationExplicitSessionHandleAndDetachedOutput(t *testing.T) {
	item := ItemInstance{Code: 0x701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	w := sourceMutationWorld(t, item)
	bindOperationsPack1115(t, w, 0)
	if !w.EquipSourceCarried(1, 0) {
		t.Fatal("bound equip")
	}
	var r SourceEquipmentReceipt
	removed, ok := w.UnequipSource(1, 7, false, SourceEquipmentOperation{Receipt: &r})
	if !ok || removed.ObjectID != 10 || r.Reservation == 0 || len(r.Events) != 2 {
		t.Fatal("explicit detached handle missing", r)
	}
	hash := w.Hash()
	prior := r
	prior.Events = append([]SourceEquipmentEvent(nil), r.Events...)
	if w.EquipSourceItem(1, removed, SourceEquipmentOperation{SessionHandle: r.Reservation + 1, Receipt: &r}) || w.Hash() != hash || !reflect.DeepEqual(r, prior) {
		t.Fatal("wrong session handle changed owner or receipt")
	}
	if !w.EquipSourceItem(1, removed, SourceEquipmentOperation{SessionHandle: r.Reservation, Receipt: &r}) || r.Reservation != 0 || len(r.Events) != 2 || w.equipment[0][6].ObjectID != 10 || w.savedObjects.sessionHandle(10) != 0 {
		t.Fatal("explicit session handle did not transfer exactly once", r)
	}
	hash = w.Hash()
	r.Events[0].Ticket++
	if w.Hash() != hash {
		t.Fatal("returned receipt aliases World state")
	}
}
