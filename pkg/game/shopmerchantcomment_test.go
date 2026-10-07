package game

import (
	"fmt"
	"image"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// shopCommentFixture enters a fixture shop through App's load and town
// routes. The merchant's armour shelf and the player's pack each hold one
// voiced item; speech reaches a recorder and no device is opened.
func shopCommentFixture(t *testing.T) (*FrontEnd, *ui.App, *townScreen, *tavernInteriorRecorder) {
	t.Helper()
	f := shellFrontEnd()
	f.SpeechBank = responseBank(t, []synth.File{
		{Path: "shop/s01i02p1.wav", Data: speechWAV(1000)},
		{Path: "shop/s06i06p1.wav", Data: speechWAV(2000)},
	})
	r := &tavernInteriorRecorder{}
	f.SpeechPlayer = r
	f.Shop = NewShop(1000)
	f.Shop.shelves[ShelfArmour] = []ShopItem{{Code: 0x0102, Count: 3, Price: 100}}
	f.Carried[0].CarriedItems = []sim.ItemInstance{sim.PlainItem(0x0606), sim.PlainItem(0x0606)}
	f.Carried[0].Carried = []uint16{0x0606, 0x0606}
	f.shopArtCache = resolved(shopInteriorTestArt(), nil)
	now := time.Unix(200, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.ShopRandom = func(int) int { return 0 }
	a := f.App("shop merchant comments")
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} }, func(string) (ui.MapOpener, bool, error) {
		f.townUI.resetForNewGame()
		return nil, true, nil
	})
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"@first", "SHOP"} {
		if err := a.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	s := f.townUI
	if s.room != roomShop || !s.shopInterior.ready {
		t.Fatalf("shop entry = room %d, interior ready %v", s.room, s.shopInterior.ready)
	}
	closeShopTip(t, a, s)
	return f, a, s, r
}

// closeShopTip clicks a showing tip panel's Close control through App.
func closeShopTip(t *testing.T, app *ui.App, s *townScreen) {
	t.Helper()
	tip := s.ShopScreen().TipPanel
	if !tip.Showing() {
		return
	}
	r := ui.TipPanelCloseRect(tip.Rect)
	p := r.Min.Add(r.Max).Div(2)
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
	}
	if s.ShopScreen().TipPanel.Showing() {
		t.Fatal("the tip's Close control left the panel showing")
	}
}

