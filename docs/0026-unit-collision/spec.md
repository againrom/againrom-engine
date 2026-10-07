# Spec — deterministic unit cell occupancy

## Problem and current behaviour

`pkg/sim` advances a world one tick per `Step`: every command in slice order sets its entity's target,
then every entity holding a target moves one cell on each axis toward it — `+1`, `-1` or `0` per axis
according to whether the target's coordinate is greater than, less than or equal to the entity's own —
and an entity standing on its target has that target cleared, leaving no residue in the target
coordinates. Entities live in a slice kept strictly ascending by id, so walking it forward is walking
them in ascending id. Movement is unclamped: bounds are recorded state and not a constraint, and an
entity may walk off the grid onto negative coordinates.

There is **no occupancy**. Each entity is resolved without reference to any other, so two or more may
end a tick on one cell, and every entity ordered onto a single cell stacks onto it. A cell has no
notion of holding one unit.

The ascending-id advancement order is required and **not observable**. Because no entity's move can
affect another's, resolving them in the opposite order yields identical state at every tick, identical
byte forms and identical digests; nothing today distinguishes the two.

Neither construction nor decoding inspects positions: `NewWorld`'s only error is a duplicate id, and
unmarshalling refuses a malformed shape and non-ascending or duplicate ids but nothing about where an
entity stands. A world whose entities already share a cell is therefore both constructible and
decodable — and the map loader can build one, because a placed unit's cell is its record position with
the sub-cell fraction dropped, so two placements inside one cell collapse onto it.

Advancement is integer-only, reads no clock and no file, draws nothing from the world's generator, and
`pkg/sim` imports nothing outside the standard library — a wall held mechanically, by an import check
over the package including its tests and by a source scan over its shipped files.

## Functional requirements

- **FR-1** From a state in which no two units share a cell, advancing one tick MUST leave **no cell
  holding more than one unit**.
- **FR-2** A unit holding a target has exactly one **desired cell** per tick: the cell one step nearer
  on each axis, by the same per-axis rule as today, so a diagonal target yields a diagonal desired
  cell. The move MUST be **all-or-nothing on that desired cell** — the unit moves there only if no
  other unit occupies it at the point that unit is resolved within the tick, and if it is unavailable
  the unit MUST NOT move on either axis. Movement along whichever single axis happens to be free is
  forbidden. A desired cell equal to the unit's current cell is not a move and MUST NOT be subject to
  the occupancy test.
- **FR-3** Contention MUST be resolved by **ascending `EntityID`**, and resolution MUST be
  **incremental**: units are resolved in ascending id, and each sees the moves already resolved in the
  same tick. Two consequences are required. Where two units would otherwise end a tick on one cell,
  the lower id takes it and the higher does not move. A cell that a lower-id unit **vacates** during a
  tick is available to a higher-id unit resolved after it.
- **FR-4** A unit whose desired cell is unavailable MUST keep its current cell for that tick and MUST
  keep its target exactly as it was. A blocked unit's target MUST NOT be cleared or altered.
- **FR-5** A unit that ends a tick on its target cell MUST have that target cleared, with no residue
  left in the target coordinates.
- **FR-6** A unit whose desired cell holds no other unit MUST move exactly as it does today: one cell
  per axis per tick, arrival after `max(|dx|,|dy|)` ticks, target cleared on arrival. The occupancy
  rule MUST alter the outcome only for a unit whose desired cell is occupied.
- **FR-7** Two worlds with identical state advanced by identical command sequences MUST hold identical
  state at every tick, contention included.
- **FR-8** From a state in which two or more units **do** share a cell, advancement MUST remain
  deterministic and MUST NOT raise any cell's occupant count above the greater of one and the count
  that cell already held. It is NOT required to separate units that already share a cell.
