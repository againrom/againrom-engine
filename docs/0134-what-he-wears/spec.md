# 0134 — the character is drawn as what he wears, and a mage is a mage

**Intensity:** spec-anchored / static. **Terrain:** brownfield (the appearance law's callers, the
party builders, the art loader and the frame driver); greenfield for the reporting tool.

## Terms

- **Equipment slot** — one of a character's twelve numbered places, 1 to 12. Slot 1 holds his
  weapon, slot 2 his shield, and slot 8 the piece that decides which directory his art is loaded
  from.
- **Body name** — the string a character's visible equipment composes, and the directory element
  his sheet sits under.
- **Body directory** — one of the two directories hero body art ships under.
- **Base row** — the shipped person row a generated character's archetype starts from.
- **Equipment cell** — one of the ten item names a base row carries: cell 0 a weapon, cell 1 a
  shield, cells 2 through 9 each an armour.
- **Bundle** — the loaded art a running mission draws its characters from.
- **Carried** — held in the character's pack rather than worn in a slot.

## Problem — current behavior

A generated character is handed exactly one weapon and nothing else, so he wears no clothing at
all whatever archetype he was generated as, and mission 10's character arrives in nothing.

The body he is drawn as is composed from that weapon alone: no shield ever changes it, and a
character generated as a mage is drawn as a fighter, because the substitution that turns the
bare-handed body into the mage's is never applied. The directory his art is loaded from is fixed
at the fighter-with-no-armour answer whatever he wears.

The weapon he is handed is chosen by his trained skill alone. A mage's five skills occupy the same
five slots a fighter's do, so a character generated as a mage picking his first skill is handed
the fighter's sword.

Exactly one body's art is loaded per process, before generation runs and before any mission opens,
and the bundle holds it under its name alone. Nothing re-derives the drawn body after the
character's equipment changes during a mission: the equip path rewrites his combat numbers and the
name the panel shows him holding, and his picture does not move.

## Functional requirements

**FR-1** A generated character MUST start wearing what his own base row states: each of that row's
equipment cells other than the weapon cell MUST be resolved and placed in the slot the resolved
piece's own row names, or carried when it names no slot among the twelve. A cell naming nothing,
and a cell this build cannot resolve, MUST leave every slot and the pack exactly as the cells
around it left them, and MUST NOT fail the assembly.

**FR-2** The weapon a generated character is handed MUST be decided by his class first and his
trained skill second. A character generated as a mage MUST NOT be handed a fighter's weapon: he
MUST be handed the mage's own weapon, and that weapon MUST NOT vary with the skill he trained. A
character generated as a fighter MUST be handed his trained skill's weapon exactly as before.

**FR-3** The weapon so handed MUST occupy his weapon slot, and MUST be the same weapon his combat
numbers are folded from.

**FR-4** The body a character is drawn as MUST be composed from his whole visible equipment: the
base name from the item in slot 1, a suffix when slot 2 is occupied, and the mage substitution
when the composed name is the bare-handed one and the character is a mage.

**FR-5** The directory that body's art is loaded from MUST be the law's three-way over the same
equipment: a mage's own directory; otherwise, when slot 8 is empty, the unarmoured directory;
otherwise the directory named by the material of the piece in slot 8. There are sixteen materials
and a material outside them MUST be refused rather than answered with a directory.

**FR-6** When what a character wears changes while a mission runs, his drawn body and its directory
MUST be re-derived from his equipment as it then stands, and the picture MUST follow no later than
the first frame drawn after the tick that applied the change.

**FR-7** The bundle MUST be able to hold more than one body at once, and MUST address a body by its
directory together with its name, so that two directories shipping a sheet under one name cannot be
confused for one another.

**FR-8** A body whose art cannot be resolved — no such entry, a stream the decoder refuses, a sheet
of no frames, or a derivation that produced no name — MUST leave the character drawn exactly as he
was drawn before, and MUST NOT fail a mission, a load or a start-up.

**FR-9** The body a mission opens on and the body a refresh derives MUST come from one expression,
so that a character's picture at the open and his picture after an equip cannot disagree about the
same equipment.

**FR-10** No equip MUST be refused on account of the character's class.

**FR-11** A developer tool MUST report, for a lawful install and without writing any file, what
each generated archetype starts wearing and holding, the body name, directory, sheet address and
drawn class each composes, and whether that sheet actually loads out of the install.

## Acceptance criteria

