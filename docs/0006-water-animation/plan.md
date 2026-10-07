# Plan — animated water in the map viewer

Reading key: `FR-x`/`AC-x`/`P-x` → `spec.md`. This file fixes the API contract, the design decisions and
the success criteria the tasks are verified against.

## Shape of the change

Two tiers, one new file each plus edits:

- `pkg/render/terrain/water.go` (ADD) — the whole decoded model as pure arithmetic: water-group
  predicate, phase formula, animated resolve, speed table, `dtMs`, and the tick accumulator. Stdlib only,
  no clock, no floats in control flow.
- `pkg/render/terrain/mapping.go` (EDIT) — `Resolve` keeps its meaning (animation off = phase 0) and is
  re-expressed in terms of the shared helper so the two paths cannot drift.
- `pkg/ui/viewer.go` (EDIT) — own the accumulator and the wall clock; pass world `(col,row)` and the
  counter into the draw path; add animation on/off and speed-index setters.
- `cmd/mapview/main.go` (EDIT) — `-noanimation` and `-speed` flags; report the cadence in the `-check`
  summary so a headless run is evidence.

No new package, so no `internal/archtest` or `docs/ARCHITECTURE.md` edit is needed.

## API contract

```go
// Water strip groups and the cycle (TERR-ANIM-006, TERR-ANIM-007).
const (
    WaterGroupLo    = 8
    WaterGroupHi    = 11
    WaterPhases     = 4
    TicksPerVariant = 4   // phase uses animCtr >> 2
    WaterCycleTicks = 16  // WaterPhases * TicksPerVariant
)

func IsWaterGroup(group int) bool
func WaterPhase(group, col, row int, animCtr uint32) int   // always 0..3
func ResolveAnimated(word uint16, col, row int, animCtr uint32) TileRef
func Resolve(word uint16) TileRef                          // == ResolveAnimated at phase 0

// Cadence (TERR-ANIM-008).
const (
    SpeedIndexMin     = 0
    SpeedIndexMax     = 8
    DefaultSpeedIndex = 4   // pushed by map load
)

func TicksPerSecond(speedIndex int) int  // clamps the index
func TickMillis(speedIndex int) int      // 1000/tps, integer division

type Ticker struct{ ... }
func NewTicker(speedIndex int) *Ticker
func (t *Ticker) SetSpeedIndex(speedIndex int)
func (t *Ticker) SpeedIndex() int
func (t *Ticker) TickMillis() int
func (t *Ticker) Count() uint32
func (t *Ticker) Advance(elapsedMs int) int   // whole ticks fired; carries the remainder
```

`pkg/ui`: `(*Viewer).SetAnimated(bool)`, `(*Viewer).SetSpeedIndex(int)`, `(*Viewer).Animation() (bool, int)`.

## Design decisions

- **DD1 — The phase is a free function of `(group, col, row, ctr)`, not viewer state.** It takes the
  *stored* group (8..11), not the overridden one, because the formula's first term is the authored `g`;
  folding the override in first would drop that term. Keeping it free makes AC-1…AC-6 unit-testable with
  no engine and no viewer.
  *Rejected:* precomputing a per-cell phase offset at map load — it would have to be recomputed on every
  counter change anyway, and it would bake a 65 536-entry table for arithmetic that is three adds.

- **DD2 — `Resolve` keeps its 0004 meaning (animation off), and is defined as the phase-0 case.** The
  research says the disable switch forces `phase = 0` rather than freezing the current phase, and phase 0
  is what 0004 already draws. Expressing one in terms of the other means FR-3 holds by construction and
  AC-9 checks it over all 65 536 words. `cmd/terraintool`'s full-map composite therefore keeps rendering
  the animation-off image with no change.
  *Rejected:* changing `Resolve`'s signature to take a phase — it would churn every 0004 call site to
  express the same default, and a caller that passed a stale counter would silently render a wrong frame.

- **DD3 — The accumulator divides rather than loops.** `Advance` computes `n := acc / dtMs` and
  `acc %= dtMs` in one step. A `for acc >= dtMs` loop would burst for as long as the window was minimised;
  division is O(1) for any elapsed value and conserves time exactly (P-4).
  *Rejected:* clamping elapsed time to a maximum — that silently drops ticks, breaking P-4's conservation
  and desynchronising the cycle from wall-clock time for no benefit the division does not already give.

- **DD4 — The clock stays in `pkg/ui`; the terrain tier sees only integer milliseconds.** `Advance(int)`
  takes elapsed ms, so the whole cadence is testable by feeding numbers, with no fake clock and no sleep
  in any test. This also keeps `pkg/render/terrain` importable by headless tools.
  *Rejected:* a `time.Time`-based ticker in the terrain tier — it would put a clock behind the render
  tier's pure arithmetic and make AC-8 a timing test rather than an arithmetic one.

