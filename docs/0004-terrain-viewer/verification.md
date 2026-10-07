# Verification — terrain graphics decoder & full-map render (ROM1)

Work item `0004-terrain-viewer`. Research pin: submodule `16335b4` (EXP-0021).
Toolchain: the Go release pinned in `go.mod` (`go 1.26.1`).

**Real-map evidence re-gathered 2026-07-25, against the EXP-0030-corrected `.alm` framing.** Every
observation in this file that came from the owner's lawful install was taken again after the map
reader was reframed (research `da54e6d`). The pre-EXP-0030 reader began each grid layer eight bytes
too early, so the type-1 tile grid it handed this renderer was **four cells out in X** and carried the
type-1 record header's identity words (`0001 0000 2b6d bfc0`) as its first four cells, losing the
map's last four authored cells off the end. Where a recorded number changed, the old value and the
reason are kept beside the new one. The synthetic suite is unaffected: its fixtures are byte streams
built in test code, and they were re-expressed under the corrected framing with **no asserted value
edited**.

## Gates

All run from the repository root with **no game install present**.

| Gate | Result |
|---|---|
| `go build ./...` | clean (no output) |
| `go vet ./...` | clean (no output) |
| `go test ./...` | ok — every package green |
| `go test ./internal/archtest/` | ok — DAG + determinism-wall checks pass |
| `gofmt -l $(git ls-files '*.go')` | no output |
| `scripts/check-no-game-assets.sh` | `check-no-game-assets: clean (tree scan)` |
| `scripts/check-no-game-assets.sh --history` | `check-no-game-assets: clean (history scan)` |

```
$ go test ./...
?       againrom/cmd/againrom   [no test files]
?       againrom/cmd/almtool    [no test files]
?       againrom/cmd/regtool    [no test files]
?       againrom/cmd/restool    [no test files]
?       againrom/cmd/sprtool    [no test files]
ok      againrom/cmd/terraintool        0.389s
ok      againrom/internal/archtest      0.388s
ok      againrom/internal/notices       0.011s
?       againrom/pkg/data       [no test files]
ok      againrom/pkg/formats/alm        0.014s
?       againrom/pkg/formats/reg        [no test files]
ok      againrom/pkg/formats/res        0.019s
ok      againrom/pkg/formats/spr256     0.021s
ok      againrom/pkg/game       0.015s
?       againrom/pkg/mapload    [no test files]
?       againrom/pkg/render     [no test files]
ok      againrom/pkg/render/terrain     0.337s
?       againrom/pkg/sim        [no test files]
?       againrom/pkg/ui [no test files]
?       againrom/pkg/vfs        [no test files]
```

```
$ go test ./pkg/render/terrain/ -count=1 -v
--- PASS: TestDecodeBMP8Pixels
--- PASS: TestDecodeBMP8RejectsOutsideSubset
--- PASS: TestDecodeBMP8RejectsMalformedHeader
--- PASS: TestDecodeBMP8NeverPanics
--- PASS: TestSliceStripSubCellCounts        (32x448 / 32x256 / 32x128)
--- PASS: TestSliceStripRejectsBadGeometry
--- PASS: TestTilePathAndSlotIndex
--- PASS: TestLoadTilesetPopulatesShippedSlots
--- PASS: TestLoadTilesetRecordsAbsent
--- PASS: TestResolveMapping
--- PASS: TestResolveIgnoresNonMappingBits
--- PASS: TestResolveIsTotalAndPure
--- PASS: TestResolveCorpusDomainHitsShippedFiles
--- PASS: TestDirtSubCell
--- PASS: TestCompositePlacesSubCells
--- PASS: TestCompositeScales
--- PASS: TestCompositeWaterIgnoresImpassable
--- PASS: TestCompositeWithoutDirt
--- PASS: TestCompositeRejectsBadArguments
--- PASS: TestCompositeNeverPanics
ok      againrom/pkg/render/terrain     0.337s

$ go test ./cmd/terraintool/ -count=1 -v
--- PASS: TestRenderEndToEnd
--- PASS: TestRenderScaleAndEnvAssetRoot
--- PASS: TestRenderFailures
--- PASS: TestRenderWithEmptyArchive
ok      againrom/cmd/terraintool        0.389s
```

## Acceptance criteria

