# 0167 — escort tick: verification

Branch `0167-escort-tick`, base `90ed4f6`, research pin `7747b9d`. Written after
the gate below was run on a clean tree.

## The gate

```
go build ./...                                     clean
go vet ./...                                       clean
gofmt -l $(git ls-files --cached --others --exclude-standard '*.go')   prints nothing
go test -trimpath -count=1 ./...                   all packages ok
scripts/check-doc-budget.sh                        ok
scripts/check-hotfix-ledger.sh                     ok
scripts/check-no-game-assets.sh                    ok (tree scan)
scripts/check-sdd-audit-selftest.sh                ok
scripts/check-sdd-audit.sh                         ok (none enforced)
```

No `Co-Authored-By` trailer is on any commit of this branch.

## The result someone can point at

The script-gap census is **unchanged**, and that is the correct outcome: this
story runs no new script operation. It gives an arm to a state a script operation
already writes.

```
mission 10 UNSUPPORTED   13   (pipeline/milestone-baseline.txt: 13)
mission 20 UNSUPPORTED   11   (pipeline/milestone-baseline.txt: 11)
```

What moved is in `builds/0167-escort-tick/`. Sixteen shipped group-command nodes
author Defend or Follow; six are bound to a trigger. **Mission 10's three are
not**, so the mission-10 drive cannot exercise these arms and is byte-identical
to its baseline, `outcome lost at tick 224`, four movers, one fell. Mission 100
is the map that can: its trigger 0 fires at tick 6 with a Follow over group 21
naming the party hero at range 3, and trigger 1 at tick 102 re-runs it at range 6.

```
builds/current, 600 ticks:   1 of 114 unit(s) moved   (u186, unrelated)
this build, 600 ticks:       2 of 114 unit(s) moved
  moved  u135   slot 5 group 21  (23,26) -> (17,20)  6 step(s), 55 hp
```

`u135` is group 21's member. The unit it follows stands at (15,16), so the
follower closed from 10 cells to 4 and then stood: the same drive to 3000 ticks
ends on the same cell, which is the stop the range of 6 implies. Both figures
were re-run from `builds/0167-escort-tick/` the owner's way, with
`$env:AGAINROM_ASSETS` set, before the README recorded them.

**Nothing was watched on a screen.** This story opens no window and sends no
synthetic input. What a player would see is witnessed through `cmd/missionrun`,
which drives `sim.Step` directly.

## FR

- **FR-1** `TestBothEscortStatesReachAnArmAndAcquireStillDoesNot`. Reverted by
  unwiring the two cases: the escort holds no destination and the test names the
  state that reached no arm.
- **FR-2** `TestAnEscortOfAUnitTheWorldDoesNotHoldIsLeftAlone`, on the whole
  record rather than on named fields.
- **FR-3** `TestAnEscortRangeOfZeroTestsAgainstTheScanRange`, with the control at
  the same distance and the stored range in force. Reverted by dropping the
  fallback.
- **FR-4** `TestAnEscortOutOfRangeClosesOnItsSubject` on both arms, and
  `TestAFollowerClosesAndThenStopsWithinItsRange` through whole ticks.
- **FR-5** `TestADefenderEngagesOnItsSubjectsBehalf` (the block is centred on the
  subject, at 4 cells and at 6) and `TestTheCoverFiltersPolarityIsTheSubjects`.
- **FR-6** `TestTheCoverEngagementPrefersAirAndThenTheNearest`, both arms of the
  preference.
- **FR-7** `TestAnEmptyCoverBlockFallsToTheStandingAcquisition`, with and without
  something visible.
- **FR-8** `TestADefenderThatIsFightingDoesNotStepAway`, with the idle control
  that does step away.
- **FR-9** `TestAFollowerInRangeReAcquiresAndNeverScansForItsSubject`, whose
  second case runs the identical fixture under both arms so that the follower's
  refusal is not a fixture artefact, and
  `TestAFollowerThatScoresNothingEndsItsWalk`.
- **FR-10** `TestACrowdedEscortStepsAwayAlongTheLine` over all nine relative
  positions and both arms, and `TestTheStepAwayIsClampedToThePlayableRectangle`
  at both corners.
- **FR-11** `TestAnEscortArmWritesNoStateAndSurvivesTheByteForm`: the actor state,
  the escort triple and the patrol ring are all read after four phases.
- **FR-12** enforced by `internal/archtest`'s source scan over `pkg/sim`, which
  fails on a float identifier or literal, and by the round-trip half of the test
  above.

## AC and P

- **AC-1** `TestAnEscortOutOfRangeClosesOnItsSubject` and
  `TestAFollowerClosesAndThenStopsWithinItsRange`, which asserts both bounds: at
  most the range and at least the crowding distance.
- **AC-2** `TestAFollowerReAimsWhenTheGapOpens`.
- **AC-3** `TestADefenderEngagesOnItsSubjectsBehalf`, both cases.
- **AC-4** `TestTheCoverFiltersPolarityIsTheSubjects`, both directions.
- **AC-5** `TestTheCoverEngagementPrefersAirAndThenTheNearest`.
- **AC-6** `TestAnEmptyCoverBlockFallsToTheStandingAcquisition`.
- **AC-7** `TestADefenderThatIsFightingDoesNotStepAway`.
- **AC-8** `TestACrowdedEscortStepsAwayAlongTheLine`.
- **AC-9** `TestAnEscortOfAUnitTheWorldDoesNotHoldIsLeftAlone`.
- **AC-10** `TestAnEscortRangeOfZeroTestsAgainstTheScanRange`.
- **AC-11** `TestAnEscortArmWritesNoStateAndSurvivesTheByteForm`: the advanced
  world marshals, decodes and hashes to the same value. `formatVersion` is
  untouched — **51 was allocated for this story and is returned unused**, because
  0166 had already landed every field these arms read.
