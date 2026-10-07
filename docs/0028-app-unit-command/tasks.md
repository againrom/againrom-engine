# Tasks — selecting a unit and ordering it from the running game

Legend: **files** what the task may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. Every entry is an implementation task;
they land in ascending order, each depending only on entries before it. A mutant is applied, run and
reverted by the entry that owns it, with its failing tests named; a survivor is reported as a
survivor, and a kill is claimed only where it was actually run.

## T1 — the pick: a screen point to a map cell, outside reported

**files** MODIFY `pkg/render/camera/camera.go`, `pkg/render/camera/camera_test.go`

DD-2 — FR-7, C-1.

**fences** no file outside `pkg/render/camera`, whose imports stay stdlib. `ScreenToWorld`,
`WorldToScreen` and `VisibleTiles` are untouched, clamp and all: the new answer sits beside them and
no clamp appears on it. `Cols`/`Rows` are read where they are, never widened into a parameter.
Nothing here learns what a unit or a selection is — the hit test is T2's and the order gate T4's.

**done when** SC-1 holds over this entry's tests, and SC-9's `ScreenToCell` mutant — the outside
test deleted — is applied, run with the failing tests named on record, and reverted. AC-10's own
cases do not exist yet, so the kill claimed here is SC-1's alone.

## T2 — the id across the seam, the slop, and the four outcomes

**files** MODIFY `pkg/ui/overlay.go`, `pkg/ui/viewer.go`, `pkg/ui/app.go`, `pkg/ui/input_test.go`,
`pkg/ui/app_test.go`, `pkg/ui/entity_overlay_test.go`, `pkg/ui/mirror_draw_test.go`,
`pkg/game/world.go`, `pkg/game/world_test.go`; ADD `pkg/ui/command.go`, `pkg/ui/command_test.go`

