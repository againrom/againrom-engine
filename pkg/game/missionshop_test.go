package game

import (
	"slices"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// missionShopFront is a first-game campaign front whose mission 10 holds one
// structure of class kind at cell 58/100, beside a party of one. The town has
// arrived with 777 gold and a shop whose one shelf holds two units of 0xe05
// at 5 each.
func missionShopFront(t *testing.T, kind uint32) *FrontEnd {
	t.Helper()
	m := synth.ALM(synth.ALMOptions{Width: 128, Height: 128,
		Units:   []synth.ALMUnit{{X: 0x0280, Y: 0x01c0}, {X: 0x0500, Y: 0x0080}},
		Objects: []synth.ALMObject{{X: 58 << 8, Y: 100 << 8, Kind: kind, Field0C: 1}}})
	f := poolFixtureFrontMap(t, m)
	f.Table = actorRegistryTable()
	units := f.Table.Units.(dbCollection)
	f.Table.Units = dbCollection{units[1], units[1]}
	f.Humans = fourBaseHumans()
	f.Table.Humans = append(fourBaseHumans(), dbEntry{name: "actor runtime", params: humansParams(50, 0, 5)})
	f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	buildings := make(dbCollection, 36)
	buildings[kind] = dbEntry{name: "structure", params: []int32{2, 2, 0, 100, 0xF, 0xF}}
	f.Table.Buildings = buildings
	f.Town = NewTown(f.Campaign.Value())
	f.Town.Arrive()
	f.Town.gold = 777
	f.Shop = NewShop(1000)
	stock := ShopItem{Code: 0xe05, Price: 5, Count: 2}
	f.Shop.shelves[shopShelfForItem(stock, f.Table)] = []ShopItem{stock}
	return f
}

// missionShop is one opened fixture mission: the hero, and his pack and purse
// as they stood when he used the structure.
type missionShop struct {
	app   *ui.App
	hero  sim.EntityID
	pack  []uint16
	purse uint32
}

// missionShopOpened opens mission 10, puts one 0xe03 priced 9 in the hero's
// pack and 50 gold in his purse, has him use the structure, and steps until
// the use has ended or the town screen shows.
func missionShopOpened(t *testing.T, f *FrontEnd) missionShop {
	t.Helper()
	app := f.App("mission shop")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	hero := f.live.mission.ids[0]
	e, _ := f.live.world.Entity(hero)
	sold := sim.PlainItem(0xe03)
	sold.Price = 9
	if !f.live.world.CheatAddItem(hero, sim.StackItem(sold, 1)) || !f.live.world.CheatAddGold(uint32(e.Owner), 50) {
		t.Fatal("pack fixture refused")
	}
	pack, purse := missionShopPack(f, hero)
	f.live.useStructure(uint32(hero), 0)
	seen, after := false, 0
	for step := 0; step < 600 && app.Screen() == ui.ScreenMap && after < 4; step++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if len(f.live.world.StructureUses()) != 0 {
			seen = true
		} else if seen {
			after++
		}
	}
	if !seen {
		t.Fatal("the use never started")
	}
	return missionShop{app: app, hero: hero, pack: pack, purse: purse}
}

func missionShopPack(f *FrontEnd, hero sim.EntityID) ([]uint16, uint32) {
	codes, _ := f.live.world.Carried(hero)
	e, _ := f.live.world.Entity(hero)
	return codes, f.live.world.Purse(uint32(e.Owner))
}

// missionShopBuy puts the shelf's first unit on the table and buys it.
func missionShopBuy(s *townScreen) {
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlShelfCell, Index: 0})
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 1})
}

