# Verification — a blow leaves a number behind

Base `8892726`. Everything below was executed in the orchestrator's own tree; nothing is reported on
belief.

## The gate (P-5, SC-4)

Run on the untouched tree first, then after every task, then again after the last commit.

```
$ go build ./...                          EXIT=0
$ go vet ./...                            EXIT=0
$ gofmt -l $(git ls-files '*.go')         EXIT=0   (no output)
$ go test -count=1 -trimpath ./...        EXIT=0   (31 packages ok)
$ bash scripts/check-no-game-assets.sh    EXIT=0
$ bash scripts/check-doc-budget.sh        EXIT=0
$ bash scripts/check-sdd-audit.sh         EXIT=0
```

The exit codes are the scripts' own, taken from `$?` on the command itself and not through a pipe —
`cmd | grep FAIL; echo $?` reports **grep's** status, which is the trap this project has hit before.

## SC-2 — no simulation state was added (FR-13, AC-15, P-4)

```
$ git diff --stat 8892726..HEAD -- pkg/sim
(no output)
$ grep -n '^const formatVersion' pkg/sim/binary.go
115:const formatVersion = 14
$ git diff --diff-filter=D --name-only 8892726 HEAD
(no output)
```

`pkg/sim` has **no diff at all**, so the form version is untouched at **14** and no digest can have
moved. `pkg/sim/budget_test.go` already asserts that literal and still passes. The import-graph check
in `internal/archtest` still refuses `pkg/ui` any simulation type, so the records, the remembered
health and the display flag have nowhere to reach a world from.

**The deletion set is empty.**

**AC-15 was narrowed during T2 and the narrowing is the honest part.** As first written it asked for
an A/B schedule run with figures and without. That run is not reachable: the ingest lives in
`Viewer.step`, which is unexported, so the tier that owns a world cannot call it and the two legs
could not have differed in what the criterion is about. Reported as a criterion revised, not as one
met.

## SC-3 — the mutations, and what each killed

Ten one-line changes, each applied to the landed tree, tested, and reverted. **All ten were killed.**

| # | Mutation | Killed by |
|---|---|---|
| M1 | the health comparison from strict to non-strict | `TestAHealthThatDidNotFallMakesNothing/unchanged` |
| M2 | the merge appends a second record instead of adding in | `TestASecondBlowMergesRatherThanAppearingBeside`, `TestAMergedFigureExpiresOnTheFirstBlowsClock` |
| M3 | the life bound from `<=` to `<` | `TestTheLifeIsAThousandMillisecondsOfWallClock`, both sub-cases |
| M4 | the horizontal drift sign made unconditional | `TestAFigureDriftsOnePerTickSidewaysAndTwoUp/someone_else's_unit` |
| M5 | the drift driven by the frame rather than by the counter delta | `TestAFigureDriftsOnePerTickSidewaysAndTwoUp`, both sub-cases, and `TestASecondBlowMergesRatherThanAppearingBeside` |
| M6 | the display gate moved from creation to drawing | `TestTheDisplayGatesCreationAndNotDrawing`, `TestTheRememberedHealthFollowsEveryPush/a_blow_taken_with_the_display_off` |
| M7 | the mission tool's resolver stops checking the world | `TestADriveNamingAnAbsentUnitFailsBeforeAnyOrder`, both sub-cases |
| M8 | an absent victim reads as fallen (the shipped defect, restored) | `TestAnAbsentVictimNeverFalls` |
| M9 | the colour keyed on mine/not-mine rather than on the owner | `TestTheColourIsTheStruckUnitsOwners` |
| M10 | the birth offset dropped to zero | `TestTheBirthOffsetTakesTheOwnershipSign`, all four sub-cases, and `TestAFigureIsPlacedOffTheStruckUnitsOwnPosition` |

M6 is the one worth reading twice. Moving the gate to the draw kills a test about the **remembered
health**, not about drawing — because with the gate at the draw the memory would still be written
under a hidden display and the two failures are one defect seen from two sides.

M5's second kill is the same shape: a per-frame drift breaks a merge test that says nothing about
drift, because the merge assertion is that the offset did **not** move.

