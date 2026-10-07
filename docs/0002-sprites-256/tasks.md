# Tasks — `.256` sprite format decoder (ROM1)

**Reading key.** `FR-x` / `AC-x` / `P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DD-x` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks. Task kinds: **implementation** (one coherent
product change → exactly one implementation commit, trailer `SDD-Task: 0002-sprites-256/T<n>`),
**developer-run verification** (agent authors, a human runs against a lawful install; no
implementation commit).

## T1 — decoder package + DAG registration  *(implementation)*

Add the pure `.256` decoder and register it in the fail-closed DAG in the same change (DD10: the
allow-map row and the package are only correct together — an unregistered package fails the live-tree
check).

- Files: `pkg/formats/spr256/spr256.go` (ADD) — `Decode(data []byte) (*Sprite, error)`; types
  `Sprite{HasPalette bool; Palette []Color; Frames []Frame}`, `Frame{Width, Height int; Pixels
  []Pixel}`, `Pixel{Index uint8; Opaque bool}`, `Color{R, G, B uint8}`; trailer read + bit-31
  palette-presence (DD3), BGR→RGB palette decode (DD8), the read-exactly-`frameCount` frame walk
  (DD2), the RLE grammar with the `0xC0`→`0x80` alias (DD6), exact-tiling validation + the FR-4 atomic
  error set (DD7), overflow-safe sizing + representability guard (DD5). `pkg/formats/spr256/doc.go`
  (ADD) — package comment (format, stdlib-only leaf tier, pointer to `spec.md`).
  `internal/archtest/dag.go` (MODIFY) — add allow-map row `"pkg/formats/spr256": {}`.
  `docs/ARCHITECTURE.md` (MODIFY) — add `pkg/formats/spr256` to the tier + DAG tables (stdlib only).
- Covers: FR-1, FR-2, FR-3, FR-4, FR-5; P-1, P-2, P-3, P-4; DD1–DD8, DD10; R-1…R-4; SC-11 (and the
  implementation that makes SC-1…SC-10 provable in T2).
- **Done when:** the package exposes the `Decode` API and types; `go build ./...`, `go vet ./...`,
  `gofmt -l` (tracked `*.go`), `go test ./...` (including `internal/archtest` with the new row), and
  `bash scripts/check-no-game-assets.sh` are all clean with no game install present.

## T2 — synthetic test suite  *(implementation; authored in a separate context)*

Author the synthetic unit suite from `spec.md` + the `plan.md` API contract, deriving each test from
the acceptance criterion it covers (not from the decoder internals). Fixtures are byte streams built in
test code; no test reads a game install; `.256` carries no strings so no CP866 byte is needed.

- Files: `pkg/formats/spr256/spr256_test.go` (ADD).
- Covers: AC-1…AC-8, AC-10, AC-11; P-1, P-2, P-3, P-4; SC-1…SC-10.
- **Done when:** `TestDecodeTwoFrameGrids` (SC-1), `TestPaletteBGRToRGB` (SC-2),
  `TestDecodeNoPaletteVariant` (SC-3), `TestRejectFrameBlockPastTrailer` (SC-4),
  `TestRejectRLETilingErrors` (SC-5), `TestRejectLiteralOverrunsBlock` (SC-6),
  `TestRejectShortAndOversizedStreams` (SC-7), `TestDecodeEmptyGridBetweenFrames` (SC-8),
  `TestDecodeBucketBSingleFrame` (SC-9), `TestDecodeC0AliasesC80` (SC-10) all pass; `go test ./...`
  green with no game install present.

## T3 — `cmd/sprtool` dump tool + DAG registration  *(implementation)*

Add the developer PNG-dump tool over `pkg/formats/res` + `pkg/formats/spr256`, registered in the DAG
in the same change (DD10).

- Files: `cmd/sprtool/main.go` (ADD) — `png <archive> <path> <dir>`: open the archive via
  `res.Open`, `ReadFile` the entry, `spr256.Decode`, write one `image/png` per frame under `<dir>`
  (opaque pixels mapped through the palette to RGBA, transparent pixels to alpha 0). Errors to stderr,
  non-zero exit. No test file (developer-run only, FR-6 / spec Out-of-scope).
  `internal/archtest/dag.go` (MODIFY) — add allow-map row
  `"cmd/sprtool": {"pkg/formats/res", "pkg/formats/spr256"}`. `docs/ARCHITECTURE.md` (MODIFY) — add
  the `cmd/sprtool` row. `.gitignore` (MODIFY) — add a git-ignored output directory for decoded PNGs.
- Covers: FR-6; DD9, DD10; SC-12 (tool half).
- **Done when:** `sprtool png` operates over an archive entry; `go build ./...`, `go vet ./...`,
  `gofmt -l` clean; `go test ./...` still green (no new test); `internal/archtest` green with the
  `cmd/sprtool` row; asset guard clean.

## T4 — AC-9 evidence against a lawful install  *(developer-run verification)*

Run `sprtool png` against a real GOG `cursors/default.256` from a lawful ROM1 install, if one is
available in the run environment, and record **evidence only** — that each frame wrote a PNG, the
image is upright and recognizable as the in-game cursor, and the palette matches the VGA colors — never
any game bytes or converted PNGs.

- Produces: evidence in `verification.md` (Phase 5), or, if no lawful install is available in this
  environment, an explicit "AC-9 not run — no lawful install present" limitation there.
- Covers: AC-9; SC-12 (dev-run half).
- **Done when:** `verification.md` records either real AC-9 evidence or the explicit limitation; no
  game asset or converted PNG is committed and `bash scripts/check-no-game-assets.sh` stays clean.

## Traceability

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (decode: trailer, palette, frame walk) | SC-1, SC-3, SC-8, SC-9, SC-10 | T1 (impl), T2 (proof) |
| FR-2 (palette presence from bit 31) | SC-3, SC-7 | T1 (impl), T2 (proof) |
| FR-3 (preserve indices) | SC-2 | T1 (impl), T2 (proof) |
| FR-4 (atomic rejection, never panic) | SC-4, SC-5, SC-6, SC-7 | T1 (impl), T2 (proof) |
| FR-5 (purity / DAG) | SC-11 | T1 |
| FR-6 (dump tool) | SC-12 | T3, T4 |
| AC-1 | SC-1 | T2 |
| AC-2 | SC-2 | T2 |
| AC-3 | SC-3 | T2 |
| AC-4 | SC-4 | T2 |
| AC-5 | SC-5 | T2 |
| AC-6 | SC-6 | T2 |
| AC-7 | SC-7 | T2 |
| AC-8 | SC-8 | T2 |
| AC-9 | SC-12 | T4 |
| AC-10 | SC-9 | T2 |
| AC-11 | SC-10 | T2 |
| P-1 (buffer is width×height; index or transparent) | SC-1, SC-5, SC-8 | T1, T2 |
| P-2 (no panic / no OOB on any input) | SC-4, SC-6, SC-7 | T1, T2 |
| P-3 (indices independent of palette) | SC-2, SC-3 | T1, T2 |
| P-4 (no write outside the grid) | SC-5 | T1, T2 |
| R-1 (hostile dimensions) | SC-7 | T1, T2 |
| R-2 (Bucket-B appended section) | SC-9 | T1, T2 |
| R-3 (width=0 / non-tiling) | SC-8 | T1, T2 |
| R-4 (0xC0 alias) | SC-10 | T1, T2 |
