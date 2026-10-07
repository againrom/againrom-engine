// Package alm implements a pure reader for the ROM1 .alm map container (magic
// "M7R\x00"): a 20-byte file header, then the number of length-prefixed typed
// records that header counts — each a 20-byte record header
// [tag=7][hdrLen=20][payloadSize][typeId][f32 perMapConst] followed by
// payloadSize bytes of pure payload. The record types are the type-0 metadata,
// three W x H terrain grids (tiles, altitudes, object overlay), and the
// object/roster/unit/trigger content records. Open takes the raw map bytes and
// returns a decoded *Map; malformed input yields a non-nil error and a nil
// *Map, never a panic. Each record is typed from its own header, so an empty
// (payloadSize 0) record is typed directly.
//
// TEN RECORDS IS THE SHIPPED WRITER'S HABIT, NOT THE CONTAINER'S RULE. The
// engine's loader gates the count only at >= 3 and requires just the type-1 and
// type-2 records, because those two are the planes its world builder
// dereferences per cell with no null test (ALM-REQ-055). A missing type-3 it
// manufactures as a W x H zero plane; types 4 and 6..9 it skips, their loop
// bounds being type-0 fields that read 0 once the records are gone; a typeId at
// or above 10 selects no case and is stepped over with no error
// (ALM-REQ-056). This reader follows all of that. It stays deliberately
// stricter in three places, each stated in docs/0003-alm-container/spec.md:
// type-0 is required, every payload-size constraint remains a rejection, and
// formatVersion 1000 is refused as an unimplemented dialect.
//
// Absence and emptiness are therefore different, and Map.Present is what tells
// them apart. It matters most for Overlay, which is a full W x H plane whether
// it was read or manufactured: a caller writing a map back must never emit a
// record whose Present is false, because those bytes were never in the input.
//
// Strings: the type-0 (and type5) name fields are ASCII; the type-0 description
// is code-page text decoded as Windows-1251 (CP1251), which reproduces ASCII
// unchanged. Both convert to UTF-8. EncodeName and EncodeDescription are the
// write halves of those same two rules, each building a whole fixed-width
// field image — the text's bytes in that field's codec, then NUL fill — or
// rejecting the string, so that a value this package decoded from a field is
// written back as that field's own bytes or not written at all.
//
// Round trip: beside the interpreted reader sits a raw-backed Document.
// OpenDocument accepts exactly the streams Open accepts — it decides by
// running Open on its own private copy — and rejects the rest with a wrapped
// error and a nil document. The document interprets nothing: Write emits the
// retained copy into a fresh buffer, so an unedited document writes back its
// input byte-for-byte, preserving everything acceptance leaves free (dataSize,
// formatVersion, the record order, the per-map-constant bits, post-NUL string
// bytes, undecoded record bytes, float bit patterns); Map re-opens the
// retained bytes for the interpreted view. See
// docs/0023-alm-roundtrip-writer/spec.md.
//
// Tier: pkg/formats/* is the lowest layer. It may import only the Go standard
// library plus golang.org/x/text (charmap.Windows1251 for the CP1251
// description) and must not import any other againrom package. See
// docs/ARCHITECTURE.md.
package alm
