package ui

import (
	"math"
	"testing"

	"againrom/pkg/ui/widgetkit"
)

// The game and the map editor are two rows of one presentation: each LayoutF
// answers the device size of the window and sets its own device scale, and
// each row's unit, stage, pointer mapping and filter are the ones the
// presenter applies at every device scale.
func TestDisplayPresentationRowsReadOneLayout(t *testing.T) {
	if gamePresentation.unit != unitDevicePixels || gamePresentation.stage != stageWindow || gamePresentation.filter != nil {
		t.Fatalf("game row %+v: want device pixels on the window with the screen's own filter", gamePresentation)
	}
	if editorPresentation.unit != unitPoints || editorPresentation.stage != stageScaledOnce || editorPresentation.filter() != filterSmooth {
		t.Fatalf("editor row %+v: want points scaled once with the smooth filter", editorPresentation)
	}
	ow, oh := float64(pointsWindow.X), float64(pointsWindow.Y)
	for _, s := range deviceScales {
		atDeviceScale(t, s)
		want := widgetkit.WindowPixels(ow, oh, s)
		a := newTestApp(t, appRows(1), nil)
		if lw, lh := a.LayoutF(ow, oh); lw != float64(want.Device.X) || lh != float64(want.Device.Y) || a.deviceScale != s {
			t.Fatalf("game at scale %v: LayoutF %v,%v and device scale %v; want %v and %v", s, lw, lh, a.deviceScale, want.Device, s)
		}
		e := NewMapEditor(nil, nil, nil)
		if ew, eh := e.LayoutF(ow, oh); ew != float64(want.Device.X) || eh != float64(want.Device.Y) || e.deviceScale != s || e.window != want.Points {
			t.Fatalf("editor at scale %v: LayoutF %v,%v, window %v and device scale %v; want %v, points %v and %v", s, ew, eh, e.window, e.deviceScale, want.Device, want.Points, s)
		}
		px, py := widgetkit.PointOfDevice(7, 5, s)
		if got := editorPresentation.scaled(s); got != (s != 1) {
			t.Fatalf("editor scaled at %v: %v, want %v", s, got, s != 1)
		}
		if x, y := editorPresentation.toLayout(7, 5, s); x != px || y != py {
			t.Fatalf("editor pointer at scale %v: %d,%d, want %d,%d", s, x, y, px, py)
		}
		if x, y := gamePresentation.toLayout(7, 5, s); x != 7 || y != 5 {
			t.Fatalf("game pointer at scale %v: %d,%d, want the device pixel", s, x, y)
		}
		if g, ed := gamePresentation.ofPoints(EdgeMargin, s), editorPresentation.ofPoints(EdgeMargin, s); g != max(1, int(math.Round(EdgeMargin*s))) || ed != EdgeMargin {
			t.Fatalf("%d points at scale %v: game %d, editor %d", EdgeMargin, s, g, ed)
		}
	}
}
