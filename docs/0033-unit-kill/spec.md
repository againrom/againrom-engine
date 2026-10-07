# Spec — health in the world: a unit that can be hurt, downed and killed

## Problem and current behaviour

A unit cannot be hurt. Its whole canonical state is an id, a cell, a target with a presence byte, an
opaque class key, a stall count and a stored route: every unit moves, every unit blocks its cell, and
nothing takes either away. The only order is a move-to naming a unit and a cell, and a command
carries no kind, there having only ever been one. The byte form stands at version 4 — a 34-byte
header, the grid, one 26-byte record per unit, then the routes — refuses every other version rather
than migrating it, and is hashed FNV-1a over its own bytes.

The front-end selects by a tap or a marquee release, marks every selected unit the latest snapshot
holds, and on a right press orders one move each, all naming one cell, ascending by id. An id the
snapshot no longer holds is skipped and stays selected: only a tap or a release replaces the set.
What crosses to the drawing side is an id, a cell and art — nothing of that unit's condition. The
map screen reads no letter key at all.

## Functional requirements

- **FR-1 — health is the state, and there is no death flag.** Every unit MUST carry two signed
  integers, its **health** and its **maximum**, and its life state MUST be read off those two and
  nothing else: **dead** where health is negative, **downed** where health is zero and the maximum
  positive, **alive** otherwise — so a maximum of zero or less is a unit with no health system, alive
  and never downed. Health MUST NOT be clamped: a killed unit's is negative and stays negative.
- **FR-2 — the state is canonical.** Both numbers MUST be carried by the byte form, MUST enter the
  digest, and MUST round-trip at any integer value: two worlds differing in one unit's health or
  maximum MUST have different forms and MUST NOT hash equal. The version MUST be raised to **5** and
  every other refused rather than migrated, a whole well-formed version-4 world included. A unit that
  is **not alive** MUST hold no target, no route and a zero stall count: a form asserting otherwise
  MUST be refused with the receiver unchanged, and a world built with one MUST have that order
  cleared. Dying MUST NOT remove a unit, release its id, or move another unit's fields.
- **FR-3 — two orders beside the move.** A command MUST name a **kind**, the move-to being the kind
  a command naming none has, so every command already written stays what it was. **Kill** MUST set
  health to **-1** whatever the maximum; **damage** MUST carry an amount and subtract it from health,
  with no clamp and no heal; each MUST clear that unit's order where it leaves health at zero or
  below. Each MUST be a no-op moving no field and no digest where it names a unit the world does not
  hold or one already dead, and damage MUST be one also for a non-positive amount or a non-positive
  maximum. A command of an undefined kind MUST be ignored.
- **FR-4 — who moves, who blocks, who may be ordered.** Only an **alive** unit MUST be advanced: a
  downed or dead one MUST keep its cell whatever its neighbours do. Only a **dead** unit MUST stop
  contributing occupancy — a living unit MUST be able to route through and stand on a corpse's cell,
  and MUST NOT be able to enter a downed unit's. A move-to naming a unit that is not alive MUST be
  ignored, leaving no target and no residue.
- **FR-5 — what a map's units start at.** Every unit built from a decoded map MUST begin at a health
  and a maximum of **100**, and no other way of assembling a world MUST set either number.
- **FR-6 — the state crosses to the drawing side.** Each unit's snapshot entry MUST carry its life
  state and its two numbers beside what it already carries, and that state MUST be the simulation's
  own answer rather than one re-derived by the side that draws.
- **FR-7 — a corpse is not selectable, a downed unit is.** A tap and a marquee release MUST select
  every unit they cover **that is not dead**, so a tap on a corpse selects nothing. A selected
  unit MUST be marked, and MUST be issued a move order, exactly when the latest snapshot holds it and
  it is not dead — one test, both readers. A unit that dies while selected MUST be **skipped, not
  dropped**: only a tap or a release replaces the selection.
