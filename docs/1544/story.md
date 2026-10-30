# Projectile records go one way

## Intent

The World flight producer (`pkg/sim/flightproducer.go`) and three window
producers (`flight.go`, `unitshot.go`, `spellbolt.go` in `pkg/game`) each
turned a launch into a projectile record, so a correction to launch, mark or
scheduling had to reach both. This story makes the World flight producer the
one producer of a projectile record in both games. The second game's
differences become profile data of that producer.

## Authority

- Owner direction: the second unification after the World carrier set
  (story 1542), ahead of the campaign projection.
- Audit shape: one event-to-record producer, with the second game's release
  clock as profile data (rank 8 of `pipeline/reviews/arch-audit-a5c60f20.md`,
  rank 6 of `arch-audit-91ad8a33.md` and `arch-audit-3d1771a7.md`, section 4
  of `arch-audit-80cd3505.md`, all in the seat).
- No ROM1 or ROM2 behaviour changes. DIV-2656, DIV-2663, DIV-2683 and
  DIV-3363 keep their text; only their code anchors moved.

## As built

### One producer

The World flight producer's `produceCast`, `produceScriptCast`,
`produceAreaBursts`, `produceUnitShots` and `releaseShot` build every record.
The window producers, the launch geometry helpers beside them and the switches
`gameFiles.recordTrailFlight` and `startRecordAtCaster` are removed. Their
launch geometry (cast start, class shoot offset, selection delta, swing tick,
shot offset, the caster's cell and the cell centre) lives in `FlightClass`
with the same arithmetic.

| Launch site | Calls now |
|---|---|
| First game, every family | the in-step producer, timing unchanged |
| spell bolt seen by the window | `World.ReleaseCasts`, per event |
| scripted flight | `World.ReleaseScriptCasts`, per event |
| staged cells | `World.ReleaseAreaPaints`, per paint |
| unit shot | `World.ReleaseClientShots` with the window swing clock |
| trail flight | the start kept from the record the World built |

In the first game the hand-back calls return nothing, because the step has
already produced the records.

### Profile data

`gameFiles.flight` holds a `sim.FlightProfile` with the window's registry and
class inputs. The first game sets only the registry rows. The second game sets
`ReleaseAtClient`, `StartFromMovedCaster` and `ShotsNeedArt` and its drawn
class. The shot clock (`ShotClock`) is an input of the producer; the producer
decides every other condition of a shot.

### Unchanged

The SAV write (`savedProjectileToSav`, `sav.SetProjectiles`), the LOAD paths,
the window painters and the cast and script-cast looks build no record and are
not changed.

### Guard

`internal/archtest` `TestProjectileRecordsHaveOneProducer` refuses a non-test
call outside `pkg/sim/` to `ReleaseCast`, `ReleaseUnitShot` or
`ReleaseAreaBurst`.

## Proof

- `TestFlightRecordsMatchTheirPinnedRuns` (`pkg/game/flightwitness_test.go`):
  two games times eight families (shot, book, staff, staged, fire ball,
  client lightning, client chain, moved teleport), 40 ticks each, every tick
  digested over the store, World hash, drivers, SAV projectile rows, trails and
  looks. The digests are pinned from the base tree `a412b87e`; base and
  candidate print the same 16 digests.
- `TestFlightProfilesAreGameData`: the profiles of both games, and the swing
  clock releases a shot in the second game only.
- Loss controls on the EN root: the SAV round trip of the saves corpus, 213
  PASS on base and candidate with the same subtest set; the projectile corpus
  audit (`TestProjectileCorpusAudit1133`) prints the same line on both.
- The adversarial review (`pipeline/reviews/story1544-adversarial-review.md`
  in the seat) reran the witness on base and added 30 runs of 15 further
  families on both games, equal on base and candidate.
- Release tests for the next release, EN and RU: the flight, fresh-shot,
  fire-ball burst, flight producer, flight records, bat projectile, hero
  appearance, second-game spell, spell launch, bolt figure and milestone-2
  projectile tests listed in the lane's return.

## Open debt

- The guard names three methods; a renamed bypass would pass it.
- In the second game a record appears only when the window hands a World
  event to the producer, as on the base.
- The ported `TestUnitShotStartPairFollowsTheSixteenWayDir` no longer checks
  the shooter's start point or a class without an offset array.
- The second game's hero class taken from the mission roster is checked by
  reading only.
