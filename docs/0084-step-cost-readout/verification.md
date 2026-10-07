# Verification — the step cost, relative to the mover

Rebased onto master `8892726` (0081's landing plus the research pin bump to `8fa25b2`) and the whole
gate re-run afterwards. Research facts read from the submodule at that pin.

## The gate

```
go build ./...                    EXIT=0
go vet ./...                      EXIT=0
gofmt -l $(git ls-files '*.go')   EXIT=0  (listed nothing)
go test -count=1 -trimpath ./...  EXIT=0  (31 packages ok, 0 failed)
scripts/check-no-game-assets.sh   EXIT=0
scripts/check-doc-budget.sh       EXIT=0
scripts/check-sdd-audit.sh        EXIT=0
```

Deletion set against this branch's own merge base — empty:

```
$ git diff --diff-filter=D --name-only 8892726 HEAD
$
```

## FR-8 — the simulation's state does not move

```
$ git diff --stat 8892726..HEAD -- pkg/sim
 pkg/sim/step.go          | 134 ++++++++++----
 pkg/sim/steprate_test.go | 316 +++++++++++++++++++++++++++++++++
 pkg/sim/world_test.go    |  11 +-

$ git diff --stat 8892726..HEAD -- pkg/sim/binary.go pkg/sim/world.go pkg/sim/hash.go
$
```

The three files that own state, the byte form and the digest are untouched. `formatVersion` reads
**14** and is 0081's, not this story's — see the correction below. `nostate_test.go`, which pins
`World`, `Entity`, `Bounds` and `rng` field sets by reflection, is unedited and green.

**A correction, because the first version of this got it wrong.** FR-8 was asserted by a Go test
pinning `formatVersion` at **13**, the value this story was written beside. It passed, and then
failed on the rebase because 0081 had legitimately moved it to 14. The test was not measuring "0084
changed nothing" — it was measuring "the number is still the one 0084 was written beside", which is
a claim this story has no standing to make and a tax on every later story that must move it. It is
removed, with the reasoning left at its site; the empty diff above and `nostate_test.go` are the
evidence, and `TestStepRateWritesNothing` covers the runtime half (bytes and digest identical across
a query).

## AC evidence

| | Evidence |
|---|---|
| AC-1 | `TestStepRateIsTheNumberTheAdvanceUses` — the query is asked, the same world is then advanced onto that very cell, and the transit the tick recorded is compared with the transit predicted. Run over five terrains: no planes, cheap, dear, uphill, downhill. The extra four are load-bearing — at multiplier 8 and mean cost 8 the law's two arms agree, so a default-terrain-only test cannot tell "computes the law" from "returns 16" |
| AC-2 | `TestStepRateFollowsTheCellUnderTheDestination` — one cost byte is the only difference between the two worlds |
| AC-3 | `TestStepRateIsRelativeToTheMover` — two movers, one destination cell, uniform terrain |
| AC-4 | `TestStepRateTakesTheGroupTermOverTheUnitsOwnSpeed` — asserted both ways: equal to what the term alone gives, and unequal to what the class speed alone gives |
| AC-5 | `TestStepRateSeparatesUphillFromDownhill` — two movers on one height plane asking about each other's cell, so nothing differs but the order of the pair |
| AC-6 | `TestADiagonalStepTakesLongerThanAnOrthogonalOne` — same rate, longer transit |
| AC-7 | `TestTheStepRowsGoWithTheOtherUnitRows`, plus the two existing row-count pins moved from 3 to 5 |
| AC-8, AC-9, AC-11 | `TestTheStepRowsStateAnAbsenceRatherThanAZero` (three causes from the viewer side), `TestStepRateRefusesWhatNoTickWouldRate` (five causes from the simulation side), `TestNoQuestionIsAskedWithoutBothOfItsInputs` |
| AC-10 | `TestTheStepRowsMarkAPairThatIsNoSingleStep`, `TestStepRateReportsWhichPairsAreOneStep` — the eight neighbours and four distant cells, and a marked pair still carries a figure |
| AC-12 | the two diffs above, `TestStepRateWritesNothing`, `nostate_test.go` unedited |
| AC-13 | `TestBothStepValuesAreInTheRebuildKey` (each value moved alone), `TestTheStepRowsRecomposeTheBox` (through the viewer's own build counter) |
| AC-14 | `TestTheShippedLayoutCarriesBothStepRows` |
| FR-3 end to end | `TestTheViewerAnswersWithTheWorldsOwnStepFigures` compares what the front-end can reach against `StepRate` called directly — an identity, not a remembered pair of numbers |

## Success criteria

| | Result |
|---|---|
| SC-1 | met. The advance's transit now comes from `stepRate`, and `rate_test.go`, `costrate_test.go`, `transit_test.go`, `replay_test.go` and `hash_test.go` are **unedited** and green — which is the whole of R-1's evidence |
| SC-2 | met. `TestStepRateIsTheNumberTheAdvanceUses` over five terrains, `TestStepRateFollowsTheCellUnderTheDestination`, `TestStepRateSeparatesUphillFromDownhill`, `TestStepRateTakesTheGroupTermOverTheUnitsOwnSpeed` |
| SC-3 | met. `TestStepRateRefusesWhatNoTickWouldRate` — five cases, each asserting the refusal *and* that the returned numbers are zero, so a caller ignoring the flag is handed no figure either |
| SC-4 | met. `TestStepRateReportsWhichPairsAreOneStep` — all eight neighbours, then four cells beyond |
| SC-5 | met. `TestTheStepRowsStateWhatTheQuestionAnswered`, `TestTheStepRowsStateAnAbsenceRatherThanAZero`, `TestTheStepRowsGoWithTheOtherUnitRows`, `TestTheStepRowsMarkAPairThatIsNoSingleStep` |
| SC-6 | met. `TestTheStepRowsStateAnAbsenceRatherThanAZero/no_question_was_ever_installed` and `TestAViewerNobodyInstalledOnAnswersNothing` |
| SC-7 | met. `TestBothStepValuesAreInTheRebuildKey` moves the rate, the transit, the marker and the presence **each alone**; `TestTheStepRowsRecomposeTheBox` reads the viewer's own build counter and also composes twice, which is R-2's check |
| SC-8 | met. The two diffs above; `nostate_test.go` unedited and green. The literal-version half of this was withdrawn — see the correction above |
| SC-9 | met. `internal/archtest` green; `pkg/ui`'s allow-map row is unedited and the package gained no import |
| SC-10 | met. `TestTheShippedLayoutCarriesBothStepRows`; `panel.go` untouched; the two new field numbers are 28 and 29 and no existing readout number moved; `readoutHidden` keeps its inverted storage, so the box is still shown by default and its hide key still hides it |

## Properties

**P-2 — one composition site.** `rateOf` has exactly one caller and `stepRate` exactly two, and the
two are the advance and the query:

```
$ grep -rn "rateOf(\|w.stepRate(" pkg --include=*.go | grep -v _test.go
pkg/sim/rate.go:97:func rateOf(d Domain, speed int32, costSrc, costDst, hSrc, hDst uint8) int32 {
pkg/sim/step.go:572:            _, t := w.stepRate(*e, from, cell{x: e.X, y: e.Y})
pkg/sim/step.go:908:    v := rateOf(e.Domain, moverSpeed(e), costSrc, costDst, hSrc, hDst)
pkg/sim/step.go:953:    r, t := w.stepRate(e, from, to)
```

Line 908 is inside `stepRate` itself. `TestStepRateIsTheNumberTheAdvanceUses` is the behavioural
half — an identity between what the query predicts and what the tick records — and M4 shows a
divergence between the two would be caught.

**P-1 — no number the movement code would not use.** `TestStepRateRefusesWhatNoTickWouldRate` covers
all five refusals; M1, M2 and M3 each remove one gate and each is killed. The gates are the advance's
own (`Alive`, `rated`), read from where the advance reads them rather than restated.

**P-3 — the drawing tier names no simulation.** Its production import set is unchanged by this story:

```
$ grep -rh "againrom/pkg/" pkg/ui/*.go | grep -v _test | grep -o 'againrom/pkg/[a-z/]*' | sort -u
againrom/pkg/render/camera
againrom/pkg/render/frame
againrom/pkg/render/menu
againrom/pkg/render/terrain
againrom/pkg/render/text
```

`StepCost` is three ints and a bool; `StepCostFunc` takes a `uint32` and two `int`s.

**P-4 — what is drawn is what is keyed.** `TestBothStepValuesAreInTheRebuildKey` moves each of the
four components alone; `TestTheStepRowsRecomposeTheBox` asserts through the viewer that an unchanged
frame does **not** recompose and a moved rate does.

## Which terms of the published law this tree has

`TERR-MOVE-056` is `● active` at pin `8fa25b2` and is not in `retracted.md`.

Present: the effective-speed source (group term else own), the domain fork, the height difference
clamped at ±32, the multiply, the arithmetic `>> 6` tilt, the byte-wide cost add, the mean, the
`c == 0 → 8` substitute, the signed divide, the `[1,63]` clamp, the diagonal constant as an exact
integer ratio, and `ceil(256/step)`.

**Absent, and disclosed rather than folded in** — both named in `stepRate`'s doc comment:

1. **The multiplier's map parameter.** `rateOf` applies it at the shipped default 8, written as a
   constant; the original reads it per map from `data/map.reg [Path Finding]`. `MOVE-PARAM-006`
   (High) establishes that both preserved roots ship all seven `[Path Finding]` scalars equal to the
   code defaults, identically in EN and RU — so the figure is correct for everything that ships, and
   a **customised** map would be rated as though it had not been customised.
2. **The cost accessor's write-back.** `costAt` is a pure indexed read. The original's `R1087`
   is not: on a cell whose block byte has `0x20` set and whose record in the `world+0x540b8` table
   has a nonzero byte at `+0xe`, it shifts the stored cost right by 2 and **stores it back**. Its
   cost plane is therefore stateful — a cell can answer a quarter of what it answered before — and
   ours is not. On such a map this figure is not a rounding away from the original's, it is a
   different quantity. Nothing in this tree reads that bit or that table.

A third difference is structural rather than missing: the published law derives a step's destination
from a facing byte through a direction table, and this tree derives a mover's step from its route
search. The readout is asked about a cell the reader points at, so the destination is an **input**
here and neither derivation is exercised.

## Mutation testing

Each mutation was applied to the landed source, the package's tests run, and the source restored.
Two mutations that failed to **compile** were reformulated — a mutation the compiler catches is not
evidence about the tests.

| Mutation | Killed by |
|---|---|
| M1 drop the `rated` gate in the query | `TestStepRateRefusesWhatNoTickWouldRate/a_mover_of_zero_effective_speed` |
| M2 drop the `Alive` gate | same test, `/a_mover_that_is_dead` and `/a_mover_that_is_downed` |
| M3 drop the same-cell refusal | same test, `/a_destination_equal_to_the_source` |
| M4 swap source and destination in the composition | `TestStepRateIsTheNumberTheAdvanceUses/a_step_downhill` and `/a_step_uphill` — and by nothing else, which is why those two cases exist |
| M5a/b/c adjacency always true, widened to ±2, losing −1 on x | `TestStepRateReportsWhichPairsAreOneStep` |
| M6 never diagonal | `TestADiagonalStepTakesLongerThanAnOrthogonalOne`, and the pre-existing `TestADiagonalCellCostsMoreInAWorldToo` |
| M7 drop the bounds guard | the pre-existing `TestATransitOffTheMapComposesAsItDidBeforeThePlanes` |
| N1 step rows ignore the selection | `TestTheStepRowsGoWithTheOtherUnitRows` and both row-count pins |
| N2 an unanswered step states a zero | `TestTheStepRowsStateAnAbsenceRatherThanAZero`, all three subtests |
| N3 the `FAR` marker never appears | `TestTheStepRowsMarkAPairThatIsNoSingleStep` |
| N4 ask with no cell resolved | `TestNoQuestionIsAskedWithoutBothOfItsInputs/no_cell_under_the_cursor` |
| N5 the rate row states the transit | `TestTheStepRowsStateWhatTheQuestionAnswered` |
| N6 ask about entity 0 | `TestTheQuestionIsAskedAboutTheSelectedUnitAndTheCursorsCell` |
| **N7 the column and row cross the seam transposed** | **SURVIVED**, then killed — see below |
| W1 the simulation's refusal is not passed through | `TestTheDriverAddsNoRefusalOfItsOwn`, all three subtests |
| W2 capture the world at install instead of at the call | `TestTheQuestionFollowsTheDriversCurrentWorld` |
| W3 column and row transposed at the wiring | `TestTheViewerAnswersWithTheWorldsOwnStepFigures`, `TestTheDriverAddsNoRefusalOfItsOwn` |
| W4 rate and transit swapped at the wiring | `TestTheViewerAnswersWithTheWorldsOwnStepFigures` |
| W5 the question is never installed | three tests |

**N7 is the one worth recording.** It survived the first version of the viewer suite: the fixture put
the cursor at screen (100,100), which the camera resolves to cell **(3,3)**, so a column and a row
that reached the far side the wrong way round were invisible. The fixture now uses (200,120) → cell
**(6,3)**, and the test fails the *fixture* — not the code — if that ever stops being asymmetric.
The same defect at the tier below was caught by W3 only because that fixture happened to be
asymmetric already.

## Scope reconciliation

Three changes outside the planned file sets, each recorded rather than absorbed:

- **`pkg/sim/world_test.go`** (T1 fenced "no existing test file is edited"). The exported
  method-set pin fired on `StepRate`, which is what that pin exists for. `StepRate` is declared a
  reader and swept. That fixture gives no entity a speed, so the sweep stops at the `rated` gate —
  stated in a comment there, and `TestStepRateWritesNothing` carries the rest rather than the sweep
  being taken for more than it proves.
- **`pkg/ui/readout_test.go`** (T2). Two pins counting the rows a selection adds move from 3 to 5.
  The count stays a stated literal, so a sixth row still has to come and say so.
- **`pkg/ui/readout.go` in T3** (not in T3's planned files). `StepCostAt` is `ReadoutState`'s twin:
  without it the installation is unobservable from `pkg/game` and T3's own `Done when:` could not be
  evidenced. `tasks.md` was cascaded to name the file.

`DD-1` was carried by no task in the first traceability table; the audit found it and T1 and T2 now
carry it.

## Limitations

- **No frame was captured and the built binary was not run against an install.** What is asserted
  above is what the code does, measured by tests with no engine. Whether the two rows read well in
  the box is the owner's to look at.
- The rate is stated as a bare integer. `MOVE-STEP-010` establishes it as 1/256 of a cell per axis
  per tick, but the denominator is a simulation constant and printing it here would put one in the
  drawing tier.
- The figure for a pair more than one cell apart is the law on that ordered pair and is marked `FAR`.
  It is a single hypothetical step; no path is searched and none is implied.
- Nothing here establishes what the original computes for a mover of zero effective speed — the
  claim transcribes the arithmetic, not the caller's guard. This tree follows its own advance, which
  does not rate such a mover, rather than guessing.
