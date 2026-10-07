# 0167 — escort tick: provenance

The evidence this story is built on, what is ours by choice, and what is open.
Claim ids are rows of the `research/` submodule pinned at `7747b9d`.

## Claims consumed

| Claim | Confidence | What this story takes from it |
|---|---|---|
| `AI-DEFEND-111` | High for the arm and for the cover filter's polarity; Medium for the 11×11 block | The defend arm's whole shape: the Chebyshev range test, the out-of-range close order, the cover engagement on the protected unit's behalf, and the order in which the three run. |
| `AI-FOLLOW-112` | High | The follow arm: the same out-of-range half store for store, and a within-range half of three instructions — at or past 2 cells the ordinary acquisition, below 2 the step-away. No cover scan and no heal. |
| `AI-FOLLOWTAB-113` | High | The defend arm's inner table reduces to one sentence: an escort that the engagement helper has just put on a pursuit or a cast does nothing further this pass. |
| `AI-FOLLOWGAP-114` | High for the packing and the clamp; Medium for the rounding helper | The minimum separation of 2, the step-away target as a straight line away from the escorted unit, the 8.8 fixed-point packing, and the clamp to `[8, dim-9]`. |
| `AI-FOLLOWRANGE-115` | High for the two reader instructions; Medium for the write population | `ord+0x70` is the stop distance both arms read, and the arm's own selector falls back to `actor+0xa5` when it is 0. Shipped ranges are 1..6. |
| `AI-FOLLOWSET-116` | High | Which state each member takes and that the escort target and range are already in place when an arm runs. Consumed, not re-landed: 0166 built the setters. |
| `AI-FOLLOWDEATH-119` | High for the absence of a test; Medium for "nothing clears the target" | Neither arm tests whether the escorted unit is alive or still present. This is the input to DD-3, which authors the one answer this build must give and the law does not. |
| `AI-ACQUIRE-002` | High for the routine; Medium that no fourth route exists | The standing acquisition both arms fall to: everything the unit can see, filtered to enemies, nearest first, the pick discarded unless it is within reach. Also the reading of the `vt+0x20 == 3` virtual as the flier term. |
| `AI-FILTER-001` | High | The filter the cover block runs is the diplomacy predicate, and it runs with the protected unit as the decider. |
| `AI-ORDER-039` | High | Order 4 is *close on the actor with stop distance*, order 1 is *walk to a cell*, order 6 is a pursuit, and order `0xb` is the idle turn. This story reads the arm map only to say which of them each escort path writes. |
| `AI-TICK-008` | High for the slot and the walks; Medium for caller completeness | The per-actor machine runs once per full tick and only while its group's order byte is 0. Both hold here: the actor pass runs on the script phase, and the escort setters put the group's order to none. |
| `AI-BREAK-041` | High | `mover+0x08` is 5, which is the radius the cover block is built at. |
| `TRIG-GRPARM-047` | High for the arms; Medium for the census | The escort range default of 3, and the shipped census this story's own instrument reproduces. |
| `AI-FOLLOWHEAL-118` | High for the control flow and the id; Unknown what spell 6 is | Named to say what SC-2 cuts and what the cut costs: the defender's heal fork, and its documented fall-through into the cover engagement. |
| `AI-FOLLOWAUTH-117` | High for the two cases and the census | The editor's names for the two states, and that both authoring surfaces reach them. Background only. |

## What is ours by choice

- **DD-3, an escorted unit this world no longer holds.** `AI-FOLLOWDEATH-119`
  establishes that nothing in the law clears the escort target and that both arms
  dereference it with no null test. That is a fact about a pointer, and this build
  holds entity ids in a slice a decay stage removes from. The answer authored here
  is that the arm does nothing at all for such an escort.
- **DD-4, the minor axis rounding.** `R0279` is read as a use and graded
  Medium. `pkg/sim` admits no floats, so the ratio is computed as an integer
  division rounded half away from zero.
- **DD-5, where the close order stops.** The law services order 4 every sub-tick
  and stops at the stop distance. This build re-decides on the actor pass alone,
  so an escort stops on the first pass that finds it within range.
- **SC-1, SC-2, SC-3.** Stated in plan.md with their reasons.

## What is open

- **The 11×11 block.** `AI-DEFEND-111` grades the radius Medium: `mover+0x08 = 5`
  rests on one constructor store plus an immediate-form sweep, and a register-form
  store is invisible to that sweep. A different radius changes which hostiles a
  defender covers and nothing else in this story.
- **What `[vt+0x20] == 3` selects.** `AI-DEFEND-111` grades it Unknown.
  `AI-ACQUIRE-002` reads the same virtual as the flier term, and this build uses
  `lawDomain(Domain) == 3`, which is air. If the virtual is something else, the
  preference in the cover engagement picks the wrong candidate class; the fallback
  arm, which takes any candidate, is unaffected.
- **Whether the corpse rule applies to the cover block.** The parking rule is
  published for the acquisition routine and not for the block build. This build
  applies the same rule in both places (plan DD-6).
- **What ends the close order for a human-owned escort.** `AI-ACQUIRE-002` states
  that a unit with nothing to acquire tries a heal when its owner is human and
  writes the idle turn otherwise. Only the second is built (SC-3).
