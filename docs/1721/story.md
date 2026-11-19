# Edit a second-game map's trigger parameters and save

## Result

On a second-game map in the editor, the trigger window edits the parameters of every action and check whose
operation the research decodes, with the same controls the first game's window uses for that kind of parameter. The
edited map saves, reloads and shows the edit. An operation the research does not decode stays shown with its raw
parameters, is not editable and says why. Adding, removing and renaming triggers and nodes stay off, each with one line.

## Authority

- Owner: «Еще задача это редактор rom2», «Редактор ром2 давай доделаем»; one builder per kind,
  the first game as profile data.
- R2-ENGINE-041, R2-ENGINE-1316: node and trigger layout. R2-ASSET-231: the editor's record 7 size and the type-0 count
  word that sums the script counts, whose parts are Medium. A field graded Medium or Unknown keeps its loaded bytes.
- R2-ENGINE-1312 to R2-ENGINE-1315: the decoded operations. R2-SESSION-019: 1000 registers.

## Rules

1. A parameter of a decoded second-game action or check can be edited; only that parameter's word changes in the file.
2. A not-decoded operation, a node's label, operation or id, and a trigger's other fields refuse edits.
3. A trigger can change its position in the stored order; its bytes move whole.
4. A register index of actions 3 and 8 and check 19 stays under 1000.
5. A second-game map saved with no edit is the installed file, byte for byte.
6. The first game's trigger window and map save are as before.

## Code

| kind | its one builder | change |
|---|---|---|
| trigger window fields | `MapEditor.scriptFields` | on the second game's profile only decoded parameters are editable |
| parameter kinds | `scriptCatalog` | the second game's row takes the node's stored type |
| edit host | `secondEdits.SetFields` | node parameter words and trigger position |
| record write | `mapedit.Editor.SetFields`, `Reorder` | unchanged; the type-7 record has the same grammar |
| map save | `alm.SaveForm` | unchanged |

## Proof

| test | rules |
|---|---|
| `TestReleaseSecondMapParameterEditSavesAndReloads` | 1, 4 |
| `TestReleaseSecondMapOtherScriptEditsAreRefused` | 2, 3 |
| `TestSecondGameNodeFieldsEditOnlyDecodedParameters` | 2 |
| `TestReleaseSecondMapSavesEveryInstalledMapUnchanged` | 5 |
| `TestReleaseFirstMapTriggerEditSaves` | 6 |

Release tests at the release: both second-game installs and the first-game install.

## Open debt

- Adding or removing a trigger or node: a new record's prefix and extra words and the type-0 count word
  are Unknown or Medium.
- Renaming a trigger: the 128-byte prefix label is Medium. Rewiring trigger slots stays off with the wizard.
- Whether the original program runs an edited trigger is untested; no original ran.
