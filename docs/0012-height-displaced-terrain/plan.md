# Plan — height-displaced terrain geometry (ROM1)

Brownfield: the base renderer's geometry changes; the raster it draws today becomes a diagnostic.

## Approach

The projection lands as new files **beside** the existing compositors in `pkg/render/terrain`, not as
a rewrite of them: `Composite` and `CompositeLit` keep their pixels and become the flat raster FR-1
preserves, two sibling entry points draw the projected geometry over one shared core, and the
vertical origin rides on the render result. Nothing observable moves until `cmd/terraintool` repoints
its default, adds the flat selection and extends its summary.

## Facts verified during planning

- `Composite` and `CompositeLit` are the only compositors and `cmd/terraintool` their only caller
  outside the package; `pkg/ui` draws cells on its own camera path, so `cmd/mapview`, `pkg/game` and
  `cmd/againrom` are untouched.
- `pkg/render/terrain` is a DAG leaf: the `internal/archtest` allow-map grants it no intra-module
  import and is fail-closed on any package not listed.
- `Render` carries `Image` and `Placeholders`. The cap is `maxRenderPixels = 1<<28`, tested with `>`
  in `int64` before allocation, per compositor. Its one "over the pixel cap" case never reaches that
  branch — the grid-length check rejects its nil `Tiles` first — and `CompositeLit` has none.
- `image.NewRGBA` zero-fills, so an unwritten pixel is already transparent; `SetRGBA` drops an
  out-of-bounds store rather than panicking.
- `ShadeRGBA(c, {0,0,0}, 64)` is the identity, and `lit_test.go` pins `CompositeLit` at that level
  byte-equal to `Composite`: the unshaded path is an unattenuated one, not a second raster.
- `InterpRow` uses one denominator for both axes and is the only level interpolator. The dirt overlay
  is read at the same source coordinates as the terrain pixel it composites with.
- `markerRects` clips to `image.Rect(0, 0, cols*cellpx, rows*cellpx)`; `pkg/ui` calls
  `ObjectMarkerRects`/`UnitMarkerRects`, `cmd/terraintool` calls `Draw*Markers`.
- Vertex rows `H-1` and `H` read the same clamped altitude row, so `maxV - minV >= 32` always.
- The `cmd/terraintool` fixture's altitudes are all zero, so **every** assertion there is blind to
  the projection: same pixels, same dimensions, origin 0, whichever path the tool takes.
- Over `d = 1..255` the recipe's rows are non-decreasing and end at 32; the walks differ at 63, 127
  and 255 with the mirrored one **ahead** by one at all 32 columns, and at 191 (all columns) and 240
  (columns 8 and 23) with it **behind** by one. Only the first sign gives the seam overdraw; the
  second leaves the seam row drawn by neither cell. The corpus maximum step is 127.

## Files to touch

| Path | Intent |
|---|---|
| `pkg/render/terrain/step.go`, `project.go`, and their `_test.go` | ADD |
| `pkg/render/terrain/composite.go`, `lit.go` (DD-6, DD-9), `shade.go` (DD-7), `overlay.go` (DD-8), `doc.go` | MODIFY |
| `pkg/render/terrain/shade_test.go`, `overlay_test.go` | MODIFY |
| `cmd/terraintool/main.go`, `main_test.go` (DD-5, DD-11) | MODIFY |

No package is added or moved, so `internal/archtest` and `docs/ARCHITECTURE.md` need no edit.

## Design decisions

**DD-1 — the new code lives in `pkg/render/terrain` itself**, as new files. *Rejected:* a
`pkg/render/terrain/geom` sub-package — the allow-map grants this package no intra-module import, so
a leaf would need a DAG edit to import its own child, for separation the file split already gives.

**DD-2 — the step table is one immutable package-level table, built at initialisation for
`d = 1..255`** — the largest step two signed altitudes can produce. Entries fit `uint8` (none exceeds
32); the table is ~33 KB. *Rejected:* building it per composite (a full rebuild per image of a
map-independent object) and `sync.Once` (it defers one 33 KB loop). `d = 0` is the constant-edge rule
and never reaches the table; the accessor returns no row outside `1..255` rather than indexing blind.

**DD-3 — two blitters stay; the flat cell keeps the rectangle.** The selector picks the rectangle
blitter for a flat cell, the span blitter for a sloped one; their agreement at `S = 32` is asserted
by test rather than shared code. *Rejected:* one blitter with the flat cell as its degenerate case —
the flat raster places *every* cell as a rectangle whatever its altitudes, so unification removes
nothing and only hides FR-3's selector. In the span blitter the dirt sub-cell is
read at the terrain pixel's own `(i, srcRow(j))`, so the decoded overlay cannot shear against its
tile; a placeholder takes the same selector and geometry as the tile it replaces, unlit and counted
as today, a rectangle on a sloped cell being where its cell is not.

