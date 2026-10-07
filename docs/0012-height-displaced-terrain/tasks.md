# Tasks — height-displaced terrain geometry

T1–T3 are independent; T4 needs all three, T9 needs T4, T5 needs T9, T7 needs T5 and T6. T9 is
numbered last and ordered fifth: it completes DD-9 on files T4 could not open.

## T1 — the step table and the two walks  *(implementation)*

- Boundary: pure arithmetic; stops before anything that reads a map or draws a pixel.
- Files: `pkg/render/terrain/step.go` (ADD), `pkg/render/terrain/step_test.go` (ADD).
- Covers: FR-6; AC-5; SC-2; DD-2, DD-12; R-3, R-5.
- Fences: touches no compositor, no `Render`, no `cmd/`. Adds no exported symbol beyond the table's
  accessor and the two walks. Does not special-case any step count.
- **Done when:** SC-2 passes, its expected disagreement set written from the recipe rather than from
  the implementation under test.

## T2 — the projection and the canvas  *(implementation)*

- Boundary: vertex values and canvas extremes only; nothing is drawn and no image is allocated.
- Files: `pkg/render/terrain/project.go` (ADD), `pkg/render/terrain/project_test.go` (ADD).
- Covers: FR-2, FR-9; AC-3; SC-1; DD-6.
- Fences: adds no compositor entry point and does not modify `Render`; does not consult the step
  table.
- **Done when:** SC-1 passes.

## T3 — the seams the projected path needs  *(implementation)*

- Boundary: behaviour-preserving changes to the shipped files — the shared validation helper with the
  canvas height as a parameter, `Render.OriginY`, a destination origin on the rectangle blitters, and
  `InterpSpan`. Stops before any projected geometry.
- Files: `pkg/render/terrain/composite.go` (MODIFY), `lit.go` (MODIFY), `shade.go` (MODIFY),
  `shade_test.go` (MODIFY).
- Covers: FR-10; DD-6, DD-7, DD-9.
- Fences: changes no pixel, dimension or rejection outcome of either flat entry point, and edits
  neither `composite_test.go` nor `lit_test.go` — they are the check that this task changed nothing.
  Adds no projected entry point.
- **Done when:** the composite, lit and `cmd/terraintool` suites pass unmodified; the `InterpSpan`
  test passes; both flat entry points report `OriginY` 0.

## T4 — the shaded projected compositor  *(implementation)*

- Boundary: `CompositeProjectedLit` end to end — selector, both blitters on the projected canvas,
  placeholders, paint order, uncovered pixels, and the projected rejection set. Stops at the unshaded
  sibling and at `cmd/`.
- Files: `pkg/render/terrain/project.go` (MODIFY), `project_test.go` (MODIFY).
- Covers: FR-3, FR-4, FR-5, FR-7, FR-8, FR-10; AC-4, AC-6, AC-7, AC-8, AC-9, AC-10; P-1, P-2, P-4,
  P-5, P-6; SC-3…SC-9, SC-11, SC-12; DD-3, DD-10, DD-12; R-1, R-5.
- Fences: does not touch `cmd/`, the overlays, or either flat entry point. Writes no clamp on a
  source or destination row, and no fill for a column the contract drops.
- **Done when:** SC-3…SC-9, SC-11 and SC-12 pass, SC-12 asserting for each case that the rejection it
  names is the one that fired.

## T5 — the unshaded sibling and flat equivalence  *(implementation)*

- Boundary: `CompositeProjected` over the same core, plus the equality property against both flat
  entry points.
- Files: `pkg/render/terrain/project.go` (MODIFY), `project_test.go` (MODIFY).
- Covers: FR-1; AC-1; P-3; SC-10, SC-12; DD-4.
- Fences: no second copy of the cell loop or of the validation; no change to the shaded entry point's
  output.
- **Done when:** SC-10 passes and SC-12 covers this entry point too.

## T6 — markers at a vertical origin  *(implementation)*

- Boundary: the `*At` entry points and the shifted clip; the glyph geometry itself is unchanged.
- Files: `pkg/render/terrain/overlay.go` (MODIFY), `overlay_test.go` (MODIFY).
- Covers: FR-11; AC-11; SC-13; DD-8; R-4.
- Fences: does not change the existing exported signatures or their results, and does not displace a
  marker onto the projected quad. Leaves `pkg/ui` untouched.
- **Done when:** SC-13 passes at a positive and a negative origin, and the existing overlay
  assertions pass unmodified.

## T7 — the tool: default, flat selection, summary  *(implementation)*

