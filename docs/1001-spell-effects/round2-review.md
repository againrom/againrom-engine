# 1001-spell-effects — consolidated adversarial review findings

Story `1001-spell-effects`, pushed at `5cee2da22d879246fb38752c95c244b911872cf3`
(branch `story/1001-spell-effects`, base `b51b439f09160968aa2b0dc115b820f63301ed3c`,
research pin `744214fe1f4b461cbd2d6766390b850ab15779ab` = research `master`).

Four independent adversarial reviewers ran at that sha under the pipeline v2 gate, with four
lenses: point effects, area effects, client and persistence, actions/AI/regressions. **All four
returned INCOMPLETE.** The story returns to its lane.

## How to read this file

Every location below is a **premise, not a verdict**. Each item states the evidence that produced
it. **Verify that evidence first, then build on it.** Two reviewers reached item A1 independently
by different routes, and the seat then read the code itself; items A1, A4, B1 and A2 were
re-verified by the seat at the file and line given. Everything else is a reviewer's reading.

Where a research claim is cited, re-read the claim in the pinned research tree
(`research/claims/*.md`, or `go run ./tools/claim <ID>` in `<seat>\research`)
before acting. Research is the only authority on ROM1 behaviour. againrom code is never evidence
of ROM1 behaviour.

Measurements quoted below were produced by the reviewers with out-of-tree probe modules and
temporary overlays. Reproduce a measurement before you rely on its exact numbers.

---

## A. Player-visible regressions in behaviour that worked before this story

### A1. A unit that is busy silently ignores a move order. Retreat and repositioning are gone.

**Seat-verified.** Two reviewers found this independently; the seat read the code.

- `pkg/sim/step.go:416` — `case KindMoveTo` now reads
  `if !e.Alive() || w.actorActionBusy(i) { continue }`. It was `if !e.Alive()`.
  The comment immediately below it still reads *"AND IT ENDS WHATEVER FIGHT THIS UNIT WAS IN"*,
  which is now unreachable for a busy actor.
- `pkg/sim/spell.go:1043-1045` — `actorActionBusy` returns true for `HasAttackTarget && HasTarget`
  (a unit merely *approaching* its victim), and at `:1049` for
  `AttackPhase != AttackReady || AttackCountdown != 0 || CastWait != 0`, which is most of the
  melee cycle.
- `pkg/sim/group.go:124` — `groupOrder` drops busy members before they enter `members`.
- `pkg/sim/group.go:245-252` — `issueGroupDestination` filters again, and then computes the
  centroid and formation rate from the **filtered** slice, so the surviving members' destinations
  and speeds differ from what the same order produced on master.
- `pkg/sim/engage.go:1315-1322` — `orderAttack` returns early for **every** actor when busy, not
  only a mage, so clicking a different enemy does not retarget a fighting unit either.

Neither `contract.md` nor `spec.md` lists movement among the guard's sharers. `contract.md:40`
names *"attack commands, weapon-spell attacks, manual casts, AI casts, offensive autocast, and
idle Heal"*. `spec.md:101` says *"An active scripted or player order remains attached"*.

Measured, mission 10, EN root, identical scenario on both builds: order p0 to attack `u30`, wait
60 ticks, order p0 to move back to its start. On master p0 is at (17,66) 120 ticks later — the
recall was obeyed. On `5cee2da` p0 is at (14,56), still walking to `u30`.

Measured through the real right-click seam (`App.step` → `Viewer.command` → `mw.enqueue` →
`queueGroup(KindGroupMoveTo)`) in the owner's mission-40 save: a busy actor and an idle actor
given the same click on the same tick — the busy one takes no destination, the idle one does.
Synthetic: the order is admitted on 2 of 40 consecutive ticks; on an approaching unit, 0 of 60.

There is no `KindStop` in the tree, so `KindMoveTo` and the group move were the only ways to
break off a fight.

`KindGroupStance` and `KindGroupPatrolTo` take the same filtered `members` slice, so Guard,
Stand-Ground and Patrol are dropped on the same actors. `pkg/sim/script.go:1985`
(`cmdGroupCommandedMove`, script group sub-commands 4 and 5) hands `groupLivingMembers` to the
same `issueGroupDestination`, so a shipped trigger that marches a group leaves its fighting
members behind.

