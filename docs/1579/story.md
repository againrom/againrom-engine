# Every Mover byte the engine writes comes from one producer

## Intent

The Mover record (U154) is the 180-byte actor block the original keeps per
actor and the SAV carries. Before this story eleven production files wrote its
bytes, each with its own offsets, and the town carrier recognised the town
writer's constructor Movers by their byte values. Now one file,
`pkg/sim/moverbytes.go`, writes every Mover byte; every other writer calls
it with typed values, except the two whole-record copies of the SAV format
(`currentActorGraph`, `SetActorMoverRoute`), which compute no byte. No player change: every SAV keeps its bytes, every LOAD
its result and every hashed World state its value, on both games.

## Authority

- Owner: one builder per kind; a kind with a second path in any shape counts
  as multi-way. Architecture work counts 0 toward the release.
- Audit-20 plan, R35 (Mover byte construction).
- The claims each writer's comment names: MOVE-119 to MOVE-122, MOVE-128,
  MOVE-CLAIM-007, MOVE-STEP-010, MOVE-TURN-044, AI-658, AI-682, AI-690 to
  AI-693, TERR-PASS-051, SAV-1092, SAV-1096, SAV-1119, SAV-CELLFAIL-583,
  SAV-CELLLEAVE-584, SAV-CROSSNEXT-585. The story changes no claimed value.

## As built

The producer, `pkg/sim/moverbytes.go`:

- `MoverHead {Facing, Desired, Mask, Radius, SiegeRoute, Rate}` is the head
  (+0x00, +0x01, +0x05, +0x08, +0x09, +0x0a); `MoverHeadOf` reads it, its
  encoder writes it, `Record` gives a Mover holding it, `EditMoverHead` edits
  it in place, `CopyMover` writes a whole Mover into record bytes keeping the
  facing or the rate (`KeepFacing`, `KeepRate`), `MoverFrom` reads a record's
  bytes into a Mover.
- The constructor's default: `constructorHead`, `placedMover` and
  `constructedMover`; `SiegeHireMover`; `TownConstructorMovers` (the town
  writer's Human and siege hire Movers for a mask and rate); `MoverRate`.
- The later state: `putStepCost` (+0x72), `putEntryCache` and
  `defaultEntryCache` (+0x82), `putLeaveCache` (+0x86), `putWalkClaim` and
  `readWalkClaim` (+0x74, +0x76, +0x90), `putMoverTarget` (+0x70),
  `nameClaimedRouteCell` (+0x06), `moverStride` with `putMoverStride`,
  `putStrideOrigin`, `putStrideElapsed` and `endStride` (+0x80, +0xa6 to
  +0xb1), `moverTurn` with `moverTurnOf` and `putMoverTurn` (+0xa0, +0x9d,
  +0xa4), `clearMoverMotion`, `clearRetreatFailure` (+0x98),
  `clearPursuitVictim` (+0x7c..+0x7f; the group roam reset and the
  stand-ground mirror call it, where each ran `clear(m.Mover[0x7c:0x80])`).

The writers that call it: `ProjectActorMotion`, `ApplyMover`, the imported
crossing and its arrival, the saved turn leaf, `publishSource`, the retreat
continuation, the World form decode; in `pkg/game` the actor value pass
(`savedActorValueRecord`), the dead construct's radius, the siege hire, the
city Human's movement mask and rate, the saved-motion and World-motion SAVE,
and the town carrier (`keptFromMission`, `keepMover`, `writeTownActor`).
`keepMover` compares the carried Mover with `TownConstructorMovers` for the
record's own mask and a zero rate; `constructorMover` is gone. The city
movement's own mask table gives way to `MoverPassabilityMask`; its domain is
checked to 1..3 first, so the byte is the same.

The guard, `internal/archtest/moverbytes.go`, refuses in `pkg` and `cmd`
production code outside the producer an index or slice write, a
`binary.*.Put*`, a `copy`, a `clear` and a byte-holding composite literal aimed at a
Mover: a field named `Mover` or `Raw154`, a `[180]byte` or `[MoverLen]byte`
value or pointer, a variable named `mover`, a variable taken from an
expression naming `"U154"`, or a `Raw["U154"]` index. Its named entries are
the SAV format's whole-record reader (`currentActorGraph`) and setter
(`SetActorMoverRoute`).

## Proof

- `TestMoverBytesRecordSim` and `TestMoverBytesRecordGame` were committed at
  the base before the change and pass unchanged after it. They record the
  180 bytes of each writer for each actor kind and the hash of small Worlds
  after a few hundred steps, from synthetic Worlds and documents only.
- A planted change of the producer's `constructorHead.SiegeRoute` from 0xff to
  0xfe fails 38 records across both tests.
- `TestCheckMoverBytesNamesEachShape` plants each refused shape;
  `TestMoverBytesHaveOneProducer` scans the live tree.
- Census of writes outside the producer, by shape, base to now: the `clear`
  builtin on a Mover, 2 to 0 (`grouproam.go`, `standground.go`); the guard
  plants `clear(m.Mover[0x7c:0x80])` and names it.
- Focused tests: `./pkg/sim -run 'Mover|Motion|Turn|WalkClaim|Stride|Route|Retreat|Source'`,
  `./pkg/game -run 'Mover|Motion|TownKept|Siege|DeadConstruct|ActorProject|SavMotion|WorldSave|Carrier' -skip '^TestRelease'`,
  `./pkg/formats/sav -run 'Actor|Mover'`, `./internal/archtest -run 'MoverBytes|Dag|TestWorldCarrier'`,
  `./internal/storyguard -run '^TestLiveTreeClean$'`.
- Release tests for the next release, both games: every release test that
  writes or loads a SAV with actors, keeps actors in a town, hires a siege
  unit or compares Mover bytes (`movercarriersave`, `moverstart`, `townkept`,
  `siegehire`, `siegeunit`, `siegeweaponspell`, `idleaimover`,
  `worldsavemotion1170`, `originalmoverroute1134`, the milestone-2 mover
  instruments).

## Open debt

- `TestTheCarrierMatrixKeepsItsRecordedValues` fails at the base `64946cc66`
  with the same output (an Order word U5C); this story does not touch it.
- `retainDeadValues` copies a loaded dead record's raw blocks, the Mover
  among them, by a name list; it carries loaded bytes and computes none, and
  the guard does not see it.
- `pkg/game/nativecity.go:250` builds the city Human's record with
  `Raw154: make([]byte, 180)`: 180 zero bytes, no byte written there; the
  producer gives its mask and rate afterwards. The guard does not see it.
- The guard names a Mover by field, type, variable name and the `"U154"`
  name; a Mover reached through another alias is outside a source check
  without types.
