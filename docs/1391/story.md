# One object in flight

## Intent

Every object in flight is one World record kind and reaches the SAV (owner
decision on the architecture audit, row 3). Before this story a unit shot at a
unit and the Fire_Ball burst were World records; a cast, a staff's bolt, a
staged burst and a shot at a structure were presentation objects in the map
world, so a SAVE taken while one flew wrote nothing and a LOAD lost it. Known
defects C4 and C5.

Base: engine main `6e752c3a`, knowledge pin k229.

## Authority

| Leaf | Authority |
|---|---|
| cast producer: picture 2 × spell + 8, two records for 60, segment switch, 0-segment records die in the building pass | ANIM-147 (High / Medium), SAV-1204 (High / Medium) |
| odd records: Fire Ball's 13 and the staged 17, 27, 51; no record for the bursts of spells 3, 7, 8, 12, 17, 19 | ANIM-148 (High) |
| each picture's driver arm: Teleport writes no position, Heal and Drain take phase 1 on a found target, the burst's last call | ANIM-149 (High / Medium), ANIM-150 (High) |
| dir and actiondir at the cast producer; 0 for the client arms | ANIM-144 (High) |
| actiondir write only on a resolving call; none on a 0-segment call | ANIM-143 (High) |
| client arms 0x8b and 0x8c build their records directly | SAV-1199 (High) |
| shot at a structure is an ordinary unit-shot record | SAV-1197 (High / Medium) |
| class picture 13 and above admitted | SAV-1198 (High) |
| trail frame i for point i, Palette gate | ANIM-142 (High) |
| a staff's release is cast-diverted | SAV-1129 |
| a siege or weapon-spell rider builds no cast record | ANIM-115 (High) |
| SAVE writes every record's 16 leaves; LOAD loses the word list, point array and trail | SAV-1202 (High) |
| Fire Ball burst at the cell centre, no stored size | MAGIC-289 (High / Medium) |
| projectile counter writers | SAV-1200 (High / Medium), SAV-1201 (Medium) |
| launch point, homing | MAGIC-261, MAGIC-288 |

## As built

**The record kind.** `sim.SavedProjectile` (the 16 `Prj` leaves) with its
`sim.SavedProjectileDriver` row is the one record kind. `stepSavedProjectile`
is its one driver, with an arm per picture class: travel (10, 12 and the unit
shots), attached snap (18, 20, 30 and the other attached pictures), the path
ramp (34, 36), the phase clocks of 51 and 60, and the standing Teleport arm.
The driver row gains a structure-target flag (binary flag bit 8).

**Producers.** All build through `insertProjectile`, which takes the id from
the shared counter and runs the record's first driver call on its release
tick.

| Producer | Builder | Record |
|---|---|---|
| unit shot at a unit | `World.ReleaseUnitShot` | unchanged |
| unit shot at a structure | `World.ReleaseUnitShot` | actiontarget is the structure id, aim at the anchor cell centre (DIV-2846); any picture of 1 or more |
| cast from a book or a staff | `mapWorld.releaseCast` → `World.ReleaseCast` | picture 2 × spell + 8 at the caster's point plus the class launch offset, homing on the target when the picture's row homes, else aimed at the target cell's centre; Teleport builds a second record at the destination |
| client arms 0x8b, 0x8c | `mapWorld.releaseScriptCast` → `World.ReleaseCast` | Lightning 5 segments, Prismatic Spray 13, from the source cell's centre, actionphase -1, dir 0 |
| staged burst | `mapWorld.releaseAreaBursts` → `World.ReleaseAreaBurst` | picture 2 × spell + 9 at each accepted cell for spells 4, 9, 21, actionphase -1 |
| Fire_Ball burst | `insertBurst` → `insertBurstRecord` | unchanged |
| imported SAV record | `ImportOriginalProjectiles` | unchanged |

