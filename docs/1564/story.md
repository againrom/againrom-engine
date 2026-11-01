# The map editor's toolbar over the map

## Intent

The bar's View and Tools menus become one toolbar row between the bar and
the map, across the whole window width. The left group holds the tools:
select, terrain brush, height brush, objects brush, passability brush and
draw trigger area; the current tool is marked. The right group holds the
Layers drop-down, the grid switch, fit map, the report and the two side
panel switches. The report moves to a bottom panel below the map. Random
landscape and random heights move under a Generate heading of the Map menu.
New trigger moves to the Scripts palette's Add. Every row the View and
Tools menus held has one named place; none is dropped.

## Authority

- Owner, forwarding the layout proposal `pipeline/BACKLOG.md` "Editor:
  toolbar over the map": the View and Tools menus become a toolbar over the
  map, the report a bottom panel, the random rows a Generate entry of the Map
  menu, New trigger the Scripts palette's Add; the icons are drawn by our own
  code.
- Owner, «Отдельно после панелей»: a story of its own after the side panels.
- The map editor is our own program: no ROM1 claim, no DIV row.
- One builder per kind (`AGENTS.md`); the kinds and builders are listed
  below.

## As built

### Each row's new place

| Base row (menu) | Row key | Key | New place |
|---|---|---|---|
| Drop cells (View) | `drops` | none | Layers drop-down |
| Footprints (View) | `footprints` | none | Layers drop-down |
| Issues (View) | `issues` | none | Layers drop-down |
| Border (View) | `border` | none | Layers drop-down |
| Trigger areas (View) | `areas` | none | Layers drop-down |
| Height numbers (View) | `heightlabels` | V | Layers drop-down |
| Passability view (View) | `passview` | Z | Layers drop-down |
| Player colours (View) | `playertint` | Y | Layers drop-down |
| Grid (View) | `grid` | none | toolbar, right group |
| Fit map (View) | `fit` | F | toolbar, right group |
| Report (View) | `report` | R | toolbar, right group; the report docks in the bottom panel |
| Objects panel (View) | `objectspanel` | F6 | toolbar, right group |
| Properties panel (View) | `propertiespanel` | F7 | toolbar, right group |
| Records (View) | `records` | none | Map menu |
| Draw trigger area (View) | `drawarea` | L | toolbar, left group |
| Terrain brush (Tools, Brushes) | `brush.terrain` | T | toolbar, left group |
| Height brush (Tools, Brushes) | `brush.height` | H | toolbar, left group |
| Objects brush (Tools, Brushes) | `brush.objects` | B | toolbar, left group |
| Passability brush (Tools, Brushes) | `brush.passability` | Q | toolbar, left group |
| Random landscape (Tools, Whole map) | `randomterrain` | Ctrl+E | Map menu, under Generate |
| Random heights (Tools, Whole map) | `randomheights` | Ctrl+Shift+R | Map menu, under Generate |
| New trigger (Tools) | `newtrigger` | none | Scripts palette, Add, kind Trigger |

New: the toolbar's Select button (row key `select`) leaves every brush and
area tool, and the bottom panel switch `reportpanel` (F8).

Every row key keeps its meaning: `menuAct(key)`, the key table
`editorKeyRows` and MCP `action_<row>` and `key` reach the same act as on
the base. The keys help names the new places: "(Layers)" after V, Z and Y,
"(toolbar)" after L, "(Map, Generate)" after Ctrl+E and Ctrl+Shift+R, and
F8 beside F6 and F7.

### Toolbar

- One row of `toolbarHigh`, 28 pixels (a 22-pixel button and 3 pixels above
  and below), between the bar and the side panels.
  `mapArea` is the base's less the toolbar and the bottom panel.
- A button whose act the map refuses is drawn disabled and does nothing: on
  a map that takes no brush the four brushes; with no editable map the
  draw-area tool.
- A switch button (grid, report, the panels) is drawn pressed while on. The
  current tool among the left group is drawn pressed.
- Resting the pointer on a button shows its tip in the editor's hover style:
  its EN or RU word and its key, for example "Terrain brush (T)".
