package game

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/secondshop"
)

func secondShopSample() secondShop {
	s := newSecondShop()
	s.filled = 1
	s.stock[0] = []secondshop.Item{{Kind: secondshop.Armor, Row: 3, Class: 1, Material: 2, Quantity: 3, Price: 125}}
	s.stock[3] = []secondshop.Item{{Kind: secondshop.Book, Spell: 5, Quantity: 2, Price: 1000}}
	s.hero = []secondshop.Item{{Kind: secondshop.Magic, Row: 70, Quantity: 4, Price: 83}}
	return s
}

// TestSecondShopDealSettlesGoldAndItems runs a purchase and a sale through
// the campaign's shop: gold moves by q*P and floor(q*P/2+0.5), items move
// between the shelf and the hero, and an unaffordable purchase stays open.
func TestSecondShopDealSettlesGoldAndItems(t *testing.T) {
	s := secondShopSample()
	gold := 300
	if !s.take(0) || !s.take(0) || s.stock[0][0].Quantity != 1 || len(s.buying) != 1 || s.buying[0].Quantity != 2 {
		t.Fatalf("take: stock %v buying %v", s.stock[0], s.buying)
	}
	if !s.buy(&gold) || gold != 50 || len(s.buying) != 0 || len(s.hero) != 2 || s.hero[1].Quantity != 2 {
		t.Fatalf("buy: gold %d buying %v hero %v", gold, s.buying, s.hero)
	}
	s.take(0)
	if s.buy(&gold) || gold != 50 || len(s.buying) != 1 {
		t.Fatalf("unaffordable buy settled: gold %d buying %v", gold, s.buying)
	}
	if !s.offer(1) || !s.sell(&gold) || gold != 50+125 || s.stock[0][0].Quantity != 2 {
		t.Fatalf("sell: gold %d stock %v", gold, s.stock[0])
	}
	s.leave()
	if len(s.buying) != 0 || s.stock[0][0].Quantity != 3 || len(s.hero) != 1 {
		t.Fatalf("leave: stock %v hero %v", s.stock[0], s.hero)
	}
	// Loss control: an odd price credits floor(q*P/2+0.5), not the cell's
	// ((P+1)/2)*q.
	s.hero = []secondshop.Item{{Kind: secondshop.Magic, Row: 70, Quantity: 3, Price: 83}}
	gold = 0
	s.offer(0)
	s.sell(&gold)
	if gold != 125 {
		t.Fatalf("sale credit of 3 at 83 = %d, want 125", gold)
	}
}

// TestSecondShopSaveRoundTrip writes the shop with an open deal: the SAV
// holds the deal undone, the live deal stays open, and an old save without
// a shop restores the new game's records.
func TestSecondShopSaveRoundTrip(t *testing.T) {
	c := &secondCampaign{current: secondLocation{1, 10}, available: []secondLocation{{1, 10}}, shop: secondShopSample()}
	c.shop.take(0)
	c.shop.offer(0)
	captured := captureSecondCampaign(c)
	if len(c.shop.buying) != 1 || len(c.shop.selling) != 1 || c.shop.stock[0][0].Quantity != 2 {
		t.Fatal("capture changed the live deal")
	}
	raw, err := json.Marshal(captured)
	if err != nil {
		t.Fatal(err)
	}
	var decoded currentSecondCampaign
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	want := secondShopSample()
	if got := decoded.restore().shop; !reflect.DeepEqual(got, want) {
		t.Fatalf("restored shop %+v, want %+v", got, want)
	}
	old, _ := json.Marshal(struct {
		Bank      []int32
		Current   currentSecondLocation
		Available []currentSecondLocation
	}{make([]int32, 1024), currentSecondLocation{1, 10}, []currentSecondLocation{{1, 10}}})
	var legacy currentSecondCampaign
	if err := json.Unmarshal(old, &legacy); err != nil {
		t.Fatal(err)
	}
	if got := legacy.restore().shop; !reflect.DeepEqual(got, newSecondShop()) {
		t.Fatalf("an old save's shop %+v", got)
	}
	if fresh := captureSecondCampaign(newSecondCampaign()); fresh.Shop != nil {
		t.Fatal("a new game's shop is written")
	}
	for name, change := range map[string]func(s map[string]any){
		"kind":     func(s map[string]any) { s["Hero"].([]any)[0].(map[string]any)["Kind"] = 9 },
		"quantity": func(s map[string]any) { s["Hero"].([]any)[0].(map[string]any)["Quantity"] = -1 },
		"filled":   func(s map[string]any) { s["Filled"] = 4 },
		"unknown":  func(s map[string]any) { s["Deal"] = 1 },
	} {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		change(doc["Shop"].(map[string]any))
		bad, _ := json.Marshal(doc)
		var refused currentSecondCampaign
		if err := json.Unmarshal(bad, &refused); err == nil || strings.Contains(err.Error(), "bank") {
			t.Errorf("%s: a malformed shop decoded: %v", name, err)
		}
	}
}

// TestSecondShopOfferedByPlace buys one unit of a two-unit shelf entry and
// offers it back: the unit left on the shelf equals the offered item in every
// field, yet only the offered place carries the sale price.
func TestSecondShopOfferedByPlace(t *testing.T) {
	s := newSecondShop()
	s.filled = 1
	s.stock[0] = []secondshop.Item{{Kind: secondshop.Armor, Row: 3, Class: 1, Material: 2, Quantity: 2, Price: 125}}
	gold := 1000
	if !s.take(0) || !s.buy(&gold) || !s.offer(len(s.hero)-1) {
		t.Fatalf("setup: stock %v hero %v selling %v", s.stock[0], s.hero, s.selling)
	}
	c := &secondCampaign{shop: s}
	list := (&secondCampaignScreen{}).shopList(c, secondShopStock)
	if len(list) != 2 || list[0] != list[1] {
		t.Fatalf("shelf and sale entries should be equal: %v", list)
	}
	if s.offeredAt(0) || !s.offeredAt(1) {
		t.Errorf("offered places: shelf %v, sale %v, want false and true", s.offeredAt(0), s.offeredAt(1))
	}
	// Loss control: matching by value, as before, marks the shelf entry too.
	byValue := func(it secondshop.Item) bool {
		for _, o := range s.selling {
			if o == it {
				return true
			}
		}
		return false
	}
	if !byValue(list[0]) {
		t.Error("the value match no longer reproduces the shelf entry as offered")
	}
}
