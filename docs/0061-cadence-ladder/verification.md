# Verification — 0061 cadence ladder

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

`git diff -M --diff-filter=D --name-only f621b75..HEAD` is **empty**: nothing was deleted.

## The defect, confirmed against the tree

Both halves of the reading this story was opened on hold.

- `mapWorld`'s clock was born at `SpeedIndexPeriod(DefaultSpeedIndex)` — the game's own
  `1000/16 = 62` ms widened, **62 000 µs**.
- The front-end's ladder was born at `TicksPerSecond(DefaultSpeedIndex)` = a **rate** of 16, and
  both consumers divided that rate for themselves at `1000000/16` = **62 500 µs**.

One correction to the reading, and it does not change the conclusion: the ladder's opening value
was not the literal `16` but `TicksPerSecond(DefaultSpeedIndex)`, which *is* 16. So the two
starting values were derived from the same speed index by two different arithmetics — which is a
sharper statement of the same defect, not a different one.

The owner's three numbers reproduce exactly: 62 000 at open, one `+` doubling the rate to 32 for
`1000000/32` = 31 250, one `−` halving back to 16 for **62 500**, with 62 000 unreachable
thereafter.

## Where the meaning of the keys lives now

One place: the ladder in `pkg/render/terrain`, on the lines under `RatePeriod`/`RateOf`. What was
removed to get there:

| Removed | Was |
|---|---|
| `flow.rate` and the `*= 2` / `/= 2` in `stepCadence` | the front-end's own parallel model |
| `terrain.RatePeriod` in `Viewer.SetRate` | the water counter dividing the rate for itself |
| `terrain.RatePeriod` in `mapWorld.setCadence` | the world's clock dividing the same rate again |
| `terrain.ClampRate` in the front-end | a second spelling of the ladder's bounds |

`stepCadence` now holds no table, no division and no bound: it adds one, subtracts one, clamps
through the ladder's own clamp, and hands `CadencePeriod` of the result to both consumers on one
statement. `MapCadence`, `Viewer.SetPeriod` and `setCadence` all carry a period, and neither
consumer computes one.

## Mutants run, and what each killed

Each was applied alone, to a clean tree, and reverted.

```
MUTANT A - setCadence stores the period beside the clock and the readout states the copy:
  FAIL TestReadoutFollowsTheClockWhenNothingAsked   (pkg/game)
  - and nothing else in the tree.

MUTANT B - SpeedIndexOf drops its exactness test and answers with the nearest rung's speed:
  FAIL TestCadenceRungIsTotalPastBothEnds                  (terrain)
  FAIL TestTheReadoutSaysWhichSideOfTheShippedSetItIsOn    (ui)
  FAIL TestReadoutStatesEveryFieldOfTheTable               (ui)

MUTANT C - the keys step TWO rungs instead of one:
  FAIL TestOneRateReachesBothConsumers/a press moves both readers to the same cadence
  FAIL TestOneRateReachesBothConsumers/the stop crosses to the world and never to the water
  FAIL TestTheCadenceKeysActOnTheMapScreenAlone/each key acts once per press and not while held
  FAIL TestTheCadenceKeysActOnTheMapScreenAlone/the keys walk every rung and clamp at both ends
  FAIL TestTheCadenceKeysActOnTheMapScreenAlone/n presses each way come back ...
  FAIL TestTheKeysMoveWhatTheBoxStates

MUTANT D - the shipped rungs built from RatePeriod(TicksPerSecond(i)) instead of the game's own
           period, i.e. "round 62000 to 62500":
  FAIL TestTheLadderIsTheShippedTableWithTwoDisclosedExtensions   (terrain)
  FAIL TestCadenceRungIsTotalPastBothEnds                         (terrain)
  FAIL TestTheMapLoadCadenceIsOnTheLadder                         (terrain)
  FAIL TestAFreshlyOpenedMapStatesTheGamesOwnTick                 (pkg/game)
  FAIL TestReadoutStatesEveryFieldOfTheTable                      (ui)
  FAIL TestOneRateReachesBothConsumers/...  and two more in ui
```

