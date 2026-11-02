# The Conditions and Variables lists author through the wizard

## Intent

The Scripts list's «Условия» and «Переменные» sections become authoring places. Add then
Condition offers the trigger wizard's condition items with their parameter fields and puts the
item's check into a trigger's first free pair, existing or new, exactly as a wizard row of that
item writes it; Add no longer writes a lone constant 0 node. A condition no pair names gets «Добавить
в триггер». Add then New variable asks a label and takes its register from the one allocator; the
variable is listed at once. Labels are edited in the variable window, shown wherever a variable is
chosen or named, and kept in one file beside the map that the original never reads.

## Authority

- Owner direction (story brief): the results 1 to 8, the out-of-scope list and the returning
  findings. The map editor is our own program: no research and no divergence row.
- Registers 0..99 hold check results and variables; check i takes register i; a variable is
  addressed by check 19 and actions 3 and 8 (`scriptAddressesVariable`). A trigger holds three
  condition pairs (TRIG-CMP-006 for the comparison codes).

## As built

| kind | builder |
|---|---|
| Add window kind (condition, variable) | `pkg/ui/mapeditoradd.go` `addKinds`: the Condition row's steps are data, a `check` step (new control `addItem`) and the trigger step; a new Variable row with a `label` step. Steps marked `Editor` are offered and checked by the editor (`scriptStepOffer`, `scriptStepRefusal` in `pkg/ui/mapeditorscriptadd.go`), not the host; add_record skips them |
| condition item list | the wizard's item table `wizConditions` through `itemChoices`; its headings are shut choices (`InspectionNewStep.Shut`) |
| parameter and comparison fields | `paramField` and `cmpField`, taken out of `wizardFields`; the wizard rows and the Add window's `p<k>` and `cmp` steps ask through them; defaults from `wizardDefaults` |
| parameter check | `wizardParam`, taken out of `wizardNode`; Apply and the Add window's step check refuse through it |
| trigger write | `rowsPlan` (new trigger, Index -1) and `keptPlan` (the trigger's rows kept, the new row after them), then `commandWriteTrigger` and the host's `writeWizardTrigger` |
| trigger pick and full refusal | `triggerChoices` and `triggerFull`, taken out of the area prompt (`areaDrawn`, `areaCreate`), which now uses them too |
| listed condition reader | `loneRow`: the first item whose `readParams` reads the node and whose `wizardNode` writes it back byte for byte |
| variable register allocator | `pkg/game/mapinspectionscriptchoices.go` `scriptTargets`; a labelled register is listed and counted as taken |
| variable list | `pkg/ui/mapeditorscriptauthor.go` `scriptVariables`: registers a node addresses and labelled registers, the same set `scriptTargets` lists as records -2 and below |
| labels state | `editorWorkspaceState.Labels` (undo, redo and dirty with every workspace edit); `liveLabels` keeps a label on an addressed register or one at or above the check count |
| label file | `pkg/game/maptextpages.go`: `mapLabels` reads it when the map opens (`prepareWorkspace`), `placeMapLabels` writes it from `writeEditedMapPages`, the file-beside-map builder of the texts folder |
| words | `pkg/words/text/{en,ru}/strings.toml`: `editor.scripts.totrigger`, `editor.add.variable`, `editor.add.kind.variable`, `editor.labels.*`, `editor.save.labels` |
| static guard | `internal/archtest/maplabels_test.go`; the wizard item guard `internal/archtest/wizarditems_test.go` still holds the one item table |

The Add window's kind step shuts Condition on a map whose script the editor does not read, and
Variable on a host that keeps no labels.

### «Добавить в триггер»

The popup of a condition no pair names shows the button; it opens the Add window on the trigger
step (a trigger or «Новый триггер»), and a full trigger is refused with `editor.area.full`.

- When `loneRow` reads the node as a wizard item, the node enters as that item's row: the pair's
  left side names the node's ID (the host keeps a node whose bytes match), the right side is the
  item's constant, the comparison is the item's own.
- Otherwise the node enters as Details enters a pair: `left<p>` is the ID, `cmp<p>` is 0 (==),
  `right<p>` is a new constant (the trigger record's field set, or a new trigger record); the
  trigger then shows its Details.

### Label file

`<map stem>.variables` beside the map, UTF-8 JSON, two-space indented, ending in a newline:
`{"variables":[{"register":N,"label":"..."}]}`, entries in register order. The reader takes at
most 64 KiB from a regular file and refuses unknown fields, trailing data, a register of 100 or
more, a register twice, and a label that is blank, longer than 64 characters or holds a control
character. A malformed file opens the map with `editor.labels.unread` and no labels, and a save
leaves the file as it is. A label on a register below the check count that no node addresses is
dropped on open and is gone from the next save. A save with no live label removes the map's own
label file and writes none. A label file in the way that is not a regular file is not replaced;
the map is saved and the note says the labels were not written.

## Proof

Focused tests, at 1920x1080 in EN and RU words (`pkg/game/mapeditorscriptadd_test.go`):

- `TestScriptsAddConditionWritesTheWizardRowOfEachParameterKind`: items of every condition
  parameter kind (unit, player, cell, variable, item, group, number, plus structure and area).
  Into an existing trigger, the map bytes equal the wizard's for the same item, and the check takes
  the first free pair. Into a new trigger, the nodes and the pair equal the wizard row's and the
  trigger reads back with the row as the item. The item list holds the item's words, and the steps
  asked are the item's parameters and comparison.
- `TestScriptsAddConditionRefusesAFullTrigger`: the refusal in the editor's words; Cancel leaves
  the map unchanged and clean.
- `TestScriptsListedConditionGoesIntoATrigger`: a check read as «Отряд погиб» (item path) and a
  unit-to-unit check (Details path), each into a new trigger and into an existing one. The pair
  names the ID, every old node, action and other trigger is unchanged, and a full trigger is
  refused.
- `TestScriptsNewVariableIsALabelledRegister`: the register is the allocator's; the map bytes do
  not move; the row and the window («пока не читает и не меняет ни один триггер») show at once. The
  label shows in the list, the wizard's and Details' choices and the row's text; a new condition
  and a second variable take other registers; Label... and Clear label work, and a cleared row reads
  «Переменная N».
- `TestScriptsVariableLabelsFileBesideTheMap`: file content; map bytes equal a save without
  labels; no file without labels; a stale register and a malformed file as above.
- `TestScriptsVariableLabelsFilePlacement`: the writer replaces and removes only the map's own
  file; a save under a new name or a directory in the way keeps what is there.
- `TestScriptsVariableLabelsFileReader`: the reader's refusals and limits.
- `internal/archtest`: `TestCheckMapLabels` (self-test) and `TestMapLabelFileHasOneHome`: the
  `.variables` suffix and the file's name builder and JSON form appear in `maptextpages.go` alone.

Each fails at the base `7401537ae` (tree exported with `git archive`): the test file does not
compile there, and a variant without the new accessors fails every editor test behaviourally (the
Add window asks no condition steps, a full trigger is not refused at a trigger step, the popup has
no «Добавить в триггер», the kind step has no New variable); the reader and placement tests name
functions the base lacks. The guard's live scan finds no suffix.

Moved ordinary tests: `pkg/ui` `TestMapEditorAddKindsAreRowsOfOneBuilder` (the new control),
`TestMapEditorAddCancelAtEveryStepWritesNothing` (kinds the modes map cannot add are skipped),
`TestEditorFieldChoicesCycleWithoutCommitting` (Down passes the shut Condition),
`TestEditorAddRecordCommandBuildsTheAddWindowsRecord` (the window's Condition is no longer
add_record's); `pkg/game` `scriptWindowAdd` adds a bare condition through add_record, and
`TestScriptNewVariableAndOpenStack` presses Back after the variable window's new buttons.

Release test, not run in the lane: `pkg/game.TestReleaseMapEditorScriptVariablesSaveUnchanged`
(population row with no root column): every installed map opens with no note, its Scripts tree
lists the triggers, variables, conditions and actions the script holds, and a save with no edit
writes the map unchanged and no label file. The changed release test
`TestReleaseMapEditorAddWindowWritesTheRouteBytes` compares the condition case with the host's
`WriteTrigger` of the Always check. The next release runs both on EN and RU with
`TestReleaseMapEditorScriptsListUntouchedBytes`, `TestReleaseMapEditorAddWindowFrames`,
`TestReleaseTriggerWizardReadsEveryShippedRowAsAnItem` and `TestReleaseMapEditorComplete*`
(its helper's bare conditions now come from add_record).

## Open debt

- Out of scope: a start value for a variable; deleting a variable that nodes address; the editor's
  MCP commands for triggers (add_record Condition still adds a lone constant node); the one-click
  trigger and the «Сценарий» palette list; the second game's installs.
- A save into a mod writes no label file, and marks the labels clean without writing them.
- When the texts folder fails to write, the labels are not written either.
- A cell or area parameter of a condition is typed in the Add window; there is no map pick.
- The register collision check of a new condition does not count a labelled register no node
  addresses; the allocator gives the highest free register, so a collision needs the checks to
  reach that register.
