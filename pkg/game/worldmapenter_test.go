package game

import (
	"image"
	"testing"

	"againrom/pkg/ui"
)

// travelObserved is what a player sees of a world-map travel: the selection,
// how much of the route is drawn, the party's own position, the Cross frame and
// whether the trip is the homeward one.
type travelObserved struct {
	selected, shown, routeLen int
	position                  image.Point
	cross                     image.Image
	returning                 bool
}

func observeTravel(s *townScreen) travelObserved {
	v := s.WorldMapView()
	return travelObserved{v.Selected, v.RouteShown, len(v.Route), v.Position, v.Cross, v.Returning}
}

// TestWorldMapEnterDoesWhatAMissedClickDoes checks DIV-1546. Enter opens
// nothing itself. At every moment of an outward travel it leaves the travel
// exactly as a click that misses every scroll leaves it: untouched at zero
// progress, and from the first reveal tick on the reveal at the route's own end
// and the Cross counter at the frame count plus one (`TOWN-485`, `TOWN-486`),
// so that the next tick arrives. A second Enter changes nothing more.
func TestWorldMapEnterDoesWhatAMissedClickDoes(t *testing.T) {
	const frames = 12
	home, anchor := image.Pt(320, 240), image.Pt(0, 0)
	miss := image.Pt(2, 2)
	probe := crossScreen(frames, home, anchor)
	if _, ok := ui.WorldMapCardAt(probe.WorldMapView(), miss); ok {
		t.Fatal("setup: the chosen miss point hits a card")
	}
	clickScroll(probe, 0)
	reveal := revealTicks(probe.WorldMapView().Route)
	if reveal < 3 || reveal+1 >= frames {
		t.Fatalf("setup: %d reveal ticks, want three or more and at least two fewer than the Cross sheet's %d frames", reveal, frames)
	}
	for _, tc := range []struct {
		name  string
		ticks int
	}{
		{"before the first reveal tick", 0},
		{"during the reveal", 1},
		{"on the reveal's last tick", reveal},
		{"after the reveal, during the Cross animation", reveal + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			enter, click := crossScreen(frames, home, anchor), crossScreen(frames, home, anchor)
			click.worldMap.assets.cross = enter.worldMap.assets.cross // one sheet, so equal frames compare equal
			clickScroll(enter, 0)
			clickScroll(click, 0)
			for i := 0; i < tc.ticks; i++ {
				enter.WorldMapTick()
				click.WorldMapTick()
			}
			if act := enter.WorldMapChoose(); act.Open != nil || act.Msg != "" {
				t.Fatalf("Enter after %d ticks opened %v with %q; it opens nothing itself", tc.ticks, act.Open != nil, act.Msg)
			}
			click.WorldMapClick(miss)
			got, want := observeTravel(enter), observeTravel(click)
			if got != want {
				t.Fatalf("Enter after %d ticks left %+v, a missed click at the same moment leaves %+v", tc.ticks, got, want)
			}
			skipped := tc.ticks > 0
			if skipped && got.cross != enter.worldMap.assets.cross[frames-1] {
				t.Fatalf("Enter after %d ticks drew Cross frame %v, want the held last frame for counter %d, %v", tc.ticks, got.cross, frames+1, enter.worldMap.assets.cross[frames-1])
			}
			wantShown := 0
			if skipped {
				wantShown = got.routeLen
			}
			if got.shown != wantShown {
				t.Fatalf("Enter after %d ticks left %d of %d coordinates drawn, want %d", tc.ticks, got.shown, got.routeLen, wantShown)
			}
			enter.WorldMapChoose()
			if again := observeTravel(enter); again != got {
				t.Fatalf("a second Enter changed %+v to %+v", got, again)
			}
			wantTicks := 1
			if !skipped {
				wantTicks = max(reveal, frames)
			}
			for tick := 1; ; tick++ {
				if tick > 2*frames {
					t.Fatalf("no arrival within %d ticks after Enter at tick %d", tick-1, tc.ticks)
				}
				a, b := enter.WorldMapTick(), click.WorldMapTick()
				if (a.Open != nil) != (b.Open != nil) || a.Msg != b.Msg || enter.worldPosition != click.worldPosition {
					t.Fatalf("tick %d after the input: Enter opened %v with %q at %v, the missed click %v with %q at %v",
						tick, a.Open != nil, a.Msg, enter.worldPosition, b.Open != nil, b.Msg, click.worldPosition)
				}
				if a.Open == nil {
					continue
				}
				if tick != wantTicks || a.Msg != "travelling to mission 30" || enter.worldPosition != anchor {
					t.Fatalf("opened %q at %v on tick %d after the input, want mission 30 at %v on tick %d",
						a.Msg, enter.worldPosition, tick, anchor, wantTicks)
				}
				return
			}
		})
	}
}

