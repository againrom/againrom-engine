package text_test

import (
	"testing"

	"againrom/pkg/render/text"
)

// The conversion is half of the index rule, so these pin the rule and not the
// arithmetic: what is checked is which RECORD a byte reaches, and the identity
// case is checked over the whole byte range rather than at a few points, because
// its whole value is that it changed nothing.

func TestConvertIsIdentityAtEverySelectorButOne(t *testing.T) {
	for sel := -1; sel <= 9; sel++ {
		if sel == text.SelectorConverting {
			continue
		}
		for b := 0; b < 256; b++ {
			if got := text.Convert(byte(b), sel); got != byte(b) {
				t.Fatalf("selector %d: Convert(%#02x) = %#02x, want it untouched", sel, b, got)
			}
		}
	}
}

func TestConvertMovesExactlyTwoBlocks(t *testing.T) {
	for b := 0; b < 256; b++ {
		got := text.Convert(byte(b), text.SelectorConverting)
		var want byte
		switch {
		case b >= 0x80 && b <= 0xAF:
			want = byte(b + 0x30)
		case b >= 0xE0 && b <= 0xEF:
			want = byte(b + 0x10)
		default:
			want = byte(b)
		}
		if got != want {
			t.Fatalf("Convert(%#02x) = %#02x, want %#02x", b, got, want)
		}
	}
}

// THE CONVERTER IS NOT INJECTIVE OVER ALL 256 BYTES, and this pins both halves
// of that because the published headline says otherwise.
//
// The reasoning behind the published claim is that the two moved blocks land on
// 0xB0..0xDF and 0xF0..0xFF, "which no unmoved byte occupies". That is not so:
// those two ranges are themselves unmoved, so each of their 64 bytes collides
// with the byte that moves onto it — 0x80 and 0xB0 both select record 0xB0.
//
// What IS true, and what the whole scheme actually rests on, is injectivity over
// the domain the shipped text occupies: every high byte of the Russian corpus
// was measured to lie in one of the two moved blocks and none anywhere else. So
// the map is injective on ASCII plus those two blocks, which is the set a
// consumer will ever hand it, and it is 64-to-1 outside it.
//
// Both are pinned rather than only the convenient one: a later change that made
// the map injective everywhere would be a change to what the engine does, and it
// should have to edit a test that says so.
func TestConvertCollidesOnlyOutsideTheShippedAlphabet(t *testing.T) {
	// shipped is ASCII plus the two blocks the Russian corpus was measured to
	// use, and nothing else.
	shipped := func(b int) bool {
		return b < 0x80 || (b >= 0x80 && b <= 0xAF) || (b >= 0xE0 && b <= 0xEF)
	}

	seen := make(map[byte]int, 256)
	for b := 0; b < 256; b++ {
		if !shipped(b) {
			continue
		}
		got := text.Convert(byte(b), text.SelectorConverting)
		if prev, ok := seen[got]; ok {
			t.Fatalf("shipped alphabet: %#02x and %#02x both convert to %#02x", prev, b, got)
		}
		seen[got] = b
	}

	collisions := 0
	all := make(map[byte]bool, 256)
	for b := 0; b < 256; b++ {
		got := text.Convert(byte(b), text.SelectorConverting)
		if all[got] {
			collisions++
		}
		all[got] = true
	}
	if collisions != 64 {
		t.Fatalf("collisions over the whole byte range = %d, want 64 — the two "+
			"unmoved ranges 0xB0..0xDF and 0xF0..0xFF each receive one moved byte", collisions)
	}
}

// The two moved blocks land on 0xB0..0xDF and 0xF0..0xFF, and the claim that
// makes the atlases fit is that those cells are exactly the converter's image.
func TestConvertImageIsTheHighHalfCyrillicBlock(t *testing.T) {
	image := map[byte]bool{}
	for b := 0x80; b <= 0xFF; b++ {
		image[text.Convert(byte(b), text.SelectorConverting)] = true
	}
	for c := 0xB0; c <= 0xDF; c++ {
		if !image[byte(c)] {
			t.Fatalf("record cell %#02x is not in the converter's image", c)
		}
	}
	for c := 0xF0; c <= 0xFF; c++ {
		if !image[byte(c)] {
			t.Fatalf("record cell %#02x is not in the converter's image", c)
		}
	}
}

