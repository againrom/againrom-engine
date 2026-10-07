# 0132-corpse-drops-worn — specification

## Intensity and terrain

Rigor **medium**, scope **narrow**. `spec-anchored / static` — the contract outlives the ticket and
no tool propagates it; humans reconcile.

Terrain: **brownfield** in `pkg/sim`, the deterministic hashed tier — a shipped death rule is
widened, so what it already does is pinned before it changes. **Greenfield** in `cmd/missionrun`,
which gains a report it never had.

## The behaviour

A person who dies leaves on the ground everything he was wearing, not only the weapon in his hand.
A clubman killed in the tenth mission leaves his club **and** his boots.

Current behaviour, verified against the baseline: the container and the two base-actor slots reach
the sack, and the ten armour slots stay on the corpse. That reading came from a routine that had
been read only as far as the middle; the four instructions after the weapon arm dispatch a call
that empties every worn field into the same container the sack is then built out of.

## Terms

- **Equipment slot** — one of an entity's twelve, numbered 1 to 12. Slot 1 and slot 2 are the two
  a hand holds; 3 to 12 are the armour set. The zero code means empty.
- **Container** — the item codes an entity carries rather than wears, in the order they were added
  to it.
- **Holding** — a container with a code in it, or a slot at other than the zero code, or both.
- **Ground sack** — one cell, a purse, and the codes lying on that cell.
- **Crossing into death** — the tick on which an entity leaves the living and takes its first decay
  stage. It happens once per entity; a body already on that ladder never crosses again.

## Functional requirements

**FR-1 — A body drops everything it was holding.** On the tick an entity crosses into death, every
non-empty equipment slot and every code in its container is placed in the ground sack at the cell
the entity occupies. The codes are **moved, not copied**: afterwards the entity carries nothing and
every one of its twelve slots is empty, and no code is both held and on the ground.

**FR-2 — The order is fixed and observable.** The sack receives, in exactly this order: the
container's codes in the container's own order; then slot 2; then slot 1; then slots 3 to 12 in
ascending slot order. An empty slot contributes nothing and shifts nothing — it is passed over, not
dropped as a zero code.

**FR-3 — A body that held anything leaves a sack.** An entity dying with an empty container and
anything at all worn leaves a sack. An entity holding nothing — empty container and every slot
empty — leaves the world's sack list exactly as it was, with no empty sack planted.

**FR-4 — Once per death.** The drop fires on the transition into death and on no later tick. A
second kill, or a second blow on a body already dead, adds nothing to the world.

**FR-5 — A corpse off the map drops nothing, all or nothing.** An entity whose cell is outside the
world's bounds leaves no sack, and its container **and all twelve slots** are left exactly as they
stand. This is a refusal, not a clamp and not a fold onto a nearby cell, and it is decided before
the container or the first slot is touched. It is **spent, not deferred**: the crossing has
happened, so no later tick retries the drop.

**FR-6 — One sack per cell, in the world's own order.** A drop onto a cell that already holds a
sack appends at that sack's tail and leaves its gold alone, rather than building a second. The sack
list stays ascending by `(Y, X)` and free of duplicate cells. When two bodies cross into death on
the same tick on the same cell, their drops appear in the sack **in the order the tick resolved the
two crossings**, each whole and in FR-2's order, one after the other: a crossing a command causes
lands where that command sits in the tick's list, a crossing one of the tick's own passes resolves
lands where that pass reaches it. Nothing re-orders the sack afterwards and nothing here reads an
unordered source, so the same commands over the same world always produce the same sack and the
same digest.

**FR-7 — The canonical byte form does not change.** No field is added, removed or resized, and the
form's version is the one it already carries. A world holding a drop encodes, decodes and
re-encodes to the same bytes.

**FR-8 — The headless drive can show the drop.** For **every** ordered attack, whether or not it
fells, `missionrun` reports what the victim was wearing and carrying **before** the blow and what
sack stands on his cell **after**, so a body's loadout and the ground under it can be compared
against a lawful install with no screen in the way. A report for an attack that did not fell is
evidence too: it shows the ground unchanged. Each item code is reported by its own seven-digit name
**and** the slot its class names, **and additionally** the weapon's name where the loaded tables
resolve one — the seven-digit name is never replaced, so a code the tables cannot name is still
readable and comparable.

