# 0147 — the party comes from the save

**Intensity:** spec-first / correctness-over-breadth. **Terrain:** brownfield defect over a
mechanism that reported success. **Threshold: High** — every value this story applies reaches a
restored character, and a restored character is hashed simulation state.

## Why

The owner loaded one of his own `game####.sav` files in the shipped build and got a fresh mission
start. Nothing of his progress arrived.

The cause is two lines. The party handed to the mission opener was `f.NextParty()`, which is the
party a fresh start builds, so nothing from the file reached it. The save's actors were joined to
the map's own unit records by map unit id, and a save's own characters are not map placements, so
they carry no id to join on and their saved positions were dropped. Measured over the owner's
fourteen install saves before this story: `MOVED` between 0 and 27 of the map's units, and 0 in two
of the files.

The build said so, and said it in a form that read as the opposite. `OriginalSaveNote` said "map
unit positions only", which is true of the file and reads as "your positions are carried". The
positions a player checks first were the ones being dropped.

The object walk EXP-0147 published is what makes the fix possible. It reaches the human
participant's own subtree — his characters, their statistics, their pools, what they wear and what
they carry — and stops at the terminator above it. That boundary is the scope, and the owner ruled
that it is enough to build on.

## Scope

**In:** the object-graph walk in `pkg/formats/sav`; the party a loaded original save opens with;
each restored character's cell, four statistics, both pool pairs, two regeneration periods, worn
set, pack and weapon; the withdrawal of a map placement a restored character claims; and the two
sentences the build shows about what a load carries.

**Out of scope:** authoring a save from nothing — this story adds no writer. Everything past the
terminator: the map's own units beyond the existing position join, ground items, sacks, corpses,
cell records, the block plane, the session block and its trigger latches. Our own `Snapshot`
format, which is a different path and is untouched. The serialized byte form and its version: no
field is added to any `sim` record, so the version constant is not spent.

## Terms

- **The walk** — running a class's serialization programme from a known start, recursively, over
  the archive's own object-reference protocol. Distinct from **the scan**, which finds 37-byte heads
  by pattern agreement and reads the two fields inside one. The scan cannot reach a member body.
- **A character** — one `Human` or `Unit` record in the human participant's own group actor lists.
- **A piece** — one `Item`, `Weapon`, `Armor` or `Shield` record a character references.
- **An item code** — the sixteen-bit word `(material << 12) | (slot << 8) | (shape << 5) | row`.
  The original assembles it at `item+0x40` and serializes it there; `data.ItemCode` is the same
  word with the same four fields.
- **Applied** — written into the world the mission builds, and therefore into the world hash.
  **Reported** — read, counted and printed, and not written.

## Functional requirements

**FR-1 — the party comes from the save.** A loaded original save opens the mission with the human
participant's own characters, in the order the file holds them, one party member each. The party a
fresh start builds is the FALLBACK, taken only where the walk reaches no character at all, and a
fallback is stated in the report rather than substituted silently. This applies on both doors: the
LOAD GAME window (`FrontEnd.RestoreOriginal`) and the developer resume (`ResumeOriginalSave`).

**FR-2 — each restored character stands where the file left him.** His cell and fine position come
from his own record's head, whether or not the map ever placed him. The cell replaces the drop walk
for that member entirely: the map's authorised start cell is where a fresh party begins and a
resumed character does not begin. His cell is marked occupied so that a member who carries none
still walks away from it.

Where a restored character carries a map unit id the map still holds a record for, that record is
withdrawn from the map before the world is built. The file names one person and the map names the
same person; without the withdrawal the world holds two entities for him.

**FR-3 — each restored character's decoded state is applied.** Precisely:

- the four statistics, on `data.Hero`, from which the whole derived-stat graph folds his eight
  combat numbers, his step rate, his sight radius and his reach exactly as it folds a generated
  character's;
- the health pair and the mana pair, from the file and NOT from the fold — both members of each
  pair together, because a pair is what a reader sees and one member from each of two programmes is
  a health bar whose ratio means nothing;
- the two regeneration periods, from the file;
- his worn set: the weapon at `+0x74`, the shield at `+0x78` and the twelve armour references, each
  routed into the equipment slot ITS OWN item code names. A piece whose code names no slot, or a
  second piece claiming a filled slot, goes into the pack rather than being dropped;
- his pack, in the order the file holds it, every element keeping its own code — being in the
  container is what says an item is not worn;
- his weapon, resolved from the code in slot 1 through the same code-to-weapon resolver a generated
  character's is resolved through, so his swing cadence is the one his sword assigns;
- his profile and his face, from the Humans row his own record's head names.

His drawn body, its directory and his class key are DERIVED from the restored worn set through the
same appearance law a generated character's are derived through. Nothing in any file states them.

