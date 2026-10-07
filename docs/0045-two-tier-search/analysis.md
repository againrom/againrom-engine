# Analysis — the two-stage search

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the profile puts the simulation's determinism and its byte form at this tier; no watcher tool exists, so it is discipline |
| Terrain — the static/dynamic split, the window, the stored route and its section of the form | **greenfield**: none of it exists in any shape |
| Terrain — the wave, the optimised search, the per-tick scratch, the resolution loop and the version-3 form | **brownfield**: every one is shipped, and each is changed here |

## Where the cost is, measured before anything was written

There is no clean-room baseline for this one: it is authored from the owner's scope and from a
defect he met in a build he ran, so nothing below is a disagreement with a document. It is what
this tree measures. Milliseconds per tick, from the harness the previous revision landed:

| shape | before that revision | after it |
|---|---:|---:|
| 256x256, 40 units to a far cell | 435.9 | 102.3 |
| 128x128, 40 units to a far cell | 104.0 | 25.7 |
| 80 movers, goal sealed by units | 231.0 | 25.4 |
| 80 movers onto a **held** cell | 183.9 | 0.013 |

Two of the three causes are gone: enterability was O(entities) per cell tested, and a search
whose destination could not be entered spent its whole budget before failing. What is left is
**~2.6 ms per unit per tick**, and it is one sentence — a whole-map route is computed for every
moving unit on every tick and everything but its first cell is thrown away. The curve is linear
in the group size now, so what is left is a per-unit cost and the fix has to change what one
unit costs.

## The fork, and the measurement that settled it

A stored route is state. Either recomputing it gives back what was stored — in which case it is
a cache, needs no digest and no version bump — or it does not, in which case a world saved
mid-order and reloaded walks a different route from one that was never interrupted, and the
route has to be in the hashed state.

The argument for the cache is that the static route is unit-blind and terrain cannot change
while a world is advanced. Its hole is that **the mover advances, so the start moves**, and the
wave stops in the first generation that labels the goal — the route is not a shortest path, and
nothing makes the tail of one route equal the route from the cell that tail starts at.

Measured rather than argued: compute a route from S to G, then from **every** cell of that route
recompute the route to G and compare it with the stored tail. Random terrains, one entity in the
world so the relation is terrain alone, a fixed seed per shape.

| shape | worlds | routes | (route, cell) pairs | tail differs | of those, refused outright |
|---|---:|---:|---:|---:|---:|
| canonical 16x16, 0 % and 10 % blocked | 3 979 | 3 846 | 25 455 | 0 | 0 |
| canonical 16x16, 25 % and 35 % | 3 976 | 3 799 | 27 057 | 13 | 0 |
| canonical 32x32, 10 % and 25 % | 1 998 | 1 982 | 28 398 | 0 | 0 |
| canonical 32x32, 35 % | 1 000 | 964 | 14 615 | 8 | 8 |
| canonical 64x64, 15 % and 30 % | 799 | 796 | 23 170 | 6 | 0 |
| canonical 64x64, 38 % | 400 | 383 | 11 741 | 102 | 43 |
| optimised 16x16, 0 / 10 / 25 / 35 % | 7 971 | 6 995 | 57 783 | 0 | 0 |

**20 123 worlds, 18 765 routes, 188 219 pairs, 129 differing tails — every one of them in
canonical mode**, and the rate climbs with obstacle density: 0 below 25 % blocked, 0.87 % at
64x64 and 38 %.

**51 of the 129 are the strongest class there is: the recomputation returns _no route at all_
where the stored tail is a route the unit was already walking.** The budget is
`max(5, D>>2) + D` generations over the Chebyshev distance from the start, so it *shrinks* as
the mover closes on the goal, and a detour that fitted from the original cell stops fitting from
a later one. The rest are quieter — an equal-cost route through different cells — and 62 of the
129 differ at the very first cell, which is the cell the tick actually walks.

So the answer is **canonical**: the route is hashed, the form's version is raised, and the
readiness threshold is **High**.

