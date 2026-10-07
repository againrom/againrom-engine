# Spec — an attack that resolves

**Intensity:** spec-anchored / static. **Terrain:** brownfield for `pkg/sim`'s advance and its
canonical byte form, greenfield for the resolution itself.

A hostile a player orders attacked must die of the blows, on the engine's own arithmetic.

## Current behavior

A world can be told to kill a unit outright and to subtract a named amount from its health; both
are debug tools and say so, rolling nothing and costing no tick. Health, the three life states —
alive, downed at exactly zero, dead below it — and the clearing of a felled unit's order exist and
are unchanged here. An advance applies its commands in slice order, resolves every alive entity
holding a destination in ascending id, then increments the tick. The world owns one seeded
generator whose whole state its byte form carries and nothing draws from it. Nothing rolls to hit,
applies absorption or attacks on a period; no order names a victim, and no unit's stats beyond
health and speed are carried.

## Functional requirements

- **FR-1 — an attack order names a victim.** One command orders one entity to attack another. It is
  applied where every other order is: before the tick's own work, in slice order, later over
  earlier. It is **ignored**, changing nothing, when it names an entity the world does not hold, a
  victim it does not hold, an attacker that is not alive, or an attacker as its own victim. An
  accepted order naming a **different** victim from the one already held, or none, restarts the
  cycle — ready, nothing owed. One naming the **same** victim leaves the cycle where it stands, so
  a caller re-issuing its order every tick still lands blows.

- **FR-2 — attacking and walking are one state, not two.** An attacker holds a victim or a
  destination and never both: an attack order ends whatever walk its attacker was on, route
  included, and a move order — alone or in a group order — ends whatever attack its mover was on.
  Ending either leaves no residue. Neither ends a cell crossing already begun, and an attacker that
  owes crossing ticks pays them and runs its cycle in the same tick.

- **FR-3 — a blow costs ticks, and the cost is the unit's own two numbers.** Every entity carries a
  **charge** and a **relax** count; a charge below one counts as one, and a negative relax as zero.
  An attacker's cycle has exactly three phases — **ready**, **charging**, **relaxing** — and ready
  is the phase an entity that has never attacked is in.
  - Every alive entity holding a victim takes one turn per advance, in **ascending id**, and all
    take it **after every entity has moved** and before the tick is incremented.
  - At its turn a ready attacker becomes charging and loads its charge **within that same turn**,
    so being ready costs no tick. It then pays one of what it owes, and if it still owes something
    the turn ends.
  - Owing nothing, a charging attacker **strikes** (FR-4), becomes relaxing and loads its relax
    plus a fresh draw from `[0,3]`; a relaxing attacker becomes ready. **Exactly one of those two
    events per attacker per advance**, whatever the commands hold.
  - So, counting the advance that applied the order as the first, a blow lands on the `charge`-th,
    and consecutive blows are `charge + relax + [0,3]` advances apart.
  - **Nothing in that period reads speed.**

- **FR-4 — one resolution, in one order.** A blow is resolved by one rule and there is no second.
  Positions are read as they stand after the tick's movement. In order:
  1. it is **refused before anything is drawn** when the victim is further than one cell away —
     Chebyshev distance above 1 — or when the victim has no health system;
  2. the damage is `damageBase + U[0, damageSpread]`;
  3. the roll is `U[-100, 100]`, and the blow **lands** when the attacker always hits, or
     `toHit + roll` is greater than the victim's **defence**, or the roll is at least **90**;
  4. a landed blow's damage is reduced by the victim's **absorption**, flat;
  5. the victim's health falls by what is left, when what is left is positive, and by nothing
     otherwise.
  Every `U[a,b]` is inclusive at both ends and is **one draw, taken whether or not its outcome can
  matter** — steps 2 and 3 each draw once even at a spread of zero or for an attacker that always
  hits, and a blow refused at step 1 draws neither. FR-3's relax draw is taken on every strike
  tick, refused or not, after these two where they happen. Every sum, difference and comparison
  here is exact for every value the seven numbers can hold; only health saturates, at the least
  value it can carry.