Mutant A is the one that matters for 0060's rule, and it is now killed by **one** test rather than
three. That is a real loss of redundancy and is stated rather than left to be discovered: with the
ladder and the clock finally agreeing, the two other 0060 tests can no longer discriminate a shadow,
because on every path where nobody bypasses the seam the shadow holds the right number. Their
comments were rewritten to say what they witness now, and
`TestReadoutFollowsTheClockWhenNothingAsked` — whose fixture re-rates the clock *behind* the
front-end, so a shadow is stale there by construction — carries the claim alone.

## Two tests whose claims changed, restated rather than left to be misread

- **`TestAFreshlyOpenedMapStatesTheGamesOwnTick`** was 0060's witness that the box is not fed by
  the ladder, and its discriminating power came from the very disagreement this story removes. It
  now pins the other half — that 62 000 is not rounded to 62 500 — and gained an assertion that the
  opening period is a rung. Mutant D kills it.
- **`TestReadoutStatesTheClockAndNotTheRequestedRate`** is renamed `...RequestedCadence` and walks
  the whole ladder plus four off-ladder periods. Its "what this does not prove" paragraph is
  *weaker* than 0060's and says so: the seam now adopts a period verbatim, so a copy beside the
  clock is the identical number and this test cannot discriminate one even in principle.

## Evidence per criterion

| Id | Evidence |
|---|---|
| **AC-1** | `TestTheMapLoadCadenceIsOnTheLadder` — the opening period is rung 7, reports as speed index 4, and every rung is reachable from it in both directions. `TestAFreshlyOpenedMapStatesTheGamesOwnTick` through the world. |
| **AC-2** | `TestTheLadderIsTheShippedTableWithTwoDisclosedExtensions` — all 17 rungs hand-written with period, rate and shipped-or-not; `TestTheLadderIsStrictlyDecreasingAndEndsAtTheRateBounds` — monotonicity, both ends against `RatePeriod(RateMin/RateMax)`, and the extension's doubling checked *across* its join with the table. |
| **AC-3** | `TestTheCadenceKeysRoundTripExactly` — every start rung × every n up to 4 past the ladder's length (357 runs), against the closed form, plus the identity assertion for every non-saturating run. Driven: `.../n presses each way come back to the cadence the map opened at`. |
| **AC-4** | `TestOneRateReachesBothConsumers/a press moves both readers to the same cadence` — six presses out of the shipped set and into the extension; at each, the period that crossed the seam, and the water counter and the world clock driven over the same second. |
| **AC-5** | `.../the keys walk every rung and clamp at both ends` — three presses past each end make no further cadence call. |
| **AC-6** | `TestAFreshlyOpenedMapStatesTheGamesOwnTick`, `TestTheMapLoadCadenceIsOnTheLadder` — both compare 62 000 against `RatePeriod(16)` explicitly and fail if they are equal. |
| **AC-7** | `TestTheReadoutSaysWhichSideOfTheShippedSetItIsOn` — every rung's text hand-written, plus three off-ladder periods, plus the nine shipped rows re-derived from `SpeedIndexPeriod`. |
| **P-1** | `TestCadenceRungIsTotalPastBothEnds` — rungs past both ends, periods slower and faster than the whole ladder, zero and negative. |
| **P-2** | `TestTheCadenceKeysRoundTripExactly` — the closed form IS the saturation rule, so a wrap or a drift fails it at the ends. |
| **P-3** | `git diff --name-only f621b75..HEAD` names no file under `pkg/sim`. `TestPacingReachesNoWorldFieldAndNoSeamState` now walks **every rung** of the ladder with the stop set and cleared at each, and asserts one distinct period per rung; `TestOneCommandStreamReachesOneDigestAtEveryRateAndStopSchedule` compares the canonical byte form and the digest at every tick index across three legs. |
| **SC-1** | The three terrain tests named under AC-1/AC-2/AC-6. |
| **SC-2** | `TestCadenceRungRoundTripsEveryRung` (all 17), `TestTheCadenceKeysRoundTripExactly` (357 runs). |
| **SC-3** | `TestTheCadenceKeysActOnTheMapScreenAlone` — the walk up, the walk down through all nine shipped settings, both clamps; `TestOneRateReachesBothConsumers`; `TestTheKeysMoveWhatTheBoxStates`. |
| **SC-4** | `TestReadoutStatesTheClockAndNotTheRequestedCadence` (17 rungs + 4 off-ladder), `TestReadoutFollowsTheClockWhenNothingAsked` (mutant A), `TestTheDrawPathHasNoSecondSourceOfTheRate`. |
| **SC-5** | `TestTheReadoutSaysWhichSideOfTheShippedSetItIsOn`, `TestReadoutStatesEveryFieldOfTheTable`. |
| **SC-6** | The gate block at the top, the empty deletion set, and the two digest suites under P-3. |

