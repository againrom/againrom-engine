package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func TestOrderedStockWithoutLoadKeepsCellsThroughRebuildAndBinary(t *testing.T) {
	item := ItemInstance{Code: 0x701, Kind: 1, Price: 19, WeightPresent: true, Weight: -3, Effects: []ItemEffect{{Kind: 8, Operand: 5}}}
	want := []ItemStack{StackItem(item, 2), {}, StackItem(item, 1)}
	w := mustStockedWorld(t, 31, []Entity{{ID: 1, X: 1, Y: 1}}, []Stock{{ID: 1, OrderedStacks: want, ItemInstances: expandItems(want)}})
	check := func(w *World) {
		t.Helper()
		got, ok := w.CarriedStacks(1)
		if !ok || !reflect.DeepEqual(got, want) {
			t.Fatal("explicit equal cells or null position changed", got)
		}
		stock := w.Stock()
		if len(stock) != 1 || stock[0].LoadState != nil || !reflect.DeepEqual(stock[0].OrderedStacks, want) || len(stock[0].ItemInstances) != 3 {
			t.Fatal("stock did not retain independent order", stock)
		}
		stock[0].OrderedStacks[0].Effects[0].Operand++
		if w.carried[0][0].Effects[0].Operand != 5 {
			t.Fatal("stock order aliases live item values")
		}
	}
	check(w)
	for range 2 {
		rebuilt := mustStockedWorld(t, 31, w.Entities(), w.Stock())
		check(rebuilt)
		form := mustMarshal(t, rebuilt)
		var cold World
		if err := cold.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		if cold.Hash() != rebuilt.Hash() || !bytes.Equal(form, mustMarshal(t, &cold)) {
			t.Fatal("ordered stock binary continuation changed")
		}
		check(&cold)
		w = &cold
	}
}

func TestOrderedStockAppendsRowsAndNilKeepsDefaultFolding(t *testing.T) {
	ents := []Entity{{ID: 1}}
	load := &ActorLoadSnapshot{Inventory: ActorLoad{Present: true, ContainerPresent: true}}
	for _, withLoad := range []*ActorLoadSnapshot{nil, load} {
		w := mustStockedWorld(t, 1, ents, []Stock{{ID: 1, LoadState: withLoad, Items: []uint16{0x101, 0x101}}})
		if len(w.carried[0]) != 1 || w.carried[0][0].Count != 2 {
			t.Fatal("nil order stopped default folding")
		}
	}
	w := mustStockedWorld(t, 1, ents, []Stock{
		{ID: 1, Items: []uint16{0x101}, OrderedStacks: []ItemStack{PlainStack(0x101, 1), {}}},
		{ID: 1, Items: []uint16{0x101}, OrderedStacks: []ItemStack{PlainStack(0x101, 1)}},
	})
	if len(w.carried[0]) != 3 || !emptyOrderedStack(w.carried[0][1]) {
		t.Fatal("multiple ordered stock rows lost their append positions", w.carried[0])
	}
	nulls := mustStockedWorld(t, 1, ents, []Stock{{ID: 1, OrderedStacks: []ItemStack{{}, {}}}})
	if got := nulls.Stock(); len(got) != 1 || len(got[0].OrderedStacks) != 2 || got[0].LoadState != nil || len(got[0].Items) != 0 {
		t.Fatal("null-only ordered container vanished", got)
	}
}

func TestOrderedStockRequiresFullInstanceAndBoundedQuantities(t *testing.T) {
	item := ItemInstance{Code: 0x101, Price: 4, Effects: []ItemEffect{{Kind: 8, Operand: 9}}}
	for name, mutate := range map[string]func(*Stock){
		"quantity":         func(s *Stock) { s.OrderedStacks[0].Count = ^uint32(0) },
		"price":            func(s *Stock) { s.OrderedStacks[0].Price++ },
		"effect":           func(s *Stock) { s.OrderedStacks[0].Effects[0].Operand++ },
		"identity":         func(s *Stock) { s.OrderedStacks[0].ObjectID = 77 },
		"weight":           func(s *Stock) { s.OrderedStacks[0].WeightPresent = true },
		"equipment":        func(s *Stock) { s.OrderedStacks[0].SourceEquipment.DefinitionRow = 9 },
		"empty projection": func(s *Stock) { s.ItemInstances = nil },
		"empty order":      func(s *Stock) { s.OrderedStacks = []ItemStack{} },
		"zero count":       func(s *Stock) { s.OrderedStacks[0].Count = 0 },
		"zero code":        func(s *Stock) { s.OrderedStacks[0].Code = 0 },
		"null residue":     func(s *Stock) { s.OrderedStacks = append(s.OrderedStacks, ItemStack{Price: 1}) },
		"absent container": func(s *Stock) { s.LoadState = &ActorLoadSnapshot{Inventory: ActorLoad{Present: true}} },
	} {
		t.Run(name, func(t *testing.T) {
			stock := Stock{ID: 1, OrderedStacks: []ItemStack{StackItem(item, 1)}, ItemInstances: []ItemInstance{item.Clone()}}
			mutate(&stock)
			if _, _, err := normaliseHoldings([]Entity{{ID: 1}}, []Stock{stock}); err == nil {
				t.Fatal("invalid explicit stock admitted")
			}
		})
	}
}

