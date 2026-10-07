# 1001-spell-effects — round 3, the second adversarial review

Reviewed at the pushed sha `70a7b74`, research pin `744214fe1f4b461cbd2d6766390b850ab15779ab`.
Three independent reviewers with fresh context, under three lenses: the fix round's own claims
(A), the untouched adjacent subsystems (B), and the documents against the code (C).

**All three returned INCOMPLETE.** Six counterexamples stand, one of them save-breaking. Two of
them are regressions introduced by fix round 2. The documentation defects the third reviewer found
are corrected in commit `936f9b2` and are recorded at the end of this file rather than handed to a
lane.

Every finding below was re-run by the seat before it was written down.

## R3-A1 — the speed floor holds at one writer and not at the others

**Rank 1. Player-visible. Not save-breaking, but the wrong value is canonical and persists.**

Fix round 2 closed the "a slowed unit was 21x faster" finding by flooring `applyEffectDelta` at
`minEffectSpeed` and by storing the landed delta so expiry restores the original value. Both halves
fail as soon as more than one `EffectSpeed` record stands on an actor.

- `pkg/sim/effect.go:125` — `removeAttachedAt` calls `w.applyEffectDelta(ti, e.Kind, -e.Magnitude)`
  and **discards the return**. When the reversal itself clamps at the floor, less is given back than
  the record stored, and `Entity.Speed` permanently diverges from `base + Σ(stored magnitudes)`.
- `pkg/sim/rearm.go:90` — `SetDerived` writes `e.Speed = d.Speed + w.effectDelta(id, EffectSpeed)`
  with **no floor at all**. The `ScanRange` line two below it is floored and `SetCombat`'s
  protection loop is clamped; speed is the one that is not.
- `pkg/game/rearm.go:100` reaches `SetDerived` from `recomputeRaisedSkills`, called every tick at
  `pkg/game/world.go:3067` for any hero whose skill level moved — which this story made routine,
  because every successful spellbook use trains.
- `pkg/sim/group.go:659` — `groupMinSpeed` turns a member at Speed −4 into `GroupSpeed 252` for the
  whole ordered group when it is scanned after a faster member.

The magnitudes that reach it are the shipped rows' own at power 100: Haste +7, Slow −7, Freezing
Cloud −7.

Measured, on the reviewer's probe re-run by the seat at `70a7b74`:

```
all three attached: speed=3
speed after every effect expired = 15, want the base 10

after Haste expired: speed=1 effectDelta=-14
after SetDerived: speed=-4 rated=false moverSpeed=-4
groupMinSpeed with the negative member last = 252

five-cell walk, speed  10: arrived at tick 104
five-cell walk, speed   1: did not arrive within 400
five-cell walk, speed   0: arrived at tick 4
five-cell walk, speed  -4: arrived at tick 4
```

A base-10 unit ends permanently at 15 with no `SetDerived` involved, and at −4 — the engine's
fastest cadence — with one.

**What it falsifies.** `spec.md`'s "Expiry reverses exactly what the application landed" and "An
effect may not take an actor's speed below 1"; `closure.md`'s Simulation row; and `DIV-036`'s
implemented-behaviour column, whose description therefore does not match the code.

**Why the lane's own witness could not catch it.** `TestASlowedActorGetsExactlyItsSpeedBack` uses
two negative effects and no positive one, so no removal ever clamps.

## R3-A2 — the occupant-slot fix makes a second ground actor immune to every area effect

**Rank 2. Player-visible. A regression introduced by fix round 2.**

`pkg/sim/celleffect.go:815` `cellSlotOccupants` returns at most one layer-0 actor per cell, and
`applyAreaCells` walks only that one. Round 1 walked up to three covering entities. The fix imports
`TERR-CELLREC-146`'s one-actor-per-domain-slot invariant into a build that does not maintain it:
`entityCoversCell` is footprint-aware while the movement occupancy plane is **anchor-only** —
`counted` at `pkg/sim/route.go:208` takes a single cell and `TokenSize` appears nowhere in
`route.go` or `step.go`.

So the ordinary mover walks a 1×1 actor inside a 2×2 actor's footprint, and the higher-indexed of
the two becomes unreachable by every blast, ring and cloud on that cell.

