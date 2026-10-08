package game

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cityShopGraph(t *testing.T, f *FrontEnd) *cityObjectTopology {
	t.Helper()
	p, err := newCityObjectProjection(nil, f.Carried, f.Table)
	if err != nil {
		t.Fatal(err)
	}
	f.Town.cityObjects = p.graph
	return p.graph
}

func cityShopSetTestPack(t *testing.T, f *FrontEnd, index int, stacks ...sim.ItemStack) {
	t.Helper()
	member := mapload.MaterializePartyCarry(f.Carried[index], f.Table)
	if err := mapload.UpdatePartyLoadOrdered(member, &member, f.Table, false, false, stacks); err != nil {
		t.Fatal(err)
	}
	f.Carried[index] = member
}

func cityShopRoots(t *testing.T, f *FrontEnd, index int) cityPartyObjectRoots {
	t.Helper()
	root, err := cityMutationParty(f.Town.cityObjects, f.Carried[index].ID)
	if err != nil {
		t.Fatal(err)
	}
	return *root
}

func cityShopColdLoad(t *testing.T, f *FrontEnd) *FrontEnd {
	t.Helper()
	raw := cityProjectionSave(t, f)
	next := &FrontEnd{InstallResources: f.InstallResources}
	if _, town, err := next.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold city LOAD", town, err)
	}
	return next
}

func TestCityShopSelectsSecondEqualNodeAcrossTableEquipAndSAV(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{false: "zero native handles", true: "bound native handles"}[native], func(t *testing.T) {
			f, screen := shopRoom(t, nil)
			f.Town.open = true
			f.Carried[0].ID = "hero"
			f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
			f.Carried[0].Name = "Hero"
			f.Carried[0].Profile.Fighter = true
			f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
			item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
			slot, _ := EquipTarget(data.ItemCode(item.Code), f.Table)
			first, second := item.Clone(), item.Clone()
			if native {
				first.ObjectID, second.ObjectID = 101, 202
			}
			cityShopSetTestPack(t, f, 0, sim.StackItem(first, 1), sim.StackItem(second, 1))
			cityShopGraph(t, f)
			ids := slices.Clone(cityShopRoots(t, f, 0).Pack)
			if ids[0] == ids[1] || len(screen.shopPackStacks()) != 2 {
				t.Fatal("equal distinct nodes folded in pack display")
			}
			if a := screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 2}, ui.ShopControl{Kind: ui.ShopControlTableCell}); strings.Contains(a.Msg, "cannot") {
				t.Fatal(a)
			}
			if got := cityShopRoots(t, f, 0).Pack; !slices.Equal(got, ids[:1]) || len(f.Shop.table) != 1 || f.Shop.table[0].cityID != ids[1] {
				t.Fatal("selected second node did not reach table", got, f.Shop.table)
			}
			if a := screen.shopEquipFromTable(0); a.Msg != "worn" {
				t.Fatal(a)
			}
			if root := cityShopRoots(t, f, 0); root.Worn[slot-1] != ids[1] || !slices.Equal(root.Pack, ids[:1]) || f.Carried[0].Carry.EquippedItems[slot-1].ObjectID != second.ObjectID {
				t.Fatal("table equip selected another identity", root)
			}
			before, _, _ := mapload.PartyDisplayWithTable(f.Carried[0], f.Table)
			if before.Combat.Absorption != 65535 {
				t.Fatal("native fixture lost the current absorption word", before.Combat.Absorption)
			}
			fresh := cityShopColdLoad(t, f)
			if root := cityShopRoots(t, fresh, 0); root.Worn[slot-1] != ids[1] || !slices.Equal(root.Pack, ids[:1]) || mapload.MemberItemEquipment(fresh.Carried[0], fresh.Table)[slot-1].ObjectID != second.ObjectID {
				t.Fatal("SAVE/LOAD changed exact worn/pack identities", root)
			}
			for cycle := 1; cycle <= 2; cycle++ {
				after, _, _ := mapload.PartyDisplayWithTable(fresh.Carried[0], fresh.Table)
				if after.Combat.Absorption != before.Combat.Absorption {
					t.Fatalf("cycle %d changed current absorption: %d -> %d", cycle, before.Combat.Absorption, after.Combat.Absorption)
				}
				if cycle == 1 {
					fresh = cityShopColdLoad(t, fresh)
				}
			}
			screen = fresh.TownScreen().(*townScreen)
			if a := screen.shopUnequipToTable(slot); a.Msg != "on the table" {
				t.Fatal(a)
			}
			if fresh.Shop.table[0].cityID != ids[1] {
				t.Fatal("cold next removal changed node identity")
			}
		})
	}
}

func TestCityShopPartialSplitClonesOccurrencesAndPreservesAliases(t *testing.T) {
	raw, newFront := cityProjectionSource(t)
	f := cityProjectionLoad(t, raw, newFront)
	for i := range f.Carried {
		f.Carried[i].Carry.EquippedItems = [sim.EquipSlots]sim.ItemInstance{}
		f.Carried[i].Carry.Equipped = [sim.EquipSlots]uint16{}
	}
	item := f.Carried[0].Carry.OrderedStacks[0].Instance()
	for i := range f.Carried {
		cityShopSetTestPack(t, f, i, sim.StackItem(item, 3))
	}
	g := cityShopGraph(t, f)
	id := cityShopRoots(t, f, 0).Pack[0]
	other, _ := cityMutationParty(g, f.Carried[1].ID)
	other.Pack[0] = id
	row, _ := cityMutationSource(g, id, cityItemLocation{})
	g.Items[row].Effects[1] = g.Items[row].Effects[0]
	screen := f.TownScreen().(*townScreen)
	screen.shopMember = 0
	before := g.Clone()
	if a := screen.shopFromPack(0, false); strings.Contains(a.Msg, "cannot") {
		t.Fatal(a)
	}
	if len(f.Shop.table) != 1 || f.Shop.table[0].cityID != before.NextID || f.Shop.table[0].Count != 1 || f.Shop.table[0].ObjectID != 0 {
		t.Fatal("partial take did not create a separate zero-handle node", f.Shop.table)
	}
	newID := f.Shop.table[0].cityID
	newRow, _ := cityMutationSource(f.Town.cityObjects, newID, cityItemLocation{})
	copy := f.Town.cityObjects.Items[newRow]
	if copy.Effects[0] == copy.Effects[1] || copy.Effects[0] == before.Items[row].Effects[0] || copy.Spell == before.Items[row].Spell || f.Town.cityObjects.NextID != before.NextID+4 {
		t.Fatal("split shared children or reused allocation", copy)
	}
	for i := range f.Carried {
		if stacks := cityMemberStacks(f.Carried[i], f.Table); len(stacks) != 1 || stacks[0].Count != 2 || cityShopRoots(t, f, i).Pack[0] != id {
			t.Fatal("surviving alias lost its node or current quantity", i, stacks)
		}
	}
	if got := f.Shop.table[0].SourceEquipment.Spell; got.Present != true || got.ID != item.SourceEquipment.Spell.ID {
		t.Fatal("split lost owned Spell constructor", got)
	}
	if a := screen.shopClear(); strings.Contains(a.Msg, "cannot") {
		t.Fatal(a)
	}
	fresh := cityShopColdLoad(t, f)
	if !slices.Contains(cityShopRoots(t, fresh, 0).Pack, newID) || cityShopRoots(t, fresh, 1).Pack[0] != id {
		t.Fatal("cold SAVE/LOAD lost split or surviving alias")
	}
	if _, err := newCityObjectProjection(fresh.Town.cityObjects, fresh.Carried, fresh.Table); err != nil {
		t.Fatal("cold split violates shared scalar agreement", err)
	}
}

