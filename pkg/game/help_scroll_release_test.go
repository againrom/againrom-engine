package game

import (
	"testing"
)

// TestReleaseHelpScrollBarDragAndWheel drives the open help panel's scroll bar
// and wheel on the installed help text: the thumb drags the text down and back
// up, a held arrow repeats, the wheel scrolls, and each moves the drawn panel.
func TestReleaseHelpScrollBarDragAndWheel(t *testing.T) {
	f := releaseFront(t)
	witness := prepareHelpScrollbarStyleWitness(t, f)
	a := helpWitnessMap(t, f, true)
	v := f.live.view
	helpKeys(t, a, "f1")
	if !v.HelpOpen() {
		t.Fatal("F1 did not open the help panel")
	}
	lines, visible := v.HelpLines()
	if lines <= visible {
		t.Skipf("the installed help text fits (%d lines, %d visible)", lines, visible)
	}
	start := helpFrame(t, f)
	pointer := func(action string, x, y int) {
		t.Helper()
		if err := a.HeadlessPointer(action, x, y); err != nil {
			t.Fatal(err)
		}
	}
	point := func(part string) (int, int) {
		t.Helper()
		x, y, err := a.HeadlessHelpBarPoint(part)
		if err != nil {
			t.Fatal(err)
		}
		return x, y
	}
	scroll := func() int { first, _ := v.HelpScroll(); return first }

	tx, ty := point("thumb")
	pointer("press", tx, ty)
	ex, ey := point("track")
	pointer("move", ex, ey)
	if _, last := v.HelpScroll(); scroll() != last {
		t.Errorf("thumb dragged to the track end: scroll %d, want %d", scroll(), last)
	}
	if !helpDiffers(start, helpFrame(t, f), start.Bounds()) {
		t.Error("the dragged panel is drawn unchanged")
	}
	pointer("move", tx, ty)
	if scroll() != 0 {
		t.Errorf("thumb dragged back up: scroll %d, want 0", scroll())
	}
	pointer("release", tx, ty)

	dx, dy := point("down")
	pointer("press", dx, dy)
	if scroll() != 1 {
		t.Errorf("down arrow press: scroll %d, want 1", scroll())
	}
	for i := 0; i < 30; i++ {
		pointer("move", dx, dy)
	}
	if scroll() < 5 {
		t.Errorf("held down arrow after 30 ticks: scroll %d, want a steady repeat", scroll())
	}
	pointer("release", dx, dy)
	held := scroll()
	pointer("wheel-up", dx, dy)
	if scroll() >= held {
		t.Errorf("wheel up: scroll %d, want less than %d", scroll(), held)
	}
	pointer("wheel-down", dx, dy)
	pointer("wheel-down", dx, dy)
	if scroll() <= held {
		t.Errorf("wheel down: scroll %d, want more than %d", scroll(), held)
	}
	if !v.HelpOpen() {
		t.Error("scroll bar and wheel input closed the help panel")
	}
	witness.run(t, f, a)
}
