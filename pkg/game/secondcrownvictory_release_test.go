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

// secondCrownRoute is how one of missions 70 and 80 gives the Gold Crown
// when it is the second of the two played: the bank slot that says the
// other was played, the parts the hero brings, the storage unit the
// script pours the crown from, and the parts its script takes.
type secondCrownRoute struct {
	mission int
	played  int
	brings  []uint16
	storage uint16
	taken   []uint16
	give    func(t *testing.T, app *ui.App, live *mapWorld, units map[uint16]sim.EntityID)
}

// secondCrownHeld names where the hero holds the Gold Crown: "pack",
// "worn", "bag" or "", with the unit.
func secondCrownHeld(pack []sim.ItemInstance, worn [sim.EquipSlots]sim.ItemInstance, bag []sim.ItemInstance) (string, sim.ItemInstance, int) {
	where, unit, n := "", sim.ItemInstance{}, 0
	for _, place := range []struct {
		name  string
		items []sim.ItemInstance
	}{{"pack", pack}, {"worn", worn[:]}, {"bag", bag}} {
		for _, it := range place.items {
			if it.Code == secondGoldCrown {
				where, unit, n = place.name, it, n+1
			}
		}
	}
	return where, unit, n
}

// secondCrownCheck fails unless the hero holds the Gold Crown exactly once
// with the storage unit's effects, and reports where.
func secondCrownCheck(t *testing.T, stage string, want sim.ItemInstance, pack []sim.ItemInstance, worn [sim.EquipSlots]sim.ItemInstance, bag []sim.ItemInstance) string {
	t.Helper()
	where, got, n := secondCrownHeld(pack, worn, bag)
	t.Logf("%s: crown %s %+v; pack %s worn %s bag %s", stage, where, got.Effects, codesOf(pack), codesOf(worn[:]), codesOf(bag))
	if n != 1 || !slices.Equal(got.Effects, want.Effects) {
		t.Fatalf("%s: the hero holds %d Gold Crowns, at %q with effects %+v; want one with %+v", stage, n, where, got.Effects, want.Effects)
	}
	return where
}

func codesOf(items []sim.ItemInstance) string {
	var out []string
	for _, it := range items {
		if it.Code != 0 {
			out = append(out, fmt.Sprintf("%#04x", it.Code))
		}
	}
	return fmt.Sprint(out)
}

// secondCrownNear places the hero beside unit id.
func secondCrownNear(t *testing.T, w *sim.World, hero, id sim.EntityID) {
	t.Helper()
	e, ok := w.Entity(id)
	if !ok {
		t.Fatal("no unit", id)
	}
	if err := w.HeadlessPlace(hero, e.X+1, e.Y); err != nil {
		if err := w.HeadlessPlace(hero, e.X-1, e.Y); err != nil {
			t.Fatal(err)
		}
	}
}

