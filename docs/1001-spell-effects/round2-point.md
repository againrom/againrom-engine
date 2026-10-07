# 1001-spell-effects — fix round 2, point-effect lane

Branch `story/1001-r2-point`, based on `5cee2da22d879246fb38752c95c244b911872cf3`.

This lane owns review findings A2, A3, B1, B2, C1, C2, C3, E2, E3, F1, F2 and F3, plus two items
the seat added mid-round: the character sheet's protection rows and the removal of the Teleport
autocast.

Every fix below has a witness that fails at `5cee2da` and passes on this branch. Each was verified
by reverting the production line and rerunning the named test; the reverts were undone afterwards
and the full suite is green.

Reproduce the whole set with:

```
GOCACHE=<seat>/.gocache-impl1001 go test -trimpath -count=1 ./...
```

## What was wrong and what changed

### A2. Speed had no lower bound, so two slowing effects made a unit faster

`pkg/sim/effect.go:57` added the magnitude to `Entity.Speed` with no clamp. `moverSpeed` returning
zero or less makes `rated` false, and an unrated mover keeps the cell-per-tick cadence, which is
the fastest rate the engine has. Slow (28) and Freezing Cloud's inner effect (7) are different
spell ids, so same-id non-stacking does not merge them, and both attach `EffectSpeed` at
`-(power/15 + 1)`.

`applyEffectDelta` now floors the result at `minEffectSpeed = 1`.

The floor is **authored**. `HERO-SPEED-008` establishes the speed expression and that the result
reaches the mover as a byte at `[actor+0x154]+0xa`; what the original does when an arm drives that
number below zero is not decoded. One is chosen rather than zero because zero is exactly the value
`rated` reads as "no rate at all". A ledger row is proposed below.

`groupMinSpeed` (`pkg/sim/group.go:661-669`, not this lane's file) converts through `int16` and
back to `uint8`, so a member at `Speed = -4` would have produced `GroupSpeed = 252` for the whole
group. With the floor in place `Entity.Speed` can no longer be negative from an effect, so that
conversion is no longer reachable through the spell path. It remains reachable by any future writer
of `Speed` that does not go through `applyEffectDelta`; that is the orders lane's file and is
recorded here rather than changed.

Witnesses: `TestTwoSlowingEffectsLeaveAnActorSlowRatherThanUnrated`,
`TestASlowedActorGetsExactlyItsSpeedBack` (`pkg/sim/spellpointfix1001_test.go`). Nothing exercised
Slow before this round.

### A3. Control Spirit raised an actor with no class

The `ghost := Entity{...}` literal set no `Class`, `TypeID`, `DyingTime`, `XPValue` or `GainsXP`,
and copied eleven further fields off the corpse. `pkg/game` resolves an actor's art and displayed
name through `classes[e.Class]`, and no shipped `units.reg` carries a class 0, so the raised actor
was drawn as nothing and had an empty name.

`MAGIC-SING-019` (c) states the original builds the new actor from the `Data.bin` template named
by the literal at `L05127` — `Ghost` — owned by the caster's Player, copying six values off the
corpse: `reaction/2 + 1`, Mind, Spirit, `healthMax/2`, `+0xa6` and `+0xbe`.

Built:

- `sim.GhostTemplate` (`pkg/sim/spell.go`), a plain value carried across the `pkg/mapload` seam
  exactly as `SpellRule` is.
- `mapload.ghostTemplate` (`pkg/mapload/ghost.go`) resolves the `Units` row whose entry name is
  `Ghost`, applies the mission's difficulty, and fills the template. The class key is the row's own
  `TypeID` column, which is the same number a creature placement carries in `ClassID` and the same
  number `pkg/game` indexes `units.reg` with.
- `sim.NewSummoningWorld`, a ninth constructor carrying the template beside the spell table.
- `raisedGhost` builds the actor from the template and copies the four decoded values off the
  corpse.
- A world with no template refuses the cast **at admission**, so no mana is spent.

Measured against the EN install with an out-of-tree probe module: the `Units` collection holds 119
entries; entry 92 is named `Ghost` with `TypeID` 69 and face 1 (entries 93-95 are `Ghost.2`..`.4`,
same `TypeID`, faces 2-4); `units.reg` carries 34 classes and class 69 is named `Ghost`.

Two distances from `MAGIC-SING-019` (c) remain and are proposed as a ledger row:

1. `+0xa6` and `+0xbe` are not decoded and are copied by nothing here.
2. The claim grades the `Ghost` clause **Medium** — the by-name constructor `R0501` and the
   `Data.bin` row it resolves were not read — so what is established is the row's **name** and not
   its statistics. Taking the named row's own columns is the nearest this build can stand to that.

A third, smaller distance: the difficulty is applied to the template, which is authored. The
original's raise-time treatment of the difficulty setting is not decoded.

**The template is not in the byte form.** It is install-derived input and `UnmarshalBinary` carries
the receiver's own value across the decode (`pkg/sim/binary.go`, and the pin entry in
`pkg/sim/nostate_test.go` states the exception). `pkg/game`'s resume unmarshals *into* the world
`StartMissionFrom` already built, so a resumed mission raises from the template the definition
table gave it. A decode into a fresh `World` — which is what the read-only save instruments do —
keeps the zero template and refuses the raise at admission.

