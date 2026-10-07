# Spec — the hero is a real fighter, and his numbers are on screen

The party's hero carries **25 in all four statistics**. That is not a character: it is the value
character generation writes into the four rows *before* the player spends a point, and the party has
been carrying the opening state of a screen as though it were a build. Of the four statistics only
Body reaches damage and only Body and Reaction reach to-hit, so half of a flat spread buys nothing a
blow reads.

This story gives him a **generated build inside the rules character generation actually enforces**,
and puts his numbers **on screen** — the four statistics, the trained skill, the weapon and the eight
derived combat numbers, on the unit information panel that already describes a selected unit.

The threshold is **High** for the build, which reaches hashed simulation state, and **Medium** for the
display, which reaches nothing.

Vocabulary. A **spread** is an assignment of the four statistics. A **step** is one click of `+` or
`-` on one statistic. `ftol` is truncation toward zero. The **sheet band** of a damage pair is
`[base, base + spread]`. A unit's **character** is its four statistics, its trained skill and the
weapon in its hand, **as the load that placed it knows them** — it is a fact about what was placed
and not a field of the simulation, so a unit the loader knows no character for has none, and that is
every unit this tree does not place from a party.

## Functional requirements

- **FR-1 — THE POINT-BUY COST OF A STATISTIC IS A CUMULATIVE FUNCTION, NOT A PRICE PER STEP.**
  `T(n) = ftol(0.349 * 1.15^(n-1) + 0.5)` is the **total** cost of one statistic standing at `n`.
  The cost of a `+` step from `v` is `T(v+1) - T(v)`; the refund of a `-` step from `v` is
  `T(v) - T(v-1)`. Consequently a refund taken at `v` equals the cost of the step that reached `v`,
  for every `v`, and the escalation is real: the step into 45 costs 22 points where a step in the low
  twenties costs 1. The step and the refund are **arithmetic that can be asked for**, not a
  transaction: nothing in this story spends a point.

- **FR-2 — A SPREAD IS ONE CHARACTER GENERATION COULD HAVE PRODUCED EXACTLY WHEN IT IS INSIDE BOTH
  BOUNDS.** Every statistic lies in `[15, 45]`, and `T(Body) + T(Reaction) + T(Mind) + T(Spirit)` is
  at most **140**. The pool a generation screen would show is `140 - sum T(stat)`, which counts what
  remains rather than what was spent. Both bounds are required and neither implies the other; the
  budget is the binding one, and under it the click ceiling of 45 is unreachable.

- **FR-3 — THE PARTY'S HERO IS A GENERATED FIGHTER, AND THE SPREAD IS OURS.** He carries
  **Body 43, Reaction 26, Mind 15, Spirit 15**, trained in the blade slot at level 10, and holds the
  weapon that slot's generation arm hands him. The spread satisfies FR-2. **Which** legal spread a
  fighter should carry is decided by no source: it is authored and it is disclosed as authored
  wherever it is stated. **Everything that states the party's hero states the same hero** — the
  party a mission is built from and the headless line that reports him answer with one character, not
  with two constructions that agree today.

- **FR-4 — THE GENERATION START REMAINS CONSTRUCTIBLE AND UNCHANGED.** A hero of four statistics at
  25 with one slot at 10 — the screen's opening state, worth 40 of the budget with 100 remaining — is
  still available and derives exactly what it derived. The party no longer uses it.

- **FR-5 — NOTHING ABOUT THE FOLD CHANGES.** The eight combat numbers a hero and a weapon produce are
  produced by the arithmetic already shipped, in its order, term for term. The hero's numbers move
  because his inputs moved and for no other reason.

- **FR-6 — THE UNIT INFORMATION PANEL STATES A UNIT'S EIGHT COMBAT NUMBERS.** For the unit it
  describes: the damage pair as the sheet band `base-(base + spread)`; to-hit; defence; absorption;
  the attack cadence as `charge/relax`; and the always-hits mark, stated only when it is set. These
  are the simulation's own values for that unit, read where they live, and every unit on the field
  has them. A subject nobody supplied them for is **not told**, and states none of these rows — which
  is a different state from a unit whose numbers happen to be zero, and the two are stated
  differently.

