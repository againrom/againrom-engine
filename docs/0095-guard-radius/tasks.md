# Tasks — 0095 the frozen guard radius

`impl` = production code plus its tests, one commit each. T2 follows T1.

## T1 — the group record, frozen at construction and carried by the form `impl`

Give `World` a per-group record holding a frozen notice base, build it in the constructor, and
carry it in the byte form and the digest. Nothing in a tick reads it yet, so behaviour is
unchanged and the existing per-decision computation stays exactly where it is.

Files: `pkg/sim/{engage,world,binary}.go`, and the tests of that package that pin the field set,
the byte form, the digests and the form version.

Covers FR-1, FR-2, FR-3, FR-5, FR-6, FR-7, FR-8; DD-1, DD-2, DD-3, DD-4, DD-5, DD-6, DD-7, DD-8,
DD-9, DD-10, DD-11.

Scope fence: no decision path changes and no clip changes — a reader is T2's. The sight march, the
centroid, the candidate sweep, the scorer and every other section of the byte form are untouched.

Done when: a built world holds one record per owned `(owner, group)` pair in ascending order; the
base is the group's living geometry raised to the floor; the form is at the version this build
defines with the section between the routes and the script; the pinned bytes and digest are hand
transcriptions and a second test strips the section off, sets the previous version byte and reaches
the previous version's pinned digest; every malformed section is refused with its own message; no
test names the version as a literal.

## T2 — the clip reads the frozen base `impl`

Make the guarding group's candidate clip take its radius from the record instead of recomputing one
per decision, and leave the clip's origin — the centroid, recomputed every decision — alone.

Files: `pkg/sim/engage.go` and its tests.

Covers FR-4, FR-9; DD-5, DD-12.

Scope fence: the byte form, the record type and the constructor's freeze are T1's, landed, and not
touched. No walk home, no latch, no roll, no group order beyond the two already implemented.

Done when: the decision reads the stored base and adds the margin, and nothing on a decision path
computes a radius from live geometry; a group whose pair no record names clips at the margin alone;
a world stepped far enough for a group to walk, spread and lose a member clips at the radius it was
built with; the geometry-and-floor computation is a named function the constructor calls and the
margin is added in exactly one place; the file comment that named this function the seam says what
it now is, and the file header's account of the absent walk home no longer claims the walk would
send a guard to the map's corner — the post is the cell the unit stood on when guard was issued.

## Traceability

| Task | Requirements | Decisions |
|---|---|---|
| T1 | FR-1, FR-2, FR-3, FR-5, FR-6, FR-7, FR-8 | DD-1, DD-2, DD-3, DD-4, DD-6, DD-7, DD-8, DD-9, DD-10, DD-11 |
| T2 | FR-4, FR-9 | DD-5, DD-12 |
