# Provenance — 0099

Research pin: **`e6f9ee6`**, read through the submodule at `research/` (`git submodule status`
shows no leading character). Every row below was read at that pin with `go run ./tools/claim <ID>`;
no ledger was opened by hand.

## Claims relied on

| Clause | Claim | Confidence |
|---|---|---|
| FR-1 sub-command 14 is Patrol, the switch is bounded at `Par0 <= 17`, and entry 13 of the table at `L00408` calls the group stop then the per-member setter | `AI-PATROL-017` | High — the bound and the arm are the routine's own instructions and the table was read out of the PE |
| the corpus: 14 `Par0 = 14` nodes over 8 campaign maps, `scn:10` x2, identical on the `ru` root | `AI-PATROL-017` | High for the census (a walk of every shipped map's own records); **Medium** for the per-node member counts |
| FR-3 the command writes `grpAI+0x20 = 0`, the group-order clear | `AI-PATROL-018` | High — one cited store, and the one write that decides whether the rest executes |
| FR-5 the command stops every member | `AI-PATROL-018` | High (six cited stores; this build models the observable, not each field) |
| FR-6 the per-member setter writes state `0xa`, empties the waypoint list and rebuilds it with exactly two nodes — the actor's own cell and the commanded one — and makes the tail current | `AI-PATROL-018` | High |
| FR-8 the arm clears the move order, and on arrival finds the current waypoint in the list and takes the next, falling back to the head — so the path is a ring | `AI-PATROL-013` | High for the ring advance (cited instructions, routine read end to end) |
| FR-9 the arm runs guard **first**, and advances only when guard leaves the order idle or guarding | `AI-PATROL-013` | High — this is what FR-9 declares a divergence from |
| D-2 a ring holding a value the search cannot find falls back to the head, and the one builder of such a ring has no live caller | `AI-PATROL-019` | High for the caller counts (printed enumerations, 0 orphan) |
| FR-11 `actor+0x50` is the per-actor AI state; its value at construction is `0xb`, written by the actor's own initialiser, and the `.alm` spawner writes none | `AI-STATE-011` | High for the bound, the table and the default; **Medium** on writer-set completeness |
| FR-7 one arm per state, `0xa` = patrol and `0xb` = guard, and states with no arm fall to a default | `AI-STATE-011`, `AI-STATE-043` | High for the listed values at the listed addresses; **Medium** that the written set is complete |
| D-3 `order+0x04` is a re-anchor latch, set on every advance and consumed at the next entry to move the guard post to the actor's current cell | `AI-PATROL-018` | High — this is the clause D-3 declines to model, not one it doubts |
| FR-4 what a group order of 0 means, and that the group layer is what it takes a member away from | `AI-GRPGUARD-074`, `AI-ORDER-010` | High (via 0096, unchanged here) |

**`AI-PATROL-013` is partially retracted and the retraction was read.** What falls is its first
clause, "no shipped file reaches it": the `.alm`'s own script does, which is the whole reason this
story exists. What stands, and what is used above, is the ring advance, the guard-first order and
the four bounds. Nothing here rests on the retracted half.

## What was measured rather than cited

The map facts in `analysis.md` — which groups the two nodes name, where their single members stand,
where the protected unit stands, and which entity kills her — are **measurements on the shipped map
through this tree's own loader**, not claims. They are reproducible from `almtool script`,
`missionrun -trace` and a stepped world; the commands are in `verification.md`.

## Owed, and to whom

- The **guard post and the re-anchor latch** (D-3): owed by the story that adds actor state `0xb`.
  Decoded, cited above, deliberately not carried.
- **`missionrun`'s aim is occupancy-blind** — it picks the nearest terrain-open cell and cannot see
  a live unit standing on it, so a waypoint line can read the same before and after a real change.
  Known before this story, unchanged by it, and named here because AC-6's measurement leans on that
  tool. A separate story.
