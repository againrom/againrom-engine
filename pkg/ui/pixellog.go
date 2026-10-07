package ui

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"unicode/utf8"

	"againrom/pkg/render/backdrop"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
)

// pixelLog records what an ebiten image was drawn with, so the text overlay
// can decide which captured glyphs survived into the frame without reading
// the frame back from the GPU. A readback in the middle of every frame
// stalls it and, in Ebitengine's command queue, strands the upload buffers
// queued before it, which grew the heap by gigabytes. A draw the log cannot
// model records its rectangle as unknown; a glyph whose verdict depends on
// such a cell sends that frame back to the readback.
type pixelLog struct {
	bounds image.Rectangle
	active bool
	// stale is set by a draw made while the log was closed: the image then
	// holds pixels the log never saw.
	stale bool
	ops   []pixelOp
}

type pixelOpKind uint8

const (
	// pixelReplace: the cells become the source (Fill, Clear, WritePixels).
	pixelReplace pixelOpKind = iota
	// pixelOver: the source is blended source-over at 1:1 (DrawImage at an
	// integer offset, an unscaled solid rectangle).
	pixelOver
	// pixelUnknown: the cells hold something the log cannot tell.
	pixelUnknown
	pixelRemap
)

type pixelOp struct {
	lookup *backdrop.Lookup
	shows  uint64
	mask   *image.RGBA
	maskAt image.Point
	kind   pixelOpKind
	rect   image.Rectangle
	// pic is the source picture; its Rect.Min lands on rect.Min.
	pic *image.RGBA
	// layer is another image's log; cell (x, y) reads layer cell
	// (x, y) - shift.
	layer *pixelLog
	shift image.Point
	// solid is the source when pic and layer are nil.
	solid color.RGBA
}

// reset opens the log for one Draw over an image whose content is unknown.
func (l *pixelLog) reset(bounds image.Rectangle) {
	l.bounds, l.active, l.stale = bounds, true, false
	l.ops = append(l.ops[:0], pixelOp{kind: pixelUnknown, rect: bounds})
}

// begin opens the log for one Draw over an image that keeps its pixels
// between frames.
func (l *pixelLog) begin(bounds image.Rectangle) {
	if l.stale || l.bounds != bounds || len(l.ops) == 0 {
		l.reset(bounds)
		return
	}
	l.active = true
}

func (l *pixelLog) end() { l.active = false }

func (l *pixelLog) add(op pixelOp) {
	if l == nil {
		return
	}
	if !l.active {
		l.stale = true
		return
	}
	op.rect = op.rect.Intersect(l.bounds)
	if op.rect.Empty() {
		return
	}
	if op.kind == pixelReplace && op.rect == l.bounds {
		l.ops = l.ops[:0]
	}
	l.ops = append(l.ops, op)
}

// upload records WritePixels of pic over the whole image.
func (l *pixelLog) upload(pic *image.RGBA) {
	if pic == nil || pic.Stride != 4*pic.Rect.Dx() || pic.Rect.Size() != l.bounds.Size() {
		l.unknown(l.bounds)
		return
	}
	l.add(pixelOp{kind: pixelReplace, rect: l.bounds, pic: pic})
}

// fill records Fill or Clear with c.
func (l *pixelLog) fill(r image.Rectangle, c color.Color) {
	if s, ok := exactRGBA(c); ok {
		l.add(pixelOp{kind: pixelReplace, rect: r, solid: s})
		return
	}
	l.unknown(r)
}

// over records DrawImage of a texture holding pic, at an integer offset and
// scale one.
func (l *pixelLog) over(pic *image.RGBA, at image.Point) {
	if pic == nil {
		return
	}
	r := image.Rectangle{Max: pic.Rect.Size()}.Add(at)
	if pic.Stride != 4*pic.Rect.Dx() {
		l.unknown(r)
		return
	}
	l.add(pixelOp{kind: pixelOver, rect: r, pic: pic})
}

func (l *pixelLog) overClipped(pic *image.RGBA, at image.Point, clip image.Rectangle) {
	if pic == nil {
		return
	}
	r := image.Rectangle{Min: at, Max: at.Add(pic.Bounds().Size())}.Intersect(clip)
	if r.Empty() {
		return
	}
	if r.Size() == pic.Bounds().Size() {
		l.over(pic, at)
		return
	}
	part := image.NewRGBA(image.Rectangle{Max: r.Size()})
	draw.Draw(part, part.Bounds(), pic, pic.Rect.Min.Add(r.Min.Sub(at)), draw.Src)
	l.over(part, r.Min)
}

// overSolid records an unscaled, non-antialiased solid rectangle.
func (l *pixelLog) overSolid(r image.Rectangle, c color.Color) {
	if s, ok := exactRGBA(c); ok {
		l.add(pixelOp{kind: pixelOver, rect: r, solid: s})
		return
	}
	l.unknown(r)
}

