# Verification — the object layer's cycle, and the gate no cell opens

Windows 11, Go 1.26.1, no game install present, research pin `8c92427` unmoved.
Every fixture is built in test code; nothing below reads an asset. `-trimpath`
throughout (Defender quarantines a test binary without it).

## The gate

Run unpiped as one `&&` chain, the last time over the finished tree.

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"

25 packages ok, 0 FAIL, 3 with no test files
863 test functions, 2638 including subtests, 0 FAIL, 0 SKIP
check-no-game-assets: clean (tree scan)
check-sdd-audit: 145 trailered commit(s) in ac6bd87..HEAD checked; ok
gofmt: nothing
docs/0031-object-animation/analysis.md      6010 /  7168  ok (83%)
docs/0031-object-animation/provenance.md    7097 /  7168  ok (99%)
docs/0031-object-animation/spec.md         13225 / 13312  ok (99%)
docs/0031-object-animation/plan.md         13291 / 13312  ok (99%)
docs/0031-object-animation/tasks.md T1..T6  670/948/759/1065/987/740 of 1400 each
docs/0031-object-animation/verification.md  9180 /  9216  ok (99%)
0031: plan <= 1.2 x spec ok · tasks <= 1.2 x plan ok

six task commits, 22 files, +2327 -89
29 new test functions in five new files; five existing test files carry
call-site changes only
```

## Criteria

| Criterion | What ran | Result |
|---|---|---|
| **AC-1** | `TestObjectTimelineExpansion` — the four pairs, each a subtest, each expected timeline hand-written | all four: 28 steps, `5,5,5,7,7`, `9,9`, empty. `TestObjectTimelineIgnoresPhases` adds both directions of "Phases is not a cycle", `TestObjectTimelineNegativeDuration` the rest of *non-positive* |
| **AC-2** | `TestObjectStepMatchesTheContract` — four cells × counters 0…31, the expectation written as FR-3's expression in the test | 128 comparisons, 0 wrong, each re-evaluated. `TestObjectStepStaggerIsNotTransposed` adds the (1,0)/(0,1) separation over 64 counters |
| **AC-3** | `TestObjectCycleOpenGate` — 8 subtests × 2 gate modes | both-bits interior open; one bit, neither, last column, last row, far corner, period 0 and a negative period closed — the three edges with **every** word set. The open case puts one bit on each of two neighbours, so a gate reading fewer words closes |
| **AC-4** | `TestSelectObjectFrameArms` — Index 3, a 28-step timeline, a 12-frame sheet, 7 counters × 3 arms | `3+timeline[step]`, `3`, `0`. `TestSelectObjectFrameOutOfRangeDrawsFrameZero` adds value 40, an out-of-range Index, a negative Index, a sheet of no frames |
| **AC-5** | `TestAnimateStaticsReanchors` — a cycle over a 2×2 and a 6×10 frame, at two counters | one ground point at both, compared as a point; top-lefts differ by exactly `(2,4)`, the halved size difference in **both** axes; each `Rect()` its own frame's |
| **AC-6** | `TestLoadStaticsCarriesTheWholeSheet` — 4 subtests over 0017's archive | the two oak classes share every frame pointer; each drawing class carries its whole sheet; the whole-sheet exclusions leave `Frames` nil, the range one the sheet and a nil `Frame`; the timeline arrives expanded |
| **AC-7** | `TestRenderStaticsAtAChosenTick` — the raster PNG against `StaticFrame.RGBA`/`RGBALit`, the window's own images, over the sprite's rectangle | byte-identical at tick 0 (frame 2, 26×10) and tick 2 (frame 1, 12×18), unshaded and lit |
| **AC-8** | `TestObjectsAndWaterReadOneCounter`, `TestObjectsCycleAsTheCounterRises`, `TestAnimationOffHoldsTheCounterAndDrawsFrameZero` | one counter at rate 8 and at 16, read through `ResolveAnimated` and through the object pass and compared as one value; three counters give three distinct frames; with the switch off it holds and every placement draws its sheet's frame 0 at its unmoved ground point, and switching on resumes at held+1 |
| **AC-9** | `TestAnimateStaticsIsTotal` (4 subtests), `TestObjectStepIsTotal`, `TestSelectObjectFrameHasNoFourthArm`, `TestObjectCycleOpenWithoutTileWords` | a nil bundle builds and draws nothing under both gates and both switch states, and so does a 4×4 map every cell of which places, at counter `2^32−1`; no panic, every rect its frame's; period 0 never re-selected; a subset not indexing its list is ignored |
| **AC-10** | **not run** — see below |

**P-1** is `TestSelectObjectFrameIsAFunction` (2 arms × 2 gates × 30 counters,
each evaluated twice) plus AC-2's re-evaluation. **P-2** rests on structure:
no file this story touches imports `pkg/sim`, `internal/archtest`'s import graph
and source scan re-run unmoved, and 0041's digest and seam-state tests pass
untouched. **P-3** is
AC-4's three arms plus AC-9's fall-throughs: a period of 0, a closed gate and an
out-of-range value each land on a named arm. **P-4** is AC-5's ground-point
comparison, and the same one with the switch off. **P-5** is
`TestAnimateStaticsClosedGateIsThePreStoryList` (8 counters including `2^32−1`,
slice **identity** as well as element equality) and
`TestRenderStaticsDecodedGateNeedsTheTileBits` (five `-tick` values against the
flagless PNG, the pre-story picture pinned as the Index frame's pixels). P-1, P-2
and P-4 are **sampled, not proved**, as the spec says.

**SC-1** AC-1 and AC-6 hold in full, every expected timeline hand-written.
**SC-2** holds over all four cells and 32 counters, the transposition asserted
separately. **SC-3** holds in both diagnostic states, last column and last row
each on a case of its own. **SC-4** AC-4 and AC-9 hold, each arm and
each refusal on its own case, the bundle-less and closed-gate builds each
compared whole rather than by count. **SC-5** holds with the two frames
differing in both dimensions, ground and top-lefts compared as points. **SC-6**
holds byte for byte at two counters selecting differently sized frames. **SC-7**
holds, the counter read through both consumers as one value, 0041's digest and
byte-form checks re-run unmoved. **SC-8** is the five mutants below.

## Mutants

Each applied to production code, run over the whole tree, reverted, the tree
confirmed identical afterwards (`git status --porcelain` empty; for the two run
before their task landed, an md5 against the pre-mutation digest).

```
M1  DD-3  the stagger transposed to water's (col+1)*row              3 killed
      TestObjectStepMatchesTheContract, TestObjectStepIsTotal,
      TestSelectObjectFrameArms
