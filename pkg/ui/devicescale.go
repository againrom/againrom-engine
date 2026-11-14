package ui

import (
	"image"
	"image/color"
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

// layoutUnit is the unit a surface lays its panels out in.
type layoutUnit uint8

const (
	// unitDevicePixels lays the surface out in the window's device pixels.
	unitDevicePixels layoutUnit = iota
	// unitPoints lays it out in whole points, presented at the device scale.
	unitPoints
)

// finalStage is how a surface's frame reaches the window.
type finalStage uint8

const (
	// stageWindow draws the frame on the window's own pixels.
	stageWindow finalStage = iota
	// stageScaledOnce draws the frame in points, then scales it to the device
	// pixels once through drawFinalFrame.
	stageScaledOnce
)

// pointerMap is how a device pixel of the pointer becomes a layout position.
type pointerMap uint8

const (
	// pointerDevice takes the device pixel as it is.
	pointerDevice pointerMap = iota
	// pointerToPoints takes the point the device pixel lies in.
	pointerToPoints
)

// displayRow is one surface's row: the unit its layout is in, its final
// stage, its pointer mapping and its scale filter. The presenter reads it.
type displayRow struct {
	unit    layoutUnit
	stage   finalStage
	pointer pointerMap
	// filter is the filter of a scaled final stage. Nil means the screen's
	// own policy (App.frameFilter) applies to its draws.
	filter func() frameFilter
	// ink fills the window under a scaled frame.
	ink color.Color
}

// gamePresentation is the front end and a lone viewer: device pixels, drawn
// on the window itself, with the screens' own filters.
var gamePresentation = displayRow{unit: unitDevicePixels, stage: stageWindow, pointer: pointerDevice}

// editorPresentation is the map editor: panels drawn in points and scaled
// once to the device pixels with the smooth filter the owner directed.
var editorPresentation = displayRow{
	unit:    unitPoints,
	stage:   stageScaledOnce,
	pointer: pointerToPoints,
	filter:  func() frameFilter { return filterOf(false, false) },
	ink:     editorInk,
}

// layout is the one LayoutF: it sets the surface's device scale through
// setScale, lays the surface out in its unit and answers the device size.
func (p displayRow) layout(outsideW, outsideH float64, setScale func(float64), lay func(w, h int) (int, int)) (float64, float64) {
	w := windowPixels(outsideW, outsideH, monitorDeviceScale())
	setScale(w.scale)
	if p.unit == unitPoints {
		lay(w.points.X, w.points.Y)
	} else {
		lay(w.device.X, w.device.Y)
	}
	return w.layoutF()
}

// scaled reports whether the frame is scaled to the device pixels at device
// scale s. At scale 1, or with no scale yet, it draws on the window.
func (p displayRow) scaled(s float64) bool {
	return p.stage == stageScaledOnce && s > 0 && s != 1
}

// toLayout is the layout position of the device pixel (x, y) at device scale s.
func (p displayRow) toLayout(x, y int, s float64) (int, int) {
	if p.pointer == pointerToPoints {
		return pointOfDevice(x, y, s)
	}
	return x, y
}

// ofPoints is n points in the row's layout pixels at device scale s.
func (p displayRow) ofPoints(n int, s float64) int {
	if p.unit == unitDevicePixels && s > 0 && !math.IsInf(s, 0) {
		return max(1, int(math.Round(float64(n)*s)))
	}
	return n
}

// present is the one final stage: draw paints the frame on the window when
// the row is not scaled at s, else on *frame, size layout pixels, which is
// scaled once to the device pixels through drawFinalFrame.
func (p displayRow) present(screen *ebiten.Image, s float64, size image.Point, frame, sharp **ebiten.Image, draw func(*ebiten.Image)) {
	if !p.scaled(s) {
		draw(screen)
		return
	}
	r := image.Rectangle{Max: size}
	if r.Empty() {
		return
	}
	if *frame == nil || (*frame).Bounds() != r {
		if *frame != nil {
			(*frame).Dispose()
		}
		*frame = ebiten.NewImage(r.Dx(), r.Dy())
	}
	draw(*frame)
	screen.Fill(p.ink)
	var op ebiten.DrawImageOptions
	op.GeoM.Scale(s, s)
	drawFinalFrame(screen, *frame, &op, sharp, p.filter())
}

// LayoutF lays the front end out in the window's device pixels; every screen
// then places its frame in them through its own placement.
func (a *App) LayoutF(outsideW, outsideH float64) (float64, float64) {
	return gamePresentation.layout(outsideW, outsideH, func(s float64) { a.deviceScale = s }, a.Layout)
}

// LayoutF lays a viewer run on its own out in the window's device pixels.
func (v *Viewer) LayoutF(outsideW, outsideH float64) (float64, float64) {
	return gamePresentation.layout(outsideW, outsideH, func(s float64) { v.deviceScale = s }, v.Layout)
}

// edgeMargin is the edge-scroll strip in the layout's pixels: EdgeMargin
// points times the device scale. It is the one reader of EdgeMargin.
func (v *Viewer) edgeMargin() int {
	return gamePresentation.ofPoints(EdgeMargin, v.deviceScale)
}

// inEdgeMargin reports a position in the strip of a window of size win.
func (v *Viewer) inEdgeMargin(x, y int, win image.Point) bool {
	m := v.edgeMargin()
	return x < m || y < m || x >= win.X-m || y >= win.Y-m
}

// LayoutF lays the map editor out in whole points, the unit its panels and
// font are drawn in, and keeps the device scale Draw presents them by.
func (e *MapEditor) LayoutF(outsideW, outsideH float64) (float64, float64) {
	return editorPresentation.layout(outsideW, outsideH, func(s float64) { e.deviceScale = s }, e.Layout)
}

func (l *frameLog) LayoutF(w, h float64) (float64, float64) { return l.app.LayoutF(w, h) }

// editorPointsSharpBuf is the method B intermediate of the editor's frame.
var editorPointsSharpBuf *ebiten.Image

// Draw draws the editor in points and presents it on the window's device
// pixels through the final-frame scaler, once. At a device scale of 1 the
// editor draws on the window itself.
func (e *MapEditor) Draw(screen *ebiten.Image) {
	editorPresentation.present(screen, e.deviceScale, e.window, &e.pointsFrame, &editorPointsSharpBuf, e.drawPoints)
}

// engineEditorInput is the engine's input reader. It is a variable so a test
// can feed a device pixel; production never replaces it.
var engineEditorInput = readEditorInput

// readNative is the editor's one input door: the test seam's input as given,
// or the engine's with the pointer mapped by the editor's row.
func (e *MapEditor) readNative() editorNativeInput {
	if e.nativeInput != nil {
		return e.nativeInput()
	}
	in := engineEditorInput()
	in.canvas.CursorX, in.canvas.CursorY = editorPresentation.toLayout(in.canvas.CursorX, in.canvas.CursorY, e.deviceScale)
	return in
}
