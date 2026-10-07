# Provenance — the rate law, its clock, and the two planes we have not got

## Backing

| Spec anchor | Source | Grade |
|---|---|---|
| FR-1 — the composition: the multiplier, the tilt as an **arithmetic** `>>6`, the **byte-wide** cost add, the `c == 0 → 8` substitution, the signed divide, the `[1,63]` clamp; the rate taken once per cell transit and frozen; and a non-ground arm taking the raw speed with no multiplier, slope or cost read — the two **agreeing exactly** at multiplier 8 and mean cost 8, so no corpus can separate them | `MOVE-RATE-029`, `MOVE-DOM-024`(d) | High — every term, its address and its arithmetic form (`SAR` against `SHR`, the byte-width add, the signed `IDIV`) transcribed from a raw listing; the call site an enumeration with 0 orphan |
| FR-1 — `SpeedMultiplier` is 8, the shipped file carrying the code default in **both** roots; the cost byte is the plane at `world+0` at the two cells, the height byte the plane at `world+0x9451c` | `MOVE-PARAM-006`, `TERR-COST-052`, `TERR-PASS-049` | High |
| FR-2 — the straight step an 8-bit multiply, the diagonal `trunc(v × K)` with `K` the bytes `39 b4 c8 76 be 9f e6 3f` = `0.70699999999999996` read through the PE section table; a transit of `ceil(256 / step)` ticks, the position advancing 1/256th per axis and the surplus discarded on arrival | `MOVE-DIR-034`, `MOVE-RATE-029`, `MOVE-STEP-010` | High — the constant is the file's own bytes |
| FR-3 — one displacement per actor per **sub-tick**, in ticks against a tick count, with no elapsed-time term in the routine that writes the position | `MOVE-CLOCK-032` | High, 0 orphan |
| FR-4 — both halves of the transit pair are mover state and survive a save: the whole `0xb4`-byte mover block serializes verbatim | `MOVE-DOM-025`, `MOVE-STEP-010` | High |
| FR-5 — `Speed` is a `Data.bin` column streamed in at spawn, and a `−1` cell leaves the constructor's **10** standing | `UNIT-STREAM-001`, `UNIT-CTOR-004` | High |
| That speed never enters a route, so two units of different speed choose the **same** one | `MOVE-SPEED-011` | High |

**Which clock ours is, and how.** The law says one displacement per actor per **sub-tick**,
and a full tick is sixteen sub-ticks (`MOVE-CLOCK-032`, `SESS-TICK-004`).
`0041-world-clock` was read rather than assumed: its FR-2 pins the map-load period at
**62 ms** and the water cycle at **992 ms**, and `TERR-ANIM-008`, the claim it was built
on, has the paced loop fire one tick per `1000/tps` ms with a step counter wrapping at
sixteen — one water cycle. Sixteen of our ticks are one water cycle; sixteen sub-ticks are
one full tick; both wrap on one pacer. `MOVE-CLOCK-032` closes it from the other side: the
simulation sub-tick and the presentation tick driving the water are two calls of **one**
loop iteration. **Our tick is the sub-tick**, so the displacement lands on `sim.Step`
directly and nothing already hashed has to be reinterpreted.

## Ours by choice

| Choice | Why it is ours |
|---|---|
| A transit is a **duration**, a position stays a whole cell, and the mover takes its next cell at the transit's **start** | The original's position is a sub-cell pair whose near cell flips at the transit's midpoint; carrying that moves every consumer of a position here and rewrites occupancy beside it, where the duration reproduces the law's **timing** exactly and diverges only in phase. Taking the cell at the end would leave the drawn body displaced toward a cell no consumer of the seam holds, and the original's own reservation marks the next cell before entering it. |
| A **non-positive** speed is unrated — a cell a tick | The law's answer for speed 0 is `v = 1`, a cell per 256 ticks; every world built here before this story names no speed and none meant that. The trade a non-positive health maximum makes. |
| An unresolved placement takes the constructor's **10** | A decoded number, the one the original leaves standing for every empty cell. A hero's speed is derived from Reaction (`HERO-SPEED-008`) on an arm this tree does not model. |
| Where the law's divisor is zero — a `v` of 1 stepped diagonally — the step is taken as 1 | `ceil(256 / 0)` has no value; ours takes the longest transit the grid allows. Unreachable from shipped data, whose smallest `v` is **2**, at the steepest uphill over the costliest ground. |

