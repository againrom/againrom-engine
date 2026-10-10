package text

import (
	"againrom/pkg/locale"
	"image"
	"image/color"
)

// FirstChar is the byte the first atlas record stands for: record k is character
// 32+k, so byte c selects record c-FirstChar. It is written here once and read
// only by index.
const FirstChar = 32

// MaxLevel is the largest intensity a glyph pixel can carry — the value that
// reproduces the caller's colour exactly. The ramp is colour*level/MaxLevel.
const MaxLevel = 15

// Pixel is one pixel of a glyph: a 4-bit intensity Level that is meaningful only
// when Painted is true.
//
// Paintedness is STRUCTURAL and carried per pixel — the atlas's own grammar
// leaves a cell's background unwritten with skip and blank-row ops rather than
// spending a level value on it, so no Level is reserved to mean transparent and
// LEVEL 0 IS A PAINTED PIXEL. Nothing shipped carries a level-0 pixel, which is
// what makes reading it as a hole invisible in data and wrong. This mirrors the
// decoder's own pixel rather than reusing it, because that package is a tier
// this one may not import; the loader converts.
type Pixel struct {
	Level   uint8
	Painted bool
}

// Glyph is one atlas record as this package draws it: a Width x Height grid of
// pixels in row-major order with row 0 at the top, plus THIS GLYPH'S OWN pen
// advance.
//
// Advance is the sidecar's value for this record and is NOT Width. The record is
// a fixed cell, strictly wider than the advance on every glyph of every shipped
// atlas, so cells overlap and that is the format's own arithmetic. The overlap
// is harmless because most of a cell is unpainted; a consumer that advanced by
// Width instead would draw legible, monospaced text at the wrong pitch.
//
// Pixels holds Width*Height elements. A grid shorter than that is not an error
// here — every read is bounded by both the declared cell and the slice — so a
// hand-built glyph can never paint outside its own cell or past its own data.
type Glyph struct {
	Width   int
	Height  int
	Pixels  []Pixel
	Advance int
	// CoverageLevels selects the .16a alpha ramp for the output overlay.
	// Native Draw keeps its raster contract; .16 glyphs keep opaque shades.
	CoverageLevels bool
}

// Font is a loaded font: the atlas's records in record order, the letter spacing
// added after every glyph, and the language selector every drawn byte is
// converted through.
//
// It is plain data. A loader outside this tier fills it; nothing here reads a
// file, opens an archive or validates that the records came from one atlas — and
// that is why Selector is a plain number rather than something this package
// discovers. WHICH language an install is comes off an entry in an archive, and
// this package opens nothing.
//
// The ZERO VALUE IS THE IDENTITY RULE, which is what makes the field safe to add
// to a type callers construct as a literal: a Font nobody set a selector on
// draws exactly what it drew before the field existed.
type Font struct {
	Glyphs   []Glyph
	Spacing  int
	Selector int
}

// SelectorConverting is the one language selector under which a byte is moved
// before it selects a record. Every other value — 0 included, and 0 is what an
// install that names no language takes — leaves every byte alone.
const SelectorConverting = 1

// Convert is the code-page pass the engine applies to EVERY string byte before
// the byte selects a glyph, and it is half of the index rule rather than a
// preprocessing step a caller may skip: the subscript is conv(b)-FirstChar, not
// b-FirstChar, and reading it as the latter is what makes an atlas look as
// though it holds no Cyrillic.
//
// Under SelectorConverting two blocks move up and nothing else changes:
//
//	0x80..0xAF  +0x30  ->  0xB0..0xDF
//	0xE0..0xEF  +0x10  ->  0xF0..0xFF
//
// Under every other selector the byte is returned untouched.
//
// NEITHER ADD CAN CARRY out of a byte (0xAF+0x30 = 0xDF, 0xEF+0x10 = 0xFF), but
// the map is NOT injective over the whole 256: 0xB0..0xDF and 0xF0..0xFF are
// themselves unmoved, so each of those 64 bytes shares a record with the byte
// that moves onto it — 0x80 and 0xB0 both select record 0xB0. It IS injective on
// ASCII plus the two moved blocks, which is the domain the shipped text was
// measured to occupy exhaustively, and 64-to-1 outside it. That is what lets a
// consumer treat the converted byte as the identity of a character it was
// actually handed, and no more than that.
//
// Both halves are pinned by TestConvertCollidesOnlyOutsideTheShippedAlphabet.
// The published claim's headline is sound for shipped data and its stated reason
// — that the images land where no unmoved byte sits — is false; this comment
// asserted that reason until 0066 T7.
//
// It is total: every byte value has an answer at every selector.
func Convert(b byte, selector int) byte {
	if !locale.FontRemaps(selector) {
		return b
	}
	switch {
	case b >= 0x80 && b <= 0xAF:
		return b + 0x30
	case b >= 0xE0 && b <= 0xEF:
		return b + 0x10
	}
	return b
}