**Reversals recorded only in tests, not in `spec.md`:**

`pkg/sim/pursuit_test.go:141` — `TestAMoveOrderStillEndsAnApproach` became
`TestAMoveOrderCannotReplaceAnActiveApproach`; the assertion flipped from "the move order's own
cell" to "the victim's cell".

`pkg/sim/combat_test.go:250` — `TestWalkingAndAttackingAreOneState`: `if !e.HasTarget { "the move
order was not taken" }` became `if e.HasTarget || !e.HasAttackTarget || e.AttackTarget != 2`.

`pkg/sim/combat_test.go:262` — `TestAGroupMoveOrderEndsAFight` became
`TestAGroupMoveOrderCannotInterruptAFight`.

`pkg/sim/commanded_test.go:104` — the AI mid-pursuit rescoring assertion flipped from "want the
nearer candidate (3)" to "want its admitted victim 2 until the action ends".

These reverse `0064` FR-1 and FR-2. A reversal belongs in `spec.md` with its authority before it
belongs in a test.

### A2. Slow and Freezing Cloud together make a unit about 21× faster.

**Seat-verified at the three cited lines.**

- `pkg/sim/effect.go:57` — `case EffectSpeed: e.Speed += amount`, with no lower clamp. Every other
  kind in that switch is clamped.
- `pkg/sim/world.go:1031` — `func rated(e Entity) bool { return moverSpeed(e) > 0 }`.
- `pkg/sim/step.go:836` — only a rated mover accrues transit; an unrated one keeps the
  cell-per-tick cadence, which is the fastest rate the engine has.

Measured, same world, 5-cell walk: speed 10 → 105 ticks, speed 8 → 129, speed 1 → did not arrive
in 400, **speed 0 → 5 ticks, speed −4 → 5 ticks**.

Reachable on shipped rows: Slow (28) and Freezing Cloud's inner effect (7) are different spell
ids, so non-stacking does not merge them. Both attach `EffectSpeed` at `−(power/15 + 1)`.
Measured on a speed-10 actor through the real command path: at power 60, `10 → 5 → 0`; at power
100, `10 → 3 → −4`. The shipped `Units` collection has four speed-8 rows, which cross zero at
power 45.

Related, by inspection and not executed: `pkg/sim/group.go:661-669` `groupMinSpeed` converts
through `int16` and back to `uint8`, so a member at `Speed = −4` yields `GroupSpeed = 252`, which
`moverSpeed` then returns for every member of the ordered group.

What ROM1 does with a negative speed is **not decoded**. `HERO-SPEED-008` establishes only that
the effective speed reaches the mover as a byte. So the floor is an authored answer and owes a
typed ledger row, not a research hold.

### A3. Control Spirit creates an actor with no class, so it is drawn as nothing and has no name.

- `pkg/sim/spell.go:239-250` — the `ghost := Entity{...}` literal never sets `Class`, `TypeID`,
  `DyingTime`, `XPValue` or `GainsXP`.
- `pkg/game/world.go:3673` — `c := classes[e.Class]` is how a world unit's art and name resolve.
- Measured against the EN install: `unit classes loaded=34; class id 0 present=false`. No panic
  (`TierFrames` guards nil), so the unit is simply drawn as nothing with an empty name.

`MAGIC-SING-019` (c) states the original constructs the new actor from the `Data.bin` template
named by the literal at `L05127` — `Ghost` — copying only `reaction/2 + 1`, Mind, Spirit,
`healthMax/2`, `+0xa6` and `+0xbe`. This build resolves no template and copies eleven further
fields instead. There is no ledger row for that.

### A4. Wall of Fire paints a 25-cell diamond instead of a wall.

**Seat-verified.** `pkg/sim/celleffect.go:316` forks on `if rule.ID == 19`, so only Wall of Earth
gets wall geometry. Wall of Fire is id 3.

`MAGIC-AREACELL-039` puts `Distribution system == 4` — **`wall_of_fire` and `wall_of_earth`** —
through `R1066` and the two wall tables. The seat re-read the claim and confirms the
wording. Measured on the shipped rows: Wall of Fire 25 cells (Manhattan diamond of radius 3),
Wall of Earth 10. The record's `Direction` is computed and serialized for spell 3 and never read:
three different bearings produce byte-identical cell lists.

