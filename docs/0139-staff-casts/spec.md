# 0139 — a caster's staff casts instead of striking

**Intensity:** spec-anchored / static. **Terrain:** brownfield.

## Why

A weapon may carry a spell. A fighter holding one carries it as a rider on his strike; a **caster**
holding one does not strike at all — his attack *is* the cast. The tree built the opposite: a
generated mage's staff resolved to a plain melee row and swung, and the spell written into the
item's own name was parsed by nothing.

## Scope

**In:** a weapon carries a spell id and a level; a caster wielding one casts it in place of every
strike, at range, without a roll; the cast reaches the same damage arithmetic a commanded cast
already reaches; item effects and their delayed deaths train the credited caster or fighter through
the ordinary skill sink; a headless way to drive a generated caster into a fight.

**Out of scope:** any shop or trade; the elemental school as an **element**, resistances and
protections; a spell's flight time. The fighter rider and point and area applications are in scope
only where they produce item-spell training or surviving kill attribution.

## Functional requirements

**FR-1 — a weapon carries a spell.** An item name may end in an attachment of the form
`{castSpell=TOKEN:LEVEL}`. A weapon resolved from such a name carries that spell's **id** and that
**level**; one resolved from a name without the attachment carries neither. The attachment plays no
part in which Weapons row the name resolves to, and none of the resolved row's own numbers changes
because of it.

**FR-1a — what the two halves mean.** `TOKEN` names the spell. `LEVEL` is the spell's **power** when
this weapon casts it. Both are *authored* readings: the source establishes that the runtime binds an
id and that the power is the item's own, and does not establish which half of the attachment feeds
which. `LEVEL` is a run of decimal digits and nothing else — no sign, no separator, no surrounding
space. A malformed attachment — no `:`, a `LEVEL` that is not such a run, a token that names no row
— leaves the weapon carrying **no** spell rather than half of one.

**FR-1b — a token names a row.** `TOKEN` matches a `Spells` row whose name is the token with every
`_` replaced by a space. The match is exact after that substitution, and where two rows carry one
name the **first** wins — the same tie-break by table order every other name search in this build
makes.

**FR-2 — the trigger.** An actor's attack becomes a cast when **all** of the following hold: it is
resolving an attack order; it holds a weapon; that weapon carries a spell; and it is a **caster**.

**FR-2a — a caster is an actor with a mana pool.** No second predicate, no class list, no flag on an
item.

**FR-2b — and the spell must name a loaded row.** A weapon whose spell id is in no loaded spell
table does not trigger. Such an actor attacks as if the weapon carried no spell at all.

**FR-3 — the cast replaces the strike.** When FR-2 holds, the strike is **never resolved**: no blow,
no damage roll, no experience, no facing change beyond what an approach already makes. The actor
instead arms a cast, counts a wind-up down, and releases the spell at the end of it.

**FR-3a — the trigger is live, in both directions.** It is asked at the two moments an attack cycle
can commit to an outcome: when an actor with nothing owed begins a cycle, and when an actor's
wind-up runs out. An actor that becomes eligible while already winding up toward a blow does **not**
land that blow; one that stops being eligible while winding up toward a cast does **not** release
one. Neither leaves residue: the actor begins a fresh cycle of whichever kind it is then eligible
for.

**FR-3b — a divert costs the generator exactly what a blow costs it.** An advance on which an
actor's wind-up runs out draws the same number of random values whatever that advance then does —
strike, refuse the strike, release, refuse the release, or divert. How many values the simulation
has drawn therefore depends on which advances happened and never on what any of them decided.

**FR-4 — a released cast rolls nothing but damage.** No to-hit roll is taken, no absorption is
subtracted, and no part of the weapon's own damage pair is delivered. A caster with a spell-carrying
weapon therefore never misses, and this is a consequence of the branch rather than a property of the
weapon.

**FR-5 — the reach test changes hands.** The melee reach test gates the strike and continues to.
While FR-2 holds, what decides that an actor has closed on its victim is the **cast's** admission
distance, not the weapon's reach — so a caster stops at spell range and casts from there instead of
walking into contact. This is asked **while the actor is still walking**, not only once it has
arrived: an eligible actor never approaches nearer than the cast admits.

