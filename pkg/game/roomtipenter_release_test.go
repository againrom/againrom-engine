package game

import (
	"image"
	"testing"

	"againrom/pkg/ui"
)

// roomTipTownApp opens the town square of a fresh campaign through the
// App's LOAD route.
func roomTipTownApp(t *testing.T, f *FrontEnd) *ui.App {
	t.Helper()
	f.Carried = MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	f.Town = NewTown(f.Campaign.Value())
	f.Shop = NewShop(5000)
	f.Shop.Generate(f.Table, 1182)
	a := f.App("room tips")
	t.Cleanup(a.StopAudio)
	a.Layout(640, 480)
	a.SetTown(f.TownScreen())
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town", Label: "town"}} }, func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("town"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown {
		t.Fatalf("LOAD opened screen %v, want the town", a.Screen())
	}
	return a
}

// roomTipEnterShop walks into the shop through App and leaves its popup open.
func roomTipEnterShop(t *testing.T, f *FrontEnd, app *ui.App) *townScreen {
	t.Helper()
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	for n := 0; s.room == roomTalk && n < 32; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if s.room != roomShop {
		t.Fatalf("App entered room %d, want the shop", s.room)
	}
	return s
}

func roomTipLeave(t *testing.T, app *ui.App, s *townScreen) {
	t.Helper()
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	roomTipSteps(t, app, 1)
	if s.room != roomSquare {
		t.Fatalf("Escape left the shop for room %d, want the square", s.room)
	}
}

func roomTipPutOnTable(t *testing.T, s *townScreen) {
	t.Helper()
	for shelf := ShopShelf(0); shelf < numShopShelves; shelf++ {
		if len(s.sess.Shop.Shelf(shelf)) > 0 && s.sess.Shop.TakeFromShelf(shelf, 0, 1) {
			return
		}
	}
	t.Fatal("no shelf element moved to the table")
}

// roomTipSteps runs n steps, each followed by a paint: the shop checks its
// second text on its idle pass, once per paint.
func roomTipSteps(t *testing.T, app *ui.App, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		if _, _, err := app.HeadlessFrame(); err != nil {
			t.Fatal(err)
		}
	}
}

func roomTipRender(t *testing.T, app *ui.App, dir, name string) {
	t.Helper()
	if dir == "" {
		return
	}
	pic, _, err := app.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	writeTipRender(t, dir, name, pic)
}

// TOWN-516: each room builds its popup at its rectangle on every enter while
// TipsMode is set. TOWN-517: shop2 replaces shop1 once per shop activation
// when the table holds an item, whatever TipsMode says after the activation.
func TestReleaseRoomTipsAtEveryEnterAndShopSecondText(t *testing.T) {
	f := missionTipFront(t)
	if f.Archives == nil {
		t.Fatal("install archives did not load")
	}
	shop1, ok1 := ReadShopTip(f.Archives.Containers, "main/text/tips/shop1.txt")
	shop2, ok2 := ReadShopTip(f.Archives.Containers, "main/text/tips/shop2.txt")
	if !ok1 || !ok2 || shop1 == shop2 {
		t.Fatalf("shop1 read %v, shop2 read %v; want two different shipped texts", ok1, ok2)
	}

	rooms := []struct {
		name   string
		choose int
		claim  image.Rectangle
		view   func(*townScreen) ui.TipPanelView
	}{
		{"tavern", 0, image.Rect(160, 0, 472, 200), func(s *townScreen) ui.TipPanelView { return s.TownSurface().Tip }},
		{"shop", 1, image.Rect(164, 162, 476, 298), func(s *townScreen) ui.TipPanelView { return s.ShopScreen().TipPanel }},
		{"school", 2, image.Rect(0, 0, 456, 200), func(s *townScreen) ui.TipPanelView { return s.TownSurface().Tip }},
	}
	for _, room := range rooms {
		g := missionTipFront(t)
		s := g.TownScreen().(*townScreen)
		for enter := 0; enter < 2; enter++ {
			s.Choose(room.choose)
			v := room.view(s)
			if !v.Showing() || v.Rect != room.claim {
				t.Fatalf("%s enter %d: popup showing %v at %v, want %v", room.name, enter, v.Showing(), v.Rect, room.claim)
			}
			s.CloseTip()
			if room.view(s).Showing() {
				t.Fatalf("%s: Close left the popup showing", room.name)
			}
			s.Back()
		}
	}
	square := f.TownScreen().(*townScreen).TownSquareView().Tip
	claim := image.Rect(328, 0, 640, 200)
	fits := square
	fits.Rect = claim
	if !square.Showing() || square.Rect.Min != claim.Min || square.Rect.Max.X != claim.Max.X ||
		(ui.TipPanelFits(fits) && square.Rect != claim) || (!ui.TipPanelFits(fits) && square.Rect.Dy() <= claim.Dy()) {
		t.Fatalf("town popup at %v, want %v, taller only when the text overflows it", square.Rect, claim)
	}

	app := roomTipTownApp(t, f)
	renders := tipRenderDir(t, f)
	s := roomTipEnterShop(t, f, app)
	roomTipSteps(t, app, 3)
	if got := s.ShopScreen().TipPanel.Text; got != shop1 {
		t.Fatalf("empty table: popup text %q, want shop1", got)
	}
	roomTipRender(t, app, renders, "shop-first-tip")
	roomTipPutOnTable(t, s)
	roomTipSteps(t, app, 1)
	if got := s.ShopScreen().TipPanel.Text; got != shop2 {
		t.Fatalf("item on the table: popup text %q, want shop2", got)
	}
	roomTipRender(t, app, renders, "shop-second-tip-after-table")
	roomTipLeave(t, app, s)

	// A new activation clears the latch; TipsMode cleared after it does not
	// stop the second text.
	s = roomTipEnterShop(t, f, app)
	if got := s.ShopScreen().TipPanel.Text; got != shop1 {
		t.Fatalf("second activation: popup text %q, want shop1", got)
	}
	f.SetTipsOff(true)
	roomTipPutOnTable(t, s)
	roomTipSteps(t, app, 1)
	if got := s.ShopScreen().TipPanel.Text; got != shop2 {
		t.Fatalf("TipsMode cleared after activation: popup text %q, want shop2", got)
	}
	roomTipLeave(t, app, s)

	// TipsMode clear at the enter: no popup, and nothing for shop2 to retext.
	s = roomTipEnterShop(t, f, app)
	roomTipPutOnTable(t, s)
	roomTipSteps(t, app, 2)
	if v := s.ShopScreen().TipPanel; v.Showing() {
		t.Fatalf("TipsMode clear at the enter: popup showing %q", v.Text)
	}
}
