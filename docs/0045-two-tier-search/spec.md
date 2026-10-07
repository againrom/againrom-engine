# Spec — two searches: one per order over the whole map, one per step inside a window

## Problem and current behaviour

Two searches produce a route. The **far** one reads terrain alone, sweeps the whole map, runs once
per order, and its route is stored, hashed and consumed up to four cells at a time; the **near**
one reads occupancy too, runs every tick inside a window of 8 on the mover, and advances it one
cell. Forty units ordered across 256x256 cost **6.08** ms a tick over the order's window and
**0.54** over 256 ticks.

The far search stops when the target is labelled, when the frontier empties, or after
`max(5, D>>2) + D` generations over the Chebyshev distance `D` to the target. A generation
advances the wave one ring, so the whole allowance for a detour over the straight line is
`max(5, D>>2)` rings — and a land route that goes up, across and down needs about twice the
crossing leg in rings the straight line does not use. The wave never labels the goal, the search
reports no route, and the order is cleared in that tick: **a unit ordered to the far bank of a
channel does not move at all, where the same trip ordered as three legs walks.** Optimised mode
has no generation budget, so the two modes disagree today about what is reachable.

## Functional requirements

- **FR-1 — two searches, and what each may see.** A route MUST be produced by one of two
  searches. The **far** search reads **terrain alone**: a cell is open to it iff it is in bounds
  and its blocks-ground bit is clear, whatever units stand where. The **near** search reads
  terrain **and** occupancy as it stands when the unit is resolved — today's relation. Each MUST
  take its route by the world's routing mode, which MUST still govern route choice and nothing
  else.
- **FR-2 — the far search is unbounded in space and runs per order.** It MUST search the whole
  map: no rectangle, no region, no window, no spatial bound at all, in **either** mode, so a goal
  reachable only by a detour leaving the rectangle optimised mode used to grow around start and
  target MUST be routed to. Its termination is FR-10's. It MUST run where FR-6 requires it and on
  no other tick.
- **FR-3 — a stored route is canonical state.** Each unit's route MUST be carried by the
  canonical byte form and MUST enter the digest: two worlds differing in one stored cell, in a
  route's length, or in whether a unit holds one at all MUST have different forms and MUST NOT
  hash equal. The version MUST be raised and the previous one refused rather than migrated. A
  form MUST be refused, leaving the receiving world unchanged, when a route is stored for a unit
  with no target, when its last cell is not that unit's target, when a cell of it is out of
  bounds or blocks ground, or when two consecutive cells are not neighbours. A unit with a target
  and **no** route MUST be accepted: that is the state before a first routing.
- **FR-4 — the near search runs every tick, inside a window, and moves one cell.** Each tick
  every unit holding a target and the route FR-6 leaves it MUST have a route searched from its
  cell to FR-5's sub-goal and MUST advance to the **first cell of it and no further**. It MUST
  NOT label, expand or otherwise consider a cell outside its **window** — the cells within
  Chebyshev distance **8** of the unit's own cell, clipped to bounds — so it labels at most
  **289** however far the sub-goal lies. Its budget MUST use a slack of **3** over the
  `max(slack, D>>2) + D` form. Its route MUST NOT be stored, reach the byte form or outlive the
  tick. A unit beginning a tick on its target MUST neither search nor step, and MUST have target
  and route cleared.
- **FR-5 — the sub-goal, and how a stored route is consumed.** The sub-goal MUST be the route's
  cell at index `min(3, len-1)` — up to four cells along it, and the target itself once four or
  fewer remain. After a unit advances, if its new cell is one of the first **four** cells of its
  route then that cell and every cell before it MUST be dropped, and nothing else MUST be: a unit
  that stepped elsewhere keeps its whole route and makes its way back to it.
- **FR-6 — when a stored route does not serve.** A unit's route MUST be replaced by a fresh far
  search, in the same tick, exactly when it holds none, when its last cell is not the unit's
  current target, or when its sub-goal lies outside the unit's window. Otherwise the stored route
  MUST be used unchanged — in particular a unit that failed to move MUST NOT run a far search on
  the tick after.
