# The map editor's two side panels

## Intent

The map editor gains two permanent side panels around the map. The left panel, "Objects and
scenarios", is a tree of the map's records with a search field. The right panel, "Properties",
shows the selected record's window. The map fills the rest. Each panel minimizes to a strip,
restores and resizes, and the editor profile keeps its state.

The owner's correction after the first try: the tree's branches are the bar's record entries in
the bar's order and words; a record entry's floating window becomes that category's palette and
lists no records; palette and tree move together; a panel minimizes instead of closing; a palette
grid cell shows a hover tooltip naming it.

## Authority

- Owner proposal (a layout picture and its points), then «Запускай историю с панелями»: keep the
  retro look; three permanent areas; on the left «Объекты и сценарии», a tree with a search; the
  map in the centre; on the right the selected object's properties (a trigger's «Когда/Сделать»);
  panels resize, hide and remember their state; a complex entity such as a full inventory keeps a
  separate, larger window.
- Owner correction after the first try (`owner-try-1.webp` in the lane notes): one category
  table for the bar and the tree; the floating window is the category's palette; palette and
  tree move together; minimize instead of close; the branch title glyph checked; and «в палитрах
  надо сделать ховер-тултип 200-300 мс с тем, что это такое».
- The map editor is this project's own program. No claim is cited and no DIV row is added.

## As built

### Kinds and builders

| kind | builder |
|---|---|
| editor window | `popupSpec`, `layoutPopup`, `drawPopup` (`pkg/ui/editorpopup.go`). A window in a panel is the same popup: `inPanel` gives it the panel's rectangle as `At`, `Size` and `Bounds`. The placement is `popupPlace` {`placeFloating`, `placeLeft`, `placeRight`}, data in the `sidePanels` table. `popupSpec.CloseLabel` puts a word on the title row's one button; empty keeps the close cross |
| record category | `editorCategories` (`pkg/ui/mapeditorsidepanels.go`), one row each {key, word, list filter, members}; `barKeyRows` builds the bar's record entries from it and `treeRows` the tree's branches; the palette is the list popup that `listSpecBase` builds for the row's filter |
| side panel | `sidePanelRect` over the two rows of `sidePanels` {place, profile name, default, least and greatest width, title word, key, help label, keyboard key, minimize and restore words}; `panelsLaid` decides restored or minimized for the window; `stripSpec` is a minimized panel |
| map rectangle | `mapArea` (`pkg/ui/mapeditorwindow.go`) takes `panelsShown` (a panel's width, or its strip) off the window; `barHeight` and `statusRect` keep their builders; `mapOrigin` puts the map frame right of the left panel |
| editor list rows | `Picker` and `drawListBox` (`pkg/ui/widgetscroll.go`) with `PickerRow.Indent` and `PickerRow.Fold` as data, and `Picker.SetRowLimit` for a list taller than 25 rows |
| editor text field | the one input (`inputTarget`, `inputAt` in `pkg/ui/editorinput.go`, `editorinputview.go`); the tree search is one more target |
| editor selection | `treeChoose` calls `Select(index, jump)`, the call a map click makes, and `chooseGroup(g, true)`, the call a group number key makes |
| object, trigger, player and group windows | `objectSpecBase` and `objectSpec` (`pkg/ui/mapeditorresize.go`), hosted through `hostedObject`; the group window's rows are built in `pkg/ui/mapeditorobject.go` |
| group list rows | `groupsList` (`pkg/ui/mapeditorgroups.go`): the tree's Groups members and `GroupsText` |
| tooltip | `tooltipController` and `composeHoverBox` (`pkg/ui/tooltip.go`, `widgethover.go`) with `hoverBoxInk` as data for the editor's fill and bevel; `pkg/ui/mapeditorhover.go` times a grid cell and draws its box through them |
| border drag | `resizeEdgeAt` and `resizePopup` (`pkg/ui/mapeditorresize.go`) route a panel to `panelEdgeAt` and `resizePanel` |
| editor profile file | `editormcp.WriteProfileFile` (`pkg/editormcp/profilefile.go`): `writeRecord` writes `mapedit-mcp.json` through it byte for byte as before; `ui.WritePanelState` writes `mapedit-panels.json` through it; `cmd/mapedit/profile.go` resolves the profile with `game.ResolveRuntimeProfile`, as `startMCP` does |
| words | `pkg/words/text/en/strings.toml` and `ru/strings.toml` (`editor.side.*`, `editor.keys.panels`, `editor.keys.panelfloat`, `editor.group.delete`) |

### Behaviour

- Categories: Scripts, Players, Groups, Units, Sacks, Structures, in this order, each worded as
  its bar entry. The bar's record entries and the tree's branches read the one table.
- Objects and Trees are brushes: their bar entries arm a brush and they get no branch.
- Left panel: a search row over the tree list. Each branch shows `Name (count)`; an empty
  branch shows `(0)`. Member rows show the text the editor already lists: `recordSummary` for
  players, units, sacks and structures, the scripts list's row text under its section headings
  (Triggers, Variables, Conditions, Actions; an empty section shows `Name: none`), the group label
  for groups.
- Search: keeps members whose name or ID holds the text, case-insensitive (`Unit #3`, `unit3`,
  `Trigger 1`, `Hide them`, `Group 2`); branches with matches show open with their match count,
  others leave. A click on the field focuses it; Enter ends typing; Escape clears the search. A
  floating window over the field takes a press there.
- A member row selects as a map click does; a unit or structure row also centres the map on it.
  A group row chooses the group as its number key does. The tree is rebuilt from the document
  each frame, so it follows add, delete, undo, redo, load and new map. A selection made on the
  map marks its row and opens its branch.
- Palettes: a record entry's window holds what places or acts, not records.
  - Units and Structures keep the placement grid, its note and Add.
  - Sacks keeps Add; it has no placement grid.
  - Scripts keeps the hint, the drop zone switch and Add.
  - Players keeps the relations grid and its buttons.
  - Groups keeps the selection notes and To group and Remove from group.
  - Record rows and the "x-y / n" count are gone. A palette with nothing to show does not open;
    its bar entry then only opens the branch.
- Moved to the right panel: the group window now carries the group's row ("Group N", the
  members' icons, "N units") and a Delete group button, which replaces the Groups list row's
  cross and asks first when a trigger uses the group. Its Used by links were already there. Every
  other list row was a record the tree now lists.
- Palette and tree together:
  - Opening a palette by its bar entry, its key, the MCP `key` or `pointer` tool restores a
    minimized left panel, opens the branch, clears a search that hides it and scrolls its title
    row into view.
  - A press on a branch row opens or closes it and shows its palette; a member chosen in another
    branch than the last shows that branch's palette.
  - A branch whose palette is empty closes a shown record palette.
- Right panel: the selected record's window (record inspector, person window, trigger window
  «Когда/Сделать», player window, group window) with the same rows, edits and words, scrolled to
  fit the panel. With nothing selected it shows one hint line. A window that floats leaves a line
  naming where it is.
