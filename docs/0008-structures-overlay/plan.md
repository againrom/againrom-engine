# Plan — placed-objects diagnostic overlay

Reading key: `FR-x`/`AC-x`/`P-x` -> `spec.md`. This file fixes the API contract, the design decisions,
and the success criteria against which the work is verified. It is derivable from the spec alone.

## Approach

Add a placed-objects diagnostic overlay as new, pure, stdlib-only geometry in `pkg/render/terrain` plus a
draw step in each of the two terrain consumers. The render tier gains: the fixed-point-to-cell shift
(`AnchorCell`), the pure cross generator with map-rect clipping (`ObjectMarkerRects`, a function of
`(anchorCell, cellPixels)` returning <=2 half-open `image.Rectangle` arms, FR-2/FR-6), and a PNG
rasterizer (`DrawObjectMarkers`) that fills the clipped arms opaque yellow over an already-composited
image (FR-1/FR-3). The PNG tool `cmd/terraintool` gains an opt-in `-objects` flag that draws the overlay
after compositing and reports the object count (FR-5). The interactive viewer `pkg/ui` gains an opt-in
object overlay drawn after the terrain tile loop, transforming each native (32 px/cell) arm through the
**same camera transform as terrain** and filling it with `vector.DrawFilledRect` (FR-6); `cmd/mapview`
gains the matching `-objects` flag. Nothing is drawn unless the flag is set, so with the overlay off the
terrain output and summary are byte-for-byte the baseline (P-3). No package is added, so the DAG is
unchanged; the marker geometry imports no `formats` package (the two cmds do the `alm.Map.Objects ->
cells` wiring).

## Facts verified during planning (baseline, frozen)

- `pkg/formats/alm` exposes `Map.Objects []alm.Object`; each `Object` carries `X, Y uint32` (fixed-point
  `/256`) plus raw `Flags`/`Kind`/`ID`/`Value` and an optional `Ext []byte` the overlay never reads. The
  map size is `Map.Width`/`Map.Height int`. Object record 0's first 8 bytes are overlaid by the section
  identity, but its `X`/`Y` are still readable (research `ALM-OBJ-019`), so object 0's anchor is valid and
  needs no special-casing. `len(Map.Objects)` is the adaptive-walk decoded count (a few records carry the
  8-byte extension), i.e. the true object count, not `type4_size/20`.
- The type4 walk discriminates that optional extension by **reading the bytes at the cursor as if a base
  record started there and testing that record's anchor**: after base record `i` (`i < n-1`) it takes the
  u32 pair at `cur+8`/`cur+12` and, if `X>>8 >= W || Y>>8 >= H`, consumes the 8 bytes at `cur` as record `i`'s
  extension instead of starting record `i+1`. (The *final* record's extension is decided by the remaining
  payload length, not by that predicate.) The predicate is value-identical to this overlay's off-map test
  for every anchor `AnchorCell` can produce - `int(X>>8)` is never negative, and `Width`/`Height` come
  from the same u32 reads - so the two tests fire together.
  Consequence for the **no-extension layout** a hand-built map naturally uses (the two cmd-tier fixtures):
  record 0 is the only index whose anchor the walk never inspects, and its `X`/`Y` at `+0x08`/`+0x0c`
  survive the 8-byte section-identity overlay, so it is the only place a single off-map anchor can sit. An
  off-map anchor at a later index makes the preceding record swallow 8 bytes, the walk desynchronises, and
  `alm.Open` fails outright (observed on hand-built maps: `type4 record N does not fit in payload`,
  `record N: no room for the next record`). This is a property of that *layout*, **not** a limit of the
  format: a payload that gives record `i-1` a genuine 8-byte extension - and whose record `i` header words
  at `+0x00..+0x08` happen to read off-map, so the peek fires - does carry an off-map anchor at index `i`
  and decodes cleanly (verified as an existence case, not as a universal). The constraint the fixtures adopt is therefore "no-extension
  layout, off-map anchor first" - deliberately the simplest representable one - not "the format permits
  only one".
  Separately, `#type4 = 0` is not representable at all: an empty section may only eliminate to type 8
  (`ALM-META-025`), and a non-empty payload with count 0 fails the walk's exact-consumption check. So
  `objects 0` cannot be exercised at the cmd tier; only the *disabled* path can witness "no count
  reported".
  All of this binds only the two cmd fixtures. The render tier takes plain integer cells, not bytes, so
  its off-map cases (SC-1, SC-3) are unconstrained.
