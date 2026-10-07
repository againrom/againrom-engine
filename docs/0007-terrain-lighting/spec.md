# Spec — terrain relief lighting

**Status: READY, revised.** This is a **research-directed RED revision** of a landed story. EXP-0031
(`TERR-LIGHT-028/029/030`) derives the height-gradient span the level byte is built from, and it
supersedes the arithmetic the first pass shipped: the gradient is the **adjacent one-cell** difference
taken **twice along the same axis** with a lateral `tan|θ|` shear, not a two-cell central difference on
two perpendicular axes. The level's *structure* — clamp each half to a level, then average — is
unchanged and was already right. No brightness curve is invented anywhere below; every constant is read
from `rom.exe`.

**Provenance basis.** Every game fact here is research-derived. The tile-word split `TERR-IDX-003` is
0004's; the per-vertex relief model is `TERR-LIGHT-011…013`/`016` (EXP-0023), with `TERR-LIGHT-013`'s
edge wording amended by EXP-0029 (`TERR-EDGE-025`) and its **`Δh` terms derived by EXP-0031**
(`TERR-LIGHT-028`); the corpus level census and its parameters are `TERR-LIGHT-029`; the engine's
cycle-off sun angle is `TERR-LIGHT-030`; the shading table, its dimensions, exact transform, neutral
level, and which globals enter it are `TERR-LIGHT-018…022` (EXP-0025); the type-0 light-field offsets
and the finding that the engine overwrites the map-stored fields before use are `TERR-LIGHT-023`
(EXP-0025, re-anchored `−8` by EXP-0030). All formula constants and both `Δh` terms are read from the
`rom.exe` x87 disassembly; the per-entry table transform is read instruction-by-instruction from the
mode-3 builder. No reference-port lighting logic is used. Brownfield for the level grid (shipped
arithmetic changes), greenfield elsewhere. Submodule pin: `research/` at `8a406d6`.

## Problem / goal

0004/0005/0006 blit terrain at full palette brightness, so our render is flat: the original shades every
cell from a runtime-computed relief map and grades it through a per-level palette-attenuation table. This
story computes that relief and applies it — end to end — so terrain renders with the game's slope shading
instead of flat palette colour.

## What the research establishes (decoded, High confidence)

### The relief level (EXP-0023; edge wording amended by EXP-0029; gradient derived by EXP-0031)

**Terrain is Gouraud relief-shaded, not flat-copied (`TERR-LIGHT-011`).** Both terrain blitters draw each
pixel as `LUT16[((level<<9) & 0xfffffe00) + srcIndex*2]`, where `level` is **bilinearly interpolated from
the four corner brightness bytes** of the cell and `srcIndex` is the source pixel's **palette index**.

**The brightness grid is computed, never stored (`TERR-LIGHT-012`).** The landscape builder allocates a
`W·H` byte grid (`P+0x18`) that is never read from the file; the lighting routine writes it and the
blitters sample it. The `.alm` carries no lightmap.

**The per-vertex level, exactly (`TERR-LIGHT-013`, `Δh` terms per `TERR-LIGHT-028`).** For each interior
vertex `(x,y)`, with `θ` the sun angle and `H` the height field:

```
tanT    = tan|θ|                                      heights read as SIGNED bytes
stepH   = 32.0 / cos|θ|
Δh₁     = H[x ∓ tanT, y+1] − H[x, y]                  FORWARD,  one row   (∓ = −tanT if θ ≥ 0, +tanT if θ < 0)
Δh₂     = H[x, y] − H[x ∓ tanT, y−1]                  BACKWARD, one row
slopeᵢ  = atan2(Δhᵢ, stepH)
axisᵢ   = clamp(L − R·sin(π/6 − slopeᵢ), 0, 95)       per half, BEFORE the average
level   = ftol(0.5·(axis₁ + axis₂))                   truncating, not rounding
```

