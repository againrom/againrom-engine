package game

import (
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// secondShopProof is what the continuous run hands the cold process: the
// shop state the SAV holds, the gold, and the outcome of the next deal.
type secondShopProof struct {
	Shop       *currentSecondShop
	Gold       int
	Next       *currentSecondShop
	NextGold   int
	NextHeader string
}

func secondShopActivate(t *testing.T, app *ui.App, rows ...string) {
	t.Helper()
	for _, row := range rows {
		if err := app.HeadlessActivate(row); err != nil {
			t.Fatalf("shop row %q: %v", row, err)
		}
	}
}

// secondShopNextDeal is the deal both runs make after the SAV: the open
// purchase is undone, one unit of the first entry is bought, and the second
// category's first entry is taken and bought.
func secondShopNextDeal(t *testing.T, f *FrontEnd, app *ui.App) (*currentSecondShop, int) {
	t.Helper()
	secondShopActivate(t, app, "UNDO", "ITEM 1", "BUY", "SHELF 2", "ITEM 1", "BUY")
	return captureSecondShop(f.Town.second.shop), f.Town.gold
}

// secondShopWitness writes the page's headless frame for the owner.
func secondShopWitness(t *testing.T, app *ui.App, name string) {
	t.Helper()
	dir := os.Getenv("AGAINROM_SHOP_WITNESS")
	if dir == "" {
		return
	}
	picture, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal("shop frame", err, note)
	}
	out, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := png.Encode(out, picture); err != nil {
		t.Fatal(err)
	}
}

// TestReleaseSecondTownShopDealAndSave opens the town 1 shop from the
// square, buys and sells through the page's rows at the claimed prices,
// saves with a deal open, and has a separate process LOAD the SAV, compare
// the shop and the gold and make the same next deal.
// secondShopNamesResolve checks that every item town 1's record admits,
// books aside, has a line in the shipped item-name table under its NameKey;
// the loss control swaps the part and class fields and must lose names.
func secondShopNamesResolve(t *testing.T, f *FrontEnd) {
	t.Helper()
	tables, ok := secondShopTables(f.Table)
	if !ok || len(f.Table.Names) == 0 {
		t.Fatal("no shop tables or item-name table")
	}
	var named, swapped, total int
	for _, c := range secondshop.NewGameRecords[1] {
		for _, it := range tables.Candidates(c, nil) {
			key, ok := tables.NameKey(it)
			if !ok {
				continue
			}
			total++
			if _, ok := f.Table.Names.NameFor(data.ItemCode(key)); ok {
				named++
			}
			if it.Kind != secondshop.Magic {
				part := uint16(tables.Part(it))
				alt := key&^0x0fe0 | uint16(it.Class&0xf)<<8 | (part&0x7)<<5
				if _, ok := f.Table.Names.NameFor(data.ItemCode(alt)); ok && alt != key {
					swapped++
				}
			}
		}
	}
	if total == 0 || named != total || swapped >= total/2 {
		t.Fatalf("item names: %d of %d named, %d named with part and class swapped", named, total, swapped)
	}
	t.Logf("item names: %d of %d named; swapped keys name %d", named, total, swapped)
}

