package text

import (
	"fmt"
	"image"
	"image/color"
	"testing"
)

// blank builds a w x h glyph with the given advance and no painted pixel, so a
// box it takes part in is decided by the pen alone.
func blank(w, h, advance int) Glyph {
	return Glyph{Width: w, Height: h, Pixels: make([]Pixel, w*h), Advance: advance}
}

// inked builds a w x h glyph whose pixels are painted where ink says so, at the
// level it returns.
func inked(w, h, advance int, ink func(x, y int) (uint8, bool)) Glyph {
	g := blank(w, h, advance)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if lv, on := ink(x, y); on {
				g.Pixels[y*w+x] = Pixel{Level: lv, Painted: true}
			}
		}
	}
	return g
}

// penFont is the worked example the contract's advance rule is stated on: 224
// records, spacing 2, record 0 a 16x15 space of advance 0, 'A' advance 9, 'B'
// advance 8, every record blank so a measured width is exactly the pen.
func penFont() *Font {
	f := &Font{Spacing: 2, Glyphs: make([]Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = blank(16, 15, 1)
	}
	f.Glyphs[0] = blank(16, 15, 0)
	f.Glyphs['A'-FirstChar] = blank(16, 15, 9)
	f.Glyphs['B'-FirstChar] = blank(16, 15, 8)
	return f
}

func TestGlyphForIsTotal(t *testing.T) {
	full := penFont()
	for c := 0; c < 256; c++ {
		want := c - FirstChar
		if want < 0 || want >= 224 {
			want = 0
		}
		if got := full.index(byte(c)); got != want {
			t.Fatalf("byte %#02x selected record %d, want %d", c, got, want)
		}
		if full.GlyphFor(byte(c)) == nil {
			t.Fatalf("byte %#02x selected no glyph in a 224-record font", c)
		}
	}

	// font3's shape: 64 records, so characters past 0x5F have none and fall back
	// to the space rather than indexing past the list.
	short := &Font{Spacing: 2, Glyphs: make([]Glyph, 64)}
	for i := range short.Glyphs {
		short.Glyphs[i] = blank(8, 6, 3)
	}
	for c := 0; c < 256; c++ {
		got := short.index(byte(c))
		if got < 0 || got >= 64 {
			t.Fatalf("byte %#02x selected record %d, outside a 64-record font", c, got)
		}
		if c >= FirstChar && c < FirstChar+64 {
			if got != c-FirstChar {
				t.Fatalf("byte %#02x selected record %d, want %d", c, got, c-FirstChar)
			}
			continue
		}
		if got != 0 {
			t.Fatalf("byte %#02x has no record and selected %d, want the space", c, got)
		}
	}

	// A font with no records refuses no byte and selects nothing.
	empty := &Font{Spacing: 2}
	for c := 0; c < 256; c++ {
		if g := empty.GlyphFor(byte(c)); g != nil {
			t.Fatalf("byte %#02x selected %v in a font with no records", c, g)
		}
	}
	if w, h := empty.Measure("hello"); w != 0 || h != 0 {
		t.Fatalf("empty font measured (%d,%d), want (0,0)", w, h)
	}
}

// AC-4: the pen sums, glyph by glyph, and the space's extra half-height term.
func TestMeasurePen(t *testing.T) {
	f := penFont()

	// advance('A') = 9+2 = 11, advance('B') = 8+2 = 10, advance(space) =
	// 0+2+15/2 = 9. Integer division: 15/2 is 7.
	const wantAB = 11 + 10
	const wantASpaceB = 11 + 9 + 10

	if w, h := f.Measure("AB"); w != wantAB || h != 15 {
		t.Fatalf(`Measure("AB") = (%d,%d), want (%d,15)`, w, h, wantAB)
	}
	if w, _ := f.Measure("A B"); w != wantASpaceB {
		t.Fatalf(`Measure("A B") = %d, want %d`, w, wantASpaceB)
	}
	if got := wantASpaceB - wantAB; got != 9 {
		t.Fatalf("a space cost %d, want 9", got)
	}
	if w, h := f.Measure(""); w != 0 || h != 0 {
		t.Fatalf(`Measure("") = (%d,%d), want (0,0)`, w, h)
	}

	// The half-height term belongs to record 0 alone. Halving record 0's height
	// halves it; changing any other record's height does not move it.
	tall := penFont()
	tall.Glyphs[0].Height = 30
	if w, _ := tall.Measure("A B"); w != wantASpaceB+8 {
		t.Fatalf(`a taller space measured %d, want %d`, w, wantASpaceB+8)
	}
	other := penFont()
	other.Glyphs['A'-FirstChar].Height = 30
	if w, _ := other.Measure("A B"); w != wantASpaceB {
		t.Fatalf(`a taller 'A' moved the space term: %d, want %d`, w, wantASpaceB)
	}
}

// AC-5: a byte with no record costs exactly one space.
func TestMeasureFallbackCostsASpace(t *testing.T) {
	f := penFont()
	base, _ := f.Measure("AB")
	spaced, _ := f.Measure("A B")
	spaceCost := spaced - base

	for _, s := range []string{"A\x00B", "A\nB", "A\x1fB"} {
		got, _ := f.Measure(s)
		if got != base+spaceCost {
			t.Fatalf("Measure(%q) = %d, want %d — a byte with no record costs one space", s, got, base+spaceCost)
		}
	}

	// The same, past the top of a short atlas: 0xFE has no record in 64.
	short := &Font{Spacing: 2, Glyphs: make([]Glyph, 64)}
	for i := range short.Glyphs {
		short.Glyphs[i] = blank(8, 6, 3)
	}
	short.Glyphs[0] = blank(8, 6, 0)
	one, _ := short.Measure("\xfe")
	spaceOnly, _ := short.Measure(" ")
	if one != spaceOnly {
		t.Fatalf(`Measure("\xfe") = %d, Measure(" ") = %d — want them equal`, one, spaceOnly)
	}
}

// AC-6: the box holds ink that runs wider than the advance, and is never
// narrower than the pen.
func TestMeasureBoxHoldsWideInk(t *testing.T) {
	// A 16-wide glyph of advance 3 that paints its whole width: the ink runs 13
	// px past where the pen leaves it.
	f := &Font{Spacing: 2, Glyphs: make([]Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = blank(16, 15, 0)
	}
	solid := inked(16, 15, 3, func(x, y int) (uint8, bool) { return 15, true })
	f.Glyphs['W'-FirstChar] = solid

	pen := f.walk("WW", nil)
	w, h := f.Measure("WW")
	if w <= pen {
		t.Fatalf("Measure(\"WW\") = %d with the pen at %d; the ink should have widened the box", w, pen)
	}
	if want := (3 + 2) + 16; w != want {
		t.Fatalf("Measure(\"WW\") = %d, want %d (second cell placed at the pen, 16 wide)", w, want)
	}
	if h != 15 {
		t.Fatalf("Measure(\"WW\") height = %d, want 15", h)
	}

	// And the other direction: a trailing space pushes the pen past all ink, and
	// the box follows the pen.
	wSpace, _ := f.Measure("W ")
	penSpace := f.walk("W ", nil)
	if wSpace < penSpace {
		t.Fatalf(`Measure("W ") = %d is narrower than its pen %d`, wSpace, penSpace)
	}
}

// AC-13: the box's height, and the advance reported apart from it.
func TestHeightAndAdvance(t *testing.T) {
	f := &Font{Spacing: 2, Glyphs: make([]Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = blank(16, 9, 4)
	}
	f.Glyphs[0] = blank(16, 15, 0)
	f.Glyphs['T'-FirstChar] = blank(16, 21, 4) // the tallest record in the font

	for _, s := range []string{"a", "ab", " ", "\x01", "hello world"} {
		w, h := f.Measure(s)
		if h != 21 {
			t.Fatalf("Measure(%q) height = %d, want the tallest record's 21", s, h)
		}
		if got := f.Advance(s); got != w {
			t.Fatalf("Measure(%q) width %d, Advance %d — no ink, so they should agree", s, w, got)
		}
	}
	if w, h := f.Measure(""); w != 0 || h != 0 {
		t.Fatalf(`Measure("") = (%d,%d), want (0,0)`, w, h)
	}
	if got := f.Advance(""); got != 0 {
		t.Fatalf(`Advance("") = %d, want 0`, got)
	}

	// With ink past the advance the two part company, and the box is the wider.
	f.Glyphs['a'-FirstChar] = inked(16, 9, 4, func(x, y int) (uint8, bool) { return 15, true })
	w, _ := f.Measure("a")
	adv := f.Advance("a")
	if adv != 6 || w != 16 {
		t.Fatalf("Measure(\"a\") = %d and Advance = %d, want 16 and 6", w, adv)
	}
}

// AC-14: the high half reaches its own records. A string is walked BY BYTE, so
// 0xFE selects record 222; decoding it as UTF-8 would make it the replacement
// rune and select record 221 instead.
func TestHighBytesSelectTheirOwnRecords(t *testing.T) {
	f := &Font{Spacing: 0, Glyphs: make([]Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = blank(16, 15, 0)
	}
	// Distinct advances on the three records the two readings could confuse.
	f.Glyphs[221] = blank(16, 15, 100) // 0xFD
	f.Glyphs[222] = blank(16, 15, 20)  // 0xFE
	f.Glyphs[223] = blank(16, 15, 3)   // 0xFF

	for c, want := range map[byte]int{0xfd: 221, 0xfe: 222, 0xff: 223, 0x80: 96, 0xc3: 163} {
		if got := f.index(c); got != want {
			t.Fatalf("byte %#02x selected record %d, want %d", c, got, want)
		}
	}
	if got := f.Advance("\xfe"); got != 20 {
		t.Fatalf(`Advance("\xfe") = %d, want 20 — a byte-indexed walk selects record 222`, got)
	}
	if got := f.Advance("\xfe\xff\xfd"); got != 20+3+100 {
		t.Fatalf(`Advance of three high bytes = %d, want %d`, got, 20+3+100)
	}
	// Two bytes that happen to form one UTF-8 rune must still be two glyphs.
	if got := f.Advance("\xc3\xa9"); got != f.Advance("\xc3")+f.Advance("\xa9") {
		t.Fatalf("a two-byte UTF-8 sequence measured %d, not its two bytes' sum", got)
	}
}

func TestHeightIsTheTallestRecord(t *testing.T) {
	if (&Font{}).Height() != 0 {
		t.Fatal("a font with no records has a non-zero height")
	}
	f := &Font{Spacing: 2, Glyphs: []Glyph{blank(8, 6, 1), blank(8, 11, 1), blank(8, 4, 1)}}
	if got := f.Height(); got != 11 {
		t.Fatalf("Height() = %d, want 11", got)
	}
}

func TestDrawnPixelsLieInsideTheMeasuredBox(t *testing.T) {
	f := &Font{Spacing: 2, Glyphs: make([]Glyph, 224)}
	for i := range f.Glyphs {
		switch i % 4 {
		case 0:
			f.Glyphs[i] = blank(16, 15, i%7)
		case 1:
			f.Glyphs[i] = inked(16, 15, 4, func(x, y int) (uint8, bool) { return uint8(1 + x%15), x < 10 })
		case 2: // ink wider than the advance, and down to the last row
			f.Glyphs[i] = inked(16, 15, 2, func(x, y int) (uint8, bool) { return 15, true })
		case 3:
			f.Glyphs[i] = inked(12, 9, 11, func(x, y int) (uint8, bool) { return 8, x == y })
		}
	}
	f.Glyphs[0] = blank(16, 15, 0)

	// A deterministic generator: no clock, no global source.
	seed := uint32(2463534242)
	next := func() byte {
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		return byte(seed)
	}

	const margin = 40
	for trial := 0; trial < 300; trial++ {
		n := int(next()%12) + 1
		buf := make([]byte, n)
		for i := range buf {
			buf[i] = next()
		}
		s := string(buf)

		w, h := f.Measure(s)
		dst := image.NewRGBA(image.Rect(-margin, -margin, w+margin, h+margin))
		f.Draw(dst, s, 0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})

		b := dst.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if _, _, _, a := dst.At(x, y).RGBA(); a == 0 {
					continue
				}
				if x < 0 || y < 0 || x >= w || y >= h {
					t.Fatalf("trial %d, %q: painted (%d,%d) outside the measured box %dx%d",
						trial, fmt.Sprintf("%x", buf), x, y, w, h)
				}
			}
		}
	}
}

// PaintedAt is the draw/hit coverage seam. This comparison enumerates every
// point in dst's bounds, including points outside the text and glyph extents.
// It also includes overlapping cells, blank pixels and a painted level-0 pixel.
func TestPaintedAtMatchesDrawCoverage(t *testing.T) {
	f := &Font{Spacing: 1, Glyphs: make([]Glyph, 224)}
	for i := range f.Glyphs {
		f.Glyphs[i] = blank(5, 4, 3)
	}
	f.Glyphs['A'-FirstChar] = inked(5, 4, 3, func(x, y int) (uint8, bool) {
		if x == 1 && y == 1 {
			return 0, true
		}
		return MaxLevel, x == 4 && y == 2
	})
	f.Glyphs['B'-FirstChar] = inked(5, 4, 2, func(x, y int) (uint8, bool) {
		return 7, x == 0 && y == 2
	})

	const x0, y0 = 7, 9
	dst := image.NewRGBA(image.Rect(-3, -2, 24, 20))
	f.Draw(dst, "AB", x0, y0, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	for y := dst.Bounds().Min.Y; y < dst.Bounds().Max.Y; y++ {
		for x := dst.Bounds().Min.X; x < dst.Bounds().Max.X; x++ {
			at := image.Pt(x, y)
			_, _, _, alpha := dst.At(x, y).RGBA()
			if got, want := f.PaintedAt("AB", x0, y0, at), alpha != 0; got != want {
				t.Fatalf("PaintedAt(%v) = %v, Draw alpha coverage = %v", at, got, want)
			}
		}
	}
}
