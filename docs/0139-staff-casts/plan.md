# 0139 — plan

## Strategy

The spell travels the road a weapon's other numbers already travel — resolve, fold, mint, re-arm —
and stops one tier short of `pkg/sim`'s string wall as an **id and a level**. The simulation then
grows one attack phase and one predicate, and the release calls the arithmetic 0127 already landed.

## Decisions

**D-1 — the name is parsed in `pkg/data`, the id is resolved in `pkg/mapload`.** `ResolveWeapon`
gains no argument (FR-1). It keeps its existing `stripSuffix` split and now *reads* what it strips,
producing `Weapon.SpellName` (the token, underscores intact) and `Weapon.SpellPower`. The token-to-id
lookup needs the `Spells` collection, which `ResolveWeapon`'s seven callers do not hold and
`mapload.Table` does. Rejected: threading `spells Collection` through `ResolveWeapon`, `WeaponFromCode`,
`ResolveShield` and `ResolveArmor` to move one lookup to where the collection is not.

**D-2 — the carrier is `data.Combat`, not a new channel.** `Combat` is the one block every armed
actor's numbers pass through: `Recompute` folds a weapon into it, `HumanDef.Combat` and
`UnitDef.Combat` produce it, and both entity mints in `pkg/mapload` read it field by field. Two
fields on it (`SpellName`, `SpellPower`) make FR-13 fall out of existing wiring for a generated
member, a placed person and a re-arm. `UnitDef` needs the same pair carried, because `spawn.go`
writes a folded block back onto the definition rather than passing the block on.

**D-3 — `FoldWeapon` ASSIGNS both, on every arm.** They join `Reach` and the cadence pair above the
melee branch, not the four additive terms below it: a ranged weapon carrying a spell must carry it,
and two weapons cannot sum their spells.

**D-4 — a fourth attack phase.** `AttackCasting` joins ready, charging and relaxing. Rejected:
reusing `AttackCharging` with a side flag, which would put two meanings on one byte and make the
byte form's cycle bound ambiguous; and releasing the spell inside `resolveBlow`, which FR-3 forbids
outright — the strike must not be entered.

**D-5 — the trigger is one predicate with one caller family.** `weaponSpell(e)` answers the spell
row when FR-2, FR-2a and FR-2b hold and nothing otherwise — FR-2a is `isMage`, which already exists
and is already derived from the mana pool, so no second caster predicate is written. `advanceAttack`
asks it at the two sites FR-3 and FR-3a name; the approach's stop arm asks it for FR-5. One
function, so the phase machine and the stop distance cannot come to disagree about who is casting,
and FR-3a's second direction costs nothing extra because the question is asked afresh at both sites
rather than remembered.

**D-6 — the second test site diverts to READY, it does not arm, and it still draws.** A charge that
expires on a now-eligible actor sets ready with nothing owed; the arm site loads the casting cycle
on the next advance. Rejected: writing the casting phase and a full count at the expiry site, which
would store a count equal to its own maximum — a state the byte form's cycle bound refuses, since
every other phase stores strictly below what it loaded.

**The jitter draw belongs to the site, not to the outcome (FR-3b).** Every advance that reaches a
zero count in a charging phase draws exactly one jitter today, a refused blow included, and the
divert takes that draw and **discards** it. `resolveBlow`'s own "both draws are taken or neither is"
is the same rule one level down: a draw skipped on a value makes the generator's position depend on
that value, and here the value would be *whether the actor is a caster* — so an actor picking up a
staff mid-swing would shift every later draw in the mission for every other unit. The discard is
written as a discard, not hidden behind a variable that is never read.

**D-7 — the release shares 0127's tail, and only the tail.** `applySpellDamage(vi, rule, power)` is
FR-8's roll, the subtraction, the health floor and the felled-clearing lifted out of `castSpell`
unchanged — it is also the whole of FR-4, because that routine already takes no to-hit roll and
subtracts no absorption, so "the cast delivers no weapon damage" is a property of reusing it rather
than a rule written twice. The commanded cast keeps **all eight of its own refusals in their own
order** above it, so the mana payment stays after the range test and a refused command still draws
nothing. The release states FR-10's five admissions of its own — row damaging, row targets a unit,
victim alive, not itself, within range — and shares no refusal with the command, because FR-9
removes two of the command's: no mana subtraction and no `knowsSpell`. FR-7's power is the entity's
own level passed straight in, with no clamp between the field and the arithmetic. FR-6 needs no
code: the release reads the pair and writes neither, so nothing consumes the weapon. FR-11 is the
same `chargeTicks`/`relaxTicks` pair the strike already loads, reached from the same two lines.

**D-8 — the entity carries an id and a level, and the range is looked up.** `WeaponSpell uint16`
(zero is "none", matching the reserved row-0 subscript) and `WeaponSpellLevel int32`. FR-12's
admission distance comes from the world's own spell table at the moment it is needed, so the row and
the actor cannot drift apart. Rejected: caching the range on the entity, which would be a third copy
of a column the world already holds. FR-1b's token never reaches this tier at all — an id is what
crosses it, which is what `pkg/sim`'s no-strings rule requires and what the source says the runtime
binds anyway.

**D-9 — `inReach` is untouched.** It is the melee test and FR-5 leaves it on the melee path. The
approach's stop arm calls a new world-level `closedOn(i, ti)` that answers the cast admission for a
caster and `inReach` otherwise. The exported `InReach` keeps its signature and meaning.

