# Tasks — 0059-formation-move

One task = one commit, ending `SDD-Task: 0059-formation-move/T<n>`, touching at least one
file outside `docs/`. `verification.md` and the build are pipeline stages, not tasks: they
carry no trailer. Each entry is written for an agent holding `spec.md`, `plan.md` and that
entry alone.

## T1 The term, its reader and its three writers — FR-5, FR-6; DD-1

Give `Entity` a `GroupSpeed uint8` and document it as the group rate term: the minimum
speed of the group as it stood when a formation order was issued, zero meaning none.

Add `moverSpeed(e Entity) int32` — the term when nonzero, `e.Speed` otherwise — and make
`rated` read it. Feed it to the single `rateOf` call in the advance, so the term reaches
both arms of the law.

Add `clearGroupSpeed` on `Entity` and call it from exactly two sites now: the `KindMoveTo`
arm, and `clearFelled`. Both mirror the original — a plain order allocates a fresh group at
zero, and a felled member is unlinked from its group. Normalise it to zero in `NewWorld`
alongside the target and the transit a not-alive entity may not keep.

Document at `moverSpeed` that this is the seam: nothing else reads the byte, and lifting
the never-cleared defect is a third call at arrival, not a rewrite.

Tests: the term replaces the speed in every domain, including on an entity whose own speed
is not positive; a plain order and a felling clear it; arrival and giving up do not.

## T2 A group order, every member to the ordered cell — FR-1, FR-4; DD-2

Add `KindGroupMoveTo` (not the zero value) and `Group uint32` to `Command`. Document the
tag as a correlation key for one advance that is stored nowhere.

In phase 1, keep the single pass in slice order. At the first unconsumed command of that
kind, collect every command in the slice with the same tag, mark them consumed, and apply
one order: members are the entities the world holds that are alive, each once; the ordered
cell is that first command's; every member's group speed is zeroed and its target set to
the ordered cell **clamped into the world's bounds**. An order with no surviving member
does nothing.

Add the clamp as one small function over `Bounds`.

Tests: two orders in one slice stay two orders and the later wins; an absent or felled
member is skipped; a duplicate naming is counted once; the destination clamps at both ends
of both axes; the tag reaches neither the byte form nor the digest.

## T3 The formation flag, the centroid, the offsets and the minimum — FR-2, FR-3, FR-4, FR-5, FR-8; DD-3, DD-4, DD-5, DD-8

Fork T2's order on one flag computed once.

Centroid: per axis, sum `cell*subCell + subCell/2` over the members in `int64`, floor-divide
by the member count, divide by `subCell`. Spell the half as `subCell/2`.

Flag: set unless some member's Chebyshev distance in whole cells to the centroid exceeds
`formationSpread = 2`. One member over the line clears it for the whole group. Document at
the flag that the per-player formation mode is not modelled, what its three behaviours are,
and that every order here runs at the shipped default.

In formation, each member's destination is the ordered cell plus `int8` of its displacement
from the centroid, then clamped as in T2; and every member's group speed becomes the
running minimum — a `uint8` at `groupSpeedInit = 250`, compared `int16(e.Speed) <
int16(min)`, taking `uint8(e.Speed)`. Out of formation, neither happens.

No branch anywhere on the member count.

Tests: the centroid table with an exact half each way and an odd count; the flag's boundary
in both directions flipping distribution and term together; a group of one as the identity;
the minimum's initial value, its strict comparison and both narrowings.

## T4 The byte form goes to version 8 — FR-7; DD-6

Raise `formatVersion` to 8 and `entityLen` to 44, with the group speed at record offset
`+43`. Extend the offset table in the doc comment and say why the byte went to the tail.

Encode and decode it. Refuse a nonzero term on an entity that is not alive, beside the
target and transit refusals already there, and keep refusing every version but this one.

Grow the hand-transcribed pin in `binary_test.go` by one byte per record, and recompute
`pinDigest` **outside this tree** with an independent FNV-1a — never from the encoder. Say
in the comment that version 7's digest does not carry over and why.

Tests: round trip through a world carrying a term; two worlds differing only in that byte
digest differently; version 7 refused; the not-alive refusal; the pin's bytes and its
digest.

## T5 The map screen issues one group order — FR-9; DD-7

Change `mapWorld.enqueue` to emit `sim.KindGroupMoveTo`. Nothing else about the seam moves:
`ui.MapOrder` keeps its signature, the pending queue keeps its shape, and the commanded set
keeps its meaning.

Document at that statement why every order queued between two advances is one group order —
one click, one slice, one tick, one tag — and that a selection of one is a group of one and
takes no other path.

Tests: a selection ordered at one click produces one group order over the whole selection
and a world advanced by it distributes; a selection of one is unchanged in destination.

## T6 The two witnesses the earlier tasks left owing — FR-7, FR-9; DD-6, DD-7

Two claims are made by the contract and asserted nowhere yet, and each is the half of
its clause that a cache or a merge would pass.

In `pkg/sim`: a world carrying a group term marshals, decodes back to the same world and
walks at the term afterwards; and two worlds alike in everything but that one byte
digest differently. The second is what says the byte is canonical rather than merely
present — a term the encoder dropped would leave the two equal and a save taken
mid-formation would resume at a speed nothing recorded.

In `pkg/game`: the click boundary, in both of its clauses. One press is one tag over the
whole selection; a second press opens another, whether it differs by naming a different
cell or by naming a unit the group already holds. A build that dropped either clause
would merge two presses into one order and take the later one's destination away.

Tests only; no non-test file changes.

## T7 The offset's two widths, exercised directly — FR-4; DD-3

Mutation testing found the one clause of the distribution nothing could kill: dropping the
offset's byte narrowing changes no test, because the spread gate admits displacements in
`[-2, +2]` only and both widths are transparent over all five. The clause is decoded and it
is the story's own customisation limit, so it is made reachable rather than left unwitnessed.

Extract the pair of narrowings into `formationOffset(delta int32) int32` — the 16-bit store
and the signed-byte read, in one line — and say at it why it is a function: nothing an order
can be given reaches past the gate, and the widths are here for the formation mode this tree
does not model.

Tests: the offset over a table that crosses both widths — 127 and 128, -129, a whole byte on,
and where the 16-bit field itself wraps.

## Traceability

Every FR and DD of `plan.md` is carried by a task above: FR-1 T2 · FR-2 T3 · FR-3 T3 ·
FR-4 T2, T3 · FR-5 T1, T3 · FR-6 T1 · FR-7 T4 · FR-8 T3 · FR-9 T5; DD-1 T1 · DD-2 T2 ·
DD-3, DD-4, DD-5, DD-8 T3 · DD-6 T4 · DD-7 T5. The AC, P and SC ids are witnessed by
`verification.md`, which is a stage. T6 carries FR-7 and FR-9 a second time, and DD-6
and DD-7 with them; T7 carries FR-4 and DD-3 a second time.
