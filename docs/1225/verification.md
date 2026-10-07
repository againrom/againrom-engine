# Verification

Observable result outside this story's own test files: the regenerated owner
kit's decoded bytes (below), and the release gate's FAIL-set diff against
master's recorded baseline.

## missionrun UNSUPPORTED census

`go build -o /tmp/mr ./cmd/missionrun`, then for each mission:
`AGAINROM_ASSETS=<EN root> /tmp/mr -mission <N> -trace -ticks 1 | grep -c UNSUPPORTED`.

| mission | before (engine main `1d18eae`) | after (this branch, `3ac5bd0`) |
|---|---|---|
| 10 | 0 | 0 |
| 20 | 0 | 0 |

Unchanged, confirmed by building and running both binaries directly. This
story is a character-generation and stat-derivation fix; it does not touch
script/trigger routing, so an unchanged census is the expected result, not an
absence of measurement.

## Owner kit as the player-facing observable

`review/owner-sav-story1225/game9252.sav` (mission 10) and `game9253.sav`
(mission 20), produced by the same `MissionParty(nil, nil, nil)` /
`ExportCurrentSave` path the prior story1224 kit used.

| file | label | bytes | sha256 |
|---|---|---|---|
| `game9252.sav` | `9252 s1225 m10 health` | 141,878 | `bf30c69ae8de9ca863ee505270c93bf41f4aba46f77ecd4fa6d391e54ebc97f9` |
| `game9253.sav` | `9253 s1225 m20 health` | 211,493 | `19a504e28a9b678e20438f412724533d80832fe7b579010400d85ec41e44f773` |

Recomputed both hashes directly from the files on disk; they match
`review/owner-sav-story1225/README.md`'s own recorded values exactly.
Independently decoded both with `cmd/savtool party`: both read hero Danath,
body 43 reaction 26 mind 15 spirit 15 speed 17, health **145/145** (the prior
kit, `review/owner-sav-story1224/game9250.sav`/`game9251.sav`, read 100/100),
skills `[0 10 0 0 0 0]`, skill XP `[0 1593 0 0 0 0]`, total XP 0. The "total
XP 0" field is the evidence DIV-1390 cites for its own candidate mechanism.

This was not driven through the live GUI this session (no `rom.exe` window
was brought forward); the owner's own play of the regenerated kit is what
confirms the fix on screen, per `review/owner-sav-story1225/README.md`'s
steps.

## Targeted numeric confirmation

Run fresh (`-count=1`, bypassing the test cache) against real EN assets and
the real save corpus:

```
go test -count=1 ./pkg/game/... -run 'TestReleaseGeneratedHeroHealthMaximumFollowsTheDerivation|TestReleaseHeroHealthMaximumGroundTruthResave|TestMissionPartyHealthMaximumComesFromTheDerivation' -v
```

All three pass: `TestMissionPartyHealthMaximumComesFromTheDerivation` pins
the fixture derivation at 145; `TestReleaseGeneratedHeroHealthMaximumFollowsTheDerivation`
confirms both missions 10 and 20 export that same 145 through the real
`ExportCurrentSave` path; `TestReleaseHeroHealthMaximumGroundTruthResave`
confirms the owner's resave ground truth reads 137 for the byte-identical
fixture (body/reaction/mind/spirit, both skill arrays, health maximum).

## Gate results on `3ac5bd0` (pre-reconcile, base `1d18eae`)

