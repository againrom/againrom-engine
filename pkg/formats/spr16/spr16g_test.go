package spr16

import (
	"reflect"
	"testing"
)

// Control bytes: op in bits 6–7, a 6-bit count below.
func litG(n byte) byte   { return n }        // 0b00: literal, n pixel bytes follow
func rowsG(n byte) byte  { return 0x40 | n } // 0b01: blank rows
func skipG(n byte) byte  { return 0x80 | n } // 0b10: skip
func aliasG(n byte) byte { return 0xC0 | n } // 0b11: decoded as skip

// nib builds a literal pixel byte from its two 4-bit values, low nibble first —
// the order the decoder paints them in.
func nib(lo, hi byte) byte { return hi<<4 | lo }

// pg is a painted pixel; the transparent pixel is the PixelG zero value.
func pg(value uint8) PixelG { return PixelG{Value: value, Painted: true} }

var trG = PixelG{}

// --- assertion helpers ---

func decodeGOK(t *testing.T, name string, data []byte) []FrameG {
	t.Helper()
	frames, err := DecodeG(data)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	if frames == nil {
		t.Fatalf("%s: nil frame list without error", name)
	}
	return frames
}

func assertFrameG(t *testing.T, name string, got FrameG, width, height int, want []PixelG) {
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

// TestDecodeGSpecExample — SC-5: the spec's printed .16 I/O example, block
// bytes verbatim, decodes to its printed grid.
func TestDecodeGSpecExample(t *testing.T) {
	block := []byte{
		0x02, // literal n=2 bytes
		0x5A, // low 0xA → value 10 at (0,0); high 0x5 → value 5 at (1,0)
		0x0C, // final byte: low 0xC → value 12 at (2,0); high 0 → pad
		0x81, // skip n=1 → (0,1) transparent
		0x01, // literal n=1 byte
		0x0F, // final byte: low 0xF → value 15 at (1,1); high 0 → pad
		0x81, // skip n=1 → (2,1) transparent
	}
	frames := decodeGOK(t, "spec example",
		concat(frameRecord(3, 2, block), trailerBytes(1, false)))
	if len(frames) != 1 {
		t.Fatalf("frame count = %d, want 1", len(frames))
	}
	assertFrameG(t, "spec example", frames[0], 3, 2, []PixelG{
		pg(10), pg(5), pg(12),
		trG, pg(15), trG,
	})
}

// TestDecodeGPadAndMidRunZero — SC-5 (AC-9): one run ends in a non-zero
// high nibble (both painted), one ends in a zero high nibble (a pad), and a
// mid-run zero high nibble is followed by further pixels — it must paint a
// value-0 pixel and advance, so every later pixel lands at its correct
// position; a pad-anywhere miswiring desyncs the cursor and lands them
// wrong.
func TestDecodeGPadAndMidRunZero(t *testing.T) {
	block := []byte{
		litG(3),
		nib(1, 0), // mid-run zero high nibble: paints 1 then 0, advances
		nib(2, 3),
		nib(4, 5), // final byte, high nibble non-zero: paints both
		litG(1),
		nib(7, 0), // final byte, high nibble 0: paints 7, pad emits nothing
	}
	frames := decodeGOK(t, "pad and mid-run zero",
		concat(frameRecord(4, 2, block), trailerBytes(1, false)))
	assertFrameG(t, "pad and mid-run zero", frames[0], 4, 2, []PixelG{
		pg(1), pg(0), pg(2), pg(3),
		pg(4), pg(5), pg(7), trG,
	})
	if frames[0].Pixels[1] == trG {
		t.Fatal("a painted value-0 pixel compares equal to a transparent one")
	}
}

// TestDecodeGBlankRowsMidFrame — SC-5: a blank-row run between literals on
// a width above one lands the following pixels exactly — a
// rows-miswired-as-skip arm would land them two rows early.
func TestDecodeGBlankRowsMidFrame(t *testing.T) {
	block := []byte{
		litG(2), nib(1, 2), nib(3, 0), // row 0: values 1, 2, 3
		rowsG(2),                      // rows 1 and 2 blank
		litG(2), nib(4, 5), nib(6, 0), // row 3: values 4, 5, 6
	}
	frames := decodeGOK(t, "blank rows mid-frame",
		concat(frameRecord(3, 4, block), trailerBytes(1, false)))
	assertFrameG(t, "blank rows mid-frame", frames[0], 3, 4, []PixelG{
		pg(1), pg(2), pg(3),
		trG, trG, trG,
		trG, trG, trG,
		pg(4), pg(5), pg(6),
	})
}

// TestDecodeGAliasSkip — SC-5 (AC-13): the same program written with op
// 0b11 decodes identically to op 0b10 (skip) — the opposite direction from
// the .16a alias, and a 0b11-as-rows miswiring would land the trailing
// literal a full extra row off, so equality discriminates.
func TestDecodeGAliasSkip(t *testing.T) {
	prog := func(skip byte) []byte {
		return []byte{litG(1), nib(1, 0), skip, litG(1), nib(2, 3)}
	}
	want := []PixelG{
		pg(1), trG, trG,
		pg(2), pg(3), trG,
		trG, trG, trG,
	}

	fSkip := decodeGOK(t, "op 0b10",
		concat(frameRecord(3, 3, prog(skipG(2))), trailerBytes(1, false)))
	fAlias := decodeGOK(t, "op 0b11",
		concat(frameRecord(3, 3, prog(aliasG(2))), trailerBytes(1, false)))

	if !reflect.DeepEqual(fSkip, fAlias) {
		t.Fatalf("op 0b11 decoded unlike op 0b10:\n%+v\nvs\n%+v", fAlias, fSkip)
	}
	assertFrameG(t, "skip program", fSkip[0], 3, 3, want)
}

// TestDecodeGCountZero — SC-5 (AC-14): a 0-count trailer decodes to an
// empty, non-nil frame list — a bare trailer is the minimum well-formed
// stream.
func TestDecodeGCountZero(t *testing.T) {
	frames := decodeGOK(t, "count 0", trailerBytes(0, false))
	if len(frames) != 0 {
		t.Fatalf("frame count = %d, want 0", len(frames))
	}
}

// TestDecodeGUnfinishedGridAndZeroArea — SC-5: a block exhausting mid-grid
// leaves the remainder transparent, zero-area frames decode empty and
// non-nil, a final pad may land the cursor exactly on the grid end, and
// trailing count-0 ops on a complete grid change nothing.
func TestDecodeGUnfinishedGridAndZeroArea(t *testing.T) {
	stream := concat(
		frameRecord(2, 2, []byte{litG(1), nib(5, 6)}), // block ends mid-grid
		frameRecord(0, 7, nil),                        // zero-width grid
		frameRecord(3, 0, nil),                        // zero-height grid
		frameRecord(1, 1, []byte{litG(1), nib(9, 0), skipG(0), rowsG(0), aliasG(0), litG(0)}),
		trailerBytes(4, false),
	)
	frames := decodeGOK(t, "tolerated shapes", stream)
	if len(frames) != 4 {
		t.Fatalf("frame count = %d, want 4", len(frames))
	}
	assertFrameG(t, "unfinished grid", frames[0], 2, 2, []PixelG{pg(5), pg(6), trG, trG})
	assertFrameG(t, "zero-width frame", frames[1], 0, 7, nil)
	assertFrameG(t, "zero-height frame", frames[2], 3, 0, nil)
	assertFrameG(t, "trailing count-0 ops", frames[3], 1, 1, []PixelG{pg(9)})
}

func TestDecodeG1175DistinguishesRawTrailerFromPaletteCount(t *testing.T) {
	payload := frameRecord(2, 1, []byte{litG(1), nib(1, 2)})
	plain := decodeGOK(t, "bit 31 clear", concat(payload, trailerBytes(1, false)))
	if len(plain) != 1 {
		t.Fatalf("frame count = %d, want 1", len(plain))
	}
	if got := decodeGOK(t, "zero count", trailerBytes(0, false)); len(got) != 0 {
		t.Fatalf("zero count produced %+v", got)
	}
	for _, raw := range []uint32{0x80000000, 0x80000001, 0xffffffff, 0x40000001, 0x7fffffff} {
		// Negative words are explicit policy refusals; large positive words
		// still hit the allocation cap instead of wrapping their count.
		got, err := DecodeG(concat(payload, u32(raw)))
		if err == nil || got != nil {
			t.Fatalf("raw trailer %#08x produced %d frames, error %v", raw, len(got), err)
		}
	}
}

// TestDecodeGIgnoresExtraBytes — SC-6 (AC-4): extra bytes before the
// trailer — a further well-formed record section included — change
// nothing.
func TestDecodeGIgnoresExtraBytes(t *testing.T) {
	counted := frameRecord(2, 1, []byte{litG(1), nib(1, 2)})
	bare := concat(counted, trailerBytes(1, false))
	extras := concat(
		counted,
		frameRecord(1, 1, []byte{litG(1), nib(9, 0)}), // a further well-formed record section
		[]byte{0xFF, 0x13},                            // junk
		trailerBytes(1, false),
	)

	want := decodeGOK(t, "bare", bare)
	got := decodeGOK(t, "with extras", extras)
	if len(got) != 1 || !reflect.DeepEqual(got, want) {
		t.Fatalf("extra bytes changed the decode:\n%+v\nvs\n%+v", got, want)
	}
}

// --- refusals ---

// malformedG — SC-6's malformed table: each stream must refuse with a nil
// frame list, and each seeds FuzzDecodeG.
var malformedG = []struct {
	name string
	data []byte
}{
	// AC-6: ops that would move or paint past width x height.
	{"skip past the grid", concat(frameRecord(2, 1, []byte{skipG(3)}), trailerBytes(1, false))},
	{"blank rows past the grid", concat(frameRecord(2, 2, []byte{rowsG(3)}), trailerBytes(1, false))},
	{"alias skip past the grid", concat(frameRecord(2, 1, []byte{aliasG(3)}), trailerBytes(1, false))},
	{"literal low nibble paints past the grid", concat(frameRecord(2, 1, []byte{litG(2), nib(1, 2), nib(3, 4)}), trailerBytes(1, false))},
	{"literal high nibble paints past the grid", concat(frameRecord(1, 1, []byte{litG(1), nib(1, 2)}), trailerBytes(1, false))},
	// The completion rule: a non-zero count once the grid is complete.
	{"op after the grid completes", concat(frameRecord(1, 1, []byte{litG(1), nib(1, 0), skipG(1)}), trailerBytes(1, false))},
	{"non-zero op on a zero-width grid", concat(frameRecord(0, 2, []byte{rowsG(1)}), trailerBytes(1, false))},
	// AC-8: streams shorter than their fixed regions.
	{"empty stream", nil},
	{"3-byte stream", []byte{1, 2, 3}},
	// AC-11: a literal run truncated by its block end.
	{"literal bytes truncated by the block end", concat(frameRecord(3, 1, []byte{litG(2), nib(1, 2)}), trailerBytes(1, false))},
}

// TestDecodeGRejectsMalformed — SC-6 (AC-6, AC-8, AC-11): every malformed
// stream refuses with an error and a nil frame list.
func TestDecodeGRejectsMalformed(t *testing.T) {
	for _, m := range malformedG {
		frames, err := DecodeG(m.data)
		if err == nil {
			t.Fatalf("%s: expected an error, got %d frame(s)", m.name, len(frames))
		}
		if frames != nil {
			t.Fatalf("%s: expected a nil frame list on error, got %d frame(s)", m.name, len(frames))
		}
	}
}

// FuzzDecodeG — SC-6: seeded from the malformed table; for any input
// DecodeG neither panics nor pairs an error with a result.
func FuzzDecodeG(f *testing.F) {
	for _, m := range malformedG {
		f.Add(m.data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		frames, err := DecodeG(data)
		if err != nil && frames != nil {
			t.Fatalf("error %v alongside a non-nil frame list", err)
		}
	})
}