func TestCityShopSourceSelectedSplitThenExactNextRemoval(t *testing.T) {
	for _, native := range []bool{false, true} {
		t.Run(map[bool]string{false: "zero", true: "bound"}[native], func(t *testing.T) {
			f, screen, item := sourceCityEquipment(t)
			first, second := item.Clone(), item.Clone()
			if native {
				first.ObjectID, second.ObjectID = 101, 202
			}
			cityShopSetTestPack(t, f, screen.shopMemberIndex(), sim.StackItem(first, 1), sim.StackItem(second, 2))
			g := cityShopGraph(t, f)
			before := slices.Clone(cityShopRoots(t, f, screen.shopMemberIndex()).Pack)
			newID := g.NextID
			if a := screen.shopEquipFromPack(2); a.Msg != "worn" {
				t.Fatal(a)
			}
			m := f.Carried[screen.shopMemberIndex()]
			root := cityShopRoots(t, f, screen.shopMemberIndex())
			if !slices.Equal(root.Pack, before) || root.Worn[6] != newID || m.Carry.OrderedStacks[0].Count != 1 || m.Carry.OrderedStacks[1].Count != 1 {
				t.Fatal("source receipt changed selected occurrence or remaining counts", root, m.Carry.OrderedStacks)
			}
			wantNative := sim.SavedObjectID(0)
			if native {
				wantNative = newID
			}
			if m.Carry.EquippedItems[6].ObjectID != wantNative {
				t.Fatal("source receipt stamped or lost native handle")
			}
			fresh := cityShopColdLoad(t, f)
			screen = fresh.TownScreen().(*townScreen)
			if a := screen.shopUnequipToTable(7); a.Msg != "on the table" {
				t.Fatal(a)
			}
			if fresh.Shop.table[0].cityID != newID || fresh.Shop.table[0].ObjectID != wantNative || cityShopRoots(t, fresh, screen.shopMemberIndex()).Worn[6] != 0 {
				t.Fatal("cold source removal lost exact split identity")
			}
		})
	}
}

func TestCityShopNullPositionAndEqualBookConsumption(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Carried[0].ID, f.Carried[0].Mage = "hero", true
	f.Table = shopBookFixtureTable()
	item := shopBookPool(f.Table, 1000)[0].Instance()
	spell, _ := item.BookSpell()
	cityShopSetTestPack(t, f, 0, sim.StackItem(item, 1), sim.ItemStack{}, sim.StackItem(item, 1))
	cityShopGraph(t, f)
	before := slices.Clone(cityShopRoots(t, f, 0).Pack)
	if len(screen.shopPackStacks()) != 3 || before[1] != 0 || before[0] == before[2] {
		t.Fatal("null/equal nodes lost their explicit positions")
	}
	if a := screen.shopEquipFromPack(3); a.Msg != "learned" {
		t.Fatal(a)
	}
	if root := cityShopRoots(t, f, 0); !slices.Equal(root.Pack, before[:2]) || f.Carried[0].KnownSpells&(1<<spell) == 0 || len(f.Carried[0].Carry.ItemInstances) != 1 {
		t.Fatal("book consume removed a different equal node or null", root)
	}
	party, graph := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone()
	if a := screen.shopEquipFromPack(2); !strings.Contains(a.Msg, "no current quantity") || !reflect.DeepEqual(f.Carried, party) || !reflect.DeepEqual(graph, f.Town.cityObjects) {
		t.Fatal("null selection changed state", a)
	}
}

func TestCityShopMalformedInputsAndLateAllocationAreAtomic(t *testing.T) {
	for _, fault := range []string{"first allocation", "child allocation", "zero quantity", "underflow quantity", "stale location", "wrong identity", "conflicting shared value"} {
		t.Run(fault, func(t *testing.T) {
			f, screen := shopRoom(t, nil)
			f.Carried[0].ID = "hero"
			item := sim.ItemInstance{Code: 0x101, Kind: 2, Price: 10, Effects: []sim.ItemEffect{{Kind: 44, Operand: 7}, {Kind: 44, Operand: 7}}}
			cityShopSetTestPack(t, f, 0, sim.StackItem(item, 3), sim.StackItem(item, 3))
			g := cityShopGraph(t, f)
			switch fault {
			case "first allocation":
				g.NextID = ^sim.SavedObjectID(0)
			case "child allocation":
				g.NextID = ^sim.SavedObjectID(0) - 2
			case "zero quantity":
				f.Carried[0].Carry.OrderedStacks[1].Count = 0
			case "underflow quantity":
				f.Carried[0].Carry.OrderedStacks[1].Count = ^uint32(0)
			case "stale location":
				g.Roots[0].Pack = g.Roots[0].Pack[:1]
			case "wrong identity":
				g.Roots[0].Pack[1] = g.Effects[0]
			case "conflicting shared value":
				g.Roots[0].Pack[1] = g.Roots[0].Pack[0]
				f.Carried[0].Carry.OrderedStacks[1].Price++
			}
			party, graph, shop, gold := mapload.CloneParty(f.Carried), g.Clone(), cloneShopMutation(f.Shop), f.Town.Gold()
			if a := screen.shopFromPack(1, false); !strings.Contains(a.Msg, "cannot") {
				t.Fatal("malformed operation was admitted", a)
			}
			if !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || !reflect.DeepEqual(shop, cloneShopMutation(f.Shop)) || gold != f.Town.Gold() {
				t.Fatal("failed city transaction leaked graph, values, floor, stock or purse")
			}
		})
	}
}

