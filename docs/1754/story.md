# Every trigger of a second-game map reads in words and its existing triggers and nodes edit in the map editor

## Result

On a second-game map the trigger window now names every check and action the installed maps hold, by what the
research reads its body to do, and no row says "not decoded". Each parameter edits with the field its stored type takes.
A trigger's name, once flag, comparisons, condition and action slots and position edit, and so does a node's label.

The trigger opens in the wizard: each row is kept as the map holds it, and a row moves or goes; a node no trigger names
any more is deleted. A trigger copies with its nodes byte for byte under new ids; one holding action 65538, a compiler
parameter, is refused in the second game's words. Every add, copy and deletion keeps the map's type-0 word +0x28 at the
sum of the three script counts.

One operand stays read-only: the type-9 parameter of check 21. A new trigger, node or wizard row is refused in words;
win and loss arms, zones, the drops and goals views and the save form question stay off.

## Authority

- Owner: «приоритетно хочу для редактора rom2 чтобы мы не имели больше пробелов по триггерам».
- R2-ENGINE-1313, R2-ENGINE-1314, R2-ENGINE-1315, R2-ENGINE-1450: the read body of each installed operation.
- R2-ASSET-363, R2-ENGINE-1680: the type vector of each operation; types 2, 3, 4 and 8 name a group, player, unit and
  item; the record a type-9 lookup returns is unread.
- R2-ENGINE-1316: the trigger's 184 bytes; R2-ASSET-364, R2-ENGINE-1681: node and trigger texts, no reader found.
- R2-ASSET-255: the type-0 word +0x28 equals the sum of the three type-7 counts in every installed map.
- R2-ENGINE-042: action 65538 adds its cell to a compiler list whatever trigger names it; the list's reader is unread.

## Rules

1. Every operation number of the installed second-game maps is a named row of the one operation table.
2. A parameter of a named operation edits with its type's field; a type-9 parameter is refused with its reason.
3. A trigger's name, once flag, comparisons, slots and position, and a node's label, edit, save and reload.
4. An operation change stays refused on a second-game map.
5. An unedited second-game map saves byte for byte; the first game's trigger window is as before.
6. A trigger copy carries each node's record verbatim under a new id.
7. A copy, a node deletion and every add keep the type-0 word +0x28 at the sum of the three type-7 counts.
8. The wizard reads a second-game trigger as kept rows; a row moves through `moveRow` and goes.
9. Apply of a second-game trigger's rows as read writes the map back byte for byte.
10. A new trigger, node or wizard row on a second-game map is refused in words.
11. A copy of a trigger holding a second-game action 65538 is refused in the word of that operation's row.

## Code

| kind | its one builder | change |
|---|---|---|
| operation table | `pkg/scriptprofile/ops.go` `secondChecks`, `secondActions`, `Vocabulary` | named rows; type 9 unread; the new-record refusal; a row's copy refusal |
| edit host row | `pkg/game/mapeditrow.go` `editRows` | trigger fields, node deletion, copy and wizard seams |
| record sections | `pkg/mapedit/second.go` `init`, `pkg/mapedit/section.go` `tallyParts` | type-7 rows with the +0x28 tally |
| trigger copy | `pkg/mapedit/script.go` `copyNodes` | the node record copied, not rebuilt; the refusal read from the row |
| wizard items | `pkg/ui/mapeditorwizard.go` `wizardTables` | each game's ready rows; the second game has none |
| wizard write | `pkg/game/mapinspectionwizard.go` `writeWizardTrigger` | a new trigger or node refused by vocabulary |
| node write | `pkg/game/mapinspectionscriptwords.go` `setParameterWords` | the label; a type-9 slot refused |
| node fields | `pkg/ui/mapeditortrigger.go` `secondNodeFields` | the label editable; a type-9 slot shown |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondMapEveryOperationFamilyEdits` | 1, 2 |
| `TestReleaseSecondMapTriggerFieldsAndLabelsEdit` | 2, 3, 4 |
| `TestSecondGameNodeFieldsEditOnlyDecodedParameters` | 2, 3 |
| `TestScriptRowsGiveTheNamesDecodedFlagsAndSignaturesTheOldTablesGave` | 1 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 5 |
| `TestReleaseFirstMapTriggerEditSaves` | 5 |
| `TestSecondScriptAddsCopiesAndDeletesKeepWord28AtTheSumOfTheCounts` | 6, 7 |
| `TestSecondScriptCopyOfACompilerParameterIsRefusedInItsOwnWord` | 11 |
| `TestReleaseSecondMapTriggerCopyAndWizardKeepTheScriptTally` | 6, 7, 8, 10 |
| `TestSecondGameTriggerWizardKeepsEveryRowAndWritesNoNewNode` | 5, 8, 10 |
| `TestEditRowsHoldBothFormats` | 7 |
| `TestReleaseSecondMapTriggerCensus` | 9 |

Release tests at the release: both second-game installs and the first-game install. The census instrument is
`TestReleaseSecondMapTriggerCensus`.

## Open debt

- A new trigger, node or wizard row, and win and loss arms: what a written node and trigger hold is unread.
- Trigger and trap zones: no zone record is read (R2-ENGINE-1685).
- A trigger holding action 65538 is not copied: what reads the compiler's list is unread (R2-ENGINE-042).
- The goals view: how a map states victory and defeat is unread. The drops view: action 65538 is no drop zone in the
  second game. The save form question: the loader reads the Againrom script record in the first game's maps only.
- The type-9 record of check 21; the arms of action 6's selector (R2-ENGINE-1314).
