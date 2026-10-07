# Provenance — a placed unit's combat numbers

Research is the `research/` submodule at the pin this branch carries, **`f56bb38`**, unbumped for
the whole story. Claims are cited by **id** and never by experiment, so a later amendment or
retraction reaches this story through the ledger that carries it.

## The eight fields, and what each rests on

Every row was read whole in its ledger, not through a summary. Slot numbers are the Units
collection's parameter slots under the schema law, and the store column is the actor field the
claim names.

| spec value | Units column (slot) | claim(s) | confidence |
|---|---|---|---|
| attack charge | `attackChargeTime` (17) → `+0x134` | `UNIT-STREAM-001`, `HERO-CADENCE-023`, `UNIT-COMBAT-006` | **High** |
| attack relax | `attackRelaxTime` (18) → `+0x135` | same three | **High** |
| to-hit | `toHit` (14) → `+0xa6`, read by the resolver as its block's first word | `UNIT-STREAM-001`, `HERO-DAMAGE-022` | **High** |
| defence | `defence` (15) → `+0xbe`, the value the hit test compares against | `UNIT-STREAM-001`, `HERO-DAMAGE-022` | **High** |
| absorption | `absorbtion` (16) → `+0xc0`, subtracted flat from the physical component | `UNIT-STREAM-001`, `HERO-DAMAGE-022`, `UNIT-COMBAT-006` | **High** |
| damage base | `physicalMin` (11) → `+0xb4` via the routing switch | `UNIT-STREAM-001`, `HERO-DAMAGE-022`, `HERO-DMG2-029` | **High** |
| damage spread | `physicalMax − physicalMin` (12 − 11) → `+0xb5` | same three | **High** |
| always-hits mark | `attackKind == 3` (13) sets the actor's own flag bit | `UNIT-STREAM-001`, `HERO-DMG2-029` | **High** |

Three supporting rows carry the shape the eight sit in:

- **`UNIT-DERIVE-003` (High)** — a non-hero's recompute slot exists, is 26 instructions, writes one
  mana field and derives nothing. **The template numbers *are* the stats**: none of the hero arm's
  derived-stat graph has a counterpart here. This is what makes "copy the column" the whole of the
  contract rather than the first step of one.
- **`UNIT-DIFF-002` (High)** — the two streamers side by side. Absorption, the protections, the
  resistances and the damage pair are **units-only**; the cadence pair, the health pair, the rate,
  the to-hit destination and the four stats are written by **both**. This is why a placement below
  the class-key floor cannot borrow the units arm's numbers.
- **`UNIT-CTOR-004` (High)** — what a non-hero holds before any template is read: **attack charge 8,
  attack relax 4, reach 1**, health 30, rate 10, and every other constructor value. Each is a named
  store with its immediate. Every empty cell in a template leaves exactly these. This is FR-4's
  source.

## The difficulty adjustment

**`UNIT-GATE-013` (High)** — the placement spawner runs a three-way gate on the actor it has just
built: nothing at the middle value; the health maximum scaled down at one; and at the other,
**to-hit `+= 50`, defence `+= 50`** and the health maximum scaled up, both adjusting arms then
assigning the current health from the maximum. The row states the consequence directly: the resolver
needs no per-hit gate, but a **spawner must carry it**. It reads at instruction level, the two
constants are read out of the image and censused over the whole file, and the humans/units split is
fixed by an allocation size and an unconditional jump.

FR-3's negative half — that it moves **none** of the other six — is the same reading: the row
enumerates every store the gate makes, and the damage pair, the absorption and the cadence pair are
not among them.

`UNIT-PANEL-010` (High for the copy map, the key and the arithmetic) is cited **only** for AC-10:
the information panel copies to-hit, defence, absorption and the damage pair among its stores, which
is what makes an owner comparison possible at all — and it does **not** copy the cadence pair, which
is why AC-10 says that pair is witnessed by nothing. Its own **scope** clause was superseded and
that retraction was read: the arithmetic it described is not display-only but the spawner's, which
is `UNIT-GATE-013` above. Nothing here rests on the superseded reading.

