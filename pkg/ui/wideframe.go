package ui

import (
	"image"
	"image/color"
	"image/draw"

	"againrom/pkg/render/frame"
)

// wideFrameMode says how one native 640x480 town-family picture occupies a
// wider logical frame: a full-frame horizontal stretch, an intact centred
// modal, or a stretched town presentation beside its native-size right column.
type wideFrameMode uint8

const (
	wideFrameCentered wideFrameMode = iota
	wideFrameStretched
	wideFrameTownColumns
)

const (
	townNativeSeamX  = 464
	townNativeRightW = frame.W - townNativeSeamX
)

// wideFrameFill is visible interface backing for newly exposed pixels. It is
// deliberately opaque and non-black: a wide window owns these pixels as part
// of the logical interface instead of leaving the old vertical letterbox.
var wideFrameFill = color.RGBA{R: 0x0d, G: 0x0d, B: 0x14, A: 0xff}

type wideFrameSegment struct {
	Source image.Rectangle
	At     image.Point
	Size   image.Point
}

func (s wideFrameSegment) destination() image.Rectangle {
	size := s.Size
	if size.X <= 0 || size.Y <= 0 {
		size = s.Source.Size()
	}
	return image.Rectangle{Min: s.At, Max: s.At.Add(size)}
}

// wideFrameLayout is the single geometry authority shared by the CPU witness,
// the GPU blit and both pointer directions.
type wideFrameLayout struct {
	width int
	mode  wideFrameMode
}

func newWideFrameLayout(width int, mode wideFrameMode) wideFrameLayout {
	if width < frame.W {
		width = frame.W
	}
	return wideFrameLayout{width: width, mode: mode}
}

func (l wideFrameLayout) bounds() image.Rectangle {
	return image.Rect(0, 0, l.width, frame.H)
}

func (l wideFrameLayout) segments() []wideFrameSegment {
	if l.width == frame.W {
		return []wideFrameSegment{{Source: image.Rect(0, 0, frame.W, frame.H)}}
	}
	if l.mode == wideFrameStretched {
		return []wideFrameSegment{{
			Source: image.Rect(0, 0, frame.W, frame.H),
			Size:   image.Pt(l.width, frame.H),
		}}
	}
	if l.mode == wideFrameTownColumns {
		mainW := l.width - townNativeRightW
		return []wideFrameSegment{
			{
				Source: image.Rect(0, 0, townNativeSeamX, frame.H),
				Size:   image.Pt(mainW, frame.H),
			},
			{
				Source: image.Rect(townNativeSeamX, 0, frame.W, frame.H),
				At:     image.Pt(mainW, 0),
			},
		}
	}
	return []wideFrameSegment{{
		Source: image.Rect(0, 0, frame.W, frame.H),
		At:     image.Pt((l.width-frame.W)/2, 0),
	}}
}

// compose widens one native frame. The native 4:3 case is a byte-for-byte copy.
// Full-frame surfaces receive a nearest-neighbour horizontal stretch. Town
// rooms stretch their 464-pixel presentation up to the complete native-size
// 176-pixel seam+right-column unit anchored at the new right edge. No fill band
// or duplicate seam remains between them. Centred mode is kept for modals whose
// own geometry must remain intact.
func (l wideFrameLayout) compose(src image.Image) *image.RGBA {
	dst := image.NewRGBA(l.bounds())
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: wideFrameFill}, image.Point{}, draw.Src)
	if src == nil || src.Bounds().Dx() != frame.W || src.Bounds().Dy() != frame.H {
		return dst
	}
	for _, segment := range l.segments() {
		source := segment.Source.Add(src.Bounds().Min)
		destination := segment.destination()
		if destination.Size() == source.Size() {
			draw.Draw(dst, destination, src, source.Min, draw.Src)
			continue
		}
		drawNearest(dst, destination, src, source)
	}
	return dst
}

func drawNearest(dst *image.RGBA, destination image.Rectangle, src image.Image, source image.Rectangle) {
	if dst == nil || src == nil || destination.Empty() || source.Empty() {
		return
	}
	for y := destination.Min.Y; y < destination.Max.Y; y++ {
		sy := source.Min.Y + (y-destination.Min.Y)*source.Dy()/destination.Dy()
		for x := destination.Min.X; x < destination.Max.X; x++ {
			sx := source.Min.X + (x-destination.Min.X)*source.Dx()/destination.Dx()
			dst.Set(x, y, src.At(sx, sy))
		}
	}
}

