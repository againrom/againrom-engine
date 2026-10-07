# Verification — selecting a unit and ordering it from the running game

Task commits, oldest first: `45f97c1` (T1), `d588cb5` (T2), `d4ba9d5` (T3), `b363b66` (T4),
`b64ee81` (T5), `7afd8b8` (T6). Base `9f8c166`. Twenty-eight further commits sit between them and
none is this story's — three trailered elsewhere (`0001-res-archive/T7`, `0003-alm-container/T9`,
`0025-mapedit-model/T8`), the rest other stories' documents and the SDD audit's own rewrite.
Submodule pin frozen at research `03a9448` throughout; nothing in this chain reads it.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Every automated criterion is a unit test
over synthetic worlds, snapshots and cameras built in test code — **no test reads a game install and
none opens a window.** Every figure below was re-run in this seat unless the commit that measured it
is named.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output; 215 files)
$ go test -trimpath -count=1 ./...           (0 FAIL, 0 SKIP; 691 tests, 2290 counting
                                              subtests, 25 packages ok, 3 without tests)
      this story's own: 33 tests, 95 counting subtests
      pkg/ui 107/346   pkg/game 42/165   pkg/render/terrain 142/291
      pkg/render/camera 17/65   cmd/mapview 21/51   pkg/sim 69/111 (unedited)
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0028-app-unit-command
  analysis.md 6376 / 7168      provenance.md 7140 / 7168
  spec.md 13272 / 13312        plan.md 13201 / 13312
  tasks.md T1..T6 all under 1400 (827 1240 1091 1221 1164 830); legend+traceability 971 / 1200
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok
  verification.md (prose) 5335 / 9216 — fenced blocks free                      (exit 0)
$ sh scripts/check-sdd-audit.sh              at 7afd8b8, before this file
check-sdd-audit: 98 trailered commit(s) in ac6bd87..HEAD checked
FAIL 0028-app-unit-command: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED                      exit 1 in 30.4s; 46 lines, ONE enforced FAIL,
                                             43 unenforced note/warn lines on 0000-0013,
                                             0019/0023/0025's builds/ dirs and 0029 in flight
$ sh scripts/check-sdd-audit.sh              with this file
check-sdd-audit: 98 trailered commit(s) in ac6bd87..HEAD checked
check-sdd-audit: ok (44 note(s)/warning(s), none enforced)
                                             exit 0 in 32.2s; the FAIL row is now this
                                             story's builds/ warning, untracked and
                                             warning-only. 46 lines, no FAIL.
$ sh scripts/check-sdd-audit.sh --story 0028-app-unit-command
check-sdd-audit: 6 trailered commit(s) for 0028-app-unit-command in ac6bd87..HEAD checked
                 (scoped - the cross-story checks need a full run)
