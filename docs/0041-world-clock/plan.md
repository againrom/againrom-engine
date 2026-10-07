# Plan — one period in microseconds, one rate, and a switch only the world reads

## Baseline

`terrain.Ticker` holds `speedIndex`, `dtMs`, `acc` and `count`. `SetSpeedIndex` clamps
into `0..8` and recomputes `dtMs = TickMillis(idx) = 1000/TicksPerSecond(idx)`;
`Advance(elapsedMs)` adds to `acc`, takes the **whole quotient at once**, subtracts what
it consumed and returns the count, ignoring a non-positive elapsed. `TicksPerSecond`,
`TickMillis` and `WaterCycleTicks` are free functions and constants over `speedTPS`, nine
rows, `8..32`; `DefaultSpeedIndex` is 4.

Two instances exist and nothing relates them. `Viewer.anim` is raised in
`advanceAnimation` from the instant `Viewer.step` is given, which **writes the baseline
before the `animate` test**, so a disabled stretch is skipped rather than accumulated;
`Viewer.SetSpeedIndex` re-rates it and `Animation()` reports the index, which
`cmd/mapview` prints in its summary line. `mapWorld.clock` is raised in `paceTo`, which
truncates the elapsed to whole milliseconds, **leaves the sub-millisecond tail on `last`**,
caps the tick count at `maxCatchUpTicks = 4` and drops the surplus. `paceTo` returns
`(ticks, applied)` and `paced` — the `ui.MapTick` the loader hands the front-end —
discards both.

`mapWorld.tick` assembles commands, truncates `pending` **between assembly and `Step`**,
calls `sim.Step`, raises `scene` and pushes; `scene + int(id)` is the tick every entity's
frame is selected at. `enqueue` appends one command and marks its entity commanded in one
statement and steps nothing. `App.step`'s `ScreenMap` arm calls `flow.tick` when non-nil,
then `Viewer.step`, then `command`, and issues each order through `flow.order`; `flow`
holds `tick` and `order`, set by `choose` and dropped by `escape` in one statement each.
`readAppInput` samples `appInput` with `inpututil.IsKeyJustPressed`; `MapLoader` returns
`(*Viewer, MapTick, MapOrder, error)` and `FrontEnd.loadMap` is its one production
implementation. `sim.Step(w, cmds)` takes no duration; `pkg/sim` imports no clock and
`internal/archtest`'s source scan fails on one; the canonical bytes carry the tick, the
bounds, the generator and one record per entity.

## Design decisions

### DD-1 — the period is microseconds, and every entry point is renamed with it

`dtMs` becomes `periodUS`; the `TickMillis()` method becomes `Period()`; `Advance` becomes
`AdvanceMicros`. `paceTo` and `advanceAnimation` divide by `time.Microsecond` and leave
the sub-microsecond tail where they left the sub-millisecond one. The renames are the
decision, not a tidy-up: the unit of an `int` parameter changing by a thousand is
invisible to the compiler, and there are two production call sites and a suite of literals
that would all keep compiling and silently mean something else.

Rejected: **nanoseconds** — a microsecond is already three orders of magnitude below the
shortest period in range and 0.1% at the ceiling, and nothing here measures below a frame.
Rejected: **a float period** — the accumulator's exactness is what the whole cadence rests
on (FR-1, FR-2).

### DD-2 — a rate and a decoded index are two ways to compute a period, and neither is the other

`RatePeriod(rate) = 1000000/clampRate(rate)` with `clampRate` into `1..1024`, total rather
than rejecting, as the shipped index clamp is; `SpeedIndexPeriod(idx) = TickMillis(idx) *
1000`, the game's truncated millisecond **widened rather than recomputed**. The ticker
holds the period alone and stops holding an index: two nominal identities on one struct is
how a "rate" and a "speed" come to disagree about what the clock is doing. `Viewer` keeps
the index it was given for its own report, written by `SetSpeedIndex` and by nothing else,
so the standalone summary keeps answering the question it answers today.

Rejected: **folding the nine into the rate model** — index 4 would move from 62 ms to 62.5
and the water cycle from 992 ms to 1000, changing numbers this project holds because the
game computes them that way (FR-1, FR-2).

