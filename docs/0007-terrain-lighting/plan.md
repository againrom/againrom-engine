# Plan — terrain relief lighting

Reading key: `FR-x`/`AC-x`/`P-x` → `spec.md`. This file fixes the API contract, the design decisions,
and the success criteria against which the work is verified. It is derivable from the spec alone.

## Approach

The pipeline itself is unchanged and stays where it is: a per-vertex **level grid** computed from the
height field and light parameters (FR-3), a **bilinear corner interpolation** selecting an integer
attenuation row per pixel (FR-4), an integer **per-channel shading transform** (FR-5), a lit compositor
over a whole map, and `cmd/terraintool` as the shading-evidence tool. 0004's unshaded `Composite` remains
untouched. No package is added, so the DAG is unchanged.

**What this revision changes is one function's arithmetic.** `LevelGrid` currently forms the height
gradient as a two-cell central difference on two perpendicular axes; `TERR-LIGHT-028` shows the engine
forms two *adjacent one-cell* differences along the *same* row axis, each laterally sheared by `tan|θ|`,
each clamped to a level before the average, over *signed* height bytes. And `DefaultDaytime.Theta`
currently holds `math.Pi/4`, where the engine stores a literal that is not `π/4` (`TERR-LIGHT-030`).
Both are corrections inside `pkg/render/terrain/light.go`; every caller, signature and downstream
behaviour is unchanged, so the blast radius is *values*, not contracts. The shading transform, the
interpolation, the compositor and the tool are not touched.

**Terrain: brownfield for `LevelGrid`.** Its behaviour is already pinned by `light_test.go`, whose oracles
are derived from the spec formula rather than the implementation; those oracles are rewritten from the
revised spec in the same commit that changes the arithmetic, because the old oracles assert the
superseded model and would otherwise pin the defect.

**Gate calibration for this pass, recorded as a deviation.** The owner set the rigor for this revision:
one good test per behaviour actually changed plus its boundary, representative cases over exhaustive
sweeps, and **no separate-context test author and no adversarial plan read** — those independent-context
gates (`SDD/PROFILE.md` → Gates) are deliberately not run here and their absence is recorded rather than
papered over. What does not relax: the four golden rules, evidence honesty, the standing gate green
before every commit, and one task → one commit.

## Facts verified during planning (baseline, frozen)

Baseline = the working tree at the start of this revision (the landed 0007 code), not the first pass's
pre-implementation state.

- `pkg/render/terrain/light.go` ships `Light{Theta, Ambient, Range, SkyTint}`, `DefaultDaytime` with
  `Theta: math.Pi/4`, `LightFromFields`, and `LevelGrid(heights []uint8, w, h int, lt Light) []uint8`
  computing `dx = H[x+1,y] − H[x−1,y]`, `dy = H[x,y+1] − H[x,y−1]` with edge-clamped indices and unsigned
  height reads. Its structure — `finalScale * (axisValue(..) + axisValue(..))` with the clamp inside
  `axisValue` — already matches `TERR-LIGHT-028`'s clamp-then-average and is retained.
- `LevelGrid` has exactly two callers: `CompositeLit` (`lit.go`) and the tests. Its signature does not
  change, so `lit.go`, `shade.go`, `composite.go`, `overlay.go`, `pkg/ui` and `cmd/mapview` need no edit.
- `cmd/terraintool` prints the light as `shaded (theta=%.4f ambient=%d range=%d)`. `%.4f` of both
  `math.Pi/4` and `0.78539815` is `0.7854`, so the summary line is character-identical before and after —
  which is what keeps 0008's recorded summary line valid. Its `-theta` flag help names the default.
- `cmd/terraintool`'s synthetic `.alm` fixture carries an all-zero type2 grid, so every fixture vertex is
  flat and levels to 46 under either arithmetic; its shading assertions are unaffected.
- `lit_test.go`'s relief case takes its expected levels from `LevelGrid` itself and asserts only that the
  render is non-uniform and differs from the unshaded one — properties the corrected arithmetic preserves
  on that fixture.
- `pkg/formats/alm` exposes `Map.Altitudes []uint8`, `Map.Angle float32`, `Map.Meta.Word10/Word14`. No
  `pkg/formats` change (spec Constraints). `pkg/render/terrain` is stdlib-only and registered in
  `internal/archtest` + `docs/ARCHITECTURE.md`; no new package, so no DAG edit.
