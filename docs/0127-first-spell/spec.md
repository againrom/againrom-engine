# 0127 — first spell: specification

**One spell, chosen and fired by the player, that takes health off a named
enemy — table-driven end to end.** This contract is self-contained.

## Functional requirements

### The table

**FR-1** `pkg/data` gains a spell-table loader over the `Spells` collection of
the parsed placeable-definition table. One row becomes one `Spell` with named
fields, read by parameter slot:

| slot | column title | field |
|---|---|---|
| 1 | `Mana Cost` | `ManaCost` |
| 2 | `Sphere` | `School` |
| 4 | `Spell Target` | `TargetsUnit` — true iff the cell is exactly 1 |
| 6 | `Max Range` | `MaxRange` |
| 16 | `damageMin` | `DamageMin` |
| 17 | `damageMax` | `DamageMax` |
| 18 | `Defensive` | `Defensive` — true iff the cell is exactly 1 |

The collection is **one-based**: entry 0 is allocated and never written, and it
is skipped rather than loaded as a spell. A row whose parameter array does not
reach slot 18 is **refused by name**, never padded. Every negative cell is the
file's own empty-cell sentinel and lands as **0** on `MaxRange`, `DamageMin` and
`DamageMax`; a negative `ManaCost` is refused. `Name` and the `Effects` string
are carried verbatim; **nothing parses `Effects`** (SC-3).

**FR-2** A `Spell` carries `Damaging`, set by the loader and by nothing else:
true iff the row's damage pair is positive **and** the row's id is not one of the
two the game excludes from its damage arm by id — `Heal` and `Drain Life`, which
carry damage columns and are not damage. Those two ids are named once, in one
place, with the exclusion stated where they are named.

**FR-3** `pkg/sim` gains `SpellRule`, the simulation's own stdlib-only record of
one row: id, mana cost, school, maximum range, the damage pair, and the two flags
`TargetsUnit` and `Damaging`. It is a plain value. `School` was **carried and read
by nothing** here: it is the elemental kind the resistance arm and the skill term
both index, and `0135-skill-moves` is where the second of those two starts
reading it.

**FR-4** A world holds a spell table. It is supplied at construction, copied,
serialized with the world, and covered by the digest. A table with a repeated id,
with id 0, with a negative cost or damage, or longer than the form can express is
**refused**, so no world exists holding one. A world built with no table casts
nothing, and every world built before this story keeps its state.

### The spellbook

**FR-4a** A person's definition states which spells he knows. The `Humans` row
carries it in the column the file titles `knownSpells`, at parameter slot 25, as
a **bitmask subscripted by spell id**: bit *i* set means the row knows spell *i*.
The `-1` empty cell means **no spells stated** and lands as an empty book, never
as every spell. The loader carries the mask and interprets no further.

**FR-4b** A simulation entity carries that mask, and **knowing a spell is a set
bit**. It is state: it serializes with the world and is covered by the digest.
Nothing in this build sets or clears a bit after construction — learning is out
of scope (SC-5) — so an entity knows exactly what its definition stated.

The game's own book is a sparse array that can hold any id. A 32-bit mask holds
ids 0..31, the id space is 1..28, and every shipped mask lies inside it — so the
two agree on every set that exists, and DD-7 is where they would not.

### The cast

**FR-5** A new command kind, `KindCast`, orders the entity it names to cast at
another: the victim's id rides in `X` (the same 32 bits, as the attack order
already does) and the spell id in `Y`. It resolves **wholly inside the command
phase**, on the tick it arrives, and leaves no state behind: nothing is stored on
either entity, so the same command applied twice costs the caster twice.

**FR-6** The economy is the game's, and it is the whole of it:

1. the caster must be held by the world and **alive**;
2. the caster must be a **mage** — DD-3 — and must **know** the spell (FR-4b);
3. the spell id must name a row of the world's table;
4. that row must be `Damaging` and `TargetsUnit`;
5. the victim must be held by the world, alive, and not the caster;
6. the victim must be within the row's `MaxRange`, in whole cells, by the same
   Chebyshev distance the rest of this package measures reach with;
