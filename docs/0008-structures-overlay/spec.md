# Spec — placed-objects diagnostic overlay (ROM1)

**Provenance basis.** The object placements this overlay draws come from `pkg/formats/alm` (story 0003),
which decodes the ALM type-4 "placed objects/structures" section from the research submodule
(`research/formats/alm/format.md`, EXP-0019 / ALM-OBJ-019): each object's anchor `(X, Y)` as a `u32`
fixed-point `/256` coordinate (integral cell = `X >> 8`, `Y >> 8`), plus its raw `kind`/`id`/`value` and
optional 8-byte extension. This story adds **no format decoding** — it consumes `alm.Map.Objects` and
renders a diagnostic marker at each object's cell. All marker geometry (the glyph shape, pixel math,
clipping, camera transform) is the project's own diagnostic-render design, not a claim about the original
game's rendering. Greenfield.

## Problem / goal

Story 0003's reader decodes a map's placed objects, but nothing shows them, so object placement is
invisible on the terrain viewers (the PNG compositor of story 0004 and the interactive viewer of story
0005). This story adds an opt-in diagnostic overlay that marks each decoded object's anchor cell over the
terrain — a placement sanity check, **not** final object artwork (object sprites/footprints need the
class registries and static-data formats, which are downstream). It changes nothing unless the overlay is
requested.

## What it consumes (no re-decode)

`alm.Map.Objects` (from 0003). Each `Object` carries `X, Y uint32` (fixed-point `/256`; the integral
anchor cell is `X >> 8`, `Y >> 8`, an integer shift), and raw `kind`/`id`/`value` plus an optional
extension whose meanings are undecoded (0003 R-2) and are **not** interpreted here. This story parses no
ALM bytes and asserts no field meanings.

The 0003 type-4 extension is an 8-byte `[coord][value]` pair of undecoded meaning (0003 R-2) — it is
**not** a width/height, so this overlay draws **no** object footprint; each object is marked by its
anchor cell alone. (Object footprints and art require the registries + static data — out of scope.)

## Behavior definition

For each object, the overlay draws an unlit, opaque marker glyph centered on the object's integral anchor
cell, over the already-drawn terrain, in a fixed color. Off-map anchors are clipped away; the overlay
never changes terrain selection, lighting, or water. The marker geometry is a pure function of the anchor
cell, the map extent and the pixels-per-cell scale — the same three inputs FR-2 states — and is
unit-testable without a window; it operates on plain integer anchor cells (the viewer wires
`alm.Map.Objects` to it), so the geometry code imports no `formats` package and lives in the render tier.

## Functional requirements

- **FR-1** On explicit opt-in, the overlay MUST draw a marker at each object's integral anchor cell
  `(X>>8, Y>>8)` over the terrain, in both the PNG compositor (0004) and the interactive viewer (0005),
  without altering terrain/lighting/water. Off by default; absent the flag the terrain output is
  byte-for-byte the baseline.
- **FR-2** Marker geometry MUST be a pure function of the anchor cell, the map extent and the
  pixels-per-cell scale, producing a small, bounded set of rectangles; the marker **model** MUST
  introduce no floating-point state and no allocation proportional to coordinate magnitude. (Placing a
  marker on screen in the interactive viewer is float arithmetic by construction — FR-6 requires the
  camera transform — but that is placement, not the model.) Invalid scale/bounds MUST yield no geometry,
  never a panic.
- **FR-3** An object whose anchor cell lies outside the map contributes nothing, and this test is applied
  **first**, before any rectangle is built — so an off-map anchor draws nothing even at a scale where one
  of its arms would have reached back onto the map (`cellpx ≤ 2`). For an in-map anchor, every marker
  rectangle MUST then be intersected with the map pixel rectangle and then confined to the output
  rectangle; clipping MUST NOT synthesize a new edge at a clip boundary.
  The second stage is realized differently on the two paths, and both satisfy the same observable
  contract — no marker pixel outside the output rectangle, no new edge at the boundary. In the **PNG**
  path it is an explicit intersection with the image bounds. In the **interactive viewer** it is realized
  by the framebuffer: rectangles lying wholly outside the view are culled, and a rectangle straddling an
  edge is drawn whole and clipped by the framebuffer to its on-screen part. The viewer MUST NOT clip by
  rounding screen coordinates to integer view bounds — that is the independent pixel snapping FR-6
  forbids, so the framebuffer clip is the only realization compatible with FR-6.
- **FR-4** The overlay MUST be independently opt-in and composable with other overlays (draw order:
  terrain → objects → [units, story 0009]).
- **FR-5** In headless/check output, enabling the overlay MUST report the decoded object count
  (`len(alm.Map.Objects)`); disabled output MUST keep its existing text shape.
