# An agent reads heights, knows the height brush and gets short results

## Intent

The owner had an agent build a map over MCP and passed on its report. Its lead: the agent read
the engine code and parsed the saved `.alm` height section itself, because MCP could change
heights but not read them. This story adds a height read, states what each height brush mode
does, gives every tool a short result with the whole window state behind `verbose`, and fixes the
report's defects that held at base `f37d72903`.

## Authority

- The owner's direction in the lane brief: the agent's report is a hypothesis list, each item
  checked against the base before a change.
- Owner decision 1, the size of results: (a), every tool answers its own result. The place is
  `everyToolOwnResult` in `pkg/ui/mapeditormcp.go`.
- Owner decision 2, the one-call dialogue: (a), a dialogue action row of `write_trigger` may
  carry `parts`, written through the `write_message` builder. The place is
  `writeTriggerDialogue` in `pkg/ui/mapeditorcommands.go`.
- `TERR-245`: the row step bound of 31 a height stroke holds.
- The map editor is this project's own program. This story adds no DIV row.

## As built

### Heights

- `heights` reads the stored height of a cell range, at most 4096 cells, from the same stored
  map the change report reads, with `min`, `max` and `label_zero` (64, the View menu's 0).
- A height `paint` answers `height_change` (covered cells, changed cells, box, smallest and
  largest change); with `verbose`, `heights` lists `[x,y,before,after]` per covered cell. The
  covered cells are the window's own `BrushCells`, so the report and the marker agree.
- `Brush.Footprint` covers a blending set brush as `heights` writes it: the effective circle of
  radius at least 1 around the first centre. Before, it walked every centre at the given radius,
  so the marker and the covered count named cells the stroke never writes.

### Height brush modes

`docs/MAPEDIT-MCP.md` (Height brush) and the `paint` description state each mode from the code
(`pkg/game` `paintStroke` over `pkg/mapedit` `Brush`): `step` adds the change once per covered
cell; `level` writes the first centre's height before the stroke plus the change, clamped to
0..127, to every covered cell, so 64 + 30 = 94 on a new map; `smooth` adds the change by the
cosine weight in sixty-fourths around each centre, so 12 gives the next cell 12 × 63 / 64 = 11
at radius 4; `set` without `flat` uses the first centre alone; `set` with `flat` writes every
covered cell. Typed centres are not joined; the window's drag joins its pointer samples before
the stroke. The doc was corrected; the route is unchanged. The row step bound is stated as the
route applies it: `boundRows` (step, level, smooth, set with flat) lowers a covered cell more than
31 above the cell in the row above it and, above the first centre's row, first raises a covered
cell more than 31 below the cell under it; `boundSetRows` (set without flat) bounds only covered
cells with a column distance below their row distance, towards the first centre's row. Other
steps, such as a covered cell against an uncovered neighbour, can exceed 31.

### Short results

- Each `editorTool` row may carry `brief`, the state keys it answers without `verbose`; the
  completed call picks them through `briefState`. Window tools answer 15 keys, `save` and
  `save_as` 11, among them `saved` (the file the call wrote) and `report_asks`.
- `map` answers `record_counts` instead of `records` and `script`; `record` drops a field's
  choices past 32 and names the field in `choices_omitted`; `trigger_items` points a unit or
  group parameter at one top-level `units` or `groups` list without the hero band's 1000 units.
  `verbose: true` restores each whole answer; `state` is always whole.

### Defects

- A save that wrote its file answers no error: `saveWith` records the written file, and `mcpSave`
  returns no error once it is recorded. A save the check would refuse asks whether to save with
  errors; that question is the result, not an error, and `key enter` answers it.
- `save` on a map with no file of its own closes the Save prompt the route opened and names
  `save_as`. A `save` that finds a prompt already open (the typed Save prompt a person opened, or
  any other) is refused through `mcpBusy`, as `save_as` and `open` are: the prompt stays open and
  keeps its typed text. A refused `save_as` or `open` closes its path prompt, so the next call is
  not refused by it.
- `save_as` onto the source map says that Save writes the map's own file. A missing folder is
  named: `mapOutputDirectory`, used by both map writers, refuses with "the folder … does not
  exist; a save creates no folder".
- `set_drop_cell` on a map with drop zones names all three drop zone tools and that each zone is
  a drop action. A drop zone edit answers `drop_zone` with the zone, its cell and the zone count.
- A refusal by an open window state names it: the system file dialog, the prompt by its title,
  or a popup field (`mcpBusy`, used by the typed edits and the path tools).

### Units and the dialogue

- `InspectionRecord.Definition` names the Humans row a unit's npc flag or definition id reaches,
  or the id no row answers; a placement whose class alone picks its row has none. `record`
  shows it and `scriptTargets` adds it to the unit choice ("Unit 3: Fighter; definition: Humans
  row 4, guard"), in EN and RU words.
- A `write_trigger` action row whose item shows a message page may carry `parts` and leave the
  message parameter out; the call builds the `write_message` owner for the new trigger, so the
  trigger and the page are one edit and one Undo step.

### Builders

| Kind | Builder |
| --- | --- |
| Map editor edit | `editorCommandRows` |
| MCP tool and result | the tool table in `pkg/ui/mapeditormcp.go` (a short result is the row's `brief`); the change report `changeReport` in `pkg/ui/mapeditorcommands.go` |
| Height brush | the window's height stroke, `paintStroke` over `mapedit.Brush`; no second height writer |
| Message page | `commandWriteMessage`, the builder behind `write_message` |
| Script parameter choices | `scriptTargets` |
| Map read | the existing readers in `pkg/ui/mapeditorcommandread.go`; `heights` joins them |
| Map write folder check | `mapOutputDirectory`, used by `writeEditedMap` and `writeEditedMapPages` |

## Proof

Tests in `pkg/game/mapeditmcpbrief_test.go` run over the in-process MCP client on synthetic
maps:

- `TestMapEditorMCPHeightsReadTheStoredPlane`: after a stroke of each of the five modes,
  `heights` equals the stored altitude bytes cell by cell, with their least and greatest; a range
  past the map's edge and a reversed range are refused, and a 65 × 64 range is refused naming
  4096.
- `TestMapEditorMCPPaintReportsHeightsBeforeAndAfter`: a verbose `smooth` stroke lists its 49
  cells with the heights `heights` reads before and after, +12 at the centre and +11 next to it;
  a step stroke without verbose lists none and answers `height_change` 5 cells, 5 changed,
  change 3..3, box 4,4..6,6.
- `TestMapEditorMCPStrokeOfThreeCentres`: three centres 9 cells apart at radius 4 cover 147
  cells and each gains the step once, no cell between the circles; `level` 30 from two centres
  on a new map writes 94 on every covered cell of their row.
- `TestMapEditorMCPSaveLeavesAnOpenPrompt`: with the typed Save prompt open outside any call and a
  path typed, `save` answers a busy error naming the prompt, `prompt_open` stays true and the text
  is kept; on a new map with no prompt open, `save` still names `save_as`.
- `TestMapEditorMCPShortResults`: `save` on a new map is refused naming `save_as` and leaves no
  prompt open; `save_as` and `save` answer `saved`, the path, `dirty` false and at most 1200
  characters; `key` answers less than a third of its verbose answer, which `state` matches;
  `map` answers `record_counts` and, with `verbose`, `records`; `trigger_items` answers less
  than a tenth of its verbose answer and keeps `hero_band`.
- `TestBlendingSetFootprintIsTheFirstCentresCircle` (`pkg/mapedit`): a blending set stroke of
  seven centres changes no cell outside its footprint, and the footprint is the first centre's
  circle.
- `TestMapEditorMCPSaveAsAnswers`: on the fixture map, `save_as` beside the source writes the
  file and answers it as `saved` with no error; `save_as` onto the map's own file is refused
  naming Save; a missing folder is refused naming the folder. On a map with a check error,
  `save_as` writes nothing and answers `report_asks` with no error; `key enter` writes the file
  and answers it as `saved`.
- `TestMapEditorMCPCallsSentTogether`: `write_trigger`, `write_message` and `paint` written to
  the stdio input in one batch answer in order with no refusal, and `write_message` writes a
  page.
- `TestMapEditorMCPDialogueInOneCall`: a `write_trigger` with two `parts` adds trigger 0 and a
  page holding both parts; one Undo removes the trigger. `parts` on an item without a message
  parameter, or beside that parameter, is refused.
- `TestMapEditorMCPMessageListHoldsEveryPage`: on a map with 14 pages on disk, `messages` lists
  14 and the dialogue action's page field offers all 14.
- `TestMapEditorMCPUnitChoiceNamesItsDefinition`: a unit whose definition id reaches Humans row
  1, one whose id 999 reaches none and one without an id are named so, or not at all, in the
  `trigger_items` unit choices; `record` gives the first one's `Definition`.

`TestMapEditorMCPVictoryTriggerNamesTheUnitAddRecordReports` reads per-parameter choices with
`verbose: true`. The editor golden and modes tests in `pkg/ui` pass unchanged; no key, gesture
or menu changed.

At the next release, run these witnesses on EN and RU: `TestReleaseMapEditorBrushFrames`,
`TestReleaseMapEditorSetHeightEdit`, `TestReleaseMapEditorStrokeStaysNearTheBrush`,
`TestReleaseMapEditorPaintedHeightsReachTheHeightPlane`, `TestReleaseMapEditorStrokesLeaveNoSeam`,
`TestReleaseMapEditorParameterFrames`, `TestReleaseMapEditorTriggerFrames`,
`TestReleaseMapEditorNamedRecordChoicesUseExistingPrompts`, `TestReleaseMapEditorOwnerFrames`,
`TestReleaseEditorInteractionFrames` and
`TestReleaseMapEditorInstalledMapStillWritesItsInstalledBytes`. No release test runs in this lane.

## Report items that did not hold

- "A stroke of several centres the doc calls interpolated": the doc at base does not call it
  interpolated (a search of `docs/MAPEDIT-MCP.md` for "interpol" finds nothing). The 147 cells are
  the route as built; the doc now says centres are not joined.
- "`set_drop_cell` refused with «map keeps drop zones»" as a dead end: the base refusal already
  named `add_drop_zone` and `move_drop_zone`. The text now also names the delete tool and why.
- "Concurrent calls are not queued": the base queue already applies calls one at a time in
  arrival order. The refusal «finish or cancel the current window prompt» came from a prompt an
  earlier call left open, such as the Save prompt `save` opens on a map with no file; that
  prompt is now closed and the refusal names what is open.
- "`save_as` into the source map's folder answered «cannot replace the source map»": not
  reproduced on a fixture or a new map; a save beside the source answers no error. The mechanism
  that gives this shape, a save-with-errors question answered as an error and then answered by
  a second `save_as`, is fixed.
- "The message list showed pages 1 to 6 of the 14": at base `record`, `messages` and the object
  fields list every page of a 14-page map.
- "`key escape` with nothing to cancel opens the unsaved-changes question": it does, as the
  window's Esc does; the doc states it and the route is unchanged.

## Open debt

- No tool runs a trigger without playing the map; a trigger dry run is left for a later story.
- `frame` relief shading was not examined; `heights` answers what the agent read the file for.
- Installed EN and RU rendering of the new unit choice text and the blending set marker is not
  shown by these synthetic tests; the release witnesses above must run.
