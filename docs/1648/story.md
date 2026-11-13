# Edit a mod's campaign from its graph

## Result

A map maker can choose which mission follows a win, set its party, and insert a mission from the campaign graph.
The graph shows each accepted edit immediately, and one Undo restores it.
Inserting a mission uses a map already held by the mod and keeps the previous automatic successor.
Save writes the project; a campaign shown without an open mod stays read-only.

## Authority

- Owner: the graph edits mission flow and parties through the existing project writer and mission row editor.
- Owner: consolidate the campaign successor decision first, preserving both games' behaviour.

## Rules

1. A win retains its successor, town, announcement, record, guest and screen behaviour in both games.
2. With a first-game mod open, graph selection edits automatic successors and the existing mission party fields.
3. Inserting a mission uses a held map, links it after the selected mission, and retains its automatic successor.
4. Each accepted edit is one workspace Undo step; the graph reads the draft immediately and Ctrl+S alone writes it.
5. The ordinary parser and game admission refuse invalid edits before publication with their own messages.
6. Without an editable mod the graph stays read-only; Open maps retains the existing mission map routes.
7. A mission its map file names (`20.alm`) takes the same edits from the graph and the Missions page and becomes a row.
8. A campaign field that cannot change shows as a line; the kind choice keeps each kind's route in the draft.

## Code

| kind | its one builder | change |
|---|---|---|
| win successor and screen | `Campaign.after` | record, offer, announcement and applied screen decision |
| campaign graph | `campaignGraphOf`, `missionFiles` | the admitted draft supplies the Campaign value; draft and mission check read mission files by one reader |
| mission fields and controls | `modFields`, `editPackageField`, `editModControl` | graph and Missions share them |
| mission rows | `mod.WriteMissions`, `mod.ParseMissions` | source-preserving draft, no format key added |
| project save | `writeEditorProject`, stage/install | row-only projects use the same publication route |
| history | `changeWorkspace`, `editorWorkspaceSnapshot` | one edit, including an insertion and its source link |
| canvas and hit test | `popupSpec.Canvas` | mission selection and target click are box hits |
| words | `pkg/words` | EN and RU controls and refusals |

## Proof

| test | rules |
|---|---|
| `TestCampaignWinStoresKeepTheirRoutes`, `TestWinRouteRecordIsTheBaseRecord`, `TestARefusedSuccessorNamesBothMissionsAndCarriesTheParty` | 1 |
| `TestCampaignGraphCanvasEditsUseWorkspaceFields` | 2, 3, 4 |
| `TestCampaignMissionRowEditsShareWorkspaceHistory` | 2, 4 |
| `TestCampaignInsertedDraftUsesTheSessionWinRoute` | 1, 3, 4 |
| `TestCampaignGraphCanvasRefusalKeepsTheSourceAndWords` | 5 |
| `TestCampaignGraphWithoutAModStaysReadOnly`, `TestCampaignGraphNodeOpensItsMissionsMap` | 6 |
| `TestCampaignFieldsThatCannotChangeAreLines`, `TestTheKindChoiceKeepsEachKindsRoute` | 6, 8 |
| `TestCampaignGraphEditsAreDraftsWithOneUndo`, `TestCampaignGraphEditsAModOfNumberNamedMaps` | 2, 3, 4, 7 |
| `TestCampaignGraphSavesReopensAndPlaysTheInsertedMission` | 1, 3, 4 |
| `TestCampaignGraphEachFieldSavesAndReopens`, `TestMissionsPageEditsANumberNamedMission` | 2, 4, 7 |
| `TestCampaignGraphRefusesWithTheGameWordsAndNoDraft`, `TestFollowsAtOnceRefusesANonNumberInWords` | 5 |
| `TestMapEditorModesHoldEveryKeyAndGesture` | 2, 4, 6 |

Release witnesses: `TestReleaseEditorPublishedModChainSAV`, `TestReleaseMapEditorOwnerFrames`,
`TestReleaseEditorInteractionFrames`, on EN and RU at the release.

## Open debt

- The 10 -> 11 -> 20 witness starts from a mod that holds mission 10's row and map; the graph adds mission 11.
- Town hand-outs, talk conditions and the second game's graph editing belong to the next campaign graph outcome.
- New empty maps use the existing map editor outside this slice; insertion here selects a held map.
