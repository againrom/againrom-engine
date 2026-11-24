# The second-game map editor changes a node's operation and offers wizard rows

## Result

On a second-game map the trigger window now changes a check or action to any other check or action whose parameters
the research reads, and the node takes that operation's parameters. The trigger wizard offers ready rows for the
second game's checks and actions, check 2 with a box, check 21 with a structure, action 6 under every arm and actions
21 and 24 as a spell cast. A stored row the wizard reads shows as its row. A map already holding 1000 checks refuses
a new one in words naming 1000.

## Authority

- Owner: «чтобы мы не имели больше пробелов по триггерам (читаются все триггеры, настраиваются все триггеры, нет
  неизвестных действий и тд)» and «эпики доделывай!»; this supersedes 1754 rule 8 and 1756 rule 7 on ready rows.
- R2-ASSET-363: the type vector of each operation and each arm of action 6.
- R2-ENGINE-1725: a type-9 value is a type-4 object of the map. R2-ENGINE-1741, R2-ENGINE-1742: actions 21 and 24.
- R2-ENGINE-1728: the loader tests no node word; a check's register is its index (Medium).
- R2-SESSION-019: the second game holds 1000 result registers (Medium).
- R2-ENGINE-042, R2-ENGINE-043: action 65538 appends to a compiler list whose reader is Unknown; check 65538 is the
  constant the wizard's right side uses.

## Rules

1. A decoded node changes to an offered decoded operation; its slots become the new operation's signature.
2. A change to an operation outside the table or with no stated slots, or off its signature, is refused.
3. The wizard offers a row for each offered decoded operation whose effect is read, one per distinct arm layout.
4. Action 25, whose terrain effect is unread, and action 65538 get no ready row and stay as they are.
5. A stored second-game row the wizard reads shows as its row; an unchanged row is kept on Apply.
6. A new check past the second game's register count is refused naming that count, with no script-form question.
7. An unedited second-game map saves byte for byte; the first game's editor is as before.
8. A wizard row's values take the trigger window's parameter check: a register outside the profile's count is refused.

## Code

| kind | its one builder | change |
|---|---|---|
| operation row | `pkg/scriptprofile/ops.go` Op, secondActions | Slots, EffectUnread on actions 25 and 65538 |
| group command signature | `pkg/mapload/scriptsignature.go` ScriptSignatureIn | reads Op.Slots |
| node write | `pkg/game/mapinspectionscriptwords.go` setParameterWords | operation and slot types through SetScriptNode |
| node parameter check | `pkg/game/mapinspectionscriptauthor.go` scriptSlotAllowed | one check for a slot type and value |
| register refusal | `pkg/game/mapinspectionscript.go` errChecksFull | the profile's register count |
| script form question | `pkg/game/mapinspectionedit.go` SetFields | asked only where the row offers the seam |
| wizard table | `pkg/ui/mapeditorwizard.go` wizardTables, opItems | second-game rows built from the operation rows |
| trigger window fields | `pkg/ui/mapeditortrigger.go` nodeEditFields, secondNodeFields | the operation field edits |

## Proof

| test | rules |
|---|---|
| `TestSecondMapNodeOperationChangesToCheckAndActionOfTheClaims` | 1, 2 |
| `TestSecondGameNodeOperationChangeRetypesToTheSignature` | 1 |
| `TestSecondGameNodeFieldsEditOnlyDecodedParameters` | 1 |
| `TestSecondGameTriggerWizardOffersTheOperationRows` | 3, 4, 5 |
| `TestSecondMapChecksFullNamesTheSecondGameLimit`, `TestSecondMapRegisterTakenNamesTheSecondGameSlots` | 6 |
| `TestSecondMapNodeOperationRefusesAnOperationWithNoStatedSlots`, `TestSecondGameOperationListHoldsOnlyStatedSignatures` | 2 |
| `TestSecondMapWizardRegisterTakesTheTriggerWindowCheck` | 8 |
| `TestFormQuestionForTheChecksIsInTheEditorsLanguage` | 6, 7 |
| `TestReleaseTriggerWizardOpensShippedTriggers` | 5, 7 |

Release tests at the release: `TestReleaseTriggerWizardOpensShippedTriggers`, `TestReleaseSecondMapTriggerCensus`,
`TestReleaseSecondMapSavesEveryInstalledMapUnchanged`. Census: review/story1758-rom2-editor-script-rest/RESULT.md.

## Open debt

- Whether the original game loads a map with a changed operation is Unknown; no headless instrument runs it.
- Trigger and trap areas: no zone record exists (R2-ENGINE-1685, R2-ENGINE-1730); the area tool stays off.
- The terrain effect of action 25 and the reader of the 65538 list stay Unknown; neither gets a ready row.
- The register of a check is its index (Medium, R2-ENGINE-1728); 1000 registers is Medium (R2-SESSION-019).
- The goals view of actions 4 and 5 and the drops view stay off; a new unit parameter defaults to 10001.
