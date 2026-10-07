# Plan — height-projected placements

## Design decisions

### DD-1 — `AnchorHeight` clamps into the cell range before `col+1`/`row+1` are formed

`Projection.AnchorHeight(col, row int) int` clamps `col` into `[0, Width-1]` and `row` into
`[0, Height-1]`, with the same min-then-max order `Altitude` already uses, before forming the four
corner reads `Altitude(col,row)`, `Altitude(col+1,row)`, `Altitude(col,row+1)`,
`Altitude(col+1,row+1)` — never after. `Altitude` clamps each index on its own, but only once it
reaches it: adding 1 to an unclamped `col` at `math.MaxInt` wraps to a negative number first, which
`Altitude`'s clamp then reads as column 0 — the wrong edge, silently. The pre-clamp keeps `col+1` an
ordinary int (at most `Width`) that can never wrap.

Rejected: passing the caller's raw `col`/`row` straight into the four `Altitude` calls and trusting
its own clamp. Correct everywhere except the one input the spec names — an anchor at the integer
extreme — where it silently returns the near edge's height for the far one.

The sum of the four divides by Go's native `/4`, truncating toward zero for a negative sum exactly as
the height lookup contract states; nothing here decides that tie, it is inherited arithmetic.

### DD-2 — the `cmd/terraintool` seam: two new exported functions, a lift reaching only the glyph

`DrawObjectMarkersAtHeights`/`DrawUnitMarkersAtHeights` add one parameter to their existing `*At`
siblings' shape: `liftY func(col, row int) int`, the output-pixel shift already carrying its sign and
its scale multiply. It reaches the unexported `markerRects` through a new `liftY int` parameter added
ONLY to `cy` — `cy := row*cellpx + cellpx/2 + offsetY + liftY` — never to `mapRect`, which stays
`image.Rect(0, offsetY, cols*cellpx, rows*cellpx+offsetY)` exactly as before. The stage-1 map-extent
clip therefore cannot move with a marker's own height (FR-4); the stage-2 clip — `img.Bounds()` inside
`drawMarkers`, unchanged — still runs after `markerRects` has baked the lift into `cy`, so it clips the
already-lifted glyph and satisfies FR-6 without a second code path.

Rejected: folding the lift into `offsetY` — the trap FR-4 names outright, since that same value also
places `mapRect`. Rejected: widening the EXISTING exported `DrawObjectMarkersAt`/`DrawUnitMarkersAt` —
every 0008/0009/0012 call site and test would grow a parameter it never uses, for behaviour only this
one caller needs; the two new functions cost nothing there.

### DD-3 — `pkg/ui` translates the rectangles `ObjectMarkerRects`/`UnitMarkerRects` already return

`overlayScreenRects` shifts each arm the glyph builder returns — `arm.Add(image.Pt(0, dy))`,
`dy = -v.proj.AnchorHeight(cell.X, cell.Y) - v.proj.MinV` — before it reaches `WorldToScreen`, and only
when **`v.Mode() == ModeDisplaced`** — the mode itself, never the weaker `v.proj != nil`. Those two
were the same predicate under DD-5 alone, and DD-7 separated them: a projection stays non-nil under
`SetFlat(true)`, so guarding on it would lift markers over deliberately flat terrain and break AC-10.
Guarding on the mode is what makes the overlay and the terrain under it unable to disagree. This is the
spec's own selected
alternative — translate the built rectangles, not widen the two render-tier builders — so
`terrain.ObjectMarkerRects`/`UnitMarkerRects` gain nothing.

Every fixture in the existing `overlay_test.go`/`unit_overlay_test.go` builds its viewer over the
plain `grid()` helper, which sets no altitude layer — `v.proj` is nil there, so none of them reaches
this branch at all. That is why neither file needs an edit: the new path is simply unreached by
anything they construct, not exempted by hand.

### DD-4 — `cmd/terraintool` recomputes its own `Projection`; `Render` gains no field

`terraintool` calls `terrain.Project(m.Altitudes, m.Width, m.Height)` itself, once, only when composing
a non-`-flat` render with `-objects`/`-units`. `Render` is shared by the two FLAT compositors, which
have no projection at all — widening it with a `*Projection` field would leave two of its four
constructors carrying a permanently-nil value, to save a caller a recomputation of data
(`m.Altitudes`/`Width`/`Height`) it already holds in full. The cost is one extra O((W+1)x(H+1)) walk,
once per invocation of a tool that renders once and exits — set against the whole-image composite it
already pays, not the per-frame cost 0014 DD-1 weighed for the window.

### DD-5 — `Mode()` drops its overlay clause outright; `mode_test.go` is the one edited test

