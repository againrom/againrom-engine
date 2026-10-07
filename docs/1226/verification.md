# Verification

This is the correction pass for `pipeline/reviews/story1226-review-2d2ab38.md`
(RETURN). All three findings are now confirmed, fixed engine bugs, each with
a fail-before/pass-after test; item D is resolved as a byproduct of Finding
C's fix. Observable result outside this story's own test files: the owner
kit's decoded bytes (below, honestly noting which findings this specific kit
pair does and does not show), and the milestone-2 census moving from one
disclosed refusal to zero.

## missionrun UNSUPPORTED census

`go build -o /tmp/mr1226 ./cmd/missionrun`, then for each mission:
`AGAINROM_ASSETS=<EN root> /tmp/mr1226 -mission <N> -trace -ticks 1 | grep -c UNSUPPORTED`.

| mission | before (engine main, `pipeline/milestone-baseline.txt`) | after (this branch) |
|---|---|---|
| 10 | 0 (no UNSUPPORTED line recorded for any mission in the baseline file) | 0 |
| 20 | 0 | 0 |

Unchanged. This story is a persistence fix (camera floor, sack token, roster
duplication), not script/trigger routing, so an unchanged census is the
expected result.

## Owner kit as the player-facing observable

`review/owner-sav-story1226/game9254.sav` (the pre-correction candidate's own
kit, kept for comparison) and `review/owner-sav-story1226/game9255.sav` (this
correction pass's own kit), both produced by loading
`gameversions/saves/2026-09-24/game0017-victory.sav` through
`RestoreOriginal`+`OpenMission` and SAVEing immediately through the same
`FrontEnd.Snapshot(true)` -> `playerMissionSave` -> `ExportCurrentSave` path a
player's own SAVE dialog uses (`pkg/game/missionsave.go`,
`pkg/game/savedialog.go`):

- `game9254.sav`: 524,171 bytes, sha256
  `fcc61390226e8f1d64ab221084407f9ca7f0db4de471e35b830034fcdf56c99b`.
- `game9255.sav`: 522,031 bytes, sha256
  `38dee6d28f60ed4fb621ce9d219f176e06a97d49604bd12c1c5087cce97bd204`.

Decoded with `cmd/savtool info`: `game9255.sav` carries chooser label
`9255 s1226 m131 fixed`, mission 131, difficulty 2, players 7, Danath as
player 0 (HUMAN, outcome 1 COMPLETE), world blocks 7099, cell records 252,
actors 102 owner-graph (100 living, 2 dead, 97 carrying a map unit id) --
identical population counts to `game9254.sav`, as expected for the same
source file.

**What this specific kit pair shows and does not show.**

