package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestReleaseMission81PlacedScrollKeepsDescriptionAcrossSAV(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.NextParty()
	prepareAcceptedCampaignMission(t, f, 81)
	f.Carried = mapload.CloneParty(party)
	app := f.App("placed scroll")
	app.Layout(1024, 768)
	quest81OK(t, app.OpenMission(f.MissionOpenerWith(81, party)))
	live := f.live
	hero := live.mission.ids[0]
	var source sim.ItemInstance
	var cell sim.CellPoint
	found := 0
	for _, sack := range live.world.Sacks() {
		for _, item := range sack.ItemInstances {
			if spell, _, ok := sim.ScrollSpell(item); ok && spell == 5 {
				source, cell = item, sim.CellPoint{X: sack.X, Y: sack.Y}
				found++
			}
		}
	}
	if found != 1 {
		t.Fatalf("mission81 protection scrolls=%d, want1", found)
	}
	if source.Code != 0x0e10 || source.Kind != 4 || source.Price != 4096 || !slices.Equal(source.Effects, []sim.ItemEffect{{Kind: 41, Operand: 5 | 99<<16}}) {
		t.Fatal("installed mission81 scroll differs from its researched tuple and general value", source)
	}
	t.Logf("fresh mission81 quantity1 fullItem%+v pricingSpell%v pricingMagicItem%v recomputed%d", source,
		f.Table.Spells.EntryParams(5), f.Table.MagicItems.EntryParams(int(uint8(source.Code))), mapload.RepriceItemInstance(source, f.Table))
	for _, e := range live.world.Entities() {
		if e.Alive() && live.world.Relations().Hostile(e.Owner, sim.SelfSlot) && max(e.X-cell.X, cell.X-e.X, e.Y-cell.Y, cell.Y-e.Y) <= 5 {
			quest81OK(t, live.world.HeadlessKill(e.ID))
		}
	}
	livePlaceAndWalk(t, live, hero, cell.X, cell.Y)
	f.LiveAdvance(1)
	quest81Dismiss(t, f, app)
	inspectionCentre(live, int(cell.X), int(cell.Y))
	quest81OK(t, app.HeadlessSelectEntity(uint32(hero)))
	quest81OK(t, app.HeadlessKey("grab"))
	check := func(g *FrontEnd, a *ui.App) []string {
		t.Helper()
		inspectionCentre(g.live, int(cell.X), int(cell.Y))
		quest81OK(t, a.HeadlessSelectEntity(uint32(hero)))
		g.live.push()
		stacks, _ := g.live.world.CarriedStacks(hero)
		index := slices.IndexFunc(stacks, func(v sim.ItemStack) bool {
			return v.Code == source.Code && v.Kind == source.Kind && v.Price == source.Price && slices.Equal(v.Effects, source.Effects)
		})
		if index < 0 || index >= len(g.live.invSubject.PackInfo) {
			t.Fatal("App Grab or cold LOAD lost the placed scroll value", stacks)
		}
		lines := g.live.invSubject.PackInfo[index]
		_, power, _ := sim.ScrollSpell(source)
		rule := mapload.SpellRules(g.Table)[4]
		want := []string{
			fmt.Sprintf("%s: %d", g.Words.Hover[spellLabelRange], int(rule.MaxRange)+int(power)/30),
			fmt.Sprintf("%s: +%d", g.Words.Hover[183], min(power/2, 100)),
		}
		for _, line := range want {
			if !slices.Contains(lines, line) {
				t.Fatalf("placed scroll info %q lacks independently expected %q", lines, line)
			}
		}
		openMissionPack(t, a)
		x, y, err := a.HeadlessPackCellPoint(index)
		quest81OK(t, err)
		a.SetTooltipDelayPreference(0, nil)
		quest81OK(t, a.HeadlessPointer("hover", x, y))
		state, pic := a.HeadlessTooltip()
		drawn, _, ok := ui.ComposeTooltipHint(lines, g.tipFont(), image.Pt(x, y), image.Rect(0, 0, 1024, 768), g.HoverBall(), g.HoverHelp())
		if !state.Visible || pic == nil || !ok || !bytes.Equal(pic.Pix, drawn.Pix) {
			t.Fatal("mission pack hover did not draw the scroll characteristics", state, lines)
		}
		return lines
	}
	before := check(f, app)
	store, _, _ := campaignSave(t, f, true)
	cold, coldApp := campaignCold(t, store)
	if after := check(cold, coldApp); !slices.Equal(before, after) {
		t.Fatal("cold LOAD changed scroll description", before, after)
	}
	quest81Dismiss(t, cold, coldApp)
	finishLoadedCycle(t, cold.live, hero)
	livePlaceAndWalk(t, cold.live, hero, 40, 24)
	cold.LiveAdvance(1)
	quest81Dismiss(t, cold, coldApp)
	quest81OK(t, coldApp.HeadlessKey("grab"))
	if !quest81Carries(cold.live.world, hero, 0x0e2c) {
		t.Fatal("the installed sword sack did not supply the quest item")
	}
	finishLoadedCycle(t, cold.live, hero)
	livePlaceAndWalk(t, cold.live, hero, 60, 62)
	cold.LiveAdvance(32)
	if cold.live.world.Outcome() != sim.OutcomeWon {
		t.Fatal("the installed sword handover did not finish mission81")
	}
	_, _, err := cold.LiveCompleteCampaign()
	quest81OK(t, err)
	townStore, _, _ := campaignSave(t, cold, false)
	town, townApp := campaignCold(t, townStore)
	quest81OK(t, townApp.HeadlessActivate("SHOP"))
	screen := registerTownFront(town.TownScreen().(*townScreen), town)
	screen.CloseTip()
	index := slices.IndexFunc(screen.shopPackStacks(), func(v sim.ItemStack) bool {
		return v.Code == source.Code && v.Kind == source.Kind && v.Price == source.Price && slices.Equal(v.Effects, source.Effects)
	})
	if index < 0 {
		t.Fatal("mission return lost the placed scroll", screen.shopPackStacks())
	}
	packCell := shopScrollPackCell(t, townApp, screen, index)
	if view := screen.ShopScreen(); view.Pack[packCell].Price != shopHalfUp(source.Price) || !slices.Equal(view.Pack[packCell].Info, before) {
		t.Fatal("town pack changed the placed scroll's sell display or description", view.Pack[packCell])
	}
	cityShopPointer(t, townApp, "pack", packCell, "press", "release")
	places := town.Shop.Table()
	if len(places) != 1 || !places[0].Mine || places[0].ShopItem.Price != source.Price || places[0].ShopItem.Count != 1 {
		t.Fatal("actual placed scroll did not reach the sale table unchanged", places)
	}
	sold := places[0].ShopItem.Instance()
	gold := town.Town.Gold()
	cityShopPointer(t, townApp, "button", 2, "press", "release")
	if town.Town.Gold()-gold != int(shopHalfUp(source.Price)) || len(town.Shop.Table()) != 0 {
		t.Fatal("placed scroll payout did not use its stored price", town.Town.Gold()-gold, town.Shop.Table())
	}
	cityShopPointer(t, townApp, "shelf_pick", int(ShelfBooks), "press", "release")
	stock := town.Shop.Shelf(ShelfBooks)
	at := slices.IndexFunc(stock, func(v ShopItem) bool { return staffSaleValueEqual(v.Instance(), sold) })
	if at < 0 {
		t.Fatal("sold placed scroll did not reach the consumable shelf", sold)
	}
	for tries := 0; at < screen.shelfBase || at >= screen.shelfBase+len(screen.ShopScreen().Shelf); tries++ {
		if tries >= 64 {
			t.Fatal("ordinary wheel did not expose the placed scroll for buy display")
		}
		edge := "wheel-down"
		if at < screen.shelfBase {
			edge = "wheel-up"
		}
		cityShopPointer(t, townApp, "shelf", 0, edge)
	}
	if shown := screen.ShopScreen().Shelf[at-screen.shelfBase]; shown.Price != source.Price {
		t.Fatal("placed scroll buy display changed the stored price", shown)
	}
	t.Logf("mission81 cell(%d,%d) placed scroll=%+v; buy%d sell%d; info%q", cell.X, cell.Y, source, source.Price, shopHalfUp(source.Price), before)
}

