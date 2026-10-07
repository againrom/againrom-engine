# Plan — one relation split in two, a window on the near half, and a route that is state

## Baseline

`Step` makes one `routeScratch` per tick and walks the entities in ascending id. At a unit's turn
`subGoal` says whether the stored route serves; where it does not one **far** search runs over
`terrainRelation` with `noWindow` and `farSlack = 5` and its route is stored in `w.routes[i]`, and
a **near** search then runs over `unitRelation` with `dynamicWindow = 8` and `nearSlack = 3` to
that sub-goal. `searchRoute` is the one place `w.mode` is read, dispatching to `canonicalRoute`
(the wave) or `optimisedRoute` (a heap Dijkstra outward from the target); both take the relation,
the window and the slack as parameters, and both refuse — before any sweep — a target outside the
window or not open under the relation.

The wave then computes `budget := generationBudget(start, target, slack)`, `max(slack, D>>2) + D`,
**unconditionally**, and returns no route on `gens >= budget` with the goal unlabelled.
`optimisedRoute` has no generation count at all: its heap runs until nothing is left to pop, so
with `noWindow` it floods the whole map and finds a detour of any length. The byte form is version
4 with a route section per entity. `grouporder_test.go` carries the harness over four shapes;
`counted_test.go` counts labelled cells off the scratch's touch list and searches by kind.

## Design decisions

### DD-1 — one search of each kind, two relations, and the relation is a parameter

`terrainOpen(x, y)` is split out of `enterable` — bounds and the blocks-ground bit, no entity read
— and both searches take a flag saying which relation they run over. The far search passes
terrain-only, the near search the whole predicate, so there is one wave and one Dijkstra rather
than two of each. The corner test in optimised mode asks the same flag, since a rule that read
occupancy for the corner and terrain for the cell would be neither relation.

Rejected: **a second copy of each search**, one per relation — two contracts to keep in agreement,
and the spec says the mode governs route choice and nothing else (FR-1).

### DD-2 — the window is a Chebyshev square on the mover, tested at the offer, and it replaces the region

`dynamicWindow = 8`. The near search carries the mover's cell and refuses a neighbour whose
Chebyshev distance from it exceeds 8; the walk back refuses the same cells, so no route can leave
what the flood could not label. The far search carries no window at all, and `region`,
`searchRegion` and `regionMargin` are **deleted**: with the window on the near search, the system
holds exactly one spatial bound and it is on the half that is allowed one.

The value follows from the look-ahead rather than from taste: a sub-goal is at most four cells
away, so 8 leaves a detour as much room again as the sub-goal is far, and bounds the labelled
cells at `17 * 17 = 289`.

Rejected: **keeping the region and adding the window** — two bounds, and the region is the
start-goal box whose refusals this story exists to remove. Rejected: **clipping the label plane to
the window** — a second indexing beside the grid's, where a test at the offer costs one compare
(FR-2, FR-4).

### DD-3 — stored routes live on the world, parallel to the entities

`World` gains **one** field, `routes [][]cell`, with `routes[i]` belonging to `entities[i]` and an
empty slice meaning none. `Entity` gains nothing, so it stays free of pointers and `Entities()`
keeps handing out copies that reach nothing of the world. The pin table moves by one row, which is
this story's declaration that the field is canonical — the pin refuses an **undeclared** field,
which is exactly what a cache would be.

Rejected: **`Entity.Route []cell`** — the record type's own contract is that every field is an
integer or a bool. Rejected: **a per-world cache outside the digest** — measured wrong: over
130 436 recomputations of a stored route from its own later cells, 129 disagreed with the tail and
51 refused outright (FR-3).

### DD-4 — the constructor does not change, so no caller can invent a route

`NewWorld` keeps its five parameters and builds a world whose routes are all empty. The only
writers are `Step` and `UnmarshalBinary`. A world therefore begins unrouted whatever built it,
which is the state FR-3 requires the form to accept, and no call site outside this package moves.

Rejected: **a sixth parameter** — every caller would have to pass nil, and the one thing it would
buy is a way to hand a world a route nothing searched for (FR-3).

### DD-5 — version 4, the routes last, and the declared entity count becomes load-bearing

`[header][grid][records][routes]`, the header's version byte raised to 4. A route is a `uint32`
cell count then that many `int32, int32` cells, one route per entity in record order. The records
can no longer be sized by dividing what is left, since a variable-length section follows them, so
the decode multiplies the **declared** count by the record width in `int64`, checks it fits, and
then requires the route section to consume the remainder **exactly** — a wrong count fails one of
those two, so the count is checked rather than trusted even though it is now used.