// Height is the font's line height: the tallest record it holds, and 0 for a
// font with no records. Every glyph is drawn top-aligned inside it, so it is
// also the height of every box Measure reports for a non-empty string.
func (f *Font) Height() int {
	h := 0
	if f == nil {
		return 0
	}
	for i := range f.Glyphs {
		if f.Glyphs[i].Height > h {
			h = f.Glyphs[i].Height
		}
	}
	return h
}

// index is the record byte c selects, or -1 when the font holds no record at
// all. A byte below FirstChar, and one whose record lies at or past the record
// count, both select record 0 — the space — so no byte can index outside the
// list and an unrepresentable byte keeps a string's visible length instead of
// closing up. The empty-font test comes last so the fallback index can never be
// taken against an empty slice.
//
// THE CONVERSION HAPPENS HERE and nowhere else, which is what makes "every drawn
// byte is converted" a property of there being one subscript site rather than of
// three call sites remembering to. GlyphFor and walk both reach a record through
// this function, so a measurement and a draw of one string cannot disagree about
// which glyph a byte is — and a desynchronised pair is precisely the defect a
// second conversion site would produce.
//
// THE BOUND IS OURS AND IS DELIBERATE. The original applies none: it subtracts,
// widens to a byte and indexes, so a control byte reads records 224..255 of a
// 224-record table and dereferences whatever follows it. We fall back to the
// space instead. That is a divergence in the safe direction, and it is the
// reason this function is total where the thing it reproduces is not.
func (f *Font) index(c byte) int {
	if f == nil || len(f.Glyphs) == 0 {
		return -1
	}
	i := int(Convert(c, f.Selector)) - FirstChar
	if i < 0 || i >= len(f.Glyphs) {
		return 0
	}
	return i
}

// GlyphFor returns the glyph byte c selects, or nil when the font holds no
// record at all. It is total for every one of the 256 byte values.
func (f *Font) GlyphFor(c byte) *Glyph {
	i := f.index(c)
	if i < 0 {
		return nil
	}
	return &f.Glyphs[i]
}

// advance is how far the pen moves after drawing record i. The spacing is added
// after every glyph, the last one included; record 0 additionally carries half
// its own height, which is the engine's own rule for the space and the reason a
// fallback byte is a visible gap rather than a hairline.
//
// The result is clamped at 0. Spacing is a caller-set field on plain data and
// nothing validates it, so this keeps the pen monotonic: every placement lies at
// or right of the origin, which is what lets Measure report a box anchored there.
func (f *Font) advance(i int, g *Glyph) int {
	a := g.Advance + f.Spacing
	if i == 0 {
		a += g.Height / 2
	}
	if a < 0 {
		return 0
	}
	return a
}

// walk is THE placement rule, and the only one. It reads s as a byte string in
// the game's own arrangement — never as UTF-8, never transcoded — selects each
// byte's record, hands the glyph and the pen position its cell's left edge sits
// at to visit, and advances the pen. It returns the pen's final position.
//
// Measure and Draw are its only callers and neither adds an advance or a spacing
// of its own. Two copies of this arithmetic that drift apart is exactly how text
// comes to measure right in a test and draw wrong on screen.
//
// A byte selecting no record — possible only against a font with no records at
// all — contributes nothing and moves the pen not at all.
func (f *Font) walk(s string, visit func(g *Glyph, penX int)) int {
	pen := 0
	for i := 0; i < len(s); i++ {
		r := f.index(s[i])
		if r < 0 {
			continue
		}
		g := &f.Glyphs[r]
		if visit != nil {
			visit(g, pen)
		}
		pen += f.advance(r, g)
	}
	return pen
}

// inkRight is one past the rightmost painted column of g, and 0 for a glyph with
// no painted pixel. It reads the pixels rather than assuming ink fills the cell:
// a cell is padding on its right in the general case.
func (g *Glyph) inkRight() int {
	right := 0
	for row := 0; row < g.Height; row++ {
		for col := g.Width - 1; col >= 0; col-- {
			i := row*g.Width + col
			if i < 0 || i >= len(g.Pixels) {
				continue
			}
			if g.Pixels[i].Painted {
				if col+1 > right {
					right = col + 1
				}
				break
			}
		}
	}
	return right
}

// Advance is where the pen ends after s — where a following run of text starts.
// It includes the spacing after the last glyph, as the engine's own layout does.
//
// It is reported apart from Measure's box because the two are normally different
// numbers: a glyph whose art runs wider than its advance widens the box past the
// pen. A caller placing a value after a label wants this one, and recovering it
// from the box would mean a second copy of the pen.
func (f *Font) Advance(s string) int { return f.walk(s, nil) }

// Measure reports the box (w, h) that contains every pixel Draw paints for the
// same font and string, anchored at Draw's own position.
//
// w is the larger of Advance and one past the rightmost painted pixel; the two
// differ in both directions, since a trailing space carries the pen past all ink.
// h is the font's line height for a non-empty string and 0 for the empty one, so
// vertical containment holds by construction: every glyph is top-aligned and
// reads only inside its own cell, and no cell is taller than the tallest.
//
// It is the walk's second reading, so it cannot disagree with the first about
// which glyph a byte selects or where the pen puts it.
func (f *Font) Measure(s string) (w, h int) {
	right := 0
	pen := f.walk(s, func(g *Glyph, penX int) {
		if r := penX + g.inkRight(); r > right {
			right = r
		}
	})
	w = pen
	if right > w {
		w = right
	}
	if len(s) == 0 {
		return 0, 0
	}
	return w, f.Height()
}