`H[x ∓ tanT, y]` is the height at a fractional column of row `y`, formed from the vertex's own column and
its single lateral neighbour: `H[x,y] − (H[x,y] − H[x−1,y])·tanT` when `θ ≥ 0`, and
`H[x,y] + (H[x+1,y] − H[x,y])·tanT` when `θ < 0`.

Three properties of this gradient are load-bearing and were **not** what the first pass implemented:

- **Both differences span exactly one cell**, and they are the **forward and backward halves of the same
  (row) axis**. There is **no two-cell central difference** and **no perpendicular `x` gradient**: every
  `x±1` read is a lateral operand of the `tanT` shear *inside* row `y+1` or `y−1`. A port using a central
  difference over-contrasts, roughly doubling every gradient.
- **The azimuth enters twice** — as the lateral shear `tanT` (the sample walks along the sun azimuth) and
  as `stepH` (which lengthens the baseline to match). Deleting the shear moves the corpus level ceiling
  from 70 to 66, so it is not decorative.
- **Each half is clamped to a level independently, before the average.** The single `0.5` multiplies two
  already-clamped **levels**, never a height difference, so the model equals no single-difference form
  once either half clamps.

Heights are read as **signed** bytes (a sign-extending read): a byte ≥ `0x80` counts as negative. That is
unobservable on the shipped corpus (0 of 880 704 height bytes reach `0x80`) but it is the decoded
behaviour. The lateral column of the backward half is selected by **`y == 1`**, not by the sign of `θ`:
the engine computes a `θ` compare and then discards it (a later decrement overwrites its flags), so the executed
semantics branch on the row. Its effect is confined to row 1 and is not measurable in the corpus range.

Light-intensity inputs: `R = range` (the directional swing) and `L = (range >> 1) + ambient + 0x20` (the
base level). Constants, IEEE-754 doubles read from the disasm: `32.0` (cell pitch), `π/6 ≈ 0.5236` (base
incidence), `95.0` (per-half clamp max), `0.5` (final scale). With the documented daytime intensities
(`ambient = 0x0e`, `range = 0x20`) a **flat** vertex sits at `L − R·sin(π/6) = 62 − 16 = 46`, **at any θ**
(a flat field gives `Δh = 0` on both halves).

