# The map editor reports what is wrong before it saves

## Intent

In the first game's map editor, Save As and Save into a mod first check the
map and show a report. Each finding names a record (kind, index, cell) and the
rule it breaks, and the window selects the record a finding names and scrolls
the map to it. An error refuses the save and keeps the report on screen; a
warning is listed and the save goes ahead. The report is also available
without saving (key R) and from the command line (`mapedit -check`). The story
started at public main `169a2eb2` (pin k275); the candidate merged main
`b006300e`.

## Authority

Owner direction (map editor, priority) and owner confirmation of the default
rule table below: five groups of errors refuse the save, five warnings are
listed and the save goes ahead. Reference meanings: `ALM-OWN-039`,
`ALM-TRIG-046`, `ALM-TRIG-047`, `ALM-SACK-065`, `ALM-ORD-068`, `ALM-CLS-036`,
`MISSION-DROP-002`, `TERR-PASS-049`, `TERR-PASS-050`. Rows: `DIV-3175` (the
report and its rule table), `DIV-3176` (owner slot), `DIV-3177` (script
reference).

## As built

### One rule table, one checker

`mapRules` (`pkg/game/mapcheck.go`) is the rule table: id, kind (`Error`),
authority, meaning and the function that finds the rule's records.
`MapInspector.CheckMap` reads the table and nothing else; `MapInspector.rules`
replaces it (a rule moves between kinds, or leaves, by editing its row).
`mapInspectionEdits.Report` answers `CheckMap` for the model's current bytes
and keeps the last answer while the bytes stay the same. The window (R), Save
As, Save into a mod and `mapedit -check` all call `Report`.

| rule | kind | what it finds |
|---|---|---|
| `terrain-planes` | error | `mapload.OriginalTerrainPlanes`, with the install's `map.reg` costs, refuses the map: a side outside 8..256 or a terrain pair of strip groups 13 to 15 |
| `mission-build` | error | `StartMissionFrom` refuses the map (no party); skipped when `terrain-planes` already named the refusal |
| `record-outside` | error | a unit, structure anchor, structure footprint or ground sack outside the map |
| `script-reference` | error | the script compile's unresolved unit or structure references (`ScriptReport.Unresolved`, from `campaignScriptRefs`), and a Target_Group value no unit's group id holds |
| `owner-slot` | error | a unit or structure owner outside the roster (`ALM-OWN-039`) |
| `item-code` | error | a sack or stock element whose item code the installed name table lacks |
| `structure-kind` | error | a structure whose kind (low byte) the installed `structures.reg` lacks |
| `unit-terrain` | warning | a unit on a cell of `mapload.Passability` that its movement domain (`Domain.BlockedBy`, from the world `FromALMWith` builds) cannot enter |
| `structure-overlap` | warning | a structure whose footprint takes cells an earlier one took (`mapload.StructureCensus` refused cells, located by the census of the map cut after each structure) |
| `sack-duplicate` | warning | two ground sacks on one cell (`ALM-SACK-065`) |
| `drop-cell` | warning | `mapload.DropCells` outside the map or on a cell blocked for ground movers |
| `stock-link` | warning | a unit's 1-based +0x18 link names no type-8 record, or one whose owner word is not the unit's id (`ALM-ORD-068`) |
| `mod-message-text` | warning | a message number the script sends has no `eventNN.txt` in the texts folder of the mod's mission entry (and, for a replacing mission, none in the install); only when the editor saves into a mod (`DIV-3220`) |

The mission build's checks are called, not copied: `OriginalTerrainPlanes`,
`StartMissionFrom`, `CompileScript` (through `tableFiles(...).compileScript`
and `campaignScriptRefs`), `DropCells`, `Passability`, `StructureCensus`,
`Structures` and `Resolve`.

Choices the claims leave open:

- A Target_Unit value from 10001 names the party or an external static unit
  (the compile reports it unresolved when no party is bound); it is not
  checked.
- `stock-link` reads the link as 1-based with 0 meaning none, and the owner
  word as the unit id (`ALM-ORD-068`: 43 linked units, 43 owned records, zero
  disagreements). An orphan stock (a type-8 record no unit links) is not a
  finding.
