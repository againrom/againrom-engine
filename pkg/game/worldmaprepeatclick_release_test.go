package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// TestReleaseWorldMapRepeatedScrollClickKeepsTravelRunning loads the owner's
// city SAV through the Load window, opens the gates and clicks mission 30's
// scroll, then clicks it twelve more times while the route reveals, through
// App's ordinary pointer input (DIV-1479). No click changes the route or moves
// its revealed prefix back. The reveal advances eight coordinates per tick to
// the route's end, the party arrives at the mission's MapPoint and mission 30
// opens. No click misses a scroll, so the skip arm never runs.
func TestReleaseWorldMapRepeatedScrollClickKeepsTravelRunning(t *testing.T) {
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
	const mission = 30
	s := f.TownScreen().(*townScreen)
	gates := s.WorldMapView()
	index := worldMarkerMissionIndex(t, gates, mission)
	anchor := gates.Missions[index].Anchor
	if !gates.AtHome || gates.HideScrolls || gates.Selected != -1 || len(gates.Route) != 0 {
		t.Fatalf("gates before the first click: at home %v, scrolls hidden %v, selected %d, route %d points",
			gates.AtHome, gates.HideScrolls, gates.Selected, len(gates.Route))
	}
	x, y, err := app.HeadlessWorldMapMissionPoint(mission)
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, x, y); err != nil {
			t.Fatal(err)
		}
	}
	first := s.WorldMapView()
	route := first.Route
	if first.Selected != index || first.RouteShown != 0 || len(route) < 2 || route[0] != gates.Position || route[len(route)-1] != anchor {
		t.Fatalf("first click: selected %d, shown %d, route %d points; want mission %d's route from %v to %v with nothing shown",
			first.Selected, first.RouteShown, len(route), mission, gates.Position, anchor)
	}
	ticks := (len(route) + worldMapRouteReveal - 1) / worldMapRouteReveal
	t.Logf("first click on mission %d's scroll: route of %d coordinates from %v to %v, %d reveal ticks",
		mission, len(route), gates.Position, anchor, ticks)

	shown, clicks := 0, 0
	var progress []int
	// dispatch sends one idle frame or pointer edge and reports whether the
	// gates still show; arrival opens the mission, and no edge follows it.
	dispatch := func(edge string) bool {
		t.Helper()
		before := shown
		var err error
		if edge == "" {
			err = app.HeadlessStep()
		} else {
			err = app.HeadlessPointer(edge, x, y)
		}
		if err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenTown {
			return false
		}
		v := s.WorldMapView()
		if v.Selected != index || !reflect.DeepEqual(v.Route, route) || v.RouteShown < shown || v.RouteShown > shown+worldMapRouteReveal {
			t.Fatalf("%q after %d repeated clicks restarted the travel: revealed %d -> %d of %d, selected %d, route %d points",
				edge, clicks, shown, v.RouteShown, len(route), v.Selected, len(v.Route))
		}
		if v.RouteShown > shown {
			shown = v.RouteShown
			progress = append(progress, shown)
		}
		if edge == "release" {
			clicks++
			t.Logf("repeated click %d with %d of %d coordinates revealed: %d after it", clicks, before, len(route), shown)
		}
		return true
	}
	// A reveal tick falls on every fifth dispatch, 100 ms of the headless
	// clock, so each cycle of a press, a release and three idle frames spans
	// one tick.
	for n := 0; n < 5000; n++ {
		edge := ""
		if n >= 4 && clicks < 12 {
			edge = [...]string{"press", "release", "", "", ""}[(n-4)%5]
		}
		if !dispatch(edge) {
			break
		}
	}

	if app.Screen() != ui.ScreenMap || f.live == nil || f.live.mission.number != mission {
		t.Fatalf("the travel did not open mission %d: screen %s", mission, app.Screen())
	}
	if clicks < 10 || len(progress) != ticks-1 || shown != (ticks-1)*worldMapRouteReveal {
		t.Fatalf("%d repeated clicks; the reveal showed %v before arrival, want %d ticks of %d coordinates after 10 or more clicks",
			clicks, progress, ticks-1, worldMapRouteReveal)
	}
	arrived := s.WorldMapView()
	if arrived.Position != anchor || s.worldPosition != anchor || arrived.Selected != -1 || len(arrived.Route) != 0 {
		t.Fatalf("arrival: position %v (field %v), selected %d, route %d points; want %v with the route cleared",
			arrived.Position, s.worldPosition, arrived.Selected, len(arrived.Route), anchor)
	}
	t.Logf("%d repeated clicks; the reveal went %v and its tick %d reached all %d coordinates; the party arrived at %v and mission %d opened",
		clicks, progress, ticks, len(route), anchor, mission)
}