- The **10 root maps' altitude bytes all lie in `[0,127]`**, so signed height reads change nothing on real
  data; the difference is observable only on synthetic input ≥ `0x80`.

## Files to touch

| Path | Intent | Why |
|---|---|---|
| `pkg/render/terrain/light.go` | MODIFY | `DefaultDaytime.Theta` → the engine's cycle-off literal, with the reason for a literal in the code (FR-2, DD13); `LevelGrid`'s gradient → the one-cell forward/backward same-axis span with the `tan|θ|` lateral shear, signed height reads and the clamped-index border fill (FR-3, DD11, DD12). Stdlib + `math` only. |
| `pkg/render/terrain/light_test.go` | MODIFY | Rewrite the level-grid oracles from the revised spec formula (AC-1…AC-4, AC-5a, AC-14…AC-16, P-1…P-4); `TestLightFromFields` unchanged. |
| `cmd/terraintool/main.go` | MODIFY | One line: the `-theta` flag help names the corrected default. No behaviour change. |

**Not touched, deliberately:** `shade.go`, `lit.go`, `composite.go`, `overlay.go` and their tests (the
transform, the row interpolation, the compositor and both overlays are unaffected by the gradient);
`cmd/mapview`/`pkg/ui` (unlit, DD10); `pkg/formats`; `internal/archtest`; `docs/ARCHITECTURE.md`. The
**0008 and 0009 stories' code and tests are out of scope entirely** — their recorded render evidence moves
because the shading moves, and that is refreshed in their own `verification.md`, in their own commits.

## Design decisions

- **DD1 — All lighting is new code in `pkg/render/terrain`; 0004's `Composite` is retained unchanged, and
  `CompositeLit` is added alongside it.** FR-6 puts the arithmetic in this tier; FR-5 requires the unshaded
  path to remain available. Adding a sibling compositor keeps 0004's contract and tests intact and makes the
  change a pure addition (greenfield), while the shared helpers (`Resolve`, `overlayPixel`, `paletteColor`,
  `drawCell` geometry) are reused, not forked in spirit.
  *Rejected:* threading a lighting flag through `Composite` and mutating it — that changes a shipped,
  tested contract (brownfield churn) for no gain, and risks a stale caller silently shading with wrong
  inputs.

- **DD2 — RETIRED.** The pre-EXP-0031 decision "central differences in the interior, one-sided differences
  on every border" rested on reading EXP-0023's "per axis `i`" as two perpendicular axes with a two-cell
  span. `TERR-LIGHT-028` shows the engine forms neither. Replaced by **DD11**. Its ID is not reused.

- **DD2a — RETIRED.** It documented an interior-vs-border slope-magnitude seam that existed *only* because
  a two-cell interior span met a one-cell border fill across a single `stepH`. Under DD11 every difference
  spans one cell, so there is no scale discontinuity at the ring and nothing to document, normalize or
  eyeball. (What DD2a got right and DD11 keeps: no `/2` is invented anywhere — the only `0.5` in the model
  is the final average of two clamped levels.) Its ID is not reused; the risk it fed, **R-4**, is retired
  with it.

- **DD3 — The per-vertex formula is implemented in `float64` exactly as `TERR-LIGHT-013`, and the byte is a
  truncating `ftol`.** `stepH = 32.0 / cos|θ|`; per half `slope = atan2(Δh, stepH)`;
  `axis = clamp(L − R·sin(π/6 − slope), 0, 95)`; `level = int(0.5·(axis₁ + axis₂))` (Go's float→int
  conversion truncates toward zero, matching `ftol` for the non-negative average). Constants are the IEEE
  doubles `32.0`, `math.Pi/6`, `95.0`, `0.5`. `L = (range>>1) + ambient + 0x20`, `R = range`, computed in
  integers from the light bytes. **The clamp stays inside the per-half helper and the `0.5` stays outside
  it** — `finalScale * (axisValue(Δh₁) + axisValue(Δh₂))` — because `TERR-LIGHT-028(e)` makes that ordering
  load-bearing: the two halves are converted to levels independently, so the routine is not algebraically
  equal to any single-difference form once either half clamps. FR-6 permits float here; the result is a
  byte and no float reaches any which-graphic decision. At a flat vertex both `Δh` are 0, so the level
  reduces to `ftol(L − R·math.Sin(math.Pi/6))`; because `math.Sin(math.Pi/6) = 0.4999999999999999…`, this
  is what the spec's `ftol(L − R/2)` means in float64, and it yields exactly **46** for the daytime
  `L=62, R=32` **at any θ**. The flat-vertex oracle any test uses MUST be this same float expression, not
  an integer `L − (R>>1)` (they diverge for odd `range`) — see SC-2.
  *Rejected:* fixed-point integer trigonometry — it would diverge from the game's FP result at the byte
  boundary for no determinism benefit (the output is already a byte and the inputs are bounded).

