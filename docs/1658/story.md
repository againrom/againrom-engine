# A mod maker turns unused maps into missions and adds maps without writing a file

## Result

The mod window's Missions page now lists each map no mission uses, with Make mission and a Replace button, and each map
that replaces a mission by its file name, with Make row. The open map can be added to the mod as an unused map, and a
whole folder of maps can be added at once through the system folder window. F5 on an unused map plays it with the mod
active. Each action is one Undo step and Ctrl+S writes it.

## Authority

- Owner: «Отдельной историей»; «можно тем же роем»: items 6 to 8 of story 1651 are this story.
- Owner («да»): the unused-map actions stand on the Missions page; the Maps tab stays gone.
- Owner: «Открыть мод должен быть нативным пикером»: a folder is asked of the system window first.
- Decided here: the buttons read Add unused (the open map) and Add maps from folder, because Add map is the existing
  button that makes the open map a mission.
- Decided here: Replace offers the lowest campaign number the install holds that no row of the mod replaces; the
  mission's number field changes it.
- Decided here: a map file named by digits takes the prefix m, so a copy never becomes a replacement by its name.
- Decided here: nothing branches on the game; the second game's editor gets the actions, and Make mission as a guest
  takes the campaign's guest slot, which an install without one refuses in the editor's words.

## Rules

1. Make mission on an unused map writes a guest row of the editor's default guest, or a replacement of a held number.
2. A number the install's campaign lacks, or one a row already replaces, is refused and leaves no draft.
3. Make row on a map named by its mission number writes the row naming the same map and number.
4. After either, the page lists the row among the missions and no longer lists the map as unused.
5. Add unused copies the open map, its pages and its labels to maps/stem with -2, -3 on a taken stem; no row names it.
6. Add maps from folder adds each map of the folder with its pages and labels as one Undo step.
7. The folder is asked of the system window; a typed path is asked only where the system has none.
8. F5 on an unused map opens the party window and plays the map as a guest mission with the mod active; no row is kept.
9. Every refusal speaks the editor's words in English and Russian.

## Code

| kind | its one builder | change |
|---|---|---|
| mod's map list | `pkg/mod/maplist.go` MapList | read by the new actions |
| mission row of an unused map | `pkg/mod/mapmake.go` MakeMission, MakeRow | new; the guest is `guestRow` |
| guest row default | `pkg/game/mapeditormodmakemap.go` guestRow | Add and Make mission both call it |
| file place of a map | `pkg/mod/missionplace.go` MissionPlace.Free | folder maps placed by it |
| campaign numbers | `pkg/game/mapeditormodcontext.go` missionAnswer | Replace and refusals ask it |
| Missions page rows | `pkg/ui/mapeditormod.go` modSpec | rows of the host's map actions |
| folder choice | `pkg/ui/mapeditormod.go` askMapsFolder | system window, typed prompt fallback |
| test play | `pkg/game/mapedittestplay.go` testPlayWith | an unused map gets a guest row in the test copy |

## Proof

| test | rules |
|---|---|
| `TestMakeMissionOfAnUnusedMapGuestAndReplace`, `TestMakeMissionReplacesACampaignNumberTheInstallHolds` (EN, RU) | 1, 4 |
| `TestMakeRowNamesTheMapThatReplacesByFileName` | 3, 4 |
| `TestMapActionsRefuseInTheEditorsWordsAndLeaveNoDraft` (EN, RU) | 2, 9 |
| `TestAddMapCopiesTheOpenMapUnderItsOwnStem` | 5 |
| `TestAddMapsFromFolderAddsSevenMapsInOneUndoStep` | 5, 6 |
| `TestAddMapsFromFolderAsksTheSystemFolderWindow`, `TestAddMapsFromFolderFallsBackToATypedFolder` | 7 |
| `TestTestPlayOfAnUnusedMapStartsItWithTheModActive` | 8 |
| `TestModWindowMissionsPageOffersTheMapActions` (EN, RU; 640 by 480, 1280 by 800) | 1, 3, 4 |
| `TestMakeMissionGuestIsTheEditorsDefaultGuestNamingTheMap`, `TestPlaceFolderMapsTakeTheNextSuffixAgainstTheModsFiles` | 1, 5, 6 |

Release tests at the release: none.

## Open debt

- The Missions page's map buttons answer the mouse only; a key route waits for an owner ruling.
