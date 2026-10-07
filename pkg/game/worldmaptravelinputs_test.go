package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// travelInputsScreen is a world map at the party's home with one enabled scroll
// per anchor. The home is the town's own point when no registry is loaded, and
// the route to an anchor is the sampled fallback line.
func travelInputsScreen(home image.Point, anchors ...image.Point) *townScreen {
	s := (&FrontEnd{}).bindTown(&townScreen{worldMap: &worldMapState{
		selected: -1, hovered: -1, assets: &worldMapAssets{}, current: home,
	}})
	for i, anchor := range anchors {
		s.worldMap.missions = append(s.worldMap.missions,
			ui.WorldMapMission{Number: 30 + 10*i, Title: "T", Enabled: true, Anchor: anchor})
	}
	return s
}

// TestWorldMapMovesKeepTravelToTheOnlyMissionRunning checks DIV-1515. With one
// mission offered, Up and Down, which the wheel also sends, leave the route and
// its revealed prefix as they are between every reveal tick, and the party
// arrives on the tick the ticks alone give. With two missions offered a move
// still selects the other one and starts its travel from nothing.
func TestWorldMapMovesKeepTravelToTheOnlyMissionRunning(t *testing.T) {
	home := image.Pt(320, 240)
	anchors := []image.Point{{X: 0, Y: 0}, {X: 620, Y: 470}}

	s := travelInputsScreen(home, anchors[0])
	s.WorldMapMove(+1) // the first Down, or wheel notch down, starts the travel
	first := s.WorldMapView()
	route := first.Route
	if first.Selected != 0 || first.RouteShown != 0 || len(route) < 2 || route[0] != home || route[len(route)-1] != anchors[0] {
		t.Fatalf("first move: selected %d, shown %d, route %v; want mission 30's route from %v with nothing shown",
			first.Selected, first.RouteShown, route, home)
	}
	want := (len(route) + worldMapRouteReveal - 1) / worldMapRouteReveal
	moves, tick := 0, 1
	var arrival ui.TownAction
	for ; tick <= 100; tick++ {
		if arrival = s.WorldMapTick(); arrival.Open != nil {
			break
		}
		for _, delta := range []int{-1, +1} {
			before := s.WorldMapView()
			s.WorldMapMove(delta)
			moves++
			after := s.WorldMapView()
			if after.Selected != 0 || after.RouteShown != before.RouteShown || !reflect.DeepEqual(after.Route, route) {
				t.Fatalf("move %+d after tick %d restarted the travel: revealed %d -> %d of %d, selected %d, route %d points",
					delta, tick, before.RouteShown, after.RouteShown, len(route), after.Selected, len(after.Route))
			}
		}
		if got := s.WorldMapView().RouteShown; got != tick*worldMapRouteReveal {
			t.Fatalf("after tick %d the route shows %d coordinates, want %d", tick, got, tick*worldMapRouteReveal)
		}
	}
	if tick != want || arrival.Msg != "travelling to mission 30" || s.worldPosition != anchors[0] {
		t.Fatalf("after %d moves: arrival on tick %d at %v with %q, want tick %d at %v to mission 30",
			moves, tick, s.worldPosition, arrival.Msg, want, anchors[0])
	}

	two := travelInputsScreen(home, anchors...)
	two.WorldMapMove(+1)
	two.WorldMapTick()
	two.WorldMapTick()
	two.WorldMapMove(+1)
	other := two.WorldMapView()
	if other.Selected != 1 || other.RouteShown != 0 || len(other.Route) < 2 || other.Route[0] != home || other.Route[len(other.Route)-1] != anchors[1] {
		t.Fatalf("a move to the other mission: selected %d, shown %d, route %v; want mission 40's route from %v with nothing shown",
			other.Selected, other.RouteShown, other.Route, home)
	}
	for tick := 1; ; tick++ {
		if tick > 100 {
			t.Fatal("the travel to the other mission never arrived")
		}
		if act := two.WorldMapTick(); act.Open != nil {
			if act.Msg != "travelling to mission 40" || two.worldPosition != anchors[1] {
				t.Fatalf("arrival at %v with %q, want mission 40 at %v", two.worldPosition, act.Msg, anchors[1])
			}
			break
		}
	}
}