7. if the row's `ManaCost` **exceeds** the caster's current mana the cast is
   **refused and nothing else happens** — no mana is spent, no damage is dealt,
   nothing is rolled;
8. otherwise the cost is subtracted **once** from the caster's current mana.

No skill, statistic, level or power term enters the cost, and a cost equal to the
caster's mana is affordable. A cast that fails any of 1-6 is a no-op, exactly as
a command naming an absent entity already is.

**FR-7** The damage, applied immediately after the cost:

- the caster's **power** is `clamp(skill + Mind - 30, 0, 100)`, and the skill
  term is the caster's own level in the school (`0135-skill-moves`);
- the damage factor is `f = power/30 + 1`, and the pair the roll reads is
  `base = trunc(DamageMin x f)` and `spread = trunc(DamageMax x f) - base`,
  so the second column is a **spread and not a maximum**;
- the amount is `base + U[0, spread]`, drawn from the world's own generator;
- the victim's health falls by that amount, and by nothing else. **A spell never
  rolls to hit**, and nothing flat reduces it: absorption reduces a physical
  component this attack does not have.

A `spread` that computes negative is taken as 0. Health is not clamped, on this
package's existing rule.

**FR-8** The integer form of FR-7's arithmetic is exact and stated:

```
base   = DamageMin * (power + 30) / 30      (Go integer division, truncating)
spread = DamageMax * (power + 30) / 30 - base
```

Its divergence from the float reference `trunc(x * (power/30 + 1))` is
**measured over the whole reachable domain and disclosed as a number**, in a test
outside the determinism wall. It is not asserted to be zero.

### Reaching the player

**FR-9** The mission world loads the table from the same parsed definition table
it already loads units, humans and items from, and hands it to the world it
builds. A build with no definition table still builds a world.

**FR-10** The player casts, and the shape of it is the owner's ruling as author:
**a click selects any spell from the book, and the cursor casts it at a target.**

- The selected unit's **book** is shown: its known spells, by name, from the
  world's own table — a unit that knows nothing shows nothing.
- **Clicking one selects it.** A second click on the same one, or selecting
  another unit, clears the selection.
- With a spell selected, the click that targets an enemy **casts it at that
  enemy** instead of ordering an attack, and the selection clears.
- With none selected, that click orders the attack it always did.

**No spell id is written into the code**: the book comes from the entity's mask
and the names from the loaded table, so a table with different rows shows and
casts different spells. Everything about this beyond the owner's one sentence —
where the book is drawn, what it looks like, how a selection is cancelled — is
ours, and DD-6 says which is which.

## Acceptance criteria

**AC-1** Loading the shipped `Spells` collection yields 28 spells, ids 1..28,
with `Fire Arrow` at id 1 costing 3 mana, school 1, range 7, damage 4..8,
targeting a unit.

**AC-2** At power 0 the pair `base .. base+spread` reproduces each row's own
`damageMin .. damageMax` columns exactly, on every row carrying a damage pair;
and the rival reading in which the second byte is not a difference reproduces
**none** of them.

**AC-3** A caster with mana at least the cost loses exactly the cost, once, and
the victim's health falls by an amount inside `[base, base+spread]`.

**AC-4** A caster with mana **one below** the cost loses no mana and the victim
loses no health — the world after is byte-identical to the world before.

**AC-5** A caster with no mana pool casts nothing, whatever the cost; and a
caster with mana who does not know the spell casts nothing and pays nothing.

**AC-5a** A `knownSpells` cell of `-1` loads as an empty book. A stated mask
loads with exactly the bits the cell sets, and `266306` — the value the shipped
`ManMage_Staff` row carries — is the four spells 1, 6, 12 and 18.

**AC-6** A victim one cell beyond `MaxRange` takes nothing and costs nothing.

**AC-7** A spell id naming no row, a row that is not `Damaging`, and a row that
does not target a unit are each a no-op.

**AC-8** A world carrying a table round-trips through the byte form unchanged,
and two worlds differing only in their tables have different digests.