- The Layers button opens the one menu drop-down over the button; each row
  carries its check mark, and a press on a row toggles that switch; the
  mark follows the switch.

### Bottom panel

The third row of `sidePanels`: name `bottom`, key `reportpanel`, F8, height
180 pixels (96 to 480), minimized at first, its strip titled "Report"
(«Отчёт»). Its inner edge is its top edge; dragging it resizes the panel.
While the panel is restored the report docks in it; minimized, the report
floats as before. Report (R) opens the report and restores the panel;
pressed again it closes both. The editor profile keeps the panel's state
beside the side panels' (`"bottom":{"minimized":...,"width":...}`); a
missing, broken or unreadable profile gives the default. The one writer,
`panelStateBytes`, also writes the object window's view (`"view"` inside
`"right"`), for example
`{"bottom":{"minimized":true,"width":180},"left":{"minimized":false,"width":300},"right":{"minimized":true,"view":"plain","width":520}}`.

### With the property views

Base `ff56581b7` carries the object window's plain and technical views
(`docs/1565/story.md`). The merge keeps them whole: the `plainViews`
table, the view switch in the title row (`popupSpec.TitleButton`), the
hosted window's scroll (`clampObjectTop`, `hostedObject`) and
`popupRow.Fold`. The toolbar takes 28 pixels from the right panel's
height, so a window in the right panel shows fewer rows before a scroll;
the popup's scroll still reaches every field
(`TestMapEditorWindowReachesEveryField` passes unchanged).

### Kinds and builders

- Toolbar button: new kind, the table `toolButtons` and its one builder
  `toolbarEntries()` in `pkg/ui/mapeditortoolbar.go`. A press goes through
  `menuAct`; the drop button through `toggleMenu` and `menuDropdown`.
- Icon: new kind, `editorIcon` rows of data in `pkg/ui/mapeditoricons.go`,
  one painter `drawIcon` / `drawIconButton` in `pkg/ui/widgeticon.go`.
- Panel: the existing `sidePanels` table gains the bottom row; `showPanel`,
  `togglePanel`, `panelsLaid`, `resizePanel` and the profile's
  `panelStateBytes` build it. The strip height comes from the popup builder
  (`panelStrip`), so a minimized bottom strip is the builder's title row.
- Tips: `toolTip` beside `gridTipAt` in `observeTip`, drawn by `drawTip`.
- Menus: `menuItems` builds the Map menu and the Layers rows; the bar's
  entries stay `barEntries`.
- Words: `pkg/words/text/en/strings.toml` and `ru/strings.toml`. The View
  menu's grid row word `editor.view.grid` left with its row; the toolbar
  reads `editor.tool.grid`.
- Text lines: the bottom panel's empty note goes through `editorTextLines`
  (the editor row of the one line breaker).

No second builder of any kind was added.

## Proof

Base `ff56581b7` (engine main with the property views). All in
`pkg/ui/mapeditortoolbar_test.go`. On the tree before the toolbar
(749963004) the file does not compile (`e.currentTool undefined`); `go vet`
exits 1.

- (a) `TestMapEditorViewAndToolsRowsHaveOnePlaceEach`: the table above; each
  row key stands once at its place, the bar holds no View or Tools entry,
  and pressing the place does what `menuAct` did with the key.
- (b) `TestMapEditorToolbarRectsAtEachSizeAndPanelState`: 1280x800,
  1024x768 and 640x480, each side panel open and minimized; the toolbar,
  panels, strips and `mapArea` rectangles.
- (c) `TestMapEditorCurrentToolFollowsButtonKeyAndMCP`.
- (d) `TestMapEditorLayersDropDownTogglesEachSwitch`.
- (e) `TestMapEditorBottomPanelOpensResizesAndKeepsItsState`: F8, the
  strip button, Report; drags clamped to 96 and 480; the profile round
  trip; a missing, broken and unreadable file give minimized, 180.
