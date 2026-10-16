# General scenario authoring

## Intent and authority

Complete general ROM1 condition/action/trigger authoring through the existing
editor window and prove its saved mission executes. Owner direction; base
`f35bfc41`, pinned knowledge unchanged. `ALM-TRIG-044` through `ALM-TRIG-047`,
`TRIG-CMP-006`, `TRIG-FIRE-007`, `TRIG-PARAM-030` and `MISSION-SLOT-008`
ground the layout, comparison, latch and register rules. The partial retraction
of `ALM-TRIG-047` concerns the opaque region, which remains carried. Authoring
defaults and refusals are `DIV-3394`; original editor layout is not claimed.

## As built

F2 lists all three script record kinds. A adds Action, Condition or Trigger;
Up/Down chooses the kind before Enter. E edits the selected record. Tab moves
among fields. Up/Down chooses known operations, parameter kinds, node targets,
comparators, variable registers and once/repeat; a numeric value can also be
typed. Names use the existing fixed-width text writer. Delete refuses nodes
still referenced by triggers. Trigger order is an editable field; the four
action slots retain authored execution order. Shared references continue to
name IDs rather than array indices after permitted deletion.

Variables remain the runtime's shared register file: constant/preset conditions
initialize slots, read-variable conditions measure them, and set/increment
actions change them. No second variable section is invented. Overwritten
runtime-check slots and out-of-range registers refuse variable writes.
Condition deletion refuses when it would change an addressed variable slot.

The compiler's support predicate filters the one caption/choice vocabulary.
The inspector reads that same vocabulary. Unknown imported operations, tags,
comparison values and opaque bytes are retained and labelled. An author selects
parameter kinds explicitly; changing an operation does not silently rewrite
its parameter block. Multiple drop actions use the same record field writer.

The empty-map Protect panel now checks for a standalone condition before
indexing the trigger array. No trigger is required to protect the first unit.

## Proof

`TestScriptWindowAuthorsACompoundMissionWithOrderedActions` creates the script
through F2/A/E/Tab, saves ALM and reopens it. Two AND clauses gate ordered
SetVariable/IncrementVariable/Message actions. The next check pass wins. False
AND, reversed actions and a changed shared constant prevent the win.
`TestScriptWindowOnceAndRepeatHaveDifferentLiveEffects` distinguishes one-shot
and repeated increments. Refusal controls cover duplicate IDs, referenced
delete/rename, missing targets, unsupported authoring operations, comparison,
once and variable ranges. Unknown record retention and two independently
edited drop actions have separate controls. Undo/Redo restores exact bytes.
`TestScriptAndWorkspaceEditsShareOneHistory` interleaves script and external
page edits, travels both directions through the shared history and proves a
refused script edit consumes no history slot or page change.

`TestReleaseScriptWindowScenarioContinuesFromSAVColdLOAD` builds the mission
through the production editor, saves into a mod and reopens it. Production
character generation enters mission 20. SAVE after the first action stage
writes SAV; cold LOAD restores the exact World. The next script pass wins.
`TestReleaseEditorTestPlayLaunchEntryWinAndAcknowledgement` captures F5's actual
launch arguments, parses them in the game command, loads the resulting mod,
and uses the production startup generation door. The authored win reaches
Victory; acknowledging it returns to the campaign town square. The editor's
unsaved bytes and owned test mod remain present.

`TestReleaseMapEditorCompleteScenarioPackageSAV` composes the records and
workspace slices with general scripts in one map. N/D/U/J/C/K/I/X/F2/A/E/W
create the map, three grouped units, two structures, stock/ground recipes,
message, variable and win triggers, page and owned replacement package.
Ctrl+S and a fresh editor reopen preserve the accepted ALM save form and page.
Production character generation reaches the script-opened event. Menu SAVE
and cold LOAD preserve all three living actors, their groups, both recipes,
structure identities and variable 99. An unrelated death and the first target
member's death do not win; only the last target member's death does.

Receipts are under `review/rom1-editor-completion/1441/` at the seat. The
combined source includes the exact workspace correction and corrected records
tree `f35bfc41`. Source equality receipts preserve each dependency patch and
show which Go files changed. The inherited dangling-script witness correction
preserves its save refusal and exact stock/unit Undo controls.

