# Verification — 0060 debug readout

Environment: Windows 11, Go as pinned in `go.mod`, 12th Gen Intel Core i7-12700K.
No game install was read by anything below; every fixture is built in test code.

## The gate

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"
EXIT=0
check-sdd-audit: ok (36 note(s)/warning(s), none enforced)
```

FAIL set: **empty**, and byte-identical to the baseline taken on this branch before the first edit,
which was also empty. Note and warning counts are not comparable from a worktree — it has no
`builds/` for the stories that predate this one — so only the FAIL set is reported.

## The one rule this story is about

**Where the cadence lives.** Three things in this tree are rate-shaped:

| | What it is | Used? |
|---|---|---|
| `flow.rate` | the front-end's key ladder — what a press doubles or halves | **No.** Unreachable from the draw path: unexported field of the flow, and a viewer holds no pointer to one. |
| `Viewer.anim` | the water counter's period | **No.** Re-rated through a second call site, and it keeps running while the world is stopped. |
| `mapWorld.clock` | the accumulator the paced advance divides elapsed time by | **Yes.** Read through `Ticker.Period()` in `pushReadout`, on the same statement as the stop, the tick and the digest. |

**The test that would fail on a shadow copy — and the correction that got it there.** The first
attempt asserted that a rate requested past the ceiling is stated as the clamp, and called that
discriminating. It is not, and the mutant proved it: `setCadence` writes the clock from
`RatePeriod(rate)` on one statement, so a rate copied beside the clock yields `RatePeriod` of the
same number — the identical period for every input, in range and out. **A mutant storing the request
passed the entire file.** That claim was withdrawn and two tests replaced it, each checked by
mutation:

- `TestReadoutFollowsTheClockWhenNothingAsked` — the clock is re-rated by a path that tells nobody.
  Any record kept at `setCadence` is stale there, which is the only shape a shadow can take that is
  not simply the same expression written twice.
- `TestAFreshlyOpenedMapStatesTheGamesOwnTick` — the one place the two genuinely disagree in
  production, before a key is pressed: the world's clock opens at the game's shipped **62000 µs**
  while the ladder opens at a rate of 16, whose period in our model is **62500 µs**. Same stated
  rate, two clocks. A box fed by the ladder states 62500.

Mutants run, and the FAIL lines they produced:

```
MUTANT A — a rate stored at setCadence, reported as RatePeriod of it:
  FAIL TestReadoutStatesTheWorldsOwnTickAndDigest
       a freshly opened map states 62500 us, want the map-load period 62000 us
  FAIL TestReadoutFollowsTheClockWhenNothingAsked
       re-rated to 512/s behind the front-end, the readout still states the last
       REQUESTED 16/s - it is reading a counter beside the clock, not the clock
  FAIL TestAFreshlyOpenedMapStatesTheGamesOwnTick
       an opened map states 62500 us - the period the LADDER's rate implies, not
       the 62000 us the world's clock is holding

MUTANT B — the front-end's ladder pushes the readout, the world pushes nothing:
  FAIL TestReadoutStatesTheClockAndNotTheRequestedRate  (every row of the table)
