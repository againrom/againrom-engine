# 0102 — verification

Run on a clean tree at the story's last commit, research pin `53f8bb7`.

```
$ go build ./... && go vet ./... && gofmt -l $(git ls-files '*.go') && go test -trimpath -count=1 ./...
ok  againrom/cmd/againrom 1.751s ... ok againrom/pkg/render/terrain 0.812s ... ok againrom/pkg/ui 3.197s
EXIT=0     (31 packages ok, 3 with no test files, 0 failures, gofmt printed nothing)

$ bash scripts/check-no-game-assets.sh      check-no-game-assets: clean (tree scan)   EXIT=0
$ bash scripts/check-doc-budget.sh          EXIT=0
$ bash scripts/check-sdd-audit.sh           FAIL set empty
```

`check-sdd-audit`'s note and warning counts are not reported: a worktree has no `builds/`, so they
are not comparable to the orchestrator seat's. Only the FAIL set is.

## The four defects, measured on the game's own data

Measured through this tree's own loaders against the `en` install, mission 10, `scenario/10.alm`:
18 structure records, 160 strip placements. The structure anchored at cell **(18,61)** — the tower
the owner photographed — decodes as `TileWidth 2, TileHeight 2, FullHeight 3, ShadowY 40`, the class
`analysis.md` names. Its silhouette at that anchor is 64 image rows over 2 strips.

| | max deviation from one straight line | worst step between adjacent world rows | distinct columns over the 64 rows |
|---|---|---|---|
| 0101 | 17.898 px | 19.000 px, at world y 1952 | **2** — one per strip |
| 0102 | **0.525 px** | 1.102 px, at world y 1952 | 38 |

**The cut is gone.** The 19 px step at y=1952 is the strip join — the sharp vertical cut in the
photograph — and it falls to 1.102 px, of which 0.577 is the lean the silhouette is supposed to have
over one row and 0.525 is the residue of two truncations. FR-9's "under 1 px" is that residue, and
it is 0.525.

**A structure shadow is one leaning silhouette.** Under 0101 the 64 image rows took **two** distinct
columns: each strip was internally flat, which is what a stack of slabs is. Under 0102 they take 38,
every one within 0.525 px of a single straight line in world `y` — one sheared silhouette rather
than a stack.

**The lean never passes through zero.** Over all 1440 minutes of the cycle,
`|ShadowSlope(SunAngle(m,true))|` lies in `[0.033345685, 0.577350257]` with **0** exact zeros. It
changes sign twice, once per half-day, and both changes are the jump across the dead band rather
than a crossing: at minutes 360 and 1080 — where `SunAngle` is exactly `0.000000000000` — the slope
is `+0.033345685`, not `0`.

**The fifteen suppressed classes.** The census matches the corpus exactly: 15 classes whose
`ShadowY` sits far outside their own art, 4 at 10000 and 11 at 20000. Over every minute of the day
and every strip of each class, the smallest displacement any of the fifteen ever takes is **330 px**;
for the eleven at `ShadowY = 20000` it is **660 px**, on class 52 at `FullHeight 6` — the corpus's
own published figure, reproduced from our reading of the dead band rather than fitted to it.

The load-bearing comparison is the counterfactual: **without the dead band, at minutes 360 and 1080
the slope is exactly `+0.000000000` and the largest displacement any of the fifteen takes is `0.0`
px.** All fifteen would land squarely on their own footprint twice a day. That is the regression the
clamp removes, and it is removed by arithmetic alone — no rule in the package tests `ShadowY`,
`FullHeight` or any other class field against a constant (P-4).

**What this does not establish.** That 330 or 660 px puts a shadow off *screen* is a camera
question, not a shadow one: this tree's window is 1024 px wide. `TERR-SHDW-136`(d) asserts those
displacements are "outside every viewport width the engine opens" and cites `MISSION-VIEW-020` for
it, but that row is about a persisted view *position* and carries no viewport width. What is
verified here is the displacement — never small, never zero, four of the fifteen only ever reaching
330 px — and not the claim that the original's viewport is narrower than that. It changes nothing we
build: the rule is the arithmetic either way.

## AC and P, one line each

- **AC-1** `TestShadowSlopeSweepAC1` — the exact value at `DefaultTheta`, the 1440-minute sweep, and
  `+0.033345685` at minutes 360 and 1080. Dropping the clamp from `ShadowAngle` was applied and the
  test went red from minute 338; restored, green.
- **AC-2** `TestShadowPivotShiftAC2` — differs across frame heights and across anchors, zero only at
  `2*(frameH/2) == anchorY`, and an odd `frameH` takes `frameH-1`'s value. A shift computed from the
  angle alone was applied and failed this and AC-3.
