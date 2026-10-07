# Verification — deterministic unit cell occupancy

Task commits, oldest first: `3868556` (T1), `01a869b` (T2), `a9bbd5e` (T3), `d120e7a` (T4),
`cf95e42` (T5), `ae54196` (T6). Base `e87cf34`. Three untrailered commits sit in the span, every
one docs-only and belonging to another story — all three authoring `docs/0027-vfs-dispatch/` —
disclosed, not a violation. Submodule pin frozen at research `e7602bb` throughout. The span
touches nine files and no others: `pkg/sim/step.go`, the five `pkg/sim` test files the tasks
added, and 0027's three.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. The gate ran in a detached worktree at
the evidence commit, so it measured that tree rather than whatever else was loose beside it. A
fresh worktree leaves the `research/` submodule uninitialised; nothing in this story's chain reads
it, so the gate was run there anyway, as it was for all six tasks. Every figure comes from
synthetic worlds built in test code; no game install is read. Every criterion here is unit-level,
so nothing in the story waits on an owner run; the review binary under
`builds/0026-unit-collision/` is untracked and no part of it is committed.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 620 tests, 2087 counting subtests,
                                              24 packages ok, 4 without tests)
      pkg/sim    69 tests, 111 counting subtests   (14 test functions added by this story)
      pkg/game   30 tests, 148 counting subtests   (not a file of it touched — R-2)
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0026-unit-collision
  analysis.md 6702 / 7168        provenance.md 7006 / 7168
  spec.md 13001 / 13312          plan.md 12547 / 13312
  tasks.md T1..T6 all under 1400; legend+traceability 820 / 1200
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok                        (exit 0)
  verification.md (prose) 9126 / 9216 — fenced blocks free
$ sh scripts/check-sdd-audit.sh                      at ae54196, before this commit
check-sdd-audit: 82 trailered commit(s) in ac6bd87..HEAD checked
FAIL 0026-unit-collision: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED
$ sh scripts/check-sdd-audit.sh                      with this commit
check-sdd-audit: 82 trailered commit(s) in ac6bd87..HEAD checked
check-sdd-audit: ok (36 note(s)/warning(s), none enforced — every one of them a story
                     before WITNESS_FROM, plus 0027 in flight)
$ git log --format='%(trailers:key=SDD-Task,valueonly)' e87cf34..HEAD | sed '/^$/d' | sort | uniq -c
      1 0026-unit-collision/T1  ...  1 0026-unit-collision/T6   (6 ids, each exactly once)
$ git log --format='%B' e87cf34..HEAD | grep -ci co-authored-by
0
```

That one FAIL is what a story with every task landed and no evidence file is supposed to look
like. This file clears it and carries no trailer. The same commit corrects one clause of a doc
comment in `pkg/mapload/schedule.go`, declared in its body: the orders are still spaced by exactly
the ticks a full-length leg takes, but the consequence that comment drew from the spacing — an
entity reaching each corner as the order for the next one arrives — is now conditional on nothing
blocking it. Comment only, so `vet`, `gofmt` and the suite read the tree the same either way.

## Witnesses

Every id and the named thing that answers for it.

```
AC-2  SC-1                      pkg/sim/occupancy_test.go
      TestStepGivesAContestedCellToTheLowerID — three geometries (along x, along y, on
      the diagonal), the named lower id taking the cell and the higher one's position,
      HasTarget and both target coordinates unchanged, contenders non-adjacent in id
      with an uninvolved unit between them, and the fixture's own arithmetic asserting
      that the two desired cells really are one empty cell.

AC-4  SC-3                      TestStepAdvancesALowIDLedConvoyAsOneBody — all three
      advance inside one tick, five ticks running.  TestStepStretchesThenFlowsWhenThe-
      LeaderCarriesTheHighestID — the contract's stretch-then-flow example as a
      hand-written table, one row per tick, nothing recomputed from the step's own rule,
      and the run asserted to have held somebody.

AC-3  AC-5  P-2  SC-4           pkg/sim/blocked_test.go
      TestStepLeavesAnEntityBlockedByATargetlessUnitUntouched — the blocker target-less
      and given no command, once ahead of the mover in id and once behind; position,
      HasTarget, TargetX and TargetY compared one field at a time, never through a struct
      or a byte form; the order naming a cell beyond the desired one.  TestStepHoldsAPair-
      EachOrderedOntoTheOthersCell — both units asserted, held four ticks.