- `pkg/render/terrain` is stdlib-only, registered in `internal/archtest` + `docs/ARCHITECTURE.md`; adding
  files to it needs no DAG edit. `CellSize = 32`. `Composite` and `CompositeLit` each return
  `*Render{Image *image.RGBA; Placeholders int}`; the image is exactly the map extent
  (`Width*CellSize*scale` x `Height*CellSize*scale`). The package already imports `image` and
  `image/color`.
- `image.Rectangle` is a half-open integer rectangle; `Intersect` truncates each edge to the overlap and
  returns the zero rectangle when disjoint, never synthesizing a new edge (FR-3). This is the clip
  primitive used throughout.
- `cmd/terraintool` composites (`Composite` when `-unshaded`, else `CompositeLit`), writes a PNG, and
  prints one summary line; `-scale N` sets the pixel scale, so the PNG has `cellpx = CellSize*scale` pixels
  per cell. The asset root comes from `-assets`/`AGAINROM_ASSETS`.
- `pkg/ui` imports `pkg/render/terrain`, `pkg/render/camera`, and `github.com/hajimehoshi/ebiten/v2`
  (+ `inpututil`); it does **not** import `pkg/formats/alm`, and the DAG forbids it to. The allow-map lets
  `pkg/ui` import `pkg/render`/`pkg/render/*` and any `github.com/hajimehoshi/ebiten/v2/...` sub-package
  (so `.../vector` is permitted). `Viewer.Draw` iterates `cam.VisibleTiles()` and draws each tile with
  `op.GeoM.Scale(zoom,zoom)` then `op.GeoM.Translate(cam.WorldToScreen(col*32,row*32))`,
  `op.Filter = FilterNearest`. `camera.Camera` exposes `WorldToScreen(wx,wy) (sx,sy float64)`, `Zoom`,
  `ViewW`, `ViewH`.
- `github.com/hajimehoshi/ebiten/v2/vector` provides
  `DrawFilledRect(dst *ebiten.Image, x, y, width, height float32, clr color.Color, antialias bool)`.
- `cmd/mapview` has a headless `-check` path that prints a one-line summary and exits without a window;
  both cmds resolve the asset root via `-assets`/`AGAINROM_ASSETS`, never hardcoded.

## Files to touch

| Path | Intent | Why |
|---|---|---|
| `pkg/render/terrain/overlay.go` | ADD | `MarkerColor`; `AnchorCell(x,y uint32) (col,row int)` (the `>>8` shift, DD3); `ObjectMarkerRects(col,row,cols,rows,cellpx int) []image.Rectangle` (pure cross + map-rect clip, DD2/DD3/DD4); `DrawObjectMarkers(img *image.RGBA, cells []image.Point, cols,rows,cellpx int)` (PNG rasterizer, DD5). Stdlib only. |
| `pkg/render/terrain/overlay_test.go` | ADD | Synthetic geometry + rasterizer tests (AC-1, AC-2, AC-3, P-1, P-2, and the draw). |
| `cmd/terraintool/main.go` | MODIFY | Add `-objects`; after compositing, if set, build cells from `m.Objects` via `AnchorCell` and call `DrawObjectMarkers`; append the object count to the summary (DD7). Disabled path unchanged (P-3). |
| `cmd/terraintool/main_test.go` | MODIFY | Extend the fixture to place objects at chosen anchor cells, the one off-map anchor first (the no-extension fixture layout - see the baseline facts); assert `-objects` marks each **in-map** anchor cell yellow and reports `objects N`; the disabled render is byte-identical to the compositor's own output for the same inputs and the summary keeps its shape (AC-4, AC-5, P-3). |
| `pkg/ui/overlay.go` | ADD | Viewer object-overlay state (`objectCells []image.Point`, `showObjects bool`), `SetObjects(show bool, cells []image.Point)`, and the pure `objectScreenRects() []screenRect` transform+cull the draw path calls (DD6). |
| `pkg/ui/overlay_test.go` | ADD | Synthetic transform/cull tests over a positioned camera (FR-6 viewer path, FR-4 toggle). |
| `pkg/ui/viewer.go` | MODIFY | `Draw` calls `vector.DrawFilledRect` for each `objectScreenRects()` entry after the terrain tile loop when `showObjects` (DD6). |
| `cmd/mapview/main.go` | MODIFY | Add `-objects`; pass the flag into `load()` (which already holds `m` and builds the summary), so when set `load()` wires `m.Objects` -> `[]image.Point` via `AnchorCell`, calls `viewer.SetObjects(true, cells)`, and appends `, objects N` to the summary it returns (DD7). Disabled path unchanged. |
| `cmd/mapview/main_test.go` | MODIFY | Extend the fixture to place objects at chosen anchor cells, the one off-map anchor first (same no-extension fixture layout); assert `-objects` reports `objects N` in `-check` and that absent it the summary is unchanged (AC-4, AC-5). |

