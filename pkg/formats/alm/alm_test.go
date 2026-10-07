package alm_test

// These tests are derived exclusively from the ALM container SPECIFICATION
// (docs/0003-alm-container/spec.md) and the public API of package alm. No test
// reads any file from disk; every fixture is a synthetic byte stream assembled
// in test code from the documented byte layout.

import (
	"bytes"
	"math"
	"testing"

	"againrom/pkg/formats/alm"
)

// ---------------------------------------------------------------------------
// Fixture primitives
// ---------------------------------------------------------------------------

const almMagic uint32 = 0x0052374D // "M7R\0"

// perMapConstBits is an arbitrary per-map f32 (spec's example bit pattern),
// written byte-identically into every record header of a map.
const perMapConstBits uint32 = 0xBFC02B6D

// testAngle is the stored type-0 angle (payload +0x08). Exact in float32.
var testAngle = float32(0.5)

// physicalOrder is the corpus-observed physical record order (spec §Record roster).
var physicalOrder = []uint32{0, 1, 2, 3, 5, 4, 9, 8, 6, 7}

func le16(v uint16) []byte { return []byte{byte(v), byte(v >> 8)} }

func le32(v uint32) []byte {
	return []byte{byte(v), byte(v >> 8), byte(v >> 16), byte(v >> 24)}
}

func lef32(f float32) []byte { return le32(math.Float32bits(f)) }

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// ---------------------------------------------------------------------------
// File header (20 bytes)
// ---------------------------------------------------------------------------

func fileHeader(magic, hdrLen, dataSize, recordCount, formatVersion uint32) []byte {
	return concat(le32(magic), le32(hdrLen), le32(dataSize), le32(recordCount), le32(formatVersion))
}

// stdFileHeader is a valid 20-byte file header (dataSize is ignored by the reader).
func stdFileHeader() []byte {
	return fileHeader(almMagic, 20, 999, 10, 990)
}

// ---------------------------------------------------------------------------
// Record header (20 bytes) + payload
// ---------------------------------------------------------------------------

// recordHdr builds a raw 20-byte record header with fully explicit fields.
func recordHdr(tag, hdrLen, payloadSize, typeID, perMapBits uint32) []byte {
	return concat(le32(tag), le32(hdrLen), le32(payloadSize), le32(typeID), le32(perMapBits))
}

// rec wraps a payload in a well-formed record header (tag=7, hdrLen=20,
// payloadSize=len(payload)).
func rec(typeID uint32, payload []byte) []byte {
	return concat(recordHdr(7, 20, uint32(len(payload)), typeID, perMapConstBits), payload)
}

// recBadSize wraps a payload but lies about its declared payloadSize.
func recBadSize(typeID, declaredSize uint32, payload []byte) []byte {
	return concat(recordHdr(7, 20, declaredSize, typeID, perMapConstBits), payload)
}

// recCustom wraps a payload with a custom tag/hdrLen (payloadSize stays honest).
func recCustom(tag, hdrLen, typeID uint32, payload []byte) []byte {
	return concat(recordHdr(tag, hdrLen, uint32(len(payload)), typeID, perMapConstBits), payload)
}

// ---------------------------------------------------------------------------
// File assembly
// ---------------------------------------------------------------------------

// physIndex returns the physical position of a typeId within physicalOrder.
func physIndex(tid uint32) int {
	for i, t := range physicalOrder {
		if t == tid {
			return i
		}
	}
	return -1
}

// wrapAll wraps each typeId's payload as a record in physical order.
func wrapAll(payload map[uint32][]byte) [][]byte {
	blobs := make([][]byte, 0, len(physicalOrder))
	for _, tid := range physicalOrder {
		blobs = append(blobs, rec(tid, payload[tid]))
	}
	return blobs
}

// buildFile prepends a file header to a list of pre-wrapped record blobs.
func buildFile(hdr []byte, blobs [][]byte) []byte {
	return concat(append([][]byte{hdr}, blobs...)...)
}

// buildMap assembles a standard-header file from a typeId->payload map.
func buildMap(payload map[uint32][]byte) []byte {
	return buildFile(stdFileHeader(), wrapAll(payload))
}

// buildPartial assembles a file carrying exactly the typeIds listed, in that
// physical order, with the file header's recordCount set to match. It is how a
// map with sections absent is built: the shipped writer emits all ten, the
// loader requires three, and everything between is a stream this reader must
// take a position on.
func buildPartial(payload map[uint32][]byte, order ...uint32) []byte {
	blobs := make([][]byte, 0, len(order))
	for _, tid := range order {
		blobs = append(blobs, rec(tid, payload[tid]))
	}
	return buildFile(fileHeader(almMagic, 20, 999, uint32(len(order)), 990), blobs)
}

// ---------------------------------------------------------------------------
// type-0 metadata payload (632 bytes)
// ---------------------------------------------------------------------------

// buildMeta builds a 632-byte type-0 payload. name is ASCII; desc is written
// as raw bytes (so CP1251 high bytes can be injected without literal text).
func buildMeta(w, h, count5, count4, count6 uint32, name string, desc []byte) []byte {
	p := make([]byte, 632)
	copy(p[0x00:], le32(w))
	copy(p[0x04:], le32(h))
	copy(p[0x08:], lef32(testAngle))
	// 0x0c..0x14: located scalars (raw). 0x18: low-bit bitmask (raw).
	copy(p[0x0c:], le32(0x0C0C0C0C))
	copy(p[0x10:], le32(0x10101010))
	copy(p[0x14:], le32(0x14141414))
	copy(p[0x18:], le32(0x00001abc))
	copy(p[0x1c:], le32(count5)) // #type5
	copy(p[0x20:], le32(count4)) // #type4
	copy(p[0x24:], le32(count6)) // #type6
	copy(p[0x28:], le32(0x28282828))
	copy(p[0x2c:], le32(0x2c2c2c2c))
	copy(p[0x30:], []byte(name)) // char[64], NUL-padded by the zeroed buffer
	copy(p[0x70:], le32(0x70707070))
	copy(p[0x74:], le32(0x74747474))
	copy(p[0x78:], desc) // char[64] CP1251, NUL-padded by the zeroed buffer
	// 0xb8..632: 448-byte trailing slots left zeroed (raw).
	return p
}

