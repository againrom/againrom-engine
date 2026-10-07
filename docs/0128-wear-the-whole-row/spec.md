# spec — 0128 a person wears his whole row

A map-placed person arrives holding only his weapon. His definition row names up to ten items and
nine of them are discarded without a word. This story resolves all ten and puts each where the
original puts it.

## Terms

* **Equipment cell** — one of the ten item-name strings a person's definition row carries, indexed 0
  through 9. An **empty cell** is one whose string is empty; the row writes a fixed ten however many
  it names.
* **Worn set** — the twelve equipment slots an entity already carries, numbered 1 to 12, the zero
  code meaning a free slot.
* **Piece** — a resolved shield or armour, as against the weapon, which this tree already resolves.
* **Slot column** — the column of an armour's own definition row naming which of the twelve slots it
  takes. It is a property of the ITEM, not of the cell it was named in and not of the wearer.
* **Hands column** — the column of a weapon's own definition row whose value 2 marks a weapon that
  occupies both hands.
* **Container** — the entity's carried list, which the simulation already holds.
* **Prefix table** — the shape table or the material table, each a collection whose entry names may
  lead an item's name.

## Functional requirements

**FR-1 — the ten cells are resolved by POSITION.** Cell 0 names a weapon, cell 1 names a shield,
cells 2 through 9 each name an armour. The class is the cell's, never the string's: nothing reads
the name before deciding what class to resolve it as. An empty cell is skipped and is not a failure.

**FR-2 — one name parse, and it is the original's own walk.** A name is reduced to a row name by, in
order: cutting everything from the first `{` and trimming; taking a shape word; taking a material
word; and — for a shield or an armour only, never a weapon — re-attaching the shape word the
material implies. A shield additionally drops a trailing ` Shield` from what is left. The remainder
must name a row of the class's collection EXACTLY.

**FR-2a — taking a prefix word is a DESCENDING walk with a substring match.** A prefix table is
searched from its last entry to its first; the first entry whose name occurs anywhere in the subject
wins, and the subject becomes everything before that occurrence followed by everything from one past
the entry's own length. The two are the same string only when the occurrence is at the start, which
is the ordinary case; a name authored out of order is therefore mangled rather than refused, and that
is the contract, not an accident of it. No match leaves the subject whole and answers index 0.

**FR-2b — the implied shape word.** After the material word is taken, a shield or an armour whose
material name contains `Leather` has `Soft ` put back on the front of what is left, and one whose
material name contains `Wood` has `Wooden ` put back. What is left is LEFT-TRIMMED first, so that the
re-attached word is followed by exactly one space however many the rebuild of FR-2a left standing. A
weapon never has any of it.

**FR-3 — a resolved item's destination is fixed by its class.** A weapon takes slot 1; a shield
takes slot 2; an armour takes the slot its own Slot column names. The slot and the item code agree
by construction — an item code's class field IS its slot — so nothing states the destination twice.

**FR-4 — the refusals are implemented, not just the happy path.** A shield whose Slot column would
be anything other than 2 does not exist; a shield is refused when its name resolves to no row. An
armour is refused when its name resolves to no row, when its Slot column is zero, and when its Slot
column is above twelve.

**FR-5 — a name that resolves to no row is not an item.** It is neither worn nor carried. A name
this build cannot resolve, an empty cell and a class it cannot model are one outcome and not three.

**FR-6 — a piece that resolves but cannot be worn is CARRIED.** An armour whose row resolved and
whose Slot column is zero or above twelve goes into the container instead of a slot. This is the
only way a resolved piece reaches the container other than FR-7.

**FR-7 — the two-handed displacement.** A shield taking slot 2 while slot 1 holds a weapon whose
Hands column is 2 first takes that weapon out of slot 1 and into the container. The shield is then
worn. Nothing else moves.

**FR-8 — the corpse gate reaches slots 1 and 2 alone.** The existing rule that a person whose
template name contains `NPC` starts with an empty weapon slot is restated as: such a person starts
with slots 1 and 2 empty and every other slot as FR-3 filled it. A corpse gives up only those two
slots, so the armour slots are outside the gate's reach by construction rather than by exception.
`0132-corpse-drops-worn` removed that reason by widening the drop to all twelve slots, so this gate
is now narrower than the drop it once matched.

**FR-9 — the whole set is one resolution, in cell order.** Cells are resolved and placed 0 first,
then 1, then 2 through 9 ascending, because FR-7 reads what an earlier cell already wore.

**FR-10 — the panel may state the worn set.** The unit panel gains a row naming what its subject
wears, by the pieces' own row names in slot order. A subject stating nothing worn contributes no row
and costs no space, exactly as every other absent row already does.

**FR-11 — the resolution rate is MEASURED, not asserted.** A developer tool reads a lawful install,
resolves every equipment cell of every row of the shipped person collection by FR-1 and FR-2, and
reports per cell class how many resolved, how many were refused, and where each resolved item landed.

## Acceptance criteria

**AC-1** A row naming a weapon in cell 0, a shield in cell 1 and armour in cells 2..9 produces an
entity wearing all ten, each in the slot FR-3 names, with an empty container.

**AC-2** A row whose cell 2 names an armour whose Slot column is 7 and whose cell 8 names an armour
whose Slot column is 4 wears the first in slot 7 and the second in slot 4. Neither lands in the slot
its cell index would suggest.

**AC-3** A row whose cell 1 names a shield and whose cell 0 names a weapon whose Hands column is 2
produces an entity wearing the shield in slot 2, slot 1 empty, and the weapon in the container.
With a Hands column of 1 the same row wears both.

**AC-4** A row naming an armour whose Slot column is 0, and one naming an armour whose Slot column is
13, each produce an entity with that piece in the container and no slot written.

**AC-5** A row whose armour cell names no row at all produces an entity with nothing extra worn and
nothing extra carried, and the rest of the row unaffected.

**AC-6** Against the shipped person collection of a lawful install, the tool of FR-11 reports the
resolution rate, and `verification.md` records the number it printed and the divergence FR-5 costs. Both preserved roots must agree.

**AC-7** A person whose template name contains `NPC` and whose row names a weapon, a shield and
armour starts with slots 1 and 2 empty and every armour slot filled.

**AC-8** The panel of a subject wearing pieces states them; the panel of a subject wearing nothing
composes byte-for-byte the picture it composed before this story.

## Properties

**P-1** The worn set, the container and the digest are the ones the simulation already holds. This
story writes no new simulation field, widens no byte form and moves no format version.

**P-2** No derived number moves. Nothing here reads or writes defence, absorption, protection,
resistance, to-hit or damage from a worn piece: what a piece is worth is not decoded and the
additive block stays at its structural zero with its disclosure intact.

**P-3** Every function that resolves a name is total: it answers a value and a refusal, never a
panic, for any string, any table and any absent collection.

**P-4** A table missing any collection a class needs answers as a table naming that class's items
nowhere — every such cell refused — rather than resolving some cells off whichever collections are
present.

**P-5** The parse is stated ONCE. The weapon path, the shield path and the armour path share it and
differ only by the two documented steps FR-2 gives each of them.

## Scope

**In:** resolving all ten equipment cells of a person's definition row; wearing each resolved piece
into the existing twelve-slot set with the refusals above; the container overflow; the panel row;
the measuring tool.

**Out:** what a worn piece contributes to any derived number — not decoded, and the additive block
stays zero. A placed person's health maximum. What a corpse drops. Any equipment a units-band
placement or a generated character names. Re-equipping or unequipping at run time, which the
simulation's own command already does.
