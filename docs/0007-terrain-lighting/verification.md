# Verification — terrain relief lighting

Work item `0007-terrain-lighting`. Research pin: submodule `8a406d6` (EXP-0023 + EXP-0025 + EXP-0029 +
**EXP-0031**). Toolchain: the Go release pinned in `go.mod` (`go 1.26.1`), windows/amd64.

**This records the RED revision.** EXP-0031 (`TERR-LIGHT-028/029/030`) superseded the gradient the first
pass shipped — a two-cell central difference on two perpendicular axes, with no lateral shear — and the
corrected arithmetic changes every shaded output value. **Every derived figure the previous revision
recorded is therefore treated as suspect, not only the ones attached to a corrected claim** (FR-7): each
is re-measured below, or explicitly marked superseded, or restated as unrepeatable. Real-map evidence
re-run 2026-07-25 on the owner's install, at the EXP-0030-corrected `.alm` framing.

**Every level figure below carries its parameters.** Unless stated otherwise: **θ = 0.78539815**
(`DefaultDaytime`, the engine's cycle-off literal), `ambient = 0x0e`, `range = 0x20` (so `L=62, R=32`),
the EXP-0030-corrected `.alm` grid framing, and the **whole `W×H` grid** — with the strict-interior
window `{1..W-2}×{1..H-2}` reported separately where it differs, because that is the window the engine
writes and the one the research census uses. θ is **not** a constant of the format; whether shipped play
runs at this angle is an open research question, and the cycle-off default is used here as a labelled
reference point only.

## Gates

All run from the repository root with **no game install present**.

| Gate | Result |
|---|---|
| `go build ./...` | clean (no output) |
| `go vet ./...` | clean (no output) |
| `go test ./...` | ok — every package green |
| `go test ./internal/archtest/` | ok — DAG + determinism-wall checks pass (no new package; `pkg/render/terrain` stays stdlib-only) |
| `gofmt -l $(git ls-files '*.go')` | no output |
| `scripts/check-no-game-assets.sh` | `check-no-game-assets: clean (tree scan)` |

The gate was run green before each of this revision's implementation commits (T6, T7) as well as at
HEAD. The first pass's commits (T1…T4) were likewise each verified in an isolated worktree.

```
$ go test -count=1 ./pkg/render/terrain/ -run 'Level|Shade|Interp|Light|CompositeLit' -v
--- PASS: TestLevelGridFlatIsFortySix
--- PASS: TestLevelGridFlatIntensity
--- PASS: TestLevelGridRidge
--- PASS: TestLevelGridRowAxisGradient
--- PASS: TestLevelGridLateralShear
--- PASS: TestLevelGridSignedHeights
--- PASS: TestLevelGridClampsAxes
--- PASS: TestLevelGridBordersAndDegenerate
--- PASS: TestLevelGridPure
--- PASS: TestLightFromFields
--- PASS: TestInterpRow
--- PASS: TestShadeChannelTransform
--- PASS: TestShadeIdentityRow
--- PASS: TestShadeTintBeforeMultiply
--- PASS: TestShadeMonotonic
--- PASS: TestShadeCorpusRange
--- PASS: TestCompositeLit
ok      againrom/pkg/render/terrain
```

**Test authorship, stated plainly.** The first pass's `light_test.go`, `shade_test.go` and `lit_test.go`
were authored by separate contexts that had read `spec.md` but not the implementation. **The level-grid
tests in this revision were not**: the owner set a lighter calibration for this pass (one good test per
behaviour actually changed plus its boundary, no separate-context author, no adversarial read), so the
implementer wrote them. `shade_test.go`, `lit_test.go` and the `cmd/terraintool` tests are untouched and
retain their separate-context provenance. The oracles are still derived from the spec: each new test
hand-derives its two `Δh` values from the spec's `Δh` definition for the fixture it builds and runs them
through a `specLevel` helper transcribing the spec's back half, so the discriminating part (which
neighbours the deltas come from) is not taken from the code.

**Because the author was not independent, the tests were mutation-checked instead** — the substitute is
recorded, not implied:

| Mutation applied to `light.go` | Result |
|---|---|
| lateral shear removed (`tanT := 0.0`) | `TestLevelGridRowAxisGradient` and `TestLevelGridLateralShear` **FAIL** |
| height reads made unsigned (`float64(heights[…])`) | `TestLevelGridSignedHeights` **FAILS** |

Both mutations were reverted before the commit. The suite also keeps the first pass's discriminating
traps: `TestLevelGridFlatIntensity` uses odd-`range` cases that diverge from an integer `L−(R>>1)`;
`TestLevelGridClampsAxes` uses an override light to reach both `[0,95]` clamps and a valley vertex that
is only reachable if the two halves clamp independently; `TestShadeTintBeforeMultiply` builds cases a
tint-after ordering would fail.

