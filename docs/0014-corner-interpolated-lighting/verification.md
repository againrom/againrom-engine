# Verification — corner-interpolated lighting in the window

Story commits, oldest first: `e4d9cb4` (T1), `bd2efd2` (T2), `20f7718` (T3), `f315cfd` (T4),
`62f112a` (plan amendment, DD-9/SC-13), `90a2eb6` (T3b), `9ec5636` (T7). Base commit `9ce7bd9`.
The developer runs used this story's `builds/` binaries at `90a2eb6`; the drag evidence is `go test`
at `9ec5636`.

## Gates

```
$ go build ./...                          (clean)
$ go vet ./...                            (clean)
$ gofmt -l cmd internal pkg                (no output)
$ bash scripts/check-no-game-assets.sh --history
check-no-game-assets: clean (history scan)
$ git status --short                       (empty)
$ go test -count=1 ./...
ok  againrom/cmd/againrom      1.076s   ok  againrom/pkg/formats/alm    0.394s
ok  againrom/cmd/mapview       0.997s   ok  againrom/pkg/formats/reg    0.486s
ok  againrom/cmd/regtool       0.393s   ok  againrom/pkg/formats/res    0.468s
ok  againrom/cmd/terraintool   1.064s   ok  againrom/pkg/formats/spr256 0.383s
ok  againrom/internal/archtest 0.419s   ok  againrom/pkg/game          0.877s
ok  againrom/internal/notices  0.436s   ok  againrom/pkg/render/camera 0.431s
ok  againrom/internal/synth    0.400s   ok  againrom/pkg/render/frame  0.571s
ok  againrom/pkg/render/menu   0.580s   ok  againrom/pkg/render/terrain 0.629s
ok  againrom/pkg/ui            1.152s
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 9ce7bd9..90a2eb6
0014-corner-interpolated-lighting/T4  T3b  T3  T2  T1   (one each, none twice)
```

The `pkg/render/terrain` Defender false positive named in the plan risk **did not occur** in
any run above.

`bash scripts/check-doc-budget.sh` passes; every 0014 artifact is within its ceiling and the
`spec >= plan >= tasks` chain holds (run separately, not pasted here — its own output names
every story in the repo, not just this one).

## Unit evidence

Every test below is windowless (SC-8): no test opens a graphics context, and `pkg/ui`'s vertex
tests read `ebiten.Vertex` output as struct fields. Each was checked against a deliberate break
in its own doc comment (transposed corners, a dropped `withScales`, a single scale reused for
all four, flat mode's other diagonal, `CornerLevels`/`ShadeScale` disagreeing at a clamp
boundary); every break failed the test meant to catch it.