**FR-6 — the weapon survives.** Releasing a weapon-borne spell destroys neither the weapon nor its
spell. The actor may release again on its next cycle, indefinitely.

**FR-7 — the power is the level.** A weapon-borne cast's power is the attachment's own level,
carried whole. It is **not** derived from any statistic and is **not** clamped — the source states
this path skips the clamp every other power passes through.

**FR-8 — the damage.** A released cast's damage is drawn from the spell row's own two damage columns
scaled by that power, by the same arithmetic a commanded cast already uses, and applied to the
victim's health by the same subtraction.

**FR-9 — nothing is charged.** A weapon-borne cast costs no mana. A caster with an empty pool casts
exactly as one with a full pool does. The actor's spellbook is not consulted either: the weapon
carries the spell, not the actor.

**FR-10 — a cast is refused, not half-applied.** A release whose row is not damaging, whose row does
not target a unit, whose victim is not alive or is the caster itself, or whose victim stands beyond
the cast's admission distance, changes nothing at all — and the actor's cycle continues exactly as
if it had released.

**FR-11 — the wind-up.** The wind-up a cast counts down is the actor's own attack charge, and the
recovery after a release is the actor's own relax, drawn exactly as a strike's is. A caster's attack
cadence therefore equals the cadence of the swing his cast replaced.

**FR-12 — the admission distance.** A cast admits a victim within the spell row's own maximum range,
measured in whole cells the same way every other separation in the simulation is measured.

**FR-13 — every armed actor gets it.** The spell a weapon carries reaches an actor by **every route
this build already carries a weapon's other numbers along**, and by no route of its own: a generated
party member at mission start, a placed person or creature whose row names such a weapon, and an
actor whose equipped weapon changes during a mission. A route this build does not have is not one
this story adds.

**FR-14 — a headless door.** A tool can open a mission with a party whose hero is generated as a
**caster**, and drive him into an attack, without a screen.

**FR-15 — the new state is canonical.** The spell an actor's weapon carries, its level, and the
casting stage of its cycle are simulation state on the same terms as the numbers beside them: they
are carried by the world's byte form, they enter its digest, a form carrying them reads back
identical, and a form carrying a casting stage no tick could have produced is refused rather than
repaired.

**FR-16 — item training follows accepted effects, not releases.** A weapon-borne release receives
no immediate half-mana award. Positive direct damage and Drain transfer, accepted Slow, Stone
Curse and Curse pseudo-damage, every nonzero signed Poison tick, and delayed credited kills feed
the existing sink separately. A caster credits the actual spell row's school; a fighter credits
the currently equipped weapon skill. The sink requires TypeID in `[0x21,0x40)`, preserves signed
amounts below its positive cap, and applies the ordinary Mind, relation and skill-100 rules. A
refused effect credits nothing.

**FR-17 — delayed credit is surviving attribution state.** Direct damage records its source before
the effect envelope finishes. An admitted non-Defensive PointEffect then records the actual spell
id; an admitted AreaEffect does the same for every nonzero-domain target. Point retains prior credit
for an ownerless defined caster and clears it for a missing definition. Area clears it for either
loss. Drain creates neither envelope, and later Poison ticks do not refresh the id seeded by the
initial area application, so intervening effects may retain, clear or replace the eventual credit.
When an owned victim first crosses below zero, the surviving source receives `XPValue/2` through
the same sink. A mage resolves the saved spell id to its school; a fighter uses its current weapon
skill. The source reference, presence bit and signed spell byte are canonical entity state at the
tail of simulation form 55. Form 54 is refused explicitly, and malformed or dangling references are
not repaired by the decoder.

## Acceptance criteria

- **AC-1** — every `castSpell=` token in both lawful installs resolves to a `Spells` row under
  FR-1b. Measured, not assumed.
- **AC-2** — a weapon resolved from a name with an attachment carries the named spell's id and the
  stated level; the same name without its attachment carries neither, and the two agree on every
  other field (FR-1).
- **AC-3** — a malformed or unresolvable attachment yields a weapon carrying no spell, and the
  weapon still resolves (FR-1a).
