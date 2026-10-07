# Plan — `.256` sprite format decoder (ROM1)

**Intensity:** spec-anchored / static (PROFILE default for `pkg/formats/*` — the format spec is a
durable contract that doubles as reverse-engineering documentation). **Terrain:** greenfield (no
`.256` code exists in this repo; `pkg/formats/spr256/` does not yet exist).

## Approach

Implement a pure, read-only decoder for the ROM1 `.256` paletted sprite in a new leaf package
`pkg/formats/spr256`, derived entirely from `spec.md` (FR-1…FR-6, AC-1…AC-11, P-1…P-4). `Decode`
takes the raw `.256` bytes and returns an ordered list of frames plus the RGB palette (when present),
modelling the shipped loader exactly (EXP-0016 as cited in the spec): it reads the 4-byte trailer,
takes `frameCount = trailer & 0x7FFFFFFF` and `hasPalette = trailer & 0x80000000`, reads the leading
1024-byte palette iff `hasPalette`, then reads **exactly `frameCount`** frame records sequentially
from the palette's end (or offset 0), decoding each RLE block onto a `width × height` grid of
palette-indexed / transparent pixels. Malformed input is rejected atomically with an error and no
output, never a panic. A companion developer tool `cmd/sprtool` (`png <archive> <path> <dir>`) reads a
`.256` entry through the 0001 `pkg/formats/res` reader, decodes it, and writes one PNG per frame into a
git-ignored directory for verification against a lawful install — never part of the test suite. The
new package and tool are registered in the `internal/archtest` DAG allow-map (fail-closed) and
`docs/ARCHITECTURE.md`; the package imports **stdlib only** (no `golang.org/x/text` — `.256` carries no
strings).

## Facts verified during planning

- **Baseline code.** `pkg/formats/spr256` does not exist. `pkg/formats/{res,reg,alm}` exist;
  `pkg/formats/res` is **fully implemented** (`res.go`: `Open`, `OpenBytes`, `Entry`,
  `(*Archive).Entries`, `(*Archive).ReadFile`) — so `sprtool`'s archive-path variant (FR-6) is
  implementable now, not a TODO. `cmd/{restool,regtool,almtool}` exist; `cmd/sprtool` does not.
- **DAG.** `internal/archtest`'s allow-map is **fail-closed**: a module package absent from the map
  fails `TestLiveTreeClean` (`internal/archtest/dag.go`). `pkg/formats/spr256` and `cmd/sprtool` are
  not yet listed, so each must be added alongside the code that introduces it. `externalAllowed`
  grants `golang.org/x/text` only to `pkg/formats/*`; `spr256` needs no external import, so no
  `externalAllowed` change is required. `cmd/restool` is mapped to `{pkg/formats/res, pkg/vfs}`; the
  new `cmd/sprtool` mirrors that shape with `{pkg/formats/res, pkg/formats/spr256}`.
- **Format facts** come from the research submodule via `spec.md` (self-contained, S-2). Load-bearing
  for the design: the trailer is `[31-bit frameCount][bit31 = has-palette]`
  (SPR256-TRLR-021 as cited); the loader builds a table of exactly `frameCount` frames and never
  indexes the Bucket-B appended section (SPR256-EXC-020 / AC-10); the RLE opcode dispatch is a
  three-way `if(0x00)/elif(0x40)/else`, the `else` handling `0x80` and `0xC0` alike
  (SPR256-RLE-020 / AC-11); palette entries are `[B,G,R,reserved]`, BGR order (AC-2); corpus max width
  is **640**, falsifying any 512 cap (SPR256-CORPUS-006), so the decoder applies no dimension
  threshold. The spec is the authority; none of these are re-derived here.
- **Purity precedent.** `pkg/formats/res` is a pure leaf (bytes in, structs out, error-not-panic on
  malformed input, no global state); `spr256` follows the same shape and is exercised by synthetic
  in-code byte fixtures with no game install present, matching `res_test.go`.