// overLayer records DrawImage of the image src describes, its cell p landing
// on p+shift inside r.
func (l *pixelLog) overLayer(src *pixelLog, r image.Rectangle, shift image.Point) {
	l.add(pixelOp{kind: pixelOver, rect: r, layer: src, shift: shift})
}

func (l *pixelLog) unknown(r image.Rectangle) {
	l.add(pixelOp{kind: pixelUnknown, rect: r})
}

func (l *pixelLog) remap(r image.Rectangle, table *backdrop.Lookup, shows uint64) {
	l.add(pixelOp{kind: pixelRemap, rect: r, lookup: table, shows: shows})
}

func (l *pixelLog) remapMask(r image.Rectangle, table *backdrop.Lookup, mask *image.RGBA, at image.Point) {
	l.add(pixelOp{kind: pixelRemap, rect: r, lookup: table, mask: mask, maskAt: at, shows: 1})
}

func (op *pixelOp) covers(p image.Point) bool {
	return p.In(op.rect) && (op.mask == nil || dialogueMaskAt(op.mask, op.maskAt, p.X, p.Y))
}

func (op *pixelOp) remaps(x, y int) uint64 {
	if op.mask != nil {
		return uint64(op.mask.RGBAAt(op.mask.Rect.Min.X+x-op.maskAt.X, op.mask.Rect.Min.Y+y-op.maskAt.Y).A)
	}
	return op.shows
}

// debugPrint records ebitenutil.DebugPrintAt: a 6x16 cell per rune, one
// pixel right of x, with a one-pixel margin.
func (l *pixelLog) debugPrint(s string, x, y int) {
	cols, rows := 0, 1
	for _, line := range strings.Split(s, "\n") {
		cols = max(cols, utf8.RuneCountInString(line))
	}
	rows += strings.Count(s, "\n")
	l.unknown(image.Rect(x, y, x+2+6*cols, y+16*rows).Inset(-1))
}

// exactRGBA is c as the 8-bit premultiplied texel a draw with it produces,
// when that texel is exact.
func exactRGBA(c color.Color) (color.RGBA, bool) {
	r, g, b, a := c.RGBA()
	if r%0x101 != 0 || g%0x101 != 0 || b%0x101 != 0 || a%0x101 != 0 {
		return color.RGBA{}, false
	}
	return color.RGBA{uint8(r / 0x101), uint8(g / 0x101), uint8(b / 0x101), uint8(a / 0x101)}, true
}

// source is the colour op draws at (x, y), when it is exact.
func (op *pixelOp) source(x, y int) (color.RGBA, bool) {
	switch {
	case op.pic != nil:
		return op.pic.RGBAAt(op.pic.Rect.Min.X+x-op.rect.Min.X, op.pic.Rect.Min.Y+y-op.rect.Min.Y), true
	case op.layer != nil:
		return op.layer.value(len(op.layer.ops), x-op.shift.X, y-op.shift.Y)
	}
	return op.solid, true
}

// value is the exact colour the image holds at (x, y) after its first n
// ops, when the log can tell it.
func (l *pixelLog) value(n, x, y int) (color.RGBA, bool) {
	p := image.Pt(x, y)
	for i := n - 1; i >= 0; i-- {
		op := &l.ops[i]
		if !op.covers(p) {
			continue
		}
		if op.kind == pixelUnknown {
			return color.RGBA{}, false
		}
		if op.kind == pixelRemap {
			c, ok := l.value(i, x, y)
			if !ok {
				return color.RGBA{}, false
			}
			for n := uint64(0); n < op.remaps(x, y); n++ {
				next := op.lookup.Color(c)
				if next == c {
					break
				}
				c = next
			}
			return c, true
		}
		s, ok := op.source(x, y)
		if !ok {
			return color.RGBA{}, false
		}
		if op.kind == pixelReplace || s.A == 255 {
			return s, true
		}
		if s == (color.RGBA{}) {
			continue
		}
		d, ok := l.value(i, x, y)
		if !ok {
			return color.RGBA{}, false
		}
		return blendExact(s, d)
	}
	return color.RGBA{}, false
}

// verdict answers textsmooth.Decide about cell (x, y) of the image.
func (l *pixelLog) verdict(x, y int, want color.RGBA) textsmooth.Verdict {
	p := image.Pt(x, y)
	for i := len(l.ops) - 1; i >= 0; i-- {
		op := &l.ops[i]
		if !op.covers(p) {
			continue
		}
		if op.kind == pixelUnknown {
			return textsmooth.Unknown
		}
		if op.kind == pixelRemap {
			c, ok := l.value(i+1, x, y)
			if !ok {
				return textsmooth.Unknown
			}
			return sameColour(c, want)
		}
		s, ok := op.source(x, y)
		if !ok {
			return textsmooth.Unknown
		}
		if op.kind == pixelReplace || s.A == 255 {
			return sameColour(s, want)
		}
		if s == (color.RGBA{}) {
			continue
		}
		d, ok := l.value(i, x, y)
		if !ok {
			return textsmooth.Unknown
		}
		return blendVerdict(s, d, want)
	}
	return textsmooth.Unknown
}

