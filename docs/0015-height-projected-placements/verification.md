# Verification — height-projected placements

Commits, oldest first — tasks trailered, contract amendments not, base `449538b`:

```
$ git log --reverse --format='%h %s | %(trailers:key=SDD-Task,valueonly)' 449538b..908aea8 | sed '/^$/d'
6cbb43d Add Projection.AnchorHeight for height-projected marker placements | 0015-height-projected-placements/T1
21c062b chore(sdd): evidence is owed when the LAST task lands, not the first |
f4ceb00 Add DrawObjectMarkersAtHeights/DrawUnitMarkersAtHeights per-marker lift | 0015-height-projected-placements/T2
1777c78 Wire AnchorHeight into cmd/terraintool's marker overlays | 0015-height-projected-placements/T3
c3c745c docs(0015): FR-2 made a sibling story's feature unreachable; FR-8 is the fix |
78a6c72 Add Viewer.SetFlat; Mode() supersedes the overlay rule with a deliberate flag | 0015-height-projected-placements/T4
e7fab4a ui: displace object/unit markers onto their anchor cell's own height | 0015-height-projected-placements/T5
ca78033 docs(0015): the lift guards on the MODE, not on the projection |
ce32355 cmd/mapview: add -flat, wiring it to Viewer.SetFlat | 0015-height-projected-placements/T6
f6ec3b3 docs(0015): SC-4 could not name the file it needed to; T7 adds it |
7eb67b2 cmd/mapview: document -flat in the tool's own flag surface | 0015-height-projected-placements/T7
1edd16b docs(0015): T8 - P-2's bound and P-6's drawing half are asserted nowhere |
908aea8 Assert P-2's bound and P-6's drawing half for height-projected placements | 0015-height-projected-placements/T8
```

Developer runs used `builds/0015-height-projected-placements/` built at **`7eb67b2`** (T7/T8
changed no production code, so it represents HEAD) against a pre-story worktree at **`449538b`**;
unit evidence is `go test` at **`908aea8`**. The lawful install was named on the command line
only; no asset entered the tree, no capture left the scratchpad.

## Revision — 2026-07-29 (docs only)

At pin `778c2a6` `TERR-SPR-041` is retracted: its `vt+0x30` dispatch draws the HP/mana bars. The
real unit draw, `vt+0x2c`/`R0553` (`TERR-SPR-048`), ignores `(col,row,alt)` and places
from the unit's own `+0x60/+0x64` minus the sprite anchor minus `+0x68` — the raw-position model
provenance had filed as the unresolved rival; the object-path lift (`TERR-SPR-038…040`) stands.
No behaviour changes — the markers' cell-mean lift is ours by choice; spec, provenance, analysis
corrected. Downstream: applying the terrain lift to a unit double-counts it, and the unit
marker/sprite disagreement 0022 will show is true, not a bug.

## Gates

```
$ go build ./...                                 (clean)
$ go vet ./...                                   (clean)
$ gofmt -l $(git ls-files '*.go')                (no output)
$ git status --short                             (empty)
$ sh scripts/check-no-game-assets.sh --history
check-no-game-assets: clean (history scan)
$ sh scripts/check-doc-budget.sh                 (exit 0; 0015 under its declared spec/plan overrun)
$ go test -count=1 ./...
ok againrom/cmd/againrom      0.964s   ok againrom/pkg/formats/alm    0.329s
ok againrom/cmd/mapview       0.910s   ok againrom/pkg/formats/reg    0.442s
ok againrom/cmd/regtool       0.350s   ok againrom/pkg/formats/res    0.442s
ok againrom/cmd/terraintool   1.069s   ok againrom/pkg/formats/spr256 0.359s
ok againrom/internal/archtest 0.400s   ok againrom/pkg/game           0.795s
ok againrom/internal/notices  0.334s   ok againrom/pkg/render/camera  0.351s
ok againrom/internal/synth    0.338s   ok againrom/pkg/render/frame   0.476s
ok againrom/pkg/render/menu   0.541s   ok againrom/pkg/render/terrain 0.577s
ok againrom/pkg/ui            1.104s
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 449538b..HEAD
T1 T2 T3 T4 T5 T6 T7 T8   (one each, none twice; no Co-Authored-By anywhere in the range)
```

## Unit evidence

Every assertion below runs windowless, holding no graphics context (**SC-10**, FR-7).

**AC-1 / SC-1** `TestAnchorHeightMatchesRecipe` checks a hand-computed truncating mean over
flat, sloped and negative fixtures at interior, edge, corner and past-`Width`/`Height` indices;
`TestAnchorHeightAtIntegerExtremes` repeats it at `math.MinInt`/`math.MaxInt`;
`TestAnchorHeightClampsBeforeFormingTheFarCorner` is DD-1's named trap;
`TestAnchorHeightTruncatesTowardZero` pins the sign of the truncation.

