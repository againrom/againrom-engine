# Spec — the world's own clock: a rate from 1 to 1024, and a stop beside it

## Problem and current behaviour

The open map screen advances its world on a cadence of its own: a wall-clock baseline, an
accumulator, and a **period in whole milliseconds**, `1000/tps` by integer division. The
rate is an index into a **nine-row table**, 8 to 32 ticks a second, brought to the nearest
row rather than refused — so the lowest index is the **slowest speed, not a stop**. A map
opens at 16 a second: a 62 ms period, a water cycle in 992 ms. **Nothing suspends the
advance**, on this screen or any other.

That period is honest to about a hundred ticks a second and then collapses: 128 is paced
at 7 ms and runs at 143, 256 truncates 3.9 to 3 and runs at 333, and every rate above 500
lands on the same 1 ms. **One paced call runs at most four ticks** and drops the rest, so
the achievable rate is four times the front-end's own call rate — 240 a second where the
engine drives it at 60 — whatever period is asked for.

The **same clock paces two things at two independent instances** — the water counter and
the world, each from its own baseline, started at one number and kept together by nothing.
The standalone terrain viewer holds the first and no world at all, its rate set by a flag.

A tick is one integer step over a command slice — no duration, no clock read, one cell of
movement. An order is queued when the player issues it and applied by the next tick, which
empties the queue. A drawn entity's frame is selected at a clock the advance itself raises;
the water counter rises whether it advances or not, and turning animation off snaps every
water cell to its authored phase rather than holding it.

## Functional requirements

- **FR-1** The open map screen's world MUST be paced by a **rate**: a whole number of
  ticks a second, selectable over at least **1 to 1024**. For any rate `r` and any elapsed
  span of a second or more, the ticks that fire MUST be within **0.1%** of `r` a second. A
  value outside the range MUST be brought to the nearest inside it, never refused.
- **FR-2** Selecting one of the **game's own nine speeds** MUST give that speed's period
  as the game computes it — 16 a second at 62 ms, a water cycle in 992 ms — unchanged.
  FR-1's rate is a **second, independent selection** over the same clock, and whichever
  was selected last is in effect. A map opened and **given no cadence input at all** MUST
  run at the map-load speed's period, 62 000 µs. The **first** cadence input of any kind —
  including the stop, which selects no rate — MUST put the screen on the rate model at the
  map-load speed's own 16 a second, making the period 62 500 µs and the water cycle
  1 000 ms. Both are correct; the second is a **disclosed divergence** of 0.8 % from the
  game's truncated 62, ruled acceptable rather than designed around (`provenance.md`).
- **FR-3** The **stop** MUST be a switch beside the rate and MUST NOT be a value of it.
  While it is set **no tick MUST fire** however much time elapses, and the world's tick
  count, every entity field, the canonical byte form and its digest MUST be identical
  across any number of stopped frames. Clearing it MUST NOT fire a tick for the time it
  was set and MUST NOT change the rate: the first advance after it MUST run **one** tick,
  not a backlog.
- **FR-4** While the stop is set the camera MUST still pan, drag and zoom, a selection
  MUST still be made, cleared and outlined, and orders MUST still be issued — **every
  order issued while stopped MUST be applied by the first tick after it is cleared**, in
  issue order, none lost and none applied twice. **No drawn entity's frame MUST change**
  while it is set.
- **FR-5** With a map open the **water counter MUST advance at the rate FR-1 and FR-2
  select**, and the stop MUST NOT reach it: water animates while the world is stopped. The
  standalone terrain viewer MUST be unchanged — its own speed selection, its own counter,
  no stop, and no key of this story read anywhere in it.
- **FR-6** The ticks **one** paced call may run MUST be bounded by a fixed span of **world
  time** rather than by a fixed count, so the bound is the same span at every rate and
  never holds the achievable rate below what the front-end's own call rate allows. **At
  least one tick MUST be permitted per call at every rate in range**, and time past the
  bound MUST be dropped rather than kept for a later call.
