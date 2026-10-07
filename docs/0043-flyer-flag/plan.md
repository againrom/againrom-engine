# Plan — a domain as a value, a layer as a plane, one rest predicate

## Baseline

`Entity` is ten integer and bool fields, and a pin refuses any field nobody declared. `routeScratch`
carries one `occ []int32` — a count, not a set — seeded from every non-dead entity and kept level by
`moved`. `terrainOpen` reads `blockGround` alone; `enterable` adds the count less the asker's own
presence on a position match; `open` is the one place a `relation` becomes an answer.

`searchRoute` dispatches on the mode, and only `canonicalRoute` takes a settle rule: it computes
`goalOpen` once, refuses an exact-goal search before any wave with it, gates the flat budget on it,
and breaks its loop on the goal being labelled. `optimisedRoute` has one goal test, `w.open` at its
head, and no generation budget. `settleFor` reads labels alone. `subGoal` takes an index and no
scratch, and its three staleness tests are pure world state. `clearOrder` is the one clearing site,
called from phase 1 by `clearFelled` before the scratch exists. `formatVersion` is 5, both widths are
34, and `decodeRoutes` refuses a route cell whose `blockGround` bit is set.

## Design decisions

- **DD-1 — `Domain` is a named `uint8` with a `defined()` test, shaped like `Mode`** — serves FR-1.
  `DomainGround = 0`, `DomainGhost = 1`, `DomainAir = 2`; every other byte refused by `NewWorld` and
  by `UnmarshalBinary`, never normalised. Rejected: a `Flying bool`, which cannot carry the third
  domain and whose later widening would reinterpret every stored world.

- **DD-2 — the terrain term is a mask, and one method yields it** — serves FR-2. `func (d Domain)
  blocks() byte` returns `blockGround` for ground and `blockAir` for ghost and air; `terrainOpen`
  takes the asking domain and tests that mask. Rejected: a per-domain predicate function value —
  `route.go` refuses a second behaviour-steering identity beside the mode byte.

- **DD-3 — occupancy is ONE slice of `2 x cells` counts, layer-major** — serves FR-3. `func (d
  Domain) layer() int` is 1 for air and 0 for ground and ghost, and a cell's slot is `layer*cells+i`
  behind one reader and one writer. The `cells` term has **one source**: a field the constructor sets
  from `gridCells` — not `len(s.plane)` at one site and the bounds at another, which is how two
  layers come to be indexed by two notions of the count. Rejected: two slices, whose two lengths and
  two resets are that drift with an allocation on top.

- **DD-4 — one predicate says who is counted, and the seed is its only loop** — serves FR-4.
  `counted(e)` is `!e.Dead() && (e.Domain != DomainAir || !e.HasTarget)`, and `occupy` walks the
  entities through it. A named function and not an inline condition because **two other sites
  subtract a presence it decides** (DD-6, DD-7). The walk is in the seed for the reason a corpse's
  is: a predicate is asked once per neighbour of every frontier cell.

- **DD-5 — `moved` is called only for a mover its plane counts while it moves** — serves FR-4. The
  advance asks DD-4's predicate; an air mover commits its position and touches no count. Rejected:
  calling `moved` unconditionally and letting the air counts go negative — a count would then be a
  signed residue rather than a presence, and `n == 0` would stop meaning what it says.

- **DD-6 — a second clearing site, `restAt`, takes the scratch** — serves FR-4. It clears the order
  and counts the entity at its cell when clearing is what put it in a plane, so a flyer becomes
  visible in the tick its order ends, before the next entity resolves. Its test is "was it counted
  before, is it counted now" and not "is it a flyer" — DD-4's predicate again, so it cannot drift
  from the seed. The four clearing sites in the move loop call it; `clearFelled` keeps plain
  `clearOrder`, phase 1 running before the scratch exists. Rejected: giving `clearOrder` the scratch
  and a nil from phase 1 — a nil-checked parameter is a branch nobody can see.

