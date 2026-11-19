# The mod window on a second-game install with no map

## Result

With a second-game install and no map open, the editor opens a mod's window: frame, tabs, overview, files and save
state. The pages that need no map edit and save the mod. The mod's missions are the second game's: the missions its
archive holds maps of, named by the map's name where the map has one and by number otherwise. The first game's mod
window behaves as before.

## Authority

- Owner: «Еще задача это редактор rom2»; «Как рекомендуешь» on mod workbench story 17. One builder per kind, the first
  game as profile data, no second editor.

## Rules

1. The mod window opens on a second-game install with no map and builds the mod's context without a scenario registry.
2. A text and an item record of a mod edit and save there, and the saved mod reloads equal.
3. A replace button names a second-game mission by its map's name, else by number; no first-game name appears.
4. The first game's mod window, mission list and save behave as before.

## Code

| kind | its one builder | change |
|---|---|---|
| install campaign | `MapInspector.installCampaign` | the registry's, else the missions the archive holds maps of |
| mod context | `admittedFork` | reads the campaign through `installCampaign` |
| mission list | `installFlow` | reads the campaign through `installCampaign` |
| mission button | `replaceCaption` | names a second-game mission by its map |
| package save | `saveWorkspacePackage` | refuses a map-writing save only |
| labels file | `labelsForSave` | no map, no labels |

The Towns page has no second-game catalog and the Dialogues page has no map, so neither is offered.

## Proof

| test | rules |
|---|---|
| `TestReleaseModWindowEditsAndSavesWithNoMap` | 1, 2, 4 |
| `TestReleaseSecondGameModListsTheInstallMissions` | 3 |

Release tests at the release: both second-game installs and the first-game install.

## Open debt

- Guest and test play stay first-game; the hero and item suit lists stay with the editor's second-game story.
- The Towns page and the campaign graph await the second game's town catalog.
