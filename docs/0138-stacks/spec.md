# 0138-stacks — specification

## Intensity and terrain

**spec-anchored / static.** The contract outlives the ticket: it fixes the shape of an actor's
container, which every later item story reads.

**Brownfield** for `pkg/sim`'s container, the equip move, the corpse drop, the give-all instant and
the byte form's carry section; **brownfield** for the inventory window's pack area; **greenfield**
for the drive's take arm. Every changed unit is pinned by an existing test before it changes.

## The behaviour

Three identical potions an actor picks up occupy **one place in his container, marked ×3** — not
three places. That is the whole of what a player sees.

Underneath it, a container stops being a list of item codes and becomes a list of **stacks**: an
element is one code together with how many of it are held. Two elements naming the same code cannot
coexist; whatever puts an item into a container — a take, a give-all, construction, a decode —
merges it into the element already there.

Nothing about how much a container may hold changes: there is no slot count, no capacity and no
weight anywhere in this build, and none is introduced.

## Terms

- **Container** — what an actor carries. Not his equipment slots, and not a sack.
- **Element** — one place in a container.
- **Stack** — an element: an item code and a **count**, the number of that item held there. A count
  is at least 1; an element with a count of 0 does not exist. A count is a **32-bit** unsigned
  number, which is the width the byte form already writes a container's length at, so folding a
  container can never lose a unit however many it holds.
- **Identical** — two items are the same thing when their **codes are equal**. This build's item is
  a bare sixteen-bit code, so there is no second term.
- **Flat expansion** — a container written out as one code per unit held, an element of count *n*
  contributing its code *n* times, in element order. It is what the container was before this story.
- **Fold** — turning a flat expansion into elements: equal codes collapse into the **first** place
  the code occupies, their counts summing.

## Functional requirements

**FR-1 — An element is a code and a count.** A container is a sequence of elements; each names one
item code and a count of at least 1. The count is canonical simulation state: it survives the byte
form and it enters the digest.

**FR-2 — A container holds at most one element per code.** Every act that puts an item into a
container merges it: construction from an authored loadout, a decode of the byte form, a take, a
give-all, and an item displaced out of an equipment slot. There is no act that leaves two elements
naming one code behind.

**FR-3 — A merge keeps the first place and sums the counts.** Element order is the order the codes
were first seen; a later occurrence of a code already held adds to the element that holds it and
occupies no place of its own. Merging is total: it refuses nothing and drops nothing.

**FR-4 — Reading a container as codes is unchanged.** The reader that answers *what does this actor
carry* answers with the flat expansion, in element order, exactly as it did before this story — as
does the whole-world holdings reader a rebuild uses. Feeding those holdings back into a fresh world
reproduces the same containers.

**FR-5 — A container can be read as elements.** One reader answers the elements themselves, each
with its count, in element order, as a copy no caller can mutate the world through. It answers
nothing for an entity the world does not hold.

**FR-6 — Taking a sack merges into the taker.** The whole of a sack becomes what the taker carries,
in the sack's own order, merged by FR-2 and FR-3. Nothing else about the transfer changes: not its
all-or-nothing outcome, not the gold, not what it refuses.

**FR-7 — A body drops units, not elements.** What a felled actor pours onto the ground is the flat
expansion of his container followed by what he wore in the order the drop already uses, so a stack
of three drops as three items. A sack is a flat code list and does not stack.

**FR-8 — Giving one actor's container to another merges it.** The receiver keeps his own elements
and gains the giver's, by FR-2 and FR-3; the giver is left with nothing.

**FR-9 — Equipping takes one unit off an element.** The command names an **element by its position
in the container**. One unit of that element's code moves into the named slot: an element of count
*n* > 1 is left at *n* − 1 in its own place, and an element of count 1 leaves the container.
Whatever the slot already held moves back into the container at that element's place — and where
the container already holds that code, FR-3 applies and it joins the element that holds it rather
than taking a place of its own. A command naming no element, or a slot outside the twelve, is
**ignored rather than reported**, and changes nothing.

**FR-10 — The byte form does not change and its version does not move.** A container is written as
its flat expansion — the carry section keeps its shape, its offsets and its meaning — and is read
back by folding it. Every world this build can construct encodes to the bytes its flat expansion
always encoded to, so no existing payload changes meaning. What a container holds after the acts of
FR-2, FR-8 and FR-9 can differ in ORDER from what the same acts left before this story, and where
it does, that world's digest differs; that is a change of behaviour and not of form.