- **FR-8 — two keys.** With a world under the map screen, **K** MUST issue one kill and **L** one
  damage of a tenth of that unit's maximum but never below 1, for each unit FR-7 marks, in ascending
  id, on the press edge and not while held. Neither MUST change the selection itself, and both MUST
  reach the world only through an advance. On every other screen, and with no world, both MUST do
  nothing.
- **FR-9 — the bar.** Every drawn unit that is not dead and whose maximum is positive MUST carry a
  horizontal bar over its cell: a ground of fixed width, and on it a fill of that width in the
  proportion health bears to maximum, in integer arithmetic — the whole ground at full health, none
  when downed. Every other unit MUST carry no bar. A bar MUST take the cell and the relief offset its
  unit's mark takes.
- **FR-10 — the wall holds.** The selection, the keys, the bar and every life state read for drawing
  MUST be front-end state alone — never world state, never hashed, never serialized — and for one
  command stream, k ticks MUST reach the same state and digest whatever ran in front of them.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | (100,100), (1,100), (0,100), (0,0), (0,-3), (-1,100), (-1,0), (-1000,7) | each classified | alive, alive, downed, alive, alive, dead, dead, dead — one state each, never two, never none |
| **AC-2** | unit | a unit at 100/100 under order | ten damages of 10, then one more | 90 down to 0, downed there with target, stall and route cleared; the eleventh leaves -10, dead |
| **AC-3** | unit | 10 damage at 5/100; a kill on an alive unit under order, on a downed one, on one at 0/0 | applied | -5 and dead, downed not passed through; -1, dead and the order cleared in all three kills |
| **AC-4** | unit | kill and damage on an absent id and on a dead unit; damage 0, -7, and 10 at 0/0; an undefined kind | each applied | every one a no-op: each digest equals that of a step carrying no command |
| **AC-5** | unit | a downed and a dead unit each on a cell, a mover ordered onto each, both ordered themselves | ticks advanced | the mover enters the corpse's cell, never the downed one's; neither moves; a move-to on either leaves no target |
| **AC-6** | unit | three units with distinct pairs, one negative maximum | marshalled | first byte 5, records 34 bytes, health at +26 and maximum at +30 little-endian; the form equals a hand transcription of the layout and its digest a value derived from those bytes outside this package |
| **AC-7** | unit | a whole well-formed version-4 world; every other first byte; forms with a target, a stall count or a route on a unit not alive | each unmarshalled | refused, the receiver unchanged field for field |
| **AC-8** | unit | worlds differing in one health, one maximum, nothing; a corpus walked to quiescence under all three orders | marshalled, hashed, and decoded into a second world advanced beside the first | the differing differ in form and digest, the identical agree, and the pair hold equal digests at every later tick |
| **AC-9** | unit | a world from a decoded map; one of hand-built units | read | every map-built unit at 100/100; the hand-built keep what they were given |
| **AC-10** | unit | a snapshot over an alive, a downed and a dead unit | pushed | each entry carries the world's own state answer and that unit's two numbers |
| **AC-11** | unit | an alive, a downed and a dead unit on known cells, all three selected | a tap on each, a box over all three, the marks, a right press, K, then L | tap and box take alive and downed, never dead; the dead id is marked and ordered by nothing and stays in the set; K and L issue one command per marked unit ascending, the set unchanged |
| **AC-12** | unit | 100/100, 75/100, 0/100, -1/100, 3/0 | the bar geometry taken | full, three quarters, an empty ground, none, none — integer widths, each bar on its own cell's relief offset |
| **AC-13** | manual | the game on a lawful install, a map with several units | a group boxed and ordered, chipped with L until one is downed, then K | the group walks; the chipped unit's bar empties and it stops while still blocking; a corpse loses bar and mark, cannot be clicked or boxed, and the others walk over it |

**Error cases:** AC-7. Nothing else rejects input: an order for an absent, downed or dead unit, an
amount of zero and a unit with no health system are **resolved outcomes** — a no-op — and never
failures, the sender of an order not knowing what the world still holds when it arrives.

## Derived properties

- **P-1 (completeness)** — For any pair of health and maximum a unit is in exactly one of the three
  states, and no field exists that could disagree with the pair that produced it.