Rejected: **the routes before the records** — the count would then have to be trusted with nothing
left to cross-check it against. Rejected: **a fixed-width route slot per record** — no bound on a
route's length exists that is not the map's own size (FR-3).

### DD-6 — the tick's shape, in the order the tests run

At a unit's turn: begin-on-target clears and returns; otherwise the stored route is taken and
replaced by a far search when it is absent, when its last cell is not the target, or when the
sub-goal it yields is outside the window; then the near search runs to that sub-goal; then the
move, then the consumption of any route cell the mover landed on among the first four.

The far search reads no occupancy, so **its answer does not depend on where in the tick it runs** —
which is what makes staggering it across ticks a scheduling change later rather than a redesign.
The near search does, so it stays where it is, at the unit's own turn, and the occupancy plane goes
on moving in the statement that moves a unit.

Rejected: **a refresh period**, the original's own `StaticRefreshRate`. It is a second canonical
byte per unit and a whole-map sweep every 16 ticks for every mover, where the three tests above
already fire exactly when the stored route cannot serve. Rejected: **re-searching on a near
failure** — that is the held-cell shape, and it would pay a whole-map sweep per stalled unit per
tick, which is the cost the previous revision just removed (FR-5, FR-6).

### DD-7 — a far failure clears the order, a near failure stalls

The far relation is bounds and the grid, both immutable while a world is advanced, and a unit that
fails does not move — so a second far search from the same cell is the same search, and sixteen of
them are sixteen identical answers. The target is therefore cleared in the tick that finds it. The
near relation is occupancy, which changes every tick, so a near failure keeps everything and raises
the stall, and the give-up at 16 stays exactly as it is, clearing the route with the target.

Rejected: **one rule for both** — either the sealed map costs sixteen whole-map sweeps, or a unit
briefly blocked by a neighbour loses its order. Rejected: **a substitute destination**, the
original's answer, whose pickers are undecoded (FR-7).

### DD-8 — the harness grows a tick count, and nothing else about it moves

`gopRun` takes the ticks to advance; the three shipped shapes keep `gopTicks` and gain a
256-tick run of the 256x256/40 shape, which is what separates the order tick's sweep from the
steady state. The two fixture tests whose claims this story **changes** — the sealed goal and the
held cell, where movers now walk instead of standing — are rewritten to the new contract rather
than deleted, so the shape each benchmark measures is still asserted.

Rejected: **a second harness** — the figures this story is judged against are the ones already
recorded, and a new instrument would not be comparable with them (FR-8).

### DD-9 — the budget arrives as a rule, where the slack arrived

`generationBudget` takes a `budgetRule` in place of a slack: `flatBudget` answers `MOVE-TERM-003`'s
constant **1000**, `scaledBudget` answers `max(nearSlack, D>>2) + D`. `budgetRule` is a
**bool-shaped named type**, exactly the shape `relation` already has — never a func value, which
would be the second behaviour-steering identity beside the mode byte that this package's own
header refuses. `step.go`'s two calls are the only choosers, `optimisedRoute` takes no budget and
is not opened, and `farSlack` goes with the arm it fed: a named constant nothing reads records the
decoded scalar worse than a line in `provenance.md`. `nearSlack` is untouched.

The gate's goal-free half is discharged by the **order of the stop tests**, not by an early
return. `generationBudget` is *computed* on every call that gets past the pre-sweep refusal —
including `start == target`, where that refusal is skipped — but it is *consulted* only after the
goal-labelled test, which breaks at generation zero when the start is the target. So `gens >=
budget` is evaluated only where the target is neither the start nor closed under the relation, and
for the far arm that relation is `terrainOpen`. The condition therefore holds wherever the budget
can bind, and a test of its own would be dead code. That invariant is **not local to `route.go`**:
it also rests on `Step` clearing an order whose unit already stands on its target, so the far arm
is never called with the two equal.

Rejected: **the budget computed by the caller** — `step.go` would then hold the formula the two
searches share, the second place it can come to disagree. Rejected: **the flat value read as "no
budget"** — 1000 is what is decoded, and it is a real bound: past a Chebyshev distance of 800 it is
*smaller* than the old one, which FR-10 states rather than smooths away (FR-10, FR-12).

### DD-10 — the gate's other half names an owner this build does not have

`UNIT-OWNER-009` puts the gated field on the mover's owning player, zero exactly for a human
participant and 1 or 2 for a scenario-authored owner. `sim.Entity` carries no owner and nothing
here scripts a scenario, so **every order this build can issue is a player's**: the condition holds
wherever it is asked, and the flat budget is taken for every far search as the ordinary case
rather than as an exception. That is a premise about our scope, not a divergence from the source,
and it is the whole of why the arm goes in unguarded. What it costs SC-13 measures rather than
argues; what it buys is the order the owner cannot currently give.

