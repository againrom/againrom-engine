# Tasks — the grid, the two searches, and the tick that walks one cell

Legend: **files** what the task may change · **fences** what it must not do · **done when** the
observable it leaves behind. Entries land in ascending order, each depending only on those before
it. Mutants are one to an entry, applied to production code, measured tree-wide and reverted by the
entry that owns one; a width here is an expectation under test, and a kill is claimed only where it
was run. An entry owning none breaks something load-bearing of its own.

## T1 — the mode and the grid arrive at construction

**files** MODIFY `pkg/sim/world.go`, `pkg/sim/world_test.go`, `pkg/mapload/fromalm.go`,
`pkg/mapload/fromalm_test.go`, `pkg/game/world_test.go`

DD-6 — FR-1.

**fences** `binary.go`, `hash.go` and their pins are not opened, so every shipped digest is unmoved
by this entry. `step.go` is not opened — nothing reads the grid yet. `Entity` gains no field. No
flag, nil case or second field records that a caller passed no grid. No second constructor and no
options struct. `pkg/sim` gains no import, and the one production call site is handed canonical mode
and no grid rather than a grid invented for it.

**done when** a no-grid world and an all-zero-grid world of equal bounds are equal field for field,
the stored grid included; a wrong length, a reserved bit and an undefined mode byte are each refused
on a case of its own with no world returned; and every call site names its mode. Then, with the
reserved-bit refusal deleted, the whole tree is run, the failing tests named, the deletion reverted
and the tree confirmed byte-identical.

## T2 — the byte form at version 3, and everything it refuses

**files** MODIFY `pkg/sim/binary.go`, `pkg/sim/world.go`, `pkg/sim/binary_test.go`,
`pkg/sim/hash_test.go`; ADD `pkg/sim/gridform_test.go`

DD-7, DD-9 — FR-1, FR-2.

**fences** `step.go` is not opened: the stall byte is carried and refused here, never raised — T6
makes it move. No path migrates a form of the previous version. The record count keeps coming from
the buffer's own length and not from the declared one. The new digest pin is recomputed by hand from
the pinned bytes, never copied out of a run of this package.

**done when** SC-1 and SC-2 hold, every offset and width re-pinned at the new header and record
lengths, and each refusal carried by a case of its own. Then SC-10's absent-grid mutant is applied,
the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T3 — the wave: enterability, the plane, the frontier, and the walk back

**files** ADD `pkg/sim/route.go`, `pkg/sim/route_test.go`

DD-1, DD-2, DD-3, DD-4, DD-5 — FR-3, FR-5.

**fences** no sub-package, no interface, no exported symbol, no `World` field, no package-level
variable, no new import. The plane and the frontiers are a caller's to make and pass, never a
receiver's to hold. `step.go` is not opened — nothing calls this yet, and the tests reach it
directly rather than by advancing a world. Nothing clamps the start into bounds, and no route
outlives the call that built it.

**done when** routes are pinned cell by cell on fixtures small enough to read by hand: one where two
frontier cells of a single generation reach a shared neighbour, one where the target is outside the
budget and the answer is no route, one start on a blocked cell and one out of bounds, and one where
the walk back has an equal orthogonal and an equal diagonal candidate at the same cell. Then
SC-10's budget-term mutant is applied, the whole tree run with the failing tests named, reverted,
and byte-identity confirmed.

## T4 — a route per tick, and one cell along it

**files** MODIFY `pkg/sim/step.go`, `pkg/sim/step_test.go`, `pkg/sim/occupancy_test.go`,
`pkg/sim/blocked_test.go`, `pkg/sim/contention_test.go`, `pkg/sim/malformed_test.go`,
`pkg/sim/nostate_test.go`, `pkg/sim/run_test.go`; ADD `pkg/sim/relaxation_test.go`

DD-3 — FR-4, FR-8.

**fences** the plane is made once per tick and passed into each search, not once per unit and not
once per world. The mode is not read yet — every unit routes by the wave, and the dispatch is T8's.
Nothing counts a stall or clears a target for stalling; that is T6's. A route reaches no field, no
byte and no digest, so `binary.go`, `hash.go` and their pins stay unedited. `pkg/game` and
`pkg/mapload` are not in the file list: their driven suites are predicted green, and one that does
move is reported rather than softened. A landed test this rule genuinely contradicts is rewritten to
the contract, not deleted.

**done when** SC-9 holds — the 5x5 fixture's route pinned cell by cell and its tick-1 digest with it
— and a unit beginning the tick already on its target neither searches nor steps. Then SC-10's
batched-wave mutant is applied, the whole tree run with the failing tests named, reverted, and
byte-identity confirmed.

## T5 — the second grid, where the two second-plane shapes part

**files** MODIFY `pkg/sim/relaxation_test.go`

