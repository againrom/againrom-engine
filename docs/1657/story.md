# A mod's missions are one list everywhere

## Result

The starter now lists the missions a mod replaces with a map named by its number, as "10. replaced by the map 10.alm";
before, such a mod said it had no missions.
The map editor's open, its campaign graph and its mod pages read the same mission list as the starter.
The Open window and the Maps page break a map's roles at each role, not at each comma of a title.
A mod's target base now takes rom2 for every second-game install, and the editor offers it.

## Authority

- Owner: «Все четыре по порядку» (the four parts of item R3 of the mod tools audit, in order).

## Rules

1. The starter lists a number-named map no row names only if the install plays it; with no install, the rows alone.
2. When the starter starts such a mission, it passes that mission number.
3. When the editor opens a mod, its open, graph draft and mod pages hold one mission set, the install's answer applied.
4. When a map has several roles, the Open window and the Maps page list them one a line.
5. When a key is added to the manifest key table, the editor's mod page edits it with no editor change.
6. When the editor offers a target base, it offers both games' bases from the one base table, rom2 among them.
7. When a mod names rom2, it applies to both second-game installs and to no first-game install.

## Code

| kind | its one builder | change |
|---|---|---|
| mod mission set | `mod.DescribeMissions`, `MissionData.Listed` | holds number-named maps; the game, editor and starter pass `campaignset` as its test |
| campaign mission answer | `campaignset.Numbers.Check` | the one test of a mission of the install; `ReadCampaign` takes its numbers from it |
| editor mission set | `MapInspector.modMissions` | open, graph draft, fields and Maps page ask it |
| editor campaign flow | `newCampaignFlow`, `forkFlow` | the only `campaignFlow` literal |
| starter mission row | `cmd/starter/app.go` `missionLabel` | a by-name row from the caption table |
| map role lines | `ui.InspectionField.ChoiceLines`, `choiceLines` | roles as a list, not a split label |
| manifest keys | `mod.ManifestKeys` | reader, writer and editor fields read the table |
| target bases | `base.ModBases`, `Edition.ModFamily` | the editor's choices; `rom2` family word |

## Proof

| test | rules |
|---|---|
| `TestMissionListShowsTheMissionsAModReplacesByFileName` (EN, RU; red on the base) | 1, 2 |
| `TestDescribeMissionsHoldsTheMapsThatReplaceMissionsByName` | 1, 3 |
| `TestStarterListsAGuestRowBesideAStrayNumberedFile`, `TestStarterListsNoNumberedFileTheCampaignLacks`, `TestStarterWithNoInstallListsRowsAndNoFile` (red on the base) | 1 |
| `TestEditorOpenDraftAndFieldsShareOneMissionSet` (EN, RU) | 3 |
| `TestModMapRolesAreAListNotASplitLabel`, `TestModWindow.*`, `TestOpenWindowLeadsWithTheMapsOfTheMod` | 4 |
| `TestAManifestKeyAddedToTheTableIsAnEditorField`, `TestManifestKeysAreOneTable` | 5 |
| `TestEditorOffersTheBasesOfBothGames`, `TestModProject.*` | 6 |
| `TestEveryBaseOfTheTableIsAdmitted` | 7 |
| `TestModMissionSetHasOneBuilder` (`internal/archtest`) | 3 |
| `TestEditorMod.*`, `TestCampaignGraph.*`, `TestMapList.*`, `TestModReplace.*`, `TestMissionMaps?.*` | 1, 3 |
| `TestMapEditorModesHoldEveryKeyAndGesture` (full walk) | 4, 6 |

Evidence and the gate lines: the review report of this story. Release tests at the release: none named.

## Open debt

- The editor asks the install's archives, not the files a mod serves to the install; a mod serving its own campaign map
  could be listed differently by the editor and the game.
- With no install open, the editor lists every number-named map as a replacement.
