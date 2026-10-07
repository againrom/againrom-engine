# Spec — unit pathfinding over a static passability grid

## Problem and current behaviour

`pkg/sim` advances one tick per step: every command in slice order sets its unit's target, then each
unit holding one is resolved in ascending id against the cell one step nearer on each axis (`+1`, `-1`
or `0` by the sign of the difference), moving there only if no other unit stands on it at that moment
and on neither axis otherwise. Resolution is incremental, so a unit sees the moves already made this
tick, and a unit ending a tick on its target has it cleared with no residue.

**The world knows nothing about the map**: advancement reads no bounds field, no terrain and no placed
object, so units cross water, mountains and buildings and walk off the grid onto negative coordinates.
**And a blocked unit never goes around** - it keeps its cell and its target while the obstacle stands.
Every unit is one cell: no wider footprint, no ground/air split, no speed, no sub-cell position, and
every unit contends with every other for a cell.

The canonical byte form is at version 2 - a header, then one fixed-width record per unit: id, cell,
target cell, target-presence flag, class - and the digest is taken over exactly its bytes. Advancement
is integer-only, draws nothing from the world's generator and imports only the standard library.

## Functional requirements

- **FR-1 - the static passability grid.** A world MUST accept at construction either **no grid** or
  exactly `W*H` row-major bytes, one per in-bounds cell, for bounds `W x H`. Bit 0 set blocks a ground
  mover, bit 1 set blocks an air mover; bits 2-7 are reserved and MUST be zero. Any other length, or
  any reserved bit set, MUST be refused rather than padded, truncated or masked. A world with no grid
  MUST behave in every respect as one with an all-zero grid of the same bounds. The grid MUST NOT
  change while a world is advanced.
- **FR-2 - grid, mode and stall count are canonical state.** The grid, the routing mode (FR-5, FR-6)
  and each unit's stall count (FR-7) MUST be carried by the canonical byte form and MUST enter the
  digest: two worlds differing in one grid byte, in the mode or in one stall count MUST have different
  byte forms and MUST NOT hash equal, while a world with no grid and one with an all-zero grid of equal
  bounds MUST have **identical** byte forms. The form's version MUST be raised and a form of the
  previous version refused rather than migrated. A form MUST be refused, leaving the receiving world
  unchanged, when its grid fails FR-1, its mode byte names no defined mode, or a stall count is at or
  above FR-7's threshold or nonzero without a target.
- **FR-3 - enterability, and the clamp.** A cell is **enterable** by a unit iff it is in bounds, its
  blocks-ground bit is clear, and no *other* unit stands on it when the check is made. A unit MUST NOT
  move onto a cell that is not enterable, so it never leaves the map, never stands on a blocked
  cell and never shares one. Its own cell is not an obstacle to itself, and a search starts
  from that cell whether or not it is enterable, so a unit beginning on a blocked or out-of-bounds cell
  can still be routed off it.
- **FR-4 - a route per tick, one cell along it, and the step cost.** Each tick every unit holding a
  target MUST have a route computed from its cell to its target over the enterability relation as it
  stands when that unit is resolved, and MUST advance to the **first cell of that route and no
  further**. A route MUST NOT persist between ticks nor appear in the canonical byte form. A unit
  standing on its target at the end of a tick MUST have it cleared with no residue; a unit that begins
  the tick there neither searches nor steps and is cleared by that same rule. Every cell has a step cost `c`,
  **2** everywhere here: entering a cell orthogonally costs `c`, entering it diagonally
  `c + (c >> 1)` - a truncating shift, so 2 and 3. A route's cost is the sum of its steps', and both
  modes MUST use this model over all eight neighbours of a cell.