// ---------------------------------------------------------------------------
// Grid payload builders
// ---------------------------------------------------------------------------

// tileBytes serializes u16 tile cells (type1), little-endian.
func tileBytes(cells ...uint16) []byte {
	var b []byte
	for _, c := range cells {
		b = append(b, le16(c)...)
	}
	return b
}

// bytesN returns a slice of n bytes filled with a ramp starting at base.
func bytesN(n int, base byte) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = base + byte(i)
	}
	return b
}

// ---------------------------------------------------------------------------
// Content-section record builders
// ---------------------------------------------------------------------------

// type4Base builds a 20-byte base object record.
func type4Base(x, y, kind uint32, f0c uint16, f0e uint32, f12 uint16) []byte {
	r := make([]byte, 20)
	copy(r[0x00:], le32(x))
	copy(r[0x04:], le32(y))
	copy(r[0x08:], le32(kind))
	copy(r[0x0c:], le16(f0c))
	copy(r[0x0e:], le32(f0e))
	copy(r[0x12:], le16(f12))
	return r
}

// type5Record builds a 76-byte roster record: scalar @+0x08, NUL-term name @+0x0c,
// and the sixteen relation words @+0x2c all zero.
func type5Record(scalar uint32, name string) []byte {
	return type5RecordRel(scalar, name, [16]uint16{})
}

// type5RecordRel is type5Record with the sixteen u16 at +0x2c written from row,
// element k at +0x2c+2k. The last word ends at 0x4c = 76, so a record built here
// is exactly full.
func type5RecordRel(scalar uint32, name string, row [16]uint16) []byte {
	r := make([]byte, 76)
	copy(r[0x08:], le32(scalar))
	copy(r[0x0c:], []byte(name))
	for k, w := range row {
		copy(r[0x2c+2*k:], le16(w))
	}
	return r
}

// type6Record builds a 70-byte unit record: X @+0x00, Y @+0x04, the class keys
// @+0x08 (primary, written as a raw u16 bit pattern) and @+0x0a, and the two
// override words Flags @+0x0c and DefID @+0x10. The ordinary helper writes the
// +0x20 signed current-health sentinel -1; explicit-health tests use the second
// helper so zero cannot be confused with unspecified fixture bytes.
func type6Record(x, y uint32, classBits, classSubID uint16, flags, defID uint32) []byte {
	return type6RecordHealth(x, y, classBits, classSubID, flags, defID, -1)
}

func type6RecordHealth(x, y uint32, classBits, classSubID uint16, flags, defID uint32, currentHP int16) []byte {
	r := make([]byte, 70)
	copy(r[0x00:], le32(x))
	copy(r[0x04:], le32(y))
	copy(r[0x08:], le16(classBits))
	copy(r[0x0a:], le16(classSubID))
	copy(r[0x0c:], le32(flags))
	copy(r[0x10:], le32(defID))
	copy(r[0x20:], le16(uint16(currentHP)))
	return r
}

// countBody builds a `[u32 count][body]` payload (type7 / type9).
func countBody(count uint32, body []byte) []byte {
	return concat(le32(count), body)
}

// ---------------------------------------------------------------------------
// A valid baseline map, parameterised by dimensions and content counts.
// ---------------------------------------------------------------------------

// baseSections returns valid payloads for all ten typeIds of a W x H map with
// the given content counts. All type4 records are non-extension (kind != 0x21),
// so the type4 payload is exactly 20*count4 bytes.
func baseSections(w, h, count5, count4, count6 uint32) map[uint32][]byte {
	m := map[uint32][]byte{}
	m[0] = buildMeta(w, h, count5, count4, count6, "map", []byte("desc"))
	m[1] = bytesN(int(2*w*h), 0x10) // tiles: 2*W*H bytes
	m[2] = bytesN(int(w*h), 0x20)   // altitudes: W*H bytes
	m[3] = bytesN(int(w*h), 0x00)   // overlay: W*H bytes

	var t4 []byte
	for i := uint32(0); i < count4; i++ {
		t4 = concat(t4, type4Base(0x0100+i, 0x0200+i, 1, 0, 0, 0))
	}
	m[4] = t4

	var t5 []byte
	for i := uint32(0); i < count5; i++ {
		t5 = concat(t5, type5Record(0, "g"))
	}
	m[5] = t5

	var t6 []byte
	for i := uint32(0); i < count6; i++ {
		t6 = concat(t6, type6Record(0x0180+i, 0x0280+i, 0, 0, 0, 0))
	}
	m[6] = t6

	m[7] = countBody(0, nil) // 4 bytes
	m[8] = nil               // empty, legal
	m[9] = countBody(0, nil) // 4 bytes
	return m
}

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

func openOK(t *testing.T, data []byte) *alm.Map {
	t.Helper()
	m, err := alm.Open(data)
	if err != nil {
		t.Fatalf("Open: unexpected error: %v", err)
	}
	if m == nil {
		t.Fatalf("Open: nil map with nil error")
	}
	return m
}

func openReject(t *testing.T, data []byte) {
	t.Helper()
	m, err := alm.Open(data)
	if err == nil {
		t.Fatalf("Open: expected error, got nil")
	}
	if m != nil {
		t.Fatalf("Open: expected nil map on rejection, got non-nil")
	}
}

// recordByType finds the (single) record carrying a given typeId.
func recordByType(m *alm.Map, tid uint32) (alm.Record, bool) {
	for _, r := range m.Records {
		if r.TypeID == tid {
			return r, true
		}
	}
	return alm.Record{}, false
}

