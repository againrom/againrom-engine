# Plan — 0106

Three tasks. T1 puts the field on the record and makes the constructor the first writer. T2 adds the
second writer, the script's stance commands. T3 is the arm: the guard stance's walk home, and the
byte form and digest that carry the state the first two produce.

The order is not arbitrary. Every task after T1 has to move the digest, so the pin is re-taken once,
at the end, in T3 — re-taking it twice would mean pinning a world the story does not ship.

## The shape of the change

`pkg/sim` already holds every part of this except two: a cell to walk to, and the outcome that walks
there. `engage.go`'s `decide` already runs the guard stance's clip, its scorer and its release;
`script.go` already dispatches the two stance sub-commands; `world.go` already writes one per-actor
field into every entity unconditionally, which is the pattern the post takes. So the change is a
field, two writers, one tail, and a version bump. No package outside `pkg/sim` is touched (SC-2,
P-4), and no new file is needed.

## FR-1 — the field

Two `int32` on `Entity`, beside the patrol ring, which is the nearest thing to it already there: a
per-actor pair of coordinates that only one arm reads. `int32` and not a packed word, because every
other coordinate on the record is `int32` and a second width would need a conversion at each end and
a rule about which is authoritative. **DD-1** is why the pair is on the entity and not on the group
record beside the order: both stances anchor one and a group's members do not share a cell, and a
group-level anchor would be a second spelling of the centroid — a different object with a different
job, which `clipToNotice`'s own comment already warns about.

