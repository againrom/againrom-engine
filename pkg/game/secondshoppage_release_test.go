package game

import (
	"fmt"
	"image"
	"slices"
	"testing"

	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// secondShopPagePoint is the window point of a page control.
func secondShopPagePoint(t *testing.T, app *ui.App, kind string, index int) image.Point {
	t.Helper()
	x, y, err := app.HeadlessShopPoint(kind, index)
	if err != nil {
		t.Fatalf("%s %d: %v", kind, index, err)
	}
	return image.Pt(x, y)
}

// secondShopDrag presses at from, moves to to and releases there.
func secondShopDrag(t *testing.T, app *ui.App, from, to image.Point) {
	t.Helper()
	for _, step := range []struct {
		action string
		at     image.Point
	}{{"press", from}, {"move", from.Add(image.Pt(4, 4))}, {"move", to}, {"release", to}} {
		if err := app.HeadlessPointer(step.action, step.at.X, step.at.Y); err != nil {
			t.Fatal(step.action, err)
		}
	}
}

// secondShopUnits is the purchase's units of the open deal.
func secondShopUnits(c *secondCampaign) int {
	var n int
	for _, it := range c.shop.buying() {
		n += int(it.Count)
	}
	return n
}

// TestReleaseSecondShopPageInput drives each of towns 1, 2 and 3's shop
// pages by pointer: the shelf lists the shown category in the deal's order
// with each item's name and price; a hover shows the item's lines; a click
// takes one unit and a shift click the rest; a drag from the shelf stages a
// unit and a drag from the hero's pack offers one; BUY and SELL settle at
// P per unit and (P+1)/2 per unit (R2-ENGINE-267).
func TestReleaseSecondShopPageInput(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
	}{{"Town1", 1}, {"Kaarg", 2}, {"Druid", 3}} {
		t.Run(tc.name, func(t *testing.T) {
			secondGameRoot(t)
			var f *FrontEnd
			var app *ui.App
			var s *townScreen
			if tc.id == 1 {
				f, app, s, _ = secondShopTownOne(t)
			} else {
				f, app, s, _ = secondShopLaterTown(t, tc.id)
			}
			secondShopEnter(t, f, app, s)
			c := f.Town.second
			f.Town.gold = 5000000

			// The shown category, in the deal's order.
			v := s.ShopScreen()
			shelf := c.shop.shelf()
			for i := 0; i < len(v.Shelf) && i < len(shelf); i++ {
				cell, it := v.Shelf[i], shelf[i]
				if int(cell.Count) != int(it.Count) || int64(cell.Price) != int64(it.Price) || !slices.Equal(cell.Text, s.shopItemName(it)) {
					t.Fatalf("shelf cell %d: count %d price %d text %q, want %+v named %q", i, cell.Count, cell.Price, cell.Text, it, s.shopItemName(it))
				}
			}
			r := v.Layout.ShelfCellRect(0)
			lines, ok := ui.ShopHoverLines(v, r.Min.Add(r.Max).Div(2))
			if !ok || len(lines) == 0 {
				t.Fatal("no tooltip over the first shelf cell")
			}
			if _, ok := ui.ShopHoverLines(v, image.Pt(630, 470)); ok {
				t.Fatal("a tooltip outside every cell")
			}
			t.Logf("tooltip of %q: %q", s.shopItemName(shelf[0]), lines)

			// The quantity choice: one unit, then the rest of the entry.
			k, entry := -1, -1
			for cat := 0; cat < 4 && entry < 0; cat++ {
				if !s.shelfOpen(cat) {
					continue
				}
				secondShopActivate(t, f, app, []string{"SHELF 1", "SHELF 2", "SHELF 3", "SHELF 4"}[cat])
				for i, it := range c.shop.shelf() {
					if it.Count >= 2 && i < len(v.Shelf) {
						k, entry = cat, i
						break
					}
				}
			}
			if entry < 0 {
				t.Fatal("no shown entry holds two units")
			}
			want := c.shop.shelf()[entry]
			at := secondShopPagePoint(t, app, "shelf", entry)
			shopTap(t, app, at, "press", "release")
			if secondShopUnits(c) != 1 || c.shop.shelf()[entry].Count != want.Count-1 {
				t.Fatalf("a click took %d units", secondShopUnits(c))
			}
			shopTap(t, app, at, "shift-press", "shift-release")
			if secondShopUnits(c) != int(want.Count) || c.shop.shelf()[entry].Count != 0 {
				t.Fatalf("a shift click left %d of %d units staged", secondShopUnits(c), want.Count)
			}
			secondShopActivate(t, f, app, "UNDO")
			if secondShopUnits(c) != 0 || c.shop.shelf()[entry].Count != want.Count {
				t.Fatal("UNDO did not return the units")
			}

			// Dragging: shelf to table stages a unit, BUY pays P for it; the
			// hero's pack to the table offers it, SELL credits (P+1)/2.
			// Since the bag goes into the mission and comes back at the next
			// town, a later town's bag already holds the hero's start unit:
			// the purchase is the stack after those held, and the drag sells it.
			gold, held := f.Town.gold, len(c.heroItems())
			secondShopDrag(t, app, at, secondShopPagePoint(t, app, "table", 0))
			if secondShopUnits(c) != 1 {
				t.Fatalf("a drag from the shelf staged %d units", secondShopUnits(c))
			}
			secondShopActivate(t, f, app, "BUY")
			if f.Town.gold != gold-int(want.Price) || len(c.heroItems()) != held+1 {
				t.Fatalf("buy at %d beside %d held: gold %d -> %d, hero %v", want.Price, held, gold, f.Town.gold, c.heroItems())
			}
			gold = f.Town.gold
			secondShopDrag(t, app, secondShopPagePoint(t, app, "pack", f.TownScreen().(*townScreen).packCell(held)), secondShopPagePoint(t, app, "table", 0))
			if len(c.shop.selling()) != 1 || c.shop.selling()[0].Price != want.Price {
				t.Fatalf("a drag from the pack offered %v", c.shop.selling())
			}
			secondShopActivate(t, f, app, "SELL")
			if f.Town.gold != gold+int(secondshop.SaleCredit(int64(want.Price), 1)) || len(c.heroItems()) != held {
				t.Fatalf("sale at %d: gold %d -> %d", want.Price, gold, f.Town.gold)
			}
			secondShopWitness(t, app, map[int]string{1: "town1", 2: "town2", 3: "town3"}[tc.id]+"-shop-input")
			t.Logf("town %d: category %d entry %d of %d units; bought at %d, sold for %d", tc.id, k, entry,
				want.Count, want.Price, secondshop.SaleCredit(int64(want.Price), 1))
		})
	}
}

