# The carriers story 1557 left become rows and steps of the one carrier set

## Intent

Story 1557 made the World carrier set (`pkg/game/worldcarriers.go`) and left
eight carriers outside it, with their own calls in the SAV producer, the LOAD,
the generated document builder, the mission settle and the town. This story
makes each of them a row of the set or a step of a row and widens the guard,
so a second path of any of them is a finding. No SAV the engine writes, no
LOAD result and no hashed World state changes, on either game.

## Authority

- Owner direction «Доводим до 50»: the next architecture audit counts at most
  50 multi-way kinds. This is audit `arch-audit-5809ddc1` R1, plan B1 (World
  carrier set, the rest). Architecture work counts 0 toward the release count.
- ROM1 and ROM2 behaviour is not changed. No DIV row is added or changed;
  DIV-4401 and DIV-4402 stay unused.

## As built

### New points and steps of a row

`worldCarrier` keeps every step 1557 gave it and takes two more:

- `build`, keyed by `carrierBuildPoint`, runs inside the generated document
  builder (`materializeCurrentWorld`): `seatBuild` once the builder is made,
  `keysBuild` once the World's keys are reserved, `joinBuild` once the
  retained keys are claimed, `deadRecordBuild` on each dead record after its
  dead tuple, `playerBuild` on each Player record before its name is
  written, `effectsBuild` once the projectile leaves are written, and
  `absentDeadBuild` on each dead record whose body the World holds as no
  entity, once that record is built. `buildWorldCarriers(at, c)` runs them
  in row order.
