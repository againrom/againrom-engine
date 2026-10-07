# Verification — the world's own clock

Windows 11, Go 1.26.1, research submodule pin `a13b3b8` (unmoved). `-trimpath`
is load-bearing: without it Windows Defender quarantines one test binary.

## The gate

```
Base 3df2b3f. Task commits, oldest first: a67ef10 (T1), e429e50 (T2),
12aaf84 (T3), ebc726b (T4), e55b282 (T5) — five commits in the range, five
trailers, each id once, no co-author trailer:

$ git log --format='%(trailers:key=SDD-Task,valueonly)' 3df2b3f..HEAD | sed '/^$/d' | sort | uniq -c
      1 0041-world-clock/T1
      1 0041-world-clock/T2
      1 0041-world-clock/T3
      1 0041-world-clock/T4
      1 0041-world-clock/T5
$ git log --format='%B' 3df2b3f..HEAD | grep -ci co-authored-by
0
$ git log --oneline 3df2b3f..HEAD | wc -l
5

go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))"

25 packages ok, 3 with no test files, 0 FAIL.
816 test functions and 1602 subtests pass; go build, go vet and gofmt print nothing.

check-no-game-assets: clean (tree scan)
docs/0041-world-clock/analysis.md      6204 /  7168 bytes  ok (86%)
docs/0041-world-clock/provenance.md    6771 /  7168 bytes  ok (94%)
docs/0041-world-clock/spec.md         12814 / 13312 bytes  ok (96%)
docs/0041-world-clock/plan.md         12595 / 13312 bytes  ok (94%)
tasks.md T1 1089 · T2 957 · T3 1155 · T4 836 · T5 1029 / 1400 bytes each  ok
tasks.md (legend+traceability)          941 /  1200 bytes  ok (78%)
plan <= 1.2 x spec  12595 <= 15376  ok      tasks <= 1.2 x plan  6007 <= 15114  ok
check-sdd-audit: ok
```

Per package, where this story landed: `pkg/render/terrain` 145 tests / 106
subtests, `pkg/game` 52 / 103, `pkg/ui` 125 / 298, `pkg/sim` 159 / 73,
`internal/archtest` 11 / 25.

`pkg/sim` and `internal/archtest` are **byte-identical across the whole story**
(`git diff 3df2b3f..HEAD -- pkg/sim internal/archtest` is empty), so the
simulation's pinned fields, byte form, digest and both wall checks — the
import-graph check and the source scan — ran green over unedited code.
`TestSpeedTableAndTickMillis`, holding the nine decoded periods, the 62 ms tick
and the 992 ms cycle, is likewise unedited.

## What witnesses what

