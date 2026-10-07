package text

import (
	"image"
	"image/color"
	"runtime"
	"strconv"
	"strings"
)

// DrawCall holds a glyph, its native position and ink for the output overlay.
//
// Under holds, for a rasterised draw, the destination pixel each glyph cell
// replaced, indexed like Glyph.Pixels; an unpainted or clipped cell holds the
// zero colour. It lets the overlay verify that a glyph survived into the final
// frame and put back what the glyph covered. A skip-raster draw leaves it nil.
type DrawCall struct {
	// RasterColors retains exact glyph cells after a nonlinear frame remap.
	RasterColors []color.RGBA
	Glyph        *Glyph
	X, Y         int
	Color        color.RGBA
	Under        []color.RGBA
	// Flat marks a DrawFlat call: every painted cell holds Color itself,
	// whatever its level.
	Flat bool
	// Clip is the bounds of the image the glyph was drawn into. A cell
	// outside it was never painted. The zero rectangle means no clip.
	Clip image.Rectangle
	// Mask, when set, marks the cells of the glyph the overlay may draw,
	// indexed like Glyph.Pixels: a false cell is covered by a later picture.
	// Only the overlay's own decision sets it.
	Mask []bool
	// Tint is a translucent colour, premultiplied, that a later draw laid over
	// every cell of the glyph: a disabled row's dim. The zero colour means
	// none. A composer sets it through TintSince; the overlay's decision adds
	// the tint of any draw it finds over a glyph, so on a call it kept Tint is
	// the whole of it.
	Tint color.RGBA
	// Erased marks a glyph drawn on a picture that is presented without it,
	// over content the frame does not model: a line of text on a transparent
	// picture laid over the world. There is no raster of it in the frame to
	// erase, and the cells the overlay may draw are the ones no later picture
	// covers.
	Erased     bool
	SourceOver bool
}

// CellColor is the colour the call paints on a cell of the given level.
func (c DrawCall) CellColor(level uint8) color.RGBA {
	if c.Flat {
		return c.Color
	}
	return Shade(c.Color, level)
}

// ShownColor is the colour a cell of the given level holds in the frame: the
// colour the call painted, under the call's tint.
func (c DrawCall) ShownColor(level uint8) color.RGBA {
	return Tinted(c.Tint, c.CellColor(level))
}

func (c DrawCall) NativeColor(level uint8, cell int) color.RGBA {
	if cell >= 0 && cell < len(c.RasterColors) {
		return Tinted(c.Tint, c.RasterColors[cell])
	}
	ink := c.CellColor(level)
	if c.SourceOver && cell >= 0 && cell < len(c.Under) {
		ink = Tinted(ink, c.Under[cell])
	}
	return Tinted(c.Tint, ink)
}

func DrawGlyphOver(dst *image.RGBA, g *Glyph, x, y int, c color.RGBA) {
	if dst == nil || g == nil || g.Width <= 0 {
		return
	}
	auditGlyph(g, capturing)
	if capturing {
		captured = append(captured, DrawCall{Glyph: g, X: x, Y: y, Color: c,
			Under: recordUnder(dst, g, x, y), Clip: dst.Bounds(), Flat: true, SourceOver: true})
		if skipRaster {
			return
		}
	}
	for n, p := range g.Pixels {
		at := image.Pt(x+n%g.Width, y+n/g.Width)
		if p.Painted && at.In(dst.Bounds()) {
			dst.SetRGBA(at.X, at.Y, Tinted(c, dst.RGBAAt(at.X, at.Y)))
		}
	}
}

// MarkErased flags every call in calls as Erased and answers calls.
func MarkErased(calls []DrawCall) []DrawCall {
	for i := range calls {
		calls[i].Erased = true
	}
	return calls
}

// Tinted is c with the translucent colour t drawn over it, both
// premultiplied, by the source-over arithmetic of image/draw's solid fill.
func Tinted(t, c color.RGBA) color.RGBA {
	if t == (color.RGBA{}) {
		return c
	}
	const m = 1<<16 - 1
	a := (m - uint32(t.A)*0x101) * 0x101
	over := func(d, s uint8) uint8 { return uint8((uint32(d)*a/m + uint32(s)*0x101) >> 8) }
	return color.RGBA{R: over(c.R, t.R), G: over(c.G, t.G), B: over(c.B, t.B), A: over(c.A, t.A)}
}

