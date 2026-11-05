package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

func TestWorldMapEscapeUsesTheMapKeyPolicy(t *testing.T) {
	for _, phase := range []string{"idle", "select travel", "route travel"} {
		t.Run(phase, func(t *testing.T) {
			open := func() (*ui.App, *townScreen, *worldMapState) {
				_, app, screen := secondCampaignFixture(t, true)
				if err := app.HeadlessActivate("new game"); err != nil {
					t.Fatal(err)
				}
				mode := secondMapSelect
				if phase == "route travel" {
					mode = secondMapRoute
				}
				state := secondMapState(t, mode, image.Pt(100, 400), secondRecord(placeTown, 2, image.Pt(100, 50)))
				c := screen.state()
				c.current, c.available = partyPlace{}, []partyPlace{{placeTown, 2}}
				c.openMap(partyPlace{placeTown, 3})
				screen.mapArt = func() *worldMapAssets { return state.assets }
				screen.worldMap, screen.worldMapOf, screen.worldMapAt = state, c, c.mapOpen
				if phase == "select travel" && !state.startTravel(0) {
					t.Fatal("fixture could not start travel")
				}
				if app.Screen() != ui.ScreenTown || !screen.AtWorldMap() || state.travelling() != (phase != "idle") {
					t.Fatal("fixture did not open the requested map phase")
				}
				return app, screen, state
			}
			app, screen, state := open()
			other, otherScreen, otherState := open()
			before := screen.WorldMapView()
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenTown || !screen.AtWorldMap() {
				t.Fatalf("Escape opened %s, map open=%v; want town with the map open", app.Screen(), screen.AtWorldMap())
			}
			if phase == "idle" {
				if !reflect.DeepEqual(before, screen.WorldMapView()) {
					t.Fatal("idle Escape changed the map")
				}
				return
			}
			if err := other.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if state.shown != state.routeEnd() || state.cross != otherState.cross || state.shown != otherState.shown {
				t.Fatalf("Escape progress %d/%d cross %d; Enter progress %d cross %d", state.shown, state.routeEnd(), state.cross, otherState.shown, otherState.cross)
			}
			for _, s := range []*townScreen{screen, otherScreen} {
				if act := s.WorldMapTick(); act.Open != nil || act.Msg != "" || s.AtWorldMap() || s.state().current != (partyPlace{placeTown, 2}) {
					t.Fatalf("next map tick did not enter the destination town: action %+v, current %v, map %v", act, s.state().current, s.AtWorldMap())
				}
			}
		})
	}
}

func TestTownEscapeStillOpensTheGameMenu(t *testing.T) {
	_, app, screen := secondCampaignFixture(t, true)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if screen.AtWorldMap() || screen.state().current != (partyPlace{placeTown, 1}) {
		t.Fatal("fixture is not in a town room")
	}
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenGameMenu {
		t.Fatalf("town Escape: error %v, screen %s", err, app.Screen())
	}
}

func TestWorldMapEscapeKeepsTheChapterTownBackRoute(t *testing.T) {
	_, screen := travelFront(t, travelCatalog)
	screen.Choose(3)
	if !screen.AtWorldMap() {
		t.Fatal("fixture did not open the chapter map")
	}
	if !screen.Back() || screen.AtWorldMap() || !screen.AtTownSquare() {
		t.Fatal("chapter map Back did not return to the square")
	}
	if screen.Back() {
		t.Fatal("chapter square Back no longer yields to the game menu")
	}
}
