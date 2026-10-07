# Verification — animated water in the map viewer

**Real-map evidence re-gathered 2026-07-25, against the EXP-0030-corrected `.alm` framing.** The
figures below were first taken with the pre-EXP-0030 reader, which began each grid layer eight bytes
too early: the type-1 tile grid it produced was **four cells out in X**, opening with the type-1
record header's identity words (`0001 0000 2b6d bfc0`) and dropping the map's last four authored
cells off the end. Every real-map figure in this file was re-taken with the corrected reader. They
all reproduced exactly; *why* they are invariant is set out with the census, because "unchanged" is
only evidence if the reason is stated.

## Gates

All run from a clean tree with **no game install required**:

| Gate | Result |
|---|---|
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l $(git ls-files '*.go')` | prints nothing |
| `go test ./...` | all packages ok |
| `go test ./internal/archtest/` | ok — `pkg/render/terrain` still stdlib-only (SC-11) |
| `go test ./internal/notices/` | ok — no dependency change |
| `scripts/check-no-game-assets.sh` | clean (tree scan) |
| `scripts/check-no-game-assets.sh --history` | clean (history scan) |

## AC coverage

| ID | Level | Evidence | |
|---|---|---|---|
| AC-1 | unit | `TestWaterGroupClassification` — over groups `0..127` exactly `8..11` classify as water; negative and far out-of-range groups are not water and do not panic | ✔ |
| AC-2 | unit | `TestWaterPhaseFormula` (eight worked cases against an independently written copy of the formula, including that the counter enters as `ctr>>2` and that 16 ticks returns to the start) and `TestResolveAnimatedWater` (every `g∈8..11` × `b∈0..3` × three sub-cells at three positions: slot is `(8+phase)*4+b`, the sub-cell survives the group override, and the ref reports water) | ✔ |
| AC-3 | unit | `TestResolveAnimatedPassesNonWaterThrough` — for **every** non-water word of all 65 536, five positions and counters (including negative coordinates and `^uint32(0)`) give a result byte-identical to the static resolve | ✔ |
| AC-4 | unit | `TestWaterCycleCadence` — over counters `0..63` the slot depends only on `ctr>>2`, repeats with period 16, and advances one phase per 4-tick variant; asserts explicitly that the image does **not** change inside a variant and **does** change at the boundary | ✔ |
| AC-5 | unit | `TestWaterVariantReachability` — every `(g,b)` water combination over 64 positions × 16 counters: `0` results outside tile3 variants `0..15`, `0` slots outside `0..127`, and all 16 variants are reached | ✔ |
| AC-6 | unit | `TestAdjacentCellsRipple` — a cell's right and lower neighbours both differ in phase, and the two resolve to different slots | ✔ |
| AC-7 | unit | `TestSpeedTableAndTickMillis` — the table reads `8,10,12,14,16,20,24,28,32`; indices `-3,-1,9,11,2^20` clamp; `TickMillis` is `1000/tps` by integer division; the default index 4 gives 16 tps / 62 ms / 992 ms per cycle | ✔ |
| AC-8 | unit | `TestTickerAccumulates` — twenty 10 ms steps fire exactly 3 ticks (186 ms consumed, 14 ms carried); a single 500 ms advance fires its whole quotient (8) at once; `0`, `-1` and `-1000` fire none and do not rewind; a speed change re-rates without disturbing the counter | ✔ |
| AC-9 | unit | `TestStaticEqualsPhaseZero` — over all 65 536 words, every water word's static resolve equals its animated resolve at a counter chosen to force phase 0 (the test asserts the phase really is 0 before comparing, so it cannot pass vacuously) | ✔ |
| AC-10 | **partly automated** | The objective half is measured headlessly below; the visual half is **pending manual verification** — see the end of this file | ◑ |

## Derived properties

| ID | Evidence | |
|---|---|---|
| P-1 | `TestWaterPhaseFormula` asserts `0 ≤ phase < 4` for five hostile inputs (negative group, negative column, negative row with a saturated counter, huge coordinates, non-water group) | ✔ |
| P-2 | `TestWaterVariantReachability` asserts every produced slot is inside `0..SlotCount-1` across 4 096 (word, position, counter) combinations | ✔ |
| P-3 | `TestResolveAnimatedPassesNonWaterThrough` (exhaustive over the word space) | ✔ |
| P-4 | `TestTickerConservesTime` — at speed indices 0, 4 and 8, 500 random advances of `0..199` ms each: the ticks fired equal `⌊Σelapsed / dtMs⌋` exactly and the counter agrees | ✔ |
| P-5 | `TestWaterCycleCadence` asserts `slot(ctr) == slot(ctr &^ 3)` for every counter in `0..63` | ✔ |

## Mutation check

The suite was checked for bite by mutating the implementation and confirming failures, then restoring:

| Mutation | Result |
|---|---|
| `counterShift` 2 → 1 (variant every 2 ticks instead of 4) | 4 tests fail: `TestWaterPhaseFormula`, `TestResolveAnimatedWater`, `TestWaterCycleCadence`, `TestStaticEqualsPhaseZero` |
| ripple term `(col+1)*row` → `col*row` | 3 tests fail: `TestWaterPhaseFormula`, `TestResolveAnimatedWater`, `TestAdjacentCellsRipple` |

## Headless run against a lawful install

`mapview -check` reports the configured cadence, and the flags reach the viewer. Re-run 2026-07-25
against the corrected reader, as the tool printed it:

```
mapview: Deadly Islands 256x256 cells (65536), tile slots 52/128, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)
mapview: Waters 144x144 cells (20736), tile slots 52/128, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)
mapview: Kids Paradise 80x80 cells (6400), tile slots 52/128, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)

