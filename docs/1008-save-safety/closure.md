# 1008 — save safety closure

## Result

The ordinary `.ags` writer now publishes through a sibling temporary file. It writes the complete
encoded value, synchronizes it and closes it before publication. Windows publication uses
`MoveFileExW` with write-through and without replacement. Name reservation and publication are one
operation. Six synchronized helper processes writing in the same second produced six distinct
files. Every pre-commit failure path attempts close and removal and leaves all older saves
unchanged. Cleanup errors are returned with the primary failure.

On non-Windows systems, exclusive hard-link creation is the commit point. Failure to remove the
private source name afterward produces an internal committed-with-cleanup result: `Write` returns
the final name successfully and never attempts to roll that valid row back. Retirement removes the
writer from the active set and runs recovery under the same physical-directory lock. In the
two-writer witness, both private unlinks and the first retirement recovery are refused; the last
writer then reaps both inactive aliases. Active and foreign-process paths stay excluded. Only an
operating system that keeps refusing every retirement and next-write recovery can leave one hidden
alias per writer already active when refusal began. A later writer refuses before creation, so that
residue cannot keep growing. If the owner process exits first, hidden names can remain and are not
claimed to have been removed. The non-Windows protocol synchronizes contents before commit but
does not claim directory-entry power-loss durability.

Active staging identity now follows the operating system's physical directory identity and
platform filename semantics. A deterministic relative/absolute alias witness pauses one writer at
publication while the other runs recovery; both complete under distinct names. The physical test
also preserves the process-id namespace which keeps independent processes apart.

An `.ags` load now prepares a complete replacement before it can replace the running game. Mission
preparation reads and decodes the map, constructs the mission, validates simulation form 53 and its
world invariants, loads candidate hero bodies into an isolated unit-set clone, applies residue and
builds the viewer and runtime seams. Only an infallible commit opener adopts that candidate. A
refusal stays on the load window over the old mission and preserves the old campaign values,
canonical world bytes and live presentation caches.

A committed synthetic fixture fixes the released policy at envelope version 1 and simulation form
53. It contains no game asset. This is an exact-version compatibility baseline, not a migration
chain. The version-1 decoder requires the file to end at the declared payload boundary and the
payload to end after exactly one gob snapshot.

The complete `.ags` envelope is capped at 16 MiB. List checks the physical size and the declared
size from a bounded prefix before showing a row. Read checks one open handle and then streams at
most the ceiling plus one byte, so file growth after Stat remains bounded. A sparse 64 MiB save is
refused before content allocation. A wire preflight also limits the checksummed gob to 1,024 type
definitions, 128 nesting levels and 65,536 aggregate wire-field/array/slice/map entries before the
standard decoder can allocate `Snapshot`. A tiny payload claiming a 1,048,576-entry residue map is
refused after allocating less than 4 MiB; bypassing preflight allocated 37,874,824 bytes before gob
reached EOF. The encoder runs the same policy and cannot emit an unreadable over-budget save.
Oversized, declared-oversized, over-populated, trailing and future-version files are refused, while
the released fixture and ordinary saves still round-trip byte-exact. A file made oversized after
the load window opens reports the refusal there and preserves the live mission.

The prepared mission opener uses a concurrency-safe once gate. Two simultaneous invocations wait
on one commit and receive the same already-prepared seams.

## Contract coverage

