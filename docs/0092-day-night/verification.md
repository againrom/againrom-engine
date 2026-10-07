# Verification — 0092 day and night

Worktree `wt-0092`, branch `impl/0092-day-night`, submodule pin `9fd0559` for the lane and
**`9aab634` from the merge on** — forward, and every claim re-read at the later pin
(`git submodule status research` shows no leading character). Go 1.26.1, windows/amd64.

## What is witnessed by what — read this before the table

**This story is visual and most of it cannot be witnessed by a test.** The honest division:

- **AC-1 … AC-8, AC-11, AC-12, AC-14, AC-15 and P-1 … P-4** are witnessed by **headless tests** over
  plain Go values — an angle, a `Light`, a `[]uint8` level grid, a byte form. No window opens and no
  pixel is produced. These are real observations of the law, not of the picture.
- **AC-9, AC-10 and AC-13** are witnessed **one step short of the screen**. The level grid and the
  sprite ramp row are what the draw path consumes, and the assertions are on those; that the
  renderer then turns them into the pixels it always has is **carried by the existing draw-path
  tests, unchanged by this story, and is not re-observed here**.
- **SC-6 alone reaches a screen**, and it is a manual criterion — nothing automated in this story
  has ever drawn a frame.
- **That the picture is one a player would call a day/night cycle is not established by anything
  here.** It cannot be: the brightness half of the cycle is undecoded (D-1), so what this build
  produces is a moving *angle* on a constant brightness. The build README says so in those words.

## Gates

```
go build ./...                                    EXIT=0
go vet ./...                                      EXIT=0
gofmt -l $(git ls-files '*.go')                   EXIT=0   (printed nothing)
go test -count=1 -trimpath ./...                  EXIT=0   (31 packages ok, 0 failures)
bash scripts/check-no-game-assets.sh              EXIT=0   clean (tree scan)
bash scripts/check-no-game-assets.sh --history    EXIT=0   clean (history scan)
bash scripts/check-doc-budget.sh                  EXIT=0
bash scripts/check-sdd-audit.sh                   EXIT=0
```

`-trimpath` on the test run because Windows Defender quarantines one test binary without it.

## The deletion set

```
base=$(git merge-base origin/master impl/0092-day-night)
git diff --diff-filter=D --name-only "$base" impl/0092-day-night
```

**Empty.** No file is removed by this story.

## Acceptance criteria

| AC | Witness | Result |
|---|---|---|
| AC-1 | `TestSunAngleEndpoints` — ten band endpoints, the two computed ones written to 17 significant digits | pass |
| AC-2 | `TestSunBandCensus` — 720/240/240/240, 960 computed, 480 literal, no minute unassigned; and a second count on the angle giving 241/241 | pass |
| AC-3 | `TestSunAngleBounds` — the bound over all 1440 minutes, day arm strictly rising, night arm strictly falling | pass |
| AC-4 | `TestSunConstantsAreTheImagesBytes` — four bit patterns **on the package's own constants**, the `pi/4` inequality, `720 x` the day step, the threefold relation | pass |
| AC-5 | `TestSunAngleCycleOff`, `TestSunAtCarriesTheDaytimeParameters` — the whole `Light`, not only the angle, at every minute | pass |
| AC-6 | `TestRelightHappensOnlyOnTheCadence` — 4 moves over 1280 sub-ticks, none between | pass |
| AC-7 | `TestRelightDue` — 72 per day, first four at 0/320/640/960, and each half of the gate shown insufficient alone | pass |
| AC-8 | `TestSwitchForcesARelight` — three bands, two that move and one that does not | pass |
| AC-9 | `TestViewerWithNoClockDrawsWhatItAlwaysDrew` — the seed is the switch-off sun and the grid is `LevelGrid` at it | pass |
| AC-10 | `TestRelightRebuildsTheReliefGrid` — three literal 12-vertex grids at minutes 0/360/720, all different; flat ground 46 at five minutes | pass |
| AC-11 | `pkg/sim/binary_test.go`'s offset partition and `pkg/sim/nostate_test.go`'s field-set table, **passing, and edited by 0091 at the merge rather than by this story**; plus `TestTheCycleReachesNoSimulationState`'s round-trip and digest | pass |
| AC-12 | `TestStepLightClockWalksADayAndReturns`, `TestOffsetDoesNotMoveTheCadence`, `TestTheCycleReachesNoSimulationState` | pass |
| AC-13 | `TestTheSpriteWireIsTheCacheAndNotAConstant` — the row and the tint ARE the cache's; `TestSpritesTakeTheirLightFromTheSameCache` — neither moves over seven minutes | pass |
| AC-14 | `TestRelightKeepsTheConstructorsValidityVerdict` — all four altitude-length fixtures | pass |
| AC-15 | `TestDayNightKeysDoNothingUnpressed`, `TestTimeFlowKeyFlipsOncePerPress`, `TestDayNightKeysAreReadOnTheMapArmAlone` | pass |