**FR-4 — what is not carried is stated in the reader's units.** Two statements, and both must be
true after this story:

- `OriginalSaveNote`, the one line the LOAD GAME window shows beside a highlighted row, says what
  loads and what does not, in words a player can check against the screen. It fits 104 columns.
- `OriginalSaveResume`'s counted form names every axis that was read and not applied, counted in
  characters, pieces, spells and journal entries rather than in records, offsets or references. A
  count of zero is a statement; silence is not.

The unapplied axes it must name: the spellbook and its spell records, the journal and its entries,
the carry capacity, the two Unknown statistic words, the untrained skill sets, and everything
outside the party.

**FR-5 — the four hotfix rules this story absorbs.** `docs/hotfix/LEDGER.md`'s four rows owed by
0143 FR-7/FR-8 are absorbed here and their `Owes` cells discharged:

- **`7a993e9`** — LOAD GAME is on the brooch's top-right button as well as its key. The button is
  the door this story's restore is reached through, and 0143 FR-7's "the key is the only way" is
  superseded: both work.
- **`2aef7c3`, `e106956`** — the LOAD GAME window lists the original game's saves and opens one. The
  list, the label, the read-through-a-type-with-no-`Write` and the never-guessed non-ASCII label are
  this story's own contract for that window; 0144 L-3's "POSITION only" is superseded by FR-1 to
  FR-3.
- **`d7aedc6`** — the LOAD list reads the RESOLVED asset root, so a build launched through
  `AGAINROM_ASSETS` lists the install's saves. The restore has no input at all without it.
- **`7a69592`** — the window says what an original save does not carry and the resume report is
  printed rather than dropped. FR-4 is that rule, restated for what the build now carries.

## Acceptance

**AC-1** A save the walk reads opens with as many party members as the file holds characters, each
at the file's own cell, with the file's own health pair and mana pair.

**AC-2** A restored character wearing a weapon in slot 1, armour in another slot and holding an
item in his pack arrives wearing both pieces in those slots with that item in his pack.

**AC-3** A member carrying no `Saved` is placed and minted exactly as before this story: the drop
cell, the fold's own health on both fields, and the base constructor's two periods.

**AC-4** The walk ends on the back-reference tag and the `0000` word `SAV-TOPLVL-052` names, on
every file of the owner's install.

**AC-5** A class the walk has no programme for is refused, naming the class, and the characters
read before it still come back.

**AC-6** `OriginalSaveNote` is at most 104 columns and names both halves. The counted report names
every axis of FR-4.

## Prohibitions

**P-1 — no value whose meaning is Unknown reaches the simulation.** An Unknown statistic word, a
spell whose id is not decoded, a journal array whose contents are not decoded: each is read,
counted and left out. A restored character's unapplied axis is exactly what a fresh start produces.

**P-2 — no mapping is invented.** A saved item's identity is the code the file already holds
because `ITEM-APPEAR-023` establishes that word's four fields are the four fields this tree's own
code carries. Where no claim relates a file value to one of this tree's, the value is reported.

**P-3 — nothing is written into a game install.** The walk reads; `savtool` and `savecheck` read;
no path in this story opens a save for writing.

**P-4 — the byte form does not move.** No field is added to any `pkg/sim` record and the serialized
version constant is not spent. Version 42 was allocated to this story and is returned unused.

**P-5 — one derivation of each value.** The pool override is one function (`restoredPools`), the
equipment routing is one loop reading `data.EquipSlotFor`, and the appearance comes from the same
`data.HeroAppearance` call a generated character's comes from. No second copy of any of them.

## Disclosed divergences

**L-1 — a restored character arrives untrained.** No published claim locates a skill level or its
experience in a `Unit` record, so all six slots are zero. His combat numbers are folded from his
four restored statistics, which is the same graph a generated character's numbers come off, so he
fights at the numbers his statistics buy and not at the numbers his training bought.

**L-2 — a script arm naming a withdrawn unit finds no entity.** The script compile builds its
unit-id-to-entity map from the units the map still holds, so an arm targeting a withdrawn
mercenary by `Target_Unit` no longer resolves. The original keeps that actor addressable and hands
him to the player at the same time; this tree cannot do both. One figure the player commands is
worth more than an arm that may not exist, and the count of withdrawn records is in the report so
the choice is visible.

**L-3 — the mission around the party starts over.** Script state, trigger latches, ground loot,
sacks, corpses and the block plane are past the terminator. A resumed mission runs its authored
arms again from the beginning.

**L-4 — a save taken between missions is still refused.** It carries mission number 0 and the
previous mission's map name. Unchanged from 0144, and the refusal names why.
