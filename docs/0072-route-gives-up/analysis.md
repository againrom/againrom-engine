# Analysis — the route the hero does not take

## What we did not know

Ordered across the campaign's tenth map, the party's hero walks fifteen cells, stops, and ends holding
no order at all. A flood fill over the passability plane from his own start reaches 1427 cells and the
ordered cell is among them, so a route exists **over terrain**. That is all the flood fill establishes:
it ignores entity occupancy, and thirty-five other units stand on the map.

Three causes produce exactly that observation and the flood fill separates none of them.

1. **The far search abandons long routes.** It runs under a generation budget; if the budget is the
   bound, the question is what the original allows and whether raising it is lawful.
2. **A body holds a chokepoint.** The terrain plane cannot see units, so a one-cell gap with a unit in
   it reads as open and the walk is genuinely blocked. Then nothing is defective and the story is a
   different one.
3. **The give-up rule fires on a mover that should have re-routed.** A near failure raises a stall
   count and the sixteenth consecutive one drops the order.

## What was measured

The mission started from the lawful install and driven headlessly with ordinary move commands,
watching the mover's own fields every tick — position, target, stall count, stored route — beside two
breadth-first walks of the map, one over terrain alone and one over terrain and occupancy together.

**(a) The far search is not the cause.** On the tick the order was applied the mover already held a
stored route of **71 cells ending on the cell ordered**, and its target was never rewritten. So that
search neither spent its budget nor settled for a substitute: it succeeded. The terrain-only walk puts
the destination **72 steps** from the start, against a flat budget of a thousand generations. Ruled
out by the route the search actually returned, not by arithmetic about the budget.

**(b) No body seals the way.** The win needs the mover within Chebyshev 3 of its point, which is 31
open cells. Under terrain **and** occupancy together, **27 of those 31 are reachable** from the hero's
own start, the nearest at 69 steps; the four that are not are the four that units stand on — including
the ordered cell itself. A walkable route to the objective exists with every body left where it is.

**(c) The stall count runs out on a search that never ran.** The mover stops at `(32,53)` holding a
route of 57 cells whose **fourth cell, `(36,51)`, a unit occupies**. That cell is the near search's
waypoint. Sixteen consecutive ticks later the order is dropped, position unchanged. The 17x17 window
around the mover shows the body standing in a three-cell gap with **two of the three free**, and a
four-step bypass back onto the route beyond it.

So it is (3), and one degree sharper than (3) was stated: the stall count is not counting failed
searches. The near search refuses a waypoint no mover may enter **before running a wave at all**, so
sixteen ticks buy sixteen refusals of a question never put — and because the far search reads terrain
alone, re-ordering reproduces the same route and the same refusal exactly.

## What the code and the decode disagree about

`route.go` gives the near search `exactGoal` — it must return the cell it was aimed at or answer that
there is no route — and says why: a near search answering with another cell "would be a second,
unstated rule about where a mover steps". The far search settles; the near one may not.

The decode says both searches settle, and differ in the ring bound alone: the static branch takes the
bound shaped on the distance ordered, the dynamic branch a flat **8**. The rule the comment reasons
its way to is one half of a decoded pair, and the missing half is the whole defect. The bound the far
arm already carries is that same claim's other branch — so the pair was read from one source and half
of it was implemented.

## What this leaves open

**Waiting.** The decode describes a mover that turns to face a blocked adjacent waypoint and waits
rather than re-searching — reachable only when the waypoint is exactly one cell away, and gated on a
predicate whose verdict table is undecoded. This tree has no wait arm and gains none here, so a mover
one cell from a body steps around it where the original may stand still.

**The cost plane.** This tree's worlds carry none, so every step costs the flat 2/3 and the rate law
takes its fallback mean. Both are visible in how long a walk takes and in which of two equal-cost
routes is chosen; neither is what stopped this mover, and both are a larger story.
