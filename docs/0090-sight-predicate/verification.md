# Verification — 0090 sight predicate

Windows 11, Go toolchain pinned by `go.mod`, worktree `wt-0090` forked from `e1a8df5`.
Every figure below is copied from the command that printed it.

## The submodule pin moved for this story

Master pins research at `96f0b15`; this branch pins `fe1c626`, which is research master and carries
the round that traced both of the predicate's input grids. Without it the two grids have no writer
and the region cannot be reproduced at all — which is why 0086 shipped without this term. The pin
moves in the same tree as the code that needs it, so a reader of either commit sees the source the
other rests on.

```
$ git submodule status research
 fe1c626c7af2e9f252f73065068468e615fc3397 research (heads/master)
```

No leading character: the index and the working tree agree.

## Gates

Each was run on its own and its own exit code read.

| Gate | Exit | Output |
|---|---|---|
| `go build ./...` | 0 | — |
| `go vet ./...` | 0 | — |
| `gofmt -l $(git ls-files '*.go')` | 0 | printed nothing |
| `go test -count=1 -trimpath ./...` | 0 | 31 packages ok, 0 FAIL |
| `bash scripts/check-no-game-assets.sh` | 0 | `check-no-game-assets: clean (tree scan)` |
| `bash scripts/check-no-game-assets.sh --history` | 0 | `check-no-game-assets: clean (history scan)` |
| `bash scripts/check-doc-budget.sh` | 0 | every artifact under its ceiling, both chain relations ok |
| `bash scripts/check-sdd-audit.sh` | 0 | 0 FAIL |

`check-sdd-audit`'s note and warning count is not recorded: its section is guarded on the top-level
`builds/` directory, so this lane emitted none until its own build stage created
`builds/0090-sight-predicate/`, at which moment every other story's gap appeared at once. Only the
FAIL set is comparable from here, and it is empty.

```
$ git diff --diff-filter=D --name-only e1a8df5 HEAD
$
```

The deletion set is empty.

## The contract, witnessed

| Id | Evidence |
|---|---|
| AC-1 | `TestEveryWindowOffsetHasACloserPredecessor` walks all 1 680 non-origin offsets; `TestWithoutTheBuilderRepairExactlyTwoOffsetsFail` re-executes the builder without its two axis repairs and finds exactly `(-1,0)` and `(1,0)` failing |
| AC-2 | `TestTheStepCostIsTheRaysMeanStep`: 128 on every axis offset, 181 on every diagonal, range 128..181 over the window |
| AC-3 | `TestTheFlatGroundRegionIsThePublishedDisc`: 9 / 21 / 45 / 69 / 105 / 145 at ranges 1..6, and 1 253 at 19 |
| AC-4 | `TestTheReachIsTheBudgetDividedByTheStep`: axis reach equals the range, diagonal reach equals the budget divided by 181, at every range 1..6 |
| AC-5 | `TestGroundBetweenTheObserverAndTheCellCostsSight` at the march; `TestARidgeStopsAnEngagementFlatGroundMakes` at the decision, the same pair acquiring across flat ground and not across a ridge |
| AC-6 | `TestWhereTheObserverStandsDecidesHowFarItSees`: 60 above its surroundings lights more than the flat figure, 60 below lights fewer |
| AC-7 | `TestABlockerIsNotPrunedAndTheMarginIsOneUnit`: two worlds differing in one byte, dark at 0 and lit at 128, plus a deep case showing the blocked cell's own value is what the next cell reads |
| AC-8 | `TestTheInsetStopsTheWalkAndTheObserverStillSeesItself` |
| AC-9 | `TestNothingWrapsAtTheEdgeOfAWideMap`: an observer at column 2 of a 256-wide world lights nothing at column 200 or beyond |
| AC-10 | `TestAGroupSeesAsOneAnimal`: the member 20 cells away takes the candidate it could not have lit itself, and a candidate no member sees is taken by neither |
| AC-11 | `TestTheSightPredicateReachesNeitherTheBytesNorTheDigest`: version 16 unmoved, byte form round-trips identical, hash unchanged |
| AC-12 | `TestSightBoundsTheCandidatePopulation`, unchanged from 0086 and still green at the range and one cell past it |
| P-1 | `internal/archtest`'s determinism scan over `pkg/sim`'s non-test files is green with `sight.go` in it — no float identifier, no float literal, no clock, no generator; `TestReplayReproducesTheRunsFinalDigest` and `TestTwoCrossingUnitsNeverShareACellAndReplayTheSame` still replay byte-identical |
| P-2 | `TestTwoDecisionsOnAnUnchangedWorldAgree`: two marches over an unchanged world light the same cells, and a second decision does not move the assignment |
| P-3 | `TestNoVisibleCellHasABlockedPredecessorOnShippedAltitudes`: 36 terrains over three observer heights, four wall heights and three lattice pitches, every byte at or below 127, no lit cell with a blocked predecessor |
| P-4 | `TestOnFlatGroundSightIsInsideTheDisk`: on a world naming no height plane every lit cell is inside the Chebyshev disk of the range |
| SC-1 | `TestTheIntegerStepCostIsThePublishedExpression`: the integer form equals the published float expression at all 1 681 window offsets; `TestTheIntegerSquareRootIsExact` pins the root at and either side of every square the table reaches |
| SC-2 | AC-3's figures, above |
| SC-3 | AC-11's evidence plus the whole suite green, no decoder arm added, no field added to any record |
| SC-4 | the milestone section below |

