# A unit shot in flight follows the original's leaves

## Intent

A ranged shot in flight keeps the leaves the original keeps: its direction
tracks the target on every driver call, it starts at the class's release
point for the shooter's sixteen-way facing, its id comes from the original's
counter, a shooter that dies before its blow lands does no damage, and the
smoke trail behind pictures 10 and 12 is the original's pre-move points.
Known defect C4: a native shot in flight at SAVE carried authored leaves.

Base: `398b5fe4` (game 0.107.0). Knowledge pin k217.

## Authority

| Leaf | Authority |
|---|---|
| sixteen-way direction helper | ANIM-138 (High) |
| per-call direction recompute, dir from actiondir | ANIM-139 (High) |
| trail append, six points, oldest dropped | ANIM-140 (High) |
| start point and pair index | SAV-1188 (High / Medium for the scale) |
| projectile counter | SAV-1189 (High / Medium), SAV-1190 (Medium), SAV-1191 (High) |
| dead shooter's pending blow | SAV-1192 (High) |
| no trail in SAVE, empty at LOAD | SAV-1193 (High) |
| leave-map countdown | SAV-1194 (High / Medium) |

## As built

| Leaf | Engine before | Engine now |
|---|---|---|
| Direction helper | `terrain.EffectFacing`: an even sixteen-way split by tangent constants, zero vector south | `sim.ProjectileDirection` is ANIM-138: signed 32-bit slope comparisons, wrapping products, zero vector 4. `game.EffectFacing` is that helper in the sheet wheel (`(dir-8)&15`); the terrain split is removed |
| Direction in flight | dir and actiondir set once at release from the start-to-target vector, never recomputed | each action-1 driver call that resolves the target (a live actor or a placed original body) sets actiondir from the shot's pre-move point to the target's current point; every action-1 call copies actiondir to dir. A detached target keeps the last direction |
| Start point | pair chosen by the eight-way rounded facing (`FacingDir`), so odd sixteen-way facings took the next pair | pair `((facing>>4)-8)&14`, the cast producer's own pair; both producers read `classShootOffset` |
| Counter | LOAD from FreeIndex low u16, SAVE writes it, each insertion adds one; a mission built from a map starts at 0 | unchanged; the mission boundary is DIV-1784 |
| Dead shooter | the tick skips an actor that is not alive and the death clears its attack, so a pending blow never lands | unchanged; now witnessed |
| Trail | presentation positions after each move, newest first, kept only for shots released in this session | the pre-move point of every driver call, oldest first, at most six, for every armed record of picture 10 or 12, restored records included; a LOAD starts empty. The draw takes frame 0 for the newest point |
| Record draws | one list of record and trail draws | `savedProjectileDraws` is the records alone, the persisted seam the M2 projectile instrument compares; the scene interleaves each record's trail |
| Off-map actor | the attack loop and the turn step skip an off-map actor, so its countdown stands | unchanged; matches SAV-1194 |

One direction helper: `sim.ProjectileDirection`. Callers: the projectile
driver (`sim.stepSavedProjectile`), and through `game.EffectFacing` the cast
and presentation-shot draw (`game.spellDraw`) and `cmd/spellcheck`. The cast
draw used a different split (equal 22.5-degree sectors, zero vector south)
and now uses this one. The
actor heading helper of AI-361 (eight byte headings, zero vector 224) is a
different routine: its outputs at (3,-1) and (2,-1) are 64 and 32 where
ANIM-138 gives 3 for both, so it does not derive from the sixteen-way helper
and DIV-1769 is unchanged.

## Divergence rows

- DIV-1781 closed: ANIM-138, ANIM-139.
- DIV-1785 closed: SAV-1192, SAV-1133; the removal policy stays DIV-1115.
- DIV-1782 revised: SAV-1188; remaining is a class with no ShootOffset array.
- DIV-1784 revised: SAV-1189, SAV-1190, SAV-1191; remaining is the mission
  boundary.
- DIV-1786 revised: ANIM-140, SAV-1193; remaining is the trail frame.
- DIV-1721, DIV-944 revised to name the answered leaves. Target altitude
  (ANIM-139 copies the target's altitude to actionz) stays DIV-944's: the
  engine arms only zero-altitude records.
- No new row: every remaining difference has a row.

## Proof

Focused tests:

- `sim.TestProjectileDirectionSplitsAtTheClaimedSlopes`: the four slope
  boundaries and the vector past each in all four quadrants (32 vectors), the
  axes, the zero vector and a wrapping vector.
- `sim.TestUnitShotDirectionFollowsTheTargetEachCall`,
  `sim.TestDriverCopiesActionDirToDirOnActionOne`: re-aim per call, the
  detached target, the dir copy on an unresolved call.
- `game.TestUnitShotStartPairFollowsTheSixteenWayDir`: every sixteen-way dir
  at both ends of its facing-byte range, the pair index, the shared helper,
  a class with no array.
- `sim.TestProjectileCounterCountsInsertionsAcrossTheBinaryForm` and the
  existing `TestReleaseUnitShotAllocatesFromTheCounterIntoBucketOrder`:
  insertion, wrap, binary round trip, 0 for a world built from a map.
  `TestReleaseFreshShotWritesProjectileRecord` checks FreeIndex through SAVE
  and cold LOAD.
- `sim.TestAShooterKilledDuringItsCountdownAppliesNoDamage`: HP 0 and -3
  one step before the blow, against a control run whose blow lands.
- `sim.TestAnOffMapAttackerKeepsItsCountdown`.
- `game.TestASmokeTrailIsThePreMovePointsOldestFirst`: a loaded picture-10
  record, nine calls, the six-point window.

Install witnesses (EN and RU):

- `TestReleaseUnitShotTracksAMovingTarget` (new): the hero walks across
  mission 41's archers; a shot at him changes dir in flight, SAVE on its build
  tick and a cold LOAD continue the same leaves tick for tick; a SAV with a
  changed actiondir fails.
- `TestReleaseOriginalProjectileFlightContinuesToLaterSaves`: the owner's EN
  mission-150 saves with a rock, the burst and a bolt; the loaded World
  reaches each later save's records leaf for leaf, the bolt it releases itself
  included.
- `TestReleaseRestoredProjectileDrawsItsSavedFacing`: now also checks one
  trail point before the SAVE and none after its LOAD.

Hashes: no World golden moved in `go test ./...`.

Gates on the merged head:

- `gofmt -l` clean; `go test -trimpath -count=1 ./...` exit 0.
- `check-release-tests.sh`, RU then EN: 937 of 983 ran and passed on each
  root; 46 lacked a subject (ROM2 subjects on a ROM1 root, cold-process
  children).
- `check-milestone2-acceptance.sh`: the same 48 FAIL lines as main
  `2884d094` on each root (the saveorcsdontgo subtests,
  `TestSAVRoundTripGateNewGameKits`, `TestSAVWriterCensusChangedWorlds`),
  and the same 16 writer-census mismatches.
- `TestReleaseSecond*` on rom2-en and rom2-ru, eight groups: 42 pass, 0 fail,
  0 skip per locale.
- Headless scenarios 1194, 0155 (three), 1193, 1087, 1089: 7 of 7 on EN and
  RU.
- `check-no-game-assets.sh` clean.

## Open debt

- DIV-1782: a class with no ShootOffset array releases from the shooter's
  point; the original window reads the array without a bound.
- DIV-1784: a new mission starts the counter at 0; SAV-1190 (Medium) finds no
  reset in one process.
- DIV-1786: which trail frame each point draws.
- DIV-1783 (admission) is not answered by the pin.
