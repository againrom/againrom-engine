package ui

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"againrom/pkg/render/camera"
)

// smoothMapApron is the margin in world pixels the zoom-1 composition keeps
// on every side of the view, so the Catmull-Rom taps at the view's edge read
// map content rather than the composition's clamped edge.
const smoothMapApron = 2

// smoothMapMaxSide bounds each side of the zoom-1 composition in pixels. A
// zoom far enough out to need more draws the map as with Smoothing off.
const smoothMapMaxSide = 4096

// smoothMapPlacement is the one rule of the smooth zoomed map (DIV-4876).
// With the game's Smoothing option on and a drawn zoom other than exactly 1,
// it answers the camera the map's world layers are composed under (zoom 1,
// its origin the view's floored to a whole world pixel less the apron, its
// size the view's at zoom 1 plus the apron) and the blit that places that
// composition on the map surface: Scale by the zoom, then Translate by the
// origin's remainder at the zoom. ok is false where the map draws as before.
func (v *Viewer) smoothMapPlacement() (unit camera.Camera, op ebiten.DrawImageOptions, ok bool) {
	c := v.cam
	if c == nil || (v.editorView && !v.graphics.Smoothing) || c.ViewW <= 0 || c.ViewH <= 0 {
		return unit, op, false
	}
	z := c.DrawnZoom()
	if z == 1 {
		return unit, op, false
	}
	w := int(math.Ceil(float64(c.ViewW)/z)) + 2*smoothMapApron + 1
	h := int(math.Ceil(float64(c.ViewH)/z)) + 2*smoothMapApron + 1
	if w > smoothMapMaxSide || h > smoothMapMaxSide {
		return unit, op, false
	}
	unit = *c
	unit.Zoom = 1
	unit.X, unit.Y = math.Floor(c.X)-smoothMapApron, math.Floor(c.Y)-smoothMapApron
	unit.ViewW, unit.ViewH = w, h
	op.GeoM.Scale(z, z)
	op.GeoM.Translate((unit.X-c.X)*z, (unit.Y-c.Y)*z)
	return unit, op, true
}

// drawMap draws the map's world layers onto dst: at the camera's zoom, or,
// where smoothMapPlacement answers, once at zoom 1 into the retained
// composition, which drawFinalFrame then scales onto dst.
func (v *Viewer) drawMap(dst *ebiten.Image) {
	v.marksComposed = false
	if unit, op, ok := v.smoothMapPlacement(); ok {
		v.drawSmoothMap(dst, unit, op)
		return
	}
	v.drawMapLayers(dst)
}

// drawSmoothMap is drawMap's smooth branch. op is its own copy, so only this
// branch moves it to the heap for blitSmoothMap.
func (v *Viewer) drawSmoothMap(dst *ebiten.Image, unit camera.Camera, op ebiten.DrawImageOptions) {
	if v.smoothMap == nil || v.smoothMap.Bounds().Dx() != unit.ViewW || v.smoothMap.Bounds().Dy() != unit.ViewH {
		if v.smoothMap != nil {
			v.smoothMap.Dispose()
		}
		v.smoothMap = ebiten.NewImage(unit.ViewW, unit.ViewH)
	} else {
		v.smoothMap.Clear()
	}
	v.underCamera(unit, func() {
		v.drawMapLayers(v.smoothMap)
		if !v.editorView {
			v.drawMapMarks(v.smoothMap, marksMap)
		}
	})
	v.marksComposed = !v.editorView
	if v.windowMap.deferred {
		v.windowMap.op, v.windowMap.composed = op, true
		return
	}
	blitSmoothMap(dst, v.smoothMap, &op, v.mapFilter())
}

// windowMap carries the zoom-1 composition from drawMap to Viewer.Draw.
// deferred asks drawMap to leave the map surface of the canvas empty and
// answer the composition's placement on that surface in op; composed says it
// did.
type windowMap struct {
	deferred, composed bool
	op                 ebiten.DrawImageOptions
}

// mapFilter is the filter of the zoom-1 composition: the pixel filter with
// the game's Smoothing option off, the Catmull-Rom kernel otherwise. The map
// editor has no Smoothing option.
func (v *Viewer) mapFilter() frameFilter {
	if !v.graphics.Smoothing && !v.editorView {
		return filterPixel
	}
	return filterSmooth
}

