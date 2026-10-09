package ui

import (
	"image"
	"testing"
	"time"
)

// helpWindowPoint is the window position of panel point p, through the notice's
// frame placement and the viewer's window placement.
func helpWindowPoint(t *testing.T, v *Viewer, p image.Point) (int, int) {
	t.Helper()
	_, at, scale, ok := v.noticePresent()
	if !ok {
		t.Fatal("no help picture")
	}
	fp := image.Pt(at.X+int(float64(p.X)*scale), at.Y+int(float64(p.Y)*scale))
	x, y, ok := v.place.FrameToWindow(fp)
	if !ok {
		t.Fatalf("panel point %v is not on a window pixel", p)
	}
	return x, y
}

// helpScaledApp opens the help panel in a window of the given size, so the
// mission frame is placed at that window's scale. A zero size keeps scale 1.
func helpScaledApp(t *testing.T, w, h int) (*App, *Viewer, time.Time) {
	t.Helper()
	now := time.Unix(1_700_000_000, 0)
	a, _ := helpApp(t, 60)
	v := a.flow.viewer
	if w > 0 {
		v.Layout(w, h)
		a.winW, a.winH = w, h
	}
	a.step(appInput{Help: true}, now)
	if !v.HelpOpen() || v.noticeLayout().Scrollbar.Empty() {
		t.Fatal("help panel with a scroll bar did not open")
	}
	return a, v, now
}

var helpWindowSizes = []struct {
	name string
	w, h int
}{{"scale 1", 0, 0}, {"scale 1.25", 1280, 960}, {"scale 2", 2048, 1536}}

func helpDown(x, y int) appInput {
	return appInput{CursorX: x, CursorY: y, Viewer: Input{PrimaryDown: true}}
}

func helpPressAt(x, y int) appInput {
	in := helpDown(x, y)
	in.PrimaryPressed = true
	return in
}

// Dragging the thumb sets the position the bar's drag arm computes from the
// pointer row alone, (range-1)*(y-top-24)/(height-3*(width-4)) clamped, both
// ways and at every window scale (MENU-078, MENU-119).
func TestHelpThumbDragFollowsTheMouse(t *testing.T) {
	for _, sz := range helpWindowSizes {
		t.Run(sz.name, func(t *testing.T) {
			a, v, now := helpScaledApp(t, sz.w, sz.h)
			g := v.helpGeo()
			l := v.noticeLayout()
			_, _, track, thumb := helpBarParts(l.Scrollbar, 0, g.lines, g.visible)
			grab := image.Pt((thumb.Min.X+thumb.Max.X)/2, (thumb.Min.Y+thumb.Max.Y)/2)
			x, y := helpWindowPoint(t, v, grab)
			a.step(helpPressAt(x, y), now)
			if first, _ := v.HelpScroll(); first != 0 {
				t.Fatalf("pressing the thumb moved the text to %d", first)
			}
			bar := helpBar(l.Scrollbar, 0, g.lines, g.visible)
			for _, dy := range []int{track.Dy(), track.Dy() + 20, track.Dy() / 2, 0, -40} {
				x, y := helpWindowPoint(t, v, grab.Add(image.Pt(0, dy)))
				a.step(helpDown(x, y), now)
				p, _ := v.helpPanelPoint(x, y)
				if first, _ := v.HelpScroll(); first != bar.dragPos(p.Y) {
					t.Errorf("thumb dragged to panel row %d: scroll %d, want %d", p.Y, first, bar.dragPos(p.Y))
				}
			}
			if first, _ := v.HelpScroll(); first != 0 {
				t.Errorf("thumb dragged past the top: scroll %d, want 0", first)
			}
			if !v.HelpOpen() {
				t.Error("a thumb drag closed the panel")
			}
		})
	}
}

