# Verification — two searches: one per order over the whole map, one per step inside a window

Task commits, oldest first: `5e3d5e6` (T1), `6e31741` (T2), `4a6986d` (T3), `3d10c08` (T4),
`952d442` (T5), `c87a1aa` (T6), `64bcb33` (T7), `157d9e9` (T8), `17c556d` (T9), `c46be66` (T10).
Base `130831f`. Nothing else sits between them: ten commits, all this story's, each id once.
Submodule pin frozen at research `a13b3b8` throughout.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11, i7-12700K, measured 2026-07-31. Every
criterion below is a unit test or a benchmark over synthetic worlds built in test code — **no test
reads a game install**, and no binary ships. Every figure was run in this seat.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output; 239 files)
$ go test -trimpath -count=1 ./...           (0 FAIL, 0 SKIP; 804 tests, 2396 counting
                                              subtests, 25 packages ok, 3 without tests)
      pkg/sim 159 tests / 232 counting subtests, of which 39 are in this story's five new files
      pkg/sim diff 130831f..HEAD: 28 files, +3410 -680
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0045-two-tier-search
  analysis.md 6470 / 7168      provenance.md 6817 / 7168
  spec.md 12796 / 13312        plan.md 12854 / 13312
  tasks.md T1..T10 all under 1400 (992 965 981 1045 1288 916 856 779 884 1034)
  legend+traceability 999 / 1200
  plan <= 1.2 x spec ok (12854 <= 15355); tasks <= 1.2 x plan ok (10739 <= 15424)
$ sh scripts/check-sdd-audit.sh              at c46be66, before this file
FAIL 0045-two-tier-search: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED                      ONE enforced FAIL, naming this story
$ sh scripts/check-sdd-audit.sh --story 0045-two-tier-search
check-sdd-audit: 10 trailered commit(s) for 0045-two-tier-search checked
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 130831f..HEAD | sed '/^$/d' | sort | uniq -c
      1 0045-two-tier-search/T1 ... 1 0045-two-tier-search/T10   (10 ids, each exactly once)
$ git log --format='%B' 130831f..HEAD | grep -ci co-authored-by
0
```

That FAIL is what a story with every task landed and no evidence file looks like. This file clears
it and carries no trailer.

## Witnesses

Every id and the named thing that answers for it. All in `pkg/sim` unless said otherwise.

```
FR-1 DD-1   relation_test.go TestTheTwoRelationsDifferExactlyOnOccupancy (six cells x three
      predicates: a cell a unit stands on is open to terrain and closed to the other),
      TestTheWaveRunsOverTheRelationItIsGiven, TestTheOptimisedSearchRunsOverTheRelationItIs-
      Given (all four combinations hand-computed on a 3x3, corner rule included),
      TestWithNoOtherUnitOnTheMapTheTwoArmsAgreeCellForCell (1200 seeded worlds, both modes,
      asserted to hold routes AND refusals).
