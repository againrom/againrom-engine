# Verification

Observable result outside this story's own test files: the regenerated owner
kit's decoded bytes (below), and the unchanged gate census against master's
recorded baseline.

## missionrun UNSUPPORTED census

`go build -o /tmp/mr ./cmd/missionrun`, then for each mission:
`AGAINROM_ASSETS=<EN root> /tmp/mr -mission <N> -trace -ticks 1 | grep -c UNSUPPORTED`.

| mission | before (engine main, `pipeline/milestone-baseline.txt`) | after (this branch) |
|---|---|---|
| 10 | 0 (no UNSUPPORTED line recorded for any mission in the baseline file) | 0 |
| 20 | 0 | 0 |

Unchanged. This story is a current-SAV field-export and character-generation
fix; it does not touch script/trigger routing, so an unchanged census is the
expected result, not an absence of measurement.

## Owner kit as the player-facing observable

`review/owner-sav-story1224/game9250.sav` (mission 10, warrior) and
`game9251.sav` (mission 20, archer), produced by the same
`MissionParty(nil, nil, nil)` / `ExportCurrentSave` path the original
story1222 kit used. Decoded and compared against the prior kit
(`review/owner-sav-story1222/game9248.sav`, `game9249.sav`):

| field | old kit (game9248/9249) | new kit (game9250/9251) |
|---|---|---|
| hero `U4C` (class flag) | `0x6` | `0x0` |
| hero mover `+0x08/0x09` | `5,255` | `0,0` |
| hero mover `+0x82/0x83` | post cell | `0,0` |
| hero mover `+0x84/0x85` | `128,128` | `0,0` |
| hero order `+0x00/0x01` | post cell | `0,0` |

Regenerated a second time at the correct seat path
(`<seat>\review\owner-sav-story1224\`, not the worktree's
own `review/`) with byte-identical SHA-256 hashes both times, confirming
deterministic reproducibility:

- `game9250.sav`: 141877 bytes, sha256
  `70f778b3d633910e96612879a53713632026ba4c65bde6cd3f7b7608237de5af`
- `game9251.sav`: 211492 bytes, sha256
  `75aa9efff830180d4bafe008f615057205ea51b808a1915a1d5f376f0c5bc73b`

This was not driven through the live GUI this session (no `rom.exe` window
was brought forward); the owner's own play of the regenerated kit is what
confirms the fix on screen, per `review/owner-sav-story1224/README.md`'s
steps.

## Gate results

- `gofmt -l .`: clean, zero files.
- `go test -trimpath -count=1 ./...` (`GOCACHE=.gocache`): all packages
  `ok`, zero `FAIL` lines, tree-wide.
- `scripts/check-no-game-assets.sh`: `check-no-game-assets: clean (tree scan)`.
- `pipeline/check-release-tests.sh <EN root> <RU root>` (one invocation,
  `AGAINROM_IMPL=<this worktree>`): each root's own known-red gated
  population reduces to exactly 30 unique root-level failing test names
  (subtests collapsed to their root). EN's 30 names equal RU's 30 names
  exactly (`diff` exit 0). This combined 30-name set is byte-identical
  (`diff` exit 0) to the pre-existing baseline recorded in
  `review/story1222-land-c93f58a/release-tests.log`
  (`grep "^--- FAIL:"`, stripped to root names). Zero new failures, zero
  newly-resolved failures, on either release root.
- `scripts/check-milestone2-acceptance.sh <EN root> <RU root>`: see below.
- `pipeline/check-preserved-installs.sh`: `check-preserved-installs: ok --
  569 file(s), every root as recorded`.
- `pipeline/check-div-claims.sh` (`AGAINROM_IMPL=<this worktree>`): soft
  instrument, 143 rows matched (expected/normal per the script's own
  doc comment). Neither new divergence row (DIV-1388, DIV-1389) triggered a
  wording problem; DIV-1388 was flagged only for AI-POST-042's own
  already-known, already-accounted-for retraction/amendment history.

## milestone2-acceptance

`scripts/check-milestone2-acceptance.sh <EN root> <RU root>`, run once
against both lawful roots from this worktree with `GOCACHE=.gocache`: both
roots' own `TestMilestone2*`/`TestReleaseGeneratedCityConversion1166`-family,
`savbyteidentity1197`, and `savroundtrip1195` instruments finish `ok`
(`ok againrom/pkg/game 172.091s` EN, `175.901s` RU), zero `FAIL` lines. Same
census both roots: 104 files audited (63 world-half, 40 between-mission, 1
unreadable -- `2026-08-27/EXP-0261-owner-runs/game9000.sav`, a pre-existing
AGS-magic file this script already excludes by design, not a new refusal),
2929 raw tagged actors, 0 scalar mismatches, 0 refused; SAV-ROUNDTRIP-
ORIGINAL-CENSUS discovered=103 exact=103 disclosed=0 migrated=0 accepted=103
refused=0 mismatched=0. This story's changes (hero exclusion from the
AI-start repair, hero class-flag fallback) touch no field this instrument's
own corpus exercises -- both fixes are specific to `MissionParty(nil, nil,
nil)`'s generated-mission-SAV path, and the milestone-2 corpus is real
original saves -- so an unchanged clean result is expected, confirming no
regression to original-save resume.

`pipeline/check-preserved-installs.sh` (run again after this save-reading
gate, from the seat root): `check-preserved-installs: ok -- 569 file(s),
every root as recorded`.

## origin/main reconciliation

`origin/main` is unchanged at `c93f58a` (story1222's own landing) throughout
this story; confirmed an ancestor of this branch's HEAD immediately before
the final gate chain.
