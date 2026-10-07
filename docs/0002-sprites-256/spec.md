# Spec — `.256` sprite format decoder (ROM1)

**Provenance basis.** The `.256` layout, RLE pixel grammar, palette, and trailer below are game-derived
in the parallel research repo (the `research/` submodule): `research/formats/spr256/format.md` across
experiments EXP-0002 (structure), EXP-0003 (RLE decode), EXP-0004 (palette), EXP-0005 (overlay class),
EXP-0013 (trailer / frame count / residual reads), and EXP-0016 (the `CSprite256` loader read directly
from `rom.exe`). Confidence is the research's own: the RLE grammar is **exact / lossless across all
23 791 frames** (SPR256-RLE-008), the palette byte order is pinned to the game's own VGA palette
(SPR256-PAL-011), and the trailer — `[31-bit frameCount][bit31 = has-palette]` — is read straight from
the shipped loader, which computes `frameCount = trailer & 0x7FFFFFFF` and reads the palette iff bit 31
is set (SPR256-TRLR-021), cross-checked by the record walk (SPR256-COUNT-002). This story **implements
the clean decoder from that layout and verifies it against a lawful install** (AC-9); no decoder or
extracted pixels exist in this repo yet. It reads a byte stream produced by the 0001 `.res` reader
(`pkg/formats/res`) but does not import it — composition is the consumer's concern. Greenfield.
Submodule pin: `research/` at `72f9d47`.

Every layout, opcode, palette, and trailer fact below is derived from the research submodule (claim IDs
cited per row); this story requires no reverse-engineering of its own and no open research.

## Problem / goal

The engine can open `.res` archives (0001) but cannot decode image data. Unit, structure, cursor and
UI sprites are stored as `.256` files inside those archives. This story adds a pure decoder: a paletted
`.256` byte stream in, an ordered list of frames + an RGB palette out. Pure `formats` leaf — bytes in,
structs out; no archive, VFS, engine, or rendering knowledge. `pkg/formats/spr256` imports **stdlib
only** (no CP866: `.256` carries no strings).

## Format definition (game-derived, little-endian)

Source: the research submodule — `research/formats/spr256/format.md` (claim IDs cited per row). A
standard `.256` stream is a fixed-size palette, a sequence of frame records, and a 4-byte trailer:

```
[ 1024 B palette ][ frame, frame, … ][ 4 B trailer ]
```

Data payloads and record sizes are read strictly sequentially. The **4-byte trailer is
`[31-bit frameCount][bit31 = has-palette]`** — `frameCount = trailer & 0x7FFFFFFF` equals the number of
frame records between the palette and the trailer (the two agree exactly), and bit 31 is the has-palette
flag the loader tests before reading the leading 1024 bytes (SPR256-TRLR-021, SPR256-COUNT-002). `u32@0`
of the stream is palette entry 0 (≈always 0), **not** a count.