## Divergence, disclosed

| What | Why, and what it costs |
|---|---|
| **The cost plane is absent.** Every cell reads 0, the mean is 0, and the law's own `c == 0 → 8` arm supplies 8. | No `data/map.reg` reader here, and no cost plane in a world. `SpeedMultiplier` is a named constant for the same reason. Ours is a **uniform cost-8 map** — the shipped mode, 58.9 % of 880 704 corpus cells (`TERR-COST-052`). Cost: the same unit does not yet take longer over rough ground, and at mean cost 8 FR-1's two arms compute one number, so only a test separates them. |
| **The height plane is absent.** Both bytes read 0, `d` is 0, the tilt is the identity. | A world's only constructor takes its planes positionally and every caller passes them so; widening it is a change of a different size from this story. The slope term is implemented and exercised whole — the world supplies the zeros. And that the ingest's height plane *is* the map's altitude grid rests on `TERR-PASS-049`'s **Medium** naming of `map+0x14`, the Medium our passability already carries for the type-3 overlay. |
| **Turning is free and instant**, and a large turn does not destroy the route. | `MOVE-TURN-031` (High) establishes a second per-unit rate off `RotationSpeed`, a snap below 33 units and a route destroyed above it; folding it in rewrites the route machinery 0029, 0037 and 0045 built. **Deferred to its own story**; until then we are **early on every direction change**, by the turn's ticks. |
| **A group order does not slow its members to the slowest.** | `MOVE-GROUP-030` is High on *what* the term is — the minimum `Speed` over the members, in `grpAI+0x44` — and **Medium** on *when* it is written, a third part Unknown. A guessed gate puts a Medium in hashed state, which the threshold forbids. |
| **The mover holds its destination cell for the whole transit.** | The original occupies its current cell and *claims* the next, holding two while crossing (`MOVE-CLAIM-007`); ours holds the destination alone, from the transit's start, so contention can differ by up to half a transit. The walk animation is on the tick clock beside it, where `ANIM-PHASE-003` advances it by distance. |

## Customisation limits (G2)

`MOVE-LIMIT-033` states these by a complete enumeration of the law's input domain — High for
the collapse classes, Medium for the alphabets. The column that matters is whether lifting
one moves a shipped file's bytes.

| Limit — each a term of the law, none of them ours | Moves a shipped file? |
|---|---|
| `v` clamped to `[1,63]`, a hard-coded pair of immediates | **No** |
| A transit is whole ticks and the surplus is discarded, so `v ∈ [1,63]` gives only **27** distinct times and `v` 52…63 all take 5 | **No** |
| The tilt is a `>>6`: a height delta under 4 changes nothing at speed 16 | **No** |
| A diagonal is 0.97…1.15× of `√2 ×` the straight transit, not `√2` | **No** |
| `Speed`, `RotationSpeed` — per-class columns | **Yes**, `world.res:data/data.bin` |
| `SpeedMultiplier` and the `Cost*` alphabet | **Yes**, `world.res:data/map.reg` |

## Open — deliberately assigned no meaning

- **A speed outside `[1,63]` after the multiplier.** The clamp is the law's, so a customised
  column past either end is brought inside it here as there, and nothing records that it was.
- **The transit owed when an order is cancelled mid-stride.** It runs to completion: a mover
  stopping half-way across a cell has no cell to stop on.

## Removed — what a reader might expect and this does not assert

| Dropped | Why |
|---|---|
| That a unit is at any *place* between two cells | The position is a whole cell and the displacement the window tier's; no test measures a sub-cell coordinate, because nothing claims one. |