func TestCityShopBuySellAndClearPreserveDistinctRootQuantities(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Carried[0].ID = "hero"
	cityShopSetTestPack(t, f, 0)
	cityShopGraph(t, f)
	item := ShopItem{Code: 0xe0d, Price: 5, Count: 3, Kind: 1, WeightPresent: true}
	other := item
	other.Code = 0xe0e
	f.Shop.shelves[ShelfMagic] = []ShopItem{item, other}
	screen.shopShelf = ShelfMagic
	for range 2 {
		if a := screen.shopFromShelf(0, true); strings.Contains(a.Msg, "cannot") {
			t.Fatal(a)
		}
	}
	if len(f.Shop.table) != 2 || f.Shop.table[0].cityID == f.Shop.table[1].cityID {
		t.Fatal("different merchant items folded on table", f.Shop.table)
	}
	ids := []sim.SavedObjectID{f.Shop.table[0].cityID, f.Shop.table[1].cityID}
	if a := screen.shopBuy(); !strings.Contains(a.Msg, "bought") || f.Town.Gold() != 470 {
		t.Fatal("buy quantity or exact cost changed", a, f.Town.Gold())
	}
	if root := cityShopRoots(t, f, 0); !slices.Equal(root.Pack, ids) || len(f.Carried[0].Carry.ItemInstances) != 6 {
		t.Fatal("buy folded distinct identities", root)
	}
	if a := screen.shopFromPack(1, true); strings.Contains(a.Msg, "cannot") || f.Shop.table[0].cityID != ids[1] || f.Shop.table[0].Count != 3 {
		t.Fatal("sale staging took wrong node", a, f.Shop.table)
	}
	if a := screen.shopSell(); !strings.Contains(a.Msg, "he pays 8;") || f.Town.Gold() != 478 {
		t.Fatal("sale did not round the selected whole quantity once", a, f.Town.Gold())
	}
	var sold ShopItem
	for _, shelf := range f.Shop.shelves {
		for _, v := range shelf {
			if v.cityID == ids[1] {
				sold = v
			}
		}
	}
	if sold.cityID != ids[1] || sold.Count != 3 || !slices.Equal(cityShopRoots(t, f, 0).Pack, ids[:1]) {
		t.Fatal("sale shelf lost exact identity/count or changed survivor", sold)
	}
	if a := screen.shopFromPack(0, true); strings.Contains(a.Msg, "cannot") {
		t.Fatal(a)
	}
	merchant := ShopItem{Code: 0xe0f, Price: 7, Count: 2, Kind: 1, WeightPresent: true}
	f.Shop.shelves[ShelfMagic] = []ShopItem{merchant}
	if a := screen.shopFromShelf(0, true); strings.Contains(a.Msg, "cannot") {
		t.Fatal(a)
	}
	merchantID := f.Shop.table[1].cityID
	if a := screen.shopClear(); a.Msg != "the table is cleared" || f.Town.Gold() != 478 {
		t.Fatal(a)
	}
	if root := cityShopRoots(t, f, 0); !slices.Equal(root.Pack, ids[:1]) || len(f.Shop.shelves[ShelfMagic]) != 1 || f.Shop.shelves[ShelfMagic][0].cityID != merchantID || f.Shop.shelves[ShelfMagic][0].Count != 2 {
		t.Fatal("clear changed owner-side routing or counts", root, f.Shop.shelves[ShelfMagic])
	}
}

func TestCityShopTableMergeRetainsDestinationAtFullCapacity(t *testing.T) {
	f := city1102Front(t, false)
	screen := f.TownScreen().(*townScreen)
	city1102Take(t, f, 1)
	target, floor := f.Shop.table[0].cityID, f.Town.cityObjects.NextID
	for i := 1; i < ShopTablePlaces; i++ {
		f.Shop.table = append(f.Shop.table, ShopPlace{ShopItem: ShopItem{Code: data.ItemCode(0xe20 + i), Count: 1, Price: 2}})
	}
	city1102Take(t, f, 2)
	if len(f.Shop.table) != ShopTablePlaces || f.Shop.table[0].cityID != target || f.Shop.table[0].Count != 3 || f.Town.cityObjects.NextID != floor+2 {
		t.Fatal("full table blocked or replaced an explicit merge destination", f.Shop.table)
	}
	for _, row := range f.Town.cityObjects.Items {
		if row.ID >= floor {
			t.Fatal("completed merge retained its unowned input", row.ID)
		}
	}
	if a := screen.shopSell(); !strings.Contains(a.Msg, "he pays 8;") || f.Town.Gold() != 3008 || len(f.Shop.table) != ShopTablePlaces-1 {
		t.Fatal("merged quantity changed original stack rounding or merchant places", a, f.Shop.table)
	}
	city1102NativeReload(t, f)
}

func TestCityShopShelfMergeKeepsTargetChildrenAndIncomingAliases(t *testing.T) {
	for _, bound := range []bool{false, true} {
		t.Run(map[bool]string{false: "zero", true: "bound"}[bound], func(t *testing.T) {
			f := city1102Front(t, false)
			source := sim.ItemInstance{Code: 0xe0d, Kind: 1, Price: 5, Weight: 7, WeightPresent: true, Effects: []sim.ItemEffect{{Kind: 44, Operand: 3}}}
			target := source.Clone()
			target.Price = 9
			if bound {
				target.ObjectID, source.ObjectID = 303, 707
			}
			cityShopSetTestPack(t, f, 0, sim.StackItem(target, 2), sim.StackItem(source, 1))
			cityShopSetTestPack(t, f, 1, sim.StackItem(source, 1))
			g := cityShopGraph(t, f)
			targetID, sourceID := g.Roots[0].Pack[0], g.Roots[0].Pack[1]
			g.Roots[1].Pack[0] = sourceID
			targetRow, _ := cityMutationSource(g, targetID, cityItemLocation{})
			sourceRow, _ := cityMutationSource(g, sourceID, cityItemLocation{})
			beforeTarget, beforeSource, floor := g.Items[targetRow], g.Items[sourceRow], g.NextID
			stock := shopItemFromInstance(target, 2)
			stock.cityID = targetID
			destinationShelf := shopShelfForItem(stock, f.Shop.tbl)
			f.Shop.shelves[destinationShelf] = []ShopItem{stock}
			screen := f.TownScreen().(*townScreen)
			if a := screen.shopFromPack(1, true); strings.Contains(a.Msg, "cannot") || f.Shop.table[0].cityID != sourceID {
				t.Fatal("staging selected a different source occurrence", a)
			}
			if a := screen.shopSell(); !strings.Contains(a.Msg, "he pays 3;") || f.Town.Gold() != 3003 {
				t.Fatal("sale did not pay the incoming item's price", a, f.Town.Gold())
			}
			shelf := f.Shop.shelves[destinationShelf]
			if len(shelf) != 1 || shelf[0].cityID != targetID || shelf[0].Count != 3 || !sim.StackStateEqual(sim.StackItem(shelf[0].Instance(), 1), sim.StackItem(target, 1)) {
				t.Fatal("shelf merge replaced target native values, child values or quantity", shelf)
			}
			if !slices.Equal(cityShopRoots(t, f, 0).Pack, []sim.SavedObjectID{targetID}) || f.Carried[0].Carry.OrderedStacks[0].Count != 3 ||
				!slices.Equal(cityShopRoots(t, f, 1).Pack, []sim.SavedObjectID{sourceID}) || !sim.StackStateEqual(f.Carried[1].Carry.OrderedStacks[0], sim.StackItem(source, 1)) ||
				!reflect.DeepEqual(f.Town.cityObjects.Items[targetRow], beforeTarget) || !reflect.DeepEqual(f.Town.cityObjects.Items[sourceRow], beforeSource) || f.Town.cityObjects.NextID != floor {
				t.Fatal("merge redirected surviving roots, retired live children or reused IDs")
			}
			city1102NativeReload(t, f)
			screen = f.TownScreen().(*townScreen)
			if a := screen.shopFromPack(0, true); strings.Contains(a.Msg, "cannot") || f.Shop.table[0].cityID != targetID || f.Shop.table[0].Count != 3 {
				t.Fatal("cold next operation lost merged identity or quantity", a)
			}
		})
	}
}

