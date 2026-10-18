package game

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/secondshop"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// secondShopLaterTown places a new campaign in town 2 at stage 30 or town 3
// at stage 60, with the shop records every earlier departure writes
// (R2-ENGINE-330), and the clock the scenes read under the test's control.
func secondShopLaterTown(t *testing.T, id int) (*FrontEnd, *ui.App, *secondCampaignScreen, *time.Time) {
	t.Helper()
	var f *FrontEnd
	var app *ui.App
	var s *secondCampaignScreen
	var now *time.Time
	if id == 2 {
		f, app, s, now = secondTownTwo(t)
	} else {
		f, app, s, now = secondTownThree(t, 60)
	}
	c := f.Town.second
	for n := 10; n < int(c.bank[768]); n += 10 {
		c.bank[896+n] = 1
	}
	c.shop.records = replayRecords(c.bank)
	return f, app, s, now
}

// secondShopTownOne is a new campaign in town 1 with the scene clock under
// the test's control.
func secondShopTownOne(t *testing.T) (*FrontEnd, *ui.App, *secondCampaignScreen, *time.Time) {
	t.Helper()
	f := secondGameFront(t)
	now := time.Unix(100, 0)
	f.TownAnimationNow = func() time.Time { return now }
	app := f.App("first town shop")
	app.Layout(640, 480)
	s := f.TownScreen().(*secondCampaignScreen)
	app.SetTown(s)
	if err := app.HeadlessActivate("new game"); err != nil || !s.AtTownSquare() {
		t.Fatal("new game town 1", err)
	}
	s.TownSquareActive(true)
	return f, app, s, &now
}

// secondShopEnter opens the shop page from the square's row and runs its
// scene; the page's art must load.
func secondShopEnter(t *testing.T, f *FrontEnd, app *ui.App, s *secondCampaignScreen) (*town.Scene, *town.Art) {
	t.Helper()
	secondShopActivate(t, f, app, "SHOP")
	c := f.Town.second
	d, ok := s.composedTown()
	if !ok || c.room != secondTownShop || f.TownScreen().Header() != "ROM2 campaign: shop" {
		t.Fatalf("SHOP opened room %d", c.room)
	}
	secondShopArtLoaded(t, s)
	s.ShopInteriorActive(true)
	return s.scene(d, "shop"), s.roomArt(d, "shop").scene
}

// secondShopArtLoaded requires the shop page in place of the square, with
// the town's room, frame, shelf, table, menu, arrows and buttons
// (R2-ENGINE-249) and its scene's interior.
func secondShopArtLoaded(t *testing.T, s *secondCampaignScreen) {
	t.Helper()
	d, _ := s.composedTown()
	if a := s.roomArt(d, "shop"); a.err != nil || !s.AtTownShop() || s.AtTownSquare() {
		t.Fatal("the shop page's art did not load:", a.err)
	}
	v := s.ShopScreen()
	art := v.Art
	if art == nil || v.Scene == nil || v.Layout == nil {
		t.Fatal("the shop page has no art, scene or layout")
	}
	pics := []*image.RGBA{art.Room, art.Frame, art.Shelf, art.Table, art.Menu, art.Arrow[0], art.Arrow[1]}
	pics = append(pics, art.Button[:]...)
	for i, p := range pics {
		if p == nil {
			t.Fatalf("shop picture %d did not load", i)
		}
	}
}

// secondShopPaint is the shop page's frame with no pointer.
func secondShopPaint(s *secondCampaignScreen) *image.RGBA {
	return ui.ComposeShopScreen(s.ShopScreen(), image.Point{}, false, nil, false)
}

// secondShopRows reports whether the shown page offers category k: a
// pixel of the room answers its rectangle.
func secondShopRows(app *ui.App, k int) bool {
	_, _, err := app.HeadlessShopPoint("shelf_pick", k)
	return err == nil
}