- **FR-9** The canonical byte form MUST be unchanged: no field added, no change to the header or the
  per-entity record width, and no version bump — so a byte form this version writes today still
  decodes and still reproduces its digest.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | three or more units whose targets direct them all onto one shared cell within the same tick | advanced many ticks | at **no** tick do two units share a cell |
| **AC-2** | unit | two units, both one step from the same empty cell, that cell being each one's desired cell | advanced one tick | the **lower-id** unit occupies the contested cell and the higher-id unit's position is unchanged |
| **AC-3** | unit | a moving unit whose desired cell is held by a unit with no target | advanced one tick | the moving unit's position, `HasTarget`, `TargetX` and `TargetY` are all unchanged |
| **AC-4** | unit | three units in a line moving the same direction, the leader carrying the **lowest** id, no obstacle ahead | advanced one tick | **all three** have advanced one cell |
| **AC-5** | unit | two adjacent units, each targeting the other's current cell | advanced one tick | **neither** has moved |
| **AC-6** | unit | one unit with a target and no other unit anywhere on its path | advanced to arrival | it moves one cell per axis per tick, arrives after `max(\|dx\|,\|dy\|)` ticks, and its target is cleared on arrival |
| **AC-7** | unit | two worlds with equal byte forms, and a command stream that forces contention | both advanced K ticks | their digests are equal after **every** tick |
| **AC-8** | unit | a unit whose desired diagonal cell is occupied while both orthogonal neighbours toward its target are free | advanced one tick | **neither** coordinate has changed |
| **AC-9** | unit | many units on distinct cells with targets drawn from a fixed seed | advanced K ticks | after **every** tick all unit cells are distinct, and a re-run of the same seed reproduces the same trajectories |
| **AC-10** | unit | a command naming the cell the unit already occupies | advanced one tick | that unit's target is cleared and its position is unchanged |
| **AC-11** | unit | two units already sharing one cell, one of them ordered to the cell it stands on, and a third unit adjacent to the shared cell and ordered onto it | advanced one tick | the self-order is cleared, the third unit has not moved, the shared cell still holds exactly the original two, and a re-run of the same world and commands gives identical state |
| **AC-12** | unit | the pinned reference world, and its pinned byte form | marshalled, and the pinned bytes unmarshalled | the bytes and the digest are the ones already pinned, the encoded length is unchanged, and the pinned form decodes without error |

**Error cases: None applicable.** Advancement rejects no input it accepted before — every command is
still valid, a command naming an absent entity remains a no-op, and contention is a normal resolved
outcome rather than a failure.

## Derived properties

- **P-1 (invariant)** — For any world reachable by advancing from a state in which no two units share
  a cell, every cell holds at most one unit.
- **P-2 (negative-invariant)** — For any tick in which a unit's desired cell is unavailable, that
  unit's position and all three of its target fields are unchanged by that tick: a blocked unit
  mutates none of its own canonical fields.
- **P-3 (invariant)** — For any two worlds with equal byte forms advanced by equal command sequences,
  the resulting states are equal at every tick.
- **P-4 (completeness)** — Every unit holding a target has exactly one of two outcomes per tick: it
  advances into its desired cell — vacuously where that cell is the one it already stands on — or it
  keeps its cell because that cell is occupied. There is no third outcome and no partial move.
- **P-5 (invariant)** — For any tick and any cell, the number of units on that cell afterwards is at
  most the greater of one and the number on it before: advancement never adds an occupant to a cell
  that already has one.

## I/O examples

No new type, field, command or signature. `sim.Step(w *World, cmds []Command)` advances one tick, and
a target is set by a `sim.Command{Entity: id, X: x, Y: y}` in that slice. Occupancy is observable only
through the resulting positions and the world digest.

Two units ordered to the same cell `(2,0)`:

```
start:   #0 (0,0) -> (2,0)          #1 (4,0) -> (2,0)
tick 1:  #0 (1,0)                   #1 (3,0)
tick 2:  #0 (2,0), target cleared   #1 desired (2,0) occupied -> stays (3,0), keeps target
tick 3+: #0 (2,0)                   #1 still blocked -> stays, keeps target; the two never share
```

