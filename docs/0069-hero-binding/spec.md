# 0069 — the hero the script cannot see

Intensity: rigour **high**, ambition **low**. Terrain: brownfield — the reference band, the
resolver and the fields it fills all exist and are correct; nobody fills them on the path a player
takes. Rigour is high because the compiled script is part of the world's hashed state.

## The problem

A map's mission script names units by an id that falls in one of three bands. One band is a **hero
ordinal**: it does not name anything the map carries, because a campaign map places no hero — the
party already exists and the load only positions it. Such a reference can be resolved only by the
tier that assembled the world, and on the path that starts a campaign mission nothing does.

Every hero-band reference in every shipped mission therefore resolves to nothing. A check whose
reference does not resolve measures nothing and **writes no register**, so the register a trigger
reads holds zero.

For mission 10 that is not a missing win but a **false** one. Its only winning trigger holds when
the hero is within a short distance of a point **and** a mission variable is non-zero. The first
clause's register is never written, so it holds zero, and zero satisfies "within". The mission
therefore declares victory on the second clause alone — with the hero anywhere on the map.

## Scope

In: a started campaign mission's script compile resolving hero ordinal 1 to the entity that
mission's own party hero became.

Out: **ordinals above 1.** The band is a subscript and this tree binds subscript 1 only; every other
ordinal names nobody and stays unresolved. What the higher ordinals name is undecided, and this
story does not decide it. That is a disclosed gap, not an oversight: 61 of the shipped corpus's 196
hero-band references are above ordinal 1 and remain unresolved after this story.

Out: the static name-band, which no shipped map references. Out: any change to which script arms are
implemented — the arms mission 10's win needs are already among them.

## Functional requirements

**FR-1 — a started mission resolves the hero.** When a campaign mission is started with a party,
every reference in its script naming hero ordinal 1 resolves to a single entity, and no such
reference is reported unresolved.

**FR-2 — it resolves to the hero and to nothing else.** That entity is the one the start placed as
the party's **first** member: the same entity, standing on the first cell the start reports. Not an
entity that merely exists, and never entity zero by default.

**FR-3 — no party, no hero.** A mission started with an empty party binds nobody. Every hero-band
reference then resolves to nothing and is reported unresolved, exactly as before this story. Entity
zero is a real entity and is never what "no hero" means.

**FR-4 — one ordinal only.** A reference naming an ordinal other than 1 resolves to nothing and is
reported unresolved, whether or not a hero was bound.

**FR-5 — the hero's identity is one rule, not two.** The entity id the compile binds and the entity
id the start assigns the party's first member are the same value **by a shared rule**, not by two
derivations that agree today. A map whose placements change moves both together or neither.

**FR-6 — the world is built once.** Resolving the hero costs no second world. Nothing is built,
stepped or discarded in order to learn which entity the hero is.

**FR-7 — the compile's other answers are unchanged.** The announcements a mission can raise, the
drop cells, the discarded build-time actions, the dropped triggers and the error carried for a
script that will not decode are exactly what they were. In particular a script that will not decode
is still **not fatal** to starting the mission.

**FR-8 — nothing that has no party moves.** A world built with no party — a map opened only to be
looked at, and every world assembled without one — is byte for byte the world built before this
story, and keeps its digest.

**FR-9 — the serialized form does not move.** The compiled script is part of the world's canonical
byte form, and the values two of its existing fields carry change on the mission path. No field is
added, no width changes, and the form's version does not change.

**FR-10 — one thing that does NOT change, stated so it is not read as an oversight.** The
map-inspection tool reports a hero-band reference as unresolved and names why. It holds no party and
assembles no world, so it has no hero to bind and continues to say so rather than inventing one.

## Acceptance criteria

**AC-1** A mission started with a party over a map whose script names hero ordinal 1 reports **zero**
unresolved references for those nodes, where it reported one per reference before.

**AC-2** In that mission's world, the check compiled from such a node names the entity that stands on
the start's first cell, and reports its reference as present.

**AC-3** A map whose winning trigger requires the hero to be within a distance of a point does **not**
reach the won outcome while the hero stands outside it, and **does** reach it once the hero stands
inside. Both halves are asserted; the first is what the second would otherwise not mean.

**AC-4** With no party, every hero-band reference in the same map is reported unresolved and no
compiled check carries a present hero reference.

**AC-5** A reference naming an ordinal other than 1 is reported unresolved in a mission started with
a party.

**AC-6** For a map with any number of placements and a party of any size, the entity id the compile
binds equals the id the started world gives the party's first member.

**AC-7** A mission whose map carries a script that will not decode still starts, still carries the
decode error, and still yields a world.

**AC-8** A world built with no party has the digest it had before this story.

## Properties

**P-1** *Invariant.* The bound entity is always the party's first member or nothing — there is no
input for which it is some third entity.

**P-2** *Negative invariant.* No world assembled without a party changes digest, and no serialized
byte form's version or layout changes.

**P-3** *Idempotence.* Compiling one map's script twice with the same party yields the same program;
the binding introduces no dependence on iteration order or on when the compile ran.

## Out of scope

Hero ordinals above 1, and what they name. The static name table. Multi-player worlds and any second
player's hero. Which script arms are implemented. The map-inspection tool's own report (FR-10).
Placing the party, which the start already owns.
