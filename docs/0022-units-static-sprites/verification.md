# Verification — units as static sprites

Task commits, oldest first: `bca6eb0` (T1), `e90604a` (T2), `6c7ecc4` (T3), `c59d8a9` (T4),
`2c254da` (T5), `a11086d` (T6). Base `53cfac3` — the plan and its tasks; before it the Stage-1
commit `8fb580b` and two owner-ruling document commits of earlier stories. No untrailered commit
sits anywhere in the task span — the first story since the gate whose tasks needed no mid-story
repair (0020 and 0021 each carried one `gofmt` fix). Submodule pin frozen at research `778c2a6`
throughout.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Evidence at `a11086d`. Two kinds of
number appear below: the synthetic suite, which reads no install, and a **developer census over
a lawful install**, marked *(probe)*. No probe result substitutes for a criterion the suite
carries.

## Gates

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 520 tests pass, 1494 counting subtests, 22 packages ok, 5 without tests)
      pkg/sim              55 tests,  87 counting subtests
      pkg/mapload          14 tests,  14
      pkg/render/terrain  128 tests, 277
      pkg/ui               88 tests, 291
      pkg/game             26 tests, 134
      cmd/terraintool      22 tests,  86
      cmd/againrom          5 tests,  36
      internal/synth        7 tests,  34
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0022-units-static-sprites
  analysis.md  6766 / 7168   provenance.md  7165 / 7168
  spec.md     12994 / 13312  plan.md       12910 / 13312
  tasks.md T1..T6 all under 1400; legend+traceability 539 / 1200
  verification.md (prose) 6558 / 9216 - fenced blocks free
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok                     (exit 0)
$ sh scripts/check-sdd-audit.sh
check-sdd-audit: 55 trailered commit(s) in ac6bd87..HEAD checked
  [41 notes/warnings on 0000-0019 ids and absent builds/ READMEs; pre-existing and unenforced.
   Elided rather than pasted: their text names ids, and an id pasted into this file would be
   counted as a witness of ours. 0022 draws ZERO id notes.]
FAIL 0022-units-static-sprites: every task in tasks.md has landed and there is no verification.md
check-sdd-audit: FAILED
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 53cfac3..HEAD | sed '/^$/d' | sort | uniq -c
      1 0022-units-static-sprites/T1  ...  1 0022-units-static-sprites/T6   (6 ids, each exactly once)
$ git log 8fb580b^..HEAD --format='%B' | grep -ci co-authored-by
0
```

Before this commit `check-sdd-audit` was **FAILED** on that one line. This file clears it; that
is the stage's own acceptance, and the commit carries no trailer.

## Witnesses

Every id and the named thing that answers for it. Everything but the rows marked *(probe)* and
*(manual)* runs windowless, reads no game install and consults no clock (FR-8). Expected values
are hand-written literals beside their fixtures throughout — bytes, cells, anchors, counts —
never values recomputed through the code under test.

```
FR-1  AC-1  AC-2  P-1  SC-1        pkg/sim: binary_test.go, hash_test.go, step_test.go, world_test.go
      TestMarshalPutsEveryFieldAtItsDocumentedOffsetAndWidth - the hand-written offset table
      partitions the 79-byte two-entity version-2 form: class at record +20, presence at +24,
      29-byte header, 25-byte records, every number spelled out.
      TestMarshalledBytesArePinned · TestThePinnedBytesDecodeBackToThePinnedWorld
      TestHashIsPinned · TestThePinnedDigestIsFNV1aOfThePinnedBytes - the pin trio re-pinned at
      version 2: hand-transcribed bytes over three entities with distinct class ids, one
      negative, the digest re-derived through the in-test FNV-1a.
      TestMarshalRoundTripsAndReMarshalsIdentically · TestEveryFieldChangesTheDigest (the class
      row included) · TestUnmarshalRefusesEveryVersionButTwo (the shipped 1 included) ·
      TestUnmarshalRefusesAWellFormedVersion1Stream · TestUnmarshalRefusesAndLeavesTheReceiverExactlyAsItWas.
      TestStepIsIndependentOfEveryClassID - two worlds apart only in class ids step in lockstep.
      TestWorldExportedMethodSetIsPinned green unedited; the T1 diff touches only the five files
      its task names - step.go, run.go and world_test.go untouched (git show bca6eb0 --numstat).