// TintSince records that the translucent colour c was drawn over within, after
// the glyphs captured since start. Every such glyph whose painted cells all lie
// in within takes it as a tint; the others are left as they are. It does
// nothing outside a capture window.
func TintSince(start int, within image.Rectangle, c color.RGBA) {
	if !capturing || c.A == 0 || c.A == 255 {
		return
	}
	for i := max(start, 0); i < len(captured); i++ {
		call := &captured[i]
		g := call.Glyph
		if g == nil || g.Width <= 0 {
			continue
		}
		inside := true
		for n, p := range g.Pixels {
			if !p.Painted {
				continue
			}
			at := image.Pt(call.X+n%g.Width, call.Y+n/g.Width)
			if !call.Clip.Empty() && !at.In(call.Clip) {
				continue
			}
			if !at.In(within) {
				inside = false
				break
			}
		}
		if inside {
			call.Tint = Tinted(c, call.Tint)
		}
	}
}

// Underlay replaces what every glyph captured since start recorded as lying
// under it with what backdrop holds at the same cells. A composer that paints
// its text on a transparent layer and lays the layer over its background
// afterwards calls it before the layer goes down: the glyph then sits on the
// background, and the recorded cell is the one to put back, not the empty cell
// of the layer. It does nothing outside a capture window.
func Underlay(start int, backdrop *image.RGBA) {
	if !capturing || backdrop == nil {
		return
	}
	b := backdrop.Bounds()
	for i := max(start, 0); i < len(captured); i++ {
		call := &captured[i]
		g := call.Glyph
		if g == nil || g.Width <= 0 || len(call.Under) != len(g.Pixels) {
			continue
		}
		for n, p := range g.Pixels {
			if !p.Painted {
				continue
			}
			if at := image.Pt(call.X+n%g.Width, call.Y+n/g.Width); at.In(b) {
				call.Under[n] = backdrop.RGBAAt(at.X, at.Y)
			}
		}
	}
}

var (
	capturing  bool
	skipRaster bool
	captured   []DrawCall
)

// ResetCapture clears every previously captured glyph and turns capture off.
// The presentation layer calls it once at the start of a frame it means to
// smooth; SetCapture/StopCapture then toggle recording within that frame
// without losing an earlier window's own captured glyphs, which is what lets
// two separately composed sub-pictures in the same frame each open and close
// their own capture window and still land in one ordered list.
func ResetCapture() {
	capturing = false
	skipRaster = false
	captured = captured[:0]
}

// SetCapture starts recording every following Draw call into Captured, tree-
// wide, until StopCapture. skip additionally suppresses the raster itself:
// Draw records the call and paints nothing, which is what lets a composer's
// own background render come back with every glyph pixel removed and
// nothing else changed, at zero call-site cost — the round-1/round-2
// prototype's own recorder/skip-raster hook, made permanent.
//
// It is a package-level switch and not a parameter Draw takes, which is what
// keeps every production call site unedited; the presentation layer is the
// only caller, and every call site not standing inside a capture window
// (nothing does, by default) draws exactly as it always has.
func SetCapture(skip bool) {
	capturing = true
	skipRaster = skip
}

// StopCapture ends capture and restores Draw's ordinary behaviour: paint
// every pixel, record nothing. It is safe to call when capture is already
// off, and it does not clear Captured — ResetCapture does that, once, at the
// start of the frame.
func StopCapture() {
	capturing = false
	skipRaster = false
}

// Captured is every glyph Draw has recorded since the last ResetCapture, in
// call order.
func Captured() []DrawCall { return captured }

// CapturedLen is len(Captured()), read before composing a detached
// sub-picture so its own glyphs can be found again after it returns.
func CapturedLen() int { return len(captured) }

