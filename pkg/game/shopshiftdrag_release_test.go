package game

import (
	"fmt"
	"image"
	"strings"
	"testing"
)

// An installed stack of eleven identical items is bought whole and sold whole
// with Shift drags through App's ordinary shop input (DIV-1463). The armour
// shelf is the one the room opens on, so no shelf pick is pressed.
func TestReleaseShopShiftDragTradesAnInstalledStack(t *testing.T) {
	app, s := releaseShopApp(t)
	f := frontOf(s)
	item, ok := releaseShiftDragItem(s)
	if !ok {
		t.Fatal("the generated armour shelf holds no priced, unenchanted item the pack lacks")
	}
	item.Count = 11
	selector := 0
	if font := f.Font.Value(); font != nil {
		selector = font.Selector
	}
	t.Logf("item %#04x %q, unit price %d, 11 units", uint16(item.Code),
		asciiLabel([]byte(itemName(item.Code, f.Table)), selector), item.Price)
	f.Shop.shelves[ShelfArmour] = []ShopItem{item}
	f.Town.gold = 100000
	if s.shopShelf != ShelfArmour {
		t.Fatalf("the room opened on shelf %v, want the armour shelf", s.shopShelf)
	}
	at := shopPoints(t, app)
	state := func(stage string) (gold int, pack, table, shelf int32) {
		gold = f.Town.Gold()
		pack, table, shelf = releaseShiftDragCounts(s, item)
		t.Logf("%s: gold %d; pack %s; armour shelf %s; table %s", stage, gold,
			releaseShiftDragStacks(s), releaseShiftDragShelf(f), releaseShiftDragTable(f))
		return gold, pack, table, shelf
	}

	gold0, pack, table, shelf := state("before")
	if pack != 0 || table != 0 || shelf != 11 {
		t.Fatalf("before: pack %d, table %d, shelf %d; want 0, 0, 11", pack, table, shelf)
	}
	shopDrag(t, app, at["shelf"], at["table"], "shift-press", "shift-move", "shift-release")
	if _, pack, table, shelf = state("Shift drag shelf to table"); pack != 0 || table != 11 || shelf != 0 {
		t.Fatalf("pack %d, table %d, shelf %d; want 0, 11, 0", pack, table, shelf)
	}
	shopTap(t, app, at["buy"], "press", "release")
	gold1, pack, table, shelf := state("Buy")
	if pack != 11 || table != 0 || shelf != 0 || gold0-gold1 != int(11*item.Price) {
		t.Fatalf("pack %d, table %d, shelf %d, spent %d; want 11, 0, 0, %d", pack, table, shelf, gold0-gold1, 11*item.Price)
	}

	cell, err := f.headlessShopCell("pack", uint16(item.Code))
	if err != nil {
		t.Fatal(err)
	}
	x, y, err := app.HeadlessShopPoint("pack", cell)
	if err != nil {
		t.Fatal(err)
	}
	shopDrag(t, app, image.Pt(x, y), at["table"], "shift-press", "shift-move", "shift-release")
	if _, pack, table, shelf = state("Shift drag pack to table"); pack != 0 || table != 11 || shelf != 0 {
		t.Fatalf("pack %d, table %d, shelf %d; want 0, 11, 0", pack, table, shelf)
	}
	shopTap(t, app, at["sell"], "press", "release")
	gold2, pack, table, shelf := state("Sell")
	if want := (11*item.Price + 1) / 2; pack != 0 || table != 0 || shelf != 11 || gold2-gold1 != int(want) {
		t.Fatalf("pack %d, table %d, shelf %d, paid %d; want 0, 0, 11, ceil(11*%d/2) = %d",
			pack, table, shelf, gold2-gold1, item.Price, want)
	}
}

// releaseShiftDragItem is the cheapest priced, unenchanted item the town's own
// generator put on the armour shelf whose code the shown pack does not hold.
// An odd price is preferred: its whole-stack payout differs from eleven unit
// payouts (SHOP-SELL-010).
func releaseShiftDragItem(s *townScreen) (ShopItem, bool) {
	held := map[uint16]bool{}
	for _, st := range s.shopPackStacks() {
		held[st.Code] = true
	}
	var best ShopItem
	found := false
	for _, item := range s.sess.Shop.Shelf(ShelfArmour) {
		if item.Price <= 0 || len(item.Effects) != 0 || held[uint16(item.Code)] {
			continue
		}
		odd, bestOdd := item.Price%2 == 1, best.Price%2 == 1
		if !found || odd && !bestOdd || odd == bestOdd && item.Price < best.Price {
			best, found = item.Clone(), true
		}
	}
	return best, found
}

func releaseShiftDragCounts(s *townScreen, item ShopItem) (pack, table, shelf int32) {
	for _, st := range s.shopPackStacks() {
		if st.Code == uint16(item.Code) {
			pack += int32(st.Count)
		}
	}
	for _, place := range s.sess.Shop.Table() {
		if place.Code == item.Code {
			table += place.Count
		}
	}
	for _, it := range s.sess.Shop.Shelf(ShelfArmour) {
		if it.Code == item.Code {
			shelf += it.Count
		}
	}
	return pack, table, shelf
}

func releaseShiftDragStacks(s *townScreen) string {
	var out []string
	for _, st := range s.shopPackStacks() {
		out = append(out, fmt.Sprintf("%#04x x%d", st.Code, st.Count))
	}
	return "[" + strings.Join(out, " ") + "]"
}

func releaseShiftDragShelf(f *FrontEnd) string {
	var out []string
	for _, it := range f.Shop.Shelf(ShelfArmour) {
		out = append(out, fmt.Sprintf("%#04x x%d @%d", uint16(it.Code), it.Count, it.Price))
	}
	return "[" + strings.Join(out, " ") + "]"
}

func releaseShiftDragTable(f *FrontEnd) string {
	var out []string
	for _, place := range f.Shop.Table() {
		out = append(out, fmt.Sprintf("%#04x x%d @%d mine=%v", uint16(place.Code), place.Count, place.Price, place.Mine))
	}
	return "[" + strings.Join(out, " ") + "]"
}