// secondCrownSeventyGive is mission 70's route to the elf's offer: the hero
// stands at the elf (unit 54), group 37 falls, and the hero is at the elf
// again, so register 74 names him.
func secondCrownSeventyGive(t *testing.T, app *ui.App, live *mapWorld, units map[uint16]sim.EntityID) {
	t.Helper()
	w, hero := live.world, live.mission.ids[0]
	secondCrownNear(t, w, hero, units[54])
	for _, e := range w.Entities() {
		if e.Group == 37 && e.Alive() {
			if err := w.HeadlessKill(e.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	for tick := 0; tick < 600 && w.ScriptRegister(74) == 0; tick++ {
		secondCrownNear(t, w, hero, units[54])
		secondBagTicks(t, app, 1)
		dismissSecondMissionDialogue(t, app, live)
	}
	if w.ScriptRegister(74) != 1 {
		t.Fatalf("mission 70: register 74 is %d, registers 70 %d 73 %d 76 %d", w.ScriptRegister(74), w.ScriptRegister(70), w.ScriptRegister(73), w.ScriptRegister(76))
	}
}

// secondCrownEightyGive is mission 80's route: the shaman (unit 38) falls
// and the hero picks crown13 out of his sack.
func secondCrownEightyGive(t *testing.T, app *ui.App, live *mapWorld, units map[uint16]sim.EntityID) {
	t.Helper()
	w, hero := live.world, live.mission.ids[0]
	if err := w.HeadlessKill(units[38]); err != nil {
		t.Fatal(err)
	}
	var sx, sy int32
	found := false
	for i := 0; i < 60 && !found; i++ {
		secondBagTicks(t, app, 1)
		dismissSecondMissionDialogue(t, app, live)
		for _, s := range w.Sacks() {
			if slices.Contains(s.Items, secondCrownThirteen) {
				sx, sy, found = s.X, s.Y, true
			}
		}
	}
	if !found {
		t.Fatal("mission 80: the shaman dropped no sack with crown13")
	}
	if err := w.HeadlessPlace(hero, sx+1, sy); err != nil {
		t.Fatal(err)
	}
	live.orderPickup(hero, sx, sy)
	for i := 0; i < 400; i++ {
		if c, _ := w.Carried(hero); slices.Contains(c, secondCrownThirteen) {
			return
		}
		if c, _ := w.Carried(hero); slices.Contains(c, secondGoldCrown) {
			return
		}
		secondBagTicks(t, app, 1)
		dismissSecondMissionDialogue(t, app, live)
	}
	t.Fatal("mission 80: the hero did not pick crown13 up")
}

var secondCrownRoutes = map[int]secondCrownRoute{
	70: {mission: 70, played: 778, brings: []uint16{secondCrownOne, secondCrownThirteen}, storage: 207, taken: []uint16{secondCrownOne, secondCrownThirteen}, give: secondCrownSeventyGive},
	80: {mission: 80, played: 777, brings: []uint16{secondCrownOne, secondCrownTwelve}, storage: 160, taken: []uint16{secondCrownOne, secondCrownTwelve, secondCrownThirteen}, give: secondCrownEightyGive},
}

// secondCrownVictory plays installed mission n as the second of missions 70
// and 80: the hero brings the other mission's parts, the script pours the
// Gold Crown from its storage unit to him, takes the parts and completes the
// mission. When worn, the hero continues after the success notice, puts the
// crown on and ends the quest. The victory carries him home; he enters the
// first town the map offers, SAVEs there and the SAV is cold-LOADed. At each
// stage the hero holds the crown once, with the storage unit's effects.
func secondCrownVictory(t *testing.T, n int, worn bool) {
	route := secondCrownRoutes[n]
	f := secondGameFront(t)
	app := f.App("crown victory")
	app.Layout(1024, 768)
	secondLaterEntryWith(t, f, app, n, int32(n), func(c *secondCampaign) {
		c.bank[route.played] = 1
		pack := mapload.MemberCarriedItems(f.Carried[0], f.Table)
		for _, code := range route.brings {
			pack = append(pack, mapload.ItemInstanceFromCode(code, f.Table))
		}
		hero, ok := memberWithPack(f.Carried[0], pack, f.Table, true)
		if !ok {
			t.Fatal("the hero's pack refused the crown parts")
		}
		f.Carried[0] = hero
	})
	live := f.live
	w, hero := live.world, live.mission.ids[0]
	units := mapload.ScriptUnits(live.mission.state.Map, live.mission.party)
	storage, ok := units[route.storage]
	if !ok {
		t.Fatal("no storage unit", route.storage)
	}
	if e, _ := w.Entity(storage); !e.OffMap {
		t.Fatal("the storage unit is on the map")
	}
	stored, _ := w.CarriedItems(storage)
	if len(stored) != 1 || stored[0].Code != secondGoldCrown || len(stored[0].Effects) == 0 {
		t.Fatalf("storage unit %d holds %+v", route.storage, stored)
	}
	want := stored[0]
	t.Logf("mission %d storage %d holds the crown: kind %d price %d effects %+v object %d", n, route.storage, want.Kind, want.Price, want.Effects, want.ObjectID)
	route.give(t, app, live, units)
	given := false
	for tick := 0; tick < 600 && !(live.mission.open && live.mission.kind == ui.NoticeSuccess); tick++ {
		if c, _ := w.Carried(hero); !given && slices.Contains(c, secondGoldCrown) {
			given = true
			pack, _ := w.CarriedItems(hero)
			eq, _ := w.EquippedItems(hero)
			secondCrownCheck(t, "after the pour", want, pack, eq, nil)
			if left, _ := w.CarriedItems(storage); len(left) != 0 {
				t.Fatalf("the storage unit keeps %+v", left)
			}
		}
		secondBagTicks(t, app, 1)
		dismissSecondMissionDialogue(t, app, live)
	}
	if won, _ := w.ScriptCounters(); w.Outcome() != sim.OutcomeWon || won == 0 || !live.mission.open || live.mission.kind != ui.NoticeSuccess {
		t.Fatalf("mission %d: outcome %v wins %d", n, w.Outcome(), won)
	}
	pack, _ := w.CarriedItems(hero)
	eq, _ := w.EquippedItems(hero)
	secondCrownCheck(t, "at the success notice", want, pack, eq, nil)
	for _, code := range route.taken {
		if slices.ContainsFunc(pack, func(it sim.ItemInstance) bool { return it.Code == code }) {
			t.Fatalf("the script left part %#04x in the hero's pack", code)
		}
	}
	if worn {
		if err := app.HeadlessActivate("continue"); err != nil {
			t.Fatal("Continue:", err)
		}
		stacks, _ := w.CarriedStacks(hero)
		i := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == secondGoldCrown })
		live.pending = append(live.pending, sim.Equip(hero, sim.ItemSlot(i), sim.EquipSlot(secondGoldCrown>>8&0xf)))
		secondBagTicks(t, app, 8)
		pack, _ = w.CarriedItems(hero)
		eq, _ = w.EquippedItems(hero)
		if where := secondCrownCheck(t, "after the crown is put on", want, pack, eq, nil); where != "worn" {
			t.Fatal("the hero did not put the crown on")
		}
		if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenGameMenu {
			t.Fatalf("game menu: %v on %s", err, app.Screen())
		}
		if err := app.HeadlessGameMenuAction("end"); err != nil {
			t.Fatal("End Quest:", err)
		}
		if err := app.HeadlessGameMenuAction("victory"); err != nil {
			t.Fatal("Victory:", err)
		}
	} else if err := app.HeadlessActivate("victory"); err != nil {
		t.Fatal("Victory:", err)
	}
	for range 4 {
		if app.Screen() != ui.ScreenCutscene {
			break
		}
		t.Logf("victory movie %s", app.CutsceneName())
		if err := app.HeadlessCutsceneStep("key"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown || f.live != nil {
		t.Fatalf("the victory did not reach the map: %v %q", app.Screen(), app.HeadlessMessage())
	}
	c := f.Town.second
	carried := func(stage string, f *FrontEnd) string {
		bag := *f.Town.second.heroBag()
		return secondCrownCheck(t, stage, want, mapload.MemberCarriedItems(f.Carried[0], f.Table), mapload.MemberItemEquipment(f.Carried[0], f.Table), bag)
	}
	carried("at the map after the victory", f)
	town := partyPlace{}
	for _, p := range c.available {
		if p.kind == placeTown && town == (partyPlace{}) {
			town = p
		}
	}
	t.Logf("available after the victory %v", c.available)
	if town == (partyPlace{}) {
		t.Fatal("the map offers no town")
	}
	if err := app.HeadlessActivate(secondLocationName(town)); err != nil || c.current != town {
		t.Fatalf("%s: %v at %v", secondLocationName(town), err, c.current)
	}
	inTown := carried(secondLocationName(town), f)
	if stacks, ok := secondshop.Stacks(*c.heroBag()); !ok || inTown == "bag" && !slices.ContainsFunc(stacks, func(it secondshop.Item) bool { return len(it.Effects) == len(want.Effects) }) {
		t.Fatalf("the shop's bag shows %+v (%v)", stacks, ok)
	}
	dir := secondMissionSaveDirectory(t)
	secondTownNamedSave(t, f, app, dir, "Crown")
	cold, _ := secondTownCold(t, dir, secondSaveFile(t, dir, "Crown"))
	if after := carried("after the town SAV and its cold LOAD", cold); after != inTown {
		t.Fatalf("the cold LOAD holds the crown at %q, the town at %q", after, inTown)
	}
	if want := map[bool]string{false: "bag", true: "worn"}[worn]; inTown != want {
		t.Errorf("in %s the Gold Crown is in the hero's %s, want the %s", secondLocationName(town), inTown, want)
	}
}

// TestReleaseSecondCrownComesHomeFromMissionSeventy plays installed mission
// 70 after mission 80 to its scripted victory with the Gold Crown in the
// pack and worn, and carries it to town and through a town SAV.
func TestReleaseSecondCrownComesHomeFromMissionSeventy(t *testing.T) {
	for _, worn := range []bool{false, true} {
		t.Run(fmt.Sprintf("worn=%v", worn), func(t *testing.T) { secondCrownVictory(t, 70, worn) })
	}
}

// TestReleaseSecondCrownComesHomeFromMissionEighty is the same over
// installed mission 80 after mission 70.
func TestReleaseSecondCrownComesHomeFromMissionEighty(t *testing.T) {
	for _, worn := range []bool{false, true} {
		t.Run(fmt.Sprintf("worn=%v", worn), func(t *testing.T) { secondCrownVictory(t, 80, worn) })
	}
}
