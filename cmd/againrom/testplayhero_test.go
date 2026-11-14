package main

import (
	"io"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/game"
	"againrom/pkg/herokind"
	"againrom/pkg/ui"
)

// heroStart is the -mission start of the arguments given over the install
// at root, through the wiring run installs.
func heroStart(t *testing.T, root string, args ...string) (*game.FrontEnd, *ui.App) {
	t.Helper()
	o, err := parse(append([]string{"-seed", "7"}, args...))
	if err != nil {
		t.Fatal(err)
	}
	front, err := frontEnd(root, o, game.OptionsStore{})
	if err != nil {
		t.Fatal(err)
	}
	front.SetDeterministicFrames(true)
	front.SetTipsOff(true)
	app := front.App("test")
	app.Layout(640, 480)
	if err := armNewGameDoor(app, front, o); err != nil {
		t.Fatal(err)
	}
	return front, app
}

// -hero, which the map editor's party window passes, starts the mission
// with no generation screen, and the hero is the one the screen makes for
// that kind when the player presses its picture and accepts it unchanged:
// the started party and its snapshot are equal field by field, and so is
// the difficulty. Without -hero the same start opens the generation screen.
func TestHeroFlagSkipsGenerationWithTheScreensHero(t *testing.T) {
	for _, lang := range []string{"english", "russian"} {
		t.Run(lang, func(t *testing.T) {
			root := missionInstall(t)
			registerProfile(t, root, base.Profile{ID: "rom1-hero-test", Title: "Constructed base", Language: lang})
			for k := herokind.MaleFighter; k <= herokind.FemaleMage; k++ {
				skip, app := heroStart(t, root, "-mission", "10", "-hero", k.Word())
				if app.Screen() != ui.ScreenMap {
					t.Fatalf("%s: -hero left screen %v", k, app.Screen())
				}
				screen, shown := heroStart(t, root, "-mission", "10")
				if shown.Screen() != ui.ScreenChargen {
					t.Fatalf("%s: without -hero the screen is %v", k, shown.Screen())
				}
				sex, class := herokind.Male, herokind.Fighter
				if k.Female() {
					sex = herokind.Female
				}
				if k.Mage() {
					class = herokind.Mage
				}
				scenario := game.HeadlessScenario{Version: game.HeadlessScenarioVersion, Steps: []game.HeadlessStep{
					{Command: "create_character", Character: &game.HeadlessCharacter{Sex: sex, Class: class}}}}
				if err := game.RunHeadlessScenario(screen, shown, scenario, io.Discard, io.Discard); err != nil {
					t.Fatalf("%s: %v", k, err)
				}
				got, want := skip.LiveParty(), screen.LiveParty()
				if len(got) == 0 || got[0].Mage != k.Mage() || !reflect.DeepEqual(got, want) {
					t.Fatalf("%s: -hero starts %+v\nthe screen starts %+v", k, got, want)
				}
				a, b := skip.HeadlessSnapshot(app.Screen()), screen.HeadlessSnapshot(shown.Screen())
				if a.Mission != 10 || a.Difficulty != b.Difficulty || !reflect.DeepEqual(a.Members, b.Members) {
					t.Fatalf("%s: -hero snapshot %+v, the screen's %+v", k, a, b)
				}
			}
		})
	}
}

// -hero names one of the four kinds and needs -mission; a start by hand
// with -mission and no -hero keeps the generation screen, and so does a
// new game from the menu.
func TestHeroFlagIsRefusedAlone(t *testing.T) {
	if _, err := parse([]string{"-hero", "female-mage"}); err == nil || !strings.Contains(err.Error(), "-mission") {
		t.Fatalf("-hero without -mission: %v", err)
	}
	if _, err := parse([]string{"-mission", "10", "-hero", "dragon"}); err == nil || !strings.Contains(err.Error(), "female-mage") {
		t.Fatalf("an unknown kind: %v", err)
	}
	root := missionInstall(t)
	if _, app := heroStart(t, root, "-mission", "10", "-mods", "testplay"); app.Screen() != ui.ScreenChargen {
		t.Fatalf("-mission without -hero: screen %v", app.Screen())
	}
	_, app := heroStart(t, root)
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenChargen {
		t.Fatalf("NEW GAME: screen %v, %v", app.Screen(), err)
	}
}
