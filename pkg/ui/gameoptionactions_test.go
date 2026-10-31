package ui

import "testing"

// The tests name each Game Options row's page action; production reaches
// them only through optionAction and actionOption.
const (
	gameMenuHealth      = gameMenuDayNight + gameMenuAction(GameOptionHealth)
	gameMenuDamage      = gameMenuDayNight + gameMenuAction(GameOptionDamage)
	gameMenuFormation   = gameMenuDayNight + gameMenuAction(GameOptionFormation)
	gameMenuRetreat     = gameMenuDayNight + gameMenuAction(GameOptionRetreat)
	gameMenuPathfinding = gameMenuDayNight + gameMenuAction(GameOptionPathfinding)
	gameMenuSmoothing   = gameMenuDayNight + gameMenuAction(GameOptionSmoothing)
	gameMenuShadows     = gameMenuDayNight + gameMenuAction(GameOptionShadows)
	gameMenuLighting    = gameMenuDayNight + gameMenuAction(GameOptionLighting)
	gameMenuAnimation   = gameMenuDayNight + gameMenuAction(GameOptionAnimation)
	gameMenuAutoHealing = gameMenuDayNight + gameMenuAction(GameOptionAutoHealing)
	gameMenuLootDots    = gameMenuDayNight + gameMenuAction(GameOptionLootDots)
	gameMenuAutoCast    = gameMenuDayNight + gameMenuAction(GameOptionAutoCast)
)

// TestGameOptionActionsKeepTheirValues pins the menu actions around the
// option block: each option's action is its row's, and the actions after the
// block keep the values they had when the block was spelled out.
func TestGameOptionActionsKeepTheirValues(t *testing.T) {
	if gameMenuDayNight != 22 || gameMenuAutoCast != 34 || gameMenuMusicVolume != 35 || gameMenuAutosaveMinutes != 51 {
		t.Fatalf("actions moved: day/night %d, autocast %d, music %d, autosave minutes %d",
			gameMenuDayNight, gameMenuAutoCast, gameMenuMusicVolume, gameMenuAutosaveMinutes)
	}
	for _, o := range GameOptions() {
		if got, ok := actionOption(optionAction(o)); !ok || got != o {
			t.Errorf("option %d action round trip %d %v", o, got, ok)
		}
	}
	if _, ok := actionOption(gameMenuMusicVolume); ok {
		t.Fatal("the action after the block names an option")
	}
}