// TestReleaseSecondTownShopLaterTowns opens the town 2 and town 3 shops from
// their squares, buys and sells at the claimed prices, saves with a deal
// open, and has a separate process LOAD the SAV: the gold, records and items
// are kept and the stock is drawn anew (DIV-3045). Town 3's fourth category is hidden and
// refused with bank slot 770 zero and shown once it is set (R2-ENGINE-259).
func TestReleaseSecondTownShopLaterTowns(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
	}{{"Kaarg", 2}, {"Druid", 3}} {
		t.Run(tc.name, func(t *testing.T) {
			secondGameRoot(t)
			if path := os.Getenv("AGAINROM_SECOND_SHOP_INPUT"); path != "" {
				secondShopColdProof(t, path, tc.id)
				return
			}
			f, app, s, _ := secondShopLaterTown(t, tc.id)
			c := f.Town.second
			if tc.id == 3 {
				secondShopDruidGate(t, f, app, s)
			}
			secondShopEnter(t, f, app, s)
			if c.shop.filled != tc.id || len(c.shop.shelves[0]) == 0 || len(c.shop.shelves[1]) == 0 || len(c.shop.shelves[2]) == 0 {
				t.Fatalf("town %d stock: filled %d, %d/%d/%d/%d entries", tc.id, c.shop.filled,
					len(c.shop.shelves[0]), len(c.shop.shelves[1]), len(c.shop.shelves[2]), len(c.shop.shelves[3]))
			}
			// Every equipment entry of a category without enchantment is
			// priced inside its record's range, rounding aside (R2-ENGINE-264,
			// R2-ENGINE-267): the town's own record filled it.
			for k, list := range c.shop.stock() {
				cat := c.shop.records[tc.id][k]
				for _, it := range list {
					if it.Kind != secondshop.Book && it.Kind != secondshop.Magic && cat.Mask&(1<<29) == 0 && (it.Price < int64(cat.Min) || it.Price > int64(cat.Max)+1) {
						t.Fatalf("category %d entry %+v outside its range %d to %d", k, it, cat.Min, cat.Max)
					}
				}
			}
			witness := map[int]string{2: "town2", 3: "town3"}[tc.id]
			secondShopWitness(t, app, witness+"-shop-open")
			f.Town.gold = 500000
			first := c.shop.shelf()[c.shop.visibleShelf()[0]]
			secondShopActivate(t, f, app, "ITEM 1", "BUY")
			if f.Town.gold != 500000-int(first.Price) || len(c.heroItems()) != 1 || c.heroItems()[0].Quantity != 1 {
				t.Fatalf("buy of one at %d: gold %d hero %v", first.Price, f.Town.gold, c.heroItems())
			}
			gold := f.Town.gold
			secondShopActivate(t, f, app, "MINE 1", "SELL")
			if f.Town.gold != gold+int(secondshop.SaleCredit(first.Price, 1)) || len(c.heroItems()) != 0 {
				t.Fatalf("sale of one at %d: gold %d -> %d", first.Price, gold, f.Town.gold)
			}
			secondShopActivate(t, f, app, "SHELF 2", "ITEM 1", "BUY", "SHELF 1", "ITEM 2")
			if c.shop.category != 0 || len(c.heroItems()) != 1 || len(c.shop.buying()) != 1 {
				t.Fatalf("second deal: hero %v buying %v", c.heroItems(), c.shop.buying())
			}
			secondShopWitness(t, app, witness+"-shop-deal-open")
			saved, savedGold, old := captureSecondCampaign(c), f.Town.gold, secondShopUndone(c.shop)
			if saved.Shop.Records != c.shop.records {
				t.Fatal("the captured records are not the campaign's")
			}
			dir := secondMissionSaveDirectory(t)
			raw := secondTownNamedSave(t, f, app, dir, "Shop")
			if actions := secondShopSaved(t, raw, saved); actions.Session.Second.Aux != nil {
				t.Fatal("the SAV carries the auxiliary array")
			}
			// Loss control: another town's records in the written shop are
			// not the shop the continuous run holds.
			lost := saved.Shop.clone()
			lost.Records[tc.id] = lost.Records[1]
			if reflect.DeepEqual(lost, saved.Shop) {
				t.Fatal("loss control compares equal")
			}
			proof, err := json.Marshal(secondShopProof{Shop: saved.Shop, Party: saved.Party, Gold: savedGold, Old: old})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, secondSaveFile(t, dir, "Shop"))
			if err := os.WriteFile(path+".json", proof, 0600); err != nil {
				t.Fatal(err)
			}
			runSpellWitnessChild(t, path, "AGAINROM_SECOND_SHOP_INPUT")
			if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare || len(c.shop.buying()) != 0 {
				t.Fatalf("escape left room %d with %d pending", c.room, len(c.shop.buying()))
			}
			t.Logf("town %d shop: %d/%d/%d/%d entries; bought at %d, sold for %d; SAV %d bytes", tc.id, len(old[0]),
				len(old[1]), len(old[2]), len(old[3]), first.Price, secondshop.SaleCredit(first.Price, 1), len(raw))
		})
	}
}

