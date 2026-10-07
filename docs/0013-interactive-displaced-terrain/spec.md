# Spec — interactive displaced terrain

## Problem and current behaviour

The windowed viewer draws every terrain cell as an axis-aligned 32-pixel square. A map's altitudes
reach the screen not at all: cliffs and slopes are flat, while the same map rendered to a PNG
already displaces its vertices. This story draws that same projection in the window, so the game
shows relief rather than a lattice of squares.

Everything the viewer does today for a map that keeps rendering flat is frozen; FR-1 names the set.
The viewer applies no terrain shading today, so this story neither preserves shading nor adds any,
and it changes neither map decoding, nor the shared projection geometry, nor the PNG paths.

## Rendering mode selection

The viewer renders in exactly one of two modes. The mode is a pure function of the viewer's current
state — the altitude grid it holds, and whether either diagnostic overlay is enabled — and not a
decision taken once, because an overlay may be enabled after the viewer exists:

```text
displaced  when  the altitude grid is valid  AND  no diagnostic overlay is requested
flat       otherwise
```

A **valid** altitude grid is one with exactly `Width*Height` entries over positive dimensions; the
shared projection is total over such a grid, so that is the whole predicate. The decision reads only
that validity and the overlay flags —
never the altitude *values*, so an all-zero valid grid still selects displaced mode.

An overlay forces flat because no marker can follow the terrain until a later story lifts it, and a
flat marker over displaced ground would sit at the wrong height. The game front-end enables no
overlay, so the game displaces.

## Displaced geometry

Native coordinates come from the shared projection and are not redefined here: for a `Width x
Height` map with signed row-major altitudes, `V(c,r) = r*32 - h(c,r)` over the closed vertex domain
`0..Width x 0..Height`, and the native canvas spans `[MinV, MaxV)` — the least and greatest `V` over
that domain. A cell owns four corners, so that domain reaches one vertex past the grid on each far
edge, where no altitude exists; those vertices take the nearest one that does,

```text
h(c,r) = altitudes[ min(r, Height-1) * Width + min(c, Width-1) ],  read signed
```

which is the shared projection's own rule, not a second convention.

**World coordinates.** The camera's world is the native canvas translated so its top is world Y
zero:

```text
worldCorner(c,r) = ( c*32 , V(c,r) - MinV )
```

World width stays `Width*32`; world height becomes `MaxV - MinV`, which equals `Height*32` exactly
when every altitude is equal.

**Tile drawing.** In displaced mode a tile `(tx,ty)` is drawn as the quadrilateral whose four
corners are `worldCorner` of `(tx,ty)`, `(tx+1,ty)`, `(tx,ty+1)` and `(tx+1,ty+1)`, textured from
that tile's resolved 32x32 source cell with nearest sampling, and placed on the screen by the same
world-to-screen transform flat terrain uses, with corner coordinates submitted unrounded as flat
mode submits its own. The quad is split along its **top-left to bottom-right** diagonal; which
diagonal is used changes the interior of a non-parallelogram tile, so it is fixed here rather than
left to the drawing call. Tiles are drawn in row-major order — `ty` ascending, then `tx` — so a
later tile owns any overlapping pixel. Water phase substitution applies exactly as in flat mode. A
tile whose source cell is unavailable draws the same placeholder fill as flat mode, through the
same quad.

An altitude step steeper than one cell edge — `h` rising by more than 32 between two vertically
adjacent vertices — puts a tile's bottom corner above its top one and inverts the quad. Such a tile
MUST still be drawn, with no winding or back-face test removing it; the painter order above is what
resolves it against its neighbours. Shipped maps carry steps far past 32, so this is the ordinary
case at a cliff rather than a degenerate one.

## Camera and culling

- **Extent.** In displaced mode the camera's world is `Width*32` wide and `MaxV - MinV` tall.
  Clamping, centering, panning at a constant world-space rate and cursor-anchored zoom behave
  exactly as they do today, now over that world.
- **Culling, bounded on both sides.** The tiles drawn each frame MUST include every tile whose
  displaced quad intersects the view rectangle: under-covering is a defect. Over-covering is
  permitted but not unlimited, or "draw the whole map every frame" would satisfy the requirement.
  Writing `up = max(0, largest altitude)` and `down = max(0, -smallest altitude)` over the grid, and
  `[r0, r1)` for the rows a flat map covers over the view window measured in **native** `V`
  coordinates — the world window shifted back by `MinV`, not the world window itself — the drawn row
  range MUST lie within `[r0 - ceil(down/32) - 1, r1 + ceil(up/32) + 1)`, clipped to the map. The
  horizontal range is unaffected and MUST NOT widen, because no altitude term reaches a destination
  column.

## Functional requirements

- **FR-1** For every input selecting flat mode, the viewer MUST retain its current world size,
  panning, zoom, clamping, centering, visible-tile range, water cycling, overlay geometry and draw
  order, and every command's summary text, with no drift.
- **FR-2** The viewer MUST render in displaced mode exactly when its altitude grid is valid and
  neither diagnostic overlay is enabled, MUST expose the current mode for inspection, and MUST
  follow a change to either overlay's enablement made after it was constructed.
- **FR-3** In displaced mode the camera's world MUST be `Width*32 x (MaxV - MinV)`, obtained from
  the shared projection, and every camera operation MUST work over that world.
- **FR-4** In displaced mode each tile's four drawn corners MUST equal `worldCorner` of its four
  native vertices, sampled nearest from that tile's 32x32 source cell, drawn in row-major painter
  order, with water substitution applied as in flat mode.
