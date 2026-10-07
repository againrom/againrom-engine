# Spec — a unit gets as close as it can, and the way there is drawn

## Baseline

A unit ordered onto a cell no route reaches does not move at all. Its far search refuses the target
before running any wave, the order is cleared in that same tick, and from the player's seat the
click did nothing. That is true of a cell closed by terrain, of a cell sealed off behind terrain,
and of a cell off the map.

Nothing in the tree can be asked where a unit is going. The route a mover walks is per-entity state
that the far search writes and the advance consumes, but it is unexported, and no front-end draws
it: there is no path overlay of any kind.

## Functional requirements

### The search (`pkg/sim`)

- **FR-1 — a search may settle for a substitute.** A search whose wave ends with the goal still
  unlabelled — its frontier spent or its budget out — MUST, when it is a search that settles, take a
  substitute goal and return the route to it. The substitute is chosen off **the labels that wave
  left and nothing else**: expanding square rings `r = 1, 2, …` around the **cell that was ordered**;
  each ring scanned **whole** and the **smallest label** in it taken, ties going to the earlier of
  the ring's own walk (`i = -r..r`, probing the sides `y+r`, `y-r`, `x+r`, `x-r` in that order); the
  first ring holding any labelled cell deciding; the centre never probed; and `r` growing while
  `r + 1 <` a limit of `(D>>2) + 4`, `D` being the Chebyshev distance from the mover to the ordered
  cell. A label is accumulated step cost from the mover, so the cell taken is the one **cheapest to
  reach from the mover**, not the one nearest what was ordered. A cell the wave never labelled is not
  a candidate however passable it is. When no ring inside the limit holds a labelled cell the answer
  is no route, exactly as before.
- **FR-2 — the flat budget is an override with a gate.** A far search MUST take the flat thousand
  generations only when its goal is **open under its own relation**; otherwise it MUST fall through
  to `max(5, D>>2) + D`. The near search MUST take `max(3, D>>2) + D` whatever its goal. So a
  settling search is bounded by the distance it was sent, and a mover sent across a map does not buy
  a thousand generations to establish what it cannot reach.
- **FR-3 — the order takes the cell the search settled for.** When a far search returns a route, the
  ordering entity's target MUST become that route's last cell. For a goal the search reached this
  changes nothing; for one it could not, the order from then on is an ordinary order to the
  substitute — its route serves, its arrival clears it, and no later tick re-runs the sweep.
- **FR-4 — which searches settle.** Exactly one does: the **far** search, under the **canonical**
  mode. The near search MUST answer only about the cell it was aimed at, so a near failure MUST go on
  raising the stall count and giving up at the limit unchanged; and the optimised search MUST go on
  returning no route, since the plane it builds is costs **to the target** and carries nothing about
  what the mover can reach.
- **FR-5 — settling on the mover's own cell ends the order.** The mover's own cell carries a label
  like any other and may be the one taken. The route to it is empty, and an empty route MUST end the
  order in that tick with no residue and no stall — which is the existing rule for a far search that
  yields nothing, reached without a new branch.
- **FR-6 — a route query.** `Route(id)` MUST return the cells that entity still intends to walk, in
  walking order, ending at the cell its order has settled on; and nothing for an entity this world
  does not hold, one holding no order, and one whose order has not been through an advance. It MUST
  be a **read of the stored route**, never a second search, and it MUST **copy**: a caller may not be
  handed a slice that still reaches into hashed state. It MUST NOT begin at the mover's own cell.

### The seam and the overlay (`pkg/game`, `pkg/ui`)

- **FR-7 — the seam carries the route.** The per-entity value the window tier receives MUST carry
  that entity's remaining route, converted to the window tier's own point type, built per **tick**
  and not per frame, and nil for an entity under no order. It MUST NOT carry the entity's current
  cell a second time.
- **FR-8 — what is previewed.** A pure decision MUST answer which entities the overlay strokes: those
  **selected**, still in the snapshot, still alive or downed, **and holding a route** — using the
  same presence-and-life predicate the orders and the selection rim already use.
- **FR-8a — all of them, whatever the count.** *(2026-08-01, the owner's ruling. FR-8 carried a cap of
  `PathOverlayMaxUnits` drawable entities over which the overlay drew **none at all**: a taste
  judgement with no claim behind it, whose failure mode was a large enough selection silently losing
  the whole overlay.)* The decision MUST answer **every** drawable entity. No count — of selected
  units, of drawable units, or of route cells — may reduce what is stroked, and **none** MUST be the
  answer only when nothing is drawable.
- **FR-9 — the drawn line.** The overlay MUST stroke, for each previewed entity, a polyline running
  from **that entity's own cell** through the cells of its route in order, each point placed through
  the same height lift and camera transform every other glyph on that cell goes through. A leg whose
  endpoints both lie outside the view MUST still be issued *(amended 2026-08-01 by FR-9a: when the leg
  itself meets the view)*. It MUST be drawn over every other pass.
