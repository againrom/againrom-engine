# Spec — height-displaced terrain geometry (ROM1)

Terrain cells stop being axis-aligned squares on a fixed lattice. Each becomes a quad whose four
corners are displaced vertically by the map's altitudes, and a cell whose corners are not all at one
altitude is drawn as 32 independent vertical spans. This projection becomes the **default** geometry
for every terrain image the project produces; the flat raster survives as an explicit diagnostic.
Brownfield revision of the base renderer.

## Problem / current behavior

The compositor places each cell's resolved 32x32 sub-cell at `(col*32*scale, row*32*scale)` and
replicates every source pixel `scale x scale`. The image is always `W*32*scale` by `H*32*scale`,
every pixel of it is covered and opaque, and a cell's pixels depend on that cell alone. Altitudes
reach only the relief-shading level grid, never the geometry. Tile-word resolution, the water phase, the dirt composite, the placeholder fill and its
count, the level grid, the shading transform, atomic rejection and the tool's summary line are
existing behavior.

## Projection and raster contract

Coordinates here are **native** pixels, before scale. A cell is 32x32 source pixels; a map is
`W x H` cells with a `W*H` row-major altitude grid whose entries are **signed** 8-bit values.

**Vertices.** An altitude is a vertex value indexed by cell coordinates. A cell `(c,r)` has corners
`(c,r)`, `(c+1,r)`, `(c,r+1)`, `(c+1,r+1)`; an index outside the grid is clamped into it:

```
h(c,r) = altitude[ min(r,H-1)*W + min(c,W-1) ]      c in [0,W], r in [0,H]
V(c,r) = r*32 - h(c,r)
```

A larger altitude yields a smaller `V`, that is, moves the pixel **up** the image. Displacement is
vertical only: a cell's destination columns are always `col*32 + 0 .. col*32 + 31`.

**Flat or sloped.** A cell is flat (a *flat cell*, not the flat raster of FR-1) exactly when its four
corner altitudes are **equal**, not by any threshold on their spread. Write `yTL = V(c,r)`, `yTR = V(c+1,r)`, `yBL = V(c,r+1)`,
`yBR = V(c+1,r+1)`.

**A flat cell** is the axis-aligned rectangle `[yTL, yTL+32)`: source pixel `(i,j)` lands at
`(col*32 + i, yTL + j)`.

**A sloped cell** is drawn column by column. For column `i` in `[0,31]`, `top(i)` walks from `yTL`
toward `yTR` and `bot(i)` walks from `yBL` toward `yBR` by the step table below; the span height is
`S(i) = bot(i) - top(i)`.

- `S(i) <= 0` — the column is **not drawn at all**: no pixel, no clamp, no fallback colour.
- otherwise destination rows `top(i) .. bot(i)-1` are drawn — **top inclusive, bottom exclusive** —
  and row `j` of the span takes source pixel `(i, srcRow(j))`:

```
srcStep   = (32 << 16) / S(i)               integer division
srcRow(j) = (j * srcStep) >> 16             floor;  j = 0 .. S(i)-1
```

A flat cell is the degenerate case of the same rule (`S = 32`, `srcStep = 0x10000`, `srcRow(j) = j`).

**The step table.** `T` is built once, independent of any map. For every step count `d >= 1`:

```
s = 0x200000 / (d + 1)                      integer division
acc = 0x8000
for k = 0 .. d:   acc += s ;   T[d][k] = acc >> 16
```

An edge running from a near vertex `yN` to a far vertex `yF` has `d = |yF - yN|`, a step direction
`dir = sign(yF - yN)`, and at column `i`:

```
forward:    edge(i) = yN + dir * #{ k in [0,d) : T[d][k]      <= i }
mirrored:   edge(i) = yN + dir * #{ k in [0,d) : 32 - T[d][k] <= i }
```

The **top** edge walks forward when `yTL < yTR` and mirrored otherwise; the **bottom** edge walks
forward when `yBL >= yBR` and mirrored otherwise. `d = 0` leaves an edge constant, and both counts
are capped at `d`, so an edge that reaches its far vertex stays there.

A cell's bottom edge and the cell below's top edge are the same line walked in opposite
orientations; at `d = 63` and `d = 127` they differ and the lower cell repaints one row of the
upper. That overdraw MUST NOT be smoothed away.

**Shading.** Only the domain the per-pixel relief level is interpolated over changes. Column `i`
takes a top and a bottom level by interpolating the cell's two top and two bottom corner levels
across the 32 columns, as today; row `j` of that column then interpolates between them over `S(i)`
rows on a sloped cell and over 32 rows on a flat one, truncating. Placeholders stay unlit.

