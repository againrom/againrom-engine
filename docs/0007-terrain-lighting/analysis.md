# Analysis — terrain relief lighting (ROM1)

## Intensity & terrain (declared for the whole work item)

- **Intensity: `spec-first / static`.** Per `SDD/PROFILE.md`, rendering is "other engine work" — a bounded
  delivery whose behavior settles quickly; upfront clarity helps but the contract will not keep evolving.
  No watcher tool exists, so synchronization is static + discipline, labelled as such.
- **Terrain: brownfield for `pkg/render/terrain`'s level grid; greenfield elsewhere.** The first pass built
  the lighting math as new code and was greenfield throughout. **This revision changes shipped
  arithmetic**: `LevelGrid`'s gradient is re-derived from EXP-0031, so that unit is brownfield and its
  behaviour is pinned by the tests that exist before it changes. The shading transform, the row
  interpolation, the lit compositor and 0004's unshaded `Composite` are untouched and stay greenfield. The
  only pre-existing behaviour that moves outside `LevelGrid` is `cmd/terraintool`'s shaded *output values*
  — different numbers through an unchanged contract. Recorded here so the terrain declaration is per
  changed area, not per ticket.

## Source & confidence

All facts below are game-derived by the research team, not by this repo's own reverse-engineering:

- **Submodule pin:** `research/` at `8a406d6` (`againrom-research`, module `rom1research`).
- **Experiments:**
  - `research/experiments/EXP-0023-terrain-lighting` — the relief-lighting model: that terrain is Gouraud
    relief-shaded (not flat-copied), that the brightness grid is runtime-computed (no stored lightmap), the
    per-vertex level byte's structure and all its float constants, and the first corpus census.
  - `research/experiments/EXP-0029-terrain-edge-cells` — the far edge: how the engine sources a cell's four
    corner levels (raw flat row-major addressing into unpadded `W·H` grids), that the per-vertex brightness
    grid's **outer ring is never computed**, and that the outer ring of cells is nonetheless drawn. It
    **amends `TERR-LIGHT-013`** (same claim ID) and retires that claim's "first row/col use a one-sided
    neighbour" wording.
  - `research/experiments/EXP-0025-shading-table` — the shading table the blitters index: its dimensions,
    stride, the **exact per-entry transform**, which level is unattenuated, which globals enter it, the
    one-table-for-all-terrain finding, and the correction that the map-stored light fields are dead on the
    engine's lighting path.
  - **`research/experiments/EXP-0031-terrain-gradient-span`** — the experiment this revision is derived
    from. It establishes the two `Δh` terms of `R0468` by reverse engineering the x87 floating-point
    code (the decompiler drops those operations, which is why the terms stayed Medium through two experiments),
    corrects the corpus level census and states its parameters, and reads the sun model's cycle-off θ as a
    literal. It **amends `TERR-LIGHT-013` and `TERR-LIGHT-018`** and publishes `TERR-LIGHT-028/029/030`.
  - `research/experiments/EXP-0021-terrain-graphics` — the tile-word → graphic mapping (`TERR-IDX-003`),
    consumed already by 0004; this story reuses the palette indices 0004 retains (its T7).
- **Corpus & strength.** 38 maps (10 standalone + 28 campaign). All 38 carry a non-flat type2 height grid
  (σ 7.0…41.1). Applying the exact per-vertex formula to all 38 grids **at θ = 0.78539815 (the engine's
  cycle-off default), `L = 62`, `R = 32`, over the interior window the engine writes, on the EXP-0030
  corrected grid base** yields levels **`[30..70]`** over 859 768 vertices; the 10 root maps alone give
  **`[30..65]`**. No vertex can ever leave the 96-row table: the per-axis clamp confines the level to
  `[30, 89]` for *any* height field, and the floor 30 is a saturation rather than a property of the corpus.
  All 53 shipped `terrain.3d` palettes are byte-identical (0 deviation, measured). The mode-3 table
  transform, its dimensions, stride, truncation, both clamps, the multiplier direction, and which globals
  enter it are single-path findings of the research, stated as claims — no decompiler
  inference; the two `Δh` terms are likewise established by reverse engineering the floating-point code. Falsification passed
  on all four experiments.