Shipped-reachable through three instant-21 nodes (`101.alm` ×1, `131.alm` ×2) and every player or
AI cast of the row. The story's own witness (`pkg/sim/effectwitness.go:69`) casts spell 3, but
both of its victim cells fall inside both shapes, so it cannot discriminate.

### A5. A staff carrier refuses an attack order outright where a plain fighter approaches.

`pkg/sim/engage.go:1301-1310` — for a weapon-spell carrier, `orderAttack` returns before writing
anything when `!w.actorSeesEntity(i, ti)`. Measured with `ScanRange = 3` and a victim 20 cells
away: the plain fighter takes the order and walks; the staff carrier takes nothing and never
moves. `actorSeesEntity` is the actor's own sight march, not the player's fog union, so a
staff-armed hero ignores a click on any enemy the player can see through another unit.

`spec.md:67-70` scopes actor-local perception to **cast** admission and release. It does not say a
weapon-spell carrier stops approaching.

---

## B. Canonical state corruption and unloadable saves

### B1. Bless's magnitude is written into the Astral protection slot on every recompute.

**Seat-verified.** `pkg/sim/rearm.go:214`:

```go
e.Protection[k] = c.Protection[k] + w.effectDelta(id, EffectProtectionFire+EffectKind(k))
```

`Protection` is `[5]int32` (`pkg/sim/world.go:674`) and the kind constants run
`EffectProtectionFire … EffectProtectionEarth` then `EffectBless` (`pkg/sim/spell.go:19-25`). At
`k = 4` the expression asks for `EffectBless`. A blessed actor's Bless magnitude —
`(power*4)/5 + 20`, so 20 to 100 — lands in `Protection[4]`.

Measured: blessed target `[0 0 0 0 0]` → after `SetCombat` → `[0 0 0 0 100]`. This is hashed,
serialized state (`pkg/sim/binary.go:1593`) and it is displayed as the Astral protection row
(`pkg/ui/panel.go:912`). Bless's removal is a no-op for it, so the value persists until the next
recompute after expiry.

### B2. Effect apply and expire are not symmetric under the clamps, so an effect permanently
rewrites base state.

`pkg/sim/effect.go:69-78` clamps `Protection` into `[0,100]` on apply; `effect.go:83-85`
(`removeAttachedAt`) subtracts the full stored magnitude on expiry with no clamp memory.

Measured: base fire protection 80, a `power/2 = 45` Protection effect → clamped to 100 → on expiry
→ **70**. Ten points of base protection destroyed permanently. `ScanRange` (`effect.go:58-66`) has
the mirror fault: a Darkness clamped at 0 restores the whole magnitude, so a low-sight victim can
end with more scan range than it started with.

`spec.md:118` states expiry merely *"unapplies a non-continuous magnitude"*.

### B3. A save taken while a staged area effect stands cannot be loaded.

`pkg/sim/celleffect.go:444-508` produces ring stage cells in the decoded visit order and
deliberately does not sort them (its own comment says the order is significant). That list is
stored on the standing record at `celleffect.go:311` and `celleffect.go:203`.
`pkg/sim/castbinary.go:222-224` refuses a decoded record whose cell list is not strictly
ascending.

Measured: Fire Sacrifice at its landing tick produces
`[5395 5139 4883 5396 4884 5397 5141 4885]`, and `UnmarshalBinary` returns
*"sim: area effect 0 cell list is not strictly ascending"* for ticks 0, 1 and 2 of its four-tick
life. Acid Stream stage 1 is refused for directions 1, 4 and 7.

`MarshalBinary` succeeds, `pkg/game/save.go:264` writes the bytes, `pkg/game/resume.go:271` fails
to read them back. `TestAttachedAndAreaStateRoundTripByteIdentically`
(`spelleffect1001_test.go:353`) round-trips only a Wall of Earth record, whose cells are ascending
by construction.

### B4. A Wall of Earth laid across a stored route makes the world unloadable.

`celleffect.go:341-343` sets `blockMagicWall` on the grid; `world.go:1302-1310` puts that bit in
`Domain.blocks()` for ground and ghost; `pkg/sim/binary.go:2498` refuses a decoded route cell the
entity's own domain cannot cross. Nothing in the area landing path invalidates stored routes, and
`subGoal` (`step.go:941-972`) does not re-check the grid.

