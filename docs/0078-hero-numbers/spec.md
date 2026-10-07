# Spec — the player's own units can fight

A player can order an attack and the blow resolves, but the party's own units carry six zeroes where
their combat numbers should be, so the order does nothing the player can see. This story gives a
party member a **hero** — four stats and one trained skill — and the **weapon character generation
hands him**, and derives his eight combat numbers from the two.

It is a filling-in, not a new mechanism. The eight fields are already on the entity, already in the
byte form, already read by the resolver `0075` shipped. Nothing here adds a field, moves a record or
changes a rule of combat.

The threshold is **High**: these values reach hashed simulation state.

Vocabulary. **Bare** means holding no weapon. The **active skill** is the skill slot a melee weapon
names by its own attack type; it is zero for a bare hero. **`ftol`** is truncation toward zero. The
**sheet band** of a pair is `[base, base + spread]`, which is what a player reads on screen and what
this document quotes when it compares against a measurement.

## Functional requirements

- **FR-1 — A PARTY MEMBER CARRIES A HERO AND A WEAPON, AND ITS COMBAT NUMBERS ARE DERIVED FROM
  THEM.** `PartyMember` gains a hero — four stats and six skill levels — and an optional weapon. A
  start fills the member's eight combat fields by folding the one over the other, and takes them from
  nowhere else. A member whose hero is the zero value and whose weapon is absent derives, by the same
  fold with no special case, exactly the numbers the party carries today.

- **FR-2 — THE FOLD IS THE ORIGINAL'S, TERM FOR TERM, AND IN ITS ORDER.** Given a hero and a weapon:

  1. each stat is first capped, `stat = min(stat, 50)`;
  2. `spread = ftol(1.1^Body / 20)` and `base` takes the same value;
  3. `toHit = ftol((1.1^Body + 1.1^Reaction) / 5)`;
  4. when the active skill is a slot in `1..5`, `toHit += 3 * skill[active]` and
     `base += skill[active] / 5`, truncating — **the skill reaches the base alone**;
  5. `defence = Reaction / 3`, truncating; `absorption = 0`;
  6. the weapon's own contribution is then **added**, never multiplied.

  Body enters the minimum once and the maximum twice; Mind and Spirit enter neither number; and
  `absorption` has no source but armour, which no hero in this story carries.

- **FR-3 — THE STARTING WEAPON IS RESOLVED OUT OF THE INSTALLED DEFINITION TABLE, NOT WRITTEN
  DOWN.** Character generation names a weapon by a string literal chosen from the trained skill slot.
  Resolving that literal is: split it into an optional leading shape name, an optional leading
  material name and a remainder; look the remainder up in the `Weapons` collection by exact name; and
  scale that row's columns by the **product** of the shape's and the material's `@.damage` factor,
  each of which is the fifth of the nine doubles its record carries. An absent shape name means index
  0, which is the item constructor's own default.

  The scaled numbers are `base = ftol(col6 * f + 0.5)` and `spread = ftol(col7 * f - base + 0.5)`,
  the second reading the first back as the unsigned byte it was stored as; both are byte-wide. The
  to-hit and defence contributions are `col8` and `col9` through the sixth and seventh doubles. The
  range, charge and relax columns are taken verbatim, the range reading 1 when its cell is empty.

- **FR-4 — A WEAPON CONTRIBUTES THROUGH THE MELEE ARM AND SETS THE ACTIVE SKILL.** Its damage base
  and spread are added to the hero's, its to-hit to his to-hit and its defence to his defence; its
  charge and relax times are **assigned** over the bare pair; and the active skill is assigned its
  own attack type. A row whose attack type is `10` or above takes an equip arm feeding a second
  damage component this tree does not model, and is **REFUSED BY NAME** at resolution rather than
  folded in part.

- **FR-5 — A BARE HERO IS A MODELLED STATE AND NOT A HOLE.** With no weapon the active skill is zero,
  both skill terms are skipped, nothing is added, and the cadence is the bare pair `8 / 4`. The
  result — `base = spread = ftol(1.1^Body / 20)`, zero below Body 32 — is the original's own answer
  and is produced by the same code path, with no branch that exists only for the empty case.

- **FR-6 — THE PARTY'S HERO IS THE CHARACTER-GENERATION START.** Four stats at **25**, skill slots
  1..5 zeroed and exactly one written at **10**. Which slot is trained is **ours** and is one named
  constant; the front end trains the Blade slot, whose literal is the `Iron Short Sword`.