There is deliberately no "unset" encoding and no boolean beside it. The law's field starts at zero
and is written on the first thing that touches it; this build writes it at construction instead, so
no reader ever has to ask whether it means anything (FR-1's last sentence).

## FR-2, FR-3 — the two writers

**The constructor.** Beside the actor-state write in the entity mint, unconditionally, from the
entity's own coordinates. **DD-2**: this makes the post an output of construction rather than an
input, so a caller cannot supply one, a hand-built entity literal needs no post, and the field is
in-bounds by construction without any clamp — the same trade the actor state already takes, and the
reason FR-1 needs no normalisation rule and no fault shape.

**The stance commands.** `cmdGroupGuard` and `cmdGroupStandGround` each gain the same per-member
write, over the living members of every record their group id names — the membership walk
`cmdGroupGuard` already performs for its notice base. Writing it in both, rather than hoisting it
into the dispatcher, keeps FR-9's claim checkable: the two arms that anchor are the two that carry
the write, and adding it to a third arm would be a visible edit.

**FR-3** is the write itself: the entity's current cell, with no test on anything. The law branches
here — its guard setters take a moving unit's step-target cell instead — and **DD-3** records why
that branch is absent rather than skipped: this tree commits a mover's coordinates to the
destination cell before it pays the crossing's ticks, so a mid-crossing entity's own cell already
*is* the cell it stepped into. The branch would be a no-op. What this tree does not have is an
equivalent of the law's "not on a cell centre" disjunct in the walk-home gate; that half is the
disclosed divergence in DD-3's second sentence, and it costs a re-issue nobody can see.

A dead member is not written, on FR-2's own words. The constructor writes every entity including a
dead one, because at construction "living member of a group" is not yet a question the mint asks and
an unwritten post would be the only zero-valued field on the record.

## FR-4, FR-5, FR-6, FR-7 — the tail

The law's per-member tail tests the member's **victim**, not the score. This build's loop tests the
score, and the two agree on every member it decides for except one: a member of the local
participant's own group that scores nothing this pass while still holding a victim. Keying the tail
on the victim rather than on the score is therefore both closer to the law and simpler, and it is
what **FR-5** states.

So the tail is: for each member of a guard group, after the existing loop has run, a member holding
no victim and standing off its post is given its post as a destination (FR-4); a member holding a
victim is untouched (FR-5); a member holding no victim and standing on its post is untouched
(FR-6). Ordering it after the loop is what makes FR-4's supersession of `0098` come out right: the
release runs first and clears the victim, and the member the release just freed is the one the tail
then sends home, in the same tick.

**FR-7 is the structural half and it is the part most likely to be built wrong.** `decide` has an
early return for an empty candidate list, and an empty candidate list is the *ordinary* state of a
quiet map — exactly where the walk home matters. The tail must run on that path too. The chosen
shape is one named method over the group's members, called at both exits the guard stance can
reach, rather than restructuring `decide`'s early returns: the early returns exist to keep the swarm
and move arms away from an unconditional release, and moving them would put those arms at risk for
no gain here. The alternative considered and rejected was making the tail a third pass in `Step`
after `actorPass` — rejected because it would read the group order a second time, in a second place,
which is the re-derivation `0096` removed.

The destination write is the plain one — clear the order, set the target, mark it held — as the
patrol arm does, and **not** the distribution helper, which offsets and clamps for a formation this
arm does not compute. No clamp is needed on a post the constructor wrote; a post that arrived
through a decode is carried whole under FR-10 and reaches the same route layer a decoded commanded
cell already reaches.

## FR-8 — the stand-ground stance

Nothing is added to its arm. This is a real clause and not an absence: the temptation is to give it
the same tail on the grounds that both are stances, and the law is explicit that it has no walk of
any kind. Its share of this story is the anchoring in FR-2. **DD-4** carries the one outcome it does
have and this story does not build — the idle turn — and the reason is the draw count, not the
evidence: it would take two draws from the world's generator per idle member per tick, which makes
every combat roll depend on how many hostiles are standing still. That is a change to the
determinism surface, and it belongs to a story whose acceptance criteria are about the generator.
**DD-5** cuts the heal, which has nothing in this tree to be inconsistent with.

## FR-9 — the reader set

One reader, in one file. The scope fence **SC-3** makes it checkable by deletion (AC-12), which is
the same instrument `0099`'s SC-1 used for the actor dispatch. **DD-6** is where `0099`'s D-3 is
answered: that decision said the post had no reader outside the per-actor guard, and it has one —
this arm. The latch half of D-3 is untouched and still owed.

## FR-10, FR-11 — the form and the digest

Version 24, allocated to this story and used. The two coordinates go on the entity record's tail,
after the reach byte the previous version added, so no existing offset moves — **DD-7**. The record
grows by eight bytes; the layout comment at the top of `binary.go` gains a row. Version 23 is
refused as every earlier version is, through the same test.

The post is carried **whole**, refusing nothing. This is the commanded cell's and the patrol ring's
rule and it is chosen for their reason: the constructor produces only in-bounds posts, but a form is
a state this package must read back, and a refusal here would make some world this package can write
one it will not read. There is no fault shape for a post on a dead entity either — every actor has a
post at every moment, and a dead one is not residue.

**FR-11**: the pinned bytes and the pinned digest are re-taken together, from the same world, at the
end of the story. A digest adjusted without its bytes would pin two different worlds.

## The properties

**P-1** follows from FR-2 having exactly two writers and neither being on a per-tick path; the test
is a tick over a world with no stance command in it. **P-2** is the one that needs a fence rather
than a test, because it is about what is *absent*: **SC-1** says no path added here draws from the
generator, and the existing draw-count discipline in `rng.go` is what makes the claim meaningful.
**P-3** is what FR-2's "every entity" and FR-3's "unconditionally" buy: no writer asks what order a
group is under. **P-4** is **SC-2**. **P-5** is the regression surface — a world with no guard group
must step identically — and it is what the existing suite already measures if nothing else in the
package moves.

**SC-4** keeps the tasks synthetic: nothing here needs an install, and no measurement against one is
asserted from a task. **SC-5** is the local gate.

## Traceability

| Concern | FRs | DDs | ACs | Ps | SCs |
|---|---|---|---|---|---|
| the field, the constructor | FR-1, FR-3 | DD-1, DD-2 | AC-7, AC-9 | P-3 | — |
| the stance commands | FR-2 | DD-3 | AC-8 | P-1 | — |
| the tail | FR-4, FR-5, FR-6, FR-7, FR-8, FR-9 | DD-4, DD-5, DD-6 | AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-12 | P-2, P-5 | SC-1, SC-3 |
| the form and the digest | FR-10, FR-11 | DD-7 | AC-10, AC-11 | P-4 | SC-2, SC-4, SC-5 |
