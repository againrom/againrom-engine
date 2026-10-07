# Spec — a body decays

## Problem and current behaviour

A unit that dies falls where it stood and stays there for as long as the map is open. The fall
plays once and the body then holds its last dying frame for ever. Nothing else about it changes:
health at or below zero never moves again, no unit is ever removed from a world, no id is ever
freed, and the picture has one corpse state and no second one.

The ground under a body is answered by two different rules. A unit at **exactly zero** health is
downed, keeps its cell for good, and only a further blow moves it. A unit **below** zero is dead and
contributes no occupancy from the instant the blow lands — so one that dies in a doorway does not
seal it, and one that stops at zero seals it permanently. And the defence a unit carries is
unchanged by dying, so a downed unit, which can still be struck, is as hard to hit as it was
standing.

## Functional requirements

- **FR-1** An entity MUST carry a **decay stage** as canonical simulation state: zero for one that
  has not died, 1 to 4 for one that has. It MUST enter the byte form and the digest and MUST be the
  only thing saying which corpse state an entity is in. **Nothing outside the simulation may advance
  it.** It MUST be positive on exactly the entities that are not alive: a stage outside 0..4 MUST be
  refused by constructor and decoder alike, and a positive one on a living entity or a zero one on a
  dead entity MUST be normalised by the constructor and refused by the decoder.
- **FR-2** The tick that leaves an entity's health at or below zero — a kill, a damage command or a
  resolved blow — MUST, in that same tick and once only, put it at **stage 1**, **halve its
  defence** by an arithmetic shift of one, and start a **dwell** counted in ticks. The dwell's
  length MUST be that entity's own dying time, carried from its class definition, and **8** where
  the definition names none; a negative one is no dwell.
- **FR-3** An entity at stage 1 whose dwell has not run out MUST **occupy its cells** exactly as a
  living one of its domain does; at every later moment a body MUST occupy nothing. This replaces
  both rules above: a dead body blocks for its dwell, a downed one no longer blocks for ever.
- **FR-4** The dwell MUST fall by one on every tick until it reaches zero. Once it is zero the
  entity's health MUST fall by one on **one tick in thirty-two** — this package's own full tick,
  taken twice — and on no other, saturating at the representable floor rather than wrapping.
- **FR-5** From the moment the dwell is zero the stage MUST be the ladder of that entity's health,
  evaluated every tick: **2** at or below -10, **3** at or below -20, **4** at or below -40,
  **5 below -600**. Stage 5 is not stored: the entity MUST be **removed from the world** on the tick
  it reaches it — its record, its route, and any attack order a surviving entity holds on it.
- **FR-6** An entity whose movement domain is **not the ground domain** MUST have its health set to
  **-1000** on the tick its dwell reaches zero, which by FR-5 removes it at once: such a unit plays
  its fall, holds it for the dwell, and vanishes leaving no body and no bones.
- **FR-7** The class animation descriptor MUST carry the **length of one direction's slot in the
  bone block**, derived by the same arithmetic and the same absent-phase clamp as every other slot
  it carries, and the render tier's mirror MUST carry it value for value. A class declaring no bone
  phase MUST come out at zero.
- **FR-8** There MUST be exactly one **bone selection**, a pure function of the corpse class's
  descriptor, that class's own frame count, the facing octant and the stage, answering a sheet frame
  and whether it draws reflected, or that there is none. The frame MUST be the bone block's base
  plus the **direction's own slot** at the bone slot length plus `stage - 2`; the direction rule and
  the mirror MUST be the animated blocks' own, read off the **corpse class's** layout. It MUST
  answer "no frame" for a class with no bone block, for a stage below 2, and for any index outside
  the frame count it was handed, and MUST answer an index it has not proved for no input.
