// Package spr256 implements a read-only decoder for the ROM1 .256 paletted
// sprite format: an optional leading 1024-byte palette, a sequence of frame
// records, and a 4-byte trailer. Decode takes the raw bytes of a .256 stream
// (for example, the payload of a .res archive entry) and returns an ordered list
// of frames — each a width x height grid whose pixels are either transparent or
// an opaque 0-255 palette index — plus the 256-color RGB palette when present.
// The trailer's low 31 bits are the frame count and its top bit is the
// has-palette flag; decoded pixels keep their palette indices (no RGB is
// resolved) so a consumer may apply an alternative palette. Malformed input is
// rejected with an error and no result, never a panic. See
// docs/0002-sprites-256/spec.md for the byte-level format.
//
// Tier: pkg/formats/* is the lowest layer. This package imports only the Go
// standard library — .256 carries no strings, so unlike the archive/registry
// readers it needs no CP866 decoder. It must not import any other againrom
// package. The boundary is enforced by internal/archtest and documented in
// docs/ARCHITECTURE.md.
package spr256
