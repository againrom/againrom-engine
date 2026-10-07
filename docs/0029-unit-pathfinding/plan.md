# Plan — routing over a static grid, in two modes

## Baseline

`Step` makes one label plane per tick and resolves each unit against `enterable(self, x, y)`, which
LINEAR-SCANS `w.entities` on every call. The wave puts it to all eight neighbours of every frontier
cell, so a tick costs units × cells × 8 × units, superlinear in the group size — the frame rate the
owner watched collapse on a group order. A search that cannot succeed is not cheaper: it spends the
whole budget first, and sixteen of those is what a give-up costs.

## Design decisions

### DD-1 — one label plane, read where it stands; the frontier is what is buffered twice

One label plane, two frontier slices. Relaxation reads and writes that plane directly, so a cell
lowered earlier in a generation is read at its new value by every frontier cell after it —
FR-5's sequential rule as a shape, not a discipline. `MOVE-SEARCH-001` (High) is exact on
the asymmetry: one label plane, two frontier lists, the double buffer being the frontier's.

Rejected: **the batched wave** — snapshot the labels, write a fresh plane, test each candidate
against the snapshot. It is deterministic and passes all ten acceptance criteria, yet it is a
different contract: choosing it would move every digest a route reaches. SC-9 is the grid that says
so (FR-4, FR-5).

### DD-2 — the frontier repeats, is walked front to back, and scans FR-5's own extraction order

A neighbour is appended on **each** improvement, so a generation may hold a cell twice; the walk is
insertion order; neighbours are scanned `dx = -1, 0, +1` outer, `dy = -1, 0, +1` inner. The append
form is `MOVE-SEARCH-001`'s. The spec fixes a scan order for the backwards walk and none for the
forward pass, yet the forward order decides which of two frontier cells reaches a shared neighbour
last; reusing the extraction's leaves one order in the file, not two.

Rejected: **a deduplicating frontier**, **a reversed walk**, **any other scan order** — measured
first: none changes a route in 148k random 5x5 or 33k random 9x9 searches, and the reversed scan
changes one twice in 12305 dense 16x16 ones. All three are fixed for reproducibility, and **no
criterion claims to discriminate them**; those three figures are owed to `verification.md` (FR-5).

### DD-3 — the plane is scratch of the tick, holds `cost + 1`, and resets by its own touch list

`[]uint64`, one slot per in-bounds cell, zero meaning unlabelled — so a fresh plane needs no fill and
a reset zeroes only the cells it recorded touching. It is **a local of `Step`**, made once
per tick and passed into the search, so `World` gains no field, `encode` cannot reach it, and
per-worker scratch later is a parameter change. `uint64` retires the overflow argument a narrower
label needs over a budget derived from an unclamped target.

Rejected: **a per-world cache** — state beside the digest, and one more thing to keep true across an
`UnmarshalBinary` that replaces the whole world. Rejected: **clearing the plane per search**, `W*H`
writes per unit per tick (FR-4).

### DD-4 — the start cell is seeded outside the plane, and the predicate takes an identity

FR-3 lets a search begin on a blocked or out-of-bounds cell, which has no plane slot. So the seed
frontier carries the start cell itself, its slot written only when it is in bounds, and the
backwards walk ends by comparing coordinates with the start, not by reading a zero out of the plane.
Enterability now takes the resolving unit's index: the flood asks about every cell inside the budget,
its own included, where 0026's predicate could take none, the one cell ever put to it being
unreachable.

Rejected: **clamping the start into bounds** — it deletes FR-3's "can still be routed off it".
Rejected: **leaning on "the start can never improve below 0"** — true today, but it makes a stated
rule a consequence of arithmetic a cost plane could falsify (FR-3).

### DD-5 — both searches are plain functions in `pkg/sim`, in no sub-package and behind no interface

`LoadSimSources` reads `pkg/sim`'s own directory and skips subdirectories, so a `pkg/sim/route`
package would satisfy the fail-closed DAG check and **silently escape the behavioural half of the
determinism wall**. Rejected for that reason. Rejected: **a `Pathfinder` interface value on the
world** — a second behaviour-steering identity beside the mode byte, one inside the digest and one
outside, which is the fault FR-2 exists to prevent (FR-8).

### DD-6 — one constructor; `Mode` a defined byte with canonical **0**; the absent grid materialised

`NewWorld(seed, bounds, mode, grid, ents)` — both new inputs positional, so the compiler finds every
call site. Canonical is 0: a zero `Mode` is what a caller who says
nothing gets, and that must be the reconstruction rather than our own invention — fidelity the
default, quality opt-in. Optimised is 1; FR-2 refuses every other byte. An absent grid becomes an
all-zero grid of `W*H` bytes **at construction**, so FR-1's all-zero equivalence and FR-2's identical
byte forms are one representation, not two kept in agreement; the count is `W*H` in `int64`,
zero when either bound is not positive, and any other length or any cell with a bit outside the low
two is refused.

Rejected: **optimised = 0** — our own routing as the silent default. **A second constructor or an
options struct** — both let a caller omit the mode, which must never be defaulted by accident
(FR-1, FR-2, FR-5, FR-6, FR-8).

