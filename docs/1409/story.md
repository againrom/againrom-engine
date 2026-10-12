# The mission-start Mover for every first-game World

## Intent

Every first-game World holds a Mover record for each actor from the moment the
actor is built, as the original's constructor, spawner, occupy and join walk
leave it, and a SAVE writes every Mover byte from that record and the actor's
current motion. At mission start every placed actor faces the way the
original's constructor turns it. Base: public main `e56dff9e` (pin k258); the
branch moves the pin to k262 and merges main up to `b31f3518`.

## Authority

Owner: story1395 lands by carrier; this story is the Mover carrier and keeps
native stepping for every move begun in the World (world-paths audit,
`pipeline/reviews/world-paths-audit-6fbd77ce.md`, "Movement"). DIV-802 and
DIV-951 stay in force. Randomness ruling: the start facing is one consumer of
the random service, in the original's construction order.

Claims: `MOVE-119` (constructor: facing 0x40 plus one draw over 0..0x80 at
`+0x00`, copied to `+0x01`; `+0x05` 0x41, `+0x08` 5, `+0x09` 0xff, `+0x0a`
0x10; stat default `+0x0a` 8), `MOVE-120` (type-6 spawner: mask by domain,
Human derive at spawn, radius 0 draws nothing), `MOVE-121` (occupy: `+0x72`,
`+0x82..+0x85`), `MOVE-122` (join walk replaces a carried member's Mover with
the constructor's), `MOVE-128` (a crossing restores `+0x72`), `MOVE-142` (the
start's `0x46` subtype-3 command runs the Human derive that writes a carried
member's `+0x0a`), `SESS-090` (the first draws of a fresh-map start are one
facing per type-6 actor in record order, then the join walk's), `SAV-1213` and
`SAV-1225` (the restart-slot census and its missions).

## As built

### Facing

The one facing is `Entity.Facing`/`DesiredFacing`; the Mover record holds no
facing and SAVE writes `+0x00`/`+0x01` from the actor.

| Path | Draw |
|---|---|
| map actors at mission start (`mapload.startMission` before the drop pick, `World.ConstructPlacementFacings`) | one per actor in record order |
| Control Spirit ghost (`raisedGhost`), cheat summon (`CheatSummon`) | one at construction, World stream |
| carried members placed by the join walk, story1408's guest members | none (`MOVE-122`) |
| revive | no in-World path: `R0825` runs between missions |
| second-game Worlds | none; the placement facing stays |

In original-stream mode the draw is `R0861(0x80)` on the shared stream. In
seeded mode it is `random.Keyed(seed, start-facing, actor id)`, which moves no
stream, so the seeded drop pick and every later World draw are unchanged
(`docs/1380/story.md`, "Streams", row start-facing).

### Mover builder and record

`World.constructedMover` (`pkg/sim/mover.go`) is the one builder: the
constructor's bytes, then for a placed actor the facing, the rate, the mask of
its domain and the occupy's stores; for a carried member the constructor's
bytes alone. The World keeps per actor (`World.movers`, `moverRecord`) what
SAVE cannot take from the actor: the mask `+0x05` and whether `+0x72` is still
the join walk's 0. `+0x08`/`+0x09` are the constructor's 5 and 0xff for every
actor; SAVE writes 0 there for an actor the controlled player owns at the save
(DIV-1388), so an actor that changes owner writes its new owner's bytes.

Paths that build a record: `constructCarriers` (`newWorld`, the byte decoder,
the controlled script copy) for every actor without one;
`ConstructMissionMovers` at the end of `startMission` and
`startMissionScripted`, with the party as joined; the Control Spirit ghost, the
cheat summon and a new actor of an original-SAV living import
(`constructMover`). `remove` and `ImportOriginalActorMotions` drop records
(`dropMovers`): an actor whose Mover an original-SAV import holds has none. The
identity remap of a LOAD carries the records. The first rated crossing ends the
join walk's 0 (`moverCrossed`, after `recomputeCrossing` in `step.go`).

No record is active: native stepping runs every move, and the existing carrier
path for a loaded crossing or pending turn is unchanged.

### SAVE and LOAD

| Mover field | Written from |
|---|---|
| `+0x00`, `+0x01` facing, wanted facing | the actor |
| `+0x05` mask | the record (domain mask; 0x41 for a carried member) |
| `+0x08`, `+0x09` | the constructor's 5, 0xff; 0 for an actor the controlled player owns (DIV-1388) |
| `+0x0a` rate | the actor's rate now (unchanged) |
| `+0x72` step cost | the speed word and cell cost byte now (`ApplyMover`, DIV-3034); 0 while the join walk's stands; 0 on a cost byte of 0 (DIV-3033) |
| `+0x82..+0x85` cell cache | another player's actor: its cell when idle else its post, 0x80/0x80 (unchanged rule); the controlled player's: 0 (DIV-3035) |
| motion bytes (`+0x70`, `+0x80`, `+0x9d`, `+0xa0..+0xb1`), routes | the actor's native motion (unchanged projection) |
| every other byte | 0, the constructor's value |

`projectMotion` (`pkg/game/worldsavemotion.go`) applies the record for every
actor whose Mover no import holds, also inside a World with an imported
carrier, where a later summon was written nowhere before. LOAD of an engine SAV
takes the record back from the loaded Mover: when the policy drops the imported
carrier (`adoptMovers`) and when an actor's action restore drops its imported
motion (`adoptMover`). Both run after the actors' owners, speed words and
motion are restored. A step cost of 0 where the cell gives another marks the
join walk's 0; a Mover with mask 0 holds no record.

`WriterCarriers` keeps the Mover bit (`currentCarrierMoverRoutes`) clear: the
bit promises the cell slots and blocks, which the cell planes story builds.
`TestMilestone2MoverRoutes` therefore still compares Againrom SAVs by the
round-trip check and original SAVs field by field, as before.

### Byte form

Wrapper form 122 (`MVR1`, `pkg/sim/moverbinary.go`), outermost around the
automatic-casting form 121, lists only records that
differ from the built record for the actor's current state; a carried member
whose cell gives a step cost of 0 is stated as crossed. A World with no carried
member and no adopted record writes no wrapper, so a map-only World's digest is
unchanged.

### Second game

Second-game Worlds hold no record and draw no facing; the projection writes
their Mover as before. No second-game evidence describes its constructor.

## Proof

- Start state: `TestReleaseEveryMissionStartHoldsItsConstructedMovers` opens
  every mission the install ships (27, 2092 placed actors on EN and on RU),
  SAVEs, and checks each placed actor's facing (0x40..0xC0, `+0x00` =
  `+0x01`), mask, `+0x08`/`+0x09` and step cost. Loss control: a mission with
  two or more placed actors holds two or more facings.
