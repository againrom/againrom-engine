# An agent's map edit tells it the IDs, groups and visible area it needs

## Intent

The owner had an agent build a 64x64 map over MCP: a goblin village, three roads, a river and a
victory trigger on the goblin chief. Three faults followed from what the editor did not tell or
refuse:

- the victory trigger named a unit value no placement held, because `add_record` answered an
  index and `write_trigger` took any number;
- every new unit was in group 0, so the goblins, turtles and centipedes formed one script group;
- a Teleport Pad and turtles stood in the 8-cell border ring, which the game does not show and
  where it removes a placed unit.

This story makes an edit's result carry the script ID, the group and the playable area an agent
needs, refuses the placements the game would drop, and attaches the map check's findings on the
touched records to each edit.

## Authority

- Owner findings, verbatim:
  1. «ты указал условие победы не на главного гоблина»
  2. «по умолчанию все враги у тебя в одной группе были»
  3. «карта ограничена рамкой 8 клеток, но ты сперва телепорт поставил за границы карты, как и
     черепах»
- Owner decision 1, the default group of a new unit: (a), the lowest group number no unit of the
  map carries, one per unit added without a group. The owner reopened the decision, then kept
  (a): «если там создавались отдельные, наверное так и надо». An explicit group value always
  wins.
- Owner decision 2, the border ring: (a), a unit, a structure or a drop zone is refused on a ring
  cell; terrain, heights and scenery are accepted there.
- `TERR-PLACE-208`: mission 100 places 2 of its 11 turtles on ring cells, and the loader logs and
  deletes an actor its seat call refuses on such a cell. `ALM-TERR-016`: the runtime grid gives
  the 8-cell map border block value `0x1f`.
- The map editor is this project's own program. This story adds no DIV row.

## As built

### Script IDs

- `InspectionRecord.ScriptID` (`pkg/ui/mapeditor.go`) is the number a script parameter names a
  unit or structure by: the unit's `UnitID`, the structure's `Field12`
  (`pkg/game/mapinspection.go`). It is 0 for every other record.
- The change report (`changeReport`, `pkg/ui/mapeditorcommands.go`) answers `script_id` when the
  record the call names is a unit or structure that still stands.
- `record` gives the record with `ScriptID` and the read-only result field `script_id`.
  That field is projected from the inspected record in `readRecord`. The window's field
  controls stay unchanged; `set_field` refuses the ID.
- `trigger_items` gives each unit or group parameter `choices`, from `wizardTargets`
  (`pkg/ui/mapeditorwizard.go`), which reads the `scriptTargets` list the host builds: the placed
  units by script ID and the hero band 10001..11000, or the groups the map's units carry.
  The window's choices still list heroes 1 to 8 and keep a loaded value. A unit parameter
  also names `hero_band`, `10001..11000`. No wizard item takes a structure parameter.
- `write_trigger` refuses, with `isError`, a unit or group value that `wizardTargetAllowed`
  rejects: one `wizardTargets` does not list.
- The map check's `script-reference` rule (`pkg/game/mapcheck.go`) now skips only the hero band
  10001..11000. A unit value above 11000 that the compiled script leaves unresolved is an error.

### Group on add

- The Unit Add steps (`addKinds`, `pkg/ui/mapeditoradd.go`) are class, tier, owner, `group` and
  cell. The window's Add route and `add_record` walk the same steps; `addRecord` stays the one
  route.
- The step's default is `newUnitGroup` (`pkg/game/mapinspectionadd.go`), the one place of
  decision 1. It takes the map and the new record's values before the step, and today answers
  `freeGroup`: the lowest group number from 1 that no unit of the map carries. Two units added
  without a group land in two groups.

### Playable area

- `mapload.Playable` (`pkg/mapload/borderplacement.go`) is the rectangle inside the ring;
  `InBorderRing` is its complement, so the ring has one definition. A map of 16 cells or fewer
  on a side has no playable cell.
- `InspectionDocument.Playable` carries it; `PlayableText` writes it as `x0,y0..x1,y1`, corners
  included, or `none`. `new_map` answers `playable`; `map` answers it beside the size.
- `insideMap` (`pkg/game/mapinspectionedit.go`) takes the kind it judges. With `cellError`, its
  judgement on a decoded map, it refuses a cell outside the map and, for a kind `ringRefuses`
  names (`Unit`, `Structure`, the drop cell), a ring cell, naming the playable area. `ringRefuses`
  is the one place of decision 2.
- Every placing host call passes its kind: the Add window's cell check (`checkNewValue`), unit
  and structure adds, `Move`, `Clone`, `SetDropCell` and the drop zone tools, and `placeGroup`
  (the selection move, delete-and-place and paste). A sack and a cell spell pass their own kinds
  and take ring cells, as before. A move onto the record's own cell is a no-op before the check,
  so a click on a unit the map already holds in the ring does not refuse.
- On a map with no playable cell the Add window refuses a unit or structure at its first step,
  saying the ring covers every cell.
- A structure is judged by its cell, the footprint's top-left. A new map check rule,
  `structure-border` (warning), names a structure whose footprint meets the ring.

### Warnings and smaller results

- The change report's `warnings` are `checker().Report()` filtered to the records the edit
  changed or names (`editWarnings`), each row built by `findingRow`, the row `check` uses. There
  is no second rule set.
- `paint` and `randomize` list `cells` only with `verbose: true`; every report gives
  `cells_total`, `cells_box`, `planes` and `records`.
- `new_map` and `open` take `discard: true`. `discardLeave` (`pkg/ui/mapeditorleave.go`)
  answers an open unsaved-changes question that holds a map open as its Open button does.