Measured: a 33-cell route, a wall landed on route cell (20,20) → *"sim: entity record 0: route
cell 16 is (20,20), which its domain 0 cannot cross"*. The refusal's own comment
(`binary.go:2452`) says *"The five refusals are the states a tick cannot produce"*. A tick can now
produce this one.

23 shipped instant-21 nodes cast Wall of Earth (`101.alm` seals four exits around (40,41);
`131.alm` a corridor at (110,89)/(110,91); `140.alm` (121,54)/(130,64)), with 22 matching
instant-29 nodes retiming them.

---

## C. The cost and refusal contract

### C1. A book cast pays its mana for a release whose ordinary apply then refuses.

`spec.md:59-60`: *"Failure removes the pending cast with no cost or recovery."* `spec.md:214-216`:
*"A refusal, a release-time cancellation and an effect that cannot apply spend no mana, start no
recovery and award no training."*

`pkg/sim/spell.go:668` pays, then `:686-688` returns after `ordinaryEffect` refuses. Same shape in
the cell form at `:773` and `:780-782`.

Two shipped rows reach it. `ordinaryEffect`'s Control Spirit arm requires `Decay == DecayBones`
(`spell.go:224`) but the admission predicate checks only liveness (`spell.go:1128`,
`if !target.Alive() && rule.ID != 25`). Teleport's unit-target arm requires `terrainOpen`
(`spell.go:216`), also unchecked at admission. Measured: Control Spirit at a **fallen** corpse —
admission refusal `""`, mana 5000 → 4970, entity count unchanged; Teleport onto a blocked cell —
refusal `""`, mana 5000 → 4940, caster does not move. A corpse spends its whole dwell at
`DecayFallen`, so casting Control Spirit on a freshly killed body is the ordinary case.

`BookSpellRefusal` — the read-only instrument `cmd/savecheck` and the closure rely on — reports
both as admissible.

A duration row whose computed duration is 0 reaches the same shape (`effect.go:125`; measured: 7
mana spent, nothing attached).

### C2. The training source and diplomacy gates were removed at every cast call site while
`spec.md` still claims they apply.

`spec.md:220`: *"The existing skill sink applies its human, mage, source, diplomacy, level,
participant and cap gates."*

The diff changes the source index from the victim to `-1` at `pkg/sim/spell.go:699`, `:740`,
`:788` and `pkg/sim/celleffect.go:682`, `:710`. `awardSkill` (`pkg/sim/skill.go:153-161`) applies
the same-owner and locked-relation refusals **only when `srcIdx >= 0`**, so both gates are dead on
the book-cast path.

Measured: a heal cast on a same-owner ally raises school 2 from 10 to 11 — on master that award
was refused (`0135` FR-5.2). An idle mage beside a permanently wounded own-team ally, with no
command at any tick, rose from school level 0 to level 4 in 2000 ticks, each rise firing the full
`SetDerived` recompute.

The now-dead fixture comment survives at `pkg/sim/skill_test.go:519`.

### C3. Prismatic Spray applies to dead entities and trains per corpse.

`pkg/sim/celleffect.go:705-712` walks every entity in radius with no `Alive()` test, so each
corpse in range takes a damage roll — consuming world RNG — and awards the caster
`(manaCost+1)/2` school experience. The area collector skips the dead (`celleffect.go:649`).

---

## D. Decoded area geometry and cadence

### D1. The diagonal wall footprint is not the shipped table B.

`MAGIC-AREACELL-039` gives table B as *"9 as a doubled anti-diagonal"*; the table values are in
`research/experiments/EXP-0172-area-tick/evidence/wall-tables-and-switches.txt`:
`[(-2,2),(-1,1),(0,0),(1,-1),(2,-2),(-1,2),(0,1),(1,0),(2,-1)]` — a 5-long anti-diagonal plus the
same line shifted by (+1,0), a 2-thick band spanning ±2.

`pkg/sim/celleffect.go:520-524` draws a 1-thick line spanning ±4:
`[(-4,-4) (-3,-3) (-2,-2) (-1,-1) (0,0) (1,1) (2,2) (3,3) (4,4)]`. All four diagonal octants
mismatch. **The four axis octants match table A exactly**, including which side the second row
falls on, which fixes the octant convention and makes the diagonal mismatch unambiguous. The cell
count (9) coincides.

