# Plan — deterministic unit cell occupancy

## Baseline

`Step`'s move phase walks `w.entities` forward and writes each position **in place**. The slice is
strictly ascending by id — `NewWorld` sorts and rejects a duplicate, `UnmarshalBinary` refuses a
non-ascending record — so walking it forward is walking ascending id, and a position written at index
`i` is already visible at `i+1`. `step1` is the three-valued sign of the difference, per axis.

`encode` is the single traversal behind both `MarshalBinary` and `Hash`: a 29-byte header, 25-byte
records, FNV-1a over exactly those bytes, with the world's RNG state at offset 9 — so a draw from the
generator would move the digest, and `TestStepAdvancesTheTickExactlyOnceAndDrawsNoRandomness` already
pins that state unmoved by a step. `binary_test.go` pins every field's offset and width, a 104-byte
form of a three-entity world, and that form's decode; `hash_test.go` pins the digest
`0x2f5f68a68add08ad` and recomputes it from the pinned bytes. `TestWorldExportedMethodSetIsPinned`
pins `*World`'s exported method set by name. Nothing pins either struct's **field set**. The shipped
files hold three constants and no package-level variable, and no map on any path that touches a world.

Co-located worlds are reachable through shipped code: `mapload.unitCell` is
`int32(u.X>>8), int32(u.Y>>8)`, `NewWorld`'s only error is a duplicate id, and `UnmarshalBinary`
inspects shape and id order but no position.

Outside `pkg/sim`, `Step` is reached by `pkg/game`'s driver and by two test suites. Measured on a
prototype of the loader's cell rule and `mapload.Schedule`: `pkg/game`'s 12x9 four-unit fixture, over
the 575 ticks its digest test drives, spends 78 entity-pair-ticks with two units standing on one
cell; `pkg/mapload`'s fixture contends on no tick at all.

## Design decisions

### DD-1 — the entity slice **is** the occupancy relation, read where it stands

Resolution stays the single forward pass it already is, and occupancy is a predicate over
`w.entities` evaluated at the moment a unit is resolved: at index `i` the entities before `i` hold
their post-move cells and those after it their pre-move cells, which is FR-3's incremental model
exactly, with nothing to build, update or invalidate.

Rejected: **a frozen snapshot of the tick's start positions with an arbitrating resolver.** Equally
deterministic and equally integer-only, and measured on a prototype of both rules it fails AC-4: the
convoy of the contract's second worked example advances `3,1,0 / 4,2,0 / 5,3,1` instead of
`3,2,1 / 4,3,2 / 5,4,3`, three ticks for one tick's work, because each follower reads the cell ahead
as held by a leader that has already left it. Choosing it would not reshape the rule, it would delete
a criterion. The frozen snapshot a later pathfinding story wants is for the **path search**, whose
answer must not depend on how far through a tick it is asked; that is a different consumer from
contention resolution, and conflating the two is how this decision gets quietly reversed.

Rejected: **a per-cell index built at tick start and updated as each unit moves.** It restores the
vacated-cell behaviour at the price of a second representation of a position for the first to drift
from — an update that adds the new cell without clearing the old one passes AC-2 and fails AC-4 — for
no gain the entity counts in play justify (R-1).

### DD-2 — the predicate materialises nothing, which is what makes C-1 structural

An unexported function beside `step1` takes the entity slice and a cell and reports whether any
entity stands there: a comparison over positions, no allocation, no receiver, nothing that outlives
the call. C-1 and FR-9 are then not disciplines to hold — there is no object for `encode` to reach,
none for `UnmarshalBinary` to rebuild, none to keep true across a restore. `World` and `Entity` gain
no field, the package gains no variable, `binary.go` and `hash.go` are not opened, and no exported
surface appears; `TestWorldExportedMethodSetIsPinned` fails on a method added here.