No `internal/archtest` or `docs/ARCHITECTURE.md` edit (no new package; `pkg/render/terrain` stays
stdlib-only, `pkg/ui` adds only an ebiten sub-package already permitted to its tier). No `pkg/formats`
edit. `composite.go`/`lit.go` are not touched — the overlay draws over their output.

## Design decisions

- **DD1 - All overlay drawing happens over the finished image; the compositors are untouched.** The PNG
  overlay runs after `Composite`/`CompositeLit`, and the viewer overlay after the terrain tile loop, so the
  overlay composes with both the unshaded and shaded terrain paths (and with 0006 water) without modifying
  any of them. With the overlay off, no overlay code runs, giving the byte-identical baseline P-3 demands
  for free.
  *Rejected:* threading an overlay flag through the compositors / the tile draw and mutating them - needless
  brownfield churn to tested contracts, and it risks a stale caller shading/overlaying with wrong inputs;
  the draw-after design keeps the change purely additive (greenfield).

- **DD2 - Marker rectangles are stdlib `image.Rectangle`; clipping is `image.Rectangle.Intersect`.**
  `image.Rectangle` is a half-open integer rect and its `Intersect` truncates to the overlap, returning an
  empty rect when disjoint - exactly FR-3's "intersect ... clipping MUST NOT synthesize a new edge." Fill
  loops iterate `[Min.X,Max.X) x [Min.Y,Max.Y)`.
  *Rejected:* a bespoke `Rect` type with hand-rolled intersection - it reinvents a tested stdlib primitive
  and is the exact place the FR-3 synthetic-edge bug would creep in.

- **DD3 - The pure entry points take plain integers and import no `formats` package.** `AnchorCell(x,y
  uint32) (col,row int) = (int(x>>8), int(y>>8))` is the only place the `/256` fixed-point -> cell shift
  lives, and it is pure integer math (no `alm` type crosses into render, honouring the DAG and the spec
  constraint). `ObjectMarkerRects(col,row,cols,rows,cellpx int)` takes the already-shifted cell plus the
  map size and scale. The two cmds compose them: `col,row := AnchorCell(o.X,o.Y)`. AC-1 tests both together
  over a fractional `/256` cell list with off-map anchors.
  *Rejected:* passing `alm.Object`/`alm.Map` into the render tier - a DAG violation (formats must not reach
  the render tier) and forbidden by the spec constraint; and *rejected:* doing the `>>8` shift inline in
  each cmd without a shared render-tier helper - it would leave the AC-1 conversion untested at the tier
  the spec assigns it to and duplicate the shift in two places.