- **Asset-guard / ignore.** `.gitignore` already globs `*.256` regardless of directory and
  `scripts/check-no-game-assets.sh` scans git-tracked content by asset extension only; `.png` is
  **not** an asset extension the guard keys on, so `sprtool`'s decoded-frame PNGs (converted game
  assets under Golden Rule 1) need an explicit ignore rule to keep them from being staged — a
  dedicated git-ignored output directory is added for that.

## Files to touch

Module `againrom` (this repo):

- `pkg/formats/spr256/spr256.go` — **ADD**. The decoder: `Decode(data []byte) (*Sprite, error)`;
  types `Sprite`, `Frame`, `Pixel`, `Color`; trailer read + palette-presence (DD3), palette BGR→RGB
  decode (DD8), the read-exactly-`frameCount` frame walk (DD2), the RLE grammar with the `0xC0`→`0x80`
  alias (DD6), exact-tiling validation and the FR-4 atomic-error set (DD7), overflow-safe sizing with
  the representability guard (DD5).
- `pkg/formats/spr256/doc.go` — **ADD**. Package comment: the format, the leaf-tier import rule
  (stdlib only), a pointer to `docs/0002-sprites-256/spec.md`.
- `pkg/formats/spr256/spr256_test.go` — **ADD**. Synthetic unit tests (AC-1…AC-8, AC-10, AC-11) built
  from in-code byte streams via a fixture builder; asserts atomic rejection + no panic for the error
  ACs (P-2). Never reads a game install.
- `internal/archtest/dag.go` — **MODIFY**. Add two allow-map rows: `"pkg/formats/spr256": {}`
  (stdlib-only leaf) and `"cmd/sprtool": {"pkg/formats/res", "pkg/formats/spr256"}`.
- `docs/ARCHITECTURE.md` — **MODIFY**. Add `pkg/formats/spr256` to the tier and DAG tables and
  `cmd/sprtool` to the cmd rows, keeping the doc identical to the allow-map.
- `cmd/sprtool/main.go` — **ADD**. `png <archive> <path> <dir>`: open the archive via
  `pkg/formats/res`, read the entry, `spr256.Decode`, write one PNG per frame under `<dir>`. Errors to
  stderr, non-zero exit. No test file (developer-run only, per FR-6 / spec Out-of-scope).
- `.gitignore` — **MODIFY**. Add a dedicated git-ignored output directory for `sprtool`'s decoded PNGs
  (converted assets must never be tracked; the asset guard does not key on `.png`).

## Design decisions

Every architectural choice is settled here; each records the alternative it beat.

- **DD1 — In-memory byte-slice decode; atomic `(*Sprite, error)`.** `Decode([]byte)` parses a whole
  in-memory `.256` stream (the bytes the 0001 `res` reader returns) and yields `(*Sprite, nil)` on
  success or `(nil, error)` on any rejection — never a partial `Sprite` (FR-4). This is the seam the
  synthetic tests drive. *Rejected:* an `io.Reader`/streaming decode — the trailer is at the **end**
  of the stream (the loader seeks there first), so a forward-only reader cannot even begin without
  buffering the whole input; a leaf decoder over an already-in-memory archive entry gains nothing from
  streaming.
- **DD2 — Read exactly `frameCount` frame records; leftover bytes are not frames.** The trailer's
  `frameCount` drives the walk: the decoder reads exactly that many records sequentially from the
  frame region and then stops, matching the shipped loader which "builds a table of exactly one frame
  and never indexes the appended section" (spec Bucket-B / AC-10). For a standard sprite the last
  record ends exactly at the trailer; for a Bucket-B sprite (`frameCount = 1` + an appended
  `0x80000001 … 0x80000001` section) the appended bytes sit between the single frame and the trailer
  and are **left unread** with no error. The trailer's bit-31 "impossible width" sentinel is thus an
  emergent cross-check, not the stopping rule. *Rejected:* walking frames until the cursor reaches the
  trailer offset — it would parse Bucket-B's appended `0x80000001` as a frame header (an impossible
  width) and error, contradicting AC-10; *rejected:* stopping at the first impossible-width sentinel
  — it happens to also satisfy AC-10, but EXP-0016 shows the loader is `frameCount`-driven, so
  `frameCount` is the authoritative rule and the sentinel is corroboration.
