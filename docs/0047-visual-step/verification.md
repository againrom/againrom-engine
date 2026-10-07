# Verification — the step a unit took, drawn between the two cells it joins

Task commits, oldest first: `7dc965d` (T1), `db2edd6` (T2), `6344573` (T3), `dfa414e` (T4),
`1365bb1` (T5). Base `a1e83a7`. Five trailered commits, each id once, and no other commit in the
range carries a trailer at all. Two untrailered commits of this story's own sit among them —
`41e76b3`, the tasks.md revision recorded under *Contract repaired* below, and this file — beside
three `docs(0045)` commits another lane pushed mid-run and this branch was rebased onto.
Submodule pin frozen at research `8c92427` throughout.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11, measured 2026-07-31. Every criterion
below but AC-10 is a unit test over hand-built worlds, viewers and remainders passed in — **no
test reads a game install, opens a window or reads a clock**, and no binary ships.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output; 266 files)
$ go list ./... | wc -l
28
$ go test -trimpath -count=1 ./...           (0 FAIL, 0 SKIP; 937 tests, 2831 counting
                                              subtests; 25 packages ok, 3 without tests)
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh             (0047 rows)
  analysis.md 6279 / 7168      spec.md 13069 / 13312      plan.md 13084 / 13312
  tasks.md T1..T5 910 878 1025 1157 656, all under 1400; legend+traceability 1135 / 1200
  plan <= 1.2 x spec ok (13084 <= 15682); tasks <= 1.2 x plan ok (5761 <= 15700)
$ sh scripts/check-sdd-audit.sh              at 3210e50, before the rebase and this file
FAIL 0047-visual-step: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED                      ONE enforced FAIL, naming this story
$ sh scripts/check-sdd-audit.sh --story 0047-visual-step      with this file
check-sdd-audit: 5 trailered commit(s) for 0047-visual-step in ac6bd87..HEAD checked
check-sdd-audit: ok (1 note(s)/warning(s), none enforced)     the warning is builds/, untracked
$ git log --format='%(trailers:key=SDD-Task,valueonly)' a1e83a7..HEAD | sed '/^$/d' | sort | uniq -c
      1 0047-visual-step/T1 ... 1 0047-visual-step/T5      (5 ids, each exactly once)
$ git log --format='%B' a1e83a7..HEAD | grep -ci co-authored-by
0
$ git diff --stat a1e83a7..HEAD -- pkg cmd internal | tail -1
 13 files changed, 2035 insertions(+), 63 deletions(-)   (pkg/sim: none of them)
```

That FAIL is what a story with every task landed and no evidence file looks like. This file clears
it and carries no trailer.

**The full sweep is red for another story, not this one.** The three `docs(0045)` commits this
branch was rebased onto revise a landed story's contract, and the audit reports
`0045-two-tier-search: nothing in verification.md witnesses` for the ids those commits introduced
— none of which existed before them. That lane owns it; 0047 is green scoped, and every other gate
above is green over the whole tree.

## Witnesses

Every id and the named thing that answers for it.

```
FR-1 AC-1 C-4  SC-1   pkg/game/facing_test.go TestAnAdjacentOrderTurnsTheUnitAndPlaysItsWalk
      — three directions, each a fresh 8x8 hand-built world: the entity reads as having moved
      in the direction it moved and draws that direction's WALKING frame, and the world is
      asserted to hold NO target at that boundary, which is the quantity the old rule read.
FR-1 FR-2 AC-2   TestEveryStepOfAWalkIsDrawnWalkingAndTheArrivalKeepsItsDirection — four cells
      east then four south, every step including the ARRIVAL step read as a move in its own
      octant, and the advance after each leg read as idle keeping that leg's direction.
FR-2 AC-3 DD-3  SC-2   TestAnEntityThatTookNoStepIsIdleInTheDirectionItLastWent — the
      never-moved entity idle in octant 0 for three ticks with a walker beside it moving
      throughout, and, in a one-row corridor with no way round, a walker held up behind a
      standing neighbour: idle on that tick, still facing east, and still HOLDING its target
      (a stall, not a give-up). Every octant expected below is derived from the cells the world
      was watched to occupy, through this file's own transcription of the eight facings.
