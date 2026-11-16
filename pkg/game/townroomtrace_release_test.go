package game

import (
	"image"
	"time"
)

// The town room trace extends the square trace through each room page: the
// tavern, the shop, the school and the world map. Each room is entered from
// the square, its centre is watched long enough for every page clock to run,
// every page control is hovered and the harmless ones clicked, the page is
// paused and resumed, and the room is left. Beside the square trace's frame,
// sound, music, room and SAV records, each tick names the room entry and exit
// step hooks the town ran. The recorded lines live in testdata/townroomtrace.

func centre(r image.Rectangle) image.Point {
	return image.Pt((r.Min.X+r.Max.X)/2, (r.Min.Y+r.Max.Y)/2)
}

// pause drops the window focus for n ticks and takes it back.
func (tr *townTrace) pause(n int) {
	if err := tr.a.HeadlessFocus(false); err != nil {
		tr.t.Fatal(err)
	}
	tr.record("unfocus")
	for i := 0; i < n; i++ {
		*tr.now = tr.now.Add(97 * time.Millisecond)
		if err := tr.a.HeadlessStep(); err != nil {
			tr.t.Fatal(err)
		}
		tr.record("paused")
	}
	if err := tr.a.HeadlessFocus(true); err != nil {
		tr.t.Fatal(err)
	}
	tr.record("focus")
}

// shopPoint is the shop's own hit point for one control.
func (tr *townTrace) shopPoint(kind string, i int) image.Point {
	x, y, err := tr.a.HeadlessShopPoint(kind, i)
	if err != nil {
		tr.t.Fatal(err)
	}
	return image.Pt(x, y)
}
