package ui

import "github.com/hajimehoshi/ebiten/v2"

// drawSharpBilinear realizes op — a uniform Scale(s, s) then Translate(ox,
// oy), the one shape every caller here builds and the geometry tests
// observe; no rotation or shear, because the step below reads the scale off
// GeoM's own diagonal term — as owner decision method B ("sharp bilinear",
// engine/docs/HOTFIXES.md): a nearest-neighbour upscale of src to the
// largest integer factor not exceeding s, then one bilinear resize to op's
// exact placement. That fixes the uneven column/row pixel widths a single
// FilterNearest draw leaves at a non-integer scale (the owner's 1920- and
// 2560-wide mission windows, 1440/768 = 1.875x) without touching a single
// call site's own placement math: WindowToFrame/FrameToWindow and every
// caller's op are unchanged, so hit-testing is unaffected.
//
// AT AN EXACT INTEGER SCALE THE TWO-STEP PATH IS SKIPPED ENTIRELY: src is
// drawn with op itself under FilterNearest, the same single DrawImage call
// this project made before method B existed. That is what makes the town
// family's shipped 3.0x (2560x1440) — and every other whole-number scale,
// including the 1x an unscaled internal composite draws itself at — byte-
// identical to today BY CONSTRUCTION, rather than trusted to a GPU bilinear
// sample landing on the same value as a nearest one at 1:1.
//
// buf is the caller's own retained scratch canvas, reused across frames and
// resized only when the source size or the integer factor changes, so
// steady state costs two GPU blits and one Clear, and allocates nothing.
func drawSharpBilinear(dst, src *ebiten.Image, op *ebiten.DrawImageOptions, buf **ebiten.Image) {
	if dst == nil || src == nil || op == nil || buf == nil {
		return
	}
	scale := op.GeoM.Element(0, 0)
	if scale <= 0 {
		return
	}
	factor := int(scale)
	if factor < 1 {
		factor = 1
	}
	if scale == float64(factor) {
		nop := *op
		nop.Filter = ebiten.FilterNearest
		dst.DrawImage(src, &nop)
		return
	}

	ox, oy := op.GeoM.Element(0, 2), op.GeoM.Element(1, 2)
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	iw, ih := sw*factor, sh*factor
	if iw <= 0 || ih <= 0 {
		return
	}
	if *buf == nil || (*buf).Bounds().Dx() != iw || (*buf).Bounds().Dy() != ih {
		if *buf != nil {
			(*buf).Dispose()
		}
		*buf = ebiten.NewImage(iw, ih)
	} else {
		(*buf).Clear()
	}

	var nop ebiten.DrawImageOptions
	nop.Filter = ebiten.FilterNearest
	nop.GeoM.Scale(float64(factor), float64(factor))
	(*buf).DrawImage(src, &nop)

	var lop ebiten.DrawImageOptions
	lop.Filter = ebiten.FilterLinear
	lop.GeoM.Scale(scale/float64(factor), scale/float64(factor))
	lop.GeoM.Translate(ox, oy)
	dst.DrawImage(*buf, &lop)
}

// The final blits sharp-bilinear method B replaces each keep their own
// retained intermediate canvas: reusing one buffer across differently-sized
// draws (the town composite, the cursor, a paused menu over the map, the
// mission frame, the mission pointer) would thrash-reallocate it every site
// every frame instead of ever settling at a steady size.
var (
	townCompositeSharpBuf *ebiten.Image
	cursorSharpBuf        *ebiten.Image
	cutsceneSharpBuf      *ebiten.Image
	menuOverMapSharpBuf   *ebiten.Image
	missionFrameSharpBuf  *ebiten.Image
	pointerSharpBuf       *ebiten.Image
)
