package game

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/ui"
)

// opaquePixels counts a frame's pixels with any alpha.
func opaquePixels(pic *image.RGBA) int {
	n := 0
	for i := 3; i < len(pic.Pix); i += 4 {
		if pic.Pix[i] != 0 {
			n++
		}
	}
	return n
}

// TestReleaseWorldMapTaskFlagAnimatesAndTheCrossHolds clicks mission 40's
// scroll on a chapter-40 city through App and lets App's cadence run to
// arrival. On every tick the task Flag1 is drawn at mission 40's anchor with
// the next frame of its 9-frame sheet (`TOWN-528`, `TOWN-529`), and the Cross
// plays its 13-frame sheet once and holds frame 12 (`TOWN-530`), the frame with
// the most drawn pixels (`TOWN-531`). Captures of several paints go to
// AGAINROM_SHOT_DIR when it is set.
func TestReleaseWorldMapTaskFlagAnimatesAndTheCrossHolds(t *testing.T) {
	w := openCrossWitness(t)
	a := w.front.worldMapAssets()
	if len(a.available) != 9 || len(a.cross) != 13 {
		t.Fatalf("installed sheets: Flag1 %d frames, Cross %d; want 9 and 13", len(a.available), len(a.cross))
	}
	last := len(a.cross) - 1
	full := opaquePixels(a.cross[last])
	for i, f := range a.cross[:last] {
		if n := opaquePixels(f); n >= full {
			t.Fatalf("Cross frame %d draws %d pixels, frame %d draws %d; want the held frame to draw the most", i, n, last, full)
		}
	}
	subset := 0
	for _, f := range a.cross[:last] {
		inside := true
		for i := 3; i < len(f.Pix) && i < len(a.cross[last].Pix); i += 4 {
			if f.Pix[i] != 0 && a.cross[last].Pix[i] == 0 {
				inside = false
				break
			}
		}
		if inside {
			subset++
		}
	}
	t.Logf("Cross frame %d draws %d pixels, more than every earlier frame; %d of %d earlier frames draw only inside it", last, full, subset, last)

	gates := w.screen.WorldMapView()
	w.clickScroll()
	w.point("hover", w.missX, w.missY)
	flagAt := w.anchor.Add(image.Pt(-4, -32))
	flagRect := image.Rect(0, 0, 32, 32).Add(flagAt)
	var previous *image.RGBA
	shots := map[int]bool{1: true, 4: true, 8: true, 12: true}
	check := func(tick int) {
		v := w.screen.WorldMapView()
		counter := w.screen.worldFlag1Frame
		if v.Available != image.Image(a.available[counter%9]) {
			t.Fatalf("tick %d: Flag1 frame is not frame %d of counter %d", tick, counter%9, counter)
		}
		if want := a.cross[min(tick, last)]; v.Cross != image.Image(want) {
			t.Fatalf("tick %d: Cross frame is not frame %d", tick, min(tick, last))
		}
		frame := ui.ComposeWorldMap(v)
		region := frame.SubImage(flagRect).(*image.RGBA)
		bare := v
		bare.AtHome = false
		hidden := ui.ComposeWorldMap(bare)
		if sameRegion(region, hidden.SubImage(flagRect).(*image.RGBA)) {
			t.Fatalf("tick %d: nothing drawn at Flag1's place over mission %d's anchor", tick, crossWitnessMission)
		}
		if previous != nil && sameRegion(region, previous) {
			t.Fatalf("tick %d: Flag1's place shows the same picture as the tick before", tick)
		}
		previous = region
		if shots[tick] {
			writeModShot(t, fmt.Sprintf("worldmap-task-tick-%02d", tick), frame)
		}
	}
	opened := w.idleUntil(func(tick int) bool { check(tick); return false })
	if opened != len(a.cross) {
		t.Fatalf("the mission opened on tick %d, want tick %d", opened, len(a.cross))
	}
	w.opened()
	t.Logf("mission %d: Flag1 drawn and changing on each of %d ticks at %v, Cross held at frame %d from tick %d; selected %d before the click",
		crossWitnessMission, opened-1, flagAt, last, last, gates.Selected)
}

func sameRegion(a, b *image.RGBA) bool {
	r := a.Bounds()
	if r != b.Bounds() {
		return false
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}
