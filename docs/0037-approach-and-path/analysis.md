# Analysis — approaching a cell that cannot be reached, and drawing the way there

Three baselines were folded into this story on the owner's call: an approach for a blocked goal, a
cache for the path preview, and a bound on the overlay that draws it. What follows is what was not
known when they were written, and what reading the tree and the claim ledger changed.

## The algorithm the first baseline is written against is gone

That baseline names an A\* first-step routine, an **octile heuristic**, a **closed set**, a search
margin and a `start↔goal` box window costing `O(distance²)`. None of those names occurs anywhere in
this tree, and none of the things they name does either. `pkg/sim/route.go` holds a
**label-correcting wave**: no priority queue, no closed set, and no distance-estimate term of any
kind. Two searches run per mover per tick, told apart by a relation, a window and a budget rule,
and `pkg/sim/step.go` holds the only two call sites that choose them.

So the baseline could be read for its *subject* and for nothing else. Every requirement here was
written against the search that is in the tree.

## The problem it opens with had already been fixed, twice, differently

It opens: since terrain became real, a unit ordered onto a blocked cell **freezes at its start
cell**, and give-up abandons the order after the stall limit. Neither half is this tree's
behaviour, and `giveup_test.go` said so before this story: a **far** failure ends the order **in
the tick that found it**, with no stall at all, because terrain cannot change while a world is
advanced and a second sweep from an unmoved cell is the same sweep. The stall counter belongs to
the **near** search, whose relation does change. 0036's own AC-9 was repaired along that line, and
0045 restored the flat budget on top of it.

What survives of the premise is the part that matters, and it is worse than the baseline says: the
unit does not freeze for sixteen ticks, it never sets out at all, and the order is gone by the time
the player sees anything. The bug is real; the diagnosis was stale.

## The substitute-goal rule is decoded, and the baseline guessed

The baseline picks "the expanded node with the minimum octile heuristic to the goal". Research has
since read the routine. `MOVE-ALT-018`…`022` say the search substitutes **inside its own tail**,
that the rings expand outward **from the requested cell**, that the winner inside a ring is the
**minimum label** — the cell cheapest to reach *from the mover* — and that "free" means **labelled
by the wave that just failed**, not passable. The baseline's rule is a different rule that would
usually pick a different cell, and it was not taken.

Reading those rows also cost this story a design it would otherwise have taken. `MOVE-ALT-022` says
the substitute is **never written back**, so the engine re-substitutes from wherever the unit then
stands. That is affordable there because `MOVE-REFRESH-012` re-runs the static search only every
sixteenth dynamic one. This tree has no such period — `subGoal`'s three staleness tests fire
exactly when a stored route stops serving — so an order left pointing at an unreachable cell buys a
whole-map sweep **every tick for the whole walk**. That is the shape 0029 was a revision about. The
divergence taken instead is recorded in `provenance.md`.

## The pre-sweep refusal is 0029's, and it had to be reopened

`canonicalRoute` refuses a target that is not open **before running any wave**, and `step.go`'s
comment leans on it twice: once for the cost, and once to discharge the decoded budget override's
goal-free half, which it says "the refusal standing before any sweep already establishes". A picker
consumes the labels of the wave that failed, so there has to *be* a wave. Reopening the refusal put
the override's gate back where the decoded store puts it, and the gate is what keeps the cost
bounded: a settling far search runs `max(5, D>>2) + D` generations, not a flat thousand.

## The second baseline's subject no longer exists

That baseline caches a render-side route query because the overlay re-runs a full search every
frame for every targeted unit, and it cites 0029's "routes are never stored". **They are stored
now.** 0045 made the route a per-entity field carried by the byte form and covered by the digest;
the far search writes one, the advance consumes its walked head, and three tests replace it when it
stops serving. So a render-side query is a **read**, there is no recompute to cache, and a cache
here would need an invalidation rule — a second opinion about when a route went stale, on the side
of the seam with no state to answer it from. The whole of that baseline was refused. What was taken
from it is its measured concern, which the query's shape answers outright.

## The third baseline names a package and a field that do not exist

It scopes the overlay to the selection held in `pkg/game`. There is no path overlay in this tree at
all, and the selection is not there: it lives in `pkg/ui`, behind a seam that exists so that tier
**cannot name a simulation type**. That moved the whole overlay decision to `pkg/ui` and left
`pkg/game` doing a conversion, which is why the route crosses as a field rather than being asked
for by the tier that draws.

## Open, and not guessed at

- Whether a settled mover should ever re-derive its substitute as it advances. Taking the substitute
  as the order's cell is a divergence, disclosed, and its visible cost is that a unit settles for
  the first substitute rather than the one its later position would have chosen. Doing it the
  decoded way needs a refresh period this tree does not have and 0029 removed on purpose.
- Whether the optimised mode should settle at all. It cannot with the plane it builds, and nothing
  was invented to give it one.
- `MOVE-ALT-022`'s tolerance test and its message are not implemented: there is no message channel
  to raise one on, and the row itself says exceeding the tolerance does not cancel the move.