I judged the alternative too expensive for this round and record the judgement so the seat can
overrule it: adding a 73-byte section to the form was implemented, and it failed twelve pin tests
across eight test files in two packages, four of which strip earlier stories' additions and would
each have needed a new stripper. Two other lanes are editing `pkg/sim` right now. The cost is a
stated property: two worlds equal in bytes may differ in which template they raise from.

Witnesses: `TestControlSpiritConsumesBonesAndCreatesANewOwnedGhost` (extended with the whole
template comparison) and `TestControlSpiritWithoutATemplateIsRefusedBeforeItsCost`
(`pkg/sim/spelleffect1001_test.go`).

### B1. Bless's magnitude landed in the Astral protection slot

`Entity.Protection` is `[5]int32` and this vocabulary has four protection effect kinds.
`rearm.go:214` indexed `EffectProtectionFire + EffectKind(k)` over all five slots, so `k = 4`
asked for `EffectBless`, whose magnitude is a probability in [20,100] (`MAGIC-SING-019` (e)). A
blessed actor's Bless magnitude therefore landed in `Protection[4]` on every `SetCombat`, where it
was hashed, serialized and displayed as the Astral row, and where Bless's own removal could not
reach it.

`SetCombat` now adds an effect delta only for the four slots that have a kind.

Witness: `TestBlessDoesNotLandInTheAstralProtectionSlot`.

### B2. Apply and expire were not inverses under the clamps

`applyEffectDelta` clamped on the way in; `removeAttachedAt` subtracted the whole nominal magnitude
on the way out. A power-45 Protection effect on a base of 80 clamped to 100 and expired to 70 —
ten points of base protection destroyed permanently. `ScanRange` had the mirror fault: a Darkness
clamped at 0 gave back its whole magnitude.

`applyEffectDelta` now returns what actually landed and whether the kind is one it moves at all,
and `attachEffect` stores the landed delta as the record's magnitude for every non-continuous
effect. Expiry is then the exact inverse by construction.

Two exclusions, both load-bearing:

- **Continuous** effects are excluded, because their stored magnitude is re-applied on every eighth
  tick rather than reversed. A continuous heal that clamped at full health once would otherwise be
  re-applied as zero for the rest of its life.
- **Kinds `applyEffectDelta` does not move** are excluded by the second return value. Bless, Curse
  and Invisibility carry a magnitude other code reads as a parameter, not a state delta.

Witness: `TestAClampedEffectGivesBackExactlyWhatItTook`, both sub-cases.

### C1. A book cast paid its mana for a release the apply then refused

`spell.go:668` paid, then `:686` returned after `ordinaryEffect` refused. `spec.md` states twice
that this must not happen. Two shipped rows reached it: Control Spirit requires
`Decay == DecayBones` and a body spends its whole dwell at `DecayFallen` first, so casting it on a
freshly killed body is the ordinary case; Teleport's unit-target arm requires `terrainOpen` at the
destination. `BookSpellRefusal`, which the read-only `cmd/savecheck` instrument and the closure
rely on, reported both as admissible.

Built:

- `pointEffect` extracts the per-arm kind, magnitude, duration and mode into one function, so the
  admission predicate and the apply cannot disagree about whether an effect can land.
- `pointEffectRefusal` names the row-specific target condition: Control Spirit's bones corpse, a
  free entity id, a loaded template; Teleport's open destination; and a computed duration of zero
  for any timed row.
- `castSpell` asks it before the cost (step 6b), and `bookSpellRefusal` asks it at admission.

Witness: `TestARefusedApplyCostsNoManaAndIsRefusedAtAdmission`, three sub-cases.

**One hole is left and it is not this lane's file.** `castBookAt` (the cell form) pays at
`spell.go:773` and then returns if `landArea` refuses at `:780-782`. `landArea` lives in
`pkg/sim/celleffect.go`, which the area lane owns. Proposed for that lane or for the seat: either
`landArea` acquires a read-only companion that `castBookAt` and `beginBookSpellAt` both ask before
the cost, or `landArea` is documented as total for an in-bounds cell and the dead `return false`
removed. I did not change it.

### C2. Withdrawn as a defect by the owner

Owner ruling, 2026-08-15: «это нормально, потом мы вынесем это в настройку в игре "auto-heal",
сейчас по умолчанию все правильно». An idle mage healing a wounded ally without an order, and
training its own school while it does so, is the intended default. No gate was restored and no
gate was added. The `awardSkill` source index stays `-1` on the book-cast path.

Three pieces of bookkeeping are below: a corrected `spec.md` sentence, a ledger row, and the stale
fixture comment at `pkg/sim/skill_test.go:519`, which now says that a book cast supplies no source
entity and that this is the owner's default. It was the only comment in the tree still asserting
the removed gate.

**The future seam.** The auto-heal setting will need a per-actor or per-player switch. Nothing in
this round makes that harder: the whole unbidden decision is `autoCastOrder`, which already returns
an ordered list and already has one owner-authored gate in it (`affordsAutoHeal`). A setting is one
more condition on the unarmed-heal arm of that function. No caller of `autoCastOrder` reads the
list's provenance, so adding the condition changes nothing else.