- **FR-7 — failing, holding and giving up.** A unit whose **far** search finds no route MUST have
  its target cleared in that same tick, with no residue, no stall and no error. A unit whose
  **near** search finds none MUST keep its cell, its target and its route unchanged and MUST have
  its **stall count** raised by one; a unit that advances MUST have it reset to zero; at **16**
  target and route MUST both be cleared with no residue. A unit holding no target MUST hold no
  route and a zero count.
- **FR-8 — what a tick may cost.** In a run where each unit's target is set once, the far
  searches MUST number at most one per unit ordered plus one per departure under FR-6, and every
  other unit-tick MUST cost at most one near search. Against the committed harness, in
  milliseconds per tick: 40 units across 256x256 at or under **25** over its 16-tick window and
  **4** over a 256-tick one; the same group on 128x128 at or under **10**; 80 movers onto a
  sealed goal and 80 onto a held cell at or under **12** each.
- **FR-9 — determinism, order and the wall.** Units MUST be resolved in ascending id, each seeing
  the moves already resolved that tick. A world marshalled at any tick and decoded into another
  MUST advance identically to it at every later tick — same cells, same routes, same digests —
  under any command sequence. Advancement MUST stay integer-only and free of clock, file and the
  world's generator.
- **FR-10 — the far search's budget is flat, and what that makes reachable.** In canonical mode a
  far search's budget MUST be a flat **1000** generations whatever the distance to its target,
  counted exactly as a budget is counted today, where the near search MUST keep the
  `max(3, D>>2) + D` form; optimised mode counts no generations and MUST NOT begin to. The wave's
  two other stop tests MUST NOT move, so a search that labelled its goal under the **old** budget
  MUST stop in the same generation and return the same route **wherever that old budget was the
  smaller of the two** — every target within Chebyshev distance **800**, past which
  `max(5, D>>2) + D` exceeds 1000 and this budget is the tighter one. A goal reachable only by a
  detour longer than the old allowance MUST therefore be routed to in **both** modes rather than
  in optimised alone: on one order, a mover MUST cross to a cell whose land route leaves and
  re-enters the straight line by any detour those 1000 generations reach. **What 1000 generations
  do not reach stays refused in canonical where optimised still finds it**, and that residual MUST
  be witnessed rather than left for a reader to infer. Nothing else about either search MUST move
  — not the relation, the window, the cost model, or the refusal, before any sweep, of a goal the
  mover may not enter.
- **FR-11 — what a far search may now cost.** By FR-10 a search that succeeded under the old
  budget labels exactly what it labelled, so **no order served today MUST pay anything for the
  change itself**. The cost falls on the orders the old budget refused, and it splits: one the
  terrain still cannot serve is searched for once and cleared in that tick, where one the flat
  budget now reaches becomes an ordinary mover costing what FR-8 already prices. What bounds a
  refused search's labelled cells becomes the map rather than `max(5, D>>2) + D`, **however near
  its goal lies**, and that count MUST be reported at a near and at a far sealed goal rather than
  assumed — it bounds distinct cells and **not work**, so the milliseconds are what answer for
  work. FR-8's ceilings MUST hold unchanged, and a fifth shape — 40 units on 256x256 ordered to a
  **near** sealed cell, where the two budgets differ most — MUST come in at or under **25** ms a
  tick, its window's mean as FR-8's figures are, and its **worst single tick MUST be reported
  beside that mean**: a hitch is what an owner sees and a mean cannot show one.
- **FR-12 — this revision moves no byte of the form.** Beyond the version FR-3 raised, no field,
  section, width or version byte MUST change, a form MUST decode exactly as it decodes before this
  revision, and every byte form and digest pinned in the tree MUST be unmoved. What changes is
  what a world *contains* — a unit whose far search found nothing now holds a route and walks — so
  **one** world advanced by the same commands on a build before this revision and on a build after
  it MUST be expected to differ in cell, route and digest exactly where a far search used to fail,
  and to agree everywhere else. Two worlds on one build still answer to FR-9 and P-5. Since the
  version does not move, **nothing in the bytes tells the two builds apart**: a form either writes
  decodes on both and then advances differently, and this contract states that rather than
  guarding it.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a wall with one gap, its detour leaving any start-target rectangle grown by 8 | a unit ordered across it, advanced to arrival in each mode | it arrives in **both** and stands on no ground-blocking cell at any tick |
