package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// tableUnits counts every unit the second game's shop page holds: the hero's
// bag, Iglez's pack, the table and the shelves.
func tableUnits(s *townScreen, c *secondCampaign) int {
	n := len(*c.heroBag()) + len(mapload.MemberCarriedItems(s.sess.Carried[1], s.in.Table))
	for _, p := range c.shop.Table() {
		n += int(p.ShopItem.Count)
	}
	for _, shelf := range c.shop.shelves {
		for _, it := range shelf {
			n += int(it.Count)
		}
	}
	return n
}

// tableMemberShop is the shop page with the hero's axe in his bag and Iglez's
// staff in his pack.
func tableMemberShop(t *testing.T) (*townScreen, *secondCampaign) {
	t.Helper()
	s, c := secondDollShop(t, secondDollAxe)
	pack := []sim.ItemInstance{secondDollUnit(s, secondDollStaff)}
	iglez, ok := memberWithPack(s.sess.Carried[1], pack, s.in.Table, true)
	if !ok {
		t.Fatal("the fixture's companion takes no pack")
	}
	s.sess.Carried[1] = iglez
	return s, c
}

func secondDollUnit(s *townScreen, in sim.ItemInstance) sim.ItemInstance {
	return secondDollClone([]sim.ItemInstance{in})[0]
}

func iglezCodes(s *townScreen) []uint16 {
	return shopMemberCodes(mapload.MemberCarriedItems(s.sess.Carried[1], s.in.Table))
}

// The hero's axe put on the table stays there when the shown character
// changes, and Iglez takes it off into his own pack.
func TestSecondShopTableCarriesAnItemToAnotherMember(t *testing.T) {
	s, c := tableMemberShop(t)
	total := tableUnits(s, c)
	s.click(ui.ShopControlPackCell, s.packCell(0))
	if len(c.shop.Table()) != 1 || len(*c.heroBag()) != 0 {
		t.Fatalf("table %v, hero's bag %d", c.shop.Table(), len(*c.heroBag()))
	}
	s.click(ui.ShopControlPickerNext, 0)
	if s.shopMemberIndex() != 1 || len(c.shop.Table()) != 1 || !c.shop.Table()[0].Mine {
		t.Fatalf("member %d, table %v after the change", s.shopMemberIndex(), c.shop.Table())
	}
	s.click(ui.ShopControlTableCell, 0)
	if len(c.shop.Table()) != 0 || len(*c.heroBag()) != 0 || !slices.Contains(iglezCodes(s), secondDollAxe.Code) || tableUnits(s, c) != total {
		t.Fatalf("table %v, hero's bag %d, Iglez %#04x, units %d of %d", c.shop.Table(), len(*c.heroBag()), iglezCodes(s), tableUnits(s, c), total)
	}
}

// A sale of the carried item pays as before.
func TestSecondShopTableSellsAnItemOfAnotherMember(t *testing.T) {
	s, c := tableMemberShop(t)
	s.click(ui.ShopControlPackCell, s.packCell(0))
	s.click(ui.ShopControlPickerNext, 0)
	gold := s.sess.Town.gold
	s.click(ui.ShopControlButton, 2)
	if s.sess.Town.gold <= gold || len(c.shop.Table()) != 0 || len(*c.heroBag()) != 0 || slices.Contains(iglezCodes(s), secondDollAxe.Code) {
		t.Fatalf("gold %d -> %d, table %v, hero's bag %d, Iglez %#04x", gold, s.sess.Town.gold, c.shop.Table(), len(*c.heroBag()), iglezCodes(s))
	}
}

// The merchant's item on the table stays through the change and the
// purchase goes to the member shown when it is paid.
func TestSecondShopTableBuysForTheMemberShownAtPayment(t *testing.T) {
	const bought = 2<<12 | 6<<8 | 1<<5 | 3
	s, c := tableMemberShop(t)
	total := tableUnits(s, c)
	s.click(ui.ShopControlShelfPick, 0)
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlPickerNext, 0)
	if len(c.shop.Table()) != 1 || c.shop.Table()[0].Mine {
		t.Fatalf("table %v after the change", c.shop.Table())
	}
	s.click(ui.ShopControlButton, 1)
	if s.sess.Town.gold != 500-125 || !slices.Contains(iglezCodes(s), bought) || len(*c.heroBag()) != 1 || tableUnits(s, c) != total {
		t.Fatalf("gold %d, Iglez %#04x, hero's bag %d, units %d of %d", s.sess.Town.gold, iglezCodes(s), len(*c.heroBag()), tableUnits(s, c), total)
	}
}

