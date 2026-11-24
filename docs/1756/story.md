# The second-game map editor writes new triggers and nodes and edits every operand the research reads

## Result

On a second-game map the trigger window now adds a new trigger, check or action. A new node holds the operation's
name as its label and zero in the words no reader is found for. The parameter of check 21 edits as the id of an object
on the map, and an id no object holds is refused. Action 6 shows its arms by name, actions 21 and 24 show as a spell
cast from a cell at a cell or a unit, 16 other actions and actions 4 and 5 show what they do, and check 2 is a unit in
a rectangle. A trigger holding action 65538 copies and names that node itself.

## Authority

- R2-ENGINE-1725: a type-9 value is the identity of a type-4 object of the same map; an absent one fails the compile.
- R2-ENGINE-1726: the arms of action 6 by selector. R2-ASSET-363: each arm's type vector.
- R2-ENGINE-042, R2-ENGINE-1727: action 65538 adds its values to a compiler list whatever trigger names it.
- R2-ENGINE-1728: the loader tests no node or trigger word; a new check's register is its index.
- R2-ENGINE-1729: action 4 adds to the win counter, action 5 stores the failure reason.
- R2-ENGINE-1730: check 2 tests a unit against a rectangle; no zone record exists.
- R2-ENGINE-1740 to R2-ENGINE-1749: actions 21 and 24 cast a spell; the effects of the 16 other actions.

## Rules

1. A new trigger, check or action on a second-game map is written and keeps the type-0 word +0x28 at the three counts.
2. A new node holds its operation's name as label and zero in word +0x48 and its texts.
3. A type-9 parameter edits; a value no type-4 object of the map holds is refused in words.
4. Action 6 shows its arm by the second game's word for the selector.
5. Actions 21 and 24 show as a spell cast with cell, unit, spell and power parameters.
6. A copy of a trigger holding action 65538 names that node itself and copies the rest under new ids.
7. The wizard writes a new second-game trigger with new nodes; its ready rows stay the first game's only.
8. An unedited second-game map saves byte for byte; the first game's editor is as before.

## Code

| kind | its one builder | change |
|---|---|---|
| operation row | `pkg/scriptprofile/ops.go` secondChecks, secondActions | names, signatures, check 2, CopyNamed |
| parameter vocabulary | `pkg/scriptprofile/ops.go` vocabularies | ObjectType 9 of the second game |
| group command signature | `pkg/mapload/scriptsignature.go` ScriptSignatureIn | reads the row's CommandSignatures |
| node copy | `pkg/mapedit/script.go` copyNodes | a CopyNamed node keeps its id |
| node parameter check | `pkg/game/mapinspectionscriptauthor.go` scriptObjectAllowed | object identity held by the map |
| edit row | `pkg/game/mapeditrow.go` second row adds | trigger, action and check |
| group command name | `pkg/ui/mapeditortrigger.go` commandName | the row's CommandWord |

## Proof

| test | rules |
|---|---|
| `TestSecondMapNewTriggerAndNodesWriteTheFieldsTheClaimsState` | 1, 2 |
| `TestSecondMapWizardWritesANewTriggerWithNewRows` | 1, 2, 3, 5, 7 |
| `TestSecondGameNodeFieldsEditOnlyDecodedParameters`, `TestSecondGameOperationNames` | 3, 4, 8 |
| `TestSecondGameTriggerWizardKeepsEveryRowAndWritesNoNewNode` | 7 |
| `TestSecondScriptCopyNamesTheCompilerParameterItself` | 6 |
| `TestScriptRowsGiveTheNamesDecodedFlagsAndSignaturesTheOldTablesGave`, `TestOpRowsCarryTheOldWordKeysAndRegisterSlots` | 4, 5, 8 |
| `TestEditRowsHoldBothFormats` | 1 |

Release tests at the release: `TestReleaseSecondMapTriggerCensus`,
`TestReleaseSecondMapTriggerCopyAndWizardKeepTheScriptTally`, `TestReleaseSecondMapTriggerFieldsAndLabelsEdit`,
`TestReleaseSecondMapSavesEveryInstalledMapUnchanged`.

## Open debt

- Whether the original game loads a map with a new trigger is Unknown; no headless instrument runs it.
- The wizard offers no ready rows for the second game; a new row is added in the trigger window.
- Trigger and trap areas: no zone record exists; check 2 is a row, the area tool stays off.
- The terrain effect of action 25, the goals view of actions 4 and 5, and the drops view stay unread or off.
- The register of a new check is its index (Medium, R2-ENGINE-1728); the reader of the 65538 list is Unknown.