- **DD4 — RETIRED.** "θ enters only through `cos|θ|`; the daytime default is `θ = π/4`" is wrong twice
  over: θ also enters as the lateral shear `tan|θ|` (`TERR-LIGHT-028`), and the engine's cycle-off default
  is a literal that is not `π/4` (`TERR-LIGHT-030`). Replaced by **DD13**. Its ID is not reused.

- **DD11 — The gradient is two adjacent one-cell differences along the same (row) axis, laterally sheared
  by `tan|θ|`, and the outer vertex ring is filled by the same formula with every height index clamped
  into the grid.** `LevelGrid(heights, w, h, lt)` still returns `W*H` bytes and keeps its signature. Per
  vertex it forms `Δh₁ = hA − H[x,y]` from row `y+1` and `Δh₂ = H[x,y] − hB` from row `y−1`, where `hA`/`hB`
  are that row's height at the fractional column `x ∓ tan|θ|`, interpolated from the vertex's own column
  and its single lateral neighbour (`−tanT` for `θ ≥ 0`, `+tanT` for `θ < 0`). No perpendicular `x`
  difference is formed anywhere. **Why one-cell and not central:** `TERR-LIGHT-028` reads both terms from
  the raw x87 at High confidence and states that no instruction forms `H[…,y+1] − H[…,y−1]`; a central
  difference roughly doubles every gradient (measured: it lifts the 38-map ceiling from 70 to 78).
  **Why the shear:** it is not decorative — deleting it moves the 38-map ceiling 70 → 66.
  **The border ring.** The formula reads `x±1` and `y±1`, so its no-clamp region is the strict interior
  `W-2 × H-2` — the *same shape* the superseded model needed, so the ring this project fills is unchanged
  in extent. We fill it by clamping every height index into `[0,W-1]×[0,H-1]` and applying the same
  formula. On the top/bottom edge the forward or backward half degenerates to a purely lateral difference;
  on the left/right edge the shear operand collapses and the shear vanishes; a `1×N` or `N×1` grid yields
  zero deltas (P-3). A flat grid is still uniformly 46, ring included (P-4). This is **our** defined,
  index-safe fill of a ring the original leaves uncomputed (`TERR-EDGE-025`), and it is labelled as ours
  everywhere — it is not promoted to engine behaviour, and no seam is smoothed away by inventing terms.
  *Rejected:* extending the formula past the grid with an extrapolated row/column — that invents heights.
  *Rejected:* leaving the ring uninitialised as the original does — non-deterministic and P-3/AC-5a require
  a total, in-range grid.
  *Rejected:* narrowing `LevelGrid` to the strict interior and having the compositor cope — it moves the
  same undefined-ring decision one layer up and breaks the total-grid contract every caller relies on.

- **DD12 — Signed height reads, and the `y == 1` lateral arm, follow the rule the engine executes.** Heights are read `int8` (`TERR-LIGHT-028(d)`: a sign-extending load), so a byte ≥ `0x80` is negative; on the
  shipped corpus nothing reaches `0x80`, so this is observable only on synthetic input, and it is
  implemented because it is decoded, not because it matters here. The backward half's lateral column is
  selected by `y == 1` rather than by the sign of θ, because the engine computes a θ compare and then
  discards it (a later decrement overwrites the flags before the branch; `TERR-LIGHT-028`). We implement the **executed** rule.
  *Honest bound:* whether the original source intended the θ test is an inference EXP-0031 explicitly
  declines to make, and we make it no more strongly; the effect is confined to row 1 and the research
  measured it as nil on the corpus range.
  *Rejected:* "fixing" the branch to the θ sign it computes — that would be implementing what we guess the
  source meant instead of what the binary does (golden rule 4).

