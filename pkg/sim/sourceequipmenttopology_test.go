package sim

import (
	"reflect"
	"testing"
)

func TestSourceEquipmentTopologyReportsActualMergeAndAliasCounts(t *testing.T) {
	for _, shared := range []bool{false, true} {
		item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
		other := item.Clone()
		other.Code, other.SourceEquipment.DefinitionRow = 0x0702, 2
		w := sourceMutationWorld(t, item)
		w.carried[0] = []ItemStack{StackItem(other, 1), StackItem(item, 2), StackItem(other, 1)}
		w.equipment[0][6] = other
		w.entities[0].ActorLoad.Accumulator = 123
		topology := SourceEquipmentTopology{Pack: []uint64{10, 20, 11}}
		topology.Worn[6] = 30
		if shared {
			topology.Pack[2] = 10
		}
		before := append([]uint64(nil), topology.Pack...)
		var r SourceEquipmentReceipt
		ok := w.EquipSourceCarried(1, 1, SourceEquipmentOperation{Topology: &topology, Receipt: &r})
		want := []SourceEquipmentEvent{
			receiptEvent(SourceEquipmentTake, 1, SourceEquipmentPack, 1, true),
			receiptEvent(SourceEquipmentTake, 2, SourceEquipmentWorn, 6, false),
			receiptEvent(SourceEquipmentPut, 1, SourceEquipmentWorn, 6, false),
			{Kind: SourceEquipmentPut, Ticket: 2, Place: SourceEquipmentPlace{SourceEquipmentPack, 0}, Merged: true},
		}
		for i := range want {
			want[i].Count = 1
		}
		if !ok || !reflect.DeepEqual(r.Events, want) || !reflect.DeepEqual(topology.Pack, before) {
			t.Fatal("actual merge location or operation-local topology changed", shared, r)
		}
		lastCount, accumulator := uint32(1), int32(123)
		if shared {
			lastCount, accumulator = 2, 125
		}
		if len(w.carried[0]) != 3 || w.carried[0][0].Count != 2 || w.carried[0][1].Count != 1 || w.carried[0][2].Count != lastCount || w.entities[0].ActorLoad.Accumulator != accumulator {
			t.Fatal("merge did not update exactly its node and source load", shared, w.carried[0], w.entities[0].ActorLoad.Accumulator)
		}
		for _, stack := range w.carried[0] {
			if stack.ObjectID != 0 {
				t.Fatal("ephemeral topology entered a native Item handle")
			}
		}
	}
}

func TestSourceEquipmentTopologyPreservesPrepareEvictOrderAcrossAliases(t *testing.T) {
	item := sourceEquipmentWeapon(0x0102, 2, 5, 7, 6, 1)
	item.Effects = []ItemEffect{{Kind: 41, Operand: 1 | 7<<16}}
	item.SourceEquipment.Spell = SourceItemSpell{Present: true, ID: 3, Range: 9, ManaCost: 5}
	w := sourceMutationWorld(t, item)
	w.entities[0].ActorLoad.Source.EquipmentRuntimePresent = true
	w.entities[0].ActorLoad.OwnWeight = item.Weight
	w.entities[0].Reach = item.SourceEquipment.OwnKind
	w.spells = []SpellRule{{ID: 1, MaxRange: 7, ManaCost: 3, Defensive: true}}
	w.carried[0] = append(w.carried[0], w.carried[0][0].Clone())
	w.equipment[0][0] = item.Clone()
	topology := SourceEquipmentTopology{Pack: []uint64{7, 7}}
	topology.Worn[0] = 7
	var r SourceEquipmentReceipt
	ok := w.EquipSourceCarried(1, 0, SourceEquipmentOperation{Topology: &topology, Receipt: &r})
	if !ok || len(w.carried[0]) != 2 || len(r.Events) != 6 {
		t.Fatal("same-node prepare and eviction lost a root", r)
	}
	for _, event := range r.Events {
		if event.Merged {
			t.Fatal("moving the same node doubled its current count")
		}
	}
	if r.Events[1].Spell != (SourceItemSpell{Present: true, ID: 1, Range: 7, ManaCost: 3, Defensive: 1}) || r.Events[3].Spell != (SourceItemSpell{}) {
		t.Fatal("ordered Spell writes disappeared", r)
	}
	for _, value := range append([]ItemInstance{w.equipment[0][0]}, w.carried[0][0].Instance(), w.carried[0][1].Instance()) {
		if value.SourceEquipment.Spell.Present || value.ObjectID != 0 {
			t.Fatal("later explicit teardown did not reach a same-node view")
		}
	}
}

func TestSourceEquipmentTopologyExternalMergeAndAtomicValidation(t *testing.T) {
	item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	w := sourceMutationWorld(t, item)
	w.carried[0] = []ItemStack{StackItem(item, 1), StackItem(item, 1)}
	w.equipment[0][6] = item.Clone()
	topology := SourceEquipmentTopology{Pack: []uint64{10, 10}}
	topology.Worn[6] = 20
	for _, bad := range []SourceEquipmentTopology{
		{Pack: []uint64{10}},
		{Pack: []uint64{0, 10}, Worn: topology.Worn},
		{Pack: []uint64{10, 10}},
		{Pack: []uint64{10, 10}, Worn: topology.Worn, External: 30},
	} {
		before := w.Hash()
		var r SourceEquipmentReceipt
		if _, ok := w.UnequipSource(1, 7, true, SourceEquipmentOperation{Topology: &bad, Receipt: &r}); ok || len(r.Events) != 0 || before != w.Hash() {
			t.Fatal("malformed topology changed current state", bad, r)
		}
	}
	var r SourceEquipmentReceipt
	removed, ok := w.UnequipSource(1, 7, true, SourceEquipmentOperation{Topology: &topology, Receipt: &r})
	if !ok || removed.Code != item.Code || len(r.Events) != 2 || !r.Events[1].Merged || r.Events[1].Place.Index != 0 || w.carried[0][0].Count != 2 || w.carried[0][1].Count != 2 {
		t.Fatal("external equipment return did not report exact merge", r)
	}
	topology.Worn[6], topology.External = 0, 40
	if ok := w.EquipSourceItem(1, item, SourceEquipmentOperation{Topology: &topology, Receipt: &r}); !ok || len(r.Events) != 2 || r.Events[0].Place.Kind != SourceEquipmentExternal || r.Events[1].Place.Kind != SourceEquipmentWorn {
		t.Fatal("external input did not retain its explicit ticket", r)
	}
}