```
SC-1   pkg/ui, pkg/render/camera, pkg/game — full suite green, NewViewer's signature
       unedited (TestNewViewerRejectsMalformedInput, TestNewViewerSetsUpCamera)
SC-2   TestUnshadedDrawIsUnitColourAtThePreChangeFlatLattice
SC-3   TestCornerScalesEqualLevelsYieldEqualMultipliers (equal case)
       TestCornerScalesMatchesShadeScaleOfCornerLevels (distinct-levels case, asserts >=1
       tile with 4 unequal corners)
SC-4   pkg/render/terrain TestShadeScaleFormula — every level 0..95, plus clamp
SC-5   pkg/render/terrain TestCornerLevelsMatchesCompositor — CompositeLit invoked in the
       test, interior/right-edge/bottom-edge/corner tiles
SC-6   TestLitAndModeAreIndependentPredicates — all 4 lit x mode combinations
SC-7   TestCornerScalesPassLeavesAltitudesUntouched — before construction AND after a full
       pass over every tile
SC-8   satisfied by construction: no _test.go file below calls ebiten.RunGame or opens a window
SC-9   developer run only, see below (no unit witness: it needs two built binaries)
SC-10  developer run only, see below
SC-11  pkg/render/terrain lit_test.go, project_test.go pass UNEDITED (TestCompositeLit,
       TestProjectedEqualsFlat, ...); TestCornerLevelsMatchesCompositor is the new check that
       the unification changed nothing. Re-run below: terraintool built from 9ce7bd9 and from
       HEAD render Kids.alm byte-identical, shaded and -flat (md5, see Developer runs)
SC-12  TestCornerScalesPlaceholderStaysUnlitWhileNeighboursAreLit
SC-13  TestDrawDisplacedSubmitsWithScaledTileVerticesThroughSharedIndices
       TestDrawFlatSubmitsWithScaledFlatVerticesThroughSharedIndices
SC-14  TestViewerStep/a_primary-button_drag_pans_by_the_screen_delta_divided_by_the_zoom,_opposite_the_cursor
       (zoom_1, zoom_2), plus a_button_already_held_on_the_very_first_tick_anchors_and_pans_zero
       and a_zero-move_drag_pans_zero,_at_any_zoom (zoom 1, 2, 0.5). Both named breaks run and
       reverted, see T7 mutation evidence below: sign inverted fails zoom_1 AND zoom_2; / z -> * z
       fails zoom_2 only (zoom_1 is z=1, where * and / agree).
SC-15  TestViewerStep/a_drag_inside_the_edge_margin_pans_exactly_once,_not_twice +
       TestViewerStep/the_same_cursor_position_with_the_button_up_still_edge-scrolls. Both named
       breaks run and reverted, see below: dropping the suppression fails the first; making it
       unconditional fails the second.

AC-1   TestCornerLevelsMatchesCompositor (asymmetric slope, x*11+y*23) and
       TestCornerScalesMatchesShadeScaleOfCornerLevels (cliffGrid, a raised 2x2 plateau) cover
       interior/edge/corner tiles over sloped grids. NOT covered by one test together: a flat
       grid, a pure single-axis ramp and a negative-altitude grid are each exercised elsewhere
       (TestCornerScalesEqualLevelsYieldEqualMultipliers is flat; mode_test.go's shortAlts/
       pushedAlts are single-axis/negative but are camera fixtures, never passed through
       CornerLevels in a test) — no single test sweeps all four named shapes for CornerLevels.
AC-2   pkg/render/terrain TestShadeScaleFormula (0,46,64,95 exactly, plus every clamp case)
AC-3   TestDisplacedCornersRetainTileVerticesGeometryAfterScaling (geometry byte-identical to
       tileVertices) + TestCornerScalesMatchesShadeScaleOfCornerLevels (the multiplier half)
AC-4   TestFlatCornersRetainFlatTileVerticesGeometryAfterScaling +
       TestFlatTileVerticesIsTheUndisplacedLatticeWithUnitColour
AC-5   TestLitAndModeAreIndependentPredicates
AC-6   TestUnshadedDrawIsUnitColourAtThePreChangeFlatLattice; cmd/mapview
       TestUnshadedFlagWiresToViewerLit, TestUnshadedFlagDoesNotChangeCheckOutput
AC-7   TestCornerScalesPlaceholderStaysUnlitWhileNeighboursAreLit covers the placeholder half.
       The water/phase half is NOT exercised by a test that advances the animation tick and
       re-reads cornerScales — it rests on cornerScales(tx,ty) taking no tick parameter at all
       (structurally phase-blind), which is a stronger guarantee than a passing assertion would
       be, but it is not what SC-8's "checked against a break" bar asks for. Flagged, not papered
       over.
AC-8   TestCornerScalesPassLeavesAltitudesUntouched
AC-9   cmd/mapview TestUnshadedFlagDoesNotChangeCheckOutput (unit) + the corpus run below
       (integration)
AC-10  manual/computed, see below
AC-11  manual, see below
AC-12  same TestViewerStep subtests as SC-14, plus
       a_drag_inside_the_edge_margin_pans_exactly_once,_not_twice and
       a_drag_cannot_pan_the_camera_past_the_clamp (clamp holds every step). Zooms
       driven: 1, 2, 0.5. NOT covered: a wheel-reachable non-integral zoom (WheelZoomStep=1.2) —
       no drag subtest runs at 1.2 or any zoom the wheel itself actually reaches away from 1.0.
AC-13  NOT RUN. Manual, both front-ends; synthetic input injection is prohibited in this project
       (see AC-11's note below) — see Not run
```

P-1 (equal->equal): `TestCornerScalesEqualLevelsYieldEqualMultipliers`. P-2's multiplier bound
is `TestShadeScaleFormula`'s range; the interior-lies-between-corners half is linear-interpolation
geometry `DrawTriangles` supplies and is not independently re-verified per pixel (SC-8 forbids a
graphics context). P-3: `TestCornerScalesUnlitIsAllOnes` plus the geometry-unchanged tests above.
P-4: `TestCornerScalesMatchesShadeScaleOfCornerLevels` (mode-independent) +
`TestLitAndModeAreIndependentPredicates` (overlay-independent); phase-independence per AC-7 above.
P-5: `TestCornerScalesPassLeavesAltitudesUntouched`.
P-6: the drag subtests above (zoom 1, 2) and the zero-move-drag subtest (zoom 1, 2, 0.5) —
a pure function of two consecutive cursor positions and the zoom; zero movement pans zero
at every zoom driven.