- **AC-3** `TestUnitShadowPlaceAC3`, `TestUnitShadowComposedLineAC3` — a sweep of 5 frame heights by
  5 anchors by 10 minutes against `body.TopLeft.X + slope*(anchorY-r)` within 2 px, the residue not
  growing with the frame height. `ShadowPivotRow` reverted to a constant `0` failed it.
- **AC-4** `TestStructureShadowCompositionAC4` — the hand-built strip list, the composed column a
  straight line to within 1 px, and the ~18.4 px join the per-strip term alone leaves. Submitting
  `Slope: 0` for structures failed it. The line is compared as **one** truncation of the combined
  term: the composed value is two independent truncations summed, so against the raw float the
  bound would be 2 px rather than the 1 px the contract states.
- **AC-5** `TestStructureShadowPlaceGuardsAC5` — `VariableSize`, no class, no frame, non-positive
  `TileWidth`.
- **AC-6** `TestStructureShadowShiftSuppressionAC6` for the numeric half, and the measurement above
  on the real 15 classes. The source half — that no function tests a class field against a constant
  — is a reading of `shadow.go` recorded here rather than made into a scan: the file's only
  comparisons are the nil and non-positive guards `StructureShadowPlace` and `ShadowPivotRow`
  already declare. No sentinel test was added, per plan DD-7.
- **AC-7** `TestObjectShadowPlaceAC7` — frame-0 anchoring and frame-0 displacement, the drawn
  frame's own silhouette and pivot, and the three refusals.
- **AC-8** `TestShadowDrawsCarrySlopeAndPivotForAllThreeCasters`,
  `TestDrawShadowsMirrorLeansTheSameWayAsUnmirrored`,
  `TestShadowDrawsCullsTheStructureOnItsWidenedRectangleNotThePlainOne`. Passing `0` instead of
  `slope` in the structure arm turned the first red; putting the old `Scale(-zoom,zoom)` /
  `Translate(d.X+d.W, d.Y)` pair back into `shadowGeoM` flipped the mirrored lean's sign (`u=2.5`,
  `m=-2.5`) and turned the second red. Both restored green.
  **AC-8's third clause is stated in the wrong direction and cannot be witnessed as written.** It
  asks for a placement whose sheared rectangle leaves the view while its unsheared one would have
  been kept. Under FR-4's pivot the bottom row's offset is `int(slope*1)`, which is `0` because
  `|slope| < 1` at every minute (AC-1), so `shadowWiden` is provably a **superset** of the plain
  rectangle on every call and can only ever keep more. The test witnesses the true direction — that
  the widened rectangle now gates the structure and unit arms too, which is FR-16's own content —
  and the clause is left standing with this note rather than quietly reinterpreted.
- **AC-9** `TestGeometryTotalityAC9` for the first half: nil frame, nil class, negative frame size,
  negative grid row, `theta` of `1e9` and `-1e9`, over every function this story names. For the
  second half, `againrom -check` and `missionrun -mission 10 -census` were run on **both** roots
  with this story's five Go files reverted to `463a9cd` and again with them restored, and the four
  outputs are character-identical: `en` `38 map rows ...`, `ru` `34 map rows ...`, and on both
  `census: 2 of 36 unit(s) moved, 0 fell, over 40000 tick(s)`.
- **P-1** No `formatVersion` bump: `pkg/sim/binary.go` is untouched and stays at 21. Nothing this
  story changed is imported by `pkg/sim`; the two changed non-test files are
  `pkg/render/terrain/shadow.go` and `pkg/ui/shadow.go`. The identical `missionrun` census on both
  roots is the behavioural half.
- **P-2** `math.Tan` is called in exactly one place, `ShadowSlope`, on the angle `Light.Theta`
  already carries. No second sun is computed in either file.
- **P-3** `TestGeometryTotalityAC9`, as above.
- **P-4** Read, not scanned — see AC-6.

## What the story did not settle

- **The object shadow's anchor frame.** `TERR-SPR-043` puts it at frame 0, `TERR-SHDW-130` at the
  drawn frame while citing `TERR-SPR-043` as agreeing. `provenance.md` records it; spec D-3 keeps
  frame 0. 8 of 66 classes, at most 4 columns and 8 rows. A question for the pipeline, unchanged by
  this story.
- **The dead band's shape** stays a disclosed divergence (spec D-4). The published displacement does
  not choose between the two readings; two weaker arguments do, and AC-1 pins the choice we made.
- **Two `pkg/ui` fixture constants re-spell the production expression.**
  `planeStructureShadowShift` and `planeUnitShadowShift` in `structures_test.go` are now computed
  from the same `terrain` functions the pass calls, because the value is no longer an integer and a
  typed literal would drift. Those two assertions therefore witness the pass's **composition** and
  its draw order, not the value of the law; the law's own witnesses are the `shadow_test.go` tests
  above, each checked by reverting it.
