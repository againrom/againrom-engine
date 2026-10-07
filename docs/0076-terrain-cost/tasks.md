# Tasks — 0076-terrain-cost

Kinds: **impl** — one commit, trailered `SDD-Task: 0076-terrain-cost/T<n>`.

## T1 — the world carries the two planes, and the form carries them  *(impl)*

**Boundary.** `pkg/sim/world.go`, `pkg/sim/binary.go`, and the tests that pin the field set, the
offset table, the version byte and the digests.

**Covers:** FR-1, FR-9 · AC-1, AC-1a, AC-12, AC-13 · DD-1, DD-2, DD-3.

**Fences.** The planes are carried and hashed here and read by nothing: no search, rate or loader
call site changes in this commit. `formatVersion` becomes **13** and no other number is yours.

**Done when:** the root constructor takes the three-plane bundle and the two existing constructors
call it unchanged in signature; an unnamed cost plane materialises at the default and an unnamed
height plane at zero; a named plane of any other length is refused at construction and at decode with
a message naming both lengths; the form carries both planes after the block plane under the header's
one existing cell count; a previous-version form is refused naming both versions; the pinned digests
across `pkg/sim` and `pkg/mapload` are re-taken, and **each re-taken digest is cross-checked by
reconstructing the previous form from the new bytes** — cut the two plane sections, put byte 0 back
to 12, and reproduce the old value exactly — with the old value kept beside the new one; suite green.

## T2 — the search charges the ground its cost  *(impl)*

**Boundary.** `pkg/sim/route.go`, `pkg/sim/optimised.go`, and their tests.

**Covers:** FR-1a, FR-2, FR-3 · AC-2, AC-2a, AC-4, AC-5, AC-6, AC-6a · DD-4, DD-5, DD-10, DD-11.

**Fences.** Four cost sites and no fifth: the canonical relaxation, the canonical extraction, the
optimised flood and the optimised step enumeration. Read DD-4 on which cell each calls the ENTERED
one before changing any of them — two of the four run backwards. Nothing about which cells are *open*
changes, and no budget, ring bound or stall rule moves.

**Done when:** the step cost takes the mover's domain and the entered cell's byte, ground reading the
plane and every other domain paying the flat arm with no read; DD-10's read is carried out and its
result — every use of the label plane, classified against all three shapes — is recorded in the
commit message; the extraction is bounded per DD-11 and an all-zero plane terminates rather than
cycles; a ground mover and a ghost take different routes over two equal-length corridors, and a
cheap detour longer in cells is NOT taken; suite green.

## T3 — the rate reads the two planes  *(impl)*

**Boundary.** `pkg/sim/step.go` and `pkg/sim/rate.go`, and their tests.

**Covers:** FR-4 · AC-7, AC-8, AC-8a · DD-4, DD-9.

**Fences.** `rateOf`, `transitOf` and `diagonalStep` are not touched — only what is passed to the
first, and the file comment that claims the substitute is always taken. One call site, and there is
no second.

**Done when:** the rate is composed from the cost and height bytes of the cell left and the cell
taken, and from four zeros when either cell is off the map; a transit over cost-6 ground takes fewer
ticks than over cost-16 ground and the tilt saturates at a difference of 32 in both directions;
`rate.go`'s standing claim about the substitute reads what is now true; suite green.

## T4 — a map describes a cost plane and a height plane  *(impl)*

**Boundary.** `pkg/mapload/passability.go` and its tests.

**Covers:** FR-5, FR-6, FR-8 · AC-9, AC-10, AC-11a · DD-6, DD-7.

**Fences.** No world is built differently in this commit — the two derivations are exported and
called by nothing yet. The block plane's five arms keep their meanings; only the mountain arm's
*reading* moves, onto the classifier's class.

**Done when:** one classifier answers a class and a cost for a tile word and both consumers read it;
the two derivations are exported, total, sized at the extent and each a function of one source plane;
an **exhaustive** test over all 65 536 tile words shows the derived block byte equal to the pre-story
build's for every word; AC-9's seven words and AC-10's two short planes answer as specified; the
classifier's rejects and the unreachable strip groups are each covered by a case; suite green.

## T5 — every world is built over all three planes  *(impl)*

**Boundary.** `pkg/mapload/fromalm.go`, `pkg/mapload/start.go`, and their tests.

**Covers:** FR-7 · AC-14, AC-16 · DD-8.

**Fences.** Nothing in `pkg/sim` changes. The bundler is one function and every construction path
calls it; a path that derived a plane of its own would be the defect DD-8 names.

**Done when:** all six world-building paths — table and none, party and none, script and none — hand
over all three planes from the one bundler; a map with a non-uniform cost plane produces a world
whose planes are the map's and not the defaults, checked on each path; one shipped map is shown to
route a ground mover differently over its derived plane than over a uniform one; the pinned digests
these fixtures carry are re-taken with the same cross-check T1 used; suite green.

## Traceability

| | FR-1 | FR-1a | FR-2 | FR-3 | FR-4 | FR-5 | FR-6 | FR-7 | FR-8 | FR-9 |
|---|---|---|---|---|---|---|---|---|---|---|
| T1 | x | | | | | | | | | x |
| T2 | | x | x | x | | | | | | |
| T3 | | | | | x | | | | | |
| T4 | | | | | | x | x | | x | |
| T5 | | | | | | | | x | | |

| | AC-1 | AC-1a | AC-2 | AC-2a | AC-3 | AC-4 | AC-5 | AC-6 | AC-6a | AC-7 | AC-8 | AC-8a | AC-9 | AC-10 | AC-11 | AC-11a | AC-12 | AC-13 | AC-14 | AC-15 | AC-16 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| T1 | x | x | | | | | | | | | | | | | | | x | x | | | |
| T2 | | | x | x | | x | x | x | x | | | | | | | | | | | | |
| T3 | | | | | | | | | | x | x | x | | | | | | | | | |
| T4 | | | | | | | | | | | | | x | x | | x | | | | | |
| T5 | | | | | | | | | | | | | | | | | | | x | | x |

AC-3, AC-11 and AC-15 are measured against a lawful install rather than in the suite, and are
discharged in Verify.
