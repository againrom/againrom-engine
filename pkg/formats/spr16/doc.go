// Package spr16 implements read-only decoders for the game's two 16-bit
// sprite formats: .16a — palette-indexed pixels with a per-pixel 4-bit level
// (cursor, interface, projectile and effect art) — and .16 — 4-bit glyph
// atlases (bitmap fonts). Both carry sequential
// [u32 width][u32 height][u32 dataSize][RLE block] frame records and a 4-byte
// trailer. The .16a decoder uses its low31 count and an explicit caller
// palette declaration (SPR16A-TRLR-012, SPR16A-RDR-017). The .16 decoder uses
// the raw nonnegative count and refuses negative values; its frames start at
// zero without a palette (SPR16A-070 through SPR16A-072, DIV-1250).
// Decoded pixels keep their raw fields — the .16a
// {index, level} pair and the .16 value; FrameA.RGBA is the one resolution of
// a .16a pair to a premultiplied colour (SPR16A-031). Malformed input is
// rejected with an error and no result, never a panic. See
// knowledge/formats/spr16a/format.md for the current byte-level contract.
//
// A font is TWO nodes, not one: beside the glyph atlas sits <base>.dat, one u32
// per atlas record giving that glyph's pen advance, which the cell width never
// is. Advances decodes it; joining the pair is the caller's, since neither file
// names the other (docs/0050-text/spec.md).
//
// Tier: pkg/formats/* is the lowest layer. This package imports the Go
// standard library and pkg/formats/pal, the one reader of the palette layout —
// neither grammar carries text, so the formats tier's golang.org/x/text grant
// is denied to it in internal/archtest's noExternalFormats map. The boundary
// is enforced by internal/archtest and documented in docs/ARCHITECTURE.md.
package spr16
