# Spec — the inventory window

Intensity: **spec-first / static**. Terrain: **brownfield** at the equipment slot, the party's
assembly and the viewer's input and draw path; **greenfield** for the item code, the art naming and
the window itself.

## Terms

**Item code.** The sixteen-bit word an item carries as its appearance, and the same word a map
authors an item as. Four fields: **A**, bits 15..12; **B**, 11..8; **C**, 7..5; **D**, 4..0. B is
the equipment slot the item occupies and also its class; D is the position of its defining row
inside that class's collection.

**Seven-digit name.** The name a code addresses its art by: the four fields spelled into decimal.
When B is 14 — the class carried and never worn — C and D are spelled as the one byte they sit in
rather than separately. The stored widths never change.

**Equipment slot.** One of twelve places on a character where a worn or held item sits, numbered 1
to 12, each **occupied** or **empty**.

**Figure.** The full-length picture the window shows: a base sheet naming a character's body and
face, with one layer over it per occupied slot.

## Why

The player controls one figure, that figure holds a weapon, and nothing on screen has ever said so.
The weapon changed his numbers, then the body he is drawn as in the world — a few pixels at map
scale, which reads as nothing. The place the original answers this is not the world: it is a window,
and in it the weapon is a full-height layer painted over the body. That window is also the only
surface where an item is a thing rather than a consequence, so everything the area still owes —
carrying, picking up, dropping, wearing — becomes visible the day it lands.

## Scope

**In scope.** The item code as a value with four fields. The name it addresses art by, both forms,
total over the sixteen-bit domain, and the three archive addresses that name reaches. The equipment
slot widened from a bare definition row to a whole code. A weapon resolved from a name stating the
code of the item it resolved to. A window that opens and closes, draws one character's figure and
twelve slot cells, and draws a pack area.

**Out of scope, each named rather than left silent.**

| Cut | To what |
|---|---|
| the container an actor carries items in | one class with no slot count and no capacity, needing a stack with a count and a per-unit weight, a running load and the penalty that load causes |
| moving an item | equipping, picking up from a sack and dropping are one command with a source and a destination code, needing an order-to-state relay that does not exist |
| the paint order over twelve slots, the second layer four of them carry, and the two-handedness rule swapping which of the first two goes last | eleven slots stay empty, so none of it is observable |
| the shipped window art | its background, frame, arrows and two backpack buttons all ship and are deliberately unread |
| armours, shields and magic items | three definition tables this build resolves nothing from; only a weapon can occupy a slot |
| what a row says an item's material and shape are | undecoded. A code is composed from a name or arrives as one; it is never read out of a row |
| the map's authored inventories | the loot section names an owner for some records and the ground for others; only the ground ones are read, as before |
| items surviving a mission | no place to keep them and no second mission to keep them for |

## The contract

**FR-1 — an item code is four fields, and reading them is total.** Every one of the 65 536 codes
answers an A, a B, a C and a D at the widths above. None is refused or repaired, and no sentinel is
invented for a class that names no item.

**FR-2 — a code names seven digits, by the original's own two rules.** When B is 14 the name is A
and B as two digits each and the code's **whole low byte** as three; otherwise A and B as two digits
each, C as one and D as two. Both forms give exactly seven digits for every code in the domain, and
no code fails to name.

**FR-3 — a name reaches three archive addresses and this build composes no fourth.** The **icon**
is that name under the inventory tree with the sixteen-bit sprite extension. The **figure layer** is
that name with the 256-colour sprite extension, under the equipment tree, inside a **figure
directory**, inside the group the original names *primary* — the one of that directory's two groups
that holds a sheet for every slot. The **figure base** is a face number directly inside the same
directory. A figure directory is one of four, by whether the character is a mage and whether female.

**FR-4 — an equipment slot carries a whole code.** What an occupied slot holds is the item's code,
not a bare row. The body a character is drawn as in the world is still derived from the first slot
and is **unchanged for every character this build can assemble**: that derivation reads field D, and
a composed code's D is the row the weapon resolved to.

**FR-5 — a weapon resolved from a name states its own code.** Resolution yields, beside the numbers
it already yields, the code of the item that name denotes, at slot number 1 — the same resolution
and the same refusals, with no name resolving that did not before and none ceasing to.

**FR-6 — the window opens and closes on one input, over the running world.** It is closed when a
mission opens. The input opens it when exactly one character this build assembled is selected, and
closes it again whatever is selected then; the cancel input closes it and does nothing else; and it
closes on its own the moment its subject stops being the one selected character. While it is open
the world is still drawn beneath it and still advances, and opening or closing it issues no command,
moves no entity and changes no selection.

*Folded from hotfix `872487b` — see `docs/hotfix/ARCHIVE.md#872487b`.* The window's rectangle has
ONE spelling, and a press, a release or a gesture latched inside it reaches neither the command
path, the selection nor the camera. One pixel outside is the map's: the world runs beneath and this
is not a modal.

**FR-7 — what the window draws.** One character's **figure** — his base sheet, and over it the
layer of each occupied slot. **Twelve cells** in slot order, each occupied slot's cell drawing its
code's icon and each empty slot's cell drawing nothing on the cell's own ground. A **pack area** of
a fixed number of cells on the same ground, all empty. The whole surface is one picture, recomposed
only when what it states changes.

**FR-8 — a missing sheet is a drawn absence, not a failure.** An address the archive does not hold,
or a payload the sprite readers refuse, leaves that one cell empty or that layer unpainted. The
window still opens, every other cell and layer still draws, and the absence is reported once rather
than each frame. Nothing panics and no mission fails to start.

