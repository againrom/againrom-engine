# Verification — the walk odometer

Windows 11, Go 1.26.1 (`go.mod`'s pin), branch `impl/0062-walk-odometer`, last code commit
`e28a2c7`, research submodule at `a93d19a8` with no leading status character. `builds/` does not exist in this worktree
— it is untracked rather than ignored — so `check-sdd-audit`'s note and warning **count** is not
comparable with the orchestrator's and only its FAIL set is quoted below.

## The gate

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go')
(no output from any of the three)

$ go test -count=1 -trimpath ./...
ok  againrom/cmd/terraintool 1.566s      ok  againrom/pkg/mapedit          0.638s
ok  againrom/cmd/texttool    0.861s      ok  againrom/pkg/mapload          0.590s
ok  againrom/internal/archtest 0.481s    ?   againrom/pkg/render  [no test files]
ok  againrom/internal/notices  0.348s    ok  againrom/pkg/render/camera    0.497s
ok  againrom/internal/synth    0.403s    ok  againrom/pkg/render/frame     0.580s
ok  againrom/pkg/data          0.419s    ok  againrom/pkg/render/menu      0.505s
ok  againrom/pkg/formats/alm   0.523s    ok  againrom/pkg/render/terrain   0.684s
ok  againrom/pkg/formats/databin 0.362s  ok  againrom/pkg/render/text      0.372s
ok  againrom/pkg/formats/pal   0.403s    ok  againrom/pkg/sim              4.015s
ok  againrom/pkg/formats/reg   0.471s    ok  againrom/pkg/ui               1.976s
ok  againrom/pkg/formats/res   0.469s    ok  againrom/pkg/vfs              0.498s
ok  againrom/pkg/formats/spr16 0.483s
ok  againrom/pkg/formats/spr256 0.348s
ok  againrom/pkg/game          1.632s

$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)          EXIT=0

$ sh scripts/check-doc-budget.sh docs/0062-walk-odometer
provenance.md   7681 / 16384  ok (46%)     tasks.md T1  1003 / 1400  ok (71%)
spec.md         9817 / 13312  ok (73%)     tasks.md T2   973 / 1400  ok (69%)
plan.md        11775 / 13312  ok (88%)     tasks.md T3  1246 / 1400  ok (89%)
tasks.md (legend+traceability)  380 / 1200  ok (31%)
plan <= 1.2 x spec   11775 <= 11780  ok
tasks <= 1.2 x plan   3602 <= 14130  ok

$ sh scripts/check-sdd-audit.sh | grep '^FAIL'
(no output — the FAIL set is empty, as it was on master before this branch)
check-sdd-audit: 254 trailered commit(s) in ac6bd87..HEAD checked
```

```
$ git diff --diff-filter=D --name-only master..HEAD
(no output — the deletion set is empty)

$ git diff --name-only master..HEAD -- pkg/sim
(no output — no file under the determinism wall was touched)

$ git diff --stat master..HEAD -- pkg cmd
 cmd/terraintool/main.go             |   2 +-
 pkg/game/animaudit.go               |  11 +-
 pkg/game/odometer_test.go           | 400 ++++++++++++++++++++++++++++++++++++
 pkg/game/world.go                   | 120 ++++++++++-
 pkg/render/terrain/unitanim.go      |  26 ++-
 pkg/render/terrain/unitanim_test.go | 141 ++++++++-----
 pkg/render/terrain/walkodo.go       | 167 +++++++++++++++
 pkg/render/terrain/walkodo_test.go  | 178 ++++++++++++++++
 8 files changed, 979 insertions(+), 66 deletions(-)
