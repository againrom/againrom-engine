package alm_test

// Tests for the raw-backed document model (docs/0023-alm-roundtrip-writer).
// Every fixture is a synthetic byte stream assembled from the 0003 builders in
// alm_test.go; every accept/reject verdict below is written from the spec's
// own acceptance and rejection lists, never captured from Open. Each fixture
// is then asserted at BOTH entry points — Open and OpenDocument — so a drift
// in either shows as a red test, not as two wrong answers agreeing.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
)

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

// roundTrip pins one accept-side fixture. Its verdict — accepted — comes from
// the spec, so Open itself must accept it, OpenDocument must produce a
// document, and the unedited Write must reproduce the input byte-for-byte.
func roundTrip(t *testing.T, data []byte) *alm.Document {
	t.Helper()
	openOK(t, data) // the accept half of the decision, pinned on Open per fixture
	doc, err := alm.OpenDocument(data)
	if err != nil {
		t.Fatalf("OpenDocument: unexpected error: %v", err)
	}
	if doc == nil {
		t.Fatalf("OpenDocument: nil document with nil error")
	}
	out := doc.Write()
	if !bytes.Equal(out, data) {
		t.Fatalf("Write: output differs from input: %s", diffReport(out, data))
	}
	return doc
}

func docReject(t *testing.T, data []byte) {
	t.Helper()
	doc, err := alm.OpenDocument(data)
	if err == nil {
		t.Fatalf("OpenDocument: expected error, got nil")
	}
	if doc != nil {
		t.Fatalf("OpenDocument: expected nil document on rejection, got non-nil")
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("OpenDocument: error %q wraps nothing", err)
	}
}

// diffReport describes the first divergence between a written stream and its
// input, for failure messages only.
func diffReport(got, want []byte) string {
	n := len(got)
	if len(want) < n {
		n = len(want)
	}
	for i := 0; i < n; i++ {
		if got[i] != want[i] {
			return fmt.Sprintf("first difference at offset %#x: got %#02x, want %#02x", i, got[i], want[i])
		}
	}
	return fmt.Sprintf("lengths differ: got %d, want %d bytes", len(got), len(want))
}

// ---------------------------------------------------------------------------
// AC-1 (round-trip and view clauses), SC-1 — the minimal accepted map.
// ---------------------------------------------------------------------------

// minimalDocMap builds AC-1's map: ten records in the shipped physical order,
// small grids, and one type-4 record carrying the kind==0x21 extension.
func minimalDocMap() []byte {
	p := baseSections(2, 2, 1, 2, 1)
	ext := []byte{0xE1, 0xE2, 0xE3, 0xE4, 0xE5, 0xE6, 0xE7, 0xE8}
	p[4] = concat(
		type4Base(0x0180, 0x0280, 0x21, 0x1111, 0x22223333, 0x4444), ext,
		type4Base(0x0380, 0x0480, 0x0007, 0x5555, 0x66667777, 0x8888),
	)
	return buildMap(p)
}

func TestDocumentRoundTripMinimalMap(t *testing.T) {
	data := minimalDocMap()
	doc := roundTrip(t, data)

	// Map() equals Open of the input in result and error.
	fromDoc, docErr := doc.Map()
	fromOpen, openErr := alm.Open(data)
	if docErr != nil || openErr != nil {
		t.Fatalf("Map()/Open errors = %v / %v, want nil / nil", docErr, openErr)
	}
	if !reflect.DeepEqual(fromDoc, fromOpen) {
		t.Errorf("Map() disagrees with Open of the input:\n got %+v\nwant %+v", fromDoc, fromOpen)
	}
}

// ---------------------------------------------------------------------------
// AC-1 (navigation clause), SC-4 — Version, RecordTypeIDs and each
// RecordPayload agree with the stream.
// ---------------------------------------------------------------------------