func TestDecodeMinimalMap(t *testing.T) {
	// Distinct, non-overlay grid cells (grids are pure: first cell is real data).
	tiles := []uint16{0x2005, 0x0006, 0x0007, 0x2008}
	alts := []uint8{10, 20, 30, 40}
	overlay := []uint8{0, 1, 2, 3}

	p := map[uint32][]byte{}
	p[0] = buildMeta(2, 2, 1, 1, 1, "MiniMap", []byte("Hello"))
	p[1] = tileBytes(tiles...)
	p[2] = []byte{alts[0], alts[1], alts[2], alts[3]}
	p[3] = []byte{overlay[0], overlay[1], overlay[2], overlay[3]}
	p[4] = type4Base(0x0180, 0x0280, 1, 0xAAAA, 0xBBBBBBBB, 0xCCCC)
	p[5] = type5Record(5000, "Self")
	p[6] = type6Record(0x0180, 0x0280, 0, 0, 0, 0)
	p[7] = countBody(0, nil)
	p[8] = nil
	p[9] = countBody(0, nil)

	m := openOK(t, buildMap(p))

	if m.HdrLen != 20 {
		t.Errorf("HdrLen = %d, want 20", m.HdrLen)
	}
	if m.RecordCount != 10 {
		t.Errorf("RecordCount = %d, want 10", m.RecordCount)
	}
	if m.FormatVersion != 990 {
		t.Errorf("FormatVersion = %d, want 990", m.FormatVersion)
	}

	// Metadata.
	if m.Width != 2 || m.Height != 2 {
		t.Errorf("dims = %dx%d, want 2x2", m.Width, m.Height)
	}
	if m.Name != "MiniMap" {
		t.Errorf("Name = %q, want %q", m.Name, "MiniMap")
	}
	if m.Description != "Hello" {
		t.Errorf("Description = %q, want %q", m.Description, "Hello")
	}
	if m.Meta.Count5 != 1 || m.Meta.Count4 != 1 || m.Meta.Count6 != 1 {
		t.Errorf("Meta counts = 5:%d 4:%d 6:%d, want 1/1/1",
			m.Meta.Count5, m.Meta.Count4, m.Meta.Count6)
	}
	if m.Angle != testAngle {
		t.Errorf("Angle = %v, want %v", m.Angle, testAngle)
	}
	if want := uint32(perMapConstBits); m.MetaRecordWord != want {
		t.Errorf("MetaRecordWord = %#08x, want %#08x", m.MetaRecordWord, want)
	}

	// Grids: exactly W*H cells, exact written values (no overlay).
	if len(m.Tiles) != 4 {
		t.Fatalf("len(Tiles) = %d, want 4", len(m.Tiles))
	}
	for i, want := range tiles {
		if m.Tiles[i] != want {
			t.Errorf("Tiles[%d] = 0x%04x, want 0x%04x", i, m.Tiles[i], want)
		}
	}
	if len(m.Altitudes) != 4 {
		t.Fatalf("len(Altitudes) = %d, want 4", len(m.Altitudes))
	}
	for i, want := range alts {
		if m.Altitudes[i] != want {
			t.Errorf("Altitudes[%d] = %d, want %d", i, m.Altitudes[i], want)
		}
	}
	if len(m.Overlay) != 4 {
		t.Fatalf("len(Overlay) = %d, want 4", len(m.Overlay))
	}
	for i, want := range overlay {
		if m.Overlay[i] != want {
			t.Errorf("Overlay[%d] = %d, want %d", i, m.Overlay[i], want)
		}
	}

	// Public tile accessors on the first cell (0x2005).
	if alm.TileIndex(m.Tiles[0]) != 5 {
		t.Errorf("TileIndex(0x2005) = %d, want 5", alm.TileIndex(m.Tiles[0]))
	}
	if !alm.Impassable(m.Tiles[0]) {
		t.Errorf("Impassable(0x2005) = false, want true")
	}
	// A non-impassable cell (0x0006).
	if alm.TileIndex(m.Tiles[1]) != 6 || alm.Impassable(m.Tiles[1]) {
		t.Errorf("cell 0x0006: idx=%d imp=%v, want 6/false",
			alm.TileIndex(m.Tiles[1]), alm.Impassable(m.Tiles[1]))
	}

	// Ten records, every typeId present exactly once (indexed by header typeId).
	if len(m.Records) != 10 {
		t.Fatalf("len(Records) = %d, want 10", len(m.Records))
	}
	for tid := uint32(0); tid <= 9; tid++ {
		if _, ok := recordByType(m, tid); !ok {
			t.Errorf("record with TypeID %d missing", tid)
		}
	}
	// Record payload sizes by typeId.
	if r, _ := recordByType(m, 0); r.PayloadSize != 632 {
		t.Errorf("type0 PayloadSize = %d, want 632", r.PayloadSize)
	}
	if r, _ := recordByType(m, 1); r.PayloadSize != 8 {
		t.Errorf("type1 PayloadSize = %d, want 8", r.PayloadSize)
	}
	if r, _ := recordByType(m, 2); r.PayloadSize != 4 {
		t.Errorf("type2 PayloadSize = %d, want 4", r.PayloadSize)
	}

	// Content section lengths.
	if len(m.Objects) != 1 {
		t.Errorf("len(Objects) = %d, want 1", len(m.Objects))
	}
	if len(m.Groups) != 1 {
		t.Errorf("len(Groups) = %d, want 1", len(m.Groups))
	}
	if len(m.Units) != 1 {
		t.Errorf("len(Units) = %d, want 1", len(m.Units))
	}
	if m.Groups[0].Name != "Self" {
		t.Errorf("Groups[0].Name = %q, want %q", m.Groups[0].Name, "Self")
	}
}

// ---------------------------------------------------------------------------
// AC-2 — reject bad file headers.
// ---------------------------------------------------------------------------

