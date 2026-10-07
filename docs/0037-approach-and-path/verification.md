# Verification — 0037 approach and path

Windows 11, Go 1.26.1, 12th Gen i7-12700K. Submodule pin research `f35be34`, unmoved. Five task
commits, in order: the machinery unreached, the far search settling, the query, the seam, the
overlay.

## The gate

Run unscoped at the pushed head, over all **28** packages (`go list ./... | wc -l`):

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"
```

`-trimpath` throughout: Windows Defender quarantines a test binary without it.

**SC-1 — the FAIL set at push.** The set was **empty** before this story and is **empty** at the
head pushed: no story other than this one is named anywhere in either check. Rebased onto
`40304b1` — `0038-dim-map-border`, which touches three of the files this story does — with no
conflict and no test moved. Its dim is applied to a tile's own corner multipliers inside the terrain
shading rather than as an overlay pass, so it lies under everything and the draw order this story
adds to is unaffected.

## Criteria

**AC-1** `giveup_test.go` `TestAnUnservableOrderIsClearedOnTheFirstTick`, canonical arm: the three
orders no exact route serves walk to `(1,2)`, `(4,2)` and `(3,2)` and park, where each used to stand
still; sixteen ticks each, so a mover setting off again later would fail rather than pass.
`wall_test.go` `TestATargetOffTheMapIsWalkedTowardOrEndedByMode` adds the two off-map bearings.

**AC-2** `approach_test.go` `TestASettlingSearchTakesTheCheapestCellOfTheNearestRing`. The fixture
leaves two cells open in the goal's first ring and the search takes `(9,10)`. Both rivals are ruled
out **by assertion, not by argument**: the test reads the plane the wave left and fails if `(8,10)`
one ring out is not strictly cheaper than the cell taken, and if `(11,10)` — same ring, offered
earlier in the walk — is not strictly dearer. The tie-break half is
`TestASettlingSearchTakesTheEarlierOfTwoEqualCells`, below.

**AC-3** `stall_test.go` `TestEachTickHasExactlyOneOutcome`, the sealed-mover fixture: a mover
walled into its own cell reaches the fourth outcome — order over in the tick that found it, no
stall, no residue — because the only cell it can settle for is the one it stands on and the route
to it is empty.

**AC-4** `approach_test.go` `TestSettlingIsTheCanonicalWavesAloneAndTheOptimisedSearchStillRefuses`:
on one fixture the canonical mover walks and the optimised one neither moves nor keeps its order.
The near half is AC-8's fixture, which shows a near failure keeping cell, target and route while its
count rises.

**AC-5** `approach_test.go` `TestASettledOrderPointsAtTheCellItSettledFor`: after the first tick the
order no longer names the sealed cell, the stored route ends exactly at the cell it now names — so
the staleness tests leave it alone and no later tick sweeps — and the mover walks the whole way and
parks with nothing left over.

**AC-6** `approach_test.go` `TestTwoWorldsSettleAlike`: two worlds, sixteen ticks, `Hash()` compared
every tick.

**AC-7** `approach_test.go` `TestTheFlatBudgetIsAnOverrideWithAGate`: four arms at `D = 40`, where
the quarter-distance term hides the slacks, plus two at `D = 4`, where it does not — 9 against 7, so
the far arm's 5 and the near arm's 3 are separately witnessed and not interchangeable.

**AC-8** `approach_test.go` `TestTheTickHandsTheNearSearchTheScaledRule`, driven **through `Step`**.
The fixture asserts its own premises before it asserts anything else: the far route is four cells,
its sub-goal is the cell four away, the scaled rule allows seven generations there, and the four
turning cells of the ten-step detour are inside the near window — so what refuses the detour is the
budget and not the bound.

**AC-9** `approach_test.go` `TestRouteIsTheStoredRouteCopied`: nothing for an absent id, nothing
before an order, the stored route cell for cell after one; a write through the answer leaves `Hash()`
unchanged, and a second call does not hand back the mutated array.

**AC-10** `path_test.go` `TestOnlySelectedUnitsUnderOrdersArePreviewed`: seven cases, with each of
the four exclusions — not selected, no route, a corpse holding a route, an id the snapshot dropped —
asserted on its own rather than lumped into one "none".

**AC-11** *(amended 2026-08-01; see the revision at the end)*
`path_test.go` `TestEveryDrawableUnitIsPreviewedWhateverTheCount`.

**AC-12** `path_test.go` `TestThePreviewedLineRunsFromTheUnitAlongItsRoute` — three segments for a
three-cell route, the first from the unit's own cell, every endpoint compared against the camera's
own forward transform rather than against pixel literals — and `TestAnOffScreenLegIsStillDrawn`,
which asserts its own premise (both endpoints outside the window) before asserting the leg survives.

**P-1** No float, clock, file or generator on the settle path, and no map anywhere in it: the
determinism scan in `internal/archtest` covers `pkg/sim` and is green (SC-4). The picker's total
order is the ring walk plus a strict compare, witnessed by AC-2's two fixtures; the int32 bounds it
holds probes to are written out rather than imported, so the package's import set did not grow.

**P-2** `world_test.go` `TestReadersDoNotMutateTheWorld` sweeps `Route` as a reader with a live
fixture id and compares the whole world before and after; AC-9 covers the copy. `pkg/game`'s
snapshot build is unchanged in kind — `death_test.go`'s repeat-build test still passes, now over the
route as well.

## Success criteria

**SC-2** Every byte-form and digest pin in the tree re-runs green and none was edited:
`routeform_test.go`, `binary_test.go`, `hash_test.go`, `nostate_test.go`, `gridform_test.go`,
`malformed_test.go`, `preserved_test.go`, `replay_test.go`. No version byte moved and no field set
changed; `TestWorldExportedMethodSetIsPinned` was updated for `Route` and for nothing else.

**SC-3** T1 landed the whole settle machinery with both call sites passing the non-settling rule,
and the full suite passed at that commit with **no test's expectations changed** — only call-site
arity. So every digest pin held with the machinery present, and the behaviour change is attributable
to the one call site T2 moves and to nothing else.

**SC-4** `internal/archtest`'s source scan over `pkg/sim` is green at every commit.

## Mutation kills

Each applied to production code, the whole tree run, then reverted; `git status` clean after each
revert, so the restore is byte-identical rather than merely green.

```
SC-7  the budget gate dropped, so the flat override applies whatever the goal is
      KILLED at T1, 1 test: TestTheFlatBudgetIsAnOverrideWithAGate (2 of its 6 arms)

