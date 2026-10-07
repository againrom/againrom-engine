# Spec — height-projected placements

## Problem and current behaviour

Two viewers show diagnostic markers for placed objects (0008) and placed units (0009) over
height-displaced terrain (0012/0013), and neither puts a marker at the height its cell's relief
actually reaches.

- `pkg/ui.Viewer` (shared by `cmd/mapview` and the game front-end `cmd/againrom`) selects flat
  terrain whenever either overlay is enabled, regardless of whether the altitude grid is valid
  (0013 FR-2): `Mode()` silently switches to flat, no error, nothing printed.
- `cmd/terraintool` raises no such conflict: `-objects`/`-units` already compose with the default
  height-displaced raster. But every marker sits on the *un-displaced* lattice, shifted only by the
  whole image's own vertical origin — on a sloped map it floats above or sinks below the relief it
  marks, a limitation 0012's own code names as this story's contract.

This story adds a height lookup for a marker's anchor cell and applies it in both places: the
overlay-forces-flat rule is lifted, and the PNG tool's markers are displaced onto the surface.

Frozen: object/unit anchor-cell decoding (0008/0009), ALM decoding, the 0012 projection geometry
and terrain raster, 0014 lighting, water, and every camera/culling contract other than the marker
offset this story adds.

## Height lookup contract

`terrain.Projection` (0012) already exposes `Altitude(c,r) int`: a cell mesh vertex's signed
native-pixel altitude, with an out-of-grid `c`/`r` clamped to the nearest valid vertex (0012's own
far-edge convention — never a read of zero).

`Projection` gains `AnchorHeight(col, row int) int`: the signed native-pixel height a marker
anchored in cell `(col,row)` is lifted by — the mean of that cell's four corner altitudes, summed
and divided by 4 with Go's native truncating-toward-zero integer division:

```
h00 = Altitude(col,row)     h10 = Altitude(col+1,row)
h01 = Altitude(col,row+1)   h11 = Altitude(col+1,row+1)
AnchorHeight(col,row) = (h00 + h10 + h01 + h11) / 4
```

It is total. `col`/`row` are clamped into the grid's own cell range **before** the four corner
indices are formed — so an off-grid anchor returns the nearest edge cell's mean, and `col+1` can
never wrap: a raw `col` at the integer maximum would otherwise overflow and leave the four "corners"
straddling opposite edges of the grid. It is pure — a function of the borrowed altitude slice and
the two indices alone — and needs no bounds test of its own.

## Marker displacement contract

Both the placed-object overlay (0008) and the placed-unit overlay (0009) already resolve every
marker's anchor to one integral cell, `AnchorCell(x,y) = (x>>8, y>>8)`, discarding any sub-cell
fraction, before any marker rectangle is built — and both overlays' existing cross geometry
(`ObjectMarkerRects`/`UnitMarkerRects`) is centred on that same cell. There is no structure/unit
distinction to make: one reference point, one offset rule, for both kinds.

In displaced mode, a marker anchored at cell `(col,row)` is displaced by translating **every**
rectangle its flat geometry produces — vertically only, nothing reshaped, added or removed — by
that cell's `AnchorHeight`, using the same sign and canvas-origin convention 0012/0013 already
apply to terrain: a positive height moves a point up the image (a smaller destination row).

- **In `pkg/ui`'s world space** (shared by `cmd/mapview` and the game front-end): flat mode's world
  Y for cell row `row` is `row*CellSize + CellSize/2`. Displaced mode replaces this with
  `row*CellSize + CellSize/2 - AnchorHeight(col,row) - proj.MinV`, matching `WorldCorner`'s own
  `V(c,r) - MinV` shift. Added in native pixels before the camera transform, exactly where
  `WorldCorner`'s callers add theirs, so it needs no rounding of its own. Flat mode is unchanged.