func TestRejectBadHeader(t *testing.T) {
	blobs := wrapAll(baseSections(2, 2, 0, 0, 0))

	t.Run("wrong magic", func(t *testing.T) {
		hdr := fileHeader(0xDEADBEEF, 20, 999, 10, 990)
		openReject(t, buildFile(hdr, blobs))
	})
	t.Run("hdrLen != 20", func(t *testing.T) {
		hdr := fileHeader(almMagic, 24, 999, 10, 990)
		openReject(t, buildFile(hdr, blobs))
	})
	t.Run("recordCount below the loader's minimum of 3", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		openReject(t, buildPartial(p, 0, 1))
	})
	t.Run("formatVersion == 1000", func(t *testing.T) {
		// Refused as an UNIMPLEMENTED dialect, not as an invalid one: the
		// loader supports it by skipping every record header, which is a second
		// framing and no shipped map exercises it.
		hdr := fileHeader(almMagic, 20, 999, 10, 1000)
		openReject(t, buildFile(hdr, blobs))
	})
	t.Run("formatVersion above the loader's gate of 1001", func(t *testing.T) {
		hdr := fileHeader(almMagic, 20, 999, 10, 1002)
		openReject(t, buildFile(hdr, blobs))
	})
}

// ---------------------------------------------------------------------------
// The widened acceptance: what the loader takes, this reader takes.
//
// Each case is a stream the engine's loader accepts and the pre-widening reader
// refused. The behaviour asserted for each is the loader's own, per ALM-REQ-055
// and ALM-REQ-056 — not a convenience this reader invented.
// ---------------------------------------------------------------------------

func TestAcceptsWhatTheLoaderAccepts(t *testing.T) {
	t.Run("three records, and the type-3 plane is manufactured", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		m := openOK(t, buildPartial(p, 0, 1, 2))

		if m.RecordCount != 3 || len(m.Records) != 3 {
			t.Errorf("RecordCount = %d, len(Records) = %d, want 3 and 3", m.RecordCount, len(m.Records))
		}
		if got := len(m.Overlay); got != 4 {
			t.Fatalf("len(Overlay) = %d, want 4 (W*H) — a short plane silently disables every consumer that sizes by it", got)
		}
		for i, b := range m.Overlay {
			if b != 0 {
				t.Errorf("Overlay[%d] = %d, want 0: the manufactured plane is all-zero", i, b)
			}
		}
		if m.Present(3) {
			t.Error("Present(3) is true, but no type-3 record was in the file")
		}
		for _, tid := range []uint32{0, 1, 2} {
			if !m.Present(tid) {
				t.Errorf("Present(%d) is false, but the record was in the file", tid)
			}
		}
		if len(m.Objects) != 0 || len(m.Groups) != 0 || len(m.Units) != 0 {
			t.Errorf("absent content records decoded to %d objects, %d groups, %d units, want none",
				len(m.Objects), len(m.Groups), len(m.Units))
		}
		if m.Triggers.EntryCount != 0 || len(m.Triggers.Body) != 0 || m.TileMarkers.Count != 0 {
			t.Error("an absent type-7/type-9 decoded to something other than a zero count and an empty body")
		}
	})

	t.Run("four records, and the type-3 plane is read", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		p[3] = []byte{0x00, 0x07, 0x00, 0x09}
		m := openOK(t, buildPartial(p, 0, 1, 2, 3))

		if !m.Present(3) {
			t.Error("Present(3) is false, but a type-3 record was in the file")
		}
		if !bytes.Equal(m.Overlay, []byte{0x00, 0x07, 0x00, 0x09}) {
			t.Errorf("Overlay = %v, want the record's own cells", m.Overlay)
		}
	})

	t.Run("type-0 counts naming records the file does not carry are inert", func(t *testing.T) {
		// The shipped four-record map advertises type-4/5/6 records it does not
		// contain (ALM-CORP-060). Each count is only its own case's loop bound,
		// and that case never runs, so the file loads and the sections are empty.
		p := baseSections(2, 2, 7, 415, 1815)
		m := openOK(t, buildPartial(p, 0, 1, 2, 3))

		if m.Meta.Count4 != 415 || m.Meta.Count5 != 7 || m.Meta.Count6 != 1815 {
			t.Error("the type-0 count words were not decoded as written")
		}
		if len(m.Objects) != 0 || len(m.Groups) != 0 || len(m.Units) != 0 {
			t.Error("a count word conjured records the file does not carry")
		}
	})

	t.Run("a typeId at or above 10 is stepped over", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		blobs := wrapAll(p)
		blobs = append(blobs, rec(10, []byte{0xAA, 0xBB}))
		data := buildFile(fileHeader(almMagic, 20, 999, 11, 990), blobs)

		m := openOK(t, data)
		if len(m.Records) != 11 {
			t.Errorf("len(Records) = %d, want 11: an unhandled record is still part of the frame", len(m.Records))
		}
		if m.Present(10) {
			t.Error("Present(10) is true, but no typeId at or above 10 can be decoded from")
		}
	})

	t.Run("a repeated typeId is last-wins", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		blobs := wrapAll(p)
		second := []byte{0x01, 0x02, 0x03, 0x04}
		blobs = append(blobs, rec(3, second)) // a second type-3, after the first
		data := buildFile(fileHeader(almMagic, 20, 999, 11, 990), blobs)

		m := openOK(t, data)
		if !bytes.Equal(m.Overlay, second) {
			t.Errorf("Overlay = %v, want the LAST type-3 record's cells %v", m.Overlay, second)
		}
	})

	t.Run("bytes after the last record are not read", func(t *testing.T) {
		data := append(buildMap(baseSections(2, 2, 0, 0, 0)), 0x00, 0x11, 0x22)
		m := openOK(t, data)
		if m.Width != 2 || m.Height != 2 {
			t.Error("a trailer disturbed the decode")
		}
	})

	t.Run("formatVersion at the gate", func(t *testing.T) {
		blobs := wrapAll(baseSections(2, 2, 0, 0, 0))
		m := openOK(t, buildFile(fileHeader(almMagic, 20, 999, 10, 1001), blobs))
		if m.FormatVersion != 1001 {
			t.Errorf("FormatVersion = %d, want 1001", m.FormatVersion)
		}
	})
}