### C3. Prismatic Spray applied to corpses

`applyPrismatic` walked every entity in radius with no liveness test, so each corpse took a damage
roll — consuming world RNG, which moves every later draw — and awarded the caster `(manaCost+1)/2`
school experience. The area collector already skips the dead.

`applyPrismatic` is in `pkg/sim/celleffect.go`, which is not this lane's file, so the guard is
stated once at the head of `ordinaryEffect` (`pkg/sim/spell.go`) instead: a dead entity takes no
ordinary apply, with Control Spirit the one exception. Every other caller already refused a dead
target before reaching it, so this changes no other path.

Witness: `TestPrismaticSprayPassesOverCorpses`, against a control world holding no corpses at all.
It compares the living victim's health, which is the generator's position, and the caster's banked
school experience.

### E2. Stone Curse's magnitude was a literal

`spell.go:279-280` wrote `mag = 5`, discarding `rule.EffectMagnitude`. `MAGIC-EFFECT-015` names
`stone_curse` as one of the two arms that write no `+0x40`, which is what makes the `Effects`
column's own number the magnitude. The shipped row carries `absorbtion=+5`, so the literal read the
same on shipped data and made the column dead — against `contract.md:66`, which claims installed
rows are the source of every row value.

The generic tail's `mag = rule.EffectMagnitude` now stands, and the arm keeps only its duration
scaling.

Measured on the EN install: spell 26 is `mana=60 school=5 range=1 defensive=true`, spell 20 is
`mana=40 school=4 range=5 spellDur=10 effKind=4 effMode=1 effMag=5 effDur=8`. The second `Effects`
record `defence=-20:duration 8` is not parsed, which matches `MAGIC-EFFECT-015`'s single
construction site.

Witnesses: `TestStoneCurseTakesItsMagnitudeFromTheInstalledRow` (two magnitudes, one shipped and
one edited) and `TestStoneCursesDurationIsShortenedByEarthProtection`, which exercises
`MAGIC-SING-019` (d) — the only place a resistance shortens an effect. Nothing exercised Stone
Curse before this round.

### E3. The untimed test was `& 3` where the claim gives `& 7`

`MAGIC-ATTACH-016` reads the original's gate as `+0x3d & 7 == 0`, which covers `charges` (4) as
well as duration and continuous. `spell.go:291` tested only two of the three bits, so a row edited
to carry `charges` would have been applied once and stored nowhere — reversible by nothing. No
shipped row uses it; this is a customisation-seam defect.

`effectTimedModes` is now `EffectDuration | EffectContinuous | EffectCharges`.

Witness: `TestAChargesModeRowIsStoredRatherThanAppliedAndForgotten`.

### F1. Invisibility was cancelled by a landed blow

`MAGIC-SING-019` (f) names `R0245`, the attacker's own melee approach, as the second of the
two cancellations. The removal sat after the to-hit gate in `resolveBlow`, so an invisible attacker
that missed stayed invisible, and one whose victim stood out of reach stayed invisible for as long
as it walked.

The removal moved to `approach` (`pkg/sim/combat.go`), which runs every tick for an actor holding
an attack order on a victim the world still has. The `resolveBlow` site is gone.

Witness: `TestAnAttackersApproachCancelsItsOwnInvisibility`.

### F2. The continuous cadence — research question, no code change

`pkg/sim/effect.go` decrements `Remaining` and then tests `Remaining % 8 == 0`. Poison Cloud is
the only shipped `continuous` row and ships `duration 8`, so the remainder is 7 after the first
decrement and the eighth-tick re-application never fires: the effect applies exactly once, at
attach.

`MAGIC-ATTACH-016` says `R0673` re-runs `vt+0x40` on every eighth tick, naming `L05207`
and `L05208`'s sign-preserving `AND 7`, and separately that `+0x42 > 9600` skips the countdown.
It does **not** state whether the `AND 7` reads `+0x42` before or after the decrement. Under the
other order a `duration 8` row applies twice rather than once, at attach and at the tick after.

**I did not change the code.** The question below is proposed for research, and a ledger row is
proposed to hold the gap until it is answered. Nothing in the pinned tree settles it: the claim is
the only published reading of that routine, and a fact that exists only inside an experiment is a
request to research rather than something to cite.

Proposed question, in re-derivable form and without handing over our own value (B1):

> In `R0673`, what is the exact order of operations on the effect's `+0x42` countdown word
> between the entry test at `L05209` and the eighth-tick re-application at `L05207`/`L05208`?
> Specifically: is the value the `AND 7` tests the word as it stood on entry, or the word after the
> decrement? State the instruction addresses of the decrement and of the load the `AND` consumes.
> A worked consequence for a row whose stored duration is a multiple of 8 would settle how many
> applications such a row performs over its life.

### F3. A duration-mode health effect could fell an actor without clearing its order

`removeAttachedAt` reversed the magnitude with no `clearFelled`, unlike attach and the continuous
tick. A duration-mode health effect that raised health gives it back on expiry, and the reversal can
take the target below zero — leaving a felled actor holding a walk order, which is one of the five
states `pkg/sim/binary.go`'s own decoder refuses on the ground that a tick cannot produce them.
Unreachable on shipped data (the only `EffectHealth` row is continuous, whose reversal is skipped);
real on the customisation seam.

