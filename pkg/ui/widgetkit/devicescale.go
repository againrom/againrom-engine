package widgetkit

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// MonitorDeviceScale is the monitor's device pixels per logical point; 1
// before the main loop. A variable so a test can feed a scale.
var MonitorDeviceScale = func() float64 {
	if m := ebiten.Monitor(); m != nil {
		if s := m.DeviceScaleFactor(); s > 0 {
			return s
		}
	}
	return 1
}

// WindowSize is one window in the two units the library speaks: the device
// pixels it presents and the whole logical points it reports.
type WindowSize struct {
	Device, Points image.Point
	Scale          float64
}

// WindowPixels is the answer to the library's LayoutF: the outside size times
// the device scale, rounded as the library rounds its window, and the
// outside size truncated to whole points.
func WindowPixels(outsideW, outsideH, scale float64) WindowSize {
	if !(scale > 0) || math.IsInf(scale, 0) {
		scale = 1
	}
	return WindowSize{
		Device: image.Pt(max(1, int(math.Round(outsideW*scale))), max(1, int(math.Round(outsideH*scale)))),
		Points: image.Pt(max(1, int(outsideW)), max(1, int(outsideH))),
		Scale:  scale,
	}
}

// LayoutF is the size LayoutF answers: the device pixels.
func (w WindowSize) LayoutF() (float64, float64) {
	return float64(w.Device.X), float64(w.Device.Y)
}

// PointOfDevice is the logical point a device pixel lies in, at the scale a
// points frame is presented by.
func PointOfDevice(x, y int, scale float64) (int, int) {
	if !(scale > 0) || scale == 1 {
		return x, y
	}
	return int(math.Floor(float64(x) / scale)), int(math.Floor(float64(y) / scale))
}
