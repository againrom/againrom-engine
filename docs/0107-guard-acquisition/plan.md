# Plan — 0107

## The shape of the change

One function, `candidateCost` in `pkg/sim/engage.go`, gains four statements and loses none. It
already carries the preference lookup, the distance term, the stand-ground refusal and the two
multiplier arms; what it does not carry is either reach. Everything else the contract names —
the candidate list, the clip, the frozen notice base, the release, the walk home — is read, not
written, and the story is finished by a diff that touches one body and its tests.

That is deliberate and it is the whole cut. The acquisition path is four things — a list, a
clip, a scorer and a latch — and three of them are already faithful. Only the scorer reads
reach, and only reach changed under it.

## FR-1, FR-3 — which row a member reads

`pref := preference[lawDomain(m.Domain)][lawDomain(c.Domain)]` becomes a two-armed choice on
`m.Reach > 1`: row 0 for a ranged member, the member's own row otherwise. The row index is the
literal `0` and not `lawDomain(something)` — the original indexes the table's base with no
member term at all, and writing a domain there would invite a later reader to make it live.

FR-3 is the else arm and needs no code of its own; it is the statement that today's line is
correct for reach 1 and stays. It is checkable by deletion in the other direction: remove the
`> 1` guard and every landed pin holds anyway, because every landed pin carries reach 1.

## FR-2 — the distance rewrite

Immediately after the row choice, under the same `m.Reach > 1` guard:

- `d <= int32(m.Reach)` → `d = 1`
- otherwise → `d = d + 1 - int32(m.Reach)`

Written as the original writes it, in that order, with no `max`. The second arm is reached only
when `d > reach`, so its result is at least 2; that is the sentence FR-2 makes checkable and it
is why P-5's absent clamp is not an omission.

**`d` is already narrowed to a byte before this point** and stays `int32` after it. The narrowing
is the law's own and this story does not move it: the rewrite happens on the narrowed value,
which is what makes a separation of 256 and a separation of 0 behave alike here as they do
everywhere else in this file.

## FR-4, FR-5 — the immobile column

A second two-armed choice, computed **before** the row choice because the column index feeds it:

- ordinary choice: `col = 0` when `lawDomain(c.Domain) == 1 && c.Reach > 1`
- stand-ground choice: `col = 0` when `c.Reach > 1`, whatever the domain

The two differ only in the domain conjunct, so they are one expression and an `||` on the order,
not two blocks. The published reading of the stand-ground variant is "the ordinary one with
three differences", and this is the first of the three; keeping it in the same expression is
what keeps the pair auditable against that sentence.

`lawDomain` itself is untouched. It is already a table rather than `d + 1`, which is what lets a
column index of `0` sit beside it without either becoming arithmetic on the other.

## FR-6 — the refusal reads the rewritten term

The existing line

    if order == orderStandGround && d > groupScorerReach { return scoreSeed }

**moves below** the FR-2 rewrite. Nothing else about it changes: `groupScorerReach` stays the
literal 1 and stays unwired from `Entity.Reach`, which is the previous story's decision and is
not reversed here — the immediate in the original is 1, and what makes a ranged member pass it
is that its distance term was already rewritten to 1 upstream.

The ordering is derived rather than quoted. The published reading gives the refusal's address
inside the stand-ground body and the rewrite's address inside the ordinary body, and states the
stand-ground body is the ordinary one with three enumerated differences, none of which is the
removal of the rewrite; the refusal's offset falls after the rewrite's. Grade it Medium and
record it: it is a design decision with a stated ground, and the guarding order — the arm this
story exists for — has no such refusal, so nothing this story is measured on depends on it.

## FR-7 — the veto, unchanged

No code. The `pref == 0 → scoreSeed` line is already there and already above everything this
story adds, and it must stay above: a vetoed pair must not reach the rewrite. The clause earns a
statement because its **consequence** moves — a ranged member now reads a row with no zero in
it — and that consequence is AC-2's whole content.

## FR-8, FR-9 — the boundary

