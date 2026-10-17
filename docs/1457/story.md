# 1457 — One cell projection for the editor and the game

## Result

A map cell's corners, centre, quad and screen vertices come from one family,
`cellSpace` in `pkg/ui/cellprojection.go`, in flat and in displaced mode. The
map editor and the game call it; no other site in `pkg/ui` computes a cell
corner or centre by hand, and `internal/archtest` fails a new one. On sloped
ground the editor's unit and structure cross, the cross on a linked footprint
and the centre cross of a square trigger area now stand on the point where the
game draws that cell's path line. Game frames, simulation, SAV and ALM bytes
are unchanged.

## Intent

Owner direction: architecture audit `8dff7769`, rank 27 (cell-to-screen
projection), taken as a small editor story; owner rule "one builder per kind".
The audit found cell corner and centre to world computed in seven places. On a
sloped cell the editor took the centre height from the midpoint of the top-left
and bottom-right corners, and the game's path line from the four-corner anchor
height.

## Authority

- Displaced centre height: `TERR-SPR-039`. The original lifts a sprite by the
  mean of its cell's four corner altitudes, read signed, summed and divided by
  4 truncated toward zero (High). `terrain.Projection.AnchorHeight` implements
  this mean, and the game's path line already used it. `TERR-STRUCT-106` reads
  a drawable's lift as a signed interpolation at its footprint centre; its
  structure consequence is partially retracted by `TERR-221`. No claim names
  an original cell-centre marker on displaced terrain; the original editor is
  not evidence here.
- Mesh corner: `TERR-GEOM-031` (`r*32 - h`, signed), through
  `terrain.Projection.WorldCorner`.
- The editor is Againrom's own (owner); its markers have no original
  counterpart. The editor adopts the game's rule, so no divergence row is
  added.

## As built

`cellSpace{proj}` is the family; `proj == nil` is flat mode.

| member | answer |
|---|---|
| `corner(c, r)` | flat `(c*32, r*32)`; displaced `Projection.WorldCorner(c, r)` |
| `lift(cell)` | flat 0; displaced `-AnchorHeight(cell) - MinV` |
| `centre(cell)` | `(c*32+16, r*32+16 + lift(cell))` |
| `quad(col, row)` | four `corner`s in outline order TL, TR, BR, BL |
| `vertices(cam, tx, ty)` | four `corner`s through the camera, TL, TR, BL, BR, as ebiten vertices |

Viewer entry points, all in the same file: `cellSpace()` (by `Mode()`),
`cellWorldCentre`, `cellScreenCentre`, `cellScreenCorner`, `cellLift`, and the
wrappers `tileVertices` and `flatTileVertices`.

Callers:

| site | before | now |
|---|---|---|
| `mapeditor.go` `cellCenter` (selected and issue crosses, report and drop crosses, footprint cross, jump, `CellPoint`) | literal 32; displaced midpoint of corners (c,r), (c+1,r+1) | clamp, then `cellWorldCentre` |
| `mapeditor.go` selected footprint, `mapeditorbrush.go` brush cells, `shroud.go` | flat or displaced vertex pair by mode | `cellSpace().vertices` |
| `mapeditorareas.go` `corner` (area outline, area label) | literal 32 or `WorldCorner` | `cellScreenCorner` |
| `mapeditorreferences.go` `referenceWorldCorners` | literal 32 or `WorldCorner` | `cellSpace().corner` |
| `overlay.go` `gridWorldQuad` | its own flat and displaced quads | `cellSpace().quad` |
| `overlay.go` `cellLift` | its own lift | moved into the family |
| `path.go` `cellWorldCentre`, `pathCellCentre` | its own centre | moved into the family; `cellScreenCentre` |
| `viewer.go` `tileVertices`, `flatTileVertices`, `flatTileScreen` | own corner arithmetic | `cellSpace` |
| `effectmark.go` `entityMarkRects` | own centre, lift and origin | `cellWorldCentre` plus the entity shift |
| `minimap.go` visible rows | four `WorldCorner` calls | `cellSpace{proj}.quad` |
| `headlesspointer.go` `HeadlessCellPoint` | flat centre in every mode, then verified | `cellScreenCentre`, then verified |

Every game-frame site computes the same integers as before: the game already
used the anchor height for path legs, glyphs and effect marks, and the flat and
displaced corners are the same expressions. The editor's centre is the one
changed answer. `HeadlessCellPoint` now aims at the displaced centre in
displaced mode; its cell verification is unchanged.

## Ratchet

`internal/archtest/cellprojection.go` scans the production files of `pkg/ui`
and `pkg/render`, outside `pkg/ui/cellprojection.go` and
`pkg/render/terrain/project.go`, for two shapes: a multiplication by
`CellSize` or the integer literal 32, and a call of `WorldCorner` or
`AnchorHeight`. `CellProjectionDebt` holds the remaining count per file,
falling only. The remaining sites are sprite, shadow, status-bar and static
placements that take a lift and an origin as separate terms, sub-cell
positions, source-texture spans and products of 32 that are not cells.

## Proof

- Same height, fixture: `TestEditorCrossesStandOnThePathCentreOfASlopedCell`
  raises the anti-diagonal corners of one cell to 60 (anchor 30, corner
  midpoint 0). The unit marker, footprint cross and area centre read from the
  editor's drawn lines equal `pathCellCentre` within 0.01 px. Loss control: the
  former midpoint rule on the same cell is at least 1 px away and fails the
  comparison.
- Same height, install: `TestReleaseEditorCrossesStandOnThePathCentreOfAnInstalledSlope`
  runs the same check on `scenario/20.alm` at the interior cell where the two
  rules differ most: cell (77,75), 21 native pixels apart, EN and RU. Run with
  the former midpoint restored in `cellCenter`, the fixture test fails all
  three crosses (640,384 against the path centre 640,354).
- Family: `TestCellSpaceFlatAndDisplacedCorners` pins flat and displaced
  corners, the centre and the quad to raw arithmetic over signed heights.
- Ratchet: `TestLiveCellGeometryStaysInItsFamily`,
  `TestCellProjectionFindingsSeeEachShape`.
- Game unchanged: the whole EN and RU release sets and the ROM2 save and load
  screen tests; counts in the lane return.
- Owner frames: headless `mapedit -snapshot` frames, EN and RU, before and
  after, on scenario/110.alm Unit:1 and scenario/141.alm Structure:2.

## Open debt

- Sprite, shadow, static, structure and status-bar placement in `pkg/render`
  and their callers in `pkg/ui` pass a lift and an origin as separate terms to
  `terrain` placement functions; they stay in `CellProjectionDebt`.
- A lift at a sub-cell position (`TERR-STRUCT-106`) is not modelled by the
  family; it answers whole cells only.
