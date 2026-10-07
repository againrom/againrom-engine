# Provenance — 0106

Research is at the pin the story was authored against. Every row below was read whole, including its
amendments and its retraction state, because this area has several rows that correct each other and
one of them corrects the row this tree's own code comments were written from.

## The post: that it exists, who writes it, and what it holds

| Spec anchor | Claim | Confidence and what it rests on |
|---|---|---|
| FR-1, FR-2 (the field, and that a stance setter writes it) | `AI-POST-095` | **High** — three routines read end to end, all 69 cited encodings re-read out of the PE section table, 0 disagree. It establishes the setter's own member walk is instruction-for-instruction the arm's, so the population written is the population read. |
| FR-2 (that the setter is the writer for a group under this stance, not a first-tick initialiser) | `AI-POST-095`, cross-read with `AI-POST-042` | **High**, and it is a *refutation*: `AI-POST-042` credited the per-actor initialiser, and three of its clauses are in `retracted.md`. The initialiser is reached only through the per-actor machine, which a group under the guard stance never evaluates. |
| FR-2 (that it is rewritten by a later command, so it is not a spawn constant) | `AI-POST-097` | **High** for the enumeration — six call sites, six owners, zero orphan — with a **Medium** that the enumeration is complete, since a dispatch through a computed pointer would be invisible to it. One of the six is the script's own group-command dispatcher, which is this tree's second writer. |
| FR-3 (unconditional anchoring, and the mid-crossing reading) | `AI-STANCE-098` | **High** for the four setter bodies and the branch; **Medium** for the step-target field being what a mid-step guard anchors at, because that field's writer set was not enumerated. This build does not depend on the Medium half: DD-3 in `spec.md` records why the two readings coincide here. |
| FR-2 (that both configurations of the map-load walk end with a post written, so one writer is enough) | `AI-GATE-100` | **High** for both gate tests and their targets, which are properties of the byte layout. **Medium** on what the session field means, which this story does not depend on: it builds one configuration. |

## The guard stance's arm

| Spec anchor | Claim | Confidence and what it rests on |
|---|---|---|
| FR-4, FR-5, FR-6 (the three outcomes and their order) | `AI-GRPGUARD-074` | **High** — one routine read end to end, every branch and store a cited instruction, and the group record and the member's order block told apart by their base registers. |
| FR-4 (that the destination is the member's own post and not the group's centre) | `AI-GRPGUARD-074` | Same row, and it is the row's own correction of an earlier gloss. The centre is the candidate clip's origin; the post is the walk's destination. This tree already names the two apart at `clipToNotice`. |
| FR-4 (that the walk home has a destination at all) | `AI-GRPGUARD-074`'s parenthesis, **refuted** by `AI-POST-095` | The parenthesis is in `retracted.md`. This is the single fact the story turns on, and it moved *toward* the story: the reason recorded in this tree for not building the walk home was that clause. |
| FR-7 (that the outcomes do not depend on the candidate list being non-empty) | `AI-GRPGUARD-074` | The per-member tail's first test is the member's own victim word; the candidate pass and the clip run before it and do not gate it. |
| FR-9, and the story's worth (that this is the arm most placements run) | `AI-CENSUS-047` | **Medium** — corpus, pessimistic and carried-only bounds of 95.6 % to 96.6 % of 8094 placements, 96.3 % restricted to creatures a player can observe breaking off. Corpus agreement alone caps at Medium and this is why the number appears in `analysis.md` and not in the contract. |

## The stand-ground stance

| Spec anchor | Claim | Confidence and what it rests on |
|---|---|---|
| FR-8 (no clip, no walk, no destination) | `AI-STAND-076` | **High** for the behaviour — one arm read end to end against a PE-read dispatch table, and its scorer read whole. **Medium** for the *name*, on three agreeing shipped or authored spellings; a name is not a measurement. The contract uses the behaviour and not the name. |
| FR-2, FR-3 (that this stance anchors a post too, with no idle test) | `AI-STANCE-098` | **High** for the two stand-ground setter bodies: both write the post unconditionally, two instructions after their own state store. This is what makes the post a property of the actor rather than of the guard stance (P-3). |
| The label inversion noted in `analysis.md` | `AI-STAND-076`, citing `AI-AUTHOR-015` and `AI-CENSUS-046` | The load walk gives the stand-ground stance to the local participant's own groups and the guard stance to everyone else's. Nothing in the contract depends on it; it is recorded because a reader who inverts it will build FR-8 the wrong way round. |

## What is cut, and how strongly it is known

The cuts are not cut for want of evidence. Each is decoded to buildable precision, which is why
`spec.md` states each as owed rather than as unknown.

- **The idle turn** — `AI-ORDER-039` (**High**, amended twice; its original arm-`0xb` clause is in
  `retracted.md`), corrected by `AI-TURN-104` (**High** for the routine, **Medium** for the range,
  inheriting `AI-RANGE-102`'s Medium on the divisor) and `AI-RETAL-056` (**High** for the hook and
  the arm, **Medium** for its "exactly one reader"). Between them the entry gate, the flag that
  skips it, the draw and the facing arithmetic are all fixed. DD-4 cuts it on a determinism argument,
  not an evidence one.
- **The heal** — named by `AI-GRPGUARD-074` as the third outcome for a member of the local
  participant's own group. Its routine is not decoded here and the tree has nothing to attach it to.
- **The re-anchor latch** — `AI-PATROL-018`, cited by `0099`'s own provenance at **High**. Still
  unmodelled, and DD-6 in `spec.md` says why the argument for that has changed shape without changing
  its answer.
- **The notice radius's per-flip re-roll** — `AI-GUARD-021` (**High** for the branches, **Medium**
  that the roll is once per flip in play), with the roll's term made exact by `AI-JITTER-103` and
  `AI-RANGE-102`. Untouched here.
- **The session branch** — `AI-GATE-100`, whose Medium is on the meaning of the session field.

## Where this corrects a document in this repository

`pkg/sim/engage.go`'s file header already carries the retraction that motivates this story: it says
the walk home is absent because this tree carries no per-actor field, and no longer because the game
never wrote one. That paragraph is accurate at the current pin and this story consumes it rather
than correcting it.

`docs/0099-patrol/`'s D-3 is the one that needs correcting, and `spec.md` DD-6 does it. Its provenance
row cited `AI-PATROL-018` for the latch, which stands; the claim it made about the post's readers
was written against `AI-GRPGUARD-074`'s parenthesis, which has since been refuted. No `0099`
artifact is edited: a landed contract records what was decided and why, and this story's own
decision is the correction.

## Nothing here is an experiment citation

Every fact above comes from a published claim row read through `tools/claim` at the pin, with its
amendments and its retraction state. No fact in this story exists only inside an experiment, and no
request to research was opened for it.