## Reach: a positive absence

`UNIT-COMBAT-006` (High for the field↔column bindings and the three weapon writes) states reach's
whole source: **1 from the constructor**, then `+= weapon range − 1` per equipped weapon. There is no
reach column — `UNIT-STREAM-001`'s slot map accounts for all 38 streamed slots and none of them is a
reach. So a reach derived from the class alone is the constant 1 for every class, and FR-5 is a
measurement rather than a simplification.

The row's reach **histogram** — 38 of 56 classes at reach 1, the rest between 4 and 20 — is the
scale of the equipment divergence, not of this story's. The figure's history was read: an earlier
publication said **30**, was **REFUTED**, and had carried **High** while believed, inside the same
cell as the instruction-level bindings. The mechanism never moved; only a sentence written beside a
measurement did. Nothing in this story depends on the count, and the corrected figure is quoted
rather than leaned on.

## Ours by choice

**OC-1 — an unresolved placement carries the constructor's own eight (FR-4).** The *values* are
decoded (`UNIT-CTOR-004`, High); the *decision to apply them to this population* is ours, because
the population does not exist in the original — every one of the corpus's 8094 placements resolves
to something (`MISSION-ARM-006`, High: 6672 units, 1405 by definition id, 15 by npc, 2 by type id).
Our unresolved set is the 1422 that take a humans arm plus anything a table does not match, and it
is an artefact of not modelling that arm.

*The limit it implies:* a placement this tree does not resolve fights on the **unit** constructor's
cadence rather than on its own collection's row. *Its class:* a stand-in for an unmodelled arm, not
a hardcoded rule — it disappears when the humans arm is modelled. *Would lifting it change a shipped
file's bytes?* No: lifting it means reading more of a file already read, and writes nothing.

The alternative — eight zeros — was rejected on evidence rather than taste. A charge below one is
floored to one by the cycle, so a zero cadence yields a blow roughly every one to four ticks against
a shipped class's twelve to fifty-five (`HERO-CADENCE-023`'s worked figures), and a cadence of zero
is a state neither the constructor nor any column-driven load can leave. The cost is recorded in
`spec.md`'s own constraint table.

