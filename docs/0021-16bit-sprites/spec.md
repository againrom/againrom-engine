# Spec — `.16a` / `.16` 16-bit sprite decoder

## Problem / current behaviour

The engine decodes the 8-bit paletted `.256` sprite format and neither of the game's two 16-bit
sprite formats. `.16a` carries palette-indexed pixels with a per-pixel 4-bit level — cursor,
interface, projectile and effect art; `.16` carries 4-bit glyph atlases — bitmap fonts. Both share
the `.256` container shape and differ in their pixel grammars; no decoder for either exists in
this tree.

## Format definition (external contract)

All integers little-endian. A **stream** is the complete byte sequence of one sprite.

### Shared container

```
[ 1024 B palette — .16a, when declared ][ frame records ][ u32 trailer ]
```

| Region | Size | Content |
|---|---|---|
| Palette | 1024 B | 256 × 4 B `[B, G, R, x]`, returned as 256 RGB colours, the 4th byte not decoded; only present when the consumer declares it |
| Frames | varies | exactly `frameCount` records, strictly sequential from the palette's end (offset 0 when none) |
| Trailer | 4 B | the **last** 4 bytes of the stream: `frameCount = trailer & 0x7FFFFFFF`; bit 31 is masked off and never consulted |

Palette presence is the consumer's declaration, never inferred from stream content. Bytes between
the end of record `frameCount` and the trailer are ignored. The minimum well-formed stream is
4 bytes without a palette, 1028 with one; a `frameCount` of 0 decodes to an empty frame list.

Frame record (both formats):

| Field | Size | Content |
|---|---|---|
| `width` | u32 | frame width in pixels |
| `height` | u32 | frame height in pixels |
| `dataSize` | u32 | byte length of the RLE block |
| RLE block | `dataSize` B | pixel program, per format below |

`width = 0` or `height = 0` is a valid header for an empty grid. Header fields and the frame count
are sanity-capped (Constraints): a value over its cap is an **error**, never a frame silently
skipped.

**Cursor model (both formats).** An RLE block decodes a `width × height` grid. A cursor walks it
in linear order — position `c` is pixel `(c mod width, c div width)`, row 0 the top row — and
every pixel starts **transparent**; only a literal paints. Any op may leave the cursor at exactly
`width × height`; one that would move or paint beyond it is malformed. The program is consumed to
the end of its own `dataSize` block: an unfinished grid stays transparent, bytes after the grid
completes are permitted only as no-effect ops (count 0), and an op and its operands must lie
wholly within the block — reading past it is malformed; the next frame's bytes are never read.

### `.16a` RLE — u16 control words, u16 pixels

Each control is a `u16` `cw`: `op = cw >> 14`, `n = cw & 0x3FFF` (a 14-bit count).

| `op` | Effect |
|---|---|
| `0b00` | literal: the next `n` u16 words paint one pixel each |
| `0b01` | blank rows: cursor advances `n × width` |
| `0b10` | skip: cursor advances `n` |
| `0b11` | decoded identically to `0b01` (blank rows) |

A literal pixel word `ss` decodes as:

| `ss` bits | Field |
|---|---|
| 1–8 | palette index: `index = (ss >> 1) & 0xFF` |
| 9–12 | level: `level = (ss >> 9) & 0x0F` |
| 0, 13–15 | not decoded (masked off) |

A painted `.16a` pixel is `{index 0–255, level 0–15}`, both preserved raw and assigned no colour
or transparency meaning; a painted level 0 is distinct from a transparent pixel, and resolving the
pair against the palette is the consumer's concern.

### `.16` RLE — byte controls, nibble pixels

Each control is one byte `b`: `op = b >> 6`, `n = b & 0x3F`. Ops `0b01` and `0b10` act exactly as
in `.16a`; `0b11` is decoded identically to `0b10` (skip) — the two grammars alias their unused
quadrant in opposite directions. `0b00` is a literal of the next `n` **bytes**, each carrying up
to two 4-bit pixels, **low nibble first**:

1. low nibble: paint value `b & 0x0F`, cursor +1;
2. high nibble: paint value `b >> 4`, cursor +1 — **except** on the run's final byte when that
   nibble is 0, where it is a pad and no pixel is emitted.

