# Verification — 0058 hit test lift

Toolchain `go 1.26.1`, Windows 11. Every fixture is synthetic; nothing below read a game install.

## The gate

```
(go build ./... && go vet ./... && go test -trimpath -count=1 ./... \
  && sh scripts/check-no-game-assets.sh && sh scripts/check-doc-budget.sh \
  && sh scripts/check-sdd-audit.sh && test -z "$(gofmt -l $(git ls-files '*.go'))")
EXIT=0
check-sdd-audit: ok (36 note(s)/warning(s), none enforced)
```

The FAIL set is **empty**, which is the baseline this branch started from. The note/warning count is
not comparable from a worktree — `builds/` is untracked, so every story that ships one is missing
here — and is recorded only so the FAIL set can be read beside it. 459 tests pass across `pkg/ui`,
`pkg/render/terrain` and `pkg/render/camera`.

## What actually caused the offset, and how much each part contributes

The two candidates were separated and measured (`terrain.Project` and `terrain.StaticAnchor` driven
directly over synthetic grids):

```
(a) relief lift, a plain rising to altitude 64: MinV=0
    cell(0,0) AnchorHeight=  0  drawn    0 world px from the flat lattice = 0.00 cells
    cell(2,2) AnchorHeight= 40  drawn  -40 world px from the flat lattice = 1.25 cells
    cell(4,4) AnchorHeight= 64  drawn  -64 world px from the flat lattice = 2.00 cells
    the same grid raised uniformly to 100: MinV=-100, gap at (4,4) = 0 — the error vanishes

(b) sprite anchor, canvas 128x128 centre (64,78) — the shipped values TERR-SPR-040 records:
    frame 64x 32: anchorY= 30, body top 14 px above the footprint top (0.44 cells)
    frame 64x 64: anchorY= 46, body top 30 px above the footprint top (0.94 cells)
    frame 64x 96: anchorY= 62, body top 46 px above the footprint top (1.44 cells)
    frame 64x128: anchorY= 78, body top 62 px above the footprint top (1.94 cells)
```

**The relief lift is the whole of the draw-versus-pick disagreement, and it is exactly the owner's
"two cells".** At altitude 64 the drawing places a cell 64 world pixels — 2.00 cells — above where
the pick looked for it. It is 0 on flat ground *and* 0 on uniformly raised ground, because the
projection's origin cancels a typical cell's own height: that is why the defect survived every flat
map and every test.

**The sprite anchor is not a disagreement and is not fixed here.** Both the rim and the footprint are
placed from the cell; the body simply stands above them, by `anchorY − 16` world pixels — between
0.44 and 1.94 cells over the frame heights a 128×128 canvas can carry. It decides where a user
*aiming at a body* is actually pointing, and it pushes the same way the lift does, which is why the
two compounded on screen. After this change a box around a unit **as it stands on the ground**
catches it, because the footprint is where its feet are; a box around a head alone does not, and
would not have before. That consequence is disclosed in `spec.md` FR-1 and is the named seam.

## Do the ledgers settle the original's unit hit test? No — and it is authored

Swept `research/claims/` and `research/formats/` for a row about resolving a click or a drag to a
unit. **There is none.** Stated positively rather than by omission.

What is published, and how it was used:

- `TERR-SPR-039` (High) — the placement lift is the four-corner mean, a *different* grid from the
  one the terrain is drawn on, measured differing on ~91 % of sloped cells by −73…+55 rows. **Used**,
  and it decides which surface the lattice rides.
- `TERR-SPR-038` (High) — the altitude is subtracted, so higher ground draws upward. **Used.**
- `TERR-GEOM-036` (d) (High) — the engine's *cell* picker resolves through the projected mesh.
  **Cited in the first draft of `provenance.md` as backing FR-1, and withdrawn**: it is about the
  ground pick, which this story leaves alone, and it resolves through the terrain's corner mesh,
  which is the surface the lattice argues against.
- `TERR-SPR-041` — the object cull feeds a **sprite rectangle's** row range into that picker. Not a
  mouse hit test, so it settles nothing; but it is the only published place a screen rectangle
  stands for a unit, and it points at the sprite. **FR-1 authors against it, deliberately and out
  loud.**

