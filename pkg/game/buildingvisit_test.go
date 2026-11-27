package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

var sessionNow func() time.Time

func visitCases() []sessionCase { return visitCasesOf(34) }

func visitCasesOf(kind uint32) []sessionCase {
	cases := sessionEndCases()
	cases[0].mapB = func(t *testing.T) []byte {
		b := synth.ALM(synth.ALMOptions{Width: 128, Height: 128, Name: "Arena",
			Objects: []synth.ALMObject{{X: 20 << 8, Y: 20 << 8, Kind: 34, Field0C: 1}}})
		binary.LittleEndian.PutUint32(b[alm.FileHeaderSize+alm.RecordHeaderSize+alm.MetaWord70:], 2)
		return b
	}
	front := cases[0].front
	cases[0].front = func(t *testing.T) *FrontEnd {
		f := front(t)
		buildings := make(dbCollection, 36)
		buildings[34] = dbEntry{name: "structure", params: []int32{2, 2, 0, 100, 0xF, 0xF}}
		f.Table.Buildings = buildings
		return f
	}
	cases[1].mapB = func(t *testing.T) []byte {
		return secondMissionMapOf(128, 128, secondMissionContent{
			objects: []alm.Object{{X: 20<<8 | 0x80, Y: 20<<8 | 0x80, Kind: kind, Field0C: 1}}})
	}
	second := cases[1].front
	cases[1].front = func(t *testing.T) *FrontEnd {
		f := second(t)
		rows := sleepCollection{secondShopTestCollection{}, 128}
		for _, i := range []int{6, 35, 69, 70, 71, 72, 73, 74} {
			rows.secondShopTestCollection[i] = []int32{10}
		}
		f.Table.MagicItems, f.Table.Magic = rows, rows
		f.Table.Spells = sleepCollection{secondShopTestCollection{}, 1}
		return f
	}
	return cases
}

type visit struct {
	f    *FrontEnd
	app  *ui.App
	dir  string
	hero sim.EntityID
	cell mapload.Cell
}

func visitEnter(t *testing.T, c sessionCase) visit {
	t.Helper()
	f, app, dir, id, _ := returnSession(t, c)
	if f.townUI == nil {
		f.townUI = f.TownScreen().(*townScreen)
		app.SetTown(f.townUI)
	}
	mw := f.live
	mw.useStructure(uint32(id), uint32(mw.world.Structures()[0].ID))
	for range 4000 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if e := returnHeroEntity(t, f, id); e.OffMap {
			return visit{f: f, app: app, dir: dir, hero: id, cell: mapload.Cell{X: e.X, Y: e.Y}}
		}
	}
	t.Fatal("the hero never entered the Shop")
	return visit{}
}

func (v visit) leave(t *testing.T) sim.Entity {
	t.Helper()
	if v.f.townUI.MissionShopOpen() {
		sessionKey(t, v.app, "escape")
	}
	for range 40 {
		if err := v.app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if e := returnHeroEntity(t, v.f, v.hero); !e.OffMap {
			return e
		}
	}
	t.Fatal("the hero never left the Shop")
	return sim.Entity{}
}

func TestSessionShopEntryTakesTheHeroOffTheMapAndLeaveReturnsHim(t *testing.T) {
	for _, c := range visitCases() {
		t.Run(c.name, func(t *testing.T) {
			v := visitEnter(t, c)
			if page := v.f.townUI != nil && v.f.townUI.MissionShopOpen(); !page {
				t.Fatalf("Shop page open %v", page)
			}
			e := v.leave(t)
			if e.X != v.cell.X || e.Y != v.cell.Y {
				t.Fatalf("left at %d,%d, want his own cell %v", e.X, e.Y, v.cell)
			}
		})
	}
}