The optimised half says something about the argument rather than about the contract. Its walk is
greedy over exact costs to the target, so a suffix of one of its routes is a route it would
itself produce — 0 differences in 57 783 pairs, on grids small enough that its region covers the
whole map. That is the cache argument coming out true, for one mode, and it changes nothing: a
byte form cannot be canonical in one mode and derived in the other.

## What the pinned claims settle, and what they leave to us

The two-search arrangement is read at High and was declared a non-goal by the story that landed
the search — that story's own analysis calls it *"a whole tier of its own"*. The claims give the
shape (one search unit-blind on the block plane, one unit-aware on the plane that carries
occupancy; the second aimed a few cells down the first's route and rebuilt on every completed
cell transit) and the scalars behind it. They do **not** give: any spatial bound on either
search — the original bounds only by generations; what a failed search should do beyond "the
caller substitutes a nearby goal", whose pickers are unread; or any staggering, which the ledger
is explicit does not exist.

## What was weighed and refused

- **One search with a window on it.** The shape the imported baseline had, and the reason this
  story exists: a goal reachable only by a detour bulging past the margin becomes unreachable,
  and around a lake that is not a corner case. A window is safe exactly when something else is
  unbounded.
- **A route stored as the cell it was computed from and recomputed on load.** Smaller in the
  form, but the decode could then validate nothing it carries without running a search, and the
  world would hold the same fact twice.

## The revision: a budget shaped by the straight line

The owner ordered a unit across a river **where a land route exists**. Up, across and down as three
orders walks; the far bank clicked once moves nobody — as if a line were drawn between the two
points and the trip rejected for crossing the water.

It is not about water. `canonicalRoute` computes `generationBudget` **unconditionally**, and a
generation advances the wave one ring, so the whole allowance for leaving the straight line is
`max(5, D>>2)` rings where an up-and-around route needs about twice the crossing leg. The budget
is spent, the goal is never labelled, and `0029`'s unservable-order rule clears the order in the
first tick; a wall of that shape does the same. AC-1's fixture could not catch it — it asserts its
own gap fits, *"the wave's budget is 50 for a walk of 40"* — so it was built inside the bound it
now has to exceed. Optimised mode has no generation count at all and has been routing these orders
all along: the two modes have disagreed about what is reachable since this story landed.

**The fork** was going to be whether to take a decoded arm on half a gate. It closed instead of
being decided: the term the gate reads was put to research and came back the same day, and
`provenance.md` carries what it says. The field is the mover's **owning player**, zero exactly for
a human participant — so the flat budget is the *ordinary* budget of a player-ordered move, and
this engine has been missing the default rather than skipping an exotic branch. Every order it can
issue is a player's, because no entity carries an owner at all: a premise about our scope that a
later ownership story inherits, and not a divergence from anything.

A **wider slack** stays refused: a slack is a function of the straight line however large, so the
same order fails at a longer river, and the defect is the bound's shape not its size.

## What we looked at

`pkg/sim/{route,optimised,step,world,binary,hash}.go` and the suites that pin them —
`nostate_test.go`'s field and digest pins, `grouporder_test.go`'s harness and fixtures,
`preserved_test.go`, `relaxation_test.go`, `optimality_test.go` — plus `pkg/mapload/fromalm.go`,
the one production constructor call site; `docs/0029-unit-pathfinding/` in full, its revision
included; in research at the pin, `claims/retracted.md` and the registry's standing corrections
first, then `claims/move.md` end to end and `formats/move/format.md`.

For the revision: `route.go`, `optimised.go`, `step.go`, `wall_test.go` and `counted_test.go`
again, against `MOVE-TERM-003`, `MOVE-PARAM-006` and `MOVE-ALT-018`…`MOVE-ALT-022` at pin
`8c92427`, and against `UNIT-OWNER-009` and the amendment around it, which are **ahead** of that
pin and were read from what research reported rather than from this tree.

**2026-07-31.** The pin moved `8c92427` → `f35be34` at the 0047 boundary, which is where the
implementing branch starts, so the sentence above is no longer true: both are **at** the pin. Both
were re-read from the submodule in this tree, end to end, rather than from a report. Nothing in
either differs from what this story assumed of it.