**FR-11 — The pack shows one cell per element, with its count.** The inventory window's pack area
draws one cell per element in element order, and a cell whose element holds **two or more** carries
that number. A count of 1 is drawn as nothing. The window still composes with no font and with no
subject at all.

**FR-12 — The drive can take a sack and report stacks.** The mission drive can be told to have a
named unit take the sack at a named cell, after its attacks, and reports that unit's container as
elements with their counts. The holdings it already reports before a blow are reported the same way.

## Acceptance criteria

| ID | Given | When | Then |
|---|---|---|---|
| AC-1 | an actor authored with three of one code and one of another | the world is built | his container has two elements, the first at count 3, in first-seen order (FR-1, FR-2, FR-3) |
| AC-2 | that actor | his container is read as codes | four codes come back, the three equal ones adjacent and first (FR-4) |
| AC-3 | that actor | his container is read as elements | two elements come back, counts 3 and 1 (FR-5) |
| AC-4 | an actor holding one potion and a sack of two more on his cell | he takes the sack | he holds one element at count 3 (FR-6) |
| AC-5 | an actor holding an element of count 3 | he is felled on a cell a sack may stand on | the sack holds that code three times (FR-7) |
| AC-6 | two actors each holding two of one code | the give-all instant fires | the receiver holds one element at count 4 and the giver holds nothing (FR-8) |
| AC-7 | an actor holding an element of count 3 and an empty slot | he equips that element | the slot holds the code, the element remains at count 2 (FR-9) |
| AC-8 | an actor holding an element of count 1 and an occupied slot | he equips that element | the slot's old code takes that element's place, at count 1 (FR-9) |
| AC-9 | any world, including one holding an element of count 3 | it is marshalled and unmarshalled | it is the same world, every element and count included, and every digest this package pins is unchanged (FR-10) |
| AC-10 | a payload whose carry record names one code in two separate runs, and one naming a single code 70 000 times | each is decoded | the first folds to one element and nothing is refused; the second folds to one element at count 70 000 (FR-2, FR-3, FR-10) |
| AC-11 | a subject whose first pack element holds 3 | the window is composed with a font | the cell differs from the same cell at count 1 (FR-11) |
| AC-12 | that subject with no font, and the zero subject with a font | each is composed | both compose, and nothing panics (FR-11) |
| AC-13 | a mission, a felled unit's sack and a unit told to take it | the drive runs | the report names that unit's elements with their counts (FR-12) |

## Properties

- **P-1 — Folding is idempotent.** Folding an already-folded container returns it unchanged.
- **P-2 — Units are conserved.** No act in FR-2, FR-6, FR-8 or FR-9 changes the total number of
  units of a code held, except the one unit FR-9 moves into a slot.
- **P-3 — Expansion and folding invert each other** on every container this build can hold.
- **P-4 — A count is never 0.** An element that would reach 0 is removed instead. Counts sum with
  no limit test of any kind, and no container this build can construct can reach the count's own
  width: a container is written with a 32-bit length, so its flat expansion can hold no more units
  than a 32-bit count can carry.
- **P-5 — Nothing panics.** No count, however large, and no container, however long, makes any
  reader or the window fail.

## Out of scope

- **Splitting a stack by a chosen quantity.** The mechanism is decoded, but no screen in this build
  asks for a quantity; FR-9's one unit is the only division that exists.
- **Weight and a container's load.** Nothing in this build carries a per-unit weight or a running
  load, and `Σ weight × count` is not implemented, stubbed or reserved.
- **The shop**, its generator and its return path.
- **Stacking in a sack or in an equipment slot.** Both stay flat.
- **Any meaning for `+0x44`, the effect list, or the flag word a merge ORs together.** None of the
  three is a field of an item in this build, and none is added.
- **Raising the pack area's eight cells.** A container is still unbounded and the window is not.

## Disclosed divergences

1. **Merging on every add is authored, not decoded.** The original's container class has a merge
   sequence; which of its callers run it is not established, and its shop generator demonstrably
   does not dedup. This build merges always.
2. **Every item is stackable here**, because nothing in this build distinguishes two items of the
   same code. The original separates them by enchantment.
3. **The count is 32 bits wide; the original's field is 16.** The count here groups a list the byte
   form already writes with a 32-bit length, and a narrower count could silently lose units of a
   container that build can construct today.
</content>
</invoke>