- Restart slots: `TestReleaseMissionStartMoversAgainstTheRestartSlots` starts
  the mission of each `game9999.sav` (7 files, missions 10, 10, 20, 30, 30, 41,
  141) fresh and compares every Mover byte except the facing for every map
  unit both hold. Differences: mission 41, 3 controlled-player map units at
  `+0x08`, `+0x09`, `+0x82..+0x85` (DIV-1388, DIV-3035); mission 141, map unit
  5 (player 3), rate 0x14 against 0x10 and step cost 0x14 against 0x10, one
  speed word that differs before this story (open debt). No other byte differs.
- Draw order: `pkg/mapload/startfacing_test.go` (one original-stream draw per
  placed actor before the drop pick; seeded facings keyed and the drop pick
  unchanged; loss control a skipped draw).
- SAVE, cold LOAD, next ticks: `TestReleaseMoverRecordsSurviveSAVEAndColdLOAD`
  (Kadagan Post, EN and RU) SAVEs with the hero turning and again with it
  between two cells, a placed AI actor standing; the cold LOAD's World bytes
  equal the live ones and 256 successor ticks hash equal. Loss control: the SAV
  with the constructor's 0 step cost for the standing actor restores a
  different World.
- Builder, byte form, crossing, removal: `pkg/sim/mover_test.go`.

### Repinned tests

| Test | Cause |
|---|---|
| `TestRaisedGhostCarriesItsCensusFields`, `TestARaisedGhostInheritsAnInactiveTurnAtItsSourcesFacing` | the ghost's facing is the constructor's draw; `MAGIC-249` copies no facing |
| weapon swing sound witness (`weaponswingsound_release_test.go`), window 12 to 32 ticks | the victim's start facing lets it strike first, so the hero's swing begins later |
| Stand Ground script witness (`scriptstandground_release_test.go`), mission 40 | with the start facings, a member strikes an adjacent actor of another player; the witness admits that victim |
| profile witnesses `testdata/profilewitness/rom1-en.txt`, `rom1-ru.txt` | World and SAV digests: the start facings, the record form and the step cost in the SAV; frames unchanged |

The swing and Stand Ground witnesses were checked by switching the start facing
off: each returned to its earlier value. The profile witnesses' frame lines are
unchanged; only digest lines moved.

`TestReleaseMission121BridgeTrollKeepsItsVictimWhenItsRouteIsRefused` placed
its pair and victim in map order through the script placement, whose
neighbour search draws from the World stream. The placement cells are equal
with and without start facings; only the troll's start facing changes, and the
previous test already depended on map order (5 of 20 EN runs failed). The
test now places in a fixed order and repeats a placement until the actor
stands on its cell; it passes on EN and RU with the troll refused at tick 39,
as before this story.

## Withdrawn ring placements

On an original stream the original constructs a placement on the map's border
ring, drawing its facing, before the cell refusal deletes it (`MOVE-120`,
`SESS-090`, `MOVE-141`). `WithdrawBorderPlacements` removes such records before
the World is built, so `ConstructPlacementFacings` draws one facing for each at
its position in record order (`withdrawnBefore`, read from `AuthoredUnits`) and
discards it. Missions 80, 90 and 100 are the shipped maps that hold them.

## Start without a party

A mission start in original mode with no party now switches the World to the
original stream, in both games, as a start with a party does. No production
path starts a mission without a party.

## Open debt

- DIV-3033, DIV-3034, DIV-3035 and DIV-1388 name the remaining Mover
  differences.
- Mission 141, map unit 5: its speed word differs from the restart slot's.
- The cell planes carrier, the cost table into `newWorld` and the projectile
  record producers are the next stories; the Mover bit of `WriterCarriers`
  waits for the cell planes.