## Properties

| P | Witness | Result |
|---|---|---|
| P-1 | `TestTheCycleReachesNoSimulationState` — 500 ticks with the switch flipped every 7th and the offset stepped every 7th, against a plain run: tick, digest and byte form equal throughout | pass |
| P-2 | `TestRelightIsIdempotent` — the same clock twice leaves the sun and the grid | pass |
| P-3 | `TestSunAngleIsTotal` — defined at `MaxUint64`; a whole number of days later is the same sun | pass |
| P-4 | `TestSunAngleBounds` — `stepH = 45.254833389639757` and `tan` at most 1.0 at the widest angle | pass |

## Success criteria

**SC-1 — the published landmarks.** All four reproduce from the shipped code, and so do the two
arithmetic identities nothing published asserts. Reproduced independently in a scratch program
*before* the contract was written, then again by the shipped tests: computed minutes **960** of 1440;
bands **720 / 240 / 240 / 240**; angle extremes **-0.78539815 … +0.78539815**; **72** unforced
relights per in-game day; `720 x` the day step = `1.5707963` (a truncated `pi/2`, `2.7e-8` short of
`pi/2`); the night step is the day step's threefold.

Two further landmarks are cross-checks on the angle, recorded here rather than in a test because
they belong to the shadow shear this story does not ship: the day band's `tan(shear)` range came out
`-0.5774 … +0.5754` against the published figure, the dead band at **44** full ticks against the
published 22+22, and the cycle-off `tan(shear)` at **0.57735025728078282**, equal in all 17 digits —
the sharpest check available on the constants and the arm structure together.

**SC-2 — gates.** Above; all green. The byte form is untouched: this story edits no file under
`pkg/sim` at all.

```
git diff --name-only $(git merge-base origin/master impl/0092-day-night) impl/0092-day-night -- pkg/sim
   (empty)
```

**SC-3 — the relight cost.**

```
BenchmarkRelight-20    20    2310835 ns/op        (256x256 vertices, i7-12700K)
```

**2.31 ms** per relight, against one relight per 20 in-game minutes — 320 sub-ticks of 62 ms, about
**19.8 s** of real time at the cadence a map opens on. So the rebuild occupies roughly **0.01%** of
wall time and lands well inside one frame at 60 Hz. `TestNoPacedCallCanFireTwoRelights` shows the
other half: the most whole ticks one paced call can run is **256** across every rung of the cadence
ladder and the extension past it, against a 320-sub-tick period, so no frame can ever fire two.

**SC-4 — the battery. 35 mutants, 32 killed, 3 equivalent survivors** (33/30 in the lane; the
landing added two and killed both).

Survivors first, as required.