**The original computes the strict interior only — the outer vertex ring is never computed.** The
brightness writer visits exactly `{1..W-2}×{1..H-2}`; it does **not** fall back to a one-sided neighbour, a
clamped index, a duplicated vertex or a wrap on row 0, row `H-1`, column 0 or column `W-1`, and nothing
else fills that ring afterwards. (The earlier reading carried in this spec — that the first row and column
use a one-sided neighbour — is **withdrawn**: the branch it rested on selects a column direction, not a
border case.) Consequently the original's outer vertex ring holds no computed level at all, and its
far-edge *cells* — which the original does draw — source their missing `+1`/`+W` corner by raw flat
addressing into the same grid, landing on an unrelated vertex (the next row's column 0) or past the grid
allocation. **So the original defines no edge shading for a reimplementation to reproduce**; a defined,
index-safe edge treatment is a choice this project makes deliberately, and no fidelity claim attaches to it
in either direction. The decoded interior is untouched by any of this.

**Relief matters on every map (own measurement).** Over the **10 root maps, at the corrected `.alm`
grid framing**, every type2 height grid is non-flat: altitudes span `0…127` (per-map σ **6.45…27.53**,
smallest on Kids), so relief lighting is never a degenerate no-op. `TERR-LIGHT-016`'s `σ 7.0…41.1`
over 38 maps is **withdrawn and does not back this**: its `hMax` column was the third byte of each
map's own `selectorA` `f32` — the identity word the pre-EXP-0030 base injected — and the same
contamination inflated σ, which is why Kids now measures below that range's floor. The conclusion
survives the correction; the figures did not.

**The corpus level range, with its parameters (`TERR-LIGHT-029`).** Applying the formula above to all 38
corpus height grids **at θ = 0.78539815, `L = 62`, `R = 32`, over the interior window, at the corrected
`.alm` grid framing** yields levels **`[30..70]`** (859 768 vertices); the 10 root maps alone yield
**`[30..65]`**. A level range is only a fact when quoted with those four parameters: the **ceiling moves
with θ** (72 at midday, up to 81 near the ends of the day's sweep), and the earlier `30…74` figure was
measured on the superseded grid base by an approximate rule. The floor **30** is a saturation of
`clamp(62 − 32·sin(π/6 − s), 0, 95)` at `s = −π/3`, not a property of the corpus. The bound that does not
depend on any of this: the per-half clamp confines the level to **`[30, 89]` for any height field**, so no
vertex can ever fall outside the 96-row table below.

### The shading table (EXP-0025, the previously-blocked half)

**The table is `[96][256]` u16, stride 512 (`TERR-LIGHT-018`).** The relight *driver* `R1368`
(mis-named the builder in the first draft) calls the real builder `R1107` for terrain with
`(nLevels = 0x60 = 96, mode = 3, useTint = 1)`, which allocates `malloc(nLevels << 9)` = **96 rows × 256
entries × u16 = 49 152 B, row stride 512** — exactly the `level<<9` the blitters add. 96 is not arbitrary:
it is the `[0,95]` clamp of the per-vertex byte. (The first draft's "8-level" wording is superseded.)

**The exact per-entry transform (`TERR-LIGHT-019`).** Per row `level` and palette entry, per channel, in
integers:

```
out_chan = clamp( ((palette_chan + skyTint_chan) × (96 − level)) / 32, 0, 255 )     truncating toward 0
```

then packed to the display surface's pixel format `(R>>(8−rBits))<<rShift | …` (shipped = RGB565). Linear
per channel — **no gamma, no cross-channel term.** The palette source order is the BMP RGBQUAD `[B,G,R,x]`
(`SPR256-PAL-011`).

**Level 64 is unattenuated; the byte is an attenuation index (`TERR-LIGHT-020`).** Row `L` carries
multiplier `(96 − L)/32`: `L=0` → ×3.0, **`L=64` → ×1.0**, `L=95` → ×1/32. Higher level = darker. With the
daytime intensities the flat-ground level is 46 → **×1.5625**, i.e. flat daytime terrain is deliberately
overdriven; the shipped art is authored dark for it (pixel-weighted mean luminance 78.5/255 at ×1.0 vs
119.2 at ×1.5625). **Consequence for us:** "unshaded" ≡ level 46, *not* ×1.0 — the current 0004 render
(raw palette ≈ level 64) is ~35 % darker than the game's flat-terrain baseline.

**Only the sky tint enters the table (`TERR-LIGHT-021`).** `L10231/491/492` (sky RGB) is added per
channel **before** the multiply, and only under `useTint=1` (terrain passes 1). In the daytime band it is
`(0,0,0)`. The intensity bytes `L05650/498` never enter the table — they set the per-vertex level's
`L`/`R`.

**One table serves all terrain (`TERR-LIGHT-022`).** The driver rebuilds only the first tile slot and
hands that one table to every renderer — sound because all 53 shipped `terrain.3d` palettes are
byte-identical (0 deviation, measured).

### Light source: the map-stored fields are dead on the engine's path (`TERR-LIGHT-023`)

The type-0 light fields land in the lighting slots — payload `+0x08` (f32 angle) → `P+0x20`; the intensity
scalars **`+0x10`/`+0x14`** → `P+0x1c`/`P+0x1d` (this resolves R-2: the offsets are `+0x10/+0x14`, **not**
`+0x0c/+0x10`; `+0x0c` → `P+0x2c`, unread). *(All type-0 offsets here are payload-relative at the EXP-0030
corrected framing — `−8` from the labels `TERR-LIGHT-023`'s prose still uses. Same bytes, same fields.)* **But** `R0468` overwrites all three from the day/night
sun globals in its first ten instructions, before reading them, and the loader is structurally incapable
of supplying the angle as a valid double. So the map-stored light fields are **not demonstrably read** by
the engine (confidence: Unknown whether *any* path reads them). A viewer therefore has no engine-faithful
per-map angle to reproduce — it must pick a light source, and using the map's stored fields is an explicit
viewer choice, not a fidelity claim.

## Design decision — light parameters (both, selectable)

The viewer defaults to the **engine's own cycle-off sun globals** — the values `R1813` writes when
the day/night cycle is switched off — and MAY override them:

- **Default:** `θ = 0.78539815`, `ambient = 0x0e`, `range = 0x20`, `skyTint = (0,0,0)`.
  Every map is lit identically; flat ground → level 46 (×1.5625). The angle is the **literal double**
  `0x3fe921fb4d12d84a` the engine stores (`TERR-LIGHT-030`), which is `3.1415926/4` and is **not** `π/4`
  (`0x3fe921fb54442d18`) — they differ from the eighth decimal, and both `stepH` and the lateral shear
  depend on the value, so the constant is transcribed rather than recomputed.
- **What this default is and is not.** With the cycle *enabled*, θ sweeps `−π/2 → +π/2` across the twelve
  daylight hours, so θ is not a constant of the format. **Which θ a live session runs at is an open
  research question** that static observation cannot settle. This spec takes the cycle-off default as the
  **reference point** for its evidence and states it wherever a level figure appears; it does **not** claim
  that this is the angle shipped play uses.
- **Override (optional):** the caller MAY supply `θ`, `ambient`, `range` from the map's stored `+0x08` /
  `+0x10` / `+0x14` (read directly as their file types via `alm`) or from explicit CLI flags, for
  exploration. The spec states plainly that per `TERR-LIGHT-023` this is a viewer choice, not what the
  engine does.
- **Sky tint** stays `(0,0,0)`; grading it is tied to the unimplemented dawn/dusk schedule (out of scope).

## Functional requirements

- **FR-1 (height plumbing)** The render tier MUST accept a map's `W×H` height grid without depending on the
  map format, in the same shape `Grid` already uses for tile words.
- **FR-2 (light parameters)** The render tier MUST accept a sun angle `θ` (radians) and `ambient`/`range`
  intensity bytes, **defaulting to the engine's cycle-off daytime values** (`θ = 0.78539815` — the literal
  `0x3fe921fb4d12d84a`, not `π/4`; `ambient = 0x0e`; `range = 0x20`). It MUST also accept these as caller
  overrides (map-stored fields or flags). The sky tint is `(0,0,0)`.
- **FR-3 (brightness grid)** The render tier MUST compute the `W×H` per-vertex level grid by the exact
  `TERR-LIGHT-013`/`TERR-LIGHT-028` formula — the **forward and backward one-cell differences along the row
  axis**, each laterally sheared by `tan|θ|` within its own row, each clamped to a level **before** the
  average, with signed height reads — as a pure total function of `(heights, W, H, θ, ambient, range)`:
  no clock, no IO, no global state, `[0,95]` clamp per half, **truncating** final conversion. The formula's
  own computed region is the strict interior `W-2 × H-2` (it reads `x±1` and `y±1`); because the original
  leaves the outer vertex ring uncomputed, the render tier MUST still fill that ring **deterministically
  and in range**, by applying the same formula with **every height index clamped into `[0,W-1]×[0,H-1]`**,
  so that the grid is total and index-safe for any `W,H ≥ 1`. That border fill is this project's defined
  choice for a region the original defines no value for, not a reproduction of engine behaviour.
- **FR-4 (interpolation)** The render tier MUST expose the per-pixel level as the bilinear interpolation of
  a cell's four corner levels; application selects the integer table row by truncating that interpolant
  (matching the blit's `& 0xfffffe00` on its fixed-point accumulator).
- **FR-5 (application)** For each terrain pixel the render tier MUST produce, per channel, the colour
  `clamp( ((srcPalette_chan + skyTint_chan) × (96 − level)) / 32, 0, 255 )` (integer, truncating toward
  zero), where `srcPalette_chan` is the source pixel's tile-palette RGB (0004 retains palette indices) and
  `level` is the truncated per-pixel interpolant of FR-4. Output is full 8-bit RGBA; the RGB565 packing the
  game applies for its 16-bpp surface is display-specific and **out of scope** (FR states the pre-pack
  channel value). The unshaded 0004 path remains available for comparison but is no longer the default.
- **FR-6 (purity / DAG)** All lighting arithmetic lives in `pkg/render/terrain` (stdlib only). Floats are
  permitted in the level computation (the game's model is FP; the result is a byte); the table transform is
  integer. No float is in any control flow that decides which graphic is drawn.
- **FR-7 (level statistics carry their parameters)** Any per-vertex level range, census or distribution
  this story records MUST be stated together with the **θ, the light bytes, the `.alm` grid framing and the
  vertex window** it was measured over. A bare level range is not a result: the ceiling is a function of θ,
  and the previously published corpus census was invalidated by a framing correction without anyone
  noticing. A figure that cannot be re-measured MUST be restated as unrepeatable, never reproduced from a
  previous revision.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a flat height grid at the default light | levelled | every interior vertex is exactly **46**, and the default is the engine's cycle-off configuration — θ the literal `0.78539815` (not `π/4`), ambient `0x0e`, range `0x20`, tint `(0,0,0)` |
| AC-2 | unit | a flat grid with a given ambient/range | levelled | every interior vertex is `ftol(L − R/2)` for that `L`, `R` |
| AC-3 | unit | a height grid with a ridge running **across the rows** (varying with `y`) | levelled | up-slope vertices exceed the flat value and down-slope vertices fall below it, each at the exact level the formula gives |
| AC-4 | unit | deltas driving a half past the clamp both ways | levelled | each half saturates at `0` and `95` and the level never leaves `[0,95]` |
| AC-5 | — | *(retired — its "border vertices use a one-sided neighbour" contract belonged to the superseded central-difference model; replaced by AC-5a. ID not reused.)* | | |
| AC-5a | unit | a `W×H` grid including its outer vertex ring | levelled | ring vertices are produced by the same formula with every height index clamped into the grid (the forward/backward half degenerates to a lateral difference on the top/bottom edge; the shear operand collapses on the left/right edge); no index out of range for any `W,H ≥ 1`, and a flat grid is still uniformly 46 including the ring |
| AC-6 | unit | four corner levels and a pixel position | interpolated | the value is the bilinear interpolation, equal to the corners at the four corners; the applied row is its truncation |
| AC-7 | unit | the ten shipped maps' stored `+0x08`/`+0x10`/`+0x14` | parsed | every angle lies in `[−π/2,π/2]` and both intensity scalars are byte-range (used only as an optional override) |
| AC-8 | manual | a real GOG map rendered with shading (default daytime light) | rendered | terrain shows slope relief; flat areas sit at ×1.5625 and steep-away faces darken; reported against the game; recorded in `verification.md` — no game bytes committed |
| AC-9 | unit | a level and a palette RGB, sky tint `(0,0,0)` | shaded | the output channel equals `clamp((chan × (96−level))/32, 0, 255)` truncating — e.g. **level 64 → chan unchanged**, level 0 → `min(chan×3,255)`, level 95 → `chan/32` |
| AC-10 | unit | a uniform-level-64 cell over any palette | shaded | every pixel renders at its unattenuated palette RGB (identity row) |
| AC-11 | unit | a non-zero sky tint | shaded | the tint is added per channel **before** the multiply, then clamped (`(chan+tint)·(96−level)/32`) |
| AC-12 | unit | levels spanning `[0,95]` for a fixed channel | shaded | output is monotonically non-increasing in level (higher level = darker) and clamped to `[0,255]` |
| AC-13 | unit | levels across the whole `[0,95]` range, and the corpus range `[30..70]` (θ = 0.78539815, `L=62`, `R=32`, corrected framing, interior window) | indexed | every level indexes inside the 96-row table; none is out of range — and the per-half clamp confines the level to `[30,89]` for any height field, so no span choice could produce one that does |
| AC-14 | unit | two height fields of equal gradient magnitude, one varying only with `y` and one only with `x`, at the default light | levelled | the `y`-varying field drives the gradient directly (both halves take the one-cell row difference) while the `x`-varying field moves the level only through the lateral shear, producing a **different** level — pinning that the gradient axis is the row axis and that no perpendicular `x` difference is formed |
| AC-15 | unit | a `0/127` checkerboard height field at the default light | levelled | every interior vertex is exactly **46**, because the `tan|θ| ≈ 1` lateral shear samples the diagonal neighbour, which shares the vertex's parity; the same fixture without the shear would not level to 46 |
| AC-16 | unit | a height field containing bytes ≥ `0x80` | levelled | those bytes are read as **signed** (`0x80` is −128), matching the engine's sign-extending read; unobservable on the shipped corpus but part of the decoded contract |

## Derived properties

- **P-1** The level is in `[0,95]` for every input, including hostile heights and any θ. (At the daytime
  `L=62, R=32` the reachable band is in fact `[30,89]` — the outer clamps never bind.)
- **P-2** The level grid is a pure function of its inputs — no clock or global state read.
- **P-3** The computation never reads outside the height grid for any `W,H ≥ 1`.
- **P-4** A flat grid yields a constant grid (uniform shading on relief-free maps).
- **P-5** The shaded output is a pure function of `(palette, skyTint, level)`; the same inputs give the
  same colour.
- **P-6** Level 64 is the identity row (unattenuated) for tint `(0,0,0)`; per-channel output equals the
  palette channel.
- **P-7** For a fixed channel and tint, output is monotonically non-increasing in level and stays in
  `[0,255]`.

## Constraints

- Builds on 0004's tile pipeline (which retains palette indices as of its T7 — required to index the
  attenuation) and the 0005 viewer / 0006 water path (water lands underneath the shading).
- No `pkg/formats` change: `alm` already exposes the height grid, the `+0x08` angle and the `+0x10/+0x14`
  scalars.
- `pkg/render/terrain` stays stdlib-only. Output is 8-bit RGBA; RGB565 quantisation is out of scope.

## Out of scope

- The day/night cycle and its per-hour dawn/dusk **tint schedule**; the viewer uses the daytime tint
  `(0,0,0)` and the engine's cycle-off angle.
- **Establishing which θ shipped play runs at.** Static observation cannot settle it (`TERR-LIGHT-030`);
  the cycle-off default is used as a reference point and labelled as one, never asserted as the game's.
- The **8-bpp** display path (a different table; mode 3 is the 16-bpp one) and the **RGB565** packing /
  per-display-mode quantisation — the render tier outputs 8-bit RGBA.
- Unit and sprite lighting (same builder, mode 2, 16-level ramp — a separate path).
- The altitude-driven vertical mesh offset.
- Water animation (0006), which lands underneath whatever this story does to the pixels.

## Verification mapping

AC-1…AC-4, AC-5a, AC-7, AC-14…AC-16 + P-1…P-4: unit tests over synthetic height grids and light
parameters, plus the shipped maps' stored scalars read as evidence. AC-9…AC-13 + P-5…P-7: unit tests over
synthetic palettes/levels/tints exercising the exact transform, the identity row, tint-before-multiply,
monotonicity, and the level range. AC-8: a developer-run shaded render of a real GOG map, evidence (relief
visible, flat ≈ ×1.5625) recorded in `verification.md` — no game bytes committed. FR-7 binds every figure
recorded there, including a re-measurement of the corpus census against `TERR-LIGHT-029`'s own numbers.
