# Spec — a placed unit's combat numbers

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/mapload`.

## Context

A map's placements build a world whose units carry a health pair, a rate and a movement domain,
each taken from the class definition the placement resolves to. They carry no combat numbers at
all. An attack ordered on a loaded map therefore runs a one-tick cycle, rolls a damage of zero
against a defence of zero, and removes nothing — every unit on every loaded map is harmless, and
unkillable by anything but the debug commands, so no mission whose end condition counts the dead
can be reached **by fighting**.

This story fills, from the same definition and on the same resolution, the numbers an attack cycle
and a hit resolution read. It adds no field to an entity: every one of them already exists, is
already carried in the canonical byte form, and is already read by a tick. What is missing is only
the load-time write.

Two things a first reading suggests are false and the contract turns on both. The class tables
carry **no reach column**, so a reach filled from a class would be one value for every class and
this story fills none. And the scenario difficulty adjusts **two** of the eight numbers, not all of
them and not the damage.

## Functional requirements

**FR-1 — the eight values a placement takes from its class.** A placement that resolves to a unit
definition MUST carry, from that definition: the attack charge, the attack relax, the to-hit, the
defence, the absorption, the damage base, the damage spread, and the mark that says its blows
always land. They come off the **same resolution** the health, the rate and the domain come off, so
no path gives an entity some of the eight and not the rest.

**FR-2 — the damage pair and the always-hits mark come off one column group.** The damage base is
the definition's minimum physical column and the spread is its maximum less its minimum. A
definition whose damage-routing column selects the always-hits arm MUST carry that same pair **and**
the mark; every other value this contract admits MUST carry the pair and no mark. A definition whose
routing column selects a pair this tree does not model MUST be refused by name, and no world built.

**FR-3 — the difficulty adjusts the to-hit and the defence, and nothing else of the eight.** The
scenario setting MUST add its constant to the resolved to-hit and to the resolved defence. It MUST
NOT change either cadence number, either damage number, the absorption, or the always-hits mark, at
any of its three values.

**FR-4 — a placement that resolves to no definition carries the constructor's own eight.** A
placement given no table, one taking a humans or an npc arm, and one whose key names nothing MUST
each carry an attack charge of **8**, an attack relax of **4**, and **zero** on the other six. A
party member placed into a started mission MUST carry exactly the same eight, so the two
populations that resolve to no definition cannot come to differ.

**FR-5 — no reach is filled.** An entity MUST gain no per-entity reach and no reach is read from a
definition. How far a blow carries stays a property of the simulation and is unchanged.

**FR-6 — the canonical byte form does not move.** Every field FR-1 writes is already in the form at
its present offset and width. The form's version byte, every field's offset, every field's width and
the record length MUST all be exactly what they were. A world written before this story and one
written after are the **same version**, and a decoder MUST read both.

**FR-7 — the numbers are load-time state and nothing else moves them.** The eight are written once,
when an entity is built, and MUST NOT be written again by any later phase of a load or by any tick.

## Acceptance criteria

Every witness below is a Go test in `pkg/mapload` unless the row says otherwise. Rows marked
**owner** cannot be reached by any test in this tree and say why.

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | a table entry whose eight source columns hold eight distinct values, none of them equal to a constructor default | the map is loaded with it | the entity carries each of the eight, **compared one field at a time**, and its health, rate and domain are what they were before this story |
| **AC-2** | one entry at each admitted value of the damage-routing column, including the empty cell and the always-hits arm | worlds are built from each | the damage pair is `(min, max − min)` on every arm, and the mark is set on the always-hits arm and on no other |
| **AC-3** | one map and one table, loaded at hard and at normal | the two worlds are compared | to-hit and defence each differ by exactly the constant; the **other six of the eight are equal field for field**, and the test fails if any of the six moves |
| **AC-4** | the same pair, loaded at easy and at normal | the two worlds are compared | all eight are equal field for field |
| **AC-5** | three unresolved shapes — no table at all, a humans-arm placement, and a key naming nothing | each is loaded | each carries 8, 4 and six zeros, and the three carry the identical eight |
| **AC-6** | a started mission with a party of more than one | the world is built | every party member's eight equal an unresolved placement's eight in the **same world**, field by field |
| **AC-7** | a world built from a map and a table | it is marshalled and read back | the two are equal field for field and equal by digest, and the **version byte is the pre-story value** |
| **AC-8** | the two pinned load digests | each fixture is rebuilt | each hashes to its new pinned value, **and** overwriting the eight fields' bytes in every record of the new form with zeros reproduces the **pre-story digest exactly** — so the new pin is the old pin plus this story's bytes and nothing else |
| **AC-9** | a table entry whose damage-routing column selects an unmodelled pair | a world is built | it is refused, the refusal names the entry, no world is returned, and no entity exists carrying a partial set of the eight |
| **AC-10** | **owner** — a lawful install, mission 10, at each of the three difficulties | a developer-run tool prints every placement's eight numbers beside its class name | the six the game's own unit information panel displays agree with the panel for a unit the owner selects. The **two cadence numbers are not on that panel** and are witnessed by no run of any kind |

**Error cases:** AC-9 above, and the difficulty value refusal FR-3's setting already carries.

## Derived properties

**P-1 — completeness.** For any map, any table and any difficulty, every entity in the built world
carries either all eight of its class's numbers or all eight of the constructor's. No entity carries
a mixture, and no code path produces one.

**P-2 — invariant.** For any world this tree builds, advancing it any number of ticks leaves all
eight of every entity's numbers exactly as the load wrote them.

**P-3 — negative invariant.** For any input matching AC-9 or the difficulty refusal, no world is
returned and no entity is built — the caller is left holding nothing, not a partly filled world.

**P-4 — negative invariant on the form.** For any world, the byte form's version, record length and
every field's offset and width are unchanged by this story; a world produced before it decodes
without a migration.

**P-5 — determinism.** For any one `(map, table, difficulty)`, two loads in two processes yield
worlds equal field for field and equal by digest.

## I/O examples

An entry whose columns read `physicalMin 3`, `physicalMax 9`, `attackKind` empty, `toHit 40`,
`defence 12`, `absorbtion 2`, `attackChargeTime 11`, `attackRelaxTime 4` yields, at the normal
setting: base 3, spread 6, no mark, to-hit 40, defence 12, absorption 2, charge 11, relax 4. At the
hard setting the same entry yields to-hit 90 and defence 62, with the other six unmoved. With
`attackKind 3` it yields the same pair **and** the mark.

## Constraints

- `pkg/sim` gains no field, no constant and no behaviour. This story writes state that package
  already defines and already reads.
- The eight are signed and are range-checked nowhere, exactly as the rate and the health pair are.
  A definition's column value passes through whole.

**The value an unresolved placement carries** is an external-behaviour choice and the alternatives
were weighed:

| | what an unresolved placement carries | observable trade-off |
|---|---|---|
| **A — chosen** | the constructor's own eight: 8, 4, six zeros | every entity holds a cadence the original can produce. Costs: both pinned load digests move, including the table-free one |
| B | eight zeros | the table-free digest does not move. But a charge of zero floors to one tick, so the units this tree models *least* would fight roughly ten times faster than any shipped class, and a cadence of zero is a state no constructor and no column-driven load can leave |
| C | refuse a placement that resolves to nothing | a map and a table need not have been shipped together; refusing would make one unmatched key destroy a whole world |

## Out of scope

- **Equipment.** A class's equipment string is not read. The damage, absorption, defence, reach and
  cadence an equipped class ends up fighting at are not reproduced, and this is the named seam that
  moves all five together.
- **Reach as a per-entity value** (FR-5), and any per-class reach.
- The second and third damage components, the damage-kind reduction, and the elemental protections
  and resistances — the definition carries all ten, and nothing reads them.
- The humans-arm collection's own combat columns: a placement below the class-key floor keeps FR-4's
  eight rather than gaining a second stat source.
- Health regeneration and mana regeneration: both periods are carried on a definition and unread.
- The provisional spawn health, which stays exactly what it is — this story moves no health.
- Any front-end door: nothing gains a way to order an attack, so no loaded map can be made to fight
  by any run this story produces.
- The experience a kill pays, the treasure a corpse drops, and the corpse's dwell time.
