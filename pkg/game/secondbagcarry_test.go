package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/secondshop"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The second game's items a fixture hero holds: an armour and a weapon worn,
// an armour, a weapon and two potions bought into the bag.
var (
	secondBagWornArmour  = sim.ItemInstance{Code: 0x1301, Kind: 1, Price: 100}
	secondBagWornWeapon  = sim.ItemInstance{Code: 0x2103, Kind: 2, Price: 50, Effects: []sim.ItemEffect{{Kind: 41, Operand: 5 | 2<<16}}}
	secondBagArmour      = sim.ItemInstance{Code: 0x1402, Kind: 1, Price: 120}
	secondBagWeapon      = sim.ItemInstance{Code: 0x2104, Kind: 2, Price: 60}
	secondBagPotion      = sim.ItemInstance{Code: 0x0e01, Price: 20}
	secondBagIglezPack   = sim.ItemInstance{Code: 0x0e03, Price: 7}
	secondBagIglezArmour = sim.ItemInstance{Code: 0x1302, Kind: 1, Price: 90}
)

// bagLabels names units by ID, type, price and effects, sorted.
func bagLabels(items []sim.ItemInstance) []string {
	var out []string
	for _, it := range items {
		if it.Code != 0 {
			out = append(out, fmt.Sprintf("%#04x/%d/%d/%v", it.Code, it.Kind, it.Price, it.Effects))
		}
	}
	slices.Sort(out)
	return out
}

func secondBagWorn(items [sim.EquipSlots]sim.ItemInstance) []string { return bagLabels(items[:]) }

// secondBagFront is a second-game campaign in town 1 with its gate open and
// mission 10 offered: the hero wears an armour and a weapon and his bag holds
// bag (none for an empty bag); Iglez carries his own pack and armour.
func secondBagFront(t *testing.T, bag []sim.ItemInstance) (*FrontEnd, *ui.App, *townScreen, string) {
	t.Helper()
	// The map holds the fallback drop, so the party stands on it.
	f := secondMissionFrontOf(t, secondMissionContent{side: 128})
	app := f.App("second-bag")
	app.Layout(1024, 768)
	dir := t.TempDir()
	save, list, load := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	app.SetSaveSeams(save, list, load)
	c := newSecondCampaign()
	c.bank[769] = 1
	c.available = append(c.available, partyPlace{kind: placeMission, id: 10})
	if f.Town == nil {
		f.Town = NewTown(Campaign{})
	}
	f.Town.second = c
	f.Carried = MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	hero := &f.Carried[0]
	hero.WornItems[0], hero.Worn[0] = secondBagWornWeapon.Clone(), secondBagWornWeapon.Code
	hero.WornItems[2], hero.Worn[2] = secondBagWornArmour.Clone(), secondBagWornArmour.Code
	iglez := mapload.CloneParty(f.Carried[:1])[0]
	iglez.ID, iglez.Name, iglez.CompanionNPC, iglez.StartingHero = "npc:22", "Igles", 22, false
	iglez.CarriedItems, iglez.Carried = []sim.ItemInstance{secondBagIglezPack.Clone()}, nil
	iglez.WornItems, iglez.Worn = [sim.EquipSlots]sim.ItemInstance{}, [sim.EquipSlots]uint16{}
	iglez.WornItems[2], iglez.Worn[2] = secondBagIglezArmour.Clone(), secondBagIglezArmour.Code
	f.Carried = append(f.Carried, iglez)
	for _, it := range bag {
		*c.heroBag() = append(*c.heroBag(), it.Clone())
	}
	screen := f.TownScreen().(*townScreen)
	app.SetTown(screen)
	return f, app, screen, dir
}

// secondBagSample is the bought bag: an armour, a weapon and two potions.
func secondBagSample() []sim.ItemInstance {
	return []sim.ItemInstance{secondBagArmour, secondBagWeapon, secondBagPotion, secondBagPotion}
}

// secondBagEnter leaves town 1 through its gates and opens mission 10.
func secondBagEnter(t *testing.T, f *FrontEnd, app *ui.App, screen *townScreen) {
	t.Helper()
	destinationTownCampaign{screen}.leave()
	if err := app.OpenMission(f.MissionOpener(10)); err != nil || f.live == nil {
		t.Fatalf("mission 10: %v %s", err, app.HeadlessMessage())
	}
}