So FR-1 is **AUTHORED** — a verdict, not a hold — on the owner's rule ("what you box is what you
get") plus two structural reasons: an entity that resolves no frame has no sprite rectangle at all,
and overlapping bodies would need a depth rule nothing in the tree has.

## Where the single shared placement lives

- `pkg/render/terrain.CellFootprint` — the cell's own rectangle, clipped, by value. Renamed from
  `BlockedCellRect`, whose own comment already said it was "the same rectangle
  `SelectionMarkerRects` draws the border of" while the two computed it apart.
- `pkg/render/terrain.footprintRimRects` — the border of that rectangle; the selection rim at two
  native pixels and `CellGridRects` at one.
- `pkg/ui.(*Viewer).cellScreenRect` — footprint + displacement + `placeArm` (the per-cell lift, the
  camera, the view cull). Called by `entityPickRect` and by `gridScreenRects`.

**The narrower claim, which is the true one.** The rim reaches the same two pieces through the glyph
path it already rides — four strips cannot be one rectangle — so what is shared is the **footprint**
and the **transform**, not a single call. `spec.md` FR-2 is worded to that after an adversarial read
found the wider claim overstated. The lift expression appears **once** in the tree, in `placeArm`.

## Evidence per criterion

| AC / P | Evidence |
|---|---|
| AC-1, AC-2 | `TestABoxTakesTheUnitWhereItIsDrawn` — cliff fixture, cell (1,1) drawn at world Y 0 against a flat lattice position of 32, the two disjoint. A box over the rim selects it; a box over the flat cell selects nothing; a second unit on the cell the flat lattice named is **not** returned. The fixture fails loudly if those numbers ever coincide. |
| AC-3 | The whole pre-existing pick suite passes **unedited** — `command_test.go`, `deadselect_test.go`, `marquee_test.go`, `selcount_test.go` and `pkg/game`'s two camera-band tests. No expectation was adjusted. |
| AC-3a, P-3 | `TestOnFlatGroundOnlyAMidStepUnitMoves` — standing, the target is the flat cell; mid-step it is the drawn position. |
| AC-4 | `TestATapAndAOnePointBoxAskOneQuestion` — swept at stride 3 over the whole view plus an 8 px margin, on a fixture that **contains** the overlap, and failing if the sweep never meets one. |
| AC-5 | `TestTheLowerIdWinsWhereTwoPlacedRectsHoldOnePoint` — cells (0,0) and (0,1) coincide on the cliff face; both snapshot orders return id 6. |
| AC-6 | `TestABoxIsOrientationIndependentOnRelief`, all three corner swaps. |
| AC-7 | `TestThePickCarriesTheEntitysOwnDisplacement`. |
| AC-8 | Existing `TestABoxTakesEveryCoveredUnitThatIsNotDead`, `TestATapTakesTheLivingAndNeverTheDead`, unedited. |
| AC-9, P-2 | `TestThePickRectIsTheRimRect` — the rim's strips' span equals the hit target on four cells. |
| AC-10 | `TestTheLatticeCoversTheTerrainPassBand`, displaced and flat, against the terrain pass's own walk **after** the view cull. |
| AC-11 | `TestTheLatticeIsOffByDefaultAndTheSliceIsUnchanged`, over a frame already holding four passes. |
| AC-12 | `TestTheGridKeyTogglesOncePerPress`; `TestTheGridSetterMovesNothingElse`. |
| AC-13, FR-10 | `TestTheOrderDestinationStillResolvesTheFlatGround` — the order names the camera's own flat inverse. |
| AC-14 | `TestTheLatticeStopsBelowItsLegibilityFloor`, including a non-finite zoom. |
| P-1 | `TestTwoEntitiesOnOneCellSharePickRect`. |
| P-4 | `picked` and `cellScreenRect` take and return values and write nothing; the selection is still replaced only by `command`'s single store. `internal/archtest`'s import check keeps `pkg/ui` off `pkg/sim`, and no file this story touched names a simulation type. |
| Error cases | `TestAnEmptyOrInvertedHitTargetIsNeverHit` — zero width, zero height, inverted, NaN. |

