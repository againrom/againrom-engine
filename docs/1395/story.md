# One kind of World from mission start

## Intent

An engine-made first-game World holds every carrier a World loaded from an
original SAV holds, from the moment it is constructed, so one rule runs after a
fresh start and after a LOAD. Base: public main `db226edb` (pin k235).

## Scope of this landing

Owner ruling: story1395 is split by carrier. This landing is the Group carrier:
the registry at mission start, the retired native arms, the dialect accessor,
the ported tails and the identity rule for SAVE and LOAD below. The session
clock and the effect driver it builds on are on the same branch. The Mover
carrier (motion), the cell planes and the projectile record producers follow as
separate stories.

## Authority

Owner decision: one kind of World from mission start; a state loaded from an
original SAV and a state the engine created are one kind of state
(`AGENTS.md`, Boundaries; `pipeline/SAV-COMPLETION.md`). The carrier list is
the architecture audit `pipeline/reviews/arch-audit-db226edb.md`, section 1.
Each carrier cites the claims of its importer and the claims behind the rule
that now runs.

`World.SecondGame()` is the one dialect accessor. A second-game World keeps its
present native paths until second-game evidence describes its carriers.

## As built

### Session clock

Constructor: `constructSessionClock` (`pkg/sim/sessionclock.go`). It runs in
`newWorld`, `NewControlledScriptWorld`, `UnmarshalBinary` and
`RestoreCurrentClock`. A first-game World holds SubTick 0 and FullTick 0 at
construction, the values the original's fresh-map start stores (`SESS-084`,
`SESS-085`, High/Medium). A World without the clock (an older byte form, an
older engine SAV whose policy marked the clock unknown) takes
`DerivedFullTick(SubTick)` = (SubTick + 1) >> 4 (`SAV-1210`, High; `SESS-086`).
`DIV-2892` is closed on these claims. The original also runs the first sub-tick
inside the mission start (`SESS-086`); the engine runs it at the first world
tick (`DIV-2893`). `RestoreCurrentClock` no longer drops the
clock of a first-game World. Byte-form wrappers decode their base through
`unmarshalForm`, which decodes exactly what was written, so each wrapper's
canonical re-encoding still compares the form it read; `UnmarshalBinary`
constructs the clock after the whole form is decoded.

Branches removed: every `hasSessionClock` test in `step.go`, `script.go` and
`training.go` is now the dialect test; the light-clock restore at mission entry
runs on every first-game LOAD.

Behaviour that moved for an engine-made first-game World (the session cadence
original-SAV Worlds already ran):

| Rule | Before | Now | Authority |
|---|---|---|---|
| Sub-tick advance | after the tick body | before commands; a command reads the incremented sub-tick, so action deadlines and the structure recharge filter read one higher | `SESS-TICK-006` |
| Script pass | phase 6 after commands and actor dispatch | phase 6 entry arm, before the command drain; a command on the pass sub-tick is first seen by the next pass | `SESS-TICK-006` |
| Script cast | resolved at the head of the next tick | resolved by the world body of the pass sub-tick | `SESS-TICK-006`, `TRIG-CASTACTOR-044` (the temporary caster joins the actor list) |
| Time register 93 | advanced only with a compiled program | advanced on every phase-6 pass | `SAV-650` |
| Outcome report | entry sub-tick 15 | entry sub-tick 14, then FullTick advances | `SESS-TICK-006` |
| Regeneration | end of tick, phase 12 | phase-12 entry arm; health on every fourth full tick | `SESS-TICK-006`, `HERO-REGEN-021` |
| Ladder walk below zero | every 32 sub-ticks for every body | live list: every fourth full tick for any actor below zero (a body set below zero without a death becomes a fallen body); dead list: every second full tick; teardown takes one rung on an even full tick | `HERO-DECAY-069`, `HERO-DEATH-026` |
| Removal of a finished body | the tick it passes the last rung | the dead-list pass of the phase-12 sub-tick | `SESS-TICK-006`, `HERO-DEATH-026` |

The window driver's restored wind-up count (`seedRestoredRuns`) no longer
subtracts one: that offset matched the native tick's deadline convention, which
no first-game World has now.

