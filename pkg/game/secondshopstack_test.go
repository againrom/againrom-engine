package game

import (
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The second game's shop stack fixture: a staff that casts a spell and an
// axe with one effect, each a pair of equal units, and a pair of equal
// rings, which the bag's rule already stacks.
var (
	secondStackStaff = sim.ItemInstance{Code: 0x0102, Kind: 2, Price: 300, Effects: []sim.ItemEffect{{Kind: 41, Operand: 3 | 2<<16}}}
	secondStackAxe   = sim.ItemInstance{Code: 0x0101, Kind: 2, Price: 90, Effects: []sim.ItemEffect{{Kind: 15, Operand: 4}}}
	secondStackRing  = sim.ItemInstance{Code: 0x0e00 | 80, Price: 200}
)

// secondStackCounts is the strip's stacks as code and count, in strip order.
func secondStackCounts(s *townScreen) [][2]uint32 {
	var out [][2]uint32
	for _, st := range s.shopPackStacks() {
		out = append(out, [2]uint32{uint32(st.Code), st.Count})
	}
	return out
}

// secondStackWant checks the strip holds the stacks want, in any order: a
// unit back from the table enters the bag at its end.
func secondStackWant(t *testing.T, who string, s *townScreen, want ...[2]uint32) {
	t.Helper()
	if len(want) == 0 {
		want = [][2]uint32{{0x0102, 2}, {0x0101, 2}, {0x0e50, 2}}
	}
	got := secondStackCounts(s)
	if len(got) != len(want) {
		t.Fatalf("%s: the strip shows %v, want %v", who, got, want)
	}
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Fatalf("%s: the strip shows %v, want %v", who, got, want)
		}
	}
}

// secondStackOf is the strip stack that holds code, -1 for none.
func secondStackOf(s *townScreen, code uint16) int {
	return slices.IndexFunc(s.shopPackStacks(), func(st sim.ItemStack) bool { return st.Code == code })
}

// TestSecondShopStripJoinsEqualMagicUnits: in the second game's shop two
// equal staves that cast a spell and two equal enchanted axes each show as
// one stack of two, for the hero and for a companion (DIV-1473). Two units
// put on the table one after the other hold one place, Undo returns both,
// a sale of the whole stack pays for two, and the hero's
// stack of axes is one stack of two again after SAVE and LOAD.
func TestSecondShopStripJoinsEqualMagicUnits(t *testing.T) {
	units := []sim.ItemInstance{secondStackStaff, secondStackStaff, secondStackAxe, secondStackAxe, secondStackRing, secondStackRing}
	s, c := secondDollShop(t, units...)
	table := s.in.Table
	var pack []sim.ItemInstance
	for _, u := range units {
		pack = append(pack, secondshop.MissionUnit(u, secondMissionItem(table)))
	}
	iglez, ok := memberWithPack(s.sess.Carried[1], pack, table, true)
	if !ok {
		t.Fatal("the fixture's companion takes no pack")
	}
	s.sess.Carried[1] = iglez
	secondStackWant(t, "the hero", s)

	s.click(ui.ShopControlPackCell, s.packCell(0))
	s.click(ui.ShopControlPackCell, s.packCell(0))
	if got := c.shop.Table(); len(got) != 1 || got[0].ShopItem.Count != 2 || len(*c.heroBag()) != 4 {
		t.Fatalf("two staves put on the table one by one: %d places, bag %d", len(got), len(*c.heroBag()))
	}
	s.click(ui.ShopControlButton, 0)
	if len(c.shop.Table()) != 0 {
		t.Fatalf("Undo left %d places", len(c.shop.Table()))
	}
	secondStackWant(t, "the hero after Undo", s)

	gold := s.sess.Town.gold
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: s.packCell(secondStackOf(s, secondStackStaff.Code)), Shift: true})
	if got := c.shop.Table(); len(got) != 1 || got[0].ShopItem.Count != 2 {
		t.Fatalf("the whole stack of staves on the table: %v", got)
	}
	s.click(ui.ShopControlButton, 2)
	if s.sess.Town.gold != gold+300 || len(*c.heroBag()) != 4 || len(c.shop.Table()) != 0 {
		t.Fatalf("a sale of two staves: gold %d -> %d, bag %d, table %d", gold, s.sess.Town.gold, len(*c.heroBag()), len(c.shop.Table()))
	}
	secondStackWant(t, "the hero after the sale", s, [2]uint32{0x0101, 2}, [2]uint32{0x0e50, 2})

	s.click(ui.ShopControlPickerNext, 0)
	if s.shopMemberIndex() != 1 {
		t.Fatalf("the picker shows member %d", s.shopMemberIndex())
	}
	secondStackWant(t, "Iglez", s)

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
	if cold.Town.second == nil || len(cold.Carried) != 2 {
		t.Fatalf("the LOAD holds %d members", len(cold.Carried))
	}
	back := secondShopPage(cold.Town.second, cold.Town.gold)
	back.in.Table = table
	back.sess.Carried = cold.Carried
	secondStackWant(t, "the hero after SAVE and LOAD", back, [2]uint32{0x0101, 2}, [2]uint32{0x0e50, 2})
}