| # | GIVEN | WHEN | THEN | Level |
|---|---|---|---|---|
| AC-1 | a base row naming an armour in a cell past the weapon cell | a party is assembled for that archetype | the worn set holds that piece's code in the slot the piece's own row names | unit |
| AC-2 | a base row naming a piece whose row names no slot among the twelve | a party is assembled | that piece is carried and no slot is overwritten | unit |
| AC-3 | a base row naming a weapon in its weapon cell | a party is assembled | slot 1 holds the weapon character generation handed him, not the row's | unit |
| AC-4 | a generation result choosing the mage class and any skill | that result is turned into a party | the member holds the mage's weapon and no fighter's weapon | unit |
| AC-5 | a generation result choosing the fighter class | that result is turned into a party | the member holds exactly the weapon his trained skill named before this story | unit |
| AC-6 | an equipment set whose slot 2 is occupied | the body name is composed | the name carries the suffix | unit |
| AC-7 | an equipment set composing the bare-handed name, for a mage | the body name is composed | the name is the mage's | unit |
| AC-8 | an equipment set whose slot 8 holds a piece of a given material, for a non-mage | the directory is derived | it is that material's own directory, and a material outside the sixteen is refused | unit |
| AC-9 | two characters differing only in sex | their bodies are derived | the body name, the directory and the drawn class are identical | unit |
| AC-10 | a running mission whose character equips a weapon composing a different body | the tick that applies it and the frame after it | the class he is drawn as is the new body's | unit |
| AC-11 | a running mission whose character's equipment does not change | any number of frames | the bundle is asked to load nothing and the drawn class does not move | unit |
| AC-12 | a bundle with no entry for the derived body, and an archive that cannot supply one | the appearance is refreshed | the character stays drawn as he was and nothing fails | unit |
| AC-13 | a lawful install | the reporting tool is run against it | it prints each archetype's worn set, body name, directory, sheet address, drawn class and whether that sheet loads, writes no file, and exits zero | manual |
| AC-14 | a character of any class holding an item of any other class's stated suitability | that item is equipped | the equip is applied | unit |
| AC-15 | a base row whose cells include an empty one and one naming no row this build can resolve | a party is assembled | the worn set and the pack hold exactly what the row's other cells resolved to, and the assembly succeeds | unit |
| AC-16 | a mage, and separately a non-mage whose slot 8 is empty | the directory is derived | the mage takes the mage directory whatever his slot 8 holds, and the non-mage takes the unarmoured one | unit |
| AC-17 | two bodies of the same name under two different directories | both are loaded into one bundle | both entries are present and each resolves to its own directory's art | unit |
| AC-18 | one equipment set | a mission opens on it, and separately a refresh derives from it | the body name, the directory and the drawn class are equal | unit |

**Error cases:** AC-2, AC-8's refusal, AC-12, AC-15 and FR-8 are the error boundaries.

## Derived properties

- **P-1 (completeness)** For any equipment set and any pair of class and dying flags, the
  derivation answers a body name, a directory and a drawn class — including for an empty set and
  an empty body list.
- **P-2 (invariant)** For any character, the drawn class is the class the composed body name
  resolves to, and no other value is written anywhere.
- **P-3 (negative-invariant)** For any body whose art cannot be resolved, no entry is added to the
  bundle and the character's drawn class is left exactly as it was.
- **P-4 (negative-invariant)** For any change of what a character wears, nothing is drawn over his
  world sprite: the whole sheet is what changes.
- **P-5 (idempotence)** For any unchanged equipment, refreshing the appearance any number of times
  performs no load and leaves the drawn class equal.
- **P-6 (invariant)** For any generated character, the weapon in slot 1 and the weapon his combat
  numbers are folded from are one resolution.

## I/O examples

The reporting tool takes `-assets DIR` or `AGAINROM_ASSETS`, writes only to standard output, and
exits non-zero only when the install cannot be read. One archetype prints as one block naming the
archetype, the weapon it was handed, each occupied slot with its number, item name and code, the
composed body name, the directory, the composed sheet address within the graphics container, the
drawn class, and whether that sheet loads:

```
mage / male   weapon=<name>
  slot 1 <name> <code>
  slot 7 <name> <code>
  body=<name> dir=<dir> class=<n> sheet=<address> loads=yes
```

## Constraints

| Choice | A | B | C |
|---|---|---|---|
| Which bodies are available to draw | load every reachable body at start-up | **load a body when it is first needed and keep it** | load and discard per change |
| A generated mage's weapon | a fighter's weapon | **the mage's own weapon** | none at all |
| Where the refresh runs | inside the simulation tick | **on the frame-paced path** | on the draw path |

A for the bodies decodes some thirty sheet pairs at every start-up for a character who may never
change a garment; C re-decodes on every change for no bound gained. A for the weapon is the
reported defect; C is authorised and further from the game than B, and is not needed. A for the
refresh puts drawn state inside hashed state; C reaches an archive from a draw.

## Out of scope

- What a worn piece contributes to protection or absorption: no worn piece may change a damage,
  defence or absorption number. A change of weapon still folds into the wielder's numbers exactly
  as it does today, so a mage handed a different weapon fights at that weapon's numbers.
- Equipping a piece of armour out of the pack.
- What a fallen character is drawn as. The two dying body names stay reachable in the law and
  unreached by any path here; the existing divergence stands.
- The overlay sheet loaded beside each body sheet.
- The inventory window's paper doll, which already follows equipment.
- Any change to what a character's health, mana or experience is.

## Verification mapping

AC-1 to AC-12 and AC-14 are CI-automatable against synthetic fixtures. AC-13 needs a lawful
install and is run by hand against both roots.

## Gate check

FR-1 covered by AC-1, AC-2, AC-15. FR-2 by AC-4, AC-5. FR-3 by AC-3, P-6. FR-4 by AC-6, AC-7,
AC-9, P-1, P-2. FR-5 by AC-8, AC-16, P-1. FR-6 by AC-10, P-4, P-5. FR-7 by AC-17. FR-8 by AC-12,
P-3. FR-9 by AC-18, AC-11. FR-10 by AC-14. FR-11 by AC-13.
