# Spec — a group move keeps its shape, and keeps its slowest member's rate

## Problem and current behaviour

Every order in this tree names **one entity**. Selecting a box of units and clicking a
cell issues one move-to per selected unit, all naming the same cell, and each of them
then walks toward that one cell. The formation the player selected is destroyed on the
first step: the units converge, contend for the goal, and the ones that lose settle for
whatever free cell the search finds nearest. The box goes in as a shape and comes out as
a queue.

A mover's rate comes from its own speed alone. A scout and a siege engine ordered
together arrive minutes apart, and nothing about being ordered *together* reaches either
of them.

There is no group in this package. A world is a tick, a generator, bounds, a routing
mode, a passability grid and entities in ascending id order; an entity is integers and
bools, and its whole order is a target, a presence flag and a stall count.

## Functional requirements

- **FR-1 A GROUP ORDER.** A new command kind MUST express one order over a **set** of
  entities, and a command MUST carry a group tag. Every command of that kind sharing a
  tag inside one advance is **one order**: it is applied once, at the position of its
  first member in the command slice, so a later order still overwrites an earlier one;
  the remaining members are not applied again. Its **ordered cell** is that first
  command's, and no other member's coordinates are read. Its **members** are the entities
  the world still holds that are **alive**, each counted once; an order with no surviving
  member does nothing at all. A command naming an absent or felled entity is ignored, as
  every command naming one already is.
- **FR-2 THE CENTROID.** A group order MUST compute one centroid cell, over its members
  and in **sub-cell units**: per axis, the sum of the members' sub-cell positions, divided
  by the member count, the whole cells of the quotient taken. A resting mover's sub-cell
  position is the **centre** of its cell — its cell times the sub-cell grid, plus half of
  it — so the mean is to the **nearest** cell, **ties up**, and this MUST NOT be written
  as a rounding rule applied afterwards. The division MUST be **floor**, so it
  agrees with the original's unsigned divide over every position the original can hold
  and is defined for the negative positions this tree admits.
- **FR-3 THE FORMATION FLAG.** A group order MUST compute exactly **one** flag, once, and
  that flag MUST gate **both** of FR-4 and FR-5 and nothing else. It is set unless some
  member's **Chebyshev distance in whole cells** to the centroid **exceeds 2** — one
  member over the threshold clears it for the **whole group**, not for itself. The
  threshold is a compile-time constant carried by no file this tree reads.
- **FR-4 THE DISTRIBUTION.** In formation, each member's destination MUST be the ordered
  cell offset by that member's own displacement from the centroid — `ordered +
  (memberCell - centroid)` per axis — with each offset narrowed to a **signed byte**
  before it is added. Out of formation, every member's destination MUST be the ordered
  cell unchanged. Under **both** arms every destination MUST then be **clamped** into the
  map's bounds. A destination that is in bounds but **blocked or unreachable** MUST be
  handled by the search that already exists and MUST NOT gain a rule here.
- **FR-5 THE GROUP SPEED TERM.** An entity MUST carry a **group speed** byte. Every group
  order MUST first set it to **zero** for every member. In formation only, it MUST then
  be set, for every member, to the **minimum** of the members' speeds, computed as: a
  running byte starting at **250**; per member, a **signed 16-bit** comparison of that
  member's speed against the running byte; on a strictly smaller speed, the running byte
  becomes that speed's **low byte**. A **nonzero** group speed MUST REPLACE the entity's
  own speed wherever a rate is computed, in **every** movement domain, and MUST be the
  only thing that decides whether such an entity has a rate at all.
- **FR-6 WHAT DOES NOT CLEAR IT.** The group speed term MUST survive **arriving**, the
  **death of another member**, giving up on an order, and a round trip through the byte
  form. Its complete writer set MUST be exactly three sites and there MUST be no fourth:
  a group order (zeroed, then set in formation), a **plain** move order (zeroed), and an
  entity being felled (zeroed). This is deliberate reproduction of the original's own
  behaviour, and the site a fix would go in MUST be named in the code.
- **FR-7 CANONICAL STATE.** The group speed MUST be carried by the byte form, MUST enter
  the digest, and the form's version MUST be raised. The decoder MUST refuse every other
  version, this build's predecessor included, and MUST refuse a nonzero group speed on an
  entity that is **not alive** — the shape a felled entity cannot be in. The constructor
  MUST normalise that same shape rather than refuse it, exactly as it already does for a
  target and for a crossing.
- **FR-8 A GROUP OF ONE IS THE IDENTITY.** A group order naming one entity MUST leave its
  destination equal to the ordered cell and its group speed equal to its own speed, and
  this MUST fall out of FR-2 to FR-5 unchanged — **no arm, no branch and no early return
  keyed on the member count may exist**.
- **FR-9 THE FRONT-END.** Ordering a selection on the map screen MUST issue **one group
  order over the whole selection**, one command per selected unit sharing a tag, in one
  advance. A selection of one is a group order of one and takes no other path.

## Acceptance criteria

- **AC-1** Three units in a row, boxed and sent to a distant cell, take **three distinct
  destinations**, each the ordered cell plus that unit's own displacement from the
  centroid; the walk preserves the row.
- **AC-2** The centroid is the nearest cell to the members' mean, ties up, over a table
  that includes an exact half in both directions and an odd member count; a truncating
  mean disagrees on at least one row of that table. Two members four cells apart are in
  formation and five apart are not, which is the reading's independent check.
- **AC-3** A group whose members are spread past the threshold takes **one** destination
  for every member **and** no group speed term — one flag, both effects, asserted in the
  same test; moving one member one cell in or out flips both together.
- **AC-4** A distributed destination that would fall outside the map is **clamped** into
  bounds, on both axes and at both ends, and the member is ordered to the clamped cell.
- **AC-5** A group of a slow and a fast unit, ordered in formation, crosses cells at the
  **slow** unit's rate — both of them — and the same pair ordered out of formation crosses
  at each unit's own rate.
- **AC-6** The slow member of a moving group is **killed**; the survivors keep crossing at
  its rate for the rest of that order, and a **new plain order** to one of them restores
  its own rate.
- **AC-7** A group that has **arrived** still carries the term; a world carrying one
  marshals, decodes back to the same world, and two worlds differing only in that byte
  have different digests.
- **AC-8** The decoder refuses the previous version, and refuses a nonzero group speed on
  an entity that is not alive; the constructor normalises the same entity instead.
- **AC-9** A group order of one leaves target and group speed exactly as a single-unit
  order would, and the group order's implementation contains no member-count special
  case.
- **AC-10** The map screen issues one group order per click over the whole selection, and
  a world advanced by it distributes.

## Properties

- **P-1** `pkg/sim` gains **no float**: no floating-point type, literal or import, and no
  value crossing into it from one. Every term of FR-2 to FR-5 is integer arithmetic over
  world state alone.
- **P-2** A world in which **no group order is ever issued** advances exactly as it did
  before this story — same positions, same routes, same ticks, same give-ups. Only the
  encoding widened.
- **P-3** The whole existing suite stays green, and no existing test is weakened. Where a
  group speed changes an outcome an existing test asserts, that test changes **with a
  stated reason** recorded in `verification.md`.
