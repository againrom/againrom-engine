# Spec — a scenario human fights with his own numbers

**Intensity: spec-anchored / static.** The contract crosses three tiers and lands in hashed
simulation state, which is the profile's cross-cutting-engine default. No watcher tool exists here,
so this is static plus discipline.

**Terrain: brownfield** for the world builder, the definition table and the collection interface;
**greenfield** for the humans definition.

Every unit a map places today that is a PERSON rather than a creature is built with no damage, no
to-hit, no defence and no absorption, and with a health this project invented. It cannot hurt
anything and nothing about it is its own. Creatures already carry their whole template. This story
closes the second arm.

## Functional requirements

- **FR-1 — a placement that reaches a `Humans` entry yields a DEFINITION, and that definition is
  twenty-three parameter slots read in ascending order.**

  The slots and their destinations, which are the contract:

  | Slots | Destination |
  |---|---|
  | 0, 1, 2, 3 | Body, Reaction, Mind, Spirit |
  | 4 | health maximum, and the current health follows it |
  | 5 | mana maximum, and the current mana follows it |
  | 6 | speed |
  | 7 | rotation speed |
  | 8 | scan range |
  | 9 | defence |
  | 10 to 15 | the six skill levels, at their own slots, 0 first |
  | 16 | type id |
  | 17 | face |
  | 18 | read and DROPPED |
  | 19, 20 | attack charge time, attack relax time |
  | 21 | token size |
  | 22 | movement type |

  **An empty cell stores nothing and advances anyway**, leaving the constructor's value standing and
  shifting nothing after it. A reader that took "field *i* from column *i*" would mis-assign
  everything from slot 10 onward and would still look right for the ten before it.

  A row carrying FEWER THAN twenty-three cells is REFUSED BY NAME, with the zero value and never a
  partly filled definition; twenty-three cells is every slot present and loads. A refusal takes the
  whole world with it, exactly as an unmodelled damage arm on the other collection does.

  **The value an empty cell leaves standing is OURS**: the base actor's, the same numbers the other
  collection's loader falls back to, reachable only for a cell a shipped row leaves empty. The
  per-cell law is decoded and shared; which numbers this object's own constructor writes is not
  established either way, so this is an authored divergence and not a reading.

  It is built for all three humans rungs alike. Which rung reached the entry says nothing about what
  the entry is worth.

- **FR-2 — the numbers a BLOW reads are DERIVED from that definition's statistics; they are not
  columns, and five families of them do not exist as columns at all.**

  This collection ships no damage pair, no routing selector, no absorption, no elemental protection
  and no damage-kind resistance. A build that read the columns and stopped would ship a person who
  swings for nothing, and would look complete.

  What is derived. It is the same graph a generated character goes through, and the ORDER MATTERS at
  two points: the caps run first, so every later term reads a capped statistic, and the base is
  copied from the spread before the skill term is added, so the halves carry different terms.

  1. the statistic caps;
  2. the damage SPREAD from Body, and the base copied from it, so a bare person's roll is `[d, 2d]`
     and is nothing at all below a Body this mission's people do not reach;
  3. to-hit from Body and Reaction together;
  4. the active skill — the equipped weapon's own kind, and none at all when bare — adding three
     times its level to to-hit and a fifth of its level to the BASE alone;
  5. defence off Reaction, and absorption zero;
  6. the weapon's own damage, to-hit and defence ADDED. Never multiplied.

  **The mark that makes a blow always land is FALSE on every placement of this arm**: its only
  source is the other collection's routing column.

  **What is carried and not read, exactly.** The streamed `defence` cell is REPLACED by step 5. The
  six skill levels are NOT: step 4 reads the level of the weapon's own kind, so five are live, and
  only level 0 reaches nothing — it is the cell the streamer copies into the live to-hit, which step
  3 replaces, and step 4's gate is on a positive kind.

  **The simulation carries no elemental protection and no damage-kind resistance FIELD at all**, so
  the families this collection lacks have nowhere to be absent from.

- **FR-3 — a person fights with the WEAPON HIS OWN ROW NAMES.**

  A `Humans` entry carries ten trailing equipment names. The weapon is the **first one that resolves
  as a melee weapon row** against the installed table's shape, material and weapon collections. A
  name that resolves as nothing, or as a weapon whose kind takes the ranged arm, is passed over — it
  is armour, a shield, or a weapon this build does not model — and the search continues. A row whose
  cells yield no weapon is BARE, an ordinary state and not a failure.

  Position is NOT the rule. Nothing here assumes the weapon is the first cell. A name is worth what
  it is worth to a generated character holding the same item: one answer, not two.

  **The three collections GO TOGETHER: a table missing ANY ONE of them arms nobody**, and it is not
  an error. Stated as a set rather than per collection because a missing scale table does not mean
  "no scaling" — its absent factor reads as one, and a shipped row is several times what an item of
  that name actually carries, so arming off half the ladder gives a weapon the game does not ship.

- **FR-4 — a weapon's cadence assigns over its wielder's OWN pair, each half separately, and only
  where the weapon's own cell is not empty. ONE RULE, for every wielder in the tree.**

  What differs between wielders is only what "his own pair" means. A placed person's is the two
  columns his row ships. A generated character has no template, so his is the bare-handed pair the
  unequip inverse restores — and that pair must not reach a placement that has columns of its own.

  **This narrows a rule the generated character already lives under**, deliberately: today a weapon
  whose cadence cell is empty assigns that empty cell onward, which the simulation's floor reads as
  the fastest possible attacker. One predicate now decides it in both places.

