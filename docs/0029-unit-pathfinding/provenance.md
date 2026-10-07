# Provenance — unit pathfinding

Claims are cited at the `research/` submodule pin `03a9448`, each at the confidence the relied-on
clause carries there — not its row's headline, since several rows are High on a mechanism and Medium
or Unknown on the part a consumer wants.

## Backing

| Spec anchor | Source | Confidence |
|---|---|---|
| FR-4 — a straight step costs the destination cell's cost `c`, a diagonal `c + (c >> 1)` truncated; the flat pair 2 and 3 | `MOVE-COST-002` | High — the four arms, the shift and the compare are transcribed instructions |
| FR-4 — eight neighbours, and no distance-estimate term anywhere in the search | `MOVE-SEARCH-001`, `MOVE-COST-002` | High — the one distance computed sizes the budget and is never added to a label |
| FR-5 — relabel on improvement, alternating frontiers, no priority queue, no closed set | `MOVE-SEARCH-001` | High — the plane clear, the seed, both list shapes and the improve-then-append form are named instructions |
| FR-5 — the three stop conditions, tested between generations, and the `max(scalar, D>>2) + D` budget for a single-cell mover | `MOVE-TERM-003` | High — the three tests and both budget forms are named instructions |
| FR-5 — the scalar 5 | `MOVE-PARAM-006` | High — the immediates are in one routine and the shipped file agrees 7/7 in both roots |
| FR-5 — the backwards walk, its cost term, the `dx` outer / `dy` inner scan order, the asymmetric accept, and the **absence of any corner rule** | `MOVE-ROUTE-004` | High — both compares and the scan order are named instructions |
| FR-1 — one flags byte per cell, bit 0 blocking ground and bit 1 blocking air | `TERR-PASS-051` | High — predicate, mask setter and default are named instructions; bit meanings follow from what writes them |
| FR-3 — one predicate can serve static blocking and live occupancy, because every mover mask carries its own occupancy bit | `MOVE-PLANE-005` | High for the plane displacements and the mask bits; the row's writer enumeration is Medium and unused |
| FR-4 — a route is recomputed at least once per cell stepped and is never persisted | `MOVE-REFRESH-012` | High for the triggers and their counters; its stuck-counter reading is Medium and unused |
| FR-8 — update order *is* the whole of contention priority, because the original applies none | `MOVE-TICK-009` | High — no ordering instruction exists in the loop body |

## Ours by choice

| Spec anchor | What is fixed here, and why no source asserts it |
|---|---|
| FR-1 — the grid as a construction input, absent meaning all-passable, reserved bits refused | Ours. The original derives its planes at world build and has no absent case; refusing reserved bits keeps the encoding injective. |
| FR-2 — mode, grid and stall count in the digest, and the version raised | Our replay contract, not the original's: it serializes only its block planes and would not notice a routing change. |
| FR-4 — `c = 2` uniformly | The original reads a **per-cell** cost plane for an ordinary ground mover. With no plane here, this takes that claim's own flat pair, the same formula at `c = 2`, which widens without a rule change when a plane arrives — faithful to the rule, not yet to the route on varied terrain. |
| FR-6 — the whole of optimised mode: minimum cost, corner-cut refusal, the 8-cell region, the `(y, x)` tie-break | Ours entirely: the original has no corner rule, no region and no cheapest-route notion, so a third party must copy every clause to reproduce our routes. |
| FR-7 — hold, the stall count, and the threshold 16 | Ours and **provisional**, never tuned against a real tick rate. No give-up counter exists in the original, so this is an addition, stated for both modes rather than hidden inside the canonical one. |
| FR-8 — ascending id as the resolution order | Ours. The original's order is insertion history in one AddTail-only list (`MOVE-TICK-013`…`MOVE-TICK-015`, High) and is **not preserved across save/load**: the loader regroups by player (`MOVE-TICK-017`, High). Cross-load contention parity is therefore not the original's property either, so ours is a fixed load-stable rule — a declared divergence, not an approximation of that one. |
| FR-5 — the extraction scans eight neighbours and not the centre | The original scans nine, the cell itself included, and needs a 1000-step cap because a self-step is reachable (`MOVE-ROUTE-004`). Excluding the centre removes the self-loop. |
| FR-3 — the bounds clamp | Ours in this shape: the original has no clamp but an always-blocked 8-cell plane border (`TERR-PASS-049`, High) that our bounds test replaces. |