// secondBagEnterFromMap leaves town 1, puts bag in the campaign's bag at
// the map, as a LOAD at the map does, and enters mission 10 from the map.
func secondBagEnterFromMap(t *testing.T, f *FrontEnd, app *ui.App, screen *townScreen, bag []sim.ItemInstance) {
	t.Helper()
	destinationTownCampaign{screen}.leave()
	c := f.Town.second
	c.current = partyPlace{}
	for _, it := range bag {
		*c.heroBag() = append(*c.heroBag(), it.Clone())
	}
	at := (&destinationTownMap{screen}).enterLocation(partyPlace{kind: placeMission, id: 10})
	if at.Open == nil {
		t.Fatalf("the map opened no mission: %s", at.Msg)
	}
	if err := app.OpenMission(at.Open); err != nil || f.live == nil {
		t.Fatalf("mission 10: %v %s", err, app.HeadlessMessage())
	}
}

// secondBagWin wins mission n by a script instant on a copy of its world
// and applies the campaign's victory: the party comes home and stands on
// the map.
func secondBagWin(t *testing.T, f *FrontEnd, n int) {
	t.Helper()
	mw := f.live
	trigger := sim.ScriptTrigger{Instants: [4]int32{0, sim.ScriptNone, sim.ScriptNone, sim.ScriptNone}, Once: true}
	program, err := sim.NewROM2Script(nil, []sim.ScriptInstant{{Op: sim.ScriptInstantWin, Args: [10]int32{1}}}, []sim.ScriptTrigger{trigger})
	if err != nil {
		t.Fatal(err)
	}
	won, err := sim.NewControlledScriptWorld(mw.world, program)
	if err != nil {
		t.Fatal(err)
	}
	for range 400 {
		if won.Outcome() == sim.OutcomeWon {
			break
		}
		sim.Step(won, nil)
	}
	if won.Outcome() != sim.OutcomeWon {
		t.Fatal("the mission was not won")
	}
	ms := &Mission{Number: n, World: won, Party: mw.mission.party, Start: mapload.Start{IDs: mw.mission.ids}}
	if _, refused := (secondCampaignRules{}).finishWon(f, n, ms); refused != "" {
		t.Fatal(refused)
	}
	if c := f.Town.second; c.current != (partyPlace{}) {
		t.Fatalf("after the victory the party stands at %v", c.current)
	}
}

// secondBagHeld is each mission member's pack and worn set in w.
func secondBagHeld(w *sim.World, ids []sim.EntityID) (packs, worn [][]string) {
	for _, id := range ids {
		items, _ := w.CarriedItems(id)
		eq, _ := w.EquippedItems(id)
		packs, worn = append(packs, bagLabels(items)), append(worn, secondBagWorn(eq))
	}
	return packs, worn
}

// secondBagPotionTable gives f's table a MagicItems row 1, so 0x0e01 is a
// potion of health at price 20 to the engine's item decoder.
func secondBagPotionTable(f *FrontEnd) {
	f.Table.MagicItems = dbCollection{{}, {name: "Potion of Healing", params: []int32{20}, strings: []string{"health=30"}}}
}

// secondBagDrink uses the hero's first pack unit with code through the
// mission pack's own use, after a wound, and fails unless the unit is used
// up and the hero's health rises by at least heal, or the wound when it is
// smaller; health regeneration alone stays below either.
func secondBagDrink(t *testing.T, mw *mapWorld, code uint16, heal int32, tick func()) {
	t.Helper()
	w, id := mw.world, mw.mission.ids[0]
	hero, _ := w.Entity(id)
	wound := hero.HP / 2
	if err := w.HeadlessDamage(id, wound); err != nil {
		t.Fatal(err)
	}
	hurt, _ := w.Entity(id)
	stacks, _ := w.CarriedStacks(id)
	i := slices.IndexFunc(stacks, func(s sim.ItemStack) bool { return s.Code == code })
	if i < 0 {
		t.Fatalf("the mission pack holds no %#04x", code)
	}
	before, _ := w.CarriedItems(id)
	mw.enqueueEquip(i)
	for range 40 {
		tick()
	}
	after, _ := w.CarriedItems(id)
	healed, _ := w.Entity(id)
	if len(after) != len(before)-1 || healed.HP-hurt.HP < min(heal, wound) {
		t.Fatalf("potion %#04x: %d units -> %d, health %d -> %d", code, len(before), len(after), hurt.HP, healed.HP)
	}
}