Eight shipped diagonal wall nodes (`101.alm` ×5 including one Wall of Fire, `140.alm` ×3).

### D2. Cloud diamonds paint four cells the decoded walk cannot reach.

`MAGIC-AREACELL-039`, High for *"every loop bound, every filter"*: `R1067` walks `dx, dy`
over `[-r, +r]` and filters on `abs(dx) + abs(dy) <= r + 1`. The filter reaches `r+1` but the loop
bound is `r`, so the four axis tips are unreachable.

`pkg/sim/celleffect.go:411-421` is called with `r+1` (`celleffect.go:320`), so it walks
`[-(r+1), r+1]` and includes them. Measured: radius 1 → 13 built vs 9 decoded; radius 2 → 25 vs
21; radius 4 → 61 vs 57. Freezing Cloud is 44% larger than decoded.

### D3. The cloud pulse phase is taken from elapsed ticks, not from the counter.

`MAGIC-AREAPULSE-037` (High, amended): `+0x4c` starts at
`V0 = (AreaDuration<<4) + (power<<4)/10`, decrements, and pulses when the **new value** is a
positive multiple of 16. `pkg/sim/celleffect.go:190-194` pulses when elapsed ticks are a multiple
of 16. The two agree only when `V0 ≡ 0 (mod 16)`.

Measured: power 5 → first pulse at tick 16, decoded 8; power 7 → 16 vs 11. The pulse **count** is
the same either way; only the phase moves, by up to 15 ticks. Both fixtures pick a power that
hides it — the unit test uses power 0 (`spelleffect1001_test.go:188`) and the mission-91 witness
uses Mind 100 → power 70 → `V0 = 352 ≡ 0 (mod 16)` (`effectwitness.go:131`).

### D4. The occupant-slot walk counts bodies and is not keyed by domain.

`pkg/sim/celleffect.go:639-651` counts every covering entity toward the three-slot cap **before**
the liveness test. Measured: three dead entities and one living on one cell, Fire Ball blast — the
living actor's HP is unchanged.

`TERR-CELLREC-146` (High) says the slots are keyed by domain — `+0x4` one actor of movement domain
1 or 2, `+0x8` one actor of domain 3, `+0xc` a structure, `+0x10` a sack — not "the first three by
id". `MAGIC-AREACELL-039` additionally says the **cloud pulse reads only `+0x4`**, while ring and
blast read all three. This build walks up to three slots in all modes.

`TestAreaApplicationWalksOnlyTheThreeCellOccupantSlots` (`spelleffect1001_test.go:541`) cements
the wrong rule with four **living** entities on one cell, a configuration the occupancy model
forbids.

### D5. Wall of Earth's occupied-cell refusal reads the anchor cell only.

`MAGIC-WALLEARTH-042`: the per-cell add skips the cell when occupant slot `+0x4` is non-null.
`TERR-FOOTPRINT-147` (High): an actor's pointer is in **every** cell of its `n × n` footprint.
`pkg/sim/celleffect.go:536-552` tests `e.X == x && e.Y == y` only. Measured: a wall cast beside a
live 3×3 ground unit painted six cells inside that unit's own footprint. It also tests `e.Alive()`,
so a downed body — which keeps its cell — does not refuse the cell.

---

## E. The customisation seam (G2), which `contract.md:66` claims explicitly

### E1. Area mode is a hard-coded spell-id switch; the decoded selector columns are unread.

`MAGIC-AREATICK-036` gives the selector: `Distribution system == 5` → ring, else
`Area Effect Duaration > 0` → cloud, else blast. `pkg/sim/celleffect.go:227-236` hard-codes
`2 → blast`, `{4,9,21} → ring`, else cloud. On shipped rows the two agree; editing the two
selector columns changes nothing.

`SpellRule.Distribution` is parsed (`pkg/mapload/spell.go:50`), serialized
(`pkg/sim/binary.go:1746`, `:2912`) and read by nothing in `pkg/sim` — a canonical field with no
consumer. `areaLife`'s `rule.ID == 21` branch (`celleffect.go:239-241`) is also unreachable, since
ring mode overwrites `Remaining` at `celleffect.go:309`.

### E2. Stone Curse's magnitude is hard-coded to 5.