- **FR-7 — AND, FOR A UNIT THAT HAS A CHARACTER, IT STATES THAT TOO.** Four statistics in the order
  **Body, Reaction, Mind, Spirit**; the trained skill by name and level; the weapon by name. **A
  skill's name is the shipped table's own warrior half** — the file titles each shared slot with a
  warrior name and a mage name in one string, and this tree, having no class axis, prints the warrior
  one and says so. A unit
  with no character states none of these rows, a hero with no trained skill states no skill row, and
  a hero holding nothing states no weapon row. **A row with no value costs no space**, so the panel
  for a unit that has no character is the panel it was before this story with the combat rows added,
  and nothing else.

- **FR-8 — THE HEADLESS CHECK LINE STATES THE SPREAD AS WELL AS THE NUMBERS.** The one line
  `-check` prints for the party's hero names his four statistics, his trained skill and level, his
  weapon, his band, his to-hit and his defence. A hero who resolved no weapon still says so and says
  why, as it does today.

- **FR-10 — THE HERO'S MOVEMENT SPEED IS DERIVED FROM HIS REACTION, NOT TAKEN FROM A CLASS
  DEFAULT.** `speed = Reaction` below a Reaction of 12, and `Reaction / 5 + 12` at 12 or above,
  truncating, with the statistic capped first as every other derived number caps it. A party member's
  speed is that number and no longer the unit-table constructor's 10.

  Two terms of the published law are **deliberately not implemented, each for a stated reason and
  neither by silence**. The `+10` arm applies to two type ids that no human class holds, so a hero
  cannot reach it. The overload penalty applies only when carried load reaches capacity, and this
  tree has no inventory, so there is no load. Neither omission is a simplification of a term that
  could fire.

  The speed is **stated on the sheet** beside the numbers a blow reads, because it is a derived hero
  statistic and the story's subject is that his derived numbers are right and visible.

- **FR-9 — NO SIMULATION FIELD, RECORD, VERSION OR RULE MOVES.** No field is added to an entity, no
  byte-form record changes shape, `formatVersion` stays at **13**, and no rule of combat, movement or
  scripting is edited. A world holding this party round-trips to equal bytes and an equal hash.

## Acceptance criteria

