# Verification — 0056-unit-speed

Gate on the final tree, one `&&`-chain, no pipes:

```
go build ./... && go vet ./... && go test -trimpath -count=1 ./... &&
sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh &&
sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))")
EXIT=0   FAIL set: EMPTY
```

The baseline taken on the branch before the first edit was `EXIT=0` with an empty FAIL
set, so the two are comparable. `check-sdd-audit` emits one WARNING per story with a build
and ten `DECLARED OVERRUN` notices from a worktree that has no `builds/`; only the FAIL set
is enforced and only it is compared.

## Acceptance criteria

| # | Witness | Result |
|---|---|---|
| **AC-1** | `TestTheWorkedExampleReproduces` — speed 16, cost bytes 8 and 8, level: rate 16, straight transit 16 ticks | pass |
| **AC-2** | `TestTheGroundArmIsTheOnlyOneThatReadsTheGround` — the three domains at one speed over mean cost 8, then 6, then 16, then a slope each way: all three agree on the first, only the ground one moves on the other four | pass |
| **AC-3** | `TestTheGroundArmsSaturationsAndSubstitutions` — the tilt saturating at ±32, uphill below level below downhill, an INEXACT tilt (an arithmetic shift where a divide differs), the byte-wide cost add at 200+100, a mean of 0 and a mean of 1-and-0 taking the substitute, and the clamp exactly at each end and one past it | pass |
| **AC-4** | `TestTheDiagonalRatioIsExactAgainstTheShippedConstant` (every `v` in 0…999 against `math.Float64frombits(0x3FE69FBE76C8B439)`), `TestTheTransitCollapsesIntoTwentySevenClasses` (27 distinct, 52…63 all at 5, 51 at 6, monotone), `TestADiagonalCostsMoreThanAStraightStep` (never quicker; the zero-step diagonal takes 256) | pass |
| **AC-5** | `TestARatedMoverCrossesOneCellPerTransit` (cell, route and target still on every tick of a crossing; the step on the tick after), `TestTwoSpeedsArriveInTheRatioTheirTransitsPredict`, `TestADiagonalCellCostsMoreInAWorldToo` — measurements below | pass |
| **AC-6** | `TestTheConstructorRefusesACrossingNoTickCanLeave` (five shapes, each named by its own error, plus the three legal boundaries), `TestACorpseCarriesNoCrossing`, `TestEachNewFieldReachesTheDigest`, the version-7 pin and its round trip, and the six new spoil cases in `TestUnmarshalRefusesAndLeavesTheReceiverExactlyAsItWas` | pass |
| **AC-7** | `TestOnlyAMatchedUnitsEntryYieldsItsOwnSpeed` (six placements over every arm, two matched rows at 8 and 35, none of them the default; the table-free build at the default throughout; no entity unrated), `TestASpeedAndAHealthComeOffOneResolution`, `TestADefaultSpeedIsTheConstructorsOwn` | pass |
| **AC-8** | `TestACrossingIsDrawnWalkingForItsWholeLength` and `TestTheCrossingCrossesTheSeamWholeOverAWholeWalk` (pkg/game), `TestTheDisplacementIsTakenOverTheCrossing` and `TestACrossingOfOneOrNoneDrawsWhatItAlwaysDrew` (pkg/ui) | pass |
| **AC-9** | `TestOneCommandStreamReachesOneDigestAtEveryRateAndStopSchedule` and `TestOneCommandStreamReachesOneDigestWhateverIsDrawnBetweenItsTicks`, both re-run over rated worlds; `internal/archtest`'s import DAG and determinism scans | pass |
| **AC-10** | developer-run, both roots — below | pass, with one figure narrower than the criterion says |
| **AC-11** | manual, owner's seat — `builds/0056-unit-speed/README.md` | **owed to the owner** |

## Derived properties