- `gofmt -l .`: clean, zero files.
- `go test -trimpath -count=1 ./...` (`GOCACHE=.gocache`): 55 packages `ok`,
  zero `FAIL` lines, tree-wide. Run once on this commit (the tree changed
  after the predecessor's own last full run: a wording-only edit to
  `docs/divergences/character-generation.md` landed after it, and
  `internal/divledger`'s own test parses that file as part of `go test
  ./...`).
- `scripts/check-no-game-assets.sh`: `check-no-game-assets: clean (tree scan)`.
- `pipeline/check-release-tests.sh <EN root> <RU root>` (one invocation,
  `AGAINROM_IMPL=<this worktree>`): checkout confirmed at `3ac5bd0`, 9
  packages, 370 gated tests, 2 roots. Each root's own known-red gated
  population reduces to exactly 30 unique root-level failing test names
  (subtests collapsed to their root, same method both sides: `grep "^---
  FAIL:"` on the raw log, stripped to the name before any `/` or `(`). EN's
  30 names equal RU's 30 names exactly (`diff` exit 0). This 30-name set is
  byte-identical (`diff` exit 0) to the pre-existing baseline recorded in
  `review/hotfix-1d18eae/release-tests.log` (independently re-extracted:
  94 raw `--- FAIL:`/indented-subtest lines collapse to the same 30 unique
  root names). Zero new failures, zero newly-resolved failures, on either
  release root. No "started/declared test did not complete" or
  manifest/census mismatch lines anywhere in the log.
- `pipeline/check-div-claims.sh` (`AGAINROM_IMPL=<this worktree>`): exit 0,
  soft instrument, 518 live rows citing 762 claim ids, 143 rows flagged for
  the pre-existing "cites a claim carrying a retraction row" note (expected
  per the script's own doc comment). Neither DIV-1390 nor DIV-1391 is named
  individually in that list.
- `scripts/check-milestone2-acceptance.sh <EN root> <RU root>`
  (`AGAINROM_SAVE_CORPUS=gameversions/saves`): **FAILED on both roots**,
  read directly from the script's own verdict line (`check-milestone2-
  acceptance: root ... FAILED (exit 1)`), not inferred from a piped exit
  code. Both failures are the same single test,
  `TestSAVRoundTrip1195OriginalCorpus`, refusing the same two corpus files
  on both roots: `2026-09-24/game0002-bigsack.sav` and
  `2026-09-24/game0017-victory.sav`, both via
  `pkg/game/savegob_preflight.go`'s gob aggregate-element ceiling ("gob
  slice count N exceeds safe aggregate maximum 65536"), a guard this story
  never touches. Census on the EN root: `discovered=105 exact=103
  accepted=103 refused=2`.

  This is not this story's own regression. `git diff --stat 1d18eae --
  pkg/game/save.go pkg/game/savegob_preflight.go
  pkg/game/savroundtrip1195_corpus_test.go` on this branch is empty: every
  file this guard and this test touch is byte-identical to base `1d18eae`.
  `preflightSaveGob` constructs a fresh, function-local scanner per call
  (`pkg/game/savegob_preflight.go`), so its aggregate count cannot leak
  between files or between test runs; the same code given the same file
  bytes must charge the same total. A same-code, same-corpus-path re-run
  of the identical single test directly against a clean `1d18eae` checkout
  minutes later (`go test -tags sessioncorpusaudit -v -run
  '^TestSAVRoundTrip1195OriginalCorpus$'`) read `discovered=105 ... refused=0`
  for the same 105-file corpus: same population, zero refusals. The seat
  reported mid-gate that story1226 (landing as `88e20cf`, touching
  `pkg/sim/sack.go` and `pkg/game/savedobjectground.go`) pushed during this
  window; the two refused files' own names name a sack/container fixture.
  The most likely account is a concurrent lane writing or replacing those
  two files in the shared, git-ignored `gameversions/saves/2026-09-24/`
  directory between this gate's read and the follow-up check, not a defect
  in either story's own code. This is not established at instruction level,
  only by the code-identity diff and the timing the seat itself reported;
  recorded here as an observation, not a claim. Re-run on the merged,
  reconciled head below.

## origin/main reconciliation

`origin/main` was unchanged at `1d18eae` (this branch's own base) for the
entire pre-reconcile gate chain above; confirmed immediately before each
gate. The seat reported `origin/main` advancing to `88e20cf` (story1226's
own landing) during this story's release-tests/M2 run.

`88e20cf` merged into this branch as `a7cfc7b`, base still `1d18eae`.
story1226 touched `pkg/game/resume.go`, `savapplication.go`,
`pkg/sim/sack.go`, `savedobjectground.go`, plus its own docs/tests; every one
of those files merged cleanly with zero conflicts. Two files conflicted,
both the expected shared-ratchet files:

- `internal/gatedtests/testdata/population.txt`: both sides added
  `TestRelease*` rows to the same region; resolved by keeping both story's
  rows in alphabetical order (this story's 2 plus story1226's 4, 378 lines
  total).
- `internal/storyguard/baseline.go`: both sides added a justifying paragraph
  above `var Committed` and bumped `CommentBytes`. Resolved by keeping both
  paragraph stacks in full (this story's paragraph, then story1226's three),
  then adding one further reconciliation paragraph, then recomputing
  `CommentBytes` with `go run ./internal/storyguard/cmd/measure` on the fully
  merged tree rather than adding the two branches' deltas by hand — the tool
  is the authority on the live tree, arithmetic on two independently-recorded
  deltas is not. Converged value: `CommentBytes: 7806489`.

  The first draft of that reconciliation paragraph named the two stories by
  their bare numbers ("story1225"/"story1226"), which is itself a
  `storymention` violation. `go test -trimpath -count=1 ./...` caught it
  before this merge commit was made (`TestLiveTreeClean` /
  `storyguard_test.go:32: comment form "storymention": count rose from 0 to
  2`), matching every other paragraph in that file's own house style, the
  paragraph was rewritten to describe the two fixes by nature ("the generated
  hero health-maximum fix" / "the sack/resume round-trip fix") and the
  measurement re-run once more to confirm `storymention: 0` and the same
  `CommentBytes: 7806489`.

On the merged head (`a7cfc7b`): `gofmt -l .` clean; the 5 tests specific to
this story and to the just-merged sack/resume fix all pass individually with
real assets; `go test -trimpath -count=1 ./...` clean, 55 packages `ok`, zero
`FAIL`; `scripts/check-no-game-assets.sh` clean. Pushed to
`wt-story-1225-hero-health-derivation`; `git ls-remote
origin refs/heads/wt-story-1225-hero-health-derivation` matches local HEAD
exactly.

### Post-merge release-tests and M2 acceptance, on `a7cfc7b`

`scripts/check-milestone2-acceptance.sh <EN root> <RU root>`: clean on both
roots, read from the script's own output (no "FAILED" verdict line, no
`--- FAIL:` line, `EXIT=0`). Both roots'
`TestSAVRoundTrip1195OriginalCorpus` now read
`SAV-ROUNDTRIP-ORIGINAL-CENSUS discovered=105 exact=105 accepted=105
refused=0 mismatched=0` — the two refusals seen pre-merge
(`game0002-bigsack.sav`, `game0017-victory.sav`) are gone, refused count back
to 0 for the full 105-file corpus on both roots. This confirms the pre-merge
account above (transient concurrent-lane condition, not a defect in this
story's own diff): the same corpus, same guard code, now reads clean once
story1226's own landing — which independently reported fixing "the AGS
aggregate-cap refusal" in its own storyguard paragraph — is part of the tree.

`pipeline/check-release-tests.sh <EN root> <RU root>` on `a7cfc7b`: same
method as the pre-reconcile run (root-level `--- FAIL:` names collapsed,
subtests folded to their root). EN and RU each reduce to the same 30 unique
names, `diff` exit 0 between them, and `diff` exit 0 against the recorded
baseline (`review/hotfix-1d18eae/release-tests.log`). Zero new failures,
zero newly-resolved failures. This run is superseded by the second
reconciliation below (engine main moved again while it and the M2 run were
in flight); recorded here as evidence for this story's own diff, not as the
gate this branch ships on.

## Second reconciliation: the roster-duplicate reader hotfix

While the gates above were running, the seat reported a second hotfix,
`29b5def` (`origin/wt-hotfix-roster-reader`), about to fast-forward onto
`main` on top of `88e20cf`. Merged into this branch as `b60308e`, base still
`1d18eae`. The hotfix touches `pkg/game/world.go`, `pkg/game/resume.go`, a
new `pkg/game/rosterduplicate_hotfix_release_test.go`, and
`docs/HOTFIXES.md`; all four merged cleanly.

The same two files conflicted again, resolved the same way:

- `internal/gatedtests/testdata/population.txt`: kept both new rows,
  alphabetical order (`TestReleaseHeroDrawsItsOwnBodyFromALegacyDuplicateRosterEntry`
  then `TestReleaseHeroHealthMaximumGroundTruthResave`).
- `internal/storyguard/baseline.go`: kept all paragraphs from all three
  lines of work, added one further reconciliation paragraph, recomputed
  `CommentBytes` with the `measure` tool on the fully merged tree:
  `7809268`. While resolving this I also found and fixed a leftover
  inconsistency in this branch's own prior reconciliation paragraph from the
  first merge: its prose quoted `7806452` while the struct itself correctly
  read `7806489` — a stale figure from an earlier fixed-point iteration that
  was never synced to the final value in the same edit. Both numbers happen
  to be 7 digits, so correcting the prose could not itself have changed the
  measured byte count -- reasoned from the two strings' equal length, not
  verified by an isolated `measure` run against only that one-line change
  (the fix landed in the same edit pass as the hotfix's own new paragraph,
  and `measure` was run against the combined result).

On the merged head (`b60308e`): `gofmt -l .` clean; the 5 tests spanning all
three lines of work
(`TestReleaseGeneratedHeroHealthMaximumFollowsTheDerivation`,
`TestReleaseHeroHealthMaximumGroundTruthResave`,
`TestMissionPartyHealthMaximumComesFromTheDerivation`,
`TestReleaseHeroDrawsItsOwnBodyAfterOurSaveAndLoad`,
`TestReleaseHeroDrawsItsOwnBodyFromALegacyDuplicateRosterEntry`) all pass
individually with real assets and the real save corpus; `go test -trimpath
-count=1 ./...` clean, 55 packages `ok`, zero `FAIL`;
`scripts/check-no-game-assets.sh` clean. Pushed; `git ls-remote
origin refs/heads/wt-story-1225-hero-health-derivation` matches local HEAD
(`b60308ef21a1682d08511dc32f45c63aebe4d627`) exactly.

`pipeline/check-preserved-installs.sh` (run once, from the seat root, ahead
of this second reconciliation): `check-preserved-installs: ok — 569 file(s),
every root as recorded`.

### Release-tests and M2 acceptance on `b60308e`

`scripts/check-milestone2-acceptance.sh <EN root> <RU root>`: clean on both
roots (no "FAILED" verdict line, zero `--- FAIL:` lines, `EXIT=0`); both
roots' `TestSAVRoundTrip1195OriginalCorpus` read `discovered=105 exact=105
accepted=105 refused=0 mismatched=0`, same clean result as `a7cfc7b`, as
expected since this hotfix does not touch the sack/save-guard files.

`pipeline/check-release-tests.sh <EN root> <RU root>` on `b60308e`: same
method as before, EN and RU each reduce to the same 30 unique names, `diff`
exit 0 both against each other and against the baseline. A further hotfix
(`origin/wt-hotfix-save-refusal`) arrived while a second, later
release-tests run and the M2 run below were in flight, so `b60308e` is
itself superseded; recorded here as evidence for this story's own diff, not
as the gate this branch ships on.

## Third reconciliation: the Group-member-decay save-refusal hotfix

While the gates above were running, the seat reported a third hotfix chain,
`ad1b818` (`origin/wt-hotfix-save-refusal`), about to fast-forward onto
`main`. `ad1b818` already contains the roster-reader hotfix (`29b5def`,
already merged into this branch) plus a test-only mercenary regression pin
(`7b7e96b`) and the actual save-refusal fix (`7ee3346`/`23cc618`,
`pkg/game/savdocument.go`'s `snapshotSavedDocument` plus a new
`pkg/game/savgroupmemberdecay_test.go`). Merged into this branch as
`f2c8c55`, base still `1d18eae`.

`internal/gatedtests/testdata/population.txt` and `docs/HOTFIXES.md` merged
with zero conflicts this time (confirmed both underlying fix commits,
`47b3472` and `7ee3346`, still have their own ledger row).
`internal/storyguard/baseline.go` conflicted in the same tail region: kept
every paragraph from every line of work (this branch's own three-way
reconciliation, the mercenary regression's own paragraph, and the hotfix
side's own reconciliation of save-refusal with roster-reader), added one
further paragraph, recomputed `CommentBytes` with the `measure` tool on the
fully merged tree: `7812796`.

On the merged head (`f2c8c55`): `gofmt -l .` clean; seven focused tests
spanning all four lines of work now on this branch
(`TestReleaseGeneratedHeroHealthMaximumFollowsTheDerivation`,
`TestReleaseHeroHealthMaximumGroundTruthResave`,
`TestMissionPartyHealthMaximumComesFromTheDerivation`,
`TestReleaseHeroDrawsItsOwnBodyAfterOurSaveAndLoad`,
`TestReleaseHeroDrawsItsOwnBodyFromALegacyDuplicateRosterEntry`,
`TestReleaseHiredMercenaryKeepsItsOwnClassAfterOurSaveAndLoad`,
`TestCurrentSaveSurvivesGroupMemberFullDecay`) all pass individually with
real assets and the real save corpus; `go test -trimpath -count=1 ./...`
clean, 55 packages `ok`, zero `FAIL`; `scripts/check-no-game-assets.sh`
clean. Pushed; `git ls-remote
origin refs/heads/wt-story-1225-hero-health-derivation` matched local HEAD
(`f2c8c55a5635c9614ec53fb2885ba45cac9b8700`) exactly at push time.

A release-tests run and an M2 run were started on this head, but the seat
paused this lane before either finished and both background processes were
killed; neither result exists. `git log --format='%h
%(trailers:key=Co-Authored-By)' 1d18eae..HEAD` reads empty for every commit
on this branch (checked again on resume): no trailer of any kind anywhere in
the range.

## Final gate chain on the pause-resume head (`f2c8c55`, unchanged)

On resume, `origin/main` was confirmed at `ad1b818`, already fully contained
in `f2c8c55` (`git merge-base --is-ancestor ad1b818 HEAD`), so no further
merge was needed; the worktree was clean at `f2c8c55` exactly as the seat
described.

`pipeline/check-release-tests.sh <EN root> <RU root>`, run alone (not
concurrently with the M2 run below, per the seat's load note about
`0xc0000142`/`STATUS_DLL_INIT_FAILED` launch failures under five concurrent
lanes): zero `0xc0000142` or `DLL_INIT` occurrences in the log. EN and RU
each reduce to the same 30 unique root-level failing test names (same
method throughout: `--- FAIL:` lines collapsed to the name before `/` or
`(`). `diff` exit 0 between EN and RU, and `diff` exit 0 against the
baseline this resume was told to use,
`review/hotfix-29b5def/release-tests.log` (independently re-extracted: also
30 unique names, byte-identical set). Zero new failures, zero
newly-resolved failures, on either root.

`scripts/check-milestone2-acceptance.sh <EN root> <RU root>`, run alone
immediately after: clean on both roots, read from the script's own output
(no "FAILED" verdict line, zero `--- FAIL:` lines, `EXIT=0`). Both roots'
`TestSAVRoundTrip1195OriginalCorpus` read `discovered=105 exact=105
accepted=105 refused=0 mismatched=0` -- the same clean result seen on every
merge head this branch has carried (`a7cfc7b`, `b60308e`), consistent with
the diagnosis that the original two-file refusal was a transient
concurrent-lane condition from before story1226 landed, not a defect in
this story's own diff.

This is the gate result this branch ships on. `f2c8c55` is unchanged from
the third reconciliation above; nothing further was committed for these
results beyond this file.

## Fourth reconciliation: the periodic-lag text-smoothing hotfix

The seat reported `origin/main` advancing to `700e274` (the periodic-lag
hotfix: `pkg/ui`'s new `pixellog.go`, `textsmoothing.go`,
`pkg/render/textsmooth`, a new `pkg/game/textsettle_release_test.go`) ahead
of this story's sole review. Merged as a scoped reconciliation only:
`internal/gatedtests/testdata/population.txt` merged with zero conflicts;
`internal/storyguard/baseline.go` conflicted in the same tail region as
every prior reconciliation, resolved the same way (kept both paragraphs,
recomputed `CommentBytes` with the `measure` tool on the fully merged tree:
`7820444`). No file in story1225's own diff was touched by this hotfix.
`gofmt -l .` clean; the seat's requested scoped suite,
`go test -trimpath -count=1 ./pkg/game/... ./pkg/ui/... ./internal/...`,
clean (9 packages `ok`, zero `FAIL` -- including `internal/storyguard`
itself, so this merge's own conflict resolution is proven clean, not just
asserted); `scripts/check-no-game-assets.sh` clean. Per the seat's
instruction, the full release-tests and M2 acceptance chains were not
re-run here; the seat runs the final chain on the landing merge.