## T7 mutation evidence — SC-14, SC-15

Each edit below applied to `pkg/ui/viewer.go`, run via `go test ./pkg/ui/ -run
TestViewerStep -count=1 -v`, then reverted (`git checkout -- pkg/ui/viewer.go`);
`git status --short` was empty before the first edit and after every revert.

```
SC-14, break one: dragIntent `return -sdx/z, -sdy/z` -> `return sdx/z, sdy/z` (sign inverted)
  FAIL a_primary-button_drag_pans_.../zoom_1
  FAIL a_primary-button_drag_pans_.../zoom_2
  FAIL a_drag_inside_the_edge_margin_pans_exactly_once,_not_twice
  FAIL the_keyboard_keeps_working_during_a_drag_tick

SC-14, break two: dragIntent `return -sdx/z, -sdy/z` -> `return -sdx*z, -sdy*z` (* not /)
  FAIL a_primary-button_drag_pans_.../zoom_2
  (zoom_1 still PASSES: z=1, so * and / agree there -- the mutation is invisible at that zoom)

SC-15, break one: panIntent, the `if in.PrimaryDown { return dx, dy }` suppression DELETED
  FAIL a_drag_inside_the_edge_margin_pans_exactly_once,_not_twice
  FAIL a_button_already_held_on_the_very_first_tick_anchors_and_pans_zero
       (unplanned second casualty: its cursor, 777x555, sits inside the right/bottom
       EdgeMargin at an 800x600 view, so edge-scroll adds on top of the anchor's zero pan)

SC-15, break two: panIntent, the same block replaced with an UNCONDITIONAL `return dx, dy`
  FAIL the_same_cursor_position_with_the_button_up_still_edge-scrolls
  FAIL edge-scroll_fires_.../left_edge, .../just_inside_the_left_margin, .../right_edge,
       .../just_inside_the_right_margin, .../top_edge, .../bottom_edge,
       .../top-left_corner_scrolls_both_axes
       (7 more: this mutation disables ALL edge-scroll, not only the case SC-15 names)
```

All four reverted cleanly; none left `go build`/`go vet` broken.

## Developer-run evidence

Install: the lawful GOG install on this machine, via `-assets`. Corpus: the 38 shipped maps — 10
loose `.alm` plus 28 from `scenario.res` via `restool extract`, matching `againrom -check`'s own
`38 map rows`.

**SC-9 / AC-9 — re-run; the prior logs were 0 bytes.** `mapview` built fresh from `9ce7bd9` in a
worktree, `-unshaded` never passed (the base predates it), vs the current binary: 38 maps x
{none, `-objects`, `-units`, both} = 152 comparisons, logged to
`sc9_mapview_base_vs_current.log`: **152/152 identical**. A second 152-run sweep, current binary
only, `-unshaded` on vs off at the same combinations, logged to `sc9_mapview_unshaded_axis.log`:
**152/152 identical**. Shape, not just equality (see below): both logs cover all 38 maps, every
one of the 304 compared lines per log contains `cells (` — a real summary naming the map and its
cell count, not an error string. `againrom -check` against a base build: identical, 3 runs
stable:

```
$ grep -c '^MAP=' sc9_mapview_base_vs_current.log ; grep -oE 'MAP=[^ ]+' sc9_mapview_base_vs_current.log | sort -u | wc -l
152 comparisons over 38 distinct maps
$ grep -E '^(BASE|HEAD): ' sc9_mapview_base_vs_current.log | grep -vc 'cells ('
0   # no non-summary line among 304
$ grep -c '^DIFF$' sc9_mapview_base_vs_current.log ; grep -c '^SAME$' sc9_mapview_base_vs_current.log
0 152
$ grep -E '^(UNSHADED_OFF|UNSHADED_ON): ' sc9_mapview_unshaded_axis.log | grep -vc 'cells ('
0   # no non-summary line among 304
$ grep -c '^DIFF$' sc9_mapview_unshaded_axis.log ; grep -c '^SAME$' sc9_mapview_unshaded_axis.log
0 152
mapview:  Kids Paradise 80x80 cells (6400), tile slots 52/128, water speed 4 (16 tps, ...)
mapview:  61.alm 80x80 cells (6400), tile slots 52/128, objects 27, units 43, water speed 4 ...
againrom: 38 map rows, 8 of 8 buttons have a mask region                    (base == HEAD)
```