## The ambient animation counter — ruled, not held

**It follows this ladder, and that is a decode rather than a decision.** `TERR-ANIM-007` (High)
finds exactly two writes to the counter in the whole binary, the increment being the handler of the
paced logic tick; `ANIM-CLOCK-001` (High) has that same function advance every drawable's animation
and increment that counter in one pass, and states the consequence for a consumer in as many words:
the frame "must be stepped by the same tick as the simulation — it slows when the game speed
drops". So the counter is not a second clock that agrees by luck, it is the same tick, and this
story makes that structural: one period value, adopted by both.

Its **stop** is a different question and is not reopened here. 0041 FR-5 keeps water running while
the world is stopped; `ANIM-CLOCK-001` adds "and stops when the tick stops", which turns 0041's
"nothing decoded" into a disclosed divergence. It is recorded in `provenance.md` and left standing:
the original has no pause, so the claim describes a state the game cannot enter. No research item
was opened.

## The cost, measured

```
BenchmarkReadoutCompose-20     62333 ns/op   77858 B/op   17 allocs/op
BenchmarkReadoutCached-20        435.4 ns/op  2592 B/op    4 allocs/op
BenchmarkReadoutHidden-20          0.8582 ns/op   0 B/op   0 allocs/op
BenchmarkReadoutSubject-20       422.0 ns/op  2592 B/op    4 allocs/op
```

The composed box is **168 × 102** with the suite's 6-pixel test font, against 0060's 168 × 94: the
new row costs 8 pixels of height and no width, since the box was already at its `MinWidth` floor and
`SETTING` fits inside it. Composition went from 58.5 µs to **62.3 µs**, +6.5%, which is what one
more row of a fill-dominated box costs. Recomposition still happens only when a stated value moves.

## Limitations, disclosed

- **The extension's shape is ours and is not claimed to be anything else.** Nothing decoded says
  the original could run at 1 or 1024 ticks a second, and nothing decoded says what its own keys do
  at the ends of its table. Both are in `provenance.md`.
- **The round trip cannot be total.** No clamped ladder is invertible for a run that saturates —
  the map is not injective — so what is proved is the identity for every run that stays on the
  ladder and the exact closed form for every run that does not. The alternative, remembering
  presses past an end, was refused: it makes a key visibly do nothing for a while afterwards.
- **No pixels were compared against a screenshot.** The new row's text is asserted on the resolver
  and its presence on the shipped layout; that the window then blits the box is one `DrawImage`,
  unasserted, exactly as every other row's is.
- **The tick counts in the driven cadence cases are exact only where the period divides a second.**
  Where it does not, the accumulator's carried remainder can move a driven second's count by one, so
  those rungs assert that the two consumers **agree** rather than a fixed number — and that
  agreement is the claim the case exists for. The far side's clock is advanced over the press frame
  too, so the two carry the same remainder into the second.

## The build

`builds/0061-cadence-ladder/` holds `againrom.exe` and a `README.md` whose first two lines are the
keys. The first instruction is the owner's own check: press `+` and `−` the same number of times and
watch the period come back to 62000.
