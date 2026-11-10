package game

import (
	"testing"

	"againrom/pkg/sim"
)

// TestShopUseKeyIsStableForNativeRecordItems: the shop's use key for a pack
// item that carries a native record is the same on every read of the view,
// although each read clones the record. A key that changes between the press
// and the use makes the double-click use refuse.
func TestShopUseKeyIsStableForNativeRecordItems(t *testing.T) {
	f, screen := shopRoom(t, nil)
	f.Carried[0].ID = "hero"
	item := f.Shop.Shelf(ShelfArmour)[shopWearableShelfIndex(t, f, ShelfArmour)].Instance()
	item.NativeRecord = &sim.NativeItemRecord{Class: sim.SourceShield, F45: 1, F48: 7}
	stack := sim.StackItem(item, 1)
	a, b := screen.shopPackCell(stack), screen.shopPackCell(stack)
	if a.UseItemKey == "" || a.UseItemKey != b.UseItemKey {
		t.Fatalf("pack cell use key changes between builds:\n%q\n%q", a.UseItemKey, b.UseItemKey)
	}

	place := ShopPlace{ShopItem: shopItemFromInstance(item, 1), Mine: true}
	c, d := screen.shopPlaceCell(place), screen.shopPlaceCell(place)
	if c.UseItemKey == "" || c.UseItemKey != d.UseItemKey {
		t.Fatalf("table cell use key changes between builds:\n%q\n%q", c.UseItemKey, d.UseItemKey)
	}
}
