# One Human derive

## Intent

Every Human's derived block comes from one derive. Before this story a native
hero went through `Hero.recompute` and an original-SAV, source-backed or town
Human through `HumanState.derive`: two producers of one state, each step
written twice. Owner decision 2 of the architecture audit at `34ae6dac`
(row 2): one derive, building on `rules.HumanSpeed`. One builder per kind; a
loaded state and an engine-made state are one kind of state. Base: `055e3936`,
merged with main `d023d087`.

## Authority

- `HERO-ORDER-014`: the recompute is one virtual method and its order is
  load-bearing.
- `HERO-CAP-015`: a stat is `min(stat, 50 + (int8) cap term)`; an effect
  raises stat and cap term together and clamps the stat alone at 100.
- `HERO-HP-005`, `HERO-MP-006`: the two pools, each intermediate truncated and
  stored as a word, reading the experience dword.
- `HERO-SIGHT-007`: the sight word `ftol(((mind + reaction)/25 + 4) x 256)`
  and capacity `Body x 10 + 1`.
- `SAV-1116`, `HERO-SPEED-008`, `MOVE-RATE-053`: speed, the overload penalty
  before the modifier, the clear of a negative sum, the mover byte.
- `HERO-DAMAGE-022`, `HERO-MOD-016`: the damage pair is two bytes; the
  modifier block is folded by plain adds.
- `HERO-GENERAL-092`: the General bonus is inert on a recomputing Human.
- `DIV-2217` (owner): a worn skill bonus may lift slots 1 to 5 above 100.

## Comparison

`pkg/game/humanderive_test.go` feeds both routes the same Human and lists every
field that differs. Population: the 210 Humans rows of each ROM1 root with
their own starting equipment, each also overloaded (load 4000) with speed
modifier 0, +4 and -9 and as a rider (1050 cases per root); the 1344 Humans the
save corpus holds (`gameversions/saves`, 121 files read, 1 file not a SAV);
21 synthetic load, modifier and rider cases. A corpus Human is compared twice:
as loaded, and rebuilt from the native inputs. EN and RU give the same table.

Before (commit `83d6fa57`, code at `10381cbe`):

| field | inputs | native | stored | decided by |
|---|---|---|---|---|
| Skill 1, to-hit, damage base | trained 100, bonus +10 (3 corpus Humans; synthetic 98 + 10) | 110, 385, 73 | 100, 355, 71 | `DIV-2217`: the bonus lifts the level |
| damage pair | Body 100 through a +50 cap term (synthetic) | 694 / 693 | 182 / 181 | `HERO-DAMAGE-022`: two bytes |
| Skill 0 | General bonus +6 (synthetic) | 10 | 4 | `HERO-GENERAL-092`; kept native, `DIV-2784` |
| pools as loaded | experience dword below the six counters' sum (29 corpus Humans, `oldsaves7/game0005.sav`) | 155 | 44 | input, not arithmetic: each route supplies its own experience |
| every other field | all cases | equal | equal | |

After: one value per field except Skill 0 under a General bonus (`DIV-2784`)
and the 29 experience inputs. The rebuilt corpus and all templates differ in
no field.

Not differing over this population, but changed where they would: the defence
clamp now follows the weapon's defence (the original folds the whole modifier
block before it clamps), sight for a negative Mind plus Reaction follows the
claimed floating form, and a stat cap term is no longer limited to the int8
range on the native route only.

## As built

### One derive

`data.DeriveHuman(HumanInput) (HumanOutput, error)` (`pkg/data/humanderive.go`)
is the derive. Input: the four stats and cap terms, the six skill levels (slot
0 the live General level), the active slot, the experience, the class flag,
the mana-pool gate, the rider flag, the carried load, the training cap and the
modifier block's additive terms (`HumanTerms`, sight in 1/256 cell). Output:
the capped stats, both pools, the sight word, capacity, the unencumbered base
speed, the derived speed word and the modifier it keeps, the six levels,
to-hit as a word, the damage pair as bytes, defence, absorption, protection and
resistance. Steps: caps, pools, sight and capacity, speed (`rules.HumanSpeed`),
damage pair and to-hit, skill restore (`rules.EffectiveSkill`), active-skill
terms, the fold, the clamps.

Two inputs differ by route, as the audit named: the rider flag (a stored
Human's type word 0x13 or 0x15, a native Human's `Profile.Rider`) and the sight
term's sub-cell part (only a stored modifier word carries one; a native item
adds whole cells).

### Callers

