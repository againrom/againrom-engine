package game

import (
	"reflect"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// The second game's shop doll fixture: an axe both classes wear, a staff only
// a mage wears and a targe, as bag units and as a Humans-row worn item.
var (
	secondDollAxe   = sim.ItemInstance{Code: 0x0101, Kind: 2, Price: 40}
	secondDollStaff = sim.ItemInstance{Code: 0x0102, Kind: 2, Price: 30}
	secondDollTarge = uint16(0x4201)
)

// secondDollTable is the shop fixture's item table read as the second game's.
func secondDollTable() *mapload.Table {
	t := shopTable()
	t.Weapons = shopCollection{{}, {name: "axe", price: 40, masks: [data.ShopShapes]uint16{1<<0 | 1<<1}},
		{name: "staff", price: 30, masks: [data.ShopShapes]uint16{1 << 0}, suit: 2}}
	e := base.GameROM2.Edition()
	t.Edition = &e
	return t
}

// secondDollShop is the second game's shop page over a campaign whose bag
// holds bag, with the hero wearing the targe and Iglez beside him.
func secondDollShop(t *testing.T, bag ...sim.ItemInstance) (*townScreen, *secondCampaign) {
	t.Helper()
	c := secondShopSample()
	*c.heroBag() = secondDollClone(bag)
	s := secondShopPage(c, 500)
	s.in.Table = secondDollTable()
	hero := mapload.PartyMember{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}
	hero.Worn[1] = secondDollTarge
	hero.WornItems[1] = mapload.ItemInstanceFromCode(secondDollTarge, s.in.Table)
	s.sess.Carried = []mapload.PartyMember{hero, {ID: "iglez", Name: "Iglez", CompanionNPC: 22}}
	if slot, ok := EquipTarget(data.ItemCode(secondDollAxe.Code), s.in.Table); !ok || slot != 1 {
		t.Fatalf("the fixture's axe wears in slot %d (%v)", slot, ok)
	}
	if slot, ok := EquipTarget(data.ItemCode(secondDollTarge), s.in.Table); !ok || slot != 2 {
		t.Fatalf("the fixture's targe wears in slot %d (%v)", slot, ok)
	}
	return s, c
}

// secondDollState is what a doll gesture may change: the party, the bag,
// the gold, the deal's table and the shelves.
type secondDollState struct {
	party   []mapload.PartyMember
	bag     []sim.ItemInstance
	gold    int
	table   []ShopPlace
	shelves [shopShelfCount][]ShopItem
}

func secondDollNow(s *townScreen, c *secondCampaign) secondDollState {
	st := secondDollState{party: mapload.CloneParty(s.sess.Carried), bag: secondDollClone(*c.heroBag()), gold: s.sess.Town.gold, table: c.shop.Table()}
	for i := range st.shelves {
		st.shelves[i] = append([]ShopItem(nil), c.shop.shelves[i]...)
	}
	return st
}

func secondDollClone(items []sim.ItemInstance) []sim.ItemInstance {
	out := make([]sim.ItemInstance, len(items))
	for i := range items {
		out[i] = items[i].Clone()
	}
	return out
}

func secondDollPack(s *townScreen, k int) ui.ShopControl {
	return ui.ShopControl{Kind: ui.ShopControlPackCell, Index: s.packCell(k)}
}

// In the second game's shop a bag item dragged onto the shown member's doll
// is worn in its slot and a worn item dragged to the bag is taken off into
// it as a bag unit; an item the hero's class may not use is refused and
// changes nothing. With NEXT the doll is Iglez's. No gesture moves gold or
// stock, and the bag still saves.
func TestSecondShopDollWearsFromTheBagAndTakesOffIntoIt(t *testing.T) {
	s, c := secondDollShop(t, secondDollAxe, secondDollStaff)
	doll := func(slot int) ui.ShopControl { return ui.ShopControl{Kind: ui.ShopControlDoll, Index: slot - 1} }
	before := secondDollNow(s, c)

	s.ShopDrag(secondDollPack(s, 1), doll(1))
	if got := secondDollNow(s, c); !reflect.DeepEqual(got, before) {
		t.Fatal("the fighter's doll took the mage's staff")
	}

	s.ShopDrag(doll(2), secondDollPack(s, 0))
	hero := s.sess.Carried[0]
	if !hero.WornItems[1].Empty() || hero.Worn[1] != 0 {
		t.Fatalf("the targe is still worn: %+v", hero.WornItems[1])
	}
	bag := *c.heroBag()
	if len(bag) != 3 || bag[2].Code != secondDollTarge || bag[2].Kind != 0 {
		t.Fatalf("the bag after the take-off is %+v, want the axe, the staff and the targe as a bag unit", bag)
	}
	if _, ok := secondshop.FromInstance(bag[2]); !ok {
		t.Fatalf("the taken-off targe %+v has no bag form", bag[2])
	}

	s.ShopDrag(secondDollPack(s, 0), doll(1))
	hero = s.sess.Carried[0]
	if hero.WornItems[0].Code != secondDollAxe.Code || hero.Worn[0] != secondDollAxe.Code {
		t.Fatalf("the hero wears %+v in slot 1, want the axe", hero.WornItems[0])
	}
	if bag = *c.heroBag(); len(bag) != 2 || bag[0].Code != secondDollStaff.Code || bag[1].Code != secondDollTarge {
		t.Fatalf("the bag after the wear is %+v, want the staff and the targe", bag)
	}
	if !itemEquipmentEmptyGame(c.party[0].WornItems) {
		t.Fatalf("the bag member wears %+v", c.party[0].WornItems)
	}

	s.ShopDrag(doll(1), secondDollPack(s, 0))
	if bag = *c.heroBag(); len(bag) != 3 || !reflect.DeepEqual(bag[2], secondDollAxe) || !s.sess.Carried[0].WornItems[0].Empty() {
		t.Fatalf("the axe taken off is %+v, want the unit worn", bag)
	}

	s.click(ui.ShopControlPickerNext, 0)
	s.ShopDrag(secondDollPack(s, 2), doll(1))
	if s.sess.Carried[1].WornItems[0].Code != secondDollAxe.Code || !s.sess.Carried[0].WornItems[0].Empty() {
		t.Fatalf("NEXT then a drag: Iglez wears %+v, the hero %+v", s.sess.Carried[1].WornItems[0], s.sess.Carried[0].WornItems[0])
	}
	s.ShopDrag(doll(1), secondDollPack(s, 0))
	if !s.sess.Carried[1].WornItems[0].Empty() || len(*c.heroBag()) != 3 {
		t.Fatalf("Iglez's axe was not taken off into the bag: %+v", *c.heroBag())
	}

	after := secondDollNow(s, c)
	if after.gold != before.gold || !reflect.DeepEqual(after.table, before.table) || !reflect.DeepEqual(after.shelves, before.shelves) {
		t.Fatal("a doll gesture moved gold, the table or the stock")
	}
	if err := captureSecondCampaign(c).validate(); err != nil {
		t.Fatalf("SAVE refuses the bag after the doll gestures: %v", err)
	}
}

// The composed shop room's pane takes the doll's drags, and after a wear the
// pane's figure and item text are the worn set's.
func TestSecondShopPaneDollTakesTheDrag(t *testing.T) {
	f, _, screen := secondCampaignFixtureFiles(t, false)
	f.Table = secondDollTable()
	f.Carried = []mapload.PartyMember{{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}}
	c := secondShopSample()
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}}, secondTownShop
	*c.heroBag() = []sim.ItemInstance{secondDollAxe}
	f.Town.second, f.Town.gold = c, 500
	screen.bindBankShop()
	screen.shop.shown = nil
	screen.composed = newTownSceneState(nil)
	d, ok := screen.composedTown()
	if !ok {
		t.Fatal("no composed town")
	}
	screen.composed.art[d.Town+"/shop"] = &townRoomArt{scene: &town.Art{}}
	v := screen.ShopScreen()
	if v.Layout == nil || !v.Layout.Character || v.Layout.NoWear || len(v.SlotInfo[0]) != 0 {
		t.Fatalf("the shop pane's layout %+v and slot 1 text %v, want a pane whose doll takes drags", v.Layout, v.SlotInfo[0])
	}
	screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: screen.packCell(0)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if v = screen.ShopScreen(); len(v.SlotInfo[0]) == 0 || f.Carried[0].WornItems[0].Code != secondDollAxe.Code {
		t.Fatalf("after the wear the pane's slot 1 text is %v", v.SlotInfo[0])
	}
}