| id | witnessed by | result |
|---|---|---|
| **AC-1**, **SC-1** | `TestRatePeriodKeepsEveryRateWithinATenthOfAPercent` drives ten seconds in 5 ms slices at rates 1, 2, 16, 17, 64, 128, 256, 257, 969 and 1024, counting ticks; -5 and 5000 are driven the same way and must fire what 1 and 1024 fired. `TestRatePeriodIsTheExactMicrosecondQuotient` pins 1 000 000, 3 906, 976. | pass; worst error 0.0929% at rate 969 (9 ticks in 9 690) |
| **AC-2**, **SC-2** | `TestSpeedIndexPeriodWidensTheGamesOwnQuotient`: all nine indices are `TickMillis(i)*1000`; index 4 is 62 000 and its cycle 992 000; index 4 and rate 16 must **differ**; a clock there fires at 62 000 µs, not 61 999. The opened map's first tick is `TestPacedAdvanceRunsAtTheDecodedLogicRate`, unedited. | pass |
| **AC-3**, **SC-3**, **P-4** | `TestAStoppedWorldRunsNoTickAndResumesWithExactlyOne`: 600 stopped frames over 10 s asserting tick, scene, byte form, digest and cells at **every** frame; a never-stopped leg over the same instants moved; the toggle left the period unchanged; the resume ran exactly one tick. | pass |
| **AC-4**, **SC-4** | `TestOrdersIssuedWhileStoppedReachTheFirstTickAfterIt` (three orders across stopped frames, entity 0 named twice so issue order is visible; the resumed world matches one headless `sim.Step` over the written-out slice; four later ticks apply none again). `TestNoEntityFrameChangesWhileTheWorldIsStopped` (300 stopped frames on the **pushed** entities; the unstopped leg's idle class moved a frame). `TestTheStopStopsNoCameraNoSelectionAndNoOutline` (two front-ends, one input schedule: pan, zoom, outline and box release all equal to the unstopped leg's, each also checked to have moved). | pass |
| **AC-5**, **SC-5**, **P-5** | `TestOneRateReachesBothConsumers`, four subtests: an opened map makes no cadence call and both clocks run 16 a second; the ladder's 32 and 64 give 32 and 64 water ticks **and the same count on the world clock**; the stop leaves the water's rise unchanged; a viewer built as `cmd/mapview` builds one keeps index 6, its own 24 a second, and `Viewer.Input` has no cadence field. | pass |
| **AC-6**, **SC-6** | `TestOnePacedCallIsBoundedByASpanOfWorldTimeAndOwesNothing`: a 10 s stall in one call runs 4 ticks at rate 16, 64 at 256 and 1 at rate 1; the two in-bound spans are 250 000 and 249 984 µs; the next call runs one tick at all three. | pass |
| **AC-7**, **SC-7** | `TestTheCadenceKeysActOnTheMapScreenAlone`, six subtests: one act per press and none while held; the ladder 16→1024 and back to 1 with no write at either end; camera and selection unmoved; the seam call ordered **before** the same frame's advance; menu and picker arms and a left map screen ignore all three. | pass |
| **AC-8**, **SC-8**, **P-1**, **P-2** | `TestOneCommandStreamReachesOneDigestAtEveryRateAndStopSchedule`: one hand-written stream, three legs differing in rate (16, 256), in elapsed schedule (one call; two half-period calls; three uneven calls) and in stop history, byte form and digest compared at **every** one of 13 tick indices against a headless `sim.Step` assembled from that table. `TestPacingReachesNoWorldFieldAndNoSeamState` sets every ladder rate and both stop states with no advance behind any of it. | pass |
| **P-3** | `TestRatePeriodIsTheExactMicrosecondQuotient` (the clamp is total: -5, 0, 5 000 and 2²⁰ all name a rate in range) and the `flow` shape pin in `TestFlow`, which now admits `rate` and `stopped` and no third field. | pass |
| **AC-9**, **SC-10** | **NOT RUN.** | see below |

## Mutants

Four, each applied to production code, run over the whole tree, reverted, the
tree then confirmed byte-identical to `HEAD`.

```
M1  T1 — RatePeriod divides millisPerSecond; SpeedIndexPeriod drops the x1000
        (the period back in milliseconds)
    3 packages FAIL, 17 "--- FAIL" lines:
    terrain  TestTickerAccumulates, TestTickerConservesTime — then the binary
             aborts on a divide by zero (rate 1024's period becomes 0); run
             alone, TestRatePeriodIsTheExactMicrosecondQuotient,
             TestRatePeriodKeepsEveryRateWithinATenthOfAPercent and
             TestSpeedIndexPeriodWidensTheGamesOwnQuotient fail too
    ui       TestViewerAnimation,
             TestViewerStep/the_water_counter_advances_from_the_injected_timestamp
    game     TestPacedAdvanceRunsAtTheDecodedLogicRate (all 7 subtests),
             TestAnOrderedUnitWalksToItsCellAndTheDigestFollowsAHeadlessRun,
             TestAnAdvanceReportsHowManyOrdersItApplied,
             TestAGroupAdvanceIsOnePerPacedCallAtZeroOneAndManyOrders,
             TestASessionsPickAndQueueReachNoWorldBeyondTheOrder

M2  T2 — maxCatchUp returns a fixed 4
    1 kill: TestOnePacedCallIsBoundedByASpanOfWorldTimeAndOwesNothing
      "rate 256: a 10s stall ran 4 ticks in one call, want 64"
      "rate 1: a 10s stall ran 4 ticks in one call, want 1"

M3  T3 — the viewer's half of the coupling write removed:
        f.cadence(f.viewer.SetRate(f.rate), f.stopped) -> f.cadence(f.rate, f.stopped)
    3 kills: TestOneRateReachesBothConsumers and its subtests
      doubling_the_rate_doubles_the_counter,_on_both_readers
        "at rate 32 one driven second ran 16 water ticks and 32 world ticks"
      the_stop_crosses_to_the_world_and_never_to_the_water

M4  T4 — the stop's baseline write dropped, so a resume unwinds the paused span
    3 kills: TestAStoppedWorldRunsNoTickAndResumesWithExactlyOne
        "the first advance after the stop ran 4 ticks and applied 0 orders, want 1 and 0"
      TestOrdersIssuedWhileStoppedReachTheFirstTickAfterIt
      TestNoEntityFrameChangesWhileTheWorldIsStopped
```

**SC-9 holds.** The plan asked for exactly these four, one per task entry.

