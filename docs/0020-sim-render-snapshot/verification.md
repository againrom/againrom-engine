# Verification — a running world under the map screen

Task commits, oldest first: `6f2146c` (T1), `83fa926` (T2), `6f0b274` (T3), `921e7ca` (T4),
`c9ff4a6` (T5). Base `1a81a38`; two document commits before T1. One untrailered commit sits between
T5 and this file — `4a90f6a`, a `gofmt` of `pkg/game/world_test.go`; see *Contract findings*.
Submodule pin frozen at research `e53c779`.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Evidence at `4a90f6a`. Unlike 0019 this
story ships something runnable, so two kinds of number appear below: the synthetic suite, which
reads no install, and a **developer probe over a lawful install**, marked as such wherever it is
used. No probe result is a substitute for a criterion the suite is supposed to carry.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output — red at c9ff4a6, see Contract findings)
$ go test -trimpath -count=1 ./...           (0 FAIL; 469 tests pass, 26 packages ok, 5 without tests)
      pkg/render/terrain  124 tests, 273 counting subtests
      pkg/ui               84 tests, 286
      pkg/mapload          14 tests,  14
      pkg/game             21 tests, 112
      internal/archtest    11 tests,  36
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0020-sim-render-snapshot
  analysis.md   6767 / 7168   provenance.md  6334 / 7168
  spec.md      12488 / 13312  plan.md       13211 / 13312
  tasks.md T1..T5 all under 1400; legend+traceability 527 / 1200
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok                     (exit 0)
$ sh scripts/check-sdd-audit.sh
check-sdd-audit: 45 trailered commit(s) in ac6bd87..HEAD checked
  [notes/warnings on 0000-0013 only; pre-existing and unenforced. Elided rather than pasted:
   their text names ids, and an id pasted into this file would be counted as a witness of ours.]
warn 0020-sim-render-snapshot: verification.md exists, builds/0020-sim-render-snapshot/README.md
   does not (builds/ is untracked - warning only)
check-sdd-audit: ok (42 note(s)/warning(s), none enforced)
      0020 draws ZERO id notes - the builds/ warn above is its only line.
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 1a81a38..HEAD | sed '/^$/d' | sort | uniq -c
      1 0020-sim-render-snapshot/T1  ...  1 0020-sim-render-snapshot/T5   (5 ids, each exactly once)
$ git log 1a81a38..HEAD --format='%B' | grep -ci co-authored-by
0
```

Before this commit `check-sdd-audit` was **FAILED** on one line — `0020-sim-render-snapshot: every
task in tasks.md has landed and there is no verification.md`. This file clears it; that is the
stage's own acceptance, and the commit carries no trailer. The `builds/` warning is the build
stage's, not this one's.

## Witnesses

Every id and the named thing that answers for it. Everything but the two rows marked *(probe)* and
*(manual)* runs windowless, reads no game install and consults no clock (FR-11).

```
FR-1  AC-1  AC-4  SC-2             pkg/game/world_test.go
      TestOpenMapWorldStandsAtTickZeroOverTheMapsOwnUnits
      TestOpeningTheSameMapAgainBuildsAFreshWorld
      TestOnlyTheGameFrontEndBuildsAWorld/the front-end's loader builds one per opening
      A 12x9 alm.Map literal with 4 hand-written unit anchors: tick 0, 4 entities, bounds 12x9.
      Open-tick-Esc-open calls the loader twice and the second world's cells equal a COPY of the
      first's taken before it was ticked. "One decode" is structural (loadMap passes mv.Map, the
      map its own call decoded) - see What no test sees.

FR-2  AC-2  SC-1                   pkg/ui/app_test.go, pkg/ui/flow_test.go
      TestMapTickAdvancesOncePerMapScreenTick - 8 subtests: opening advances nothing; exactly one
      advance per map-screen tick under neutral input, pan, drag and wheel; the camera and the
      water counter move on the tick the count rises; the leaving tick advances nothing and drops
      both; none on menu or picker; reopening advances the second and never the first; a failing
      load leaves neither; a nil tick still opens a working map screen.
      TestFlowHoldsTheMapTickWithTheViewer - 5 subtests: stored together, neither on failure,
      dropped together by escape, replaced on reopen, replaced when one is already held.