DD-1, DD-3, DD-4 (the machinery's unexportedness) — FR-1, FR-3, FR-7, FR-8.

**fences** nothing is wired yet: the map arm still calls `v.step` and nothing else, so the order
`command` returns is dropped by its only caller until T4 — add no `flow` field, no loader result,
no queue. No highlight and no pass (T3). `pkg/sim`, `pkg/render` and `cmd/` are untouched;
`pkg/game`'s whole share is `entityDraws` filling the new field and the witness for it. `dragMoved`
is raised inside the shipped `dragIntent`, never by a second delta beside it, and `decide` reaches
`ScreenToCell` and nothing else.

**done when** SC-2 and SC-3 hold over this entry's tests; an entity's own id is witnessed crossing
`entityDraws` rather than assumed from a hand-built slice; and SC-9's slop mutant — the comparison
widened to `<=` — is applied, run with the failing tests named on record, and reverted.

## T3 — the highlight: the footprint's rim, drawn last

**files** MODIFY `pkg/render/terrain/overlay.go`, `pkg/render/terrain/entity_overlay_test.go`,
`pkg/render/terrain/static_marker_test.go`, `pkg/ui/overlay.go`, `pkg/ui/marker_height_test.go`,
`pkg/ui/entity_overlay_test.go`; ADD `pkg/render/terrain/selection_marker_test.go`,
`pkg/ui/selection_overlay_test.go`

DD-8 — FR-2.

**fences** the four shipped glyph builders keep their arms, their colours and their places in
`overlayPasses`; the new pass is appended and no existing pass moves. The rim rides
`overlayScreenRects` — no transform, lift or cull of its own — and reads the selection T2 already
stores instead of taking one as a parameter. Nothing here issues, queues or applies an order, and
`pkg/render/terrain` gains no import.

**done when** SC-4 holds over this entry's tests; the new builder answers the family's shared
rejection and per-rect clip on the same terms the four already tested there do; and with nothing
selected no pass is appended at all, so every pass count the entity-layer tests pin is unmoved.

## T4 — the order leaves through the seam, and one advance drains the queue

**files** MODIFY `pkg/ui/flow.go`, `pkg/ui/app.go`, `pkg/ui/flow_test.go`, `pkg/ui/app_test.go`,
`pkg/game/world.go`, `pkg/game/frontend.go`, `pkg/game/world_test.go`,
`pkg/game/frontend_test.go`, `pkg/game/frontend_statics_test.go`

DD-4 (the arm's call order), DD-5, DD-6 — FR-3, FR-4, FR-10.

**fences** `MapTick` keeps its signature and its one unconditional call per map-screen tick, ahead
of `v.step`, with `command` after it. `paceTo` reports its second result and `paced` discards both,
so `ui.MapTick` still names nothing. `sim.Step` keeps its single call site. The script exclusion is
T5's — a commanded unit still takes scripted targets after this entry, so add no `commanded` map and
no filter. `pkg/sim`, `pkg/render` and `cmd/` are untouched.

**done when** SC-5 and SC-6 hold over this entry's tests; `flow`'s pinned field set admits the new
seam and the tests witness it dropped on the same statement the tick is; and SC-9's untruncated-queue
mutant is applied, run with the failing tests named on record, and reverted — killed by SC-6's count
alone, no digest moving. Report that as it stands; do not widen the claim.

## T5 — the exclusion, and the stream one tick is advanced on

**files** MODIFY `pkg/game/world.go`, `pkg/game/world_test.go`

DD-7 — FR-5, FR-6, FR-9.

**fences** `commanded` is written by the statement that enqueues, read by lookup, never ranged, and
never cleared while the map is open. A schedule entry's own slice is never written into: with nothing
commanded and nothing pending the assembled slice must BE that entry, by identity rather than by
equality. No world field, byte form or digest changes, and `pkg/sim` is not opened.

**done when** SC-7 and SC-8 hold over this entry's tests, plus a direct assertion over the
**assembled slice** that pending follows that tick's scripted commands — the only witness the splice
order has, and the reason FR-4's "after" is carried by a slice and not a digest: the exclusion
removes the scripted command an order could lose to, so no world state distinguishes the two orders.
SC-9's exclusion-write and reversed-splice mutants are each applied, run with the failing tests named
on record, and reverted; the first is killed by SC-7 alone, the second by that slice assertion alone,
and neither moves a digest.

## T6 — the surfaces this story leaves alone

**files** MODIFY `pkg/game/world_test.go`, `pkg/game/frontend_test.go`, `cmd/mapview/main_test.go`;
ADD `pkg/game/order_invariance_test.go`

DD-4, DD-6 — FR-9, FR-10.

**fences** no production file changes. Nothing under `pkg/sim` is edited, its pinned field-set and
byte-form tests included — those passing **unedited** is the evidence, and touching one destroys it.
The standalone viewer's share is asserted through the surface `cmd/mapview` actually has, never by
reaching inside `pkg/ui`.

**done when** SC-10 holds: a session driving a selection, a pick and a pending order reaches the
digests of the same advances driven without one; `pkg/sim`'s suite and both wall checks pass
unedited; and the standalone viewer's render and the headless check line are byte-identical.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-7, AC-1, C-1 | DD-2, SC-1, SC-9 |
| T2 | FR-1, FR-3, FR-7, FR-8, AC-2, AC-3, AC-4, AC-10, P-5 | DD-1, DD-3, DD-4, SC-2, SC-3, SC-9 |
| T3 | FR-2, AC-9 | DD-8, SC-4 |
| T4 | FR-3, FR-4, FR-10, AC-5, AC-8, P-1, P-6 | DD-4, DD-5, DD-6, SC-5, SC-6, SC-9 |
| T5 | FR-5, FR-6, FR-9, AC-6, AC-7, P-3, P-4 | DD-7, SC-7, SC-8, SC-9 |
| T6 | FR-9, FR-10, AC-11, P-2 | DD-4, DD-6, SC-10 |
