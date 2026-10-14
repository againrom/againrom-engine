package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// packJoinSpear is the Spear 0x112f the merchant sells and the mission-50
// spearmen wear.
const packJoinSpear = 0x112f

// packJoinCells is the spear cells of hero's pack as "x<count>", in pack
// order.
func packJoinCells(f *FrontEnd, hero sim.EntityID) []string {
	stacks, _ := f.live.world.CarriedStacks(hero)
	var out []string
	for _, st := range stacks {
		if st.Code == packJoinSpear {
			out = append(out, fmt.Sprintf("x%d", st.Count))
		}
	}
	return out
}

// packJoinSackOf is the Sack lying on at, which holds a spear.
func packJoinSackOf(t *testing.T, f *FrontEnd, at sim.CellPoint) sim.Sack {
	t.Helper()
	s := groundAt(f.live.world.Sacks(), at.X, at.Y)
	if s == nil || !slices.ContainsFunc(s.ItemInstances, func(it sim.ItemInstance) bool { return it.Code == packJoinSpear }) {
		t.Fatalf("no Sack holding the spear lies at (%d,%d)", at.X, at.Y)
	}
	return *s
}

// packJoinTake walks hero beside the Sack on at and picks it up.
func packJoinTake(t *testing.T, f *FrontEnd, hero sim.EntityID, at sim.CellPoint) {
	t.Helper()
	livePlaceAndWalk(t, f.live, hero, at.X+1, at.Y)
	packStackPickUp(t, f, hero, at.X, at.Y)
}

// TestReleaseBoughtSpearJoinsTheLootedSpears: with mission 50 taken after
// controlled prior victories, the hero buys a Spear 0x112f, which mission
// entry binds as a saved object of its plain form. Mission 50's spearmen wear
// the spear its constructor builds; both are one item (ITEM-STACK-003,
// DIV-762). A spear looted from a corpse joins the bought one. The hero drops
// both units, saved objects now, loots two more into a new cell and picks his
// own back: they join it. One cell of four survives a mission SAVE, which
// writes one Weapon record of count 4, and a cold LOAD.
func TestReleaseBoughtSpearJoinsTheLootedSpears(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	prepareAcceptedCampaignMission(t, f, 50)
	s := currentTownShop(f, "hero")
	f.Town.gold = 100000
	for n := 0; n < 30; n++ {
		s.chooseRoomShelf(roomWeapons)
		if slices.ContainsFunc(s.sess.Shop.Shelf(s.shopShelf), func(it ShopItem) bool { return it.Code == packJoinSpear }) {
			break
		}
		s.restockShop()
	}
	currentTownBuy(t, s, roomWeapons, packJoinSpear)
	if err := f.App("bought spear").OpenMission(f.MissionOpenerWith(50, f.NextParty())); err != nil {
		t.Fatalf("open mission 50: %v", err)
	}
	hero := equipmentReturnHero(t, f)
	w := f.live.world
	stacks, _ := w.CarriedStacks(hero)
	bought := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == packJoinSpear })
	if bought < 0 || stacks[bought].ObjectID == 0 || stacks[bought].WeightPresent {
		t.Fatalf("mission entry spear cells %v; want the bought spear as a plain saved object", packJoinCells(f, hero))
	}
	h, _ := w.Entity(hero)
	var fell []sim.CellPoint
	for _, e := range w.Entities() {
		if worn, _ := w.EquippedItems(e.ID); e.Owner != h.Owner && worn[0].Code == packJoinSpear && worn[0].SourceEquipment.Class != 0 {
			f.live.pending = append(f.live.pending, sim.Kill(e.ID))
			fell = append(fell, sim.CellPoint{X: e.X, Y: e.Y})
		}
	}
	if len(fell) < 3 {
		t.Fatalf("mission 50 has %d spearmen wearing the constructed spear; want 3", len(fell))
	}
	fell = fell[:3]
	for tick := 0; ; tick++ {
		f.live.tick()
		lying := 0
		for _, at := range fell {
			if s := groundAt(w.Sacks(), at.X, at.Y); s != nil && slices.ContainsFunc(s.ItemInstances, func(it sim.ItemInstance) bool { return it.Code == packJoinSpear }) {
				lying++
			}
		}
		if lying == len(fell) {
			break
		}
		if tick > 6000 {
			t.Fatalf("after %d ticks %d of the spearmen's Sacks lie", tick, lying)
		}
	}
	for _, at := range fell {
		if loot := packJoinSackOf(t, f, at).ItemInstances; loot[slices.IndexFunc(loot, func(it sim.ItemInstance) bool { return it.Code == packJoinSpear })].ObjectID != 0 {
			t.Fatalf("the spear at (%d,%d) is a saved object; want the corpse's own value", at.X, at.Y)
		}
	}
	packJoinTake(t, f, hero, fell[0])
	if got := packJoinCells(f, hero); !slices.Equal(got, []string{"x2"}) {
		t.Fatalf("after looting one spear the pack holds spear cells %v; want [x2]", got)
	}
	// A drop leaves one unit at a time; two empty the cell.
	h, _ = w.Entity(hero)
	dropped := sim.CellPoint{X: h.X, Y: h.Y}
	for len(packJoinCells(f, hero)) != 0 {
		stacks, _ = w.CarriedStacks(hero)
		slot := slices.IndexFunc(stacks, func(st sim.ItemStack) bool { return st.Code == packJoinSpear })
		f.live.pending = append(f.live.pending, sim.DropCarried(hero, sim.ItemSlot(slot), dropped))
		f.live.tick()
	}
	if loot := packJoinSackOf(t, f, dropped).ItemInstances; len(loot) != 2 || slices.ContainsFunc(loot, func(it sim.ItemInstance) bool { return it.ObjectID == 0 || it.WeightPresent }) {
		t.Fatalf("the dropped Sack holds %+v; want the two spears as plain saved objects", loot)
	}
	packJoinTake(t, f, hero, fell[1])
	packJoinTake(t, f, hero, fell[2])
	stacks, _ = w.CarriedStacks(hero)
	if got := packJoinCells(f, hero); !slices.Equal(got, []string{"x2"}) || !slices.ContainsFunc(stacks, func(st sim.ItemStack) bool {
		return st.Code == packJoinSpear && st.ObjectID != 0 && st.WeightPresent && st.SourceEquipment.Class != 0
	}) {
		t.Fatalf("after looting two more spears the pack holds spear cells %v; want one constructed saved object x2", got)
	}
	packJoinTake(t, f, hero, dropped)
	check := func(label string, f *FrontEnd) {
		t.Helper()
		if got := packJoinCells(f, equipmentReturnHero(t, f)); !slices.Equal(got, []string{"x4"}) {
			t.Fatalf("%s: the pack holds spear cells %v; want [x4]", label, got)
		}
	}
	check("after the own spear is picked up", f)
	path := saveCorpseMission(t, f, t.TempDir())
	saved := packStackSavedItems(t, path, packJoinSpear)
	if len(saved) != 1 || !strings.HasPrefix(saved[0], "x4 ") {
		t.Fatalf("mission SAVE writes 0x112f records %v; want one of count 4", saved)
	}
	check("cold LOAD", loadAreaContinuation(t, path))
}