// wideToNative maps one logical-frame point back to the native 640x480 hit
// geometry. Stretched pixels use the exact nearest-neighbour inverse; only the
// centred mode's outer extensions are inert.
func (l wideFrameLayout) wideToNative(p image.Point) (image.Point, bool) {
	if !p.In(l.bounds()) {
		return image.Point{}, false
	}
	if l.width == frame.W {
		return p, true
	}
	if l.mode == wideFrameStretched {
		return image.Pt(p.X*frame.W/l.width, p.Y), true
	}
	if l.mode == wideFrameTownColumns {
		mainW := l.width - townNativeRightW
		if p.X < mainW {
			return image.Pt(p.X*townNativeSeamX/mainW, p.Y), true
		}
		return image.Pt(townNativeSeamX+p.X-mainW, p.Y), true
	}
	offset := (l.width - frame.W) / 2
	if p.X < offset || p.X >= offset+frame.W {
		return image.Point{}, false
	}
	return image.Pt(p.X-offset, p.Y), true
}

// nativeToWide is wideToNative's output direction for headless pointer
// oracles. Stretched pixels choose the first exact inverse. The town seam and
// right-column controls remain native-size at the right edge.
func (l wideFrameLayout) nativeToWide(p image.Point) (image.Point, bool) {
	if !p.In(image.Rect(0, 0, frame.W, frame.H)) {
		return image.Point{}, false
	}
	if l.width == frame.W {
		return p, true
	}
	if l.mode == wideFrameStretched {
		// Pick the first expanded pixel whose inverse lands on p.X. The
		// ceiling makes nativeToWide followed by wideToNative exact.
		return image.Pt((p.X*l.width+frame.W-1)/frame.W, p.Y), true
	}
	if l.mode == wideFrameTownColumns {
		if p.X >= townNativeSeamX {
			return image.Pt(l.width-townNativeRightW+p.X-townNativeSeamX, p.Y), true
		}
		mainW := l.width - townNativeRightW
		return image.Pt((p.X*mainW+townNativeSeamX-1)/townNativeSeamX, p.Y), true
	}
	return image.Pt((l.width-frame.W)/2+p.X, p.Y), true
}

func townHasAnchoredRightColumn(t TownScreen) bool {
	if t == nil {
		return false
	}
	if world, ok := t.(TownWorldMapScreen); ok && world.AtWorldMap() {
		return false
	}
	if surface, ok := t.(TownSurfaceScreen); ok && surface.AtTownSurface() {
		return true
	}
	shop, ok := t.(TownShopScreen)
	return ok && shop.AtTownShop()
}

func (a *App) wideFrameWidth() int {
	if a != nil {
		if size := a.place.FrameSize(); size.X >= frame.W && size.Y == frame.H {
			return size.X
		}
	}
	return frame.W
}

// baseWideFrameLayout keeps the main menu and complete town family native
// 640x480.
func (a *App) baseWideFrameLayout() wideFrameLayout {
	return newWideFrameLayout(frame.W, wideFrameCentered)
}

func (a *App) inputWideFrameLayout() wideFrameLayout {
	return a.baseWideFrameLayout()
}

func (a *App) widenNativeFrame(src image.Image) *image.RGBA {
	return a.baseWideFrameLayout().compose(src)
}

// detachedTownDialogue reports the modal that must be composed after a wide
// town-column split. At native width the old one-pass 640x480 composition is
// retained exactly. Centred town-family screens also keep that ordinary path.
func (a *App) detachedTownDialogue() (*image.RGBA, bool) {
	if a == nil || a.flow == nil {
		return nil, false
	}
	layout := a.baseWideFrameLayout()
	if layout.width == frame.W || layout.mode != wideFrameTownColumns {
		return nil, false
	}
	dialogue, open := townDialogue(a.flow.town)
	return dialogue, open && dialogue != nil
}

func overlayDetachedTownDialogue(dst, dialogue *image.RGBA) {
	if dst == nil || dialogue == nil {
		return
	}
	origin := image.Pt((dst.Bounds().Dx()-dialogue.Bounds().Dx())/2,
		(frame.H-dialogue.Bounds().Dy())/2)
	draw.Draw(dst, dialogue.Bounds().Add(origin), dialogue, dialogue.Bounds().Min, draw.Over)
}

func (a *App) windowToNativeFrame(x, y int) (image.Point, bool) {
	if a == nil {
		return image.Point{}, false
	}
	wide, ok := a.place.WindowToFrame(x, y)
	if !ok {
		return image.Point{}, false
	}
	return a.inputWideFrameLayout().wideToNative(wide)
}

func (a *App) nativeFrameToWindow(p image.Point) (int, int, bool) {
	if a == nil {
		return 0, 0, false
	}
	wide, ok := a.inputWideFrameLayout().nativeToWide(p)
	if !ok {
		return 0, 0, false
	}
	return a.place.FrameToWindow(wide)
}
