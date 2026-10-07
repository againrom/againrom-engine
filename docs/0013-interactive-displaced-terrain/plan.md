# Plan — interactive displaced terrain

## Shape

No new package. `pkg/render/terrain` owns the world-space geometry and the culling range,
`pkg/render/camera` is told a world height it does not derive, `pkg/ui` gains the second draw path
and the mode that selects it, `pkg/game` passes the altitudes it already holds.

## DD-1 — the camera is told its world height; its constructor does not move

`internal/archtest` holds `pkg/render/camera` to stdlib alone (`dag.go:48`, empty allow-list), and
the package doc states that model is deliberately free of the windowing engine and every formats
package. So the camera cannot ask a projection how tall the world is.

`New` keeps its signature and initialises the world height to `Rows*CellSize`; the camera gains that
field, `WorldH()` returns it, and `SetWorldHeight(h float64)` re-clamps like every other mutator.
FR-2 makes the mode mutable, so the mutator is needed regardless — and once it exists the
constructor needs no new argument, which is why all nine call sites in `camera_test.go` and the one
in `viewer.go:91` compile and assert unchanged (SC-1). `Cols`/`Rows` stay the tile grid the column
clip works against — a different quantity that happened to coincide.

Rejected. **Widening `New`**: it moves ten call sites to a signature nobody needs and makes SC-1's
"unmodified" claim false, while proving nothing `SetWorldHeight` does not. **A DAG edge
`camera -> render/terrain`**: buys nothing the caller cannot pass, costs the property that keeps the
camera unit-testable without a map. **A second `DisplacedCamera` type**: duplicates clamp, pan and
zoom, and 0005 shipped a single camera model precisely so the viewer and the game cannot drift.

## DD-2 — the culling range belongs to the projection, not the camera

The camera's row band comes from `c.Y` (`camera.go:163`), a **translated** coordinate in displaced
mode, while a pad computed from altitudes alone lives in native `V`. The two are offset by `MinV` —
up to the 255-unit altitude spread, eight tile rows. Padding `VisibleTiles()` therefore under-covers
wherever `MinV != 0`: a 1x8 grid pushed -128 for its first four rows drops rows 3 and 4 at `Y = 0`,
and over 20 000 random grids and views the composition misses tiles 22% of the time. No pad repairs
it, because the base range is the wrong quantity.

So the camera gets no padded range. The row range moves to `pkg/render/terrain`, where `MinV` and
the altitude extremes already live:

```text
RowRange(top, bottom float64) (r0, r1 int)   // a world-Y window -> rows, clipped to [0, Height]
```

with `a = top + MinV`, `b = bottom + MinV`, and — since row `r`'s four corners span `V` within
`[r*32 - MaxH, (r+1)*32 - MinH]` — `r0 = floor((a + MinH)/32) - 1` and `r1 = ceil((b + MaxH)/32)`.
That is sufficient by the intersection inequality, and it sits inside the spec's two-sided bound
because `floor((a + MinH)/32) >= floor(a/32) - ceil(down/32)` and symmetrically above.

The viewer takes the window from `cam.ScreenToWorld(0,0)` and `cam.ScreenToWorld(ViewW, ViewH)` —
both already exported, so the camera needs no new reader — and the **columns** from
`cam.VisibleTiles()` unchanged: exact, because no altitude term reaches a destination column
(`TERR-GEOM-035`), which makes AC-6a's column identity hold by construction.

## DD-3 — the projection answers in world space

Three additions, all pure and all in `pkg/render/terrain`:

- `WorldCorner(c, r int) (x, y int)` = `(c*CellSize, Vertex(c,r) - MinV)`. The subtraction of `MinV`
  is currently open-coded in the compositor; giving it a name is what stops the viewer inventing a
  second translation, and FR-7 forbids exactly that.
- `MinH`/`MaxH`, the grid's signed altitude extremes, recorded in the pass `Project` already walks
  for `MinV`/`MaxV`.
- `RowRange` above.

