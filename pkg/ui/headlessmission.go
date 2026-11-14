package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/catmullrom"
)

// headlessMapLayers composes the terrain and the art band under the current
// camera on the CPU, the way HeadlessMapFrame states.
func (v *Viewer) headlessMapLayers() (*image.RGBA, []HeadlessArtDraw, error) {
	r := &headlessTerrainTarget{v: v, pix: image.NewRGBA(image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH)), sources: map[*ebiten.Image]*image.RGBA{}}
	if v.Mode() == ModeDisplaced {
		v.drawDisplaced(r)
	} else {
		v.drawFlat(r)
	}
	if r.err != nil {
		return nil, nil, r.err
	}
	draws, err := v.HeadlessArtDraws()
	if err != nil {
		return nil, nil, err
	}
	for _, d := range draws {
		compositeHeadlessArt(r.pix, d)
	}
	return r.pix, draws, nil
}

// headlessOverlayRects fills the overlay passes and the shot markers the way
// drawMapMarks submits part of them: a pixel is covered where its centre is
// inside.
func (v *Viewer) headlessOverlayRects(dst *image.RGBA, part marksPart) {
	for _, pass := range v.overlayPasses() {
		if !part.holds(pass) {
			continue
		}
		for _, rect := range pass.Rects {
			if pass.HalfAdd {
				blendStatusBarRect(dst, rect, pass.Color)
				continue
			}
			fillHeadlessRect(dst, rect, pass.Color)
		}
	}
	if part == marksFrame {
		return
	}
	for _, r := range v.shotScreenRects() {
		fillHeadlessRect(dst, r, ShotMarkerColor)
	}
}

func fillHeadlessRect(dst *image.RGBA, r screenRect, c color.RGBA) {
	draw.Draw(dst, statusBarPixelRect(r, dst.Bounds()), image.NewUniform(c), image.Point{}, draw.Over)
}

// headlessMissionFrame composes the mission frame for a win-sized window on
// the CPU and names every scaling stage it passed. It is the CPU executor of
// missionLayers: it runs each step's cpu in list order, the map surface, the
// overlay rectangles, the numerals, the panel backings and the HUD pictures
// on the logical canvas, then the canvas placed in the window by the
// final-frame scaler's reference. A step with no cpu, and what a step's
// cpuSkips names, is not composed.
func (v *Viewer) headlessMissionFrame(win image.Point) (*image.RGBA, []string, error) {
	if v == nil || v.cam == nil || win.X <= 0 || win.Y <= 0 {
		return nil, nil, fmt.Errorf("headless mission frame: no placed viewer")
	}
	v.Layout(win.X, win.Y)
	if !v.place.Valid() {
		return nil, nil, fmt.Errorf("headless mission frame: no placement in %v", win)
	}
	v.syncMapViewport()
	v.refreshHoverLighting()
	f := &cpuPass{win: win, stages: []string{fmt.Sprintf("zoom %d%%", v.cam.ZoomPercent())}}
	if _, _, _, visible := v.noticePresent(); visible {
		f.stages = append(f.stages, "a notice is open and is not composed")
		if v.canvasRemapsMap() {
			f.stages = append(f.stages, "the map follows the canvas route; the dialogue backdrop is not composed")
		}
	}
	f.canvas = image.NewRGBA(image.Rect(0, 0, v.frameW, v.frameH))
	f.mapRect = image.Rect(0, 0, v.cam.ViewW, v.cam.ViewH)
	f.out = image.NewRGBA(image.Rect(0, 0, win.X, win.Y))
	draw.Draw(f.out, f.out.Rect, image.NewUniform(color.RGBA{A: 0xff}), image.Point{}, draw.Src)
	f.s = v.place.Scale()
	f.ox, f.oy = v.place.Origin()
	for _, l := range missionLayers {
		if l.cpu == nil {
			continue
		}
		if err := l.cpu(v, f); err != nil {
			return nil, nil, err
		}
	}
	return f.out, f.stages, nil
}