// ---------------------------------------------------------------------------
// The two records the open path really requires, and the one this reader adds.
// ---------------------------------------------------------------------------

func TestRejectMissingRequiredRecords(t *testing.T) {
	p := baseSections(2, 2, 0, 0, 0)

	t.Run("no type-1 record", func(t *testing.T) {
		openReject(t, buildPartial(p, 0, 2, 3))
	})
	t.Run("no type-2 record", func(t *testing.T) {
		openReject(t, buildPartial(p, 0, 1, 3))
	})
	t.Run("no type-0 record", func(t *testing.T) {
		// Stricter than the loader on purpose: with no type-0 the engine reads
		// case 1's grid length out of a stack slot only case 0 writes, so the
		// file is undefined rather than defaulted (ALM-ORD-057).
		openReject(t, buildPartial(p, 1, 2, 3))
	})
}

func TestRejectRecordOutsideTheStream(t *testing.T) {
	t.Run("payloadSize overruns EOF", func(t *testing.T) {
		blobs := wrapAll(baseSections(2, 2, 0, 0, 0))
		// The physically-last record (type7) claims far more payload than present.
		body := countBody(0, nil)
		blobs[physIndex(7)] = recBadSize(7, uint32(len(body))+256, body)
		openReject(t, buildFile(stdFileHeader(), blobs))
	})
	t.Run("a counted record header past EOF", func(t *testing.T) {
		blobs := wrapAll(baseSections(2, 2, 0, 0, 0))
		// Eleven records counted, ten present: the eleventh header is not there.
		openReject(t, buildFile(fileHeader(almMagic, 20, 999, 11, 990), blobs))
	})
}

// ---------------------------------------------------------------------------
// AC-4 — reject a bad record header (tag != 7 or hdrLen != 20).
// ---------------------------------------------------------------------------

func TestRejectBadRecordHeader(t *testing.T) {
	t.Run("tag != 7", func(t *testing.T) {
		blobs := wrapAll(baseSections(2, 2, 0, 0, 0))
		// Rebuild the type3 record with a bad tag but honest size.
		p3 := baseSections(2, 2, 0, 0, 0)[3]
		blobs[physIndex(3)] = recCustom(6, 20, 3, p3)
		openReject(t, buildFile(stdFileHeader(), blobs))
	})
	t.Run("record hdrLen != 20", func(t *testing.T) {
		blobs := wrapAll(baseSections(2, 2, 0, 0, 0))
		p3 := baseSections(2, 2, 0, 0, 0)[3]
		blobs[physIndex(3)] = recCustom(7, 24, 3, p3)
		openReject(t, buildFile(stdFileHeader(), blobs))
	})
}

// ---------------------------------------------------------------------------
// AC-5 — reject a bad record roster.
// ---------------------------------------------------------------------------

func TestRejectRecordRoster(t *testing.T) {
	t.Run("type-0 payloadSize != 632", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		p[0] = make([]byte, 600) // wrong metadata length
		openReject(t, buildMap(p))
	})
}

func TestRejectGridSizes(t *testing.T) {
	t.Run("type1 size != 2*W*H", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		p[1] = bytesN(6, 0) // 6 != 2*2*2 = 8
		openReject(t, buildMap(p))
	})
	t.Run("type2 size != W*H", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		p[2] = bytesN(3, 0) // 3 != 2*2 = 4
		openReject(t, buildMap(p))
	})
	t.Run("W*H product overflows 64-bit", func(t *testing.T) {
		// W = H = 0xFFFFFFFF makes 2*W*H wrap a uint64. The reader must reject
		// atomically, with no panic and no oversized allocation.
		p := baseSections(2, 2, 0, 0, 0)
		p[0] = buildMeta(0xFFFFFFFF, 0xFFFFFFFF, 0, 0, 0, "", nil)
		p[1] = bytesN(8, 0) // tiny type1 payload vs. astronomical claimed grid
		openReject(t, buildMap(p))
	})
}

func TestRawBodiesRoundTrip(t *testing.T) {
	body7 := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00, 0x7F, 0x80, 0xFF}
	body8 := []byte{0x01, 0x02, 0xC0, 0xFE, 0x00, 0x00, 0xAB}
	body9 := []byte{0xCA, 0xFE, 0xBA, 0xBE, 0xFF}

	p := baseSections(2, 2, 0, 0, 0)
	p[7] = countBody(0x11223344, body7)
	p[8] = body8
	p[9] = countBody(0x55667788, body9)

	m := openOK(t, buildMap(p))

	if m.Triggers.EntryCount != 0x11223344 {
		t.Errorf("Triggers.EntryCount = 0x%08x, want 0x11223344", m.Triggers.EntryCount)
	}
	if !bytes.Equal(m.Triggers.Body, body7) {
		t.Errorf("Triggers.Body = % x, want % x", m.Triggers.Body, body7)
	}
	if !bytes.Equal(m.LootSection.Body, body8) {
		t.Errorf("LootSection.Body = % x, want % x", m.LootSection.Body, body8)
	}
	if m.TileMarkers.Count != 0x55667788 {
		t.Errorf("TileMarkers.Count = 0x%08x, want 0x55667788", m.TileMarkers.Count)
	}
	if !bytes.Equal(m.TileMarkers.Body, body9) {
		t.Errorf("TileMarkers.Body = % x, want % x", m.TileMarkers.Body, body9)
	}
}

// ---------------------------------------------------------------------------
// AC-8 — decode ASCII name and CP1251 description (no literal Cyrillic).
// ---------------------------------------------------------------------------