// TestReleaseSecondShopItemsWithoutBagForm counts the items the shipped
// assortment records admit that have no bag form, which no shelf lists: every
// category of the new game's four records and the druid record's fourth
// category with class 4 admitted (R2-ENGINE-331).
func TestReleaseSecondShopItemsWithoutBagForm(t *testing.T) {
	secondGameRoot(t)
	f := secondGameFront(t)
	tables, ok := secondShopTables(f.Table)
	if !ok {
		t.Fatal("no shop tables")
	}
	records := secondshop.NewGameRecords
	druid := records[3][3]
	druid.Mask |= secondShopClass4
	var total, missing int
	seen := map[string]bool{}
	count := func(c secondshop.Category) {
		for _, rnd := range []secondshop.Rand{nil, func() int { return 1 << 30 }} {
			for _, it := range tables.Candidates(c, rnd) {
				if seen[fmt.Sprintf("%+v", it)] {
					continue
				}
				seen[fmt.Sprintf("%+v", it)] = true
				total++
				if _, ok := tables.Instance(it); !ok {
					missing++
					t.Logf("no bag form: %+v", it)
				}
			}
		}
	}
	for _, r := range records {
		for _, c := range r {
			count(c)
		}
	}
	count(druid)
	if total == 0 {
		t.Fatal("the records admit no item")
	}
	t.Logf("assortment candidates: %d distinct items, %d without a bag form", total, missing)
}

// TestReleaseSecondTownShopFullBagRefusesPurchase fills town 1's hero bag to
// the bound the SAVE producer writes and buys one shelf item by pointer: the
// keeper refuses before any gold moves, the item stays on the table, and
// the campaign still exports a SAV.
func TestReleaseSecondTownShopFullBagRefusesPurchase(t *testing.T) {
	secondGameRoot(t)
	f, app, s, _ := secondShopTownOne(t)
	secondShopEnter(t, f, app, s)
	c := f.Town.second
	tables, ok := secondShopTables(f.Table)
	if !ok {
		t.Fatal("the install's shop tables")
	}
	units, ok := tables.Units(secondshop.Item{Table: secondshop.Magic, Row: 70, Count: maxSecondBagUnits, Price: 83})
	if !ok {
		t.Fatal("potion row 70 has no bag form")
	}
	*c.heroBag() = units
	if err := captureSecondCampaign(c).validate(); err != nil {
		t.Fatal("a full bag does not save:", err)
	}
	f.Town.gold = 5000
	secondShopActivate(t, f, app, "ITEM 1", "BUY")
	if f.Town.gold != 5000 || len(*c.heroBag()) != maxSecondBagUnits || secondShopUnits(c) != 1 {
		t.Fatalf("purchase past the bound: gold %d bag %d pending %d", f.Town.gold, len(*c.heroBag()), secondShopUnits(c))
	}
	snap, label, err := f.Snapshot(false)
	if err == nil {
		_, err = f.ExportCurrentSave(snap, label)
	}
	if err != nil {
		t.Fatalf("SAVE after the refused purchase: %v", err)
	}
}

