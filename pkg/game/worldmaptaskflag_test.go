package game

import (
	"image"
	"testing"

	"againrom/pkg/ui"
)

// TestWorldMapTaskFlagAnimatesAndTheCrossHolds walks the paint state from hover
// to arrival. Flag1's counter advances on every tick, drawn or not, and wraps
// around its sheet (`TOWN-529`); the flag is drawn at the hovered task, then at
// the chosen task through every route step with the pointer elsewhere
// (`TOWN-528`). The Cross plays its sheet once and holds the last frame
// (`TOWN-530`) until the party arrives, when neither marker is drawn.
func TestWorldMapTaskFlagAnimatesAndTheCrossHolds(t *testing.T) {
	const flagFrames, crossFrames = 9, 13
	home, anchor := image.Pt(320, 240), image.Pt(320, 160)
	s := crossScreen(crossFrames, home, anchor)
	flags := crossSheet(flagFrames)
	s.worldMap.assets.available = flags
	crosses := s.worldMap.assets.cross
	flagAt := anchor.Add(image.Pt(-4, -32))

	drawn := func(when string, ticks int, want bool) {
		t.Helper()
		v := s.WorldMapView()
		if got, wantFrame := v.Available, flags[ticks%flagFrames]; got != image.Image(wantFrame) {
			t.Fatalf("%s: Flag1 frame %v, want frame %d of counter %d", when, got, ticks%flagFrames, ticks)
		}
		c := ui.ComposeWorldMap(v).RGBAAt(flagAt.X, flagAt.Y)
		if on := c == flags[ticks%flagFrames].RGBAAt(0, 0); on != want {
			t.Fatalf("%s: Flag1 drawn at the task %v, want %v (pixel %v)", when, on, want, c)
		}
	}

	ticks := 0
	for ; ticks < 2; ticks++ {
		s.WorldMapTick()
	}
	drawn("two idle ticks, nothing hovered", ticks, false)
	card := ui.WorldMapCardRect(0)
	s.WorldMapHover(image.Pt((card.Min.X+card.Max.X)/2, (card.Min.Y+card.Max.Y)/2))
	drawn("hover over the task's scroll", ticks, true)
	clickScroll(s, 0)
	s.WorldMapHover(image.Pt(600, 400))
	drawn("the choice, pointer moved away", ticks, true)
	if got := s.WorldMapView().Cross; got != image.Image(crosses[0]) {
		t.Fatalf("Cross at the choice %v, want frame 0", got)
	}
	route := s.WorldMapView().Route
	if revealTicks(route) >= crossFrames {
		t.Fatalf("setup: the route reveals in %d ticks, want fewer than the Cross sheet's %d frames", revealTicks(route), crossFrames)
	}
	for step := 1; ; step++ {
		ticks++
		act := s.WorldMapTick()
		if act.Open != nil {
			if step != crossFrames {
				t.Fatalf("arrived on route step %d, want step %d", step, crossFrames)
			}
			break
		}
		v := s.WorldMapView()
		if v.Selected != 0 || !v.AtHome {
			t.Fatalf("route step %d: selected %d, at home %v", step, v.Selected, v.AtHome)
		}
		drawn("a route step", ticks, true)
		if want := crosses[min(step, crossFrames-1)]; v.Cross != image.Image(want) {
			t.Fatalf("route step %d: Cross %v, want frame %d", step, v.Cross, min(step, crossFrames-1))
		}
	}
	v := s.WorldMapView()
	if v.Selected != -1 || v.AtHome || len(v.Route) != 0 {
		t.Fatalf("arrival: selected %d, at home %v, route %d points", v.Selected, v.AtHome, len(v.Route))
	}
	drawn("arrival", ticks, false)
}

// TestWorldMapCrossHoldsItsLastFrameAfterASkip checks TOWN-530's clamp on the
// counter a skip assigns (TOWN-486): the frame count plus one draws the
// sheet's last frame, not a wrapped one.
func TestWorldMapCrossHoldsItsLastFrameAfterASkip(t *testing.T) {
	const frames = 13
	s := crossScreen(frames, image.Pt(320, 240), image.Pt(0, 0))
	clickScroll(s, 0)
	s.WorldMapTick()
	s.WorldMapClick(image.Pt(2, 2))
	if got, want := s.WorldMapView().Cross, s.worldMap.assets.cross[frames-1]; got != image.Image(want) {
		t.Fatalf("Cross after a skip %v, want the last frame %v", got, want)
	}
}
