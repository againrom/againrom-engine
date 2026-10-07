# Spec — corner-interpolated terrain lighting in the window

## Problem and current behaviour

The windowed viewer draws every terrain pixel at full palette brightness: it applies no relief
lighting in either of its two modes. The PNG compositor has shaded since 0007, the displaced raster
too, so the same map arrives lit in a file and unlit on screen. This story lights the window.

Everything the viewer does that is not a pixel's brightness is frozen; FR-1 names the set. The story
changes neither map decoding, nor the shared projection, nor the level grid and its light parameters,
nor any PNG path.

## The lighting contract

### The level grid

For a `Width x Height` map the shared renderer computes a per-vertex attenuation **level** grid from
the signed height field and a light: `levels[r*Width + c]`, a byte in `[0,95]`, is the level at grid
vertex `(c,r)`. **Higher level is darker.** The window uses the PNG path's own default light — sun
angle `0.78539815`, ambient `0x0e`, range `0x20`, sky tint `(0,0,0)` — under which flat ground levels
to 46.

### Corner levels

Tile `(c,r)`'s four corner levels are the grid values at vertices `(c,r)`, `(c+1,r)`, `(c,r+1)` and
`(c+1,r+1)`, each index **clamped** into `[0,Width-1] x [0,Height-1]`. A `Width*Height` vertex grid
does not carry the far-edge `+1` vertex, and this clamp is the same read the PNG compositors make —
not a second convention.

### The shade

The shading transform is per channel, integer, and truncating, and with the daytime tint `(0,0,0)` the
additive term vanishes and it is exactly a multiply:

```text
out = clamp( (channel + tint) * (LevelCount - level) / 32, 0, 255 ),  LevelCount = 96
m(level) = (LevelCount - level) / 32
```

Level 64 is x1.0, level 46 is x1.5625, level 0 is x3.0 — the multiplier is **above 1 over most of the
range**. The multiply is what the window applies: **a tile is drawn with `m` of each of its four corner levels at
that corner, and the interior interpolated between the four.** All three colour channels take the
same multiplier; alpha is untouched, and a tile stays fully opaque. The interpolation runs over the
two triangles the quad is split into along its **top-left to bottom-right** diagonal, in **both**
modes: the split changes a non-uniform tile's interior, so it is fixed here rather than left to the
drawing call.

This is shading, not per-facet fill: a tile whose four corner levels differ MUST vary in brightness
across its own interior, and a single value filled over the whole tile is non-conformant even where
that value is correct at the centre.

### When the window is lit

Lighting is selected independently of the render mode, and by a different predicate:

```text
lit        when  the altitude grid is valid  AND  the unshaded diagnostic is off
displaced  when  the altitude grid is valid  AND  no diagnostic overlay is requested
```

A **valid** altitude grid is one with exactly `Width*Height` entries over positive dimensions — the
same predicate mode selection already uses. Lighting is derived from that grid, so an invalid one
leaves the window unlit exactly as it leaves it flat. Lighting does **not** read the overlay flags:
an overlay forces flat mode, and flat mode is then lit on the same terms displaced mode is — by the
predicate above and nothing else. All four combinations are reachable and each is specified: flat
unlit, flat lit, displaced lit, displaced unlit.

The unshaded diagnostic is a flag on the standalone developer viewer only. The game front-end
requests neither an overlay nor the diagnostic, so the game is displaced and lit.

## Functional requirements

- **FR-1** For every input, the viewer MUST retain its current world size, per-mode visible-tile
  range, mode selection, tile and water resolution, displaced mesh and painter order, overlay
  geometry and draw order, and every command's summary text, with no drift. Only a drawn pixel's
  brightness may change, with two exceptions, both named here. **The camera's input surface.** Every
  camera invariant — the scroll clamp, centring on a small axis, zoom about the cursor, world extent,
  visible range — MUST NOT change; only the set of inputs that can pan gains the member FR-9 adds.
  **The drawing call.** Flat mode's tiles MAY be submitted through a different drawing call, whose
  pixel-centre tie-break differs. Its differences MUST be confined to a tile boundary that falls
  **exactly** on a pixel centre: there a boundary may move by one pixel, a tile's pixel width may
  differ by one, and that one column may go unpainted. No tile may be distorted, and the set of tiles
  drawn MUST NOT change.
