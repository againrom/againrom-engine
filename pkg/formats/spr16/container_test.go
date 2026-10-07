package spr16

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

// --- synthetic fixture builders (no game bytes) ---

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

// frameRecord builds one on-disk frame record:
// [u32 width][u32 height][u32 len(block)][block].
func frameRecord(width, height uint32, block []byte) []byte {
	return concat(u32(width), u32(height), u32(uint32(len(block))), block)
}

// rawFrame builds a frame header with an explicit dataSize that need not match
// the following bytes — used to craft a block that overruns the trailer.
func rawFrame(width, height, dataSize uint32, block []byte) []byte {
	return concat(u32(width), u32(height), u32(dataSize), block)
}

// trailerBytes builds a .16a trailer: low31 count and separate palette bit.
func trailerBytes(count uint32, bit31 bool) []byte {
	v := count & 0x7FFFFFFF
	if bit31 {
		v |= 0x80000000
	}
	return u32(v)
}

// --- assertion helpers ---

func recordsOK(t *testing.T, name string, data []byte, start int) []record {
	t.Helper()
	recs, err := records(data, start)
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", name, err)
	}
	if recs == nil {
		t.Fatalf("%s: nil records without error", name)
	}
	return recs
}

func assertRecordsReject(t *testing.T, name string, data []byte, start int) {
	t.Helper()
	recs, err := records(data, start)
	if err == nil {
		t.Fatalf("%s: expected an error, got %d record(s)", name, len(recs))
	}
	if recs != nil {
		t.Fatalf("%s: expected nil records on error, got %d", name, len(recs))
	}
}

// --- the container walk ---

// The .16a count helper masks bit31. DecodeG applies its separate signed
// trailer admission before reaching this helper.
func TestRecordsMasksPaletteTrailerBit31(t *testing.T) {
	block := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	set := concat(frameRecord(2, 3, block), trailerBytes(1, true))
	plain := concat(frameRecord(2, 3, block), trailerBytes(1, false))

	recsSet := recordsOK(t, "bit 31 set", set, 0)
	recsPlain := recordsOK(t, "bit 31 clear", plain, 0)

	if len(recsSet) != 1 {
		t.Fatalf("len = %d, want 1", len(recsSet))
	}
	r := recsSet[0]
	if r.width != 2 || r.height != 3 || !bytes.Equal(r.block, block) {
		t.Fatalf("record = %dx%d block %v, want 2x3 block %v", r.width, r.height, r.block, block)
	}
	if !reflect.DeepEqual(recsSet, recsPlain) {
		t.Fatalf("bit-31 twins differ: %v vs %v", recsSet, recsPlain)
	}
}

// TestRecordsStopsAtCount — SC-1: the walk stops at count; a further
// well-formed record section and junk before the trailer are never read.
func TestRecordsStopsAtCount(t *testing.T) {
	blockA := []byte{1, 2, 3}
	bare := concat(frameRecord(4, 1, blockA), trailerBytes(1, false))
	extras := concat(
		frameRecord(4, 1, blockA),
		frameRecord(2, 2, []byte{9, 9, 9, 9}), // a further well-formed record section
		[]byte{0xFF, 0x00, 0xFF},              // junk
		trailerBytes(1, false),
	)

	want := recordsOK(t, "bare", bare, 0)
	got := recordsOK(t, "with extras", extras, 0)
	if len(got) != 1 || !reflect.DeepEqual(got, want) {
		t.Fatalf("extra bytes changed the walk: got %v, want %v", got, want)
	}
}

// TestRecordsWalksFromStart — SC-1: the palette case hands the walk start
// 1024; records read from there, the leading bytes untouched.
func TestRecordsWalksFromStart(t *testing.T) {
	pad := make([]byte, 1024)
	block := []byte{7, 8}
	stream := concat(pad, frameRecord(2, 1, block), trailerBytes(1, false))

	recs := recordsOK(t, "start 1024", stream, 1024)
	if len(recs) != 1 || recs[0].width != 2 || recs[0].height != 1 || !bytes.Equal(recs[0].block, block) {
		t.Fatalf("recs = %v, want one 2x1 record with block %v", recs, block)
	}
}

