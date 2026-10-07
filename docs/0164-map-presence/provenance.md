# 0164 — provenance

Every fact this story reconstructs comes from four research claims, all published by EXP-0169 and
all active at pin `a5ea82f`. No other source was consulted. Read them with
`cd research && go run ./tools/claim <ID>`.

| Claim | Confidence | What the spec takes from it |
|---|---|---|
| `TRIG-OFFMAP-041` | High for the arm, the bit, the list removal and the untouched state; Medium for the corpus counts | FR-2 (idempotence, the single bit, the resolved unit), FR-3 (footprint cleared, unlinked from the list every acquisition walk visits), FR-4 (group membership, owner, position, container and health untouched, so the count stays whole) |
| `TRIG-RETURN-042` | High for the arm, the retained cell, the two-stage search and the failure path; Unknown what the fit test rejects a cell for beyond occupancy | FR-5 in full: the retained cell, two attempts at `r = 0`, six at `r = 3`, the exhaustive square, and a failure that changes nothing and does not retry |
| `TRIG-MAPGROUP-043` | High for the two group arms and instant 18's two references | FR-6 (first unit removed, second placed at its cell, `r = 3` only), FR-7 (32 and 33 are 16 and 17 per member, membership untouched) |
| `TRIG-INSTCENSUS-046` | High for the widths and search bounds; Medium for the counts and the once-flag universal | The 81 authored nodes and the 56 trigger-referenced ones, the absence of string, item and building parameters, and `once = 1` on every referencing trigger — which is what makes SC-4 unobservable |

## What the claims do not decide, and what this story did with it

- **Whether the off-map bit is written to a save.** `TRIG-OFFMAP-041` reads the arm that sets the
  bit and not the routine that writes a save file. EXP-0169 records this as its one open thread
  reaching hashed simulation state. AUTHORED here (SC-1): this build serializes the bit in its own
  format at `formatVersion` 48, and claims nothing about the original's format.
- **What the placement fit test rejects a cell for beyond occupancy.** `TRIG-RETURN-042` marks
  `R1287` Unknown; it was not read. AUTHORED here: this build's fit test is bounds, terrain
  open to the unit's own domain, and no other counted entity of that layer. That is the same pair
  of relations this package's own movement predicate already asks, so no new rule is invented — the
  seam is named rather than filled with a guess.
- **The axis order of the exhaustive scan** (SC-6) and **the six random attempts' exact window**
  (SC-5). The first is not in the claim; the second has two readings inside the claim and the
  quoted arithmetic was followed.

## Prior stories this rests on

- 0129 gave `ScriptInstant` its second unit reference, which instant 18 needs and which no arm of
  the vocabulary had before.
- 0156 established that an arm resolving its references through the world by id, and doing nothing
  on a miss, is this package's rule for every script arm.
- 0117 established `effectiveGroup`; FR-7 deliberately does not use it, because the map's own group
  word is what every other script arm over a group reads.