- Finding C (heroes drawn as NPCs after this engine's own SAVE+LOAD) is what
  this kit demonstrates: `game9255.sav` is exactly the SAVE+LOAD round trip
  the fix targets. This is the finding to check on screen.
- Finding B (sack size) does NOT differ between the two files:
  `go run ./cmd/savtool sacks game9254.sav game9255.sav` prints identical
  `t1c=414985` (sack at (69,35)) and `t1c=137768` (sack at (123,86), gold
  124568) for both. `game0017-victory.sav`'s own two ground sacks are loaded
  and resaved untouched, and an untouched sack keeps its own original `T1C`
  bytes by design (`savedobjectground.go`'s "Existing source Sacks retain
  their exact row"). The fix is proven by
  `TestReleaseSackTokenValueRoundTripsFromTheOriginal` and the owner's own
  `game0002-bigsack.sav` settling save, not by this kit pair.
- Finding A (camera floor) does NOT differ between the two files either:
  both decode to `/View/X`=40, `/View/Y`=8 -- identical to
  `game0017-victory.sav`'s own value, since Y is already exactly at the
  floor and needs no correction. The fix is proven by
  `TestApplicationCurrentRawFloorsViewOriginToTheOriginalsOwnBound` and the
  62-file corpus census, not by this kit pair.

This was not driven through the live GUI this session (no `rom.exe` window
was brought forward); the owner's own play of `game9255.sav` -- confirming a
party hero draws its own body rather than an NPC, on the map and on its
portrait -- is what confirms Finding C on screen, per
`review/owner-sav-story1226/README.md`'s predictions.

## Gate results

- `gofmt -l .`: clean, zero files.
- `go test -trimpath -count=1 ./...` (`GOCACHE=.gocache`, corpus env vars
  unset): 55 packages `ok`, zero `FAIL` lines, tree-wide.
- `scripts/check-no-game-assets.sh`: clean (tree scan).
- `pipeline/check-release-tests.sh <EN root> <RU root>` (one invocation,
  `AGAINROM_IMPL=<this worktree>`, checkpoint `79296e7`): EN root failed with
  30 distinct top-level test names (47 lines counting subtests); RU root
  failed with the same 30 names (47 lines), byte-identical to EN's own set.
  Both match, name for name, `review/hotfix-893b408/fails.txt`'s 30-name
  baseline (`893b408`, two commits below this story's own base `1d18eae`) and
  this pass's own pre-recorded `baseline-names.txt` capture. Zero new
  failures, zero newly-resolved failures, on either release root.
- `scripts/check-milestone2-acceptance.sh <EN root> <RU root>`: see below.
- `pipeline/check-preserved-installs.sh`: clean, every root as recorded.
- `internal/storyguard`: `TestLiveTreeClean` passes. `CommentBytes` raised
  7791500 -> 7801334 for this correction pass's own new code and comments
  (`internal/storyguard/baseline.go` names every file: resume.go's dedup
  comment, `pkg/sim/sack.go`'s `SackTokenValue` and its callers, `savapplication.go`'s
  `viewOriginFloor`/`originalViewOrigin`, the new
  `savapplication_viewfloor_test.go`, and updated comments in
  `legacyprojection_test.go`, `savapplication_projection_test.go` and
  `savroundtrip1195_corpus_test.go`). `TestIdentCount` and `TestFileCount`
  are unchanged at 5848/271: this pass's one new test file and its one new
  test name carry no digits.

## milestone2-acceptance

`scripts/check-milestone2-acceptance.sh <EN root> <RU root>`, run against
both lawful roots with `AGAINROM_IMPL=<this worktree>` and
`AGAINROM_AGS_CORPUS=<main checkout>/saves`: both roots' own `TestMilestone2*`,
`savbyteidentity1197`, and `savroundtrip1195` instruments finish `ok`
(`ok againrom/pkg/game 194.293s` and `180.016s`), zero `FAIL` lines.

Same corpus both roots: the per-instrument reader census audits 106 files (65
world-half, 40 between-mission, 1 unreadable -- the same pre-existing
AGS-magic exclusion, not a new refusal). The narrower
`SAV-ROUNDTRIP-ORIGINAL-CENSUS` (original-SAV-only scope) reports
`discovered=105 exact=105 disclosed=0 migrated=0 accepted=105 refused=0
mismatched=0 comparator-exercised=105` on both roots. This is the direct
measurement of item D's resolution: the pre-correction candidate's own
disclosed refusal (`discovered=104 ... refused=1`, `game0017-victory.sav`'s
own `EXPORT-AGS` aggregate-cap refusal) is gone outright, not merely
disclosed, once Finding C's fix stops `CurrentRoster` from duplicating the
entry party. `discovered` rose 104 -> 105 in the same measurement because this
pass also added `gameversions/saves/2026-09-24/game0002-bigsack.sav` (Finding
B's own settling save) to the corpus root.

## Coordinator scope item D: disposition (updated from "disclosed" to "resolved")

The pre-correction candidate traced the `EXPORT-AGS` refusal for
`game0017-victory.sav` to three large top-level `Snapshot` fields
(`CurrentRoster` 19507, `Party` 12785, `SavedDocument` 32242 elements) and
concluded, wrongly, that none was an engine-side duplicate. `CurrentRoster`'s
own 19507 elements included exactly the Finding C duplication: resume.go's
own tail loop restated every entry-party hero's fuller
`mission.party`-derived clone on top of `Start.Roster`'s own simpler entry
for the same id. With that duplication fixed, a direct rerun of
`TestSAVRoundTrip1195OriginalCorpus` (`sessioncorpusaudit` build tag) shows
the refusal gone (see census above); `pkg/game/savroundtrip1195_corpus_test.go`'s
own baseline comment and `sav1195OriginalBaseline.refusals` map are updated
to match. AGS remains retired (SAV is the only save format); this resolution
is a byproduct of the Finding C fix, not new investment in the AGS export
path.

## origin/main reconciliation

`origin/main` is unchanged at `1d18eae` (this story's own base, the restored-
hero hover hotfix) throughout this correction pass; confirmed an ancestor of
this branch's HEAD immediately before the final gate chain. story1225 (hero
health derivation) and story1227 (AGS writer removal), separate worktrees,
had not landed as of this check; neither touches `pkg/game/resume.go`,
`pkg/game/savapplication.go`, `pkg/sim/sack.go`,
`pkg/sim/savedobjectground.go`, or `pkg/game/savcurrentitems.go`, so no
reconciliation was needed.