// TestWorldMapEnterWithNoSelectionDoesNothing checks DIV-1546. With missions
// offered and none selected, Enter selects nothing and starts no travel, as a
// click that misses every scroll does, and ticks afterwards open nothing.
func TestWorldMapEnterWithNoSelectionDoesNothing(t *testing.T) {
	home := image.Pt(320, 240)
	s := crossScreen(12, home, image.Pt(0, 0), image.Pt(400, 240))
	before := observeTravel(s)
	if before.selected != -1 || before.routeLen != 0 {
		t.Fatalf("setup: selected %d, route %d points; want nothing selected", before.selected, before.routeLen)
	}
	if act := s.WorldMapChoose(); act.Open != nil || act.Msg != "" {
		t.Fatalf("Enter with nothing selected opened %v with %q", act.Open != nil, act.Msg)
	}
	if got := observeTravel(s); got != before {
		t.Fatalf("Enter with nothing selected changed %+v to %+v", before, got)
	}
	for tick := 1; tick <= 40; tick++ {
		if act := s.WorldMapTick(); act.Open != nil {
			t.Fatalf("tick %d after Enter with nothing selected opened %q", tick, act.Msg)
		}
	}
	if got := observeTravel(s); got.selected != -1 || got.routeLen != 0 || got.position != home {
		t.Fatalf("40 ticks after Enter with nothing selected: %+v, want nothing selected and the party at %v", got, home)
	}
}

// TestWorldMapEnterHastensTheHomewardTripAsAMissedClickDoes checks DIV-1546 on
// the homeward trip, which the missed click skips the same way (DIV-1516): Enter
// before the first reveal tick changes nothing, Enter after it finishes the
// reveal, and the next tick brings the party home.
func TestWorldMapEnterHastensTheHomewardTripAsAMissedClickDoes(t *testing.T) {
	const frames = 12
	f := restoredWorldMapFront(t, false, true)
	f.worldMapAssets().cross = crossSheet(frames)
	r := f.TownScreen().(*townScreen)
	r.Choose(3)
	back := r.WorldMapView()
	if !back.Returning || back.RouteShown != 0 || revealTicks(back.Route) < 2 || revealTicks(back.Route) >= frames {
		t.Fatalf("setup: returning %v, shown %d, %d reveal ticks; want a homeward route that reveals in two or more ticks and fewer than %d",
			back.Returning, back.RouteShown, revealTicks(back.Route), frames)
	}
	if act := r.WorldMapChoose(); act.Open != nil || act.Msg != "" {
		t.Fatalf("Enter before the first homeward tick opened %v with %q", act.Open != nil, act.Msg)
	}
	if got := observeTravel(r); got.shown != 0 || !got.returning || !r.AtWorldMap() {
		t.Fatalf("Enter before the first homeward tick: %+v, gates open %v; want the trip untouched", got, r.AtWorldMap())
	}
	r.WorldMapTick()
	if act := r.WorldMapChoose(); act.Open != nil || act.Msg != "" {
		t.Fatalf("Enter after the first homeward tick opened %v with %q", act.Open != nil, act.Msg)
	}
	if got := observeTravel(r); got.shown != len(back.Route) || got.cross != r.worldMap.assets.cross[len(r.worldMap.assets.cross)-1] || !r.AtWorldMap() {
		t.Fatalf("Enter after the first homeward tick: %+v, gates open %v; want %d of %d coordinates drawn, the Cross counter at the frame count plus one and the party still on its way",
			got, r.AtWorldMap(), len(back.Route), len(back.Route))
	}
	if act := r.WorldMapTick(); act.Open != nil || !r.AtTownSquare() {
		t.Fatalf("the tick after Enter: opened %v, at the square %v; want the party home", act.Open != nil, r.AtTownSquare())
	}
}

// TestWorldMapEnterIgnoresTheSelection checks DIV-1546 and `TOWN-486`: the
// helper reads no selection, so Enter on a selected mission that cannot be
// travelled to (no route, zero progress) reports nothing and changes nothing.
func TestWorldMapEnterIgnoresTheSelection(t *testing.T) {
	home := image.Pt(320, 240)
	s := travelInputsScreen(home, image.Pt(0, 0))
	s.worldMap.missions[0].Enabled, s.worldMap.missions[0].Problem = false, "Mission 30 has no valid map location"
	clickScroll(s, 0)
	before := observeTravel(s)
	if before.selected != 0 || before.routeLen != 0 {
		t.Fatalf("setup: selected %d, route %d points; want the disabled mission selected with no route", before.selected, before.routeLen)
	}
	if act := s.WorldMapChoose(); act.Open != nil || act.Msg != "" {
		t.Fatalf("Enter on the disabled mission opened %v with %q, want nothing", act.Open != nil, act.Msg)
	}
	if got := observeTravel(s); got != before {
		t.Fatalf("Enter on the disabled mission changed %+v to %+v", before, got)
	}
}

// TestWorldMapEnterAssignsTheCrossCounterWhateverItHeld checks `TOWN-486`: the
// helper stores frame count plus one even over a larger counter, which a reveal
// that outlasts the Cross sheet leaves behind.
func TestWorldMapEnterAssignsTheCrossCounterWhateverItHeld(t *testing.T) {
	const frames = 12
	s := crossScreen(frames, image.Pt(320, 240), image.Pt(0, 0))
	clickScroll(s, 0)
	s.WorldMapTick()
	s.worldMap.cross = 3 * frames
	s.WorldMapChoose()
	if got := s.worldMap.cross; got != frames+1 {
		t.Fatalf("Enter over a Cross counter of %d left %d, want %d", 3*frames, got, frames+1)
	}
}
