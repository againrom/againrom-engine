# Spec — the definition table, and the numbers a placed unit is built with

## Problem and current behaviour

A world built from a map gives every placed unit the same health pair — one constant, declared
provisional at the spawn site that sets it — because nothing here reads the file holding the real
numbers. A placement's class key reaches the graphics registry, which holds the drawable's record
and no stat, so a wolf and a dragon are the same monster in the only quantity the simulation has.

## Terms

A **collection** is one of the table file's eleven by-value tables, named by group below. An
**entry** is one row of one — a name, usually a parameter array, per-kind extra fields. A **slot**
indexes that array, slot *i* being the collection's column *i+1*. A **unit definition** is a Units
entry resolved into named stat fields. **Difficulty** is the three-valued scenario setting a spawner
applies to what it builds.

## Functional requirements

- **FR-1** The table file MUST be read as the eight groups below, in that order: a group is a string
  array of the titles its collections share, then each collection as a `u32` count and its entries.
  In the **first two groups** every entry the count names is present; in the **other six** the count
  is one larger than the entries on the wire — entry 0 is allocated and never written — so entries
  run 1 to count−1 and every subscript is 1-based.
- **FR-2** The walk MUST consume the payload **exactly**: a length past the end, an unsatisfiable
  count, an undefined string-length escape and any byte left over after the last group MUST each be
  an error, and MUST NOT yield a partially filled table.
- **FR-3** A parsed file MUST expose, per collection: its entries at their own indices — index 0 of
  a 1-based collection present and empty — each entry's name bytes, its parameters as signed 32-bit
  values with −1 preserved, its extra strings and raw bytes; and its group's titles, which sibling
  collections share. **No character encoding is applied**, and parameters this contract gives no
  meaning to MUST be exposed whole rather than dropped.
- **FR-4** A unit definition MUST be the constructor's own defaults with a Units entry's slots 0 to
  37 written over them **in ascending order**, per the slot map below. A slot holding −1 MUST leave
  the default standing and be consumed all the same. Writing the health maximum MUST also set
  health, and the mana maximum mana. **Slots 34–36 MUST be consumed and MUST NOT be carried.**
- **FR-5** The damage pair MUST be built and routed as the map below states, an absent selector
  taking arm 0 because that selector is **pre-set to zero**. Arms 1 and 2 name a **different** field
  pair this contract does not model, so an entry carrying either MUST be refused by name, not
  silently given no damage.
- **FR-6** A placement MUST be resolved by the arms below, in order, with both class keys truncated
  to their low byte, empty-named entries skipped, and the first match taken. No match MUST be
  reported as no match, never as an error. Which arm ran, and what it reached, MUST be observable on
  its own and not only through a world.
- **FR-7** Only a **Units** match MUST yield a stat block; a placement reaching a Humans entry,
  taking the npc path, or matching nothing MUST keep exactly the health pair a world built with no
  table gives it. A resolved definition MUST have the adjustment below applied **once** before
  anything is built from it, and at 1 and 3 health MUST then be set to the new maximum. Difficulty
  takes exactly 1, 2 and 3, with 2 what a caller who says nothing gets; any other value MUST be
  refused. The adjustment MUST apply to a unit definition alone, and be a function of it and the
  value alone.
- **FR-8** A world MUST be buildable from a map together with a table and a difficulty. Every
  placement resolving to a Units entry MUST take that definition's **adjusted** health maximum as
  both its health and its maximum, every other placement keeping the provisional pair. The existing
  single-argument entry point MUST keep its behaviour exactly — the same entities, byte form and
  digest — and a world MUST carry no field naming the table or the difficulty.
- **FR-9** A developer tool MUST read the table from a lawful install and report the bytes consumed
  against the file's size, each collection's entry and title counts, which Units rows carry
  parameters and how many, and — for a named map — each placement's arm, the entry reached and its
  adjusted health maximum at a chosen difficulty. No test may read an install.

## I/O examples