// What the shop's doll puts on the hero is his own: a town SAVE writes it and
// a cold LOAD restores it on him, with the bag beside it.
func TestSecondShopWornItemsKeepThroughTownSaveAndColdLoad(t *testing.T) {
	f, _, screen := secondCampaignFixtureFiles(t, false)
	f.Table = secondDollTable()
	f.Carried = []mapload.PartyMember{{ID: "hero", Name: "Aldor", PlayerCharacter: true, StartingHero: true}}
	c := secondShopSample()
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}}, secondTownShop
	c.bank[768] = 10
	*c.heroBag() = []sim.ItemInstance{secondDollAxe, secondDollStaff}
	f.Town.second, f.Town.gold = c, 500
	screen.bindBankShop()
	screen.shop.shown = nil
	screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: screen.packCell(0)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	if f.Carried[0].WornItems[0].Code != secondDollAxe.Code {
		t.Fatal("the drag wore nothing")
	}
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	cold, _, _ := secondCampaignFixtureFiles(t, false)
	cold.Table = secondDollTable()
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("cold town LOAD", town, err)
	}
	worn := mapload.MemberItemEquipment(cold.Carried[0], cold.Table)
	if len(cold.Carried) != 1 || worn[0].Code != secondDollAxe.Code {
		t.Fatalf("the cold LOAD's hero wears %+v in slot 1, want the axe", worn[0])
	}
	if bag := cold.Town.second.party; len(bag) != 1 || len(bag[0].CarriedItems) != 1 || bag[0].CarriedItems[0].Code != secondDollStaff.Code {
		t.Fatalf("the cold LOAD's bag is %+v, want the staff", bag)
	}
}

