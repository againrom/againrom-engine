# Spec — the ground carries a cost, and the mover pays it

**Intensity: spec-anchored / static. Terrain: brownfield** for the route search, the rate call site
and the canonical byte form — all three already ship the behaviour this changes — and **greenfield**
for the two new map derivations.

A world in this build stands on one per-cell plane: which cells are closed to which movers. Every
cell it leaves open is identical to every other — a road and a stretch of sand are the same ground to
cross, and a hillside costs what level ground costs both ways. The map says otherwise, cell by cell,
and nothing reads it.

## Context

A world gains two more planes here — what each cell costs to enter and how high it stands — derived
from the map and read by the two things built to consume them: the search's per-step cost, and the
rate law's divisor and its slope tilt. The rate law is already whole and is handed four zeros; the
search's one flat step cost is what the NON-ground arms charge, applied to every mover.

## Functional requirements

- **FR-1 — a world carries a cost plane and a height plane.** Each is one byte per in-bounds cell,
  row-major from (0,0), exactly as long as the block plane, and each is **copied** into the world.
  Both are canonical: carried by the byte form, entering the digest, read while a world is advanced.
  Either may be left unnamed; a named plane of any other length is **refused**, as a block plane of
  the wrong length already is. An unnamed cost plane is materialised at the **uniform cost 8**, an
  unnamed height plane at **zero**. No byte value is refused in either, and no flag records whether a
  plane was named: a world with an all-8 cost plane and one with none are one world.

- **FR-1a — a world naming neither plane behaves exactly as it did before either existed**, tick for
  tick, in every position, route, transit pair and stall count. It holds only while **no rule
  compares a search label against an absolute constant, adds one to a non-label term, or compares
  labels from two different searches** — every budget, cap and limit counting generations, ticks or
  cells instead. A uniform plane scales every label by a constant factor, which orders every pair of
  routes as the flat constant did; the factor is exact only because the default cost is **even**.

- **FR-2 — the search charges a ground mover the entered cell's own cost.** A step costs, for a mover
  in the **ground** domain, the cost byte of the cell being **entered** orthogonally, and that byte
  plus half of it truncated diagonally. For **any other** domain it is a flat 2 and 3, and no cost
  byte is read. At a cost byte of 1 the two arms are equal, and at 0 both are free.
  **Cost changes labels, not reach.** The wave advances one ring per generation whatever the ground
  costs and stops in the first generation that labels the goal, so the SET of cells a search labels is
  the same under every cost plane and only their labels differ. Cost chooses among the routes the wave
  reached; a cheaper one outside them is not found. That is the reconstruction's behaviour, not a
  defect.

- **FR-3 — a route read out of a completed search charges the same four arms at the cell each step
  ENTERS**, whichever way the walk runs: over labels measuring cost FROM the mover it runs backwards
  and the entered cell is the one being left, over labels measuring cost TO the goal it runs forwards
  and it is the one being taken. Scan order and the asymmetric accept are unchanged. **A walk is
  bounded**: one taking more steps than the world has in-bounds cells has revisited a cell and yields
  no route. It cannot fire while every cost on the walk is positive, the label then falling strictly.

- **FR-4 — the rate is composed from the two cells the transit joins.** At a transit's start the rate
  law receives the cost and height bytes of the cell left and the cell taken, and the law itself does
  not change: the multiplier, the saturating tilt, the **byte-wide** cost sum — which may wrap, and
  whose wrap to zero takes the substitute — the clamp, the diagonal constant and the round-up. **If
  either cell lies outside the bounds the law receives four zeros**, which is what it received before
  either plane existed.

