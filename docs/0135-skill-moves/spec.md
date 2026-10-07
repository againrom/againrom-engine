# 0135-skill-moves — a skill level moves by one, and says so

Intensity: **spec-anchored / static**. Terrain: **brownfield** in `pkg/sim`,
`pkg/data`, `pkg/mapload` and `pkg/game` — every area this touches already has
behaviour, and one of them has behaviour this story contradicts.

## Vocabulary

**Slot** — one of six skill positions, 0 to 5. Slot 0 is `General`; 1 to 5 are
the five shared skills, named for a weapon on one class and for an element on
the other.

**Level** — a whole number in `[0, 100]` held per slot.

**Experience** — a whole number held per slot. `S(n) = ftol((1.1ⁿ − 1) × 1000)`
is the curve joining the two, non-decreasing over `[0, 100]`. **`ftol`
truncates toward zero**, at every appearance in this document.

**Carrier** — an entity with a mana pool. It is the class discriminator.

**Gains** — a separate, per-entity fact from Carrier: whether this entity's
class earns experience **at all**. A creature does not; a person does. Both a
carrier and a non-carrier may be either.

**Owner slot** — the roster position an entity stands on. **Locked relation** —
one direction of the map's own relation matrix carrying its treaty bit. Both
are exactly what the two refusals `0125` FR-5 already states are about, named
here so this contract needs no other document.

**Credited slot** — the slot an award actually lands in, which is not always
the slot the award named.

**Award** — one act: a named slot, an amount, and an optional source entity.

## Amended elsewhere

`0125-experience-from-use` FR-8, FR-10, FR-11 and FR-14, and its AC-6, state a
model this story contradicts: that a level is derived from an experience and
that the simulation carries no level. They are corrected in place, in that
story's own `spec.md`, rather than left to disagree with this one.
`0127-first-spell` FR-3, FR-7 and SC-2 all state that the power's skill term is
zero and that a spell's school is read by nothing; FR-11 below delivers the one
and falsifies the other, and those three clauses are corrected the same way.

## Functional requirements

### The state

**FR-1** An entity carries **six skill levels**, one per slot, in slot order,
beside the six experiences it already carries. A level is state in its own
right: nothing in the simulation derives one from an experience, and after the
first raise the two no longer determine each other.

**FR-2** Every level value is carried whole and none is refused, on the same
ground the entity's Mind is: a definition column and a loadout bonus both reach
it, and a value this tree can produce must be a value it can read back. It is
canonical — it is in the serialized byte form and therefore in the digest.

**FR-3** A unit a map places and a party member minted from a character are
both born with the levels their own definition states, and with each slot's
experience equal to `S` of that slot's level exactly. A slot's first award of
one point or more therefore always raises it, once per slot per character.

**FR-4** A member carried from one mission into the next arrives holding the
six levels **and** the six experiences he ended the last one with, each
exactly. Neither is recomputed from the other at the boundary.

### The award

**FR-5** Nothing is awarded unless **all** of the following hold. Each is a
separate refusal and none is a clamp to zero — a refused award computes no
amount and writes nothing at all:

1. the awarding entity **gains**;
2. where a source entity is supplied, the two must not stand on the same owner
   slot, and the awarding entity's relation toward the source must not be
   locked. **Whether the source is still alive does not enter it** — see Out of
   scope, which is where that clause of the original is accounted for;
3. the credited slot must resolve — FR-7;
4. that slot's level must be **below 100**, and this is tested **before**
   anything is added, so a slot at 100 banks no experience either.

**FR-6** The amount is scaled by the awarding entity's own Mind first,
`gain = amount × (4 × mind + 30) / 120` truncated toward zero, and is then
capped at one level's worth of the credited slot's **current** level,
`S(n+1) − S(n)`.

**FR-7** Which slot is credited follows the class:

- a **carrier** takes the slot the award named, and refuses an award naming
  slot 0;
- a **non-carrier** refuses an award naming any slot but 0, and then credits
  the skill of the weapon in its hand, refusing that too when it is slot 0.
  An empty hand, and a weapon whose kind names no skill, both resolve to slot
  0 and are therefore refused.

Two consequences, and neither is softened: a spellbook carrier **earns nothing
from melee or missile combat**, and slot 0 — `General` — **is raised by
nothing**.

**FR-8** The capped amount is added to the credited slot's experience and to
nothing else. Then, and only if that experience now stands **strictly above**
`S` of that slot's **current** level, the level rises by **exactly one**. No
single award can raise a slot twice, however large the amount.