- **FR-6 (pixel geometry — the project's own)** Markers are opaque yellow `#FFD000`, drawn after
  terrain. In native world pixels, for anchor cell `(ax,ay)`, center `(cx,cy) = (32·ax+16, 32·ay+16)`;
  the cross is the union of the half-open rectangles `[cx-6,cx+7)×[cy-1,cy+2)` and
  `[cx-1,cx+2)×[cy-6,cy+7)`. In the PNG at `cellpx` pixels/cell the center is
  `(ax·cellpx + floor(cellpx/2), ay·cellpx + floor(cellpx/2))`, arm radius `r = max(1, round(6·cellpx/32))`,
  thickness `t = max(1, round(3·cellpx/32))`, with non-negative `round(n/32) = floor((n+16)/32)`. A centered
  strip of thickness `t` occupies `[center-floor(t/2), center-floor(t/2)+t)`, and an arm of radius `r`
  centered at coordinate `c` occupies the half-open span `[c-r, c+r+1)` (length `2r+1`) along its long
  axis; so the horizontal arm is `[cx-r,cx+r+1)×[cy-floor(t/2),cy-floor(t/2)+t)` and the vertical arm is
  `[cx-floor(t/2),cx-floor(t/2)+t)×[cy-r,cy+r+1)`. At `cellpx = 32` (`r=6, t=3`) this reduces exactly to the
  two native rectangles above. `cellpx < 1` is invalid and yields no geometry (FR-2). The interactive viewer
  transforms this native geometry through the **same** camera transform as terrain (nearest filtering,
  per-rectangle culling); no independent pixel snapping is applied.

## Acceptance criteria (synthetic `Objects`, no game assets)

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic `[]alm.Object`-equivalent cell list with fractional `/256` coords and some off-map anchors | converted to overlay geometry | each in-map object yields a cross at its integral anchor cell `(X>>8,Y>>8)`; off-map anchors yield nothing; no footprint is drawn |
| AC-2 | unit | native and downscaled geometry at representative odd/even `cellpx` | rasterized | occupied half-open rectangles match FR-6 exactly, including original-edge clipping without synthetic borders |
| AC-3 | unit | extreme coordinate magnitudes and a tiny output rectangle | converted/drawn | geometry is arithmetically clipped to map/view bounds; no panic; no magnitude-proportional allocation |
| AC-4 | unit | the overlay disabled for either viewer | rendered | output/draw inputs remain the terrain-only baseline |
| AC-5 | unit | a headless check with the overlay enabled | queried | reports `len(Objects)` |
| AC-6 | manual | a real GOG map with placed objects, loaded by both viewers with the overlay on | rendered to PNG + live window | markers align with terrain cells through pan/zoom; the reported count equals the map's object count; recorded in verification.md — no game bytes committed |

## Derived properties

- **P-1** (negative-invariant) Marker conversion never indexes outside map/output bounds; ≤ 2 rectangles
  per object.
- **P-2** (negative-invariant) For any coordinate/scale, geometry generation does not allocate
  proportionally to coordinate magnitude and does not panic.
- **P-3** (invariant) With the overlay disabled, terrain output is byte-identical to the baseline.

## Constraints

- Consumes `alm.Map.Objects` only; no archive access, registry lookup, or object-art rendering; no
  floating-point in the marker model; game assets are never test fixtures (synthetic objects only).
- The marker geometry lives in the render tier and takes plain integer anchor cells — it imports no
  `formats` package; the viewer performs the `alm.Map.Objects → cells` wiring.
- Draw order and the marker glyph are the project's own diagnostic design, not a decoded game fact. The
  specific color and size *values* are a project choice — they carry no game-fidelity claim and could have
  been chosen otherwise — but once chosen, the FR-6 geometry (color `#FFD000`, radius/thickness, the exact
  half-open rectangles) is the frozen contract the acceptance tests pin (AC-2). "Not normative" means the
  values assert nothing about the original engine, not that an implementation may drift from FR-6.

## Out of scope

- Object identity, class/template, real object sprites/art, footprints, animation, shadows, selection,
  interaction — these need the class registries + static-data formats, downstream.
- Interpreting the object `kind`/`id`/`value`/extension fields — 0003 R-2, not decoded here.
- Units, groups, triggers, terrain lighting/water — other stories.
- Decoding or writing ALM bytes — 0003 owns the container; this story never parses a map.

## Gate check

FR-1 → AC-1, AC-4, AC-6, P-3 · FR-2 → AC-3, P-2 · FR-3 → AC-2, AC-3, AC-6, P-1 · FR-4 → AC-4 ·
FR-5 → AC-5 · FR-6 → AC-1, AC-2, AC-6.

FR-1's and FR-3's *enabled, both-viewers* half is carried by **AC-6**, the manual criterion — the unit
ACs cover the geometry, the disabled baseline and the count, not the on-screen result. AC-6 is therefore
load-bearing, not a nicety: until it is run, those halves rest on the tier-level evidence alone.

## Verification mapping

AC-1…AC-5 + P-1…P-3: unit tests on the pure marker geometry and the overlay toggle. AC-6: a developer-run
render of a real map (PNG + live window), evidence recorded in verification.md.
