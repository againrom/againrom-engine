# Spec — 0120-ranged-combat

## Purpose

A ranged unit strikes from a distance instead of walking into contact, and the hero can be
given a bow. Most of the first half already stands; this story finishes the parts that feed it
and removes the two places where a ranged weapon is refused or discarded.

## Terms

**Weapons row.** One entry of the definition table's `Weapons` collection: a name and a
parameter array. The slots this story reads are the **attack type**, the **damage pair**, the
**to-hit**, the **defence**, the **range** and the **cadence pair** (charge and relax). An
empty cell is the sentinel -1 and means "the row states none".

**Ranged row.** A Weapons row whose attack type is 10 or above. Below 10 the row is a **melee
row**. The distinction is the row's own, not a name and not a class.

**Range** is the row's own cell; **reach** is how far a blow carries, in whole cells, as an
entity finally holds it. A weapon sets its bearer's reach to its range; a bearer with no
weapon has reach 1.

**Equip fold.** The step that turns "this character holds this weapon" into the numbers a blow
reads. It is an *addition* on top of what the bearer already has, not a replacement, except for
the cadence pair and the reach, which are assignments.

**Class equipment.** The trailing equipment strings a unit-class row carries. A name there is
`[shape ][material ]row`, optionally followed by a brace clause this story does not read.

**Trained skill.** The one weapon skill the party's hero is generated with. It selects the
literal weapon name character generation hands him.

## Scope

In: the equip fold's ranged arm; folding a unit class's own equipment; making the hero's
trained skill a choice rather than a constant; drawing a shot in flight.

Out: the simulation's reach arithmetic, its target scorers and its approach rule, all of which
already carry reach and are not touched. Out: any new field on a simulated entity, any change
to the serialized byte form, and any change to a hashed digest.

## The contract

**FR-1 — a ranged row resolves.** Resolving a weapon name whose row is a ranged row yields a
weapon rather than an error. The weapon carries its row's attack type, so a caller can tell the
two kinds apart; every other number it carries is scaled exactly as a melee row's is.

**FR-2 — the fold has two arms.** Folding a weapon onto a bearer:

- **always** assigns the bearer's reach from the weapon's range, and assigns each half of the
  cadence pair from the weapon's own half when that half is not an empty cell;
- **only for a melee row** adds the weapon's damage pair, its to-hit and its defence.

A ranged row therefore gives its bearer reach and cadence and nothing else. A bearer holding
one keeps the damage, to-hit and defence he had without it.

**FR-3 — a unit class's own equipment folds whole.** A placed creature whose class row names an
equipment item folds that item's weapon onto the definition under FR-2, in place of reading its
range alone. The first name in the row that resolves is the one that counts; a row that names
none, or none that resolves, is unchanged.

**FR-4 — no shipped reach moves.** For every unit class and every character in the shipped
table, the reach after this story equals the reach before it. FR-3 widens what else is folded;
it does not change the number FR-2 assigns.

**FR-5 — the trained skill is a choice.** The party hero's trained skill is settable once,
before a mission is loaded, and defaults to what it is today. Setting it to the shooting skill
gives the hero the bow that skill's literal already names, and every number derived from a
weapon — his damage, his to-hit, his defence, his cadence and his reach — follows from that
weapon through the same fold FR-2 defines. The front end offers the choice; nothing else in the
tree learns a new constant.

**FR-6 — a shot is drawn.** While an entity whose reach is above 1 is in its attack cycle
against a target it is not adjacent to, a mark is drawn between the two, at a position
interpolated along the straight line from attacker to target by how far through that cycle the
attacker is. It disappears with the cycle. A reach-1 attacker draws nothing new.

**FR-7 — the drawing tier learns nothing.** FR-6's mark is positioned from values the
simulation already holds and hands across the existing readout seam. No rule about attacks,
reach or cadence is restated in the tier that draws.

## Divergences

**D-1 — the third damage component is not built.** In the original a ranged weapon's damage
does not join the ordinary damage pair: it feeds a separate component with its own protection
selector, resolved by its own arm of the damage resolver. This tree has no such component, so
under FR-2 a ranged weapon contributes no damage at all and its bearer strikes with whatever
damage he had. For every shipped unit class this is exactly the behaviour the tree has today,
because the only shipped ranged row is carried by classes whose own row states their damage.
Disclosed rather than approximated: routing that damage into the ordinary pair would make a
shipped creature hit harder than the original, which is worse than making it hit as it does now.

**D-2 — a ranged weapon's to-hit is not reassigned.** The original replaces the bearer's to-hit
from another of its own fields on the ranged arm. This tree leaves the bearer's to-hit alone.
The field that would be copied is not modelled here at all, so copying it would mean inventing
a value.

**D-3 — the active skill is not modelled.** The original sets a bearer's active skill from a
melee weapon's kind and clears it for a ranged one. This tree has no active-skill field; the
skill term it would gate is already absent.

**D-4 — a shot's own art is not resolved.** The class registry names a projectile class, a
shoot delay and a per-direction shoot offset for each unit class. None is read. FR-6's mark is
this project's own diagnostic drawing, in the same spirit as the placement markers, and asserts
nothing about how the original drew an arrow.

**D-5 — a brace clause on an equipment name is still ignored.** A class equipment string may
carry a trailing brace clause. It is stripped and discarded, exactly as it is today.

## Acceptance

**AC-1** — a name whose row is a ranged row resolves to a weapon, and that weapon reports the
row's attack type; a name whose row is a melee row is unchanged in every field.

**AC-2** — folding a ranged weapon onto a bearer moves the bearer's reach and cadence and moves
neither his damage pair, nor his to-hit, nor his defence. Folding a melee weapon moves all of
them.

**AC-3** — a creature whose class row names a melee weapon arrives with that weapon's damage,
to-hit and defence added to its row's own and with its cadence assigned from the weapon; one
whose row names a ranged weapon arrives with its row's numbers and the weapon's reach.

**AC-4** — for every class and character built from a table of the shipped shape, the reach is
the same before and after this story.

**AC-5** — setting the trained skill to the shooting skill and then generating the party hero
yields a hero holding the bow that skill names, with a reach above 1; leaving it unset yields
exactly the hero the tree yields today.

**AC-6** — an attacker of reach above 1, mid-cycle against a target two or more cells away,
puts one mark on the drawing seam whose position lies on the segment between the two and
advances along it with the cycle. An adjacent attacker, or one of reach 1, puts none.

**AC-7** — the serialized byte form's version and every pinned digest in the simulation package
are unchanged, and the simulation package gains no field.

## Derived properties

**P-1** — because the fold assigns reach and adds everything else, folding is not idempotent
for damage and is idempotent for reach. Nothing folds twice; this is stated so that a later
caller that does will know which half it corrupts.

**P-2** — a bearer's reach is never 0: a weapon's range cell is the sentinel or a positive
number, the sentinel means 1, and a bearer with no weapon keeps 1.

**P-3** — FR-4 holds without a special case because the range slot is read the same way on both
arms. The only thing FR-1 changes about a ranged row is that the resolution now returns instead
of failing.

**P-4** — FR-6 draws nothing for the whole of a shipped mission that contains no ranged unit,
so the mark can be added without changing what any existing mission looks like.
