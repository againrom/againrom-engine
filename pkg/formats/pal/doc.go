// Package pal reads the 256-entry colour table a per-class palette file
// carries.
//
// A `.pal` the unit loader opens is NOT a container of its own: it is a Windows
// BMP whose colour table the engine lifts and whose pixel data it never opens.
// The engine seeks 0x36 and reads exactly 0x400 bytes — 256 entries of
// [B, G, R, reserved], the same layout as a .256 sheet's own leading 1024 bytes
// — and reads no other byte of the file. This package does exactly that and no
// more.
//
// IT PARSES NO BMP HEADER, and that is a decision rather than an omission. No
// width, height, bit depth, palette count or pixel-data offset is read, because
// the engine reads none of them: a decoder that honoured a header would
// disagree with the engine on precisely the file whose header disagrees with
// its content, and the whole point of this package is to be the same 1024 bytes
// the engine takes.
//
// It DOES check the two-byte magic, which the engine does not. That check is
// ours and it is a positive exclusion: one further file in the same tree is a
// palette of another shape entirely — sixteen consecutive 1024-byte tables read
// whole with no seek — and its first two bytes are 00 00 rather than BM. Handed
// that file, a reader written for the common case returns 1024 bytes of
// somebody's pixels and says nothing. Refusing on the magic makes the mistake
// loud instead. Every address the per-class name construction builds begins BM
// on both lawful roots, so the check excludes the other shape and nothing else.
//
// It opens nothing. A caller resolves the archive node and hands over its
// bytes, exactly as the registry parser's caller does, which is what keeps a
// format package testable from a byte slice alone.
//
// Tier: pkg/formats/pal is held to the STANDARD LIBRARY. It carries no strings
// at all, so it takes no text-encoding dependency and the import check denies
// it the formats tier's grant. See docs/ARCHITECTURE.md.
package pal