// streamRecords walks a fixture by the spec's frame arithmetic alone — a
// 20-byte file header, then per record a 20-byte header whose +0x08 word is
// the payload size and +0x0c word the typeId — so every expected navigation
// value is derived from the stream's own bytes at the spec's offsets, never
// captured from the package.
func streamRecords(data []byte) (ids []uint32, payloads [][]byte) {
	cursor := 20
	for i := 0; i < 10; i++ {
		size := int(binary.LittleEndian.Uint32(data[cursor+0x08:]))
		ids = append(ids, binary.LittleEndian.Uint32(data[cursor+0x0c:]))
		payloads = append(payloads, data[cursor+20:cursor+20+size])
		cursor += 20 + size
	}
	return ids, payloads
}

func TestDocumentNavigationAgreesWithStream(t *testing.T) {
	data := minimalDocMap()
	doc := roundTrip(t, data)

	if got, want := doc.Version(), binary.LittleEndian.Uint32(data[0x10:]); got != want {
		t.Errorf("Version() = %d, want %d (the stream's +0x10 word)", got, want)
	}
	wantIDs, wantPayloads := streamRecords(data)
	if got := doc.RecordTypeIDs(); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("RecordTypeIDs() = %v, want %v (file order)", got, wantIDs)
	}
	for i, want := range wantPayloads {
		if got := doc.RecordPayload(i); !bytes.Equal(got, want) {
			t.Errorf("RecordPayload(%d): %s", i, diffReport(got, want))
		}
	}
}

func TestDocumentRecordPayloadOutOfRangePanics(t *testing.T) {
	doc := roundTrip(t, minimalDocMap())
	for _, idx := range []int{-1, 10} {
		idx := idx
		t.Run(fmt.Sprintf("index %d", idx), func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Errorf("RecordPayload(%d): expected an out-of-range panic, got none", idx)
					return
				}
				// An out-of-range subscript panics with a runtime error.
				if _, ok := r.(error); !ok {
					t.Errorf("RecordPayload(%d): recovered %v (%T), want a runtime error", idx, r, r)
				}
			}()
			doc.RecordPayload(idx)
		})
	}
}

// ---------------------------------------------------------------------------
// AC-2, SC-2 — the free header fields survive.
// ---------------------------------------------------------------------------