FR-2 AC-1 DD-2  SC-4   wall_test.go TestTheDetourLeavesAnyStartTargetRectangleGrownByEight
      (the fixture's own claim: the gap misses the rectangle by twelve columns, the wall is a
      wall, the wave's budget is 50 for a walk of 40), TestAUnitWalksToAGoalNoRectangleReaches
      (both modes, arrival, P-1 at every tick). optimised_test.go TestATargetReachableOnlyBy-
      ADetourRightAcrossTheMapIsRoutedTo — forty-two cells pinned one at a time, and the same
      search WITH a window refused on the same world.
FR-3 AC-5 AC-6 DD-3 DD-4 DD-5  SC-1   routeform_test.go: the version-4 form of a routed world
      transcribed by hand and its digest computed outside this tree (TestARoutedWorldMarshals-
      ToItsPinnedBytes, TestThePinnedRoutedDigestIsFNV1aOfThePinnedBytes, TestThePinnedRouted-
      BytesDecodeBackToTheRoutedWorld, TestADecodedRouteIsTheRouteThatWasWritten); AC-5's four
      worlds pairwise apart in form and digest (TestARouteIsCanonicalStateInFormAndDigest);
      AC-6's ten refusals, each its own subtest with the receiver's whole form compared either
      side (TestUnmarshalRefusesAMalformedRouteAndLeavesTheReceiverAsItWas), and the two forms
      that must NOT be refused beside them. TestAWorldWithNoRouteAndOneWhoseRoutesAreEmpty-
      AreOneWorld. world_test.go TestAWorldBeginsWithNoRouteForAnyUnitWhateverBuiltIt,
      TestAnEntityHandedOutReachesNothingOfTheWorld. nostate_test.go's pin table carries
      routes. binary_test.go re-pins every offset at version 4 and refuses a well-formed
      version-3 stream; hash_test.go's four route cases move the digest.
FR-3  SC-3   routefork_test.go TestAStoredRouteIsNotRecoverableFromItsOwnLaterCells — 107208
      (route, cell) pairs over 11520 seeded worlds, 70 tails differing, 57 refusing outright.
FR-4 FR-5 FR-6 AC-8 DD-6  SC-7   subgoal_test.go TestTheSubGoalIsTheCellAtIndexThreeUntil-
      FewerRemain, TestAStoredRouteThatServesIsAskedForNoFarSearch (a stored route the far
      search would not have returned, so using it and replacing it are different worlds),
      TestAMoverThatLandsOnItsRoutesThirdCellDropsThatCellAndEveryCellBefore (five landings),
      TestEachStalenessTestSendsTheTickBackToTheFarSearch, TestAReTargetedMoverRunsOneFar-
      SearchAndKeepsNothingOfTheOldRoute, TestAMoverThatFailedToMoveRunsNoFarSearchOnTheTick-
      After, TestAMoverThatBeginsOnItsTargetLosesItsRouteWithItsTarget.
FR-4 P-1   route_test.go TestAWindowedSearchLabelsNoCellOutsideItAndNoMoreThan289 (three
      fixtures: the optimised flood fills the window at exactly 289, and the wave against a
      sealed sub-goal at each of the two distances a sub-goal can lie at), TestTheNearSearches-
      BudgetCarriesTheSmallerSlack, optimised_test.go TestTheWindowIsAChebyshevSquareOnThe-
      Mover (int64 extremes, and noWindow admitting them).
FR-7 AC-2 AC-3 DD-7  SC-5   giveup_test.go TestAnUnservableOrderIsClearedOnTheFirstTick
      (three unservable orders x two modes x sixteen ticks each, every tick's world marshalled
      and read back), TestAGroupOrderedOntoAHeldCellWalksToItCrowdsAndGivesUp (eight movers,
      80 ticks: each advances, none enters the goal, each gives up on its sixteenth stalled
      tick aiming at a cell another unit stands on, and each ends within five cells of it).
      stall_test.go's four cases, rebuilt around a mover boxed in BY UNITS.
FR-8 AC-4  SC-6   counted_test.go TestANearSearchOnALargeMapLabelsAtMost289CellsAndNoneOutside-
      ItsWindow and TestOverAWholeRunTheFarSearchesAreTheOrdersPlusTheDepartures — the table
      below.
FR-8 AC-10  SC-9   grouporder_test.go's four benchmark shapes and their three fixtures.
FR-9 AC-7 AC-9  SC-2 SC-8   routetrip_test.go TestEveryCutOfEveryRunDecodesAndWalksOnIdentically
      (twelve runs, both modes, every one of nineteen cuts decoded and walked to the end, the
      WHOLE form compared at every tick), TestTheCorpusForcesDetours (56 arrivals later than a
      clear line). replay_test.go TestADecodedWorldStepsOnToTheSameDigests, now comparing the
      routes cell for cell. arrival_test.go TestAUnitArrivesAfterTheChebyshevDistanceUnderBoth-
      Modes, TestTwoCrossingUnitsNeverShareACellAndReplayTheSame.
P-2   every tick of every run in routetrip_test.go and giveup_test.go is marshalled and
      decoded, and the decoder refuses exactly P-2's four conditions — so a stored route that
      ended anywhere but on its target, left the bounds, crossed ground blocking or skipped a
      neighbour would fail there. Sampled, not proved.
P-3   stall_test.go TestAStalledTickWritesTheCountAndNothingElse — the whole entity compared
      across each tick with the count alone excused, and the route compared cell for cell.
P-4   stall_test.go TestEachTickHasExactlyOneOutcome — four outcomes over two runs, 15 holds
      and 1 give-up in the first, 1 order ended by a far failure in the second.
P-5   routetrip_test.go, as above: equal forms advanced by equal commands, compared as forms.
DD-8  SC-9   the harness's second window, and the table below.
SC-10   nine mutants, all run in this seat — below.
```

## SC-9, AC-10 — the cost, one harness, two trees

`go test -trimpath -run '^$' -bench Benchmark -benchtime=1x ./pkg/sim/`, milliseconds per tick,
one iteration each, at `130831f` and at `c46be66` with **the same harness file** — T10's
`grouporder_test.go`, copied unaltered into a worktree at the base, which it compiles against
because it names no symbol whose signature moved. The two columns are one measurement.

```
                                      before      after     FR-8's ceiling
group ordered to a far cell
  128x128    1 unit                    0.7071     0.08635
             5 units                   3.159      0.2520
            10 units                   6.336      0.4179
            20 units                  13.03       0.8161
            40 units                  25.46       1.692        <= 10   ok
  256x256    1 unit                    2.692      0.2378
             5 units                  13.37       0.8297
            10 units                  24.55       1.604
            20 units                  48.85       3.114
            40 units                 102.8        6.082        <= 25   ok
  256x256   40 units, 256 ticks       90.95       0.5380        <=  4   ok
80 movers, a goal sealed by units     26.97       3.128         <= 12   ok
80 movers ordered onto a HELD cell     0.01259    3.304         <= 12   ok
```

The owner's case, forty units on 256x256: **102.8 -> 6.08 ms a tick, 16.9x**, and over 256 ticks
**90.9 -> 0.54, 169x** — the ratio between those two says what the arrangement bought. The order
tick still pays one whole-map sweep per unit ordered (R-1), which is why the sixteen-tick figure is
eleven times the long-run one; every tick after it pays a windowed search per mover.

The counted criteria are machine-independent and are reported separately, from `counted_test.go`:

```
near search, 256x256, unit mid-order   canonical    81 cells labelled, reach 4
                                       optimised   196 cells labelled, reach 8
far search, same world, same moment    canonical 63504 cells labelled, reach 246
                                       optimised 65536 cells labelled, reach 250
8 units on 64x64, run to arrival       8 far searches: 8 first orders, 0 re-targets,
                                       0 departures; 456 near searches over 456
                                       unit-ticks, every unit arriving
```

## The mutants

Nine applied to a production file, each run against the **whole** tree and reverted; every touched
file byte-identical either side, at the value HEAD carries.

```
M1  T1  route.go — the terrain arm made to read occupancy
    KILLED, 3 tests, pkg/sim. TestTheTwoRelationsDifferExactlyOnOccupancy, TestTheWaveRunsOver-
    TheRelationItIsGiven, TestTheOptimisedSearchRunsOverTheRelationItIsGiven. NOT killed by the
    1200-world corpus in the same file, and it cannot be: with one unit on the map the two
    relations are the same predicate.
M2  T2  route.go — noWindow set to 8, so the far search carries the near search's bound
    KILLED, 14 tests / 2 packages. pkg/sim TestAUnitWalksToAGoalNoRectangleReaches, TestATarget-
    ReachableOnlyByADetourRightAcrossTheMapIsRoutedTo, TestAWindowedSearchLabelsNoCellOutside-
    ItAndNoMoreThan289, TestTheWindowIsAChebyshevSquareOnTheMover, TestTheGroupFixtureIsActually-
    Walking, TestTheOptimisedWalkCostsTheAdmissibleMinimum, TestTheCanonicalWalkCostsMoreThanThe-
    MinimumOnItsOwnGrid, TestTheseRunsCarryTheDigestsTheyCarriedBeforeTheSearchWasMadeCheaper,
    TestTheseRunsContendStallAndGiveUp, TestStepMutatesTheWorldInPlace; pkg/game TestEntityZero-
    WalksItsFirstLeg, TestAnOrderedUnitWalksToItsCellAndTheDigestFollowsAHeadlessRun, TestAn-
    AdvanceReportsHowManyOrdersItApplied, TestACommandedUnitTakesNoFurtherScriptedTarget.
M3  T4  binary.go — the route section written as a count of zero, its cells dropped
    KILLED, 3 tests, pkg/sim. TestARoutedWorldMarshalsToItsPinnedBytes, TestThePinnedRouted-
    DigestIsFNV1aOfThePinnedBytes, TestARouteIsCanonicalStateInFormAndDigest.
M4  T5  step.go — the sub-goal taken as the route's LAST cell (subGoalIndex = n-1)
    KILLED, 14 tests / 2 packages. pkg/sim TestTheSubGoalIsTheCellAtIndexThreeUntilFewerRemain,
    TestAStoredRouteThatServesIsAskedForNoFarSearch, TestAUnitWalksToAGoalNoRectangleReaches,
    TestTheGroupFixtureIsActuallyWalking, TestTheSealedFixtureIsSealedAndItsGoalIsEnterable,
    TestTheHeldFixtureIsHeldAndItsMoversWalkToIt, TestTheOptimisedWalkCostsTheAdmissibleMinimum,
    TestTheCanonicalWalkCostsMoreThanTheMinimumOnItsOwnGrid, TestTheseRunsContendStallAndGiveUp,
    TestStepMutatesTheWorldInPlace; pkg/game, the same four as M2.
M5  T6  step.go — a far failure made to stall instead of ending the order
    KILLED, 6 tests, pkg/sim. TestAnUnservableOrderIsClearedOnTheFirstTick, TestATargetOffTheMap-
    EndsTheOrderOnTheFirstTickUnderBothModes, TestEachTickHasExactlyOneOutcome, TestOneWorld-
    DrivenByBothModes, TestTheModeIsReadFromTheWorldAndNowhereElse, TestStepIsClampedByTheBounds.
M6  T7  step.go — the last-cell staleness test dropped
    KILLED, 4 tests / 2 packages. pkg/sim TestEachStalenessTestSendsTheTickBackToTheFarSearch,
    TestAReTargetedMoverRunsOneFarSearchAndKeepsNothingOfTheOldRoute; pkg/game TestSchedule-
    EntryIsAppliedAtItsOwnTickAndNothingPastTheEnd, TestACommandedUnitTakesNoFurtherScripted-
    Target.
M7  T8  step.go — the route left behind on give-up (clearTarget instead of clearOrder)
    KILLED, 2 tests, pkg/sim: TestAGroupOrderedOntoAHeldCellWalksToItCrowdsAndGivesUp,
    TestABoxedInOrderIsHeldFifteenTicksAndGivenUpOnTheSixteenth.
M8  T9  route.go — the window test deleted (window.holds always true)
    KILLED, 5 tests, pkg/sim. TestANearSearchOnALargeMapLabelsAtMost289CellsAndNoneOutsideIts-
    Window, TestAWindowedSearchLabelsNoCellOutsideItAndNoMoreThan289, TestTheWindowIsAChebyshev-
    SquareOnTheMover, TestATargetReachableOnlyByADetourRightAcrossTheMapIsRoutedTo, TestEach-
    StalenessTestSendsTheTickBackToTheFarSearch.
M9  T10 route.go — the near search's budget slack moved from 3 to 5
    KILLED, 2 tests, pkg/sim, and it was DECLARED A SURVIVOR. See the disclosure below.
```

## Disclosures

- **The declared survivor is not one.** SC-10's ninth mutant was to run green and discharge DD-2's
  claim that "the window is the binding bound". It is killed by `TestTheNearSearchesBudgetCarries-
  TheSmallerSlack`, which pins FR-4's two slacks, and by `TestAWindowedSearchLabelsNoCellOutsideIt-
  AndNoMoreThan289`, which measures how far a near flood actually reaches. **DD-2's claim is false
  as measured**: with a sub-goal four cells off the budget is `max(3, 4>>2) + 4 = 7` and the window
  reaches 8, so the BUDGET binds first and the window binds only when the sub-goal sits at its edge.
  Nothing behavioural moved under the mutant — no route, digest or outcome in the tree changed, and
  that much of "inert" holds — but the two bounds are not ordered the way the plan says. No variant
  was hunted and no test was weakened to let it live; the fixture that catches it was written at T2,
  eight commits before the mutant was run, for a different criterion.
- **Two landed contracts now answer differently, and both are deliberate.** A group ordered onto an
  occupied or sealed cell walks there and crowds: `0026`'s AC-3 shape — a mover whose destination is
  held by another unit — now advances instead of standing still whenever it is more than four cells
  away, since the far search reads terrain and cannot see the unit on the cell. And an order the
  terrain does not serve is cleared on the first tick: `0029`'s **AC-2** (fifteen holds, cleared on
  the sixteenth) and **AC-3** (an out-of-bounds target cleared on the sixteenth) are now first-tick
  answers, and its FR-7 splits in two. The give-up count inherited from `0032` is unchanged as a
  number and now applies to near failures alone. `0029`'s AC-1, AC-4 to AC-10 are unaffected.
- **AC-3's "stalls only once it can get no closer" holds in a weaker sense than its wording.** A
  near search aims at its own sub-goal and at nothing else, so a mover whose sub-goal has been taken
  by the crowd ahead of it stalls where it stands, which can be three or four cells short of the
  nearest free cell — substitute destinations being a non-goal. What is asserted instead is that a
  mover gives up aiming at a cell another unit is standing on. Two movers in an early fixture also
  stalled transiently in open ground for the same reason, and walked on.
- **AC-11 and SC-11 were not run.** They need a lawful install and a window; nothing here reads one,
  and the brief for this stage skipped the `builds/` step. The frame-rate and lake claims are
  therefore untested, and the timed table above is the only performance evidence.
- **The held-cell benchmark is 262x SLOWER**, 0.0126 to 3.30 ms a tick. That is R-2 realised: the
  shape used to be eighty movers refusing before any sweep, and it is now eighty movers walking
  across the map. It is the behaviour the story wants and it reads as a regression next to the last
  revision's figure.
- **`preserved_test.go`'s digest tables are retired, not re-recorded.** They were recorded from the
  tree before 0029's optimisation and were a preservation witness; this story deliberately changes
  what those very runs do, so re-pinning them from the tree they measure would have kept the shape
  and thrown away the meaning. T4 kept them alive across the version bump by projecting each tick's
  world back onto the version-3 form; T5 dropped them when routes began to be stored. The runs and
  their contend/hold/give-up condition remain, and `routetrip_test.go` is a stronger instrument over
  the same ground.
- **Four entries touched a file outside their own list, and each was a signature or a contract
  following through.** T1 and T2 opened `step.go` only to pass the new parameters through
  `searchRoute` — nothing called the terrain arm or carried a window until T5, which is what those
  fences protect. T4 opened `relaxation_test.go` and `preserved_test.go`, whose digests are derived
  from the byte form the entry changed. T5 opened `grouporder_test.go` and `preserved_test.go`, and
  T6 `mode_test.go`, `step_test.go` and `wall_test.go`: each held a landed assertion the new
  contract contradicts, and the tree cannot be green with one left standing.
- **The three benchmark fixtures were rewritten at T5, not T10.** DD-8 assigns two of them to T10;
  T5 is the commit whose contract change contradicted them, and a red tree cannot be pushed. T10
  added the tick parameter and the fourth shape as planned.
- **Single-iteration figures carry real variance.** `-benchtime=1x` keeps the base column at
  half a minute rather than minutes. Nothing is claimed from a difference under about 10%.
- **P-1 to P-5 are sampled, not proved**, as `spec.md`'s own verification mapping says.

## The revision — the third clause `MOVE-TERM-003` gives

Task commits, oldest first: `875c183` (T11), `d810357` (T12), `bd5a877` (T13), `857dc44` (T14),
`a191987` (T15). Base `b040859`, the 0047 boundary. Five commits, each id once. Pin `f35be34` throughout — not the `a13b3b8` above; `UNIT-OWNER-009` and
`MOVE-TERM-003`'s amendment are **at** it and were re-read from the submodule (`analysis.md`,
`provenance.md`, each appended to). Environment as above, 2026-07-31.

### Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output; 268 files)
$ go list ./... | wc -l
28                                           25 packages ok, 3 without tests
$ go test -trimpath -count=1 ./...           (0 FAIL, 0 SKIP; 941 tests, 2841 counting subtests)
      pkg/sim 184 tests, of which 8 are in this revision's two new files
      pkg/sim diff b040859..HEAD: 12 files, +997 -160
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh             0045 under its DECLARED OVERRUN
  analysis.md 9083 / 9216       provenance.md 10186 / 10240
  spec.md 17533 / 18432         plan.md 20477 / 21504
  tasks.md T11..T15 785 992 1486 997 1305, all under 1600
  verification.md 9215 / 9216 prose
  plan <= 1.2 x spec ok (20477 <= 21039); tasks <= 1.2 x plan ok (16470 <= 24572)
$ sh scripts/check-sdd-audit.sh              the FULL sweep, not --story: it was RED before this
                                             file, on AC12 AC13 AC14 / SC12 SC13 SC14 SC15, and
                                             on nothing else in the tree
check-sdd-audit: 170 trailered commit(s) in ac6bd87..HEAD checked
check-sdd-audit: ok (34 note(s)/warning(s), none enforced)
$ git log --format='%(trailers:key=SDD-Task,valueonly)' b040859..HEAD | sed '/^$/d' | sort | uniq -c
      1 0045-two-tier-search/T11 ... 1 0045-two-tier-search/T15   (5 ids, each exactly once)
$ git log --format='%B' b040859..HEAD | grep -ci co-authored-by
0

The whole chain exits 0.
```

### Witnesses

```
AC-12 SC-12  FR-10   channel_test.go TestTheChannelFixtureAssertsItsOwnClaim — the wall checked
      blocked above the way round and open below it, the target terrain-open, and the way round
      62 hops against a straight line of 39: 23 rings off that line where max(5, 39>>2) allowed
      9. TestOneOrderCarriesAUnitAcrossTheChannelInBothModes (ONE order; arrival at tick 60
      canonical, 62 optimised, against a floor of 60; checkP1 every tick; the order asserted
      never given up, which is the defect itself). TestUnderTheOldAllowanceTheChannelRefusesIn-
      CanonicalAndArrivesInOptimised — the old budget DRIVEN, not described: the scaled rule
      reproduces the far search's old 48 exactly here, the wave refuses under it, the optimised
      search arrives. TestARouteOverAThousandRingsIsStillRefusedInCanonical is FR-10's stated
      residual: the corridor's 1228 hops, found in optimised mode and refused in canonical.
AC-13 SC-13  FR-11   counted_test.go TestWhatBoundsARefusedFarSearchIsNowTheMap — labelled
      cells off the scratch's own touch list, at both revisions, one 256x256 map, start (4,4),
      goals sealed by TERRAIN (the shipped sealed shape is sealed by units, which a terrain-only
      search cannot see):
                                     scaled    flat
        a sealed goal, near  (D 8)      306   65527
        a sealed goal, far   (D 246)  65527   65527
        a search that succeeds        63001   63001   <- asserted EQUAL; FR-11's central claim
      grouporder_test.go TestTheTerrainSealedFixtureIsUnreachableAndClearsEveryOrderAtOnce: the
      goal open, ringed by terrain, unoccupied and near; every order cleared on the FIRST tick
      with no move and no stall. Benchmarks, -benchtime=1x, ms/tick against ceiling:
        128x128/40 over 16     1.752 / 10      256x256/40 over 16     6.215 / 25
        sealed (units) 80      3.277 / 12      held cell 80           3.389 / 12
        256x256/40 over 256    0.574 /  4
        FIFTH: 256x256/40 to a NEAR terrain-sealed cell   6.181 / 25
               worst single tick 97.76 ms — the order tick, forty whole-map sweeps at once
AC-14 SC-14  FR-12   budget_test.go TestTheRecordedRunDiffersFromTheFirstTick — eight digests
      recorded from the tree BEFORE the budget moved, pinned as literals and never regenerated,
      asserted different at every tick; the unit reaches (8,8) holding 52 route cells where it
      stood at (0,0) with its order cleared. TestTheByteFormsVersionIsUnmovedByThisRevision.
      Every other form and digest pin in the package re-runs green: nostate_test.go,
      binary_test.go, hash_test.go, gridform_test.go, routeform_test.go, malformed_test.go.
FR-10 FR-12 DD-9 DD-10   budget_test.go TestTheFlatBudgetReachesAGoalTheScaledOneCannot — one
      3x9 world, one start, one target, two rules: refused under the scaled rule's 5 generations,
      16 hops under the flat one, so the refusal was never the map's. TestTheFlatBudgetIsFlat
      (the same 1000 at five distances, and the D=900 crossover where the scaled rule would
      allow MORE, which FR-10 states rather than smooths away). route_test.go TestTheScaled-
      BudgetIsChebyshev, TestTheNearSearchesBudgetIsStillTheScaledOne, and TestAWindowedSearch-
      LabelsNoCellOutsideItAndNoMoreThan289's reach-7 case four cells off — the near search's
      budget still pinning a REACH, not the window's 289.
DD-11  SC-15   routefork_test.go, same corpus, same shapes, same seed:
        budget    worlds   routes    pairs   tails differ   refused
        scaled     11520     9072   107208             70        57
        flat       11520     9117   108240             40         0
      The refusals go to zero, which DD-11 predicted by construction and which is now reported
      rather than required. The differences do NOT — they rise, because the flat budget finds 45
      routes the scaled one refused — and a zero there would have been a finding against FR-3.
DD-9 FR-10   route_test.go TestATargetOutsideTheBudgetIsNoRoute and TestTheThreeRefusalsAreTold-
      ApartByWhatWasLabelled's budget-spent row, both driven from the serpentine corridor: 1228
      hops between two cells 58 apart, refused under both budgets, found in optimised mode, and
      asserted to leave a non-empty frontier so it measures the budget stop and not the sealed
      map's. wall_test.go TestTheFixtureFitsBothModesBounds and TestTheDetourLeavesAnyStart-
      TargetRectangleGrownByEight now assert their detours' own hop counts, 10 and 42, in place
      of an allowance — the clause "the wave's budget is 50 for a walk of 40" is gone.
```

### Mutants

Both SC-14's, applied to production code, run tree-wide, reverted, byte-identity confirmed.

```
T13  the flat budget given to the NEAR search too — generationBudget made to answer flat for
     BOTH rules. KILLED by 5 tests. SC-14's named killer fired: TestAWindowedSearchLabelsNoCell-
     OutsideItAndNoMoreThan289, "four cells off ... the flood reached 8 cell(s) out, want 7" —
     the reach assertion, not the 289-cell bound, exactly as SC-14 said. Also TestTheFlatBudget-
     ReachesAGoalTheScaledOneCannot, TestTheFlatBudgetIsFlat, TestTheScaledBudgetIsChebyshev,
     TestTheNearSearchesBudgetIsStillTheScaledOne.
T14  the flat budget put back to the scaled form on the far search — step.go's chooser. KILLED
     by 2 tests: TestOneOrderCarriesAUnitAcrossTheChannelInBothModes ("the order was given up at
     tick 1", which is AC-12's channel and the owner's own report), and TestTheRecordedRun-
     DiffersFromTheFirstTick, which reproduced all eight pre-change digests EXACTLY — independent
     evidence that those literals were recorded honestly from the tree before the change.
SURVIVOR (not planned, found on the way): the same near-search mutant applied instead at
     step.go's near CALL SITE leaves the whole tree green. Reported below, not hunted.
```

### What is not witnessed, and where this run left its contract

- **AC-11 and SC-11 are still not run.** They need a lawful install and a window: whether a lake
  crossing now *feels* right, and whether a 97.76 ms order tick reads as a hitch, are the
  owner's to judge.
- **T11, T12 and T15 own no mutant** and say so rather than filling one: the first two open no
  production file, the third measures. The two this revision owes are SC-14's.
- **The near search's rule is not covered through the tick.** Moving `step.go`'s near call site to
  the flat budget leaves the tree green: the reach assertion that kills the same mutant at the
  rule drives `canonicalRoute` directly, and nothing asserts which rule the *tick* hands it. A
  real gap, disclosed not closed.
- **`formatVersion` is 5 and this revision does not touch it.** AC-14 asks for the version byte
  where FR-3 left it and names no number, so there is nothing to reconcile: `0033-unit-kill/T2`
  raised it to 5 for the health pair.
- **T13 opened two files outside its list**, both forced: `generationBudget`, `canonicalRoute` and
  `searchRoute` take a `budgetRule` where they took an `int64` slack, so every caller moved or the
  tree would not compile. They are `counted_test.go` and `routefork_test.go`, at the call alone. `budget_test.go` (T13) also holds the channel fixture `channel_test.go` (T14) uses:
  AC-14's recorded run needs it and T13 lands first.
- **T15's fence "neither file may name a symbol whose signature moved" could not be met**, for
  that reason. The columns are still one measurement: `cntBefore` was taken at `d810357` with the
  identical map, start, goals and touch-list read, in a file never committed, before T13 landed.
  Only the budget argument's spelling differs — the thing under test.