**DD-4 — the entry points are additive.** `Composite` and `CompositeLit` keep their signatures,
pixels and rejection sets and are the flat raster; `CompositeProjected` (unshaded) and
`CompositeProjectedLit` (shaded) join them over one unexported core differing only in whether a level
grid is consulted. *Rejected:* changing `Composite`'s signature, or one options struct for all four —
both rewrite the call sites and the pins FR-1 needs held still.

**DD-5 — geometry and shading are independent axes**, so all four combinations are reachable. The
summary gains geometry and origin as two adjacent tokens, in that order, after the light descriptor
and before the overlay counts, so the existing "summary + `, objects N`" composition still holds.
*Rejected:* folding the flat raster into the unshaded flag — one flag would silently change two
unrelated things, against the premise that the projection is the default for *every* terrain image.

**DD-6 — the projection is a value and the origin rides on the result.** `Project` takes altitudes as
`[]uint8` read signed at use (what `alm` decodes and both compositors take) and returns an exported
`Projection` holding the clamped vertex accessor and the canvas extremes, computed in one pass over
the `(W+1)*(H+1)` vertices before anything is allocated; `Render` gains `OriginY int`, the native
`minV`, 0 on both flat paths. *Rejected:* extremes computed inside the cell loop (allocation needs
them first) and a tool-side recomputation (two implementations of one number). Exporting also lets
AC-3 test the formula rather than infer it from pixels.

**DD-7 — level interpolation gains `InterpSpan`, with independent x and y denominators**;
`InterpRow` becomes the equal-denominator call, arithmetic unchanged and still pinned. *Rejected:*
widening `InterpRow` (every caller and test for one argument) and interpolating in the blitter (the
one place levels are computed becomes two).

**DD-8 — markers get additive `*At` entry points taking `offsetY` in output pixels**, which the
caller sets to `-OriginY * scale`: native row `r*32` lands at `r*32*scale - minV*scale`, so the
lattice takes the terrain's own translation. The clip rectangle moves with it; the second clip
against the image bounds is unchanged, so an in-map anchor whose glyph leaves the canvas is truncated
by that clip, and the off-map drop stays a cell-index test. The existing `Draw*Markers` become the
`offsetY = 0` case. *Rejected:* changing those signatures (`pkg/ui`
consumes them and has no origin to pass) and passing the native origin (the callee would have to
recover `scale` from `cellpx`).

**DD-9 — one validation helper serves all four entry points**, in the existing order and with the
existing atomic rejection, taking the canvas height as a parameter so the flat pair keeps measuring
`H*32*scale` and the projected pair `(maxV-minV)*scale`; its message recommends a smaller scale. The
projected pair therefore validates in two steps — arguments, project, budget. *Rejected:* a second
copy of the preamble; divergent copies of one rejection set is how a rejection quietly stops applying
to one path.

**DD-10 — integer arithmetic throughout, and no defensive clamps.** All geometry is `int`; only the
budget product is `int64`, as today. No clamp on the source row, because `srcRow(S-1) <= 31` holds
for every positive `S` by the division's own definition; none on the destination row, because each
edge is bounded by the vertex pair it walks between and the canvas is those vertices' extremes.
*Rejected:* clamping either — it turns a geometry bug into a slightly wrong image instead of a
failing test. `SetRGBA`'s bounds check is the residual guarantee behind P-6, not what P-2 relies on.

**DD-11 — the existing flat pins stay as they are, and the tool gains a sloped fixture.** The
composite, lit and overlay suites keep asserting today's pixels, that being the flat raster's
contract; new coverage goes in new files. In `cmd/terraintool` the summary assertions move, the
baseline oracles are rebuilt from the entry point the tool actually calls, and a second fixture with
a non-uniform altitude grid is added, because every assertion there runs on an all-zero grid and
would pass whether or not the tool projects. *Rejected:* repointing the existing suites at the new
default (it deletes the only evidence the flat raster survived) and giving the existing fixture
altitudes (it invalidates every cell-centre assertion built on it).

**DD-12 — the walk is drawn as specified, including where the extension's seam is a gap.** Past the
engine's 127 rows the walks disagree in both directions (*Facts*), so at some step counts the lower
cell repaints one row of the upper and at others the seam row is drawn by neither; neither is
special-cased, and the classification is pinned by test. *Rejected:* nudging the disagreeing walk or
clamping the seam — a second rule for input the first already answers, against the decision to extend
one recipe over every step count.

