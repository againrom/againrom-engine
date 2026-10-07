# 1001-spell-effects — fix round 3, weapon-borne release lane

Branch `story/1001-r3-weapon`, based on `efd89b8`. This lane owns R3-B1, R3-B2 and R3-B3, the
weapon-borne release findings of `docs/1001-spell-effects/round3-review.md`.

Files touched: `pkg/sim/spell.go`, `pkg/sim/combat.go`, `internal/archtest/dag.go` (the new
developer tool's own DAG row), and the test files listed under each finding below. A new developer
tool, `cmd/weaponspellcheck`, reproduces the review's own before/after measurement against a lawful
install; it is described under Measurements.

Every fix below has a witness that fails under the revert named beside it and passes on this
branch. Each was verified by making the named change, running the named test, reading its failure
message, and undoing the change. The full suite is green on a clean tree:

```
go build ./... && go vet ./... && gofmt -l $(git ls-files --cached --others --exclude-standard '*.go') \
  && go test -trimpath -count=1 ./...
bash scripts/check-no-game-assets.sh
```

## R3-B1 — a weapon-borne release refused every non-damaging row

**What was wrong.** `releaseWeaponSpell` (`pkg/sim/spell.go`) applied only `applySpellDamage`,
behind `!rule.Damaging || rule.ID == 14` and, separately, `!rule.TargetsUnit`. It is reached as the
**replacement** for a mage's physical blow, so a mage whose weapon carried a non-damaging row (Stone
Curse) neither cast nor struck: the attack was replaced by a release that then refused itself,
strictly worse than being unarmed. `MAGIC-ITEM-007` and `MAGIC-CAST-003` establish that a weapon or
item release runs the ordinary apply with the mana gate exempted; nothing in either claim requires
the row to be damaging or unit-targeted in the `TargetsUnit` column's own sense — that column gates
a **commanded** cast's choice of victim (`SpellRule.Area`'s own doc, spell.go), not an applied
release.