## SC-1 — the criteria, and where each is witnessed

Every AC below is a `go test` case in `pkg/ui/numeral_test.go` unless stated. The suite is synthetic:
no window, no clock, no world, no install.

| AC | Witness |
|---|---|
| AC-1 | `TestAStrictDecreaseMakesAFigureOfTheDifference`, `TestAHealthThatDidNotFallMakesNothing` |
| AC-2 | `TestTheRememberedHealthFollowsEveryPush`, `TestADeadUnitIsRememberedAndStillShowsItsBlow` |
| AC-3 | `TestTheDisplayGatesCreationAndNotDrawing` |
| AC-4 | `TestASecondBlowMergesRatherThanAppearingBeside`, `TestAMergedFigureExpiresOnTheFirstBlowsClock` |
| AC-5 | `TestTheColourIsTheStruckUnitsOwners` |
| AC-6 | `TestNoPropertyOfAFigureDependsOnTheVictimsRemainingHealth` |
| AC-7 | `TestTheLifeIsAThousandMillisecondsOfWallClock` (999 / 1000 / 1001 ms, with the ambient clock running and with it never advanced), `TestAClockThatWentBackwardsReapsNothing` |
| AC-8 | `TestAFigureDriftsOnePerTickSidewaysAndTwoUp`, `TestAFigureInFlightKeepsTheDirectionItSetOffIn` |
| AC-9, AC-10 | `TestTheBirthOffsetTakesTheOwnershipSign` (four cases: local, foreign, and both with no local participant established) |
| AC-11 | `TestCtrlLFlipsTheDisplayAndBareLDoesNot`, `TestTheNumeralKeyDoesNothingOffTheMapScreen` |
| AC-12 | `TestAFigureIsDrawnTwiceAShadowUnderAFace` |
| AC-13 | `TestAFigureIsPlacedOffTheStruckUnitsOwnPosition`, `TestAFigureCarriesTheUnitsOwnDisplacement`, `TestAFigureOverAnAbsentUnitPlacesNothingAndSurvives` |
| AC-14 | `cmd/missionrun`: `TestAbsentIsDistinguishableFromDowned`, `TestAnAbsentVictimNeverFalls`, `TestADriveNamingAnAbsentUnitFailsBeforeAnyOrder` |
| AC-15 | the diffs above |
| AC-16 | below |
| P-1 | `TestEveryOwnerValueTakesAColour` (0, 1, 7, 8, 9, 2^31, 2^32-1), `TestAClockThatWentBackwardsReapsNothing` |
| P-2 | by construction — a record is born from a strict decrease and merging only adds |
| P-3 | `pkg/ui/numeral.go` holds the only subtraction; `pkg/sim` has no diff |
| P-4 | the diffs above |

## AC-16 — measured against both lawful roots

**The instrument, before.** On master, the exact invocation the brief names:

```
$ missionrun -assets ..\gameversions\en -mission 10 -attack p0:p1        EXIT=0
mission 10  scenario/10.alm  80x80  36 entities
attack 1  p0 -> p1 : FELLED it after 1 ticks, victim at 0 hp, attacker facing N
outcome undecided at tick 65
```

The world holds 36 entities with ids 0..35; `p1` resolves to 36, which is nobody. It is reported as a
kill, at exit code **0**.

**The instrument, after.** Both roots, byte for byte:

```
$ missionrun -assets ..\gameversions\{en,ru} -mission 10 -attack p0:p1   EXIT=1
mission 10  scenario/10.alm  80x80  36 entities
missionrun: p1: this world holds no entity 36
```

**A real drive still answers exactly as it did.** 0081's own fixture, both roots:

```
$ missionrun -assets ..\gameversions\{en,ru} -mission 10 -ticks 4000 \
    -waypoint u28:79:79:200 -waypoint u29:79:79:200 -attack u28:u29      EXIT=0
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u28 -> (79,79) r200 : reached (69,51), Chebyshev 28, after 1 ticks
waypoint 2  u29 -> (79,79) r200 : reached (64,46), Chebyshev 33, after 1 ticks
attack 1  u28 -> u29 : FELLED it after 128 ticks, victim at -1 hp, attacker facing NW
outcome undecided at tick 194
```

