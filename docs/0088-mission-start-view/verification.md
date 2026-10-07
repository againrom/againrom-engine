# Verification — the mission opens where the party lands

Environment: `go1.26.1 windows/amd64`, the toolchain `go.mod` pins. Baseline is the fork point
`a41ad33`. Every figure below is copied from the command's own output. `go test` is run with
`-trimpath`, which is a local workaround for Windows Defender quarantining a test binary and changes
nothing about what is executed.

## Gates

Each was run separately and its own exit code read — never a pipeline's.

```
go build ./...                                        EXIT=0
go vet ./...                                          EXIT=0
gofmt -l $(git ls-files '*.go')                       EXIT=0, printed nothing
go test -count=1 -trimpath ./...                      EXIT=0, 31 packages ok, 0 FAIL
bash scripts/check-no-game-assets.sh                  EXIT=0   check-no-game-assets: clean (tree scan)
bash scripts/check-no-game-assets.sh --history        EXIT=0   check-no-game-assets: clean (history scan)
bash scripts/check-doc-budget.sh                      EXIT=0
bash scripts/check-sdd-audit.sh                       EXIT=0, zero FAIL
git diff --diff-filter=D --name-only a41ad33 HEAD     printed nothing — the deletion set is empty
```

The audit's `note`/`warn` counts are **not recorded as comparable figures**. This is a lane worktree,
where that section is guarded on `[ -d builds ]` (`check-sdd-audit.sh:481`); it emitted counts here
only because this story created its own `builds/0088-mission-start-view/`, which turned the section
on for every *other* story at once. Only the FAIL set is enforced and only it is compared.

Doc budget, this story's rows:

```
analysis.md    4603 /  7168 bytes  ok (64%)     spec.md    9235 / 13312 bytes  ok (69%)
provenance.md  5786 / 16384 bytes  ok (35%)     plan.md   10368 / 13312 bytes  ok (77%)
tasks.md T1  918 / T2 1393 / T3 1045 of 1400    legend+traceability  465 / 1200
0088-mission-start-view: plan <= 1.2 x spec     10368 <=  11082 bytes  ok
0088-mission-start-view: tasks <= 1.2 x plan     3821 <=  12441 bytes  ok
```

## Tasks and commits

Three implementation tasks, three trailered commits, one each; no `Co-Authored-By` anywhere.

```
40de59c  0088 T1: a camera arrives somewhere, not only a distance     0088-mission-start-view/T1
f002480  0088 T2: how much of the map opens, authored and said once   0088-mission-start-view/T2
5576d03  0088 T3: the mission opens where its own start put the party 0088-mission-start-view/T3
```

## What was executed

```
--- PASS: TestCenterOnPutsTheWorldPointAtTheViewCentre                  pkg/render/camera
--- PASS: TestCenterOnNearAnEdgeStaysInsideTheWorld                     pkg/render/camera
--- PASS: TestCenterOnKeepsTheClampInvariant                            pkg/render/camera
--- PASS: TestCenterOnLeavesASmallAxisCentredOnTheWorld                 pkg/render/camera
--- PASS: TestStartViewSpansTheAuthoredColumnsAndCentresOnTheCell       pkg/ui
--- PASS: TestStartViewRowsFollowTheViewProportions                     pkg/ui
--- PASS: TestStartViewSurvivesAnArmingWithNoViewSizeYet                pkg/ui
--- PASS: TestStartViewIsAppliedOnceAndThenTheViewIsThePlayers          pkg/ui
--- PASS: TestAViewerNeverArmedIsUnmoved                                pkg/ui
--- PASS: TestStartViewOnDisplacedGroundFollowsTheDrawnCell             pkg/ui
--- PASS: TestStartViewObeysTheExistingBounds                           pkg/ui
--- PASS: TestStartViewFollowsTheCellItWasGiven                         pkg/ui
--- PASS: TestAuthoredStartColumnsIsWhatTheApplicationUses              pkg/ui
--- PASS: TestPathCellCentreIsTheCameraOverTheWorldCentre               pkg/ui
--- PASS: TestStartViewCellIsTheFirstMemberOrTheDecidedDrop             pkg/game
--- PASS: TestStartViewCellReadsTheReportAndNotTheDrop                  pkg/game
```

## The headless drive — before and after, on a lawful install

A camera cannot reach a headless drive, so this is a free invariant and it was run as one. Both runs:
`AGAINROM_ASSETS=<ru install> go test -count=1 -trimpath -run TestTheTenthMissionIsDrivenToAWin
./cmd/missionrun`. It did **not** skip in either run — a skip would have been a failure to
investigate, not a pass.

At `a41ad33`:

```
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u21 -> (56,21) r3 : reached (43,46), Chebyshev 25, after 480 ticks
outcome lost at tick 480
```

At `5576d03`:

```
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u21 -> (56,21) r3 : reached (43,46), Chebyshev 25, after 480 ticks
outcome lost at tick 480
```

Identical, line for line. **This test is red at the fork point and is red here for the same reason**
— it reached `won at tick 2608` before `0086`, is under investigation elsewhere, and was neither
fixed nor edited by this story. What is claimed is only that the outcome and the tick did not move.

## Mutations

Four, chosen at the points where a wrong answer would look right rather than swept broadly. All four
were killed, and each was reverted immediately.