// A Shop use opens the town's Shop page over the mission; buy and sell act
// on the hero's pack and purse; the mission's clock stands while the page
// shows; Escape returns to the mission, whose clock runs again, and the
// town's own shop and gold are as they were.
func TestMissionShopUseOpensTheShopPageAndTradesWithTheHero(t *testing.T) {
	f := missionShopFront(t, 35)
	townGold := f.Town.gold
	m := missionShopOpened(t, f)
	s := f.townUI
	if m.app.Screen() != ui.ScreenTown || !s.MissionShopOpen() || !s.AtTownShop() {
		t.Fatalf("screen %v open %v at shop %v, want the Shop page over the mission", m.app.Screen(), s.MissionShopOpen(), s.AtTownShop())
	}
	if s.sess == &f.CampaignSession || s.sess.Town.Gold() != int(m.purse) || len(s.sess.Carried) != 1 || s.sess.Town.Open() {
		t.Fatalf("the visit's session: gold %d party %d open %v", s.sess.Town.Gold(), len(s.sess.Carried), s.sess.Town.Open())
	}
	tick := f.live.world.Tick()
	for range 8 {
		if err := m.app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Tick() != tick || m.app.Screen() != ui.ScreenTown {
		t.Fatalf("the clock moved %d -> %d under the Shop page", tick, f.live.world.Tick())
	}
	missionShopBuy(s)
	codes, purse := missionShopPack(f, m.hero)
	if want := append(slices.Clone(m.pack), 0xe05); !slices.Equal(codes, want) || purse != m.purse-5 || s.sess.Town.Gold() != int(purse) {
		t.Fatalf("after the buy: pack %x purse %d page gold %d", codes, purse, s.sess.Town.Gold())
	}
	sell := slices.Index(s.shopPackItems(), 0xe03)
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: s.packCell(sell)})
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	sold := slices.DeleteFunc(append(slices.Clone(m.pack), 0xe05), func(c uint16) bool { return c == 0xe03 })
	codes, after := missionShopPack(f, m.hero)
	if !slices.Equal(codes, sold) || after <= purse || int(after) != s.sess.Town.Gold() {
		t.Fatalf("after the sale: pack %x purse %d page gold %d", codes, after, s.sess.Town.Gold())
	}
	if shelf := f.Shop.Shelf(shopShelfForItem(ShopItem{Code: 0xe05}, f.Table)); f.Town.gold != townGold || len(shelf) != 1 || shelf[0].Count != 2 {
		t.Fatal("the visit traded with the town's own shop or gold")
	}
	if err := m.app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if m.app.Screen() != ui.ScreenMap || s.MissionShopOpen() || s.sess != &f.CampaignSession {
		t.Fatalf("Escape left screen %v open %v", m.app.Screen(), s.MissionShopOpen())
	}
	for range 4 {
		if err := m.app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if f.live.world.Tick() == tick {
		t.Fatal("the mission clock did not run again after the Shop page closed")
	}
	if got, _ := missionShopPack(f, m.hero); !slices.Equal(got, sold) {
		t.Fatalf("Exit changed the pack to %x", got)
	}
}

// The page's Exit button closes it as Escape does, and a gesture on the doll
// is refused while the page shows over a mission.
func TestMissionShopExitButtonReturnsAndTheDollIsFixed(t *testing.T) {
	f := missionShopFront(t, 34)
	m := missionShopOpened(t, f)
	s := f.townUI
	if !s.MissionShopOpen() {
		t.Fatal("Shop 1 did not open the page")
	}
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: s.packCell(0)}, ui.ShopControl{Kind: ui.ShopControlDoll})
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlDoll, Index: 0})
	if got, _ := missionShopPack(f, m.hero); !slices.Equal(got, m.pack) {
		t.Fatalf("a doll gesture changed the pack %x -> %x", m.pack, got)
	}
	s.ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 3})
	if err := m.app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if m.app.Screen() != ui.ScreenMap || s.MissionShopOpen() {
		t.Fatalf("Exit left screen %v open %v", m.app.Screen(), s.MissionShopOpen())
	}
}

// A structure use of a class that is not a Shop opens no page.
func TestMissionNonShopStructureUseOpensNoPage(t *testing.T) {
	f := missionShopFront(t, 28)
	m := missionShopOpened(t, f)
	if m.app.Screen() != ui.ScreenMap || f.townUI.MissionShopOpen() || f.live.shop.waiting {
		t.Fatalf("class 28 use: screen %v open %v", m.app.Screen(), f.townUI.MissionShopOpen())
	}
}

// A SAVE taken in the mission after a purchase writes the bought item and
// the purse, and a fresh LOAD restores them.
func TestMissionShopPurchaseSavesAndLoads(t *testing.T) {
	f := missionShopFront(t, 35)
	m := missionShopOpened(t, f)
	missionShopBuy(f.townUI)
	if err := m.app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	want, purse := missionShopPack(f, m.hero)
	if !slices.Equal(want, append(slices.Clone(m.pack), 0xe05)) || purse != m.purse-5 || m.app.Screen() != ui.ScreenMap {
		t.Fatalf("purchase fixture: pack %x purse %d screen %v", want, purse, m.app.Screen())
	}
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{}, nil)
	m.app.SetSaveSeams(save, list, load)
	if err := m.app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := m.app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 {
		t.Fatalf("SAVE wrote %+v %v: %s", entries, err, m.app.HeadlessMessage())
	}
	fresh := missionShopFront(t, 35)
	freshApp := fresh.App("mission shop load")
	fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
	freshApp.SetSaveSeams(fs, fl, ff)
	groundAppLoad(t, freshApp, fl, entries[0].Name)
	if fresh.live == nil || fresh.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("LOAD did not restore the world the SAVE wrote")
	}
	if got, gold := missionShopPack(fresh, m.hero); !slices.Equal(got, want) || gold != purse {
		t.Fatalf("LOAD restored pack %x purse %d, want %x %d", got, gold, want, purse)
	}
}
