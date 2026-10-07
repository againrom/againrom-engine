# Spec — a mover has a domain, and a flyer keeps its distance at rest

## Baseline

Every entity in `pkg/sim` is a ground mover. `Entity` carries no domain, no layer and no flag about
where it may go; the enterability relation asks one question of one occupancy count, so any two
entities contend for any cell; and the block plane's bit 1 — the bit that stops an air mover — is
carried through the byte form, enters the digest, and is read by nothing.

The block plane is derived by `pkg/mapload`, which sets bit 0 for impassable terrain, water,
mountain, scenery and the eight-cell border ring, and bit 1 for **the border alone**. `pkg/render`'s
`terrain.Grid.BorderCell` reads bit 1 as its margin predicate, and the dimming pass rests on the
border being that bit's only writer.

Entities are resolved **in ascending id**, and that order is the whole of the priority: two movers
converging on one cell leave it to the lower id.

A far search reads terrain alone and sweeps the whole map; a near search reads terrain and occupancy
as it stands at the mover's turn, inside a window, aimed at a sub-goal at most four cells along a
stored route. **One** far search settles: a wave that cannot label its goal takes a substitute among
the cells it did label, and the order then takes that cell as its target. The other answers no route,
which ends the order in that tick. A near failure raises a stall count, and the order is given up at
its limit. A stored route stops serving on three tests: no route, the route's last cell is not the
current target, or the sub-goal it yields is outside the window. The byte form is version 5, its
entity record 34 bytes, and a decoded route cell is refused when its blocks-ground bit is set.

## Functional requirements

### The domain

- **FR-1 — a mover has a domain, and there are exactly three.** `Entity` MUST carry a **domain**,
  one of **ground**, **ghost** and **air**, and those MUST be the values **0, 1 and 2** in that
  order. **Ground is therefore the zero value**, so an entity built without naming one is a ground
  mover and every world assembled before this field existed keeps the behaviour it had. Any other
  value MUST be refused, by the constructor and by the decoder alike, rather than clamped, masked or
  folded into one of the three.

- **FR-2 — the domain decides which terrain blocks the mover.** A cell is closed to a mover when it
  is out of bounds, or when its grid byte carries **that domain's** terrain bit: **bit 0** for a
  ground mover and **bit 1** for a ghost and for an air mover. No domain reads a bit this plane does
  not derive, and none reads both.

  In the shipped derivation bit 1 is the map border and nothing else, so this is what the three
  domains come to on real data: **water, mountains, impassable terrain and scenery block a ground
  mover and neither of the other two; the border blocks all three.** The only thing that stops a
  flyer is the edge of the map, and that is what makes a flyer worth having.

- **FR-3 — the domain decides which occupancy the mover contends on.** There MUST be **two**
  occupancy planes: a **ground** plane carrying ground and ghost movers, and an **air** plane
  carrying air movers; a mover contends only with the movers of its own plane. So a flyer and a
  ground unit share a cell freely, a ghost and a ground unit do not, and two ground units keep the
  hard collision they have, byte for byte.

### The flyer

- **FR-4 — a flyer is counted only while it is at rest.** The air plane MUST count a flying entity
  **iff it holds no target**. A moving flyer is in no plane and MUST put nothing into one — it
  neither blocks another mover nor is blocked by one, and neither adds nor removes a count as it
  advances. It enters the air plane in the tick its order ends, before the next entity is resolved,
  and is absent from it for as long as it holds an order.

- **FR-5 — a flyer's route and its step read terrain alone.** Neither search MUST consult occupancy
  for an air mover: it flies through moving and resting units alike, routing only around what FR-2
  closes to it. Ground and ghost movers MUST keep the relations they have — the far search
  terrain-only, the near search terrain and their own plane.

- **FR-6 — where a flyer's order may END is the one place occupancy reaches it.** A cell is a
  **rest** cell for a mover when FR-2 leaves it open **and** no *other* mover of its own plane is
  counted there. It MUST be asked of an **air** mover at exactly the four sites below and nowhere
  else, and of a ground or a ghost mover at **none** of them — their occupancy is resolved by the
  near search, tick by tick, and this story does not move it.

  1. **A settling search's goal test.** The wave MUST stop on its goal only when the goal is
     labelled **and** is a rest cell. A labelled goal that is *not* a rest cell MUST be treated
     exactly as an unlabelled one: the wave runs on to its frontier or its budget and then takes
     the substitute path.
  2. **A settle candidate** MUST be a rest cell as well as labelled. When no ring inside the limit
     holds a cell that is both, the search answers **no route** and the order ends in that tick —
     the outcome a search that labels nothing already has, reached without a new branch.
  3. **A stored route's staleness.** For an air mover only, a stored route MUST stop serving when
     its last cell — the mover's own target — is not a rest cell. This is one more test beside the
     three that already exist and does not alter them.
  4. **The goal test of the far search that does NOT settle.** It MUST answer **no route** for a
     goal that is not a rest cell, ending the order in that tick, and MUST NOT choose a substitute —
     the plane it builds carries costs *to* the target and says nothing about what the mover can
     reach. Without this site, test 3 would discard a route every tick and that search would
     rebuild the same one, unbounded.

  Together these MUST make the following true: **every cell a flyer's order ends on by arriving at
  its target or by settling for a substitute is a cell no other flyer was resting on at that
  entity's turn.** An order that ends *without choosing a cell* — a search that finds no route, or
  the give-up at the stall limit — leaves the flyer resting where it stood, which may be a cell
  another flyer holds; a world whose entities share a cell is advanced and not repaired, and this is
  the one way this story can produce one.

### The form