- **DD4 - The cross is two arms per the reconfirmed FR-6, and the native cross is the `cellpx=32`
  instance.** For anchor cell `(col,row)` at `cellpx`: center `c=(col*cellpx+floor(cellpx/2),
  row*cellpx+floor(cellpx/2))`, arm radius `r=max(1,round(6*cellpx/32))`, thickness
  `t=max(1,round(3*cellpx/32))`, `round(n/32)=floor((n+16)/32)` for `n>=0`. The horizontal arm is
  `[cx-r,cx+r+1) x [cy-floor(t/2),cy-floor(t/2)+t)` and the vertical arm
  `[cx-floor(t/2),cx-floor(t/2)+t) x [cy-r,cy+r+1)`. At `cellpx=32` this is `[cx-6,cx+7)x[cy-1,cy+2)` and
  `[cx-1,cx+2)x[cy-6,cy+7)`, the native rectangles. `cellpx<1` yields nil (invalid scale -> no geometry,
  FR-2). All arithmetic is integer; the two rects are built into a fixed-capacity slice, so nothing
  allocates proportionally to coordinate magnitude (P-2). `ObjectMarkerRects` then intersects each arm with
  the map pixel rect `[0,cols*cellpx) x [0,rows*cellpx)` and drops any arm that is `image.Rectangle.Empty()`
  after the intersection (NOT a `== image.Rectangle{}` test - Go's `Intersect` returns an empty-but-nonzero
  rect on an edge-only overlap); an off-map anchor (`col<0||col>=cols||row<0||row>=rows`) returns nil before
  any geometry is built (AC-1). The result is <=2 rects (P-1). Integer width: `col` is `int(X>>8) <=
  0xFFFFFF` and `cellpx` is bounded (`CellSize*scale`, capped by the compositor); on the amd64 target
  (64-bit `int`) the products stay well in range, and Go wraps rather than panics regardless, so AC-3's
  "no panic" holds unconditionally.
  **Honesty note (clipping reachability):** at every realistic `cellpx` (>= 3) the whole cross fits strictly
  inside its own cell (`r <= ceil(cellpx/2)-1` fails), so the *map-rect* intersection truncates nothing for
  an in-map anchor - it is only observable at `cellpx <= 2` (a degenerate scale) or at an edge/corner anchor
  where an arm crosses the map boundary. The verification (SC-2/SC-4) therefore exercises the clip with a
  `cellpx <= 2` edge anchor and with an undersized output rectangle, rather than pretending a `cellpx=32`
  edge cell truncates.
  *Rejected:* a symmetric `[c-r,c+r)` arm (length `2r`) - it fails to reproduce the native `[cx-6,cx+7)` and
  would fail AC-2; the reconfirmed FR-6 pins `[c-r,c+r+1)`.

- **DD5 - `DrawObjectMarkers` fills each map-clipped arm intersected with the image bounds in
  `MarkerColor`, opaque, after compositing.** For each cell it takes `ObjectMarkerRects(...)` (the map-rect
  clip, FR-3 stage 1) and intersects each arm with `img.Bounds()` (the output-rect clip, FR-3 stage 2 -
  when `img` is the map extent the two coincide, but both are applied and an `img` smaller than the map
  makes stage 2 truncate), skips any arm that is `image.Rectangle.Empty()` after that, then `SetRGBA`s every
  pixel in the surviving rect to the opaque yellow. `MarkerColor = color.RGBA{0xff,0xd0,0x00,0xff}` (FR-6).
  **Precondition, stated on the exported function:** `img.Bounds().Min` must be the origin. The arms are in
  *map pixel* space, so intersecting them with `img.Bounds()` is only coherent when the image origin is the
  map origin - which every production caller satisfies (`Composite`/`CompositeLit` both allocate
  `image.Rect(0,0,w,h)`). An origin-shifted `img` would compare two coordinate frames and `SetRGBA`'s own
  bounds check would silently swallow the writes, so SC-4's undersized case keeps the origin and shrinks
  only `Max` - otherwise the criterion would pass vacuously by drawing nothing.
  Pixels outside the arms are never touched, so a disabled overlay (function never called) is byte-identical
  (P-3). The undersized-`img` path is what SC-4 uses to exercise the FR-3 output-rect clip and prove no
  synthetic border is drawn at the clip boundary.
  *Rejected:* alpha-blending the marker over terrain - FR-6 says opaque; a blend would make the marker
  colour terrain-dependent and defeat the diagnostic's at-a-glance readability.

