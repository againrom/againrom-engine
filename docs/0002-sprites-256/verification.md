# Verification — `.256` sprite format decoder (ROM1)

## Environment

- Toolchain: Go 1.26.1 (`windows/amd64`), the version pinned in `go.mod`.
- Dependencies: none new. `pkg/formats/spr256` imports the standard library only (`encoding/binary`,
  `fmt`); `cmd/sprtool` adds `image`, `image/color`, `image/png`.
- The unit suite runs with **no game install present** — every fixture is a byte stream built in test
  code; `.256` carries no strings, so no CP866 byte appears.
- AC-9 requires a lawful GOG "Rage of Mages" (ROM1) install; **none is present in this environment**
  (`AGAINROM_ASSETS` unset, no ROM install on disk), so AC-9 is recorded as a manual developer step
  with its procedure and an explicit limitation — no game bytes and no converted PNGs are committed.

## Commands

```
go build ./...
go vet ./...
go test ./...
go test ./internal/archtest/          # proves the DAG allow-map edit (fail-closed)
gofmt -l $(git ls-files '*.go')
bash scripts/check-no-game-assets.sh            # tree scan
bash scripts/check-no-game-assets.sh --history  # full-history scan
```

Gate results: `build`, `vet`, `test` all pass; `internal/archtest` passes with the two new allow-map
rows (`pkg/formats/spr256` stdlib-only, `cmd/sprtool → {res, spr256}`); `gofmt -l` prints nothing; the
asset guard reports clean on both the tree and full-history scans.

## Acceptance criteria

| AC | Method | Evidence / outcome |
|---|---|---|
| AC-1 | Unit `TestDecodeTwoFrameGrids` | A two-frame stream exercising literal (`0x00`), transparent-pixel (`0x80`) and blank-row (`0x40`) ops, closed by trailer `0x80000002`, decodes to exactly two frames whose pixels, transparency, palette and count match the expected grids; the bit-31 trailer ends the frame list. **PASS** |
| AC-2 | Unit `TestPaletteBGRToRGB` | A `[B,G,R,reserved]` entry `{0x11,0x22,0x33,0x44}` returns `Color{R:0x33,G:0x22,B:0x11}` (reserved byte dropped); entry 0 (the reserved transparent key) is present, not zeroed. **PASS** |
| AC-3 | Unit `TestDecodeNoPaletteVariant` | A palette-less stream (frames from offset 0, trailer bit 31 clear) decodes its frame; `HasPalette == false`, `Palette == nil`, the count is honored. **PASS** |
| AC-4 | Unit `TestRejectFrameBlockPastTrailer` | A frame whose `dataSize` (100) runs past the trailer yields an error and a nil `*Sprite`. **PASS** |
| AC-5 | Unit `TestRejectRLETilingErrors` | An RLE block whose row tokens sum to less than `width`, and one that overruns `height`, each yield an error and a nil `*Sprite`. **PASS** |
| AC-6 | Unit `TestRejectLiteralOverrunsBlock` | A literal run (`0x05`) claiming more bytes than its block holds yields an error, no crash. **PASS** |
| AC-7 | Unit `TestRejectShortAndOversizedStreams` | A 0-byte stream, and a bit-31-set stream shorter than 1028 B, each reject; a near-2³² `width×height` header rejects via the representability guard rather than a `makeslice` panic. **PASS** |
| AC-8 | Unit `TestDecodeEmptyGridBetweenFrames` | A `width=0` empty-grid frame between two valid frames decodes to three frames; the empty one is `0×0` with a length-0 pixel slice, the two valid frames decode correctly. **PASS** |
| AC-9 | Manual / developer-run (`sprtool`) | **Not run — no lawful install present.** Procedure recorded below; visual confirmation is the developer's. |
| AC-10 | Unit `TestDecodeBucketBSingleFrame` | A Bucket-B stream (palette + one frame + trailer `0x80000001` + an appended `0x80000001 … 0x80000001` section) decodes to exactly one correct frame; the appended section — which opens with an impossible-width `0x80000001` that would be an invalid frame if parsed — is left untouched, no error. **PASS** |
| AC-11 | Unit `TestDecodeC0AliasesC80` | A `0xC0`-class control decodes identically to `0x80` (a transparent skip of `N`), byte-for-byte equal to the `0x82` decoding; no error. **PASS** |

## Properties