// secondShopDruidGate opens the druid shop with bank slot 770 zero: no
// fourth-category row, button or elven picture; the druid inn's NPC 675
// topic 78 then sets the slot and the class-4 mask (R2-ENGINE-331), and the
// reopened page offers and selects the fourth category under hili_elven.
func secondShopDruidGate(t *testing.T, f *FrontEnd, app *ui.App, s *secondCampaignScreen) {
	t.Helper()
	c := f.Town.second
	if c.bank[770] != 0 {
		t.Fatal("bank slot 770 is set at stage 60")
	}
	_, art := secondShopEnter(t, f, app, s)
	shut := secondShopPaint(s)
	if secondShopRows(app, 3) {
		t.Fatal("the fourth category is offered with bank slot 770 zero")
	}
	if shopTap(t, app, image.Pt(305, 224), "press", "release"); c.shop.category == 3 || s.shopChosen == 3 {
		t.Fatal("a click on the fourth category's rectangle chose it with bank slot 770 zero")
	}
	elven, elvenAt := art.Pictures("elven")[0], image.Pt(257, 172)
	p := druidDiffPoint(t, shut, elven, elvenAt)
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare {
		t.Fatal("escape", err)
	}
	// The mission 70 departure stores 1 in slot 770: the page offers the
	// fourth category before the conversation, and the stock drawn from the
	// record without class 4 holds nothing in it.
	c.bank[770] = 1
	secondShopEnter(t, f, app, s)
	if !secondShopRows(app, 3) || len(c.shop.shelves[3]) != 0 || c.shop.filled != 3 {
		t.Fatalf("slot 770 = 1 before the conversation: fourth category %d entries, filled %d", len(c.shop.shelves[3]), c.shop.filled)
	}
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare {
		t.Fatal("escape", err)
	}
	c.talkTo(secondInnOption{kind: 0, npc: 675, topic: 78})
	if c.bank[770] != 2 || c.shop.records[3][3].Mask != 0x2bc042f0|secondShopClass4 {
		t.Fatalf("inn NPC 675 topic 78: bank 770 %d, record 3 category 3 mask %#x", c.bank[770], c.shop.records[3][3].Mask)
	}
	// Loss control: the stock kept as drawn, which the stale mark replaces,
	// still holds nothing in the fourth category.
	if c.shop.filled == 3 {
		t.Fatal("the conversation left the stock marked as drawn for the current record")
	}
	c.shop.filled = 3
	secondShopEnter(t, f, app, s)
	if len(c.shop.shelves[3]) != 0 {
		t.Fatal("the loss control's stock gained a fourth category")
	}
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare {
		t.Fatal("escape", err)
	}
	c.shop.filled = 0
	secondShopEnter(t, f, app, s)
	secondShopClassFour(t, c)
	open := secondShopPaint(s)
	if !druidProbe(open, elven, elvenAt, p) || !secondShopRows(app, 3) {
		t.Fatal("the fourth category is not drawn or offered with bank slot 770 set")
	}
	shopTap(t, app, image.Pt(305, 224), "press", "release")
	hili := art.Pictures("shelf/3")[0]
	if c.shop.category != 3 || s.shopChosen != 3 || !druidProbe(secondShopPaint(s), hili, elvenAt, druidDiffPoint(t, open, hili, elvenAt)) {
		t.Fatalf("the fourth category: category %d", c.shop.category)
	}
	secondShopWitness(t, app, "town3-shop-elven")
	secondShopActivate(t, f, app, "SHELF 1")
	if err := app.HeadlessKey("escape"); err != nil || c.room != secondTownSquare {
		t.Fatal("escape", err)
	}
}

