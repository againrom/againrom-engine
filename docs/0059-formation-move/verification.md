# Verification — 0059-formation-move

Threshold **High**: the group rate term reaches hashed simulation state, so a clause that
is wrong here changes what every replay and every digest means. Everything below was run
in this worktree at `impl/0059-formation`.

## The gate

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"
```

`EXIT=0`. The **FAIL set is empty** and byte-identical to the baseline taken on this
branch before the first edit (`14651da`, also `EXIT=0`, zero `FAIL` lines). Note and
warning counts are not compared: a worktree checks out no `builds/`, so
`check-sdd-audit` emits one warning per story that has a `verification.md` and no build
README, and this tree is missing every one of them.

`git diff --diff-filter=D --name-only 14651da..HEAD` is **empty** — no file was deleted.
`git submodule status` reads `cf68f4d…` with no leading character throughout.

## Witnesses

| id | witnessed by | result |
|---|---|---|
| **AC-1** | `TestAThreeUnitRowKeepsItsShape` — three units at (4,8) (5,8) (6,8) ordered to (40,40) take (39,40) (40,40) (41,40) and come to rest on exactly those; `TestAGroupOrderedToOneCellKeepsItsShape` (`pkg/game`) over four units on four cells | pass |
| **AC-2** | `TestTheCentroidIsTheMeanRoundedToTheNearestCell` — eight rows including an exact half up and an exact half at a negative, a third either side of an integer, and a one-member row; `TestOneFlagGatesBothTheDistributionAndTheTerm` re-measures it as four-cells-in / five-cells-out | pass |
| **AC-3** | `TestOneFlagGatesBothTheDistributionAndTheTerm` — one fixture, one changed gap, distribution and term asserted in the same statement and required to agree | pass |
| **AC-4** | `TestAGroupDestinationIsClampedIntoTheMap` (both ends of both axes, and one axis at a time); `TestADistributedDestinationIsClamped` (the case only a distribution creates — the ordered cell is on the map and the offset takes a member off it) | pass |
| **AC-5** | `TestAFormationCrossesAtItsSlowestMembersRate` — speeds 8 and 40, both members at the slow one's 32-tick transit in formation, 32 and 7 out of it; `TestTheGroupTermReplacesTheClassSpeedInEveryDomain` over all three domains | pass |
| **AC-6** | `TestTheSurvivorsKeepADeadMembersRate` — the slow member killed one tick into the walk, the survivor's transit asserted at the dead member's 32 for four transits' worth of ticks, then a plain order restoring its own | pass |
| **AC-7** | `TestArrivingDoesNotClearTheGroupTerm`; `TestTheGroupTermSurvivesARoundTripAndDistinguishesTwoWorlds` (marshal, decode, compare, then walk at the term); the hand-transcribed pins carry terms of 19 and 250 and decode back | pass |
| **AC-8** | `TestUnmarshalRefusesEveryVersionButEight` (all 256 values); `TestUnmarshalRefusesAndLeavesTheReceiverExactlyAsItWas` gains three group-term rows — killed below zero, downed at zero, and on a dead unit holding no target; `TestTheConstructorNormalisesATermOnAUnitThatIsNotAlive` | pass |
| **AC-9** | `TestAGroupOfOneIsTheIdentity` at three positions; `TestTheGroupOrderHasNoMemberCountBranch` reads `group.go` and requires exactly the two accounted-for mentions of the member count | pass |
| **AC-10** | `TestAGroupOrderedToOneCellKeepsItsShape`, `TestOnePressIsOneGroupOrderAndTwoPressesAreTwo`, `TestAGroupOrdersDigestFollowsAHeadlessRunAtEveryTick` (all `pkg/game`) | pass |
| **P-1** | `internal/archtest`'s source scan over `pkg/sim`'s non-test files, green in the run above. Every term of this story is integer arithmetic; the centroid is `int64` and the two narrowings are conversions | pass |
| **P-2** | `TestNoTermIsStillTheClassSpeed`; the five headless-mirror runs in `pkg/game` (`cadence`, `drawn`, `order`, `stopped`, `world`) compare a driven world against a hand-built stream at every tick and agree | pass |
| **P-3** | The whole suite is green and every changed test is listed below with its reason | pass |
| **SC-1** | AC-1's two tests | pass |
| **SC-2** | Sixteen mutants, table below | 15 killed, 1 equivalent |
| **SC-3** | AC-5's `TestAFormationCrossesAtItsSlowestMembersRate` | pass |
| **SC-4** | AC-6 and AC-7's tests together | pass |
| **SC-5** | AC-8's tests, plus `TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth` (the offset table is a partition of the whole form, so a widening fails it) and `TestMarshalledBytesArePinned` | pass |
| **SC-6** | P-1 | pass |
| **SC-7** | The gate above, and the FAIL-set comparison | pass |
| **SC-8** | The changed-test table below | pass |

## Mutation testing — the arithmetic that reaches the digest

Each mutant was applied to one line, `go test ./pkg/sim/ ./pkg/game/` run, and the tree
restored. Killers are named by their first failing test.

```
M1  centroid: drop the half-cell (a truncating mean)    KILLED TestTheCentroidIsTheMean… (+3)
M2  centroid: floor -> Go's truncating divide           KILLED TestTheCentroidIsTheMean…
M3  spread: > becomes >=                                KILLED TestOneFlagGatesBoth… (+1)
M4  spread: the threshold 2 -> 3                        KILLED TestOneFlagGatesBoth… (+1)
M5  offset: drop the byte narrowing                     KILLED TestTheFormationOffsetNarrows…
M6  offset: the byte read UNSIGNED (the rival)          KILLED TestAThreeUnitRowKeeps… (+5)
M7  minimum: initial value 250 -> 255                   KILLED TestMarshalledBytesArePinned (+3)
M8  minimum: strict < becomes <=                        EQUIVALENT — see below
M9  minimum: drop the 16-bit narrowing                  KILLED TestTheGroupTermIsTheMinimum…
M10 clamp: the high end off by one                      KILLED TestAGroupDestinationIsClamped…
M11 one flag: distribute unconditionally                KILLED TestOneFlagGatesBoth… (+3)
M12 one flag: store the term unconditionally            KILLED TestOneFlagGatesBoth… (+1)
M13 the read: ignore the group term                     KILLED TestAFormationCrossesAt… (+4)
M14 felling no longer unlinks the member                KILLED TestTheSurvivorsKeepA… (+1)
M15 a plain order no longer allocates a fresh group     KILLED TestTheSurvivorsKeepA… (+1)
M16 duplicates counted twice                            KILLED TestADuplicateNamingIsCountedOnce
```

**M8 is equivalent, not a survivor.** The running minimum is a `uint8` and the comparison
is against it zero-extended, so two speeds that compare equal at sixteen bits have equal
low bytes; re-storing on equality writes the value that is already there. No test can
separate `<` from `<=` because no execution can.

**M5 was a real survivor and is the reason T7 exists.** Nothing an order can be given
reaches the offset's widths — the spread gate admits displacements in `[-2, +2]` and both
narrowings are transparent over all five — so the clause was extracted into
`formationOffset` and exercised over a table that crosses both widths. The mutant dies
there now.

## Tests changed, and why

| Test | Change | Reason |
|---|---|---|
| `TestAGroupOrderedToOneCellArrivesOneUnitDeep` (`pkg/game`) | **replaced** by `TestAGroupOrderedToOneCellKeepsItsShape` | It asserted that a group sent to one cell arrives ONE UNIT DEEP, with the rest abandoning the order — behaviour its own comment recorded as a divergence that was *"OURS AND WORSE"*. That divergence is what this story removes. Four members now take four cells and every one of them arrives; the fixture, the shared-cell invariant and the tick budget are unchanged, and no assertion was dropped to make it pass |
| `TestEveryNoOpBlowLeavesTheWorldWhereAQuietTickLeavesIt`, `TestAnUndefinedKindIsIgnoredRatherThanReadAsAMoveTo` | kind `3` → `4` | Kind 3 was their example of a kind this build does not define, and 0059 defines it. 4 is the lowest byte with no arm today; both keep every assertion |
| `TestUnmarshalRefusesEveryVersionButSeven` | renamed `…ButEight`, version 7 added to the refused list | The name moves with the number so the two cannot come apart; the sweep still covers all 256 values |
| Record-width arithmetic in `binary_test`, `gridform_test`, `relaxation_test`, `routeform_test`, `domain_test`, `nostate_test` (`pkg/sim`) and `gridform_test` (`pkg/mapload`) | `43` → `44`, second-record offsets `+1` | The record gained a byte. Every one of these is written out rather than computed from `entityLen`, deliberately, so that a widening has to be acknowledged here |
| `pinDigest`, `rtfDigest`, `rlxTick1Digest`, `hybTick1Digest` (`pkg/sim`), `gfDigest` (`pkg/mapload`) | re-derived | Each was recomputed **outside this tree** with a third FNV-1a implementation checked against the published vectors first: the two `pkg/sim` byte pins from their own hand-written literals, the two relaxation digests and `gfDigest` by re-assembling the form from the prose that describes it. None was captured from the encoder — that is what those pins are for. Version 8 moves byte 0, so no value carries over by arithmetic |
| `preStoryDigest`/`preStoryFormLength` and the structure-pass `pinDigest` (`pkg/mapload`) | **re-recorded**, with the previous values kept in the comment | These are recordings by construction, as their own comments say. The form widened by one zero byte per record and the version byte moved; this loader writes no group term |
| Five headless mirrors (`cadence`, `drawn`, `order`, `stopped`, `world` in `pkg/game`) | gained `grouped()`, a second implementation of the click boundary | The front-end now issues group orders, so a mirror built from single move-tos would compare two different streams. The scripted half of each stream is deliberately left untagged: a schedule entry is not an order |
| `nostate_test`'s field-set pin | `GroupSpeed` declared | Declaring it is the statement that the term is canonical state; the pin's whole function is to refuse a field nobody declared |

## Figures, and how each was obtained

Every number in this file was measured in this worktree by the command it sits beside.
The mutation table is sixteen runs of `go test`; the digests are five derivations run
outside the tree; the transit lengths (32 at speed 8, 7 at speed 40) are computed by the
tests from `rateOf`/`transitOf` rather than written down, so they cannot drift from the
rate law. Nothing here was estimated.