Sign, written so it cannot be read both ways: a **positive** altitude subtracts from `V`, so a
raised tile is drawn **above** its nominal band and is therefore reached by looking at **larger**
row indices. `MaxH` widens `r1`; `MinH`, being negative, widens `r0`.

Rejected. **A new `pkg/render/displace` package**: `pkg/ui` would reach it for free through the
`pkg/render/` wildcard, but the geometry would live in two packages and FR-7's single convention
would rest on discipline instead of one owner.

## DD-4 — altitudes ride in `terrain.Grid`

`Grid` gains `Altitudes []uint8`. Every construction site in the tree uses keyed literals, so the
field is source-compatible and no test signature moves; the one production site
(`pkg/game/mapload.go:51`) already holds `m.Altitudes` and drops it today.

The compositors keep taking altitudes as their own parameter and do **not** read `g.Altitudes`: the
field is the viewer's channel alone, and unifying the two is a later story's work. This is the one
place FR-7's convention rests on a caller rather than a type — recorded, not fixed.

`ui.validateGrid` is untouched: it validates the tile grid, whose failure is fatal. An altitude grid
of the wrong length is **not** an error — it selects flat mode — so it is checked where the mode is
decided, not where the map is rejected.

Rejected. **A parameter on `NewViewer`**: 12 test call sites and three helpers would move for a
value that is part of the map, not part of the viewer's configuration.

## DD-5 — the mode is computed, the world is synced at every entry

`Viewer.Mode() Mode` returns `ModeDisplaced` when a projection was built and neither overlay is
enabled, `ModeFlat` otherwise. Only the projection is stored and nothing here can replace the grid
after construction, so neither half of the predicate goes stale: the overlay half is read live.

`NewViewer` builds the projection once — per frame would re-walk every vertex of a 256x256 map for
a value that cannot change — and then calls `syncWorld()`. So do `SetObjects` and `SetUnits`. That
`NewViewer` is one of the call sites is load-bearing, not tidiness: `pkg/game/frontend.go:106`
reaches `NewViewer` and never touches either overlay, so syncing only on an overlay toggle would
leave the game displaced over a camera still clamped to `Height*32`, clipping up to 255 world pixels
off the bottom of the map and failing FR-3 and AC-9 on the one path the story exists for.

`syncWorld()` passes `CanvasHeight()` in displaced mode and `Rows*CellSize` in flat, then re-clamps.
It does not preserve the view point across a mode flip: the position stands and the clamp decides.
AC-2a asks only that the extent follow, and re-anchoring here would be UX nobody specified.

## DD-6 — one `DrawTriangles` call per tile

A displaced quad is not a parallelogram in general, so an affine `GeoM` cannot express it and
`DrawImage` is unavailable.

Vertices are assembled by a pure `tileVertices(cam, proj, tx, ty) [4]ebiten.Vertex`. A vertex is a
struct literal needing no graphics context, so FR-6's seam is that function rather than the draw
call, and SC-4 has something to assert against instead of restating DD-3.

Per tile, vertex order TL, TR, BL, BR:

- `DstX/DstY` = `cam.WorldToScreen` of `proj.WorldCorner`, submitted unrounded; the only rounding is
  the `float32` the vertex takes, which `overlay.go` already accepts for its rects.
- `SrcX/SrcY` span the cell image's own `0..32` bounds.
- `ColorR/G/B/A = 1`. **Not** the zero value: `ebiten.Vertex` documents `ColorA == 0` as fully
  transparent, which draws the whole terrain invisible.
- indices `{0,1,3, 0,3,2}` — `TL,TR,BR` and `TL,BR,BL`, whose shared edge is `TL-BR`, the diagonal
  the spec fixes. `{0,1,2, 1,3,2}` shares `TR-BL` and is the wrong one.
- options: `Filter: FilterNearest`, every other field zero. `DrawTrianglesOptions` in v2.9.9 carries
  no winding or culling field at all, so an inverted cliff quad rasterises with nothing set; the
  `FillRule` that actually delivers that is deprecated as of v2.9 and is not relied on. `Address`
  stays `AddressUnsafe`, matching what `DrawImage` already does for flat mode — the only clamp
  offered is `AddressClampToZero`, which would punch transparent pixels at tile seams.