`v.proj != nil` becomes the whole of what remains of the predicate (the spec's own selected
alternative); DD-7 then adds the deliberate `!v.flat` term. The one test this
falsifies byte-for-byte is `TestModeFollowsOverlaysEnabledAfterConstruction`, whose own name and body
assert exactly the rule FR-2 deletes — it is rewritten IN PLACE, in `mode_test.go`, to assert the
supersession instead: toggling either or both overlays, in any order, leaves `Mode()` and `WorldH()`
unchanged over a valid altitude grid. Nothing else in the repo asserts the superseded rule, so
`pkg/ui/mode_test.go` is the only existing test file this story touches, anywhere.

### DD-6 — nothing else moves

Draw order (terrain, objects, units; later wins), the stage-1/stage-2 clip split, the camera view-cull
test in `overlayScreenRects`, `cornerScales`/lighting, and every `-flat`/`-unshaded` path are untouched.
A marker's rectangles move vertically and only vertically, by a value that is zero whenever `v.proj`
(pkg/ui) or the composed geometry (`-flat`, terraintool) says there is no relief to place them on.

**The two seams are not symmetric, and that is the decision, not an oversight.** In `terraintool` the
lift enters `cy` *inside* `markerRects`, so the stage-1 map-extent clip runs **after** it and can
truncate an over-lifted arm at the flat map's boundary — which is what AC-7 pins. In `pkg/ui` the arms
come back from `ObjectMarkerRects`/`UnitMarkerRects` already clipped at a zero lift, and `dy` is added
afterwards, so **no map-extent cap ever sees the lift there**; the only clip that does is the camera
view-cull. That is right for a scrolling world, whose markers are bounded by what the camera shows and
not by a canvas — but it means the two front-ends can disagree at a map edge, deliberately.

### DD-7 — flat becomes something you ask for, not something you get by accident

`Viewer` gains a `flat bool` and `SetFlat(bool)`; `Mode()` is `v.proj != nil && !v.flat`, and
`cmd/mapview` wires `-flat` to it. DD-5 removes one term from that predicate and this adds another,
which is the point: the term it removes was a **side effect** of asking for an overlay, and the term
it adds is a request. Rejected: leaving the overlay clause as the way to reach flat — it is the defect
this story exists to remove, and it had already coupled three lighting tests to a rule they never
assert. Also rejected: a test-only hook, which would leave the shipped flat-lit path unreachable while
pretending it was covered.

The reachability is not cosmetic. `NewViewer` builds the projection **and** the level grid inside one
`if validAltitudes(g)`, so without DD-7 the predicate `v.proj != nil` makes flat mode and lighting
mutually exclusive, and 0014's flat-path corner colours become code no front-end can reach.

## Self-checks

- **SC-1** `AnchorHeight` matches a hand-computed truncating mean of the four corner altitudes on
  synthetic flat, sloped and negative grids, for an interior, an edge, a corner and a past-`Width`/
  `Height` anchor — and a fixture built by adding 1 to an UNCLAMPED `col` at `math.MaxInt` (DD-1's
  named trap) disagrees with the clamped answer, so clamping after the `+1` fails it.
- **SC-2** A borrowed altitude slice is byte-compared after many `AnchorHeight` calls over every grid
  above; unchanged (P-6).
- **SC-3** Over a valid altitude grid, `Mode()` reports Displaced for all four `showObjects`/
  `showUnits` combinations; over an invalid one it reports Flat regardless of them (AC-3/AC-4). A
  version that still consults either flag fails at least one combination.
- **SC-4** Exactly **four** existing test files change — three because they depend on a rule this
  story supersedes, one because it pins the surface this story extends:
  - `pkg/ui/mode_test.go`'s `TestModeFollowsOverlaysEnabledAfterConstruction` (0013 FR-2 — an overlay
    forces flat), which **asserts** it;
  - `cmd/terraintool/main_test.go`'s `TestRenderMarkersAtProjectedOrigin` (0012 FR-11/SC-14 — markers
    on the **un-displaced** lattice), which asserts it too: its sloped fixture puts an object anchor
    at `(0,0)` where `AnchorHeight` is `(20+20+0+0)/4 = 10`, so it must fail once T3 lands, and
    asserting otherwise would pin the defect this story removes;
  - `pkg/ui/light_test.go`, which **never asserts the rule and depends on it anyway** — two of its
    tests reach flat mode by switching an overlay on over a *lit* grid (`SetObjects(true, nil) //
    force flat mode`) purely as a fixture device, and a third encodes it in an expectation table.
    Measured, not predicted: dropping the clause turns exactly those three red. Their fixtures move to
    `SetFlat(true)` (DD-7), which is the same request stated honestly; no expectation about geometry
    or lighting changes.
  - `cmd/mapview/flagset_test.go`, which depends on no superseded rule at all — it pins the tool's
    **flag surface**, which this story *extends*. `usage` is compared to its `shippedUsage` byte for
    byte, so documenting `-flat` turns it red, while **leaving** `-flat` undocumented turns nothing
    red: its "the usage line names every flag the parser defines" subtest walks a hardcoded list, not
    the parser. Measured, not argued — `fs.SetOutput(io.Discard)` means `-h` prints nothing, so that
    one string is the only place a user ever reads the flag set. T7 moves both halves together.

  A diff that touches a **fifth** — `viewer_test.go`, `overlay_test.go`, `unit_overlay_test.go`,
  `input_test.go`, `project_test.go`, `cmd/mapview/main_test.go`, or any other
  `pkg/render/terrain`/`pkg/render/camera` test — fails this outright, even if every test passes.
  Every task otherwise carries its new assertions in a **new** test file, so the count stays checkable
  by `git show --numstat` and not by judgement. The count has now been wrong three times, always low:
  twice because a rule was depended on somewhere nobody thought to look, once because "depends on a
  rule this story supersedes" cannot name a file that pins an interface the story adds to. A criterion
  of this shape is only worth having if the number is measured rather than reasoned.