Measured:

```
2x2 actor 1 at (6,6); 1x1 actor 2 at (7,7) — inside its footprint
cellSlotOccupants(7,7) = ground index 0, air index -1
Fire Ball at (7,7): 2x2 HP 100 -> -4, 1x1 HP 100 -> 100
```

With round 1's occupant walk pasted back, the same probe gives `1x1 HP 100 -> -6`.

**Reachable on shipped data.** 15 installed `Units` rows carry `TokenSize > 1` — Ogre ×4, Troll ×4,
Catapult, Ballista, Daemon at 2×2 ground, Dragon ×4 at 3×3 air — and 147 such actors stand across
the 28 EN campaign maps. In mission 140 on the EN root, ordering the nearest 1×1 ground actor onto a
footprint cell of the 2×2 at (48,116) puts entity 56 at (49,117), inside it, through the ordinary
mover and real terrain. A second route needs no large unit: `castBookAt`'s Teleport arm tests only
`terrainOpen` at `pkg/sim/route.go:428`, terrain grid and no occupancy, so a mage teleports onto a
cell another actor already stands on.

**Not covered by `DIV-033`**, which states the keying "is decoded and implemented" and discloses
only the corpse-dwell question.

**Why the lane's own witness could not catch it.**
`TestAreaApplicationReadsTheDomainKeyedCellOccupantSlots` uses three decayed bodies plus one living
actor — a configuration in which only one entity holds the slot.

## R3-B1 — a weapon-borne release refuses every non-damaging row, so its carrier does nothing at all

**Rank 1. Player-visible. Reachable in shipped content on both roots.**

`releaseWeaponSpell` at `pkg/sim/spell.go:1091` applies only `applySpellDamage`, behind two
refusals: `!rule.Damaging || rule.ID == 14`, and `!rule.TargetsUnit`. `pkg/sim/combat.go:461`
reaches it as the **replacement** for the physical blow — `weaponSpellFor` selects the casting cycle
for any known row on a mage, and `advanceAttack` then calls `releaseWeaponSpell` instead of
`resolveBlow`. An actor whose weapon spell is refused therefore does nothing at all, which is
strictly worse than being unarmed.

`contract.md` claims the slice owns "manual, unbidden, script, and weapon/item releases through one
application path". `closure.md` claims "All 28 installed spells now reach one ordinary application
path" and marks Inventory/equipment PASS. The cited claims agree with the contract and not with the
code: `MAGIC-ITEM-007` and `MAGIC-CAST-003` establish that a weapon or item cast runs the ordinary
apply with the mana gate exempted, `MAGIC-TRAIN-018` that training alone is gated off, and
`MAGIC-SING-019` (g) names **id 14 only** as what the item-cast entry refuses. Nothing refuses
id 20.

**Shipped, both roots.** Mission 130 entity 55, `M130_Veglud`, Humans row 211, carrying
`Elven Magic Wood Staff {castSpell=Stone_Curse:50}`, class 23, mana 447/447. The mercenary-band
template `M_GoodFemale4`, Humans row 180, carries the same staff at power 30.

Measured, the placed actor lifted out of its own mission world and set beside a dummy on open
ground, ordered to attack, 240 ticks, identical on EN and RU:

```
as shipped (staff spell 20)                    victim hp=1000  attachedEffects=0  casts=0
control: same actor with WeaponSpell cleared   victim hp=629
```

Zero damage and zero applications where the same actor without the staff deals 371.

**Prismatic Spray's exclusion is not part of this finding and must stay.** `MAGIC-SING-019` (g) is
High and says the item-cast entry refuses id 14 outright, and `MAGIC-AUTOCAST-020` is High and says
the caster's attack is replaced by the cast rather than running beside it. A ROM1 mage with a
Prismatic staff also does nothing. The reviewer measured 28 shipped carriers at zero damage and
confirmed by removing the clause that master delivered 769 in the same control. `spec.md` states the
exclusion without citing the claim; add the citation.

## R3-B2 — the item tooltip advertises weapon-spell damage the release refuses

**Player-visible, low severity. New in this story.**

