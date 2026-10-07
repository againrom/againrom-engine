package alm_test

import (
	"bytes"
	"testing"

	"againrom/pkg/formats/alm"
)

// permutedOrderDocMap builds the accepted permuted-order seed: the ten records
// in descending typeId order — a permutation the shipped maps never use.
func permutedOrderDocMap() []byte {
	p := baseSections(2, 2, 0, 0, 0)
	blobs := make([][]byte, 0, 10)
	for _, tid := range []uint32{9, 8, 7, 6, 5, 4, 3, 2, 1, 0} {
		blobs = append(blobs, rec(tid, p[tid]))
	}
	return buildFile(stdFileHeader(), blobs)
}

// FuzzDocumentRoundTrip holds OpenDocument to agreement with Open on every
// input — the agreement clause IS the acceptance oracle; no second
// acceptance walk is implemented here — plus, on accept, round-trip
// identity and copy-independence. The per-class two-way verdicts, written
// from the spec's rejection list, live in document_test.go; the seeds below
// only carry each class into the corpus.
func FuzzDocumentRoundTrip(f *testing.F) {
	// The accept side: a minimal accepted map in the shipped physical order,
	// and the permuted-order variant.
	f.Add(minimalDocMap())
	f.Add(permutedOrderDocMap())

	// The frame-variant side: streams the widened acceptance now takes, whose
	// value as seeds is that they are the shapes least like the shipped writer's.
	// The target asserts accept-or-clean-error either way, so a seed moving from
	// one side to the other costs nothing but a misleading label.
	sections := func() map[uint32][]byte { return baseSections(2, 2, 0, 0, 0) }
	withBlob := func(tid uint32, blob []byte) []byte {
		blobs := wrapAll(sections())
		blobs[physIndex(tid)] = blob
		return buildFile(stdFileHeader(), blobs)
	}

	f.Add(buildPartial(sections(), 0, 1, 2))                                       // three records, type-3 manufactured
	f.Add(buildPartial(sections(), 0, 1, 2, 3))                                    // four records, the shipped RU shape
	f.Add(buildFile(fileHeader(almMagic, 20, 999, 9, 990), wrapAll(sections())))   // nine counted, the tenth trailing
	f.Add(buildFile(fileHeader(almMagic, 20, 999, 10, 1001), wrapAll(sections()))) // formatVersion at the gate

	// The reject side: one seed per rejection class.
	f.Add(buildMap(sections())[:10])                                                // short stream
	f.Add(buildFile(fileHeader(0xBAD0BAD0, 20, 999, 10, 990), wrapAll(sections()))) // bad magic
	f.Add(buildFile(fileHeader(almMagic, 24, 999, 10, 990), wrapAll(sections())))   // file hdrLen not 20
	f.Add(buildFile(fileHeader(almMagic, 20, 999, 2, 990), wrapAll(sections())))    // recordCount below 3
	f.Add(buildFile(fileHeader(almMagic, 20, 999, 10, 1000), wrapAll(sections())))  // formatVersion 1000
	f.Add(buildFile(fileHeader(almMagic, 20, 999, 10, 1002), wrapAll(sections())))  // formatVersion above 1001
	f.Add(buildPartial(sections(), 0, 2, 3))                                        // no type-1 record
	f.Add(buildPartial(sections(), 1, 2, 3))                                        // no type-0 record
	f.Add(withBlob(3, recCustom(6, 20, 3, sections()[3])))                          // record tag not 7
	f.Add(withBlob(3, recCustom(7, 24, 3, sections()[3])))                          // record hdrLen not 20

	overrun := countBody(0, nil)
	f.Add(withBlob(7, recBadSize(7, uint32(len(overrun))+256, overrun))) // payload overruns EOF

	f.Add(withBlob(9, rec(1, sections()[1])))       // a repeated typeId (last-wins)
	f.Add(withBlob(9, rec(10, countBody(0, nil))))  // a typeId no case handles
	f.Add(append(buildMap(sections()), 0x5A, 0x5B)) // trailing bytes

	badType0 := sections()
	badType0[0] = make([]byte, 631)
	f.Add(buildMap(badType0)) // 631-byte type-0

	badGrid := sections()
	badGrid[1] = bytesN(6, 0x10) // 6 != 2*W*H = 8
	f.Add(buildMap(badGrid))     // grid-size mismatch

	badWalk4 := baseSections(2, 2, 0, 1, 0)
	badWalk4[4] = concat(type4Base(0x0100, 0x0200, 1, 0, 0, 0), []byte{0, 0, 0, 0}) // 24 bytes, walk consumes 20
	f.Add(buildMap(badWalk4))                                                       // type-4-walk mismatch

	badSize5 := baseSections(2, 2, 2, 0, 0)
	badSize5[5] = type5Record(0, "g") // one 76-byte record, #type5 = 2
	f.Add(buildMap(badSize5))         // type-5-size mismatch

	badSize6 := baseSections(2, 2, 0, 0, 2)
	badSize6[6] = type6Record(0x0180, 0x0280, 0, 0, 0, 0) // one 70-byte record, #type6 = 2
	f.Add(buildMap(badSize6))                             // type-6-size mismatch

	badType7 := sections()
	badType7[7] = []byte{0x01, 0x00} // too short for the count word
	f.Add(buildMap(badType7))        // type-7 shorter than its count word

	f.Fuzz(func(t *testing.T, data []byte) {
		// The oracle: Open's decision on the very same bytes.
		_, openErr := alm.Open(data)

		// The engine reuses its input slice in-place under -fuzz, so zeroing
		// `data` would poison mutation and corpus recording. OpenDocument gets
		// a working clone; only that clone is ever written to below.
		work := append([]byte(nil), data...)
		doc, docErr := alm.OpenDocument(work)

		if openErr != nil {
			// The reject half: agreement, plus the wrapper's atomicity — never a
			// document beside an error.
			if docErr == nil {
				t.Fatalf("Open rejected (%v) but OpenDocument accepted", openErr)
			}
			if doc != nil {
				t.Fatal("OpenDocument returned a document beside an error")
			}
			return
		}
		if docErr != nil {
			t.Fatalf("Open accepted but OpenDocument rejected: %v", docErr)
		}
		if doc == nil {
			t.Fatal("OpenDocument returned a nil document with a nil error")
		}

		// Zero the working clone post-open: the document must not alias it. Write
		// must still equal the engine's untouched input — round-trip identity
		// and copy-independence in one assertion.
		for i := range work {
			work[i] = 0
		}
		first := doc.Write()
		if !bytes.Equal(first, data) {
			t.Fatalf("Write differs from the accepted input: %s", diffReport(first, data))
		}

		// Mutate the first write buffer: a second write must not see it.
		for i := range first {
			first[i] ^= 0xFF
		}
		if second := doc.Write(); !bytes.Equal(second, data) {
			t.Fatalf("second Write after mutating the first buffer: %s", diffReport(second, data))
		}
	})
}