DD-1 — FR-5.

**fences** no production file is opened, and the fixture already there is left as it stands: this
grid stands beside it rather than widening it. Its route and its digest are read off the contract by
hand, never off a run of the search under test. Nothing here changes what a mode does.

**done when** the 8x8 fixture pins its walked route cell by cell and its tick-1 digest. Then
SC-10's hybrid-wave mutant is applied, the whole tree run with the failing tests named, reverted,
and byte-identity confirmed.

## T6 — the stall count, and giving up inside the tick that reaches the threshold

**files** MODIFY `pkg/sim/step.go`; ADD `pkg/sim/stall_test.go`

DD-9 — FR-7.

**fences** the count lives on `Entity` and nowhere else — no parallel structure keyed by id, nothing
`UnmarshalBinary` has to rebuild. Neither search is opened, and the byte form's shape does not move.
Nothing here gives up per mode.

**done when** SC-4 holds: all three target fields compared one at a time at each of the sixteen
ticks, a unit seen returning its count to zero by advancing after a stall, and a unit holding no
target asserted to carry zero. No stored count reaches the threshold, so T2's refusal cannot fire on
a world this package produced — asserted over the run, not argued. Then SC-10's stall-reset mutant
is applied, the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T7 — optimised mode: the region, the heap, and the walk forward

**files** ADD `pkg/sim/optimised.go`, `pkg/sim/optimised_test.go`

DD-5, DD-8 — FR-6.

**fences** a function beside the wave in this same directory — no sub-package, no interface, no
exported symbol, no `World` field. Nothing here is reachable from `Step` yet: the dispatch is T8's,
and no shipped digest moves. The region is clipped from a rectangle wide enough for any cell a
target may name. No route is enumerated and no set of routes is sorted.

**done when** the search is pinned on fixtures small enough to read by hand: a corner its route may
not cut, a target reachable only by leaving the region and so refused, a start on a cell it may not
stand on, and two equal-cost routes separated by the order the contract names. Then SC-10's tenth is
applied — the heap key's `(y, x)` terms dropped — and reported as what it is: a **survivor**, with
the whole tree green. Hunt no variant that kills; the measurement is the discharge. Revert it and
confirm byte-identity.

## T8 — the mode drives the tick

**files** MODIFY `pkg/sim/step.go`, `pkg/sim/step_test.go`; ADD `pkg/sim/mode_test.go`

DD-5 — FR-5, FR-6, FR-8.

**fences** the mode is read where the route is chosen and nowhere else: no second dispatch, and no
resolution order, cost model or neighbour set that differs by it. It is read from the world, never
from a parameter, a build tag or a package variable. Neither search is edited here.

**done when** SC-6 holds, one world driving both modes. Then SC-10's corner-cut mutant is applied,
the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T9 — the wall, the gap, and the edge of the map

**files** ADD `pkg/sim/wall_test.go`

DD-4, DD-8 — FR-3, FR-4.

**fences** no production file is opened. The gap is placed so that both modes arrive; a wall that
defeats one of them is a fixture defect to fix here, never a give-up to record. Every unit is
checked after every tick rather than at the end of the run, and nothing here asserts a route — the
criterion is where a unit stands.

**done when** SC-3 holds in both modes, P-1 checked over every unit after every tick. Then SC-10's
region-growth mutant is applied, the whole tree run with the failing tests named, reverted, and
byte-identity confirmed.

## T10 — arrival, crossing, and the world decoded beside itself

**files** ADD `pkg/sim/arrival_test.go`, `pkg/sim/replay_test.go`

DD-3 — FR-4, FR-8.

**fences** no production file is opened. The all-passable run holds no second unit anywhere near the
path, so the tick count it pins is the contract's and not a detour's. The second world of the pair
is built only by decoding the first's bytes, never by constructing an equal one, and the two are
compared at every tick rather than at the end.

**done when** SC-5 and SC-8 hold, P-3 sampled over the decoded pair. Then SC-10's straight-accept
mutant is applied, the whole tree run with the failing tests named, reverted, and byte-identity
confirmed.

## T11 — the cost, against an enumeration that shares nothing with the solver

**files** ADD `pkg/sim/optimality_test.go`

DD-8 — FR-4, FR-5, FR-6.

**fences** no production file is opened. The enumeration shares no function, table or constant with
either search; a grid it cannot exhaust is too large for this entry. The canonical-worse half is
built to be worse, and a random sweep is not evidence for it.

**done when** SC-7 holds, and SC-10's diagonal-accept mutant is applied, the whole tree run with the
failing tests named, reverted, and byte-identity confirmed. This entry being the last, the SDD audit
then reports **one** enforced FAIL naming this story — every task landed, `verification.md` not yet
written — which is the design. Count the FAIL lines rather than accept the first: a second one is
this entry's to fix. That check exits nonzero, so the `&&` chain cannot carry this entry to the
push; run the steps after it, and the push, on their own.