FR-4  SC-3  SC-5                   pkg/game/world_test.go, pkg/mapload/schedule_test.go
      TestScheduleEntryIsAppliedAtItsOwnTickAndNothingPastTheEnd - a hand-written 4-tick schedule
      driven 6 ticks, every entity's cell written out per tick as a literal table; the two rows
      past the end show one entity walking on to its last target under no order and the rest
      standing. TestEntityZeroWalksItsFirstLeg.
      TestScheduleIsTheLapWrittenOutByHand · TestScheduleNamesNoTargetOutsideTheMap
      TestScheduleNamesNoTargetOutsideAnyExtent · TestScheduleLegsAreReachableBeforeTheNextOrder
      TestScheduleStaggersSuccessiveEntities · TestScheduleNeverTurnsNearbyEntitiesTogether
      TestScheduleIsAFunctionOfTheMapAlone · TestScheduleReadsTheUnitsAndTheExtentAlone
      TestScheduleOverAMapWithNoUnits
      Two calls over one map are deeply equal; a unit against each edge; the last filled index
      carries a command.

FR-3  AC-3  AC-10  P-2  P-6        pkg/ui/app_test.go, pkg/ui/entity_overlay_test.go,
                                   pkg/game/world_test.go
      TestMapTickAdvancesOncePerMapScreenTick (the menu/picker/Esc subtests)
      TestOnlyTheGameFrontEndBuildsAWorld/the shared load path owns no world
      TestViewerGivenNoEntityCellsIsTheOneThatShipped - 4 subtests: never given cells yields no
      pass and the shipped slice; no diagnostics and no cells yields no passes at all; an empty or
      wholly off-map list yields none; a shrunken set leaves no stale rects.
      TestSetEntityCellsResyncsNothing.
      A viewer from LoadMapViewer reports EntityMarkers() == 0 and the loader hands back no tick.

FR-6  AC-6  SC-6                   pkg/render/terrain/entity_overlay_test.go
      TestEntityMarkerRectsNativeGeometry · TestEntityMarkerSquareStaysInsideItsOwnCell
      TestEntityMarkerRectsRejectsOffMapAndBadScale · TestMarkerFamilyRejectionIsOneSharedRule
      TestEntityMarkerColorDiffersFromEveryDiagnostic
      10 px on both axes at CellSize 32, centred on the cell centre; one rect per in-map cell,
      none off-map or at cellpx < 1; side <= cellpx at every scale checked, so the map clip cannot
      trim an in-map square; the colour equals none of the three diagnostics.

FR-5  FR-7  AC-7  P-5  SC-7        pkg/ui/entity_overlay_test.go
      TestEntityScreenRectsPlacesTheSquareThroughTheFamilyTransform (3 subtests)
      TestEntityScreenRectsCullOnThePlacedRect (3 subtests)
      TestEntityRectCarriesItsCellsLiftDisplacedAndNoneFlat
      TestEntityPassIsFirstAndTheDiagnosticsKeepTheirOrder
      One square per entity in list order; an off-map cell contributes nothing; the count before
      the cull equals the cells given; the cull reads the PLACED rect and the view as it is now.
      The entity pass is first and the three diagnostics keep their order behind it.

FR-8  AC-8  P-3  SC-8              pkg/game/world_test.go
      TestEntityStandsOnItsOwnUnitsMarkerCellAtTickZero
      Units at hand-written anchors, one with a low byte other than 0x80. Entity cells and unit
      marker cells each equal literals written beside the fixture, never a shift recomputed. In a
      viewer holding both, the unit cross's arm centre falls INSIDE the entity square, in flat and
      displaced mode and at two camera positions - containment, not equal centres (DD-6).

FR-10  AC-5  P-1  SC-4             pkg/game/world_test.go
      TestDrivenWorldReachesTheDigestOfAHeadlessRun - k = 10 and k = 575, the production driver
      against sim.Run over a fresh world of the same map. See What no test sees for which k
      discriminates.

FR-9  SC-9                         internal/archtest/dag_test.go, and this seat
      TestUITierAllowanceIsPinned - allow["pkg/ui"] pinned to exactly {pkg/render, pkg/render/},
      then the evaluator driven: 9 rejected edges named one at a time (pkg/sim first, then
      mapload, game, data, vfs and the four format packages) and 5 accepted render edges.
      TestLiveTreeClean · TestSimSourcesAreDeterministic still green.
$ git diff --stat 1a81a38..HEAD -- pkg/sim internal/archtest/dag.go     (no output: both untouched)
$ go list -f '{{.ImportPath}}{{range .Deps}}{{if eq . "againrom/pkg/sim"}} REACHES-SIM{{end}}{{end}}' \
      ./pkg/ui ./pkg/render/terrain ./pkg/game ./cmd/againrom ./cmd/mapview