`pkg/sim/spell.go:279-280` writes `mag = 5`, discarding `rule.EffectMagnitude`.
`MAGIC-EFFECT-015` is explicit that Stone Curse is one of the two rows where the column's number
**is** the magnitude (`absorbtion=+5`).

### E3. The untimed test is `& 3` where the claim gives `& 7`.

`pkg/sim/spell.go:291` uses `mode&(EffectDuration|EffectContinuous) == 0`; `MAGIC-ATTACH-016`
reads the original's gate as `+0x3d & 7 == 0`, which also covers `charges` (4). No shipped row
uses `charges`.

---

## F. Spec-versus-code, and documents

### F1. Invisibility is cancelled by a landed blow, not by the attack approach.

`spec.md:138-140` says it is removed *"by the owner's next cast at another actor or by the owner's
physical attack approach"*; `MAGIC-SING-019` (f) names `R0245`, the approach. The only two
removal sites are `pkg/sim/spell.go:670` (the cast, correct) and `pkg/sim/combat.go:516`, which
sits **after** the to-hit gate at `combat.go:512-514`. An invisible attacker that misses stays
invisible.

### F2. The continuous cadence is unreachable on shipped data. **Suspicion, not proven.**

`pkg/sim/effect.go:168-176` decrements first and then tests `e.Remaining % 8 == 0`. Poison Cloud
is the only shipped `continuous` row and ships `duration 8`, so the remainder is 7 after the first
decrement and the eighth-tick re-application never fires. Whether ROM1 applies twice depends on
the order of its `AND 7` against the decrement, which `MAGIC-ATTACH-016` does not pin. **Research
question first, code change second.**

### F3. A duration-mode health effect can fell an actor without clearing its order.

`pkg/sim/effect.go:83-85` reverses the magnitude on expiry with no `clearFelled` call, unlike
attach (`effect.go:155`) and the continuous tick (`effect.go:174`). Unreachable on shipped data
(the only `EffectHealth` row is continuous, whose reversal is skipped), so this is a latent
customisation-seam hole.

### F4. Further modified assertions, for the record

- `pkg/sim/spell_test.go:384` — `Step(w, []Command{cast, cast})` became `spRunCast(w, cast)`. Two
  casts in one command slice is no longer exercised anywhere, yet the failure message still reads
  *"after two casts in one slice"*. The `released[e.ID]` guard at `step.go:522` is unwitnessed on
  that path.
- `pkg/sim/combat_test.go:705` — `TestReIssuingAnOrderDoesNotResetTheCycle`: the 40-tick
  alternating-victim loop was replaced with two `Step`s. The "a different victim every tick never
  lands a blow" half of `0064` FR-1 is gone.
- `pkg/render/terrain/overlay_test.go:1108` — `…StaysInsideItsOwnCell` became
  `…StaysAboveItsOwnCell`; `if !ground.In(own)` became `if ground.Max.Y > own.Min.Y`. The old
  comment named what is being given up: *"nor reach into a neighbour's cell, which is what would
  make one unit's bar look like another's."* Bars now sit 8 px above the cell top, inside the
  northern neighbour's cell. `pkg/ui/fogindicators_test.go:32` moved its fixture from (0,0) to
  (0,1) **to dodge the new geometry**.
- `pkg/sim/autocast_test.go:114` — "most hurt" became "nearest". Owner-directed and stated at
  `spec.md:99`; it reverses `0154` AC-8 and should be recorded as such.
- `pkg/sim/heal_test.go:135` — full-health target takes no cast. Owner-directed
  (`contract.md:42`), correctly recorded.
- `pkg/sim/weaponspell_test.go:660` — `TestAReleasedCastTrainsTheStaffsOwnSchoolAndNoOther` became
  `TestAWeaponBorneReleaseTrainsNothing`. **Checked against the pin: `MAGIC-TRAIN-018` (High)
  gates training on "`caster+0x68 == 0`, not an item cast". This reversal is research-backed** and
  is a correction of `0139` FR-16, not a weakening.
- `pkg/sim/skill_test.go:514` — fixture damage widened 4–8 → 30–60 so a new assertion becomes
  observable; the original level assertion is intact.
- `scenarios/0154-synthetic-spells.json` — `p1` hp 20 → 54 and the heal assertion 30 → 59, a
  fixture retune around the new idle-Heal timing.

