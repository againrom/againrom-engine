# 0129 — give all: specification

On the tenth mission the witch is supposed to hand the player her healing potions. She holds them —
the map's stock record puts three of one code into her container and the world already builds it —
and the trigger that should move them is authored, fires, and does nothing, because the instant it
names is an arm this build does not run. This story is that arm: opcode 28, **Give All**.

It is about the potions **arriving**. Their icon resolves and their name resolves already; nothing
here draws anything.

## The shape of the gap

A compiled instant carries three references — a unit, a group, a player — and one of each. Opcode 28
names **two units**: it moves a container from the first to the second. The record has nowhere to put
the second, so the node compiles with the giver bound and the receiver dropped, and even a
dispatched arm would have had nothing to give to. The record's own widening is therefore not
incidental to this story, it is half of it.

## Functional requirements

**FR-1 — a compiled instant carries a SECOND unit reference.** `ScriptInstant` gains `Unit2` and
`HasUnit2` on exactly the terms `ScriptCheck`'s pair already stands on: an entity id **already
resolved** by whoever compiled the script, with its own presence flag because entity id zero is a
real entity and no id value is free to mean "none". It does **not** join `Args`: the plain parameters
are packed in encounter order, and admitting a reference into that packing would shift the parameters
of every other node that carries one, including the arms this build does not run.

**FR-2 — the binder fills it.** The tier that compiles a map's script already resolves a node's first
two unit parameters into a pair, through the same three id bands, and already carries that pair onto
a compiled **check**. It carries the same pair onto a compiled **instant**. Nothing about how a
reference is resolved changes: the first unit parameter encountered is the first reference and the
second is the second, and a reference that resolves to nothing sets no flag and is reported exactly
as it is today.

**FR-3 — opcode 28 is an arm this build runs.** It joins the single table that decides what this
build evaluates, so the compile-time report and the runtime dispatch cannot come to disagree about
whether it exists. Nothing else about the report changes: implementing it arms **no** trigger,
because inertness is derived from unimplemented *checks* and this adds none.

**FR-4 — the pour.** The arm appends the **whole** of the first reference's container to the second
reference's container, **in the first's own order**, after whatever the second was already holding,
and leaves the first's container **empty**. That is one act with no partial outcome: there is no
capacity to reach, nothing that can refuse an item, and no per-item failure to stop half way.

**FR-5 — what the arm does NOT move.** Not an equipment slot, on either side — the container and the
twelve worn places are different state, and a giver wearing armour is still wearing it afterwards.
Not gold. Not a position, an owner, a group, a state byte, a health value, an order or the tick.

**FR-6 — the four refusals, each leaving the world exactly as it was found.** A node whose first
reference is absent; a node whose second reference is absent; a reference naming an entity this world
does not hold; and the two references resolving to **one** entity. None of the four is an error a
caller sees — a script arm has no caller — and none of the four writes a partial result.

**FR-7 — the second reference is state.** It is carried by the byte form, so it enters the digest and
survives a round trip. The instant record widens by five bytes, appended at its own tail so that no
offset outside it moves, and the form takes version **38**.

## Acceptance criteria

**AC-1 — the potions arrive.** A world in which entity A holds three codes and entity B holds one,
stepped over a trigger whose instant is opcode 28 with A first and B second: B holds its own code
followed by A's three, in A's order, and A holds nothing. Asserted as the exact two lists, not as a
count and not as non-emptiness.

**AC-2 — the giver keeps her armour.** The same transfer with both entities wearing something: both
equipment records are byte-for-byte what they were, and the giver — now holding nothing — still wears
what she wore.

**AC-3 — each of the four refusals changes nothing.** For each of FR-6's four cases separately, the
whole `MarshalBinary` output before the step equals the whole output after it. Comparing the form
rather than the two containers is what makes "nothing else moved either" part of the assertion.

**AC-4 — the receiver's own order is preserved across a second pour.** Two opcode-28 nodes firing in
one trigger, A into C and then B into C, leave C holding its own codes, then A's, then B's.

**AC-5 — the binder carries both references.** A synthetic script whose action node names two unit
parameters compiles to an instant with both `Unit`/`HasUnit` and `Unit2`/`HasUnit2` set to the two
resolved entities, in parameter order; a node naming one names the first alone and leaves the second
absent; a node whose second parameter resolves to nothing leaves the second absent and reports it.

**AC-6 — opcode 28 is no longer reported as a gap.** A script carrying one is not in
`Script.Unsupported`, and a trigger naming it is not inert.

**AC-7 — the form.** A world whose script carries an instant with a second reference encodes and
decodes to an equal world; the form declares version 38; a buffer declaring 37 is refused; and two
worlds differing only in an instant's second reference hash differently.

## Properties

**P-1 — the arm reads no storage order.** It finds both entities by id, so which entities it writes
does not depend on the order the world happens to hold them in.

**P-2 — determinism.** The arm draws nothing from the generator and consults no clock, so a refused
transfer and a completed one both leave the generator's state exactly as they found it.

**P-3 — the giver is re-seated, not emptied in place.** After the pour, the giver's container is a
container that holds nothing — the same state every entity that has never held anything is in — and
not a distinguishable "used" one. A world in which the witch has given her potions away and a world
in which she never had any are the same world in that respect, and encode identically.

## What this story does not do

- **The notification packet.** The routine the original calls twice, once per owner, sits behind a
  gate that research has located and not interpreted. Nothing is built for it.
- **Drawing.** The icon and the name both already resolve; the inventory window's layout and any
  hover behaviour are a separate story.
- **Spawn-time equipment.** A parallel story owns it.
- **A player-issued transfer.** This is a script arm. Nothing here gives a player a way to move a
  container, and no command kind is added.
