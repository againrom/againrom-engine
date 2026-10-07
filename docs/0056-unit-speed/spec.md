# Spec — a unit crosses a cell at its own rate

## Problem and current behaviour

Every entity in a world advances **one whole cell per tick**, all-or-nothing, and a map
opens at sixteen ticks a second — so every unit crosses sixteen cells a second, and a scout
and a siege engine cross them at the same sixteen. Nothing about a unit's class reaches how
far it gets: the definition table's `Speed` and `RotationSpeed` columns are decoded, carried
on the definition record and **read by nothing**, and the movement domain a placement
resolves to decides only where it may go, never how quickly.

A unit is either on one cell or on the next; there is no state between them. The window
tier draws the step it took across the **one tick** the advance took, from the cell it left
to the cell it entered, and reads it as walking while that delta is nonzero.

A world carries a passability grid — one byte per cell, two block bits — and nothing else
about the ground. There is **no terrain cost plane and no height plane**: no reader of
`data/map.reg` exists here, and the world's only constructor takes bounds, a mode, that one
grid and the entities.

## Functional requirements

- **FR-1** A mover's per-tick displacement `v`, in 1/256ths of a cell, MUST be computed
  **once per cell transit** and held for that transit, from its own speed, its movement
  domain, and the two cells' cost and height bytes:
  - **ground**: `v = clamp((m + ((m·d) >> 6)) / c, 1, 63)` with `m = SpeedMultiplier·speed`,
    `d = clamp(int8(h[src] − h[dst]), −32, +32)` (an ARITHMETIC shift, so uphill reduces),
    and `c = uint8(cost[src] + cost[dst]) >> 1` — a **byte-wide** add that wraps at 256 —
    replaced by **8** when it is zero. `SpeedMultiplier` is **8**.
  - **every other domain**: `v = clamp(speed, 1, 63)` — no multiplier, no tilt, no cost
    read. The two arms agree exactly at a multiplier of 8 and a mean cost of 8.
- **FR-2** A transit MUST take `ceil(256 / step)` ticks, where `step` is `v` for a straight
  move and `trunc(v × 0.707)` for a diagonal one. The whole computation MUST be **integer
  arithmetic** and MUST reproduce the decoded law's own values over the entire clamped
  domain `v ∈ [1,63]`; where that law's divisor is zero the transit MUST take the longest
  value the grid allows rather than being undefined.
- **FR-3** A mover that owes transit ticks MUST NOT search, step or be given a route, and
  MUST NOT be advanced by any other rule; its owed count MUST fall by exactly one per tick
  **whether or not it still holds an order**, so a transit begun always completes. It takes
  its next cell at the transit's **start** and owes the transit for having taken it, so one
  cell costs exactly `ceil(256 / step)` ticks and never one more or one fewer.
- **FR-4** The speed and the transit pair MUST be **canonical state**: carried by the byte
  form, entering the digest, and refused by the decoder in every shape a tick cannot
  produce — an owed count at or past its own transit's length, a transit longer than the
  law's own maximum, and either value nonzero on a unit that is not alive. A speed of
  **zero or less is a mover with NO RATE**: it advances one cell per tick, which is what
  every mover in this tree did before this story, so a world naming no speed anywhere
  advances exactly as it did.
- **FR-5** The map loader MUST take a placement's speed from the definition table's own
  `Speed` column, off the **same resolution** the health and the movement domain already
  come from. A placement that resolves to no units entry MUST take the base constructor's
  own default speed, so no placed unit is ever unrated and none is given a class's column
  it did not resolve to.
- **FR-6** A mover MUST be drawn **walking, and displaced between the two cells it joins,
  for the whole transit** — on the cell it left at the transit's first instant, on the cell
  it entered at its last, proportionally between — and its sprite, facing, selection mark,
  health bar and route MUST carry that one displacement together. A mover with no rate MUST
  draw exactly what it drew before.