- **DD13 — θ enters twice, and the default is the engine's cycle-off literal.** `tan|θ|` shears the lateral
  sample and `cos|θ|` sets `stepH`, so the azimuth both picks the sample point and lengthens the baseline.
  `DefaultDaytime.Theta` is the decimal literal `0.78539815` — the double `0x3fe921fb4d12d84a` the sun
  model stores when the day/night cycle is off (`TERR-LIGHT-030`) — **not** `math.Pi/4`. The code says why:
  the engine stores a constant rather than computing one, the two doubles differ from the eighth decimal,
  and writing `math.Pi/4` would substitute our arithmetic for the game's. The flat-vertex level stays 46 at
  any θ because `Δh = 0` on both halves (AC-1, P-4). Because both `stepH` and the shear depend on θ, and
  because with the cycle enabled θ sweeps `−π/2 → +π/2` daily, every level statistic is quoted with its θ
  (FR-7).
  *Rejected:* keeping `math.Pi/4` as "close enough" — it is a different double, it is not what the engine
  stores, and the difference is exactly the kind of silent substitution this revision exists to remove.
  *Rejected:* asserting the cycle-off default is what shipped play uses — the research leaves that open on
  purpose; it is our labelled reference point, nothing more.

- **DD5 — Shading attenuates the resolved 8-bit palette colour per channel; there is no precomputed
  `[96][256]` table.** `ShadeChannel(chan, tint, level) = clamp(((int(chan)+int(tint))·(96−level))/32, 0,
  255)` with truncating integer division, and `ShadeRGBA` applies it to R,G,B (alpha forced opaque). The
  game bakes this into a `[96][256]` u16 table addressed by `(level, palette index)` and packs to RGB565
  (`TERR-LIGHT-018/019`), but because the transform is per-channel linear, applying it to the already
  palette-resolved `color.RGBA` yields the identical **pre-pack 8-bit** channel value FR-5 asks for, with
  no table and no dependence on all strips sharing a palette. RGB565 packing is out of scope (spec).
  *Rejected:* building the literal `[96][256]` table — it would bind us to one palette (sound per
  `TERR-LIGHT-022` but an unnecessary runtime assumption) and to RGB565, which the spec excludes; the
  table is a game-side speed optimization, not part of the observable contract.

- **DD6 — Per-pixel row selection is the truncated bilinear interpolation of the cell's four corner
  levels.** `InterpRow(l00, l10, l01, l11, x, y, cell)` interpolates in `float64` across a `cell`-wide
  square (`cell = CellSize = 32`) and returns `int(interpolant)`, truncating. FR-4's observable contract is
  "truncate the bilinear interpolant to an integer row," which this implements exactly; it also **mirrors**
  the intent of the blit's `& 0xfffffe00` mask, though a compute-then-truncate float bilinear can differ
  from the game's fixed-point accumulate-and-mask by **±1 row at an exact truncation boundary**. FR-6
  explicitly permits float here — the level selects an attenuation row, never which graphic is drawn — so
  the divergence is acceptable and is recorded rather than glossed. At the four corner pixel positions
  `InterpRow` returns the corner level exactly (AC-6).
  Interior cells are unambiguous: cell `(col,row)` uses the level-grid values at `(col,row)`, `(col+1,row)`,
  `(col,row+1)`, `(col+1,row+1)`. **The far-edge mapping is decoded, and the finding is that there is no
  original behaviour to match.** `W×H` vertices only fully bound `(W-1)×(H-1)` cells, so the last cell
  row/column needs a `(col+1)`/`(row+1)` that does not exist. The original resolves that by not resolving it:
  it reads the corner at a raw flat offset into the same unpadded grid — for a last-column cell that lands on
  the **next row's column 0**, for a last-row cell **past the grid allocation** — and the brightness values
  those reads sample were never computed on the outer ring in the first place. The ring *is* drawn, so this
  is not never-rendered padding; but its shading is undefined, not merely undecoded. We therefore **clamp**
  the `+1` indices to the grid edge as a **deliberate, defined choice** — the safe treatment the research
  itself names for a port — keeping the compositor total (P-3) and the last cell row/column degenerate along
  the clamped axis. It is neither a fidelity claim nor an approximation of a known original behaviour: there
  is no defined original value to approximate. SC-14 exercises it on a **non-flat** grid so the edge
  behaviour is asserted for self-consistency with our edge-vertex levels.
  *Rejected:* nearest-vertex (flat per-cell) shading — it would drop the Gouraud gradient `TERR-LIGHT-011`
  interpolates and fail AC-6. *Rejected:* treating the last row/column as an error — the compositor must
  stay total over any `W×H` grid (P-3).