// TestWorldMapMissedClickBeforeTheFirstRevealTickLeavesTravelRunning checks
// DIV-1516. A click that misses every scroll before the first reveal tick
// changes nothing, on the outward trip and on the homeward trip alike. After
// the first tick the same click finishes the reveal and the next tick arrives,
// which TOWN-121 states for non-zero route progress.
func TestWorldMapMissedClickBeforeTheFirstRevealTickLeavesTravelRunning(t *testing.T) {
	home, anchor := image.Pt(320, 240), image.Pt(0, 0)
	miss := image.Pt(2, 2)
	s := travelInputsScreen(home, anchor)
	if _, ok := ui.WorldMapCardAt(s.WorldMapView(), miss); ok {
		t.Fatal("setup: the chosen miss point hits a card")
	}
	if _, ok := ui.WorldMapRegionAt(s.WorldMapView(), miss); ok {
		t.Fatal("setup: the chosen miss point hits a mission region")
	}
	card := ui.WorldMapCardRect(0)
	s.WorldMapClick(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2))
	started := s.WorldMapView()
	route := started.Route
	if started.Selected != 0 || started.RouteShown != 0 || len(route) < 2*worldMapRouteReveal {
		t.Fatalf("scroll click: selected %d, shown %d, route %d points; want a route of at least two reveal ticks with nothing shown",
			started.Selected, started.RouteShown, len(route))
	}

	if act := s.WorldMapClick(miss); act.Open != nil {
		t.Fatal("a missed click opened the mission")
	}
	early := s.WorldMapView()
	if early.Selected != 0 || early.RouteShown != 0 || !reflect.DeepEqual(early.Route, route) {
		t.Fatalf("a missed click before the first tick: selected %d, revealed %d of %d, route %d points; want the travel as it was",
			early.Selected, early.RouteShown, len(route), len(early.Route))
	}
	if act := s.WorldMapTick(); act.Open != nil || s.WorldMapView().RouteShown != worldMapRouteReveal {
		t.Fatalf("first tick after the early click: revealed %d, opened %v; want %d and not opened",
			s.WorldMapView().RouteShown, act.Open != nil, worldMapRouteReveal)
	}
	s.WorldMapClick(miss)
	if shown := s.WorldMapView().RouteShown; shown != len(route) {
		t.Fatalf("a missed click after the first tick left %d of %d coordinates revealed, want the route's own end", shown, len(route))
	}
	if act := s.WorldMapTick(); act.Open == nil || act.Msg != "travelling to mission 30" || s.worldPosition != anchor {
		t.Fatalf("tick after the skip: opened %v with %q at %v, want mission 30 at %v", act.Open != nil, act.Msg, s.worldPosition, anchor)
	}

	f := restoredWorldMapFront(t, false, true)
	r := f.TownScreen().(*townScreen)
	r.Choose(3)
	back := r.WorldMapView()
	if !back.Returning || back.RouteShown != 0 || len(back.Route) < 2*worldMapRouteReveal {
		t.Fatalf("homeward trip: returning %v, shown %d, route %d points; want a route of at least two reveal ticks with nothing shown",
			back.Returning, back.RouteShown, len(back.Route))
	}
	r.WorldMapClick(miss)
	if shown := r.WorldMapView().RouteShown; shown != 0 || !r.AtWorldMap() {
		t.Fatalf("a missed click before the homeward trip's first tick: revealed %d, gates open %v; want 0 and open", shown, r.AtWorldMap())
	}
	r.WorldMapTick()
	r.WorldMapClick(miss)
	if shown := r.WorldMapView().RouteShown; shown != len(back.Route) {
		t.Fatalf("a missed click after the homeward trip's first tick left %d of %d coordinates revealed, want the route's own end", shown, len(back.Route))
	}
	r.WorldMapTick()
	if !r.AtTownSquare() {
		t.Fatal("the tick after the skip did not bring the party home")
	}
}