| **AC-2** | unit | three orders no route serves: that wall with its gap closed, a target out of bounds, a target blocking ground | each advanced 16 ticks | *(amended 2026-07-31 by 0037, whose FR-1 makes a wave that cannot label its goal settle for the nearest cell it can reach)* under the **optimised** mode, each target is cleared on the **first** tick, the unit unmoved, its stall zero; under the **canonical** mode each unit instead walks to the cell its own failed wave settled for and its order ends on arrival there. Either way its stall is zero and no error is raised — FR-7 is untouched, and what moved is when a far search finds nothing |
| **AC-3** | unit | a cell another unit stands on, eight movers ordered onto it from across the map | advanced until they give up | each **advances** toward it, none enters it, each stalls only once it can get no closer and loses target and route on its sixteenth stalled tick |
| **AC-4** | unit | a 256x256 world with a unit mid-order; and a 64x64 world, eight units ordered once each | a tick advanced on the first, the cells its near search labels recorded; the second advanced to arrival, searches counted by kind | at most 289 cells are labelled and none outside the window, where a far search there labels more; and far searches equal the units ordered plus the departures recorded, no unit-tick running two near ones |
| **AC-5** | unit | worlds differing in one route cell, in a route's length and in holding a route at all; one of them marshalled then unmarshalled | each marshalled and hashed | the three differ pairwise in form and digest; the round-tripped pair are equal in both |
| **AC-6** | unit | forms with a route on a targetless unit, one ending other than at the target, an out-of-bounds cell, a cell blocking ground, two non-neighbouring cells, and the previous version | each unmarshalled | each is refused and the receiving world is unchanged |
| **AC-7** | unit | a corpus of worlds walked to arrival, obstacles forcing detours | each marshalled every tick, decoded into a second world, both advanced | the two hold equal digests at every later tick, in both modes |
| **AC-8** | unit | a hand-read route of eight cells; three units — one re-targeted mid-walk, one that failed to move, one whose sub-goal left its window | the sub-goal read, a unit landed on route cell 2, a unit landed off the route; a tick advanced for each of the three | the sub-goal is cell 3; cells 0 to 2 are dropped; nothing is dropped off the route; the first and third run one far search, the second none |
| **AC-9** | unit | an all-passable grid with one mover; two units whose direct lines cross | advanced to arrival in each mode | each arrives after `max(\|dx\|,\|dy\|)` ticks with its target cleared; the pair never share a cell and a re-run reproduces both |
| **AC-10** | bench | the harness at 256x256/40, 128x128/40, the sealed goal, the held cell | each over its 16-tick window, and 256x256/40 over 256 ticks | every figure is at or under FR-8's ceiling for its shape |
| **AC-11** | manual | the game against a lawful install, a large map with a lake | 40 units ordered across it, then onto an occupied cell | the frame rate holds, the group walks round the lake, and the second order makes it walk over and crowd rather than stand still |
| **AC-12** | unit | a map split by a channel with a land route round one end, the way round needing more rings than `max(5, D>>2)` over the straight line; and a corridor whose only route needs more than 1000 | a unit on each ordered across, advanced to arrival in each mode | the first arrives in **both**, on no ground-blocking cell, where under the old budget that world refuses in canonical and arrives in optimised; the second is refused in canonical and routed in optimised, which is FR-10's stated residual |
| **AC-13** | unit, bench | far searches whose goal terrain seals off, one near and one far; a search that succeeds; the four harness shapes and a fifth, 40 units ordered to a near sealed cell | labels counted before this change and after; each benchmark over its 16-tick window | the succeeding search labels the same cells in both; a sealed one labels no more than the map holds, at **either** distance; the four shapes hold FR-8's ceilings and the fifth FR-11's |
| **AC-14** | unit | every byte form and digest pinned in the tree; and a run recorded from the tree **before** this change whose far search finds nothing | each re-run; that run repeated tick for tick | every pin is unmoved and the version byte is where FR-3 left it; the recorded run's digests differ from the first tick on, its unit walking where it stood and ending elsewhere |

