package game

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func setShopIdentityPack(f *FrontEnd, items ...sim.ItemInstance) {
	f.Carried[0].Carry.ItemInstances = nil
	f.Carried[0].Carry.Items = nil
	for _, item := range items {
		f.Carried[0].Carry.ItemInstances = append(f.Carried[0].Carry.ItemInstances, item.Clone())
		f.Carried[0].Carry.Items = append(f.Carried[0].Carry.Items, item.Code)
	}
}

func requireShopPackIDs(t *testing.T, f *FrontEnd, want ...sim.SavedObjectID) {
	t.Helper()
	var got []sim.SavedObjectID
	for _, item := range f.Carried[0].Carry.ItemInstances {
		got = append(got, item.ObjectID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("remaining pack IDs = %v, want %v", got, want)
	}
}

func TestShopIdentifiedPackMovesSelectedCountAndReequips(t *testing.T) {
	f, screen := shopRoom(t, nil)
	item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
	slot, _ := EquipTarget(data.ItemCode(item.Code), f.Table)
	first, second := item.Clone(), item.Clone()
	first.ObjectID, second.ObjectID = 101, 202
	setShopIdentityPack(f, first, second, second)

	view := screen.ShopScreen()
	if view.Pack[1].Count != 1 || view.Pack[2].Count != 2 {
		t.Fatalf("pack row counts = %d, %d, want distinct nodes with counts 1, 2", view.Pack[1].Count, view.Pack[2].Count)
	}
	clickShift(screen, ui.ShopControlPackCell, 2)
	requireShopPackIDs(t, f, 101)
	if table := f.Shop.Table(); len(table) != 1 || table[0].ObjectID != 202 || table[0].Count != 2 {
		t.Fatalf("selected whole stack on table = %+v", table)
	}
	click(screen, ui.ShopControlPackCell, 1)
	requireShopPackIDs(t, f)
	if table := f.Shop.Table(); len(table) != 2 || table[1].ObjectID != 101 || table[1].Count != 1 {
		t.Fatalf("equal-valued distinct table node = %+v", table)
	}
	if action := screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlDoll}); action.Msg != "worn" {
		t.Fatalf("table equip = %+v", action)
	}
	if worn := f.Carried[0].Carry.EquippedItems[slot-1]; worn.ObjectID != 202 {
		t.Fatalf("worn table node = %+v", worn)
	}
	if table := f.Shop.Table(); len(table) != 2 || table[0].ObjectID != 202 || table[0].Count != 1 || table[1].ObjectID != 101 || table[1].Count != 1 {
		t.Fatalf("table after taking one selected unit = %+v", table)
	}
	screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1}, ui.ShopControl{Kind: ui.ShopControlTableCell})
	if table := f.Shop.Table(); len(table) != 2 || table[0].ObjectID != 202 || table[0].Count != 2 || table[1].ObjectID != 101 || table[1].Count != 1 {
		t.Fatalf("table after returning worn node = %+v", table)
	}
	screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlPackCell})
	clickShift(screen, ui.ShopControlTableCell, 0)
	requireShopPackIDs(t, f, 101, 202, 202)
	if action := screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 2}, ui.ShopControl{Kind: ui.ShopControlDoll}); action.Msg != "worn" {
		t.Fatalf("second pack row equip = %+v", action)
	}
	requireShopPackIDs(t, f, 101, 202)
	if worn := f.Carried[0].Carry.EquippedItems[slot-1]; worn.ObjectID != 202 {
		t.Fatalf("worn pack node = %+v", worn)
	}
}

func TestShopIdentifiedBookUseConsumesSelectedNode(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Table = shopBookFixtureTable()
	f.Carried[0].Mage = true
	first := shopBookPool(f.Table, 1000)[0].Instance()
	spell, _ := first.BookSpell()
	second := first.Clone()
	first.ObjectID, second.ObjectID = 101, 202
	setShopIdentityPack(f, first, second)
	if action := screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 2}, ui.ShopControl{Kind: ui.ShopControlDoll}); action.Msg != "learned" {
		t.Fatalf("second book use = %+v", action)
	}
	requireShopPackIDs(t, f, 101)
	if got := f.Carried[0].KnownSpells; got != uint32(1)<<spell {
		t.Fatalf("known spells = %#x, want selected book's spell %d", got, spell)
	}
	if action := screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlDoll}); action.Msg != "learned" {
		t.Fatalf("remaining book use = %+v", action)
	}
	requireShopPackIDs(t, f)
}