A convoy whose leader carries the lowest id, all ordered to `x = 9`:

```
start:   #0 (2,0)  #1 (1,0)  #2 (0,0)
tick 1:  #0 (3,0)  #1 (2,0)  #2 (1,0)     -- all three advance within the one tick
```

The same three ordered so that the leader carries the highest id:

```
start:   #0 (0,0)  #1 (1,0)  #2 (2,0)
tick 1:  #0 (0,0)  #1 (1,0)  #2 (3,0)
tick 2:  #0 (0,0)  #1 (2,0)  #2 (4,0)
tick 3:  #0 (1,0)  #1 (3,0)  #2 (5,0)     -- the line stretches, then flows again
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The rule stores no new state in the canonical world: the occupancy relation is derived while a tick is advanced and never serialized. | **(A) a per-cell occupancy grid carried in the world** — a persistent index, paid for by changing the serialized and hashed shape, invalidating every pinned byte form and digest, and adding one more invariant to maintain across a restore. **(B, chosen) derive it per tick** — no format change, and nothing to keep true between ticks. |
| **C-2** | Resolution is integer-only: no IO, no wall-clock, no floating-point value, and no import outside the standard library. | Not negotiable: lockstep advancement over a network and a reproducible digest both rest on it. |
| **C-3** | Movement stays one cell per axis per tick over the unbounded integer lattice. No speed, no footprint wider than one cell, no bounds clamp and no terrain passability enter here. | A richer movement model would couple occupancy to decoded terrain and to per-unit speed, at which point a blocked unit's behaviour could no longer be pinned by a test over positions alone. Each of those is a story of its own. |

## Out of scope

- **Terrain and object passability** — water, mountains, an impassable tile flag, placed-object cells.
  A unit still walks onto any cell no unit holds, off the grid included. Blocking on terrain is a
  separate story, not a gap left here.
- **Pathfinding and routing.** A blocked unit waits; it does not go around.
- **Swap and deadlock resolution, pushing, and yield priority.** The only priority is ascending id.
- **Separate occupancy domains.** Every unit contends with every other: there is no ground/air split,
  so a flying unit and a walking unit contend for a cell exactly as two walking units do.
- **Multi-cell footprints, variable speed, sub-cell positions, and bounds clamping.**
- **Repairing a malformed start.** Two units that begin on one cell stay there unless movement happens
  to part them; FR-8 bounds what advancement may do, not what it must undo.

**Disclosed limitations** — deterministic, accepted, and owned by the story that adds routing rather
than defects of this one:

- two units each ordered onto the other's cell deadlock (AC-5);
- a cycle of three or more units, each waiting on the next, deadlocks likewise;
- a convoy ordered so that its leader carries the highest id does not advance as a body: each follower
  waits until the unit ahead of it has vacated, so the line stretches until one-cell gaps open and
  then advances every tick again — the mirror of AC-4;
- a unit whose target cell is held by a unit that never moves keeps that target forever.

## Verification mapping

Every criterion is a unit test over synthetic worlds built in test code — no game asset, no file, no
GPU — so AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7, AC-8, AC-9, AC-10, AC-11 and AC-12 are all
CI-automatable, AC-9 as a fixed-seed randomized sampling. P-1, P-4 and P-5 are universals, and between
a set of named scenarios and one seeded sweep they are **sampled, not proved** over all reachable
states; nothing here claims otherwise.

## Gate check

FR-1 → AC-1, AC-9, P-1 · FR-2 → AC-6, AC-8, AC-10, P-4 · FR-3 → AC-2, AC-4, P-3 ·
FR-4 → AC-3, AC-5, P-2 · FR-5 → AC-6, AC-10 · FR-6 → AC-6, AC-9 · FR-7 → AC-7, P-3 ·
FR-8 → AC-11, P-5 · FR-9 → AC-12.