Loop order is `ty` ascending then `tx`, so a later tile owns any overlapping pixel (SC-9). On a
cliff that order is the only thing resolving the overlap, and this change makes overlaps common.

Per-tile calls rather than a batch: batching is out of scope, and a batch must respect
`MaxVertexCount` and break runs at every source-image change — new failure surface for a performance
property the contract does not ask for. The per-`(slot, sub-cell)` GPU cache is reused unchanged, so
both modes upload the same textures and P-4 holds by sharing the resolution path rather than by
asserting two copies agree.

## DD-7 — what is pinned before it changes

Brownfield: the camera, the viewer draw loop, the one load path. Pinned in the same commit that
changes them — every flat construction still yields `WorldH() == Rows*CellSize`, and a flat viewer
reports `ModeFlat` with its extent, visible range, water state and overlay passes unchanged.

`overlay_test.go:230` and `unit_overlay_test.go:208` build a 10x10 world into a 320x320 view, i.e.
exactly `world == view`, and look like a two-sided guard on that number. They are not: `clampAxis`
centres when `world <= view` (`camera.go:92`), and centring a world equal to the view also yields 0,
so they catch an undershoot and pass an overshoot. DD-1's exactness rests on SC-1 instead.

## Risks

- **A displaced world can be shorter than the flat one.** `MaxV - MinV` is not bounded below by
  `Height*CellSize`; a two-row grid with altitudes 0 and 20 spans 44 rather than 64. Below the view
  height the camera switches from clamping to centring, a path a real flat map never reaches.
  Covered by AC-3's short grid and AC-5.
- **`RowRange`'s anchoring is invisible to a containment test taken near the middle of a map.** The
  witness must be grids with `MinV != 0` — push-only as well as raise-only — at views against both
  edges. A raise-only grid with a flat first row is exactly the family where the omitted `MinV` term
  vanishes and a wrong implementation passes.
- **Draw-call volume at 256x256 zoomed out.** ~6 000 visible tiles is ~6 000 calls per frame. If it
  is visibly slow, that is a finding for a later story and is recorded, not fixed here.
- **The linked test binary trips Windows Defender.** `pkg/render/terrain`'s test executable is
  intermittently quarantined on this machine — a hash-specific false positive cleared by
  `-ldflags=-s`. Not a test failure, and not to be reported as one.

## Success criteria

| ID | Criterion | Serves |
|---|---|---|
| SC-1 | `New` is unchanged, every pre-existing camera and viewer test passes unmodified, and a flat construction reports `WorldH() == Rows*CellSize` | FR-1, P-2 |
| SC-2 | `Mode()` is `ModeDisplaced` exactly for a valid grid with no overlay enabled, and follows an overlay enabled or disabled after construction | FR-2 |
| SC-3 | After construction with no overlay, the camera's world is `Width*CellSize x CanvasHeight()` taken from the projection, including where that is smaller than the flat height | FR-3, P-1 |
| SC-4 | Every `tileVertices` corner equals `WorldToScreen(WorldCorner(..))` of its vertex, with the `TL-BR` index list, the `0..32` source span and `ColorA == 1`, at interior, right-edge, bottom-edge and corner tiles | FR-4, FR-7 |
| SC-5 | `RowRange` contains every row whose corner box meets the window, on grids with `MinV` zero and non-zero at both edges; lies inside the two-sided bound; and the columns come from the unmodified `VisibleTiles` | FR-5 |
| SC-6 | Mode, extent, corners, vertices and range are computed in tests that open no window | FR-6 |
| SC-7 | Constructing and drawing a displaced viewer leaves the altitude slice byte-identical | P-5 |
| SC-8 | Every command's `-check` output is byte-identical to the pre-change output in every overlay combination | FR-1, AC-7 |
| SC-9 | The displaced loop visits tiles row-major, so a later tile overwrites an overlapping pixel | FR-4 |
| SC-10 | The shipped-map run records each map's outcome, the `Kids`/`61` comparison and every fidelity limitation observed, naming anything not run as not run | AC-8, AC-9 |