- **In `cmd/terraintool`'s output space**: the existing whole-image shift, `markerOffsetY =
  -render.OriginY * scale`, stays, and each marker additionally shifts by `-AnchorHeight(col,row) *
  scale`. `scale` is always a positive integer multiplier of `CellSize`, so this is an exact
  multiply — no rounding division is introduced. `-flat` output never goes through `AnchorHeight`.

Draw order is unchanged in both modes: terrain, then object markers, then unit markers; within a
layer a later marker owns overlapping pixels (0009 FR-4). Displacement moves vertical position
only, never that order — on a slope two markers can receive different offsets, so a higher marker
may newly overlap a lower one, resolved by the same painter order.

## Mode-selection change

`Viewer.Mode()` selects displaced whenever the altitude grid is valid (`proj != nil`), independent
of `showObjects`/`showUnits` — superseding 0013's rule that either overlay forces flat (FR-2).
`Viewer.Lit()`'s predicate is untouched (it already ignores the overlay flags, 0014 FR-4), which is
why FR-8 belongs here: level grid and projection share one guard, so once validity alone selects
displaced, *flat* and *lit* cannot co-occur unless something selects flat deliberately.
`cmd/mapview -objects`/`-units` over a valid altitude grid now renders displaced with projected
markers instead of silently falling back to flat; the game front-end, which enables neither
overlay itself, is unaffected either way.

## Functional requirements

- **FR-1** `Projection` MUST expose `AnchorHeight(col, row int) int` per the *Height lookup
  contract*, total over any `col`/`row`, pure over the borrowed altitude slice, and MUST NOT
  mutate it.
- **FR-2** `Viewer.Mode()` MUST select `ModeDisplaced` whenever the altitude grid is valid,
  independent of `showObjects`/`showUnits`, superseding 0013 FR-2; flat remains selected only when
  the altitude grid is invalid or absent.
- **FR-3** In displaced mode, `pkg/ui` MUST translate every rectangle of an object or unit marker
  vertically by `-AnchorHeight(col,row) - proj.MinV` world pixels relative to its flat-mode
  position, applied before the camera transform; horizontal geometry, rectangle count and colour
  are unchanged. Flat mode MUST keep byte-identical marker rectangles to pre-story behaviour.
- **FR-4** `cmd/terraintool`'s object/unit overlays MUST, when composed onto the default (non-
  `-flat`) projected geometry, additionally translate every marker rectangle vertically by
  `-AnchorHeight(col,row) * scale` output pixels on top of the existing canvas-origin shift; `-flat`
  output MUST be unaffected. The height MUST translate the rectangles a glyph builder returns, and
  MUST NOT be folded into the batch offset the canvas shift uses: that offset also positions the
  map-extent rectangle every arm is clipped against, so a height inside it moves the clip along with
  the marker and the on-map test silently stops meaning what it says.
- **FR-5** Marker draw order (terrain → objects → units; later marker wins on overlap) MUST hold in
  displaced mode exactly as in flat mode, in both `pkg/ui` and `cmd/terraintool`.
- **FR-6** In displaced mode, marker rectangles MUST be culled/clipped by the same view/camera
  intersection logic flat markers use, applied *after* the height offset, with no under-coverage.
- **FR-7** `AnchorHeight` and the per-marker offset it feeds MUST be pure functions computable
  without a graphics context; only the final draw call needs one.
- **FR-8** The viewer MUST expose an explicit way to select flat mode over a **valid** altitude grid,
  and `cmd/mapview` MUST expose it as `-flat`. Without it FR-2 leaves flat meaning *no altitude data*,
  and the level grid is built under that same guard — so flat would imply unlit and the flat path's
  own lighting would be unreachable. Under this selection the map stays lit, markers stay
  flat-positioned, and FR-3's other flat invariants hold. `cmd/againrom` gains no flag: like
  `-unshaded`, this is a developer diagnostic.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | synthetic grids (flat, slope, negative heights); cells interior, edge, corner, at/past `width`/`height` | `AnchorHeight` called | equals the hand-computed truncating mean of the four corner altitudes; a past-edge cell returns the nearest edge cell's mean; a flat cell returns that altitude |
| AC-2 | unit | a borrowed altitude slice | many `AnchorHeight` calls | slice byte-for-byte unchanged |
| AC-3 | unit | valid Heights, either/both overlays requested | a `Viewer` is built | `Mode()` reports Displaced (supersedes 0013 AC-2/AC-2a); `Lit()` unaffected |
| AC-4 | unit | Heights absent/wrong-length, both overlays requested | a `Viewer` is built | `Mode()` reports Flat; world size/overlay geometry/water/light byte-identical to pre-story |
| AC-5 | unit | a sloped grid; an object and a unit anchored in the same cell | rectangles built | both equal flat rectangles translated by `-AnchorHeight(col,row) - proj.MinV`; extents/count/colour unchanged; both kinds show the same offset |
| AC-6 | unit | two markers of different `AnchorHeight` whose offset rectangles overlap | draw order inspected | objects precede units; later rectangle overlaps earlier, matching flat order |
| AC-7 | unit | `terraintool` compose of a sloped map, `-objects`/`-units`, several `-scale`, with/without `-flat`, **including an anchor within an arm's reach of the map's top edge whose lift exceeds that reach** | composed | projected: pixels translated by `-AnchorHeight(col,row)*scale` beyond the canvas shift, and the edge anchor's arm is clipped at the map extent, not at an extent moved by its own height — an implementation that folds the height into the batch offset fails this case; `-flat`: byte-identical to pre-story |
| AC-8 | integration | `mapview -check` / `terraintool` summary, synthetic+representative inputs, every overlay/`-flat` combination | invoked | summary text byte-identical to pre-story |
| AC-10 | unit | a valid, sloped altitude grid, the flat selection applied, overlays both on and off | a `Viewer` is built and `mapview -check` run | `Mode()` reports Flat while the map stays **lit**, markers carry no height offset, world size is the flat `w*32×h*32`, and the summary text is byte-identical with and without the flag |
| AC-9 | manual | representative maps with objects/units | opened displaced with both overlays in `mapview`/front-end, rendered by `terraintool -objects -units` | markers sit on the surface, not floating; relief/water/light match 0012/0013/0014; fidelity gaps recorded |

Error cases: none new. Every parsed map still
yields flat or displaced exactly as 0012/0013 decide, unaffected by this story.

## Derived properties

- **P-1** (invariant) `AnchorHeight(col,row)` is a pure integer function of the borrowed altitude
  slice; on a flat cell (four equal corner altitudes `h`) it returns exactly `h`.
- **P-2** (bound) `AnchorHeight` lies between the min and max of its cell's four corner altitudes,
  so a marker never displaces beyond the relief of the cell it anchors in.
- **P-3** (negative-invariant) For inputs that select flat mode, no displacement or origin shift is
  applied to any marker: world size, overlay rectangles, order, water and lighting are byte-for-byte
  the flat scene, and every command's summary text is unchanged.
- **P-4** (invariant) Displacing a marker translates its rectangles only vertically; rectangle
  count, each rectangle's width, and its X extent are identical to the flat marker's.
- **P-5** (completeness) In displaced mode, every marker whose vertically-offset rectangle
  intersects the view (or, in `cmd/terraintool`, the output image) is included in the drawn/clipped
  set; the offset is applied before culling/clipping.
- **P-6** (invariant) Constructing or drawing displaced markers never mutates the borrowed altitude
  slice.

## I/O examples

```text
mapview -assets <dir> -map Cross.alm -objects -units
# displaced terrain with object/unit markers projected onto it (was: flat)