func TestDocumentRoundTripFreeHeaderFields(t *testing.T) {
	// Acceptance leaves these free: dataSize (any value), formatVersion (any
	// but 1000), the ten per-map-constant words, and the record order.
	const freeDataSize = 0xDEADBEEF
	const freeVersion = 991
	const snanBits = 0x7FBFFFFF // exponent all ones, quiet bit clear: a signalling NaN

	// Descending typeIds — a permutation the shipped maps never use (their
	// order is physicalOrder).
	order := []uint32{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	bits := map[uint32]uint32{}
	for tid := uint32(0); tid <= 9; tid++ {
		bits[tid] = 0xB0000000 | tid // ten distinct words...
	}
	bits[3] = snanBits // ...one of them the signalling NaN

	p := baseSections(2, 2, 0, 0, 0)
	blobs := make([][]byte, 0, len(order))
	for _, tid := range order {
		blobs = append(blobs, concat(recordHdr(7, 20, uint32(len(p[tid])), tid, bits[tid]), p[tid]))
	}
	data := buildFile(fileHeader(almMagic, 20, freeDataSize, 10, freeVersion), blobs)

	doc := roundTrip(t, data)

	view, err := doc.Map()
	if err != nil {
		t.Fatal(err)
	}
	if view.MetaRecordWord != bits[0] || len(view.Records) != len(order) {
		t.Fatalf("decoded record words: %+v", view.Records)
	}
	for i, tid := range order {
		if view.Records[i].Word10 != bits[tid] {
			t.Errorf("decoded record %d word = %#08x, want %#08x", i, view.Records[i].Word10, bits[tid])
		}
	}

	// Every named byte, read back from a fresh write rather than trusted to
	// the fixture's construction.
	out := doc.Write()
	if got := binary.LittleEndian.Uint32(out[0x08:]); got != freeDataSize {
		t.Errorf("written dataSize = %#08x, want %#08x", got, uint32(freeDataSize))
	}
	if got := binary.LittleEndian.Uint32(out[0x10:]); got != freeVersion {
		t.Errorf("written formatVersion = %d, want %d", got, freeVersion)
	}
	cursor := 20
	for i, tid := range order {
		size := int(binary.LittleEndian.Uint32(out[cursor+0x08:]))
		if got := binary.LittleEndian.Uint32(out[cursor+0x0c:]); got != tid {
			t.Errorf("written record %d typeId = %d, want %d (order not preserved)", i, got, tid)
		}
		if got := binary.LittleEndian.Uint32(out[cursor+0x10:]); got != bits[tid] {
			t.Errorf("written record %d per-map word = %#08x, want %#08x", i, got, bits[tid])
		}
		cursor += 20 + size
	}
}

// ---------------------------------------------------------------------------
// AC-3, SC-2 — payload bytes the typed view drops survive.
// ---------------------------------------------------------------------------

// droppedBytesMap builds AC-3's stream: every payload byte the typed view
// drops carries a distinctive non-zero value, so identity cannot hold by the
// bytes happening to be zero.
func droppedBytesMap(type8 []byte) []byte {
	p := baseSections(2, 2, 1, 0, 1)

	meta := buildMeta(2, 2, 1, 0, 1, "", nil)
	copy(meta[0x08:], le32(0x7FC12345))                             // angle bits: a NaN
	copy(meta[0x30:], concat([]byte("Post\x00"), bytesN(20, 0x81))) // name, non-zero past its NUL
	copy(meta[0x78:], concat([]byte{0xC0, 0x00}, bytesN(20, 0x91))) // description, likewise
	p[0] = meta

	r5 := type5Record(5000, "")
	copy(r5[0x00:], bytesN(8, 0xA1))                             // the undecoded type-5 head
	copy(r5[0x0c:], concat([]byte("Grp\x00"), bytesN(16, 0xB1))) // name, non-zero past its NUL
	p[5] = r5

	r6 := type6Record(0x0180, 0x0280, 1, 0, 0, 0)
	copy(r6[0x14:], bytesN(70-0x14, 0xC1)) // the undecoded type-6 tail (+0x14..+0x45)
	p[6] = r6

	p[7] = countBody(2, []byte{0x11, 0x00, 0xFE, 0x7F, 0x80})
	p[8] = type8
	p[9] = countBody(5, []byte{0xCA, 0xFE, 0x00, 0xBA})
	return buildMap(p)
}

func TestDocumentRoundTripDroppedPayloadBytes(t *testing.T) {
	t.Run("non-empty type8", func(t *testing.T) {
		roundTrip(t, droppedBytesMap([]byte{0x0D, 0x00, 0x0C, 0xFF}))
	})
	t.Run("empty type8", func(t *testing.T) {
		roundTrip(t, droppedBytesMap(nil))
	})
}

// ---------------------------------------------------------------------------
// AC-4, SC-3 — the rejection set, both entry points per fixture.
// ---------------------------------------------------------------------------

func TestDocumentRejectsSpecRejectionList(t *testing.T) {
	// One fixture per clause of the spec's rejection list; each verdict —
	// reject — is written here from that list, not asked of Open. Both entry
	// points are then held to it.
	sections := func() map[uint32][]byte { return baseSections(2, 2, 0, 0, 0) }
	withBlob := func(tid uint32, blob []byte) []byte {
		blobs := wrapAll(sections())
		blobs[physIndex(tid)] = blob
		return buildFile(stdFileHeader(), blobs)
	}

	overrun := countBody(0, nil)

	badType0 := sections()
	badType0[0] = make([]byte, 631)

	badGrid := sections()
	badGrid[1] = bytesN(6, 0x10) // 6 != 2*W*H = 8

	badWalk4 := baseSections(2, 2, 0, 1, 0)
	badWalk4[4] = concat(type4Base(0x0100, 0x0200, 1, 0, 0, 0), []byte{0, 0, 0, 0}) // 24 bytes, walk consumes 20

	badSize5 := baseSections(2, 2, 2, 0, 0)
	badSize5[5] = type5Record(0, "g") // one 76-byte record, #type5 = 2

	badSize6 := baseSections(2, 2, 0, 0, 2)
	badSize6[6] = type6Record(0x0180, 0x0280, 0, 0, 0, 0) // one 70-byte record, #type6 = 2

	badType7 := sections()
	badType7[7] = []byte{0x01, 0x00} // too short for the count word

	cases := []struct {
		label string
		data  []byte
	}{
		{"short stream", buildMap(sections())[:10]},
		{"bad magic", buildFile(fileHeader(0xBAD0BAD0, 20, 999, 10, 990), wrapAll(sections()))},
		{"file hdrLen not 20", buildFile(fileHeader(almMagic, 24, 999, 10, 990), wrapAll(sections()))},
		{"recordCount below the minimum of 3", buildFile(fileHeader(almMagic, 20, 999, 2, 990), wrapAll(sections()))},
		{"formatVersion 1000", buildFile(fileHeader(almMagic, 20, 999, 10, 1000), wrapAll(sections()))},
		{"formatVersion above 1001", buildFile(fileHeader(almMagic, 20, 999, 10, 1002), wrapAll(sections()))},
		{"record tag not 7", withBlob(3, recCustom(6, 20, 3, sections()[3]))},
		{"record hdrLen not 20", withBlob(3, recCustom(7, 24, 3, sections()[3]))},
		{"payload overruns EOF", withBlob(7, recBadSize(7, uint32(len(overrun))+256, overrun))},
		{"no type-1 record", buildPartial(sections(), 0, 2, 3)},
		{"no type-2 record", buildPartial(sections(), 0, 1, 3)},
		{"no type-0 record", buildPartial(sections(), 1, 2, 3)},
		{"631-byte type-0", buildMap(badType0)},
		{"grid-size mismatch", buildMap(badGrid)},
		{"type-4-walk mismatch", buildMap(badWalk4)},
		{"type-5-size mismatch", buildMap(badSize5)},
		{"type-6-size mismatch", buildMap(badSize6)},
		{"type-7 shorter than its count word", buildMap(badType7)},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			openReject(t, tc.data) // the reader's half of the decision
			docReject(t, tc.data)  // the document's half: wrapped error, nil document
		})
	}
}

