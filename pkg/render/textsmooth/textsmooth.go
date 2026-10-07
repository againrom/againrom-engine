// Package textsmooth is owner decision method C (DIV-1385): a presentation-
// layer text overlay drawn at OUTPUT resolution from captured text.DrawCall
// values, instead of the baked native-resolution glyph a nearest upscale
// then stair-steps. Art is untouched; only text gets this treatment.
//
// Coverage and premultiplied colour use a Mitchell-Netravali bicubic kernel
// (B=C=1/3), with no contrast remap. The .16a class uses its encoded alpha
// ramp; .16 shades remain opaque. Compositing uses the destination's colour
// space, matching the live GPU blend rather than linear light (DIV-1386).
package textsmooth

import (
	"image"
	"image/color"
	"math"
	"slices"

	"againrom/pkg/render/text"
)

// mitchellB, mitchellC parameterize the resize kernel: Mitchell-Netravali,
// the same values round-1's own method C resized glyph coverage with.
const (
	mitchellB = 1.0 / 3.0
	mitchellC = 1.0 / 3.0
)

// kernel is the Mitchell-Netravali cubic convolution filter, zero outside
// [-2, 2].
func kernel(x float64) float64 {
	if x < 0 {
		x = -x
	}
	b, c := mitchellB, mitchellC
	switch {
	case x < 1:
		return ((12-9*b-6*c)*x*x*x + (-18+12*b+6*c)*x*x + (6 - 2*b)) / 6
	case x < 2:
		return ((-b-6*c)*x*x*x + (6*b+30*c)*x*x + (-12*b-48*c)*x + (8*b + 24*c)) / 6
	default:
		return 0
	}
}

