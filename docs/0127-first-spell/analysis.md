# 0127 — first spell: analysis

What we did not know, and what we looked at before writing the contract.

## The subsystem is decoded and has no code

Nineteen `MAGIC-*` claims are live, none retracted whole, and this tree carries
no magic at all. So the risk here is not "what does the game do" — it is **how
much of a decoded subsystem to build at once**. The owner's ruling settles that:
breadth ahead of fidelity, a playable mechanism ahead of a complete one. This
story therefore builds ONE spell path end to end and cuts sixteen of the
seventeen arms `MAGIC-ARM-014` enumerates, on record in `spec.md`.

## The castSpell weapon was believed to be the cheapest first spell — it is not

`MAGIC-ITEM-007` describes a weapon that fires a spell on a landed melee hit,
exempt from the mana gate, and needing no player affordance at all. That reads
like the shortest path to a first spell, and it was carried as such.

It was measured instead of assumed, over the shipped `Humans` collection on
**both** lawful roots, and the two roots agree exactly:

| | en | ru |
|---|---|---|
| Humans rows (1-based, entry 0 unwritten) | 215 | 215 |
| rows carrying a brace-form modifier | 78 | 78 |
| rows carrying `{castSpell=...}` | **46** | **46** |
| of those, rows with no mana column | **0** | **0** |

Every one of the 46 carries a staff and a mana pool. `MAGIC-ITEM-007` gates the
melee trigger on the wielder being a **fighter** — `HERO-CLASS-020`'s exact
negation of the mage test — so **the item-cast path never fires for a shipped
person**. Two `Units` rows do carry one (`Catapult`, `Ballista`, both
`{castSpell=Fire_Ball:...}`); nothing in `Data.bin` states their class bit, and
they are siege engines whose blow is not obviously the melee strike the claim
names. Building that path first would have handed the owner a mechanism he
cannot see, and would still have owed the `{castSpell=...:n}` parser that
`MAGIC-ITEM-007` records as an **Unknown**. It is out of scope here.

## `MAGIC-DMG-005`'s corpus test reproduces

Re-executed against both installs at power 0 (`f = power/30 + 1 = 1`): the
rolled range `base ... base+spread` reproduces the file's own
`damageMin ... damageMax` on **9 of 9** rows carrying a damage pair, and the
rival reading in which the second byte is not a difference scores **0 of 9**.
Numbers in `verification.md`. That is the acceptance test for the damage arm and
it was run before the arm was written.

## What the entity does not carry

`0119` put `Mana`/`MaxMana` on `sim.Entity` and `0125` put `Mind` and the six
`SkillXP` slots there. It carries **no skill level**, so `MAGIC-POWER-004`'s
`skill[school] + Mind - 30` can only be evaluated with the skill term at zero,
and **no elemental protection**, so `MAGIC-RESIST-006`'s reduction has nothing
to read. Both are named seams in `spec.md`, not silent zeros.

## The class bit

`0125` reported that no class bit exists on `UnitDef`, `HumanDef` or `Hero`, and
`data.Profile.Fighter`'s own doc says nothing decides it. Both `MAGIC-CAST-003`
and `MAGIC-ITEM-007` gate on exactly that predicate. Measured on the shipped
table: the "no mana" cell is the **-1 empty-cell sentinel**, not 0, and
`HumanDef` normalises it, so `data.Profile`'s `ManaMax == 0` reading is sound on
the loader's output — `PC_Danath` and `PC_Naira` carry no mana, `PC_Fergard` and
`PC_Reniesta` carry 70. The class axis and the mana column coincide on every
shipped row that matters. This story therefore authors the mage predicate as
**a caster is an entity with a mana pool** and names it in one place, rather
than inventing a bit no file states.

## The affordance

Nothing decoded says what makes a player cast — no key, no click, no spellbook
interaction. That is an **AUTHORED** verdict, and `spec.md` says so in its own
words rather than leaving it to be inferred.

## The spellbook is in the shipped data, and it is a bitmask (measured after the owner spoke)

The owner's account — *"a mage has a spellbook; he learns spells from one-spell
books, permanently; a click picks any spell and the cursor casts it at a
target"* — sent us back to the file. `MAGIC-BOOK-002` (High) has the runtime
book: a sparse array at `actor+0x140` subscripted by spell id, "knowing" a spell
being a non-null slot, learning only through the `teachSpell` effect arm, and the
allocator setting `actor+0x4c` **bit 2 — the mage bit — when `manaMax != 0`**.

Two things follow, one of which was not expected.

**The mage predicate is decoded, not authored.** The class bit both
`MAGIC-CAST-003` and `MAGIC-ITEM-007` gate on is *derived from having a mana
pool*, at a named instruction. `0119` landed that pool on the entity. So this
story reads it rather than inventing it.

**The `Humans` collection states each row's known spells.** Column title 26 is
the file's own `knownSpells`, at parameter slot 25, and the bit-per-spell-id
reading discriminates hard against its rivals — measured on **both** roots,
identically:

| | en | ru |
|---|---|---|
| rows stating a mask (positive cell) | **48** | **48** |
| rows with the `-1` empty cell | 160 | 160 |
| stated masks setting bit 0 — the id the resolver refuses | **0** | **0** |
| stated masks setting any bit 29..31 — ids no row exists for | **0** | **0** |
| stated masks on a row with no mana column | **0** | **0** |

`ManMage_Staff` decodes to `Fire Arrow, Heal, Light, Shield` — the cheapest spell
of four of the five schools. Under any rival reading (a count, a single id, an
unrelated integer) bit 0 and the high bits would be set at chance rate across 48
values; they are set nowhere. A placed mage therefore arrives knowing his own
row's spells and no story has to author that.

**Where the owner's account and the shipped data part company.** The string
`teachSpell` occurs **exactly once** in the whole table — in the `Magic`
collection, which is the fifty-name effect-verb vocabulary the arms search — and
**no shipped item uses it**. `MagicItems` holds five `Scroll <School>` and five
`Book <School>` rows whose only columns are a price and a weight; **no row names
a spell**. So learning-from-a-book is real, decoded, and exercised by nothing a
player can buy. It is out of scope, and nothing about it is authored here.

**Two counts corrected against what this lane was handed.** `castSpell` occurs
**46** times on `Humans` rows, not 48 — 48 is the number of rows stating a
`knownSpells` mask. And `knownSpells` at `-1` is `databin`'s empty-cell
sentinel, not a full mask; read as unsigned it looks like all 32 spells known,
which would have made every peasant an archmage.
