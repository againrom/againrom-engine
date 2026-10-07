package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestInstanceWeightNonpartyBothLoadDoorsAndNewTransfer(t *testing.T) {
	// Independent serializer controls: held zero, held positive, sparse worn
	// negative, and three equal-code carried objects with distinct signed values.
	a := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 7, maxHP: 31, holdings: &holdingFixture{humanoid: true,
		weapon: &holdingFixtureItem{class: "Weapon", code: 0x0101, count: 1, kind: 2, price: -200, weight: 0},
		shield: &holdingFixtureItem{class: "Shield", code: 0x0201, count: 1, kind: 1, price: 20, weight: 7},
		items: []*holdingFixtureItem{
			{class: "Item", code: 0x0e06, count: 5, kind: 3, price: 50, weight: -5},
			{class: "Item", code: 0x0e06, count: 2, kind: 3, price: 50, weight: 0},
			{class: "Item", code: 0x0e06, count: 3, kind: 3, price: 50, weight: 8},
		}}}
	a.holdings.worn[11] = &holdingFixtureItem{class: "Armor", code: 0x0c01, count: 1, kind: 1, price: 70, weight: -3}
	b := &poolFixtureActor{mapID: 92, cell: 0x0606, hp: 9, maxHP: 41, holdings: &holdingFixture{}}
	payload := poolFixtureSave(a, b)
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	holdings, err := source.ActorHoldings()
	if err != nil || len(holdings) != 2 || holdings[0].HeldWeapon.Weight != 0 || holdings[0].HeldShield.Weight != 7 || holdings[0].Worn[11].Weight != -3 || holdings[0].Items[0].Weight != -5 || holdings[0].Items[1].Weight != 0 || holdings[0].Items[2].Weight != 8 {
		t.Fatalf("literal all-Player weight projection: %+v %v", holdings, err)
	}
	worn := [sim.EquipSlots]sim.ItemInstance{
		0:  {Code: 0x0101, Kind: 2, Price: -200, WeightPresent: true, SourceEquipment: sim.SourceEquipment{Class: sim.SourceWeapon, DefinitionRow: 1}},
		1:  {Code: 0x0201, Kind: 1, Price: 20, Weight: 7, WeightPresent: true, SourceEquipment: sim.SourceEquipment{Class: sim.SourceShield, DefinitionRow: 1}},
		11: {Code: 0x0c01, Kind: 1, Price: 70, Weight: -3, WeightPresent: true, SourceEquipment: sim.SourceEquipment{Class: sim.SourceArmor, DefinitionRow: 1, OwnKind: 12}},
	}
	pack := []sim.ItemStack{
		{Code: 0x0e06, Kind: 3, Price: 50, Count: 5, Weight: -5, WeightPresent: true},
		{Code: 0x0e06, Kind: 3, Price: 50, Count: 2, Weight: 0, WeightPresent: true},
		{Code: 0x0e06, Kind: 3, Price: 50, Count: 3, Weight: 8, WeightPresent: true},
	}
	assert := func(w *sim.World, transferred bool) {
		t.Helper()
		id := poolEntity(t, w, 91).ID
		gotWorn, _ := w.EquippedItems(id)
		gotPack, _ := w.CarriedStacks(id)
		wantPack := append([]sim.ItemStack(nil), pack...)
		if transferred {
			wantPack[0].Count = 3
			other, _ := w.CarriedStacks(poolEntity(t, w, 92).ID)
			if !reflect.DeepEqual(other, []sim.ItemStack{{Code: 0x0e06, Kind: 3, Price: 50, Count: 2, Weight: -5, WeightPresent: true}}) {
				t.Fatalf("new transfer lost signed weight/count: %+v", other)
			}
		}
		if !reflect.DeepEqual(gotWorn, worn) || !reflect.DeepEqual(gotPack, wantPack) {
			t.Fatalf("weight roles changed: worn=%+v pack=%+v", gotWorn, gotPack)
		}
	}
	f := currentPoolFixtureFront(t, 91, 92)
	diagnostic, _, err := ResumeOriginalSave(f.Archives.Containers, payload, nil, mapload.DifficultyNormal, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	assert(diagnostic.World, false)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9999.sav"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	app := f.App("1109-signed-nonparty")
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9999.sav")
	assert(f.live.world, false)
	if err := f.live.world.MoveCarried(poolEntity(t, f.live.world, 91).ID, poolEntity(t, f.live.world, 92).ID, 0x0e06, 2); err != nil {
		t.Fatal(err)
	}
	assert(f.live.world, true)
	hash := f.live.world.Hash()
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("save"); err != nil {
		t.Fatal(err)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
		t.Fatalf("menu SAVE: %v %v: %s", entries, err, app.HeadlessMessage())
	}
	fresh := currentPoolFixtureFront(t, 91, 92)
	app2 := fresh.App("1109-fresh-native")
	s2, l2, load2 := fresh.SaveSeams(store, OriginalStore{}, nil)
	app2.SetSaveSeams(s2, l2, load2)
	groundAppLoad(t, app2, l2, entries[0].Name)
	assert(fresh.live.world, true)
	if fresh.live.world.Hash() != hash {
		t.Fatal("ordinary SAVE/fresh LOAD changed signed holdings")
	}
}