- **FR-7 — the byte form carries the domain, and the digest covers it.** The version MUST become
  **6** and every other version byte MUST be refused, this build's predecessors included. The entity
  record MUST grow by **one byte at its tail**, and nothing already in the record may move. A domain
  byte the build does not define MUST be refused, leaving the receiving world exactly as it was. A
  decoded **route cell** MUST be tested against **its own entity's domain terrain bit** rather than
  the blocks-ground bit, so a flyer's route over water reads back and a ground mover's does not. The
  digest MUST go on being taken over exactly the bytes the form produces.

- **FR-8 — nothing outside `pkg/sim` moves.** The derived block plane, its five arms and its two bits
  MUST be untouched, so **the border remains bit 1's only writer** and the margin predicate that
  reads it goes on answering exactly the border. No render, UI or loader behaviour changes, and a
  world built from a map MUST spawn every entity in the **ground** domain: the column that decides
  otherwise is not decoded here, and no other field is read as a substitute for it.

- **FR-9 — determinism is unmoved.** The domain, both planes and the rest test MUST be integer-only,
  reachable from no clock, file or generator, and independent of map iteration order. Two identical
  worlds advanced under one schedule MUST hash equal at every tick.

## Acceptance criteria (synthetic, no game install)

| # | Given | When | Then |
|---|---|---|---|
| **AC-1** | an entity built naming no domain; one naming each of the three; and a form carrying a domain byte outside them | constructed, marshalled, decoded | the unnamed one is **ground**; each named one round-trips; the out-of-range byte is refused and the receiving world is unchanged field for field |
| **AC-2** | one grid carrying bit 0 alone, one bit 1 alone, and one both — the byte a border cell derives | each domain asked to cross it | bit 0 stops **ground** only; bit 1 stops **ghost** and **air** only; both stop all three. Water and mountains do not stop a flyer and the border does |
| **AC-3** | a flyer and a ground unit ordered onto one cell; a ghost and a ground unit likewise; two ground units on crossing paths | stepped to rest | the flyer and the ground unit end **on the same cell**; the ghost and the ground unit end **distinct**; the ground pair route around each other and never overlap — hard collision unchanged |
| **AC-4** | two flyers on crossing paths, each ordered to the other's start; the same layout with two ground units | stepped to convergence | at some tick the flyers occupy the **same** cell and both still arrive, with no detour; the ground pair never share a cell |
| **AC-5** | a flyer with a **resting** flyer standing on its straight line to a free target in open terrain | stepped to rest | it flies **through** the resting flyer's cell and arrives exactly at the target — its route is the straight line and it does not go round |
| **AC-6** | two flyers ordered onto the **same** free cell, and a second fixture where they arrive on the same tick | stepped to rest | they interpenetrate while moving and end on **distinct** cells, one on the target and one beside it; at no tick are two flyers **at rest** on one cell; the lower id takes the target |
| **AC-7** | a resting flyer on a cell, and a second flyer ordered onto that cell from across the map | stepped to rest | the second settles on a **distinct** cell near it, and its own target names that cell from the first tick on — no later tick re-runs the sweep |
| **AC-8** | a flyer whose straight line to its target crosses a cell with bit 1 set | stepped to rest | it never stands on that cell at any tick, and it reaches the target by going round |
| **AC-9** | a world of one flyer holding a route across a **bit 0** cell, and one of a ground mover holding the same route | marshalled and decoded | the version byte is **6**, the record is **35** bytes, the flyer's form reads back cell for cell, and the ground mover's is refused |
| **AC-10** | two identical worlds carrying flyers, ghosts and ground units, under one schedule | stepped in lockstep to rest | equal `Hash()` at every tick, and equal byte forms |
| **AC-11** | a resting flyer re-ordered onto **its own** cell; and a flyer whose far search finds no route while it stands on another flyer's rest cell | one `Step` each | the first ends the tick at rest there and counted; the second ends its order where it stands, sharing the cell — the disclosed case of FR-6 |
| **AC-12** | a plane derived from a map, and a world built from that map | the plane censused, the world read | bit 1 is set on the border cells and on no others, the margin predicate answers true on exactly those cells, and every spawned entity is in the **ground** domain |
| **AC-13** | AC-7's fixture under the search that does not settle | stepped to rest | the second flyer's order ends in the **first** tick and it never reaches the resting flyer's cell — a refusal, not a substitute — and it does not re-run a whole-map search on every later tick |

## Derived properties

- **P-1** The rest test is a pure function of (the grid byte, the asking mover's domain, the counts
  at that cell) — no float, clock, file or generator, and no dependence on map iteration order.
- **P-2** The two relations stay **nested per domain**: a cell its domain's terrain closes is closed
  under both searches, so no route is admissible under the occupancy relation and inadmissible under
  the terrain one.
- **P-3** The byte form is injective in the domain: two worlds differing only in one entity's domain
  marshal to different bytes, so no two forms decode to one world.

## Out of scope

- **Which domain a placed unit is born in.** Its source is a column this tree cannot read, and no
  registry key is read as a proxy for it.
- **A bit-2 term in the block plane** — an object or building blocking a ghost. The plane derives no
  such bit, so a ghost here is stopped by the border alone.
- **Per-domain step costs.** Every mover keeps one flat cost model; no cost plane exists.
- **Soft collision for ground or ghost movers**, and **anchor-only overlap** for resting flyers of
  more than one cell — this tree has no footprint larger than a cell.
- **Settling** for the near search, for the search that does not settle, and for a ground mover's
  occupied goal — all three stay where the approach story left them, and FR-6's fourth site adds a
  refusal to one of them rather than a substitute.
- **A canonical resting flag.** A flyer's presence in the air plane follows from the target it
  already carries, and a second field would be a second source for one fact.