// cpuPass is what the CPU executor's steps pass on. comp is the zoom-1
// composition, nil when the map is drawn at the camera's zoom.
type cpuPass struct {
	win         image.Point
	canvas, out *image.RGBA
	mapRect     image.Rectangle
	s, ox, oy   float64
	stages      []string
	comp        *image.RGBA
	mapOp       ebiten.DrawImageOptions
	remapped    bool
	mapStage    string
}

func (f *cpuPass) over(pic image.Image, at image.Point, ok bool) {
	if ok {
		draw.Draw(f.canvas, pic.Bounds().Sub(pic.Bounds().Min).Add(at), pic, pic.Bounds().Min, draw.Over)
	}
}

func (v *Viewer) cpuMap(f *cpuPass) error {
	if unit, op, ok := v.smoothMapPlacement(); ok {
		var err error
		v.underCamera(unit, func() {
			if f.comp, _, err = v.headlessMapLayers(); err == nil {
				v.headlessOverlayRects(f.comp, marksMap)
			}
		})
		if err != nil {
			return err
		}
		f.mapOp, f.remapped = op, v.canvasRemapsMap()
		if f.remapped {
			v.cpuPlaceMap(f, f.canvas.SubImage(f.mapRect).(*image.RGBA), "canvas", 1, 0, 0)
		}
		return nil
	}
	layers, _, err := v.headlessMapLayers()
	if err != nil {
		return err
	}
	draw.Draw(f.canvas, f.mapRect, layers, image.Point{}, draw.Src)
	return nil
}

// cpuPlaceMap scales the zoom-1 composition onto target by the map's own
// placement times placement, shifted by x, y.
func (v *Viewer) cpuPlaceMap(f *cpuPass, target *image.RGBA, targetName string, placement, x, y float64) {
	sc := f.mapOp.GeoM.Element(0, 0) * placement
	filter := referenceFinalFrame(target, f.comp, sc, f.mapOp.GeoM.Element(0, 2)*placement+x, f.mapOp.GeoM.Element(1, 2)*placement+y, v.mapFilter())
	f.mapStage = fmt.Sprintf("map with status bars and marks %dx%d at zoom 1 -> %s: %s x%.4f", f.comp.Rect.Dx(), f.comp.Rect.Dy(), targetName, filter, sc)
}

func (v *Viewer) cpuMarks(f *cpuPass) error {
	if f.comp != nil {
		v.headlessOverlayRects(f.canvas.SubImage(f.mapRect).(*image.RGBA), marksFrame)
		return nil
	}
	v.headlessOverlayRects(f.canvas.SubImage(f.mapRect).(*image.RGBA), marksAll)
	if z := v.cam.DrawnZoom(); z != 1 {
		f.stages = append(f.stages, fmt.Sprintf("map sprites, tiles, status bars and marks -> canvas: nearest x%.4f each", z))
	}
	return nil
}

func (v *Viewer) cpuNumerals(f *cpuPass) error {
	for _, p := range v.numeralPlacements() {
		draw.Draw(f.canvas, p.Pic.Bounds().Add(p.At), p.Pic, p.Pic.Bounds().Min, draw.Over)
	}
	return nil
}

func (v *Viewer) cpuBackings(f *cpuPass) error {
	if v.cam.ViewH < v.frameH {
		draw.Draw(f.canvas, image.Rect(0, v.cam.ViewH, v.cam.ViewW, v.frameH), image.NewUniform(invFill), image.Point{}, draw.Src)
	}
	if v.cam.ViewW < v.frameW {
		draw.Draw(f.canvas, image.Rect(v.cam.ViewW, 0, v.frameW, v.frameH), image.NewUniform(missionColumnBackdrop), image.Point{}, draw.Src)
	}
	return nil
}

// cpuPicture places one HUD picture that its present function answers.
func (f *cpuPass) cpuPicture(present func() (*image.RGBA, image.Point, bool)) error {
	pic, at, ok := present()
	f.over(pic, at, ok && pic != nil)
	return nil
}

