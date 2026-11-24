# The second-game map editor paints heights and passability, clears objects and lays random heights

## Result

On a second-game map the map editor's height brush now raises, lowers and levels the ground, the passability
brush marks and clears blocked cells, and the object brush clears the objects of the cells it covers.
Random heights lays a seeded relief over the whole map. Every one of these leaves the map's 8-cell edge ring as
the map holds it, and a stroke that lies wholly in the ring is refused with its reason.

Picking an object class, laying a bridge and random terrain stay refused there, each with its reason.
A map left untouched still saves byte for byte as installed, and the first game's editor paints as before.

## Authority

- Owner: «эпики доделывай!»; epic «Редактор ROM2», group G2: height, object, passability and bridge brushes,
  random terrain and heights.
- `R2-ENGINE-1666`, `R2-ENGINE-645`: the type-2 record is the height plane the game reads; installed values are
  0 to 127 (`R2-ASSET-261`), so the shared 0..127 clamp reads the same signed or unsigned (Medium, chosen).
- `R2-ENGINE-1340`: the type-1 word's bit 0x2000 sets the cell's block plane; the brush writes that bit alone.
- `R2-ENGINE-1667`: a nonzero type-3 byte blocks movement; which object a value shows is Unknown, so only 0 is
  written.
- `R2-ENGINE-1668`, `R2-ENGINE-1341`: the map band is x 8..W - 9, y 8..H - 9, and the server overwrites the ring;
  the fog walk reads heights from cell 7 (`R2-ENGINE-645`), so the height band of 8 is a choice (Medium).
- `R2-ASSET-262`, `R2-ASSET-263`: no game reader of the bridge element is read; bridges stay refused.
- `R2-ENGINE-1665`: a look-alike tile word may hold another class and terrain cost; random terrain stays refused.

## Rules

1. When a height, passability or clearing stroke is made on a second-game map, the saved map decodes it back.
2. When a stroke reaches the edge ring, ring cells keep their bytes; a stroke wholly in the ring is refused.
3. When random heights runs on a second-game map, only heights inside the ring change.
4. When an object class, a forest, a bridge or random terrain is asked on a second-game map, it is refused.
5. When an installed second-game map is opened and saved untouched, the file is byte-equal.
6. When the first game's editor paints, it paints to the map's edge as before.

## Code

| kind | its one builder | change |
|---|---|---|
| brush stroke on the host | `pkg/game/mapinspectionbrush.go` Paint | plane rules from the edit row, band refusal worded |
| editor row of a format | `pkg/game/mapeditrow.go` editRows | `paints` plane rules and `randomTerrain`, second row data |
| grid brush | `pkg/mapedit/brush.go` Brush.walk, Editor.Paint | `Band` field; ring cells skipped; empty stroke refused |
| format row of the model | `pkg/mapedit/section.go` formatRow | `band`, 8 for the second format |
| whole-map heights | `pkg/mapedit/wholemap.go` RandomizeHeights | the format's band |
| whole-map seam | `pkg/game/mapinspectionterrain.go` RandomTerrain | refused by the row |

## Proof

| test | rules |
|---|---|
| `TestSecondMapBrushesPaintHeightsPassabilityAndClearObjects` | 1 |
| `TestSecondMapBrushRefusesTheEdgeRing` | 2 |
| `TestSecondMapPlaneBrushesKeepTheEdgeRing` | 1, 2 |
| `TestSecondMapRandomHeightsKeepTheEdgeRing` | 3 |
| `TestSecondMapBrushRefusalsAndRandomHeights` | 3, 4 |
| `TestEditRowsHoldBothFormats` | 4, 6 |
| `TestFirstMapPlaneBrushPaintsToTheEdge` | 6 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 5 |

Release tests at the release: `TestReleaseSecondMapSavesEveryInstalledMapUnchanged`,
`TestReleaseSecondMapTerrainEditSavesAndReloads` on the EN and RU second-game installs.
Evidence: `review/story1759-rom2-editor-brushes/RESULT.md`.

## Open debt

- Whether the original game loads a painted map is Unknown; nothing headless runs it.
- Object classes: the join of a type-3 value to an object and the client readers of the copied plane are Unknown.
- Bridges: the game reader of the type-4 element, its extents and the original editor's plane writes are Unknown.
- Random terrain: no claim names which tile words of the second game are look-alikes of one class and cost.
- The height row step of 31 is the first game's rule; the second game's is Unknown.
