# 0156 — mission 30: the item instants and the finishable map

This is the contract. It states what the build must do and how each statement is
judged. Research provenance is in `provenance.md` (B2); nothing here depends on
reading it.

## Contract

**Mission 30 can be finished by a player, and the two item operations its script
authors run.**

The map creates a quest item at start, hands it to the first hero, and destroys
it at the destination. Its win is one distance test and nothing in the map tests
for the item. Both halves are built here: the operations run, and the win is
reached in play.

## Scope

The mission script's instant opcodes 12 (add item) and 13 (take item), the
compiled item reference both read, the byte form that carries it, and one
mission-stage scenario that walks the map to its win.

Check opcode 17, `Item in inventory`, is **out of scope**. `30.alm` authors none
and arming it would change trigger inertness on other maps.

## Functional requirements

**FR-1 — the compiled item reference.** A script node's first `Target_Item`
parameter is carried onto the compiled record as a **packed item code**, computed
as `0x0e18 + V` and stored as a 16-bit word, together with its own presence flag.
Later `Target_Item` slots on one node are dropped. A node naming no item carries
no code and the flag is clear. The reference is **carried, not resolved**: there
is no table to look it up in and nothing that can fail.

**FR-2 — instant 12, add item.** The arm creates **one unit** of the node's item
code and puts it in the named unit's own container. Where that container already
holds the code, the unit joins that element and its count rises by one; where it
does not, a new element appends at the tail. The arm reads no source and takes
nothing from anywhere.

**FR-3 — instant 13, take item.** The arm removes **exactly one unit** of the
node's item code from the named unit's own container. An element at a count of 1
is removed entirely; an element above 1 has its count reduced by one and stays
where it is. A code the container does not hold changes nothing.

**FR-4 — four refusals, each leaving the world as it was found.** Either arm
changes nothing when the node binds no unit, when it binds no item, when the
bound unit is not in the world, and when the code is zero. A zero is not an item:
class is bits 8..11 and a class of zero resolves to nothing.

**FR-5 — the arms write the container and nothing else.** Neither touches the
twelve worn equipment places, the purse, the position, the owner, the group, the
health, the order or the tick. Neither arms a trigger: inertness is derived from
unimplemented **checks** and neither arm adds one.

**FR-6 — the byte form carries the reference.** The compiled instant record
carries the item code and its presence flag, so a world encoded and decoded again
holds the same compiled program. The form's version rises; the previous versions
are refused as every earlier version is.

**FR-7 — a scenario may assert what a unit carries.** A mission-stage
`assert_unit` step accepts a `carries` expectation: a list of `code` values, each
optionally with an exact `count`, and optionally the assertion that a code is
**absent**. This is the observable that shows an item arriving and being
destroyed inside a real map.

## Acceptance criteria

**AC-1** A world whose script runs an instant-12 node with code `0x0e1e` against
a hero leaves that hero's container holding one element, code `0x0e1e`, count 1.

**AC-2** The same node run twice leaves **one** element at count 2, not two
elements.

**AC-3** An instant-13 node against a container holding count 2 of the code
leaves count 1 at the same place; against count 1 it removes the element; against
a code the container does not hold it changes nothing.

**AC-4** Instant 13 against a container holding two different codes removes one
unit of the named code only, and leaves the other element's code, count and place
unchanged.

**AC-5** Neither arm changes any entity's equipment, purse, position, owner,
group, health, order or the world tick.

**AC-6** A node binding no unit, a node binding no item, a node naming an entity
the world does not hold, and a node whose code is zero each leave the world
byte-identical to what it was.

**AC-7** A world whose compiled script carries item-bearing instants encodes and
decodes to an equal world, and a buffer declaring either of the two previous
versions is refused.

**AC-8** Mission 30 compiles with **no** unrunnable instant-12 or instant-13
node, on both preserved roots.

**AC-9** A mission-stage scenario over the lawful EN root starts mission 30, finds
the first hero holding code `0x0e1e`, walks that hero to the destination, reaches
the **won** outcome, and finds the code gone from both heroes' containers.

## Properties

**P-1** The container invariant holds through both arms: after either, no
container holds two elements naming one code, and no element has a count of zero.

**P-2** Which entity each arm writes is a property of the id the node names, not
of the order the world happens to hold its entities in.

**P-3** The compiled item code is a property of the map. The same map compiles to
the same code on both preserved roots.

## Out of scope

Check opcode 17. Instant opcode 11, the transfer, which no shipped map authors.
What the created object *is* beyond its packed code: the class-14 row's own
identity is not read here. Instant opcode 2, the broadcast, which writes no
simulation state.