// ---------------------------------------------------------------------------
// The document over a stream that is not ten records.
//
// This is a REGRESSION TEST before it is a feature test.
// ---------------------------------------------------------------------------

func TestDocumentOverAShorterRoster(t *testing.T) {
	four := buildPartial(baseSections(2, 2, 0, 0, 0), 0, 1, 2, 3)

	doc, err := alm.OpenDocument(four)
	if err != nil {
		t.Fatalf("OpenDocument rejected a four-record map: %v", err)
	}
	if got := doc.RecordCount(); got != 4 {
		t.Errorf("RecordCount() = %d, want 4", got)
	}
	if got := doc.RecordTypeIDs(); !reflect.DeepEqual(got, []uint32{0, 1, 2, 3}) {
		t.Errorf("RecordTypeIDs() = %v, want [0 1 2 3]", got)
	}
	if got := doc.Write(); !bytes.Equal(got, four) {
		t.Error("Write did not reproduce the four-record input byte-for-byte")
	}
	for i := 0; i < doc.RecordCount(); i++ {
		if !bytes.Equal(doc.RecordPayload(i), baseSections(2, 2, 0, 0, 0)[uint32(i)]) {
			t.Errorf("RecordPayload(%d) is not the record's own payload", i)
		}
	}

	// Preservation is what makes the manufactured plane safe: the decoded view
	// gained a type-3 the file does not have, and the byte view must not.
	m, err := doc.Map()
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if len(m.Overlay) != 2*2 {
		t.Errorf("len(Overlay) = %d, want the full W*H plane", len(m.Overlay))
	}

	three := buildPartial(baseSections(2, 2, 0, 0, 0), 0, 1, 2)
	doc3, err := alm.OpenDocument(three)
	if err != nil {
		t.Fatalf("OpenDocument rejected a three-record map: %v", err)
	}
	if got := doc3.RecordCount(); got != 3 {
		t.Errorf("RecordCount() = %d, want 3", got)
	}
	if got := doc3.Write(); !bytes.Equal(got, three) {
		t.Error("Write invented a record the three-record input did not carry")
	}
	m3, err := doc3.Map()
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if m3.Present(3) {
		t.Error("Present(3) is true on a document whose bytes hold no type-3 record")
	}
}