Rejected: **an owner field to carry the condition** — canonical state, a new field in a form FR-12
freezes, and a whole ownership model to answer a branch nothing here can reach. Rejected:
**keeping the scaled form behind that absent condition** — dead code chosen to look cautious, and
a second budget rule to keep in agreement with the first (FR-10, FR-11).

### DD-11 — the fork measurement outlives the budget it was built on

`routefork_test.go` is DD-3's evidence that a stored route is not recoverable from its own later
cells, and its strongest class — a recomputation **refused outright**, 57 of the 70 differences —
exists because the old budget *shrank* as a mover closed on its goal. Under a flat budget it goes
to zero by construction: the goal is still terrain-open, the tail proves the component
is connected, and no hop count on that corpus approaches 1000. So the test's `refused != 0`
assertion is an assertion about the rule this revision removes, and it is rewritten to the
property it was always evidence *for* — the tail is not recoverable — with the refusal count
**reported** rather than required. What remains is the weaker class, an equal-cost route through
different cells, which survives because the wave still stops in the first generation to label the
goal, so which route it returns still depends on where it started.

Rejected: **deleting the test with the budget it measured** — FR-3 would then rest on a
measurement nothing re-runs. Rejected: **weakening the corpus until a refusal reappears** — hunting
a fixture that keeps an assertion alive is how a green suite stops meaning anything (FR-3, FR-10).

## Risks

- **R-1** The order tick is a spike: one whole-map sweep per unit ordered, ~104 ms for 40 units on
  256x256. It is one tick rather than every tick, and staggering is out of scope, so the owner will
  still see a hitch when he issues a large group order.
- **R-2** The sealed-goal and held-cell shapes get **slower**, from 25.4 and 0.013 ms a tick to the
  cost of an ordinary walk, because a unit-blind far search routes to a cell units are standing on
  or around. That is the behaviour we want and the figure will still read as a regression next to
  the last revision's; it is disclosed rather than tuned away.
- **R-3** The byte form grows with the routes — up to a few hundred cells per moving unit, so of
  the order of 100 KB for a large group on 256x256, beside the grid's 64 KB. Every `Hash` walks it.
- **R-4** A unit whose sub-goal is pushed out of its window buys a far search, and nothing bounds
  how often an adversarial corridor could make that happen beyond one every four ticks.
- **R-5** A stored route was bounded by ~1.25 x the straight-line distance and is now bounded by
  the budget: up to 1001 cells, ~8 KB a unit, ~320 KB for forty of them — walked by every `Hash`.
  R-3's "a few hundred cells" understated it by about three times.
- **R-6** The two modes still disagree past 1000 generations, and above a Chebyshev distance of
  800 canonical is now the **tighter** of the two budgets. No map this build loads reaches either,
  since a decoded map is at most 256 a side, but `sim.Bounds` accepts any size and nothing refuses
  one — so the residual is bounded by what the loader produces rather than by the contract.

## Success criteria

- **SC-1** AC-5 and AC-6 hold in full: every offset and width re-pinned at version 4, the digest
  pin recomputed by hand from the pinned bytes rather than from a run of this package, and each
  refusal carried by a case of its own (FR-3).
- **SC-2** AC-7 holds over a randomised corpus in both modes, the world marshalled at **every**
  tick and the decoded copy stepped beside the original, so routing state left outside the form
  shows as a divergence rather than as a passing pin (FR-3, FR-9).
- **SC-3** **The fork is measured in the tree, not argued in the docs.** A committed test
  recomputes a stored route from every one of its own cells and compares it with the tail, over at
  least 100 000 pairs of random terrains in canonical mode, reporting disagreements and outright
  refusals. A run that came back zero would say the route need not be hashed, and would be a
  finding rather than a pass (FR-3).
- **SC-4** AC-1 holds in **both** modes, with the gap placed outside any start-target rectangle
  grown by 8. That fixture is what discriminates the region's removal: against the shipped
  optimised mode it answers no route, so a green run there is evidence and not a formality
  (FR-1, FR-2).
- **SC-5** AC-2 and AC-3 hold in full — the three unservable orders each cleared on the **first**
  tick with the stall left at zero, and the held-cell group seen to advance, to stall only once it
  can get no closer, and to give up on the sixteenth stalled tick (FR-7).