```

`TestTheDrawPathHasNoSecondSourceOfTheRate` (ui tier) had its claim narrowed for the same reason:
its pusher is a stand-in, so it witnesses that the **draw** path reads what was pushed and consults
nothing else, not what production pushes.

## Evidence per criterion

| Id | Evidence |
|---|---|
| **AC-1** | `TestReadoutIsShownByDefault`, `TestTheReadoutKeyTogglesOncePerPress` — a zero `Viewer` literal reports shown, and an opened map is showing it before any key. |
| **AC-2** | `TestTheKeysMoveWhatTheBoxStates` — `+` doubles the stated rate and halves the period, `-` reverses it, driven through `App.step`. |
| **AC-3** | Same test: twenty doublings from 16 and twenty halvings, i.e. twelve presses past the ceiling and sixteen past the floor. The stated value is the clamped one at every step. |
| **AC-4** | `TestReadoutStatesTheClockAndNotTheRequestedRate` over `{16, 32, 256, 1024, 2048, 4096, 1<<20, 1, 0, -7}`, plus the two mutation-checked tests above. |
| **AC-5** | `TestTheKeysMoveWhatTheBoxStates` (Space, then Space again; rate unchanged across both) and `TestReadoutStatesTheStopOnTheFrameItIsSet`. |
| **AC-6** | `TestReadoutStatesTheWorldsOwnTickAndDigest` — 40 paced frames, tick and digest compared against the world's own on each; and the stopped span in `TestReadoutStatesTheStopOnTheFrameItIsSet`, where the pushed value does not move over 18 frames. |
| **AC-7** | Same test — compared against `World.Hash()` rather than a pinned constant, because what is under test is that the box reports *this* world. |
| **AC-8** | `TestReadoutOmitsWhatItCannotState`, `TestReadoutUnitLinesComeFromTheSelectedEntry` — the three unit rows appear and vanish together and the box is shorter by exactly three. |
| **AC-9** | `TestTheReadoutKeyTogglesOncePerPress` — press, press, and a held key over three frames acting once. |
| **AC-10** | `TestReadoutAndPanelDoNotMeet` — the two boxes' rectangles are disjoint at 1024×768, and the unit panel's pixels and origin are identical with the readout shown and hidden, with no recomposition provoked. |
| **AC-11** | `TestReadoutNeedsAFont` — nothing presented, nothing composed, no failure. |
| **AC-12** | `TestReadoutCursorIsTheGroundPicksOwnAnswer` — four cursor positions including two off the map, each compared against `Camera.ScreenToCell` for the same pixel; an unstepped viewer states the absence marker rather than cell (0, 0). |
| **AC-13** | `TestReadoutStatesTheGroupRateBesideTheSpeed`, `TestReadoutGroupRateComesFromTheEntry`, `TestDrawnEntityCarriesTheGroupRate` — speed 40 beside group 12, stated as two numbers. |
| **P-1** | `TestReadoutStatesTheClockAndNotTheRequestedRate` and `TestReadoutFollowsTheClockWhenNothingAsked` — ten requested rates and four bypass re-rates reach the same stated cadence whenever they leave the same period. |
| **P-2** | `TestReadoutRecomposesOnlyWhenTheStatedValuesChange` — a repeat frame returns the same pointer and the same origin and composes nothing. |
| **P-3** | `TestReadoutWritesNothing`, `TestTheReadoutKeyMovesNothingElse` — camera, selection, pushed value, water counter and period, and the lattice, all unmoved. |
| **P-4** | `TestReadoutStatesEveryFieldOfTheTable`, `TestReadoutOmitsWhatItCannotState` — every field states a value or omits its row; no empty line is drawn. |
| **P-5** | `TestRateOfRoundTripsEveryRateInRange` — all 1024. |
| **SC-1** | `TestReadoutStatesEveryFieldOfTheTable`, `TestComposeItemsIsThePanelsOwnPath` — the field table and the shared composition, with no window and no graphics context. |
| **SC-2** | The two mutation-checked tests above, with the withdrawn claim recorded. |
| **SC-3** | `TestRateOfRoundTripsEveryRateInRange`, `TestRateOfIsTotalPastBothEnds`, `TestRateOfReadsTheGamesOwnSpeedPeriods`. |
| **SC-4** | `TestTheKeysMoveWhatTheBoxStates`, `TestTheDrawPathHasNoSecondSourceOfTheRate`. |
| **SC-5** | `TestReadoutStatesTheWorldsOwnTickAndDigest`, `TestDrawnEntityCarriesTheSpeed`. |
| **SC-6** | `TestTheReadoutKeyTogglesOncePerPress`, `TestTheReadoutKeyIsReadOnTheMapArmAlone`, `TestHiddenReadoutComposesNothing`, `TestHiddenReadoutIsNotPushed`, `TestReadoutNeedsAFont`. |
| **SC-7** | The benchmarks below. |
| **SC-8** | `TestReadoutAndPanelDoNotMeet`, `TestReadoutWritesNothing`, `TestReadoutOmitsWhatItCannotState`. |
| **SC-9** | `TestReadoutOmitsWhatItCannotState`, `TestReadoutCursorIsTheGroundPicksOwnAnswer`, `TestReadoutUnitLinesComeFromTheSelectedEntry`. |
| **SC-10** | `TestReadoutStatesTheGroupRateBesideTheSpeed`, `TestReadoutGroupRateComesFromTheEntry`, `TestDrawnEntityCarriesTheGroupRate` — every fixture makes the two numbers differ. |

## The cost, measured

```
BenchmarkReadoutCompose-20     58508 ns/op   69494 B/op   15 allocs/op
BenchmarkReadoutCached-20        335.7 ns/op  2592 B/op    4 allocs/op
BenchmarkReadoutHidden-20          1.333 ns/op   0 B/op    0 allocs/op
BenchmarkReadoutSubject-20       218.0 ns/op  2592 B/op    4 allocs/op