`removeAttachedAt` now calls `clearFelled` after the reversal.

Witness: `TestAHealthEffectThatFellsItsTargetOnExpiryClearsTheOrder`, which also marshals the
resulting world.

### The character sheet's protection rows (owner play report, 2026-08-15)

Verbatim: «защиты от огня и тд это спец спрайт горящий над головой … и в статах я не вижу что
сопротиваления выросли».

Verified, not assumed: `pkg/ui/panel.go:903-912` reads `s.Char.Protection`, and `UnitCharacter`'s
protections are filled from the **data-derived base** — `mapload.Sheet.Elemental` through
`sheetCharacter` (`pkg/game/panelchars.go:91-94`) for a placement, and `PartySpawnWithTable`'s
derived block through `partyPanelSubject` (`pkg/game/world.go:1082-1085`) for a party member. Both
are computed once, at mission start. A Protection effect moves `sim.Entity.Protection`, which is
what the damage resolver reads, and never reached the sheet.

`entityDraws` (`pkg/game/world.go`) already overlays Experience and Skills from the entity of the
tick. The five protections now join them. Both arrays are `UNIT-COMBAT-015`'s column order — Fire,
Water, Air, Earth, Astral — so this is a widening and not a permutation.

Item B1 was fixed first, as instructed, so the Astral row is no longer showing a Bless probability
while this was being tested.

Witness: `TestTheSheetsProtectionRowStatesTheLiveValueWhileAnEffectStands`
(`pkg/game/spellsheet1001_test.go`). It reads the value the panel is handed — `MapEntity.Char`,
through `entityDraws` — not the entity field, and asserts the rise while the effect stands and the
return when it expires.

The owner's four-protections-at-once observation is presentation and is not in this lane. No sprite
was built.

### The Teleport autocast, removed (owner ruling, 2026-08-15)

Verbatim: «давай телепорт пока не будем тогда трогать, но текущий автокаст его нужно убрать, потому
что сейчас маг просто стоит на месте и тратит ману впустую».

`autoCastable(rule)` returns false for spell 26. `autoCastOrder` treats such a row as not armed, and
`stepAutoCasts` **clears** a stored `AutoSpell` naming it.

The clear is the part that matters for what the player sees. The front end reads
`Entity.AutoSpell` for the dashed border and derives nothing, so filtering alone would have left the
border standing over a cell that never fires — the state the ruling explicitly forbids. Clearing
also covers a setting that reaches this build from an older save or from a script.

**No change to `pkg/ui/spellbook.go`, and the reason is a seam.** `pkg/ui` does not import
`pkg/sim` and knows nothing about spell semantics; `SpellbookEntry.Autocast` is carried from the far
side and derived from nothing locally. So a UI-side refusal would need either a hard-coded spell id
in `pkg/ui` or a new field on `SpellbookEntry` filled in `pkg/game`. Neither is necessary: with the
simulation refusing to hold the setting, the toggle is sent, the setting is not kept, and the next
push shows no border.

