package game

import (
	"testing"

	"againrom/pkg/secondshop"
)

// TestShopDealOneCoreBothGames drives both games' shops through the one
// deal core: the same calls, with each game's rules as its data, give each
// game's buying and selling.
func TestShopDealOneCoreBothGames(t *testing.T) {
	first := NewShop(1000)
	for i := range ShopTablePlaces + 1 {
		first.shelves[ShelfWeapons] = append(first.shelves[ShelfWeapons], ShopItem{Code: 0x0101, Price: int32(10 + i), Count: 1})
	}
	for range ShopTablePlaces {
		if !first.TakeFromShelf(ShelfWeapons, 0, 1) {
			t.Fatal("first game: a place under the bound refused")
		}
	}
	if first.TakeFromShelf(ShelfWeapons, 0, 1) {
		t.Fatal("first game: a sixth place opened (places 5)")
	}
	if _, _, ok := first.Buy(first.BuyTotal() - 1); ok {
		t.Fatal("first game: a purse short of the whole basket bought (whole-basket guard)")
	}

	second := &secondShop{records: secondshop.NewGameRecords, filled: 1}
	second.shelves[0] = secondShopItems(secondshop.Tables{}, []secondshop.Item{
		{Kind: secondshop.Magic, Row: 70, Quantity: 1, Price: 40},
		{Kind: secondshop.Magic, Row: 71, Quantity: 1, Price: 30},
		{Kind: secondshop.Magic, Row: 72, Quantity: 1, Price: 20},
	})
	r := second.rules()
	for i := range 3 {
		if !second.takeFromShelf(r, 0, i, 1) {
			t.Fatalf("second game: take %d refused", i)
		}
	}
	if len(second.shelves[0]) != 3 || second.shelves[0][0].Count != 0 {
		t.Fatalf("second game: emptied entries %v, want placeholders kept", second.shelves[0])
	}
	bought, spent, ok := second.shopDeal.buy(r, 75)
	if !ok || len(bought) != 2 || spent != 70 || len(second.buying()) != 1 || second.buying()[0].Row != 72 || len(second.shelves[0]) != 3 {
		t.Fatalf("second game: buy at 75 bought %v spent %d left %v, want the first two and the third kept", bought, spent, second.buying())
	}
	// Loss control: the first game's rules on the same deal refuse it whole.
	whole := r
	whole.wholeBasket = true
	second.table = append(second.table, ShopPlace{ShopItem: magicUnit(73, 1, 50)})
	if _, _, ok := second.shopDeal.buy(whole, 60); ok {
		t.Fatal("whole-basket guard bought a basket the purse cannot cover")
	}
	second.table = nil
	if !second.putOnTable(r, magicUnit(74, 2, 0)) {
		t.Fatal("second game: offer refused")
	}
	paid, back, ok := second.shopDeal.sell(r)
	if !ok || paid != 0 || len(back) != 1 || len(second.table) != 0 {
		t.Fatalf("second game: unpriced sale paid %d back %v table %v, want it returned", paid, back, second.table)
	}
	second.dropEmptied(r)
	if len(second.shelves[0]) != 0 {
		t.Fatalf("second game: leave kept %v, want the emptied entries dropped", second.shelves[0])
	}
}

// magicUnit is n units of the second game's magic item row at price.
func magicUnit(row, n int, price int64) ShopItem {
	i, _ := secondShopItem(secondshop.Tables{}, secondshop.Item{Kind: secondshop.Magic, Row: row, Quantity: n, Price: price})
	return i
}