Rejected: **a method on `*World`.** It publishes an "is this cell free?" query the contract does not
define, and it is the wrong question the moment terrain also answers it (C-3). Rejected: **a
package-level scratch buffer reused per tick.** Nothing of it enters the digest, so a criterion built
on the byte-form pins alone would pass it (SC-8 is written against exactly that), and it makes `Step`
a shared-state call for no measured gain.

### DD-3 — one desired cell: computed once, tested once, written once

Both axes' steps are computed first, giving the one desired cell FR-2 names. The two arms of the
resulting branch are P-4's two outcomes and there is no third: either both coordinates are written
and the arrival check runs against the new cell, or nothing is assigned at all. A blocked unit's
position and three target fields are not restored, they are never written — which is what makes P-2 a
shape rather than a discipline. `Step`'s own phase-2 sentence is amended with the branch: a doc
comment left promising an unconditional move states a contract this package no longer has.

Rejected: **testing each axis and taking whichever is free.** FR-2 forbids it, and it turns a
diagonal order into a wall-hugging drift no criterion over positions can pin. Rejected: **clearing a
blocked unit's target.** It makes an obstacle indistinguishable from an arrival to every reader of
`HasTarget` — the one distinction a caller cannot recover for itself.

### DD-4 — the zero-distance case is decided before the occupancy test, never inside it

When both steps are zero the desired cell is the unit's own, which FR-2 makes not a move: that branch
goes straight to the arrival check and clears the target. Because every cell that does reach the test
therefore differs from the resolving unit's own, a unit can never be its own blocker, and the
predicate needs no self-exclusion, no index and no id argument.

Rejected: **testing the current cell and excluding self by identity.** It is right in a well-formed
world and wrong in a malformed one: the co-occupant reads as a blocker and the self-order is
stranded, twice over where both co-located units are ordered onto the cell they stand on. AC-10 kills
the variant that forgets the self-exclusion; only AC-11's malformed fixture kills this one, which is
why SC-7 carries it.

### DD-5 — no precondition, and P-5 instead of a repair

Nothing in the design asks whether the world it was handed is well-formed, because shipped code can
hand it one that is not. P-5 then follows from the test alone: a unit enters a cell only when that
cell holds no occupant, so across a tick a cell either keeps its count or loses one, and no cell's
count rises above one it did not already carry. FR-8 is a consequence of the rule rather than a
second rule in it, and a malformed pair parts only if movement happens to part it.

Rejected: **refusing a co-located world at construction or decode.** A new error on two shipped entry
points, and unless both gain it together `NewWorld` would reject what `UnmarshalBinary` accepts; it
also converts a loader defect into a load failure in the owner's hands. Rejected: **parting a
co-located pair during advancement.** Nothing decoded says which unit goes where, so the repair would
be invented state, and it is out of scope by name.

### DD-6 — the sweep's randomness is the test's own, never the world's

AC-9's fixed-seed sweep draws its start cells and targets from an `rng` value the test constructs and
seeds with a constant. The world's own generator is never advanced, so the sweep asserts nothing
about a state it moved and the digest comparison inside it stays meaningful. Rejected: **a second
generator brought in for the test.** The package already owns one whose whole subject is being the
only one; a sweep is not a reason to explain a second.

### DD-7 — the fixtures are shaped so that a wrong rule cannot pass by coincidence

Four shapes, each aimed at a specific wrong implementation. Contenders **non-adjacent in id**, an
uninvolved unit between them, so a rule that compares a unit only against its predecessor in the
slice answers AC-1's three-way case wrongly. A blocker carrying **no target**, so an occupancy
relation assembled out of movers alone fails AC-3. Every contention criterion asserting that a block
**actually occurred** in that run, so neither a rule that never blocks nor a fixture that never
contends passes empty. And the resolution order claimed only where identity is asserted: AC-1's "no
two share a cell" holds under a descending order too, so the order's witnesses are AC-2's named
winner and AC-4's convoy, and the claim is discharged by running the mutant (SC-9).