## Acceptance criteria

| AC | Level | Covered by | Status |
|---|---|---|---|
| AC-1 | unit | `TestLevelGridFlatIsFortySix` — a flat 5×5 grid at `DefaultDaytime` levels **every** vertex (ring included) to exactly 46, at three height levels and five θ; guards the default bytes, the tint, and θ as the literal `0.78539815` by value **and** by bit pattern `0x3fe921fb4d12d84a`, asserting it differs from `math.Pi/4` | PASS |
| AC-2 | unit | `TestLevelGridFlatIntensity` — flat grid at three (ambient,range) pairs equals `ftol(L − R·sin(π/6))` in float64; the odd-range cases (0x21, 0x1f) diverge from the integer trap, proving genuine truncation | PASS |
| AC-3 | unit | `TestLevelGridRidge` — a **row-axis** ramp (20/cell) levels the up-ramp vertices to exactly 58 (strictly >46) and the down-ramp to 36 (strictly <46). Along a row-constant field the shear contributes nothing, so both halves take the same one-cell difference and there is no halving toward 46 | PASS |
| AC-4 | unit | `TestLevelGridClampsAxes` — with the override light `range=0xFF, ambient=0` each half saturates at exactly 0 and 95 (level 95 up-ramp, 0 down-ramp) and a **valley** vertex levels to 47, which is only reachable because the halves clamp independently; a hostile `0x80`/`0x7f` grid over 6 θ × 2 lights keeps every level in `[0,95]` | PASS |
| AC-5 | — | *retired with the superseded model; not reproduced* | — |
| AC-5a | unit | `TestLevelGridBordersAndDegenerate` — on a row-axis ramp the top ring (52) and bottom ring (52) differ from the interior (58) in the documented direction (the backward resp. forward half collapses when its row clamps); degenerate 1×1, 1×N, N×1 return full-length non-nil grids with no panic; flat degenerate grids are uniformly 46 | PASS |
| AC-6 | unit | `TestInterpRow` — exact at the four corners; exact interior blends and fractional-truncation cases (12.5→12, 17.5→17); constant for equal corners | PASS |
| AC-7 (unit) | unit | `TestLightFromFields` — θ passes through, ambient/range are the low byte of the stored u32s (0xFF12→0x12), sky tint stays {0,0,0} | PASS |
| AC-7 (corpus) | manual (headless) | the ten shipped maps' stored `+0x08`/`+0x10`/`+0x14` — every angle in `[−π/2,π/2]`, both scalars byte-range, and `-maplight` reading them unchanged across the framing relabel (see below) | PASS |
| AC-8 | manual | a real GOG map rendered with default daytime shading — the **measurable half is recorded** (summary lines, the exact ×1.5625 flat-ground reading, the lit-vs-unshaded distribution, the corpus level range); the **fidelity half** — a developer's eye on relief, the border seam and the outer ring — remains **pending** (see below) | ◑ |
| AC-9 | unit | `TestShadeChannelTransform` — the exact `clamp((ch·(96−level))/32,0,255)` truncating over a channel×level grid, with the spot checks level 64→unchanged, 0→min(ch·3,255), 95→ch/32 | PASS |
| AC-10 | unit | `TestShadeIdentityRow` (tint {0,0,0}, level 64 → unattenuated RGB, A forced 0xff) and `TestCompositeLit` (a uniform-level-64 render equals the unshaded `Composite` byte-for-byte) | PASS |
| AC-11 | unit | `TestShadeTintBeforeMultiply` — the tint is added before the multiply, with discriminating cases that a tint-after ordering would fail | PASS |
| AC-12 | unit | `TestShadeMonotonic` — output is monotonically non-increasing over level 0…95 and stays in `[0,255]` | PASS |
| AC-13 (table bound) | unit | `TestShadeCorpusRange` — every level 0…95 is defined, never panics, and `ShadeRGBA` forces A=0xff. That is a superset of the measured corpus range `[30..70]` (θ = 0.78539815, `L=62`, `R=32`, corrected framing, interior window) and of the analytic `[30,89]` band | PASS |
| AC-13 (corpus census) | manual (headless) | re-measured below over all 38 maps, with its parameters, and cross-checked against `TERR-LIGHT-029` | PASS |
| AC-14 | unit | `TestLevelGridRowAxisGradient` — a `y`-varying and an `x`-varying field of the same gradient (20/cell) level to **58** and **47** respectively, each equal to the exact spec value; under a perpendicular two-axis model they would be equal, so this discriminates the corrected gradient from the superseded one | PASS |
| AC-15 | unit | `TestLevelGridLateralShear` — a `0/127` checkerboard levels every interior vertex to exactly 46; the test computes the shear-free reading of the same fixture (level 56) and asserts it is not 46, so dropping or mis-signing the shear fails | PASS |
| AC-16 | unit | `TestLevelGridSignedHeights` — an asymmetric fixture straddling `0x80` levels to 43 (signed reading) and is asserted not to be 69 (the unsigned reading) | PASS |