AC-8  P-4  SC-5                 TestStepRefusesToSlideWhenTheDesiredDiagonalIsHeld —
      both orthogonal neighbours toward the target checked empty before the tick and
      after every tick, which is "neither coordinate moved" said from the other side.
      TestStepStepsDiagonallyWhenOnlyAnOrthogonalNeighbourIsHeld — the mirror, once with
      the x neighbour held and once with the y.

AC-1  AC-9  P-1  SC-2           pkg/sim/contention_test.go
      TestStepKeepsEveryUnitOnACellOfItsOwnThroughAnNWayConvergence — three equidistant
      contenders on one cell, ids pairwise non-adjacent, run past the arrival tick,
      distinctness after every tick.  TestStepKeepsEveryUnitOnACellOfItsOwnThroughA-
      SeededSweep — twelve units in sixty-four cells, twenty-four ticks, targets redrawn
      from the test's own seeded generator, the world's generator asserted unmoved, and a
      second run of the seed reproducing every trajectory and every per-tick digest.

AC-11 P-5  SC-7                 pkg/sim/malformed_test.go
      TestStepAdvancesACoLocatedPairWithoutRaisingAnyCellsCount — three cases: the
      self-order on the lower of the two co-located ids, on the higher with the third
      unit approaching diagonally, and on both at once. Occupant counts compared cell by
      cell over every cell either state touches, before against after, on every tick.
      TestStepPartsACoLocatedPairOnlyWhereMovementPartsIt — the count may fall.

AC-6  AC-10 SC-6                pkg/sim/step_test.go, unedited by this story
      TestStepWalksOneCellPerTickAndClearsTheTargetOnArrival (the walk table) and
      TestStepClearsATargetNamingTheEntitysOwnCell (the self-order).

AC-7  AC-12 P-3  SC-8           pkg/sim/nostate_test.go
      TestTheCanonicalWorldsFieldSetsArePinned — a literal table of name, type and order
      for World, Entity, Bounds and rng, compared for exact equality rather than searched
      for an absence.  TestTheByteFormAndItsDigestAreTheOnesAlreadyPinned — reads
      binary_test.go's pinBytes and hash_test.go's pinDigest, both left unedited, so
      AC-12's pin is not touched by the file that judges it.  TestADecodedWorldStepsOnTo-
      TheSameDigestsAsTheFirst — the crossing: at every tick of a contended run the world
      is marshalled, decoded into a second world and stepped through the rest of the
      schedule beside the first, digest for digest and form for form.

SC-9                            measured, not asserted — see the register below.
```

## The mutation campaigns

Every witness was made to fail before it was trusted, against one shared 34-mutant harness so the
numbering lines up across tasks. Counts as the task agents reported them:

```
task  ran                          killed  left standing
T1    29 + a control               26      3  the movers-only predicate, M23, M24
T2    34 (the control among them)  23      4  the control, M23, M24, M32
T3    34                           20      3  the control, M23, M24
T4    34                           22      3  the control, M23, M24     sole killer of none
T5    34, and M34 which it wrote   25      3  the control, M23, M24     sole killer of none
T6    34, and 9 of its own         11 + 8  3 + M39, its own          sole killer of none

T6's two figures are 11 of the shared 34 and 8 of its own 9. "Sole killer of none" is of
the shared 34: everything those three tasks killed, an earlier case already killed.
```

The mutants the record turns on, and the five of them re-run in this seat:

```
M23   the desired cell tested with the resolving unit excluded by index, the arrival check
      left OUTSIDE the guarded arm.  RE-RUN: whole tree green, 24 packages ok.
M24   the blocked arm falling through to the arrival check — unkillable, and provably: a
      step on either axis means the unit is not standing on its target.
M32   the desired cell AND both orthogonal neighbours tested. Survived T1 and T2; killed by
      T3's mirror and, ever, by nothing else.
M34   what DD-4 actually rejects: self excluded by index AND the refusal skipping the rest of
      that unit's resolution.  RE-RUN: exactly one test in the tree is red, in all three of
      its cases — TestStepAdvancesACoLocatedPairWithoutRaisingAnyCellsCount.
