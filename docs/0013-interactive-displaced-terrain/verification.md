# Verification — interactive displaced terrain

Story commits: `9597ac2` (T1), `ec8998e` (T2), `f7b4cee` (T3), `b5c0d3b` (T4). Evidence below was
run against `builds/0013-interactive-displaced-terrain/`, built from `b5c0d3b`.

## Gates

```
$ go build ./...                          (clean)
$ go vet ./...                            (clean)
$ gofmt -l $(git ls-files '*.go')         (no output)
$ sh scripts/check-doc-budget.sh          exit 0
$ go test ./...
ok  againrom/cmd/againrom 0.979s      ok  againrom/pkg/formats/alm 0.342s
ok  againrom/cmd/mapview  0.906s      ok  againrom/pkg/formats/reg 0.422s
ok  againrom/cmd/regtool  0.353s      ok  againrom/pkg/formats/res 0.443s
ok  againrom/cmd/terraintool 0.947s   ok  againrom/pkg/formats/spr256 0.327s
ok  againrom/internal/archtest 0.396s ok  againrom/pkg/game 0.788s
ok  againrom/internal/notices 0.351s  ok  againrom/pkg/render/camera 0.348s
ok  againrom/internal/synth 0.349s    ok  againrom/pkg/render/frame 0.481s
ok  againrom/pkg/render/menu 0.488s   ok  againrom/pkg/render/terrain 0.556s
ok  againrom/pkg/ui 1.075s
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 8f2db83..HEAD
0013-interactive-displaced-terrain/T4   T3   T2   T1     (one each, none twice)
```

The `pkg/render/terrain` Defender false positive recorded as a plan risk **did not occur** in any run
above; every `pkg/render/terrain` line is a genuine pass with no flag perturbation.

## Unit evidence

```
pkg/render/terrain  TestProjectWorldCorner              SC-4  AC-4
                    TestProjectRowRange                 SC-5  AC-6 AC-6a
                    TestProjectedEqualsFlat  TestLevelGridFlat*      FR-1
pkg/render/camera   TestNewWorldHeightIsTheTileGrid           SC-1
                    TestSetWorldHeight                        FR-3
                    TestZoomAboutCursorOverDisplacedWorld     AC-5
                    TestRandomizedDisplacedWorldKeepsInvariants AC-5
pkg/ui              TestModeFlatInputs                        SC-2  AC-1
                    TestModeDisplacedIsChosenByValidityNotByRelief   AC-2
                    TestModeFollowsOverlaysEnabledAfterConstruction  AC-2a
                    TestShortDisplacedWorldCentresRatherThanClamps   SC-3  AC-3
                    TestFlatTileScreenIsTheUndisplacedLattice        FR-1  P-2
                    TestTileVerticesCorners  TestTileVerticesAreLiteralScreenPixels  SC-4
                    TestDisplacedSetCoversEveryIntersectingQuad      SC-5  P-3
                    TestDisplacedSetStaysInsideTheTwoSidedBound      AC-6a
                    TestDisplacedLoopVisitsRowMajor                 SC-9
                    TestResolvedCellIsTheSameInBothModes            P-4
                    TestDisplacedGeometryDoesNotMutateTheAltitudes  SC-7  P-5
pkg/game            TestLoadMapViewer/the_map's_altitudes_reach_the_viewer_and_displace_its_world
                    TestLoadMapViewer/a_map_with_no_relief_still_displaces,_at_the_flat_height
```

All windowless (SC-6): no test opens a graphics context, and `pkg/ui`'s vertex tests read
`tileVertices` output as struct values.

**Every assertion was checked against a deliberate break**, and each break failed the test it was
meant to fail. The list, all reverted: `syncWorld` dropped from `NewViewer`; dropped from both
overlay setters; using the flat height while displaced; `Mode()` ignoring the overlay flags;
`validAltitudes` always true; `mapload` dropping the altitudes; `SetWorldHeight` not re-clamping; the
index list `{0,1,2, 1,3,2}`; TR/BL transposed; the zero-value vertex colour; the source span
collapsed and its axes swapped; the `MinV` translation dropped from the window; column-major visit
order; "draw the whole map every frame"; and **DD-2's rejected composition** — the camera's band
padded by `ceil(down/32)+1` / `ceil(up/32)+1` — which under-covers on the cliff fixture and on the
1x8 ditch, reproducing the plan's own counterexample.