- `Hero.Recompute` and `RecomputeWithSkillXP` (`pkg/data/recompute.go`):
  native heroes, the party (`mapload.partySpawn`, `PartyDisplayWithTable`;
  the second game's generated hero comes this way, `DIV-2772`),
  rearm (`pkg/game/rearm.go`), placed persons (`HumanDef.DerivedWithLoadout`,
  `blockFor`), chargen previews, the town Human's native projection
  (`missioncity.go`, `savnativeattributes.go`). It sums the six experience
  counters, puts the weapon's additive part into the terms and keeps
  `FoldWeapon` for cadence, reach and the ranged component. It reports the
  unencumbered sum speed with the modifier inside, as a native Entity holds it.
- `HumanState.derive` (`pkg/data/humanstate.go`): original-SAV and
  source-backed Humans (`mapload.deriveSourceActor`, `actorload.go`,
  `potion.go`), training and sale (`Train`, `RefreshInventoryLoad`), town
  Humans (`nativeCityHumanFromDerived`). It stores the outputs at the original
  widths and keeps the state-only steps: the load from weight and container,
  the health and mana clamps, the mana floor, the mover byte.
- `HumanState.NativeMovementBase` reads `data.HumanBaseSpeed`.

### Overload in pkg/sim

The overload a native Human takes at a load change stays in `pkg/sim`
(`humanSpeedWord`, `deriveNativeHumanSpeed`, `spawnSpeedWord` in `mapload`).
`pkg/sim` does not import `pkg/data`, and its load moves at every pick-up and
drop without a recompute. All three now call `rules.NativeHumanSpeed`, the
derive's own speed step (`rules.HumanSpeed`) on the unencumbered sum.

### Story 1385 review notes

- N1: engine LOAD of a SAV without the split operand whose derived word
  differs from its speed word (a pre-split SAV of an overloaded native Human)
  runs the derive once (`Entity.restoreValues`). Speed word and turn rate
  agree from the first tick. A Human whose word equals its speed word keeps
  the saved turn rate.
- N2: the clear of a negative sum zeroes the modifier speed word wherever the
  native basis knows its bytes (`clearSpeedModifierBasis`, in
  `deriveNativeHumanSpeed` and after an effect's basis delta). SAVE writes 0,
  as `SAV-1116` stores it. `DIV-2760` revised.
- N3: `DIV-2785` states that a ROM2 native Human runs the ROM1 overload rule.
  `R2-ENGINE-290` gives the ROM2 speed step "minus the load term, minimum 6"
  with item modifiers after it; the load term is Unknown there.
- N5: load and capacity are compared and divided in 32 bits
  (`rules.HumanSpeed`). No claim states the width of the original's compare. A
  stored Human passes its signed load and capacity words, so its compare is
  the word's.

### Mod seam

A mod formula for the derived block (MOD milestone item 4) attaches at
`DeriveHuman`: its input and output are the whole block, and both routes
reach it. No mod surface is added.

## Hashed state

Moves only where the comparison found a difference a claim or owner row
decides:

- a stored Human re-derived with a trained level plus bonus above 100
  (`DIV-2217`);
- a derived damage byte above 255 (Body above about 85 through cap terms);
- a native defence below zero before the weapon's defence;
- a pre-split engine SAV of an overloaded native Human with a modifier (N1);
- a cleared negative modifier's SAV word (N2);
- a native load above 32767 (N5).

No World golden, scenario result or census line moved: the full test, release
and scenario sets and the milestone-2 family ran unchanged (see Proof).

## Proof

- `TestHumanRoutesCompareSynthetic`, `TestReleaseHumanRoutesCompare` (EN, RU):
  the comparison above, failing on any difference other than `DIV-2784` and the
  experience input.
- `pkg/data/humanderive_test.go`: each step against hand-worked claimed
  arithmetic: pools 42/105 for experience 3881, caps, sight word 1484,
  speed with rider, overload, Haste, Slow and a load of 40000, damage bytes
  (689 stores 177), the skill restore and `DIV-2217`'s bound, the clamps.
- `pkg/rules/humanspeed_test.go`: the 32-bit compare and `NativeHumanSpeed`.
- `pkg/sim/overloadorder_test.go`: `TestPreSplitSAVDerivesTheTurnRateAtLOAD`
  (word 10, turn 10 from saved 19 and byte 6), `TestClearedModifierClearsTheBasisWord`.
- `TestReleaseNativeAndOriginalHumanShareOneDerivedBlock` (EN, RU): mission
  101, a native mage (Body 60, Reaction 60) and an overloaded companion. Each
  acts with one block (mage: health 208, mana 419, word 22, sight 7, to-hit 46,
  damage 5-5, defence 16; companion: 29, 0, word 6, turn 6, sight 4, to-hit 3,
  defence 5) through SAVE, cold LOAD and 24 ticks; the same file without its
  engine-state leaf loads both as original Humans with the same block for 24
  ticks, and the stored-state derive of each gives it again.

## Ledger

- `DIV-2784` opened: a native hero's General level carries the worn General
  bonus.
- `DIV-2785` opened: a ROM2 native Human's overload step is the ROM1 rule.
- `DIV-2786` opened: a ROM2 Human's stat caps are ROM1's 50 plus the term.
- `DIV-2217` revised: the effective level applies to every Human.
- `DIV-2760` revised: the clear also zeroes the SAV modifier word.

## Open debt

- `DIV-2784`: the General bonus leaves a native hero's derived block only with
  the training inverse.
- `DIV-2785`, `DIV-2786`: the ROM2 derive's caps and load term; a ROM2
  generated hero (`DIV-2772`) takes the ROM1 caps until a per-game cap input.
- Placed persons keep their spawn speed from the row's Speed column and their
  sight from its ScanRange column; the derive's values reach their combat
  block, pools and capacity only.
- The pools of a native Human read the sum of his six counters; an original
  Human reads his stored experience dword.