**OC-2 — the adjustment is applied to the definition, not to the constructed entity.** The original
applies it to the actor **after** creation is complete, which includes the equipment fold
(`formats/unit/format.md`'s creation order, steps 8 and 10; `UNIT-CTOR-004` for the fold's position,
`UNIT-GATE-013` for the gate's). This tree equips nothing, so the fold is the identity and the two
orders give the same eight numbers. **This equivalence expires with the equipment seam**: whatever
lands it must fold **before** the adjustment, or a hard-difficulty to-hit and defence come out wrong.
Recorded here rather than left to be rediscovered.

**OC-3 — carried forward from the previous story, unchanged and now re-checked.** A blow whose
damage is not positive after absorption removes nothing; the damage-kind reduction is not modelled;
the second and third damage components are not modelled. The second and third remain unreachable on
shipped data — `HERO-DMG2-029` (High for the resolver's components, the switch and the column
census) finds the routing column is the empty cell on 48 shipped classes and the always-hits arm on
8, and **1 or 2 on none**, so the component's own gate skips it. The damage-kind reduction remains
the identity: its index is the attacker's active-skill byte, which only a **melee weapon** sets, and
index 0 is filled by no column and zeroed by the constructor (`UNIT-COMBAT-006`, High). This tree
equips nothing, so every attacker it builds indexes 0.

The **first** of the three changes status without changing text. Until this story every loaded map's
absorption was zero, so the branch was unreachable outside a hand-built world; filling absorption
makes it reachable. See *Open* below.

## Read and deliberately not modelled

Each was read and each is a positive statement that it cannot reach one of the eight.

- **The to-hit copy into the skill slot** (`UNIT-DIFF-002`, High). The units streamer writes to-hit
  to its own field and then copies it into the first skill word. The resolver reads the **field**,
  not the copy (`HERO-DAMAGE-022`, High, the block's first word), and a non-hero derives nothing
  (`UNIT-DERIVE-003`), so no skill term reaches a to-hit on this arm.
- **The tier-4 and the Dragon/Daemon creation arms** (`formats/unit/format.md`, creation step 7).
  The first assigns the live skill words; the second sets flag bits **1 and 2**. The always-hits
  mark is bit **4** of the same byte, so the two sets are disjoint and neither arm can set or clear
  it. Neither touches any of the eight.
- **The spellbook, the treasure columns, the order-block columns, the two regeneration periods, the
  five protections and the five resistances.** All are decoded and all are carried on a definition;
  none is read by the blow resolution this tree implements.

## Open — requests to research, none of which gate this story

**R-1 — publish the shipped Units table's per-class combat columns as a claim.** They are measured
and committed, but inside an experiment's evidence, so they are not citable and this story cannot
say what any named class's eight numbers are. Without them, no statement of the form "the first
mission's placed brigands can be killed" has evidence behind it, and AC-10 is an owner comparison
rather than an assertion.

**R-2 — is the physical damage component clamped at zero before it is subtracted from health?**
`HERO-DAMAGE-022` (High) publishes the arithmetic order — roll, hit test, flat absorption, then the
resistance factor — and `HERO-DMG2-029` says explicitly that the **second** component's reduction is
clamped at zero, which makes the first component's silence conspicuous rather than settled. If it is
unclamped, a blow absorbed past zero heals its victim. OC-3 keeps the previous story's answer and
this story adds no assertion either way.

**R-3 — publish the Humans streamer's slot → field map as a claim.** `UNIT-DIFF-002` gives it by
difference from the units map and `HERO-CADENCE-023` names the cadence pair's two humans slots, but
no row publishes the whole map. Until one does, a placement below the class-key floor cannot be
given its own numbers without inventing the ones that are not published.

**R-4 — the equipment seam's remaining gap.** The equip routines and their arithmetic are published
(`HERO-EQUIP-017`, High, each fill a named store and each guard an instruction), and the equipment
string's population is censused (`UNIT-EQUIP-005`: 26 of 56 shipped classes carry exactly one, none
carries two, none names a shield). What is **Medium** there is the name grammar — a corpus fit of 26
strings against three collections, with the parse routine itself unread. That is the gap a story
implementing equipment would open, and it is named now rather than discovered then.

## What this decode limits, and what customising it would cost (G2)

A placed unit's eight combat numbers are a function of exactly three things: the installed
definition table's Units row selected by the placement's class-key pair, the scenario difficulty,
and nothing else. In particular they are **not** a function of the map. The type-6 placement record
is accounted for byte by byte (`ALM-UNIT-040`, High for the read map — 27 reads summing to exactly
70) and carries no stat field of any kind, so a per-placement override cannot be expressed in a
shipped map at all.

The consequences for customisation, stated rather than left implicit:

- **Changing a class's numbers means changing `Data.bin`'s bytes.** That is a shipped file, and the
  Units collection is byte-for-byte identical between the two shipped roots on all 56 × 55
  parameters (`DAT-SCHEMA-007`), so any divergence there is detectable rather than silent.
- **Per-placement customisation changes no shipped file's bytes**, because there is nowhere in a map
  to put it: it needs a sidecar this project would author. The loader's seam is already the right
  shape for one — the definition is resolved and adjusted in a single place, and a sidecar would
  substitute at exactly that point.
- **The limit this story adds is data-driven, not hardcoded.** No number among the eight is written
  into shipped source; the two that look like constants (charge 8, relax 4) are reached through the
  definition tier's own constructor defaults and belong to it.

## Removed

Nothing was removed from the contract. Two candidate scopes were considered and cut into the named
seam rather than half-modelled: **reach as a per-entity value** (there is no column, and equipment
is what moves it) and **equipment itself** (which moves five of the eight, on the classes that carry
it). Cutting them together is deliberate — they are one seam, and a story that lifted reach alone
would give a class its armed reach and its unarmed damage.
