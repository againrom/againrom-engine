package spr16

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func u16(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

// words concatenates u16 values in LE order — the .16a RLE block builder.
func words(vs ...uint16) []byte {
	var out []byte
	for _, v := range vs {
		out = append(out, u16(v)...)
	}
	return out
}

// Control words: op in bits 14–15, a 14-bit count below.
func litA(n uint16) uint16   { return n }          // 0b00: literal, n pixel words follow
func rowsA(n uint16) uint16  { return 0x4000 | n } // 0b01: blank rows
func skipA(n uint16) uint16  { return 0x8000 | n } // 0b10: skip
func aliasA(n uint16) uint16 { return 0xC000 | n } // 0b11: decoded as blank rows

// pixA builds a literal pixel word: index in bits 1–8, level in bits 9–12.
func pixA(index, level uint8) uint16 {
	return uint16(index)<<1 | uint16(level)<<9
}

// paletteA builds the 1024-byte palette region; entry i of entries is the
// on-disk [B, G, R, x] quad. Unset entries stay zero.
func paletteA(entries map[int][4]byte) []byte {
	p := make([]byte, 1024)
	for i, e := range entries {
		copy(p[i*4:], e[:])
	}
	return p
}

// pa is a painted pixel; the transparent pixel is the PixelA zero value.
func pa(index, level uint8) PixelA { return PixelA{Index: index, Level: level, Painted: true} }

var trA = PixelA{}

// --- assertion helpers ---

func decodeAOK(t *testing.T, name string, data []byte, palette bool) *SpriteA {
	t.Helper()
	s, err := DecodeA(data, palette)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	if s == nil {
		t.Fatalf("%s: nil *SpriteA without error", name)
	}
	return s
}

func assertFrameA(t *testing.T, name string, got FrameA, width, height int, want []PixelA) {
	t.Helper()
	if got.Width != width || got.Height != height {
		t.Fatalf("%s: got %dx%d, want %dx%d", name, got.Width, got.Height, width, height)
	}
	if got.Pixels == nil {
		t.Fatalf("%s: nil Pixels", name)
	}
	if len(got.Pixels) != len(want) {
		t.Fatalf("%s: got %d pixels, want %d", name, len(got.Pixels), len(want))
	}
	for i := range want {
		if got.Pixels[i] != want[i] {
			t.Fatalf("%s: pixel %d = %+v, want %+v", name, i, got.Pixels[i], want[i])
		}
	}
}

// --- well-formed decodes ---

// TestDecodeAMultiFrame — SC-2 (AC-1): a two-frame stream with a declared
// palette, exercising literal, skip and blank-row ops. The blank rows sit
// mid-frame on a 3-wide grid, so a skip-miswired arm lands every later pixel
// wrong; frame 1 carries the spec's numeric anchor — the bytes 27 82
// (0x8227, op 0b10, n = 0x227) skipping 551 pixels of a 552-wide frame, a
// count above 255 with bits 8–13 set.
func TestDecodeAMultiFrame(t *testing.T) {
	block0 := words(
		litA(2), pixA(10, 3), pixA(20, 7), // (0,0), (1,0)
		skipA(1),                                        // (2,0) stays transparent
		rowsA(2),                                        // rows 1 and 2 blank
		litA(3), pixA(30, 1), pixA(40, 15), pixA(50, 0), // row 3
	)
	grid0 := []PixelA{
		pa(10, 3), pa(20, 7), trA,
		trA, trA, trA,
		trA, trA, trA,
		pa(30, 1), pa(40, 15), pa(50, 0),
	}
	block1 := concat([]byte{0x27, 0x82}, words(litA(1), pixA(9, 5)))
	grid1 := make([]PixelA, 552)
	grid1[551] = pa(9, 5)

	stream := concat(
		paletteA(map[int][4]byte{0: {1, 2, 3, 0}, 5: {10, 20, 30, 99}, 255: {0xFF, 0xEE, 0xDD, 4}}),
		frameRecord(3, 4, block0),
		frameRecord(552, 1, block1),
		trailerBytes(2, false),
	)
	s := decodeAOK(t, "two-frame", stream, true)

	if len(s.Frames) != 2 {
		t.Fatalf("frame count = %d, want 2", len(s.Frames))
	}
	assertFrameA(t, "frame 0", s.Frames[0], 3, 4, grid0)
	assertFrameA(t, "frame 1", s.Frames[1], 552, 1, grid1)

	if len(s.Palette) != 256 {
		t.Fatalf("palette has %d entries, want 256", len(s.Palette))
	}
	for i, want := range map[int]Color{
		0:   {R: 3, G: 2, B: 1},
		5:   {R: 30, G: 20, B: 10},
		255: {R: 0xDD, G: 0xEE, B: 0xFF},
		7:   {},
	} {
		if s.Palette[i] != want {
			t.Fatalf("palette[%d] = %+v, want %+v", i, s.Palette[i], want)
		}
	}
}

// TestDecodeASpecExample — SC-2: the spec's printed .16a I/O example,
// block bytes verbatim, decodes to its printed grid.
func TestDecodeASpecExample(t *testing.T) {
	block := []byte{
		0x02, 0x00, // literal n=2
		0x0A, 0x1E, // ss=0x1E0A → index 5, level 15   at (0,0)
		0x0C, 0x10, // ss=0x100C → index 6, level 8    at (1,0)
		0x01, 0x80, // skip n=1  → (0,1) stays transparent
		0x01, 0x00, // literal n=1
		0x0E, 0x1E, // ss=0x1E0E → index 7, level 15   at (1,1)
	}
	s := decodeAOK(t, "spec example",
		concat(frameRecord(2, 2, block), trailerBytes(1, false)), false)
	if len(s.Frames) != 1 {
		t.Fatalf("frame count = %d, want 1", len(s.Frames))
	}
	if s.Palette != nil {
		t.Fatalf("undeclared palette came back non-nil: %d entries", len(s.Palette))
	}
	assertFrameA(t, "spec example", s.Frames[0], 2, 2, []PixelA{
		pa(5, 15), pa(6, 8),
		trA, pa(7, 15),
	})
}

// TestDecodeAPreservesRawFields — SC-2 (AC-10): literal words spanning
// index 0–255 and level 0–15, level 0 included; each painted pixel
// preserves its exact {index, level}, a painted level 0 is distinguishable
// from a transparent pixel, and the masked bits 0 and 13–15 change
// nothing.
func TestDecodeAPreservesRawFields(t *testing.T) {
	block := words(
		litA(5),
		pixA(0, 0),
		pixA(255, 15),
		pixA(255, 0),
		pixA(0, 15),
		pixA(77, 5)|0xE001, // bits 0 and 13–15 set: masked off, {77, 5} kept
	)
	s := decodeAOK(t, "raw fields",
		concat(frameRecord(3, 2, block), trailerBytes(1, false)), false)
	assertFrameA(t, "raw fields", s.Frames[0], 3, 2, []PixelA{
		pa(0, 0), pa(255, 15), pa(255, 0),
		pa(0, 15), pa(77, 5), trA,
	})
	if s.Frames[0].Pixels[0] == trA {
		t.Fatal("a painted {0, 0} pixel compares equal to a transparent one")
	}
}

// TestDecodeAAliasBlankRows — SC-2 (AC-13): the same program written with
// op 0b11 decodes identically to op 0b01 — and a 0b11-as-skip miswiring
// would land the trailing literals two rows early, so equality
// discriminates.
func TestDecodeAAliasBlankRows(t *testing.T) {
	prog := func(rows uint16) []byte {
		return words(litA(1), pixA(1, 1), skipA(1), rows, litA(2), pixA(2, 2), pixA(3, 3))
	}
	want := []PixelA{
		pa(1, 1), trA,
		trA, trA,
		pa(2, 2), pa(3, 3),
	}

	sRows := decodeAOK(t, "op 0b01",
		concat(frameRecord(2, 3, prog(rowsA(1))), trailerBytes(1, false)), false)
	sAlias := decodeAOK(t, "op 0b11",
		concat(frameRecord(2, 3, prog(aliasA(1))), trailerBytes(1, false)), false)

	if !reflect.DeepEqual(sRows, sAlias) {
		t.Fatalf("op 0b11 decoded unlike op 0b01:\n%+v\nvs\n%+v", sAlias, sRows)
	}
	assertFrameA(t, "blank-row program", sRows.Frames[0], 2, 3, want)
}

// TestDecodeACountZero — SC-2 (AC-14): a 0-count trailer decodes to an
// empty, non-nil frame list, the palette still returned when declared.
func TestDecodeACountZero(t *testing.T) {
	withPal := decodeAOK(t, "count 0, palette declared",
		concat(paletteA(map[int][4]byte{1: {9, 8, 7, 0}}), trailerBytes(0, false)), true)
	if withPal.Frames == nil || len(withPal.Frames) != 0 {
		t.Fatalf("frames = %v, want empty and non-nil", withPal.Frames)
	}
	if len(withPal.Palette) != 256 {
		t.Fatalf("palette has %d entries, want 256", len(withPal.Palette))
	}
	if want := (Color{R: 7, G: 8, B: 9}); withPal.Palette[1] != want {
		t.Fatalf("palette[1] = %+v, want %+v", withPal.Palette[1], want)
	}

	noPal := decodeAOK(t, "count 0, no palette", trailerBytes(0, false), false)
	if noPal.Frames == nil || len(noPal.Frames) != 0 {
		t.Fatalf("frames = %v, want empty and non-nil", noPal.Frames)
	}
	if noPal.Palette != nil {
		t.Fatal("undeclared palette came back non-nil")
	}
}

// TestDecodeAUnfinishedGridAndZeroArea — SC-2: a block exhausting mid-grid
// leaves the remainder transparent, a zero-area frame decodes empty and
// non-nil, and trailing count-0 ops on a complete grid change nothing.
func TestDecodeAUnfinishedGridAndZeroArea(t *testing.T) {
	stream := concat(
		frameRecord(2, 2, words(litA(1), pixA(5, 5))), // block ends mid-grid
		frameRecord(0, 7, nil),                        // zero-width grid
		frameRecord(3, 0, nil),                        // zero-height grid
		frameRecord(1, 1, words(litA(1), pixA(2, 9), skipA(0), rowsA(0), aliasA(0), litA(0))),
		trailerBytes(4, false),
	)
	s := decodeAOK(t, "tolerated shapes", stream, false)
	if len(s.Frames) != 4 {
		t.Fatalf("frame count = %d, want 4", len(s.Frames))
	}
	assertFrameA(t, "unfinished grid", s.Frames[0], 2, 2, []PixelA{pa(5, 5), trA, trA, trA})
	assertFrameA(t, "zero-width frame", s.Frames[1], 0, 7, nil)
	assertFrameA(t, "zero-height frame", s.Frames[2], 3, 0, nil)
	assertFrameA(t, "trailing count-0 ops", s.Frames[3], 1, 1, []PixelA{pa(2, 9)})
}

// TestDecodeATrailerBit31Twins — SC-3 (AC-2): streams identical except
// trailer bit 31 decode deeply equal, under both palette declarations —
// bit 31 is not the sibling format's has-palette flag.
func TestDecodeATrailerBit31Twins(t *testing.T) {
	payload := frameRecord(2, 1, words(litA(2), pixA(1, 2), pixA(3, 4)))

	plain := decodeAOK(t, "bit 31 clear", concat(payload, trailerBytes(1, false)), false)
	set := decodeAOK(t, "bit 31 set", concat(payload, trailerBytes(1, true)), false)
	if !reflect.DeepEqual(plain, set) {
		t.Fatalf("bit-31 twins differ:\n%+v\nvs\n%+v", plain, set)
	}
	if len(plain.Frames) != 1 {
		t.Fatalf("frame count = %d, want 1", len(plain.Frames))
	}

	palPayload := concat(paletteA(map[int][4]byte{3: {5, 6, 7, 8}}), payload)
	palPlain := decodeAOK(t, "palette, bit 31 clear", concat(palPayload, trailerBytes(1, false)), true)
	palSet := decodeAOK(t, "palette, bit 31 set", concat(palPayload, trailerBytes(1, true)), true)
	if !reflect.DeepEqual(palPlain, palSet) {
		t.Fatalf("bit-31 twins differ under a declared palette:\n%+v\nvs\n%+v", palPlain, palSet)
	}
	if len(palPlain.Palette) != 256 {
		t.Fatalf("palette has %d entries, want 256", len(palPlain.Palette))
	}
}

// TestDecodeAIgnoresExtraBytes — SC-3 (AC-4): extra bytes before the
// trailer — a further well-formed record section included — change
// nothing.
func TestDecodeAIgnoresExtraBytes(t *testing.T) {
	counted := frameRecord(2, 1, words(litA(2), pixA(1, 1), pixA(2, 2)))
	bare := concat(counted, trailerBytes(1, false))
	extras := concat(
		counted,
		frameRecord(1, 1, words(litA(1), pixA(9, 9))), // a further well-formed record section
		[]byte{0xFF, 0x13},                            // junk
		trailerBytes(1, false),
	)

	want := decodeAOK(t, "bare", bare, false)
	got := decodeAOK(t, "with extras", extras, false)
	if len(got.Frames) != 1 || !reflect.DeepEqual(got, want) {
		t.Fatalf("extra bytes changed the decode:\n%+v\nvs\n%+v", got, want)
	}
}

// TestDecodeAPaletteDeclaration — SC-3 (AC-7): one payload decoded under
// both declarations yields frames from offset 1024 and 0 respectively, the
// palette present then nil — presence is the declaration, never a sniff.
func TestDecodeAPaletteDeclaration(t *testing.T) {
	rec0 := frameRecord(1, 1, words(litA(1), pixA(3, 9))) // 16 bytes, opens the payload
	payload := concat(
		rec0,
		make([]byte, 1024-len(rec0)), // pads the palette region to 1024
		frameRecord(2, 1, words(litA(2), pixA(1, 1), pixA(2, 2))),
		trailerBytes(1, false),
	)

	declared := decodeAOK(t, "palette declared", payload, true)
	if len(declared.Frames) != 1 {
		t.Fatalf("declared: frame count = %d, want 1", len(declared.Frames))
	}
	assertFrameA(t, "frames from offset 1024", declared.Frames[0], 2, 1,
		[]PixelA{pa(1, 1), pa(2, 2)})
	if len(declared.Palette) != 256 {
		t.Fatalf("declared: palette has %d entries, want 256", len(declared.Palette))
	}
	// The palette is those same leading bytes read as [B, G, R, x] entries:
	// entry 0 is rec0's width word 01 00 00 00, entry 3 its block 01 00 06 12.
	for i, want := range map[int]Color{
		0: {R: 0, G: 0, B: 1},
		3: {R: 0x06, G: 0x00, B: 0x01},
		4: {},
	} {
		if declared.Palette[i] != want {
			t.Fatalf("declared: palette[%d] = %+v, want %+v", i, declared.Palette[i], want)
		}
	}

	absent := decodeAOK(t, "palette declared absent", payload, false)
	if absent.Palette != nil {
		t.Fatalf("absent: palette came back non-nil: %d entries", len(absent.Palette))
	}
	if len(absent.Frames) != 1 {
		t.Fatalf("absent: frame count = %d, want 1", len(absent.Frames))
	}
	assertFrameA(t, "frames from offset 0", absent.Frames[0], 1, 1, []PixelA{pa(3, 9)})
}

// --- refusals ---

// malformedA — SC-4's malformed table: each stream must refuse with a nil
// *SpriteA, and each seeds FuzzDecodeA.
var malformedA = []struct {
	name    string
	data    []byte
	palette bool
}{
	// AC-3: each sanity cap, one over its limit.
	{"width one over cap", concat(frameRecord(2049, 1, nil), trailerBytes(1, false)), false},
	{"height one over cap", concat(frameRecord(1, 2049, nil), trailerBytes(1, false)), false},
	{"dataSize one over cap", concat(rawFrame(1, 1, 1<<24+1, nil), trailerBytes(1, false)), false},
	{"frame count one over cap", trailerBytes(4097, false), false},
	// AC-5: the last frame's dataSize runs past the trailer.
	{"block runs past the trailer", concat(rawFrame(2, 2, 100, words(litA(0))), trailerBytes(1, false)), false},
	// AC-6: ops that would move or paint past width x height.
	{"skip past the grid", concat(frameRecord(2, 1, words(skipA(3))), trailerBytes(1, false)), false},
	{"blank rows past the grid", concat(frameRecord(2, 2, words(rowsA(3))), trailerBytes(1, false)), false},
	{"alias rows past the grid", concat(frameRecord(2, 2, words(aliasA(3))), trailerBytes(1, false)), false},
	{"literal paints past the grid", concat(frameRecord(1, 1, words(litA(2), pixA(1, 1), pixA(2, 2))), trailerBytes(1, false)), false},
	// The completion rule: a non-zero count once the grid is complete.
	{"op after the grid completes", concat(frameRecord(1, 1, words(litA(1), pixA(1, 1), skipA(1))), trailerBytes(1, false)), false},
	{"non-zero op on a zero-width grid", concat(frameRecord(0, 2, words(rowsA(1))), trailerBytes(1, false)), false},
	// AC-8: streams shorter than their fixed regions.
	{"empty stream", nil, false},
	{"empty stream with a declared palette", nil, true},
	{"4-byte stream with a declared palette", trailerBytes(1, false), true},
	// AC-11: a literal run truncated by its block end; the odd byte where a
	// word should start is the same refusal.
	{"literal words truncated by the block end", concat(frameRecord(2, 1, words(litA(2), pixA(1, 1))), trailerBytes(1, false)), false},
	{"odd byte where a control should start", concat(frameRecord(2, 1, []byte{0x01}), trailerBytes(1, false)), false},
	{"odd byte where an operand should start", concat(frameRecord(1, 1, concat(words(litA(1)), []byte{0x06})), trailerBytes(1, false)), false},
}

// TestDecodeARejectsMalformed — SC-4 (AC-3, AC-5, AC-6, AC-8, AC-11):
// every malformed stream refuses with an error and a nil *SpriteA — no
// frames, no palette.
func TestDecodeARejectsMalformed(t *testing.T) {
	for _, m := range malformedA {
		s, err := DecodeA(m.data, m.palette)
		if err == nil {
			t.Fatalf("%s: expected an error, got a sprite with %d frame(s)", m.name, len(s.Frames))
		}
		if s != nil {
			t.Fatalf("%s: expected a nil *SpriteA on error, got non-nil", m.name)
		}
	}
}

// FuzzDecodeA — SC-4: seeded from the malformed table; for any input
// DecodeA neither panics nor pairs an error with a result.
func FuzzDecodeA(f *testing.F) {
	for _, m := range malformedA {
		f.Add(m.data, m.palette)
	}
	f.Fuzz(func(t *testing.T, data []byte, palette bool) {
		s, err := DecodeA(data, palette)
		if err != nil && s != nil {
			t.Fatalf("error %v alongside a non-nil *SpriteA", err)
		}
	})
}