- **FR-2** With lighting on, each drawn tile MUST be shaded by `m` of its own four corner levels,
  one multiplier per corner, with the interior interpolated between them — in **both** render modes.
- **FR-3** Corner levels MUST be read as the four grid vertices above, with far-edge indices clamped
  into the grid, from a level grid computed from the map's altitudes and the fixed daytime light.
- **FR-4** Lighting MUST be on exactly when the altitude grid is valid and the unshaded diagnostic
  is off, MUST NOT read either overlay flag, and MUST expose whether it is on for inspection.
- **FR-5** The standalone viewer MUST gain exactly one new flag, `-unshaded`, which renders at full
  palette brightness; no flag may move the light itself. The game front-end MUST gain no flag.
- **FR-6** A cell drawn as the placeholder fill (its tile slot absent or short) MUST stay unlit: it
  is a diagnostic, not terrain.
- **FR-7** The corner levels, the multiplier and each tile's four corner colours MUST be computed by
  pure functions that run without a window; only the final draw call may require a graphics context.
- **FR-8** Lighting MUST obtain the level grid from the shared renderer, MUST NOT introduce a second
  light model or level convention, and MUST NOT mutate the altitude slice it borrows.
- **FR-9** Holding the primary mouse button MUST pan: the world point under the cursor stays under
  the cursor, so a screen delta of `d` moves the world by `d/zoom`. The pan MUST come from the
  difference between consecutive ticks while held, never from an absolute cursor position — the first
  tick of a drag anchors it and pans zero, so a button already held when the map opens, or a cursor
  that jumps, cannot jump the view. A tick that pans by drag MUST NOT also edge-scroll; the keyboard
  keeps working. Releasing the button ends the drag wherever the cursor is, and no state survives it.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | flat, single-axis-slope, asymmetric and negative-altitude synthetic grids | a tile's four corner levels are read for interior, right-edge, bottom-edge and corner tiles | each equals the level grid at that vertex with far-edge indices clamped, and the values agree with the PNG compositor's own read for the same tile |
| AC-2 | unit | levels 0, 46, 64, 95 and the clamp boundaries | the multiplier is computed | it is `(96 - level)/32` — 3.0, 1.5625, 1.0, 1/32 — with an out-of-range level clamped into `[0,95]` first |
| AC-3 | unit | a displaced viewer over a sloped synthetic map | a tile's four drawn corners are computed | each corner carries the multiplier of its own corner level, all three channels equal, alpha opaque, and the four geometry fields are byte-identical to the pre-change values |
| AC-4 | unit | a flat-mode viewer over the same map | the same is computed for flat mode | the four corners carry the same four multipliers, and the four destination positions are the un-displaced cell lattice through the camera, unchanged |
| AC-5 | unit | a map with no altitude grid, one with an invalid-length grid, one valid, and each with the unshaded diagnostic on and off | lighting selection is queried | lit exactly when the grid is valid and the diagnostic is off; enabling either overlay changes the mode and NOT the lighting; with lighting off every drawn corner multiplier is exactly 1 |
| AC-6 | unit | a valid map, unshaded | the tiles' corner colours are computed | they are byte-identical to the pre-change viewer's |
| AC-7 | unit | a cell whose tile slot is absent, and a water cell mid-cycle | both are drawn lit | the placeholder is unlit; the water cell is lit by its cell's own corner levels, identically at every animation phase — the level is a property of position, not of phase |
| AC-8 | unit | a lit viewer built and drawn over a borrowed altitude slice | the slice is compared before and after | it is unchanged, and no level or multiplier is stored on the grid |
| AC-9 | integration | every command that prints a summary, over synthetic and representative inputs, in every combination of the two overlay flags and the unshaded diagnostic | invoked with `-check` | the summary text is byte-identical to the pre-change output — no summary derives from lighting |
| AC-10 | manual | the 38 shipped maps | opened in the standalone viewer | each opens without panic; slopes and banks read as shaded relief, and the same map under `-unshaded` is the flat-bright image that shipped before; on at least two maps the window is compared against that map's PNG render: wherever the PNG renders one cell darker than a neighbour the window MUST agree on which is darker, and every difference in magnitude is recorded |
| AC-11 | manual | the game front-end, which requests neither an overlay nor the diagnostic | a map is opened from the picker | the terrain is displaced and lit, panning and zoom keep it lit, and Esc still unwinds to the picker |
| AC-12 | unit | a lit viewer over a synthetic map, at zoom 1 and at a zoom != 1 | synthetic input drives press, move, move, release — including a zero-move drag, a first tick with the button already down, and a drag inside the edge margin | the camera moves by the tick's screen delta divided by the zoom, opposite to the cursor; the anchoring tick and the zero-move drag move it not at all; the edge-margin drag pans once, not twice; the clamp holds at every step |
| AC-13 | manual | the standalone viewer and the game front-end | the primary button is held and the mouse moved, then released | the terrain follows the cursor under it; releasing stops it, including a release outside the window |