FR-2  AC-3  SC-2                   pkg/mapload/fromalm_test.go
      TestFromALMBuildsOneEntityPerUnit - keys in slice order over an alm.Map literal; class -24
      is the sign-extension witness (zero-extended it would load 65512); 200 names no class and
      arrives raw; cells, ids, bounds, tick exactly the 0020 literals.
      TestFromALMIsIdenticalAcrossLoads · TestFromALMSeedsFromTheConstant
      TestSteppingALoadedWorldLeavesTheMapUnchanged. schedule.go and schedule_test.go untouched
      by the T2 diff (git show e90604a --numstat: fromalm.go and fromalm_test.go alone).

FR-3  AC-4  SC-3                   pkg/game/units_test.go, internal/synth/reg_test.go
      TestLoadUnits - 5 subtests over a synthetic archive: drawable classes at both Flip values,
      keyed by ID, canvas across, SHEET FRAME 0 under either layout (3x2 and 2x2 where each
      sheet's frame 1 differs in size); File inherited through Parent onto the class's own
      canvas; each exclusion - palette-less, undecodable, absent, no-frame-0 - keeps a frameless
      entry distinct from a missing id at either sign; only the unreadable or unparseable
      registry errors, the read error an unwrapped *fs.PathError naming units/units.reg; a nil
      archive errors, never panics. Headless, no GPU. TestUnitsRegBuilder - DD-9's builder.

FR-3  AC-5  SC-4                   pkg/render/terrain/units_test.go
      TestUnitPlaceAnchorRuleAtTheFrameSize - hand literals: 64x96 canvas, centre (30,80), 20x24
      frame -> anchor (8,44), top-left (104,132) at cell (3,5); a frame LARGER than its canvas
      places at (32,-30); Rect() is the frame's own 20x24, Ground() the cell centre.
      TestUnitPlaceDisplacesVerticallyOnly - five lift/originY pairs, the zero-sum (1,-1)
      included: exactly -(lift+originY) in Y, zero in X, the anchor pixel unmoved.
      TestUnitPlaceNilAndFramelessPlaceNothing · TestUnitSetSparseSignedKeys.

FR-4  AC-6  SC-5  P-2              pkg/ui/entity_overlay_test.go
      TestEntityLayerRoutesSpriteOrSquareInSliceOrder - 7 entities: resolved at two arts,
      no-class, frameless, off-map WITH art and without; one walk in slice order, off-map
      dropped from both lists on the one bounds test; placements against hand-transcribed
      formula literals, displaced AND flat, the lifts read off the fixture's own altitude table.
      TestOverlayPassesPrependSpritesThenSquares · TestEntitySpriteCullKeepsACrownOnlySprite -
      the exact-rect cull keeps a sprite whose ground cell is off-view; both discriminators
      (ground point below the view, cell row outside the tile band) asserted, not assumed.
      TestEntitySquaresPlaceThroughTheFamilyTransform (5 camera positions) ·
      TestEntitySquaresCullOnThePlacedRect · TestEntitySquareCarriesItsCellsLiftDisplacedAndNoneFlat.
      TestDrawPaintsEntitySpritesLazilyAndSkipsZeroArea - textures lazy on first draw, once per
      frame identity; a zero-area frame places and culls but paints nothing and builds none.

AC-7  SC-5                         pkg/ui/entity_overlay_test.go, pkg/game/world_test.go
      TestEntityPassesPrecedeTheThreeDiagnostics - 5 passes: sprites, squares, objects, units,
      statics; the fixture refuses to run unless an entity square really covers a unit cross.
      TestViewerGivenNoEntitiesIsTheOneThatShipped - never-given yields the shipped 3-pass
      slice; an art-less push yields exactly the 0020 4-pass slice, no sprite pass anywhere;
      empty and off-map lists yield no pass; a shrunken set leaves no stale rects.
      TestSetEntitiesResyncsNothing, with its SetUnits control.
      TestOnlyTheGameFrontEndBuildsAWorld/the shared load path owns no world - a viewer out of
      LoadMapViewer holds no entity cells over a map with units.
$ git diff --stat 53cfac3..HEAD -- cmd/mapview internal/archtest        (no output: both untouched)

FR-5  AC-8  SC-6                   pkg/game/world_test.go
      TestSpriteGroundPointLandsOnItsUnitCrossAnchorAtTickZero - the two production derivations
      (entityDraws + UnitPlace's Ground() against MarkerCells + MarkerAnchor), an INDEPENDENT
      terrain.Project over a copy of the fixture's altitude bytes, each side carrying its own
      cell's lift; exact screen-pixel equality, both geometries, three cameras; non-vacuity
      guarded - displaced really displaces, unevenly, and every fixture unit resolves.
      TestPerturbingTheClassCentreMovesOnlyTheSprite - centre +(3,2): the art moves on both
      axes, the cross stands, and BOTH ground points still land on it - the instrument's bound
      stated exactly. The ui half: the pass tests above pin that the sprite pass draws exactly
      UnitPlace's placement at the mode's own lift terms and the square pass exactly the family
      transform, so the compared expressions are the drawn ones by composition (DD-7).

FR-6  AC-9  SC-7                   pkg/game/world_test.go, internal/archtest
      TestDrivenWorldReachesTheDigestOfAHeadlessRun - k=10 mid-schedule and k one past the end;
      the driven side resolves through the FULL fixture bundle on every one of the k pushes,
      the headless side holds no bundle and no viewer; digests equal, class ids included; both
      k values are checked to have moved somebody and to differ from an untouched world's.
      TestEntityDrawsHandArtExactlyWhereTheBundleHoldsAFrame - art is the bundle's own entry by
      POINTER IDENTITY exactly where that entry holds a frame; a frameless entry, an id naming
      no class and a nil bundle all cross as nil; cells ride unchanged; the world's digest is
      untouched by resolving. TestLiveTreeClean and the sim source scan green unedited.
$ grep -rn 'againrom/pkg/sim' pkg/ui/*.go                               (no output)

FR-7  AC-10  SC-8                  cmd/againrom/main_test.go, pkg/game/world_test.go
      TestUnitBundle - AC-10's two install LAYOUTS: the complete one loads the bundle once in
      NewFrontEnd and keeps it (front.Units non-nil), the omitUnitRegistry one fails -check
      naming units/units.reg with nothing on stdout; the whole run is the headless path, which
      has no graphics context to build a texture in. cmd/againrom/main.go is untouched by the
      T5 diff - no flag, no line. TestOnlyTheGameFrontEndBuildsAWorld/the front-end's loader
      builds one per opening - ten ticks of pushes, then a second, fresh opening.

FR-3  AC-11  SC-9                  pkg/game/census_test.go, cmd/terraintool/units_test.go
      TestUnitCensus - 3 sprites (one at a NEGATIVE framed key: a census that zero-extended
      would call it no-class), 2 no-frame (an excluded class and a nil entry), 2 no-class (one
      negative); the buckets partition the entities; a nil and a zero set census all-no-class;
      a unitless map censuses zeroes. TestUnitsCensusLine - the exact line, byte for byte,
      through -assets, -graphics and AGAINROM_ASSETS over a pairwise-distinct 3/2/1 fixture.
      TestUnitsErrors - 7 failure cases, each an error with NOTHING printed.
      The corpus half is the probe below *(probe)*.

FR-8  P-3                          the whole run above
      AGAINROM_ASSETS is unset in this seat and the suite is green, windowless: every fixture
      in the packages the story touched is a literal, a synth-built stream or a hand-assembled
      bundle. P-3's witnesses: the art-less push is the 0020 slice byte for byte, the art-less
      viewer's Draw builds no texture, and the nil-bundle half of the entityDraws test.

AC-12  SC-9  (manual)              the owner's run - OPEN, see below
```

## The corpus census over a lawful install *(probe)*

AC-11's evidence, run in this seat with the production `terraintool units` — T6's shipped tool,
built from the evidence tree — against the GOG install (read, never modified). The 10 loose maps
were censused in place; the 28 campaign maps live inside `scenario.res`, whose entries the
shipped `restool extract` wrote to a session temp directory **outside the repo** — extracted
bytes are game assets and enter neither the repo nor `builds/`. No probe source beyond the two
shipped tools exists, so `builds/_probe-0022/` holds nothing; the landing gate's asset scan runs
after all of this.

```
$ terraintool units -assets "<install root>" -map <map>      one line per map, verbatim counts
map                    entities  sprites  no-class  no-frame
scenario.res/10.alm          35       35         0         0
scenario.res/20.alm          56       56         0         0
scenario.res/30.alm          39       39         0         0
scenario.res/31.alm          30       30         0         0
scenario.res/40.alm          47       47         0         0
scenario.res/41.alm          42       42         0         0
scenario.res/50.alm          73       73         0         0
scenario.res/51.alm          17       17         0         0
scenario.res/60.alm          84       84         0         0
scenario.res/61.alm          43       43         0         0
scenario.res/70.alm         109      109         0         0
scenario.res/71.alm          75       75         0         0
scenario.res/80.alm         106      106         0         0
scenario.res/81.alm          19       19         0         0
scenario.res/90.alm         154      154         0         0
scenario.res/91.alm          45       45         0         0
scenario.res/100.alm        113      113         0         0
scenario.res/101.alm         41       41         0         0
scenario.res/110.alm         39       39         0         0
scenario.res/111.alm         81       81         0         0
scenario.res/120.alm         96       96         0         0
scenario.res/121.alm         43       43         0         0
scenario.res/130.alm        115      115         0         0
scenario.res/131.alm        222      222         0         0
scenario.res/140.alm        171      171         0         0
scenario.res/141.alm         68       68         0         0
scenario.res/150.alm        133      133         0         0
scenario.res/151.alm        237      237         0         0
Beast.ALM                   964      964         0         0
Cross.ALM                   774      774         0         0
Forester.alm                672      672         0         0
Horror.alm                 1815     1815         0         0
Islands.alm                 434      434         0         0
Kids.alm                     51       51         0         0
Kids2.ALM                    88       88         0         0
LuMoir.alm                  228      228         0         0
Tomb.ALM                    462      462         0         0
Waters.alm                  273      273         0         0
38 maps, 8094 entities, 8094 sprites, 0 no-class, 0 no-frame, 0 failure(s)
cross-foot: sprites + no-class + no-frame == entities on every row
```

Read narrowly: every class key a shipped map stores names a registry class, and every one of
those classes resolves to a drawn standing frame — there is no id to name, on any map. The 38
maps and 8094 entities equal the 0020 probe's census through the same `FromALM`, so the two
probes agree about what the corpus holds. What the sweep checks is resolution and counting, not
pixels: no placement, no draw and no comparison of art runs here.

## Mutants, re-run in this seat

Each mutation is applied to production code only, `go test -trimpath -count=1 ./...` over the
whole module decides, the tree is restored and checked clean after each. Two controls with
predicted kills calibrate the campaign against known-caught defects.

```
M1  UnmarshalBinary accepts version byte 1 beside 2       KILLED  (3 tests, pkg/sim: both version-refusal
                                                                   tests + the receiver-unchanged one)
M2  the class decoded from record +16, TargetY's slot     KILLED  (4 tests, pkg/sim: the pinned decode,
                                                                   round-trip, interrupted run, refusals)
M3  entityLayer's bounds test moved after the routing     KILLED  (1 test, pkg/ui: the routing test's
                                                                   off-map-WITH-art entity joins no list)
M4  UnitPlace's ok routed to squares, !ok to sprites      KILLED  (9 tests, pkg/ui)
M5  census buckets swapped: no-class <-> no-frame         KILLED  (pkg/game: the no-bundle subtest;
                                                                   cmd/terraintool: the census line, all
                                                                   3 archive-locating subtests)
C1  control: UnitPlace anchors at the CANVAS size         KILLED as predicted  (1 test in pkg/render/
    instead of the frame's                                 terrain + 5 pass tests in pkg/ui)
C2  control: FromALM zero-extends the class key           KILLED as predicted  (2 tests, pkg/mapload +
                                                           pkg/game's census over its framed negative key)
```

No survivors. Two results carry information beyond the kill:

- **M5's kill does not come from the census test's headline subtest.** "The three answers are
  counted apart" holds 2 no-class against 2 no-frame and is SWAP-SYMMETRIC — it passes under M5,
  verified in this seat. The kill is the no-bundle subtest (7 no-class) plus the tool's
  deliberately pairwise-distinct 3/2/1 line. Recorded so a future edit that drops either does
  not quietly leave the bucket swap alive.
- **Under C1 every pkg/game instrument test stays green.** The tick-0 ground-point equality is
  algebraically invariant to class geometry, exactly as the contract discloses — see the third
  note below, for which this is now mutation-level evidence.

## What no test sees

- **AC-7's "after the static art" is structural.** The entity layer's readable order is the pass
  slice's, pinned above; that the slice as a whole paints after the static-object art rides
  `Draw`'s shipped drawStatics-before-the-slice call order, disclosed in DD-6 and untouched by
  this story — and no test observes that call order itself. Drawing sprites between the two in
  `Draw`'s body was rejected for exactly this reason: it would put sprites-under-squares beyond
  observation too.
- **DD-4's resolved-`Flip` clause is vacuous at the fixed facing.** Facing 0 is sheet frame 0
  under either layout — 16 stored frames or 9 — so no branch reads `Flip` at all, and the loader
  tests pin the OUTCOME (frame 0 of each sheet, at both `Flip` values), not the block
  arithmetic. The clause starts binding at the first story that draws a second facing.
- **AC-8's equality is ALGEBRAICALLY invariant to class geometry.** The anchor cancels out of
  `Ground()`, so the tick-0 instrument catches a wrong lift, sign, origin or cell mapping —
  each moves one compared side alone — and cannot catch wrong class geometry, which moves only
  the art. That is caught by AC-5's hand literals plus the ui pass pins; the perturbation test
  states the bound and C1 above shows it live: under a canvas-for-frame anchor miswire the
  instrument stays green while the AC-5 and pass tests fail.
- **T4 renamed TestSetEntityCellsResyncsNothing to TestSetEntitiesResyncsNothing.** 0020's
  frozen verification.md cites the old name. A historical record cites what existed when it was
  written; left alone.
- **The square fallback binds only off-corpus.** The probe's zeroes mean no shipped map
  exercises the no-class bucket, the no-frame bucket, `UnitPlace`'s false arm or the square
  draw: every artless-path witness in the story is synthetic, and the owner's AC-12 run over
  shipped maps should see no square at all — a square there would itself be a finding.
- **Nothing automated reads a sprite's pixels.** The engine cannot read pixels back before the
  game starts, so the Draw test's witnesses are texture-cache identity, cache growth and the
  absence of a panic — transparency, nearest sampling and zoom scaling are asserted structurally
  (one blit path, the object layer's own) and judged by eye in AC-12. The probe checks counts,
  not art.
- **AC-12 is the only open item**: a lawful install, representative maps, opened, panned,
  zoomed, run — each resolving unit its own sprite under its own cross at tick 0, walking with
  the schedule; on shipped maps the corpus says every unit resolves, so squares appear only if
  something is wrong. It stays with the owner.

## Contract findings

Nothing in `spec.md` (12994 B, 318 free) or `plan.md` (12910 B, 402 free) was found false at the
implemented tree, and neither is edited by this stage. The span `53cfac3..a11086d` holds exactly
the six task commits, each trailered once, every gate green at the evidence tree; unlike 0020
and 0021, no untrailered repair was needed anywhere in it.

## Owner ruling — AC-12

2026-07-29, over the walkthrough page and the `builds/0022-units-static-sprites/` binaries run
against the owner's install. The owner ruled — verbatim, «22 все верно»: real sprites standing
on their crosses at tick 0, walking with the schedule, no fallback square anywhere. AC-12 is
closed; nothing of this story remains open.

One observation came back with the ruling and is deliberately not a finding of this story: tree
and unit sprites read far darker than the surrounding terrain. The sprite path draws the game's
own palette colors unshaded, so the inflated contrast points at the terrain lighting path, whose
over-brightening `docs/0007-terrain-lighting/verification.md` already measured (mean channel
ratio 1.5318, 98.23 % of pixels brighter). It joins that standing owner-deferred item; the
sprite path changes nothing.
