package game

import "testing"

func TestShopEntryOpensFirstStockedShelf(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stock []ShopShelf
		want  int
	}{
		{"armour first", []ShopShelf{ShelfArmour, ShelfWeapons, ShelfMagic, ShelfBooks}, 0},
		{"weapons", []ShopShelf{ShelfWeapons, ShelfBooks}, 1},
		{"magic", []ShopShelf{ShelfMagic, ShelfBooks}, 2},
		{"books", []ShopShelf{ShelfBooks}, 3},
		{"empty", nil, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := shellFrontEnd()
			f.Shop = NewShop(0)
			for _, shelf := range tc.stock {
				f.Shop.shelves[shelf] = []ShopItem{{Count: 1}}
			}
			s := f.townUI
			s.atSquare()
			s.shelfBase, s.packBase, s.shopBook = 8, 5, true
			s.Choose(1)
			if s.shopChosen != tc.want || s.ShopScreen().Chosen != tc.want || shopRacks(s).Selected != tc.want {
				t.Fatalf("selection=%d view=%d animation=%d; want %d", s.shopChosen, s.ShopScreen().Chosen, shopRacks(s).Selected, tc.want)
			}
			if s.shelfBase != 0 || s.packBase != 0 || s.shopBook {
				t.Fatal("shop entry retained scrolling or the spell book")
			}
			if tc.want >= 0 && (!shopRacks(s).Enabled[tc.want] || s.shopShelf != shopRoomShelves[tc.want].shelf) {
				t.Fatal("visible rack and item shelf disagree")
			}
		})
	}
}
