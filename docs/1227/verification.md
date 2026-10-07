# Paused candidate verification

These records belong to `6caa6a19caa46ab37a0caa8e2690e0cb9b1ad012`.
They are not evidence for the reconciled candidate. Current evidence is in
`story.md` and the resumed candidate receipts.

This is Phase 1 evidence only. Phase 2 (merge current engine main, run
`scripts/check-milestone2-acceptance.sh` on both lawful roots) is explicitly
not reached; the seat sequences it after story1225 lands.

## Observable result

Outside this story's own test files: the save dialog's own control list and
`SaveDialogSeams.Prepare` refusal (`pkg/ui/save_dialog.go`,
`pkg/game/savedialog.go`), proved by `TestSaveDialogOffersOnlySAV` and
`TestCurrentSaveDialogRejectsLegacyOutputBeforePreparation`, and the
whole-module `CheckNoAGSWriter` build-failing guard
(`internal/archtest/agswriter.go`), proved against the real module by
`TestNoProductionCallerWritesAGS`. This story does not touch script/trigger
routing or shipped-data decoding, so the missionrun script-node census below
is expected unchanged, not unmeasured.

This was not driven through the live GUI this session: no window was
verified foreground, so no synthetic keystroke was sent. The proof above is
the headless dispatch and whole-module scan the test suite runs through the
same production code paths a GUI save would use; it is not a substitute for
having watched the dialog on screen.

## missionrun UNSUPPORTED census

`go build -o /tmp/mr ./cmd/missionrun`, then for each mission:
`AGAINROM_ASSETS=<seat>/gameversions/en /tmp/mr -mission <N>
-trace -ticks 1 | grep -c UNSUPPORTED`.

| mission | result |
|---|---|
| 10 | 0 |
| 20 | 0 |

Unchanged from expectation for a save-writer-only change. For context, not
re-run this session: `pipeline/milestone-baseline.txt` (a separate, deeper
multi-tick instrument behind `pipeline/check-milestone.sh`, not the tool
this recipe runs) records `en m10 script 16 checks, 27 instants, 12
triggers` and `en m20 script 14 checks, 15 instants, 11 triggers`; this
story has no reason to expect either figure moved and did not attempt to
reproduce that deeper measurement.

## Gate results

- `gofmt -l` on every file this story touches (16 files): clean, zero
  files.
- `go test -trimpath -count=1 ./...` (`GOCACHE=<seat>/.gocache`):
  55 packages `ok`, zero `--- FAIL` or `FAIL` lines, zero occurrences of
  `0xc0000142`/`STATUS_DLL_INIT_FAILED` (checked directly against the run's
  own log; five lanes were running concurrently on this machine when this
  ran, and the seat separately reported that signature hitting nine tests
  in its own chain under the same load — none of this run's fresh-process
  release tests hit it).
- `scripts/check-no-game-assets.sh`: `check-no-game-assets: clean (tree
  scan)`.
- `pipeline/sav-export-census.sh`: not run (it reaches `engine/saves/` and
  `builds/current/`, both outside this worktree's remit). Read in full: its
  one invocation is already `saveconvert -assets ... -in <ags> -out
  <scratch>/<name>.sav -to sav`, the surviving AGS-read/SAV-write
  direction; `cmd/saveconvert/main.go`'s current `-to` flag requires
  exactly `sav`, matching this invocation exactly. No script change is
  needed for this story. The seat should re-run it once `builds/current/`
  is rebuilt from this story's landed commit, to reconfirm the exported
  count is unchanged, since that is a live rebuild against the owner's real
  `.ags` corpus this worktree cannot reach.
- `check-release-tests.sh`, `scripts/check-milestone2-acceptance.sh`: not
  run, per the seat's explicit phase-1 instruction.

## Storyguard

`internal/storyguard/baseline.go`: `TestIdentCount` fell to 5847 (removing
a round trip through the retired `-to ags` direction removed its one call
to a pre-existing story-numbered test helper name). `TestFileCount` stays
271. `CommentBytes` rose to 7793347, justified in the same commit by two
paragraphs naming every file and why (test fixtures now build their legacy
AGS bytes directly instead of round-tripping through the removed
direction). All six comment-form counts stay at zero. `go test
./internal/storyguard/...`: all subtests pass.

## Pre-existing failures re-confirmed, not fixed

- `TestReleaseOwnerFidelityHUDAndSavedCursor`: known baseline debt before
  this story.
- `TestReleaseCitySalesUseSAVAndFreshProcesses`: "partial sale changed
  fields outside independent semantic diff" — byte-identical failure text
  in `review/hotfix-1d18eae/release-tests.log` at base `1d18eae`.
- `TestReleaseMageSpellbookTrainSAVEAndContinuation`: "fresh SAV-AGS-SAV
  differs" against `game0010.sav` — byte-identical failure text in the same
  baseline log.

None of the three run under plain `go test ./...` (they need
`AGAINROM_ASSETS`/`AGAINROM_SAVE_CORPUS` and `releaseFront(t)` skips
without them), so they do not appear in the gate result above; they were
re-run directly against real corpus data this session solely to confirm
they predate this story, then left untouched.

## Divergences

Three new rows in `docs/divergences/persistence-current-sav.md`
(DIV-1395..DIV-1397, all `FIDELITY-DEBT`/`OPEN`), summarized in
`docs/1227/story.md`'s Open debt section. File total: 28 rows, matching its
own header.
