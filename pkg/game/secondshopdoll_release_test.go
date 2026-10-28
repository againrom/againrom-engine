package game

import (
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondDollReleaseCount is how many units with code the campaign's bag holds.
func secondDollReleaseCount(c *secondCampaign, code uint16) int {
	var n int
	for _, in := range *c.heroBag() {
		if in.Code == code {
			n++
		}
	}
	return n
}

// secondDollReleasePack is the window point of the pack cell showing code.
func secondDollReleasePack(t *testing.T, app *ui.App, s *townScreen, code uint16) ui.ShopControl {
	t.Helper()
	k := slices.IndexFunc(s.shopPackStacks(), func(st sim.ItemStack) bool { return st.Code == code })
	if k < 0 {
		t.Fatalf("no pack cell shows %#04x", code)
	}
	return ui.ShopControl{Kind: ui.ShopControlPackCell, Index: s.packCell(k)}
}

// secondDollReleaseWear buys the shown rows of each open category one by one
// and applies gesture to the bought unit's pack cell until the hero wears
// one. A refused unit leaves the hero's equipment as it was and stays in the
// bag or on the table, where UNDO returns it; no gesture moves gold. It
// returns the worn unit and its slot.
func secondDollReleaseWear(t *testing.T, f *FrontEnd, app *ui.App, s *townScreen, gesture func(ui.ShopControl)) (sim.ItemInstance, int) {
	t.Helper()
	c := f.Town.second
	var refused int
	for cat := 0; cat < 4; cat++ {
		if !s.shelfOpen(cat) {
			continue
		}
		secondShopActivate(t, f, app, []string{"SHELF 1", "SHELF 2", "SHELF 3", "SHELF 4"}[cat])
		for n := 1; n <= 3 && n <= len(c.shop.visibleShelf()); n++ {
			units := len(*c.heroBag())
			secondShopActivate(t, f, app, []string{"ITEM 1", "ITEM 2", "ITEM 3"}[n-1], "BUY")
			bag := *c.heroBag()
			if len(bag) != units+1 {
				t.Fatalf("a buy of one unit left %d units in the bag, was %d", len(bag), units)
			}
			unit := bag[len(bag)-1].Clone()
			slot, ok := EquipTarget(data.ItemCode(unit.Code), f.Table)
			if !ok {
				continue
			}
			worn, gold, held := mapload.MemberItemEquipment(f.Carried[0], f.Table), f.Town.gold, secondDollReleaseCount(c, unit.Code)
			gesture(secondDollReleasePack(t, app, s, unit.Code))
			if f.Town.gold != gold {
				t.Fatalf("a gesture on %#04x moved the gold %d -> %d", unit.Code, gold, f.Town.gold)
			}
			now := mapload.MemberItemEquipment(f.Carried[0], f.Table)
			if now[slot-1].Code == unit.Code {
				if secondDollReleaseCount(c, unit.Code) != held-1 || len(c.shop.Table()) != 0 {
					t.Fatalf("the worn %#04x: bag holds %d of it, was %d; table %+v", unit.Code, secondDollReleaseCount(c, unit.Code), held, c.shop.Table())
				}
				t.Logf("worn %#04x in slot %d after %d refusals", unit.Code, slot, refused)
				return unit, slot
			}
			if !secondDollReleaseSame(now, worn) {
				t.Fatalf("a refused %#04x changed the hero's equipment", unit.Code)
			}
			if secondDollReleaseCount(c, unit.Code) != held {
				if table := c.shop.Table(); len(table) != 1 || !table[0].Mine || table[0].ShopItem.Code != data.ItemCode(unit.Code) {
					t.Fatalf("a refused %#04x left the bag and the table holds %+v", unit.Code, table)
				}
				secondShopActivate(t, f, app, "UNDO")
				if secondDollReleaseCount(c, unit.Code) != held {
					t.Fatalf("UNDO did not return the refused %#04x", unit.Code)
				}
			}
			refused++
		}
	}
	t.Fatalf("the hero wore none of the shown rows; %d refused", refused)
	return sim.ItemInstance{}, 0
}

// secondDollReleaseSame reports whether two equipment sets hold the same
// codes.
func secondDollReleaseSame(a, b [sim.EquipSlots]sim.ItemInstance) bool {
	for i := range a {
		if a[i].Code != b[i].Code {
			return false
		}
	}
	return true
}

// secondDollReleaseCold saves the town as name and LOADs it in a fresh
// front end: the hero wears code in slot and the bag holds held of it.
func secondDollReleaseCold(t *testing.T, f *FrontEnd, app *ui.App, name string, code uint16, slot, held int) {
	t.Helper()
	dir := secondMissionSaveDirectory(t)
	raw := secondTownNamedSave(t, f, app, dir, name)
	cold, _ := secondTownCold(t, dir, name)
	if len(cold.Carried) == 0 {
		t.Fatal("the cold LOAD has no party")
	}
	worn := mapload.MemberItemEquipment(cold.Carried[0], cold.Table)
	if worn[slot-1].Code != code || secondDollReleaseCount(cold.Town.second, code) != held {
		t.Fatalf("cold LOAD: slot %d holds %#04x, bag %d of %#04x; want it worn and %d in the bag", slot, worn[slot-1].Code,
			secondDollReleaseCount(cold.Town.second, code), code, held)
	}
	t.Logf("cold LOAD of a %d-byte SAV keeps %#04x worn in slot %d", len(raw), code, slot)
}

// TestReleaseSecondShopDollWearsAndTakesOff opens town 1's shop and drags
// bought items from the bag onto the hero's doll until one is worn: a unit
// his class may not use is refused, the worn one leaves the bag and no gold
// moves. A drag from the doll to the bag takes it off into the bag; worn
// again, a town SAVE and a fresh LOAD keep it on him.
func TestReleaseSecondShopDollWearsAndTakesOff(t *testing.T) {
	secondGameRoot(t)
	f, app, s, _ := secondShopTownOne(t)
	secondShopEnter(t, f, app, s)
	c := f.Town.second
	f.Town.gold = 5000000
	drag := func(from ui.ShopControl) {
		secondShopDrag(t, app, secondShopPagePoint(t, app, "pack", from.Index), secondShopPagePoint(t, app, "doll_box", 0))
	}
	unit, slot := secondDollReleaseWear(t, f, app, s, drag)
	secondShopWitness(t, app, "town1-shop-doll-worn")
	held, gold := secondDollReleaseCount(c, unit.Code), f.Town.gold
	secondShopDrag(t, app, secondShopPagePoint(t, app, "doll", slot), secondShopPagePoint(t, app, "pack", s.packCell(0)))
	if worn := mapload.MemberItemEquipment(f.Carried[0], f.Table); worn[slot-1].Code == unit.Code || secondDollReleaseCount(c, unit.Code) != held+1 || f.Town.gold != gold {
		t.Fatalf("take-off: slot %d holds %#04x, bag %d of it (was %d), gold %d -> %d", slot, worn[slot-1].Code,
			secondDollReleaseCount(c, unit.Code), held, gold, f.Town.gold)
	}
	if !secondDollReleaseBagForm(*c.heroBag(), unit.Code) {
		t.Fatalf("the taken-off %#04x has no bag form", unit.Code)
	}
	secondShopWitness(t, app, "town1-shop-doll-taken-off")
	drag(secondDollReleasePack(t, app, s, unit.Code))
	if worn := mapload.MemberItemEquipment(f.Carried[0], f.Table); worn[slot-1].Code != unit.Code {
		t.Fatalf("the second wear left slot %d holding %#04x", slot, worn[slot-1].Code)
	}
	secondDollReleaseCold(t, f, app, "Doll", unit.Code, slot, held)
}

// TestReleaseSecondShopDoubleClickWearsTheItem opens town 1's shop and
// double-clicks bought items in the bag until one is worn: the unit is on
// the hero, not lost, and no gold or stock moves; a unit his class may not
// use stays offered on the table and UNDO returns it. A town SAVE and a
// fresh LOAD keep the worn unit.
func TestReleaseSecondShopDoubleClickWearsTheItem(t *testing.T) {
	secondGameRoot(t)
	f, app, s, _ := secondShopTownOne(t)
	secondShopEnter(t, f, app, s)
	c := f.Town.second
	f.Town.gold = 5000000
	double := func(at ui.ShopControl) {
		p := secondShopPagePoint(t, app, "pack", at.Index)
		for _, edge := range []string{"press", "release", "press", "release"} {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	unit, slot := secondDollReleaseWear(t, f, app, s, double)
	secondShopWitness(t, app, "town1-shop-double-click-worn")
	secondDollReleaseCold(t, f, app, "DoubleClick", unit.Code, slot, secondDollReleaseCount(c, unit.Code))
}

// secondDollReleaseBagForm reports whether every bag unit with code has the
// bag form a SAVE writes.
func secondDollReleaseBagForm(bag []sim.ItemInstance, code uint16) bool {
	for _, in := range bag {
		if _, ok := secondshop.FromInstance(in); in.Code == code && !ok {
			return false
		}
	}
	return true
}