Nothing outside `candidateCost` is edited. In particular `candidates`, `clipToNotice`,
`noticeBase`, `noticeRadius`, `freezeGroups`, `groupAI`, `decide`'s arms, `orderAttack` and
`releaseAttack` are unchanged, and `binary.go` is not opened at all. **The byte-form version
stays 24.** No field is added, so there is nothing for a version to carry; the version allocated
to this story is not spent and is left free.

The determinism wall is not approached: every term added is integer arithmetic on two entity
records. The `internal/archtest` source scan needs no exception and gets none.

## DD-1 — one body, not two

The two choices stay one function taking an order, which is the decision two stories back and is
re-affirmed rather than re-argued: the published reading of the pair is "the same routine with
three differences", and two Go functions would let the shared thirteen lines drift. Every
statement this story adds is either shared (FR-1, FR-2) or is one of the marked differences
(FR-5).

## DD-2 — the row choice and the column choice are separate statements

They could be folded into one indexing expression. They are not, because they are gated on
**different entities' reaches** — the row on the member's, the column on the candidate's — and a
single expression reading both would be the one shape a later reader could transpose without the
tests noticing, since a symmetric pair of reach-4 ground entities scores identically either way.

## DD-3 — the rewrite is not a `max`

`d = d + 1 - reach` in the outside arm and `d = 1` in the inside arm is the original's own pair,
and it is **not** `max(1, d + 1 - reach)`: the two agree for every value this build can produce,
and writing the closed form would hide which of the two arms a future divergence came from. The
same reasoning kept the distance narrowing a conversion rather than a check.

## DD-4 — no clamp, and why that is safe rather than lucky

The constructor folds a reach of 0 to 1 and the decoder refuses one, so `reach >= 1` holds for
every entity any path can present. The inside arm therefore covers `d <= reach`, the outside arm
`d > reach >= 1`, and `d + 1 - reach >= 2`. P-5 is that pair of facts and not a defensive branch.

## DD-5 — the notice floor stays a constant

The floor under a guarding group's notice circle is a constant in this file, and the value it
carries is the value the game's own registry ships on both roots. It is left a constant. Reading
it would put a file read behind a package that reads no file, and the seam that would carry it —
a constructor input — is a customisation story with its own contract. What this story owes is
the disclosure, which the constant's comment already carries and which this plan does not
duplicate.

## DD-6 — the notice circle is not touched at all

Not the jitter, not the margin, not the freeze. Measured on the tenth mission through this
tree's own loader: the group that intercepts holds one member, its frozen base is the floor, its
working radius is twelve, and the target is acquired at a separation of five. A one-cell jitter
cannot bind at that separation and neither can the margin. The circle is not what admits the
target there, so a story that changed it would be changing something for a reason the
measurement does not support.

## The properties

**P-1** is a table test over separations, not an argument: the near band is flat at 1 and the far
band rises by one per cell, so the sequence is non-decreasing across the join. **P-2** is the
existing determinism scan, unchanged and re-run. **P-3** is `scoreSeed` still having exactly two
uses, the loop's seed and the veto's return, checkable by grep. **P-4** is `candidateCost` still
being the only scorer, checkable the same way. **P-5** is DD-4.

## Traceability

| Contract | Where it lands |
|---|---|
| FR-1, FR-3 | the row choice |
| FR-2 | the distance rewrite |
| FR-4, FR-5 | the column choice |
| FR-6 | the refusal, moved below the rewrite |
| FR-7 | the veto line, unmoved, with its consequence stated |
| FR-8, FR-9 | nothing outside `candidateCost`; `binary.go` unopened |
| AC-1 | a census through the loader, both roots, at the verification stage |
| AC-2, AC-3, AC-4 | unit tests over hand-built worlds in `pkg/sim` |
| AC-5 | the whole landed suite, green with no pin re-taken |
| AC-6 | the tenth mission driven on both roots, before and after |
| AC-7 | a decode of a world encoded before the change |
| DD-1..DD-6 | comments in `engage.go` at the statements they explain |
| P-1..P-5 | table test, determinism scan, two greps, DD-4 |