### F5. Document defects

- `closure.md` §"Mission-40 mercenary dolls" attributes entities 36/37/38 to mission 40.
  Reproducing its own commands shows those actors and those four digests come from `game0020.sav`,
  which the same document decodes as **mission 41**. The digests are correct; the heading is not.
- `pkg/sim/scriptcast.go:226-231` still documents that *"Every one of the 20 shipped instant-24
  nodes therefore resolves here and changes nothing"* — stale after this story wired protections,
  Stone Curse and Bless into the ordinary path.
- `pkg/sim/binary.go`'s version comment describes versions through 52 and says nothing about 53.
  The standing rule is that the constant's own comment is the authority for the live version.
  **The seat will write this paragraph at the landing; lanes leave it alone.**
- **A `CONFLICT` ledger row is owed** regardless of which side is right: `MAGIC-RESIST-006` reads
  the jump table at `L04134` as `1→Fire, 2→Water, 3→Air, 4→Earth, 5→Astral`, while
  `HERO-DMG2-029` (restated by `UNIT-COMBAT-006`) reads the same table with schools 2 and 4
  swapped. `pkg/sim/spell.go:822` implements the first. `pkg/ui/panel.go:176-181` already carries a
  comment warning about this.
- Canonical save version 50 → 53 with no migration: the owner's existing `builds/current` saves
  will fail to load after this lands. The failure is graceful and the policy is deliberate, but
  `closure.md` does not state the consequence.

---

## G. AI and group coupling

### G1. A casting member is removed from its group's decision.

`pkg/sim/engage.go:196-204` filters `g.members` to non-casters before `w.decide(g)`. `decide`
reads `g.members` for the guard centroid (`engage.go:494`), and `candidates` reads it for the
decider (`engage.go:814`) and for the shared group sight stamp (`engage.go:815`). A group whose
long-sighted mage begins a cast loses that mage's vision for the same pass and its guard notice
circle moves. If every member casts, `decide` is skipped entirely.

### G2. Two shipped missions got worse under the unattended drive.

`scripts/campaign-sweep.sh` over all 28 EN campaign maps, master versus story. The script-gap
census is unchanged at 59 unsupported / 7375 reached, and every map loads and drives.

| mission | master alive/fallen | story alive/fallen |
|---|---|---|
| 80 | 106 / 1 | 105 / 2 |
| 90 | 153 / 1 | 150 / 4 |

### G3. The guard is not universal.

`armSwarm` (`engage.go:691-692`) and `escortClose` (`escort.go:143-150`) still call `clearAttack()`
and overwrite the destination of an actor mid-cycle. The AI and the escort arm can interrupt an
action the player cannot. **Suspicion — mechanism identified, no failing scenario produced.**

### G4. Idle Heal crosses ownership freely.

Measured: an owner-1 mage healed a wounded owner-9 actor with no relation set at all.
`spec.md:99` sanctions the neutral tier, so this is a scope question rather than a defect, but in
a campaign mission it spends party mana on any wounded non-hostile third party in range.

---

## What the reviewers confirmed as genuinely delivered

Recorded so the fix round does not re-litigate settled ground.

- All 18 shipped point rows land canonical state through the real command path at power 0 and
  power 100. Every arm's magnitude formula matches `MAGIC-EFFECT-015` instruction for instruction.
  Duration sourcing is faithful, including the split between `lastingTicks` and the effect
  record's own count. Stone Curse's second `Effects` record is correctly **not** applied.
- The resistance expression reproduces `MAGIC-RESIST-006` exactly. Bless/Curse annihilation,
  same-id non-stacking, continuous refresh-only and the >9600 countdown skip match
  `MAGIC-ATTACH-016`. Consumers exist and are real for Absorption, Bless/Curse, Invisibility,
  ScanRange and `Protection[0..3]`.
- Axis wall geometry matches shipped table A byte for byte in all four axis octants. Fire Ball
  footprint division matches `MAGIC-FIREDIV-047`. Layer conflicts match the dispatch table exactly.
  Ring cadence runs at ticks 0, 3, 6 with terminal removal. Meteor Storm's 32 stages, two draws per
  stage, `[-2,3]` range, byte truncation and the `8 ≤ c ≤ dim-9` gate are all correct. The Acid
  Stream orientation transform table matches `MAGIC-RING-048` in all eight orientations.