// tableBothSides puts the hero's axe and a merchant item on the table, then
// shows Iglez.
func tableBothSides(t *testing.T) (*townScreen, *secondCampaign, int) {
	t.Helper()
	s, c := tableMemberShop(t)
	total := tableUnits(s, c)
	s.click(ui.ShopControlPackCell, s.packCell(0))
	s.click(ui.ShopControlShelfPick, 0)
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlPickerNext, 0)
	if len(c.shop.Table()) != 2 || s.shopMemberIndex() != 1 {
		t.Fatalf("table %v, member %d", c.shop.Table(), s.shopMemberIndex())
	}
	return s, c, total
}

// Leaving the room, by the exit or by any room change, sends the player's
// unit to the pack shown and the merchant's to its shelf.
func TestSecondShopTableLeavingKeepsEveryUnit(t *testing.T) {
	s, c, total := tableBothSides(t)
	townHooks["shop-leave"].run(s, "")
	if len(c.shop.Table()) != 0 || !slices.Contains(iglezCodes(s), secondDollAxe.Code) || len(*c.heroBag()) != 0 || tableUnits(s, c) != total {
		t.Fatalf("table %v, Iglez %#04x, hero's bag %d, units %d of %d", c.shop.Table(), iglezCodes(s), len(*c.heroBag()), tableUnits(s, c), total)
	}
	if c.shop.shelves[0][0].Count != 3 {
		t.Fatalf("the shelf holds %d, want 3", c.shop.shelves[0][0].Count)
	}
}

// A pack that refuses the unit gets none: it goes to the hero's bag, the
// shop's own undo, and nothing vanishes.
func TestSecondShopTableLeavingWithARefusingPackLosesNothing(t *testing.T) {
	s, c := tableMemberShop(t)
	*c.heroBag(), _ = secondShopTestTables.Units(secondshop.Item{Table: secondshop.Magic, Row: 70, Count: 4, Price: 83})
	total := tableUnits(s, c)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: s.packCell(0), Shift: true})
	s.click(ui.ShopControlPickerNext, 0)
	if len(c.shop.Table()) != 1 || len(*c.heroBag()) != 0 {
		t.Fatalf("table %v, hero's bag %d", c.shop.Table(), len(*c.heroBag()))
	}
	write := s.row.packWrite
	s.row.packWrite = func(sess *CampaignSession, i int, items []sim.ItemInstance, refresh bool) bool {
		return i != 1 && write(sess, i, items, refresh)
	}
	townHooks["shop-leave"].run(s, "")
	if len(c.shop.Table()) != 0 || len(iglezCodes(s)) != 1 || len(*c.heroBag()) != 4 || tableUnits(s, c) != total {
		t.Fatalf("table %v, Iglez %#04x, hero's bag %d, units %d of %d", c.shop.Table(), iglezCodes(s), len(*c.heroBag()), tableUnits(s, c), total)
	}
}

// A save made with units on the table writes them in the pack shown, and its
// LOAD holds them.
func TestSecondShopTableSaveAndLoadKeepEveryUnit(t *testing.T) {
	s, c, total := tableBothSides(t)
	hero, iglez := tableSaveAndLoad(t, s, c, total)
	if len(hero) != 0 || !slices.Contains(iglez, secondDollAxe.Code) || len(iglez) != 2 {
		t.Fatalf("after the LOAD: hero's bag %#04x, Iglez %#04x", hero, iglez)
	}
}

// The same save with the hero still shown writes the unit in his bag.
func TestSecondShopTableSaveAndLoadKeepTheShownMembersUnit(t *testing.T) {
	s, c := tableMemberShop(t)
	total := tableUnits(s, c)
	s.click(ui.ShopControlPackCell, s.packCell(0))
	hero, iglez := tableSaveAndLoad(t, s, c, total)
	if !slices.Equal(hero, []uint16{secondDollAxe.Code}) || len(iglez) != 1 {
		t.Fatalf("after the LOAD: hero's bag %#04x, Iglez %#04x", hero, iglez)
	}
}

// tableSaveAndLoad saves the open shop page's game, checks the live units and
// returns the hero's and Iglez's codes a cold LOAD holds.
func tableSaveAndLoad(t *testing.T, s *townScreen, c *secondCampaign, total int) (hero, iglez []uint16) {
	t.Helper()
	f, _, _ := secondCampaignFixtureFiles(t, false)
	f.Table = s.in.Table
	f.Carried = s.sess.Carried
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}, {1, 10}}, secondTownShop
	c.bank[768] = 10
	f.Town.second, f.Town.gold = c, s.sess.Town.gold
	s.sess = &f.CampaignSession
	f.townUI = s
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.shop.Table()) != 0 || tableUnits(s, c) != total {
		t.Fatalf("after the save: table %v, units %d of %d", c.shop.Table(), tableUnits(s, c), total)
	}
	raw, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	cold := coldSecondTown(t, raw)
	if len(cold.Town.second.party) != 0 {
		hero = shopMemberCodes(cold.Town.second.party[0].CarriedItems)
	}
	return hero, shopMemberCodes(mapload.MemberCarriedItems(cold.Carried[1], s.in.Table))
}
