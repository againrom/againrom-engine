# Spec — a hero's stats are recomputed, whole and in order

## Terms

**The recompute** — the one routine that turns a character's four statistics, his six skill levels
and what he is wearing into every number derived from them. In the original it is a single virtual
method; in this tree it has been three functions stating four of its terms.

**Loadout** — what a character is wearing, as the recompute reads it: the weapon in his hand, and the
accumulated **modifier block** every other equipped item folds into.

**Modifier block** — the additive contribution equipment makes: a to-hit term, a damage base and a
damage spread term, a defence and an absorption term, five elemental protection terms and five
damage-kind resistance terms. It is added, never multiplied.

**Derived set** — everything the recompute produces: the four capped statistics, the six per-slot
experience costs and their sum, the health and mana maxima, the eight combat numbers and reach, the
five elemental protections, the five damage-kind resistances, the step rate and the sight radius.

**The bare pair** — a character's cadence with nothing in his hands: charge 8, relax 4.

## Problem

`Hero.Derive` is a partial statement of the recompute. It takes a bare weapon where the routine takes
the whole character; it produces neither pool, neither resistance family, no experience and no clamp;
its order is not the routine's; and it runs once at load and never again. `Hero.Speed` and
`Hero.Sight` are two further terms of the same routine standing outside it, each capping its own
inputs.

So a hero's numbers are stale the moment anything about him changes, three functions each hold their
own copy of step 0, and the information window can state eight numbers about a character out of a set
of more than thirty. The owner asked for stats recomputed correctly from statistics, experience and
equipment, and for the full derived set to reach the window.

## Scope

One recompute, complete and in the routine's order, with the full derived set reaching the
information window and a path by which a later recompute reaches a live entity. The interaction that
changes equipment is not here; nor is any change to what a blow reads or to how a world is
serialized.

## The contract

### One statement of the graph

**FR-1** There is **exactly one** implementation of the derived-stat graph in the tree. Every other
spelling of any part of it — the combat block, the step rate, the sight radius — is a thin accessor
that calls the one implementation and computes nothing itself.

**FR-2** The recompute takes a character, a **profile** and a **loadout**, and returns the whole
derived set as one value. The profile carries what the four statistics and six skills do not: whether
the class flag is set, and whether the health and mana columns are nonzero.

**FR-3** It reads nothing but its arguments: no clock, no generator, no global, no file. The same
three arguments yield the same derived set in every process.

### The order

**FR-4** The recompute runs in the original's order, and the four points where that order is
load-bearing are honoured:

1. the **statistic caps run first**, so every later term reads the capped value;
2. the **two pools are computed before the skills are restored**, so a skill change reaches them
   only through the next recompute;
3. the **protection block is cleared** after defence's inputs are read and before defence is written,
   so defence, absorption, the five protections and the five damage-kind resistances all start from
   zero on every recompute rather than accumulating across two of them;
4. the **equipment fold runs between the bare values and the clamps**, so the clamps bound equipment
   and not the bare statistics.

**FR-5** Step 0 is `stat = min(stat, 50 + modifier)`, destructive, with 100 the ceiling in principle
and 50 what an untouched statistic is capped at. It applies to all four statistics.

### The terms

**FR-6** A skill **level** and its **experience** are two spellings of one value, joined by
`S(L) = ftol((1.1^L - 1) x 1000)` and by `S`'s inverse, and both directions are live in the original.
The recompute writes one of them: the six per-slot experience values are produced from the six
levels, and their sum is the character's total. The per-slot values are carried, because they are the
primary storage and the total is the derived one. The **inverse** — experience back to a level, which
the loss path uses when it takes a tenth off a slot and writes the resulting level back — is a
**named seam**: its arithmetic is not read, and so it is not written here. Which slot a gain is
credited to is likewise unread.