func TestSourceEquipmentTopologyMovesWholeWornQuantity(t *testing.T) {
	for _, targets := range []int{0, 1, 2} {
		item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
		w := sourceMutationWorld(t, item)
		w.carried[0] = nil
		w.equipment[0][6] = item
		w.entities[0].ActorLoad.Accumulator, w.entities[0].ActorLoad.OwnWeight = 123, 2
		topology := SourceEquipmentTopology{}
		topology.Worn[6], topology.WornCounts[6] = 20, 2
		for i := 0; i < targets; i++ {
			w.carried[0] = append(w.carried[0], StackItem(item, 1))
			topology.Pack = append(topology.Pack, 10)
		}
		var r SourceEquipmentReceipt
		removed, ok := w.UnequipSource(1, 7, true, SourceEquipmentOperation{Topology: &topology, Receipt: &r})
		if !ok || removed.ObjectID != 0 || len(r.Events) != 2 || r.Events[0].Count != 2 || r.Events[1].Count != 2 || r.Events[1].Merged != (targets > 0) || r.Events[1].Place != (SourceEquipmentPlace{SourceEquipmentPack, 0}) {
			t.Fatal("whole quantity or exact destination missing", targets, r)
		}
		wantCount, wantLoad := uint32(2), int32(127)
		if targets > 0 {
			wantCount = 3
			wantLoad += int32(targets-1) * 4
		}
		if len(w.carried[0]) != max(1, targets) || !w.equipment[0][6].Empty() || w.entities[0].ActorLoad.Accumulator != wantLoad || w.entities[0].ActorLoad.OwnWeight != 0 {
			t.Fatal("whole move changed roots or source load", targets, w.carried[0], w.entities[0].ActorLoad)
		}
		for _, stack := range w.carried[0] {
			if stack.Count != wantCount || stack.ObjectID != 0 {
				t.Fatal("target aliases lost quantity or acquired native handles", targets, stack)
			}
		}
	}
}

func TestSourceEquipmentTopologyExternalQuantityRoundTrip(t *testing.T) {
	item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	w := sourceMutationWorld(t, item)
	w.carried[0] = nil
	topology := SourceEquipmentTopology{External: 20, ExternalCount: 2}
	var r SourceEquipmentReceipt
	ok := w.EquipSourceItem(1, item, SourceEquipmentOperation{Topology: &topology, Receipt: &r})
	if !ok || len(r.Events) != 2 || r.Events[0].Count != 2 || r.Events[1].Count != 2 || w.equipment[0][6].ObjectID != 0 {
		t.Fatal("external quantity lost before attach", r)
	}
	topology = SourceEquipmentTopology{}
	topology.Worn[6], topology.WornCounts[6] = 20, 2
	_, ok = w.UnequipSource(1, 7, true, SourceEquipmentOperation{Topology: &topology, Receipt: &r})
	if !ok || len(r.Events) != 2 || r.Events[0].Count != 2 || r.Events[1].Count != 2 || len(w.carried[0]) != 1 || w.carried[0][0].Count != 2 || w.carried[0][0].ObjectID != 0 {
		t.Fatal("explicit Worn quantity lost on next operation", r, w.carried[0])
	}
}

func TestSourceEquipmentTopologyQuantityFailuresAreAtomic(t *testing.T) {
	for _, mode := range []string{"null count", "count bound", "pack conflict", "worn conflict", "merge overflow", "late source refusal"} {
		t.Run(mode, func(t *testing.T) {
			item := ItemInstance{Code: 0x0701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
			w := sourceMutationWorld(t, item)
			w.carried[0] = []ItemStack{StackItem(item, 1)}
			w.equipment[0][6] = item
			topology := SourceEquipmentTopology{Pack: []uint64{10}}
			topology.Worn[6], topology.WornCounts[6] = 20, 2
			switch mode {
			case "null count":
				topology.WornCounts[0] = 2
			case "count bound":
				topology.WornCounts[6] = MaxOriginalHoldingValues + 1
			case "pack conflict":
				topology.Worn[6] = 10
			case "worn conflict":
				w.equipment[0][5] = item
				topology.Worn[5], topology.WornCounts[5] = 20, 3
			case "merge overflow":
				w.carried[0][0].Count = MaxOriginalHoldingValues
			case "late source refusal":
				w.entities[0].Capacity = 0
			}
			before := w.Hash()
			r := SourceEquipmentReceipt{Events: []SourceEquipmentEvent{{Ticket: 71, Count: 3}}, Reservation: 72}
			prior := r
			prior.Events = append([]SourceEquipmentEvent(nil), r.Events...)
			if _, ok := w.UnequipSource(1, 7, true, SourceEquipmentOperation{Topology: &topology, Receipt: &r}); ok || !reflect.DeepEqual(r, prior) || before != w.Hash() {
				t.Fatal("invalid quantity committed a prefix", r)
			}
		})
	}
}
