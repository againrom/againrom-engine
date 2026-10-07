# Verification — selecting a group and ordering it from the running game

Run on Windows 11, Go 1.26.1, no game install present and no window opened. Every
figure below was produced by the command printed beside it.

```
$ go build ./... && go vet ./...                     clean, no output
$ go test -trimpath -count=1 ./...                   25 packages ok, 3 with no test files
$ go test -trimpath -count=1 -v ./... | grep -c PASS 2493 (765 top-level)
$ sh scripts/check-no-game-assets.sh                 check-no-game-assets: clean (tree scan)
$ sh scripts/check-no-game-assets.sh --history       check-no-game-assets: clean (history scan)
$ sh scripts/check-doc-budget.sh                     ok (spec 13232/13312, plan 13202/13312, chain ok)
$ sh scripts/check-sdd-audit.sh                      ok
$ gofmt -l $(git ls-files '*.go') | wc -l            0
```

`-trimpath` is load-bearing here: Defender quarantines a test binary without it.

## What each criterion was measured by

| Id | Evidence |
|---|---|
| **AC-1** | `TestShiftDecidesWhetherALeftPressPansOrDrawsABox`, 4 rows: with no world, plain and Shift both pan (-20,+10); with one, Shift pans (-20,+10) and plain pans (0,0). The accumulator reads 130 in all four. |
| **AC-2** | `TestABoxReplacesTheSelectionWithTheUnitsItCovers`, 10 rows over a camera at (-8,40) zoom 2: three units caught, one, cleared over empty ground, three corner-swappings equal to the un-swapped set, zero width, zero height, the coincident-points case, and a release wholly outside. Each replaces a prior selection of three. |
| **AC-3** | `TestATapOverANonIdentityCameraYieldsASetOfOne`, same camera: a hit replaces the set of three with one; an empty cell clears; an off-map point clears; two units on one cell yield the lower id in either slice order. |
| **AC-4** | `TestARightPressOrdersEveryPresentMemberAscending`: 3 orders naming one cell in ascending id, with the snapshot given ascending AND descending; 2 orders where one member is absent; none in each of the four refused cases (nothing selected, outside the extent, every member gone, a left press in progress). |
| **AC-5** | `TestAGroupOrderedToOneCellArrivesOneUnitDeep` over a hand-built 9x5 field. Targets field by field at the applying advance; no shared cell after any of 21 ticks; entity 14 reached (7,2) at tick 5 and stands there at quiescence, exactly one unit on the cell. **No arrival is claimed for the rest** — recorded below. |
| **AC-6** | `TestEverySelectedPresentUnitIsMarkedOnItsOwnCell`, displaced and flat: 3, 2, 1 marks and an absent id beside two present ones, each rim on its own cell with its own cell's lift (three different lifts on this fixture). The frame still draws 1 unit sprite and 2 entity squares. |
| **AC-7** | `TestTheOutlineRunsFromTheFrameThePressPassesTheSlop` (no pass at 1, 2, 3 px of travel; a pass at 4, 5 and 44), `TestTheOutlineIsAbsentForEveryOtherPress` (under-slop press, Shift drag, no-world viewer — every frame and the release), `TestTheOutlineIsFourStripsLeavingItsInteriorUncovered`. |
| **AC-8** | `TestTheGestureIsFixedAtThePressAndNotReadAgain`: Shift pressed mid-drag still pans (0,0); Shift released mid-drag still pans (-20,+10). The two rows answer each other's numbers exactly. |
| **AC-9** | `TestAGroupOrdersDigestFollowsAHeadlessRunAtEveryTick`: 40 ticks, byte form and digest against a headless run over the literal stream, and two runs of one script compared tick by tick. `TestNoOrderEscapesAPressThatNeverReleases`: a press held 7 frames with a right press on each issues zero orders. |
| **AC-10** | `pkg/sim`, `internal/archtest` and `cmd/mapview` are **byte-for-byte untouched** by this story (`git diff --name-only 6a1154a..HEAD` matches none of them) and their suites pass: the pinned fields, the byte form, the digest, both wall checks (`TestLiveTreeClean`, `TestSimSourcesAreDeterministic`) and the standalone viewer's render and headless line. The advance count is one at zero, one and many orders — `TestAGroupAdvanceIsOnePerPacedCallAtZeroOneAndManyOrders` and the map arm's own subtest. |
| **AC-11** | **NOT RUN.** It needs a lawful install and a window; neither is available in this environment. Owed by the owner-review artifact. |
| **P-1** | Sampled in `TestAGroupQueueAndTheFrontEndsRectangleAreNotWorldState`: a group's orders queued with no advance behind them leave tick, bounds, entities, byte form and digest exactly where they were. |
| **P-2** | Same test, plus `TestASessionsPickAndQueueReachNoWorldBeyondTheOrder`, whose session leg now also resolves a screen rectangle through the camera's own inverse and reads the ids it catches. Its digests match a bare leg, a headless leg and differ from an unordered one at every tick. |
| **P-3** | `TestEveryEdgeLatchAndSelectionLandsInExactlyOneOutcome`: 36 cases — 4 edges (tap, box, pan-latched release, right) x hit/miss/outside x a prior selection of none/one/many — each landing in exactly one of the four outcomes, against a table transcribed from the contract. |
| **P-4** | Sampled by AC-9's two-runs-of-one-script comparison, which is byte form and digest at every one of 40 ticks. |
| **P-5** | `TestNoOrderEscapesAPressThatNeverReleases`, with the non-vacuity control: the same right press with no press in progress issues three orders. Every box row of AC-2 also asserts the release issued none. |
| **SC-1** | AC-1 and AC-8 above, both drags driven through the shipped `dragIntent`, the no-world half on a viewer built as `cmd/mapview` builds one, the accumulator read in every case. `TestThePressPointOutlivesTheDragAnchor` shows the press point still readable three ticks in where the drag anchor is not. |
| **SC-2** | `TestScreenToCellRangeAtOnePointIsTheCellScreenToCellNames` (10 positions and zooms, each pinned against `ScreenToCell` itself), `TestScreenToCellRangeCoversTheCellsTheRectangleMeets` (10 rows: degenerate, straddling, and one wholly past each of the four edges), `TestScreenToCellRangeIsOrientationIndependent` (3), `TestScreenToCellRangeRefusesNonFiniteCorners` (4). |
| **SC-3** | AC-2 and AC-3 above. The camera is at (-8,40) zoom 2 throughout — neither offset nor the zoom is the identity — and every case replaces a prior selection of three. |
| **SC-4** | AC-4, P-3 and P-5 above. The ascending order is asserted on the emitted slice with the snapshot given in both id orders. |
| **SC-5** | AC-5 and P-1 above: targets compared field by field at the applying advance, the shared-cell check after every tick, the surplus recorded as an observation. |
| **SC-6** | AC-6 above; `TestASelectionWithNoPresentMemberAppendsNoPass` asserts the empty and all-absent cases on the **pass slice itself** — the frame holds exactly the pass count a viewer with nothing selected holds. |
| **SC-7** | AC-7 above. The four strips are checked against the rectangle's own interior at all four drag orientations, and `TestTheOutlineIsAppendedAfterEveryOtherPass` reads its position in the slice. |
| **SC-8** | AC-9 and AC-10 above, and P-2's sample. |
| **SC-9** | Six mutants, one per entry, each applied to production code, run over the whole tree and reverted with the file confirmed byte-identical. Table below. |
| **SC-10** | **NOT RUN**, with AC-11 and for the same reason. |