**Proposed for the seat, in `pkg/sim/step.go` (the orders lane's file), to refuse at the sink as
well as at the sweep.** The `KindAutocast` arm at `step.go:513-517` currently reads:

```go
			if c.X <= 0 || c.X > 0xffff {
				w.entities[i].AutoSpell = 0
			} else {
				w.entities[i].AutoSpell = uint16(c.X)
			}
```

Proposed replacement:

```go
			// A row automatic selection may not pick is not armable
			// (autoCastable, spell.go): the store is refused rather than kept
			// for stepAutoCasts to clear a tick later.
			rule, known := w.findSpell(uint32(c.X))
			if c.X <= 0 || c.X > 0xffff {
				w.entities[i].AutoSpell = 0
			} else if !known || autoCastable(rule) {
				w.entities[i].AutoSpell = uint16(c.X)
			}
```

An id the world's table does not hold is stored exactly as it is today, so the arm's existing
behaviour for an unknown row is unchanged.

This is optional. `stepAutoCasts` already clears the setting within one tick; the sink refusal only
makes the state never exist rather than existing for part of a tick.

**Measured, and it is not the cause of the standing-still symptom.** The shipped Teleport row
carries `Spell Defensive` and is not restorative, so `autoCastTarget`'s defensive shortcut answers
with the caster's own id; the arm then teleports the caster onto the cell it already stands on — an
open cell, so nothing refuses it — spends the row's 60 mana for a move of zero, and clears the
caster's own walk order and route. The standing-still symptom the owner reported has a different
cause, general to every armed offensive row, measured by the orders lane: the mover stood down for
`CastWait` as well as for the wind-up, and `stepAutoCasts` runs before the mover in the tick. This
removal does not fix that and is not claimed to.

Witnesses: `TestAnArmedTeleportNeitherFiresNorStaysArmed` and `TestAPlayerIssuedTeleportStillWorks`
— the manual path is untouched.

## Coverage the review found missing

`TestShieldLightAndDarknessLandTheirDecodedMagnitudes` casts Shield (`p/10 + 3`), Light
(`p/30 + 1`) and Darkness (`-1 - p/30`) and reads the resulting magnitudes.
`TestStoneCursesDurationIsShortenedByEarthProtection` and
`TestStoneCurseTakesItsMagnitudeFromTheInstalledRow` cover Stone Curse.
`TestTwoSlowingEffectsLeaveAnActorSlowRatherThanUnrated` and
`TestASlowedActorGetsExactlyItsSpeedBack` cover Slow and a resulting non-positive speed.

## The observable result

Script-gap census, this branch, EN root:

```
go build -o /tmp/mr ./cmd/missionrun
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 10 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
AGAINROM_ASSETS=<againrom>/gameversions/en /tmp/mr -mission 20 -trace -ticks 1 | grep -c UNSUPPORTED   -> 0
```

Both are 0 at 1 tick and at 200 ticks. Zero is the floor, so these two numbers cannot fall and this
round did not move them. `pipeline/milestone-baseline.txt` records the two missions' script sizes
(`en m10 script 16 checks, 27 instants, 12 triggers`; `en m20 script 14 checks, 15 instants, 11
triggers`) and carries no unsupported count of its own.

The result this round owes is therefore in `builds/current/`, not in the census: a Control Spirit
cast now raises a drawn, named Ghost instead of an invisible nameless actor; a Protection cast now
moves the number on the character sheet; a slowed unit is slower rather than 21 times faster; and
an armed Teleport no longer drains a mage's mana.

**Not witnessed on screen.** I did not drive the GUI. Every claim above rests on a headless test or
on a probe module reading the lawful install; none rests on having seen the game window.

## Proposed `spec.md` sentences

The seat folds these in at the landing. `spec.md` is not edited by this lane.

Replacing `spec.md:220`:

> The existing skill sink applies its human, mage, level, participant and cap gates. A book cast
> supplies no source entity, so the sink's same-owner and locked-relation refusals do not apply to
> it: an idle mage that heals an ally of its own team trains its school for doing so. That is the
> owner's default (2026-08-15) and it becomes an in-game "auto-heal" setting in a later story.

Adding beside `spec.md:118` (effect expiry):

> Expiry reverses exactly what the application landed. Where a clamp meant less than the nominal
> magnitude was applied, the stored magnitude is what was applied, so no effect can rewrite an
> actor's base protection or scan range. A reversal that takes an actor below zero health fells it
> through the ordinary path.

Adding beside `spec.md:59-60` and `spec.md:214-216` (the refusal contract):

> A row's own target condition is asked at admission and again before the cost: Control Spirit
> requires a corpse at the bones stage and a loaded `Ghost` template, Teleport requires an open
> destination cell, and any timed row requires a computed duration above zero. `BookSpellRefusal`
> answers with the same predicate, so the read-only instrument and the cast agree.

Adding to the automatic-cast section:

> Automatic selection never picks Teleport, and a stored autocast naming it is cleared. A
> player-issued Teleport is unaffected.

Adding to the effect-mode section:

> An effect is untimed when none of the duration, continuous and charges bits is set. An untimed
> effect is applied once and stored nowhere; every other mode is stored and counted down.

Adding to the speed section:

> An effect may not take an actor's speed below 1.

## Proposed `docs/DIVERGENCES.md` rows

Ready to paste, in the ledger's own column order. `DIV-028` onward assumes `DIV-027` is the last
row at the landing; renumber if another lane lands rows first.

| DIV-028 | simulation / effect speed floor | Slow must slow (2026-08-15 review: a slowed unit was about 21x faster) | `HERO-SPEED-008` gives the speed expression and establishes that the effective speed reaches the mover as a byte at `[actor+0x154]+0xa`. What the original does when an arm drives that value below zero is not decoded | An effect may not take `Entity.Speed` below 1. The stored magnitude is what was applied, so expiry restores the original value exactly | UNKNOWN | Zero is the value `rated` reads as "no rate at all", which puts an unrated mover on the fastest cadence the engine has. Reachable on shipped rows: Slow at power 60 for a speed-10 unit, power 45 for the four speed-8 rows | A claim reading what the original stores when the speed arm computes a negative value | OPEN |

| DIV-029 | magic / Control Spirit template | Control Spirit must raise something the player can see and name | `MAGIC-SING-019` (c) states the original builds the actor from the `Data.bin` template named by the literal at `L05127` — `Ghost` — copying `reaction/2 + 1`, Mind, Spirit, `healthMax/2`, `+0xa6` and `+0xbe` from the corpse. The clause is graded **Medium**: the by-name constructor `R0501` and the row it resolves were not read, so the row's NAME is established and its statistics are not | The raise resolves the `Units` row named `Ghost` and takes its class key, domain, speed, sight, reach, token size, dying time, experience value, protections and combat block; the four decoded values come off the corpse. `+0xa6` and `+0xbe` are copied by nothing. The mission's difficulty is applied to the template | FIDELITY-DEBT | The visible defect — an actor with no class, drawn as nothing and named nothing — is fixed on the claim's own name. The two unnamed offsets and the template's statistics are not established | A claim reading `R0501` and the `Ghost` row's own columns | OPEN |

| DIV-030 | magic / book-cast training source | An idle mage healing an ally without an order, and training for it, is the correct default; it becomes an in-game "auto-heal" setting later (2026-08-15) | `MAGIC-TRAIN-018` establishes that training runs on **every** apply from a book, gated on the spellbook bit and on not being an item cast, and credited to the spell's `Sphere`. `HERO-SKILLGATE-074` establishes that the sink refuses a same-owner source and a treaty-protected source **when a source actor is supplied and is not already dead**. Neither claim states whether the cast feed `R1053` supplies a source actor at all: the kill and damage feeds are described as reading `victim+0x1c`, the cast feed is described as reading only the spell's `Sphere` and `Mana Cost` | A book cast supplies `awardSkill` no source entity, so the same-owner and locked-relation refusals do not reach that path. A commanded or unbidden heal on an own-team ally trains the caster's school | UNKNOWN | Whether the original's cast feed supplies a source is not established by any claim. The behaviour is the owner's directive and outweighs the research absence | A claim stating the arguments `R1053` passes to `vt+0x5c` | ACCEPTED |

| DIV-031 | magic / continuous re-application cadence | — | `MAGIC-ATTACH-016` states a `continuous` effect re-runs `vt+0x40` on every eighth tick (`L05207`, `L05208`'s sign-preserving `AND 7`) but does not state whether the `AND 7` reads the `+0x42` countdown before or after the decrement | The countdown is decremented and then tested, so a row shipping `duration 8` applies once, at attach, and the eighth-tick re-application never fires. Poison Cloud is the only shipped `continuous` row and ships `duration 8` | UNKNOWN | The order is not decoded and the two orders differ by one application on the only shipped row. No guess was made | A claim stating the order of the decrement against the `AND 7` in `R0673` | OPEN |

| DIV-032 | magic / Teleport autocast | Remove the current Teleport autocast (2026-08-15: «текущий автокаст его нужно убрать»). The route-following Teleport autocast is deferred | Book-spell autocast has no counterpart in ROM1 at all; it is an againrom affordance (`0154`) | Automatic selection never picks Teleport, and a stored autocast naming it is cleared on the next tick. A player-issued Teleport is unaffected | DEVIATION | The shipped row is `Spell Defensive` and not restorative, so automatic targeting answers with the caster's own cell: the cast is a move of zero that costs 60 mana and clears the caster's own order. With route-following deferred there is no sensible automatic behaviour | A story building the deferred route-following Teleport autocast | ACCEPTED |

| DIV-033 | persistence / ghost template | Control Spirit must work in a resumed mission | — | `sim.GhostTemplate` is install-derived input carried on the world and **not** in the byte form. `UnmarshalBinary` carries the receiver's own value across the decode, so a resume — which unmarshals into the world the mission start already built — keeps its template. A decode into a fresh world keeps the zero template and refuses the raise at admission | DEVIATION | Adding the record to the form failed twelve pin tests across eight test files in two packages while two other lanes were editing `pkg/sim`. Two worlds equal in bytes may differ in which template they raise from | A byte-form story with the package to itself | OPEN |

## What I did not do

- **F2**: no code change. The question is above and `DIV-031` holds the gap.
- **`castBookAt`'s pay-then-refuse hole**: `landArea` is the area lane's file. The proposal is under
  C1.
- **`pkg/sim/step.go`'s autocast sink**: the orders lane's file. The exact proposed replacement is
  under the Teleport section.
- **`pkg/sim/group.go`'s `groupMinSpeed`**: not this lane's file. The floor removes its
  reachability through the spell path; the conversion itself is unchanged.
- **`pkg/ui/spellbook.go`**: no change, and the seam reason is under the Teleport section.
- **`spec.md`, `closure.md`, `docs/DIVERGENCES.md`**: not edited. The proposed text is above.
- **No GUI drive.** No claim here rests on having seen the game window.

## Files this lane changed

Production: `pkg/sim/spell.go`, `pkg/sim/effect.go`, `pkg/sim/rearm.go`, `pkg/sim/combat.go`,
`pkg/sim/world.go`, `pkg/sim/binary.go`, `pkg/mapload/ghost.go` (new), `pkg/mapload/fromalm.go`,
`pkg/mapload/start.go`, `pkg/game/world.go`.

Tests: `pkg/sim/spellpointfix1001_test.go` (new), `pkg/game/spellsheet1001_test.go` (new),
`pkg/sim/heal_test.go`, `pkg/sim/skill_test.go`, `pkg/sim/spelleffect1001_test.go` (point tests
only), `pkg/sim/nostate_test.go` (the World field-set pin), `pkg/sim/world_test.go` (the exported
method pin). The last two are pins that any new field or method must move; both are one-line
additions.

`pkg/sim/skill.go` was not changed: C2 was withdrawn before any change was made to it.

---

# Round 2, second pass: the two F4 items no lane took

Branch `story/1001-r2-point2`, based on the merged `story/1001-spell-effects` at
`9f590d9`. Scope: review section F4's first and third bullets. Nothing else was touched,
and `spec.md`, `closure.md` and `docs/DIVERGENCES.md` were left to the seat.

## Item 1 — the `released` guard had no witness

`TestTheCadenceFloorHoldsASecondCastOffForCastPeriodTicks` was rewritten from
`Step(w, []Command{cast, cast})` to `spRunCast(w, cast)`, and its failure message went on
naming *"two casts in one slice"* after the slice was gone. Two things were lost with the
slice: the one-slice-two-casts path itself, and the `released` set of `pkg/sim/step.go`,
which is the only thing that path exercised.

`step.go` is the orders lane's file and it is merged. I read it and did not change it. The
guard is correct where it stands; what it lacked was a test.

**The stale message is corrected** — the test admits one command and its assertions now say
so.

**Three tests were added to `pkg/sim/spell_test.go`.**

`TestTwoCastsInOneCommandSliceArmOneWindUp` restores the lost path: one slice carrying the
same cast twice arms one wind-up, nothing is paid at admission, and exactly one cast is paid
when it releases.

That test pins a property and not an implementation, and the note in it says so, because
the path turned out to be **guarded twice, redundantly**. Measured by removing each guard:

| removed | result |
|---|---|
| `actorCastBusy`'s pending-cast clause (`pkg/sim/actionguard.go`) | test still green |
| `queueBookCast`'s existing-cast check (`pkg/sim/spell.go`) | test still green |
| both | **reddens**: two wind-ups armed, want 1 |

`TestACastThatReleasedThisTickCannotBeReArmedInTheSameTick` is the witness for the
`released` set at the `KindCast` arm. It reddens when `!released[e.ID] &&` is deleted from
`pkg/sim/step.go`:

```
spell_test.go:504: a cast that released at the head of this tick armed another in the
same tick: [{Caster:1 Target:2 Spell:1 X:3 Y:0 Remaining:8 AtCell:false}]
```

**The set is load-bearing only for a zero-recovery caster**, which is why the fixture gives
the caster `AttackRelax = 0`. `stepBookCasts` runs at the head of the tick, before the
command loop; `castSpell` then loads `CastWait` with `castRecoveryTicks`, which is the
caster's own `AttackRelax`. With a non-zero relax, `bookSpellRefusal`'s busy gate refuses
the same-tick command on its own and `released` is redundant. With a zero relax the caster
stands at `CastWait 0` with no pending cast the instant its old one applied, so `released`
is the only thing between a command slice and a second cast in the same tick.

`TestAnAttackOrderIsRefusedOnTheTickACastReleased` is the same set at its other arm. It
reddens when `if released[e.ID] { continue }` is deleted from the `KindAttack` arm:

```
spell_test.go:537: an attack order was taken on the tick the caster's own cast released
(target 2)
```

Reproduce all three:

```
go test -trimpath -count=1 -run 'TestTwoCastsInOneCommandSliceArmOneWindUp|TestACastThatReleasedThisTickCannotBeReArmedInTheSameTick|TestAnAttackOrderIsRefusedOnTheTickACastReleased' ./pkg/sim/
```

**Nothing in `step.go` needs to move.** The guard is in the right place; the redundancy in
the other path is not a defect either, only a fact a test cannot attribute, and the test
says which property it pins rather than pretending to pin one of the two guards.

## Item 2 — the raised health bars, and whose unit they read as

The reversal itself stands: the owner asked for bars above the unit square. What follows is
the answer to whether one unit's bar can be mistaken for the unit north of it.

### The measurement

`poolBarRects` puts a bar at `y0 = row*cellpx - scaleDim(above, cellpx)` with height
`scaleDim(3, cellpx)`, `above` being 8 for health and 4 for mana. Measured at every
supported scale, for a unit at row R:

| cellpx | health band | mana band | own cell top | northern cell | px above own cell | px below northern top |
|---|---|---|---|---|---|---|
| 16 | 44–46 | 46–48 | 48 | 32–48 | 2 | 12 |
| 32 | 88–91 | 92–95 | 96 | 64–96 | 5 | 24 |
| 48 | 132–137 | 138–143 | 144 | 96–144 | 7 | 36 |
| 64 | 176–182 | 184–190 | 192 | 128–192 | 10 | 48 |
| 96 | 264–273 | 276–285 | 288 | 192–288 | 15 | 72 |

### The answer

**Distinguishable, with one named exception.**

Three properties hold at every scale, and `TestABarReadsAsItsOwnUnitAndNotTheOneNorthOfIt`
(`pkg/render/terrain/overlay_test.go`) asserts all three:

1. The band **is** inside the northern neighbour's cell. That is the cost of the
   arrangement, and the test asserts it rather than leaving it unsaid, so a later change to
   the band positions has to face it.
2. The band **hugs its own cell**: it stands at least four times closer to the top edge of
   the cell it belongs to than to the top edge of the cell it is drawn in (2 px against 12
   at the tightest scale, 15 against 72 at the loosest). It reads as sitting on its own
   unit rather than floating in the neighbour's square. The factor of four is the margin
   this arrangement has, not a law; it is pinned so a change has to face the question.
3. Two units in vertically adjacent cells produce four bands that are pairwise **disjoint**,
   and the two groups are separated by more than the height of either group — 12 px at
   cellpx 16 against a 4 px group, 25 px at cellpx 32 against a 7 px group. One unit's bar
   cannot be read as the other unit's bar, because the two are nowhere near each other.

The test discriminates. Measured by moving the constants:

| change | result |
|---|---|
| `healthBarAbove` 8→20, `manaBarAbove` 4→16 | **reddens** on property 2: 8 px above its own cell against 6 px below the northern top |
| both → 0 (bars back inside their own cell) | **reddens** on property 1 |

### The exception, disclosed and pinned

**A unit's mana bar crosses the bottom edge of the northern cell's selection rim**, at every
supported scale. The mana band occupies the four scaled pixels immediately above its own
cell; the rim's bottom piece occupies that cell's bottom two scaled pixels. At cellpx 32
the mana bar is 92–95 and the rim's bottom piece is 94–96.

So a unit standing directly south of a **selected** unit draws its mana bar across that
unit's selection rim. It is not an attribution error — nobody reads the bar as the northern
unit's — but it does break the rim the player is using to see which unit he has selected.

The health bar, four scaled pixels higher, is clear of the rim and of every other glyph
piece of the northern cell.

`TestAManaBarCrossesTheNorthernCellsSelectionRim` pins this: it fails if the health bar
starts hitting the rim, if either bar hits any other northern glyph, **and** if the mana bar
stops hitting the rim. The last one is deliberate — the collision cannot be quietly fixed or
quietly made worse.

**No correction is proposed with it, and the obvious one is measured not to work.** The mana
bar cannot move alone: at cellpx 16 the health and mana bands already sit one scaled pixel
apart. Moving both — `manaBarAbove` 4→6 with `healthBarAbove` 8→9 — clears the rim at all
five scales and then drives the health bar into the northern cell's own unit marker at
cellpx 16, where the bar lands on 43–45 and that marker's vertical piece runs 37–44. Both
outcomes were measured by editing the constants and rerunning. Clearing the rim without
buying a worse collision means changing a glyph rather than a constant, and the band
positions are the owner's approved arrangement, so this is his call and not a lane's.

### The fixture that dodged the change

`pkg/ui/fogindicators_test.go` moved `fiSeen` from (0,0) to (0,1) with the comment *"Row one
leaves room for the owner-directed bars above the unit. Keeping this fog test away from the
viewport edge ensures it still measures fog, not clipping."*

**That ground does not hold for this fixture, measured.** `fiViewer` lays out 640×480 over a
4×4 grid of 32 px cells, so the map occupies 128 px and the camera centres it with roughly
256 px of margin on each side. A row-zero bar lands at screen y 168 — nowhere near an edge.
Probed at three cells with every cell visible: (0,0), (0,1) and (2,2) each produce exactly
one ground, at screen y 168, 200 and 232.

`fiSeen` is back at **(0,0)**, so the case faces the raised geometry, and the whole `pkg/ui`
suite is green with it there. `TestARowZeroHealthBarIsDrawnOnScreenAndNotClippedAway` was
added beside it: it asserts the row-zero bar is produced, is non-empty, and lies inside the
640×480 viewport. If a layout or camera change ever does clip a row-zero bar away, that test
fails as a clipping failure instead of the fog counts silently dropping to one.

Reproduce:

```
go test -trimpath -count=1 -run 'TestABarReadsAsItsOwnUnitAndNotTheOneNorthOfIt|TestAManaBarCrossesTheNorthernCellsSelectionRim' ./pkg/render/terrain/
go test -trimpath -count=1 ./pkg/ui/
```

## Proposed `docs/DIVERGENCES.md` row

One row, from `DIV-042` as instructed.

| DIV-042 | client / unit bars over the northern cell | Health and mana bars are drawn above the unit square (2026-08-15) | Bars over a unit have no ROM1 counterpart; this is an againrom affordance | A unit's health and mana bands sit wholly inside the cell of the unit to the north. They hug their own cell — at least four times closer to its top edge than to the northern cell's — and two vertically adjacent units' bands are pairwise disjoint and separated by more than either group's height, so neither reads as the other's. The mana band does cross the bottom edge of the northern cell's selection rim, at every supported scale | DEVIATION | The owner asked for bars above the unit square, which necessarily puts them in the northern cell. The rim crossing has no correction that is not worse: the mana bar cannot move alone at cellpx 16, and moving both bars up clears the rim but drives the health bar into the northern cell's unit marker at that scale | An owner decision on the band positions or on the selection rim's own art | ACCEPTED |

## Proposed `spec.md` sentence

Only one, for the client-presentation section:

> A unit's health and mana bars are drawn above its own cell, which places them inside the
> cell of the unit to the north. They stay clear of every glyph of both cells except the
> northern cell's selection rim, which the mana bar crosses, and they stay far enough from
> the northern unit's own bars that neither can be read as the other's.

## What I did not do, this pass

- **`pkg/sim/step.go`**: read, not changed. The `released` guard is correct where it is; it
  only lacked a witness.
- **`pkg/render/terrain/overlay.go`**: not changed. The band constants are the owner's
  arrangement and the one collision has no correction that is not worse.
- **No GUI drive.** The bar geometry is measured from `HealthBarRects`, `ManaBarFootprintRects`
  and the marker rect functions, and the viewport placement from `healthBarScreenRects`. No
  claim here rests on having seen the game window; the rim crossing in particular is a
  rectangle intersection and not a screenshot.

## Files this pass changed

Tests only: `pkg/sim/spell_test.go`, `pkg/render/terrain/overlay_test.go`,
`pkg/ui/fogindicators_test.go`. No production file was touched.