## Derived properties

| P | Covered by | Status |
|---|---|---|
| P-1 (level in [0,95] for every input) | `TestLevelGridClampsAxes` (hostile grid straddling `0x80`, six θ, two lights) | PASS |
| P-2 (level grid is pure) | `TestLevelGridPure` (identical inputs → identical output; input not mutated) | PASS |
| P-3 (no read outside the grid) | `TestLevelGridBordersAndDegenerate` (degenerate shapes, no panic/OOB) and `TestCompositeLit` (the far-edge clamp path) | PASS |
| P-4 (flat grid yields a constant grid) | `TestLevelGridFlatIsFortySix` (every vertex 46 at three heights and five θ, ring included) and the flat degenerate shapes in `TestLevelGridBordersAndDegenerate` | PASS |
| P-5 (shaded output is pure) | `TestShadeChannelTransform` (determinism) | PASS |
| P-6 (level 64 is the identity row) | `TestShadeIdentityRow` | PASS |
| P-7 (monotone non-increasing, in [0,255]) | `TestShadeMonotonic` | PASS |

## Plan success criteria

SC-1…SC-4, SC-5a and SC-6…SC-14 map to the unit tests above; SC-5 is retired. SC-15 (FR-6:
`pkg/render/terrain` still stdlib-only, DAG green) is proven by `internal/archtest`. SC-16 (terraintool
wiring) is proven by `cmd/terraintool` tests: `TestRenderShadedDefault` (shaded by default, `-unshaded`
reproduces the 0004 image, placeholder stays unshaded), `TestRenderLightControls`
(`-ambient`/`-theta`/`-range`/`-maplight` reach the compositor and are reported), and
`TestRenderEndToEnd` / `TestRenderScaleAndEnvAssetRoot` — all unchanged by this revision, because the
tool's synthetic `.alm` fixture carries an all-zero altitude grid whose every vertex is flat and levels
to 46 under either arithmetic. SC-18, SC-19 and SC-20 map to `TestLevelGridRowAxisGradient`,
`TestLevelGridLateralShear` and `TestLevelGridSignedHeights`. SC-17 is the developer-run evidence below.

## AC-7 / AC-8 / AC-13 — developer-run evidence, re-measured

These require a lawful GOG install and are **not automated**: no test in this repository reads a
game file, and no game bytes are committed. They were re-run on 2026-07-25 on the owner's install
(`C:\Program Files (x86)\GOG Galaxy\Games\Rage of Mages`, the ten root maps plus the 28 `M7R` members of
`scenario.res`), as:

```
go run ./cmd/terraintool render -assets <install-dir> -map <map.alm> -out terraintool-out/lit.png
go run ./cmd/terraintool render -assets <install-dir> -map <map.alm> -out terraintool-out/flat.png -unshaded
```

(or with `AGAINROM_ASSETS` set; `-scale N` enlarges the output; the summary line reports the light in
effect). The corpus statistics were taken by a throwaway probe outside the module's package set, calling
the shipped `terrain.LevelGrid` / `terrain.CompositeLit` / `terrain.Composite` directly; it was deleted
before every commit and is not part of the repository. Only the observations below are committed; every
rendered PNG is a converted game asset and stays in the git-ignored `terraintool-out/`.

### What changed, figure by figure

The arithmetic moved, so this table is the audit FR-7 requires. "Unchanged" means re-measured and equal,
not carried over.

