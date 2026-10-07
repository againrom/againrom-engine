# Spec — the object layer's cycle, and the cell that opens it

## Problem and current behaviour

Every placed object draws one frame, chosen once. When a map opens, each cell holding an object
byte resolves to a class and the class to the single sheet frame its `Index` names; that frame, its
top-left and its ground-anchor pixel are fixed then, and nothing rebuilds them. A map's objects are
frozen for the life of the screen, in both front-ends.

The registry does carry a cycle. An object class may declare two equal-length arrays — frame values
and durations — both already decoded and resolved through the class inheritance chain, and nothing
reads them. A class also declares a phase count, set on classes carrying no arrays at all, so it
does not say whether a class has a cycle.

The map screen already has a counter that rises with the world's cadence. Water reads it per cell
to pick its strip variant; the rate control re-rates it; the world's stop does not reach it, so
water animates while the world is stopped, and turning animation off holds it where it is and draws
every water cell at its authored phase. The object layer sees none of this — not the counter, not
the rate, not the switch.

Sprites are already lit — one ramp row per rendered frame over each frame's own palette, alike in
both front-ends — and the cull is the exact world rectangle of whichever frame a placement holds.

## Functional requirements

- **FR-1 — a class's cycle is a timeline it does not store.** An object class's two arrays MUST be
  expanded into one **timeline**: the i-th frame value repeated as many times as the i-th duration
  says, in array order, a non-positive duration contributing nothing while still consuming
  its round, and the walk ending when either array runs out. The timeline's length is the cycle's
  **period**; a class whose expansion is empty has **no cycle**, whatever its phase count says.
  Every frame a timeline can name MUST be drawable, so a class that draws at all MUST carry **every
  frame of its sheet**.
- **FR-2 — the drawn frame, three arms.** A placed object MUST draw exactly one frame, by the first
  arm that holds: with animation **off**, sheet frame **0**, whatever its `Index` says and whether or not
  it has a cycle; else with its cycle **open**, `Index + timeline[step]`; else
  `Index`. A resolved frame outside the class's own sheet MUST draw sheet frame **0** — never
  nothing, never a refusal, never a panic.
- **FR-3 — the step is a pure function of the counter and the cell.** `step` MUST be
  `counter + col*(row+1)` reduced modulo the period into `[0, period)`, for any counter and any
  cell. No placement MUST hold a phase, a clock or any per-cell animation state; no randomness or
  wall-clock read MUST enter it; the arithmetic MUST be integer throughout. Two cells of one
  class MUST select different steps at some counter, and one cell MUST select the same step for the
  same counter however often it is asked.
- **FR-4 — the cycle's gate is a property of the cell, not of the class alone.** A cell's cycle is
  open only when **both** hold: the class's period is non-zero, **and** the cell's own tile word
  ORed with its three neighbours to the east, south and south-east has **both of its top two bits**
  set. A cell on the last column or the last row has no such neighbours and its cycle MUST be
  closed. A **diagnostic** MUST be able to open the cycle of every class with a non-zero period
  regardless of any tile word; it MUST default to off and MUST reach the object layer alone.
- **FR-5 — one counter, already ticking.** Object animation MUST advance on the same counter the
  water phase reads, at that counter's own value **unreduced**. No second clock, second rate, second
  stop or counter of its own MUST be introduced. Re-rating that counter MUST move objects and water
  together; the world's stop MUST move neither, so objects cycle while the world is stopped;
  switching animation off MUST hold the counter, and switching it on MUST resume from
  the held value.
- **FR-6 — animation moves a frame, never a ground point.** For any counter, the world point a
  placement's ground meets MUST be the one its build produced. The frame's top-left MUST be
  re-derived from the **drawn** frame's own size, so a cycle whose frames differ in size stands each
  of them on that same ground point; the rectangle culled and drawn MUST be the drawn frame's own.
  Draw order MUST stay the placement order the build produced.