- **DD7 — The lit compositor shades the final per-pixel colour, including an impassable cell's dirt
  overlay, by that cell's interpolated level. This is an own-engineering assumption inherited from 0004,
  not a decoded fact.** The impassable dirt overlay is a 0004 construct the spec never mentions; the research
  does not say whether the engine shades it. We choose to shade the composited pixel (whichever of terrain or
  dirt shows after `overlayPixel`) by the cell level, because the dirt overlay is terrain-class ground drawn
  from the shared terrain palette (`TERR-LIGHT-022`) and leaving it full-bright would make blocked ground
  brighter than the slope it sits on. Under FR-5 the composited pixel *is* "the source pixel" at that
  location once resolved, so this stays within FR-5's letter; it is labelled a project choice, not a claim
  about the engine, and is a one-line change if the research later decodes the impassable blit's shading.
  *Rejected:* shading terrain but leaving dirt at full brightness — visually inconsistent and equally
  un-decoded.

- **DD8 — Light overrides are built by `LightFromFields(angle float32, ambient, rng uint32) Light`,
  consuming the `.alm`'s stored fields as their file types.** θ is the stored float passed through; ambient
  and range are the low bytes of the stored `u32`s (the engine's `P+0x1c/0x1d` byte slots,
  `TERR-LIGHT-023`). `terraintool -maplight` sources them from `Map.Angle`/`Meta.Word10`/`Meta.Word14`;
  `-theta`/`-ambient`/`-range` override individually. The spec states plainly (per `TERR-LIGHT-023`) that
  using the map-stored fields is a viewer choice, not engine fidelity, so the override is opt-in and the
  default stays the documented daytime globals.
  *Rejected:* defaulting to the map-stored light — `TERR-LIGHT-023` shows the engine overwrites those
  fields before use, so it would be a fidelity claim the research explicitly denies.