`WeaponSpellDamageFor` at `pkg/sim/spell.go:608` refuses on `!rule.Damaging || !rule.TargetsUnit`
but not on `rule.ID == 14`, while `releaseWeaponSpell` gained `|| rule.ID == 14` in this story.
`pkg/game/iteminfo.go:52`'s own comment states the two agree — "the same entity fields, spell row
and arithmetic `releaseWeaponSpell` uses" — and they no longer do. This is the defect class the
story fixed for Reniesta's 0–0 staff, inverted.

Measured on both roots:

```
m120 entity 84   spell 14 level 30   tooltip=true 10..30
m150 entity 0    spell 14 level 99   tooltip=true 21..64
m151 entity 114  spell 14 level 35   tooltip=true 10..32
```

28 such placed actors on each root, across missions 120, 130, 131, 150 and 151, two of them
owner-slot actors. **The fix is to make the reader agree with the release, not to remove the
exclusion.**

## R3-B3 — the fighter rider arm is not implemented and has no ledger row

**Latent, pre-existing, in scope and undisclosed.**

`weaponSpellFor` at `pkg/sim/spell.go:587` requires `isMage(e)`, so only the caster's replacement
arm of `MAGIC-AUTOCAST-020` exists. The fighter's **rider** arm — `MAGIC-ITEM-007`, active, High
for what the strike does: after the damage lands, `L03045` calls `Spell::Apply`, and a dead target
still triggers it when the weapon's spell is id 2 — has no implementation. `!rule.TargetsUnit` would
refuse Fire Ball in any case.

Shipped on both roots: the `Units` row `Boulder Thrower{castSpell=Fire_Ball:40}` and `:70`, placed
as mission 90 entity 152, mission 111 entity 0 and mission 140 entity 165. All have `MaxMana 0`, so
they fall through to plain blows — 594, 794 and 794 damage over 240 ticks — and never deliver the
Fire Ball ROM1 gives them.

This one is a **judgement call for the lane**: implement the rider arm, or disclose it as a typed
row. Either is acceptable; leaving it neither built nor disclosed is not, because `contract.md` puts
weapon and item releases inside the slice.

## R3-C1 — a move order on a casting actor writes a world the byte form refuses

**Rank 1. Save-breaking. Player-reachable by clicking a spell and then clicking a destination.**

A `KindMoveTo` or `KindGroupMoveTo` landing on an actor that carries a pending book cast writes a
new `TargetX`/`TargetY` and leaves the old route in place. `pkg/sim/step.go:441` writes the target
and never clears `w.routes[i]`; the move loop that would rebuild it skips that actor at
`pkg/sim/step.go:676` (`if _, casting := w.bookCastIndex(e.ID); casting { continue }`).
`MarshalBinary` writes the world and `UnmarshalBinary` refuses it at `pkg/sim/binary.go:2537`:

```
sim: entity record 0: its route ends at (2,12) and its target is (12,12)
```

The game's save is that byte form verbatim (`pkg/game/save.go:264` → `pkg/game/resume.go:271`), so
the save is written and cannot be loaded.

Matched arms — same walker, same tick count, both re-settled to zero transit before the re-order,
differing only in whether a cast is pending:

```
no cast : transit=18 pending=0 load=<nil>
mid-cast: transit=0  pending=1 load=sim: entity record 0: its route ends at (2,12) and its target is (12,12)
```

**Honesty about the root cause.** The invariant break is older than this story: at `origin/master`
the same state is reachable mid-transit, because the transit branch also skips the route rebuild —
the reviewer ran the control in a `b51b439` worktree and it fails there too, 41 consecutive
unloadable ticks at speed 6. What is **new to 1001** is a path that needs no transit at all: with
the mover owing zero transit ticks the plain re-order is loadable and the mid-cast one is not,
because the body stands down for the whole wind-up. Window width measured at 11 consecutive
unloadable ticks at `AttackCharge 12`, 80 at 40, and 240 at 200, where `castWindupTicks` caps at
255. The group arm has it too, at `pkg/sim/group.go:320`, which writes the target without clearing
the route — and that is exactly the member `spec.md` says a group order never drops. An 800-tick
AI-only drive with no move order was clean, so the trigger is a move order specifically.

