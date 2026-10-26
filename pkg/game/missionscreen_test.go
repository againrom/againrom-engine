package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

// R2-ENGINE-588, SESS-VIEW-028: the right column's width and the notice
// placement are each game's mission screen data.
func TestMissionScreenIsEachGamesData(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files *gameFiles
		want  ui.MissionScreen
	}{
		{"first", &firstGameFiles, ui.MissionScreen{ColumnW: ui.MissionPanelW}},
		{"second", &secondGameFiles, ui.MissionScreen{ColumnW: 176, NoticeOnScreen: true}},
	} {
		if tc.files.missionScreen != tc.want {
			t.Errorf("%s game: mission screen %+v, want %+v", tc.name, tc.files.missionScreen, tc.want)
		}
	}
}

// MISSION-ROUTE-093, R2-ENGINE-588, DIV-4020: both games open a new mission
// with the pack row closed; the second game opens it with the hero selected
// (R2-ASSET-148's sheet and portrait from the first capture).
func TestNewMissionPanelsAreEachGamesData(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		files                    *gameFiles
		packClosed, heroSelected bool
	}{
		{"first", &firstGameFiles, true, false},
		{"second", &secondGameFiles, true, true},
	} {
		if tc.files.missionPackClosed != tc.packClosed || tc.files.missionHeroSelected != tc.heroSelected {
			t.Errorf("%s game: pack closed %v, hero selected %v; want %v, %v", tc.name,
				tc.files.missionPackClosed, tc.files.missionHeroSelected, tc.packClosed, tc.heroSelected)
		}
	}
}

// MISSION-ROUTE-093: the first game's purse hovers with the install's money
// word on one line; the second game keeps the name and quantity.
func TestPurseHoverIsEachGamesData(t *testing.T) {
	var w ui.Words
	w.Hover[purseHoverSlot] = "Money"
	for _, tc := range []struct {
		name  string
		files *gameFiles
		want  []string
	}{
		{"first", &firstGameFiles, []string{"Money"}},
		{"second", &secondGameFiles, []string{"Gold", "Quantity: 100"}},
	} {
		if got := purseInfo(tc.files, w, 100); !slices.Equal(got, tc.want) {
			t.Errorf("%s game: purse hover %q, want %q", tc.name, got, tc.want)
		}
	}
	var sub ui.InventorySubject
	appendInventoryGold(&sub, 100, nil, purseInfo(&firstGameFiles, w, 100))
	if got := inventoryGold(sub); got != 100 {
		t.Errorf("a purse hovering with the money word reads %d gold, want 100", got)
	}
}

// R2-ASSET-148: the mission's statistics sheet draws in the generator's text
// font, not in the tip font the second game's description names.
func TestMissionCardDrawsInTheGeneratorTextFont(t *testing.T) {
	textFont, tipFont := &text.Font{}, &text.Font{}
	in := &InstallResources{ChargenAssets: &ChargenAssets{Presentation: &ui.ChargenPresentation{Font: textFont, TipFont: tipFont}}}
	if in.missionCardFont() != textFont {
		t.Fatal("the mission card does not draw in the generator's text font")
	}
}

// openMissionPack presses the inventory key when the pack row is closed, as a
// player clicks the bag: a new mission opens with the row closed (DIV-4020),
// and a test that works the pack opens it first. It reads the row's place, not
// its picture, so it paints nothing.
func openMissionPack(t testing.TB, app *ui.App) {
	t.Helper()
	if missionPackShown(app) {
		return
	}
	if err := app.HeadlessKey("inventory"); err != nil {
		t.Fatal(err)
	}
	if !missionPackShown(app) {
		t.Fatal("the inventory key did not open the pack row")
	}
}

// missionPackShown reports whether the frame has a pack row; only a closed
// row answers a pack cell with "draws no pack bar".
func missionPackShown(app *ui.App) bool {
	_, _, err := app.HeadlessPackCellPoint(0)
	return err == nil || !strings.Contains(err.Error(), "draws no pack bar")
}