| Survivor | Why it survives |
|---|---|
| `daySunStep` last digit `…777` → `…778` | **Equivalent.** Both decimals round to the same `float64`, `0x3f61df469d353918`. No test can distinguish them because no *program* can. |
| `nightSunStep = 0.0065449845833333332` → `3 * daySunStep` | **Equivalent.** Go evaluates untyped constant arithmetic exactly and then converts, so the expression yields the identical double `0x3f7acee9ebcfd5a4`. DD-9's "written as its own literal so the relation stays a check" is therefore a **source-reading** discipline, not one the suite can enforce — stated rather than claimed as a kill. |
| dawn band `hour <= 5` → `hour <= 6` | **Equivalent, and it says something.** The day arm (`6…17`) is tested first, so hour 6 never reaches the dawn arm. The band partition's disjointness is carried by **arm order**, not by the bounds, and a wrong dawn upper bound is invisible for that reason. |

The battery's first pass found the bit-pattern assertions were pinning the **test file's own copies**
of the four constants rather than `sun.go`'s. That is exactly the failure the calibration warns
about — an expectation computed from the thing it pins — and it was corrected: the assertions now
name the package's identifiers, and the independent transcription is kept beside them for the
behavioural comparison. A genuine digit change to the night step (`M31`) is killed after the fix.

The 30 kills, by area: every angle constant and sign (`M03`, `M04`, `M11`, `M12`, `M18`, `M31`);
every band boundary that is reachable (`M05`, `M06`, `M07`); both phase moduli (`M09`, `M10`); the
clock origin and the tick ratio (`M13`, `M14`); both halves of the relight gate (`M16`, `M17`) and
its period (`M15`); the switch's path into the sun (`M19`, `M30`); the offset's application, its
size and its forced relight (`M20`, `M26`, `M27`); the cadence test at the viewer (`M23`, `M24`);
the switch's forced relight (`M25`); the grid rebuild and its guard (`M21`, `M22`); and the clock
push and its unit (`M28`, `M29`).

**SC-5 — the mission drive, both roots.** Unchanged, character for character, from the baseline at
the branch's merge base (`0b2251a`, run in the orchestrator tree):

```
AGAINROM_ASSETS=.../gameversions/ru  go test -count=1 -trimpath -run TestTheTenthMissionIsDrivenToAWin ./cmd/missionrun/
    mission 10  scenario/10.alm  80x80  36 entities
    waypoint 1  u21 -> (56,21) r3 : reached (44,46), Chebyshev 25, after 272 ticks
    outcome lost at tick 272

AGAINROM_ASSETS=.../gameversions/en  go test -count=1 -trimpath -run TestTheTenthMissionIsDrivenToAWin ./cmd/missionrun/
    mission 10  scenario/10.alm  80x80  36 entities
    waypoint 1  u21 -> (56,21) r3 : reached (44,46), Chebyshev 25, after 272 ticks
    outcome lost at tick 272
```

Still red, identically on both roots and identically to the baseline. Nothing here was tuned toward
it and a light should not have moved it; it did not.

**SC-6 — the manual criterion. NOT PERFORMED IN THIS LANE.** No frame has been drawn by anything in
this story. The build under `builds/0092-day-night/` exists and its README names what to look at, on
what ground, and in what order — including the two things an observer would otherwise misread as
failures: flat ground is invariant under the whole cycle, and the switch starts **on**, so the first
press of `N` gives the *fixed* sun. **The criterion is outstanding and is the owner's to make.**

## What was found on the way, and what it cost

The **peer-prediction** read of `spec.md` alone and the **adversarial** read of `spec.md` + `plan.md`
were run by two independent contexts before any of this was committed. Between them they found five
material defects, all in the artifacts rather than in the code, and all fixed before implementation:

1. `DD-8` argued against version-number literals in tests; the tree **already carries one** — the
   byte form's offset partition — and it is legitimate, catching the un-bumped field. The plan also
   proposed a round-trip and a digest as the witness for "no field added", which
   `pkg/sim/nostate_test.go`'s own comment says cannot work. Both corrected, the witnesses named.
2. `FR-6`'s "one cache" was **false in this tree**: two sprite-shading sites read the package's fixed
   daytime light directly. Left alone, `AC-13` would have passed because sprites were disconnected
   from the sun rather than because the light holds still. Repointed (`DD-3a`).