```
groups, in file order
  A titles Shapes Materials | B titles Magic | C titles Armors Shields Weapons
  D titles MagicItems | E titles Units | F titles Humans | G titles Buildings | H titles Spells
  0-based: A and B.  1-based: C..H.
wire primitives
  string = u8 len (0xFF -> u16 len) + bytes ; TITLE array = u16 count + strings
  param array = u16 count + count x u32 LE  ; collection count = u32
entry payload after the name, by collection kind
  Shapes Materials  9 x f64 raw, NO param array      Magic Buildings  params
  Armors Shields Weapons  params + 10 raw + params   MagicItems  params + 1 raw + string
  Units Humans  params + 2 / 10 uncounted strings    Spells  params + string
slot map, Units. defaults are the constructor's; -1 leaves them standing
   0 body 30  1 reaction 30  2 mind 20  3 spirit 20  4 healthMax 30 (health := it)
   5 hpRegen 100  6 manaMax 0 (mana := it)  7 mpRegen 50  8 speed 10  9 rotation 8
  10 scanRange 5  11,12,13 -> the damage pair, below  14 toHit  15 defence  16 absorption
  17 attackCharge 8  18 attackRelax 4  19..23 protection fire water air earth astral
  24..28 resistance, blade .. shooting  29 typeID  30 face  31 tokenSize 1  32 movementType 1
  33 dyingTime (consumed, dropped, re-read from this slot elsewhere)  34..36 consumed, not carried
  37 xpValue.  sight 0 and reach 1 have no column
damage pair: base = s11, spread = s12 - s11, selector = s13 with the local PRE-SET TO 0
  s13 <= 0 (and -1) -> the pair        3 -> the pair, and alwaysHits
  1, 2              -> a field pair this contract does not model -> refuse the entry
resolution, k = ClassID & 0xff, k2 = ClassSubID & 0xff
  Flags&1                        -> npc path, no entry
  DefID != 0 && != 0xcdcdcdcd    -> Humans, DOWNWARD from the last, serverID == DefID
  k < 0x40 && k != 26 && k != 27 -> Humans, upward from 1, typeID == k
  otherwise                      -> Units,  upward from 1, typeID == k && face == k2
adjustment, h = healthMax, t = toHit, d = defence, all integers, truncating
  1 -> h*66/100          2 -> unchanged          3 -> h*3/2, t+50, d+50
  h=30 -> 19 / 30 / 45     h=99 -> 65 / 99 / 148     h=1 -> 0 / 1 / 1
```

## Acceptance criteria

| AC | Level | GIVEN | WHEN | THEN |
|---|---|---|---|---|
| **AC-1** | unit | a stream with all eight groups, every entry kind, an empty parameter array and a name at the length escape | it is parsed | titles, names, parameters, strings and raw bytes read back as written; a 1-based collection's index 0 present and empty, its first written entry at index 1; no byte left over |
| **AC-2** | unit | five malformed streams — a parameter count past the end, an unsatisfiable collection count, a string past the end, an undefined length escape, one trailing byte | each is parsed | each refused with an error naming where; no table returned |
| **AC-3** | unit | a Units entry whose parameters are all −1; and one where every slot holds a distinct value | each becomes a definition | the first equals the defaults exactly; in the second every mapped field holds its own slot's value, health equals the health maximum, mana the mana maximum, and the experience value is slot 37's — which it can be only if 33 to 36 were consumed |
| **AC-4** | unit | entries with selector −1, 0, 3, 1 and 2 | each becomes a definition | −1 and 0 give the plain pair and no always-hit mark; 3 gives the pair and the mark; 1 and 2 are refused, naming the entry |
| **AC-5** | unit | placements over every arm: the npc flag; a definition id and the sentinel; primary keys 26, 27, `0x39`, `0x140`; a key with two faces; an empty-named entry before the match; two entries on one key | each is resolved | npc and unmatched reach nothing; the id one reaches the entry found downward; 26 and 27 take the Units arm and `0x39` the Humans arm; `0x140` reaches what `0x40` reaches; the empty name is skipped; the earlier duplicate wins |
| **AC-6** | unit | a definition at each value, over health maxima 0, 1, 99, 100, 65535 | the adjustment is applied | 1 and 3 give the truncated results above, 3 also moving to-hit and defence; 2 leaves the definition equal; health equals the maximum after 1 and 3; a fourth value is refused |
| **AC-7** | unit | one map built with no table, and with a table at each value | the worlds are compared | the no-table world's entities, byte form and digest are those the single-argument entry point gave before this story; the others carry each resolved placement's adjusted maximum and leave the rest at the provisional pair |
| **AC-8** | manual | a lawful install and one shipped map | the tool is run over both | the walk consumes the whole file, nothing left over; the eleven collections report their counts; the parameterised Units rows all carry the same number of parameters; every placement lands on a named arm with no unexplained residue |