func TestCityShopShelfMergeFailurePreservesPurseValuesAndFloor(t *testing.T) {
	for _, exhausted := range []bool{false, true} {
		t.Run(map[bool]string{false: "quantity overflow", true: "target allocation overflow"}[exhausted], func(t *testing.T) {
			f := city1102Front(t, false)
			city1102Take(t, f, 1)
			stock := f.Shop.table[0].ShopItem.Clone()
			stock.cityID, stock.cityOrigins = 0, nil
			stock.Count = sim.MaxOriginalHoldingValues
			if exhausted {
				stock.Count = 1
				f.Town.cityObjects.NextID = ^sim.SavedObjectID(0)
			}
			f.Shop.shelves[shopShelfForItem(stock, f.Shop.tbl)] = []ShopItem{stock}
			party, graph, shop, gold := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.cityBookCandidate().Shop, f.Town.Gold()
			if a := f.TownScreen().(*townScreen).shopSell(); !strings.Contains(a.Msg, "cannot") {
				t.Fatal("invalid merge succeeded", a)
			}
			if !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || !reflect.DeepEqual(shop, f.Shop) || f.Town.Gold() != gold {
				t.Fatal("late merge failure published a prefix")
			}
		})
	}
}

func TestCityShopMergedSaleRefreshesEveryExplicitSourceOwner(t *testing.T) {
	f := city1102Front(t, false)
	item := f.Carried[0].Carry.OrderedStacks[0].Instance()
	for i := range f.Carried {
		cityShopSetTestPack(t, f, i, sim.StackItem(item, 3))
	}
	cityShopGraph(t, f)
	before := mapload.CloneParty(f.Carried)
	screen := f.TownScreen().(*townScreen)
	for i := range f.Carried {
		screen.shopMember = i
		if a := screen.shopFromPack(0, false); strings.Contains(a.Msg, "cannot") {
			t.Fatal(a)
		}
	}
	if len(f.Shop.table) != 1 || f.Shop.table[0].Count != 2 || len(f.Shop.table[0].cityOrigins) != 2 {
		t.Fatal("merged table lost explicit source owners", f.Shop.table)
	}
	if a := screen.shopSell(); !strings.Contains(a.Msg, "he pays 5;") || f.Town.Gold() != 3005 {
		t.Fatal("merged source-owner sale", a, f.Town.Gold())
	}
	for i := range f.Carried {
		want := currentCitySaleHuman(t, before[i])
		want.InventoryWeight -= 7
		want.Load = uint16(int16(want.Weight) + int16(want.InventoryWeight/2))
		if got := currentCitySaleHuman(t, f.Carried[i]); got != want || f.Carried[i].Carry.OrderedStacks[0].Count != 2 {
			t.Fatal("sale did not refresh the exact removed source owner", i, got, want)
		}
	}
	city1102NativeReload(t, f)
}

func TestCityShopAliasedWholeTableRoundtripKeepsOccurrences(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Carried[0].ID = "hero"
	item := sim.ItemInstance{Code: 0xe0d, Price: 5, Kind: 1, WeightPresent: true}
	cityShopSetTestPack(t, f, 0, sim.StackItem(item, 1), sim.StackItem(item, 1))
	g := cityShopGraph(t, f)
	g.Roots[0].Pack[1] = g.Roots[0].Pack[0]
	id := g.Roots[0].Pack[0]
	for cycle := 0; cycle < 2; cycle++ {
		if a := screen.shopFromPack(1, true); strings.Contains(a.Msg, "cannot") {
			t.Fatal(a)
		}
		if f.Shop.table[0].cityID != id || !slices.Equal(cityShopRoots(t, f, 0).Pack, []sim.SavedObjectID{id}) {
			t.Fatal("moving an alias removed the surviving occurrence")
		}
		if a := screen.shopOffTable(0, true); strings.Contains(a.Msg, "cannot") {
			t.Fatal(a)
		}
		if root := cityShopRoots(t, f, 0); !slices.Equal(root.Pack, []sim.SavedObjectID{id, id}) || len(cityMemberStacks(f.Carried[0], f.Table)) != 2 {
			t.Fatal("table return folded repeated same-node roots", root)
		}
	}
}