- Floating windows: the mod window, the dialogue page editor, the variable window, the
  multi-selection window and the person window's inventory tab open as before. A double click on
  the title row of the window in the right panel takes it out to float; the next selection comes
  back into the panel.
- Minimize: each panel's title row has one button, `<` on the left panel and `>` on the right
  (`editor.side.toleft`, `editor.side.toright`, the same in EN and RU). It minimizes the panel to a
  strip one button wide (`panelStrip`: the title row's button height plus `popupRowExtra` plus 2;
  16 px with the test font). The strip shows one button, `>` on the left and `<` on the right; it
  restores the remembered width. F6 and F7 and the View menu's two entries minimize and restore.
- Widths: left 260 (least 180, greatest 420), right 520 (least 360, greatest 800). A drag of the
  inner edge stops at these and where the map would fall below 480.
- Fit: a panel shows restored only while the map keeps 480 px; a narrower window minimizes the
  right panel first, then the left. A fresh, missing or broken profile asks for both panels
  restored at default widths, so 1280x800 shows both (map 500), 1024x768 the left one and the
  right strip (map 748), 640x480 two strips. A panel the window has no room for stays minimized
  when its key is pressed and says so.
- Profile: `mapedit-panels.json` beside `mapedit-mcp.json`, `{"left":{"minimized":..,"width":..},
  "right":{..}}` and a newline, written on each button, key, menu choice or edge drag. A file
  with the earlier `"open"` field reads as restored or minimized. A failed write shows its error;
  the change stands until the editor closes.
- Tooltip: the pointer resting 300 ms (`editorTipDelay`) on a palette grid cell shows a hover box
  in the editor's body font, paper ink, panel fill and gold bevel, beside the pointer. Moving off
  the cell or pressing hides it. The words:
  - Units and Structures placement grid: the cell's choice name, the word the grid already drew
    as its label.
  - Objects and Trees brush grid: `objectWord`, the object's name.
  - Height brush modes: `heightModeWord`.
  - Players relations grid: the `editor.relation.cell` line naming the two players and the
    relation.
- Branch title glyph: the mark over the "U" of "Units (42)" is the capital U of the install's
  font1, which carries a calligraphic swash. It is the install font's glyph, not a drawing
  fault; nothing changed.
- With both panels minimized the map is the base map rectangle less the two strips; the bar,
  the status strip and every window that is not a record category's palette keep their rows and
  behaviour.

## Proof