- **FR-7** `pkg/sim` MUST gain no float: no floating-point type, literal or import, and no
  value crossing into it from one. The rate MUST be a function of world state alone — no
  clock, no pacing rate, no front-end value reaches it — so for one command stream `k`
  ticks MUST reach the same state and the same digest whatever paced them.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a ground mover of speed 16 on level ground, both cost bytes 8 | its rate and its straight transit are computed | `v` is 16 and the transit is 16 ticks |
| **AC-2** | unit | one mover in each of the three domains, at one speed | rates are taken at mean cost 8 level, then at mean cost 6 and 16, then over a slope | all three agree in the first; only the ground one moves in the other two |
| **AC-3** | unit | the ground arm | `d` is driven past ±32 both ways, the two cost bytes are made to sum past 255, their mean is made 0, and speeds above and below the clamp are used | `d` saturates at ±32, uphill reduces and downhill increases, the add wraps in a byte, a mean of 0 reads as 8, and `v` never leaves `[1,63]` |
| **AC-4** | unit | the diagonal term | `(v·707)/1000` is compared with the truncation of `v` times the law's own shipped constant for every `v` in `0…999`; then the straight transit is enumerated over `v ∈ [1,63]`; then a `v` of 1 is stepped diagonally | they agree on every `v`; the transit takes exactly **27** distinct values and `v` 52…63 all take 5; the `v` of 1 takes the longest transit rather than dividing by zero |
| **AC-5** | unit | two rated movers of different speeds and one unrated one, each ordered the same distance | the world is stepped until each arrives | each crosses one cell per its own transit and searches on no tick between, the two rated arrivals stand in the ratio their transits predict, and the unrated one arrives in one tick per cell |
| **AC-6** | unit | a world holding a mid-transit mover | it is marshalled and read back; then forms carrying an owed count at its transit's length, a transit past the law's maximum, and a transit on a corpse are read | the round trip is exact and the digest moved with each of the three fields; each form is refused, naming what it carried |
| **AC-7** | unit | a map and a definition table | a placement resolving to a units entry, one resolving to nothing, and every placement of a table-free world are built | the first takes its entry's `Speed` column, the second and the whole of the third take the constructor's default, and no entity is unrated |
| **AC-8** | unit | a rated mover mid-transit and an unrated one | the seam is pushed on every tick of a transit and the displacement asked for at several phases | the rated one reads as walking throughout, keeps one step vector and one facing, and runs from the cell it left to the cell it entered across the **transit**; the unrated one draws what it drew before |
| **AC-9** | unit | one command stream over a world of rated movers | it is run at three pacing rates and through a stop, and twice in one process | every tick index carries one digest in all of them, and the determinism wall's own checks pass unchanged |
| **AC-10** | developer-run | a lawful install, both roots | the definition table's `Speed` column is dumped with the transits it yields | the alphabet is 22 distinct values in 8…35; every one of them puts the diagonal transit between 0.97× and 1.15× of √2 × the straight one; **23 is absent**, which is what that bound requires |
| **AC-11** | manual | the game on a shipped map carrying two classes of different `Speed`, against a lawful install | both are ordered the same distance side by side | they separate visibly and the faster arrives first, each walking smoothly rather than jumping between cells |

**Error cases.** A speed is never refused — every int32 is a state the constructor accepts
and non-positive means unrated. The three refusals (FR-4) are states this package cannot
write, so no caller can provoke one from a world it built.

## Derived properties

- **P-1 (invariant)** — A transit's length is a function of the mover's speed, its domain,
  the two cells' cost and height bytes and whether the step is diagonal, and of nothing
  else; it is decided at the transit's first tick and no later tick can change it.
- **P-2 (invariant)** — No tick can leave an owed count at or past its transit's length, so
  the decoder's refusal of one can never fire on a world this package produced.
- **P-3 (completeness)** — Every mover is in exactly one cadence state: unrated, at a cell
  a tick, or rated, at a cell per its own transit. A speed decides which, there is no
  third, and no mover is in none.
- **P-4 (negative-invariant)** — Nothing outside `pkg/sim` decides a rate: no pacing rate,
  no elapsed time and no front-end field reaches the law, and the drawn displacement writes
  nothing back.
