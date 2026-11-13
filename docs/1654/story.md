# A mod sets a town's hand-outs and what opens a talk

## Result

A mod of the first game now says which mission a town's shop and school hand out in a chapter,
as it already says for the tavern's talks. A talk or a hand-out can wait until named missions are won,
and can stand from a chapter on instead of in one chapter.
A waiting talk appears in the tavern after the win, also after a save and load in the town.

In the map editor the Campaign graph edits these hand-outs of an open mod: a click selects a town or a
hand-out, its fields change, add, switch off or remove a row, each edit is one undo step, and Ctrl+S
writes the mod's towns file keeping its comments and untouched rows.

## Authority

- Owner: a mod sets which tavern NPC, shop or school hands out which mission, and what opens a talk.
- Owner: the first conditions are "mission X is won" and "the chapter is at least N"; the first game first.
- Owner: the graph edits hand-outs; the missions row route is generalized by data, not copied.
- REG-SCN-062, REG-SCN-064: the three offer lists, one reader each, no condition in the original.
- DIV-4925: a waiting or standing row; DIV-4926: shop and school rows of a mod.

## Rules

1. When a mod names a shop or school row, that building hands the mission out in that chapter of that town.
2. When a mod switches a shop or school row off, the building no longer hands that mission out.
3. When a row names won missions, its building shows it only after every one of them is won.
4. When a row stands from a chapter on, it shows in that chapter and each later one until its mission is won.
5. When the town is saved and loaded, a waiting row shows or waits exactly as before the save.
6. When a row has neither new key, the tavern, the shop and the school behave as before.
7. When a condition names a mission the campaign does not hold, the mod is refused with file, line and row.
8. When a hand-out of a hidden row is asked for by its index, the town gives nothing.
9. When a graph edit of a hand-out is accepted, it is one undo step and the graph shows the draft with its mark.
10. When a graph edit breaks the towns check, it is refused with the check's words and the draft stays.
11. When the mod is saved, `data/towns.toml` equals the hand-made file of the same rows and keeps its comments.
12. When an install row is changed or switched off, the mod writes an `off` row for it, never deletes it.
13. When a shop or school row gives a mission its building already gives in a shared chapter, the mod is refused.

## Code

| kind | its one builder | change |
|---|---|---|
| talk and hand-out row | `town.OfferRow`, `Catalog.validateOffers` | place, floor and won list on the one row type |
| row tables | `mod.ParseTowns`, `offerTables`, `mod.WriteTowns` | `[[shop]]`, `[[school]]`, `from`, `won` as table data |
| source row writer | `mod.writeRows` (was `WriteMissions`' body) | rows as data; `WriteMissions`, `WriteOfferRows` call it |
| whether a row shows | `innState.admits` | floor and won missions; `Town.Offers` asks it with `Town.Done` |
| chapter lists, layer 0 | `Campaign.chapter`, `innOffers`, `townBook.baseOffers` | shop and school rows; layer 0 shared with the editor |
| the check | `offerRules.check` through `missionDraft` | won missions checked; the editor's draft asks the same check |
| editor draft and save | `editorModDocument`, `changeWorkspace`, `retainedPackageFiles` | towns records beside the missions |
| graph page | `campaignGraphOf`, `popupSpec.Canvas` | node pick, selection, added and changed marks; scope `handout` |

## Proof

| test | rules |
|---|---|
| `TestAConditionedTavernTalkWaitsForAWinThroughSaveAndLoad` (EN and RU base) | 3, 5 |
| `TestShopAndSchoolRowsAndTheChapterFloor` | 1, 2, 3, 4, 8 |
| `TestHandoutRowRefusals`, `TestCampaignGraphHandoutRefusesAMissionTwiceInOnePlace` (EN, RU) | 7, 10, 13 |
| `TestScenarioShopAndSchoolAreLayerZeroRows`, `TestTavernLayerLeavesTheScenarioChapters` | 6 |
| `TestOfferTablesReadTheFloorTheWonListAndTheBuildings`, `TestOfferTableRefusalsOfTheNewKeys` | 1, 3, 4, 7 |
| `TestWriteTownsWritesEachRowUnderItsTable`, `TestTavernLayerAddsChangesAndSwitchesOff` | 6 |
| `TestOfferRowsKeepTheSourceAndChangeOnlyTheEditedValues`, `TestRowWriterRemovingAMiddleRowKeepsTheNextRowsComment` | 11, 12 |
| `TestCampaignGraphHandoutEditsAreDraftsWithOneUndo`, `TestCampaignGraphHandoutPlaceChangeOfAModRow` (EN, RU) | 9, 11, 12 |
| `TestCampaignGraphHandoutsSaveAsTheHandMadeModAndReopen` (EN, RU) | 9, 11, 12 |
| `TestCampaignGraphHandoutRefusalKeepsTheDraft`, `TestCampaignGraphHandoutEditKeepsTheModsSource` | 10, 11 |
| `TestCampaignGraphHandoutNodesSelectThroughTheHost` (EN, RU), `TestMapEditorModesHoldEveryKeyAndGesture` | 9, 10 |

Release tests at the release: `TestReleaseTavernModRowsSaveAndLoad`, `TestReleaseCampaignGraphEqualsTheHandoutCensus`
and the base dump witness, on EN and RU; none ran in this lane.

## Open debt

- The graph draws a waiting row as any other; it states neither the floor nor the won list on the node.
- A talk's text is not written from the graph; its field states which file holds the line.
- A row that stands from a chapter on gets a fresh side record in each later chapter, also after its win.
- The second game's inn rows take no won list; the graph's hand-out edits are proven on first-game fixtures only.
- The base dump now writes shop and school tables; its release witness was not run here.