- **DD3 — Palette presence from trailer bit 31 alone.** `hasPalette = trailer & 0x80000000 != 0`
  decides whether the leading 1024 bytes are read (FR-2), exactly as the loader does — never a
  size/width heuristic on the body. The frame region starts at offset 1024 when present, else 0.
  *Rejected:* inferring a palette from stream length or a plausible-width scan — the spec forbids a
  content heuristic (FR-2) and the no-palette arrow variant is self-described by bit 31.
- **DD4 — `Pixel{Index uint8; Opaque bool}`; transparent is the zero value.** Each grid cell is a
  palette **index** with an `Opaque` flag; a transparent cell is the zero value `{0, false}`. The
  freshly-allocated grid is therefore all-transparent, and only a literal run writes opaque pixels;
  blank-row and transparent-skip ops merely advance the cursor. This preserves indices (FR-3, P-3 —
  no RGB resolved at decode) and carries transparency **structurally**, independent of index 0's value
  (index 0 is the reserved key but is not treated specially in decode). *Rejected:* an `int16` grid
  with `-1` = transparent — it conflates transparency with the index value, and a literal that
  legitimately paints index 0 would be indistinguishable from a skip; *rejected:* resolving pixels to
  RGB at decode time — violates FR-3 (a consumer must be able to apply an alternative palette).
- **DD5 — Overflow-safe sizing + a representability guard; no dimension threshold.** All size
  arithmetic (`width × height`, block bounds, cursor advances) is done in `uint64`, which cannot
  overflow for two `uint32` factors ((2³²−1)² < 2⁶⁴). Before allocating a frame grid the pixel count
  is checked to fit `int`; a count that does not (only reachable with near-2³² dimensions) is an atomic
  FR-4 error rather than a `makeslice` panic (P-2). No `width`/`height` value is otherwise rejected or
  used to skip a frame — the corpus reaches width 640 (SPR256-CORPUS-006), so any cap in the
  513–640 range (or a "large width ⇒ null frame" skip) would misclassify real frames and desync the
  stream. *Rejected:* a `width > 512` null-frame skip heuristic — falsified by the corpus and would
  desync; *rejected:* trusting `width × height` into `make` unchecked — a crafted near-2³² pair
  overflows `int` and panics in `makeslice`, violating P-2. *Residual (see Risks R-1):* a
  validly-framed but absurdly-large frame (e.g. a one-byte all-blank row with a billion-wide header)
  allocates memory proportional to its dimensions; this is inherent to any faithful dense-grid decoder
  of the format and is bounded away from a `make` panic by the representability guard.
- **DD6 — RLE opcode dispatch `class = c & 0xC0`, `N = c & 0x3F`; `0xC0` aliases `0x80`.** Three
  branches mirroring the loader: `0x00` literal (paint the next `N` opaque palette-index bytes,
  cursor +1 each), `0x40` blank rows (emit `N` fully-transparent rows; valid only at a row boundary /
  column 0), and an `else` handling both `0x80` **and** `0xC0` as a transparent skip of `N` pixels
  (SPR256-RLE-020 / AC-11). `N = 0` is a no-op for every class. *Rejected:* treating `0xC0` as a
  fourth opcode or an error — the shipped loader has no fourth branch; `0xC0` is a decode-time alias,
  and rejecting it would diverge from the loader (AC-11 forbids the error).
- **DD7 — Exact-tiling validation with per-pixel column wrap; the FR-4 error set.** A cursor advances
  left→right, wrapping to the next row when the column reaches `width` (tracked explicitly, never a
  modulo, so `width = 0` is division-safe). A frame is accepted iff, after the block is fully
  consumed, exactly `height` rows were produced and the column is back at 0 (which forces the linear
  cursor onto `width × height` exactly). The atomic errors are: a literal run whose bytes overrun its
  block; a blank-row op not at column 0; any op that would advance the cursor past `width × height`; a
  frame header or RLE block that does not fit before the trailer; and a `frameCount` the frame region
  cannot supply. *Rejected:* a lenient decode that pads short rows or truncates overruns — it would
  silently accept corrupt streams, dropping the FR-4 / P-1 / P-4 guarantees the format's exact-tiling
  property (RLE-008) is meant to enforce.