**SC-11 re-run.** `terraintool` built from `9ce7bd9` and from HEAD, `Kids.alm`, shaded and
`-flat`: all four renders identical dimensions; md5 **matches** on both pairs
(`dbd5535a...` shaded, `cec6de28...` flat) — the whole story's PNG path is unchanged, not just T1's.

**AC-10 — computed.** A throwaway probe (`builds/_probe-0014-verify/`, own `go.mod`, replaced
onto this module — see hazards) computes two orderings per adjacent cell pair over all 38 maps,
using only exported production code (`LevelGrid`/`CornerLevels`/`ShadeScale`, `Composite`/
`CompositeLit`): the PNG's real per-cell mean brightness, and a texture-cancelled "window
estimate" (`Composite`'s own unshaded mean x the corner-scale triangle-mean `cornerScales` would
compute). The **RAW** reading the brief suggests (PNG mean vs bare multiplier, no texture
cancelling) disagrees on 32% of pairs — because adjacent cells usually carry different textures,
which dominates raw brightness and has nothing to do with lighting; recorded, not hidden. The
**RATIO** reading (texture cancelled both sides) is the meaningful one:

```
maps=38 pairs=1,750,864
>0            level (any nonzero float): agree 1,376,605 disagree 323,421  81.0%
>1/64  (0.5 level of PNG difference):     agree 1,069,715 disagree  64,889  94.3%
>1/32  (1   level of PNG difference):     agree   753,842 disagree  20,238  97.4%
>1/16  (2   levels of PNG difference):    agree   388,158 disagree   6,372  98.4%
```

