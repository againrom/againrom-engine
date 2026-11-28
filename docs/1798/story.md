# Walking and pursuing around occupied cells

## Result

A wide unit in the first game can plan past another unit while still needing room for its whole body to step.
A pursuer can keep its chosen step through a turn. Walkers in both games drop a nearby waypoint when they search again.
In the second game a unit whose step has no movement can take another order after its first advance.

## Authority

- Owner: one order, route and AI machine per game through one builder per kind.
- ROM1 EN: AI-882, AI-883, AI-885, AI-887, AI-888, MOVE-192, MOVE-205 and MOVE-208.
- ROM2 RU: R2-ENGINE-1960 through R2-ENGINE-1970. Equal EN trace instructions do not prove the EN walk driver.
- DIV-2561 and DIV-4953 retain the owner's answers to traffic at a player's goal.

## Rules

1. Local search tests the anchor. Entering the cell requires room for the whole unit (ROM1 EN, Medium).
2. A saved ground reservation blocks local ground search even on an empty cell (ROM1 EN, Medium).
3. A long route with a nearby head uses the configured lookup index (ROM1 EN and ROM2 RU, Medium).
4. A local search drops one head within lookup distance (ROM1 EN and ROM2 RU, High for the branch).
5. Pursuers keep a step until taken or due for another search on a centred call (ROM1 EN, Medium in play).
6. Canonical traces beyond a thousand steps return empty (ROM1 EN, Medium, and ROM2 RU, High).
7. A straight predecessor wins a tie. A diagonal must cost less. The scan includes centre (ROM2 RU, High).
8. A zero-vector step counts as standing after one advance with centred fractions (ROM2 RU, Medium).
9. A loaded mover tests both position fractions after advancing (ROM2 RU, High for the test).
10. An empty local search to its final goal gives a computer walk its refusal answer (ROM2 RU, High for the branch).

## Code

| kind | its one builder | change |
|---|---|---|
| route search and trace | `route.go` canonicalRoute | neighbour mask, trace cap and centre tie |
| local waypoint | `pursuitsearch.go` routeEndGoal | index and destination mode for walking and pursuit |
| local step list | `walkrefresh.go` nearStep | search removes the static head |
| pursuit | `pursuitsearch.go` pursue | centred-call age and retained step |
| cell crossing | `step.go` advanceStep | footprint admission; standing after a zero vector is the `standingBodies` row |
| loaded motion | `savedmotion.go` SavedActorPosition.centred | shared fraction test |
| refused route | `refusedroute.go` answerRefusedRoute | dispatch an empty destination search through the order row |
| occupancy | `route.go` routeScratch.occupy | hold the whole stepping footprint |

## Proof

| test | rules |
|---|---|
| `TestRouteSearchFootprintAndStepAdmission` | 1 |
| `TestNearRouteReadsHeldPlaneWithoutAnOccupant` | 2 |
| `TestRouteRepathChoosesConfiguredListWord` | 3 |
| `TestNearSearchUnlinksTheHeadBeforeTheStep` | 4 |
| `TestPursuitRefreshCountsCentredCalls` | 5 |
| `TestRouteTraceStopsBeyondAThousandSteps` | 6 |
| `TestSecondGameZeroCostCentreCanWinTheRouteTie` | 7 |
| `TestTheWalkBackAcceptsAnEqualStraightAndRefusesAnEqualDiagonal` | 7 |
| `TestStandingAfterZeroVectorSurvivesReload` | 8 |
| `TestLoadedStepReadsBothFractionBytes` | 9 |
| `TestSecondGameEmptyDestinationRepathStopsComputerWalk` | 10 |

Rule 4 also unlinks the head after an empty local search (AI-373, MOVE-208, R2-ENGINE-1965), so four older tests
changed. The damage ladder and the stalled tick no longer expect a boxed-in unit to keep its whole route. The
full-map group order holds 250 cells after its first tick. The patrol hashes of the group words record were
refreshed: a patroller stuck beside its goal loses the head cell, so the wait gate no longer holds it.

Release tests at the release: `TestReleaseMissionRoutesAndCostsMatchTheNativeReader`,
`TestReleaseCellPlanesSurviveSaveColdLoadAndTicks`.

## Open debt

- DIV-2556 and DIV-5214 need delivered calls and the delay from arrival to the next crossing.
- DIV-4759, DIV-4760 and DIV-4761 need occupied goals, pursuit endings and deferred order stores.
- DIV-5353 and DIV-5356 need first-game wide entry and a next-cell claim sampled while turning.
- DIV-5354 and DIV-5355 need second-game repeated zero-vector steps and target-ring side order.
