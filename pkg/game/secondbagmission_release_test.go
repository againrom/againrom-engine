package game

import (
	"fmt"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// secondBagStage reports what the hero holds at one stage of the bag's
// route: the campaign's bag, his pack and his worn items, each as
// secondBagForm names them.
type secondBagStage func(stage string, bag, pack, worn []string)

// secondBagForm names items as bagLabels does, each in the bag's form where
// it has one at its own price, so a mission's type byte, potion payload,
// equipment, weight and object binding do not count.
func secondBagForm(items []sim.ItemInstance) []string {
	var out []sim.ItemInstance
	for _, it := range items {
		it = it.Clone()
		it.ObjectID, it.NativeRecord = 0, nil
		unit := secondshop.Unit(it)
		if _, ok := secondshop.FromInstance(unit); ok && unit.Price == it.Price {
			it = unit
		}
		out = append(out, it)
	}
	return bagLabels(out)
}

// secondBagHeldForm is the hero's pack and worn items in w, in the bag's
// item form.
func secondBagHeldForm(w *sim.World, id sim.EntityID) (pack, worn []string) {
	items, _ := w.CarriedItems(id)
	eq, _ := w.EquippedItems(id)
	return secondBagForm(items), secondBagForm(eq[:])
}

// secondBagTown is the campaign hero's pack and worn items in town, in the
// bag's item form.
func secondBagTown(f *FrontEnd) (pack, worn []string) {
	eq := mapload.MemberItemEquipment(f.Carried[0], f.Table)
	return secondBagForm(mapload.MemberCarriedItems(f.Carried[0], f.Table)), secondBagForm(eq[:])
}

// secondBagHolds reports whether have holds every label of want, counted.
func secondBagHolds(have, want []string) bool {
	left := slices.Clone(have)
	for _, w := range want {
		i := slices.Index(left, w)
		if i < 0 {
			return false
		}
		left = slices.Delete(left, i, i+1)
	}
	return true
}

// secondBagBuyAndEnter starts a new campaign, talks in the inn, buys the
// first entry of every shop category with units, leaves through the gates
// and enters mission 10 from the map. It answers the bought units, and
// fails unless the mission hero wears what he wore in town and packs every
// bought unit.
func secondBagBuyAndEnter(t *testing.T, report secondBagStage) (*FrontEnd, *ui.App, []string) {
	t.Helper()
	f, app := secondTownNew(t, false, true)
	secondShopActivate(t, f, app, "SHOP")
	c := f.Town.second
	f.Town.gold = 100000
	for k := 1; k <= len(secondshop.Record{}); k++ {
		secondShopActivate(t, f, app, fmt.Sprintf("SHELF %d", k))
		if len(c.shop.visibleShelf()) != 0 {
			secondShopActivate(t, f, app, "ITEM 1", "BUY")
		}
	}
	bag := secondBagForm(*c.heroBag())
	if len(bag) == 0 {
		t.Fatal("the shop sold nothing")
	}
	pack, worn := secondBagTown(f)
	report("town 1 after BUY", bag, pack, worn)
	secondShopActivate(t, f, app, "EXIT")
	for _, target := range []string{"GATES", "mission 10"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
	}
	if app.Screen() != ui.ScreenMap || f.live == nil || f.liveMission != 10 {
		t.Fatalf("mission 10 did not open: %v %q", app.Screen(), app.HeadlessMessage())
	}
	held, wearing := secondBagHeldForm(f.live.world, f.live.mission.ids[0])
	report("mission 10 first frame", secondBagForm(*c.heroBag()), held, wearing)
	if !secondBagHolds(held, bag) || !secondBagHolds(held, pack) || !slices.Equal(wearing, worn) {
		t.Fatalf("mission hero packs %v and wears %v; bought %v, town pack %v, worn %v", held, wearing, bag, pack, worn)
	}
	if left := secondBagForm(*c.heroBag()); len(left) != 0 {
		t.Fatalf("the bag kept %v in the mission", left)
	}
	return f, app, bag
}

// secondBagTicks steps the mission n times.
func secondBagTicks(t *testing.T, app *ui.App, n int) {
	t.Helper()
	for range n {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
}

// secondBagUseInMission wears a bought armour when one was bought, drops a
// bought unit beside the hero and picks it up again through the mission's
// commands.
func secondBagUseInMission(t *testing.T, f *FrontEnd, app *ui.App, bag []string) {
	t.Helper()
	secondBagTicks(t, app, 20)
	dismissSecondMissionDialogue(t, app, f.live)
	w, id := f.live.world, f.live.mission.ids[0]
	bought := func(s sim.ItemStack) bool {
		return slices.Contains(bag, secondBagForm([]sim.ItemInstance{s.Instance()})[0])
	}
	stacks, _ := w.CarriedStacks(id)
	for i, s := range stacks {
		// The equipment slot is the item's part, counted from 1.
		if part := int(s.Code >> 8 & 0xf); s.Kind == 1 && part >= 2 && part <= sim.EquipSlots && bought(s) {
			f.live.pending = append(f.live.pending, sim.Equip(id, sim.ItemSlot(i), sim.EquipSlot(part)))
			secondBagTicks(t, app, 8)
			if eq, _ := w.EquippedItems(id); !slices.ContainsFunc(eq[:], func(it sim.ItemInstance) bool { return it.Code == s.Code }) {
				t.Fatalf("the bought armour %#04x was not worn", s.Code)
			}
			break
		}
	}
	stacks, _ = w.CarriedStacks(id)
	i := slices.IndexFunc(stacks, bought)
	if i < 0 {
		return
	}
	units := func() int { items, _ := w.CarriedItems(id); return len(items) }
	before := units()
	hero, _ := w.Entity(id)
	at, ok := w.DropLanding(id, hero.X+1, hero.Y)
	if !ok {
		t.Fatal("no cell beside the hero takes a drop")
	}
	f.live.pending = append(f.live.pending, sim.DropCarried(id, sim.ItemSlot(i), at))
	secondBagTicks(t, app, 8)
	if len(w.Sacks()) == 0 || units() != before-1 {
		t.Fatalf("the bought unit was not dropped: %d sacks, %d units of %d", len(w.Sacks()), units(), before)
	}
	f.live.pending = append(f.live.pending, sim.PickUp(id, at))
	for range 400 {
		if units() == before {
			break
		}
		secondBagTicks(t, app, 1)
	}
	if units() != before {
		t.Fatalf("the dropped unit was not picked up: %d units, want %d", units(), before)
	}
}

// secondBagSaveAndLoad SAVEs mission 10 and cold-LOADs it in a new front
// end, failing unless the hero holds the same pack and worn items.
func secondBagSaveAndLoad(t *testing.T, f *FrontEnd, app *ui.App, report secondBagStage) {
	t.Helper()
	secondBagTicks(t, app, 20)
	dismissSecondMissionDialogue(t, app, f.live)
	pack, worn := secondBagHeldForm(f.live.world, f.live.mission.ids[0])
	dir := secondMissionSaveDirectory(t)
	secondMissionNamedSave(t, f, app, dir, "Bag")
	cold, _ := secondMissionCold(t, dir, secondSaveFile(t, dir, "Bag"))
	got, gotWorn := secondBagHeldForm(cold.live.world, cold.live.mission.ids[0])
	report("mission 10 after SAVE and cold LOAD", secondBagForm(*cold.Town.second.heroBag()), got, gotWorn)
	if !slices.Equal(got, pack) || !slices.Equal(gotWorn, worn) {
		t.Fatalf("cold LOAD packs %v and wears %v, want %v and %v", got, gotWorn, pack, worn)
	}
}

// secondBagComeHome wins mission 10 by its route, which enters mission 20
// with the pack, wins mission 20 by a script instant on a copy of its
// world, arrives at town 2 from the map and sells a bought unit there.
func secondBagComeHome(t *testing.T, f *FrontEnd, app *ui.App, bag []string, report secondBagStage) {
	t.Helper()
	enterSecondCampaignNextMission(t, f, app, false)
	c := f.Town.second
	pack, worn := secondBagHeldForm(f.live.world, f.live.mission.ids[0])
	report("mission 20 after the victory", secondBagForm(*c.heroBag()), pack, worn)
	// A bought unit the mission put on is worn, not packed.
	if !secondBagHolds(append(slices.Clone(pack), worn...), bag) {
		t.Fatalf("mission 20 hero packs %v and wears %v, bought %v", pack, worn, bag)
	}
	secondBagWin(t, f, 20)
	(&destinationTownMap{f.TownScreen().(*townScreen)}).enterLocation(partyPlace{kind: placeTown, id: 2})
	home := secondBagForm(*c.heroBag())
	townPack, townWorn := secondBagTown(f)
	report("town 2 after the return", home, townPack, townWorn)
	left := slices.ContainsFunc(townPack, func(l string) bool { return slices.Contains(bag, l) })
	if !secondBagHolds(append(slices.Clone(home), townWorn...), bag) || !slices.Equal(townWorn, worn) || left {
		t.Fatalf("town 2: bag %v pack %v worn %v; bought %v, worn in mission %v", home, townPack, townWorn, bag, worn)
	}
	tables, ok := secondShopTables(f.Table)
	if !ok {
		t.Fatal("the install has no shop tables")
	}
	c.shop.filled, c.shop.tables = 2, tables
	gold, units := f.Town.gold, len(*c.heroBag())
	if !c.shop.offer(c.heroBag(), 0) || !c.shop.sell(c.heroBag(), &f.Town.gold) || len(*c.heroBag()) >= units || f.Town.gold < gold {
		t.Fatalf("the sale at town 2: gold %d -> %d, units %d -> %d", gold, f.Town.gold, units, len(*c.heroBag()))
	}
}

// secondBagHealing is the health payload potion code has to the engine's
// item decoder, or false when the code is no usable potion that heals at
// once.
func secondBagHealing(code uint16, table *mapload.Table) (int32, bool) {
	it := mapload.ItemInstanceFromCode(code, table)
	if !sim.UsablePotion(it) {
		return 0, false
	}
	for _, e := range it.Effects {
		if e.Kind == 6 && e.Mode == 0 {
			return int32(e.Operand), true
		}
	}
	return 0, false
}

// A healing potion bought in town 1 is drunk in mission 10 through the
// mission pack's own use: the unit is used up and the hero's health rises.
func TestReleaseSecondBagPotionIsDrunkInMissionTen(t *testing.T) {
	f, app := secondTownNew(t, false, true)
	secondShopActivate(t, f, app, "SHOP")
	c := f.Town.second
	f.Town.gold = 100000
	var code uint16
	var heal int32
	// The shop's own take and buy, as the shelf's entry may lie past the
	// rows the page shows.
	for k := range c.shop.shelves {
		c.shop.choose(k)
		for i, it := range c.shop.shelf() {
			u, ok := c.shop.tables.Instance(it)
			if h, potion := secondBagHealing(u.Code, f.Table); ok && potion && it.Count > 0 && code == 0 {
				if !c.shop.take(i) || !c.shop.buy(c.heroBag(), &f.Town.gold) {
					t.Fatalf("the shop did not sell potion %#04x", u.Code)
				}
				code, heal = u.Code, h
			}
		}
	}
	if code == 0 {
		t.Fatal("town 1's shop sells no healing potion")
	}
	if !slices.ContainsFunc(*c.heroBag(), func(it sim.ItemInstance) bool { return it.Code == code }) {
		t.Fatalf("the bag holds no %#04x after BUY", code)
	}
	secondShopActivate(t, f, app, "EXIT")
	for _, target := range []string{"GATES", "mission 10"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
	}
	if app.Screen() != ui.ScreenMap || f.live == nil || f.liveMission != 10 {
		t.Fatalf("mission 10 did not open: %v %q", app.Screen(), app.HeadlessMessage())
	}
	secondBagTicks(t, app, 20)
	dismissSecondMissionDialogue(t, app, f.live)
	secondBagDrink(t, f.live, code, heal, func() { secondBagTicks(t, app, 1) })
}

func secondBagLog(t *testing.T) secondBagStage {
	return func(stage string, bag, pack, worn []string) {
		t.Logf("%s: bag %v pack %v worn %v", stage, bag, pack, worn)
	}
}

// What the hero buys in town 1 enters mission 10 with him, worn and packed,
// and the mission's commands wear, drop and pick it up.
func TestReleaseSecondBagEntersMissionTen(t *testing.T) {
	f, app, bag := secondBagBuyAndEnter(t, secondBagLog(t))
	secondBagUseInMission(t, f, app, bag)
}

// A mission SAVE and its cold LOAD keep the bought units in the pack and
// the worn items worn.
func TestReleaseSecondBagMissionSaveAndColdLoad(t *testing.T) {
	f, app, _ := secondBagBuyAndEnter(t, secondBagLog(t))
	secondBagSaveAndLoad(t, f, app, secondBagLog(t))
}

// The victory carries the bought units on, and the arrival at a town puts
// them back in the bag, where the shop sells them.
func TestReleaseSecondBagComesHomeToTheBag(t *testing.T) {
	f, app, bag := secondBagBuyAndEnter(t, secondBagLog(t))
	secondBagComeHome(t, f, app, bag, secondBagLog(t))
}