**Palette (1024 bytes, present iff the trailer's bit 31 is set — SPR256-TRLR-021):**

| Region | Size | Content | Basis (research claim) |
|---|---|---|---|
| Palette | 1024 B | 256 entries × 4 bytes `[B, G, R, reserved]` — byte order **BGR**, 4th byte 0 in 99.96% | SPR256-PAL-003, SPR256-PAL-011 |

The decoder returns the palette as 256 RGB colors (reordering BGR→RGB). **Index 0 is the reserved
transparent/background key** — 0 literal occurrences across 9.54 M literal pixels (SPR256-PAL-012); all
transparency is carried structurally by the RLE opcodes below, so index 0 never appears as a painted
pixel. The reserved 4th byte is 0 for every standard palette; the only non-zero cases (138 / 350 720
entries) are entirely within `cursors/pickup.256` + `attack.256`, whose leading 1024 B is not a
standard palette at all (SPR256-PAL-018) — the OVL-015 anomaly, out of scope below.

**Frame record** (repeat until the trailer — SPR256-STRUCT-001):

| Field | Size | Content |
|---|---|---|
| `width` | u32 | frame width in pixels (corpus max 640 — SPR256-CORPUS-006) |
| `height` | u32 | frame height in pixels (corpus max 480 — SPR256-CORPUS-006) |
| `dataSize` | u32 | byte length of the RLE block that follows |
| RLE block | `dataSize` B | pixel program — see below |

`width = 0` or `height = 0` is a valid header describing an empty grid.

**Trailer (last 4 bytes):** `[31-bit frameCount][bit31 = has-palette]` little-endian
(SPR256-TRLR-016/021). `frameCount = trailer & 0x7FFFFFFF`. **Bit 31** is the has-palette flag: the
loader reads the leading 1024-byte palette iff it is set (SPR256-TRLR-021). For a palette-bearing sprite
bit 31 is set, which also makes the trailer's first `u32` an impossible width — the **end-of-frames
sentinel** a frame-walker stops at. The 6 no-palette arrows are the same format with bit 31 clear (a
plain `u32` count, no sentinel — see Variants). Either way the count equals the record walk.

**RLE program** (SPR256-RLE-007/008/009/010). The block decodes a `width × height` grid. A cursor moves
left→right and wraps to the next row at `width`; the background is transparent (no pixel emitted). Each
control byte `c` is `[2-bit opcode | 6-bit count]` — `class = c & 0xC0`, `N = c & 0x3F`:

| Class | Effect | Basis |
|---|---|---|
| `0x00` | literal: the next `N` bytes are palette indices; each paints one opaque pixel, cursor +1 each | SPR256-RLE-007 |
| `0x40` | blank rows: emit `N` fully-transparent rows (occurs only at a row boundary, column 0) | SPR256-RLE-007/008 |
| `0x80` | transparent: emit `N` transparent pixels, cursor +`N` | SPR256-RLE-007 |
| `0xC0` | **alias of `0x80`** — the loader decodes it as a transparent skip of `N` (shared `else` branch, no 4th opcode); never emitted by ROM1 data (0 / 23 895 frames) | SPR256-RLE-020, SPR256-RLE-009 |

The decode is **exact**: every row's tokens sum to exactly `width`, the block yields exactly `height`
rows, and the block is consumed with no leftover byte (SPR256-RLE-008). `N = 0` paints/moves nothing.
The grammar is corpus-universal (SPR256-RLE-010).

**Variants (research-established):**

- **No-palette variant** — 6 projectile-arrow sprites carry no 1024-byte palette; frames start at
  offset 0. This is the **same format with the trailer's has-palette bit clear** (SPR256-TRLR-021): the
  decoder reads bit 31, sees it clear, and skips the palette read (FR-2). Which files these are, and the
  external palette they should borrow, is not in the stream and is the consumer's concern
  (SPR256-VAR-004) — but *whether* a palette is present is self-described by the trailer.
- **Frame + trailing section (Bucket-B)** — 8 sprites are a normal palette + one frame + its
  count-trailer `0x80000001`, then an **appended secondary section bracketed `0x80000001 … 0x80000001`**
  (SPR256-EXC-005/017). The shipped loader reads the *last* 4 bytes as the trailer (`frameCount = 1`),
  builds a table of exactly one frame, and **never indexes, decodes, or blits the appended section** —
  it is runtime-inert (SPR256-EXC-020). The decoder matches this exactly: it returns the single frame up
  to the terminating trailer and leaves the appended section unread. Its authoring origin is non-runtime
  and out of scope.

## Functional requirements

- **FR-1 (decode)** — `Decode(data)` reads the 4-byte trailer, takes `frameCount = trailer & 0x7FFFFFFF`
  and `hasPalette = trailer & 0x80000000`, reads the leading 1024-byte palette iff `hasPalette`, walks
  frame records from the palette's end (or offset 0 when absent) to the trailer, decodes each RLE block,
  and returns an ordered list of frames — each a `width × height` grid where every pixel is either
  **transparent** or an opaque 0–255 palette index — plus the 256-color RGB palette when present. Frame
  order is stream order. Framing ends at the trailer; the number of frames read MUST equal `frameCount`.
  For a palette-bearing stream the trailer's first `u32` also reads as an impossible width (bit 31 set),
  the end-of-frames sentinel (SPR256-TRLR-016/021).
- **FR-2 (palette presence from the trailer, never a content heuristic)** — the decoder determines
  whether a 1024-byte palette is present from the trailer's **bit 31** alone (SPR256-TRLR-021), exactly
  as the shipped loader does; it MUST NOT infer presence from a width/size heuristic on the stream body.
  The external palette a no-palette sprite borrows remains the consumer's concern (SPR256-VAR-004).
- **FR-3 (preserve indices)** — decoded pixels keep their palette **indices**, not pre-resolved RGB, so
  a consumer may apply an alternative palette (e.g. the `spritesb` overlay case) to the same frames.
- **FR-4 (atomic rejection, never panic)** — reject with an error and no result: a stream shorter than
  its fixed regions (1028 B with palette, 4 B without); a frame header or RLE block that does not fit
  before the trailer; a literal run overrunning its RLE block; an RLE program whose row tokens do not
  sum to `width`, that does not yield exactly `height` rows, or that leaves the cursor past
  `width × height`; a trailer whose `frameCount` (`trailer & 0x7FFFFFFF`) disagrees with the number of
  frames read. Rejection is atomic — an error value and no frames or palette, never a crash or partial
  output presented as success. Error classes need not be distinguishable. *(A `0xC0`-class control is
  **not** rejected — it decodes as `0x80`, matching the loader; see FR-1 / the RLE table.)*
- **FR-5 (purity)** — package `formats/spr256` imports only stdlib; it takes a byte source and returns
  structs; it knows nothing about archives, the VFS, palettes-from-other-files, or rendering. No
  floats, no global state.
- **FR-6 (dump tool)** — `cmd/sprtool` (or an extension of a shared tool) exposes at least
  `png <archive> <path> <dir>`: decode one `.256` entry (via the 0001 reader) and write each frame to a
  PNG in a git-ignored folder, for developer-run verification against a lawful install — never part of
  the test suite.

## Acceptance criteria (synthetic streams, no GOG assets)

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic two-frame stream exercising literal, transparent-pixel (`0x80`) and blank-row (`0x40`) ops, closed by trailer `0x80000002` (frameCount 2, bit 31 set) | decoded | pixels, transparency, palette and frame count match the expected grids exactly; the bit-31 sentinel ends the frame list |
| AC-2 | unit | a stream with a palette entry in `[B,G,R,reserved]` order | decoded | the returned palette color is the RGB reordering; index 0 is the transparent key |
| AC-3 | unit | a palette-less stream (frames from offset 0) closed by a trailer with bit 31 clear (plain `u32` count) | decoded | frames decode as in AC-1; the decoder reads bit 31 = 0 and reports the palette absent; the count is honored (SPR256-VAR-004, SPR256-TRLR-021) |
| AC-4 | unit | a stream whose last frame's `dataSize` runs past the trailer | decoded | error; no frames returned |
| AC-5 | unit | an RLE block whose row tokens do not sum to `width`, or that overruns `height` | decoded | error; no frames returned |
| AC-6 | unit | an RLE literal run longer than its block | decoded | error; no crash |
| AC-7 | unit | an empty (0-byte) stream; separately a stream whose trailer has bit 31 set but which is shorter than 1028 B | decoded | error; no crash |
| AC-8 | unit | a frame with `width = 0` (empty grid) between two valid frames | decoded | three frames; the empty grid is size 0, the two valid frames decode correctly |
| AC-9 | manual | a real GOG cursor sprite `cursors/default.256` (via the 0001 archive reader, lawful install, developer-run) | decoded to PNG | the image is upright and visually recognizable as the in-game cursor; palette matches the VGA colors; results recorded in `verification.md` — no game bytes committed |
| AC-10 | unit | a Bucket-B-shaped stream: palette + one frame + trailer `0x80000001` + an appended section bracketed `0x80000001 … 0x80000001` | decoded | exactly one frame is returned (decoded correctly); the appended section is left untouched, matching the shipped loader; no error (SPR256-EXC-020) |
| AC-11 | unit | a frame whose RLE contains a `0xC0`-class control byte | decoded | the byte decodes identically to `0x80` — a transparent skip of `N = c & 0x3F` pixels — matching the loader; no error (SPR256-RLE-020) |

Error cases: AC-4, AC-5, AC-6, AC-7. AC-1…AC-3, AC-8, AC-10, AC-11 verify the decoder's behavior on
synthetic input against the shipped-loader semantics (EXP-0016).

## Derived properties

- **P-1** (invariant) Every decoded frame's buffer is exactly `width × height`; each pixel is
  transparent or carries a palette index 0–255.
- **P-2** (negative-invariant) For any input byte sequence whatsoever, the decoder does not panic and
  never reads or writes out of bounds; for any FR-4 error class, no frames or palette are produced.
- **P-3** (invariant) Decoded pixels are palette indices, independent of the concrete palette; the same
  frames decode identically whether the palette is present, absent, or replaced.
- **P-4** (negative-invariant) For any RLE program that would paint or skip outside the `width × height`
  grid, no pixel outside the grid is written and the frame yields an error.

## Out of scope

- `.16` / `.16A` sprite formats (same family, different pixel encodings) — their own stories.
- The `spritesb.256` **overlay blend rule** (additive glow / team-tint / replace) — engine/runtime, not
  the format (SPR256-OVL-014, open in the research). The decoder decodes overlay frames correctly
  (sparse, over the base palette); compositing is the consumer's concern.
- The `cursors/attack.256` + `pickup.256` banded anomaly (SPR256-OVL-015 / PAL-018, open in the
  research) — their leading 1024 B is not a standard palette; interpreting them is not this story.
- External palette sources: player-color palettes, borrowed sibling palettes, per-class attribute flags
  that tell the game whether a sprite is paletted — engine data, not this format.
- The Bucket-B inner block's authoring origin (loaded but runtime-inert — SPR256-EXC-020); rendering,
  atlasing, animation sequencing, per-class frame-block roles, frame anchor points (the format carries
  none); writing/encoding `.256`.

## Gate check

FR-1 → AC-1, AC-3, AC-8, AC-10, AC-11, P-1 · FR-2 → AC-3, AC-7 · FR-3 → AC-2, P-3 · FR-4 → AC-4, AC-5,
AC-6, AC-7, P-2, P-4 · FR-5 → (archtest DAG: `pkg/formats/spr256` stdlib-only) · FR-6 → AC-9.
