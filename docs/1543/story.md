# Typed map editor commands over one command table

## Intent

The owner's direction for the map editor's MCP connection: extend MCP where
tools are missing. Before this story an agent could edit a map only as a
person does, by window keys, pointer gestures and typed text; story 1540
left typed tools out because they needed one editor command table that both
the window and MCP dispatch, and direct model writes would be a second edit
path.

After this story the editor has that table. Every window route of the edits
below calls the table's apply method, and MCP lists one typed tool per table
row, generated from the row. A typed tool takes a JSON object checked
against its input schema, applies the same host operation as the window
route, joins the same undo history, and answers what changed: the record it
names, the cells whose terrain, height or objects differ, and the records
added, removed or changed. Seven read tools answer what an agent needs
before an edit: one record and its fields, the records in a cell range, the
Add window's steps and choices, the brushes, the map check, the trigger
wizard's items and the map property fields.

## Authority

- Owner direction: "MCP расширяем если каких то тулов нет" (extend MCP where
  tools are missing), with typed edits through one table the window and MCP
  share.
- One builder per kind (`AGENTS.md`). The map editor edit's builder is the
  command table; the MCP tool's builder is the one tool table
  `editorTools` in `pkg/ui/mapeditormcp.go`; key dispatch stays
  `editorKeyRows`; undo history stays in `pkg/mapedit` behind the host.
- No ROM1 claim is cited or changed: the commands are editor authoring, and
  the window writes the bytes it wrote before. No DIV row is added.

## Census

Observation over the base `5fd74b88`: every host operation the pkg/ui map
editor files (`mapeditor*.go`, `editor*.go`, not tests) call on a document's
`Edits` or its optional seams, by source search. "Before" is what an MCP
agent had at the base: the 1540 tools `key`, `text`, `pointer`, `cell`,
`open`, `save`, `save_as` and one `action_<key>` per key row, so every edit
was reachable only by keys, pointer gestures and typed text.

### Edits a typed command covers

| Edit | Window route | Host seam and pkg/mapedit operation | Typed command |
| --- | --- | --- | --- |
| New map of a size | `new` (catalogue New question: size, single or multiplayer, starting height) | `MapInspector.Open("new/WxH…")`, `mapedit.NewMap` | `new_map` |
| Terrain stroke | `terrain` mode, list row, Size/Shape/Fill rows, left drag | `InspectionBrush.Paint`, `Editor.Paint` (terrain plane) | `paint` plane `terrain` |
| Height stroke (raise, lower, set, sample; step, level, smooth) | `height` mode, rows, drag | `Paint`, height plane | `paint` plane `height` |
| Map object stroke | `objects` mode, row, drag | `Paint`, object plane | `paint` plane `objects` |
| Passability stroke | `passability` mode, row, drag | `Paint`, passability plane | `paint` plane `passability` |
| Whole-map terrain or heights from a seed | `randomterrain` (Ctrl+E), `randomheights` (Ctrl+Shift+R), seed prompt | `InspectionWholeMap.RandomTerrain`/`RandomHeights`, `RandomizeTerrain`/`RandomizeHeights` | `randomize` |
| Add a unit, structure, sack, player, trigger, action, condition or enchantment | `unit`, `structure`, `sack`, `player` (players and scripts lists), `casts`, Tools menu New trigger; the Add window's steps | `InspectionNewRecords.AddRecord`, `Place`, `PlaceUnit`, `AddSpellFrom` and the group, script and trigger writers | `add_record` |
| Move a record | left drag on the map | `Edits.Move`, `Editor.Move` | `move_record` |
| Clone a unit, structure or sack | `clone` (C), click a cell | `Edits.Clone`, `CloneRecord` | `clone_record` |
| Delete a record, a trigger from the Triggers list | `delete` (Del), the list's delete | `Edits.Delete`, `Editor.Delete` | `delete_record` |
| One field of any record | `field` (F) prompt; the players window's relation buttons | `Edits.SetField`, `InspectionTextFields.SetTextField`; `SetField`, `SetTextField`, `Reorder` for an order field | `set_field` |
| Object popup Apply: fields and an optional move | popup Apply | `InspectionObjectSet.SetObject`, else `SetFields` and `Move` | `set_object` |
| Script record popup Apply, area trigger fields | popup Apply, area prompt | `InspectionFieldSet.SetFields`, `SetFields` | `set_fields` |
| Drop cell (map without drop zones) | `drop` (D), click | `InspectionScript.SetDropCell`, `SetDropCell` | `set_drop_cell` |
| Drop zones | `drop`, click an empty cell, drag a zone, Del | `InspectionDropZones.AddDropZone`/`MoveDropZone`/`DeleteDropZone`, `AddDropZone` and the script writers | `add_drop_zone`, `move_drop_zone`, `delete_drop_zone` |
| Trigger from the wizard's ready items | trigger wizard Apply | `InspectionTriggerWizard.WriteTrigger`, `SetScriptNode` and the trigger writers | `write_trigger` |
| Map metadata: name, description, angle, raw words, mode | `metadata` (M) prompt | `InspectionWorkspace.SetWorkspaceField("metadata")`, `SetEncodedName`, `SetDescription`, `SetAngle`, `SetWord0C`, `SetWord70`, `SetBitmask` | `set_map_field` scope `metadata` |
| Mission pages | `pages` (X) prompt | `SetWorkspaceField("pages")` | `set_map_field` scope `pages` |
| Undo, redo | `undo`, `redo` | `Edits.Undo`/`Redo`, `Editor.Undo`/`Redo` | `undo`, `redo` |

