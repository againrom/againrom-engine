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
	p := l.plan()
	dst := image.NewRGBA(p.bounds)
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: wideFrameFill}, image.Point{}, draw.Src)
	if src == nil || src.Bounds().Dx() != frame.W || src.Bounds().Dy() != frame.H {
		return dst
	}
	p.paint(dst, src, draw.Src)
	return dst
}

// wideFrameBlit is one planned copy of a native rectangle into the wide
// frame, scaled when the two sizes differ.
type wideFrameBlit struct {
	source, destination image.Rectangle
}

func (b wideFrameBlit) scaled() bool { return b.destination.Size() != b.source.Size() }

// wideFramePlan is one native-to-wide composite: the wide frame's bounds and
// the blits placing a native picture in it. wideFrameLayout.plan builds it;
// paint runs it on the CPU, blitNativeIntoWide on the GPU and
// logNativeIntoWide on the pixel log.
type wideFramePlan struct {
	bounds image.Rectangle
	blits  []wideFrameBlit
}

// plan is the layout's composite route.
func (l wideFrameLayout) plan() wideFramePlan {
	p := wideFramePlan{bounds: l.bounds()}
	for _, segment := range l.segments() {
		p.blits = append(p.blits, wideFrameBlit{source: segment.Source, destination: segment.destination()})
	}
	return p
}

// paint runs the plan on a CPU destination: an unscaled blit draws src with
// op, a scaled one is the nearest-neighbour stretch.
func (p wideFramePlan) paint(dst *image.RGBA, src image.Image, op draw.Op) {
	if dst == nil || src == nil {
		return
	}
	for _, b := range p.blits {
		source := b.source.Add(src.Bounds().Min)
		if !b.scaled() {
			draw.Draw(dst, b.destination, src, source.Min, op)
			continue
		}
		drawNearest(dst, b.destination, src, source)
	}
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

// detachedDialoguePlan is a town dialogue kept whole on the wide frame: the
// picture, the origin centring it and the clip it is drawn through.
// planDetachedDialogue builds it; paint runs it on the CPU and
// paintDetachedTownDialogue on the GPU.
type detachedDialoguePlan struct {
	pic  *image.RGBA
	at   image.Point
	clip image.Rectangle
}

// planDetachedDialogue centres dialogue on a wide frame of bounds, clipped
// by policy.
func planDetachedDialogue(bounds image.Rectangle, dialogue *image.RGBA, policy DialogueBackdrop) detachedDialoguePlan {
	size := dialogue.Bounds().Size()
	return detachedDialoguePlan{
		pic:  dialogue,
		at:   image.Pt((bounds.Dx()-size.X)/2, (frame.H-size.Y)/2),
		clip: dialoguePolicyClip(policy, bounds),
	}
}

// paint runs the plan on a CPU destination.
func (p detachedDialoguePlan) paint(dst *image.RGBA) {
	composeDialogueImageClipped(dst, p.pic, p.at, p.clip)
}

func overlayDetachedTownDialogue(dst, dialogue *image.RGBA) {
	if dst == nil || dialogue == nil {
		return
	}
	planDetachedDialogue(dst.Bounds(), dialogue, DialogueBackdrop{}).paint(dst)
}

func (a *App) windowToNativeFrame(x, y int) (image.Point, bool) {
	if a == nil {
		return image.Point{}, false
	}
	return a.flow.menuPlacement().transform.toFrame(a, x, y)
}

func (a *App) nativeFrameToWindow(p image.Point) (int, int, bool) {
	if a == nil {
		return 0, 0, false
	}
	return a.flow.menuPlacement().transform.toWindow(a, p)
}
