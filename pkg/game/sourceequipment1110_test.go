package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestSourceEquipment1110LiteralBothAppDoorsFreshNativeAndShop(t *testing.T) {
	attack := [24]byte{11, 0, 13, 0, 14, 0, 15, 0, 16, 0, 17, 0, 18, 0, 7, 9, 3, 6, 8, 2, 4, 1, 93, 94}
	defence := [22]byte{19, 0, 21, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18}
	item := &holdingFixtureItem{class: "Weapon", code: 0x0101, row: 37, ownKind: 3, count: 2, kind: 2, weight: 7, attack: attack, defence: defence}
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, loadWords: &[4]int16{37, 7, 53, 301}, holdings: &holdingFixture{items: []*holdingFixtureItem{item}}}
	b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 10, maxHP: 100, loadWords: &[4]int16{19, 2, 0, 301}, holdings: &holdingFixture{}}
	want := sim.ItemInstance{Code: 0x0101, Kind: 2, Weight: 7, WeightPresent: true, SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 37, OwnKind: 3, Attack: attack, Defence: defence}}
	payload := poolFixtureSave(a, b)
	sf, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	holdings, err := sf.ActorHoldings()
	if err != nil || len(holdings) != 2 || holdings[0].Items[0].W52 != attack || holdings[0].Items[0].W6A != defence || holdings[0].Items[0].W50 != 3 || holdings[0].Items[0].Row != 37 {
		t.Fatal("literal record projection", err)
	}
	for _, fromMap := range []bool{false, true} {
		f := currentPoolFixtureFront(t, 91, 92)
		app := f.App("retained source equipment")
		if fromMap {
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
		}
		originals := t.TempDir()
		if err := os.WriteFile(filepath.Join(originals, "game1110.sav"), payload, 0600); err != nil {
			t.Fatal(err)
		}
		store := SaveStore{Dir: t.TempDir()}
		save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
		app.SetSaveSeams(save, list, load)
		groundAppLoad(t, app, list, "game1110.sav")
		one, two := poolEntity(t, f.live.world, 91), poolEntity(t, f.live.world, 92)
		if err := f.live.world.MoveCarried(one.ID, two.ID, want.Code, 1); err != nil {
			t.Fatal(err)
		}
		got, _ := f.live.world.CarriedItems(two.ID)
		if len(got) != 1 || !reflect.DeepEqual(got[0], want) {
			t.Fatal("source transfer lost operands", got)
		}
		if !reflect.DeepEqual(shopItemFromInstance(got[0], 1).Instance(), want) {
			t.Fatal("shop item drops source state")
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		entries, err := store.List()
		if err != nil || len(entries) != 1 {
			t.Fatal(entries, err, app.HeadlessMessage())
		}
		fresh := currentPoolFixtureFront(t, 91, 92)
		freshApp := fresh.App("fresh retained source equipment")
		fs, fl, ff := fresh.SaveSeams(store, OriginalStore{}, nil)
		freshApp.SetSaveSeams(fs, fl, ff)
		groundAppLoad(t, freshApp, fl, localOriginalSaveToken(entries[0].Name))
		if fresh.live.world.Hash() != f.live.world.Hash() {
			t.Fatal("fresh native lost equipment state")
		}
		for _, world := range []*sim.World{f.live.world, fresh.live.world} {
			if err := world.MoveCarried(two.ID, one.ID, want.Code, 1); err != nil {
				t.Fatal(err)
			}
		}
		if fresh.live.world.Hash() != f.live.world.Hash() {
			t.Fatal("next transfer differs")
		}
	}
}