- **DD9 — AC-7 splits into a unit part and a corpus-evidence part.** The unit test drives
  `LightFromFields` over synthetic stored values (a representative in-range angle and byte-range scalars),
  asserting θ passes through and the scalars are taken as bytes — the render-tier behavior that is
  testable without a game install. The literal corpus claim ("the ten shipped maps' angles lie in
  `[−π/2,π/2]` and both scalars are byte-range") is recorded as developer-run evidence in `verification.md`
  alongside AC-8, exactly as the spec's own verification mapping routes it ("the shipped maps' stored
  scalars read as evidence"). No game bytes are committed.
  *Rejected:* baking the ten maps' actual stored values into a fixture and asserting the range as a "unit"
  test — that would smuggle game-derived constants into the suite and misrepresent evidence as a unit
  check.

- **DD10 — The interactive viewer (`pkg/ui`/`cmd/mapview`) is not lit in this story; the shading-evidence
  path is the headless `terraintool`.** AC-8 (the only render-evidence AC) asks for "a real GOG map
  rendered with shading … recorded in `verification.md`", which the headless PNG serves directly, as 0004's
  AC-8 did. Lighting the GPU viewer would require re-keying its one-image-per-`(slot,sub)` cache on the four
  per-cell corner levels and CPU-pre-shading each variant — a materially different caching model and a
  separable vertical slice that no FR/AC requires. The viewer stays flat, which contradicts nothing in the
  spec (its FRs are tier-level; its only viewer mention is water ordering). Recorded as a limitation in
  `verification.md`.
  *Rejected:* forcing GPU-viewer shading into this story — it doubles the blast radius for behavior no
  acceptance criterion pins, and risks a cache blow-up on a large map; it is the natural next story.

## Success criteria

Each maps to a named test (unit) or a developer-run procedure (manual).

1. **SC-1 (FR-2/FR-3, AC-1, P-4)** — a flat grid at the daytime default levels every interior vertex to
   exactly 46, and the whole grid is constant; the test also guards the default itself — θ is the literal
   `0.78539815` and is **not** `math.Pi/4` (asserted as a difference, so a silent swap back fails), ambient
   `0x0e`, range `0x20`, tint `(0,0,0)`. *Test:* `TestLevelGridFlatIsFortySix`.
2. **SC-2 (FR-3, AC-2)** — a flat grid at a given `ambient`/`range` levels every interior vertex to
   `ftol(L − R·math.Sin(math.Pi/6))` for that `L=(range>>1)+ambient+0x20`, `R=range` — the flat-vertex value
   at `slope = 0`, computed in `float64` identically to the level formula (the spec's `ftol(L − R/2)` in
   float, NOT an integer `L − (R>>1)`). Tested over the daytime pair (→ 46) and at least one further pair,
   with `range` chosen so the truncation is not on an integer boundary. *Test:* `TestLevelGridFlatIntensity`.
3. **SC-3 (FR-3, AC-3)** — a **row-axis** ramp (height varying with `y`, constant along `x`) pins the
   sign→level direction against the exact computed value. Along such a field the lateral shear contributes
   nothing (each row is flat), so both halves take the same one-cell difference `Δh` and the level is
   `ftol(axis(Δh))` with **no halving toward 46** — an up-ramp of `+20/cell` gives `axis≈68` → level 68, a
   down-ramp `−20/cell` gives `axis≈32` → level 32. The test asserts the **exact** level from the spec
   formula on both signs and that it is strictly above 46 up-slope and strictly below 46 down-slope.
   *Test:* `TestLevelGridRidge`.
4. **SC-4 (FR-3, AC-4, P-1)** — with a saturating light (`range = 0xFF`, `ambient = 0`, so `L=159, R=255`),
   each half reaches both clamp ends and no level leaves `[0,95]`: a row-ramp steep enough in the up
   direction drives both halves past 95 (level exactly **95**), the same ramp reversed drives both below 0
   (level exactly **0**), and a fixture that drives the two halves in opposite directions levels to
   `ftol(0.5·(95+0)) = 47` — showing the halves clamp independently, which `TERR-LIGHT-028(e)` makes
   load-bearing. No level leaves `[0,95]` for any hostile height or θ. (Height deltas at the daytime
   intensities cannot reach the outer clamps — the reachable band there is `[30,89]` — so the test uses an
   override light, which FR-2 permits.) *Test:* `TestLevelGridClampsAxes`.
5. **SC-5 — RETIRED** (mapped to the retired AC-5, whose one-sided-neighbour contract belonged to the
   superseded model). Its ID is not reused.
5a. **SC-5a (FR-3, AC-5a, P-3)** — a grid including its outer vertex ring levels ring vertices by the same
   formula with clamped indices: on a row-varying ramp a top/bottom-edge vertex differs from the interior
   in the documented direction (its missing row collapses that half to a lateral difference), a flat grid
   is uniformly 46 **including** the ring, and `LevelGrid` reads no index out of range for every
   `W,H ≥ 1` (including `1×N`, `N×1`, `1×1`). *Test:* `TestLevelGridBordersAndDegenerate`.
6. **SC-6 (FR-3, P-2)** — `LevelGrid` is a pure function of its inputs: identical inputs give an identical
   grid, and it reads no clock/global. *Test:* `TestLevelGridPure`.
7. **SC-7 (FR-4, AC-6)** — `InterpRow` equals the bilinear interpolation of four corner levels, equals each
   corner at that corner, and returns the truncated integer row between them. *Test:* `TestInterpRow`.
8. **SC-8 (FR-5, AC-9, P-5)** — `ShadeChannel` equals `clamp(((chan+tint)·(96−level))/32,0,255)` truncating,
   with the spot values level 64 → chan unchanged, level 0 → `min(chan·3,255)`, level 95 → `chan/32`.
   *Test:* `TestShadeChannelTransform`.
9. **SC-9 (FR-5, AC-10, P-6)** — a uniform-level-64 cell renders every pixel at its unattenuated palette RGB
   (identity row) for tint `(0,0,0)`. *Test:* `TestShadeIdentityRow`.
10. **SC-10 (FR-5, AC-11)** — a non-zero sky tint is added per channel **before** the multiply, then
    clamped. *Test:* `TestShadeTintBeforeMultiply`.
11. **SC-11 (FR-5, AC-12, P-7)** — for a fixed channel and tint, output is monotonically non-increasing in
    level across `[0,95]` and stays in `[0,255]`. *Test:* `TestShadeMonotonic`.
12. **SC-12 (FR-4/FR-5, AC-13)** — every level in the full `0…95` range (a superset of the corpus range
    `[30..70]` at θ = 0.78539815 over the interior window, and of the analytic `[30,89]` band) indexes a
    defined shade with no out-of-range access. *Test:* `TestShadeCorpusRange`. The corpus figure itself is
    developer-run evidence under SC-17, not a unit assertion — it is a statement about game data.
13. **SC-13 (FR-2, AC-7 unit)** — `LightFromFields` passes θ through and takes `ambient`/`range` as the low
    byte of the stored scalars. *Test:* `TestLightFromFields`.
14. **SC-14 (FR-1/FR-5, AC-1/AC-10 at composite scope)** — `CompositeLit` over a uniform-level-64 grid
    renders the same image `Composite` does (identity row); over a **non-flat** grid (a ridge that reaches
    the last row and column) it shades cells non-uniformly and shades the clamped edge cells consistently
    with the edge-vertex levels (exercising the DD6 corner-clamp on real data); and it still counts
    placeholders for absent slots. *Test:* `TestCompositeLit`.
15. **SC-15 (FR-6)** — `pkg/render/terrain` still imports only stdlib; the fail-closed DAG check stays
    green. *Test:* the existing `internal/archtest` live-tree check.
16. **SC-16 (FR-1/FR-2, cmd wiring)** — `terraintool` defaults to shaded, `-unshaded` reproduces the 0004
    image, and `-maplight`/`-theta`/`-ambient`/`-range` reach the compositor. *Test:*
    `cmd/terraintool/main_test.go`.
17. **SC-17 (AC-8, AC-7 corpus, AC-13 corpus, FR-7)** — a developer run against a lawful install shows
    relief with flat ground at ×1.5625, and the ten shipped maps' stored light fields are in range.
    Because this revision changes the arithmetic, the run also **re-measures every derived figure the
    previous revision recorded** — per-map level ranges, the corpus census, the lit-vs-unshaded
    distribution and the flat-cell figures — each quoted with its θ, light bytes, framing and window
    (FR-7), and marks as superseded or unrepeatable anything that cannot be re-measured. The corpus census
    doubles as a **cross-check against `TERR-LIGHT-029`'s own numbers**: an independent implementation of
    the same claim should land on `[30..65]` over the 10 root maps and `[30..70]` over 38 at the stated θ,
    and reproduce the claim's θ-sweep table. *Method:* developer-run (manual), recorded in
    `verification.md`; no game bytes.
18. **SC-18 (FR-3, AC-14)** — a height field varying only with `y` and one varying only with `x`, at equal
    gradient magnitude, produce **different** levels, with each equal to the exact value the spec formula
    gives: the `y` field drives both halves directly, the `x` field moves the level only through the
    lateral shear. Under a perpendicular-axis model the two would be equal, so this discriminates the
    corrected gradient from the superseded one. *Test:* `TestLevelGridRowAxisGradient`.
19. **SC-19 (FR-3, AC-15)** — a `0/127` checkerboard at the default light levels every interior vertex to
    exactly 46, and the test computes what a shear-free reading would give on the same fixture and asserts
    it is **not** 46 — so the assertion fails if the `tan|θ|` shear is dropped or mis-signed. *Test:*
    `TestLevelGridLateralShear`.
20. **SC-20 (FR-3, AC-16)** — height bytes ≥ `0x80` are read as signed: a fixture whose bytes straddle
    `0x80` levels to the value the signed reading gives and not to the unsigned one. *Test:*
    `TestLevelGridSignedHeights`.

## Risks (product)

- **R-1 — the attenuation index is counter-intuitive (higher level = darker), so "unshaded" is not ×1.0.**
  With the daytime intensities flat ground is level 46 = ×1.5625, and the old 0004 render (≈ level 64) is
  ~35 % darker than the game's flat baseline. A reviewer could read the brighter default as a bug.
  *Mitigation:* AC-1/AC-9/AC-10 pin the exact numbers; the spec documents the ×1.5625 baseline and the
  identity row; SC-17's developer evidence reports flat ≈ ×1.5625 against the game.

- **R-2 — the light source is not engine-faithful per map.** `TERR-LIGHT-023` shows the map-stored light
  fields are overwritten before use and cannot be read as the runtime angle. Defaulting to them would be a
  false fidelity claim. *Mitigation:* DD8 defaults to the documented daytime globals and makes the
  map-stored override explicit and opt-in; the spec and `verification.md` state it is a viewer choice.

- **R-3 — CLOSED, and it fired.** The risk was "the exact per-axis neighbour selection is Medium-confidence
  in the research", mitigated by implementing the model the disassembly appeared to show.
  `TERR-LIGHT-028` closes the question at High confidence, and the answer is that the shipped
  implementation was wrong on three counts (span, axes, missing shear). Recorded rather than deleted,
  because it is the load-bearing lesson of this pass: **a Medium-confidence reading of a formula is an
  implementation nothing can verify** — the unit tests were spec-derived and green throughout, and the only
  signal was a real-map ceiling that disagreed with the research census. The successor risk is R-6.

- **R-4 — RETIRED with DD2a.** The interior/border slope-magnitude seam existed only because a two-cell
  interior span met a one-cell border fill across a single `stepH`. Under DD11 every difference spans one
  cell, so no scale discontinuity arises at the ring; there is no seam to look for, and SC-17 no longer
  asks a developer to look for one. What remains true and is carried by R-5 instead: the ring is **ours**,
  the original computes no border vertex, and no fidelity claim attaches to it in either direction.

- **R-6 — every level statistic is conditional on θ, and which θ shipped play uses is unknown.**
  `TERR-LIGHT-030` shows the sun model sweeping θ over `[−π/2, +π/2]` across the daylight hours when the
  day/night cycle is enabled, and storing the literal `0.78539815` when it is off; `TERR-LIGHT-029`
  measures the corpus ceiling travelling 70 → 81 across that sweep. Whether a live session runs with the
  cycle on is **not established** and static observation cannot establish it (research: Medium that the
  literal is what a player sees). *Mitigation:* the default is the cycle-off configuration, labelled as a
  **reference point** and never as what the game runs; FR-7 requires θ beside every recorded figure, and
  the `-theta` override exists precisely so the sweep can be explored. *Not mitigated, by design:* nothing
  here decides the question — it goes back to the research team.

- **R-7 — the `y == 1` lateral arm is emitted semantics, not established intent.** `TERR-LIGHT-028` records
  that the engine computes a θ-sign compare for the backward half and then discards it (a later decrement clobbers
  the flags), so the arm actually taken is selected by the row. We implement what executes (DD12). If the
  original source intended the θ test, our row 1 differs from the intent — but not from the binary.
  *Mitigation:* the effect is confined to row 1 and the research measured it as **nil** on the corpus level
  range (identical ceilings either way); the choice and its bound are documented rather than silently
  resolved in either direction.

- **R-5 — the far edge has no defined shading in the original, so our border ring cannot be verified
  against it (DD6).** `W×H` level vertices only fully bound `(W-1)×(H-1)` cells. The research has now
  decoded what the original does there, and the answer removes the fidelity question rather than answering
  it: the outer vertex ring is never computed and never zeroed, and the far-edge cells source their missing
  corner by raw flat addressing (next row's column 0, or past the allocation). The ring is drawn, but its
  shading is **undefined** — tolerated in the original because gameplay is bounded by a derived 8-cell sim
  border and the camera keeps that band at the extreme edge. *Mitigation:* we clamp the `+1` corner indices
  to the grid edge — the treatment the research names as the safe, intent-matching port choice — so the
  compositor stays total (P-3) and the border is deterministic; SC-14 asserts self-consistency with our
  edge-vertex levels. **Residual risk, honestly bounded:** whether the outermost ring is ever a visible
  pixel in the original is **Medium** (an un-pinned scroll clamp), and what it holds at runtime is
  **Unknown** (static analysis only, the running game was not observed) — so "our border looks different
  from the original's" is neither claimed nor refuted. It affects a one-cell-wide border only; the decoded
  interior is unaffected.
