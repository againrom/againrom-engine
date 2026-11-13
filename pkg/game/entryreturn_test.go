package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// TestAbandonReturnsEveryGameToItsEntryState drives the one return of an
// abandoned mission over each game's item-home row. The party enters with a
// known party, backpack and bag, picks an item up, drops one and drinks one,
// and abandons; the party, each backpack slot, each worn slot, the bag and
// the town place equal those at entry. A row with no bag and a row with one
// must both run.
func TestAbandonReturnsEveryGameToItsEntryState(t *testing.T) {
	rows := []struct {
		name   string
		second bool
	}{{"first game", false}, {"second game", true}}
	ran := map[bool]bool{}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			f, a, screen, n := abandonBaseTown(t, row.second)
			if !row.second {
				f.Campaign = resolved(saveCampaign(), nil)
				if _, town, err := f.RestoreOriginal(currentTownSave(t, currentTrainingCity(t))); err != nil || !town {
					t.Fatalf("entry town: town=%t error=%v", town, err)
				}
				f.Archives.Containers = poolFixtureFrontMap(t, synth.ALM(synth.ALMOptions{Width: 128, Height: 128})).Archives.Containers
				n = 10
				a = f.App("entry return")
				a.Layout(1024, 768)
				screen = f.TownScreen().(*townScreen)
			}
			secondBagPotionTable(f)
			potion := mapload.ItemInstanceFromCode(secondBagPotion.Code, f.Table)
			home := f.partyItems(f.Table)
			if bagged := home.profile.bagUnits > 0; bagged != row.second {
				t.Fatalf("the session took the item-home row with %d bag units", home.profile.bagUnits)
			}
			ran[row.second] = true
			stacks := slices.Insert(sim.FoldItems([]sim.ItemInstance{secondBagWeapon, potion, potion, secondBagArmour}), 1, sim.ItemStack{})
			if err := home.writeShopStacks(0, stacks, true); err != nil {
				t.Fatal(err)
			}
			var bag []sim.ItemInstance
			var town partyPlace
			if row.second {
				f.Carried[0], _ = memberWithPack(f.Carried[0], []sim.ItemInstance{potion}, f.Table, true)
				bag, town = slices.Clone(*f.Town.second.heroBag()), f.Town.second.current
			}
			party := mapload.CloneParty(f.Carried)
			if row.second {
				secondBagEnter(t, f, a, screen)
				if len(*f.Town.second.heroBag()) != 0 {
					t.Fatal("the departure left the bag in town, so the return has nothing to put back")
				}
			} else if err := a.OpenMission(f.MissionOpener(n)); err != nil {
				t.Fatal(err)
			}
			mw := f.live
			id := mw.mission.ids[0]
			actor, _ := mw.world.Entity(id)
			at := sim.CellPoint{X: actor.X + 1, Y: actor.Y}
			found := sim.ItemInstance{Code: 0x1303, Kind: 1, Price: 77}
			if !mw.world.CheatAddItem(id, sim.StackItem(found, 1)) {
				t.Fatal("could not construct the mission loot")
			}
			loot, _ := mw.world.CarriedStacks(id)
			lootSlot := slices.IndexFunc(loot, func(s sim.ItemStack) bool { return s.Code == found.Code })
			mw.pending = append(mw.pending, sim.DropCarriedStack(id, sim.ItemSlot(lootSlot), at))
			for range 32 {
				mw.tick()
			}
			mw.pending = append(mw.pending, sim.PickUp(id, at))
			for range 400 {
				mw.tick()
			}
			picked, _ := mw.world.CarriedItems(id)
			if !slices.ContainsFunc(picked, func(it sim.ItemInstance) bool { return it.Code == found.Code }) {
				t.Fatal("the pickup control did not gain the ground item")
			}
			held, _ := mw.world.CarriedStacks(id)
			slot := slices.IndexFunc(held, func(s sim.ItemStack) bool { return s.Code == secondBagWeapon.Code })
			if slot < 0 {
				t.Fatal("the drop control has no weapon")
			}
			actor, _ = mw.world.Entity(id)
			mw.pending = append(mw.pending, sim.DropCarriedStack(id, sim.ItemSlot(slot), sim.CellPoint{X: actor.X - 1, Y: actor.Y}))
			for range 32 {
				mw.tick()
			}
			dropped, _ := mw.world.CarriedItems(id)
			if slices.ContainsFunc(dropped, func(it sim.ItemInstance) bool { return it.Code == secondBagWeapon.Code }) {
				t.Fatal("the drop control kept the weapon")
			}
			secondBagDrink(t, mw, potion.Code, 30, func() { mw.tick() })
			if dest, line, _ := f.continuity(n, mw.mission.state, mw.advanceNotice)(ui.NoticeAbandon); dest != ui.NoticeToTown {
				t.Fatalf("abandon: destination=%v message=%q", dest, line)
			}
			for range 1000 {
				if !screen.AtWorldMap() {
					break
				}
				screen.WorldMapTick()
			}
			requireAbandonItems(t, f, party, bag)
			if !reflect.DeepEqual(f.Carried, party) {
				campaignFlowValueDiff(t, "party", party, f.Carried)
				t.Fatal("the abandon did not return the party as it entered")
			}
			if row.second && f.Town.second.current != town || !row.second && !f.Town.Open() {
				t.Fatal("the abandon did not return to the town the mission was entered from")
			}
			if f.Town.Done(n) {
				t.Fatalf("the abandon completed mission %d", n)
			}
		})
	}
	if !ran[false] || !ran[true] {
		t.Fatalf("rows ran without a bag %t, with a bag %t; both games must run", ran[false], ran[true])
	}
}