| AC | Level | Covered by | Status |
|---|---|---|---|
| AC-1 | unit | `TestDecodeBMP8Pixels` — a 3x2 synthetic BMP (padded stride) decodes to its palette colours (B,G,R → R,G,B, opaque) with the stored bottom-up rows in top-down order; the top-left pixel discriminates against a decoder that skips the flip | ✔ |
| AC-2 | unit | `TestDecodeBMP8RejectsOutsideSubset` — bpp 24, `BI_RLE8`, a truncated palette, truncated pixel data, a pixel offset past EOF, a pixel offset inside the palette, and `clrUsed > 256` each reject with a nil image | ✔ |
| AC-3 | unit | `TestDecodeBMP8RejectsMalformedHeader` — empty, 2-byte, 53-byte, bad magic, DIB size 12, height 0, negative height, negative width and overflowing dimensions each reject with no image and no panic | ✔ |
| AC-4 | unit | `TestSliceStripSubCellCounts` — synthetic 32x448 / 32x256 / 32x128 strips slice into 14 / 8 / 4 sub-cells, each bounded 32x32, and sub-cell `k` carries the fill painted into the `k`-th top-down block (the ordering the game's `8 + k*0x400` offset walks) | ✔ |
| AC-5 | unit | `TestResolveMapping` (all `g` 0..12 × all `b` × all `sub`: slot `= g*4+b`, verified equal to `SlotIndex(G=(g>>2)+1, V=(g&3)*4+b)`, `G ∈ 1..4`, sub `= w&0xf`, water forced to `g=8`, road `g=12` → tile4-00..03) and `TestResolveIgnoresNonMappingBits` (bit 13 and bits 14/15 do not change the result). Land/water sub-cell ranges are checked in `TestResolveCorpusDomainHitsShippedFiles` | ✔ |
| AC-6 | unit | `TestCompositePlacesSubCells` — each cell shows its resolved sub-cell at `(col*32, row*32)`; the impassable non-water cell is dirt-composited with sub-cell `(0+1*5)&3 = 1` and is shown to differ from all three other dirt cells (so the selector is genuinely observable); the absent-slot cell is a placeholder and is counted. `TestCompositeScales` covers the scale factor. `TestRenderEndToEnd` repeats the whole scenario through a **synthetic `res` archive** and a synthetic `.alm` | ✔ |
| AC-7 | unit | `TestLoadTilesetRecordsAbsent` (empty archive, junk bytes, truncated BMP, odd geometry, nil source, out-of-range slot/sub-cell lookups) and `TestCompositeNeverPanics` (empty tileset, off-range words including the maximum group 127 → slot 511, a sub-cell past a present strip's height). `TestRenderWithEmptyArchive` covers it end-to-end | ✔ |
| AC-8 | **manual** | **Measured half recorded** — the ten shipped root maps render at the corrected framing with 52/128 tile slots and **0 placeholder fallbacks**, the figure `TERR-VER-005` predicts; the "composite matches the game's terrain" half needs a human eye and stays pending — see below | ◑ |

## Derived properties

| P | Covered by | Status |
|---|---|---|
| P-1 | `TestDecodeBMP8NeverPanics` — a full truncation sweep over a valid BMP, a single-byte mutation sweep over all 54 header bytes × 5 values, and structured garbage; every call either errors with no image or returns a self-consistent one, and none panics | ✔ |
| P-2 | `TestDecodeBMP8Pixels` (`len(Pix) == w*h*4`) and `TestSliceStripSubCellCounts` (`height/32` sub-cells exactly) | ✔ |
| P-3 | `TestResolveIsTotalAndPure` (deterministic and total over all 65 536 words, sub-cell always `w&0xf`) and `TestResolveCorpusDomainHitsShippedFiles` (over `g` 0..12 every resolved slot is a file a shipped install carries and every sub-cell is in range — with the discriminating negative that a land-range sub-cell 13 does **not** resolve inside an 8-cell water strip) | ✔ |
| P-4 | `TestCompositeNeverPanics`, `TestCompositeRejectsBadArguments`, plus the nil-safe bounds-checked `Tileset.Slot` / `Strip.SubCell` accessors exercised in `TestLoadTilesetRecordsAbsent` | ✔ |

## AC-8 — real-map evidence (measured half recorded; visual confirmation pending)

AC-8 requires a lawful GOG install and is **not automated**: no test in this repository reads a
game file, and no game bytes are committed. It is run by a developer as:

```
go run ./cmd/terraintool render -assets <install-dir> -map <map.alm> -out terraintool-out/map.png -unshaded
```

(or with `AGAINROM_ASSETS` set instead of `-assets`; `-scale N` enlarges the output). `-unshaded` is
what selects **this story's** image: since 0007 the tool relief-shades by default, and the flag
reproduces the flat full-brightness composite AC-8 is about.

Run on 2026-07-25 over the ten root maps of the owner's install
(`C:\Program Files (x86)\GOG Galaxy\Games\Rage of Mages`), each line as the tool printed it:

| Map | Cells | Output px | Tile slots | Placeholder cells |
|---|---|---|---|---|
| `Beast.ALM` | 256×256 (65536) | 8192×8192 | 52/128 | **0** |
| `Cross.ALM` | 256×256 (65536) | 8192×8192 | 52/128 | **0** |
| `Forester.alm` | 256×256 (65536) | 8192×8192 | 52/128 | **0** |
| `Horror.alm` | 256×256 (65536) | 8192×8192 | 52/128 | **0** |
| `Islands.alm` | 256×256 (65536) | 8192×8192 | 52/128 | **0** |
| `Kids.alm` | 80×80 (6400) | 2560×2560 | 52/128 | **0** |
| `Kids2.ALM` | 80×80 (6400) | 2560×2560 | 52/128 | **0** |
| `LuMoir.alm` | 144×144 (20736) | 4608×4608 | 52/128 | **0** |
| `Tomb.ALM` | 256×256 (65536) | 8192×8192 | 52/128 | **0** |
| `Waters.alm` | 144×144 (20736) | 4608×4608 | 52/128 | **0** |

The loaded tile-slot count is **52/128** on every map (tile1/2/3 × 16 plus tile4-00..03, as
`TERR-LOAD-002` describes), and the **placeholder-fallback count is 0** on every map — the figure
`TERR-VER-005` predicts, now observed rather than assumed. FR-5's placeholder path is therefore
*unexercised* on the shipped corpus; it is covered by the synthetic AC-6/AC-7 cases only.

**This number changed, and the change is the point.** Under the pre-EXP-0030 framing the same maps
reported **2** placeholder cells apiece. Those two were never real terrain: they were two of the four
record-header identity words the old reader mistook for the grid's first cells. `2b6d` resolves to
strip group 45 → slot 182 and `bfc0` to group 127 → slot 508, both outside the 128-slot array, so the
compositor filled them with `PlaceholderColor` and counted them — exactly the "recorded condition,
never a crash" behaviour FR-3/FR-5 require, applied to bytes that were never tile words. (The other
two identity words, `0001` and `0000`, resolve to slot 0, a file that does ship, so they drew as
ordinary `tile1-00` cells and left no trace in the count.) With the framing corrected, no cell of any
shipped map names an absent slot.

Only these observations are committed; the rendered PNG is a converted game asset and is git-ignored
(`terraintool-out/`).

**Still pending, and not claimed:** a developer's confirmation, against the running game, that the
composite *matches* the game's terrain. That is an eye-level fidelity comparison; nothing above
substitutes for it, and no result may be recorded here that was not observed on a real install.

## Notes and judgment calls

- **The impassable dirt composite operation is this project's engineering, not a decoded fact.**
  The research pins *which* dirt sub-cell an impassable non-water cell uses (`(col + row*5) & 3`,
  `TERR-SEM-004`) but not the pixel operation the game's blit performs — it records per-display-mode
  blit behaviour and 8-bpp palette quantization as runtime detail outside its scope — and the spec
  correspondingly assigns "the compositor, scaling, placeholder fill, and image output" to the
  project's own engineering. Because FR-1 decodes BMPs fully **opaque**, a plain source-over would
  discard the dirt entirely and leave the FR-5 requirement invisible and unverifiable, so
  `blendPixel` averages the two sub-cells per channel. It is documented as own engineering in
  `pkg/render/terrain/composite.go` and is a single function to replace if the real blend is decoded.
- **Water is static (phase 0).** The research transcribes the phase expression from render
  immediates but flags the *timing* as not re-derived; the spec scopes cycling to story 0006. The
  group override to `8` is the research's own animation-disabled path, so nothing is invented.
- **Top-down BMPs (negative height) are rejected.** The spec's format table documents only positive
  height (bottom-up), which is what every terrain tile uses; accepting a negative height would mean
  guessing at a case outside the documented subset.
- **`Resolve` takes the tile word alone.** The spec frames the mapping as `f(w, col, row)`; at
  phase 0 the slot and sub-cell do not depend on the position, so `(col, row)` enters at the
  compositor, where it actually matters (the dirt selector, and the future water phase). Each
  function therefore declares only what it really depends on, and P-3 is tested at both levels.
- **`pkg/vfs` is still a doc-only stub**, so the harness reads `graphics.res` through
  `pkg/formats/res` directly — permitted for the cmd tier. When `pkg/vfs` lands, only
  `cmd/terraintool` changes: `terrain.LoadTileset` takes a one-method `EntrySource`, which both an
  archive and a VFS satisfy.
- **No research gaps were hit.** Every fact FR-1…FR-6 needs is present at High confidence in
  EXP-0021; nothing was reverse-engineered, guessed, or filled in locally.

## Commits

| Commit | Subject | `SDD-Task` |
|---|---|---|
| `3bc7f89` | docs(0004-terrain-viewer): plan and task breakdown | — (workflow) |
| `cd1ed31` | chore(arch): register pkg/render/terrain and cmd/terraintool in the DAG | — (workflow) |
| `e0e43bd` | feat(render/terrain): 8-bpp Windows BMP decoder with atomic subset validation | `0004-terrain-viewer/T1` |
| `1b58bd9` | feat(render/terrain): tile strip slicer and 128-slot tileset loader | `0004-terrain-viewer/T2` |
| `28ae597` | feat(render/terrain): tile-word to graphic mapping (TERR-IDX-003) | `0004-terrain-viewer/T3` |
| `a4100dc` | feat(render/terrain): full-map terrain compositor | `0004-terrain-viewer/T4` |
| `58028aa` | feat(terraintool): headless terrain render harness | `0004-terrain-viewer/T5` |

One `SDD-Task` id per implementation commit, none reused; workflow commits carry none; no
`Co-Authored-By` trailer anywhere.

## T7 — palette indices retained; dirt composite corrected

**Defect.** T4's impassable dirt composite averaged the tile and dirt pixels per channel. That was a
documented stand-in, chosen while the pixel operation was undecoded — the note in `blendPixel` said so and
said it was "a single function to replace if the real blend is ever decoded". Research EXP-0024 decoded
it, and the average is wrong.

`TERR-DIRT-017`: the renderer copies the terrain sub-cell, then overlays a `dirt.bmp` sub-cell byte by
byte, where a **non-zero** dirt palette index replaces the terrain pixel and a **zero** index (the 8-bpp
transparent key, `SPR256-PAL-013`) leaves the terrain showing. There is no arithmetic. Averaging both
lightened every opaque dirt pixel by mixing in terrain the game replaces outright, *and* darkened every
transparent one by mixing in dirt that should not have been drawn at all.

**Prerequisite.** Keying on the index needs the index, which `DecodeBMP8` was discarding by resolving
straight to RGBA. The decoder now returns `*image.Paletted` and strips carry paletted sub-cells. The same
value is what the terrain blitters' shading table is addressed by (`TERR-LIGHT-011`,
`LUT[(level, palette index)]`), so this is also the prerequisite 0007 needs — done once here rather than
twice.