A rider release builds no cast record. The presentation tier keeps, per
record id, only what the record has no field for (`flightLook`: a path
figure's seed, its link tag and the victims a Prismatic Spray links); a LOAD
starts without it, as the original's LOAD loses the word list.

**Draw.** `savedProjectileScene` is the one draw producer: each armed record
from its position, direction and phase, a path record as its figures with
the frame from the record's own phase, and the smoke trail behind pictures 10
and 12 for a row whose Palette is not 0. The removed presentation lifecycles:
`spellBolt` lists, `spawnCast`, `spawnBurst`, `advanceBolts`,
`weaponBoltDraws`, the unit-shot `flying` list and `unitShotDraws`. A legacy
visual snapshot's bolt list is read and dropped.

**What the player sees change.** A spell, burst or shot in flight survives
SAVE and LOAD. A staff's bolt is drawn from its release, not across the
wind-up. A cast no longer draws a burst at the target for spells 3, 7, 8, 12,
17 and 19, and a siege rider no longer draws a cast bolt. The trail's newest
point draws frame 5 of the smoke sheet, not frame 0. A cast with no
homing, such as a Fire Ball, carries the direction of its flight as its
record's dir (DIV-2850), so it is drawn as on main.

## Rows

Closed: DIV-1786 (trail frame and Palette gate), DIV-1878 (pictures 14 and
above). Revised: DIV-1783, DIV-1784, DIV-1877, DIV-1879, DIV-944, DIV-1721,
DIV-1453, DIV-2686, DIV-2658. Added: DIV-2846 (structure aim point), DIV-2847
(cast segment distance), DIV-2848 (structure target after LOAD), DIV-2849
(loaded record's derived cell), DIV-2850 (dir of a cast with no homing). Reviewed and unchanged: DIV-1782 (the
remainder is `SAV-1196`'s Unknown), DIV-1876, DIV-939. DIV-2851 to DIV-2853
are returned unused.

## Proof

Focused: `pkg/sim/flightrecord_test.go` (structure shot, cast record, homing,
client and burst records, Teleport), `pkg/game` tests ported from the
presentation lists to the records (`flight_test.go` helpers).

Installed, EN and RU, `pkg/game/flightsave_release_test.go`: for a book cast,
a staff release, a staged burst and a structure shot, SAVE in flight writes
the record's leaves, a cold LOAD continues it leaf for leaf to the same
retirement tick, and the target's hit points or the structure's health match
the uninterrupted run on every tick for 48 ticks; a SAV with the record
dropped or one leaf changed fails the same check. The unit shot at a unit and
the Fire_Ball burst keep their witnesses
(`TestReleaseFreshShotWritesProjectileRecord`,
`TestReleaseNativeFireBallBurstSurvivesSaveAndLoad`).

A retired record's driver row leaves the World at once, so a SAVE and its
LOAD hold the same rows. A World that carries projectile records and no area
driver retires its Document's area rows, as a World with no drivers does.
Without the projectile registry a loaded record is armed with no phase count,
as the live producer builds it from the same install.

Release witnesses moved by a named cause:

| Witness | Cause |
|---|---|
| `TestReleaseMilestone2Projectiles1157` | a Heal cast in mission 18 builds its own record (`ANIM-147`); the source oracle stops at the first new record and the source record is read by its id |
| `TestReleaseRangedShotsDrawTheirInstalledSprites/staff` | a staff's record is built at the release (`SAV-1129`), so the wind-up draws no firebolt; a cast on the same tick takes the newest id, so the shot is read by picture |
| `TestReleaseLightningLeavesTheStaffTipAndTheHandInEightDirections` | the figure is drawn from the record's fifth call and the record's cell; the launch is measured from the caster's cell |
| `TestItemObjects1115ScrollSessionNativeContinuation` | the scroll's cast builds a record; the fixture has no registry and LOAD now arms it as the live producer does |
| `TestEveryMapWorldFieldIsRuled` | `mapWorld.bolts` is gone; `flights` is ruled cosmetic in `docs/0143-save-and-load/spec.md` FR-5 |

Gates on the candidate before the last main merge: the full asset-free
suite; the ROM1 release set on RU and EN (1011 gated tests; no failure but
the environment-gated tavern witnesses); 53 of 54 headless scenarios on each
root, the remaining `1060-campaign-130-reachable` failing on main with the
same hash; the milestone-2 instruments with main's failure set; 47 of 47
second-game witnesses on each locale.

## Open debt

DIV-2846 to DIV-2850. The installed bow does no damage to a structure, so
the structure witness's effect is the unchanged health and the landing tick
is the record's retirement. Fire Sacrifice does not hurt the party fighter, so
the staged-burst witness's effect check compares a constant hit-point series
("effect 145 changed on tick -1"); it rests on the record's leaves and
retirement tick, which are compared on every tick.