| Requirement | Evidence | Result |
|---|---|---|
| FR-1, AC-1, AC-2, AC-10, AC-11 | `TestAtomicSavePublicationBoundaries`, `TestSaveFailureReportsCleanupRefusalAndRecoversOnRetry`, `TestSaveRecoveryRetriesRemovalBeforeCreatingAnotherTemp`, `TestSaveRecoveryDoesNotRemoveAnotherProcessTemp`, `TestLinkedPublicationMakesLinkCreationTheCommitPoint`, `TestSaveStoreReportsCommittedLinkedSaveAndRecoversItsPrivateName`, `TestConcurrentLinkedCommitsReapAllRetiredAliases`, `TestActiveSaveTempIdentitySurvivesDirectoryAliases`, `TestConcurrentSameSecondPublicationIsExclusive`, `TestSaveStoreConcurrentProcesses`, production `saveFileOps` | PASS |
| FR-2 | Windows `publishSaveFile` calls no-replace `MoveFileExW` with write-through; the portable injected hard-link protocol fixes link creation as commit and carries source-unlink refusal as post-commit cleanup; `TestPublicationRefusesAndPreservesAnExistingSave` checks the occupied destination | PASS |
| FR-3, AC-5, AC-13 | `prepareRestore`, `missionOpenerMode`, `openPrepared`, `installCandidate`; `TestPreparedMissionCommitsOnceThroughTheLoadWindow`, `TestPreparedOpenerCommitsOnceAcrossConcurrentCalls` and `TestCandidateBodyCacheIsAdoptedOnlyAtCommit` | PASS |
| FR-4, AC-3, AC-4 | `TestFailedCandidateLoadKeepsTheRunningGame` covers stale form, invalid world, absent map and malformed map through `ui.App`; `TestFailedCandidateDoesNotPopulateTheLiveBodyCache` covers candidate-only body loading before a corrupt world refusal | PASS |
| AC-6 | Replacing the candidate seam with the former direct `Restore` call makes the stale-form subtest fail because Town, live driver or world pointer changes | PASS |
| FR-5, AC-7, AC-12 | `TestReleasedEnvelopeOneSimulationFormFiftyThreeFixture` byte-compares the current encoder and committed fixture, then decodes it; `TestDecodeRejectsAppendedBytes` and `TestDecodeRejectsTrailingDataInsideDeclaredPayload` cover both trailing boundaries; `TestSaveStoreBoundsAndScreensEnvelopeFiles`, `TestBoundedSaveReadStopsFileGrowthAfterStat` and `TestOversizedSaveRefusalKeepsTheRunningGame` cover the safe byte maximum, list and live UI; `TestDecodeRejectsHugeGobMapCountBeforeAllocation` and `TestEncodeSaveDoesNotProduceOverBudgetGob` cover the allocation-count policy on both sides | PASS |
| FR-6 | `DIV-026` reconciled; `DIV-095` through `DIV-100` added with open/closed status in reader terms | PASS |
| AC-8 | Existing save-directory, unique-name, traversal, mixed-list and UI label tests remain green in the full suite | PASS |
| AC-9 | `TestReleaseLoadCandidateKeepsTheLiveMissionUntilCommit` passes against both shipped roots | PASS |

The fixture's expected canonical world hash is `084d1cd674bfe3cd`; its literal snapshot has mission
10, gold 321, purse 777, completed mission 10, available mission 20 and offered mission 20. The
encoder reproduces the committed bytes exactly.

## Mutation evidence

Eighteen production mutations were run separately and were not committed:

- adding `MOVEFILE_REPLACE_EXISTING` to the Windows publish call made
  `TestPublicationRefusesAndPreservesAnExistingSave` receive success instead of a collision and
  made `TestConcurrentSameSecondPublicationIsExclusive` receive one name from both writers;
- removing `MOVEFILE_WRITE_THROUGH` made
  `TestWindowsPublicationIsWriteThroughAndNoReplace` fail;
- restoring the former shorter-only payload comparison and `body[:declared]` slice made
  `TestDecodeRejectsAppendedBytes` accept the appended byte;
- removing the second gob decode and EOF requirement made both the second-value and unread-byte
  cases in `TestDecodeRejectsTrailingDataInsideDeclaredPayload` accept their rechecksummed payload;
- discarding the cleanup removal error made all four primary-boundary cases in
  `TestSaveFailureReportsCleanupRefusalAndRecoversOnRetry` lose that secondary cause;
- bypassing pre-create recovery made `TestSaveRecoveryRetriesRemovalBeforeCreatingAnotherTemp`
  publish a new save while the refused staging file remained;
- passing `f.Units` into candidate mission construction instead of the cloned unit set made
  `TestFailedCandidateDoesNotPopulateTheLiveBodyCache` find the candidate body in the running
  map's cache;
- the production load seam was changed back to the former eager state replacement. The stale-form
  subtest failed because the town, live driver or world pointer changed.
- replacing physical `os.SameFile` directory identity with Go interface identity made
  `TestActiveSaveTempIdentitySurvivesDirectoryAliases` lose the paused relative writer's staging
  file under the absolute writer's recovery;
- removing the physical-versus-declared size equality from List made
  `TestSaveStoreBoundsAndScreensEnvelopeFiles` list the trailing file;
