# Analysis — 0105, the equipment channel

## What 0085 left standing

The owner has twice said there is **no weapon in his hands**. `0085` installed the game's law from a
**body name** onward — the name-to-class chain, the suffix and the two substitutions, the art
directory, the sheet address, the draw that prefers that sheet. What it could not install was the
**first step**, and its FR-5 said so: the body name was AUTHORED, because "the ordered list the
original's equipment slot indexes is not decoded and this tree has no such slot". Both halves of
that sentence are now false.

## The finding that decides the whole cut

`HERO-APPEAR-053` establishes that **nothing is composited over the world sprite**. The
`graphics\equipment` tree the original loads per equipped item is 928 sheets of exactly one
160x240 frame; the sixteen hero body sheets are 129..216 frames of 24x40..40x48. A world-space
overlay would have to carry the body's frame count at the body's scale and none of the 928 does.
`SPR256-EQUIP-042` measures the same tree from the archive side and calls it a portrait tree.

So **in the world a held weapon is visible because the whole body sheet changes**, and the sheet is
chosen by a name. There is no overlay to draw, no layer to order, no compositor to build. Everything
the corpus holds about *what is drawn and in what order* — the draw order, the two-handedness layer
predicate, the stencil slot byte — belongs to the **info window**, a screen this tree does not have.
That cut rests on evidence, not on budget.

What survives into the world is three links, each of them High:

1. an equipped item's appearance word carries a five-bit field D, and D is **the item's own
   definition row index** in its collection (`ITEM-APPEAR-023`);
2. the routine that names a character's body takes D from **equipment slot 1** and indexes a
   shipped ordered list with `D - 1` (`HERO-APPEAR-042`, `HERO-APPEAR-047`);
3. that list is the shipped file `text/heropicture.txt` (`HERO-APPEAR-052`).

Link 3 hands `0085`'s law its missing input, and nothing else has to move.

## The measurement this story took

`text/heropicture.txt` is 248 bytes in `main.res` and **byte-identical on both roots**. Read with
this repo's own archive reader it is **26 CRLF lines**, one of which — index 22 — is **blank**.

Lined up against the shipped `Weapons` collection of `world.res` under "D is the Weapons row index,
the list is indexed by `D - 1`", the two agree on **22 of 22** rows below the blank: `BareHands` to
`unarmed`, the four one-handed swords to `swordsman`, the four clubs and maces to `clubman`, the two
staves to `mage_st`, pike, halberd and lance to `pikeman`, the two bows to `archer`, `Crossbow` to
`xbowman`, and so on. The three `2h` names fall on exactly the three rows whose two-handedness
column is set.

Neither side was fitted to the other, and the alignment did not derive the rule: the rule comes
from the chain above and this is a check on it.

**The blank line at index 22 aligns with `Weapons` row 23, which is named `rem`**: a removed row's
placeholder. That is a twenty-third agreement under "the reader keeps blank lines" and a
disagreement under "the reader drops them". `HERO-APPEAR-052` publishes the list as 25 non-blank
lines and enumerates 22 as `Sonic Beam`, 23 as `Flame Thrower`, 24 as `swordsman`; the `rem`
alignment implies 23, 24 and 25 instead. The two readings agree completely for `D` up to 22 and can
differ only above it. This tree keeps the shipped blank line, which is the reading the corpus
supports, and discloses the divergence. **Nothing this story runs can produce a D of 23 or more**:
all ten weapons character generation hands out resolve to rows 3, 6, 9, 15, 18 and 20.

## The premise about the missing garment names — answered

The premise handed to this story was that the English names for equipment slots 2-6 and 8-11 are
missing, and that this might make a shipped hero undrawable. **It does not, and the gap is code
rather than data.**

The *label* on a slot is graded no better than Medium anywhere: `HERO-APPEAR-042` grades "slot 0 is
the weapon, slot 1 the shield" Medium and says the labels come from sheet names, "which are not
evidence". But the slot's *function* is High and needs no label. `HERO-APPEAR-050` fixes field B as
the slot **number** against a five-offset corpus discrimination, `HERO-APPEAR-047` fixes the wire
slot as the equipment slot minus one from named instructions, and `HERO-FIGURE-058` reproduces that
increment from a second instrument. A body name is produced from **slot 1 by its number**; an
English word for slot 5 is consulted only by the per-slot **info-window figure**, which this story
cuts.

## The reach hook, and a correction to how it was described

`0104` located this story's debt at `pkg/mapload/start.go`. The location is right; the mechanism
named for it is not. A party member's combat is built by `Hero.Derive`, and `Hero.Derive` does
**not** set `Reach` at all — it is `HumanDef.Combat` that carries the constructor's floor of 1, and
a party member never goes through `HumanDef`. `start.go` then mints the entity without writing
`Reach`, so the field reaches the world constructor as **zero** and the 1 comes from that
constructor's own repair, which folds a zero reach to 1. The defect is a step worse than described:
his reach is not a floor taken in place of a range, it is a normalisation of a field nobody wrote.

It is invisible today for the reason given: the blade arm's `Iron Short Sword` is `Weapons` row 3,
whose range column is empty and therefore 1 — verified against the shipped table on both roots
rather than taken on report. Four of the other nine start weapons are also range 1; the bow is not.

## What is cut, and to what

The full in/out list is `spec.md`'s Scope, where a contract's boundary belongs. Three cuts are made
on evidence rather than on budget and their evidence is above: the figure compositor, which draws
the info window and not the world; the voice bank, for which this tree has no audio tier at all —
no package, no consumer, nothing to feed; and the garment names, which nothing on the world path
consults. The rest — containers, carried items, picking up, dropping, the packed-word item code —
each need a tier this tree has not got, and `0103` already owes every one. The placed population is
untouched: `0085` P-4 holds, and a map's own humans are a second story.

## Byte form version 24

**Allocated to this story and not used.** No simulation field is added, removed or retyped:
`Entity.Reach` has existed since version 23 and this story only writes a value into it, so the form
literal stays at 23 and a world assembled from identical entities keeps its digest.

## Nothing here is owed to research

Every fact is a published claim row read at the pin, or a measurement taken here from the two lawful
roots. One question is raised and not opened — whether the original's list reader keeps or drops a
blank line — and it cannot affect anything this story ships.