func TestDecodeNameAndCP1251Description(t *testing.T) {
	// description bytes: 0x41 ('A') then 0xC0. In Windows-1251, 0xC0 -> U+0410.
	desc := []byte{0x41, 0xC0}

	p := baseSections(2, 2, 0, 0, 0)
	p[0] = buildMeta(2, 2, 0, 0, 0, "Level", desc)

	m := openOK(t, buildMap(p))

	if m.Name != "Level" {
		t.Errorf("Name = %q, want %q", m.Name, "Level")
	}
	want := "A" + string(rune(0x0410))
	if m.Description != want {
		t.Errorf("Description = %q (% x), want %q (% x)",
			m.Description, []byte(m.Description), want, []byte(want))
	}
}

func TestType5Roster(t *testing.T) {
	t.Run("two records decode", func(t *testing.T) {
		p := baseSections(2, 2, 2, 0, 0)
		p[5] = concat(type5Record(0, "Self"), type5Record(5000, "Enemy"))

		m := openOK(t, buildMap(p))
		if len(m.Groups) != 2 {
			t.Fatalf("len(Groups) = %d, want 2", len(m.Groups))
		}
		if m.Groups[0].Name != "Self" || m.Groups[0].Scalar != 0 {
			t.Errorf("Groups[0] = {%q, %d}, want {Self, 0}", m.Groups[0].Name, m.Groups[0].Scalar)
		}
		if m.Groups[1].Name != "Enemy" || m.Groups[1].Scalar != 5000 {
			t.Errorf("Groups[1] = {%q, %d}, want {Enemy, 5000}", m.Groups[1].Name, m.Groups[1].Scalar)
		}
	})
	t.Run("payloadSize != 76*#5 rejects", func(t *testing.T) {
		p := baseSections(2, 2, 2, 0, 0) // meta declares #5 = 2
		p[5] = type5Record(0, "Self")    // but only one 76-byte record present
		openReject(t, buildMap(p))
	})

	// AC-15 (the file half). The sixteen u16 at +0x2c come back in record order
	// and at their own width. The row is deliberately NOT {0,1,2}: those are all
	// the shipped corpus carries, and a decoder that narrowed, masked or
	// re-based the words would agree with a corpus-shaped row and disagree here.
	t.Run("the relation row decodes in order and at full width", func(t *testing.T) {
		var row [16]uint16
		for k := range row {
			row[k] = uint16(0x1100 + k) // distinct, and every one above a byte
		}
		row[3] = 0xffff  // the widest word the field can hold
		row[7] = 0x0102  // low byte 2, high byte 1 — a narrowing would keep 2
		row[11] = 0x0200 // low byte 0 under a nonzero high byte

		p := baseSections(2, 2, 2, 0, 0)
		p[5] = concat(type5RecordRel(0, "Self", row), type5Record(5000, "Enemy"))

		m := openOK(t, buildMap(p))
		if len(m.Groups) != 2 {
			t.Fatalf("len(Groups) = %d, want 2", len(m.Groups))
		}
		if m.Groups[0].Relation != row {
			t.Errorf("Groups[0].Relation = %v, want %v", m.Groups[0].Relation, row)
		}
		// The second record's row is all zero, so a decoder reading the words at
		// a fixed offset from the payload rather than from the record would put
		// record 0's row here.
		if m.Groups[1].Relation != ([16]uint16{}) {
			t.Errorf("Groups[1].Relation = %v, want all zero", m.Groups[1].Relation)
		}
		// Reading sixteen words at +0x2c must not have disturbed the two fields
		// that were already there.
		if m.Groups[0].Name != "Self" || m.Groups[1].Name != "Enemy" || m.Groups[1].Scalar != 5000 {
			t.Errorf("the other roster fields moved: %+v / %+v", m.Groups[0], m.Groups[1])
		}
	})

	// The absent-record half of AC-15's file side: no type-5 record at all is
	// not a roster of empty rows, it is no roster.
	t.Run("an absent type-5 record decodes to no roster at all", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		m := openOK(t, buildPartial(p, 0, 1, 2, 3))
		if len(m.Groups) != 0 {
			t.Errorf("len(Groups) = %d with no type-5 record in the file, want 0", len(m.Groups))
		}
	})
}

func TestType6Units(t *testing.T) {
	// Record 1's primary key has the high bit set (0x8001): read as the loader
	// reads it (MOVSX) that is -32767; read as a u16 it would be 32769. The
	// shipped domain is 1..80, so only a hostile-looking value like this one can
	// tell the two readings apart.
	const highBitKey = 0x8001
	const highBitSigned = int16(-32767)

	t.Run("three records decode, primary key signed", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 3)
		p[6] = concat(
			// no override: DefID 0, Flags bit 0 clear.
			type6Record(0x0180, 0x0280, 0x0007, 0x0011, 0x00000000, 0x00000000),
			// high-bit primary key; DefID override (nonzero, != 0xcdcdcdcd).
			type6Record(0x0380, 0x0480, highBitKey, 0x0022, 0x00000004, 0x0000abcd),
			// Flags bit 0 set (NPC path); DefID at the 0xcdcdcdcd sentinel.
			type6Record(0x0580, 0x0680, 0x0050, 0x0033, 0x00000001, 0xcdcdcdcd),
		)

		m := openOK(t, buildMap(p))
		if len(m.Units) != 3 {
			t.Fatalf("len(Units) = %d, want 3", len(m.Units))
		}

		want := []alm.Unit{
			{X: 0x0180, Y: 0x0280, ClassID: 7, ClassSubID: 0x0011, Flags: 0x00000000, DefID: 0x00000000, CurrentHP: -1},
			{X: 0x0380, Y: 0x0480, ClassID: highBitSigned, ClassSubID: 0x0022, Flags: 0x00000004, DefID: 0x0000abcd, CurrentHP: -1},
			{X: 0x0580, Y: 0x0680, ClassID: 80, ClassSubID: 0x0033, Flags: 0x00000001, DefID: 0xcdcdcdcd, CurrentHP: -1},
		}
		for i, w := range want {
			if m.Units[i] != w {
				t.Errorf("Units[%d] = %+v, want %+v", i, m.Units[i], w)
			}
		}

		// The point of the high-bit record: the key must read negative, not ~65000.
		if got := m.Units[1].ClassID; got >= 0 {
			t.Errorf("Units[1].ClassID = %d, want negative (0x%04x read sign-extended = %d); "+
				"a non-negative value means the field is read unsigned", got, highBitKey, highBitSigned)
		}

		// Placement anchor -> tile cell is a bare >>8.
		if m.Units[0].X>>8 != 1 || m.Units[0].Y>>8 != 2 {
			t.Errorf("tile of Units[0] = (%d,%d), want (1,2)", m.Units[0].X>>8, m.Units[0].Y>>8)
		}
	})
	t.Run("payloadSize != 70*#6 rejects", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 2)                         // meta declares #6 = 2
		p[6] = type6Record(0x0180, 0x0280, 0x0007, 0, 0, 0x0000) // only one 70-byte record present
		openReject(t, buildMap(p))
	})
}