- **SC-6** AC-4 holds in full, both halves **counted** rather than inferred: labelled cells per
  near search, taken from the scratch's own touch list, and searches by kind over a whole run
  (FR-2, FR-4, FR-6, FR-8).
- **SC-7** AC-8 holds on a route small enough to read by hand, each of the three staleness cases
  its own fixture, and the unit that stepped off its route asserted to keep every cell of it
  (FR-5, FR-6).
- **SC-8** AC-9 holds in both modes (FR-4, FR-9).
- **SC-9** AC-10 holds: the harness is run at all four shapes on the tree **before** this story and
  again after it, both sets recorded, and the 256-tick run reported beside the 16-tick one so the
  order tick's sweep is separable from the steady state (FR-8).
- **SC-10** Eight mutants, each applied to production code, run over the whole tree with its
  failing tests named, and reverted: the far search made to read occupancy; the window test
  deleted; the window applied to the far search as well; the route section dropped from the
  encoding; the last-cell staleness test dropped; the sub-goal taken as the route's last cell; a
  far failure made to stall instead of clearing; the route left behind on give-up. A ninth runs as
  a **deliberate survivor** — the near search's
  budget slack moved from 3 to 5 — which the window makes inert, and which is how DD-2's claim
  that the window is the binding bound is discharged as a measurement (FR-1, FR-2, FR-3, FR-4,
  FR-5, FR-6, FR-7).
- **SC-11** AC-11 is run against a lawful install on a map with a lake (FR-8).
- **SC-12** **AC-12's fixture discriminates and asserts its own claim**: the crossing really
  sealed, the way round really needing more rings than `max(5, D>>2)` over the straight line, so a
  green arrival is evidence and not a fixture that happens to fit. The same world driven under the
  **old** budget refuses in canonical and arrives in optimised — the defect, in the tree, before
  the fix takes it away (FR-10).
- **SC-13** **The cost is measured, on both instruments, at both revisions.** The four shipped
  shapes all end in a far search that succeeds and so cannot see this change; the fifth orders 40
  units on 256x256 to a cell terrain seals off at a **short** distance, where the old budget was
  smallest and the new one is the same full sweep — the largest ratio the map admits, a far sealed
  goal showing about one. Beside it, labelled cells are counted for a sealed goal near and far and
  for a search that succeeds, side by side at both revisions, because milliseconds are this
  machine's and a label count is not. The four shipped shapes are re-run rather than assumed, and
  a figure over FR-11's ceiling is reported as this revision failing, not tuned (FR-11).
- **SC-14** **Nothing pinned moves.** Every byte form and digest this tree pins is re-run and
  compared and the version byte asserted to be where FR-3 left it; one run whose far search fails
  today is pinned digest by digest **before** the change and asserted to differ from the first tick
  after it. Two mutants, applied to production code, run tree-wide with their failing tests named,
  and reverted: the flat budget put back to the scaled form, which AC-12's channel must kill; and
  the flat budget given to the **near** search too. The second is **not** killed by the 289-cell
  bound — the window is enforced at the offer, independently of any budget, so a flat near search
  still labels no more than 289. Its killer is the near flood's own **reach** assertion, the case
  four cells off where the budget stops the flood before the window does; that assertion is one of
  the budget assertions being rewritten here, and it must come out of the rewrite still pinning a
  reach, not relaxed to a bound the window already gives (FR-10, FR-12).
- **SC-15** **The fork is re-measured, not inherited.** SC-3's corpus is re-run at the flat budget
  over the same shapes and seeds, its differing and refused counts reported beside the figures the
  old budget produced. A refusal count of zero is the expected outcome and is reported as
  such; a **difference** count of zero is a finding against FR-3 and is reported as a failure of
  this revision rather than as a pass (FR-3, FR-10).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1 | SC-4, SC-5, SC-10 |
| FR-2 | DD-2, DD-6 | SC-4, SC-6, SC-10 |
| FR-3 | DD-3, DD-4, DD-5, DD-11 | SC-1, SC-2, SC-3, SC-10, SC-15 |
| FR-4 | DD-1, DD-2 | SC-6, SC-8, SC-10 |
| FR-5 | DD-6 | SC-7, SC-10 |
| FR-6 | DD-6 | SC-6, SC-7, SC-10 |
| FR-7 | DD-7 | SC-5, SC-10 |
| FR-8 | DD-8 | SC-6, SC-9, SC-11 |
| FR-9 | DD-3, DD-6 | SC-2, SC-8 |
| FR-10 | DD-9, DD-10, DD-11 | SC-12, SC-14, SC-15 |
| FR-11 | DD-10 | SC-13 |
| FR-12 | DD-9 | SC-14 |