**Ownership.** Cells are painted in row-major order — rows ascending, then columns ascending — and
a later cell owns every pixel two cells cover. A pixel no drawn column covers is **transparent**.

**Canvas and scale.** Let `minV` and `maxV` be the least and greatest `V(c,r)` over all
`(W+1)*(H+1)` vertices. The native canvas is `[0, W*32) x [minV, maxV)`, translated so `minV`
becomes output row 0 — `maxV` is a bottom edge, not a drawn row. At integer scale `s >= 1` every
native pixel is replicated `s x s`:

```
outputWidth  = W * 32 * s
outputHeight = (maxV - minV) * s
```

## Functional requirements

- **FR-1 (what holds, what changes)** — Composition MUST keep every rule named above as existing
  behavior, and MUST keep the output width `W*32*scale`. The whole-image flat raster MUST stay reachable by an
  explicit selection and MUST then yield pixels, dimensions and counts identical to the pre-change
  render. Under the new default the PNG bytes change for every map whose altitudes are not all
  equal.
- **FR-2 (projection)** — Composition MUST project every vertex by `V(c,r)` above, reading altitudes
  as signed 8-bit and clamping an out-of-grid index into the grid.
- **FR-3 (selector)** — A cell MUST take the flat path exactly when its four corner altitudes are
  equal, the sloped path otherwise.
- **FR-4 (flat cell)** — A flat cell MUST be drawn as the axis-aligned 32x32 rectangle at its
  projected top edge.
- **FR-5 (sloped cell)** — A sloped cell MUST be drawn as 32 per-column spans with the inclusive top,
  exclusive bottom, the 16.16 source resample, and the untouched column for a non-positive span.
- **FR-6 (edges)** — Both edges MUST come from the step table by the recipe and walks above, for any
  step count the altitudes produce.
- **FR-7 (shading domain)** — The per-pixel level MUST be interpolated vertically over the drawn span
  on a sloped cell and over 32 rows on a flat one.
- **FR-8 (ownership)** — Cells MUST be painted row-major with the later cell owning a shared pixel,
  and every pixel no drawn column covers MUST be transparent.
- **FR-9 (canvas)** — The image MUST span exactly `[minV, maxV)` vertically, translated to row 0,
  and every native pixel MUST be replicated `scale x scale`.
- **FR-10 (rejection and totality)** — Composition MUST reject atomically, returning no image: a
  missing tileset, a non-positive dimension, a tile or altitude grid whose length is not `W*H`, a
  scale below 1, or an output beyond the pixel budget. It MUST be total over every signed
  altitude grid — including deltas of 128 or more between adjacent vertices — and MUST NOT panic.
- **FR-11 (markers)** — Both diagnostic overlays MUST stay independently available on either
  geometry, MUST keep drawing on the un-displaced cell lattice shifted by the same vertical origin
  as the terrain, and MUST keep dropping an anchor outside the map.
- **FR-12 (report)** — The headless summary MUST name the geometry used and the native vertical
  origin alongside the dimensions and counts it already reports.

## Acceptance criteria (synthetic data, no game assets)

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a map whose altitudes are all equal — zero, a positive value, a negative one | composed at several scales | pixels and dimensions are byte-identical to the flat render of the same map |
| AC-2 | unit | varied altitudes and a missing strip slot | composed with the flat raster explicitly selected | pixels, dimensions and placeholder count match the pre-change flat render |
| AC-3 | unit | a grid containing `0x80` (-128) and `0x7f`, with cells on the far edge | projected | every vertex, the canvas bounds, the origin and both output dimensions equal the formulas |
| AC-4 | unit | cells whose four corners are equal, and cells differing in one corner | composed | the first are axis-aligned 32x32 rectangles at their projected top, the second per-column spans |
| AC-5 | unit | the step table for every `d` in `1..127` | built | every row is non-decreasing and ends in 32; the two walks agree at every `(d,i)` except `d = 63` and `d = 127`, where the mirrored one is one further along at all 32 columns |
| AC-6 | unit | a sub-cell with a distinct colour per row and column, on a sloped cell with distinct corner levels | composed at native scale | each pixel's source column is `i` and its source row `srcRow(j)`; the top row is drawn, the bottom is not; the level ramps over the span, not over 32 rows |
| AC-7 | unit | corner altitudes that collapse and that invert one column's span | composed | that column is untouched and stays transparent; its neighbours are unaffected |
| AC-8 | unit | two cells whose quads overlap vertically | composed | the later cell in row-major order owns every shared pixel |
| AC-9 | unit | adjacent altitudes differing by 128 or more, e.g. -128 beside 127 | composed | it completes without panic and those edges follow the same recipe at that step count |
| AC-10 | unit | in turn: no tileset; a non-positive dimension; a tile grid and an altitude grid of the wrong length; a scale of 0; an output past the budget | composed | each returns an explanatory error and no image; none panics |
| AC-11 | unit | both overlays over a projected render, plus an anchor outside the map | composed | markers sit on the un-displaced lattice shifted by the vertical origin, both compose in the existing order, and the out-of-map anchor draws nothing |
| AC-12 | CLI | a synthetic map and tileset, with and without the flat selection | rendered through the tool's argument path | the summary names the geometry and the native vertical origin, and its dimensions equal the written image's |
| AC-13 | manual | a lawful install and the shipped maps, at representative scales | rendered by the developer tool | every map completes without panic, relief is visible, and the PNG hashes the earlier terrain stories recorded are re-taken at the new default and superseded |