- **FR-9** An entity the simulation reports not alive MUST be drawn through the bone selection when
  its stage is at or above 2 and through the fall selection otherwise; where the bone selection
  yields no frame it MUST fall through to the drawing it had before this story. **No unit ever stops
  being drawn on account of decaying**, and no input here is an error.
- **FR-10** The canonical byte form MUST carry the stage, the dwell and the entity's dying time at a
  **new version 17**; any earlier version MUST be refused rather than migrated. A decoded world MUST
  hold exactly the state it was cut from and MUST refuse what FR-1 refuses.
- **FR-11** The ladder MUST be a pure function of the world and the tick index: for one command
  stream `k` ticks MUST reach the same state and digest on every run and machine, and a world
  marshalled at tick `k` and read back MUST advance identically to the one it was cut from.

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | worlds built with a positive stage on a living entity, a zero stage on a dead one, and stages 5, 6 and 255 | each is constructed | the first two are normalised, the last three refused, and every world built has a positive stage on exactly its not-alive entities |
| **AC-2** | unit | a living entity with a known defence and dying time, under order and mid-crossing | it is killed, damaged to death, and felled by a resolved blow | each ends that same tick at stage 1, its defence shifted right by one, and the order, crossing and group term it already loses gone; and over dying times of 8, 1, 0 and none the dwell comes out 8, 1, 0 and 8, the last two torn down on the tick they fall |
| **AC-4** | unit | a mover whose route runs through the cell a second unit stands on, and that unit is killed outright, and separately damaged to exactly zero | the mover is advanced across the dwell and past it | in both cases the cell is refused while the dwell runs and entered on the first tick after it runs out |
| **AC-6** | unit | a body at stage 1 with its dwell run out at a known tick index, and one at the representable health floor | each is advanced over three full periods | the first falls by exactly one per thirty-two ticks, on the same phase every time and on no other tick; the second does not move and does not wrap |
| **AC-7** | unit | bodies felled to -1, -15 and -50 | each is advanced from its dwell's end | the first climbs rung by rung as its health passes each threshold; the second reaches stage 3 and the third stage 4 on their first walk tick; no stage ever falls |
| **AC-8** | unit | a body at health -600 exactly and one at -601 | each is advanced | the first is at stage 4 and stays; the second is gone — no record, no route — and an entity that held an attack order on it holds none |
| **AC-9** | unit | entities in the ghost and air domains and one on the ground, felled together | each is advanced past its dwell | both non-ground entities reach health -1000 and are gone on that tick; the ground one is at stage 1 and remains |
| **AC-11** | unit | classes whose phase scalars are written by hand, some declaring no bone phase | the descriptor is derived | the bone slot is the declared count, zero where absent, and every base already derived is unchanged |
| **AC-12** | unit | a corpse descriptor with a known tail base, direction count and bone slot | the selection is asked at stages 0 to 5 and at every octant | stages 0 and 1 have no frame; 2, 3 and 4 give the tail base plus that direction's slot plus 0, 1, 2; at the five-way layout the upper octants fold onto the lower slots and mirror; and a descriptor with no bone block, or one whose index its sheet cannot hold, answers that it has no frame |
| **AC-14** | unit | a hand-built world, a class whose corpse is a second class with its own sheet, an entity felled at a known tick | the pushed entities are read across the ladder | the fall plays, the body holds its last dying frame, and at stages 2, 3 and 4 it carries the corpse class's art at that class's bone frame 0, 1 and 2 for the direction it died facing; with a corpse class holding no bone block it is still drawn, by the drawing it had before this story |
| **AC-16** | unit | worlds spanning the whole state — living, at every stage, dwell running and run out | each is marshalled and read back | the bytes round-trip, the digest agrees, the version byte is 17, a version-16 buffer is refused, and every shape FR-1 refuses is refused on the way in |
| **AC-17** | unit | one command stream over a hand-built world containing kills and damage | `k` ticks are run, and run again from a world cut and restored at every index | the pinned fields, the byte form and the digest agree at every index in both runs |
| **AC-18** | unit | a placed unit and a placed person from hand-built definition tables | each is spawned | the entity carries the dying time its own row names, and the constructor's default where the row leaves it empty |
| **AC-19** | manual | the game on a map against a lawful install | a unit is killed and watched | it falls, lies still while its dwell runs, then becomes bones and steps through them as its health walks down |

