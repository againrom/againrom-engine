# Analysis — deterministic unit cell occupancy

## Intensity & terrain

| Axis | Declaration |
|---|---|
| Intensity | **spec-anchored / static** — the profile names simulation determinism by hand as a cross-cutting engine contract; no watcher tool exists, so it is discipline |
| Terrain — `pkg/sim` movement | **brownfield**: `Step`'s move phase ships and its behaviour changes here, so every altered outcome is pinned before it moves |
| Terrain — the occupancy rule itself | **greenfield**: no cell has ever had a notion of an occupant |

## The debt 0019 left, measured

0019's FR-3(b) requires a step to "advance every entity **in ascending entity id**", and **nothing in
that story could witness it**. Two entities' moves never interacted, so the order was unobservable
and the requirement was carried by a comment. Measured, not argued: a copy of `pkg/sim` with the
move loop reversed to descending index

    for i := len(w.entities) - 1; i >= 0; i-- {

passes `go test -trimpath -count=1 ./pkg/sim/` unchanged — the whole suite, `binary_test.go`,
`hash_test.go`, `run_test.go`, `step_test.go`, `world_test.go`. `TestStepIsIndependentOfInsertionOrder`
does not catch it: it proves the *slice* is ordered by id, which `NewWorld`'s sort does, not that the
loop walks it forward. No test outside `pkg/sim` can catch it either — one `sim.Step` call site in
`pkg/mapload`'s tests and one `sim.Run` in `pkg/game`'s, both over non-interacting entities.

This story is where that becomes observable, and it is the reason to claim the debt closed here
rather than to restate 0019's requirement. Two criteria fail under a descending order: the contested
single cell goes to the wrong unit, and the convoy stops flowing.

## Two resolution models, both deterministic, observably different

The contract resolves units one at a time in id order, each seeing the moves already resolved in the
same tick. The alternative is a frozen snapshot: every unit computes its desired cell against the
tick's start state, and a resolver then arbitrates by id. Both are integer-only and reproducible;
they disagree on a convoy. Measured on a prototype of both rules over three units at `x = 2, 1, 0`,
ids `0, 1, 2`, all ordered to `x = 9` — so the leader carries the lowest id:

| | tick 1 | tick 2 | tick 3 |
|---|---|---|---|
| incremental | `3, 2, 1` | `4, 3, 2` | `5, 4, 3` |
| frozen snapshot | `3, 1, 0` | `4, 2, 0` | `5, 3, 1` |

Under the snapshot each follower reads the cell ahead as still held by a leader that has already
left, so the line advances one unit per tick and takes three ticks to do what the contract does in
one. Choosing the snapshot would not be a refactor of the rule; it would delete a criterion.

The distinction is worth writing down because a later pathfinding story *does* want a frozen
snapshot — for the **path search**, which must not depend on how far through a tick it is asked. That
is a different consumer from occupancy resolution, and the two must not be conflated when it lands.

## The precondition the baseline asserts is false in this tree

Our prior spec says every world produced through the normal construction and command path holds no
two units on one cell. It does not hold here:

- `mapload.unitCell` is `int32(u.X >> 8), int32(u.Y >> 8)` — the fixed-point 1/256 fraction is
  dropped, never rounded, and its own comment says so: "a unit sitting at the centre of its cell and
  a unit sitting at the corner load onto the same cell". Two placements inside one cell collapse onto
  it.
- `sim.NewWorld`'s only error is a duplicate id. It does not look at positions.
- `UnmarshalBinary` refuses truncation, an unknown version, a bad presence byte and non-ascending
  ids. It does not look at positions either.

So a malformed start is reachable through the shipped loader, and the story that adds the invariant
cannot leave the case undefined. Measured on the prototype: two co-located units both under orders
separate on the first tick (the lower id moves, the higher is blocked, and the pair is well-formed
from then on); two co-located units with no orders stay co-located forever. The rule never adds an
occupant to an occupied cell, so it cannot make a malformed state worse — but it is not a repair.

Related edge, also measured: a unit ordered to the cell it already stands on must clear its target
even while a co-occupant is present. Any occupancy test applied to a zero-distance move reads the
co-occupant — or the unit itself — as a blocker and strands the order.

## What else the baseline names that this repo does not have

- **`MoveTo(id, x, y)` does not exist.** There is no such symbol anywhere in the tree. A target is
  set by a `sim.Command{Entity, X, Y}` value in the slice handed to `Step`, and `Step` is the only
  exported call that advances a world.
- **`openrom/` is not a module path here.** The determinism wall is `internal/archtest`: a structural
  import check holding `pkg/sim` to stdlib-only including its tests, plus `CheckSimDeterminism`, a
  parsed-syntax scan over `pkg/sim`'s non-test files that fails on importing `os`, `time` or
  `math/rand` (or anything under them), on a `float32`/`float64`/`complex64`/`complex128` identifier,
  and on a float or imaginary literal.
- Its closing section reporting that no research item is needed is **refused outright** by
  `check-doc-budget.sh`'s content bans, as is its opening provenance-basis paragraph.
- Its disclosed artifact "a convoy in descending-ID order stalls one cell per tick" is not what the
  rule does. Measured over ids `0, 1, 2` at `x = 0, 1, 2` all ordered to `x = 9`: `0,1,3` then
  `0,2,4` then `1,3,5` — the line **stretches** until one-cell gaps open and then every unit moves
  every tick again. A transient stagger, not a stall.

## The original's own rule is decoded, and it is not this one

Worth knowing before writing a contract that will be read as fidelity: research at the current pin
has the original's block planes, its per-mover bitmask, its `n x n` footprint test and its sub-cell
per-step speed. None of it is consumed here and the ledger records why, but one consequence belongs
in the record now — the original lets a ground unit and an air unit share a cell, and ours does not.
One unit per cell is a foundation choice, not a reading of the game.

## What we looked at

`pkg/sim/{step.go,world.go,binary.go,hash.go,run.go,doc.go}` and every `_test.go` beside them,
`pkg/mapload/{fromalm.go,schedule.go}`, `internal/archtest/determinism.go`,
`docs/0019-walking-skeleton/{spec.md,provenance.md}`, `AGENTS.md`, both check scripts, and in
research `claims/retracted.md` first, then `claims/terrain.md` and `claims/alm.md`.
