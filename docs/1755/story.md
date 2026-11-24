# A walking unit keeps its cell until it crosses, holds both cells and waits for the walker ahead

## Result

A walking unit now stays on the cell it leaves until it is halfway across, and only then stands on the new
cell, in both games. While it steps it keeps both cells, so a second unit cannot walk into the cell it is
leaving or the cell it is entering. A unit whose path is blocked by a standing unit searches round it,
while one whose path is blocked by a walker turns to the walker and waits for it to move on. A blocked unit
searches again only after a set number of passes, as in the original.

## Authority

- MOVE-STEP-010 (High), MOVE-088 (High), MOVE-194 (High): the cell changes on the crossing call.
- MOVE-CLAIM-007 (High): the next cell is claimed at the step's start and the old one released at landing.
- MOVE-130, MOVE-WAIT-008 (High / Medium): the dynamic count passes 32; the waypoint one cell off is tested.
- MOVE-REFRESH-012 (High / Medium): the static route is searched again past 16 near searches.
- MOVE-132 (High / Medium): the blocked-cell verdict; MOVE-119 (High): the constructor's +0x09.
- DIV-2027 and DIV-035 closed; DIV-2048, DIV-4527 and DIV-1843 rewritten; DIV-5125, 5126, 5127 and 5128 written.

## Rules

1. When a unit steps, its cell is the departure cell until the crossing call, then the destination cell.
2. When a unit steps, it holds the cell it leaves until it lands and the cell it enters from the start.
3. When a held cell is the next step of another unit, that unit does not enter it while it is held.
4. When a listed near step is held, the unit searches again only on the pass its count passes 32.
5. When a stored route has served more than 16 near searches, the next centred pass searches it again.
6. When the next route cell is one off and a walker holds it, a participant's unit turns to it and waits.
7. When the next route cell is one off and a standing unit holds it, the unit searches round it.
8. The walk counters survive the byte form, the digest and a native SAV and LOAD in both games.

## Code

| kind | its one builder | change |
|---|---|---|
| native step | `pkg/sim/step.go:2285` advanceStep, `:2256` payStride | commits the cell on the crossing call |
| stride state | `pkg/sim/stride.go:63` retainStride, `:92` StepCell, `:102` heldCell | the step's destination and held cell |
| occupancy plane | `pkg/sim/route.go:192` occupy | a stepping unit holds both cells |
| walk refresh | `pkg/sim/step.go:1050` walk section, `pkg/sim/walkrefresh.go:41` waitsBehind | counters, wait gate |
| byte form and digest | `pkg/sim/binary.go:1651` encodeInto, `pkg/sim/formwrapper.go:57` | the walk record, form 145 |
| SAV continuation | `pkg/sim/actioncontinuation.go:66` ActorContinuation | carries the walk record |
| drawn cell | `pkg/game/world.go:4424` drawnCell | the step's destination |
| imported actor cells | `pkg/sim/nativeactorcells.go:9` syncNativeActorCells | reads the step's destination |

## Proof

| test | rules |
|---|---|
| `TestAStepCommitsItsCellOnTheCrossingCall` | 1 |
| `TestAStepHoldsTheCellItLeavesUntilItLands` | 2, 3 |
| `TestABlockedListedStepIsSearchedAgainOnlyPastTheDynamicCounter` | 4 |
| `TestAStoredRouteIsSearchedAgainAfterSixteenNearSearches` | 5 |
| `TestAFollowerWaitsBehindAWalkerAndGoesRoundAStandingUnit` | 6, 7 |
| `TestTeardownRecomputesTheBodyCellOfAMoverFelledInTransit` | 1 |
| `TestABodyFelledMidStepSlidesFromWhereItWasDrawn` | 1 |
| `TestTheWalkStateSurvivesTheByteFormAndTheActionContinuation` | 8 |
| `TestARelocatedWalkerDropsItsWalkStateWithItsTarget` | 8 |
| `TestSavedStride1115BothDoorsMenuSaveAndContinuation` | 8 |

Release tests at the release: `TestReleaseAreaLayerCostRecomputesOnTheCrossingTick`,
`TestReleaseUnitKilledMidStepFinishesTheStepBeforeItsBodyLies` and
`TestReleaseMissionRoutesAndCostsMatchTheNativeReader` on the EN and RU installs of both games.

## Open debt

- DIV-5125: an original-format SAV mover record does not carry the walk counters.
- DIV-5126: the verdict's participant word, pending codes and retry cadence are mapped, not read.
- DIV-5127: the second game's walk step takes the first game's claims.
- DIV-5128: a body felled before its crossing slides back to its cell; MOVE-096 leaves it unstepped to teardown.
- DIV-2048: a cell closed by terrain within reach still sends a far search.