### Effect driver

Constructor: `constructEffectDriver` (`pkg/sim/carriers.go`), called by
`constructCarriers` with the clock in `newWorld`, `NewControlledScriptWorld` and
`UnmarshalBinary`. A first-game World holds an empty world-effect driver table
from construction, the table `ImportOriginalWorldEffectDrivers` arms; an import
with no row, the retirement of the last carried effect
(`clearRetiredWorldEffectCarriers`) and an older engine SAV whose policy marked
the table absent leave it empty, never absent. An empty table writes no byte
form (its presence carries no state), so no byte form or digest moved.

Branches removed: the lazy creation at the first projectile (`insertProjectile`)
is now second-game only; the driver nil tests inside spell-graph code
(`RestoreCurrentAreas`, `RestoreCurrentSpellDeliveries`,
`detachCurrentSpellNodes`, `refreshSavedSpellGraph`, `captureSavedGraphAreas`,
`tickSavedAreaID`, `retireUnreachableSpellNodes`) go; `restoreCurrentPolicy`
no longer drops the table. In `pkg/game` an empty table with no binding
metadata answers as a World without effects (`validateSavedWorldEffectsWorld`,
`projectSavedWorldEffects`).

Behaviour that moved: none. An empty table and an absent one ran the same rules
in every remaining site.

### Group carrier

Constructor: `constructGroups` (`pkg/sim/carriers.go`), called by
`constructCarriers` in `newWorld`, `NewControlledScriptWorld` and
`UnmarshalBinary`. A first-game World holds the Group registry the original's
mission start builds and `ImportSavedGroups` fills:

- one Group per (owner, group) pair, Players in slot order and a Player's
  Groups in placement order; each Group's `+0x45` = 1 and order 0 from the
  Group AI constructor (`AI-449`, `AI-GROUP-009`);
- the controlled slot's Groups take Stand Ground, which stores only `+0x20` = 3
  (`AI-451`, `SAV-1219`); every other Player's Groups take Guard through the
  guard setter (`AI-450`, `SAV-1218`);
- `+0x48` = 1 on a Group the compiled script names by selector or through a
  member unit (`AI-452`; the opcode-1 gate is `DIV-2896`);
- each member's order holds its post cell word, `+0x14` its reach and `+0x71`
  1, in state 0x0b under Guard and 0x0c under Stand Ground (`SAV-1220`; the
  Units-table bytes are `DIV-2895`). An actor of owner 0 (no Player) is placed
  in a Group of owner 0 with no member orders (`DIV-2897`);
- the Player containers come from the current Players (`RestoreCurrentPlayers`
  rebuilds an as-constructed registry);
- every other AI byte keeps the constructor's zero. The controlled Player's
  `+0x38` is 0 (`SAV-1219`), owner 0's `+0x20` and `+0x38` are 0, and the roam
  counter `+0x15` and commanded cell `+0x0a`/`+0x0b` are 0.

A World built from an older engine's state carries the AI bytes that engine
held: the byte form without the session clock (`UnmarshalBinary`) and a SAV
whose policy lacks the Group carrier (`restoreCurrentPolicy`) construct the
registry with `native` set, so a pair with a native Group record takes that
record's order, notice base, roam counter and commanded cell. The flag
persists: the byte form's construction mark is 2 for it, and the SAV supplement
writes `GroupsNative`. A World loaded from a SAV with the Group carrier keeps
the AI bytes the SAV holds.

The byte form omits a registry equal to the constructed one
(`groupsForm`); the decoder constructs it again. The activity gate
(`AI-ACTIVITY-324`) rewrites `+0x45` from party coverage every tick, so a
stepped World's registry usually differs and is written. A load-time edit to an
as-constructed registry rebuilds it: `ImportOriginalDeadActors`
(`reconstructFreshGroups`) and `RestoreCurrentContinuation`
(`syncConstructedMemberArchives`).

