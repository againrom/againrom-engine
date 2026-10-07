package spr256_test

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/spr256"
)

// --- synthetic fixture builders (no game bytes; .256 carries no strings) ---

func u32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// frameRecord builds one on-disk frame: [u32 width][u32 height][u32 len(block)][block].
func frameRecord(width, height uint32, block []byte) []byte {
	return concat(u32(width), u32(height), u32(uint32(len(block))), block)
}

// rawFrame builds a frame header with an explicit dataSize that need not match
// the following bytes — used to craft a block that overruns the trailer.
func rawFrame(width, height, dataSize uint32, block []byte) []byte {
	return concat(u32(width), u32(height), u32(dataSize), block)
}

// palette builds a 1024-byte palette; each entry i is set from entries[i] as the
// on-disk [B, G, R, reserved] quad. Unset entries stay zero.
func palette(entries map[int][4]byte) []byte {
	p := make([]byte, 1024)
	for i, e := range entries {
		copy(p[i*4:i*4+4], e[:])
	}
	return p
}

// trailer builds the 4-byte trailer: low 31 bits = frameCount, bit 31 = has-palette.
func trailer(frameCount uint32, hasPalette bool) []byte {
	v := frameCount & 0x7FFFFFFF
	if hasPalette {
		v |= 0x80000000
	}
	return u32(v)
}

// --- assertion helpers ---

func op(i uint8) spr256.Pixel { return spr256.Pixel{Index: i, Opaque: true} }

var tp = spr256.Pixel{} // transparent (the zero value)

func decodeOK(t *testing.T, name string, data []byte) *spr256.Sprite {
	t.Helper()
	s, err := spr256.Decode(data)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	if s == nil {
		t.Fatalf("%s: nil *Sprite without error", name)
	}
	return s
}

func assertReject(t *testing.T, name string, data []byte) {
	t.Helper()
	s, err := spr256.Decode(data)
	if err == nil {
		t.Fatalf("%s: expected an error, got nil", name)
	}
	if s != nil {
		t.Fatalf("%s: expected a nil *Sprite on error, got non-nil", name)
	}
}

