# The map editor's Tools, View and Window menus

## Intent

The bar holds three word menus right after Map: Tools (RU «Инструменты»), View (RU «Вид») and
Window (RU «Окно»). They give every toolbar command a word path beside the icons, and Window lists
every panel the editor has with a mark while it shows. Each opens the one bar drop-down the Map
menu opens. The rows are the toolbar's own table rows: a row reads as its button's tip, carries its
button's mark and is disabled while its button is. The Window menu's toolbar row hides the toolbar,
docked or floating, and the editor remembers it. A bar category entry («Отряды» and the rest)
opens its palette and no longer restores a minimized left panel or unfolds a tree branch.

## Authority

- Owner, on a tester's remarks: a Window menu with every panel the editor has and whether it is on
  («добавить WINDOW со всеми нашими настройками какие панели в данный момент доступны»).
- Owner, on the tester's remark that he keeps to word buttons: bring back the word drop-downs for
  the brushes and the view beside the toolbar («добавь отдельный дропдаун который у нас был про
  кисти и обзор... просто чтобы у нас остался такой способ»). They are the Tools and View menus
  `docs/1564/story.md` turned into the toolbar.
- Tester, through the owner: «Отряды» brought back a left panel he had switched off and unfolded
  the unit list. Seat decision, open to the owner: a bar category entry opens its palette and
  leaves the side panels and the tree as they are.
- Owner: screens are at least 1920x1080; a 640x480 editor test this change breaks is deleted.
- The map editor is our own program: no claim, no DIV row.

## As built

| kind | builder |
|---|---|
| bar entry | `barKeys` / `barRows` / `barEntries`: three `barKey` rows with `menu: true`; `barEntries` and `barAct` treat every menu entry alike |
| bar menu and its rows | `menuItems` and `menuDropdown`: the Map rows stay its own; the other three menus take their rows from `toolbarEntries`; `menuItem.off` is a disabled row, drawn in the hint ink and not choosable; the list widens to its longest row |
| editor command row (toolbar button and menu row) | `toolButtons` and `toolbarEntries`: each row names its `menu` and, for the terrain brush, the `heading` word over it; the rows `toolbar` and `reportpanel` have no icon and stand in the Window menu only; `toolbarEntries` places only rows with an icon and gives every row its state |
| row word | `toolWord`, the word and its key label, read by the toolbar tip and the menu row |
| row action | `menuAct`, the path a toolbar button already took; the `toolbar` row calls `toggleToolbar` |
| switch caption | `markWord` / `CheckSpec` |
| side panel state | `sidePanels`, `panels.state`, the F6/F7/F8 path through the row's `keyRow` |
| toolbar layout: docked, floating, hidden | `toolbarRect`, `toolbarSpec`, `toolbarGrip`, `toolbarEntries` read `panels.toolbarHidden`; the placement store row `popToolbar` keeps the floating rectangle while hidden |
| editor profile file | `panelStateBytes` writes `"hidden": true` in the toolbar entry; the existing reader `panelFloatsOf` reads it through `profileEntries`; no new reader |
| category palette and tree | `showRecords` and `openPalette`; `treeReveal` is gone; `treeFollow` still unfolds the branch of a record the map maker selects; `selectUnfollowed` is `Select` for the record the Players or Scripts entry selects itself, marked as followed so the tree stays as it is |
| editor words | the EN and RU word tables: `editor.bar.tools`, `editor.bar.view`, `editor.bar.window`, `editor.tool.brushes`, `editor.tool.toolbar`, `editor.tool.reportpanel` |

The menus' rows, EN (RU), with the marks of a fresh map:

- Tools: `[x] Select` (Выбор); heading Brushes (Кисти); `[ ] Terrain brush (T)` (Кисть ландшафта),
  `[ ] Height brush (H)` (Кисть высот), `[ ] Objects brush (B)` (Кисть объектов),
  `[ ] Passability brush (Q)` (Кисть проходимости); `[ ] Draw trigger area (L)` (Нарисовать область
  триггера).
- View: `[x] Map border`, `[x] Trigger areas`, `[x] Drop cells`, `[x] Footprints`, `[x] Issues`,
  `[ ] Height numbers (V)`, `[ ] Passability view (Z)`, `[ ] Player colours (Y)`, `[ ] Grid`,
  `Fit map (F)` (an action, no mark), `[ ] Report (R)`.
