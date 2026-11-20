# A mod town's tavern page is laid out from its own description

## Result

A town's tavern page takes its room buttons, seat appearance, seats and backdrop from that town's description, and
keeps the game's value only where the room names nothing. Each button acts by its name: hire, talk, exit and sleep; a
button with another name does nothing. The shipped towns of both games show the same page.

## Authority

- Owner order «Сначала дешёвое» on the architecture audit, item 4. No divergence: the room data is Againrom's own and
  no game rule changes.

## Rules

1. When a town's tavern room lists buttons, the page shows those buttons and each acts by its name.
2. When a town's tavern room sets a seat appearance, the page paints it; when the room names none, the game's shows.
3. When the first town is the shipped one, its tavern page layout equals the instrument layout.

## Code

| kind | its one builder | change |
|---|---|---|
| town room page | `pkg/game/roompageview.go` `buildRoomPage` | `tavernRoom` merges the town's room over the game's once for every reader |
| room lookup of a town description | `pkg/game/roompageview.go` `tavernRoom` | buttons and appearance join seats and backdrop |

## Proof

| test | rules |
|---|---|
| `TestModTavernPageTakesButtonsAndSeatAppearanceFromItsTown`, `TestModTavernButtonsPlaceSleepByName` | 1, 2 |
| `TestModTavernPageFallsBackWhereTheTownNamesNothing` | 2 |
| `TestShippedTavernPageLayoutIsUnchanged`, `TestTavernRoomWithoutTheTownsOwnIsTheGamesRoom` | 3 |

Release witness at the release: headless tavern page on rom1-en, rom1-ru and rom2 with a mod tavern room.

## Open debt

- `TownRoomButtonRect`, `TavernPageLayout`, `SurfacePageLayout` and `TownSchoolRegion` are town-less instruments and
  read the first game's description; the school page's own reads wait for a school-page backlog item.