### DD-7 — the byte form goes to version 3, and the grid sits in the header

Header 29 → 34: the mode byte at +29, the grid's cell count at +30, then that many cell bytes; the
record grows to 26 with the stall byte at +25. The grid goes **before** the records because it is the
only variable-length section: put after them, the decode would have to trust the declared entity
count to find the records, where today it comes from dividing what is left. Rejected for
that reason (FR-1, FR-2).

### DD-8 — optimised mode is a Dijkstra from the target, and the tie-break is a greedy walk forward

Costs to the target are computed outward from it with a binary heap over the region's enterable
cells; a step's cost depends only on its diagonality and the corner condition is symmetric in its two
cells, so the reverse graph is the forward one. The start's own distance is the minimum over its
legal first steps, which routes a unit off a cell it may not stand on. The route is built
forward: at each cell take, among the neighbours whose step keeps the total at the minimum, the
smallest `(y, x)`. Sequences are compared from the start, so the first difference decides and greedy
is exact; no route is enumerated. The region is clipped from an `int64` rectangle, since a
target may name any cell.

Rejected: **enumerating minimum-cost routes and sorting them** — exponential, for a contract naming one. **Predecessor links from a forward search** — a predecessor cannot express an order over
the whole forward sequence. Distances do not depend on the heap's pop order among equal costs, so the
key's `(y, x)` terms are inert, and SC-10 discharges that by measurement (FR-6).

### DD-9 — the stall count is a byte on `Entity`, and giving up fires inside the tick that reaches 16

Incremented on a tick whose search returns no route; if it then equals 16 the target is cleared and
the count zeroed in the same tick, so a stored count is always 0–15 and FR-2's refusal cannot fire
falsely. Any advance resets it, and a unit with no target holds zero. Rejected: **a
parallel structure keyed by id** — a second representation `UnmarshalBinary` must rebuild.
Rejected: **firing on the following tick** — AC-2 pins the clear to the 16th (FR-7).

### DD-10 — occupancy is a plane of the tick's scratch, moved as each unit advances

`enterable` answers its occupancy half from a count per in-bounds cell — free to the asking unit when
that count, less one where the unit stands there itself, is zero. A count, not a set: a world whose
units already share a cell is advanced rather than repaired. The plane sits on the tick's scratch
beside the label plane, since on the world it is the state `TestTheCanonicalWorldsFieldSetsArePinned`
refuses by name — and rightly, that pin holding whether or not a field moves a byte, a digest or an
outcome. It is not re-pinned here.

It is **not** a snapshot read all tick. The predicate answers over the entities as they stand when it
is called and `Step` rests on that — at index i the units before i hold post-move cells and those
after pre-move ones — so the counts move in the statement that moves a unit. A snapshot of the tick's
top would change contention outcomes and the digest.

Rejected: **recounting per search** — no drift, but an O(units) pass back inside the loop this
revision exists to empty (FR-3, FR-8).

### DD-11 — a destination no route may enter is refused before the wave, not after it

The target is labelled if and only if it is the start or is enterable and inside the budget, so
`canonicalRoute` tests its enterability once before the seed and answers no route where that fails —
the same answer the spent sweep reaches, at the same point, raising the stall by the same rule, with
start-equals-target excused as the empty route. An identity and not an approximation, which is why it
is admissible at all: a search that could not have succeeded cannot be made to succeed by being cut
short. Optimised mode refuses on this ground already (FR-5, FR-7).

## Risks

- **R-1** Both the grid in every byte form and digest (65536 bytes on a 256x256 map) and the label
  plane made once per tick (512 KB there, reused by every unit of that tick) grow with the map, not
  with the unit count.
- **R-2** Canonical's budget makes a long detour fail, so a unit can refuse an order it could
  physically walk, stall, and give up. The original substitutes a nearby destination there and we do
  not, so a legitimate order can read as a bug in a build the owner runs.
- **R-3** Both searches run for every unit holding a target, every tick; caching and staggering are
  out of scope.

## Success criteria

- **SC-1** AC-6 and AC-7 hold in full; every offset and width re-pinned at version 3, the digest pin
  recomputed from the pinned bytes, not from a run of this package, and the no-grid/all-zero
  pair compared byte for byte as well as by digest (FR-1, FR-2).
- **SC-2** AC-8 holds in full, each refusal its own case; P-5 checked by comparing the receiving
  world's whole byte form before and after each, not its digest alone (FR-1, FR-2).
- **SC-3** AC-1 and AC-3 hold in full in **both** modes, AC-1's wall placing its gap so the detour
  fits canonical's `max(5, D >> 2) + D` generations **and** optimised's 8-cell region — outside
  either, that mode finds no route and witnesses give-up, not arrival. P-1 checked over every unit
  after every tick (FR-3, FR-4).
- **SC-4** AC-2 holds in full, all three target fields compared individually at each of the 16 ticks;
  P-2 and P-4 checked over that run, and a unit that stalls then advances is seen returning its count
  to zero (FR-7).
