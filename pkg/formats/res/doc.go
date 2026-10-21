// Package res reads and writes the ROM1 .res archive container: a 24-byte
// header, a data blob, and a trailing 32-byte-node registry forming a tree.
// Open (or OpenBytes) indexes every file by a normalized, case-insensitive,
// separator-agnostic path; ReadFile returns its bytes and Tree the nodes as
// stored. Malformed archives are rejected with an error, never a panic. Write
// builds an archive the reader reads, for a packed mod. See
// docs/0001-res-archive/spec.md for the byte-level format.
//
// Tier: pkg/formats/* is the lowest layer. It may import only the Go standard
// library (plus golang.org/x/text for CP866 string decoding) and must not import
// any other againrom package. The boundary is enforced by internal/archtest and
// documented in docs/ARCHITECTURE.md.
package res