- **FR-5 - canonical mode.** The route MUST be what this procedure yields; the procedure is the
  contract. Every cell begins unlabelled; the start cell's label is 0 and is the whole first frontier.
  A **generation** relaxes each frontier cell: for each of its eight enterable neighbours, that cell's
  label plus the cost of the step into the neighbour replaces the neighbour's label when **strictly
  less**, and the neighbour joins the next frontier - so a cell may be relabelled any number of times,
  and a diagonal is taken with **no test on the two cells it passes between**. Relaxation is sequential
  over the frontier and reads labels as they stand, so one frontier cell sees a label another lowered
  earlier in the same generation. Before each generation the search stops if the target is labelled, if the frontier is
  empty, or after `max(5, D >> 2) + D` generations, `D` being the Chebyshev distance `max(|dx|, |dy|)`
  from start to target; a target unlabelled then means **no route**. Otherwise the route is read
  backwards from the target, taking at each cell, among its eight labelled neighbours, the one
  minimising `label(neighbour) + the cost of the step from it into the current cell`, scanned
  `dx = -1, 0, +1` outer and `dy = -1, 0, +1` inner, an orthogonal candidate accepted on **less than or
  equal** and a diagonal only on **strictly less**. The walk ends at the start cell and the route is
  it reversed.
- **FR-6 - optimised mode.** A route is **admissible** iff every cell after the start is enterable,
  every cell lies in the search region, and every diagonal step has both of the cells that diagonal
  passes between enterable. The **search region** is the smallest rectangle containing start and
  target, grown by **8** cells on each side and clipped to bounds. The route MUST be an admissible
  route of **minimum total cost**; where several share it, the one taken MUST be that whose
  sequence of cells after the start is smallest compared cell by cell as `(y, x)`, a shorter sequence
  ordering before any extending it. If none exists there is **no route**, even where one
  leaving the region would reach the target.
- **FR-7 - no route, holding, and giving up.** A unit whose search returns no route MUST keep its cell
  and its target unchanged, the tick MUST complete normally, and its **stall count** MUST rise by one;
  a unit that advances MUST have it reset to zero. When a stall count reaches **16** that
  unit's target MUST be cleared with no residue and the count reset to zero. A unit holding no target
  MUST have a stall count of zero.
- **FR-8 - determinism, order, and the wall.** Units MUST be resolved in ascending id, each seeing the
  moves already resolved that tick, in **both** modes: the mode governs route selection and nothing
  else. Advancement MUST stay integer-only and free of clock, file and the world's generator, and two
  worlds with equal byte forms advanced by equal commands MUST hold equal state at every tick. The mode
  MUST be fixed when a world is built and MUST NOT change while it is advanced.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a grid walled across the map with one gap, a unit ordered to the far side | advanced to arrival | it arrives, and at no tick stands on a cell whose blocks-ground bit is set |
| **AC-2** | unit | the same world with the gap closed | advanced 16 ticks | for 15 ticks its cell and all three target fields are unchanged; on the 16th the target is cleared and its stall count is zero; no error is raised |
| **AC-3** | unit | a unit whose target lies outside the world's bounds | advanced 16 ticks | every cell it holds is in bounds at every tick, and its target is cleared on the 16th |
| **AC-4** | unit | two units whose direct lines cross at one cell, both ordered across | advanced K ticks | at no tick do two units share a cell, and a re-run of the same world and commands reproduces both trajectories |
| **AC-5** | unit | a unit at `(0,0)` ordered to `(1,1)`, with `(1,0)` and `(0,1)` both blocking ground | advanced one tick in each mode | in optimised mode it has not moved; in canonical mode it stands on `(1,1)` |
| **AC-6** | unit | a world with a non-trivial grid; the same with one grid bit toggled, ground or air; the same in the other mode; and a no-grid world beside an all-zero-grid world of equal bounds | each marshalled and hashed | the first three differ pairwise in byte form and in digest, and the last pair are equal in both |
| **AC-7** | unit | a world with a non-trivial grid, a nonzero stall count and either mode | marshalled, those bytes unmarshalled into a second world | the second world's byte form is byte-for-byte the first's and their digests are equal |
| **AC-8** | unit | a bad grid length at construction; and byte forms with a wrong grid cell count, a reserved grid bit set, an undefined mode byte, an out-of-range stall count, a stall count on a unit with no target, or the previous version | each construction attempted, each form unmarshalled | each is refused with an error and the receiving world is unchanged |
| **AC-9** | unit | small grids with obstacles, walked to arrival in each mode | each walked route's cost compared with the minimum over all admissible routes by exhaustive search | the optimised cost equals that minimum on every grid, and on at least one grid the canonical cost is strictly greater |
| **AC-10** | unit | an all-passable grid, one unit, no other unit on its path | advanced to arrival in each mode | each tick moves it at most one cell on each axis, it arrives after `max(\|dx\|,\|dy\|)` ticks in **both** modes, and its target is cleared on arrival |