- **P-1 (buffer is `width × height`; each pixel transparent or an index 0–255).** Every decoded frame
  returns `len(Pixels) == Width*Height`; the empty grid (AC-8) is length 0. Pixels are `{Index, Opaque}`
  pairs — transparent is the zero value, opaque carries the literal's palette index (AC-1). Exercised by
  `TestDecodeTwoFrameGrids`, `TestDecodeEmptyGridBetweenFrames`.
- **P-2 (no panic / no out-of-bounds on any input).** All size arithmetic is `uint64` (two `uint32`
  factors cannot overflow); the pixel count is checked against the platform `int` max before any `make`,
  so a crafted near-2³² dimension pair is an atomic error, not a `makeslice` panic
  (`TestRejectShortAndOversizedStreams`). Every enumerated malformed class (block past the trailer,
  literal overrun, bad tiling, short/empty streams) returns `(nil, error)` with no crash — a panic would
  fail these tests. P-2 is thus exercised by the targeted rejection tests rather than a fuzz corpus; see
  Limitations for the one residual (a representable-but-absurd valid dimension allocates proportional
  memory).
- **P-3 (indices independent of the palette).** Decoded pixels carry palette **indices**, not resolved
  RGB; the same frame decodes identically whether a palette is present (AC-1) or absent (AC-3). Exercised
  by `TestDecodeNoPaletteVariant` (identical grid to the paletted case) and `TestPaletteBGRToRGB`.
- **P-4 (no write outside the grid).** Any op that would advance the cursor past `width × height`, or a
  block that fails to tile the grid exactly, yields an error before the frame is returned
  (`TestRejectRLETilingErrors`); the fill only ever writes at the in-range linear cursor.

## `sprtool` end-to-end (synthetic; SC-12 tool half)

To validate the `res → spr256.Decode → PNG` glue without any game asset, a synthetic `.res` archive was
built in a scratch directory (outside the repository) holding one entry `cur.256` whose payload is a
synthetic `.256` (1024-byte palette with two set entries + one `2×2` frame + trailer `0x80000001`).
`sprtool png <scratch>/smoke.res cur.256 <scratch>/out` reported `wrote 1 of 1 frames` and produced
`frame_000.png`, a valid `2×2` 8-bit RGBA PNG whose opaque pixels carry the palette colors and whose
skipped pixels are alpha 0. The scratch archive and PNG live outside the repository and are not
committed; the asset guard's tree scan is clean.

## AC-9 procedure (manual, developer-run against a lawful install)

For a developer with a lawful ROM1 install:

```
sprtool png <install>/graphics.res cursors/default.256 ./sprtool-out
```

Expected: one PNG per cursor frame under `./sprtool-out` (git-ignored). Confirm each frame is **upright**
and **visually recognizable as the in-game cursor**, and that the colors match the game's VGA palette
(the decoder applies the sprite's own 256-color palette). Record the frame count and the visual
confirmation here; **commit no game bytes and no converted PNGs** (the `sprtool-out/` folder is
git-ignored and the asset guard is the backstop). This step needs a human's visual judgment and a lawful
install, so it is not part of the automated suite and was not executed in this environment.

## Limitations and residual risks

- **AC-9 not executed here** — no lawful install is present. The decoder's correctness on real cursor
  data is inferred from the game-derived spec and the synthetic ACs; the upright/recognizable/VGA-color
  check awaits a developer run.
- **Representable-but-absurd valid dimensions (R-1 residual).** The `.256` grammar admits an
  arbitrarily wide all-blank frame from a tiny block (e.g. a one-byte blank-row op with a billion-wide
  header), which any faithful dense-grid decoder must materialize; such a frame allocates memory
  proportional to `width × height`. The representability guard keeps this from becoming a `makeslice`
  panic (P-2), but does not cap the allocation — no real `.256` frame approaches it (corpus max
  `640 × 480`), and the spec forbids a dimension/skip heuristic that would misclassify real frames.
- **Bucket-B appended section is not interpreted.** The decoder returns exactly `frameCount` frames and
  leaves the appended section unread, matching the shipped loader (runtime-inert); its authoring origin
  is out of scope.
- **Overlay blend and the attack/pickup banded anomaly are out of scope** (engine/runtime and open in
  the research); the decoder decodes those frames by the same grammar, and compositing/interpretation is
  the consumer's concern.

## Conclusion

Every automated acceptance criterion has passing evidence: AC-1…AC-8, AC-10 and AC-11 by synthetic unit
tests, with P-1…P-4 exercised across them and the `sprtool` glue validated end-to-end on a synthetic
`.res`/`.256` pair. AC-9 is a documented manual step, not executed here for lack of a lawful install. The
decoder meets its contract as specified; no game data entered the repository (asset guard clean on tree
and full history).
