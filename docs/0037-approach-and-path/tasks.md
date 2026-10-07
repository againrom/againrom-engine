# Tasks — a wave that settles, an order that takes what it settled for, and a line that shows it

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. Entries land in ascending order, each
depending only on those before it. The mutants named in the plan's success criteria are one to an
entry, applied to production code, measured over the whole tree and reverted by the entry that owns
one; a kill is claimed only where it was run.

## T1 — the machinery, present and unreached

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/step.go` and the search's own fixtures —
`route_test.go`, `budget_test.go`, `channel_test.go`, `counted_test.go`, `optimised_test.go`,
`relation_test.go`, `relaxation_test.go`, `routefork_test.go`; ADD `pkg/sim/approach_test.go`

DD-1, DD-2, DD-3, DD-4, DD-5 — FR-2.

**fences** both call sites in `Step` pass the non-settling rule, so no wave reaches the picker and
no mover's behaviour moves. `World` gains no field, no byte and no pin moves, and no landed
criterion anywhere in the tree is edited — an entry that has to touch one has turned the rule on
early. The picker imports nothing: the int32 range is written out rather than reached for.

**done when** the four arms of the budget rule are driven directly, including the pair of goals
close enough that the slack rather than the quarter-distance binds, so the two slacks are shown to
be 5 and 3 and not interchangeable. Every digest and byte-form pin in the tree is unmoved and the
whole suite is green with no test's expectations changed. SC-7's mutant — the flat override applied
whatever the goal is — is applied, the whole tree run with the failing tests named, and reverted.

## T2 — the far search settles, and the order takes the cell

**files** MODIFY `pkg/sim/step.go`, `pkg/sim/giveup_test.go`, `pkg/sim/grouporder_test.go`,
`pkg/sim/stall_test.go`, `pkg/sim/step_test.go`, `pkg/sim/wall_test.go`,
`pkg/sim/approach_test.go`, `pkg/mapload/routing_test.go`,
`docs/0045-two-tier-search/spec.md`

DD-6, DD-14 — FR-1, FR-3, FR-4, FR-5.

**fences** the near call site and the optimised search are not touched. The stall count, the give-up
limit, the cost model, the window and the byte form are not touched. A superseded criterion is
amended where it is written, never deleted, and the entry that renames a test records the old name
so the record citing it stays followable.

**done when** every criterion this entry's requirements carry has a fixture that runs, and the one
for the picker's own rule is built so that both rivals — the cheapest labelled cell there is, and
the first the scan meets — would answer differently on it. Then SC-5's and SC-6's mutants are
applied one at a time, the whole tree run with the failing tests named, and each reverted.

## T3 — where is this unit going

**files** MODIFY `pkg/sim/step.go`, `pkg/sim/world_test.go`, `pkg/sim/approach_test.go`

DD-7 — FR-6.

**fences** no search runs behind the query and no state is added for it to read. It is swept as a
reader, not declared a writer; the fail-closed arity test keeps its teeth, so a reader taking an
argument is declared with the argument to sweep it at rather than exempted.

**done when** the query answers nothing for an entity no world holds, nothing for one that has taken
no order, and the stored route cell for cell for one that has walked; a write through the answer
moves no digest, and two calls do not share a backing array.

## T4 — the route crosses the seam

**files** MODIFY `pkg/ui/overlay.go`, `pkg/game/world.go`, `pkg/game/cadence_invariance_test.go`,
`pkg/game/death_test.go`, `pkg/ui/life_test.go`

DD-8, DD-9 — FR-7.

**fences** the window tier still names no simulation type and gains no reader the world tier calls
back through. Nothing here derives a path, remembers one between ticks, or carries the entity's
current cell a second time. The conversion happens in the tier that already holds both a world and a
bundle, and nowhere else.

**done when** the per-entity value carries the remaining route for an entity under orders and
nothing for one that is not, the snapshot build stays repeatable when asked twice with no tick
between, and the three comparisons that lost `==` assert the same thing more strictly.

## T5 — the line

**files** MODIFY `pkg/ui/viewer.go`; ADD `pkg/ui/path.go`, `pkg/ui/path_test.go`

DD-10, DD-12, DD-13 — FR-8, FR-9.

**fences** no cache, no per-frame state, no second presence-or-life predicate — the one the orders
and the rim already use is called. The overlay pass type is not widened. Nothing is culled on an
endpoint.

**done when** each of the four exclusions is asserted on its own — not selected, no route, a corpse,
an id the snapshot dropped; the first segment runs from the unit's own cell to its route's head and
the rest along it, each endpoint at the camera's own transform of that cell's centre; and a leg with
both ends outside the view is still issued. Then SC-9's scope mutant — the preview widened past the
selection — is applied, the whole tree run with the failing tests named, and reverted. *(2026-08-01:
this entry also built the count cap FR-8 carried, and T6 removes it on the owner's ruling.)*

## T6 — every line, and the legs that cannot be seen

**files** MODIFY `pkg/ui/path.go`, `pkg/ui/viewer.go`, `pkg/ui/path_test.go`,
`pkg/sim/grouporder_test.go`

DD-11, DD-15 — FR-8a, FR-9a.

**fences** no number replaces the removed one: not a larger cap, not a fade, not a budget. The reject
is chosen on the leg's own bounding box and never on its endpoints, and it is the only thing that may
drop a leg. Nothing else in the overlay moves — the scope predicate, the anchor at the unit's own
cell, the draw order and the pass slice are untouched, and no state or cache is added.

**done when** the preview answers with every drawable unit at 31, 32, 33, two hundred and two
thousand of them, in the selection's own order, and with the three movers inside a selection of five
hundred; the segment build drops exactly the legs whose box misses the view and keeps the ones that
cross it from outside on both sides; and the cost is measured over the route length the tick itself
produces, at two group sizes and two zooms, on both arms. Then SC-9's two mutants and SC-12's
endpoint test are applied one at a time, the whole tree run with the failing tests named, and each
reverted.

## Traceability

T1 → FR-2, DD-1, DD-2, DD-3, DD-4, DD-5, SC-7 · T2 → FR-1, FR-3, FR-4, FR-5, DD-6, DD-14, SC-5,
SC-6 · T3 → FR-6, DD-7 · T4 → FR-7, DD-8, DD-9 · T5 → FR-8, FR-9, DD-10, DD-12, DD-13 ·
T6 → FR-8a, FR-9a, DD-11, DD-15, SC-9, SC-11, SC-12

FR-10 is a fence on every entry above and the deliverable of none: each states the pins it may not
move. SC-1, SC-2, SC-3, SC-4, SC-8 and SC-10 are the evidence stage's, being sweeps and measurements
over the finished tree rather than the product of any one slice.
