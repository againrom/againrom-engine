# Plan — terrain graphics decoder & full-map render (ROM1)

Implementation approach for `spec.md` (frozen). Research pin: submodule `16335b4` (EXP-0021);
facts consumed are `TERR-LOC-001`…`TERR-VER-005` plus `ALM-GRID-012` (the tile word, story 0003).

## Package layout & DAG placement

Two new packages, both registered in `internal/archtest` (fail-closed) and documented in
`docs/ARCHITECTURE.md`.

| New package | Tier | May import | Why here |
|---|---|---|---|
| `pkg/render/terrain` | render | **stdlib only** | The four pure pieces (BMP decode, strip slice, tile-word mapping, compositor). The spec's Constraints require these to take plain bytes / decoded images / integers and import **no** `pkg/formats` package. It imports neither `pkg/sim` nor `pkg/vfs` — a strict subset of what the render tier is allowed. |
| `cmd/terraintool` | cmd | `pkg/render/terrain`, `pkg/formats/res`, `pkg/formats/alm`, `pkg/game` | The FR-6 harness: the only place archive + map wiring happens. It cannot live in `cmd/almtool` (pinned to `pkg/formats/alm`). `pkg/vfs` is still a doc-only stub, so the archive is read through `pkg/formats/res` directly (cmd → any format package is allowed). |

`pkg/render` itself stays a doc-only stub; 0004's code is a focused subpackage so the top render
package is not pre-committed to terrain specifics.

**How the pure tier stays free of `pkg/formats`:** the tileset loader takes a one-method interface

```go
type EntrySource interface { ReadFile(name string) ([]byte, error) }
```

which `*res.Archive` already satisfies. The archive type never crosses the package boundary.

## Files and types

### `pkg/render/terrain/bmp.go` — FR-1 / FR-2

```go
func DecodeBMP8(data []byte) (*image.RGBA, error)
```

