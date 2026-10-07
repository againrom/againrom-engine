// Package reg implements the .reg registry format.
//
// Tier: pkg/formats/* is the lowest layer. This package may import only the Go
// standard library and must not import any other againrom package. It takes no
// text-encoding dependency either: the format stores byte strings and defines no
// character encoding, so names and string values come back as the bytes the
// stream holds and this package converts none of them. See docs/ARCHITECTURE.md.
package reg