// TestRecordsCountZero — SC-1: a 0-count trailer walks nothing and returns
// an empty, non-nil record list, ignored bytes included.
func TestRecordsCountZero(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"trailer alone", trailerBytes(0, false)},
		{"ignored bytes before the trailer", concat([]byte{1, 2, 3, 4, 5}, trailerBytes(0, true))},
	} {
		recs := recordsOK(t, tc.name, tc.data, 0)
		if len(recs) != 0 {
			t.Fatalf("%s: len = %d, want 0", tc.name, len(recs))
		}
	}
}

// TestRecordsRefusesCaps — SC-1: each sanity cap refuses one over its
// limit — width, height, dataSize, count, stream length — with no
// records; the at-cap dimensions pass.
func TestRecordsRefusesCaps(t *testing.T) {
	assertRecordsReject(t, "width 2049",
		concat(frameRecord(2049, 1, nil), trailerBytes(1, false)), 0)
	assertRecordsReject(t, "height 2049",
		concat(frameRecord(1, 2049, nil), trailerBytes(1, false)), 0)
	assertRecordsReject(t, "dataSize 2^24+1",
		concat(rawFrame(1, 1, 1<<24+1, nil), trailerBytes(1, false)), 0)
	assertRecordsReject(t, "count 4097", trailerBytes(4097, false), 0)

	// Trailing 4 zero bytes would read as count 0 were the length cap absent.
	over := make([]byte, 1<<26+1)
	assertRecordsReject(t, "stream 2^26+1", over, 0)

	// The caps sit one past the largest legal value, not at it.
	recs := recordsOK(t, "at-cap dimensions",
		concat(frameRecord(2048, 2048, nil), trailerBytes(1, false)), 0)
	if len(recs) != 1 || recs[0].width != 2048 || recs[0].height != 2048 {
		t.Fatalf("recs = %v, want one 2048x2048 record", recs)
	}
}

// TestRecordsRefusesHeaderOrBlockPastTrailer — SC-1: a header or block
// crossing the trailer's first byte refuses with no records.
func TestRecordsRefusesHeaderOrBlockPastTrailer(t *testing.T) {
	// count 1 with nothing before the trailer: no room for a header.
	assertRecordsReject(t, "header past trailer", trailerBytes(1, false), 0)

	// The header claims a 100-byte block; 4 bytes exist before the trailer.
	assertRecordsReject(t, "block past trailer",
		concat(rawFrame(2, 2, 100, []byte{1, 2, 3, 4}), trailerBytes(1, false)), 0)

	// count 2 with one record present: frame 1's header does not fit.
	assertRecordsReject(t, "second header past trailer",
		concat(frameRecord(1, 1, []byte{5}), trailerBytes(2, false)), 0)
}

// TestRecordsRefusesShortStream — SC-1: a stream shorter than its fixed
// regions refuses before any read — the 4-byte declared-palette form
// included.
func TestRecordsRefusesShortStream(t *testing.T) {
	assertRecordsReject(t, "empty stream", nil, 0)
	assertRecordsReject(t, "3 bytes", []byte{1, 2, 3}, 0)
	assertRecordsReject(t, "4 bytes under a declared palette", trailerBytes(1, false), 1024)
}

// --- the cursor ---

// TestCursorLandsExactlyOnGridEnd — SC-1: skip, rows and paint may each
// land the cursor exactly on w*h; paint yields linear indices in order.
func TestCursorLandsExactlyOnGridEnd(t *testing.T) {
	c := cursor{w: 3, h: 2}
	if err := c.skip(6); err != nil {
		t.Fatalf("skip(6) onto the end: %v", err)
	}
	if c.pos != 6 {
		t.Fatalf("pos = %d, want 6", c.pos)
	}

	c = cursor{w: 3, h: 2}
	if err := c.rows(2); err != nil {
		t.Fatalf("rows(2) onto the end: %v", err)
	}
	if c.pos != 6 {
		t.Fatalf("pos = %d, want 6", c.pos)
	}

	c = cursor{w: 3, h: 2}
	for want := 0; want < 6; want++ {
		got, err := c.paint()
		if err != nil {
			t.Fatalf("paint %d: %v", want, err)
		}
		if got != want {
			t.Fatalf("paint yielded %d, want %d", got, want)
		}
	}
	if _, err := c.paint(); err == nil {
		t.Fatal("paint past the grid end: want an error")
	}
}

