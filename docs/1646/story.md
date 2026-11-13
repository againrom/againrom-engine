# Several maps of one mod open at once in the map editor, as tabs

## Result

With a mod open, the map editor keeps several of its maps open at once. Opening a second map adds a strip of
tabs under the menu bar, one tab per open map with its mission number and name, and a click on a tab shows that
map. Each map keeps its own view, selection, unsaved changes and undo history while another one is shown, and
a star on its tab marks unsaved changes. Save writes the map shown; closing a tab or leaving the mod asks the
usual unsaved-changes question about each changed map. With one map and no mod the editor looks as before.

## Authority

- Owner: «In the editor, allow opening many maps within one mod; for this we show them as tabs somewhere at the
  top, so one can switch between them.»

## Rules

1. When a mod is open, a map opened from the mission list, the graph, the map list or Open opens beside the open maps.
2. When two or more maps are open, a strip under the menu bar holds one tab per map: mission number and name.
3. When a map has unsaved changes of its own, its tab ends in a star.
4. When a tab is clicked, its map is shown with the view, selection, brush, changes and undo history it was left with.
5. When a map that is already open is opened again, its tab is shown; no map has two tabs, in any folder of the mod.
6. When 8 maps are open, a ninth is refused with a message that says to close one; it opens once a tab is closed.
7. When Save is pressed, the map shown and the mod's rows are written; no other open map's file changes.
8. When the x of the strip is pressed, the map shown closes; with unsaved changes of its own the question asks first.
9. When the editor is closed or another mod is opened, the question asks about each changed map; no old tab stays.
10. When another tab is shown, W, the graph and test play read the one mod with every open map's unsaved draft.
11. An undo step made before another tab was shown restores its map's own part; a removed mission comes back whole.
12. When one map is open, or no mod, there is no strip and the window is drawn as before.
13. The tabs have no keys: no free pair fits the key table. An agent lists the open maps and shows one by its place.

## Code

| kind | its one builder | change |
|---|---|---|
| tab strip | `pkg/ui/editorpopup.go` tabStrip, popupLayout.hit | W's tabs and the map tabs lay out, hit and draw through it |
| open map document | `pkg/ui/mapeditor.go` openMap, showDoc | a second map is added beside the first; showDoc restores a kept map |
| mode layers | `pkg/ui/mapeditormode.go` keepMode, returnMode | the brush a map is left with |
| unsaved-changes question | `pkg/ui/mapeditorleave.go` askLeave, runLeave | a queue of changed maps; a close-map step |
| mod project | `pkg/game/mapeditoropenmaps.go` Shown, carry | one project, carried to the edits of the map shown |
| mod save | `pkg/game/mapeditorpackage.go` saveWorkspacePackage | other open maps' drafts held out and returned |
| undo history | `pkg/game/mapinspectionworkspace.go` travelWorkspace | a step recorded before a carry restores the map's own part |
| agent tools, control finder | `pkg/ui/mapeditormcp.go` editorTools, `controlfinder.go` editorControlRows | rows `open_maps`, `show_map`, `ControlMapTab` |

## Proof

| test | rules |
|---|---|
| `TestEditorKeepsSeveralMapsOfOneModOpen` (EN, RU) | 1, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12 |
| `TestEditorOpenMapsStopAtTheLimit`, `TestAnExtendedMapSavesIntoAModAsAlmx` | 6, 9 |
| `TestEditorMCPListsAndShowsOpenMaps` | 3, 13 |
| `TestEditorModMissionOrderAndHistorySurviveMapSwitches` | 1, 5, 11 |
| `TestEditorUndoOfARemovedMissionAfterATabSwitch` | 11 |
| `TestEditorOpenMapsFollowTheModToAnotherFolder`, `TestEditorOpenMapsFollowAnImportedModsCopy` | 5, 7 |
| `TestMapEditorModesHoldTheTabsRows` | 2, 13 |
| `TestMapEditorModesHoldEveryKeyAndGesture`, `TestMapEditorSidePanelsGolden` | 12 |

Release tests at the release: none.

## Open debt

- A map opened while another mod's map is new and unsaved still replaces it: New keeps the one-map route.
- Save in a mod writes the mod's rows and manifest with the map shown; a save of the map alone waits for the owner.
- The tabs have no keys; a pair waits for an owner choice of keys.
