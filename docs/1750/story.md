# A mod's maps share global variables that their triggers and replies read, set and add

## Result

A mod can now declare its own global variables, each with a number, a name and a starting value.
The triggers and dialogue replies of the mod's maps read a global, set it and add to it, and a later
trigger compares it like any other value. A mission starts with the values its mod declares.
A map that names a global its mod does not declare is refused by the check, the launch and the save.
A mission save keeps the values; carrying them across missions comes later.

## Authority

- Owner: «a mod declares global variables that its triggers and replies read, set and add».
- Owner: globals are declared in `data/globals.toml`, int32, at most 1024 per mod; a map sees only its own mod's.
- Owner: loss, exit and restart discard the mission's writes; a map that uses a global becomes `.almx`.
- `DIV-5108`: the three global nodes, the seat, the World wrapper and the refusal.

## Rules

1. When a mod's mission starts, the World holds every global its mod declares at its initial value.
2. When a map is played outside a mod, it holds no global.
3. When a set node runs, the global takes the value; when an add node runs, it takes the sum, wrapping at int32.
4. When a dialogue reply names an add node, picking it adds to the global.
5. When a read node is compared, the comparison sees the global's current value.
6. When a node names a global its mod does not declare, it reads 0 and writes nothing.
7. When a mod map names an undeclared global, launch and editor check name file, line, map, trigger or reply, id.
8. When a globals file is malformed, the launch refuses it, naming the file and the line.
9. When the same mission is replayed from the same start and commands, the globals and the World hash are equal.
10. When a World holds no global, its bytes and hash are those it had before.
11. When a script holds a global node, it stays an extended map: folding it to the original records is refused.
12. When a mission is saved and loaded, its globals, trigger state and World hash are those before SAVE.
13. When a SAV holds no global values, LOAD seats the initials; a World without globals saves none.

## Code

| kind | its one builder | change |
|---|---|---|
| mod data kind | `pkg/mod/kinds.go` dataKinds | `globals` row before missions; reader and writer in `pkg/mod/globals.go` |
| mod data applier | `pkg/game/moddata.go` modDataAppliers | `globals` sets the table's declarations |
| World carrier | `pkg/game/worldcarriers.go` worldCarrierSet | "World mod globals" row, entry seat |
| script node | `pkg/scriptprofile/ops.go` op table | check 0x4102, actions 0x4103 and 0x4104 |
| script evaluator | `pkg/sim/script.go` runCheck and the instant executor | read, set and add bodies |
| World form | `pkg/sim/formwrapper.go` formWrappers | outermost `GLB1` wrapper, version 144 |
| map check rule | `pkg/game/mapcheck.go` mapRules | `global-undeclared`; the live `report` reads the mod's globals |
| SAV script state | `pkg/game/currentscriptprogram.go` | `Globals`, restored by `World.RestoreScriptExtendedState` |

`pkg/mapedit/scriptform.go` FoldScript gains the `LimitGlobal` refusal.

## Proof

| test | rules |
|---|---|
| `TestGlobalsMillSeatsTheInitials` | 1, 2 |
| `TestGlobalsMillTriggersAndReplyReadSetAndAdd`, `TestScriptGlobalsSetAddWrapAndCompare` | 3, 4, 5 |
| `TestGlobalsMillUndeclaredIDIsRefused`, `TestGlobalsMillEditorCheckNamesAnUndeclaredGlobal` | 6, 7 |
| `TestGlobalsFileRefusalsNameFileAndLine`, `TestGlobalsFileReadsAndWritesBack` | 8 |
| `TestGlobalsMillReplayIsEqual`, `TestScriptGlobalsReplayEqual` | 9 |
| `TestWorldWithoutGlobalsKeepsItsForm`, `TestScriptGlobalsCrossTheByteForm` | 10 |
| `TestFoldScriptRefusesAGlobalNode` | 11 |
| `TestGlobalsMillSaveAndLoadKeepTheValues`, `TestScriptExtendedStateCarriesTheGlobals` | 12, 13 |

Release tests: `TestReleaseGlobalsMillPlaysOnTheInstall`, `TestReleaseOriginalControlResaveUnchanged` (13).
The shipped-map census is in the review report.

## Open debt

- The campaign carry of the globals and the mod mark store wait for story 1751; mission SAVE and LOAD keep the values.
- The editor's node lists do not offer the global nodes yet.
