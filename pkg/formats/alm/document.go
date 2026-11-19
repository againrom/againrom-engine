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
	data   []byte       // the accepted stream, one private flat copy
	spans  []recordSpan // one per record the file header counts, file order
	format Format
}

// OpenDocument retains a map accepted by the shared reader in the selected
// format, defaulting to the first game's. Its physical spans use the reader's
// actual payload extents; declared lengths can differ in the second game.
func OpenDocument(data []byte, formats ...Format) (*Document, error) {
	buf := cloneBytes(data)
	format := selectedFormat(formats)
	m, err := open(buf, format)
	if err != nil {
		return nil, fmt.Errorf("alm: document rejected: %w", err)
	}
	d := &Document{data: buf, format: format, spans: make([]recordSpan, 0, len(m.Records))}
	cursor := FileHeaderSize
	for _, rec := range m.Records {
		d.spans = append(d.spans, recordSpan{headerOff: cursor, payloadSize: int(rec.PayloadSize)})
		cursor += RecordHeaderSize + int(rec.PayloadSize)
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

// Map reads the retained bytes in the document's accepted format.
func (d *Document) Map() (*Map, error) {
	return open(d.data, d.format)
}

// Version returns the file header's formatVersion, read from the u32 at
// +0x10 of the retained copy on every call. The document stores no decoded
// state, so the result cannot disagree with the written bytes.
func (d *Document) Version() uint32 {
	return binary.LittleEndian.Uint32(d.data[FileFormatVersion:])
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
		ids[i] = binary.LittleEndian.Uint32(d.data[sp.headerOff+RecordTypeID:])
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
	start := sp.headerOff + RecordHeaderSize
	return cloneBytes(d.data[start : start+sp.payloadSize])
}