// secondShopClassFour requires the druid shop's fourth category to hold goods
// and every one of them to admit class 4.
func secondShopClassFour(t *testing.T, c *secondCampaign) {
	t.Helper()
	if c.shop.filled != 3 || len(c.shop.shelves[3]) == 0 {
		t.Fatalf("after the conversation: filled %d, fourth category %d entries", c.shop.filled, len(c.shop.shelves[3]))
	}
	if c.shop.records[3][3].Mask&secondShopClass4 == 0 {
		t.Fatal("record 3 category 3 lacks the class-4 bit")
	}
	for _, it := range c.shop.stock()[3] {
		if it.Kind != secondshop.Book && it.Kind != secondshop.Magic && it.Class != 4 {
			t.Fatalf("fourth category entry %+v is not class 4", it)
		}
	}
}

// secondShopKeeperFrame checks that the page shows frame pic at at, where
// it differs from the resting page base: the loss control.
func secondShopKeeperFrame(t *testing.T, s *secondCampaignScreen, base image.Image, pic image.Image, at image.Point, what string) {
	t.Helper()
	p := druidDiffPoint(t, base, pic, at)
	if !druidProbe(secondShopPaint(s), pic, at, p) {
		t.Fatalf("%s is not drawn at %v", what, at)
	}
}

