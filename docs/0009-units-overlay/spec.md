# Spec — placed-units diagnostic overlay (ROM1)

**Provenance basis.** The unit placements this overlay draws come from `pkg/formats/alm` (story 0003),
which decodes the ALM type-6 "placed units" section from the research submodule
(`research/formats/alm/format.md`, EXP-0019, amended by EXP-0030 / `ALM-UNIT-018`): 70-byte fixed
records, count `= meta+0x24`, each unit's anchor `X @ rec+0x00` / `Y @ rec+0x04` (`u32` fixed-point `/256`,
low byte usually `0x80` = cell centre; integral cell = `X >> 8`, `Y >> 8`). *(All `.alm` offsets here are
quoted at the EXP-0030 corrected container framing; the pre-EXP-0030 labels for the same file bytes were
`meta+0x2c` / `rec+8` / `rec+12`.)* That claim is **High**
confidence for exactly those three things — the stride, the count word, and the coordinate pair
(`70·count == payloadSize` on 38/38 corpus maps, all 8094/8094 anchors landing inside `[0,W)×[0,H)`) —
and **Medium** for the "unit" label itself and for every non-coordinate field: the rest of the record
(id / stats / inventory) is located only as runs of `0xFF` sentinel words and is **not individually
pinned** (0003 R-2). This story adds **no format decoding** — it consumes `alm.Map.Units` and renders a
diagnostic marker at each unit's cell. All marker geometry is the project's own diagnostic-render
design, not a claim about the original game's rendering. Greenfield.

## Problem / goal

Story 0003 decodes a map's placed units, but nothing shows them, so unit placement is invisible on the
terrain viewers (the PNG compositor of story 0004 and the interactive viewer of story 0005). This story
adds an opt-in diagnostic overlay that marks each decoded unit's anchor cell over the terrain — a
placement sanity check, **not** unit artwork (unit sprites, player colors, and identity need the class
registries + static data, which are downstream). It changes nothing unless the overlay is requested, and
composes with the objects overlay (0008).

## What it consumes (no re-decode)

`alm.Map.Units` (from 0003). Each `Unit` carries `X, Y uint32` (fixed-point `/256`; the integral anchor
cell is `X >> 8`, `Y >> 8`, an integer shift) **and nothing else** — the type-6 record's remaining bytes
are undecoded (0003 R-2) and are not surfaced by the reader. No unit type, owner, stat, or inventory
value is therefore available to this story, let alone read or asserted. In particular this story claims
**no** unit-type id: the research pins the 70-byte stride and the `X@+0x00` / `Y@+0x04` coordinates, and pins
no type-id offset. This story parses no ALM bytes and asserts no field meanings.

## Behavior definition

For each unit, the overlay draws an unlit, opaque marker glyph centered on the unit's integral anchor
cell, over the already-drawn terrain, in a fixed color, composed after the objects overlay. A unit whose
anchor cell lies off the map is dropped; the overlay never changes terrain, lighting, water, or objects.
The marker geometry is a pure function of `(anchorCell, cellPixels)` and is unit-testable without a
window; it operates on plain integer anchor cells (the viewer wires `alm.Map.Units` to it), so the
geometry code imports no `formats` package and lives in the render tier.

## Functional requirements

- **FR-1** On explicit opt-in, the overlay MUST draw a marker at each unit's integral anchor cell
  `(X>>8, Y>>8)` over the terrain, in both the PNG compositor (0004) and the interactive viewer (0005),
  without altering terrain, lighting, water, or objects. It is off by default; absent the opt-in the
  output is byte-for-byte what the same invocation produces without this story.
- **FR-2** Marker geometry MUST be a pure function of `(anchorCell, cellPixels)` producing a small,
  bounded set of rectangles; it MUST introduce no floating-point state and no allocation proportional to
  coordinate magnitude. An invalid scale or an empty target rectangle MUST yield no geometry, never a
  panic.
