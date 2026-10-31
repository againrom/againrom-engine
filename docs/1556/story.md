# The map editor's two side panels

## Intent

The map editor gains two permanent side panels around the map. The left panel, "Objects and
scenarios", is a tree of the map's players, triggers, units, structures and groups with a search
field. The right panel, "Properties", shows the selected record's window. The map fills the
rest. Each panel opens, closes and resizes, and the editor profile keeps its state.

## Authority

- Owner proposal (a layout picture and its points), then «Запускай историю с панелями»: keep the
  retro look; three permanent areas; on the left «Объекты и сценарии», a tree with a search; the
  map in the centre; on the right the selected object's properties (a trigger's «Когда/Сделать»);
  panels resize, hide and remember their state; a complex entity such as a full inventory keeps a
  separate, larger window.
- The map editor is this project's own program. No claim is cited and no DIV row is added.

## As built

### Kinds and builders

| kind | builder |
|---|---|
| editor window | `popupSpec`, `layoutPopup`, `drawPopup` (`pkg/ui/editorpopup.go`), unchanged. A window in a panel is the same popup: `inPanel` gives it the panel's rectangle as `At`, `Size` and `Bounds`. The placement is `popupPlace` {`placeFloating`, `placeLeft`, `placeRight`}, data in the `sidePanels` table |
| side panel | `sidePanelRect` (`pkg/ui/mapeditorsidepanels.go`) over the two rows of `sidePanels` {place, profile name, default, least and greatest width, title word, key, help label, keyboard key} |
| map rectangle | `mapArea` (`pkg/ui/mapeditorwindow.go`) takes `panelsShown` off the window; `barHeight` and `statusRect` keep their builders; `mapOrigin` puts the map frame right of the left panel |
| editor list rows | `Picker` and `drawListBox` (`pkg/ui/widgetscroll.go`) with `PickerRow.Indent` and `PickerRow.Fold` as data, and `Picker.SetRowLimit` for a list taller than 25 rows; zero values draw every existing list unchanged |
| editor text field | the one input (`inputTarget`, `inputAt` in `pkg/ui/editorinput.go`, `editorinputview.go`); the tree search is one more target |
| editor selection | `treeChoose` calls `Select(index, jump)`, the call a map click makes, and `chooseGroup(g, true)`, the call a group number key makes |
| object, trigger and player windows | `objectSpecBase` and `objectSpec` (`pkg/ui/mapeditorresize.go`), hosted through `hostedObject` |
| border drag | `resizeEdgeAt` and `resizePopup` (`pkg/ui/mapeditorresize.go`) route a panel to `panelEdgeAt` and `resizePanel` |
| editor profile file | `editormcp.WriteProfileFile` (`pkg/editormcp/profilefile.go`): `writeRecord` writes `mapedit-mcp.json` through it byte for byte as before; `ui.WritePanelState` writes `mapedit-panels.json` through it; `cmd/mapedit/profile.go` resolves the profile with `game.ResolveRuntimeProfile`, as `startMCP` does |
| words | `pkg/words/text/en/strings.toml` and `ru/strings.toml` (`editor.side.*`, `editor.keys.panels`, `editor.keys.panelfloat`) |

### Behaviour

- Left panel: a search row over the tree list. Branches Players, Triggers, Units, Structures and
  Groups, each `Name (count)`; an empty branch shows `(0)`. Member rows show the text the editor
  already lists: `recordSummary` for players, units and structures, the trigger list's row text
  for triggers, the group label for groups. A press on a branch row closes or opens it.
- Search: keeps members whose name or ID holds the text, case-insensitive (`Unit #3`, `unit3`,
  `Trigger 1`, `Hide them`, `Group 2`); branches with matches show open with their match count,
  others leave. A click on the field focuses it; Enter ends typing; Escape clears the search.
- A member row selects as a map click does; a unit or structure row also centres the map on it.
  A trigger or player row selects it for the right panel. A group row chooses the group as its
  number key does. The tree is rebuilt from the document each frame, so it follows add, delete,
  undo, redo, load and new map. A selection made on the map marks its row and opens its branch.
