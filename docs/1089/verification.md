# 1089 verification

Result: R and the formerly disabled Retreat cell queue persistent state 0x16.
On both preserved EN/RU roots, the installed App moves healthy hero 35 from
x=20 to x=19 away from hostile 0; automatic thresholds stay 0/0.
This is an implementation runtime result, not original-runtime parity.

Base: landed master `335672d822bb83fa0cf4e801764c2683c10d0f10`; integration
merge `c19eb467` includes Defend correction `8fae845f`. Research pin
`1172d41a` matches master and descends from `5ef2cdf2` (ancestry exit 0).
Native format 72, newer SAV state and all inherited release rows are retained.

## Sole correction checks

R1 defers new automatic casts during Retreat without clearing AutoSpell or
interrupting pre-existing wind-up/recovery. R2 carries each selection in one
Retreat-only event and assigns one fresh queue tag per press. Generic group
batching and the same-press first-member refusal remain unchanged.

The exact immutable overlay command in
`pipeline/reviews/story1089-pass1.md` passes all 14 leaf cases (six top-level
tests). Armed Retreat moves to x=9, with zero new releases; the independent
paused presses get Group 1/2 and the healthy actor reaches x=15. All per-tick
native comparisons pass. All three probe/overlay SHA256 values match the report.

`go test -trimpath -count=1 ./pkg/sim ./pkg/ui ./pkg/game -run
'Retreat|Defend|Autocast|AutoCast|Scroll|Withdrawal|Group|BookWindup'`: PASS.
Added regressions retain genuinely prior autocast work and AutoSpell, resume
the setting after a replacement, and distinguish one invalid mixed press from
two independent presses. gofmt, diff whitespace and no-assets checks pass.
The correction-built missionrun reports EN M10/M20 unsupported 0/0, unchanged.
No second review or repeat full branch chain is claimed; root owns final gates.

## Pre-review author checks (`5f3202e6`)

- `go test -trimpath -count=1 ./...`: PASS after correcting the destination
  writer registration and FR-2 explanation. The first run found only the stale
  `withdrawFrom` name after extraction of `withdrawFromAny`; the corrected
  architecture package and full rerun pass. No gameplay change followed.
- Focused sim/UI/game/gated-test packages: PASS. New tests first reproduced
  stale scroll targeting and an unrefunded approach. R/panel now clear targeting;
  an unstarted approach returns the exact item, while a started scroll releases
  once before Retreat. Native next-Retreat/next-Move, refused-first-member and
  later-scroll-replacement checks pass without changing book admission.
- EN/RU `TestReleasePlayerRetreat1089AppPanelAndNative`: PASS. Three native
  checkpoints per root restore exact bytes, enqueue nothing, and match 65 ticks.
  Retreat's 34x34 region matches the independently named installed BMP rectangle.
- EN/RU installed Defend App/native and four-state command-panel art tests:
  PASS, no skips. Defend's three checkpoints each match 33 ticks.
- Production App scenarios per root: 1087 22/22, 1089 19/19, 1090 25/25; exit 0.
  Only 1090's input/output paths were redirected in private copies. Commands
  are unchanged; native saves stay under this worktree's ignored
  `.local/final-en` and `.local/final-ru`.
- `check-div-claims.sh --ids`: 291 live rows, 440 distinct IDs, no malformed
  rows, 72 retraction-bearing rows. DIV-348/575 use the amended whole-list
  centroid and distinct HP gates. DIV-201 retains the TOWN-091 correction and
  uses corrected panel ownership, Stand Ground and Retreat labels.
- Allocation pre/post: 32 namespaces, missing answers 0. No new IDs.
- Preserved installs: 181 files, both roots as recorded. No-assets tree scan
  clean; gofmt and diff whitespace clean; candidate-only coauthor trailers absent.

## Milestone and drive

The lane-built `.local/missionrun-final.exe` passes the complete paired
28-map milestone gate: "the script gap and the drive are where they were
recorded, both roots". The unattended drive remains lost at tick 304, with
4 of 36 units moved and 1 fallen; that outcome is not a victory criterion.

| EN mission | Master baseline checks/instants/triggers | Current | Unsupported before / after |
|---|---|---|---|
| 10 | 16/27/12 | 16/27/12 | 0 / 0 |
| 20 | 14/15/11 | 14/15/11 | 0 / 0 |

Counts come from `pipeline/milestone-baseline.txt` and successful
`-trace -ticks 1` runs. The baseline has no unsupported column; before values
are the measured Defend landing values. The observable change is the installed
Retreat command above, not a smaller script census.

Commands use `GOCACHE=<seat>/.cache/go-build` and
`GOFLAGS=-buildvcs=false`. Local receipts are `.local/final-*`.

## Remaining scope

Root owns one fresh-context Retreat review, final merge gates, the combined
EN/RU release chain and rebuilding `builds/current/`. This lane did not write
the shared build, drive the desktop or run the original executable.
DIV-574/575/576 retain intermediate-death admission, exact original action
timing, dense-list byte wrap, corpse occupancy and transitive lifecycle debt.
