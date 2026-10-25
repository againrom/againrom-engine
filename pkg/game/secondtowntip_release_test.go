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
	if _, _, err := app.HeadlessFrame(); err != nil {
		t.Fatal(err)
	}
}

func secondTipShows(t *testing.T, what string, v ui.TipPanelView, text string, rect image.Rectangle) {
	t.Helper()
	if !v.Showing() || v.Text != text || v.Rect != rect {
		t.Fatalf("%s: popup showing %v at %v with %q, want %v with %q", what, v.Showing(), v.Rect, v.Text, rect, text)
	}
}

// R2-ENGINE-308, R2-ENGINE-310, R2-ENGINE-312: the town 1 square, inn and
// shop build their popup at each entry while the tips option is set; Close
// holds until the next entry; the option cleared stops the next builds.
func TestReleaseSecondTownRoomTips(t *testing.T) {
	f := secondGameFront(t)
	app := f.App("second town tips")
	app.Layout(640, 480)
	s := f.TownScreen().(*secondCampaignScreen)
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
