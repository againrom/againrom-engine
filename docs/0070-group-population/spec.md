# Spec — the group a check counts

**Intensity: spec-anchored / static. Terrain: brownfield** for the script runtime, the binder and the
byte form — each already ships behaviour this changes — and greenfield for the arm itself.

A mission script asks a question this build cannot answer: how many units a named group still has.
Twenty-four of the campaign's twenty-eight maps ask it, ninety-six times, and every trigger that asks
is inert — never evaluated, never fired. On the campaign's first mission it is the first of three
triggers in the win chain, so the mission cannot begin the sequence that ends in a win.

The map already carries the answer's ingredients and nothing carries them far enough: the placed-unit
record's group word is decoded and then dropped, both by the binder — which packs only plain integer
parameters into a compiled check, and a group reference is not one — and by the entity, which has no
membership field at all.

## Context

The compiled check writes its result into a register, and a trigger compares registers. An arm this
build does not evaluate writes no register, and every trigger reading a register such an arm owns is
marked inert and skipped whole. That rule is what keeps an unimplemented arm from reading as an
evaluated-false one, and it is what makes ninety-six authored questions visibly unanswered instead of
silently answered zero.

This story implements one arm. The other unimplemented check arm on that mission — the one that asks
about a sack at a cell — is left unimplemented on purpose (see *Out of scope*).

## Functional requirements

- **FR-1 — an entity carries the group its map placed it in.** Every entity built from a placed-unit
  record carries that record's group identifier — a **32-bit unsigned** value, the width the record
  carries it at. Every other entity, however it was built, carries the identifier's zero, which is a
  real group and not an absence.

- **FR-2 — a compiled check carries the group it names.** A check node's group parameter reaches the
  compiled check as a resolved reference with its own presence flag, alongside the two unit
  references already carried. It does **not** join the plain integer parameters: the packing of those
  is unchanged, so no other arm's parameters move.

- **FR-3 — the arm counts the group's living members.** The arm is **check opcode 1** of the
  twenty-two-arm check vocabulary. It writes into its register the number
  of entities in the world that carry the named group identifier and are not dead — the same sense of
  dead every other arm of this runtime uses. A group with no living member counts zero, and a group
  identifier no entity carries counts zero: both are measurements, not failures.

- **FR-4 — a check naming no group measures nothing.** A check of this arm whose node carried no
  group parameter has no reference and **writes no register**: the register keeps whatever value it
  already held. This is distinct from FR-3's zero, which is an answer. It does **not** make the arm
  unimplemented and does not make its readers inert — inertness is a property of the opcode, and this
  opcode is one this build now evaluates.

- **FR-5 — the count is canonical simulation state.** The entity's group identifier and the compiled
  check's group reference are both carried by the world's byte form and both enter its digest. A
  world resumed from its bytes counts the same members. The byte form takes its next version and
  refuses every earlier one.

- **FR-6 — the loudness rule is unchanged for what is still unimplemented.** The arm leaves the
  unsupported-arm report and the inert-trigger rule exactly as they are for every arm this build
  still does not evaluate. A trigger reading only registers this build now writes stops being inert;
  a trigger reading any register an unimplemented arm owns stays inert, is still never evaluated, and
  still leaves its latch untouched.

## Acceptance criteria

- **AC-1** — a world built from a map whose placed units carry several distinct group identifiers
  gives each entity its own record's identifier, including the zero one; a world built from no map
  gives every entity the zero identifier.

- **AC-2** — a check node carrying a group parameter compiles to a check whose group reference is
  present and whose value is the node's; a node carrying none compiles to a check whose reference is
  absent. A node carrying a group parameter **and** plain parameters compiles to the same plain
  parameters, in the same slots, as it does today.

- **AC-3** — the arm's register over a world holding: three living members of the named group and two
  of another (3); the same after two of the three are killed (1); the same after all three (0); a
  group identifier no entity carries (0); a downed member, which counts as dead; and a check naming
  the identifier **zero** over a world holding two members of group zero (2), which is an ordinary
  count and not an absence. And a second check of the arm naming a different group in the same pass
  writes its own register and not the first's.

- **AC-4** — a check of this arm with no group reference leaves its register at a preset value the
  pass could not have produced, over a world whose entities would have made the count some other
  number. The trigger reading that register is **not** inert, is evaluated, and its latch is written
  as any live trigger's is.

- **AC-5** — a world holding entities in groups and a script holding a group check marshals, reads
  back with every group identifier and the check's reference crossing record for record, and both
  worlds step on to identical digests. Two worlds differing only in one entity's group identifier
  have different digests. A byte form at the previous version is refused.

- **AC-6** — a hand-built script whose only unimplemented arm is **check opcode 14**, the sack arm,
  reports that arm and marks its readers inert, while a trigger reading only group checks is live and
  fires. The unsupported report no longer names opcode 1.

- **AC-7** — the **shape** of the first mission's win chain, driven end to end on a synthetic world
  and a hand-built script rather than on a map: the named group's last member dying arms the first
  trigger, a unit reaching a cell sets a mission variable, and a distance with that variable set wins.
  The chain is built so that no arm outside this build's supported set is needed to advance it — the
  ownership hand-over the shipped map uses is stood in for by moving the unit directly.

## Properties

- **P-1 — the determinism wall holds.** The count reads entities and nothing else — no clock, no
  float, no input, no map. Iteration is over the world's own ordered entities, so the answer does not
  depend on storage order.

- **P-2 — total.** Every opcode still answers. The new arm cannot panic on a group no entity carries,
  on an empty world, or on a check with no reference.

- **P-3 — the byte form stays injective and refuses what no tick can leave.** The two widened records
  consume the buffer exactly, a truncated one is refused, and a check carrying a group presence byte
  outside its value set is refused rather than read as truthy.

## Out of scope, and disclosed

**The sack arm — check opcode 14 — stays unimplemented, and that is a decision rather than an
omission.** This tree's
world holds no sack, no item and no container, so the only thing that arm could answer is a constant,
every time, without measuring anything — which is exactly the failure the inert-trigger rule exists
to prevent, and it would arm twenty-seven further triggers across the campaign on a measurement never
taken. Over both installed roots it moves no map into the set whose win instant is reachable. It
stays in the unsupported report and its readers stay inert.

**Not owned here:** the instant that hands a group to a player, which is the next gate on the same
mission's win chain and is not a check; the binding of the hero band; and every other unimplemented
arm.

**Divergences, each deliberate:**

- **Membership is the group identifier alone.** The original keys a runtime group on the owning
  player as well, and a script names an identifier without one. Nothing in this tree carries a placed
  unit's owner at all, so the distinction is not expressible here; where one map gives one identifier
  to two owners this counts both.

- **The count excludes the dead.** The arm's own body was not read. The whole-membership reading is
  the rival, and under it the shipped trigger that compares this count against zero could never fire
  and the campaign's first mission could never be won. The reading here is the one the shipped map's
  own comparison forces, and a downed unit is dead to it, as it is to every other arm.

- **Membership is held on the member.** The original hangs a group off a player and lists its
  members. Nothing in this tree adds a member to a group, removes one, or moves one between groups,
  so the two are the same observable behaviour and one field is the shape that adds no state nothing
  can exercise. It is the same trade the rate term already makes.

- **The identifier's zero is a real group.** It is carried and compared like any other, and the
  presence of a reference is a flag of its own rather than a reserved identifier value.