- **DD6 - The interactive viewer draws the overlay after the terrain tile loop, transforming each native
  (`cellpx=32`) map-clipped arm through the same camera transform as terrain, with per-rect view culling.**
  A pure `objectScreenRects()` computes, for each `objectCells` entry, `terrain.ObjectMarkerRects(col,row,
  cols,rows,terrain.CellSize)` (native world-pixel arms, map-clipped), and maps each arm to a screen rect
  with the **identical** transform `Draw` applies to a terrain tile (viewer.go): top-left
  `cam.WorldToScreen(minX,minY)`, size `(maxX-minX)*v.cam.Zoom` x `(maxY-minY)*v.cam.Zoom`. It drops any
  rect fully outside `[0,ViewW) x [0,ViewH)` (per-rect culling, FR-6). `Draw` fills each surviving
  `screenRect` with `vector.DrawFilledRect(screen, x,y,w,h, terrain.MarkerColor, false)` - float
  coordinates passed straight through, no independent pixel snapping (FR-6).
  **The float boundary is decided here, not in a task:** `screenRect` holds **`float64`** `X,Y,W,H`, and
  the narrowing to the `float32` that `vector.DrawFilledRect` takes happens at that call site in `Draw`.
  So the transform is exactly the camera's own `float64` arithmetic and SC-6 can compare it against the
  camera contract with no rounding step between them; a `float32` field would make SC-6's oracle
  inexact at almost every reachable zoom (`WheelZoomStep = 1.2`, and `1.2^n` is not float32-exact).
  **Which zoom:** `Draw` reads `zoom := v.cam.Zoom` for the tile path, so the overlay reading the same
  exported `Zoom` field matches the tile transform *unconditionally* - including the unclamped state a
  caller can create by assigning `Camera.Zoom` directly (both are then equally offset from the private
  clamped `zoom()` that `WorldToScreen` uses). That, not "post-`Clamp` they happen to be equal", is why
  the pair is right; SC-6's oracle uses the `Zoom` field for the same reason.
  **FR-3's second stage in the viewer:** the map-rect intersection is done in world space by
  `ObjectMarkerRects`; the output-rect stage is realized by the framebuffer clip, which is what FR-3 itself
  prescribes for this path (`DrawFilledRect` onto the `ViewW x ViewH` screen draws only the on-screen part
  - the drawn pixels *are* the intersection, with no synthetic edge), with fully-outside culling on top.
  FR-3 also states why: clipping the screen rect to integer view bounds would require rounding the float
  screen coordinates, the independent pixel snapping FR-6 forbids. This design therefore *derives* from
  FR-3 rather than interpreting it. The overlay is an independent toggle (`showObjects`), drawn between terrain
  and the future units overlay (FR-4). Factoring the transform into `objectScreenRects` keeps it
  unit-testable without an engine context (as the codebase already does for `advanceAnimation`/`resolveCell`);
  with the overlay off it returns nil, the automatable witness for AC-4 in the viewer (full-frame identity
  needs a window and is the manual AC-6).
  *Rejected:* pre-rasterizing markers into the `(slot,sub)` tile cache - markers are position-specific and
  the cache key carries no position, so this cannot work; *rejected:* running `DrawObjectMarkers` on a CPU
  image and uploading it - it bypasses the camera transform FR-6 requires and would not track pan/zoom.

