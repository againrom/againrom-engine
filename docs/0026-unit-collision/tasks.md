# Tasks — deterministic unit cell occupancy

Legend: **files** the task may change · **fences** what it must not do · **done when** the observable
it must leave behind. Every entry is an implementation task; they land in ascending order and each
depends only on entries before it. **(small)** marks one light enough to pair with a neighbour.
Intended batching: T1 alone, then T2+T3 to one agent and T4+T5 to another, T6 alone. T1 is the only
entry that changes shipped behaviour, and nothing pairs across it.

## T1 — the rule: one desired cell, computed once, tested once, written once

**files** MODIFY `pkg/sim/step.go`; ADD `pkg/sim/occupancy_test.go`

DD-1, DD-2, DD-3, DD-4, DD-7 — FR-2, FR-3, FR-5, FR-6.

**fences** no field on `World` or `Entity`, no package-level variable, no exported symbol, no new
import; `binary.go`, `hash.go` and every `_test.go` already in the package left unedited; the
predicate takes a slice and a cell, and neither an index nor an id. `pkg/game` is not in the file
list: its driven suite is predicted green, and a guard that does fire there is that story's apparatus
and never this rule (R-2) — stop and report rather than soften either. SC-9's reversed move loop is
measured and reverted, never committed.

**done when** SC-1, SC-3 and SC-6 hold and SC-9 is measured. The contested cell goes to the named
lower id with the two contenders non-adjacent in id; the convoy advances as one body inside a single
tick; the high-id-leader mirror reproduces the contract's stretch-then-flow ticks one at a time; the
shipped walk table and self-order test pass unedited. Then, with the move loop reversed to descending
index, SC-1's and SC-3's tests are named individually as the ones that fail, while the package still
builds, vets, and passes both halves of the determinism wall.

## T2 — a blocked unit changes nothing of its own (small)

**files** ADD `pkg/sim/blocked_test.go`

DD-3, DD-7 — FR-4, P-2.

**fences** no non-test file opened. The blocker of the first case carries no target and is never
given one. The blocked unit's position, `HasTarget`, `TargetX` and `TargetY` are compared one field
at a time, never through a struct or byte-form equality. Each case asserts that the blocked unit
still holds a target naming a cell it has not reached, so "unmoved" cannot be satisfied by a unit
with nothing left to do.

**done when** SC-4 holds: over a target-less blocker and over two units each ordered onto the other's
cell, all four fields unchanged on their own comparisons, both units of the pair asserted, and the
pair held across more than one tick so the standstill is shown to be stable rather than late.

## T3 — all-or-nothing on the one desired cell (small)

**files** MODIFY `pkg/sim/blocked_test.go`

DD-3 — FR-2, P-4.

**fences** no non-test file opened. The blocked case leaves both orthogonal neighbours toward the
target free; the mirror holds one orthogonal neighbour and leaves the diagonal free. Neither reuses
T2's positions, and neither asserts one coordinate where the criterion is about both.

**done when** SC-5 holds: with the desired diagonal cell occupied neither coordinate changes, and in
the mirror the unit still steps diagonally — so a rule that slides along whichever axis is free and a
rule that tests more cells than the one desired each fail a case of their own.

## T4 — the invariant over a whole run (small)

**files** ADD `pkg/sim/contention_test.go`

DD-6, DD-7 — FR-1, FR-6, P-1.

**fences** no non-test file opened. The contenders are non-adjacent in id, with an uninvolved unit
standing between them in the slice. The sweep's start cells and targets come from a generator the
test constructs and seeds with a constant; the world's own generator is never advanced. Distinctness
is asserted after every tick, not once at the end. A reversed move loop survives everything in this
entry by design — SC-9 owns that measurement, so do not add a criterion here to kill it.

**done when** SC-2 holds: at no tick of either run do two units share a cell, a re-run of the seed
reproduces every trajectory, and both runs assert that at least one resolution was actually blocked,
so neither a rule that never blocks nor a fixture that never contends can pass empty.

## T5 — the malformed start, and the count that may not rise (small)

**files** ADD `pkg/sim/malformed_test.go`

DD-5, DD-7 — FR-8, P-5.

**fences** no non-test file opened, and nothing here separates, repairs or refuses a co-located pair.
The self-order is carried by one of the two co-located units, not by the third. Occupant counts are
compared cell by cell over every cell either state touches, before against after, on every tick —
never on the shared cell alone.

**done when** SC-7 holds: the self-order cleared with the co-occupant still standing there, the third
unit unmoved with its target intact, the shared cell still holding exactly the original two, a re-run
of the same world and commands identical field for field, and no cell's count anywhere rising above
the greater of one and the count it already held — with at least one tick of the run asserted to have
actually blocked.

## T6 — no new state, seen where the byte form cannot see

**files** ADD `pkg/sim/nostate_test.go`

DD-2 — FR-7, FR-9, P-3, C-1.

**fences** `binary_test.go` and `hash_test.go` carry AC-12 and stay unedited — a pin edited by the
task it judges witnesses nothing. The field-set pin is a literal table of name, type and order for
`World` and `Entity` both, compared for exact equality rather than searched for an absence. The two
worlds of the round-trip take the same commands and are compared at every tick.

**done when** SC-8 holds: the pinned bytes, encoded length and digest unchanged and the pinned form
still decoding; a field added to either struct failing the pin; and a world marshalled at every tick
of a contended run, decoded into a second world and stepped on, agreeing digest for digest with the
first to the end of the run — so occupancy kept anywhere the byte form does not reach, a
package-level index included, diverges rather than passing.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-2, FR-3, FR-5, FR-6 | DD-1, DD-2, DD-3, DD-4, DD-7, SC-1, SC-3, SC-6, SC-9 |
| T2 | FR-4 | DD-3, DD-7, SC-4 |
| T3 | FR-2 | DD-3, SC-5 |
| T4 | FR-1, FR-6 | DD-6, DD-7, SC-2 |
| T5 | FR-8 | DD-5, DD-7, SC-7 |
| T6 | FR-7, FR-9 | DD-2, SC-8 |
