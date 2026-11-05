package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/ui"
)

func TestShopBuyRefusesWholeBasket(t *testing.T) {
	for _, row := range []struct {
		name string
		shop *Shop
	}{
		{"chapter", NewShop(1000)}, {"bank", func() *Shop { s := newSecondShop(); s.filled = 1; return &s }()},
	} {
		t.Run(row.name, func(t *testing.T) {
			s := row.shop
			s.shelves[0] = []ShopItem{magicUnit(70, 1, 20), magicUnit(71, 1, 40)}
			if !s.TakeFromShelf(0, 0, 1) {
				t.Fatal("first place")
			}
			index := 0
			if s.profile().keepEmptied {
				index = 1
			}
			if !s.TakeFromShelf(0, index, 1) {
				t.Fatal("second place")
			}
			before := cloneShopMutation(s)
			items, spent, ok := s.Buy(30)
			if ok || spent != 0 || len(items) != 0 || !reflect.DeepEqual(before, cloneShopMutation(s)) {
				t.Fatalf("short purse changed the basket: items %v spent %d accepted %v", items, spent, ok)
			}
		})
	}
	c := secondShopSample()
	c.shop.shelves[0] = []ShopItem{magicUnit(70, 1, 20), magicUnit(71, 1, 40)}
	s := secondShopPage(c, 30)
	s.shopShelf = 0
	s.shopFromShelf(0, false)
	s.shopFromShelf(1, false)
	if c.shop.BuyTotal() != 60 {
		t.Fatal("fixture did not stage the two lots", c.shop.Table())
	}
	before, bag := cloneShopMutation(&c.shop), slices.Clone(*c.heroBag())
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 1})
	if s.sess.Town.gold != 30 || !reflect.DeepEqual(before, cloneShopMutation(&c.shop)) || !reflect.DeepEqual(bag, *c.heroBag()) {
		t.Fatal("live refusal changed gold, shelves, bag or pending places")
	}
}
