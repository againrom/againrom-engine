# The left World carriers become rows of the one carrier set

## Intent

The World carrier set (`pkg/game/worldcarriers.go`) held most of the state a
SAV carries, but the Diaries, the dead bodies, the town purse and the LOAD of
the carrier absence kept their own calls on the SAV producer, the LOAD and the
mission seat. A fix to one of them went through its own call chain, outside
anything the set's guard checks. This story makes each of them a row of the
set and widens the guard so a carrier outside the set is a finding. No SAV the
engine writes and no state a LOAD builds changes, on either game.

## Authority

- Owner direction: a third of architecture audit `bdac396d`'s multi-way kinds
  become one builder each; the World carrier set is the audit's rank 1.
- Audit shape: each left carrier a row {home, capture, decode, absence,
  transform, seat, writer}; the Diary owners and the dead bodies' held state
  are the rows' decode data; the byte codecs stay.
- ROM1 behaviour is not changed. No DIV row is added or changed.

## As built

### Steps of a row

`worldCarrier` keeps its steps. Three take a point:

- `decode` is keyed by the record it reads (`carrierRecord`): the SAVE's
  `snapshotRecord`, the original archive walk once the living actors are
  admitted (`archiveLiving`) and once they are restored (`archiveDead`), the
  SAV document a LOAD binds (`documentRecord`) and the engine's actions leaf
  (`actionsRecord`). `decodeWorldCarriersFrom(from, c)` runs every row's
  decode of one record in row order; `decodeWorldCarriers(s, w)` is its
  snapshot point.
- `writer` runs at one `carrierStage`: `keysStage` and `lateStage` are the
  former early and late writers; `graphStage` runs on the mission document's
  graph inside `currentWorldDocument`, where the Diary and dead body
  projections ran.
- `seat` is keyed by the mission start it runs at (`carrierSeatPoint`):
  `entrySeat` is a mission door's entry (`seatWorld`), `lineSeat` is the
  mission `FrontEnd.AdvanceLine` starts with no door
  (`startLineMission`). `seatWorldCarriers(at, c)` runs every row's seat of
  one start in row order. The purse row seats at both. The Diary and
  Scenario bank rows seat only at `entrySeat`, so the headless line start
  keeps the World it had: the started mission given the town gold.

### Kinds and builders

| kind | builder |
|---|---|
| World carrier | `worldCarrierSet` (`pkg/game/worldcarriers.go`); every carrier below is a row |
| SAV document producer | `ExportCurrentSave` (`currentsave.go`) and its mission graph `currentWorldDocument` (`worldsave.go`); they run `writeWorldCarriers` at a stage |
| SAV LOAD | the original LOAD door (`prepareMission`, `originalrestore.go`), `importSavedDocument` (`savdocument.go`) and `restoreOriginalActions` (`savactioncurrent.go`); they run `decodeWorldCarriersFrom` at a record |
| mission seat | `seatWorld` (`missionentry.go`) and `FrontEnd.startLineMission` (`frontend.go`, for `AdvanceLine`); they run `seatWorldCarriers` at a start |
| byte codecs of the Diary and the dead state | unchanged: `sav.ReadDocumentDiary`, `sav.ProjectDocumentDiary`, `sim.ImportOriginalDiaries`, `sim.ImportOriginalDeadActors` and its held rule |

### Rows added or widened

| row (home) | step | hook | was called from |
|---|---|---|---|
| World purse of the local Player | seat `entrySeat` | `seatTownPurse` | `seatWorld`, after the resume branch |
| | seat `lineSeat` | `seatTownPurse` | `FrontEnd.AdvanceLine`, which wrote `SetPurse(SelfSlot, Town.Gold())` itself |
| World saved Diaries, by owner | decode `archiveLiving` | `applyOriginalDiaries` | `prepareMission` |
| | decode `documentRecord` | `importSavedDiaries` (owners: `savedDiaryRecords`, order `sim.CompareSavedDiaryOwners`) | `importSavedDocument` |
| | seat | `seatTownDiaries` | `seatWorld` |
| | writer `graphStage` | `projectSavedDiaries` | `currentWorldDocument` |
| World original dead bodies and their held state | decode `archiveDead` | `applyOriginalDeadWithOrigins`, whose import holds the absorbed body terminal (`sim.ImportOriginalDeadActors`, DIV-4281) | `prepareMission` |
| | writer `graphStage` | `projectDeadActors` | `currentWorldDocument` |
| World carrier absence (cell tails, cell records, motion cells, Sack keys, blocks, Diaries) | decode `actionsRecord` | `restoreCurrentCarrierAbsence` | `restoreOriginalActions` |