- **DD5 — Speed-index clamping, not rejection.** `TicksPerSecond`/`TickMillis`/`NewTicker` clamp into
  `0..8`, matching the game (the research describes a *clamped* index) and keeping every function total,
  consistent with 0004's total `Resolve`.

- **DD6 — The existing GPU cache key needs no phase field.** The phase is folded into the slot
  (`(8+phase)*4 + b`), so distinct phases are already distinct cache keys. Water costs at most 4× the
  cached images of a static map, bounded by the 128-slot tileset either way.

## Success criteria

1. **SC-1 (FR-1, AC-1)** — `IsWaterGroup` is true for exactly `8..11` over groups `0..127`. *Test:*
   `TestWaterGroupClassification`.
2. **SC-2 (FR-1, AC-2, P-1)** — the phase matches `(g + (col+1)·row + (ctr>>2)) & 3` on worked cases and
   stays in `0..3` for negative coordinates and extreme counters. *Test:* `TestWaterPhaseFormula`.
3. **SC-3 (FR-2, AC-2, AC-5, P-2)** — animated resolution preserves `b`/`sub`, selects group `8+phase`,
   reports water, and over every `(g,b,phase)` water combination yields `V = phase*4 + b ∈ 0..15` with the
   slot inside `0..127`. *Test:* `TestResolveAnimatedWater`, `TestWaterVariantReachability`.
4. **SC-4 (FR-2, AC-3, P-3)** — for every non-water word, animated == static for a spread of positions and
   counters. *Test:* `TestResolveAnimatedPassesNonWaterThrough`.
5. **SC-5 (FR-1, AC-4, P-5)** — holding a cell fixed and running the counter `0..63`, the slot changes
   exactly every 4 ticks, has period 16, and steps the phase by +1 each variant. *Test:*
   `TestWaterCycleCadence`.
6. **SC-6 (AC-6)** — horizontally and vertically adjacent water cells with equal `g` do not share a phase
   at every counter value. *Test:* `TestAdjacentCellsRipple`.
7. **SC-7 (FR-3, AC-9)** — over all 65 536 words, `Resolve(w)` equals `ResolveAnimated` with the phase
   forced to 0. *Test:* `TestStaticEqualsPhaseZero`.
8. **SC-8 (FR-4, AC-7, DD5)** — the speed table reads `8,10,12,14,16,20,24,28,32`, out-of-range indices
   clamp, and `TickMillis` is `1000/tps` by integer division (`62` at index 4). *Test:*
   `TestSpeedTableAndTickMillis`.
9. **SC-9 (FR-4, AC-8, P-4)** — the accumulator fires whole ticks only, carries the remainder, fires a
   large elapsed's whole quotient at once, ignores a negative elapsed without rewinding, and conserves
   time over a randomised sequence. *Test:* `TestTickerAccumulates`, `TestTickerConservesTime`.
10. **SC-10 (FR-5)** — the viewer defaults to animation on at the map-load index, its setters take effect,
    and `-noanimation`/`-speed` reach them; `-check` reports the cadence. *Test:* `TestViewerAnimation`
    (ui) and `TestCheckReportsCadence` / `TestAnimationFlags` (mapview).
11. **SC-11 (FR-6)** — `pkg/render/terrain` still imports only stdlib; the fail-closed DAG check stays
    green. *Test:* the existing `internal/archtest` live-tree check.
12. **SC-12 (AC-10)** — a developer run against a lawful install shows water cycling with a
    scroll-stable diagonal ripple, and the disable switch reproducing the 0004 image. *Method:*
    developer-run (manual), recorded in `verification.md`.

## Risks (product)

- **R-1 — the integer `1000/tps` is visibly wrong at some speeds.** At index 4, `1000/16 = 62` rather than
  62.5, so a cycle takes 992 ms instead of 1000. This is the game's own arithmetic (an `IDIV`), and
  reproducing it is the point. *Mitigation:* the spec states the ideal rationals alongside the integers so
  the 8 ms/cycle deficit is a recorded property, not a suspected bug; AC-7 pins the integer values.

- **R-2 — frame-rate coupling.** Ebitengine calls `Update` at its own rate, which is not the logic-tick
  rate. *Mitigation:* the counter advances from measured elapsed milliseconds through the accumulator
  (DD3/DD4), not once per `Update`, so the cycle runs at the decoded cadence regardless of frame rate, and
  P-4 makes that conservation testable without a window.

- **R-3 — the ripple could be mistaken for a bug on a large water body.** Neighbouring cells deliberately
  differ. *Mitigation:* AC-6 pins it as intended behaviour and AC-10 has the developer confirm it is
  scroll-stable — the property that distinguishes the decoded model from a screen-space artefact.
