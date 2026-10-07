# Tasks — terrain graphics decoder & full-map render (ROM1)

Ordered.

## T0 — DAG registration (chore, no trailer)

Register `pkg/render/terrain` (stdlib-only) and `cmd/terraintool` in the `internal/archtest`
allow-map and the `docs/ARCHITECTURE.md` tables. Lands before the code so the fail-closed import
check is green at every later commit.

## T1 — 8-bpp BMP decoder (FR-1, FR-2)

`pkg/render/terrain/{doc.go,bmp.go,bmp_test.go}`. `DecodeBMP8` with atomic subset validation and
bottom-up→top-down palette expansion.
Covers AC-1, AC-2, AC-3, P-1, P-2 (pixel count).

## T2 — strip slicer & tileset loader (FR-3)

`pkg/render/terrain/{tileset.go,tileset_test.go}`. `SliceStrip` (32×32 sub-cells, 14/8/4 by height)
and `LoadTileset` over the 128-slot `tiles[(G-1)*16+V]` model + `dirt.bmp`, reading through the
`EntrySource` interface; absent/undecodable entries recorded, never fatal.
Covers AC-4, AC-7 (loader half), P-2 (sub-cell count).

## T3 — tile-word → graphic mapping (FR-4)

`pkg/render/terrain/{mapping.go,mapping_test.go}`. `Resolve` implementing `TERR-IDX-003` exactly,
with the static water override to `g = 8`.
Covers AC-5, P-3.

## T4 — full-map compositor (FR-5)

`pkg/render/terrain/{composite.go,composite_test.go}`. `Composite` placing each resolved sub-cell at
`(col*32, row*32)·scale`, the impassable non-water dirt composite `(col+row*5)&3`, and reported
placeholder fills for absent/short slots.
Covers AC-6 (unit half), AC-7 (compositor half), P-4.

## T5 — headless render harness (FR-6)

`cmd/terraintool/{main.go,main_test.go}`. Asset root from `-assets`/`AGAINROM_ASSETS`, opens
`graphics.res` + a `.alm`, writes a PNG, prints the one-line summary, non-zero exit on load failure.
The test drives the whole archive→map→image path over a synthetic `&YA1` archive and a synthetic
`.alm`.
Covers FR-6 and AC-6's "synthetic `res` archive" wiring end-to-end.

## T6 — verification (workflow, no trailer)

`verification.md`: gate commands + green output, AC-1…AC-7 coverage, and AC-8 recorded as **pending
manual verification** on a lawful install (not automated here; no game bytes committed).

## T7 — retain palette indices; correct the dirt composite  *(implementation)*

Defect fix, enabled by research EXP-0024 (`TERR-DIRT-017`). T1 resolved BMP pixels straight to RGBA and
T4 composited the impassable dirt overlay by averaging the tile and dirt pixels per channel — a stand-in
chosen (and documented as such) while the real pixel operation was undecoded. It is now decoded, and the
average is wrong: the overlay is a **transparent-keyed replace**, where a non-zero dirt palette index
replaces the terrain pixel and index 0 leaves it showing, with no arithmetic. Averaging lightened every
dirt pixel by mixing in terrain the game replaces outright.

Keying on the index requires the index, which the decoder was discarding. The same value is what the
terrain blitters' shading table is addressed by (`TERR-LIGHT-011`), so retaining it is also the
prerequisite 0007 needs.

- Files: `pkg/render/terrain/bmp.go` (EDIT) — `DecodeBMP8` returns `*image.Paletted`.
  `pkg/render/terrain/tileset.go` (EDIT) — `Strip.SubCells`/`SubCell`/`SliceStrip` carry paletted
  sub-cells, slicing the index plane and sharing the strip palette.
  `pkg/render/terrain/composite.go` (EDIT) — replace `blendPixel` with `overlayPixel` (transparent key on
  `DirtTransparentIndex`) plus `paletteColor`; `drawCell` takes paletted sources.
  `pkg/render/terrain/{bmp,tileset,composite}_test.go`, `cmd/terraintool/main_test.go` (EDIT) — assert
  the decoded behaviour instead of the average, and add the transparent-key test.
- Covers: FR-1, FR-5; AC-9.
- **Done when:** `TestCompositeDirtIsTransparentKeyed` passes; `go build ./...`, `go vet ./...`,
  `gofmt -l` (tracked `*.go`), `go test ./...`, `internal/archtest` and
  `bash scripts/check-no-game-assets.sh` are clean with no game install present.
