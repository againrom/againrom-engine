# Verification — unit pathfinding over a static passability grid

Task commits, oldest first: `7991e09` (T1), `d53f7f4` (T2), `3c0a61a` (T3), `f0517b1` (T4),
`5d2d778` (T5), `3253688` (T6), `2570571` (T7), `4132a0a` (T8), `3161461` (T9), `dd0572e` (T10),
`cbaa759` (T11). Base `eeee466`. Nothing else sits between them: the range holds exactly eleven
commits, all this story's, each id once. Submodule pin frozen at research `ce40c15` throughout.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11, measured 2026-07-30. Every criterion is a
unit test over synthetic worlds built in test code — **no test reads a game install.** Every figure
below was run in this seat, T1–T4's four mutants included: they were measured once by the executor
that landed them and are re-measured here rather than carried over on report.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output; 227 files)
$ go test -trimpath -count=1 ./...           (0 FAIL, 0 SKIP; 742 tests, 2358 counting
                                              subtests, 25 packages ok, 3 without tests)
      pkg/sim 120 tests / 179 counting subtests, of which 47 are in this story's ten new files
      pkg/sim diff eeee466..HEAD: 24 files, +3950 -446
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0029-unit-pathfinding
  analysis.md 5966 / 7168     provenance.md 7118 / 7168
  spec.md 13296 / 13312       plan.md 13312 / 13312
  tasks.md T1..T11 all under 1400 (1101 880 1118 1271 638 899 1051 623 669 707 972)
  legend+traceability 1163 / 1200
  plan <= 1.2 x spec ok (13312 <= 15955); tasks <= 1.2 x plan ok (11092 <= 15974)
$ sh scripts/check-sdd-audit.sh              at cbaa759, before this file
FAIL 0029-unit-pathfinding: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED                      ONE enforced FAIL, naming this story; 43 unenforced
                                             note/warn lines on 0000-0013 and on 0019/0023/0025's
                                             untracked builds/ dirs
$ sh scripts/check-sdd-audit.sh --story 0029-unit-pathfinding
check-sdd-audit: 11 trailered commit(s) for 0029-unit-pathfinding in ac6bd87..HEAD checked
$ git log --format='%(trailers:key=SDD-Task,valueonly)' eeee466..HEAD | sed '/^$/d' | sort | uniq -c
      1 0029-unit-pathfinding/T1 ... 1 0029-unit-pathfinding/T11   (11 ids, each exactly once)
$ git log --format='%B' eeee466..HEAD | grep -ci co-authored-by
0
```

That FAIL is what a story with every task landed and no evidence file looks like. This file clears
it and carries no trailer.

## Witnesses

Every id and the named thing that answers for it. All in `pkg/sim` unless said otherwise.

```
FR-1 AC-8 P-5  SC-2   world_test.go TestNewWorldRefusesAMalformedGridOrMode (a short grid, a
      long one, a reserved bit in the first cell, in the last, every reserved bit, and two
      undefined mode bytes — each its own case, each returning no world);
      TestNewWorldStoresTheModeAndTheGridItWasGiven; gridform_test.go TestUnmarshalRefuses-
      AndLeavesTheReceiverExactlyAsItWas — the receiver's WHOLE byte form compared before and
      after each refusal, not its digest: a wrong cell count, a reserved bit (three cases), an
      undefined mode byte, a stall count at the limit, a count on an entity with no target,
      and a version-2 form.
AC-6 AC-7  SC-1   gridform_test.go TestTheGridAndTheModeAreCanonicalState — a non-trivial
      grid against one ground bit toggled, one AIR bit toggled, and the other mode, the three
      differing pairwise in form and digest; the no-grid and all-zero-grid pair equal BYTE FOR
      BYTE and by digest. binary_test.go pins every offset and width at header 34 / record 26;
      hash_test.go recomputes the version-3 digest from its own pinned bytes.
FR-3 AC-1 AC-3 P-1  SC-3   wall_test.go TestTheFixtureFitsBothModesBounds (budget 13 for the
      crossing, region the whole map), TestAUnitCrossesTheWallThroughItsGapUnderBothModes,
      TestATargetOffTheMapIsHeldThenGivenUpUnderBothModes (two off-map targets x two modes,
      16 ticks each). checkP1 runs over every unit after every tick of both — in bounds, ground
      bit clear, alone on it. route_test.go TestAnotherUnitIsNotEnterable, TestAStartOnABlocked-
      CellIsRoutedOffIt, TestAStartOutsideTheBoundsIsRoutedOntoTheMap.
