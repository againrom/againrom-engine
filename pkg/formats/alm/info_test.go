package alm_test

// Tests for the metadata-only entry point. Every fixture is a synthetic byte
// stream built from the documented layout; no file is read from disk.
//
// The point of OpenInfo is that it is STRICTLY WEAKER than Open, so these tests
// are mostly about the boundary between them: what both accept, what both
// reject, and the one class of stream that decodes here and fails there.

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
)

// wellFramed builds a valid map, then optionally replaces the type-1 payload with
// one of a different length. The record still declares its own real length, so
// the file tiles exactly to EOF and stays well framed: only the grid's
// consistency with W*H is broken.
func wellFramed(t *testing.T, w, h int, name, desc string, type1 []byte) []byte {
	t.Helper()
	return synth.ALM(synth.ALMOptions{
		Width: w, Height: h,
		Name: name, Description: desc,
		Type1Payload: type1,
	})
}

// TestOpenInfoAgreesWithOpen pins the fields OpenInfo is responsible for against
// the full decode, on streams both accept. They share one type-0 decoder, so a
// disagreement here would mean that sharing had been undone.
func TestOpenInfoAgreesWithOpen(t *testing.T) {
	cases := []struct {
		label      string
		w, h       int
		name, desc string
	}{
		{"named map", 8, 5, "Crossroads", "a description"},
		// Most campaign maps record no name at all. That is normal input, not a
		// failure, and it is the case a picker has to keep listing.
		{"empty name", 4, 4, "", ""},
		{"1x1", 1, 1, "Tiny", ""},
		{"non-square", 13, 7, "Wide", "d"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			data := wellFramed(t, tc.w, tc.h, tc.name, tc.desc, nil)

			m, err := alm.Open(data)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			info, err := alm.OpenInfo(data)
			if err != nil {
				t.Fatalf("OpenInfo: %v", err)
			}

			if info.Width != m.Width || info.Height != m.Height {
				t.Errorf("size = %dx%d, Open says %dx%d", info.Width, info.Height, m.Width, m.Height)
			}
			if info.Name != m.Name {
				t.Errorf("Name = %q, Open says %q", info.Name, m.Name)
			}
			if info.Description != m.Description {
				t.Errorf("Description = %q, Open says %q", info.Description, m.Description)
			}
			if info.FormatVersion != m.FormatVersion {
				t.Errorf("FormatVersion = %d, Open says %d", info.FormatVersion, m.FormatVersion)
			}

			// ...and against the values asked for, so the two agreeing on a wrong
			// answer would still fail.
			if info.Width != tc.w || info.Height != tc.h {
				t.Errorf("size = %dx%d, want %dx%d", info.Width, info.Height, tc.w, tc.h)
			}
			if info.Name != tc.name {
				t.Errorf("Name = %q, want %q", info.Name, tc.name)
			}
			if info.Description != tc.desc {
				t.Errorf("Description = %q, want %q", info.Description, tc.desc)
			}
		})
	}
}

// TestOpenInfoSucceedsWhereOpenFails is the whole reason the entry point exists:
// a map whose metadata reads cleanly but which does not fully decode is a state
// a caller must be able to observe. Through Open alone it is indistinguishable
// from "this file is not a map".
func TestOpenInfoSucceedsWhereOpenFails(t *testing.T) {
	const w, h = 6, 4

	// A type-1 payload that is well formed as bytes but inconsistent with W*H.
	// Both a short and a long one, since the size check is an equality.
	for _, tc := range []struct {
		label string
		cells int
	}{
		{"grid too small", w*h - 1},
		{"grid too large", w*h + 3},
	} {
		t.Run(tc.label, func(t *testing.T) {
			data := wellFramed(t, w, h, "Half Decoded", "", make([]byte, 2*tc.cells))

			// The file must still be well framed, or this would be testing the
			// framing check instead of the grid check.
			assertTilesToEOF(t, data)

			if _, err := alm.Open(data); err == nil {
				t.Fatalf("Open accepted a map whose type-1 grid is %d cells, want %d", tc.cells, w*h)
			}

			info, err := alm.OpenInfo(data)
			if err != nil {
				t.Fatalf("OpenInfo: %v, want success -- the metadata is intact", err)
			}
			if info.Width != w || info.Height != h {
				t.Errorf("size = %dx%d, want %dx%d", info.Width, info.Height, w, h)
			}
			if info.Name != "Half Decoded" {
				t.Errorf("Name = %q, want %q", info.Name, "Half Decoded")
			}
		})
	}
}

// TestOpenInfoRejectsWhatOpenRejects covers the framing faults. OpenInfo shares
// the record walk, so it must reject every one of them -- it is weaker than Open
// only after the framing holds, never before it.
func TestOpenInfoRejectsWhatOpenRejects(t *testing.T) {
	valid := wellFramed(t, 4, 4, "Valid", "", nil)

	mutate := func(fn func(b []byte)) []byte {
		b := append([]byte(nil), valid...)
		fn(b)
		return b
	}

	cases := []struct {
		label string
		data  []byte
	}{
		{"empty", nil},
		{"shorter than the file header", valid[:10]},
		// Truncation is the case worth being explicit about: a physically short
		// file fails BOTH, because the walk requires every record it counts to
		// lie in bounds. "OpenInfo is weaker" does not mean "OpenInfo salvages a
		// damaged file". (Bytes AFTER the last record are a different matter and
		// are now accepted by both — the loader never reaches them.)
		{"truncated mid-file", valid[:len(valid)-1]},
		{"bad magic", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[0x00:], 0xdeadbeef) })},
		{"wrong file header length", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[0x04:], 24) })},
		{"record count below the minimum of 3", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[0x0c:], 2) })},
		{"a counted record that is not there", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[0x0c:], 11) })},
		{"header-skipping format version", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[0x10:], 1000) })},
		{"format version above the gate", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[0x10:], 1002) })},
		{"bad record tag", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[20+0x00:], 6) })},
		{"bad record header length", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[20+0x04:], 16) })},
		// Retyping the first record to 10 does not fail because 10 is out of
		// range — that is skipped now — but because it takes the type-0 record
		// away, and this reader requires one (ALM-ORD-057).
		{"the type-0 record retyped away", mutate(func(b []byte) { binary.LittleEndian.PutUint32(b[20+0x0c:], 10) })},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			_, openErr := alm.Open(tc.data)
			info, infoErr := alm.OpenInfo(tc.data)

			if openErr == nil {
				t.Fatalf("Open accepted it; the fixture does not exercise what it claims to")
			}
			if infoErr == nil {
				t.Fatalf("OpenInfo accepted a stream Open rejects (%v)", openErr)
			}
			if info != nil {
				t.Errorf("OpenInfo returned a non-nil Info alongside an error")
			}
		})
	}
}