A run of `n` bytes paints `2n` pixels, or `2n − 1` when its final byte's high nibble is 0. A
**mid-run** zero nibble paints a value-0 pixel and advances the cursor; a final byte with a
non-zero high nibble paints both. A painted `.16` pixel is a raw 4-bit value 0–15, distinct from
transparent; its interpretation is outside this contract.

## Functional requirements

- **FR-1** — a well-formed `.16a` stream MUST decode to an ordered list of `frameCount` frames —
  each a `width × height` grid of transparent or painted `{index, level}` pixels — plus the
  256-colour palette when declared.
- **FR-2** — a well-formed `.16` stream MUST decode to an ordered list of `frameCount` frames of
  transparent or painted 4-bit values, with no palette.
- **FR-3** — palette presence for `.16a` MUST be taken from the consumer's declaration and MUST
  NOT be inferred from stream content; `.16` never has a palette.
- **FR-4** — the frame count MUST be `trailer & 0x7FFFFFFF`; decoded output MUST be independent of
  trailer bit 31; bytes between the last counted record and the trailer MUST be ignored; a count
  of 0 MUST decode to an empty list.
- **FR-5** — decoded pixels MUST preserve their raw fields — the `.16a` `{index, level}` pair and
  the `.16` value — with no expansion, no colour resolution and no reinterpretation; the palette
  is returned separately for the consumer to apply it, another palette, or its own mapping.
- **FR-6** — malformed input MUST be rejected with an error and no decoded result (palette
  included), never a panic or partial output: a stream shorter than its fixed regions; a header
  field or frame count over its sanity cap, or a stream over the stream cap (Constraints); a frame
  header or RLE block that does not fit before the trailer; a literal run whose operands overrun
  their block; an op that would move or paint beyond `width × height`. Error classes need not be
  distinguishable.
- **FR-7** — the decoder consumes a byte stream alone — no archive, filesystem, rendering or
  game-install knowledge — and everything here but FR-8's visual check MUST be verifiable over
  synthetic streams with no game install.
- **FR-8** — a developer-run dump MUST render decoded frames of real files to image files for
  visual verification against a lawful install; any choice beyond this contract (how a level or
  glyph value is shown, palette application) is presentation, disclosed in its output or usage
  text, never format data.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic multi-frame `.16a` stream, palette declared: literal, skip and blank-row ops, one count above 255 (bits 8–13 set) | decoded | indices, levels, transparency, palette and frame count match the expected grids exactly |
| AC-2 | unit | two streams identical except trailer bit 31 (each format) | both decoded | outputs identical |
| AC-3 | unit | streams with `width`, `height`, `dataSize` or frame count just over its cap, each in turn | decoded | error; no frames returned |
| AC-4 | unit | a stream (each format) with extra bytes, including a further well-formed record section, before the trailer | decoded | exactly `frameCount` frames; the extra bytes change nothing |
| AC-5 | unit | a stream whose last frame's `dataSize` runs past the trailer | decoded | error; no frames returned |
| AC-6 | unit | RLE programs (each format) whose skip, blank-row or literal would move or paint past `width × height` | decoded | error; no frames returned |
| AC-7 | unit | a `.16a` stream decoded with palette declared absent | decoded | frames decode from offset 0; the palette is reported absent |
| AC-8 | unit | an empty stream; a 4-byte stream with palette declared | decoded | error; no crash |
| AC-9 | unit | a `.16` stream with runs ending in a zero and a non-zero high nibble, and a mid-run zero high nibble followed by further pixels | decoded | the grid matches exactly — the mid-run zero paints a value-0 pixel and every later pixel lands at its correct position |
| AC-10 | unit | `.16a` literal words spanning index 0–255 and level 0–15, level 0 included | decoded | each painted pixel preserves its exact `{index, level}`; a level-0 painted pixel is distinguishable from a transparent one |
| AC-11 | unit | a literal run (words for `.16a`, bytes for `.16`) truncated by its block end | decoded | error; no frames returned |
| AC-12 | manual | a real cursor sprite (`.16a`) and a real font glyph (`.16`) from a lawful install | dumped to images | upright and recognizable; the dump's presentation choices are stated in its output |
| AC-13 | unit | programs (each format) using op `0b11` | decoded | `.16a` output identical to the same program written with the blank-rows op; `.16` output identical to the skip op |
| AC-14 | unit | a stream whose trailer count is 0 (each format) | decoded | success: an empty frame list, the palette still returned when declared |