func shopScrollPackCell(t *testing.T, app *ui.App, screen *townScreen, index int) int {
	t.Helper()
	for tries := 0; index < screen.packBase || index >= screen.packBase+len(screen.ShopScreen().Pack); tries++ {
		if tries >= 64 {
			t.Fatal("ordinary wheel could not expose pack item", index)
		}
		edge := "wheel-down"
		if index < screen.packBase {
			edge = "wheel-up"
		}
		cityShopPointer(t, app, "pack", 0, edge)
	}
	return index - screen.packBase
}

func TestReleaseShopEnchantedSalesKeepMagicShelfAndValues(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for _, restored := range []bool{false, true} {
		t.Run(fmt.Sprintf("restored-%v", restored), func(t *testing.T) {
			app, screen := releaseShopApp(t)
			f := frontOf(screen)
			m := releaseMissionMap(t, f, 70)
			mission, err := StartMissionFrom(m, "placed equipment", 70, f.Table, mapload.DifficultyNormal, nil)
			quest81OK(t, err)
			var items []sim.ItemInstance
			for _, code := range []uint16{0x8134, 0x4461} {
				var item sim.ItemInstance
				for _, sack := range mission.World.Sacks() {
					for _, placed := range sack.ItemInstances {
						if placed.Code == code {
							item = placed
						}
					}
				}
				if item.Code != code || !item.HasEnchantment() {
					t.Fatalf("mission70 has no enchanted %#x", code)
				}
				items = append(items, item, mapload.ItemInstanceFromCode(code, f.Table))
			}
			generated := NewShop(1_000_000)
			generated.Generate(f.Table, 81)
			bodyArmour := slices.IndexFunc(generated.Shelf(ShelfMagic), func(v ShopItem) bool {
				return v.Code.B() == 5 && v.Instance().HasEnchantment()
			})
			if bodyArmour < 0 {
				t.Fatal("installed shop generator produced no enchanted body armour")
			}
			armour := generated.Shelf(ShelfMagic)[bodyArmour].Instance()
			items = append(items, armour, mapload.ItemInstanceFromCode(armour.Code, f.Table))
			var stacks []sim.ItemStack
			for _, item := range items {
				stacks = append(stacks, sim.StackItem(item, 3))
			}
			cityShopSetTestPack(t, f, 0, stacks...)
			if restored {
				cityShopGraph(t, f)
				store, _, _ := campaignSave(t, f, false)
				f, app = campaignCold(t, store)
				quest81OK(t, app.HeadlessActivate("SHOP"))
				screen = registerTownFront(f.TownScreen().(*townScreen), f)
			} else {
				f.Town.cityObjects = nil
			}
			screen.CloseTip()
			packCell := func(index int) int {
				return shopScrollPackCell(t, app, screen, index)
			}
			for _, item := range items {
				index := slices.IndexFunc(screen.shopPackStacks(), func(v sim.ItemStack) bool { return staffSaleValueEqual(v.Instance(), item) })
				if index < 0 {
					t.Fatal("prepared or restored pack lost installed item", item)
				}
				before := staffSaleObserve(screen)
				cityShopPointer(t, app, "pack", packCell(index), "shift-press", "shift-release")
				if placed := f.Shop.Table(); len(placed) != 1 || placed[0].ShopItem.Count != 3 || !staffSaleValueEqual(placed[0].ShopItem.Instance(), item) {
					t.Fatal("App pack staging lost value or count", placed)
				}
				cityShopPointer(t, app, "button", 2, "press", "release")
				after := staffSaleObserve(screen)
				if after.Gold-before.Gold != int(shopHalfUp(3*item.Price)) || len(after.Table) != 0 {
					t.Fatal("sale changed payout or table", before.Gold, after.Gold, after.Table)
				}
				wantShelf := shopShelfFor(data.ItemCode(item.Code))
				if item.HasEnchantment() {
					wantShelf = ShelfMagic
				}
				old, next := staffSaleCounts(before, item, true), staffSaleCounts(after, item, true)
				for shelf := range old {
					want := int64(0)
					if ShopShelf(shelf) == wantShelf {
						want = 3
					}
					if next[shelf]-old[shelf] != want {
						t.Fatalf("sale %#x effects%v shelf%s delta%d want%d", item.Code, item.Effects, ShopShelf(shelf), next[shelf]-old[shelf], want)
					}
				}
				cityShopPointer(t, app, "shelf_pick", int(wantShelf), "press", "release")
				stock := f.Shop.Shelf(wantShelf)
				at := slices.IndexFunc(stock, func(v ShopItem) bool { return staffSaleValueEqual(v.Instance(), item) })
				if at < 0 {
					t.Fatal("destination shelf lost sold item", item)
				}
				for attempts := 0; at < screen.shelfBase || at >= screen.shelfBase+len(screen.ShopScreen().Shelf); attempts++ {
					if attempts >= 64 {
						t.Fatal("ordinary wheel could not expose sold item", at)
					}
					edge := "wheel-down"
					if at < screen.shelfBase {
						edge = "wheel-up"
					}
					cityShopPointer(t, app, "shelf", 0, edge)
				}
				cityShopPointer(t, app, "shelf", at-screen.shelfBase, "press", "release")
				gold := f.Town.Gold()
				cityShopPointer(t, app, "button", 1, "press", "release")
				if gold-f.Town.Gold() != int(item.Price) || len(f.Shop.Table()) != 0 {
					t.Fatal("repurchase changed price or table", gold-f.Town.Gold(), f.Shop.Table())
				}
				bought := slices.IndexFunc(screen.shopPackStacks(), func(v sim.ItemStack) bool { return staffSaleValueEqual(v.Instance(), item) && v.Count == 1 })
				if bought < 0 {
					t.Fatal("repurchase lost exact value or count", screen.shopPackStacks())
				}
				t.Logf("restored%v sold %#x effects%v x3 price%d payout%d shelf%s", restored, item.Code, item.Effects, item.Price, after.Gold-before.Gold, wantShelf)
			}
			gold := f.Town.Gold()
			store, _, _ := campaignSave(t, f, false)
			f, app = campaignCold(t, store)
			if f.Town.Gold() != gold {
				t.Fatal("post-trade SAV changed purse", f.Town.Gold(), gold)
			}
			quest81OK(t, app.HeadlessActivate("SHOP"))
			screen = registerTownFront(f.TownScreen().(*townScreen), f)
			screen.CloseTip()
			for _, item := range items {
				index := slices.IndexFunc(screen.shopPackStacks(), func(v sim.ItemStack) bool { return staffSaleValueEqual(v.Instance(), item) && v.Count == 1 })
				if index < 0 {
					t.Fatal("post-trade SAV lost the repurchased value or quantity", item, screen.shopPackStacks())
				}
				before := staffSaleObserve(screen)
				cityShopPointer(t, app, "pack", packCell(index), "press", "release")
				cityShopPointer(t, app, "button", 2, "press", "release")
				after := staffSaleObserve(screen)
				if after.Gold-before.Gold != int(shopHalfUp(item.Price)) || len(after.Table) != 0 {
					t.Fatal("next sale after cold LOAD changed payment or tray", before.Gold, after.Gold, after.Table)
				}
				wantShelf := shopShelfFor(data.ItemCode(item.Code))
				if item.HasEnchantment() {
					wantShelf = ShelfMagic
				}
				old, next := staffSaleCounts(before, item, true), staffSaleCounts(after, item, true)
				for shelf := range old {
					want := int64(0)
					if ShopShelf(shelf) == wantShelf {
						want = 1
					}
					if next[shelf]-old[shelf] != want {
						t.Fatal("next sale after cold LOAD changed destination", item, old, next)
					}
				}
			}
		})
	}
}