**FR-6a** The **skill restore**. Slots 1 to 5 are the incoming level plus the loadout's own per-slot
bonus, clamped to `[0, 100]`. **Slot 0 is in neither the restore nor the clamp** and passes through
untouched, while remaining inside the experience sum, which runs over all six. A level is an integer;
no fractional level is representable. The restore runs **after** the two pools and **before** the
damage and to-hit terms that read a level, so a bonus reaches those terms at once and reaches a pool
only through the next recompute.

**FR-7** The health maximum is three steps: `h = Body x (flag ? 2 : 1)`, **skipped entirely when the
health column is zero**; then `h = ftol(h + log(1.1, xp/5000 + 1) x (flag ? 2 : 1))`; then
`h = ftol(h x (1.1^Body / 100 + 1))`. Every intermediate is truncated toward zero before the next
step reads it, so the truncations compound.

**FR-8** The mana maximum is the same three steps on Spirit with the class multiplier inverted:
`m = Spirit x 2` — **not class-conditional** — gated on the mana column being nonzero; then
`m = ftol(m + log(1.1, xp/5000 + 1) x (flag ? 1 : 2))`; then `m = ftol(m x (1.1^Spirit / 100 + 1))`.

**FR-9** The damage pair, to-hit, the skill terms and the cadence are unchanged in value from what
this tree already derives: the spread is `ftol(1.1^Body / 20)` and the base is copied from it; to-hit
is `ftol((1.1^Body + 1.1^Reaction) / 5)`; the weapon's own kind names the active skill slot and, when
that slot is positive, adds three times its level to to-hit and a fifth of its level to the **base
alone**; the weapon's cadence is assigned per half and guarded on the empty cell; reach is 1 bare and
the weapon's range otherwise.

**FR-10** Defence is `Reaction / 3` and absorption is 0. **Absorption has no source but armour**: no
weapon term writes it, so a character carrying only a weapon absorbs nothing, and that is the value
rather than a gap.

**FR-11** The five elemental protections all start from **Spirit halved**, truncating toward zero.
The five damage-kind resistances start from zero and are **never re-derived** for a character.

**FR-12** The step rate and the sight radius are terms of this recompute and are produced by it:
`Reaction` below 12 and `Reaction/5 + 12` at 12 and above, and `(Mind + Reaction)/25 + 4`.

### Equipment

**FR-13** The equipment fold is **addition only** — a to-hit term, a damage base and spread term, a
defence and an absorption term, five protection terms and five resistance terms — plus the weapon's
own assignments of cadence, reach and active-skill slot. There is **no multiply** anywhere on the
path from an item to a damage number.

**FR-14** The **weapon arm is complete**: a resolved weapon's own numbers reach the fold with nothing
missing. The **armour and shield arm is a seam**: the modifier block accepts their contribution and
carries zero for it, because which shipped column fills it is not decoded. No column is invented for
it and no value is fitted to any remembered figure.

### The clamps

**FR-15** After the fold, each of the five elemental protections is clamped
`clamp(min(Spirit/2 + 70, p), 0, 100)`.

**FR-16** The last step of the recompute clamps a live health and a live mana to their maxima. A zero
mana column therefore leaves a character at zero mana through the maximum, not through a second rule.

### Reaching the player

**FR-17** The information window states the derived set it did not before: the character's
experience, his five elemental protections and his five damage-kind resistances. The panel's layout,
geometry, colours and existing rows are not redesigned.

*Folded from hotfix `ea870ef` — see `docs/hotfix/ARCHIVE.md#ea870ef`.* The derived set the window
states includes the name of the weapon in hand, written at every equipment change and not once at
the open.

**FR-18** A recompute's result can be written onto a **live entity**, so that a later change of
equipment can be applied to a unit already in a world rather than only at load.

## Acceptance

**AC-1** Removing the one implementation breaks every accessor: no second spelling of any derived
number survives its deletion.

**AC-2** A recompute of an unequipped character reads **the same number on all five elemental
protections**, and that number is his capped Spirit halved.