FR-5  SC-9   relaxation_test.go TestTheRelaxationOrderDecidesThisRoute (the 5x5 route and six
      labels), TestTheTickOneDigestIsTheContractsRoutesFirstCell (digest derived by hand from
      the format's layout through FNV-1a's published constants), TestTheWalkNeverStandsOnThe-
      BlockedCell; the 8x8 second grid TestTheStaleSourceLabelWouldCostThisRoute, TestThe-
      SecondGridsTickOneDigest, TestTheSecondGridIsWalkedCellByCell. route_test.go's twelve
      cases carry the wave itself: the two-frontier-cells fixture, the asymmetric accept read
      off the labels, all three stop conditions, the scratch's reuse, the step cost and the
      budget.
FR-4 P-4   step_test.go TestStepWalksOneCellPerTickAndClearsTheTargetOnArrival; stall_test.go
      TestEachTickHasExactlyOneOutcome — each of sixteen ticks classified as exactly one of
      advance / hold / give up, 15 holds and 1 give-up. Sampled, not proved.
FR-6 AC-5  SC-6   optimised_test.go TestTheRouteMayNotCutACorner (against the wave cutting it
      on the same grid), TestATargetReachableOnlyOutsideTheRegionIsRefused (the same wall with
      its gap one column inside), TestTheRegionIsClippedFromAnInt64Rectangle, TestATargetOff-
      TheMapIsNoRoute, TestAStartOnACellItMayNotStandOnIsRoutedOffIt, TestTwoEqualCostRoutes-
      AreSeparatedByTheOrderTheContractNames (both costs read off the plane), TestThe-
      OptimisedSearchLeavesTheScratchReusable. mode_test.go TestOneWorldDrivenByBothModes is
      AC-5, the two worlds asserted to differ in exactly ONE byte of the form before either
      advances.
