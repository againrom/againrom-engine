package ui

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// monitorDeviceScale is the device scale of the monitor the window is on:
// device pixels per logical point, the factor the library divides the
// window's pixels by. Before the main loop, or with no monitor, it is 1. It
// is a variable so a test can feed a scale; production never replaces it.
var monitorDeviceScale = func() float64 {
	if m := ebiten.Monitor(); m != nil {
		if s := m.DeviceScaleFactor(); s > 0 {
			return s
		}
	}
	return 1
}

// windowSize is one window in the two units the library speaks: the device
// pixels it presents and the whole logical points it reports.
type windowSize struct {
	device, points image.Point
	scale          float64
}

// windowPixels is the one answer to the library's LayoutF. The device size
// is the outside size in points times the device scale, rounded as the
// library rounds the window it makes, so the frame the game draws is the
// window's own pixels and the library presents it as an exact copy. The
// points are the outside size truncated as the library truncates it for an
// integer Layout.
func windowPixels(outsideW, outsideH, scale float64) windowSize {
	if !(scale > 0) || math.IsInf(scale, 0) {
		scale = 1
	}
	return windowSize{
		device: image.Pt(max(1, int(math.Round(outsideW*scale))), max(1, int(math.Round(outsideH*scale)))),
		points: image.Pt(max(1, int(outsideW)), max(1, int(outsideH))),
		scale:  scale,
	}
}

// layoutF is the size LayoutF answers: the device pixels.
func (w windowSize) layoutF() (float64, float64) {
	return float64(w.device.X), float64(w.device.Y)
}

// pointOfDevice is the logical point a device pixel lies in, at the scale a
// points frame is presented by.
func pointOfDevice(x, y int, scale float64) (int, int) {
	if !(scale > 0) || scale == 1 {
		return x, y
	}
	return int(math.Floor(float64(x) / scale)), int(math.Floor(float64(y) / scale))
}

// LayoutF lays the front end out in the window's device pixels; every screen
// then places its frame in them through its own placement.
func (a *App) LayoutF(outsideW, outsideH float64) (float64, float64) {
	w := windowPixels(outsideW, outsideH, monitorDeviceScale())
	a.Layout(w.device.X, w.device.Y)
	return w.layoutF()
}

// LayoutF lays a viewer run on its own out in the window's device pixels.
func (v *Viewer) LayoutF(outsideW, outsideH float64) (float64, float64) {
	w := windowPixels(outsideW, outsideH, monitorDeviceScale())
	v.Layout(w.device.X, w.device.Y)
	return w.layoutF()
}

// LayoutF lays the map editor out in whole points, the unit its panels and
// font are drawn in, and keeps the device scale Draw presents them by.
func (e *MapEditor) LayoutF(outsideW, outsideH float64) (float64, float64) {
	w := windowPixels(outsideW, outsideH, monitorDeviceScale())
	e.deviceScale = w.scale
	e.Layout(w.points.X, w.points.Y)
	return w.layoutF()
}

func (l *frameLog) LayoutF(w, h float64) (float64, float64) { return l.app.LayoutF(w, h) }

// editorPointsSharpBuf is the method B intermediate of the editor's frame.
var editorPointsSharpBuf *ebiten.Image

// Draw draws the editor in points and presents it on the window's device
// pixels through the final-frame scaler, once. At a device scale of 1 the
// editor draws on the window itself.
func (e *MapEditor) Draw(screen *ebiten.Image) {
	s := e.deviceScale
	if !(s > 0) || s == 1 {
		e.drawPoints(screen)
		return
	}
	size := image.Rectangle{Max: e.window}
	if size.Empty() {
		return
	}
	if e.pointsFrame == nil || e.pointsFrame.Bounds() != size {
		if e.pointsFrame != nil {
			e.pointsFrame.Dispose()
		}
		e.pointsFrame = ebiten.NewImage(size.Dx(), size.Dy())
	}
	e.drawPoints(e.pointsFrame)
	screen.Fill(editorInk)
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(s, s)
	drawFinalFrame(screen, e.pointsFrame, &op, &editorPointsSharpBuf, e.frameFilter())
}

// frameFilter is the filter of the editor's points frame: the open map's
// FrameSmoothing switch, smooth when no map is open.
func (e *MapEditor) frameFilter() frameFilter {
	return filterOf(e.doc != nil && e.doc.Viewer != nil && e.doc.Viewer.frameSmoothingOff)
}

// engineEditorInput is the engine's input reader. It is a variable so a test
// can feed a device pixel; production never replaces it.
var engineEditorInput = readEditorInput

// readNative is the editor's one input door: the test seam's input as given,
// or the engine's with the pointer brought from device pixels to points.
func (e *MapEditor) readNative() editorNativeInput {
	if e.nativeInput != nil {
		return e.nativeInput()
	}
	in := engineEditorInput()
	in.canvas.CursorX, in.canvas.CursorY = pointOfDevice(in.canvas.CursorX, in.canvas.CursorY, e.deviceScale)
	return in
}