## The mutants, as measured

| Entry | Mutant | Killed by |
|---|---|---|
| T1 | the latch read live in `dragIntent`'s difference branch | both rows of `TestTheGestureIsFixedAtThePressAndNotReadAgain` — each row returned the other's numbers |
| T2 | the range's half-open `+1` dropped | `TestScreenToCellRangeAtOnePointIsTheCellScreenToCellNames` (10/10 rows), `TestScreenToCellRangeCoversTheCellsTheRectangleMeets` (4 rows) |
| T3 | the range's normalisation removed | `TestScreenToCellRangeIsOrientationIndependent` (3/3), `TestABoxReplacesTheSelectionWithTheUnitsItCovers` (3 corner-swapped rows) |
| T4 | the emitted order slice reversed | `TestARightPressOrdersEveryPresentMemberAscending` (2 rows), the map arm's k-calls-in-ascending-id subtest |
| T5 | the presence filter deleted from `presentSelected` | `TestEverySelectedPresentUnitIsMarkedOnItsOwnCell`, `TestASelectionWithNoPresentMemberAppendsNoPass`, `TestSelectionHighlightAbsentOrNoneDrawsNothing`, `TestARightPressOrdersEveryPresentMemberAscending` — **both readers**, which is what says the one predicate serves both |
| T6 | the command-mode gate deleted from the latch | `TestShiftDecidesWhetherALeftPressPansOrDrawsABox`, `TestCommandModeIsSetWithTheSeamAndDroppedWithIt`, `TestTheOutlineIsAbsentForEveryOtherPress`, `TestTheSlopSeparatesATapFromADrag`, and 3 subtests of the shipped `TestViewerStep` — the standalone viewer's pan, exactly as the entry predicted |

T7 owns no mutant. What was broken instead is its own independence, in two
measurements, both reverted:

- skewing the applying tick by two while the literal command stream stays at tick
  3 **fails** the byte-form comparison at tick 3, so the comparison discriminates;
- taking the headless side from the driver's own `commands()` under that same skew
  makes every byte-form and digest comparison **pass**, leaving only the
  unordered-leg clause standing.

## The group order, as the world actually resolves it

Four units ordered to one free cell on a 9x5 open field, advanced to quiescence:

```
entity 14 reached (7,2) at tick 5
entity  2 was ordered to (7,2) and the world left it on (5,1)
entity  5 was ordered to (7,2) and the world left it on (5,2)
entity  9 was ordered to (7,2) and the world left it on (5,3)
the run settled after 21 ticks  (arrival at 5, then stallLimit = 16)
```

This is the disclosed limitation, measured: the group arrives **one unit deep**
and the surplus freezes one cell short rather than settling on nearby free cells.
It is ours and worse than the original, which substitutes a nearby goal when its
own search budget fails; nothing here is presented as faithful to it.

## Limitations

AC-11 and SC-10 are not run: both need a lawful install and a window. Everything
else is automated and reproduced by the commands above. P-1, P-2 and P-4 are
sampled rather than proved — the samples are named above.