- **FR-7 — one rule for the pixels.** A cycled frame MUST reach the destination through the blit,
  the ramp and the per-rendered-frame row a static one does, and the unshaded diagnostic MUST cover
  it. For one map, one counter and one gate the two front-ends MUST resolve every placement to the
  same frame and paint it with the same pixels.
- **FR-8 — both front-ends, and what they report.** The map screen MUST cycle its objects as its
  counter advances, with no switch of its own beyond FR-4's diagnostic. The raster tool MUST render
  its objects at a counter value the caller selects, defaulting to the value a map opens at, so a
  still may be taken at any point of a cycle. Both MUST report, beside the placement count they
  already print, how many of those placements have an **open cycle**.
- **FR-9 — nothing reaches the simulation.** A timeline, a step, a counter, a gate and the animation
  switch MUST be renderer and front-end state alone — never world state, never hashed, never
  serialized, and no field, canonical byte form or digest MUST change. For one command stream, `k`
  ticks MUST reach the same state and digest whatever the renderer ran under.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | the pairs `([4]*7, [0..6])`, `([3,0,2],[5,6,7])`, `([2,2],[9])`, `([],[1])` | each expanded | 28 steps `0000 1111 … 6666`; `5,5,5,7,7`; `9,9`; empty — a non-positive duration drops its frame but not its round, and the shorter array ends the walk |
| **AC-2** | unit | a period of 28 and the cells `(0,0)`, `(1,0)`, `(0,1)`, `(3,5)` | the step taken at counters 0…31 | each equals FR-3's expression computed independently in the test; `(1,0)` and `(0,1)` differ at every counter, so a transposed stagger cannot pass; two evaluations of one input agree |
| **AC-3** | unit | interior cells whose four-word OR sets both top bits, one, and neither; the both-bits case at the last column and the last row; a class of period 0 | the gate asked, diagnostic off and on | off: open only for the interior both-bits cell of non-zero period. On: open at every non-zero period, still closed at 0 |
| **AC-4** | unit | `Index` 3, a 28-step timeline over values 0…6, a 12-frame sheet; and a timeline naming value 40 | the frame drawn with the switch on and the cycle open, with it closed, and with the switch off | `3 + timeline[step]`; `3`; `0`; and the out-of-range selection draws `0` without panicking |
| **AC-5** | unit | one animated placement whose cycle reaches two frames of different sizes | drawn at two counters selecting each | one ground point at both; the top-lefts differ by exactly the halved size difference; each rectangle is its own drawn frame's |
| **AC-6** | unit | a synthetic archive: two classes naming one sheet, a palette-less sheet, an `Index` outside its sheet | loaded | the two share one frame collection pointer for pointer; the other two stay the artless answers they are today; every drawing class carries its whole sheet |
| **AC-7** | unit | one map, one counter, one gate | rendered by the raster path and by the window's texture path | every placement resolves to the same frame in both and the pixels agree — at the row every other sprite of that frame took, and raw under the unshaded diagnostic |
| **AC-8** | unit | a map screen at one rate | the rate doubled; a long stopped span driven; animation switched off, driven, switched on | objects and water read one counter throughout; it rises while stopped; with animation off it holds and every object draws frame 0; switching on resumes from the held value |
| **AC-9** | unit | a nil bundle; absent tile words; a class of period 0; placements at `(0,0)` and at the far corner; the counter at its maximum | placements built and drawn in each | nothing panics, no index leaves its slice, no division by zero occurs, and the bundle-less build draws what it draws today |
| **AC-10** | manual | a lawful install, a shipped map, both front-ends | the map opened; the raster tool run at several counter values, with the diagnostic and without it, and with animation off | without it the open-cycle count is 0 and the art is identical at every counter; with it, classes of non-zero period cycle; with animation off every object collapses to frame 0 |

**Error cases:** AC-9. Nothing here rejects input that is accepted today: an unusable timeline, an
absent tile grid and an out-of-range frame each resolve to a drawn frame rather than a failure, so
no caller gains an error path.

## Derived properties