One test of mine was initially non-discriminating and is recorded as such: the two-sided-bound test
first recomputed `RowRange` instead of reading the rows the draw loop visits, so "draw the whole map"
passed it. It now reads the loop, and that break fails.

**Not covered, by construction:** that `Draw` dispatches on `Mode()` at all. `Draw` needs a graphics
context, which golden rule 2 forbids in tests. Everything below the dispatch is covered headlessly;
the dispatch itself is covered only by the developer runs below.

## Developer-run evidence

Install: the lawful GOG install on this machine, reached via `-assets`. Corpus: the 38 shipped maps
— 10 loose `.alm` at the install root, 28 inside `scenario.res`, extracted with `restool extract`
to a scratch directory outside the repository. The picker's own count agrees: `38 map rows`.

**AC-7, SC-8 — the summaries did not move.** 38 maps x {none, `-objects`, `-units`, both} = **152
`mapview -check` comparisons against a binary built from `d9ad2be`, the last commit before T1: 0
differing**, and all 152 are real summaries (an earlier sweep passed vacuously by comparing
identical "file not found" errors on both sides — `-map` takes a filesystem path, not an archive
name; that run is discarded and not counted here). `againrom -check` is byte-identical too.

```
$ mapview -assets "$INSTALL" -map "$INSTALL/Kids.alm" -check
mapview: Kids Paradise 80x80 cells (6400), tile slots 52/128, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)
$ mapview -assets "$INSTALL" -map "$CORPUS/61.alm" -check -objects -units
mapview: 61.alm 80x80 cells (6400), tile slots 52/128, objects 27, units 43, water speed 4 (16 tps, 62 ms/tick, 992 ms/cycle)
```

**AC-8 — the whole corpus opens and draws.** Each of the 38 maps was opened in the windowed viewer
with no overlay, left drawing for 2.5 s, then killed: **38 of 38 survived with no early exit, no
panic and empty stderr.** `Islands` and the other four 256x256 maps included.

**AC-8 — `Kids` and `61` against the relief the PNG story recorded.** Displaced/flat pairs were
captured off the live window (`-objects` or `-units` is the flat side, since either forces flat).
`Kids`: the flat shot's top edge is straight, the displaced shot's is serrated — only the single
highest vertex reaches world Y 0 — and the riverbanks move by more than a rigid translation. `61`:
the ground's diagonal pattern and the rock patch outline are warped where the flat shot's are
regular. Both agree with the PNG story's numbers, which come from the same `Projection` the window
now uses: `Kids` 2560x2571 at `y origin -69`, `61` 2560x2648 at `-126`, against a flat 2560.

**AC-9 — the game front-end is displaced.** Driven through the real UI: click NEW GAME, the picker
lists `maps 1-25 of 38`, arrow keys and Enter open a map. The terrain is displaced — on `Forester`
the snowfield's upper boundary is a jagged mountain silhouette, not a 32-px staircase. Edge-scroll
pans into it, and Esc returns to the picker with the process still running and stderr empty. This is
the path DD-5 called out: it touches neither overlay setter, so it is displaced only because
`NewViewer` itself syncs the world.

## Fidelity limitations observed (AC-8)

- **Steep cells read as a vertical smear.** One 32x32 texture stretched across a quad up to ~130 px
  tall streaks each source row over several destination rows. Visible on `61` and on `Forester`'s
  snowfield. This is the disclaimed difference, not a defect: the engine resamples 32 source rows
  into each destination column's own span (`TERR-GEOM-032`…`034`), which this story deliberately does
  not transcribe. Whether the streak *matches* the original at a given zoom is unmeasured — the spec
  claims no pixel equality, and nothing here establishes one.
- **The world's top and bottom edges are serrated.** Correct — the canvas spans `[MinV, MaxV)`, so
  the extreme rows are reached by one vertex each. The engine hides its own outer ring behind an
  over-scan margin we do not reproduce (`TERR-EDGE-026`, open).
- **No shading in the window.** Unchanged by this story; the window has never applied 0007's
  lighting.

## Not run

- **Pixel comparison against the 0012 PNGs.** The `Kids`/`61` check above is a live displaced-vs-flat
  comparison plus numeric agreement with the PNG canvas, not a pixel diff against the PNG files.
  A pixel diff would fail by construction: the rasters differ by design.
- **Draw-call cost at 256x256 zoomed fully out.** The 256x256 maps open and draw, but no frame-time
  measurement was taken. The plan records this as a later story's finding.
- **Zoom by mouse wheel in the game**, driven only by unit tests (AC-5); the manual drive used
  edge-scroll panning.