Each hook runs at the same point and in the same order as before. The two
seat rows now run at the top of `seatWorld`, before the projectile counter
seed and the resume branch: the purse row returns at once when the request
resumes a saved World (the base reached it only without one), and the Diary
row seats only a fresh mission, which never resumes. Neither the counter seed
nor the Scenario bank seat reads the purse or the Diaries. At the line seat
`seatTownPurse` reads a request with no snapshot, no initial purse and no
loaded document, so it writes `Town.Gold()`, the value `AdvanceLine` wrote.

### Guard

`internal/archtest/worldcarriers.go` names, for each hook, for each World
writer a row owns and for the Diary owner table (`savedDiaryRecords`,
`sim.CompareSavedDiaryOwners`), the functions that may reference them:

| World writer | owners |
|---|---|
| `ImportOriginalDiaries` | `applyOriginalDiaries`, `importSavedDiaries` |
| `SetSavedDiaries` | `seatTownDiaries` |
| `ImportOriginalDeadActors` | `applyOriginalDeadWithOrigins` |
| `SetPurse` | `seatTownPurse`; the Player purse binding of a loaded document: `importSavedPlayerPurses`, `savedPlayerPurseRows`, `restoreCurrentPlayerSlots` |
| `RestoreCurrentSackKeyAbsence`, `RestoreCurrentCarrierAbsence` | `restoreCurrentCarrierAbsence` |
| `SetROM2ScenarioState` | `secondCampaignRules.seatWorld` |
| `SetSavedProjectiles` | `leaveUnboundEffectsOut`; `seedProjectileCounter` |
| `SetSavedSpellEffects` | `leaveUnboundEffectsOut`; `applyOriginalSpellEffects` |
| `RestoreStructureUseMetadata` | `restorePointerSlots`; `mapWorld.restoreStructureUseMetadata` |
| `SetLoadedRecord` | the set; `restoreSavedDocumentDecoded` |

It also reports a function of `pkg/game` that a row of the set references and
the table does not name, so a new row's hook cannot stand outside the rule.
`TestWorldCarrierRuleNamesAPlantedBypass` plants a LOAD that decodes the Diary
itself, a LOAD that writes it into the World, a `FrontEnd.AdvanceLine` that
seats the town purse with `SetPurse` itself, and a row whose hook the rule
does not name; the rule names each.

### World writers of the rows outside the set

Production `.go` files under `pkg` and `cmd` were searched for the World
writers of the Diary, dead state, purse and carrier absence rows. Each caller
outside the set:

| writer | caller | route |
|---|---|---|
| `SetPurse` | `FrontEnd.AdvanceLine` | through the set's `lineSeat` |
| `SetPurse` | `importSavedPlayerPurses`, `savedPlayerPurseRows`, `restoreCurrentPlayerSlots` | left: they bind the Player purses a loaded document carries, the state the purse row yields to; a probe World in `savedPlayerPurseRows` |
| `CopyDiaryUnits`, `SetDiaryUnits` | `restoreOriginalActorStock`, `pkg/mapload` mission start | left: the Diary kill table built from the map, not a saved Diary |
| `RestoreAbsentStructureCells` | `restoreOriginalActions` | left: absent structure cells, not one of the absence row's carriers |

The other rows' writers keep the callers they had (the original spell effect
LOAD, the projectile counter seed, the map start's structure use); the owner
table names them, so a new caller is a finding.

