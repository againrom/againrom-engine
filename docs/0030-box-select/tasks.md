# Tasks — the modifier, the band of cells, the set, and the marks over it

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it must
not do · **done when** the observable it leaves behind. Every entry below is an implementation entry,
and they land in ascending order, each depending only on entries before it. SC-9's mutants are one to
an entry, applied to production code, measured over the whole tree and reverted by the entry that owns
one; a kill is claimed only where it was run. An entry owning no mutant says so and breaks something
load-bearing of its own instead.

## T1 — the modifier arrives, and the gesture is latched where it begins

**files** MODIFY `pkg/ui/viewer.go`, `pkg/ui/viewer_test.go`, `pkg/ui/flow.go`, `pkg/ui/flow_test.go`,
`pkg/ui/input_test.go`

DD-2, DD-3 — FR-1, FR-7.

**fences** `command.go` is not opened: no selection, no rectangle and no pick moves here, and nothing
here draws. The mode flag is unexported and reachable from no exported method, so `cmd/mapview` gains no
call, no flag and no field. `panIntent`'s existing suppression is not edited. The difference branch
keeps its single subtraction — no second delta appears beside it — and the accumulator is still raised
there in both modes.

**done when** SC-1 holds: with the flag clear a plain drag pans by the delta it panned by before and a
Shift drag pans identically; with it set a Shift drag pans by that same delta while a plain drag pans
zero, the accumulator rising equally in both; a press begun under one modifier state and continued under
the other resolves as the state at the press; and the press point is still readable after several ticks
of movement, where the drag anchor is not. Then SC-9's live-latch mutant is applied, the whole tree run
with the failing tests named, reverted, and byte-identity confirmed.

## T2 — the camera answers a rectangle of cells

**files** MODIFY `pkg/render/camera/camera.go`, `pkg/render/camera/camera_test.go`

DD-4 — FR-2.

**fences** `ScreenToCell` is not edited and no clamp appears on its path. Nothing here names a unit, a
selection or an entity id — the camera learns about rectangles and nothing else — and no second extent
field joins `Cols`/`Rows`. No rounding of its own is introduced beside the floor already in use, and
nothing is returned for a rectangle that missed the map beyond the report that it did.

**done when** SC-2 holds, each named case carried by a case of its own. Then SC-9's dropped-`+1` mutant
is applied, the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T3 — the selection becomes a set, and a release fills it

**files** MODIFY `pkg/ui/command.go`, `pkg/ui/command_test.go`, `pkg/ui/viewer.go`,
`pkg/ui/viewer_test.go`

DD-1, DD-5 — FR-2, FR-3.

**fences** `decide` stays pure, calling the camera and nothing else, and no second pure function appears
for the release. The set is built by one walk of the snapshot and nothing sorts it afterwards; no Go map
holds it and nothing ranges one. The order path is not widened here and the seam is not opened — that is
T4's — and neither the mark nor the draw path is touched. Nothing prunes the stored set.

**done when** SC-3 holds in full, its camera non-identity throughout, and a tap is seen yielding a set
of one carrying the minimum-id rule. Then SC-9's normalisation mutant is applied, the whole tree run
with the failing tests named, reverted, and byte-identity confirmed.

## T4 — one order per present member, ascending, through the seam as it stands

**files** MODIFY `pkg/ui/command.go`, `pkg/ui/command_test.go`, `pkg/ui/app.go`, `pkg/ui/app_test.go`

DD-1, DD-5, DD-6, DD-8 — FR-4, FR-8.

**fences** `MapOrder`'s signature does not move and no collection crosses the seam; `pkg/game` and
`pkg/sim` are not in the file list and their suites are predicted green, one that moves being reported
rather than softened. The advance stays first, unconditional and single on the map arm. The presence
walk written here is the only one — the mark reads it in T5 rather than growing a second — and it
filters without writing, so the stored set is unchanged by any tick.

**done when** SC-4 holds in full, and a frame carrying k orders is seen reaching the seam as k calls in
ascending id while the advance count reads one at zero, one and many orders. Then SC-9's reversed-slice
mutant is applied, the whole tree run with the failing tests named, reverted, and byte-identity
confirmed.

## T5 — a mark per present selected unit

**files** MODIFY `pkg/ui/overlay.go`, `pkg/ui/selection_overlay_test.go`

DD-6 — FR-5.

**fences** `overlayScreenRects` is not edited and no second transform, lift or cull is written; the glyph
stays the shipped rim and no new one is added to the render tier. The presence predicate is the one T4
left, not a second one. Nothing here writes a selection, and the pass's position in the slice does not
move.

**done when** SC-6 holds, the empty and all-absent cases asserted on the pass slice itself rather than on
an empty rect list. Then SC-9's presence-filter mutant is applied, the whole tree run with the failing
tests named, reverted, and byte-identity confirmed.

## T6 — the outline, in screen pixels, over everything

**files** MODIFY `pkg/ui/overlay.go`, `pkg/ui/viewer.go`; ADD `pkg/ui/marquee_test.go`

DD-2, DD-7 — FR-6.

**fences** the rectangle is built from the press point and the current cursor alone: it does not go
through the world-to-screen transform, no cell reaches it, and no builder is added to
`pkg/render/terrain`. It is hollow — no filled rectangle appears anywhere on this path. It is appended
after every existing pass and nothing before it moves. A viewer not in command mode builds none, and no
exported method reaches it.

**done when** SC-7 holds, the four strips asserted to leave the interior uncovered at both drag
orientations and the under-slop press asserted to build no pass at all. Then SC-9's command-mode-gate
mutant is applied — which takes the standalone viewer's pan away and this entry's no-command-mode case
with it — the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T7 — a group ordered against a world, and the digests it does not move

**files** ADD `pkg/game/group_order_test.go`; MODIFY `pkg/game/order_invariance_test.go`

DD-8 — FR-8.

**fences** no production file is opened. The world is built by hand over a layout small enough to read,
and no game install is read. Nothing here asserts that a surplus unit arrives: the criteria are the
command stream, the digests and the shared-cell check, and where the surplus stops is recorded rather
than required. The headless side of every digest comparison is assembled from the command stream this
entry asserts, never from a second run of the driver under test.

**done when** SC-5 and SC-8 hold, P-1 and P-2 sampled over that run. This entry **owns no mutant**, and
that is stated rather than filled: every production line it exercises is already carried by an entry
above, and one invented here would be measured here and killed there. What it breaks instead is its own
independence — the digest comparison is re-run with the headless side taken from the driver under test,
seen to pass, and reverted, which is the measurement that the comparison could otherwise not
discriminate.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, FR-7, AC-1, AC-8 | DD-2, DD-3, SC-1 |
| T2 | FR-2 | DD-4, SC-2 |
| T3 | FR-2, FR-3, AC-2, AC-3 | DD-1, DD-5, SC-3 |
| T4 | FR-4, FR-8, AC-4, P-3, P-5 | DD-1, DD-5, DD-6, DD-8, SC-4 |
| T5 | FR-5, AC-6 | DD-6, SC-6 |
| T6 | FR-6, AC-7 | DD-2, DD-7, SC-7 |
| T7 | FR-8, AC-5, AC-9, AC-10, P-1, P-2 | DD-8, SC-5, SC-8 |