// A potion bought in town is drunk in the mission through the pack's own
// use: the unit is used up and the hero's health rises. Fails when the bag
// enters the mission in its bag form, type 0 with no payload.
func TestSecondBagPotionIsDrunkInTheMission(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, []sim.ItemInstance{secondBagPotion})
	secondBagPotionTable(f)
	secondBagEnter(t, f, app, screen)
	secondBagDrink(t, f.live, secondBagPotion.Code, 30, func() { f.live.tick() })
}

// A bought potion and a potion of the mission's own making both hold the
// potion's type and payload in the mission, and both come home to the bag in
// its form at the next town. Fails when the bag enters the mission in its bag
// form, or when the arrival compares a unit with its bag form instead of its
// mission form, which leaves the mission's potion in the hidden pack.
func TestSecondBagPotionsComeHome(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, []sim.ItemInstance{secondBagPotion})
	secondBagPotionTable(f)
	found := mapload.ItemInstanceFromCode(secondBagPotion.Code, f.Table)
	hero, ok := memberWithPack(f.Carried[0], []sim.ItemInstance{found}, f.Table, true)
	if !ok {
		t.Fatal("the hero's pack refused the potion")
	}
	f.Carried[0] = hero
	secondBagEnter(t, f, app, screen)
	items, _ := f.live.world.CarriedItems(f.live.mission.ids[0])
	if n := len(items); n != 2 || !sim.UsablePotion(items[0]) || !sim.UsablePotion(items[1]) {
		t.Fatalf("the mission pack holds %v, want two usable potions", bagLabels(items))
	}
	secondBagWin(t, f, 10)
	(&destinationTownMap{screen}).enterLocation(partyPlace{kind: placeTown, id: 2})
	if got, want := bagLabels(*f.Town.second.heroBag()), bagLabels([]sim.ItemInstance{secondBagPotion, secondBagPotion}); !slices.Equal(got, want) {
		t.Fatalf("bag at town 2 %v, want %v", got, want)
	}
	if left := mapload.MemberCarriedItems(f.Carried[0], f.Table); len(left) != 0 {
		t.Fatalf("the hero's pack kept %v", bagLabels(left))
	}
}

// At a town's arrival a pack unit comes home only when the bag keeps all it
// holds: the mission's weapon, with its weight, comes home as the shop's
// unit, while a unit at a negative price and an armour with a second effect
// stay in the hero's pack. Fails when the arrival writes every unit through
// secondshop.Unit, which would price the first at 0 and drop the second's
// other effect.
func TestSecondBagArrivalKeepsWhatTheBagWouldLose(t *testing.T) {
	f, _, screen, _ := secondBagFront(t, nil)
	carried := secondBagWeapon.Clone()
	carried.Weight, carried.WeightPresent = 9, true
	unpriced := sim.ItemInstance{Code: 0x0e4d, Price: -1}
	twice := sim.ItemInstance{Code: 0x1402, Kind: 1, Price: 30, Effects: []sim.ItemEffect{{Kind: 41, Operand: 5 | 2<<16}, {Kind: 7}}}
	hero, ok := memberWithPack(f.Carried[0], []sim.ItemInstance{secondBagPotion, unpriced, carried, twice}, f.Table, true)
	if !ok {
		t.Fatal("the hero's pack refused the units")
	}
	f.Carried[0] = hero
	if err := screen.sess.enteredTown(screen.in.townInstall()); err != nil {
		t.Fatal(err)
	}
	if got, want := bagLabels(*f.Town.second.heroBag()), bagLabels([]sim.ItemInstance{secondBagPotion, secondBagWeapon}); !slices.Equal(got, want) {
		t.Fatalf("bag %v, want %v", got, want)
	}
	if got, want := bagLabels(mapload.MemberCarriedItems(f.Carried[0], f.Table)), bagLabels([]sim.ItemInstance{unpriced, twice}); !slices.Equal(got, want) {
		t.Fatalf("hero's pack %v, want %v", got, want)
	}
}