terraintool render -assets <dir> -map Cross.alm -out out.png -objects -units
# height-displaced terrain with markers sitting on the relief (was: floating on the flat lattice)

terraintool render -assets <dir> -map Cross.alm -out flat.png -flat -objects -units
# unchanged: flat raster, flat markers

mapview -assets <dir> -map Cross.alm -objects -units -check
# mapview: Cross 256x256 cells (65536), tile slots .../..., objects N, units M, water speed ...
```

## Constraints and alternatives

| Choice | Observable trade-off | Decision |
|---|---|---|
| Mean of the anchor cell's four corner heights, truncating divide by 4 (the engine's decoded object-sprite lift) | decoded for the object path; the engine's unit draw ignores the cell and places from the unit's own raw position, so on unit markers it is ours by choice; free since both marker kinds already collapse to one cell | selected |
| A general bilinear sampler over an arbitrary 8.8 sub-cell fraction | generalizes to a future fractional-anchor marker, but unexercised today | rejected, deferred |
| Translate the marker's existing rectangles as a whole, after they are built | keeps `ObjectMarkerRects`/`UnitMarkerRects`'s tested contract untouched | selected |
| Drop `Mode()`'s overlay check outright, rather than narrow it | `AnchorHeight` is total over any valid projection; no case remains it cannot serve | selected |

This story is not claimed pixel-identical to the original: the engine's terrain mesh and
object-lift mean agree on only a minority of sloped cells, and its unit draw takes no cell
altitude at all, placing a unit from its own raw position; this story reconciles none of that,
only places a diagnostic marker at its own anchor cell's height. Tests use synthetic maps only.

## Out of scope

- Object identity/art/footprints, unit sprites/art — downstream of the class registries and
  static-data formats (0008/0009's own scope, unchanged).
- A general fractional-position height sampler for a future marker that keeps a sub-cell fraction.
- The engine's exact tie-break for a negative corner-sum mean, and reconciling its two altitude
  models — no pixel fidelity to the original is claimed.
- Per-corner footprint projection, slope occlusion, depth buffering.
- Changes to ALM decoding, the 0012 projection geometry, terrain rasterisation, water, or 0014's
  lighting.
