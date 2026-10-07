package game

import (
	"testing"

	"againrom/pkg/ui"
)

// idle steps App for frames frames.
func (w *crossWitness) idle(frames int) {
	w.t.Helper()
	for i := 0; i < frames; i++ {
		if err := w.app.HeadlessStep(); err != nil {
			w.t.Fatal(err)
		}
	}
}

// afterATick steps App until the world map's tick counter moves, so that the
// next tick is a full interval away and the input that follows meets none.
func (w *crossWitness) afterATick() {
	w.t.Helper()
	before := w.screen.worldMap.frame
	for n := 0; w.screen.worldMap.frame == before; n++ {
		if n > 10 {
			w.t.Fatalf("no tick within %d idle frames", n)
		}
		w.idle(1)
	}
}

// TestReleaseWorldMapEnterHastensTravel presses Enter through App on a
// chapter-40 city whose gates offer mission 40, a route of 64 coordinates whose
// reveal ends on tick 8 while the installed Cross sheet has 13 frames
// (DIV-1546). With nothing selected Enter starts no
// travel. Before the first reveal tick it changes nothing. During the reveal and
// after it, while the Cross animation runs, it does what a click that misses
// every scroll does (`TOWN-121`): the reveal completes with the Cross counter at
// the frame count plus one and the mission opens on the next tick.
func TestReleaseWorldMapEnterHastensTravel(t *testing.T) {
	t.Run("nothing selected", func(t *testing.T) {
		w := openCrossWitness(t)
		w.key("enter")
		w.idle(60)
		v := w.screen.WorldMapView()
		if w.app.Screen() != ui.ScreenTown || !w.screen.AtWorldMap() || v.Selected != -1 || len(v.Route) != 0 || v.RouteShown != 0 || v.Position != w.home {
			t.Fatalf("Enter with nothing selected, then 12 ticks: screen %s, gates open %v, selected %d, route %d points, revealed %d, position %v; want the gates open at %v with nothing selected",
				w.app.Screen(), w.screen.AtWorldMap(), v.Selected, len(v.Route), v.RouteShown, v.Position, w.home)
		}
		t.Logf("Enter with nothing selected started no travel and opened no mission over 12 ticks")
	})
	t.Run("before the first reveal tick", func(t *testing.T) {
		w := openCrossWitness(t)
		w.afterATick()
		w.clickScroll()
		w.steady("the scroll click", 0)
		w.key("enter")
		w.steady("Enter before the first reveal tick", 0)
		opened := w.idleUntil(nil)
		if want := max(revealTicks(w.route), w.frames); opened != want {
			t.Fatalf("after Enter before the first tick the mission opened on tick %d, want tick %d, as with no input", opened, want)
		}
		w.opened()
		t.Logf("Enter before the first reveal tick changed nothing: the mission opened on tick %d, when the %d-frame Cross animation passed its end", opened, w.frames)
	})
	for _, tc := range []struct {
		name string
		tick func(reveal int) int
	}{
		{"during the reveal", func(int) int { return 3 }},
		{"during the Cross animation", func(reveal int) int { return reveal + 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := startCrossWitness(t)
			reveal := revealTicks(w.route)
			at := tc.tick(reveal)
			if at+1 >= w.frames {
				t.Fatalf("setup: Enter on tick %d, mission opening on tick %d; want it before the Cross animation ends on tick %d", at, at+1, w.frames)
			}
			if got := w.idleUntil(func(tick int) bool { return tick == at }); got != at || w.app.Screen() != ui.ScreenTown {
				t.Fatalf("the mission had already opened on tick %d, before tick %d", got, at)
			}
			shown := min(at*worldMapRouteReveal, len(w.route))
			w.steady("the tick before Enter", shown)
			w.key("enter")
			w.steady("Enter", len(w.route))
			if got, want := w.screen.WorldMapView().Cross, w.screen.worldMap.assets.cross[1]; got != want {
				t.Fatalf("Enter drew Cross frame %v, want the frame of counter %d, %v", got, w.frames+1, want)
			}
			pressed := w.ticks()
			opened := w.idleUntil(nil)
			if opened != pressed+1 {
				t.Fatalf("Enter on tick %d: the mission opened on tick %d, want the next tick, tick %d (without Enter it opens on tick %d)",
					pressed, opened, pressed+1, max(reveal, w.frames))
			}
			w.opened()
			t.Logf("Enter on tick %d (%d coordinates drawn, reveal of %d ticks) hastened the arrival; the mission opened on tick %d instead of tick %d",
				pressed, shown, reveal, opened, max(reveal, w.frames))
		})
	}
}