**FR-10 — the simulation does not move, and neither does the milestone.** No field is added to,
removed from or retyped in any simulation state type; the byte form's version literal and encoded
layout are unchanged; a world assembled from given entities encodes to the same bytes and the same
digest. The scripted mission the milestone check runs ends as it ended before, on both installs.

**FR-11 — the drawing tier opens nothing.** It receives decoded pictures and codes, looks nothing
up, reads no file and composes no archive address.

## Acceptance

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | codes putting every field at a boundary, all bits clear, all bits set, and a walk of the whole sixteen-bit domain | the four fields are read | each answers the value its bit range holds, the four recompose the code they came from, and none of the 65 536 is refused |
| **AC-2** | one code with B equal to 14, one with B not equal to 14, and a walk of the whole sixteen-bit domain | the name is composed | the first spells its whole low byte as three digits, the second spells C and D separately, and every code in the domain produces exactly seven digits |
| **AC-3** | each weapon literal character generation can hand out, on **each** install | its code is composed and all three addresses are looked up | the icon address is a shipped node every time; so is the figure base for the directory the character takes; and so is the figure layer wherever that directory carries the item |
| **AC-4** | a fresh character | a code is written into one slot and every slot is read back | that slot answers the code that was written and every other slot answers empty |
| **AC-5** | each trained skill this front end can generate a character for, and every name the resolution accepted and refused before this story | the party is assembled and every name is resolved | the member's world body is the same name it was, the art it selects for him is the same art, and every name resolves or refuses exactly as it did |
| **AC-6** | a mission just opened, with the character selected and with nothing selected | the input is applied, applied again, the cancel input, and the selection is then cleared with it open | it is closed at the start; it opens only with the character selected; the second application closes it; cancel closes an open one and leaves a closed one closed; clearing the selection closes it |
| **AC-7** | a character with his first slot occupied and his other eleven empty | the window is composed | the figure carries exactly two layers, the base beneath and that slot's above it; twelve cells are drawn, the first holding that code's icon and the other eleven none; the pack area is drawn at its fixed cell count and every one is empty |
| **AC-8** | a code whose icon the archive lacks, and a payload the sprite reader refuses | the window is composed | it composes, that cell is empty, every other cell is unchanged, and one report is made |
| **AC-9** | a world before and after this story | it is encoded and hashed | the byte form's version literal is unchanged, and identical entities encode to identical bytes and to the same digest |
| **AC-10** | the scripted mission the milestone check runs, on both installs | it is run to its outcome | the outcome and the tick it lands on are what the check already records |
| **AC-12** | the drawing tier's sources | they are read | no archive type, no file read and no composed archive address appears in them |

## Properties

**P-1 — invariant: one composition.** Exactly one expression in the tree turns a code into a
seven-digit name and one turns a name into each of the three addresses; every caller goes through
them.

**P-2 — invariant: totality.** No code, no equipment and no absent art makes naming, composing or
opening the window fail.

**P-3 — negative invariant: the drawing tier derives nothing.** It holds no archive, no definition
table and no equipment slot.

**P-4 — invariant: the window is inert.** For any sequence of openings and closings, the simulation
state and its digest are exactly what the same run without them produces.

**P-5 — negative invariant: no invented correspondence.** Nothing maps a statistic, a skill or a
body name onto a code. A code comes from a name's own three indices or arrives as one.

## Decisions and divergences

**DD-1 Field C is the shape index a weapon's own name yields.** Field C is undecoded and a code
cannot be named without one, so this build takes the index of the quality word the name begins with.
AC-3 fails if that is wrong, and the cost is bounded: a wrong C names a sheet that does not exist,
which FR-8 draws as an absence.

**DD-2 The figure directory and the face are authored.** Whether a generated character is female,
and which face he wears, are character generation's inputs and this tree has no character
generation. He is a male fighter at the first face.

**DD-3 The window is drawn with the frame the drawing tier already has.** The original's own
background, frame and buttons ship in the archive and are left unread, so this window is plainer
than the original's.

**DD-4 The pack is drawn and empty.** An actor has no container. Showing the area is truthful and
showing none would be a window missing a part the original has.

**DD-5 The binding is authored.** Nothing decoded names the key that opens the window.

**DD-6 The layers paint base-first, in slot order.** The original decides which of the first two
layers goes last from the body the weapon names. Only slot 1 can be occupied here, so that rule has
no two layers to order and is unimplemented, not contradicted.

**DD-7 A slot's code is a loader value.** It reaches no simulation type, no byte form and no digest,
and the figure is composed once at mission open: nothing in a running mission changes what a
character holds.

## Traceability

| Requirement | Criteria |
|---|---|
| FR-1 | AC-1 |
| FR-2 | AC-2 |
| FR-3 | AC-3 |
| FR-4 | AC-4, AC-5 |
| FR-5 | AC-3, AC-5 |
| FR-6 | AC-6 |
| FR-7 | AC-7 |
| FR-8 | AC-8 |
| FR-10 | AC-9, AC-10 |
| FR-11 | AC-12 |
| P-1 | AC-2, AC-3 |
| P-2 | AC-1, AC-2, AC-8 |
| P-3 | AC-12 |
| P-4 | AC-6, AC-9 |
| P-5 | AC-3 |
| DD-1 | AC-3 |
| DD-2 | AC-3 |
| DD-3 | AC-7 |
| DD-4 | AC-7 |
| DD-5 | AC-6 |
| DD-6 | AC-7 |
| DD-7 | AC-9 |
