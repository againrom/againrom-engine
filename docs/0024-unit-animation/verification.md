# Verification — unit animation (idle and move, with facing)

Task commits, oldest first: `5d50161` (T1), `4d43542` (T2), `42e8063` (T3), `66d7c09` (T4),
`2c7842e` (T5), `29427ec` (T6), `c2c19b9` (T7) — and, in the 2026-07-29 revision, `50bd5ec` (T8)
and `117ce77` (T9) over the untrailered contract correction `b355ba5`. Base `e433d0e` — the plan
and its tasks; before it the Stage-1 commit `9c4624a`. `e433d0e..HEAD` touches neither `pkg/sim`
nor `cmd/mapview` in any byte, and the revision touches neither `pkg/ui` nor `pkg/render`: both
`git diff --stat` runs come back empty, so FR-7's freezes and DD-8's untouched window tier are
measured, not inferred. Submodule pin frozen at research `3d95f2a` throughout.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Evidence at `117ce77`. Two kinds of
number appear below: the synthetic suite, which reads no install, and a **developer audit over a
lawful install**, marked *(probe)*. No probe result substitutes for a criterion the suite
carries.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 23 packages ok, 4 without tests)
      pkg/data             36 tests,  84 counting subtests
      pkg/render/terrain  137 tests, 286
      pkg/game             30 tests, 148
      pkg/ui               94 tests, 297   (unedited by the revision, and green)
      cmd/terraintool      24 tests,  94
      cmd/againrom          5 tests,  36
      internal/synth        7 tests,  34
      pkg/sim              55 tests,  87   (identical to 0022's count: unedited)
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0024-unit-animation
check-doc-budget: 0024 runs under a DECLARED OVERRUN (spec 15360, plan 16384,
  verification prose 10240) - a contract correction to a landed story
  analysis.md  5383 / 7168   provenance.md  7151 / 7168
  spec.md     14608 / 15360  plan.md       15989 / 16384
  tasks.md T1..T9 all under 1400; legend+traceability 697 / 1200
  verification.md (prose) 10164 / 10240 - fenced blocks free
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok                     (exit 0)
$ sh scripts/check-sdd-audit.sh
check-sdd-audit: 68 trailered commit(s) in ac6bd87..HEAD checked
FAIL 0024-unit-animation: nothing in verification.md witnesses: AC12
FAIL 0024-unit-animation: nothing in verification.md witnesses: SC10
$ git log --format='%(trailers:key=SDD-Task,valueonly)' e433d0e..HEAD | sed '/^$/d' | sort | uniq -c
      1 0024-unit-animation/T1  ...  1 0024-unit-animation/T9   (9 ids, each exactly once)
$ git log e433d0e..HEAD --format='%B' | grep -ci co-authored-by
0
```

Before this commit `check-sdd-audit` was **FAILED** on those two lines — the revision's new ids,
declared in the contract and not yet witnessed. This file clears them, exactly as the first
evidence commit cleared the story's own missing-verification line; that is the stage's own
acceptance, and the commit carries no trailer.

## Witnesses

Every id and the named thing that answers for it. Everything but the probe and the manual AC-11
runs windowless, reads no game install and consults no clock. Expected values are hand-written
literals beside their fixtures throughout — bases, indices, tracks, rects, counts — never values
recomputed through the code under test.

```
SC-1  FR-1  AC-1                   pkg/data/defaults_test.go (T1 5d50161)
      TestAUnitKeyNoSectionSetsTakesItsDefault - a chain setting no scalar keys and a section
      setting nothing at all: the seventeen -1s, the eight decoded 0s, TileSize 1, InMapEditor
      0, per key; SpritePath() empty on a class whose chain never sets File.
      TestFileDoesNotInheritInObjectsButDoesInUnits - File inherited here, not in objects.
      DD-1's tripwires green unedited: TestAKeyNoSectionSetsTakesItsDefault (objects) and
      TestStructuresKeepTheZero (keep-the-zero's structures half).
      TestEveryDefaultRowNamesAnIntKeyOfItsOwnTable - the unit rows pinned as literals,
      InMapEditor row-less, no row noInherit. load.go untouched by the T1 diff (git show
      5d50161 --stat: keys.go, defaults_test.go, internal/synth/reg_test.go alone - the last
      a declared deviation, Notes below).

SC-2  FR-2  AC-2                   pkg/data/anim_test.go (T2 4d43542)
      TestAnimBasesStridesAndTotals - hand literals at both (S, D); the spec's example class
      pins bases 9/24/34/44 and total 54; bone and idle share TailBase.
      TestAnimTrackExpansion - the [0 1 2 1] ping-pong survives, a Time = 0 entry adds no
      tick, the empty pair expands empty. TestAnimPhasesScalarIsNeverALength - MovePhases
      above its pair's expanded length keeps the track's period. TestAnimNeverConsultsDying -
      a sentinel Dying changes nothing. TestAnimGates - MoveOK/IdleOK fail on empty tracks
      and non-positive scalars. No IO, no sheet, no float in anim.go.
      TestAnimClampsAnAbsentPhaseToNoBlock (T8 50bd5ec) - the clamp: bases 9/24/34/34 and
      total 34 for a Flip-1 class with DY = BN = -1, where the unclamped arithmetic gave 29
      and 29; that class deriving the descriptor its 0-resolving twin derives, field for
      field; slot 3 / wind-up 0 / bases 16/40/48/80 / total 120 for a Flip-0 class with
      MB = -1; and no gate opened by the clamp. Every figure a hand literal.

SC-3  FR-3  AC-3  P-1              pkg/render/terrain/unitanim_test.go (T3 42e8063)
      TestSelectUnitFrameMovingFlip0 / MovingFlip1 / IdleFlip0 / IdleFlip1 /
      TestSelectUnitFrameStanding - DD-3's boundary tables verbatim as literals at both
      layouts, the spec's worked example (16, mirror) among them; mirror exactly on octants
      5-7 (standing: g > 8) at Flip 1, never at Flip 0.
      TestSelectUnitFrameTickRun - steps advance, loop, de-sync by the id difference.
      TestSelectUnitFrameGuard - an over-reaching index, frameCount 0 and a negative tick
      all answer (0, false), nothing panics.
      TestSelectUnitFrameEqualInputsEqualAnswers - P-1 stated as repetition.

SC-4  FR-5  AC-8  AC-9             pkg/game/units_test.go (T4 66d7c09), cmd/againrom/main_test.go
      TestLoadUnits - "File inherits through Parent, and the shared sheet is ONE slice": one
      decode per distinct path, the converted slice shared by pointer identity; "a drawable
      class is keyed by ID, canvas across, the WHOLE sheet aboard"; "the descriptor rides
      the entry, copied value for value"; "each exclusion keeps a frameless entry, distinct
      from a missing id"; "an unreadable registry is the error, and the only one"; headless,
      no GPU. TestUnitBundle - AC-9's two install layouts: the complete one passes -check
      loading full sheets headlessly, the omitUnitRegistry one fails naming units/units.reg.

SC-5  FR-4  FR-6  AC-4  AC-5       pkg/game/world_test.go (T5 2c7842e)
      TestSeamDrawsFrameZeroUnmirroredWhenTheSheetFallsShort - a sheet shorter than the
      selection reaches draws frame 0 unmirrored; no drawable art and an id naming no class
      cross as nil - the square; nothing errors (AC-4 through the seam).
      TestIdleEntityKeepsItsLastOctantPerId - the walker keeps its octant at rest, the
      never-moved faces octant 0, the memories independent per id (AC-5).
      TestEntityDrawsHandArtExactlyWhereTheBundleHoldsAFrame - the SELECTED drawable by
      pointer identity where the bundle holds frames; the world's digest untouched.

SC-6  FR-6  AC-6                   pkg/render/terrain/units_test.go (T4), pkg/ui/mirror_draw_test.go (T6 29427ec)
      TestMirroredAndPlainPlacementsAreIdenticalRectangles - identical rects at the drawn
      frame's own size. TestUnitPlaceDisplacesVerticallyOnly - displaced minus flat exactly
      -(lift+originY) in Y, 0 in X. TestUnitPlaceCarriesTheMirrorBitAndReadsItNowhere.
      TestStaticScreenRectsCarryTheMirrorBitThroughTheCull.
      TestSpriteGeoMReflectsInsideItsOwnRectangle - GeoM-only reflection.
      TestMirroredDrawSharesTheTextureAndCacheKey - no second texture, no new cache.

SC-7  FR-4  FR-7  AC-7  P-4        pkg/game/world_test.go (T5), internal/archtest
      TestDrivenWorldReachesTheDigestOfAHeadlessRun - the driven bundle resolves a Flip-1
      mover and an idle-cycle class and the test asserts both hold sprites mid-run (DD-7's
      non-vacuity clause), so digest equality at every k - and against the untouched map's -
      is measured over a screen that ticked the scene clock, wrote facing memory and
      selected mirrored frames. TestSimSourcesAreDeterministic and TestLiveTreeClean green
      unedited; the task-span diff over pkg/sim is empty (header above).

SC-8  FR-6  FR-7  P-2  P-3         pkg/ui/mirror_draw_test.go (T6)
      TestEveryInMapEntityDrawsExactlyOneItem - a mixed snapshot: exactly one item each,
      sprite or square; off-map the only other case (P-2).
      TestNilBundlePassSliceIsTheArtlessDerivationDriven - structural equality at tick 0 AND
      after k driven ticks, no sprite pass, no texture map (P-3).

SC-9  FR-2  FR-3  AC-10            pkg/game/animaudit_test.go, cmd/terraintool/unitanim_test.go (T7 c2c19b9)
      Unit half: TestUnitAnimAudit - the spec's example class over a full 54-frame sheet and
      over a deliberately short 20-frame one (43 in range, 5 guarded, worked octant by
      octant in the comment), a Flip-0 class with both gates on fitting a 64-frame sheet
      exactly, a frameless class guarding all 16, a nil entry contributing no row; the frame
      slices all-nil, so a sweep touching a pixel would panic. TestUnitAnimLines - the
      printed rows and summary by whole-string equality. TestUnitAnimErrors - non-zero only
      on a load failure, nothing printed. Corpus half: the probe below.

SC-10 FR-8  AC-12                  pkg/game/world_test.go (T9 117ce77)
      TestPacedAdvanceRunsAtTheDecodedLogicRate - every elapsed run a literal and no clock
      read anywhere, paceTo taking its instant as an argument. The decoded 62 ms / 16 tps is
      asserted at the top, so the literals under it have a stated provenance. The first call
      takes the baseline and advances nothing; 30 / 60 / 62 ms shows the remainder carried
      ACROSS calls; ten calls 100 ms apart run 16 ticks - 1000/62, the decoded rate, not the
      ten a per-call advance gives - with the scene clock at 16 and not at 10; a 10 s stall
      runs exactly maxCatchUpTicks = 4 and the call after it runs 1, not the 157 a queued
      debt would hand back; 124 calls of 500 us run one tick, so nothing is lost below a
      millisecond either; those same 16 paced ticks reach the digest, the scene and the cells
      of 16 DIRECTLY driven ones, against a never-stepped digest that discriminates; a
      backwards clock advances nothing and rewinds no baseline.
      tick() is unedited and pkg/sim untouched (header): wall-clock decides when a tick
      fires and nothing about what it does.

AC-11  (manual)                    the owner's run - OPEN, see below.
```

## The corpus audit over a lawful install *(probe)*

AC-10's corpus half — SC-9's second figure — run in this seat with T7's shipped tool built from
the evidence tree, against the GOG Galaxy install of Rage of Mages, **EN version, reinstalled
2026-07-29**, read and never modified; its `graphics.res` is 61,394,716 bytes, MD5
`B38A478BF66021028457155BEF86D61D`, both re-measured at this commit. The root came from
`-assets`; no install path is in source. Figures and counts recorded, no asset byte leaves the
install. **Re-run at `117ce77`, after the clamp**: the twelve mismatched rows the first run at
`c2c19b9` reported are gone, and the summary falls from 13 mismatched to 1.

```
$ terraintool unitanim -assets "<install root>"          verbatim, all 34 rows
class 1: predicted 129, frames 129, in-range 256, guarded 0
class 2: predicted 208, frames 129, in-range 256, guarded 0
class 3: predicted 152, frames 152, in-range 256, guarded 0
class 4: predicted 208, frames 208, in-range 256, guarded 0
class 5: predicted 152, frames 152, in-range 256, guarded 0
class 7: predicted 152, frames 152, in-range 256, guarded 0
class 8: predicted 152, frames 152, in-range 256, guarded 0
class 9: predicted 152, frames 152, in-range 256, guarded 0
class 10: predicted 152, frames 152, in-range 256, guarded 0
class 11: predicted 152, frames 152, in-range 256, guarded 0
class 12: predicted 136, frames 136, in-range 256, guarded 0
class 13: predicted 136, frames 136, in-range 256, guarded 0
class 14: predicted 168, frames 168, in-range 256, guarded 0
class 15: predicted 168, frames 168, in-range 256, guarded 0
class 19: predicted 224, frames 224, in-range 224, guarded 0
class 21: predicted 224, frames 224, in-range 224, guarded 0
class 23: predicted 144, frames 144, in-range 256, guarded 0
class 24: predicted 216, frames 216, in-range 256, guarded 0
class 26: predicted 160, frames 160, in-range 64, guarded 0
class 27: predicted 152, frames 152, in-range 64, guarded 0
class 64: predicted 248, frames 248, in-range 320, guarded 0
class 65: predicted 256, frames 256, in-range 256, guarded 0
class 66: predicted 200, frames 200, in-range 192, guarded 0
class 68: predicted 164, frames 164, in-range 256, guarded 0
class 69: predicted 99, frames 99, in-range 192, guarded 0
class 70: predicted 109, frames 109, in-range 384, guarded 0
class 71: predicted 164, frames 164, in-range 256, guarded 0
class 72: predicted 154, frames 154, in-range 256, guarded 0
class 73: predicted 89, frames 89, in-range 64, guarded 0
class 74: predicted 104, frames 104, in-range 160, guarded 0
class 75: predicted 114, frames 114, in-range 192, guarded 0
class 76: predicted 94, frames 94, in-range 224, guarded 0
class 79: predicted 224, frames 224, in-range 320, guarded 0
class 80: predicted 224, frames 224, in-range 256, guarded 0
unitanim: 34 classes, 1 mismatched, 8000 in-range, 0 guarded
```

Reading the figures:

- **34 rows** — every class of the registry, the count `REG-UNITS-018` documents, none skipped.
  A row's domain is `16 x max(1, longer track's period)`, which is why in-range varies 64–384;
  the 34 domains sum to the 8000, and in-range + guarded closes each row's ledger.
- **0 guarded of 8000.** The bounds guard never fires on this corpus: every selection the full
  domain produces — both states, 8 octants, every step — lands inside its class's own sheet.
  The digest-witness concern (a guard firing on classes a driven map exercises) is empty here.
- **Twelve of the first run's thirteen mismatches were OUR arithmetic, and the clamp closed
  them.** Each was short by exactly 8 = `D*1`: classes 3, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 23
  each resolve **`DyingPhases = -1`** — FR-1's absent-key sentinel — and the sheet contract's sum
  took that `-1` as a **count**, subtracting one whole direction block for a dying block the
  sheet does not contain (measured in the orchestrator seat with the shipped `classdump` over
  this same install). T8 clamps it, and the predicted total now reproduces each sheet's own frame
  count **exactly, 12 of 12**, across four distinct pairs: 144→152, 128→136, 160→168, 136→144.
  The frames were never surplus art; the formula was short.
- **The clamp changed no selection this story makes, and that is measured rather than argued.**
  In-range and guarded are **8000 and 0 in both runs**, row for row: the sentinel reached only
  `AttackBase`, `DyingBase`, `TailBase` and `Total`, and `TailBase` is read only under `IdleOK`
  (`ID > 0`), which all twelve fail. The defect was **latent** — it becomes load-bearing at
  0033/0040, which select from the dying block — and the correction is likewise inert on this
  corpus except in the one figure it was found in.
- **Class 2 is the one genuine sheet mismatch, and it has a named rival.** Resolved `Flip` 0,
  phases `(MB, MV, AT, DY, BN, ID) = (2, 8, 7, 4, 3, 0)` — no sentinel anywhere — predicting 208
  against a 129-frame sheet (`File = 21`). It guards nothing because every index this story
  computes for it stays below 95. But **129 is exactly what that same phase set predicts under
  the Flip-1 layout**, `9 + 5*24`, which class 1 confirms by matching it exactly. Two readings
  survive this evidence: the sheet is simply smaller than its class's scalars predict, or class
  2's true layout is Flip 1 and our resolved 0 is wrong — under which every class-2 selection
  addresses the wrong block and the unit misdraws. What discriminates: whether class 2 is placed
  on any shipped map (the `classdump -sweep` census reports per-map totals only, so this is
  **unmeasured**) and the owner's eye at AC-11. Recorded as open, not resolved.
- **T7's disclosed blind spot (Notes) caveats the 8000**: "in-range" means the sheet's count did
  not change the answer, so a selection whose raw index were negative would be counted in-range
  too. Nothing drawn depends on the count being what it is; that is all the figure claims.

## Notes

- **T1's declared deviation** (commit `5d50161`): DD-1 predicted "every other suite must pass
  unedited", and one did not — `internal/synth/reg_test.go`, a 0022 fixture, pinned `Width = 0`
  on a class whose chain never sets `Width`: the pre-FR-1 Go zero, exactly the behaviour T1
  replaces. The minimal true correction was made and declared in the commit message — a 4-line
  diff moving one expectation literal `0` to `-1` under a two-line comment; the fixture and
  everything else the test asserts are untouched.
- **A stale comment stands in `pkg/data/classes.go`** (the `UnitClass` doc, lines 30–33): it
  still says a key a class omits is "inherited from its Parent, or left zero — this registry's
  own per-key defaults are not decoded, unlike objects.reg's". Since T1 that tail is false:
  an absent-everywhere scalar resolves through `unitDefaults` in `keys.go` (-1, with the spec's
  named exceptions), the same mechanism as objects.reg's. The comment predates FR-1, sits
  outside T1's file list, and misleads only about which tier fills the value — the decoded
  values themselves are pinned by SC-1's tests. Left for a cleanup commit; recorded here so it
  is a known debt, not a discovery.
- **T7's disclosed audit blind spot** (documented in `pkg/game/animaudit.go`): guardedness is
  measured through the selector's interface — each tuple selected twice, once against the
  sheet's count, once against an unbounded one — so a selection whose raw index is negative
  answers frame 0 at EVERY count, is indistinguishable from a genuine frame-0 answer, and is
  counted in-range. Inherent to measuring through the one selection path rather than
  re-deriving indices from the bases (which would be the second copy of the sheet contract
  DD-3 bars). Its corpus consequence is the caveat above.

## The 0024 revision — the two corrections, landed 2026-07-29

Both edited the contract, so both were paid there first, in the untrailered `b355ba5`.

1. **The absent-phase sentinel** (T8 `50bd5ec`, FR-2/DD-2/SC-2). The sheet contract now states
   that every phase scalar enters the block arithmetic clamped at zero; `phaseCount` is the one
   expression that does it. Found by this story's own instrument on its first corpus run,
   confirmed by the re-run above: 13 mismatched → 1, with the in-range ledger unmoved.
2. **The scene clock's cadence** (T9 `117ce77`, FR-8/DD-8/SC-10). The advance now fires at the
   decoded logic rate through `terrain.Ticker`, bounded at 4 ticks per front-end call. See
   AC-11 below for what the owner reported and what remains open.

The budget, measured before deciding rather than written around. `spec.md` stood at 13310 of
13312 bytes and `plan.md` at 13305 of 13312 — 2 and 7 bytes of headroom — with this file's prose
at 9111 of 9216. A compaction pass ran first and harvested 212 B from the spec, all of it genuine
duplication (the bounds guard stated three times in one file; stored direction 0's unestablished
meaning twice); no further cut paid for the corrections without deleting contract. So
`check-doc-budget.sh` gained a declared `0024` row — spec 15360, plan 16384, verification prose
10240 — with the numbers and their asymmetry argued in the script: the plan's ceiling is still
4 KB below the pre-tightening 20480, while the spec's is 1 KB above the pre-tightening 14336 and
is the real admission. Final: spec 14608, plan 15989, tasks 5986.

## AC-11 — the owner's window check (STILL OPEN)

**The owner ran it 2026-07-29: the animation is there and it is too fast.** One root, diagnosed
then and corrected since. The logic-tick rate is decoded and already in this tree —
`terrain.TicksPerSecond`, default index 4 = **16 ticks/s** — and the water layer paces itself by
it. The scene clock was incremented once per front-end Update instead, at the display's 60/s, so
every track ran **3.75x** fast. The registry confirms the model while condemning the clock: a
foot soldier's move track is eight frames of two ticks, period 16 — exactly **one second** per
cycle at the game's rate, 0.27 s at ours. So `AnimTime` is in logic ticks, `expandTrack` is
right, and only the clock source was wrong. The pacing predated this story (per-Update since
0020); 0024 made it visible, because a square crossing cells has no cadence to be read against
and a walk cycle does.

T9 corrects it under FR-8, and SC-10 measures the correction — but only headlessly, over handed
timestamps. **AC-11 stays OPEN on both counts**: nothing has been ruled on facing, mirroring or
the idle cycles, and the cadence itself now needs a fresh look through a window, where 16/s is
either what the game moved at or is not.