| Instrument | Result |
|---|---|
| Protect before the fix, `protect-before.txt` | Reproduced `goalListing` index-out-of-range panic on zero triggers |
| Combined focused tests, `records-combined-focused-01.txt` | PASS: ui 1.173s, mapedit 0.410s, game 2.206s; command compiled |
| EN composite, `composite-en-01.txt` | PASS: witness 0.67s, game 2.277s |
| EN retained release witnesses and help width, `records-combined-release-en-01.txt` | PASS: game 3.839s, command 1.693s |
| RU seven release witnesses, `records-combined-release-ru-01.txt` | PASS: game 3.877s, command 1.551s; composite 0.51s |
| RU help width, `records-help-width-ru.txt` | PASS: game 1.921s |
| Combined modes generation, `records-golden-write.txt` | PASS: 10.575s |
| Guard measurement, `records-guard-measure.txt` | CommentBytes 9044458 -> 9044338; all other counters unchanged |
| Combined ordinary packages, `records-combined-ordinary-01.txt` | PASS: ui 43.165s, mapedit 0.193s, mapload/sim cached, game 95.984s, command 5.238s |
| Inherited dangling-script correction, `dangling-correction-en.txt` and `dangling-correction-ru.txt` | PASS: composite plus refusal witness, EN 1.997s and RU 2.422s |
| Corrected ordinary game and guard, `dangling-correction-ordinary.txt` | PASS: game 92.505s and storyguard 3.733s |
| Strict guards in that invocation | PASS: archtest 19.466s, gatedtests 1.713s, storyguard 3.950s |
| Packed-variable regression, `correction-packed-red.log` / `correction-packed-green.log` | RED: None0 and unknown88 accepted the colliding write; GREEN: focused game 1.162s |
| Correction ordinary game and strict guard, `correction-ordinary.log` | PASS: game 93.114s and storyguard 4.283s; no baseline change |
| Correction EN eight retained witnesses, `correction-release-en.log` | PASS: game 4.538s, command 1.425s |
| Correction RU same eight witnesses, `correction-release-ru.log` | PASS: game 3.990s, command 1.692s |
| `go build ./...`, `records-combined-build.txt` | PASS |
| Asset guard, `records-assets.txt` | `check-no-game-assets: clean (tree scan)` |
| Revision-stamped mission drive, `census-m10.txt`, `census-m20.txt` | Earlier EN one-tick trace: 0 and 0 `UNSUPPORTED` lines; counts 16/27/12 and 14/15/11 match the milestone baseline |

The combined modes golden has 95 paths and 65 actions. Against the script
baseline it retains all 93 paths and 62 actions, adding the two structure
paths and J/K/I. Against the records baseline it retains all 94 paths and
64 actions, adding the script path and F2. No existing path, action or
independent key/mode/layout/refusal control was removed.
`records-combined-mode-population.json` records retained populations and
changed cells for existing actions. The F1 all-mode correction is the exact
workspace landing source. Script and ordinary record choices use its one
choice builder; script choices no longer overwrite installed record choices.
The manifest retains all dependency witnesses and adds the two script/F5
witnesses plus the combined scenario. The guide is in `docs/MODS.md`.

The sole review returned F1: a leading None or unknown parameter made the
variable guard validate an inactive value instead of the runtime's packed
target. The correction limits that read to integer/X/Y/constant tags1/5/6/7,
matching the existing binder. Runtime dispatch and imported bytes are unchanged.
`TestScriptPackedVariableRefusesCollisionWithoutChangingHistory` failed on the
reviewed source for both None0 and unknown88, then passed after the correction.
It creates the None form through F2/A/E/Tab, saves and reopens both forms, and
checks window/public-API refusal of register0 with exact bytes, clean checkpoint
and an armed Undo/Redo edit retained. The same saved free-register98 form writes7
without winning while registers0/99 remain0. The final chain, main landing and
promotion remain pending.

## Open debt

No original program or owner desktop was driven. F5 launch uses a recording
process adapter; the captured arguments and actual command entry are exercised
headlessly. This proves the command lifecycle without claiming an OS process
or visible window. Unknown imported operation meaning and parameter schemas
stay unknown. ROM2 authoring remains outside this story.

## Merge interfaces

`ui.InspectionChoice` has `Value int64` and `Label string`.
`ui.InspectionField.Choices` is a slice of those rows. Numeric field prompts
cycle it through `stepChoice`; `choiceTitle` supplies the current caption.
The workspace dependency supplies this foundation in `mapeditorchoices.go`;
duplicate declarations in the script files were removed.
`ui.InspectionScriptAuthor.AddScript(kind string)` returns the changed document.
F2 uses key `scripts`, filter 9, and `promptNewScript`; A routes to this host
only in that filter. Trigger fields join `fieldKind` and safe Delete.

`mapload.ScriptOperations` and `ScriptOperationName` are the one authoring and
inspection vocabulary, filtered by `sim.ScriptOperationSupported`. The typed
parameter and comparator names are in that same file. The runtime support
tables and dispatch are unchanged. `mapedit.Editor.Reorder` moves an existing
section record in one edit. `CheckScriptPlayer` prevents script player fields
from reaching the unit-specific owner validation. The script host adds guards
and choices in existing Fields/SetField/Delete; it does not alter history or
external text state. Shared files need additive reconciliation with workspace
history/field dispatch and records owner/identity handling. The preliminary
combined source retains those transactions and the separate structure owner
validation. The seat verifies the published parent before candidate creation.
