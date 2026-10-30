# The map editor's remaining edits go through the command table and reach MCP

## Intent

Story 1543 made one editor command table, `editorCommandRows` in `pkg/ui/mapeditorcommands.go`,
that the window and the MCP tool table dispatch alike, and left ten edit kinds on their window
routes. An agent reached those only by keys, pointer and text, with no typed refusal and no
change report. This story moves the ten into the table: each is a typed row, the window route
calls the row's apply method with the state it collected, and the MCP tool is generated from the
row. `set_map_field` gains the mod package scope.

## Authority

- Owner, verbatim: «MCP расширяем если каких то тулов нет».
- Architecture audit at engine `c40f21fe`, row 37.
- `docs/1543/story.md`, "Edits left on their window route", and its debt on the package scope.
- The map editor is this project's own program: no ROM1 behaviour, no DIV row.

## As built

### Rows

The new rows sit in `editorCommandRows`. Their decoders and apply methods are in
`pkg/ui/mapeditorcommandedits.go`.

| Row | Host call | Window route that calls the row |
| --- | --- | --- |
| `move_records` | `MoveRecords` | a drag of a selection (`moveSelection`) |
| `delete_records` | `DeleteRecords` | Del on a selection (`deleteRecords`) |
| `copy_records` | `CopyRecords` or `CopyArea`, then `PasteRecords` | Ctrl+C (`copyOf`) and the Ctrl+V click (`pasteAt`, `commandPaste`) |
| `move_area` | `MoveArea` | a drag inside the selection box (`moveArea`) |
| `set_units_group` | `SetUnitsGroup` | the group keys and the To group question (`groupTo`) |
| `add_stock` | `AddStock` | K (`stock`) |
| `add_item_recipe` | `AddItemRecipe` | I and its item prompt (`recipe`) |
| `write_message` | `SetMessageParts`, `CreateMessageParts`, `PointMessageParts` | the message editor's Apply (`applyMessage`, `createMessage`) |
| `add_box_area` | `AddBoxArea` | the box area tool's Create (`areaCreate`) |
| `set_goal_target` | `SetGoalTarget` | a pick from the Goals report (`goalAt`) |

- `paint` takes `forest` (`kinds`, `share`, `thin`). That is the object brush's Forest stroke:
  the same `BrushStroke` with its `Forest` set, drawn from the window's stroke seed.
- `add_record` takes kind `Message` with `parts`. Its value is `addValue`: a Message becomes a
  `write_message` with owner `Message` and a new page. Every other kind goes through the
  unchanged `addRecord`/`newRecordOf`.
- The Add window's Message step still opens the message editor. Its Apply is `write_message`.
- `set_map_field` takes scope `package`. It refuses `folder`, `folder-open`, `output` and
  `source`, which open a folder or switch the mission.
- New read tools:
  - `goals`: the Goals report.
  - `messages`: the pages, or one page's parts and speakers.
  - `unit_settings`: the unit settings window's preview through `UnitSettings`. It writes
    nothing.

### Change report

- An edit of several records answers the host's `SelectionEdit` as `selected`, `note` and
  `refusals`.
- A message edit answers `page`.

### Argument checks

Each check refuses with `isError` before the host is called, so a refused call writes nothing.
A selection move has no check against the map edge: the host refuses each record that would
leave the map or enter a closed cell (`placeGroup`), moves the rest as one step and returns the
refusals, so the window's drag and `move_records` give the same map, the same Undo step and the
same `refusals`. A move that takes every record past the edge changes nothing and reports
`changed` false.

- a record that does not exist, cannot be selected, or is named twice;
- an area whose corners are out of order or off the map;
- both or neither of `records` and `area`;
- a moved box off the map;
- records outside the moved area;
- a group below -1;
- a recipe outside a Stock or Sack;
- a message call with neither page nor action;
- a part source the page does not hold;
- a trigger out of range, or one without a free condition pair;
- a goal that takes no target, or a kind the goal does not take;
- a forest kind the Forest grid does not offer, a kind named twice, or a share outside 1..100;
- `value` given with a forest, or a forest on a plane other than objects.

### Message part numbering

A message card the page holds keeps its source, number and tag. On a page with no parts, the
first new card takes the page's first card. Every other new card takes the next number below
the smallest number on the page, as the editor's "+" numbers a card.

