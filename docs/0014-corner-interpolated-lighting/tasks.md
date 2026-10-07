# Tasks — corner-interpolated lighting in the window

Legend: **files** the task may change · **done when** the observable it must leave behind.

## T1 — the two shading accessors

**files** `pkg/render/terrain/shade.go`, `pkg/render/terrain/shade_test.go`, `pkg/render/terrain/lit.go`,
`pkg/render/terrain/project.go`

Add `CornerLevels` (DD-2) and `ShadeScale` (DD-3), and factor the level clamp `ShadeChannel` already
performs into one helper both call. Then **route the two existing consumers through `CornerLevels`**:
`CompositeLit`'s local `lvlAt` closure and `compositeProjected`'s four `levelAt` calls are today two
independent copies of the far-edge clamp, and DD-2's "one place" is only true once they are one.

**done when** SC-4, SC-5 and SC-11 pass. SC-11 is what makes the refactor a claim and not a hope:
`lit_test.go` and `project_test.go` pass **unedited**, and a real `terraintool` render is byte-compared
across the change, shaded and `-flat`. A grid too short for its dimensions, a non-positive dimension
and a tile outside the grid each have a defined answer and a test.

## T2 — the viewer's lighting state, still drawing as it does

**files** `pkg/ui/viewer.go`, a new `pkg/ui/light_test.go`

Build the level grid in `NewViewer` under the existing altitude guard (DD-1); add the `unshaded` field,
`SetUnshaded`, `Lit()` (DD-6) and `cornerScales` (DD-5, DD-7). The draw loops are **not** changed in
this task: nothing on screen moves yet.

**done when** SC-6, SC-7 and SC-12 pass and SC-1 holds — `NewViewer`'s signature is unchanged and no
existing test file in `pkg/ui`, `pkg/render/camera` or `pkg/game` is edited. `cornerScales` returns
all-1 for an unlit viewer and for a placeholder cell, and its values for a lit cell are `ShadeScale` of
that cell's four corner levels in TL, TR, BL, BR order.

## T3 — both draw paths carry the corner colours

**files** `pkg/ui/viewer.go`, `pkg/ui/light_test.go`

Add `withScales` and `flatTileVertices`, and have `drawFlat` submit a quad through `DrawTriangles` with
the same `quadIndices` (DD-4) while `drawDisplaced` composes the scales onto the vertices it already
builds. `tileVertices` keeps its signature and its all-1 colours (DD-5), so `viewer_test.go` is **not**
edited — if it needs editing, the split was done wrong. This is the task the window changes in.

**done when** SC-2, SC-3 and SC-8 pass and the geometry half of AC-3/AC-4 holds: the displaced corners
are byte-identical to what `tileVertices` produced before, and the flat corners are the un-displaced
lattice through the camera. Every assertion is checked against a deliberate break — at least the four
corners transposed, the scales applied to the wrong corner, a single scale used for all four, the
colour left at its zero value, and the flat quad split along the other diagonal.

## T3b — the draw call becomes observable

**files** `pkg/ui/viewer.go`, `pkg/ui/light_test.go`

T3's assertions all test the **builders**: deleting `withScales` from both draw loops leaves every
package green, measured. Give `drawFlat` and `drawDisplaced` a `triangleTarget` parameter (DD-9) so a
recording target can read what each loop actually submits.

**done when** SC-13 passes and SC-1 still holds — no exported signature moves and no existing test
file is edited. Both breaks are checked: `withScales` dropped from either loop, and flat mode given
its own diagonal, each fails a test.

## T4 — the diagnostic flag

**files** `cmd/mapview/main.go`, `cmd/mapview/*_test.go`

Add `-unshaded` and wire it to `SetUnshaded`. The summary text gains nothing: no token, no reordering,
no dependence on lighting (FR-1).

**done when** the flag's help line reads as a diagnostic, `-check` output is unchanged by it, and
`cmd/againrom` has gained no flag.

## T5 — evidence

**files** `docs/0014-corner-interpolated-lighting/verification.md`

Run the gates, the unit map (each SC and AC to the test that witnesses it), and the developer runs:
SC-9 over the shipped corpus against a build from this story's base commit; AC-10 on the corpus with
and without `-unshaded`, comparing at least two maps against that map's PNG render; SC-10's tying zoom,
so FR-1's unpainted column is observed rather than assumed absent; AC-11 through the real front-end.

**done when** every figure is a real measurement with the compared shape asserted, every difference
found against the PNG render is recorded rather than explained away, and anything not run is listed as
not run.

## T7 — drag to pan

**files** `pkg/ui/viewer.go`, `pkg/ui/input_test.go`

Add `PrimaryDown` to `Input` and read it in `readInput` (DD-10); hold the drag anchor on the viewer
and pan from the difference between consecutive held ticks, dividing the screen delta by the zoom. A
tick that pans by drag does not also edge-scroll. `step` stays the pure entry both the app and the
tests drive, so nothing new needs a window.

**done when** SC-14 and SC-15 pass and SC-1 still holds — no exported signature moves and no existing
test's expectations change (an `Input` literal without the new field still means "button up"). Both
named breaks are run and each fails a test: the zoom division inverted, and multiplied instead of
divided. The clamp, the centring and `ZoomAbout` are untouched — a drag cannot leave the world.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-3, FR-8 | DD-2, DD-3, SC-11 |
| T2 | FR-3, FR-4, FR-6, FR-7 | DD-1, DD-5, DD-6, DD-7, SC-12 |
| T3 | FR-1, FR-2, FR-7 | DD-4, DD-5, DD-8 |
| T3b | FR-1, FR-2, FR-7 | DD-9, SC-13 |
| T4 | FR-5 | DD-6 |
| T5 | AC-9, AC-10, AC-11 | SC-9, SC-10 |
| T7 | FR-1, FR-9 | DD-10, SC-14, SC-15 |
