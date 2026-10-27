# The test chain to the seven-minute target

## Intent

The seat's final chain passed in 665 s on release commit `c4834b96`: release
660 s, go test 467 s, milestone-2 423 s, second-game save 403 s, scenarios
70 s. The owner's measure is the chain's seven-minute target (420 s), not a
share of deleted tests. This story applies the test audit's order of work,
steps 1 to 6, and adds a failure ledger. The chain gets one switch for
release mode, in which every root runs every release test and every
milestone-2 instrument.

## Authority

- The test audit `pipeline/reviews/test-audit-d944752e.md` in the seat
  repository: its delete list, shrink list, order of the work and owner
  decisions. Its per-test tables are `review/test-audit-d944752e/q1-census.tsv`
  to `q5-cost.tsv`.
- The owner ruling "Test chain" in the seat's `pipeline/OWNER-RULINGS.md`:
  - the measure is the chain's 420 s target;
  - the release tests the audit lists as reading no installed text or art run
    on EN only, on both roots at every release commit and on a change to a
    loader or to root data;
  - the milestone-2 leg runs on EN, on both roots on a change to SAV reading or
    writing and at milestone acceptance;
  - of the 14 corpus audits no gate runs, those a story proof names join the
    milestone-2 leg and the rest are deleted with a note in that proof;
  - the map editor modes test walks one row in four, the full walk on request;
  - the 201 tests that fail only on an error return stay;
  - a failure ledger (test, root, class, commit) lives where no cleanup
    removes it.

## As built

### Engine

- `internal/gatedtests/testdata/population.txt` takes an optional root column
  after the test. `en` marks a test the chain runs on the first root (EN)
  only; no column means every root. 15 rows carry `en` (see Root column). A
  test added later takes no column and runs on every root.
- `TestScanMatchesTheCheckedInPopulationList` reads the column, refuses any
  value but `en` and refuses a test listed twice. `TestPopulationRootColumnOnlyFalls`
  holds the `en` row count at `firstRootOnlyRows` (15): a rise fails, a fall
  fails until the constant is lowered in the same commit.
- `pkg/ui.TestMapEditorModesHoldEveryKeyAndGesture` walks rows 0, 4, 8 and so
  on of the 103-row table (26 rows). A skipped row is still compared by name,
  so the row order and count stay checked. `MAPEDITOR_MODES_FULL=1` walks all
  103 rows; `MAPEDITOR_MODES_WRITE` always walks all rows.
- `scripts/check-milestone2-acceptance.sh` also selects
  `TestProjectileCorpusAudit1133`, `TestCurrentWorldSAVAdmission1170` and
  `TestCurrentWorldSAVStrideQualification1170`.
- Deleted, the audit's 18:
  - 15 opt-in corpus audits in `pkg/game`: `TestGroupRateTermCorpusCensus`,
    `TestSAVD56OriginalGroundSackLineage`, the four
    `TestReleaseWriterCensus*` tests, `TestRoundTripObservationUsesCurrentMissionHoldings`,
    `TestCorpusResavePreservesCurrentGroupOrders`, `TestLowSAVResaveCorpusCensus`,
    `TestRoundTripRosterPermutationKeepsMemberChecks`,
    `TestRoundTripRosterIdentityCannotHideMissingMembers`,
    `TestSAVHiredPoolMigrationBound`,
    `TestLootConstructionCapturePreservesAbsenceNamesAndInputOwnership`,
    `TestSAVWriterLootContinuationWitness`, `TestCorpusDyingMoverSpeedRoundTrip`;
  - `TestReleaseGeneratedWidePoolsSAV`, a subset of
    `TestReleaseGeneratedWideNumericContinuationSAV`; the proof line in
    `docs/1256/story.md` now names the continuation test;
  - `TestTownScreenImplementsTheWorldMapPresentationSeam`, now a file-level
    `var _ ui.TownWorldMapScreen = (*townScreen)(nil)` in `pkg/game/worldmap_test.go`;
  - `pkg/rules.TestExactFormulaAgreesWithTheOriginalTable`, which only logged.
- Deleted under the ruling's corpus-audit clause: `TestAttackTargetSaveCensus`
  and `TestPermanentMercenaryOrderCorpusRoundTrip`. No story proof names them;
  a `docs/HOTFIXES.md` row names each (rows for `ed7d75a6` and
  `0e1fdf06`/`3ce6dd90`/`af4819ff`, frozen as historical records) and DIV-886
  names the second, now with a note of its deletion.