Retired native arms on a first-game World: the native actor pass for every
non-patrol actor (`actorPass`), the native withdrawal pass, the native Group
decision (`aiGroups`, `decide`) inside `engagementPassTraced`, the native
command Group (`commandGroup`, `commandPatrol`, `cmdGroupRoam`), the native
script Group count, order and stop arms (`runCheck`, `cmdGroupOrder`,
`stopGroupMembers`, `groupMembers`), the hand-over and Player reference arms
(`handOver`, `playerReferenceKey`) and the native record teardown in
`remove`. The registry dispatch `savedGroupPass` runs instead: order 0 through
`savedActorDispatch`, other orders through `savedDecision`, then the withdrawal
tail for every Group, with the withdrawal observer threaded through
(`savedGroupPassTraced`).

Ported tails: `resumeSavedSwarm` (`savedgroupsai.go`, `DIV-1517`) and
`joinSavedEngagedGroups` (`groupengaged.go`, `DIV-1523`).

Dialect accessor: `groupDialectNative` (`carriers.go`). It reports a World with
no registry, which on every World the engine constructs or decodes is exactly a
second-game World; the native Group arms run only behind it. It replaces the 58
`savedGroups` nil tests in `pkg/sim` production code.

Behaviour that moved:

- Enemy Groups away from the party do not decide until a party unit comes near
  (`AI-ACTIVITY-324`); a Group the script names decides wherever the party is.
- A command moves the actor into a new Group (`AI-449`); a mixed-owner move
  builds one command Group.
- A Group selector the script names but the World does not hold resolves to no
  Group: a count reads 0 and a command does nothing (was a refused tick).
- The registry Group count excludes a downed member (HP 0), as the native count
  did.
- A member order whose destination was never written acquires in place
  (`DIV-2894`).
- An escort range of zero compares the stored zero (`AI-DEFEND-111`,
  `AI-FOLLOW-112`); a large mover's Move order re-issues and reaches its goal.
- The order progress byte (`+0x09`) has a producer: a zero byte with a loaded
  cycle reads 1, so a queued cast or release waits for the strike.

Identity across SAVE and LOAD. An engine World names a Group owner by slot
alone (Class 1, Key 0, Archive 0) and holds zero in the order bytes the writer
fills from current targets. The writer mints a Player key, writes the Player's
object index and projects victim keys (`+0x0c`, `+0x28`) and reach (`+0x14`,
`+0x50`) into `U158`. A fresh actor holds no key in the World either; the
current supplement restores that absence, and the Group carrier follows the
same rule:

- `KeylessOwner` (`currentGroupContinuation`) names the minted key of an owner
  reference the World held by slot alone; `RestoreGroupContinuations` zeroes Key
  and Archive while the file still names that key.
- `Projected` (`ActionOrderContinuation`) holds each run of `U158[0:144]` the
  writer projected over the World's order, with the World's bytes; LOAD restores
  a run while the file still holds the projected bytes
  (`finalizeCurrentOrderProjection`, `restoreProjected`).
- `GroupsConstructed` records that the registry is the constructed one, as
  play changed it, and not an imported one; LOAD calls
  `AdoptConstructedGroups`, so the restored registry carries the same mark.
  The byte form carries the mark too: a constructed registry written because
  the current state no longer builds it (a stepped Group, a changed reach)
  ends its payload with one mark byte, and a payload without it decodes as an
  imported registry. Without the mark a decoded copy wrote the registry where
  the live World omitted it once the state built it again, and the two hashed
  apart (`TestAConstructedRegistryKeepsItsMarkThroughTheByteForm`).

The importer reads every file byte; an edited file keeps what it names. The
seat's premise (the World holds the minted identity before SAVE) was not
built: the actors' precedent holds no key in the World for a fresh actor, so
the Group carrier records absence the same way. `constructedGroups` walks
Players in current Player order when the current Players cover every owner.

LOAD of an original SAV drops the patrol ring of a body's registry order in a
living-only state (`applyOriginalGroups`, `DIV-1433`), as the entity drops it.

### Writer carrier record