- **FR-5 — the cost plane a decoded map describes.** A function of the map's tile plane alone — not
  its overlay, altitudes, records or structures — and total: one byte per cell, no error, a short tile
  plane reading as the word zero past its end and a long one ignored past the extent. For tile word
  `w`, over `i = w & 0x3ff`, the answer is a terrain class and a cost:

  1. **Water is taken first and taken whole.** If `i & 0x300` is `0x200`, a low nibble of 8 or more is
     a **reject**; a low nibble of exactly 4 with `i & 0x30` equal to `0x10` is class Land; anything
     else is class Water. Both classes cost **8** here whatever the scalars say — the water scalar is
     never consulted on a water cell.
  2. Otherwise `s = i & 0xf`, `b = (i >> 4) & 3`, `g = (i >> 6) & 0xf`. An `s` of 14 or more is a
     **reject**.
  3. `g` names a pair of terrain classes, `(primary, secondary)`:

     | g | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 12 |
     |---|---|---|---|---|---|---|---|---|----|
     | primary | Grass | Cracked | Sand | Savanna | Stones | Cracked | Flowers | Mountain | Road |
     | secondary | Land | Land | Land | Land | Land | Stones | Savanna | Stones | Land |

     `g` in 8-11 cannot be reached — those words are exactly the water range. `g` of **13, 14 or 15
     is a reject** (see *Divergences*).
  4. `(b, s)` names a blend level in 1-5:

     | b | s = 0 .. 13 |
     |---|---|
     | 0 | 2 3 2 4 3 4 2 2 2 2 4 4 4 4 |
     | 1 | 3 5 3 3 1 3 2 4 2 2 4 2 4 4 |
     | 2 | 2 3 2 4 3 4 2 4 2 2 4 2 4 4 |
     | 3 | 5 5 5 5 5 5 2 2 2 2 4 4 4 4 |

  5. The class is the **secondary** at level 1 or 2 and the **primary** at 3, 4 or 5. The cost is
     **not** that class's scalar: it blends the pair by level — `sec`, `(3*sec + pri) >> 2`,
     `(sec + pri) >> 1`, `(3*pri + sec) >> 2`, `pri` for levels 1 to 5, coinciding with the class's
     own scalar only at 1 and 5. Both shifts truncate. The ten scalars are

     | class | Land | Grass | Flowers | Sand | Cracked | Stones | Savanna | Mountain | Water | Road |
     |---|---|---|---|---|---|---|---|---|---|---|
     | cost | 8 | 8 | 8 | 14 | 6 | 12 | 8 | 16 | 8 | 6 |

  6. A **reject** has no class and costs **255**.

- **FR-6 — the height plane a decoded map describes** is the map's own altitude plane, byte for byte
  and cell for cell, unscaled and unshifted, zero past a short one and ignored past a long one.

- **FR-7 — every world built from a map is built over all three planes**, on every path: with a
  definition table and without, with a party and without, with a mission script and without —
  including the two that REBUILD a world from an earlier one, which re-derive all three from the map
  exactly as they already re-derive the block plane.

- **FR-8 — the block plane does not move.** For every map of both shipped roots the derived block
  plane is byte-identical to today's and its per-arm census unchanged count for count; and for
  **every** tile word, carried by a map or not, the cell blocks a ground mover exactly where it does
  today.

- **FR-9 — the byte form takes a new version and the old is refused**, naming both, with no migration.
  Two worlds differing only in a cost or height byte have different forms and digests.

## Acceptance criteria

