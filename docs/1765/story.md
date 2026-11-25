# The second-game map editor searches, tabs and draws its views

## Result

On a second-game map the editor's search finds its units, structures, sacks and triggers, and the legend of the
player colours lists that map's players. The passability view and brush fill the cells the terrain build marks;
the hover names each mark and its value, and the effect on movement is not established.

The border view marks the edge ring, trigger areas are drawn as the cells a node names, and height numbers show the
stored height. A new map made on a second-game install says it is a first-game map. Untouched maps save byte for
byte as before.

## Authority

- Owner: «Редактор ром2 давай доделаем»; group G9 of the editor gap census.
- R2-ENGINE-1341: the edge ring is 8 cells; the meaning of its stored byte is not read.
- R2-ENGINE-1340: the terrain build marks type-1 bit 0x2000, tile class 8, `(word & 0x300) == 0x200` (value 1) and
  a nonzero type-3 byte (value 5); what the mark does to a mover is Unknown.
- R2-ENGINE-1665: the tile classifier is read, but which classes a cell word closes to a mover is Unknown.
- R2-ASSET-231 and R2-ASSET-232: the second game's map writer is read; the layout of an empty map is not stated.

## Rules

1. When a second-game map is shown, the search lists its units, structures, sacks and triggers by name.
2. When several maps are open, the search and the legend of player colours answer from the map shown.
3. When a second-game map shows passability, the fill and hover name the marked cells; the effect is not established.
4. When a second-game map is shown, the border view uses the edge ring.
5. When a second-game script is shown, a node draws one area per cell its slots name, and no box or distance.
6. When a second-game map shows height numbers, each is the stored height; any other map counts from 64.
7. When a new map is made on a second-game install, the editor says it is a first-game map.

## Code

| kind | its one builder | change |
|---|---|---|
| editor seam table | `pkg/game/mapeditrow.go` editRows | second row: ring and mark arms, note word, raw heights, no shapes |
| ground arms | `pkg/mapload/groundarms.go` MarkRows, MarkedArms | the marks are data rows beside GroundArms |
| area decoder | `pkg/mapload/scriptareas.go` ScriptAreas | the shape checks are the game's AreaOps data |
| game files | `pkg/game/gamefiles.go` gameFiles | the new-map note of a game |
| search index | `pkg/ui/mapeditorsearch.go` searchItems | unchanged; a headless reader of its rows |
| legend | `pkg/ui/mapeditorplayertint.go` legendSpec | unchanged; a headless reader of its rows |

## Proof

| test | rules |
|---|---|
| `TestSecondGameMapsSearchAndLegendFollowTheShownTab` | 1, 2 |
| `TestSecondGameSearchFindsATriggerByName` | 1 |
| `TestPassViewSaysTheHostsNote` | 3 |
| `TestSecondMapPassabilityBrushShowsTheMarkedCells` | 3 |
| `TestSecondRowDerivesTheRingAndTheMarksAndNotesThePassabilityView` | 4, 5, 6, 7 |
| `TestScriptAreasOfAGameWithNoShapesDrawNamedCells` | 5 |
| `TestRawHeightNumbersShowTheStoredHeight` | 6 |

Release tests at the release, EN and RU: `TestReleaseSecondMapViewsDrawOnlyWhatIsDecoded`,
`TestReleaseSecondMapSavesEveryInstalledMapUnchanged`.

## Open debt

- Passability fill: what a mark (1 or 5 in the build's plane) does to a ground mover; a unit sent across a marked
  cell would settle it. The tile-class mark (class 8) is not drawn: the class table is not engine data, and the
  class a cell word takes is read in R2-ENGINE-1665.
- Map tabs: a second-game map has no mod project, and tabs open only inside one; the question is the layout of
  a second-game mod's map set.
- A new second-game map: the defaults an empty map holds, which R2-ASSET-231 leaves outside its reading.
- Box and distance areas: the metric of each second-game location check.
