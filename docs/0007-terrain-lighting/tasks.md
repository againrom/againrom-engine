# Tasks — terrain relief lighting

**Reading key.** `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DD-x` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks. Task kinds: **implementation** (one coherent
product change → exactly one implementation commit, trailer `SDD-Task: 0007-terrain-lighting/T<n>`),
**developer-run verification** (agent authors, a human runs against a lawful install; no implementation
commit).

`pkg/render/terrain` and `cmd/terraintool` are already registered in the DAG allow-map and
`docs/ARCHITECTURE.md`, and this story adds **no package**, so no `internal/archtest`/`ARCHITECTURE.md`
edit is part of any task. All arithmetic stays stdlib-only (FR-6). For the first pass, tests were authored
by a separate context that had read `spec.md` but not the implementation; **for this revision the owner
set a lighter calibration** — one good test per behaviour actually changed plus its boundary, written by
the implementer, no separate-context author and no adversarial read (recorded as a deviation in
`plan.md`).

**Revision note.** T1…T5 (in git history) built the first pass against the superseded gradient reading.
Their IDs are **not reused**; the tasks below continue from T6. T1's `LevelGrid` arithmetic and T1's
level-grid tests are what T7 replaces — T2's transform, T3's compositor and T4's tool wiring stand.

## T6 — the daytime default sun angle becomes the engine's literal  *(implementation)*

`TERR-LIGHT-030`: the sun model's cycle-off arm stores the double `0x3fe921fb4d12d84a` = `0.78539815`
(`3.1415926/4`), not `π/4`. Land that correction on its own, before the gradient changes, so the two are
separable in history and in a bisect.

- Files: `pkg/render/terrain/light.go` (MODIFY) — `DefaultDaytime.Theta` becomes the decimal literal, with
  a code comment stating **why a literal rather than `math.Pi/4`**: the engine stores a constant, the two
  doubles differ from the eighth decimal, and both `stepH` and the lateral shear depend on the value
  (DD13). `pkg/render/terrain/light_test.go` (MODIFY) — the `DefaultDaytime` guard asserts the literal and
  asserts it **differs** from `math.Pi/4`, so a silent swap back fails. `cmd/terraintool/main.go` (MODIFY)
  — the `-theta` flag help names the corrected default (help text only; no behaviour change).
- Covers: FR-2; AC-1 (the default half); SC-1 (the guard); DD13.
- **Done when:** `TestLevelGridFlatIsFortySix` passes with the new guard; the `terraintool` summary line is
  unchanged (`%.4f` of both values is `0.7854`); `go build ./...`, `go vet ./...`, `gofmt -l` (tracked
  `*.go`), `go test -count=1 ./...`, `go test ./internal/archtest` and
  `bash scripts/check-no-game-assets.sh` are clean with no game install present.

## T7 — the per-vertex gradient becomes the one-cell same-axis span  *(implementation)*

The correction this revision exists for. `LevelGrid` currently forms `dx = H[x+1,y] − H[x−1,y]` and
`dy = H[x,y+1] − H[x,y−1]`; `TERR-LIGHT-028` shows the engine forms two adjacent one-cell differences
along the **same row axis**, laterally sheared by `tan|θ|`, over signed height bytes. Brownfield: the
existing spec-derived level tests pin the superseded model, so they are rewritten from the revised spec in
this same commit — leaving them would pin the defect.

- Files: `pkg/render/terrain/light.go` (MODIFY) — `LevelGrid`'s gradient: forward `Δh₁` to row `y+1`,
  backward `Δh₂` from row `y−1`, each `H[x ∓ tan|θ|, ·]` interpolated from the vertex's column and its one
  lateral neighbour; `int8` height reads; the `y == 1` lateral arm as emitted; every height index clamped
  into the grid for the outer ring. The clamp stays **inside** the per-half helper and the single `0.5`
  outside it (DD3, DD11, DD12). `pkg/render/terrain/light_test.go` (MODIFY) — oracles rewritten from the
  revised spec formula.