// atlas builds a font of n records whose k-th glyph is identifiable by its
// advance, so a test can say which record a byte reached without inspecting
// pixels. The height is even and non-zero so that record 0 — the space, which
// carries half its own height — still moves the pen, which is what makes the
// fallback a visible gap rather than nothing.
func atlas(n int) *text.Font {
	f := &text.Font{Spacing: 0, Glyphs: make([]text.Glyph, n)}
	for i := range f.Glyphs {
		f.Glyphs[i] = text.Glyph{Width: 1, Height: 4, Pixels: []text.Pixel{{Painted: false}}, Advance: i}
	}
	return f
}

// record reports which glyph byte c selected, read back through the advance
// each record was built to carry.
func record(t *testing.T, f *text.Font, c byte) int {
	t.Helper()
	g := f.GlyphFor(c)
	if g == nil {
		t.Fatalf("font holds no record for %#02x", c)
	}
	return g.Advance
}

func TestSelectorReachesTheGlyphChoice(t *testing.T) {
	const records = 224 // font1's own count: characters 0x20..0xFF
	plain, ru := atlas(records), atlas(records)
	ru.Selector = text.SelectorConverting

	// A Russian byte in the first moved block reaches a DIFFERENT record under
	// the two selectors, and under selector 1 it reaches the converted one.
	const b = 0x90
	if got, want := record(t, ru, b), int(text.Convert(b, text.SelectorConverting))-text.FirstChar; got != want {
		t.Fatalf("selector 1: byte %#02x reached record %d, want %d", b, got, want)
	}
	if got, want := record(t, plain, b), b-text.FirstChar; got != want {
		t.Fatalf("selector 0: byte %#02x reached record %d, want %d", b, got, want)
	}
	if record(t, ru, b) == record(t, plain, b) {
		t.Fatal("the selector changed nothing for a byte in a moved block")
	}

	// An ASCII byte is untouched by both.
	for _, c := range []byte{' ', 'A', 'z', 0x7F} {
		if got, want := record(t, ru, c), record(t, plain, c); got != want {
			t.Fatalf("byte %#02x moved under selector 1: %d != %d", c, got, want)
		}
	}
}

// The bound is ours and it is what makes the whole path total. The original has
// none and reads past its table; every byte here must land inside the font.
func TestEveryByteLandsInsideTheFontAtEitherSelector(t *testing.T) {
	for _, sel := range []int{0, text.SelectorConverting} {
		for _, n := range []int{64, 224} {
			f := atlas(n)
			f.Selector = sel
			for b := 0; b < 256; b++ {
				g := f.GlyphFor(byte(b))
				if g == nil {
					t.Fatalf("selector %d, %d records: no glyph for %#02x", sel, n, b)
				}
				if g.Advance < 0 || g.Advance >= n {
					t.Fatalf("selector %d: byte %#02x reached record %d of %d", sel, b, g.Advance, n)
				}
			}
			// A byte whose record lies past the count falls back to record 0,
			// and the pen still moves: the string keeps its visible length.
			if got := record(t, f, 0xFF); n == 64 && got != 0 {
				t.Fatalf("%d records: byte 0xff reached record %d, want the fallback 0", n, got)
			}
			if w := f.Advance("\x01"); w <= 0 {
				t.Fatalf("selector %d: a control byte advanced the pen by %d, want a visible gap", sel, w)
			}
		}
	}
}

// The measurement and the draw must agree about which glyph a byte is, or a
// wrapped line would be measured against one record and painted with another.
func TestMeasureAndDrawAgreeUnderTheConverter(t *testing.T) {
	f := atlas(224)
	f.Selector = text.SelectorConverting
	s := "\x90\x91\xe0A "
	w, _ := f.Measure(s)
	sum := 0
	for i := 0; i < len(s); i++ {
		sum += record(t, f, s[i])
		if record(t, f, s[i]) == 0 {
			sum += f.Glyphs[0].Height / 2
		}
	}
	if f.Advance(s) != sum {
		t.Fatalf("Advance = %d, want the sum of the selected records' advances %d", f.Advance(s), sum)
	}
	if w <= 0 {
		t.Fatalf("Measure reported width %d for a non-empty string", w)
	}
}
