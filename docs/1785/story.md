# The first game's units follow the answered behaviour checks

## Result

In the first game a walking unit now aims its near search where the original does: at the end of a route of five
cells or fewer, at the route's head when it lies more than three cells away, else at the fourth cell. A guard post and
a defender's cover look as far as the unit's own search radius byte. A defender ranks enemies edge to edge and ignores
fliers while the unit it protects reaches one cell. A retreating unit keeps an invisible enemy in mind when a group
member can see invisible at that distance. The second game is unchanged.

## Authority

- Owner: «исследовать если не знаем и внедрить» on seven unit-behaviour questions.
- `MOVE-208`, `MOVE-REFRESH-012`: the near search goal. `AI-810`, `AI-811`, `AI-BREAK-041`: the radius byte.
- `AI-812`, `AI-813`, `AI-DEFEND-111`: cover. `AI-814`, `AI-340`: group ties. `AI-815`: retreat and invisibility.
- `AI-816`: no Teleport branch in a first-game withdrawal. `MOVE-COST-002`, `MOVE-TERM-003`: route cost and stop.
- DIV-5264 to DIV-5268: the Unknown parts and the original runs that settle them.

## Rules

1. A stored route of five cells or fewer is aimed at its last cell.
2. A longer route is aimed at its head when the head is over three cells from the unit, else at its fourth cell.
3. A guard post scans the block of the guard's own mover byte, 5 as constructed, a restored 0 included.
4. A defender's cover scans the block of the protected unit's mover byte about that unit.
5. Cover picks the enemy nearest edge to edge, a flier first, the first listed on a tie.
6. Cover drops every flier when the protected unit's reach is below 2.
7. A retreating unit keeps an invisible enemy when a group member's See invisible byte reaches its edge distance.
8. Wimpy and Retreat scan the mover byte, Withdraw scans 2, and no first-game withdrawal casts Teleport.
9. A group's target tie keeps the first candidate; route steps cost the destination cell byte, diagonal plus half.
10. The second game's cover, guard, retreat and route aiming keep their behaviour.

## Code

| kind | its one builder | change |
|---|---|---|
| route-end goal | `pkg/sim/pursuitsearch.go` routeEndGoal | the walk and the pursuit both call it, no switch |
| scan radius | `pkg/sim/withdraw.go` fleeRadius | guard post and cover read it too |
| defender list | `pkg/sim/escort.go` coverEngage | the flier drop and group filter run in both games |
| distance | `pkg/sim/targetchoice.go` targetEdgeDistance | cover and retreat ask it |
| retreat hidden enemy | `pkg/sim/withdraw.go` withdrawalHidden | the group rule runs in both games; the edge measure is the listDetect field |
| route search | `pkg/sim/route.go` canonicalRoute | unchanged, matches the claims |
| spell choice | `pkg/sim/withdraw.go` teleportEscape | unchanged, second game only |

## Proof

| test | rules |
|---|---|
| `TestTheNearSearchAimsAtTheRouteEndHeadOrFourthCell` | 1, 2 |
| `TestSecondWalkAimsByTheRepathRule` | 1, 2 |
| `TestAGuardPostScansItsOwnMoverByte` | 3 |
| `TestACoverScansTheProtectedUnitsMoverByte` | 4 |
| `TestACoverRanksByEdgeDistance`, `TestTheCoverEngagementPrefersAirAndThenTheNearest` | 5 |
| `TestACoverDropsFliersForAProtectedUnitOfReachBelowTwo` | 6 |
| `TestARetreatSeesAnInvisibleHostileByTheGroupsEdgeDistance` | 7 |
| `TestWimpyAndRetreatSearchTheMoverRadiusNotScanRange`, `TestFirstGameWithdrawUnchanged` | 8 |
| `TestTargetChoiceKeepsGroupAndRepickTieRules`, `TestStepCostIsTheContractsFormula` | 9 |
| `TestFirstGameOrderCellsKeepTheirBytes`, `TestSecondGameWithdrawalKeepsScanRange` | 10 |

Release tests at the release: none.

## Open debt

- StaticIsntNeeded and DynamicByStaticLookup stay named constants equal to the shipped values; the map data is not read.
- The walk drops route cells on landing, the original unlinks the head per near search (DIV-5267).