func TestReleaseOwnerShopScrollStoredValues(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	dir := os.Getenv("AGAINROM_SHOP_SCROLL_OWNER_DIR")
	if dir == "" {
		t.Skip("AGAINROM_SHOP_SCROLL_OWNER_DIR is required")
	}
	for _, name := range []string{"magicscroll.sav", "citywithscroll.sav", "beforesword.sav", "gog/game0033.sav", "gog/game0032.sav", "gog/game9999.sav"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if os.IsNotExist(err) {
				t.Skip("optional owner input is absent")
			}
			quest81OK(t, err)
			f := releaseFront(t)
			input := t.TempDir()
			quest81OK(t, os.WriteFile(filepath.Join(input, filepath.Base(name)), raw, 0600))
			f.SetDeterministicFrames(true)
			app := f.App("scroll input")
			app.Layout(1024, 768)
			save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: input}, nil)
			app.SetSaveSeams(save, list, load)
			rows := list()
			if len(rows) != 1 {
				t.Fatal("owner input is absent from the load list", rows)
			}
			quest81OK(t, headlessOpenLoad(app))
			quest81OK(t, app.HeadlessActivate(rows[0].Label))
			if app.Screen() != ui.ScreenTown && app.Screen() != ui.ScreenMap {
				t.Fatal("owner input LOAD failed", app.Screen(), app.HeadlessMessage())
			}
			var observed []sim.ItemInstance
			check := func(g *FrontEnd) {
				t.Helper()
				var found []sim.ItemInstance
				if g.live != nil && g.live.world != nil {
					for _, entity := range g.live.world.Entities() {
						items, _ := g.live.world.CarriedItems(entity.ID)
						found = append(found, items...)
					}
					for _, sack := range g.live.world.Sacks() {
						found = append(found, sack.ItemInstances...)
					}
				} else {
					for _, member := range g.Carried {
						found = append(found, mapload.MemberCarriedItems(member, g.Table)...)
					}
				}
				var scrolls []sim.ItemInstance
				for _, item := range found {
					spell, power, ok := sim.ScrollSpell(item)
					if !ok {
						continue
					}
					lines := itemInstanceInfoLines(item, g.Table, g.Words)
					if len(lines) <= 1 {
						t.Fatal("loaded owner scroll lacks spell characteristics", item, lines)
					}
					t.Logf("screen%s scroll code%#x spell%d power%d price%d payout%d info%q", app.Screen(), item.Code, spell, power, item.Price, shopHalfUp(item.Price), lines)
					t.Logf("fullItem%+v pricingSpell%v pricingMagicItem%v recomputed%d", item,
						g.Table.Spells.EntryParams(int(spell)), g.Table.MagicItems.EntryParams(int(uint8(item.Code))), mapload.RepriceItemInstance(item, g.Table))
					scrolls = append(scrolls, item)
				}
				if len(scrolls) == 0 {
					t.Log("no scroll found in loaded actor holdings or ground sacks")
				}
				if observed != nil {
					if len(observed) != len(scrolls) {
						t.Fatal("owner scroll count changed across SAVE", len(observed), len(scrolls))
					}
					for i := range observed {
						if !staffSaleValueEqual(observed[i], scrolls[i]) {
							t.Fatal("SAVE or cold LOAD changed owner scroll value", observed[i], scrolls[i])
						}
					}
				}
				observed = scrolls
			}
			check(f)
			store, _, _ := campaignSave(t, f, f.live != nil && f.live.world != nil)
			cold, _ := campaignCold(t, store)
			check(cold)
		})
	}
}