- **FR-5 — death by a blow is the death the world already has.** A blow taking health to zero or
  below fells its victim by the rule any other blow fells one by — downed at exactly zero, dead
  below it — and a felled unit's orders, walk and attack alike, are dropped in the same tick with
  no residue. A **downed** victim may still be ordered attacked and struck; that is the only way
  out of the state.

- **FR-6 — an order ends with the unit it named.** An attack order whose victim is **dead** ends,
  with no residue and nothing drawn, at its attacker's own turn — so a victim killed by a lower id
  ends a higher id's order in that same tick, and one killed by a higher id ends the lower id's on
  the next. A felled attacker holds no attack order.

- **FR-7 — the whole cycle is canonical state.** The victim, its presence, the phase, the count
  owed, the **seven numbers** a cycle and a resolution read — charge, relax, to-hit, defence,
  absorption, damage base and damage spread — and the always-hits mark are **twelve fields**, all
  carried by the world's byte form and all entering its digest. A world advanced from bytes is the
  world those bytes were taken from, blow for blow.

- **FR-8 — the byte form refuses what a tick cannot produce.** A form is refused when it carries an
  attack order on a unit that is not alive; a phase, a count owed or a victim id other than zero on
  a unit holding no order; a count owed below zero or above what its own phase could have loaded; a
  phase byte outside the three; a presence or always-hits byte other than 0 or 1; or an order
  naming its own unit or one the form does not hold.

## Acceptance criteria

All at the **unit** level, against synthetic worlds.

| # | GIVEN | WHEN | THEN |
|---|---|---|---|
| AC-1 | two adjacent units, the attacker always-hits with a damage base above absorption | it is ordered to attack, and the world advanced until the victim is dead | the victim is downed on one advance and dead on a later one, and neither holds an order |
| AC-2 | an always-hits attacker with charge `c`, relax `r`, damage base above absorption | the world is advanced one tick at a time | health falls on the `c`-th advance and next `c + r + j` later, `j` in `[0,3]` |
| AC-3 | two worlds from one seed, alike but for one entity's speed | both are advanced by the same commands for many ticks | they agree tick for tick in every other field, and health falls on the same advances |
| AC-4 | an attacker whose to-hit cannot beat the defence and which does not always hit | many blows are advanced through | some remove nothing and the rest remove `base + [0, spread]` less absorption |
| AC-5 | an always-hits attacker against a defence no roll can beat | one blow is advanced through | it lands |
| AC-6 | an attacker whose damage never exceeds the victim's absorption | many blows are advanced through | health never changes |
| AC-7 | an attacker and a victim two cells apart | many cycle periods are advanced through | health never changes, the cycle still runs, and each strike advance costs one draw where a reachable victim would cost three |
| AC-8 (error) | one live entity and one victim | the live one is ordered to attack an id the world does not hold, and separately itself | the digest equals that of the same world advanced by no command |
| AC-9 (error) | a downed and a dead entity | each is ordered to attack a live neighbour | neither takes the order |
| AC-10 | an attacker mid-cycle | the world is marshalled, decoded into a second, and both advanced alike | they agree in every field and in their digests at every tick |
| AC-11 (error) | bytes carrying each refused shape of FR-8, one at a time | they are decoded | each is refused and the receiving world is left exactly as it was |
| AC-12 | an attacker under a walk order, mid-crossing, holding a route | it is ordered to attack, then to move again | victim and no destination or route, then destination and no victim, the crossing untouched throughout |
| AC-13 | two attackers on one victim, the lower id able to kill it | the killing tick is advanced through | the higher id's order is gone at the end of it and it drew nothing that tick |
| AC-14 | a world at full attack state | its bytes are measured and its digest taken | width, version byte and offsets are the layout below, and moving any of the twelve fields — victim and presence together, the rest alone — moves the digest |
| AC-15 | one attacker re-issued the same order every tick, one re-issued a different victim every tick | several periods are advanced through | the first lands blows on FR-3's period, the second never lands one |

