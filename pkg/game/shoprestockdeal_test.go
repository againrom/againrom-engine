package game

import (
	"reflect"
	"testing"

	"againrom/pkg/random"
)

func TestShopRestockKeepsOpenDeal(t *testing.T) {
	mutable := func(s *Shop) *Shop {
		out := cloneShopMutation(s)
		out.tables.TemplateCode = nil
		return out
	}
	for _, row := range []struct {
		name string
		make func() (*FrontEnd, *townScreen)
	}{
		{"chapter", func() (*FrontEnd, *townScreen) {
			f := shellFrontEnd()
			f.Table = shopTable()
			f.Shop = NewShop(1000)
			f.Shop.Generate(f.Table, 7)
			return f, f.townUI
		}},
		{"bank", func() (*FrontEnd, *townScreen) { return sleepTown(1) }},
	} {
		for _, mine := range []bool{false, true} {
			t.Run(row.name+map[bool]string{false: " merchant", true: " customer"}[mine], func(t *testing.T) {
				f, screen := row.make()
				control, _ := row.make()
				s := screen.campaignTown().townShop()
				if mine {
					if !s.PutOnTable(magicUnit(70, 1, 11)) {
						t.Fatal("offer")
					}
				} else {
					for k, shelf := range s.shelves {
						if len(shelf) > 0 {
							if !s.TakeFromShelf(ShopShelf(k), 0, 1) {
								t.Fatal("take")
							}
							break
						}
					}
				}
				if len(s.table) != 1 {
					t.Fatal("open deal fixture")
				}
				before, gold := mutable(s), f.Town.gold
				if screen.sleep() {
					t.Error("restocked an open deal")
				}
				if !reflect.DeepEqual(before, mutable(s)) || f.Town.gold != gold {
					t.Error("refusal changed stock, deal, ordinal, category or gold")
				}
				if f.randomService().Stream(random.ShopStock).Raw() != control.randomService().Stream(random.ShopStock).Raw() {
					t.Error("refusal advanced the stock stream")
				}
			})
		}
	}
}

func TestShopRestockResetsTheBankCategory(t *testing.T) {
	_, screen := sleepTown(1)
	s := screen.state()
	s.shop.category = 2
	if !screen.sleep() || s.shop.category != 0 {
		t.Fatal("Sleep did not select category zero", s.shop.category)
	}
}
