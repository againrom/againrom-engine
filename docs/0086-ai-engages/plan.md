# Plan — the decision, and the state it needs

## Design decisions

**DD-1 — the relation is a value type of its own, not a bare slice on the world.**
`Relations` wraps one `[relationSlots*relationSlots]byte`-shaped slice with two methods: a
constructor that copies, and `hostile(from, to uint32) bool`. Making it a type is what lets the
loader build one, a test build one and the world hold one without three places agreeing on the
stride by hand — the stride is the failure this shape removes, because a transposed index is
silently a different, plausible game.

**DD-2 — 50 slots, fixed, materialised, carried whole.** `newRelations` follows `newGrid`'s rule
exactly: an empty input materialises the all-zero matrix, any other length is refused, and the
bytes are copied. So "a world named no relation" and "a world named the zero relation" are one
world in the fields, in the bytes and in the digest, and no flag records which way it was built.
No byte is masked on the way in — every value is a state a caller may build (P-2).

**DD-3 — the world gains one field and the byte form one version.** The relation is appended to
the encoding after the planes and before the entities, at a fixed 2500 bytes with no length
prefix, because its length is a compile-time constant of this build. `formatVersion` moves to 16.
No test asserts the literal: the version is a fact about the encoding, not a contract this story
holds, and pinning it taxes every later bump. `TestTheCanonicalWorldsFieldSetsArePinned`'s table
gains the field, which is the pin that matters — what it refuses is the field nobody declared.

**DD-4 — a fourth constructor takes it positionally, and the three older names keep their
signatures.** `NewRelatedWorld` becomes the root and every other constructor delegates to it with
no relation, which is the rule this package already applies to the script, the cost plane and the
height plane: an unnamed input is materialised to exactly the behaviour a world without it had, so
the older names are the honest signature for a caller with nothing to put in the new slot rather
than a compatibility shim. The relation is a parameter and not a fourth `Terrain` field, because it
is not a per-cell plane and a struct field is what makes a value easy to default by accident.

**DD-5 — the decision is a phase of `Step`, between the script pass and the scratch.** It runs on
`scriptPassPhase`, in the same `switch` arm and after `scriptPass`, so the two cannot come to
disagree about which tick they share. It stands before `newRouteScratch` deliberately: the engage
clears a member's destination and route, and every other writer of those runs either in the
command phase or, in `approach`'s case, against a live occupancy plane. Running here puts the
decision in the same position as a command, which is what it is.

**DD-6 — groups are built once per decision, into a scratch the pass owns.** One pass over the
entities in ascending id, appending each alive entity with a nonzero owner to the group whose
`(owner, group)` pair it matches, and starting a new group when none does. The search is a linear
scan over the groups built so far. It is not a map, for the reason `containsIndex` is not: Go
randomises map iteration order, and this is a path that touches a world. It also gives the two
orderings the law needs for free — groups in ascending lowest-member id, members in ascending id,
so "the group's first member" is the head of its slice and needs no second rule.

**DD-7 — the candidate list is built once per group and scored per member.** The law clears one
shared sight map, stamps every member into it, sweeps the whole actor list against it, then filters
and parks; only the scoring is per member. Building per member instead would be the same answer at
`members × candidates` cost and would lose the property that makes the shape observable — that a
candidate seen by one member is a candidate for all.

**DD-8 — the sweep is over the whole entity slice, and that is not a defect to fix.** The law's own
sweep is head to tail of the world actor list with no neighbourhood and no spatial index, twice.
Reproducing the cost is not the point; reproducing the *set* is, and any index would have to agree
with the full sweep on every world anyway. It runs once per full tick, not once per tick.

**DD-9 — sight is `sightRadius`, a package constant of 5, and the window is a second constant.**
Both are written where the population is computed, with the writer set that makes 5 the only
reachable value. The effective radius is `min(sightRadius, sightWindow)` rather than 5 outright, so
the window is a term of the expression and a story that gives sight a field does not have to
rediscover the cap.

**DD-10 — the two scorers are one function and a mode, not two functions.** `AI-REACH-072` reads
the second as the first with three differences, and writing them as two bodies would let the two
drift on the thirteen lines they share. The mode is the group's order, which is the one thing that
selects between them in the law as well.

**DD-11 — the domain axis is an explicit three-entry table, not `d + 1`.** `sim.Domain` numbers
ground, ghost and air 0, 1, 2; the law's byte numbers them 1, 2, 3 and reserves 0 for the column a
reach above 1 folds onto. The arithmetic happens to agree today and the table is what keeps a
fourth domain, or a renumbering of ours, from silently reindexing the preference matrix.

**DD-12 — the preference matrix is written out as the four rows the claim states.** All sixteen
cells, including the two rows and the column no world this build can construct reaches, because the
veto is a property of the table and a table missing its unreachable half cannot be checked against
the source.

