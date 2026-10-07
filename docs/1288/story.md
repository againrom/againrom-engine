# Story 1288: unit-shot build tick and Fire_Ball transport target

## Intent

Re-read the open rows on the tick a ranged unit's shot record is built and on
the Fire_Ball transport (DIV-1876, DIV-1877, DIV-1879, and DIV-944, DIV-1878,
DIV-939 where touched) against the claims promoted since they were written,
implement what the claims establish, and state the rest as open questions.

## Authority

Pin k123. Claims read: `ANIM-113`, `ANIM-114`, `ANIM-115`, `SAV-1153`,
`SAV-1154`, `SAV-1155`, amended `SAV-1143` and `SAV-1149`. B1 holds: no
question below carries an expected answer.

- `SAV-1153` (High / Medium): a unit shot's first driver call is on its
  creation tick; the three saved shots fit creation = damage tick minus
  (charge - ShootDelay + flight term). Medium for the offset.
- `SAV-1155` (High / Medium): the Fire_Ball transport countdown is the
  truncated Euclidean distance of cell * 256 + fine words, caster to target
  point, over spell parameter 7; a target of size byte 1 or less is its cell and
  fine bytes. The multi-cell and null-target branches are not characterised.
- `ANIM-115`: the siege rider builds the transport, sends the damage message
  in its own tick and leaves the burst to the inner effect.
- `SAV-1154` (Medium / Unknown): the code chain puts the burst record 3 ticks
  after the rider tick (2 if another effect follows the transport); the saved
  case shows 4.

## As-built behaviour

- Unit shot build tick (`pkg/game/unitshot.go`). A shot at a unit is built when
  the swing count equals ShootDelay less one, not ShootDelay. The swing count's
  zero is the tick the attacker loads its wind-up, one after the original's
  inferred swing start; the build now lands on the original's creation tick. A
  class with ShootDelay 0 is built on the swing's first tick with two driver
  calls (`UnitShot.Late`), because the original created it on the tick before.
  A shot at a structure is a presentation object and keeps its former tick.
- Restored wind-up (`pkg/game/spellbolt.go`). The swing count seeded from an
  original action clock is one lower, the count a native run of the same
  wind-up has at that tick; the clock counts from the original's swing start.
- Fire_Ball transport countdown (`pkg/sim/spelldelivery.go`). A cast or weapon
  rider aimed at an actor whose footprint is one cell times the transport to
  that actor's cell and fine position (restored fine position or native stride
  point). A larger actor and a cast at a cell keep the aimed cell's centre.
  Native actors are cell-centred outside a stride, so a native cast changes
  only mid-stride; a restored actor changes wherever its fine position is off
  centre. The size byte is read as the footprint side.
- Burst record timing is unchanged. The claims give the record on the third
  tick after the rider tick and the saved case shows the fourth; the surplus
  stays placed at the build.

Rows: DIV-1876 and DIV-1879 narrowed, DIV-1877 narrowed (metric and size-1
target point implemented, timing surplus open), DIV-944 sentence updated for
the seed, DIV-1783 cites `ANIM-115` for the rider's object. DIV-1878 and
DIV-939 are untouched: no new claim names a picture 14 record or a retained
world-effect lifetime. DIV-2004 to DIV-2009 stay unused.

## Proof

Installed EN data, owner saves `game0022` to `game0024`:

- `TestReleaseOriginalProjectileFlightContinuesToLaterSaves`: the record set
  of the World loaded from game0022 or game0023 equals the later save's
  Projectiles, now including the shots the loaded World releases itself: rock
  id 28 equals game0023 at tick 3095 and game0024 at its tick, bolt id 30 equals
  game0024 (before this story the shots were compared one tick late). Loss
  controls: the set one tick before or after fails, the shot one tick after no
  longer equals the original, a changed leaf fails. With the build tick
  reverted to the swing count ShootDelay the test fails on records 28 and 30.
- `TestReleaseSiegeRiderQueuesItsTransportWithTheBlowBeforeTheBurst`: the
  loaded catapult's rider lowers its target's hit points and queues the
  transport on tick 3098, the transport is the one pending delivery until
  3101 with no burst, and the burst exists on 3102.
- `TestReleaseLoadedWorldShotAndSiegeTransportContinueAfterColdLoad`: SAVE at
  +6, +12 and +22 after the game0022 load writes the shot, the pending
  transport and the burst; a cold LOAD of the written SAV flies on record for
  record as the live World does. Loss controls: the continuation one tick late
  differs, a SAV with a changed or dropped shot differs.
- `TestAShotRecordHasMadeTheDriverCallsOfItsCreationTick`,
  `TestARestoredWindUpCountsTheNativeSwingClockAndBuildsItsShotOnTheSameTick`
  (fixture; the latter fails at restored swing 4, native 3 without the seed
  correction) and the updated swing-count assertions of the unit-shot tests.
- `pkg/sim/fireballaim_test.go`: the countdown for a one-cell actor
  mid-stride (1 against the cell centre's 2), for a restored actor's fine
  position, and on the weapon-rider route; each beside controls with a
  two-cell actor and a cast at a cell, which keep the cell centre.

The EN and RU native Fire_Ball burst witness is unchanged and passes. No RU
save holds a projectile.

## Open debt

- `DIV-1876`: the offset is Medium at one distance per class; a save taken on
  the creation tick of a ShootDelay 0 shot would hold a record this build
  does not yet have.
- `DIV-1877`: one to two ticks between the chain and the observed burst are
  unexplained. Falsifiable questions: what the transport's countdown field holds
  at ticks 3098 to 3101 of the mission-150 catapult battle; how many ticks the
  client dispatcher's dequeue takes between the flush and the record build; what
  the effect list holds at the fire tick; what point the multi-cell and
  null-target branches use.
- `DIV-1879`: the Ballista's weapon spell and the client's handling of the
  damage message while the burst is pending.
- No corpus save holds a Fire_Ball target between cell centres, so the
  fine-position countdown has only fixture and native-stride witnesses.
