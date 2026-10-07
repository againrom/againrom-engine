# Tasks — placed-objects diagnostic overlay

**Reading key.** `FR-x`/`AC-x`/`P-x` → `spec.md`; `SC-x` → `plan.md` §Success criteria; `DDx` →
`plan.md` §Design decisions; `R-x` → `plan.md` §Risks. Task kinds: **implementation** (one coherent
product change → exactly one implementation commit, trailer `SDD-Task: 0008-structures-overlay/T<n>`),
**developer-run verification** (agent authors, a human runs against a lawful install; produces
evidence, not an implementation commit).

This story adds **no package**. `pkg/render/terrain`, `pkg/ui`, `cmd/terraintool` and `cmd/mapview`
are already in the DAG allow-map and `docs/ARCHITECTURE.md`, so no `internal/archtest` or
`ARCHITECTURE.md` edit belongs to any task — SC-8 is the existing fail-closed check *staying* green.
The render-tier geometry stays stdlib-only and imports no `formats` package (DD3); `pkg/ui` adds only
`github.com/hajimehoshi/ebiten/v2/vector`, a sub-package its tier is already allowed. The two cmds do
the `alm.Map.Objects → cells` wiring (DD3, DD8).

**Separate-context tests.** `pkg/render/terrain/overlay_test.go` (T1) and `pkg/ui/overlay_test.go`
(T3) are authored by a context that has read `spec.md` and the relevant plan section — the API
signatures and the FR-6/DD6 contract — but **not** the implementation diff, so their oracles derive
from the spec formula rather than the code.

## T1 — the pure marker geometry and its PNG rasterizer  *(implementation)*

The whole render-tier half: the fixed-point → cell shift, the pure cross generator with its map-rect
clip, and the rasterizer that fills the clipped arms over an already-composited image. Nothing in the
compositors is touched — the overlay draws over their output (DD1).

- Files: `pkg/render/terrain/overlay.go` (ADD) — `MarkerColor = color.RGBA{0xff,0xd0,0x00,0xff}`
  (FR-6); `AnchorCell(x, y uint32) (col, row int)`, the single home of the `>>8` shift (DD3);
  `ObjectMarkerRects(col, row, cols, rows, cellpx int) []image.Rectangle`, the pure two-arm cross at
  `cellpx` with the `[c-r, c+r+1)` / `[c-⌊t/2⌋, c-⌊t/2⌋+t)` extents, `nil` for an off-map anchor or
  `cellpx < 1`, each arm intersected with the map pixel rect and dropped when `Empty()` (DD2, DD4);
  `DrawObjectMarkers(img *image.RGBA, cells []image.Point, cols, rows, cellpx int)`, which intersects
  each map-clipped arm with `img.Bounds()` and sets exactly those pixels to `MarkerColor` (DD5).
  Stdlib only, integer arithmetic only, ≤ 2 rectangles per object.
  `pkg/render/terrain/overlay_test.go` (ADD, separate-context) — synthetic cell lists only; no game
  bytes.
- Covers: FR-2, FR-3, FR-6, FR-1 (raster half); AC-1, AC-2, AC-3; P-1, P-2; SC-1, SC-2, SC-3, SC-4,
  SC-8; DD1, DD2, DD3, DD4, DD5; **R-3** (the one-pixel low-side bias of an even-thickness strip at
  odd `cellpx` is the exact intended contract, pinned by the odd/even cases of
  `TestObjectMarkerRectsGeometry`, not a defect to correct).
- Trade-off accepted here: this is the story's largest slice and the only render-tier-only one (not
  demoable by itself). It stays whole because its three functions are only correct together and both
  T2 and T3 consume them — folding it into either would duplicate the `>>8` shift that DD3 exists to
  prevent.
- **Done when:** `TestAnchorCellAndOffMap`, `TestObjectMarkerRectsGeometry`,
  `TestObjectMarkerRectsExtremes` and `TestDrawObjectMarkers` pass — including the `cellpx ≤ 2`
  tiny-map case where the map-rect clip actually truncates (SC-2) and the undersized-`img` case where
  the output-rect clip does (SC-4), both with no synthetic border — and the standing gate is clean.