The writer leaves the Groups bit (bit 4, `0x010`) of `WriterCarriers` clear
(`docs/SAV-SUPPLEMENT.md`). It writes every Group field and order from the
registry, but not the identity part of the carrier from state: the World
holds no Player key for a Group owner named by slot and no source archive
binding for a fresh actor. The writer mints the owner key and the members'
object bindings at SAVE, and LOAD restores their absence (`KeylessOwner`
above). The Groups instrument judges a file by that identity. Probe: the
writer set the bit for a first-game World, and fresh RU saves of missions 10,
20 and 80 ran through `TestMilestone2Groups` as their own population. All
three failed, 480 differences each: 79 Group owner references (`G44`)
differ, 199 members have no live entity bound by archive, and 36 live orders
have no bound actor. Setting the bit needs a World that holds the minted
Player keys and actor bindings; that is open debt.

### Basis scalars an action writer replaces

A known basis scalar (`NativeActorBasis.Scalars`) travels in the file word;
the supplement keeps only its availability. The action writers then write
`U58`, `U6C`, `U60` and `U61` from the actor's current attack, cast or dwell,
so LOAD read the action's word back as the basis, and the World hash moved.
`currentNativeBasisWire.Scalars` now names each such slot with the file word
and the World's value; LOAD keeps the World's value while the file still
holds that word, and an edited word wins
(`TestNativeScalarReplacedByActionWriterSurvivesColdLOAD`, with loss
control). Old supplements carry no rows and read as before.

### Stone-held step after LOAD

A unit Stone holds mid-step stays in its crossing (`DIV-073`). Its drawn step
came from the view's memory of the pre-step cell, which a first-game SAV does
not carry, so after cold LOAD it drew `(0,0)` for the whole curse while the
live view drew the crossing. `entityDraws` now takes a Stone-held unit's step
from the World's stride, the same in both
(`TestStoneDrawStepComesFromTheWorldStride`).

### Release-gate fixes

The landing gate ran the whole release set and failed 18 tests; each failed
already at `8c6a7603`. Code fixes:

- The Move arm re-issues the move only for a Group holding a rate byte
  (`savedGroupMove`, `AI-MOVE-023`: `R0154` re-issues while `grpAI+0x44` is
  nonzero). It re-issued the walk every decision, which ended a pick-up and a
  fight. The Swarm 2 fallback keeps its re-issue.
- A far-search substitute moves the registry order's destination with it
  (`settleSavedDestination`), so the dispatch does not re-issue the cell the
  substitute took over from.
- The script Move setter writes every on-map member's destination, a
  Stone-held member included (`savedOnMap`); the statue Lord walks once he thaws.
- A constructed order carries the class spell slot window (`AI-341`,
  `SAV-1220`), as LOAD fills it.
- A script placement writes the known native position block bytes
  (`placeAt`), as the SAV writer does.
- `NewControlledScriptWorld` rebuilds only a registry that is still as
  constructed; a registry play changed is kept.
- A Group of a registry built from an older engine's SAV (no Group carrier)
  builds a member's missing order at its first decision, as the stance setter
  does (`constructMissingOrders`). Such a file holds no member order, so every
  Guard Group refused its dispatch; LOAD keeps the file's state words.
- A group sub-command outside opcode 6's ten cases no longer blocks its
  trigger; it returns without acting (`TRIG-GROUP-005`).

## Proof

### Session clock

- `TestAFirstGameWorldIsConstructedWithTheSessionClock`,
  `TestAWorldWithoutTheClockTakesTheDerivedOne` (`pkg/sim`); both fail to build
  before the change.
- `go test ./pkg/sim ./pkg/game ./cmd/missionrun` pass.