func TestCityShopSourceOwnedSpellWritesFollowAliasedEvictionOrder(t *testing.T) {
	f, screen, _ := sourceCityEquipment(t)
	i := screen.shopMemberIndex()
	spell := sim.SourceItemSpell{Present: true, ID: 1, Range: 7, Defensive: 2, ManaCost: 93}
	weapon := sim.ItemInstance{Code: 0x101, Kind: 2, Price: 10, WeightPresent: true, Weight: 2,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 1, OwnKind: 1, Spell: spell,
			Definition: sim.SourceWeaponDefinition{Present: true, AttackType: 1, Hands: 1, Charge: 8, Relax: 4}}}
	cityShopSetTestPack(t, f, i, sim.StackItem(weapon, 1))
	f.Carried[i].Carry.EquippedItems[0], f.Carried[i].Carry.Equipped[0] = weapon, weapon.Code
	f.Carried[i].Book.State, f.Carried[i].KnownSpells = sim.BookPresent, 1<<1
	f.Carried[i].SpellbookRestored, f.Carried[i].SpellbookPresent = true, true
	f.Carried[i].Book.Slots[0] = sim.BookSpell{Range: 7, Defensive: 2, ManaCost: 93}
	g := cityShopGraph(t, f)
	root, _ := cityMutationParty(g, f.Carried[i].ID)
	id := root.Worn[0]
	root.Pack[0] = id
	row, _ := cityMutationSource(g, id, cityItemLocation{})
	oldSpell := g.Items[row].Spell
	for j := range g.Books {
		if string(g.Books[j].PartyID) == f.Carried[i].ID {
			g.Books[j].Slots[0] = oldSpell
		}
	}
	beforeBook := f.Carried[i].Book
	if a := screen.shopEquipFromPack(1); a.Msg != "worn" {
		t.Fatal(a)
	}
	root = nil
	got := cityShopRoots(t, f, i)
	row, _ = cityMutationSource(f.Town.cityObjects, id, cityItemLocation{})
	if got.Worn[0] != id || !slices.Equal(got.Pack, []sim.SavedObjectID{id}) || f.Town.cityObjects.Items[row].Spell != 0 {
		t.Fatal("same-node eviction did not clear the current owned Spell edge", got, f.Town.cityObjects.Items[row])
	}
	if f.Carried[i].Carry.OrderedStacks[0].SourceEquipment.Spell.Present || f.Carried[i].Carry.EquippedItems[0].SourceEquipment.Spell.Present || f.Carried[i].Book != beforeBook || cityBookSlot(t, f.Town.cityObjects, f.Carried[i].ID, 0) != oldSpell {
		t.Fatal("owned Spell teardown changed survivor book or chose an obsolete item view")
	}
	fresh := cityShopColdLoad(t, f)
	if fresh.Carried[i].Book != beforeBook || cityShopRoots(t, fresh, i).Worn[0] != id {
		t.Fatal("shared child survivor or repeated Item root lost on cold LOAD")
	}
}

func TestCityShopSourceWornQuantitySurvivesColdMerge(t *testing.T) {
	for _, mode := range []string{"empty pack", "pack target", "shared worn target"} {
		t.Run(mode, func(t *testing.T) {
			f, screen, _ := sourceCityEquipment(t)
			i := screen.shopMemberIndex()
			if a := screen.shopEquipFromPack(1); a.Msg != "worn" {
				t.Fatal(a)
			}
			if mode == "empty pack" {
				cityShopSetTestPack(t, f, i)
			}
			if mode == "shared worn target" {
				for _, id := range []string{"passive-one", "passive-two"} {
					member := mapload.CloneParty([]mapload.PartyMember{f.Carried[i]})[0]
					member.ID, member.StartingHero, member.PlayerCharacter = id, false, false
					f.Carried = append(f.Carried, member)
					cityShopSetTestPack(t, f, len(f.Carried)-1)
				}
			}
			g := cityShopGraph(t, f)
			root := cityShopRoots(t, f, i)
			wornID := root.Worn[6]
			row, _ := cityMutationSource(g, wornID, cityItemLocation{})
			g.Items[row].WornCount = 2
			wantID, wantCount := wornID, uint32(2)
			if mode != "empty pack" {
				wantID, wantCount = root.Pack[0], 3
			}
			if mode == "shared worn target" {
				for _, id := range []string{"passive-one", "passive-two"} {
					passive, _ := cityMutationParty(g, id)
					passive.Worn[6] = wantID
				}
			}
			f = cityShopColdLoad(t, f)
			screen = f.TownScreen().(*townScreen)
			if count, err := screen.cityShopNodeCount(wornID); err != nil || count != 2 {
				t.Fatal("cold Worn-only quantity", count, err)
			}
			floor := f.Town.cityObjects.NextID
			if a := screen.shopUnequipDoll(7); a.Msg != "off, into the pack" {
				t.Fatal("whole Worn quantity return", a)
			}
			root = cityShopRoots(t, f, i)
			if root.Worn[6] != 0 || !slices.Equal(root.Pack, []sim.SavedObjectID{wantID}) || len(f.Carried[i].Carry.OrderedStacks) != 1 ||
				f.Carried[i].Carry.OrderedStacks[0].Count != wantCount || f.Carried[i].Carry.OrderedStacks[0].ObjectID != 0 || f.Town.cityObjects.NextID != floor {
				t.Fatal("source return lost quantity, destination, native-zero handle or floor", root, f.Carried[i].Carry.OrderedStacks)
			}
			if mode == "shared worn target" {
				for _, id := range []string{"passive-one", "passive-two"} {
					passive, _ := cityMutationParty(f.Town.cityObjects, id)
					if passive.Worn[6] != wantID {
						t.Fatal("merge detached a surviving worn alias")
					}
				}
			}
			city1102NativeReload(t, f)
		})
	}
}

func TestCityShopNativeWornQuantityMovesAndMergesAcrossSAV(t *testing.T) {
	for _, mode := range []string{"empty pack", "pack target", "replacement"} {
		t.Run(mode, func(t *testing.T) {
			f, screen := shopRoom(t, nil)
			f.Town.open = true
			f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
			f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
			f.Carried[0].Profile.Fighter = true
			f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
			item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
			slot, _ := EquipTarget(data.ItemCode(item.Code), f.Table)
			cityShopSetTestPack(t, f, 0, sim.StackItem(item, 1), sim.StackItem(item, 1))
			cityShopGraph(t, f)
			if a := screen.shopEquipFromPack(1); a.Msg != "worn" {
				t.Fatal(a)
			}
			if mode == "empty pack" {
				cityShopSetTestPack(t, f, 0)
			}
			g := cityShopGraph(t, f)
			root := cityShopRoots(t, f, 0)
			wornID := root.Worn[slot-1]
			row, _ := cityMutationSource(g, wornID, cityItemLocation{})
			g.Items[row].WornCount = 2
			wantPack, wantCount, wantWorn := wornID, uint32(2), sim.SavedObjectID(0)
			if mode == "pack target" {
				wantPack, wantCount = root.Pack[0], 3
			} else if mode == "replacement" {
				wantWorn = root.Pack[0]
			}
			if mapload.HasSourceActor(f.Carried[0]) {
				t.Fatal("fixture no longer reaches native equipment mutation")
			}
			floor := g.NextID
			if mode == "replacement" {
				if a := screen.shopEquipFromPack(1); a.Msg != "worn" {
					t.Fatal(a)
				}
			} else if a := screen.shopUnequipDoll(slot); a.Msg != "off, into the pack" {
				t.Fatal(a)
			}
			for cycle := 0; cycle < 2; cycle++ {
				root = cityShopRoots(t, f, 0)
				stacks := cityMemberStacks(f.Carried[0], f.Table)
				if root.Worn[slot-1] != wantWorn || !slices.Equal(root.Pack, []sim.SavedObjectID{wantPack}) || len(stacks) != 1 || stacks[0].Count != wantCount || stacks[0].ObjectID != 0 || f.Town.cityObjects.NextID != floor {
					t.Fatal("native move changed current count, root or floor", cycle, root, stacks)
				}
				f = cityShopColdLoad(t, f)
			}
		})
	}
}