- `town`, keyed by `carrierTownPoint`, moves a family between the town and its
  saved forms: `townCapture` (Town to Snapshot), `townRestore` (Snapshot to
  Town), `townArchive` (an original city file to Town), `townCarry` (an ended
  mission's World to Town), `townWrite` (Snapshot to city document).
  `seatTownCarriers(at, c)` runs them in row order.

`carrierRecord` takes eleven decode points: `fileRecord` (a SAV file's
decoded document), `materialRecord` (a SAVE's Snapshot as the World-built
document replaces its loaded one), `identityRecord` (the actions leaf's
identity remap), `terminalRecord` (inside the remap, once the identities are
restored), `selectionRecord` and `orderRecord` (before the continuation is
restored), `tokenRecord` (once the native actor bases are restored),
`cellRecord` (once the script program is restored), `heldRecord` (a LOAD's
SAV document once the mission holds it), `resumeRecord` (a native resume's
document once the mission holds it) and `entryRecord` (an original LOAD once
the spell book walk resumes). `carrierStage` takes `graftStage`, the mission
document once the World effect order is written. `carrierSeatPoint` takes
`settleSeat`, the mission door's settle once the driver is open.

### Each left carrier, before and after

| carrier | before | after: row, step |
|---|---|---|
| loaded document kept for the mission writers | `s.loadedDocument = s.SavedDocument` in `currentMissionDocument` | row "the loaded document ...": decode `materialRecord` |
| reads of the loaded record in the generated builder | `b.loaded = ...`, `reserveRetainedDeadEffectKeys`, the identity joins in `materializeCurrentWorld`; `retainDeadValues`, `retainDeadBook`, `retainDeadEffects` in `currentDeadRecord` | loaded document row: build `seatBuild` (`seatLoadedDocument`), `joinBuild` (`joinLoadedActors`, into `b.joined`); dead body row: build `keysBuild` (`reserveRetainedDeadEffectKeys`), `deadRecordBuild` (`retainLoadedDead`) |
| identity remap of the actions leaf | `restoreCurrentActorIdentities` called by `restoreOriginalActions` | row "World actor identities ...": decode `identityRecord`, run once |
| terminal actor rows (`ImportCurrentTerminalActors`, held rule) | inside the remap | row "World terminal actors ...": decode `terminalRecord` (`importCurrentTerminalActors`), run by the remap through the set |
| Token Reference misses capture | `decodeSavedDocument` built `ReferenceMisses` itself | misses row: decode `fileRecord` |
| `settleMission` application state | `restoreApplicationState` and the original action view in `settleMission` | row "the application state ...": seat `settleSeat` (`settleLoadedApplication`) |
| LOAD matchers in `restoreOriginalActions` | `matchCurrentBookSelection`, `matchCurrentOrderSpellAbsence`, `restoreCurrentWorldEffectTokens`, plane residue with the cell costs | book selection row: decode `selectionRecord`; order spell absence row: `orderRecord`; World spell-effect Token row: `tokenRecord`; plane residue row: `cellRecord` (`restoreCurrentPlaneResidue`) |
| town Diary knowledge | `projectTownKnowledge` (`currentsave.go`), `savedPlayerDiary` (`originalrestore.go`), `Town.carryKnowledge` (`campaignreturn.go`), the Snapshot copies in `save.go` | Diary row: town steps `townWrite`, `townArchive`, `townCarry`, `townCapture`, `townRestore`; `townknowledge.go` moved into the set file |
| `applyOriginalDead` | a production entry beside the dead body row | a test helper (`originaldeadjoin_test.go`); production keeps the row's hook only |
| world effect keys and records | `reserveWorldEffectKeys` and `currentWorldEffects` called by `materializeCurrentWorld` with `s.SavedDocument` | World spell-effect row: build `keysBuild` (before the dead body row's keys), `effectsBuild` |
| Player name | `currentPlayerName(s.SavedDocument, …)` in the builder's Player loop | loaded document row: build `playerBuild` (`keepLoadedPlayerName`) |
| actor actions and the order words `U40`, `U5C` | `projectSavedActorActions(state, &world, s.loadedDocument)` in `currentWorldDocument` | row "World actor actions …": writer at `graphStage`, first of that stage |
| unknown spans of the loaded objects | `graftUnknownObjects(&doc, state, s.loadedDocument, &world)` in `currentWorldDocument` | loaded document row: writer at `graftStage` |
| dead body art (`Mission.DeadArt`, `T0E`) | `Mission.restoreDeadArt` in `restoreSavedDocumentDecoded`; the `T0E` write in `appendAbsentDeadRecords` | dead body row: decode `heldRecord` (`restoreSavedDeadArt`), build `absentDeadBuild` (`writeDeadArt`); the original LOAD keeps its decode in the row's `archiveDead` hook |
| session head absence | `restoreAbsentSessionHead` in `restoreOriginalActions` and in `resumeWorld` | row "World session head absence": decode `actionsRecord` (before the carrier absence row), `resumeRecord` (`restoreResumedSessionHead`) |
| session entry publication | `publishSessionEntry(ms)` in the original LOAD | row "World session entry …": decode `entryRecord` |

Each step runs at the point and in the order the code it replaces ran. Two
moves shift a line without changing what it reads: the builder's `b.loaded`
is set by `seatBuild` at the same place, and the identity joins run at
`joinBuild` where they ran, with their held keys computed inside the step.
The widened guard found the session head absence as a read of the loaded
record passed outside the set; the session entry publication stood on the
nil allow list without being a validator. Both moved the same way.

### Kinds and builders

| kind | builder |
|---|---|
| SAV World carrier (capture, decode, absence, transform, seat, writer, build, town) | `worldCarrierSet` in `pkg/game/worldcarriers.go`; each carrier a row, each step a field |
| SAV byte codecs | `pkg/formats/sav`, unchanged |
| document capture chain and its LOAD validators | their existing functions, named in the guard's `worldCarrierNilDocumentOwners` |
| town Diary knowledge | the Diary row's town steps |
| identity remap of the actions leaf | the set's identity step (`identityRecord`), run once |

### Guard

`internal/archtest/worldcarriers.go` names every moved hook and World writer
in `worldCarrierOwners`, so a reference from a function the table does not
name is a finding. In production files of `pkg/game` other than
`worldcarriers.go` it refuses:

- a write of a field of `worldCarrierFields` (assignment, index assignment,
  `&`, `++`, composite literal key) by a function its entry does not name:
  `Snapshot.loadedDocument` and `Town.knowledge` by none, `Mission.DeadArt`
  only by the dead body row's original LOAD hook, the identity step and the
  autosave's detached mission copy;
- a read of `Snapshot.SavedDocument` or `Snapshot.loadedDocument`, through
  the field or through a local it was assigned to under any name, passed to
  a function `worldCarrierLoadedCallees` does not name (the capture chain's
  `cloneSavedDocument` and `generatedCurrentMissionDocument`, the LOAD's
  `restoreSavedDocument`, and `applicationActorIDs`, each with its reason);
- a nil test of the loaded record (a name of `worldCarrierDocumentNames`, or
  such a field or local) in a function that is no step and is not named.
  A step is a key of `worldCarrierOwners`. The named functions are the 57 of
  the capture chain and its LOAD validators in
  `worldCarrierNilDocumentOwners` and three readers with a reason in
  `worldCarrierNilDocumentReaders` (`assertHeadlessState`,
  `loadedTavernStockIDs`, `mapWorld.virtualDeadDraws`).

## Proof

- `TestTheCarrierMatrixKeepsItsRecordedValues` (`worldcarriermatrix_test.go`):
  seven fixture rows, both games (first-game fresh mission, original dead
  bodies, terminal actors, bound corpse, town Diary; second-game original
  mission file and town Diary), each with the World hash, terminal, dead,
  Diary and application digests or the town's knowledge and gold, the SAVE
  bytes, the LOAD of that SAVE and the SAVE again. The table was recorded on
  the base before the first change (commit `dcd8d1d63`) and passes unchanged
  after. A planted change of the Diary row's `townArchive` step fails the
  first-game and second-game town rows.
- Six more matrix rows plant a loaded record and record what its LOAD seats
  and what the SAVE after it writes: the first-game Player name, Player
  tail spans (the graft), order word `U40`, world effects and a virtual
  dead body's art, and the second game's LOAD of planted Player names and
  tails. They were recorded at `65c79eb50` before any of these sites moved
  (commit `90aa04c56`) and pass on a `git archive` extract of `ff56581b7`.
  On that second-game fixture the SAVE after the LOAD writes the hero's name
  and no tail. `U5C` is not reached: no fixture actor meets its idle residue
  test. A planted `keepLoadedPlayerName` that writes no name fails six rows,
  the Player name row among them; a planted `writeDeadArt` that writes no
  `T0E` fails the dead body art row.
- One test per moved carrier through the set (`worldcarriermoved_test.go`):
  `TestTheMovedCarriersAreStepsOfTheSet`,
  `TestTheSetKeepsTheLoadedDocumentAndItsMisses`,
  `TestTheSetSeatsTheLoadedRecordInTheBuilder`,
  `TestTheIdentityStepImportsTheTerminalActors`,
  `TestTheDiaryRowCarriesTheTownsKnowledge`,
  `TestTheLoadedRecordSitesAreStepsOfTheirRows`.
- Guard: `TestWorldCarriersRunThroughOneSet`,
  `TestWorldCarrierRuleNamesAPlantedBypass` and
  `TestWorldCarrierRuleNamesAPlantedFieldWrite` (a town Diary write by
  assignment and by literal, a kept loaded document write, a nil test outside
  the allow list, direct calls of a town step, a build step and the identity
  remap, and the set file's own writes and tests admitted; a dead body art
  write by index outside its hooks and inside the original LOAD hook, the
  loaded record passed to a function outside the set directly and through a
  chain of renamed locals, the capture chain's copy admitted, a nil test
  through a renamed local, a step's nil test and a named reader's admitted).
- Unchanged and passing: the 1557 set tests (`worldcarrierrows_test.go`,
  `worldcarriers_test.go`), the SAV write and LOAD tests, the hashed World
  tests and the original-SAV import tests of both games in the ordinary
  `pkg/game` run.

## Open debt

- Step functions keep their definitions in their own files and are called
  only from the set: `applyOriginalDeadWithOrigins` (`originaldead.go`),
  `reserveWorldEffectKeys` and `currentWorldEffects`
  (`currentworldeffects.go`), `currentPlayerName` (`currentplayerroots.go`,
  through `keepLoadedPlayerName`), `projectSavedActorActions`
  (`savactoractions.go`), `graftUnknownObjects` (`currentunknowngraft.go`),
  `restoreAbsentSessionHead` (`currentsessionhead.go`) and
  `publishSessionEntry` (`sessionpublication.go`).
- The argument shape covers the two Snapshot fields only.
  `Mission.savedDocument` passed to a function is not checked; the LOAD
  passes it to its validators, matchers and readers at many sites. A read
  passed inside a composite literal is not checked either.
- Not reached by a fixture: `U5C`, the native Snapshot resume's session head
  absence, and on the second game the world effects, the order words and the
  dead body art.
- The release tests to run at the next release, on both roots of both games:
  the SAV corpus round-trip gate (`TestSAVRoundTripGateCorpusPart1`-`4`,
  `NewGameKitsPart1`-`3`), `TestReleaseEnemyCardOriginalSAVDiaryLevels`,
  `TestReleaseDeadHiddenActorStaysDeadAcrossSaveLoad`,
  `TestReleaseSecondOriginalTownLoadFollowsTheFile`, the terminal actor
  release tests (`TestReleaseTerminalActorBeforeFirstSAV`,
  `TestReleaseNonGroundTerminalActorReloads`), the original world effect
  release tests and the milestone-2 family
  (`engine/scripts/check-milestone2-acceptance.sh`).