- Test helpers only the deleted tests used are removed with them
  (`pkg/game/savcensus_world_test.go`, `savcensus_release_test.go`).
- `internal/gatedtests/testdata/corpusaudit.txt` loses the 17 deleted corpus
  audits (63 rows); the population loses `TestReleaseGeneratedWidePoolsSAV`
  (1498 rows).

### Root column

Rule: a row carries `en` only when the test and every helper it reaches read
no data that differs between the EN and RU installs. Engine words
(`pkg/words`, embedded in the binary) are not install data.

What differs, from an entry-by-entry comparison of the two installs (read
only, the review's `resdiff.py` and `entrydiff.py`): installed text and the
main menu, global map, hall of fame and character generation art in
`main.res` (446 entries); 88 voice files in `sfx.res`; `font2` in
`graphics.res`; 11 movie payloads in `Allods/video4.res` and `video8.res`;
maps 100 and 140 in `scenario.res` and the loose maps (38 on EN, 34 on RU);
the `world.res:data/data.bin` Humans rows `F_Kadagan` and `NPC10_` and a
block near `M120_BrigandChief`. Equal by root: the other 2722 `graphics.res`
entries (unit, terrain, structure, object and spell art), the other
`scenario.res` maps and `music.res`.

Count: 15 of the 726 rows that carried `en` keep it; 711 lose it.

Method, over the 726 rows in order:

1. Output comparison. 65 rows print different output on EN and RU in a
   both-roots chain at `570632bd` once timings, paths, PIDs, save labels and
   root names are removed (the review's `outdiff2.py`). They lose `en`. This
   is a lower bound.
2. A script. A name-level call graph over every Go file of the module, test
   and production, from each test to a loader of data that differs:
   `NewFrontEnd` and `NewModFrontEnd` (main-menu and chargen art, sounds,
   fonts, `data.bin`, text), `LoadDefinitions*` and `LoadTable` (`data.bin`),
   `LoadTextTable`, `LoadInstallWords` and the other text readers,
   `LoadFont`, `OpenSounds`, `OpenSpeech`, `LoadCampaign` and the global map
   and hall-of-fame readers, and the movie path. 643 more rows reach one
   through calls and lose `en`; 689 of the 708 rows in steps 1 and 2 reach a
   front end. The graph also reaches a loader from each of the 65 rows of
   step 1.
3. Reading. The 18 rows the graph reaches no loader from, or reaches one only
   through a method-name edge, were read with their helpers. 3 lose `en`:
   `pkg/video/smacker.TestReleaseSmackerDecoderAgainstOracles` decodes both
   movie archives, and two mission-10 tests in `pkg/game` open the archives,
   whose base detection reads `main.res`. 15 keep it: they read only
   `graphics.res` art and registries, `scenario.res` map 20, `music.res`, the
   owner save corpus or a witness or dependency file.

The table (row, kept or removed, by which step, reason) and the script are in
the story's review notes (`en-column.tsv`, `tools/enclass.py`).

Expected wall. The ordinary mode now runs 1483 rows on RU against 772
before. Over this population and `.gocache.release-timings.json`, the
planner's plans give RU 4566 test seconds (2013 before) and a 12-worker pool
model of 812 s, the same as release mode. Over the old population the model
gives 1.17 to 1.26 times the measured pools of the three chains at
`570632bd` (567 s against 449 s ordinary; 812 s against 665 s and 697 s in
release mode), so the expected release leg, with about 11 s of prebuild and
retries, is about 655 to 705 s, and the ordinary chain about 660 to 715 s
(the two release-mode chains took 683 s and 714 s), against the 420 s target.

### Seat chain scripts

- `pipeline/final-chain.sh --both-roots` is release mode. Without it the
  release leg runs the `en` rows on the first root only and the milestone-2
  leg runs on the first root (EN) only. With it both legs run every row and
  every instrument on both roots. The first line of `status.txt` is
  `mode both-roots` or `mode ordinary`.
- The go test leg runs `pkg/game` in four processes (`unitparts=4`).
- `pipeline/check-release-tests.sh` reads the root column. A later root runs
  only the rows without a column unless `AGAINROM_RELEASE_ALL_ROOTS=1`
  (`final-chain.sh` sets it from `--both-roots`). A first-root-only test that
  runs on a later root fails the run. `pipeline/release-shards.py` plans the
  first root and the later roots from the same manifest with `--all-roots`.
- `pipeline/check-rom2-save-tests.sh` runs two subject streams per locale
  (stream 0: TownTwo, Town([^T]|T[^w]), Campaign, Stage and the other
  packages; stream 1: Game[A-M], Game[^A-M], Mission(Named|Notice|Save|Ten),
  MissionTwenty, Later, Movie). Each of the 120 `pkg/game` subjects matches
  exactly one group. `TestReleaseSecondTownTwoContinuation` runs alone
  (TownTwo) because the town group passed its 240 s budget on both locales in
  a release-mode chain. From the recorded test seconds of three chains, no
  group is expected past 240 s: the largest are TownTwo 80 to 158 s and the
  other town subjects 81 to 143 s; a stream takes 197 to 324 s. The streams'
  JSON joins into the locale's file before the 123-subject manifest check.

### Failure ledger

`pipeline/chain-failure-ledger.py` runs at the end of every final chain and
appends to `C:\Users\Eugen\againrom\.gocache.failure-ledger.tsv`, beside
`.gocache.test-timings.txt`, outside `review\` (`.gitignore` covers
`/.gocache*`). One tab-separated row per failure, appended and never
rewritten:

```
test  root  class  commit  leg  chain  utc
```

- `test`: `<package>.<Test>` relative to the module, the scenario file, the
  gofmt file, or the leg's log name when a leg failed without naming a test.
- `root`: the install directory's name (`en`, `ru`, `rom2-en`, `rom2-ru`), `-`
  for a leg that reads no install.
- `class`: `fail`, `panic`, `timeout`, `incomplete`, `build`, `known` (a
  release failure on `pipeline/release-known-fails.txt`), `gofmt`, or
  `leg-exit N`.
- `commit`: the engine commit the chain ran (`sha.txt`); `chain`: its out-dir.

A chain whose ledger could not be written fails with that reason.

## Proof

- `go test -count=1` of `internal/gatedtests`, `internal/storyguard`,
  `internal/archtest`, `internal/divledger`, `pkg/rules`, `pkg/ui` and
  `pkg/game`; `scripts/check-fast.sh`.
- Map editor loss control: the story 1474 defect reapplied in a scratch copy
  (the `ui:menu-fit` gesture pressing the Passability row of the View menu
  instead of Fit) fails the 26-row sample on 21 rows.
- The three audits joined to milestone-2 pass on EN and RU in focused runs;
  the nine below fail on EN.
- Seat harnesses: `pipeline/tests/check-release-tests.test.sh` (five new
  cases), `pipeline/tests/release-shards.test.py` (12 cases),
  `pipeline/tests/chain-failure-ledger.test.py` (3 cases).
- Two full chains at the final heads, one with `--both-roots` and one
  ordinary, with their leg walls from `status.txt`. They are recorded in the
  story's review notes, not in this file.

## Open debt

- Nine opt-in corpus audits that story proofs name fail on today's corpus
  (focused EN run with `AGAINROM_SESSION_CORPUS_AUDIT=1`):
  `TestSessionCorpusAudit1130`, `TestCellRecordCorpusAudit1131`,
  `TestSpellEffectCorpusAudit1132`, `TestMoverRouteCorpusAudit1134`,
  `TestDiaryCorpusAudit1135`, `TestGroundContainerCorpusAudit1136`,
  `TestUnitResidualCorpusAudit1138`, `TestNativeResumeCorpusCycle1139` and
  `TestSAVU130CorpusResaveKeepsTheLoadedAggregate`. Joining them would turn
  the milestone-2 leg red; deleting them would remove a check a proof names.
  They stay opt-in, neither joined nor deleted, until the owner decides.
  `TestSAVU130CorpusResaveKeepsTheLoadedAggregate` fails on two files with a
  resaved aggregate that differs from the loaded one (for example
  `2026-10-05/game0005.sav`, "Danath", 14912452 against 15146770); whether this
  is a save defect or a stale expectation is Unknown.
- The ordinary chain is expected at about 660 to 715 s, not 420 s: with 15
  `en` rows the ordinary release leg runs nearly every row on both roots.
  Reaching the target needs a release leg that is faster per row or an owner
  decision on which rows may run on one root. The next release chain measures
  the wall.
- The call graph resolves names, not types, so a row it removed may in fact
  read nothing that differs; such a row runs on both roots, which costs time
  and checks more, not less.