**AC-2 / SC-2** `TestAnchorHeightNeverMutatesTheAltitudeSlice` byte-compares the borrowed slice after
sweeping every fixture.

**AC-3 / AC-4 / SC-3** `TestModeDisplacedIsChosenByValidityNotByRelief` and
`TestModeFollowsOverlaysEnabledAfterConstruction` hold Displaced across all four overlay combinations
over a valid grid; `TestModeFlatInputs` and `TestFlatTileScreenIsTheUndisplacedLattice` hold Flat,
and the un-displaced lattice, over an invalid one.

**AC-5 / P-4 / SC-5** `TestDisplacedMarkerSharesOffsetBetweenObjectAndUnit`. Its oracle rebuilds the
expected rectangle from the **flat** arm shifted by `-AnchorHeight(col,row) - MinV` through an
independently built `terrain.Project`, never through `v.proj`, then compares X, Y, width and height —
so a horizontal component fails it, which is P-4.

**AC-6 / SC-6** `TestDisplacedMarkersOverlapResolvedByDrawOrder`: two markers whose offset rectangles
newly overlap still draw objects then units.

**AC-7 / SC-7** `TestMarkerLiftClipsAtUnliftedTopExtent`/`…BottomExtent` and their `cmd/terraintool`
counterparts pin both halves of DD-2's two-stage clip in both sign directions;
`TestRenderMarkerLiftTranslatesInteriorAnchorBySignedScaledAmount` covers the `*scale` factor.

**AC-8 / SC-9** `TestFlatFlagCheckOutputByteIdentical`, plus M3 below.

**AC-10 / SC-13** `TestSetFlatSelectsFlatWhileStayingLit`, `TestSetFlatSuppressesMarkerHeightOffset`,
`TestFlatFlagSelectsFlatModeWhileStayingLit` (over a sloped fixture self-checked against
`CanvasHeight()` so it cannot degenerate), `TestFlatFlagIsDefined`, and `TestFlagSet`.

**P-1** `TestAnchorHeightFlatCellReturnsTheAltitude`, with the purity half from AC-2's test.

**P-2** `TestAnchorHeightBoundedByCornerExtremes` — asserted, not inferred. The four corners are read
independently through `Altitude` (never through `AnchorHeight`, which would be a tautology) and the
result bracketed between their min and max, over a negative fixture where truncation toward zero
rather than floor is what a bound argued from `/4` gets wrong.

**P-3** `TestRenderFlatOutputNeverLiftsMarkers`, `TestSetFlatSuppressesMarkerHeightOffset`, and M1/M5
below, which measure it on real maps rather than fixtures.

**P-5 / SC-11** `TestMarkerLiftCullsAfterTheHeightOffset` and `TestDisplacedCullRunsAfterOffset` cover
both directions: a rectangle entering the view only after the offset is drawn, one leaving it dropped.

**P-6** Construction by `TestConstructionLeavesTheAltitudeSliceUntouched`, the lookup by AC-2's test,
and — added by T8 — `TestMarkerLiftNeverMutatesTheBorrowedAltitudeSlice`
and `TestOverlayScreenRectsNeverMutatesTheBorrowedAltitudeSlice`, which byte-compare the slice across
a displaced **draw** in both front-ends. Such a mutation can only originate in `AnchorHeight`:
DD-2's seam hands the draw path a `liftY` closure, never the slice.

**SC-4** Exactly four existing test files changed, measured by `git diff --name-only 449538b HEAD`
and not by judgement: `cmd/mapview/flagset_test.go`, `cmd/terraintool/main_test.go`,
`pkg/ui/light_test.go`, `pkg/ui/mode_test.go`. Five new files carry the rest; T8's three were created
by this story and do not count against the four.

**SC-8** `TestRenderFlatOutputNeverLiftsMarkers`, and M1 at MD5 level on real maps.

## Mutation evidence

Each task's claim was checked by applying a break and observing which tests turned red:

```
T1 6cbb43d  clamp moved after col+1 is formed  -> 3 tests, incl. ...ClampsBeforeFormingTheFarCorner
T2 f4ceb00  liftY folded into mapRect          -> exactly 2 tests (the trap FR-4 names)
T3 1777c78  the !*flat guard dropped           -> only TestRenderFlatOutputNeverLiftsMarkers
T4 78a6c72  Mode() ignores v.flat              -> 4 named tests
T5 e7fab4a  guard weakened to v.proj != nil    -> exactly TestSetFlatSuppressesMarkerHeightOffset
T6 ce32355  -flat also forces SetUnshaded      -> "Lit() = false, want true -- -flat must not
                                                  suppress lighting"
T6 ce32355  run() never calls configureFlat    -> GREEN (see Honest notes)
T7 7eb67b2  -flat undocumented in both strings -> "usage line does not mention -flat"
T8 908aea8  AnchorHeight returns min-1         -> TestAnchorHeightBoundedByCornerExtremes
T8 908aea8  AnchorHeight returns h00 always    -> bound test PASSES, recipe test FAILS
```