- **FR-5** In displaced mode the drawn tile set MUST include every tile whose displaced quad
  intersects the view rectangle, MUST NOT extend beyond the two-sided bound above, MUST NOT widen
  horizontally, and MUST stay within the map's tile bounds.
- **FR-6** Mode selection, world extent, tile-corner world coordinates and the padded culling range
  MUST be computed by pure functions that run without a window; only the final draw call may
  require a graphics context.
- **FR-7** Displaced rendering MUST obtain all vertex geometry from the shared projection, MUST NOT
  introduce a second height convention, and MUST NOT mutate the altitude slice it borrows.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | maps with the altitude grid absent, with an invalid-length grid, and with a valid grid but an overlay enabled | a viewer is built and drawn | it is in flat mode, world size is `Width*32 x Height*32`, and overlay state, draw order and water state match the pre-change viewer |
| AC-2 | unit | a map with a valid altitude grid and no overlay enabled | a viewer is built | it is in displaced mode |
| AC-2a | unit | a displaced viewer | either overlay is enabled, then disabled again | it reports flat while the overlay is on and displaced again once it is off, and the world extent follows the mode both ways |
| AC-3 | unit | flat, single-axis-slope, asymmetric and negative-altitude synthetic grids, plus one whose vertices span **less** than `Height*32` | the displaced world extent is computed | world width is `Width*32` and world height is `MaxV - MinV` from the shared projection; an all-zero grid yields `Height*32`; the short grid yields its own smaller height and the camera centres that axis rather than clamping it |
| AC-4 | unit | the same synthetic grids | tile-corner world coordinates are computed for interior, right-edge, bottom-edge and corner tiles | each corner equals `(c*32, V(c,r) - MinV)` |
| AC-5 | unit | a displaced world and varied camera positions, zooms and view sizes | the camera clamps, pans and zooms | the shipped camera invariants hold over the displaced world, whether it is taller or shorter than the flat one: in bounds, or centered on an axis smaller than the view; the world point under the cursor is preserved by zoom away from a clamped edge |
| AC-6 | unit | synthetic grids including a 127-raise cliff and a negative-push cliff, at many camera positions and zooms | the visible tile range is computed | every tile whose displaced quad intersects the view rectangle is inside the range, and the range stays within `[0,Width) x [0,Height)`. A quad counts as intersecting when the axis-aligned box of its four corners meets the view rectangle, both taken half-open as the flat range already is |
| AC-6a | unit | the same grids and views | the same range is computed | it lies within the two-sided bound, so returning the whole map fails; and its column bounds equal the flat range's column bounds exactly |
| AC-7 | integration | every command that prints a summary, over synthetic and representative inputs, for every overlay combination | invoked with `-check` | the summary text is byte-identical to the pre-change output — no summary derives from the viewer's world extent or its mode |
| AC-8 | manual | the 38 shipped maps | opened in the standalone viewer without overlays | each opens without panic; water still animates; on `Kids` and `61` the cells raised in that map's PNG render are raised in the window and the flat cells stay flat, checked against the renders the PNG story left behind; any fidelity limitation is recorded with the evidence |
| AC-9 | manual | the game front-end, which requests no overlay | a map is opened from the picker | the terrain is displaced, panning and zoom reach the whole displaced world, and Esc still unwinds to the picker |

Error cases: mode selection is total — every decoded map yields flat or displaced, and an invalid
grid falls back to flat.

## Derived properties

- **P-1** (invariant) In displaced mode, world width is `Width*32` and world height is
  `MaxV - MinV`. That height numerically equals the flat `Height*32` when every altitude is equal —
  a size identity, not a mode change, since mode never depends on altitude values.
- **P-2** (negative-invariant) When an overlay is requested or the grid is invalid, no displacement
  occurs: world size, overlays and water match the flat viewer exactly, and no tile corner is
  vertically displaced.
- **P-3** (completeness) For any displaced grid, camera and view, every tile whose displaced quad
  intersects the view rectangle is in the culled draw set.
- **P-4** (invariant) For a given tile, the resolved source cell and its water-substituted phase are
  identical between flat and displaced mode; only the destination geometry differs. Both resolve
  without a graphics context — only the upload needs one — so this is checkable headlessly.
- **P-5** (invariant) Displaced geometry is a pure function of the borrowed altitude slice and never
  mutates it; constructing or drawing a displaced viewer leaves the slice unchanged.

## I/O examples

```text
mapview -assets <dir> -map Kids.alm
# window shows Kids with its cliffs and banks displaced; water animates; Esc closes

mapview -assets <dir> -map Kids.alm -objects
# an overlay is enabled, so terrain stays flat and the markers sit on their own cells
```

## Constraints

Interior rasterisation is GPU nearest sampling of each tile's texture across its displaced quad, and
is not claimed pixel-identical to the PNG raster or to the original: the contract shared between the
two paths is the vertex projection, not the raster. Geometry uses the shared projection's integer
arithmetic. Tests use synthetic maps only. No new command-line flag is introduced — displacement is
not opt-in.

## Out of scope

- Terrain shading in the window at all, four-corner light interpolation included.
- Height-projected placements: markers, objects and units follow the terrain in a later story, which
  is also what lifts the overlay-forces-flat rule.
- Pixel-exact original edge stepping, slope occlusion, the dirt composite, hidden-border cropping.
- Draw-call batching or mesh building for performance.