func TestSessionShopLeaveWithTheCellTakenStandsWithinThree(t *testing.T) {
	for _, c := range visitCases() {
		t.Run(c.name, func(t *testing.T) {
			v := visitEnter(t, c)
			if _, err := v.f.live.world.CheatSummon(sim.Entity{Owner: 3, HP: 50, MaxHP: 50}, nil,
				[sim.EquipSlots]sim.ItemInstance{}, v.cell.X, v.cell.Y); err != nil {
				t.Fatalf("the cell of the hero inside is not free: %v", err)
			}
			e := v.leave(t)
			dx, dy := e.X-v.cell.X, e.Y-v.cell.Y
			if dx == 0 && dy == 0 || dx < -3 || dx > 3 || dy < -3 || dy > 3 {
				t.Fatalf("left at %d,%d, want a free cell within 3 of %v", e.X, e.Y, v.cell)
			}
		})
	}
}

func TestSessionShopDealsWriteTheFileOncePerFifteenSeconds(t *testing.T) {
	now := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	sessionNow = func() time.Time { return now }
	t.Cleanup(func() { sessionNow = nil })
	c := visitCases()[0]
	f, app, dir, id, _ := returnSession(t, c)
	mw := f.live
	e := returnHeroEntity(t, f, id)
	if !mw.world.CheatAddGold(uint32(e.Owner), 100) {
		t.Fatal("purse fixture refused")
	}
	shop := NewShop(1000)
	stock := ShopItem{Code: 0xe05, Price: 5, Count: 4}
	shop.shelves[shopShelfForItem(stock, f.Table)] = []ShopItem{stock}
	shop.inputEpoch = new(byte)
	mw.shop.stock = shop
	path := filepath.Join(dir, c.file)
	old := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	stamp := func() {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	written := func() bool {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		return !fi.ModTime().Equal(old)
	}
	stamp()
	mw.useStructure(uint32(id), uint32(mw.world.Structures()[0].ID))
	for range 4000 {
		if f.townUI != nil && f.townUI.MissionShopOpen() {
			break
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !f.townUI.MissionShopOpen() {
		t.Fatal("the Shop page never opened")
	}
	for i, tc := range []struct {
		after time.Duration
		write bool
	}{{0, true}, {10 * time.Second, false}, {16 * time.Second, true}} {
		now = now.Add(tc.after)
		missionShopBuy(f.townUI)
		if got := written(); got != tc.write {
			t.Fatalf("deal %d at +%v: file written %v, want %v", i+1, tc.after, got, tc.write)
		}
		stamp()
	}
}

func TestCampaignShopDealKeepsTheHeroOnTheMapAndWritesNothing(t *testing.T) {
	f := missionShopFront(t, 35)
	m := missionShopOpened(t, f)
	if f.live.mission.buildings.dealt != nil || f.live.world.VisitsBuildings() {
		t.Fatal("a campaign map carries the session's building rule")
	}
	missionShopBuy(f.townUI)
	if e, _ := f.live.world.Entity(m.hero); e.OffMap {
		t.Fatal("a campaign hero went off the map in the Shop")
	}
}

func TestSessionSummonUsingAShopOrAnInnStaysOnTheMap(t *testing.T) {
	for _, tc := range []struct {
		name string
		c    sessionCase
	}{{"first game Shop", visitCases()[0]}, {"second game Shop", visitCases()[1]}, {"second game Inn", visitCasesOf(67)[1]}} {
		t.Run(tc.name, func(t *testing.T) {
			f, app, _, id, _ := returnSession(t, tc.c)
			mw := f.live
			hero := returnHeroEntity(t, f, id)
			summon, err := mw.world.CheatSummon(sim.Entity{Owner: hero.Owner, HP: 50, MaxHP: 50}, nil,
				[sim.EquipSlots]sim.ItemInstance{}, 19, 19)
			if err != nil {
				t.Fatal(err)
			}
			mw.useStructure(uint32(summon), uint32(mw.world.Structures()[0].ID))
			for range 400 {
				if err := app.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				if e, _ := mw.world.Entity(summon); e.OffMap || len(mw.shop.inside) != 0 || mw.shop.waiting {
					t.Fatalf("a summon entered the building: off map %v, inside %d", e.OffMap, len(mw.shop.inside))
				}
			}
		})
	}
}
