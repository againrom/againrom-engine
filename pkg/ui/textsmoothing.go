package ui

import (
	"image"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

// SetTextSmoothing turns owner decision method C's presentation-layer text
// overlay (DIV-1385) on or off. It defaults on (NewApp). Off reproduces
// exactly today's renderer: Draw never opens a capture window, so nothing
// about the composed frame changes.
func (a *App) SetTextSmoothing(enabled bool) {
	if a == nil {
		return
	}
	a.textSmoothingEnabled = enabled
	if v := a.flow.viewer; v != nil {
		v.SetTextSmoothing(enabled)
	}
}

// TextSettle reports how many glyphs the last drawn frame captured and how
// many of them it smoothed, and how many frames, map screen included, had to
// read the frame back because the pixel logs could not decide.
func (a *App) TextSettle() (captured, kept, fallbacks int) {
	if a == nil {
		return 0, 0, 0
	}
	captured, kept, fallbacks = a.textCaptured, a.textKept, a.textSettleFallbacks
	if v := a.flow.viewer; v != nil {
		fallbacks += v.textSettleFallbacks
		if a.flow.mapShowing() {
			captured, kept = v.textCaptured+a.menuCaptured, v.textKept+a.menuKept
		}
	}
	return captured, kept, fallbacks
}

// TextSmoothing reports the current switch.
func (a *App) TextSmoothing() bool { return a != nil && a.textSmoothingEnabled }

// textLayer is the glyphs one settled layer of a frame captured, with the
// bounds and pixel log that decide which of them show.
type textLayer struct {
	calls  []text.DrawCall
	bounds image.Rectangle
	log    *pixelLog
}

// TextFate is one captured glyph and what the overlay settled for it.
type TextFate struct {
	Call text.DrawCall
	Fate textsmooth.Fate
}

// TextFates answers every glyph the last drawn frame captured, layer by layer
// and in call order within a layer, with the fate the overlay settled for it.
// It reads the pixel log Draw decided from, so a witness sees the decision the
// window frame was drawn with; drawing never calls it.
func (a *App) TextFates() []TextFate {
	if a == nil {
		return nil
	}
	var out []TextFate
	for _, l := range a.textLayers {
		outcomes, _ := textsmooth.Explain(l.calls, l.bounds, l.log.oracle())
		for i, o := range outcomes {
			call := l.calls[i]
			if o.Fate == textsmooth.Kept {
				call = o.Call
			}
			out = append(out, TextFate{call, o.Fate})
		}
	}
	return out
}

// textOverlay composites the kept glyphs, captured at native resolution in
// the frame's own coordinates, onto the window at window resolution with the
// SAME scale and origin the art was upscaled and placed with. It keeps a
// window-sized buffer and texture and recomposites and re-uploads only when
// the glyphs or the placement change, and then only the rectangle that
// changed; every other frame it draws the texture it already holds.
type textOverlay struct {
	buf     *image.RGBA
	tex     *ebiten.Image
	cache   textsmooth.Cache
	calls   []text.DrawCall
	scale   float64
	origin  [2]float64
	dirty   image.Rectangle
	valid   bool
	scratch []byte
}

func (o *textOverlay) draw(screen *ebiten.Image, calls []text.DrawCall, scale, originX, originY float64) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return
	}
	if full := image.Rect(0, 0, w, h); o.buf == nil || o.buf.Rect != full {
		o.buf = image.NewRGBA(full)
		if o.tex != nil {
			o.tex.Dispose()
		}
		o.tex = ebiten.NewImage(w, h)
		o.dirty, o.valid = image.Rectangle{}, false
	}
	if !o.valid || o.scale != scale || o.origin != [2]float64{originX, originY} || !sameCalls(o.calls, calls, false) {
		clearRect(o.buf, o.dirty)
		now := o.cache.Composite(o.buf, calls, scale, originX, originY)
		uploadRect(o.tex, o.buf, o.dirty.Union(now), &o.scratch)
		o.dirty, o.valid = now, true
		o.calls = copyCalls(o.calls[:0], calls, false)
		o.scale, o.origin = scale, [2]float64{originX, originY}
	}
	drawRect(screen, o.tex, o.dirty)
}

// textEraser restores, over the composed frame, every cell a kept glyph
// painted from what that glyph replaced — textsmooth.Erase's own result —
// through a transparent frame-sized texture drawn over the frame. Each
// restored cell is opaque, so the draw copies it exactly. The texture is
// re-uploaded only where the kept glyphs changed.
type textEraser struct {
	buf     *image.RGBA
	tex     *ebiten.Image
	kept    []text.DrawCall
	dirty   image.Rectangle
	scratch []byte
}