### Window order

Two window routes now adopt the host's document before the step that came after the host call:

- `areaCreate` adopts before it closes its prompt.
- The message editor's point-and-rewrite case adopts before it sets `messageFrom`.

The frames, the golden test and the modes walk are unchanged.

### Guard

`internal/archtest/editorcommands_test.go` lists the sixteen new host operations. They are
`MoveRecords`, `DeleteRecords`, `CopyRecords`, `PasteRecords`, `CopyArea`, `MoveArea`,
`SetUnitsGroup`, `AddStock`, `AddItemRecipe`, `SetMessageParts`, `CreateMessageParts`,
`PointMessageParts`, `SetMessage`, `CreateMessage`, `AddBoxArea` and `SetGoalTarget`. The guard
allows calls to them only in the table's two files, `mapeditorcommands.go` and
`mapeditorcommandedits.go`. Its add-walk check covers both files.

### Builders

| Kind | Builder |
| --- | --- |
| Each editor edit | `editorCommandRows` |
| MCP tool | the tool table in `pkg/ui/mapeditormcp.go`, from the rows (unchanged) |
| Add window record | `addRecord` (unchanged) |
| Key dispatch | `editorKeyRows` (unchanged) |
| Undo | `pkg/mapedit` (unchanged) |

## Proof

- `TestMapEditorCommandEditsMatchTheWindowRoutes` (`pkg/game`) has 18 cases. In each, the window
  route and the MCP tool give the same map bytes and the same workspace (message pages and mod
  package). Each undo step back to the start matches, and so does each redo. The cases:
  - a selection move, a selection move with three of four records past the map edge, delete and
    copy;
  - an area copy and an area move;
  - a group;
  - a stock and a recipe;
  - a new message page and a page rewrite;
  - the Add window's Message;
  - a box area on a new trigger and on an existing trigger;
  - a goal target;
  - a forest stroke and a thinning stroke;
  - a mod package field.
- `TestMapEditorCommandEditsRefuseBadArguments` (`pkg/game`) sends 56 bad calls over real hosts.
  Every call is refused with `isError`. Neither the map, the workspace nor the history changes.
- `TestMapEditorMoveRecordsPastTheEdgeReportsRefusals` (`pkg/game`) checks the partial move's
  note "Moved 1 of 4" and its three `refusals`, and that a move of one record off the map
  changes nothing and reports `changed` false.
- `TestMapEditorMCPExampleSessionSavesAndReopens` (`pkg/game`) runs the documented session over
  the in-process MCP client:
  - a selection move, a group, a stock, a message and a goal target, then `save_as`;
  - the saved file is the map in its save form, and it reopens to those bytes;
  - the reopened page holds both parts;
  - five undos give the opened map back.
- `TestEditorUnitSettingsToolPreviewsWithoutWriting` (`pkg/ui`) checks the preview and its two
  refusals, and that nothing is written.
- `TestEditorCommandToolsRefuseBadArgumentsAndKeepTheMap` (`pkg/ui`) adds the new tools and the
  package `folder` refusal.
- These pass unchanged: `TestEditorCommandToolsComeFromTheTable`,
  `TestMapEditorCommandsMatchTheWindowRoutes`, the editor golden and modes tests, and
  `MAPEDITOR_MODES_FULL=1 TestMapEditorModesHoldEveryKeyAndGesture`.

## Open debt

- A paste into another open map keeps its window route. The clip comes from the other document,
  and a tool names only the open map.
- A message page written from inside the trigger wizard, with the wizard's unapplied trigger,
  goes through `write_message`'s apply method as one edit. No tool argument names an unapplied
  wizard plan. `write_trigger` and then `write_message` give the same trigger and page in two
  steps.
- The mod package window's folder, output and mission choice, and its confirmations, stay in the
  window.
- No window route calls an arm-adding method for a goal. `SetGoalTarget` adds the arm itself,
  through `mapedit.Editor.AddArm`, on a goal's add row (the `add` route of
  `TestArmRoutesByteHashes`).
- Saving writes `alm.SaveForm`. On a map without player records, units are written as player
  1's, so the saved bytes can differ from the open document. That is the save's documented form,
  not a change of this story.
