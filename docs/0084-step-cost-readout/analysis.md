# Analysis — what a step onto that cell costs this mover

## What was not known

The readout states a unit's `SPEED` and its `GROUP` term, and how many ticks the crossing it is
**currently on** runs for. None of those answers what a step onto a *named* cell would cost that
unit: the two speed rows are inputs the law consumes, and the crossing row is the answer for a step
already taken. What was not known is whether this tree holds that number at all, or only some of
the terms that produce it.

## The published law, term by term, against this tree

`TERR-MOVE-056` publishes `R1088` end to end. Setting its terms beside what this tree
computes had to come before any number could be drawn, because a readout that presents a partial
law as a whole one is worse than no readout — it is believed.

| Term of the published law | This tree |
|---|---|
| `dir = ((facing + 0x10) >> 5) & 0xff`; `dstCell = srcCell + (i16)world[0x58ec0 + 4*dir]` | absent as a mechanism — a step's destination comes from the route search, never from a facing byte and a direction table |
| raw speed `actor->[0x70]->[0x3c]->[0x44]` when that byte is nonzero, else `(i16)actor->[0x8c]` | present — `moverSpeed`: the group term when nonzero, else the entity's own speed |
| `vt+0x20() != 1` so the raw speed is used as is | present — the fork on the ground domain |
| `d = clamp((i8)(height[src] - height[dst]), -32, +32)` | present |
| `v = SpeedMultiplier * speed`, `SpeedMultiplier` = `data/map.reg [Path Finding]` | **partial** — the multiply is present at the shipped default 8, written as a constant; this tree has no reader for that registry key, so a map that customises it is not read |
| `v += (v*d) >> 6`, arithmetic | present |
| `c = (u8)(cost[src] + cost[dst]) >> 1`, a byte add; `if c == 0 then c = 8` | present |
| `v /= c`, signed | present |
| `v = clamp(v, 1, 63)` | present |
| the per-axis steps scaled by the `0.707` double when `dx*dy != 0` | present, as the exact integer ratio 707/1000 |
| `mover[0xaa] = ceil(256 / max(...))` | present |
| **`cost(cell)` is `R1087` and is not a pure read** — on `block[cell] & 0x20` with a nonzero byte at `record+0xe` of the `world+0x540b8` table it shifts the stored cost byte right by 2 and **writes it back** | **absent** — `costAt` is an indexed read of the cost plane; nothing in this tree reads that block bit or that table, and no cell's stored cost is ever rewritten |

Two terms are therefore owed, and they are owed differently. The multiplier's *arithmetic* is here
and only its *source* is missing, so the number is right for every shipped map and wrong only for a
customised one. The cost accessor's write-back is missing outright, and it is the more serious of
the two: it makes the original's cost plane **stateful** — a cell that has been walked over can
answer a quarter of what it answered before — so on a map holding such cells our figure is not a
rounding away from the original's, it is a different quantity.

## Where the number lives

`pkg/sim/step.go` takes the rate once per transit, from the cell left and the cell taken, and holds
it until the next one. It reads the two planes at exactly one site, composes `rateOf`'s six
arguments there, and passes the result to `transitOf`. Nothing else in the tree computes a rate.

`rateOf` and `transitOf` are unexported, take no world, and the block that feeds them is inline in
the advance. So the number exists, it is reachable, and it is reachable **only** from inside
`pkg/sim`.

## The seam

`internal/archtest`'s allow-map gives `pkg/ui` the row `{"pkg/render", "pkg/render/"}` — the render
tier and nothing else. `pkg/ui` cannot name a simulation type, and the readout's own file says so.

Both inputs to the question live in `pkg/ui` and nowhere else: the cursor's cell is resolved at the
draw from `cursorX`/`cursorY` through the camera, and the selected unit is `presentSelected`'s
first element. `pkg/game` — the tier that may import both — knows neither.

That rules out the two obvious shapes. Recomputing the law beside the mover in `pkg/ui` is a second
copy of a movement law in the drawing tier, which is the failure `MapEntity.Speed` already refuses
in writing. Pushing a precomputed value per frame the way `SetReadout` pushes the clock's period
cannot work either, because the pusher does not hold the question's inputs. What is left is to push
the **question** rather than the answer: a function of builtins, installed once, called at the draw.

## What was read

`research/claims/terrain.md` (`TERR-MOVE-056`), `pkg/sim/rate.go`, `pkg/sim/step.go`'s advance and
its `Route` query, `pkg/sim/world.go`'s `costAt`/`heightAt`/`describes`/`moverSpeed`/`rated`,
`pkg/ui/readout.go`, `pkg/ui/overlay.go`'s `MapEntity`, `pkg/game/world.go`'s `pushReadout` and
constructor, and `internal/archtest/dag.go`.

## An observation the contract had to answer

`rateOf` is total: handed a speed of zero it returns `rateFloor`, and `transitOf` then returns 256.
The advance never asks it — `rated` gates the whole block — so that pair of numbers is a value the
movement code **never uses**. A query that did not carry the same gate would put it on screen.
