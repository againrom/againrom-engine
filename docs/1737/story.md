# Each game picks its own version of a rule from its profile

## Result

Nothing changes for the player in either game. Where the two games play one rule in two ways, the game now picks
its version from its profile instead of one shared routine holding both versions behind a switch. Every unit, mission
and saved game behaves as before, and recorded worlds hash the same.

## Authority

- Owner: «Объединить N100 отдельной историей» on the architecture audit of engine d85bb2f3, row N100.
- Claim-free: no new fact about the original is stated; each version keeps the claims it cited before.

## Rules

1. When a withdrawal finds no route, the first game reacquires within reach and the second stops in place, as before.
2. When a player sets the retreat mode, each game writes the same Withdraw and Wimpy thresholds as before.
3. When a unit walks up or down a slope, each game rounds its step rate as before.
4. When the mission script counts losses and wins, each game reports the outcome as before.
5. When a computer mage chooses an automatic cast, each game takes the same units' sight as before.
6. When a projectile aims or a unit flees, each game reads the same point of a unit as before.
7. Support holds, skill raises, creature cast steps and blow draws keep each game's order and tests.
8. A world played or loaded in either game hashes the same as before.

## Code

| kind | its one builder | change |
|---|---|---|
| refused-flee answer | `pkg/sim/withdraw.go` answerRefusedFlee | two named answers; the profile picks one |
| retreat thresholds | `pkg/sim/playercommand.go` setRetreatMode | uniform and by-class rules; the profile picks |
| step rate | `pkg/sim/rate.go` rateOf | slope rounding is a row of two rules |
| outcome and sight | `pkg/sim/script.go` scriptReport; `autocasttarget.go` autoCastObservers | rows |
| aim and flee point | `pkg/sim/heading.go` projectileAimPoint; `withdraw.go` fleePoint | point rows |
| policy rows | `supportHeld`, `skillAwardProfile.raises`, `creatureStepCell`, `blowValue` | nested rows |
| guard | `internal/archtest/profilerules_test.go` | no converted flag name in `pkg/sim` |

## Proof

| test | rules |
|---|---|
| `TestBlockedFleeAnswerByGame`, `TestWithdrawalWhoseFleeCellHasNoRouteTakesTheStandingPickAndShoots` | 1 |
| `TestRetreatModeThresholdsByGame`, `TestRetreatPlayerCommandRefusesUnknownModeAndPlayerLikeAQuietTick` | 2 |
| `TestGroundStepUphillRoundingByGame`, `TestTheWorkedExampleReproduces` | 3 |
| `TestOriginalOutcome1097IndependentCountersNativeAndReporter` | 4 |
| `TestTheSecondGameAutomaticCastTakesWhatItsSideSees`, `TestOwnerSeesAnswersAsEachUnitsCastSight` | 5 |
| `TestAttackFacingSecondGameKeepsAimAndAcquisition`, `TestSecondGameFleeRadiusIsTenWhateverTheScanRange` | 6 |
| `TestComputerSupportVictimHold`, `TestAwardSkillRaisesOnlyWhenStrictlyAboveTheThreshold` | 7 |
| `TestACreatureStepCellFollowsTheEightWayHeading`, `TestTheSecondGameBlowDrawsInTheOriginalOrder` | 7 |
| `TestTheFirstGameBlowKeepsItsDraws`, `TestALoadedNativeSkillHoldsItsGamesLevel` | 7 |
| `pipeline/landing-witnesses.py` world hashes and witnesses, base against candidate | 8 |
| `TestProfileRulesAreRows` | 1, 2, 3, 4, 5, 6, 7 |

Release tests at the release: none.

## Open debt

- Profile fields left with a two-version reader (cast reach, cast recovery count, activity mask, native tick, native
  decay, live admission and others) wait for an owner decision; the review report lists them.
