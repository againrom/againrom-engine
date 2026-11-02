package ui

import (
	"image"
	"testing"
	"time"
)

func TestChromeHitKeepAnswersAsBuiltAndEndsWithTheSample(t *testing.T) {
	e := newMapEditorFixture(t)
	e.Layout(640, 480)
	e.showRecords(0)
	e.ChromeImage()
	now := time.Unix(0, 0)
	for _, p := range []image.Point{{5, 5}, {320, 240}, {630, 470}, {100, 300}, {500, 100}} {
		want := e.chromeHitBuilt(p)
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, now)
		if e.hitKeep.on || e.hitKeep.held {
			t.Fatalf("the keep outlived the sample at %v", p)
		}
		if got := e.chromeHit(p); got != want {
			t.Fatalf("hit at %v = %v, built %v", p, got, want)
		}
	}
	// A sample with a press keeps nothing.
	e.hitKeep = chromeHitKeep{}
	e.chromePointer(Input{CursorX: 5, CursorY: 5, PrimaryDown: true}, now)
	if e.hitKeep.on || e.hitKeep.held {
		t.Fatal("a pressed sample left a kept hit")
	}
}