| # | GIVEN | WHEN | THEN |
|---|---|---|---|
| AC-1 | a world naming neither new plane | it is marshalled | the form carries an all-8 cost plane and an all-zero height plane, each of the block plane's length (FR-1) |
| AC-1a | a named plane one byte shorter or longer than the block plane | a world is built, and a form of that shape decoded | both are refused (FR-1) |
| AC-2 | in the canonical mode, two corridors of EQUAL cell length between one pair of cells, one of cost 16 and one of cost 6 | a ground mover is ordered along them | it takes the cost-6 corridor, and takes the other one when the plane is uniform (FR-2) |
| AC-2a | one world and one order, over a uniform cost plane and a varied one | a canonical search runs over each | the same set of cells is labelled and their labels differ: cost moves numbers, not reach (FR-2) |
| AC-3 | a world naming neither plane, and mission 10 of both roots under the mission driver | each is advanced | it behaves as the pre-story build tick for tick and the decided tick is unmoved, the oracle being the pre-story digest reproduced by cutting the two planes out of the new form and restoring the old version byte (FR-1a) |
| AC-4 | AC-2's world | a ghost mover is ordered along the same pair | it takes the same corridor it takes over a uniform plane: no cost byte reaches it (FR-2) |
| AC-5 | destination cost bytes of 1, 0 and 255 | a diagonal and an orthogonal step into each are costed | 1 and 1; 0 and 0; 255 and 382 (FR-2) |
| AC-6 | a completed canonical search and a completed optimised one over one non-uniform plane | each route is read out | each step is charged at the cell it ENTERS, and the canonical tie-break is what it is today (FR-3) |
| AC-6a | an all-zero cost plane and a labelled region of more than two cells | a route is read out | the walk terminates and yields no route rather than cycling (FR-3) |
| AC-7 | two cells of cost 6 and two of cost 16, at equal height | a transit starts on each | the cheap pair takes fewer ticks (FR-4) |
| AC-8 | two adjacent cells whose heights differ by more than 32, crossed each way | a transit starts | uphill takes more ticks than downhill, and both equal what a difference of exactly 32 gives (FR-4) |
| AC-8a | a transit one of whose cells lies outside the bounds, on a world naming neither plane | a transit starts | its length is what the pre-story build gave (FR-4) |
| AC-9 | tile words for the water range; group 7 at level 5; group 7 at level 2; group 5 at level 3; a low nibble of 14; a low nibble of 9 inside the water range; group 13 | each is classified | 8, 16, 13, 9, 255, 255, 255 (FR-5) |
| AC-10 | a map whose altitude plane is short, and one whose tile plane is | each plane is derived | altitudes then zeros, and the missing words classified as the word zero, both at the extent's length (FR-5, FR-6) |
| AC-11 | each shipped map of both roots | its block plane and per-arm census are derived | both identical to the pre-story build's, map for map and count for count (FR-8) |
| AC-11a | every one of the 65 536 tile words | the block byte is derived | it is what the pre-story build derives for that word (FR-8) |
| AC-12 | a byte form of the previous version | it is decoded | refused, naming both versions; a current form round-trips to an equal world and digest (FR-9) |
| AC-13 | two worlds alike but for one cost byte, and two alike but for one height byte | each pair is hashed | each pair's digests differ (FR-1, FR-9) |
| AC-14 | a map with a non-uniform cost plane, through each world-building path — table and none, party and none, script and none | a world is built | its cost and height planes are the map's, not the defaults (FR-7) |
| AC-15 | mission 10 of both roots, driven to its outcome with the loaders wired | it is run | the decided tick is reported, whether or not it moved (FR-7) |
| AC-16 | one shipped map, one ground mover, one order | it is routed over the derived plane and over a uniform one | the routes differ on at least one shipped map: the derived plane reaches a real search (FR-2, FR-5, FR-7) |

## Divergences

**Strip groups 13, 14 and 15 are rejected, and that is ours.** In the original those three reach a
table never written for them and read whatever memory is there — an answer not defined, not
reproducible and not the same twice. No shipped cell names one. A reject keeps the derivation a
total, deterministic function of the tile word.

**The ten class scalars are compiled in.** They are per-map parameters in the original, in a registry
section this build has no reader for; the values in FR-5 are the ones both roots carry.

**FR-3's walk bound is ours.** The original discards a walk longer than 1000 steps; this build bounds
by the cell count, unreachable wherever the walk's costs are positive. The decoded 1000 is not
reproduced.

## Out of scope

- **Passability**: no cost, height or slope narrows any mover's verdict. FR-8 states it.
- **The cost byte's runtime mutation** and the per-cell records that would carry it — neither exists
  here, and neither reaches a path cost.
- **A structure changing a cell's cost**: a building rebuilds block bytes and restores the baseline.
- **Reading the class scalars, the rate multiplier or the pathfinding scalars from a registry.**
- **The AI, formations, group orders**; the group rate term is read exactly as today.
- **Rendering**: no plane here is drawn, and the render tier's split is not reconciled against this
  one.

**Error cases.** One is added, and it is a length: a named plane neither empty nor exactly one byte
per in-bounds cell is refused, at construction and decode alike (AC-1a). Both derivations are total
and both planes accept every byte value, so there is no other.