| # | GIVEN | WHEN | THEN |
|---|---|---|---|
| AC-1 | the cost function | asked for 15, 25 and 45 | it answers **2**, **10** and **164** |
| AC-2 | the cost function | asked for the step from 44 to 45 | it answers **22**, and the step from 20 to 21 answers 1 |
| AC-3 | the cost function | a refund is taken at any `v` in `16..45` | it equals the cost of the step that reached `v` |
| AC-4 | the budget and the floor | three statistics are floored at 15 | the fourth reaches **43** and not 44 |
| AC-5 | the budget | three statistics stand at 25 | the fourth reaches **42** and not 43 |
| AC-6 | the budget | all four are raised together | each reaches **34**, spending exactly 140 |
| AC-7 | the budget | any spread is offered | 4 x T(25) + 100 equals 140, so the pool and the budget are one number |
| AC-8 | the legality test | a spread holds a statistic at 14, or at 46, or spends 141 | it is refused, and each of the three is refused for its own reason |
| AC-9 | the party's spread | it is tested | it is legal; it holds Mind and Spirit at the floor; no legal spread has a higher Body; no legal spread with its Body **and both floors** has a higher Reaction; and those three conditions together admit **exactly one** spread, which is it. It spends 139 of 140, and the point is unspendable rather than unspent — no legal spread raises Body or Reaction by holding it |
| AC-10 | the party's hero | he is derived against the blade slot's generation weapon at `(5, 3)` with to-hit 5 | his band is **10-16**, his to-hit **49**, his defence **8**, his cadence the weapon's |
| AC-11 | the owner's measured bands | the hero's Body is 43 | his band is the band measured at Body 43, `10-16`, on the nose |
| AC-12 | the generation start | it is constructed and derived against the same weapon | it answers exactly what it answered before this story: `7-10`, to-hit 39, defence 8 |
| AC-13 | a panel subject holding a character | the panel is composed | it states the four statistics in the order Body, Reaction, Mind, Spirit, then the skill, then the weapon |
| AC-14 | a panel subject holding no character but holding combat numbers | the panel is composed | it states every row it stated before this story — the selection count when the selection holds two or more, the name, the health pair and the cell — plus the combat rows, the always-hits row among them only when the mark is set; and **no** statistic, skill or weapon row. The box is exactly as tall as its kept rows |
| AC-14a | a panel subject told neither a character nor combat numbers | the panel is composed | it is the panel it was before this story, row for row and pixel for pixel: a value nobody supplied is *not told*, never a unit whose numbers are zero |
| AC-15 | a subject whose damage pair is `(10, 6)` | the damage row is composed | it reads `10-16`, the sheet's own composition, and never `10-6` |
| AC-16 | a subject that does not always hit | the panel is composed | the always-hits row is absent; a subject that does states it |
| AC-17 | a running mission | the party's hero is selected | his panel states his statistics and his numbers, and a monster's panel states a monster's numbers |
| AC-18 | a complete install | `-check` is run | the line names the spread, the skill, the weapon, the band, the to-hit and the defence — and the four numbers it names are **the party member's own**, equal value for value to what a started mission places, not merely of the right shape |
| AC-19 | a world holding this party | it is encoded and decoded | the bytes are equal, the hash is equal, and the version byte is **13** |
| AC-21 | the speed law | a Reaction of 11, 12, 26 and 45 is offered | it answers **11**, **14**, **17** and **21**, and the branch changes at 12 and not at 11 or 13 |
| AC-22 | a started party member | his speed is read off the entity | it is his hero's derived speed and not the unit table's 10; a member with no statistics derives his by the same arithmetic and takes no fallback |
| AC-23 | the sheet | a unit with a character is described | it states that speed, and a unit told no combat numbers states no speed row |
| AC-20 | the whole suite | it is run with no game present | it is green; and the **only** expectations that changed are those that state the party hero's own numbers or the line that reports them, each one enumerated. No placement, spawn, digest, byte-form, panel or readout expectation is edited at all — a pin that had to be re-baselined is a finding to be reported, not a number to be adjusted |

## Properties

- **P-1 — The legality test is complete and is not a sampler.** It admits a spread exactly when both
  bounds hold, for every spread, and the four statistics are independent of one another in it.
- **P-2 — `pkg/sim` is not edited.** No rule, no field, no version. The determinism wall is untouched
  from both sides, and every float this story introduces stays below it.
- **P-3 — The import graph is unchanged in direction**, and the window tier still names no
  simulation, format or data type: everything it receives is a builtin or a value of its own.
- **P-4 — The panel is a function of its arguments.** Composing it reads no viewer state, opens no
  file and consults no clock, so what is on screen and what a test asserts are the same pixels.
- **P-5 — The picture is rebuilt exactly when what it states changes.** A statistic, a combat number
  or a weapon that differs rebuilds it; a walking unit whose stated values are identical does not.

## Out of scope

Character generation **as a screen** — nothing spends a point at runtime. Levelling, experience,
equipping, unequipping and the inventory. Health, mana, speed, sight, capacity, reach, the five
resistances and the five protections: each is derived from these statistics by the original, none is
derived here, and the panel states none of them. A class axis, and with it the mage arm of generation
and the mage half of every skill name. Any change to the debug readout.

## Error cases

A spread outside either bound is refused by the legality test and is never handed to a hero.
A weapon that will not resolve leaves the hero bare, which is a state of the original: he keeps his
statistics, derives his unarmed numbers, and the panel states no weapon row. No new error path
reaches the player.