func TestCityShopSourcePartialAliasesUseTheOriginalTakeBoundary(t *testing.T) {
	for _, passiveFault := range []bool{false, true} {
		t.Run(map[bool]string{false: "all owners", true: "passive owner refuses"}[passiveFault], func(t *testing.T) {
			f, screen, item := sourceCityEquipment(t)
			i := screen.shopMemberIndex()
			cityShopSetTestPack(t, f, i, sim.StackItem(item, 2), sim.ItemStack{}, sim.StackItem(item, 2))
			passive := mapload.CloneParty([]mapload.PartyMember{f.Carried[i]})[0]
			passive.ID, passive.Name = "passive", "Passive"
			f.Carried = append(f.Carried, passive)
			j := len(f.Carried) - 1
			cityShopSetTestPack(t, f, j, sim.StackItem(item, 2))
			if passiveFault {
				f.Carried[j].Carry.LiveLoad.Capacity = 0
				f.Carried[j].Carry.LiveLoad.Movement.Capacity = 0
				f.Carried[j].Carry.LiveLoad.Inventory.Source.Stats[7] = 0
			}
			g := cityShopGraph(t, f)
			root, _ := cityMutationParty(g, f.Carried[i].ID)
			id := root.Pack[0]
			root.Pack[2] = id
			other, _ := cityMutationParty(g, passive.ID)
			other.Pack[0] = id
			before, graph := mapload.CloneParty(f.Carried), g.Clone()
			expected, _, ok := mapload.SourceTownEquipment(before[i], f.Table, 2, 7, sim.ItemInstance{}, false, sim.SourceEquipmentOperation{PackAliases: []int{0}})
			if !ok {
				t.Fatal("literal explicit alias operation failed")
			}
			a := screen.shopEquipFromPack(3)
			if passiveFault {
				if !strings.Contains(a.Msg, "divide by zero") || !reflect.DeepEqual(before, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) {
					t.Fatal("passive alias refusal leaked successful active prefix", a)
				}
				return
			}
			if a.Msg != "worn" {
				t.Fatal(a)
			}
			got := f.Carried[i]
			if !reflect.DeepEqual(got.Carry.LiveLoad, expected.Carry.LiveLoad) || got.Hero != expected.Hero || got.Carry.LiveLoad.Inventory.Accumulator != 406 {
				t.Fatal("city moved additional alias delta across source derive boundary", got.Carry.LiveLoad, expected.Carry.LiveLoad)
			}
			if !slices.Equal(cityShopRoots(t, f, i).Pack, []sim.SavedObjectID{id, 0, id}) || got.Carry.OrderedStacks[0].Count != 1 || got.Carry.OrderedStacks[2].Count != 1 || f.Carried[j].Carry.OrderedStacks[0].Count != 1 {
				t.Fatal("split lost null position or alias counts")
			}
		})
	}
}

func TestCityShopConsumableClickKeyUsesTopologyIdentity(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Carried[0].ID = "hero"
	item := sim.ItemInstance{Code: 0xe0d, Kind: 3, Price: 10}
	cityShopSetTestPack(t, f, 0, sim.StackItem(item, 1), sim.StackItem(item, 1))
	cityShopGraph(t, f)
	view := screen.ShopScreen()
	if view.Pack[1].UseItemKey == "" || view.Pack[1].UseItemKey == view.Pack[2].UseItemKey {
		t.Fatal("equal zero-handle consumables share a double-click identity")
	}
}