- A structure footprint is the rectangle `mapload.Structures` gives, at least
  one cell a side, as the editor's own edit check uses.
- A unit standing in the 8-cell border ring is a `unit-terrain` warning: the
  border blocks ground and air movers.

### The window

R opens the report (R again or Escape closes it); the help line begins with
`R: report`. The list rows are `E|W rule Kind #index`; the selected finding's
full line is shown below the list. Up and Down, a click or Enter select a
finding: its record is selected and the map scrolls to it; a finding that names
a cell and a record without one (a drop cell's action node) scrolls to the
cell; a finding that names the whole map selects nothing. Flagged records are
marked on the map. Save As or Save into a mod with an error returns a
`*ui.ReportRefusal`, writes nothing, closes the path prompt and opens the
report with the note `Save refused: ...`; with only warnings it saves and
opens the report with `Saved with N warnings`. An edit closes the report.

### The command line

`mapedit -map <key> -check` prints the record lines as before, then one line
per finding (`ui.InspectionFinding.Line`, the line the window shows), then
`report: N errors, M warnings`; it exits 1 when N is above 0.

## Installed maps

`TestReleaseEveryInstalledMapReportsNoError` opens every first-game map and
counts the report.

| root | maps | errors | warnings |
|---|---:|---:|---|
| EN | 38 | 0 | `unit-terrain` 5, `structure-overlap` 2, `drop-cell` 1 |
| RU | 34 | 0 | `unit-terrain` 5, `drop-cell` 1 |

Instrument: `mapedit -check` over `-list` of each root, headless, and the
release test above. The `ALM-CLS-051` residue on maps 131 and 150 produces no
finding under these rules. An installed map is never refused.

## Proof

- `pkg/game` (`mapcheck_test.go`): the rule table's kinds as data; one
  synthetic case per rule that produces its finding, each with its loss control
  (the same map without the defect reports nothing); the mission-build refusal
  over a table whose rows are too short; the terrain refusal over an empty and
  a 4x4 map and a strip-group-13 tile; the report kept for the same bytes; Save
  As and Save into a mod refused with nothing written, then allowed after the
  repair.
- `pkg/ui` (`mapeditorreport_test.go`): the report key, selecting a finding
  selects its record and scrolls, a refused Save As and Save into a mod keep the
  report on screen, a save with warnings lists them, a host without a report.
- Release, EN and RU: `TestReleaseMapEditorRefusesASaveThatLeavesAScriptDangling`
  (open mission 10, delete a unit a script names, Save As into a temporary mod
  folder is refused with the `script-reference` finding, the window selects the
  script record, undo, Save As writes the installed bytes),
  `TestReleaseMapEditorSavesAUnitOnGroundItCannotEnterAndListsTheWarning`,
  `TestReleaseEveryInstalledMapReportsNoError`,
  `TestReleaseMapEditorHelpLinesFitTheRail`,
  `cmd/mapedit.TestReleaseCheckPrintsTheFindingsTheWindowLists` (`-check` over
  every installed map prints the lines the window lists).
- `TestReleaseMapEditorCloneNeverTakesADeletedScriptedUnitID` reads the
  model's bytes, since its edit now leaves a dangling reference and Save As
  refuses it.

## Open debt

- The report is computed on a key press and on a save, not on every edit, so
  the list rows of the record list do not carry its findings; the inspector's
  per-record notes (art, definition, anchor) stay as they were.
- `mission-build` runs with an empty party and mission number from the opened
  installed map, else 0; a map the build refuses only for a particular party is
  not found.
- The Target_Group rule counts a group no unit holds; whether the original
  treats such a reference as an error is Unknown.
- The rule table is code; reading it from a data file is a later step.
- The `structure-kind` and `item-code` rules check the installed tables, so a
  mod map that uses a structure class or item code the mod itself adds is
  refused by Save into a mod (review note 1; owner-confirmed rules, DIV-3133,
  DIV-3134). Terrain costs come from the install's `map.reg`, not the mod's.
- The help line test measures the prefix with the report key (285 of 300
  pixels); the whole line is 531 pixels and the rail cuts it, as on main.