func (e *textEraser) apply(frame *ebiten.Image, kept []text.DrawCall) {
	b := frame.Bounds()
	if e.buf == nil || e.buf.Rect != b {
		e.buf = image.NewRGBA(b)
		if e.tex != nil {
			e.tex.Dispose()
		}
		e.tex = ebiten.NewImage(b.Dx(), b.Dy())
		e.kept, e.dirty = nil, image.Rectangle{}
	}
	if !sameCalls(e.kept, kept, true) {
		clearRect(e.buf, e.dirty)
		textsmooth.Erase(e.buf, kept)
		now := glyphBounds(kept).Intersect(b)
		uploadRect(e.tex, e.buf, e.dirty.Union(now), &e.scratch)
		e.dirty = now
		e.kept = copyCalls(e.kept[:0], kept, true)
	}
	drawRect(frame, e.tex, e.dirty)
}

// settleText is settleTextFrame without the readback wherever log can decide:
// it keeps the captured glyphs still visible in frame, erases their raster
// from frame itself and answers frame. A glyph the log cannot decide sends
// the frame to settleTextFrame, counted in fallbacks.
func settleText(frame *ebiten.Image, log *pixelLog, calls []text.DrawCall, er *textEraser, buf **image.RGBA, tex **ebiten.Image, fallbacks *int) (*ebiten.Image, []text.DrawCall) {
	if frame == nil {
		return frame, nil
	}
	kept, certain := textsmooth.DecideWith(calls, frame.Bounds(), log.oracle())
	if !certain {
		*fallbacks++
		return settleTextFrame(frame, calls, buf, tex)
	}
	er.apply(frame, kept)
	return frame, kept
}

// sameCalls reports whether a and b draw the same glyphs at the same places
// in the same colours, and with under, over the same cells.
func sameCalls(a, b []text.DrawCall, under bool) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		x, y := &a[i], &b[i]
		if x.Glyph != y.Glyph || x.X != y.X || x.Y != y.Y || x.Color != y.Color || x.Flat != y.Flat ||
			x.Tint != y.Tint || x.Clip != y.Clip || x.Erased != y.Erased || x.SourceOver != y.SourceOver || !slices.Equal(x.Mask, y.Mask) || !slices.Equal(x.RasterColors, y.RasterColors) {
			return false
		}
		if under && !slices.Equal(x.Under, y.Under) {
			return false
		}
	}
	return true
}

// copyCalls appends src to dst, with a private copy of each Under when under
// is set and without it otherwise.
func copyCalls(dst, src []text.DrawCall, under bool) []text.DrawCall {
	for _, c := range src {
		if under {
			c.Under = slices.Clone(c.Under)
		} else {
			c.Under = nil
		}
		c.Mask = slices.Clone(c.Mask)
		c.RasterColors = slices.Clone(c.RasterColors)
		dst = append(dst, c)
	}
	return dst
}

// glyphBounds is the rectangle every glyph in calls covers.
func glyphBounds(calls []text.DrawCall) image.Rectangle {
	var r image.Rectangle
	for _, c := range calls {
		if g := c.Glyph; g != nil && g.Width > 0 {
			r = r.Union(image.Rect(c.X, c.Y, c.X+g.Width, c.Y+len(g.Pixels)/g.Width))
		}
	}
	return r
}

// clearRect zeroes r of img.
func clearRect(img *image.RGBA, r image.Rectangle) {
	r = r.Intersect(img.Rect)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		clear(img.Pix[img.PixOffset(r.Min.X, y):img.PixOffset(r.Max.X, y)])
	}
}

// uploadRect writes r of buf into the same rectangle of tex.
func uploadRect(tex *ebiten.Image, buf *image.RGBA, r image.Rectangle, scratch *[]byte) {
	r = r.Intersect(buf.Rect)
	if r.Empty() {
		return
	}
	n := 4 * r.Dx() * r.Dy()
	if cap(*scratch) < n {
		*scratch = make([]byte, n)
	}
	pix := (*scratch)[:n]
	row := 4 * r.Dx()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		o := buf.PixOffset(r.Min.X, y)
		copy(pix[(y-r.Min.Y)*row:], buf.Pix[o:o+row])
	}
	tex.SubImage(r).(*ebiten.Image).WritePixels(pix)
}

// drawRect draws r of src onto dst at the same place.
func drawRect(dst, src *ebiten.Image, r image.Rectangle) {
	if r.Empty() {
		return
	}
	var op ebiten.DrawImageOptions
	op.GeoM.Translate(float64(r.Min.X), float64(r.Min.Y))
	dst.DrawImage(src.SubImage(r).(*ebiten.Image), &op)
}