Rejected: **AC-1 and AC-9 as the order's witnesses.** Both pass under either order, which is how the
ascending-order requirement went unwitnessed from 0019 to here.

## Risks

- **R-1** The predicate is linear in the entity count and runs once per moving unit, so per-tick cost
  is quadratic in the units on a map. Accepted at the counts the loader produces; the contract is
  stated over resulting positions and not over a structure, so an index can replace the scan later
  with no criterion rewritten.
- **R-2** The rule changes what a shipped review build shows. Measured on the prototype over
  `pkg/game`'s fixture and schedule: the 78 co-located pair-ticks become none, at 102–126 blocked
  ticks per entity across 575, and the four laps desynchronise. The tests there that pin cells pin
  them inside windows where no two entities contend, and the driven-versus-headless digest test
  compares like with like; should a liveness guard there fire, the fixture is that story's apparatus
  and this contract is not the thing to weaken.
- **R-3** Deadlock is now reachable in a build the owner runs — two units ordered onto each other's
  cells, or a cycle of them, stand forever. Disclosed by the contract and owned by the routing story;
  the cost is that a frozen pair reads as a bug rather than as the absence of a rule.

## Success criteria

- **SC-1** AC-2 holds in full over DD-7's non-adjacent contenders (FR-3).
- **SC-2** AC-1 and AC-9 hold in full, AC-1's contenders likewise non-adjacent; the sweep's targets
  come from the test's own seeded generator, and both runs assert that at least one resolution was
  actually blocked (FR-1, FR-6, P-1).
- **SC-3** AC-4 holds in full; and the same three units re-ordered so the leader carries the highest
  id reproduce the contract's stretch-then-flow example tick for tick, so the disclosed limitation is
  witnessed rather than asserted (FR-3).
- **SC-4** AC-3 and AC-5 hold in full, AC-3's blocker target-less per DD-7, and every one of the
  blocked unit's three target fields compared individually (FR-4, P-2).
- **SC-5** AC-8 holds in full; and the mirror case — the diagonal free while one orthogonal
  neighbour toward the target is occupied — still moves diagonally, so the criterion tells
  all-or-nothing apart from a rule that tests more cells than the one desired (FR-2, P-4).
- **SC-6** AC-6 and AC-10 hold in full, over the shipped walk table and self-order test unchanged
  (FR-2, FR-5, FR-6).
- **SC-7** AC-11 holds in full, its self-order carried by one of the two co-located units; and P-5 is
  checked as an occupant-count comparison over every cell before and after every tick of a malformed
  run, not on the shared cell alone (FR-8).
- **SC-8** AC-7 and AC-12 hold in full; `World`'s and `Entity`'s field sets are pinned by name, type
  and order, and a world marshalled at every tick of a contended run and decoded into a second world
  steps on to the same digests as the first — so occupancy held anywhere the byte form does not reach
  shows up as a divergence rather than as a passing pin (FR-7, FR-9, P-3).
- **SC-9** The resolution order is observable: with the move loop reversed to descending index, SC-1
  and SC-3 fail, which is the ascending-order requirement discharged as a measurement rather than a
  comment. Both halves of the determinism wall stay green over a package that gained no import (FR-3).

## Traceability

| Spec | Design | Checked by |
|---|---|---|
| FR-1 | DD-1, DD-3 | SC-2 |
| FR-2 | DD-3, DD-4 | SC-5, SC-6 |
| FR-3 | DD-1, DD-7 | SC-1, SC-3, SC-9 |
| FR-4 | DD-3 | SC-4 |
| FR-5 | DD-3, DD-4 | SC-6 |
| FR-6 | DD-3, DD-4 | SC-2, SC-6 |
| FR-7 | DD-1, DD-2, DD-6 | SC-8 |
| FR-8 | DD-5 | SC-7 |
| FR-9 | DD-2 | SC-8 |
| C-1 | DD-2 | SC-8 |