func TestUnitCurrentHealthSignedSentinelAndDocumentRoundTrip(t *testing.T) {
	want := []struct {
		raw     int16
		present bool
	}{{-1, false}, {0, true}, {-10, true}, {321, true}}

	var payload []byte
	for i, tc := range want {
		payload = concat(payload, type6RecordHealth(uint32(0x100+i), uint32(0x200+i), 7, 0, 0, 0, tc.raw))
	}
	p := baseSections(2, 2, 0, 0, uint32(len(want)))
	p[6] = payload
	raw := buildMap(p)
	doc := roundTrip(t, raw)
	if got := doc.Write(); !bytes.Equal(got, raw) {
		t.Fatal("decoding current health changed the document bytes")
	}
	m, err := doc.Map()
	if err != nil {
		t.Fatalf("Map(): %v", err)
	}
	for i, tc := range want {
		got := m.Units[i]
		if got.CurrentHP != tc.raw || got.HasCurrentHP != tc.present {
			t.Errorf("unit %d current health = %d/present %v, want %d/%v",
				i, got.CurrentHP, got.HasCurrentHP, tc.raw, tc.present)
		}
	}
}

func TestType4ExtensionWalk(t *testing.T) {
	ext := []byte{0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8}

	t.Run("extension attaches only to kind==0x21", func(t *testing.T) {
		base0 := type4Base(0x0180, 0x0280, 0x21, 0x1111, 0x22223333, 0x4444)
		base1 := type4Base(0x0380, 0x0480, 0x0007, 0x5555, 0x66667777, 0x8888)
		payload := concat(base0, ext, base1) // 20 + 8 + 20 = 48 bytes, consumed exactly

		p := baseSections(2, 2, 0, 2, 0) // #4 = 2
		p[4] = payload

		m := openOK(t, buildMap(p))
		if len(m.Objects) != 2 {
			t.Fatalf("len(Objects) = %d, want 2", len(m.Objects))
		}

		o0 := m.Objects[0]
		if o0.X != 0x0180 || o0.Y != 0x0280 || o0.Kind != 0x21 {
			t.Errorf("Objects[0] X/Y/Kind = 0x%x/0x%x/0x%x, want 0x180/0x280/0x21", o0.X, o0.Y, o0.Kind)
		}
		if o0.Field0C != 0x1111 || o0.Field0E != 0x22223333 || o0.Field12 != 0x4444 {
			t.Errorf("Objects[0] ctor fields = 0x%x/0x%x/0x%x, want 0x1111/0x22223333/0x4444",
				o0.Field0C, o0.Field0E, o0.Field12)
		}
		if o0.Ext == nil {
			t.Errorf("Objects[0].Ext = nil, want the 8 extension bytes")
		} else if !bytes.Equal(o0.Ext, ext) {
			t.Errorf("Objects[0].Ext = % x, want % x", o0.Ext, ext)
		}

		o1 := m.Objects[1]
		if o1.X != 0x0380 || o1.Y != 0x0480 || o1.Kind != 0x0007 {
			t.Errorf("Objects[1] X/Y/Kind = 0x%x/0x%x/0x%x, want 0x380/0x480/0x07", o1.X, o1.Y, o1.Kind)
		}
		if o1.Ext != nil {
			t.Errorf("Objects[1].Ext = % x, want nil", o1.Ext)
		}
	})

	t.Run("non-exact payload rejects", func(t *testing.T) {
		// #4 = 1 but 24 bytes present: walk consumes 20, leaves 4 -> reject.
		p := baseSections(2, 2, 0, 1, 0)
		p[4] = concat(type4Base(0x0100, 0x0200, 1, 0, 0, 0), []byte{0, 0, 0, 0})
		openReject(t, buildMap(p))
	})

	t.Run("hostile #4 count rejects without oversized alloc", func(t *testing.T) {
		p := baseSections(2, 2, 0, 1, 0)
		p[0] = buildMeta(2, 2, 0, 0x7FFFFFFF, 0, "map", []byte("desc")) // #4 astronomically large
		p[4] = type4Base(0x0100, 0x0200, 1, 0, 0, 0)                    // only one record present
		openReject(t, buildMap(p))
	})
}

// ---------------------------------------------------------------------------
// AC-13 — trigger sections: counts, raw bodies, empty type8.
// ---------------------------------------------------------------------------