// Dragging down and then back up moves the text up (the owner's report).
func TestHelpThumbDragUpFromTheMiddle(t *testing.T) {
	a, v, now := helpScaledApp(t, 0, 0)
	g := v.helpGeo()
	l := v.noticeLayout()
	_, _, track, thumb := helpBarParts(l.Scrollbar, 0, g.lines, g.visible)
	span := track.Dy() - thumb.Dy()
	grab := image.Pt((thumb.Min.X+thumb.Max.X)/2, (thumb.Min.Y+thumb.Max.Y)/2)
	x, y := helpWindowPoint(t, v, grab)
	a.step(helpPressAt(x, y), now)
	x, y = helpWindowPoint(t, v, grab.Add(image.Pt(0, span)))
	a.step(helpDown(x, y), now)
	x, y = helpWindowPoint(t, v, grab.Add(image.Pt(0, span/4)))
	a.step(helpDown(x, y), now)
	first, _ := v.HelpScroll()
	if first <= 0 || first >= g.maxScroll()/2 {
		t.Errorf("thumb dragged back up to a quarter: scroll %d of %d", first, g.maxScroll())
	}
}

// Releasing ends the drag wherever the cursor is, and a later move without the
// button does not scroll.
func TestHelpThumbDragReleasesOutsideTheBar(t *testing.T) {
	a, v, now := helpScaledApp(t, 0, 0)
	g := v.helpGeo()
	l := v.noticeLayout()
	_, _, track, thumb := helpBarParts(l.Scrollbar, 0, g.lines, g.visible)
	grab := image.Pt((thumb.Min.X+thumb.Max.X)/2, (thumb.Min.Y+thumb.Max.Y)/2)
	x, y := helpWindowPoint(t, v, grab)
	a.step(helpPressAt(x, y), now)
	x, y = helpWindowPoint(t, v, image.Pt(l.Box.Min.X+100, grab.Y+(track.Dy()-thumb.Dy())/2))
	a.step(helpDown(x, y), now)
	mid, _ := v.HelpScroll()
	if mid == 0 {
		t.Fatal("dragging sideways off the bar did not keep scrolling")
	}
	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
	x, y = helpWindowPoint(t, v, image.Pt(l.Box.Min.X+100, track.Max.Y))
	a.step(appInput{CursorX: x, CursorY: y}, now)
	a.step(helpDown(x, y), now)
	if first, _ := v.HelpScroll(); first != mid {
		t.Errorf("text moved from %d to %d after the release", mid, first)
	}
	if !v.HelpOpen() {
		t.Error("releasing a drag outside the bar closed the panel")
	}
}

// An arrow or the track acts on the press tick itself, at every window scale.
// The first line down only resyncs the current line, -1, to the top
// (MENU-078); a page down moves visible rows minus one.
func TestHelpBarActsOnThePressTick(t *testing.T) {
	for _, sz := range helpWindowSizes {
		t.Run(sz.name, func(t *testing.T) {
			a, v, now := helpScaledApp(t, sz.w, sz.h)
			g := v.helpGeo()
			l := v.noticeLayout()
			up, down, track, _ := helpBarParts(l.Scrollbar, 0, g.lines, g.visible)
			centre := func(r image.Rectangle) (int, int) {
				return helpWindowPoint(t, v, r.Min.Add(r.Max).Div(2))
			}
			press := func(x, y int) {
				a.step(helpPressAt(x, y), now)
				a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
			}
			x, y := centre(down)
			press(x, y)
			if first, _ := v.HelpScroll(); first != 0 {
				t.Fatalf("first down arrow: scroll %d, want 0", first)
			}
			press(x, y)
			if first, _ := v.HelpScroll(); first != 1 {
				t.Fatalf("down arrow on its press tick: scroll %d, want 1", first)
			}
			x, y = centre(up)
			press(x, y)
			if first, _ := v.HelpScroll(); first != 0 {
				t.Fatalf("up arrow on its press tick: scroll %d, want 0", first)
			}
			x, y = centre(image.Rect(track.Min.X, track.Max.Y-3, track.Max.X, track.Max.Y))
			press(x, y)
			if first, _ := v.HelpScroll(); first != g.visible-helpPageDownLess {
				t.Fatalf("track below the thumb on its press tick: scroll %d, want %d", first, g.visible-helpPageDownLess)
			}
		})
	}
}