## Open

| Question | Why nothing here rests on it |
|---|---|
| Whether a unit's own outstanding reservation can block its own next search | `MOVE-CLAIM-007` grades exactly this **Unknown** while grading the mechanism High. No reservation is implemented, so it is not reached. |
| Which blocker makes a unit wait facing a cell rather than re-search | `MOVE-WAIT-008` grades the verdict table **Medium**, two helper routines unread. Not implemented. |
| Which cell the original substitutes when a search fails | The pickers are named and unread, and the promoted spec lists them unspecified. Failure is handled our own way. |
| Which terrain tiles block | The arms are High (`TERR-PASS-049`, `TERR-PASS-050`), but the placement corpus does **not** corroborate the ground rule: 263 shipped placements stand on cells it blocks and a null model scores higher. A second reason, beside ownership, to leave map derivation out of scope. |
| Whether one generation can exceed the original's 4096-entry frontier on a shipped map | `MOVE-TERM-003` grades that consequence **Medium**. Our frontier has no fixed capacity. |

## Removed

| What was dropped | Why |
|---|---|
| Two step-cost constants, and the heuristic coefficients derived from their ratio | A **reference-derived input**, deleted at import rather than disclosed. Nothing here recovers them: the cost model is `MOVE-COST-002`'s, read out of the image with no candidate value supplied to the experiment, so it is a derivation and not a confirmation. |
| The baseline's admissible-heuristic acceptance criterion | The contract states the *result* — a minimum-cost route — so admissibility asserts nothing at contract level and becomes a design choice downstream. |
| Corner-cut refusal as a shared rule | Refuted as reconstruction by `MOVE-ROUTE-004` (High): the original permits the diagonal. It survives as optimised mode's rule. |
| Multi-cell footprints, and a per-unit ground/air layer | Both are new hashed per-unit state, and both are values the placeable-definition database streams per instance (`TERR-MOVE-057`), which no story has imported. The air bit stays in the grid because the map's own plane carries it. |
| The map-derivation seam the baseline froze here | Owned by the terrain story, whose open items are why the classification is unsettled. |
| The original's 1000-step route cap | Unreachable once the extraction excludes the centre and the budget bounds a route's length. |

## Appended 2026-08-01 — pin `130bb79`: `MOVE-TERM-003` is REFUTED, and this story's `analysis.md` carries the refuted half

The rows above cite `MOVE-TERM-003` for the three stop tests and the `max(scalar, D>>2) + D`
budget. Those survive untouched and FR-5 does not move. What is refuted, and now classed
**REFUTED** in `retracted.md`, is the clause this story never cited but its `analysis.md` did:

> *"`MOVE-TERM-003` is High that a failed search makes **the caller** route to a nearby cell
> instead, and the routines that *pick* that cell are unread."* — `analysis.md`, *Substitute
> destinations*

Both halves are now false. The substitution happens **inside `R0053`'s own tail** — the two
pickers have two and one call sites, all three in that routine, none anywhere else — and the
pickers have since been read: they scan Chebyshev rings outward from the requested cell, scan each
ring to the end, and take the smallest **label**, so the substitute is the cell cheapest to reach
*from the mover*, not the nearest to the request. The eighth search argument `altTarget` is a
pointer to a target **actor**, not a caller-computed cell, and the substitute is never written back
(`MOVE-ALT-018`…`MOVE-ALT-022`).

`analysis.md` is left as written — it is a dated record of what was known — and the decision it
justified is **unaffected**: failure is still handled our own way, hold then give up, and the
reason is now better rather than worse. A picker built outside the search, where the refuted
reading put it, could not see the label plane the real picker reads. `0037` and `0045` already
carry the corrected reading; this file is the third copy and the one that had not been swept.

Two further rows, neither reaching a requirement here. `TERR-MOVE-057` is **SUPERSEDED** — its
`0x44`/`0x82` gloss reads as a ladder and the two domains disagree on 83 203 of 880 704 cells
(`MOVE-DOM-026`) — and this story cites it only to say the domain is state no story has imported,
which is still true. `TERR-COST-052` is retracted with **no Kind assigned**: only the per-step
*duration* read goes through `R1087`, the search reading the cost plane directly in nine
places, so the bit-5 quartering never reaches a path cost. FR-4's flat `c = 2` is unaffected —
it takes `MOVE-COST-002`'s flat pair, not the getter.
