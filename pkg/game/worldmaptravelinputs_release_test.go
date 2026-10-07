package game

import (
	"image"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

const travelWitnessMission = 30

// travelWitness is the owner's city SAV loaded through the Load window with the
// gates open and mission 30's scroll, the only scroll offered, clicked through
// App's pointer input. Nothing is revealed yet and every later input goes
// through App as well.
type travelWitness struct {
	t      *testing.T
	app    *ui.App
	front  *FrontEnd
	screen *townScreen
	route  []image.Point
	anchor image.Point
	// missX and missY are a window pixel that misses every scroll and region.
	missX, missY int
}

func startTravelWitness(t *testing.T) *travelWitness {
	t.Helper()
	_, raw := groundCorpusFile(t, "2026-08-15/game0010.sav", "89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "source.sav"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	f := releaseFront(t)
	app := openLocalTownSAV(t, f, dir, "source.sav")
	t.Cleanup(app.StopAudio)
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	s := f.TownScreen().(*townScreen)
	gates := s.WorldMapView()
	if len(gates.Missions) != 1 || gates.Missions[0].Number != travelWitnessMission || !gates.AtHome || gates.Selected != -1 {
		t.Fatalf("gates before the first click: %d scrolls, at home %v, selected %d; want mission %d's scroll alone with nothing selected",
			len(gates.Missions), gates.AtHome, gates.Selected, travelWitnessMission)
	}
	w := &travelWitness{t: t, app: app, front: f, screen: s, anchor: gates.Missions[0].Anchor}
	x, y, err := app.HeadlessWorldMapMissionPoint(travelWitnessMission)
	if err != nil {
		t.Fatal(err)
	}
	w.point("press", x, y)
	w.point("release", x, y)
	started := s.WorldMapView()
	w.route = started.Route
	if started.Selected != 0 || started.RouteShown != 0 || len(w.route) < 6*worldMapRouteReveal || w.route[0] != gates.Position || w.route[len(w.route)-1] != w.anchor {
		t.Fatalf("scroll click: selected %d, shown %d, route %d points; want mission %d's route from %v to %v with nothing shown",
			started.Selected, started.RouteShown, len(w.route), travelWitnessMission, gates.Position, w.anchor)
	}
	if w.missX, w.missY, err = app.HeadlessWorldMapMissPoint(); err != nil {
		t.Fatal(err)
	}
	t.Logf("scroll click on mission %d: route of %d coordinates from %v to %v, %d reveal ticks",
		travelWitnessMission, len(w.route), gates.Position, w.anchor, (len(w.route)+worldMapRouteReveal-1)/worldMapRouteReveal)
	return w
}

func (w *travelWitness) step() {
	w.t.Helper()
	if err := w.app.HeadlessStep(); err != nil {
		w.t.Fatal(err)
	}
}

func (w *travelWitness) key(name string) {
	w.t.Helper()
	if err := w.app.HeadlessKey(name); err != nil {
		w.t.Fatal(err)
	}
}

func (w *travelWitness) point(edge string, x, y int) {
	w.t.Helper()
	if err := w.app.HeadlessPointer(edge, x, y); err != nil {
		w.t.Fatal(err)
	}
}

// input sends Up or Down as a key and the wheel up or down at the miss pixel.
func (w *travelWitness) input(name string) {
	w.t.Helper()
	if name == "up" || name == "down" {
		w.key(name)
		return
	}
	w.point(name, w.missX, w.missY)
}

// steady fails unless the travel is exactly as the scroll click left it, with
// shown coordinates of its route revealed and the gates still open.
func (w *travelWitness) steady(what string, shown int) {
	w.t.Helper()
	v := w.screen.WorldMapView()
	if w.app.Screen() != ui.ScreenTown || v.Selected != 0 || !reflect.DeepEqual(v.Route, w.route) || v.RouteShown != shown {
		w.t.Fatalf("%s: screen %s, selected %d, revealed %d of %d, route %d points; want the travel as it was with %d revealed",
			what, w.app.Screen(), v.Selected, v.RouteShown, len(w.route), len(v.Route), shown)
	}
}

// reveal idles until the next reveal tick and fails unless that tick added one
// step of eight coordinates to shown and changed nothing else.
func (w *travelWitness) reveal(shown int) {
	w.t.Helper()
	for n := 0; w.screen.WorldMapView().RouteShown == shown; n++ {
		if n > 10 {
			w.t.Fatalf("no reveal tick within %d idle frames with %d revealed", n, shown)
		}
		w.step()
	}
	w.steady("a reveal tick", shown+worldMapRouteReveal)
}

// finish idles until the travel opens mission 30 and returns the number of
// reveal ticks it took after shown coordinates were revealed, the arrival
// tick included. Each tick before the arrival must add eight coordinates.
func (w *travelWitness) finish(shown int) int {
	w.t.Helper()
	ticks := 0
	for n := 0; w.app.Screen() == ui.ScreenTown; n++ {
		if n > 5*(len(w.route)/worldMapRouteReveal+2) {
			w.t.Fatalf("mission %d did not open within %d idle frames with %d of %d revealed", travelWitnessMission, n, shown, len(w.route))
		}
		w.step()
		if w.app.Screen() != ui.ScreenTown {
			ticks++
			break
		}
		if got := w.screen.WorldMapView().RouteShown; got != shown {
			if got != shown+worldMapRouteReveal {
				w.t.Fatalf("a reveal tick took the reveal from %d to %d of %d, want a step of %d", shown, got, len(w.route), worldMapRouteReveal)
			}
			shown = got
			ticks++
			w.steady("a reveal tick", shown)
		}
	}
	if w.app.Screen() != ui.ScreenMap || w.front.live == nil || w.front.live.mission.number != travelWitnessMission {
		w.t.Fatalf("the travel did not open mission %d: screen %s", travelWitnessMission, w.app.Screen())
	}
	arrived := w.screen.WorldMapView()
	if arrived.Position != w.anchor || w.screen.worldPosition != w.anchor || arrived.Selected != -1 || len(arrived.Route) != 0 {
		w.t.Fatalf("arrival: position %v (field %v), selected %d, route %d points; want %v with the route cleared",
			arrived.Position, w.screen.worldPosition, arrived.Selected, len(arrived.Route), w.anchor)
	}
	return ticks
}

// TestReleaseWorldMapTravelInputsKeepTravelRunning loads the owner's city SAV
// through the Load window, opens the gates and clicks mission 30's scroll, the
// only scroll offered, then sends input through App while the route reveals.
// After each of the first four reveal ticks one of Up, Down, the wheel up and
// the wheel down leaves the route and its reveal as they are, and the mission
// opens on the tick the route's length gives (DIV-1515). A click that misses
// every scroll before the first reveal tick changes nothing, and the same click
// after it finishes the reveal so that the next tick opens the mission
// (`TOWN-121`, DIV-1516).
func TestReleaseWorldMapTravelInputsKeepTravelRunning(t *testing.T) {
	t.Run("Up, Down and the wheel", func(t *testing.T) {
		w := startTravelWitness(t)
		w.reveal(0)
		shown := worldMapRouteReveal
		for _, input := range []string{"up", "down", "wheel-up", "wheel-down"} {
			w.input(input)
			w.steady(input, shown)
			t.Logf("%s with %d of %d coordinates revealed: %d after it", input, shown, len(w.route), w.screen.WorldMapView().RouteShown)
			w.reveal(shown)
			shown += worldMapRouteReveal
		}
		total := shown/worldMapRouteReveal + w.finish(shown)
		if want := (len(w.route) + worldMapRouteReveal - 1) / worldMapRouteReveal; total != want {
			t.Fatalf("after Up, Down and the wheel the mission opened on tick %d, want tick %d", total, want)
		}
		t.Logf("the mission opened on tick %d of %d coordinates, the tick the route's length gives; the party arrived at %v",
			total, len(w.route), w.anchor)
	})
	t.Run("a click before the first reveal tick", func(t *testing.T) {
		w := startTravelWitness(t)
		w.steady("before the click", 0)
		w.point("press", w.missX, w.missY)
		w.steady("the press", 0)
		w.point("release", w.missX, w.missY)
		w.steady("a click that misses every scroll before the first reveal tick", 0)
		t.Logf("a click that misses every scroll before the first reveal tick: nothing revealed, the travel untouched")
		w.reveal(0)
		w.point("press", w.missX, w.missY)
		w.point("release", w.missX, w.missY)
		if v := w.screen.WorldMapView(); v.RouteShown != len(w.route) || v.Selected != 0 || w.app.Screen() != ui.ScreenTown {
			t.Fatalf("a missed click with %d of %d coordinates revealed: revealed %d, selected %d, screen %s; want the route's own end and the gates open",
				worldMapRouteReveal, len(w.route), v.RouteShown, v.Selected, w.app.Screen())
		}
		if ticks := w.finish(len(w.route)); ticks != 1 {
			t.Fatalf("the mission opened %d ticks after the skip, want the next one", ticks)
		}
		t.Logf("a click that misses every scroll with %d of %d coordinates revealed finished the reveal; the next tick brought the party to %v and opened mission %d",
			worldMapRouteReveal, len(w.route), w.anchor, travelWitnessMission)
	})
}