| Figure recorded by the previous revision | Status now |
|---|---|
| The ten maps' stored `+0x08`/`+0x10`/`+0x14` and their ambient/range | **unchanged** — re-measured, identical (the `.alm` reader is untouched by this revision) |
| `Islands.alm` altitude grid: row 0 `26 30 31 30 29 30 33 29 …`, range 16…127, mean 40.7 | **unchanged** — re-read, identical (mean 40.71) |
| Per-map altitude min…max and mean, ten maps | **unchanged** — re-measured, identical |
| `terraintool` summary lines (dimensions, tile slots, placeholder cells, `theta=0.7854 ambient=14 range=32`) | **unchanged** — `%.4f` of the corrected θ literal is still `0.7854` |
| `-unshaded` reproduces the 0004 flat image | **unchanged** — and now pinned harder: the `-unshaded` render of `Islands.alm` is **byte-identical** across the revision (`8ccafd00b293b315339aaba59c2e3b44` from both the pre- and post-revision binary) |
| Per-map level ranges, % at 46, mean attenuation (ten maps) | **superseded** — re-measured below |
| "Levels 30…77 over the ten maps together" | **superseded** — now `[30..65]` |
| "Our outer vertex ring spans only 35…62 across the ten maps" | **superseded** — now 35…58, and it is a different fill (clamped indices, not a one-sided difference) |
| "The strict interior alone reaches 76–77" | **superseded** — the interior now reaches 65 and coincides with the whole-grid range on every root map |
| `Islands.alm` mean channel sum 223.62 → 342.54, ratio 1.5318, 98.23 % brighter / 1.53 % darker / 0.24 % equal, 246 flat cells all exactly `ShadeRGBA(level 46)` | **superseded** — re-measured below (the unshaded 223.62 is unchanged; everything derived from the lit image moved) |
| `Kids.alm` 201.73 → 316.73, ratio 1.5701, 100 % brighter, 221 flat cells | **superseded** — re-measured below (unshaded 201.73 unchanged) |
| "Our ceiling 77 against the research census's 74" — recorded as an open R-3 consequence | **resolved, and both figures were wrong.** EXP-0031 shows 77 was our two-cell perpendicular span and 74 was the EXP-0025 probe's approximate rule on the superseded grid base. The corrected implementation now reproduces the research's own numbers exactly (below) |
| "Recomputing the levels from a reconstruction of the pre-EXP-0030 altitude grid reproduces each map's interior range exactly, map for map" | **not re-run, and not reproduced from memory.** It was a check on the superseded arithmetic against a superseded framing; it is superseded twice over and would carry no information now. Recorded as retired rather than restated |

### AC-7 (corpus) — the ten maps' stored light fields

`Angle` is the type-0 payload's f32 at `+0x08`; the two intensity scalars are `Meta.Word10` (`+0x10`)
and `Meta.Word14` (`+0x14`). Those are the **same file bytes** (`+0x30`, `+0x38`, `+0x3c`) that the
pre-EXP-0030 labels `+0x10`/`+0x18`/`+0x1C` (`Meta.Word18`/`Word1C`) named — the correction moved the
payload's origin, not the fields.

| Map | `Angle` (+0x08) | `+0x10` | `+0x14` | → ambient | → range |
|---|---|---|---|---|---|
| `Beast.ALM` | 0.7853981 | `0x15` | `0x3f` | 21 | 63 |
| `Cross.ALM` | 0.7853981 | `0x0c` | `0x3c` | 12 | 60 |
| `Forester.alm` | 0.7679449 | `0x0e` | `0x3f` | 14 | 63 |
| `Horror.alm` | 0.7853981 | `0x15` | `0x3f` | 21 | 63 |
| `Islands.alm` | 0.7853981 | `0x15` | `0x3f` | 21 | 63 |
| `Kids.alm` | 0.7853981 | `0x15` | `0x3f` | 21 | 63 |
| `Kids2.ALM` | 0.7853981 | `0x15` | `0x3f` | 21 | 63 |
| `LuMoir.alm` | 0.7853981 | `0x00` | `0x39` | 0 | 57 |
| `Tomb.ALM` | 0.6283185 | `0x10` | `0x40` | 16 | 64 |
| `Waters.alm` | −0.7853981 | `0x1a` | `0x28` | 26 | 40 |

**Every angle lies in `[−π/2, π/2]`** (seven maps store exactly π/4, `Forester` ≈ 0.768, `Tomb`
exactly π/5, and `Waters` −π/4 — the one negative angle in the ten), and **both scalars fit a byte on
every map**, so `LightFromFields`' low-byte read truncates nothing here. The tool agrees with the
table on the two maps checked end-to-end: `-maplight` reports `shaded (theta=0.7854 ambient=21 range=63)` on
`Islands.alm` and `shaded (theta=-0.7854 ambient=26 range=40)` on `Waters.alm`. AC-7's corpus half is
**closed**; it remains what the spec says it is — an optional viewer override, not an engine-fidelity
claim (`TERR-LIGHT-023`). *(Note the coincidence, which is not a finding: seven maps store a stored angle
of `0.7853981`, an f32 of π/4. That is the map author's stored value; the engine's own cycle-off θ is the
unrelated literal `0.78539815`, and the engine overwrites the stored field before reading it.)*

### AC-13 (corpus census) — and a cross-check against the research's own numbers

Parameters for the whole subsection: **θ = 0.78539815, ambient `0x0e`, range `0x20` (`L=62, R=32`),
EXP-0030-corrected framing.** Two windows are reported because they answer different questions.

| Corpus | Window | Vertices | Levels |
|---|---|---|---|
| 38 maps (10 root + 28 `scenario.res`) | strict interior `{1..W-2}×{1..H-2}` | 859 768 | **`[30..70]`** |
| 38 maps | whole `W×H` grid (interior + our ring fill) | 880 704 | `[30..70]` |
| 10 root maps | strict interior | 439 592 | **`[30..65]`** |
| 10 root maps | whole grid | 447 488 | `[30..65]` |