// An arrival whose town item graph cannot follow the hero's pack moves
// nothing and says so: the units stay in his pack, the bag stays as it was
// and the graph is the one the town had. Fails when the arrival moves the
// units and keeps a graph that writes them under the records of others.
func TestSecondBagArrivalRefusesAGraphThatCannotFollow(t *testing.T) {
	f, _, screen, _ := secondBagFront(t, nil)
	hero, ok := memberWithPack(f.Carried[0], []sim.ItemInstance{secondBagPotion}, f.Table, true)
	if !ok {
		t.Fatal("the hero's pack refused the unit")
	}
	f.Carried[0] = hero
	broken := &cityObjectTopology{}
	f.Town.cityObjects = broken
	bag := bagLabels(*f.Town.second.heroBag())
	if err := screen.sess.enteredTown(screen.in.townInstall()); err == nil {
		t.Fatal("the arrival accepted a town item graph with no header")
	}
	if got := bagLabels(*f.Town.second.heroBag()); !slices.Equal(got, bag) {
		t.Fatalf("bag %v after the refused arrival, want %v", got, bag)
	}
	if got, want := bagLabels(mapload.MemberCarriedItems(f.Carried[0], f.Table)), bagLabels([]sim.ItemInstance{secondBagPotion}); !slices.Equal(got, want) {
		t.Fatalf("hero's pack %v after the refused arrival, want %v", got, want)
	}
	if f.Town.cityObjects != broken {
		t.Fatal("the refused arrival replaced the town item graph")
	}
}

// The hero enters mission 10 wearing what he wore in town, his pack holding
// his own pack and the bag, unit for unit with their IDs, types, prices and
// effects; Iglez enters with his own pack and armour. An empty bag adds
// nothing. A bag the map holds, as after a LOAD at the map, enters from the
// map the same way. Fails with the bag left in the campaign at the town's
// leave or at the map's mission entry.
func TestSecondBagEntersTheMissionWithTheHero(t *testing.T) {
	for _, tc := range []struct {
		name  string
		bag   []sim.ItemInstance
		atMap bool
	}{{"bought", secondBagSample(), false}, {"empty", nil, false}, {"at the map", secondBagSample(), true}} {
		t.Run(tc.name, func(t *testing.T) {
			town := tc.bag
			if tc.atMap {
				town = nil
			}
			f, app, screen, _ := secondBagFront(t, town)
			own := mapload.MemberCarriedItems(f.Carried[0], f.Table)
			if tc.atMap {
				secondBagEnterFromMap(t, f, app, screen, tc.bag)
			} else {
				secondBagEnter(t, f, app, screen)
			}
			packs, worn := secondBagHeld(f.live.world, f.live.mission.ids)
			if len(packs) != 2 {
				t.Fatalf("mission party of %d, want the hero and Iglez", len(packs))
			}
			if want := bagLabels(append(own, tc.bag...)); !slices.Equal(packs[0], want) {
				t.Fatalf("hero's mission pack %v, want %v", packs[0], want)
			}
			if want := bagLabels([]sim.ItemInstance{secondBagWornWeapon, secondBagWornArmour}); !slices.Equal(worn[0], want) {
				t.Fatalf("hero wears %v, want %v", worn[0], want)
			}
			if !slices.Equal(packs[1], bagLabels([]sim.ItemInstance{secondBagIglezPack})) || !slices.Equal(worn[1], bagLabels([]sim.ItemInstance{secondBagIglezArmour})) {
				t.Fatalf("Iglez carries %v and wears %v", packs[1], worn[1])
			}
			if c := f.Town.second; len(c.party) != 0 && len(c.party[0].CarriedItems) != 0 {
				t.Fatalf("the bag kept %d units in the mission", len(c.party[0].CarriedItems))
			}
		})
	}
}