- **FR-7 — NOTHING ELSE ABOUT A PARTY MEMBER MOVES.** Health and health maximum stay at `SpawnHP`,
  speed stays at the constructor's, reach stays at the simulation's one cell, the movement domain
  stays ground, and placement — the drop cell, the outward walk, the crowding count — is untouched.
  Each of health, speed and reach is derived from these same stats by the original and each is a
  DISCLOSED DIVERGENCE of this story, not an oversight.

- **FR-8 — NO FIELD IS ADDED TO THE ENTITY AND THE BYTE FORM DOES NOT MOVE.** `formatVersion` stays
  **12**. Every value this story produces lands in a field that already exists and is already
  serialized.

- **FR-9 — AN INSTALL THAT CANNOT ARM THE HERO SAYS SO.** The definition table is read once; the
  starting weapon is resolved beside it and, when it cannot be, the reason is CARRIED rather than
  returned — the front end still opens every map and still plays. The headless check line reports the
  hero's own damage band on every run, and the reason instead when there is no weapon.

- **FR-10 — THE MISSION TOOL CAN ORDER AN ATTACK.** `cmd/missionrun` gains a repeatable flag naming
  an attacker and a victim by the references it already resolves, drives the world until the victim
  falls or a tick ceiling is reached, and reports which and after how many ticks. It is the
  instrument this story's evidence is taken with, and it changes no default behaviour.

## Acceptance criteria

- **AC-1** A party member built from the chargen hero and the resolved `Iron Short Sword` carries
  `DamageBase 7`, `DamageSpread 3`, `ToHit 39`, `Defence 8`, `Absorption 0`, `AlwaysHits false`,
  `AttackCharge` and `AttackRelax` the weapon's own.
- **AC-2** The four measured band edges reproduce: at Body 15 / 32 / 39 / 43 with one skill at 10 and
  that sword, the sheet bands are `7-10`, `8-12`, `9-14`, `10-16`.
- **AC-3** The band is constant between those points and steps at Body 32, 39, 43, 46 and 49.
- **AC-4** A hero with no weapon at Body 31 derives `0-0`; at Body 32, `1-2`; at 39, `2-4`; at 43,
  `3-6`.
- **AC-5** A hero whose weapon names no shape word resolves through index 0 of the shape table.
- **AC-6** A weapon literal carrying a two-word material resolves to that material and not to its
  one-word suffix.
- **AC-7** Resolving a row whose attack type is 11 fails, and the failure names the row.
- **AC-8** Resolving a name whose remainder is in no `Weapons` entry fails, and the failure names it.
- **AC-9** The spread subtracts the **already-rounded** base: a row and factor for which
  `round(max*f) - round(min*f)` and `round(max*f - round(min*f))` differ yields the second.
- **AC-10** A stat above 50 derives as 50; a stat at 50 derives unchanged.
- **AC-11** Mind and Spirit reach none of the eight numbers: changing either alone moves nothing.
- **AC-12** The skill reaches the base and the to-hit and **not** the spread.
- **AC-13** A zero-value hero with no weapon derives the eight numbers the party carries today —
  five zeroes, `AlwaysHits false`, cadence `8 / 4`.
- **AC-14** A world holding a party built this way round-trips through the byte form to equal bytes
  and an equal hash, and the encoded version byte is **12**.
- **AC-15** The check line states the hero's damage band; with a nil weapon it states why instead.
- **AC-16** Over Body and Reaction each in `0..100`, no term of the fold comes within `1e-6` of an
  integer boundary before truncation.
- **AC-17** `missionrun` with an attack flag reports the victim's fall and the tick it happened on;
  with none it prints exactly what it printed before.
- **AC-18** Every existing test of placement, spawning, the world's digest and the byte form is
  unchanged in outcome.

## Properties

- **P-1 — The derive is a pure function.** It reads its hero and its weapon and nothing else: no
  clock, no generator, no global, no file. The same pair yields the same eight numbers in every
  process.
- **P-2 — `pkg/sim` is not edited.** No rule of combat, no field, no version. The determinism wall is
  untouched: every float in this story lives in `pkg/data`, below it, and only integers cross.
- **P-3 — The refusals are total.** A weapon that does not resolve yields the zero value and an
  error, never a partly filled one — the rule `NewUnitDef` already holds to.
- **P-4 — One place reads the definition table.** The starting weapon is resolved from the same parse
  that builds the placement table, so an install cannot answer one of them and not the other.
- **P-5 — The import graph is unchanged in direction.** `pkg/data` gains `math` and no intra-module
  edge; `pkg/mapload` and `pkg/game` gain no import at all.
