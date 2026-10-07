# Story `1037` — verification

**Candidate evidence, 2026-08-24.** Commands run from `wt-story-1037`.
Install-backed commands name the preserved EN or RU root explicitly.

## Requirement accounting

| Id | Evidence |
|---|---|
| FR-1 | `TestEveryMappedFieldHoldsItsOwnSlot`, `TestEachSlotHasExactlyOneOutcome`, sentinel and short-row tests. |
| FR-2 | Resolved placement and retained-Ghost tests; `TestReleaseWithdrawalPopulationAndMissionWitness` covers every definition and placement on EN and RU. |
| FR-3 | `TestEveryGroupAndActorDecisionReachesTheCommonWithdrawalTail`, `TestWithdrawalEligibilityAndHostileVisibilityPopulation`, the threshold table, and `DIV-351`/`DIV-352` for unrepresented dispatcher and health-width states. |
| FR-4 | `TestWimpyPrecedesWithdrawAndEachUsesItsOwnRadius`, including corpse-only priority; the living-mean and sight/invisibility/off-map-hostile rows. |
| FR-5 | Mean, zero-axis, four-clamp, state-clearing and blocked-route-substitute tests. |
| FR-6 | `TestPlayerCommandedMoveSucceedsOrSurvivesWithTheWithdrawalTail`, the six group orders, three actor states and failed-tail byte identity. `SESS-PARAM-017` identifies opcode `0x46`, parameter 3, mode at `cmd+0x0e`, player scope and exact writers `R0186`/`R0187`; `DIV-346` records the absent implementation session-command producer. |
| FR-7 | Current pin, predecessor peel, signed round trip, per-field digest changes, `TestForm60ResumesWithTheSameNextWithdrawalDecision`, the HP 65,566 no-wrap witness and explicit upgrade-loss tests. |
| FR-8 | Both-root release tests and direct mission-100 drives. The class census resolves EquipItem attack classification and asserts maximum HP 2,000 overall and 1,024 among threshold-active placements. |
| DD-1 | Entity binary offsets, per-field digest tests and the constructor/form-60 HP 65,566 witness. |
| DD-2 | Resolved placement, unresolved/Humans defaults and retained-Ghost tests. |
| DD-3 | The production `StepWithdrawalTraced` boundary and missionrun's immediate-before-tail ordinary decision. |
| DD-4 | Corpse-only Wimpy and multiple-living-hostile tests. |
| DD-5 | Distinct Wimpy/Withdraw radius tests and `DIV-347`. |
| DD-6 | Exact integer geometry, zero axes, clamps and `DIV-348`. |
| DD-7 | State-clearing and ordinary blocked-route substitute tests. |
| DD-8 | Form-60 offset, pin, round trip, digest, next-decision equality and form-59 upgrade tests. |
| DD-9 | Classdump and missionrun install-gated release tests on both roots. |

No `tasks.md` exists, so this table accounts for every FR and DD directly.

## Mutation evidence

Three focused production mutations were applied one at a time and removed
before the candidate was built:

- filtering `withdrawalHostiles` through `invisibleToActor` made
  `TestWithdrawalEligibilityAndHostileVisibilityPopulation/zero_sight_and_invisible_hostile_still_count`
  fail because the threshold-active actor kept no target;
- narrowing only `Entity.HP` to a signed WORD before the `Withdraw` comparison
  made `TestCanonicalInt32HealthDoesNotWrapAtTheOriginalSignedWordBoundary`
  fail: HP 65,566 retreated at threshold 65,565;
- replacing the resolved attack-classification predicate with `Weapon.Range >
  1` made `TestWithdrawalRangedClassificationReadsTheResolvedAttackNotReach`
  fail on a `SkillShoot` weapon at reach 1.

The existing call-site and geometry mutations remain discriminated: removing
the phase-6 tail call fails
`TestWithdrawalTailReplacesTheDecisionMadeOnTheProductionFullTick`, and changing
the three-cell stride fails
`TestWithdrawalUsesTheMeanOfEveryLivingHostileAndDecodedIntegerGeometry`.

## Observable result

Both lawful roots print the same mission-100 production-tick witness:

```text
withdrawal mission=100 map="scenario/100.alm" entity=109 class="Bat_Sonic.2" mode=wimpy threshold=4 radius=7 hostiles=1
  before-tail tick=6 cell=(129,62) target=none attack=108
  after  tick=7 cell=(130,61) target=(131,59)
```

The EN census is 56 parameterised definitions, 12 positive `Withdraw`, 24
positive `Wimpy`, 38 maps and 8,094 placements; 1,551 placements have positive
`Withdraw` and 3,464 positive `Wimpy`. RU has the same definition counts, 34
maps and 3,991 placements; 817 have positive `Withdraw` and 1,754 positive
`Wimpy`. All 12 positive definitions and every positive placement resolve
`SkillShoot`. Both roots report maximum HP 2,000 overall and 1,024 among
threshold-active placements.

## Gate record

The clean candidate runs this complete set:

- `go build ./...`, `go vet ./...` and `go test -trimpath -count=1 ./...`;
- `check-release-tests.sh` on EN and RU, selecting and passing 52 of 52
  install-gated tests with zero skips on each root;
- `check-scenarios.sh` on EN and RU, selecting and passing 15 of 15 scenarios
  on each root;
- `check-milestone.sh` against a candidate-built `missionrun` and the complete
  two-root baseline;
- `check-preserved-installs.sh` against all 162 recorded lawful-install files;
- `check-seat-tree.sh` against both the seat checkout and the candidate;
- `check-div-claims.sh` over all 244 live rows, `check-pin-forward.sh`,
  `check-claim-citations.sh` and `check-no-game-assets.sh` against the candidate;
- direct EN and RU `classdump -withdraw` and mission-100 `missionrun
  -withdrawal` drives producing the values above.

The gate outputs name the candidate checkout, commit and selected population;
the remote/tree verification after push records the same commit.
