# 0124 — equip from the pack: spec

**Self-contained.** No claim id, no experiment and no research provenance appears below; that is
`provenance.md`'s. What is stated here is the contract.

## Terms

- **Item code** — the sixteen-bit word an item carries. Four fields: A bits 15..12, B bits 11..8,
  C bits 7..5, D bits 4..0. **B is the equipment slot the item occupies and also its class**, D the
  position of its defining row inside that class's collection.
- **Equipment slot** — one of twelve, numbered 1 to 12 by the original's own numbering. Slot 1 is
  the weapon.
- **Container** — an actor's carried item codes, an ordered list with no capacity. The inventory
  window's **pack area** shows the first eight of it, one cell per code, in the container's order.
- **Loadout** — what a character is wearing as the derived-stat graph reads it: the weapon in his
  hand, and the accumulated modifier every other equipped item folds into.
- **Combat block** — the nine numbers a blow and a strike distance resolve against.

## Scope

**A weapon, out of the pack, into the weapon slot, with the numbers moving.** Specifically: the
figure re-layers, the slot icon appears, the pack cell changes, and the entity that is already
fighting fights with the new numbers.

**Out of scope, and stated so a reader is not left looking for it:** armour, shields and the other
eleven slots — the state and the command below are written over *the slot an item's own code names*,
so all eleven are representable, but **nothing folds them into a stat**, and equipping one would
therefore change the picture and no number. Taking a worn item off. Drag-and-drop. Map-authored
starting equipment (see P-4).

## Functional requirements

**FR-1 — the slot an item names.** A function of an item code alone answers which equipment slot it
occupies, and whether it occupies one at all: the code's own field B, and it is a slot exactly when
B is 1 to 12. B of 0 and B of 14 — the class carried and never worn — both answer *no slot*. No
constant naming a particular slot for a particular kind of item is written anywhere.

**FR-2 — a weapon from its code.** A function turns an item code back into the weapon it names, by
reading the three collection indices the code carries — A the material, C the shape, D the row —
recovering each collection's own entry name, and resolving that name through the one weapon
resolver this tree has. A code whose B is not the weapon class is refused. The result is
**identical, field for field, to what the resolver returns for the name the code was composed
from**, that name apart.

**FR-3 — equipment is state the world holds.** Every entity a world holds carries twelve equipment
slots, each an item code, in slot order. Zero is **empty**, and it is the only spelling of empty.
It is canonical, hashed state: two worlds differing only in one entity's one slot hash differently.

*Folded from hotfix `dbd5c89` — see `docs/hotfix/ARCHIVE.md#dbd5c89`.* The equipment a world
starts with states a carried member's own loadout as well as a class row's, so a member who
survived a mission enters the next one wearing what he ended in.

**FR-4 — the equip is a command.** A command kind moves an item from an actor's container into one
of his equipment slots, naming the **index in the container** and the **destination slot**. It is
applied by the world's own advance, in the command phase, in slice order, exactly as every other
kind is. It:

- takes the code at that container index out of the container;
- writes it into that slot;
- puts whatever that slot held **back into the container at the index the moved item came from**;
- and where the slot was empty, leaves the container one shorter.

Nothing else about the entity moves — not its health, its position, its order block, its combat
block or the tick. A command naming an entity the world does not hold, an index the container does
not have, or a slot outside 1 to 12 changes nothing at all and is not an error.

**FR-5 — equipment can be read back.** A world answers one entity's twelve slots as a copy;
mutating the copy reaches nothing.

**FR-6 — the byte form carries it.** The canonical byte form gains an equipment section and its
version becomes **34**. A form at any other version is refused, whole, as before. The section is one
fixed-width record per entity, in entity order, twelve little-endian sixteen-bit codes each; it
declares no count of its own.

**FR-7 — a double-click on a pack cell asks for an equip.** While the inventory window is open, two
primary presses on the **same** pack cell, within an authored number of map-screen frames of each
other, raise a one-shot request naming **that cell's index**. The tier above drains it; a drained
request is gone. Presses on two different cells are two first clicks, never a double-click. A
double-click anywhere in the window that is not a pack cell raises nothing.

**FR-8 — the threshold is authored and named once.** The number of frames is one named constant,
in one place, disclosed as this project's own choice. No clock is read.

**FR-9 — the wiring decides whether it is applicable.** The request is drained once per frame. It
turns into a command only when the container actually holds a code at that index, that code names
an equipment slot, and — because only a weapon is folded — that code resolves to a weapon. Any
other case does **nothing**: no command, no error shown, no state moved.

**FR-10 — it reaches the world only through an advance.** The command joins the same queue a move
order joins and is applied by the same single advance. Nothing on the draw path, the input path or
the frame path writes an entity's equipment or its combat block.

**FR-11 — the numbers move, at one deterministic point.** Immediately after the advance that applied
an equip, and before anything is drawn, the subject's loadout is recomputed through the derived-stat
graph and the resulting combat block is written onto the entity. That statement is the **only** call
site in this tree that writes a combat block onto a live entity.

*Folded from hotfixes `b55f111` and `ea870ef` — see `docs/hotfix/ARCHIVE.md#b55f111` and
`#ea870ef`.* The recompute's consumers are the whole derived set the equip moves — the combat
numbers, the credited experience slot, and the panel's weapon NAME, which is written at the equip
and is neither stable nor overlaid.

**FR-12 — the window shows it.** After an equip, the character's figure is recomposed with the layer
of each occupied slot, the twelve slot icons are recomposed, and the pack is recomposed from the
container. The window's composition therefore runs **more than once per mission**, which it did not
before this story: it ran once, at mission open. It still runs only when something it draws has
actually changed.

## Acceptance criteria