| AC | Evidence | |
|---|---|---|
| AC-9 | `TestCompositeDirtIsTransparentKeyed` — an impassable cell over a dirt sub-cell that is index 0 across its left half and a non-zero index across its right: every left-half pixel is the terrain colour **byte-identical**, every right-half pixel is the dirt colour **outright**, and the test asserts the two colours differ so neither half can pass vacuously. A blend would land between them on both halves | ✔ |
| AC-6 | `TestCompositeSelectsAndPlaces` updated — the impassable cell now equals its dirt sub-cell outright, and the fixture check that the four dirt sub-cells stay distinguishable is retained, so the `(col + row*5) & 3` selector is still observable | ✔ |

The old expectation was load-bearing, not cosmetic: `cmd/terraintool`'s `TestRenderEndToEnd` failed on the
first build after the change (`impassable cell = {203 52 144 255}, want {102 152 80 255}`) because it
asserted the average. Both it and the package test were updated to the decoded behaviour.

**Real-install check (re-run 2026-07-25 at the corrected framing).** `terraintool render -unshaded` over
`Cross.ALM` completes: `256x256 cells (65536), 8192x8192 px at scale 1, tile slots 52/128, placeholder
cells 0`. The placeholder count reads **0, where the pre-EXP-0030 run recorded 2** — those two cells were
never dirt-related and never terrain; they were record-header identity words resolving to absent slots
182 and 508, as set out under AC-8 above.

The dirt composite itself is untouched by the framing correction (it is a per-cell operation; the
correction moved *which* cell is which, not what the compositor does to one), and the earlier eyeball
note — "crisply over the terrain rather than a washed-out mixture", made on the pre-correction
render — is now replaced by a measurement that does not depend on either the framing or an eye. Over
every impassable non-water cell of a real map, each composited pixel was compared against the two
colours it is allowed to be:

| Map | Impassable non-water cells | Pixels left as terrain (dirt index 0) | Pixels replaced by dirt | Pixels that are **neither colour outright** |
|---|---|---|---|---|
| `Cross.ALM` | 646 | 593 907 | 67 597 | **0** |
| `Islands.alm` | 291 | 267 607 | 30 377 | **0** |

Every one of the 661 504 (`Cross`) and 297 984 (`Islands`) composited pixels is *either* the terrain
sub-cell's colour where the dirt index is 0 *or* the dirt sub-cell's colour outright where it is not.
The pre-EXP-0024 average would have produced a value between the two on **every** pixel of both
columns; it produces one on none. This is AC-9's transparent-key contract, witnessed on authored map
data instead of a fixture.

All gates green with no game install present (build, vet, gofmt, test, archtest, notices, asset guard on
tree and history).