Error cases: AC-10 (rejections) and AC-7 (a span the contract drops).

## Derived properties

- **P-1** (invariant) For any altitude grid, every drawn pixel samples a source column and row in
  `[0,31]` — never outside the 32x32 sub-cell.
- **P-2** (invariant) Every drawn pixel lies inside the image: `x` in `[0, outputWidth)`, `y` in
  `[0, outputHeight)`.
- **P-3** (invariant) For any map whose altitudes are all equal, the projected result equals the flat
  result at every scale.
- **P-4** (negative-invariant) For any input FR-10 rejects, no image is returned or retained and no
  caller-supplied slice is modified.
- **P-5** (negative-invariant) For any column whose span is non-positive, no pixel of that column is
  written: it never borrows a neighbour's pixel nor falls back to a fill colour.
- **P-6** (completeness) Composition is total: for every accepted argument set and every signed
  altitude grid it returns an image or an error, never a panic.

## I/O examples

```text
terraintool render -assets <dir> -map <file.alm> -out out.png -scale 2
terrain: 256x256 cells (65536), 16384x<h> px at scale 2, tile slots 53/128,
placeholder cells 0, shaded (...), geometry projected, y origin <signed integer>

terraintool render -assets <dir> -map <file.alm> -out flat.png -scale 2 -flat
# ..., geometry flat, y origin 0
```

The pixel budget is the existing cap on `outputWidth * outputHeight`; equality is allowed, and an
over-budget error recommends a smaller scale. Validation completes before the output path is opened.

## Constraints and alternatives

| Choice | Observable trade-off | Decision |
|---|---|---|
| Flat default, projection as an opt-in mode | every recorded output stays byte-stable, but the default image stays a raster the engine does not contain and is wrong on most cells | rejected |
| Projection as the default, flat raster as an explicit diagnostic | a one-time change to every terrain PNG and to the hashes earlier stories recorded; the default matches the engine and the flat path stays available for comparison | selected |
| A step count past the last row the original's table holds: reject, clamp, or extend the recipe | rejecting refuses input the format permits; clamping silently mis-draws; extending keeps one rule for every input and contradicts nothing the engine defines | extend |
| Markers: keep the flat lattice, suppress the overlays, or displace them now | suppressing removes a working diagnostic; displacing needs sprite geometry this story does not have | keep the lattice |
| An uncovered pixel: transparent, or a fill colour | a fill colour cannot be told from a drawn pixel and hides a dropped column | transparent |

Geometry is integer arithmetic throughout; the level grid keeps its float model and no float reaches
a placement decision. The renderer consumes decoded grids and 32x32 images, and depends on no
archive or map type.

## Out of scope

- **Placing units, structures, objects or any sprite on the projected surface** — story 0015. The
  markers kept here sit on the un-displaced lattice, so on a projected image they do not register
  with the terrain beneath them: a stated limitation of this story's diagnostic output.
- The interactive viewer, which resolves and draws cells on its own path.
- Any change to the lighting model itself: the level formula, the light parameters, four-corner
  interpolation, dynamic light, fog, slope occlusion.
- The engine's viewport — scroll, over-scan margin and clipping. A still image has no clip rectangle.
- Hit-testing and picking geometry; altitude decoding; the dirt overlay; hidden-border cropping.

## Verification mapping

AC-1..AC-12 and P-1..P-6 are CI-automatable on synthetic data; AC-13 is developer-run.

## Gate check

FR-1 -> AC-1, AC-2, P-3 · FR-2 -> AC-3 · FR-3, FR-4 -> AC-4 · FR-5 -> AC-6, AC-7, P-1, P-5
· FR-6 -> AC-5, AC-9 · FR-7 -> AC-6 · FR-8 -> AC-8 · FR-9 -> AC-3, P-2 · FR-10 -> AC-10, P-4, P-6 ·
FR-11 -> AC-11 · FR-12 -> AC-12, AC-13
