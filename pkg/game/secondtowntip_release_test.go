package game

import (
	"image"
	"testing"

	"againrom/pkg/ui"
)

func secondTipText(t *testing.T, f *FrontEnd, section string) string {
	t.Helper()
	text := readTipText(f.Archives.Containers, ui.TipText{File: "main/text/town.txt", Section: section}, f.textCode())
	if text == "" {
		t.Fatalf("section %s is empty", section)
	}
	return text
}

func secondTipStep(t *testing.T, app *ui.App) {
	t.Helper()
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := app.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame}); err != nil {
		t.Fatal(err)
	}
}

func secondTipShows(t *testing.T, what string, v ui.TipPanelView, text string, rect image.Rectangle) {
	t.Helper()
	if !v.Showing() || v.Text != text || v.Rect != rect {
		t.Fatalf("%s: popup showing %v at %v with %q, want %v with %q", what, v.Showing(), v.Rect, v.Text, rect, text)
	}
}

// R2-ENGINE-1352: the shop popup shows its first text at the entry, takes the
// second once the deal table holds an item, and starts again at the next
// entry.
func TestReleaseSecondTownShopSecondTip(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("second shop second tip")
	app.Layout(640, 480)
	s := f.TownScreen().(*townScreen)
	app.SetTown(s)
	f.SetTipsOff(false)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	first, second := secondTipText(t, f, "tips3"), secondTipText(t, f, "tips4")
	rect := image.Rect(164, 162, 476, 298)
	for visit := 0; visit < 2; visit++ {
		secondShopActivate(t, f, app, "SHOP")
		secondTipStep(t, app)
		secondTipShows(t, "shop entry", s.ShopScreen().TipPanel, first, rect)
		secondTipStep(t, app)
		secondTipShows(t, "empty table", s.ShopScreen().TipPanel, first, rect)
		secondShopActivate(t, f, app, "ITEM 1")
		if len(f.Town.second.shop.Table()) == 0 {
			t.Fatal("the shelf click put nothing on the deal table")
		}
		secondTipStep(t, app)
		secondTipShows(t, "item on the table", s.ShopScreen().TipPanel, second, rect)
		secondShopActivate(t, f, app, "UNDO")
		secondTipStep(t, app)
		secondTipShows(t, "table emptied", s.ShopScreen().TipPanel, second, rect)
		secondShopActivate(t, f, app, "EXIT")
		secondTipStep(t, app)
	}
}

// R2-ENGINE-308, R2-ENGINE-310, R2-ENGINE-312: the town 1 square, inn and
// shop build their popup at each entry while the tips option is set; Close
// holds until the next entry; the option cleared stops the next builds.
func TestReleaseSecondTownRoomTips(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("second town tips")
	app.Layout(640, 480)
	s := f.TownScreen().(*townScreen)
	app.SetTown(s)
	f.SetTipsOff(false)
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("NEW GAME first town", err, app.Screen())
	}
	secondTipStep(t, app)
	square, inn, shop := secondTipText(t, f, "tips1"), secondTipText(t, f, "tips2"), secondTipText(t, f, "tips3")
	squareRect, innRect, shopRect := image.Rect(328, 0, 640, 200), image.Rect(160, 0, 472, 200), image.Rect(164, 162, 476, 298)
	secondTipShows(t, "arrival", s.TownSquareView().Tip, square, squareRect)
	s.CloseTip()
	if s.TownSquareView().Tip.Showing() {
		t.Fatal("Close left the square popup")
	}
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	secondTipStep(t, app)
	secondTipShows(t, "inn entry", s.TownSquareView().Tip, inn, innRect)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	secondTipStep(t, app)
	secondTipShows(t, "return from the inn", s.TownSquareView().Tip, square, squareRect)
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	secondTipStep(t, app)
	secondTipShows(t, "shop entry", s.ShopScreen().TipPanel, shop, shopRect)
	s.ToggleTips()
	if !f.TipsOff() || !s.ShopScreen().TipPanel.Showing() || s.ShopScreen().TipPanel.ToggleOn {
		t.Fatal("the option cleared must keep the open popup with its box clear")
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	secondTipStep(t, app)
	if v := s.TownSquareView().Tip; v.Showing() {
		t.Fatalf("option off: the square popup shows %q", v.Text)
	}
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	secondTipStep(t, app)
	if v := s.TownSquareView().Tip; v.Showing() {
		t.Fatalf("option off: the inn popup shows %q", v.Text)
	}
}