## Acceptance criteria

| # | GIVEN | WHEN | THEN |
|---|---|---|---|
| AC-1 | an entity wearing codes in slots 1, 2 and several of 3..12, carrying two codes | it is killed | the sack at its cell holds the two carried codes, then slot 2, then slot 1, then the armour codes in ascending slot order |
| AC-2 | the same entity after that kill | its slots and container are read | every slot is empty and the container is empty |
| AC-3 | an entity wearing armour only, carrying nothing | it is killed | one sack stands at its cell holding those codes in ascending slot order |
| AC-4 | an entity holding nothing at all | it is killed | the sack list is unchanged |
| AC-5 | an entity with slots 1, 5 and 12 filled and 2, 3, 4, 6..11 empty | it is killed | the sack holds exactly three codes — slot 1, then slot 5, then slot 12 — with no zero among them |
| AC-6 | a dead entity that has already dropped | it is killed again, or struck again | the sack list and its own holdings are unchanged |
| AC-7 | an entity standing outside the world's bounds, wearing a full set | it is killed | no sack is planted and all twelve slots still hold what they held |
| AC-8 | a cell already holding a sack with gold and codes | a body dies on it | that sack's codes gain the body's at the tail, its gold is unchanged, and there is still one sack on the cell |
| AC-9 | a world in which such a death has happened | it is encoded, decoded and encoded again | the second encoding equals the first and the form's version byte is unchanged |
| AC-10 | the tenth mission on a lawful install, both language roots | a clubman is felled by an ordered attack | the drive reports the club and the boots on the ground where he fell, and reports him wearing them beforehand |
| AC-11 | two entities on one cell, each wearing and carrying something | both are killed in one advance, by two commands | the single sack holds the first-killed body's whole drop, in FR-2's order, ahead of the second's — and killing them in the opposite order reverses the two blocks and nothing else |

## Properties

**P-1 (completeness).** For every entity and every reachable holding, the multiset of codes in the
sack after the death equals the multiset the entity held before it. Nothing is duplicated and
nothing is lost.

**P-2 (negative invariant).** No code appears both in a sack and in a living or dead entity's
holdings as a result of a drop.

**P-3 (invariant).** A death changes no derived number in this tier: an entity's combat block is
state here and is computed nowhere, so emptying its slots leaves it exactly as it stands.

**P-4 (idempotence).** Applying the death transition to an entity already in it is a no-op on the
sack list and on that entity's holdings.

**P-5 (negative invariant).** The drop draws nothing from the simulation's random source: two runs
differing only in a death consume the same number of values.

## Out of scope

- What the player may then **do** with a dropped armour piece. Equipping armour is another story's
  contract; this one only puts it on the ground.
- **Gold at death.** The original rolls a purse for some classes; no purse is added here.
- Any **recompute** of defence, absorption or damage from a changed loadout.
- The pick-up side. The transfer primitive that lifts a sack is unchanged and is not re-specified.

## Disclosed divergences

1. **Slot 1 is dropped unconditionally.** The original drops it only when the weapon's own data row
   says it may be. That column is not readable from this tier, so the narrow, visible behaviour is
   kept. Carried in unchanged; named again because this story widens the set around it.
2. **The template-name suppression stays as wide as it was, and is now too narrow.** The original
   destroys the whole inventory of a body whose template name marks it as an NPC — armour included
   — so such a body leaves nothing. No entity in this tier carries a name and giving one a
   suppression bit would change the byte form, so this build applies the suppression **at load
   time**, by leaving the two hand slots empty on an NPC-templated person. That gate was drawn at
   two slots because two slots were all a corpse gave up. This story does **not** widen it, so such
   a body now drops the armour it is still wearing where the original destroys it. The narrowness
   is deliberate rather than overlooked: widening the gate would also strip a **living** NPC of the
   armour he is seen wearing, which is a different question from what his corpse leaves and is not
   settled here.
3. **Two of the original's fourteen equipment fields have no representation here.** The original's
   own take-off path can address twelve; this contract's twelve are those. Two fields it never
   addresses are unrepresented, so nothing can be worn in them and nothing drops out of them.