Changed tests, by class. Repin (cause: the session clock is constructed with
the World): `TestMarshalledBytesArePinned`, `TestHashIsPinned`,
`TestTheByteFormAndItsDigestAreTheOnesAlreadyPinned`,
`TestARoutedWorldMarshalsToItsPinnedBytes`,
`TestThePinnedRoutedDigestIsFNV1aOfThePinnedBytes`,
`TestMidStrikePendingOrderMalformedBytesAndHistoricalDefaults`,
`TestPendingAttackBytesHashAndOldBytes` (pin bytes carry clock presence 1; the
form-91 digests `pinDigestV91`, `rtfDigestV91` repinned; the frozen form 77..93
digests now read `pinBytesHistorical`/`rtfBytesHistorical`, unchanged),
`TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth` and
`TestWorldMarshalBinaryIntoCanonicalBytes` (fixture tick within uint32),
`TestTheAlwaysScoringPopulationMatchesTheGoldenDigest` (300-step golden),
`TestWorldExportedMethodSetIsPinned`, `TestReadersDoNotMutateTheWorld`
(`SecondGame`). Behaviour (claim cited in the test): the decay ladder tests in
`decay_test.go`, `corpse_target_hotfix_test.go`, `currentdeadstate_test.go`,
`currentdecayhealth_test.go`, `zeroblow_test.go`, `engage_test.go`,
`spellpointfix1001_test.go`, `regen_test.go`, `originalregen_test.go`,
`scriptgroup_test.go`; the removal fixtures in `currentterminal_test.go`,
`entityidentity1177_test.go`, `nativebasisremoval_test.go`,
`nativescalars_test.go`, `originaldead_test.go`, `currentcontinuation_test.go`
and `pkg/game/nativeterminalsubject_controls_test.go`; the command-before-pass
fixtures in `playerretreat_test.go`, `reacquire_test.go`,
`retreat_refused_test.go`, `savedtactical1115_test.go`, `withdraw_test.go`;
the pass-alone fixtures in `scriptcast_test.go`, `instant30_hotfix_test.go`;
deadlines in `activity1146_test.go`, `turnstate_test.go`,
`structureuse1150_test.go`; `cmd/missionrun/trace_test.go` (teardown rung).
Second-game or old-form only: `currentdeliveries_test.go`,
`currentvalues_test.go`, `currentcontinuation_test.go`,
`secondgamescript_test.go`, `rom2activity_test.go`, `scriptclock_hotfix_test.go`,
`nativeclass_test.go`, `rom2statebinary_test.go`, `training_test.go`.

### Effect driver

- `TestAFirstGameWorldIsConstructedWithTheEffectDriverTable` (`pkg/sim`); it
  fails before the change.
- Changed tests, all of one class (the table is held, so "retired" reads an
  empty table, not an absent one): `currentdeliveries_test.go`,
  `currentpolicy_test.go`, `savedcontinuation_cells_test.go`,
  `savedcontinuation_test.go`, `savedspellgraph_test.go` (`pkg/sim`);
  `worldeffects1162_test.go`, `area1164_release_test.go`,
  `worldeffectscells1162_release_test.go`,
  `originalworldeffects1162_release_test.go` (`pkg/game`).
- Release tests `^TestRelease.*(WorldEffect|Area|Projectile|Shot|Bolt|Burst|Cast)`
  RU and EN pass after the corrections.
- `TestReleaseIdleRegenerationAfterManualCast` (behaviour, session clock
  carrier): the Teleport deadline is 13, not 12, because the command reads the
  advanced sub-tick (`SESS-TICK-006`).

### Group carrier

- `TestAnUnnamedGroupDecidesOnlyWithAPartyUnitNear` (`groupactivity_test.go`):
  the activity gate itself.
