# Analysis — unit pathfinding

## The premise the story arrived with, and what turned out to be true

The shape this story was expected to take was a best-first search: an open set ordered by
`f = g + h`, a heuristic scaled to the ratio of two step costs, and a route that is the cheapest one.
The original's own routine had already been read at instruction level before any of this was written,
and it is not that.

`MOVE-SEARCH-001` (High) — the search is a **double-buffered label-correcting wave**: two alternating
frontier lists, no priority queue, no closed set, and **no distance-estimate term of any kind**. A cell
is re-expanded every time its label improves. `MOVE-TERM-003` (High) — it stops in the first generation
that puts *any* label on the destination, so the label it then reads is a minimum over paths of at most
that many steps, not a minimum over all paths. `MOVE-COST-002` (High) — the step costs come out of the
image: a straight step costs the destination cell's cost byte, a diagonal that byte plus half of it
truncated, and for a mover outside the ordinary ground domain the cost plane is not consulted at all
and the two costs are flat 2 and 3.

So **the original's route is not the cheapest route, and a better pathfinder is a different
pathfinder.** Fidelity and quality are not two axes to trade off here; they are opposed, and one search
cannot serve both. That is why the contract carries two modes rather than one search with a quality
knob, and why the mode must be state a digest covers: a mode steering routing from outside the encoding
would make every recorded digest and every replay silently invalid across a switch, and the divergence
would present as a phantom determinism fault rather than as a mode change.

The second reading, `MOVE-ROUTE-004` (High), settled a rule that had been assumed the other way round.
The original has **no corner rule at all** — a diagonal step between two blocked cells is legal, and
passability is not even re-tested during route extraction. Refusing to cut corners is therefore **our**
behaviour, not a reconstruction of anything, and it had to be moved out of the shared rules and into
the optimised mode alone. The extraction's tie-break is asymmetric too: an equal straight candidate
displaces the best, an equal diagonal does not.

## What was looked at, and what was checked rather than assumed

The `research/` submodule at the pin: `claims/move.md` end to end, `formats/move/format.md`, and the
passability rows `TERR-PASS-049`…`053` behind the grid. Every claim relied on was read for its **own**
confidence clause rather than for its row's headline, because several rows are High on the mechanism
and Medium or Unknown on exactly the part a consumer wants (`MOVE-CLAIM-007`'s self-claim question,
`MOVE-WAIT-008`'s verdict table, `MOVE-TERM-003`'s frontier overflow).

The landed `pkg/sim` was re-read rather than taken from the baseline, and the baseline's description of
it was wrong in two ways worth recording. It described occupancy as **per-layer**: there is no layer
anywhere in the package, and the previous story put "no ground/air split" in its own out-of-scope list
in as many words. And it wrote requirements over an `n x n` footprint: there is no footprint field
either, and a unit is one cell. Both would have arrived here as new hashed per-unit state, and neither
is this story's to add — the footprint and the movement-type byte are per-instance values streamed from
the placeable-definition database (`TERR-MOVE-057`), which no story has imported yet.

## What is still unknown, and what was refused because of it

Four things were available and were deliberately not built on:

- **Reservations.** `MOVE-CLAIM-007` is High that a unit stamps the cell it intends to enter, and
  **Unknown** on whether a unit's own outstanding stamp can block its own next search. A reservation
  tier whose self-interaction is undecided cannot be canonical, so none is implemented.
- **Substitute destinations.** `MOVE-TERM-003` is High that a failed search makes the caller route to a
  nearby cell instead, and the routines that *pick* that cell are unread. So failure is handled our own
  way — hold, then give up — rather than by inventing a picker and calling it reconstruction.
  **Both halves are answered since, and appended 2026-07-31 at the `8c92427` pin rather than
  substituted, because this is a dated record.** The "**caller**" half is **retracted at High**: the
  search substitutes in three branches of its own tail, and nothing outside it calls either picker
  (`MOVE-ALT-018`). The "unread" half is closed: rings expand from the requested cell, but inside a
  ring the winner is the **minimum label**, so the choice is the cell cheapest to reach *from the
  mover*, and "free" means *labelled by the wave that just failed* — a passable cell past the budget
  is invisible (`MOVE-ALT-019`, `MOVE-ALT-021`). This story's decision stands unchanged; what moved is
  that the alternative is no longer unread, so it is 0037's to schedule rather than a gap to disclose.
- **Wait-versus-re-search.** `MOVE-WAIT-008` grades the verdict table Medium. Nothing depends on it.
- **The engine's two searches**, one unit-blind to the goal and one unit-aware a few waypoints ahead,
  each with its own refresh counters. Read at High, and a whole tier of its own.

The give-up rule is the reverse case: research is explicit that **no give-up counter exists**
(`MOVE-REFRESH-012`), so it is an addition rather than a reconstruction, and it is stated as ours in
both modes rather than smuggled into the canonical one.

## Alternatives weighed

- **One mode, not two.** Rejected on the first section's finding: a single search either reproduces the
  original's routes or produces good ones, and the story exists to make the difference measurable.
- **The mode as a build flag or a command-line switch.** Rejected: it puts a behaviour-steering value
  outside the digest, which is the one thing a lockstep contract cannot survive.
- **A per-cell cost plane now.** Deferred. The uniform cost `c = 2` is `MOVE-COST-002`'s own flat pair
  and keeps that claim's formula unchanged, so a per-cell plane arriving with terrain is a widening
  rather than a rule change. It costs one property: on varied terrain the canonical mode is faithful to
  the *rule* and not yet to the *route*. Checked rather than assumed — under uniform costs the two modes
  still diverge, because a route with more steps can cost less than one with fewer while the wave stops
  by step count.
- **Giving up per mode.** Rejected as over-fitting: the mode governs route selection and nothing else,
  which keeps a second axis out of the contract and out of every test that has to name a mode.
