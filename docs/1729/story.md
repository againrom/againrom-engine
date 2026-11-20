# Selection, copy, paste and groups on a second-game map

## Result

On a second-game map the map editor's selection frame, Ctrl+C, Ctrl+V, groups 1-9 and Ctrl+1-9 work for units,
structures and ground sacks. A pasted element is written as a placement writes it, and a pasted unit keeps its own
sack. The store and item recipe tools and the area copy (terrain and objects) stay off on that map.

## Authority

- Owner: the second-game map editor epic, and «Как рекомендуешь» on starting with the selection group.
- Engine rule «One builder per kind»: the first game's selection, paste and group builders, the second game as data.

## Rules

1. A second-game map takes the selection's move, copy, paste and delete for units, structures and ground sacks.
2. A pasted unit and structure take the lowest free ids, the map counters follow, and a pasted unit's sack is its own.
3. A pasted element stands at the centre of its cell; a structure within 8 cells of the edge is refused.
4. A unit's group number is the unit record's word at +0x2c (R2-ASSET-119); a group edit writes it on each unit.
5. The store, recipe and area copy tools do nothing on a second-game map; the first game's tools are as before.
6. An unchanged second-game map saves byte for byte equal.

## Code

| kind | its one builder | change |
|---|---|---|
| group selection edit | `mapInspectionEdits` selection methods | cell words and admission read the map's row |
| capability of a format | `editRows` in `pkg/game/mapeditrow.go` | the second row offers selection and groups |
| unit field of a format | `secondUnitFields` | the group field |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondMapCopiesPastesAndGroups` | 1, 2, 3, 4 |
| `TestReleaseSecondMapStructureEdgeRuleHoldsForPasteAndMove` | 3 |
| `TestEditRowsHoldBothFormats`, `TestHostOffersEverySeamOnTheFirstRow` | 5 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 6 |
| `TestMapEditorPasteEqualsOneClonePerRecord` | 5 |

## Open debt

- The original program's load of an edited map is not tested.
- The sack record's form is not read for a store or recipe; the area copy would carry planes the notes do not read.
