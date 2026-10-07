package alm

import (
	"encoding/binary"
	"fmt"
)

// recordSpan locates one record inside a document's retained copy: the
// absolute offset of its 20-byte header and its payload byte length, in file
// order.
type recordSpan struct {
	headerOff   int
	payloadSize int
}

// Document is a raw-backed model of one accepted .alm stream. It retains a
// single private flat copy of the bytes, interprets none of them, and writes
// itself back; the header fields the interpreted reader validates or stores
// without using — dataSize, the record tags and hdrLens, the per-map
// constants — ride the copy verbatim like every other byte, whatever they
// turn out to mean. No byte of the copy is ever routed through a float32, so
// a signalling-NaN bit pattern cannot be quieted by a conversion.
type Document struct {
	data  []byte       // the accepted stream, one private flat copy
	spans []recordSpan // one per record the file header counts, file order
}

// OpenDocument reads a ROM1 .alm map as a raw document. It accepts precisely
// the byte streams Open accepts — the decision is taken by running Open on
// the document's own copy of the input, so the two entry points cannot
// drift — and rejects the rest with a wrapped error and a nil document, never
// a partial result or a panic. dataSize and the per-map-constant words play
// no part in the decision. The input is copied in: mutating the caller's
// buffer afterwards does not change what Write emits.
func OpenDocument(data []byte) (*Document, error) {
	buf := cloneBytes(data)
	if _, err := Open(buf); err != nil {
		return nil, fmt.Errorf("alm: document rejected: %w", err)
	}

	// Open has proven the frame: recordCount records, every header and payload
	// in bounds. The walk is bounds-checked all the same, and that is not
	// belt-and-braces — the count is now a FILE FIELD rather than the constant
	// ten, so an arithmetic slip here would be a slice-bounds panic on valid
	// input rather than an impossible state. Bytes after the last record, which
	// acceptance no longer forbids, are covered by no span and ride the copy.
	n := binary.LittleEndian.Uint32(buf[fhRecordCount:])
	d := &Document{data: buf, spans: make([]recordSpan, 0, minRecordCount)}
	cursor := 0 + fileHeaderSize
	for i := uint32(0); i < n; i++ {
		if cursor+recordHeaderSize > len(buf) {
			break
		}
		size := uint64(binary.LittleEndian.Uint32(buf[cursor+recPayloadSize:]))
		if size > uint64(len(buf)-cursor-recordHeaderSize) {
			break
		}
		d.spans = append(d.spans, recordSpan{headerOff: cursor, payloadSize: int(size)})
		cursor += recordHeaderSize + int(size)
	}
	return d, nil
}

// Write serializes the document into a fresh buffer. The writer emits the
// retained copy and reconstructs nothing from interpreted values, so the
// unedited document writes back exactly the bytes it was opened from.
// Mutating a returned buffer affects neither the document nor any other
// write.
func (d *Document) Write() []byte {
	return cloneBytes(d.data)
}

// Map returns the interpreted view of the document: Open of the document's
// bytes, run per call, equal to Open of the original input in result and
// error. For a document OpenDocument produced, that call already succeeded
// at open time.
func (d *Document) Map() (*Map, error) {
	return Open(d.data)
}

// Version returns the file header's formatVersion, read from the u32 at
// +0x10 of the retained copy on every call. The document stores no decoded
// state, so the result cannot disagree with the written bytes.
func (d *Document) Version() uint32 {
	return binary.LittleEndian.Uint32(d.data[fhFormatVersion:])
}

// RecordTypeIDs returns the records' typeIds in file order, read from each
// indexed header's typeId word on every call into a fresh slice. Its length is
// the stream's own record count, which is at least three and need not be ten;
// an id at or above 10 appears here like any other, since the document types
// nothing and interprets nothing. Mutating a returned slice changes nothing the
// document holds.
func (d *Document) RecordTypeIDs() []uint32 {
	ids := make([]uint32, len(d.spans))
	for i, sp := range d.spans {
		ids[i] = binary.LittleEndian.Uint32(d.data[sp.headerOff+recTypeID:])
	}
	return ids
}

// RecordCount returns how many records the document holds — the length
// RecordTypeIDs returns and the exclusive upper bound of RecordPayload's index.
func (d *Document) RecordCount() int { return len(d.spans) }

// RecordPayload returns the byte-exact payload of record i in file order,
// cloned from the retained copy on every call. Mutating a returned slice
// changes nothing the document holds. An i outside 0..RecordCount()-1 reaches
// the span subscript unguarded and panics exactly as an out-of-range slice
// index does: misuse, not data.
func (d *Document) RecordPayload(i int) []byte {
	sp := d.spans[i]
	start := sp.headerOff + recordHeaderSize
	return cloneBytes(d.data[start : start+sp.payloadSize])
}
