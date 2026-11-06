package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func shopMemberCodes(items []sim.ItemInstance) []uint16 {
	var out []uint16
	for _, it := range items {
		out = append(out, it.Code)
	}
	return out
}

// In the second game's shop the chosen party member's bag is the strip: a
// purchase puts the item into that member's bag, a sale takes it from that
// member's bag, and each member's items stay where they were put through
// leaving the shop and a SAVE and LOAD.
func TestSecondShopTradesTheChosenMembersBag(t *testing.T) {
	const bought = 2<<12 | 6<<8 | 1<<5 | 3
	s, c := secondDollShop(t, secondDollAxe)
	table := s.in.Table
	pack := []sim.ItemInstance{secondshop.MissionUnit(secondDollStaff, secondMissionItem(table))}
	iglez, ok := memberWithPack(s.sess.Carried[1], pack, table, true)
	if !ok {
		t.Fatal("the fixture's companion takes no pack")
	}
	s.sess.Carried[1] = iglez
	heroBag := func() []uint16 { return shopMemberCodes(*c.heroBag()) }
	iglezPack := func() []uint16 {
		return shopMemberCodes(mapload.MemberCarriedItems(s.sess.Carried[1], table))
	}

	if got := shopMemberCodes(s.shopPackItemInstances()); !slices.Equal(got, []uint16{secondDollAxe.Code}) {
		t.Fatalf("the hero's strip shows %#04x, want his axe", got)
	}
	s.click(ui.ShopControlPickerNext, 0)
	if got := shopMemberCodes(s.shopPackItemInstances()); s.shopMemberIndex() != 1 || !slices.Equal(got, []uint16{secondDollStaff.Code}) {
		t.Fatalf("member %d: the strip shows %#04x, want Iglez's staff", s.shopMemberIndex(), got)
	}

	s.click(ui.ShopControlShelfPick, 0)
	s.click(ui.ShopControlShelfCell, 0)
	s.click(ui.ShopControlButton, 1)
	if s.sess.Town.gold != 500-125 || !slices.Equal(heroBag(), []uint16{secondDollAxe.Code}) ||
		!slices.Equal(iglezPack(), []uint16{secondDollStaff.Code, bought}) {
		t.Fatalf("a purchase for Iglez: gold %d, hero's bag %#04x, Iglez's pack %#04x", s.sess.Town.gold, heroBag(), iglezPack())
	}

	gold := s.sess.Town.gold
	s.click(ui.ShopControlPackCell, s.packCell(0))
	s.click(ui.ShopControlButton, 2)
	if s.sess.Town.gold <= gold || !slices.Equal(heroBag(), []uint16{secondDollAxe.Code}) ||
		!slices.Equal(iglezPack(), []uint16{bought}) {
		t.Fatalf("a sale by Iglez: gold %d -> %d, hero's bag %#04x, Iglez's pack %#04x", gold, s.sess.Town.gold, heroBag(), iglezPack())
	}

	s.click(ui.ShopControlPickerPrev, 0)
	if got := shopMemberCodes(s.shopPackItemInstances()); s.shopMemberIndex() != 0 || !slices.Equal(got, []uint16{secondDollAxe.Code}) {
		t.Fatalf("member %d: the strip shows %#04x, want the hero's axe", s.shopMemberIndex(), got)
	}
	c.shop.leave(c.heroBag())
	s.shopMember = 0
	s.click(ui.ShopControlPickerNext, 0)
	if got := shopMemberCodes(s.shopPackItemInstances()); !slices.Equal(got, []uint16{bought}) {
		t.Fatalf("after leaving, Iglez's strip shows %#04x", got)
	}

	f, _, _ := secondCampaignFixtureFiles(t, false)
	f.Table = table
	f.Carried = mapload.CloneParty(s.sess.Carried)
	c.current, c.available, c.room = partyPlace{2, 1}, []partyPlace{{2, 1}, {1, 10}}, secondTownShop
	c.bank[768] = 10
	f.Town.second, f.Town.gold = c, s.sess.Town.gold
	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatal(err)
	}
	cold := coldSecondTown(t, raw)
	if len(cold.Carried) != 2 || cold.Town.second == nil {
		t.Fatalf("the LOAD holds %d members", len(cold.Carried))
	}
	if got := shopMemberCodes(cold.Town.second.party[0].CarriedItems); !slices.Equal(got, []uint16{secondDollAxe.Code}) {
		t.Fatalf("after SAVE and LOAD the hero's bag is %#04x", got)
	}
	if got := shopMemberCodes(mapload.MemberCarriedItems(cold.Carried[1], table)); !slices.Equal(got, []uint16{bought}) {
		t.Fatalf("after SAVE and LOAD Iglez's pack is %#04x", got)
	}
}
