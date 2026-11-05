# A new trigger takes one click; the «Сценарий» palette lists the script records

## Intent

Before, a new trigger took the Add window's three steps (kind, name, once), and the «Сценарий»
(Scenario) palette held only its hint, the drop zone switch and «Добавить» (Add). Now the palette
lists «Новый триггер» (New trigger) first and then the rows of the tree's «Сценарий» branch, in the
tree's order and words. A click on a record row opens the record as its tree row does. One click on
«Новый триггер», F4, or the «Инструменты» (Tools) menu row adds the trigger the Add window adds with
its proposals accepted. The trigger opens at once in the wizard, its name field focused with the
proposed name selected, so typing replaces the name.

## Authority

- Owner direction: the story brief (`review/story1577-one-click-trigger/BRIEF.md`).
- The map editor is our program, so no divergence row and no claim applies.

## As built

| kind | builder |
|---|---|
| tree rows and the Scenario palette's record rows | `pkg/ui/mapeditorsidepanels.go` `treeRowsOf`: `treeRows` passes every category with the tree's search and closed branches; `scenarioRows` (`pkg/ui/mapeditorpalettes.go`) passes the Scenario category open and unsearched, removes the branch row and puts New trigger at index 0 |
| list popup rows and choice | `listRows("records")` and `listChoose` in `pkg/ui/mapeditorwindow.go`; `scenarioChoose` sends row 0 to `menuAct("newtrigger")` and a member row to `memberChoose`, the tree's member choice moved out of `treeChoose` |
| the trigger's proposals and record | `pkg/ui/mapeditoradd.go` `addAccepted`, the Add window's completion moved out of `addPlacement`: every step takes the host's `NewRecordStep` proposal through `addOffer`, then `addRecord`; `commandAddRecord` writes it, and the host's `addScriptRecord` adds the trigger |
| the one-click action | `pkg/ui/mapeditortrigger.go` `addTriggerNow`: it leaves picks and prompts, closes the list popup with `closeList` (the Map panel included), adds the trigger, selects it with `selectNewest`, loads the wizard, focuses `trigName` and selects its text |
| the command row | key table row `newtrigger` on F4 (`pkg/ui/mapeditorkeys.go`, act `menuKey`, which calls `menuAct`), toolbar table row `newtrigger` with no icon in the Tools menu (`pkg/ui/mapeditortoolbar.go`), the native F4 in `MapEditor.Update`, the help line `editor.keys.newtrigger` from the key row |
| history | `WizardTrigger.Join` (`pkg/ui/mapeditorwizardhost.go`): the first written Apply of the wizard session the one click opened, with no save in between, joins the add's history entry in `WriteTrigger` (`pkg/game/mapinspectionwizard.go`) |
| headless readers | `ListLines`, `OpenWindows`, `FieldFocus` |
| words | `editor.tool.newtrigger` («Новый триггер»), `editor.keys.newtrigger` («F4: новый триггер») |
| static guard | `internal/archtest/editorscenariorows_test.go` |

Facts the brief asks for:

- Name field: the proposed name is selected (`selectAll`: anchor 0, caret at the end).
- Cancel leaves the trigger in the map, and one undo removes it. Apply after typing writes the
  name in the add's history entry, so one undo removes the named trigger and one redo restores it.
  The join ends with the wizard session the one click opened: Cancel, the popup's close or the
  selection moving off the trigger ends it, and so does a written save. A later Apply of the same
  trigger is a history entry of its own, so after a save one undo returns the saved map, and
  after Cancel and a reopening one undo takes back only the rename. Any edit, undo or redo between
  the add and the Apply replaces the editor's document, and the Apply is then its own entry too.
- The palette stays closed after the wizard closes, by Apply or Cancel.
- F4 was free: no key table row, no native binding in `MapEditor.Update` and no text field reads
  it. The game's F4 QuickSave is `pkg/ui/app.go`, a separate program loop. While a field has
  focus, the native key block is off, so F4 does nothing and typing goes to the field.