- **FR-3** On a `W×H`-cell map, a unit whose integral anchor cell falls outside `[0,W)×[0,H)` MUST
  contribute nothing — including at extreme coordinate magnitudes, where it MUST NOT wrap into a visible
  marker. Every remaining marker rectangle MUST be intersected with the map pixel rectangle
  `[0,W·cellpx)×[0,H·cellpx)` and then with the output/view rectangle (the PNG image bounds, or the
  viewer's visible region); clipping MUST NOT synthesize a new edge at a clip boundary. Two units
  resolving to the same anchor cell each contribute a marker; the markers are identical and opaque, so
  the visible result is one marker.
- **FR-4** The overlay MUST be independently opt-in and composable with the objects overlay (0008):
  either overlay can be enabled without the other, and when both are enabled the draw order is
  terrain → objects → units, so a unit marker is drawn over a coincident object marker.
- **FR-5** In headless/check output, enabling the overlay MUST report the decoded unit count
  (`len(alm.Map.Units)`); when the objects overlay is enabled alongside it, the composed summary MUST
  order the object count before the unit count. With this overlay disabled, the output MUST keep the
  shape it has without this story.
- **FR-6 (pixel geometry — the project's own)** Markers are opaque cyan `#00E5FF`, drawn after terrain
  and after the objects overlay, at most 2 rectangles per unit. In the PNG at `cellpx` pixels per cell,
  for anchor cell `(ax,ay)` the center is `(cx,cy) = (ax·cellpx + floor(cellpx/2), ay·cellpx +
  floor(cellpx/2))`, the cross arm radius is `r = max(1, round(4·cellpx/32))` and its thickness is
  `t = max(1, round(cellpx/32))`, with non-negative `round(n/32) = floor((n+16)/32)`. A centered strip of
  thickness `t` occupies `[center-floor(t/2), center-floor(t/2)+t)`, and an arm of radius `r` centered at
  coordinate `c` occupies the half-open span `[c-r, c+r+1)` (length `2r+1`) along its long axis; so the
  horizontal arm is `[cx-r,cx+r+1)×[cy-floor(t/2),cy-floor(t/2)+t)` and the vertical arm is
  `[cx-floor(t/2),cx-floor(t/2)+t)×[cy-r,cy+r+1)`. At the native `cellpx = 32` this gives `r=4, t=1`,
  i.e. exactly the two rectangles `[cx-4,cx+5)×[cy,cy+1)` and `[cx,cx+1)×[cy-4,cy+5)` around the native
  center `(32·ax+16, 32·ay+16)`. `cellpx < 1` is invalid and yields no geometry (FR-2). The interactive
  viewer transforms this native geometry through the **same** camera transform as terrain (nearest
  filtering, per-rectangle culling); no independent pixel snapping is applied.

## Acceptance criteria (synthetic `Units`, no game assets)

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a synthetic `[]alm.Unit`-equivalent cell list with fractional `/256` coords, two units on one cell, and some off-map anchors | converted to overlay geometry | each in-map unit yields a cross at its integral anchor cell `(X>>8,Y>>8)`; coincident units yield identical geometry; off-map anchors yield nothing; the fractional low byte changes no output |
| AC-2 | unit | native and downscaled geometry at representative odd/even `cellpx`, including markers straddling the map edge | rasterized | occupied half-open rectangles match FR-6 exactly, including original-edge clipping without synthetic borders |
| AC-3 | unit (error case) | extreme coordinate magnitudes (up to the `uint32` maximum), an invalid scale (`cellpx < 1`), and an empty target rectangle | converted/drawn | an extreme anchor contributes nothing and never wraps into a visible marker; other geometry is arithmetically clipped to map/view bounds or empty; no panic; no magnitude-proportional allocation |
| AC-4 | unit | the unit overlay disabled for either viewer, with the objects overlay both off and on | rendered | output/draw inputs are byte-identical to the same invocation without this story |
| AC-5 | unit | a headless check with both overlays enabled | queried | the summary reports the object count and then the unit count (`len(Units)`) |
| AC-6 | manual | a real GOG map with placed units, loaded by both viewers with both overlays on | rendered to PNG + live window | unit markers align with terrain cells through pan/zoom and sit above coincident object markers; the reported unit count matches the count the map's own type-6 metadata declares; recorded in verification.md — no game bytes committed |

## Derived properties

- **P-1** (negative-invariant) Marker conversion never indexes outside map/output bounds; at most 2
  rectangles per unit.
- **P-2** (negative-invariant) For any coordinate, scale, or target rectangle — including invalid ones —
  geometry generation does not allocate proportionally to coordinate magnitude and does not panic.
- **P-3** (invariant) With the unit overlay disabled, output is byte-identical to what the same
  invocation produces without this story: terrain alone, or terrain + objects when that overlay is on.

## Constraints

- Consumes `alm.Map.Units` only; no archive access, registry lookup, or unit-art rendering; no
  floating-point in the marker model; game assets are never test fixtures (synthetic units only).
- The marker geometry lives in the render tier and takes plain integer anchor cells — it imports no
  `formats` package; the viewer performs the `alm.Map.Units → cells` wiring.
- Draw order and the marker glyph are the project's own diagnostic design, not a decoded game fact. The
  specific color and size *values* are a project choice — they carry no game-fidelity claim and could
  have been chosen otherwise — but once chosen, the FR-6 geometry (color `#00E5FF`, radius/thickness, the
  exact half-open rectangles) is the frozen contract the acceptance tests pin (AC-2). "Not normative"
  means the values assert nothing about the original engine, not that an implementation may drift from
  FR-6.
- The composition statements in FR-4/FR-5 are requirements this story places on the composed viewers —
  its own opt-in must be independent and its own count must follow the object count — not a restatement
  of the objects overlay's contract.

## Out of scope

- Unit sprites/palettes/animation/selection/HP/player colors; unit identity, type, owner, or faction —
  these need the class registries + static-data formats, downstream.
- Interpreting any non-coordinate field of the unit record, including a unit-type id — 0003 R-2, not
  decoded here and not exposed by the reader.
- Objects (0008), groups/triggers/effects, collision/movement/combat/sim.
- Decoding or writing ALM bytes — 0003 owns the container; this story never parses a map.

## Gate check

FR-1 → AC-1, AC-4, AC-6, P-3 · FR-2 → AC-3, P-1, P-2 · FR-3 → AC-1, AC-2, AC-3, P-1 ·
FR-4 → AC-4, AC-5, AC-6 · FR-5 → AC-4, AC-5 · FR-6 → AC-1, AC-2, AC-6.

## Verification mapping

AC-1…AC-5 + P-1…P-3: unit tests on the pure marker geometry and the overlay composition/toggle — all
CI-automatable with synthetic units. AC-6: a developer-run render of a real map (PNG + live window),
evidence recorded in verification.md — no game bytes committed.