Error cases: lighting selection is total — every decoded map is lit or unlit, and an invalid altitude
grid falls back to unlit, as it already falls back to flat.

## Derived properties

- **P-1** (invariant) Where a tile's four corner levels are equal, its four corner multipliers are
  equal and its interior is a single brightness — the flat-shaded result, reached as the degenerate
  case of interpolation rather than by a separate path.
- **P-2** (bound) Every drawn multiplier lies in `[1/32, 3]`, and every interpolated interior value
  lies between the tile's least and greatest corner multiplier.
- **P-3** (negative-invariant) With lighting off, every corner multiplier is exactly 1 and no
  submitted geometry, source cell, overlay or summary differs from the pre-change viewer. Rasterised
  pixels may differ only by FR-1's named tie.
- **P-4** (invariant) A tile's four corner multipliers depend only on its position and the map's
  altitudes — never on the render mode, the zoom, the camera position, the animation phase or the
  overlay flags. A cell drawn in both modes at any zoom carries the same four multipliers.
- **P-5** (invariant) Lighting is a pure function of the borrowed altitude slice and never mutates
  it.
- **P-6** (invariant) A drag's pan is a pure function of two consecutive cursor positions and the
  zoom; a drag that does not move the cursor moves the camera zero, at every zoom.

## I/O examples

```text
mapview -assets <dir> -map Kids.alm
# window shows Kids displaced and relief-shaded; water animates; Esc closes

mapview -assets <dir> -map Kids.alm -unshaded
# the same displaced geometry at full palette brightness -- the pre-change image

mapview -assets <dir> -map Kids.alm -objects
# an overlay forces flat, and the flat terrain is still lit
```

## Constraints

The window's interior brightness is **not** claimed pixel-identical to the PNG raster or to the
original. Three differences are known and disclosed: the interior is interpolated by the drawing
call across the tile's two triangles rather than bilinearly over the quad, so a non-parallelogram
tile's interior depends on the split; the level is not truncated to an integer attenuation row, so
the ramp is continuous where the original indexes one of 96 discrete rows; and the final per-channel
rounding is the drawing call's, not our integer division's. What the two paths share is the corner
levels and the multiplier at each corner, not the raster between them.

FR-1's tie is **bounded, not predicted**: at a boundary landing exactly on a pixel centre a synthetic
probe leaves that one-pixel column unpainted, while the same tie through the production vertex path —
bit-identical coordinates — shows none, and the gap proved sensitive to draw order and canvas state
rather than to the coordinate. Whether either mode reaches it is not established; FR-1 permits it
without asserting it (verification.md). The overlay rects, drawn by another call, do not share the
tie. A non-zero sky tint is not expressible as a multiplier and is out of scope. Tests use synthetic maps only, and no test opens a graphics context.

## Out of scope

- The light-value formula, its constants, range, and the far-edge ring's undefined value.
- Day/night cycling, dynamic lights, sprite shadows and fog.
- Lighting anything but terrain: markers, objects and units stay unlit, as does the placeholder fill.
- The dirt composite in the window, which it has never drawn.
- Truncating the interior level to an attenuation row (what a fragment shader would be for; none is
  added), and any RGB565 packing of the result.
- Draw-call batching or mesh building for performance.
- Drag on any button but the primary, pan momentum, and a cursor change while dragging.