| # | Witness |
|---|---|
| **P-1** | Sampled. The rate is a function of six values and nothing else by its signature; that it is decided ONCE per crossing is `TestARatedMoverCrossesOneCellPerTransit`, which asks the pair on every tick of a crossing and finds only the owed count moving. |
| **P-2** | Witnessed at every tick of AC-5's runs, and structurally: the count is written strictly below its length and only ever falls, so `transitFault`'s refusal cannot fire on a world this package wrote. `TestAMoverFelledMidCrossingCarriesNone` marshals and reads back the world a kill leaves. |
| **P-3** | Sampled by `TestOnlyAPositiveSpeedIsARate` over six speeds either side of the line, and by AC-5's unrated leg crossing a cell a tick. |
| **P-4** | AC-9: one stream, three pacing schedules and a stop, one digest at every tick index; and the determinism scan, which reports no float, no banned import and no forbidden literal in `pkg/sim`. |
| **P-5** | Sampled. `pkg/ui`'s displacement is one function every glyph family calls, and 0047's own sprite/mark/bar agreement tests run unchanged over it; the crossing tests above add the span to the same call. |

## The measurements

Ticks to arrive, ten cells, printed by the suite itself:

```
speed  8: transit 32 tick(s) a cell, 10 cells in 288 ticks
speed 16: transit 16 tick(s) a cell, 10 cells in 144 ticks
speed 35: transit  8 tick(s) a cell, 10 cells in  72 ticks
speed  0: transit  1 tick(s) a cell, 10 cells in   9 ticks   (unrated)
speed 16 over 4 cells: straight 48 ticks, diagonal 72 ticks
```

At the map-load rate of sixteen ticks a second those are 18.0 s, 9.0 s and 4.5 s for the
same ten cells. A rate wired to a term that is not the speed comes out 1:1 here and passes
every arithmetic case above.

The diagonal against its published band, over the whole of 8…35 (test log):

```
rate straight diagonal  ratio to root-two        rate straight diagonal  ratio
   8       32        52  1.1490  <- the maximum    22       12        18  1.0607
  16       16        24  1.0607                    23       12        16  0.9428  <- OUTSIDE
  17       16        22  0.9723  <- the minimum    34        8        11  0.9723
```

Exactly one rate in 8…35 falls outside `[0.97, 1.15]`, and the extremes over the rest are
0.9723 and 1.1490 — the two published bounds to two figures. So the band is a **prediction**
that the shipped alphabet does not contain 23.

## Developer run — both lawful roots (AC-10, SC-10)

`classdump -databin <root>/world.res <map> 2` over every loose `.alm` of each root, EN and
RU. The union of the speeds that reach a world:

```
8 9 10 11 12 13 15 16 17 18 19 20 21 24 25 26 27 32 33 34 35     21 values, EN
8 9 10 11 12 13 15 16 17 18 19 20 21 24 25 26 27 32 33 34 35     21 values, RU — identical
```

**23 is absent from both**, which is what the published band requires — the prediction is
met. Every value lies in 8…35, as published.

The count is **21 and not the 22 the published alphabet has**, and the difference is this
run's scope rather than a disagreement: the census is over the speeds PLACEMENTS reach a
world with, not over the table's own column, so a column value no loose map places does not
appear — and the campaign's embedded maps live inside `scenario.res`, which this verb does
not open. One of the 21, the value 10, is also the loader's default for an unresolved
placement, so between 20 and 21 of them are column values. What the run establishes is the
absence, which is one-directional and unaffected by scope: no map of either root places 23.

One per-map census, EN `Horror.alm`, 1815 placements:

```
speed 8: 15   10: 475   11: 150   13: 79   18: 57   19: 173   21: 122   27: 570   35: 174
domain ground 1371   ghost 230   air 214   total 1815 of 1815
```

The 475 at speed 10 are exactly the 474 server-id placements plus the 1 humans placement —
the arms that resolve to no units entry — which is the default reaching a world where it
should and nowhere else. RU `Horror.alm` reports 0 placements: it is the truncated file the
research already records, and it is reported rather than skipped.

