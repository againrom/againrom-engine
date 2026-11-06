package game

import (
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// A second-game town screen is bound before NEW GAME makes the campaign: the
// application holds the screen it was given when it was made, and NEW GAME
// starts the campaign on that screen. Its shop page still trades with the
// campaign's one shop.
func TestSecondTownScreenBoundBeforeNewGameTradesWithTheCampaignShop(t *testing.T) {
	f, _, screen := secondNewGameOnBoundScreen(t)
	if screen.deal() != &f.Town.second.shop {
		t.Fatal("the shop page trades with another deal than the campaign's shop")
	}
}

// secondNewGameOnBoundScreen runs NEW GAME on the town screen the
// application was made with.
func secondNewGameOnBoundScreen(t *testing.T) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f, app, screen := secondCampaignFixture(t, true)
	if screen.deal() != nil || screen.AtTownShop() {
		t.Fatal("a town screen with no campaign shows or trades with a shop")
	}
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenTown || f.Town.second == nil {
		t.Fatal("new game", err, app.Screen())
	}
	return f, app, screen
}

// Asking whether the shop room shows leaves the new campaign as NEW GAME
// made it.
func TestSecondTownShopQuestionLeavesTheNewCampaign(t *testing.T) {
	f, _, screen := secondNewGameOnBoundScreen(t)
	c := f.Town.second
	for i := 0; i < 3; i++ {
		if screen.AtTownShop() {
			t.Fatal("the square shows the shop room")
		}
	}
	if !reflect.DeepEqual(c, newSecondCampaign()) {
		t.Fatalf("asking for the shop room changed the new campaign: shop location %d", c.shop.location)
	}
}

// A town SAV loaded from the main menu, with no New Game before it, runs
// on the town screen the application was made with: its shop page trades
// with the loaded campaign's one shop.
func TestSecondTownScreenBoundBeforeLoadTradesWithTheCampaignShop(t *testing.T) {
	raw := identityTownSave(t)
	cold, _, screen := secondCampaignFixtureFiles(t, false, identityNames())
	cold.Table = identityDollTable()
	if screen.deal() != nil {
		t.Fatal("a town screen with no campaign trades with a shop")
	}
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town || cold.Town.second == nil {
		t.Fatal("town LOAD", town, err)
	}
	if screen.deal() != &cold.Town.second.shop {
		t.Fatal("after LOAD the shop page trades with another deal than the campaign's shop")
	}
}

// Sleep draws the stock for the party's town even when the shop was last
// drawn for no town, as in a new campaign whose shop was never entered.
func TestSecondInnSleepDrawsForThePartysTown(t *testing.T) {
	for _, id := range []int{1, 2, 3} {
		f, s := sleepTown(id)
		c := f.Town.second
		c.shop.location, c.shop.filled, c.shop.shelves = 0, 0, [shopShelfCount][]ShopItem{}
		chooseSleep(t, s)
		if c.shop.location != id || c.shop.filled != id || len(c.shop.shelves[3]) == 0 {
			t.Fatalf("town %d Sleep: location %d, filled %d, %d entries", id, c.shop.location, c.shop.filled, len(c.shop.shelves[3]))
		}
	}
}