- A denied mod output path now keeps the filesystem error from `os.Lstat` or `pathWithin`
  instead of reporting a successful containment check as an outside path. The existing
  physical path and install fences still refuse the write.

### Builders

| Kind | Builder |
| --- | --- |
| Map editor edit | `editorCommandRows` |
| MCP tool and result | the tool table in `pkg/ui/mapeditormcp.go`; the change report `changeReport` in `pkg/ui/mapeditorcommands.go` |
| Add window record | `addRecord` |
| New unit record | `NewUnitRecord` (unchanged); the group arrives as the `group` step's value through `SetFields` |
| Script parameter choices | `scriptTargets`, read through `wizardTargets` |
| Map check rule | `mapCheck` rows in `mapRules`; `warnings` filter its findings |
| Cell bounds test | `insideMap` with `cellError`, which gained the ring; the ring itself is `mapload.Playable` |
| Mod map write and output boundary | `writeModMap`, `checkModMapDirectory` and the shared `pathWithin`; error propagation only |

## Proof

- `TestMapEditorMCPVictoryTriggerNamesTheUnitAddRecordReports` (`pkg/game`) runs the village
  session of `docs/MAPEDIT-MCP.md` over the in-process MCP client:
  - `new_map` and `map` answer `8,8..23,23`;
  - a structure's and two units' `add_record` answer `script_id`, the two units' differ, and
    `record` gives the second one's as a read-only field; writing that field is refused;
  - the two units added without a group are in groups 1 and 2, and a third added with group 1 is
    in group 1;
  - `trigger_items` lists the second unit and hero 11000 among the `c.unitdead` choices,
    and group 2 among the group choices;
  - `write_trigger` with unit 20001 or missing group 99 is refused and leaves the bytes;
  - with the listed ID, a condition of the script names the unit's cell 14,14, and `check` finds
    no `script-reference`; an independent `mapload.CompileScript` call binds the alive
    check to entity 1, the second placement;
  - after a planted value 12345 and `open` with `discard: true`, `check` names 12345 under
    `script-reference` as an error; `new_map` with `discard: true` also replaces a dirty map.
- `TestMapEditorMCPRefusesTheBorderRingForPlacements` (`pkg/game`): on 64x64, `playable` is
  `8,8..55,55`; a unit, a structure and a drop zone on 4,4 are refused with that text and the
  bytes kept; `move_record` into the ring is refused; a height stroke on 4,4 is taken and
  answers no `cells` but a `cells_box`; a `verbose` passability stroke lists every cell.
  A null verbose flag is refused; randomized heights omit `cells` by default.
- `TestMapEditorMCPWarnsOfAStructureInTheBorderRing` (`pkg/game`): `check` names the planted
  structure on 3,3 alone under `structure-border`; an edit of it carries that finding in
  `warnings`, and an edit of a structure on 20,20 carries only its own findings.
- `TestMapRulesFindTheirDefectAndNothingInTheCleanMap` gains a footprint meeting the ring and a
  unit value above the hero band; `TestMapRuleTableIsDataWithAKindPerRow` lists
  `structure-border`. `TestPlayableIsTheCellsInsideTheRing` (`pkg/mapload`) pins the rectangle to
  `InBorderRing` cell by cell on five sizes.
- Tests whose fixtures placed records in the ring now place them inside it. The 1118 fixture's
  units move from 3,4 and 7,8 to 11,12 and 15,16, with each drag and byte check shifted by 8;
  the authoring and records fixtures grow from 16x16 to 32x32 and keep their records; drop
  zones on 5,5 move to 12,12; the Paste refusals name the ring. The selection move past the edge
  moves by -7, so three of four records land in the ring.
- `TestMapEditorCommandReportNamesWhatTheMapHolds` checks that a paint lists its cells only with
  `verbose`.
- The editor modes golden changes in the Unit Add rows: one more Add step for the group. Of its
  112 rows, 58 differ only in the `unit` key's column and 8 are unit-add paths. No key, gesture or
  menu changed. Final mode and frame results are recorded in the lane's result.

At the next release, run these witnesses on EN and RU: `TestReleaseMapEditorOwnerFrames`,
`TestReleaseEditorInteractionFrames`, `TestReleaseMapEditorTriggerFrames`,
`TestReleaseMapEditorNewUnitTakesTheChosenTier`,
`TestReleaseMapEditorUnitAuthoringPlaysAsModMission`,
`TestReleaseMapEditorStructuresAndSacksPlayAsModMission`,
`TestReleaseMapEditorMovesTheDropCellAndRetargetsTheWinArm`,
`TestReleaseMapEditorWholeMapCommandsFollowTheSeed` and
`TestReleaseMapEditorGroupPopupFollowsUndoAndRedo`. No release test runs in this lane.

## Open debt

- The original map editor's own default group for a new unit is unread. The shipped maps hold
  group 0 on 5 of their 8094 placed units (`AI-GROUP-009`), so their authors gave nearly every
  unit a group; decision 1 rests on the owner's direction, not on that editor.
- A structure is refused by its cell. A footprint that starts on a playable cell and reaches into
  the ring is accepted, and the edit's `warnings` carry the `structure-border` finding.
- A sack added on a ring cell is accepted, while a selection move or paste refuses a sack that
  would land there, by the ground arms check that predates this story.
- No ready trigger-wizard item takes a structure parameter. Structure IDs remain available in
  `record` and `add_record`; structure-reference checks stay in the existing map checker.
- Installed EN and RU rendering of the refusals and the new Add step is not shown by these
  synthetic tests; the release editor witnesses must run.
