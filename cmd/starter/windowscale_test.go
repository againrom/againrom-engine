package main

import (
	"testing"

	"againrom/pkg/ui/widgetkit"
)

// At a display scale the starter lays out in device pixels.
func TestStarterWindowLaysOutInDevicePixels(t *testing.T) {
	old := widgetkit.MonitorDeviceScale
	t.Cleanup(func() { widgetkit.MonitorDeviceScale = old })
	for _, c := range []struct {
		s    float64
		w, h float64
	}{{1, 760, 760}, {1.25, 950, 950}, {1.5, 1140, 1140}, {2, 1520, 1520}} {
		widgetkit.MonitorDeviceScale = func() float64 { return c.s }
		w := &window{}
		gw, gh := w.LayoutF(winW, winH)
		if gw != c.w || gh != c.h || w.scale != c.s {
			t.Errorf("scale %v: LayoutF %v x %v, scale %v; want %v x %v", c.s, gw, gh, w.scale, c.w, c.h)
		}
	}
}
