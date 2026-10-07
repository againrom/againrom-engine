# Tasks — the timeline, the gate, and the two front-ends

Legend: **files** what an entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. All six are *implementation* entries,
in ascending order, each depending only on entries before it. SC-8's five mutants are distributed
one or two to an entry, applied to production code, measured over the whole tree and reverted by
the entry that owns one; a kill is claimed only where it was run.

## T1 — the class's own timeline

**files** MODIFY `pkg/data/anim.go`; ADD `pkg/data/objectanim_test.go`

DD-1 — FR-1.

**fences** `expandTrack` is read and neither edited nor generalised, and the three unit tracks and
`UnitClass.Anim()` are untouched. No key is added to the object key table, no validation rule
moves, and the resolved arrays are read rather than re-resolved. Nothing outside `pkg/data` is
opened, and nothing here learns what a frame is or where one is drawn.

**done when** AC-1 holds and SC-1 with it, each of the four pairs on a case of its own and every
expected timeline written out by hand rather than produced by the expansion under test.

## T2 — the frame, the step and the guard

**files** ADD `pkg/render/terrain/objectanim.go`, `pkg/render/terrain/objectanim_test.go`

DD-3 — FR-2, FR-3.

**fences** no existing file is opened and no caller is converted, so this entry moves no pixel
anywhere in the tree. The gate arrives as an argument and no tile word is read here; `WaterPhase`,
its `>>2` and its `(col+1)*row` are read and neither reused nor generalised. Nothing is cached, no
package-level variable appears, and the guard stays the selector's last act rather than a test
repeated inside each arm.

**done when** AC-2 and AC-4 hold and SC-2 with them, together with AC-9's selector half — a period
of 0, an empty timeline, a negative `Index`, and the counter at its maximum. Then SC-8's
transposed-stagger mutant and its `Index`-instead-of-frame-0 mutant are each applied, the whole
tree run with the failing tests named, reverted, and the tree confirmed byte-identical.

## T3 — the loader carries the sheet, and `Frame` keeps its meaning

**files** MODIFY `pkg/render/terrain/statics.go`, `pkg/game/statics.go`, `pkg/game/statics_test.go`

DD-2 — FR-1, FR-2.

**fences** the four exclusions and the two artless answers are unchanged and the census still counts
them apart. `sheetCache.frame` and `frames` gain no parameter and no second failure mode;
`DeadObject` and `FireObject` stay unread. No GPU work, no window, and no archive held past the
load. The placement builder is not opened by this entry.

**done when** AC-6 holds, and the loader's own invariant — `Frame` non-nil exactly when `Frames` is
non-nil and `Index` lies inside it — is pinned by a test over all four exclusions rather than
argued in a comment.

## T4 — the gate, the animated subset and the per-counter pass

**files** MODIFY `pkg/render/terrain/statics.go`, `pkg/render/terrain/static_place_test.go`;
ADD `pkg/render/terrain/objectplace_test.go`

DD-4, DD-5, DD-6, DD-8 — FR-4, FR-6, FR-9.

**fences** `StaticAnchor` is neither edited nor called by the per-counter pass, and `Ground()`,
`Rect()`, the builder's row-major order and its two skip counters keep their bodies. No lift,
origin or projection reaches the pass. A build whose subset is empty returns the built slice itself
and allocates nothing. Nothing here reads a clock, opens a window, or imports past the standard
library.

**done when** AC-3 and AC-5 hold and SC-3 and SC-5 with them, together with AC-9's builder half — a
nil bundle, absent tile words, and cells at `(0,0)` and at the far corner — and SC-4's whole-list
comparison against the pre-story build. Then SC-8's period-only-gate mutant and its
dropped-re-anchor mutant are each applied, the whole tree run with the failing tests named,
reverted, and byte-identity confirmed.

## T5 — the window: one counter, one switch, one pass

**files** MODIFY `pkg/ui/viewer.go`, `pkg/ui/statics.go`, `pkg/ui/statics_test.go`,
`cmd/mapview/main.go`, `cmd/mapview/main_test.go`; ADD `pkg/ui/objectanim_test.go`

DD-7, DD-8, DD-10 — FR-5, FR-7, FR-8.

**fences** `advanceAnimation`, `SetAnimated`, `SetRate` and `SetSpeedIndex` gain no clause and no
stop, and `Ticker` is not edited. The texture cache key is not widened. Both geometry lists stay
built at construction and neither is rebuilt; the scratch buffer is the viewer's own and is never
handed out. No second flag selects a counter, and the standalone viewer gains no rate, stop or
switch it does not have today.

**done when** AC-8 holds and SC-7 with it, the counter read through both consumers and compared as
one value, and 0041's digest and byte-form checks re-run unmoved. Then SC-8's shifted-counter
mutant is applied, the whole tree run with the failing tests named, reverted, and byte-identity
confirmed.

## T6 — the raster tool: one still, at a counter the caller names

**files** MODIFY `cmd/terraintool/main.go`, `cmd/terraintool/statics_test.go`;
ADD `cmd/terraintool/objectanim_test.go`

DD-9, DD-10 — FR-7, FR-8.

**fences** no existing flag is renamed or given a second meaning; the scale-1 refusal, the ground
check and the three marker passes are untouched. The descriptor's tokens keep their order and
spelling, the new count riding beside the placement count. One image is written, at one counter,
and the statics loop gains no second pass.

**done when** AC-7 holds and SC-6 with it, the raster pixels and the standalone-image pixels
compared byte for byte at one counter and at a second counter that selects a different frame.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, AC-1 | DD-1, SC-1 |
| T2 | FR-2, FR-3, AC-2, AC-4, AC-9 | DD-3, SC-2, SC-8 |
| T3 | FR-1, FR-2, AC-6 | DD-2 |
| T4 | FR-4, FR-6, FR-9, AC-3, AC-5, AC-9 | DD-4, DD-5, DD-6, DD-8, SC-3, SC-4, SC-5 |
| T5 | FR-5, FR-7, FR-8, AC-8 | DD-7, DD-8, DD-10, SC-7 |
| T6 | FR-7, FR-8, AC-7 | DD-9, DD-10, SC-6 |
