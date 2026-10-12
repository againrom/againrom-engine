# The map editor moves the drop cell and sets what wins and loses a mission

## Intent

In the first game's map editor the author moves or adds the party's drop cell
and sets what wins and loses the mission: the target of the win arm (the unit,
structure or group whose death wins) and a protect-unit check (the unit whose
death loses). The saved map plays as a mod mission: the party lands at the new
drop cell, only the new target wins, and the protected unit's death loses. The
story started from story 1423's candidate `1c493ce7` (main `b006300e`, release
0.120.0, pin k275).

## Authority

Owner direction (map editor, priority; audit item I5). Claims: `MISSION-DROP-002`
(High), `MISSION-WIN-003` (High), `MISSION-VIP-004` (Medium), `ALM-TRIG-044` to
`ALM-TRIG-047` (Medium; `ALM-TRIG-047` partly retracted in its 64 opaque
bytes, which the editor never touches), `ALM-TRIG-046` for the identifier
domains. Rows: `DIV-3183` (script nodes), `DIV-3184` (drop cell), `DIV-3185`
(win and lose arms), `DIV-3186` (protect check).

Choices the claims leave open (audit question Q3):

- The win arm. `MISSION-WIN-003` fixes instant 4 as the only win but no claim
  names a "target". The editor reads the arm from the installed maps' own
  shape: a trigger that runs instant 4 (or 5) names condition nodes by id in
  its three pairs, and the target is the Target_Unit, Target_Group or
  Target_Structure parameter of those nodes. Installed census over 38 EN maps:
  41 win, 15 lose and 10 protect target parameters (release test log).
- A win that reads a script variable another trigger sets is not followed.
- Several drop actions. `MISSION-DROP-002` picks at random; the editor moves
  the first and leaves the rest as node fields.
- A protect check is a check 18 node with no trigger (`MISSION-VIP-004`: armed
  by authorship). The claim is Medium; the register rule of `MISSION-SLOT-008`
  is guarded.
- A node's unwritten bytes. Neither label nor slot names of a new node have a
  claim; they are zero (`ALM-TRIG-045`: the label tail is uninitialised editor
  memory).

## As built

### One table, one carrying rule

The type-7 payload is three counted arrays (`ALM-TRIG-044`). `pkg/mapedit`
adds two rows to the one section table, `KindAction` and `KindCondition`
(`script.go`). A row may carry a `layout`: it finds the array's count word and
first record inside the payload, and refuses a payload the three arrays do not
tile. A row without one keeps the type-0 count word and a payload that is the
array. Place, Delete, Record, Count and the field table read only the row;
the count word moved by Place and Delete is the walk's `countAt`.

`SetFields` writes several fields of one record as one edit (`SetField` is its
one-field case), so one Undo reverts them together. A node's fields are one
per typed parameter slot (`value0` to `value9`); unchanged bytes are carried,
and an edit writes only the named word. Three new field checks name the
reference rule a host runs: `CheckScriptUnit`, `CheckScriptStructure` and
`CheckScriptGroup`.

| operation | bytes written |
|---|---|
| set a target (`SetField` on `value<i>` of a condition) | the 4-byte value word at node +0x4c+4i |
| move the drop cell (`SetDropCell`, existing node) | the two 4-byte words at node +0x4c and +0x50, one history entry |
| add a drop cell (map without one) | a 796-byte action appended to the action array, the action count word and the payloadSize word (+796 each in size) |
| add a protect check | a 796-byte condition (opcode 18, id one above the highest, slot 0 = Target_Unit) appended to the condition array, the condition count word and the payloadSize word |

### The host

`pkg/game/mapinspectionscript.go`: `Goals()` lists the arms, `SetGoalTarget`
sets one or adds the protect check, `SetDropCell` moves or adds the drop cell.
All go through `mapInspectionEdits.change`. `editKinds` gains `Action` and
`Condition`, so the field prompt (E) works on a script record too, with the
reference checks of `checkScriptReference`. A picked record is refused when
its id is 0 or another record shares it (the script would name the other).
`InspectionDocument.Drops` carries the drop cells for the map marker.

### The window

- D arms the drop cell on the next clicked cell; the cell is marked with a blue
  box and cross.
- G opens the script panel in the report's list: rows `Win`, `Lose`, `Protect`
  with the trigger or check, the operation and the current target; the last row
  adds a protect check. Up and Down select (the map scrolls to the target),
  Enter arms the pick, a click on a unit or structure sets the target, and the
  panel stays open on the row. G or Escape closes it.
- The help rows were three lines clipped at the rail width; they are now one
  list (`ui.EditorHelpLines`), each line within 300 px
  (`TestReleaseMapEditorHelpLinesFitTheRail`), and the rail reserves four
  bottom rows.

### The report

No new checker. A drop cell outside the map or on ground the party cannot
stand on is the `drop-cell` warning; a target number the map does not hold is
the `script-reference` error and refuses the save
(`TestAnEditThatBreaksTheDropCellOrAReferenceIsReported`). The window's own
operations pick records from the map and refuse a drop cell outside it, so only
the node fields (E) can produce these findings.

## Proof

- `pkg/mapedit` (`script_test.go`): script rows, one-word target edit with undo
  and redo to the exact bytes, refusals writing nothing, drop cell as one
  history entry, appended action, appended condition leaving the trigger array
  byte for byte, scripts that do not tile.
- `pkg/game` (`mapeditscript_test.go`): the arms listed, a retarget, a group arm,
  refusals, protect check add and undo, the register refusal, drop cell move and
  add, the window keys end to end, an unedited save writing the bytes, the
  report on edits that break a rule.
- Release, EN and RU (`mapeditscript_release_test.go`):
  `TestReleaseMapEditorMovesTheDropCellAndRetargetsTheWinArm` (mission 131: the
  installed map lands at its own drop cell and its target's death wins; the
  edited map, saved into a mod through the window's Save, lands the party at the
  moved cell, the old target's death does not win in 600 frames, the new
  target's does), `TestReleaseMapEditorSetsAProtectUnitCheck` (the death of a
  unit the script never names does not lose the installed mission and loses
  the mod's), `TestReleaseEveryInstalledMapListsItsScriptArmsAndWritesItsBytes`
  (all installed maps: arms listed, unedited save equals the installed bytes).

## Open debt

- A win that is only variable-gated cannot be retargeted from the panel; the
  author edits the node fields.
- Edits of other script fields (opcode, comparison codes, trigger pairs, new
  triggers, the 64 opaque trigger bytes) are not offered.
- A map with several drop actions moves only the first.
- A retarget edits the condition node itself; other triggers that name the
  same node (ten shared nodes in EN and ten in RU, for example the win node of
  EN map 60 and six more triggers) change with it, and the panel does not show
  them (review note 1, `DIV-3185`).
- Whether the builder compiles a check the editor appends (the `MISSION-VIP-004`
  Unknown) rests on the release test, which runs this build, not the original.
- The original editor's node writes (slot names, labels) are unknown
  (`DIV-3183`).