**Error cases:** AC-6, with AC-2 for an order no route serves. A refused form leaves the receiving
world untouched, an unservable order ends as a cleared target and never as an error.

## Derived properties

- **P-1 (invariant)** — Every cell a near search labels lies inside that unit's window, and no
  more than 289 do.
- **P-2 (invariant)** — After every tick a stored route ends at its unit's target, holds only
  in-bounds cells clear of ground blocking and steps between neighbours; a unit with no target
  holds none.
- **P-3 (negative-invariant)** — On a tick where a unit's near search finds no route, its cell,
  all three target fields and its whole route are unchanged; the stall count is the only field of
  its own it moves.
- **P-4 (completeness)** — A unit holding a target has exactly four outcomes per tick and no
  fifth: it advances one cell; it holds with its stall raised; it gives up at the threshold; or
  its far search finds nothing and its target is cleared in that tick.
- **P-5 (invariant)** — For two worlds with equal byte forms advanced by equal commands the
  states are equal at every tick, stored routes included.

## I/O examples

```
route  (4,7) (5,6) (5,5) (5,4) (5,3) (2,0)  ->  sub-goal (5,4), the cell at index 3
walk   lands on (5,6): route becomes (5,5) (5,4) (5,3) (2,0), sub-goal (2,0)
step   steps to (6,6), not among the first four: the whole route is unchanged
form   a unit's route is a cell count and that many cells; with none, a count of 0
```

## Out of scope

- **Staggering searches across ticks**: the order tick pays one far search per unit ordered.
- **Substitute destinations**, and approaching a goal that cannot be entered. Where a far search
  still finds nothing the original does not refuse: the search itself picks a nearby goal and
  extracts to it. Ours clears the order. That is decoded and it belongs to
  **`0037-approach-and-path`**; here a unit walks as near as its route takes it, then stalls.
- **Ownership.** No entity carries an owner, and nothing here scripts a scenario, so every order
  this build can issue is a player's and FR-10's budget applies to all of them. The budget a
  non-participant owner's unit would take is a later story's, and so is the owner field.
- **Reservations**: no unit marks the cell it means to enter, so two may still be routed toward
  one and the later-resolved one finds it taken.
- **Per-cell terrain cost, speed, sub-cell movement, parallel search, group movement and
  formations, flow fields, path smoothing, the drawn path preview, fog of war, combat**, the
  sprite, animation and facing layers, and **multi-cell footprints and flyers**, as before.

## Verification mapping

AC-1 to AC-9, AC-12, AC-14 and AC-13's counted half are unit tests over synthetic worlds built in
test code, so all are CI-automatable; AC-10 and AC-13's timed half are the committed benchmark
harness, whose figures are machine-dependent — which is why FR-8 and FR-11 each carry a counted
bound beside the milliseconds — and AC-11 needs a lawful install and a window. P-1 to P-5 are
**sampled, not proved**: P-1 over AC-4's runs, P-2 and P-5 over AC-7's corpus, P-3 and P-4 over
AC-3's give-up run.

## Gate check

FR-1 -> AC-1, AC-3 · FR-2 -> AC-1, AC-4, AC-12 · FR-3 -> AC-5, AC-6, AC-7, P-2 · FR-4 -> AC-4,
AC-9, P-1 · FR-5 -> AC-8 · FR-6 -> AC-4, AC-8 · FR-7 -> AC-2, AC-3, P-3, P-4 · FR-8 -> AC-4,
AC-10, AC-11 · FR-9 -> AC-7, AC-9, P-5 · FR-10 -> AC-12, AC-14 · FR-11 -> AC-13 ·
FR-12 -> AC-14.
