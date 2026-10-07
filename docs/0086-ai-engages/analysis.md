# Analysis — nothing in this tree fights unless it is told to

## What was not known

This tree resolves a blow, pursues a victim, gives up on an unreachable one and reports the
numbers. Every one of those begins at a `KindAttack` command, and the only producer of one is a
player click. Left alone, a world of monsters and guards stands still forever.

What was not known was **which layer decides**, and the round of research that closed it inverts
the intuition the earlier rounds were built on.

## The layer that decides

The per-actor state machine (`actor+0x50`) is the one a reader meets first, and it governs about
one creature in a hundred and fifty. The load walk gives a group order to every group on every
shipped map, and the constructor's `0` — the only value under which the per-actor machine runs at
all — **survives no map**: over the corpus the load-time byte is 1 for 8085 of 8094 EN placements
and 3 for the other 9 (`AI-CENSUS-046`). Restricted to creatures hostile to slot 0, the
group-centre rule takes 7293 of 7571 and the per-actor post rule 52 (`AI-CENSUS-047`).

So the group is what fights. A build that started at `actor+0x50` would have built the layer
almost nothing runs.

## Three things that had to be read before anything was designed

**Order 3 is not attack.** It is Stand Ground (`AI-STAND-076`), and the load walk gives it to the
groups of type-5 slot 0 — the player's own. Its scorer refuses any candidate past reach and its
arm contains no walk. The player's units stand; the scenario's units watch a circle. Both
published labels for that byte said the opposite.

**A diplomacy change is observed, never announced** (`AI-DIPLO-086`). Five of the six writers end
bare. A change therefore does not stop a unit that is already attacking — it stops that unit being
re-selected, at the next AI tick, when the candidate list is rebuilt from scratch.

**Two cells of the preference matrix are an absolute veto** (`AI-PREF-070`, `AI-FLIER-073`). A
ground or ghost member never auto-selects a flier. The veto is reachable here: this tree's three
movement domains are exactly the law's 1, 2 and 3, and fliers are placed on 30 of the 38 EN maps.

## The terms the law needs and this tree does not have

The line-of-sight predicate. `R0136` is read whole (`AI-LOS-081`) and it is an accumulator
march, not an altitude comparison — but its two input grids, the step grid at `fog+0x22000` and
the cost grid at `fog+0x28000`, **were read and never traced to a writer**. The claim says so in
its own Unknown clause. Without them the *shape* of the visible region cannot be reproduced, only
its bound: the expansion runs inside a 41x41 window and stops at the first entirely-blocked ring,
so the region is a subset of the Chebyshev disk of the sight radius.

A second term is missing for the same reason and at a much lower price: `R0115`, the turn
cost that occupies the low byte of the selection cost, appears in the ledger only as a call site.
It is a tie-break under a distance term shifted eight bits left, so its absence cannot change
which candidate wins unless two candidates are equidistant.

A third is not missing but unheld: which roster slot a human participant carries. The tree already
records that gap twice — `pkg/sim`'s owner field against the route budget, and `SetLocalOwner(0)`
in `pkg/game` — and this story inherits it rather than answering it.

## What sight and reach turn out to be

`actor+0xa5`'s complete writer set is two instructions (`AI-SIGHT-006`): the constructor's default
**5**, and one store inside an arm `AI-GATE-079` shows nothing reaches. Neither class streamer
writes it. So sight has exactly one reachable value in the image — which is the argument
`combat.go`'s `reach` already makes about `actor+0x12c`, reached independently. Both are constants
here, and the parallel is what decides that neither becomes a field.

## What was read

`formats/ai/format.md` end to end; claim rows `AI-FILTER-001`, `AI-ACQUIRE-002`, `AI-DIPLO-004`,
`AI-DIPLO-005`, `AI-SIGHT-006`, `AI-GUARD-007`, `AI-TICK-008`, `AI-RADIUS-014`, `AI-GROUPSEE-068`,
`AI-SCORE-069`, `AI-PREF-070`, `AI-COST-071`, `AI-REACH-072`, `AI-FLIER-073`, `AI-GRPGUARD-074`,
`AI-RADFREEZE-075`, `AI-STAND-076`, `AI-REISSUE-077`, `AI-CLOCK-080`, `AI-LOS-081`,
`AI-DIPLO-086`, and `MOVE-DOM-024`, `MOVE-TURN-031`.

In the tree: `pkg/sim`'s `world.go`, `step.go`, `group.go`, `combat.go`, `rng.go`, `facing.go`,
`binary.go`, `hash.go`; `pkg/mapload/fromalm.go` and `spawn.go`; `pkg/formats/alm/alm.go`;
`pkg/game/world.go` and `pkg/ui/command.go` for the local-participant slot.

## An observation the scope had to answer

The guard arm's targetless branch walks a member to `ord+0x00`, its own post — and
`AI-GRPGUARD-074` states, as part of a claim read end to end, that **for a group under order 1
from load nothing has ever written `ord+0x00`**, because `AI-POST-042`'s complete writer set
contains no group-order path. An unwritten packed cell is 0, and 0 is outside the playable
rectangle every shipped map declares.

Taken at face value that means every idle guard creature is ordered toward the map's corner. It
may be exactly right; it is also the kind of reading that deserves its own falsification rather
than arriving as a side effect of a story about acquisition. The walk half is therefore out of
scope, and the boundary is drawn there for that reason and not for size.
