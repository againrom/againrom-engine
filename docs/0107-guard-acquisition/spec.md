# Spec — what a weapon's reach does to the target a group picks

## Terms

**Member** — an entity a group's decision is taken for. **Candidate** — a hostile entity the
group can see, after the guarding group's notice circle has clipped the list.

**The ordinary choice** — the scorer a group under Guard, Swarm, Move or Swarm 2 reaches.
**The stand-ground choice** — the variant a group standing its ground reaches, which refuses
anything past a distance of one and so never takes a step toward a target.

**The preference table** — the fixed four-by-four table of multipliers the choice runs through,
indexed by the member's movement domain and the candidate's, in the numbering the original uses:
0 the immobile column, 1 ground, 2 ghost, 3 flier. A zero cell is an absolute veto.

**Reach** — how far a blow carries, in whole cells, per entity, at least 1.

## Why

A group's target choice is already built: the shared sight stamp, the hostility filter, the
notice circle, the four-by-four table, the distance-dominated cost and the strict comparison
that lets list order break a tie. One input to it is not: **the member's own reach, and the
candidate's**.

Reach became a per-entity value carried off each placement's weapon two stories ago, and every
term in the choice that reads it was left at the value it had when reach was the constant 1.
Three of them are wrong for anything holding a bow, and the largest is a veto: today a ground
member will not auto-select a flier at any distance, and a shipped map places eighteen
bow-armed ground units and five flying ones. In the original, reach — not class — is what
decides whether a creature chases something in the air.

## Scope

**In:** the two reach terms of the choice, in the one body both variants share, and the order
in which the stand-ground refusal reads them.

**Out, each with what it is owed to:**

| Cut | Owed to |
|---|---|
| The one-cell jitter on the notice circle | A story that gives the simulation a per-group latch and a lawful draw. The draw crosses the determinism wall and the latch is group state this tree does not carry. It was cut once before on the same ground. |
| Reading the notice circle's floor out of the game's own registry | A customisation seam. The simulation package reads no file and may not; the floor arrives as a constructor input or not at all. The value it would carry is the value already written here. |
| A group remembering, for twenty ticks, the cell of something that attacked it, and stamping that cell into the shared sight map | Two per-member fields and a byte-form version, **and** a published writer for the remembered attacker. Neither exists. |
| The turn cost — the low byte of the cost, under a distance term shifted eight bits left | A published body for it. Nothing describes one, so there is nothing to transcribe; it stays zero and ties fall to list order. |
| The flat penalty a candidate under one particular spell carries | Spells. This tree has none. |
| The two gates on the multipliers — a candidate one cell across, and a mind above a threshold | A footprint field. Every entity here occupies one cell and the mind threshold is satisfied by the original's own default, so both gates are satisfied by construction wherever this build can ask. |
| Any leash that breaks a pursuit off | A per-actor-state story. The five-cell break-off belongs to the per-actor machine, which a group under the guarding order never evaluates; the group arm re-issues its engagement unconditionally and has no leash at all. |

Nothing in this story adds state. **The byte form does not move and the byte-form version is
unchanged.**

## The contract

### The member's own reach

**FR-1** — A member whose reach is **above 1** reads **row 0** of the preference table alone:
the multiplier is the row-0 cell for the candidate's domain, and the member's own domain is not
consulted at all.

**FR-2** — A member whose reach is **above 1** rewrites its distance term before the cost is
formed: a candidate at a distance **within** that reach scores a distance term of exactly **1**;
one beyond it scores its distance **less one short of that reach**, so the term falls by the
same amount for every candidate outside. The rewritten term is at least 2 for anything outside
reach, so nothing outside can ever score as though it were inside.

**FR-3** — A member whose reach is **1** is unchanged in both respects: it reads the cell at
[its own domain][the candidate's], and its distance term is the plain distance.

### The candidate's reach

**FR-4** — Under the **ordinary** choice, a candidate that is a **ground** mover and whose reach
is **above 1** is indexed as the **immobile column** — column 0 — instead of the ground column.
A candidate of any other domain is indexed by its own domain whatever its reach.

**FR-5** — Under the **stand-ground** choice, **any** candidate whose reach is above 1 is
indexed as the immobile column, whatever its domain. This is the one place the two variants
differ on this term, and it is the difference stated for them.

### The refusal and the veto

**FR-6** — The stand-ground choice's outright refusal reads the **rewritten** distance term of
FR-2, not the plain distance. Its ceiling stays the literal **1** and is not the member's reach:
what lets a member of reach above 1 pass the refusal is FR-2 having already turned its distance
term into 1, and what still stops a member of reach 1 is that nothing rewrote its.

**FR-7** — The absolute veto is unchanged: a zero cell scores the selection loop's own seed, and
the loop's strict comparison can never take it. Row 0 holds **no zero**, so a member of reach
above 1 has **no domain veto at all** and may select a flier — where a member of reach 1, ground
or ghost, still may not.

### What does not move

**FR-8** — The candidate list, the notice circle, the clip, the emptiness branch, the release
and the walk home are untouched. This story changes what a candidate is **worth**, never which
candidates exist.

**FR-9** — No field is added to any record, the byte form's layout and version are unchanged,
and no value crosses the determinism wall: no draw, no clock, no float, no allocation whose
order a decision can see.

## Acceptance

**AC-1** — A census, through this tree's own loader, against **both** installed roots, of the
members and candidates the contract can reach on a shipped map: how many entities carry a reach
above 1, how many are fliers, on which maps the two meet, and whether the roster slots they
stand on are hostile — a pair that cannot see each other as candidates is not a pair. Identical
figures on both roots or the difference is explained.

**AC-2** — A ground member of reach above 1 scoring a flying candidate: **the seed today, a
finite cost under the contract**, and that cost is the row-0 value and not the member's own
row's. The same pair with the member's reach at 1 still scores the seed.

**AC-3** — The rewritten distance term, tabulated for one member reach above 1 across every
separation from 0 to twice that reach: 1 up to and including the reach, then rising by one per
cell, never below 2 outside it.

**AC-4** — Stand ground with a reach above 1: a candidate at exactly that reach is taken, one a
cell further is refused, and the same member at reach 1 takes only what is adjacent.

**AC-5** — Every landed test whose entities all carry a reach of 1 is unchanged, including every
byte-form and digest pin. A red one is a defect in this story, not a pin to re-take.

**AC-6** — The tenth mission, driven by its own two waypoints on **both** roots, before and
after. The prediction on record is that it is **unchanged** — the unit that intercepts carries
reach 1, its victim is a ground mover of reach 1, and the cell they meet at is well inside a
notice circle this story does not touch. A change is a finding to report, not a success.

**AC-7** — The byte-form version and the record length are the same integers after this story as
before, witnessed by a decode of a world encoded before it.

## Properties

**P-1** — For a fixed member and a fixed candidate domain, the cost is non-decreasing in the
separation. The rewrite compresses the near band to a single value; it never inverts the order.

**P-2** — The choice is a pure function of the world: no draw, no map iteration whose order is
not fixed, no time. The determinism scan passes unchanged.

**P-3** — The seed remains the only value a vetoed candidate can score, and remains the value
the selection loop starts at. The two stay one constant.

**P-4** — The two choices remain **one body and an order**, not two functions. Every term this
story adds is either shared or marked as one of the differences between them.

**P-5** — A reach of 0 is unrepresentable — the constructor folds it to 1 and the decoder
refuses it — so no term added here needs a clamp, and none is written.