- Wall of Earth passability is a domain mask with no spell-id branch in route search, and
  overlapping walls restore every bit on expiry.
- Script casts correctly bypass actor perception; no shipped script cast is disabled by the
  visibility gate.
- The right-button autocast route is the real per-frame input path, end to end. The dashed border
  is canonical presence plus the real ambient clock. HUD bars use the real camera, relief,
  footprint and fog transform. Point and area feedback are simulation-driven with no client timer.
  One Heal burst per positive restore, from the real sheet, with the travelling arm suppressed.
- The popup has exactly one formula, in `pkg/sim/spell.go`, with no client cache and no second
  copy anywhere in `pkg/ui` or `pkg/game`.
- Format 53 round-trips byte-identically and hash-identically in all five states tested, all 50
  entity fields are written and read, the record is 260 bytes with Reaction at +252 and Spirit at
  +256, and the five new writes are each witnessed by a mutation test.
- Original-save restoration is generic: the selector is `PC_` prefixed row names, the stable id is
  derived, 10575 appears only in a fixture, and no `Brian` literal exists in production code. The
  doll compositor is slot- and code-driven with no name-based fallback; all four digests reproduce
  exactly on both roots.
- `SetDerived` preserves current pools and re-applies active effect deltas to the fresh base, so a
  level rise does not refill.

---

## Amendment, 2026-08-15 — finding C2 withdrawn by owner ruling

The owner ruled that an idle mage healing a wounded ally without an order, and training its own
school from those casts, is the **intended default**. Finding C2 above is therefore not a defect.
The source and diplomacy gates are not restored on the book-cast path. Auto-heal becomes an
in-game setting in a later story; `1001` does not build it.

What C2 still owes is bookkeeping: `spec.md:220`'s claim that the skill sink applies its source
and diplomacy gates to a book cast is false about the built behaviour and is corrected at the
landing, with a typed row in `docs/DIVERGENCES.md` carrying the ruling as its reason. The stale
fixture comment at `pkg/sim/skill_test.go:519` names a gate that no longer applies on this path.

The ruling is recorded in full in `pipeline/OWNER-RULINGS.md`, 2026-08-15. Every other finding in
this document stands; the owner separately confirmed the order-ignoring regression.

## Amendment, 2026-08-15 — review premise E4 is refuted by the fix round

The AREA lane refuted the second half of finding E1/E4. The review stated that `areaLife`'s
`rule.ID == 21` branch is unreachable because ring mode overwrites `Remaining`. That is true of the
simulation path and false of the whole program: `SpellCharacteristicsFor` in `pkg/sim/spell.go`
calls `areaLife` for every row with `rule.Area`, so the branch was reached by the spellbook popup
and reported Meteor Storm an invented area lifetime of `(Radius+2)*2 = 12` where the decoded value
is 94.

The finding was therefore right that the branch is wrong and wrong about why it mattered. Recorded
as a refutation rather than corrected in place: the reviewer's reasoning was sound over the
population it examined, and the branch had a second consumer that population did not contain.

The seat verified the lane's own witness independently on both installed roots:
`go run ./cmd/spelleffectcheck -mission 91` reports `outside-wall=1000/1000` on `en` and `ru`, from
an actor placed inside the 25-cell diamond and outside the 10-cell wall table. That actor is the
instrument the round-1 witness lacked; the review's A4 noted that the story's own witness could not
discriminate the two shapes.

## Amendment, 2026-08-15 — a gate that cannot run looks like one that passed

`scripts/check-no-game-assets.sh` needs `git` on `PATH`. Invoked with `PATH=/usr/bin:/bin` alone it
exits 127 and prints nothing about assets. Round 1's `closure.md` records the guard as PASS and
states that it was run *"using Git's installed Bash with `/usr/bin:/bin` on `PATH`"*. If that is
literally what was run, the recorded PASS is an exit-127 that was read as a pass.

The seat's own round-1 gate run does not share the defect: it used `PATH=/usr/bin:/bin:$PATH`, which
keeps `git` reachable, and it printed `check-no-game-assets: clean (tree scan)` with exit 0. The
guard's result is therefore verified for `5cee2da` on this machine regardless of what the closure
describes. The closure sentence is corrected at the landing.
