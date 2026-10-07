# Difficulty verification

Base: `bd7d7b60958ea8a3d63e96e2ae004fb5433742e3`.
Research pin unchanged: `949c5572384c549494ac75372ca19db1c578e9d8`.

## Production result

The actual `cmd/againrom -headless scenarios/1082-difficulty.json` drive
records a Normal pre-create draft and a Hard campaign (`difficulty: 3`,
`screen: map`, one party member) after Play. The seat scenario gate reports
`ok (1 of 1)` on both EN and RU. Process-start chargen and menu NEW GAME share
NewGameOpener; the installed test exercises both entrances.

The installed arithmetic witness covers all three difficulty values:

| Mission | Units | Unchanged Humans | Unchanged party |
|---|---:|---:|---:|
| 10 | 19 | 16 | 1 |
| 20 | 41 | 15 | 1 |
| 30 | 26 | 13 | 2 |

The witness completes missions directly through FinishMission; it is not a
combat-victory claim. It verifies byte-exact native map resume, candidate
difficulty independent of the previous live value, town save/load and next
mission, and bounded invalid-difficulty refusal. Original-import tests use
authored SAV headers over installed maps; they prove Head.Difficulty wiring,
not original actor-state completeness. Synthetic tests additionally cover a
frozen pre-field native gob schema, source-city provenance fallback, failed
new-game preservation, mouse/keyboard draft controls and Forward/Back/Reset.

Installed Levels art fits at (60,196), (0,110), (48,65), with sparse colour
difference scores 4598/7171/13725 on EN. Pointer-selected on-state comparison
checks 3862/8154/12292 nonblack pixels. The queen's later-painted hero overlap
is excluded. The composed Hard frame was inspected; no desktop input or game
window was used. These are measured fits, not decoded ROM1 coordinates.

## Census and gates

The mandatory missionrun trace at one tick reports EN m10/m20 UNSUPPORTED
**0/0 before and 0/0 after**. The unchanged seat milestone baseline reports
m10 16 checks/27 instants/12 triggers and m20 14 checks/15 instants/11 triggers;
it does not contain UNSUPPORTED counts. This UI/session slice changes no
script-support population. Its observable result is the production selector
and persisted campaign value, not a lower script census.

- gofmt and git diff --check: clean.
- `go test -trimpath -count=1 ./...`: PASS, asset-free, reviewed candidate
  `e08a92ac` (the full gate chain below predates the bounded correction).
- Paired `check-release-tests.sh`: EN 103/103 and RU 103/103 PASS,
  0 lacked a subject on either root (6 packages, 103 gated tests).
- `check-scenarios.sh` with filter 1082: EN and RU each 1/1 PASS.
- `check-div-claims.sh`: exit 0; 266 live rows, 358 claim IDs; existing
  66 partial-retraction warnings. DIV-532 cites the active clauses read from
  the pin. Allocation sweeps before/after: missing answers 0.
- `check-no-game-assets.sh`: clean (tree scan). No file deletions or research
  pin change. The story commit has no task or Co-Authored-By trailer.

Git Bash cache creation needed elevated execution. Scenario builds and the
release gate's nested 386 cutscene build needed `GOFLAGS=-buildvcs=false`;
the first paired run failed only on that VCS-stamping error. No code change
was made between those gate attempts. Assets remained read-only. Raw gate
output, drive JSON and rendered frames are untracked under
`review/story1082/` in the seat.

## Bounded review correction

The sole review returned visible hero/Levels input overlap. A new synthetic
regression first reproduced `(147,174)` choosing Hard instead of the female
fighter. The corrected hit order gives nonblack hero-mask pixels priority,
while black-key/transparent gaps retain the underlying Levels hit.

Focused `TestDifficulty|TestChargenDifficulty`: 8 tests PASS. Installed
`TestReleaseDifficultyLevelsInstalledArtAndPointer`: PASS on EN and RU; all
175 independently decoded overlapping hero pixels resolve to choice 2,
`(147,174)` selects that hero while retaining Normal, and all three Levels
centers still select the expected difficulty without changing the hero.
Coordinates, arithmetic, saves and research pin are unchanged. Gofmt and
diff-check are clean. The full branch chain was not repeated; final merge
gates, counterexample confirmation and builds/current rebuild belong to the
seat, without a second adversarial review.