- **AC-4** — an actor with a mana pool, a spell-carrying weapon and an attack order resolves **no**
  blow: the victim's health falls by an amount drawn from the spell's columns, and the actor's own
  damage pair is not part of it (FR-2, FR-3, FR-4, FR-8).
- **AC-5** — the same actor without a mana pool resolves an ordinary blow, and so does the same
  actor with a weapon carrying no spell (FR-2, FR-2a).
- **AC-6** — an actor whose weapon's spell id names no loaded row resolves an ordinary blow (FR-2b).
- **AC-7** — a caster whose victim stands beyond the weapon's melee reach but within the spell's
  range stops walking and casts (FR-5, FR-12).
- **AC-8** — repeated cycles keep casting: the weapon and its spell are still there after a release,
  and the second release is at the same power as the first (FR-6, FR-7).
- **AC-9** — a caster with zero mana and an unknown-to-him spell still casts, and his mana is
  unchanged after the release (FR-9).
- **AC-10** — an actor that becomes eligible while charging never resolves the blow that charge was
  loaded for, and one that stops being eligible while winding up a cast releases nothing (FR-3a).
- **AC-11** — a refused release leaves the world's hashed state, other than the actor's own cycle,
  byte-identical (FR-10, FR-15).
- **AC-11a** — two worlds identical but for one actor's eligibility, stepped the same number of
  ticks, have drawn the same number of random values (FR-3b).
- **AC-12** — a released cast's damage reproduces the commanded cast's arithmetic at the same power
  and row (FR-8).
- **AC-13** — a generated mage opened through the headless door attacks a unit at range and removes
  health without a to-hit roll (FR-13, FR-14).
- **AC-14** — the world's byte form round-trips an actor carrying a spell, a level and a casting
  cycle, and refuses a form whose casting cycle no tick could have left (FR-15).
- **AC-15** — a token matching no row, a level that is not a run of digits, and an attachment with
  no `:` each leave a weapon that still resolves and carries no spell (FR-1a).
- **AC-16** — staff Stone Curse raises Earth by its accepted 3%-health pseudo-damage event without
  a half-mana award; a direct and area item spell pay per positive damage application; signed
  Poison can reduce persisted XP; a fighter's rider credits its current weapon skill; low-TypeID
  Humans and refused effects receive nothing (FR-16).
- **AC-17** — direct and delayed deaths pay exactly once from the final surviving attribution;
  later point attribution redirects the award, a fighter receives it in the current weapon slot,
  Drain and a refused effect retain history, Point and Area apply their distinct clear rules, and a
  Poison tick can kill under an intervening source without restoring id 8. Attribution round-trips,
  distinguishes the digest and refuses invalid form-55 tails (FR-17).

## Success criteria

- **SC-1** — a generated mage's first attack in a real mission is a cast at range, not a swing.
- **SC-2** — no existing behaviour of a non-caster, or of a caster holding no spell-carrying weapon,
  changes.

## Divergences, disclosed

- **D-1** — a released cast lands on the tick it is released. The original delays a delivery-system
  spell by distance over its own speed column; that delay is modelled nowhere in this tree.
- **D-2** — the cast range term the original adds from the power is not applied. It is zero for every
  shipped staff level below 30 and is absent from the commanded cast too, so adding it on one path
  alone would make the two disagree.
- **D-3** — the item destroyed at release when its kind is the point-aimed one is not modelled: this
  tree has no item kind on an entity's weapon, and nothing it can build is that kind.
- **D-4** — the school a released spell belongs to names the skill its accepted event trains and the
  temporary damage kind written before an effect envelope records the actual spell id (FR-16,
  FR-17). Nothing reads it as an **element**: no resistance, no protection and no
  damage term turns on it, exactly as a commanded cast leaves it. *(Amended: this divergence
  originally said the school was read by nothing at all.)*
- **D-5** — a weapon known only by its item code carries no spell. The code holds a material, a
  shape and a row and has no field an attachment could live in, so a staff acquired as a bare code —
  off the ground, off a corpse — is a staff whose spell this build cannot recover. A weapon still
  held as the object it was resolved from keeps its spell across an equipment change.