// shopPointer sends pointer edges through App at shop surfaces; the layout is
// fixed at 640x480, where window and frame pixels coincide. A shelf, pack or
// table cell is pressed at its rect's centre, which must answer that cell.
// Any other surface is located once by the App's locator, which scans the
// whole frame.
func shopPointer(t *testing.T, app *ui.App) func(surface string, index int, edges ...string) {
	type surfaceKey struct {
		name  string
		index int
	}
	grid := map[string]struct {
		rect func(int) image.Rectangle
		kind ui.ShopControlKind
	}{
		"shelf": {ui.ShopShelfCellRect, ui.ShopControlShelfCell},
		"pack":  {ui.ShopPackCellRect, ui.ShopControlPackCell},
		"table": {ui.ShopTableCellRect, ui.ShopControlTableCell},
	}
	points := map[surfaceKey]image.Point{}
	return func(surface string, index int, edges ...string) {
		t.Helper()
		p, ok := points[surfaceKey{surface, index}]
		if cell, isGrid := grid[surface]; !ok && isGrid {
			r := cell.rect(index)
			p = r.Min.Add(r.Size().Div(2))
			if c, hit := ui.ShopControlAt(p); !hit || c.Kind != cell.kind || c.Index != index {
				t.Fatalf("the centre of %s cell %d answers %+v", surface, index, c)
			}
		} else if !ok {
			x, y, err := app.HeadlessShopPoint(surface, index)
			if err != nil {
				t.Fatal(err)
			}
			p = image.Pt(x, y)
			points[surfaceKey{surface, index}] = p
		}
		for _, edge := range edges {
			if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// Hover, clicks and a drag over the player's own goods leave the merchant
// silent and his reaction bits clear. A click on his sale stock describes it.
// A press on the merchant himself starts nothing, stops nothing and leaves the
// message line as it was (TOWN-478). A changed rack still arms Yes
// (SHOP-ANIMATION-082, SHOP-ANIMATION-084).
func TestShopMerchantDescribesOnlyHisSaleStock(t *testing.T) {
	f, app, s, r := shopCommentFixture(t)
	point := shopPointer(t, app)
	idle := func(action string, samples int) {
		t.Helper()
		if len(r.samples) != samples {
			t.Fatalf("%s: merchant started %d recording(s), want %d", action, len(r.samples), samples)
		}
		if s.shopInterior.merchantModes != 0 {
			t.Fatalf("%s armed merchant modes %#x", action, s.shopInterior.merchantModes)
		}
	}
	places := func(action string, want ...bool) {
		t.Helper()
		table := f.Shop.Table()
		if len(table) != len(want) {
			t.Fatalf("%s: table holds %d place(s), want %d", action, len(table), len(want))
		}
		for i, mine := range want {
			if table[i].Mine != mine {
				t.Fatalf("%s: table place %d Mine=%v, want %v", action, i, table[i].Mine, mine)
			}
		}
	}

	point("shelf", 0, "hover")
	point("pack", 1, "hover")
	idle("hover", 0)

	point("pack", 1, "press", "release")
	places("pack click", true)
	idle("pack click", 0)

	point("table", 0, "press", "release")
	places("table click")
	idle("table click", 0)

	point("pack", 1, "press")
	point("table", 0, "move", "release")
	places("pack drag", true)
	idle("pack drag", 0)

	line := app.HeadlessMessage()
	point("merchant", 0, "press", "release")
	idle("merchant press over player goods", 0)
	if got := app.HeadlessMessage(); got != line {
		t.Fatalf("merchant press over player goods changed the line from %q to %q", line, got)
	}

	point("table", 0, "press", "release")
	point("shelf", 0, "press", "release")
	places("shelf click", false)
	idle("shelf click", 1)
	if r.samples[0].PCM[0] != 1000 || !r.voices[0].Playing() {
		t.Fatal("shelf click did not describe the merchant's item")
	}

	point("pack", 1, "press", "release")
	places("pack click after sale stock", false, true)
	idle("pack click after sale stock", 1)
	if !r.voices[0].Playing() {
		t.Fatal("pack click stopped the merchant's description")
	}

	point("merchant", 0, "press", "release")
	idle("merchant press", 1)
	if !r.voices[0].Playing() {
		t.Fatal("merchant press stopped the description that was playing")
	}

	point("shelf_pick", roomWeapons, "press", "release")
	if len(r.samples) != 1 || s.shopInterior.merchantModes != shopMerchantYes {
		t.Fatalf("changed rack: recordings %d, modes %#x; want 1 and Yes", len(r.samples), s.shopInterior.merchantModes)
	}
}

// A town whose goods are city objects stages the player's pack through its
// own mutation path. A pack click and a pack drag to the table start no
// description; a click on the merchant's shelf stack starts his.
func TestCityShopMerchantDescribesOnlyHisSaleStock(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Town.open = true
	f.Carried[0].ID = "hero"
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	f.Carried[0].Name = "Hero"
	f.Carried[0].Profile.Fighter = true
	f.Carried[0].Hero = data.NewHero(data.Spread{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}, 1)
	his := f.Shop.Shelf(ShelfArmour)[0]
	mine := f.Shop.Shelf(ShelfWeapons)[0].Instance()
	cityShopSetTestPack(t, f, 0, sim.StackItem(mine, 2))
	cityShopGraph(t, f)
	name := func(code data.ItemCode) string { return fmt.Sprintf("shop/s%02di%02dp1.wav", code.B(), code.D()) }
	if name(his.Code) == name(data.ItemCode(mine.Code)) {
		t.Fatal("the fixture's two stacks share one recording")
	}
	f.SpeechBank = responseBank(t, []synth.File{
		{Path: name(data.ItemCode(mine.Code)), Data: speechWAV(1000)},
		{Path: name(his.Code), Data: speechWAV(2000)},
	})
	r := &tavernInteriorRecorder{}
	f.SpeechPlayer = r

	click(s, ui.ShopControlPackCell, 1)
	s.ShopDrag(ui.ShopControl{Kind: ui.ShopControlPackCell, Index: 1}, ui.ShopControl{Kind: ui.ShopControlTableCell})
	var staged int32
	for _, place := range f.Shop.Table() {
		if !place.Mine {
			t.Fatalf("city pack staging put a merchant place on the table: %+v", place)
		}
		staged += place.Count
	}
	if staged != 2 || len(r.samples) != 0 {
		t.Fatalf("city pack click and drag staged %d unit(s) and started %d recording(s); want 2 and 0", staged, len(r.samples))
	}
	click(s, ui.ShopControlShelfPick, roomArmour)
	click(s, ui.ShopControlShelfCell, 0)
	if len(r.samples) != 1 || r.samples[0].PCM[0] != 2000 {
		t.Fatalf("city shelf click started %d recording(s); want the merchant's stack", len(r.samples))
	}
}
