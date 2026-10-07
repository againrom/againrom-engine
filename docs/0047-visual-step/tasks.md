# Tasks — pin the picture, take the step, read the clock, move the glyph

Legend: **files** what the entry may change — a permission, not a prediction · **fences**
what it must not do · **done when** the observable it leaves behind. Every entry is an
implementation entry and they land in ascending order, each depending only on those before
it. T1 is the characterization entry the brownfield areas owe: it runs against the
unmodified push and glyph paths, so it lands before anything changes them. SC-10's two
mutants both name the displacement arithmetic, so both are applied by the entry that
WRITES that arithmetic, measured over the whole tree, and reverted there; a mutant applied
to a line an entry has not written yet kills nothing, and a kill is claimed only where it
was run.

## T1 — the entity picture as it stands, pinned before it is touched

**files** ADD `pkg/game/drawn_invariance_test.go`; MODIFY `pkg/ui/overlay_test.go`

FR-7, FR-8 — SC-7, SC-8.

**fences** no production file is opened and nothing is corrected in passing: what is
asserted is what the tree does today, including the classification this story goes on to
replace, which is pinned **only** in the parts that must survive it — the push's order, its
art resolution, its life byte and health pair, and the frame cadence of an entity at rest.
No test here names a displacement, a phase or a step. The world is hand-built and no game
install is read.

**done when** SC-8 holds against the unmodified tree and the digest half of SC-7 is
established over a `k`-tick run with nothing drawn. This entry owns no mutant, and that is
stated rather than filled: it changes no production line for one to be applied to.

## T2 — the step the unit took, and the direction read off it

**files** MODIFY `pkg/game/world.go`, `pkg/game/world_test.go`, `pkg/ui/overlay.go`; ADD
`pkg/game/facing_test.go`

DD-1, DD-2, DD-3 — FR-1, FR-2.

**fences** `pkg/sim` is not opened: no field, no clearing rule and no arrival test moves,
and the octant table and its centre value are left exactly as they are. The new memory is
looked up by key and never ranged, the snapshot build writes none of it, and the push gains
no second call site. Nothing here draws anything, reads a clock or computes a pixel, and no
other field of the seam value changes meaning.

**done when** SC-1 and SC-2 hold, the direction asserted on the pushed entities rather than
on the memory behind them, the tick-0 push seen to carry no step for any entity, and a
snapshot built twice in succession seen to answer the same both times.

## T3 — where a frame stands inside the tick

**files** MODIFY `pkg/render/terrain/water.go`, `pkg/render/terrain/water_test.go`,
`pkg/game/world.go`, `pkg/game/world_test.go`, `pkg/ui/viewer.go`

DD-4, DD-5 — FR-6.

**fences** the cadence itself does not move: no period, no accumulator rule, no catch-up
bound, no baseline write and no order of the stop against them. The accessor added is
read-only and the remainder stays unexported. The one value pushed is read from the driver's
pacing clock and from no other instance of that type. Nothing consumes the phase yet — no
glyph, no placement and no cull changes here — and the standalone viewer gains no call.

**done when** the two states SC-6's clamp and never-told arms are stated over exist and are
read back — the remainder carried across an advance and across a re-rate that shortens the
period below it, and a viewer never told a phase holding none — and the pushed phase seen
to be unchanged across stopped frames. What those two states DRAW is T4's.

## T4 — one vector, added to everything one unit draws

**files** MODIFY `pkg/ui/overlay.go`; ADD `pkg/ui/shift_test.go`

DD-6, DD-7, DD-8, DD-9 — FR-3, FR-4, FR-5, FR-8.

**fences** the shared per-cell transform keeps its single copy and its order — lift, camera,
cull — and neither it nor the cell-list helper is widened to carry a vector: the three
diagnostic overlays keep that helper's signature and allocate nothing new per frame. Draw
order, glyph shapes, colours and the sprite-or-square split do not change. No arithmetic
here reads a clock or a world, and nothing is displaced after the camera.

**done when** SC-3, SC-4 and SC-5 hold, the equality across sprite, mark and bar asserted
per entity at each phase rather than per family, SC-6's clamp and never-told arms hold as
geometry, and SC-8 re-run unchanged. Then SC-10's two mutants are applied one at a time to
the arithmetic this entry writes, the whole tree run with the failing tests named, reverted,
and byte identity confirmed. Beside them the shared vector is broken — the mark's copy given
a phase of its own, the three families seen to separate — and that reverted too.

## T5 — what a drawing does not move

**files** MODIFY `pkg/game/drawn_invariance_test.go`

FR-7 — SC-7, P-1, P-2.

**fences** no production file is opened. The headless side of every comparison is assembled
from the command stream this entry asserts and never from a second run of the driver under
test; no test here asserts a wall-clock duration or reads a game install.

**done when** SC-7 holds in full, the digests compared at every tick index rather than at
the end, and the stopped span of SC-6 re-read as the frame-by-frame identity P-2 states.
This entry owns no mutant: it exercises no production line an entry above does not already
carry.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-7, FR-8, AC-9 | SC-7, SC-8 |
| T2 | FR-1, FR-2, AC-1, AC-2, AC-3 | DD-1, DD-2, DD-3, SC-1, SC-2 |
| T3 | FR-6, AC-7 | DD-4, DD-5, SC-6 |
| T4 | FR-3, FR-4, FR-5, FR-8, AC-4, AC-5, AC-6, P-3, P-4 | DD-6, DD-7, DD-8, DD-9, SC-3, SC-4, SC-5 |
| T5 | FR-7, AC-8, P-1, P-2 | SC-7 |