- **AC-12** `TestAWorldWithNoEscortAdvancesExactlyAsItDid`, and the mission-10
  drive above, which is the same claim on a shipped map.
- **P-1** `TestTheCoverBlockAndTheAcquisitionAreOneSweepEach` drives a
  hundred-entity world through several passes. It bounds the shape, not a time.
- **P-2** `TestAnEscortArmAllocatesNoRoute`.

## DD and SC

- **DD-1** two arms, `armDefend` and `armFollow`, sharing `escortClose` and
  `escortStepAway`. Both arms are run over the same fixtures in the FR-4 and FR-10
  tests, which is what makes the shared halves' identity checkable.
- **DD-2** `escortSubject` resolves once and returns the index; every later term
  reads that index.
- **DD-3** FR-2's test, and the doc block on `escortSubject` records the dead-but-
  present case the claim predicts and this build leaves standing.
- **DD-4** `TestACrowdedEscortStepsAwayAlongTheLine`. The nine expected cells are
  the cell-granular reading; the alternative reading is stated in plan.md and
  would move six of the nine.
- **DD-5** `TestRoundDivRoundsHalfAwayFromZero` on the helper directly.
- **DD-6** the corpse rule is exercised through the same `live`/`dead` shape
  `candidates` uses; no shipped-map case reached it in the drives.
- **DD-7** `acquireStanding` calls `candidateCost` under `orderStandGround`;
  FR-7's and FR-9's tests are its witnesses.
- **DD-8** `actorCandidates`, witnessed by the same two.
- **DD-9** `TestADefenderThatIsFightingDoesNotStepAway`.
- **DD-10** the victim assertions in `TestAnEscortOutOfRangeClosesOnItsSubject`.
- **DD-11** `TestAFollowerThatScoresNothingEndsItsWalk` for the mechanism and
  `TestAFollowerClosesAndThenStopsWithinItsRange` for the consequence.
- **SC-1** `TestBothEscortStatesReachAnArmAndAcquireStillDoesNot`'s second
  assertion. Reverted by wiring actor state 0xc to `acquireStanding`: the named
  unit's destination is cleared and the test names the cut.
- **SC-2** not built and not witnessed by a test; `AI-FOLLOWHEAL-118` is cited in
  provenance.md and the fall-through it describes is what `coverEngage` runs.
- **SC-3** an acquisition that scores nothing clears the order and writes no
  facing, which `TestAFollowerThatScoresNothingEndsItsWalk` reads.
- **SC-4** no per-tick hook was added; the move loop is untouched, which the
  absence of any `escort` symbol in `step.go` shows.

## Mutation kills

Eleven single-line mutants of `pkg/sim/escort.go` and `pkg/sim/actor.go` were
introduced one at a time and the whole `pkg/sim` suite run against each. Every
one was killed, by the test named:

| Mutant | Killed by |
|---|---|
| the busy test removed | `TestADefenderThatIsFightingDoesNotStepAway` |
| the cover filter's decider changed to the defender | `TestTheCoverFiltersPolarityIsTheSubjects` |
| the air preference always false | `TestTheCoverEngagementPrefersAirAndThenTheNearest` |
| the close order keeps its victim | `TestAnEscortOutOfRangeClosesOnItsSubject` |
| the zero-forcing dropped | `TestACrowdedEscortStepsAwayAlongTheLine` |
| the clamp's low bound dropped | `TestTheStepAwayIsClampedToThePlayableRectangle` |
| the crowding test off by one | `TestAFollowerReAimsWhenTheGapOpens` |
| the acquisition keeps the walk | `TestAFollowerThatScoresNothingEndsItsWalk` |
| the two arms unwired from the pass | `TestBothEscortStatesReachAnArmAndAcquireStillDoesNot` |
| the scan-range fallback dropped | `TestAnEscortRangeOfZeroTestsAgainstTheScanRange` |
| a missing subject not refused | `TestAnEscortOfAUnitTheWorldDoesNotHoldIsLeftAlone` |

## Two pins widened, and one renamed

`internal/archtest`'s `wantDestinationWriters` gains `escortClose` and
`escortStepAway`, each triaged in that file as a genuine originator: both give a
destination to a unit holding no victim, and both are protected from the group
decision by the explicit order check `armPatrol` already relies on, since the
escort setters put the group's order to none.

The same file's diff-driver subtests built their synthetic sources from literal
name lists and asserted literal counts. They now build from
`wantDestinationWriters` itself, so the next writer moves one place instead of
five.

`pkg/sim`'s `TestTheActorPassSwitchHasExactlyOneCaseAndNoDefault` is renamed
`TestTheActorPassSwitchHasNoDefaultArm` and no longer asserts the number of
cases. The claim D-1 makes is the absence of a default; the count was in the
assertion and in the name, and both had to be edited to add an arm.

## Two figures worth recording

Sixteen shipped nodes author sub-command 11 or 15 — six Defend and ten Follow —
which reproduces `TRIG-GRPARM-047`'s authored counts exactly. Of those, **one
Defend and five Follow** are named by a trigger, which reproduces that row's
reachable counts exactly as well. The instrument was a throwaway probe over all
28 campaign maps through this tree's own loader, run once and deleted; it is
recorded here because it is what decided which map witnesses this story.