| SC | Met by |
|---|---|
| SC-1 | The AC-1..AC-8 rows above, all on the cliff fixture, none opening a window. |
| SC-2 | `TestATapAndAOnePointBoxAskOneQuestion`, whose fixture contains the overlap and which fails if it ever stops containing it. |
| SC-3 | `TestThePickRectIsTheRimRect`, comparing the two paths' output rather than the source. |
| SC-4 | The pre-existing pick suite unedited, plus `TestOnFlatGroundOnlyAMidStepUnitMoves` for the carve-out. |
| SC-5 | The five lattice tests; AC-11's is over a frame already holding four passes. |
| SC-6 | `TestTheOrderDestinationStillResolvesTheFlatGround`. |

## The lattice's cost, measured

```
144x144 map (20736 cells) in a 1280x720 window:
    zoom 1      cell = 32.0 screen px  band   920 cells  strips drawn  3378
    zoom 0.5    cell = 16.0 screen px  band  3600 cells  strips drawn 13979
    zoom 0.125  cell =  4.0 screen px  band 20736 cells  strips drawn     0   (below the floor)
```

Timed before the floor existed: 146 µs/frame at zoom 1, 618 µs at zoom 0.5, and **3.5 ms with 82 944
strips and 15.6 MB allocated at `ZoomMin`** — the whole map, because the band in cells grows as the
inverse square of the zoom. The plan's original claim that the cost was "bounded by the view, not the
map" was **false at low zoom**; the floor is what makes it true, and it was chosen on legibility (a
four-pixel cell cannot carry a readable outline) rather than on the timing.

## Limitations, disclosed

1. **The ground pick is unchanged, and the two halves have stopped being wrong together.** Before,
   aiming high selected the right unit *and* ordered it to the right cell. Now the selection is right
   and the destination is still off by the relief. Same defect class, different question, and it
   needs an inverse of the drawn mesh. Named as a seam (spec FR-10, out of scope).
2. **A box can no longer reach past the window edge.** The old path clipped to the tile grid; the new
   one carries `placeArm`'s view cull. A drag whose cursor leaves the window no longer selects cells
   the view does not reach — a narrowing, and the direct consequence of "what you box is what you
   get". Not covered by a test, because no fixture in the tree constructs a cursor outside the
   window.
3. **Boundary rounding at non-dyadic zoom is not bounded.** The old world-space band test expands to
   exactly the new comparison in exact arithmetic, but the rounding differs, and production zoom is
   `1.2^n`. The suite runs at zoom 1 and 2, both exact, so it verifies the algebra and not this. A
   one-ULP flip at a cell boundary is accepted.
4. **On a slope the lattice does not sit exactly on the terrain's drawn quad**, by design (DD-8).
5. **A mid-step unit is picked between two outlines and coincides with neither.** The lattice has no
   entity and cannot shift; the hit target does. Stated in P-2.
6. `camera.ScreenToCellRange` now has **no production caller**; two `pkg/game` tests still use it. It
   is kept and re-documented rather than deleted, which is a judgement recorded here so it can be
   revisited.

## What the adversarial plan read changed

The plan gate ran from a structurally independent context holding only the artifacts and the tree.
It returned sixteen objections; five were material and all five are landed as T4. Recorded because
each was invisible to the suite as it then stood:

1. **FR-5 was a false contract.** The tap reduces to the lowest id and the box does not, so where two
   targets hold one point they differ — and the AC-4 sweep had been built on a fixture whose units
   never overlap, seventy lines above a test that deliberately constructs the overlap. The claim was
   inherited from 0030's comment, was already false there, and this story had written a test that
   could not see it.
2. **`meets` had an unstated precondition.** Correct only for a rectangle with area, and `placeArm`
   scales its extent by the camera's **raw** `Zoom` while taking its position through the **clamped**
   one — so an empty target was hittable and an inverted one was hit from its left.
3. **AC-3 and P-3 were false.** The entity displacement is not gated on displaced mode, so a mid-step
   unit on flat ground moved. `grep` for `SetPhase` and `Step:` in the pick suite returns nothing:
   "the existing suite passes unchanged" was **vacuous** over the one case that changed.
4. **R-2's cost bound was wrong**, and AC-10 was unsatisfiable as written because the displaced band
   is a documented superset of what is visible.
5. **`provenance.md` graded a High row for a clause it does not support** — see above.

The remaining objections were narrowing rather than defects and are reflected in FR-2's wording, in
`decide`'s doc comment, and in the limitations above.
