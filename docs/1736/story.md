# The map editor edits the fields of a second-game unit, structure and sack

## Result

On a second-game map the editor now opens a field window for a unit, a structure and a sack. A unit lists its owner,
definition, health and maximum health, flags and group; a structure its type, owner and, for the large kind, its two
extension numbers; a sack its word 0x10 and each item with its second word. A field whose meaning the research has not
found shows as "Field 0x0a" and its offset, editable as a number. The record card lists the same fields, so the
technical lines are gone. An edit changes only that field's bytes and Undo restores them.

## Authority

- Owner: epic «Редактор карт ROM2», group G1 of the gap list: edit the fields of a unit, a structure and a bag, the
  record card built from fields.
- R2-ENGINE-1510, R2-ENGINE-1511, R2-ENGINE-1512, R2-ENGINE-1513, R2-ENGINE-1514: row slots the builder reads, the
  health pair, the definition lookup, the item builder tables, the row getters.
- R2-ASSET-313, R2-ASSET-314, R2-ASSET-315, R2-ASSET-316, R2-ASSET-317, R2-ASSET-318, R2-ASSET-319, R2-ASSET-320:
  value sets, owner and flags, health pair, definition join, structure kinds and extensions, sack items, table sizes.
- The health name is Medium (R2-ENGINE-1511). Units fields 0x08, 0x0a, 0x10, 0x20, the structure word 0x0c, its
  extension numbers and the item second word have an Unknown meaning and keep neutral names.

## Rules

1. When a second-game unit, structure or sack is opened, its window lists the fields the claims give, by plain name.
2. When a field is edited, only that field's bytes change; the saved map reloads with the edit and Undo restores it.
3. When a record card is shown, it lists the record's fields; no raw line of coordinates, flags or extension remains.
4. When a unit's definition or a structure's type is chosen, the choices are the Data.bin rows by catalogue name.
5. When a sack item code is set, a code naming a Data.bin item row of the second game is accepted, any other refused.
6. When a first-game map is opened, its unit fields and cards are as before.

## Code

| kind | its one builder | change |
|---|---|---|
| record field list | `pkg/game/mapinspectionedit.go` Fields:367 | second row lists unit, structure, sack fields |
| per-game field table | `pkg/mapedit/second.go` secondUnitFields:71, secondStructureFields:97, secondLootFields:120 | fields as data |
| record card | `pkg/game/mapinspectionedit.go` recordFieldDetails:481 | card built from Fields when the row asks |
| edit row | `pkg/game/mapeditrow.go` editRows | second row gains field kinds and field cards |
| item name | `pkg/mapload/itemname.go` ResolveItemName | a second-game code is named by its Data.bin rows |
| field choices | `pkg/game/mapinspectionchoices.go` fieldChoiceRows | definition and type rows |

## Proof

| test | rules |
|---|---|
| `TestSecondUnitFieldsEditSingleFieldSaveReloadAndUndo` | 2 |
| `TestSecondStructureFieldsEditSingleFieldReshapeSaveReloadAndUndo` | 2 |
| `TestSecondSackFieldsEditItemsSaveReloadAndUndo` | 2 |
| `TestSecondEditorFieldsAndCardsForUnitStructureAndSack` | 1, 3, 4, 5 |
| `TestFirstEditorUnitFieldListUnchanged` | 6 |
| `TestReleaseSecondMapEditsUnitStructureAndSackFields` | 1, 2, 3, 4, 5 |
| `TestReleaseFirstMapUnitFieldsUnchanged` | 6 |

Release tests at the release: `TestReleaseSecondMapEditsUnitStructureAndSackFields`,
`TestReleaseFirstMapUnitFieldsUnchanged`.

## Open debt

- none