## T2 — the `terraintool` opt-in overlay and count  *(implementation)*

Wire the PNG path: an opt-in flag that draws the markers after compositing and reports the decoded
object count, leaving the disabled path byte-for-byte the baseline. Depends on T1.

- Files: `cmd/terraintool/main.go` (MODIFY) — add `-objects` (default off); when set, build
  `[]image.Point` cells from `m.Objects` via `terrain.AnchorCell`, call `terrain.DrawObjectMarkers`
  on the composited image with `m.Width`/`m.Height` and `terrain.CellSize*scale`, and append the
  literal `, objects N` (`N = len(m.Objects)`) to the end of the existing summary line (DD7).
  `cmd/terraintool/main_test.go` (MODIFY) — extend the synthetic `.alm` fixture to carry placed
  objects at chosen anchor cells with the one off-map anchor **first**, the only index the type-4
  walk accepts in a no-extension layout (plan §Facts verified during planning).
- Covers: FR-1, FR-4 (independent opt-in), FR-5; AC-4, AC-5; P-3; SC-5; DD5, DD7; R-2 (the reported
  count is `len(m.Objects)` — the decoded total, off-map anchors included — never a recomputed
  `type4_size/20`).
- **Done when:** `TestRenderObjectsOverlay` passes — every **in-map** anchor cell's centre pixel is
  `MarkerColor`, the off-map object is marked nowhere, the summary carries `objects N` with
  `N = len(m.Objects)`, and the flag-off PNG is byte-identical to the bytes `Composite`/`CompositeLit`
  produce for the same inputs (SC-5's non-tautological oracle) with the pre-0008 summary shape — every
  pre-existing `cmd/terraintool` test still passes unchanged, and the standing gate is clean.

## T3 — the interactive viewer's object overlay  *(implementation)*

The viewer half: overlay state and toggle, the pure camera transform + view cull, and the draw step
after the terrain tile loop. The transform is the *same* one a terrain tile gets, so markers track
their cells through pan and zoom rather than a fixed pixel grid (R-1). Depends on T1.

- Files: `pkg/ui/overlay.go` (ADD) — `objectCells []image.Point` / `showObjects bool` viewer state,
  `SetObjects(show bool, cells []image.Point)`, and the pure `objectScreenRects() []screenRect`:
  per cell `terrain.ObjectMarkerRects(col, row, cols, rows, terrain.CellSize)`, each arm mapped by
  `cam.WorldToScreen(Min)` plus a `Zoom`-scaled size, dropping rects fully outside
  `[0,ViewW) × [0,ViewH)`, and returning nil when the overlay is off or holds no cells (DD6, DD8).
  `screenRect` holds `float64` fields per DD6; the narrowing to `float32` happens only at the
  `vector.DrawFilledRect` call site.
  `pkg/ui/viewer.go` (MODIFY) — `Draw` fills each `objectScreenRects()` entry with
  `vector.DrawFilledRect(screen, …, terrain.MarkerColor, false)` after the tile loop when
  `showObjects`, float coordinates passed straight through (no independent snapping, FR-6).
  `pkg/ui/overlay_test.go` (ADD, separate-context) — synthetic grid + positioned camera; the oracle
  is `cam.WorldToScreen(corner)` and the `Zoom`-scaled size derived from the camera contract, never a
  second call to `objectScreenRects`.
- Covers: FR-1, FR-3 (the view-rect stage, realized by the framebuffer clip per DD6), FR-4, FR-6
  (viewer path); AC-4 (the automatable viewer witness); SC-6, SC-8; DD6, DD8; R-1.
- **Done when:** `TestObjectScreenRects` passes at more than one pan/zoom position, culls a rect
  fully outside the view, and returns nil with the overlay off; every pre-existing `pkg/ui` test still
  passes; and the standing gate is clean.

## T4 — the `mapview` opt-in overlay and count  *(implementation)*

Wire the interactive path: the flag reaches `load()`, which does the `alm.Map.Objects → []image.Point`
conversion, hands the cells to the viewer, and reports the count on the headless summary. Depends on
T3.

- Files: `cmd/mapview/main.go` (MODIFY) — add `-objects` (default off) and pass it into `load()`;
  when set, `load()` builds the cells via `terrain.AnchorCell`, calls `viewer.SetObjects(true, cells)`
  and appends `, objects N` (`N = len(m.Objects)`) to the summary it returns, so `-check` prints it
  too (DD7, DD8). `cmd/mapview/main_test.go` (MODIFY) — extend the synthetic `.alm` fixture with
  placed objects at chosen anchor cells, the one off-map anchor **first** (same walk constraint as
  T2); assert `-objects` reports `objects N` under `-check` and that without the flag the summary
  keeps its pre-0008 shape.
- Covers: FR-1, FR-4, FR-5; AC-4, AC-5; SC-7; DD7, DD8; R-2 (the reported count is
  `len(Map.Objects)`, never a recomputed record count).
- **Done when:** `TestObjectsFlag` passes — `objects N` appears with `N = len(m.Objects)` in the DD7
  position (after `tile slots …`, before `, water speed …`), and the flag-off summary is
  character-for-character the pre-0008 shape — every pre-existing `cmd/mapview` test still passes
  unchanged, and the standing gate is clean.

## T5 — overlay evidence against a lawful install  *(developer-run verification)*

Load a real GOG map with placed objects in **both** viewers with the overlay on and record
**evidence only** — no game bytes, no rendered PNG.

- Produces: evidence in `verification.md` — the map name and size; that `terraintool -objects` marks
  cells that hold visible structures; that `mapview -objects` shows the same markers and that they
  stay on their terrain cells through pan and zoom (the R-1 sub-pixel difference from the PNG is
  expected and is not a failure); and that the `objects N` both tools report equals the map's object
  count (R-2).
- Covers: AC-6; SC-9; R-1, R-2.
- **Done when:** a developer has run both viewers against a lawful install and `verification.md`
  records what was actually observed. Absent that run the task is **not** satisfied and AC-6/SC-9
  stay unclaimed: `verification.md` must then carry them as an explicit, labelled *pending
  developer-run limitation* naming what was not observed, and the conclusion must not rest on them.
  Either way `bash scripts/check-no-game-assets.sh` stays clean and no rendered image or game byte is
  committed.

## Traceability

Each row's plan criterion is one the plan itself attributes to that requirement.

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (opt-in marker draw in both viewers) | SC-4, SC-5, SC-6, SC-7 | T1, T2, T3, T4 |
| FR-2 (pure bounded geometry, no panic) | SC-1, SC-3 | T1 |
| FR-2 / constraint (render tier stays stdlib-only, DAG) | SC-8 | T1, T3 |
| FR-3 (map-rect then output/view-rect clip) | SC-2, SC-4 | T1 |
| FR-3 (view-rect stage in the viewer, via the framebuffer clip) | SC-6 | T3 |
| FR-4 (independent, composable opt-in) | SC-5, SC-6, SC-7 | T2, T3, T4 |
| FR-5 (report `len(Objects)`) | SC-5, SC-7 | T2, T4 |
| FR-6 (the pinned pixel geometry) | SC-2, SC-4, SC-6, SC-9 | T1, T3, T5 |
| AC-1 | SC-1 | T1 |
| AC-2 | SC-2, SC-4 | T1 |
| AC-3 | SC-3 | T1 |
| AC-4 | SC-5, SC-6, SC-7 | T2, T3, T4 |
| AC-5 | SC-5, SC-7 | T2, T4 |
| AC-6 | SC-9 | T5 |
| P-1 (≤ 2 rects, never indexes out of bounds) | SC-3, SC-4 | T1 |
| P-2 (no panic, no magnitude-proportional allocation) | SC-3 | T1 |
| P-3 (disabled ⇒ byte-identical terrain) | SC-5 | T2 |
| R-1 (viewer not pixel-identical to the PNG — intended) | SC-6, SC-9 | T3, T5 |
| R-2 (count is the decoded total, not `size/20`) | SC-5, SC-7, SC-9 | T2, T4, T5 |
| R-3 (one-pixel low-side bias at even thickness) | SC-2 | T1 |
