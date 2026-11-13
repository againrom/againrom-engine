# One mods folder, and a mod's mission started from the starter

## Result

The starter, the game and the map editor now use one mods folder. A mod maker saves a new mod
with no path typed: the map editor proposes a working folder in the player profile and publishes
the packed mod into the mods folder. The starter sees the published mod when its window comes
back to the front, with the ticks and the selection kept. A ticked mod shows its missions by
number and title, and Start mission opens the game on the chosen one; both start buttons store the list.

## Authority

- Owner: «one mods folder, and a mod's mission started from the starter» (story brief).
- Owner: the starters of both games behave the same; the second game's editor is out of scope.

## Rules

1. The starter's settings, its mod scan, the game and the map editor resolve one mods folder.
2. A game or editor started without the starter resolves that folder from the starter's file beside it.
3. The Map editor button starts the editor with the starter's mods folder.
4. A mod with no folder shows the proposed working folder, workbench/id in the player profile.
5. Save with no folder typed saves there; a folder already there is refused and nothing is written.
6. Publish with no path typed writes the packed mod into the mods folder; a path can still be typed.
7. Publish refuses: no complete copy, map not a mission of the mod, mod changed outside the editor.
8. Publish refuses: folder in an install, folder in the working mod, same-id folder there, no folder known.
9. Publish asks before saving unsaved changes and before replacing a packed mod.
10. The starter rescans when its window regains focus; ticks, order and selection stay.
11. A ticked mod lists its missions by number and by title in the starter's language.
12. A mod not ticked, a mod with no missions and an unreadable missions file each say so, on both games.
13. Start mission passes the enabled mods and the chosen mission; Play passes no listed mission.
14. Play and Start mission write the settings first; a refused start writes nothing.
15. The Rescan button and the typed mission number stay.
16. The starter window stays 760 pixels high; its buttons and status line keep their place.
17. A refused or unanswered Publish makes no folder; a chosen mission never moves to another mod.
18. An id Windows keeps no folder for proposes no working folder.

## Code

| kind | its one builder | change |
|---|---|---|
| mods folder | `pkg/launch/starterini.go` `ModsFolder` | new; `mod.DefaultDir` and its 3 callers removed |
| mission description | `pkg/mod/describemissions.go` `DescribeMissions` | new; the editor's inline reader removed |
| starter list | `cmd/starter/app.go` `scrollLists` | fourth list: missions, under the status line |
| starter.ini writer | `cmd/starter/app.go` `writeSettings` | Save, Play and Start mission call it |
| game start | `cmd/starter/app.go` `start` | takes the mission; `play` is `start("")` |
| launch rows | `pkg/launch/settings.go` `Editor` | row `EditorModsDir` |
| profile folder | `pkg/game/mapeditorworkbench.go` `profileFolder` | maps and workbench share it |
| Publish folder | `pkg/game/mapeditorworkbench.go` `publishFolder` | typed, else resolved |

## Proof

| test | rules |
|---|---|
| `pkg/launch` `TestModsFolderHasOneResolver` | 1, 2 |
| `pkg/editorapp` `TestModsFolderIsTheOneTheStarterAndTheGameRead` | 1, 2, 3 |
| `TestEditorArgumentsNameTheAssetRootAndTheModsFolder` | 3 |
| `TestModWindowProposesTheWorkbenchAndPublishesIntoTheModsFolder`, `TestModWindowSaveRefusesATakenProposedFolder` | 4, 5, 6, 9 |
| `TestModWindowPublishRefusalsStandWithNoPathTyped` | 7, 8, 9, 17 |
| `TestRegainingFocusListsAModPublishedMeanwhile` | 10, 13, 15 |
| `TestMissionListShowsTheMissionsOfTheEnabledMod`, `TestDescribeMissionsReadsAFolderAndAPackedMod` | 11 |
| `TestMissionListAsksToEnableTheMod`, `TestMissionListReportsAMissionsFileItCannotRead` | 12 |
| `TestSecondGameBaseShowsAModWithNoMissions`, `TestModWindowProposesNoFolderWindowsCannotKeep` | 12, 18 |
| `TestTheWindowFitsAScreen768High`, `TestAChosenMissionDoesNotMoveToAnotherMod` | 16, 17 |
| `TestStartNamesTheChosenMissionAndPlayNamesNone` | 13, 14, 15 |
| `TestARefusedStartWritesNoSettings` | 14 |

Release tests at the release: `TestReleaseEditorPublishedModChainSAV`,
`TestReleasePackedModPlaysAsItsFolder`. Both are extended to the resolved folder and the missions
the starter lists, compile, and were not run in the lane. No editor modes row or key changed.

## Open debt

- Programs inside an install: the proposed workbench and the mods folder beside them fall under
  the install fence, so Save and Publish refuse there until the profile grant reaches them.
- The editor frame needs the install's font; the fixture frame shows the window without letters.
- While a mod is selected the Check output shows two lines, not six; the mission list shows three rows.
- Play writes the settings read when the starter opened over a starter file edited by hand meanwhile.
- The first frame of the window scans the mods folder a second time.
- An editor started alone does not know a starter file kept in another folder.