Four lines identical to 0081's on both roots, so nothing in the fix moved a real measurement — and
the victim is at **-1 hp**, held by the world and dead, which is the state the old reader could not
tell from absence.

## What is NOT claimed

- **Nothing here was looked at.** No screenshot, no window, no statement that it looks right. What is
  established is what the code does: which predicate answers when, what value is composed, and what
  the placement seam receives. The looking is the owner's.
- **The two sounds** of `ANIM-SND-022` are not built and were never begun. This tree has no audio
  layer of any kind, which is why they are declared absent rather than approximated.
- **The severity colour is not built, and its absence is asserted** rather than left to inspection
  (AC-6). Three writes and zero reads in the original; a consumer that rendered one would add what
  the original discards.
- **Four decoded things have no counterpart here**, each named in the code at the constant it would
  move:
  - `vt+0x20()`, the per-unit scalar the birth offsets are multiplied by — taken as **1**, which is
    what makes them exactly `(±16, -48)`;
  - the **player colour table**, thirty-two bytes per index — the rule (colour is a function of the
    struck unit's owner) is reproduced, the values are ours;
  - the **shadow's offset and colour** inside the glyph blit — the second issue is decoded, the two
    numbers are ours;
  - the **second half of the merge key**, a value taken from an undecoded global — this keys on the
    victim alone.
- **The suppression flag** (`drawable+0x78`) has no counterpart and none was invented.
- **Two things are ours because this tree has a camera and the original has not**: the offset scales
  with the zoom and the glyph does not. Neither reproduces anything.
- **A figure follows any health decrease, not only a blow.** The seam carries state and not events,
  so a chip key produces one too. Accepted: the original's own notification is named "take damage"
  and has five producers, of which the melee strike is one (R-1).

## Criteria revised during the pass

Two, both recorded rather than quietly fixed.

- **AC-10** stated a consequence the ownership comparison does not have. With no local participant
  established, an *unowned* victim compares equal to the zero and reads as the local participant's;
  an owned one does not. Amended to that, and it is what the four-case test asserts.
- **AC-15** asked for an experiment that cannot be run (above).

## The rebase

`0084` landed while this story was finishing, so the branch was rebased and the **whole** gate re-run
on the result — an auto-merge succeeding is not evidence that two stories composed.

```
$ git rebase origin/master
Successfully rebased and updated refs/heads/impl/0083-damage-numerals.
$ git log --oneline -1 origin/master
c49d5b8 bump the research pin to b2876c0 at the 0084 story boundary
```

That story touched `pkg/sim/step.go`, `pkg/ui/readout.go`, `pkg/ui/viewer.go` and `pkg/game/world.go`,
and it moved the research pin. Three things were therefore redone rather than argued about:

1. **The whole gate**, in the order at the top of this file. All seven exit `0` on the rebased tree;
   the deletion set against the **new** merge base is empty; `pkg/sim` still has no diff from this
   story and the version literal still reads `14`.
2. **The submodule.** The rebase left `git submodule status research` showing a **leading `+`** —
   index and working tree disagreeing, which reads research at the *previous* pin while claiming the
   current one. `git submodule update --init research` cleared it; it now shows **no leading
   character** at `b2876c0`. The four claims this story is built on — `ANIM-BLOW-019`,
   `ANIM-NUM-020`, `ANIM-NUM-021`, `ANIM-SND-022` — are **still `● active`** at that pin and none
   appears in `retracted.md`.
3. **The lawful-install measurement below**, re-run on both roots after the rebase. `0084` changed
   the step-cost arm in the determinism package, which is exactly the kind of change that moves a
   tick count; the four lines came back **identical**, which is a measurement and not an expectation.

The two stories touch `pkg/ui/viewer.go` in different places — `0084` adds one field, this adds three
fields and two call sites — and `pkg/ui/readout.go` is untouched here.

## The run note

`builds/0083-damage-numerals/` holds `againrom.exe` and `missionrun.exe` built `-trimpath` from this
branch, with a `README.md` naming the invocation and the five things to watch for. It is untracked
and gitignored, as every build directory is.
