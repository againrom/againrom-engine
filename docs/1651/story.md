# The mod window lists every map of the mod, and the Open window leads with them

## Result

With a mod open, the mod window has a Maps page that lists every map of the mod once and says what the mod
does with it: which missions it is, that its file name replaces a mission, that nothing uses it, or that its
file is missing. A map opens from that page as a tab, and the Open window shows the same list as its first
group. Ctrl+Tab and Ctrl+Shift+Tab turn the open maps. The mod check warns about a map no mission uses.

## Authority

- Owner: the mod's map list, the Open window's mod group and adding maps are one story; the tab keys are
  Ctrl+Tab and Ctrl+Shift+Tab; files on old paths are not moved on save; the check report stays English.
- Decided here: the list is W's own page, not rows of the first page, which has no room at 640x480.
- Decided here: a map opened from the list takes the row that names it, the chosen row when several do.
- Decided here: the second game's editor gets the same page, group and keys; nothing branches on the game.

## Rules

1. One list joins each map file of a mod to the rows that name it; the editor and the check both read it.
2. The check warns of a map no mission names; a mission whose map file is absent stays a refusal.
3. When a mod is open, W's last page, Maps, has one row per map with its role in the editor's language.
4. A map two rows name is one row with both roles; at 640x480 a long role goes on under the row, whole.
5. The rows scroll; Tab and the arrows walk W's buttons and then each map's Open; Enter there opens the map.
6. Open on a row opens its map as a tab through the switch the mission choice uses; a missing map refuses.
7. A map no row names keeps its draft and its Undo across a switch to a map with a row and back.
8. Ctrl+S on a map no row names writes that map at its own path and changes no mission row.
9. F5 on a map no row names refuses, in the editor's language, and says to make it a mission first.
10. While a mod is open, the Open window's first group is the mod's maps; a choice opens a tab in the mod.
11. Ctrl+Tab shows the next open map and Ctrl+Shift+Tab the previous, wrapping; W stays open if it was.
12. The mission rows of W's first page, W's other pages and the editor with no mod stay as they were.
13. A file named by a mission number replaces that mission only where the loader kept it; else it is unused.
14. When a save makes the editor's copy of the mod, the tab of a map no row names holds the copy's file.

## Code

| kind | its one builder | change |
|---|---|---|
| a mod's map list | `pkg/mod/maplist.go` MapList | new: files joined to rows, roles, MapWarnings |
| mod check warnings | `pkg/modrt/modrt.go` Load | adds MapWarnings of the one list |
| map switch in a mod | `pkg/game/mapeditoropenmaps.go` openModMap | openMissionMap generalised: a row or a map no row names |
| mod project document | `pkg/game/mapeditormod.go` editorModDocument | Loose: the path of a map no row names; save writes it there |
| W pages | `pkg/ui/mapeditorworkspace.go` modPages | fourth page Maps, rows by modMapRows over the host's list |
| Open window rows | `pkg/ui/mapeditorwindow.go` catalogRows | first group from the same host list, opened by OpenModMap |
| key table | `pkg/ui/mapeditorkeys.go` editorKeyRows | rows nextmap, prevmap with their help line |

## Proof

| test | rules |
|---|---|
| `TestMapListJoinsEveryMapToItsDeclarations`, `TestMapListOfAModWithoutMaps` | 1 |
| `TestMapListTakesTheFileFormFromTheLoader` | 13 |
| `TestCopyOfAModKeepsTheUnusedMapsTab` | 14 |
| `TestCheckWarnsOfUnusedMaps` | 1, 2 |
| `TestEditorListsAndOpensEveryMapOfTheMod` (EN, RU) | 1, 3, 6, 7, 8, 9, 10, 11 |
| `TestEditorModMapListDrawsInEachLanguage` (EN, RU; 640x480, 1280x800) | 3, 4, 10 |
| `TestModWindowMapsPageListsEveryMapOfTheMod` (EN, RU; 640x480, 1280x800) | 3, 4, 5, 6 |
| `TestOpenWindowLeadsWithTheMapsOfTheMod` | 4, 10 |
| `TestModWindowCtrlTabIsNotTheFieldTab` | 11 |
| `TestModWindowMissionRowsHaveNoDropdownInBothLanguages`, `TestModWindowTurnsToTheCampaignGraph` | 12 |
| `TestMapEditorModesHoldEveryKeyAndGesture` (full walk, rows `mod-maps`, `mod-maps-row`) | 5, 11, 12 |

Release tests at the release: `TestReleaseMapEditorOwnerFrames` gained the frame `mod-maps-small`; not run here.

## Open debt

- Make mission on an unused map and Make row on a by-name map are not built; they wait for the next story.
- Add map with a stem and Add maps from folder are not built; they wait for the next story.
- F5 on an unused map refuses instead of playing as the test mod does; it waits for the next story.
- No second-game fixture ran these tests; the second game's editor is covered by the shared code only.
