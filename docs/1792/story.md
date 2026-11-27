# A multiplayer hero enters a Shop or an Inn and the deal is saved

## Result

On a multiplayer map, a hero who uses a Shop opens the Shop page and, in the second game, an Inn opens the Inn page.
He is off the map while the page shows. When he leaves, he stands on his own cell, or on a free cell within three of
it. Each buy or sell saves the character file, at most once every 15 seconds. Campaign maps are unchanged.

## Authority

- Owner: BACKLOG MP-8, the map buildings are the shop, tavern quests and the town hall.
- Owner: the second game's session pages come from the single-player Shop and Inn builders.
- `R2-ENGINE-1912`: Shop and Inn entry take the unit off the map; leave places it at its cell, then within 3.
- `R2-ENGINE-1911`: a Shop buy or sell calls the character writer once per 15 seconds.
- `R2-ENGINE-1910`, `R2-ENGINE-1913`: the page a server opens and the Church effect are Unknown.
- DIV-5308 to DIV-5312: entry and leave, the deal write, the second game's pages, the Church, the first game.

## Rules

1. On a session map only a party hero that uses a Shop or an Inn leaves the map, until his page closes or is refused.
2. He leaves onto his own cell, else a free cell within 3, else the nearest free cell.
3. A Shop deal writes the character file through the session writer, at most once per 15 seconds of the clock.
4. The second game's Shop shows its town's single-player stock; a buy moves the unit and price into pack and purse.
5. The second game's Inn shows the single-player offers of its town; taking one follows the single-player rule.
6. The Church builds no behaviour, and the first game's Shop kinds follow the second game's rule.
7. A summon, hire, guest or monster that uses a Shop or an Inn stays on the map and opens nothing.
8. A campaign map keeps its hero on the map and a campaign Shop deal writes no character file.

## Code

| kind | its one builder | change |
|---|---|---|
| map entry | `pkg/game/missionentry.go` mapEntry | `buildings` value, seated campaign state |
| Shop and Inn page | `pkg/game/missionshop.go` openMissionVisit | a rule per game as profile data |
| character file | `pkg/game/mapsession.go` writeSession | the end and a deal share it |
| presence | `pkg/sim/buildingvisit.go` | enter and leave; kind, class and town |
| command | `pkg/sim/command.go` LeaveBuilding | new kind and its constructor |
| game profile | `pkg/sim/gameprofile.go` buildings | Shop and Inn kinds as data |

## Proof

| test | rules |
|---|---|
| `TestVisitTakesTheUserOffTheMapAndLeaveBringsItBack` | 1, 2, 6 |
| `TestChurchAndTheFirstGamesNoInnBuildNoVisit` | 6 |
| `TestSessionShopEntryTakesTheHeroOffTheMapAndLeaveReturnsHim` | 1, 2 |
| `TestSessionShopLeaveWithTheCellTakenStandsWithinThree` | 2 |
| `TestSessionShopDealsWriteTheFileOncePerFifteenSeconds` | 3 |
| `TestSessionSecondShopOpensTheKindsTownPage` | 4 |
| `TestSessionSecondShopBuyMovesTheUnitAndThePriceAndWrites` | 3, 4 |
| `TestSessionSecondInnListsTheSinglePlayerOffers` | 5 |
| `TestOnlyAVisitorEntersAShopOrAnInn` | 7 |
| `TestSessionSummonUsingAShopOrAnInnStaysOnTheMap` | 7 |
| `TestVisitsOffLeaveShopsAndInnsAsTheyWere` | 8 |
| `TestCampaignShopDealKeepsTheHeroOnTheMapAndWritesNothing` | 8 |

Release tests at the release: none.

## Open debt

- The kind-to-town map and the seated new-game campaign state are the owner's single-player choice (DIV-5310).
- A saved multiplayer map would need the visit rule and the visitors, which no SAV path holds yet.