- **The far-edge findings' strength (EXP-0029), carried as three different confidences.** The *mechanism*
  is **High**: the flat corner reads (`idx, idx+1, idx+W, idx+W+1`) are single-path instruction reads across
  all four terrain renderers, the unpadded `malloc(W·H)` allocation is read from the allocator and
  corroborated by the corpus (type1 `== 2·W·H`, type2 `== W·H`, 38/38), and the brightness writer's loop
  bounds `{1..W-2}×{1..H-2}` plus the allocator's lack of zeroing are likewise instruction-level. Two things
  are **not** High and are not treated as if they were: whether the outermost ring ever reaches a *visible*
  (non-culled) pixel is **Medium** — it depends on a CMapView scroll clamp the experiment located in the
  draw path but did not reduce to a single instruction; and what the uncomputed ring actually **holds at
  runtime** is **Unknown** — the engine never writes or zeroes it, but a first-touch heap page may still
  arrive demand-zeroed from the OS, and the running game was not observed.

## Claim inventory (what the spec is built on)

| Claim | Statement | Confidence | Used by spec |
|---|---|---|---|
| TERR-IDX-003 | Tile-word → graphic mapping (four agreeing `rom.exe` renderers): `g=(w&0x1fff)>>6`, image `= tiles[g*4 + ((w>>4)&3)]`, sub-cell `= w&0xf` | High | tile pipeline this story lights (Constraints); reused from 0004 |
| TERR-LIGHT-011 | Terrain is Gouraud relief-shaded, not flat-copied: both blitters draw `LUT16[(level<<9 & 0xfffffe00) + srcIndex*2]`, `level` bilinearly interpolated from four corner brightness bytes | High | FR-4 (interpolation), FR-5 (LUT application) |
| TERR-LIGHT-012 | The brightness grid `P+0x18` is runtime-computed, never read from file (builder allocates, lighting routine writes, blitters sample); the `.alm` carries no lightmap | High | FR-3 (grid is computed), Constraints (no format change) |
| TERR-LIGHT-013 (amended by EXP-0029 and EXP-0031) | Per-vertex relief model, exact byte: `stepH=32/cos|θ|`; per half `sᵢ=atan2(Δhᵢ,stepH)`; `axisᵢ=clamp(L−R·sin(π/6−sᵢ),0,95)`; level `= ftol(0.5·(axis₁+axis₂))`. `R=range`, `L=(R>>1)+ambient+0x20`. Constants 32.0, π/6, 95.0, 0.5. Flat day → 46. **Written region: only the strict interior `{1..W-2}×{1..H-2}`** — the earlier "first row/col use a one-sided neighbour" wording is **withdrawn** (EXP-0029): nothing is clamped or substituted, the outer ring is simply not written. **The `Δh` terms are now derived (EXP-0031): two adjacent ONE-CELL differences along the SAME row axis — not two perpendicular axes and not a two-cell central difference** | High (formula + all constants + the Δh terms read from x87 disasm; loop bounds likewise) | FR-3 (formula), AC-1…AC-4, AC-5a, AC-14…AC-16, P-1…P-4 |
| **TERR-LIGHT-028** | **The height-gradient span** (amends `TERR-LIGHT-013`; resolves this story's R-3). `Δh₁ = H[x∓tan\|θ\|, y+1] − H[x,y]` (forward, one row) and `Δh₂ = H[x,y] − H[x∓tan\|θ\|, y−1]` (backward, one row). (a) **no two-cell central difference exists** — the code forms no difference `H[…,y+1] − H[…,y−1]`, and the only `0.5` multiplies the sum of two already-**clamped levels**; (b) **there is no perpendicular axis** — EXP-0023's "two axes" are the forward and backward halves of the *same* row axis, and every `x±1` read is a lateral operand of the `tan\|θ\|` shear *inside* the adjacent row; (c) the azimuth enters twice, as that shear and as `stepH`; (d) heights are read as **signed** bytes, unobservable on the shipped corpus (0/880 704 bytes ≥ `0x80`); (e) the two halves are clamped **independently** before averaging, so the routine equals no single-difference form. The axis-2 lateral column is selected by `y == 1`, not by the θ sign the code computes and discards (a later decrement clobbers the compare's flags); measurable effect nil. Falsified on synthetic fixtures (flat → 46 at any height, sign-correct ramps, a 0/127 checkerboard → 46 only if the shear is right); deleting the shear moves the 38-map ceiling 70→66, so no term is decorative | High (single writer, falsified) | FR-3 (formula), AC-14, AC-15, AC-16; DD11, DD12 |
| **TERR-LIGHT-029** | **Corrected corpus level census, with its parameters** (corrects `TERR-LIGHT-018`'s `30…74`). At θ = 0.78539815, `L=62`, `R=32`, corrected grid base, interior window: **`[30..70]`** over the 38-map corpus (859 768 vertices), **`[30..65]`** over the 10 root maps. The published `30…74` is **stale** — reproduced bit-exactly (range *and* the 880 704 vertex count) only on the superseded EXP-0007 framing, with its maximum inside the 8 record-header bytes that split injected into the height grid. Two facts make a bare ceiling unsafe: (i) it moves with θ — 70 at the default, 72 at midday, up to 81 near the sweep ends, and the sun model sweeps θ over `[−π/2,+π/2]` every game day; (ii) the floor **30** is a **saturation** of `clamp(62 − 32·sin(π/6 − s),0,95)` at `s = −π/3`, not an agreement between models. Analytic corollary: the level is confined to `[30, 89]` for **any** height field | High (arithmetic over the full corpus, 0 invariant violations) / Medium (that a live session runs at this θ — static observation only) | AC-13; every level figure in `verification.md`; R-6 |
| **TERR-LIGHT-030** | **The sun model's cycle-off default θ is a literal, and it is not π/4.** `R1813`'s `L06260 == 0` arm stores `0x3fe921fb4d12d84a` = **0.78539815** (= `3.1415926/4`, *not* `π/4` = `0x3fe921fb54442d18`), paired with ambient `0x0e` and range `0x20` — exactly the `L=62, R=32` daytime configuration. With the cycle enabled, the 6…17 band computes `θ = −π/2 + minutes·0.00218` over 720 minutes, so θ runs `−π/2 → +π/2` across the twelve daylight hours. Because `stepH = 32/cos\|θ\|` **and** the lateral shear `tan\|θ\|` both depend on it, **no per-vertex level statistic is meaningful without its θ** | High (immediates read from the listing) | FR-2 (the default), AC-1; DD13; R-6 |
| ~~TERR-LIGHT-016~~ | ~~Corpus: 38/38 maps carry a non-flat type2 height grid (σ 7.0…41.1)~~ — **WITHDRAWN by the 2026-07-25 confidence sweep, and refuted, not merely stale.** Every published `hMax` is the third byte of that map's own `selectorA` `f32` (Forester `0xBFF62B6D`→246, Islands `0xBFC02B6D`→192) — the identity word the pre-EXP-0030 grid base injected — so the whole column read 192/245/246, contradicting `TERR-LIGHT-028`(d) (0 of 880 704 height bytes ≥ `0x80`). σ is contaminated by the same bytes. Replaced by our own measurement at the corrected base, **10 root maps**: altitudes `0…127`, per-map σ **6.45…27.53** — Kids at 6.45 falls *below* the withdrawn range's floor, exactly as predicted for an 80×80 map. Relief is still meaningful everywhere; nothing in the story's arithmetic depended on the figures | ~~High~~ → withdrawn (figures) / own measurement (the conclusion) | motivation |
| TERR-LIGHT-018 | The table is `[96][256]` u16, stride 512 = `1<<9`; real builder `R1107` via driver `R1368`, terrain `(nLevels=0x60, mode=3, useTint=1)`. 96 = the `[0,95]` clamp of the per-vertex byte. Supersedes EXP-0023's "8-level" wording. **Its embedded census figure `30…74` is corrected by `TERR-LIGHT-029`**; the table-bound conclusion is unaffected and in fact unbreakable — the per-axis clamp bounds the level to `[30,89]`, so no span choice could ever have put a vertex outside the 96 rows | High (single-path instruction reads) / High (table bound) — census figure amended | FR-4 (row select), FR-5, AC-13, table dims |
| TERR-LIGHT-019 | Exact per-entry transform (mode 3), per channel, integer: `out = clamp(((palette_chan + skyTint_chan) × (96 − level)) / 32, 0, 255)` truncating toward zero; then packed to the surface format (RGB565). Linear per channel — no gamma, no cross-channel term. Palette source order BGR (`SPR256-PAL-011`) | High | FR-5 (transform), AC-9, AC-11, AC-12, P-5…P-7 |
| TERR-LIGHT-020 | Level 64 is unattenuated (×1.0); row `L` multiplier `(96−L)/32`; `L=0`→×3.0, `L=95`→×1/32. Higher level = darker. Flat day = level 46 = ×1.5625; shipped art authored dark for it (mean luminance 78.5 at ×1.0 vs 119.2 at ×1.5625) | High (arithmetic) / Medium (the "authored for it" reading, inferred from luminance) | FR-5 baseline, AC-9 (identity row), AC-10, AC-12, P-6 |
| TERR-LIGHT-021 | Only the sky tint (`L10231/491/492`) enters the table, added per channel **before** the multiply, and only under `useTint=1` (terrain passes 1). Daytime band = `(0,0,0)`. The intensity bytes `L05650/498` never enter the table — they set the per-vertex level's `L`/`R` | High | FR-2 (skyTint), FR-5 (tint-before-multiply), AC-11 |
| TERR-LIGHT-022 | One table serves all terrain (driver rebuilds only the first tile slot and hands it to every renderer); sound because all 53 shipped `terrain.3d` palettes are byte-identical | High | FR-5 (one transform per palette), design |
| TERR-EDGE-024 | Far-edge cell corner sourcing = **raw flat row-major addressing**, no clamp/dup/wrap/side-table. All render grids are unpadded `W·H` (`malloc(W·H)`; corpus type1 `==2·W·H`, type2 `==W·H`, 38/38); the four renderers read `grid[idx], grid[idx+1], grid[idx+W], grid[idx+W+1]` (`idx=col+row·W`). A **last-column** cell's `+1` corner therefore reads the **next row's column 0**; a **last-row** cell's `+W` corner reads **past the `W·H` allocation** | High | the spec's "no defined edge shading to reproduce" statement — the *cell* half |
| TERR-EDGE-025 | The per-vertex brightness **outer ring is never computed** (amends `TERR-LIGHT-013`): `R0468(1,1,0,0)` writes only `{1..W-2}×{1..H-2}`, there is **no one-sided fallback and no edge fill**, nothing else on the relight path fills the ring, and the allocator never zeroes the block. So even the in-bounds far-corner reads of the second-to-last cells sample memory the engine never initialised | High (loop bounds, no-fill, no-zeroing) / **Unknown** (what those bytes are at runtime — a first-touch heap page may still arrive demand-zeroed; not observed running) | FR-3 border wording; the far-edge design position — the *vertex* half |
| TERR-EDGE-026 | The outer ring **is drawn**, not padding-never-drawn: no renderer skips or special-cases it, the mesh is built for it plus an over-scan margin, blits are issued, and the draw's only bounds check (`worldRow < H`) proves the edge is reached. The degenerate band is tolerated because gameplay is bounded by the derived 8-cell sim border and the camera. **For a reimplementation the far edge has no defined shading to reproduce — clamp-to-edge is the safe, intent-matching choice** | High (not skipped) / **Medium** (whether the outermost ring reaches a *visible* pixel — bounded by an un-pinned CMapView scroll clamp) | the spec's "no defined edge shading to reproduce" statement (that the ring is nonetheless drawn is what makes the statement load-bearing) |
| TERR-LIGHT-023 | Type-0 light-field offsets at the EXP-0030 corrected framing: payload `+0x08` (f32 angle) → `P+0x20`; ambient `+0x10` → `P+0x1c`, range `+0x14` → `P+0x1d`. **And** `R0468` overwrites all three from the day/night sun globals before reading them; the loader cannot supply the angle as a valid double. So the map-stored light fields are not demonstrably read by the engine | High (offsets + the overwrite) / **Unknown** (whether *any* path reads the stored values) | FR-2 (override provenance), AC-7, the light-source design decision |

Supporting `alm` claims (already implemented in 0003, no change this story):

| Claim | Statement | Used by spec |
|---|---|---|
| ALM-GRID-013 | type2 = Altitudes (height), u8 (0..~246), `W·H` cells row-major — a height field, not illumination | FR-1 (height grid), exposed as `alm.Map.Altitudes` |
| ALM-META-008/009 | type-0 layout: `W@+0x00`, `H@+0x04`, f32 angle `@+0x08`, u32 scalars `@+0x0c/+0x10/+0x14` — the offsets `TERR-LIGHT-023` anchors against | FR-2 override fields, exposed as `alm.Map.Angle` / `Meta.Word10` / `Meta.Word14` |

**Framing note — the gap the previous revision recorded is now closed.** Every type-0 offset in the two
tables above is quoted at the **corrected** container framing — `−8` from the labels EXP-0018/0025 used,
because `typeId`/`selectorA` turned out to be record-header words, not payload. The previous revision of
this file recorded a research-lane consistency gap: `ALM-META-008` carried the amendment but
`TERR-LIGHT-023`'s own prose did not, still reading `+0x10`/`+0x18`/`+0x1C`. At pin `8a406d6` that row
carries the amendment explicitly (re-anchored `W@+0x00 … θ@+0x08 … ambient@+0x10 … range@+0x14`, with the
`P+…` runtime slot displacements unchanged), so the gap is closed at the source rather than annotated here.
They were always the **same file bytes**, and `alm.Map.Angle`/`Meta.Word10`/`Meta.Word14` read them
byte-identically before and after (`-maplight` evidence in `verification.md`, on ten real maps).

## Decoded-vs-open reconciliation

- **The gradient span — decoded, and it invalidates the arithmetic this story shipped.** The first pass
  implemented `Δh` as a two-cell central difference on two perpendicular axes, reading EXP-0023's "per axis
  `i`" as `x` and `y`. `TERR-LIGHT-028` finds **three** separate errors in that reading: the span is **one**
  cell, not two; the two terms are the **forward and backward halves of the same row axis**, not two
  perpendicular axes; and the **lateral `tan|θ|` shear** — omitted entirely by the first pass — is
  load-bearing (deleting it moves the 38-map ceiling 70 → 66). What the first pass *did* get right, and
  what this revision preserves unchanged, is the **structure**: each half is clamped to a level **before**
  the average, and the single `0.5` multiplies two already-clamped levels rather than a height difference.
- **`R-3` is closed, and the answer is that the risk fired.** The story carried "the exact per-axis
  neighbour selection is Medium-confidence" from its first draft, mitigated by "we implement the model the
  disassembly shows". `TERR-LIGHT-028` resolves the question at High confidence and the first reading
  turns out to have been wrong. The correct lesson is not that the risk was mispriced — it was recorded
  honestly and its consequence (the ceiling disagreement) was measured on real maps — but that a
  Medium-confidence *reading* of a formula is an implementation the code cannot be verified against.
- **The interior/border slope-magnitude seam (`R-4`) dissolves.** It existed only because a two-cell
  interior span met a one-cell border fill across the same `stepH`. With the corrected one-cell span there
  is no scale difference between interior and border, so there is no seam to document or to look for.
- **Every derived number this story recorded was measured on the superseded arithmetic.** This is exactly
  the failure mode the research lane recorded in its own `AGENTS.md` off the back of EXP-0031: *a corpus
  statistic can be silently invalidated by a later base correction, and re-auditing claims is not the same
  as re-auditing numbers* — `TERR-LIGHT-018`'s `30…74` survived a whole cycle that way. The corresponding
  discipline here is that `verification.md` re-measures or explicitly supersedes **every** figure, not only
  the ones attached to a corrected claim.
- **θ is not a constant of the format.** `TERR-LIGHT-030` pins the cycle-off default as a literal and shows
  the enabled cycle sweeping θ across `[−π/2, +π/2]` every game day, with the corpus ceiling travelling
  70 → 81. A level range without its θ, framing and window is therefore not a fact. The spec states its θ,
  and the verification quotes θ, framing and window beside every level figure.
- **The whole pipeline is decoded.** The first spec draft was BLOCKED on R-1 (the shading-table
  transform) and raised R-2 (the type-0 intensity offsets). EXP-0025 resolved both: R-1 by deriving the
  mode-3 builder's rule step by step (`TERR-LIGHT-018/019/020/021/022`), R-2 by anchoring the
  type-0 read order against the corpus-pinned `W`/`H` slots (`TERR-LIGHT-023`, re-anchored `−8` by
  EXP-0030). No brightness curve, table layout, or field meaning is invented in the spec — every constant
  is read from `rom.exe`.
- **Superseded wording.** EXP-0023's `TERR-LIGHT-014` called the table "8-level"; EXP-0025 corrects it to
  **96** rows and shows `R1368` is a relight *driver*, not the builder. The spec uses the corrected
  96-row model throughout; the first draft's "8-level" wording is dead.
- **The far edge — decoded, and the answer is that there is nothing to reproduce.** The story shipped with
  the far-edge cell corner mapping open; EXP-0029 closes it, and the finding is negative rather than a
  value we now have to match. Two independent halves:
  *(a)* the original's **cell** corner sourcing is raw flat row-major addressing (`TERR-EDGE-024`), so a
  last-column cell's `+1` corner reads the next row's column 0 (measured on the corpus as an unrelated
  vertex — mean `|Δh|` 8.74, per-map up to ~39, not a clamp) and a last-row cell's `+W` corner reads past
  the allocation entirely; *(b)* the **vertex** brightness the corner read samples does not exist on the
  outer ring at all (`TERR-EDGE-025`) — it is never computed, never filled by anything else on the relight
  path, and never zeroed by the allocator. The ring **is** drawn (`TERR-EDGE-026`), so this is not
  "never-rendered padding"; what is degenerate is the shading, not the drawing.
  **Consequence for us:** the original has **no defined edge shading**, so an edge-shading choice cannot be
  right or wrong against it — and clamp-to-edge is what the research itself names as the safe,
  intent-matching port behaviour. The shipped one-cell clamp therefore stands as a **deliberate, defined
  choice**, not an approximation of a known original behaviour. What is *not* claimed: that the ring is
  visible in the original (**Medium**, un-pinned scroll clamp) or what it holds at runtime (**Unknown**,
  static analysis only).
- **The withdrawn "one-sided border" wording.** EXP-0023's `TERR-LIGHT-013` said the first row/column use a
  one-sided neighbour; EXP-0029 shows instruction-level that they use nothing — the loop never visits them.
  The engine neither clamps nor substitutes at any edge. Our `LevelGrid` still fills the outer vertex ring,
  and that fill remains **our own defined, index-safe fill of a region the original leaves uncomputed**,
  not a transcription of engine behaviour. What the corrected span changes is only *what our fill
  computes*: the region needing a fill is the same strict interior complement (both spans read `x±1` and
  `y±1`), but the fill is now "the same formula with every height index clamped into the grid" rather than
  "a one-sided difference".
- **Light source — an honest gap carried into the spec, not papered over.** `TERR-LIGHT-023` establishes
  that the map-stored light fields are overwritten before use and cannot be read as the runtime angle, at
  **Unknown** confidence for whether *any* path reads them. The engine's own daytime path feeds documented
  sun globals (`θ=±π/4` band, ambient `0x0e`, range `0x20`, tint `(0,0,0)`). The spec therefore *defaults*
  to those globals (engine-faithful for the daytime look) and treats reading the map's stored fields as an
  explicit, labelled **viewer choice**, never a fidelity claim. This is the one place the story cannot be
  fully engine-faithful, and the spec says so.

## Scope notes carried into the spec

The research decodes the full relief-lighting pipeline; it deliberately leaves the following out, and the
spec scopes them out rather than guessing:

- **The day/night cycle and its per-hour dawn/dusk tint schedule** — the sun model's colour bands are read
  but not exhaustively enumerated (Medium confidence on the per-hour schedule); the spec uses the daytime
  tint `(0,0,0)` and a fixed representative angle, and scopes the schedule out.
- **The 8-bpp display path** — a different table (mode 3 is the 16-bpp one); out of scope.
- **The RGB565 packing / per-display-mode quantization** — runtime display detail; the render tier outputs
  8-bit RGBA and states the pre-pack channel value, per `TERR-LIGHT-019`.
- **Unit and sprite lighting** — the same builder, mode 2, a 16-level ramp (`nLevels/2` neutral row), a
  separate path noted but not verified; out of scope.
- **The altitude-driven vertical mesh offset** — a separate geometry feature, not the shading; out of scope.
- **Which θ a live session actually runs at** — whether shipped play leaves the day/night cycle off (and so
  takes the `0.78539815` literal) or enables it (and so sweeps `−π/2 → +π/2` across the daylight hours).
  The research leaves this **open on purpose**: static observation cannot settle it, and it decides which
  row of the θ→level-range table is "the real one". The spec takes the cycle-off default as its **reference
  point** for evidence and says so; it does **not** assert that this is what the game runs. Every level
  figure this story records carries its θ for exactly that reason.
- **Whether the axis-2 `y == 1` branch is what the original source intended.** EXP-0031 reports the
  *emitted* semantics — the θ compare is computed and then discarded because a later decrement overwrites its flags
  — and explicitly declines to claim intent. The effect is confined to row 1 and is not measurable in the
  corpus range. The spec states the rule that executes and labels the intent question as not ours to settle.
- **What the outermost ring holds at runtime, and whether it is ever visible** — Unknown and Medium
  respectively (see above); neither is needed by any FR, and neither may be asserted.
- *(The `Δh` neighbour structure — which neighbour is sampled and over how many cells — was the standing
  open item through EXP-0023 and EXP-0029. It is **closed** by `TERR-LIGHT-028` and is no longer a scope
  note; it is the substance of this revision.)*

When the research team closes any open item (the per-hour tint schedule, which θ a live session runs at,
whether any path reads the stored light fields, or what the uncomputed ring holds), the flow is: repull the
submodule (`git submodule update --remote research`), derive the newly-decoded fact here, then revise
`spec.md` from the earliest affected stage per the SDD revision process.
