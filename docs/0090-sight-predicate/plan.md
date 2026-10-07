# Plan — 0090 sight predicate

## Approach

One new file, `pkg/sim/sight.go`, holding three things in the order they depend on each other: the
window tables (FR-1, FR-2, FR-3), the march over a world from one observer (FR-4 … FR-9), and the
group union (FR-10). `pkg/sim/engage.go` then loses its Chebyshev test and reads the union instead
(FR-11), and the comment above its sight constant is corrected where the correction is load-bearing.

Nothing else in the tree is touched. No package outside `pkg/sim` learns anything, no byte form
moves, and no field is added to any record.

## Design decisions

**DD-1 — the window tables are package-level values, built once per process.**
The alternatives were a per-world field and a lazily built cache. A per-world field is state no
world could vary — the tables depend on the shift alone and the shift is a constant here — so it
would have to be excluded from the byte form and from the digest by hand, and the field-set pin
exists to refuse exactly that kind of second source. A lazy cache needs a guard, and a guard on a
path that advances a world is a synchronisation primitive inside the determinism wall. A
package-level value initialised from a pure function has neither problem and is what FR-1's "not
canonical state" means in Go.

**DD-2 — the cost table is integer arithmetic over an integer square root, not floating point.**
`pkg/sim` is held to no float by a source scan over its non-test files, so the published expression
cannot be transcribed as written. `trunc(128*sqrt(s)/m)` for non-negative integers equals
`isqrt(16384*s) / m` in integer division, because `floor(floor(a)/m) = floor(a/m)` for a positive
integer `m`. The identity is exercised rather than asserted: the two forms were compared at all
1 681 window offsets before this was adopted, and the published bounds 128..181 are what the integer
form must produce (SC-1).

**DD-3 — one flat array of 41x41 running values per march, indexed by offset, reused across rings.**
The march needs a predecessor's value and nothing else, and FR-2 guarantees the predecessor sits one
ring in. So the array can be filled ring by ring with no revisiting and no queue, and the visiting
order *within* a ring cannot change the result — which is worth having, because it removes an
ordering this build would otherwise have to defend as the law's.

**DD-4 — the group's stamp is one byte per world cell, allocated per group per decision.**
It answers exactly one question, `is this cell lit`, and it is thrown away at the end of the
decision. A set keyed by cell would be a Go map on a path that advances a world; a sorted slice
would need a search per candidate. The array is what the law itself keeps, at the same lifetime, and
its cost is one allocation per group on one tick in sixteen.

**DD-5 — the inset test is `grid & blockAir`, and it is spelled at the march.**
The law's rectangle is arithmetic on the map's own extent. The grid this package already carries has
that same region marked, and marked by nothing else, so the test is a read of state the world
already holds rather than a second derivation of the map's shape that could come to disagree with
the first. It also leaves a world naming no grid uninset, which is what keeps the predicate testable
on a small synthetic world. The divergence this buys is named in the contract; the alternative —
computing `8 <= x <= W-9` here — was rejected because it would make every world under 18 cells on a
side blind, including every world in this package's own test suite, for a rule whose only effect on
a loaded map is on cells no map places a unit in.

**DD-6 — the observer's own cell is marked before the rings, and its budget is seeded whether or not
that cell is inset.**
Splitting FR-8 from FR-9 costs one line and buys the case that would otherwise be silent: a unit
standing in the border ring, which no loader produces and a hand-built world can, would otherwise
see nothing at all — including itself — and the candidate sweep would then not find it under its own
group's stamp.

## Files

| File | What changes |
|---|---|
| `pkg/sim/sight.go` | new — FR-1, FR-2, FR-3, FR-4, FR-5, FR-6, FR-7, FR-8, FR-9, FR-10 |
| `pkg/sim/engage.go` | `seen` replaced by the union read; the sight constant's justification corrected — FR-11 |
| `pkg/sim/sight_test.go` | new — the window's own properties and the march's |
| `pkg/sim/engage_test.go` | the one test that called `seen` directly, and the occlusion cases |

## Success criteria

- **SC-1** The integer cost form and the published float form agree at every window offset, and the
  table's extremes are 128 and 181.
- **SC-2** The flat-ground visible counts reproduce the published figures at ranges 1..6 and at 19,
  which no wrong reading of the zone tests, the mirroring or the truncation produces.
- **SC-3** The byte form, the field-set pin and the digest are untouched: the whole suite is green
  with no version moved and no decoder arm added.
- **SC-4** The tenth mission's drive is measured on both roots before and after, and the number is
  reported whichever way it goes.

## Risks

- The region can be **larger** than the disk it replaces where the observer stands high, so this is
  not a pure narrowing and a world that acquired nothing before may now acquire something.
- The stamp is rebuilt per group per decision; a map with many groups pays an allocation per group
  on the decision tick. Measured by the existing drive rather than assumed.
