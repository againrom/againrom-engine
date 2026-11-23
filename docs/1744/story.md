# The remaining two-body profile switches become rule rows

## Result

Nothing changes for the player in either game. Where the two games still ran one rule through two branches behind a
boolean switch in the simulation profile, spell presentation or effect picture row, each game now selects its version
from a rule row table. Every tick clock, decay walk, cell tail, cast reach, recovery count, group activity refresh,
attack admission, empty-route answer, spell relief height and unit cast life behaves as before.

## Authority

- Owner: architecture audit of engine d85bb2f3, row N100 ("flags that choose logic"), continuing the conversion begun
  in `docs/1737/story.md`.
- Claim-free: no new fact about the original is stated; each version keeps the claims it cited before.

## Rules

1. Each game advances its tick clock, script phase cadence, pass query and clock restore as before.
2. Each game walks decaying bodies and teardown health steps in one pass as before.
3. Instant 25 encodes each game's six cell-tail bytes and repairs older targetless events as before.
4. Automatic and scroll unit-target cast reach uses each game's distance test with zero sight allocations when idle.
5. Cast recovery narrows to each game's byte rule for book, scroll, staff and blow origins as before.
6. The engagement pass refreshes saved Group activity by each game's rule as before.
7. Attack admission, weapon-cast reach, refused-walk arms, reacquisition reach and no-victim refusals match each game.
8. Spell bolt and display lift pick cell-corner or point-blended height, and unit cast life picks each row's rule.

## Code

| kind | its one builder | change |
|---|---|---|
| tick clock | `pkg/sim/step.go` stepWorld; `sessionclock.go`, `script.go`, `currentworldpolicy.go` | `tickClocks` row replaces `nativeTick` |
| decay walk | `pkg/sim/step.go` decayPass | `decayWalks` row replaces `nativeDecay` in one walk |
| cell tail & activity | `pkg/sim/celltail.go` cellTailBytes; `engage.go` engagementPassTraced | `cellTailEncoders` and `groupActivityRefreshers` rows; on/off gates stay data |
| cast reach & recovery | `pkg/sim/spell.go` withinCastReach, castRecoveryTicks | `castReachRules` (value args, zero alloc) and `castRecoveryRules` rows |
| order admission & routes | `pkg/sim/combat.go` closedOn, weaponCastInRange, refusedWalkArm; `reacquire.go`, `refusedroute.go` | `attackReachRules` and `emptyRouteRules` rows; on/off gates stay data |
| spell lift | `pkg/ui/spellbolt.go` spellBoltLift; `cellprojection.go` groundHeight; `pkg/data`, `pkg/game` | `SpellLift` enum and `spellBoltLifts`/`groundHeights` rows replace `PointLift` |
| unit cast life | `pkg/sim/flightproducer.go` unitSegments; `pkg/data`, `pkg/game` | `UnitLifeRule` and `unitSegmentRules` replace `UnitLifeZero` |
| guards | `internal/archtest/profilerules_test.go`, `internal/storyguard/baseline.go` | extend forbidden flag scan; lower comment-byte baseline |

## Proof

| test | rules |
|---|---|
| `TestSessionClockReadsEntryAndPostBodyPhasesSeparately`, `TestCurrentROM2PolicyPreservesBankAndActivity` | 1 |
| `TestTheWalkTakesOneHealthPerPeriodAndSaturates`, `TestCurrentDecayHealthPublishesOnlyExistingBasis` | 2 |
| `TestSecondGameTrapStoresSourceAndTarget`, `TestFirstGameTrapKeepsItsOwnBytes`, `TestRepairCellEventsRewritesOnlyTheOlderForm` | 3 |
| `TestIdleAutocastDoesNotAllocateSight`, `TestAUnitCastReachIsTheFootprintGap`, `TestTheSecondGameAutomaticCastKeepsSightAndAnchorCells` | 4 |
| `TestCastRecoveryCountPerGameAndSource`, `TestCastRecoveryEstimateDrawsNothing` | 5 |
| `TestAnUnnamedGroupDecidesOnlyWithAPartyUnitNear`, `TestAuthoredControlGroupActivityAndColdRestore` | 6 |
| `TestClosedOnIsTheCastAdmissionForAnEligibleCasterAndInReachOtherwise`, `TestTheSecondGameStaffCastStartsAtTheWeaponRangeFromTheNearestCell` | 7 |
| `TestRePickUsesTheCurrentAttackRange`, `TestASuppressedMageRefusedRouteWritesThePickAndIdlesTheOrder` | 7 |
| `TestABoltsDepartureCarriesNoHandHeight`, `TestDisplayRowFindsTheGroundRowUnderADisplayPoint` | 8 |
| `TestSecondUnitCasterOfAHeldMarkSpellEndsItsCastRecordAtOnce`, `TestFlightProducerOrdinaryRangedOrders` | 8 |
| `TestRuleRowsAgreeWithTheirGates` | 1, 2, 3, 6, 7 |
| `TestProfileRulesAreRows`, `TestDAG`, `TestCastRecoveryHasOneBuilder`, `TestCellProjection`, `TestWorldGame`, `TestLiveTreeClean` | 1, 2, 3, 4, 5, 6, 7, 8 |

Release tests at the release: none.

## Open debt

- none
