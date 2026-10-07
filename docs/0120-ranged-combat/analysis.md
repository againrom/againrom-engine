# Analysis — 0120-ranged-combat

## The request

The owner: ranged units do not shoot, and the hero cannot use a bow. Two acceptance facts —
a ranged unit strikes from a distance instead of walking into contact, and the hero can equip
and use a bow.

## What was already built, and this is the finding that shapes the story

The simulation half of ranged combat **is already standing**, data-driven end to end, and was
built by `0104`. Read before assuming otherwise:

- a placed creature's reach comes from its class row's own equipment string —
  `pkg/mapload/spawn.go` `definitionFor` calls `unitReach`, which resolves the row's first
  equipment name against the `Shapes`/`Materials`/`Weapons` collections and takes its range
  column;
- a placed person's and the hero's reach come from the weapon the character carries —
  `pkg/data/recompute.go` assigns `combat.Reach = l.Weapon.Range`;
- `pkg/sim/combat.go` `approach` stops the walk at `inReach`, which is `strikeDistance <=
  Entity.Reach`, and `resolveBlow` fires there. `pkg/sim/reach_test.go`'s
  `TestAReachOfFourStrikesAtFourRefusesAtFiveAndStopsItsWalkAtFour` witnesses it;
- both target scorers already read row 0 and rewrite the distance term the moment
  `Entity.Reach > 1` (`pkg/sim/engage.go`);
- a landed blow already draws a damage numeral (`pkg/ui/numeral.go`, `0083`) and the attacker
  already plays its attack animation (`pkg/game/world.go`, the swing path).

So `groupScorerReach = 1` is **not** reversed by this story and did not need to be: a member of
reach above 1 passes that refusal because the distance rewrite one block above has already
turned its distance term into 1. `0104` DD-4's reason stands.

## What the shipped table actually says

Census of the shipped `Data.bin` `Weapons` collection, read with the tree's own `databin`
package against a lawful install; no bytes copied anywhere.

| row | `@.attackType` | `@.range` |
|---|---|---|
| every sword, axe, club, pike | 1-4 | -1, i.e. 1 |
| Short Bow / Long Bow / Crossbow | **5** | 4 / 5 / 6 |
| Sonic Beam / Boulder Thrower | **5** | 4 / 20 |
| Staff / Shaman Staff | 3 | 5 / 4 |
| Flame Thrower | **11** | 8 |

Every bow in the game is attack type **5**, which is below the ranged-arm threshold of 10, so
`ResolveWeapon` already accepts it and `Man_Bow`, `Orc_Bow`, `Goblin_Sling`, `Bat_Sonic`,
`Catapult` and `Ballista` already arrive with reach 4, 5, 6 or 20. **The premise that a bow
cannot be decoded is false.** The one row the refusal at `pkg/data/weapon.go:203` actually
costs is `Flame Thrower`, carried by the four `Dragon` classes.

## The four real gaps

1. **The ranged equip arm is a refusal rather than an arm.** `ResolveWeapon` errors on any
   attack type at or above 10, and that same error is `firstWeapon`'s "is this a weapon"
   predicate — so a row whose only weapon is ranged comes out bare rather than armed.
2. **A creature's equipment is read for its range and then discarded.** `definitionFor` takes
   only `Reach` from the row's `EquipItem` string. 26 of the 56 shipped classes carry one, so
   26 classes lose their weapon's damage, to-hit, defence and cadence — every archer among them.
3. **The hero's trained skill is a compile-time constant**, `PartySkillSlot = SkillBlade`. The
   bow literal for the shoot slot already exists and is unreachable, so the hero is a swordsman
   by construction. This is the literal reading of "the hero cannot use a bow".
4. **Nothing is drawn flying.** The `units.reg` `Projectile`, `ShootDelay` and `ShootOffset`
   keys are parsed and read by nothing, and the drawing seam carries no shot.

## Divergence taken knowingly

A ranged weapon's damage does not reach the physical damage pair in the original: it feeds a
**third damage component** with its own protection selector, which this tree does not build.
That component is declined here rather than approximated, and the reasoning is in `spec.md`
D-2.
