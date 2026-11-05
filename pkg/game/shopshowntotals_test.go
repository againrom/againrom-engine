package game

import (
	"testing"

	"againrom/pkg/formats/itempkt"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
)

func TestShopPendingTotalsUseShownPricesAndSettleServerPrices(t *testing.T) {
	for _, enchanted := range []bool{false, true} {
		name := "record"
		var effects []sim.ItemEffect
		if enchanted {
			name = "server effect"
			effects = []sim.ItemEffect{{Kind: 5, Operand: 1}}
		}
		t.Run(name, func(t *testing.T) {
			table := secondShopCellTable()
			table.Armors = shownPriceRows{secondShopTestCollection{3: {0, 0, 0, 0, 6}}}
			table.ItemRecords = map[uint16]itempkt.Record{0x1603: {Flags: 2, Price: 51, HasPrice: true}}
			it := secondshop.Item{Table: secondshop.Armor, Row: 3, Material: 1, Count: 2, Price: 50, Effects: effects}
			s, _ := secondShopCellPage(t, table, 0, []secondshop.Item{it})
			shop := s.deal()
			if !shop.TakeFromShelf(0, 0, 2) {
				t.Fatal("purchase")
			}
			it.Count = 3
			offer, ok := secondShopItem(secondShopTestTables, it)
			if !ok || !shop.PutOnTable(offer) {
				t.Fatal("sale")
			}
			shown := int32(51)
			if enchanted {
				shown = 50
			}
			v := s.ShopScreen()
			buy, sell := 2*shown, 3*((shown+1)/2)
			if v.Buy != buy || v.Sell != sell || v.Total != v.Purse+sell-buy {
				t.Errorf("pending totals buy %d sell %d exit %d; want %d %d %d", v.Buy, v.Sell, v.Total, buy, sell, v.Purse+sell-buy)
			}
			if shop.BuyTotal() != 100 || shop.SellTotal() != 75 {
				t.Fatal("presentation repriced settlement")
			}
			gold := s.sess.Town.gold
			s.shopBuy()
			if s.sess.Town.gold != gold-100 {
				t.Fatal("purchase did not charge server price")
			}
			s.shopSell()
			if s.sess.Town.gold != gold-100+75 || len(shop.table) != 0 {
				t.Fatal("sale did not credit server price")
			}
		})
	}
}