// A double click on a wearable bag item is the shop's pack tap followed by
// the doll's drop of what the tap staged (pkg/ui useShopTap): the item ends
// on the shown member's doll, never on the bag member, and no gold or stock
// moves. A staff the fighter may not use and a book, which no doll wears,
// stay offered on the table and Undo returns them to the bag.
func TestSecondShopDoubleClickWearsAndKeepsTheItem(t *testing.T) {
	s, c := secondDollShop(t, secondDollAxe)
	before := secondDollNow(s, c)
	doubleClick := func() {
		s.ShopClick(secondDollPack(s, 0))
		s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlTableCell, Index: 0}, ui.ShopControl{Kind: ui.ShopControlDoll})
	}
	doubleClick()
	after := secondDollNow(s, c)
	if after.party[0].WornItems[0].Code != secondDollAxe.Code {
		t.Fatalf("the double click left the hero wearing %+v and the bag member %+v; bag %+v table %+v",
			after.party[0].WornItems[0], c.party[0].WornItems[0], after.bag, after.table)
	}
	if len(after.bag) != 0 || len(after.table) != 0 || after.gold != before.gold || !reflect.DeepEqual(after.shelves, before.shelves) {
		t.Fatalf("after the double click: bag %+v table %+v gold %d (was %d)", after.bag, after.table, after.gold, before.gold)
	}

	book := sim.ItemInstance{Code: 0x0e81, Price: 20, Effects: []sim.ItemEffect{{Kind: 42, Operand: 1}}}
	for _, unit := range []sim.ItemInstance{secondDollStaff, book} {
		s, c = secondDollShop(t, unit)
		before = secondDollNow(s, c)
		doubleClick()
		after = secondDollNow(s, c)
		if !reflect.DeepEqual(after.party, before.party) || len(after.bag) != 0 || len(after.table) != 1 || !after.table[0].Mine || after.table[0].ShopItem.Code != data.ItemCode(unit.Code) || after.gold != before.gold {
			t.Fatalf("the refused %#04x: bag %+v table %+v gold %d", unit.Code, after.bag, after.table, after.gold)
		}
		s.click(ui.ShopControlButton, 0)
		if after = secondDollNow(s, c); !reflect.DeepEqual(after, before) {
			t.Fatalf("Undo after the refused double click on %#04x: bag %+v table %+v", unit.Code, after.bag, after.table)
		}
	}
}
