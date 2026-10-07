# Plan — corner-interpolated lighting in the window

## Design decisions

### DD-1 — the level grid is built once, in `NewViewer`, under the guard that already exists

`terrain.LevelGrid(g.Altitudes, g.Width, g.Height, terrain.DefaultDaytime)` runs beside the existing
`Project` call, inside the same `validAltitudes(g)` branch — and it is **that guard**, not
`LevelGrid`'s own totality, that makes `v.levels != nil` mean "the altitude grid is valid". The two
predicates are not equivalent on their own: `LevelGrid` rejects a slice shorter than `w*h` and accepts
an overlong one, where `validAltitudes` demands exact length, so a 2x2 grid with five altitudes is
invalid to one and acceptable to the other. Sharing the branch is what makes `v.levels != nil` and
`v.proj != nil` decide one map the same way, and a later refactor that hoists the call out of the
branch loses that.

Rejected: a per-frame rebuild, over every vertex, for a value that cannot change while the viewer
exists. Rejected: widening `NewViewer` or `LoadMapViewer` with a `terrain.Light`. No caller has a reason to
choose one; the flag that would justify it is a `terraintool` diagnostic the spec keeps off the
window. `NewViewer` therefore keeps its signature, and every existing call site and test compiles
unedited.

### DD-2 — one clamp, exported once: `terrain.CornerLevels`

`func CornerLevels(levels []uint8, w, h, col, row int) [4]uint8` returns the four corner levels in
**TL, TR, BL, BR** order — the order the vertex builders consume.

"One place" was not true of the repository, and T1 made it true rather than claiming it: the far-edge
clamp existed **twice**, as a local closure in `lit.go` and a package-level function in `project.go` —
two independently written, coincidentally identical clamps. All three consumers now go through
`CornerLevels`, and that it changed no behaviour is checked rather than asserted (SC-11).

Rejected: exporting `levelAt` instead. That leaves every caller free to order the corners its own way,
and the order is the part that must not vary.

### DD-3 — `terrain.ShadeScale`, and it is exact

`func ShadeScale(level int) float32` returns `float32(LevelCount-level) / 32`, clamping the level
into `[0, LevelCount-1]` through the same helper `ShadeChannel` uses, so the two cannot disagree
about an out-of-range level. Every value it can return is `k/32` for an integer `k` in `[1,96]`
and so exactly representable in float32: no rounding enters the pipeline at this step.

### DD-4 — flat mode draws through `DrawTriangles`, and its one-pixel cost is measured

A `DrawImage` carries one `ColorScale` for a whole cell, which is per-facet fill — the thing FR-2
forbids. So flat mode submits a quad of the four lattice corners instead, each corner through
`cam.WorldToScreen`, which is the transform `flatTileScreen` already applies to one of them, and
through the **same** `quadIndices` the displaced path uses — the spec pins the TL–BR split for both
modes because it decides a non-uniform tile's interior, and a rectangle hides the difference in the
geometry while showing it in the shade.

The cost was measured, not assumed (probes under the untracked `builds/`). The two calls agree
byte-for-byte at every destination offset **except where an edge lands exactly on a pixel centre**,
where they break the tie in opposite directions — `DrawImage` right, `DrawTriangles` left. On a lattice
of adjacent tiles that is **not** a translation: some tiles moved and others did not, one tile's pixel
width changed, and at a tying boundary `DrawTriangles` left that column **unpainted**.

Column `c`'s edge is `(c*32 - X)*z`, so consecutive edges differ by `32z` and all boundaries share one
fractional offset exactly when `32z` is an integer — true at 1.0, 0.5, 1.5, 2.0, false at every zoom
the wheel reaches away from 1.0, since `WheelZoomStep` is `6/5` and `32*(6/5)^n` is never integral. So
a tie catches one fractional class and misses the others. That is why FR-1's exception is written per
boundary and bounds a tile's width instead of promising a uniform shift: the uniform reading is not
merely unproven, it is false at every zoomed state.

The seam is not new behaviour, it is newly **shared**: the displaced path has drawn every tile through
`DrawTriangles` since 0013, so the unpainted column is already reachable there. This story does not
fix it — the fix moves the `SrcX`/`SrcY` span 0013's DD-6 pins, a revision to 0013's contract rather
than a tweak inside this one. It is recorded as a defect with its reproduction.

Rejected: `DrawImage` when unlit, `DrawTriangles` when lit. It buys bit-identity with the shipped
binary and pays by making `-unshaded` compare two rasterisers as well as two brightnesses — the A/B
this story is checked by would stop isolating the light.