- Right panel: the selected record's window (record inspector, person window, trigger window
  «Когда/Сделать», player window) with the same rows, edits and words, scrolled to fit the
  panel. With nothing selected it shows one hint line. A window that floats leaves a line naming
  where it is.
- Floating windows: the mod window, the dialogue page editor, the variable window, the group and
  multi-selection windows and the person window's inventory tab open as today. A double click on
  the title row of the window in the right panel takes it out to float; the next selection comes
  back into the panel. No button was added.
- Keys: F6 opens and closes the left panel, F7 the right one (key rows `objectspanel` and
  `propertiespanel`, help line `editor.keys.panels`, and `editor.keys.panelfloat` for the double
  click). The View menu has two entries, «Объекты и сценарии» and «Свойства», marked while open.
- Widths: left 260 (least 180, greatest 420), right 520 (least 360, greatest 800). A drag of the
  inner edge stops at these and where the map would fall below 480.
- Fit: a panel shows only while the map keeps 480 px; a narrower window hides the right panel
  first, then the left. A fresh, missing or broken profile asks for both panels at default
  widths, so 1280x800 shows both (map 500), 1024x768 the left only (map 764), 640x480 none.
  A panel the window has no room for stays closed when its key is pressed and says so.
- Profile: `mapedit-panels.json` beside `mapedit-mcp.json`, `{"left":{"open":..,"width":..},
  "right":{..}}` and a newline, written on each key, menu choice or edge drag. A failed write
  shows its error; the change stands until the editor closes.
- With both panels closed every rectangle, the map frame and the window behaviour are those of
  the editor before this story.

## Proof

| item | test |
|---|---|
| (a) closed panels keep map, bar, status strip and popup rects | `TestMapEditorClosedPanelsKeepTheChromeRects` against `pkg/ui/testdata/mapeditor_chrome_rects.golden`, recorded by `chromeRectsText` on base e48509ed3 (10 states at 1280x800, 1024x768, 640x480) |
| (b) widths, least map width, defaults, keys, menu, edge drag | `TestMapEditorPanelsTakeTheirWidthsOffTheMap` |
| (c) tree rows and counts | `TestMapEditorTreeRowsFollowTheMap` |
| (d) search | `TestMapEditorTreeSearchKeepsMatchingRows` |
| (e) tree choice equals map click | `TestMapEditorTreeChoiceIsAMapClick` |
| (f) tree after add, delete, undo, redo, load equals a fresh tree | `TestMapEditorTreeFollowsEdits` |
| (g) right panel rows equal the floating window's | `TestMapEditorRightPanelHostsTheFloatingWindow` |
| (h) state round trip, defaults, MCP bytes | `TestMapEditorPanelStateRoundTrip`, `TestProfileWriterKeepsTheMCPRecordBytes` (`pkg/editormcp`), `TestPanelStateLivesInTheEditorProfile` (`cmd/mapedit`) |
| goldens | `TestMapEditorSidePanelsGolden` (`pkg/ui/testdata/mapeditor_side_panels.golden`): `trigger-en-1280x800`, `trigger-ru-1280x800`, `search-en-1280x800`, `defaults-en-1024x768` |

`pkg/ui/testdata/mapeditor_modes.golden` was re-recorded: only its `help` column changed (78
rows), the F1 list's two new lines. `mapeditor_speaker_portraits.golden` is unchanged.

Release tests for the next release: `TestReleaseMapEditorHelpLinesFitTheRail` (the new help
lines at the install font, EN and RU), `TestReleaseMapEditorRefusesASaveThatLeavesAScriptDangling`,
`TestReleaseNewWritesTheBlankMapAndPrintsItsReport`, `TestReleaseCheckPrintsTheFindingsTheWindowLists`.

## Open debt

- With nothing selected the tree's list keeps a highlighted row (the first row or the last row
  pressed): the list model always holds a selection.
- Each panel draws the popup frame's resize grip at its outer bottom corner; only the inner edge
  drags.
- Member rows read as `recordSummary` words them, with the record kind in English in both
  languages (`Player #0`, `Unit #0`), as the editor's lists already do.
- No window is chosen to float by its size; a hosted window scrolls its rows to fit the panel.
- Fixture tests only; no install witness ran.