// A mission SAVE and its cold LOAD keep the hero's and Iglez's packs and
// worn items. Fails with the bag left in the campaign at the town's leave.
func TestSecondBagMissionSaveAndColdLoadKeepThePacks(t *testing.T) {
	f, app, screen, dir := secondBagFront(t, secondBagSample())
	secondBagEnter(t, f, app, screen)
	packs, worn := secondBagHeld(f.live.world, f.live.mission.ids)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := SaveStore{Dir: dir}.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("mission SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
	}
	if _, err := os.Stat(filepath.Join(dir, entries[0].Name)); err != nil {
		t.Fatal(err)
	}
	cold, _ := secondMissionLoadOf(t, secondMissionContent{side: 128}, dir, entries[0].Name)
	if cold.live == nil {
		t.Fatal("the cold LOAD opened no mission")
	}
	gotPacks, gotWorn := secondBagHeld(cold.live.world, cold.live.mission.ids)
	if !slices.EqualFunc(gotPacks, packs, slices.Equal) || !slices.EqualFunc(gotWorn, worn, slices.Equal) {
		t.Fatalf("cold LOAD packs %v worn %v, want %v and %v", gotPacks, gotWorn, packs, worn)
	}
	if !slices.Contains(gotPacks[0], bagLabels([]sim.ItemInstance{secondBagArmour})[0]) {
		t.Fatalf("cold LOAD hero pack %v lacks the bought armour", gotPacks[0])
	}
}

// A won mission comes home with what the mission left: the armour worn in
// the mission stays worn, the dropped weapon stays behind, the potion
// dropped and picked up again comes back. The arrival at town 2 puts the
// hero's pack into the bag, and the shop sells the potion picked up in the
// mission. Iglez keeps his own items. Fails with the arrival leaving the
// pack with the hero.
func TestSecondBagComesHomeAsTheMissionLeftIt(t *testing.T) {
	f, app, screen, _ := secondBagFront(t, secondBagSample())
	own := mapload.MemberCarriedItems(f.Carried[0], f.Table)
	secondBagEnter(t, f, app, screen)
	mw := f.live
	w, id := mw.world, mw.mission.ids[0]
	slot := func(code uint16) sim.ItemSlot {
		stacks, _ := w.CarriedStacks(id)
		for i, s := range stacks {
			if s.Code == code {
				return sim.ItemSlot(i)
			}
		}
		t.Fatalf("pack holds no %#04x", code)
		return 0
	}
	hero, _ := w.Entity(id)
	east, west := sim.CellPoint{X: hero.X + 1, Y: hero.Y}, sim.CellPoint{X: hero.X - 1, Y: hero.Y}
	steps := func(cmd sim.Command, ticks int) {
		mw.pending = append(mw.pending, cmd)
		for range ticks {
			mw.tick()
		}
	}
	steps(sim.Equip(id, slot(secondBagArmour.Code), 4), 8)
	steps(sim.DropCarried(id, slot(secondBagWeapon.Code), east), 8)
	steps(sim.DropCarried(id, slot(secondBagPotion.Code), west), 8)
	if n := len(w.Sacks()); n != 2 {
		t.Fatalf("%d sacks after the two drops, want 2", n)
	}
	steps(sim.PickUp(id, west), 400)
	packs, worn := secondBagHeld(w, mw.mission.ids)
	if want := bagLabels(append(slices.Clone(own), secondBagPotion, secondBagPotion)); !slices.Equal(packs[0], want) {
		t.Fatalf("mission pack %v after the pick-up, want %v", packs[0], want)
	}
	if want := bagLabels([]sim.ItemInstance{secondBagWornWeapon, secondBagWornArmour, secondBagArmour}); !slices.Equal(worn[0], want) {
		t.Fatalf("mission worn %v, want %v", worn[0], want)
	}
	secondBagWin(t, f, 10)
	c := f.Town.second
	if len(c.party) != 0 && len(c.party[0].CarriedItems) != 0 {
		t.Fatalf("after the victory the bag holds %d units on the map", len(c.party[0].CarriedItems))
	}
	if at := (&destinationTownMap{screen}).enterLocation(partyPlace{kind: placeTown, id: 2}); at.Msg != "" {
		t.Fatalf("the arrival at town 2 reports %q", at.Msg)
	}
	wantBag := bagLabels(append(own, secondBagPotion, secondBagPotion))
	if got := bagLabels(*c.heroBag()); !slices.Equal(got, wantBag) {
		t.Fatalf("bag at town 2 %v, want %v", got, wantBag)
	}
	if left := mapload.MemberCarriedItems(f.Carried[0], f.Table); len(left) != 0 {
		t.Fatalf("the hero's pack kept %v", bagLabels(left))
	}
	// The town views follow the carried pack; a stale view is what the
	// town SAV's LOAD does not give back.
	if hero := f.Carried[0]; hero.Carry == nil || len(hero.Carried) != 0 || len(hero.CarriedItems) != 0 {
		t.Fatalf("the hero's town views hold %#04x and %d units beside the carried pack", hero.Carried, len(hero.CarriedItems))
	}
	// The town's item graph lets the units go with them: no pack edge of the
	// hero, and no Item or record that no root reaches, which the town SAV
	// would write under the kept units.
	g := f.Town.cityObjects
	if g == nil {
		t.Fatal("the return left no town item graph")
	}
	reached := map[sim.SavedObjectID]bool{}
	for _, root := range g.Roots {
		if string(root.PartyID) == f.Carried[0].ID && len(root.Pack) != 0 {
			t.Fatalf("the hero's town pack edges %v outlive his pack", root.Pack)
		}
		for _, id := range append(slices.Clone(root.Pack), root.Worn[:]...) {
			reached[id] = true
		}
	}
	for _, item := range g.Items {
		if !reached[item.ID] {
			t.Fatalf("town Item %d is held by no one", item.ID)
		}
	}
	for id := range g.ItemRecords {
		if !reached[id] {
			t.Fatalf("town Item record %d is held by no one", id)
		}
	}
	wantWorn := bagLabels([]sim.ItemInstance{secondBagWornWeapon, secondBagWornArmour, secondBagArmour})
	if got := secondBagWorn(mapload.MemberItemEquipment(f.Carried[0], f.Table)); !slices.Equal(got, wantWorn) {
		t.Fatalf("hero wears %v at town 2, want %v", got, wantWorn)
	}
	if got := bagLabels(mapload.MemberCarriedItems(f.Carried[1], f.Table)); !slices.Equal(got, bagLabels([]sim.ItemInstance{secondBagIglezPack})) {
		t.Fatalf("Iglez carries %v at town 2", got)
	}
	stacks, ok := secondshop.Stacks(*c.heroBag())
	if !ok {
		t.Fatal("the bag holds a unit with no bag form")
	}
	i := slices.IndexFunc(stacks, func(it secondshop.Item) bool { return it.Table == secondshop.Magic && it.Row == 1 })
	c.shop.filled, c.shop.tables = 1, secondShopTestTables
	gold := 0
	if i < 0 || !c.shop.offer(c.heroBag(), i) || !c.shop.sell(c.heroBag(), &gold) || gold != 20 {
		t.Fatalf("the sale of the potions: stack %d gold %d", i, gold)
	}
}