**What was built.** A new shared function, `weaponSpellApply(ai, ti int, rule SpellRule, power
int32) bool` (`pkg/sim/spell.go`), is the one place a weapon-borne release's row reaches an apply. It
routes by the row's own shape (DD-4, spec.md): an Area row lands at the target's own cell through
`landAreaFacing`, the same function a unit-targeted book cast reaches; a point row reaches
`ordinaryEffect` at `ai`/`ti` directly, gated by the same `pointEffectRefusal` a book cast's landing
asks. `releaseWeaponSpell` (the caster's replacement arm) now keeps only the refusals that are its
own — victim alive, not self, sighted, in range — and delegates the row's own applicability entirely
to `weaponSpellApply`. The two former gates, `!rule.Damaging` and `!rule.TargetsUnit`, are gone from
`releaseWeaponSpell` itself.

Two rows are refused **inside** `weaponSpellApply`, both by id, and neither is a narrowing of this
fix:

- **id 14, Prismatic Spray, stays refused.** `MAGIC-SING-019` (g) is High: the item-cast entry
  refuses id 14 by name (`L05236 CMP EAX,0xe`). This is the one row research measured refused, and
  the review's own control — removing this exact clause — measured a shipped carrier deliver damage
  it must not.
- **id 25, Control Spirit, is refused too**, for ownership reasons the decode does not itself
  require. See R3-B3's own note below and `DIV-056`.

**Witnesses.**

- `TestANonDamagingRowNowReleasesThroughTheOrdinaryApply` (`pkg/sim/weaponspell_test.go`): a
  Stone-Curse-shaped row (`EffectKind: EffectAbsorption`, `Damaging: false`) released through a
  weapon now raises the victim's `Absorption` from 0 to 5 and marks it. **Revert**: restore
  `releaseWeaponSpell`'s former `if !rule.Damaging || rule.ID == 14 { return }` ahead of the
  `weaponSpellApply` call. **Failure**: `the victim's Absorption is 0, want 5 — a non-damaging
  weapon spell must still release` and `the victim carries no spell effect mark, want one`.
- `TestTargetsUnitDoesNotGateAWeaponRelease`: a Damaging row with `TargetsUnit: false` still damages
  its victim. **Revert**: restore the same two former gates. **Failure**: `the victim is at 100,
  want 90 — TargetsUnit does not gate an applied release`.
- `TestFR10sFiveRefusalsLeaveTheVictimUntouched/the_row_is_id_14,_Prismatic_Spray`: id 14 is refused
  even though `Damaging` and `TargetsUnit` are both true. **Revert**: drop `rule.ID == 14` from
  `weaponSpellApply`'s cut, leaving `rule.ID == 25 || !spellApplicable(rule)`. **Failure**: `the
  victim is at 90, want 100 unchanged — the release was meant to be refused`.

**Measured against shipped content, mission 130 entity 55** (`M130_Veglud`, Humans row 211, an
`Elven Magic Wood Staff {castSpell=Stone_Curse:50}`, class 23, mana 447/447 — the round-3 review
also names the mercenary template `M_GoodFemale4` carrying the same staff at power 30, not
independently re-measured in this lane). Identical on EN and RU. Method: `cmd/weaponspellcheck`,
below — the actor is lifted into a fresh 2-actor world, ordered to attack a 100,000-HP dummy placed
at its own natural engagement distance, and run 240 ticks.

```
before (both roots): as shipped   victim hp=100000  attachedEffects=0  casts=0
                      control     victim hp=99670    attachedEffects=0  casts=0   (330 damage, unarmed-equivalent)
after  (both roots):  as shipped  victim hp=100000  attachedEffects=1  casts=15
                      control     victim hp=99670    attachedEffects=0  casts=0
```

Before the fix the staff-armed mage did nothing at all where the same actor stripped of the staff
dealt 330 damage over 240 ticks. After the fix the staff releases 15 times and the victim carries
Stone Curse's own absorption effect — zero HP damage is correct, since Stone Curse is not a damaging
row.

## R3-B2 — the item tooltip advertised damage the release refused

**What was wrong.** `WeaponSpellDamageFor` (`pkg/sim/spell.go`) refused on `!rule.Damaging ||
!rule.TargetsUnit` but carried no `rule.ID == 14` clause, while `releaseWeaponSpell` gained that
exclusion in this story. `pkg/game/iteminfo.go:52`'s own comment claims the two use "the same entity
fields, spell row and arithmetic `releaseWeaponSpell` uses" — untrue for id 14 before this fix: the
tooltip stated a damage interval for a Prismatic Spray staff that the live release never paid out.

**What was built.** `WeaponSpellDamageFor` now refuses `rule.ID == 14` identically to
`weaponSpellApply`. `pkg/game/iteminfo.go` needed no code change: `liveWeaponSpellDamage` and
`storedWeaponSpellDamage` both delegate to `WeaponSpellDamageFor`/`World.WeaponSpellDamage`, so its
own comment is true again once the shared function agrees with the release, exactly as instructed —
the fix makes the reader agree with the release rather than removing the release's own exclusion.

**Witness.** `TestWeaponSpellDamageForRefusesPrismaticSprayLikeTheRelease`
(`pkg/sim/weaponspell_test.go`): a Damaging, TargetsUnit id-14 row now answers `ok=false`. **Revert**:
drop `|| rule.ID == 14` from `WeaponSpellDamageFor`'s refusal. **Failure**: `WeaponSpellDamageFor =
10 + 20, true; want ok=false — id 14 is refused`.

**Measured against shipped content, mission 120 entity 84** (weapon spell 14, level 30, mana
402/402), one of the 28 placed actors across missions 120, 130, 131, 150 and 151 the review counted
for this mismatch. Same method as above:

```
as shipped (staff spell 14)   victim hp=100000  casts=0   (unchanged before/after: the release was already, and remains, refused)
control (staff spell cleared) victim hp=99776   casts=0   (224 damage: the physical blow alone)
```

The release side was already correct at id 14 before this round (R3-B1's fix does not touch the
id-14 cut); what changed is that the **tooltip** now reports the same refusal instead of a nonzero
interval. No shipped-content sweep of the tooltip's own text was run — that would need
`pkg/game`'s popup-building path over the mission's own inventory, which this lane's file ownership
does not include; the unit witness above and this release-side confirmation are what this lane
verifies.

**Ready-to-paste `spec.md` sentence** — replaces the existing bullet at spec.md:197-199, adding the
citation the review asked for and nothing else:

> - Prismatic Spray performs a complete ordinary application on every living actor in radius
>   `min(power / 20 + 2, 7)` and does not travel as a weapon spell: a weapon or item release refuses
>   it outright (`MAGIC-SING-019` (g)). The ordinary application refuses a target that is not alive,
>   on every arm and not only this one;

## R3-B3 — the fighter's rider arm

**What was wrong.** `weaponSpellFor` (`pkg/sim/spell.go`) requires `isMage(e)`, so only the caster's
replacement arm of `MAGIC-AUTOCAST-020` existed. The fighter's own rider — `MAGIC-ITEM-007`, active,
High: after a landed blow, `L05098` requires only that the weapon carry a spell, and the row's own
text states "a dead target still triggers it" for id 2 (Fire Ball) — had no implementation at all,
and no ledger row disclosed the gap.

**What was built — implemented, not disclosed-only.** The rider is built, reusing R3-B1's shared
`weaponSpellApply`:

- `weaponRiderSpellFor(spells []SpellRule, e Entity) (SpellRule, bool)` (`pkg/sim/spell.go`) is the
  exact negation of `weaponSpellFor`'s `isMage` gate — `L05093`'s own fighter predicate is
  `HERO-CLASS-020`'s negation of the mage test `weaponSpellFor` reads — so the caster's arm and the
  fighter's arm can never both claim one strike.
- `weaponRiderApply(ai, ti int, obs *castObs)` (`pkg/sim/spell.go`) asks almost none of
  `releaseWeaponSpell`'s own admission: the strike that reaches it already proved reach and sight,
  and `L05098` names no other gate. It does **not** ask the victim's liveness — "a dead target
  still triggers it" is the row's own text. `weaponSpellApply`'s Area branch already applies with no
  liveness test (Fire Ball is the one area row with no duration column, hence a blast, and
  `areaLandingRefusal` never refuses a blast); a point row reaching a dead target still self-refuses
  inside `ordinaryEffect`'s own general rule, since no shipped fighter weapon carries a non-damaging
  point row and this build claims no wider bypass than the evidence shows.
- `resolveBlow` (`pkg/sim/combat.go`) now takes `obs *castObs` and calls `weaponRiderApply(ai, ti,
  obs)` in its own tail, **after** the `dmg <= 0` return: the rider runs only once damage has
  actually landed, on the row's own "after the damage lands" text. `advanceAttack`'s one call site,
  and the two direct test call sites (`experiencepay_test.go`, `spelleffect1001_test.go`), were
  updated to pass `obs` (`nil` where no test reads it).

**What was disclosed rather than built: Control Spirit (id 25) at a weapon-borne release, both
arms.** `weaponSpellApply` refuses id 25 outright. Its own admitted arm inside `ordinaryEffect`
removes the target entity from the world and appends a new one; `castSpell` (the book-cast path)
rebinds both its own caster and victim indices after that swap for exactly this reason. A
weapon-borne release hands `weaponSpellApply` plain `ai`/`ti` indices that its own caller
(`advanceAttack`, `resolveBlow`) resumes using the instant the call returns; reproducing the
book-cast rebind inside a release risks corrupting that caller's own index for a row no finding in
this round, and no probe run in this lane, found carried by any shipped weapon or item. Recorded as
`DIV-056` below — this is the judgement call the brief named, made toward disclosure over an
unevidenced, index-unsafe build.

A second, smaller inference is disclosed as `DIV-057`: a weapon-borne Area row lands at the struck
target's own cell by direct analogy to DD-4's decoded unit-target book-cast convention. No claim
read for this round independently confirms the item-cast entry's own landing-coordinate or facing
choice for a weapon or item trigger specifically; the shipped Fire Ball measurement below confirms
the row **applies** and **reaches nearby actors**, which is what the analogy predicts, but not the
exact coordinate math underneath it.

**Witnesses.**

- `TestANoManaPoolFighterStrikesAndTakesTheWeaponsOwnRider`: a non-mage (`MaxMana: 0`) with a
  Damaging point row on its weapon takes a blow of 40 **and** the rider's own 12, landing the victim
  at 48. **Revert**: remove the `w.weaponRiderApply(ai, ti, obs)` call from `resolveBlow`'s tail.
  **Failure**: `the victim is at 60, want 48 — a blow of 40 and the weapon's own rider of 12` and
  `the victim carries no spell effect mark, want one`.
- `TestTheFightersRiderLandsAnAreaRowAtTheVictimsCellAndADeadTargetStillTriggersIt`: a fighter's
  blow fells its primary target outright (HP 5, blow 900); a **bystander** standing in the Fire
  Ball's own blast radius still takes area damage, proving the rider fired even though its own
  primary target died from the same blow. **Revert**: same removal as above. **Failure**: `the
  bystander's health did not move — the rider never fired on its own dead primary target`.
- `TestWeaponRiderSpellForRefusesAMage`: called directly (the companion integration test,
  `TestAMageNeverTakesTheFightersRiderThroughItsOwnAttack`, cannot witness this guard alone, because
  a mage never reaches `resolveBlow` through the ordinary tick at all — documented in both tests'
  own comments). **Revert**: drop `isMage(e)` from `weaponRiderSpellFor`. **Failure**:
  `weaponRiderSpellFor found a row for a mage, want none`.
- `TestTheFightersRiderTrainsNothing`: differential — the same survivable blow, with and without a
  weapon spell, must leave the attacker's `XPSlot` at the same level and experience either way (a
  landed physical blow trains that slot on its own, via `payExperience`'s unconditional
  `xpBlowBonus`, so an absolute "unchanged" assertion would conflate that pre-existing grant with
  the rider's own). **Revert**: append `w.awardSkill(ai, 0, (int64(rule.ManaCost)+1)/2, -1)` to
  `weaponRiderApply`'s own tail. **Failure**: `slot 4 is level 11, XP 1601 with the rider firing;
  level 11, XP 1595 without it — a rider must add no training beyond the ordinary blow's own`.
- `TestAnAbsorbedBlowNeverFiresTheRider`: a blow that lands (`AlwaysHits`) but is fully cancelled by
  `Absorption` must not fire the rider. **Revert**: move the `weaponRiderApply` call to immediately
  after the `Absorption` subtraction, ahead of the `dmg <= 0` return. **Failure**: `the victim is at
  88, want 100 unchanged — an absorbed-to-nothing blow must not fire the rider` and the accompanying
  mark check.
- `TestWeaponSpellApplyRefusesControlSpiritEvenOverAValidCorpse` (the `DIV-056` cut): a loaded ghost
  template and a corpse already in `DecayBones` — the one fixture
  `TestControlSpiritConsumesBonesAndCreatesANewOwnedGhost` (`spelleffect1001_test.go`) proves raises
  a ghost through the book-cast path — still refuses at `weaponSpellApply`, called directly.
  **Revert**: drop `rule.ID == 25` from `weaponSpellApply`'s cut. **Failure**: `weaponSpellApply
  applied id 25, want it refused` and `the corpse's own decay stage moved to 0, want it untouched at
  DecayBones`. The parallel subtest inside `TestFR10sFiveRefusalsLeaveTheVictimUntouched` ("the row
  is id 25, Control Spirit") stays **green** under this same revert — it is masked by
  `pointEffectRefusal`'s own "not a bones corpse" refusal on a living victim, which is why this
  dedicated test exists and why the in-file comment beside that subtest says so.

**Measured against shipped content**, the three placed `Boulder Thrower{castSpell=Fire_Ball:N}`
actors the review names: mission 90 entity 152 (level 70), mission 111 entity 0 and mission 140
entity 165 (level 40, same row twice). All `MaxMana 0`. Method as above, with the dummy placed at
the unit's own Reach (20) plus a short walk-in margin, so the siege unit engages from its own
natural stand-off distance rather than a forced adjacency — Fire Ball's Radius-1 blast would
otherwise land on the attacker's own cell too and the measurement would show the rider's
self-damage instead of its damage to the target.

```
mission 90 entity 152, EN:
  before: as shipped victim hp=99735  casts=0   |  control victim hp=99735  casts=0   (identical: no Fire Ball ever)
  after:  as shipped victim hp=99605  casts=4   |  control victim hp=99735  casts=0   (130 extra damage, 4 releases)

mission 111 entity 0 and mission 140 entity 165, EN (identical row and level):
  before: as shipped victim hp=99869  casts=0   |  control victim hp=99869  casts=0
  after:  as shipped victim hp=99800  casts=3   |  control victim hp=99869  casts=0   (69 extra damage, 3 releases)
```

Before the fix, "as shipped" and "control" (weapon spell cleared) are byte-for-byte identical in
every case measured: the Fire Ball enchant contributes nothing, exactly as the review found. After
the fix the rider fires and adds damage beyond the plain blow, and the attacker's own health stays
at its starting value in every case (250, 170, 170) — its own natural engagement distance keeps it
outside its own blast.

## Measurements: `cmd/weaponspellcheck`

A new developer tool, `cmd/weaponspellcheck/main.go`, reproduces the review's own method: lift one
placed actor out of its own mission world, set it beside a synthetic dummy on open ground, order an
attack, and run a fixed number of advances — once as shipped and once with `WeaponSpell` cleared
(the control). It reports the dummy's own final health, the attacker's own final health (to surface
self-splash from an area rider), how many effects stand attached to the dummy, and how many
weapon-borne casts the probe observed.

```
weaponspellcheck [-assets <root>] [-mission 130] [-entity 55] [-ticks 240]
```

It is registered in `internal/archtest`'s DAG allow-map (`cmd/weaponspellcheck: pkg/game,
pkg/mapload, pkg/sim`) and writes nothing to the repository; its output is a measurement over
converted game data and was never committed, per golden rule 1.

Two things a caller of this tool should know, both discovered while building it and left as
comments in `probe`'s own doc:

1. **A lifted actor's own `Owner` must be reassigned to `SelfSlot`.** The mission's own AI group
   engagement (`pkg/sim/engage.go`) re-decides any other owner's attack order on its own schedule
   and overrides a directly issued one within a handful of ticks, which would measure the AI's own
   targeting rather than the weapon-spell release. This is not a defect in `pkg/sim` — an AI-owned
   unit is supposed to have its orders re-decided — it is a precondition of isolating one actor's
   own release arithmetic.
2. **The dummy's placement must respect the attacker's own Reach**, not a fixed adjacency, or a
   large-Reach unit measured beside an area weapon spell shows its own self-splash damage instead of
   the damage it deals to its target (see the Boulder Thrower measurement above, where an early,
   adjacent-placement version of this tool showed the "as shipped" case dealing **less** total
   damage than the control — the attacker was standing inside its own Fire Ball blast).

## Divergence ledger rows

Allocated ids `DIV-056` and `DIV-057`. `DIV-058` is unused and returned.

| ID | Subsystem | Owner directive | ROM1 behaviour (claims) | Implemented behaviour | Type | Reason | Revisit condition | Status |
|---|---|---|---|---|---|---|---|---|
| DIV-056 | magic / weapon-borne release, Control Spirit | — | `MAGIC-ITEM-007` establishes the item-cast entry's general apply mechanism (`R0002`) with no exclusion named for id 25; `MAGIC-SING-019` (c) establishes Control Spirit's own admitted arm removes the target entity and constructs a new one, at Medium confidence for the raised actor's own statistics | A weapon or item release refuses Control Spirit (id 25) outright, on both the caster's replacement trigger and the fighter's rider trigger. Every other applicable row reaches the same ordinary or area apply a book cast does | FIDELITY-DEBT | Control Spirit's own admitted arm removes the target entity and appends a new one (`castSpell`'s own index rebind, `pkg/sim/spell.go`). A weapon-borne release hands its shared apply plain `ai`/`ti` indices a caller (`advanceAttack`, `resolveBlow`) resumes using the instant the call returns, and reproducing the book-cast rebind inside that call risks corrupting the caller's own index. No claim read for this round describes how, or whether, the original's item-cast entry performs an equivalent rebind across its own callers, and no shipped weapon or item carrying id 25 was found in this round's own probes (not an exhaustive campaign sweep) | A claim on the original item-cast entry's handling of a caller-held index across a raise, or a shipped weapon/item found to carry Control Spirit | OPEN |
| DIV-057 | magic / weapon-borne area release, landing point | — | `MAGIC-AUTOCAST-020` establishes the item-cast entry (`R0002`) is the same routine a book cast's own apply reaches; this build's own DD-4 (spec.md) establishes a unit-targeted **book** cast of an area row lands at the target's own cell. No claim read for this round independently confirms the item-cast entry's own area-landing coordinate or facing choice for a weapon or item trigger | A weapon-borne release of an Area row lands at the struck target's own cell, with the releasing actor's current facing fixed — DD-4's own convention, carried over by direct analogy since both paths reach the identical apply | UNKNOWN | The shipped Fire Ball measurement (Boulder Thrower, missions 90/111/140) confirms the row applies and reaches a nearby actor, which is what the analogy predicts, but not the exact coordinate/facing arithmetic underneath a weapon or item trigger specifically, which no claim read for this round states | A claim reading the item-cast entry's own area-landing argument construction | OPEN |

## Two pre-existing test assertions that encoded the bug

`TestFR10sFiveRefusalsLeaveTheVictimUntouched`'s "the row is not damaging" subtest, and its "the row
does not target a unit" subtest, encoded the pre-fix bug as intended behaviour. Both were rewritten:
the first now witnesses `spellApplicable`'s own refusal of a truly inapplicable row (renamed "an
inapplicable row"), and the second's assertion was wrong under the new design (`TargetsUnit` must
not gate a release) and was replaced by `TestTargetsUnitDoesNotGateAWeaponRelease`, a positive test
of the corrected behaviour.

`max32` (`pkg/sim/spell.go`), flagged by the review as dead code, is removed. `grep -rn max32
pkg/` before removal showed one match, its own declaration; none after.

## Gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files --cached --others --exclude-standard '*.go') \
  && go test -trimpath -count=1 ./...
(all packages ok, gofmt printed nothing)
$ bash scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
```
