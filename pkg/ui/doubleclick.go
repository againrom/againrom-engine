package ui

import (
	"image"
	"time"

	"againrom/pkg/ui/systemclick"
)

// The one double-click detector. The original's frame window class asks the
// system for double clicks: a second left press inside the system's
// double-click time and rectangle arrives as the double-click message, after a
// release, with no game-set value; the press after it starts a new pair
// (MENU-143). The detector classifies every primary press the App sees that
// way, on every screen. What the second press of a pair does is each screen's
// own: the Load list loads its stored selection (MENU-144), the shop backpack
// uses a carried item (MENU-145), the tavern roster and the statistic panel run
// their press again at the second press's point (MENU-146).

// doubleClick pairs primary presses in window pixels. The setting is read at
// the first press of each pair and holds for its second, so a changed system
// setting reaches the next pair. A nil read uses systemclick.Fallback: the
// windowed game installs systemclick.Read, and every headless and test App
// keeps the fixed fallback.
type doubleClick struct {
	read func() systemclick.Setting

	// armed: a single press waits for its second. released: the button came
	// up after it. down: the button is held.
	armed, released, down bool
	at                    time.Time
	point                 image.Point
	pair                  systemclick.Setting
}

// observe classifies this tick's primary edges at the cursor and reports
// whether its press is the second press of a double click. A secondary press
// ends the pair. When one tick carries both edges, a held button's release
// comes first.
func (d *doubleClick) observe(in appInput, now time.Time) bool {
	if in.SecondaryPressed {
		d.armed = false
	}
	releaseFirst := in.PrimaryReleased && in.PrimaryPressed && d.down
	if releaseFirst {
		d.release()
	}
	double := false
	if in.PrimaryPressed {
		double = d.press(image.Pt(in.CursorX, in.CursorY), now)
	}
	if in.PrimaryReleased && !releaseFirst && d.down {
		d.release()
	}
	return double
}

// press records one primary press at p and now and reports whether it
// completes a double click: an armed press, released since, with p and now
// inside the pair's time and rectangle.
func (d *doubleClick) press(p image.Point, now time.Time) bool {
	d.down = true
	if d.armed && d.released && d.inside(p, now) {
		d.armed = false
		return true
	}
	d.pair = systemclick.Fallback
	if d.read != nil {
		d.pair = d.read()
	}
	d.armed, d.released, d.at, d.point = true, false, now, p
	return false
}

func (d *doubleClick) release() {
	d.down = false
	if d.armed {
		d.released = true
	}
}

// inside reports whether a press at p and now lies within the pair's time and
// in its rectangle centred on the first press.
func (d *doubleClick) inside(p image.Point, now time.Time) bool {
	elapsed := now.Sub(d.at)
	if elapsed < 0 || elapsed > d.pair.Time {
		return false
	}
	dx, dy := p.X-d.point.X, p.Y-d.point.Y
	return 2*absInt(dx) <= d.pair.Width && 2*absInt(dy) <= d.pair.Height
}
