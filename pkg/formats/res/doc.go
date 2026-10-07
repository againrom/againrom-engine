// Package res implements a read-only reader for the ROM1 .res archive container
// format: a 24-byte little-endian header, a concatenated data blob, and a
// trailing 32-byte-node registry forming a directory tree. Open (or OpenBytes)
// indexes every file entry by a normalized, case-insensitive, separator-agnostic
// path; ReadFile returns an entry's bytes. Malformed archives are rejected with
// an error, never a panic. See docs/0001-res-archive/spec.md for the byte-level
// format.
//
// Tier: pkg/formats/* is the lowest layer. It may import only the Go standard
// library (plus golang.org/x/text for CP866 string decoding) and must not import
// any other againrom package. The boundary is enforced by internal/archtest and
// documented in docs/ARCHITECTURE.md.
package res
