package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/random"
	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// nextSecondStock is the stock the next fill of town id's record draws from
// f's shop stock stream; it advances that stream.
func nextSecondStock(f *FrontEnd, id int) secondshop.Stock {
	tables, _ := secondShopTables(f.Table)
	want := tables.Fill(*f.Town.second.shop.records[id], f.randomService().Stream(random.ShopStock).Raw)
	for k, shelf := range stockOf(tables, want) {
		want[k] = secondItems(shelf)
	}
	return want
}

func sameStockStream(t *testing.T, f, control *FrontEnd) {
	t.Helper()
	if a, b := f.randomService().Stream(random.ShopStock).Raw(), control.randomService().Stream(random.ShopStock).Raw(); a != b {
		t.Fatal("the shop stock stream is at another draw", a, b)
	}
}

func leaveSecondTown(t *testing.T, s *townScreen) {
	t.Helper()
	destinationTownCampaign{s}.leave()
	if c := s.state(); c.current != (partyPlace{}) || c.mapOpen == nil {
		t.Fatal("the party did not leave for the map", c.current)
	}
}

// A town arrival from the map refills the shop: every category is emptied
// and drawn anew from the town's record on the shop stock stream, and the
// departure draws nothing (R2-ENGINE-470, R2-ENGINE-471).
func TestSecondTownArrivalRefillsTheShop(t *testing.T) {
	for _, id := range []int{2, 3} {
		f, s := sleepTown(id)
		control, _ := sleepTown(id)
		c := f.Town.second
		old := c.shop.stock()
		c.shop.category = 2
		leaveSecondTown(t, s)
		if !reflect.DeepEqual(c.shop.stock(), old) {
			t.Fatal("the departure drew the stock")
		}
		want := nextSecondStock(control, id)
		if act := s.enterLocation(secondTown(id)); act.Msg != "" || act.Open != nil {
			t.Fatalf("town %d arrival: %+v", id, act)
		}
		got := c.shop.stock()
		if !reflect.DeepEqual(got, want) || reflect.DeepEqual(got, old) {
			t.Fatalf("town %d arrival: the stock is not the next draw of its record", id)
		}
		if c.shop.filled != id || c.shop.category != 0 {
			t.Fatalf("town %d arrival: filled %d, category %d", id, c.shop.filled, c.shop.category)
		}
		sameStockStream(t, f, control)
	}
}

// An open deal keeps the stock through an arrival (owner ruling): the shop,
// its deal and the stock stream stay as they were.
func TestSecondTownArrivalKeepsAnOpenDeal(t *testing.T) {
	f, s := sleepTown(2)
	control, _ := sleepTown(2)
	c := f.Town.second
	for k, shelf := range c.shop.shelves {
		if len(shelf) > 0 {
			if !c.shop.TakeFromShelf(ShopShelf(k), 0, 1) {
				t.Fatal("take")
			}
			break
		}
	}
	if len(c.shop.table) != 1 {
		t.Fatal("open deal fixture")
	}
	leaveSecondTown(t, s)
	mutable := func(s *Shop) *Shop {
		out := cloneShopMutation(s)
		out.tables.TemplateCode = nil
		return out
	}
	before := mutable(&c.shop)
	if act := s.enterLocation(secondTown(2)); act.Msg != "" {
		t.Fatal(act.Msg)
	}
	if !reflect.DeepEqual(before, mutable(&c.shop)) {
		t.Fatal("the arrival changed the stock or the deal")
	}
	sameStockStream(t, f, control)
}

// An arrival at a mission sends no refill (R2-ENGINE-470): the stock and the
// stock stream stay.
func TestSecondMissionArrivalDrawsNoStock(t *testing.T) {
	f, s := sleepTown(2)
	control, _ := sleepTown(2)
	c := f.Town.second
	c.available = append(c.available, partyPlace{kind: placeMission, id: 10})
	s.openMission = func(int) ui.MapOpener { return nil }
	old := c.shop.stock()
	leaveSecondTown(t, s)
	s.enterLocation(partyPlace{kind: placeMission, id: 10})
	if !reflect.DeepEqual(c.shop.stock(), old) {
		t.Fatal("the mission arrival drew the stock")
	}
	sameStockStream(t, f, control)
}

// A SAV written in a town before a departure and after the arrival back
// carries the same shop record, and a SAV after the arrival does not depend
// on the stock drawn: no shop byte moves with the refill.
func TestSecondTownArrivalWritesNoShopByte(t *testing.T) {
	f, s := sleepTown(2)
	save := func() ([]byte, json.RawMessage) {
		snapshot, _, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, "Town")
		if err != nil {
			t.Fatal(err)
		}
		doc := engineSaveDocument(t, raw)
		actions, err := readCurrentActions(&doc)
		if err != nil {
			t.Fatal(err)
		}
		shop, err := json.Marshal(actions.Session.Second.Shop)
		if err != nil {
			t.Fatal(err)
		}
		return raw, shop
	}
	_, before := save()
	leaveSecondTown(t, s)
	drawn := f.Town.second.shop.stock()
	s.enterLocation(secondTown(2))
	if reflect.DeepEqual(f.Town.second.shop.stock(), drawn) {
		t.Fatal("the arrival did not refill the shop")
	}
	raw, after := save()
	if !bytes.Equal(before, after) {
		t.Fatalf("the shop record moved: %s, %s", before, after)
	}
	tables, _ := secondShopTables(f.Table)
	f.Town.second.shop.drawStock(tables, 2, f.randomService().Stream(random.ShopStock).Raw)
	if again, _ := save(); !bytes.Equal(raw, again) {
		t.Fatal("the SAV bytes depend on the drawn stock")
	}
}