### DD-3 — the stop is a bool on `mapWorld`, read by `paceTo` first

Stopped, `paceTo` **takes the baseline and returns zero**: the elapsed span is consumed by
that write, so nothing is owed when the stop clears and the resume cannot burst. It is not
a field of the ticker, because the water instance would then carry a switch it must never
read, and `cmd/mapview` would gain one it cannot honour.

Rejected: **the front-end simply not calling the tick** — the baseline would go untaken
while stopped, so the first call after the resume would carry the whole paused span and be
cut to the catch-up bound rather than to nothing, making the burst's size a function of how
long the pause was (FR-3, FR-8).

### DD-4 — the catch-up bound is a span of world time, converted at the period in force

`maxCatchUpMicros = 250000` replaces `maxCatchUpTicks = 4`, and one call's cap is
`max(1, maxCatchUpMicros/period)`: 4 at the map-load period — the shipped number
re-derived rather than moved — 64 at 256 ticks a second, and **1 at rate 1**, where the
bound is shorter than a single tick and a cap of zero would be a stop nobody asked for.

Rejected: **the fixed four** — it is a rate ceiling wearing a stall bound's name. Four
ticks a call at a 60-per-second front-end is 240 ticks a second whatever period is
selected, so the rate this story exists to reach is unreachable while it stands (FR-6).

### DD-5 — one rate is written to both consumers by one statement; the stop to one

`flow` holds the cadence — a rate and a stopped flag — and one method writes it: the
viewer's rate through a new `Viewer.SetRate`, and the seam of DD-6 with both scalars, in
one statement. Water therefore takes the rate and not the stop, which is the coupling
decision made structural rather than conventional: there is one rate value and two readers
of it, and no path that re-rates one reader without the other.

Rejected: **one shared `*Ticker`** — the world's tick stream is gated by the stop and cut
by the catch-up bound and water's is neither, so a shared instance forces water to inherit
both: a freeze on every pause and a stutter on every stall. Rejected: **two independent
rates**, which the contract rules out (FR-4, FR-5, FR-7).

### DD-6 — the cadence crosses the seam as a third loader-supplied function

`MapCadence func(rate int, stopped bool)` joins `MapTick` and `MapOrder` in `MapLoader`'s
result and in `flow`, set and dropped in the statements that already set and drop those
two, so no transition can leave one held without the others. Two scalars name no
simulation type, which is that seam's own rule. It is called **when the front-end's
cadence changes and not per tick**, and both sides are born at the map-load speed, so an
opened map needs no call to agree.

Rejected: **widening `MapTick`** — the once-per-tick call takes nothing and returns
nothing on purpose (0020 DD-1), and 0028 rejected the same widening for orders. Rejected:
**`pkg/game` reading the rate off the viewer it already holds** — the standalone viewer
would carry a stop it can neither honour nor be asked about, and the world's cadence state
would live where there is no world (FR-5, FR-7).

### DD-7 — three press edges on the front-end's own snapshot, resolved before the advance

`appInput` gains `Pause`, `Faster` and `Slower`, sampled with the just-pressed sampler its
other keys use, from `KeySpace`, `KeyEqual`/`KeyNumpadAdd` and `KeyMinus`/`KeyNumpadSubtract`.
The map arm resolves them **before** its tick call, so a press takes effect on its own
frame; every other arm ignores them; and `Input` — the viewer's own snapshot, which
`cmd/mapview` fills — gains nothing, so the standalone viewer cannot read one.

The ladder doubles and halves. Rejected: **stepping by one**, which is 240 presses from
the map-load speed to the rate the owner asked for and 1008 to the ceiling. Rejected:
**reading a level rather than an edge** — a held key would toggle the stop every frame
(FR-7).

### DD-8 — the pause needs no queue, no drain and no freeze of its own

Every clause of FR-4 is already the shape of shipped statements: an order is queued by the
statement that issues it and drained by the tick that applies it, so orders issued while
stopped are applied in issue order by the first tick that fires; a frame is selected at a
clock the advance raises, so frames freeze; the water counter is raised by the viewer's
step, which the stop does not reach, so water does not. **Nothing is written for any of
them** — what this story adds is the criteria that measure them, and the contract clause
that stops a later story from changing one by accident.

