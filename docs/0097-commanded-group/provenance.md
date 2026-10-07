# Provenance — 0097, the commanded group

Research pin: **`03d3ac2`**. Every row below was read from the submodule at that pin with
`go run ./tools/claim <ID>`, which prints the retraction state with the row.

## What the contract rests on

| Claim | Confidence | What it decides here |
|---|---|---|
| `AI-CMD-033` | High for the chain; **Medium** that the family split is complete | That every player order allocates a fresh group, and — the clause the whole story turns on — that a **move** order (`0x16`/`0x1c`) leaves that group at order **4**, not at the constructor's 0. The Medium is on completeness of the split, not on the move's value, which the row cites as a store at `L00433`. |
| `AI-MOVE-023` | High, one routine read end to end | The whole behaviour of group order 4: no candidate list, no scorer, no engage; a per-member latch `ord+0x50`; re-issue while the latch is clear; acquisition once it is set. This is the arm the contract implements. |
| `AI-ORDER-010` | High for the table; Medium on writer-set completeness | That `grpAI+0x20` is the group order byte, that 4 is a live value with its own arm, and that orders 1 and 3 never evaluate `actor+0x50`. |
| `AI-GROUPCMD-020` | High for the census and the order byte each arm writes | Independent confirmation of the same store: the script's `Move` sub-command reaches order 4 through `R0108` at `L00433`. Cited for corroboration, not for the script surface, which is out of scope. |
| `AI-STAND-076` | High for the behaviour, **Medium** for the name | That group order 3 is Stand Ground and that its members never step — the stance the party keeps when it is not under a command, and the stance the nine authored slot-0 placements keep. |
| `AI-REACH-072` | High | That order 3's scorer refuses anything past reach, which is why a party member that is *not* under an order still strikes what is adjacent and still takes no step toward anything else. |
| `AI-ACQUIRE-002` | High for the routine; Medium that no fourth route exists | What order 4 hands an arrived member to — acquire with no leash, whose pick is discarded unless it is within reach. Used to bound the divergence in DD-2, not to build anything. |
| `AI-RETAL-056` | High for the hook and the arm; Medium on "exactly one reader" | That being struck issues no order and produces no target. This is why a unit under orders walking past a hostile is faithful rather than pacifist, and it is what AC-6 asserts. |
| `AI-REISSUE-077` | High for the mechanism; Medium for the negative | That the group arms re-issue unconditionally and carry no break-off — so the per-tick re-decision this build already does is right, and only the population it runs over is wrong. |
| `AI-GROUPSEE-068` | (via `AI-REISSUE-077`, `candidates`) | That a group's candidate list is built from one shared stamp of every member's sight — the reason removing a commanded member from its authored group is a behavioural change and not bookkeeping. |
| `AI-CENSUS-046` | **Medium** — a census, capped there by `METHODOLOGY.md` | The nine authored slot-0 placements on maps 41, 71, 150, 151. Reproduced independently through this tree's loader on both roots before it was relied on; see `verification.md`. |
| `AI-CLOCK-080` | High for the two clocks; **Unknown** on a second dispatch path | That the decision runs on a slow clock while the order it writes is executed on a fast one — the shape this build already has, and the reason a destroyed order is never recovered. |
| `AI-GROUP-009` | High for the allocation; Medium for the corpus figures | That a group is an object with membership, which is what makes "the command builds a new group" a statement about membership rather than about a byte. |

## What is ours by choice, and why

**The commanded state is derived, not stored.** The law's latch is `ord+0x50`; this build reads it
as *the unit still holds the destination its order wrote and holds no victim*. That is a
re-expression of the same latch in fields this tree already has, so the byte form, its version and
its digest are untouched. It is exact today because there are exactly three producers of a
destination in this package and only the two command arms produce one without a victim. It is
recorded as a named predicate with that dependency written on it, because the branch that would
break it is already scheduled: `0086` FR-20's walk-home would give an uncommanded unit a
destination.

**A commanded member is removed from the partition of deciders rather than placed in a group of its
own.** Order 4's arm has no group-wide term with any effect — the notice radius it recomputes is
never read by it, and the re-issue is per member — so how commanded members are grouped is
unobservable, and the cheapest faithful shape is the absence of a group.

**An arrived member falls back to the stance its owner slot selects** rather than to a per-actor
acquisition this tree does not have. Bounded in DD-2 and disclosed in the contract.

## What is open

- Whether a commanded actor is unlinked from its authored group, or belongs to both. Not
  established at the pin. This build unlinks it; the consequence is one shared sight stamp and it is
  unobservable on a one-member party.
- What group order 4 does with `grpAI+0x44`, the gate on its re-issue branch. `AI-MOVE-023` names
  the field and not what fills it. This build re-issues unconditionally, which is the same
  behaviour whenever that gate is set.
- Whether the arrival latch and this build's "the walk ended" coincide in every case. The law sets
  the latch on *arrived and idle*; this build ends a walk on arrival, on a route that cannot be
  built, and on the give-up the move loop already owns. The third has no counterpart in the arm and
  is this build's own walk contract, unchanged here.

## What was considered and removed

- **A per-entity group-order byte, and `formatVersion` 21 with it.** Rejected: nothing in the
  contract needs a value that survives a tick which is not already survived by the destination, and
  a second field free to disagree with the first is how two readings of one latch come apart. No
  version number is taken by this story.
- **A third `stance` value for order 4.** Rejected under S-7: it would be a value whose only
  behaviour is to return, and the population it names is better said once, where the partition is
  built.
- **Making `orderAttack` conditional.** Rejected: it is also the attack command's writer, and
  `AI-CMD-054` has that command clear every member before it engages. The rule belongs to who is
  decided over, not to what the decision writes.
