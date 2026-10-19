package game

import (
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/ui"
)

// crossSheet is n one-pixel frames of different colours, so the frame a view
// carries names the counter that picked it.
func crossSheet(n int) []*image.RGBA {
	frames := make([]*image.RGBA, n)
	for i := range frames {
		frames[i] = image.NewRGBA(image.Rect(0, 0, 1, 1))
		frames[i].SetRGBA(0, 0, color.RGBA{R: uint8(i + 1), A: 255})
	}
	return frames
}

// crossScreen is travelInputsScreen with a Cross sheet of the given length.
func crossScreen(frames int, home image.Point, anchors ...image.Point) *townScreen {
	s := travelInputsScreen(home, anchors...)
	s.worldMap.assets.cross = crossSheet(frames)
	return s
}

func clickScroll(s *townScreen, slot int) {
	card := ui.WorldMapCardRect(slot)
	s.WorldMapClick(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2))
}

func revealTicks(route []image.Point) int {
	return (len(route) + worldMapRouteReveal - 1) / worldMapRouteReveal
}

// waitsForTheCross ticks a travel that has just started until it opens
// mission and fails unless it opens on tick want, with the route and the
// party's position as the scroll click left them until then.
func waitsForTheCross(t *testing.T, s *townScreen, route []image.Point, home image.Point, want int, mission string) {
	t.Helper()
	for tick := 1; tick <= want; tick++ {
		act := s.WorldMapTick()
		if act.Open != nil {
			if tick != want || act.Msg != mission {
				t.Fatalf("opened %q on tick %d, want %q on tick %d (the reveal takes %d ticks)", act.Msg, tick, mission, want, revealTicks(route))
			}
			return
		}
		v := s.WorldMapView()
		shown := tick * worldMapRouteReveal
		if shown > len(route) {
			shown = len(route)
		}
		if v.RouteShown != shown || v.Position != home || !reflect.DeepEqual(v.Route, route) {
			t.Fatalf("tick %d: revealed %d of %d, position %v, route %d points; want %d revealed, position %v and the route as it was",
				tick, v.RouteShown, len(route), v.Position, len(v.Route), shown, home)
		}
	}
	t.Fatalf("%q did not open by tick %d", mission, want)
}

// TOWN-530 requires two paints beyond the loaded sheet's frame count.
func TestWorldMapArrivalWaitsForTheCrossAnimationToPassItsEnd(t *testing.T) {
	const frames = 6
	home, anchor := image.Pt(320, 240), image.Pt(320, 160)
	s := crossScreen(frames, home, anchor)
	clickScroll(s, 0)
	started := s.WorldMapView()
	if started.Selected != 0 || started.RouteShown != 0 || revealTicks(started.Route) < 2 || revealTicks(started.Route) >= frames {
		t.Fatalf("setup: selected %d, shown %d, %d reveal ticks; want a route that reveals in two or more ticks and fewer than the Cross sheet's %d frames",
			started.Selected, started.RouteShown, revealTicks(started.Route), frames)
	}
	waitsForTheCross(t, s, started.Route, home, frames+2, "travelling to mission 30")
	if s.worldPosition != anchor {
		t.Fatalf("arrival at %v, want %v", s.worldPosition, anchor)
	}
}

func TestWorldMapCrossAnimationContinuesAcrossScrolls(t *testing.T) {
	const frames = 8
	home := image.Pt(320, 240)
	s := crossScreen(frames, home, image.Pt(320, 160), image.Pt(400, 240))
	for i := 0; i < 3; i++ {
		s.WorldMapTick()
	}
	clickScroll(s, 0)
	if s.worldMap.cross != 0 {
		t.Fatal("idle paints advanced the Cross")
	}
	s.WorldMapTick()
	s.WorldMapTick()
	clickScroll(s, 1)
	v := s.WorldMapView()
	if v.Selected != 1 || v.RouteShown != 0 || s.worldMap.cross != 2 || v.Cross != image.Image(s.worldMap.assets.cross[2]) {
		t.Fatalf("second scroll reset the Cross: selected %d shown %d counter %d", v.Selected, v.RouteShown, s.worldMap.cross)
	}
	waitsForTheCross(t, s, v.Route, home, frames, "travelling to mission 40")
	if s.worldMap.cross != frames+2 {
		t.Fatalf("arrival reset the Cross to %d", s.worldMap.cross)
	}
	s.enterWorldMap()
	if s.worldMap.cross != 0 {
		t.Fatal("enter did not reset the Cross")
	}
}

func TestWorldMapArrivalPassesAnExactRouteEnd(t *testing.T) {
	s := crossScreen(1, image.Pt(320, 240), image.Pt(0, 0))
	clickScroll(s, 0)
	s.worldMap.route = make([]image.Point, 24)
	s.worldMap.cross = 10
	for tick := 1; tick <= 3; tick++ {
		if act := s.WorldMapTick(); act.Open != nil {
			t.Fatalf("arrived on tick %d before passing all 24 coordinates", tick)
		}
	}
	if act := s.WorldMapTick(); act.Open == nil {
		t.Fatal("did not arrive after passing the exact route end")
	}
}