- Test classes, with files in `PROGRESS.md` of the lane's review directory:
  behaviour (dormancy fixtures name their Groups with `+0x48`, `AI-452`;
  command, count and hand-over tests read the registry); native arm (tests of
  the native arm's own structure run it with no registry, `nativeGroupArm`);
  repin with cause (`commanded_test.go` five legacy digests: the native group
  section holds two records, not three; `nostate_test.go`,
  `relaxation_test.go`, `script_test.go`: the registry the activity gate
  rewrote is written; `regen_test.go`: the reference takes the gate's write;
  `binary_test.go`: the command's order record outlives arrival;
  `rom2statebinary_test.go`, `savedgroupplayers_test.go`: second-game and
  first-game registry presence).

### Mission-start Group AI bytes

- `TestReleaseMissionStartGroupAIIsTheConstructors` (`pkg/game`, gated release
  population), EN and RU: missions 10, 20, 41 and 80 opened fresh; every
  registry Group holds `+0x45` = 1, Stand Ground `+0x20` = 3 and `+0x38` = 0 on
  the controlled Player, Guard `+0x20` = 1 and `+0x38` = `+0x2d` on every other
  Player, nothing on owner 0, and zero in every byte no constructor or setter
  writes; each SAV `G3C` equals the live bytes. Loss control: the same SAV is
  LOADed as an older engine's (no Group carrier in the policy, no
  `GroupsConstructed`); the registry is built from the native Group records,
  the controlled Player's `+0x38` takes their notice base and the check fails. With the copy restored
  for a fresh World the witness fails on the controlled Group, RU `+0x38` = 8,
  8, 10, 8 on missions 10, 20, 41, 80. `TestAFreshRegistryHoldsOnlyTheConstructorsAIBytes`
  (`pkg/sim`) holds the same bytes per owner on a fixture, and an older byte
  form, without the session clock, builds its registry from the native records
  and keeps that mark through the byte form.
- The correction changes no trace. `missionrun -census -trace` for 320 ticks on
  missions 10, 20, 41 and 80, EN and RU, gives output byte-equal to the reviewed
  candidate's. The bytes it changes are not read at mission start by the
  registry decision: `+0x38` is read only under Guard, where on these four missions the guard
  setter's value equals the native notice base the copy wrote (the witness
  checks `+0x38` = `+0x2d`); owner 0 is not
  dispatched; `+0x15` and `+0x0a`/`+0x0b` were 0. Mission 80's owner-2 units
  115 to 119 still move differently from main: that difference comes from the
  registry Group decision that replaced the native one, which this correction
  does not change. Their original trajectories are Unknown.

### Group carrier through SAVE and LOAD

- `TestReleaseGroupCarrierSurvivesSAVEAndColdLOAD` (`pkg/game`, gated release
  population), EN and RU: mission 41, the hero attacks the gate warrior until
  two members of its Group have moved (tick 951), SAV SAVE, cold LOAD into a
  fresh front end. 13 Groups, 43 orders; the restored registry and the World
  bytes equal the live ones (RU hash `727bb9226768a243`, EN
  `a2909bcc6aec978f`); the next 60 ticks agree; 30 projected order runs are
  restored; the supplement marks the registry constructed. Loss control: the
  `KeylessOwner` rows are removed from the file; the LOAD then restores 13
  owner keys and the hash differs (RU `5ffb07b9b228a090`, EN
  `3aa9b4bb11d88004`).
- 75 AI and Group release tests, RU and EN: 72 pass, 3 skip (second-game
  roots), among them `TestReleaseAnAttackedGroupAtKadaganPostFightsAsAGroup`.
- Changed tests. Repin: `partialCurrentGraph` (the Group binding's container
  comes from the Player binding), `currentplayerroots_test.go` (the World
  without a Group carrier is a second-game one), `gameoptions_test.go` (fixture
  Self Player), `readout_test.go` (the rate is read from the registry `+0x44`),
  `midstrikeorders_test.go` (the requested-loss control also drops the spell-key
  anchor), `pkg/sim/world_test.go` (method set),
  `pkg/mapload/currentplayers_test.go` (a first-game World holds the
  constructed registry, `AI-449`); the release instruments
  `holdposition_release_test.go`, `holdretreat_release_test.go`,
  `scriptstandground_release_test.go` read the member's registry Group;
  `footprint_release_test.go` places a Self party unit near the retreating
  Group (`AI-ACTIVITY-324`); `attackmove_release_test.go` waits for the last
  step's transit before it calls the hero arrived;
  `terminalfine_hotfix_release_test.go` stops the `game0036.sav` route two ticks
  sooner, because the engaged-Group tail (`DIV-1523`) sets that Group on the
  party two ticks sooner. Behaviour: `deadpatrolsave_hotfix_release_test.go`
  passes by the `DIV-1433` reader rule above; the round-trip gate baseline
  records `game0008-original-m70-dying.sav` resave as accepted (its
  `savedGroups.Orders.Raw` mismatch no longer occurs).