func TestCityShopDollTableMergePreservesDestinationAndAliases(t *testing.T) {
	for _, origin := range []string{"native", "native cold", "source"} {
		for _, mode := range []string{"room", "full", "different price", "merchant"} {
			t.Run(origin+"/"+mode, func(t *testing.T) {
				var f *FrontEnd
				var screen *townScreen
				var item sim.ItemInstance
				var slot int
				if origin == "source" {
					f, screen, item = sourceCityEquipment(t)
					slot = int(item.SourceEquipment.OwnKind)
				} else {
					f, screen = shopRoom(t, nil)
					f.Town.open = true
					f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
					f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
					f.Carried[0].Profile.Fighter = true
					f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
					item = f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
					slot, _ = EquipTarget(data.ItemCode(item.Code), f.Table)
				}
				cityShopSetTestPack(t, f, 0, sim.StackItem(item, 1), sim.StackItem(item, 1))
				cityShopGraph(t, f)
				if a := screen.shopEquipFromPack(1); a.Msg != "worn" {
					t.Fatal(a)
				}
				ownerIndices := map[string]int{}
				for j, id := range []string{"target-owner", "source-owner"} {
					member := mapload.CloneParty([]mapload.PartyMember{f.Carried[0]})[0]
					member.ID, member.StartingHero, member.PlayerCharacter = id, false, false
					f.Carried = append(f.Carried, member)
					ownerIndices[id] = len(f.Carried) - 1
					cityShopSetTestPack(t, f, ownerIndices[id], sim.StackItem(item, uint32(j+1)))
				}
				g := cityShopGraph(t, f)
				root, _ := cityMutationParty(g, "hero")
				targetID, sourceID := root.Pack[0], root.Worn[slot-1]
				for j, id := range []string{"target-owner", "source-owner"} {
					other, _ := cityMutationParty(g, id)
					node := []sim.SavedObjectID{targetID, sourceID}[j]
					other.Pack[0], other.Worn[slot-1] = node, node
				}
				if origin == "native cold" {
					f = cityShopColdLoad(t, f)
					screen = f.TownScreen().(*townScreen)
				}
				if mapload.HasSourceActor(f.Carried[0]) != (origin == "source") {
					t.Fatal("fixture lost its distinct operation route")
				}
				target := shopItemFromInstance(cityMemberStacks(f.Carried[0], f.Table)[0].Instance(), 1)
				target.cityID = targetID
				place := ShopPlace{ShopItem: target, Mine: mode != "merchant", From: ShelfArmour}
				if mode == "different price" {
					place.Price++
					place.cityID = 0
				}
				f.Shop.table = []ShopPlace{place}
				if mode != "room" {
					for j := 1; j < ShopTablePlaces; j++ {
						f.Shop.table = append(f.Shop.table, ShopPlace{ShopItem: ShopItem{Code: data.ItemCode(0xe50 + j), Count: 1, Price: 2}})
					}
				}
				before, graph, shop, gold := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.cityBookCandidate().Shop, f.Town.Gold()
				a := screen.shopUnequipToTable(slot)
				if mode == "different price" || mode == "merchant" {
					if a.Msg != "" || !reflect.DeepEqual(before, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || !reflect.DeepEqual(shop, f.Shop) || f.Town.Gold() != gold {
						t.Fatal("ineligible full-table removal committed a prefix", a)
					}
					return
				}
				if a.Msg != "on the table" || len(f.Shop.table) != len(shop.table) || f.Shop.table[0].cityID != targetID || f.Shop.table[0].Count != 3 || f.Town.Gold() != gold || f.Town.cityObjects.NextID != graph.NextID {
					t.Fatal("eligible doll drop lost destination, quantity or capacity", a, f.Shop.table)
				}
				root, _ = cityMutationParty(f.Town.cityObjects, "hero")
				if root.Worn[slot-1] != 0 || !slices.Equal(root.Pack, []sim.SavedObjectID{targetID}) {
					t.Fatal("doll drop moved the wrong root", root)
				}
				for j, id := range []string{"target-owner", "source-owner"} {
					other, _ := cityMutationParty(f.Town.cityObjects, id)
					wantID, count := targetID, uint32(3)
					if j == 1 {
						wantID, count = sourceID, 2
					}
					stack := cityMemberStacks(f.Carried[ownerIndices[id]], f.Table)[0]
					if !slices.Equal(other.Pack, []sim.SavedObjectID{wantID}) || other.Worn[slot-1] != wantID || stack.Count != count || stack.ObjectID != 0 || mapload.MemberItemEquipment(f.Carried[ownerIndices[id]], f.Table)[slot-1].ObjectID != 0 {
						t.Fatal("merge rewrote a surviving shared occurrence", id, other, stack)
					}
				}
				if cityMemberStacks(f.Carried[0], f.Table)[0].Count != 3 || len(f.Town.cityObjects.Items) != 2 {
					t.Fatal("merge missed the active target alias or retained unowned fixture nodes")
				}
				for _, id := range []sim.SavedObjectID{targetID, sourceID} {
					oldAt, oldErr := cityMutationSource(graph, id, cityItemLocation{})
					newAt, newErr := cityMutationSource(f.Town.cityObjects, id, cityItemLocation{})
					if oldErr != nil || newErr != nil || !reflect.DeepEqual(graph.Items[oldAt], f.Town.cityObjects.Items[newAt]) {
						t.Fatal("merge replaced a surviving item's child identities", id, oldErr, newErr)
					}
				}
			})
		}
	}
}

// A flat pack shows and saves equal units as one cell (ITEM-MERGE-129). A
// graph naming one record per unit, as an older SAV may, still trades.
func TestCityFlatPackFoldsEqualUnits(t *testing.T) {
	axe, other := uint16(0x0101), uint16(0x0122)
	f, screen := shopRoom(t, []uint16{axe, other, axe})
	f.Town.open = true
	f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	flat := mapload.CloneParty(f.Carried)[0]
	folded := func(stacks []sim.ItemStack) bool {
		return len(stacks) == 2 && stacks[0].Code == axe && stacks[0].Count == 2 && stacks[1].Code == other && stacks[1].Count == 1
	}
	cityShopGraph(t, f)
	if got := screen.shopPackStacks(); !folded(got) || len(cityShopRoots(t, f, 0).Pack) != 2 {
		t.Fatalf("pack cells %+v roots %v; want the axe x2 and the other item x1", got, cityShopRoots(t, f, 0).Pack)
	}
	fresh := cityShopColdLoad(t, f)
	if got := cityMemberStacks(fresh.Carried[0], fresh.Table); !folded(got) || len(cityShopRoots(t, fresh, 0).Pack) != 2 {
		t.Fatalf("cold LOAD pack cells %+v roots %v", got, cityShopRoots(t, fresh, 0).Pack)
	}

	unit := func(code uint16) sim.ItemStack { return sim.StackItem(sim.PlainItem(code), 1) }
	cityShopSetTestPack(t, f, 0, unit(axe), unit(other), unit(axe))
	cityShopGraph(t, f)
	perUnit := slices.Clone(cityShopRoots(t, f, 0).Pack)
	f.Carried[0] = flat
	if len(perUnit) != 3 || !folded(screen.shopPackStacks()) {
		t.Fatalf("fixture roots %v cells %+v", perUnit, screen.shopPackStacks())
	}
	if a := screen.shopFromPack(0, true); strings.Contains(a.Msg, "cannot") {
		t.Fatal(a)
	}
	if len(f.Shop.table) != 1 || uint16(f.Shop.table[0].Code) != axe || f.Shop.table[0].Count != 2 || f.Shop.table[0].cityID != perUnit[0] {
		t.Fatalf("table %+v; want the axe x2 as record %d", f.Shop.table, perUnit[0])
	}
	if root := cityShopRoots(t, f, 0).Pack; len(root) != 1 || len(cityMemberStacks(f.Carried[0], f.Table)) != 1 {
		t.Fatalf("pack roots %v after the whole cell left", root)
	}
}

// A bought axe joins the pack's equal axe whose blocks and weight a SAV
// restored, a block byte no constructor writes and a weight the tables do not
// give the axe included, and the cell keeps them through a cold LOAD
// (ITEM-STACK-003, ITEM-MERGE-129). An enchanted axe keeps its own cell.
func TestCityPurchaseJoinsTheRestoredItem(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Town.open = true
	f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	axe := ShopItem{Code: 0x0101, Kind: 2, Price: 40, Count: 3}
	magic := axe
	magic.Price, magic.Effects = 400, []sim.ItemEffect{{Kind: 12, Operand: 5}}
	f.Shop.shelves[ShelfWeapons] = []ShopItem{axe, magic}
	restored := axe.Instance()
	restored.WeightPresent, restored.Weight = true, 9
	restored.SourceEquipment = sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 1, OwnKind: 1}
	restored.SourceEquipment.Attack[22] = 15
	cityShopSetTestPack(t, f, 0, sim.StackItem(restored, 1))
	cityShopGraph(t, f)
	screen.shopShelf = ShelfWeapons
	for _, i := range []int{0, 1} {
		if a := screen.shopFromShelf(i, false); strings.Contains(a.Msg, "cannot") {
			t.Fatal(a)
		}
	}
	if a := screen.shopBuy(); !strings.Contains(a.Msg, "bought") {
		t.Fatal(a)
	}
	check := func(label string, stacks []sim.ItemStack) {
		t.Helper()
		if len(stacks) != 2 || stacks[0].Count != 2 || stacks[0].Weight != 9 || stacks[0].SourceEquipment.Attack[22] != 15 ||
			stacks[1].Count != 1 || !stacks[1].Instance().HasEnchantment() {
			t.Fatalf("%s %+v; want the restored axe x2 and the enchanted axe x1", label, stacks)
		}
	}
	check("pack", screen.shopPackStacks())
	fresh := cityShopColdLoad(t, f)
	check("cold LOAD pack", cityMemberStacks(fresh.Carried[0], fresh.Table))
}