## Derived properties

- **P-1 (negative-invariant)** — No field of a loaded definition holds −1: the only thing a sentinel
  cell can do is leave a default standing.
- **P-2 (invariant)** — Parsing and definition-building are functions of the input bytes alone: no
  clock, environment, file system or float takes part.
- **P-3 (completeness)** — Each of slots 0 to 37 has exactly one outcome: a named field, a named
  intermediate, or consumed-and-dropped. None has two, and none has none.
- **P-4 (idempotence)** — At difficulty 2 the adjustment is the identity, and at every value a pure
  function of its two arguments, so two copies of one definition adjusted alike stay equal.

## Constraints

| # | Constraint | Alternatives and trade-off |
|---|---|---|
| **C-1** | The table is a **third input to a world**, never a fact of the map. | **(A) read it inside the map loader** — archive IO in a tier that has none, and one map giving two worlds depending on what is installed. **(B, chosen) an argument**, absent meaning "no table". |
| **C-2** | Damage-switch arms 1 and 2 are **refused**, not modelled. | **(A) model all three destinations** — two field pairs nothing here reads, one with an open question about who applies it. **(B) drop the value** — a monster silently built with no damage. **(C, chosen) refuse the entry**: unexercised on both shipped roots, so the cost is measured, and loudly wrong beats quietly wrong. |
| **C-3** | The two multiplications are done in **integers**. | **(A) the image's two floating-point constants and a truncation** — the result crosses into hashed state, reopening a question this project spends effort keeping closed. **(B, chosen)** the integer forms, exact over the whole domain a stored maximum can occupy. |
| **C-4** | Only **Units** yields stats. | **(A) stream the Humans slots too** — that slot list is not published as a claim, and a hero's health maximum is *computed* from body and experience rather than read from a column, so the block would be an invention wearing a decode's clothes. **(B, chosen) resolve, carry no stats.** |

## Out of scope

- **Every stat but health reaching the simulation.** To-hit, defence, absorption, protections,
  resistances, damage, speed, reach and cadence sit on the definition and are read by nothing: there
  is no resolver, and unread fields in a hashed record buy a digest change for it.
- **Equipment** — the column, its grammar, and the weapon that would overwrite the cadence and
  extend reach; **the spellbook** and the spell, probability and treasure columns; **the seven
  collections this story only frames**, exposed and given no meaning; the **Buildings** footprint
  masks, made reachable and not taken; and **the prototype cache and information panel**.
- **Any front-end wiring**: the game and the viewer keep the entry point they call today, so nothing
  on screen changes. **Writing the table**, and the CSV path that regenerates it.

**Disclosed limitations**, accepted and owned: a placement reaching the Humans arm keeps the
provisional health, so a map's human placements stay uniform; the cadence and reach carried are the
template's, which the original moves once equipment is resolved; and the additions at difficulty 3
are done in 32 bits where the original stores 16 — a wrap no value the file can hold reaches.

## Gate check

FR-1 → AC-1 · FR-2 → AC-2 · FR-3 → AC-1, P-2 · FR-4 → AC-3, P-1, P-3 · FR-5 → AC-4, C-2 ·
FR-6 → AC-5 · FR-7 → AC-6, P-4, C-3, C-4 · FR-8 → AC-7, C-1 · FR-9 → AC-8. P-1 and P-3 are witnessed where AC-3 and AC-4 read, P-2 by parsing
one stream twice, P-4 at AC-6; the Level column says which criterion needs an install.