- **DD8 — Palette returned as 256 RGB `Color`s, BGR→RGB reordered; `HasPalette` flag.** Each 4-byte
  entry `[B,G,R,reserved]` becomes `Color{R,G,B}` (the reserved 4th byte is dropped); the decoder
  returns `Palette` (256 entries when present, `nil` when absent) plus a `HasPalette` bool so a
  consumer can distinguish "absent" (AC-3) from a present palette whose entry 0 is black. Index 0 is
  returned verbatim as the reserved transparent key — not zeroed or dropped (FR-3). *Rejected:*
  pre-multiplying transparency into the palette or omitting entry 0 — a consumer needs the full 256
  and applies transparency from the pixel `Opaque` flag, not from the palette.
- **DD9 — `cmd/sprtool` reads via `pkg/formats/res`.** Because 0001's `res` reader is implemented,
  `sprtool png <archive> <path> <dir>` opens the archive with `res.Open`, reads the entry bytes with
  `ReadFile`, `spr256.Decode`s them, and writes one `image/png` per frame (opaque pixels mapped
  through the palette to RGBA, transparent pixels to alpha 0) into `<dir>`. It is a developer tool,
  never compiled into or run by the test suite, and points at a git-ignored directory so decoded
  (converted) assets are never tracked. *Rejected:* a raw-`.256`-file-only tool with the archive path
  left as a TODO — that fallback is only warranted if `res` were spec-only, which it is not; the
  archive path FR-6 specifies is available today.
- **DD10 — Register both new packages in the DAG allow-map and ARCHITECTURE, with the code that needs
  them.** `pkg/formats/spr256` (stdlib-only, empty intra-module set) lands with the decoder;
  `cmd/sprtool` (`{pkg/formats/res, pkg/formats/spr256}`) lands with the tool. Each pairing keeps its
  commit green against the fail-closed live-tree check. `docs/ARCHITECTURE.md` is updated in lockstep
  so the human-readable DAG stays identical to the allow-map. *Rejected:* adding both allow-map rows
  up front in one commit — the `cmd/sprtool` row would reference a package that does not yet exist in
  that commit, muddying the task↔commit mapping; each row ships with its package.

## Risks (product)

- **R-1 (plan) — hostile dimensions: `makeslice` panic / unbounded allocation.** A crafted frame
  header with near-2³² `width`/`height` could overflow the `int` passed to `make` (panic) or a
  validly-framed billion-wide all-blank row could allocate gigabytes, threatening P-2.
  *Mitigation:* all sizing in `uint64` (no wrap), a representability guard that turns an
  `int`-overflowing pixel count into an atomic error before any `make` (DD5), and negative unit
  assertions that hostile headers yield an error and no panic. The residual proportional allocation for
  representable-but-absurd valid dimensions is inherent to a dense-grid decoder and documented as a
  limitation, not a decode-correctness gap.
- **R-2 (plan) — Bucket-B appended section mis-parsed as frames.** Walking to the trailer offset would
  read the appended `0x80000001` bracket as a frame and reject a legitimate Bucket-B sprite.
  *Mitigation:* the read-exactly-`frameCount` walk (DD2) stops after the single frame and leaves the
  appended section unread; AC-10 exercises it with an appended section that would be an invalid frame
  if parsed.
- **R-3 (plan) — `width = 0` division / non-tiling.** A zero-width frame could divide-by-zero in a
  modulo-based column wrap, or fail to tile. *Mitigation:* column wrap is tracked by explicit
  comparison, never modulo (DD7); a zero-width grid is `n = 0` and accepts an empty (or blank-row)
  block; AC-8 exercises a `width = 0` frame between two valid frames.