(docs/0062-walk-odometer: 4 files, 414 insertions)
```

## Acceptance criteria

| ID | Witness | Result |
|---|---|---|
| AC-1 | `pkg/game` `TestTwoSpeedsAdvanceTheSameSixteenSteps` — units at speed 8 and 32 cross one cell in 32 ticks and 8; both reach exactly 256 units and 16 timeline steps, the slow one is drawn on steps 0..16 (seventeen values) and the fast one on eight, skipping every odd step | pass |
| AC-2 | `pkg/render/terrain` `TestWalkAdvanceStraightCellIsSixteenSteps` — every span in `1..512` and all four straight deltas: total 256, phase 16, no exception | pass |
| AC-3 | `TestWalkAdvanceDiagonalCosts` — every span in `1..512` lands in `[256, 362]`, and the degenerate case is required rather than tolerated. Measured: span 1 → 362, 8 → 360, 16 → 352, 52 → 356, 64 → 320, 86 → 340, 128 and beyond → 256 | pass |
| AC-4 | `TestTheWalkFrameFollowsTheOdometerAcrossCells` — a cycle-14 class draws frames 1, 2, 3 at the three boundaries, and the property "no two consecutive boundaries alike" is asserted beside the literals; `TestTheOdometerIsTheRunningTotalOfTheCrossings` checks the count at **every** tick of three crossings, not only at the boundaries | pass |
| AC-5 | The same test's cycle-16 arm: frame 0 at every boundary | pass |
| AC-6 | `TestAStopResetsOnlyAClassWithNoIdleCycle` — after one cell both stand at 256; one tick later the class with no idle cycle holds 0 and the class with one holds 256 | pass |
| AC-7 | `TestWalkAdvanceIsTotal` (zero delta, non-positive span, ticks outside the crossing, a delta of 2^40), `TestSelectUnitFrameGuard` (negative count, zero-length track, non-positive frame count), `TestSelectUnitFrameEqualInputsEqualAnswers` | pass |
| AC-8 | `pkg/game` `drawn_invariance_test.go` and `cadence_invariance_test.go`, unchanged by this story and green: each drives this seam against a **headless** `sim.Step` over the same command stream and compares digests at every tick index. `TestBuildingASnapshotTwiceSelectsTheSameFrames` covers the second half | pass |
| AC-9 | The idle, standing, death and object selections' own tests, unchanged where they drove those arms and green — `unitanim_test.go`'s idle/standing/gate-failure cases, `deathanim_test.go`, `objectanim_test.go`, and `cadence_invariance_test.go` for the "at every cadence setting" clause | pass |

## Properties

| ID | Witness | Result |
|---|---|---|
| P-1 | **By construction, and stated rather than measured.** `advanceOdometers` reads no clock, no period and no wall time: its inputs are the entity's cell, the step memory, `Transit` and the counted crossing tick. A test at two cadences would drive `tick()` the same number of times and could only re-assert that. What a measurement *would* add is that no future edit introduces a clock read there; that is not claimed today | pass, by construction |
| P-2 | `TestTheOdometerIsTheRunningTotalOfTheCrossings` — the count is asserted at every tick of three crossings and the sequence it is asserted against is strictly increasing; `TestWalkAdvanceDiagonalCosts` bounds one tick's contribution at 362, the exact cell diagonal. The reset to zero is the one decrease and is FR-5's | pass |
| P-3 | `git diff --name-only master..HEAD -- pkg/sim` is empty (above); `internal/archtest`'s import graph and determinism source scan green; `drawn_invariance_test.go`'s headless leg green | pass |
| P-4 | `TestBuildingASnapshotTwiceSelectsTheSameFrames` — over 53 ticks, two builds of one tick's picture select the same frame pointers and mirror bits, and the odometer map is byte-identical before and after both builds | pass |

## Success criteria

| ID | Witness | Result |
|---|---|---|
| SC-1 | `TestWalkAdvanceIsTheEngineRecurrence` — the decoded `remaining / ticksRemaining` recurrence is re-executed in the test itself for every span in `1..512` against six delta shapes, straight, diagonal, negative and multi-cell, and compared share for share | pass, 0 disagreements |
| SC-2 | `TestWalkAdvanceStraightCellIsSixteenSteps` (AC-2's row) | pass |
| SC-3 | `TestWalkAdvanceDiagonalCosts` (AC-3's row) | pass |
| SC-4 | `TestTheOdometerIsTheRunningTotalOfTheCrossings` and `TestTheWalkFrameFollowsTheOdometerAcrossCells` (AC-4's row) | pass |
| SC-5 | `TestAStopResetsOnlyAClassWithNoIdleCycle` (AC-6's row) | pass |
| SC-6 | `TestSelectUnitFrameGuard`, `TestSelectUnitFrameEqualInputsEqualAnswers`, `TestWalkPhaseRoundsDown`, `TestWalkAdvanceIsTotal` | pass |
| SC-7 | AC-8's row | pass |
| SC-8 | `pkg/game` `animaudit_test.go`, unchanged and green: the sweep's in-range/guarded split still partitions a domain of the same size, and its figures are bit-identical because sixteen times the index puts the moving arm on the step the tick used to | pass |
| SC-9 | `TestTheSubCellGridIsOneNumberAcrossTheWall` — a mover at the rate floor is given a transit of 256 ticks by `pkg/sim`, and one straight cell costs the odometer 256 units | pass, 256 = 256 |

## The green-but-hollow audit

Two mutations run against the landed code and reverted:

```
span from the stale TransitTotal instead of tick+1+Transit
  -> FAIL TestAnUnratedMoverPaysAWholeCellEveryTick: walked 16 in one tick, want 256