SC-5  the ring accept relaxed from strictly-less to less-or-equal
      KILLED, 1 test: TestASettlingSearchTakesTheEarlierOfTwoEqualCells

SC-6  the ring scan stopped at its first labelled cell instead of scanning whole
      KILLED, 4 tests over 2 packages: TestASettlingSearchTakesTheCheapestCellOfTheNearestRing,
      TestAnUnservableOrderIsClearedOnTheFirstTick, TestATargetOffTheMapIsWalkedTowardOrEndedByMode,
      and in pkg/mapload TestAChannelWithNoGapIsWalkedUpToAndNeverCrossed

SC-8  the near call site moved to the flat budget
      KILLED, 1 test: TestTheTickHandsTheNearSearchTheScaledRule, through Step()

SC-9  first mutant: the cap trimming to its limit instead of skipping whole
      KILLED, 1 test: TestThePreviewIsCappedOnTheDrawableCount (2 arms)
SC-9  second mutant: the preview's scope widened from the selection to every entity in the snapshot
      KILLED, 1 test: TestOnlySelectedUnitsUnderOrdersArePreviewed (all 7 arms)
```

**SC-5 survived its first fixture and the reason is worth recording.** A tie between two cells on
the ring's `x-r` side is decided identically by either accept, because the four sides overlap at the
corners and that side is walked again at the far end of the scan — so the cell offered first is also
the cell offered last. The fixture was rebuilt on two mid-side cells the walk visits once each, and
the mutant then died. A tie-break claimed as contract and witnessed only by fixtures containing no
tie is a claim with no evidence, which is what the first run measured.

**SC-7 was planned for T1 and is only killable there because its witness moved.** Wired in but
unreached, the gate is unreachable through any world — the pre-sweep refusal returns first — so the
mutant survived a full run. The budget rule's own criterion was moved into T1 with the rule it
tests, and `tasks.md` was corrected before T1 landed rather than after.

## SC-10 — the benchmark shape this story changes most

`-benchtime=1x`, ms/tick, against 0045's own ceilings and beside the figures 0045 recorded:

```
                                          0045      now    ceiling
  128x128/40 over 16                     1.752    1.814     10
  256x256/40 over 16                     6.215    6.884     25
  256x256/40 over 256                    0.574    0.595      4
  80 movers, sealed by units             3.277    3.452     12
  80 movers onto a held cell             3.389    3.618     12
  256x256/40 to a NEAR terrain-sealed cell
    mean                                 6.181    6.867     25
    worst single tick                    97.76   107.9      -
