# One record card in the map editor for both games

## Result

In the map editor, selecting a unit, structure or sack on a first-game map builds its record card
through the same field-list card builder the second game uses. The first game's card still shows
every stored value it showed before: raw coordinates, raw record values, definition and sack items,
with labels in English and Russian. Two derived lines, the definition row and the frame count, are
gone. An unresolved unit definition and the choice lists each have one path for both games.

## Authority

- Owner: architecture audit kind N98 "record card and record fields in the map editor" — one record card
  builder and one choice-list builder for both games, driven by per-game data tables, with every new card
  label present in both English and Russian string tables.

## Rules

1. Every unit, structure and sack record card in both games is built from its format's card field table.
2. First-game card field tables list raw coordinates, raw values, definition and items without altering edit fields.
3. Every record card label resolves in both English and Russian string tables.
4. An unresolved unit definition formats its warning through one per-format warning path.
5. Unit definition and structure kind choice lists use one catalog builder and keep each game's list contents.
6. Second-game record cards and editable field lists keep their existing values and labels unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| record card fields | `pkg/mapedit` `Editor.CardFields` | added `cardFields` hook and first-game card tables (`unitCardFields`, `structureCardFields`, `lootCardFields`); second game uses `fields` |
| record card builder | `pkg/game/mapinspectionedit.go` `recordFieldDetails` | reads `CardFields` and `Record` for both games; removed `fieldCards` flag and raw `fmt.Sprintf` branches in `inspectModel` |
| unresolved unit warning | `pkg/game/mapinspection.go` `inspectModel` | replaced two warning branches with one path reading `editRow.unitNoRowWord` and `editRow.unitNoRowID` |
| choice lists | `pkg/game/mapinspectionchoices.go` `buildChoiceList` / `choiceList` | unified `unitDefinitionChoices`, `structureKindChoices` and `recordCatalog` kind/definition cases over `gameFiles.choiceCatalogs` |
| structure script target | `pkg/game/mapinspectionchoices.go` `fieldChoiceRows` | the field offers no choices, so a card never names a unit for it |
| localized card labels | `pkg/words/text/{en,ru}/strings.toml` | added `editor.unit.nodef` and `editor.card.{x,y,class,sub,unitid,items}` in both tables |

## Proof

| test | rules |
|---|---|
| `TestFirstCardFieldsReadAllRecordValues` | 1, 2 |
| `TestFirstEditorRecordCardsShowAllValuesInBothLanguages` | 1, 2, 3, 4 |
| `TestChoiceCatalogsKeepBothGamesContents` | 5 |
| `TestSecondEditorFieldsAndCardsForUnitStructureAndSack` | 1, 5, 6 |
| `TestFirstEditorUnitFieldListUnchanged` | 2 |
| `TestFieldChoiceRegistryGivesTodaysLists` | 5 |
| `TestEditRowsHoldBothFormats` | 1, 4 |
| `TestShippedTablesLackNoID` | 3 |

Release tests at the release: `TestReleaseMapInspection1092EveryAuthoredPlacement`,
`TestReleaseMapEditor1118MoveUndoRedoSaveAsReopen`, `TestReleaseSecondMapEditsUnitStructureAndSackFields`,
`TestReleaseFirstMapUnitFieldsUnchanged`, `TestReleaseSecondMapPalettesListTheCatalogAndNameEveryPlacement` (EN, RU).

## Open debt

- none
