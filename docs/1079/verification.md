# School diamond verification

## Candidate and observable result

The story starts at implementation `2ee6eaa2fce218c6ed7d980565c6fd41ee9b4004`
and reconciles master `befe948d1e28f3135cf17ac961fcb106896735d1` before final
checks. Research is pinned to accepted
`4b49d6524016a32cc0f3414c122a7c327f6fe613`.

The former school omitted the diamond. The production school compositor now
shows it after admitted Train and restores the background on completion.
On each preserved EN/RU install, the peak changes 4,842 of the 6,080 pixels
in (200,60)-(280,136); completion restores all 6,080. Independent archive
decoding checks 54,720 pixels across all nine frames. The production Train
route yields the explicit 16-transition sequence with 15 draw-eligible states.

Untracked captures are `review/1079/{en,ru}/school-{idle,active,completed}.png`
at the seat. They are actual production CPU compositions using installed art,
not an original-game screenshot or a live window witness. No synthetic input
was sent to the owner's desktop. The EN active capture was visually inspected.
The seat owns the exact-master rebuild of `builds/current/` after landing.

## Checks

`gofmt` and `git diff --check` pass. The focused school, App paint/update,
new-game-reset, and gated-population tests pass. A full asset-free
`go test -trimpath -count=1 ./...` passes once on the final code.
`scripts/check-no-game-assets.sh` prints `clean (tree scan)`.

For each of `gameversions/en` and `gameversions/ru`:

```text
AGAINROM_ASSETS=<root> go test -trimpath -count=1 ./pkg/game -run '^TestReleaseSchoolDiamond' -v
```

Both tests pass on both roots. The optional `AGAINROM_SCHOOL_DIAMOND_PNG`
points to the corresponding untracked review directory. After capture,
`check-preserved-installs.sh` prints `ok — 181 file(s), both roots as recorded`.

The allocation sweep brackets the divergence edit and reports
`missing answers: 0` both times. DIV-524 and DIV-525 consume the reserved
range. `check-div-claims.sh` exits 0: 263 live rows, 341 cited IDs, and 65
rows with existing partial-retraction references. None of the new rows is
among those matches. DIV-147 moves to the closed ledger.

The seat must run its final EN/RU release chain through one invocation of
`pipeline/check-release-tests.sh gameversions/en gameversions/ru` with
`AGAINROM_IMPL` set to the landed checkout. The two new witnesses are registered
in `internal/gatedtests/testdata/population.txt`. No broad input/session
scenario changed; the direct production paint and App-update regressions are
the relevant headless witnesses.

## Mission census

Before implementation, a VCS-stamped `cmd/missionrun` build from the story's
base was run with EN assets and `-mission <N> -trace -ticks 1`. Both missions
exited 0 and printed zero `UNSUPPORTED` nodes. The population counters match
`pipeline/milestone-baseline.txt`:

| mission | baseline checks / instants / triggers | before unsupported | after unsupported |
|---|---|---|---|
| 10 | 16 / 27 / 12 | 0 | 0 |
| 20 | 14 / 15 / 11 | 0 | 0 |

The story changes presentation, not the script population. Both repeated
mission runs exit 0 and print the unchanged population counters.
`cmd/missionrun` and `cmd/againrom` built at code commit
`3d9ef996715ec486a2e376dfd0d51f22956cfa74`; `go version -m` reports that exact
revision and `vcs.modified=false` for both. `againrom -assets <en> -check`
also exits 0 with 66 map rows and 8 of 8 menu mask regions.

On this machine Go's VCS discovery skips a worktree's `.git` file and climbs
to the seat. The build command alone therefore receives explicit worktree
`GIT_DIR` and `GIT_WORK_TREE`, plus ephemeral `safe.directory` entries.
No build disables VCS stamping. The evidence-record commit changes prose only.

## Scope and remaining evidence

TOWN-154: frame paths, dimensions, opaque placement and draw gate.
TOWN-379: admitted Train and refusal preservation.
TOWN-380: one updater invocation per school room paint, no client-tick timer.
TOWN-381: full ordinary orbit, every reachable phase/direction retrigger,
peak repeat, and hidden cached zero-frame completion.
TOWN-382: school entry, picker non-reset, hidden leave, and reentry; the existing
reflection-backed new-game reset audit also covers the new presentation state.

No simulated or saved state is added. Training prices and skill gains are
unchanged. DIV-524 retains original display cadence, visible duration, later
reply effects and indirect event Unknowns. DIV-525 records immutable startup
frame caching instead of original entry/leave allocation. No original GUI
runtime, server-refusal path, column animation, or missing dynamic writer was
invented or claimed verified. Independent review, merge and current-build
publication belong to the seat.
