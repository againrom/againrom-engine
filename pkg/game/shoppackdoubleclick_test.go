package game

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestCityShopEquipmentTakesTheDoubleClickKey: on the city item graph a pack
// item with an equipment slot carries a double-click key as a potion does, so
// the second click of a double click wears it instead of landing on the cell
// the first click emptied. Each of two equal items has its own key; an item
// with no slot keeps the immediate single click.
func TestCityShopEquipmentTakesTheDoubleClickKey(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Carried[0].ID = "hero"
	wearable := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
	plain := sim.ItemInstance{Code: 0xe0d, Kind: 5, Price: 10}
	if _, ok := EquipTarget(data.ItemCode(plain.Code), f.Table); ok {
		t.Fatal("the fixture's plain item has an equipment slot")
	}
	cityShopSetTestPack(t, f, 0, sim.StackItem(wearable, 1), sim.StackItem(wearable, 1), sim.StackItem(plain, 1))
	cityShopGraph(t, f)
	view := screen.ShopScreen()
	if view.Pack[1].UseItemKey == "" || view.Pack[1].UseItemKey == view.Pack[2].UseItemKey {
		t.Fatalf("equal wearable pack items have keys %q and %q, want two distinct double-click keys", view.Pack[1].UseItemKey, view.Pack[2].UseItemKey)
	}
	if view.Pack[3].UseItemKey != "" {
		t.Fatalf("an item with no equipment slot has double-click key %q", view.Pack[3].UseItemKey)
	}
}

// TestCityShopEmptyPackCellSelectsNothing: on the city item graph a click or
// a drag that starts on the money cell or on an empty pack cell selects no
// stack, as it already did without the graph. It changes nothing and shows no
// message; it never reaches the pack selection's stale-location refusal.
func TestCityShopEmptyPackCellSelectsNothing(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Carried[0].ID = "hero"
	item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
	cityShopSetTestPack(t, f, 0, sim.StackItem(item, 1))
	cityShopGraph(t, f)
	party, graph, shop := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), cloneShopMutation(f.Shop)
	money := ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 0}
	empty := ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 2}
	for _, gesture := range []struct {
		name string
		act  func() ui.TownAction
	}{
		{"money cell click", func() ui.TownAction { return screen.ShopClick(money) }},
		{"empty cell click", func() ui.TownAction { return screen.ShopClick(empty) }},
		{"money cell drag to the doll", func() ui.TownAction { return screen.ShopDrag(money, ui.ShopControl{Kind: ui.ShopControlDoll}) }},
		{"empty cell drag to the doll", func() ui.TownAction { return screen.ShopDrag(empty, ui.ShopControl{Kind: ui.ShopControlDoll}) }},
		{"empty cell drag to the table", func() ui.TownAction { return screen.ShopDrag(empty, ui.ShopControl{Kind: ui.ShopControlTableCell}) }},
	} {
		if a := gesture.act(); a.Msg != "" {
			t.Fatalf("%s: %q", gesture.name, a.Msg)
		}
		if !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || !reflect.DeepEqual(shop, cloneShopMutation(f.Shop)) {
			t.Fatalf("%s changed the party, the item graph or the table", gesture.name)
		}
	}
}
