# Spec — interactive terrain map viewer

**Provenance basis.** This story is camera/UI engineering — pan, zoom, edge-scroll, clamping — which is
the project's own design, not a claim about the original engine's camera or UX. The only game-format tie
is the tile-word bit layout it reads through `pkg/formats/alm` (0003): `index = cell & 0x3ff`,
`impassable = cell & 0x2000` (EXP-0020 / ALM-GRID-012). The actual terrain **graphics** the viewer draws
come from story 0004, which is implemented: the `terrain.3d` tileset and the research-decoded tile-word →
graphic mapping (`TERR-LOC-001`, `TERR-IDX-003`, EXP-0021) are available as `pkg/render/terrain`. This
story adds **no format decoding** — it consumes that pipeline and puts a camera in front of it.
Greenfield.

## Problem / goal

A whole-map composite (0004) is fine for verification, but you cannot explore a map: a 256×256 map is
8192×8192 px. This story adds a windowed, real-time viewer built on Ebitengine that draws terrain at
native tile size and lets you scroll and zoom, through a camera. The camera/UI is the project's own
design; the terrain content it renders is whatever 0004 provides (currently gated on 0004 R-1/R-2).

## Behavior definition

The viewer opens a resizable window and draws the map's terrain through a camera:

- **Native zoom** — at zoom 1.0 a tile is 32×32 screen px, so only part of a large map is visible.
- **Edge-scroll pan** — when the cursor is within an edge margin, the view scrolls in that direction at a
  constant world-space speed independent of zoom. Arrow keys (and WASD) pan as well.
- **Wheel zoom** — the mouse wheel zooms about the cursor position, clamped to a sensible range; the world
  point under the cursor stays under the cursor.
- **Clamping** — the camera never scrolls past the map edges; an axis smaller than the window is centered.
- **Quit** — Esc closes the window.

All of the above (pan model, zoom limits, edge margin, cadence) are the project's own UX design — not a
reproduction of any documented original-engine camera behavior, and the spec makes no such claim. Water
tiles render with their base cell (no animation — 0006); terrain is drawn flat (no height displacement or
lighting — 0007). Those are later stories.

## Functional requirements

- **FR-1** The viewer MUST load a map (loose `.alm` or archive entry, via `pkg/formats/alm`) and the
  terrain tileset (`pkg/render/terrain`, 0004) from `graphics.res`, and open a window titled with the map
  name. The asset root comes from `-assets`/`AGAINROM_ASSETS`, never hardcoded.
- **FR-2** Each frame the viewer MUST draw only the tiles intersecting the current camera view, resolving
  each cell's terrain graphic via the 0004 pipeline (tile index from `alm.TileIndex`), scaled by the zoom.
- **FR-3** The camera MUST support pan (edge-scroll + keys) and wheel-zoom-about-cursor, and MUST clamp so
  the view never leaves the map bounds (centering a map axis smaller than the window). *(Camera model:
  ours.)*
- **FR-4** A cell whose terrain graphic is unavailable MUST render as a solid fill, never a crash.
- **FR-5** Asset loading and camera math MUST be separable from the Ebitengine run loop so they are
  unit-testable without opening a window; malformed input MUST surface as an error before the window opens.

## Acceptance criteria

| ID | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| AC-1 | unit | a camera over a 100×100-tile world in an 800×600 window | pan past an edge | position clamps to `[0, worldPx-viewPx]` on each axis |
| AC-2 | unit | a world smaller than the window on an axis | clamped | that axis is centered |
| AC-3 | unit | a camera at zoom 1 | zoom about a cursor point | scale changes within limits and the world point under the cursor is preserved (within rounding) |
| AC-4 | unit | a camera view | queried for visible tile range | the range covers exactly the tiles intersecting the view, clipped to the map |
| AC-5 | unit | a headless `-check` run over a synthetic archive + map | invoked | loads map + tileset, prints a one-line summary, exits 0 without opening a window |
| AC-6 | manual | a real GOG map + `graphics.res` | launched | window shows terrain; edge-scroll, keys, wheel zoom and Esc all work; scrolling stops at map edges; recorded in verification.md |

Error cases handled before the window opens: missing archive/map, unparseable map.

## Derived properties

- **P-1** (invariant) For any pan/zoom sequence the camera offset stays within the clamp bounds.
- **P-2** (negative-invariant) Rendering never indexes a tile or graphic out of range (missing graphics
  filled, indices bounded).

## Constraints

- Ebitengine (`github.com/hajimehoshi/ebiten/v2`) is the windowing/rendering layer — the project's chosen
  2D engine. Its DAG placement (a `pkg/ui`/`pkg/game`/viewer-cmd tier) and the viewer's executable name
  are settled with the viewer stories; the camera math itself imports no `formats` package.
- Reuses `pkg/formats/alm` (0003) and the 0004 terrain pipeline; no new format work here.

## Out of scope

- Water animation (0006), height-displaced terrain / lighting (0007), land/water edge blending.
- Objects/units/markers (0008/0009), minimap, tile picking, screenshots (the 0004 compositor covers
  static output).
- The terrain-graphic decode/mapping itself — that is 0004.

## Verification mapping

AC-1…AC-5 + P-1/P-2: unit tests on the camera, the visible-tile-range logic, and the headless `-check`
path (no window). AC-6: a manual run against a GOG install, recorded in verification.md.