### The feeds

**FR-9** A landed blow makes an award naming **slot 0**, of the amount
`0125` FR-6 already states, **with the target as the source entity**, once the
target's health has already been reduced. Its own two caller refusals — a blow
that removed nothing, and a target already dead before it — are unchanged.

**FR-10** A cast that is applied makes an award naming the spell's own
**school**, of `(manaCost + 1) / 2` truncated — which is `ftol(manaCost × 0.5 +
0.5)` over the non-negative costs this build admits — with the victim as the
source entity. This build's cast has exactly one victim and is refused without
one, so there is no second victim and no victimless cast to answer for.

**FR-11** A spell's power reads the caster's level in the spell's own school:
`clamp(level + Mind − 30, 0, 100)`. A school outside the six slots contributes
a level of zero rather than refusing the cast.

### What follows a raise

**FR-12** A raise re-runs the derived-stat graph for the character it belongs
to, and the numbers a blow resolves against move with the level.

*What that reaches in this tree, stated rather than implied.* Ten numbers move
with the level: the damage pair, to-hit, defence, absorption, the two attack
times, always-hits, reach, and which slot the next blow trains. **Health, mana,
sight, speed and the five protections do not move until the next mission
begins**, where a member is rebuilt from his levels — the same derive produces
them, and nothing in a running mission can put them onto a unit already
fighting. And only the character whose pack is open moves at all; another party
member's numbers wait for the same boundary. Both limits are this tree's, not
the game's.

**FR-13** A raise puts **one row of text on screen**, naming the skill and the
level it reached, on the same transient log a pick-up writes to. A tick on
which no level moved posts nothing.

### The readout

**FR-14** Wherever a level is shown, the entity's own stored level is what is
shown. Nothing converts an experience into a level to produce it.

## Acceptance criteria

**AC-1** A non-carrier that gains, holding a weapon whose skill is not slot 0,
lands one blow on a hostile of another owner and ends the tick with that slot's
level exactly one higher and every other slot untouched.

**AC-2** That world replayed from its byte form holds the same six levels, and
two worlds differing only in one level hash differently.

**AC-3** A carrier that lands the same blow ends the tick with all six levels
and all six experiences unchanged.

**AC-4** No feed raises slot 0: a non-carrier whose weapon skill is slot 0
gains nothing from a blow, and a carrier's award naming slot 0 is refused.

**AC-5** A slot at level 100 gains neither a level nor a point of experience.

**AC-6** An award worth more than one level's worth of the credited slot raises
it by one and banks exactly `S(n+1) − S(n)`.

**AC-7** A cast by a carrier raises the spell's own school when the amount
crosses that slot's boundary, and a spell's damage rises with that school's
level through FR-11.

**AC-8** A refused award leaves both entities byte-identical and draws no
randomness, so no other rule's outcome reveals whether one was refused.

**AC-9** The announcement states the skill's name and its new level; a tick
with no raise posts nothing.

**AC-10** Driven against both lawful roots, the tenth mission's party member
raises a skill level, and the drive prints it.

## Properties

**P-1** The curve the simulation reads and the curve the character graph reads
answer the same integer at every level 0 to 100. `S` has one definition and
two implementations, and neither may drift from the other.

**P-2** The simulation stays free of floating-point arithmetic — no float
identifier, literal or import — and takes on no new dependency.

**P-3** No single award raises a slot by more than one, at any amount, at any
level.

## Out of scope, and cut deliberately

- The **kill feed** `vt+0x60`. Decoded in amount and in slot rule; **no claim
  names its caller**, so there is no trigger to build without authoring one.
- The **loss at death** and the **purchase for gold**. Both decoded, both whole
  stories, and neither is needed for a level to move.
- The **more-than-one-participant** fifth of FR-6's cap. This tree has no
  session object carrying that flag, and the flag's meaning is not High.
- **The killing blow's exemption from FR-5.2.** The original applies both
  source refusals **only while the source is still alive**, and a killing blow
  arrives with the victim already below zero — so in the original a killing
  blow on an ally, or on a treaty-protected owner, pays. It is not built: it
  reverses `0125` FR-5.4 and FR-5.5 on the one blow those refusals matter most
  on, it makes killing one's own ally profitable, and nothing this story is for
  needs it. This is a **disclosed divergence**, not an omission.
- **The original's experience-to-level routine.** Undecoded, and no longer
  needed by anything here.
- **Widening the derive's door.** Health, mana, sight, speed and the
  protections keep having no live setter; see FR-12.
