# Place map records from their palettes

## Result

Map makers choose a picture in a palette and click the map to place a record.
The Sacks palette shows item folders in one grid and lists the items ticked for the next sack.
Traps and spells can be placed or copied from their palette, and the condition window chooses its trigger.
The Players list keeps its one-click Add, and a variable label opens a text field.

## Authority

- Owner: «Макеты сперва. Добавить убираем полностью».
- The owner's five answers and route table are in the seat's editor mockup result.

## Rules

1. When U, J, S or F3 is pressed, the corresponding placement palette opens.
2. When a structure or object is placed, it takes the palette's player.
3. When a sack is placed, it holds the ticked items from the folder grid.
4. When a spell is selected, a map click places its copy and C clones the selected record.
5. When a spell needs a unit or structure, its link comes from the clicked record.
6. When a spell has no book icon, the grid shows the number and the tip gives the name in the chosen language.
7. When A is pressed, it adds a player only from the Players list.
8. When Apply is pressed in an unassigned condition, the Trigger row chooses where it goes.
9. When Label is pressed for a variable, it opens a field prompt.
10. When a trap area is drawn, one edit fills it with traps of the palette's spell and power.
11. When the command interface reads steps or adds a record, it uses the same record assembly as the palettes.
12. The Traps and spells palette uses the command interface's templates and spell field bounds.

## Code

| kind | its one builder | change |
|---|---|---|
| Palette grid | `placeGridOf`, `placeGridShell` and `popupGrid` | Ticked items, folder heads and spell numbers |
| Player row | `placeOwnerRow` | Structures and Objects use the Units row |
| Field and condition windows | `promptSpecBase` and `nodeFields` | Label prompt and Trigger row |
| Record assembly | `addKinds`, `newRecordOf` and host `AddRecord` | Palette values and headless commands |
| Spell templates and bounds | `cellSpellPalette`, `spellTemplateKind` and `enchantmentFields` | Where choices, copied spell type and numeric bounds |
| Record clone | `CloneRecord` and the section table | Cell coordinates for known spell records |
| Pictures and tips | host `Pictures`, `spellPictures`, `cutBookIcon` and `popupTipAt` | Spell icons and hover names |

## Proof

| test | rules |
|---|---|
| `TestMapEditorKeysOpenPlacementPalettes`, `TestEditorNativeCellSpellOpensPalette`, `TestMapEditorPlayersAddOpensNoAddWindow` | 1, 7 |
| `TestEditorStructurePaletteMatchesAddRecordBytes`, `TestMapEditorObjectsGridPlacesAndReturnsToBrush` | 2 |
| `TestSackPaletteTicksItemsInOneGridWithFolderHeads`, `TestEditorSackPaletteMatchesAddRecordBytes` | 3 |
| `TestEditorSpellPaletteCopiesSelectedRecordAndCClones`, `TestEditorNativeSpellCloneWhilePlacing` | 4 |
| `TestEditorSpellPaletteStructureClickWritesSignedBinding` | 5 |
| `TestSpellPaletteShowsNumbersAndHoverNamesWithoutBookIcons`, `TestSpellPaletteUsesRussianRecordTerms` | 6 |
| `TestScriptsListedConditionGoesIntoATrigger` | 8 |
| `TestScriptsNewVariableIsALabelledRegister` | 9 |
| `TestEditorTrapAreaMatchesSingleAddsAndDeletesOneStroke` | 10 |
| `TestEditorAddRecordAndAddStepsUseTheRecordBuilder`, `TestEditorSpellPaletteMatchesAddRecordBytes`, `TestSpellPaletteUsesHostTemplateChoices` | 11, 12 |
| `TestChangedPlacementPalettesFitTheEditorTarget` | 2, 3, 5, 6 |

Release test at the release in EN and RU: `TestReleaseMapEditorPlacementPaletteFrames`.

## Open debt

- none