**Error cases:** AC-8, AC-9, AC-11, AC-13.

## Derived properties

- **P-1 (idempotence)** — duplicating every attack order in a command list yields exactly the world
  the undoubled list yields, field for field.
- **P-2 (invariant)** — exactly one phase event per attacking entity per advance, and so at most
  one blow, for every charge and relax including zero and negative.
- **P-3 (negative invariant)** — a blow that does not land, or whose damage after absorption is not
  positive, or whose victim has no health system or is out of reach, changes no health.
- **P-4 (negative invariant)** — an attack order naming an absent entity, an absent victim, the
  attacker itself, or an attacker that is not alive changes no field of any entity.
- **P-5 (completeness)** — every field the attack state consists of is carried by the byte form and
  enters the digest; a world round-tripped through its bytes is identical in all of them.
- **P-6 (invariant)** — the advance a blow lands on is a function of the advance its order was
  applied on, the attacker's charge, its relax and the draws, and of nothing else about it.

## I/O examples

The order is a command of the shape every other order has: a kind, the attacker, two numbers. The
victim rides in the first as its id — the same 32 bits — and the second is unread. The kind is a
new value beside the existing ones, so nothing already written changes.

The unit record grows by 39 bytes at its tail — the only part of the form this work touches —
everything little-endian, and the version byte moves, so no form written before this work is read
by this build and there is no migration.

The tail, by offset in the record: the victim id at +44 as an unsigned 32-bit, its presence byte
at +48 (0 or 1), the phase byte at +49 (0 ready, 1 charging, 2 relaxing), then count owed, charge,
relax, to-hit, defence, absorption, damage base and damage spread as signed 32-bit at +50, +54,
+58, +62, +66, +70, +74 and +78, and the always-hits byte at +82 (0 or 1).

## Constraints

**Reach.** A blow's range test is a fixed one cell, between whole-cell positions. The alternatives:
a **per-unit reach number** buys an expressible reach at the cost of four bytes of hashed state
only equipment could move, and equipment is out of scope; a **footprint-derived reach** collapses
to the fixed test, every entity here occupying one cell.

**Determinism.** No draw may come from anywhere but the world's own generator, and everything an
advance does — which values are drawn, how many, and in what order — must follow from the world's
own state and its commands alone. Within one advance a draw may decide how many later draws that
advance makes; that is still a function of state the byte form carries. Nothing outside the world
may reach any of it.

## Out of scope

- **A unit deciding whom to attack**, and **walking to a victim out of reach**: an attack order
  neither acquires nor moves.
- **Retaliation and hostility**; **what a kill pays** — experience, gold, the corpse's dwell; and
  **equipment**, with the charge, relax and reach a weapon carries.
- **The second and third damage components**, the damage-kind reduction, the elemental protections,
  ranged flight time, and everything a spell does.
- **Filling a placed unit's combat numbers from its class.** A world built by a loader carries
  whatever the loader puts there; this work defines what a world does with those numbers.
- **The existing debug kill and damage commands**, which keep exactly the behaviour they have —
  including that two damage commands in one advance remove twice the health.

## Verification mapping

Every AC is CI-automatable; none needs a live or manual run.

## Gate check

FR-1 → AC-8, AC-9, AC-12, AC-15, P-4. FR-2 → AC-12. FR-3 → AC-2, AC-3, AC-15, P-2, P-6. FR-4 →
AC-4, AC-5, AC-6, AC-7, P-3. FR-5 → AC-1, AC-12. FR-6 → AC-1, AC-13. FR-7 → AC-3, AC-10, AC-14,
P-1, P-5. FR-8 → AC-11, AC-14.