check-sdd-audit: ok (1 note(s)/warning(s), none enforced)    exit 0 in 1.5s
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 9f8c166..HEAD | sed '/^$/d' | sort | uniq -c
      1 0028-app-unit-command/T1  ...  1 0028-app-unit-command/T6   (6 ids, each exactly once,
      beside the three other stories' named above)
$ git log --format='%B' 9f8c166..HEAD | grep -ci co-authored-by
0
```

That one FAIL is what a story with every task landed and no evidence file looks like. This file
clears it and carries no trailer.

## Witnesses

Every id and the named thing that answers for it.

```
AC-1  SC-1   pkg/render/camera/camera_test.go — TestScreenToCellNamesTheCellUnderTheCursor
      (several positions and zooms), TestScreenToCellRoundTripsBothWays (both directions),
      TestScreenToCellReportsOutsideNeverClamped (ten cases, each reading the bool alone,
      so no index the contract refuses to promise is pinned), TestScreenToCellIgnoresThe-
      WorldHeight (four world heights spanning the flat one by the whole altitude spread).
AC-2  SC-2   pkg/ui/command_test.go TestTheSlopSeparatesATapFromADrag — the three named
      sequences driven through the shipped dragIntent, not a second delta of the test's.
AC-3  SC-3   TestTapSelectsClearsAndReplaces; TestTapHitsTheLowerIdOnASharedCell, the tie
      asserted over ids given in both slice orders.
AC-4  SC-3   TestRightPressOrdersTheSelectedUnitAtTheResolvedCell.
AC-10 SC-3   TestOutsideTheExtentClearsAndOrdersNothing — no order, and the tap clears.
P-5   SC-3   TestEveryEdgeCellAndSelectionLandsInExactlyOneOutcome — twelve frames, each
      matching exactly one outcome label. Sampled, not proved.
AC-9  SC-4   pkg/ui/selection_overlay_test.go TestSelectionHighlightIsThatUnitsFootprint,
      TestSelectionHighlightAbsentOrNoneDrawsNothing, TestSelectionPassIsAppendedLastAnd-
      MovesNothing (five shipped passes equal colour by colour, a sixth appended),
      TestSelectionHighlightFollowsTheTapThatMadeIt; pkg/render/terrain/selection_marker_
      test.go TestSelectionMarkerRectsNativeGeometry, TestSelectionRimIsHollowAndStays-
      InsideItsOwnCell, TestSelectionRimCoversNoOtherGlyph (empty pixel intersection at
      five scales), TestSelectionMarkerRectsRejectsOffMapAndBadScale, TestSelectionMarker-
      ColorDiffersFromEveryOtherGlyph.
AC-5  P-1  SC-5   pkg/game/world_test.go TestAnOrderedUnitWalksToItsCellAndTheDigestFollows-
      AHeadlessRun — 13 advances over an EMPTY script, the digest compared at every tick
      against a headless run of the one command at the tick it drained, the three unnamed
      entities unmoved on every row; P-1 sampled inside it by enqueueing with no advance.
      With pkg/ui/app_test.go TestTheOrderLeavesTheMapArmThroughTheSeam, which drives the
      gesture — two packages by necessity, below.
AC-8  P-6  SC-6   TestAnAdvanceReportsHowManyOrdersItApplied — 2, then 0, then 1, and a
      three-tick advance reporting 1 rather than a running total; the unit heads for the
      SECOND cell. pkg/ui/flow_test.go TestFlowHoldsTheOrderSeamWithTheTick pins the seam
      set and dropped on the same statement the tick is.
AC-6  SC-7   TestACommandedUnitTakesNoFurtherScriptedTarget — a 13-row hand-walked cell
      table, the commanded unit driven past three further scripted turns and holding no
      target at the end, a second unit reaching its own throughout.
AC-7  P-3  P-4  SC-8   TestAnUnorderedRunFollowsTheScriptAndAnOrderedRunRepeats — nothing
      enqueued, the driven digest equals a headless run over the script alone at every
      tick; one order script run twice from equal worlds agrees at every tick, guarded
      against ending on the unordered run's digest. TestTheAssembledStreamIsTheScripts-
      OwnEntryUntilAnOrderJoinsIt is P-3 by IDENTITY — data pointer, length and capacity,
      not DeepEqual.
AC-11 P-2  SC-10   pkg/game/order_invariance_test.go TestASessionsPickAndQueueReachNoWorld-
      BeyondTheOrder — four legs (session, bare, headless, quiet), byte form first and
      digest second at every tick; TestAPendingOrderAndAPickAreNotWorldState — three orders
      with no advance, tick/bounds/entity field set/byte form/digest unmoved, the queue and
      the commanded set read afterwards. pkg/game/frontend_test.go TestTheHeadlessCheckLine-
      IsWhatItWas; cmd/mapview/main_test.go TestTheStandaloneViewerHasNothingToSelect (zero
      entities over a map carrying three placed units, before and after run()'s own
      configuration) and TestTheStandaloneCheckLineIsWhatItWas; pkg/ui/command_test.go
      TestNoExportedMethodCarriesASelectionOrAnOrder. pkg/sim's suite and both wall checks
      pass UNEDITED: `git diff 9f8c166..HEAD -- pkg/sim internal/archtest` is empty.
DD-1  pkg/game/world_test.go TestEntityIdsCrossTheSeamAsTheirOwnIds — ids 9, 4 and 258,
      sparse and out of order, read back through entityDraws.
SC-9  the five mutants, all re-run in this seat — below.
AC-12 SC-11  NOT RUN. The manual pass needs a window and a lawful install; it belongs to
      the build stage, which this story has not reached (no builds/0028-app-unit-command).

The two pinned lines, whole:
  "againrom: 3 map rows, 2 of 8 buttons have a mask region"
  "mapview: Testmap 4x3 cells (12), tile slots 2/128, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)"
```

## The mutants and the hollowness probes

All ten re-run here, each applied to a production file, run against the **whole** tree and reverted;
every file md5-identical either side, at the value HEAD carries.

```
SC-9's five
M1  camera.go 98abac152cbb78a88a718efd7bfaf04c — the outside test deleted
    KILLED, 5 tests / 3 packages. camera's TestScreenToCellReportsOutsideNeverClamped in all
    ten cases ("ScreenToCell(239.5,300) reported inside; want outside"); pkg/ui's
    TestOutsideTheExtentClearsAndOrdersNothing in both ("order {entity:12 x:-9 y:-6} issued
    outside the extent, want none") and TestEveryEdgeCellAndSelectionLandsInExactlyOneOutcome
    at right/outside/selected; pkg/game's TestASessionsPickAndQueueReachNoWorldBeyondTheOrder
    and TestAPendingOrderAndAPickAreNotWorldState ("a cursor 480 world pixels past the extent
    came back inside, on cell (15,15)").
M2  command.go 16db8781fffe6c3b3c9dec0cd454ca49 — the slop widened to <=
    KILLED, one test tree-wide: TestTheSlopSeparatesATapFromADrag/exactly_four_pixels_is_
    not_a_tap. Exactly as T2 reported.
M3  world.go 55a72ad8462ae1aeeb2317e8f783ea67 — the queue left untruncated
    KILLED, 3 tests, all in pkg/game and all on a COUNT; no digest assertion fired.
      TestAnOrderedUnitWalksToItsCellAndTheDigestFollowsAHeadlessRun world_test.go:1529
        the queue holds 1 orders after the run, want 0 — one order, one advance
      TestAnAdvanceReportsHowManyOrdersItApplied world_test.go:1600
        the advance after the drain ran 1 ticks and reported 2 orders applied, want 1 and 0
      TestASessionsPickAndQueueReachNoWorldBeyondTheOrder order_invariance_test.go:274
        the session's queue holds 2 orders after the run, want 0
M4  world.go (same md5) — the exclusion write removed
    KILLED, 3 tests, all in pkg/game.
      TestACommandedUnitTakesNoFurtherScriptedTarget world_test.go:1730
        after the tick indexed 4: cell 0 = (4,5), want (6,7)
      TestAnUnorderedRunFollowsTheScriptAndAnOrderedRunRepeats world_test.go:1894
        the ordered runs end on the digest the unordered one does (0xbbfa3a9446ff65ab), so
        their agreeing at every tick says nothing about what an order did
      TestAPendingOrderAndAPickAreNotWorldState order_invariance_test.go:327
        the commanded set does not hold both ordered entities, so the front-end state under
        test never moved
M5  world.go (same md5) — the splice reversed, joined built pending-first
    KILLED by the slice assertion alone, one test, no digest anywhere, exactly as predicted:
      TestTheAssembledStreamIsTheScriptsOwnEntryUntilAnOrderJoinsIt world_test.go:1796
        the assembled stream is [{2 9 1} {0 11 3} {1 5 3}], want [{0 11 3} {1 5 3} {2 9 1}]
        — position 0 differs

The hollowness probes, owned by the entries that own no mutant
T6  world.go — the splice disabled in commands(), so pending reaches no world
    5 deaths, all in pkg/game: TestASessionsPickAndQueueReachNoWorldBeyondTheOrder (its
    HEADLESS leg, order_invariance_test.go:235), TestAnOrderedUnitWalksToItsCellAndThe-
    DigestFollowsAHeadlessRun, TestAnAdvanceReportsHowManyOrdersItApplied, TestACommanded-
    UnitTakesNoFurtherScriptedTarget, TestTheAssembledStreamIsTheScriptsOwnEntryUntilAn-
    OrderJoinsIt. Six were predicted; five is the measurement, both in T6's seat and mine.
T2  world.go — MapEntity.ID filled from the loop index
    TestEntityIdsCrossTheSeamAsTheirOwnIds alone.
T4  app.go 422ea1c087599d5a654f57df7f998f16 — command judged BEFORE v.step
    TestTheOrderLeavesTheMapArmThroughTheSeam/the_release_is_judged_after_the_camera_step
    alone.
T3  ui/overlay.go cb2549fa893e5dae0cf9fb6b5f187d7a, terrain/overlay.go
    d38af82b96b4d6a79540e42a5f456502
    the pass draws nothing (selectionScreenRects nil after resolving): 4 ui tests.
    the rim one row down: 2 terrain + 3 ui tests.
    filled rather than hollow (the thickness raised to the whole cell, so the four strips
    collapse to the footprint): 3 terrain + 4 ui tests, TestSelectionRimCoversNoOtherGlyph
    among them — the one probe that separates hollow from marked.
    All three lists are T3's, test for test.
```

## Disclosures

- **Three kill widths came out other than predicted, and `plan.md` is not amended.** T4's untruncated
  queue was predicted killed by SC-6's count alone and killed one assertion wider — a trailing
  queue-length check in SC-5's test. T5's exclusion-write mutant falsified **both** halves of "killed
  by SC-7 alone, and neither moves a digest": it killed wider, and the extra failure **is** a
  digest-valued assertion. T6's hollowness check predicted six deaths and measured five. The plan's
  substance holds: what it meant is that **no world-state digest witnesses the exclusion**, and
  indeed P-4's tick-by-tick comparison and the FR-5 headless comparison both did **not** fire — what
  fired is the non-vacuity guard, a digest-valued assertion about the **test's own power** and not a
  witness of the exclusion in world state. Every executor reported rather than repaired, and no
  assertion was changed to make a count match. Two further widths are mine, not theirs: at HEAD the
  tree kills wider than the entries could measure, because M1 and M3 now meet cases that did not
  exist when T1 and T4 ran them.
- **AC-7's front-end half is structurally unwitnessable at this tier, and is not owed to a later
  task.** A click script would need `App.step`, `Viewer.command` and `decide`, all unexported in
  `pkg/ui`, while a digest exists only one tier up in `pkg/game` — which cannot drive a gesture,
  against a `pkg/ui` that has no world to hash. That seam is DD-4's deliberate design. What is
  covered is the identical-stream-twice property at the `pkg/game` tier and the session-level
  invariance of the four-leg test.
- **AC-5 is discharged across two packages by necessity.** `Viewer.command` is unexported and
  `pkg/ui` may not import `pkg/sim`, so no single package can both drive a gesture and compute a
  digest. A fact, not a defect.
- **FR-4's "after the scripted commands" has no digest witness at all.** The exclusion removes the
  one scripted command an order could lose to under last-write, so no world state distinguishes the
  two splice orders; the assembled slice is the only witness the direction has, which is why M5 dies
  on an assertion and not a hash. A limit of the evidence, disclosed.
- **T5 made the queue truncation unconditional.** T4 described a truncation guarded by `applied > 0`;
  when assembly moved into `commands()`, `mw.pending = mw.pending[:0]` became unconditional, since
  emptying an empty queue is already nothing and a guard only adds a path on which the queue survives
  a step. Behaviour-neutral, and named here so the two commit bodies do not silently disagree.
- **The two `-check` pins are characterization pins**, quoted whole above. A future story adding a
  token to either summary must update them deliberately — intended, and a new maintenance edge this
  story introduces.
- **`entityCells(...)` in `pkg/game/order_invariance_test.go` is indexed by entity id.** That holds
  because `mapload.FromALM` mints ids `0..n-1` for that fixture, and existing tests already rely on
  the same identity. No guard was added.
- **AC-12 and SC-11 are not run.** They need a window, a lawful install and a map with relief. No
  automated criterion depends on them, and nothing here claims the manual pass.

## Contract findings

Nothing in `spec.md` (13272 B) or `plan.md` (13201 B) was found false at the implemented tree, and
neither is edited here. SC-9's predicted kill widths are the one place plan and measurement differ,
and the difference is width, never direction: every mutant SC-9 names died, none survived, and the
two claims that failed verbatim — "by the count alone" and "neither moves a digest" — failed by
naming fewer assertions than the suite actually holds, which is a strengthening recorded rather than
a defect repaired. The conclusion this file supports is that AC-1 … AC-11 and P-1 … P-6 hold on the
evidence above, P-1 … P-5 sampled rather than proved, P-6 witnessed at the advance boundary and
argued within one advance; AC-12 is outstanding.