func TestShopPackSelectionRequiresPresentIDAndKeepsCodeOnlyLookup(t *testing.T) {
	f, screen := shopRoom(t, nil)
	first, second := sim.PlainItem(0x0101), sim.PlainItem(0x0101)
	first.ObjectID, second.ObjectID = 101, 202
	setShopIdentityPack(f, first, second)
	missing := second.Clone()
	missing.ObjectID = 303
	if screen.shopTakeOnePackItem(missing) {
		t.Fatal("a missing selected ID removed an equal-valued node")
	}
	requireShopPackIDs(t, f, 101, 202)
	if !screen.shopTakeOnePackUnit(first.Code) {
		t.Fatal("code-only legacy take refused an available plain item")
	}
	requireShopPackIDs(t, f, 202)
}

func TestShopSelectedCountShortageLeavesPackAndTableUnchanged(t *testing.T) {
	for _, existingPlace := range []bool{false, true} {
		t.Run(map[bool]string{false: "new place", true: "existing place"}[existingPlace], func(t *testing.T) {
			f, screen := shopRoom(t, nil)
			first := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
			second := first.Clone()
			first.ObjectID, second.ObjectID = 101, 202
			setShopIdentityPack(f, first, second)
			f.Carried[0].Carry.LiveLoad = &sim.ActorLoadSnapshot{}
			// The selected stack claims two units, but only one belongs to its ID.
			// An equal-valued neighbor cannot supply the missing unit.
			f.Carried[0].Carry.OrderedStacks = []sim.ItemStack{sim.StackItem(second, 2)}
			if existingPlace && !f.Shop.PutOnTable(shopItemFromInstance(second, 1)) {
				t.Fatal("could not stage existing place")
			}
			beforeTable := f.Shop.Table()
			if action := clickShift(screen, ui.ShopControlPackCell, 1); action.Msg != "cannot change that pack" {
				t.Fatalf("shortage action = %+v", action)
			}
			requireShopPackIDs(t, f, 101, 202)
			if table := f.Shop.Table(); !reflect.DeepEqual(table, beforeTable) {
				t.Fatalf("shortage changed table: got %+v, want %+v", table, beforeTable)
			}
			if stack := f.Carried[0].Carry.OrderedStacks[0]; stack.ObjectID != 202 || stack.Count != 2 {
				t.Fatalf("shortage changed selected stack: %+v", stack)
			}
		})
	}
}

func TestShopShelfReturnKeepsObjectIdentities(t *testing.T) {
	for _, test := range []struct {
		name        string
		first, next sim.SavedObjectID
	}{
		{"distinct nodes", 101, 202},
		{"legacy and identified", 0, 202},
		{"legacy only", 0, 0},
		{"same node", 101, 101},
	} {
		t.Run(test.name, func(t *testing.T) {
			f, screen := shopRoom(t, nil)
			item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)]
			first, next := item.Clone(), item.Clone()
			first.ObjectID, first.Count = test.first, 1
			next.ObjectID, next.Count = test.next, 2
			f.Shop.shelves[ShelfArmour] = []ShopItem{first, next}
			click(screen, ui.ShopControlShelfPick, roomArmour)
			click(screen, ui.ShopControlShelfCell, 0)
			click(screen, ui.ShopControlShelfCell, 0)
			wantPlaces := 2
			if test.first == test.next {
				wantPlaces = 1
			}
			if table := f.Shop.Table(); len(table) != wantPlaces {
				t.Fatalf("staged nodes = %+v, want %d places", table, wantPlaces)
			}
			click(screen, ui.ShopControlButton, 0)
			want := map[sim.SavedObjectID]int32{}
			want[test.first]++
			want[test.next] += 2
			got := map[sim.SavedObjectID]int32{}
			for _, returned := range f.Shop.Shelf(ShelfArmour) {
				got[returned.ObjectID] += returned.Count
			}
			if !reflect.DeepEqual(got, want) || len(f.Shop.Shelf(ShelfArmour)) != len(want) || len(f.Shop.Table()) != 0 {
				t.Fatalf("returned shelf = %+v, want node counts %v and empty table", f.Shop.Shelf(ShelfArmour), want)
			}
		})
	}
}

func TestShopSaleKeepsEqualValuedNodesOnShelf(t *testing.T) {
	f, screen := shopRoom(t, nil)
	item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
	first, second := item.Clone(), item.Clone()
	first.ObjectID, second.ObjectID = 101, 202
	setShopIdentityPack(f, first, second)
	f.Shop.shelves[ShelfArmour] = nil
	click(screen, ui.ShopControlPackCell, 2)
	click(screen, ui.ShopControlPackCell, 1)
	click(screen, ui.ShopControlButton, 2)
	requireShopPackIDs(t, f)
	got := map[sim.SavedObjectID]int32{}
	for _, sold := range f.Shop.Shelf(ShelfArmour) {
		got[sold.ObjectID] += sold.Count
	}
	if want := map[sim.SavedObjectID]int32{101: 1, 202: 1}; !reflect.DeepEqual(got, want) || len(f.Shop.Table()) != 0 {
		t.Fatalf("sold shelf = %+v, table = %+v; want node counts %v", f.Shop.Shelf(ShelfArmour), f.Shop.Table(), want)
	}
}
