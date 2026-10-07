// Package databin walks the placeable-definition table and hands its eleven
// collections back verbatim.
//
// The file is a serialized image of eleven by-value tables, written as eight
// GROUPS in one fixed order. A group is a string array of the column titles its
// collections share, then each of its collections as a u32 entry count followed
// by that many entries. The first two groups write every entry their count
// names; the other six allocate an entry 0 and never write it, so their entries
// run 1 to count-1 and every subscript into them is the file's own 1-based
// index. The walk tiles the payload exactly: a byte left over after the last
// group is an error, not slack.
//
// The file counts strings TWO WAYS and the difference is load-bearing. A
// group's title array is counted: a u16 and then that many strings. An entry's
// own trailing strings are NOT — the field is sized by the class, so its
// elements are written back to back with no count word at all, two for a unit,
// ten for a human, one for a magic item or a spell. Reading the second as the
// first mis-reads from the fifth group on and leaves the file untiled, which is
// what a whole-file residue test is for.
//
// Parse INTERPRETS NOTHING. A name is the file's own bytes with no character
// encoding applied; a parameter is an int32, so the -1 an empty cell is stored
// as arrives as -1 rather than as a large unsigned number; every extra field an
// entry carries comes back as raw bytes or as strings, including the ones this
// project has no meaning for. No column is named here and nothing in this
// package knows what a unit is: what a parameter means belongs to the tier that
// consumes it, and a collection whose consumer has not been written is framed
// and handed over all the same.
//
// It opens nothing. A caller resolves the archive node and hands over its bytes,
// exactly as the registry parser's caller does, which is what keeps a format
// package testable from a byte slice alone.
//
// Tier: pkg/formats/databin is held to the STANDARD LIBRARY. It converts no
// text, so it takes no text-encoding dependency and the import check denies it
// the formats tier's grant. See docs/ARCHITECTURE.md.
package databin