The shape of the fix is one line at each of the two target writers: drop `w.routes[i]` when the new
target differs, mirroring what `invalidateBlockedRoutes` already does for the wall case. **Prove it
with a witness at both writers, and prove the older mid-transit path too or say plainly that it is
left standing.**

## Minor findings, not counterexamples

- `pkg/sim/effect.go:18` documents `ActiveEffect` as "the read-only public shape used by
  presentation and tools", and `ActiveEffects()` has no consumer outside `pkg/sim`. The contract's
  lasting-mark feedback is carried by `Entity.SpellFX`/`SpellFXSpell`, which does work; only the doc
  comment overstates.
- `max32` in `pkg/sim/spell.go` is added and never called, tests included.
- `stepAttachedEffects`' tail writes `SpellFXSpell = uint8(e.Spell)` without the `id > 0xff → 0`
  guard `markSpellEffect` states explicitly. No shipped id exceeds 28; a G2 seam only.
- A protection effect that clamped at 100 on attach, then a level rise, then expiry, leaves the
  actor below its base: `SetCombat` re-adds the stored magnitude and re-clamps, and expiry subtracts
  the full stored magnitude. Same family as R3-A1.
- `MAGIC-SING-019` (f) says Invisibility reads its own `Spell Duration` as a `> 0` gate;
  `spellPointDuration`'s case 15 ignores `rule.SpellDuration`. The shipped row is 20, so there is no
  shipped difference.

## Documentation defects, corrected by the seat in `936f9b2`

Recorded so a later reader knows they were found by review and not by chance.

- The version-53 authority paragraph above `formatVersion` called a **thirty**-byte tail four bytes
  and omitted `Protection [5]int32` at +230 through +249 entirely.
- `DIV-029`'s reason cell claimed the AI spell arm has "no `spec.md` sentence" in the same commit
  that added five of them.
- `DIV-031` typed research absence as `DEVIATION`; the file's own definition makes it `UNKNOWN`.
  `DIV-043` was `ACCEPTED` while its own revisit cell asks for an owner decision; it is `OPEN`.
- `releaseWeaponSpell`'s headline claimed the training this story removed from its body.
- The grid decoder's error named bits 2-7 as reserved after this story made bit 2 `blockMagicWall`.
- `addAreaEffect`'s comment named `placeCellEffect` as script instant 21's path; instant 21 reaches
  `landAreaCast` → `landArea`, and `placeCellEffect` has no production caller at all.
- `closure.md`'s "Owner observation" section still read "reported no visible or playable defect"
  after fifteen presentation reports and one simulation report arrived on 2026-08-15. It is marked
  superseded and kept as the record of what was believed at the round-1 push.
- `closure.md`'s quoted tool output omitted `outside-wall=1000/1000`, which is the discriminating
  half of the wall witness, and presented a selection as verbatim.
- `closure.md` cited `EXP-0176` for a technical fact where the project rule is cite-claims-never-
  experiments. It cites `MAGIC-AREARADIUS-054`.

## What the three reviewers tried and could not break

Recorded because a review that reports only its findings does not say how far it looked.

Reviewer A drove all 28 installed rows through both cast forms over three target states, 168 cases,
and found no mana-without-apply and no instrument/release disagreement; verified Control Spirit's
`Ghost` row resolves in five real missions on both roots, which was a real risk because only 56 of
119 `Units` entries parse; re-read the wall tables against the experiment's own evidence arm for
arm; and ran the campaign sweep over 28 EN maps against a `b51b439` baseline, where exactly one row
differs — mission 90, the disclosed `DIV-029` AI arm.

Reviewer B ran a byte-form and hash sweep over all 28 campaign maps, 1200 ticks each, marshalling
every 10 ticks with no refusal and no difference; cast every shipped row in a real mission world;
bound all 20 shipped instant-24 nodes and 35 instant-21 nodes on both roots; and swept every symbol
the story adds for dead code, finding only `max32`.

Reviewer C ran a 600-tick soak over 28 synthetic rows covering every arm, marshalling and
unmarshalling every tick, where only the stale-route defect fired; re-derived every gate number in
`closure.md`; re-ran every original-save witness on both roots character for character; and read all
fourteen cited claims at the pin through `tools/claim`, finding none retracted in a way that changes
what the code should do.