-noanimation → mapview: Deadly Islands 256x256 cells (65536), tile slots 52/128, water static
-speed 0     → mapview: Deadly Islands 256x256 cells (65536), tile slots 52/128, water speed 0 (8 tps, 125 ms/tick, 2000 ms/cycle)
```

Every value here is unchanged from the first run. Map names and `W×H` come from the type-0 metadata
record, whose fields the correction re-labelled without moving the bytes they read; the tile-slot
count is a property of `graphics.res`, and the cadence is a property of the speed table. None of them
is derived from a grid layer.

### AC-10, objective half: the corpus expansion, reproduced by our implementation

A throwaway probe (not committed) expanded **all four phases of every water cell** of the ten shipped
root maps through our own `ResolveAnimated`, independently reproducing the TERR-ANIM-010
falsification. The expansion was re-run on 2026-07-25 against the corrected reader and **every figure
below reproduced exactly** — map by map, total, pair count, out-of-range count, variant set and blend
distribution:

| Map | Water cells | Cells cycling through 4 distinct variants |
|---|---|---|
| Beast.ALM | 6 767 | 6 767 |
| Cross.ALM | 4 772 | 4 772 |
| Forester.alm | 6 197 | 6 197 |
| Horror.alm | 6 383 | 6 383 |
| Islands.alm | 8 329 | 8 329 |
| Kids.alm | 987 | 987 |
| Kids2.ALM | 663 | 663 |
| LuMoir.alm | 1 428 | 1 428 |
| Tomb.ALM | 10 387 | 10 387 |
| Waters.alm | 2 841 | 2 841 |
| **total** | **48 754** | **48 754** |

```
(cell,phase) pairs expanded:                          195 016
pairs naming an out-of-range tile3 variant or slot:         0
tile3 variants reached:            0,1,2,3,…,15  (all sixteen)
authored blend distribution:   {0:7564  1:29469  2:7252  3:4469}
```

**Why the correction leaves every one of these numbers alone.** The mis-framing was a *rotation* of
the cell sequence, not a different grid: the old reader's tiles were `[0001 0000 2b6d bfc0]` followed
by authored cells `0 … W·H−5`, the corrected reader's are authored cells `0 … W·H−1`. So the multiset
of tile words differs by exactly eight cells — the four identity words leave, each map's last four
authored cells enter. None of the four identity words is water (their strip groups are 0, 0, 45 and
127), and on all ten maps the last four authored words are land or stone (groups 0…4), never the
water groups 8…11. Every figure in this census counts *words* or a per-word property that holds at
any position, so all of them are invariant across the correction; the re-run confirms the arithmetic
rather than merely agreeing with it.

What did move is *where* each water cell sits: the corrected grid places every authored cell four
positions earlier in row-major order — four columns left in X, wrapping to the previous row at the
map's left edge — and `WaterPhase` uses `(col+1)·row`, so the phase a given map location shows is not the
phase it showed before. No figure recorded here depends on that — the ripple table below is computed
from the formula and positions, not from map data — and the property AC-6 asserts (neighbours differ
in phase) is positional and therefore unaffected.

Three things this establishes beyond the unit tests:

1. **Zero** out-of-range results on real authored data — a wrong `b`/`V` split would push variants past
   15 into `tile3` files that do not ship, and none does.
2. **Every** water cell reaches four *distinct* variants, so no real cell is silently static.
3. The authored blend distribution's shape matches the research's independently-measured 38-map figure
   (`{0:14639, 1:50878, 2:13328, 3:8735}`) — `b=1` dominant, then `b=0`, `b=2`, `b=3`, at a consistent
   ratio. Our figure covers only the ten root maps; the research's also includes the 28 campaign maps
   embedded in `scenario.res`, which this probe did not extract.

The ripple is diagonal, as the `(col+1)·row` term requires (phase at `g=8`, counter 0):

```
row 0:  0 0 0 0 0 0 0 0 0 0
row 1:  1 2 3 0 1 2 3 0 1 2
row 2:  2 0 2 0 2 0 2 0 2 0
row 3:  3 2 1 0 3 2 1 0 3 2
row 4:  0 0 0 0 0 0 0 0 0 0
row 5:  1 2 3 0 1 2 3 0 1 2
```

Row 0 is uniform because the position term vanishes at `row = 0`; every other row carries a distinct
per-column pattern, and the whole field repeats every 4 rows. No game bytes were committed — only these
counts.

## AC-10, visual half — pending manual verification

The remaining half needs a display and a lawful install, so it is **not automated**: no test in this
repository opens a window or reads a game file. To close it, run:

```
mapview -assets <install> -map <install>/Islands.alm
mapview -assets <install> -map <install>/Islands.alm -noanimation
mapview -assets <install> -map <install>/Islands.alm -speed 8
```

and record here: that water visibly cycles at roughly one full cycle per second at the default speed;
that the ripple reads as diagonal rather than a uniform pulse; that **panning does not shift the ripple**
(the property that distinguishes the decoded world-coordinate model from a screen-space artefact); that
`-noanimation` freezes water to the 0004 image; and that `-speed 8` is visibly about twice as fast as
`-speed 4`.

## Limitations and residual risks

- **The integer `1000/tps` is the game's, not a rounding bug.** At the default index a cycle takes 992 ms
  rather than 1000. Reproducing the truncation is deliberate (R-1) and AC-7 pins the integer values.
- **Saved/multiplayer speed resumption is out of scope.** The research rates *which* index such a session
  resumes at as Medium confidence; the viewer always starts at the map-load default, so that uncertainty
  does not reach this story.
- **Water is still drawn at full brightness.** Relief lighting and the day/night tint are story 0007; the
  animation lands underneath whatever that story does to the pixels.
- **The four-variant cycle is the whole model.** No other terrain animation is claimed or implemented.