```

The shape whose meaning changed is the last one, and it is reported rather than replaced: its goal
is open and ringed by terrain, so the sweep always ran: what is new is that the sweep now ends in a
substitute and forty movers then walk instead of stopping. The mean rises 11% and the worst tick 10%
— the order tick, forty whole-map sweeps at once, now each followed by a ring scan bounded by
`(D>>2)+4`. Its test moved with it: it asserted every order cleared at once, and now asserts what
that claim was ever about — that the sweep lands on **one** tick, which is witnessed by no mover
still pointing at the sealed cell after it.

## Not witnessed, and why

- **FR-9's stroke itself.** `pathScreenSegments` is asserted whole; the loop that hands each segment
  to the drawer is three lines with no branch and needs a window. The same limit `0046`'s baseline
  named for its own draw loop.
- **`MOVE-ALT-022`'s tolerance and message** are out of scope in the contract, so nothing measures
  them.
- **The divergence's own cost** — that a mover settles for the first substitute rather than one
  re-derived as it advances — is stated in `provenance.md` and is not measured here. Measuring it
  needs the behaviour that was not built.

## A renamed test

`wall_test.go`'s `TestATargetOffTheMapEndsTheOrderOnTheFirstTickUnderBothModes` is now
`TestATargetOffTheMapIsWalkedTowardOrEndedByMode`. The old name asserts what is no longer true of the
canonical mode. It is the same test, and 0045's own record cites the old name in its mutation list.

---

# The revision — the cap the owner overruled

Commits, oldest first: `95c0026` (the contract correction, untrailered), `eef3231` (T6). Base
`8cc6860`; one trailered commit, one id, once. Pin research `5df4a39`, unmoved — the overlay is ours
and no claim is cited. Windows 11, Go 1.26.1, i7-12700K, 2026-08-01. Nothing reads an install.

## SC-11 — the measurement

The shape is the largest a shipped-size map produces: every selected mover on a full corner-to-corner
route over 256x256, viewed at 1280x720. The route length is **not chosen** — `pkg/sim`
`TestAFullMapGroupOrderHoldsARouteOfTheMapsDiagonal` drives a group through a tick and fixes it at
**251** cells, so a change in what the tick produces fails a test instead of rotting a premise.

```
go test -trimpath -count=1 -run TestAFullMapGroupOrderHoldsARouteOfTheMapsDiagonal ./pkg/sim/
go test -trimpath -run '^$' -bench BenchmarkThePathOverlayOnAFullMapGroupOrder \
    -benchtime=30x ./pkg/ui/

                             every leg               only legs meeting the view
                         legs   build  stroke         legs   build  stroke
  zoom 1    40 movers   10040   0.194   4.731          808   0.072   0.318
           320 movers   80320   2.396  34.25          3680   0.598   1.848
  ZoomMin   40 movers   10040   0.259   3.886         7128   0.148   2.984
           320 movers   80320   3.141  30.38         54240   1.192  21.99