DD-1 DD-2   pkg/game/world_test.go TestTheTickZeroPushCarriesNoStepAndAsecondBuildRepeatsIt —
      no entity carries a step at tick 0, and after three ticks a snapshot built, pushed and
      built again answers identically while at least one entity is carrying a step. Perturbed
      (memory written from the build instead of the tick) this reports zero movers and fails.
FR-6 DD-4   pkg/render/terrain/water_test.go TestTickerReportsTheRemainderItCarries — the
      remainder across five advances against elapsed-minus-consumed computed here, and across
      a re-rate that SHORTENS the period below the remainder held, where it is reported
      unbounded and the next advance consumes it as the whole ticks it is worth.
FR-6 AC-7 DD-5  SC-6   pkg/game/world_test.go TestThePacedAdvancePushesItsOwnClocksPhaseAnd-
      AStopHoldsIt — the viewer's own water clock re-rated to 256/s first, so a phase read off
      the wrong instance shows as the wrong period; nine sub-tick frames each pushed the
      pacing clock's own remainder (5 distinct); twelve stopped frames leave it unmoved; the
      resume moves it again. A viewer nobody paced holds (0, 0).
FR-3 AC-4 P-3 P-5 DD-6 DD-9  SC-3   pkg/ui/shift_test.go TestOneStepIsDrawnBetweenTheTwoCells-
      ItJoins — a sprite entity and a square entity, both selected and both carrying a bar, at
      six phases over the contract's own period 62000: offsets -32, -24, -16, 0, 0 (clamped),
      -32 (clamped). Each family is read through the reader the frame itself uses and compared
      PER ENTITY against one vector. P-5's endpoints are compared against a second viewer
      holding the same units AT REST on the cell left and on the cell entered.
FR-4 AC-5 DD-7  SC-4   TestTheReliefRunsAcrossTheTickWithTheStep — the same six phases over a
      relief rising four units a column, the vertical term running from the left cell's lift to
      the entered cell's, with the two cells asserted to differ in height; and at a remainder of
      zero the drawn height compared against the resting picture on the cell left.
FR-5 AC-6 P-4  SC-5   TestNothingThatDidNotMoveIsEverDisplaced — standing, blocked, downed and
      dead, flat and displaced, at every phase: no displacement anywhere, and the four squares
      still on their own cells.
FR-6 FR-8 DD-8   TestAViewerToldNothingDrawsEveryEntityOnItsOwnCell — never told, a period of
      zero with a remainder held, a negative period, and a period shrunk below the remainder:
      all four compared against a viewer holding the same units with no step at all.
FR-8 AC-9 P-5  SC-8   pkg/ui/overlay_test.go TestEveryEntityGlyphStandsOnItsOwnCellsMarker-
      Geometry — the T1 characterization, unchanged and re-run after T4: the sprite's ground
      point is the marker path's own anchor for its cell, and the square, the mark and the two
      bar arms are the render tier's own glyphs for their cells, flat and displaced, at two
      camera positions. It discriminates: a lift perturbed by one pixel fails it.
FR-7 AC-8 P-1 P-2   pkg/game/drawn_invariance_test.go TestOneCommandStreamReachesOneDigest-
      WhateverIsDrawnBetweenItsTicks — one hand-written stream, a leg drawing nothing between
      ticks and a leg making five front-end frames a tick (65 frames, 5 distinct phases, 50 of
      them with somebody mid-stride), both compared to a HEADLESS run assembled from that same
      table: canonical byte form and digest equal at every one of 13 tick indices. Beside the
      digest it pins what must survive — snapshot order, art resolution, life byte and health
      pair at every tick, and the at-rest entity's frame cadence.
P-2 SC-7   TestAStoppedWorldIsDrawnFromOneSnapshotAtOnePhaseInEveryFrame — stopped mid-stride
      part-way through a tick, twenty frames: the phase, the freshly rebuilt snapshot, the tick,
      the byte form and the digest all unmoved; the resume then moves all of them.
SC-7   pkg/sim/nostate_test.go TestTheCanonicalWorldsFieldSetsArePinned and TestTheByteForm-
      AndItsDigestAreTheOnesAlreadyPinned PASS unedited. internal/archtest TestLiveTreeClean,
      TestUITierAllowanceIsPinned and TestSimSourcesAreDeterministic PASS — the import-graph
      wall and the pkg/sim source scan, both unmoved. pkg/sim has no diff in this story.