## Risks

**R-1 — the pixel budget can now reject an invocation that worked before**: a 256x256 map with a wide
altitude span at scale 2 reaches about 16384 x 16894 and exceeds `1<<28`, where the flat render
fitted exactly. *Mitigation:* keep the cap, recommend a smaller scale in the error, leave the flat
selection as the escape hatch, and record in the developer-run pass which maps hit it.

**R-2 — every terrain PNG changes, and every hash the earlier terrain stories recorded is
superseded.** *Mitigation:* the flat selection reproduces them exactly on demand, so an old hash
stays checkable; the developer-run pass re-takes them at the new default.

**R-3 — the step recipe past `d = 127` cannot be checked against the engine**, the corpus maximum
being exactly 127 (provenance, *Open*). *Mitigation:* one rule for every step count rather than a
rejection or a clamp, its behaviour classified by test across the whole range rather than assumed
uniform.

**R-4 — markers on a projected image do not register with the terrain under them**, keeping the
un-displaced lattice. *Mitigation:* sprite geometry is 0015's; the summary names the geometry, and
the flat selection still registers.

**R-5 — the seam looks like a bug in both of its forms**: a repainted row at the step counts the spec
names, an undrawn row at two counts only synthetic altitudes reach. *Mitigation:* pin both forms with
the step counts that produce them, so an attempt to smooth either fails a test naming it as intended.

**R-6 — one outlier vertex inflates the whole canvas** by up to 255 near-empty native rows.
*Mitigation:* the reported origin and height make that visible in the summary rather than as an
unexplained image size.

## Success criteria

- **SC-1** — `TestProjectVertices`: the vertex formula, its out-of-grid clamp, the canvas bounds, the
  origin and both output dimensions, over a grid holding `0x80` and `0x7f`. (FR-2, FR-9, AC-3)
- **SC-2** — `TestStepTable`: row shape and sentinel over the whole built range, the walks' agreement
  everywhere else, and the disagreeing step counts with the sign of each. (FR-6, AC-5, R-3, R-5)
- **SC-3** — `TestProjectedSeam`: at a step count of each class, vertically adjacent cells repaint one
  row or leave one undrawn, as classified. (FR-6, FR-8, R-5)
- **SC-4** — `TestProjectedSelector`: which blitter each corner set takes, and where its output
  lands. (FR-3, FR-4, AC-4)
- **SC-5** — `TestProjectedSpanSampling`: the per-pixel source column and row, on a plain and on an
  impassable cell; the drawn top and undrawn bottom; `srcRow(S-1)` swept over every span. (FR-5,
  AC-6, P-1)
- **SC-6** — `TestProjectedShadingDomain`: the level ramp's vertical domain on each path, with
  distinct corner levels. (FR-7, AC-6)
- **SC-7** — `TestProjectedCollapsedColumn`: an untouched, still-transparent column and unaffected
  neighbours, on a tile and on a placeholder cell. (FR-5, AC-7, P-5)
- **SC-8** — `TestProjectedOwnership`: who owns a shared pixel, and that an uncovered one stays
  transparent. (FR-8, AC-8)
- **SC-9** — `TestProjectedPixelsInBounds`: containment and `scale x scale` replication on a relief
  grid at several scales. (FR-9, P-2)
- **SC-10** — `TestProjectedEqualsFlat`: byte and dimension equality with the flat counterpart for
  uniform zero, positive and negative grids, at several scales, on both entry points. (FR-1, AC-1,
  P-3)
- **SC-11** — `TestProjectedExtremeDeltas`: deltas of 128 and more, including -128 beside 127. (FR-6,
  AC-9, P-6)
- **SC-12** — `TestProjectedRejectsBadArguments`: the rejection set, atomically, from every entry
  point, each case reaching the branch it names — the budget one via a scale over the cap on a
  full-length grid — with caller slices unmodified. (FR-10, AC-10, P-4, R-1)
- **SC-13** — `TestMarkersAtOrigin`: the shifted lattice at a non-zero origin of each sign, the
  composition order, and the dropped off-map anchor. (FR-11, AC-11)
- **SC-14** — `TestRenderGeometrySummary`, `TestRenderFlatSelection`, on the sloped fixture: the two
  summary tokens, reported against written dimensions, and the flat selection's pixels, dimensions
  and placeholder count against the pre-change render. (FR-1, FR-12, AC-2, AC-12)
- **SC-15** — manual: the shipped maps at representative scales, the re-taken hashes recorded in this
  story's own verification record; the earlier stories' remain historical. (AC-13, R-1, R-2)
