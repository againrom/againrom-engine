# Spell launch point and projectile light

## Intent

Two player results. A cast's flying object leaves the caster's staff tip or
hand for his class and facing, not his cell centre plus an authored hand
offset. Lightning, Prismatic Spray, Fire Arrow, Fire Ball and the Fire Ball
explosion light the ground and the units around them while they fly, under the
Dynamic lighting option as the original gates it.

Base: `ead1eb9f` (game 0.100.5). Knowledge pin: k205, moved to k207.

## Authority

| Part | Claims |
|---|---|
| launch point `A+8*(ShootOffset-Center)`, Selection fallback, `A` with TileSize | MAGIC-261 |
| hero class from equipment: `mage` 23, `mage_st` 24 | MAGIC-262 |
| human geometry, three ShootOffset arrays, eight-direction deltas | MAGIC-263 |
| creature casters with and without ShootOffset | MAGIC-264 |
| Teleport fallback and destination-derived second object; message arms at cell centres | MAGIC-265 |
| simulation SpellTransport starts at caster Position | MAGIC-266 |
| SAV stores the current point, no launch point | MAGIC-267 |
| light pictures 10, 12, 13, 34, 36 | MAGIC-269 |
| path stamps `u8(10*phase)` on four vertices per cell, overwrite | MAGIC-270 |
| point helper footprint; Fire Arrow r0, Fire Ball r1 at 16; explosion phase table | MAGIC-271 |
| per-frame rebuild; phase sequences of normal and direct construction | MAGIC-272 |
| Lighting gates point stamps and terrain; the unit merge reads no option | MAGIC-273 |
| client draw state only; SAV omits the light grids | MAGIC-274 |
| Wall of Fire radius-1 stamp with a 0 or 12 flicker | TERR-LIGHT-061, MAGIC-UNITLIGHT-057 |

The partial retractions of MAGIC-CASTSPAWN-033 (caster-centre placement),
MAGIC-UNITLIGHT-057 (square radius) and TERR-LIGHT-061 (sole stamp writer)
retire the code that relied on them: the authored hand table
(`boltHandOffset`), the hand lift (`spellHandLift`), the 3x3 Wall of Fire
splat in `spellLightingCells`, and the comments in `towerglow.go` that cited
that splat's loop shape.

## As built

### Launch point

- `castLaunch` (`pkg/game/castlaunch.go`) returns the offset from the caster's
  cell centre in `ui.ShotScale` units: `128*(TileSize-1)` on both axes, plus
  `8*(ShootOffset[i:i+2]-Center)` with `i=((facing>>4)-8)&14`, or
  `trunc((Selection2-Selection1)/2)-Center` for an empty array and for
  Teleport. One class pixel is one map pixel (DIV-2656).
- `casterClass` reads the class `spellClientClass` selects: for a hero, the
  equipment-derived class (DIV-2657); for a creature, its own class row.
- `spawnCast` stores the offset as `spellBolt.launch`. Every flying picture,
  its trail and the path figure start there (`castOrigin`, `castShotPoint`,
  `boltPath`). A staff's live wind-up figure (`weaponBolts`) and a unit shot at
  a structure use the same rule. A direct-route path object keeps the cell
  centre (MAGIC-265's message arms).
- Teleport's second object stands at the destination plus the Selection delta
  (DIV-2658). A multi-tile caster adds the TileSize term to its anchor cell
  centre (DIV-2663).
- Hashed state does not change. The cast objects are presentation records;
  the simulation transport starts at caster Position (MAGIC-266).
- The visual snapshot carries `SnapshotSpellBolt.Launch`. An older envelope
  restores a zero offset, so an object in flight at that SAVE departs from the
  cell centre for the rest of its life. World projectile records keep their
  current point (MAGIC-267).

### Projectile light

- `objectLightStamps` (`pkg/game/spelllight.go`) builds the stamp list on every
  push: cast objects, armed SAV projectile records, the staff wind-up figure,
  unit shots, then Wall of Fire. A later stamp at a vertex overwrites
  (DIV-2661).
- Pictures 34 and 36 stamp the four vertices of the cell of every drawn figure
  point at `10*phase`, phases 4,3,2,1,0,1,2,1,0,1,2,3,4 by age; the five-call
  direct route takes the first five (DIV-2659).
- Pictures 10 and 12 stamp radius 0 and 1 at level 16; picture 13 follows the
  explosion table by its drawn phase. Wall of Fire stamps radius 1 at its
  flicker level. These are point stamps.
- `ui.Viewer.SetLightStamps` (`pkg/ui/objectlight.go`) holds the grid. With
  Dynamic lighting on, a stamped vertex shades at the brighter of stamp and
  terrain, and a unit on a cell with a stamped corner takes the merged level.
  With it off, point stamps are dropped, path stamps still light units, and
  terrain ignores stamps. The bit-set branch is not built (DIV-2660). The
  composition onto the gain-scale renderer is DIV-2662.
- `ui.Viewer.HeadlessMapFrame` composes a map window on the CPU from the
  production terrain triangles and art submissions, for release witnesses.

## Proof

Focused tests:

- `pkg/game`: `TestCastLaunchHumanClassesInEightDirections`,
  `TestCastLaunchTeleportTakesTheSelectionFallback`,
  `TestCastLaunchCreatureCasters`, `TestACastObjectLeavesTheCastersStaffTip`,
  `TestACastObjectInFlightKeepsItsLaunchAcrossTheVisualSnapshot`,
  `TestAProjectileInFlightKeepsItsCurrentPointAndLightAcrossSAV`,
  `TestLightningAndPrismaticLightTheirPathPerPhase`,
  `TestADirectLightningLightsFivePhases`, `TestPointLightFootprints`,
  `TestFireArrowFireBallAndExplosionLight`,
  `TestObjectLightLeavesTheWorldUnchanged`,
  `TestLiveWallOfFireStampsTheLightGridAndRestoresOnUpdate`.
- `pkg/ui`: `TestALightStampTakesTheBrighterOfStampAndTerrain`,
  `TestTheUnitMergeOfLightStamps`, `TestDynamicLightingOffLightsUnitsAndNotGround`,
  `TestABoltsDepartureCarriesNoHandHeight`.

Release witnesses, EN and RU:

- `TestReleaseLightningLeavesTheStaffTipAndTheHandInEightDirections`: the
  installed mage with a staff takes class 24 and bare-handed class 23; the
  installed geometry equals MAGIC-263; a Lightning cast observed at each of
  the eight facings starts at the claimed delta; one cast ordered at a unit
  starts at the delta for the facing the order turned him to.
- `TestReleaseALightningBoltLightsTheGroundAndTheCaster`: at phase 0 the ground
  is brighter with the stamps; the caster's sprite scale is the same with
  Dynamic lighting on and off; with it off no ground pixel outside the caster
  changes.

## Open debt

- The drawn Lightning frame ramp (`boltRampPhase`, DIV-081) is still authored.
  MAGIC-272 publishes the phase sequence the light follows; drawing frames
  from it is not in this story.
- A loaded picture 34 or 36 SAV record is neither drawn nor lit (DIV-2661).
- The first rendered Teleport point, cast-time class freshness, the pixel
  scale and the multi-tile anchor remain Unknown (DIV-2656, DIV-2657,
  DIV-2658, DIV-2663).