func (v *Viewer) cpuPanel(f *cpuPass) error        { return f.cpuPicture(v.panelPresent) }
func (v *Viewer) cpuSpellbook(f *cpuPass) error    { return f.cpuPicture(v.spellbookPresent) }
func (v *Viewer) cpuMinimap(f *cpuPass) error      { return f.cpuPicture(v.minimapPresent) }
func (v *Viewer) cpuPackBar(f *cpuPass) error      { return f.cpuPicture(v.packBarPresent) }
func (v *Viewer) cpuCommandPanel(f *cpuPass) error { return f.cpuPicture(v.commandPanelPresent) }
func (v *Viewer) cpuMessage(f *cpuPass) error      { return f.cpuPicture(v.messagePresent) }

func (v *Viewer) cpuColumnFiller(f *cpuPass) error {
	if pic, at, ok := v.columnFillerPresent(); ok && pic != nil {
		f.over(pic, at, true)
	}
	return nil
}

func (v *Viewer) cpuMapToWindow(f *cpuPass) error {
	if f.comp == nil {
		return nil
	}
	if !f.remapped {
		v.cpuPlaceMap(f, f.out, "window", f.s, f.ox, f.oy)
	}
	s, ox, oy, win := f.s, f.ox, f.oy, f.win
	frame := image.Rect(int(math.Ceil(ox-0.5)), int(math.Ceil(oy-0.5)),
		int(math.Ceil(ox+s*float64(v.frameW)-0.5)), int(math.Ceil(oy+s*float64(v.frameH)-0.5)))
	black := image.NewUniform(color.RGBA{A: 0xff})
	for _, strip := range []image.Rectangle{
		image.Rect(0, 0, frame.Min.X, win.Y), image.Rect(frame.Max.X, 0, win.X, win.Y),
		image.Rect(0, 0, win.X, frame.Min.Y), image.Rect(0, frame.Max.Y, win.X, win.Y),
	} {
		draw.Draw(f.out, strip, black, image.Point{}, draw.Src)
	}
	f.stages = append(f.stages, f.mapStage)
	return nil
}

func (v *Viewer) cpuCanvasToWindow(f *cpuPass) error {
	layer := image.NewRGBA(f.out.Rect)
	filter := referenceFinalFrame(layer, f.canvas, f.s, f.ox, f.oy, v.canvasFilter())
	draw.Draw(f.out, f.out.Rect, layer, image.Point{}, draw.Over)
	f.stages = append(f.stages, fmt.Sprintf("canvas %dx%d -> window %dx%d: %s x%.4f", v.frameW, v.frameH, f.win.X, f.win.Y, filter, v.place.Scale()))
	return nil
}

// headlessWindowFrame is the frame as the laid-out window shows it. The
// mission composes through headlessMissionFrame; every other screen is its
// native CPU frame placed by the final-frame scaler's reference, centred.
func (a *App) headlessWindowFrame() (*image.RGBA, string, error) {
	win := image.Pt(a.winW, a.winH)
	if win.X <= 0 || win.Y <= 0 {
		return nil, "", fmt.Errorf("headless frame: the scenario names no window to scale into")
	}
	menu := a.flow.menuPlacement().transform.placedPanel(a)
	if a.cutscene == nil && a.flow.mapShowing() && menu == nil {
		v := a.flow.viewer
		if v == nil {
			return nil, "", fmt.Errorf("headless frame: no mission viewer")
		}
		pix, stages, err := v.headlessMissionFrame(win)
		return pix, "stages: " + strings.Join(stages, "; "), err
	}
	native, note, err := a.headlessFrame(false)
	if err != nil {
		return nil, "", err
	}
	if !a.place.Valid() {
		return nil, "", fmt.Errorf("headless frame: no placement in %v", win)
	}
	s := a.place.Scale()
	out := image.NewRGBA(image.Rect(0, 0, win.X, win.Y))
	draw.Draw(out, out.Rect, image.NewUniform(color.RGBA{A: 0xff}), image.Point{}, draw.Src)
	ox := (float64(win.X) - s*float64(native.Rect.Dx())) / 2
	oy := (float64(win.Y) - s*float64(native.Rect.Dy())) / 2
	filter := referenceFinalFrame(out, native, s, ox, oy, a.frameFilter())
	stage := fmt.Sprintf("stages: native %dx%d -> window %dx%d: %s x%.4f", native.Rect.Dx(), native.Rect.Dy(), win.X, win.Y, filter, s)
	if note != "" {
		stage = note + "; " + stage
	}
	return out, stage, nil
}