- The Add window («Добавить», and the wizard's «Новый», which opens it through `newTrigger`) is
  unchanged: kind, name, once.

## Proof

Focused tests at 1920x1080, EN and RU:

- `pkg/game/mapeditortriggernow_test.go`:
  - `TestMapEditorScenarioPaletteListsTheTreesRows`: the palette lines equal New trigger and then
    the tree's Scenario branch lines one step less indented. Every row shows inside the window.
    Each member row opens the record, selection and popup lines that its tree row opens.
  - `TestMapEditorNewTriggerTakesOneClick`: the Add window asks `kind, name, once`. One click writes
    the same bytes as the Add window with defaults accepted. The wizard is the only open window,
    the name field is focused with the name selected, and the map is dirty.
  - `TestMapEditorOneClickTriggerUndoesInOneStep`: typing and Apply write "Gate", one undo gives
    the opened bytes and one redo gives the named bytes. Cancel keeps the trigger, closes every
    window, and one undo and one redo take it out and back.
  - `TestMapEditorOneClickJoinEndsWithTheWizardSession`: after a save, rename and Apply, one undo
    gives the saved bytes and a second the map before the add. After Cancel and a reopening from the
    palette, one undo takes back the rename and a second removes the trigger.
  - `TestMapEditorNewTriggerKeyWorksFromTheMap`: native F4 from the map view writes the click's
    bytes and opens the wizard alone. F4 in the focused name field adds nothing and keeps the
    selection, and typing then names the trigger. With the Map panel open, F4 closes the panel.
- `pkg/ui/mapeditortriggernow_test.go` `TestMapEditorNewTriggerIsOneCommandRow`: one key table
  row, the only row on F4; one help line; one toolbar table row with no icon; one Tools menu row
  reading the word and F4; no row in any other menu.
- `internal/archtest`: `TestEditorScenarioRowsAndOneClickTriggerHaveOneBuilder` and its self-test
  `TestEditorScenarioGuardSeesASecondBuilder`. In `pkg/ui`, `treeMembers` is named only by
  `treeMatches`, `treeRowsOf` and `treeFollow`. `treeRowsOf` is named only by `treeRows` and
  `scenarioRows`. `addAccepted` is named only by `addPlacement` and `addTriggerNow`.
  `NewRecordStep` is named only by `addOffer` and `placeChoices`. `addTriggerNow` is named only
  by `menuAct`.

Failure at the base (`433b3eeb3`): the base tree came from `git archive`, with the new test files
and the three readers copied in. All five new tests fail there. The palette lists no row; the key
table has no row on F4; F4 leaves 2 triggers, not 3. With the history join switched off, the undo
test fails with "one undo left 3 triggers". With the old list close (`listShown`, `filter` reset),
the Map panel case fails with windows `["object" "list"]`.

Existing tests changed:

- `pkg/ui/mapeditormenus_test.go`: the key route ends field typing before the key, as a menu
  click does.
- `pkg/ui/mapeditortoolbar_test.go`: New trigger stands in the Tools menu; the Scenario Add's
  place is gone.
- `pkg/ui/mapeditorpalettes_test.go`: the Scenario palette has a list.
- `pkg/ui/mapeditorselection_test.go`: the Russian help check takes F4 as a key name.
- `pkg/game` arms, scripts list and script add tests: `closePalette` before a tree click.

The Scenario palette opened while the left panel is minimized stays where it stood. When the
panel then opens, the palette's list covers the tree's top rows.

Golden: `pkg/ui/testdata/mapeditor_modes.golden`, re-recorded once with
`MAPEDITOR_MODES_WRITE=1`. It gains the column `newtrigger`. Of the 112 rows, 33 are unchanged.
In 79 rows the columns `triggers`, `scripts` and `help` change: the palette lists rows, and the
help screen has the F4 line. The rows `full/add-script`, `full/triggers` and `full/scripts`
change in every column, because the palette stands open there. The column comparison is in the
lane notes (`tmp/modes-cols.txt`).

Release tests for the next release, EN and RU, not run in the lane:

- `TestReleaseMapEditorScriptsListUntouchedBytes` (changed: `closePalette` before the tree
  clicks);
- `TestReleaseMapEditorScriptsListFrames`, `TestReleaseMapEditorOwnerFrames`,
  `TestReleaseMapEditorTriggerFrames` and `TestReleaseEditorUnitBlockFrames` (their frames show
  the Scenario palette with its list);
- `TestReleaseMapEditorScriptVariablesSaveUnchanged`.

## Open debt

- Out of scope: a trigger template list; a condition or action made from the palette in one
  click; the MCP commands for triggers (`write_trigger` keeps `Join` false); the second game's
  installs; the side panels' layout.
- The Scenario palette does not move when the left panel opens under it.
- `WizardShown` loads the selected record's popup, so after Cancel it reports the wizard as shown
  while no window is open. The tests read `OpenWindows`.

## Later change

The Scenario palette no longer lists «Новый триггер» as row 0 and no longer has «Добавить»; its
buttons are «Зоны высадки (D)», «Новый триггер» and «Новая переменная». Both new buttons, F4, the
Tools row and the wizard's «Новый» go through `addNow` (`pkg/ui/mapeditortrigger.go`), the one-click
add of a script record by kind, and the script Add window with its kind list is gone. The rows,
tests and routes described above that read row 0 or the Add window stand replaced by
`pkg/game/mapeditorscenariobuttons_test.go` and `pkg/ui/mapeditorscenariobuttons_test.go`.