### DD-5 — the pure seam is `cornerScales` plus two vertex builders

`func (v *Viewer) cornerScales(tx, ty int) [4]float32` holds the entire lighting decision and needs
no window: all-1 when `!v.Lit()` or when the tile has no source sub-cell (DD-7), else `ShadeScale` of
each `CornerLevels` entry.

`tileVertices` **keeps its signature and its all-1 colours**: it is the geometry, and its four existing
call sites and the colour assertion over them keep their meaning unedited. The colours arrive through a
second pure function, `withScales(verts [4]ebiten.Vertex, sc [4]float32) [4]ebiten.Vertex`, and the
flat lattice through a sibling `flatTileVertices(cam, tx, ty)`; each draw loop composes the two.

Rejected: widening `tileVertices` to take the scales. It breaks four existing call sites and falsifies
an existing assertion that every displaced corner colour is 1, for no gain.

### DD-6 — `Lit()` is computed, and `SetUnshaded` deliberately resyncs nothing

`Lit()` is `v.levels != nil && !v.unshaded`, mirroring `Mode()` in being computed on each call rather
than latched, so the diagnostic can be set after construction. `SetUnshaded(bool)` is its only entry
and calls no `syncWorld`: 0013's DD-5 hazard was a cached world extent that depended on the flipped
state, and nothing cached depends on this one — the level grid is a function of the altitudes alone
and no geometry moves.

### DD-7 — the placeholder decision is made in the pure layer, not behind the upload

Whether a cell has a source sub-cell is decidable without a window —
`v.set.Slot(ref.Slot).SubCell(ref.Sub) == nil` over the resolution both modes already share — so
`cornerScales` answers it and returns all-1 for a placeholder cell.

Rejected: comparing the uploaded image against `v.placeholder` in the draw loop. The check would then
sit behind a GPU upload where no test can reach it, and AC-7's placeholder half would degrade from a
unit test to a manual observation.

### DD-8 — nothing else moves

`cellImage`, `resolveCell`, the cache key, `forEachDisplacedTile`, `RowRange`, `VisibleTiles`, the
overlay passes, the water ticker and both commands' `-check` summaries are untouched. The
`againrom` front-end gains no flag and no call: it is lit because `NewViewer` builds the level grid,
exactly as it is displaced because `NewViewer` builds the projection.

### DD-9 — the draw call is observable, so the loops are checked and not only their builders

`drawFlat` and `drawDisplaced` take a `triangleTarget` — the one method `*ebiten.Image` already
satisfies; `Draw` passes `screen`. Measured, not supposed: with T3's suite as first written, deleting
`withScales` from **both** loops left every package green — every assertion recomposed the builders
inside the test, so the story's visible effect, and flat mode's diagonal with it, was unverified.
`app_test.go` already drives both loops headlessly: the call was reachable, only its argument
type was not. Rejected: reading pixels back — unreadable before the game starts (SC-8).

### DD-10 — one level field, not two edges, and the drag anchor is not latched

`Input` gains `PrimaryDown bool` — the button's **level**, `ebiten.IsMouseButtonPressed`, not the
just-pressed/just-released edges `appInput` carries for the menu. The level is what makes the hard
cases fall out instead of being handled: a release outside the window, a lost focus, and a button
already held when the map opens are all just `PrimaryDown` false or a first true tick, and none needs
a second field. The viewer holds `dragging bool` plus the last cursor position; a tick with
`PrimaryDown` and no drag in progress **anchors and pans zero**, the next pans by the difference, and
`!PrimaryDown` clears it. Nothing survives a release, so no state can be stale on the next drag.

The delta is divided by the zoom, not the accumulated origin scaled: at zoom 2 a 10 px cursor move is
a 5 px world move, the same cursor-anchoring principle `ZoomAbout` already applies — so the two
interactions cannot contradict each other. Rejected: the cursor's absolute position against a
press-time origin. Identical arithmetic until the cursor warps or a tick is dropped, and then it is a
jump. Rejected: `PrimaryPressed`/`PrimaryReleased` — an edge cannot see a button already down.

## Risks

- **A vertex colour above 1 could have been clamped**, which would have left the story unbuildable
  without a fragment shader. Measured false before the contract was written: 1.5625 over a channel of
  100 yields **156**, the byte the CPU transform produces, and 3.0 saturates at 255 as it does.