FR-7 AC-2 P-2  SC-4   stall_test.go TestASealedOrderIsHeldFifteenTicksAndGivenUpOnThe-
      Sixteenth (all three target fields and the count read one at a time at each of sixteen
      ticks, and each tick's world marshalled and read back), TestAStalledTickWritesTheCount-
      AndNothingElse (the whole entity compared across each tick with the count alone
      excused), TestAdvancingReturnsTheCountToZero (the advance deliberately not an arrival),
      TestAClearedTargetTakesTheCountWithIt.
FR-8 AC-4 P-3  SC-5   arrival_test.go TestTwoCrossingUnitsNeverShareACellAndReplayTheSame —
      both modes, 20 ticks, no shared cell at any tick, a second run equal entity for entity
      at every tick, and both units asserted across. replay_test.go TestADecodedWorldStepsOn-
      ToTheSameDigests — both modes, 20 ticks, the twin built ONLY by decoding, over a
      receiver sharing neither bounds, mode, grid nor units; digest and every entity field
      compared before and after each step; the run asserted to contain both an arrival and a
      give-up. TestTheStallCountReachesTheDigest — the digest moves on each of fifteen ticks
      where nothing but a count changed. mode_test.go TestTheModeIsReadFromTheWorldAnd-
      NowhereElse, TestTheResolutionOrderIsAscendingIdUnderBothModes.
AC-10  SC-8   arrival_test.go TestAUnitArrivesAfterTheChebyshevDistanceUnderBothModes — six
      displacements x two modes, one cell per axis per tick, arrival on exactly max(|dx|,|dy|).
AC-9  SC-7   optimality_test.go TestTheOptimisedWalkCostsTheAdmissibleMinimum (six grids) and
      TestTheCanonicalWalkCostsMoreThanTheMinimumOnItsOwnGrid (36 over 12 ticks against 32
      over 16). The minimum comes from an enumeration in that file with its own bounds test,
      blocking bit, step costs, margin and admissibility rule.
DD-1 DD-2 DD-3 DD-4 DD-5 DD-6 DD-7 DD-8 DD-9   carried by the tasks that built them; every
      one has code and cases above.
SC-10  the ten mutants, all re-run in this seat — below.
```

## The mutants

All ten applied to a production file, run against the **whole** tree and reverted; every touched
file sha256-identical either side, at the value HEAD carries. Nine were expected to die and did; the
tenth is a declared survivor.

```
M1  T1  world.go — the reserved-bit refusal deleted
    KILLED, 2 tests / 4 assertions, pkg/sim. TestNewWorldRefusesAMalformedGridOrMode;
    TestUnmarshalRefusesAndLeavesTheReceiverExactlyAsItWas in all three of its reserved-bit
    subtests (first cell, last cell, every reserved bit).
M2  T2  world.go — the absent grid left unmaterialised, so it encodes as zero cells
    KILLED, 3 packages, each aborting on a PANIC rather than a clean failure: the grid a
    no-grid world holds is then empty and the first index into it is out of range.
    pkg/sim TestUnmarshalReplacesTheWholeReceiver (index 714), pkg/mapload TestSteppingA-
    LoadedWorldLeavesTheMapUnchanged (index 2), pkg/game TestASessionsPickAndQueueReachNo-
    WorldBeyondTheOrder (index 24). A panic ends its package's run, so those three are the
    first failures per package and not the whole list — the kill is certain, its width is not.
M3  T3  route.go — the budget's max(5, D>>2) term dropped, leaving D
    KILLED, 8 tests, pkg/sim. TestTheGenerationBudgetIsChebyshev, TestADetourInsideThe-
    BudgetIsFound, TestATargetOutsideTheBudgetIsNoRoute, TestASealedTargetIsNoRoute,
    TestTheScratchCarriesNothingBetweenSearches, TestStepRoutesRoundAHeldDiagonal,
    TestTheSecondGridIsWalkedCellByCell, TestTheFixtureFitsBothModesBounds.
M4  T4  route.go — the batched wave (see the disclosure below on how it was rendered)
    KILLED, 6 tests, pkg/sim. TestTwoFrontierCellsReachingOneNeighbour, TestTheRelaxation-
    OrderDecidesThisRoute, TestTheTickOneDigestIsTheContractsRoutesFirstCell, TestTheStale-
    SourceLabelWouldCostThisRoute, TestTheSecondGridsTickOneDigest, TestTheSecondGridIs-
    WalkedCellByCell.
M5  T5  route.go 295b0974... — the hybrid wave: a snapshot source label, the minimum still
        kept in the plane
    KILLED, 3 tests, pkg/sim, all three of them the 8x8 grid's: TestTheStaleSourceLabel-
    WouldCostThisRoute, TestTheSecondGridsTickOneDigest, TestTheSecondGridIsWalkedCellBy-
    Cell. The 5x5 grid's three cases pass under it, exactly as the plan predicted — no
    frontier cell there is lowered by an earlier cell of its own frontier.
M6  T6  step.go 9c1edd01... — the stall reset on advance deleted
    KILLED, 1 test tree-wide: TestAdvancingReturnsTheCountToZero.
M7  T7  optimised.go 2b7a70e1... — the heap key's (y, x) terms dropped
    SURVIVED. Whole tree green, 0 FAIL. This is the declared survivor and the measurement
    IS the discharge: Dijkstra's distances do not depend on which of two equal-cost cells
    is popped first, so DD-8's claim that the two terms are inert holds against every case
    in this tree. No variant that kills was hunted.
M8  T8  optimised.go — the corner-cut refusal deleted
    KILLED, 7 tests, pkg/sim. TestTheRouteMayNotCutACorner, TestATargetReachableOnly-
    OutsideTheRegionIsRefused, TestAStartOnACellItMayNotStandOnIsRoutedOffIt, TestTwoEqual-
    CostRoutesAreSeparatedByTheOrderTheContractNames, TestTheOptimisedSearchLeavesThe-
    ScratchReusable, TestOneWorldDrivenByBothModes, TestTheModeIsReadFromTheWorldAndNowhere-
    Else.
M9  T9  optimised.go — the region's growth set to 0
    KILLED, 4 tests, pkg/sim. TestATargetReachableOnlyOutsideTheRegionIsRefused, TestThe-
    RegionIsClippedFromAnInt64Rectangle, TestTheFixtureFitsBothModesBounds, TestAUnit-
    CrossesTheWallThroughItsGapUnderBothModes.
M10 T10 route.go — the extraction's straight accept narrowed to <
    KILLED, 3 tests / 2 packages, and NONE of them T10's own. pkg/sim TestStepWalksOneCell-
    PerTickAndClearsTheTargetOnArrival/diagonal_until_the_short_axis_arrives,_then_straight;
    pkg/game TestAnOrderedUnitWalksToItsCellAndTheDigestFollowsAHeadlessRun and TestA-
    CommandedUnitTakesNoFurtherScriptedTarget. See the disclosure below.
M11 T11 route.go — the extraction's diagonal accept widened to <=
    KILLED, 4 tests, pkg/sim. TestAnotherUnitIsNotEnterable, TestTheStaleSourceLabelWould-
    CostThisRoute, TestTheSecondGridsTickOneDigest, TestTheSecondGridIsWalkedCellByCell.
```

## Disclosures

- **The batched wave does not terminate as the contract's own append rule renders it.** Written
  faithfully — a snapshot source label, each offer tested against the snapshot, the last write
  winning, and a neighbour appended on *every* accepted offer — the next frontier multiplies by the
  accepting-neighbour count each generation, because with last-write-wins a cell is appended even
  when nothing improved. On two landed fixtures (`TestUnmarshalReplacesTheWholeReceiver`, whose
  target is at (30,30), and a `pkg/mapload` case) the run had not finished after five minutes; under
  `-timeout 40s` both packages panic inside `canonicalRoute`. M4 above was therefore run with the
  next frontier **deduplicated**, which changes no label: in a batched wave every entry for a cell
  relaxes from the same snapshot value, so duplicates are inert. The kill list is that run's. The
  non-termination is itself a fact about the rejected shape and is recorded here rather than in
  `plan.md`, which is not amended.
- **M10 was killed by no case T10 added.** T10's own criteria are SC-5 and SC-8 — arrival ticks, the
  no-shared-cell invariant, and a decoded twin compared against its original. None of the three can
  see a straight-accept change: the mutant picks a different predecessor of equal cost, so the route
  is a different walk of the same length, the twin is mutated identically, and the arrival tick is
  unmoved. The kill is real and tree-wide, and it came from T4-era and `pkg/game` fixtures that pin
  cells rather than counts. No test was added or altered to make the entry kill its own mutant.
- **`spec.md`'s I/O example names an optimised route the contract refuses, and the spec is not
  amended here.** For a unit at `(0,2)` ordered to `(4,2)` with `(2,1)`, `(2,2)`, `(2,3)` blocking
  ground, it gives `(1,1) (2,0) (3,1) (4,2)` at cost 12. The step `(1,1) -> (2,0)` passes between
  `(2,1)` and `(1,0)`, and `(2,1)` is blocked, so FR-6 refuses it; so does `(2,0) -> (3,1)`. That
  grid is one of the six in `TestTheOptimisedWalkCostsTheAdmissibleMinimum`, where the independent
  enumeration and the walk agree at **14**, not 12. FR-6 is the contract and the example is
  illustrative, so nothing built is wrong — but the example is, and correcting it is a spec revision
  and not this file's to make.
- **AC-1's arrival tick is not pinned.** The criterion asks that the unit arrive and never stand on
  a blocked cell, and that is what is asserted, with a floor of the Chebyshev distance and a ceiling
  of 24 ticks. AC-10 pins an exact tick count where the criterion asks for one.
- **P-1, P-3 and P-4 are sampled, not proved,** as `spec.md`'s own verification mapping says: P-1
  over the two `wall_test.go` runs, P-3 over the decoded pair, P-4 over the sealed sixteen ticks.
- **DD-2's three measured figures are not reproduced here.** `plan.md` owes `verification.md` the
  148k random 5x5, 33k random 9x9 and 12305 dense 16x16 searches behind "no other scan order changes
  a route". Those sweeps were run when the plan was written and were not re-run in this seat, so
  nothing below claims them; the scan order is fixed for reproducibility and no criterion in this
  story claims to discriminate it.
- **`stallLimit = 16` is ours and provisional.** It was 0032's give-up tick count and has never been
  measured against a real tick rate. It is stated as an engineering choice in `provenance.md`, and
  nothing here presents it as a fact about the game.
- **Nothing was run against a game install.** The manual pass over a real map belongs to whatever
  story first draws a unit walking; this one ships a headless binary and a note.

---

# The revision — the two fixes, and what each was worth

Commits, oldest first: `2356237` (plan and tasks), `f599be0` (T12), `9e2cf48` (T13), `26477bd`
(T14), `8b0d4d4` (a third shape), `2830594` (T15). Base `59189b8`; six commits, four trailered, each
id once. Pin frozen at research `a13b3b8`. Go 1.26.1, Windows 11, i7-12700K, measured 2026-07-31.
No test reads a game install; no new binary ships.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output; 232 files)
$ go test -trimpath -count=1 ./...           (0 FAIL, 0 SKIP; 774 tests, 2502 counting
                                              subtests, 25 packages ok, 3 without tests)
      pkg/sim 129 tests / 188 counting subtests
      pkg/sim diff 59189b8..HEAD: 9 files, +932 -56
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0029-unit-pathfinding
check-doc-budget: 0029 runs under a DECLARED OVERRUN (plan 16384, verification prose 10240)
  analysis.md 5966 / 7168 (unchanged)      provenance.md 7118 / 7168 (unchanged)
  spec.md 13296 / 13312 (unchanged)        plan.md 15879 / 16384
  tasks.md T1..T15 all under 1400 (T12 1140, T13 1034, T14 909, T15 806)
  legend+traceability 1200 / 1200          verification.md 9209 / 10240 prose
  plan <= 1.2 x spec ok (15879 <= 15955)   tasks <= 1.2 x plan ok (14201 <= 19054)
$ sh scripts/check-sdd-audit.sh --story 0029-unit-pathfinding
check-sdd-audit: 15 trailered commit(s) for 0029-unit-pathfinding checked
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 59189b8..HEAD | sed '/^$/d' | sort | uniq -c
      1 0029-unit-pathfinding/T12 ... 1 0029-unit-pathfinding/T15   (4 ids, each once)
$ git log --format='%B' 59189b8..HEAD | grep -ci co-authored-by
0
```

## SC-12 — the cost, one harness, four trees

`go test -trimpath -run '^$' -bench Benchmark -benchtime=1x ./pkg/sim/`, milliseconds per tick, one
iteration each, at four commits of this revision with **the same harness file** — T15's
`grouporder_test.go`, copied unaltered into a worktree at each earlier one, which it compiles against
because it names no symbol whose signature moved. The four columns are one measurement.

```
                                     T12 base   T13 index   T14 early-out   HEAD
group ordered to a far cell
  128x128    1 unit                    0.842      0.737        0.716        0.736
             5 units                   5.344      3.290        3.154        3.160
            10 units                  12.90       6.717        6.324        6.526
            20 units                  33.06      13.12        12.72        12.73
            40 units                 104.0       25.99        25.45        25.69
  256x256    1 unit                    2.828      3.190        2.659        2.727
             5 units                  21.34      14.32        12.16        13.27
            10 units                  52.28      25.51        24.53        24.73
            20 units                 134.8       49.98        48.24        48.06
            40 units                 435.9      106.0        101.3        102.3
80 movers, a goal sealed by units    231.0       27.00        24.60        25.35
80 movers ordered onto a HELD cell   183.9       22.63         0.0163       0.0130
```

The owner's case, forty units on 256x256: **435.9 -> 102.3 ms a tick, 4.3x**, and the curve is now
linear in the group size where it was superlinear — 2.6 ms a unit at twenty and at forty, against 6.7
and 10.9 before.

The two are separable and not alike. **DD-10 moved the first two shapes**, 4.1x and 8.6x; DD-11 adds
2-4% there, which at one iteration is noise. **DD-11 moved the third**, 22.63 -> 0.0163, **1389x**
and 11290x over the base — the only one of the three whose destination cannot be entered, which is
the ordinary end of the first shape and not an accident of the fixture: the leading member arrives,
and every other then spends sixteen full sweeps finding the cell taken.

## SC-11 — the fixes changed nothing they were not meant to change

```
the oracle      enterable_test.go TestTheIndexedPredicateAnswersWhatTheWalkAnswered — the pre-
      change linear scan kept as entNaive and compared with the indexed predicate for EVERY unit
      as the asker and every cell of the bounds grown by 2, over 400 generated worlds and after
      each unit of each is walked to a fresh cell, which is the mid-tick state Step resolves in.
      The corpus is asserted to have produced a shared cell, a unit off the map and a blocked
      cell, so the awkward answers are measured and not merely available.
      TestAReOccupiedScratchAnswersForTheWorldItWasPointedAt — the counts belong to ONE world's
      entities, so a scratch pointed at a second replaces them rather than adding to them.
the digests     preserved_test.go TestTheseRunsCarryTheDigestsTheyCarriedBeforeTheSearchWasMade-
      Cheaper — three runs of twenty ticks recorded at 59189b8 and pinned as literals: four units
      ordered as a group across a wall, with one walking a clear line and one ordered onto the
      wall itself, in each mode; and five ordered onto a cell a sixth is standing on. Compared
      tick by tick, never at the end. TestTheseRunsContendStallAndGiveUp asserts each run holds an
      advance, a hold and a give-up.
the witness     route_test.go TestTheThreeRefusalsAreToldApartByWhatWasLabelled — the three
      no-route paths separated by the scratch's own touch list and frontier: an unenterable
      destination labels nothing and leaves no frontier, a sealed map labels cells and runs the
      frontier out, a spent budget labels cells and leaves the frontier holding work.
      TestAStartOnItsOwnTargetIsStillTheEmptyRoute — the case the refusal excuses by name.
the fixtures    grouporder_test.go TestTheGroupFixtureIsActuallyWalking, TestTheSealedFixtureIs-
      SealedAndItsGoalIsEnterable, TestTheHeldFixtureIsHeldAndNobodyEverMoves — each benchmark's
      claim about its own world, asserted rather than assumed.
```

## The mutants

Both applied to a production file, run against the **whole** tree and reverted; `step.go` and
`route.go` sha256-identical either side.

```
M12 T13  step.go — the occupancy move on advance deleted
    KILLED, 9 tests / 2 packages. pkg/sim TestStepGivesAContestedCellToTheLowerID (all three
    subtests), TestStepKeepsEveryUnitOnACellOfItsOwnThroughAnNWayConvergence, ...ThroughASeeded-
    Sweep, TestStepAdvancesALowIDLedConvoyAsOneBody, TestTheResolutionOrderIsAscendingIdUnderBoth-
    Modes, TestTwoCrossingUnitsNeverShareACellAndReplayTheSame, TestTheseRunsCarryTheDigestsThey-
    CarriedBeforeTheSearchWasMadeCheaper; pkg/game TestAGroupOrderedToOneCellArrivesOneUnitDeep.
    Rendered as `_ = from` rather than as a deletion, which would not compile.
M13 T14  route.go — the early refusal removed
    KILLED, 1 test tree-wide: TestTheThreeRefusalsAreToldApartByWhatWasLabelled.
```

## Disclosures

- **M13 is killed by one test and can be killed by no other.** The refusal is an identity — the same
  answer, at the same point, the same stall raised — so nothing behavioural can see it. The witness
  observes the WORK instead, through a touch list the scratch already kept. Delete that test and the
  fix is unmeasured, not merely unmutated.
- **M12 was not killed by the oracle test,** the test written for this defect: it drives the index
  itself rather than through `Step`, so a `Step` that stops calling it is invisible there. The kill
  comes from the contention fixtures and the pinned digests — the shape M10's disclosure recorded.
- **The pins witness preservation, not correctness:** a defect `59189b8` had is reproduced with the
  rest.
- **Single-iteration figures carry real variance.** `-benchtime=1x` keeps a base column at twenty
  seconds rather than minutes; the 256x256 one-unit row reads 2.83 / 3.19 / 2.66 / 2.73 across the
  four trees, which is scatter. Nothing is claimed from a difference under about 10% — hence DD-11's
  2-4% on the first two shapes being called noise, and the two shapes that existed at T12 reading
  104.0 against 105.8 and 231.0 against 215.3 under the two versions of the harness.
- **`plan.md` and this file run under declared overruns; `spec.md` does not move,** both fixes being
  behaviour-preserving. The chain relation was not waived and bound throughout: 15879 of 15955.
- **The third cause of the collapse is untouched.** A route is recomputed from scratch every tick and
  all but its first cell thrown away; the ~2.6 ms a unit a tick that remains is that. It is an
  architectural divergence — the original runs an expensive static search rarely and a cheap dynamic
  one every transit — and belongs to `0037-approach-and-path`. Nothing here makes it harder: both
  searches keep their shape, the scratch is a parameter and not a field, and a per-tick occupancy
  plane is what a two-tier scheme needs anyway.
- **A bounded search window was considered and is not here.** As a BOUND it refuses a goal reachable
  only by a detour bulging past the margin, which a lake or a ridge makes ordinary. As a FAST PATH it
  changes no reachability, but neither can it help the shape it was aimed at, a sealed goal being a
  failing search that falls through to the full sweep. Not implemented, not measured; it belongs with
  the two-tier scheme above.