// Draw paints s into dst with the text box's top-left corner at (x, y) in dst's
// own coordinates, in colour c.
//
// A painted pixel of level v is written as c with each of R, G and B scaled by
// v/MaxLevel — truncating integer division, so level MaxLevel reproduces c
// exactly — and c's alpha written through unscaled. Scaling the alpha too would
// fade and darken at once, and image.RGBA is alpha-premultiplied, so scaling RGB
// alone is what keeps a valid colour valid. The write REPLACES the destination:
// nothing here reads a destination pixel and there is no blend, which is the
// blitter's own behaviour. An unpainted pixel writes nothing at all.
//
// Bytes are drawn left to right, so where two cells overlap the later glyph's
// painted pixels win. Everything outside dst's bounds is clipped away — by
// SetRGBA's own bounds test and by no rectangle arithmetic of ours, which is
// deliberate: a per-glyph rejection test would be the one branch here with no
// counterpart in Measure, and an off-by-one in it drops a whole glyph. A nil
// destination draws nothing.
//
// Inside an active capture window (SetCapture, capture.go) every placed
// glyph is ALSO recorded into Captured before it is (or is not) rasterised —
// the presentation layer's smoothed-text overlay reads that list. Outside one
// this adds one boolean test per glyph and changes nothing else.
func (f *Font) Draw(dst *image.RGBA, s string, x, y int, c color.RGBA) {
	if dst == nil {
		return
	}
	record := capturing && !dst.Bounds().Empty()
	f.walk(s, func(g *Glyph, penX int) {
		gx := x + penX
		auditGlyph(g, record)
		if record {
			call := DrawCall{Glyph: g, X: gx, Y: y, Color: c, Clip: dst.Bounds()}
			if !skipRaster {
				call.Under = recordUnder(dst, g, gx, y)
			}
			captured = append(captured, call)
			if skipRaster {
				return
			}
		}
		for row := 0; row < g.Height; row++ {
			for col := 0; col < g.Width; col++ {
				i := row*g.Width + col
				if i >= len(g.Pixels) {
					return
				}
				p := g.Pixels[i]
				if !p.Painted {
					continue
				}
				dst.SetRGBA(gx+col, y+row, Shade(c, p.Level))
			}
		}
	})
}

// DrawFlat paints s as Draw places it, but every painted pixel takes c
// itself, whatever its level: a ramp that holds one colour in each of its
// entries. Like Draw it replaces the destination, clips at dst's bounds and
// records each glyph in a capture window, so the smoothed-text overlay
// redraws a flat shadow together with the face drawn over it.
func (f *Font) DrawFlat(dst *image.RGBA, s string, x, y int, c color.RGBA) {
	if dst == nil {
		return
	}
	record := capturing && !dst.Bounds().Empty()
	f.walk(s, func(g *Glyph, penX int) {
		gx := x + penX
		auditGlyph(g, record)
		if record {
			call := DrawCall{Glyph: g, X: gx, Y: y, Color: c, Flat: true, Clip: dst.Bounds()}
			if !skipRaster {
				call.Under = recordUnder(dst, g, gx, y)
			}
			captured = append(captured, call)
			if skipRaster {
				return
			}
		}
		for row := 0; row < g.Height; row++ {
			for col := 0; col < g.Width; col++ {
				i := row*g.Width + col
				if i >= len(g.Pixels) {
					return
				}
				if g.Pixels[i].Painted {
					dst.SetRGBA(gx+col, y+row, c)
				}
			}
		}
	})
}

// PaintedAt reports whether Draw paints at in a draw of s at (x, y). It uses
// the same byte conversion, glyph placement and structural painted-pixel bit
// as Draw. The colour and pixel level do not affect coverage: every painted
// pixel Draw writes has the caller's full alpha, including a painted level-0
// pixel.
func (f *Font) PaintedAt(s string, x, y int, at image.Point) bool {
	painted := false
	f.walk(s, func(g *Glyph, penX int) {
		if painted {
			return
		}
		row := at.Y - y
		col := at.X - (x + penX)
		if row < 0 || row >= g.Height || col < 0 || col >= g.Width {
			return
		}
		i := row*g.Width + col
		if i >= 0 && i < len(g.Pixels) && g.Pixels[i].Painted {
			painted = true
		}
	})
	return painted
}

// Shade is the ramp a glyph's 4-bit value indexes: the caller's colour scaled to
// level/MaxLevel in R, G and B, with the alpha carried through. A level above
// MaxLevel is clamped, so no input can brighten the colour past the one the
// caller asked for.
func Shade(c color.RGBA, level uint8) color.RGBA {
	if level > MaxLevel {
		level = MaxLevel
	}
	scale := func(v uint8) uint8 { return uint8(int(v) * int(level) / MaxLevel) }
	return color.RGBA{R: scale(c.R), G: scale(c.G), B: scale(c.B), A: c.A}
}