- **P-2 (invariant)** — After every tick, and in every form a decode accepts, a unit that is not
  alive holds no target, no route and a zero stall count.
- **P-3 (negative-invariant)** — For a refused form the receiving world is unchanged in every field;
  for a no-op command no field of any unit and no digest differs from a tick carrying no command.
- **P-4 (invariant)** — A world marshalled at any tick and decoded into another advances identically
  at every later tick — same cells, same health, same digests — under any stream of orders.
- **P-5 (negative-invariant)** — No selection, key press, bar, snapshot entry or life state read for
  drawing changes a world field, its byte form or its digest.

## I/O examples

```
record   ... class i32 | present u8 | stall u8 | health i32 | maximum i32      34 bytes
form     version 5, a 34-byte header, the grid, the records, then the routes as before

command  kind 0 move-to (x, y)      kind 1 kill      kind 2 damage, amount in x
K, L     selection {3,7,9} -> three commands, ascending, applied by one advance
```

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The life state is **derived from the two numbers**; no flag records it. | **(A) a death flag beside the health** — two sources for one fact, free to disagree after a write that moves one and not the other, and a combination the byte form must then refuse rather than derive. **(B, chosen)** the numbers alone: a unit killed outright and one damaged to death are one state by construction. |
| **C-2** | A **dead** unit frees its cell at once; a **downed** unit keeps it. | **(A) free it at death, no state between** — a body leaves the collision map the instant it is hit. **(B) hold it for a countdown** — a per-unit timer in hashed state and a per-class duration nothing here has. **(C, chosen)** two occupancy phases with a blow between them. |
| **C-3** | A maximum of **zero or less means no health system**: alive, immune to damage, still killable. | **(A) a maximum for every unit at construction** — every world already built changes state and digest. **(B) a has-health flag** — C-1's hazard, one field out. **(C, chosen)** the maximum carries it. |
| **C-4** | A unit that dies while selected is **skipped, not dropped**. | **(A) drop it** — the set would then be replaced by something other than a tap or a release. **(B, chosen)** skip it: nothing revives, so no observer can tell a skipped id from a dropped one. |

**Disclosed limitations**, accepted and owned: a downed unit has no exit but a further blow, so a
body blocks its cell for as long as the world runs; and a corpse is drawn exactly as it was alive,
so until it has art the missing bar and the refused click are all that say it is dead.

## Out of scope

- **The death and downed animations and any corpse art** — a tint or colour scale included: what a
  dead unit looks like is a later story's whole subject.
- **Combat**: who attacks whom, an attack's cadence and reach, the hit roll, absorption, resistance,
  damage kinds, experience and gold. K and L are debug tools, not a formula.
- **Per-class health**, and any table that would supply one.
- **Healing, regeneration, armour, revival, and any exit from downed but a further blow.**
- **Corpse decay**: nothing counts down or rots, no unit leaves the world, no id is freed, no loot
  is dropped.
- **Multi-cell footprints and flying units**, as before, and any reservation of a cell.

## Verification mapping

AC-1 to AC-12 are unit tests over worlds, forms, snapshots and selections built in test code — no
window, no clock, no install — so all twelve are CI-automatable; AC-13 needs an install. P-1 is
structural, witnessed by AC-1's table; P-2 by AC-2, AC-3, AC-5, AC-7; P-3 and P-5 are **sampled, not
proved**, over AC-4, AC-7, AC-11; P-4 over AC-8.

## Gate check

FR-1 → AC-1, AC-2, P-1 · FR-2 → AC-6, AC-7, AC-8, P-2, P-3 · FR-3 → AC-2, AC-3, AC-4, P-3 ·
FR-4 → AC-5, P-2 · FR-5 → AC-9 · FR-6 → AC-10, P-5 · FR-7 → AC-11, C-4 · FR-8 → AC-11, AC-13 ·
FR-9 → AC-12, AC-13 · FR-10 → AC-8, AC-10, AC-11, P-4, P-5.