BenchmarkPushReadout/entities=1-20      382.6 ns/op    352 B/op   1 allocs/op
BenchmarkPushReadout/entities=64-20    4167   ns/op   3456 B/op   1 allocs/op
BenchmarkPushReadout/entities=256-20  14710   ns/op  13578 B/op   1 allocs/op
BenchmarkPushReadoutHidden-20             1.200 ns/op   0 B/op    0 allocs/op
BenchmarkWorldHash-20                 14209   ns/op  13568 B/op   1 allocs/op
```

The composed box is **168×94** with the suite's 6-pixel test font. The composition's cost is
dominated by area (the frame fill writes every pixel), so the shipped font — taller — scales it
roughly with its own line height; at a plausible 10 pixels the box is about 136 tall and the
composition about **85 µs**. That figure is an extrapolation and is labelled as one; the 58.5 µs is
measured.

**Worst case per frame**, box shown, recomposing every frame (the top of the rate ladder, where a
tick fires faster than the frame rate), 256 entities: 58.5 + 14.7 ≈ **73 µs**, or **0.44%** of a
16.67 ms frame. At the map-load cadence of 16 ticks a second the box recomposes 16 times in 60
frames and the rest cost 336 ns each, so the true average is far below that.

R-1 is answered: the digest is the larger half at high entity counts and it is linear in them, but
at 256 entities it is 0.09% of a frame. R-2 is answered: the recomposition ceiling is the frame
rate, and that ceiling costs 0.35%. `F1` takes both to about a nanosecond.

## Limitations, disclosed

- **The frame-rate line is not exercised against the engine.** `ebiten.ActualFPS()` is read at the
  one call site in `Draw` and handed in as an integer; every test supplies it. Its rounding is
  tested, its value is not.
- **No pixels were compared against a screenshot.** The box's placement, size, row set and text are
  asserted on the composed `image.RGBA`; that the window then blits it is one `DrawImage` with the
  unit panel's own arithmetic, and is unasserted, exactly as the unit panel's is.
- **The benchmark font is not the game's.** Stated above with the extrapolation labelled.
- **The cursor line will disagree with the `F2` lattice on slopes.** That is FR-6, deliberate, and
  the divergence belongs to the ground pick rather than to this box. It is written down in the
  build's README so the owner does not read it as a defect.
- **A viewer that is never stepped states no cursor cell.** Correct, and worth naming because it
  means the line is blank-ish until the mouse moves once over a freshly opened map.

## The build

`builds/0060-debug-readout/` holds `againrom.exe` and a `README.md` whose first two lines are the
keys. The README names what each line is and what should move when the owner presses Space, `+` and
`-`, and calls out the three checks worth doing first — the ceiling, the stop, and the group rate
that does not clear.