// TestReleaseSecondTownShopKeepers runs each shop keeper on its page's
// 100 ms step (R2-ENGINE-268, R2-ENGINE-279, R2-ENGINE-259). Town 1: Yes on a
// changed category, No on a buy beyond the gold, Yes on a credited sale and
// the idle Pose2-3 episode after its 5000 ms wait. Towns 2 and 3: a1 or a2
// at idle as R(2) picks, a2 on a buy, nothing on a refusal; the Kaarg fire
// runs up to its cicle loop when weapons are chosen and back (R2-ENGINE-278).
// Each frame is probed where it differs from the resting page.
func TestReleaseSecondTownShopKeepers(t *testing.T) {
	type keeperCase struct {
		name string
		id   int
		at   image.Point
	}
	for _, tc := range []keeperCase{{"Town1", 1, image.Pt(277, 112)}, {"Kaarg", 2, image.Pt(289, 116)}, {"Druid", 3, image.Pt(361, 92)}} {
		t.Run(tc.name, func(t *testing.T) {
			var f *FrontEnd
			var app *ui.App
			var s *secondCampaignScreen
			var now *time.Time
			if tc.id == 1 {
				f, app, s, now = secondShopTownOne(t)
			} else {
				f, app, s, now = secondShopLaterTown(t, tc.id)
			}
			draw := 0
			f.ShopRandom = func(n int) int { return draw % n }
			scene, art := secondShopEnter(t, f, app, s)
			keeper := scene.Actor("keeper").(*town.Priority)
			step := func(n int) {
				for range n {
					*now = now.Add(100 * time.Millisecond)
					s.AdvanceShopInteriorAnimation()
				}
			}
			step(1) // the first paint of the process stamps the episode baseline
			rest := secondShopPaint(s)
			if !druidProbe(rest, art.Pictures("keeper")[0], tc.at, tc.at) || keeper.Running() >= 0 {
				t.Fatal("the keeper does not rest on its first file")
			}
			if tc.id == 1 {
				secondShopActivate(t, f, app, "SHELF 2")
				step(1)
				secondShopKeeperFrame(t, s, rest, art.Pictures("yes")[0], tc.at, "Yes file 2 after a category change")
				step(11)
				if keeper.Running() >= 0 || keeper.Index != 0 {
					t.Fatalf("Yes did not end at counter 12: running %d index %d", keeper.Running(), keeper.Index)
				}
				f.Town.gold = 0
				secondShopActivate(t, f, app, "ITEM 1", "BUY")
				step(1)
				secondShopKeeperFrame(t, s, rest, art.Pictures("no")[0], tc.at, "No file 2 after a buy beyond the gold")
				step(11)
				secondShopActivate(t, f, app, "UNDO")
				f.Town.gold = 100000
				secondShopActivate(t, f, app, "ITEM 1", "BUY", "MINE 1", "SELL")
				if keeper.Running() != 1 {
					t.Fatal("a credited sale raised no Yes", keeper.Running())
				}
				step(12)
				// The idle wait: 5000+1000*(0%5) ms from the last end.
				step(49)
				if keeper.Running() >= 0 {
					t.Fatal("the idle episode started before 5000 ms")
				}
				step(1)
				secondShopKeeperFrame(t, s, rest, art.Pictures("pose")[0], tc.at, "Pose2-3 file 2 at the idle start")
				step(26)
				if keeper.Running() != 0 || keeper.Index != 27 {
					t.Fatalf("idle at file 28: running %d index %d", keeper.Running(), keeper.Index)
				}
				step(1)
				if keeper.Running() >= 0 {
					t.Fatal("the idle episode did not end at counter 28")
				}
				secondShopWitness(t, app, "town1-shop-keeper-rest")
				t.Logf("town 1 keeper: Yes, No, Yes and the 27-file idle episode drawn at %v", tc.at)
				return
			}
			// A refused buy starts nothing; an affordable buy starts a2.
			f.Town.gold = 0
			secondShopActivate(t, f, app, "ITEM 1", "BUY")
			step(1)
			if keeper.Running() >= 0 {
				t.Fatal("a refused buy started an episode")
			}
			secondShopActivate(t, f, app, "UNDO")
			f.Town.gold = 500000
			secondShopActivate(t, f, app, "ITEM 1", "BUY")
			step(1)
			secondShopKeeperFrame(t, s, rest, art.Pictures("a2")[0], tc.at, "a2 file 0002 after a buy")
			steps := len(art.Pictures("a2")) + 1
			step(steps - 1)
			if keeper.Running() >= 0 || keeper.Index != 0 {
				t.Fatalf("a2 did not end at its wrap: running %d index %d", keeper.Running(), keeper.Index)
			}
			// The idle start: R(2) of 30000 picks a2, of 0 a1.
			for _, pick := range []struct {
				raw   int
				state int
				name  string
			}{{30000, 1, "a2"}, {0, 0, "a1"}} {
				draw = pick.raw
				step(49)
				if keeper.Running() >= 0 {
					t.Fatal("the idle episode started before 5000 ms")
				}
				step(1)
				if keeper.Running() != pick.state {
					t.Fatalf("raw %d started state %d, want %s", pick.raw, keeper.Running(), pick.name)
				}
				secondShopKeeperFrame(t, s, rest, art.Pictures(pick.name)[0], tc.at, pick.name+" at the idle start")
				step(steps - 1)
			}
			if tc.id == 2 {
				fire := scene.Actor("fire").(*town.Transit)
				secondShopActivate(t, f, app, "SHELF 4")
				if cell, selected := fire.Shown(); !selected || fire.Dir != 1 {
					t.Fatalf("weapons chosen: cell %d selected %v direction %d", cell, selected, fire.Dir)
				}
				step(18)
				if cell, _ := fire.Shown(); cell < 12 || fire.Dir != 0 {
					t.Fatalf("the fire did not reach its cicle loop: cell %d direction %d", cell, fire.Dir)
				}
				firePic := art.Pictures("fire-cicle-select")[fire.Cursor-12]
				secondShopKeeperFrame(t, s, rest, firePic, image.Pt(357, 204), "the selected cicle cell")
				secondShopWitness(t, app, "town2-shop-fire")
				secondShopActivate(t, f, app, "SHELF 1")
				step(18)
				if cell, selected := fire.Shown(); cell > 5 || selected {
					t.Fatalf("the fire did not return to its lite loop: cell %d selected %v", cell, selected)
				}
			}
			t.Logf("town %d keeper: a2 on a buy, nothing on a refusal, a2 and a1 at idle, %d files each, drawn at %v", tc.id, steps-1, tc.at)
		})
	}
}