- Covers: FR-3, FR-6 (level half); AC-3, AC-4, AC-5a, AC-14, AC-15, AC-16; P-1, P-2, P-3, P-4;
  SC-2…SC-4, SC-5a, SC-6, SC-18, SC-19, SC-20; DD3, DD11, DD12.
- **Done when:** `TestLevelGridFlatIsFortySix`, `TestLevelGridFlatIntensity`, `TestLevelGridRidge`,
  `TestLevelGridClampsAxes`, `TestLevelGridBordersAndDegenerate`, `TestLevelGridPure`,
  `TestLevelGridRowAxisGradient`, `TestLevelGridLateralShear`, `TestLevelGridSignedHeights` and
  `TestLightFromFields` pass; `TestCompositeLit` and the `cmd/terraintool` suite stay green untouched; all
  gates above clean with no game install present.

## T8 — re-measured evidence against a lawful install  *(developer-run verification)*

Run the tool and the render tier against a lawful install and record **evidence only** — no game bytes.
Because the arithmetic moved, this is not a spot check: **every derived figure the previous revision
recorded is re-measured or explicitly superseded** (FR-7).

- Produces: evidence in `verification.md` — the corpus level census at the stated θ, light bytes, framing
  and window, cross-checked against `TERR-LIGHT-029`'s `[30..65]`/`[30..70]` and its θ-sweep table; the
  per-map level table and outer-ring range; the lit-vs-unshaded distribution and the exact flat-cell
  figures; that `-unshaded` still reproduces the 0004 image byte-for-byte; and the ten shipped maps'
  stored `+0x08`/`+0x10`/`+0x14` re-read (AC-7 corpus part). Anything that cannot be re-measured is
  restated as unrepeatable, never carried over.
- Covers: AC-8; AC-7 (corpus part); AC-13 (corpus part); FR-7; SC-17; R-1, R-5, R-6.
- **Done when:** `verification.md` records the run (or an explicit "not run" limitation with the reason);
  `bash scripts/check-no-game-assets.sh` stays clean; no rendered PNG or game byte is committed.

## T9 — downstream evidence re-runs  *(developer-run verification, outside this story)*

The shading arithmetic changes, so every shaded render moves; 0008 and 0009 recorded render MD5s. Their
**code and tests are out of scope** — only their `verification.md` evidence is refreshed, one commit per
story, separate from this story's commits. What must **not** move is their object/unit counts and their
marker coordinates; that is a falsifiable check, not an eyeball. Anything found that would need a code
change in either story stops this pass and goes back to the owner as a new finding.

- Produces: refreshed developer-run evidence in `docs/0008-structures-overlay/verification.md` and
  `docs/0009-units-overlay/verification.md`.
- **Done when:** both files record the re-run with the new digests and the unchanged counts/coordinates;
  no Go file in either story is touched.

---

*(Retired task list from the first pass, kept for lineage. IDs T1…T5 are not reused.)*

## T1 — the per-vertex level grid and light parameters  *(implementation, superseded by T6/T7)*

The pure relief-lighting front half: the light parameters, their daytime defaults, the map-stored override
builder, and the `W×H` level grid computed by the exact `TERR-LIGHT-013` formula.

- Files: `pkg/render/terrain/light.go` (ADD) — `Light{Theta float64; Ambient, Range uint8; SkyTint [3]uint8}`,
  `DefaultDaytime` (θ=π/4, ambient=0x0e, range=0x20, tint (0,0,0)); `LightFromFields(angle float32,
  ambient, rng uint32) Light` (DD8); `LevelGrid(heights []uint8, w, h int, lt Light) []uint8` — central
  difference in the interior, one-sided on all four borders, indices clamped, `float64` slope math widened
  from `int` heights, truncating `ftol` (DD2, DD2a, DD3, DD4).
  `pkg/render/terrain/light_test.go` (ADD, separate-context).
- Covers: FR-2, FR-3, FR-6 (level half); AC-1, AC-2, AC-3, AC-4, AC-5, AC-7 (unit part); P-1, P-2, P-3,
  P-4; SC-1…SC-6, SC-13; DD2, DD2a, DD3, DD4, DD8.