- M2 (`check-milestone2-acceptance.sh`, `AGAINROM_M2_JOBS=2`), EN and RU each,
  on main `ea142c0d` merged: no failing name. `saveorcsdontgo.sav` is judged
  by the writer rule (`WriterCarriers` absent); its round-trip gate dying case
  is accepted. That case fought on this branch where main's did not (the
  activity gate and the engaged-Group tail), which exposed the two defects
  above: three actors' replaced basis scalars and a Stone-held unit's step.

### Release-gate repins and instruments

Repins, each to behaviour the story moved toward the original:

- `TestGeneratedMissionSAVFullTickFollowsSubTick`: a generated mission holds
  the session clock, 113/7 after the run (`SESS-084`, `SESS-085`, `SAV-1210`).
- `TestReleaseGeneratedMissionSAVOriginalConstraints`: the hero's `U158` post
  is its own cell and `U50` is 12 (`AI-351`, `SAV-1220`); seven `game9999.sav`
  files and `game0006`, `game0011`, `game0012` hold the same for every hero.
- `TestReleaseCheatSummonUsesAppAndOrdinarySAV`,
  `TestReleaseMission100ServantLeavesAtTheWin`,
  `TestRoodThreeOwnerSaveRescueContinuation`: the order is read from the
  registry Group (`holdGroupOrder`); `FrozenGroupAI` reads the native record,
  which a first-game World no longer updates.
- `TestSavedGroups1113BothDoorsOrdinaryMenuFreshLoadAndNextAction`: the
  synthetic Move Group holds a rate byte, so its saved Move still executes
  after LOAD (`AI-MOVE-023`, `MOVE-GATE-035`).
- `TestReleaseProfileWitnessIsUnchanged`: `rom1-en.txt` and `rom1-ru.txt`
  rewritten; only the `world=` and `save=` hashes moved, every frame is equal.

Instruments:

- `cmd/missionrun` withdrawal witness runs its cloned World in safe mode: a
  Group the party's coverage does not reach skips its pass, the withdrawal
  tail included (`AI-ACTIVITY-324`). The witness is the one main prints
  (mission 100, Bat 107).
- `cmd/scriptcoverage` reads a hand-over through the registry (a fresh
  one-member Group, `AI-449`) and reads each probe's postcondition right after
  the instant (`sim.StepTracedAt`): a session tick runs the script pass before
  its body resolves casts and ages effects (`SESS-TICK-006`).

The 16 `pkg/game` names and the two instruments pass on EN and RU.

## Open debt

- Second-game Worlds keep the native uint64 tick and the native cadence; the
  dialect sites are the `SecondGame()` tests in `pkg/sim/step.go`, `script.go`,
  `training.go`, `currentworldpolicy.go` (`RestoreCurrentClock`) and the
  fallback in `pkg/game/savdocument.go` (`snapshotSavedDocument`,
  `validateSavedDocumentWorld`).
- `DIV-2893`: the mission start runs no session step, so an engine World is
  one sub-tick behind the original for the whole mission.
- An older byte form whose uint64 tick exceeds 32 bits keeps only the low word.
- Second-game Worlds keep the lazily created driver table: `insertProjectile`
  creates it, and nil-safe reads remain in `currentEffectPopulation`,
  `CellEffects` and `setCellEffectTime` (`celleffect.go`), `clearSavedAreaLayer`
  (`arealayers.go`), `currentidentity.go`, `restoreCurrentProjectileTargets`,
  `detachSavedProjectileTargets`, `dropSavedProjectile` and the window driver
  (`armedRecords`, `noteShotPreMoves`).
- Group carrier: `DIV-2894`..`DIV-2897`; the owner-0 actors of `DIV-2897`
  stay as built, UNKNOWN, until research answers the original's rule. The native Group records
  (`w.groups`) are still built and saved beside the registry on a first-game
  World; the native arms that read them run only in a second-game World.
- `WriterCarriers` bit 4 (Groups) stays clear until the World holds the Player
  keys and actor bindings the writer now mints at SAVE.
