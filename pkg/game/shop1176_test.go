package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/data"
)

func TestShop1176UnpricedDocumentsSurviveMixedAndSingleSale(t *testing.T) {
	for _, price := range []int32{-1, 0} {
		t.Run(fmt.Sprint(price), func(t *testing.T) {
			s := NewShop(1000)
			document := ShopItem{Code: data.QuestDocumentCode, Price: price, Count: 1}
			if !s.PutOnTable(document) {
				t.Fatal("stage documents")
			}
			before := s.Table()
			if paid, ok := s.Sell(); ok || paid != 0 || !reflect.DeepEqual(before, s.Table()) {
				t.Fatal("documents alone changed the tray or produced payment", paid, ok, s.Table())
			}
			priced := ShopItem{Code: data.ComposeItemCode(0, 1, 0, 3), Price: 5, Count: 3}
			if !s.PutOnTable(priced) || s.SellTotal() != 9 || s.SellPayout() != 8 {
				t.Fatal("ordinary odd-price stack accounting changed", s.SellTotal(), s.SellPayout())
			}
			if paid, ok := s.Sell(); !ok || paid != 8 {
				t.Fatal("mixed sale changed ordinary stack payout", paid, ok)
			}
			if !reflect.DeepEqual(before, s.Table()) || len(s.Shelf(ShelfMagic)) != 0 {
				t.Fatal("mixed sale took or changed physical documents", s.Table(), s.Shelf(ShelfMagic))
			}
			if shelf := s.Shelf(ShelfWeapons); len(shelf) != 1 || shelf[0].Count != 3 || shelf[0].Price != 5 {
				t.Fatal("ordinary sale did not return its complete stack", shelf)
			}
			if paid, ok := s.Sell(); ok || paid != 0 || !reflect.DeepEqual(before, s.Table()) {
				t.Fatal("repeated sale consumed the retained documents", paid, ok)
			}
		})
	}
}
