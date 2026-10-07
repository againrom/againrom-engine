# Spec — the owner a script hands over

**Intensity: spec-anchored / static. Terrain: brownfield** for the map decoder, the binder, the script
runtime and the world's byte form — each already ships behaviour this changes — and greenfield for the
two arms themselves.

A map places every unit under one of its roster's owners, and a mission script can move a unit or a
whole group from one owner to another. This build carries neither: the placed-unit record's owner word
is skipped by the decoder, an entity has no owner field, and the two script arms that change one are
reported as unimplemented and do nothing when a trigger fires them. So a map that authors an escort —
a unit that becomes the player's part-way through the mission and stops being the player's again —
loads as a map on which ownership never existed.

This story carries the owner from the placed record to the simulation, and runs the two arms that
change it. It deliberately adds **no reader**: nothing in this build branches on an owner afterwards,
and the contract says which two things a later story would make of it.

## Context

The compiled program has three arrays — checks, instants and triggers. A check writes a register, a
trigger compares registers and runs the instants of the triggers whose comparisons hold. An arm this
build cannot evaluate is reported before any tick runs; a **check** it cannot evaluate additionally
poisons its register, so that every trigger reading that register is marked inert and skipped whole
rather than being told a measurement was taken. An **instant** it cannot run is skipped where it
stands, and the trigger around it still fires and still runs the arms that are implemented.

That asymmetry decides the shape of this story's claims: implementing an instant arm cannot make any
trigger anywhere live, evaluated, or firing that was not already. This spec claims no such effect, and
its criteria are written so that a build which produced one would fail them.

Two shipped arms are implemented here. Both take a player as their second parameter and differ only in
what they name first: **opcode 22** names a group and **opcode 19** names a unit.

## Functional requirements

- **FR-1 — a placed-unit record's owner is decoded.** The decoded unit record carries the owner word
  the placed-unit record holds, as a **32-bit unsigned** value — the width the record carries it at.
  The value is a **1-based index into the map's roster**, so 1 names the first roster entry. No other
  decoded field changes, and a map opened and written back is byte-identical, as it is today.

- **FR-2 — an entity carries the owner its map placed it with.** Every entity built from a placed-unit
  record carries that record's owner. Every entity built any other way — from no map, or by a caller
  naming none — carries **zero, which is no owner at all** and not a roster entry. No path silently
  gives an entity the first roster entry.

- **FR-3 — a compiled instant carries the references it names.** An instant node's unit, group and
  player parameters reach the compiled instant as three references, each with its own presence flag,
  alongside the plain integer parameters it already carries. They do **not** join those plain
  parameters: their packing is unchanged, so no other arm's parameters move. A unit reference resolves
  through the same identifier bands a check's unit reference does, and one that does not resolve is
  reported exactly as an unresolvable check reference is.

- **FR-4 — the two arms hand ownership over.**
  **Opcode 22** sets the owner of **every entity carrying the named group identifier** to the named
  player. **Opcode 19** sets the owner of the **one named entity** to the named player. Both write the
  player's value as the map wrote it. Both include entities that are dead or downed: an entity's group
  and its identity outlive its death everywhere in this runtime, and an arm that skipped the fallen
  would make a hand-over depend on when it fired. A group identifier no entity carries is not a
  failure — nothing changes.

- **FR-5 — an arm naming nothing writes nothing.** An instant of either arm whose node carried **no
  player parameter** changes no owner anywhere. So does an opcode 22 whose node carried no group
  parameter, and an opcode 19 whose unit reference is absent or did not resolve. This is distinct from
  assigning zero: an entity's owner is left exactly as it stood.

- **FR-6 — ownership is canonical simulation state.** The entity's owner and the compiled instant's
  three references are carried by the world's byte form and both enter its digest. A world resumed from
  its bytes holds the owners the world it was cut from held, and re-runs the arms to the same answers.
  The byte form takes its next version and refuses every earlier one.

- **FR-7 — the loudness rule is unchanged, and no trigger changes state.** The unsupported-arm report
  stops naming instant opcodes 19 and 22 and names every other unimplemented arm exactly as it did.
  **No trigger becomes live, inert, evaluated or unevaluated because of this story**, and no latch is
  written that was not written before.

## Acceptance criteria

- **AC-1** — a synthetic placed-unit record set carrying distinct owner words decodes to exactly those
  words, including the field's maximum value; a document opened and written back is byte-identical.

- **AC-2** — a world built from a map whose placements carry several distinct owners gives each entity
  its own record's owner; a world built from no map gives every entity zero.

