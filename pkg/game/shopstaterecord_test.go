package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/random"
)

func shopStateRecord(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	line := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	for n := 0; n <= 3; n++ {
		recordFirstSleeps(t, line, n, true)
		for _, id := range []int{1, 2, 3} {
			recordSecondSleeps(t, line, id, n, true)
		}
	}
	for _, s := range modStockRecord(t) {
		line("mod %s", s)
	}
	record := func(key string, f *FrontEnd, screen *townScreen) {
		s := screen.deal()
		var shelves [][]ShopItem
		for k := ShopShelf(0); k < shopShelfCount; k++ {
			shelves = append(shelves, s.Shelf(k))
		}
		raw, err := json.Marshal([]any{shelves, s.Table(), screen.shopPackItemInstances(), f.Carried, f.Town.Gold()})
		if err != nil {
			t.Fatal(err)
		}
		line("%s %s", key, raw)
		for i, p := range s.Table() {
			line("%s place %d city %d owners %+v", key, i, p.ShopItem.CityID, p.ShopItem.Owners)
		}
		line("%s sav %s next %d", key, sleepSAV(f), f.randomService().Stream(random.ShopStock).Raw())
	}
	f := shopSaveFixture(t, 3, 2)
	s := f.TownScreen().(*townScreen)
	s.room = roomShop
	record("first arrival", f, s)
	s.shopFromPack(0, false)
	record("first offer", f, s)
	s.shopSell()
	record("first sell", f, s)
	merchant := shopItemFromInstance(s.shopPackItemInstances()[0], 2)
	merchant.ObjectID, merchant.CityID, merchant.Owners, merchant.Price = 0, 0, nil, 11
	f.Shop.shelves[ShelfMagic] = []ShopItem{merchant}
	s.shopShelf = ShelfMagic
	s.shopFromShelf(0, false)
	if len(f.Shop.table) != 1 {
		t.Fatal("first take did not open a place")
	}
	record("first take", f, s)
	s.shopOffTable(0, false)
	record("first one-place return", f, s)
	s.shopFromShelf(0, false)
	gold := f.Town.Gold()
	s.shopBuy()
	if f.Town.Gold() != gold-11 {
		t.Fatal("first buy did not settle")
	}
	record("first buy", f, s)
	s.shopFromPack(0, false)
	record("first pending SAVE", f, s)
	s.shopClear()
	record("first undo", f, s)
	s.shopFromPack(0, false)
	s.Back()
	record("first exit", f, s)
	unpriced := NewShop(100)
	merchant.Price = 0
	merchant.Owners = []shopOwnerQuantity{{PartyID: "hero", Count: merchant.Count}}
	unpriced.PutOnTable(merchant)
	paid, back, ok := unpriced.SellBack()
	raw, _ := json.Marshal([]any{paid, back, ok, unpriced.Table()})
	line("first unpriced %s", raw)
	for _, id := range []int{1, 2, 3} {
		f, s := sleepTown(id)
		c := f.Town.second
		tables, _ := secondShopTables(f.Table)
		c.shop.enter(tables, id, f.randomService().Stream(random.ShopStock).Raw, c.heroBag())
		s.bindShop(s.bankShopRow())
		s.shopShelf = ShelfBooks
		key := fmt.Sprintf("second %d", id)
		record(key+" entry", f, s)
		s.shopFromShelf(0, false)
		if len(c.shop.table) != 1 {
			t.Fatal("second take did not open a place")
		}
		record(key+" take", f, s)
		s.shopOffTable(0, false)
		record(key+" one-place return", f, s)
		s.shopFromShelf(0, false)
		s.shopBuy()
		record(key+" buy", f, s)
		s.shopFromPack(0, false)
		record(key+" offer SAVE", f, s)
		s.shopSell()
		record(key+" sell", f, s)
		unit := c.shop.shelves[3][0].Instance()
		unit.Price = 0
		*c.heroBag() = append(*c.heroBag(), unit)
		s.shopFromPack(0, true)
		s.shopSell()
		record(key+" unpriced", f, s)
		s.shopFromShelf(0, false)
		s.shopClear()
		record(key+" undo", f, s)
		c.shop.leave(c.heroBag())
		record(key+" exit", f, s)
		c.shop.depart(50)
		c.shop.stage(90)
		c.shop.enter(tables, 3, f.randomService().Stream(random.ShopStock).Raw, c.heroBag())
		record(key+" departure stage", f, s)
	}
	for _, ceiling := range []int32{25, 1000} {
		s := NewShop(ceiling)
		s.Generate(shopTable(), shopSeed(int(ceiling), 2, ceiling))
		if os.Getenv("AGAINROM_SHOP_RECORD_PLANT") == "1" {
			slices.Reverse(s.shelves[0])
		}
		raw, _ := json.Marshal(s.shelves)
		line("chapter %d seed %d stock %s", ceiling, s.drawSeed, raw)
	}
	return b.String()
}

func TestShopStateRecord(t *testing.T) {
	got := shopStateRecord(t)
	if strings.Contains(got, "refused:") {
		t.Fatal("the record must contain SAV bytes, not a refusal")
	}
	// R2-ENGINE-941: a candidate precedes the potion lead.
	// a72c6ee21: SAVE returns table items before later trades.
	path := filepath.Join("testdata", "shop_state_record.txt")
	if os.Getenv("AGAINROM_WRITE_SHOP_STATE_RECORD") == "1" {
		if err := os.WriteFile(path, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Fatalf("shop record changed: got %x, want %x", sha256.Sum256([]byte(got)), sha256.Sum256(want))
	}
}