// below is the colour cell (x, y) held just before the draw that put want
// there, when the log can tell. The last draw that changes the cell must be an
// opaque one of exactly want; a translucent or different colour on top, an
// unmodelled draw, or an unknown beneath answers false.
func (l *pixelLog) below(x, y int, want color.RGBA) (color.RGBA, bool) {
	p := image.Pt(x, y)
	for i := len(l.ops) - 1; i >= 0; i-- {
		op := &l.ops[i]
		if !op.covers(p) {
			continue
		}
		if op.kind == pixelUnknown {
			return color.RGBA{}, false
		}
		s, ok := op.source(x, y)
		if !ok {
			return color.RGBA{}, false
		}
		if op.kind == pixelOver && s == (color.RGBA{}) {
			continue
		}
		if s != want || s.A != 255 {
			return color.RGBA{}, false
		}
		return l.value(i, x, y)
	}
	return color.RGBA{}, false
}

// oracle is how Explain reads the image this log describes.
func (l *pixelLog) oracle() textsmooth.Oracle {
	return textsmooth.Oracle{Cell: l.verdict, Below: l.below, Tint: l.tint}
}

// tint is the translucent colour the draws after the one that put want on cell
// (x, y) laid over it. It answers Matches when the last opaque draw there is
// exactly want and every draw after it is a translucent source-over the log
// knows (the colour is zero when there is none), Differs when the last opaque
// draw there is another colour, and Unknown when a draw the log cannot model
// stands in the way or more than four translucent draws do.
func (l *pixelLog) tint(x, y int, want color.RGBA) (color.RGBA, textsmooth.Verdict) {
	p := image.Pt(x, y)
	var later [4]color.RGBA
	n := 0
	for i := len(l.ops) - 1; i >= 0; i-- {
		op := &l.ops[i]
		if !op.covers(p) {
			continue
		}
		if op.kind == pixelUnknown {
			return color.RGBA{}, textsmooth.Unknown
		}
		var s color.RGBA
		var ok bool
		if op.kind == pixelRemap {
			s, ok = l.value(i+1, x, y)
		} else {
			s, ok = op.source(x, y)
		}
		if !ok {
			return color.RGBA{}, textsmooth.Unknown
		}
		if op.kind == pixelOver && s == (color.RGBA{}) {
			continue
		}
		if op.kind == pixelOver && s.A != 255 {
			if n == len(later) {
				return color.RGBA{}, textsmooth.Unknown
			}
			later[n] = s
			n++
			continue
		}
		if s != want || s.A != 255 {
			return color.RGBA{}, textsmooth.Differs
		}
		var t color.RGBA
		for k := n - 1; k >= 0; k-- {
			t = text.Tinted(later[k], t)
		}
		return t, textsmooth.Matches
	}
	return color.RGBA{}, textsmooth.Unknown
}

func sameColour(c, want color.RGBA) textsmooth.Verdict {
	if c == want {
		return textsmooth.Matches
	}
	return textsmooth.Differs
}

// blendChannels is source-over on premultiplied channels, in real numbers.
func blendChannels(s, d color.RGBA) [4]float64 {
	sc := [4]uint8{s.R, s.G, s.B, s.A}
	dc := [4]uint8{d.R, d.G, d.B, d.A}
	var out [4]float64
	for k := range out {
		out[k] = math.Min(255, float64(sc[k])+float64(dc[k])*float64(255-s.A)/255)
	}
	return out
}

// blendExact is source-over when every channel lands on an integer, the only
// case a GPU's rounding cannot move.
func blendExact(s, d color.RGBA) (color.RGBA, bool) {
	out := blendChannels(s, d)
	var c [4]uint8
	for k, v := range out {
		if v != math.Trunc(v) {
			return color.RGBA{}, false
		}
		c[k] = uint8(v)
	}
	return color.RGBA{c[0], c[1], c[2], c[3]}, true
}

// blendVerdict compares an inexact blend with want: the GPU rounds it to a
// neighbouring integer, so only a channel at least 1.5 away proves the cell
// differs.
func blendVerdict(s, d, want color.RGBA) textsmooth.Verdict {
	if c, ok := blendExact(s, d); ok {
		return sameColour(c, want)
	}
	out := blendChannels(s, d)
	wc := [4]uint8{want.R, want.G, want.B, want.A}
	for k, v := range out {
		if math.Abs(v-float64(wc[k])) >= 1.5 {
			return textsmooth.Differs
		}
	}
	return textsmooth.Unknown
}