- **AC-3** — an instant node carrying a unit, a group and a player parameter compiles to three present
  references holding the node's values; a node carrying none compiles to three absent ones; a node
  carrying references **beside** plain parameters compiles to the same plain parameters, in the same
  slots, as it does today. An action node whose unit reference does not resolve is reported, and its
  reference is absent.

- **AC-4** — opcode 22 over a world holding three entities of the named group — one of them dead — and
  two of another: all three take the named player and the other two are untouched. A group identifier
  no entity carries leaves every owner as it was. Two instants of the arm naming different groups in
  one trigger each write their own members.

- **AC-5** — opcode 19 over a world: the named entity takes the named player, no other entity moves,
  and an instant naming an entity the world no longer holds changes nothing.

- **AC-6** — each of FR-5's three cases, over a world whose entities carry owners a successful arm
  would have overwritten, leaves every one of those owners unchanged.

- **AC-7** — a world holding entities with distinct owners and a script holding both arms marshals,
  reads back with every owner and every reference crossing record for record, and both worlds step on
  to identical digests. Two worlds differing only in one entity's owner have different digests. A byte
  form at the previous version is refused; a truncated widened record is refused; a reference presence
  byte outside its value set is refused rather than read as truthy.

- **AC-8** — a hand-built script authoring both arms and one arm this build still does not implement
  reports **only** that third arm. Over the same script, the set of inert triggers and the set of
  triggers evaluated in a pass are identical to what they are with both arms unimplemented — the
  latches after one pass agree position for position (FR-7).

- **AC-9** — the hand-over choreography of a shipped escort mission, driven end to end on a synthetic
  world and a hand-built script rather than on a map: a trigger hands a group to one player; a second
  trigger, firing later, hands one member of that group to a different player; and the owner of every
  entity after each pass is the one the two arms wrote. No arm outside this build's supported set is
  needed to advance it.

## Properties

- **P-1 — the determinism wall holds.** Both arms read and write entities and nothing else — no clock,
  no float, no input, no map. Opcode 22's scan is over the world's own ordered entities, so which
  entities it writes does not depend on storage order.

- **P-2 — total.** Neither arm can panic: not on a group no entity carries, not on an empty world, not
  on an absent reference, and not on a unit reference naming an entity the world no longer holds.

- **P-3 — the byte form stays injective and refuses what no tick can leave.** The two widened records
  consume the buffer exactly; a truncated one is refused; and every owner value the constructor accepts
  survives a round trip, so no world this package can build is one it cannot read back.

## Out of scope, and disclosed

**Nothing in this build reads an owner after this story, and no reader is added.** Ownership enters on
the same footing the entity's class key already has: canonical state that is carried, hashed and
round-tripped, which no step reads, changes or branches on. That is a deliberate stopping point, and
the two things a later story makes of it are named rather than left implicit:

- **Order legality.** Which units the player may command is decided today without consulting anything,
  so every unit on a map is equally commandable. Gating it needs to know which roster entry the human
  participant is, and a map does not say.
- **The route budget.** The decoded far-search budget has a term that reads the mover's owning player,
  and this build takes the human-owned form of it unconditionally, on the stated premise that no entity
  carries an owner. This story falsifies that premise's antecedent and settles nothing about the term,
  because the term asks whether a **human participant** owns the mover and not which roster entry does.
  The premise is left as it stands and the arm keeps its current form; a story that decides the first
  bullet decides this one with it.

**No progress on any mission's win condition is claimed, and none is delivered.** Implementing an
instant arm cannot arm a trigger. AC-8 exists to make a build that changed a trigger's state fail.

**Not owned here:** the roster itself, its per-entry relation words, the owner field the map's structure
records carry, and every other unimplemented arm.

**Divergences, each deliberate:**

- **An owner is a number, not a roster object.** This build has no roster, no player and no group
  object; the number is the map's own word and every arm assigns it directly.

- **Opcode 22's membership is the group identifier alone.** The original keys a group on its owner as
  well, so where a map gives one identifier to units of two owners, this arm moves them all. This story
  is also the first that can **create** such a collision, by giving two owners' units one owner and
  leaving their identifiers alone.

- **Zero is no owner.** The identifier space begins at 1, which leaves zero free to mean absent without
  a flag on the entity. It is the opposite of the group identifier, whose zero is a real group, and the
  two are opposite because the two identifier spaces are.

- **The hero, and every unit not placed by the map, is owned by nobody.** No arm gives one an owner and
  no arm reads one, so this is a value at rest rather than a behaviour.