// TestDocumentPreservesBytesNoRecordCovers is the other half of preservation: a
// trailer and an undecodable record are both accepted now, and both must come
// back out of Write untouched.
func TestDocumentPreservesBytesNoRecordCovers(t *testing.T) {
	blobs := wrapAll(baseSections(2, 2, 0, 0, 0))
	blobs = append(blobs, rec(10, []byte{0xDE, 0xAD}))
	data := buildFile(fileHeader(almMagic, 20, 999, 11, 990), blobs)
	data = append(data, 0x5A, 0x5B) // and a trailer past the last record

	doc, err := alm.OpenDocument(data)
	if err != nil {
		t.Fatalf("OpenDocument: %v", err)
	}
	if got := doc.RecordCount(); got != 11 {
		t.Errorf("RecordCount() = %d, want 11 — the unhandled record is still a record", got)
	}
	if got := doc.Write(); !bytes.Equal(got, data) {
		t.Error("Write dropped the trailer or the undecodable record")
	}
}

// ---------------------------------------------------------------------------
// AC-5, SC-5 — the input buffer and the write buffers are independent.
// Every verdict is asserted against a clone taken before the mutation.
// ---------------------------------------------------------------------------

func TestDocumentOwnershipInputAndWriteBuffers(t *testing.T) {
	data := minimalDocMap()
	orig := append([]byte(nil), data...) // the pre-mutation clone every verdict uses

	doc, err := alm.OpenDocument(data)
	if err != nil {
		t.Fatalf("OpenDocument: unexpected error: %v", err)
	}

	// Zero the caller's input buffer: the document must not alias it.
	for i := range data {
		data[i] = 0
	}
	first := doc.Write()
	if !bytes.Equal(first, orig) {
		t.Errorf("Write after zeroing the input buffer: %s", diffReport(first, orig))
	}

	// Mutate the returned write buffer: neither the document nor a second
	// write may see it.
	for i := range first {
		first[i] ^= 0xFF
	}
	second := doc.Write()
	if !bytes.Equal(second, orig) {
		t.Errorf("second Write after mutating the first buffer: %s", diffReport(second, orig))
	}
}

// ---------------------------------------------------------------------------
// AC-6, SC-5 — navigation results are fresh copies; the accessors mutate
// nothing. Every verdict is asserted against a clone taken before the
// mutation.
// ---------------------------------------------------------------------------

func TestDocumentNavigationResultsAreFreshCopies(t *testing.T) {
	data := minimalDocMap()
	orig := append([]byte(nil), data...) // the pre-mutation clone every verdict uses

	doc, err := alm.OpenDocument(data)
	if err != nil {
		t.Fatalf("OpenDocument: unexpected error: %v", err)
	}

	// Overwrite every navigation result the document hands out.
	ids := doc.RecordTypeIDs()
	for i := range ids {
		ids[i] = 0xEEEEEEEE
	}
	for i := 0; i < 10; i++ {
		p := doc.RecordPayload(i)
		for j := range p {
			p[j] = 0xEE
		}
	}

	// The document writes as before...
	if out := doc.Write(); !bytes.Equal(out, orig) {
		t.Errorf("Write after overwriting navigation results: %s", diffReport(out, orig))
	}
	// ...and navigates as before, expectations derived from the clone at the
	// spec's offsets.
	wantIDs, wantPayloads := streamRecords(orig)
	if got := doc.RecordTypeIDs(); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("RecordTypeIDs after overwrite = %v, want %v", got, wantIDs)
	}
	for i, want := range wantPayloads {
		if got := doc.RecordPayload(i); !bytes.Equal(got, want) {
			t.Errorf("RecordPayload(%d) after overwrite: %s", i, diffReport(got, want))
		}
	}
}
