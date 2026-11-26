# The map editor refuses every extended-only operation at one step

## Result

The map editor now treats the four things only an extended map runs alike: an interactive dialogue, a mod global,
the check that a unit left its cell, and the teleport action. On a map in the original format, the edit that adds one
asks to extend the map, in that operation's words. A message turned into a dialogue at its own field is now asked
there, not only in the wizard. Saving such a map that already holds one asks the same question, and a refused fold
names the reason in the editor's language. Maps, scripts, play and saved bytes are otherwise unchanged.

## Authority

- Owner: «Как рекомендуешь» on the architecture audit at `1a214825b`, item 1: one table of extended-record
  operations and one refusal check, before the global-variables epic.
- DIV-5108, DIV-5200, DIV-5204: the global, teleport and unit-left-cell operations run only on the extended record.

## Rules

1. On an original-format map, an edit that adds a node of an extended-only operation asks to extend, in its words.
2. An edit that adds no such node is not refused, even on a map that already holds one.
3. Saving an original-format map that holds such a node asks to extend; folding one is refused in its words.
4. A new extended-only operation is one row of the table; the editor and the simulation read it with no other code.
5. The simulation runs each operation on the machines it ran on before; the same maps load and save the same bytes.

## Code

| kind | its one builder | change |
|---|---|---|
| extended-record operation | `pkg/scriptprofile/extendedops.go` ExtendedOps | new table; op ids named once |
| script refusal | `pkg/mapedit/scriptform.go` ExtendedOnly | checks added nodes per row under the game's binding rule |
| refusal of the original record | `pkg/game/mapinspectionscriptform.go` originalRefusal | asked by every edit and the save |
| script edit | `pkg/game/mapinspectionedit.go` change | asks originalRefusal; message page writes route through it |
| refusal words | `pkg/game/mapinspectionwizard.go` checkWords | the row's words |
| trigger wizard | `pkg/game/mapinspectionwizard.go` wizardRoom | per-kind counts removed |
| script fold | `pkg/mapedit/scriptform.go` FoldScript | asks ExtendedOnly |
| simulation gates | `pkg/sim/script.go` supportsCheck, supportsInstant | read the row's Original in compile, dispatch and trace |

## Proof

| test | rules |
|---|---|
| `TestEveryExtendedOperationIsRefusedAtTheEditThatWritesIt` | 1, 2, 3 |
| `TestMessagePageAndWriteTriggerRefuseExtendedOperations` | 1 |
| `TestOriginalSaveDialogueRefusalMatchesCompilerInBothGames`, `TestExtendedOnlyMatchesCompilerDialogueBinding`, `TestInteractiveMessageMatchesCompilerInBothGames` | 1, 3 |
| `TestWizardRefusesNewGlobalActionWhenRemovingGlobalCheck`, `TestExtendedOnlyRefusesAddedNodeWhenAnotherIsRemoved` | 1, 2 |
| `TestExtendedOnlyRefusesARiseOfARow` | 1, 2 |
| `TestATestOnlyExtendedRowIsRefusedTheSameWay`, `TestFoldRefusesATestOnlyExtendedRow` | 3, 4 |
| `TestScriptGatesReadTheExtendedRow`, `TestExtendedOpOriginalFlagGatesCompileDispatchAndTrace` | 4, 5 |
| `TestFoldScriptRefusesAGlobalNode`, `TestFoldScriptRefusesATeleportNode`, `TestFormQuestionForABareLimitNamesTheLimit` | 3 |
| `TestFirstGameTeleportActionWritesAlmxReadsBackAndPlays`, `TestUnitLeftCellInAPlainMapRequiresConversion` | 1, 3, 5 |
| `TestExtendedOpsHaveOneTable` | 4 |

Release tests at the release: none.

Review report: review/story1780-extended-op-table/RESULT.md.

## Open debt

- The loader still decides a dialogue's extended-only reading by the message opcode and its own constant.