// ShiftCaptured adds (dx, dy) to every captured entry at index start or
// later. A caller that composes a picture on its own small canvas and pastes
// it onto a destination at a computed offset — the tavern/school dialogue
// modal, the wide detached dialogue, a hover tooltip, the mission character
// panel — records start := CapturedLen() before composing the picture, then
// calls this once with the SAME offset it already computes to paste the
// picture's own pixels, so the captured glyphs land in the destination's own
// coordinate space exactly where the pasted pixels do.
func ShiftCaptured(start, dx, dy int) {
	if start < 0 {
		start = 0
	}
	for i := start; i < len(captured); i++ {
		captured[i].X += dx
		captured[i].Y += dy
		captured[i].Clip = captured[i].Clip.Add(image.Pt(dx, dy))
	}
}

// Capturing reports whether a capture window is open.
func Capturing() bool { return capturing }

// Record runs draw inside its own rasterising capture window and returns the
// glyphs it placed, restoring the caller's window afterwards. A composer whose
// picture outlives the frame keeps these beside the picture and hands them to
// Append each frame the picture is presented, so its text is smoothed on
// every frame, not only the one that drew it.
func Record(draw func()) []DrawCall {
	wasCapturing, wasSkip, outer := capturing, skipRaster, captured
	captured = nil
	capturing, skipRaster = true, false
	draw()
	own := captured
	captured, capturing, skipRaster = outer, wasCapturing, wasSkip
	return own
}

// Append adds calls to the open capture window, shifted by (dx, dy). It does
// nothing when no window is open.
func Append(calls []DrawCall, dx, dy int) {
	if !capturing {
		return
	}
	for _, c := range calls {
		c.X += dx
		c.Y += dy
		c.Clip = c.Clip.Add(image.Pt(dx, dy))
		captured = append(captured, c)
	}
}

// recordUnder snapshots the destination cells g will replace at (x, y).
func recordUnder(dst *image.RGBA, g *Glyph, x, y int) []color.RGBA {
	under := make([]color.RGBA, len(g.Pixels))
	b := dst.Bounds()
	for i, p := range g.Pixels {
		if !p.Painted || g.Width == 0 {
			continue
		}
		at := image.Pt(x+i%g.Width, y+i/g.Width)
		if at.In(b) {
			under[i] = dst.RGBAAt(at.X, at.Y)
		}
	}
	return under
}

// Audit is what StartAudit counts: every glyph Draw or DrawFlat painted at
// least one cell of, by whether a capture window recorded it. A test reads it
// to find a route that paints text no window sees.
type Audit struct {
	// Captured counts glyphs a capture window recorded.
	Captured int
	// Uncaptured counts glyphs painted with no window recording them, and
	// Sites names the two innermost callers outside this package of each.
	Uncaptured int
	Sites      map[string]int
}

var audit *Audit

// StartAudit begins counting from zero. It costs one nil test per glyph while
// off and is meant for tests and diagnostics.
func StartAudit() { audit = &Audit{Sites: map[string]int{}} }

// StopAudit ends counting and answers what it counted since StartAudit; it is
// the zero Audit when none was started.
func StopAudit() Audit {
	a := audit
	audit = nil
	if a == nil {
		return Audit{}
	}
	return *a
}

// inked reports whether g paints at least one cell.
func (g *Glyph) inked() bool {
	for _, p := range g.Pixels {
		if p.Painted {
			return true
		}
	}
	return false
}

// auditGlyph counts one placed glyph, recorded or not.
func auditGlyph(g *Glyph, recorded bool) {
	if audit == nil || !g.inked() {
		return
	}
	if recorded {
		audit.Captured++
		return
	}
	audit.Uncaptured++
	audit.Sites[callerSites()]++
}

// callerSites names the two innermost frames outside this package as
// "function:line <- function:line".
func callerSites() string {
	var pcs [16]uintptr
	frames := runtime.CallersFrames(pcs[:runtime.Callers(3, pcs[:])])
	var route []string
	for len(route) < 2 {
		fr, more := frames.Next()
		if fr.Function != "" && !strings.Contains(fr.Function, "pkg/render/text.") {
			route = append(route, fr.Function[strings.LastIndex(fr.Function, "/")+1:]+":"+strconv.Itoa(fr.Line))
		}
		if !more {
			break
		}
	}
	return strings.Join(route, " <- ")
}
