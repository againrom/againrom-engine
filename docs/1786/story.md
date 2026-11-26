# Second-game computer units follow the answered behaviour checks

## Result

In the second game a computer unit now walks, defends and withdraws by what the original's code was read to do. Its
long route search ends one generation later when the goal is first reached on an odd step, so a cheaper approach to
the goal can be taken. Its walking steps aim at the route end once five cells remain. A defender of a short-range
unit ignores flyers and ranks enemies by the distance between bodies. Invisible enemies stay in a withdrawal or defend
list while a comrade of the group detects them. The first game is unchanged.

## Authority

- Owner: «исследовать если не знаем и внедрить» on seven questions of unit behaviour.
- `R2-ENGINE-1876` (High / Medium): size-1 far search tests goal, frontier and limit once per pair of half passes.
- `R2-ENGINE-1761`: the repath aims at the route end for five cells or fewer, at the head beyond three cells, else
  at the fourth cell.
- `R2-ENGINE-1878`, `R2-ENGINE-1188`: defender list drops kind 3 below range 2, ranks kind 3 then edge distance.
- `R2-ENGINE-1879`: an invisible candidate is dropped unless a list unit detects it. `R2-ENGINE-1877`: ties keep the
  first. `R2-ENGINE-1252`, `R2-ENGINE-1440`, `R2-ENGINE-1880`: radii 8, 5 and 10; Teleport tests book and mana.
- DIV-5272 to DIV-5276: the Unknown parts, kept as the first game reads them.

## Rules

1. The far search of a second-game one-cell unit tests its goal once per pair of generations.
2. A second-game walk aims at the route end for five cells or fewer, else the head beyond three cells, else the fourth.
3. A defender of a unit with range below 2 does not list flyers, and ranks the list by edge distance.
4. A withdrawal or defend list keeps an invisible enemy only while a living unit of the group detects it.
5. Equal group scores keep the first candidate; the defender list reaches 5 cells; the post floor is 8.
6. The first game's search, walk, defender and withdrawal are unchanged.

## Code

| kind | its one builder | change |
|---|---|---|
| route search | `pkg/sim/route.go` canonicalRoute | pair test from a profile field |
| route-end goal | `pkg/sim/pursuitsearch.go` routeEndGoal | the pursuit and the walk read the one rule |
| target choice | `pkg/sim/targetchoice.go` chooseTarget | edge distance from a profile field |
| defender list | `pkg/sim/escort.go` coverEngage | flyer and invisible filters from a profile field |
| retreat decision | `pkg/sim/withdraw.go` withdrawalHostiles | group detection from a profile field |

## Proof

| test | rules |
|---|---|
| `TestSecondFarSearchTestsTheGoalOncePerPairOfGenerations`, `TestSecondNearSearchKeepsTheTestBeforeEveryGeneration` | 1, 6 |
| `TestSecondWalkAimsByTheRepathRule` | 2, 6 |
| `TestSecondDefenderDropsFlyersForAShortRangeUnit`, `TestSecondDefenderRanksByEdgeDistance` | 3, 6 |
| `TestSecondWithdrawalListAsksTheGroupToDetect`, `TestSecondDefenderListDropsAnInvisibleEnemyNoMemberDetects` | 4, 6 |
| `TestSecondGroupTargetTieKeepsTheFirstOfTheList`, `TestSecondDefenderListReachesFiveCellsFromTheDefendedUnit` | 5 |
| `TestSecondGuardPostRadiusFloorIsEight`, `TestSecondWalkOverACostPlaneIsDeterministic` | 5, 1 |

Release tests at the release: none.

## Open debt

- DIV-5272 to DIV-5276 wait for the original runs they name; the later landing reconciles the shared code with the
  first game's story.