```

## Mutants — SC-10

Each applied to `pkg/ui/overlay.go` alone, the whole tree run, then restored by copy and the
restore confirmed by MD5 (`70fcbb02eec0df1540ad71ec0b0a29a4` before and after each).

```
1  the tick part taken as elapsed rather than as what remains
   -  left := period - elapsed          +  left := elapsed
   FAIL pkg/ui only; 13 subtests across TestOneStepIsDrawnBetweenTheTwoCellsItJoins,
   TestTheReliefRunsAcrossTheTickWithTheStep and TestAViewerToldNothingDrawsEveryEntityOn-
   ItsOwnCell. 24 packages still ok.

2  the remainder clamp removed  (both bounds of clamp(elapsed, 0, period) deleted)
   FAIL pkg/ui only; 5 subtests — the two "clamped" phases of each geometry test and
   "a period shrunk below the remainder". 24 packages still ok.

3  the shared vector broken (not an SC-10 mutant; T4's own): the mark's walk given a phase of
   its own, half the frame's.
   FAIL pkg/ui only; 7 subtests across the two geometry tests. 24 packages still ok.
```

Mutant 3 was run twice. The **first run killed one subtest only**, and the cause was a defect in
the test rather than a weak mutant: the mark was being read through the shared per-entity walk
instead of through `selectionScreenRects`, so a build whose glyph paths had come apart above that
walk would have passed. The reader was moved onto the production path for all three families and
the mutant re-run; the seven kills above are from that second run. Recorded because the first
number was the honest one at the time.

## Not run

- **AC-10 — the owner's, and not run here.** It needs a window and a lawful install: a unit
  ordered one cell, then across the map, with the world paused mid-stride and resumed and the
  rate raised and lowered. SC-9 is that criterion and is likewise unrun. Nothing in this file
  stands in for it, and the disclosed limitations the contract owns — the entity layer trailing
  the world by up to one tick, a unit alternating between walking and idle frames while
  repeatedly held up, whole-render-pixel displacement, and a catch-up burst still jumping — are
  visible only there.
- **AC-7's stopped arm is witnessed at its inputs, not at its geometry.** `pkg/game` cannot
  reach the window tier's placement functions, so what is measured there is that the snapshot
  and the phase — the two values a frame is drawn from — do not move across twenty stopped
  frames. That the geometry is a function of exactly those two is measured in `pkg/ui`. The
  composition is sound; it is not one test.
- **AC-8's "both wall checks unmoved" is a separate run**, `internal/archtest`, not an assertion
  inside the invariance test. Listed under SC-7 above.
- No corpus run and no benchmark: this story adds no format claim and no measured cost.

## Contract repaired

`tasks.md` asked T2 for SC-10's elapsed-for-remaining mutant and T3 for its clamp-removal twin,
but both quantities belong to the displacement arithmetic, which **T4 writes**. At T2 and T3
there is no line to mutate and no test for one to kill, so a kill claimed there would have been a
kill claimed over nothing. Both moved to T4 in `41e76b3`, before T2 landed; T3 kept the part it
can witness — that the two states SC-6's arms are stated over exist and read back — and handed the
geometry to T4. Its files line gained `pkg/game/world_test.go`, the only place its observable is
reachable from.

Three deviations from the files lines, each disclosed rather than silent. T3 added a `Phase()`
reporter beside `SetPhase` on the viewer, mirroring `EntityMarkers()`: without it "the pushed
phase seen to be unchanged" cannot be seen at all. T4's DD-9 makes the entity split hand back
entities instead of bare cells, which reaches `entity_overlay_test.go`, `mirror_draw_test.go` and
`deadselect_test.go`; the cell-returning selection reader was deleted rather than left with no
production caller. T1's ui-side characterization went into `pkg/ui/overlay_test.go`, which
`tasks.md` names and `plan.md`'s file list does not.

## Conclusion

Every automated criterion holds. The defect the owner reported is fixed wider than reported — a
one-cell order and the arrival step of every walk now turn the unit and play its walk — and the
jerk between cells is smoothed render-side only: one command stream reaches the same byte form and
the same digest at every tick index whether it was drawn at many remainders or at none, `pkg/sim`
has no diff, and both determinism walls are unmoved. What remains is the owner's own run.