M35   an occupancy index field on World rebuilt from the entities every tick — field table.
M36b  a persistent index field, built at construction, whose restore the decode forgets —
      C-1's alternative (A). Field table and crossing both; the crossing parts the two worlds
      at the first tick of the first cut.
M37   a package-level index that HOLDS occupancy between ticks, built at tick 0 and never
      rebuilt.  RE-RUN: the field-set pin passes it and the crossing kills it, as reported —
      but it is also red in step_test.go (3), hash_test.go (1), binary_test.go (1) and
      pkg/game (1), so of nostate_test.go's two pins only the crossing sees it, and "the
      crossing is its sole killer" is not a claim this seat can make.
M38   a blocked flag on Entity written on the refusal path — field table only.
M39   a package-level buffer REBUILT from the entity slice at the top of every Step and kept
      current as each unit moves: behaviourally identical, holds nothing across a call.
      RE-RUN: whole tree green, internal/archtest included.
M40   a field on Bounds.   M41  a field on rng.  Field table only, and why it reaches past
      World and Entity — World holds both by value.
M42   a record widened by a byte.   M43  a bumped format version.  Red here and in the older
      byte-form pins together — FR-9's byte half still holding, not a claim of this file's.
SC-9  the move loop reversed to descending index.  RE-RUN: build and vet clean, both halves
      of the determinism wall green, and exactly three tests red, all of them in pkg/sim —
      TestStepGivesAContestedCellToTheLowerID in all three geometries,
      TestStepAdvancesALowIDLedConvoyAsOneBody,
      TestStepStretchesThenFlowsWhenTheLeaderCarriesTheHighestID.
```

**A mutant is itself a claim, and one of these was mis-coded for three tasks.** M23 was meant to be
DD-4's rejected variant, but as coded it left the arrival check outside the guarded arm — so
refusing a zero-distance order refuses a **no-op**: the unit does not move, the arrival check runs
anyway, and the order clears exactly as it does under the shipped rule. It is an equivalent
mutant, unkillable for the same reason M24 is, and T1, T2 and T3 each reported it as separated
from the shipped rule by "only a malformed co-located fixture" — a prediction carried through
three tasks against a mutant no fixture can kill. T5 built the variant DD-4 *actually* rejects,
numbered it M34, and showed `malformed_test.go` is its sole killer tree-wide; both re-runs above
confirm it. **DD-4's design claim is right and only the mutant's coding was wrong, so no plan edit
is owed** — the co-located self-order really is the one input that separates the two forms, and
the malformed fixture really is the only thing in the tree that constructs it. T6 then met the
same hazard in its own work and applied the lesson: its M36 as first coded panicked instead of
diverging, which naively reported would read "killed elsewhere" and mean nothing, so only the
rewrite (M36b) is reported at all.

**"It did not move" does not assert "it was blocked".** DD-7 asks each contention criterion to
witness a block, and T2 and T3 measured what the natural wording is worth: with the movement guard
weakened to `dx != 0 && dy != 0` nothing orthogonal moves at all, every field comparison in
`blocked_test.go` stays green, and only the counterfactual controls fail. So each blocked case
carries one — same entity, same order, the obstacle parked in a corner the case has checked empty,
asserting that the entity **does** arrive. Over a run that control does not survive, because which
unit obstructs which changes tick by tick and there is no one cell to park an obstacle in: T4 walks
each unit again through the same orders in a world holding it alone and requires some unit at some
tick to be further from its own target than its own solo walk was. It is one-directional, so a
broken control cannot manufacture a false witness — under the same weakened guard the solo walks
stand still too and the comparison goes quiet.

**Neither run contends by luck**, and the figures are read off real runs of `Step` in this seat: a
unit that held a target, whose desired step was non-zero, and whose position did not change was
refused, which is exact because `Step` is the only mover.

```
sweep         12 units, 24 ticks, 288 unit-ticks: 145 refused resolutions, 143 moves,
              163 lagging unit-ticks against the solo walks.
              T4 reported 141: that is the same count with the 4 refusals that fall on the
              tick a unit's order arrived excluded. Both are right about different sets.
