package game

import (
	"image"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/ui"
)

// crossWitnessMission is the chapter-40 main mission, whose installed route
// from the city reveals in fewer ticks than the installed Cross sheet has
// frames.
const crossWitnessMission = 40

// crossWitness is a town at chapter 40 loaded through App's Load keys with the
// gates open, and once clickScroll has run, mission 40's scroll clicked through
// App's pointer input. Every later input goes through App as well.
type crossWitness struct {
	t      *testing.T
	app    *ui.App
	front  *FrontEnd
	screen *townScreen
	index  int
	route  []image.Point
	home   image.Point
	anchor image.Point
	// frame0 is the world map's tick counter when the scroll click returned;
	// ticks are counted from it. frames is the installed Cross sheet's length.
	frame0, frames int
	missX, missY   int
}

// startCrossWitness opens the gates and clicks mission 40's scroll.
func startCrossWitness(t *testing.T) *crossWitness {
	t.Helper()
	w := openCrossWitness(t)
	w.clickScroll()
	return w
}

// openCrossWitness loads the city and opens its gates with nothing selected.
func openCrossWitness(t *testing.T) *crossWitness {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("world-map-cross-release")
	app.Layout(1024, 768)
	t.Cleanup(app.StopAudio)
	store := SaveStore{Dir: t.TempDir()}
	saved := 0
	save, list, load := f.SaveSeams(store, OriginalStore{}, func() time.Time {
		saved++
		return time.Date(2001, 1, 1, 0, 0, saved, 0, time.UTC)
	})
	app.SetSaveSeams(save, list, load)
	// Controlled precondition: the city at chapter 40 with its offers taken,
	// booted through App from a native SAV. Everything after it is input.
	f.Carried = f.ChargenParty(ui.ChargenResult{Name: "Cross witness", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	f.FinishMission(30, f.Carried, nil, nil)
	for _, building := range []TownBuilding{TownTavern, TownShop, TownSchool} {
		for _, offer := range f.Town.Offers(building) {
			if offer.Mission > 0 {
				if _, ok := f.Town.Take(building, offer.Index); !ok {
					t.Fatalf("could not accept installed offer %+v", offer)
				}
			}
		}
	}
	if !containsMission(f.Town.Available(), crossWitnessMission) {
		t.Fatalf("the installed chapter offered %v, want mission %d among them", f.Town.Available(), crossWitnessMission)
	}
	if _, err := save(false); err != nil {
		t.Fatal(err)
	}
	w := &crossWitness{t: t, app: app, front: f}
	w.key("load")
	w.key("enter")
	w.screen = f.TownScreen().(*townScreen)
	if app.Screen() != ui.ScreenTown || !w.screen.AtTownSquare() {
		t.Fatalf("the city did not load: screen %s, square %v", app.Screen(), w.screen.AtTownSquare())
	}
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	gates := w.screen.WorldMapView()
	w.index = -1
	for i, m := range gates.Missions {
		if m.Number == crossWitnessMission {
			w.index, w.anchor = i, m.Anchor
		}
	}
	if w.index < 0 || !gates.AtHome || gates.HideScrolls || gates.Selected != -1 {
		t.Fatalf("gates before the click: mission %d at scroll %d, at home %v, hidden %v, selected %d",
			crossWitnessMission, w.index, gates.AtHome, gates.HideScrolls, gates.Selected)
	}
	w.home = gates.Position
	var err error
	if w.missX, w.missY, err = app.HeadlessWorldMapMissPoint(); err != nil {
		t.Fatal(err)
	}
	w.frames = len(f.worldMapAssets().cross)
	return w
}

// clickScroll clicks mission 40's scroll through App's pointer input.
func (w *crossWitness) clickScroll() {
	w.t.Helper()
	x, y, err := w.app.HeadlessWorldMapMissionPoint(crossWitnessMission)
	if err != nil {
		w.t.Fatal(err)
	}
	w.point("press", x, y)
	w.point("release", x, y)
	started := w.screen.WorldMapView()
	w.route = started.Route
	w.frame0 = w.screen.worldMap.frame
	if started.Selected != w.index || started.RouteShown != 0 || len(w.route) < 2 || w.route[0] != w.home || w.route[len(w.route)-1] != w.anchor {
		w.t.Fatalf("scroll click: selected %d, shown %d, route %d points; want mission %d's route from %v to %v with nothing shown",
			started.Selected, started.RouteShown, len(w.route), crossWitnessMission, w.home, w.anchor)
	}
	if reveal := revealTicks(w.route); reveal < 3 || reveal >= w.frames {
		w.t.Fatalf("mission %d: route of %d coordinates reveals in %d ticks and the Cross sheet has %d frames; want a reveal of three or more ticks that ends before the Cross animation does",
			crossWitnessMission, len(w.route), reveal, w.frames)
	}
	w.t.Logf("scroll click on mission %d: route of %d coordinates from %v to %v, reveal ends on tick %d, Cross sheet of %d frames",
		crossWitnessMission, len(w.route), w.home, w.anchor, revealTicks(w.route), w.frames)
}

func (w *crossWitness) key(name string) {
	w.t.Helper()
	if err := w.app.HeadlessKey(name); err != nil {
		w.t.Fatal(err)
	}
}

func (w *crossWitness) point(edge string, x, y int) {
	w.t.Helper()
	if err := w.app.HeadlessPointer(edge, x, y); err != nil {
		w.t.Fatal(err)
	}
}

func (w *crossWitness) ticks() int { return w.screen.worldMap.frame - w.frame0 }

// steady fails unless the travel is as the scroll click left it with shown
// coordinates revealed, the party still at its starting point and the gates
// open.
func (w *crossWitness) steady(what string, shown int) {
	w.t.Helper()
	v := w.screen.WorldMapView()
	if w.app.Screen() != ui.ScreenTown || v.Selected != w.index || !reflect.DeepEqual(v.Route, w.route) || v.RouteShown != shown || v.Position != w.home {
		w.t.Fatalf("%s at tick %d: screen %s, selected %d, revealed %d of %d, position %v; want the travel under way with %d revealed from %v",
			what, w.ticks(), w.app.Screen(), v.Selected, v.RouteShown, len(w.route), v.Position, shown, w.home)
	}
}

// idleUntil steps App until the mission opens, or until stop reports true after
// a tick, and returns the tick count. Every tick must add one reveal step and
// change nothing else until the opening.
func (w *crossWitness) idleUntil(stop func(tick int) bool) int {
	w.t.Helper()
	last := w.ticks()
	for n := 0; w.app.Screen() == ui.ScreenTown; n++ {
		if n > 10*(w.frames+len(w.route)/worldMapRouteReveal+2) {
			w.t.Fatalf("no opening within %d idle frames, at tick %d", n, w.ticks())
		}
		if err := w.app.HeadlessStep(); err != nil {
			w.t.Fatal(err)
		}
		if w.app.Screen() != ui.ScreenTown {
			break
		}
		if tick := w.ticks(); tick != last {
			if tick != last+1 {
				w.t.Fatalf("one frame ran %d ticks", tick-last)
			}
			last = tick
			shown := tick * worldMapRouteReveal
			if shown > len(w.route) {
				shown = len(w.route)
			}
			w.steady("a tick", shown)
			if stop != nil && stop(tick) {
				return tick
			}
		}
	}
	return w.ticks()
}

// opened fails unless mission 40 is open and the party arrived at its scroll.
func (w *crossWitness) opened() {
	w.t.Helper()
	if w.app.Screen() != ui.ScreenMap || w.front.live == nil || w.front.live.mission.number != crossWitnessMission {
		w.t.Fatalf("mission %d is not open: screen %s", crossWitnessMission, w.app.Screen())
	}
	v := w.screen.WorldMapView()
	if v.Position != w.anchor || w.screen.worldPosition != w.anchor || v.Selected != -1 || len(v.Route) != 0 {
		w.t.Fatalf("arrival: position %v (field %v), selected %d, route %d points; want %v with the route cleared",
			v.Position, w.screen.worldPosition, v.Selected, len(v.Route), w.anchor)
	}
}

// TestReleaseWorldMapArrivalWaitsForTheCrossAnimation clicks the scroll of
// mission 40 on a chapter-40 city, whose installed route reveals in 8 ticks
// while the installed Cross sheet has 13 frames, and lets App's own cadence run.
// The reveal is complete and the party still at the city on tick 8, and the
// mission opens on tick 13, when the Cross animation passes its end
// (`TOWN-121`, DIV-1545). A click that misses every scroll after tick 8 sets
// both counters past their ends, so the next tick opens the mission.
func TestReleaseWorldMapArrivalWaitsForTheCrossAnimation(t *testing.T) {
	t.Run("App cadence alone", func(t *testing.T) {
		w := startCrossWitness(t)
		reveal := revealTicks(w.route)
		opened := w.idleUntil(nil)
		if want := max(reveal, w.frames); opened != want {
			t.Fatalf("the mission opened on tick %d, want tick %d (the reveal ends on tick %d, the Cross sheet has %d frames)",
				opened, want, reveal, w.frames)
		}
		w.opened()
		t.Logf("the reveal ended on tick %d and the mission opened on tick %d, when the %d-frame Cross animation passed its end; the party arrived at %v",
			reveal, opened, w.frames, w.anchor)
	})
	t.Run("a missed click after the reveal", func(t *testing.T) {
		w := startCrossWitness(t)
		reveal := revealTicks(w.route)
		if got := w.idleUntil(func(tick int) bool { return tick == reveal }); got != reveal || w.app.Screen() != ui.ScreenTown {
			t.Fatalf("with the reveal complete on tick %d the mission had already opened on tick %d", reveal, got)
		}
		w.steady("the reveal's last tick", len(w.route))
		w.point("press", w.missX, w.missY)
		w.point("release", w.missX, w.missY)
		clicked := w.ticks()
		w.steady("a click that misses every scroll", len(w.route))
		opened := w.idleUntil(nil)
		if opened != clicked+1 {
			t.Fatalf("the mission opened on tick %d, want the tick after the click, tick %d", opened, clicked+1)
		}
		w.opened()
		t.Logf("a click that misses every scroll on tick %d, with the reveal complete and the Cross animation at frame %d of %d, opened the mission on tick %d",
			clicked, clicked, w.frames, opened)
	})
}