- **The one-pixel tie of DD-4, and the unpainted column at a tying boundary.** Named in FR-1 and
  bounded there. The manual pass drives a zoom where a boundary ties and reports what it sees, rather
  than asserting the seam away; it also looks for terrain drifting one pixel against the overlay
  rects, which are drawn by a different call and do not share the tie.
- **`pkg/render/terrain`'s linked test binary has previously tripped a Windows Defender false
  positive.** If a `pkg/render/terrain` line reports oddly, re-run before believing it and record
  which runs were clean.

## Self-checks

- **SC-1** Every existing `pkg/render/camera`, `pkg/ui` and `pkg/game` test passes **unedited**: no
  exported signature in either package changes, `NewViewer` included.
- **SC-2** With the diagnostic on, every corner colour is exactly 1 and the four positions equal the
  lattice the pre-change `flatTileScreen` produced for the same camera.
- **SC-3** Four equal corner levels give four equal multipliers, so flat shading is the degenerate case
  of interpolation and not a second path (P-1) — **and** four unequal corner levels give four different
  multipliers, so a per-facet fill fails it. The equal case alone cannot tell the two apart.
- **SC-4** `ShadeScale` is compared with `==` against `(96-level)/32` at **every** level in `[0,95]`,
  not only at 0, 46, 64 and 95, so a hard-coded table of the interesting four fails it; and an
  out-of-range level clamps rather than escaping the range.
- **SC-5** `CornerLevels` for an interior, right-edge, bottom-edge and corner tile equals the read the
  CPU compositor makes for the same tile, with the compositor called in the same test rather than its
  result restated.
- **SC-6** All four lighting/mode combinations are exercised, and enabling an overlay is shown to move
  `Mode()` while leaving `Lit()` alone.
- **SC-7** The borrowed altitude slice is byte-compared after construction and after a full pass over
  every tile's corner scales.
- **SC-8** No test opens a graphics context; the vertex tests read struct fields.
- **SC-9** Both commands' `-check` output is compared against a build from the pre-change commit over
  the shipped corpus, and the **shape** of every compared line is asserted too — that each side is a
  real summary and how many were compared — so an equality that holds because both sides failed
  identically cannot pass for evidence.
- **SC-11** T1's unification is a no-op on the PNG paths: `lit_test.go` and `project_test.go` pass
  **unedited**, and a real map rendered by `terraintool` is byte-compared across the refactor, shaded
  and `-flat`, so a clamp that differs only on a far-edge tile cannot hide.
- **SC-12** A cell whose tile slot is absent has all-1 corner scales while its neighbours are lit, and
  the decision is reachable without a graphics context (FR-6, AC-7).
- **SC-13** Both loops are driven against a recording target: each tile's submitted vertices equal
  `withScales` of that tile's builder output at its corner scales, and the indices are `quadIndices`
  itself, in both modes. Dropping `withScales` from either loop, or giving flat mode its
  own diagonal, fails it.
- **SC-10** The manual pass runs the shipped corpus, records the fidelity differences it finds against
  that map's PNG render including the ones the spec disclaims, and drives at least one zoom at which a
  tile boundary ties so FR-1's seam is observed rather than assumed absent.
- **SC-14** The drag is driven through `v.step` as an `Input` sequence, so it is checked without a
  window: press, move, move, release at zoom 1 and at a zoom != 1, plus the anchoring tick and a
  zero-move drag. Dividing by the zoom in the wrong direction, and multiplying instead of dividing,
  each fail it — both are run.
- **SC-15** A drag whose cursor sits inside `EdgeMargin` pans exactly once, and the same cursor
  position with the button up still edge-scrolls. Dropping the suppression fails the first; making it
  unconditional fails the second.

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-4, DD-8, DD-9, DD-10 | SC-1, SC-2, SC-9, SC-11, SC-13, AC-9, AC-11 |
| FR-2 | DD-4, DD-5, DD-9 | SC-3, SC-13, AC-3, AC-4, AC-10 |
| FR-3 | DD-1, DD-2 | SC-5, AC-1 |
| FR-4 | DD-1, DD-6 | SC-6, AC-5 |
| FR-5 | DD-6, DD-4 | SC-2, AC-6 |
| FR-6 | DD-7 | SC-12, AC-7 |
| FR-7 | DD-5 | SC-8, AC-3, AC-4 |
| FR-8 | DD-1, DD-3 | SC-4, SC-7, AC-2, AC-8 |
| FR-9 | DD-10 | SC-14, SC-15, AC-12, AC-13 |