// TestWorldMapMissedClickSkipsTheCrossAnimationToo checks TOWN-121: a click
// that misses every scroll once the reveal has begun sets both completion
// counters past their ends, so the next tick opens the mission whether the
// reveal is still running or has finished and the Cross animation has not. The
// Cross counter becomes the sheet's frame count plus one (TOWN-486), so the
// frame on screen is the sheet's held last frame until the next tick (TOWN-530).
func TestWorldMapMissedClickSkipsTheCrossAnimationToo(t *testing.T) {
	const frames = 12
	home, anchor := image.Pt(320, 240), image.Pt(0, 0)
	miss := image.Pt(2, 2)
	for _, tc := range []struct {
		name     string
		finished bool
	}{{"during the reveal", false}, {"after the reveal", true}} {
		t.Run(tc.name, func(t *testing.T) {
			s := crossScreen(frames, home, anchor)
			if _, ok := ui.WorldMapCardAt(s.WorldMapView(), miss); ok {
				t.Fatal("setup: the chosen miss point hits a card")
			}
			clickScroll(s, 0)
			route := s.WorldMapView().Route
			ticks, reveal := 1, revealTicks(route)
			if tc.finished {
				ticks = reveal
			}
			if reveal < 3 || reveal >= frames {
				t.Fatalf("setup: %d reveal ticks, want three or more and fewer than the Cross sheet's %d frames", reveal, frames)
			}
			for tick := 1; tick <= ticks; tick++ {
				if act := s.WorldMapTick(); act.Open != nil {
					t.Fatalf("opened on tick %d, before the click, with the Cross animation at frame %d of %d", tick, tick, frames)
				}
			}
			s.WorldMapClick(miss)
			if shown := s.WorldMapView().RouteShown; shown != len(route) {
				t.Fatalf("the click left %d of %d coordinates revealed, want the route's own end", shown, len(route))
			}
			if got, want := s.WorldMapView().Cross, s.worldMap.assets.cross[frames-1]; got != want {
				t.Fatalf("after the click the drawn Cross frame is %v, want the held last frame for counter %d, %v", got, frames+1, want)
			}
			if act := s.WorldMapTick(); act.Open == nil || act.Msg != "travelling to mission 30" || s.worldPosition != anchor {
				t.Fatalf("tick after the skip: opened %v with %q at %v, want mission 30 at %v", act.Open != nil, act.Msg, s.worldPosition, anchor)
			}
		})
	}
}

// TestWorldMapReturnHomeWaitsForTheCrossAnimationToPassItsEnd checks DIV-1545
// on the homeward trip, which TOWN-121 sends through the same arrival step
// with destination zero: the party reaches the square on the tick the Cross
// animation passes its end, not when the reveal ends.
func TestWorldMapReturnHomeWaitsForTheCrossAnimationToPassItsEnd(t *testing.T) {
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
	for tick := 1; tick <= frames+2; tick++ {
		if act := r.WorldMapTick(); act.Open != nil {
			t.Fatalf("the homeward trip opened a mission on tick %d", tick)
		}
		if r.AtTownSquare() {
			if tick != frames+2 {
				t.Fatalf("the party reached the square on tick %d, want tick %d (the reveal takes %d ticks)", tick, frames+2, revealTicks(back.Route))
			}
			return
		}
		if v := r.WorldMapView(); !v.Returning || v.Position != back.Position {
			t.Fatalf("tick %d: returning %v, position %v; want the trip under way from %v", tick, v.Returning, v.Position, back.Position)
		}
	}
	t.Fatalf("the party did not reach the square by tick %d", frames+2)
}

func TestWorldMapCrossAdvancesOnlyWhenDrawn(t *testing.T) {
	home := image.Pt(320, 240)
	for _, at := range []image.Point{home, image.Pt(300, 220)} {
		s := crossScreen(6, home, at)
		s.worldMap.assets.available = []*image.RGBA{image.NewRGBA(image.Rect(0, 0, 1, 1))}
		clickScroll(s, 0)
		v := s.WorldMapView()
		frame := ui.ComposeWorldMap(v)
		v.Cross = image.NewRGBA(s.worldMap.assets.cross[0].Bounds())
		bare := ui.ComposeWorldMap(v)
		in, out := secondMapChanged(frame, bare, image.Rect(0, 0, 640, 480))
		s.WorldMapTick()
		want := 1
		if at == home {
			want = 0
		}
		if (in > 0 || out > 0) != (want == 1) || s.worldMap.cross != want {
			t.Fatalf("destination %v: changed pixels %d, counter %d; want drawn and advanced %v", at, in+out, s.worldMap.cross, want == 1)
		}
	}
}