| item | test |
|---|---|
| (a) minimized panels: map = base less the strips; other windows keep their rects | `TestMapEditorMinimizedPanelsKeepTheChromeRects` against the base record `pkg/ui/testdata/mapeditor_chrome_rects.golden` (unchanged, recorded on base e48509ed3; 10 states at 1280x800, 1024x768, 640x480) |
| (b) widths, least map width, defaults, keys, menu, edge drag | `TestMapEditorPanelsTakeTheirWidthsOffTheMap` |
| (c) tree rows and counts | `TestMapEditorTreeRowsFollowTheMap` |
| (d) search | `TestMapEditorTreeSearchKeepsMatchingRows` |
| (e) tree choice equals map click | `TestMapEditorTreeChoiceIsAMapClick` |
| (f) tree after add, delete, undo, redo, load equals a fresh tree | `TestMapEditorTreeFollowsEdits` |
| (g) right panel rows equal the floating window's | `TestMapEditorRightPanelHostsTheFloatingWindow` |
| (h) state round trip, defaults, earlier format, MCP bytes | `TestMapEditorPanelStateRoundTrip`, `TestProfileWriterKeepsTheMCPRecordBytes` (`pkg/editormcp`), `TestPanelStateLivesInTheEditorProfile` (`cmd/mapedit`) |
| tree order equals the bar order, no Objects or Trees branch, EN and RU | `TestMapEditorTreeBranchesAreTheBarsRecordEntries` |
| palettes hold no record row | `TestMapEditorPalettesHoldNoRecordRow` |
| routes: bar, key, MCP key, MCP pointer; restore, open, scroll | `TestMapEditorPaletteRoutesRevealTheirBranch` |
| branch press and member choice show the palette; empty palette | `TestMapEditorTreeShowsTheBranchPalette` |
| a palette over the search field drags by its title | `TestMapEditorPaletteOverTheSearchFieldMovesByItsTitle` |
| minimize, strip width, restore button, words, profile, F6 | `TestMapEditorPanelsMinimizeToAStrip`; the narrow order in (b) and (a) |
| group row and Delete group in the right panel | `TestMapEditorGroupWindowHoldsTheGroupRow` |
| tooltip delay, words, hide | `TestMapEditorPaletteCellTooltipWaitsForTheDelay`, `TestMapEditorPaletteGridsNameTheirCells` |
| goldens | `TestMapEditorSidePanelsGolden` (`pkg/ui/testdata/mapeditor_side_panels.golden`): `trigger-en-1280x800`, `trigger-ru-1280x800`, `search-en-1280x800`, `defaults-en-1024x768` |

Goldens re-recorded by the correction:

- `mapeditor_side_panels.golden`: the tree's branch order and words, the Scripts branch with its
  section headings, the Sacks branch, the frame hashes; `defaults-en-1024x768` shows the right
  strip (`panels=[260 16]`, map to x 1008).
- `mapeditor_speaker_portraits.golden`: the frame hash only; the frame draws the new tree and
  the title rows' minimize buttons.
- `mapeditor_modes.golden`: every frame hash; the frames draw the new tree, the minimize buttons
  and palettes without record rows.

Release tests for the next release: `TestReleaseMapEditorHelpLinesFitTheRail`,
`TestReleaseMapEditorRefusesASaveThatLeavesAScriptDangling`,
`TestReleaseNewWritesTheBlankMapAndPrintsItsReport`,
`TestReleaseCheckPrintsTheFindingsTheWindowLists`, and the ones the correction rewrote to read the
tree: `TestReleaseMapEditorOwnerFrames`, `TestReleaseEditorInteractionFrames`,
`TestReleaseMapEditorScriptsListUntouchedBytes`, `TestReleaseMapEditorScriptsListFrames`,
`TestReleaseMapEditorUsedByCensusPreservesBytes`, `TestReleaseMapEditorObjectsAndGroupsFrames`,
`TestReleaseMapEditorGroupPopupFollowsUndoAndRedo`, `TestReleaseMapEditorPlacingAndGroupsWithoutAnEditSaveUnchanged`.

## Open debt

- At 640x480 the left panel cannot restore (the map would keep less than 480 px), so the tree is
  out of reach; records are chosen on the map or in the View menu's Records list.
- A floating window's bounds are the whole window, so it can lie over a strip; a window placed
  at the bottom right covers the lower part of the right strip.
- The Groups palette opens: it keeps the selection notes and its two buttons.
- The grid cells' tips now show after the delay; a column head and a cut row keep the immediate
  tip of the popup builder.
- With nothing selected the tree's list keeps a highlighted row: the list model always holds a
  selection.
- Each panel draws the popup frame's resize grip at its outer bottom corner; only the inner edge
  drags.
- Member rows read as `recordSummary` words them, with the record kind in English in both
  languages (`Player #0`, `Unit #0`), as the editor's lists already do.
- Fixture tests only; no install witness ran.