// referenceFinalFrame is drawFinalFrame on the CPU: the exact copy at scale
// 1, otherwise the Catmull-Rom reference, the sharp bilinear reference, or
// for the pixel filter below a scale of 2 the nearest texel. It answers the
// name of the filter it ran.
func referenceFinalFrame(dst, src *image.RGBA, s, ox, oy float64, filter frameFilter) string {
	switch stageOf(filter, s).cpu {
	case scaleNone:
		return "none"
	case scaleNearest:
		referenceNearest(dst, src, s, ox, oy)
		switch {
		case s == 1:
			return "exact copy"
		case s == math.Trunc(s):
			return "nearest (whole factor)"
		}
		return "nearest"
	case scaleCatmull:
		catmullrom.Resample(dst, src, s, ox, oy)
		return "catmull-rom"
	}
	factor := max(int(s), 1)
	big := image.NewRGBA(image.Rect(0, 0, src.Rect.Dx()*factor, src.Rect.Dy()*factor))
	referenceNearest(big, src, float64(factor), 0, 0)
	referenceLinear(dst, big, s/float64(factor), ox, oy)
	return fmt.Sprintf("sharp bilinear (nearest x%d, then linear x%.4f)", factor, s/float64(factor))
}

// referenceNearest replaces the covered pixels of dst with src's nearest
// texel, sampling at pixel centres.
func referenceNearest(dst, src *image.RGBA, s, ox, oy float64) {
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	x0, x1 := catmullrom.Covered(sw, s, ox)
	y0, y1 := catmullrom.Covered(sh, s, oy)
	x0, x1 = max(x0, dst.Rect.Min.X), min(x1, dst.Rect.Max.X)
	y0, y1 = max(y0, dst.Rect.Min.Y), min(y1, dst.Rect.Max.Y)
	for y := y0; y < y1; y++ {
		sy := min(max(int(math.Floor((float64(y)+0.5-oy)/s)), 0), sh-1)
		for x := x0; x < x1; x++ {
			sx := min(max(int(math.Floor((float64(x)+0.5-ox)/s)), 0), sw-1)
			so, do := src.PixOffset(src.Rect.Min.X+sx, src.Rect.Min.Y+sy), dst.PixOffset(x, y)
			copy(dst.Pix[do:do+4], src.Pix[so:so+4])
		}
	}
}

// referenceLinear replaces the covered pixels of dst with the bilinear
// interpolation of src's four nearest texel centres, clamped at src's edge.
func referenceLinear(dst, src *image.RGBA, s, ox, oy float64) {
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	x0, x1 := catmullrom.Covered(sw, s, ox)
	y0, y1 := catmullrom.Covered(sh, s, oy)
	x0, x1 = max(x0, dst.Rect.Min.X), min(x1, dst.Rect.Max.X)
	y0, y1 = max(y0, dst.Rect.Min.Y), min(y1, dst.Rect.Max.Y)
	for y := y0; y < y1; y++ {
		ty := (float64(y)+0.5-oy)/s - 0.5
		iy := math.Floor(ty)
		fy := ty - iy
		ya, yb := min(max(int(iy), 0), sh-1), min(max(int(iy)+1, 0), sh-1)
		for x := x0; x < x1; x++ {
			tx := (float64(x)+0.5-ox)/s - 0.5
			ix := math.Floor(tx)
			fx := tx - ix
			xa, xb := min(max(int(ix), 0), sw-1), min(max(int(ix)+1, 0), sw-1)
			do := dst.PixOffset(x, y)
			for c := range 4 {
				at := func(px, py int) float64 {
					return float64(src.Pix[src.PixOffset(src.Rect.Min.X+px, src.Rect.Min.Y+py)+c])
				}
				top := at(xa, ya)*(1-fx) + at(xb, ya)*fx
				bot := at(xa, yb)*(1-fx) + at(xb, yb)*fx
				dst.Pix[do+c] = byte(math.Round(top*(1-fy) + bot*fy))
			}
		}
	}
}