- Window: `[x] Toolbar` (Панель инструментов), `[x] Objects and scenarios panel (F6)`,
  `[x] Properties panel (F7)`, `[ ] Report panel (F8)` (Панель отчета). The search for other
  panels covered `sidePanels` (three rows) and the profile's `floatWindows` (the two side panels
  and the toolbar); it found none.

While the toolbar is hidden the map and the side panels start right under the bar, as while it
floats; a second press shows it docked or floating where it stood. A profile from an earlier build
has no `hidden` field and reads as shown. With `treeReveal` gone, a category entry no longer
clears a tree search that leaves its branch out, and the case-fold allow list in
`internal/archtest/casefold.go` loses its `treeReveal` entry.

## Proof

New tests in `pkg/ui/mapeditormenus_test.go`, at 1920x1080, EN and RU; at the base they do not
compile (no Tools, View or Window menu, no table fields, no hidden toolbar):

- `TestMapEditorToolsViewAndWindowMenusListTheToolbarRows`: the bar's first four entries, each
  menu's rows and order as one drop-down, the tip words, the concrete labels above, marks after
  grid, terrain, F6 and F8, and the four menus in the bar's first row.
- `TestMapEditorMenuRowsAreDisabledWhereTheirButtonsAre`: a map that takes no brush and no map.
- `TestMapEditorMenuRowPressDoesWhatItsButtonDoes`: every table row pressed twice from its menu
  against its button or key.
- `TestMapEditorEveryToolbarButtonIsAMenuRow`: the table test, and the menu and heading words in
  both tables.
- `TestMapEditorWindowMenuTogglesEachPanelAndTheToolbar`: the three panel rows; the toolbar row
  docked and floating, the map and panels under the bar, no button, hit or tip while hidden.
- `TestMapEditorHiddenToolbarProfileRoundTrip`: docked and floating round trips; files from an
  earlier build, with a floating toolbar only, with a broken field and none read as shown.

`TestMapEditorPaletteRoutesLeaveTheSidePanels` (`pkg/ui/mapeditorpalettes_test.go`, replacing
`TestMapEditorPaletteRoutesRevealTheirBranch`) fails at the base on every route: six bar
categories, F2, P, an MCP key and an MCP pointer leave a minimized left panel minimized, the closed
branches and the search as they were, and write no profile; a map click on a unit still unfolds
its closed branch.

Goldens moved by the bar's three new entries; a pixel dump of each frame at the base and the
candidate (`frames-pixel-diff.txt` in the lane notes) differs only inside the bar's first row,
y 3 to 13:

- `mapeditor_plain_views.golden` (16 frame hashes), `mapeditor_side_panels.golden` (9 frame
  hashes), `mapeditor_speaker_portraits.golden` (1 frame hash), and the three `chrome/*` pins of
  `TestFrameInstancesKeepTheirPixels`; no text line moved.
- `mapeditor_technical_reach.golden`: the Russian bar wraps to a second row at the test's 1280x800
  window with the fixture font, so the Russian panels at 520 and 800 show one row fewer.
- `mapeditor_modes.golden`, re-recorded once with `MAPEDITOR_MODES_WRITE=1`: every sampled probe
  with its pixel hash removed is the same at the base and the candidate (2380 probes).
- The popup placement transcript: its 1280x800 half differs from the base in 18 lines of the help
  scenario, where P no longer restores the minimized left panel; nothing else moved.

Deleted 640x480 cases, the bar wrapping to two rows there: the 640x480 size of
`TestMapEditorMinimizedPanelsKeepTheChromeRects` (10 golden lines) and the 640x480 half of
`TestEditorPopupPlacementTranscript` (1155 lines).

## Open debt

- The fixture font draws a Cyrillic letter twice a Latin letter's width, so the Russian bar is held
  only to the four menus in its first row. The bar's width with the game's font at 1920x1080 is
  for the release witnesses.
- Tests that clicked a category and read the tree now show the left panel first, as F6 does
  (`showTree` in `pkg/game/mapeditortrigger_test.go` and `showPanel` in the `pkg/ui` fixtures).
- Release witnesses, EN and RU, for the next release, in `pkg/game`: the editor frames that move
  with the bar, `TestReleaseMapEditorOwnerFrames`, `TestReleaseEditorInteractionFrames`,
  `TestReleaseMapEditorScriptsListFrames` and `TestReleaseMapEditorUsedByFrames`; and the tests
  that read the tree after a category entry, `TestReleaseMapEditorScriptsListUntouchedBytes` and
  `TestReleaseMapEditorUsedByCensusPreservesBytes`. None ran in the lane.