func TestSourceEquipmentReceiptPreservesNullPackIndices(t *testing.T) {
	item := ItemInstance{Code: 0x701, WeightPresent: true, Weight: 2, SourceEquipment: SourceEquipment{Class: SourceArmor, DefinitionRow: 1, OwnKind: 7}}
	w := sourceMutationWorld(t, item)
	order := []ItemStack{{}, StackItem(item, 1), {}, StackItem(item, 1)}
	if !w.ReplaceStock(Stock{ID: 1, OrderedStacks: order, ItemInstances: expandItems(order), LoadState: w.entities[0].CurrentActorLoad()}) {
		t.Fatal("ordered source scratch stock rejected")
	}
	hash := w.Hash()
	var r SourceEquipmentReceipt
	if ok := w.EquipSourceCarried(1, 2, SourceEquipmentOperation{Receipt: &r}); ok || len(r.Events) != 0 || w.Hash() != hash {
		t.Fatal("null slot selected or failed selection changed state")
	}
	ok := w.EquipSourceCarried(1, 3, SourceEquipmentOperation{Receipt: &r})
	if !ok || len(r.Events) != 2 || r.Events[0].Place.Index != 3 || len(w.carried[0]) != 3 || !emptyOrderedStack(w.carried[0][0]) || !emptyOrderedStack(w.carried[0][2]) || w.carried[0][1].Code != item.Code {
		t.Fatal("receipt shifted null or surviving item positions", r, w.carried[0])
	}
}

func TestOrderedNullSlotsSurviveImportAndNativeInsertion(t *testing.T) {
	for _, load := range []*ActorLoadSnapshot{nil, {Inventory: ActorLoad{Present: true, ContainerPresent: true}}} {
		w := mustStockedWorld(t, 31, []Entity{{ID: 1, HP: 10, MaxHP: 10}}, nil)
		order := []ItemStack{{}, PlainStack(0x101, 1), {}}
		if err := w.ImportOriginalActorStock([]OriginalActorStock{{ID: 1, Carried: order, LoadState: load}}); err != nil {
			t.Fatal(err)
		}
		hash := w.Hash()
		w.equip(0, 0, 1, 0)
		if w.Hash() != hash {
			t.Fatal("native equip selected an empty ordered slot")
		}
		if !w.addCarried(0, PlainStack(0x201, 1)) {
			t.Fatal("native insertion failed beside a null slot")
		}
		want := []ItemStack{{}, PlainStack(0x101, 1), {}, PlainStack(0x201, 1)}
		if load != nil {
			want = []ItemStack{PlainStack(0x201, 1), {}, PlainStack(0x101, 1), {}}
		}
		if !reflect.DeepEqual(w.carried[0], want) {
			t.Fatal("insertion folded or filled a null slot", w.carried[0])
		}
		var cold World
		if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil || cold.Hash() != w.Hash() {
			t.Fatal("native insertion produced unreadable ordered stock", err)
		}
	}
}

func TestOrderedNullRootDoesNotGrantUnboundItemCoverage(t *testing.T) {
	graph := &SavedObjects{Version: SavedObjectsVersion, NextID: 1, Containers: []SavedObjectContainer{{Owner: SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 1}, Present: true, Items: []SavedObjectID{0}}}}
	for _, empty := range []bool{true, false} {
		stack := ItemStack{}
		if !empty {
			stack = PlainStack(0x101, 1)
		}
		w := mustStockedWorld(t, 1, []Entity{{ID: 1}}, []Stock{{ID: 1, OrderedStacks: []ItemStack{stack}, ItemInstances: expandItems([]ItemStack{stack})}})
		hash := w.Hash()
		err := w.ImportSavedObjects(graph, nil)
		if empty {
			if err != nil || w.savedObjects == nil {
				t.Fatal("exact null root required fake coverage", err)
			}
		} else if err == nil || w.savedObjects != nil || w.Hash() != hash {
			t.Fatal("null root silently covered an unbound Item", err)
		}
	}
}