**Error cases:** the selection path is total, so there is nothing to refuse there. The simulation
refuses exactly what FR-1 and FR-10 name, and a refused construction or decode yields no world.

## Derived properties

- **P-1 (invariant)** — A positive stage and being not alive hold of the same entities, in every
  world this package builds, decodes or advances.
- **P-2 (invariant)** — The stage never falls and health never rises, so the ladder is monotone.
- **P-3 (negative-invariant)** — No frame the bone selection answers lies outside the **bone block
  of the sheet it came from**: every answer is at or after that block's base for its own direction
  and strictly before the next direction's slot.
- **P-4 (completeness)** — Every entity is drawn by exactly one of four things: the bone selection,
  the fall selection, the live selection, or the square. No fifth case, and none is drawn by none.
- **P-5 (idempotence)** — A world cut and restored at tick `k` reaches, at every later index, the
  state, bytes and digest the uncut world reaches.

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The stage is **state**, not a function of health. | **(A) derive it** — an overkilled body would be bones on the tick it fell and never play its fall. **(B, chosen) carry it**, and let health drive it only after the body is torn down. |
| **C-3** | Stage 5 **removes** the entity. | **(A) keep it and mark it gone** — a record nothing draws, nothing routes around and nothing may name, carried for ever in every save. **(B, chosen) remove it**, and accept that a script check naming a finished body measures nothing, as one naming a unit the world never held already does. |
| **C-4** | The dwell is a **per-entity number carried from the class**, not a constant. | **(A) one constant** — the number the definition names would be decoded and unused. **(B, chosen) carry the column.** |
| **C-5** | The bone frame is **direction-slotted**. | **(A) an absolute index into the bone block** — the reading the shortest statement of the rule invites; it draws one octant's bones for all eight and never mirrors. **(B, chosen) the slot arithmetic every other animated block takes.** |


## Out of scope

- **The replay of the last two dying frames when a body is struck again**, which needs a blow on a
  corpse — refused here.
- **Any second effect of dying beyond the defence**, and **corpse art beyond the bone block** —
  gibs, the shadow, overlay sheets, sound, any tint or fade.
- **A body's cells being anything but its own.** The footprint rule is unchanged.
- **The engine's own arm in which an idle cycle wins over the corpse fork.** Every not-alive entity
  goes through the death path here, as it already did.

**Disclosed limitations**, accepted and owned: a corpse class whose bone phase count is the absent
sentinel has no bone block here, where the engine indexes outside it; the dwell is measured in this
package's own tick, the shorter of the two readings; and a body that has finished decaying is no
longer visible to a script check naming it.

## Verification mapping

Every criterion above but the last is a unit test over hand-built worlds, descriptors and definition tables — no window,
no clock, no game install — so all are CI-automatable; AC-19 needs a lawful install and a window.
P-1 is witnessed by AC-1 and AC-2, P-2 by AC-7 and AC-8, P-3 by AC-12, P-4 by AC-14 over one snapshot, P-5 by AC-17.

## Gate check

FR-1 to AC-1, C-1 · FR-2 to AC-2 · FR-3 to AC-4 · FR-4 to AC-6 · FR-5 to AC-7,
AC-8, C-3, P-2 · FR-6 to AC-9 · FR-7 to AC-11 · FR-8 to AC-12, P-3, C-5 · FR-9 to AC-14, P-4 · FR-10 to AC-16 · FR-11 to AC-17, P-5. C-4 to AC-2 and AC-18. AC-19 reads FR-2,
FR-5 and FR-9 together in the running game.