func TestReleaseSecondTownShopDealAndSave(t *testing.T) {
	secondGameRoot(t)
	if path := os.Getenv("AGAINROM_SECOND_SHOP_INPUT"); path != "" {
		raw, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var proof secondShopProof
		if err := json.Unmarshal(raw, &proof); err != nil {
			t.Fatal(err)
		}
		f, app := secondTownCold(t, filepath.Dir(path), filepath.Base(path))
		if f.Town.second.room != secondTownShop || f.Town.gold != proof.Gold ||
			!reflect.DeepEqual(captureSecondShop(f.Town.second.shop), proof.Shop) {
			t.Fatalf("cold LOAD: room %d gold %d/%d shop equal %v", f.Town.second.room, f.Town.gold, proof.Gold,
				reflect.DeepEqual(captureSecondShop(f.Town.second.shop), proof.Shop))
		}
		next, gold := secondShopNextDeal(t, f, app)
		if gold != proof.NextGold || !reflect.DeepEqual(next, proof.Next) || f.TownScreen().Header() != proof.NextHeader {
			t.Fatalf("cold next deal: gold %d/%d shop equal %v", gold, proof.NextGold, reflect.DeepEqual(next, proof.Next))
		}
		t.Log("fresh-process LOAD: shop, gold and the next deal equal the continuous run")
		return
	}
	f, app := secondTownNew(t, false, false)
	secondShopActivate(t, app, "SHOP")
	c := f.Town.second
	if c.room != secondTownShop || f.TownScreen().Header() != "ROM2 campaign: shop" || len(c.shop.stock[0]) == 0 || c.shop.filled != 1 {
		t.Fatalf("SHOP opened room %d, filled %d, %d entries", c.room, c.shop.filled, len(c.shop.stock[0]))
	}
	if s, ok := f.TownScreen().(*secondCampaignScreen); !ok || !s.AtTownSquare() {
		d, _ := s.composedTown()
		t.Fatal("the shop page's art did not load:", s.roomArt(d, "shop").err)
	}
	secondShopWitness(t, app, "town1-shop-open")
	secondShopNamesResolve(t, f)
	f.Town.gold = 5000
	first := c.shop.shelf()[c.shop.visibleShelf()[0]]
	secondShopActivate(t, app, "ITEM 1", "BUY")
	if f.Town.gold != 5000-int(first.Price) || len(c.shop.hero) != 1 || c.shop.hero[0].Quantity != 1 {
		t.Fatalf("buy of one at %d: gold %d hero %v", first.Price, f.Town.gold, c.shop.hero)
	}
	secondShopWitness(t, app, "town1-shop-bought")
	gold := f.Town.gold
	secondShopActivate(t, app, "MINE 1", "SELL")
	if f.Town.gold != gold+int(secondshop.SaleCredit(first.Price, 1)) || len(c.shop.hero) != 0 {
		t.Fatalf("sale of one at %d: gold %d -> %d", first.Price, gold, f.Town.gold)
	}
	secondShopActivate(t, app, "SHELF 3", "ITEM 1", "BUY", "SHELF 1", "ITEM 2")
	if len(c.shop.hero) != 1 || len(c.shop.buying) != 1 {
		t.Fatalf("second deal: hero %v buying %v", c.shop.hero, c.shop.buying)
	}
	secondShopWitness(t, app, "town1-shop-deal-open")
	saved, savedGold := captureSecondShop(c.shop), f.Town.gold
	dir := secondMissionSaveDirectory(t)
	raw := secondTownNamedSave(t, f, app, dir, "Shop")
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if got := actions.Session.Second.Shop; !reflect.DeepEqual(got, saved) || len(c.shop.buying) != 1 {
		t.Fatal("the SAV's shop differs from the captured shop, or SAVE closed the deal")
	}
	// Loss control: an emptied first category in the written shop is not
	// the shop the continuous run holds.
	lost := saved.clone()
	lost.Stock[0] = nil
	if reflect.DeepEqual(lost, saved) {
		t.Fatal("loss control compares equal")
	}
	next, nextGold := secondShopNextDeal(t, f, app)
	if nextGold >= savedGold {
		t.Fatalf("next deal bought nothing: gold %d -> %d", savedGold, nextGold)
	}
	proof, err := json.Marshal(secondShopProof{Shop: saved, Gold: savedGold, Next: next, NextGold: nextGold, NextHeader: f.TownScreen().Header()})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "Shop.sav")
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	runSpellWitnessChild(t, path, "AGAINROM_SECOND_SHOP_INPUT")
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare || len(c.shop.buying) != 0 {
		t.Fatalf("escape left room %d with %d pending", c.room, len(c.shop.buying))
	}
	t.Logf("town 1 shop: %d/%d/%d/%d entries; bought at %d, sold for %d; SAV %d bytes", len(saved.Stock[0]), len(saved.Stock[1]),
		len(saved.Stock[2]), len(saved.Stock[3]), first.Price, secondshop.SaleCredit(first.Price, 1), len(raw))
}
