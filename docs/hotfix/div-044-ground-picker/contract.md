# DIV-044 ground picker contract

## Result

A point on height-displaced terrain resolves to the cell selected by ROM1's
per-column corner-mesh picker. A move, patrol, swarm, empty-ground cast, sack
pick-up, hover query, and ground item drop use that same cell. The existing
mean-of-four-corners placement surface no longer determines ground picks.

The player-visible result is that a click on a strongly sloped cell names that
cell at the point where its corner mesh is drawn. The command target does not
jump to a neighbouring row because the cell's average height differs from the
height at the clicked column.

## Research state

The implementation is based on `TERR-GEOM-036` part (d), High and active at the
pinned research commit `d7ee0c62cfa4a16083d356f24b1870035e1f0209`. The claim
states that `R0377` bounds each cell column with

`y0 + ((y1-y0)*(x&31))/32`

using truncation toward zero. The top and bottom bounds use the corresponding
two projected corner pairs. This picker is intentionally distinct from the
terrain rasterizer's table walk. The claim measures differences of up to three
rows between the two edge models.

`DIV-044` is the existing HOTFIX row for the mean-of-four placement picker. Its
revisit condition is this implementation. No new divergence ID is allocated.
If another mismatch requires a row, implementation stops and asks the seat for
an ID.

## Behaviour

1. The projection exposes one cell column's top and bottom mesh bounds. It uses
   the cell's TL/TR and BL/BR projected corners, `x&31`, integer division, and
   Go's truncation toward zero. No floating-point interpolation replaces this
   rule.
2. The displaced ground picker converts the frame point through the current
   camera, derives the map column and native local column, and checks the rows
   traversed by the terrain pass in ascending draw order. Both edge bounds are
   inclusive. The first matching row owns a shared horizontal seam. A column
   whose top bound is below its bottom bound in screen order can match; an
   inverted span matches no point.
3. Points outside the map viewport, outside the horizontal map extent, outside
   every drawn row, or in an inverted span return no cell. A collapsed span has
   one inclusive coordinate and can match at that coordinate. The no-altitude
   flat fallback retains `Camera.ScreenToCell` because no corner projection
   exists in that mode.
4. Every existing consumer continues to call `Viewer.groundCellAt`. Hover uses
   its cell for fog and sack capability bits. Cell-naming clicks use it for move,
   patrol, swarm, pick-up, and empty-ground cast orders. Inventory releases use
   it for the ground-drop command target. No consumer recomputes the mesh.
5. The picked coordinates cross the existing UI-to-game command seams unchanged.
   This change adds no simulation state, digest field, serialization field, or
   byte-form version.

## Surface enumeration

| Surface | Producer or consumer | Required check |
|---|---|---|
| Altitude mesh | `terrain.Project`, `Projection.WorldCorner` | Four corners, both edge orientations, signed heights, far-edge clamping |
| Column bounds | the new projection helper | Columns 0, 1, 16, and 31; positive and negative fractional quotients; truncation toward zero |
| Coordinate transform | `Camera.ScreenToWorld`, viewport capture, window-to-frame caller boundary | Pan, zoom, fractional camera offsets, screen edges, horizontal map bounds |
| Row population | `Viewer.forEachDrawnTile` / `forEachDisplacedTile` | Steep cells, overlapping rows, inverted spans, top and bottom map rows |
| Hover | `Viewer.hoverMask` | Fog and sack lookup use the mesh-selected cell |
| Click orders | `decide` through `Viewer.command` | Move target is the literal mesh-selected cell; old averaging selects a different row |
| Ground drop | `dropCellAt` | The same picked cell or the existing off-map sentinel |
| Downstream command | App order dispatch and `mapWorld` queue | Target coordinates cross unchanged; pending input does not add canonical state or a digest field |

The committed expected values are literal values worked out from synthetic
corner grids. They do not call the production interpolation helper to obtain an
expectation. A temporary mutation restoring the old mean-of-four rectangle
search must fail the ground-pick tests.

## Domains and aspects

The touched domain is **Client**. The implementation stays within
`pkg/render/terrain` and `pkg/ui`. The existing command seam into `pkg/game` is
verified but not changed.

Applicable closure aspects are data, runtime state, player input, UI/HUD,
shipped content, and interactions with existing mechanics. Simulation, AI,
triggers/scripts, inventory/equipment rules, persistence/save-load, and
campaign/session rules are not changed. Inventory input is applicable only as
an existing consumer of the picked ground cell.

## Exclusions

- The terrain rasterizer keeps its decoded table walk.
- Entity hit rectangles keep the placement surface and entity displacement.
- Minimap cell picking is a separate surface and is unchanged.
- Camera, map format, altitude decoding, movement/pathing, command kinds, and
  simulation hashing are unchanged.
- No new canonical state, hash field, serialization field, or form version is
  introduced.

## Adversarial ceiling and stopping condition

The ordinary ceiling is three fresh-context passes. The chain stops at the
first fresh pass with zero P findings and an empty remaining surface. Only a P
finding returns the implementation. A W finding is recorded in the divergence
ledger, and a D finding is corrected in place. A repeated finding requires an
explicit enumeration of the complete defect class before that class is closed.
