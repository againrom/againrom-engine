package game

import (
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/ui"
)

// textSettleFrames draws frames of a on a window-sized screen and answers
// the heap bytes and wall time one frame costs after a warm-up.
func textSettleFrames(t *testing.T, a *ui.App, step bool) (bytesPerFrame uint64, perFrame time.Duration) {
	t.Helper()
	a.Layout(2560, 1440)
	screen := ebiten.NewImage(2560, 1440)
	draw := func() {
		if step {
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		a.Draw(screen)
	}
	for range 5 {
		draw()
	}
	const frames = 60
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	for range frames {
		draw()
	}
	perFrame = time.Since(start) / frames
	runtime.ReadMemStats(&after)
	return (after.TotalAlloc - before.TotalAlloc) / frames, perFrame
}

// TestTextSettleDecidesRealFramesWithoutReadback draws a real town frame and
// a real mission frame with the character panel open, text smoothing on, and
// requires the pixel logs to decide every captured glyph (on a fresh mission
// the panel's glyphs stand on transparent cells, so the rule keeps none of
// them, exactly as a readback would): a frame they cannot
// decide reads the whole frame back from the GPU, and that per-frame readback
// is what stalled the game for seconds at a time. The per-frame heap bytes
// and time are logged for comparison.
func TestTextSettleDecidesRealFramesWithoutReadback(t *testing.T) {
	t.Run("town", func(t *testing.T) {
		f := releaseFront(t)
		f.Carried = f.NextParty()
		f.arriveInTown()
		f.Town.announceMission(f.Town.currentMain())
		now := time.Unix(100, 0)
		f.TownAnimationNow = func() time.Time { return now }
		f.TownAnimationRandom = func(int) int { return 0 }
		snap, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		payload, err := EncodeSave(snap, label)
		if err != nil {
			t.Fatal(err)
		}
		store := SaveStore{Dir: t.TempDir()}
		if _, err = store.Write(now, payload); err != nil {
			t.Fatal(err)
		}
		a := f.App("text-settle-town")
		a.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
		if err = a.HeadlessKey("load"); err != nil {
			t.Fatal(err)
		}
		if err = a.HeadlessActivate("@first"); err != nil {
			t.Fatal(err)
		}
		if a.Screen() != ui.ScreenTown {
			t.Fatalf("screen %v, want the town", a.Screen())
		}
		bytes, took := textSettleFrames(t, a, false)
		captured, kept, fallbacks := a.TextSettle()
		t.Logf("town: %d of %d glyphs smoothed, %d B and %v per frame", kept, captured, bytes, took)
		if kept == 0 || fallbacks != 0 {
			t.Fatalf("town: %d glyphs smoothed, %d readback frames; want some glyphs and no readback", kept, fallbacks)
		}
	})
	t.Run("mission", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		a := f.App("text-settle-mission")
		if err := a.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessKey("select-all"); err != nil {
			t.Fatal(err)
		}
		bytes, took := textSettleFrames(t, a, true)
		captured, kept, fallbacks := a.TextSettle()
		t.Logf("mission: %d of %d glyphs smoothed, %d B and %v per frame", kept, captured, bytes, took)
		if captured == 0 || fallbacks != 0 {
			t.Fatalf("mission: %d glyphs captured, %d readback frames; want captured glyphs and no readback", captured, fallbacks)
		}
	})
}