- **DD7 - Wiring is an opt-in `-objects` flag in each cmd, off by default.** `terraintool -objects`: after
  compositing, build `cells := AnchorCell(o.X,o.Y)` for each `m.Objects`, call `DrawObjectMarkers(img,
  cells, m.Width, m.Height, CellSize*scale)`, and append the literal `, objects N` (`N=len(m.Objects)`) to
  the end of the existing summary line. `mapview -objects`: the flag is passed into `load()` (where `m` is
  in scope and the summary is built); when set, `load()` builds the same cells, calls
  `viewer.SetObjects(true, cells)`, and appends `, objects N` to the summary it returns (so the `-check`
  path prints it too). The token is appended only when the flag is set, so absent the flag the compositor
  output and the summary are the pre-0008 baseline byte-for-byte / character-for-character (P-3, AC-4), and
  the count is not reported (FR-5). `N` is the decoded total `len(m.Objects)` in both cmds - off-map
  anchors are counted even though they are marked nowhere (FR-5 asks for the decoded count, not the drawn
  one). In `mapview` the token lands where `load()` ends its summary: after `tile slots ...` and before
  the `, water speed ...` that `run()` appends - that is the fixed enabled shape SC-7 pins. Both cmds
  build `[]image.Point` cells with stdlib `image` only.
  *Rejected:* on-by-default - the spec requires off-by-default with a byte-identical baseline.

- **DD8 - The viewer receives already-shifted `[]image.Point` cells from `cmd/mapview`, not `alm` types.**
  `pkg/ui` must not import `pkg/formats/alm` (the DAG forbids it, and it currently does not). `cmd/mapview`
  does the `alm.Map.Objects -> []image.Point` wiring once at load and hands the slice to the viewer via
  `SetObjects`. The viewer stores the cells and its map size (already known via the grid) and needs no
  format type.
  *Rejected:* importing `alm` into `pkg/ui` - a DAG violation and unnecessary; the cell list is all the
  viewer needs.

- **DD9 - AC-6 is developer-run against a lawful install; no unit AC drives a real window.** The pure
  geometry (AC-1/2/3), the PNG raster + count (AC-2/4/5), and the viewer transform/cull are unit-tested; the
  live, both-viewers alignment-through-pan/zoom check (AC-6) needs a GOG map and a window and is recorded as
  developer-run evidence in `verification.md`, exactly as 0004's/0005's/0007's live ACs are. No game bytes
  are committed.
  *Rejected:* faking a window in a unit test - it would assert nothing about real alignment and misrepresent
  manual evidence as automated.

## Success criteria

Each maps to a named test (unit) or a developer-run procedure (manual).

1. **SC-1 (FR-2, AC-1)** - `AnchorCell` shifts `X,Y` by 8; over a fractional `/256` cell list (some
   off-map) each in-map object yields a cross at `(X>>8,Y>>8)` and each off-map anchor yields nil; no
   footprint is ever produced (always <=2 arms). *Test:* `TestAnchorCellAndOffMap`.
2. **SC-2 (FR-6, FR-3, AC-2)** - `ObjectMarkerRects` returns exactly the FR-6 rectangles at native
   `cellpx=32` and at representative downscaled odd/even `cellpx` (32, 16, 17, 15) - at all of which the
   cross fits inside its cell (no clipping). Plus a `cellpx <= 2` case on a tiny map (e.g. a `1x1` map at
   `cellpx=1` or `2`, where the arms overrun the map on every side), where the map-rect intersection
   **truncates** each arm to the boundary via `min`/`max` only (no synthetic border), proving the FR-3
   map-rect clip. *Test:* `TestObjectMarkerRectsGeometry`.
3. **SC-3 (FR-2, AC-3, P-1, P-2)** - `cellpx<1` yields nil; an **off-map** extreme-magnitude anchor (huge
   `col`/`row`, small `cols`/`rows`) yields nil (no panic, no allocation); an **in-map** extreme-magnitude
   anchor (huge `col` within a large `cols` at a large `cellpx`) yields <=2 arms with no panic and no
   coordinate-magnitude-proportional allocation. *Test:* `TestObjectMarkerRectsExtremes`.
4. **SC-4 (FR-1, FR-3, FR-6, AC-2, P-1)** - `DrawObjectMarkers` fills exactly the clipped arm pixels of
   each in-map cell in `MarkerColor` over an RGBA and leaves every other pixel untouched; off-map cells
   draw nothing; and with an **`img` smaller than the map extent but still anchored at the origin** (DD5's
   precondition) the arms are truncated to the image bounds (the FR-3 output-rect clip) with no synthetic
   border and no out-of-bounds write. *Test:* `TestDrawObjectMarkers`.