func assertFrame(t *testing.T, name string, got spr256.Frame, width, height int, want []spr256.Pixel) {
	t.Helper()
	if got.Width != width || got.Height != height {
		t.Fatalf("%s: got %dx%d, want %dx%d", name, got.Width, got.Height, width, height)
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

// Shared frame blocks. block2x2 fills row 0 with two literals (indices 10, 20)
// and skips row 1 transparently; block2x3 emits a blank row, a literal row, and
// a transparent-skip row.
var (
	block2x2 = []byte{0x02, 10, 20, 0x82}             // literal 2 + transparent skip 2
	grid2x2  = []spr256.Pixel{op(10), op(20), tp, tp} // 2x2, row 1 transparent

	block2x3 = []byte{0x41, 0x02, 30, 40, 0x82} // blank row + literal 2 + transparent skip 2
	grid2x3  = []spr256.Pixel{tp, tp, op(30), op(40), tp, tp}
)

// TestDecodeTwoFrameGrids - AC-1 / SC-1: a two-frame stream exercising literal,
// 0x80 transparent-pixel and 0x40 blank-row ops, closed by trailer 0x80000002.
func TestDecodeTwoFrameGrids(t *testing.T) {
	stream := concat(
		palette(nil),
		frameRecord(2, 2, block2x2),
		frameRecord(2, 3, block2x3),
		trailer(2, true),
	)
	s := decodeOK(t, "two-frame", stream)

	if !s.HasPalette {
		t.Fatal("HasPalette = false, want true")
	}
	if len(s.Palette) != 256 {
		t.Fatalf("len(Palette) = %d, want 256", len(s.Palette))
	}
	if len(s.Frames) != 2 {
		t.Fatalf("len(Frames) = %d, want 2 (the bit-31 trailer must end the list)", len(s.Frames))
	}
	assertFrame(t, "frame 0", s.Frames[0], 2, 2, grid2x2)
	assertFrame(t, "frame 1", s.Frames[1], 2, 3, grid2x3)
}

// TestPaletteBGRToRGB - AC-2 / SC-2: a [B,G,R,reserved] entry is returned as its
// RGB reordering; entry 0 (the reserved transparent key) is present, not dropped.
func TestPaletteBGRToRGB(t *testing.T) {
	stream := concat(
		palette(map[int][4]byte{
			0: {0x01, 0x02, 0x03, 0x00}, // B,G,R,reserved
			5: {0x11, 0x22, 0x33, 0x44},
		}),
		trailer(0, true),
	)
	s := decodeOK(t, "palette", stream)

	if len(s.Palette) != 256 {
		t.Fatalf("len(Palette) = %d, want 256", len(s.Palette))
	}
	if got, want := s.Palette[0], (spr256.Color{R: 0x03, G: 0x02, B: 0x01}); got != want {
		t.Fatalf("Palette[0] (transparent key) = %+v, want %+v", got, want)
	}
	if got, want := s.Palette[5], (spr256.Color{R: 0x33, G: 0x22, B: 0x11}); got != want {
		t.Fatalf("Palette[5] = %+v, want %+v (reserved byte 0x44 dropped)", got, want)
	}
	if len(s.Frames) != 0 {
		t.Fatalf("len(Frames) = %d, want 0", len(s.Frames))
	}
}

// TestDecodeNoPaletteVariant - AC-3 / SC-3: a palette-less stream (frames from
// offset 0, trailer bit 31 clear) decodes its frames and reports HasPalette false.
func TestDecodeNoPaletteVariant(t *testing.T) {
	stream := concat(
		frameRecord(2, 2, block2x2),
		trailer(1, false),
	)
	s := decodeOK(t, "no-palette", stream)

	if s.HasPalette {
		t.Fatal("HasPalette = true, want false (bit 31 clear)")
	}
	if s.Palette != nil {
		t.Fatalf("Palette = %v, want nil for a palette-less stream", s.Palette)
	}
	if len(s.Frames) != 1 {
		t.Fatalf("len(Frames) = %d, want 1 (count honored)", len(s.Frames))
	}
	assertFrame(t, "frame 0", s.Frames[0], 2, 2, grid2x2)
}

// TestRejectFrameBlockPastTrailer - AC-4 / SC-4: a frame whose dataSize runs past
// the trailer is rejected atomically.
func TestRejectFrameBlockPastTrailer(t *testing.T) {
	stream := concat(
		palette(nil),
		rawFrame(2, 2, 100, []byte{1, 2, 3, 4}), // claims 100 block bytes, 4 present
		trailer(1, true),
	)
	assertReject(t, "block past trailer", stream)
}

func TestRejectRLETilingErrors(t *testing.T) {
	// Row tokens sum to 2 but width is 3 (row never completes).
	shortRow := concat(frameRecord(3, 1, []byte{0x02, 1, 2}), trailer(1, false))
	assertReject(t, "row tokens < width", shortRow)

	// Three transparent-skip rows into a 2-row grid: the third overruns height.
	tallGrid := concat(frameRecord(2, 2, []byte{0x82, 0x82, 0x82}), trailer(1, false))
	assertReject(t, "RLE overruns height", tallGrid)
}

func TestRejectLiteralOverrunsBlock(t *testing.T) {
	// Control 0x05 claims 5 literal bytes, but only 2 follow in the block.
	stream := concat(frameRecord(5, 1, []byte{0x05, 1, 2}), trailer(1, false))
	assertReject(t, "literal overruns block", stream)
}

func TestRejectShortAndOversizedStreams(t *testing.T) {
	assertReject(t, "empty stream", []byte{})

	// bit 31 set but only 8 bytes total (a full palette + trailer needs 1028).
	assertReject(t, "has-palette but short", concat([]byte{0, 0, 0, 0}, trailer(1, true)))

	// width*height = (2^32-1)^2 exceeds the addressable pixel limit -> error, not
	// a panic, and no giant allocation (the guard fires before make).
	oversize := concat(frameRecord(0xFFFFFFFF, 0xFFFFFFFF, []byte{0x00}), trailer(1, false))
	assertReject(t, "oversized dimensions", oversize)
}

// TestDecodeEmptyGridBetweenFrames - AC-8 / SC-8: a width=0 empty-grid frame
// between two valid frames decodes to three frames, the empty one of size 0.
func TestDecodeEmptyGridBetweenFrames(t *testing.T) {
	blockA := []byte{0x02, 7, 8, 0x82} // 2x2, row 1 transparent
	blockC := []byte{0x02, 9, 10}      // 2x1, one full literal row

	stream := concat(
		palette(nil),
		frameRecord(2, 2, blockA),
		frameRecord(0, 0, nil), // empty grid
		frameRecord(2, 1, blockC),
		trailer(3, true),
	)
	s := decodeOK(t, "empty-between", stream)

	if len(s.Frames) != 3 {
		t.Fatalf("len(Frames) = %d, want 3", len(s.Frames))
	}
	assertFrame(t, "frame 0", s.Frames[0], 2, 2, []spr256.Pixel{op(7), op(8), tp, tp})
	assertFrame(t, "empty frame 1", s.Frames[1], 0, 0, []spr256.Pixel{})
	assertFrame(t, "frame 2", s.Frames[2], 2, 1, []spr256.Pixel{op(9), op(10)})
}

// TestDecodeBucketBSingleFrame - AC-10 / SC-9: a Bucket-B stream (palette + one
// frame + trailer 0x80000001 + an appended 0x80000001 ... 0x80000001 section)
// decodes to exactly one frame; the appended section is left untouched, no error.
func TestDecodeBucketBSingleFrame(t *testing.T) {
	block1 := []byte{0x02, 5, 6, 0x82} // 2x2, row 1 transparent

	// The appended section opens with 0x80000001 (an impossible frame width) and
	// closes with 0x80000001, which is the actual last-4-byte trailer.
	appended := concat(
		u32(0x80000001),
		[]byte{1, 2, 3, 4, 5, 6, 7, 8},
		u32(0x80000001),
	)
	stream := concat(
		palette(nil),
		frameRecord(2, 2, block1),
		appended,
	)
	s := decodeOK(t, "bucket-b", stream)

	if !s.HasPalette {
		t.Fatal("HasPalette = false, want true")
	}
	if len(s.Frames) != 1 {
		t.Fatalf("len(Frames) = %d, want exactly 1 (appended section must be unread)", len(s.Frames))
	}
	assertFrame(t, "frame 0", s.Frames[0], 2, 2, []spr256.Pixel{op(5), op(6), tp, tp})
}

// TestDecodeC0AliasesC80 - AC-11 / SC-10: a 0xC0-class control decodes identically
// to 0x80 (a transparent skip of N), matching the shipped loader; no error.
func TestDecodeC0AliasesC80(t *testing.T) {
	viaC0 := decodeOK(t, "0xC0", concat(frameRecord(2, 1, []byte{0xC2}), trailer(1, false)))
	viaC80 := decodeOK(t, "0x80", concat(frameRecord(2, 1, []byte{0x82}), trailer(1, false)))

	// 0xC2 is a transparent skip of 2: a fully transparent 2x1 row.
	assertFrame(t, "0xC0 frame", viaC0.Frames[0], 2, 1, []spr256.Pixel{tp, tp})
	// And it must be byte-identical to the 0x82 decoding.
	assertFrame(t, "0x80 frame", viaC80.Frames[0], 2, 1, viaC0.Frames[0].Pixels)
}