Rejected: **a drain of its own on the resume** — a second path into `Step` beside the one
the tick already performs. Rejected: **holding the water counter while stopped**, which is
a new control on the render tier bought for a consistency the contract does not promise
(FR-4).

## Risks

- **R-1** The ladder reaches rates no machine runs in real time. Past that point the world
  runs slower than the number selected and the only signal is that it looks wrong — the
  contract discloses it, and nothing in this story detects it.
- **R-2** One rate makes water strobe at the top of the ladder. The owner will meet it on
  the first run at 256, and it is the decided cost of the coupling rather than a defect to
  be found later.
- **R-3** The loader's result grows a fourth member, so every loader in the suite is
  edited. A mechanical change spread over many files is where an unrelated behaviour
  change hides.

## Success criteria

- **SC-1** AC-1 holds in full: ticks counted over a driven elapsed schedule at all ten
  rates and both out-of-range values, each within 0.1%; the period asserted against the
  exact quotient at 1, 256 and 1024 (FR-1).
- **SC-2** AC-2 holds in full, and the existing pins on the nine periods, on 62 ms and on
  the 992 ms cycle are **unedited** — a story that had to edit them would have moved the
  decoded numbers rather than widened the clock under them (FR-2).
- **SC-3** AC-3 and P-4 hold: no tick over a stopped schedule of many frames and many
  seconds, byte form and digest identical throughout, exactly one tick after the resume,
  and the rate unchanged across the toggle (FR-3).
- **SC-4** AC-4 holds in full: stopped, a pan moves the camera by its unstopped delta, the
  outline pass is built, a box release replaces the selection, and every entity's frame is
  the one it had; after the resume the first tick applies exactly the orders issued while
  stopped, in issue order (FR-4).
- **SC-5** AC-5 and P-5 hold: the water counter's rise per driven second measured at two
  rates and the ratio asserted, measured again over a stopped span and asserted to rise,
  and a viewer built as `cmd/mapview` builds one asserted unchanged in counter, speed and
  summary (FR-5).
- **SC-6** AC-6 holds in full, the bound read as a **span of world time** at 16, 256 and 1
  ticks a second rather than as a count, the surplus asserted dropped rather than queued,
  and the one-tick floor witnessed at rate 1 (FR-6).
- **SC-7** AC-7 holds in full: each key acts once per press and not while held, the ladder
  clamps at both ends, the camera and the selection are unmoved, and the menu and picker
  arms are asserted to change nothing (FR-7).
- **SC-8** AC-8, P-1 and P-2 hold: one command stream advanced at two rates and through a
  stop-and-resume schedule, digests compared at every tick index against a headless run
  **assembled from that command stream** and never from a second run of the driver under
  test; the simulation's pinned fields, byte form, digest and both wall checks read unmoved
  (FR-8).
- **SC-9** Four mutants, each applied to production code, run over the whole tree with its
  failing tests named, and reverted: the period returned to the millisecond quotient; the
  catch-up cap returned to a fixed four; the stop's baseline write dropped, so a resume
  unwinds the paused span; the viewer's half of the coupling write removed (FR-1, FR-3,
  FR-5, FR-6).
- **SC-10** AC-9 is run against a lawful install on a map with several units (FR-1, FR-3,
  FR-4, FR-7).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1, C-3 | DD-1, DD-2 | SC-1, SC-9, SC-10 |
| FR-2 | DD-1, DD-2 | SC-2 |
| FR-3, C-1 | DD-3 | SC-3, SC-9, SC-10 |
| FR-4 | DD-5, DD-8 | SC-4, SC-10 |
| FR-5, C-2 | DD-5, DD-6 | SC-5, SC-9 |
| FR-6 | DD-4 | SC-6, SC-9 |
| FR-7 | DD-5, DD-6, DD-7 | SC-7, SC-10 |
| FR-8, C-4 | DD-3, DD-8 | SC-8 |