**AC-3** Recomputing twice in a row over the same inputs yields the same derived set, and the cleared
block does not accumulate across two recomputes.

**AC-4** The combat block the recompute produces is **identical, field for field**, to what
`Hero.Derive` produced before this story, for the bare case, the armed case, an untrained character,
a character above the cap and the zero value.

**AC-5** A character whose skill slots are all zero has experience 0; one skill at level 10 accounts
for 1593 and nothing else contributes. Slot 0 at a positive level contributes to the sum.

**AC-13** A per-slot bonus raises a level, and the raised level moves to-hit and the damage base;
the same bonus on slot 0 moves nothing; a bonus that carries a level past 100 is bounded there, and
one that carries it below 0 is bounded at 0.

**AC-6** The health maximum with a zero health column omits the first arm entirely, and is not the
same number as the one with the column set.

**AC-7** A live health above the maximum clamps to it; a character with a zero mana column ends at
zero mana.

**AC-8** The five damage-kind resistances of a character carrying only a weapon are all zero.

**AC-9** A modifier block carrying a defence term moves defence and no damage number; one carrying a
damage term moves the damage pair and not defence.

**AC-10** A protection term large enough to exceed the clamp is bounded to the smaller of
`Spirit/2 + 70` and 100, and a negative one is bounded at 0.

**AC-11** The panel states the three new rows for a party member and states none of them for a unit
the map placed, on the same "was told" rule the existing character rows use.

**AC-12** Writing a recompute's result onto a live entity changes exactly the eight numbers and the
reach, and changes them on that entity only.

## Derived properties

**P-1** No simulation field, byte form or digest changes. The protections, resistances and experience
are loader values: nothing in the simulation reads them, so none is hashed and **no byte-form version
is spent**.

**P-2** The floating-point arithmetic stays in `pkg/data`, which is outside the determinism wall, and
the simulation continues to receive integers only.

**P-3** The party's health pair is unchanged. A generated character has no shipped row, so his health
and mana columns and his class flag are inputs nobody can state; the standing divergence that a party
member spawns at a fixed health survives this story rather than being papered over with an invented
column.

**P-4** Truncation is toward zero everywhere, in every intermediate, matching the original's rounding
control.

**P-5** The zero value of a character remains a legal input and derives by the same arithmetic.

## I/O examples

A character at Body 25, Reaction 25, Mind 25, Spirit 25 with one skill at 10, holding nothing:
experience 1593; protections 12 on all five; resistances 0 on all five; defence 8; absorption 0;
damage 0-0; step rate 17; sight 6.

The same character with the health column set and the class flag clear: `h = 25`, then
`h = ftol(25 + log(1.1, 1593/5000 + 1))`, then `h = ftol(h x (1.1^25/100 + 1))`.

## Out of scope

The equipping interaction and its user interface. Any change to what a blow reads, to the party's
health pair, or to how a world is serialized. Compacting the information window for ordinary units.
The experience **gain** and **loss** paths, and with them `S`'s inverse — a different routine, whose
arithmetic is unread and which no term of this one calls. Deciding which archetype sets the class
flag.

## Verification mapping

FR-1..FR-3 -> AC-1, AC-3, AC-4. FR-4, FR-5 -> AC-3, AC-4. FR-6 -> AC-5. FR-6a -> AC-13.
FR-7, FR-8 -> AC-6.
FR-9, FR-10 -> AC-4, AC-9. FR-11 -> AC-2, AC-8. FR-12 -> AC-4. FR-13, FR-14 -> AC-9. FR-15 -> AC-10.
FR-16 -> AC-7. FR-17 -> AC-11. FR-18 -> AC-12.

## Gate check

`go build ./...`, `go vet ./...`, `gofmt -l` clean over tracked Go files, `go test -trimpath -count=1
./...`, `scripts/check-no-game-assets.sh`, `scripts/check-doc-budget.sh`, `scripts/check-sdd-audit.sh`.
