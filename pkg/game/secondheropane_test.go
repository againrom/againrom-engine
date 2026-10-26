package game

import (
	"image"
	"testing"

	"againrom/pkg/town"
	"againrom/pkg/ui"
)

// Each second-game town's inn and shop carry the hero pane at the shots'
// rectangle; only the shop has the book corner, its book two rows lower.
func TestSecondTownRoomsCarryTheHeroPane(t *testing.T) {
	want := town.Rect{480, 238, 640, 480}
	for id := 1; id <= 3; id++ {
		d := secondTownDescription(id)
		if d == nil {
			t.Fatalf("town %d: no description", id)
		}
		inn, shop := d.Room("inn"), d.Room("shop")
		if inn == nil || inn.Pane == nil || inn.Pane.Rect != want || inn.Pane.Book {
			t.Errorf("town %d inn pane = %+v", id, inn)
		}
		if shop == nil || shop.Pane == nil || shop.Pane.Rect != want || !shop.Pane.Book || shop.Pane.BookOffset != (town.Point{0, 2}) {
			t.Errorf("town %d shop pane = %+v", id, shop)
		}
	}
}

// With one hero the pane shows that hero; the eye swaps the figure and the
// card, and PREVIOUS and NEXT keep him.
func TestSecondHeroPaneWithOneHero(t *testing.T) {
	_, _, screen := secondCampaignFixtureFiles(t, false)
	screen.composed = &secondTownState{}
	spec := &town.PaneSpec{Rect: town.Rect{480, 238, 640, 480}}
	v, _, _ := screen.heroPane(spec)
	if !v.HasSubject || v.Statistics || !v.NoBook || v.MemberCount != 1 || v.Member != 0 || v.PaneRect != image.Rect(480, 238, 640, 480) {
		t.Fatalf("inn pane = has %v stats %v nobook %v member %d/%d rect %v", v.HasSubject, v.Statistics, v.NoBook, v.Member, v.MemberCount, v.PaneRect)
	}
	screen.TownSquarePane(ui.CharacterPaneMode)
	if v, _, _ = screen.heroPane(spec); !v.Statistics {
		t.Fatal("the eye kept the figure")
	}
	screen.TownSquarePane(ui.CharacterPaneMode)
	if v, _, _ = screen.heroPane(spec); v.Statistics {
		t.Fatal("the second eye click kept the card")
	}
	screen.TownSquarePane(ui.CharacterPanePrev)
	screen.TownSquarePane(ui.CharacterPaneNext)
	if v, _, _ = screen.heroPane(spec); v.Member != 0 || screen.shopMember != 0 {
		t.Fatalf("the picker moved one hero to %d", v.Member)
	}
	spec.Book = true
	screen.shopBook = true
	if v, _, _ = screen.heroPane(spec); v.NoBook || !v.BookOpen {
		t.Fatal("the shop pane has no open book corner")
	}
}