reset unconditionally instead of forking on the idle gate
  -> FAIL TestAStopResetsOnlyAClassWithNoIdleCycle: a class with an idle cycle kept 0, want 256
```

Both killed. `TestWalkAdvanceIsTheEngineRecurrence` cannot be hollow in the same way: its oracle is
the recurrence written out as a loop over a remaining delta, which is not the arithmetic under test.

## Open, and what closes each

- **The engine's two clocks are one field; ours are two.** In the engine the walk odometer and the
  idle counter are the same word, so a class with an idle cycle resumes its walk from wherever
  idling left the count, and a walking unit's idle timeline is displaced by the ground it covered.
  Ours keeps the idle cycle on the free-running scene clock. **Closed by** a story that moves the
  idle clock off the scene clock onto a per-entity count and then merges the two — which is 0024's
  DD-5 contract and has to start there, not here.
- **The drawn position and the odometer disagree inside a long crossing.** Measured over spans
  `1..256`: **43 sub-cell units, two timeline steps, at span 173**, and **0 to 2 units — under one
  timeline step — at every crossing length a shipped speed produces** (spans 5, 13, 14, 16, 32
  measured at 0, 2, 2, 0, 0). The drawn position interpolates linearly; the odometer uses the
  engine's truncating split. In the engine there is no gap, because the same share moves both.
  **Closed by** a story that moves the drawn position onto the same split — 0047 FR-3 and 0056
  FR-6's contract, not this one's.
- **A diagonal cell costs a customised slow unit exactly what a straight one costs it.** From
  span 128 on, every per-axis share is one or two units and the root truncates the surplus away, so
  the odometer cannot tell a diagonal step from a straight one. Shipped speeds never reach there —
  the slowest shipped diagonal crossing is 52 ticks and pays 356 — but a customised speed of 4 or
  below does, and this is a **customisation limit**, the class of fact G2 says every decode owes.
  **Closed by** carrying a sub-unit remainder across ticks instead of discarding it, which would
  make the walk diverge from the engine and therefore needs an owner's ruling first, not a fix.
- **Our diagonal totals are not asserted against research's own 20/21/22.** That figure is a
  function of how many ticks a crossing takes, and our tick count is our own rate law's. We
  reproduce the derivation and assert its bounds. **Closed by** a per-pair comparison once this
  tree carries a cost plane and can produce the same crossings research measured.
- **The separate-context test gate was met for the spec and the plan but not for the tests.** The
  peer-prediction reader (a fresh zero-context subagent, `spec.md` alone) and the adversarial plan
  reader (separate context, no access to the authoring conversation) both ran and both found
  defects that are fixed above. The tests themselves were written by the implementing lane. What
  stands in for the missing independence is that SC-1's oracle is the decoded recurrence rather
  than the implementation, and the two mutation kills. **Closed by** a test-authoring subagent on
  the next story in this area.

## What the reviews found, and where it went

The peer-prediction reader built the right thing from `spec.md` alone but had to invent the
per-tick split, and it found **AC-1 and AC-3 both false as written** — a five-tick crossing shows
five frames rather than "the whole of the same sixteen steps", and a diagonal is not *strictly*
dearer than a straight cell. FR-3 now states the split itself and both criteria are corrected.

The adversarial plan reader found four more. The one that mattered was **DD-3's tick index**: taken
from `TransitTotal`, it is wrong for an entity that stops being rated, because that total outlives
the transit it measured — it would have paid a whole cell the last share of a sixteen-tick
crossing. The index is now counted where `recordCells` already tests `Transit > 0`, and
`TransitTotal` is not read at all. It also measured R-1's gap at 44 rather than the "under one
sub-cell unit" the plan claimed, named the second class lookup DD-9 denied, and named the
sub-cell 256 now standing on both sides of the determinism wall — which is what SC-9 exists for.