// Remap clamps one resized coverage value to a legal coverage: the bicubic
// kernel's negative lobes can overshoot either end.
func Remap(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Coverage reads .16a levels as (level+1)/16 alpha. The .16 class uses
// paintedness alone: its levels shade opaque strokes, not their coverage.
func Coverage(g *text.Glyph) []float64 {
	return coverage(g, false)
}

func coverage(g *text.Glyph, flat bool) []float64 {
	if g == nil || g.Width <= 0 || g.Height <= 0 {
		return nil
	}
	out := make([]float64, g.Width*g.Height)
	n := min(len(g.Pixels), len(out))
	for i := 0; i < n; i++ {
		if g.Pixels[i].Painted {
			out[i] = 1
			if g.CoverageLevels && !flat {
				out[i] = float64(min(g.Pixels[i].Level, text.MaxLevel)+1) / (text.MaxLevel + 1)
			}
		}
	}
	return out
}

// shadeFields premultiplies ink by coverage before resizing. Alpha glyphs
// retain the caller's ink; opaque glyphs retain Draw's shades. Flat draws
// keep unshaded ink and full coverage on every painted cell.
func shadeFields(g *text.Glyph, c color.RGBA, flat bool, tint color.RGBA) [3][]float64 {
	var out [3][]float64
	for k := range out {
		out[k] = make([]float64, g.Width*g.Height)
	}
	n := min(len(g.Pixels), g.Width*g.Height)
	for i := 0; i < n; i++ {
		p := g.Pixels[i]
		if !p.Painted {
			continue
		}
		s := text.DrawCall{Color: c, Flat: flat, Tint: tint}.ShownColor(p.Level)
		cov := 1.0
		if g.CoverageLevels && !flat {
			s = text.Tinted(tint, c)
			cov = float64(min(p.Level, text.MaxLevel)+1) / (text.MaxLevel + 1)
		}
		out[0][i], out[1][i], out[2][i] = float64(s.R)*cov, float64(s.G)*cov, float64(s.B)*cov
	}
	return out
}

// tap is one resampled output position: up to four source indices and the
// (already normalised) weight the kernel gives each.
type tap struct {
	idx [4]int
	w   [4]float64
}

// taps builds one 1D resampling plan per output position, srcN -> dstN, with
// every source sample clamped to [0, srcN) — the glyph's own cell has no
// more data past its edge, so a kernel radius that reaches past it repeats
// the edge sample rather than reading black or reading another glyph.
// Weights are normalised so a resize of a constant field reproduces that
// same constant exactly, including at a clamped edge.
func taps(srcN, dstN int) []tap {
	out := make([]tap, dstN)
	if srcN <= 0 || dstN <= 0 {
		return out
	}
	scale := float64(srcN) / float64(dstN)
	for d := 0; d < dstN; d++ {
		center := (float64(d)+0.5)*scale - 0.5
		base := int(math.Floor(center)) - 1
		var t tap
		sum := 0.0
		for k := 0; k < 4; k++ {
			s := base + k
			w := kernel(center - float64(s))
			if s < 0 {
				s = 0
			}
			if s >= srcN {
				s = srcN - 1
			}
			t.idx[k] = s
			t.w[k] = w
			sum += w
		}
		if sum != 0 {
			for k := range t.w {
				t.w[k] /= sum
			}
		}
		out[d] = t
	}
	return out
}

// Resize resamples src (srcW x srcH, row-major) to dstW x dstH with the
// Mitchell-Netravali kernel above, separably: a horizontal pass then a
// vertical one. It answers nil for a non-positive size on either side, or a
// source shorter than srcW*srcH.
func Resize(src []float64, srcW, srcH, dstW, dstH int) []float64 {
	if srcW <= 0 || srcH <= 0 || dstW <= 0 || dstH <= 0 || len(src) < srcW*srcH {
		return nil
	}
	xTaps := taps(srcW, dstW)
	yTaps := taps(srcH, dstH)

	mid := make([]float64, dstW*srcH)
	for y := 0; y < srcH; y++ {
		row := src[y*srcW : y*srcW+srcW]
		for x := 0; x < dstW; x++ {
			t := xTaps[x]
			v := row[t.idx[0]]*t.w[0] + row[t.idx[1]]*t.w[1] + row[t.idx[2]]*t.w[2] + row[t.idx[3]]*t.w[3]
			mid[y*dstW+x] = v
		}
	}

	out := make([]float64, dstW*dstH)
	for x := 0; x < dstW; x++ {
		for y := 0; y < dstH; y++ {
			t := yTaps[y]
			v := mid[t.idx[0]*dstW+x]*t.w[0] + mid[t.idx[1]*dstW+x]*t.w[1] +
				mid[t.idx[2]*dstW+x]*t.w[2] + mid[t.idx[3]*dstW+x]*t.w[3]
			out[y*dstW+x] = v
		}
	}
	return out
}

// TargetRect is one glyph's OWN output rectangle, its four edges rounded
// independently rather than derived from a rounded width and height. That is
// what keeps two adjacent glyphs' target rectangles touching with no gap or
// overlap at a non-integer scale (round-1's own recommendation 4); rounding
// width = round(w*scale) instead reintroduces uneven inter-glyph spacing.
func TargetRect(nativeX, nativeY, nativeW, nativeH int, scale, originX, originY float64) image.Rectangle {
	left := int(math.Round(originX + float64(nativeX)*scale))
	top := int(math.Round(originY + float64(nativeY)*scale))
	right := int(math.Round(originX + float64(nativeX+nativeW)*scale))
	bottom := int(math.Round(originY + float64(nativeY+nativeH)*scale))
	return image.Rect(left, top, right, bottom)
}

// Composite draws every call in calls onto dst — already at output
// resolution — as owner decision method C: each glyph's coverage is resized
// to its own TargetRect, remapped, then alpha-composited over dst's existing
// content in dst's own colour space at the caller's own colour, scaled by
// the caller's own alpha. scale/originX/originY are the SAME affine
// transform (frame.Placement.Scale/Origin) the art itself was already
// upscaled and placed with, so a glyph's overlay lands exactly where its
// native-resolution pixels would have.
//
// dst is expected fully transparent (a freshly cleared overlay buffer) or
// already holding earlier glyphs from the same frame; it is never the
// background art itself, so this never needs to read the art to blend over
// it — a live GPU DrawImage of dst onto the real screen does that, in the
// engine's own ordinary (non-gamma) alpha blend, matching this function's
// own colour space (see the package doc).
func Composite(dst *image.RGBA, calls []text.DrawCall, scale, originX, originY float64) {
	composite(dst, calls, scale, originX, originY, nil)
}

// Cache keeps each glyph's resampled output across frames. The resample is
// a pure function of the glyph, its colour and its target size, and it was
// the dominant per-frame cost of the overlay: every frame redrew every
// visible glyph from scratch. The zero value is ready to use.
type Cache struct {
	plans map[planKey][]planDot
}

// cacheLimit bounds the plans a Cache holds; past it the cache starts over
// rather than grow without bound on a screen that keeps changing size.
const cacheLimit = 2048

type planKey struct {
	g    *text.Glyph
	c    color.RGBA
	flat bool
	tint color.RGBA
	w, h int
}

// planDot is one output pixel a glyph's resample paints, at its offset
// inside the glyph's own target rectangle.
type planDot struct {
	col, row int
	c        color.RGBA
	alpha    float64
}

// Composite is the package Composite through the cache. It answers the
// bounding rectangle of every target rectangle it painted, clipped to dst:
// no pixel outside it changed.
func (c *Cache) Composite(dst *image.RGBA, calls []text.DrawCall, scale, originX, originY float64) image.Rectangle {
	if c.plans == nil || len(c.plans) > cacheLimit {
		c.plans = make(map[planKey][]planDot)
	}
	return composite(dst, calls, scale, originX, originY, c.plans)
}

func composite(dst *image.RGBA, calls []text.DrawCall, scale, originX, originY float64, plans map[planKey][]planDot) image.Rectangle {
	var dirty image.Rectangle
	if dst == nil {
		return dirty
	}
	for _, call := range calls {
		g := call.Glyph
		if g == nil || g.Width <= 0 || g.Height <= 0 {
			continue
		}
		rect := TargetRect(call.X, call.Y, g.Width, g.Height, scale, originX, originY)
		dw, dh := rect.Dx(), rect.Dy()
		if dw <= 0 || dh <= 0 {
			continue
		}
		// A glyph drawn into a clipped image paints only the cells inside the
		// clip, so its overlay stops at the same edge.
		limit := rect
		if !call.Clip.Empty() {
			limit = rect.Intersect(TargetRect(call.Clip.Min.X, call.Clip.Min.Y, call.Clip.Dx(), call.Clip.Dy(), scale, originX, originY))
		}
		key := planKey{g, call.Color, call.Flat, call.Tint, dw, dh}
		plan, ok := plans[key]
		if len(call.RasterColors) > 0 {
			plan = rasterPlan(call, dw, dh)
		} else if !ok {
			plan = resamplePlan(g, call.Color, call.Flat, call.Tint, dw, dh)
			if plans != nil {
				plans[key] = plan
			}
		}
		for _, d := range plan {
			x, y := rect.Min.X+d.col, rect.Min.Y+d.row
			if limit != rect && !image.Pt(x, y).In(limit) {
				continue
			}
			if call.Mask != nil && !call.Mask[sourceCell(g, d.col, d.row, dw, dh)] {
				continue
			}
			blendPixel(dst, x, y, d.c, d.alpha)
		}
		if len(plan) > 0 {
			dirty = dirty.Union(limit.Intersect(dst.Bounds()))
		}
	}
	return dirty
}

// sourceCell is the index in g.Pixels of the cell output pixel (col, row) of
// a dw x dh resample of g sits over.
func sourceCell(g *text.Glyph, col, row, dw, dh int) int {
	x := min(g.Width-1, int((float64(col)+0.5)*float64(g.Width)/float64(dw)))
	y := min(g.Height-1, int((float64(row)+0.5)*float64(g.Height)/float64(dh)))
	return y*g.Width + x
}

// resamplePlan is one glyph's resample at dw x dh: every output pixel it
// paints, in the row-major order Composite blends them.
func resamplePlan(g *text.Glyph, c color.RGBA, flat bool, tint color.RGBA, dw, dh int) []planDot {
	resized := Resize(coverage(g, flat), g.Width, g.Height, dw, dh)
	if resized == nil {
		return nil
	}
	fields := shadeFields(g, c, flat, tint)
	return resampleFields(g, c, dw, dh, resized, fields)
}

func rasterPlan(call text.DrawCall, dw, dh int) []planDot {
	g := call.Glyph
	fields := shadeFields(g, call.Color, true, call.Tint)
	for n, p := range g.Pixels {
		if p.Painted && n < len(call.RasterColors) {
			c := call.NativeColor(p.Level, n)
			fields[0][n], fields[1][n], fields[2][n] = float64(c.R), float64(c.G), float64(c.B)
		}
	}
	return resampleFields(g, color.RGBA{A: 255}, dw, dh, Resize(coverage(g, true), g.Width, g.Height, dw, dh), fields)
}

func resampleFields(g *text.Glyph, c color.RGBA, dw, dh int, resized []float64, fields [3][]float64) []planDot {
	var shade [3][]float64
	for k := range fields {
		shade[k] = Resize(fields[k], g.Width, g.Height, dw, dh)
	}
	alphaScale := float64(c.A) / 255
	if alphaScale <= 0 {
		return nil
	}
	var plan []planDot
	for row := 0; row < dh; row++ {
		for col := 0; col < dw; col++ {
			cov := resized[row*dw+col]
			v := Remap(cov)
			if v <= 0 || cov <= 0 {
				continue
			}
			pc := c
			ch := func(k int) uint8 {
				return uint8(math.Round(math.Max(0, math.Min(255, shade[k][row*dw+col]/cov))))
			}
			pc.R, pc.G, pc.B = ch(0), ch(1), ch(2)
			plan = append(plan, planDot{col: col, row: row, c: pc, alpha: v * alphaScale})
		}
	}
	return plan
}

// blendPixel writes c over dst's existing pixel at (x, y) with straight
// alpha alpha, clamped to dst's own bounds. It is the one place a pixel is
// written, so a test can pin its exact arithmetic once.
func blendPixel(dst *image.RGBA, x, y int, c color.RGBA, alpha float64) {
	b := dst.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	if alpha >= 1 {
		i := dst.PixOffset(x, y)
		dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = c.R, c.G, c.B, 255
		return
	}
	i := dst.PixOffset(x, y)
	inv := 1 - alpha
	dst.Pix[i] = uint8(float64(c.R)*alpha + float64(dst.Pix[i])*inv)
	dst.Pix[i+1] = uint8(float64(c.G)*alpha + float64(dst.Pix[i+1])*inv)
	dst.Pix[i+2] = uint8(float64(c.B)*alpha + float64(dst.Pix[i+2])*inv)
	dst.Pix[i+3] = uint8(255*alpha + float64(dst.Pix[i+3])*inv)
}

// Settle keeps only the captured glyphs that survived into frame, the final
// native composite, and erases their baked raster so the overlay can repaint
// them. A glyph survives when every painted cell inside frame still holds the
// colour Draw wrote, or the colour of a later kept glyph painted over it, and
// every cell it replaced was opaque. A glyph a later picture covered, one
// drawn onto a picture that was never pasted, or one shifted to the wrong
// place fails the test and stays baked, pixel-exact. Repeated identical draws
// are kept once. frame is modified in place.
func Settle(frame *image.RGBA, calls []text.DrawCall) []text.DrawCall {
	kept, _ := Decide(calls, frame.Bounds(), func(x, y int, want color.RGBA) Verdict {
		if frame.RGBAAt(x, y) == want {
			return Matches
		}
		return Differs
	})
	Erase(frame, kept)
	return kept
}

// Verdict is what a cell oracle knows about one cell of the final frame
// against the colour a glyph painted there.
type Verdict uint8

const (
	// Differs: the cell certainly holds another colour.
	Differs Verdict = iota
	// Matches: the cell certainly holds exactly that colour.
	Matches
	// Unknown: the oracle cannot tell.
	Unknown
)

// Fate is what Explain settled for one captured call.
type Fate uint8

const (
	// Kept: every painted cell inside the frame and the call's clip shows the
	// glyph, on its own or under a later kept glyph, or the cells that do not
	// are covered by a later picture and left out of Mask. The overlay
	// redraws it.
	Kept Fate = iota
	// Blank: the glyph paints no cell.
	Blank
	// Repeat: an earlier call drew the same glyph at the same place in the
	// same colour.
	Repeat
	// Hidden: no painted cell inside the frame and the clip shows the glyph.
	Hidden
	// Translucent: a painted cell replaced a cell that was not opaque and
	// the colour beneath is not known, so the raster cannot be put back. It
	// stays baked.
	Translucent
	// Undecided: the oracle cannot tell whether the glyph shows.
	Undecided
	// Unrecorded: the call holds no record of the cells it replaced.
	Unrecorded
)

// String names the fate.
func (f Fate) String() string {
	switch f {
	case Kept:
		return "kept"
	case Blank:
		return "blank"
	case Repeat:
		return "repeat"
	case Hidden:
		return "hidden"
	case Translucent:
		return "translucent"
	case Undecided:
		return "undecided"
	case Unrecorded:
		return "unrecorded"
	}
	return "unknown"
}

// Outcome is Explain's answer for one call: its fate and, for a kept call,
// the call as the overlay draws it. The kept call carries an Under record
// resolved over the colour beneath wherever the glyph replaced a translucent
// cell, and a Mask when a later picture covers part of it.
type Outcome struct {
	Fate Fate
	Call text.DrawCall
}

// Oracle is how Explain reads the final frame. A nil field is a question the
// oracle cannot answer, and a glyph that depends on it stays baked.
type Oracle struct {
	// Cell answers whether cell (x, y) holds want.
	Cell func(x, y int, want color.RGBA) Verdict
	// Below answers the colour a cell held just before the draw that put want
	// on it, when it can tell. A glyph drawn onto a picture that is not opaque
	// there needs it: a kept glyph of that kind carries an Under record
	// resolved over that colour, so putting it back leaves the cell as the
	// frame would hold it without the glyph.
	Below func(x, y int, want color.RGBA) (color.RGBA, bool)
	// Tint answers the translucent colour drawn over the cell since the draw
	// that put want on it: Matches when want was the last opaque draw there and
	// every later draw is a translucent one the oracle knows (the colour is
	// zero when there is none), Differs when the last opaque draw there is
	// another colour, Unknown otherwise. It is how a dimmed glyph is told from
	// a covered one.
	Tint func(x, y int, want color.RGBA) (color.RGBA, Verdict)
}

// Decide is Settle's own selection over any cell oracle instead of a frame
// read back from the GPU. It answers certain false when some glyph's
// survival hangs on a cell the oracle cannot tell and no other cell of that
// glyph already rules it out: only the real frame can settle that glyph.
func Decide(calls []text.DrawCall, bounds image.Rectangle, cell func(x, y int, want color.RGBA) Verdict) (kept []text.DrawCall, certain bool) {
	return DecideWith(calls, bounds, Oracle{Cell: cell})
}

// DecideWith is Decide over a full oracle.
func DecideWith(calls []text.DrawCall, bounds image.Rectangle, o Oracle) (kept []text.DrawCall, certain bool) {
	out, certain := Explain(calls, bounds, o)
	kept = make([]text.DrawCall, 0, len(calls))
	for _, r := range out {
		if r.Fate == Kept {
			kept = append(kept, r.Call)
		}
	}
	return kept, certain
}

// Cell states Explain reads a glyph's cells into.
const (
	cellNone uint8 = iota
	cellShown
	cellCovered
	cellUnsure
)

// holds answers whether cell (x, y) holds want under dim, the translucent
// colour the glyph being judged is dimmed by (zero for none).
func (o Oracle) holds(dim color.RGBA, x, y int, want color.RGBA) Verdict {
	if dim == (color.RGBA{}) || o.Tint == nil {
		return o.Cell(x, y, want)
	}
	t, v := o.Tint(x, y, want)
	if v == Matches && t != dim {
		return Differs
	}
	return v
}

// dimOf finds the translucent colour a later draw laid over the call's
// glyph, from the first cell the oracle can explain by one. It answers zero
// for a glyph whose first decisive cell holds its own colour.
func (o Oracle) dimOf(call text.DrawCall, clip image.Rectangle) color.RGBA {
	if o.Tint == nil {
		return color.RGBA{}
	}
	g := call.Glyph
	for n, p := range g.Pixels {
		if !p.Painted {
			continue
		}
		at := image.Pt(call.X+n%g.Width, call.Y+n/g.Width)
		if !at.In(clip) {
			continue
		}
		want := call.NativeColor(p.Level, n)
		if o.Cell(at.X, at.Y, want) == Matches {
			return color.RGBA{}
		}
		if t, v := o.Tint(at.X, at.Y, want); v == Matches {
			return t
		}
	}
	return color.RGBA{}
}

// Explain settles every call and answers its outcome, in call order.
//
// Calls are read latest first. A glyph drawn earlier than a kept glyph may
// have a cell that the later glyph painted over: a flat shadow under its
// face, or a neighbour's cell that overlaps its own. Such a cell still counts
// as showing the earlier glyph, because the later glyph is redrawn above it
// by the overlay and both rasters are erased.
//
// A glyph some of whose cells a later picture covers is kept with a Mask of
// the cells that show. A cell the oracle cannot tell, on such a glyph, is
// left out of the Mask rather than sending the frame to a readback.
//
// A glyph a translucent draw dimmed, after the glyph was drawn, is kept with
// the dim as its Tint and the cells it replaced dimmed the same way.
func Explain(calls []text.DrawCall, bounds image.Rectangle, o Oracle) (out []Outcome, certain bool) {
	type key struct {
		g    *text.Glyph
		x, y int
		c    color.RGBA
		flat bool
		tint color.RGBA
		over bool
	}
	out = make([]Outcome, len(calls))
	settled := make([]bool, len(calls))
	seen := make(map[key]int, len(calls))
	for i, call := range calls {
		g := call.Glyph
		if g == nil || g.Width <= 0 || (!call.Erased && len(call.Under) != len(g.Pixels)) {
			out[i].Fate, settled[i] = Unrecorded, true
			continue
		}
		k := key{g, call.X, call.Y, call.Color, call.Flat, call.Tint, call.SourceOver}
		if previous, ok := seen[k]; ok && slices.Equal(call.RasterColors, calls[previous].RasterColors) && (!call.SourceOver || slices.Equal(call.Under, calls[previous].Under)) {
			out[i].Fate, settled[i] = Repeat, true
			continue
		}
		seen[k] = i
	}
	certain = true
	var above owners
	var states []uint8
	for i := len(calls) - 1; i >= 0; i-- {
		if settled[i] {
			continue
		}
		call := calls[i]
		g := call.Glyph
		clip := clipOf(call, bounds)
		dim := o.dimOf(call, clip)
		states = slices.Grow(states[:0], len(g.Pixels))[:len(g.Pixels)]
		clear(states)
		inked, translucent := false, false
		inside, shown, covered, unsure := 0, 0, 0, 0
		var resolved []color.RGBA
		for n, p := range g.Pixels {
			if !p.Painted {
				continue
			}
			inked = true
			at := image.Pt(call.X+n%g.Width, call.Y+n/g.Width)
			if !at.In(clip) {
				continue
			}
			inside++
			want := call.NativeColor(p.Level, n)
			if !call.Erased && call.Under[n].A != 255 {
				if call.Tint != (color.RGBA{}) || dim != (color.RGBA{}) {
					translucent = true
					break
				}
				r, ok := resolveUnder(call.Under[n], at, want, &above, bounds, o.Below)
				if !ok {
					translucent = true
					break
				}
				if resolved == nil {
					resolved = slices.Clone(call.Under)
				}
				resolved[n] = r
				states[n] = cellShown
				shown++
				continue
			}
			switch o.holds(dim, at.X, at.Y, want) {
			case Matches:
				states[n] = cellShown
				shown++
			case Unknown:
				// An erased glyph has no raster for a readback to find, so a
				// cell nothing is known to cover counts as showing.
				if call.Erased {
					states[n] = cellShown
					shown++
					break
				}
				states[n] = cellUnsure
				unsure++
			default:
				over, ok := above.at(at, bounds)
				if !ok {
					states[n] = cellCovered
					covered++
					break
				}
				switch o.holds(dim, at.X, at.Y, over) {
				case Matches:
					states[n] = cellShown
					shown++
				case Unknown:
					if call.Erased {
						states[n] = cellShown
						shown++
						break
					}
					states[n] = cellUnsure
					unsure++
				default:
					states[n] = cellCovered
					covered++
				}
			}
		}
		switch {
		case !inked:
			out[i].Fate = Blank
		case inside == 0:
			out[i].Fate = Hidden
		case translucent:
			out[i].Fate = Translucent
		case shown == 0 && (covered > 0 || unsure == 0):
			out[i].Fate = Hidden
		case covered == 0 && unsure > 0:
			out[i].Fate = Undecided
			certain = false
		default:
			kept := call
			kept.Under = orElse(resolved, call.Under)
			if covered > 0 || unsure > 0 {
				kept.Mask = maskOf(g, states)
			}
			if total := text.Tinted(dim, call.Tint); total != (color.RGBA{}) {
				kept.Tint = total
				kept.Under = dimmed(g, call.Under, total)
			}
			out[i].Fate, out[i].Call = Kept, kept
			above.add(call)
		}
	}
	return out, certain
}

// dimmed is under with the translucent colour t drawn over every cell the
// glyph paints: what the frame holds there without the glyph, once a dim
// covers it.
func dimmed(g *text.Glyph, under []color.RGBA, t color.RGBA) []color.RGBA {
	out := make([]color.RGBA, len(under))
	for n, p := range g.Pixels {
		if p.Painted {
			out[n] = text.Tinted(t, under[n])
		}
	}
	return out
}

// orElse is resolved when there is one, and fallback otherwise.
func orElse(resolved, fallback []color.RGBA) []color.RGBA {
	if resolved != nil {
		return resolved
	}
	return fallback
}

// maskOf is the cells of g an overlay may draw, given how each cell fared:
// a painted cell that shows, and an unpainted cell no covered cell touches,
// so that the resampled edge of the glyph does not bleed onto the picture
// that covers it.
func maskOf(g *text.Glyph, states []uint8) []bool {
	mask := make([]bool, len(g.Pixels))
	for n := range mask {
		if g.Pixels[n].Painted {
			mask[n] = states[n] == cellShown
			continue
		}
		mask[n] = true
		x, y := n%g.Width, n/g.Width
		for dy := -1; dy <= 1 && mask[n]; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx >= g.Width || ny >= g.Height {
					continue
				}
				if s := states[ny*g.Width+nx]; s == cellCovered || s == cellUnsure {
					mask[n] = false
					break
				}
			}
		}
	}
	return mask
}