Error cases: AC-3, AC-5, AC-6, AC-8, AC-11.

## Derived properties

- **P-1** (invariant) every decoded frame's buffer is exactly `width × height`; every pixel is
  transparent or painted with in-range fields (index 0–255, level/value 0–15).
- **P-2** (negative-invariant) for any input byte sequence whatsoever the decoder does not panic
  and never reads or writes out of bounds; for any FR-6 error class, no frames and no palette are
  produced.
- **P-3** (invariant) decoded output is independent of trailer bit 31.
- **P-4** (negative-invariant) for any program that would move or paint outside the grid, no pixel
  outside the grid is written and the decode errors.
- **P-5** (completeness) in a successful decode every painted pixel comes from exactly one literal
  at that cursor position, and every pixel no literal painted is transparent — painted zeros
  included.

## I/O examples

### `.16a` (no palette declared)

Frame `width=2, height=2, dataSize=12`; RLE block (byte pairs are LE u16):

```
02 00   literal n=2
0A 1E   ss=0x1E0A → index 5, level 15   at (0,0)
0C 10   ss=0x100C → index 6, level 8    at (1,0)
01 80   skip n=1  → (0,1) stays transparent
01 00   literal n=1
0E 1E   ss=0x1E0E → index 7, level 15   at (1,1)
```

Result (`.` = transparent, cells `index/level`):

```
row 0:  5/15   6/8
row 1:  .      7/15
```

Numeric anchors: a skip of 551 pixels is `27 82` (`0x8227` — op `0b10`, `n = 0x227`), a count an
8-bit field could not carry. Trailer `01 00 00 80` = count 1 (bit 31 set, masked); `01 00 00 00`
decodes identically.

### `.16`

Frame `width=3, height=2, dataSize=7`:

```
02      literal n=2 bytes
5A      low 0xA → value 10 at (0,0); high 0x5 → value 5 at (1,0)
0C      final byte: low 0xC → value 12 at (2,0); high 0 → pad, not emitted
81      skip n=1 → (0,1) transparent
01      literal n=1 byte
0F      final byte: low 0xF → value 15 at (1,1); high 0 → pad
81      skip n=1 → (2,1) transparent
```

```
row 0:  10  5  12
row 1:  .   15  .
```

## Constraints and alternatives

Sanity caps on header fields — an external-contract choice:

| Option | Trade-off |
|---|---|
| **A. Reject over caps — chosen:** `width` or `height` > 2048, frame count > 4096, `dataSize` > 2^24, stream longer than 2^26 → error | every real sheet decodes with wide margin; a corrupt header cannot demand an absurd allocation; a breach is visible |
| B. Treat an oversized header as a "null frame": consume its 12 bytes, skip its RLE block, keep walking | an oversized-but-real frame loses its pixel block and the walk re-parses pixel data as headers — wrong frames, no error |
| C. No caps | a corrupt header turns into an allocation bounded only by a u32 |

- The decoder consumes one byte stream and returns structs; archive composition, palette sourcing
  and rendering are the consumer's concern. Raw preservation (FR-5) rules out any output type
  storing pre-resolved colours, expanded values or merged fields.

## Out of scope

- `.256` sprites (story 0002) — same container shape, different pixel grammar.
- Rendering and blending: resolving `{index, level}` or a glyph value to a displayed colour, level
  tables, framebuffer composition.
- Font semantics: glyph-to-character mapping, metrics, spacing, kerning — this story decodes the
  container only.
- Appended record sections after the counted records (real font streams carry them): ignored
  bytes under FR-4, never decoded.
- External and player-colour palettes; the palette entries' 4th byte; encoding or writing either
  format.

## Verification mapping

AC-1…AC-11, AC-13, AC-14 and P-1…P-5: CI-automatable unit tests over synthetic streams — no game
install. AC-12: a developer-run dump against a lawful install; evidence recorded, no game bytes
committed.

## Gate check

FR-1 → AC-1, AC-13, AC-14, P-1, P-5 · FR-2 → AC-9, AC-13, AC-14, P-1, P-5 · FR-3 → AC-7 ·
FR-4 → AC-2, AC-4, AC-14, P-3 · FR-5 → AC-1, AC-10, P-1 · FR-6 → AC-3, AC-5, AC-6, AC-8, AC-11,
P-2, P-4 · FR-7 → every unit AC · FR-8 → AC-12.
