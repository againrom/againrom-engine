# Spec — the hero is drawn as, and reaches with, the weapon he holds

Intensity: **spec-first / static**. Terrain: **greenfield** for the slots, the shipped body list and
the derivation; **brownfield** where a party is assembled, where a party member's entity is minted,
and where a weapon's row is resolved.

## Terms

**Equipment slot.** One of twelve places on a character where a worn or held item sits. They are
numbered 1 to 12. A slot is **occupied** or **empty**, and empty is a value rather than an absence.

**Definition row.** The position of an item's defining row inside the collection it was defined in.
It is the quantity the original carries in the low bits of an equipped item's appearance word.

**Body list.** The shipped ordered list of body names the original indexes with an occupied first
slot's definition row, less one. It is a text payload of the install, not a table of this project's.

**Body name.** One entry of that list. Everything downstream of it — the drawn class, the art
directory, the sheet address, the geometry, the corpse link — is already law in this tree.

## Why

The one figure the player controls holds nothing. He was given a weapon two stories ago and the
weapon changed his numbers, but not his picture: the picture came from a body name this project
wrote down, because when that law was installed nothing said which name a character wearing a given
item should take.

Something does. The step is a single lookup — an occupied first slot names a definition row, and
the row names a line of a list the game ships — and installing it is the difference between a hero
whose appearance is a constant and a hero whose appearance is a consequence.

The same slot answers a second question the tree currently answers by accident. A generated
character's blow leaves the mission start with **no reach written at all**, and reaches one cell
only because the simulation repairs an unwritten field. That is not the weapon's range and it is not
a rule; it is a normalisation standing in for a derivation. The two are one story because they are
one slot.

## Scope

**In scope.** Twelve equipment slots on a character, numbered and empty. An occupied slot's
definition row. The shipped body list, read from an install and held verbatim. The derivation from
the first slot to a body name, total. The retirement of the authored body name on the party's path.
A generated character's reach, taken from the weapon in his first slot.

**Out of scope, and each one named rather than left silent.**

| Cut | To what |
|---|---|
| the per-item figure and its layers | the original composes a figure from a sheet per occupied slot, in a fixed order, with a two-handedness rule deciding which of two layers is painted last. That figure is the **info window's**, not the world's: what a player sees of a weapon in the world is that the whole body sheet changed. This tree draws the world and has no info window, so the compositor, the order and the layer rule are all cut |
| the contents of slots 2 through 12 | modelled and left empty. Nothing decoded says what a hero of this tree wears, and inventing it would be the authored constant this story exists to remove, one slot over |
| containers and carried items | a carried item is a different place from an equipped one. There is no container here, and the story before this one already owes it |
| picking an item up, and dropping one | needs a container, a purse and an order-to-state relay, none of which exists. The story before this one cut the same thing and named the same debt |
| resolving an authored item's packed code to a definition row | that is a map-placed item's path. A party member's weapon arrives as a name, and only the name-to-row half is built here |
| the voice bank | the original selects one of eight banks partly from the first slot. There is no audio tier in this tree at all |
| the placed population | a map's own people keep the class their record stores and the art it resolves, exactly as before |
| recomputing appearance while a mission runs | there is still nothing that can change what a character holds after the start |

## The contract

**FR-1 — a character has twelve equipment slots, numbered.** They are numbered 1 to 12, every one
of them empty unless something fills it, and an occupied slot is distinguishable from an empty one
without reference to what it holds. The width and the numbering are the original's; a thirteenth
slot and a slot numbered 0 are both absent.

**FR-2 — an occupied slot carries a definition row and nothing else.** What a slot holds is the
position of its item's defining row in that item's collection. No name, no statistic, no kind and
no appearance word is carried beside it: the row is the only part of an equipped item this build
consumes, and carrying more would be modelling a channel this story does not open.

**FR-3 — the body list is read from the install and held verbatim.** It is the shipped text payload
at the address the original names, read from the install's own container. Every line of it is an
entry, **including an empty line**, and the list's length is whatever ships rather than a number
written here. No copy of it, and no transcription of its contents, is written into this repository.

**FR-4 — the body name is derived from the first slot, and the derivation is total.** An occupied
first slot naming a definition row inside the list answers the list's entry one before that row. An
**empty** first slot answers the bare-handed name, which is the list's own first entry. A row
naming no entry — before the list's start, past its end, or an empty entry — produces **no name**,
reports that it produced none, and invents nothing.

**FR-5 — no body name is authored on the party's path.** The name a party member is drawn as is
FR-4's answer for the weapon he was handed. No body-name literal remains anywhere a party is
assembled, and moving the weapon moves the name without anything else being edited.

**FR-6 — a resolved weapon knows its own definition row.** Resolving a weapon by name yields, beside
the numbers it already yields, the row it resolved to. It is the same resolution and the same
refusals: no name resolves that did not resolve before, and none stops resolving.