## T12 — the harness, and the digests this tree carries before anything is made cheaper

**files** ADD `pkg/sim/grouporder_test.go`, `pkg/sim/preserved_test.go`

SC-11, SC-12 — FR-4, FR-8.

**fences** no production file is opened, so this entry moves no route, no digest and no measurement
it is about to take. The pinned digests are recorded from the tree as it stands here and are never
regenerated afterwards: a pin recomputed from the code it judges witnesses nothing. The benchmarks
are benchmarks, so `go test` compiles them and runs neither; what it runs is their fixtures.

**done when** the harness reports milliseconds per tick for a group ordered to a far cell at two map
sizes and five group sizes, and for eighty movers ordered onto a free cell nothing can reach; each
fixture is asserted to be the thing it claims — the group really walking and never arriving inside
the window measured, the sealed goal really enterable and really unreachable, since a benchmark over
a standstill measures nothing. Three runs past `stallLimit` are pinned digest by digest and each is
asserted to hold an advance, a hold and a give-up.

## T13 — enterability answered from the tick's own occupancy plane

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/optimised.go`, `pkg/sim/step.go`,
`pkg/sim/route_test.go`, `pkg/sim/optimised_test.go`, `pkg/sim/relaxation_test.go`;
ADD `pkg/sim/enterable_test.go`

DD-10 — FR-3, FR-8.

**fences** `World` gains no field and `nostate_test.go` is neither edited nor re-pinned — it is the
constraint, not an obstacle. The byte form, the digest and their pins are not opened. The reference
predicate is a test-only symbol no production path calls. The counts are moved where a unit is moved
and nowhere else: no second call site, and no rebuild inside a search.

**done when** SC-11's oracle half holds — the indexed predicate compared with the reference one unit
for unit and cell for cell, over randomised worlds and over mid-tick states walked one unit at a
time — and T12's pins are unmoved. Then the occupancy-move mutant is applied, the whole tree run with
the failing tests named, reverted, and byte-identity confirmed.

## T14 — a destination no route may enter fails before the sweep, not after it

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/route_test.go`

DD-11 — FR-5, FR-7.

**fences** `optimised.go` is not opened; it already refuses on this ground. Nothing else about the
wave moves — not the seed, not the stop conditions, not the walk back — and the answer to a search
whose start is its target is the one it was.

**done when** the witness holds: a refused destination leaves the scratch's touch list and frontier
empty, where the budget's own refusal and the sealed map's each leave them full, so the three
no-route paths are told apart by what was labelled rather than by which answer came back. T12's pins
are unmoved. Then the early-out mutant is applied, the whole tree run with the failing tests named,
reverted, and byte-identity confirmed — and it is reported that no behavioural test can see it.

## T15 — the shape the early refusal is actually for, and one harness for four trees

**files** MODIFY `pkg/sim/grouporder_test.go`

SC-12 — FR-4, FR-7.

**fences** no production file is opened. The existing two shapes are left exactly as they stand,
figures already taken against them; this one is a third beside them. Nothing here may call a symbol
whose signature moved inside this revision — the harness has to compile against every tree in its
range, or the four columns are four measurements rather than one.

**done when** a group ordered onto a cell another unit is standing on is measured in milliseconds per
tick beside the other two shapes, and its fixture is asserted: the destination really held, the
movers really refused on every tick, and the whole group giving up on the sixteenth.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1, AC-8, P-5 | DD-6 |
| T2 | FR-1, FR-2, AC-6, AC-7, AC-8, P-5 | DD-7, DD-9, SC-1, SC-2 |
| T3 | FR-3, FR-5 | DD-1, DD-2, DD-3, DD-4, DD-5 |
| T4 | FR-4, FR-8 | DD-3, SC-9 |
| T5 | FR-5 | DD-1 |
| T6 | FR-7, AC-2, P-2, P-4 | DD-9, SC-4 |
| T7 | FR-6 | DD-5, DD-8 |
| T8 | FR-5, FR-6, FR-8, AC-5 | DD-5, SC-6 |
| T9 | FR-3, FR-4, AC-1, AC-3, P-1 | DD-4, DD-8, SC-3 |
| T10 | FR-4, FR-8, AC-4, AC-10, P-3 | DD-3, SC-5, SC-8 |
| T11 | FR-4, FR-5, FR-6, AC-9 | DD-8, SC-7 |
| T12 | FR-4, FR-8 | SC-11, SC-12 |
| T13 | FR-3, FR-8 | DD-10 |
| T14 | FR-5, FR-7 | DD-11 |
| T15 | FR-4, FR-7 | SC-12 |