The mod package window's fields (`package`) go through the same table
method as the metadata prompt (`commandSetMapField`, scope `package`), but
`set_map_field` refuses that scope: the mod window's admission, folder and
publish steps are a confirmation sequence the agent drives with the
existing key and text tools.

### Edits left on their window route

Each stays as it was, reachable with `key`, `pointer` and `text`, and gets
no typed tool. Each host call sits inside window state the table has no
typed form of yet: a selection set, a copy buffer, the message editor, the
unit settings preview, or a goal list.

| Edit | Window route | Host call | File |
| --- | --- | --- | --- |
| Move, delete, copy and paste a selection | box select, drag, Del, Ctrl+C, Ctrl+V | `InspectionSelection.MoveRecords`/`DeleteRecords`/`CopyRecords`/`PasteRecords` | `mapeditorselection.go` |
| Copy or move an area with its terrain | area selection, Ctrl+C, drag | `InspectionAreaCopy.CopyArea`/`MoveArea` | `mapeditorselection.go` |
| Units into a group | `group`, `togroup` | `InspectionGroups.SetUnitsGroup` | `mapeditorgroups.go` |
| Shop stock and item recipe | `stock`, `recipe` | `InspectionRecords.AddStock`/`AddItemRecipe` | `mapeditorauthoring.go` |
| Message text and parts | message editor | `InspectionMessages.SetMessageParts`/`CreateMessageParts`/`PointMessageParts` | `mapeditormessage.go` |
| Box area for a trigger | `drawarea` | `InspectionAreaAuthor.AddBoxArea` | `mapeditorareaedit.go` |
| Goal target | `goals` report | `InspectionScript.SetGoalTarget` (`AddArm` when the goal takes an argument) | `mapeditorscript.go` |
| Forest density and thinning | `objects` forest rows, right drag | `Paint` with a forest | `mapeditorbrush.go` (the stroke goes through `paint`'s table method; the typed tool takes no forest) |
| Message record of the Add window | Add window kind Message | opens the message editor | `mapeditoradd.go` |
| Unit settings preview | unit settings window | `InspectionUnitSettings.UnitSettings` (a reader for the preview) | `mapeditorunitsettings.go` |

Out of scope by the census: there is no window route that resizes an open
map, so no command does; test play, Save and Save As keep their 1540 tools.

### Readers an agent lacked

At the base the `map` tool read the map's size, records and one cell, and
`state` the window. An agent could not read without driving the window:
one record's fields with their keys, choices and read-only flags; the
records in a cell range; the Add window's steps and choices for a kind;
the brushes' terrain classes and object codes; the map check; the trigger
wizard's items; the map property fields. Each is now a read tool over the
existing reader (`Edits.Fields`, `InspectionNewRecords.NewRecordStep`,
`InspectionBrush.BrushTerrains`/`BrushObjects`, `InspectionChecker.Report`,
the wizard's item table, `InspectionWorkspace.WorkspaceFields`).

## As built

Kinds and builders:

- Map editor edit: the command table, `editorCommandRows` in
  `pkg/ui/mapeditorcommands.go`. Each row is a name, help, typed arguments,
  a decoder from JSON and one apply method (`commandPaint`,
  `commandAddRecord`, `commandSetField`, `commandSetFields`,
  `commandSetObject`, `commandMove`, `commandClone`, `commandDelete`,
  `commandSetDropCell`, `commandAddDropZone`, `commandMoveDropZone`,
  `commandDeleteDropZone`, `commandWriteTrigger`, `commandSetMapField`,
  `commandRandomize`, `commandHistory`, `commandNewMap`). The apply method
  is the host call and the adoption of its document, moved out of the
  window route; the route keeps its own checks and follow-ups (status
  words, selection, the next prompt). Fifteen route files call it:
  `mapeditorbrush.go`, `mapeditoradd.go`, `mapeditorrecords.go`,
  `mapeditorplayers.go`, `mapeditormove.go`, `mapeditortrigger.go`,
  `mapeditorareaedit.go`, `mapeditorobject.go`, `mapeditordrop.go`,
  `mapeditorwizard.go`, `mapeditorworkspace.go`, `mapeditormod.go`,
  `mapeditorterrain.go`, `mapeditorkeys.go`, `mapeditormappanel.go`.
- Add record decoding: `newRecordOf` drives an Add run (`addRun`) through
  the window's own walk: `addOffer` gives each step's offer and skip,
  `addDefault` its starting value, the host checks each value as the
  window's Enter does, and `addRecord` assembles the record. The window's
  `enterAddStep` asks its steps through the same `addOffer` and `addDefault`.
  `internal/archtest` (`TestEditorAddRecordAsksTheAddWindowsWalk`) holds the
  command table to that walk. The pick path's `addPlacement`, which places
  the record on a clicked cell, keeps its own assembly; the table does not
  call it.
- Trigger decoding: `commandWizardRow` builds the wizard's own rows from the
  wizard's item table, and `rowsPlan` plans them as the wizard's Apply does.
- MCP tool: the one tool table `editorTools`; it appends one row per command
  row and per reader row (`editorReaders`, `pkg/ui/mapeditorcommandread.go`)
  with `InputSchema` from `commandSchema`. A typed row runs in `updateMCP`
  inside the window's Update, as every 1540 tool does.
- Change report: `changeReport` compares what the map stores before and
  after, read through the host seam `InspectionStore.Stored`
  (`mapInspectionEdits` decodes the document's bytes: each cell's tile word
  with its passability bit, height and object byte, and each editable
  record's bytes). It names the record the command names, up to 1024
  changed cells with the total and the count per plane, and up to 256
  changed records with the total; a record the host does not store by
  reference compares its list entry. A new map names its size. Every typed
  edit reports through it.
- Field values: `fieldValue` refuses a typed 0 in an inheriting field
  through the window's `fieldNumber` and answers `fieldNumberNote`, the
  field prompt's words; a number field given no value passes as the
  prompt's empty text and inherits.
- Undo and redo: `commandHistory` is the window's whole Undo and Redo (leave
  a drag or pick, one history step, `refreshMessagePage`); the Undo key's
  `keyHistory` and the typed rows call it.
- Guard: `internal/archtest/editorcommands_test.go`. No non-test
  `pkg/ui/mapeditor*.go` or `pkg/ui/editor*.go` file other than the table
  names the owned host operations (`Paint`, `AddRecord`, `SetObject`,
  `SetFields`, `SetTextField`, `WriteTrigger`, `SetDropCell`,
  `AddDropZone`, `MoveDropZone`, `DeleteDropZone`, `SetWorkspaceField`,
  `RandomTerrain`, `RandomHeights`, and `Move`, `Clone`, `Delete`,
  `SetField`, `Undo`, `Redo` on `Edits`), with self-tests that it refuses
  each form and accepts the table.
- Key dispatch (`editorKeyRows`), undo history (`pkg/mapedit` behind
  `mapInspectionEdits`), save routes and the 1540 tools: unchanged.

Refusals: a typed tool refuses with `isError` and changes nothing for an
unknown or null argument, a missing required one, a wrong type, a cell off
the map, an unknown record, kind, field, item or choice, a read-only field,
a value the host's Add step check refuses, and any call while a window
prompt, path question or popup field is open. A host refusal also shows on
the window's status line, as the window route's refusal does.

## Proof

Ordinary tests, synthetic maps and tables, no install:

- `TestMapEditorCommandsMatchTheWindowRoutes` (`pkg/game`), 17 subtests on
  the real host: move, add unit, add structure, add player, text field,
  number field, delete, clone, add, move and delete drop zone, the
  wizard's hero-area trigger, height stroke, passability stroke, random
  terrain, random heights, metadata name. Each drives the window route on
  one editor and the typed tool over the in-process MCP pipe on another
  opened from the same bytes; the bytes are equal, each undo step of the
  window equals the tool's `undo` step down to the starting bytes, and redo
  gives the edit back.
- `TestMapEditorNewMapCommandMatchesTheWindow`: `new_map` opens the map the
  window's New question makes, byte for byte.
- `TestMapEditorMCPBuildsAPlayableMapFromEmpty`: from an editor with no map,
  typed tools alone make a 32 x 32 map, paint terrain and height, add a
  player, two units and two structures of two sides, a drop zone and a
  hero-area victory trigger; the map check reports nothing; `save_as`
  writes a temporary file whose bytes are the document's and which reopens
  with the same bytes and records.
- `TestEditorCommandToolsComeFromTheTable` and
  `TestEditorCommandToolsRefuseBadArgumentsAndKeepTheMap` (`pkg/ui`): every
  command and reader row is one tool with its schema; thirteen bad calls
  refuse and leave the document; a call through an open path prompt
  refuses.
- `TestEditorEditsGoThroughTheCommandTable` (`internal/archtest`).
- The existing editor tests, the editor golden and modes tests, including
  `MAPEDIT_MODES_FULL=1 MAPEDITOR_MODES_FULL=1` for
  `TestMapEditorModesHoldEveryKeyAndGesture`, pass unchanged.

The terrain stroke is not in the window comparison: the window's default
fill draws a fresh seed per stroke, so two strokes differ by design. Its
route calls the same `commandPaint` the height and passability cases prove.

Release witnesses for the next release, EN and RU: the editor frame,
interaction, move/undo/save/reopen, workspace, script-refusal, Add window
and new-map release tests named in this story's return.

## Open debt

- The routes in "Edits left on their window route" have no typed command.
- `paint` takes no forest stroke.
- `set_map_field` refuses the mod package scope.
- The typed `add_record` of kind Message is refused; messages are written in
  the message editor.