- **P-5 (invariant)** — Everything drawn on a mover carries one displacement, so a unit and
  its mark, its bar and its sprite cannot drift apart within a transit.

## I/O examples

```
ground, speed 16, cost 8+8, level    -> v 16, straight 16 ticks, diagonal 24
ground, speed  8, cost 8+8, level    -> v  8, straight 32 ticks, diagonal 52
ground, speed 35, cost 8+8, level    -> v 35, straight  8 ticks, diagonal 11
ground, speed 16, cost 8+8, d = +32  -> v 24  (128 + (128*32>>6) = 192, /8)
ground, speed 16, cost 6+6, level    -> v 21  (128/6, toward zero)
air/ghost, speed 16, any ground      -> v 16  -- no multiplier, tilt or cost
speed 0 or less                      -> unrated: a cell a tick, as before
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | A transit is a **duration**, and a position stays a whole cell. | **(A) a sub-cell position** — the original's own 1/256 pair, and the faithful one; it moves every consumer of a position in the tree and flips a mover's cell at the transit's midpoint, a rewrite of occupancy and contention beside it. **(B, chosen) a duration**, the cell taken at the transit's start; the phase divergence is disclosed. |
| **C-2** | The rate is a **pure function taking both terrain bytes**, called with zeros. | **(A) omit the terms this tree has no plane for** — the law lands half-implemented and the fork nothing exercises rots. **(B, chosen) implement it whole and exercise it whole**; the world supplies zeros, and the law's arithmetic at zero is the identity, not an approximation. |
| **C-3** | A non-positive speed is **unrated**, not a rate of one. | **(A) the law's own answer** — `v` clamps to 1 and every world naming no speed crawls at a cell per 256 ticks, changing the behaviour of every world built before this story for a value none of them meant to set. **(B, chosen) unrated**, the trade the health pair already makes. |
| **C-4** | The transit pair is **canonical**, not a front-end derivation. | The original holds both — the ticks a transit needs and the ticks it has run — in the mover block its save writes whole. A pair rebuilt at the seam would also miss a transit begun on a tick no push observed. |

## Out of scope

- **Turning.** The original charges a second rate for it and destroys the route on a large
  turn (`MOVE-TURN-031`); ours turns instantly and keeps its route.
- **The group speed override** (`MOVE-GROUP-030`): grouped units do not slow to their
  slowest member.
- **The terrain cost plane and the height plane.** Neither exists here; the law is
  implemented and exercised over both, and the world supplies zero bytes for each.
- **A sub-cell position**, the reservation of a cell before entering it, and any change
  to who blocks whom.
- **`SpeedMultiplier` read from a file**, the `Cost*` alphabet, and any customisation of
  either.
- **Animation cadence**: the walk cycle is selected on the tick clock, not by distance.

**Disclosed limitations**, accepted and owned: with no cost plane every cell reads a cost of
zero, so the law's own zero-mean arm makes ours a **uniform cost-8 map** — the shipped mode,
and the value at which FR-1's two arms agree, so they compute one number here and only a
test separates them; with no height plane the slope term is the identity; a mover takes its
next cell at the transit's **start** where the original's near cell flips at its midpoint,
so ours is visible to occupancy up to half a transit early; and every unit now crosses
ground about **sixteen times slower**, which is the defect this closes, not a regression.

## Verification mapping

AC-1 … AC-9 are unit tests over hand-built worlds and synthetic fixtures — no install, no
window, no clock — so all are CI-automatable. AC-10 needs a lawful install and is run on
both roots. AC-11 needs an install and a window. P-1, P-3 and P-5 are **sampled, not
proved**; P-2 is witnessed at every tick of AC-5's runs; P-4 is witnessed by AC-9.

## Gate check

FR-1 → AC-1, AC-2, AC-3, P-1, C-2 · FR-2 → AC-4, P-1 · FR-3 → AC-5, P-2, P-3, C-1 ·
FR-4 → AC-6, P-2, P-3, C-3, C-4 · FR-5 → AC-7 · FR-6 → AC-8, AC-11, P-5 ·
FR-7 → AC-9, AC-10, P-4.
