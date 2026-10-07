# Analysis — the world's own clock

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-first / static** — the profile names engine work at this tier; no watcher tool exists, so it is discipline |
| Terrain — the rate model, the stop, the cadence seam and the keys that select it | **greenfield**: none of it exists in any form |
| Terrain — the tick accumulator, the paced advance, its catch-up bound, the front-end's input snapshot and its map arm | **brownfield**: every one is shipped and running today, and each is touched here |

## There is no baseline to reconcile against

Every story so far imported a clean-room spec and spent its first stage reconciling
that document with the tree. This one has none. The contract is authored from the
owner's scope, so nothing here is a disagreement with a document: it is what the tree
measures, and what those measurements foreclose.

## The foreclosures, measured against this tree

| Measured | What it forecloses |
|---|---|
| The period is whole milliseconds, `1000/tps` by integer division | 128 ticks a second is paced at 7 ms and runs at 143; 256 truncates 3.9 to 3 and runs at 333; every rate above 500 lands on 1 ms; above 1000 the quotient is 0 and the accumulator's division has nothing to divide by. The accumulator already carries the sub-tick remainder, so what has to change is the unit the period is held in and nothing else |
| `clampSpeedIndex` brings an index below the table to its slowest row | Index 0 is 8 ticks a second, not a stop, and nothing anywhere suppresses an advance. A stop has to be a switch beside the rate |
| The shipped animation switch resolves the tile word itself when off | It does not hold the counter, so turning it off snaps every water cell to its authored phase. It is not a freeze and cannot be borrowed as one |
| One paced call runs at most four ticks and drops the rest | A fixed **count**, so the world's achievable rate is four times the front-end's own call rate — 240 a second at the 60 ticks the engine defaults to — whatever period is asked for. This one was not on the list, and it is why the rate the owner asked for is unreachable while the bound stands as written |

## Two clocks, two instances, and only one of them under a world

`Viewer.anim` and `mapWorld.clock` are separate `terrain.Ticker` values, both built at
the map-load speed index, each raised from its own wall-clock baseline: the viewer's
inside `Viewer.step`, the world's inside `paceTo`. They agree today only because they
were given the same starting number, and nothing keeps them agreeing. The standalone
viewer holds the first and no world at all, which is what makes the coupling a
decision rather than a lookup — a shared rate has to be written to two places, and
whichever way it is decided, the tree will not decide it by itself.

## Whether the rate reaches hashed state — what was checked, not assumed

`sim.Step(w, cmds)` takes a command slice and no duration; `pkg/sim` imports no clock,
and `internal/archtest`'s source scan over that package fails on an import of `time`.
The world's canonical bytes are the tick, the bounds, the generator and one record per
entity — position, target, class, stall — and no cadence quantity appears among them.
The period, the accumulator, the wall-clock baseline and the catch-up bound are fields
of `mapWorld` and `Viewer`, born with the map screen and dropped with it. So a rate
decides how often `Step` is called and a stop decides whether it is called at all, and
neither is an argument to it.

One consequence is worth writing down rather than leaving to be discovered: the rate
does move which tick an asynchronous order lands on, because an order is queued when
the player issues it and applied by the next tick that fires. That makes two runs at
different rates different *histories*; it does not make the state a function of the
rate, which is what the threshold asks about.

## What freezes when the world does not advance, measured

The scene clock every entity's drawn frame is selected at is raised inside the advance
itself, so with no advance no frame changes. The water counter is raised by the
viewer's own step, which runs whether or not an advance fires. The order queue is
appended to by the statement that issues an order and emptied by the tick that applies
it, so orders issued while nothing advances accumulate and are applied, in issue order,
by the first tick that does. All three fall out of where the shipped statements already
stand; none needed a decision, and the contract states them because a later story
reading only the code would have to measure them again.

## What a microsecond period costs, and what a millisecond one costs

Over every rate from 1 to 1024 the truncated microsecond period is at worst 0.0962%
fast (at 969), 0.0576% at the ceiling and 0.0064% at 256. The millisecond period is
11.6% fast at 128 and 30.2% at 256. Nanoseconds were weighed and buy nothing here: a
microsecond is already three orders of magnitude below the shortest period in range.

## We searched the pinned claims for a rate model and for a pause

Read at the submodule as checked out here, pin `a13b3b8`, with `claims/retracted.md`
and the registry's standing corrections first; neither bears on terrain animation. The
ledgers carry the original's own cadence in detail — the paced loop, the period's
arithmetic, the nine-row speed table, its map-load default, its keys and its config
path — and **no pause of any kind**, on any screen, in any ledger. What follows from
that for what we are entitled to assert is the ledger's business, not this file's.

## What we looked at

`pkg/render/terrain/water.go`, `pkg/game/world.go`, `pkg/game/frontend.go`,
`pkg/ui/{viewer,app,flow,command}.go`, `pkg/sim/{step,world,binary,hash}.go`,
`internal/archtest/determinism_test.go`, `cmd/mapview/main.go`,
`pkg/game/world_test.go`, `pkg/render/terrain/water_test.go`, `pkg/ui/input_test.go`,
`docs/0030-box-select/`, `docs/0026-unit-collision/provenance.md`, `AGENTS.md`, the
three check scripts, and in research `claims/retracted.md` and `claims/registry.md`
first, then `claims/terrain.md` and `claims/move.md`.