`TERR-LIGHT-029` publishes `[30..70]` over 38 maps at 859 768 vertices and `[30..65]` over the 10 root
maps, at the same θ and window. **Our independent implementation of `TERR-LIGHT-028` reproduces both
exactly, including the vertex count.** The θ-sweep the claim tabulates reproduces exactly as well
(38 maps, interior window):

| θ | `TERR-LIGHT-029` | measured here |
|---|---|---|
| 0.00000 (midday) | `[30..72]` | `[30..72]` |
| 0.39270 | `[30..70]` | `[30..70]` |
| 0.52360 | `[30..70]` | `[30..70]` |
| **0.78540** (the default) | `[30..70]` | `[30..70]` |
| 1.04720 | `[30..75]` | `[30..75]` |
| 1.40000 | `[30..81]` | `[30..81]` |
| −0.78540 | `[30..75]` | `[30..75]` |
| −1.04720 | `[30..80]` | `[30..80]` |

That is the strongest single result of this revision: the disagreement the previous revision recorded
(our 77 against a published 74) is gone, and it is gone by both figures being replaced rather than by
one of them winning. Every level indexes inside the 96-row table, as AC-13 requires; no vertex on real
data approaches either `[0,95]` clamp (the reachable band at these light bytes is `[30,89]`, so the
outer clamps cannot bind — P-1's clamp coverage stays the synthetic case).

**The ceiling is a function of θ, so none of these numbers may be quoted bare.** With the day/night cycle
enabled the sun model sweeps θ across `[−π/2,+π/2]`, and the corpus ceiling travels from 70 to 81 over
the angles above. Which θ shipped play runs at is not established (R-6).

### AC-8 — the default daytime render, measured

Summary lines on `Islands.alm` (256×256), lit and flat — unchanged from the previous revision, since the
light descriptor prints `%.4f`:

```
terrain: 256x256 cells (65536), 8192x8192 px at scale 1, tile slots 52/128, placeholder cells 0, shaded (theta=0.7854 ambient=14 range=32)
terrain: 256x256 cells (65536), 8192x8192 px at scale 1, tile slots 52/128, placeholder cells 0, unshaded
```

**Flat ground reads at exactly ×1.5625, not approximately.** Comparing the lit composite against the
unshaded one pixel by pixel over the same map, tileset and scale (whole grid, θ as above):

| Map | Pixels | Mean channel sum, unshaded | Lit | Ratio | Brighter | Darker | Equal | Cells with all four corner levels 46 | Of those, **exactly** `ShadeRGBA(level 46)` on every pixel |
|---|---|---|---|---|---|---|---|---|---|
| `Islands.alm` | 67 108 864 | 223.62 | 344.05 | 1.5385 | 67 087 931 (99.9688 %) | **1** (0.0000 %) | 20 932 (0.0312 %) | 475 | **475** |
| `Kids.alm` | 6 553 600 | 201.73 | 316.90 | 1.5709 | 6 553 600 (100.0000 %) | 0 | 0 | 310 | **310** |

Every fully-flat cell shades to the level-46 transform on **every** pixel — `(96−46)/32 = 1.5625`,
truncating — which is the ×1.5625 AC-8 asks about, measured rather than eyeballed.

**The darker-pixel figure moved from 1.53 % to a single pixel, and that is a consequence of the
correction, not a defect.** Level 64 is the unattenuated row, so a pixel is darker than the unshaded
image only where the level exceeds 64. The superseded two-cell span roughly doubled every gradient and
pushed `Islands` levels to 77; the corrected one-cell span tops out at 65 on that map, so exactly one
pixel (of 67 million) lands on the one level above 64 with a channel value that actually decreases. The
"steep-away faces darken" half of AC-8 is therefore **no longer visible in this comparison on `Islands`**
— the comparison is against the ×1.0 unshaded image, and the corrected arithmetic almost never reaches
×1.0 on this map. What the level table below does show is that slopes still move the level to both sides
of the flat 46; what it does not show, and what is not claimed, is a pixel-level darkening against the
game.

**Relief is present across the corpus**, from the same level grid the compositor uses (ten root maps;
whole-grid window, with the outer-ring range broken out because that ring is our own fill):

| Map | Altitudes min…max (mean) | Levels min…max | Outer ring | Vertices at the flat 46 | Mean attenuation |
|---|---|---|---|---|---|
| `Beast.ALM` | 0…127 (57.18) | 30…64 | 41…46 | 20.6 % | ×1.5513 |
| `Cross.ALM` | 0…127 (38.50) | 30…64 | 35…58 | 28.2 % | ×1.5594 |
| `Forester.alm` | 0…127 (50.95) | 30…64 | 42…49 | 20.7 % | ×1.5608 |
| `Horror.alm` | 0…127 (52.57) | 30…64 | 40…49 | 20.2 % | ×1.5627 |
| `Islands.alm` | 16…127 (40.71) | 30…65 | 42…49 | 23.0 % | ×1.5555 |
| `Kids.alm` | 51…114 (62.90) | 35…62 | 42…49 | 30.8 % | ×1.5734 |
| `Kids2.ALM` | 51…127 (63.31) | 35…62 | 42…49 | 29.8 % | ×1.5730 |
| `LuMoir.alm` | 0…127 (35.44) | 30…64 | 46…46 | 38.1 % | ×1.5609 |
| `Tomb.ALM` | 9…127 (43.73) | 30…64 | 38…49 | 17.9 % | ×1.5578 |
| `Waters.alm` | 0…127 (37.34) | 30…64 | 41…49 | 23.6 % | ×1.5673 |

Levels spread to both sides of the flat 46 on every map — `[30..65]` over the ten together — so slopes
both brighten and darken the ground rather than the render being a uniform lift. Compared with the
superseded arithmetic the distribution is **narrower and flatter**: the ten-map range contracts from
`30…77` to `30…65`, and the share of vertices sitting exactly at 46 rises on every map (e.g. `Islands`
17.3 % → 23.0 %, `LuMoir` 33.3 % → 38.1 %). That is the expected direction — a one-cell span is half the
gradient of a two-cell one — and it is what `TERR-LIGHT-028`'s "a port using a central difference
over-contrasts" predicts.

**The outer ring no longer stands out.** On nine of the ten maps our ring fill lands inside 35…49, well
within the interior's own spread, and on `LuMoir` it is uniformly 46. Under the superseded arithmetic the
ring was on a different slope scale from the interior by construction (`DD2a`/`R-4`); under the corrected
one-cell span there is no scale difference at all, and the measurement is consistent with that. The ring
remains **our** fill of a region the game leaves uncomputed — no fidelity claim attaches to it.

### Still pending — the parts that need an eye

Nothing above is a fidelity claim, and none of it can be turned into one headlessly:

- **AC-8 (fidelity):** a developer's confirmation, against the running game, that the shaded terrain
  reads as the original's does. The numbers show *our* transform applied to real heights; they cannot
  show it matches what the game draws. **This is now the more interesting check than it was**, because
  the render visibly changed: the same maps are flatter and more uniform than the previous build's.
- **AC-8 / R-5 (edge cells):** whether our clamped outer ring of cells reads as a plausible border
  rather than a visible defect. Not a fidelity check — the original's own outer ring has no defined
  shading (see *Limitations*).
- **The seam check is retired, not pending.** The previous revision asked a developer to look for a
  one-vertex-wide edge seam caused by `DD2a`. That seam was an artefact of the superseded interior span;
  under the corrected arithmetic there is nothing to look for, and the measurement above is consistent
  with its absence.

**Status of this half: not yet run.** No result may be recorded here that was not observed on a real
install.

## Limitations and research flags

- **R-5 — the far edge is now decoded, and it has no defined shading to reproduce.** The research
  question this story raised (does the outer cell ring display, and what corners does the blit use?)
  is answered, and the answer dissolves the fidelity question rather than settling it: the original
  reads a cell's four corners at raw flat offsets into an unpadded `W·H` grid — a last-column cell's
  `+1` corner lands on the **next row's column 0**, a last-row cell's `+W` corner lands **past the
  allocation** — and the per-vertex brightness those reads sample is **never computed on the outer
  ring**, never filled by anything else on the relight path, and never zeroed by the allocator. The
  ring *is* drawn (it is not never-rendered padding), but what it shows is undefined. Our
  clamp-to-edge therefore stands as a **deliberate, defined choice** — the treatment the research
  itself names as the safe, intent-matching one for a port — and is no longer described anywhere as
  an approximation of the original. Carried honestly and **not** upgraded: whether the outermost ring
  ever reaches a visible (non-culled) pixel in the original is **Medium** (an un-pinned camera scroll
  clamp), and what the uncomputed ring holds at runtime is **Unknown** (static analysis; the running
  game was not observed). It affects a one-cell-wide border only; the decoded interior is unaffected.
- **R-4 — RETIRED.** The interior/border slope-magnitude seam existed only because a two-cell interior
  span met a one-cell border fill across a single `stepH`. Under the corrected span every difference is
  one cell, so the ring is on the same slope scale as the interior; the measured ring ranges above are
  consistent with that. No developer eyeball is asked for it any more.
- **R-3 — CLOSED, and it fired.** The story carried "the exact per-axis neighbour selection is
  Medium-confidence" from its first draft, and shipped the reading anyway. `TERR-LIGHT-028` closes the
  question and the reading was wrong on three counts: span, axes, and a missing lateral shear. The
  directional shading is now transcribed at High confidence from the raw x87, and the census cross-check
  above is independent corroboration that our transcription matches the research's. What is still **not**
  claimed: that the resulting render matches what the game draws on screen (see AC-8, pending).
- **R-6 — every level figure here is conditional on θ, and which θ shipped play uses is unknown.** The
  cycle-off default is the reference point for all of the above and is labelled as one throughout. With
  the cycle enabled θ sweeps `[−π/2,+π/2]` and the corpus ceiling travels 70 → 81. The research left this
  open on purpose (static observation cannot settle it), and nothing here settles it either.
- **R-7 — the `y == 1` lateral arm is emitted semantics.** We transcribe the branch the binary executes
  (selected by the row, because a later decrement clobbers the θ-sign compare's flags), not the branch the source
  may have intended. Effect confined to row 1; the research measured it as nil on the corpus level range,
  and it is not separately measured here.
- **The interactive viewer (`pkg/ui`/`cmd/mapview`) is not lit** in this story (`DD10`). Its GPU
  per-`(slot,sub)` cache would need re-keying on per-cell corner levels — a separable slice no FR/AC
  requires. The headless `terraintool` is the shading-evidence path (as 0004's AC-8 was). The viewer
  renders flat, consistent with the spec (its FRs are tier-level).
- **No research gap remains open for the decoded pipeline.** Every fact FR-1…FR-7 needs at the cycle-off
  default is present at High confidence in EXP-0023/EXP-0025/EXP-0029/EXP-0031 — the level formula, all
  constants, **both `Δh` terms**, the transform, the identity row, tint-before-multiply. Nothing was
  reverse-engineered or invented here. What is left open is not a gap in the arithmetic: it is which θ a
  live session runs at, what the uncomputed ring holds at runtime, and whether that ring is ever visible.
- **Two research readings this story relied on were corrected, neither by us.** EXP-0029 withdrew
  `TERR-LIGHT-013`'s "first row/column use a one-sided neighbour"; EXP-0031 derived the `Δh` terms and
  withdrew the two-perpendicular-axis, two-cell reading this story had implemented. The first cost
  nothing in code; the second is the whole of this revision.
- **The outer ring fill is still ours, and its arithmetic changed with everything else.** It is now the
  same formula with clamped height indices rather than a one-sided difference. Still a defined,
  index-safe fill of a region the original leaves uncomputed (FR-3, DD11) — not engine behaviour.
- **A stale statistic outlived the claim it came from, on both sides.** `TERR-LIGHT-018`'s `30…74`
  survived a framing correction unregenerated; this story's `30…77` survived a wrong-arithmetic
  implementation and was recorded as an open research consequence rather than as a symptom of our own
  code. Both were reported honestly and neither was caught by any test. The structural fix is FR-7 (a
  level figure without its θ, framing and window is not a result) and the figure-by-figure audit above.

## Notes and judgment calls

- **"Unshaded" is not ×1.0.** With the daytime intensities flat ground is level 46 (×1.5625), so the
  lit default is brighter than the old 0004 render (≈ level 64, ×1.0). The `-unshaded` flag keeps the
  0004 image available for comparison; `TestRenderShadedDefault` pins the exact ×1.5625 attenuation of
  a flat cell and that `-unshaded` reproduces the flat palette colour.
- **Downstream stories' evidence moved, their behaviour did not.** 0008 and 0009 record real-map render
  MD5s, and every shaded render changed. Their object/unit counts, marker pixel totals, marker
  coordinate digests and placement results are **byte-for-byte unchanged** across this revision; only the
  image digests moved. Their evidence is refreshed in their own `verification.md`, in their own commits,
  with no Go file in either story touched. Nothing was found there that would need a code change.
- **The map-stored light is an opt-in viewer choice.** `-maplight` seeds the sun from the map's
  stored fields, but `TERR-LIGHT-023` shows the engine overwrites those before use, so the default
  stays the documented daytime globals and the override is labelled non-fidelity in the code and the
  flag help.
- **The impassable dirt overlay is shaded by the cell level** (`DD7`) — an own-engineering assumption
  inherited from 0004 (the research does not decode the impassable blit's shading), labelled as such;
  a one-line change if it is later decoded.
- **No `pkg/formats` change and no new package**, so the DAG and `docs/ARCHITECTURE.md` are unchanged.

## Commits

| Commit | Subject | `SDD-Task` |
|---|---|---|
| `a9943ae` | chore(research): bump submodule to 87a72f4 (EXP-0025 shading table) | — (workflow) |
| `3441e04` | docs(0007-terrain-lighting): revise spec against EXP-0025 | — (workflow) |
| `9b0322f` | docs(0007-terrain-lighting): analysis — research provenance | — (workflow) |
| `99216df` | docs(0007-terrain-lighting): implementation plan | — (workflow) |
| `0d77ea7` | docs(0007-terrain-lighting): revise plan (adversarial re-read) | — (workflow) |
| `b5be2c7` | docs(0007-terrain-lighting): task breakdown | — (workflow) |
| `1215a1a` | feat(render/terrain): per-vertex relief level grid and light parameters | `0007-terrain-lighting/T1` |
| `de6027f` | feat(render/terrain): shading transform and per-pixel row interpolation | `0007-terrain-lighting/T2` |
| `264fe7f` | feat(render/terrain): lit full-map compositor | `0007-terrain-lighting/T3` |
| `e6eb1c6` | feat(terraintool): default to relief-shaded render; light controls | `0007-terrain-lighting/T4` |
| `23629d9` | docs(0007-terrain-lighting): record real-map evidence at the corrected .alm framing | — (workflow) |
| `78a2699` | docs(0007-terrain-lighting): relabel the type-0 light offsets to the corrected framing | — (workflow) |

The RED revision (research pin `8a406d6`, EXP-0031):

| Commit | Subject | `SDD-Task` |
|---|---|---|
| `c63cb25` | chore(research): bump the pinned submodule to 8a406d6 (EXP-0031) | — (workflow) |
| `fe25636` | docs(0007-terrain-lighting): analysis — EXP-0031 provenance for the RED revision | — (workflow) |
| `6aed243` | docs(0007-terrain-lighting): revise spec — the one-cell same-axis gradient | — (workflow) |
| `2b776e1` | docs(0007-terrain-lighting): revise plan — corrected span; DD2/DD2a/DD4 retired | — (workflow) |
| `2ee2298` | docs(0007-terrain-lighting): revise tasks — T6 the literal, T7 the span | — (workflow) |
| `cab4a07` | feat(render/terrain): the daytime default sun angle is the engine's literal | `0007-terrain-lighting/T6` |
| `09bef64` | feat(render/terrain): the per-vertex gradient is the one-cell same-axis span | `0007-terrain-lighting/T7` |

One `SDD-Task` id per implementation commit, none reused across either pass (T1…T5 are retired and T6/T7
continue from the highest existing id); workflow commits carry none; no `Co-Authored-By` trailer
anywhere.

## Conclusion

The terrain relief-lighting pipeline now implements the **decoded** gradient. Every acceptance
criterion's automatable part and all 7 derived properties pass, with every gate green and no game
install present; the three new criteria (AC-14 row axis, AC-15 lateral shear, AC-16 signed heights) each
discriminate against the model this story previously shipped, and the two mutations recorded above
confirm the tests can fail.

**The strongest single result is the census cross-check.** An independent implementation of
`TERR-LIGHT-028`, written from the spec rather than from the research probe, reproduces
`TERR-LIGHT-029`'s corpus levels exactly — `[30..70]` over 38 maps at 859 768 vertices, `[30..65]` over
the 10 root maps — and reproduces its whole θ-sweep table. The disagreement the previous revision
recorded (our 77 against a published 74) is resolved by both figures being wrong, not by one winning.

**What the correction did to the render:** the level distribution narrows and flattens. Ten-map levels
go `30…77` → `[30..65]`; the share of vertices at exactly 46 rises on every map; `Islands`' mean lit
channel sum moves 342.54 → 344.05 (ratio 1.5318 → 1.5385) while the unshaded 223.62 is unchanged; and
pixels darker than the unshaded image fall from 1.53 % to **one pixel in 67 million**, because a
one-cell span rarely pushes the level past the unattenuated row 64. Flat cells still shade to exactly
`ShadeRGBA(level 46)` on every pixel (475 cells on `Islands`, 310 on `Kids`), and `-unshaded` is
byte-identical across the revision.

**AC-8's fidelity half remains pending and is not claimed:** no one has looked at a lit render beside the
running game, and it matters more now than before, because the render visibly changed. The far-edge ring
(R-5) is unchanged in status — the original defines no edge shading, so our clamp is a deliberate choice
rather than an approximation — and the border-seam question (R-4) is retired rather than pending, since
the seam was an artefact of the superseded span. Every level figure above is conditional on
θ = 0.78539815; whether shipped play runs at that angle is open (R-6) and is not decided here.

The confidence of this conclusion is bounded accordingly: the tier arithmetic is now verified against
the research at the level of the corpus census, not just the formula's shape; the on-screen appearance of
the default daytime render is measured but still not compared against the game; and the outer ring is our
own defined treatment of a region the original leaves undefined — a plausibility question, not a fidelity
one.