- (f) `TestMapEditorGenerateAndAddDoWhatTheirRowsDid`.
- Also `TestMapEditorToolbarDisablesWhatTheMapRefuses` and
  `TestMapEditorToolbarTipsAndIcons`.

### Goldens

New cases of `TestMapEditorSidePanelsGolden`
(`pkg/ui/testdata/mapeditor_side_panels.golden`): `toolbar-en-1280x800`
and `toolbar-ru-1280x800` (terrain brush current), `layers-en-1280x800`,
`report-en-1280x800`, `nobrush-en-1280x800`. Each line gains the toolbar
text (keys, `*` on, `-` disabled, the open menu's rows).

Re-recorded: `pkg/ui/testdata/mapeditor_modes.golden` (every row: its
hash includes the panel rectangles), `mapeditor_side_panels.golden` (the
four existing cases: header and toolbar line),
`mapeditor_speaker_portraits.golden` (frame hash) and
`popupTranscriptHash` in `pkg/ui/mapeditorpopupstore_test.go` (2310
lines).

Re-recorded on the merge with `ff56581b7`, from the merged code:

- `mapeditor_modes.golden`: 2156 hash tokens differ from the toolbar
  candidate, at exactly the positions the property views moved against
  their base (the view switch in each unit, structure and sack window's
  title); none other.
- `mapeditor_side_panels.golden`: the pixel hashes of `search-en-1280x800`
  and `report-en-1280x800`, whose right panel hosts a unit window that now
  draws the view switch in its title; rows unchanged.
- `mapeditor_plain_views.golden`: every window rectangle moves by the
  toolbar, the right panel's top 16 to 44 and a floating window +28; the
  rows are unchanged.
- `mapeditor_technical_reach.golden`: in the right panel the person window
  shows fewer rows before a scroll (EN 360: 5 to 3; RU 360: 4 to 3; EN 520
  and 800: 10 to 9; RU 520 and 800 unchanged), as the toolbar shortens the panel by 28 pixels; floating
  windows and the button lines are unchanged.

### Witness

A dump of the editor frames at 1280x800 in five states from the base and
the candidate, compared outside the repository:

- The status strip: 0 differing pixels in every state.
- Map columns below the toolbar match the base rows 28 higher: 0 differ.
- The bar row differs only where the View and Tools entries stood and the
  entries after them.
- The side panels match shifted 28 rows, except the tree's scroll bar
  column (the panel is 28 rows shorter) and its last 28 rows.
- The popup transcript: x and width never change; every other change is a
  move of +28, the side panel top +28, the added bottom strip, or a clamp
  to bounds whose top moved from 16 to 44.
- The modes probes: of 9520 blocks, 862 differ with the panel lines set
  aside; in each the first difference follows a pointer gesture. No key
  action's own state differs.

The witness found one fault, fixed before the commit: a minimized bottom
strip of 16 rows drew the builder's 26-row title over the status strip.

## Open debt

- The `pkg/game` release tests that opened View or Tools are rewritten and
  not run in the lane: `mapeditbrush_release_test.go`,
  `mapeditorareaedit_release_test.go`, `mapeditorselection_release_test.go`,
  `mapeditpassability_release_test.go`, the comment of
  `mapeditwholemap_release_test.go`, and every release test that calls
  `wizNewTrigger`. The ordinary tests rewritten beside them
  (`mapeditorareaedit_test.go`, `mapeditortrigger_test.go`,
  `mapeditorwizard_test.go`) pass with `go test -skip '^TestRelease'
  ./pkg/game`.
- The Scripts palette opens over the map's top rows, and Add leaves it
  open. `wizNewTrigger` closes the chosen record's window and drags the
  palette to the left edge before it draws a trigger area; without that, a
  drag that starts on the palette's Add button opens Add again. A map maker
  drawing an area there moves the palette the same way.
- In the right panel the person window's identity, equipment and stats
  tabs show 1 to 2 fewer rows before a scroll than without the toolbar; at
  a 360-pixel panel in EN and RU the equipment tab shows no field before a scroll.
  Every field stays in reach by the popup's scroll.
- Icons are 16x16 in one size; a larger font does not scale them.