convergence   5 units, 8 ticks: 12 refusals.
crossing      5 units, 9 ticks: 18 refusals, and 8 of the 9 ticks refuse somebody.
R-2 re-measured on the SHIPPED rule over pkg/game's 12x9 four-unit fixture and its own
              schedule, 575 ticks: 126 / 108 / 102 / 120 blocked ticks per entity and ZERO
              co-located pair-ticks. The planning prototype's 102-126 and its "the 78
              co-located pair-ticks become none" both reproduce exactly.
```

**`spec.md` C-1 mis-prices its rejected alternative, and that is the hole T6 closes.** C-1 says a
persistent in-world index is paid for by changing the serialized and hashed shape, invalidating
every pinned byte form and digest. M36b is the cheaper and worse variant: a persistent index field
that is simply **not serialized** costs no byte and no digest, passes every byte-form and digest
pin in the tree, and its real price is a decode that silently loses occupancy. The conclusion —
derive it per tick — stands; the stated reason was incomplete. It is recorded here and **`spec.md`
is deliberately not edited**: the constraint itself is correct and unambiguous, and spending
contract headroom to improve a rationale column is the wrong trade.

**SC-8's crossing has a boundary, and DD-2 should not be read past it.** The crossing kills a
package-level index that *holds* occupancy between ticks (M37) and no field table can see one. But
an index **rebuilt** from the entity slice at the top of every `Step` (M39) is behaviourally
identical, holds nothing across a call, and survives the whole tree — so it is DD-2's *engineering*
argument that refuses it, not SC-8, and a reader must not infer that SC-8 covers any package-level
scratch. Related, and it belongs in the open register below: nothing in the tree refuses a new
package-level variable in `pkg/sim`, or a new type beside `World`, `Entity`, `Bounds` and `rng` —
`internal/archtest` passes M39 too.

**What each task was worth, counted honestly.** T4, T5 and T6 were each sole killer of **none** of
the shared 34: everything they killed, an earlier case already killed. An invariant sampled over a
run is worth less, mutant for mutant, than a case that names a winner — which is not an argument
against writing them, since they are this story's universals and discharge criteria nothing else
reaches, but it is an argument against reading a kill count as coverage. Two measurements are worth
keeping. All 11 of T6's kills among the shared 34 fire through the contention precondition and not
through the crossing, because a mutant that is a pure function of world state computes the same
wrong answer in *both* worlds and a two-world comparison structurally cannot see it — which is
exactly why the crossing needed a new class of mutant built for it. And M32, which blocks strictly
more than the contract allows, is invisible to every case that expects a block: only a case that
expects a *move* past an occupied neighbour has ever failed on it.

## Disclosed notes and the open register

- **SC-9 discharges an ordering debt standing since story 0019.** Before T1 the ascending order was
  required and unobservable, so nothing pre-existing anywhere in the tree notices the reversal —
  the debt had no witness to lean on, which is the point rather than a gap in the older suites.
- **A new package-level variable, or a new type beside the four pinned structs, is refused by
  nothing.** The natural home is `internal/archtest`'s source scan, which already reads parsed
  syntax for imports, float type names and float literals. T6 correctly did not add it: it is a
  candidate criterion for a later story, not an unplanned boundary taken here.
- **T1's campaign found two gaps in its own fixtures** — a single along-x contest is blind both to a
  rule that tests occupancy only when x moves and to one that tests an orthogonal neighbour of the
  desired cell — and widened AC-2's witness into the three-geometry table above rather than
  reporting a pass on one case.
- **T6 widened the field-set pin past `World` and `Entity` to `Bounds` and `rng`** and measured
  rather than argued the extension: M40 and M41 die there and nowhere else.

## Contract findings

Nothing in `spec.md` (13001 B) or `plan.md` (12547 B) was found false at the implemented tree, and
neither is edited by this stage. C-1's rejected-alternative column is incomplete rather than wrong,
recorded above with the reason it is not being edited; DD-4's claim survived a mutant that was
coded wrongly against it. R-2's prediction held in both directions: `pkg/game`'s driven suite is
green with no fixture there softened and not a file of that package touched in the span, and its
measured cost reproduces on the shipped rule. Every reverted mutant left its file byte-identical —
`pkg/sim/step.go` md5 `e666f0f5f72f3ee00f69a1a321f81f64` before and after, the same digest all six
tasks reported.
