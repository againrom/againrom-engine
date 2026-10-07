# Provenance — the route the hero does not take

The evidence behind each normative statement of `spec.md`, organised by spec anchor. Nothing is built
from this file; it points at the contract and is never pointed back at.

## Backing

| spec anchor | source | confidence |
|---|---|---|
| FR-1 — the near search settles at all | `MOVE-ALT-018` | High |
| FR-1 — ring order, and the choice inside a ring | `MOVE-ALT-019`, `MOVE-ALT-021` | High |
| FR-1 — membership is "the failed wave labelled it", not passability | `MOVE-ALT-021` | High |
| FR-1 — a substitute is chosen over the unit-aware plane, so a cell another mover holds is never returned | `MOVE-ALT-021` | High |
| FR-2 — the near bound is a flat eight | `MOVE-ALT-018` | High |
| FR-2 — the far bound is shaped on the distance ordered | `MOVE-ALT-018` | High |
| FR-2 — the bound is read as a growth limit, not as a ring count | `MOVE-ALT-019` | High |
| FR-3, FR-4 — settling is entered only where the wave ends with the goal unlabelled | `MOVE-ALT-018` | High |
| FR-3 — the near search's substitute is not written back into the order | `MOVE-ALT-022` | High |
| AC-5 — the near search's generation budget is the scaled form | `MOVE-TERM-003` | High |
| P-1 — the accept is a strict minimum over the label plane in a fixed ring walk | `MOVE-ALT-019` | High |

`MOVE-ALT-018` is the load-bearing row and it is the one that names the two branches together: the
static branch takes the picker with the distance-shaped bound and ignores the target-actor argument;
the dynamic branch takes either the contact-ring picker, when a target actor was named, or the same
distance picker with a bound of eight. Both branches are entered from the tail the search reaches when
its goal is still unlabelled, and the call-site sweep that bounds them is exhaustive rather than an
enumeration. So "the near search does not settle" was never a decoded fact; it was the unread half of
a row whose other half this tree already ships.

`MOVE-ALT-021` is an exhaustive read of both pickers' whole byte ranges rather than a search for
readers, which is why "the only plane either consults is the label plane" is a statement about all the
instructions and not about the ones that were found.

## Ours by choice

| what the spec fixes | why it is ours |
|---|---|
| The near search's spatial window | The decoded search carries no spatial bound at all; its only bound is the generation budget. The window is this tree's, and it predates this story — it bounds one near search at 289 cells however far the order pointed. Unchanged here. |
| The settle rule as a three-valued search parameter | The decoded search branches on two of its own arguments to reach the same three outcomes. Which shape carries it here is engineering: it stores nothing, so no byte form and no digest can tell the two apart. |
| Which of two equal-label candidates wins | Fixed by the ring walk's own order and a strictly-less accept, which is the decoded arrangement. It is listed here because equal labels are far commoner in this tree than in the original: with no cost plane every step costs the same flat pair, so ties that the shipped cost bytes would have broken arrive here as ties. |
| The contact-ring picker is not implemented | It is selected only by an order that names a target actor. Nothing in this build issues one, so the arm would be a branch no input reaches. |
| Where the tool's evidence lives | The tool reports one line per ordered unit and one outcome line. Nothing decoded says anything about a developer tool. |

## Open — deliberately assigned no meaning

| what | why it is left open |
|---|---|
| The wait rule | The decode has a mover turn and stand still instead of re-searching when a blocked waypoint is exactly one cell away, but the predicate that decides wait from re-search is undecoded — its two helpers were not read, so the verdict table is not pinned and the row carries only Medium for which blocker yields which. Implementing the gate would mean inventing it. This build re-searches in every case. |
| Per-mover claims on the shared plane | The decode has a mover mark the cell it intends to enter before entering it, and a claimed cell is invisible to another mover's dynamic wave. This tree's occupancy plane counts where units *are*, not where they intend to be. Not touched here, and it is what would decide two movers converging on one free cell. |
| The cost plane | Absent from this tree's worlds, so every label this story's substitute chooses between is accumulated from the flat step pair rather than from the shipped cost bytes. The rule is unchanged by the plane's arrival; only the numbers it compares are. |

## Removed

Nothing was dropped from the contract. One statement in the shipped source is **contradicted** rather
than removed, and it is named here because it reads as a decided design point: `route.go`'s note that
a near search returning a cell other than its sub-goal "would be a second, unstated rule about where a
mover steps". It is stated, and it is the decoded rule.
