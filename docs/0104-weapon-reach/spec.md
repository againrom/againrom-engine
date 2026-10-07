# Spec — a weapon's range becomes the reach a blow carries

## Terms

**Reach** — how far a blow carries, per actor, in whole cells. An actor with reach 1 strikes the
eight cells around it and the one it stands on.

**Strike distance** — the number a reach is compared against. It is not a plain Chebyshev delta: it
is floored at 1, so an actor standing on its victim's own cell is at distance 1, not 0.

**Equipment string** — a trailing name on a definition row, of the form
`[tier ][material ]weapon[{suffix}]`, naming a row of the weapons table.

**Range column** — the weapons table's own per-row range. A row whose cell is empty carries `−1`
and means a range of 1.

## Why

Archers fight in melee. Every unit in this build closes to an adjacent cell before it strikes,
including the ones the game arms with bows, sonic attacks and siege engines, because reach is a
compile-time constant of 1 shared by every entity. The visible half is the walk: an attack order
against a distant victim walks the attacker until it is touching, so a Goblin archer crosses open
ground to swing a bow, and a Ballista behaves like a spearman.

The value that fixes it is already in the files this build reads. A unit's definition row names its
weapon; that weapon's row carries a range; the original sets an actor's reach to 1 at construction
and the equip that follows raises it. This story reads that chain and makes reach a per-actor field.

## Scope

**In scope.** Deriving reach from a definition row's own equipment string. Carrying it on a
simulation entity and through the byte form. The strike gate and the stop distance of an approach.

**Out of scope.** The equipment channel — inventory, containers, slots, picking up, dropping,
wearing, drawing what is held. The shield arm of the same construction loop (no shipped unit row
names one). Anything else a weapon carries: its damage pair, its to-hit, its attack cadence. What
a `{…}` suffix on an equipment string means. Projectiles. Actor footprints larger than one cell.
Any reach the group-order layer might consult; it consults none today.

## The contract

### The data tier

**FR-1** A function resolves an equipment string to its weapon's **range** alone, for **any** attack
type, and reports whether the name resolved. The result is the row's range column, or **1** when
that cell is `−1`. It computes no damage, no to-hit and no cadence, and it refuses no row on the
grounds of its attack type.

**FR-2** That function and the existing full weapon resolver take the same first step: a trailing
`{…}` suffix is removed before the name is matched, along with any whitespace it leaves. The step
has ONE implementation. Everything after it — the longest tier-name prefix, then the longest
material-name prefix, then the remainder as a table row — is unchanged.

**FR-3** The combat block a definition hands a world builder carries **Reach**. A definition that
resolves no weapon yields **1**, the value the game's own constructor sets before any equip runs.
Both definition kinds fill it, so a builder reads one field however a placement resolved.

### Placement

**FR-4** A placed **unit** takes its reach from its class row's trailing equipment strings: the
first non-empty string that resolves under FR-1 supplies it. A row that names nothing, and a row
whose every name resolves to no table row, leaves reach at 1.

**FR-5** A placed **person** takes its reach from the weapon the loader already resolves for it.
A person carrying no weapon this build can resolve has reach 1.

**FR-6** Nothing else moves reach. The difficulty adjustment scales none, and no later stage of a
load raises or lowers one. Reach is written once, where a definition becomes an entity.

### The simulation

**FR-7** An entity carries **Reach**, one unsigned byte, and the legal set is **1 to 255**. Zero is
refused by the constructor and by the decoder alike, in the same words: the strike distance is
floored at 1, so a reach of 0 is an actor that can never strike anything and there is no value to
fold it onto. An entity built with no reach named is built with **1**.

**FR-8** A strike distance function stands beside the existing distance helpers and replaces none of
them. It takes two entities and answers:

```
d = (max(|dx|, |dy|) << 8) - (((1 + 1) << 7) - 0x100)   the footprint term is 0 at one cell each
answer = 1                                              when d <= 0x180
answer = (d + 0x40) >> 8                                otherwise
```

It is integer arithmetic throughout and reads nothing but the two entities' cells. It is written in
those terms rather than as the `max(1, Chebyshev)` it equals, because the footprint term is the one
part this build cannot yet supply and the shape has to survive until it can. A test asserts the
equality over a range of separations, so the equivalence is measured and not asserted.

**FR-9** A blow lands only when the strike distance from attacker to victim is at most the
**attacker's own** reach. The victim's reach is not consulted.

**FR-10** An approach stops at the same predicate. An attacker walking toward a victim it cannot yet
strike stops as soon as the strike distance is within its reach, turns to face, and strikes from
there. It is the same predicate as FR-9 and not a second comparison, so "walked close enough" and
"close enough to strike" cannot come apart. Everything else about the walk is unchanged: the
destination is still the victim's current cell, re-read every turn, and a dead or vanished victim
still ends it.

**FR-11** The byte form gains reach and the version byte rises to **23**. Reach is **one byte** at
the tail of the entity record, so the record grows by one and every offset after the entity block
moves; the version byte itself is at offset 0 and does not. A decoder refuses any version but 23,
exactly as it refuses every other version today, and a form carrying a reach outside 1 to 255 is
malformed rather than a variant.

The bump is not optional. Reach is derived from a definition table a saved form does not name and
cannot reach, so a form that omitted it would hand back a world whose archers had become spearmen,
silently, and whose digest then means something else.

## Acceptance

**AC-1** Against an installed root, and against both where both can be reached: every unit class row
that names an equipment string resolves to a weapons row with no residue; the number of rows whose
derived reach exceeds 1 is recorded, with the distinct reach values and the class names carrying
each.

**AC-2** A map placement of a bow-carrying class produces an entity whose reach is that bow's range,
through the loader and not through a fixture that sets the field.

**AC-3** A bow-armed attacker ordered against a victim several cells away stops at its reach, faces
it, and lands a blow without ever standing adjacent. A spear-armed attacker in the same situation
still closes to an adjacent cell.

**AC-4** A world holding entities of several reaches round-trips through the byte form unchanged. A
buffer at the previous version is refused; one whose reach byte is 0 is refused; one whose reach
byte is 255 is accepted.

**AC-5** Every literal digest and byte-form offset pin in the tree is re-pinned from a run, and the
old and new values are recorded together.

**AC-6** The tenth mission is driven from its own start and its outcome recorded. A change from the
outcome recorded before this story is reported, not repaired here.

## Properties

**P-1** Reach is a property of an entity and of nothing else: no global constant, no per-class
lookup at strike time, no consultation of a definition table from inside the simulation.

**P-2** The strike distance depends on the two cells alone. Two entities at the same separation
answer the same distance whatever their reach, class or health.

**P-3** Reach is never below 1 anywhere in the tree: not after construction, not after a decode, not
after a load, not after a difficulty adjustment.

**P-4** An entity whose definition names no weapon has reach exactly 1, and behaves exactly as this
build behaved before this story.

**P-5** The stop distance and the strike gate are one predicate, with one implementation and no
second spelling of the comparison anywhere.

**P-6** The simulation stays deterministic: the distance arithmetic is integer, reads no clock and
draws no random number.

**P-7** An equipment string this build cannot resolve is not a load failure. The entity is bare and
the load continues.

**P-8** The byte form is self-describing for reach: a decoder needs no definition table, no install
and no map to reconstruct an entity's reach from the bytes.
