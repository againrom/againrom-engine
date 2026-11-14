# The campaign graph reads by tens, in colours, with its films

## Result

In the map editor the Campaign graph now stands the missions of one ten on one row in number order,
and each next ten on the row below, so the campaign reads top to bottom.
Each kind of node has one colour: start, main mission, side mission, town, tavern talk, shop or school, film.
A legend above the graph names the kinds shown, in English or in Russian.
The start film and each victory film the game will play appear as nodes under the start and under their mission.

## Authority

- Owner: missions of one ten stand on one level, so the campaign reads top to bottom by tens.
- Owner: one colour scheme tells main missions, side missions, talks, towns and films apart, with a legend.
- Owner: films are shown too, where the campaign has them.
- REG-SCN-059: the campaign's main sections are those numbered by whole tens; the rest are side sections.

## Rules

1. When the graph is shown, missions 10 to 19 share one row in number order and each later ten stands lower.
2. When a chapter has a town, the town stands at the left of its ten's row with its hand-outs beside it, one a row.
3. When a mission's number is a whole ten of the campaign, it is main; any other, and any mission a mod adds, is side.
4. When a node is drawn, its edge and top band carry its kind's colour from the one table, on the panel's ground.
5. When the graph is shown, the legend names each kind it shows, in table order, in the editor's language.
6. When the install's video archive holds `start/01.smk`, a film node under the start names the start's files.
7. When it holds `m<N>/01.smk`, a film node under mission N names the files played on its victory.
8. When neither file is held, no film node and no film legend row appears.
9. When a node is clicked, selected or edited, it behaves as before; the second game still shows no graph.

## Code

| kind | its one builder | change |
|---|---|---|
| film lookup | `CutsceneBank.find` (was `Media`'s body) | `Media` and the new `Films` ask it |
| film directory names | `ui.NumberedCutscenes`, `StartCutscenes`, `MissionCutscenes` | play path and graph share the names |
| graph page | `campaignGraphOf` | levels by tens, `Side` from `Campaign.Main`, film nodes as data |
| graph canvas | `MapEditor.graphCanvas`, `graphBox` | kind table `graphKinds`; lanes per gap; band and legend |
| popup canvas | `popupSpec.Canvas`, `layoutPopup`, `drawPopup` | a legend strip over the view; a box band |
| words | `pkg/words/text/{en,ru}/strings.toml` | kind names, film tips, the levels caption |

## Proof

| test | rules |
|---|---|
| `TestCampaignGraphStandsByTensInKindColoursWithFilms` (EN, RU; install, no films, mod) | 1, 3, 4, 5, 6, 7, 8 |
| `TestCampaignGraphColoursEachKindFromOneTable` (EN, RU) | 4, 5 |
| `TestCampaignGraphLayoutComesFromItsData`, `TestCampaignGraphNamesAnArrowsHeroes` | 1, 2 |
| `TestCampaignGraphHoldsTheFlowTheTownsAndEachHandout`, `TestCampaignGraphMarksWhatAModChanges` | 1, 2, 3 |
| `TestCampaignGraphHandoutNodesSelectThroughTheHost`, `TestEditorGraphOpensAMissionAMapFileReplaces` | 9 |
| `TestCampaignFieldsThatCannotChangeAreLines` (second game unread) | 9 |
| `TestMapEditorModesHoldEveryKeyAndGesture`, `TestMapEditorModesHoldTheGraphRows` | 9 |

Frames of the synthetic page: the review report of this story. Release tests at the release: none named.

## Open debt

- A mod cannot carry a film; the graph reads the install's video archive only.
- The graph reads the default video route; whether the other speed route holds the same names is Unknown.
- The second game has no graph, so neither levels nor films show for it.
