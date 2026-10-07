# 0123-corpse-loot — tasks

Two tasks, in order. Each is one commit trailered `SDD-Task: 0123-corpse-loot/T<n>`.
`verification.md` and the build are stages, not tasks, and carry no trailer.

## T1 — repair the two things in `pkg/sim` the drop stands on

Both are pre-existing defects, neither adds behaviour, and the drop in T2 is unobservable
or wrong without them.

**FR-6, DD-4.** `remove` in `pkg/sim/step.go` rebuilds the entity slice and the route slice
together and leaves the per-entity container slice at its old length. Add it to the same
compaction loop, as a third parallel slice — not a second sweep afterwards, and say so in
the function's doc block.

**FR-7, DD-7.** The doc block on `pkg/sim`'s random source states that nothing in this
project recovers the game's generator and no research claim describes it. That is stale.
Rewrite it: research now describes the original's generator (cite the claim ids from
`provenance.md`, not an experiment folder), ours is a different generator by choice, and
that difference is a disclosed divergence. Do not change any code in that file — the
generator, its constant and its bounded draw stay exactly as they are.

**Tests.** Add a test for AC-6: build a world of two entities where the second carries a
code, remove the first, and assert the second still reads back its own code, that the
container list and entity list are the same length, and that the whole-world stock read
answers without panicking. It panics on master; it must pass here.

## T2 — a dying entity's container becomes a ground sack

**FR-1, FR-2, FR-3, FR-4, FR-5; DD-1, DD-2, DD-3, DD-5, DD-6.**

In `pkg/sim`, inside `clearFelled`'s existing `Decay == DecayNone` guard (DD-1), drop the
entity's container onto its own cell. Add one unexported method on the world that does the
merge-or-insert (DD-2): `sort.Search` over the `(Y, X)`-ascending sack list, then append
onto the found sack's codes or insert a new sack at the search's own index. It must not
re-sort and must not touch a standing sack's gold.

Hand the entity's own slice over and set the entity's to `nil` (DD-3). Build nothing for an
empty container (FR-2). Refuse a cell the package's one per-sack predicate refuses, leaving
the container as it stands (FR-5, DD-5). Take no draw from the world's generator and add no
field to any record (DD-6).

**Tests.** AC-1 through AC-5, AC-7, AC-8, AC-9, and SC-5's off-map case. AC-7 compares an
advance in which a carrying entity dies against the same advance with nothing to drop, and
asserts the generator state is the same. AC-8 runs the existing transfer primitive from a
second entity standing on the corpse's cell. AC-9 encodes and decodes a world in which a
death has occurred and asserts equality, and asserts the form version constant is the value
it held before this story.

Touch no file outside `pkg/sim`.