// resolveUnder answers what a cell holds without the glyph that painted it,
// for a cell that replaced a translucent one: the replaced cell over what lay
// beneath the draw that put the glyph, or the kept glyph on top of it, there.
// It fails when the beneath colour is not known exactly or the result is not
// opaque, since the eraser can only put back an opaque cell.
func resolveUnder(under color.RGBA, at image.Point, want color.RGBA, above *owners, bounds image.Rectangle, below func(x, y int, want color.RGBA) (color.RGBA, bool)) (color.RGBA, bool) {
	if below == nil {
		return color.RGBA{}, false
	}
	beneath, ok := below(at.X, at.Y, want)
	if !ok {
		over, has := above.at(at, bounds)
		if !has {
			return color.RGBA{}, false
		}
		if beneath, ok = below(at.X, at.Y, over); !ok {
			return color.RGBA{}, false
		}
	}
	r, ok := composeOver(under, beneath)
	if !ok || r.A != 255 {
		return color.RGBA{}, false
	}
	return r, true
}

// composeOver is under, premultiplied, drawn source-over beneath, when every
// channel lands on an integer.
func composeOver(under, beneath color.RGBA) (color.RGBA, bool) {
	if under.A == 0 {
		return beneath, true
	}
	if under.A == 255 {
		return under, true
	}
	inv := 255 - int(under.A)
	var out [4]uint8
	for k, pair := range [4][2]uint8{{under.R, beneath.R}, {under.G, beneath.G}, {under.B, beneath.B}, {under.A, beneath.A}} {
		add := int(pair[1]) * inv
		if add%255 != 0 {
			return color.RGBA{}, false
		}
		v := int(pair[0]) + add/255
		if v > 255 {
			v = 255
		}
		out[k] = uint8(v)
	}
	return color.RGBA{R: out[0], G: out[1], B: out[2], A: out[3]}, true
}