- removing the same-handle Stat ceiling made that test read the sparse file until the stream guard
  instead of refusing its reported 67,108,864-byte size before content allocation;
- changing the stream guard from the ceiling plus one byte to exactly the ceiling made
  `TestBoundedSaveReadStopsFileGrowthAfterStat` accept 16 MiB from an endless post-Stat stream;
- removing the declared-total ceiling from `DecodeSave` changed the declared-oversize refusal into
  an ordinary truncation and failed `TestSaveStoreBoundsAndScreensEnvelopeFiles`;
- removing the encoder ceiling made the same test produce a complete envelope larger than the
  reader's supported maximum;
- replacing `sync.Once` with the former boolean made both simultaneous calls in
  `TestPreparedOpenerCommitsOnceAcrossConcurrentCalls` enter the commit.
- restoring the former non-Windows rollback protocol made
  `TestLinkedPublicationMakesLinkCreationTheCommitPoint` report an error after the injected final
  unlink refusal and made `TestSaveStoreReportsCommittedLinkedSaveAndRecoversItsPrivateName`
  return an empty name while the final row remained visible.
- bypassing wire preflight before the standard gob decoder made the valid, rechecksummed hostile
  map-count case allocate 37,874,824 bytes before refusal; the bounded path allocates less than the
  test's 4 MiB ceiling;
- disabling the recovery pass that runs as each writer retires left two inactive private aliases
  after two concurrent committed hard links and two injected private-unlink refusals.

Restoring each production line made its focused command pass.

## Shipped-content witness

The real UI-to-game load path was exercised by
`TestReleaseLoadCandidateKeepsTheLiveMissionUntilCommit`. It opens shipped mission 10, navigates
from its map through the pause menu into Load, chooses a stale `.ags` candidate and proves that the
load window remains over the same live mission with the same canonical hash. It then chooses a
valid candidate and proves that a new live driver adopts the saved hash.

Both this witness and the existing generated-character campaign continuity witness passed against
each preserved root:

| Root | Candidate witness | Campaign continuity witness |
|---|---:|---:|
| EN | PASS | PASS |
| RU | PASS | PASS |

The repository's nine shipped headless scenarios also passed against both preserved roots after
the story branch merged current master. That merge brought the gates-screen world-map activation
seam. The cross-root run then exposed that the scenario runner selected the in-game SAVE and LOAD
rows by English display text. `HeadlessGameMenuAction` now resolves those controls by their stable
production action and reaches them through the same arrow/Enter dispatch, so the scenario
vocabulary remains language-independent while the visible labels remain installation-localized.
`TestHeadlessGameMenuActionsDoNotDependOnDisplayedWords` fixes that boundary with deliberately
non-English labels.

The story build is in `builds/1008-save-safety/`. Its `againrom.exe -check` production startup
reported 66 map rows for EN and 62 for RU, with all 8 button mask regions and the same generated
hero summary on both roots. No GUI automation was used on the owner's desktop.

The worktree `missionrun` measured 0 unsupported nodes for mission 10 and 0 for mission 20 on each
root. Across the current 28-mission census it measured 59 on EN and 59 on RU, matching the expanded
implementation baseline. The story was not intended to move that counter.

## Twelve-aspect closure

| Aspect | Status | Closure |
|---|---|---|
| Data | PASS | The synthetic `.ags` fixture fixes envelope 1 plus sim form 53 without containing shipped bytes; the complete envelope is bounded at 16 MiB and its aggregate gob container population at 65,536. |
| Runtime state | PASS | Old and candidate live objects and mutable body caches coexist until the concurrency-safe one-shot commit adopts the candidate. |
| Simulation | PASS | Existing unmarshal version and invariant checks run during preparation; `formatVersion` remains 53. |
| Player input | PASS | The existing Save and Load controls route through the new seams; headless witnesses resolve their stable actions independently of localized display text, and no new player gesture or binding exists. |
| AI | N-A | No AI rule or representation changes; restored AI stays within existing canonical world bytes. |
| UI/HUD | PASS | Refusals leave the load screen, backing map, selection and row in place and show the returned error. |
| Triggers/scripts | PASS | Saved trigger state is unmarshaled into the fully constructed candidate mission before commit. |
| Inventory/equipment | PASS | A failed load preserves current party and carried state; this story adds no item representation. |
| Persistence/save-load | PASS | Active temp identity is physical, retired aliases are recovered serially, publication is exclusive and no-replace, reads and gob container allocations are bounded, the envelope contains exactly one gob snapshot, and preparation is non-destructive until one commit. |
| Campaign/session | PASS | Town, carried, offered and live mission state change only after successful candidate construction. |
| Shipped content | PASS | The production seam passed on mission 10 from both preserved EN and RU installations. |
| Existing mechanics | PASS | Default directories, override, unique naming, traversal refusal, mixed original lists and menus remain covered. |