Agreement rises sharply once the PNG shows at least one level's worth of real difference — the
residual disagreements at low thresholds sit almost entirely at deltas below one level step (i.e.
sub-quantum noise in a MEAN-based proxy, not a sign a corpus map's relief reads backwards) and,
inspecting the first 40 by hand, cluster where the two cells' corner levels are **exactly equal**
(the "window" side reads a flat 0.0000 delta) while the true per-pixel-clamped PNG mean still
differs slightly — a limitation of a linear texture x mean-shade proxy that does not model
`ShadeChannel`'s 255 clamp, not a claim about the real window's pixels, which this probe cannot
read (SC-8).

**AC-10 — observed.** `Kids.alm` and `61.alm` (scenario map): `mapview` window screenshots
(scratchpad only) against `terraintool` PNG crops of the same region. `Kids`, lit vs `-unshaded`:
the lit window is visibly brighter/more saturated (grass greener, sand lighter) — the flat
daytime multiplier 1.5625 doing exactly what the formula says. `61.alm`: the window's default
1024x768 view and the PNG's top-left 1032x765 crop show the **same** hill silhouette, the same
rock/snow patch, and the **same** directional shading — a dark band across the checkerboard
flat ground and a lit-from-one-side dome on the hill — matching in both images. Both comparisons
were seen, not computed.

**SC-10 — the tying zoom, corrected.** Two probes exist; the prior draft conflated them. The
real-camera probe (`sc10.go`) covers **5 states**: `z=1.0/1.5/2.0` at one tie (`X=0.5`) each, plus
`z=1.2` at `X=0.25`/`7.75`. The **33-state sweep** (`z=1.0/1.5/2.0` x 11 `X` values) belongs to
the synthetic quad probe (`hole.go`), which also runs the same two `z=1.2` cases (35 states
total). Re-run fresh below: both probes agree on all 33 "32z-integer" states (no hole) and on the
`z=1.2` tying edges (bit-identical, see below) — but disagree on the pixel at
`z=1.2`: `hole.go` shows FR-1's one-pixel unpainted column; `sc10.go` shows none, there or
anywhere, every rerun.

Not the vertex data: a temporary, deleted `pkg/ui` test drove the real `drawFlat` through
`recordingTarget` at this state and read `DstX` as bits — `0x42990000`, 76.5 exactly, matching
both probes. The brief's "one probe a hair off in float32" hypothesis is **false**, measured. A
follow-on scratch diagnostic (`diag.go`, not kept evidence) varied canvas size, tile Y, and
whether an unrelated `DrawImage` was interleaved at this SAME edge: several superficially
irrelevant changes flip the hole, including a faithful replay of `hole.go`'s own recipe later in
the same process. The gap is real (`hole.go` reproduces it every time it is run as written) but
is order/state-sensitive, not a stable function of the tie alone — whether shipped `drawFlat`'s
very different draw history hits it at any state is **not established**; FR-1 only bounds the
seam if it occurs.

```
sc10.go (real camera):  5 states, 0/5 show a hole (incl. both z=1.2 cases)
hole.go (synthetic):   35 states, 2/35 show a hole (exactly the two z=1.2 cases)
edges[2] at z=1.2,X=0.25: sc10.go=0x42990000 hole.go=0x42990000 drawFlat(recordingTarget)=0x42990000
diag.go, same z=1.2,X=0.25 edge, canvas/order varied (scratch, not kept evidence):
  h40 sy4  shared-src  img-then-tri (hole.go's own recipe, replayed later)   -- no hole
  h96 sy10 shared-src  img-then-tri                                         -- no hole
  h40 sy4  shared-src  tri-then-img                                         -- no hole
  h96 sy10 fresh-src   tri-then-img                                         -- no hole
  h40 sy4  shared-src  triangles only, no DrawImage interleaved             -- HOLE
  h96 sy10 shared-src  triangles only, no DrawImage interleaved             -- no hole
`vector.DrawFilledRect` at the tying boundary: no internal gap in either probe -- the overlay
  rects do not share the tie, matching DD-4's claim.
```

**AC-11 — the real front-end.** `againrom.exe`, mouse+keyboard driven, screenshots in
`ac11/`(scratchpad): clicking NEW GAME opens the picker, `SELECT A MAP maps 1-25 of 38` (matches
the corpus count); Enter opens `Beast.ALM` **displaced and lit** — a snow-capped mountain range
with directional relief shading, not a flat 32px staircase; Esc returns to the picker with the
process still alive (`HasExited=False`), matching "Esc unwinds one screen at a time." **Not
confirmed live:** synthetic pan/zoom (`keybd_event` with the extended-key flag, then `SendInput`,
then a synthetic wheel event) produced **no visible movement** in either `againrom` or standalone
`mapview` across several attempts — likely an input-injection/focus issue in this environment,
not a claim about the app. The "stays lit while panning/zooming" property therefore rests on
`TestCornerScalesMatchesShadeScaleOfCornerLevels` and P-4's tests (corner scales take no camera
state at all), not on a live pixel observation.

## Honest notes

- **The one-pixel unpainted column is a shipped defect of 0013**, newly shared by flat mode, not
  introduced or fixed here (spec Constraints, FR-1).
- **The window's Gouraud-interpolated interior and the CPU bilinear interior are not claimed
  pixel-equal** (spec Constraints) — nothing above claims otherwise.
- **The repo moved one commit during this session**: `bfbd706` (owner, `chore(sdd): raise the
  tight doc ceilings by 1 KB each`, unpushed) landed on local `master` mid-verification, raising
  `verification.md`'s own prose ceiling from 8192 to 9216 B. It touches only
  `scripts/check-doc-budget.sh`; every gate above was re-run after it landed.
- AC-10's computed comparison is a **mean-brightness proxy**, not a pixel read of the window
  (impossible per SC-8); its ratio metric does not model `ShadeChannel`'s 255 clamp, which is the
  best explanation found for its residual, sub-level disagreements.

## Not run

- A live pixel diff of the window against the PNG (AC-10's strong form is computed from exported
  production code, not from the window's own raster — see above).
- Automated pan/zoom inside the real front-end (input injection did not register — see AC-11).
- Every one of the 38 maps' corner-scale corpus was compared computationally (AC-10); only two
  (`Kids`, `61.alm`) were opened and screenshotted by eye.
- **AC-13, both clauses, both front-ends: not run.** Synthetic input injection is off-limits here
  (AC-11's `keybd_event`/`SendInput` attempt above showed no effect and is not a manual stand-in).
  Pending the owner's eye against `builds/0014-corner-interpolated-lighting/`. Open: cursor-follows-
  while-held, stop-on-release, and release OUTSIDE the window specifically.
- A wheel-reachable non-integral zoom (e.g. 1.2, `WheelZoomStep`) is not driven by any drag
  subtest; the zooms exercised are 1, 2 and 0.5.