T5 owns no mutant, and its substitute went further than asked. Taking the
headless side from the driver under test made the comparison pass, as expected.
To check that this really removes the discriminating power rather than merely
looking like it, a fault *shared by all three driven legs* was applied —
`commands()` indexing the schedule at `Tick()+1` — and measured both ways: with
the substituted side it still **passed**; with the independent side it **failed
at tick 1**. So the independence of the headless assembly is what the comparison
rests on, measured rather than declared.

## Not run, and why

- **AC-9 / SC-10** need a lawful game install and a window. Neither exists in
  this seat, and tests here are synthetic by golden rule 2. Nothing below
  substitutes for them: what they would witness is that a human sees units
  freeze mid-stride while the camera pans and the water moves, that a resumed
  order is obeyed at once, and that units visibly walk faster and slower.
  Everything automatable in them is covered by AC-3, AC-4, AC-5 and AC-7 over
  driven instants; what is not is the *seeing*.
- No `builds/` directory was made — that is the owner's step and skipping it
  here is deliberate.

## Open items, and one the papers underdetermine

Three of the four stayed open and were not touched. The **1024 ceiling** is
exactly C-3's (`terrain.RateMax`); the **key bindings** are exactly DD-7's
(Space, `=`/NumpadAdd, `-`/NumpadSubtract) and nothing else reads them; there is
**no on-screen readout** of the rate or the stop, and nothing was added toward
one.

**`MapCadence` R-3 was touched, and forced three files T3's list does not
name.** The loader's fourth member does not compile until every loader in the
suite is edited, so `pkg/game/world_test.go`, `frontend_test.go` and
`frontend_statics_test.go` changed. Each edit is the signature and nothing else,
except one addition in `world_test.go`: the subtest that already asserted tick
and order are handed over together now asserts the cadence seam is too, and that
a failed load hands back none of the three. R-3 predicted the spread; the file
list did not.

**What the papers underdetermine — a bare pause moves a never-rated map onto the
rate model.** DD-6 says both consumers are born at the map-load speed, so an
opened map needs no call; the flow's own rate is born at 16, the map-load
speed's ticks a second. The first cadence key press therefore writes rate 16 to
both consumers *whichever key it was* — so a pause and an immediate resume,
with no rate ever selected, moves the period from the decoded 62 000 µs to the
rate model's 62 500 and the water cycle from 992 ms to 1 000. FR-2's "a map
opened and never given a rate MUST run as it does now" reads against that; P-3's
"there is no third state" and DD-5's single unconditional write read for it.

I was forced toward the unconditional write and took it. The alternative — each
consumer remembering the last rate it was told and re-rating only on a change —
invents a second rate value on each side of the seam, which is the exact thing
DD-5 exists to prevent, and P-3 forbids the "no rate yet" state that would make
it clean. The cost is 0.8% of period on a map nobody re-rated, invisible on
screen. **This is a contract question, not an implementation one**, and it is
the owner's: if FR-2's last clause is meant to survive a pause, the fix starts at
`spec.md`.

**Answered 2026-08-01 — the owner ruled 62 500 µs and 1 000 ms both acceptable.**
So P-3 and DD-5 stand as written and **FR-2's last clause was amended** instead:
it now separates a map given no cadence input at all, which runs at 62 000 µs,
from the first input of any kind, which puts the screen on the rate model at 16
and makes the cycle 1 000 ms. **No code moved and nothing in this file's
measurements changes** — the tree already behaved as ruled, which is why this
was a contract question. The 0.8 % is recorded as a disclosed divergence in
`provenance.md`, the document that owns divergences.

## Two smaller readings recorded rather than assumed

- **AC-6's "the same span at all three".** At rate 1 the bound (250 ms) is
  shorter than one tick (1 s), so DD-4's floor gives one tick and that call
  covers 1 000 000 µs, not 250 000. The test asserts the equal-span clause at 16
  and 256 (250 000 and 249 984 µs, within one period of each other) and the floor
  case separately, which is how AC-6's own "and at least one tick at rate 1"
  reads.
- **The gate before the last push** was piped to `tail`, masking
  `check-sdd-audit.sh`'s exit code. Its one finding was the one this file closes
  — *every task has landed and there is no verification.md*. Unpiped after this
  file lands, it is green.

## Conclusion

Every automatable criterion is witnessed by something that runs; the two that
are not are named above with what they would witness. The threshold rests on a
negative — the rate and the stop are never hashed and never serialized — and
that negative is carried by a test that would fail if it moved: one command
stream, three pacings, digests compared at every tick index against an
independently assembled headless run whose independence was itself measured.
