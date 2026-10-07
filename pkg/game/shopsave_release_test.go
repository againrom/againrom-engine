package game

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseShopStagedGoodsF2CurrentSAV(t *testing.T) {
	for _, original := range []bool{false, true} {
		t.Run(map[bool]string{false: "current city", true: "original city"}[original], func(t *testing.T) {
			var f *FrontEnd
			if original {
				f = releaseFront(t)
				_, raw := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
				if _, city, err := f.RestoreOriginal(raw); err != nil || !city {
					t.Fatal("original city LOAD", city, err)
				}
			} else {
				f = currentTown(t, func(f *FrontEnd, hero sim.EntityID) { equipmentReturnPick(t, f, hero, 20, 65) }, nil)
				screen := currentTownShop(f, "hero")
				before := mapload.MemberCarriedItems(f.Carried[screen.shopMemberIndex()], f.Table)
				gold := f.Town.Gold()
				screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlTableCell})
				if len(f.Shop.table) != 1 {
					t.Fatal("fresh city did not stage owned goods")
				}
				cold := currentTownReload(t, currentTownSave(t, f))
				var member *mapload.PartyMember
				for i := range cold.Carried {
					if cold.Carried[i].ID == "hero" {
						member = &cold.Carried[i]
					}
				}
				if member == nil || len(mapload.MemberCarriedItems(*member, cold.Table)) != len(before) || cold.Town.Gold() != gold {
					t.Fatal("first city SAVE lost fresh staged goods")
				}
				f = cold
			}
			store := SaveStore{Dir: t.TempDir()}
			save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
			if _, err := save(false); err != nil {
				t.Fatal(err)
			}
			app := f.App("shop SAVE")
			f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
			if err := headlessOpenLoad(app); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
				t.Fatal("city LOAD", err)
			}
			screen := f.TownScreen().(*townScreen)
			screen.room, screen.shopMember, screen.packBase = roomShop, 0, 0
			if len(f.Carried) < 2 || len(screen.shopPackStacks()) == 0 {
				t.Fatal("installed fixture lacks two heroes or owned pack goods")
			}
			before, gold := mapload.CloneParty(f.Carried), f.Town.Gold()
			beforeRoots := f.Town.cityObjects.Clone()
			item := screen.shopPackStacks()[0]
			screen.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlTableCell})
			if len(f.Shop.table) != 1 || !f.Shop.table[0].Mine || f.Shop.table[0].Count != 1 {
				t.Fatal("pack drag did not stage one owned unit")
			}
			screen.shopMember = 1
			staged := cloneShopMutation(f.Shop)
			for cycle := 0; cycle < 2; cycle++ {
				if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
					t.Fatal("shop F2 lost SAVE", err, app.Screen())
				}
				label := "pending-" + string(rune('a'+cycle))
				if err := app.HeadlessSaveEdit(store.Dir, label, ui.SaveSAV); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessSaveAction("save"); err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(filepath.Join(store.Dir, label+".sav"))
				if err != nil {
					t.Fatal(err)
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil || doc.Head.Mission != 0 || doc.World != nil {
					t.Fatal("shop SAVE is not ordinary city SAV", err)
				}
				cold := currentTownReload(t, raw)
				if cold.Town.Gold() != gold || len(cold.Shop.table) != 0 || !reflect.DeepEqual(staged, cloneShopMutation(f.Shop)) {
					t.Fatal("SAVE paid for goods, left a cold trade, or changed the live table")
				}
				for i := range before {
					var want, got uint32
					for _, stack := range cityMemberStacks(before[i], f.Table) {
						if stack.Code == item.Code {
							want += stack.Count
						}
					}
					for _, stack := range cityMemberStacks(cold.Carried[i], cold.Table) {
						if stack.Code == item.Code {
							got += stack.Count
						}
					}
					if got != want {
						t.Fatal("SAV changed staged ownership quantity", i, got, want)
					}
					if i > 0 && !slices.Equal(beforeRoots.Roots[i].Pack, cold.Town.cityObjects.Roots[i].Pack) {
						t.Fatal("selected other hero received staged goods")
					}
				}
				coldScreen := cold.TownScreen().(*townScreen)
				coldScreen.room, coldScreen.shopMember = roomShop, 0
				if action := coldScreen.shopFromPack(0, true); len(cold.Shop.table) == 0 {
					t.Fatal("next cold action", action)
				}
				if action := coldScreen.shopClear(); action.Msg != "the table is cleared" {
					t.Fatal(action)
				}
				if err := app.HeadlessGameMenuAction("return"); err != nil {
					t.Fatal(err)
				}
			}
			if err := app.HeadlessKey("f2"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(staged, cloneShopMutation(f.Shop)) || f.Town.Gold() != gold {
				t.Fatal("SAVE cancel changed the pending trade")
			}
			t.Logf("%s: staged owner goods survive two F2 SAVs, cold LOAD and next action; live trade and gold unchanged", os.Getenv("AGAINROM_ASSETS"))
		})
	}
}