- **FR-9a — a leg that can paint no pixel is not issued.** *(2026-08-01, with FR-8a.)* A leg whose own
  **bounding box**, grown by the stroke width, does not meet the view MUST NOT be issued. The test is
  the leg's box and never its endpoints: an endpoint test drops a leg that crosses the view, and this
  one cannot, because a box that misses the view holds no point inside it. It is therefore not a
  limit on what is drawn — **the picture is identical at every camera with it and without it** — and
  it is what bounds the cost FR-8a's removal of the cap would otherwise leave unbounded.
- **FR-10 — nothing canonical moves.** No new field, no codec change, no version change, no digest
  shape change. The overlay and the query MUST be reachable from no path that advances a world, and
  the tick MUST stay integer-only, with no clock, file or generator on any path added here.

## Acceptance criteria (synthetic, no game install)

| # | Given | When | Then |
|---|---|---|---|
| **AC-1** | a unit ordered onto a terrain-blocked cell, one sealed behind terrain, and one off the map | stepped to rest | it **walks toward** each and stops on the nearest cell it can reach, in every case — it does not stand still and its order is not thrown away in the first tick |
| **AC-2** | a blocked goal whose first free ring holds two cells, one cheap from the mover and one dear, the dear one offered **earlier** in the ring's walk; and a **cheaper** cell one ring further out | the search settles | the **cheap cell of the nearer ring** is taken — so neither "the cheapest labelled cell there is" nor "the first labelled cell scanned" produces this answer |
| **AC-3** | a mover sealed into its own cell, ordered anywhere | one tick | the order ends in that tick with no residue and no stall — the settle is its own cell and the route to it is empty |
| **AC-4** | AC-1's fixture under the optimised mode; and a near search aimed at a sub-goal it cannot reach | stepped | the optimised mover does not move and its order ends at once; the near failure raises the stall count and keeps cell, target and route |
| **AC-5** | a unit ordered onto a blocked cell | the first tick, then to rest | its order no longer names the blocked cell, its stored route ends exactly at the cell its order now names, and it walks the whole way there and parks — so no later tick re-runs the sweep |
| **AC-6** | two identical worlds on AC-2's fixture | lockstep to the stall limit | equal `Hash()` every tick — the same substitute at the same tick |
| **AC-7** | the four arms of the budget rule, and a goal close enough that the slack rather than the quarter-distance binds | asked directly | flat-and-open gives 1000; flat-and-closed gives `max(5, D>>2) + D`; the near rule gives `max(3, D>>2) + D` under either goal — so the two slacks are 5 and 3 and are not interchangeable |
| **AC-8** | a mover whose far route hands its near search a sub-goal four cells off, closed by **bodies** so the way round is ten steps, wholly inside the near window | one `Step` | the mover holds its cell and its count rises by one — driven **through the tick**, so the near call site's budget rule is what is measured and not a search called directly |
| **AC-9** | a unit before any order, an id no world holds, and a unit that has walked | `Route(id)` | nothing, nothing, and the stored route cell for cell; a write through the answer changes no digest, and two calls do not share a backing array |
| **AC-10** | a selection over units of which one is under orders, one is idle, one is a corpse holding a route, plus a targeted unit that is **not** selected, plus a selected id absent from the snapshot | the preview decision | only the selected units under orders, in the selection's own order; each of the four exclusions asserted on its own |
| **AC-11** | *(amended 2026-08-01 with FR-8a)* selections of **31, 32 and 33** drawable movers, of two hundred and of two thousand, and one far larger of which three are under orders | the preview decision | **all** of them every time, in the selection's own order, and the three — so no count answers short, and the value the removed cap held is not special |
| **AC-12** | a selected unit on a known cell holding a three-cell route; and one whose leg runs clean across the view from outside it | the segment build | three segments, the first from the **unit's own cell** to the route's head and the rest along it, each endpoint at the camera's own transform of that cell's centre; and the off-screen leg still issued |
| **AC-13** | one route leaving the view across its right edge and returning, and one leaving it over the top, each holding a leg **wholly beyond** that edge between two legs that **cross** it | the segment build | exactly the legs that meet the view, the two beyond it dropped — and an **endpoint** test, which would take three of these seven, does not produce this answer |

## Derived properties

- **P-1** The settle is a pure function of (the labels the wave left, the ordered cell), with a total
  order over candidates fixed by the ring walk and a strict comparison — no float, clock, file or
  generator on the path, and no dependence on map iteration order.
- **P-2** The query and the overlay cannot change a world: the query copies, the seam value is built
  from a copy, and neither is reachable from `Step`.

## Out of scope

- A **render-side route cache**. There is no recompute to cache: the route is state the tick already
  maintains, and an invalidation rule here would be a second opinion about staleness.
- **Re-deriving a substitute as a mover advances**, and the refresh period it would need.
- The decoded **tolerance test and its message** on a static substitute.
- Settling for the **near** search, for an **occupied** goal, and for the **optimised** mode.
- **Multi-unit distribution** of a group order: there is none to implement.
- Any change to the stall count, the give-up limit, the cost model, the window or the byte form.
- Any **further** reduction of the overlay's cost at a zoom that fits the whole map inside the
  window, where FR-9a has almost nothing left to reject. What that costs is measured and disclosed
  rather than capped: a cap there is the judgement FR-8a removed, and its failure mode is a frame
  rate that recovers as the routes are walked, not an overlay that disappears.