- **FR-7** On the map screen one key MUST toggle the stop and two MUST double and halve
  the rate, clamped into range. All three MUST act on the **press**, so holding one acts
  once; each MUST take effect on the frame it is pressed, MUST move no camera and MUST
  change no selection; on every other screen all three MUST do nothing.
- **FR-8** The simulation package MUST NOT change: no new field, no change to the
  canonical byte form or its digest. The rate, the stop, the period, the accumulator and
  the wall-clock baseline MUST be driver and front-end state alone — never world state,
  never hashed, never serialized. For one command stream, `k` ticks MUST reach the same
  state and digest whatever rate, stop schedule or elapsed-time schedule paced them.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a clock driven by an instant passed in | rates 1, 2, 16, 17, 64, 128, 256, 257, 969 and 1024 each run a second or more of elapsed time; then -5 and 5000 | each fires within 0.1% of its rate; the two outside run at 1 and at 1024 |
| **AC-2** | unit | the game's nine speeds | each is selected; separately a map is opened and given no rate | each period is the one the game computes, index 4 at 62 ms with a 992 ms cycle; the opened map fires its first tick at that period |
| **AC-3** | unit | a world advanced to a known tick, then stopped | many frames and many seconds are driven; then the stop is cleared and one period driven | no tick fires and the tick count, byte form and digest hold throughout; then exactly one fires, at the rate selected before the stop |
| **AC-4** | unit | a stopped world, a selection, `k` orders issued while stopped | a drag, a wheel zoom and a box release run stopped; then the stop clears and one period is driven | the camera moves by its unstopped delta, the outline is built, the selection changes as it does unstopped, no entity's frame differs; then the first tick applies exactly those `k`, in issue order |
| **AC-5** | unit | a map open at one rate; separately a viewer built as the standalone binary builds one | the rate is doubled; a stopped span is driven; the standalone viewer is driven over that span | the counter's rise per driven second doubles; it keeps rising while stopped; the standalone viewer's counter, speed and summary are what they were |
| **AC-6** | unit | a world paced at 16, at 256 and at 1 tick a second | a ten-second stall is driven in one call, then one further period | each call runs the bound's worth of world time and no more, the same span at all three, and at least one tick at rate 1; the next runs one tick, so the dropped time is not owed |
| **AC-7** | unit | the map screen, then the menu and the picker screens | each of the three keys is pressed, then held for several frames | on the map screen the stop toggles and the rate doubles and halves once per press, clamping at both ends, on the frame of the press; the camera and the selection are unmoved; on the other screens nothing changes |
| **AC-8** | unit | one command stream over a hand-built world | it runs `k` ticks at 16 a second, at 256 a second, and through a stop-and-resume schedule | the digest at every tick index is the same in all three and equal to a headless run over that stream; the simulation's pinned fields, byte form, digest and both wall checks are unmoved |
| **AC-9** | manual | the game on a map with several units, against a lawful install | the stop is toggled, the camera panned and an order given while stopped, the stop cleared, then the rate raised and lowered | the world stops with units frozen mid-stride while the camera pans and the water moves; the order is obeyed at once on the resume; units walk visibly faster and slower |

**Error cases: None applicable.** A rate outside the range is brought inside it and a
toggle is total, so there is no cadence input to refuse; a machine too slow for the rate
selected is a **disclosed limitation** below, not a failure the contract detects.

## Derived properties

- **P-1 (invariant)** — Pacing decides **when** a tick fires and never **what** it does:
  for any elapsed-time, rate and stop schedule, the world after `k` ticks is the world
  `k` direct steps produce.
- **P-2 (invariant)** — No world field, byte form or digest differs on account of a rate,
  a period, an accumulator, a baseline or a stop.
- **P-3 (completeness)** — Every cadence state is exactly one of: stopped, or running at
  a rate in range. No rate means stopped, no stop means a rate, and a value outside the
  range names the nearest one inside it — there is no third state and none is undefined.