- **FR-5 — the health and the rate come off the definition too.** The health maximum is the row's
  own column and the current health is that same number, so a placed person is at full health by
  construction; it replaces the provisional value this project chose, which stays where it is for a
  placement resolving to nothing. The rate is the row's own speed column.

- **FR-6 — the difficulty setting NEVER reaches this arm, at any of its three values.** Not omitted:
  the thing being reconstructed steps over the whole adjustment for this class. A person's health,
  to-hit and defence are the same three numbers at every setting.

- **FR-7 — nothing else about a world moves.** A placement reaching the OTHER collection is built
  exactly as it is today — same eight numbers, health, rate, domain and adjustment at every
  difficulty. One reaching NOTHING keeps the provisional health, the constructor's rate and its
  eight zeroes. A world built with no table at all is unchanged, byte for byte.

- **FR-8 — NO SIMULATION FIELD IS ADDED and the byte form's version does not move.** Every field
  this story fills was already declared, carried in the canonical form at its own offset and width,
  and read by a tick, so a world written before this change decodes without a migration. What moves
  is the DIGEST of any world holding a person a map placed — which is the point.

## Out of scope, each with its reason

- **Armour and shields.** They move absorption and defence, and modelling them needs two more
  collections' fills and a slot map this story does not open. Absorption stays zero on this arm,
  which is what a wielder of a weapon alone carries in any case.
- **The other collection's own equipment cell.** Twenty-six of its fifty-six classes carry one; none
  of the three this mission places does.
- **A ranged weapon.** Its damage feeds a second component nothing in this tree reads, so such a
  cell is PASSED OVER like any other cell that is not a melee weapon and the search continues — a
  person whose only weapon is ranged is bare, and one carrying a bow and a sword takes the sword.
  Nothing here swings for a number taken off the wrong arm.
- **The auto-hit mark's absorption skip.** The blow resolver subtracts absorption unconditionally;
  the mark is decoded to skip it. No person carries the mark — FR-2 — so this is entirely about the
  OTHER arm, where eleven of this mission's twenty-one hostiles do carry it; and it changes no
  outcome while the party's absorption is zero, which it is. It belongs to the resolver, not to a
  loader, and it is named here so it is a known gap.
- **Whether the derivation runs again after the map is loaded.** It runs once, here.
- **Health derived from Body and experience.** That alternative reading needs a class bit settled
  for a generated hero and for no scenario person. The column is taken and the alternative is
  recorded rather than approximated.

## Acceptance criteria

- **AC-1** Each of the twenty-three slots moves exactly the fields FR-1 names for it and no others,
  witnessed by a row empty everywhere but that one cell. Three slots are not one-to-one and the table
  says so: two of them move a pair, and one moves nothing at all.
- **AC-2** A row of entirely empty cells yields exactly the constructor defaults, and no field of any
  loaded definition ever holds the empty value.
- **AC-3** A row of fewer than twenty-three cells is refused, the refusal names the row, the result
  is the zero value, and a map holding such a placement yields no world.
- **AC-4** All three humans rungs reaching one entry carry the same numbers.
- **AC-5** A bare person's damage pair is two equal halves of the Body term, his to-hit carries both
  Body and Reaction, his defence is a third of Reaction and his absorption is zero; the always-hits
  mark is false.
- **AC-6** An armed person's four numbers each exceed the bare person's by that weapon's own, and
  the weapon's skill kind adds its level's terms to to-hit and to the base alone.
- **AC-7** The weapon is the first equipment cell that resolves: a row whose first cell names a
  shield or a mail and whose second names a weapon is armed with the second; a row none of whose
  cells resolves is bare; a row whose only resolvable cell is a ranged weapon is bare.
- **AC-8** A table missing any one of the three item collections leaves every person on a map bare
  and raises nothing — each of the three witnessed missing on its own, not only all three together.
- **AC-9** A bare person's charge and relax are his row's own two columns — witnessed on a row whose
  columns are NOT the pair a generated character falls back to, so the two cannot pass by being
  equal. An armed one takes his weapon's where the weapon's cells are not empty and keeps his own
  where they are, each half witnessed separately. A generated character holding a weapon with an
  empty cell keeps his bare pair by the same predicate.
- **AC-10** A placed person's health and health maximum are both his row's health column, and his
  rate is his row's speed column.
- **AC-11** A placed person carries the same health, to-hit and defence at all three difficulty
  values.
- **AC-12** At every difficulty, creature and unresolved placements carry exactly the numbers,
  health, rate and domain they carry before this story, and a table-less world is unchanged.
- **AC-12a** A world holding a person a map placed hashes DIFFERENTLY from the same map built with
  no table, and the difference is exactly the fields this story fills: lifting them back out of every
  record reproduces the table-less digest.
- **AC-13** `pkg/sim`'s byte form is untouched: the version literal, the record width and every
  offset are where they were, and a world holding armed people round-trips through the form.
- **AC-14** Against both lawful roots, mission 1's two brigand placements resolve to their own
  entry, carry that entry's health, and swing for a band whose top is above zero; and the mission
  tool has a party member fell one of them.

## Properties

- **P-1** Total. No parameter array, equipment cell or missing collection, in any combination,
  panics, divides by zero, or leaves a definition half built.
- **P-2** Pure. A definition and its derived numbers are a function of the row and the installed
  table alone — no clock, no generator, no global, no file — so the same pair yields the same numbers
  everywhere.
- **P-3** One authority per question: what an item of a given name carries is answered identically
  for a placed person and a generated character, and an empty cadence cell means the same to both.
- **P-4** A definition never holds the empty cell value in any field.
- **P-5** The full local gate is clean: build, vet, gofmt, the test suite, and the three repo scripts.
