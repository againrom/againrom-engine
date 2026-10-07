# Tasks — two relations, a window, a route that is state, and the tick that spends it

Legend: **files** what the entry may change — a permission, not a prediction · **fences** what it
must not do · **done when** the observable it leaves behind. Entries land in ascending order, each
depending only on those before it. SC-10's mutants are one to an entry, applied to production
code, measured over the whole tree and reverted by the entry that owns one; a kill is claimed only
where it was run. An entry owning none says so and breaks something load-bearing of its own.

## T1 — one search of each kind, over either relation

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/optimised.go`, `pkg/sim/route_test.go`,
`pkg/sim/optimised_test.go`; ADD `pkg/sim/relation_test.go`

DD-1 — FR-1.

**fences** no second copy of either search, and no branch inside one that chooses a relation per
cell: the relation arrives as a parameter and is read at the offer. The corner test asks the same
relation as the cell test. `World` gains no field, `step.go` is not opened — nothing calls the
terrain-only arm yet — and no byte, digest or pin moves.

**done when** both searches are driven directly over each relation on fixtures small enough to
read: a cell a unit stands on is open to the terrain arm and closed to the other, and every route
this package already pins is reproduced cell for cell by the unit-aware arm. Then SC-10's
occupancy-reading-far-search mutant is applied, the whole tree run with the failing tests named,
reverted, and byte-identity confirmed.

## T2 — the window arrives and the region goes

**files** MODIFY `pkg/sim/optimised.go`, `pkg/sim/route.go`, `pkg/sim/optimised_test.go`,
`pkg/sim/route_test.go`, `pkg/sim/optimality_test.go`, `pkg/sim/wall_test.go`

DD-2 — FR-2, FR-4.

**fences** `region`, `searchRegion` and `regionMargin` are **deleted**, not widened and not kept
behind a flag. A search given no window sweeps the whole map, and the walk back refuses exactly
the cells the flood refused, so no route can leave what could not be labelled. Nothing here reads
a sub-goal — the window is a parameter and its caller is T5's.

**done when** SC-4's fixture holds in both modes, its gap outside any start-target rectangle grown
by 8, and a windowed search over a large map is asserted to label no cell outside the square and
no more than 289 in all. Then SC-10's window-on-the-far-search mutant is applied, the whole tree
run with the failing tests named, reverted, and byte-identity confirmed.

## T3 — routes on the world, declared canonical

**files** MODIFY `pkg/sim/world.go`, `pkg/sim/nostate_test.go`, `pkg/sim/world_test.go`

DD-3, DD-4 — FR-3.

**fences** `Entity` gains nothing, so it stays free of pointers and slices. `NewWorld` keeps its
five parameters and every existing call site compiles unchanged. `binary.go`, `hash.go` and their
pins are not opened — the field exists here and is encoded in T4 — and `step.go` is not opened, so
nothing writes a route yet.

**done when** the pinned field table carries the new field and reads as this story's declaration
that it is canonical; a world built by any constructor path holds no route for any unit; and
`Entities()` still hands back copies that reach nothing of the world.

This entry **owns no mutant**, and that is stated rather than filled: what it adds is a field
declaration, and every mutant that could reach it is a mutant of the encoding T4 lands or of the
tick T5 lands, where each is measured.

## T4 — the byte form at version 4, and everything it refuses

**files** MODIFY `pkg/sim/binary.go`, `pkg/sim/binary_test.go`, `pkg/sim/hash_test.go`,
`pkg/sim/nostate_test.go`, `pkg/sim/gridform_test.go`, `pkg/sim/malformed_test.go`;
ADD `pkg/sim/routeform_test.go`

DD-5 — FR-3.

**fences** no path migrates a version-3 form. The declared entity count is checked, not trusted:
the record span is multiplied in `int64` and the route section is required to consume the
remainder exactly. `step.go` is not opened, so every route encoded here got there by being
decoded. The new digest pin is recomputed by hand from the pinned bytes, never copied out of a run
of this package.

**done when** SC-1 holds: every offset and width re-pinned at the new version, each of the five
refusals carried by a case of its own, and a world with no route compared byte for byte with one
whose routes are all empty. Then SC-10's dropped-route-section mutant is applied, the whole tree
run with the failing tests named, reverted, and byte-identity confirmed.

## T5 — the tick: staleness, the sub-goal, the near search, the move, the consumption

**files** MODIFY `pkg/sim/step.go`, `pkg/sim/step_test.go`, `pkg/sim/run_test.go`,
`pkg/sim/blocked_test.go`, `pkg/sim/occupancy_test.go`, `pkg/sim/contention_test.go`,
`pkg/sim/arrival_test.go`, `pkg/sim/relaxation_test.go`, `pkg/sim/mode_test.go`;
ADD `pkg/sim/subgoal_test.go`

DD-6 — FR-4, FR-5, FR-6, FR-9.

**fences** the three staleness tests are the only path to a far search, and they are asked once
per unit per tick, before the near search and nowhere else. The occupancy plane keeps moving in
the statement that moves a unit. Nothing counts ticks or re-searches, and no field is added to
carry a period. Give-up and the far failure are not touched here — they are T6's — and a landed
test this contract genuinely contradicts is rewritten to it rather than deleted.

**done when** SC-7 holds: the sub-goal read off a hand-written route, a unit landing on route cell
2 seen to drop cells 0 to 2, a unit that stepped elsewhere seen to keep every cell, and each of
the three staleness cases its own fixture with the far searches counted. Then SC-10's
sub-goal-is-the-last-cell mutant is applied, the whole tree run with the failing tests named,
reverted, and byte-identity confirmed.

## T6 — a far failure ends the order, a near failure stalls

**files** MODIFY `pkg/sim/step.go`, `pkg/sim/stall_test.go`, `pkg/sim/blocked_test.go`;
ADD `pkg/sim/giveup_test.go`

DD-7 — FR-7.

**fences** the two failures are told apart by which search returned nothing and by nothing else —
no flag, no counter and no second predicate. A far failure writes the target fields and the route
and touches no stall count; a near failure writes the stall count and nothing else. The threshold
stays the one constant it is.

**done when** SC-5 holds: three unservable orders each cleared on the first tick with the stall
left at zero, and the held-cell group seen to advance, to stall only once it can get no closer,
and to lose target and route on its sixteenth stalled tick. Then SC-10's far-failure-stalls mutant
is applied, the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T7 — the fork, measured in the tree rather than argued in the docs

**files** ADD `pkg/sim/routefork_test.go`

SC-3 — FR-3.

**fences** no production file is opened. The corpus is built in test code from a seeded integer
generator, reads no install and no clock, and the terrains are varied by size and by blocked
fraction rather than by one shape repeated. The test reports its counts and asserts the property
it is measuring, never that the count is zero — a zero would be a finding about the searches, not
a pass.

**done when** SC-3 holds: at least 100 000 (route, cell) pairs recomputed and compared with their
tails in canonical mode, with disagreements and outright refusals reported separately. Then
SC-10's dropped-staleness-test mutant is applied, the whole tree run with the failing tests named,
reverted, and byte-identity confirmed.

## T8 — the world decoded beside itself, walked to arrival

**files** ADD `pkg/sim/routetrip_test.go`; MODIFY `pkg/sim/replay_test.go`

DD-3 — FR-3, FR-9.

**fences** no production file is opened. The second world of every pair is built only by decoding
the first's bytes, never by constructing an equal one, and the two are compared at **every** tick
rather than at the end of a run. The corpus holds terrains that force a detour, since a straight
walk cannot show a route diverging.

**done when** SC-2 and SC-8 hold: equal digests at every later tick in both modes, and the
all-passable arrival and the crossing pair reproduced. Then SC-10's route-left-on-give-up mutant
is applied, the whole tree run with the failing tests named, reverted, and byte-identity
confirmed.

## T9 — what a tick actually costs, counted rather than timed

**files** ADD `pkg/sim/counted_test.go`

— FR-2, FR-4, FR-6, FR-8.

**fences** no production file is opened, and nothing here reads a clock: this entry counts, and
T10 times. The labelled cells are read off the scratch's own touch list rather than re-derived,
the searches are counted by kind, and the departures are counted from the contract's own test
rather than assumed to be zero.

**done when** SC-6 holds: a near search on 256x256 labels at most 289 cells and none outside its
window where a far search there labels more, and over a 64x64 run of eight units the far searches
equal the units ordered plus the departures recorded, with no unit-tick running two near searches.
Then SC-10's deleted-window mutant is applied, the whole tree run with the failing tests named,
reverted, and byte-identity confirmed.

## T10 — the harness's second window, and the two fixtures whose claim this story changed

**files** MODIFY `pkg/sim/grouporder_test.go`

DD-8 — FR-8.

**fences** the three shipped shapes keep their builders, their sizes and their group counts, so
their figures stay comparable with the ones already recorded; what is added is a tick count and
one longer run of the existing 256x256 shape. The sealed and held fixtures are rewritten to assert
what the contract now says — movers that walk, and stall only at the end — rather than deleted for
failing.

**done when** SC-9's four shapes report milliseconds per tick over the 16-tick window and the
256x256 group over 256 ticks besides, each fixture asserting the shape its benchmark claims. Then
SC-10's ninth is applied — the near search's budget slack moved from 3 to 5 — and reported as what
it is, a **survivor**, with the whole tree green: the window binds first, which is the measurement
DD-2 owes. Hunt no variant that kills it. Revert it and confirm byte-identity.

## T11 — the fork assertion outlives the budget it was built on

**files** MODIFY `pkg/sim/routefork_test.go`

DD-11 — FR-3, SC-15.

**fences** no production file is opened and no budget moves: this entry lands **before** T13, so
the tree is green either side of it. The corpus, its shapes and its seeds are untouched — a corpus
tuned until an assertion survives measures the tuning. The refusal count is still computed and
still reported; only what is *required* of it changes.

**done when** the file asserts the property it was always evidence for — a stored route's tail is
not recoverable from its own later cells — with differing and refused counts reported separately,
and the figures the shrinking budget produced recorded beside them as what that budget was worth.

## T12 — the budget-stop fixtures move to a bound both budgets refuse

**files** MODIFY `pkg/sim/route_test.go`, `pkg/sim/wall_test.go`

DD-9 — FR-10.

**fences** no production file is opened and no budget moves; this entry lands **before** T13 and
is green under the budget in the tree today. Nothing here relaxes an assertion to make room: a
fixture whose detour the flat budget will reach is **replaced** by one neither budget reaches, not
widened. The wall fixture keeps its gap, its wall and its arrival — only the clause naming a
budget of 50 for a walk of 40 goes, and what replaces it says what the fixture is for without
naming a number this revision deletes.

**done when** the budget stop is witnessed by a corridor whose only route needs more than **1000**
rings, refused under both budgets and reachable in optimised mode, and the three-refusals table's
budget-spent row is driven from it; and the wall fixture asserts its detour's own hop count in
place of an allowance.

## T13 — the far search's budget stops being a function of the distance

**files** MODIFY `pkg/sim/route.go`, `pkg/sim/step.go`, `pkg/sim/route_test.go`,
`pkg/sim/relaxation_test.go`, `pkg/sim/relation_test.go`, `pkg/sim/optimised_test.go`;
ADD `pkg/sim/budget_test.go`

DD-9, DD-10 — FR-10, FR-12, AC-14.

**fences** `optimised.go` is not opened: it has no generation count and MUST NOT gain one. The
near search's slack, relation and window do not move, and **no test of the goal's own openness is
added** — the refusal already standing before the budget is consulted is the gate. No field,
section or version byte moves: `binary.go`, `hash.go` and `world.go` stay shut. Three comments go
false as this lands and are repaired with it: the shared budget form, the label plane's
unclamped-target justification, and `step.go`'s shrinking budget.

**done when** both rules are driven directly on fixtures small enough to read — a far search
reaching a goal the old allowance could not, a near search still bounded by `max(3, D>>2) + D` and
still pinning the **reach** its budget stops the flood at, not the window's 289 — and every pin in
this tree is unmoved, the version byte where FR-3 left it. AC-14's recorded run is pinned
digest by digest from the tree **before** the budget moves, as literals never regenerated from the
tree they judge. Then SC-14's near-search mutant is applied, the whole tree run with the failing
tests named, reverted, and byte-identity confirmed.

## T14 — the channel, and the order that could not be given

**files** ADD `pkg/sim/channel_test.go`

SC-12 — FR-10, AC-12.

**fences** no production file is opened. The fixture asserts its own claim before it asserts an
arrival, and the **old** budget is driven from the test rather than described, so the refusal it
used to give is witnessed here and not recalled. Nothing here reads a clock or counts
milliseconds.

**done when** SC-12 holds: one order carries the unit across in **both** modes, on no
ground-blocking cell at any tick, over a crossing asserted sealed and a way round asserted longer
in rings than the old allowance; under that old allowance the same world is seen to refuse in
canonical while arriving in optimised; and FR-10's residual is witnessed beside it, a route over
1000 rings still refused in canonical where optimised finds it. Then SC-14's scaled-budget mutant
is applied, the whole tree run with the failing tests named, reverted, and byte-identity confirmed.

## T15 — what the flat budget costs, counted first and timed second

**files** MODIFY `pkg/sim/counted_test.go`, `pkg/sim/grouporder_test.go`

SC-13 — FR-11, AC-13.

**fences** no production file is opened. The fifth shape is added **beside** the four rather than
replacing one, and the four keep their builders, sizes and group counts, so every figure already
recorded stays comparable. The counts come off the scratch's own touch list. The sealed goal is a
cell that is **open and enclosed by blocking terrain**: a goal that itself blocks ground is
refused before any sweep and would measure nothing, and the shipped sealed shape is sealed by
*units*, which the far search cannot see. Neither file may name a symbol whose signature moved
inside this revision, or the two columns are two measurements rather than a before and an after.

**done when** SC-13 holds: labelled cells reported for a sealed goal near and far and for a search
that succeeds, at this revision and at the one before it; the four shipped shapes at or under
FR-8's ceilings; and the fifth — forty units ordered to a near sealed cell — at or under FR-11's
mean with its worst single tick reported beside it, its fixture asserting that the goal really is
unreachable and that every order really is cleared in the first tick.

## Traceability

| Task | Spec | Plan |
|---|---|---|
| T1 | FR-1 | DD-1 |
| T2 | FR-2, FR-4, AC-1 | DD-2, SC-4 |
| T3 | FR-3 | DD-3, DD-4 |
| T4 | FR-3, AC-5, AC-6 | DD-5, SC-1 |
| T5 | FR-4, FR-5, FR-6, FR-9, AC-8 | DD-6, SC-7 |
| T6 | FR-7, AC-2, AC-3 | DD-7, SC-5 |
| T7 | FR-3 | SC-3 |
| T8 | FR-3, FR-9, AC-7, AC-9 | DD-3, SC-2, SC-8 |
| T9 | FR-2, FR-4, FR-6, FR-8, AC-4 | SC-6 |
| T10 | FR-8, AC-10 | DD-8, SC-9 |
| T11 | FR-3 | DD-11, SC-15 |
| T12 | FR-10 | DD-9 |
| T13 | FR-10, FR-12, AC-14 | DD-9, DD-10, SC-14 |
| T14 | FR-10, AC-12 | SC-12 |
| T15 | FR-11, AC-13 | SC-13 |
