# Analysis — 0107, what makes a guard pick a target

## The question

The previous story built what a guarding group does with **no** target. What gives it one is the
other half, and the reason to open it now is the tenth mission: a driven unit is intercepted and
killed, and the interceptor belongs to a guarding group no script node names.

So the question was not "build the acquisition path" — most of it is already here — but **read
this tree's path against the corpus, term by term, and find which terms are missing and which of
those a shipped map can reach.**

## What is already faithful

Read out of `pkg/sim/engage.go` rather than assumed. The candidate list clears one shared sight
map, stamps every member into it, walks the whole entity list, filters by the **first** member's
hostility row, parks the dead in a second list and falls back to that list when the live one is
empty. The clip drops every candidate past the group's frozen notice base plus a margin,
measured from a centroid recomputed each decision. The choice runs a four-by-four preference
table indexed [member domain][candidate domain] over the original's own numbering, forms
`(distance << 8) + turnCost`, applies a multiplier for a preference of 1 and of 4, seeds the
loop at the veto value and takes strictly less, so list order breaks a tie. Both variants are one
body and an order. The engagement is re-issued every decision and an order naming the victim
already held is left alone.

That is the general scorer, and the guarding order reaches it. **Nothing in the list, the clip or
the shape of the cost is missing.**

## What is missing

Four things, and only one class of them is reachable from a shipped map.

**The two reach terms.** A member whose reach is above 1 should read **row 0** of the table — its
own domain not consulted — and should rewrite its distance term so that everything within reach
ties at 1 and everything outside loses one short of that reach. A candidate that is a ground
mover with reach above 1 should be indexed as the **immobile column**. Neither is here. Both
were correct when reach was the package constant 1; `0104` made reach a per-entity value read
off each placement's weapon and left every reader of it in the choice at the old value.

**The jitter.** The working radius is the frozen base plus 4 plus one of {-1, 0, +1}, rolled on
the tick a has-members latch flips. This tree carries neither the latch nor a lawful draw.

**The group's twenty-tick memory of what attacked it** — a remembered cell stamped into the
shared sight map so a group keeps a candidate nobody can see. Two per-member fields, and no
published writer for the remembered attacker.

**The turn cost.** The low byte of the cost, under a distance term shifted eight bits left. No
row publishes the routine's body, so there is nothing to transcribe; it stays zero and ties fall
to list order.

## What a shipped map reaches

Measured through this tree's own loader, against **both** installed roots, identical figures:

| Mission | Reach above 1 | Fliers |
|---|---|---|
| 10 | 4, all slot 2, ground | none |
| 20 | 23 — 18 ground on slot 3, 5 **flying** on slot 4 | 5, all reach 4 |

The second line of that table is the story. Eighteen bow-armed ground units and five fliers
stand on one shipped map, and today's table vetoes every one of those pairs: a ground member
reads its own row, whose flier cell is a zero. Under row 0 the same cell is a 4 — not merely
allowed but **preferred**. That is a behaviour a player can watch, on a map that ships, and it is
why the cut is the reach terms and nothing else.

The two slots those units stand on are **mutually hostile** on that map, measured from the
world's own relation, so each is in the other's candidate list and the veto is not a hypothetical.

The candidate-side fold is reachable on the tenth mission too: four ground units of reach 4
stand on it, and any melee member scoring one of them would take a preference of 4 where it
takes 2 today.

## The tenth mission, measured

Driven by its own two waypoints, the run ends `outcome lost at tick 272` on both roots. Taken
apart with a throwaway probe against the English root:

- The victim is the driven unit, slot 2, a **ground** mover of reach 1, sight 5.
- The only entity that ever holds her is one member of slot 4's group 7 — a **ghost** mover, 10
  HP, reach 1, sight 6 — which acquires her at tick 103 at a separation of **5** and holds her
  until she falls at tick 251.
- That group has one member. Its centroid at load is its own cell, its frozen notice base is the
  **floor** (its geometry gives 6, the floor is 8) and its working radius is **12**.

So the target is acquired 5 cells out inside a circle of 12. **The notice circle is not what
admits her**, and a one-cell jitter cannot make it so. Sight was excluded by `0090` by
experiment. What is left on that decision is the sight *range* and the relation the map authors —
neither of which is this story.

Both reach terms are also inert there: the interceptor's reach is 1, so it reads its own row and
its plain distance; the victim is a ground mover of reach 1, so no column folds. The prediction
this story records is therefore that the tenth mission does **not move**, and it is recorded as a
prediction precisely because a contract tuned until that mission passed would be worthless.

## Two things this tree says about itself that are no longer true

Both are the same failure — a comment true when written and false two stories later — and both
were read before they were believed.

The entity record's `Reach` field says "no caller of `NewWorld` yet has a way to name anything
but 1". The map loader has filled it from each row's first resolving weapon since `0104`'s third
task; the census above is that field, loaded.

`candidateCost`'s own comment says the candidate-side fold "cannot fire here", on the ground that
reach is a constant 1. It was, and is not.

## What we looked at and did not use

The five-cell break-off measured from a creature's post is real and is not this arm's: it lives
in the per-actor state machine, which a group under the guarding order never evaluates. The group
arm has no leash at all — it re-issues its engagement unconditionally — so the interceptor
chasing its victim nine cells from its post is not, by itself, evidence of a missing term.

Where the release lives under the stand-ground order remains unestablished by any published row.
The previous story flagged it; this story does not touch that path and does not resolve it.