func TestTriggerSectionsCountsAndRawBodies(t *testing.T) {
	t.Run("counts decode, bodies raw, empty type8 accepted", func(t *testing.T) {
		nodes := []byte{0x10, 0x20, 0x30, 0x40, 0x50}
		recs := []byte{0xA1, 0xB2, 0xC3}

		p := baseSections(2, 2, 0, 0, 0)
		p[7] = countBody(3, nodes)
		p[8] = nil // empty type8 (payloadSize 0)
		p[9] = countBody(7, recs)

		m := openOK(t, buildMap(p))
		if m.Triggers.EntryCount != 3 {
			t.Errorf("Triggers.EntryCount = %d, want 3", m.Triggers.EntryCount)
		}
		if !bytes.Equal(m.Triggers.Body, nodes) {
			t.Errorf("Triggers.Body = % x, want % x", m.Triggers.Body, nodes)
		}
		if m.TileMarkers.Count != 7 {
			t.Errorf("TileMarkers.Count = %d, want 7", m.TileMarkers.Count)
		}
		if !bytes.Equal(m.TileMarkers.Body, recs) {
			t.Errorf("TileMarkers.Body = % x, want % x", m.TileMarkers.Body, recs)
		}
		if len(m.LootSection.Body) != 0 {
			t.Errorf("len(LootSection.Body) = %d, want 0 (empty type8)", len(m.LootSection.Body))
		}
	})

	t.Run("type7 payload < 4 bytes rejects", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		p[7] = []byte{0x00, 0x01} // only 2 bytes: cannot hold the count word
		openReject(t, buildMap(p))
	})
}

// ---------------------------------------------------------------------------
// AC-15 — empty records are typed from their own header (no elimination).
// ---------------------------------------------------------------------------

func TestEmptyRecordsTypedFromHeader(t *testing.T) {
	t.Run("empty type8 typed from header", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0)
		p[8] = nil // payloadSize 0

		m := openOK(t, buildMap(p))

		// All ten ids present, in physical order (no elimination, no reorder).
		if len(m.Records) != 10 {
			t.Fatalf("len(Records) = %d, want 10", len(m.Records))
		}
		for i, want := range physicalOrder {
			if m.Records[i].TypeID != want {
				t.Errorf("Records[%d].TypeID = %d, want %d", i, m.Records[i].TypeID, want)
			}
		}
		r8, ok := recordByType(m, 8)
		if !ok {
			t.Fatalf("record with TypeID 8 missing")
		}
		if r8.PayloadSize != 0 {
			t.Errorf("type8 PayloadSize = %d, want 0", r8.PayloadSize)
		}
		if len(m.LootSection.Body) != 0 {
			t.Errorf("len(LootSection.Body) = %d, want 0", len(m.LootSection.Body))
		}
	})

	t.Run("empty type4 with #4=0 -> zero objects", func(t *testing.T) {
		p := baseSections(2, 2, 0, 0, 0) // #4 already 0, type4 payload already empty
		m := openOK(t, buildMap(p))
		if len(m.Objects) != 0 {
			t.Errorf("len(Objects) = %d, want 0", len(m.Objects))
		}
		if _, ok := recordByType(m, 4); !ok {
			t.Errorf("record with TypeID 4 missing")
		}
	})
}

// ---------------------------------------------------------------------------
// The placed record's OWNER SLOT at +0x14: decoded, at the record's own width,
// and costing the container nothing.
// ---------------------------------------------------------------------------

// TestUnitOwnerSlotIsDecodedAtItsOwnWidth is AC-1's first half.
//
// The four values are chosen so that no two can be confused and so that every
// claim about the field is exercised by one of them: 1 is the FIRST roster slot,
// the value the space begins at; 9 is the highest any shipped map authors; 0 is
// the value that names NO slot, which a decoder substituting a default would
// have to invent something for; and 0xffffffff is the top of the field, which a
// reader narrowing it to 16 or 8 bits could not carry.
//
// The words are written straight into the record at +0x14 rather than through a
// parameter of type6Record, so this fixture states the offset itself — a decoder
// reading the neighbouring DefID at +0x10 or the bounded index at +0x18 fails
// here rather than agreeing with a helper that moved with it.
func TestUnitOwnerSlotIsDecodedAtItsOwnWidth(t *testing.T) {
	want := []uint32{1, 9, 0, 0xffffffff}

	var t6 []byte
	for i, o := range want {
		r := type6Record(0x0180+uint32(i), 0x0280+uint32(i), 0, 0, 0, 0)
		copy(r[0x14:], le32(o))
		t6 = concat(t6, r)
	}
	p := baseSections(2, 2, 0, 0, uint32(len(want)))
	p[6] = t6
	data := buildMap(p)

	m := openOK(t, data)
	if len(m.Units) != len(want) {
		t.Fatalf("decoded %d units, want %d", len(m.Units), len(want))
	}
	for i, w := range want {
		if got := m.Units[i].Owner; got != w {
			t.Errorf("unit %d: Owner = %d, want %d (+0x14, u32)", i, got, w)
		}
	}

	// The DEFINITION ID at +0x10 and the bounded index at +0x18 sit either side
	// of the owner, and both are zero in this fixture — so a read one word out in
	// either direction gives zero for every record and the table above would pass
	// on its third case alone. Stated here rather than left to be noticed.
	for i := range m.Units {
		if m.Units[i].DefID != 0 {
			t.Errorf("unit %d: DefID = %d, want 0 — this fixture writes only the owner", i, m.Units[i].DefID)
		}
	}
}

// TestUnitOwnerSlotCostsTheDocumentNothing is AC-1's second half: reading one
// more word off a record changes nothing about the record, so a document opened
// and written back is byte-identical exactly as it was before this field was
// read. roundTrip asserts that equality itself and fails inside if it does not
// hold; what is added here is that Map() over the same bytes really does see the
// owners, so the round trip is over a fixture the field is present in.
func TestUnitOwnerSlotCostsTheDocumentNothing(t *testing.T) {
	r := type6Record(0x0180, 0x0280, 0, 0, 0, 0)
	copy(r[0x14:], le32(7))
	p := baseSections(2, 2, 0, 0, 1)
	p[6] = r

	doc := roundTrip(t, buildMap(p))
	m, err := doc.Map()
	if err != nil {
		t.Fatalf("Map(): %v", err)
	}
	if len(m.Units) != 1 || m.Units[0].Owner != 7 {
		t.Fatalf("the round-tripped document decodes to %+v, want one unit owned by slot 7", m.Units)
	}
}