**D-10 — `formatVersion` 41.** FR-15's three obligations are met by putting the pair in the record
and the phase byte in `AttackPhase.defined()`: the digest is FNV-1a over the byte form, so entering
the form *is* entering the digest, and `attackFault` is the one predicate both the constructor and
the decoder ask, so a casting cycle no tick could leave is refused in one place rather than two.
The version tests are already version-free by name and read the constant, so only the constant and
the record's own layout move.

**D-11 — FR-14's headless door is one flag on `missionrun` and one exported call.**
`game.MissionPartyAs(mage bool, w, list, t)` takes the class axis the caller names, resolving the
mage arm's own weapon off the table when `mage` is true; `MissionParty` becomes it at the authored
default, so no existing caller changes. `missionrun -mage` passes true. Rejected: a new tool, and
exporting the chargen internals.

**D-12 — a malformed attachment is silent.** FR-1a's refusals leave the pair zero and the weapon
otherwise whole; nothing logs and nothing errors, because `ResolveWeapon`'s existing failures are all
"this name resolves to no row" and an attachment is not that.

## Files

| File | What moves |
|---|---|
| `pkg/data/weapon.go` | `Weapon.SpellName`, `Weapon.SpellPower`; `ResolveWeapon` reads what `stripSuffix` cuts |
| `pkg/data/itemparse.go` | `takeCastSpell` — the attachment parse, beside the one prefix walk |
| `pkg/data/hero.go` | `Combat.SpellName`, `Combat.SpellPower` |
| `pkg/data/foldweapon.go` | both assigned above the melee branch |
| `pkg/data/unitdef.go` | `UnitDef` carries the pair; `UnitDef.Combat` emits it |
| `pkg/sim/combat.go` | `AttackCasting`, both trigger sites, the cycle bound, `closedOn` |
| `pkg/sim/spell.go` | `weaponSpell`, `releaseWeaponSpell`, `applySpellDamage` |
| `pkg/sim/world.go` | the two entity fields and their doc |
| `pkg/sim/rearm.go` | `CombatBlock` grows the pair; `SetCombat` writes it |
| `pkg/sim/binary.go` | the pair in the record, `formatVersion` 41 |
| `pkg/mapload/spell.go` | `spellIDByToken` — the underscore substitution and the row search |
| `pkg/mapload/start.go` | a started member's pair |
| `pkg/mapload/fromalm.go` | a placed actor's pair |
| `pkg/mapload/spawn.go` | the folded pair written back onto the definition |
| `pkg/game/world.go` | `rearm` carries the pair into `CombatBlock` |
| `pkg/game/hero.go` | `MissionPartyAs`; the mage literal regains its attachment |
| `cmd/missionrun/main.go` | `-mage` |

**D-14 — an item code cannot carry an attachment, so the re-arm prefers the object it already
holds.** `WeaponFromCode` recomposes a name out of three collection indices; no field of a code
holds `{castSpell=…}`, so a weapon re-resolved from a code carries no spell however it was first
built. `rearm` therefore uses the party's **start weapon** when the slot-1 code equals that weapon's
own composed code, and re-resolves only when the code names something else. This is not a patch
around the round-trip: the start weapon *is* the object that code was composed from, and going
through the code is the lossy direction. What is left is a real limit, disclosed at spec D-5: a
staff acquired as a bare code — from the ground, from a corpse — casts nothing.

**D-13 — the mage literal regains the half that was dropped.** The chargen mage arm carries
`Wood Staff` bare, with the attachment cut and disclosed, because nothing could hold it. Something
can now, so the constant becomes the shipped literal whole. The high-tier arm stays absent: which
arm a shipped campaign takes is unresolved and this story does not resolve it.

**D-15 — FR-16's award is a second CALL SITE, not a move into the shared tail.** The obvious reading
of "both paths now train" is to push the award down into `applySpellDamage`, the tail D-7 already
shares. It cannot go there: that function is handed the **victim's** index and the row, never the
caster's, so it has no awarding entity to name, and widening its signature to carry one would make
the shared tail carry an argument only one of its two obligations uses. The award therefore sits at
the end of `releaseWeaponSpell`, beside the damage call and past all five of FR-10's refusals, which
is the same shape the commanded cast and the landed blow already have. One sink, three feeds, each
stating its own award — unchanged from the arrangement the skill-award story landed.

Two consequences worth naming. A **refused** release trains nothing for free: the refusals are
early returns above this line, so no clause has to say so. And the credited slot needs no new
narrowing: only a carrier reaches a release at all — the trigger's own predicate is a mana pool —
so the sink's carrier arm takes the award's named school rather than falling back to the weapon's
slot.

## Risks

- **A placed shaman now casts.** 25 shipped `Humans` rows and 2 `Units` rows name a spell-carrying
  staff, so missions change wherever such a row is placed *and* the row carries a mana pool. This is
  the behaviour, not a side effect, but it moves the campaign — the mission-10 outcome is measured
  before and after and reported either way.
- **The stop distance widens for a caster.** An actor that used to close to contact now halts at
  spell range, which changes where it stands and therefore what its group's engagement sees.
- **The digest moves for every world.** Two new fields on every entity change every hash, so every
  pinned digest in the existing tests is expected to move; a test whose digest did *not* move would
  be the finding.

## Success criteria

- Every FR has a test that fails when its line is reverted.
- `SC-2`: the existing `pkg/sim` suite passes unchanged except for pinned digests and form versions.
- `SC-1`: observed on both lawful roots through the new flag, with the numbers recorded.
