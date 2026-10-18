# Object script uses

## Intent and authority

The owner requires every unit, structure, group and player popup to list the
scripts that name it and open those scripts. The editor is owner-directed.
Its interface is not a reconstruction of the original editor.

The knowledge pin stays at k326. `ALM-TRIG-046` establishes parameter kinds,
reference domains and the unit bands at Medium, with the item-domain amendment
stated separately in that claim. `ALM-TRIG-047` establishes trigger links by
node ID at Medium. Its partially retracted opaque-span interpretation is not
used here. Neither claim establishes an original-editor reverse-reference UI.
The authored interface is recorded in DIV-3641.

## As built

`scriptTargets` supplies the host choices. `InspectionScriptModel.OpenTarget`
is the shared admission rule for parameter Open and the reverse index.
`inspectScriptUses` attaches each admitted parameter's node, slot and referring
triggers to that target. It reads the projected document and writes no ALM data.
Every projection, including after Apply, Undo and Redo, rebuilds the index.

Unit, structure, group and player popups show Used by (RU: Используется в).
Trigger links follow trigger file order. Each trigger appears once, with its
condition nodes followed by its action nodes in array order. Repeated slots
and repeated trigger references do not repeat a node caption. Long captions
continue on text rows; only the first row is the trigger link. Nodes without
a trigger follow, once per node, labelled No trigger (RU: Без триггера), and
open that node. An empty section says Used by: none (RU: Используется в: нет).

Group matching uses the group number, as parameter Open does. The groups list
also divides members by player; each such popup shows the uses of that group
number. Group zero, missing or ambiguous record identities, hero/named-unit
values from 10001, and items have no admitted Open and no reverse entry.
Items have catalog rows but no popup and are outside this slice.

`scriptLinkSection` builds every section, including the variable popup's
existing Changed by triggers and Read by triggers sections. Those variable
sections retain their previous rows, links, wrapping and empty labels.

Used by navigation uses the existing object stack. It suspends the source
popup's field buffers without applying them. Back reloads the current source
record and restores its changed buffers by field key. The source scroll position
returns too. Other navigation retains its existing Apply-first rule.

The trigger and node layouts, G panel, area tools, simulation and persistence
formats do not change. This slice retains the integration base's version and
k326 pin.

## Proof

`TestObjectUsedByProductionPopup` first failed against the base: a placed unit
named by the Rule trigger's objects node had no reverse section. The production
route is `MapInspector.project` to `objectSpec`, its link click, and Back.

`TestScriptUsedByKinds/{Unit,Structure,Group,Player}` constructs both condition
and action references for each kind. It checks parameter slots, trigger order,
node order, deduplication, shared nodes, unattached nodes, actual row clicks and
unchanged source bytes. A separate loss overlay removes each kind from the
host index; that kind's named subtest must fail.

`TestScriptUsedByEmptyAndUnopenable` covers empty objects, unresolved identities,
group zero, a hero and an item. `TestScriptUsedByBackPreservesDraft` types an
unapplied unit field, opens its trigger, presses Back and checks both the buffer
and unchanged ALM bytes. Existing variable-popup tests pin its complete rows
and link arrays. `TestScriptUsedByRebuildAfterApplyUndoRedo` retargets an action
and verifies the old and new targets after Apply, Undo and Redo.

The mode-table control compares the object-section change against engine base
`5bb70bba`, with the panel hash omitted while still rendering the panel.
All 101 path rows match. The pixel golden is
regenerated for the added section. The group-member test helper now counts
only links to units, excluding section headings and script links.

The EN and RU installed witnesses are
`TestReleaseMapEditorUsedByCensusPreservesBytes` and
`TestReleaseMapEditorUsedByFrames`. The census compares every reverse parameter
entry against a forward scan through the host resolver, validates popup link
order for every unit, structure, player and player/group popup, then writes
each map through Save As and compares every byte. Frames use the installed
campaign map and fonts through the headless renderer.

Gate receipts, measured totals, loss logs and inspected frames are kept in
`review/story1465-used-by/` beside the brief, outside the repository.

## Open debt

The brief asks for both one link per trigger and link-count equality with
parameter-count. Those are different populations: a trigger can name a target
several times, a node can be shared, and a node can have no trigger. The chosen
interface keeps one trigger link and lists all its nodes. The census proves
parameter-entry equality and popup-link equality separately and logs both
counts. It also logs link sums across every opened popup separately from sums
over distinct host targets, since player subgroups can share a group number.
The owner clarification remains in the notes directory.

Original-editor reverse-reference interaction remains Unknown. No game-window
or GPU-window acceptance is claimed. The active-pause GPU release witness is
excluded by a disclosed test-only overlay in the notes directory to honor the
owner's windowless fence; it remains a named not-run witness.