**Error cases:** AC-8. A refused construction produces no world and a refused byte form leaves the
receiving world untouched; nothing else here rejects input accepted before.

## Derived properties

- **P-1 (invariant)** - For any world whose units all begin on enterable cells, after every tick every
  unit stands in bounds, on a cell whose blocks-ground bit is clear, and alone on it.
- **P-2 (negative-invariant)** - For any tick in which a unit's search returns no route, that unit's
  cell and all three target fields are unchanged; its stall count is the only field of its own it
  mutates, until FR-7's threshold fires.
- **P-3 (invariant)** - For any two worlds with equal byte forms advanced by equal command sequences,
  the states are equal at every tick, mode and grid and stall counts included.
- **P-4 (completeness)** - A unit holding a target has exactly one outcome per tick: it advances to the
  first cell of a route, holds with its stall count raised, or gives up and has its target cleared.
  There is no fourth outcome and no partial move.
- **P-5 (negative-invariant)** - For any refused construction or refused byte form, no world a caller
  can observe carries any part of the refused input: a grid is never half-applied.

## I/O examples

A world is built with bounds, units, a seed, a **routing mode** and an optional grid of `W*H` row-major
bytes. The byte form gains a mode byte and a grid section - a cell count that MUST equal `W*H`, then
that many cell bytes - and each unit record gains a stall-count byte; the version byte is raised, so a
version-2 form is refused.

A unit at `(0,2)` ordered to `(4,2)`, with `(2,1)`, `(2,2)` and `(2,3)` blocking ground:

```
optimised: (1,1) (2,0) (3,1) (4,2) -- round the wall's top, cost 3+3+3+3 = 12
canonical: also round the wall, but the wave's first-labelled route, not necessarily this one
sealed:    all of column 2 blocking -> no route; it holds, keeps its target, and after
           16 such ticks the target is cleared and the count returns to 0
```

## Out of scope

- **Which terrain tiles block, and deriving a grid from a decoded map at all.** This story consumes a
  grid and fixes how one is carried, hashed and read; producing one belongs to the terrain story.
- **Multi-cell footprints.** Every unit occupies one cell, so none can be too wide for a gap.
- **Flyers.** No unit carries a layer, so the blocks-air bit is carried and hashed but read by
  nothing here, and every unit contends with every other exactly as today.
- **Reservations.** No unit marks the cell it means to enter, so two may still be routed toward one
  cell and the later-resolved one simply finds it taken.
- **The original's two-search structure, its refresh cadences, any staggering of searches across
  ticks, and its substitute destinations.**
- **Per-cell terrain cost, speed, sub-cell movement, parallel search, route caching, group movement,
  flow fields, path smoothing, fog of war, combat, and the sprite, animation and facing layers.**

## Verification mapping

Every criterion is a unit test over synthetic worlds built in test code, so all ten are
CI-automatable. AC-4 runs a fixed scenario set and AC-9 grids small enough to enumerate
exhaustively, so P-1, P-3 and P-4 are **sampled, not proved**.

## Gate check

FR-1 -> AC-6, AC-8, P-5 · FR-2 -> AC-6, AC-7, AC-8 · FR-3 -> AC-1, AC-3, AC-4, P-1 ·
FR-4 -> AC-1, AC-9, AC-10, P-4 · FR-5 -> AC-5, AC-9, AC-10 · FR-6 -> AC-5, AC-9, AC-10 ·
FR-7 -> AC-2, P-2, P-4 · FR-8 -> AC-4, AC-6, P-3.