Atomic subset validation before any pixel work: length ≥ 54, magic `BM`, DIB size 40, bpp 8,
compression 0 (`BI_RGB`), width > 0, height > 0 (positive ⇒ bottom-up, the only orientation the
spec's format table documents), palette entry count from `biClrUsed` (0 ⇒ 256, ≤ 256), palette range
`[54, 54+4·clrUsed)` inside `bfOffBits`, pixel range `[bfOffBits, bfOffBits + stride·height)` inside
the buffer, `stride = (width+3) &^ 3`. Every palette index must be `< clrUsed`. Decode writes into a
local image returned only on success — on any error the result is `(nil, err)`, never partial (FR-2,
P-1). Palette bytes are `B,G,R,X`; `X` is ignored and alpha is set opaque. Display row `y` reads BMP
row `height-1-y` (bottom-up ⇒ top-down).

### `pkg/render/terrain/tileset.go` — FR-3

```go
const CellSize = 32
const SlotCount = 128

type Strip struct{ SubCells []*image.RGBA } // each 32x32, top-down, index k = rows [k*32,(k+1)*32)
type Tileset struct {
    Slots   [SlotCount]*Strip // tiles[(G-1)*16+V]; nil = absent
    Dirt    *Strip            // terrain.3d/dirt.bmp
    Loaded  int
    Missing []string          // recorded absent/undecodable entries
}

func SliceStrip(img *image.RGBA) (*Strip, error)
func LoadTileset(src EntrySource) *Tileset
```

`LoadTileset` walks the 128-slot model exactly as `TERR-LOAD-002` states it — `G = 1..8`,
`V = 0..15`, path `terrain.3d/tileG-VV.bmp` (`%02d`, matching the game's `%d-%d` / `%d-0%d` pair) —
plus `terrain.3d/dirt.bmp`. Any failure (entry absent, non-BMP bytes, short/odd geometry) leaves the
slot nil and appends to `Missing`; nothing about a slot can fail the load (FR-3, AC-7). Groups 5–8
and `tile4` `V ≥ 4` are absent on a real install by construction, which is the claim's own
"stay null". `SliceStrip` requires width 32 and height a positive multiple of 32, yielding
`height/32` sub-cells (14 / 8 / 4 for 448 / 256 / 128) — AC-4, P-2.

### `pkg/render/terrain/mapping.go` — FR-4

```go
type TileRef struct { Slot, Sub int; Water bool }
func Resolve(word uint16) TileRef
```

Exactly `TERR-IDX-003`: `g=(w&0x1fff)>>6`, `b=(w>>4)&3`, `sub=w&0xf`, `Slot=g*4+b`, `Sub=sub`; for
`g ∈ 8..11` `Water` is set and `g` is forced to `8` (phase 0, static — 0006 owns the phase). No
other bit split is read; bit 13 is deliberately *not* consulted here (it is a compositor input).
`Resolve` is total and allocation-free: it is defined for all 65 536 words and never rejects, so an
off-corpus word simply resolves to a slot the tileset may not have (→ placeholder, P-4). It is a
pure function of the word alone; the `(col,row)` half of the spec's `f(w,col,row)` domain enters at
the compositor (dirt selector, and the future water phase), which keeps each function honest about
what it actually depends on.

### `pkg/render/terrain/composite.go` — FR-5

```go
type Grid struct { Width, Height int; Tiles []uint16 }
type Render struct { Image *image.RGBA; Placeholders int }
func Composite(ts *Tileset, g Grid, scale int) (*Render, error)
```

Validates `ts != nil`, `Width/Height > 0`, `len(Tiles) == W*H`, `scale >= 1`, and that the output
pixel count does not overflow — then allocates one `W*32*scale × H*32*scale` RGBA. Per cell
`(col,row)`: `ref := Resolve(word)`; fetch sub-cell `ref.Sub` of `ts.Slots[ref.Slot]`; a nil slot, an
out-of-range slot index, or a slot too short for `ref.Sub` ⇒ solid placeholder fill and
`Placeholders++`. Impassable (`word & 0x2000`) **non-water** cells composite the resolved sub-cell
with dirt sub-cell `(col+row*5)&3`; if the dirt slot is absent/short the tile is drawn alone (no
placeholder — the tile itself is present). Scaling is nearest-neighbour pixel replication. Every
index is bounds-checked before use (P-4).

**Engineering decision — the composite operation.** The research decodes *which* dirt sub-cell is
used but not the pixel operation the DirectDraw blit performs (it lists per-display-mode blit detail
as out of scope), and the spec assigns "the compositor, scaling, placeholder fill, and image output"
to the project's own engineering. FR-1 decodes BMPs fully **opaque**, so a plain source-over of an
opaque tile would make dirt invisible and the FR-5 requirement unobservable. The compositor
therefore blends the two sub-cells per channel (50/50 average) so the blocked-ground marker is
visible and the selected dirt sub-cell is verifiable. This is labelled in code as own engineering,
**not** a decoded game fact; substituting the real blend when research decodes it is a one-function
change. The placeholder fill (opaque magenta) is likewise own engineering.

### `cmd/terraintool/main.go` — FR-6

```
terraintool render -assets <dir> [-map <file.alm>] [-scale N] -out <file.png>
```

Asset root via `game.ResolveAssetRoot(-assets, AGAINROM_ASSETS)`; no install path in source. Opens
`<root>/graphics.res` with `res.Open`, builds the tileset from it, decodes the `.alm` with
`alm.Open`, composites `alm.Map.Tiles`, writes a PNG, prints one summary line (`W×H`, cell count,
placeholder count) and exits non-zero on any load failure. `run(args, stdout) error` is factored out
so the end-to-end test can drive it.

## FR / AC / P → code + test map

| Item | Code | Test |
|---|---|---|
| FR-1 | `DecodeBMP8` | `bmp_test.go` AC-1 |
| FR-2 | `DecodeBMP8` validation | `bmp_test.go` AC-2, AC-3 |
| FR-3 | `LoadTileset`, `SliceStrip` | `tileset_test.go` AC-4, AC-7 |
| FR-4 | `Resolve` | `mapping_test.go` AC-5 |
| FR-5 | `Composite` | `composite_test.go` AC-6, AC-7 |
| FR-6 | `cmd/terraintool` | `cmd/terraintool/main_test.go` (synthetic `.res` + `.alm`, end-to-end) |
| AC-1 | palette BGR→RGB, top-down rows | `TestDecodeBMP8Pixels` |
| AC-2 | bpp≠8 / compression≠0 / truncated palette / truncated pixels | `TestDecodeBMP8RejectsOutsideSubset` |
| AC-3 | short & malformed headers | `TestDecodeBMP8RejectsMalformedHeader` |
| AC-4 | 32×448 / 32×256 / 32×128 ⇒ 14 / 8 / 4 | `TestSliceStripSubCellCounts` |
| AC-5 | `g=0..12`, all `b`, all `sub`, water, bit-13 | `TestResolveMapping`, `TestResolveFilenameIdentity` |
| AC-6 | composite placement, dirt, placeholder | `TestCompositePlacesSubCells`, `TestCompositeEndToEndFromArchive` (cmd) |
| AC-7 | non-BMP/short strips, empty archive, off-range word | `TestLoadTilesetRecordsAbsent`, `TestCompositeOffRangeWord` |
| AC-8 | — | **manual**, recorded in `verification.md`; not automated, no game bytes |
| P-1 | never panics / OOB on any bytes | `TestDecodeBMP8NeverPanics` (fuzz-style corpus incl. truncation sweep) |
| P-2 | `w×h` pixels; `height/32` sub-cells | `TestDecodeBMP8PixelCount`, `TestSliceStripSubCellCounts` |
| P-3 | mapping purity + corpus-valid ranges | `TestResolveIsPure`, `TestResolveCorpusDomain` |
| P-4 | compositing never indexes out of range | `TestCompositeNeverPanics` |

## Test strategy

100 % synthetic, green with no game install. Fixtures are built in test code: `buildBMP8` emits a
byte-exact 8-bpp BMP (14-B file header + 40-B DIB + 256×4 palette + padded bottom-up rows) from a
palette and an index grid; `mapSource` is an in-memory `EntrySource`; the cmd test additionally
builds a real synthetic `&YA1` archive (24-B header + payloads + 32-B node registry, per 0001) and a
synthetic `.alm` (per 0003) so the archive→map→image wiring is exercised for real. No non-ASCII
literal is authored anywhere; archive node names are ASCII.

## Task order

One implementation task = one commit (trailer `SDD-Task: 0004-terrain-viewer/T<n>`). The DAG
registration (`internal/archtest` + `docs/ARCHITECTURE.md`) lands first as a workflow/chore commit so
every implementation commit is green on `go test ./internal/archtest/`. See `tasks.md`.
