# Tasks — a running world under the map screen

Legend: **files** the task may change · **done when** the observable it must leave behind. Every
entry is an implementation task; they land in ascending order, and every dependency of a task is a
task before it.

## T1 — `pkg/render/terrain`: the entity glyph

**files** `pkg/render/terrain/overlay.go`; new `pkg/render/terrain/entity_overlay_test.go`

DD-4 — the square, the colour, and the two helpers both glyph builders come to share.

**done when** SC-6 passes (FR-6). No raster `Draw…` entry point is added for this glyph; the three
existing glyph builders keep their signatures, and the marker, unit and static overlay tests stay
green with no edit, which is what says the extraction moved no arm.

## T2 — `pkg/ui`: the entity pass

**files** `pkg/ui/overlay.go`, `pkg/ui/viewer.go` (the one field); new
`pkg/ui/entity_overlay_test.go`

DD-5, over the glyph T1 leaves.

**done when** SC-7 passes (FR-5, FR-7). No field, parameter or method here names a simulation type;
`Draw`'s body is untouched, so the pass order this task moves is the one inside `overlayPasses`; the
setter re-syncs nothing; and the pass-order assertions in the unit and static overlay tests are not
edited.

## T3 — `pkg/mapload`: the schedule

**files** new `pkg/mapload/schedule.go`, `pkg/mapload/schedule_test.go`; `pkg/mapload/fromalm.go`
(the cell conversion the two now share)

DD-3.

**done when** SC-5 passes (FR-4). `FromALM`'s signature and the world it returns are unchanged, and
its own test needs no edit; nothing here reads a clock, a generator, an environment or a file;
`pkg/sim` gains nothing.

## T4 — `pkg/ui`: one advance per map-screen tick, and nowhere else

**files** `pkg/ui/flow.go`, `pkg/ui/app.go`, `pkg/ui/flow_test.go`, `pkg/ui/app_test.go`;
`pkg/game/frontend.go` (`loadMap`'s signature alone); `internal/archtest/dag_test.go` (the new pin)

DD-1 — the seam alone, driven in its tests by stub tickers.

**done when** SC-1 passes (FR-2, FR-3, FR-9). `loadMap` hands back a nil tick and builds no world —
T5 owns that; the allow map in `dag.go` is not edited, only pinned; `Viewer` gains no field and no
method, and `Viewer.step` is untouched, so the standalone entry point keeps the behaviour its
characterization pin already froze.

## T5 — `pkg/game`: the world under the map screen

**files** new `pkg/game/world.go`, `pkg/game/world_test.go`; `pkg/game/frontend.go` (`loadMap`'s
body)

DD-2 and DD-6, under DD-7's fixtures.

**done when** SC-2, SC-3, SC-4, SC-8 and SC-9 pass (FR-1, FR-8, FR-10, FR-11). `LoadMapViewer` and
`MarkerCells` are unchanged and no anchor is shifted in this package; `pkg/sim` and `pkg/mapload`
gain no exported call; the fixture is an `alm.Map` literal with hand-written anchors and the
expected cells are literals beside it, never a shift the test recomputes.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-6 | DD-4, SC-6 |
| T2 | FR-5, FR-7 | DD-5, SC-7 |
| T3 | FR-4 | DD-3, SC-5 |
| T4 | FR-2, FR-3, FR-9 | DD-1, SC-1 |
| T5 | FR-1, FR-8, FR-10, FR-11 | DD-2, DD-6, DD-7, SC-2, SC-3, SC-4, SC-8, SC-9 |