- **P-1 (invariant)** — The drawn frame is a pure function of the class, the cell, the counter, the
  animation switch and the gate; nothing else can move it, and two evaluations of one input agree.
- **P-2 (negative-invariant)** — For any counter, gate, timeline or animation switch, no world
  field, canonical byte form or digest differs, and the simulation gains no error path.
- **P-3 (completeness)** — Every frame drawn is one of FR-2's three arms: a period of 0, a closed
  gate and an out-of-range value each land on a named arm, and there is no fourth.
- **P-4 (invariant)** — For any counter, a placement's ground point equals the one its build
  produced.
- **P-5 (invariant)** — With the diagnostic off, over a map no cell of which opens FR-4's gate,
  every placement's frame at every counter is the frame drawn before this story.

## I/O examples

```
AnimationTime [4 4 4 4 4 4 4] + AnimationFrame [0 1 2 3 4 5 6]
  -> timeline 0000 1111 2222 3333 4444 5555 6666    period 28

cell (10,6) counter 100 -> (100 + 10*7) mod 28 = 2 -> frame Index+0
cell (11,6) counter 100 -> (100 + 11*7) mod 28 = 9 -> frame Index+2

-noanimation   every object draws sheet frame 0; the counter holds
-objectanim    every class of non-zero period cycles, whatever the tile words hold
-tick N        the raster tool renders at counter N (default 0, a map's opening value)
```

`-objectanim` and `-tick` are new, defaulting to off and 0; the other two already exist and gain no
new spelling.

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The cycle is gated **per cell** by the four-word tile test, with an off-by-default diagnostic beside it. | **(A) a non-zero period alone** — the layer then has no per-cell condition and no map can be drawn with its objects still. **(B) no cycle at all** — the timeline stays unreachable. **(C, chosen)** the cell test, so the map decides; the diagnostic makes the arm visible where no cell opens it. |
| **C-2** | **One counter** — the one water reads. | **(A) a counter of its own** — a third cadence on one screen, re-rated, stopped and switched in step with two others. **(B) the clock a stop halts** — objects would freeze while the water beside them moved. **(C, chosen)** the existing counter. |
| **C-3** | An unusable frame draws **sheet frame 0**. | **(A) the class's `Index`** — a fourth outcome to state, test and keep equal to three others. **(B) nothing** — an object disappears for part of its cycle, the one thing a total renderer must not do. **(C, chosen)** frame 0, which FR-2's switch-off arm already draws. |

**Disclosed limitation**, accepted and owned: where no cell opens a gate (P-5), the reported
open-cycle count and the diagnostic are the only witnesses that a cycle exists at all.

## Out of scope

- **The dead form a cell can select**, and the object-state swap with it: the key naming it resolves
  to a valid class id on a third of the classes when absent, so nothing can be substituted until
  that resolution is settled.
- **Fire variants and any burning state.**
- **Structure animation**, and any claim that a structure's cycle is this one.
- **Any writer of the tile-word bits the gate reads.** No tile word is modified at run time here.
- **Object shadows, overlay sheets, per-cell light, sound, wind or direction coupling.**
- **A rate or a stop for the standalone terrain viewer**, and any change to the water phase, the
  counter's cadence or the world's stop policy.

## Verification mapping

AC-1 … AC-9 are unit tests over classes, grids, sheets and counters built in test code — no window,
no clock, no install — so all nine are CI-automatable; AC-10 needs a lawful install. P-1, P-2 and
P-4 are **sampled, not proved**; P-3 is structural, witnessed by AC-4 and AC-9 together; P-5 by
every counter AC-4's closed-gate case drives.

## Gate check

FR-1 → AC-1, AC-6 · FR-2 → AC-4, AC-8, P-3, C-3 · FR-3 → AC-2, P-1 ·
FR-4 → AC-3, AC-10, P-5, C-1 · FR-5 → AC-8, AC-10, C-2 · FR-6 → AC-5, P-4 · FR-7 → AC-7 ·
FR-8 → AC-7, AC-10 · FR-9 → AC-9, P-2.
