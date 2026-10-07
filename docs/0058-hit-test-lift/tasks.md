# Tasks — 0058 hit test lift

## T1 — the cell's own rectangle becomes one value, and the rim is its border

FR-2, FR-7 (the glyph), FR-9; DD-1, DD-2.

In the render tier. The rectangle the blocked tint fills is the cell footprint: rename it for what it
is, leave its clip and its rejection rule exactly as they stand, and move it beside the glyph family
it now serves. Update its existing callers and their tests — a rename, not a behaviour change.

Write the border of a rectangle once, as strips, and rebuild the selection rim on it. The rim's
output must be **byte-identical** to what it produces today at every scale the existing tests cover:
the footprint is wholly inside the map for an in-map cell, so the per-strip clip that the rim applies
today is an identity — prove that rather than assume it, by leaving the existing rim tests untouched
and requiring them to pass unedited.

Add the lattice outline as the same border at one native pixel, and a colour for it. The colour is
premultiplied: no channel may exceed its alpha, and say in its comment why, since a wrong value there
renders as undefined rather than as visibly wrong.

Tests: the footprint under its new name; the outline's four strips at native scale and at a scale
where the thickness rounds; that the outline and the rim stand on the same rectangle; the off-map and
invalid-scale rejections the family shares.

## T2 — the pick tests the rectangle the unit is drawn on

FR-1, FR-3, FR-4, FR-5, FR-6, FR-10, FR-11; P-1, P-3, P-4; DD-3, DD-4, DD-5, DD-6, DD-7.

In `pkg/ui`. One function turns a cell and a displacement into a rectangle in the view, by composing
the footprint with the transform every glyph already takes. It must **call** that transform, not
restate it: if the lift expression appears twice in the tree when you are done, the task is not done.

`decide` takes the placement as a parameter and keeps its camera for the ground pick alone. Form the
drag rectangle once, normalised and closed, and test it against the half-open placement — one
predicate serving both branches.

Re-point the camera's cell-range inverse: it is no longer the box-selection path. Do not delete it,
and do not leave its comment claiming a role it has lost.

Tests: a displaced fixture with a known lift, boxed at the rim and boxed at the flat cell; tap and
one-point box swept over a set of points; the id tie; a mid-step entity boxed where it is drawn;
orientation independence; the corpse rule; the order destination untouched. **Every existing pick
test must pass unedited** — a change there is a finding, not something to adjust.

## T3 — the lattice, and the key that shows it

FR-7, FR-8; P-2; DD-8, DD-9, DD-10, DD-11.

In `pkg/ui`. One pass, first in the slice, built by walking the band the terrain pass itself walks —
the displaced loop's own row range in displaced mode, the camera's visible tiles in flat — and
placing each cell's outline through the very function T2 built. A second band, or a second placement,
is the defect this story exists to close, reappearing in the instrument meant to reveal it.

The toggle is a press-edge field on the input snapshot, bound to a free key, read on the map arm
alone, and off by default. Add a reader for the switch beside the ones the other overlays have, so a
front-end can see the state without opening a window.

Tests: the pass is absent with the lattice off and the slice otherwise unchanged; present with it on;
its band equals the terrain pass's after the cull; an outline carries its cell's own lift in
displaced mode and none in flat; the outline of the cell a still unit stands on equals that unit's
hit target; the key toggles once per press edge.

## T4 — what the adversarial read found

FR-5, FR-7, FR-10; AC-3a, AC-4, AC-14; P-2, P-3; DD-5's guard, DD-12; R-1, R-2.

The plan gate found four defects the built code and its tests could not see. Each is a revision, not
a tidy-up, and each needs the evidence the original claim lacked.

The tap and the box do **not** answer identically where several hit targets hold one point, and the
sweep that claimed they did was run on a fixture with no overlap. Rebuild it on one that contains the
overlap, and assert the relation that is actually true.

`meets` is correct only for a rectangle with area, and a zero or non-finite extent beside a finite
position is reachable through the raw zoom field. Guard it, and test an empty and an inverted target.

The entity displacement is not gated on displaced mode, so a mid-step unit on flat ground is now
picked where it is drawn. No existing test sets a step or a phase, so nothing saw it. Pin it.

The lattice's band is the view in cells and grows as the inverse square of the zoom: at the camera's
minimum it is the whole map. Add the floor DD-12 names, choosing it on legibility, and record the
measurement at native zoom, at the floor, and below it.

## Traceability

| Task | Requirements | Criteria | Design |
|---|---|---|---|
| T1 | FR-2, FR-7, FR-9 | AC-9; SC-3 | DD-1, DD-2 |
| T2 | FR-1, FR-3, FR-4, FR-6, FR-11 | AC-1, AC-2, AC-5..AC-9, AC-13; P-1, P-4; SC-1, SC-6 | DD-3, DD-4, DD-6, DD-7 |
| T3 | FR-7, FR-8 | AC-10, AC-11, AC-12; SC-5 | DD-8..DD-11 |
| T4 | FR-5, FR-10 | AC-3, AC-3a, AC-4, AC-14; P-2, P-3; SC-2, SC-4 | DD-5, DD-12; R-1, R-2 |