## Mutation kills

Sixteen mutants over the arithmetic that decides a hashed byte, applied one at a time and
reverted. **16 of 16 killed**, thirteen by a test whose name says what broke:

```
M1  diagonal 707 -> 708            KILLED TestTheDiagonalRatioIsExactAgainstTheShippedConstant
M2  transit rounds down not up     KILLED TestTheTransitCollapsesIntoTwentySevenClasses (+2)
M3  slope >>6 -> /64               KILLED TestTheGroundArmsSaturationsAndSubstitutions
M4  cost add widened past a byte   KILLED TestTheGroundArmsSaturationsAndSubstitutions
M5  zero-mean substitute 8 -> 1    KILLED TestTheGroundArmsSaturationsAndSubstitutions (+2)
M6  domain fork inverted           KILLED TestTheGroundArmIsTheOnlyOneThatReadsTheGround (+1)
M7  multiplier 8 -> 1              KILLED TestTheWorkedExampleReproduces (+4)
M8  ceiling clamp off by one       KILLED TestTheGroundArmsSaturationsAndSubstitutions
M9  owed count t-1 -> t            KILLED TestARatedMoverCrossesOneCellPerTransit (+7)
M10 every step read as straight    KILLED TestADiagonalCellCostsMoreInAWorldToo (+2)
M11 transit gate > 0 -> > 1        KILLED TestARatedMoverCrossesOneCellPerTransit (+7)
M12 rated: > 0 -> >= 0             KILLED TestASettledOrderPointsAtTheCellItSettledFor (+47)
M13 TransitTotal dropped from the  KILLED TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth
    byte form                             (+8)
M14 the seam's crossing skip cut   KILLED TestOneCommandStreamReachesOneDigestWhateverIsDrawn... (+4)
M15 displacement over the tick     KILLED TestTheDisplacementIsTakenOverTheCrossing
M16 loader ignores the column      KILLED TestOnlyAMatchedUnitsEntryYieldsItsOwnSpeed (+1)
```

M3, M4 and M8 **survived the first run**, and each named a case that measured nothing rather
than a rule that was unchecked: the shift and a divide agree wherever the quotient is exact,
200+200 comes out at the clamp's floor either way, and no case produced a rate of exactly
the ceiling. Three cases replaced them and the three now die. That finding is the reason
this section exists.

## Success criteria

SC-1 → AC-1/2/3 · SC-2 → AC-4 · SC-3 → AC-4 · SC-4 → AC-5 · SC-5 → the whole pre-existing
`pkg/sim` suite, unchanged and green · SC-6 → AC-6 · SC-7 → AC-7 · SC-8 → AC-8 ·
SC-9 → AC-9 · SC-10 → the developer run above · SC-11 → AC-11, owed · SC-12 → the gate, and:

```
git diff --diff-filter=D --name-only 8c178d0..HEAD    (empty)
```

No file was deleted by this story.

## What a reader should know that no criterion asked

- **A mover under a crossing takes no stall count**, because a stall counts consecutive
  ticks on which a near search found nothing and such a mover makes none. Without it a
  32-tick crossing — twice the give-up limit — would lose its own order half-way across one
  cell. `TestAMoverOwingACrossingTakesNoStallCount`.
- **`mapload.Schedule`'s legs are no longer the ticks a leg takes.** Its orders are spaced
  24 ticks apart, which was a full leg while a cell cost one tick; an un-commanded unit now
  takes its next corner before finishing a leg and drifts around its start instead of
  walking laps. The function reads only a map and cannot see a speed, so a spacing derived
  from one would be an invention; it is left alone and recorded here. **A unit the player
  orders is unaffected** — the driver cuts a commanded entity out of the script for good.
- **Every digest and form length in `pkg/mapload` moved twice** on this branch: once when
  the record widened, once when the loader began filling the speed. Each pin says in place
  what moved it and what it read before.