// TestReleaseSecondTownShopBookPartialBuy clicks one shelf book of town 1
// twice by pointer and buys with gold for one: one book enters the bag, the
// gold is spent and the second book stays pending, each in its own table
// place (R2-ENGINE-265, R2-ENGINE-266, DIV-3065).
func TestReleaseSecondTownShopBookPartialBuy(t *testing.T) {
	secondGameRoot(t)
	f, app, s, _ := secondShopTownOne(t)
	secondShopEnter(t, f, app, s)
	c := f.Town.second
	*c.heroBag() = nil
	category, entry := -1, -1
	var book secondshop.Item
	for k, list := range c.shop.stock() {
		for i, it := range list {
			if it.Table == secondshop.Book && it.Count >= 2 && it.Price > 0 && entry < 0 {
				category, entry, book = k, i, it
			}
		}
	}
	if entry < 0 {
		t.Fatal("town 1's shelves hold no book of two units")
	}
	secondShopActivate(t, f, app, []string{"SHELF 1", "SHELF 2", "SHELF 3", "SHELF 4"}[category])
	s.ShopScroll(ui.ShopWheelShelf, entry/2)
	at := secondShopPagePoint(t, app, "shelf", entry-s.shelfBase)
	f.Town.gold = int(book.Price)
	shopTap(t, app, at, "press", "release")
	shopTap(t, app, at, "press", "release")
	if len(c.shop.Table()) != 2 {
		t.Fatalf("two units of book spell %d staged as %d places", book.Spell, len(c.shop.Table()))
	}
	secondShopActivate(t, f, app, "BUY")
	if f.Town.gold != 0 || len(*c.heroBag()) != 1 || secondShopUnits(c) != 1 {
		t.Fatalf("book spell %d at %d: gold %d bag %d pending %d", book.Spell, book.Price, f.Town.gold, len(*c.heroBag()), secondShopUnits(c))
	}
}

// TestReleaseSecondTownShopTableScrolls clicks town 1's six visible shelf
// cells by pointer: the table holds six places, the wheel over the table
// strip shows each of them, the shown places cost what BUY asks, the sixth
// shows its lines under the pointer and returns to its shelf by its own
// cell; BUY then settles the five left.
func TestReleaseSecondTownShopTableScrolls(t *testing.T) {
	secondGameRoot(t)
	f, app, s, _ := secondShopTownOne(t)
	secondShopEnter(t, f, app, s)
	c := f.Town.second
	f.Town.gold = 100000
	for i := 0; i < 6; i++ {
		shopTap(t, app, secondShopPagePoint(t, app, "shelf", i), "press", "release")
	}
	table := c.shop.Table()
	if len(table) != 6 {
		t.Fatalf("six shelf cells staged %d places", len(table))
	}
	secondShopWitness(t, app, "table-last-five")
	strip := secondShopPagePoint(t, app, "table", 2)
	shown, cost := map[int]bool{}, int32(0)
	var v ui.ShopScreenView
	for _, wheel := range []string{"wheel-up", "wheel-up", "wheel-down"} {
		if err := app.HeadlessPointer(wheel, strip.X, strip.Y); err != nil {
			t.Fatal(wheel, err)
		}
		v = s.ShopScreen()
		for i, cell := range v.Table {
			k := v.TableOffset + i
			if !cell.Occupied() || k >= len(table) || cell.Price != table[k].ShopItem.Price || !slices.Equal(cell.Text, s.label(table[k].ShopItem.Instance())) {
				t.Fatalf("strip from %d: cell %d shows %q at %d, place %d", v.TableOffset, i, cell.Text, cell.Price, k)
			}
			if !shown[k] {
				shown[k], cost = true, cost+cell.Price*int32(cell.Count)
			}
		}
	}
	if len(shown) != 6 || cost != v.Buy || v.TableOffset != 1 {
		t.Fatalf("shown places %v cost %d, BUY %d, strip from %d", shown, cost, v.Buy, v.TableOffset)
	}
	last := secondShopPagePoint(t, app, "table", 4)
	if lines, ok := ui.ShopHoverLines(v, last); !ok || len(lines) == 0 {
		t.Fatalf("the sixth place shows no lines: %q", lines)
	}
	shopTap(t, app, last, "press", "release")
	if got := c.shop.Table(); len(got) != 5 || !shopItemEqual(got[4].ShopItem, table[4].ShopItem) {
		t.Fatalf("the sixth place did not leave the table: %d places", len(got))
	}
	bag, gold, buy := len(*c.heroBag()), f.Town.gold, int(s.ShopScreen().Buy)
	secondShopActivate(t, f, app, "BUY")
	if len(c.shop.Table()) != 0 || f.Town.gold != gold-buy || len(*c.heroBag()) != bag+secondShopPlacesUnits(table[:5]) {
		t.Fatalf("BUY: table %d gold %d (from %d, BUY %d) bag %d (from %d)", len(c.shop.Table()), f.Town.gold, gold, buy, len(*c.heroBag()), bag)
	}
}

// secondShopPlacesUnits is the units of places.
func secondShopPlacesUnits(places []ShopPlace) int {
	n := 0
	for _, p := range places {
		n += int(p.ShopItem.Count)
	}
	return n
}