- **Done when:** `TestLevelGridFlatIsFortySix`, `TestLevelGridFlatIntensity`, `TestLevelGridRidge`,
  `TestLevelGridClampsAxes`, `TestLevelGridBordersAndDegenerate`, `TestLevelGridPure`, and
  `TestLightFromFields` pass; `go build ./...`, `go vet ./...`, `gofmt -l` (tracked `*.go`),
  `go test ./...`, `internal/archtest` and `bash scripts/check-no-game-assets.sh` are clean with no game
  install present.

## T2 — the shading transform and per-pixel row interpolation  *(implementation)*

The pure relief-lighting back half: the integer per-channel attenuation and the bilinear-then-truncate
row selection. Independent of T1 (both are pure; `InterpRow` takes corner levels as inputs).

- Files: `pkg/render/terrain/shade.go` (ADD) — `ShadeChannel(chan, tint uint8, level int) uint8` =
  `clamp(((int(chan)+int(tint))·(96−level))/32, 0, 255)` truncating; `ShadeRGBA(c color.RGBA,
  tint [3]uint8, level int) color.RGBA` (alpha opaque); `InterpRow(l00, l10, l01, l11 uint8, x, y, cell int)
  int` — `float64` bilinear across a `cell`-wide square, truncated (DD5, DD6).
  `pkg/render/terrain/shade_test.go` (ADD, separate-context).
- Covers: FR-4, FR-5, FR-6 (transform half); AC-6, AC-9, AC-10, AC-11, AC-12, AC-13; P-5, P-6, P-7;
  SC-7…SC-12; DD5, DD6.
- **Done when:** `TestInterpRow`, `TestShadeChannelTransform`, `TestShadeIdentityRow`,
  `TestShadeTintBeforeMultiply`, `TestShadeMonotonic`, and `TestShadeCorpusRange` pass; all gates above stay
  clean with no game install present.

## T3 — the lit full-map compositor  *(implementation)*

Compose T1's level grid with T2's transform over a whole map: per cell take the four clamped corner
levels, per pixel interpolate → truncate → shade the resolved palette colour (including a dirt-overlaid
pixel). `Composite` is left untouched (DD1). Depends on T1 and T2.

- Files: `pkg/render/terrain/lit.go` (ADD) — `CompositeLit(ts *Tileset, g Grid, heights []uint8, lt Light,
  scale int) (*Render, error)`: same validation as `Composite` plus a heights-length check, reusing
  `Resolve`/`IsImpassable`/`DirtSubCell`/`overlayPixel`/`paletteColor`; supplies each cell's four
  edge-clamped corner levels and shades every pixel by the interpolated row (DD1, DD6, DD7).
  `pkg/render/terrain/lit_test.go` (ADD, separate-context).
- Covers: FR-1, FR-5 (composite scope); AC-1/AC-10 at composite scope; SC-14; DD1, DD6 (edge-clamp), DD7;
  R-5 (the deliberate far-edge clamp, for a ring the original leaves undefined).
- **Done when:** `TestCompositeLit` passes (identity over a uniform-level-64 grid equals `Composite`; a
  ridge reaching the last row/column shades non-uniformly and consistently at the clamped edge; absent
  slots still counted as placeholders); all gates above stay clean with no game install present.

## T4 — wire shading into the headless render harness  *(implementation)*

Make `terraintool` the shading-evidence tool: default to `CompositeLit` with `DefaultDaytime`, source the
height grid from the `.alm`, and expose the light controls. Depends on T3.

- Files: `cmd/terraintool/main.go` (MODIFY) — default path calls `CompositeLit` with `m.Altitudes` and
  `DefaultDaytime`; add `-unshaded` (route back through `Composite`), `-maplight` (build the light from
  `m.Angle`/`m.Meta.Word10`/`m.Meta.Word14` via `LightFromFields`), and `-theta`/`-ambient`/`-range`
  individual overrides; extend the summary line with the light in effect. `cmd/terraintool/main_test.go`
  (MODIFY) — update the existing colour assertions to the shaded default (flat fixture → level 46 →
  ×1.5625), assert `-unshaded` reproduces the 0004 image, and assert the override flags reach the
  compositor.