- **P-4 (negative-invariant)** — For any span spent stopped, no tick fires and no elapsed
  time is held for later: the digest at the resume equals the digest at the stop, and the
  first tick after it is one tick.
- **P-5 (invariant)** — The water counter's advance is a function of elapsed time and the
  rate alone; neither the stop nor the catch-up bound enters it.

## I/O examples

```
rate  r   -> period 1000000/r us      16 -> 62500   256 -> 3906   1024 -> 976
speed idx -> period (1000/tps) ms     idx 4 -> 62000 us, the game's own truncation
stopped   -> elapsed consumed, 0 ticks, digest unmoved, orders queued
one call  -> at most 250 ms of world time:  16/s -> 4 ticks   256/s -> 64   1/s -> 1
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The stop is a **switch beside the rate**. | **(A) rate 0** — the shipped clamp reads it as the slowest speed, so a stop written that way is a slow game, and one over zero is no period at all. **(B) an unreachably long period** — it fires eventually, and the resume must unwind what it accumulated. **(C, chosen) a switch**, the only one with nothing to unwind. |
| **C-2** | **One rate paces the world and the water.** | **(A) a second, independent water rate** — "speed" then names two things on one screen, a player cannot tell which a control moved, and the next consumer of a cadence must be told which of three clocks it takes. **(B, chosen) one rate, two readers**; the cost is disclosed below. |
| **C-3** | The rate's ceiling is **1024**. | **(A) unbounded** — the period falls to zero and the pacing has nothing to divide by. **(B) 256** — that is the request, not its ceiling, and it leaves no headroom above itself. **(C, chosen) 1024**, four times it, the period still within 0.1% throughout. |
| **C-4** | A rate scales **the tick**, never a unit's stride. | A speed multiplier on movement instead would make one tick mean different distances at different rates, so no digest could be compared across rates — the property that makes a rate safe to hand a player. |

## Out of scope

- **Any on-screen indication** of the rate or of the stop: no HUD, no text, no icon.
- **Persisting a rate** across maps or runs — a configuration key, a flag, a saved index.
- **Slow motion below one tick a second**, fractional rates, and a rate or a stop on any
  screen but the map.
- **Freezing water**, and any use of the shipped animation switch to approximate one.
- **Per-unit speed**: a rate changes how often the world steps, never how far anything
  moves in a step, and no unit becomes faster than another here.
- **A third cadence consumer** — object animation on a render clock — and any rate or
  stop for the standalone viewer beyond the speed selection it has.

**Disclosed limitations**, accepted and owned: at high rates the water cycle **strobes**
— at 256 ticks a second its four variants run sixteen times a second — and aliases
against the display's refresh, the price of C-2's one rate; the rate is a **target**, so
a machine that cannot run that many ticks in real time runs the world slower than asked
with nothing on screen to say so (FR-6 drops that time rather than owing it); a drawn
entity's frame is selected from the advance itself, so it freezes while stopped,
deliberately (FR-4); and because the rate decides which tick an asynchronous order lands
on, two runs at different rates are different **histories** of one world — a different
statement from FR-8's equality of state.

## Verification mapping

AC-1 … AC-8 are unit tests over hand-built worlds and clocks driven by an instant passed
in — no window, no install, no wall clock read — so all are CI-automatable. AC-9 needs a
lawful install and a window. P-1, P-2, P-3 and P-5 are **sampled, not proved**; P-4 is
witnessed at every stopped frame of the schedules AC-3 and AC-4 drive.

## Gate check

FR-1 → AC-1, P-3, C-3 · FR-2 → AC-2 · FR-3 → AC-3, P-4, C-1 · FR-4 → AC-4, P-4 ·
FR-5 → AC-5, P-5, C-2 · FR-6 → AC-6 · FR-7 → AC-7 · FR-8 → AC-8, P-1, P-2, C-4.
