package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func buyTrainingItem(t *testing.T, f *FrontEnd) {
	t.Helper()
	f.Shop = NewShop(1000)
	// A plain item: the synthetic table has no definition row for an
	// equipment code, and the SAV writer constructs a bought item from it.
	f.Shop.shelves[ShelfArmour] = []ShopItem{{Code: 0x0e01, Kind: 1, Count: 1, Price: 10, Weight: 1, WeightPresent: true}}
	s := f.TownScreen().(*townScreen)
	s.room, s.shopMember = roomShop, 0
	takeFromShelf(s, roomArmour)
	want := fmt.Sprintf("bought for 10 gold; you have %d left", f.Town.Gold()-10)
	if action := click(s, ui.ShopControlButton, 1); action.Msg != want {
		t.Fatal(action.Msg)
	}
}

func reloadTrainingCity(t *testing.T, f *FrontEnd) {
	t.Helper()
	party, graph, gold := mapload.CloneParty(f.Carried), f.Town.cityObjects.Clone(), f.Town.Gold()
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || filepath.Ext(name) != ".sav" {
		t.Fatalf("ordinary SAVE after native mutation: %q %v", name, err)
	}
	raw, err := os.ReadFile(filepath.Join(store.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(party, f.Carried) || !reflect.DeepEqual(graph, f.Town.cityObjects) || gold != f.Town.Gold() {
		t.Fatal("ordinary SAVE changed the live party, graph or purse")
	}
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range party {
		if want, _, ok := currentCityHuman(member); ok {
			found := false
			for _, c := range p.Roster() {
				if c.Name != member.Name {
					continue
				}
				human, err := p.Human(c.Identity)
				if err != nil {
					t.Fatal(err)
				}
				if got := cityHumanState(c, human); got != want {
					t.Fatalf("ordinary Human differs from current fields for %s\ngot %+v\nwant %+v", member.ID, got, want)
				}
				found = true
			}
			if !found {
				t.Fatal("ordinary Human is absent", member.ID)
			}
		}
	}
	fresh := &FrontEnd{InstallResources: f.InstallResources, RuntimeServices: f.RuntimeServices}
	if _, town, err := fresh.RestoreOriginal(raw); err != nil || !town {
		t.Fatalf("SAV reload: %t %v", town, err)
	}
	if fresh.Town.Gold() != gold || !reflect.DeepEqual(fresh.Town.cityObjects, graph) {
		t.Fatalf("cold SAV changed the purse or identity graph: gold %d/%d\ngraph before%+v\ngraph after%+v", gold, fresh.Town.Gold(), graph, fresh.Town.cityObjects)
	}
	for _, before := range party {
		after := trainingPartyMember(t, fresh, before.ID)
		if before.Hero != after.Hero || before.Book != after.Book || before.KnownSpells != after.KnownSpells ||
			!reflect.DeepEqual(before.Carry, after.Carry) || !reflect.DeepEqual(before.Saved, after.Saved) ||
			!reflect.DeepEqual(mapload.MemberItemEquipment(before, f.Table), mapload.MemberItemEquipment(after, fresh.Table)) {
			t.Fatal("cold SAV changed current Human, Item or book fields", before.ID)
		}
	}
	*f = *fresh
}

func TestTrainedCityOrdinaryPurchaseKeepsSourceSchool(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "current", true: "legacy"}[legacy], func(t *testing.T) {
			f := city1099Front(t, city1099Fixture(t))
			baseline := mapload.CloneParty(f.Carried)
			buyTrainingItem(t, f)
			bought := trainingPartyMember(t, f, "hero")
			items := mapload.MemberCarriedItems(bought, f.Table)
			want := currentCitySaleHuman(t, baseline[0])
			want.InventoryWeight++
			if f.Town.Gold() != 2990 || len(items) != 1 ||
				!reflect.DeepEqual(items[0], sim.ItemInstance{Code: 0x0e01, Kind: 1, Price: 10, Weight: 1, WeightPresent: true}) ||
				currentCitySaleHuman(t, bought) != want {
				t.Fatal("purchase changed more than the exact Item, accumulator and purse", items, currentCitySaleHuman(t, bought))
			}
			for _, binding := range f.originalCity.bindings {
				for _, prior := range baseline {
					if binding.partyID == prior.ID && !reflect.DeepEqual(binding.baseline, prior) {
						t.Fatal("purchase mutated the source baseline", prior.ID)
					}
				}
			}
			if legacy {
				// Removing the older city-export basis does not retire the
				// independent source actor retained in LiveLoad.
				s, _, err := f.Snapshot(false)
				if err != nil {
					t.Fatal(err)
				}
				for i := range s.Party {
					s.Party[i].OriginalHuman = nil
				}
				for i := range s.OriginalCity.Bindings {
					s.OriginalCity.Bindings[i].Baseline.OriginalHuman = nil
				}
				if _, _, err := f.Restore(s); err != nil {
					t.Fatal(err)
				}
			}
			for step, want := range []struct {
				message string
				level   int32
				gold    int
			}{{"trained Blade to 12 for 471", 12, 2519}, {"trained Blade to 13 for 518", 13, 2001}} {
				if msg := train1099(t, f, "hero", 0); msg != want.message {
					t.Fatalf("Train %d: %q", step, msg)
				}
				member := trainingPartyMember(t, f, "hero")
				if member.Hero.Skill[1] != want.level || f.Town.Gold() != want.gold ||
					!reflect.DeepEqual(mapload.MemberCarriedItems(member, f.Table), items) {
					t.Fatalf("Train %d lost native skill, purse or purchased item", step)
				}
				reloadTrainingCity(t, f)
			}
		})
	}
}

func TestTrainedCityPurchaseAndTrainingUseCurrentHumanOverRetainedFields(t *testing.T) {
	f := city1099Front(t, city1099Fixture(t))
	if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 12 for 471" {
		t.Fatal(msg)
	}
	buyTrainingItem(t, f)
	if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 13 for 518" || f.Town.Gold() != 2001 {
		t.Fatalf("native continuation after source training: %q gold=%d", msg, f.Town.Gold())
	}
	reloadTrainingCity(t, f)
	want := currentCitySaleHuman(t, trainingPartyMember(t, f, "hero"))
	f.Carried[0].OriginalHuman.State.Modifier.Attack.Tail[0]++
	if msg := train1099(t, f, "hero", 0); msg != "trained Blade to 14 for 570" || f.Town.Gold() != 1431 {
		t.Fatal("stale retained modifier displaced current training", msg, f.Town.Gold())
	}
	if got := currentCitySaleHuman(t, trainingPartyMember(t, f, "hero")); got.Modifier != want.Modifier || got.Attack.Skill[1] != 14 || got.Base.Skill[1] != 12 {
		t.Fatal("training used retained modifier values", got)
	}
	reloadTrainingCity(t, f)
}