- **SC-5** AC-4 holds in full in both modes; and a world marshalled at every tick and decoded into a
  second world steps on to the same digests, so routing state outside the byte form shows as a
  divergence, not a passing pin. P-3 sampled over that pair (FR-8).
- **SC-6** AC-5 holds in full, one world driving both modes (FR-5, FR-6).
- **SC-7** AC-9 holds in full, its minimum from an enumeration written against FR-6's words and never
  from the solver under test. Its canonical-worse half needs a **purpose-built** grid — over ~51k
  random walked small grids the canonical cost never once exceeded the admissible minimum — so it is
  a forced diagonal zigzag racing a longer orthogonal corridor, long enough to be past the crossover,
  which sits between 6 and 12 zigzag steps: at 12 the canonical walk costs 36 against a minimum of
  32 (FR-4, FR-5, FR-6).
- **SC-8** AC-10 holds in full in both modes (FR-4, FR-5, FR-6).
- **SC-9** **The relaxation order is discriminated, not asserted.** Canonical mode, a 5x5 grid with
  the blocks-ground bit set at `(1,1)` alone, one unit at `(3,2)` ordered to `(0,1)`: generation 1
  labels `(2,2)` at 2 and `(2,3)` at 3, and both are frontier cells neighbouring `(1,2)`. Relaxed in
  frontier order, `(2,3)`'s diagonal offer of 6 is refused because `(2,2)` has already written 4
  where FR-5 says to read it, so `(1,2)` is **4**, the target 7, the route `(2,2) (1,2) (0,1)`. A
  batched wave writes all three offers into a fresh plane, `(2,3)`'s lands last, and `(1,2)` is
  **6**, the target 9, the route `(2,1) (1,0) (0,1)` — the unit stands on `(2,1)` after one tick
  instead of `(2,2)`, so the byte form and the digest differ at tick 1. The route is pinned cell by
  cell and the tick-1 digest with it, and SC-10's batched mutant fails here (FR-4, FR-5, FR-8).
- **SC-10** Nine mutants, each applied to production code, run over the whole tree with its failing
  tests named, and reverted: DD-1's batched wave; **the hybrid wave**, its other second-plane shape —
  a stale source label with the minimum kept in the new plane; the extraction's straight accept
  narrowed to `<`; its diagonal accept widened to `<=`; the corner-cut refusal deleted; the region's
  growth set to 0; the budget's `max(5, D >> 2)` term dropped; the stall reset on advance deleted;
  the absent grid encoded as zero cells. SC-9's grid kills the batched wave alone — the hybrid walks
  the contract's own route there — so the hybrid gets its own: 8x8 blocking `(3,2) (4,2) (1,3)
  (2,3)`, a unit at `(0,7)` ordered to `(4,1)`, where both routes cost 18 yet part at the **first**
  step, `(0,6)` against `(1,6)`, so the digest moves at tick 1 there too; that grid kills the batched
  wave as well. A tenth runs as a **deliberate survivor** — the heap key's `(y, x)` terms dropped —
  which is DD-8's inertness discharged as a measurement (FR-1, FR-4, FR-5, FR-6, FR-7).
- **SC-11** **DD-10 and DD-11 are measured against the tree they replace, never asserted.** The linear
  scan is kept as a **test-only reference predicate** and the indexed one compared with it unit for
  unit and cell for cell, over a randomised corpus of worlds and of **mid-tick** states — units walked
  one at a time, so what a half-resolved tick answers is inside the comparison. Three runs recorded
  from the tree **before** either change — units crossing a wall in each mode, a group ordered onto a
  held cell, both past `stallLimit` — are pinned digest by digest and reproduced tick for tick, each
  asserted to hold an advance, a hold and a give-up. DD-11's witness is that nothing was labelled: a
  refusal leaves the touch list and the frontier empty where a spent sweep leaves both full. Two
  mutants run tree-wide and revert — the occupancy move deleted, the early-out removed — the second
  alters no behaviour, so only that witness kills it (FR-3, FR-5, FR-7, FR-8).
- **SC-12** **The cost is a figure this repo reproduces.** One committed harness advances a group
  ordered to a far cell over two map sizes and five group sizes, and eighty movers ordered onto a free
  cell nothing can reach, in milliseconds per tick. It lands **before** either fix and runs at each,
  so a contribution is separable and both halves come from one piece of code (FR-4).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-6, DD-7 | SC-1, SC-2 |
| FR-2 | DD-6, DD-7 | SC-1, SC-2 |
| FR-3 | DD-4, DD-10 | SC-3, SC-11 |
| FR-4 | DD-1, DD-3 | SC-3, SC-7, SC-8, SC-9, SC-12 |
| FR-5 | DD-1, DD-2, DD-6, DD-11 | SC-6, SC-7, SC-8, SC-9, SC-11 |
| FR-6 | DD-6, DD-8 | SC-6, SC-7, SC-8 |
| FR-7 | DD-9, DD-11 | SC-4, SC-11 |
| FR-8 | DD-5, DD-6, DD-10 | SC-5, SC-9, SC-11 |