| Mutation | Killed by |
|---|---|
| `applyStartView` no longer clears `startArmed` (the one-shot becomes a follow camera) | `TestStartViewIsAppliedOnceAndThenTheViewIsThePlayers` — `twenty layouts moved the view from (3196,2856)@1.6 to (2896,2656)@1.6`, and `a resize rescaled the view to 2.1875, want the player's 1.6` |
| `cellWorldCentre` drops the displaced lift (`wy += 0`) | `TestStartViewOnDisplacedGroundFollowsTheDrawnCell` |
| the zoom is a literal `1` instead of the authored derivation | four tests: `…SpansTheAuthoredColumns…`, `…SurvivesAnArmingWithNoViewSizeYet`, `…ObeysTheExistingBounds`, `TestAuthoredStartColumnsIsWhatTheApplicationUses` |
| `startViewCell` prefers `Drop` over the placed party | both `pkg/game` tests |

## Criteria

| | |
|---|---|
| **AC-1** | met — `TestCenterOnPutsTheWorldPointAtTheViewCentre` at four scales and an off-lattice point; `TestStartViewSpansTheAuthoredColumnsAndCentresOnTheCell` at three view sizes |
| **AC-2** | met — `TestStartViewFollowsTheCellItWasGiven`; `TestStartViewCellReadsTheReportAndNotTheDrop` |
| **AC-3** | met — `TestStartViewCellIsTheFirstMemberOrTheDecidedDrop`, the three- and one-member arms, with a deliberate first-member-not-the-drop arm as the discriminator |
| **AC-4** | met — the same test's no-party arms, including the drawn fallback and an empty report |
| **AC-5** | met — `TestStartViewOnDisplacedGroundFollowsTheDrawnCell`, which asserts the fixture's lift is nonzero and the drawn centre differs from the flat one **before** asserting the centring, so the test cannot pass by agreeing with both |
| **AC-6** | met — `TestStartViewSpansTheAuthoredColumnsAndCentresOnTheCell` measures the span in whole columns; `TestAuthoredStartColumnsIsWhatTheApplicationUses` pins the zoom to the function's own value |
| **AC-7** | met — the same span test at 1024x768, 1600x900 and 600x1000; `TestStartViewRowsFollowTheViewProportions` |
| **AC-8** | met — `TestStartViewSurvivesAnArmingWithNoViewSizeYet`, which arms, drives `Layout(0,0)` as the front-end does before the window is known, and then checks the span against the size actually adopted |
| **AC-9** | met — `TestStartViewIsAppliedOnceAndThenTheViewIsThePlayers`: a pan, twenty layouts, then a resize |
| **AC-10** | met — `TestCenterOnNearAnEdgeStaysInsideTheWorld` at four corner and beyond-corner points; `TestStartViewObeysTheExistingBounds`, corner arm |
| **AC-11** | met — `TestCenterOnLeavesASmallAxisCentredOnTheWorld`; `TestStartViewObeysTheExistingBounds`, small-world arm |
| **AC-12** | met — `TestAViewerNeverArmedIsUnmoved`: origin and native scale, unchanged over ten layouts |
| **AC-13** | met — the headless drive above, identical before and after |
| **AC-14** | met **by construction, and that is the stronger evidence**: `git diff --name-only a41ad33 HEAD` names no file under `pkg/sim/`, so `formatVersion` (16) and every encoder are untouched; `pkg/sim`'s own suite, which pins the version and the digests, is green |
| **P-1** | met — `TestCenterOnKeepsTheClampInvariant`, 300 randomised worlds, view sizes, zooms and world heights, including NaN and infinite points, asserting the package's own `assertAxis` and that `VisibleTiles` stays inside the grid |
| **P-2** | met — `TestAViewerNeverArmedIsUnmoved` and `TestStartViewIsAppliedOnceAndThenTheViewIsThePlayers`, with the one-shot mutation as the discriminator |
| **P-3** | met — no test names `20`; the zoom mutation killed four tests at once, which is what one authored place looks like from the outside |
| **SC-1** | **not closable here** — see *Limitations* |
| **SC-2** | met — the zoom mutation; and the value is read through `AuthoredStartColumns()` at its single use site |
| **SC-3** | met — AC-12, and `LoadMapViewer` is not in the diff |
| **SC-4** | met — the headless drive, and AC-14's construction argument |
| **SC-5** | **the owner's, and no test can close it** — see *Limitations* |
| **SC-6** | met — the gate block above |

## Limitations

**SC-5 is the owner's and no evidence here bears on it.** Whether 20 columns is *the right amount to
see* is a judgement about a look. Every test above can only confirm the view spans the value this
project chose, which is a tautology about our own choice and is stated as one rather than dressed as
a pass. The number is authored, disclosed in `spec.md` and behind one function precisely so that his
answer — or a measurement of the original's viewport — costs one edit.

**SC-1 was not run from a window.** That a mission *visibly* opens on the party rather than on a
corner of empty ground is asserted here only through the unit level and the built binary; it was not
observed on screen in this lane. `builds/0088-mission-start-view/` ships the binary and the exact
invocation for it.

**The door's own call is witnessed by the build and by reading, not by a test.** `MissionOpener`
needs a lawful install to reach, and tests here never read one, so what is pinned by test is the
decision (`startViewCell`) and the mechanism (`SetStartView` + `Layout`), while the one statement
joining them is witnessed by compilation and by the headless drive going through the same
`StartMissionFrom`. This is a real gap and it is named rather than papered over.

**The authored number's derivation is an upper bound, not a measurement.** 640/32 = 20 uses the
original's frame, which is established; how much of that frame its map area occupied is not, because
the battle screen's panel is undecoded. The claim made is the bound, and `spec.md`'s Divergence
section says so.
