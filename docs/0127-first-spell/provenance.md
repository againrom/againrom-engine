# 0127 — first spell: provenance

Claim ids from the `research/` submodule at this story's pin. Cite claims, never
experiments.

## Decoded, and built on

| Claim | Confidence | What this story takes |
|---|---|---|
| `MAGIC-SPELL-001` | High | A spell is a handle onto a `Data.bin` **Spells** row; ids run 1..28 and id 0 is refused. `getParam(row, 1)` = `Mana Cost`, `getParam(row, 6)` = `Max Range`, `getParam(row, 0x12)` = `Defensive`. FR-1, FR-4. |
| `MAGIC-CAST-003` | High | The whole economy: gated on the mage predicate; **refused with nothing else happening** when `cost > current mana`; otherwise the flat column subtracted once. No skill, stat, level or power term multiplies the cost. FR-5. |
| `MAGIC-POWER-004` | High (formula), **partially retracted** | `power = clamp(skill[school] + Mind - 30, 0, 100)`, and the damage factor `f = power/30 + 1`. The retracted clause was the "exactly four expressions" enumeration; the formula and the damage factor are not what was withdrawn, and this story consumes only those. FR-6. |
| `MAGIC-DMG-005` | High | `spell+0x0e = ftol(damageMin x f)`, `spell+0x0f = ftol(damageMax x f) - spell+0x0e` — **the second byte is a spread, not a maximum** — and the arm is skipped when the pair sums to 0. FR-6, FR-7. |
| `MAGIC-RESIST-006` | High | The roll is `base + U[0, spread]`; **a spell never rolls to hit**; absorption reduces the physical component only, so nothing flat reduces a spell. FR-7. |
| `MAGIC-ARM-014` | High | The damage half runs first and the per-spell switch is reached only when the spell has no damage — so the seven pure damage spells execute no arm at all, and `heal` (6) and `drain_life` (11) carry damage columns and are **not** damage. FR-2. |
| `MAGIC-TARGET-017` | High | `Spell Target == 1` queues the cast against the **target unit**; anything else against a point. FR-2, FR-4. |
| `HERO-CLASS-020` | High | `R0442 != 0` is the mage and `R0859 != 0` is the fighter — the two are exact negations of one bit. The predicate this story authors stands in for that bit; DD-3. |
| `MAGIC-ITEM-007` | High / **Unknown** on the authoring hop | The melee trigger requires the wielder to be a **fighter**. Measured out of scope (analysis); the `{castSpell=...:n}` parser stays unwritten. |

## Read, weighed, and deliberately not built

| Claim | Why not |
|---|---|
| `MAGIC-EFFECT-015` | The `Effects` string parser and the eleven effect-building arms. Sixteen of seventeen arms are cut; the column is carried verbatim and parsed by nothing (SC-3). |
| `MAGIC-TRAIN-018` | Casting trains the spell's own school by `round(manaCost/2)`. `0125` landed the six `SkillXP` slots this would credit, so it is one call away, and it is out of scope by name. |
| `MAGIC-SHAPE-008`, `MAGIC-EFFMODE-009`, `MAGIC-ATTACH-016` | Area shape, duration modes and the active-spell bitmask. All belong to the effect half this story does not build. |
| `MAGIC-BOOK-002`, `MAGIC-SING-019`, `MAGIC-AI-012`, `MAGIC-MIND-010`, `MAGIC-SPIRIT-011`, `MAGIC-CEIL-013` | The spellbook, the eight singular spells, monster casting, and the two statistics' wider reach. |
| `MAGIC-RESIST-006`'s resistance arm | `sim.Entity` carries no elemental protection field. Building one is a record widening for a term nothing yet writes; SC-2 names it as a seam instead. |

## Ours by choice (AUTHORED)

1. **The mage predicate is a mana pool.** `MaxMana > 0`. No file states the class
   bit and no research resolves which archetype carries it; the shipped table
   makes the mana column and the caster coincide (analysis). One expression,
   one place, `DD-3`.
2. **The affordance.** Nothing decoded says what makes a player cast. A key arms
   the cast and the existing enemy-click targeting then issues one. `DD-6`.
3. **The integer form of the damage factor.** The decode is a float; `pkg/sim`
   is behind the determinism wall. The exact integer form is stated in `spec.md`
   and its divergence from the float reference is **measured and disclosed**,
   not asserted away. `DD-4`, SC-4.
4. **Which spell is armed.** The table's own first row, not a literal id.

## Open

- What writes `MAGIC-RESIST-006`'s `block+0x11`/`+0x12` pair is still Unknown;
  this story builds no path that would read it.
- `MAGIC-ITEM-007`'s authoring hop for `{castSpell=...:n}` is Unknown. We already
  read brace-form strings ourselves (`pkg/mapload/spawn.go`), so parsing one
  would be **ours to author**; this story does not.

## Added after the owner spoke (2026-08-09)

| Claim | Confidence | What this story takes |
|---|---|---|
| `MAGIC-BOOK-002` | High / **Unknown** on the allocator's reachability | The book is a sparse array subscripted by spell id and **knowing a spell is a non-null slot**; the allocator sets `actor+0x4c` bit 1 and **bit 2 — the mage bit — when `manaMax != 0`**. That last clause is why DD-3 is decoded rather than authored. Its amendment matters and is taken: **nothing about the book is gated on a stat**, so neither Mind nor Spirit bounds it. |
| `HERO-EFFECT-019` | High | Names the `teachSpell` arm (kind 42) as the book's only writer. Cited for why learning is a decoded mechanism this story does not build. |

**The owner ruled as author** on the one link no research answers: *a click
selects any spell from the book, and the cursor casts it at a target.* That is
FR-10. His account of the original — that a mage has a book and learns
permanently from one-spell books — is testimony rather than a verdict, and it is
**confirmed** by `MAGIC-BOOK-002` for the book and the permanence, and
**qualified** by measurement for the learning: the mechanism exists and no
shipped item exercises it (analysis).

**Measured here, not taken from a claim.** That the `Humans` column the file
titles `knownSpells` is a bitmask subscripted by spell id is **our reading of a
column the file names**, discriminated on the corpus: over 48 stated masks on
both roots, none sets bit 0 — the id the resolver refuses — none sets a bit above
28, and none sits on a row with no mana column. It is not a research claim and
`spec.md` does not present it as one. If research later publishes the column, the
loader is the one place that changes.