```

**The count of units was never the cost.** A frame pays per **leg**, and a unit's legs are its
route's length — 1 to 251 here — so the cap drew 32 movers on full diagonals (8032 legs, 3.8 ms)
while refusing 200 movers on two-cell routes (400 legs, under 0.2 ms). At the default zoom **95.4%**
of a 320-mover order's legs are outside the window; not issuing them costs no pixel and takes the
worst case from 34.25 ms a frame to **1.85 ms**. That is why no limit on the count survives.

**R-5, measured rather than capped:** at `ZoomMin` the map is nearly all in the window, 67.5% of the
legs survive the reject, and the worst case is **21.99 ms** a frame.

The load-bearing figure is the **count**, exact and machine-independent. The milliseconds are the
**CPU half** of the stroke — ebitengine's draw-side calls run before the game starts and the GPU
flush does not — so they are a lower bound, and the decision rests on the ratio.

## Criteria

**AC-11** `path_test.go` `TestEveryDrawableUnitIsPreviewedWhateverTheCount`: 31, 32 and 33 drawable
movers, then 200 and 2000, each previewed whole and in the selection's own order, plus three movers
inside a selection of 500. The three straddling the removed cap are the point: a rule merely
**relaxed** to a bigger number answers all three correctly, one still skipping whole at 32 does not.

**AC-13** `path_test.go` `TestALegThatCannotReachTheViewIsNotIssued`: two routes leaving the view and
returning, six legs, the two lying wholly beyond an edge dropped. It asserts which of its cells are
outside the view and which two are inside first, so a fixture that drifted into the window would fail
rather than pass vacuously.

## Mutation kills

Each applied to production code, the whole tree run, then reverted; `git status` clean after each.

```
SC-9  first mutant (amended): a count answers short - drawable units trimmed to 32
      KILLED, 1 test: TestEveryDrawableUnitIsPreviewedWhateverTheCount (3 of its 6 arms)
SC-9  second mutant: the preview's scope widened from the selection to every entity
      KILLED, 2 tests: TestOnlySelectedUnitsUnderOrdersArePreviewed (all 7 arms),
      TestThePreviewedLineRunsFromTheUnitAlongItsRoute
SC-12 the reject rewritten as an ENDPOINT test - a leg kept if either end is inside
      KILLED, 2 tests: TestALegThatCannotReachTheViewIsNotIssued, TestAnOffScreenLegIsStillDrawn
```

**SC-12 survived AC-13's first fixture, and that is worth recording.** It died against AC-12's
off-screen leg but not against AC-13, whose two dropped legs are dropped by either rule — so the new
criterion was witnessed by a case that could not tell the two rules apart, and the test's comment
claimed a discrimination it did not make. The fixture was rebuilt with its first leg spanning the
view from **outside on both sides**, which the box rule keeps and an endpoint rule takes away, and
AC-13 now kills the mutant on its own.

## The gate

Re-run unscoped at the pushed head, over all **30** packages, one `&&` chain with no pipe:

```
$ go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
  sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
  sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"
  27 ok, 3 without tests, 0 FAIL
  check-no-game-assets: clean (tree scan)
  check-doc-budget: 0037 under a DECLARED OVERRUN (plan 14336, verification prose 13312)
  gofmt: no output over 292 files
```

**SC-1 at this head.** The FAIL set is **empty**, as before the revision. It was red at `95c0026` on
exactly AC-13, SC-11 and SC-12 — ids that existed before their evidence did — and nothing was pushed
until the revision was whole.

## Not witnessed, and why

- **The GPU half of the stroke**, per the note above. What ships is the CPU cost and the exact count.
- **That the picture is unchanged by FR-9a.** Argued from the geometry and witnessed only at the
  criterion level, over the legs a rule might wrongly drop. Comparing rendered frames needs a window
  and a pixel read-back — the limit `0046`'s baseline named for its own draw loop.
- **A real session's selection size**, and the 320-mover row generally: benchmark shapes, not
  observations.