// An original mission file's hero bag is the mission hero's pack: the LOAD
// leaves the campaign's bag empty, and after the victory the town's bag
// holds each unit once. Fails with the file's bag also read into the
// campaign's bag.
func TestSecondBagOfAMissionFileComesHomeOnce(t *testing.T) {
	hero := secondMissionFixtureHero()
	f, _, _ := secondMissionLoadFile(t, secondMissionFile(hero, secondMissionStore(4, 0, -1)))
	if f.live == nil || f.Town == nil || f.Town.second == nil {
		t.Fatal("the mission file opened no second-game mission")
	}
	c := f.Town.second
	if bag := bagLabels(*c.heroBag()); len(bag) != 0 {
		t.Fatalf("the LOAD read the mission's bag %v into the campaign's bag", bag)
	}
	pack, _ := f.live.world.CarriedItems(f.live.mission.ids[0])
	want := bagLabels(pack)
	if len(want) == 0 {
		t.Fatal("the mission hero carries no bag")
	}
	secondBagWin(t, f, 10)
	(&destinationTownMap{f.TownScreen().(*townScreen)}).enterLocation(partyPlace{kind: placeTown, id: 2})
	if got := bagLabels(*c.heroBag()); !slices.Equal(got, want) {
		t.Fatalf("bag at town 2 %v, want %v", got, want)
	}
}