FR-7's two remaining halves are witnessed by `TestTheWalkNeverLeavesTheNineteenthRing` — a range-25
march reaching ring 19 and lighting nothing past it — and by `TestAnAllBlockedRingEndsTheWalk`.

## Mutations

Production code only, each run against `./...` rather than the package under change, each reverted
before the next.

| Mutation | Result |
|---|---|
| M1 the zone boundary `j < i>>1` becomes `j <= i>>1` | KILLED — `pkg/sim` |
| M2 the seed drops its half-cell rounding term | KILLED — `pkg/sim` |
| M3 a blocked cell's value is not stored, leaving its slot at zero | KILLED — `pkg/sim` |
| M4 the ring bound walks ring 20 as well | KILLED — `pkg/sim` |
| M5 the builder's two axis fix-up stores are dropped | KILLED — `pkg/sim` |
| M6 an all-blocked ring no longer ends the walk | KILLED — `pkg/sim` |
| M7 the inset test is dropped | KILLED — `pkg/sim` |
| M8 the height plane is read unsigned | KILLED — `pkg/sim` |

**The first battery left three survivors and that is the part worth recording.** M3, M4 and M6 all
had tests that claimed to cover them. M3's case was too shallow — both readings lit the cell behind
the blocker, so the seam the whole story turns on was untested. M4's test read the ring bound off the
constant it was pinning, so it moved with the mutation. M6 is only observable where the accumulator
can rise again, which needs a world whose first ring is entirely blocked and whose second holds an
authored pit. All three tests were rewritten and the battery re-run; the table above is the second
run, and every mutation in it typechecked.

## The milestone

Measured on both roots, in this worktree, before any edit and after the last commit.

```
$ AGAINROM_ASSETS=<ru> go test -count=1 -trimpath -run TestTheTenthMissionIsDrivenToAWin ./cmd/missionrun
```

| Root | Before (`e1a8df5`) | After |
|---|---|---|
| ru | `reached (44,46), Chebyshev 25, after 272 ticks` / `outcome lost at tick 272`, EXIT 1 | identical, EXIT 1 |
| en | `reached (44,46), Chebyshev 25, after 272 ticks` / `outcome lost at tick 272`, EXIT 1 | identical, EXIT 1 |

**It did not move, and the trace is identical to the cell and to the tick.** Nothing was tuned to try
to move it: the predicate was built from the claims, the drive was run once at the end, and this is
what it printed.

One thing was measured about it, because "identical" and "not reached" print the same. With
`groupSight` returning an empty stamp — nobody sees anything at all — the same drive **passes**. So
acquisition is what ends this mission, the predicate is on that path and is running, and the groups
that intercept can genuinely see their target under the engine's own rule as well as under the
Chebyshev disk that stood there before. That removes line of sight as the cause and leaves the other
inputs to the same decision: the sight *range*, which this build fixes at 5 for every unit where the
game reads it off each template (D-1); the relation the map authors; and the notice clip. The probe
was reverted immediately and is in no commit.

## Limitations

The corpus figures this story rests on — that no shipped altitude byte reaches 128, and the spread of
visible area over the shipped maps — are graded Medium at their source and are not re-measured here;
this tree holds no map. What is measured here is the arithmetic that makes them consequences: the
128-vs-127 margin is exercised in both directions on synthetic terrain.

The flat-ground figure at range 6 is contested between the two implementations of this algorithm in
the original, and research has not adjudicated it. Nothing built here depends on the answer: the
contract comes from the server side's own closed forms and 145 falls out of them rather than being
put in.

`builds/0090-sight-predicate/` holds `againrom.exe`, `missionrun.exe` and a README giving the exact
invocation against a lawful install, sourcing the asset root from `-assets`/`AGAINROM_ASSETS`.