**FR-7 — a generated character's blow reaches his weapon's reach.** The reach a party member's
entity leaves a mission start with is the range of the weapon in his first slot, and a bare
member's is the one cell a bare person's already is. **No entity leaves a mission start with an
unwritten reach**, so no reach in a started world is the simulation's repair of a zero.

**FR-8 — the simulation's state does not move, and neither does the drawing tier.** No field is
added to, removed from or retyped in any simulation state type; the canonical byte form's version
literal and encoded layout are unchanged; a world assembled from given entities hashes exactly as
it did. The drawing tier gains no list, no slot and no lookup: everything crossing into it is the
shape it already carries.

## Acceptance

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | a fresh character | its slots are read | there are twelve, numbered 1 to 12, and every one is empty; filling one and reading it back answers what was put in, and every other slot still answers empty |
| **AC-2** | each of the two shipped installs | the body list is read | the same payload is read from each, every line is an entry including the empty one, and the entry count is the file's line count rather than a number this build holds |
| **AC-3** | a first slot occupied with a definition row inside the list | the body name is derived | it is the list's entry one before that row, and a name is reported as produced |
| **AC-4** | a first slot that is empty | the body name is derived | it is the list's first entry, which is the bare-handed name, and a name is reported as produced |
| **AC-5** | a first slot naming a row before the list's start, one past its end, and one whose entry is empty | the body name is derived | each produces no name and reports that it produced none |
| **AC-6** | each trained skill this front end can generate a character for | the party is assembled | the member's body name is the entry the shipped list holds for that skill's start weapon, and his class key is what the existing law derives from that name |
| **AC-7** | a party member holding a weapon of a given range, and one holding nothing | a mission is started | the first member's entity reaches that range and the second's reaches one cell |
| **AC-8** | any mission this front end can start | the started world is inspected | no entity in it carries a reach of zero, and no member's reach came from the simulation's own repair |
| **AC-9** | a world before and after this story | it is encoded and hashed | the byte form's version literal is unchanged, and a world assembled from identical entities encodes to identical bytes and to the same digest |
| **AC-10** | the party path | it is searched for a body-name literal | there is none, and the one expression that produces a name is the derivation |

## Properties

**P-1 — invariant: one source for a body name.** A character's body name is a function of his first
slot and the shipped list alone. There is no second expression in the tree that produces one, and no
literal beside it to disagree.

**P-2 — invariant: the drawing tier learns nothing new.** That tier gains no body list, no slot, no
definition row and no lookup; its permitted imports do not grow.

**P-3 — negative invariant: no invented correspondence.** Nothing in the tree maps a skill, a
statistic, a shape word or a material onto a body name. The only map from an item to a name is the
shipped list, indexed by the original's own arithmetic.

**P-4 — invariant: the placed population is untouched.** For every person a map places, the class
key, the resolved art, the drawn frame and the reach are what they were before this story.

**P-5 — invariant: no shipped payload enters the repository.** The body list is read from an
install at run time. Neither the file nor a transcription of its entries is committed, and no test
asserts its contents from a copy held here.

## Decisions and divergences

**DD-1 Eleven slots are modelled and empty.** The width is the original's and is worth having,
because a slot number is the only thing that makes "the first slot" mean anything. What goes in the
other eleven is not decoded for a character this tree builds, so they stay empty and the existing
appearance law keeps receiving the same "no second slot, no armour" answers it already receives.

**DD-2 Appearance is derived once, at the start.** The original recomputes it whenever a character's
equipment is re-sent. Nothing here can change what a character holds after a mission opens, so the
derivation runs where the party is assembled. This is the same divergence the body story disclosed,
narrowed rather than removed.

**DD-3 The dying substitution is still unreached.** The law holds it; a fallen member is still drawn
through his class record's corpse link, which is the placed unit's rule.

**DD-4 An empty line of the list is an entry.** The shipped payload contains one, and this build
keeps it, so its entries from that line onward sit one later than a reading that dropped it. Which
the original does is not established. The two readings cannot be told apart by anything this build
ships, because every weapon a character can be generated with names a row before that line.

**DD-5 A row naming no entry produces no name, and that is authored.** What the original does with
an index past its list is not established. The behaviour chosen is the one the appearance law
already gives an unmatched name: refuse totally, report the refusal, and let the caller draw the
member as it drew one with no name at all. It is a disclosed authored term, not a hold.

## Traceability

| Requirement | Criteria |
|---|---|
| FR-1 | AC-1 |
| FR-2 | AC-1, AC-3 |
| FR-3 | AC-2 |
| FR-4 | AC-3, AC-4, AC-5 |
| FR-5 | AC-6, AC-10 |
| FR-6 | AC-6 |
| FR-7 | AC-7, AC-8 |
| FR-8 | AC-9 |
| P-1 | AC-10 |
| P-2 | AC-6 |
| P-3 | AC-5 |
| P-4 | AC-8 |
| P-5 | AC-2 |
| DD-1 | AC-1 |
| DD-2 | AC-6 |
| DD-3 | AC-6 |
| DD-4 | AC-2 |
| DD-5 | AC-5 |
