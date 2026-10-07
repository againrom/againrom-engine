# DIV-044 ground picker specification

## Ground column bounds

For a projected cell `(col,row)` and a native world X coordinate, let
`i = floor(worldX) & 31`. The cell's top and bottom rows are:

```text
top    = TL.y + ((TR.y - TL.y) * i) / 32
bottom = BL.y + ((BR.y - BL.y) * i) / 32
```

`TL`, `TR`, `BL`, and `BR` are the cell's four projected corners after the
projection canvas is translated to camera world coordinates. Both divisions
are signed integer divisions truncated toward zero. The bounds are not sorted.

Native column 31 remains one column short of the right corners. Native column
32 belongs to the next cell and has local column 0.

## Ground cell selection

`Viewer.groundCellAt` performs these steps for displaced terrain:

1. Reject a frame point outside the map viewport.
2. Convert the point from screen coordinates to camera world coordinates.
3. Select `col = floor(worldX/32)` and reject a column outside the map.
4. Walk the rows supplied by the current terrain pass in ascending order.
5. For the selected column, compute each row's top and bottom bounds at
   `floor(worldX)`.
6. Return the first row for which `top <= worldY <= bottom`.

Both bounds are inclusive. Two vertically adjacent cells share one edge, so a
point on that edge matches both cells and the earlier row wins. A collapsed
span can match at its one coordinate. An inverted span (`top > bottom`) cannot
match. A point that matches no row returns no cell.

A map without a valid altitude projection remains in flat mode and uses
`Camera.ScreenToCell`. This fallback retains the flat lattice's half-open cell
boundaries.

## Consumers

The following paths use the one `groundCellAt` result:

- hover fog and sack lookup;
- move, patrol, swarm, pick-up, and empty-ground cast click targets;
- inventory item drops onto the map;
- the debug cursor-cell readout, its selected-unit step-cost question, and the
  headless ground-cell query.

Entity selection remains on the entity placement rectangle. Minimap cell
selection remains on minimap geometry. Neither path uses the ground column
bounds.

The picked cell crosses the existing UI-to-game command functions as integer
cell coordinates. The command kinds, movement rules, simulation state, digest,
binary form, and format version are unchanged.

## Error and edge behaviour

- Frame points in the mission side panel or letterbox return no cell.
- Negative coordinates and points on the exclusive right or bottom viewport
  edge return no cell.
- The projection's existing far-edge corner clamp defines the last map column
  and row.
- Non-finite coordinates return no cell before integer conversion.
- An off-map ground drop retains the existing sentinel and simulation fallback.

## Witnesses

The synthetic steep-slope point `(1,80)` selects cell `(0,0)` from the corner
mesh. The replaced mean picker selects `(0,1)`, while the flat camera lattice
finds no map cell there. Literal UI expectations use that point for hover fog,
sack/pick-up, click, ground drop, the cursor readout, and the selected unit's
step-cost target. The two candidate fog rows deliberately have different
visibility, so the fog gate cannot pass without consuming the mesh-selected
row.

The installed EN and RU mission-10 witness uses a separate raw-altitude oracle.
At frame point `(91,64)`, world point `(219,1818)`, the corner mesh selects
`(6,57)` and the replaced mean picker selects `(6,56)`. The production App
queues the corner result and retains a matched pre-advance digest.
