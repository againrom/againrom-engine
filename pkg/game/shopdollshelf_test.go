package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/ui"
)

// Two items worn in the form a mission leaves them (a carried weight) and put
// from the doll on the shop's table each stay themselves: the second never
// shows or becomes a copy of the first.
func TestSecondShopDollItemsOnTableStayDistinct(t *testing.T) {
	s, c := secondDollShop(t)
	armour, ok := secondShopTestTables.Units(secondshop.Item{Table: secondshop.Armor, Row: 3, Class: 1, Material: 2, Count: 1, Price: 125})
	if !ok || len(armour) != 1 {
		t.Fatalf("the cuirass has no bag unit: %+v", armour)
	}
	sword, cuirass := secondDollAxe.Clone(), armour[0].Clone()
	sword.Weight, sword.WeightPresent = 30, true
	cuirass.Weight, cuirass.WeightPresent = 50, true
	hero := &s.sess.Carried[0]
	hero.Worn[0], hero.WornItems[0] = sword.Code, sword
	hero.Worn[5], hero.WornItems[5] = cuirass.Code, cuirass
	doll := func(slot int) ui.ShopControl { return ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1} }
	cell := func(i int) ui.ShopControl { return ui.ShopControl{Kind: ui.ShopControlTableCell, Index: i} }

	s.ShopDrag(doll(1), cell(0))
	s.ShopDrag(doll(6), cell(1))
	places := c.shop.Table()
	if len(places) != 2 {
		t.Fatalf("the table holds %d places, want 2: %+v", len(places), places)
	}
	if places[0].ShopItem.Code != data.ItemCode(sword.Code) || places[1].ShopItem.Code != data.ItemCode(cuirass.Code) {
		t.Fatalf("the table holds %#x and %#x, want the sword and the cuirass", places[0].ShopItem.Code, places[1].ShopItem.Code)
	}
	v := s.ShopScreen()
	if v.Table[0].Price == v.Table[1].Price || v.Table[0].Count != 1 || v.Table[1].Count != 1 {
		t.Fatalf("the cells hold prices %d and %d, counts %d and %d", v.Table[0].Price, v.Table[1].Price, v.Table[0].Count, v.Table[1].Count)
	}
	if !hero.WornItems[0].Empty() || !hero.WornItems[5].Empty() {
		t.Fatalf("the hero still wears %+v and %+v", hero.WornItems[0], hero.WornItems[5])
	}
	s.leaveSecondShop()
	bag := *c.heroBag()
	if len(c.shop.Table()) != 0 || len(bag) != 2 || bag[0].Code == bag[1].Code {
		t.Fatalf("after leaving the table holds %d places and the bag %+v, want a sword and a cuirass", len(c.shop.Table()), bag)
	}
	if !hero.WornItems[0].Empty() || !hero.WornItems[5].Empty() {
		t.Fatalf("after leaving the hero wears %+v and %+v", hero.WornItems[0], hero.WornItems[5])
	}
}

// The first game's table keeps two items put from the doll apart as well, and
// the shop's leave returns both to the pack.
func TestFirstShopDollItemsOnTableStayDistinct(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Archives = shopFigureArchive(t)
	axe, helm := uint16(0x0101), uint16(0x2601)
	worn := s.shopWornItemSlots(0)
	worn[0], worn[5] = mapload.ItemInstanceFromCode(axe, f.Table), mapload.ItemInstanceFromCode(helm, f.Table)
	doll := func(slot int) ui.ShopControl { return ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1} }
	cell := func(i int) ui.ShopControl { return ui.ShopControl{Kind: ui.ShopControlTableCell, Index: i} }

	s.ShopDrag(doll(1), cell(0))
	s.ShopDrag(doll(6), cell(1))
	places := f.Shop.Table()
	if len(places) != 2 || places[0].ShopItem.Code != data.ItemCode(axe) || places[1].ShopItem.Code != data.ItemCode(helm) {
		t.Fatalf("the table holds %+v, want the axe and the helm", places)
	}
	if v := s.ShopScreen(); v.Table[0].Price == v.Table[1].Price {
		t.Fatalf("the cells hold one price %d", v.Table[0].Price)
	}
	if !worn[0].Empty() || !worn[5].Empty() {
		t.Fatalf("the hero still wears %+v and %+v", worn[0], worn[5])
	}
}