3. `DD-5`'s stated reason licensed an unsafe reading of the altitude guard. Restated, and the test
   extended to all four fixtures — removing the guard now fails on exactly the two **overlong** ones,
   confirmed by mutation `M22`.
4. `FR-9` and `FR-10` contradicted each other for a view that is toggled before it holds a clock.
   `FR-10` narrowed.
5. `DD-11` claimed one line where there are **two** `push()` call sites. The push moved *inside*
   `push()`, which makes the claim true.

Two smaller: `SC-1` named two landmarks no artifact an executor holds defines (both shadow-shear —
cut, with the cross-checks moved here); and the front-end's letter/function-key convention gained a
live counterexample in `N`, so it was narrowed rather than left silently contradicted.

## At the landing — re-run in the orchestrator seat

Four defects, none in shipped behaviour; the three that admit a fix are fixed here rather than
reported.

**1. A gate line recorded from a dirty tree.** The chain above says `check-sdd-audit.sh EXIT=0`; on
the tree as committed it exits 1 —

```
FAIL 0092-day-night: plan.md accounts for no: FR13
```

— DD-7's FR-13 heading and the traceability row having been left uncommitted. Committed unchanged
(`f85208b`).

**2. AC-13's witness did not witness.** `TestSpritesTakeTheirLightFromTheSameCache` claims *the row
is the cache's* separates a sprite layer reading the sun from one reading a constant. It does not —
FR-5 makes the two the same number at every minute — and reverting **either** of `statics.go`'s
reads to `terrain.DefaultDaytime` left the suite green, so DD-3a shipped witnessed by nothing.
`TestTheSpriteWireIsTheCacheAndNotAConstant` writes a light no band can produce into the cache and
kills both.

**3. The tolerance's stated reason was not its reason.** `sun_test.go` blamed FMA fusion. Two
endpoints are the **exact decimal** value of the expression while the day arm cancels 0.785 against
0.785, so this machine lands ~4e-17 off: `SunAngle(360)` returns exactly `0` against the pinned
`-2.8e-17`. And AC-1's *"a mutation … in its last decimal digit still fails"* is false twice — that
digit moves no double, and the smallest change that does (two ulp) fails the bit patterns alone.
Comments corrected, no assertion changed.

**4. The build README's round trip needs a pause.** Its invocation is real and was run, but *press
F3 twenty-four times and the picture comes back* holds only with the tick stopped. `Space` is named
there now.

**Reproduced.** Every SC-1 landmark and both shear cross-checks, recomputed from EXP-0116's PE byte
dump by a program importing nothing of ours — to the digit. Six SC-4 kills re-applied and dead to
the tests named; both arithmetic survivors re-derived as equal doubles; `BenchmarkRelight` 2.26 ms.

**On the merged tree** (`349d039`): chain green, 31 packages, deletion set empty against both
parents, `pkg/sim` untouched against `origin/master` — `formatVersion` stays **18** and the
conditional 19 is **released**. The drive is master's baseline on both roots, `reached (43,46) …
lost at tick 272`, the 43 being 0091's cell.

## Limits

- **The brightness half of the cycle is not implemented and is not approximated** (D-1). Every band
  carries the daytime ambient, range and tint. Nothing darkens at night. This is the gap between
  what this story ships and what "a day/night cycle" means to a player, and it is a research item.
- **No shadow swings, because none is drawn.** The asymmetry a future shadow pass must reproduce —
  the shadow shears, the body computes the same value and discards it — is recorded in the contract's
  out-of-scope section so that it is not later "fixed".
- **The scrub has no on-screen acknowledgement.** Nothing displays the in-game hour, so an operator
  counts presses. A readout line was considered and dropped: it needs a number from a shared
  sequential namespace this lane was not allocated.
- `BenchmarkRelight`, the corrected bit-pattern assertions and the landing's sprite-wire test all
  arrive **untrailered**: evidence infrastructure the Verify stage produced, not work a task briefed.