againrom/pkg/ui                        againrom/pkg/render/terrain
againrom/pkg/game REACHES-SIM          againrom/cmd/againrom REACHES-SIM
againrom/cmd/mapview REACHES-SIM       <- through pkg/game's LoadMapViewer; see below
$ grep -n 'againrom/pkg/sim' pkg/ui/*.go                                (no output)

AC-9  P-4                          pkg/ui/flow_test.go, pkg/game/world_test.go
      TestFlowHoldsTheMapTickWithTheViewer/a failing load stores neither
      TestMapTickAdvancesOncePerMapScreenTick/a failing load leaves neither a viewer nor a tick
      TestOnlyTheGameFrontEndBuildsAWorld/a row that will not decode builds nothing
      The picker stays showing, the reason is recorded, the row is marked unusable, and the flow
      holds neither viewer nor tick - so nothing exists to advance.

FR-11                              the whole run above
      AGAINROM_ASSETS is unset in this seat and the suite is green; every fixture in the five
      packages is an alm.Map literal, a grid literal or a hand-written schedule.

AC-11  SC-10  (manual)             the owner's run - see The build
```

## The probe over a lawful install

`AC-11`/`SC-10` are the owner's to walk, and nothing below discharges them. What it does discharge
is 0019's open *"no real map has ever been through `FromALM`"*: the probe calls the **production**
`alm.Open`, `mapload.FromALM`, `mapload.Schedule`, `sim.Step` and `terrain.AnchorCell`, never a
re-derivation, over every map the install holds. Sources under `builds/_probe-0020/` (untracked).

```
$ AGAINROM_ASSETS="<install root>" builds/_probe-0020/probe.exe
map                            W     H  units    sched   moved  findings
scenario.res/10.alm           80    80     35      791      31  -
... 27 further campaign maps, then the 10 loose ones ...
Horror.alm                   256   256   1815    13251      81  -
38 maps decoded, 8094 units total through FromALM
FR-8 entity-cell vs unit-cross disagreements: 0
schedule targets outside the extent (units that start on the map): 0
units whose own record cell is already off the grid: 0
```

Read narrowly: over 8094 real unit records the two independent cell derivations never disagree, no
schedule target leaves the map, and every entity that has been ordered by tick 600 has moved. The
third line matters for DD-3's unstated precondition — **no shipped map places a unit off its own
grid**, so the gap below is real but unreached by real content. `moved` caps at 81 because entity
`i`'s first order arrives at tick `7i`, so only ids 0..85 have one by tick 600; of those, five stand
on their start cell at that instant, four because their 24-leg script has *finished* and its last
leg goes home.

## Mutants, re-run in this seat

Each mutation is applied to the tree, `go test -trimpath -count=1` over the five packages decides,
the tree is restored. Five controls are included so a survivor cannot be read as a suite that fails
to run.

```
M1  the entity cells pushed BEFORE the step, not after       SURVIVED  (not equivalent - disclosed)
M2  the schedule re-derived on every tick                    SURVIVED  (argued equivalent)
M3  the world built from a SECOND decode of the same bytes   SURVIVED  (argued equivalent)
M4  EntityMarkerRects' own clip DROP branch deleted          SURVIVED  (unreachable)
M5  EntityMarkerRects skips the map-extent clip entirely     SURVIVED  (unreachable)
K1  clipToMapRect's drop branch deleted for EVERY glyph      KILLED  (5 tests, pkg/render/terrain)
K2  the entity pass APPENDED last instead of prepended       KILLED  (1 test,  pkg/ui)
K3  the square's side 10 -> 12                               KILLED  (7 tests, terrain + pkg/ui)
K4  stagger 7 -> 8                                           KILLED  (2 tests, pkg/mapload)
K5  the map arm calls tick() TWICE per front-end tick        KILLED  (1 test,  pkg/ui)
K6  the schedule indexed by the world's tick + 1             KILLED  (3 tests, pkg/game)
K7  choose keeps a tick it already holds                     KILLED  (flow_test, "choose replaces
                                                                      a tick the flow already
                                                                      holds")
K8  escape leaves the tick in place (K7's mirror)            KILLED  (2 tests, pkg/ui)
```

## What no test sees

- **The thinnest seam in the story is M1, and it is a real defect nothing catches.** Push the cells
  before the step and the viewer holds the *previous* tick's positions — one frame of staleness,
  every frame. It is unkillable from `pkg/game` because `pkg/ui` exposes only
  `EntityMarkers() (cells int)` and that count is invariant under position. Killing it would mean
  adding an exported cells accessor to `pkg/ui`, reversing DD-5's deliberate decision not to hand
  the slice back, for a test-only benefit against a defect that is cosmetic in a diagnostic overlay
  and touches neither determinism nor FR-10. **Accepted as a disclosed limitation.** What *is*
  witnessed instead: cells arrive on every tick (the count moves with the world), and the values
  the driver builds are right (SC-2, SC-8).
- **AC-5 at the large k is weaker than it reads.** `Schedule`'s last leg returns every entity to its
  start cell, so at a k comfortably past the end a correct world and a world given *no commands at
  all* hold identical cells and differ only by the tick — which `Hash` does cover, but that is a
  much smaller claim than the criterion sounds. k = 575 was chosen because an entity is still
  mid-walk there and the test asserts it; **k = 10 is what actually discriminates.** The probe above
  shows the return-home is real on shipped maps. A future change to `legs` — 24, so the last leg is
  index 23 and `23 mod 4 == 3`, the home leg — could quietly make the large k vacuous.
- **The map-extent clip's drop branch is unreachable through the entity glyph for the whole of
  0020** (M4, M5). Not a defect: `s <= cellpx` at every scale, so the square never leaves its own
  cell, and every exported builder passes a zero lift. K1 shows the same branch is very much alive
  for the three crosses, whose arms do reach past a map edge — so the shared helper is not dead
  code, only inert on this one path. What would make it reachable is the raster twin DD-4 rejected.
- **DD-3's "every target lands inside the map" has an unstated precondition.** It holds for a unit
  whose own cell is in the extent, and the format does not constrain a unit to the map. No clamp was
  added: the arithmetic is not pinned by any decoded fact, and `pkg/sim` deliberately lacks such a
  constraint. Stated, not fixed. The probe found no shipped map that reaches it.
- **`legCells` and `legs` are both 24, so transposing them is the same program for every input.**
  That survivor is unkillable in principle, not thinly tested. K4 shows each constant is separately
  load-bearing.
- **Two survivors are argued-equivalent.** M2 re-derives the schedule per tick: `Schedule` is a pure
  function of the map and is pinned deeply equal across calls, so the values are identical and only
  the allocation differs. M3 rebuilds the world from a second decode of the same bytes: FR-1's "one
  decode" claim is **structural** — it rests on `loadMap` passing the `mv.Map` its own call
  produced — and has no observable, because two decodes of one byte slice agree.
- **The order of `tick()` and `v.step` inside `App.step` is asserted nowhere, by design.** DD-1 says
  it is unobservable — neither reads what the other writes — and Stage 2's adversarial read deleted
  the claim that it was pinned. Swapping them today changes nothing; it stops being free at the
  first state either one reads from the other.
- **`cmd/mapview` links `pkg/sim` transitively**, through `pkg/game`'s `LoadMapViewer`. FR-3 is
  about *owning and advancing* a world, not about the link graph, and it is witnessed by the test
  that `LoadMapViewer` returns a viewer with no entity cells and no tick — not by the import check,
  which would not fail if the standalone viewer began advancing one.
- **AC-11/SC-10 is the only witness for one real thing.** This is the first real map ever through
  `FromALM`, and **no automated test opens a campaign map** — every fixture in the suite is a 12x9
  `alm.Map` literal. The probe above is a developer run, not a test, and it sees arithmetic, not
  pixels: nothing automated has ever drawn an entity square.
- **Nothing measures performance, frame time, or the cost of pushing one slice per tick** over a map
  with 1815 units. No criterion asks for one, and none of the numbers above is a timing.

## The build

Confirmed independently in this seat: **the game binary now reaches the simulation core**, which is
exactly what 0019 recorded as absent.

```
$ go list -deps ./cmd/againrom | grep -c 'againrom/pkg/sim'
1
$ go build ./cmd/againrom && echo built
built
```

`builds/0020-sim-render-snapshot/` belongs to the build stage and is not created here. What AC-11
needs is the built `cmd/againrom` run with `-assets`/`AGAINROM_ASSETS` against a lawful install, a
campaign map chosen from the picker, and the `-markers` switch on so the unit cross is drawn beside
the entity square. What it would show: at tick 0 every magenta square standing **under** the cyan
unit cross it was built from — the cross reads on top, because content is drawn beneath the
instrument — and then the field walking laps of a square, entity by entity, each starting seven
ticks after the one before it, with nothing else on screen changed.

## Contract findings

Nothing in `spec.md` (12488 B, 824 free) or `plan.md` (13211 B, 101 free) was found false, and
neither is edited. Two things are recorded rather than corrected: DD-3's in-map guarantee holds only
for a unit already on the grid, and SC-4's second k does less than it appears to. Both are above.

One defect **was** found and fixed, outside the contract: `pkg/game/world_test.go` landed at
`c9ff4a6` unformatted, so `gofmt -l $(git ls-files '*.go')` — a standing gate — was red at HEAD
before this stage. `4a90f6a` reformats six end-of-line comments and nothing else. It carries no
trailer: it is not a task, and no task entry describes it.

## Owner ruling — AC-11 / SC-10

2026-07-29, over the build in `builds/0020-sim-render-snapshot/` against the owner's lawful GOG
install. The owner saw the AC-11 picture (a screenshot was shown: magenta squares with centred
cyan crosses on units, red crosses at structure bases), confirmed motion on small maps and the
staggered partial motion on a large one — the designed 7-tick stagger — and ruled **ок**.
AC-11 and SC-10 are closed; nothing of this story remains open.