// glyphPicture composes a picture of glyphs that is laid over content the
// frame does not model, as the viewer presents it: the picture with its glyphs
// (what a test or the pixel log reads), the picture to blit, and the glyphs to
// append each frame the picture is presented. With the overlay off the blit is
// the picture itself and there are no glyphs. With it on the blit is a copy
// without the glyphs, which the overlay draws instead, and the glyphs are
// marked erased.
func glyphPicture(enabled bool, compose func() *image.RGBA) (pic, blit *image.RGBA, calls []text.DrawCall) {
	if !enabled {
		return compose(), nil, nil
	}
	calls = text.Record(func() { pic = compose() })
	if pic == nil {
		return nil, nil, nil
	}
	blit = &image.RGBA{Pix: slices.Clone(pic.Pix), Stride: pic.Stride, Rect: pic.Rect}
	textsmooth.Erase(blit, calls)
	return pic, blit, text.MarkErased(calls)
}

// settleTextFrame reads the finished native composite back, keeps the
// captured glyphs that are still visible in it (textsmooth.Settle) and
// returns a copy with their raster erased for the upscale. Glyphs that did
// not survive stay baked in the copy. With nothing to smooth it returns
// frame itself.
func settleTextFrame(frame *ebiten.Image, calls []text.DrawCall, buf **image.RGBA, tex **ebiten.Image) (*ebiten.Image, []text.DrawCall) {
	if frame == nil || len(calls) == 0 {
		return frame, nil
	}
	w, h := frame.Bounds().Dx(), frame.Bounds().Dy()
	if *buf == nil || (*buf).Bounds().Dx() != w || (*buf).Bounds().Dy() != h {
		*buf = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	if !readFramePixels(frame, (*buf).Pix) {
		return frame, calls
	}
	// An erased glyph has no raster in the frame to find; the readback cannot
	// tell whether a later picture covered it, so it is drawn.
	var plain, erased []text.DrawCall
	for _, c := range calls {
		if c.Erased {
			erased = append(erased, c)
		} else {
			plain = append(plain, c)
		}
	}
	kept := append(textsmooth.Settle(*buf, plain), erased...)
	if len(kept) == 0 {
		return frame, nil
	}
	if *tex == nil || (*tex).Bounds().Dx() != w || (*tex).Bounds().Dy() != h {
		if *tex != nil {
			(*tex).Dispose()
		}
		*tex = ebiten.NewImage(w, h)
	}
	(*tex).WritePixels((*buf).Pix)
	return *tex, kept
}

// readFramePixels reads frame back into pix. Ebitengine refuses a readback
// before the game loop starts, which is where headless tests call Draw; there
// it reports false and the caller keeps its unsettled capture.
func readFramePixels(frame *ebiten.Image, pix []byte) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	frame.ReadPixels(pix)
	return true
}

// beginTextCapture opens its OWN standalone text.SetCapture window when
// enabled, returning the index Captured() stood at first — endTextCapture's
// own start argument. It answers -1 when disabled, which endTextCapture
// treats as "do nothing". Use this around a compose call that is NOT
// already inside another capture window (the mission viewer's own narrow,
// independently audited sites, missiontextsmoothing.go): it turns capture on
// and back off around exactly that one call, leaving every Font.Draw call
// before or after it — including ones this story has not audited — on the
// ordinary, unsmoothed path.
func beginTextCapture(enabled bool) int {
	if !enabled {
		return -1
	}
	start := text.CapturedLen()
	text.SetCapture(false)
	return start
}

// endTextCapture closes a window beginTextCapture opened and shifts every
// glyph it captured by at — the SAME offset the caller already computes to
// paste the composed sub-picture's own pixels onto its destination.
func endTextCapture(start int, at image.Point) {
	if start < 0 {
		return
	}
	text.StopCapture()
	text.ShiftCaptured(start, at.X, at.Y)
}

// markCapture and shiftCapture are beginTextCapture/endTextCapture's own
// pair for a compose call that happens INSIDE an ALREADY-OPEN outer capture
// window (the App's own single window spanning its whole Draw switch,
// app.go) — they record where a sub-picture's own glyphs start and shift
// them once it is known where the picture is pasted, without touching
// text.SetCapture/StopCapture, which would otherwise end the outer window
// early and leave everything after the sub-picture unsmoothed.
//
// Both are unconditional and always safe to call: text.CapturedLen() and
// text.ShiftCaptured are already no-ops whenever no capture window is open
// (smoothing off, or a headless/CPU caller that never opens one) because
// nothing was appended to shift, so a call site needs no enabled check of
// its own.
func markCapture() int { return text.CapturedLen() }

func shiftCapture(start int, at image.Point) {
	if start < 0 {
		return
	}
	text.ShiftCaptured(start, at.X, at.Y)
}