- **R-4 (plan) — `0xC0` wrongly rejected.** Treating `0xC0` as an unknown opcode would reject a valid
  (if data-unused) control. *Mitigation:* `0xC0` aliases `0x80` in the shared `else` branch (DD6);
  AC-11 asserts it decodes identically to `0x80`.

## Success criteria

Each maps to an upstream requirement and a verification method (unit = synthetic Go test over an
in-code byte fixture; archtest = the existing `internal/archtest` live-tree check; dev-run = `sprtool`
against a lawful install).

1. **SC-1 (FR-1, AC-1, P-1; DD2, DD4, DD7)** — a synthetic two-frame stream exercising literal, `0x80`
   transparent-pixel and `0x40` blank-row ops, closed by trailer `0x80000002`, decodes to two frames
   whose pixels, transparency, palette and count match the expected grids exactly. *Test:*
   `TestDecodeTwoFrameGrids`.
2. **SC-2 (FR-3, AC-2, P-3; DD8)** — a palette entry in `[B,G,R,reserved]` order is returned as its
   RGB reordering; entry 0 is present as the reserved transparent key. *Test:* `TestPaletteBGRToRGB`.
3. **SC-3 (FR-1, FR-2, AC-3; DD3)** — a palette-less stream (frames from offset 0) closed by a trailer
   with bit 31 clear decodes its frames and reports `HasPalette == false` with the count honored.
   *Test:* `TestDecodeNoPaletteVariant`.
4. **SC-4 (FR-4, AC-4, P-2)** — a stream whose last frame's `dataSize` runs past the trailer yields an
   error and a nil sprite, no panic. *Test:* `TestRejectFrameBlockPastTrailer`.
5. **SC-5 (FR-4, AC-5, P-1/P-4)** — an RLE block whose row tokens do not sum to `width`, and one that
   overruns `height`, each yield an error and a nil sprite. *Test:* `TestRejectRLETilingErrors`.
6. **SC-6 (FR-4, AC-6, P-2)** — an RLE literal run longer than its block yields an error with no crash.
   *Test:* `TestRejectLiteralOverrunsBlock`.
7. **SC-7 (FR-2, FR-4, AC-7, P-2; DD3, DD5)** — a 0-byte stream, and a stream whose trailer has bit 31
   set but which is shorter than 1028 B, each yield an error with no crash; a near-2³² dimension header
   yields an error rather than a `makeslice` panic. *Test:* `TestRejectShortAndOversizedStreams`.
8. **SC-8 (FR-1, AC-8, P-1; DD7)** — a `width = 0` empty-grid frame between two valid frames decodes to
   three frames, the empty one of size 0 and the two valid frames correct. *Test:*
   `TestDecodeEmptyGridBetweenFrames`.
9. **SC-9 (FR-1, AC-10; DD2)** — a Bucket-B-shaped stream (palette + one frame + trailer `0x80000001` +
   an appended `0x80000001 … 0x80000001` section) decodes to exactly one correct frame with the
   appended section left untouched and no error. *Test:* `TestDecodeBucketBSingleFrame`.
10. **SC-10 (FR-1, AC-11; DD6)** — a frame whose RLE contains a `0xC0`-class control decodes it
    identically to `0x80` (a transparent skip of `N`) with no error. *Test:* `TestDecodeC0AliasesC80`.
11. **SC-11 (FR-5)** — `pkg/formats/spr256` imports only the standard library (no external, no other
    intra-module package), and `cmd/sprtool` imports only `pkg/formats/res`, `pkg/formats/spr256` and
    stdlib; the fail-closed DAG check stays green with both packages registered. *Test:* the existing
    `internal/archtest` live-tree check.
12. **SC-12 (FR-6, AC-9)** — `sprtool png <archive> <path> <dir>` decodes one `.256` entry through the
    `res` reader and writes one PNG per frame to a git-ignored directory; run against a lawful
    `cursors/default.256` it yields an upright, recognizable cursor whose palette matches the VGA
    colors, recorded as evidence — no game bytes committed. *Method:* developer-run (manual), output
    directory git-ignored.