// clipOf is the rectangle of the frame a call can have painted.
func clipOf(call text.DrawCall, bounds image.Rectangle) image.Rectangle {
	if call.Clip.Empty() {
		return bounds
	}
	return bounds.Intersect(call.Clip)
}

// owners indexes the cells that kept glyphs paint, so an earlier glyph can
// tell that a cell it painted now holds a later kept glyph's colour. The
// index is built on first use: a frame in which no glyph overlaps another
// never pays for it.
type owners struct {
	list  []text.DrawCall
	built int
	cells map[image.Point]color.RGBA
}

// add records a kept glyph. Glyphs are added latest first, so the first
// colour recorded for a cell is the one on top.
func (o *owners) add(call text.DrawCall) { o.list = append(o.list, call) }

// at answers the colour the topmost kept glyph painted on p.
func (o *owners) at(p image.Point, bounds image.Rectangle) (color.RGBA, bool) {
	if o.built < len(o.list) && o.cells == nil {
		o.cells = make(map[image.Point]color.RGBA)
	}
	for ; o.built < len(o.list); o.built++ {
		call := o.list[o.built]
		clip := clipOf(call, bounds)
		for n, px := range call.Glyph.Pixels {
			if !px.Painted {
				continue
			}
			at := image.Pt(call.X+n%call.Glyph.Width, call.Y+n/call.Glyph.Width)
			if !at.In(clip) {
				continue
			}
			if _, taken := o.cells[at]; !taken {
				o.cells[at] = call.NativeColor(px.Level, n)
			}
		}
	}
	c, ok := o.cells[p]
	return c, ok
}

// Erase puts back, in frame, the cells every kept glyph painted, from what
// each replaced. Where two kept glyphs overlap the earlier one's record
// wins, because it holds what was there before either.
func Erase(frame *image.RGBA, kept []text.DrawCall) {
	b := frame.Bounds()
	for n := len(kept) - 1; n >= 0; n-- {
		call := kept[n]
		g := call.Glyph
		if call.Erased {
			continue
		}
		clip := clipOf(call, b)
		for i, p := range g.Pixels {
			at := image.Pt(call.X+i%g.Width, call.Y+i/g.Width)
			if p.Painted && at.In(clip) && (call.Mask == nil || call.Mask[i]) {
				frame.SetRGBA(at.X, at.Y, call.Under[i])
			}
		}
	}
}