// canvasFilter is the filter of the frame canvas and the pointer: the map's
// pixel filter with Smoothing off, the starter's switch otherwise.
func (v *Viewer) canvasFilter() frameFilter {
	if f := v.mapFilter(); f == filterPixel {
		return f
	}
	return filterOf(v.frameSmoothingOff)
}

// playerZoom is the zoom the player set, also while the map is composed
// under the zoom-1 camera.
func (v *Viewer) playerZoom() float64 {
	if v.composedFrom != 0 {
		return v.composedFrom
	}
	return v.cam.Zoom
}

// canvasRemapsMap reports a frame whose dialogue backdrop or frame shadows
// rewrite the canvas pixels under them. The map must then be on the canvas,
// so Draw does not place it on the window apart.
func (v *Viewer) canvasRemapsMap() bool {
	if _, shows, _ := v.DialogueBackdropPlan(); shows > 0 {
		return true
	}
	return v.noticeOpen && v.font != nil && v.noticeLayout().styleRow().frameShadows
}

// blitMapToWindow scales the zoom-1 composition onto the window once, by the
// zoom times the window scale, under the canvas that Draw places after it.
// op is the composition's placement on the canvas's map surface. The apron
// that lands outside the placed frame is painted over in black.
func (v *Viewer) blitMapToWindow(screen *ebiten.Image, op ebiten.DrawImageOptions) {
	s := v.place.Scale()
	ox, oy := v.place.Origin()
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(ox, oy)
	blitSmoothMap(screen, v.smoothMap, &op, v.mapFilter())
	b := screen.Bounds()
	w, h := float32(b.Dx()), float32(b.Dy())
	right, bottom := float32(ox+s*float64(v.frameW)), float32(oy+s*float64(v.frameH))
	black := color.RGBA{A: 0xff}
	if ox > 0 {
		vector.DrawFilledRect(screen, 0, 0, float32(ox), h, black, false)
	}
	if oy > 0 {
		vector.DrawFilledRect(screen, 0, 0, w, float32(oy), black, false)
	}
	if right < w {
		vector.DrawFilledRect(screen, right, 0, w-right, h, black, false)
	}
	if bottom < h {
		vector.DrawFilledRect(screen, 0, bottom, w, h-bottom, black, false)
	}
}

// pinnedHover is a hover decision held for the duration of underCamera.
type pinnedHover struct {
	ref InspectionSubject
	ok  bool
	set bool
}

// underCamera runs draw with the viewer's camera set to c and restores the
// camera afterwards, also when draw panics. The pointer is in screen pixels of
// the real camera, so the hover is decided under it first and draw reads that.
func (v *Viewer) underCamera(c camera.Camera, draw func()) {
	saved := *v.cam
	ref, ok := v.hoverInspection()
	v.pinnedHover = pinnedHover{ref: ref, ok: ok, set: true}
	v.composedFrom = saved.Zoom
	defer func() { *v.cam = saved; v.pinnedHover = pinnedHover{}; v.composedFrom = 0 }()
	*v.cam = c
	draw()
}

// drawMapLayers is the map's world: terrain, then the art band (shadows,
// structures, units, sacks, projectiles, spell art), then, on the mission
// screen, the Heal/Drain shower and the fog. The map editor draws the first
// two only.
func (v *Viewer) drawMapLayers(dst *ebiten.Image) {
	if v.Mode() == ModeDisplaced {
		v.drawDisplaced(dst)
	} else {
		v.drawFlat(dst)
	}
	// drawArt owns the complete retained-overlay/shadow/projectile/body order
	// (ANIM-047). Its existing static-object depth merge stays inside the body
	// pass; all of this content remains below diagnostic glyphs and shroud.
	v.drawArt(dst)
	if v.editorView {
		return
	}
	// The separate Heal/Drain shower retains its existing pass.
	v.drawHealArt(dst)
	v.drawShroud(dst)
}

// blitSmoothMap places the zoom-1 composition on the map surface through the
// final-frame scaler's Catmull-Rom kernel; the builder's own fallback runs
// when the shader is unavailable. It is a variable so a test can record the
// placement drawMap requests; production never replaces it.
var blitSmoothMap = func(dst, src *ebiten.Image, op *ebiten.DrawImageOptions, filter frameFilter) {
	drawFinalFrame(dst, src, op, &smoothMapSharpBuf, filter)
}
