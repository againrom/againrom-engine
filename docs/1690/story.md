# Map editor: the mod window's pages on the one window framework

## Result

Every page of the mod window is one width, its tabs on one line, in English and Russian. Pages of things list them
left and show the chosen one right with its actions on its name line. Removing a mission and adding maps happen on the
Missions page; the window's own buttons are Open, Save, Publish and Close. The overview counts what each page changes
in one line that opens the page, and folds the game's own check under its details. The working and publish folders
are chosen in the system folder window, and a long text stands on one line with its whole text in its tip.

## Authority

- Owner: the T7 mockup answers (board cards o-t7-q1 to o-t7-q10): eight took (a), questions 5 and 10 took (b);
  answer 5 keeps the needed, incompatible and load-after mods as typed IDs, as before this story.
- Owner (RULEBOOK Q6): «Всегда системное»: folders are chosen in the system window; answer 9 keeps the typed
  prompt where no window exists. Owner: the editor targets 1920x1080 and above.
- Seat reading of «the notes hold only the save state»: a refusal or error stands in the editor's message line.

## Rules

1. Every page of the mod window has the width its tab row needs to keep the 8 captions on one line.
2. A page of things lists its adders and things left; a press on a list row shows that thing right, actions on its line.
3. Remove stands beside the mission, Add map on the map rows; the button row is Open, Save, Publish and Close.
4. The campaign chain line stands in the Mission block with Move up and Move down.
5. The overview has one count line per page that opens it; the game's check lines fold under Check details.
6. A terrain grid cell is 44 pixels and names its ground in its tip.
7. With the system window the working and publish folders are chosen by Choose and take no typing.
8. With no system window a folder or mod is asked in the typed prompt, its instruction inside the prompt.
9. A refusal or error stands in the editor's message line; the window's notes hold only the save state.
10. A long text is one line, cut, with its whole text in its tip.
11. The Texts page says what it is for behind the title's (?) and carries Delete on each language's row.
12. At 1920x1080 in English and Russian each page stands inside the screen and each part takes its press.
13. Every caption, note, tip and button comes from the words table in Russian and English.
14. What each page edits and what a mod saves stay as before: joins, saves, publishes, spells drag and drop.

## Code

| kind | its one builder | change |
|---|---|---|
| mod window frame and pages | `modSpec`, `objectFieldRows`, `modListRows` | pages are data; one width |
| row buttons on a field's row | `modRowButtons` | data: `InspectionField.Of` |
| path questions | `modPathRows`, `modFolderQuestion`, `promptSpecBase` | folder Choose rows, typed fallback |
| folder field input | `inputTarget` | `modChosenOnly` refuses typing while the host has a window |
| overview | `modOverview`, `modPageCounts`, `modOverviewRows` | count lines with page links, folded check |
| one-line text rows | `popupFlowOf` | a text row with a tip keeps its buttons on its line |
| words | `pkg/words/text/*/strings.toml` | new words in both languages, each read; four unused dropped |

## Proof

| test | rules |
|---|---|
| `TestModWindowKeepsItsTabsOnOneRowAtOneWidth` | 1 |
| `TestModItemsPageListsItsThingsAndShowsTheChosenOne`, `TestOverviewCountsTheItemsThePageLists`, `TestItemsPageListsTwoRecordsOfOneGameItemAsTwoRows` | 2 |
| `TestModMissionsPageCarriesRemoveAndAddMap`, `TestModWindowWrappedButtonsStayInsideTheEditor` | 3 |
| `TestModMissionBlockCarriesTheChainLine` | 4 |
| `TestOverviewCountsPagesAndFoldsTheGamesCheck`, `TestModPagesAnswerPressesThroughTheInputPath` | 5, 2 |
| `TestModTerrainGridWrapsGroundsWithVariationTips` | 6 |
| `TestModPublishFolderRowChoosesInTheFolderWindow`, `TestEditorPathRowsAskAsOnBase` | 7, 8 |
| `TestModWindowOpenWithoutAPathWindowAsksForTheTypedPath`, `TestCampaignGraphCanvasRefusalKeepsTheSourceAndWords` | 8, 9 |
| `TestModMissionRowShowsALongNameWholeInItsTip`, `TestModPagesHaveTheirLayoutPropertiesOnTheTargetScreen` | 10, 12 |
| `TestTextsPageMarksATextMissingInOneLanguage`, `TestModPagesSayWhatTheyAreForBehindTheTitleMark` | 11 |
| `TestModWindowWordsAreRussianInRussian`, `pkg/words` tests | 13 |
| `TestSpellsPage*`, `TestEditorPublish*`, `TestEditorOpensEveryFixtureMod`, `TestFilesPageModSavesPublishesAndServes` | 14 |

Release tests at the release: none. Review report: `review/editor-ui/t7/story/RESULT.md`.

## Open debt

- Row buttons on a row (Choose, Delete, chain moves, list heads) are reached by the pointer only; Tab skips them.