- **DD-7 — the rest question is a second answer, not a third relation** — serves FR-5, FR-6.
  `relation` stays two-valued; `restFree(s, self, x, y)` is a new predicate that answers **true for
  a ground or a ghost mover** and, for an air mover, the air count at that cell being zero once the
  asker's own presence is taken off. **That subtraction is conditioned on `counted(self)` (DD-4),
  not on a position match.** Every site asking it is reached with the asker holding a target, so an
  air asker is never in the plane, and the unconditional shape `enterable` uses — right *for a
  ground mover, which always is counted* — would take off a presence never seeded, drive the count
  to −1 on the asker's own cell and stop that cell being a rest cell. `settleFor` is explicit that
  the start cell is a legal answer meaning *it can get no nearer*, so filtering it out would send a
  flyer further from its target than it stands. `enterable`'s subtraction goes behind the same
  predicate. Rejected: a third `relation` constant, which both searches would honour at every offer
  — precisely the fly-through FR-5 forbids.

- **DD-8 — the air mover's near search is handed `terrainRelation`** — serves FR-5. The relation is
  chosen at the near call site from the mover's domain; no parameter is added and neither search
  learns a domain. The far call site already passes terrain.

- **DD-9 — `canonicalRoute` gains one term, computed once** — serves FR-6. `goalOpen` becomes "open
  under `r` **and**, when this search may settle, a rest cell", and the loop breaks on `labelled &&
  (start == target || settle == exactGoal || goalOpen)`. For ground and ghost `restFree` is true, so
  both expressions are what they are today — a cell carries a label only if it is the start or
  passed `open`, and the far search runs unwindowed, so `labelled` already implied `goalOpen`. This
  is the whole of "a labelled goal that is not a rest cell is treated as unlabelled".

- **DD-10 — `settleFor` takes `self` and filters candidates by `restFree`** — serves FR-6. The ring
  order, the strict accept and the growth bound are untouched, so the tie-break and the bound stay
  the approach story's and only membership narrows. A ring whose labelled cells are none of them rest
  cells does **not** stop the growth: the search goes to the next, and answers no route at the bound.

- **DD-11 — the fourth staleness test goes in `subGoal`, which therefore takes the scratch** —
  serves FR-6. Gated on the air domain and placed **after** the last-cell test and **before** the
  window test, so it is asked only of a route still ending at the mover's target. The signature
  change reaches one test caller, updated rather than routed around. What it costs: the three
  existing tests are pure world state and this one is **mid-tick** state. Determinism survives — the
  walk is in ascending id as the plane's updates are — but `subGoal` stops being a function of the
  world alone.

- **DD-16 — the far search that does not settle gets the rest term on its ONE goal test** — serves
  FR-6, and it is the decision without which DD-11 has no remedy behind it. `optimisedRoute` refuses
  a target not open under its relation before it floods; the rest term joins that refusal, so a
  flyer ordered onto a resting flyer's cell is answered **no route** and its order ends in that tick.
  Without it, DD-11 would discard the route every tick and that search rebuild the same one —
  and it has **no generation budget**, its only bound a window the far call site never passes, so
  the cost is a whole-map flood per tick for the walk's length. Rejected: gating DD-11 on the mode,
  which leaves that mode no distinctness at all.

- **DD-12 — the record grows one byte at its tail** — serves FR-7. `formatVersion` 5 to 6,
  `entityLen` 34 to 35, the domain at record offset +34. At the tail because inserting it earlier
  moves eight offsets to no purpose, which is why the health pair sits there.

- **DD-13 — `decodeRoutes` tests a route cell against its own entity's mask** — serves FR-7. The
  domain is decoded before the route is read, so the test is `grid[idx] & ents[i].Domain.blocks()`;
  the other four route refusals are untouched.

- **DD-14 — no PRODUCTION file outside `pkg/sim` changes** — serves FR-8. `pkg/mapload`'s derivation
  keeps its five arms and two bits, so the border stays bit 1's only writer; `FromALM` builds its
  entity literal by field name and names no domain, so it yields ground movers by DD-1 with no line
  added. **One test file outside the package moves**: `pkg/mapload/gridform_test.go` pins a digest
  and a form length for a map-built world, which the version and record width put out of date. It is
  re-derived outside the tree like the pins inside, and is where FR-8's "every map-built entity is
  ground" is pinned as a byte.

- **DD-15 — every pin moves deliberately, one literal at a time, and none is deleted** — serves
  FR-7, FR-9. The canonical field table gains a `Domain` row, which is the declaration that the
  field is canonical; the version literal goes 5 to 6; the pinned digests and byte strings are
  recomputed from the new bytes rather than read off the encoder. **Exactly one of the two 34s
  moves**: the pins spell out both widths as literals on purpose, and the two appear mixed inside
  single expressions (`34 + cells + 34*units + 4*units`) and as absolute offsets. The header stays 34
  and only the record becomes 35, so each site is re-derived by hand. A search-and-replace yields a
  pin agreeing with the encoder for the wrong reason, the one failure a hand-written pin prevents.

## Risks

- **R-1 — a second writer of bit 1 would silently redefine the drawn margin.** The margin predicate
  reads bit 1 and the dimming pass calls every such cell non-playable, so a derivation setting it for
  a second reason would dim live map. This story adds no writer (DD-14); SC-6 measures that.
- **R-2 — a settling air search whose goal is occupied runs its whole budget.** By DD-9 it falls to
  the computed `max(5, D>>2) + D`, as a settling ground search does, so the cost is bounded by the
  distance ordered. **That bound is the settling search's alone** — the other has no generation
  budget, which is why DD-16 ends its order rather than letting it re-search.
- **R-3 — the disclosed overlap.** A flyer whose order ends without choosing a cell rests where it
  stands, which may be another flyer's cell. It needs a mover terrain-sealed in the air domain, which
  on a derived plane means sealed by the border, so it is unreachable from a map. It is in the
  contract and witnessed rather than left to be found.

## Success criteria

- **SC-1** (FR-1, FR-7) — the form's version byte, total length and record width measured against
  hand-written numbers, so the encoder is not asserted against itself; a domain byte outside the
  three refused, the receiving world unchanged field for field.
- **SC-2** (FR-2) — each domain crossed against each of the three derivable grid bytes: nine
  answers, none inferred from another's.
- **SC-3** (FR-3) — three pairings asserted separately: flyer/ground, ghost/ground, ground/ground.
- **SC-4** (FR-4, FR-5) — two moving flyers share a cell at some tick and both arrive; one crosses a
  resting flyer's cell on a straight line and lands on its target.
- **SC-5** (FR-6) — two flyers converging on one cell end distinct, including where they arrive on
  the same tick; one ordered onto a resting flyer's cell settles beside it, its target naming the
  substitute from the first tick; and a settle onto the mover's own cell stays reachable.
- **SC-5a** (FR-6) — the same fixture under the search that does not settle: the order ends in the
  **first** tick, the flyer never reaches the peer's cell, no whole-map search runs per tick.
- **SC-6** (FR-8) — a census of a derived plane: bit 1 on the border cells and no others, the margin
  predicate true on exactly those, every map-built entity ground, reported as counts.
- **SC-7** (FR-9) — two identical worlds carrying all three domains hash equal every tick and
  marshal alike.
- **SC-8** (FR-2, FR-4, FR-5, FR-6, FR-7) — nine mutants, one at a time on production code, the whole
  tree run, the failing tests named, each reverted: the ghost's mask set to bit 0; the air seed made
  unconditional; the near search handed the unit relation for air; `moved` called for an air mover;
  `restAt` reduced to a plain order-clear; the settle candidate filter dropped; the fourth staleness
  test removed; the non-settling search's rest term dropped; `decodeRoutes` holding every domain to
  bit 0. **The two bookkeeping mutants belong to the rest test's slice, not the seed's**: until
  something READS the air plane a corrupted count changes no outcome, and a kill claimed where the
  plane has no reader is claimed against nothing.

## Traceability

FR-1 → DD-1 → SC-1 · FR-2 → DD-2 → SC-2, SC-8 · FR-3 → DD-3 → SC-3 · FR-4 → DD-4, DD-5, DD-6 →
SC-4, SC-8 · FR-5 → DD-7, DD-8 → SC-4, SC-8 · FR-6 → DD-7, DD-9, DD-10, DD-11, DD-16 → SC-5, SC-5a,
SC-8 · FR-7 → DD-12, DD-13, DD-15 → SC-1, SC-8 · FR-8 → DD-14 → SC-6 · FR-9 → DD-15 → SC-7