### Carriers left

| carrier | site | why left |
|---|---|---|
| the loaded document kept for the mission writers (`s.loadedDocument = s.SavedDocument`) | `currentsave.go` | a decode onto the Snapshot needs the snapshot point to take the Snapshot by pointer; `worldcarriers_test.go` calls that point by value and must pass unchanged |
| terminal actor rows (`ImportCurrentTerminalActors`, held rule `CurrentTerminalActor.held`) | `savactoridentity.go` | produced inside the actor identity remap of the actions leaf; a row would need a second identity pass |
| the Token Reference misses capture | `decodeSavedDocument` | a field of the decoded record, not of the World; its writer is already a row |
| reads of the loaded record inside the generated document builder (`retainDeadValues`, `retainDeadBook`, `retainDeadEffects`, `loadedDeadObject`, `reserveRetainedDeadEffectKeys`, the identity joins) | `currentdeadconstruct.go`, `currentworldbuild.go` | per-record reads inside another builder; a row would be a per-record hook the set does not run |
| application state from the loaded document | `missionentry.go` (`settleMission`) | Presentation state, not World state |
| LOAD matchers of rows already in the set: book selection, order spell absence, effect Tokens, plane residue | `restoreOriginalActions` | each needs its own actions-leaf point; not in this story's census of left carriers |
| the document capture chain and its LOAD validators | `savdocument.go`, `worldsave.go` | as in story 1542 |

The census of `Document [!=]= nil` tests in non-test `pkg/game` files stays
78 lines in 48 files. One moved into the set (the purse seat); the line seat
adds none. The rest are
the nil-document guards of the document capture chain and its validators,
which this story leaves.

## Proof

- `TestTheLeftCarriersAreRowsOfTheSet`: the Diary, dead body, purse and
  carrier absence rows hold their decode, seat and writer steps at their
  points.
- `TestTheSetSeatsTheTownPurseAndDiaries`: a fresh seat through the set puts
  the town purse and the town's Diaries in the World; a loaded record that
  binds the Player purses keeps the World's purse.
- `TestTheGraphStageWritesTheDiaries`: the graph stage through the set writes
  nothing for an empty World and refuses a live Player's Diary the document
  has no record for.
- `TestTheLineStartSeatsOnlyTheTownPurse`: `startLineMission` on the
  continuity fixture's mission 10 with town gold 511 and a town Diary gives
  World hash `ebce66e3005d76dc`, equal to the start given the purse by
  `SetPurse` and to the hash the base's `AdvanceLine` lines give on the same
  fixture; the town's Diaries stay out.
- Unchanged and passing: `TestASaveKeepsTheLiveWorldAndAResaveItsBytes`,
  `worldcarriers_test.go`, the Diary and dead-state tests of `pkg/game` and
  `pkg/sim`. `TestSecondSeatWorldKeepsTheEngineDocumentsLiveBank` passes with
  the seat point added to its `seatWorldCarriers` call.
- Byte inspection over the SAV corpus (`gameversions/saves`, the walk of the
  corpus round-trip gate, plus its second-game files): base and candidate
  `savecheck -resave load` write equal bytes for each of the 128 first-game
  files (EN root) and for 44 of the 48 second-game files on `rom2-en` and
  `rom2-ru` (save directory route, `-saves D -name local-sav:<file> -resave
  load`). Both builds refuse the other four with the same message: one file
  the save directory scan does not list, and three mission files at "saving
  is not available for this game yet". The table is in the lane's return.

## Open debt

- The carriers left above.
- The release tests to run at the next release: the SAV corpus round-trip
  gate (`TestSAVRoundTripGateCorpusPart1`-`4`, `NewGameKitsPart1`-`3`) on the
  first game's EN and RU roots, the Diary and dead release tests
  (`TestReleaseEnemyCardOriginalSAVDiaryLevels`,
  `TestReleaseDeadHiddenActorStaysDeadAcrossSaveLoad`) and
  `TestReleaseSecondOriginalTownLoadFollowsTheFile` on both second-game roots.