// TestCursorRefusesOneStepPast — SC-1: one pixel past w*h refuses by skip,
// rows and paint alike, and a failed op does not move the cursor.
func TestCursorRefusesOneStepPast(t *testing.T) {
	c := cursor{w: 3, h: 2}
	if err := c.skip(7); err == nil {
		t.Fatal("skip(7) on a 6-pixel grid: want an error")
	}
	if c.pos != 0 {
		t.Fatalf("failed skip moved the cursor to %d", c.pos)
	}
	if err := c.rows(3); err == nil {
		t.Fatal("rows(3) on a 2-row grid: want an error")
	}
	if c.pos != 0 {
		t.Fatalf("failed rows moved the cursor to %d", c.pos)
	}

	// From one short of the end: a 2-skip refuses, then a 1-skip still lands.
	c = cursor{w: 3, h: 2, pos: 5}
	if err := c.skip(2); err == nil {
		t.Fatal("skip(2) from pos 5: want an error")
	}
	if err := c.skip(1); err != nil {
		t.Fatalf("skip(1) from pos 5: %v", err)
	}
}

// TestCursorRowsFromMidRow — SC-1: a blank-row op is plain cursor
// arithmetic from any position, not a row-boundary op.
func TestCursorRowsFromMidRow(t *testing.T) {
	c := cursor{w: 3, h: 3}
	if err := c.skip(1); err != nil {
		t.Fatalf("skip(1): %v", err)
	}
	if err := c.rows(1); err != nil {
		t.Fatalf("rows(1) from a mid-row position: %v", err)
	}
	got, err := c.paint()
	if err != nil {
		t.Fatalf("paint after mid-row rows: %v", err)
	}
	if got != 4 {
		t.Fatalf("paint yielded %d, want 4 (pos 1 + one 3-wide row)", got)
	}
	// From pos 5, two more rows overrun the 9-pixel grid.
	if err := c.rows(2); err == nil {
		t.Fatal("rows(2) from pos 5 on a 3x3 grid: want an error")
	}
}

// TestCursorCompletionRule — SC-1: a non-zero announced count on a
// complete grid refuses — the zero-width form included, though its
// blank-row op would move nothing — and count-0 ops pass anywhere.
func TestCursorCompletionRule(t *testing.T) {
	c := cursor{w: 2, h: 2}
	if err := c.announce(3); err != nil {
		t.Fatalf("announce(3) mid-grid: %v", err)
	}
	if err := c.skip(4); err != nil {
		t.Fatalf("skip(4): %v", err)
	}
	if err := c.announce(1); err == nil {
		t.Fatal("announce(1) on a complete grid: want an error")
	}
	if err := c.announce(0); err != nil {
		t.Fatalf("announce(0) on a complete grid: %v", err)
	}
	if err := c.skip(0); err != nil {
		t.Fatalf("skip(0) on a complete grid: %v", err)
	}
	if err := c.rows(0); err != nil {
		t.Fatalf("rows(0) on a complete grid: %v", err)
	}

	// A zero-width grid is born complete: its blank-row op would move nothing
	// and still refuses through the announced count.
	zw := cursor{w: 0, h: 5}
	if err := zw.announce(1); err == nil {
		t.Fatal("announce(1) on a zero-width grid: want an error")
	}
	if err := zw.announce(0); err != nil {
		t.Fatalf("announce(0) on a zero-width grid: %v", err)
	}

	za := cursor{w: 0, h: 0}
	if err := za.announce(2); err == nil {
		t.Fatal("announce(2) on a zero-area grid: want an error")
	}
}