## Divergence reconciliation

`DIV-026` remains the single original-save coverage row. It now distinguishes the supported
between-mission party and purse from the original campaign progress and offers that are not
restored, and keeps unsupported original world state in that same row.

- `DIV-095` records exact simulation-form refusal, the authored byte and gob-container ceilings and
  the absence of migration. The released fixture remains byte-exact and accepted; a customized
  envelope-1/form-53 save above either ceiling is refused.
- `DIV-096` records the dependency on current-install campaign, map and presentation assets and the
  absence of a content fingerprint.
- `DIV-097` records direct final-name writing, the replacing-rename race, the lexical active-temp
  alias defect, the non-Windows rollback/status contradiction and concurrent retired-alias
  accumulation as fixed by no-replace publication, physical identity, an explicit hard-link commit
  result and serialized retirement recovery. It distinguishes a surfaced pre-commit cleanup
  failure from a successful final row whose post-commit private alias awaits recovery, states the
  truthful continuous-refusal residue limit and names the non-Windows directory-sync limit.
- `DIV-098` records destructive `.ags` load ordering and the first revision's shared body-cache write as fixed by isolated candidate construction, and records the concurrency-safe one-shot commit.
- `DIV-099` records the omitted ROM save-menu controls and option rows.
- `DIV-100` records that corrupt or unreadable files remain absent from the load list.

Allocated ids `DIV-101` and `DIV-102` were not needed and are permanently returned in the ledger.
No row duplicates `DIV-026`, `DIV-093` or `DIV-094`.

## Gates

On the committed story tree before final review:

- implementation: `go build ./...`, `go vet ./...` and `go test -trimpath -count=1 ./...` passed;
- implementation formatting selected no Go file, and `scripts/check-no-game-assets.sh` reported
  `clean (tree scan)`;
- the focused Windows safety set passed, including two synchronized callers, six helper processes,
  a paused relative/uppercase-absolute alias pair, occupied-destination preservation,
  write-through/no-replace flags, bounded envelopes and gob populations, one concurrent commit and
  isolated candidate body loading;
- the portable non-Windows publisher set passed on Windows CI, including injected private-name
  refusal, prohibition of final-name rollback, committed `Write` status, repeated recovery refusal
  and eventual next-save alias recovery, plus two concurrent commits whose last retirement reaped
  both inactive aliases;
- `pipeline/check-scenarios.sh` selected and passed all 9 scenarios against EN and all 9 against
  RU, including the save/load and mission-to-town paths;
- research: `go build ./...`, `go vet ./...` and `go test -trimpath -count=1 ./...` passed;
- research `check-claim-ids.sh` selected 1,255 distinct ids across 31 ledgers and read 29 ids back
  through `tools/claim`;
- research `check-retraction-status.sh` selected 195 overturned ids and found every one marked.

The pinned research checkout has a pre-existing formatting baseline of 334 selected Go files. This
story changed none of them and does not report that baseline as a passing formatting gate.

## Remaining limits

Exact-version refusal, current-install content dependency, original campaign/world omissions, ROM
menu omissions and corrupt-file visibility remain open exactly as ledgered. An operating-system
cleanup refusal can leave one hidden staging name per writer that was already active when refusal
began; it cannot appear as a save or overwrite one. Retirement retries the inactive set, and a
future writer refuses before creating another file while any residue remains continuously
undeletable. On non-Windows systems the file contents are synchronized before hard-link
commit, but the containing directory is not synchronized and final-name survival across sudden
power loss is not claimed beyond the operating system's guarantees. A customized `.ags` above
16 MiB or whose aggregate gob container population exceeds 65,536 is refused rather than streamed.
The story did not add form 50–52 migration, original
difficulty or spellbook/session materialization, content fingerprint enforcement, original save
writing, typed naming, overwrite, deletion or autosave. These are out-of-contract limits, not
unclosed in-scope gaps.