5. **SC-5 (FR-1, FR-4, FR-5, AC-4, AC-5, P-3)** - `terraintool -objects` marks the anchor cell of every
   **in-map** object yellow - an off-map anchor is marked nowhere (FR-3, DD5) - and its summary reports
   `objects N` with `N = len(m.Objects)`, the decoded total including off-map anchors (DD7); without
   `-objects` the PNG is byte-identical to the **compositor's own output for the same inputs**
   (`Composite`/`CompositeLit` called directly with the same map, tileset, light and scale - not merely a
   second no-flag run, which would be tautological) and the summary keeps its existing shape. *Test:*
   `TestRenderObjectsOverlay` (+ the existing terraintool tests stay green).
6. **SC-6 (FR-1, FR-3, FR-4, FR-6, AC-4 viewer)** - at a couple of pan/zoom positions the viewer's
   `objectScreenRects` equals, for each in-map object, the arm transformed by the camera - the oracle being
   the **independent** `cam.WorldToScreen(corner)` + `cam.Zoom`-scaled size of the known native arm, in
   `float64` (derived from the camera contract per DD6, not by re-running `objectScreenRects`), i.e. the
   same transform a terrain tile gets; rects fully outside `[0,ViewW) x [0,ViewH)` are culled; and with the
   overlay off (or no cells) it returns nil (the AC-4 viewer witness). *Test:* `TestObjectScreenRects`.
7. **SC-7 (FR-1, FR-4, FR-5, AC-4, AC-5)** - `mapview -objects` reports `objects N` (`N = len(m.Objects)`,
   off-map anchors included) on the `-check` summary, in the DD7 position, and passes the cells to the
   viewer; without it the summary is the pre-0008 shape. *Test:* `TestObjectsFlag`.
8. **SC-8 (FR-2, DAG)** - `pkg/render/terrain` still imports only stdlib and `pkg/ui` adds only an ebiten
   sub-package; the fail-closed DAG check stays green. *Test:* the existing `internal/archtest` live-tree
   check.
9. **SC-9 (FR-5, FR-6, AC-6)** - a developer run loads a real GOG map with placed objects in both viewers with the
   overlay on: markers sit on the objects' terrain cells and stay aligned through pan/zoom, and the reported
   count equals the map's object count. *Method:* developer-run (manual), recorded in `verification.md`; no
   game bytes.

## Risks (product)

- **R-1 - the viewer overlay is not pixel-identical to the PNG overlay.** FR-6 routes the viewer's markers
  through the float camera transform (as terrain is), while the PNG uses integer `cellpx` math, so at a
  given zoom a marker can differ from the PNG by up to a pixel at a boundary. This is intended - FR-6
  requires the viewer to use the same camera transform as terrain with no independent snapping, so a marker
  tracks its terrain cell rather than a fixed pixel grid. *Mitigation:* documented; SC-6 asserts the viewer
  transform matches the terrain-tile transform (the property that matters), and AC-6 checks alignment with
  terrain cells through pan/zoom, not PNG-equality.
- **R-2 - the reported count could be mistaken for a raw record count.** A minority of type4 records carry
  an 8-byte extension, so `type4_size/20` overcounts; but `alm` already decodes the adaptive walk, so
  `len(Map.Objects)` is the true decoded object count. *Mitigation:* FR-5 reports `len(Map.Objects)`
  directly (never a recomputed size/20); AC-6 confirms it equals the map's object count as the editor/game
  shows. Object 0's anchor is valid despite the section-identity overlay (research `ALM-OBJ-019`), so it is
  treated like any other (counted, and marked whenever its anchor is in-map).
- **R-3 - markers are one pixel off-centre at even thickness/odd scales.** The centered-strip formula
  `[c-floor(t/2), c-floor(t/2)+t)` biases an even-`t` strip one pixel toward the low side of the centre
  pixel (e.g. `cellpx=17 -> t=2`). This is the exact, deterministic FR-6 contract, not a defect.
  *Mitigation:* AC-2 pins the exact rectangles at odd/even `cellpx`; documented as intended. The native
  `cellpx=32` render (`t=3`, odd) is symmetric.