- **SC-5** For an object and a unit anchored in the same sloped cell, the displaced screen rects equal
  the flat rects translated by exactly `-AnchorHeight(col,row) - proj.MinV` world pixels applied
  BEFORE `WorldToScreen`; both kinds carry the identical offset, extent/count/colour unchanged (AC-5,
  FR-3, P-4). Shifting after the camera transform, or dropping `-proj.MinV`, fails it.
- **SC-6** Two markers of different `AnchorHeight` whose offset rectangles now overlap are still drawn
  objects, then units, the later winning — matching flat order (AC-6, FR-5).
- **SC-7** A `terraintool` compose of a sloped map with an anchor within an arm's reach of the map's
  top edge, whose lift exceeds that reach: the arm still clips at the UNLIFTED map extent (stage 1),
  and separately at `img.Bounds()` once lifted (stage 2) (AC-7). Folding the lift into `offsetY` moves
  the stage-1 clip with the marker and fails this case specifically. The mirror case runs too — a
  bottom-edge anchor over a **negative** `AnchorHeight`, lifted downward — because a sign flip in the
  closure (`+AnchorHeight` for `-AnchorHeight`) survives a suite that only ever lifts upward.
- **SC-8** `terraintool -flat` output is byte-identical across this story, at several scales, with and
  without `-objects`/`-units`: `AnchorHeight` is never called on that path (P-3, FR-4).
- **SC-9** `mapview -check`/`terraintool` summary text is byte-identical to pre-story output for every
  overlay/`-flat` combination (AC-8): the geometry moves, no summary token does.
- **SC-10** `AnchorHeight` and every `liftY`/`dy` computation are asserted from plain Go values, in
  tests that open no window and hold no graphics context (FR-7).
- **SC-11** A rectangle that intersects the view/canvas only AFTER its height offset is applied is
  still drawn, and one that intersected only BEFORE the offset is dropped — in both `pkg/ui`'s view
  cull and `terraintool`'s `img.Bounds()` clip (FR-6, P-5). Culling before the offset fails both
  directions.
- **SC-12** A manual pass over representative maps shows markers sitting on the displaced surface in
  both `mapview` (both overlays) and `terraintool -objects -units`, relief/water/light matching
  0012/0013/0014, any fidelity gap recorded (AC-9). One artifact is **expected, not a regression**,
  and the pass must record it as such: on a steep cell near a map edge `terraintool` can cut a
  marker's arm off at the un-displaced map extent while the relief under it continues past that line —
  that is DD-2's stage-1 clip doing exactly what FR-4 requires. `pkg/ui` has no such cap (below), so
  the two front-ends legitimately differ there.

- **SC-13** Over a valid, sloped altitude grid, `SetFlat(true)` reports Flat while `Lit()` stays
  **true**, world size is the flat `w*32×h*32`, and no marker carries a height offset; `SetFlat(false)`
  returns Displaced. `cmd/mapview -flat` selects it and leaves `-check` byte-identical (AC-10). A
  `Mode()` that ignores `v.flat` fails the first; a `-flat` that also suppresses the level grid fails
  the lit half, which is the whole reason the flag exists.

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1 | SC-1, SC-2, SC-10 |
| FR-2 | DD-5 | SC-3, SC-4 |
| FR-3 | DD-3 | SC-5 |
| FR-4 | DD-2, DD-4 | SC-7, SC-8 |
| FR-5 | DD-6 | SC-6 |
| FR-6 | DD-2, DD-3 | SC-11 |
| FR-7 | DD-1, DD-2 | SC-10 |
| FR-8 | DD-7 | SC-13, SC-4 |