// TestOpenInfoRejectsBadMetadata covers the one non-framing fault OpenInfo can
// still hit: the type-0 payload itself being the wrong size.
func TestOpenInfoRejectsBadMetadata(t *testing.T) {
	valid := wellFramed(t, 4, 4, "Valid", "", nil)

	// Shorten the type-0 payload and its declared size together, so the file
	// still tiles to EOF and the fault is the payload length alone.
	const type0Start = 20 // the first record header follows the 20-byte file header
	size := binary.LittleEndian.Uint32(valid[type0Start+0x08:])
	if size != 632 {
		t.Fatalf("fixture type-0 payload is %d bytes, expected 632", size)
	}
	data := append([]byte(nil), valid[:type0Start+0x08]...)
	data = append(data, 0, 0, 0, 0)
	data = append(data, valid[type0Start+0x0c:type0Start+20]...)
	data = append(data, valid[type0Start+20+int(size):]...)
	binary.LittleEndian.PutUint32(data[type0Start+0x08:], 0)

	assertTilesToEOF(t, data)

	if _, err := alm.OpenInfo(data); err == nil {
		t.Errorf("OpenInfo accepted a zero-length type-0 payload")
	}
	if _, err := alm.Open(data); err == nil {
		t.Errorf("Open accepted a zero-length type-0 payload")
	}
}

// assertTilesToEOF walks the record headers independently of the package under
// test and fails unless the ten records end exactly at EOF. Without it, a fixture
// meant to exercise a later decode stage could silently be exercising the framing
// check instead.
func assertTilesToEOF(t *testing.T, data []byte) {
	t.Helper()
	if len(data) < 20 {
		t.Fatalf("fixture is %d bytes, too short to be well framed", len(data))
	}
	if n := binary.LittleEndian.Uint32(data[0x0c:]); n != 10 {
		t.Fatalf("fixture declares %d records, want 10", n)
	}
	cursor := 20
	for i := 0; i < 10; i++ {
		if cursor+20 > len(data) {
			t.Fatalf("fixture record %d header runs past EOF", i)
		}
		size := int(binary.LittleEndian.Uint32(data[cursor+0x08:]))
		cursor += 20 + size
		if cursor > len(data) {
			t.Fatalf("fixture record %d payload runs past EOF", i)
		}
	}
	if cursor != len(data) {
		t.Fatalf("fixture records end at %d, not EOF %d -- it is not well framed", cursor, len(data))
	}
}

// The words are written at +0x70 and +0x74 beside a distinct neighbour.
func TestOpenInfoCarriesTheTwoMapListWords(t *testing.T) {
	data := wellFramed(t, 8, 5, "WordProbe", "", nil)
	at := bytes.Index(data, []byte("WordProbe"))
	if at < 0x30 {
		t.Fatal("name not found in the type-0 payload")
	}
	payload := at - 0x30
	binary.LittleEndian.PutUint32(data[payload+0x6c:], 0xdead)
	binary.LittleEndian.PutUint32(data[payload+0x70:], 7)
	binary.LittleEndian.PutUint32(data[payload+0x74:], 9)
	info, err := alm.OpenInfo(data)
	if err != nil {
		t.Fatal(err)
	}
	if info.Word70 != 7 || info.Word74 != 9 {
		t.Fatalf("words = %d, %d, want 7, 9", info.Word70, info.Word74)
	}
}

func TestOpenInfoListDescriptionReadsTheWholeBlock(t *testing.T) {
	data := wellFramed(t, 8, 5, "ListProbe", "", nil)
	at := bytes.Index(data, []byte("ListProbe"))
	if at < 0x30 {
		t.Fatal("name not found in the type-0 payload")
	}
	payload := at - 0x30
	first := strings.Repeat("a", 60)
	text := first + "\nsecond line, past the 64-byte field"
	copy(data[payload+0x78:payload+0x78+512], make([]byte, 512))
	copy(data[payload+0x78:], text)
	info, err := alm.OpenInfo(data)
	if err != nil {
		t.Fatal(err)
	}
	if want := first + "#second line, past the 64-byte field"; info.ListDescription != want {
		t.Fatalf("ListDescription = %q, want %q", info.ListDescription, want)
	}
	if len(info.Description) != 64 {
		t.Fatalf("Description is %d bytes, want the 64-byte field", len(info.Description))
	}
	copy(data[payload+0x78:], bytes.Repeat([]byte("b"), 512))
	if info, err = alm.OpenInfo(data); err != nil || len(info.ListDescription) != 512 {
		t.Fatalf("unterminated block: len %d, %v", len(info.ListDescription), err)
	}
}