- Boundary: the command's surface — the flat flag, which entry point each flag combination reaches,
  the marker offset, and the two new summary tokens. Stops at the render tier's behaviour.
- Files: `cmd/terraintool/main.go` (MODIFY), `main_test.go` (MODIFY),
  `pkg/render/terrain/doc.go` (MODIFY).
- Covers: FR-1, FR-12; AC-2, AC-12; SC-14; DD-5, DD-11; R-1, R-2.
- Fences: adds the sloped fixture beside the existing one rather than giving the existing one
  altitudes; leaves every assertion that does not concern the summary or a baseline oracle alone; adds
  no flag beyond the flat selection.
- **Done when:** SC-14 passes on the sloped fixture, and the pre-existing `cmd/terraintool`
  assertions pass with only their summary strings and baseline oracles changed.

## T8 — AC-13 evidence against a lawful install  *(developer-run verification)*

- Produces: evidence only, no game bytes — every shipped map rendered at representative scales at the
  new default and under the flat selection; relief visible; the summary's geometry and origin; any
  map the budget now refuses (R-1); and the re-taken PNG hashes superseding those the earlier terrain
  stories recorded (R-2). An untracked `builds/0012-height-displaced-terrain/` carries the binary and
  its run note.
- Covers: AC-13; SC-15; R-1, R-2.
- Fences: records outcomes only — no code change, no edit to an earlier story's records, and no
  rendered image or converted asset inside the repo.
- **Done when:** the run is recorded, or an explicit "not run" limitation with its reason is.

## T9 — DD-9's second step: the mesh is measured only after the arguments pass  *(implementation)*

- Boundary: the shared helper's canvas-height parameter and its call sites, so the projected pair
  validates in DD-9's three stated steps — arguments, project, budget — rather than projecting ahead
  of the argument checks. Changes no pixel, no dimension, no rejection outcome and no message on any
  entry point.
- Files: `pkg/render/terrain/composite.go` (MODIFY), `lit.go` (MODIFY), `project.go` (MODIFY),
  `project_test.go` (MODIFY).
- Covers: FR-10; P-4; DD-9.
- Fences: adds no exported symbol, no second copy of the validation, and no rejection the flat pair
  does not already make. Touches neither compositor loop, the overlays nor `cmd/`. Edits neither
  `composite_test.go` nor `lit_test.go` — they are the check that the flat pair is untouched.
- **Done when:** an argument set the flat path refuses is refused by the projected path with the
  identical message and without walking the mesh, on dimensions large enough that walking it could
  not complete; SC-12 still passes for every case it names; and the entry point's order-of-work
  comment describes what the code now does.

## Traceability

| Requirement (spec) | Plan criterion | Task |
|---|---|---|
| FR-1 (what holds, what changes) | SC-10, SC-14 | T5, T7 |
| FR-2 (projection) | SC-1 | T2 |
| FR-3, FR-4 (selector, flat cell) | SC-4 | T4 |
| FR-5 (sloped cell) | SC-5, SC-7 | T4 |
| FR-6 (edges) | SC-2, SC-3, SC-11 | T1, T4 |
| FR-7 (shading domain) | SC-6 | T4 |
| FR-8 (ownership) | SC-3, SC-8 | T4 |
| FR-9 (canvas) | SC-1, SC-9 | T2, T4 |
| FR-10 (rejection, totality) | SC-12 | T3, T4, T9 |
| FR-11 (markers) | SC-13 | T6 |
| FR-12 (report) | SC-14 | T7 |
| AC-1 | SC-10 | T5 |
| AC-2 | SC-14 | T7 |
| AC-3 | SC-1 | T2 |
| AC-4 | SC-4 | T4 |
| AC-5 | SC-2 | T1 |
| AC-6 | SC-5, SC-6 | T4 |
| AC-7 | SC-7 | T4 |
| AC-8 | SC-8 | T4 |
| AC-9 | SC-11 | T4 |
| AC-10 | SC-12 | T4, T5 |
| AC-11 | SC-13 | T6 |
| AC-12 | SC-14 | T7 |
| AC-13 | SC-15 | T8 |
| P-1 (source inside the sub-cell) | SC-5 | T4 |
| P-2 (pixels inside the image) | SC-9 | T4 |
| P-3 (flat equals projected when level) | SC-10 | T5 |
| P-4 (atomic rejection) | SC-12 | T4, T5, T9 |
| P-5 (dropped column) | SC-7 | T4 |
| P-6 (totality) | SC-11, SC-12 | T4 |
