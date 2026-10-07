# Analysis — a unit executing a player's order is not under an authored group order

## The report

Playing the current build, the owner ordered a group to move: the route is drawn, the units take a
step or two, and they stop, with the whole path reachable.

It is already in the measurements. `0094`'s SC-1 recorded `p0 -> (25,55) r1 : STOPPED SHORT of
(24,57), Chebyshev 2, after 400 ticks` and read it as an interception — which it is, but the
interception is permanent, and that half was not named.

## What was measured before anything was designed

A probe drove mission 10's party member from its own start to (25,55) and printed every tick on
which its order state changed. Both roots agree.

    tick  19  (19,64)  holds a destination     tick 131  (24,58)  holds a destination
    tick 155  (25,57)  holds a destination     tick 167  (25,57)  DESTINATION GONE, victim held
    tick 210  (25,57)  victim gone             tick 461  (25,57)  standing, no order at all

The unit walks eight cells of ten. At tick 166 — one of the sixteen the decision runs on — a
hostile of slot 3 stands at (24,56), Chebyshev **1**. Stand Ground's scorer admits it, the engage is
issued, and the engage ends in `clearOrder`. The player's order is not suspended: it is destroyed,
and nothing re-issues it. The unit kills what it was given and then stands where it stopped forever.

So the defect is not a pathing failure and not a stall. It is the engagement decision consuming a
player's order, and it appeared with `0094` — before that the party carried slot 0, which
`aiGroups` skipped outright, so player orders were accidentally immune.

## What we did not know, and what the ledger turned out to say

**The brief's reading was that a commanded group sits at group order 0** — `AI-CMD-033`'s "every
player order builds a brand-new group at group order 0" — and that the story is therefore an
*absence*: order 0 hands each member to its own `actor+0x50` state, which needs a 27-arm machine
this tree has not got, so all that could be carried honestly is that a commanded group takes no
group decision.

That is not what the row says about a **move**. `AI-CMD-033` divides the orders into two families by
what they leave behind, and names the split explicitly: *guard leaves 1, aggressive 3, **move 4**
(`L00433`, orders `0x16`/`0x1c`), `0x1a` 5 — those four act through `R0023`'s group-order
arm — while ten live arms leave 0.* The constructed 0 is what the dispatcher establishes before the
order routine runs; the move routine then overwrites it. `AI-GROUPCMD-020` and `AI-ORDER-010` give
the same store at the same address from two other directions.

A player's movement order is therefore group order **4**, and order 4's arm is published whole.

## Order 4, and why it is a better fit than order 0

`AI-MOVE-023` reads `R0154` end to end at High. What matters is what it does **not** contain:
no `R0110` (the candidate list), no `R0177` (the scorer), no `R0009` (the
engage). Orders 1, 2, 3 and 5 all call at least the first two; order 4 calls none of them. Its whole
per-member body is a latch:

- not arrived, `ord+0x50 == 0` — the move is **re-issued** and nothing else happens;
- arrived and idle — `ord+0x50 := 1`, the stop distance becomes the member's reach, route cleared;
- route consumed short of the destination — `actor+0x50 := 0xc`, `ord+0x08 := 0`;
- and then, `ord+0x50 != 0` — `R0022`, acquire with no leash.

So the story does not have to be an absence at all. It is one more arm of `AI-ORDER-010`'s dispatch
table, and it is the one arm whose behaviour this tree can carry in full, because the tree already
holds its latch: `ord+0x50 == 0` is *still holding the ordered destination*, and both of the arm's
exits — arrival and a route that ran out — are exactly the two ways this build's move loop ends a
walk. What ends the commanded state is therefore not a rule authored here; it is the lifetime of the
destination the order wrote.

The per-actor state machine stays out of scope, and now it stays out for a better reason than
"unaffordable": order 4 never enters it while the walk is running.

## Three questions this left, and where each went

**Does a commanded actor leave the group the map placed it in?** `AI-CMD-033` establishes that the
dispatcher allocates a new group and adds the commanded actors to it. Whether the actor is
simultaneously unlinked from its authored group is not established by anything at the pin. It
matters here for one thing only — a group sees as one animal (`AI-GROUPSEE-068`), so a commanded
member either does or does not still stamp its sight for its authored group. This build takes the
reading that follows from "a brand-new group": it does not. `spec.md` discloses it. On the campaign
party it is unobservable today, because a start places one member.

**What happens to a commanded unit that is struck?** `AI-RETAL-056`: being struck issues no order
and produces no target — it sets one flag whose single consumer makes the victim turn. So a unit
walking under orders past a hostile does not stop to fight it, and that is the law's answer rather
than this build's simplification. It is the one visible cost of the fix and it is measured.

**Which scorer does a unit get once its order ends?** The law's order-4 arm hands it to
`R0022`, acquire-with-no-leash, which discards its pick unless it is within reach
(`AI-ACQUIRE-002`). This build hands it back to the stance its owner slot selects, and for the
party that is Stand Ground, whose scorer vetoes every candidate past reach (`AI-REACH-072`). The two
agree on *whether* anything is taken and can differ on *which*, when more than one candidate is in
reach at once — the group scorer runs a preference matrix, the acquisition a turn-cost tie-break.
Disclosed rather than hidden; with reach fixed at 1 the population that can differ is a unit with
two or more adjacent hostiles.

## What was checked rather than believed

- `AI-CENSUS-046`'s nine authored slot-0 placements reproduce through this tree's own loader on
  **both roots**: maps 41 (3), 71 (4), 150 (1), 151 (1), total 9, and **none of them holds a
  destination at load**. So none is commanded and none can become so, and the stance they take is
  the one they take today.
- The brief located `orderAttack` in `pkg/sim/step.go`; it is `pkg/sim/engage.go:493`. The line
  number is right and the file is not.
- `engage.go`'s `SelfSlot` doc says the stance "is reached by comparing against `SelfSlot + 1`". The
  code compares against `SelfSlot`. A stale sentence, not a defect, and not touched here — two other
  lanes hold this file.
