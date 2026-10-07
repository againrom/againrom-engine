# Provenance — 0104, weapon reach

Every claim below was read from the research submodule at its pin through `tools/claim`, which
prints a row's amendments and its retraction state. All eight are `● active`.

## What the contract rests on

**`HERO-REACH-025` — High, with a Medium tail.** Reach is a byte at `actor+0x12c`, set to **1 by
construction** (`L00325`) and written by neither class streamer. The strike's gate is that byte
against `R0247` (`L00735` / `L02044` / `L02045 CMP ESI,EDX ; JGE`), which computes
`d = max(|Δx|, |Δy|)` in 1/256-cell units, subtracts `((aSize + tSize) << 7) − 0x100`
(`L04080`, `L04081`) and returns **1** for `d <= 0x180`, else `(d + 0x40) >> 8`
(`L04082`…`L04083`). The sizes are `vt+0x1c` = `R0256` = `actor+0x49`, the `tokenSize`
column. `Weapon::Equip` **adds** `range − 1` (`L04088` / `L04089` / `L04090` / `L04091`)
with `R0853` subtracting the same term; a `disp:12c` sweep finds exactly four writes — the
constructor, that pair, and `L00278`. Out of range, `R0042` issues a move-to-**actor**
order and sets `actor+0x54 = 1`. This is the whole of FR-7 through FR-11.

The **Medium** half is the re-path cadence: `R0043`'s body was not read. It does not reach
this contract — the approach already re-reads its victim's cell every turn (0075 FR-6) and this
story changes only where that walk stops.

**`UNIT-EQUIP-005` — High, Medium on the name grammar.** `R0184` loops the entry's **two**
trailing CStrings (`L12598 CMP …,0x2`), routes a name containing `"Shield"` to `R0973` and
everything else to `R0665`, then hands the item to `actor->vt+0x3c` = `R0976`, which
calls `item->vt+0x38(actor)`. Corpus, both roots: **30 classes carry no string, 26 carry exactly
one, 0 carry two, 0 contain `"Shield"`**, and 2 carry a `{castSpell=…}` suffix. Stripping `{…}`
and then the longest `Shapes` and `Materials` name prefixes leaves a `Weapons` row on **26 of 26**.
This is FR-1, FR-2 and FR-4, and it is why the shield arm is cut.

The **Medium** half is that the grammar is a corpus fit over 26 strings — `R0665`'s own
parse was not read. This build already carries the same fit for the human arm (0078 DD-3) and this
story reuses it rather than inventing a second one.

**`DAT-ACT-006` — High.** The Units spawn arm (`L02115 new(0x198)` → ctor `R0501`) reaches
the base actor constructor `R0185`, **which continues past its `1` defaults into
`R0184`**. This is the hinge: it puts `UNIT-EQUIP-005`'s equip loop on the construction path
of a placed unit, so reach is a **construction-time** value and the story needs no equipment
channel. It also states the `−1` skip law that `data.NewUnitDef` already implements.

**`DAT-SCHEMA-007` — High for the slot law.** `@.range` is a slot of the shared
Armors/Shields/Weapons title array, and `HERO-EQUIP-017`'s fill fixes the numbering of its
neighbours independently of the titles. Re-measured here through this tree's parser: `@.range` is
title index 12, **slot 11**. The row also records that the **Units** collection agrees on every
name, every equipment string and all 56×55 parameters across both shipped roots.

**`UNIT-ROOT-017` — High.** The `Units` span of `data.bin` hashes **equal** between the two roots
while the `Humans` span does not, so every Units figure this story uses is made on both roots by
construction rather than by a second measurement.

**`HERO-EQUIP-017` — High.** `Weapon::Equip` branches on `@.attackType`: melee (`< 10`) and ranged
(`0xb`/`0xc`) route the damage fields differently. It is cited for the branch alone; this story
takes the **range** and leaves the branch's damage routing where 0078 left it.

## Ours by choice

- Assigning the weapon's range where the original adds `range − 1` to a reach of 1. Equal over every
  shipped row (0 rows carry two equipment strings) and stated as a divergence, DD-2.
- Refusing a reach outside 1…255 in the constructor and the decoder. The original's field is a byte
  and its distance function floors at 1, so a reach of 0 is an actor that can never strike; nothing
  in the corpus produces one. DD-3.
- Writing the distance in the original's 1/256-cell arithmetic with the footprint term at its
  size-1 value, rather than as the `max(1, Chebyshev)` it collapses to. DD-1.

## Open, and cut to a named story

- **The footprint term.** `tokenSize` never reaches `pkg/sim`; 15 of 56 shipped rows carry 2 or 3,
  six of them among the 18 that gain reach. Owed by the story that gives the simulation a footprint.
  Nothing is missing from research.
- **Charge and relax from the weapon.** `HERO-CADENCE-023` (active, with an amendment) says
  `Weapon::Equip` assigns `actor+0x134`/`+0x135` from the weapon's `@.charge`/`@.relax` when the
  cell is not `−1`, moving the pair on **16 of 56** classes. A lane implementing this story holds
  the weapon that carries those two columns and must not apply them: it is a combat-pacing change
  across the whole roster and belongs to its own story. Cut, DD-5.
- **The group scorer's own reach test.** `AI-REACH-072` (High) says group order 3's scorer refuses
  a candidate whose distance term exceeds 1 outright. This build's group layer consults no reach at
  all, so widening reach here changes no group decision. Cut, DD-4.
- **The ranged equip arm's damage routing**, the `{castSpell=…}` payload, projectiles, and the
  facing precondition (already a named divergence, 0081 FR-5). None is a research gap.

## What was measured here, not cited

The 26 / 26 / 18 counts, the slot index, the attack types and the `tokenSize` distribution in
`analysis.md` were derived in this lane through `pkg/formats/databin` and `pkg/data.NewUnitDef`
against both installed roots. They agree with `UNIT-EQUIP-005`'s 26 and add the range values, which
no claim enumerates.
