package game

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cityPotionLoadApp(t *testing.T, raw []byte) (*FrontEnd, *ui.App, SaveStore) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(store.Dir, "potion.sav"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app := openLocalTownSAV(t, f, store.Dir, "potion.sav")
	t.Cleanup(app.StopAudio)
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	return f, app, store
}

func cityPotionPointer(t *testing.T, app *ui.App, target string, index int, edges ...string) {
	t.Helper()
	x, y, err := app.HeadlessShopPoint(target, index)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range edges {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
}

func cityPotionWire(t *testing.T, raw []byte, memberID string, effect *sim.ActiveEffect, regeneration int32) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World != nil {
		t.Fatal("ordinary city SAV", err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil {
		t.Fatal("current city policy", err)
	}
	var object uint16
	for _, p := range a.Party {
		if string(p.ID) != memberID {
			continue
		}
		if p.City == nil || !reflect.DeepEqual(p.City.Potion, effect) {
			t.Fatalf("ordinary SAVE lost current potion: %+v want %+v", p.City, effect)
		}
		for _, binding := range a.Bindings {
			if !binding.Structure && !binding.Missing && binding.ID == p.Entity {
				object = binding.Object
			}
		}
	}
	if object == 0 {
		t.Fatal("current potion member has no ordinary actor binding")
	}
	characters, err := sav.ReadDocumentCharacters(doc, []uint16{object})
	if err != nil || len(characters) != 1 || characters[0].Character.Basis == nil {
		t.Fatal("ordinary Human", err)
	}
	modifier := characters[0].Character.Basis.Human.Fields.Modifier
	if got := int32(int16(binary.LittleEndian.Uint16(modifier[10:]))); got != regeneration {
		t.Fatalf("ordinary Human modifier differs from changed live state: %d want %d", got, regeneration)
	}
}

func TestReleaseCityPotionF2SAVAndMissionExpiry(t *testing.T) {
	t.Run("native unmodified control", func(t *testing.T) {
		store := SaveStore{Dir: t.TempDir()}
		live, app := saveDialogArrivedTown(t, store.Dir)
		t.Cleanup(app.StopAudio)
		raw := cityRosterF2Save(t, app, store, "unmodified")
		cold, coldApp, _ := cityPotionLoadApp(t, raw)
		for i, f := range []*FrontEnd{live, cold} {
			if err := []*ui.App{app, coldApp}[i].OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
				t.Fatal(err)
			}
		}
		cityPotionMissionEqual(t, live.live.world, cold.live.world, 0)
	})
	for _, origin := range []string{"current arrival", "original city"} {
		t.Run(origin, func(t *testing.T) {
			var f *FrontEnd
			var app *ui.App
			store := SaveStore{Dir: t.TempDir()}
			if origin == "current arrival" {
				f, app = saveDialogArrivedTown(t, store.Dir)
				t.Cleanup(app.StopAudio)
				if f.originalCity != nil {
					t.Fatal("native arrival unexpectedly depends on a loaded city")
				}
			} else {
				_, raw := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
				f, app, store = cityPotionLoadApp(t, raw)
			}
			if err := app.HeadlessActivate("SHOP"); err != nil {
				t.Fatal(err)
			}
			shop := f.TownScreen().(*townScreen)
			for n := 0; shop.room == roomTalk && n < 32; n++ {
				if err := app.HeadlessActivate("dialogue"); err != nil {
					t.Fatal(err)
				}
			}
			if shop.room != roomShop {
				t.Fatal("shop did not open", shop.room)
			}
			member := shop.shopPartyMember(shop.shopMemberIndex())
			if member == nil || member.PotionEffect != nil {
				t.Fatal("fixture requires a member without an existing potion")
			}
			memberID := member.ID
			before, _, _ := mapload.PartyDisplayWithTable(*member, f.Table)
			stock := shopPotionStock(f.Table, rand.New(rand.NewSource(1)))
			if len(stock) == 0 {
				t.Fatal("installed shop has no potion stock")
			}
			f.Shop.shelves[ShelfBooks] = []ShopItem{stock[0]}
			f.Town.gold = 100000
			cityPotionPointer(t, app, "shelf_pick", 3, "press", "release")
			if shop.ShopScreen().Shelf[0].Icon == nil {
				t.Fatal("installed potion lacks shop art")
			}
			gold, quantity := f.Town.Gold(), f.Shop.Shelf(ShelfBooks)[0].Count
			packAt := releaseShopBuyToPack(t, app, f, uint16(stock[0].Code))
			if err := app.HeadlessPointer("press", packAt.X, packAt.Y); err != nil {
				t.Fatal(err)
			}
			cityPotionPointer(t, app, "doll_box", 0, "move", "release")
			member = shop.shopPartyMember(shop.shopMemberIndex())
			effect := member.PotionEffect
			if effect == nil || effect.Kind != sim.EffectHealthRegeneration || effect.Spell != 0 || effect.Magnitude != 100 || effect.Remaining != 960 {
				t.Fatalf("production potion use: %+v", effect)
			}
			if f.Town.Gold() != gold-int(stock[0].Price) || f.Shop.Shelf(ShelfBooks)[0].Count != quantity-1 {
				t.Fatal("potion use did not pay for exactly one merchant item")
			}
			want := before.HealthRegeneration + 100
			live, liveApp := f, app
			party := mapload.CloneParty(f.Carried)
			for cycle := 0; cycle < 2; cycle++ {
				raw := cityRosterF2Save(t, app, store, fmt.Sprintf("timed-%d", cycle))
				cityPotionWire(t, raw, memberID, effect, want)
				if !reflect.DeepEqual(live.Carried, party) {
					t.Fatal("city SAVE mutated the uninterrupted party")
				}
				cold, coldApp, coldStore := cityPotionLoadApp(t, raw)
				cityRosterSame(t, live, cold)
				for i, p := range cold.Carried {
					if !reflect.DeepEqual(p.PotionEffect, live.Carried[i].PotionEffect) {
						t.Fatal("cold city restarted or dropped a potion", p.ID)
					}
					if p.ID == memberID {
						d, _, _ := mapload.PartyDisplayWithTable(p, cold.Table)
						if d.HealthRegeneration != want {
							t.Fatal("cold city modifier", d.HealthRegeneration, want)
						}
					}
				}
				f, app, store = cold, coldApp, coldStore
			}
			for i, target := range []*FrontEnd{live, f} {
				missionApp := []*ui.App{liveApp, app}[i]
				if err := missionApp.OpenMission(target.MissionOpener(target.Town.Chapter())); err != nil {
					t.Fatal(err)
				}
			}
			var id sim.EntityID
			for i, p := range live.live.mission.party {
				if p.ID == memberID {
					id = live.live.mission.ids[i]
				}
			}
			if id == 0 {
				t.Fatal("next mission omitted the potion owner")
			}
			for tick := 0; tick <= 961; tick++ {
				cityPotionMissionEqual(t, live.live.world, f.live.world, tick)
				for _, target := range []*FrontEnd{live, f} {
					e, ok := target.live.entity(id)
					expected := want
					if tick >= 960 {
						expected = before.HealthRegeneration
					}
					if !ok || e.HealthRegeneration != expected {
						t.Fatalf("tick %d regeneration=%d want %d", tick, e.HealthRegeneration, expected)
					}
					count := 0
					for _, current := range target.live.world.ActiveEffects() {
						if current.Target == id && current.Spell == 0 {
							count++
							if tick >= 960 || current.Kind != effect.Kind || current.Magnitude != 100 || current.Remaining != uint16(960-tick) {
								t.Fatalf("tick %d potion attachment %+v", tick, current)
							}
						}
					}
					if tick < 960 && count != 1 || tick >= 960 && count != 0 {
						t.Fatal("potion duplicated or failed to expire", tick, count)
					}
				}
				if tick < 961 {
					sim.Step(live.live.world, nil)
					sim.Step(f.live.world, nil)
				}
			}
			t.Logf("%s: paid one potion, two F2 city SAV cycles, mission %d, +100 regeneration through tick 959, one expiry at 960, live/cold hashes equal through 961", origin, f.Town.Chapter())
		})
	}
}

func cityPotionMissionEqual(t *testing.T, live, cold *sim.World, tick int) {
	t.Helper()
	if live.Hash() == cold.Hash() {
		return
	}
	wantEntities, gotEntities := live.Entities(), cold.Entities()
	if len(wantEntities) == len(gotEntities) {
		for i, want := range wantEntities {
			wantValue, gotValue := reflect.ValueOf(want), reflect.ValueOf(gotEntities[i])
			for field := 0; field < wantValue.NumField(); field++ {
				if wantValue.Field(field).CanInterface() && !reflect.DeepEqual(wantValue.Field(field).Interface(), gotValue.Field(field).Interface()) {
					t.Logf("actor %d %s: live=%+v cold=%+v", want.ID, wantValue.Type().Field(field).Name, wantValue.Field(field).Interface(), gotValue.Field(field).Interface())
				}
			}
		}
	}
	t.Logf("live effects=%+v cold=%+v", live.ActiveEffects(), cold.ActiveEffects())
	for _, field := range []struct {
		name       string
		live, cold any
	}{
		{"stock", live.Stock(), cold.Stock()},
		{"actions", live.Actions(), cold.Actions()},
		{"policy", live.CurrentPolicy(), cold.CurrentPolicy()},
		{"saved objects", live.SavedObjects(), cold.SavedObjects()},
		{"sacks", live.Sacks(), cold.Sacks()},
		{"item weights", live.ItemWeights(), cold.ItemWeights()},
		{"script", live.Script(), cold.Script()},
		{"cells", live.SavedCellRecords(), cold.SavedCellRecords()},
	} {
		if !reflect.DeepEqual(field.live, field.cold) {
			a, b := fmt.Sprintf("%+v", field.live), fmt.Sprintf("%+v", field.cold)
			at := 0
			for at < len(a) && at < len(b) && a[at] == b[at] {
				at++
			}
			t.Logf("%s differ at %d (lengths %d/%d): live=%s cold=%s", field.name, at, len(a), len(b), a[max(0, at-80):min(len(a), at+400)], b[max(0, at-80):min(len(b), at+400)])
		}
	}
	a, aerr := live.MarshalBinary()
	b, berr := cold.MarshalBinary()
	if aerr != nil || berr != nil {
		t.Fatal("mismatched World encoding", aerr, berr)
	}
	at := 0
	for at < len(a) && at < len(b) && a[at] == b[at] {
		at++
	}
	t.Logf("world bytes differ at %d (lengths %d/%d): live=%x cold=%x", at, len(a), len(b), a[max(0, at-16):min(len(a), at+32)], b[max(0, at-16):min(len(b), at+32)])
	t.Fatalf("uninterrupted and cold next mission differ at tick %d", tick)
}
