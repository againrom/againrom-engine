# Place, move and delete on a second-game map

## Result

On a second-game map the editor now places, moves, copies and deletes units, structures and ground sacks. A new
unit or structure is a copy of one the map already holds, set on the chosen cell. A deleted unit takes its sacks
with it, and a deleted structure takes what hangs on it. The map saves and opens again with the change. A map saved
without an edit is still byte for byte the file that was read.

## Authority

- Claims R2-ASSET-253 to R2-ASSET-259 and R2-ENGINE-1324 to R2-ENGINE-1328: ids, sacks, counters, delete.
- Owner: one builder per kind; the first game's tools and save are unchanged.

## Rules

1. A new unit, structure or ground sack takes its place at the cell's centre.
2. Structures and units share one id space; a new one takes the lowest free id from 1.
3. A copied unit brings its own sack, linked to it, with its bound elements.
4. A ground sack is placed with one item; its owner word is 0.
5. The counts in the map header equal the records of structures, units and sacks.
6. Deleting a unit removes its sacks and bound elements and renumbers links; a structure takes its elements and entries.
7. No other id changes, and a script that names a deleted id is left as it was.
8. A structure within 8 cells of the map edge is refused; a move keeps the same bound.
9. Objects, scenery, spells and the dialog fields stay off, with the one refusal line.
10. A map saved unchanged is byte equal on every installed second-game map; the first game's save is unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| map record | `mapedit` section rows | three rows for the second game, with delete and move hooks |
| id | `Editor.takeID` | one allocator over structures, units and sacks |
| edit host | `mapInspectionEdits` Move, Clone, Delete, AddRecord | `secondEdits` hands the three kinds to them |
| cell admission | `admissionFor` | rows for the second game with an edge bound |
| map writer | `alm.SaveForm`, `Editor.SaveBytes` | none |

## Proof

| test | rules |
|---|---|
| `TestSecondPlacedMapSavesUnchangedAndWalks` | 5, 10 |
| `TestSecondUnitCopyTakesItsOwnIdSackAndBoundElement` | 2, 3 |
| `TestSecondStructureCopyTakesTheLowestFreeId` | 1, 2 |
| `TestSecondGroundSackIsPlacedMovedAndDeleted` | 1, 4, 5 |
| `TestSecondUnitMoveCarriesItsTagAndSack` | 1 |
| `TestSecondUnitDeleteTakesSackBoundAndBookElements` | 6, 7 |
| `TestSecondStructureDeleteTakesItsElementsAndListEntries` | 6, 7 |
| `TestSecondPlacementCreatesAbsentRecordsInTheEditorsOrder` | 5 |
| `TestReleaseSecondMapPlacesMovesAndDeletes` | 1 to 8 |
| `TestReleaseSecondMapTerrainEditSavesAndReloads` | 9 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 10 |
| `TestReleaseFirstMapPlacementSavesAsBefore` | 10 |

Release tests at the release: the four `TestRelease` rows above, on both second-game roots and the first game.

## Open debt

- The original program's load of an edited map is not tested; the type-11 default and the object tool wait for it.
