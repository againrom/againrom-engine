package ui

import (
	"math"
	"testing"
	"time"
)

func notchZoom(t *testing.T, v *Viewer, zoom float64, wheel float64) {
	t.Helper()
	v.Camera().SetZoom(zoom)
	v.syncMapViewport()
	v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: wheel}, time.Unix(1_700_000_000, 0))
}

func TestMissionWheelNotchMovesByFivePercentOfNativeScale(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start float64
		wheel float64
		want  float64
	}{
		{"in from native", 1, +1, 1.05},
		{"out from native", 1, -1, 0.95},
		{"in from off-grid 93 percent", 0.93, +1, 0.95},
		{"out from off-grid 93 percent", 0.93, -1, 0.90},
		{"in from 195 lands on 200", 1.95, +1, 2.00},
		{"in from 200 goes by 25", 2.00, +1, 2.25},
		{"out from 225 lands on 200", 2.25, -1, 2.00},
		{"out from 200 goes by 5", 2.00, -1, 1.95},
		{"in from 210 lands on 225", 2.10, +1, 2.25},
		{"out from 210 lands on 200", 2.10, -1, 2.00},
		{"in stops at ZoomMax", 8, +1, 8},
		{"out stops at the camera floor", 0.125, -1, 0.125},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := stepViewer(t)
			notchZoom(t, v, tc.start, tc.wheel)
			if got := v.Camera().Zoom; math.Abs(got-tc.want) > 1e-12 {
				t.Errorf("Zoom = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMissionWheelTwentyNotchesReturnToNativeOnTheGrid(t *testing.T) {
	v := stepViewer(t)
	v.Camera().SetZoom(1)
	v.syncMapViewport()
	at := time.Unix(1_700_000_000, 0)
	for i := 0; i < 20; i++ {
		v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: +1}, at)
		if p := v.Camera().ZoomPercent(); p%5 != 0 {
			t.Fatalf("notch %d in shows %d percent, off the 5 percent grid", i+1, p)
		}
	}
	for i := 0; i < 20; i++ {
		v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: -1}, at)
		if p := v.Camera().ZoomPercent(); p%5 != 0 {
			t.Fatalf("notch %d out shows %d percent, off the 5 percent grid", i+1, p)
		}
	}
	if got := v.Camera().Zoom; got != 1 {
		t.Errorf("Zoom after 20 in and 20 out = %v, want exactly 1", got)
	}
}

func TestMissionWheelCrossesTwoHundredAndComesBackOnTheGrid(t *testing.T) {
	v := stepViewer(t)
	v.Camera().SetZoom(1)
	v.syncMapViewport()
	at := time.Unix(1_700_000_000, 0)
	for i := 0; i < 27; i++ {
		v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: +1}, at)
	}
	if got := v.Camera().ZoomPercent(); got != 375 {
		t.Fatalf("after 27 notches in: %d percent, want 375", got)
	}
	for i := 0; i < 27; i++ {
		v.step(Input{CursorX: centre.x, CursorY: centre.y, WheelY: -1}, at)
	}
	if got := v.Camera().Zoom; got != 1 {
		t.Errorf("Zoom after 27 in and 27 out = %v, want exactly 1", got)
	}
}

func TestEditorWheelNotchStillMultipliesByTheCursorFactor(t *testing.T) {
	v := stepViewer(t)
	v.editorView = true
	v.wheelZoomPercent = 0
	v.Camera().SetZoom(1)
	v.syncMapViewport()
	v.step(Input{CursorX: 150, CursorY: 240, WheelY: +1}, time.Unix(1_700_000_000, 0))
	if got := v.Camera().Zoom; math.Abs(got-WheelZoomStep) > 1e-12 {
		t.Errorf("editor Zoom = %v, want %v", got, WheelZoomStep)
	}
}