M2  DD-3  the disabled arm drawing Index instead of frame 0          2 killed
      TestSelectObjectFrameArms, TestSelectObjectFrameHasNoFourthArm
M3  DD-4  the gate reduced to the period test alone
          (ObjectCycleOpen -> return period > 0)                     7 killed
      TestAnimateStaticsClosedGateIsThePreStoryList, TestObjectCycleOpenGate (5
      subtests), TestObjectCycleOpenWithoutTileWords,
      TestStaticPlacementsAnimatedSubset, TestAnimateStaticsIsTotal,
      TestLoadStatics, TestLoadMapViewerStatics
M4  DD-5  the re-anchor dropped, an animated placement keeping
          the built frame's top-left                                 2 killed
      TestAnimateStaticsReanchors, TestAnimateStaticsSwitchOffDrawsFrameZero
M5  DD-7  the counter shifted >>2 as water's is                      5 killed
      TestObjectStepMatchesTheContract, TestObjectStepIsTotal,
      TestSelectObjectFrameArms, TestAnimateStaticsReanchors,
      TestObjectsCycleAsTheCounterRises
```

No survivors. **M3 is the one that matters**: it is the baseline's own looser
gate, under which every foliage class cycles against a still original, and it
fails seven tests including P-5's own. P-5 has a witness that breaks if the gate
opens.

M4 killed only `TestAnimateStaticsReanchors` at first: `Ground()` is
`TopLeft + Anchor` and a dropped re-anchor moves neither, so the switch-off test
could not see it. An anchor and top-left assertion was added there before the
re-run, so the arm a lawful install exercises has its own witness.

## Findings in the papers

**DD-4's subset shape cannot serve FR-2's first arm, so the shape was changed.**
The plan has the subset carry "that placement's index in the list and the class
it drew from", which suffices for the cycling arm — but FR-2's *disabled* arm
reaches **every** placement, and on shipped data the subset is empty. So
`StaticPlacement` gained `Class` and the subset is the index alone; carrying the
class in both would be one fact in two places. The pass walks the subset with the
switch on and the whole list with it off, the only shape that draws frame 0
everywhere.

**Two tasks changed files outside their declared lists**, each forced by a
signature the plan fixes. T4 widens `StaticPlacements` (DD-4, DD-8), called from
`pkg/ui/viewer.go`, `cmd/terraintool/main.go` and three test files, all updated
mechanically in the same commit since a task's commit must build. T5 widens
`ui.NewViewerWithStatics` to take the gate mode, so `pkg/game/mapload.go` gained
`StaticLayer.AnimGate` (the `Art` precedent, zero value `AnimGateTiles`) and
`pkg/ui/static_draw_test.go` an argument. `flagset_test.go` gaining `-objectanim`
is that pin's own documented maintenance.

**DD-5 costs the anchor halving a second spelling.** The pass may not call
`StaticAnchor` (its signature requires the lift and the origin) and T4 forbids
editing it, so `centre − canvas/2 + frame/2` now exists twice in
`pkg/render/terrain`. R-4's warning about two nearly equal expressions in one
package applies to this pair as well as to the two staggers; AC-5's
hand-computed anchors discriminate them.

**`-objectanim` reached both front-ends**, on FR-8's "no switch of its own beyond
FR-4's diagnostic"; `-tick` is the raster tool's alone.

## What is not witnessed, and what would witness it

- **AC-10 in full.** It needs a lawful install; nothing here may read one, so no
  byte above is evidence about a shipped registry, sheet or tile grid. What would
  witness it: the owner's run of `mapview -statics` and `terraintool render
  -statics` at several `-tick` values, with and without `-objectanim` and with
  `-noanimation` — the owner-review artifact. Its three clauses do run on a
  shipped-**shaped** fixture (count 0, art identical at every counter, frame 0
  with animation off), witnessed by AC-9, P-5 and AC-8 above.
- **That the GPU texture holds the bytes of the image it was built from.** An
  `*ebiten.Image` cannot be read back before the game starts, so FR-7's
  cross-front-end equality is witnessed at the CPU image the window uploads
  (AC-7), not at the upload. What would witness it: a windowed harness with a
  real graphics context reading the framebuffer back.
- **That objects cycle while the world is stopped (FR-5).** The viewer has no
  stop and gains none, so this follows from two facts rather than one test: the
  object layer reads `Ticker.Count()`, witnessed as one value in AC-8, and 0041
  exempted that counter from the stop (`stopped_test.go` and
  `cadence_invariance_test.go` re-run unmoved). What would witness it directly: a
  map-screen test driving a stopped span and reading the drawn frames.
- **Whether the animated arm is ever entered by the original.** Unknown on
  research's side and assigned no answer here. Nothing above is evidence that
  ROM1's foliage moves or that it does not; `-objectanim` is ours and opens a
  gate we implemented.
