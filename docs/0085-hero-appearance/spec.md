# Spec — the party member is drawn as what he wears

Intensity: **spec-first / static**. Terrain: **brownfield** in `pkg/game` (a placed party member's
class key changes source, and the entity push gains an override arm) and in `pkg/mapload` /
`pkg/render/terrain` (one field each); **greenfield** for the appearance law itself.

## Why

The one figure the player controls is drawn as an unarmed man holding nothing. It carries a class
key this project chose, and the key it chose names the roster's bare-handed human body.

Replacing that key with a better key would fix the picture and leave the defect: the value would
still be a constant this project invented. In the original a player's character does not carry a
drawn class at all — the shipped id is discarded on arrival and the class is **produced** from what
the character visibly wears, through a name. That production is what this story installs, and the
one step of it this tree cannot take is named rather than filled in.

## Requirements

**FR-1 — a party member's class key is derived.** The key a party member is placed with is produced
by the appearance law from that member's **body name**. No class-key literal remains anywhere on the
party's path, and nothing else in the tree derives one: an actor a map places keeps carrying the key
its record stores, unchanged.

**FR-2 — the name-to-class law, whole and total.** The tree holds the published mapping from a body
name to a class key — seventeen names onto sixteen keys — and it is **total**: a name the mapping
does not carry answers the key the original leaves standing, and reports that no name matched. Also
held, as one law with it: the suffix a second occupied equipment slot appends to a name, and the two
substitutions — the one a mage takes and the one a dying character takes.

**FR-3 — the body directory is derived.** The directory a body's art sits under is produced by the
published three-way: the mage arm, the fighter-with-no-armour arm, and the armour material's own
path over its sixteen published values. The party's own directory is that function's answer, not a
string written down beside it.

**FR-4 — a party member's pixels come from the composed path.** A party member is drawn from the
sheet at `units/<directory>/<body>/sprites.256`, and never from the class record's own art address.
Its canvas, its centre, its animation descriptor, its name and its corpse link come from the class
record its body name resolved to. The two halves are independent: which sheet is drawn does not
change the geometry, and which class record supplies the geometry does not change the sheet.

**FR-5 — one authored term, and it is named as authored.** The body name itself is authored, because
the ordered list the original's equipment slot indexes is not decoded and this tree has no such slot.
It is **one function**, it is the only value in this story that is not derived, and it is described
as authored at its own definition, here, and in the story's evidence. No mapping from a weapon, a
skill or a statistic to a body name is invented.

**FR-6 — an absent sheet is not a failure.** An install whose composed sheet is absent, undecodable
or palette-less leaves the member drawn from the derived class record's own art, and every mission
still opens and plays. A body name that resolves to no class record does the same.

**FR-7 — the simulation's state does not move.** No field is added to, removed from or retyped in
any simulation state type; the canonical byte form's version literal and encoded layout are
unchanged; a world assembled from given entities hashes exactly as it did. What changes is a
**value** a loader puts into an existing field.

**FR-8 — the divergences are disclosed.** Three terms of the published law are not reached by this
tree, and none is papered over:

- the appearance is derived **once**, where the party is built, not recomputed as the original does
  on ordinary state messages — this tree has no channel that can change what a member wears;
- the **dying** substitution is held by the law and is not reached: a fallen party member is drawn
  through the class record's own corpse link, which is the placed unit's rule;
- there is **no equipment channel** — no slot array, no shield and no armour — so the shield suffix
  and the material arm of the directory are held and unexercised by anything this tree runs.

Each is stated at the site that would otherwise imply otherwise, and in the story's evidence.

## Acceptance criteria

| | Given | When | Then |
|---|---|---|---|
| **AC-1** | each of the seventeen published body names | the name is mapped | it answers the published key for that name, and reports that a name matched |
| **AC-2** | a name the mapping does not carry — including a suffixed name that has no arm | the name is mapped | it answers the key the original leaves standing, and reports that no name matched |
| **AC-3** | a base name and an occupied second slot | the name is composed | the suffix is appended; with the slot empty the name is unchanged |
| **AC-4** | a mage, and a dying character of either kind | the name is composed | the published substitution is taken in each case, and a living fighter's name is untouched |
| **AC-5** | each of the sixteen material blocks, a mage, and a fighter with no armour | the directory is resolved | each answers its published directory, and the two special arms answer theirs regardless of the material |
| **AC-6** | a directory and a body name | the sheet address is composed | it is the published composition, and its sibling differs from it only by the published insertion |
| **AC-7** | the party this front end starts a mission with | the member is inspected | its class key equals the law's answer for the authored body name, and is not the key an unmatched name leaves |
| **AC-8** | a bundle holding the class record, and an archive holding the composed sheet | the body is loaded | the loaded body draws the composed sheet's frames and carries the class record's canvas, centre, descriptor, name and corpse link |
| **AC-9** | the same, with the composed sheet absent from the archive | the body is loaded | no body is produced, and the entity that would have used it draws the class record's own art instead |
| **AC-10** | a mission's world with a body resolved for the party member | the entity picture is composed | that member is drawn from the body's frames; every other entity is drawn from its own class record, unchanged |
| **AC-11** | a world before and after this story | it is encoded and hashed | the byte form's version literal is unchanged, and a world assembled from identical entities encodes to identical bytes and the same digest |
| **AC-12** | the authored body name and the authored trained skill | they are read together | they are the pair the authored rule names, so moving one without the other is caught |

## Properties

**P-1 — invariant: one source for a party member's class.** The key a party member carries is a
function of its body name alone. There is no second expression in the tree that produces one, and no
literal to disagree with the function.

**P-2 — invariant: the drawing tier learns nothing new.** Everything crossing into the drawing tier
is the shape it already carries: one loaded class. That tier gains no registry knowledge, no name
table and no path composition, and its permitted imports do not grow.

**P-3 — negative invariant: no invented correspondence.** Nothing in the tree maps a weapon, a shape
name, a skill slot or a statistic onto a body name. The authored name is a literal at one site and
is reached from nothing.

**P-4 — invariant: the placed population is untouched.** For every entity a map places, the class
key, the resolved art and the drawn frame are what they were before this story.

## Divergence from the published law

Beyond FR-8's three terms: the original's twelve-slot array is not modelled at all, so *which* slot
supplies *what* — a reading the corpus grades below the arithmetic around it — is never relied on.
The law here takes the two facts that reading rests on as its arguments: whether a suffix is due and
whether an armour material is known.

## Out of scope

Character generation; an equipment or inventory channel; the mage class axis and the sex axis; the
dialogue portrait; the sibling overlay sheet's use; per-tier colour tables for a body; the unit
panel's fields; any change to how a map's own placements resolve, animate, fall or are drawn.

## Traceability

| Requirement | Criteria |
|---|---|
| FR-1 | AC-7, AC-11 |
| FR-2 | AC-1, AC-2, AC-3, AC-4 |
| FR-3 | AC-5 |
| FR-4 | AC-6, AC-8, AC-10 |
| FR-5 | AC-7, AC-12 |
| FR-6 | AC-9 |
| FR-7 | AC-11 |
| FR-8 | AC-3, AC-4, AC-5 |
| P-1 | AC-7 |
| P-2 | AC-10 |
| P-3 | AC-12 |
| P-4 | AC-10 |