## Developer-run evidence

Three maps of differing relief:

```
Cross.ALM   256x256  1066 anchors  AnchorHeight 3..110
Waters.alm  144x144   386 anchors  AnchorHeight 0..125  (hilliest)
Kids.alm    mild                   AnchorHeight 59..73
```

**M1 — SC-8, P-3 on real data.** `terraintool -flat`, HEAD vs pre-story, every combination that
rendered: **28/28 MD5-identical**.

**M2 — AC-7.** Projected mode, HEAD vs pre-story: **7/7 identical** with no overlay (this story moves
no terrain), **21/21 differing** with `-objects`, `-units` or both. Diff magnitude grows with
`scale²`, as `liftY = -AnchorHeight*scale` predicts.

**M3 — AC-8, SC-9.** `terraintool` summary HEAD vs pre-story: **36/36 identical**. `mapview -check`:
**12/12 identical** HEAD vs pre-story and **12/12 with and without `-flat`** (AC-10's summary clause).
`mapview`'s summary emits no geometry, dimension or origin token at all, which is why FR-2 and
AC-8 were compatible to begin with.

`terraintool`'s flat and projected summaries **do** differ, in dimension/`geometry`/`y origin` tokens
only. Not an AC-8 finding: AC-8 requires identity **to pre-story**, and those tokens have differed
between modes since 0012. Object, unit and tile counts are identical in every pair and cannot
differ — they are computed before the flat/projected branch.

**M4 — the size of the effect.** Max lift **125 px at scale 1** on `Waters.alm`, **110 px** on
`Cross.ALM`, **73 px** on `Kids.alm`. Crops around the maximum-lift anchor match the predicted shift
pixel-for-pixel. Tens to low hundreds of pixels on real maps, not one or two.

**M5 — AC-9, SC-12, manual.** Window captures via `PrintWindow` on the window handle only, never a
screen grab. Markers land on the relief; `Waters`'s maximum-lift marker sits on the slope edge in
the displaced render and at the unlifted position in the pre-story one.

The sharpest measurement here is a byte comparison, not an eye judgement:

```
9a06063d431eff0bfccc180bd1002ad5  waters_flat_ou.png       HEAD, -flat -objects -units
9a06063d431eff0bfccc180bd1002ad5  waters_prestory_ou.png   449538b, -objects -units
```

Before this story an enabled overlay forced flat — the rule FR-2 deletes — so pre-story
`-objects -units` **was** the flat scene, and `-flat` now reproduces it byte-for-byte through the
real window. That is P-3 end-to-end. The HEAD capture without `-flat` differs from both, so FR-2's
change is visible on the same map. Against a `-flat -unshaded` control the lit capture carries
shading and a drop shadow the control lacks: `-flat` is genuinely lit — the reachability FR-8 exists
to restore.

## Honest notes

**SC-12's expected edge artifact was not observed on real data.** A probe over every anchor on all
three maps found **zero** cases where a lifted arm reaches the un-displaced map extent. The behaviour
is pinned by adversarial synthetic fixtures in both front-ends; it does not arise on these maps.
Recorded as unobserved, not as seen.

**Eight cells were never compared.** `Cross.ALM` and `Waters.alm` at `-scale 4`, all four overlay
combinations, exceed `terraintool`'s 268 435 456-pixel cap. HEAD and pre-story failed identically,
but nothing rendered — gaps, not passes. Full scale 1/2/4 coverage exists only for
`Kids.alm`.

**One wiring step has no headless observable.** Deleting `run()`'s call to `configureFlat` leaves the
suite green. Structural, not an oversight: AC-8 requires `-flat` to change no `-check` output, which
removes the only headless observable, and `run()`'s window path needs a display. The control that
establishes this — deleting the analogous `configureViewer` call **does** fail `TestAnimationFlags`,
since `-noanimation` moves the water cadence the summary prints. The body is covered, the call site
is not.

**Two of three `mapview` captures do not discriminate AC-9**: on `Waters` every marker sits on the
flat shelf and `Kids Paradise` is near-level. Where the lift is ~0 an image cannot show it is right;
the discriminating evidence is the `terraintool` crop pair.

**A flat-mode shadow floats above its footprint**, identically in the HEAD `-flat` and pre-story
captures, so it predates this story.

**The high terrain brightness the owner observed remains**, deliberately not acted on.

## Not run

`cmd/againrom` gained no flag by design (FR-8), so no front-end run exercises `-flat`. No performance
measurement: this story adds four array reads and a shift per marker, and no criterion asks for one.