**AC-9** A byte form declaring the previous version is refused.

**AC-10** With a spell selected, the enemy click produces a cast command naming
that spell and not an attack command; with none selected, it produces the attack
command it always did. The book offered for a unit is exactly the spells its mask
names that the table holds — no more, and in the table's order.

## Properties

**P-1** The cast reads and writes only the two entities it names and the world's
generator. It touches no route, no group, no script register and no third entity.

**P-2** Two worlds stepped from the same seed with the same commands stay
byte-identical. Nothing on this path reads a clock, the environment or a
package-global generator, and no float value exists inside `pkg/sim`.

**P-3** A refused cast is indistinguishable from no command at all, including in
the generator's state: **nothing is rolled before the refusal**.

**P-4** The table is data. Replacing a row's numbers changes what the spell does
and changes no code; no spell id is compared to a literal anywhere in `pkg/sim`.

## Design decisions

**DD-1** The cast resolves immediately rather than being queued. The game queues
it with a delivery delay for two of its twenty-eight spells; a queue is state and
a spell that lands this tick is what makes the mechanism visible. SC-1.

**DD-2** The victim rides in `X` and the spell in `Y`, so the command widens by
nothing. A third argument field would sit unused in every other kind.

**DD-3 (DECODED)** **A mage is an entity with a mana pool**: `MaxMana > 0`. The
cast is gated on a class bit, and that bit is *set from a nonzero mana maximum*
at a named instruction in the routine that allocates the book. So the predicate
is derived rather than chosen, and it is one expression in one named place. What
remains open is only how an actor first acquires a book, which nothing here
needs. The shipped table agrees: every row stating a spell mask states a mana
column.

**DD-4** The damage arithmetic is integerised inside the simulation rather than
computed outside it. `pkg/sim` admits no float, and computing the pair in the
loader would fix it at load time and make the caster's own power unreadable.
FR-8 is the cost of that choice and it is paid in the open.

**DD-6** **The affordance is the owner's ruling, and only the ruling.** Nothing
decoded says what makes a player cast; the owner, as author, ruled that a click
selects a spell from the book and the cursor casts it at a target, and FR-10 is
that sentence. Everything else — that the book is a strip of named buttons rather
than a grid or a page, that a second click cancels, that the cast reuses the
enemy-targeting click already wired to the world — is **ours**, chosen for being
the least that satisfies the ruling, and it lives in one place.

**DD-7** The book is a 32-bit mask, not the original's sparse array. It is
lossless for every shipped set and narrower for one no shipped file contains.

## Out of scope, and each one is a cut

**SC-1** Delivery: the projectile delay and the effect speed, so no spell flies.

**SC-2** Every elemental resistance. No protection value reaches a simulation
entity in this build; it is a named seam, not a silent zero.

**SC-3** The `Effects` string and the sixteen non-damage arms — every buff,
curse, protection, wall, cloud, heal, teleport and singular spell. The column is
carried and parsed by nothing.

**SC-4** Area shape and distribution, so no spell hits more than one unit.

**SC-5** **Learning.** A spell is known because the definition said so; nothing
in this build learns, forgets, buys or is taught one. The game's only teacher is
the `teachSpell` effect, that verb occurs exactly once in the whole shipped
table — in the effect vocabulary itself — and **no shipped item uses it**: the
five books and five scrolls carry a price and a weight and name no spell. So the
mechanism is decoded and exercised by nothing a player can reach, and this story
neither builds it nor invents what a book would teach.

**SC-5a** Training the school on a cast, scroll and book prices, monster casting,
the active-spell bitmask.

**SC-6** The item cast: a weapon firing a spell on a landed melee hit. It is
gated on the wielder being a fighter and no shipped person carrying one is a
fighter, so it fires for nobody a player can see; and its parser is not decoded.

**SC-7** Decoding how an actor first acquires a book, which MAGIC-BOOK-002 records
as an Unknown with its instrument stated. Nothing here needs it: an entity's
book comes from its own definition.