**DD-13 — the engage reuses the command arm's rule, extracted.** `orderAttack` carries the
"naming the victim already held leaves the cycle alone" rule and the `clearOrder` that ends the
walk; `KindAttack` and the decision both call it. That rule is load-bearing here rather than a
convenience: the group re-issues unconditionally every decision, so without it a member's charge
would reload on every decision tick.

**DD-14 — a member with no candidate is touched in no field.** Not the attack order, not the
target, not the facing. This is FR-20 and it is the whole of what makes D-3's absent walk half a
*named* omission rather than an invented break-off.

**DD-15 — the loader builds the relation and hands it over; `alm` only exposes bytes.**
`pkg/formats/alm` gains the sixteen words on its roster record and interprets nothing — the store's
index arithmetic, the low-byte narrowing and the forced diagonal are the loader's, because they are
the engine's map-load path and not the file's layout. A map with no type-5 roster yields no rows
and therefore the all-zero relation, which is DD-2's materialisation and not a second rule.
**And every rebuild on the start path has to name it** (T5). Both start entry points build a world
and then build a second one out of its parts; a rebuild carries only what it names, and a dropped
relation is silent — the world still builds, still hashes and still ticks, and its one symptom is
that nothing ever starts a fight, which is what every world looked like before the relation existed.
It is taken off the world already in hand rather than restored from the map, so the store stays in
one place and a start cannot come to disagree with a plain load.

**DD-16 — the notice radius is computed where the clip is.** One function taking the members and
the centroid, returning the byte. It narrows each member's `distance + sight` to a byte before the
maximum, which is the width the field is stored at; the floor and the `+4` follow. That the result
is fresh rather than frozen is D-3 and is written in the function's own doc, beside the arithmetic
it diverges from, rather than only in the spec.

## Risks

**R-1 — the byte form grows by 2500 bytes on every world, including the ones that will never hold
a relation.** Accepted: it is the law's own object size, it is what the original's save carries,
and the alternative is a sparse encoding that has to prove injectivity separately. The cost is
linear in worlds, not in ticks, and `Hash` already walks the whole form.

**R-2 — a group that loses its candidate keeps chasing.** The law's break here is the walk-home
branch, which is out of scope, and there is no break-off under group orders 1/2/3/5 at all. So a
creature that acquires and then loses sight of its victim pursues it across the map. This is a
product risk and it is the visible cost of the scope line; it is named in D-3's neighbourhood in
the spec and belongs to the story that implements the post.

**R-3 — every existing pinned digest and byte form in the tree changes.** Unavoidable at any
version bump. The mitigation is that they are pinned in one place each and recomputed from the
same encoder, so a wrong new pin is caught by the round trip rather than by the pin.

**R-4 — the decision could make a world nondeterministic without failing a test.** The sweep, the
grouping and the scoring all have orderings that a map or a sort with an unstable comparator would
scramble. Mitigated structurally: no map anywhere on the path, no sort, no draw, and the
determinism half of `internal/archtest`'s source scan already refuses a clock or a generator in
this package.

## Success criteria

- **SC-1** Two hostile entities and no commands at all end in a fight, and the fight kills one
  (FR-19, AC-4).
- **SC-2** Every acquisition boundary the contract names — sight, notice radius, reach under stand
  ground, the flier veto, the corpse rule, the direction of the relation — is witnessed on both
  sides of the boundary (AC-3, AC-6 … AC-11).
- **SC-3** A world round-trips with its relation, and two worlds differing in one byte differ in
  their digests (AC-1, AC-2, P-2).
- **SC-4** A `missionrun` drive over a real map produces identical per-tick digests on the EN and
  RU roots, and identical digests across two runs of the same root (P-1).
- **SC-5** The decision fires on the decision phase only (AC-5, FR-7).
- **SC-6** A map's roster row reaches the world (AC-15, FR-6).

## Traceability

| Decision | Serves |
|---|---|
| DD-1, DD-2 | FR-1, FR-2, FR-3, FR-4, FR-5, P-2 |
| DD-3, DD-4 | FR-1, AC-1, AC-2, P-2 |
| DD-5 | FR-7, AC-5, P-1 |
| DD-6 | FR-8, FR-11, P-1 |
| DD-7, DD-8 | FR-10, FR-12 |
| DD-9 | FR-10, D-1 |
| DD-10, DD-12 | FR-9, FR-15, FR-16, FR-17, FR-18 |
| DD-11 | FR-17, AC-9 |
| DD-13 | FR-19, AC-12, P-4 |
| DD-14 | FR-20, AC-13, P-4 |
| DD-15 | FR-6, AC-15 |
| DD-16 | FR-13, FR-14, AC-7 |