// A held arrow repeats after the held-button delay and then at the held-button
// interval; releasing stops it. The press tick's line down only resyncs the
// current line (MENU-078).
func TestHelpBarHeldArrowRepeats(t *testing.T) {
	a, v, now := helpScaledApp(t, 1280, 960)
	g := v.helpGeo()
	_, down, _, _ := helpBarParts(v.noticeLayout().Scrollbar, 0, g.lines, g.visible)
	x, y := helpWindowPoint(t, v, down.Min.Add(down.Max).Div(2))
	a.step(helpPressAt(x, y), now)
	scrolls := func() int { first, _ := v.HelpScroll(); return first }
	if scrolls() != 0 {
		t.Fatalf("press: scroll %d, want 0", scrolls())
	}
	for i := 1; i < chargenRepeatDelayTicks; i++ {
		a.step(helpDown(x, y), now)
		if scrolls() != 0 {
			t.Fatalf("repeat before the delay, held tick %d: scroll %d", i, scrolls())
		}
	}
	a.step(helpDown(x, y), now)
	if scrolls() != 1 {
		t.Fatalf("first repeat: scroll %d, want 1", scrolls())
	}
	for i := 1; i < chargenRepeatIntervalTicks; i++ {
		a.step(helpDown(x, y), now)
		if scrolls() != 1 {
			t.Fatalf("repeat inside the interval, tick %d: scroll %d", i, scrolls())
		}
	}
	a.step(helpDown(x, y), now)
	if scrolls() != 2 {
		t.Fatalf("second repeat: scroll %d, want 2", scrolls())
	}
	a.step(appInput{CursorX: x, CursorY: y, PrimaryReleased: true}, now)
	for i := 0; i < 3*chargenRepeatDelayTicks; i++ {
		a.step(appInput{CursorX: x, CursorY: y}, now)
	}
	if scrolls() != 2 {
		t.Errorf("scroll kept moving to %d after release", scrolls())
	}
}

// Holding the track repeats a page at a time and stops under the thumb.
func TestHelpBarHeldTrackRepeatsToTheThumb(t *testing.T) {
	a, v, now := helpScaledApp(t, 0, 0)
	g := v.helpGeo()
	_, _, track, _ := helpBarParts(v.noticeLayout().Scrollbar, 0, g.lines, g.visible)
	x, y := helpWindowPoint(t, v, image.Pt(track.Min.X+2, track.Max.Y-2))
	a.step(helpPressAt(x, y), now)
	if first, _ := v.HelpScroll(); first != g.visible-helpPageDownLess {
		t.Fatalf("track press: scroll %d, want %d", first, g.visible-helpPageDownLess)
	}
	for i := 0; i < 20*(chargenRepeatDelayTicks+chargenRepeatIntervalTicks); i++ {
		a.step(helpDown(x, y), now)
	}
	if first, last := v.HelpScroll(); first != last {
		t.Errorf("held track below the thumb ended at %d, want %d", first, last)
	}
}

// The wheel scrolls the open help text, away from the user is up, and clamps.
func TestHelpWheelScrolls(t *testing.T) {
	a, v, now := helpScaledApp(t, 0, 0)
	_, last := v.HelpScroll()
	a.step(appInput{WheelY: -1}, now)
	if first, _ := v.HelpScroll(); first != modWheelLines {
		t.Fatalf("wheel down: scroll %d, want %d", first, modWheelLines)
	}
	a.step(appInput{WheelY: 1}, now)
	a.step(appInput{WheelY: 1}, now)
	if first, _ := v.HelpScroll(); first != 0 {
		t.Errorf("wheel up past the top: scroll %d, want 0", first)
	}
	for i := 0; i < last; i++ {
		a.step(appInput{WheelY: -1}, now)
	}
	if first, _ := v.HelpScroll(); first != last {
		t.Errorf("wheel down past the end: scroll %d, want %d", first, last)
	}
	if !v.HelpOpen() {
		t.Error("the wheel closed the panel")
	}
}