- Covers: FR-1, FR-2 (override wiring); SC-16; DD8, DD10.
- **Done when:** the updated `TestRenderEndToEnd`/`TestRenderScaleAndEnvAssetRoot`/`TestRenderFailures`/
  `TestRenderWithEmptyArchive` and a new `-unshaded`/override test pass; all gates above stay clean with no
  game install present.

## T5 — shaded-render evidence against a lawful install  *(developer-run verification)*

Run `terraintool` against a lawful install and record **evidence only** — no game bytes.

- Produces: evidence in `verification.md` — a real map rendered with the default daytime light showing slope
  relief with flat ground at ×1.5625 and steep-away faces darker (AC-8), whether the one-vertex border seam
  of DD2a/R-4 is visible (a plausibility check on our own border fill — the original computes no border
  vertex, so there is nothing there to match), whether our clamped far-edge cell ring (R-5) reads as a
  plausible border rather than a defect, that `-unshaded` reproduces the 0004 flat image, and the ten
  shipped maps' stored `+0x08`/`+0x10`/`+0x14` read as evidence (every angle in `[−π/2, π/2]`, both scalars
  byte-range — AC-7 corpus part).
- Covers: AC-8; AC-7 (corpus part); SC-17; R-1, R-4, R-5.
- **Done when:** `verification.md` records the run (or an explicit "not run" limitation with the reason);
  `bash scripts/check-no-game-assets.sh` stays clean; no rendered PNG or game byte is committed.

## Traceability

Current-revision tasks are T6…T9; T1…T5 are the first pass and appear only where a requirement is still
carried entirely by code they landed and this revision does not touch.

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (height plumbing) | SC-5a, SC-14, SC-16 | T1, T3, T4 (unchanged) |
| FR-2 (light params + override + default) | SC-1, SC-2, SC-13, SC-16 | **T6**, T1/T4 (unchanged parts) |
| FR-3 (brightness grid) | SC-1…SC-4, SC-5a, SC-6, SC-18…SC-20 | **T7** |
| FR-4 (interpolation) | SC-7, SC-12 | T2 (unchanged) |
| FR-5 (application) | SC-8…SC-12, SC-14 | T2, T3 (unchanged) |
| FR-6 (purity / DAG) | SC-15 | **T7**, T2, T3 |
| FR-7 (statistics carry their parameters) | SC-17 | **T8** |
| AC-1 | SC-1 | **T6** (the default), **T7** (the level) |
| AC-2 | SC-2 | **T7** |
| AC-3 | SC-3 | **T7** |
| AC-4 | SC-4 | **T7** |
| AC-5 | *(retired)* | — |
| AC-5a | SC-5a | **T7** |
| AC-6 | SC-7 | T2 (unchanged) |
| AC-7 (unit part) | SC-13 | T1 (unchanged) |
| AC-7 (corpus part) | SC-17 | **T8** |
| AC-8 | SC-17 | **T8** |
| AC-9 | SC-8 | T2 (unchanged) |
| AC-10 | SC-9, SC-14 | T2, T3 (unchanged) |
| AC-11 | SC-10 | T2 (unchanged) |
| AC-12 | SC-11 | T2 (unchanged) |
| AC-13 (table bound) | SC-12 | T2 (unchanged) |
| AC-13 (corpus census) | SC-17 | **T8** |
| AC-14 (row axis, no perpendicular gradient) | SC-18 | **T7** |
| AC-15 (lateral shear) | SC-19 | **T7** |
| AC-16 (signed height reads) | SC-20 | **T7** |
| P-1 (level in [0,95]) | SC-4 | **T7** |
| P-2 (grid pure) | SC-6 | **T7** |
| P-3 (no read outside grid) | SC-5a | **T7** |
| P-4 (flat ⇒ constant) | SC-1 | **T7** |
| P-5 (shade pure) | SC-8 | T2 (unchanged) |
| P-6 (level 64 identity) | SC-9 | T2 (unchanged) |
| P-7 (monotone, in [0,255]) | SC-11 | T2 (unchanged) |
