# 0123-corpse-loot — specification

## Intensity and terrain

Rigor **medium**, scope **narrow**. Terrain: `pkg/sim`, the simulation tier — stdlib-only,
deterministic, hashed. One behaviour is added and one defect in the state it stands on is
repaired.

## The behaviour

A unit that dies leaves what it was carrying on the ground, in a sack, and that sack is
picked up by the transfer primitive this tree already has. Nothing else about a death
changes.

## Functional requirements

**FR-1 — The container becomes the sack.** On the tick an entity crosses into death, every
item code its container holds is placed in a ground sack at the cell the entity occupies,
**in the container's own order**, and the entity is left carrying nothing. The codes are
moved, not copied: after the drop no code is both carried and on the ground.

*Folded from hotfixes `c9d0a2c` and `026e936` — see `docs/hotfix/ARCHIVE.md#c9d0a2c` and
`#026e936`.* The drop set is the container AND the base actor's two weapon slots, unequipped
second then first and appended at the tail, all-or-nothing; the humanoid armour fields stay on
the corpse. A unit whose loadout the loader withheld leaves nothing of it. And a weapon
suitable for no consumer class is not an item: the loader composes no code for it, so an
innate attack never enters the equipment set. **Superseded by `0132-corpse-drops-worn`:** the
humanoid armour fields no longer stay on the corpse — the drop set is the container and all
twelve equipment slots, in that story's own order.

**FR-2 — A sack is built only if there is something to put in it.** An entity dying with an
empty container leaves no sack, and the world's sack list is unchanged.

*Folded from hotfix `c9d0a2c` — see `docs/hotfix/ARCHIVE.md#c9d0a2c`.* *Is this a holding* and
*would this body drop* are two questions asked by name; a body wearing only what it cannot
drop plants no sack, empty or otherwise. **Superseded by `0132-corpse-drops-worn`:** every slot
is droppable now, so the two questions have one answer, and a body wearing only armour plants
a sack.

**FR-3 — One sack per cell.** A drop onto a cell that already holds a sack pours into that
sack — its codes appended at that sack's tail, its gold left alone — rather than building a
second. The world's sack list stays ascending by `(Y, X)` and free of duplicate cells.

**FR-4 — Once per death.** The drop fires on the transition into death and on no later
tick. A second kill or a second blow on a body that has already died adds nothing to the
world.

**FR-5 — A corpse off the map drops nothing.** An entity whose cell is outside the world's
bounds leaves no sack, and its container is left as it stands. This is a refusal, not a
clamp and not a fold onto a nearby cell.

**FR-6 — A container belongs to its entity for as long as that entity exists.** Removing an
entity from a world leaves every surviving entity carrying exactly what it carried before,
and leaves the world's per-entity container list the same length as its entity list. No
read of a container, and no encode of a world, may depend on whether some other entity has
been removed.

**FR-7 — The random source is documented against what is known.** `pkg/sim`'s generator
doc block must not assert that nothing describes the original engine's generator. It states
what the original's is, that ours is deliberately a different one, and that this is a
disclosed divergence rather than a reconstruction.

## Acceptance criteria

**AC-1** Killing an entity whose container holds two codes leaves exactly one new sack, at
that entity's cell, holding those two codes in the order the container held them; the
entity carries nothing afterwards.

**AC-2** Killing an entity with an empty container leaves the sack list byte-identical to
what it was.

**AC-3** Two entities standing on one cell, each carrying codes, dying in the same advance
leave **one** sack on that cell holding both sets, in death order.

**AC-4** A drop onto a cell that already holds a sack leaves one sack there, holding the
standing sack's codes followed by the corpse's, with the standing sack's gold unchanged;
the list is still ascending by `(Y, X)`.

**AC-5** A second kill command on an entity that has already died leaves the sack list and
every container unchanged.

**AC-6** After a world removes an entity, every surviving entity's container reads back as
its own, the container list and the entity list are the same length, and the world's
whole-world stock read answers without panicking.

**AC-7** A death draws nothing from the world's random source: the generator state after an
advance in which an entity dies and drops is the state that advance would have reached with
no death in it.

**AC-8** An entity standing on the cell where another died picks up the dropped codes with
the existing transfer primitive, and afterwards carries them.

**AC-9** A world in which a death has occurred encodes and decodes back to an equal world,
and the encoded form still declares the same version this tree shipped before this story.

## Properties

**P-1 — No new saved state.** This story adds no field to any record the byte form carries
and does not move the form version. Everything it writes is already serialised.

**P-2 — Determinism is unchanged.** No float, no clock, no process-global randomness enters
`pkg/sim`. The drop is pure integer state movement with no draw.

**P-3 — The drop refuses nothing on size, ownership or distance.** A container of any
length drops whole; who owned the dead unit and who is nearby are not questions the drop
asks.

## Out of scope

- **Gold.** The dead unit's purse and the roll that fills it are not built. That needs
  three per-template treasure columns and a type gate carried per entity, which is new
  saved state and a byte-form version this story is not taking.
- **Equipment.** Nothing here unequips anything, because this tier has no equipment slots.
  The drop moves the container; whatever puts a unit's starting weapon and boots into that
  container is another story's, and this one needs no change when it lands.
- **Suppressed loot.** A unit whose template should leave nothing is not distinguished
  here: no entity in this tier carries a name or a mark standing in for one.
- **A pick-up order.** The walk-to-the-sack order remains out of scope, as it was when the
  transfer primitive landed; a player still picks up from the cell the unit stands on.
- **Which frame a sack draws.** Unchanged, and still the sheet's first frame for every
  sack.
- **`pkg/mapload`.** No file there is touched.

## I/O examples

- A world with one entity at `(4, 6)` carrying `[0x101, 0x102]` and no sacks. One kill
  command. Result: one sack at `(4, 6)` holding `[0x101, 0x102]`; the entity carries `[]`.
- The same world with a sack already at `(4, 6)` holding `[0x090]` and 25 gold. Result: one
  sack at `(4, 6)` holding `[0x090, 0x101, 0x102]` and 25 gold.
- A world of two entities, ids 1 and 2, where 2 carries `[0x111]`. Remove 1. Result: entity
  2 still carries `[0x111]`, and the world's stock read answers `[{2, [0x111]}]`.

## Success is visible as

Select an enemy unit on the map screen and press the debug kill key; a sack appears on the
cell it fell on. Move a party unit onto that cell and press the pick-up key; the sack is
gone and the codes are in that unit's inventory panel.