- **AC-1** A code with B of 1 answers slot 1; B of 0 and B of 14 answer no slot; every B from 1 to
  12 answers itself.
- **AC-2** For every weapon resolvable from a name, re-resolving from its own code yields the same
  weapon, every numeric field equal.
- **AC-3** A world whose entity holds `[a, b, c]` and an empty weapon slot, given the equip command
  for index 1 into slot 1, holds `[a, c]` and slot 1 = `b`.
- **AC-4** The same world with slot 1 already holding `d` holds `[a, d, c]` and slot 1 = `b`.
- **AC-5** An equip command naming an absent entity, an out-of-range index or an out-of-range slot
  leaves the world byte-identical.
- **AC-6** A world round-trips through the byte form byte-identically with equipment set; a
  version-32 buffer is refused; two worlds differing only in one slot hash differently.
- **AC-7** Two presses on one pack cell inside the threshold raise exactly one request; the same two
  one frame beyond it raise none; two presses on different cells raise none.
- **AC-8** A double-click on a pack cell holding a resolvable weapon changes the entity's damage
  base — a **number**, not a flag — from what it was before to the value the recompute produces for
  the new loadout. Reverting the fold reddens it on that number.
- **AC-9** A double-click on an empty pack cell, and on a cell holding a code that names no
  equipment slot, leaves the world byte-identical.
- **AC-10** After an equip the window's subject is rebuilt: the pack cell that held the weapon now
  holds what the slot displaced (or nothing), and the weapon slot cell holds an icon.

## The pick-up log

Added to this story by the owner on 2026-08-09, in the same surface: it fires on the pick-up act
the inventory window is already about. *"при подъеме предметов надо сверху под экраном писать
список того что поднято и в каком количестве. должно исчезать через N секунд."*

**FR-13 — a pick-up says what it took.** When a pick-up moves items into a character's container,
a list appears at the **top of the screen** naming what was taken and **in what quantity**. One row
per distinct item, the quantity being how many of that item the act moved. A pick-up that moved
nothing shows nothing.

**FR-14 — it fades on its own and blocks nothing.** Each row disappears after an authored number of
frames. Nothing is dismissed, nothing is clicked, and the world, the camera and every binding keep
running underneath it exactly as if it were not there.

**FR-15 — it is not the notice.** The notice box (`pkg/ui/notice.go`) is **modal**: it gates the
whole map arm while it is open, it carries a dismiss button and an advance seam that can navigate
away from the mission, and it holds one string. A pick-up list is several rows, transient, and must
take no input at all — so reusing it would mean giving the notice a non-modal mode, a per-row
lifetime and a multi-row layout, which is three changes to a modal box against one small surface of
its own. This is a separate, smaller surface, and it names no simulation type either.

**FR-16 — an item is named from the data, and an unnameable one still shows.** A row's text is the
item's own name, recovered from the definition collections already parsed — the same recovery the
equip half uses to turn a code back into a weapon. A code that cannot be named prints **the code**
rather than being dropped, so a gap is visible instead of silent.

**FR-17 — the dwell is authored and counted in frames.** One named constant, one place, disclosed
as this project's own: nothing decoded says how long such a list lives. Frames, not wall-clock.

- **AC-11** A pick-up of three codes, two of them equal, shows two rows: one at quantity 2 and one
  at quantity 1. After the authored number of frames, no row remains.
- **AC-12** With rows on screen, a press anywhere the map would otherwise take still selects,
  orders and pans exactly as it does with none.
- **DD-6** Quantity is counted over the codes the act moved, not over the container afterwards: two
  pick-ups of the same item are two separate lists, each stating its own act, rather than one row
  restating a running total the pack area already shows.

## Properties

- **P-1** The simulation tier stays stdlib-only and reads no clock, no float and no randomness for
  any of this.
- **P-2** Nothing here can name a slot for a kind of item: every slot number that reaches the world
  came off an item's own code.
- **P-3** A run with the inventory window never opened is byte-identical, tick for tick and digest
  for digest, to the same run before this story — the section is present and zeroed.
- **P-4** Equipment a mission **starts** with does not reach the simulation in this story. The
  character's starting weapon is the loader's, and it is what the recompute uses until the first
  equip; after that the world's own slot is. So the first weapon equipped displaces nothing, and the
  starting weapon is superseded rather than returned to the pack. Every later equip displaces
  properly.

  *Folded from hotfix `c9d0a2c` — see `docs/hotfix/ARCHIVE.md#c9d0a2c`.* Closed: a unit's
  class-row weapon reaches the simulation as part of the loadout the loader composes, so the
  first equip displaces a starting weapon into the pack instead of superseding nothing.
- **P-5** The drawing tier still holds no archive, no definition table and no item code. It reports
  *which cell was double-clicked* and nothing about what is in it.

## Design decisions

- **DD-1** Equipment is a fixed twelve-wide record per entity rather than a list, because a slot's
  number is the whole of what it means and an empty slot must be representable at its own index.
- **DD-2** The equipment section takes the purse section's shape — fixed width, no count of its own
  — and sits between the carry section and the purse section. Every offset above it is unmoved.
- **DD-3** The request crosses from the drawing tier as **state drained by the tier above**, not as
  a new callback in the map-opener's result list. The reason is blast radius: that list is threaded
  through the flow, the app, the front-end and every test that builds one, and this story has no
  need of a ninth seam to say one integer.
- **DD-4** The recompute stands immediately after the advance rather than inside the simulation,
  because the simulation cannot see a definition table. It is deterministic by position: same tick,
  same statement, ascending entity id.
- **DD-5** The re-resolution recovers a **name** and re-enters the existing resolver rather than
  recomputing the scaled numbers itself, so there is one arithmetic for a weapon and not two.