// A bought axe that the shelf holds with its own saved Weapon bytes 22 and 23
// joins the pack's axe that differs from it only there, and the cell keeps the
// pack axe's two bytes through a cold LOAD (ITEM-STACK-003, ITEM-MERGE-129,
// DIV-762). An axe that differs in another block byte keeps its own cell.
func TestCityPurchaseJoinsTheAxeDifferingInWeaponBytes22And23(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Town.open = true
	f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	restored := sim.ItemInstance{Code: 0x0101, Kind: 2, Price: 40, WeightPresent: true, Weight: 9,
		SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 1, OwnKind: 1}}
	restored.SourceEquipment.Attack[22], restored.SourceEquipment.Attack[23] = 15, 1
	shelved := func(a, b, other byte) ShopItem {
		item := shopItemFromInstance(restored, 3)
		item.SourceEquipment.Attack[22], item.SourceEquipment.Attack[23] = a, b
		item.SourceEquipment.Attack[21] = other
		return item
	}
	f.Shop.shelves[ShelfWeapons] = []ShopItem{shelved(3, 4, 0), shelved(5, 6, 7)}
	cityShopSetTestPack(t, f, 0, sim.StackItem(restored, 1))
	cityShopGraph(t, f)
	screen.shopShelf = ShelfWeapons
	for _, i := range []int{0, 1} {
		if a := screen.shopFromShelf(i, false); strings.Contains(a.Msg, "cannot") {
			t.Fatal(a)
		}
	}
	if a := screen.shopBuy(); !strings.Contains(a.Msg, "bought") {
		t.Fatal(a)
	}
	check := func(label string, stacks []sim.ItemStack) {
		t.Helper()
		if len(stacks) != 2 || stacks[0].Count != 2 || stacks[0].SourceEquipment != restored.SourceEquipment ||
			stacks[1].Count != 1 || stacks[1].SourceEquipment.Attack[21] != 7 {
			t.Fatalf("%s %+v; want the pack axe x2 with its own bytes and the axe with another block byte x1", label, stacks)
		}
	}
	check("pack", screen.shopPackStacks())
	fresh := cityShopColdLoad(t, f)
	check("cold LOAD pack", cityMemberStacks(fresh.Carried[0], fresh.Table))
}

// A sold shield that is exactly what the definition constructor makes of its
// code holds no saved operand, so it joins the shelf's plain shield cell
// (DIV-322, ITEM-MERGE-129).
func TestCitySoldItemJoinsTheShelfCell(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Town.open = true
	f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	defs := eqDefsTable(t)
	f.Table.Shapes, f.Table.Materials, f.Table.Weapons, f.Table.Shields, f.Table.Armors = defs.Shapes, defs.Materials, defs.Weapons, defs.Shields, defs.Armors
	plain := sim.ItemInstance{Code: eqShieldCode, Kind: 1, Price: 701}
	built := mapload.SourceConstructedItem(plain, f.Table)
	if !built.WeightPresent || built.SourceEquipment.Class != sim.SourceShield {
		t.Fatalf("constructed shield %+v", built)
	}
	f.Shop.shelves[ShelfArmour] = []ShopItem{shopItemFromInstance(plain, 2)}
	cityShopSetTestPack(t, f, 0, sim.StackItem(built, 1))
	cityShopGraph(t, f)
	if a := screen.shopFromPack(0, true); strings.Contains(a.Msg, "cannot") {
		t.Fatal(a)
	}
	if a := screen.shopSell(); !strings.Contains(a.Msg, "he pays") {
		t.Fatal(a)
	}
	if shelf := f.Shop.Shelf(ShelfArmour); len(shelf) != 1 || shelf[0].Count != 3 || shelf[0].WeightPresent {
		t.Fatalf("armour shelf %+v; want the plain shield x3", shelf)
	}
}

// A bought potion joins the pack's equal potion that a SAV restored with the
// per-unit weight 1 (ITEM-STACK-003), a weight the tables do not give a
// potion, and the cell keeps it through a cold LOAD (ITEM-MERGE-129).
func TestCityPurchaseJoinsTheRestoredPotion(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Town.open = true
	f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	potion := ShopItem{Code: 0x0e06, Kind: 3, Price: 50, Count: 3, Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 5}}}
	f.Shop.shelves[ShelfBooks] = []ShopItem{potion}
	restored := potion.Instance()
	restored.WeightPresent, restored.Weight = true, 1
	cityShopSetTestPack(t, f, 0, sim.StackItem(restored, 1))
	cityShopGraph(t, f)
	screen.shopShelf = ShelfBooks
	if a := screen.shopFromShelf(0, false); strings.Contains(a.Msg, "cannot") {
		t.Fatal(a)
	}
	if a := screen.shopBuy(); !strings.Contains(a.Msg, "bought") {
		t.Fatal(a)
	}
	check := func(label string, stacks []sim.ItemStack) {
		t.Helper()
		if len(stacks) != 1 || stacks[0].Count != 2 || !stacks[0].WeightPresent || stacks[0].Weight != 1 {
			t.Fatalf("%s %+v; want the restored potion x2 at weight 1", label, stacks)
		}
	}
	check("pack", screen.shopPackStacks())
	fresh := cityShopColdLoad(t, f)
	check("cold LOAD pack", cityMemberStacks(fresh.Carried[0], fresh.Table))
}

// A sold potion that a SAV restored at the per-unit weight 1 (ITEM-STACK-003)
// holds no saved operand, since the city SAV writer gives every unweighted
// potion that weight, so it joins the shelf's plain potion cell (DIV-322,
// ITEM-MERGE-129).
func TestCitySoldRestoredPotionJoinsTheShelfCell(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Town.open = true
	f.Carried[0].ID, f.Carried[0].Name = "hero", "Hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	potion := sim.ItemInstance{Code: 0x0e06, Kind: 3, Price: 50, Effects: []sim.ItemEffect{{Kind: 8, Mode: 1, Operand: 5}}}
	restored := potion.Clone()
	restored.WeightPresent, restored.Weight = true, 1
	f.Shop.shelves[ShelfBooks] = []ShopItem{shopItemFromInstance(potion, 2)}
	cityShopSetTestPack(t, f, 0, sim.StackItem(restored, 1))
	cityShopGraph(t, f)
	if a := screen.shopFromPack(0, true); strings.Contains(a.Msg, "cannot") {
		t.Fatal(a)
	}
	if a := screen.shopSell(); !strings.Contains(a.Msg, "he pays") {
		t.Fatal(a)
	}
	if shelf := f.Shop.Shelf(ShelfBooks); len(shelf) != 1 || shelf[0].Count != 3 || shelf[0].WeightPresent {
		t.Fatalf("fourth shelf %+v; want the plain potion x3", shelf)
	}
}
